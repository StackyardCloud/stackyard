package server

import (
	"net/http"
	"strings"
	"testing"
)

// azurePrivateNetworkTestSetup creates a network with a /29 endpoints
// subnet, which has three usable addresses, and returns the subnet ID.
func azurePrivateNetworkTestSetup(t *testing.T, sub string, do func(method, path, body string) *http.Response) string {
	t.Helper()
	vnet := sub + "/resourceGroups/rg-pe/providers/Microsoft.Network/virtualNetworks/spoke"
	azureResourcesExpectStatus(t, do(http.MethodPut, vnet+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.1.0.0/16"]},"subnets":[{"name":"endpoints","properties":{"addressPrefix":"10.1.2.0/29"}}]}}`), http.StatusCreated)
	return strings.TrimPrefix(vnet, "/azure") + "/subnets/endpoints"
}

func azurePrivateEndpointTestBody(subnet, target, extra string) string {
	return `{"location":"eastus","properties":{"subnet":{"id":"` + subnet + `"},"privateLinkServiceConnections":[{"name":"blob","properties":{"privateLinkServiceId":"` + target + `","groupIds":["blob"]}}]` + extra + `}}`
}

func TestAzureNetworkPrivateEndpointLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pe")
	subnet := azurePrivateNetworkTestSetup(t, sub, do)
	account := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-pe/providers/Microsoft.Storage/storageAccounts/ShopData"
	pe := sub + "/resourceGroups/rg-pe/providers/Microsoft.Network/privateEndpoints/shop-blob"

	// A client round-tripping stale DNS configs gets the current ones.
	resp := do(http.MethodPut, pe+azureNetworkTestAPIVersion, azurePrivateEndpointTestBody(subnet, account, `,"customDnsConfigs":[{"fqdn":"shopdata.blob.core.windows.net","ipAddresses":["10.9.9.9"]}]`))
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	conn := props["privateLinkServiceConnections"].([]any)[0].(map[string]any)
	if conn["properties"].(map[string]any)["privateLinkServiceConnectionState"].(map[string]any)["status"] != "Approved" || !strings.HasSuffix(conn["id"].(string), "/privateLinkServiceConnections/blob") {
		t.Fatalf("unexpected connection: %#v", conn)
	}
	dns := props["customDnsConfigs"].([]any)[0].(map[string]any)
	if dns["fqdn"] != "shopdata.blob.core.windows.net" || dns["ipAddresses"].([]any)[0] != "10.1.2.4" {
		t.Fatalf("unexpected DNS config: %#v", dns)
	}
	nicID := props["networkInterfaces"].([]any)[0].(map[string]any)["id"].(string)
	resp = do(http.MethodGet, "/azure"+nicID+azureNetworkTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	nic := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	config := nic["ipConfigurations"].([]any)[0].(map[string]any)["properties"].(map[string]any)
	if config["privateIPAddress"] != "10.1.2.4" || nic["privateEndpoint"].(map[string]any)["id"] != strings.TrimPrefix(pe, "/azure") || nic["macAddress"] == nil {
		t.Fatalf("unexpected interface: %#v", nic)
	}

	static := func(address string) string {
		return azurePrivateEndpointTestBody(subnet, account, `,"ipConfigurations":[{"name":"ip","properties":{"groupId":"blob","memberName":"blob","privateIPAddress":"`+address+`"}}]`)
	}
	for _, address := range []string{"10.1.2.1", "10.1.2.7", "10.1.2.4"} {
		azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-pe/providers/Microsoft.Network/privateEndpoints/static"+azureNetworkTestAPIVersion, static(address)), http.StatusBadRequest, "InvalidRequestContent")
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPatch, pe+azureNetworkTestAPIVersion, `{"tags":{"a":"b"}}`), http.StatusMethodNotAllowed, "MethodNotAllowed")

	// A second endpoint takes the next address and keeps it on update;
	// the /29 has room for one more after that, then none.
	second := sub + "/resourceGroups/rg-pe/providers/Microsoft.Network/privateEndpoints/vault"
	vault := `{"location":"eastus","properties":{"subnet":{"id":"` + subnet + `"},"customNetworkInterfaceName":"vault-nic","manualPrivateLinkServiceConnections":[{"name":"vault","properties":{"privateLinkServiceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/kv","groupIds":["vault"],"requestMessage":"please"}}]}}`
	for range 2 {
		resp = do(http.MethodPut, second+azureNetworkTestAPIVersion, vault)
		props = providerContractJSONMap(t, resp)["properties"].(map[string]any)
		if got := props["customDnsConfigs"].([]any)[0].(map[string]any)["ipAddresses"].([]any)[0]; got != "10.1.2.5" {
			t.Fatalf("expected the second address, got %v", got)
		}
	}
	if got := props["manualPrivateLinkServiceConnections"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateLinkServiceConnectionState"].(map[string]any)["status"]; got != "Pending" {
		t.Fatalf("expected a pending manual connection, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, sub+"/resourceGroups/rg-pe/providers/Microsoft.Network/networkInterfaces/vault-nic"+azureNetworkTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-pe/providers/Microsoft.Network/privateEndpoints/third"+azureNetworkTestAPIVersion, azurePrivateEndpointTestBody(subnet, account, "")), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-pe/providers/Microsoft.Network/privateEndpoints/fourth"+azureNetworkTestAPIVersion, azurePrivateEndpointTestBody(subnet, account, "")), http.StatusBadRequest, "InvalidRequestContent")

	// Renaming the interface replaces it; deleting the endpoint deletes it.
	azureResourcesExpectStatus(t, do(http.MethodPut, second+azureNetworkTestAPIVersion, strings.Replace(vault, "vault-nic", "vault-nic2", 1)), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, sub+"/resourceGroups/rg-pe/providers/Microsoft.Network/networkInterfaces/vault-nic"+azureNetworkTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	group := pe + "/privateDnsZoneGroups/default"
	azureResourcesExpectStatus(t, do(http.MethodPut, group+azureNetworkTestAPIVersion, `{"properties":{"privateDnsZoneConfigs":[{"name":"blob","properties":{"privateDnsZoneId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/privateDnsZones/privatelink.blob.core.windows.net"}}]}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodDelete, pe+azureNetworkTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, "/azure"+nicID+azureNetworkTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodGet, group+azureNetworkTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureNetworkPrivateLinkServiceAndNAT(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pe")
	subnet := azurePrivateNetworkTestSetup(t, sub, do)
	ids := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-pe/providers/Microsoft.Network"

	pls := sub + "/resourceGroups/rg-pe/providers/Microsoft.Network/privateLinkServices/api"
	resp := do(http.MethodPut, pls+azureNetworkTestAPIVersion, `{"location":"East US","properties":{"loadBalancerFrontendIpConfigurations":[{"id":"`+ids+`/loadBalancers/lb/frontendIPConfigurations/fe"}],"ipConfigurations":[{"name":"nat","properties":{"subnet":{"id":"`+subnet+`"}}}],"visibility":{"subscriptions":["*"]}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	alias := props["alias"].(string)
	if !strings.HasPrefix(alias, "api.") || !strings.HasSuffix(alias, ".eastus.azure.privatelinkservice") || props["ipConfigurations"].([]any)[0].(map[string]any)["properties"].(map[string]any)["primary"] != true {
		t.Fatalf("unexpected private link service: %#v", props)
	}
	for check, want := range map[string]bool{alias: true, "other.azure.privatelinkservice": false} {
		resp = do(http.MethodPost, sub+"/providers/Microsoft.Network/locations/eastus/checkPrivateLinkServiceVisibility"+azureNetworkTestAPIVersion, `{"privateLinkServiceAlias":"`+check+`"}`)
		if got := providerContractJSONMap(t, resp)["visible"]; got != want {
			t.Fatalf("alias %s: expected visible %v, got %v", check, want, got)
		}
	}
	// An endpoint can connect by alias without group IDs.
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-pe/providers/Microsoft.Network/privateEndpoints/api"+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{"subnet":{"id":"`+subnet+`"},"manualPrivateLinkServiceConnections":[{"name":"api","properties":{"privateLinkServiceId":"`+alias+`"}}]}}`), http.StatusCreated)

	nat := sub + "/resourceGroups/rg-pe/providers/Microsoft.Network/natGateways/egress"
	resp = do(http.MethodPut, nat+azureNetworkTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard"},"zones":["1"],"properties":{"idleTimeoutInMinutes":10,"publicIpAddresses":[{"id":"`+ids+`/publicIPAddresses/egress"}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if providerContractJSONMap(t, resp)["properties"].(map[string]any)["resourceGuid"] == nil {
		t.Fatal("expected a NAT gateway resourceGuid")
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, "/azure"+subnet+azureNetworkTestAPIVersion, `{"properties":{"addressPrefix":"10.1.2.0/29","natGateway":{"id":"`+strings.TrimPrefix(nat, "/azure")+`"}}}`), http.StatusOK)
	asg := sub + "/resourceGroups/rg-pe/providers/Microsoft.Network/applicationSecurityGroups/web"
	azureResourcesExpectStatus(t, do(http.MethodPut, asg+azureNetworkTestAPIVersion, `{"location":"eastus"}`), http.StatusCreated)
}

func TestAzureNetworkPrivateValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pe")
	subnet := azurePrivateNetworkTestSetup(t, sub, do)
	network := sub + "/resourceGroups/rg-pe/providers/Microsoft.Network"
	account := "/subscriptions/x/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/data"
	fe := `"loadBalancerFrontendIpConfigurations":[{"id":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/loadBalancers/lb/frontendIPConfigurations/fe"}]`
	cases := []struct{ name, path, body string }{
		{"pe subnet", "/privateEndpoints/pe", azurePrivateEndpointTestBody("spoke", account, "")},
		{"pe missing subnet", "/privateEndpoints/pe", azurePrivateEndpointTestBody(strings.Replace(subnet, "endpoints", "nowhere", 1), account, "")},
		{"pe no connection", "/privateEndpoints/pe", `{"location":"eastus","properties":{"subnet":{"id":"` + subnet + `"}}}`},
		{"pe group ids", "/privateEndpoints/pe", strings.Replace(azurePrivateEndpointTestBody(subnet, account, ""), `,"groupIds":["blob"]`, ``, 1)},
		{"pe target", "/privateEndpoints/pe", azurePrivateEndpointTestBody(subnet, "data", "")},
		{"pe nic name", "/privateEndpoints/pe", azurePrivateEndpointTestBody(subnet, account, `,"customNetworkInterfaceName":"bad name"`)},
		{"pe static outside", "/privateEndpoints/pe", azurePrivateEndpointTestBody(subnet, account, `,"ipConfigurations":[{"name":"ip","properties":{"groupId":"blob","memberName":"blob","privateIPAddress":"10.9.9.9"}}]`)},
		{"pe asg", "/privateEndpoints/pe", azurePrivateEndpointTestBody(subnet, account, `,"applicationSecurityGroups":[{"id":"web"}]`)},
		{"nat sku", "/natGateways/n", `{"location":"eastus","sku":{"name":"Basic"}}`},
		{"nat zones", "/natGateways/n", `{"location":"eastus","zones":["1","2"]}`},
		{"nat timeout", "/natGateways/n", `{"location":"eastus","properties":{"idleTimeoutInMinutes":121}}`},
		{"nat ip", "/natGateways/n", `{"location":"eastus","properties":{"publicIpAddresses":[{"id":"ip"}]}}`},
		{"pls frontend", "/privateLinkServices/p", `{"location":"eastus","properties":{"ipConfigurations":[{"name":"a","properties":{"subnet":{"id":"` + subnet + `"}}}]}}`},
		{"pls ip configs", "/privateLinkServices/p", `{"location":"eastus","properties":{` + fe + `}}`},
		{"pls primary", "/privateLinkServices/p", `{"location":"eastus","properties":{` + fe + `,"ipConfigurations":[{"name":"a","properties":{"subnet":{"id":"` + subnet + `"}}},{"name":"b","properties":{"subnet":{"id":"` + subnet + `"}}}]}}`},
		{"pls auto approval", "/privateLinkServices/p", `{"location":"eastus","properties":{` + fe + `,"ipConfigurations":[{"name":"a","properties":{"subnet":{"id":"` + subnet + `"}}}],"autoApproval":{"subscriptions":["*"]}}}`},
		{"subnet nat", "/virtualNetworks/spoke/subnets/endpoints", `{"properties":{"addressPrefix":"10.1.2.0/29","natGateway":{"id":"nat"}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, network+tc.path+azureNetworkTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, network+"/privateEndpoints/pe"+azureNetworkTestAPIVersion, azurePrivateEndpointTestBody(subnet, account, "")), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, network+"/privateEndpoints/pe/privateDnsZoneGroups/g"+azureNetworkTestAPIVersion, `{"properties":{"privateDnsZoneConfigs":[{"name":"a","properties":{"privateDnsZoneId":"zone"}}]}}`), http.StatusBadRequest, "InvalidRequestContent")
}
