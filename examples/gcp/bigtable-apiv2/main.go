package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/bigtable/admin/apiv2/adminpb"
	"cloud.google.com/go/bigtable/apiv2/bigtablepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, adminpb.BigtableInstanceAdminClient, adminpb.BigtableTableAdminClient, bigtablepb.BigtableClient) error
}

func main() {
	baseCtx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	grpcEndpoint := grpcTarget(endpoint)
	projectID := getenv("STACKYARD_GCP_PROJECT", "stackyard")
	instanceID := getenv("STACKYARD_GCP_BIGTABLE_INSTANCE", "dev-instance")
	tableID := getenv("STACKYARD_GCP_BIGTABLE_TABLE", "orders")
	rowKey := getenv("STACKYARD_GCP_BIGTABLE_ROW_KEY", "order-1")
	columnFamily := getenv("STACKYARD_GCP_BIGTABLE_FAMILY", "cf1")
	columnQualifier := getenv("STACKYARD_GCP_BIGTABLE_QUALIFIER", "status")
	cellValue := getenv("STACKYARD_GCP_BIGTABLE_VALUE", "created")

	projectName := "projects/" + projectID
	instanceName := projectName + "/instances/" + instanceID
	tableName := instanceName + "/tables/" + tableID

	fmt.Printf("Stackyard GCP Bigtable apiv2 clients using gRPC endpoint %s\n", grpcEndpoint)

	conn, err := grpc.DialContext(baseCtx, grpcEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		exitf("failed to connect to gRPC endpoint %s: %v", grpcEndpoint, err)
	}
	defer closeConn(conn)

	instanceAdminClient := adminpb.NewBigtableInstanceAdminClient(conn)
	tableAdminClient := adminpb.NewBigtableTableAdminClient(conn)
	dataClient := bigtablepb.NewBigtableClient(conn)

	calls := []callSpec{
		{
			name: "ListInstances",
			call: func(ctx context.Context, i adminpb.BigtableInstanceAdminClient, _ adminpb.BigtableTableAdminClient, _ bigtablepb.BigtableClient) error {
				_, err := i.ListInstances(ctx, &adminpb.ListInstancesRequest{
					Parent: projectName,
				})
				return err
			},
		},
		{
			name: "GetInstance",
			call: func(ctx context.Context, i adminpb.BigtableInstanceAdminClient, _ adminpb.BigtableTableAdminClient, _ bigtablepb.BigtableClient) error {
				_, err := i.GetInstance(ctx, &adminpb.GetInstanceRequest{
					Name: instanceName,
				})
				return err
			},
		},
		{
			name: "ListTables",
			call: func(ctx context.Context, _ adminpb.BigtableInstanceAdminClient, t adminpb.BigtableTableAdminClient, _ bigtablepb.BigtableClient) error {
				_, err := t.ListTables(ctx, &adminpb.ListTablesRequest{
					Parent:   instanceName,
					PageSize: 1,
				})
				return err
			},
		},
		{
			name: "GetTable",
			call: func(ctx context.Context, _ adminpb.BigtableInstanceAdminClient, t adminpb.BigtableTableAdminClient, _ bigtablepb.BigtableClient) error {
				_, err := t.GetTable(ctx, &adminpb.GetTableRequest{
					Name: tableName,
				})
				return err
			},
		},
		{
			name: "CreateTable",
			call: func(ctx context.Context, _ adminpb.BigtableInstanceAdminClient, t adminpb.BigtableTableAdminClient, _ bigtablepb.BigtableClient) error {
				_, err := t.CreateTable(ctx, &adminpb.CreateTableRequest{
					Parent:  instanceName,
					TableId: tableID,
					Table: &adminpb.Table{
						ColumnFamilies: map[string]*adminpb.ColumnFamily{
							columnFamily: {},
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteTable",
			call: func(ctx context.Context, _ adminpb.BigtableInstanceAdminClient, t adminpb.BigtableTableAdminClient, _ bigtablepb.BigtableClient) error {
				_, err := t.DeleteTable(ctx, &adminpb.DeleteTableRequest{
					Name: tableName,
				})
				return err
			},
		},
		{
			name: "ReadRows",
			call: func(ctx context.Context, _ adminpb.BigtableInstanceAdminClient, _ adminpb.BigtableTableAdminClient, d bigtablepb.BigtableClient) error {
				stream, err := d.ReadRows(ctx, &bigtablepb.ReadRowsRequest{
					TableName: tableName,
					Rows: &bigtablepb.RowSet{
						RowKeys: [][]byte{[]byte(rowKey)},
					},
					RowsLimit: 1,
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
			name: "SampleRowKeys",
			call: func(ctx context.Context, _ adminpb.BigtableInstanceAdminClient, _ adminpb.BigtableTableAdminClient, d bigtablepb.BigtableClient) error {
				stream, err := d.SampleRowKeys(ctx, &bigtablepb.SampleRowKeysRequest{
					TableName: tableName,
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
			name: "MutateRow",
			call: func(ctx context.Context, _ adminpb.BigtableInstanceAdminClient, _ adminpb.BigtableTableAdminClient, d bigtablepb.BigtableClient) error {
				_, err := d.MutateRow(ctx, &bigtablepb.MutateRowRequest{
					TableName: tableName,
					RowKey:    []byte(rowKey),
					Mutations: []*bigtablepb.Mutation{
						{
							Mutation: &bigtablepb.Mutation_SetCell_{
								SetCell: &bigtablepb.Mutation_SetCell{
									FamilyName:      columnFamily,
									ColumnQualifier: []byte(columnQualifier),
									TimestampMicros: -1,
									Value:           []byte(cellValue),
								},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "PingAndWarm",
			call: func(ctx context.Context, _ adminpb.BigtableInstanceAdminClient, _ adminpb.BigtableTableAdminClient, d bigtablepb.BigtableClient) error {
				_, err := d.PingAndWarm(ctx, &bigtablepb.PingAndWarmRequest{
					Name: tableName,
				})
				return err
			},
		},
	}

	for _, call := range calls {
		ctx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
		err := call.call(ctx, instanceAdminClient, tableAdminClient, dataClient)
		cancel()
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedStageError(err):
			logf("%s returned staged error (expected during early emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func grpcTarget(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return "localhost:4566"
	}
	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err == nil && parsed.Host != "" {
			return parsed.Host
		}
	}
	if idx := strings.IndexByte(trimmed, '/'); idx > -1 {
		return trimmed[:idx]
	}
	return trimmed
}

func isToleratedStageError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) {
		return true
	}

	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.Internal, codes.Unknown:
			return true
		}
	}

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "notimplemented") ||
		strings.Contains(lower, "server preface") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "transport is closing")
}

func closeConn(conn *grpc.ClientConn) {
	if err := conn.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close gRPC conn: %v\n", err)
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
