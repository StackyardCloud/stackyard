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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v5"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-web")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	planName := getenv("STACKYARD_AZURE_APPSERVICE_PLAN", "plan-a")
	siteName := getenv("STACKYARD_AZURE_APPSERVICE_SITE", "site-a")
	staticSiteName := getenv("STACKYARD_AZURE_APPSERVICE_STATIC_SITE", "static-a")
	certificateName := getenv("STACKYARD_AZURE_APPSERVICE_CERTIFICATE", "cert-a")

	fmt.Printf("Stackyard Azure App Service resource-manager-2025-05-01 example using %s\n", endpoint)

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

	factory, err := armappservice.NewClientFactory(subscriptionID, credential, armOptions)
	if err != nil {
		exitf("create ClientFactory failed: %v", err)
	}

	providerClient := factory.NewProviderClient()
	plansClient := factory.NewPlansClient()
	certificatesClient := factory.NewCertificatesClient()
	webAppsClient := factory.NewWebAppsClient()
	staticSitesClient := factory.NewStaticSitesClient()

	calls := []struct {
		name string
		run  func() error
	}{
		{
			name: "ListOperations",
			run: func() error {
				pager := providerClient.NewListOperationsPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListAppServicePlansBySubscription",
			run: func() error {
				pager := plansClient.NewListPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "ListAppServicePlansByResourceGroup",
			run: func() error {
				pager := plansClient.NewListByResourceGroupPager(resourceGroup, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "CreateAppServicePlan",
			run: func() error {
				_, err := plansClient.BeginCreateOrUpdate(ctx, resourceGroup, planName, armappservice.AppServicePlan{
					Location: to.Ptr(location),
					SKU: &armappservice.SKUDescription{
						Name:     to.Ptr("F1"),
						Tier:     to.Ptr("Free"),
						Size:     to.Ptr("F1"),
						Capacity: to.Ptr[int32](1),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetAppServicePlan",
			run: func() error {
				_, err := plansClient.Get(ctx, resourceGroup, planName, nil)
				return err
			},
		},
		{
			name: "ListCertificatesBySubscription",
			run: func() error {
				pager := certificatesClient.NewListPager(nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "GetCertificate",
			run: func() error {
				_, err := certificatesClient.Get(ctx, resourceGroup, certificateName, nil)
				return err
			},
		},
		{
			name: "CreateWebApp",
			run: func() error {
				_, err := webAppsClient.BeginCreateOrUpdate(ctx, resourceGroup, siteName, armappservice.Site{
					Location: to.Ptr(location),
					Properties: &armappservice.SiteProperties{
						ServerFarmID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Web/serverfarms/" + planName),
						HTTPSOnly:    to.Ptr(true),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetWebApp",
			run: func() error {
				_, err := webAppsClient.Get(ctx, resourceGroup, siteName, nil)
				return err
			},
		},
		{
			name: "ListWebAppsByResourceGroup",
			run: func() error {
				pager := webAppsClient.NewListByResourceGroupPager(resourceGroup, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
			},
		},
		{
			name: "UpdateWebAppConfiguration",
			run: func() error {
				_, err := webAppsClient.UpdateConfiguration(ctx, resourceGroup, siteName, armappservice.SiteConfig{
					AlwaysOn:      to.Ptr(true),
					HTTP20Enabled: to.Ptr(true),
				}, nil)
				return err
			},
		},
		{
			name: "GetWebAppConfiguration",
			run: func() error {
				_, err := webAppsClient.GetConfiguration(ctx, resourceGroup, siteName, nil)
				return err
			},
		},
		{
			name: "CreateStaticSite",
			run: func() error {
				_, err := staticSitesClient.BeginCreateOrUpdateStaticSite(ctx, resourceGroup, staticSiteName, armappservice.StaticSiteARMResource{
					Location: to.Ptr(location),
					Properties: &armappservice.StaticSite{
						RepositoryURL: to.Ptr("https://github.com/example/static-a"),
						Branch:        to.Ptr("main"),
					},
				}, nil)
				return err
			},
		},
		{
			name: "GetStaticSite",
			run: func() error {
				_, err := staticSitesClient.GetStaticSite(ctx, resourceGroup, staticSiteName, nil)
				return err
			},
		},
		{
			name: "ListStaticSitesByResourceGroup",
			run: func() error {
				pager := staticSitesClient.NewGetStaticSitesByResourceGroupPager(resourceGroup, nil)
				if pager.More() {
					_, err := pager.NextPage(ctx)
					return err
				}
				return nil
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
		fmt.Println("All appservice resource-manager routes are staged in this Stackyard build.")
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
