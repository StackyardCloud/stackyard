package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	webrisk "cloud.google.com/go/webrisk/apiv1"
	"cloud.google.com/go/webrisk/apiv1/webriskpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *webrisk.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectNumber := getenv("STACKYARD_GCP_PROJECT_NUMBER", "123456789")
	parent := "projects/" + projectNumber
	submissionURI := getenv("STACKYARD_GCP_WEBRISK_URI", "http://phish.stackyard.test/report")
	operationName := parent + "/operations/submitUri.op-1"

	fmt.Printf("Stackyard GCP Web Risk apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "webrisk",
		},
	}

	client, err := webrisk.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create webrisk client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ComputeThreatListDiff",
			call: func(ctx context.Context, c *webrisk.Client) error {
				_, err := c.ComputeThreatListDiff(ctx, &webriskpb.ComputeThreatListDiffRequest{
					ThreatType: webriskpb.ThreatType_MALWARE,
					Constraints: &webriskpb.ComputeThreatListDiffRequest_Constraints{
						MaxDiffEntries:     1024,
						MaxDatabaseEntries: 2048,
					},
				})
				return err
			},
		},
		{
			name: "SearchUris",
			call: func(ctx context.Context, c *webrisk.Client) error {
				_, err := c.SearchUris(ctx, &webriskpb.SearchUrisRequest{
					Uri:         submissionURI,
					ThreatTypes: []webriskpb.ThreatType{webriskpb.ThreatType_SOCIAL_ENGINEERING},
				})
				return err
			},
		},
		{
			name: "SearchHashes",
			call: func(ctx context.Context, c *webrisk.Client) error {
				_, err := c.SearchHashes(ctx, &webriskpb.SearchHashesRequest{
					HashPrefix:  []byte{0x01, 0x02, 0x03, 0x04},
					ThreatTypes: []webriskpb.ThreatType{webriskpb.ThreatType_MALWARE},
				})
				return err
			},
		},
		{
			name: "CreateSubmission",
			call: func(ctx context.Context, c *webrisk.Client) error {
				_, err := c.CreateSubmission(ctx, &webriskpb.CreateSubmissionRequest{
					Parent: parent,
					Submission: &webriskpb.Submission{
						Uri: submissionURI,
					},
				})
				return err
			},
		},
		{
			name: "SubmitUri",
			call: func(ctx context.Context, c *webrisk.Client) error {
				op, err := c.SubmitUri(ctx, &webriskpb.SubmitUriRequest{
					Parent: parent,
					Submission: &webriskpb.Submission{
						Uri: submissionURI,
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *webrisk.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{
					Name: operationName,
				})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *webrisk.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotImplemented {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close webrisk client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
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
