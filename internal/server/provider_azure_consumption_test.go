package server

import (
	"net/http"
	"testing"
)

func TestAzureConsumptionLifecycle(t *testing.T) {
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
	subscriptionScope := "/azure/subscriptions/" + subscriptionID
	budgetPath := subscriptionScope + "/providers/Microsoft.Consumption/budgets/monthly-prod?api-version=" + azureConsumptionAPIVersion
	billingAccountScope := "/azure/providers/Microsoft.Billing/billingAccounts/mca-account"
	managementGroupScope := "/azure/providers/Microsoft.Management/managementGroups/contoso-platform"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Consumption/operations?api-version="+azureConsumptionAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing consumption operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, budgetPath, []byte(`{
		"properties":{
			"amount":1500,
			"category":"Cost",
			"timeGrain":"Monthly",
			"timePeriod":{"startDate":"2026-04-01","endDate":"2026-04-30"},
			"notifications":{"Actual_GreaterThan_80_Percent":{"enabled":true,"operator":"GreaterThan","threshold":80}}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating budget, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	budget := providerContractJSONMap(t, resp)
	properties, ok := budget["properties"].(map[string]any)
	if !ok || properties["currentSpend"] == nil || properties["forecastSpend"] == nil {
		t.Fatalf("expected budget spend snapshots, got %#v", budget)
	}

	for _, path := range []string{
		budgetPath,
		subscriptionScope + "/providers/Microsoft.Consumption/budgets?api-version=" + azureConsumptionAPIVersion,
		subscriptionScope + "/providers/Microsoft.Consumption/charges?api-version=" + azureConsumptionAPIVersion + "&top=2",
		subscriptionScope + "/providers/Microsoft.Consumption/marketplaces?api-version=" + azureConsumptionAPIVersion,
		subscriptionScope + "/providers/Microsoft.Consumption/usageDetails?api-version=" + azureConsumptionAPIVersion,
		subscriptionScope + "/providers/Microsoft.Consumption/tags?api-version=" + azureConsumptionAPIVersion,
		subscriptionScope + "/providers/Microsoft.Consumption/tags/env?api-version=" + azureConsumptionAPIVersion,
		subscriptionScope + "/providers/Microsoft.Consumption/reservationRecommendations?api-version=" + azureConsumptionAPIVersion,
		subscriptionScope + "/providers/Microsoft.Consumption/reservationRecommendationDetails?api-version=" + azureConsumptionAPIVersion,
		subscriptionScope + "/providers/Microsoft.Consumption/reservationDetails?api-version=" + azureConsumptionAPIVersion,
		subscriptionScope + "/providers/Microsoft.Consumption/reservationSummaries?api-version=" + azureConsumptionAPIVersion,
		billingAccountScope + "/providers/Microsoft.Consumption/balances?api-version=" + azureConsumptionAPIVersion,
		billingAccountScope + "/providers/Microsoft.Consumption/credits?api-version=" + azureConsumptionAPIVersion,
		billingAccountScope + "/providers/Microsoft.Consumption/events?api-version=" + azureConsumptionAPIVersion,
		billingAccountScope + "/providers/Microsoft.Consumption/lots?api-version=" + azureConsumptionAPIVersion,
		billingAccountScope + "/providers/Microsoft.Consumption/pricesheets?api-version=" + azureConsumptionAPIVersion,
		billingAccountScope + "/providers/Microsoft.Consumption/reservationTransactions?api-version=" + azureConsumptionAPIVersion,
		managementGroupScope + "/providers/Microsoft.Consumption/aggregatedcost?api-version=" + azureConsumptionAPIVersion,
		billingAccountScope + "/providers/Microsoft.Consumption/billingPeriods/202604/pricesheets?api-version=" + azureConsumptionAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodHead, budgetPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading budget, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, budgetPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting budget, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureConsumptionValidation(t *testing.T) {
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

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.Consumption/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "budget missing amount",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Consumption/budgets/test-budget?api-version=" + azureConsumptionAPIVersion,
			body:       []byte(`{"properties":{"category":"Cost","timeGrain":"Monthly","timePeriod":{"startDate":"2026-04-01","endDate":"2026-04-30"}}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "budget missing period",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Consumption/budgets/test-budget?api-version=" + azureConsumptionAPIVersion,
			body:       []byte(`{"properties":{"amount":100,"category":"Cost","timeGrain":"Monthly"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid top query",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Consumption/charges?api-version=" + azureConsumptionAPIVersion + "&top=bad",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid skiptoken query",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Consumption/usageDetails?api-version=" + azureConsumptionAPIVersion + "&skiptoken=-1",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "budget under billing period rejected",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Consumption/billingPeriods/202604/budgets/test-budget?api-version=" + azureConsumptionAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}
}
