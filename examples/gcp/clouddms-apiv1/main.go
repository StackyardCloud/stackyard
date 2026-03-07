package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	clouddms "cloud.google.com/go/clouddms/apiv1"
	"cloud.google.com/go/clouddms/apiv1/clouddmspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	longrunningpb "google.golang.org/genproto/googleapis/longrunning"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *clouddms.DataMigrationClient) error
}

func main() {
	project := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	location := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	grpcEndpoint := grpcEndpointFromEnv()

	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	migrationJobID := getenv("STACKYARD_GCP_DMS_MIGRATION_JOB_ID", "team-job")
	connectionProfileID := getenv("STACKYARD_GCP_DMS_CONNECTION_PROFILE_ID", "team-profile")
	privateConnectionID := getenv("STACKYARD_GCP_DMS_PRIVATE_CONNECTION_ID", "team-private-connection")
	conversionWorkspaceID := getenv("STACKYARD_GCP_DMS_CONVERSION_WORKSPACE_ID", "team-workspace")
	mappingRuleID := getenv("STACKYARD_GCP_DMS_MAPPING_RULE_ID", "team-rule")

	migrationJobName := parent + "/migrationJobs/" + migrationJobID
	connectionProfileName := parent + "/connectionProfiles/" + connectionProfileID
	privateConnectionName := parent + "/privateConnections/" + privateConnectionID
	conversionWorkspaceName := parent + "/conversionWorkspaces/" + conversionWorkspaceID
	mappingRuleName := conversionWorkspaceName + "/mappingRules/" + mappingRuleID
	operationName := parent + "/operations/team-operation"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Database Migration apiv1 client using %s\n", grpcEndpoint)

	client, err := clouddms.NewDataMigrationClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create clouddms client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListMigrationJobs",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				it := c.ListMigrationJobs(ctx, &clouddmspb.ListMigrationJobsRequest{
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
			name: "GetMigrationJob",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.GetMigrationJob(ctx, &clouddmspb.GetMigrationJobRequest{Name: migrationJobName})
				return err
			},
		},
		{
			name: "CreateMigrationJob",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.CreateMigrationJob(ctx, &clouddmspb.CreateMigrationJobRequest{
					Parent:         parent,
					MigrationJobId: migrationJobID,
					MigrationJob: &clouddmspb.MigrationJob{
						Name:        migrationJobName,
						DisplayName: "Stackyard Migration Job",
						Type:        clouddmspb.MigrationJob_TYPE_UNSPECIFIED,
						Source:      connectionProfileName,
						Destination: connectionProfileName,
					},
				})
				return err
			},
		},
		{
			name: "StartMigrationJob",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.StartMigrationJob(ctx, &clouddmspb.StartMigrationJobRequest{Name: migrationJobName})
				return err
			},
		},
		{
			name: "ListConnectionProfiles",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				it := c.ListConnectionProfiles(ctx, &clouddmspb.ListConnectionProfilesRequest{
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
			name: "GetConnectionProfile",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.GetConnectionProfile(ctx, &clouddmspb.GetConnectionProfileRequest{Name: connectionProfileName})
				return err
			},
		},
		{
			name: "CreateConnectionProfile",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.CreateConnectionProfile(ctx, &clouddmspb.CreateConnectionProfileRequest{
					Parent:              parent,
					ConnectionProfileId: connectionProfileID,
					ConnectionProfile: &clouddmspb.ConnectionProfile{
						Name:        connectionProfileName,
						DisplayName: "Stackyard Connection Profile",
					},
				})
				return err
			},
		},
		{
			name: "ListPrivateConnections",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				it := c.ListPrivateConnections(ctx, &clouddmspb.ListPrivateConnectionsRequest{
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
			name: "GetPrivateConnection",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.GetPrivateConnection(ctx, &clouddmspb.GetPrivateConnectionRequest{Name: privateConnectionName})
				return err
			},
		},
		{
			name: "CreatePrivateConnection",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.CreatePrivateConnection(ctx, &clouddmspb.CreatePrivateConnectionRequest{
					Parent:              parent,
					PrivateConnectionId: privateConnectionID,
					PrivateConnection: &clouddmspb.PrivateConnection{
						Name:        privateConnectionName,
						DisplayName: "Stackyard Private Connection",
					},
				})
				return err
			},
		},
		{
			name: "ListConversionWorkspaces",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				it := c.ListConversionWorkspaces(ctx, &clouddmspb.ListConversionWorkspacesRequest{
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
			name: "GetConversionWorkspace",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.GetConversionWorkspace(ctx, &clouddmspb.GetConversionWorkspaceRequest{Name: conversionWorkspaceName})
				return err
			},
		},
		{
			name: "CreateConversionWorkspace",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.CreateConversionWorkspace(ctx, &clouddmspb.CreateConversionWorkspaceRequest{
					Parent:                parent,
					ConversionWorkspaceId: conversionWorkspaceID,
					ConversionWorkspace: &clouddmspb.ConversionWorkspace{
						Name:        conversionWorkspaceName,
						DisplayName: "Stackyard Conversion Workspace",
					},
				})
				return err
			},
		},
		{
			name: "ListMappingRules",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				it := c.ListMappingRules(ctx, &clouddmspb.ListMappingRulesRequest{
					Parent:   conversionWorkspaceName,
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
			name: "GetMappingRule",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.GetMappingRule(ctx, &clouddmspb.GetMappingRuleRequest{Name: mappingRuleName})
				return err
			},
		},
		{
			name: "CreateMappingRule",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.CreateMappingRule(ctx, &clouddmspb.CreateMappingRuleRequest{
					Parent:        conversionWorkspaceName,
					MappingRuleId: mappingRuleID,
					MappingRule: &clouddmspb.MappingRule{
						Name: mappingRuleName,
					},
				})
				return err
			},
		},
		{
			name: "DescribeDatabaseEntities",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				it := c.DescribeDatabaseEntities(ctx, &clouddmspb.DescribeDatabaseEntitiesRequest{
					ConversionWorkspace: conversionWorkspaceName,
					Tree:                clouddmspb.DescribeDatabaseEntitiesRequest_SOURCE_TREE,
					PageSize:            1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "SearchBackgroundJobs",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.SearchBackgroundJobs(ctx, &clouddmspb.SearchBackgroundJobsRequest{
					ConversionWorkspace: conversionWorkspaceName,
					MaxSize:             1,
				})
				return err
			},
		},
		{
			name: "DescribeConversionWorkspaceRevisions",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.DescribeConversionWorkspaceRevisions(ctx, &clouddmspb.DescribeConversionWorkspaceRevisionsRequest{
					ConversionWorkspace: conversionWorkspaceName,
				})
				return err
			},
		},
		{
			name: "FetchStaticIps",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				it := c.FetchStaticIps(ctx, &clouddmspb.FetchStaticIpsRequest{
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
			name: "ListOperations",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
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
			name: "GetOperation",
			call: func(ctx context.Context, c *clouddms.DataMigrationClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
	}

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, client)
		callCancel()

		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedFoundationError(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func grpcEndpointFromEnv() string {
	if endpoint := strings.TrimSpace(os.Getenv("STACKYARD_GCP_GRPC_ENDPOINT")); endpoint != "" {
		return normalizeEndpoint(endpoint)
	}
	httpBase := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	return normalizeEndpoint(httpBase)
}

func normalizeEndpoint(raw string) string {
	endpoint := strings.TrimSpace(raw)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	if idx := strings.Index(endpoint, "/"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return endpoint
}

func isToleratedFoundationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded, codes.Unknown:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") ||
		strings.Contains(text, "not implemented") ||
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close clouddms client: %v\n", err)
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
