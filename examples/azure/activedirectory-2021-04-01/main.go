package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/azureadexternalidentities/mgmt/2021-04-01/azureadexternalidentities"
	"github.com/Azure/go-autorest/autorest"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_ACTIVEDIRECTORY_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_ACTIVEDIRECTORY_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-b2c")
	location := getenv("STACKYARD_AZURE_REGION", "United States")
	b2cTenantName := getenv("STACKYARD_AZURE_ACTIVEDIRECTORY_B2C_TENANT", "tenant-a.onmicrosoft.com")
	guestUsageName := getenv("STACKYARD_AZURE_ACTIVEDIRECTORY_GUEST_USAGE", "guest-usage-a")
	tenantID := getenv("STACKYARD_AZURE_ACTIVEDIRECTORY_TENANT_ID", "11111111-1111-1111-1111-111111111111")

	fmt.Printf("Stackyard Azure Active Directory typed SDK example using %s\n", endpoint)

	transport := azsdkshim.NewTransport("/azure", account, subscriptionKey)

	operationsClient := azureadexternalidentities.NewOperationsClientWithBaseURI(endpoint, subscriptionID)
	b2cTenantsClient := azureadexternalidentities.NewB2CTenantsClientWithBaseURI(endpoint, subscriptionID)
	guestUsagesClient := azureadexternalidentities.NewGuestUsagesClientWithBaseURI(endpoint, subscriptionID)

	operationsClient.Sender = transport
	b2cTenantsClient.Sender = transport
	guestUsagesClient.Sender = transport

	calls := []struct {
		name string
		run  func() error
	}{
		{
			name: "ListOperations",
			run: func() error {
				_, err := operationsClient.List(ctx)
				return err
			},
		},
		{
			name: "CheckNameAvailability",
			run: func() error {
				_, err := b2cTenantsClient.CheckNameAvailability(ctx, &azureadexternalidentities.CheckNameAvailabilityRequestBody{
					Name:        strPtr(b2cTenantName),
					CountryCode: strPtr("US"),
				})
				return err
			},
		},
		{
			name: "CreateB2CTenant",
			run: func() error {
				_, err := b2cTenantsClient.Create(ctx, resourceGroup, b2cTenantName, &azureadexternalidentities.CreateTenantRequestBody{
					Location: strPtr(location),
					Sku: &azureadexternalidentities.B2CResourceSKU{
						Name: azureadexternalidentities.B2CResourceSKUNameStandard,
						Tier: azureadexternalidentities.B2CResourceSKUTierA0,
					},
					CreateTenantRequestBodyProperties: &azureadexternalidentities.CreateTenantRequestBodyProperties{
						CreateTenantProperties: &azureadexternalidentities.CreateTenantProperties{
							CountryCode: strPtr("US"),
							DisplayName: strPtr("Stackyard Tenant"),
						},
					},
					Tags: map[string]*string{
						"env": strPtr("dev"),
					},
				})
				return err
			},
		},
		{
			name: "GetB2CTenant",
			run: func() error {
				_, err := b2cTenantsClient.Get(ctx, resourceGroup, b2cTenantName)
				return err
			},
		},
		{
			name: "ListB2CTenantsByResourceGroup",
			run: func() error {
				_, err := b2cTenantsClient.ListByResourceGroup(ctx, resourceGroup)
				return err
			},
		},
		{
			name: "ListB2CTenantsBySubscription",
			run: func() error {
				_, err := b2cTenantsClient.ListBySubscription(ctx)
				return err
			},
		},
		{
			name: "UpdateB2CTenant",
			run: func() error {
				_, err := b2cTenantsClient.Update(ctx, resourceGroup, b2cTenantName, &azureadexternalidentities.B2CTenantUpdateRequest{
					Sku: &azureadexternalidentities.B2CResourceSKU{
						Name: azureadexternalidentities.B2CResourceSKUNamePremiumP1,
						Tier: azureadexternalidentities.B2CResourceSKUTierA0,
					},
					B2CTenantResourceProperties: &azureadexternalidentities.B2CTenantResourceProperties{
						TenantID: strPtr(tenantID),
						BillingConfig: &azureadexternalidentities.B2CTenantResourcePropertiesBillingConfig{
							BillingType: azureadexternalidentities.BillingTypeMAU,
						},
					},
					Tags: map[string]*string{
						"env":   strPtr("stage"),
						"owner": strPtr("stackyard"),
					},
				})
				return err
			},
		},
		{
			name: "CreateGuestUsage",
			run: func() error {
				_, err := guestUsagesClient.Create(ctx, resourceGroup, guestUsageName, &azureadexternalidentities.GuestUsagesResource{
					Location: strPtr("global"),
					Tags: map[string]*string{
						"env": strPtr("dev"),
					},
					GuestUsagesResourceProperties: &azureadexternalidentities.GuestUsagesResourceProperties{
						TenantID: strPtr(tenantID),
					},
				})
				return err
			},
		},
		{
			name: "GetGuestUsage",
			run: func() error {
				_, err := guestUsagesClient.Get(ctx, resourceGroup, guestUsageName)
				return err
			},
		},
		{
			name: "ListGuestUsagesByResourceGroup",
			run: func() error {
				_, err := guestUsagesClient.ListByResourceGroup(ctx, resourceGroup)
				return err
			},
		},
		{
			name: "ListGuestUsagesBySubscription",
			run: func() error {
				_, err := guestUsagesClient.ListBySubscription(ctx)
				return err
			},
		},
		{
			name: "UpdateGuestUsage",
			run: func() error {
				_, err := guestUsagesClient.Update(ctx, resourceGroup, guestUsageName, &azureadexternalidentities.GuestUsagesResourcePatch{
					Tags: map[string]*string{
						"env":   strPtr("prod"),
						"owner": strPtr("stackyard"),
					},
				})
				return err
			},
		},
		{
			name: "DeleteGuestUsage",
			run: func() error {
				_, err := guestUsagesClient.Delete(ctx, resourceGroup, guestUsageName)
				return err
			},
		},
		{
			name: "DeleteB2CTenant",
			run: func() error {
				_, err := b2cTenantsClient.Delete(ctx, resourceGroup, b2cTenantName)
				return err
			},
		},
	}

	for _, call := range calls {
		if err := call.run(); err != nil {
			if isNotImplemented(err) {
				fmt.Printf("%s: not implemented in emulator yet\n", call.name)
				continue
			}
			exitf("%s failed: %v", call.name, err)
		}
		fmt.Printf("%s: ok\n", call.name)
	}
}

func isNotImplemented(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "StatusCode=501") || strings.Contains(err.Error(), "NotImplemented") {
		return true
	}
	var detailed autorest.DetailedError
	if errors.As(err, &detailed) && detailed.StatusCode == http.StatusNotImplemented {
		return true
	}
	return false
}

func strPtr(v string) *string {
	return &v
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
