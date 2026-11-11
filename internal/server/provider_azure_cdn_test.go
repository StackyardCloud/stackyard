package server

import (
	"net/http"
	"testing"
)

func TestAzureCDNLifecycle(t *testing.T) {
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
	resourceGroup := "rg-cdn"
	profileName := "profile-a"
	endpointName := "endpoint-a"
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	profileBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Cdn/profiles/" + profileName
	endpointBase := profileBase + "/endpoints/" + endpointName

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Cdn/operations?api-version="+azureCDNAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Cdn/edgenodes?api-version="+azureCDNAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing edge nodes, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		"/azure/providers/Microsoft.Cdn/checkNameAvailability?api-version=" + azureCDNAPIVersion,
		subscriptionBase + "/providers/Microsoft.Cdn/checkNameAvailability?api-version=" + azureCDNAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodPost, path, []byte(`{
			"name":"endpoint-a",
			"type":"Microsoft.Cdn/Profiles/Endpoints"
		}`), headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 checking name availability for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPost, subscriptionBase+"/providers/Microsoft.Cdn/checkResourceUsage?api-version="+azureCDNAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing subscription resource usage, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, subscriptionBase+"/providers/Microsoft.Cdn/validateProbe?api-version="+azureCDNAPIVersion, []byte(`{
		"probeURL":"https://example.com/healthz"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 validating probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	probe := providerContractJSONMap(t, resp)
	if probe["isValid"] != true {
		t.Fatalf("expected valid probe payload, got %#v", probe)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, profileBase+"?api-version="+azureCDNAPIVersion, []byte(`{
		"location":"global",
		"tags":{"env":"dev"},
		"properties":{"sku":{"name":"Standard_Microsoft"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating profile, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	profile := providerContractJSONMap(t, resp)
	if profile["type"] != "Microsoft.Cdn/profiles" {
		t.Fatalf("unexpected profile payload: %#v", profile)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, profileBase+"?api-version="+azureCDNAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading profile, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/providers/Microsoft.Cdn/profiles?api-version="+azureCDNAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing profiles by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Cdn/profiles?api-version="+azureCDNAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing profiles by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		profileBase + "/checkResourceUsage?api-version=" + azureCDNAPIVersion,
		profileBase + "/getSupportedOptimizationTypes?api-version=" + azureCDNAPIVersion,
		profileBase + "/generateSsoUri?api-version=" + azureCDNAPIVersion,
		profileBase + "/canMigrate?api-version=" + azureCDNAPIVersion,
		profileBase + "/migrate?api-version=" + azureCDNAPIVersion,
		profileBase + "/migrationAbort?api-version=" + azureCDNAPIVersion,
		profileBase + "/migrationCommit?api-version=" + azureCDNAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodPost, path, []byte(`{}`), headers)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected 200/202 for profile action %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPut, endpointBase+"?api-version="+azureCDNAPIVersion, []byte(`{
		"location":"global",
		"tags":{"env":"dev"},
		"properties":{
			"originHostHeader":"origin.stackyard.local",
			"origins":[{"name":"origin-inline","properties":{"hostName":"origin.example.com","httpPort":80,"httpsPort":443}}],
			"originGroups":[{"name":"group-inline","properties":{"healthProbeSettings":{"probePath":"/healthz"}}}]
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating endpoint, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	endpoint := providerContractJSONMap(t, resp)
	if endpoint["type"] != "Microsoft.Cdn/profiles/endpoints" {
		t.Fatalf("unexpected endpoint payload: %#v", endpoint)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, profileBase+"/endpoints?api-version="+azureCDNAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing endpoints, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, endpointBase+"?api-version="+azureCDNAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading endpoint, got %d", resp.StatusCode)
	}

	for _, path := range []string{
		endpointBase + "/start?api-version=" + azureCDNAPIVersion,
		endpointBase + "/stop?api-version=" + azureCDNAPIVersion,
		endpointBase + "/load?api-version=" + azureCDNAPIVersion,
		endpointBase + "/purge?api-version=" + azureCDNAPIVersion,
		endpointBase + "/checkResourceUsage?api-version=" + azureCDNAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodPost, path, []byte(`{"contentPaths":["/a","/b"]}`), headers)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected 200/202 for endpoint action %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPost, endpointBase+"/validateCustomDomain?api-version="+azureCDNAPIVersion, []byte(`{
		"hostName":"cdn.example.com"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 validating custom domain, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	validation := providerContractJSONMap(t, resp)
	if validation["customDomainValidated"] != true {
		t.Fatalf("expected customDomainValidated=true, got %#v", validation)
	}

	originBase := endpointBase + "/origins/origin-a"
	resp = providerContractRequest(t, ts, http.MethodPut, originBase+"?api-version="+azureCDNAPIVersion, []byte(`{
		"properties":{"hostName":"origin-a.example.com","httpPort":80,"httpsPort":443}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating origin, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, endpointBase+"/origins?api-version="+azureCDNAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing origins, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	originGroupBase := endpointBase + "/originGroups/group-a"
	resp = providerContractRequest(t, ts, http.MethodPut, originGroupBase+"?api-version="+azureCDNAPIVersion, []byte(`{
		"properties":{"healthProbeSettings":{"probePath":"/healthz"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating origin group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, endpointBase+"/originGroups?api-version="+azureCDNAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing origin groups, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	customDomainBase := endpointBase + "/customDomains/domain-a"
	resp = providerContractRequest(t, ts, http.MethodPut, customDomainBase+"?api-version="+azureCDNAPIVersion, []byte(`{
		"properties":{"hostName":"cdn.example.com"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating custom domain, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPost, customDomainBase+"/enableCustomHttps?api-version="+azureCDNAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 enabling custom https, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPost, customDomainBase+"/disableCustomHttps?api-version="+azureCDNAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 disabling custom https, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, endpointBase+"/customDomains?api-version="+azureCDNAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing custom domains, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, endpointBase+"?api-version="+azureCDNAPIVersion, []byte(`{
		"tags":{"env":"stage"},
		"properties":{"originPath":"/content"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching endpoint, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, profileBase+"?api-version="+azureCDNAPIVersion, []byte(`{
		"tags":{"env":"stage"},
		"properties":{"responseBasedOriginErrorDetectionSettings":{"responseBasedDetectedErrorTypes":"TcpErrorsOnly"}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching profile, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		customDomainBase + "?api-version=" + azureCDNAPIVersion,
		originGroupBase + "?api-version=" + azureCDNAPIVersion,
		originBase + "?api-version=" + azureCDNAPIVersion,
		endpointBase + "?api-version=" + azureCDNAPIVersion,
		profileBase + "?api-version=" + azureCDNAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodDelete, path, nil, headers)
		if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 202/204 deleting %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}
}

func TestAzureCDNValidationAndNotFound(t *testing.T) {
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
	resourceGroup := "rg-cdn"
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	profileBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Cdn/profiles"

	resp := providerContractRequest(t, ts, http.MethodPut, profileBase+"/seed-profile?api-version="+azureCDNAPIVersion, []byte(`{"location":"global"}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 seeding profile, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, profileBase+"/seed-profile/endpoints/seed-endpoint?api-version="+azureCDNAPIVersion, []byte(`{"location":"global"}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 seeding endpoint, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "invalid profile name",
			method:     http.MethodPut,
			path:       profileBase + "/bad:name?api-version=" + azureCDNAPIVersion,
			body:       []byte(`{"location":"global"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing profile location",
			method:     http.MethodPut,
			path:       profileBase + "/profile-a?api-version=" + azureCDNAPIVersion,
			body:       []byte(`{"tags":{"env":"dev"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid name availability type",
			method:     http.MethodPost,
			path:       "/azure/providers/Microsoft.Cdn/checkNameAvailability?api-version=" + azureCDNAPIVersion,
			body:       []byte(`{"name":"endpoint-a","type":"wrong/type"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid probe url",
			method:     http.MethodPost,
			path:       subscriptionBase + "/providers/Microsoft.Cdn/validateProbe?api-version=" + azureCDNAPIVersion,
			body:       []byte(`{"probeURL":"bad-url"}`),
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing profile get",
			method:     http.MethodGet,
			path:       profileBase + "/missing?api-version=" + azureCDNAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing endpoint get",
			method:     http.MethodGet,
			path:       profileBase + "/seed-profile/endpoints/missing?api-version=" + azureCDNAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "origin invalid port",
			method:     http.MethodPut,
			path:       profileBase + "/seed-profile/endpoints/seed-endpoint/origins/origin-a?api-version=" + azureCDNAPIVersion,
			body:       []byte(`{"properties":{"hostName":"origin.example.com","httpPort":70000}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "custom domain missing hostname",
			method:     http.MethodPut,
			path:       profileBase + "/seed-profile/endpoints/seed-endpoint/customDomains/domain-a?api-version=" + azureCDNAPIVersion,
			body:       []byte(`{"properties":{}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "validate custom domain missing host",
			method:     http.MethodPost,
			path:       profileBase + "/seed-profile/endpoints/seed-endpoint/validateCustomDomain?api-version=" + azureCDNAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "custom https missing domain",
			method:     http.MethodPost,
			path:       profileBase + "/seed-profile/endpoints/seed-endpoint/customDomains/missing/enableCustomHttps?api-version=" + azureCDNAPIVersion,
			body:       []byte(`{}`),
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
