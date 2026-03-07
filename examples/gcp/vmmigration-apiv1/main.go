package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	vmmigration "cloud.google.com/go/vmmigration/apiv1"
	"cloud.google.com/go/vmmigration/apiv1/vmmigrationpb"
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
	call func(context.Context, *vmmigration.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	sourceID := getenv("STACKYARD_GCP_SOURCE_ID", "source-1")
	migratingVMID := getenv("STACKYARD_GCP_MIGRATING_VM_ID", "migrating-vm-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	projectName := fmt.Sprintf("projects/%s", projectID)
	sourceName := fmt.Sprintf("%s/sources/%s", parent, sourceID)
	migratingVMName := fmt.Sprintf("%s/migratingVms/%s", sourceName, migratingVMID)
	groupName := fmt.Sprintf("%s/groups/group-1", parent)
	globalParent := fmt.Sprintf("projects/%s/locations/global", projectID)
	targetProjectName := fmt.Sprintf("%s/targetProjects/target-project-1", globalParent)
	imageImportName := fmt.Sprintf("%s/imageImports/image-import-1", parent)
	imageImportJobName := fmt.Sprintf("%s/imageImportJobs/image-import-job-1", imageImportName)
	diskMigrationJobName := fmt.Sprintf("%s/diskMigrationJobs/disk-migration-job-1", sourceName)
	operationName := fmt.Sprintf("%s/operations/createSource.%s", parent, sourceID)

	fmt.Printf("Stackyard GCP VM Migration apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "vmmigration",
		},
	}

	client, err := vmmigration.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create vmmigration client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *vmmigration.Client) error {
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
			name: "GetLocation",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{
					Name: parent,
				})
				return err
			},
		},
		{
			name: "ListSources",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				it := c.ListSources(ctx, &vmmigrationpb.ListSourcesRequest{
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
			name: "GetSource",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.GetSource(ctx, &vmmigrationpb.GetSourceRequest{Name: sourceName})
				return err
			},
		},
		{
			name: "CreateSource",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				op, err := c.CreateSource(ctx, &vmmigrationpb.CreateSourceRequest{
					Parent:   parent,
					SourceId: sourceID,
					Source: &vmmigrationpb.Source{
						Description: "Stackyard VM Migration source",
						Labels: map[string]string{
							"env": "staged",
						},
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
			name: "UpdateSource",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.UpdateSource(ctx, &vmmigrationpb.UpdateSourceRequest{
					Source: &vmmigrationpb.Source{
						Name:        sourceName,
						Description: "Stackyard VM Migration source updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"description"},
					},
				})
				return err
			},
		},
		{
			name: "FetchInventory",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.FetchInventory(ctx, &vmmigrationpb.FetchInventoryRequest{
					Source: sourceName,
				})
				return err
			},
		},
		{
			name: "FetchStorageInventory",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				it := c.FetchStorageInventory(ctx, &vmmigrationpb.FetchStorageInventoryRequest{
					Source:   sourceName,
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
			name: "ListMigratingVms",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				it := c.ListMigratingVms(ctx, &vmmigrationpb.ListMigratingVmsRequest{
					Parent:   sourceName,
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
			name: "GetMigratingVm",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.GetMigratingVm(ctx, &vmmigrationpb.GetMigratingVmRequest{
					Name: migratingVMName,
				})
				return err
			},
		},
		{
			name: "StartMigration",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.StartMigration(ctx, &vmmigrationpb.StartMigrationRequest{
					MigratingVm: migratingVMName,
				})
				return err
			},
		},
		{
			name: "PauseMigration",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.PauseMigration(ctx, &vmmigrationpb.PauseMigrationRequest{
					MigratingVm: migratingVMName,
				})
				return err
			},
		},
		{
			name: "ResumeMigration",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.ResumeMigration(ctx, &vmmigrationpb.ResumeMigrationRequest{
					MigratingVm: fmt.Sprintf("%s/migratingVms/migrating-vm-paused", sourceName),
				})
				return err
			},
		},
		{
			name: "CreateCloneJob",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.CreateCloneJob(ctx, &vmmigrationpb.CreateCloneJobRequest{
					Parent:     migratingVMName,
					CloneJobId: "clone-job-1",
					CloneJob:   &vmmigrationpb.CloneJob{},
				})
				return err
			},
		},
		{
			name: "ListCloneJobs",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				it := c.ListCloneJobs(ctx, &vmmigrationpb.ListCloneJobsRequest{
					Parent:   migratingVMName,
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
			name: "GetCloneJob",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.GetCloneJob(ctx, &vmmigrationpb.GetCloneJobRequest{
					Name: fmt.Sprintf("%s/cloneJobs/clone-job-1", migratingVMName),
				})
				return err
			},
		},
		{
			name: "CancelCloneJob",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.CancelCloneJob(ctx, &vmmigrationpb.CancelCloneJobRequest{
					Name: fmt.Sprintf("%s/cloneJobs/clone-job-1", migratingVMName),
				})
				return err
			},
		},
		{
			name: "ListGroups",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				it := c.ListGroups(ctx, &vmmigrationpb.ListGroupsRequest{
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
			name: "GetGroup",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.GetGroup(ctx, &vmmigrationpb.GetGroupRequest{Name: groupName})
				return err
			},
		},
		{
			name: "AddGroupMigration",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.AddGroupMigration(ctx, &vmmigrationpb.AddGroupMigrationRequest{
					Group:      groupName,
					MigratingVm: migratingVMName,
				})
				return err
			},
		},
		{
			name: "RemoveGroupMigration",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.RemoveGroupMigration(ctx, &vmmigrationpb.RemoveGroupMigrationRequest{
					Group:      groupName,
					MigratingVm: migratingVMName,
				})
				return err
			},
		},
		{
			name: "ListTargetProjects",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				it := c.ListTargetProjects(ctx, &vmmigrationpb.ListTargetProjectsRequest{
					Parent:   globalParent,
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
			name: "GetTargetProject",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.GetTargetProject(ctx, &vmmigrationpb.GetTargetProjectRequest{
					Name: targetProjectName,
				})
				return err
			},
		},
		{
			name: "ListReplicationCycles",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				it := c.ListReplicationCycles(ctx, &vmmigrationpb.ListReplicationCyclesRequest{
					Parent:   migratingVMName,
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
			name: "GetReplicationCycle",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.GetReplicationCycle(ctx, &vmmigrationpb.GetReplicationCycleRequest{
					Name: fmt.Sprintf("%s/replicationCycles/replication-cycle-1", migratingVMName),
				})
				return err
			},
		},
		{
			name: "ListImageImports",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				it := c.ListImageImports(ctx, &vmmigrationpb.ListImageImportsRequest{
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
			name: "GetImageImport",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.GetImageImport(ctx, &vmmigrationpb.GetImageImportRequest{
					Name: imageImportName,
				})
				return err
			},
		},
		{
			name: "CreateImageImport",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.CreateImageImport(ctx, &vmmigrationpb.CreateImageImportRequest{
					Parent:        parent,
					ImageImportId: "image-import-1",
					ImageImport:   &vmmigrationpb.ImageImport{},
				})
				return err
			},
		},
		{
			name: "ListImageImportJobs",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				it := c.ListImageImportJobs(ctx, &vmmigrationpb.ListImageImportJobsRequest{
					Parent:   imageImportName,
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
			name: "GetImageImportJob",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.GetImageImportJob(ctx, &vmmigrationpb.GetImageImportJobRequest{
					Name: imageImportJobName,
				})
				return err
			},
		},
		{
			name: "CancelImageImportJob",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.CancelImageImportJob(ctx, &vmmigrationpb.CancelImageImportJobRequest{
					Name: imageImportJobName,
				})
				return err
			},
		},
		{
			name: "ListDiskMigrationJobs",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				it := c.ListDiskMigrationJobs(ctx, &vmmigrationpb.ListDiskMigrationJobsRequest{
					Parent:   sourceName,
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
			name: "GetDiskMigrationJob",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.GetDiskMigrationJob(ctx, &vmmigrationpb.GetDiskMigrationJobRequest{
					Name: diskMigrationJobName,
				})
				return err
			},
		},
		{
			name: "CreateDiskMigrationJob",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.CreateDiskMigrationJob(ctx, &vmmigrationpb.CreateDiskMigrationJobRequest{
					Parent:             sourceName,
					DiskMigrationJobId: "disk-migration-job-1",
					DiskMigrationJob:   &vmmigrationpb.DiskMigrationJob{},
				})
				return err
			},
		},
		{
			name: "RunDiskMigrationJob",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.RunDiskMigrationJob(ctx, &vmmigrationpb.RunDiskMigrationJobRequest{
					Name: diskMigrationJobName,
				})
				return err
			},
		},
		{
			name: "CancelDiskMigrationJob",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.CancelDiskMigrationJob(ctx, &vmmigrationpb.CancelDiskMigrationJobRequest{
					Name: diskMigrationJobName,
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *vmmigration.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
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
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotImplemented {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close vmmigration client: %v\n", err)
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
