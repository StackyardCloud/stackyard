package server

import (
	"net/http"
	"testing"
)

func TestAzureResourceGraphLifecycle(t *testing.T) {
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
	resourceGroup := "rg-graph"
	queryName := "query-a"
	base := "/azure/subscriptions/" + subscriptionID
	resource := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ResourceGraph/queries/" + queryName + "?api-version=" + azureResourceGraphAPIVersion

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.ResourceGraph/operations?api-version="+azureResourceGraphAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/providers/Microsoft.ResourceGraph/resources?api-version="+azureResourceGraphAPIVersion, []byte(`{
		"subscriptions":["00000000-0000-0000-0000-000000000000"],
		"query":"Resources | project id, name, type",
		"facets":[{"expression":"type"}],
		"options":{"$top":1}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 querying resources, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	queryResp := providerContractJSONMap(t, resp)
	if count, ok := queryResp["count"].(float64); !ok || count != 1 {
		t.Fatalf("expected one paged result, got %#v", queryResp)
	}
	if _, ok := queryResp["skipToken"].(string); !ok {
		t.Fatalf("expected skipToken in paged response, got %#v", queryResp)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, resource, []byte(`{
		"location":"global",
		"tags":{"env":"dev"},
		"properties":{
			"query":"Resources | project id, name, type",
			"description":"Graph query"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating graph query, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	created := providerContractJSONMap(t, resp)
	if created["type"] != "Microsoft.ResourceGraph/queries" {
		t.Fatalf("unexpected query payload: %#v", created)
	}
	props, ok := created["properties"].(map[string]any)
	if !ok || props["query"] == "" || props["timeModified"] == "" {
		t.Fatalf("expected query properties, got %#v", created)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting query, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	got := providerContractJSONMap(t, resp)
	if got["name"] != queryName {
		t.Fatalf("unexpected get payload: %#v", got)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading query, got %d", resp.StatusCode)
	}

	for _, path := range []string{
		base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ResourceGraph/queries?api-version=" + azureResourceGraphAPIVersion,
		base + "/providers/Microsoft.ResourceGraph/queries?api-version=" + azureResourceGraphAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
		listResp := providerContractJSONMap(t, resp)
		if values, ok := listResp["value"].([]any); !ok || len(values) != 1 {
			t.Fatalf("expected one query in %s, got %#v", path, listResp)
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, resource, []byte(`{
		"tags":{"env":"stage","owner":"stackyard"},
		"properties":{"description":"Updated graph query"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching query, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	patched := providerContractJSONMap(t, resp)
	patchedTags, ok := patched["tags"].(map[string]any)
	if !ok || patchedTags["owner"] != "stackyard" {
		t.Fatalf("expected patched tags, got %#v", patched)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/providers/Microsoft.ResourceGraph/resourcesHistory?api-version="+azureResourceGraphHistoryAPIVersion, []byte(`{
		"subscriptions":["00000000-0000-0000-0000-000000000000"],
		"query":"Resources | project id, name, type",
		"options":{"$top":2}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 querying resource history, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	history := providerContractJSONMap(t, resp)
	if count, ok := history["count"].(float64); !ok || count == 0 {
		t.Fatalf("expected non-empty history payload, got %#v", history)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/providers/Microsoft.ResourceGraph/resourceChanges?api-version="+azureResourceGraphChangesAPIVersion, []byte(`{
		"subscriptions":["00000000-0000-0000-0000-000000000000"],
		"interval":{"start":"2026-03-01T00:00:00Z","end":"2026-04-01T00:00:00Z"},
		"$top":1,
		"fetchPropertyChanges":true,
		"fetchSnapshots":true
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 querying resource changes, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	changes := providerContractJSONMap(t, resp)
	changeItems, ok := changes["changes"].([]any)
	if !ok || len(changeItems) != 1 {
		t.Fatalf("expected one paged change item, got %#v", changes)
	}
	firstChange, ok := changeItems[0].(map[string]any)
	if !ok || firstChange["changeId"] == "" {
		t.Fatalf("expected changeId in changes payload, got %#v", changes)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/providers/Microsoft.ResourceGraph/resourceChangeDetails?api-version="+azureResourceGraphChangesAPIVersion, []byte(`{
		"changeIds":["`+firstChange["changeId"].(string)+`"],
		"resourceIds":["/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-app/providers/Microsoft.Compute/virtualMachines/vm-a"]
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 querying resource change details, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	details := providerContractJSONMap(t, resp)
	if values, ok := details["changes"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one change detail item, got %#v", details)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting query, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, resource, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing query, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureResourceGraphValidationAndNotFound(t *testing.T) {
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
	resourceGroup := "rg-graph"
	base := "/azure/subscriptions/" + subscriptionID

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "resources missing query",
			method:     http.MethodPost,
			path:       "/azure/providers/Microsoft.ResourceGraph/resources?api-version=" + azureResourceGraphAPIVersion,
			body:       []byte(`{"subscriptions":["00000000-0000-0000-0000-000000000000"]}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid query resource name",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ResourceGraph/queries/bad:name?api-version=" + azureResourceGraphAPIVersion,
			body:       []byte(`{"properties":{"query":"Resources | project id"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "create missing properties query",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ResourceGraph/queries/query-a?api-version=" + azureResourceGraphAPIVersion,
			body:       []byte(`{"properties":{"description":"missing query"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing query get",
			method:     http.MethodGet,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ResourceGraph/queries/missing?api-version=" + azureResourceGraphAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "resources history wrong version",
			method:     http.MethodPost,
			path:       "/azure/providers/Microsoft.ResourceGraph/resourcesHistory?api-version=" + azureResourceGraphAPIVersion,
			body:       []byte(`{"query":"Resources | project id"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "resource changes missing interval",
			method:     http.MethodPost,
			path:       "/azure/providers/Microsoft.ResourceGraph/resourceChanges?api-version=" + azureResourceGraphChangesAPIVersion,
			body:       []byte(`{"subscriptions":["00000000-0000-0000-0000-000000000000"]}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "resource change details missing ids",
			method:     http.MethodPost,
			path:       "/azure/providers/Microsoft.ResourceGraph/resourceChangeDetails?api-version=" + azureResourceGraphChangesAPIVersion,
			body:       []byte(`{"changeIds":[],"resourceIds":[]}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid api version empty",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.ResourceGraph/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}
}
