package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/avs/armavs/v2"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_AVS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AVS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-avs")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	privateCloudName := getenv("STACKYARD_AZURE_AVS_PRIVATE_CLOUD", "pc-a")
	clusterName := getenv("STACKYARD_AZURE_AVS_CLUSTER", "cluster-a")

	fmt.Printf("Stackyard Azure AVS typed SDK example using %s\n", endpoint)

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

	factory, err := armavs.NewClientFactory(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create AVS client factory failed: %v", err)
	}

	operationsClient := factory.NewOperationsClient()
	locationsClient := factory.NewLocationsClient()
	privateCloudsClient := factory.NewPrivateCloudsClient()
	authorizationsClient := factory.NewAuthorizationsClient()
	globalReachConnectionsClient := factory.NewGlobalReachConnectionsClient()
	licensesClient := factory.NewLicensesClient()
	addonsClient := factory.NewAddonsClient()
	maintenancesClient := factory.NewMaintenancesClient()
	scriptExecutionsClient := factory.NewScriptExecutionsClient()
	iscsiPathsClient := factory.NewIscsiPathsClient()
	clustersClient := factory.NewClustersClient()
	hostsClient := factory.NewHostsClient()
	datastoresClient := factory.NewDatastoresClient()
	placementPoliciesClient := factory.NewPlacementPoliciesClient()
	virtualMachinesClient := factory.NewVirtualMachinesClient()
	workloadNetworksClient := factory.NewWorkloadNetworksClient()

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
			name: "CheckQuotaAvailability",
			run: func() error {
				_, err := locationsClient.CheckQuotaAvailability(ctx, location, nil)
				return err
			},
		},
		{
			name: "CheckTrialAvailability",
			run: func() error {
				_, err := locationsClient.CheckTrialAvailability(ctx, location, nil)
				return err
			},
		},
		{
			name: "CreatePrivateCloud",
			run: func() error {
				_, err := privateCloudsClient.BeginCreateOrUpdate(ctx, resourceGroup, privateCloudName, armavs.PrivateCloud{
					Location: to.Ptr(location),
					Tags: map[string]*string{
						"env": to.Ptr("dev"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetPrivateCloud",
			run: func() error {
				_, err := privateCloudsClient.Get(ctx, resourceGroup, privateCloudName, nil)
				return err
			},
		},
		{
			name: "ListPrivateCloudsByResourceGroup",
			run: func() error {
				pager := privateCloudsClient.NewListPager(resourceGroup, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListPrivateCloudsBySubscription",
			run: func() error {
				pager := privateCloudsClient.NewListInSubscriptionPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListAdminCredentials",
			run: func() error {
				_, err := privateCloudsClient.ListAdminCredentials(ctx, resourceGroup, privateCloudName, nil)
				return err
			},
		},
		{
			name: "GetVCFLicense",
			run: func() error {
				_, err := privateCloudsClient.GetVcfLicense(ctx, resourceGroup, privateCloudName, nil)
				return err
			},
		},
		{
			name: "UpdatePrivateCloud",
			run: func() error {
				_, err := privateCloudsClient.BeginUpdate(ctx, resourceGroup, privateCloudName, armavs.PrivateCloudUpdate{
					Tags: map[string]*string{
						"env":   to.Ptr("stage"),
						"owner": to.Ptr("stackyard"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "RotateNsxtPassword",
			run: func() error {
				_, err := privateCloudsClient.BeginRotateNsxtPassword(ctx, resourceGroup, privateCloudName, nil)
				return err
			},
		},
		{
			name: "RotateVcenterPassword",
			run: func() error {
				_, err := privateCloudsClient.BeginRotateVcenterPassword(ctx, resourceGroup, privateCloudName, nil)
				return err
			},
		},
		{
			name: "CreateAuthorization",
			run: func() error {
				_, err := authorizationsClient.BeginCreateOrUpdate(ctx, resourceGroup, privateCloudName, "auth-a", armavs.ExpressRouteAuthorization{
					Properties: &armavs.ExpressRouteAuthorizationProperties{
						ExpressRouteID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/expressRouteCircuits/er-a"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "ListAuthorizations",
			run: func() error {
				pager := authorizationsClient.NewListPager(resourceGroup, privateCloudName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "CreateGlobalReachConnection",
			run: func() error {
				_, err := globalReachConnectionsClient.BeginCreateOrUpdate(ctx, resourceGroup, privateCloudName, "grc-a", armavs.GlobalReachConnection{
					Properties: &armavs.GlobalReachConnectionProperties{
						PeerExpressRouteCircuit: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/expressRouteCircuits/peer-a"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "ListGlobalReachConnections",
			run: func() error {
				pager := globalReachConnectionsClient.NewListPager(resourceGroup, privateCloudName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "CreateLicense",
			run: func() error {
				_, err := licensesClient.BeginCreateOrUpdate(ctx, resourceGroup, privateCloudName, armavs.LicenseNameVmwareFirewall, armavs.License{
					Properties: &armavs.LicenseProperties{
						Kind: to.Ptr(armavs.LicenseKindVmwareFirewall),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetLicenseProperties",
			run: func() error {
				_, err := licensesClient.GetProperties(ctx, resourceGroup, privateCloudName, armavs.LicenseNameVmwareFirewall, nil)
				return err
			},
		},
		{
			name: "ListAddons",
			run: func() error {
				pager := addonsClient.NewListPager(resourceGroup, privateCloudName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListMaintenances",
			run: func() error {
				pager := maintenancesClient.NewListPager(resourceGroup, privateCloudName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "InitiateMaintenanceChecks",
			run: func() error {
				_, err := maintenancesClient.InitiateChecks(ctx, resourceGroup, privateCloudName, "default", nil)
				return err
			},
		},
		{
			name: "RescheduleMaintenance",
			run: func() error {
				_, err := maintenancesClient.Reschedule(ctx, resourceGroup, privateCloudName, "default", armavs.MaintenanceReschedule{
					RescheduleTime: to.Ptr(time.Now().UTC().Add(24 * time.Hour)),
				}, nil)
				return err
			},
		},
		{
			name: "CreateScriptExecution",
			run: func() error {
				_, err := scriptExecutionsClient.BeginCreateOrUpdate(ctx, resourceGroup, privateCloudName, "exec-a", armavs.ScriptExecution{
					Properties: &armavs.ScriptExecutionProperties{
						Timeout:        to.Ptr("PT30M"),
						ScriptCmdletID: to.Ptr("Get-Cluster"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetScriptExecutionLogs",
			run: func() error {
				_, err := scriptExecutionsClient.GetExecutionLogs(ctx, resourceGroup, privateCloudName, "exec-a", nil)
				return err
			},
		},
		{
			name: "CreateIscsiPath",
			run: func() error {
				_, err := iscsiPathsClient.BeginCreateOrUpdate(ctx, resourceGroup, privateCloudName, armavs.IscsiPath{}, nil)
				return err
			},
		},
		{
			name: "ListIscsiPaths",
			run: func() error {
				pager := iscsiPathsClient.NewListByPrivateCloudPager(resourceGroup, privateCloudName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "CreateCluster",
			run: func() error {
				_, err := clustersClient.BeginCreateOrUpdate(ctx, resourceGroup, privateCloudName, clusterName, armavs.Cluster{
					SKU: &armavs.SKU{
						Name: to.Ptr("AV36"),
					},
					Properties: &armavs.ClusterProperties{
						ClusterSize: to.Ptr(int32(3)),
					},
				}, nil)
				return err
			},
		},
		{
			name: "ListClusters",
			run: func() error {
				pager := clustersClient.NewListPager(resourceGroup, privateCloudName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListClusterZones",
			run: func() error {
				_, err := clustersClient.ListZones(ctx, resourceGroup, privateCloudName, clusterName, nil)
				return err
			},
		},
		{
			name: "ListHosts",
			run: func() error {
				pager := hostsClient.NewListPager(resourceGroup, privateCloudName, clusterName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "CreateDatastore",
			run: func() error {
				_, err := datastoresClient.BeginCreateOrUpdate(ctx, resourceGroup, privateCloudName, clusterName, "ds-a", armavs.Datastore{
					Properties: &armavs.DatastoreProperties{},
				}, nil)
				return err
			},
		},
		{
			name: "ListDatastores",
			run: func() error {
				pager := datastoresClient.NewListPager(resourceGroup, privateCloudName, clusterName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "CreatePlacementPolicy",
			run: func() error {
				_, err := placementPoliciesClient.BeginCreateOrUpdate(ctx, resourceGroup, privateCloudName, clusterName, "policy-a", armavs.PlacementPolicy{
					Properties: &armavs.PlacementPolicyProperties{
						Type: to.Ptr(armavs.PlacementPolicyTypeVMVM),
					},
				}, nil)
				return err
			},
		},
		{
			name: "UpdatePlacementPolicy",
			run: func() error {
				_, err := placementPoliciesClient.BeginUpdate(ctx, resourceGroup, privateCloudName, clusterName, "policy-a", armavs.PlacementPolicyUpdate{
					Properties: &armavs.PlacementPolicyUpdateProperties{
						State: to.Ptr(armavs.PlacementPolicyStateEnabled),
					},
				}, nil)
				return err
			},
		},
		{
			name: "ListPlacementPolicies",
			run: func() error {
				pager := placementPoliciesClient.NewListPager(resourceGroup, privateCloudName, clusterName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListClusterVirtualMachines",
			run: func() error {
				pager := virtualMachinesClient.NewListPager(resourceGroup, privateCloudName, clusterName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "RestrictClusterVirtualMachineMovement",
			run: func() error {
				_, err := virtualMachinesClient.BeginRestrictMovement(ctx, resourceGroup, privateCloudName, clusterName, "vm-1", armavs.VirtualMachineRestrictMovement{
					RestrictMovement: to.Ptr(armavs.VirtualMachineRestrictMovementStateEnabled),
				}, nil)
				return err
			},
		},
		{
			name: "GetWorkloadNetwork",
			run: func() error {
				_, err := workloadNetworksClient.Get(ctx, resourceGroup, privateCloudName, nil)
				return err
			},
		},
		{
			name: "CreateDNSZone",
			run: func() error {
				_, err := workloadNetworksClient.BeginUpdateDNSZone(ctx, resourceGroup, privateCloudName, "corp.local", armavs.WorkloadNetworkDNSZone{}, nil)
				return err
			},
		},
		{
			name: "ListDNSZones",
			run: func() error {
				pager := workloadNetworksClient.NewListDNSZonesPager(resourceGroup, privateCloudName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "CreateVMGroup",
			run: func() error {
				_, err := workloadNetworksClient.BeginUpdateVMGroup(ctx, resourceGroup, privateCloudName, "apps", armavs.WorkloadNetworkVMGroup{}, nil)
				return err
			},
		},
		{
			name: "ListVMGroups",
			run: func() error {
				pager := workloadNetworksClient.NewListVMGroupsPager(resourceGroup, privateCloudName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "GetWorkloadVirtualMachine",
			run: func() error {
				_, err := workloadNetworksClient.GetVirtualMachine(ctx, resourceGroup, privateCloudName, "vm-1", nil)
				return err
			},
		},
		{
			name: "DeleteVMGroup",
			run: func() error {
				_, err := workloadNetworksClient.BeginDeleteVMGroup(ctx, resourceGroup, "apps", privateCloudName, nil)
				return err
			},
		},
		{
			name: "DeleteDatastore",
			run: func() error {
				_, err := datastoresClient.BeginDelete(ctx, resourceGroup, privateCloudName, clusterName, "ds-a", nil)
				return err
			},
		},
		{
			name: "DeletePlacementPolicy",
			run: func() error {
				_, err := placementPoliciesClient.BeginDelete(ctx, resourceGroup, privateCloudName, clusterName, "policy-a", nil)
				return err
			},
		},
		{
			name: "DeleteCluster",
			run: func() error {
				_, err := clustersClient.BeginDelete(ctx, resourceGroup, privateCloudName, clusterName, nil)
				return err
			},
		},
		{
			name: "DeletePrivateCloud",
			run: func() error {
				_, err := privateCloudsClient.BeginDelete(ctx, resourceGroup, privateCloudName, nil)
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
		fmt.Println("All avs resource-manager routes are staged in this Stackyard build.")
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
