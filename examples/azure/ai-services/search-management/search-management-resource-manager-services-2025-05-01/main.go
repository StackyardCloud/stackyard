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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/search/armsearch"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_SEARCH_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SEARCH_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-search")
	searchServiceName := getenv("STACKYARD_AZURE_SEARCH_SERVICE_NAME", "my-search")

	fmt.Printf("Stackyard Azure Search Management Services typed SDK example using %s\n", endpoint)

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

	servicesClient, err := armsearch.NewServicesClient(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create ServicesClient failed: %v", err)
	}

	calls := []struct {
		name string
		run  func() error
	}{
		{
			name: "CheckNameAvailability",
			run: func() error {
				_, err := servicesClient.CheckNameAvailability(ctx, armsearch.CheckNameAvailabilityInput{
					Name: to.Ptr(searchServiceName),
					Type: to.Ptr("searchServices"),
				}, nil, nil)
				return err
			},
		},
		{
			name: "CreateOrUpdate",
			run: func() error {
				_, err := servicesClient.BeginCreateOrUpdate(ctx, resourceGroup, searchServiceName, armsearch.Service{
					Location: to.Ptr("eastus"),
					SKU: &armsearch.SKU{
						Name: to.Ptr(armsearch.SKUNameBasic),
					},
				}, nil, nil)
				return err
			},
		},
		{
			name: "Get",
			run: func() error {
				_, err := servicesClient.Get(ctx, resourceGroup, searchServiceName, nil, nil)
				return err
			},
		},
		{
			name: "Update",
			run: func() error {
				_, err := servicesClient.Update(ctx, resourceGroup, searchServiceName, armsearch.ServiceUpdate{
					Tags: map[string]*string{"env": to.Ptr("test")},
				}, nil, nil)
				return err
			},
		},
		{
			name: "ListByResourceGroup",
			run: func() error {
				pager := servicesClient.NewListByResourceGroupPager(resourceGroup, nil, nil)
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
				pager := servicesClient.NewListBySubscriptionPager(nil, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "Upgrade",
			run: func() error {
				_, err := servicesClient.BeginUpgrade(ctx, resourceGroup, searchServiceName, nil)
				return err
			},
		},
		{
			name: "Delete",
			run: func() error {
				_, err := servicesClient.Delete(ctx, resourceGroup, searchServiceName, nil, nil)
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
		fmt.Println("All search-management services routes are staged in this Stackyard build.")
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
