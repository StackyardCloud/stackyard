package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	iampb "cloud.google.com/go/iam/apiv1/iampb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	metastore "cloud.google.com/go/metastore/apiv1"
	"cloud.google.com/go/metastore/apiv1/metastorepb"
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
	call func(context.Context, *metastore.DataprocMetastoreClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	serviceID := getenv("STACKYARD_GCP_METASTORE_SERVICE_ID", "hive-a")
	importID := getenv("STACKYARD_GCP_METASTORE_IMPORT_ID", "import-a")
	backupID := getenv("STACKYARD_GCP_METASTORE_BACKUP_ID", "backup-a")
	operationID := getenv("STACKYARD_GCP_METASTORE_OPERATION_ID", "op-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	parent := projectName + "/locations/" + locationID
	serviceName := parent + "/services/" + serviceID
	metadataImportName := serviceName + "/metadataImports/" + importID
	backupName := serviceName + "/backups/" + backupID
	operationName := parent + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Dataproc Metastore apiv1 client using %s\n", apiEndpoint)

	client, err := metastore.NewDataprocMetastoreRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create dataproc metastore client: %v", err)
	}
	defer closeClient("dataproc metastore", client.Close)

	calls := []callSpec{
		{
			name: "ListServices",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				it := c.ListServices(ctx, &metastorepb.ListServicesRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetService",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.GetService(ctx, &metastorepb.GetServiceRequest{Name: serviceName})
				return err
			},
		},
		{
			name: "CreateService",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.CreateService(ctx, &metastorepb.CreateServiceRequest{
					Parent:    parent,
					ServiceId: serviceID,
					Service: &metastorepb.Service{
						Name:   serviceName,
						Labels: map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "UpdateService",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.UpdateService(ctx, &metastorepb.UpdateServiceRequest{
					Service: &metastorepb.Service{
						Name:   serviceName,
						Labels: map[string]string{"owner": "stackyard"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "DeleteService",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.DeleteService(ctx, &metastorepb.DeleteServiceRequest{Name: serviceName})
				return err
			},
		},
		{
			name: "ListMetadataImports",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				it := c.ListMetadataImports(ctx, &metastorepb.ListMetadataImportsRequest{Parent: serviceName, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetMetadataImport",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.GetMetadataImport(ctx, &metastorepb.GetMetadataImportRequest{Name: metadataImportName})
				return err
			},
		},
		{
			name: "CreateMetadataImport",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.CreateMetadataImport(ctx, &metastorepb.CreateMetadataImportRequest{
					Parent:           serviceName,
					MetadataImportId: importID,
					MetadataImport: &metastorepb.MetadataImport{
						Name: metadataImportName,
						Metadata: &metastorepb.MetadataImport_DatabaseDump_{
							DatabaseDump: &metastorepb.MetadataImport_DatabaseDump{
								DatabaseType: metastorepb.MetadataImport_DatabaseDump_MYSQL,
								GcsUri:       "gs://stackyard-metastore/import.sql",
								Type:         metastorepb.DatabaseDumpSpec_MYSQL,
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateMetadataImport",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.UpdateMetadataImport(ctx, &metastorepb.UpdateMetadataImportRequest{
					MetadataImport: &metastorepb.MetadataImport{
						Name:        metadataImportName,
						Description: "updated by stackyard example",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "ExportMetadata",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.ExportMetadata(ctx, &metastorepb.ExportMetadataRequest{
					Service: serviceName,
					Destination: &metastorepb.ExportMetadataRequest_DestinationGcsFolder{
						DestinationGcsFolder: "gs://stackyard-metastore/export/",
					},
					DatabaseDumpType: metastorepb.DatabaseDumpSpec_MYSQL,
				})
				return err
			},
		},
		{
			name: "RestoreService",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.RestoreService(ctx, &metastorepb.RestoreServiceRequest{
					Service:     serviceName,
					Backup:      backupName,
					RestoreType: metastorepb.Restore_METADATA_ONLY,
				})
				return err
			},
		},
		{
			name: "ListBackups",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				it := c.ListBackups(ctx, &metastorepb.ListBackupsRequest{Parent: serviceName, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetBackup",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.GetBackup(ctx, &metastorepb.GetBackupRequest{Name: backupName})
				return err
			},
		},
		{
			name: "CreateBackup",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.CreateBackup(ctx, &metastorepb.CreateBackupRequest{
					Parent:   serviceName,
					BackupId: backupID,
					Backup: &metastorepb.Backup{
						Name:        backupName,
						Description: "stackyard backup",
					},
				})
				return err
			},
		},
		{
			name: "DeleteBackup",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.DeleteBackup(ctx, &metastorepb.DeleteBackupRequest{Name: backupName})
				return err
			},
		},
		{
			name: "QueryMetadata",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.QueryMetadata(ctx, &metastorepb.QueryMetadataRequest{
					Service: serviceName,
					Query:   "select 1",
				})
				return err
			},
		},
		{
			name: "MoveTableToDatabase",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.MoveTableToDatabase(ctx, &metastorepb.MoveTableToDatabaseRequest{
					Service:           serviceName,
					TableName:         "orders",
					DbName:            "analytics",
					DestinationDbName: "archive",
				})
				return err
			},
		},
		{
			name: "AlterMetadataResourceLocation",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.AlterMetadataResourceLocation(ctx, &metastorepb.AlterMetadataResourceLocationRequest{
					Service:      serviceName,
					ResourceName: "databases/analytics",
					LocationUri:  "gs://stackyard-metastore/analytics",
				})
				return err
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: serviceName})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: serviceName,
					Policy:   &iampb.Policy{},
				})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    serviceName,
					Permissions: []string{"metastore.services.get"},
				})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{Name: projectName, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *metastore.DataprocMetastoreClient) error {
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

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
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
