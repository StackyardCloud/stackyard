package server

import (
	"net/http"
	"testing"
)

func TestAzureComputeLimitLifecycle(t *testing.T) {
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
	base := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.ComputeLimit/locations/" + location

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.ComputeLimit/operations?api-version="+azureComputeLimitAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) < 5 {
		t.Fatalf("expected compute limit operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/features?api-version="+azureComputeLimitAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing features, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	featureList := providerContractJSONMap(t, resp)
	if values, ok := featureList["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty feature list, got %#v", featureList)
	}

	featurePath := base + "/features/CrossSubscriptionSharing?api-version=" + azureComputeLimitAPIVersion
	resp = providerContractRequest(t, ts, http.MethodGet, featurePath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting feature, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	feature := providerContractJSONMap(t, resp)
	featureProperties, ok := feature["properties"].(map[string]any)
	if !ok || featureProperties["state"] != "Disabled" {
		t.Fatalf("expected disabled feature by default, got %#v", feature)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/features/CrossSubscriptionSharing/enable?api-version="+azureComputeLimitAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 enabling feature, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	featureAction := providerContractJSONMap(t, resp)
	if featureAction["status"] != "Succeeded" {
		t.Fatalf("expected succeeded feature action, got %#v", featureAction)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, featurePath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting enabled feature, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	feature = providerContractJSONMap(t, resp)
	featureProperties, ok = feature["properties"].(map[string]any)
	if !ok || featureProperties["state"] != "Enabled" {
		t.Fatalf("expected enabled feature state, got %#v", feature)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/features/CrossSubscriptionSharing/disable?api-version="+azureComputeLimitAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 disabling feature, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	sharedLimitPath := base + "/sharedLimits/dsv5-share?api-version=" + azureComputeLimitAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, sharedLimitPath, []byte(`{
		"properties":{
			"resourceName":{"value":"standardDSv5Family","localizedValue":"Standard DSv5 Family"},
			"limit":64,
			"unit":"Count"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating shared limit, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	sharedLimit := providerContractJSONMap(t, resp)
	if sharedLimit["type"] != "Microsoft.ComputeLimit/locations/sharedLimits" {
		t.Fatalf("unexpected shared limit payload: %#v", sharedLimit)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/sharedLimits?api-version="+azureComputeLimitAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing shared limits, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	sharedLimitList := providerContractJSONMap(t, resp)
	if values, ok := sharedLimitList["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one shared limit, got %#v", sharedLimitList)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, sharedLimitPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting shared limit, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	sharedLimit = providerContractJSONMap(t, resp)
	sharedLimitProperties, ok := sharedLimit["properties"].(map[string]any)
	if !ok || sharedLimitProperties["limit"] != float64(64) {
		t.Fatalf("expected shared limit properties, got %#v", sharedLimit)
	}

	guestID := "11111111-1111-1111-1111-111111111111"
	guestPath := base + "/guestSubscriptions/" + guestID + "?api-version=" + azureComputeLimitAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, guestPath, []byte(`{
		"properties":{
			"displayName":"Guest Subscription A",
			"notes":"linked for eastus"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating guest subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	guest := providerContractJSONMap(t, resp)
	if guest["name"] != guestID {
		t.Fatalf("unexpected guest subscription payload: %#v", guest)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/guestSubscriptions?api-version="+azureComputeLimitAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing guest subscriptions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	guestList := providerContractJSONMap(t, resp)
	if values, ok := guestList["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one guest subscription, got %#v", guestList)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, guestPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting guest subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	guest = providerContractJSONMap(t, resp)
	guestProperties, ok := guest["properties"].(map[string]any)
	if !ok || guestProperties["status"] != "Active" {
		t.Fatalf("expected active guest subscription payload, got %#v", guest)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/vmFamilies?api-version="+azureComputeLimitAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing vm families, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	vmFamilies := providerContractJSONMap(t, resp)
	if values, ok := vmFamilies["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected vm family list, got %#v", vmFamilies)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/vmFamilies/standardDSv5Family?api-version="+azureComputeLimitAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting vm family, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	vmFamily := providerContractJSONMap(t, resp)
	vmFamilyProperties, ok := vmFamily["properties"].(map[string]any)
	if !ok || vmFamilyProperties["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected vm family payload: %#v", vmFamily)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/vmFamilies?api-version="+azureComputeLimitAPIVersion+"&$filter=generalpurpose", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 filtering vm families, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	filteredVMFamilies := providerContractJSONMap(t, resp)
	if values, ok := filteredVMFamilies["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected filtered vm families, got %#v", filteredVMFamilies)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, guestPath, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting guest subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, sharedLimitPath, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting shared limit, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureComputeLimitValidationAndNotFound(t *testing.T) {
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
			path:       "/azure/providers/Microsoft.ComputeLimit/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid subscription id",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/not-a-sub/providers/Microsoft.ComputeLimit/locations/eastus/features?api-version=" + azureComputeLimitAPIVersion,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid shared limit name",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ComputeLimit/locations/eastus/sharedLimits/no?api-version=" + azureComputeLimitAPIVersion,
			body:       []byte(`{"properties":{"resourceName":{"value":"standardDSv5Family"},"limit":1}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "shared limit missing properties",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ComputeLimit/locations/eastus/sharedLimits/share-a?api-version=" + azureComputeLimitAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "shared limit invalid json",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ComputeLimit/locations/eastus/sharedLimits/share-a?api-version=" + azureComputeLimitAPIVersion,
			body:       []byte(`{`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid guest subscription id",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ComputeLimit/locations/eastus/guestSubscriptions/not-a-uuid?api-version=" + azureComputeLimitAPIVersion,
			body:       []byte(`{"properties":{"displayName":"guest"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "guest subscription missing properties",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ComputeLimit/locations/eastus/guestSubscriptions/11111111-1111-1111-1111-111111111111?api-version=" + azureComputeLimitAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing shared limit",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ComputeLimit/locations/eastus/sharedLimits/missing-share?api-version=" + azureComputeLimitAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing guest subscription",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ComputeLimit/locations/eastus/guestSubscriptions/11111111-1111-1111-1111-111111111111?api-version=" + azureComputeLimitAPIVersion,
			wantStatus: http.StatusNotFound,
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
