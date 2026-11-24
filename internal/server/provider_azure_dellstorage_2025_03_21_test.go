package server

import (
	"net/http"
	"strings"
	"testing"
)

const dellTestAPIVersion = "?api-version=2025-03-21"

const dellTestBody = `{"location":"eastus","properties":{
	"marketplace":{"planId":"p","offerId":"o","planName":"n"},
	"delegatedSubnetId":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/v/subnets/dell",
	"delegatedSubnetCidr":"10.1.0.0/24","user":{"email":"ops@example.com"},"dellReferenceNumber":"DELL-1",
	"encryption":{"encryptionType":"Microsoft-managed keys (MMK)"},"smartConnectFqdnPassword":"secret"}}`

func TestAzureDellStorage20250321Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dell")
	fs := sub + "/resourceGroups/rg-dell/providers/Dell.Storage/filesystems/fs1"

	resp := do(http.MethodPut, fs+dellTestAPIVersion, dellTestBody)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["provisioningState"] != "Succeeded" || props["smartConnectFqdnPassword"] != nil || !strings.HasPrefix(props["smartConnectFqdn"].(string), "fs1.") {
		t.Fatalf("unexpected file system properties: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPatch, fs+dellTestAPIVersion, `{"tags":{"env":"dev"}}`), http.StatusOK)
	if tags := providerContractJSONMap(t, do(http.MethodGet, fs+dellTestAPIVersion, ""))["tags"].(map[string]any); tags["env"] != "dev" {
		t.Fatalf("tags not patched: %#v", tags)
	}
	for _, list := range []string{sub + "/resourceGroups/rg-dell/providers/Dell.Storage/filesystems", sub + "/providers/Dell.Storage/filesystems"} {
		resp = do(http.MethodGet, list+dellTestAPIVersion, "")
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		if n := len(providerContractJSONMap(t, resp)["value"].([]any)); n != 1 {
			t.Fatalf("%s listed %d file systems", list, n)
		}
	}
	resp = do(http.MethodGet, "/azure/providers/Dell.Storage/operations"+dellTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if len(providerContractJSONMap(t, resp)["value"].([]any)) == 0 {
		t.Fatal("expected operations")
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, fs+dellTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, fs+dellTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureDellStorage20250321Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dell")
	base := sub + "/resourceGroups/rg-dell/providers/Dell.Storage/filesystems/"
	cases := []struct{ name, path, body string }{
		{"bad name", base + "bad.name", dellTestBody},
		{"bad cidr", base + "fs", strings.Replace(dellTestBody, "10.1.0.0/24", "10.1.0.0/20", 1)},
		{"bad encryption", base + "fs", strings.Replace(dellTestBody, "Microsoft-managed keys (MMK)", "none", 1)},
		{"missing email", base + "fs", strings.Replace(dellTestBody, `"email":"ops@example.com"`, `"email":""`, 1)},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+dellTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectStatus(t, resp, http.StatusBadRequest) })
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, base+"missing"+dellTestAPIVersion, ""), http.StatusNotFound)
}
