package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

type azureNCWTest struct {
	t    *testing.T
	base string // the resource group's Microsoft.NetworkCloud path
	arm  string // base without /azure, as resource IDs have it
	do   func(method, path, body string) *http.Response
}

func newAzureNCWTest(t *testing.T) *azureNCWTest {
	t.Helper()
	sub, do := newAzureDatabaseTestServer(t, "rg-nc")
	base := sub + "/resourceGroups/rg-nc/providers/Microsoft.NetworkCloud/"
	return &azureNCWTest{t: t, base: base, arm: strings.TrimPrefix(base, "/azure"), do: do}
}

func (h *azureNCWTest) call(method, path string, body any, want int) map[string]any {
	h.t.Helper()
	text := ""
	if body != nil {
		raw, _ := json.Marshal(body)
		text = string(raw)
	}
	resp := h.do(method, h.base+path+"?api-version=2026-07-01", text)
	raw := mustBody(h.t, resp)
	if resp.StatusCode != want {
		h.t.Fatalf("%s %s = %d, want %d: %s", method, path, resp.StatusCode, want, raw)
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func (h *azureNCWTest) props(method, path string, body any, want int) map[string]any {
	h.t.Helper()
	return azureARMObject(h.call(method, path, body, want)["properties"])
}

func (h *azureNCWTest) fails(method, path string, body any, status int, text string) {
	h.t.Helper()
	out := h.call(method, path, body, status)
	if msg := azureARMString(azureARMObject(out["error"])["message"]); !strings.Contains(msg, text) {
		h.t.Fatalf("%s %s error = %q, want it to mention %q", method, path, msg, text)
	}
}

var azureNCWTestLocation = map[string]any{"type": "CustomLocation", "name": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-nc/providers/Microsoft.ExtendedLocation/customLocations/cl1"}

// body returns a tracked resource body with the custom location.
func azureNCWBody(props map[string]any) map[string]any {
	body := map[string]any{"location": "eastus", "extendedLocation": azureNCWTestLocation}
	if props != nil {
		body["properties"] = props
	}
	return body
}

const azureNCWTestDomain = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-nc/providers/Microsoft.ManagedNetworkFabric/l3IsolationDomains/l3d"

// networks creates a cloud services network, L2, L3 and trunked network
// and a volume named csn, l2, l3, trunk and vol.
func (h *azureNCWTest) networks() {
	h.t.Helper()
	h.call(http.MethodPut, "cloudServicesNetworks/csn", azureNCWBody(nil), http.StatusCreated)
	h.call(http.MethodPut, "l2Networks/l2", azureNCWBody(map[string]any{"l2IsolationDomainId": azureNCWTestDomain}), http.StatusCreated)
	h.call(http.MethodPut, "l3Networks/l3", azureNCWBody(map[string]any{"l3IsolationDomainId": azureNCWTestDomain, "vlan": 100, "ipv4ConnectedPrefix": "198.51.100.0/24", "ipv6ConnectedPrefix": "2001:db8::/64"}), http.StatusCreated)
	h.call(http.MethodPut, "trunkedNetworks/trunk", azureNCWBody(map[string]any{"isolationDomainIds": []any{azureNCWTestDomain}, "vlans": []any{10, 11}}), http.StatusCreated)
	h.call(http.MethodPut, "volumes/vol", azureNCWBody(map[string]any{"sizeMiB": 10240}), http.StatusCreated)
}

func TestAzureNetworkCloudNetworks(t *testing.T) {
	h := newAzureNCWTest(t)
	h.networks()

	l3 := h.call(http.MethodGet, "l3Networks/l3", nil, http.StatusOK)
	props := azureARMObject(l3["properties"])
	if props["ipAllocationType"] != "DualStack" || props["hybridAksPluginType"] != "SRIOV" || props["detailedStatus"] != "Available" || l3["etag"] == nil {
		t.Fatalf("l3 = %v", l3)
	}
	if azureARMObject(l3["extendedLocation"])["type"] != "CustomLocation" {
		t.Fatalf("extendedLocation = %v", l3["extendedLocation"])
	}
	if v := h.props(http.MethodGet, "volumes/vol", nil, http.StatusOK); v["allocatedSizeMiB"] != 10240.0 || v["detailedStatus"] != "Active" || v["serialNumber"] == "" {
		t.Fatalf("volume = %v", v)
	}
	if csn := h.props(http.MethodGet, "cloudServicesNetworks/csn", nil, http.StatusOK); csn["enableDefaultEgressEndpoints"] != "True" || azureARMObject(csn["storageStatus"])["mode"] != "None" {
		t.Fatalf("csn = %v", csn)
	}

	bad := []struct{ path, text string }{
		{"l2Networks/l2x", "extendedLocation is required"},
		{"l2Networks/-bad", "name"},
		{"l3Networks/l3x", "ipv6ConnectedPrefix is required"},
		{"trunkedNetworks/tx", "vlans"},
		{"volumes/vx", "sizeMiB"},
	}
	bodies := []map[string]any{
		{"location": "eastus", "properties": map[string]any{"l2IsolationDomainId": azureNCWTestDomain}},
		azureNCWBody(map[string]any{"l2IsolationDomainId": azureNCWTestDomain}),
		azureNCWBody(map[string]any{"l3IsolationDomainId": azureNCWTestDomain, "vlan": 1, "ipv4ConnectedPrefix": "198.51.100.0/24"}),
		azureNCWBody(map[string]any{"isolationDomainIds": []any{azureNCWTestDomain}, "vlans": []any{}}),
		azureNCWBody(map[string]any{"sizeMiB": 0}),
	}
	for i, c := range bad {
		h.fails(http.MethodPut, c.path, bodies[i], http.StatusBadRequest, c.text)
	}
	h.fails(http.MethodPut, "l3Networks/l3y", azureNCWBody(map[string]any{"l3IsolationDomainId": azureNCWTestDomain, "vlan": 1, "ipAllocationType": "IPV4", "ipv4ConnectedPrefix": "2001:db8::/64"}),
		http.StatusBadRequest, "IPV4 CIDR")
	h.fails(http.MethodPut, "l2Networks/l2y", azureNCWBody(map[string]any{"l2IsolationDomainId": azureNCWTestDomain, "interfaceName": "thirteen-char"}), http.StatusBadRequest, "interfaceName")
	h.fails(http.MethodPut, "l2Networks/l2z", map[string]any{"location": "eastus", "extendedLocation": map[string]any{"type": "CustomLocation", "name": "cl1"}, "properties": map[string]any{"l2IsolationDomainId": azureNCWTestDomain}},
		http.StatusBadRequest, "custom location ID")
	h.fails(http.MethodPut, "cloudServicesNetworks/csn2", azureNCWBody(map[string]any{"additionalEgressEndpoints": []any{map[string]any{"category": "c", "endpoints": []any{map[string]any{"domainName": "x", "port": 70000}}}}}),
		http.StatusBadRequest, "port")

	// Only tags change through PATCH; create-only properties stay put.
	h.call(http.MethodPatch, "l3Networks/l3", map[string]any{"tags": map[string]any{"env": "test"}}, http.StatusOK)
	h.fails(http.MethodPatch, "l3Networks/l3", map[string]any{"properties": map[string]any{"vlan": 5}}, http.StatusBadRequest, "can't be updated")
	h.fails(http.MethodPut, "l3Networks/l3", azureNCWBody(map[string]any{"l3IsolationDomainId": azureNCWTestDomain, "vlan": 101, "ipv4ConnectedPrefix": "198.51.100.0/24", "ipv6ConnectedPrefix": "2001:db8::/64"}),
		http.StatusBadRequest, "vlan can't be changed")
	moved := azureNCWBody(map[string]any{"l2IsolationDomainId": azureNCWTestDomain})
	moved["extendedLocation"] = map[string]any{"type": "CustomLocation", "name": strings.Replace(azureNCWTestLocation["name"].(string), "cl1", "cl2", 1)}
	h.fails(http.MethodPut, "l2Networks/l2", moved, http.StatusBadRequest, "extendedLocation can't be changed")
	if p := h.props(http.MethodPut, "l2Networks/l2", azureNCWBody(map[string]any{"l2IsolationDomainId": azureNCWTestDomain, "associatedResourceIds": []any{"forged"}}), http.StatusOK); p["hybridAksPluginType"] != "SRIOV" || len(azureARMSlice(p["associatedResourceIds"])) != 0 {
		t.Fatalf("re-PUT l2 = %v", p)
	}
	csn := h.props(http.MethodPatch, "cloudServicesNetworks/csn", map[string]any{"properties": map[string]any{
		"storageOptions":            map[string]any{"mode": "Standard", "sizeMiB": 2048},
		"additionalEgressEndpoints": []any{map[string]any{"category": "updates", "endpoints": []any{map[string]any{"domainName": "example.com", "port": 443}}}},
	}}, http.StatusOK)
	if azureARMObject(csn["storageStatus"])["status"] != "Available" || len(azureARMSlice(csn["enabledEgressEndpoints"])) != 1 {
		t.Fatalf("patched csn = %v", csn)
	}
}
