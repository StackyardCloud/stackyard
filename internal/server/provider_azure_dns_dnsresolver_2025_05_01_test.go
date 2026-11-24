package server

import (
	"net/http"
	"strings"
	"testing"
)

const dnsrTestAPIVersion = "?api-version=2025-05-01"

func TestAzureDNSResolverLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dnsr")
	vnetID, vnetPath := pdnsTestVNet(t, sub, "rg-dnsr", do)
	providers := sub + "/resourceGroups/rg-dnsr/providers/Microsoft.Network/"
	idOf := func(path string) string { return strings.TrimPrefix(path, "/azure") }

	resolver := providers + "dnsResolvers/resolver1"
	resp := do(http.MethodPut, resolver+dnsrTestAPIVersion, `{"location":"eastus","properties":{"virtualNetwork":{"id":"`+vnetID+`"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["dnsResolverState"] != "Connected" || props["resourceGuid"] == nil {
		t.Fatalf("unexpected resolver: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, resolver+"/inboundEndpoints/in1"+dnsrTestAPIVersion,
		`{"location":"eastus","properties":{"ipConfigurations":[{"subnet":{"id":"`+vnetID+`/subnets/dns"},"privateIpAllocationMethod":"Dynamic"}]}}`), http.StatusCreated)
	outbound := resolver + "/outboundEndpoints/out1"
	azureResourcesExpectStatus(t, do(http.MethodPut, outbound+dnsrTestAPIVersion,
		`{"location":"eastus","properties":{"subnet":{"id":"`+vnetID+`/subnets/web"}}}`), http.StatusCreated)

	ruleset := providers + "dnsForwardingRulesets/rules1"
	azureResourcesExpectStatus(t, do(http.MethodPut, ruleset+dnsrTestAPIVersion,
		`{"location":"eastus","properties":{"dnsResolverOutboundEndpoints":[{"id":"`+idOf(outbound)+`"}]}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, ruleset+"/forwardingRules/corp"+dnsrTestAPIVersion,
		`{"properties":{"domainName":"corp.contoso.com.","targetDnsServers":[{"ipAddress":"10.10.0.4","port":53}],"forwardingRuleState":"Enabled"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, ruleset+"/virtualNetworkLinks/hub"+dnsrTestAPIVersion,
		`{"properties":{"virtualNetwork":{"id":"`+vnetID+`"}}}`), http.StatusCreated)

	resp = do(http.MethodPost, vnetPath+"/listDnsResolvers"+dnsrTestAPIVersion, "")
	if items := adnsTestValue(t, resp); len(items) != 1 || items[0].(map[string]any)["id"] != idOf(resolver) {
		t.Fatalf("unexpected listDnsResolvers: %#v", items)
	}
	resp = do(http.MethodPost, vnetPath+"/listDnsForwardingRulesets"+dnsrTestAPIVersion, "")
	if items := adnsTestValue(t, resp); len(items) != 1 || items[0].(map[string]any)["id"] != idOf(ruleset) {
		t.Fatalf("unexpected listDnsForwardingRulesets: %#v", items)
	}

	domains := providers + "dnsResolverDomainLists/blocked"
	azureResourcesExpectStatus(t, do(http.MethodPut, domains+dnsrTestAPIVersion, `{"location":"eastus","properties":{"domains":["malware.example.com."]}}`), http.StatusCreated)
	resp = do(http.MethodPost, domains+"/bulk"+dnsrTestAPIVersion, `{"properties":{"action":"Download","storageUrl":"https://acct.blob.core.windows.net/lists/blocked.txt"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["domains"].([]any); len(got) != 1 {
		t.Fatalf("unexpected bulk response domains: %#v", got)
	}

	policy := providers + "dnsResolverPolicies/policy1"
	azureResourcesExpectStatus(t, do(http.MethodPut, policy+dnsrTestAPIVersion, `{"location":"eastus"}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, policy+"/dnsSecurityRules/block"+dnsrTestAPIVersion,
		`{"location":"eastus","properties":{"priority":100,"action":{"actionType":"Block"},"dnsResolverDomainLists":[{"id":"`+idOf(domains)+`"}]}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, policy+"/virtualNetworkLinks/hub"+dnsrTestAPIVersion,
		`{"location":"eastus","properties":{"virtualNetwork":{"id":"`+vnetID+`"}}}`), http.StatusCreated)
	if items := adnsTestValue(t, do(http.MethodPost, vnetPath+"/listDnsResolverPolicies"+dnsrTestAPIVersion, "")); len(items) != 1 {
		t.Fatalf("unexpected listDnsResolverPolicies: %#v", items)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, resolver+dnsrTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, outbound+dnsrTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureDNSResolverValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dnsr")
	vnetID, _ := pdnsTestVNet(t, sub, "rg-dnsr", do)
	providers := sub + "/resourceGroups/rg-dnsr/providers/Microsoft.Network/"
	missing := "/subscriptions/" + azureResourcesTestSubscription + "/resourceGroups/rg-dnsr/providers/Microsoft.Network/virtualNetworks/none"

	cases := []struct{ name, path, body string }{
		{"resolver without network", "dnsResolvers/r1", `{"location":"eastus","properties":{}}`},
		{"resolver on missing network", "dnsResolvers/r1", `{"location":"eastus","properties":{"virtualNetwork":{"id":"` + missing + `"}}}`},
		{"ruleset without endpoints", "dnsForwardingRulesets/rs", `{"location":"eastus","properties":{"dnsResolverOutboundEndpoints":[]}}`},
		{"domain list bad domain", "dnsResolverDomainLists/dl", `{"location":"eastus","properties":{"domains":["no-trailing-dot.com"]}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, providers+tc.path+dnsrTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	resolver := providers + "dnsResolvers/r1"
	azureResourcesExpectStatus(t, do(http.MethodPut, resolver+dnsrTestAPIVersion, `{"location":"eastus","properties":{"virtualNetwork":{"id":"`+vnetID+`"}}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, resolver+"/inboundEndpoints/in"+dnsrTestAPIVersion,
		`{"location":"eastus","properties":{"ipConfigurations":[{"subnet":{"id":"`+vnetID+`/subnets/missing"}}]}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, resolver+"/inboundEndpoints/in"+dnsrTestAPIVersion,
		`{"location":"eastus","properties":{"ipConfigurations":[{"subnet":{"id":"`+vnetID+`/subnets/dns"},"privateIpAllocationMethod":"Static"}]}}`), http.StatusBadRequest, "InvalidRequestContent")

	ruleset := providers + "dnsForwardingRulesets/rs"
	outbound := strings.TrimPrefix(resolver, "/azure") + "/outboundEndpoints/out"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, ruleset+dnsrTestAPIVersion,
		`{"location":"eastus","properties":{"dnsResolverOutboundEndpoints":[{"id":"`+outbound+`"}]}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodGet, ruleset+"/forwardingRules"+dnsrTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")

	policy := providers + "dnsResolverPolicies/p"
	azureResourcesExpectStatus(t, do(http.MethodPut, policy+dnsrTestAPIVersion, `{"location":"eastus"}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, policy+"/dnsSecurityRules/r"+dnsrTestAPIVersion,
		`{"location":"eastus","properties":{"priority":5,"action":{"actionType":"Block"},"dnsResolverDomainLists":[]}}`), http.StatusBadRequest, "InvalidRequestContent")

	domains := providers + "dnsResolverDomainLists/dl"
	azureResourcesExpectStatus(t, do(http.MethodPut, domains+dnsrTestAPIVersion, `{"location":"eastus"}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, domains+"/bulk"+dnsrTestAPIVersion, `{"properties":{"action":"Upload","storageUrl":"http://insecure"}}`), http.StatusBadRequest, "InvalidRequestContent")
}
