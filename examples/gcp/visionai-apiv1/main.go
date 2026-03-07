package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	visionai "cloud.google.com/go/visionai/apiv1"
	visionaipb "cloud.google.com/go/visionai/apiv1/visionaipb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type clients struct {
	health    *visionai.HealthCheckClient
	streams   *visionai.StreamsClient
	app       *visionai.AppPlatformClient
	live      *visionai.LiveVideoAnalyticsClient
	warehouse *visionai.WarehouseClient
	streaming *visionai.StreamingClient
}

type callSpec struct {
	name string
	call func(context.Context, *clients) error
}

func main() {
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	streamID := getenv("STACKYARD_GCP_VISIONAI_STREAM_ID", "stream-1")
	corpusID := getenv("STACKYARD_GCP_VISIONAI_CORPUS_ID", "corpus-1")
	grpcEndpoint := grpcEndpointFromEnv()

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	clusterName := parent + "/clusters/cluster-1"
	streamName := parent + "/streams/" + streamID
	corpusName := parent + "/corpora/" + corpusID
	seriesName := parent + "/series/series-1"

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Vision AI apiv1 clients using %s\n", grpcEndpoint)

	clientOptions := []option.ClientOption{
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	healthClient, err := visionai.NewHealthCheckClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create visionai healthcheck client: %v", err)
	}
	defer closeClient("healthcheck", healthClient.Close)

	streamsClient, err := visionai.NewStreamsClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create visionai streams client: %v", err)
	}
	defer closeClient("streams", streamsClient.Close)

	appClient, err := visionai.NewAppPlatformClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create visionai app platform client: %v", err)
	}
	defer closeClient("appplatform", appClient.Close)

	liveClient, err := visionai.NewLiveVideoAnalyticsClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create visionai live video analytics client: %v", err)
	}
	defer closeClient("livevideoanalytics", liveClient.Close)

	warehouseClient, err := visionai.NewWarehouseClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create visionai warehouse client: %v", err)
	}
	defer closeClient("warehouse", warehouseClient.Close)

	streamingClient, err := visionai.NewStreamingClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create visionai streaming client: %v", err)
	}
	defer closeClient("streaming", streamingClient.Close)

	allClients := &clients{
		health:    healthClient,
		streams:   streamsClient,
		app:       appClient,
		live:      liveClient,
		warehouse: warehouseClient,
		streaming: streamingClient,
	}

	streamOperationName := ""
	corpusOperationName := ""

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *clients) error {
				it := c.health.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     "projects/" + projectID,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.health.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "HealthCheck",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.health.HealthCheck(ctx, &visionaipb.HealthCheckRequest{
					Cluster: clusterName,
				})
				return err
			},
		},
		{
			name: "ListStreams",
			call: func(ctx context.Context, c *clients) error {
				it := c.streams.ListStreams(ctx, &visionaipb.ListStreamsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetStream",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.streams.GetStream(ctx, &visionaipb.GetStreamRequest{Name: streamName})
				return err
			},
		},
		{
			name: "CreateStream",
			call: func(ctx context.Context, c *clients) error {
				op, err := c.streams.CreateStream(ctx, &visionaipb.CreateStreamRequest{
					Parent:   parent,
					StreamId: streamID,
					Stream: &visionaipb.Stream{
						DisplayName: "Stackyard Stream",
					},
				})
				if err != nil {
					return err
				}
				streamOperationName = strings.TrimSpace(op.Name())
				return nil
			},
		},
		{
			name: "GetStreamOperation",
			call: func(ctx context.Context, c *clients) error {
				if streamOperationName == "" {
					return nil
				}
				_, err := c.streams.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: streamOperationName})
				return err
			},
		},
		{
			name: "ListApplications",
			call: func(ctx context.Context, c *clients) error {
				it := c.app.ListApplications(ctx, &visionaipb.ListApplicationsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "ListPublicOperators",
			call: func(ctx context.Context, c *clients) error {
				it := c.live.ListPublicOperators(ctx, &visionaipb.ListPublicOperatorsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CreateCorpus",
			call: func(ctx context.Context, c *clients) error {
				op, err := c.warehouse.CreateCorpus(ctx, &visionaipb.CreateCorpusRequest{
					Parent: parent,
					Corpus: &visionaipb.Corpus{
						DisplayName: "Stackyard Corpus",
					},
				})
				if err != nil {
					return err
				}
				corpusOperationName = strings.TrimSpace(op.Name())
				return nil
			},
		},
		{
			name: "GetCorpusOperation",
			call: func(ctx context.Context, c *clients) error {
				if corpusOperationName == "" {
					return nil
				}
				_, err := c.warehouse.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: corpusOperationName})
				return err
			},
		},
		{
			name: "ListCorpora",
			call: func(ctx context.Context, c *clients) error {
				it := c.warehouse.ListCorpora(ctx, &visionaipb.ListCorporaRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetCorpus",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.warehouse.GetCorpus(ctx, &visionaipb.GetCorpusRequest{Name: corpusName})
				return err
			},
		},
		{
			name: "AcquireLease",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.streaming.AcquireLease(ctx, &visionaipb.AcquireLeaseRequest{
					Series: seriesName,
					Owner:  "stackyard-owner",
				})
				return err
			},
		},
		{
			name: "RenewLease",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.streaming.RenewLease(ctx, &visionaipb.RenewLeaseRequest{
					Id:     "lease-1",
					Series: seriesName,
					Owner:  "stackyard-owner",
				})
				return err
			},
		},
		{
			name: "ReleaseLease",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.streaming.ReleaseLease(ctx, &visionaipb.ReleaseLeaseRequest{
					Id:     "lease-1",
					Series: seriesName,
					Owner:  "stackyard-owner",
				})
				return err
			},
		},
		{
			name: "NegativeListStreamsMissingParent",
			call: func(ctx context.Context, c *clients) error {
				it := c.streams.ListStreams(ctx, &visionaipb.ListStreamsRequest{})
				_, err := it.Next()
				if err == nil {
					return fmt.Errorf("expected invalid argument from list streams without parent")
				}
				return expectGRPCCode(err, codes.InvalidArgument)
			},
		},
	}

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, 7*time.Second)
		err := call.call(callCtx, allClients)
		callCancel()

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

func expectGRPCCode(err error, want codes.Code) error {
	if err == nil {
		return fmt.Errorf("expected grpc code %s, got nil", want.String())
	}
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("expected grpc status error code %s, got %v", want.String(), err)
	}
	if st.Code() != want {
		return fmt.Errorf("expected grpc code %s, got %s: %v", want.String(), st.Code().String(), err)
	}
	return nil
}

func grpcEndpointFromEnv() string {
	if endpoint := strings.TrimSpace(os.Getenv("STACKYARD_GCP_GRPC_ENDPOINT")); endpoint != "" {
		return normalizeEndpoint(endpoint)
	}
	httpBase := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	return normalizeEndpoint(httpBase)
}

func normalizeEndpoint(raw string) string {
	endpoint := strings.TrimSpace(raw)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	if idx := strings.Index(endpoint, "/"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return endpoint
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
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
