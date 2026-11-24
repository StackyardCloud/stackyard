package server

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

const xrTestAPI = "?api-version=2025-05-01"

func TestAzureExpressRouteCircuits(t *testing.T) {
	t.Parallel()
	c := newAzureARMTestClient(t, "rg-xr")
	base := c.sub() + "/resourceGroups/rg-xr/providers/Microsoft.Network"
	circuit := base + "/expressRouteCircuits/hq"
	body := `{"location":"eastus","sku":{"name":"Standard_MeteredData","tier":"Standard","family":"MeteredData"},"properties":{"serviceProviderProperties":{"serviceProviderName":"Equinix","peeringLocation":"Silicon Valley","bandwidthInMbps":200}}}`

	created := c.expect(http.MethodPut, circuit+xrTestAPI, body, http.StatusCreated)
	props := created["properties"].(map[string]any)
	if props["serviceKey"] == "" || props["serviceProviderProvisioningState"] != "NotProvisioned" {
		t.Fatalf("unexpected circuit: %#v", props)
	}
	peering := `{"properties":{"peeringType":"AzurePrivatePeering","peerASN":65010,"primaryPeerAddressPrefix":"192.168.10.0/30","secondaryPeerAddressPrefix":"192.168.10.4/30","vlanId":200}}`
	c.expectError(http.MethodPut, circuit+"/peerings/Private"+xrTestAPI, peering, http.StatusBadRequest, "InvalidRequestContent")
	c.expect(http.MethodPut, circuit+"/peerings/AzurePrivatePeering"+xrTestAPI, peering, http.StatusCreated)
	c.expectError(http.MethodPut, circuit+"/peerings/AzurePrivatePeering"+xrTestAPI, strings.Replace(peering, "/30", "/29", 1), http.StatusBadRequest, "InvalidRequestContent")
	if got := c.expect(http.MethodGet, circuit+xrTestAPI, "", http.StatusOK)["properties"].(map[string]any)["peerings"].([]any); len(got) != 1 {
		t.Fatalf("the circuit lists its peering inline: %#v", got)
	}

	auth := c.expect(http.MethodPut, circuit+"/authorizations/branch"+xrTestAPI, `{}`, http.StatusCreated)
	if auth["properties"].(map[string]any)["authorizationUseStatus"] != "Available" {
		t.Fatalf("unexpected authorization: %#v", auth)
	}

	stats := c.expect(http.MethodGet, circuit+"/stats"+xrTestAPI, "", http.StatusOK)
	if _, ok := stats["primarybytesIn"]; !ok {
		t.Fatalf("unexpected stats: %#v", stats)
	}
	c.expect(http.MethodGet, circuit+"/peerings/AzurePrivatePeering/stats"+xrTestAPI, "", http.StatusOK)
	arp := azureTestList(t, c.expect(http.MethodPost, circuit+"/peerings/AzurePrivatePeering/arpTables/primary"+xrTestAPI, "", http.StatusOK))
	if len(arp) != 2 || arp[0].(map[string]any)["ipAddress"] != "192.168.10.1" {
		t.Fatalf("unexpected ARP table: %#v", arp)
	}
	summary := azureTestList(t, c.expect(http.MethodPost, circuit+"/peerings/AzurePrivatePeering/routeTablesSummary/secondary"+xrTestAPI, "", http.StatusOK))
	if summary[0].(map[string]any)["neighbor"] != "192.168.10.5" {
		t.Fatalf("unexpected route summary: %#v", summary)
	}
	c.expectError(http.MethodPost, circuit+"/peerings/AzurePrivatePeering/routeTables/tertiary"+xrTestAPI, "", http.StatusBadRequest, "InvalidRequest")
	c.expectError(http.MethodGet, base+"/expressRouteCircuits/missing/stats"+xrTestAPI, "", http.StatusNotFound, "ResourceNotFound")

	// Global Reach: a connection from hq's peering to branch's appears under branch as a peer connection.
	branch := base + "/expressRouteCircuits/branch"
	c.expect(http.MethodPut, branch+xrTestAPI, body, http.StatusCreated)
	c.expect(http.MethodPut, branch+"/peerings/AzurePrivatePeering"+xrTestAPI, peering, http.StatusCreated)
	hqPeering, branchPeering := strings.TrimPrefix(circuit, "/azure")+"/peerings/AzurePrivatePeering", strings.TrimPrefix(branch, "/azure")+"/peerings/AzurePrivatePeering"
	c.expect(http.MethodPut, circuit+"/peerings/AzurePrivatePeering/connections/to-branch"+xrTestAPI,
		`{"properties":{"expressRouteCircuitPeering":{"id":"`+hqPeering+`"},"peerExpressRouteCircuitPeering":{"id":"`+branchPeering+`"},"addressPrefix":"10.200.0.0/29"}}`, http.StatusCreated)
	peers := azureTestList(t, c.expect(http.MethodGet, branch+"/peerings/AzurePrivatePeering/peerConnections"+xrTestAPI, "", http.StatusOK))
	if len(peers) != 1 || peers[0].(map[string]any)["name"] != "to-branch" {
		t.Fatalf("unexpected peer connections: %#v", peers)
	}

	c.expectError(http.MethodPut, base+"/expressRouteCircuits/bad"+xrTestAPI, strings.Replace(body, `"bandwidthInMbps":200`, `"bandwidthInMbps":300`, 1), http.StatusBadRequest, "InvalidRequestContent")
	c.expectError(http.MethodPut, base+"/expressRouteCircuits/bad"+xrTestAPI, strings.Replace(body, "Standard_MeteredData", "Gold", 1), http.StatusBadRequest, "InvalidRequestContent")
}

func TestAzureExpressRoutePortsFiltersAndCatalogs(t *testing.T) {
	t.Parallel()
	c := newAzureARMTestClient(t, "rg-xrp")
	base := c.sub() + "/resourceGroups/rg-xrp/providers/Microsoft.Network"

	port := c.expect(http.MethodPut, base+"/ExpressRoutePorts/direct"+xrTestAPI,
		`{"location":"eastus","properties":{"peeringLocation":"Equinix-Ashburn-DC2","bandwidthInGbps":100,"encapsulation":"QinQ","links":[{"name":"link1","properties":{"adminState":"Enabled"}}]}}`, http.StatusCreated)
	links := port["properties"].(map[string]any)["links"].([]any)
	if len(links) != 2 || links[0].(map[string]any)["properties"].(map[string]any)["adminState"] != "Enabled" {
		t.Fatalf("unexpected port links: %#v", links)
	}
	if got := azureTestList(t, c.expect(http.MethodGet, base+"/ExpressRoutePorts/direct/links"+xrTestAPI, "", http.StatusOK)); len(got) != 2 {
		t.Fatalf("unexpected link list: %#v", got)
	}
	c.expect(http.MethodGet, base+"/ExpressRoutePorts/direct/links/link2"+xrTestAPI, "", http.StatusOK)
	loa := c.expect(http.MethodPost, base+"/ExpressRoutePorts/direct/generateLoa"+xrTestAPI, `{"customerName":"Contoso"}`, http.StatusOK)
	letter, _ := base64.StdEncoding.DecodeString(loa["encodedContent"].(string))
	if !strings.Contains(string(letter), "Contoso") {
		t.Fatalf("unexpected letter: %q", letter)
	}
	c.expectError(http.MethodPost, base+"/ExpressRoutePorts/direct/generateLoa"+xrTestAPI, `{}`, http.StatusBadRequest, "InvalidRequestContent")
	c.expectError(http.MethodPut, base+"/ExpressRoutePorts/bad"+xrTestAPI, `{"location":"eastus","properties":{"peeringLocation":"x","bandwidthInGbps":7}}`, http.StatusBadRequest, "InvalidRequestContent")
	c.expect(http.MethodPut, base+"/ExpressRoutePorts/direct/authorizations/tenant-b"+xrTestAPI, `{}`, http.StatusCreated)

	cross := c.expect(http.MethodPut, base+"/expressRouteCrossConnections/provider-1"+xrTestAPI, `{"location":"eastus","properties":{"peeringLocation":"Silicon Valley","bandwidthInMbps":1000}}`, http.StatusCreated)
	if cross["properties"].(map[string]any)["serviceProviderProvisioningState"] != "NotProvisioned" {
		t.Fatalf("unexpected cross connection: %#v", cross)
	}
	c.expectError(http.MethodDelete, base+"/expressRouteCrossConnections/provider-1"+xrTestAPI, "", http.StatusMethodNotAllowed, "MethodNotAllowed")

	c.expect(http.MethodPut, base+"/routeFilters/ms-peering"+xrTestAPI, `{"location":"eastus"}`, http.StatusCreated)
	c.expect(http.MethodPut, base+"/routeFilters/ms-peering/routeFilterRules/storage"+xrTestAPI, `{"properties":{"access":"Allow","routeFilterRuleType":"Community","communities":["12076:5010"]}}`, http.StatusCreated)
	c.expectError(http.MethodPut, base+"/routeFilters/ms-peering/routeFilterRules/bad"+xrTestAPI, `{"properties":{"access":"Allow","routeFilterRuleType":"Community","communities":["storage"]}}`, http.StatusBadRequest, "InvalidRequestContent")
	c.expect(http.MethodPut, base+"/serviceEndpointPolicies/storage-only"+xrTestAPI,
		`{"location":"eastus","properties":{"serviceEndpointPolicyDefinitions":[{"name":"allow-sub","properties":{"service":"Microsoft.Storage","serviceResources":["/subscriptions/`+azureResourcesTestSubscription+`"]}}]}}`, http.StatusCreated)
	if defs := azureTestList(t, c.expect(http.MethodGet, base+"/serviceEndpointPolicies/storage-only/serviceEndpointPolicyDefinitions"+xrTestAPI, "", http.StatusOK)); len(defs) != 1 {
		t.Fatalf("inline definitions are child resources too: %#v", defs)
	}

	for _, catalog := range []string{"expressRouteServiceProviders", "bgpServiceCommunities", "ExpressRoutePortsLocations", "expressRouteProviderPorts"} {
		if items := azureTestList(t, c.expect(http.MethodGet, c.sub()+"/providers/Microsoft.Network/"+catalog+xrTestAPI, "", http.StatusOK)); len(items) == 0 {
			t.Fatalf("expected %s entries", catalog)
		}
	}
	filtered := azureTestList(t, c.expect(http.MethodGet, c.sub()+"/providers/Microsoft.Network/expressRouteProviderPorts"+xrTestAPI+"&$filter=location%20eq%20'Washington%20DC'", "", http.StatusOK))
	if len(filtered) != 1 {
		t.Fatalf("the location filter narrows provider ports: %#v", filtered)
	}
}
