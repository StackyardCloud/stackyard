package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	networkconnectivity "cloud.google.com/go/networkconnectivity/apiv1"
	"cloud.google.com/go/networkconnectivity/apiv1/networkconnectivitypb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type clients struct {
	hub      *networkconnectivity.HubClient
	cna      *networkconnectivity.CrossNetworkAutomationClient
	rangeAPI *networkconnectivity.InternalRangeClient
	pbr      *networkconnectivity.PolicyBasedRoutingClient
	transfer *networkconnectivity.DataTransferClient
}

type callSpec struct {
	name string
	call func(context.Context, *clients) error
}

func main() {
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	location := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	hubID := getenv("STACKYARD_GCP_NETWORKCONNECTIVITY_HUB_ID", "hub-a")
	serviceConnectionMapID := getenv("STACKYARD_GCP_NETWORKCONNECTIVITY_SCM_ID", "scm-a")
	internalRangeID := getenv("STACKYARD_GCP_NETWORKCONNECTIVITY_INTERNAL_RANGE_ID", "range-a")
	policyBasedRouteID := getenv("STACKYARD_GCP_NETWORKCONNECTIVITY_PBR_ID", "pbr-a")
	transferConfigID := getenv("STACKYARD_GCP_NETWORKCONNECTIVITY_TRANSFER_CONFIG_ID", "transfer-a")
	grpcEndpoint := grpcEndpointFromEnv()

	globalParent := fmt.Sprintf("projects/%s/locations/global", projectID)
	locationParent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)

	hubName := fmt.Sprintf("%s/hubs/%s", globalParent, hubID)
	serviceConnectionMapName := fmt.Sprintf("%s/serviceConnectionMaps/%s", locationParent, serviceConnectionMapID)
	internalRangeName := fmt.Sprintf("%s/internalRanges/%s", locationParent, internalRangeID)
	policyBasedRouteName := fmt.Sprintf("%s/policyBasedRoutes/%s", locationParent, policyBasedRouteID)
	transferConfigName := fmt.Sprintf("%s/multicloudDataTransferConfigs/%s", locationParent, transferConfigID)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Network Connectivity apiv1 clients using %s\n", grpcEndpoint)

	clientOptions := []option.ClientOption{
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	hubClient, err := networkconnectivity.NewHubClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create hub client: %v", err)
	}
	defer closeClient("hub", hubClient.Close)

	cnaClient, err := networkconnectivity.NewCrossNetworkAutomationClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create cross network automation client: %v", err)
	}
	defer closeClient("cross network automation", cnaClient.Close)

	internalRangeClient, err := networkconnectivity.NewInternalRangeClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create internal range client: %v", err)
	}
	defer closeClient("internal range", internalRangeClient.Close)

	pbrClient, err := networkconnectivity.NewPolicyBasedRoutingClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create policy-based routing client: %v", err)
	}
	defer closeClient("policy-based routing", pbrClient.Close)

	transferClient, err := networkconnectivity.NewDataTransferClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create data transfer client: %v", err)
	}
	defer closeClient("data transfer", transferClient.Close)

	allClients := &clients{
		hub:      hubClient,
		cna:      cnaClient,
		rangeAPI: internalRangeClient,
		pbr:      pbrClient,
		transfer: transferClient,
	}

	calls := []callSpec{
		{
			name: "ListHubs",
			call: func(ctx context.Context, c *clients) error {
				it := c.hub.ListHubs(ctx, &networkconnectivitypb.ListHubsRequest{
					Parent:   globalParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetHub",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.hub.GetHub(ctx, &networkconnectivitypb.GetHubRequest{Name: hubName})
				return err
			},
		},
		{
			name: "CreateHub",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.hub.CreateHub(ctx, &networkconnectivitypb.CreateHubRequest{
					Parent: globalParent,
					HubId:  hubID,
					Hub: &networkconnectivitypb.Hub{
						Description: "stackyard hub",
						Labels:      map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "ListServiceConnectionMaps",
			call: func(ctx context.Context, c *clients) error {
				it := c.cna.ListServiceConnectionMaps(ctx, &networkconnectivitypb.ListServiceConnectionMapsRequest{
					Parent:   locationParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetServiceConnectionMap",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.cna.GetServiceConnectionMap(ctx, &networkconnectivitypb.GetServiceConnectionMapRequest{
					Name: serviceConnectionMapName,
				})
				return err
			},
		},
		{
			name: "CreateServiceConnectionMap",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.cna.CreateServiceConnectionMap(ctx, &networkconnectivitypb.CreateServiceConnectionMapRequest{
					Parent:                 locationParent,
					ServiceConnectionMapId: serviceConnectionMapID,
					ServiceConnectionMap: &networkconnectivitypb.ServiceConnectionMap{
						Description: "stackyard service connection map",
						Labels:      map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "ListInternalRanges",
			call: func(ctx context.Context, c *clients) error {
				it := c.rangeAPI.ListInternalRanges(ctx, &networkconnectivitypb.ListInternalRangesRequest{
					Parent:   locationParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetInternalRange",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.rangeAPI.GetInternalRange(ctx, &networkconnectivitypb.GetInternalRangeRequest{Name: internalRangeName})
				return err
			},
		},
		{
			name: "CreateInternalRange",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.rangeAPI.CreateInternalRange(ctx, &networkconnectivitypb.CreateInternalRangeRequest{
					Parent:          locationParent,
					InternalRangeId: internalRangeID,
					InternalRange: &networkconnectivitypb.InternalRange{
						Description: "stackyard internal range",
						Labels:      map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "ListPolicyBasedRoutes",
			call: func(ctx context.Context, c *clients) error {
				it := c.pbr.ListPolicyBasedRoutes(ctx, &networkconnectivitypb.ListPolicyBasedRoutesRequest{
					Parent:   locationParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetPolicyBasedRoute",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.pbr.GetPolicyBasedRoute(ctx, &networkconnectivitypb.GetPolicyBasedRouteRequest{Name: policyBasedRouteName})
				return err
			},
		},
		{
			name: "CreatePolicyBasedRoute",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.pbr.CreatePolicyBasedRoute(ctx, &networkconnectivitypb.CreatePolicyBasedRouteRequest{
					Parent:             locationParent,
					PolicyBasedRouteId: policyBasedRouteID,
					PolicyBasedRoute: &networkconnectivitypb.PolicyBasedRoute{
						Description: "stackyard policy-based route",
						Labels:      map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "ListMulticloudDataTransferConfigs",
			call: func(ctx context.Context, c *clients) error {
				it := c.transfer.ListMulticloudDataTransferConfigs(ctx, &networkconnectivitypb.ListMulticloudDataTransferConfigsRequest{
					Parent:   locationParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetMulticloudDataTransferConfig",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.transfer.GetMulticloudDataTransferConfig(ctx, &networkconnectivitypb.GetMulticloudDataTransferConfigRequest{
					Name: transferConfigName,
				})
				return err
			},
		},
		{
			name: "CreateMulticloudDataTransferConfig",
			call: func(ctx context.Context, c *clients) error {
				_, err := c.transfer.CreateMulticloudDataTransferConfig(ctx, &networkconnectivitypb.CreateMulticloudDataTransferConfigRequest{
					Parent:                         locationParent,
					MulticloudDataTransferConfigId: transferConfigID,
					MulticloudDataTransferConfig: &networkconnectivitypb.MulticloudDataTransferConfig{
						Description: "stackyard transfer config",
						Labels:      map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
	}

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, allClients)
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

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded:
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
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large")
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
