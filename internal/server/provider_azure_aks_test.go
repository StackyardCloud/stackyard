package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAKSRoutesReturnFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{
			name:   "operations list",
			method: http.MethodGet,
			path:   "/azure/providers/Microsoft.ContainerService/operations?api-version=2025-10-01",
		},
		{
			name:   "managed clusters create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/cluster-a?api-version=2025-10-01",
			body:   []byte(`{"location":"eastus","identity":{"type":"SystemAssigned"},"properties":{"dnsPrefix":"stackyardaks"}}`),
		},
		{
			name:   "agent pools create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/cluster-a/agentPools/system?api-version=2025-10-01",
			body:   []byte(`{"properties":{"count":1,"vmSize":"Standard_DS2_v2","mode":"System"}}`),
		},
		{
			name:   "machines list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/cluster-a/agentPools/system/machines?api-version=2025-10-01",
		},
		{
			name:   "maintenance configuration create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/cluster-a/maintenanceConfigurations/default?api-version=2025-10-01",
			body:   []byte(`{"properties":{"maintenanceWindow":{"schedule":{"weekly":{"dayOfWeek":"Monday","intervalWeeks":1}}}}}`),
		},
		{
			name:   "managed namespaces create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/cluster-a/managedNamespaces/ns-a?api-version=2025-10-01",
			body:   []byte(`{"properties":{"namespaceLabels":{"env":"dev"}}}`),
		},
		{
			name:   "private endpoint connections list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/cluster-a/privateEndpointConnections?api-version=2025-10-01",
		},
		{
			name:   "private link resources list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/cluster-a/privateLinkResources?api-version=2025-10-01",
		},
		{
			name:   "resolve private link service id",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/cluster-a/resolvePrivateLinkServiceId?api-version=2025-10-01",
			body:   []byte(`{"name":"cluster-a","privateLinkServiceConnectionState":{"status":"Approved","description":"approved"}}`),
		},
		{
			name:   "snapshots create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aks/providers/Microsoft.ContainerService/snapshots/snapshot-a?api-version=2025-10-01",
			body:   []byte(`{"location":"eastus","properties":{"snapshotType":"NodePool"}}`),
		},
		{
			name:   "trusted access role bindings create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/cluster-a/trustedAccessRoleBindings/binding-a?api-version=2025-10-01",
			body:   []byte(`{"properties":{"sourceResourceId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-app/providers/Microsoft.App/containerApps/app-a","roles":["Microsoft.ContainerService/managedClusters/trustedAccessRoleBindings/read"]}}`),
		},
		{
			name:   "trusted access roles list",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ContainerService/locations/eastus/trustedAccessRoles?api-version=2025-10-01",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers := map[string]string{
				"Authorization":             "SharedKey devstoreaccount1:signature",
				"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
			}
			if tt.body != nil {
				headers["Content-Type"] = "application/json"
			}

			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 for %s %s, got %d body=%s", tt.method, tt.path, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			payload := providerContractJSONMap(t, resp)
			if payload["status"] != "ok" {
				t.Fatalf("expected success payload, got %#v", payload)
			}
			if payload["provider"] != providerAzure {
				t.Fatalf("expected provider azure in payload, got %#v", payload)
			}

			expectedPath := tt.path
			if idx := strings.Index(expectedPath, "?"); idx >= 0 {
				expectedPath = expectedPath[:idx]
			}
			if payload["path"] != expectedPath {
				t.Fatalf("expected path %q in payload, got %#v", expectedPath, payload["path"])
			}
		})
	}
}

func TestAzureAKSInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/providers/Microsoft.ContainerService/operations?api-version="
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error payload, got %#v", payload)
	}
	if payload["provider"] != providerAzure || payload["path"] != "/azure/providers/Microsoft.ContainerService/operations" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureAKSUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/cluster-a/custom-preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown AKS nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
