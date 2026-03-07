package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	workstations "cloud.google.com/go/workstations/apiv1"
	workstationspb "cloud.google.com/go/workstations/apiv1/workstationspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	longrunningpb "google.golang.org/genproto/googleapis/longrunning"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *workstations.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	clusterID := getenv("STACKYARD_GCP_WORKSTATION_CLUSTER_ID", "cluster-1")
	configID := getenv("STACKYARD_GCP_WORKSTATION_CONFIG_ID", "config-1")
	workstationID := getenv("STACKYARD_GCP_WORKSTATION_ID", "workstation-running")
	startableWorkstationID := getenv("STACKYARD_GCP_WORKSTATION_STARTABLE_ID", "workstation-stopped")

	locationParent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	clusterName := fmt.Sprintf("%s/workstationClusters/%s", locationParent, clusterID)
	configName := fmt.Sprintf("%s/workstationConfigs/%s", clusterName, configID)
	workstationName := fmt.Sprintf("%s/workstations/%s", configName, workstationID)
	startableWorkstationName := fmt.Sprintf("%s/workstations/%s", configName, startableWorkstationID)

	fmt.Printf("Stackyard GCP Workstations workstations/apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "workstations",
		},
	}

	client, err := workstations.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create workstations client: %v", err)
	}
	defer closeClient(client.Close)

	createClusterOpName := ""

	calls := []callSpec{
		{
			name: "ListWorkstationClusters",
			call: func(ctx context.Context, c *workstations.Client) error {
				it := c.ListWorkstationClusters(ctx, &workstationspb.ListWorkstationClustersRequest{
					Parent:   locationParent,
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
			name: "GetWorkstationCluster",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.GetWorkstationCluster(ctx, &workstationspb.GetWorkstationClusterRequest{
					Name: clusterName,
				})
				return err
			},
		},
		{
			name: "CreateWorkstationCluster",
			call: func(ctx context.Context, c *workstations.Client) error {
				op, err := c.CreateWorkstationCluster(ctx, &workstationspb.CreateWorkstationClusterRequest{
					Parent:               locationParent,
					WorkstationClusterId: clusterID,
					WorkstationCluster: &workstationspb.WorkstationCluster{
						Name:    clusterName,
						Network: fmt.Sprintf("projects/%s/global/networks/default", projectID),
					},
				})
				if err != nil {
					return err
				}
				createClusterOpName = strings.TrimSpace(op.Name())
				return nil
			},
		},
		{
			name: "UpdateWorkstationCluster",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.UpdateWorkstationCluster(ctx, &workstationspb.UpdateWorkstationClusterRequest{
					WorkstationCluster: &workstationspb.WorkstationCluster{
						Name:    clusterName,
						Network: fmt.Sprintf("projects/%s/global/networks/default", projectID),
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"display_name", "network"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteWorkstationCluster",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.DeleteWorkstationCluster(ctx, &workstationspb.DeleteWorkstationClusterRequest{
					Name: clusterName,
				})
				return err
			},
		},
		{
			name: "ListWorkstationConfigs",
			call: func(ctx context.Context, c *workstations.Client) error {
				it := c.ListWorkstationConfigs(ctx, &workstationspb.ListWorkstationConfigsRequest{
					Parent:   clusterName,
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
			name: "ListUsableWorkstationConfigs",
			call: func(ctx context.Context, c *workstations.Client) error {
				it := c.ListUsableWorkstationConfigs(ctx, &workstationspb.ListUsableWorkstationConfigsRequest{
					Parent:   clusterName,
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
			name: "GetWorkstationConfig",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.GetWorkstationConfig(ctx, &workstationspb.GetWorkstationConfigRequest{
					Name: configName,
				})
				return err
			},
		},
		{
			name: "CreateWorkstationConfig",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.CreateWorkstationConfig(ctx, &workstationspb.CreateWorkstationConfigRequest{
					Parent:              clusterName,
					WorkstationConfigId: configID,
					WorkstationConfig: &workstationspb.WorkstationConfig{
						Name: configName,
						Host: &workstationspb.WorkstationConfig_Host{
							Config: &workstationspb.WorkstationConfig_Host_GceInstance_{
								GceInstance: &workstationspb.WorkstationConfig_Host_GceInstance{
									MachineType: "e2-standard-4",
								},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateWorkstationConfig",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.UpdateWorkstationConfig(ctx, &workstationspb.UpdateWorkstationConfigRequest{
					WorkstationConfig: &workstationspb.WorkstationConfig{
						Name: configName,
						Host: &workstationspb.WorkstationConfig_Host{
							Config: &workstationspb.WorkstationConfig_Host_GceInstance_{
								GceInstance: &workstationspb.WorkstationConfig_Host_GceInstance{
									MachineType: "e2-standard-4",
								},
							},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"display_name", "host"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteWorkstationConfig",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.DeleteWorkstationConfig(ctx, &workstationspb.DeleteWorkstationConfigRequest{
					Name: configName,
				})
				return err
			},
		},
		{
			name: "ListWorkstations",
			call: func(ctx context.Context, c *workstations.Client) error {
				it := c.ListWorkstations(ctx, &workstationspb.ListWorkstationsRequest{
					Parent:   configName,
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
			name: "ListUsableWorkstations",
			call: func(ctx context.Context, c *workstations.Client) error {
				it := c.ListUsableWorkstations(ctx, &workstationspb.ListUsableWorkstationsRequest{
					Parent:   configName,
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
			name: "GetWorkstation",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.GetWorkstation(ctx, &workstationspb.GetWorkstationRequest{
					Name: workstationName,
				})
				return err
			},
		},
		{
			name: "CreateWorkstation",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.CreateWorkstation(ctx, &workstationspb.CreateWorkstationRequest{
					Parent:        configName,
					WorkstationId: "workstation-new",
					Workstation: &workstationspb.Workstation{
						Name: configName + "/workstations/workstation-new",
					},
				})
				return err
			},
		},
		{
			name: "UpdateWorkstation",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.UpdateWorkstation(ctx, &workstationspb.UpdateWorkstationRequest{
					Workstation: &workstationspb.Workstation{
						Name: workstationName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"display_name"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteWorkstation",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.DeleteWorkstation(ctx, &workstationspb.DeleteWorkstationRequest{
					Name: workstationName,
				})
				return err
			},
		},
		{
			name: "StartWorkstation",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.StartWorkstation(ctx, &workstationspb.StartWorkstationRequest{
					Name: startableWorkstationName,
				})
				return err
			},
		},
		{
			name: "StopWorkstation",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.StopWorkstation(ctx, &workstationspb.StopWorkstationRequest{
					Name: workstationName,
				})
				return err
			},
		},
		{
			name: "GenerateAccessToken",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.GenerateAccessToken(ctx, &workstationspb.GenerateAccessTokenRequest{
					Workstation: workstationName,
				})
				return err
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{
					Resource: clusterName,
				})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: clusterName,
					Policy: &iampb.Policy{
						Version: 1,
						Bindings: []*iampb.Binding{
							{
								Role:    "roles/workstations.user",
								Members: []string{"user:stackyard@example.com"},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *workstations.Client) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    clusterName,
					Permissions: []string{"workstations.workstationClusters.get"},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *workstations.Client) error {
				opName := firstNonEmpty(createClusterOpName, locationParent+"/operations/op-done")
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: opName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *workstations.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationParent,
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
			call: func(ctx context.Context, c *workstations.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{
					Name: locationParent + "/operations/op-1",
				})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *workstations.Client) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{
					Name: locationParent + "/operations/op-1",
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
		fmt.Fprintf(os.Stderr, "warning: close workstations client: %v\n", err)
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
