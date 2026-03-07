package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	iampb "cloud.google.com/go/iam/apiv1/iampb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	networkservices "cloud.google.com/go/networkservices/apiv1"
	networkservicespb "cloud.google.com/go/networkservices/apiv1/networkservicespb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *networkservices.Client, *networkservices.DepClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	projectName := fmt.Sprintf("projects/%s", projectID)

	endpointPolicyName := parent + "/endpointPolicies/ep-1"
	gatewayName := parent + "/gateways/gw-1"
	grpcRouteName := parent + "/grpcRoutes/gr-1"
	httpRouteName := parent + "/httpRoutes/hr-1"
	tcpRouteName := parent + "/tcpRoutes/tr-1"
	tlsRouteName := parent + "/tlsRoutes/tlr-1"
	serviceBindingName := parent + "/serviceBindings/sb-1"
	meshName := parent + "/meshes/mesh-1"
	serviceLbPolicyName := parent + "/serviceLbPolicies/slp-1"
	gatewayRouteViewName := gatewayName + "/routeViews/default"
	meshRouteViewName := meshName + "/routeViews/default"
	operationName := parent + "/operations/op-1"

	lbTrafficExtensionName := parent + "/lbTrafficExtensions/lbt-1"
	lbRouteExtensionName := parent + "/lbRouteExtensions/lbr-1"
	lbEdgeExtensionName := parent + "/lbEdgeExtensions/lbe-1"
	authzExtensionName := parent + "/authzExtensions/authz-1"

	fmt.Printf("Stackyard GCP Network Services apiv1 clients using %s\n", apiEndpoint)

	client, err := networkservices.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create networkservices client: %v", err)
	}
	defer closeClient("networkservices", client.Close)

	depClient, err := networkservices.NewDepRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create networkservices dep client: %v", err)
	}
	defer closeClient("networkservices dep", depClient.Close)

	calls := []callSpec{
		{
			name: "ListEndpointPolicies",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListEndpointPolicies(ctx, &networkservicespb.ListEndpointPoliciesRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetEndpointPolicy",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetEndpointPolicy(ctx, &networkservicespb.GetEndpointPolicyRequest{Name: endpointPolicyName})
				return err
			},
		},
		{
			name: "ListGateways",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListGateways(ctx, &networkservicespb.ListGatewaysRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetGateway",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetGateway(ctx, &networkservicespb.GetGatewayRequest{Name: gatewayName})
				return err
			},
		},
		{
			name: "ListGrpcRoutes",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListGrpcRoutes(ctx, &networkservicespb.ListGrpcRoutesRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetGrpcRoute",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetGrpcRoute(ctx, &networkservicespb.GetGrpcRouteRequest{Name: grpcRouteName})
				return err
			},
		},
		{
			name: "ListHttpRoutes",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListHttpRoutes(ctx, &networkservicespb.ListHttpRoutesRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetHttpRoute",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetHttpRoute(ctx, &networkservicespb.GetHttpRouteRequest{Name: httpRouteName})
				return err
			},
		},
		{
			name: "ListTcpRoutes",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListTcpRoutes(ctx, &networkservicespb.ListTcpRoutesRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetTcpRoute",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetTcpRoute(ctx, &networkservicespb.GetTcpRouteRequest{Name: tcpRouteName})
				return err
			},
		},
		{
			name: "ListTlsRoutes",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListTlsRoutes(ctx, &networkservicespb.ListTlsRoutesRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetTlsRoute",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetTlsRoute(ctx, &networkservicespb.GetTlsRouteRequest{Name: tlsRouteName})
				return err
			},
		},
		{
			name: "ListServiceBindings",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListServiceBindings(ctx, &networkservicespb.ListServiceBindingsRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetServiceBinding",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetServiceBinding(ctx, &networkservicespb.GetServiceBindingRequest{Name: serviceBindingName})
				return err
			},
		},
		{
			name: "ListMeshes",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListMeshes(ctx, &networkservicespb.ListMeshesRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetMesh",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetMesh(ctx, &networkservicespb.GetMeshRequest{Name: meshName})
				return err
			},
		},
		{
			name: "ListServiceLbPolicies",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListServiceLbPolicies(ctx, &networkservicespb.ListServiceLbPoliciesRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetServiceLbPolicy",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetServiceLbPolicy(ctx, &networkservicespb.GetServiceLbPolicyRequest{Name: serviceLbPolicyName})
				return err
			},
		},
		{
			name: "GetGatewayRouteView",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetGatewayRouteView(ctx, &networkservicespb.GetGatewayRouteViewRequest{Name: gatewayRouteViewName})
				return err
			},
		},
		{
			name: "GetMeshRouteView",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetMeshRouteView(ctx, &networkservicespb.GetMeshRouteViewRequest{Name: meshRouteViewName})
				return err
			},
		},
		{
			name: "ListGatewayRouteViews",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListGatewayRouteViews(ctx, &networkservicespb.ListGatewayRouteViewsRequest{Parent: gatewayName, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListMeshRouteViews",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListMeshRouteViews(ctx, &networkservicespb.ListMeshRouteViewsRequest{Parent: meshName, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: gatewayName})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{Resource: gatewayName, Policy: &iampb.Policy{}})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    gatewayName,
					Permissions: []string{"networkservices.gateways.get"},
				})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{Name: projectName, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *networkservices.Client, _ *networkservices.DepClient) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
			},
		},
		{
			name: "ListLbTrafficExtensions",
			call: func(ctx context.Context, _ *networkservices.Client, dep *networkservices.DepClient) error {
				it := dep.ListLbTrafficExtensions(ctx, &networkservicespb.ListLbTrafficExtensionsRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetLbTrafficExtension",
			call: func(ctx context.Context, _ *networkservices.Client, dep *networkservices.DepClient) error {
				_, err := dep.GetLbTrafficExtension(ctx, &networkservicespb.GetLbTrafficExtensionRequest{Name: lbTrafficExtensionName})
				return err
			},
		},
		{
			name: "ListLbRouteExtensions",
			call: func(ctx context.Context, _ *networkservices.Client, dep *networkservices.DepClient) error {
				it := dep.ListLbRouteExtensions(ctx, &networkservicespb.ListLbRouteExtensionsRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetLbRouteExtension",
			call: func(ctx context.Context, _ *networkservices.Client, dep *networkservices.DepClient) error {
				_, err := dep.GetLbRouteExtension(ctx, &networkservicespb.GetLbRouteExtensionRequest{Name: lbRouteExtensionName})
				return err
			},
		},
		{
			name: "ListLbEdgeExtensions",
			call: func(ctx context.Context, _ *networkservices.Client, dep *networkservices.DepClient) error {
				it := dep.ListLbEdgeExtensions(ctx, &networkservicespb.ListLbEdgeExtensionsRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetLbEdgeExtension",
			call: func(ctx context.Context, _ *networkservices.Client, dep *networkservices.DepClient) error {
				_, err := dep.GetLbEdgeExtension(ctx, &networkservicespb.GetLbEdgeExtensionRequest{Name: lbEdgeExtensionName})
				return err
			},
		},
		{
			name: "ListAuthzExtensions",
			call: func(ctx context.Context, _ *networkservices.Client, dep *networkservices.DepClient) error {
				it := dep.ListAuthzExtensions(ctx, &networkservicespb.ListAuthzExtensionsRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetAuthzExtension",
			call: func(ctx context.Context, _ *networkservices.Client, dep *networkservices.DepClient) error {
				_, err := dep.GetAuthzExtension(ctx, &networkservicespb.GetAuthzExtensionRequest{Name: authzExtensionName})
				return err
			},
		},
	}

	for _, c := range calls {
		err := c.call(ctx, client, depClient)
		switch {
		case err == nil:
			logf("%s succeeded", c.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", c.name)
		default:
			exitf("%s failed: %v", c.name, err)
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
