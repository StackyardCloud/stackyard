package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureCloudNGFWLifecycle(t *testing.T) {
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
	resourceGroup := "rg-ngfw"
	firewallName := "fw-a"
	localRulestackName := "local-a"
	globalRulestackName := "global-a"
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	firewallBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/PaloAltoNetworks.Cloudngfw/firewalls/" + firewallName
	localBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/PaloAltoNetworks.Cloudngfw/localRulestacks/" + localRulestackName
	globalBase := "/azure/providers/PaloAltoNetworks.Cloudngfw/globalRulestacks/" + globalRulestackName

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/PaloAltoNetworks.Cloudngfw/operations?api-version="+azureCloudNGFWAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, firewallBase+"?api-version="+azureCloudNGFWAPIVersion, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{
			"localRulestackName":"`+localRulestackName+`",
			"globalRulestackName":"`+globalRulestackName+`"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating firewall, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	firewall := providerContractJSONMap(t, resp)
	if firewall["type"] != "PaloAltoNetworks.Cloudngfw/firewalls" {
		t.Fatalf("unexpected firewall payload: %#v", firewall)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, firewallBase+"?api-version="+azureCloudNGFWAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting firewall, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, firewallBase+"?api-version="+azureCloudNGFWAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading firewall, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/providers/PaloAltoNetworks.Cloudngfw/firewalls?api-version="+azureCloudNGFWAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing firewalls by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/resourceGroups/"+resourceGroup+"/providers/PaloAltoNetworks.Cloudngfw/firewalls?api-version="+azureCloudNGFWAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing firewalls by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, firewallBase+"/statuses?api-version="+azureCloudNGFWAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing firewall statuses, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	statuses := providerContractJSONMap(t, resp)
	if values, ok := statuses["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one status row, got %#v", statuses)
	}

	for _, path := range []string{
		firewallBase + "/statuses/default?api-version=" + azureCloudNGFWAPIVersion,
		firewallBase + "/getGlobalRulestack?api-version=" + azureCloudNGFWAPIVersion,
		firewallBase + "/getLogProfile?api-version=" + azureCloudNGFWAPIVersion,
		firewallBase + "/getSupportInfo?api-version=" + azureCloudNGFWAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for firewall path %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPost, firewallBase+"/saveLogProfile?api-version="+azureCloudNGFWAPIVersion, []byte(`{
		"logType":"Threat",
		"storageURL":"https://logs.stackyard.local/fw-a"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 saving log profile, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, localBase+"?api-version="+azureCloudNGFWAPIVersion, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating local rulestack, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	local := providerContractJSONMap(t, resp)
	if local["type"] != "PaloAltoNetworks.Cloudngfw/localRulestacks" {
		t.Fatalf("unexpected local rulestack payload: %#v", local)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/providers/PaloAltoNetworks.Cloudngfw/localRulestacks?api-version="+azureCloudNGFWAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing local rulestacks by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/resourceGroups/"+resourceGroup+"/providers/PaloAltoNetworks.Cloudngfw/localRulestacks?api-version="+azureCloudNGFWAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing local rulestacks by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		localBase + "/commit?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "/revert?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "/getChangeLog?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "/getSupportInfo?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "/listFirewalls?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "/listCountries?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "/listAppIds?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "/listPredefinedUrlCategories?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "/listSecurityServices?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "/listAdvancedSecurityObjects?api-version=" + azureCloudNGFWAPIVersion,
	} {
		method := http.MethodGet
		if strings.Contains(path, "/commit") || strings.Contains(path, "/revert") {
			method = http.MethodPost
		}
		resp = providerContractRequest(t, ts, method, path, []byte(`{}`), headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for local rulestack path %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	localRuleBase := localBase + "/localRules/100"
	resp = providerContractRequest(t, ts, http.MethodPut, localRuleBase+"?api-version="+azureCloudNGFWAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"actionType":"Allow"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating local rule, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		localBase + "/localRules?api-version=" + azureCloudNGFWAPIVersion,
		localRuleBase + "?api-version=" + azureCloudNGFWAPIVersion,
		localRuleBase + "/getCounters?api-version=" + azureCloudNGFWAPIVersion,
		localRuleBase + "/refreshCounters?api-version=" + azureCloudNGFWAPIVersion,
		localRuleBase + "/resetCounters?api-version=" + azureCloudNGFWAPIVersion,
	} {
		method := http.MethodGet
		if strings.Contains(path, "refreshCounters") || strings.Contains(path, "resetCounters") {
			method = http.MethodPost
		}
		resp = providerContractRequest(t, ts, method, path, []byte(`{}`), headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for local rule path %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	for _, tc := range []struct {
		collection string
		name       string
		body       string
	}{
		{"prefixlists", "prefix-a", `{"location":"eastus","properties":{"prefixes":["10.0.0.0/24"]}}`},
		{"fqdnlists", "fqdn-a", `{"location":"eastus","properties":{"fqdnList":["example.com"]}}`},
		{"certificates", "cert-a", `{"location":"eastus","properties":{"certificateSelfSigned":{"commonName":"stackyard.local"}}}`},
	} {
		base := localBase + "/" + tc.collection + "/" + tc.name
		resp = providerContractRequest(t, ts, http.MethodPut, base+"?api-version="+azureCloudNGFWAPIVersion, []byte(tc.body), headers)
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 201/200 creating local %s, got %d body=%s", tc.collection, resp.StatusCode, string(providerContractBody(t, resp)))
		}
		resp = providerContractRequest(t, ts, http.MethodGet, localBase+"/"+tc.collection+"?api-version="+azureCloudNGFWAPIVersion, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 listing local %s, got %d body=%s", tc.collection, resp.StatusCode, string(providerContractBody(t, resp)))
		}
		resp = providerContractRequest(t, ts, http.MethodGet, base+"?api-version="+azureCloudNGFWAPIVersion, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 getting local %s, got %d body=%s", tc.collection, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/providers/PaloAltoNetworks.Cloudngfw/globalRulestacks?api-version="+azureCloudNGFWAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing global rulestacks, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, globalBase+"?api-version="+azureCloudNGFWAPIVersion, []byte(`{
		"location":"global",
		"tags":{"env":"dev"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating global rulestack, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		globalBase + "?api-version=" + azureCloudNGFWAPIVersion,
		globalBase + "/commit?api-version=" + azureCloudNGFWAPIVersion,
		globalBase + "/revert?api-version=" + azureCloudNGFWAPIVersion,
		globalBase + "/getChangeLog?api-version=" + azureCloudNGFWAPIVersion,
		globalBase + "/listFirewalls?api-version=" + azureCloudNGFWAPIVersion,
		globalBase + "/listCountries?api-version=" + azureCloudNGFWAPIVersion,
		globalBase + "/listAppIds?api-version=" + azureCloudNGFWAPIVersion,
	} {
		method := http.MethodGet
		if strings.Contains(path, "/commit") || strings.Contains(path, "/revert") {
			method = http.MethodPost
		}
		resp = providerContractRequest(t, ts, method, path, []byte(`{}`), headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for global rulestack path %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	for _, tc := range []struct {
		collection string
		name       string
		body       string
	}{
		{"prefixlists", "global-prefix", `{"location":"global","properties":{"prefixes":["192.168.0.0/16"]}}`},
		{"fqdnlists", "global-fqdn", `{"location":"global","properties":{"fqdnList":["global.example.com"]}}`},
		{"certificates", "global-cert", `{"location":"global","properties":{"certificateSelfSigned":{"commonName":"global.stackyard.local"}}}`},
	} {
		base := globalBase + "/" + tc.collection + "/" + tc.name
		resp = providerContractRequest(t, ts, http.MethodPut, base+"?api-version="+azureCloudNGFWAPIVersion, []byte(tc.body), headers)
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 201/200 creating global %s, got %d body=%s", tc.collection, resp.StatusCode, string(providerContractBody(t, resp)))
		}
		resp = providerContractRequest(t, ts, http.MethodGet, globalBase+"/"+tc.collection+"?api-version="+azureCloudNGFWAPIVersion, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 listing global %s, got %d body=%s", tc.collection, resp.StatusCode, string(providerContractBody(t, resp)))
		}
		resp = providerContractRequest(t, ts, http.MethodGet, base+"?api-version="+azureCloudNGFWAPIVersion, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 getting global %s, got %d body=%s", tc.collection, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	for _, path := range []string{
		globalBase + "/certificates/global-cert?api-version=" + azureCloudNGFWAPIVersion,
		globalBase + "/fqdnlists/global-fqdn?api-version=" + azureCloudNGFWAPIVersion,
		globalBase + "/prefixlists/global-prefix?api-version=" + azureCloudNGFWAPIVersion,
		globalBase + "?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "/certificates/cert-a?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "/fqdnlists/fqdn-a?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "/prefixlists/prefix-a?api-version=" + azureCloudNGFWAPIVersion,
		localRuleBase + "?api-version=" + azureCloudNGFWAPIVersion,
		localBase + "?api-version=" + azureCloudNGFWAPIVersion,
		firewallBase + "?api-version=" + azureCloudNGFWAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodDelete, path, nil, headers)
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 deleting %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}
}

func TestAzureCloudNGFWValidation(t *testing.T) {
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
			path:       "/azure/providers/PaloAltoNetworks.Cloudngfw/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid firewall name",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/" + subscriptionID + "/resourceGroups/rg-ngfw/providers/PaloAltoNetworks.Cloudngfw/firewalls/-bad?api-version=" + azureCloudNGFWAPIVersion,
			body:       []byte(`{"location":"eastus"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "local child missing parent",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/" + subscriptionID + "/resourceGroups/rg-ngfw/providers/PaloAltoNetworks.Cloudngfw/localRulestacks/missing/fqdnlists/fqdn-a?api-version=" + azureCloudNGFWAPIVersion,
			body:       []byte(`{"location":"eastus","properties":{"fqdnList":["example.com"]}}`),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "global child missing parent",
			method:     http.MethodPut,
			path:       "/azure/providers/PaloAltoNetworks.Cloudngfw/globalRulestacks/missing/prefixlists/prefix-a?api-version=" + azureCloudNGFWAPIVersion,
			body:       []byte(`{"location":"global","properties":{"prefixes":["10.0.0.0/24"]}}`),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "rule counters missing rule",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/" + subscriptionID + "/resourceGroups/rg-ngfw/providers/PaloAltoNetworks.Cloudngfw/localRulestacks/missing/localRules/100/getCounters?api-version=" + azureCloudNGFWAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid json",
			method:     http.MethodPut,
			path:       "/azure/providers/PaloAltoNetworks.Cloudngfw/globalRulestacks/global-a?api-version=" + azureCloudNGFWAPIVersion,
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
