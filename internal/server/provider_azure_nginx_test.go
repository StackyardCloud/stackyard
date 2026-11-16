package server

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

const azureNginxTestAPIVersion = "?api-version=2025-11-01"

func azureNginxTestDeploymentBody(ids string) string {
	return `{"location":"eastus","sku":{"name":"standardv3_Monthly"},"identity":{"type":"SystemAssigned"},"properties":{"networkProfile":{"frontEndIPConfiguration":{"privateIPAddresses":[{"privateIPAddress":"10.0.1.10","privateIPAllocationMethod":"Static","subnetId":"` + ids + `/Microsoft.Network/virtualNetworks/v/subnets/nginx"}]},"networkInterfaceConfiguration":{"subnetId":"` + ids + `/Microsoft.Network/virtualNetworks/v/subnets/nginx"}},"scalingProperties":{"capacity":20},"autoUpgradeProfile":{"upgradeChannel":"stable"},"userProfile":{"preferredEmail":"ops@contoso.com"},"nginxAppProtect":{"webApplicationFirewallSettings":{"activationState":"Enabled"}}}}`
}

func TestAzureNginxLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-nginx")
	ids := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-nginx/providers"
	deployment := sub + "/resourceGroups/rg-nginx/providers/Nginx.NginxPlus/nginxDeployments/edge"

	resp := do(http.MethodPut, deployment+azureNginxTestAPIVersion, azureNginxTestDeploymentBody(ids))
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["ipAddress"] != "10.0.1.10" || !strings.HasSuffix(props["dataplaneApiEndpoint"].(string), ".eastus.nginxaas.net") || props["nginxAppProtect"].(map[string]any)["webApplicationFirewallStatus"] == nil {
		t.Fatalf("unexpected deployment properties: %#v", props)
	}

	conf := base64.StdEncoding.EncodeToString([]byte("http { server { listen 80; location / { return 200; } } }"))
	secret := base64.StdEncoding.EncodeToString([]byte("user:password"))
	config := deployment + "/configurations/default"
	resp = do(http.MethodPut, config+azureNginxTestAPIVersion, `{"properties":{"rootFile":"/etc/nginx/nginx.conf","files":[{"virtualPath":"/etc/nginx/nginx.conf","content":"`+conf+`"}],"protectedFiles":[{"virtualPath":"/etc/nginx/htpasswd","content":"`+secret+`"}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	protected := providerContractJSONMap(t, resp)["properties"].(map[string]any)["protectedFiles"].([]any)[0].(map[string]any)
	if protected["content"] != nil || len(azureARMString(protected["contentHash"])) != 64 {
		t.Fatalf("expected the protected content hashed, not returned: %#v", protected)
	}
	resp = do(http.MethodPost, config+"/analyze"+azureNginxTestAPIVersion, `{"config":{"rootFile":"/etc/nginx/nginx.conf","files":[{"virtualPath":"/etc/nginx/nginx.conf","content":"`+conf+`"}]}}`)
	if got := providerContractJSONMap(t, resp)["status"]; got != "SUCCEEDED" {
		t.Fatalf("expected a clean analysis, got %v", got)
	}
	resp = do(http.MethodPost, config+"/analyze"+azureNginxTestAPIVersion, `{"config":{"rootFile":"/etc/nginx/missing.conf","files":[{"virtualPath":"/etc/nginx/nginx.conf","content":"`+conf+`"}]}}`)
	if got := providerContractJSONMap(t, resp)["status"]; got != "FAILED" {
		t.Fatalf("expected a failed analysis, got %v", got)
	}

	resp = do(http.MethodPut, deployment+"/certificates/site"+azureNginxTestAPIVersion, `{"properties":{"keyVirtualPath":"/etc/nginx/ssl/site.key","certificateVirtualPath":"/etc/nginx/ssl/site.crt","keyVaultSecretId":"https://contoso.vault.azure.net/secrets/site/0123abcd"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["keyVaultSecretVersion"]; got != "0123abcd" {
		t.Fatalf("unexpected secret version: %v", got)
	}
	key := deployment + "/apiKeys/automation"
	azureResourcesExpectStatus(t, do(http.MethodPut, key+azureNginxTestAPIVersion, `{"properties":{"secretText":"Ünï-3f8d-ApiKey!","endDateTime":"2027-01-01T00:00:00Z"}}`), http.StatusCreated)
	resp = do(http.MethodGet, key+azureNginxTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any); got["hint"] != "Ünï" || got["secretText"] != nil {
		t.Fatalf("unexpected API key properties: %#v", got)
	}
	resp = do(http.MethodGet, config+azureNginxTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["protectedFiles"].([]any)[0].(map[string]any); got["content"] != nil {
		t.Fatalf("protected content was stored: %#v", got)
	}
	// These APIs have no PATCH.
	for _, path := range []string{key, config, deployment + "/certificates/site"} {
		azureResourcesExpectErrorCode(t, do(http.MethodPatch, path+azureNginxTestAPIVersion, `{"properties":{}}`), http.StatusMethodNotAllowed, "MethodNotAllowed")
	}
	policy := base64.StdEncoding.EncodeToString([]byte(`{"policy":{"name":"custom"}}`))
	resp = do(http.MethodPut, deployment+"/wafPolicies/custom"+azureNginxTestAPIVersion, `{"properties":{"filepath":"/etc/app_protect/conf/custom.json","content":"`+policy+`"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	resp = do(http.MethodPost, deployment+"/listDefaultWafPolicies"+azureNginxTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 2 {
		t.Fatalf("expected two default policies, got %d", len(got))
	}
	resp = do(http.MethodPost, deployment+"/wafPolicies/custom/analyzeWafPolicy"+azureNginxTestAPIVersion, `{"filepath":"/etc/app_protect/conf/custom.json","content":"not base64!"}`)
	if got := providerContractJSONMap(t, resp)["status"]; got != "FAILED" {
		t.Fatalf("expected a failed WAF analysis, got %v", got)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, deployment+azureNginxTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, config+azureNginxTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureNginxValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-nginx")
	ids := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-nginx/providers"
	deployment := sub + "/resourceGroups/rg-nginx/providers/Nginx.NginxPlus/nginxDeployments/edge"
	body := azureNginxTestDeploymentBody(ids)
	cases := []struct{ name, old, new string }{
		{"sku", `"sku":{"name":"standardv3_Monthly"},`, ``},
		{"subnet", `"networkInterfaceConfiguration":{"subnetId":"` + ids + `/Microsoft.Network/virtualNetworks/v/subnets/nginx"}`, `"networkInterfaceConfiguration":{"subnetId":"nginx"}`},
		{"frontend", `"frontEndIPConfiguration":{"privateIPAddresses":[{"privateIPAddress":"10.0.1.10","privateIPAllocationMethod":"Static","subnetId":"` + ids + `/Microsoft.Network/virtualNetworks/v/subnets/nginx"}]},`, `"frontEndIPConfiguration":{},`},
		{"static ip", `"privateIPAddress":"10.0.1.10",`, ``},
		{"capacity and autoscale", `"scalingProperties":{"capacity":20}`, `"scalingProperties":{"capacity":20,"autoScaleSettings":{"profiles":[{"name":"p","capacity":{"min":10,"max":30}}]}}`},
		{"autoscale range", `"scalingProperties":{"capacity":20}`, `"scalingProperties":{"autoScaleSettings":{"profiles":[{"name":"p","capacity":{"min":30,"max":10}}]}}`},
		{"channel", `"upgradeChannel":"stable"`, `"upgradeChannel":"beta"`},
		{"email", `"preferredEmail":"ops@contoso.com"`, `"preferredEmail":"ops"`},
		{"waf", `"activationState":"Enabled"`, `"activationState":"On"`},
	}
	for _, tc := range cases {
		changed := strings.Replace(body, tc.old, tc.new, 1)
		if changed == body {
			t.Fatalf("case %s did not change the body", tc.name)
		}
		resp := do(http.MethodPut, deployment+azureNginxTestAPIVersion, changed)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, deployment+azureNginxTestAPIVersion, body), http.StatusCreated)
	children := []struct{ name, path, body string }{
		{"config root", "/configurations/default", `{"properties":{"files":[{"virtualPath":"/etc/nginx/nginx.conf","content":"aHR0cA=="}]}}`},
		{"config root file", "/configurations/default", `{"properties":{"rootFile":"/etc/nginx/x.conf","files":[{"virtualPath":"/etc/nginx/nginx.conf","content":"aHR0cA=="}]}}`},
		{"config base64", "/configurations/default", `{"properties":{"rootFile":"/etc/nginx/nginx.conf","files":[{"virtualPath":"/etc/nginx/nginx.conf","content":"not base64!"}]}}`},
		{"config path", "/configurations/default", `{"properties":{"rootFile":"nginx.conf","files":[{"virtualPath":"nginx.conf","content":"aHR0cA=="}]}}`},
		{"cert path", "/certificates/c", `{"properties":{"keyVirtualPath":"site.key","certificateVirtualPath":"/c.crt","keyVaultSecretId":"https://kv.vault.azure.net/secrets/s"}}`},
		{"cert secret", "/certificates/c", `{"properties":{"keyVirtualPath":"/k.key","certificateVirtualPath":"/c.crt","keyVaultSecretId":"site"}}`},
		{"api key short", "/apiKeys/k", `{"properties":{"secretText":"short"}}`},
		{"api key date", "/apiKeys/k", `{"properties":{"secretText":"long-enough-secret","endDateTime":"next year"}}`},
		{"waf content", "/wafPolicies/w", `{"properties":{"filepath":"/etc/app_protect/conf/w.json"}}`},
	}
	for _, tc := range children {
		resp := do(http.MethodPut, deployment+tc.path+azureNginxTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, deployment+"/configurations/main"+azureNginxTestAPIVersion, `{"properties":{}}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-nginx/providers/Nginx.NginxPlus/nginxDeployments/-edge"+azureNginxTestAPIVersion, body), http.StatusBadRequest, "InvalidResourceName")
}
