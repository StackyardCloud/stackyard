package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *instance.InstanceAdminClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	location := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	instanceID := getenv("STACKYARD_GCP_SPANNER_INSTANCE", "stackyard-instance")
	configID := getenv("STACKYARD_GCP_SPANNER_INSTANCE_CONFIG", "custom-stackyard-primary")
	newConfigID := getenv("STACKYARD_GCP_SPANNER_INSTANCE_CONFIG_NEW", "custom-stackyard-new")
	newInstanceID := getenv("STACKYARD_GCP_SPANNER_INSTANCE_NEW", "stackyard-instance-new")
	partitionID := getenv("STACKYARD_GCP_SPANNER_INSTANCE_PARTITION", "partition-a")
	newPartitionID := getenv("STACKYARD_GCP_SPANNER_INSTANCE_PARTITION_NEW", "partition-new")

	projectName := fmt.Sprintf("projects/%s", projectID)
	instanceName := fmt.Sprintf("%s/instances/%s", projectName, instanceID)
	configName := fmt.Sprintf("%s/instanceConfigs/%s", projectName, configID)
	newConfigName := fmt.Sprintf("%s/instanceConfigs/%s", projectName, newConfigID)
	newInstanceName := fmt.Sprintf("%s/instances/%s", projectName, newInstanceID)
	partitionName := fmt.Sprintf("%s/instancePartitions/%s", instanceName, partitionID)
	newPartitionName := fmt.Sprintf("%s/instancePartitions/%s", instanceName, newPartitionID)
	operationName := fmt.Sprintf("%s/operations/create-instance-%s", instanceName, instanceID)

	fmt.Printf("Stackyard GCP Spanner Admin Instance apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, projectID, location); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "spanner-admin-instance",
		},
	}

	client, err := instance.NewInstanceAdminRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create spanner admin instance client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListInstanceConfigs",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				it := c.ListInstanceConfigs(ctx, &instancepb.ListInstanceConfigsRequest{Parent: projectName, PageSize: 1})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "GetInstanceConfig",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				_, err := c.GetInstanceConfig(ctx, &instancepb.GetInstanceConfigRequest{Name: configName})
				return err
			},
		},
		{
			name: "CreateInstanceConfigAndGetOperation",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				op, err := c.CreateInstanceConfig(ctx, &instancepb.CreateInstanceConfigRequest{
					Parent:           projectName,
					InstanceConfigId: newConfigID,
					InstanceConfig: &instancepb.InstanceConfig{
						DisplayName: "Stackyard New Config",
					},
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(op.Name()) == "" {
					return fmt.Errorf("create instance config returned empty operation name")
				}
				_, err = c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: op.Name()})
				return err
			},
		},
		{
			name: "UpdateInstanceConfigAndGetOperation",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				op, err := c.UpdateInstanceConfig(ctx, &instancepb.UpdateInstanceConfigRequest{
					InstanceConfig: &instancepb.InstanceConfig{
						Name:        configName,
						DisplayName: "Updated Config",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				if err != nil {
					return err
				}
				_, err = c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: op.Name()})
				return err
			},
		},
		{
			name: "ListInstanceConfigOperations",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				it := c.ListInstanceConfigOperations(ctx, &instancepb.ListInstanceConfigOperationsRequest{Parent: projectName, PageSize: 1})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "ListInstances",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				it := c.ListInstances(ctx, &instancepb.ListInstancesRequest{Parent: projectName, PageSize: 1})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "GetInstance",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				_, err := c.GetInstance(ctx, &instancepb.GetInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "CreateInstanceAndGetOperation",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				op, err := c.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
					Parent:     projectName,
					InstanceId: newInstanceID,
					Instance: &instancepb.Instance{
						Config:      configName,
						DisplayName: "Stackyard New Instance",
						NodeCount:   1,
					},
				})
				if err != nil {
					return err
				}
				_, err = c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: op.Name()})
				return err
			},
		},
		{
			name: "UpdateInstanceAndGetOperation",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				op, err := c.UpdateInstance(ctx, &instancepb.UpdateInstanceRequest{
					Instance: &instancepb.Instance{
						Name:        instanceName,
						DisplayName: "Updated Instance",
						NodeCount:   2,
					},
					FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name", "node_count"}},
				})
				if err != nil {
					return err
				}
				_, err = c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: op.Name()})
				return err
			},
		},
		{
			name: "ListInstancePartitions",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				it := c.ListInstancePartitions(ctx, &instancepb.ListInstancePartitionsRequest{Parent: instanceName, PageSize: 1})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "GetInstancePartition",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				_, err := c.GetInstancePartition(ctx, &instancepb.GetInstancePartitionRequest{Name: partitionName})
				return err
			},
		},
		{
			name: "CreateInstancePartitionAndGetOperation",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				op, err := c.CreateInstancePartition(ctx, &instancepb.CreateInstancePartitionRequest{
					Parent:              instanceName,
					InstancePartitionId: newPartitionID,
					InstancePartition: &instancepb.InstancePartition{
						DisplayName: "Stackyard New Partition",
					},
				})
				if err != nil {
					return err
				}
				_, err = c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: op.Name()})
				return err
			},
		},
		{
			name: "UpdateInstancePartitionAndGetOperation",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				op, err := c.UpdateInstancePartition(ctx, &instancepb.UpdateInstancePartitionRequest{
					InstancePartition: &instancepb.InstancePartition{
						Name:        partitionName,
						DisplayName: "Updated Partition",
					},
					FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				if err != nil {
					return err
				}
				_, err = c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: op.Name()})
				return err
			},
		},
		{
			name: "ListInstancePartitionOperations",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				it := c.ListInstancePartitionOperations(ctx, &instancepb.ListInstancePartitionOperationsRequest{Parent: instanceName, PageSize: 1})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "MoveInstanceAndGetOperation",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				op, err := c.MoveInstance(ctx, &instancepb.MoveInstanceRequest{
					Name:         instanceName,
					TargetConfig: fmt.Sprintf("%s/instanceConfigs/custom-stackyard-analytics", projectName),
				})
				if err != nil {
					return err
				}
				_, err = c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: op.Name()})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: instanceName,
					Policy: &iampb.Policy{
						Bindings: []*iampb.Binding{
							{
								Role:    "roles/spanner.admin",
								Members: []string{"user:stackyard@example.com"},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: instanceName})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    instanceName,
					Permissions: []string{"spanner.instances.get", "resourcemanager.projects.get"},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: fmt.Sprintf("%s/operations", projectName), PageSize: 1})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteInstancePartition",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				return c.DeleteInstancePartition(ctx, &instancepb.DeleteInstancePartitionRequest{Name: newPartitionName})
			},
		},
		{
			name: "DeleteInstance",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				return c.DeleteInstance(ctx, &instancepb.DeleteInstanceRequest{Name: newInstanceName})
			},
		},
		{
			name: "DeleteInstanceConfig",
			call: func(ctx context.Context, c *instance.InstanceAdminClient) error {
				return c.DeleteInstanceConfig(ctx, &instancepb.DeleteInstanceConfigRequest{Name: newConfigName})
			},
		},
	}

	for _, call := range calls {
		if err := call.call(ctx, client); err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		logf("%s succeeded", call.name)
	}

	fmt.Println("Done.")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close spanner admin instance client: %v\n", err)
	}
}

func waitForStackyardReady(ctx context.Context, apiEndpoint, projectID, location string) error {
	readyURL := fmt.Sprintf("%s/v1/projects/%s/locations/%s/spanner_admin_instance?stackyard_contract_probe=1&typedSuccess=1", strings.TrimRight(apiEndpoint, "/"), projectID, location)
	httpClient := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "spanner-admin-instance")

		resp, err := httpClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("ready probe status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s: %w", readyURL, lastErr)
		}
		time.Sleep(300 * time.Millisecond)
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
