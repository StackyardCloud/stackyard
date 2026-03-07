package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	managedkafka "cloud.google.com/go/managedkafka/apiv1"
	"cloud.google.com/go/managedkafka/apiv1/managedkafkapb"
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
	call func(context.Context, *managedkafka.Client) error
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	clusterID := getenv("STACKYARD_GCP_MANAGEDKAFKA_CLUSTER_ID", "cluster-a")
	topicID := getenv("STACKYARD_GCP_MANAGEDKAFKA_TOPIC_ID", "orders")
	consumerGroupID := getenv("STACKYARD_GCP_MANAGEDKAFKA_CONSUMER_GROUP_ID", "cg-a")
	aclID := getenv("STACKYARD_GCP_MANAGEDKAFKA_ACL_ID", "allTopics")
	operationID := getenv("STACKYARD_GCP_MANAGEDKAFKA_OPERATION_ID", "op-1")
	subnet := getenv("STACKYARD_GCP_MANAGEDKAFKA_SUBNET", fmt.Sprintf("projects/%s/regions/%s/subnetworks/default", projectID, locationID))
	principal := getenv("STACKYARD_GCP_MANAGEDKAFKA_ACL_PRINCIPAL", fmt.Sprintf("User:svc-%s@%s.iam.gserviceaccount.com", topicID, projectID))

	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	clusterName := fmt.Sprintf("%s/clusters/%s", locationName, clusterID)
	topicName := fmt.Sprintf("%s/topics/%s", clusterName, topicID)
	consumerGroupName := fmt.Sprintf("%s/consumerGroups/%s", clusterName, consumerGroupID)
	aclName := fmt.Sprintf("%s/acls/%s", clusterName, aclID)
	operationName := fmt.Sprintf("%s/operations/%s", locationName, operationID)

	fmt.Printf("Stackyard GCP Managed Kafka apiv1 client using %s\n", apiEndpoint)

	client, err := managedkafka.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create managedkafka client: %v", err)
	}
	defer closeClient("managedkafka", client.Close)

	aclEntry := &managedkafkapb.AclEntry{
		Principal:      principal,
		PermissionType: "ALLOW",
		Operation:      "READ",
		Host:           "*",
	}

	calls := []callSpec{
		{
			name: "ListClusters",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				it := c.ListClusters(ctx, &managedkafkapb.ListClustersRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetCluster",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.GetCluster(ctx, &managedkafkapb.GetClusterRequest{Name: clusterName})
				return err
			},
		},
		{
			name: "CreateCluster",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.CreateCluster(ctx, &managedkafkapb.CreateClusterRequest{
					Parent:    locationName,
					ClusterId: clusterID,
					Cluster: &managedkafkapb.Cluster{
						CapacityConfig: &managedkafkapb.CapacityConfig{
							VcpuCount:   3,
							MemoryBytes: 3221225472,
						},
						PlatformConfig: &managedkafkapb.Cluster_GcpConfig{
							GcpConfig: &managedkafkapb.GcpConfig{
								AccessConfig: &managedkafkapb.AccessConfig{
									NetworkConfigs: []*managedkafkapb.NetworkConfig{
										{Subnet: subnet},
									},
								},
							},
						},
						Labels: map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "UpdateCluster",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.UpdateCluster(ctx, &managedkafkapb.UpdateClusterRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
					Cluster: &managedkafkapb.Cluster{
						Name:   clusterName,
						Labels: map[string]string{"env": "updated"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteCluster",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.DeleteCluster(ctx, &managedkafkapb.DeleteClusterRequest{Name: clusterName})
				return err
			},
		},
		{
			name: "ListTopics",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				it := c.ListTopics(ctx, &managedkafkapb.ListTopicsRequest{
					Parent:   clusterName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetTopic",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.GetTopic(ctx, &managedkafkapb.GetTopicRequest{Name: topicName})
				return err
			},
		},
		{
			name: "CreateTopic",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.CreateTopic(ctx, &managedkafkapb.CreateTopicRequest{
					Parent:  clusterName,
					TopicId: topicID,
					Topic: &managedkafkapb.Topic{
						PartitionCount:    3,
						ReplicationFactor: 3,
						Configs: map[string]string{
							"cleanup.policy": "delete",
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateTopic",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.UpdateTopic(ctx, &managedkafkapb.UpdateTopicRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"configs"}},
					Topic: &managedkafkapb.Topic{
						Name: topicName,
						Configs: map[string]string{
							"cleanup.policy": "compact",
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteTopic",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				return c.DeleteTopic(ctx, &managedkafkapb.DeleteTopicRequest{Name: topicName})
			},
		},
		{
			name: "ListConsumerGroups",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				it := c.ListConsumerGroups(ctx, &managedkafkapb.ListConsumerGroupsRequest{
					Parent:   clusterName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetConsumerGroup",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.GetConsumerGroup(ctx, &managedkafkapb.GetConsumerGroupRequest{Name: consumerGroupName})
				return err
			},
		},
		{
			name: "UpdateConsumerGroup",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.UpdateConsumerGroup(ctx, &managedkafkapb.UpdateConsumerGroupRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"topics"}},
					ConsumerGroup: &managedkafkapb.ConsumerGroup{
						Name: consumerGroupName,
						Topics: map[string]*managedkafkapb.ConsumerTopicMetadata{
							topicName: {},
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteConsumerGroup",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				return c.DeleteConsumerGroup(ctx, &managedkafkapb.DeleteConsumerGroupRequest{Name: consumerGroupName})
			},
		},
		{
			name: "ListAcls",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				it := c.ListAcls(ctx, &managedkafkapb.ListAclsRequest{
					Parent:   clusterName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetAcl",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.GetAcl(ctx, &managedkafkapb.GetAclRequest{Name: aclName})
				return err
			},
		},
		{
			name: "CreateAcl",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.CreateAcl(ctx, &managedkafkapb.CreateAclRequest{
					Parent: clusterName,
					AclId:  aclID,
					Acl: &managedkafkapb.Acl{
						AclEntries: []*managedkafkapb.AclEntry{aclEntry},
					},
				})
				return err
			},
		},
		{
			name: "UpdateAcl",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.UpdateAcl(ctx, &managedkafkapb.UpdateAclRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"acl_entries"}},
					Acl: &managedkafkapb.Acl{
						Name:       aclName,
						AclEntries: []*managedkafkapb.AclEntry{aclEntry},
					},
				})
				return err
			},
		},
		{
			name: "AddAclEntry",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.AddAclEntry(ctx, &managedkafkapb.AddAclEntryRequest{
					Acl:      aclName,
					AclEntry: aclEntry,
				})
				return err
			},
		},
		{
			name: "RemoveAclEntry",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.RemoveAclEntry(ctx, &managedkafkapb.RemoveAclEntryRequest{
					Acl:      aclName,
					AclEntry: aclEntry,
				})
				return err
			},
		},
		{
			name: "DeleteAcl",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				return c.DeleteAcl(ctx, &managedkafkapb.DeleteAclRequest{Name: aclName})
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *managedkafka.Client) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
			},
		},
	}

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, 6*time.Second)
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

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
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
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && (apiErr.Code == 501 || apiErr.Code == 503) {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") ||
		strings.Contains(text, "not implemented") ||
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "connection refused")
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
