package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	dataproc "cloud.google.com/go/dataproc/v2/apiv1"
	"cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	batchID := getenv("STACKYARD_GCP_DATAPROC_V2_BATCH_ID", "team-batch")
	sessionID := getenv("STACKYARD_GCP_DATAPROC_V2_SESSION_ID", "interactive-1")
	templateID := getenv("STACKYARD_GCP_DATAPROC_V2_TEMPLATE_ID", "analytics-template")
	operationID := getenv("STACKYARD_GCP_DATAPROC_V2_OPERATION_ID", "operation-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	batchName := parent + "/batches/" + batchID
	sessionName := parent + "/sessions/" + sessionID
	templateName := parent + "/sessionTemplates/" + templateID
	operationName := parent + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Dataproc v2 apiv1 clients using %s\n", apiEndpoint)

	batchClient, err := dataproc.NewBatchControllerRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create batch client: %v", err)
	}
	defer closeClient("batch", batchClient.Close)

	sessionClient, err := dataproc.NewSessionControllerRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create session client: %v", err)
	}
	defer closeClient("session", sessionClient.Close)

	templateClient, err := dataproc.NewSessionTemplateControllerRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create session template client: %v", err)
	}
	defer closeClient("session template", templateClient.Close)

	calls := []callSpec{
		{
			name: "ListBatches",
			call: func(ctx context.Context) error {
				it := batchClient.ListBatches(ctx, &dataprocpb.ListBatchesRequest{
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
			name: "CreateBatch",
			call: func(ctx context.Context) error {
				_, err := batchClient.CreateBatch(ctx, &dataprocpb.CreateBatchRequest{
					Parent:  parent,
					BatchId: batchID,
					Batch:   &dataprocpb.Batch{},
				})
				return err
			},
		},
		{
			name: "GetBatch",
			call: func(ctx context.Context) error {
				_, err := batchClient.GetBatch(ctx, &dataprocpb.GetBatchRequest{
					Name: batchName,
				})
				return err
			},
		},
		{
			name: "DeleteBatch",
			call: func(ctx context.Context) error {
				return batchClient.DeleteBatch(ctx, &dataprocpb.DeleteBatchRequest{
					Name: batchName,
				})
			},
		},
		{
			name: "ListSessions",
			call: func(ctx context.Context) error {
				it := sessionClient.ListSessions(ctx, &dataprocpb.ListSessionsRequest{
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
			name: "CreateSession",
			call: func(ctx context.Context) error {
				_, err := sessionClient.CreateSession(ctx, &dataprocpb.CreateSessionRequest{
					Parent:    parent,
					SessionId: sessionID,
					Session: &dataprocpb.Session{
						Name: sessionName,
						SessionConfig: &dataprocpb.Session_JupyterSession{
							JupyterSession: &dataprocpb.JupyterConfig{
								Kernel: dataprocpb.JupyterConfig_PYTHON,
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetSession",
			call: func(ctx context.Context) error {
				_, err := sessionClient.GetSession(ctx, &dataprocpb.GetSessionRequest{
					Name: sessionName,
				})
				return err
			},
		},
		{
			name: "TerminateSession",
			call: func(ctx context.Context) error {
				_, err := sessionClient.TerminateSession(ctx, &dataprocpb.TerminateSessionRequest{
					Name: sessionName,
				})
				return err
			},
		},
		{
			name: "DeleteSession",
			call: func(ctx context.Context) error {
				_, err := sessionClient.DeleteSession(ctx, &dataprocpb.DeleteSessionRequest{
					Name: sessionName,
				})
				return err
			},
		},
		{
			name: "ListSessionTemplates",
			call: func(ctx context.Context) error {
				it := templateClient.ListSessionTemplates(ctx, &dataprocpb.ListSessionTemplatesRequest{
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
			name: "CreateSessionTemplate",
			call: func(ctx context.Context) error {
				_, err := templateClient.CreateSessionTemplate(ctx, &dataprocpb.CreateSessionTemplateRequest{
					Parent: parent,
					SessionTemplate: &dataprocpb.SessionTemplate{
						Name: templateName,
						SessionConfig: &dataprocpb.SessionTemplate_JupyterSession{
							JupyterSession: &dataprocpb.JupyterConfig{
								Kernel: dataprocpb.JupyterConfig_PYTHON,
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetSessionTemplate",
			call: func(ctx context.Context) error {
				_, err := templateClient.GetSessionTemplate(ctx, &dataprocpb.GetSessionTemplateRequest{
					Name: templateName,
				})
				return err
			},
		},
		{
			name: "UpdateSessionTemplate",
			call: func(ctx context.Context) error {
				_, err := templateClient.UpdateSessionTemplate(ctx, &dataprocpb.UpdateSessionTemplateRequest{
					SessionTemplate: &dataprocpb.SessionTemplate{
						Name: templateName,
						Labels: map[string]string{
							"env": "local",
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteSessionTemplate",
			call: func(ctx context.Context) error {
				return templateClient.DeleteSessionTemplate(ctx, &dataprocpb.DeleteSessionTemplateRequest{
					Name: templateName,
				})
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context) error {
				_, err := sessionClient.GetOperation(ctx, &longrunningpb.GetOperationRequest{
					Name: operationName,
				})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := sessionClient.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent + "/operations",
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
			call: func(ctx context.Context) error {
				return sessionClient.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{
					Name: operationName,
				})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context) error {
				return sessionClient.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{
					Name: operationName,
				})
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
