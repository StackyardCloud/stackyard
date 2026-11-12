package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureBillingLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	subscriptionID := "00000000-0000-0000-0000-000000000000"
	accountName := "mca-account"
	base := "/azure/providers/Microsoft.Billing"
	accountBase := base + "/billingAccounts/" + accountName
	subscriptionBase := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.Billing"

	resp := providerContractRequest(t, ts, http.MethodPost, base+"/validateAddress?api-version="+azureBillingAPIVersion, []byte(`{
		"address":{
			"line1":"1 Microsoft Way",
			"city":"Redmond",
			"country":"US"
		}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 validating address, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/billingAccounts?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing billing accounts, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	accounts := providerContractJSONMap(t, resp)
	if values, ok := accounts["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected seeded billing accounts, got %#v", accounts)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting billing account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, accountBase+"?api-version="+azureBillingAPIVersion, []byte(`{
		"tags":{"env":"test"},
		"properties":{"displayName":"Contoso Billing Updated"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching billing account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/billingProperty/default?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting billing property, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, subscriptionBase+"/billingProperty/default?api-version="+azureBillingAPIVersion, []byte(`{
		"properties":{"invoiceEmailOptIn":"Disabled"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching billing property, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/billingAccounts/missing-account?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 getting missing billing account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	billingProfilePath := accountBase + "/billingProfiles/profile-a?api-version=" + azureBillingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, billingProfilePath, []byte(`{
		"properties":{"displayName":"Profile A","currency":"USD"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating billing profile, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	invoiceSectionPath := accountBase + "/billingProfiles/profile-a/invoiceSections/section-a?api-version=" + azureBillingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, invoiceSectionPath, []byte(`{
		"properties":{"displayName":"Section A"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating invoice section, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/billingProfiles/profile-a/customers?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing customers, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/availableBalances?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting account available balance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/billingRoleDefinitions?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing billing role definitions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	roleAssignmentPath := accountBase + "/billingRoleAssignments/assignment-a?api-version=" + azureBillingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, roleAssignmentPath, []byte(`{
		"properties":{
			"principalId":"44444444-4444-4444-4444-444444444444",
			"roleDefinitionId":"owner"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating billing role assignment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/billingPermissions?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing billing permissions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/paymentMethods?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing payment methods, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/policies?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing policies, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	billingSubscriptionPath := accountBase + "/billingSubscriptions/sub-a?api-version=" + azureBillingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPatch, billingSubscriptionPath, []byte(`{
		"properties":{"status":"Active"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating billing subscription via patch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, strings.TrimSuffix(billingSubscriptionPath, "?api-version="+azureBillingAPIVersion)+"/validateMoveEligibility?api-version="+azureBillingAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 validating subscription move eligibility, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	productPath := accountBase + "/products/product-a?api-version=" + azureBillingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPatch, productPath, []byte(`{
		"properties":{"status":"Active"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating product via patch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/invoices?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing invoices, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/transactions?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing transactions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	billingRequestPath := base + "/billingRequests/request-a?api-version=" + azureBillingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, billingRequestPath, []byte(`{
		"properties":{"status":"Submitted"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating billing request, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	transferPath := accountBase + "/transfers/transfer-a?api-version=" + azureBillingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, transferPath, []byte(`{
		"properties":{"status":"Draft"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating transfer, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, strings.TrimSuffix(transferPath, "?api-version="+azureBillingAPIVersion)+"/initiate?api-version="+azureBillingAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 initiating transfer, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	recipientTransferPath := base + "/recipientTransfers/recipient-a?api-version=" + azureBillingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, recipientTransferPath, []byte(`{
		"properties":{"status":"Pending"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating recipient transfer, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, strings.TrimSuffix(recipientTransferPath, "?api-version="+azureBillingAPIVersion)+"/accept?api-version="+azureBillingAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 accepting recipient transfer, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	associatedTenantPath := accountBase + "/associatedTenants/tenant-a?api-version=" + azureBillingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, associatedTenantPath, []byte(`{
		"properties":{"tenantId":"55555555-5555-5555-5555-555555555555"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating associated tenant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/departments?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing departments, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/departments/department-default/enrollmentAccounts?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing enrollment accounts, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/reservationOrders?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing reservation orders, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	reservationPath := accountBase + "/reservations/reservation-a?api-version=" + azureBillingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPatch, reservationPath, []byte(`{
		"properties":{"status":"Active"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating reservation via patch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/savingsPlanOrders?api-version="+azureBillingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing savings plan orders, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	savingsPlanPath := accountBase + "/savingsPlans/savings-plan-a?api-version=" + azureBillingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPatch, savingsPlanPath, []byte(`{
		"properties":{"status":"Active"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating savings plan via patch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/validateAddress?api-version="+azureBillingAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 validating empty address payload, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
