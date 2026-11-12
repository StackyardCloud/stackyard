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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/servicenetworking/armservicenetworking"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_APPLICATION_GATEWAY_CONTAINERS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_APPLICATION_GATEWAY_CONTAINERS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-servicenetworking")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	trafficControllerName := getenv("STACKYARD_AZURE_AGFC_TRAFFIC_CONTROLLER", "tc-a")
	frontendName := getenv("STACKYARD_AZURE_AGFC_FRONTEND", "fe-a")
	associationName := getenv("STACKYARD_AZURE_AGFC_ASSOCIATION", "as-a")
	securityPolicyName := getenv("STACKYARD_AZURE_AGFC_SECURITY_POLICY", "sp-a")

	fmt.Printf("Stackyard Azure Application Gateway for Containers typed SDK example using %s\n", endpoint)

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

	clientFactory, err := armservicenetworking.NewClientFactory(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create ClientFactory failed: %v", err)
	}

	operationsClient := clientFactory.NewOperationsClient()
	trafficControllersClient := clientFactory.NewTrafficControllerInterfaceClient()
	frontendsClient := clientFactory.NewFrontendsInterfaceClient()
	associationsClient := clientFactory.NewAssociationsInterfaceClient()
	securityPoliciesClient := clientFactory.NewSecurityPoliciesInterfaceClient()

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
			name: "CreateTrafficController",
			run: func() error {
				_, err := trafficControllersClient.BeginCreateOrUpdate(ctx, resourceGroup, trafficControllerName, armservicenetworking.TrafficController{
					Location: to.Ptr(location),
					Tags: map[string]*string{
						"env": to.Ptr("dev"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetTrafficController",
			run: func() error {
				_, err := trafficControllersClient.Get(ctx, resourceGroup, trafficControllerName, nil)
				return err
			},
		},
		{
			name: "ListTrafficControllersByResourceGroup",
			run: func() error {
				pager := trafficControllersClient.NewListByResourceGroupPager(resourceGroup, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListTrafficControllersBySubscription",
			run: func() error {
				pager := trafficControllersClient.NewListBySubscriptionPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "UpdateTrafficController",
			run: func() error {
				_, err := trafficControllersClient.Update(ctx, resourceGroup, trafficControllerName, armservicenetworking.TrafficControllerUpdate{
					Tags: map[string]*string{
						"env":   to.Ptr("stage"),
						"owner": to.Ptr("stackyard"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateFrontend",
			run: func() error {
				_, err := frontendsClient.BeginCreateOrUpdate(ctx, resourceGroup, trafficControllerName, frontendName, armservicenetworking.Frontend{
					Location: to.Ptr(location),
					Tags: map[string]*string{
						"tier": to.Ptr("public"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetFrontend",
			run: func() error {
				_, err := frontendsClient.Get(ctx, resourceGroup, trafficControllerName, frontendName, nil)
				return err
			},
		},
		{
			name: "ListFrontends",
			run: func() error {
				pager := frontendsClient.NewListByTrafficControllerPager(resourceGroup, trafficControllerName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "UpdateFrontend",
			run: func() error {
				_, err := frontendsClient.Update(ctx, resourceGroup, trafficControllerName, frontendName, armservicenetworking.FrontendUpdate{
					Tags: map[string]*string{
						"tier": to.Ptr("edge"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateAssociation",
			run: func() error {
				_, err := associationsClient.BeginCreateOrUpdate(ctx, resourceGroup, trafficControllerName, associationName, armservicenetworking.Association{
					Location: to.Ptr(location),
					Properties: &armservicenetworking.AssociationProperties{
						AssociationType: to.Ptr(armservicenetworking.AssociationTypeSubnets),
						Subnet: &armservicenetworking.AssociationSubnet{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/subnet-a"),
						},
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetAssociation",
			run: func() error {
				_, err := associationsClient.Get(ctx, resourceGroup, trafficControllerName, associationName, nil)
				return err
			},
		},
		{
			name: "ListAssociations",
			run: func() error {
				pager := associationsClient.NewListByTrafficControllerPager(resourceGroup, trafficControllerName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "UpdateAssociation",
			run: func() error {
				_, err := associationsClient.Update(ctx, resourceGroup, trafficControllerName, associationName, armservicenetworking.AssociationUpdate{
					Properties: &armservicenetworking.AssociationUpdateProperties{
						AssociationType: to.Ptr(armservicenetworking.AssociationTypeSubnets),
						Subnet: &armservicenetworking.AssociationSubnetUpdate{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/subnet-b"),
						},
					},
				}, nil)
				return err
			},
		},
		{
			name: "CreateSecurityPolicy",
			run: func() error {
				_, err := securityPoliciesClient.BeginCreateOrUpdate(ctx, resourceGroup, trafficControllerName, securityPolicyName, armservicenetworking.SecurityPolicy{
					Location: to.Ptr(location),
					Properties: &armservicenetworking.SecurityPolicyProperties{
						WafPolicy: &armservicenetworking.WafPolicy{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies/waf-a"),
						},
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetSecurityPolicy",
			run: func() error {
				_, err := securityPoliciesClient.Get(ctx, resourceGroup, trafficControllerName, securityPolicyName, nil)
				return err
			},
		},
		{
			name: "ListSecurityPolicies",
			run: func() error {
				pager := securityPoliciesClient.NewListByTrafficControllerPager(resourceGroup, trafficControllerName, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "UpdateSecurityPolicy",
			run: func() error {
				_, err := securityPoliciesClient.Update(ctx, resourceGroup, trafficControllerName, securityPolicyName, armservicenetworking.SecurityPolicyUpdate{
					Properties: &armservicenetworking.SecurityPolicyUpdateProperties{
						WafPolicy: &armservicenetworking.WafPolicy{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies/waf-b"),
						},
					},
				}, nil)
				return err
			},
		},
		{
			name: "DeleteFrontend",
			run: func() error {
				_, err := frontendsClient.BeginDelete(ctx, resourceGroup, trafficControllerName, frontendName, nil)
				return err
			},
		},
		{
			name: "DeleteAssociation",
			run: func() error {
				_, err := associationsClient.BeginDelete(ctx, resourceGroup, trafficControllerName, associationName, nil)
				return err
			},
		},
		{
			name: "DeleteSecurityPolicy",
			run: func() error {
				_, err := securityPoliciesClient.BeginDelete(ctx, resourceGroup, trafficControllerName, securityPolicyName, nil)
				return err
			},
		},
		{
			name: "DeleteTrafficController",
			run: func() error {
				_, err := trafficControllersClient.BeginDelete(ctx, resourceGroup, trafficControllerName, nil)
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
		fmt.Println("All application-gateway-containers resource-manager routes are staged in this Stackyard build.")
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
