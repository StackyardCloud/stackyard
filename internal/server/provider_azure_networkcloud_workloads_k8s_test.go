package server

import (
	"net/http"
	"testing"
)

func azureNCWK8s(h *azureNCWTest, change func(props, network, pool map[string]any)) map[string]any {
	pool := map[string]any{
		"name": "SystemPool-1", "count": 2, "mode": "System", "vmSkuName": "NC_P46_224_v1",
		"attachedNetworkConfiguration": map[string]any{"l2Networks": []any{map[string]any{"networkId": h.arm + "l2Networks/l2", "pluginType": "DPDK"}}},
		"agentOptions":                 map[string]any{"hugepagesCount": 96, "hugepagesSize": "1G"},
		"upgradeSettings":              map[string]any{"maxSurge": "1"},
	}
	network := map[string]any{
		"cloudServicesNetworkId": h.arm + "cloudServicesNetworks/csn", "cniNetworkId": h.arm + "l3Networks/l3",
		"serviceCidrs": []any{"198.51.101.0/24"}, "podCidrs": []any{"198.51.102.0/24"}, "dnsServiceIp": "198.51.101.2",
		"bgpServiceLoadBalancerConfiguration": map[string]any{
			"ipAddressPools":    []any{map[string]any{"name": "pool1", "addresses": []any{"198.51.103.0/24"}}},
			"bgpPeers":          []any{map[string]any{"name": "peer1", "peerAddress": "203.0.113.254", "peerAsn": 64497, "password": "secret1"}},
			"bgpAdvertisements": []any{map[string]any{"ipAddressPools": []any{"pool1"}, "peers": []any{"peer1"}}},
		},
	}
	props := map[string]any{
		"kubernetesVersion":              "1.32.7",
		"controlPlaneNodeConfiguration":  map[string]any{"count": 3, "vmSkuName": "NC_G6_28_v1"},
		"initialAgentPoolConfigurations": []any{pool},
		"networkConfiguration":           network,
	}
	if change != nil {
		change(props, network, pool)
	}
	return azureNCWBody(props)
}

func TestAzureNetworkCloudKubernetesCluster(t *testing.T) {
	h := newAzureNCWTest(t)
	h.networks()

	bad := []struct {
		text   string
		change func(props, network, pool map[string]any)
	}{
		{"at least one System agent pool", func(_, _, pool map[string]any) { pool["mode"] = "User" }},
		{"existing L3 network", func(_, network, _ map[string]any) { network["cniNetworkId"] = h.arm + "l2Networks/l2" }},
		{"dnsServiceIp", func(_, network, _ map[string]any) { network["dnsServiceIp"] = "10.0.0.10" }},
		{"which ipAddressPools doesn't define", func(_, network, _ map[string]any) {
			azureARMObject(network["bgpServiceLoadBalancerConfiguration"])["bgpAdvertisements"] = []any{map[string]any{"ipAddressPools": []any{"nope"}}}
		}},
		{"single-address prefix", func(_, network, _ map[string]any) {
			azureARMObject(network["bgpServiceLoadBalancerConfiguration"])["ipAddressPools"] = []any{map[string]any{"name": "pool1", "addresses": []any{"198.51.103.1/32"}}}
		}},
		{"must be greater than 0", func(_, _, pool map[string]any) {
			pool["upgradeSettings"] = map[string]any{"maxSurge": "0", "maxUnavailable": "0%"}
		}},
		{"isn't an existing l2Network", func(_, _, pool map[string]any) {
			pool["attachedNetworkConfiguration"] = map[string]any{"l2Networks": []any{map[string]any{"networkId": h.arm + "l3Networks/l3"}}}
		}},
		{"used twice", func(props, _, pool map[string]any) {
			props["initialAgentPoolConfigurations"] = []any{pool, azureARMMerge(pool, map[string]any{"name": "systempool-1"})}
		}},
	}
	for _, c := range bad {
		h.fails(http.MethodPut, "kubernetesClusters/kbad", azureNCWK8s(h, c.change), http.StatusBadRequest, c.text)
	}

	cluster := h.props(http.MethodPut, "kubernetesClusters/k1", azureNCWK8s(h, nil), http.StatusCreated)
	if len(azureARMSlice(cluster["initialAgentPoolConfigurations"])) != 0 || len(azureARMSlice(cluster["nodes"])) != 5 ||
		cluster["controlPlaneKubernetesVersion"] != "1.32.7" || len(azureARMSlice(cluster["attachedNetworkIds"])) != 3 {
		t.Fatalf("cluster = %v", cluster)
	}
	bgp := azureARMObject(azureARMObject(cluster["networkConfiguration"])["bgpServiceLoadBalancerConfiguration"])
	if azureARMObject(azureARMSlice(bgp["bgpPeers"])[0])["password"] != nil {
		t.Fatalf("BGP password returned: %v", bgp)
	}
	pool := h.props(http.MethodGet, "kubernetesClusters/k1/agentPools/SystemPool-1", nil, http.StatusOK)
	if pool["kubernetesVersion"] != "1.32.7" || pool["mode"] != "System" || pool["name"] != nil {
		t.Fatalf("initial pool = %v", pool)
	}
	clusterID := h.arm + "kubernetesClusters/k1"
	if p := h.props(http.MethodGet, "l2Networks/l2", nil, http.StatusOK); azureARMSlice(p["hybridAksClustersAssociatedIds"])[0] != clusterID {
		t.Fatalf("l2 associations = %v", p)
	}
	h.fails(http.MethodDelete, "l2Networks/l2", nil, http.StatusConflict, "in use")

	// A GET body (no BGP password, empty initial pools) can be sent back.
	got := h.call(http.MethodGet, "kubernetesClusters/k1", nil, http.StatusOK)
	h.call(http.MethodPut, "kubernetesClusters/k1", got, http.StatusOK)
	h.fails(http.MethodPatch, "kubernetesClusters/k1", map[string]any{"properties": map[string]any{"kubernetesVersion": ""}}, http.StatusBadRequest, "kubernetesVersion")
	h.fails(http.MethodPatch, "kubernetesClusters/k1", map[string]any{"properties": map[string]any{"kubernetesVersion": "1.20.0"}}, http.StatusBadRequest, "kubernetesVersion")

	// Agent pools: a User pool comes and goes; the last System pool can't
	// lose its nodes.
	user := azureNCWBody(map[string]any{"count": 1, "mode": "User", "vmSkuName": "NC_P46_224_v1"})
	delete(user, "extendedLocation")
	h.call(http.MethodPut, "kubernetesClusters/k1/agentPools/user1", user, http.StatusCreated)
	if p := h.props(http.MethodGet, "kubernetesClusters/k1", nil, http.StatusOK); len(azureARMSlice(p["nodes"])) != 6 {
		t.Fatalf("nodes after user pool = %v", p["nodes"])
	}
	h.fails(http.MethodPatch, "kubernetesClusters/k1/agentPools/SystemPool-1", map[string]any{"properties": map[string]any{"count": 0}}, http.StatusConflict, "last one")
	h.fails(http.MethodDelete, "kubernetesClusters/k1/agentPools/SystemPool-1", nil, http.StatusConflict, "last one")
	h.fails(http.MethodPatch, "kubernetesClusters/k1/agentPools/user1", map[string]any{"properties": map[string]any{"mode": "System"}}, http.StatusBadRequest, "can't be updated")
	h.fails(http.MethodPut, "kubernetesClusters/k1/agentPools/user1", azureARMMerge(user, map[string]any{"properties": map[string]any{"vmSkuName": "other"}}), http.StatusBadRequest, "vmSkuName can't be changed")
	h.call(http.MethodPatch, "kubernetesClusters/k1/agentPools/user1", map[string]any{"properties": map[string]any{"count": 3}}, http.StatusOK)
	if p := h.call(http.MethodGet, "kubernetesClusters/k1/agentPools/user1", nil, http.StatusOK); azureARMObject(p["extendedLocation"])["name"] != azureNCWTestLocation["name"] {
		t.Fatalf("pool extendedLocation = %v", p["extendedLocation"])
	}
	// Upgrade settings are checked once a PATCH merges: dropping maxSurge
	// to 0 is fine while maxUnavailable is above 0, not both.
	h.call(http.MethodPatch, "kubernetesClusters/k1/agentPools/user1", map[string]any{"properties": map[string]any{"upgradeSettings": map[string]any{"maxUnavailable": "1"}}}, http.StatusOK)
	h.call(http.MethodPatch, "kubernetesClusters/k1/agentPools/user1", map[string]any{"properties": map[string]any{"upgradeSettings": map[string]any{"maxSurge": "0"}}}, http.StatusOK)
	h.fails(http.MethodPatch, "kubernetesClusters/k1/agentPools/user1", map[string]any{"properties": map[string]any{"upgradeSettings": map[string]any{"maxUnavailable": "0%"}}},
		http.StatusBadRequest, "One of properties.upgradeSettings.maxSurge and maxUnavailable must be greater than 0.")
	h.fails(http.MethodPatch, "kubernetesClusters/k1/agentPools/user1", map[string]any{"properties": map[string]any{"upgradeSettings": map[string]any{"maxSurge": 2}}}, http.StatusBadRequest, "maxSurge must be")
	h.call(http.MethodDelete, "kubernetesClusters/k1/agentPools/user1", nil, http.StatusOK)

	// Updates: version, control plane count and keys; not the network.
	k := h.props(http.MethodPatch, "kubernetesClusters/k1", map[string]any{"properties": map[string]any{
		"kubernetesVersion": "1.33.3", "controlPlaneNodeConfiguration": map[string]any{"count": 1},
	}}, http.StatusOK)
	if k["controlPlaneKubernetesVersion"] != "1.33.3" || len(azureARMSlice(k["nodes"])) != 3 {
		t.Fatalf("patched cluster = %v", k)
	}
	if p := h.props(http.MethodGet, "kubernetesClusters/k1/agentPools/SystemPool-1", nil, http.StatusOK); p["kubernetesVersion"] != "1.33.3" {
		t.Fatalf("pool version = %v", p["kubernetesVersion"])
	}
	h.fails(http.MethodPatch, "kubernetesClusters/k1", map[string]any{"properties": map[string]any{"networkConfiguration": map[string]any{}}}, http.StatusBadRequest, "can't be updated")
	h.fails(http.MethodPut, "kubernetesClusters/k1", azureNCWK8s(h, func(_, network, _ map[string]any) { network["podCidrs"] = []any{"10.0.0.0/16"} }), http.StatusBadRequest, "networkConfiguration can't be changed")
	h.fails(http.MethodPut, "kubernetesClusters/k1", azureNCWK8s(h, func(props, _, _ map[string]any) {
		props["controlPlaneNodeConfiguration"] = map[string]any{"count": 1, "vmSkuName": "other"}
	}), http.StatusBadRequest, "vmSkuName can't be changed")

	h.fails(http.MethodPost, "kubernetesClusters/k1/restartNode", map[string]any{"nodeName": "nope"}, http.StatusBadRequest, "isn't a node")
	if out := h.call(http.MethodPost, "kubernetesClusters/k1/restartNode", map[string]any{}, http.StatusBadRequest); azureARMObject(out["error"])["code"] != "InvalidRequestContent" {
		t.Fatalf("restartNode error = %v", out)
	}
	h.call(http.MethodPost, "kubernetesClusters/k1/restartNode", map[string]any{"nodeName": "k1-control-plane-1"}, http.StatusAccepted)

	f := h.props(http.MethodPut, "kubernetesClusters/k1/features/metrics-server", map[string]any{"location": "eastus", "properties": map[string]any{"options": []any{map[string]any{"key": "a", "value": "b"}}}}, http.StatusCreated)
	if f["detailedStatus"] != "Installed" || f["version"] == nil {
		t.Fatalf("feature = %v", f)
	}
	h.fails(http.MethodPut, "kubernetesClusters/k1/features/bad", map[string]any{"location": "eastus", "properties": map[string]any{"options": []any{map[string]any{"key": "a"}}}}, http.StatusBadRequest, "value")
	if p := h.props(http.MethodGet, "kubernetesClusters/k1", nil, http.StatusOK); len(azureARMSlice(p["featureStatuses"])) != 1 {
		t.Fatalf("featureStatuses = %v", p["featureStatuses"])
	}

	h.call(http.MethodDelete, "kubernetesClusters/k1", nil, http.StatusOK)
	h.call(http.MethodGet, "kubernetesClusters/k1/agentPools/SystemPool-1", nil, http.StatusNotFound)
	h.call(http.MethodDelete, "l2Networks/l2", nil, http.StatusOK)
}
