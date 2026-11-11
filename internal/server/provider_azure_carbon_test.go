package server

import (
	"net/http"
	"testing"
)

func TestAzureCarbonReportsAndMetadata(t *testing.T) {
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

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Carbon/operations?api-version="+azureCarbonAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) != 3 {
		t.Fatalf("expected three operations, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/providers/Microsoft.Carbon/queryCarbonEmissionDataAvailableDateRange?api-version="+azureCarbonAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 querying available date range, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	dateRange := providerContractJSONMap(t, resp)
	if dateRange["startDate"] == "" || dateRange["endDate"] == "" {
		t.Fatalf("expected startDate and endDate, got %#v", dateRange)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version="+azureCarbonAPIVersion, []byte(`{
		"reportType":"Overall Summary Report",
		"subscriptionList":["00000000-0000-0000-0000-000000000000","11111111-1111-1111-1111-111111111111"],
		"carbonScopeList":["Scope1","Scope2","Scope3"],
		"dateRange":{"startDate":"2025-01-01","endDate":"2025-02-01"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 querying overall summary report, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	overall := providerContractJSONMap(t, resp)
	overallValues, ok := overall["value"].([]any)
	if !ok || len(overallValues) != 1 {
		t.Fatalf("expected one overall summary row, got %#v", overall)
	}
	firstOverall, ok := overallValues[0].(map[string]any)
	if !ok || firstOverall["totalCarbonEmission"] == nil {
		t.Fatalf("expected totalCarbonEmission in overall summary, got %#v", overall)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version="+azureCarbonAPIVersion, []byte(`{
		"reportType":"Monthly Summary Report",
		"subscriptionList":["00000000-0000-0000-0000-000000000000"],
		"carbonScopeList":["Scope1","Scope2","Scope3"],
		"dateRange":{"startDate":"2025-01-01","endDate":"2025-02-01"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 querying monthly summary report, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	monthly := providerContractJSONMap(t, resp)
	if values, ok := monthly["value"].([]any); !ok || len(values) != 2 {
		t.Fatalf("expected two monthly rows, got %#v", monthly)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version="+azureCarbonAPIVersion, []byte(`{
		"reportType":"Item Details Report",
		"subscriptionList":["00000000-0000-0000-0000-000000000000"],
		"carbonScopeList":["Scope1","Scope2"],
		"dateRange":{"startDate":"2025-01-01","endDate":"2025-01-31"},
		"pageSize":1
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 querying item details report, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	details := providerContractJSONMap(t, resp)
	detailValues, ok := details["value"].([]any)
	if !ok || len(detailValues) != 1 {
		t.Fatalf("expected one paged detail row, got %#v", details)
	}
	skipToken, _ := details["skipToken"].(string)
	if skipToken == "" {
		t.Fatalf("expected skipToken for paged detail response, got %#v", details)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version="+azureCarbonAPIVersion, []byte(`{
		"reportType":"Item Details Report",
		"subscriptionList":["00000000-0000-0000-0000-000000000000"],
		"carbonScopeList":["Scope1","Scope2"],
		"dateRange":{"startDate":"2025-01-01","endDate":"2025-01-31"},
		"pageSize":1,
		"skipToken":"`+skipToken+`"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 querying second item details page, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	secondPage := providerContractJSONMap(t, resp)
	if values, ok := secondPage["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one second-page detail row, got %#v", secondPage)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version="+azureCarbonAPIVersion, []byte(`{
		"reportType":"Top Items Summary Report",
		"subscriptionList":["00000000-0000-0000-0000-000000000000"],
		"carbonScopeList":["Scope1","Scope2"],
		"dateRange":{"startDate":"2025-01-01","endDate":"2025-01-31"},
		"topItems":2
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 querying top items summary, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	topItems := providerContractJSONMap(t, resp)
	if values, ok := topItems["value"].([]any); !ok || len(values) != 2 {
		t.Fatalf("expected two top items, got %#v", topItems)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version="+azureCarbonAPIVersion, []byte(`{
		"reportType":"Top Items Monthly Summary Report",
		"subscriptionList":["00000000-0000-0000-0000-000000000000","11111111-1111-1111-1111-111111111111"],
		"carbonScopeList":["Scope1","Scope2","Scope3"],
		"dateRange":{"startDate":"2025-01-01","endDate":"2025-02-01"},
		"topItems":2
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 querying top items monthly summary, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	topItemsMonthly := providerContractJSONMap(t, resp)
	if values, ok := topItemsMonthly["value"].([]any); !ok || len(values) != 2 {
		t.Fatalf("expected two monthly top-item groups, got %#v", topItemsMonthly)
	}
}

func TestAzureCarbonValidation(t *testing.T) {
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
			path:       "/azure/providers/Microsoft.Carbon/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing report type",
			method:     http.MethodPost,
			path:       "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version=" + azureCarbonAPIVersion,
			body:       []byte(`{"subscriptionList":["00000000-0000-0000-0000-000000000000"],"carbonScopeList":["Scope1"],"dateRange":{"startDate":"2025-01-01","endDate":"2025-01-31"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid page size",
			method:     http.MethodPost,
			path:       "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version=" + azureCarbonAPIVersion,
			body:       []byte(`{"reportType":"Item Details Report","subscriptionList":["00000000-0000-0000-0000-000000000000"],"carbonScopeList":["Scope1"],"dateRange":{"startDate":"2025-01-01","endDate":"2025-01-31"},"pageSize":0}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid top items",
			method:     http.MethodPost,
			path:       "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version=" + azureCarbonAPIVersion,
			body:       []byte(`{"reportType":"Top Items Summary Report","subscriptionList":["00000000-0000-0000-0000-000000000000"],"carbonScopeList":["Scope1"],"dateRange":{"startDate":"2025-01-01","endDate":"2025-01-31"},"topItems":11}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "details multi month unsupported",
			method:     http.MethodPost,
			path:       "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version=" + azureCarbonAPIVersion,
			body:       []byte(`{"reportType":"Item Details Report","subscriptionList":["00000000-0000-0000-0000-000000000000"],"carbonScopeList":["Scope1"],"dateRange":{"startDate":"2025-01-01","endDate":"2025-02-01"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid skip token",
			method:     http.MethodPost,
			path:       "/azure/providers/Microsoft.Carbon/carbonEmissionReports?api-version=" + azureCarbonAPIVersion,
			body:       []byte(`{"reportType":"Item Details Report","subscriptionList":["00000000-0000-0000-0000-000000000000"],"carbonScopeList":["Scope1"],"dateRange":{"startDate":"2025-01-01","endDate":"2025-01-31"},"skipToken":"bad"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			method:     http.MethodPost,
			path:       "/azure/providers/Microsoft.Carbon/queryCarbonEmissionDataAvailableDateRange?api-version=" + azureCarbonAPIVersion,
			body:       []byte(`{`),
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
