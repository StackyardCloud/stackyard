package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	documentai "cloud.google.com/go/documentai/apiv1"
	"cloud.google.com/go/documentai/apiv1/documentaipb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *documentai.DocumentProcessorClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us")
	processorID := getenv("STACKYARD_GCP_DOCUMENTAI_PROCESSOR_ID", "proc-1")
	processorTypeID := getenv("STACKYARD_GCP_DOCUMENTAI_PROCESSOR_TYPE_ID", "FORM_PARSER_PROCESSOR")
	processorVersionID := getenv("STACKYARD_GCP_DOCUMENTAI_PROCESSOR_VERSION_ID", "ver-1")
	evaluationID := getenv("STACKYARD_GCP_DOCUMENTAI_EVALUATION_ID", "eval-1")
	operationID := getenv("STACKYARD_GCP_DOCUMENTAI_OPERATION_ID", "op-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	processorName := parent + "/processors/" + processorID
	processorTypeName := parent + "/processorTypes/" + processorTypeID
	processorVersionName := processorName + "/processorVersions/" + processorVersionID
	humanReviewConfigName := processorName + "/humanReviewConfig"
	evaluationName := processorVersionName + "/evaluations/" + evaluationID
	operationName := parent + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Document AI apiv1 SDK client using %s\n", apiEndpoint)

	client, err := documentai.NewDocumentProcessorRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create documentai client: %v", err)
	}
	defer closeClient("documentai client", client.Close)

	calls := []callSpec{
		{
			name: "FetchProcessorTypes",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.FetchProcessorTypes(ctx, &documentaipb.FetchProcessorTypesRequest{
					Parent: parent,
				})
				return err
			},
		},
		{
			name: "ListProcessorTypes",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				it := c.ListProcessorTypes(ctx, &documentaipb.ListProcessorTypesRequest{
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
			name: "GetProcessorType",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.GetProcessorType(ctx, &documentaipb.GetProcessorTypeRequest{
					Name: processorTypeName,
				})
				return err
			},
		},
		{
			name: "ListProcessors",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				it := c.ListProcessors(ctx, &documentaipb.ListProcessorsRequest{
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
			name: "CreateProcessor",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.CreateProcessor(ctx, &documentaipb.CreateProcessorRequest{
					Parent: parent,
					Processor: &documentaipb.Processor{
						DisplayName: "stackyard-processor",
						Type:        processorTypeID,
					},
				})
				return err
			},
		},
		{
			name: "GetProcessor",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.GetProcessor(ctx, &documentaipb.GetProcessorRequest{
					Name: processorName,
				})
				return err
			},
		},
		{
			name: "ProcessDocument",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.ProcessDocument(ctx, &documentaipb.ProcessRequest{
					Name: processorName,
				})
				return err
			},
		},
		{
			name: "BatchProcessDocuments",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.BatchProcessDocuments(ctx, &documentaipb.BatchProcessRequest{
					Name: processorName,
				})
				return err
			},
		},
		{
			name: "ListProcessorVersions",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				it := c.ListProcessorVersions(ctx, &documentaipb.ListProcessorVersionsRequest{
					Parent:   processorName,
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
			name: "GetProcessorVersion",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.GetProcessorVersion(ctx, &documentaipb.GetProcessorVersionRequest{
					Name: processorVersionName,
				})
				return err
			},
		},
		{
			name: "DeployProcessorVersion",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.DeployProcessorVersion(ctx, &documentaipb.DeployProcessorVersionRequest{
					Name: processorVersionName,
				})
				return err
			},
		},
		{
			name: "UndeployProcessorVersion",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.UndeployProcessorVersion(ctx, &documentaipb.UndeployProcessorVersionRequest{
					Name: processorVersionName,
				})
				return err
			},
		},
		{
			name: "DeleteProcessorVersion",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.DeleteProcessorVersion(ctx, &documentaipb.DeleteProcessorVersionRequest{
					Name: processorVersionName,
				})
				return err
			},
		},
		{
			name: "EnableProcessor",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.EnableProcessor(ctx, &documentaipb.EnableProcessorRequest{
					Name: processorName,
				})
				return err
			},
		},
		{
			name: "DisableProcessor",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.DisableProcessor(ctx, &documentaipb.DisableProcessorRequest{
					Name: processorName,
				})
				return err
			},
		},
		{
			name: "SetDefaultProcessorVersion",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.SetDefaultProcessorVersion(ctx, &documentaipb.SetDefaultProcessorVersionRequest{
					Processor:               processorName,
					DefaultProcessorVersion: processorVersionName,
				})
				return err
			},
		},
		{
			name: "ReviewDocument",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.ReviewDocument(ctx, &documentaipb.ReviewDocumentRequest{
					HumanReviewConfig: humanReviewConfigName,
					Source: &documentaipb.ReviewDocumentRequest_InlineDocument{
						InlineDocument: &documentaipb.Document{},
					},
				})
				return err
			},
		},
		{
			name: "EvaluateProcessorVersion",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.EvaluateProcessorVersion(ctx, &documentaipb.EvaluateProcessorVersionRequest{
					ProcessorVersion: processorVersionName,
				})
				return err
			},
		},
		{
			name: "GetEvaluation",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.GetEvaluation(ctx, &documentaipb.GetEvaluationRequest{
					Name: evaluationName,
				})
				return err
			},
		},
		{
			name: "ListEvaluations",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				it := c.ListEvaluations(ctx, &documentaipb.ListEvaluationsRequest{
					Parent:   processorVersionName,
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
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{
					Name: parent,
				})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
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
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{
					Name: operationName,
				})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
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
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{
					Name: operationName,
				})
			},
		},
		{
			name: "DeleteProcessor",
			call: func(ctx context.Context, c *documentai.DocumentProcessorClient) error {
				_, err := c.DeleteProcessor(ctx, &documentaipb.DeleteProcessorRequest{
					Name: processorName,
				})
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
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s: %v\n", label, err)
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
