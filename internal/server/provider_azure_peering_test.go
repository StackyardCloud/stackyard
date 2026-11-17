package server

import (
	"net/http"
	"strings"
	"testing"
)

const azurePeeringTestAPIVersion = "?api-version=2025-05-01"

const azurePeerAsnBody = `{"properties":{"peerAsn":65000,"peerName":"Contoso","peerContactDetail":[{"role":"Noc","email":"noc@contoso.com","phone":"+1 (234) 567-8999"}]}}`

func azurePeeringDirectBody(asnID, connection string) string {
	return `{"kind":"Direct","location":"eastus","sku":{"name":"Basic_Direct_Free"},"properties":{"peeringLocation":"Seattle","direct":{"directPeeringType":"Edge","peerAsn":{"id":"` + asnID + `"},"connections":[` + connection + `]}}}`
}

const azurePeeringDirectConnection = `{"peeringDBFacilityId":71,"bandwidthInMbps":10000,"sessionAddressProvider":"Peer","useForPeeringService":true,"bgpSession":{"sessionPrefixV4":"10.21.31.0/31","sessionPrefixV6":"fe01::3e:0/127","maxPrefixesAdvertisedV4":1000,"maxPrefixesAdvertisedV6":100}}`

func TestAzurePeeringDirectAndExchange(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-peering")
	asn := sub + "/providers/Microsoft.Peering/peerAsns/contoso"
	asnID := strings.TrimPrefix(asn, "/azure")
	peering := sub + "/resourceGroups/rg-peering/providers/Microsoft.Peering/peerings/seattle-direct"
	// The ASN must be associated first.
	azureResourcesExpectErrorCode(t, do(http.MethodPut, peering+azurePeeringTestAPIVersion, azurePeeringDirectBody(asnID, azurePeeringDirectConnection)), http.StatusBadRequest, "InvalidRequestContent")
	created := providerContractJSONMap(t, do(http.MethodPut, asn+azurePeeringTestAPIVersion, azurePeerAsnBody))
	if created["properties"].(map[string]any)["validationState"] != "Approved" {
		t.Fatalf("the emulator approves ASNs at once: %#v", created)
	}

	body := providerContractJSONMap(t, do(http.MethodPut, peering+azurePeeringTestAPIVersion, azurePeeringDirectBody(asnID, azurePeeringDirectConnection)))
	sku := body["sku"].(map[string]any)
	direct := body["properties"].(map[string]any)["direct"].(map[string]any)
	conn := direct["connections"].([]any)[0].(map[string]any)
	session := conn["bgpSession"].(map[string]any)
	if sku["tier"] != "Basic" || sku["family"] != "Direct" || sku["size"] != "Free" {
		t.Fatalf("sku should carry tier, family and size: %#v", sku)
	}
	if conn["connectionState"] != "Active" || conn["provisionedBandwidthInMbps"] != float64(10000) || direct["useForPeeringService"] != true || direct["peerAsn"] == nil {
		t.Fatalf("unexpected connection: %#v", direct)
	}
	if session["microsoftSessionIPv4Address"] != "10.21.31.0" || session["peerSessionIPv4Address"] != "10.21.31.1" || session["sessionStateV6"] != "Established" {
		t.Fatalf("unexpected session: %#v", session)
	}
	// PATCH changes tags only.
	patched := providerContractJSONMap(t, do(http.MethodPatch, peering+azurePeeringTestAPIVersion, `{"tags":{"env":"prod"},"properties":{"peeringLocation":"Ashburn"}}`))
	if patched["tags"].(map[string]any)["env"] != "prod" || patched["properties"].(map[string]any)["peeringLocation"] != "Seattle" {
		t.Fatalf("unexpected patched peering: %#v", patched)
	}
	untagged := providerContractJSONMap(t, do(http.MethodPatch, peering+azurePeeringTestAPIVersion, `{}`))
	if untagged["tags"].(map[string]any)["env"] != "prod" {
		t.Fatalf("a PATCH without tags should keep them: %#v", untagged)
	}

	exchange := sub + "/resourceGroups/rg-peering/providers/Microsoft.Peering/peerings/seattle-ix"
	ex := providerContractJSONMap(t, do(http.MethodPut, exchange+azurePeeringTestAPIVersion, `{"kind":"Exchange","location":"eastus","sku":{"name":"Basic_Exchange_Free"},"properties":{"peeringLocation":"Seattle","exchange":{"peerAsn":{"id":"`+asnID+`"},"connections":[{"peeringDBFacilityId":11,"bgpSession":{"peerSessionIPv4Address":"198.32.134.22","peerSessionIPv6Address":"2001:504:12::22"}}]}}}`))
	exSession := ex["properties"].(map[string]any)["exchange"].(map[string]any)["connections"].([]any)[0].(map[string]any)["bgpSession"].(map[string]any)
	if exSession["microsoftSessionIPv4Address"] != "198.32.134.152" || exSession["microsoftSessionIPv6Address"] != "2001:504:12::15" {
		t.Fatalf("Exchange sessions should use the exchange's Microsoft addresses: %#v", exSession)
	}

	for name, conn := range map[string]string{
		"facility":    strings.Replace(azurePeeringDirectConnection, `"peeringDBFacilityId":71`, `"peeringDBFacilityId":72`, 1),
		"bandwidth":   strings.Replace(azurePeeringDirectConnection, `"bandwidthInMbps":10000`, `"bandwidthInMbps":15000`, 1),
		"prefix v4":   strings.Replace(azurePeeringDirectConnection, `10.21.31.0/31`, `10.21.31.0/30`, 1),
		"prefix v6":   strings.Replace(azurePeeringDirectConnection, `fe01::3e:0/127`, `10.0.0.0/31`, 1),
		"no session":  `{"peeringDBFacilityId":71,"bandwidthInMbps":10000,"sessionAddressProvider":"Peer"}`,
		"provider":    strings.Replace(azurePeeringDirectConnection, `"Peer"`, `"Customer"`, 1),
		"max prefix":  strings.Replace(azurePeeringDirectConnection, `"maxPrefixesAdvertisedV4":1000`, `"maxPrefixesAdvertisedV4":-1`, 1),
		"no connects": ``,
	} {
		resp := do(http.MethodPut, peering+azurePeeringTestAPIVersion, azurePeeringDirectBody(asnID, conn))
		t.Run("direct "+name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	direct2 := azurePeeringDirectBody(asnID, azurePeeringDirectConnection)
	for name, payload := range map[string]string{
		"kind":     strings.Replace(direct2, `"kind":"Direct"`, `"kind":"Transit"`, 1),
		"sku":      strings.Replace(direct2, `Basic_Direct_Free`, `Basic_Exchange_Free`, 1),
		"location": strings.Replace(direct2, `"Seattle"`, `"Atlantis"`, 1),
		"type":     strings.Replace(direct2, `"Edge"`, `"Backbone"`, 1),
		"asn":      strings.Replace(direct2, asnID, "/subscriptions/x/providers/Microsoft.Peering/peerAsns/other", 1),
		"exchange ip": `{"kind":"Exchange","location":"eastus","sku":{"name":"Basic_Exchange_Free"},"properties":{"peeringLocation":"Seattle","exchange":{"peerAsn":{"id":"` + asnID +
			`"},"connections":[{"peeringDBFacilityId":11,"bgpSession":{"peerSessionIPv4Address":"10.0.0.1"}}]}}}`,
	} {
		resp := do(http.MethodPut, peering+azurePeeringTestAPIVersion, payload)
		t.Run("peering "+name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, asn+"2"+azurePeeringTestAPIVersion, `{"properties":{"peerAsn":65000,"peerName":"Contoso","peerContactDetail":[{"role":"Boss","email":"a@b.c"}]}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, asn+"2"+azurePeeringTestAPIVersion, `{"properties":{"peerAsn":0,"peerName":"Contoso","peerContactDetail":[{"role":"Noc","email":"a@b.c"}]}}`), http.StatusBadRequest, "InvalidRequestContent")

	locations := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/Microsoft.Peering/peeringLocations"+azurePeeringTestAPIVersion+"&kind=Direct", ""))["value"].([]any)
	if len(locations) != 2 {
		t.Fatalf("expected Seattle for each kind, got %#v", locations)
	}
	if routes := providerContractJSONMap(t, do(http.MethodGet, peering+"/receivedRoutes"+azurePeeringTestAPIVersion, ""))["value"].([]any); len(routes) != 0 {
		t.Fatalf("expected no received routes, got %#v", routes)
	}
}

func TestAzurePeeringServicePrefixes(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-peering")
	asnID := strings.TrimPrefix(sub, "/azure") + "/providers/Microsoft.Peering/peerAsns/contoso"
	azureResourcesExpectStatus(t, do(http.MethodPut, "/azure"+asnID+azurePeeringTestAPIVersion, azurePeerAsnBody), http.StatusCreated)
	peering := sub + "/resourceGroups/rg-peering/providers/Microsoft.Peering/peerings/seattle-direct"
	azureResourcesExpectStatus(t, do(http.MethodPut, peering+azurePeeringTestAPIVersion, azurePeeringDirectBody(asnID, azurePeeringDirectConnection)), http.StatusCreated)

	// The provider registers a prefix, whose key the customer quotes.
	registered := providerContractJSONMap(t, do(http.MethodPut, peering+"/registeredPrefixes/customer1"+azurePeeringTestAPIVersion, `{"properties":{"prefix":"192.168.1.0/24"}}`))
	key := registered["properties"].(map[string]any)["peeringServicePrefixKey"].(string)
	if registered["properties"].(map[string]any)["prefixValidationState"] != "Verified" || key == "" {
		t.Fatalf("unexpected registered prefix: %#v", registered)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, peering+"/registeredPrefixes/customer1/validate"+azurePeeringTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, peering+"/registeredPrefixes/bad"+azurePeeringTestAPIVersion, `{"properties":{"prefix":"192.168.1.1/24"}}`), http.StatusBadRequest, "InvalidRequestContent")
	asnRec := providerContractJSONMap(t, do(http.MethodPut, peering+"/registeredAsns/customer1"+azurePeeringTestAPIVersion, `{"properties":{"asn":65010}}`))
	if asnRec["properties"].(map[string]any)["peeringServicePrefixKey"] == "" {
		t.Fatalf("registered ASN should have a key: %#v", asnRec)
	}

	service := sub + "/resourceGroups/rg-peering/providers/Microsoft.Peering/peeringServices/branch"
	valid := `{"location":"westus2","properties":{"peeringServiceLocation":"Washington","peeringServiceProvider":"Contoso Networks","providerPrimaryPeeringLocation":"Seattle"}}`
	azureResourcesExpectStatus(t, do(http.MethodPut, service+azurePeeringTestAPIVersion, valid), http.StatusCreated)
	for name, payload := range map[string]string{
		"location":    strings.Replace(valid, "Washington", "Atlantis", 1),
		"provider":    strings.Replace(valid, "Contoso Networks", "Fabrikam", 1),
		"unavailable": strings.Replace(valid, "Washington", "Ontario", 1),
		"peering":     strings.Replace(valid, `"providerPrimaryPeeringLocation":"Seattle"`, `"providerPrimaryPeeringLocation":"Ashburn"`, 1),
		"same backup": strings.Replace(valid, `"providerPrimaryPeeringLocation":"Seattle"`, `"providerPrimaryPeeringLocation":"Seattle","providerBackupPeeringLocation":"Seattle"`, 1),
	} {
		resp := do(http.MethodPut, service+azurePeeringTestAPIVersion, payload)
		t.Run("service "+name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	verified := providerContractJSONMap(t, do(http.MethodPut, service+"/prefixes/office"+azurePeeringTestAPIVersion, `{"properties":{"prefix":"192.168.1.0/24","peeringServicePrefixKey":"`+key+`"}}`))
	if p := verified["properties"].(map[string]any); p["prefixValidationState"] != "Verified" || p["learnedType"] != "ViaServiceProvider" {
		t.Fatalf("a prefix with the provider's key should be verified: %#v", p)
	}
	failed := providerContractJSONMap(t, do(http.MethodPut, service+"/prefixes/other"+azurePeeringTestAPIVersion, `{"properties":{"prefix":"192.168.2.0/24","peeringServicePrefixKey":"`+key+`"}}`))
	if p := failed["properties"].(map[string]any); p["prefixValidationState"] != "Failed" || p["errorMessage"] != "Prefix is not announced by the service provider to Microsoft." {
		t.Fatalf("a prefix nobody registered should fail: %#v", p)
	}

	availability := sub + "/providers/Microsoft.Peering/checkServiceProviderAvailability" + azurePeeringTestAPIVersion
	for location, want := range map[string]string{"Washington": `"Available"`, "Ontario": `"Unavailable"`} {
		resp := do(http.MethodPost, availability, `{"peeringServiceLocation":"`+location+`","peeringServiceProvider":"Contoso Networks"}`)
		if got := strings.TrimSpace(string(providerContractBody(t, resp))); got != want {
			t.Fatalf("availability in %s = %s, want %s", location, got, want)
		}
	}
	monitor := service + "/connectionMonitorTests/web" + azurePeeringTestAPIVersion
	azureResourcesExpectStatus(t, do(http.MethodPut, monitor, `{"properties":{"sourceAgent":"agent1","destination":"10.0.0.1","destinationPort":443,"testFrequencyInSec":30}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, monitor, `{"properties":{"sourceAgent":"agent1","destination":"10.0.0.1","destinationPort":70000}}`), http.StatusBadRequest, "InvalidRequestContent")
	if got := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/Microsoft.Peering/peeringServiceProviders"+azurePeeringTestAPIVersion, ""))["value"].([]any); len(got) != 1 {
		t.Fatalf("expected one provider, got %#v", got)
	}
}
