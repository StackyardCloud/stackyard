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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_APPLICATION_GATEWAY_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_APPLICATION_GATEWAY_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-net")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	gatewayName := getenv("STACKYARD_AZURE_APPLICATION_GATEWAY_NAME", "appgw-a")
	sslPolicyName := getenv("STACKYARD_AZURE_APPLICATION_GATEWAY_SSL_POLICY", "AppGwSslPolicy20220101")

	fmt.Printf("Stackyard Azure Application Gateway typed SDK example using %s\n", endpoint)

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

	client, err := armnetwork.NewApplicationGatewaysClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create ApplicationGatewaysClient failed: %v", err)
	}

	gatewayID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/applicationGateways/%s",
		subscriptionID,
		resourceGroup,
		gatewayName,
	)

	calls := []struct {
		name string
		run  func() error
	}{
		{
			name: "ListAvailableRequestHeaders",
			run: func() error {
				_, err := client.ListAvailableRequestHeaders(ctx, nil)
				return err
			},
		},
		{
			name: "ListAvailableResponseHeaders",
			run: func() error {
				_, err := client.ListAvailableResponseHeaders(ctx, nil)
				return err
			},
		},
		{
			name: "ListAvailableServerVariables",
			run: func() error {
				_, err := client.ListAvailableServerVariables(ctx, nil)
				return err
			},
		},
		{
			name: "ListAvailableSSLOptions",
			run: func() error {
				_, err := client.ListAvailableSSLOptions(ctx, nil)
				return err
			},
		},
		{
			name: "ListAvailableSSLPredefinedPolicies",
			run: func() error {
				pager := client.NewListAvailableSSLPredefinedPoliciesPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "GetSSLPredefinedPolicy",
			run: func() error {
				_, err := client.GetSSLPredefinedPolicy(ctx, sslPolicyName, nil)
				return err
			},
		},
		{
			name: "ListAvailableWafRuleSets",
			run: func() error {
				_, err := client.ListAvailableWafRuleSets(ctx, nil)
				return err
			},
		},
		{
			name: "CreateOrUpdateApplicationGateway",
			run: func() error {
				_, err := client.BeginCreateOrUpdate(ctx, resourceGroup, gatewayName, armnetwork.ApplicationGateway{
					Location: to.Ptr(location),
					Tags: map[string]*string{
						"env": to.Ptr("dev"),
					},
					Properties: &armnetwork.ApplicationGatewayPropertiesFormat{},
				}, nil)
				return err
			},
		},
		{
			name: "GetApplicationGateway",
			run: func() error {
				_, err := client.Get(ctx, resourceGroup, gatewayName, nil)
				return err
			},
		},
		{
			name: "UpdateTags",
			run: func() error {
				_, err := client.UpdateTags(ctx, resourceGroup, gatewayName, armnetwork.TagsObject{
					Tags: map[string]*string{
						"env":   to.Ptr("stage"),
						"owner": to.Ptr("stackyard"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "ListByResourceGroup",
			run: func() error {
				pager := client.NewListPager(resourceGroup, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListBySubscription",
			run: func() error {
				pager := client.NewListAllPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "BackendHealth",
			run: func() error {
				_, err := client.BeginBackendHealth(ctx, resourceGroup, gatewayName, nil)
				return err
			},
		},
		{
			name: "BackendHealthOnDemand",
			run: func() error {
				_, err := client.BeginBackendHealthOnDemand(ctx, resourceGroup, gatewayName, armnetwork.ApplicationGatewayOnDemandProbe{
					BackendAddressPool: &armnetwork.SubResource{ID: to.Ptr(gatewayID + "/backendAddressPools/pool-default")},
					BackendHTTPSettings: &armnetwork.SubResource{
						ID: to.Ptr(gatewayID + "/backendHttpSettingsCollection/http-default"),
					},
					Path:     to.Ptr("/"),
					Protocol: to.Ptr(armnetwork.ApplicationGatewayProtocolHTTP),
				}, nil)
				return err
			},
		},
		{
			name: "StopApplicationGateway",
			run: func() error {
				_, err := client.BeginStop(ctx, resourceGroup, gatewayName, nil)
				return err
			},
		},
		{
			name: "StartApplicationGateway",
			run: func() error {
				_, err := client.BeginStart(ctx, resourceGroup, gatewayName, nil)
				return err
			},
		},
		{
			name: "DeleteApplicationGateway",
			run: func() error {
				_, err := client.BeginDelete(ctx, resourceGroup, gatewayName, nil)
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
		fmt.Println("All application-gateway resource-manager routes are staged in this Stackyard build.")
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
