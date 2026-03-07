package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	workflows "cloud.google.com/go/workflows/apiv1"
	"cloud.google.com/go/workflows/apiv1/workflowspb"
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
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	workflowID := getenv("STACKYARD_GCP_WORKFLOW_ID", "workflow-1")

	projectName := "projects/" + projectID
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	workflowName := fmt.Sprintf("%s/workflows/%s", parent, workflowID)

	fmt.Printf("Stackyard GCP Workflows workflows/apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "workflows",
		},
	}

	client, err := workflows.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create workflows client: %v", err)
	}
	defer closeClient(client.Close)

	createOperationName := ""
	updateOperationName := ""
	deleteOperationName := ""

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context) error {
				it := client.ListLocations(ctx, &locationpb.ListLocationsRequest{
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
			name: "GetLocation",
			call: func(ctx context.Context) error {
				_, err := client.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListWorkflows",
			call: func(ctx context.Context) error {
				it := client.ListWorkflows(ctx, &workflowspb.ListWorkflowsRequest{
					Parent:   parent,
					PageSize: 1,
					Filter:   `state="ACTIVE"`,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetWorkflow",
			call: func(ctx context.Context) error {
				_, err := client.GetWorkflow(ctx, &workflowspb.GetWorkflowRequest{
					Name:       workflowName,
					RevisionId: "000001-a4d",
				})
				return err
			},
		},
		{
			name: "CreateWorkflow",
			call: func(ctx context.Context) error {
				op, err := client.CreateWorkflow(ctx, &workflowspb.CreateWorkflowRequest{
					Parent:     parent,
					WorkflowId: workflowID,
					Workflow: &workflowspb.Workflow{
						Name:        workflowName,
						Description: "Stackyard staged workflow",
						SourceCode: &workflowspb.Workflow_SourceContents{
							SourceContents: "main:\n  params: [input]\n  steps:\n  - done:\n      return: ${input}",
						},
					},
				})
				if err != nil {
					return err
				}
				createOperationName = strings.TrimSpace(op.Name())
				return nil
			},
		},
		{
			name: "UpdateWorkflow",
			call: func(ctx context.Context) error {
				op, err := client.UpdateWorkflow(ctx, &workflowspb.UpdateWorkflowRequest{
					Workflow: &workflowspb.Workflow{
						Name:        workflowName,
						Description: "Stackyard staged workflow updated",
						SourceCode: &workflowspb.Workflow_SourceContents{
							SourceContents: "main:\n  params: [input]\n  steps:\n  - done:\n      return: ${input + \"-updated\"}",
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"description", "source_contents"},
					},
				})
				if err != nil {
					return err
				}
				updateOperationName = strings.TrimSpace(op.Name())
				return nil
			},
		},
		{
			name: "ListWorkflowRevisions",
			call: func(ctx context.Context) error {
				it := client.ListWorkflowRevisions(ctx, &workflowspb.ListWorkflowRevisionsRequest{
					Name:     workflowName,
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
			call: func(ctx context.Context) error {
				operationName := firstNonEmpty(updateOperationName, createOperationName)
				if operationName == "" {
					return nil
				}
				_, err := client.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := client.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent,
					PageSize: 1,
					Filter:   "done=false",
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "DeleteWorkflow",
			call: func(ctx context.Context) error {
				op, err := client.DeleteWorkflow(ctx, &workflowspb.DeleteWorkflowRequest{Name: workflowName})
				if err != nil {
					return err
				}
				deleteOperationName = strings.TrimSpace(op.Name())
				return nil
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context) error {
				operationName := firstNonEmpty(deleteOperationName, updateOperationName, createOperationName)
				if operationName == "" {
					return nil
				}
				return client.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx)
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
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
		fmt.Fprintf(os.Stderr, "warning: close workflows client: %v\n", err)
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
