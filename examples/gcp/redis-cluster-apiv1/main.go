package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	cluster "cloud.google.com/go/redis/cluster/apiv1"
	"cloud.google.com/go/redis/cluster/apiv1/clusterpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type callSpec struct {
	name string
	call func(context.Context, *cluster.CloudRedisClusterClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	clusterID := getenv("STACKYARD_GCP_CLUSTER_ID", "cluster-1")
	backupCollectionID := getenv("STACKYARD_GCP_BACKUP_COLLECTION_ID", "collection-1")
	backupID := getenv("STACKYARD_GCP_BACKUP_ID", "backup-1")

	locationParent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	projectParent := fmt.Sprintf("projects/%s", projectID)
	clusterName := fmt.Sprintf("%s/clusters/%s", locationParent, clusterID)
	backupCollectionName := fmt.Sprintf("%s/backupCollections/%s", locationParent, backupCollectionID)
	backupName := fmt.Sprintf("%s/backups/%s", backupCollectionName, backupID)
	operationName := fmt.Sprintf("%s/operations/op-1", locationParent)

	fmt.Printf("Stackyard GCP Redis Cluster apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "redis_cluster",
		},
	}

	client, err := cluster.NewCloudRedisClusterRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create redis cluster client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{Name: projectParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationParent})
				return err
			},
		},
		{
			name: "ListClusters",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				it := c.ListClusters(ctx, &clusterpb.ListClustersRequest{Parent: locationParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetCluster",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				_, err := c.GetCluster(ctx, &clusterpb.GetClusterRequest{Name: clusterName})
				return err
			},
		},
		{
			name: "CreateCluster",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				op, err := c.CreateCluster(ctx, &clusterpb.CreateClusterRequest{
					Parent:    locationParent,
					ClusterId: clusterID,
					Cluster: &clusterpb.Cluster{
						Name: clusterName,
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "UpdateCluster",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				op, err := c.UpdateCluster(ctx, &clusterpb.UpdateClusterRequest{
					Cluster: &clusterpb.Cluster{
						Name: clusterName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"size_gb", "replica_count"}},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "DeleteCluster",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				op, err := c.DeleteCluster(ctx, &clusterpb.DeleteClusterRequest{Name: clusterName})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "GetClusterCertificateAuthority",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				_, err := c.GetClusterCertificateAuthority(ctx, &clusterpb.GetClusterCertificateAuthorityRequest{
					Name: clusterName + "/certificateAuthority",
				})
				return err
			},
		},
		{
			name: "RescheduleClusterMaintenance",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				op, err := c.RescheduleClusterMaintenance(ctx, &clusterpb.RescheduleClusterMaintenanceRequest{
					Name:           clusterName,
					RescheduleType: clusterpb.RescheduleClusterMaintenanceRequest_SPECIFIC_TIME,
					ScheduleTime:   timestamppb.New(time.Now().UTC().Add(2 * time.Hour)),
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "ListBackupCollections",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				it := c.ListBackupCollections(ctx, &clusterpb.ListBackupCollectionsRequest{Parent: locationParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetBackupCollection",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				_, err := c.GetBackupCollection(ctx, &clusterpb.GetBackupCollectionRequest{Name: backupCollectionName})
				return err
			},
		},
		{
			name: "ListBackups",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				it := c.ListBackups(ctx, &clusterpb.ListBackupsRequest{Parent: backupCollectionName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetBackup",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				_, err := c.GetBackup(ctx, &clusterpb.GetBackupRequest{Name: backupName})
				return err
			},
		},
		{
			name: "DeleteBackup",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				op, err := c.DeleteBackup(ctx, &clusterpb.DeleteBackupRequest{Name: backupName})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "ExportBackup",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				op, err := c.ExportBackup(ctx, &clusterpb.ExportBackupRequest{
					Name: backupName,
					Destination: &clusterpb.ExportBackupRequest_GcsBucket{
						GcsBucket: "gs://stackyard-backups/export",
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "BackupCluster",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				op, err := c.BackupCluster(ctx, &clusterpb.BackupClusterRequest{Name: clusterName})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: locationParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *cluster.CloudRedisClusterClient) error {
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
		fmt.Fprintf(os.Stderr, "warning: close redis cluster client: %v\n", err)
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
