package server

import (
	"net/http"
	"strings"
	"testing"
)

const afwTestAPI = "?api-version=2025-05-01"

func TestAzureFirewallAndPolicies(t *testing.T) {
	t.Parallel()
	c := newAzureARMTestClient(t, "rg-afw")
	base := c.sub() + "/resourceGroups/rg-afw/providers/Microsoft.Network"
	c.expect(http.MethodPut, base+"/virtualNetworks/hub"+afwTestAPI,
		`{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]},"subnets":[{"name":"AzureFirewallSubnet","properties":{"addressPrefix":"10.0.1.0/26"}},{"name":"apps","properties":{"addressPrefix":"10.0.2.0/24"}}]}}`, http.StatusCreated)
	firewallSubnet := strings.TrimPrefix(base, "/azure") + "/virtualNetworks/hub/subnets/AzureFirewallSubnet"
	appsSubnet := strings.TrimPrefix(base, "/azure") + "/virtualNetworks/hub/subnets/apps"

	policy := base + "/firewallPolicies/edge"
	c.expect(http.MethodPut, policy+afwTestAPI, `{"location":"eastus","properties":{"sku":{"tier":"Premium"},"threatIntelMode":"Alert","intrusionDetection":{"mode":"Alert"}}}`, http.StatusCreated)
	c.expectError(http.MethodPut, base+"/firewallPolicies/basic"+afwTestAPI, `{"location":"eastus","properties":{"sku":{"tier":"Standard"},"intrusionDetection":{"mode":"Deny"}}}`, http.StatusBadRequest, "InvalidRequestContent")

	group := `{"properties":{"priority":200,"ruleCollections":[{"ruleCollectionType":"FirewallPolicyFilterRuleCollection","name":"web","priority":300,"action":{"type":"Allow"},"rules":[{"ruleType":"ApplicationRule","name":"github","targetFqdns":["github.com"],"protocols":[{"protocolType":"Https","port":443}],"sourceAddresses":["10.0.2.0/24"]}]}]}}`
	c.expect(http.MethodPut, policy+"/ruleCollectionGroups/apps"+afwTestAPI, group, http.StatusCreated)
	c.expectError(http.MethodPut, policy+"/ruleCollectionGroups/bad"+afwTestAPI, strings.Replace(group, `"priority":200`, `"priority":50`, 1), http.StatusBadRequest, "InvalidRequestContent")
	c.expectError(http.MethodPut, policy+"/ruleCollectionGroups/bad"+afwTestAPI, strings.Replace(group, "ApplicationRule", "NatRule", 1), http.StatusBadRequest, "InvalidRequestContent")
	groups := c.expect(http.MethodGet, policy+afwTestAPI, "", http.StatusOK)["properties"].(map[string]any)["ruleCollectionGroups"].([]any)
	if len(groups) != 1 {
		t.Fatalf("the policy lists its rule collection groups: %#v", groups)
	}

	// Drafts replace the live policy and groups when deployed.
	c.expect(http.MethodPut, policy+"/firewallPolicyDrafts/default"+afwTestAPI, `{"properties":{"sku":{"tier":"Premium"},"threatIntelMode":"Deny"}}`, http.StatusCreated)
	c.expect(http.MethodPut, policy+"/ruleCollectionGroups/apps/ruleCollectionGroupDrafts/default"+afwTestAPI, strings.Replace(group, `"priority":200`, `"priority":250`, 1), http.StatusCreated)
	c.expectError(http.MethodPut, policy+"/firewallPolicyDrafts/other"+afwTestAPI, `{"properties":{}}`, http.StatusBadRequest, "InvalidResourceName")
	c.expect(http.MethodPost, policy+"/deploy"+afwTestAPI, "", http.StatusOK)
	if mode := c.expect(http.MethodGet, policy+afwTestAPI, "", http.StatusOK)["properties"].(map[string]any)["threatIntelMode"]; mode != "Deny" {
		t.Fatalf("deploy promotes the draft policy, got threatIntelMode %v", mode)
	}
	if priority := c.expect(http.MethodGet, policy+"/ruleCollectionGroups/apps"+afwTestAPI, "", http.StatusOK)["properties"].(map[string]any)["priority"]; priority != 250.0 {
		t.Fatalf("deploy promotes the draft group, got priority %v", priority)
	}
	c.expectError(http.MethodGet, policy+"/firewallPolicyDrafts/default"+afwTestAPI, "", http.StatusNotFound, "ResourceNotFound")

	// IDPS: overrides change a signature's mode in the catalog.
	c.expect(http.MethodPatch, policy+"/signatureOverrides/default"+afwTestAPI, `{"properties":{"signatures":{"2008983":"Deny"}}}`, http.StatusOK)
	c.expectError(http.MethodPatch, policy+"/signatureOverrides/default"+afwTestAPI, `{"properties":{"signatures":{"2008983":"Block"}}}`, http.StatusBadRequest, "InvalidRequestContent")
	found := c.expect(http.MethodPost, policy+"/listIdpsSignatures"+afwTestAPI, `{"resultsPerPage":10,"filters":[{"field":"Mode","values":["Deny"]}]}`, http.StatusOK)
	if found["matchingRecordsCount"] != 1.0 {
		t.Fatalf("expected the overridden signature, got %#v", found)
	}
	options := c.expect(http.MethodPost, policy+"/listIdpsFilterOptions"+afwTestAPI, `{"filterName":"Protocol"}`, http.StatusOK)
	if len(options["filterValues"].([]any)) != 2 {
		t.Fatalf("unexpected protocol filter values: %#v", options)
	}
	c.expectError(http.MethodPost, policy+"/listIdpsFilterOptions"+afwTestAPI, `{"filterName":"Color"}`, http.StatusBadRequest, "InvalidRequestContent")
	c.expectError(http.MethodPost, policy+"/listIdpsSignatures"+afwTestAPI, `{"resultsPerPage":5000}`, http.StatusBadRequest, "InvalidRequestContent")

	firewall := base + "/azureFirewalls/edge-fw"
	fwBody := `{"location":"eastus","properties":{"sku":{"name":"AZFW_VNet","tier":"Premium"},"firewallPolicy":{"id":"` + strings.TrimPrefix(policy, "/azure") + `"},"ipConfigurations":[{"name":"ipconfig","properties":{"subnet":{"id":"%s"}}}]}}`
	resp := c.do(http.MethodPut, firewall+afwTestAPI, strings.Replace(fwBody, "%s", appsSubnet, 1))
	azureResourcesExpectStatus(t, resp, http.StatusBadRequest)
	if msg, _ := providerContractJSONMap(t, resp)["error"].(map[string]any)["message"].(string); !strings.Contains(msg, "AzureFirewallSubnet") {
		t.Fatalf("the error must name the required subnet, got %q", msg)
	}
	created := c.expect(http.MethodPut, firewall+afwTestAPI, strings.Replace(fwBody, "%s", firewallSubnet, 1), http.StatusCreated)
	config := created["properties"].(map[string]any)["ipConfigurations"].([]any)[0].(map[string]any)["properties"].(map[string]any)
	if config["privateIPAddress"] != "10.0.1.4" {
		t.Fatalf("the firewall takes the subnet's first usable address: %#v", config)
	}
	c.expect(http.MethodPost, firewall+"/learnedIPPrefixes"+afwTestAPI, "", http.StatusOK)
	c.expect(http.MethodPost, firewall+"/packetCapture"+afwTestAPI, `{"properties":{"durationInSeconds":60,"numberOfPacketsToCapture":500,"sasUrl":"https://st.blob.core.windows.net/c?sig=x","fileName":"cap"}}`, http.StatusAccepted)
	c.expectError(http.MethodPost, firewall+"/packetCapture"+afwTestAPI, `{"properties":{"durationInSeconds":5}}`, http.StatusBadRequest, "InvalidRequestContent")
	status := c.expect(http.MethodPost, firewall+"/packetCaptureOperation"+afwTestAPI, `{"operation":"Status"}`, http.StatusOK)
	if status["statusCode"] == "" {
		t.Fatalf("unexpected packet capture status: %#v", status)
	}

	tags := azureTestList(t, c.expect(http.MethodGet, c.sub()+"/providers/Microsoft.Network/azureFirewallFqdnTags"+afwTestAPI, "", http.StatusOK))
	if len(tags) == 0 {
		t.Fatalf("expected FQDN tags")
	}
	category := c.expect(http.MethodGet, c.sub()+"/providers/Microsoft.Network/azureWebCategories/Gambling"+afwTestAPI, "", http.StatusOK)
	if category["properties"].(map[string]any)["group"] != "Liability" {
		t.Fatalf("unexpected web category: %#v", category)
	}
	c.expectError(http.MethodGet, c.sub()+"/providers/Microsoft.Network/azureWebCategories/Unknown"+afwTestAPI, "", http.StatusNotFound, "ResourceNotFound")
}
