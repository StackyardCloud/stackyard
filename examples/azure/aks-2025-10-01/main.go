package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-aks")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	clusterName := getenv("STACKYARD_AZURE_AKS_CLUSTER", "cluster-a")
	agentPoolName := getenv("STACKYARD_AZURE_AKS_AGENT_POOL", "system")
	snapshotName := getenv("STACKYARD_AZURE_AKS_SNAPSHOT", "snapshot-a")

	fmt.Printf("Stackyard Azure AKS typed SDK example using %s\n", endpoint)

	credential := azsdkshim.StaticTokenCredential{}
	armOptions := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			InsecureAllowCredentialWithHTTP: true,
			Cloud: cloud.Configuration{
				ActiveDirectoryAuthorityHost: endpoint,
				Services: map[cloud.ServiceName]cloud.ServiceConfiguration{
					cloud.ResourceManager: {
						Endpoint: endpoint,
						Audience: endpoint,
					},
				},
			},
			Transport: azsdkshim.NewTransport("/azure", account, subscriptionKey),
		},
	}

	operationsClient, err := armcontainerservice.NewOperationsClient(credential, armOptions)
	if err != nil {
		exitf("create OperationsClient failed: %v", err)
	}
	clustersClient, err := armcontainerservice.NewManagedClustersClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create ManagedClustersClient failed: %v", err)
	}
	agentPoolsClient, err := armcontainerservice.NewAgentPoolsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create AgentPoolsClient failed: %v", err)
	}
	maintenanceClient, err := armcontainerservice.NewMaintenanceConfigurationsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create MaintenanceConfigurationsClient failed: %v", err)
	}
	privateEndpointClient, err := armcontainerservice.NewPrivateEndpointConnectionsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create PrivateEndpointConnectionsClient failed: %v", err)
	}
	privateLinkClient, err := armcontainerservice.NewPrivateLinkResourcesClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create PrivateLinkResourcesClient failed: %v", err)
	}
	resolvePrivateLinkServiceIDClient, err := armcontainerservice.NewResolvePrivateLinkServiceIDClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create ResolvePrivateLinkServiceIDClient failed: %v", err)
	}
	snapshotsClient, err := armcontainerservice.NewSnapshotsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create SnapshotsClient failed: %v", err)
	}

	calls := []struct {
		name string
		run  func() error
	}{
		{
			name: "ListOperations",
			run: func() error {
				pager := operationsClient.NewListPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListManagedClustersBySubscription",
			run: func() error {
				pager := clustersClient.NewListPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListManagedClustersByResourceGroup",
			run: func() error {
				pager := clustersClient.NewListByResourceGroupPager(resourceGroup, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "GetManagedCluster",
			run: func() error {
				_, err := clustersClient.Get(ctx, resourceGroup, clusterName, nil)
				return err
			},
		},
		{
			name: "ListAgentPools",
			run: func() error {
				pager := agentPoolsClient.NewListPager(resourceGroup, clusterName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "GetAgentPool",
			run: func() error {
				_, err := agentPoolsClient.Get(ctx, resourceGroup, clusterName, agentPoolName, nil)
				return err
			},
		},
		{
			name: "ListMaintenanceConfigurations",
			run: func() error {
				pager := maintenanceClient.NewListByManagedClusterPager(resourceGroup, clusterName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "GetMaintenanceConfiguration",
			run: func() error {
				_, err := maintenanceClient.Get(ctx, resourceGroup, clusterName, "default", nil)
				return err
			},
		},
		{
			name: "ListPrivateEndpointConnections",
			run: func() error {
				_, err := privateEndpointClient.List(ctx, resourceGroup, clusterName, nil)
				return err
			},
		},
		{
			name: "ListPrivateLinkResources",
			run: func() error {
				_, err := privateLinkClient.List(ctx, resourceGroup, clusterName, nil)
				return err
			},
		},
		{
			name: "ResolvePrivateLinkServiceID",
			run: func() error {
				_, err := resolvePrivateLinkServiceIDClient.POST(ctx, resourceGroup, clusterName, armcontainerservice.PrivateLinkResource{
					Name: to.Ptr(clusterName),
				}, nil)
				return err
			},
		},
		{
			name: "ListSnapshotsBySubscription",
			run: func() error {
				pager := snapshotsClient.NewListPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListSnapshotsByResourceGroup",
			run: func() error {
				pager := snapshotsClient.NewListByResourceGroupPager(resourceGroup, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "CreateOrUpdateSnapshot",
			run: func() error {
				_, err := snapshotsClient.CreateOrUpdate(ctx, resourceGroup, snapshotName, armcontainerservice.Snapshot{
					Location: to.Ptr(location),
				}, nil)
				return err
			},
		},
		{
			name: "GetSnapshot",
			run: func() error {
				_, err := snapshotsClient.Get(ctx, resourceGroup, snapshotName, nil)
				return err
			},
		},
		{
			name: "UpdateSnapshotTags",
			run: func() error {
				_, err := snapshotsClient.UpdateTags(ctx, resourceGroup, snapshotName, armcontainerservice.TagsObject{
					Tags: map[string]*string{"env": to.Ptr("dev")},
				}, nil)
				return err
			},
		},
		{
			name: "DeleteSnapshot",
			run: func() error {
				_, err := snapshotsClient.Delete(ctx, resourceGroup, snapshotName, nil)
				return err
			},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		err := call.run()
		switch {
		case err == nil:
			fmt.Printf("%s: ok\n", call.name)
		case isNotImplemented(err):
			notImplementedCount++
			fmt.Printf("Route is recognized but not implemented yet: %s\n", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	if notImplementedCount == len(calls) {
		fmt.Println("All AKS routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
}

func isNotImplemented(err error) bool {
	if err == nil {
		return false
	}
	var responseErr *azcore.ResponseError
	if errors.As(err, &responseErr) {
		if responseErr.StatusCode == http.StatusNotImplemented {
			return true
		}
		if strings.EqualFold(strings.TrimSpace(responseErr.ErrorCode), "NotImplemented") {
			return true
		}
	}
	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
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
