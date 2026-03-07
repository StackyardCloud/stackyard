package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPVMwareEngineRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPVMwareEngineContractServer(t)

	parent := "/gcp/v1/projects/stackyard/locations/us-central1"
	globalParent := "/gcp/v1/projects/stackyard/locations/global"
	privateCloud := parent + "/privateClouds/private-cloud-1"
	cluster := privateCloud + "/clusters/cluster-1"
	networkPolicy := parent + "/networkPolicies/network-policy-1"
	vmwareEngineNetwork := parent + "/vmwareEngineNetworks/vmware-engine-network-1"
	privateConnection := parent + "/privateConnections/private-connection-1"
	globalPeering := globalParent + "/networkPeerings/network-peering-1"
	managementDNSZoneBinding := parent + "/managementDnsZoneBindings/management-dns-zone-binding-1"
	dnsBindPermission := globalParent + "/networkPolicies/network-policy-1/dnsBindPermission"

	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, parent+"/privateClouds?pageSize=1", nil, "privateClouds")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, privateCloud, nil, "/privateClouds/private-cloud-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, parent+"/privateClouds?privateCloudId=private-cloud-1", []byte(`{"privateCloud":{"description":"pc"}}`), "operations/create.privateClouds.private-cloud-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPatch, privateCloud+"?updateMask=description", []byte(`{"privateCloud":{"name":"projects/stackyard/locations/us-central1/privateClouds/private-cloud-1","description":"updated"}}`), "operations/update.privateClouds.private-cloud-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodDelete, privateCloud, nil, "operations/delete.privateClouds.private-cloud-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, privateCloud+":undelete", []byte(`{}`), "operations/undelete.private-cloud-1")

	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, privateCloud+"/clusters?pageSize=1", nil, "clusters")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, cluster, nil, "/clusters/cluster-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, privateCloud+"/clusters?clusterId=cluster-1", []byte(`{"cluster":{"description":"cluster"}}`), "operations/create.clusters.cluster-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, cluster+"/nodes?pageSize=1", nil, "nodes")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, cluster+"/nodes/node-1", nil, "/nodes/node-1")

	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, networkPolicy+"/externalAddresses?pageSize=1", nil, "externalAddresses")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, networkPolicy+"/externalAddresses/external-address-1", nil, "/externalAddresses/external-address-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, networkPolicy+"/externalAddresses?externalAddressId=external-address-1", []byte(`{"externalAddress":{"description":"ea"}}`), "operations/create.externalAddresses.external-address-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPatch, networkPolicy+"/externalAddresses/external-address-1?updateMask=description", []byte(`{"externalAddress":{"name":"projects/stackyard/locations/us-central1/networkPolicies/network-policy-1/externalAddresses/external-address-1","description":"ea"}}`), "operations/update.externalAddresses.external-address-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodDelete, networkPolicy+"/externalAddresses/external-address-1", nil, "operations/delete.externalAddresses.external-address-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, networkPolicy+":fetchNetworkPolicyExternalAddresses", []byte(`{}`), "externalAddresses")

	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, parent+"/networkPolicies?pageSize=1", nil, "networkPolicies")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, networkPolicy, nil, "/networkPolicies/network-policy-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, parent+"/networkPolicies?networkPolicyId=network-policy-1", []byte(`{"networkPolicy":{"description":"np"}}`), "operations/create.networkPolicies.network-policy-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPatch, networkPolicy+"?updateMask=description", []byte(`{"networkPolicy":{"name":"projects/stackyard/locations/us-central1/networkPolicies/network-policy-1","description":"np"}}`), "operations/update.networkPolicies.network-policy-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodDelete, networkPolicy, nil, "operations/delete.networkPolicies.network-policy-1")

	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, globalParent+"/networkPeerings?pageSize=1", nil, "networkPeerings")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, globalPeering, nil, "/networkPeerings/network-peering-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, globalParent+"/networkPeerings?networkPeeringId=network-peering-1", []byte(`{"networkPeering":{"description":"np"}}`), "operations/create.networkPeerings.network-peering-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPatch, globalPeering+"?updateMask=description", []byte(`{"networkPeering":{"name":"projects/stackyard/locations/global/networkPeerings/network-peering-1","description":"np"}}`), "operations/update.networkPeerings.network-peering-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodDelete, globalPeering, nil, "operations/delete.networkPeerings.network-peering-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, globalPeering+"/peeringRoutes?pageSize=1", nil, "peeringRoutes")

	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, parent+"/vmwareEngineNetworks?pageSize=1", nil, "vmwareEngineNetworks")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, vmwareEngineNetwork, nil, "/vmwareEngineNetworks/vmware-engine-network-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, parent+"/vmwareEngineNetworks?vmwareEngineNetworkId=vmware-engine-network-1", []byte(`{"vmwareEngineNetwork":{"description":"ven"}}`), "operations/create.vmwareEngineNetworks.vmware-engine-network-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPatch, vmwareEngineNetwork+"?updateMask=description", []byte(`{"vmwareEngineNetwork":{"name":"projects/stackyard/locations/us-central1/vmwareEngineNetworks/vmware-engine-network-1","description":"ven"}}`), "operations/update.vmwareEngineNetworks.vmware-engine-network-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodDelete, vmwareEngineNetwork, nil, "operations/delete.vmwareEngineNetworks.vmware-engine-network-1")

	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, parent+"/privateConnections?pageSize=1", nil, "privateConnections")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, privateConnection, nil, "/privateConnections/private-connection-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, parent+"/privateConnections?privateConnectionId=private-connection-1", []byte(`{"privateConnection":{"description":"pc"}}`), "operations/create.privateConnections.private-connection-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPatch, privateConnection+"?updateMask=description", []byte(`{"privateConnection":{"name":"projects/stackyard/locations/us-central1/privateConnections/private-connection-1","description":"pc"}}`), "operations/update.privateConnections.private-connection-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodDelete, privateConnection, nil, "operations/delete.privateConnections.private-connection-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, privateConnection+"/peeringRoutes?pageSize=1", nil, "peeringRoutes")

	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, privateCloud+":showNsxCredentials", nil, "username")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, privateCloud+":resetNsxCredentials", []byte(`{}`), "operations/resetNsxCredentials.private-cloud-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, privateCloud+"/dnsForwarding", nil, "forwardingRules")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPatch, privateCloud+"/dnsForwarding?updateMask=forwarding_rules", []byte(`{"dnsForwarding":{"name":"projects/stackyard/locations/us-central1/privateClouds/private-cloud-1/dnsForwarding"}}`), "operations/update.dnsForwarding")

	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, parent+"/managementDnsZoneBindings?pageSize=1", nil, "managementDnsZoneBindings")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, managementDNSZoneBinding, nil, "/managementDnsZoneBindings/management-dns-zone-binding-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, parent+"/managementDnsZoneBindings?managementDnsZoneBindingId=management-dns-zone-binding-1", []byte(`{"managementDnsZoneBinding":{"description":"mdzb"}}`), "operations/create.managementDnsZoneBindings.management-dns-zone-binding-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, managementDNSZoneBinding+":repair", []byte(`{}`), "operations/repair.management-dns-zone-binding-1")

	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, dnsBindPermission, nil, "principals")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, dnsBindPermission+":grant", []byte(`{}`), "operations/grant.network-policy-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, dnsBindPermission+":revoke", []byte(`{}`), "operations/revoke.network-policy-1")

	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, parent+"/operations?pageSize=1", nil, "operations")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodGet, parent+"/operations/vmwareengine-op-1", nil, "operations/vmwareengine-op-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, parent+"/operations/vmwareengine-op-1:cancel", []byte(`{}`), `{}`)
	assertGCPVMwareEngineSuccess(t, ts, http.MethodDelete, parent+"/operations/vmwareengine-op-1", nil, `{}`)

	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vmwareengine.v1.VmwareEngine/ListPrivateClouds", []byte(`{"parent":"projects/stackyard/locations/us-central1","pageSize":1}`), "privateClouds")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vmwareengine.v1.VmwareEngine/GetPrivateCloud", []byte(`{"name":"projects/stackyard/locations/us-central1/privateClouds/private-cloud-1"}`), "/privateClouds/private-cloud-1")
	assertGCPVMwareEngineSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vmwareengine.v1.VmwareEngine/CreatePrivateCloud", []byte(`{"parent":"projects/stackyard/locations/us-central1","privateCloudId":"private-cloud-1","privateCloud":{"description":"pc"}}`), "operations/create.privateClouds.private-cloud-1")
}

func TestGCPVMwareEngineRouter_ListPrivateCloudsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPVMwareEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/privateClouds?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "vmwareengine",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vmwareengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVMwareEngineRouter_CreatePrivateCloudRequiresBody(t *testing.T) {
	t.Parallel()

	ts := newGCPVMwareEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/privateClouds?privateCloudId=private-cloud-1", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vmwareengine",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vmwareengine create private cloud, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVMwareEngineRouter_UpdatePrivateCloudRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPVMwareEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/privateClouds/private-cloud-1", []byte(`{"privateCloud":{"name":"projects/stackyard/locations/us-central1/privateClouds/private-cloud-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vmwareengine",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vmwareengine update private cloud, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVMwareEngineRouter_NetworkPeeringsRequireGlobalLocation(t *testing.T) {
	t.Parallel()

	ts := newGCPVMwareEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/networkPeerings?networkPeeringId=network-peering-1", []byte(`{"networkPeering":{"description":"np"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vmwareengine",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vmwareengine create network peering, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVMwareEngineRouter_DNSBindPermissionActionRequiresGlobalLocation(t *testing.T) {
	t.Parallel()

	ts := newGCPVMwareEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/networkPolicies/network-policy-1/dnsBindPermission:grant", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vmwareengine",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vmwareengine grant dns bind permission, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVMwareEngineRouter_GRPCBridgeCreatePrivateCloudRequiresPrivateCloud(t *testing.T) {
	t.Parallel()

	ts := newGCPVMwareEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vmwareengine.v1.VmwareEngine/CreatePrivateCloud", []byte(`{"parent":"projects/stackyard/locations/us-central1","privateCloudId":"private-cloud-1"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vmwareengine grpc bridge create private cloud, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVMwareEngineRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPVMwareEngineContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "vmwareengine",
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/privateClouds?pageSize=1", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vmwareengine list private clouds, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	privateClouds, ok := listBody["privateClouds"].([]any)
	if !ok || len(privateClouds) == 0 {
		t.Fatalf("expected privateClouds array, got %#v", listBody["privateClouds"])
	}
	privateCloud, _ := privateClouds[0].(map[string]any)
	if _, ok := privateCloud["name"].(string); !ok {
		t.Fatalf("expected privateClouds[0].name string, got %#v", privateCloud["name"])
	}

	dnsForwardingResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/privateClouds/private-cloud-1/dnsForwarding", nil, headers)
	if dnsForwardingResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vmwareengine get dnsForwarding, got %d body=%s", dnsForwardingResp.StatusCode, string(providerContractBody(t, dnsForwardingResp)))
	}
	dnsForwardingBody := providerContractJSONMap(t, dnsForwardingResp)
	if _, ok := dnsForwardingBody["forwardingRules"].([]any); !ok {
		t.Fatalf("expected forwardingRules array, got %#v", dnsForwardingBody["forwardingRules"])
	}

	credsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/privateClouds/private-cloud-1:showNsxCredentials", nil, headers)
	if credsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vmwareengine show credentials, got %d body=%s", credsResp.StatusCode, string(providerContractBody(t, credsResp)))
	}
	credsBody := providerContractJSONMap(t, credsResp)
	if _, ok := credsBody["username"].(string); !ok {
		t.Fatalf("expected credentials username string, got %#v", credsBody["username"])
	}
	if _, ok := credsBody["password"].(string); !ok {
		t.Fatalf("expected credentials password string, got %#v", credsBody["password"])
	}

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/privateClouds?privateCloudId=private-cloud-1", []byte(`{"privateCloud":{"description":"pc"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vmwareengine",
	})
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vmwareengine create private cloud, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := providerContractJSONMap(t, createResp)
	if _, ok := createBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", createBody["name"])
	}
	if _, ok := createBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", createBody["done"])
	}
	if _, ok := createBody["metadata"].(map[string]any); !ok {
		t.Fatalf("expected operation metadata object, got %#v", createBody["metadata"])
	}
}

func TestGCPVMwareEngineRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/vmwareengine?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vmwareengine contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "vmwareengine" {
		t.Fatalf("expected service=vmwareengine, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name in probe response, got %#v", body["name"])
	}
}

func newGCPVMwareEngineContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPVMwareEngineSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "vmwareengine",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vmwareengine router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
