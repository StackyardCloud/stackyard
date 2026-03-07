package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	edgenetwork "cloud.google.com/go/edgenetwork/apiv1"
	"cloud.google.com/go/edgenetwork/apiv1/edgenetworkpb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *edgenetwork.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	zoneID := getenv("STACKYARD_GCP_EDGENETWORK_ZONE_ID", "us-central1-a")
	networkID := getenv("STACKYARD_GCP_EDGENETWORK_NETWORK_ID", "mesh-network")
	subnetID := getenv("STACKYARD_GCP_EDGENETWORK_SUBNET_ID", "mesh-subnet")
	interconnectID := getenv("STACKYARD_GCP_EDGENETWORK_INTERCONNECT_ID", "edge-ic")
	attachmentID := getenv("STACKYARD_GCP_EDGENETWORK_ATTACHMENT_ID", "edge-attach")
	routerID := getenv("STACKYARD_GCP_EDGENETWORK_ROUTER_ID", "edge-router")
	operationID := getenv("STACKYARD_GCP_EDGENETWORK_OPERATION_ID", "op-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	zoneName := locationName + "/zones/" + zoneID
	networkName := locationName + "/networks/" + networkID
	subnetName := zoneName + "/subnets/" + subnetID
	interconnectName := locationName + "/interconnects/" + interconnectID
	attachmentName := locationName + "/interconnectAttachments/" + attachmentID
	routerName := zoneName + "/routers/" + routerID
	operationName := locationName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Distributed Cloud Edge Network apiv1 client using %s\n", apiEndpoint)

	client, err := edgenetwork.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create edgenetwork client: %v", err)
	}
	defer closeClient("edgenetwork client", client.Close)

	calls := []callSpec{
		{
			name: "InitializeZone",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.InitializeZone(ctx, &edgenetworkpb.InitializeZoneRequest{Name: zoneName})
				return err
			},
		},
		{
			name: "ListZones",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				it := c.ListZones(ctx, &edgenetworkpb.ListZonesRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetZone",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.GetZone(ctx, &edgenetworkpb.GetZoneRequest{Name: zoneName})
				return err
			},
		},
		{
			name: "ListNetworks",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				it := c.ListNetworks(ctx, &edgenetworkpb.ListNetworksRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetNetwork",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.GetNetwork(ctx, &edgenetworkpb.GetNetworkRequest{Name: networkName})
				return err
			},
		},
		{
			name: "DiagnoseNetwork",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.DiagnoseNetwork(ctx, &edgenetworkpb.DiagnoseNetworkRequest{Name: networkName})
				return err
			},
		},
		{
			name: "CreateNetwork",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.CreateNetwork(ctx, &edgenetworkpb.CreateNetworkRequest{
					Parent:    locationName,
					NetworkId: networkID,
					Network:   &edgenetworkpb.Network{Name: networkName},
				})
				return err
			},
		},
		{
			name: "DeleteNetwork",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.DeleteNetwork(ctx, &edgenetworkpb.DeleteNetworkRequest{Name: networkName})
				return err
			},
		},
		{
			name: "ListSubnets",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				it := c.ListSubnets(ctx, &edgenetworkpb.ListSubnetsRequest{Parent: zoneName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetSubnet",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.GetSubnet(ctx, &edgenetworkpb.GetSubnetRequest{Name: subnetName})
				return err
			},
		},
		{
			name: "CreateSubnet",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.CreateSubnet(ctx, &edgenetworkpb.CreateSubnetRequest{
					Parent:   zoneName,
					SubnetId: subnetID,
					Subnet:   &edgenetworkpb.Subnet{Name: subnetName},
				})
				return err
			},
		},
		{
			name: "UpdateSubnet",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.UpdateSubnet(ctx, &edgenetworkpb.UpdateSubnetRequest{
					Subnet: &edgenetworkpb.Subnet{Name: subnetName},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"description"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteSubnet",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.DeleteSubnet(ctx, &edgenetworkpb.DeleteSubnetRequest{Name: subnetName})
				return err
			},
		},
		{
			name: "ListInterconnects",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				it := c.ListInterconnects(ctx, &edgenetworkpb.ListInterconnectsRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetInterconnect",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.GetInterconnect(ctx, &edgenetworkpb.GetInterconnectRequest{Name: interconnectName})
				return err
			},
		},
		{
			name: "DiagnoseInterconnect",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.DiagnoseInterconnect(ctx, &edgenetworkpb.DiagnoseInterconnectRequest{Name: interconnectName})
				return err
			},
		},
		{
			name: "ListInterconnectAttachments",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				it := c.ListInterconnectAttachments(ctx, &edgenetworkpb.ListInterconnectAttachmentsRequest{Parent: locationName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetInterconnectAttachment",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.GetInterconnectAttachment(ctx, &edgenetworkpb.GetInterconnectAttachmentRequest{Name: attachmentName})
				return err
			},
		},
		{
			name: "CreateInterconnectAttachment",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.CreateInterconnectAttachment(ctx, &edgenetworkpb.CreateInterconnectAttachmentRequest{
					Parent:                   locationName,
					InterconnectAttachmentId: attachmentID,
					InterconnectAttachment:   &edgenetworkpb.InterconnectAttachment{Name: attachmentName},
				})
				return err
			},
		},
		{
			name: "DeleteInterconnectAttachment",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.DeleteInterconnectAttachment(ctx, &edgenetworkpb.DeleteInterconnectAttachmentRequest{Name: attachmentName})
				return err
			},
		},
		{
			name: "ListRouters",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				it := c.ListRouters(ctx, &edgenetworkpb.ListRoutersRequest{Parent: zoneName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetRouter",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.GetRouter(ctx, &edgenetworkpb.GetRouterRequest{Name: routerName})
				return err
			},
		},
		{
			name: "DiagnoseRouter",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.DiagnoseRouter(ctx, &edgenetworkpb.DiagnoseRouterRequest{Name: routerName})
				return err
			},
		},
		{
			name: "CreateRouter",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.CreateRouter(ctx, &edgenetworkpb.CreateRouterRequest{
					Parent:   zoneName,
					RouterId: routerID,
					Router:   &edgenetworkpb.Router{Name: routerName},
				})
				return err
			},
		},
		{
			name: "UpdateRouter",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.UpdateRouter(ctx, &edgenetworkpb.UpdateRouterRequest{
					Router: &edgenetworkpb.Router{Name: routerName},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"description"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteRouter",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.DeleteRouter(ctx, &edgenetworkpb.DeleteRouterRequest{Name: routerName})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationName,
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
			call: func(ctx context.Context, c *edgenetwork.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *edgenetwork.Client) error {
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

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s: %v\n", label, err)
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
