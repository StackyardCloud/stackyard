package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureStackAdminLifecycle(t *testing.T) {
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
	location := "eastus"
	base := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.Storage.Admin"
	settingsPath := base + "/locations/" + location + "/settings?api-version=" + azureStackAdminAPIVersion
	deletedAccountBase := base + "/locations/" + location + "/storageAccounts/account-deleted-001"
	activeAccountBase := base + "/locations/" + location + "/storageAccounts/account-active-001"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Storage.Admin/operations?api-version="+azureStackAdminAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/storageServices?api-version="+azureStackAdminAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing storage services, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	services := providerContractJSONMap(t, resp)
	if values, ok := services["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty storage services payload, got %#v", services)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, settingsPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting settings, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	settings := providerContractJSONMap(t, resp)
	props, ok := settings["properties"].(map[string]any)
	if !ok || props["retentionPeriodForDeletedStorageAccountsInDays"] == nil {
		t.Fatalf("expected settings properties, got %#v", settings)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, settingsPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading settings, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, settingsPath, []byte(`{
		"properties":{"retentionPeriodForDeletedStorageAccountsInDays":14}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 updating settings, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	updatedSettings := providerContractJSONMap(t, resp)
	updatedProps, ok := updatedSettings["properties"].(map[string]any)
	if !ok || updatedProps["retentionPeriodForDeletedStorageAccountsInDays"] != float64(14) {
		t.Fatalf("expected updated settings, got %#v", updatedSettings)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/locations/"+location+"/storageAccounts?api-version="+azureStackAdminAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing storage accounts, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	accounts := providerContractJSONMap(t, resp)
	if values, ok := accounts["value"].([]any); !ok || len(values) < 2 {
		t.Fatalf("expected seeded storage accounts, got %#v", accounts)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/locations/"+location+"/storageAccounts?summary=true&api-version="+azureStackAdminAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing summary storage accounts, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	summaryAccounts := providerContractJSONMap(t, resp)
	summaryValues, ok := summaryAccounts["value"].([]any)
	if !ok || len(summaryValues) == 0 {
		t.Fatalf("expected summary account payload, got %#v", summaryAccounts)
	}
	firstSummary, ok := summaryValues[0].(map[string]any)
	if !ok {
		t.Fatalf("expected summary account object, got %#v", summaryValues[0])
	}
	firstSummaryProps, ok := firstSummary["properties"].(map[string]any)
	if !ok || firstSummaryProps["tenantStorageAccountName"] == nil {
		t.Fatalf("expected summary properties, got %#v", firstSummary)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, activeAccountBase+"?api-version="+azureStackAdminAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting active account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	account := providerContractJSONMap(t, resp)
	if account["name"] != "account-active-001" {
		t.Fatalf("unexpected account payload: %#v", account)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, activeAccountBase+"?api-version="+azureStackAdminAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading storage account, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/locations/"+location+"/acquisitions?api-version="+azureStackAdminAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing acquisitions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	acquisitions := providerContractJSONMap(t, resp)
	if values, ok := acquisitions["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected seeded acquisitions, got %#v", acquisitions)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, deletedAccountBase+"/undelete?api-version="+azureStackAdminAPIVersion, []byte(`{
		"newAccountName":"tenantrestored001"
	}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 undeleting account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	asyncURL := resp.Header.Get("Azure-AsyncOperation")
	if asyncURL == "" {
		t.Fatalf("expected Azure-AsyncOperation header on undelete")
	}

	resp = providerContractRequest(t, ts, http.MethodGet, strings.TrimPrefix(asyncURL, "http://127.0.0.1:0"), nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting async operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	asyncPayload := providerContractJSONMap(t, resp)
	if asyncPayload["status"] != "Succeeded" {
		t.Fatalf("expected succeeded async payload, got %#v", asyncPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, deletedAccountBase+"?api-version="+azureStackAdminAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting restored account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	restored := providerContractJSONMap(t, resp)
	restoredProps, ok := restored["properties"].(map[string]any)
	if !ok || restoredProps["status"] != "Active" || restoredProps["tenantStorageAccountName"] != "tenantrestored001" {
		t.Fatalf("expected restored account properties, got %#v", restored)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, activeAccountBase+"/reclaimStorageCapacity?api-version="+azureStackAdminAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 reclaiming capacity, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	reclaimAsyncURL := resp.Header.Get("Azure-AsyncOperation")
	if reclaimAsyncURL == "" {
		t.Fatalf("expected async header on reclaim")
	}

	resp = providerContractRequest(t, ts, http.MethodGet, strings.TrimPrefix(reclaimAsyncURL, "http://127.0.0.1:0"), nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting reclaim async operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureStackAdminValidationAndNotFound(t *testing.T) {
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

	base := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Storage.Admin"
	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "invalid settings body missing retention",
			method:     http.MethodPut,
			path:       base + "/locations/eastus/settings?api-version=" + azureStackAdminAPIVersion,
			body:       []byte(`{"properties":{}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid settings json",
			method:     http.MethodPut,
			path:       base + "/locations/eastus/settings?api-version=" + azureStackAdminAPIVersion,
			body:       []byte(`{"properties":`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing storage account",
			method:     http.MethodGet,
			path:       base + "/locations/eastus/storageAccounts/missing?api-version=" + azureStackAdminAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unknown async operation",
			method:     http.MethodGet,
			path:       base + "/locations/eastus/asyncOperations/missing?api-version=" + azureStackAdminAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "undelete active account invalid state",
			method:     http.MethodPost,
			path:       base + "/locations/eastus/storageAccounts/account-active-001/undelete?api-version=" + azureStackAdminAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.Storage.Admin/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tc.method, tc.path, tc.body, headers)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tc.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}
}
