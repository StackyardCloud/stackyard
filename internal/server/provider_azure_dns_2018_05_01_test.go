package server

import (
	"net/http"
	"strings"
	"testing"
)

const adnsTestAPIVersion = "?api-version=2018-05-01"

func adnsTestValue(t *testing.T, resp *http.Response) []any {
	t.Helper()
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	items, _ := providerContractJSONMap(t, resp)["value"].([]any)
	return items
}

func TestAzureDNSZoneAndRecordSetLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dns")
	zone := sub + "/resourceGroups/rg-dns/providers/Microsoft.Network/dnsZones/contoso.com"

	resp := do(http.MethodPut, zone+adnsTestAPIVersion, `{"location":"global","tags":{"env":"dev"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	body := providerContractJSONMap(t, resp)
	props := body["properties"].(map[string]any)
	if servers, _ := props["nameServers"].([]any); len(servers) != 4 || body["etag"] == nil || props["zoneType"] != "Public" {
		t.Fatalf("unexpected zone: %#v", body)
	}

	// The apex SOA and NS record sets come with the zone.
	resp = do(http.MethodGet, zone+"/SOA/@"+adnsTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if soa := providerContractJSONMap(t, resp)["properties"].(map[string]any); soa["fqdn"] != "contoso.com." || soa["SOARecord"] == nil {
		t.Fatalf("unexpected SOA record set: %#v", soa)
	}
	if ns := adnsTestValue(t, do(http.MethodGet, zone+"/NS"+adnsTestAPIVersion, "")); len(ns) != 1 {
		t.Fatalf("expected the apex NS record set, got %#v", ns)
	}

	resp = do(http.MethodPut, zone+"/A/www"+adnsTestAPIVersion, `{"properties":{"TTL":300,"ARecords":[{"ipv4Address":"10.1.2.3"}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	rs := providerContractJSONMap(t, resp)
	if rs["type"] != "Microsoft.Network/dnsZones/A" || rs["properties"].(map[string]any)["fqdn"] != "www.contoso.com." {
		t.Fatalf("unexpected record set: %#v", rs)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, zone+"/TXT/_verify"+adnsTestAPIVersion, `{"properties":{"TTL":60,"TXTRecords":[{"value":["token"]}]}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, zone+"/A/www"+adnsTestAPIVersion, `{"properties":{"TTL":600,"ARecords":[{"ipv4Address":"10.1.2.4"}]}}`), http.StatusOK)

	if all := adnsTestValue(t, do(http.MethodGet, zone+"/recordsets"+adnsTestAPIVersion, "")); len(all) != 4 {
		t.Fatalf("expected 4 record sets, got %d", len(all))
	}
	if all := adnsTestValue(t, do(http.MethodGet, zone+"/all"+adnsTestAPIVersion, "")); len(all) != 4 {
		t.Fatalf("expected 4 record sets in /all, got %d", len(all))
	}
	if a := adnsTestValue(t, do(http.MethodGet, zone+"/A"+adnsTestAPIVersion, "")); len(a) != 1 {
		t.Fatalf("expected 1 A record set, got %d", len(a))
	}
	resp = do(http.MethodGet, zone+adnsTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if n := providerContractJSONMap(t, resp)["properties"].(map[string]any)["numberOfRecordSets"]; n != float64(4) {
		t.Fatalf("expected numberOfRecordSets 4, got %v", n)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, zone+"/A/www"+adnsTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, zone+"/A/www"+adnsTestAPIVersion, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodDelete, zone+adnsTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, zone+"/TXT/_verify"+adnsTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureDNSValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dns")
	zones := sub + "/resourceGroups/rg-dns/providers/Microsoft.Network/dnsZones/"
	zone := zones + "fabrikam.net"

	cases := []struct{ name, path, body, code string }{
		{"zone outside global", zones + "example.org", `{"location":"eastus"}`, "InvalidRequestContent"},
		{"single label zone", zones + "localhost", `{"location":"global"}`, "InvalidResourceName"},
		{"private zone type", zones + "example.org", `{"location":"global","properties":{"zoneType":"Private"}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+adnsTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, zone+adnsTestAPIVersion, `{"location":"global"}`), http.StatusCreated)
	records := []struct{ name, path, body, code string }{
		{"invalid ipv4", "/A/web", `{"properties":{"ARecords":[{"ipv4Address":"10.0.0.999"}]}}`, "InvalidRequestContent"},
		{"ipv6 in A", "/A/web", `{"properties":{"ARecords":[{"ipv4Address":"::1"}]}}`, "InvalidRequestContent"},
		{"wrong member", "/A/web", `{"properties":{"CNAMERecord":{"cname":"x.example.com"}}}`, "InvalidRequestContent"},
		{"negative ttl", "/A/web", `{"properties":{"TTL":-1}}`, "InvalidRequestContent"},
		{"cname at apex", "/CNAME/@", `{"properties":{"CNAMERecord":{"cname":"x.example.com"}}}`, "InvalidResourceName"},
		{"soa off apex", "/SOA/web", `{"properties":{}}`, "InvalidResourceName"},
		{"bad caa tag", "/CAA/@", `{"properties":{"caaRecords":[{"flags":0,"tag":"nope","value":"ca"}]}}`, "InvalidRequestContent"},
		{"mx without exchange", "/MX/@", `{"properties":{"MXRecords":[{"preference":10}]}}`, "InvalidRequestContent"},
	}
	for _, tc := range records {
		resp := do(http.MethodPut, zone+tc.path+adnsTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
	tooMany := `{"properties":{"ARecords":[` + strings.TrimSuffix(strings.Repeat(`{"ipv4Address":"10.0.0.1"},`, adnsMaxRecordsPerSet+1), ",") + `]}}`
	azureResourcesExpectErrorCode(t, do(http.MethodPut, zone+"/A/web"+adnsTestAPIVersion, tooMany), http.StatusBadRequest, "InvalidRequestContent")

	// The apex SOA and NS record sets cannot be removed.
	azureResourcesExpectStatus(t, do(http.MethodDelete, zone+"/SOA/@"+adnsTestAPIVersion, ""), http.StatusConflict)
	azureResourcesExpectStatus(t, do(http.MethodDelete, zone+"/NS/@"+adnsTestAPIVersion, ""), http.StatusConflict)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, zones+"missing.com/recordsets"+adnsTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, zones+"missing.com/A/web"+adnsTestAPIVersion, `{"properties":{}}`), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureDNSResourceReference(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dns")
	zone := sub + "/resourceGroups/rg-dns/providers/Microsoft.Network/dnsZones/contoso.com"
	target := "/subscriptions/" + azureResourcesTestSubscription + "/resourceGroups/rg-dns/providers/Microsoft.Network/publicIPAddresses/pip"
	reference := sub + "/providers/Microsoft.Network/getDnsResourceReference" + adnsTestAPIVersion

	azureResourcesExpectStatus(t, do(http.MethodPut, zone+adnsTestAPIVersion, `{"location":"global"}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, zone+"/A/alias"+adnsTestAPIVersion, `{"properties":{"TTL":60,"targetResource":{"id":"`+target+`"}}}`), http.StatusCreated)

	resp := do(http.MethodPost, reference, `{"properties":{"targetResources":[{"id":"`+target+`"}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	refs := providerContractJSONMap(t, resp)["properties"].(map[string]any)["dnsResourceReferences"].([]any)
	resources := refs[0].(map[string]any)["dnsResources"].([]any)
	if len(resources) != 1 || !strings.HasSuffix(resources[0].(map[string]any)["id"].(string), "/dnsZones/contoso.com/A/alias") {
		t.Fatalf("unexpected references: %#v", refs)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, reference, `{"properties":{}}`), http.StatusBadRequest, "InvalidRequestContent")
}
