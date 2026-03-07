package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	videointelligence "cloud.google.com/go/videointelligence/apiv1"
	videointelligencepb "cloud.google.com/go/videointelligence/apiv1/videointelligencepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-east1")
	inputURI := getenv("STACKYARD_GCP_VIDEO_INPUT_URI", "gs://stackyard-inputs/video-1.mp4")
	outputURI := getenv("STACKYARD_GCP_VIDEO_OUTPUT_URI", "gs://stackyard-outputs/video-1.json")
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)

	fmt.Printf("Stackyard GCP Video Intelligence videointelligence/apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "videointelligence",
		},
	}

	client, err := videointelligence.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create videointelligence client: %v", err)
	}
	defer closeClient(client.Close)

	op, err := client.AnnotateVideo(ctx, &videointelligencepb.AnnotateVideoRequest{
		InputUri:   inputURI,
		OutputUri:  outputURI,
		Features:   []videointelligencepb.Feature{videointelligencepb.Feature_SHOT_CHANGE_DETECTION},
		LocationId: locationID,
	})
	if err != nil {
		exitf("AnnotateVideo failed: %v", err)
	}
	if strings.TrimSpace(op.Name()) == "" {
		exitf("AnnotateVideo returned an empty operation name")
	}
	logf("AnnotateVideo succeeded: %s", op.Name())

	metadata, err := op.Metadata()
	if err != nil {
		exitf("AnnotateVideo Metadata failed: %v", err)
	}
	if metadata == nil || len(metadata.GetAnnotationProgress()) == 0 {
		exitf("AnnotateVideo Metadata returned no annotation_progress entries")
	}
	logf("AnnotateVideo Metadata succeeded")

	waitCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	waitResp, err := op.Wait(waitCtx)
	if err != nil {
		exitf("AnnotateVideo Wait failed: %v", err)
	}
	if len(waitResp.GetAnnotationResults()) == 0 {
		exitf("AnnotateVideo Wait returned no annotation results")
	}
	logf("AnnotateVideo Wait succeeded")

	opByName := client.AnnotateVideoOperation(op.Name())
	pollResp, err := opByName.Poll(ctx)
	if err != nil {
		exitf("AnnotateVideo Poll failed: %v", err)
	}
	if pollResp == nil || len(pollResp.GetAnnotationResults()) == 0 {
		exitf("AnnotateVideo Poll returned no annotation results")
	}
	logf("AnnotateVideo Poll succeeded")

	gotOp, err := client.LROClient.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: op.Name()})
	if err != nil {
		exitf("GetOperation failed: %v", err)
	}
	if gotOp.GetName() != op.Name() {
		exitf("GetOperation returned unexpected operation name: %q", gotOp.GetName())
	}
	logf("GetOperation succeeded")

	listOpsIt := client.LROClient.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
		Name:     parent,
		PageSize: 1,
	})
	listedOp, err := listOpsIt.Next()
	if errors.Is(err, iterator.Done) {
		exitf("ListOperations returned no operations")
	}
	if err != nil {
		exitf("ListOperations failed: %v", err)
	}
	if listedOp == nil || strings.TrimSpace(listedOp.GetName()) == "" {
		exitf("ListOperations returned an operation without a name")
	}
	logf("ListOperations succeeded")

	if err := client.LROClient.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: op.Name()}); err != nil {
		exitf("CancelOperation failed: %v", err)
	}
	logf("CancelOperation succeeded")

	if err := client.LROClient.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: op.Name()}); err != nil {
		exitf("DeleteOperation failed: %v", err)
	}
	logf("DeleteOperation succeeded")

	_, err = client.AnnotateVideo(ctx, &videointelligencepb.AnnotateVideoRequest{
		InputUri: inputURI,
	})
	if err == nil {
		exitf("AnnotateVideo missing features unexpectedly succeeded")
	}
	if !isInvalidArgument(err) {
		exitf("AnnotateVideo missing features returned unexpected error: %v", err)
	}
	logf("AnnotateVideo invalid request returned InvalidArgument as expected")

	fmt.Println("Done.")
}

func isInvalidArgument(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.InvalidArgument {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusBadRequest {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "invalidargument")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close videointelligence client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	return base.RoundTrip(clone)
}
