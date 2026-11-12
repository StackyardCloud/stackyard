package server

import (
	"net/http"
	"testing"
)

func TestAzureComputeRecommenderLifecycle(t *testing.T) {
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
	base := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.Compute/locations/" + location + "/placementScores/spot"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Compute/operations?api-version="+azureComputeRecommenderAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) != 2 {
		t.Fatalf("expected two recommender operations, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"?api-version="+azureComputeRecommenderAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting spot placement score metadata, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	metadata := providerContractJSONMap(t, resp)
	properties, ok := metadata["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties in metadata payload, got %#v", metadata)
	}
	if supported, ok := properties["supportedResourceTypes"].([]any); !ok || len(supported) == 0 {
		t.Fatalf("expected supportedResourceTypes in metadata payload, got %#v", metadata)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/generate?api-version="+azureComputeRecommenderAPIVersion, []byte(`{
		"availabilityZones":["1","2"],
		"desiredCount":2,
		"desiredLocations":["eastus","westus2"],
		"desiredSizes":[
			{"sku":"Standard_DS2_v2"},
			{"sku":"Standard_D4s_v5"}
		]
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 generating spot placement scores, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if resp.Header.Get("Location") == "" {
		t.Fatalf("expected Location header on generate response")
	}
	generated := providerContractJSONMap(t, resp)
	if generated["desiredCount"] != float64(2) {
		t.Fatalf("expected desiredCount echoed back, got %#v", generated)
	}
	placementScores, ok := generated["placementScores"].([]any)
	if !ok || len(placementScores) != 8 {
		t.Fatalf("expected eight placement score rows, got %#v", generated)
	}
	first, ok := placementScores[0].(map[string]any)
	if !ok || first["score"] == "" || first["sku"] == "" || first["region"] == "" {
		t.Fatalf("expected placement score rows with score/sku/region, got %#v", generated)
	}
}

func TestAzureComputeRecommenderValidation(t *testing.T) {
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
			path:       "/azure/providers/Microsoft.Compute/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid subscription id",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/bad/providers/Microsoft.Compute/locations/eastus/placementScores/spot?api-version=" + azureComputeRecommenderAPIVersion,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Compute/locations/eastus/placementScores/spot/generate?api-version=" + azureComputeRecommenderAPIVersion,
			body:       []byte(`{`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing desired count",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Compute/locations/eastus/placementScores/spot/generate?api-version=" + azureComputeRecommenderAPIVersion,
			body:       []byte(`{"desiredLocations":["eastus"],"desiredSizes":[{"sku":"Standard_DS2_v2"}]}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing desired locations",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Compute/locations/eastus/placementScores/spot/generate?api-version=" + azureComputeRecommenderAPIVersion,
			body:       []byte(`{"desiredCount":1,"desiredSizes":[{"sku":"Standard_DS2_v2"}]}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing desired sizes",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Compute/locations/eastus/placementScores/spot/generate?api-version=" + azureComputeRecommenderAPIVersion,
			body:       []byte(`{"desiredCount":1,"desiredLocations":["eastus"]}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "desired size missing sku",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Compute/locations/eastus/placementScores/spot/generate?api-version=" + azureComputeRecommenderAPIVersion,
			body:       []byte(`{"desiredCount":1,"desiredLocations":["eastus"],"desiredSizes":[{}]}`),
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
