package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	firestore "cloud.google.com/go/firestore/apiv1"
	"cloud.google.com/go/firestore/apiv1/firestorepb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *firestore.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	databaseID := getenv("STACKYARD_GCP_FIRESTORE_DATABASE_ID", "(default)")
	collectionID := getenv("STACKYARD_GCP_FIRESTORE_COLLECTION_ID", "users")
	documentID := getenv("STACKYARD_GCP_FIRESTORE_DOCUMENT_ID", "user-1")
	operationID := getenv("STACKYARD_GCP_FIRESTORE_OPERATION_ID", "op-1")

	databaseName := fmt.Sprintf("projects/%s/databases/%s", projectID, databaseID)
	documentsRoot := databaseName + "/documents"
	documentName := documentsRoot + "/" + collectionID + "/" + documentID
	operationName := databaseName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Cloud Firestore apiv1 client using %s\n", apiEndpoint)

	client, err := firestore.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create firestore client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "GetDocument",
			call: func(ctx context.Context, c *firestore.Client) error {
				_, err := c.GetDocument(ctx, &firestorepb.GetDocumentRequest{
					Name: documentName,
				})
				return err
			},
		},
		{
			name: "ListDocuments",
			call: func(ctx context.Context, c *firestore.Client) error {
				it := c.ListDocuments(ctx, &firestorepb.ListDocumentsRequest{
					Parent:       documentsRoot,
					CollectionId: collectionID,
					PageSize:     1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CreateDocument",
			call: func(ctx context.Context, c *firestore.Client) error {
				_, err := c.CreateDocument(ctx, &firestorepb.CreateDocumentRequest{
					Parent:       documentsRoot,
					CollectionId: collectionID,
					DocumentId:   documentID,
					Document:     sampleDocument(""),
				})
				return err
			},
		},
		{
			name: "UpdateDocument",
			call: func(ctx context.Context, c *firestore.Client) error {
				_, err := c.UpdateDocument(ctx, &firestorepb.UpdateDocumentRequest{
					Document: sampleDocument(documentName),
					UpdateMask: &firestorepb.DocumentMask{
						FieldPaths: []string{"displayName"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteDocument",
			call: func(ctx context.Context, c *firestore.Client) error {
				return c.DeleteDocument(ctx, &firestorepb.DeleteDocumentRequest{
					Name: documentName,
				})
			},
		},
		{
			name: "BatchGetDocuments",
			call: func(ctx context.Context, c *firestore.Client) error {
				stream, err := c.BatchGetDocuments(ctx, &firestorepb.BatchGetDocumentsRequest{
					Database:  databaseName,
					Documents: []string{documentName},
				})
				if err != nil {
					return err
				}
				_, err = stream.Recv()
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			},
		},
		{
			name: "BeginTransaction",
			call: func(ctx context.Context, c *firestore.Client) error {
				_, err := c.BeginTransaction(ctx, &firestorepb.BeginTransactionRequest{
					Database: databaseName,
				})
				return err
			},
		},
		{
			name: "Commit",
			call: func(ctx context.Context, c *firestore.Client) error {
				_, err := c.Commit(ctx, &firestorepb.CommitRequest{
					Database: databaseName,
					Writes: []*firestorepb.Write{
						{
							Operation: &firestorepb.Write_Update{
								Update: sampleDocument(documentName),
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "Rollback",
			call: func(ctx context.Context, c *firestore.Client) error {
				return c.Rollback(ctx, &firestorepb.RollbackRequest{
					Database:    databaseName,
					Transaction: []byte("tx-1"),
				})
			},
		},
		{
			name: "RunQuery",
			call: func(ctx context.Context, c *firestore.Client) error {
				stream, err := c.RunQuery(ctx, &firestorepb.RunQueryRequest{
					Parent: documentsRoot,
					QueryType: &firestorepb.RunQueryRequest_StructuredQuery{
						StructuredQuery: sampleStructuredQuery(collectionID),
					},
				})
				if err != nil {
					return err
				}
				_, err = stream.Recv()
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			},
		},
		{
			name: "RunAggregationQuery",
			call: func(ctx context.Context, c *firestore.Client) error {
				stream, err := c.RunAggregationQuery(ctx, &firestorepb.RunAggregationQueryRequest{
					Parent: documentsRoot,
					QueryType: &firestorepb.RunAggregationQueryRequest_StructuredAggregationQuery{
						StructuredAggregationQuery: &firestorepb.StructuredAggregationQuery{
							QueryType: &firestorepb.StructuredAggregationQuery_StructuredQuery{
								StructuredQuery: sampleStructuredQuery(collectionID),
							},
							Aggregations: []*firestorepb.StructuredAggregationQuery_Aggregation{
								{
									Alias: "doc_count",
									Operator: &firestorepb.StructuredAggregationQuery_Aggregation_Count_{
										Count: &firestorepb.StructuredAggregationQuery_Aggregation_Count{},
									},
								},
							},
						},
					},
				})
				if err != nil {
					return err
				}
				_, err = stream.Recv()
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			},
		},
		{
			name: "PartitionQuery",
			call: func(ctx context.Context, c *firestore.Client) error {
				it := c.PartitionQuery(ctx, &firestorepb.PartitionQueryRequest{
					Parent:         documentsRoot,
					PartitionCount: 2,
					PageSize:       1,
					QueryType: &firestorepb.PartitionQueryRequest_StructuredQuery{
						StructuredQuery: sampleStructuredQuery(collectionID),
					},
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "ListCollectionIds",
			call: func(ctx context.Context, c *firestore.Client) error {
				it := c.ListCollectionIds(ctx, &firestorepb.ListCollectionIdsRequest{
					Parent:   documentName,
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
			name: "BatchWrite",
			call: func(ctx context.Context, c *firestore.Client) error {
				_, err := c.BatchWrite(ctx, &firestorepb.BatchWriteRequest{
					Database: databaseName,
					Writes: []*firestorepb.Write{
						{
							Operation: &firestorepb.Write_Update{
								Update: sampleDocument(documentName),
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *firestore.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *firestore.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     databaseName,
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
			call: func(ctx context.Context, c *firestore.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *firestore.Client) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
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

func sampleDocument(name string) *firestorepb.Document {
	doc := &firestorepb.Document{
		Fields: map[string]*firestorepb.Value{
			"displayName": {
				ValueType: &firestorepb.Value_StringValue{StringValue: "stackyard-user"},
			},
		},
	}
	if name != "" {
		doc.Name = name
	}
	return doc
}

func sampleStructuredQuery(collectionID string) *firestorepb.StructuredQuery {
	return &firestorepb.StructuredQuery{
		From: []*firestorepb.StructuredQuery_CollectionSelector{
			{CollectionId: collectionID},
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
		fmt.Fprintf(os.Stderr, "warning: close firestore client: %v\n", err)
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
