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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appconfiguration/armappconfiguration"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_APPCONFIG_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_APPCONFIG_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-appcfg")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	configStore := getenv("STACKYARD_AZURE_APPCONFIG_STORE", "cfg-store")
	keyValue := getenv("STACKYARD_AZURE_APPCONFIG_KEYVALUE", "featureA")
	privateEndpointConnection := getenv("STACKYARD_AZURE_APPCONFIG_PRIVATE_ENDPOINT_CONNECTION", "pec-a")
	privateLinkGroup := getenv("STACKYARD_AZURE_APPCONFIG_PRIVATE_LINK_GROUP", "configStore")

	fmt.Printf("Stackyard Azure App Configuration Resource Manager typed SDK example using %s\n", endpoint)

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

	operationsClient, err := armappconfiguration.NewOperationsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create OperationsClient failed: %v", err)
	}
	configurationStoresClient, err := armappconfiguration.NewConfigurationStoresClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create ConfigurationStoresClient failed: %v", err)
	}
	keyValuesClient, err := armappconfiguration.NewKeyValuesClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create KeyValuesClient failed: %v", err)
	}
	privateEndpointConnectionsClient, err := armappconfiguration.NewPrivateEndpointConnectionsClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create PrivateEndpointConnectionsClient failed: %v", err)
	}
	privateLinkResourcesClient, err := armappconfiguration.NewPrivateLinkResourcesClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create PrivateLinkResourcesClient failed: %v", err)
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
			name: "CheckNameAvailability",
			run: func() error {
				_, err := operationsClient.CheckNameAvailability(ctx, armappconfiguration.CheckNameAvailabilityParameters{
					Name: to.Ptr(configStore),
					Type: to.Ptr(armappconfiguration.ConfigurationResourceTypeMicrosoftAppConfigurationConfigurationStores),
				}, nil)
				return err
			},
		},
		{
			name: "RegionalCheckNameAvailability",
			run: func() error {
				_, err := operationsClient.RegionalCheckNameAvailability(ctx, location, armappconfiguration.CheckNameAvailabilityParameters{
					Name: to.Ptr(configStore),
					Type: to.Ptr(armappconfiguration.ConfigurationResourceTypeMicrosoftAppConfigurationConfigurationStores),
				}, nil)
				return err
			},
		},
		{
			name: "CreateConfigurationStore",
			run: func() error {
				_, err := configurationStoresClient.BeginCreate(ctx, resourceGroup, configStore, armappconfiguration.ConfigurationStore{
					Location: to.Ptr(location),
					SKU: &armappconfiguration.SKU{
						Name: to.Ptr("Standard"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetConfigurationStore",
			run: func() error {
				_, err := configurationStoresClient.Get(ctx, resourceGroup, configStore, nil)
				return err
			},
		},
		{
			name: "UpdateConfigurationStore",
			run: func() error {
				_, err := configurationStoresClient.BeginUpdate(ctx, resourceGroup, configStore, armappconfiguration.ConfigurationStoreUpdateParameters{
					Tags: map[string]*string{
						"env": to.Ptr("dev"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "ListConfigurationStoresByResourceGroup",
			run: func() error {
				pager := configurationStoresClient.NewListByResourceGroupPager(resourceGroup, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListConfigurationStoresBySubscription",
			run: func() error {
				pager := configurationStoresClient.NewListPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListKeys",
			run: func() error {
				pager := configurationStoresClient.NewListKeysPager(resourceGroup, configStore, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "RegenerateKey",
			run: func() error {
				_, err := configurationStoresClient.RegenerateKey(ctx, resourceGroup, configStore, armappconfiguration.RegenerateKeyParameters{
					ID: to.Ptr("primary"),
				}, nil)
				return err
			},
		},
		{
			name: "ListDeletedConfigurationStores",
			run: func() error {
				pager := configurationStoresClient.NewListDeletedPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "GetDeletedConfigurationStore",
			run: func() error {
				_, err := configurationStoresClient.GetDeleted(ctx, location, configStore, nil)
				return err
			},
		},
		{
			name: "PurgeDeletedConfigurationStore",
			run: func() error {
				_, err := configurationStoresClient.BeginPurgeDeleted(ctx, location, configStore, nil)
				return err
			},
		},
		{
			name: "CreateOrUpdateKeyValue",
			run: func() error {
				_, err := keyValuesClient.CreateOrUpdate(ctx, resourceGroup, configStore, keyValue, &armappconfiguration.KeyValuesClientCreateOrUpdateOptions{
					KeyValueParameters: &armappconfiguration.KeyValue{
						Properties: &armappconfiguration.KeyValueProperties{
							Value: to.Ptr("true"),
						},
					},
				})
				return err
			},
		},
		{
			name: "GetKeyValue",
			run: func() error {
				_, err := keyValuesClient.Get(ctx, resourceGroup, configStore, keyValue, nil)
				return err
			},
		},
		{
			name: "CreateOrUpdatePrivateEndpointConnection",
			run: func() error {
				_, err := privateEndpointConnectionsClient.BeginCreateOrUpdate(ctx, resourceGroup, configStore, privateEndpointConnection, armappconfiguration.PrivateEndpointConnection{
					Properties: &armappconfiguration.PrivateEndpointConnectionProperties{
						PrivateLinkServiceConnectionState: &armappconfiguration.PrivateLinkServiceConnectionState{
							Status:      to.Ptr(armappconfiguration.ConnectionStatusApproved),
							Description: to.Ptr("approved by stackyard"),
						},
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetPrivateEndpointConnection",
			run: func() error {
				_, err := privateEndpointConnectionsClient.Get(ctx, resourceGroup, configStore, privateEndpointConnection, nil)
				return err
			},
		},
		{
			name: "ListPrivateEndpointConnections",
			run: func() error {
				pager := privateEndpointConnectionsClient.NewListByConfigurationStorePager(resourceGroup, configStore, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "GetPrivateLinkResource",
			run: func() error {
				_, err := privateLinkResourcesClient.Get(ctx, resourceGroup, configStore, privateLinkGroup, nil)
				return err
			},
		},
		{
			name: "ListPrivateLinkResources",
			run: func() error {
				pager := privateLinkResourcesClient.NewListByConfigurationStorePager(resourceGroup, configStore, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "DeletePrivateEndpointConnection",
			run: func() error {
				_, err := privateEndpointConnectionsClient.BeginDelete(ctx, resourceGroup, configStore, privateEndpointConnection, nil)
				return err
			},
		},
		{
			name: "DeleteKeyValue",
			run: func() error {
				_, err := keyValuesClient.BeginDelete(ctx, resourceGroup, configStore, keyValue, nil)
				return err
			},
		},
		{
			name: "DeleteConfigurationStore",
			run: func() error {
				_, err := configurationStoresClient.BeginDelete(ctx, resourceGroup, configStore, nil)
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
		fmt.Println("All app-configuration resource-manager routes are staged in this Stackyard build.")
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
