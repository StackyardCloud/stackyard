package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	functions "cloud.google.com/go/functions/apiv2"
	"cloud.google.com/go/functions/apiv2/functionspb"
	iampb "cloud.google.com/go/iam/apiv1/iampb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *functions.FunctionClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	functionID := getenv("STACKYARD_GCP_FUNCTIONS_V2_FUNCTION_ID", "hello-fn-v2")
	operationID := getenv("STACKYARD_GCP_FUNCTIONS_V2_OPERATION_ID", "op-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	functionName := locationName + "/functions/" + functionID
	operationName := locationName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Cloud Functions apiv2 client using %s\n", apiEndpoint)

	client, err := functions.NewFunctionRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloud functions v2 client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListFunctions",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				it := c.ListFunctions(ctx, &functionspb.ListFunctionsRequest{
					Parent:   locationName,
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
			name: "GetFunction",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				_, err := c.GetFunction(ctx, &functionspb.GetFunctionRequest{Name: functionName})
				return err
			},
		},
		{
			name: "CreateFunction",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				_, err := c.CreateFunction(ctx, &functionspb.CreateFunctionRequest{
					Parent:     locationName,
					FunctionId: functionID,
					Function:   sampleFunction(functionName),
				})
				return err
			},
		},
		{
			name: "UpdateFunction",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				_, err := c.UpdateFunction(ctx, &functionspb.UpdateFunctionRequest{
					Function: &functionspb.Function{
						Name:        functionName,
						Description: "updated by stackyard example",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "DeleteFunction",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				_, err := c.DeleteFunction(ctx, &functionspb.DeleteFunctionRequest{Name: functionName})
				return err
			},
		},
		{
			name: "GenerateUploadUrl",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				_, err := c.GenerateUploadUrl(ctx, &functionspb.GenerateUploadUrlRequest{
					Parent: locationName,
				})
				return err
			},
		},
		{
			name: "GenerateDownloadUrl",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				_, err := c.GenerateDownloadUrl(ctx, &functionspb.GenerateDownloadUrlRequest{
					Name: functionName,
				})
				return err
			},
		},
		{
			name: "ListRuntimes",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				_, err := c.ListRuntimes(ctx, &functionspb.ListRuntimesRequest{
					Parent: locationName,
				})
				return err
			},
		},
		{
			name: "GetIAMPolicy",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{
					Resource: functionName,
				})
				return err
			},
		},
		{
			name: "SetIAMPolicy",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: functionName,
					Policy:   &iampb.Policy{},
				})
				return err
			},
		},
		{
			name: "TestIAMPermissions",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    functionName,
					Permissions: []string{"cloudfunctions.functions.get"},
				})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
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
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *functions.FunctionClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationName,
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

func sampleFunction(name string) *functionspb.Function {
	return &functionspb.Function{
		Name: name,
		BuildConfig: &functionspb.BuildConfig{
			Runtime:    "go123",
			EntryPoint: "HelloHTTP",
			Source: &functionspb.Source{
				Source: &functionspb.Source_StorageSource{
					StorageSource: &functionspb.StorageSource{
						Bucket: "stackyard-source",
						Object: "functions/source.tar.gz",
					},
				},
			},
		},
		ServiceConfig: &functionspb.ServiceConfig{
			TimeoutSeconds:  60,
			AvailableMemory: "256Mi",
		},
	}
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

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close cloud functions v2 client: %v\n", err)
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
