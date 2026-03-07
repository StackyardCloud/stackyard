package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type clients struct {
	dataset  *aiplatform.DatasetClient
	model    *aiplatform.ModelClient
	endpoint *aiplatform.EndpointClient
	job      *aiplatform.JobClient
}

type callSpec struct {
	name string
	call func(context.Context, *clients) error
}

func main() {
	parent := getenv("STACKYARD_GCP_PARENT", "projects/stackyard/locations/us-central1")
	grpcEndpoint := grpcEndpointFromEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Vertex AI apiv1 client using %s\n", grpcEndpoint)

	clientOptions := []option.ClientOption{
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	datasetClient, err := aiplatform.NewDatasetClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create dataset client: %v", err)
	}
	defer closeClient("dataset", datasetClient.Close)

	modelClient, err := aiplatform.NewModelClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create model client: %v", err)
	}
	defer closeClient("model", modelClient.Close)

	endpointClient, err := aiplatform.NewEndpointClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create endpoint client: %v", err)
	}
	defer closeClient("endpoint", endpointClient.Close)

	jobClient, err := aiplatform.NewJobClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create job client: %v", err)
	}
	defer closeClient("job", jobClient.Close)

	allClients := &clients{
		dataset:  datasetClient,
		model:    modelClient,
		endpoint: endpointClient,
		job:      jobClient,
	}

	calls := []callSpec{
		{
			name: "ListDatasets",
			call: func(ctx context.Context, c *clients) error {
				it := c.dataset.ListDatasets(ctx, &aiplatformpb.ListDatasetsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "ListModels",
			call: func(ctx context.Context, c *clients) error {
				it := c.model.ListModels(ctx, &aiplatformpb.ListModelsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "ListEndpoints",
			call: func(ctx context.Context, c *clients) error {
				it := c.endpoint.ListEndpoints(ctx, &aiplatformpb.ListEndpointsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "ListCustomJobs",
			call: func(ctx context.Context, c *clients) error {
				it := c.job.ListCustomJobs(ctx, &aiplatformpb.ListCustomJobsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
	}

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, allClients)
		callCancel()

		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		logf("%s succeeded", call.name)
	}

	fmt.Println("Done.")
}

func grpcEndpointFromEnv() string {
	if endpoint := strings.TrimSpace(os.Getenv("STACKYARD_GCP_GRPC_ENDPOINT")); endpoint != "" {
		return normalizeEndpoint(endpoint)
	}
	if endpoint := strings.TrimSpace(os.Getenv("STACKYARD_ENDPOINT")); endpoint != "" {
		hostPort := normalizeEndpoint(endpoint)
		host, _, err := net.SplitHostPort(hostPort)
		if err == nil {
			return net.JoinHostPort(host, "4567")
		}
		return hostPort
	}
	return "localhost:4567"
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
