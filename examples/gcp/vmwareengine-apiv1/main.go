package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	vmwareengine "cloud.google.com/go/vmwareengine/apiv1"
	"cloud.google.com/go/vmwareengine/apiv1/vmwareenginepb"
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
	call func(context.Context, *vmwareengine.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	privateCloudID := getenv("STACKYARD_GCP_PRIVATE_CLOUD_ID", "private-cloud-1")
	clusterID := getenv("STACKYARD_GCP_CLUSTER_ID", "cluster-1")
	networkPolicyID := getenv("STACKYARD_GCP_NETWORK_POLICY_ID", "network-policy-1")
	vmwareEngineNetworkID := getenv("STACKYARD_GCP_VMWARE_ENGINE_NETWORK_ID", "vmware-engine-network-1")
	privateConnectionID := getenv("STACKYARD_GCP_PRIVATE_CONNECTION_ID", "private-connection-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	parent := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	privateCloudName := fmt.Sprintf("%s/privateClouds/%s", parent, privateCloudID)
	clusterName := fmt.Sprintf("%s/clusters/%s", privateCloudName, clusterID)
	networkPolicyName := fmt.Sprintf("%s/networkPolicies/%s", parent, networkPolicyID)
	vmwareEngineNetworkName := fmt.Sprintf("%s/vmwareEngineNetworks/%s", parent, vmwareEngineNetworkID)
	privateConnectionName := fmt.Sprintf("%s/privateConnections/%s", parent, privateConnectionID)
	dnsForwardingName := fmt.Sprintf("%s/dnsForwarding", privateCloudName)
	operationName := fmt.Sprintf("%s/operations/vmwareengine-op-1", parent)

	fmt.Printf("Stackyard GCP VMware Engine apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "vmwareengine",
		},
	}

	client, err := vmwareengine.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create vmwareengine client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
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
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListPrivateClouds",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				it := c.ListPrivateClouds(ctx, &vmwareenginepb.ListPrivateCloudsRequest{
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
			name: "GetPrivateCloud",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				_, err := c.GetPrivateCloud(ctx, &vmwareenginepb.GetPrivateCloudRequest{Name: privateCloudName})
				return err
			},
		},
		{
			name: "CreatePrivateCloud",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				op, err := c.CreatePrivateCloud(ctx, &vmwareenginepb.CreatePrivateCloudRequest{
					Parent:         parent,
					PrivateCloudId: privateCloudID,
					PrivateCloud: &vmwareenginepb.PrivateCloud{
						Description: "Stackyard VMware Engine private cloud",
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
			name: "UpdatePrivateCloud",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				op, err := c.UpdatePrivateCloud(ctx, &vmwareenginepb.UpdatePrivateCloudRequest{
					PrivateCloud: &vmwareenginepb.PrivateCloud{
						Name:        privateCloudName,
						Description: "Stackyard VMware Engine private cloud updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"description"},
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
			name: "DeletePrivateCloud",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				op, err := c.DeletePrivateCloud(ctx, &vmwareenginepb.DeletePrivateCloudRequest{Name: privateCloudName})
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
			name: "UndeletePrivateCloud",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				op, err := c.UndeletePrivateCloud(ctx, &vmwareenginepb.UndeletePrivateCloudRequest{Name: privateCloudName})
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
			name: "ListClusters",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				it := c.ListClusters(ctx, &vmwareenginepb.ListClustersRequest{
					Parent:   privateCloudName,
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
			name: "GetCluster",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				_, err := c.GetCluster(ctx, &vmwareenginepb.GetClusterRequest{Name: clusterName})
				return err
			},
		},
		{
			name: "CreateCluster",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				op, err := c.CreateCluster(ctx, &vmwareenginepb.CreateClusterRequest{
					Parent:    privateCloudName,
					ClusterId: clusterID,
					Cluster: &vmwareenginepb.Cluster{
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
			name: "ListNetworkPolicies",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				it := c.ListNetworkPolicies(ctx, &vmwareenginepb.ListNetworkPoliciesRequest{
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
			name: "GetNetworkPolicy",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				_, err := c.GetNetworkPolicy(ctx, &vmwareenginepb.GetNetworkPolicyRequest{Name: networkPolicyName})
				return err
			},
		},
		{
			name: "CreateNetworkPolicy",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				op, err := c.CreateNetworkPolicy(ctx, &vmwareenginepb.CreateNetworkPolicyRequest{
					Parent:          parent,
					NetworkPolicyId: networkPolicyID,
					NetworkPolicy: &vmwareenginepb.NetworkPolicy{
						Description: "Stackyard network policy",
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
			name: "FetchNetworkPolicyExternalAddresses",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				it := c.FetchNetworkPolicyExternalAddresses(ctx, &vmwareenginepb.FetchNetworkPolicyExternalAddressesRequest{
					NetworkPolicy: networkPolicyName,
					PageSize:      1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "ListVmwareEngineNetworks",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				it := c.ListVmwareEngineNetworks(ctx, &vmwareenginepb.ListVmwareEngineNetworksRequest{
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
			name: "GetVmwareEngineNetwork",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				_, err := c.GetVmwareEngineNetwork(ctx, &vmwareenginepb.GetVmwareEngineNetworkRequest{
					Name: vmwareEngineNetworkName,
				})
				return err
			},
		},
		{
			name: "CreateVmwareEngineNetwork",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				op, err := c.CreateVmwareEngineNetwork(ctx, &vmwareenginepb.CreateVmwareEngineNetworkRequest{
					Parent:                parent,
					VmwareEngineNetworkId: vmwareEngineNetworkID,
					VmwareEngineNetwork: &vmwareenginepb.VmwareEngineNetwork{
						Description: "Stackyard VMware Engine network",
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
			name: "ListPrivateConnections",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				it := c.ListPrivateConnections(ctx, &vmwareenginepb.ListPrivateConnectionsRequest{
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
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				_, err := c.GetPrivateConnection(ctx, &vmwareenginepb.GetPrivateConnectionRequest{Name: privateConnectionName})
				return err
			},
		},
		{
			name: "CreatePrivateConnection",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				op, err := c.CreatePrivateConnection(ctx, &vmwareenginepb.CreatePrivateConnectionRequest{
					Parent:              parent,
					PrivateConnectionId: privateConnectionID,
					PrivateConnection: &vmwareenginepb.PrivateConnection{
						Description: "Stackyard private connection",
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
			name: "ShowNsxCredentials",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				_, err := c.ShowNsxCredentials(ctx, &vmwareenginepb.ShowNsxCredentialsRequest{PrivateCloud: privateCloudName})
				return err
			},
		},
		{
			name: "GetDnsForwarding",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				_, err := c.GetDnsForwarding(ctx, &vmwareenginepb.GetDnsForwardingRequest{Name: dnsForwardingName})
				return err
			},
		},
		{
			name: "UpdateDnsForwarding",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
				op, err := c.UpdateDnsForwarding(ctx, &vmwareenginepb.UpdateDnsForwardingRequest{
					DnsForwarding: &vmwareenginepb.DnsForwarding{
						Name: dnsForwardingName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"forwarding_rules"},
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
			name: "ListOperations",
			call: func(ctx context.Context, c *vmwareengine.Client) error {
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
			call: func(ctx context.Context, c *vmwareengine.Client) error {
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
		fmt.Fprintf(os.Stderr, "warning: close vmwareengine client: %v\n", err)
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
