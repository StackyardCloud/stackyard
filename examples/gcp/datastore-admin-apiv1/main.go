package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	datastoreadmin "cloud.google.com/go/datastore/admin/apiv1"
	adminpb "cloud.google.com/go/datastore/admin/apiv1/adminpb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *datastoreadmin.DatastoreAdminClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	indexID := getenv("STACKYARD_GCP_DATASTORE_ADMIN_INDEX_ID", "order-state-amount")
	operationID := getenv("STACKYARD_GCP_DATASTORE_ADMIN_OPERATION_ID", "operation-1")

	projectName := "projects/" + projectID
	operationName := projectName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Datastore Admin apiv1 client using %s\n", apiEndpoint)

	client, err := datastoreadmin.NewDatastoreAdminRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create datastore admin client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ExportEntities",
			call: func(ctx context.Context, c *datastoreadmin.DatastoreAdminClient) error {
				_, err := c.ExportEntities(ctx, &adminpb.ExportEntitiesRequest{
					ProjectId:       projectID,
					OutputUrlPrefix: "gs://stackyard-datastore/export",
					EntityFilter: &adminpb.EntityFilter{
						Kinds:        []string{"Order"},
						NamespaceIds: []string{""},
					},
				})
				return err
			},
		},
		{
			name: "ImportEntities",
			call: func(ctx context.Context, c *datastoreadmin.DatastoreAdminClient) error {
				_, err := c.ImportEntities(ctx, &adminpb.ImportEntitiesRequest{
					ProjectId: projectID,
					InputUrl:  "gs://stackyard-datastore/export/overall_export_metadata",
					EntityFilter: &adminpb.EntityFilter{
						Kinds:        []string{"Order"},
						NamespaceIds: []string{""},
					},
				})
				return err
			},
		},
		{
			name: "CreateIndex",
			call: func(ctx context.Context, c *datastoreadmin.DatastoreAdminClient) error {
				_, err := c.CreateIndex(ctx, &adminpb.CreateIndexRequest{
					ProjectId: projectID,
					Index: &adminpb.Index{
						Kind:     "Order",
						Ancestor: adminpb.Index_NONE,
						Properties: []*adminpb.Index_IndexedProperty{
							{Name: "state", Direction: adminpb.Index_ASCENDING},
							{Name: "amount", Direction: adminpb.Index_DESCENDING},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetIndex",
			call: func(ctx context.Context, c *datastoreadmin.DatastoreAdminClient) error {
				_, err := c.GetIndex(ctx, &adminpb.GetIndexRequest{
					ProjectId: projectID,
					IndexId:   indexID,
				})
				return err
			},
		},
		{
			name: "ListIndexes",
			call: func(ctx context.Context, c *datastoreadmin.DatastoreAdminClient) error {
				it := c.ListIndexes(ctx, &adminpb.ListIndexesRequest{
					ProjectId: projectID,
					PageSize:  1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "DeleteIndex",
			call: func(ctx context.Context, c *datastoreadmin.DatastoreAdminClient) error {
				_, err := c.DeleteIndex(ctx, &adminpb.DeleteIndexRequest{
					ProjectId: projectID,
					IndexId:   indexID,
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *datastoreadmin.DatastoreAdminClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{
					Name: operationName,
				})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *datastoreadmin.DatastoreAdminClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
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
			name: "CancelOperation",
			call: func(ctx context.Context, c *datastoreadmin.DatastoreAdminClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{
					Name: operationName,
				})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *datastoreadmin.DatastoreAdminClient) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{
					Name: operationName,
				})
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

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close datastore admin client: %v\n", err)
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
