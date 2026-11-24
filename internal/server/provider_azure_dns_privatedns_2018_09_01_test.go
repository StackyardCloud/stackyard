package server

import (
	"net/http"
	"testing"
)

const pdnsTestAPIVersion = "?api-version=2018-09-01"

// pdnsTestVNet creates virtual network hub with subnets web and dns in rg and
// returns its ARM ID (without the /azure prefix) and request path.
func pdnsTestVNet(t *testing.T, sub, rg string, do func(method, path, body string) *http.Response) (string, string) {
	t.Helper()
	path := sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/virtualNetworks/hub"
	azureResourcesExpectStatus(t, do(http.MethodPut, path+azureNetworkTestAPIVersion,
		`{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]},"subnets":[`+
			`{"name":"web","properties":{"addressPrefix":"10.0.1.0/24"}},{"name":"dns","properties":{"addressPrefix":"10.0.2.0/28"}}]}}`), http.StatusCreated)
	return path[len("/azure"):], path
}

func TestAzurePrivateDNSLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pdns")
	vnetID, _ := pdnsTestVNet(t, sub, "rg-pdns", do)
	zone := sub + "/resourceGroups/rg-pdns/providers/Microsoft.Network/privateDnsZones/internal.contoso.com"

	resp := do(http.MethodPut, zone+pdnsTestAPIVersion, `{"location":"global"}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["numberOfRecordSets"] != float64(1) || props["maxNumberOfVirtualNetworkLinks"] != float64(pdnsMaxLinks) {
		t.Fatalf("unexpected private zone: %#v", props)
	}
	resp = do(http.MethodGet, zone+"/SOA/@"+pdnsTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if soa := providerContractJSONMap(t, resp)["properties"].(map[string]any)["soaRecord"].(map[string]any); soa["host"] != "azureprivatedns.net" {
		t.Fatalf("unexpected SOA: %#v", soa)
	}

	resp = do(http.MethodPut, zone+"/A/db"+pdnsTestAPIVersion, `{"properties":{"ttl":10,"aRecords":[{"ipv4Address":"10.0.1.4"}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if fqdn := providerContractJSONMap(t, resp)["properties"].(map[string]any)["fqdn"]; fqdn != "db.internal.contoso.com." {
		t.Fatalf("unexpected fqdn %v", fqdn)
	}
	if all := adnsTestValue(t, do(http.MethodGet, zone+"/ALL"+pdnsTestAPIVersion, "")); len(all) != 2 {
		t.Fatalf("expected 2 record sets, got %d", len(all))
	}

	link := zone + "/virtualNetworkLinks/hub-link"
	resp = do(http.MethodPut, link+pdnsTestAPIVersion, `{"location":"global","properties":{"registrationEnabled":true,"virtualNetwork":{"id":"`+vnetID+`"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if state := providerContractJSONMap(t, resp)["properties"].(map[string]any)["virtualNetworkLinkState"]; state != "Completed" {
		t.Fatalf("unexpected link state %v", state)
	}
	resp = do(http.MethodGet, zone+pdnsTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["numberOfVirtualNetworkLinks"] != float64(1) ||
		props["numberOfVirtualNetworkLinksWithRegistration"] != float64(1) || props["numberOfRecordSets"] != float64(2) {
		t.Fatalf("unexpected zone counts: %#v", props)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, zone+pdnsTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, link+pdnsTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzurePrivateDNSValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pdns")
	zones := sub + "/resourceGroups/rg-pdns/providers/Microsoft.Network/privateDnsZones/"
	zone := zones + "corp.local"
	missingVNet := "/subscriptions/" + azureResourcesTestSubscription + "/resourceGroups/rg-pdns/providers/Microsoft.Network/virtualNetworks/none"

	azureResourcesExpectErrorCode(t, do(http.MethodPut, zones+"other.local"+pdnsTestAPIVersion, `{"location":"westus"}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, zone+pdnsTestAPIVersion, `{"location":"global"}`), http.StatusCreated)

	cases := []struct{ name, path, body string }{
		{"public member name", "/A/db", `{"properties":{"ARecords":[{"ipv4Address":"10.0.0.1"}]}}`},
		{"bad srv port", "/SRV/_sip._tcp", `{"properties":{"srvRecords":[{"priority":1,"weight":1,"port":70000,"target":"sip.corp.local"}]}}`},
		{"empty txt", "/TXT/note", `{"properties":{"txtRecords":[{"value":[]}]}}`},
		{"link without network", "/virtualNetworkLinks/l1", `{"location":"global","properties":{}}`},
		{"link to missing network", "/virtualNetworkLinks/l1", `{"location":"global","properties":{"virtualNetwork":{"id":"` + missingVNet + `"}}}`},
		{"link registration not bool", "/virtualNetworkLinks/l1", `{"location":"global","properties":{"registrationEnabled":"yes","virtualNetwork":{"id":"` + missingVNet + `"}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, zone+tc.path+pdnsTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, zone+"/SOA/@"+pdnsTestAPIVersion, ""), http.StatusConflict)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, zones+"absent.local/ALL"+pdnsTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}
