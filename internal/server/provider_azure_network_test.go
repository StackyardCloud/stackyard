package server

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

const azureNetworkTestAPIVersion = "?api-version=2024-05-01"

func azureNetworkTestList(t *testing.T, resp *http.Response, prop string) []any {
	t.Helper()
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	list, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)[prop].([]any)
	return list
}

func TestAzureNetworkVirtualNetworkSubnetSync(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-net")
	vnet := sub + "/resourceGroups/rg-net/providers/Microsoft.Network/virtualNetworks/hub"
	body := func(subnets string) string {
		return `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}` + subnets + `}}`
	}

	// Inline subnets become child resources.
	resp := do(http.MethodPut, vnet+azureNetworkTestAPIVersion, body(`,"subnets":[{"name":"web","properties":{"addressPrefix":"10.0.1.0/24"}}]`))
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	subnets := props["subnets"].([]any)
	if props["resourceGuid"] == nil || len(subnets) != 1 || !strings.HasSuffix(subnets[0].(map[string]any)["id"].(string), "/virtualNetworks/hub/subnets/web") {
		t.Fatalf("unexpected network properties: %#v", props)
	}
	resp = do(http.MethodGet, vnet+"/subnets/web"+azureNetworkTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["privateEndpointNetworkPolicies"]; got != "Disabled" {
		t.Fatalf("expected subnet defaults, got %v", got)
	}

	// Child subnets show up inline, checked against the address space and
	// their siblings.
	azureResourcesExpectStatus(t, do(http.MethodPut, vnet+"/subnets/data"+azureNetworkTestAPIVersion, `{"properties":{"addressPrefix":"10.0.2.0/24"}}`), http.StatusCreated)
	if got := azureNetworkTestList(t, do(http.MethodGet, vnet+azureNetworkTestAPIVersion, ""), "subnets"); len(got) != 2 {
		t.Fatalf("expected two subnets, got %d", len(got))
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, vnet+"/subnets/cache"+azureNetworkTestAPIVersion, `{"properties":{"addressPrefix":"10.0.1.128/25"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, vnet+"/subnets/cache"+azureNetworkTestAPIVersion, `{"properties":{"addressPrefix":"192.168.0.0/24"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, vnet+"/subnets/web"+azureNetworkTestAPIVersion, `{"properties":{"addressPrefix":"10.0.1.0/24","serviceEndpoints":[{"service":"Microsoft.Storage"}]}}`), http.StatusOK)

	// A PUT without subnets keeps them, and PATCH lists them as they are.
	azureResourcesExpectStatus(t, do(http.MethodPut, vnet+azureNetworkTestAPIVersion, body(`,"dhcpOptions":{"dnsServers":["10.0.0.4"]}`)), http.StatusOK)
	if got := azureNetworkTestList(t, do(http.MethodPatch, vnet+azureNetworkTestAPIVersion, `{"tags":{"env":"test"}}`), "subnets"); len(got) != 2 {
		t.Fatalf("expected subnets kept, got %d", len(got))
	}
	// A PUT with subnets replaces them.
	azureResourcesExpectStatus(t, do(http.MethodPut, vnet+azureNetworkTestAPIVersion, body(`,"subnets":[{"name":"web","properties":{"addressPrefix":"10.0.1.0/24"}}]`)), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, vnet+"/subnets/data"+azureNetworkTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")

	// Peerings are inline children too.
	spoke := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-net/providers/Microsoft.Network/virtualNetworks/spoke"
	resp = do(http.MethodPut, vnet+"/virtualNetworkPeerings/to-spoke"+azureNetworkTestAPIVersion, `{"properties":{"remoteVirtualNetwork":{"id":"`+spoke+`"},"allowForwardedTraffic":true}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["peeringState"] != "Connected" || props["allowVirtualNetworkAccess"] != true || props["allowForwardedTraffic"] != true {
		t.Fatalf("unexpected peering properties: %#v", props)
	}
	if got := azureNetworkTestList(t, do(http.MethodGet, vnet+azureNetworkTestAPIVersion, ""), "virtualNetworkPeerings"); len(got) != 1 {
		t.Fatalf("expected one peering, got %d", len(got))
	}

	// Deleting a child drops it from the parent.
	azureResourcesExpectStatus(t, do(http.MethodDelete, vnet+"/subnets/web"+azureNetworkTestAPIVersion, ""), http.StatusOK)
	if got := azureNetworkTestList(t, do(http.MethodGet, vnet+azureNetworkTestAPIVersion, ""), "subnets"); len(got) != 0 {
		t.Fatalf("expected no subnets, got %d", len(got))
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, vnet+azureNetworkTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, vnet+"/virtualNetworkPeerings/to-spoke"+azureNetworkTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureNetworkSecurityGroupRules(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-net")
	nsg := sub + "/resourceGroups/rg-net/providers/Microsoft.Network/networkSecurityGroups/web-nsg"
	rule := func(priority, direction string) string {
		return `{"properties":{"protocol":"Tcp","access":"Allow","direction":"` + direction + `","priority":` + priority + `,"sourceAddressPrefix":"*","sourcePortRange":"*","destinationAddressPrefix":"*","destinationPortRanges":["80","443","8000-8080"]}}`
	}

	resp := do(http.MethodPut, nsg+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{"securityRules":[{"name":"allow-ssh","properties":{"protocol":"Tcp","access":"Allow","direction":"Inbound","priority":100,"sourceAddressPrefix":"10.0.0.0/8","sourcePortRange":"*","destinationAddressPrefix":"*","destinationPortRange":"22"}}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if len(props["defaultSecurityRules"].([]any)) != 6 || len(props["securityRules"].([]any)) != 1 {
		t.Fatalf("unexpected security group properties: %#v", props)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, nsg+"/securityRules/allow-web"+azureNetworkTestAPIVersion, rule("100", "Inbound")), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, nsg+"/securityRules/allow-web"+azureNetworkTestAPIVersion, rule("100", "Outbound")), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, nsg+"/securityRules/allow-web"+azureNetworkTestAPIVersion, rule("110", "Outbound")), http.StatusOK)
	if got := azureNetworkTestList(t, do(http.MethodGet, nsg+azureNetworkTestAPIVersion, ""), "securityRules"); len(got) != 2 {
		t.Fatalf("expected two rules, got %d", len(got))
	}
	resp = do(http.MethodGet, nsg+"/securityRules"+azureNetworkTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 2 {
		t.Fatalf("expected two listed rules, got %d", len(got))
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, nsg+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{"securityRules":[{"name":"a","properties":`+strings.TrimSuffix(strings.TrimPrefix(rule("200", "Inbound"), `{"properties":`), "}")+`},{"name":"b","properties":`+strings.TrimSuffix(strings.TrimPrefix(rule("200", "Inbound"), `{"properties":`), "}")+`}]}}`), http.StatusBadRequest, "InvalidRequestContent")
}

func TestAzureNetworkAddressesAndInterfaces(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-net")
	group := sub + "/resourceGroups/rg-net/providers/Microsoft.Network"
	ids := strings.TrimPrefix(group, "/azure")

	resp := do(http.MethodPut, group+"/publicIPAddresses/web-ip"+azureNetworkTestAPIVersion, `{"location":"East US","sku":{"name":"Standard"},"properties":{"dnsSettings":{"domainNameLabel":"stackyard-web"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if !regexp.MustCompile(`^20\.\d+\.\d+\.\d+$`).MatchString(props["ipAddress"].(string)) || props["publicIPAllocationMethod"] != "Static" ||
		props["dnsSettings"].(map[string]any)["fqdn"] != "stackyard-web.eastus.cloudapp.azure.com" || props["idleTimeoutInMinutes"] != float64(4) {
		t.Fatalf("unexpected public IP properties: %#v", props)
	}
	resp = do(http.MethodPut, group+"/publicIPAddresses/basic-ip"+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{}}`)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["publicIPAllocationMethod"] != "Dynamic" || props["ipAddress"] != nil {
		t.Fatalf("unexpected dynamic IP properties: %#v", props)
	}

	resp = do(http.MethodPut, group+"/networkInterfaces/web-nic"+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{"networkSecurityGroup":{"id":"`+ids+`/networkSecurityGroups/web-nsg"},"ipConfigurations":[{"name":"ipconfig1","properties":{"subnet":{"id":"`+ids+`/virtualNetworks/hub/subnets/web"},"publicIPAddress":{"id":"`+ids+`/publicIPAddresses/web-ip"}}}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props = providerContractJSONMap(t, resp)["properties"].(map[string]any)
	config := props["ipConfigurations"].([]any)[0].(map[string]any)
	cp := config["properties"].(map[string]any)
	if !regexp.MustCompile(`^00-0D-3A-[0-9A-F]{2}-[0-9A-F]{2}-[0-9A-F]{2}$`).MatchString(props["macAddress"].(string)) ||
		!strings.HasSuffix(config["id"].(string), "/networkInterfaces/web-nic/ipConfigurations/ipconfig1") || cp["primary"] != true || cp["privateIPAllocationMethod"] != "Dynamic" {
		t.Fatalf("unexpected interface properties: %#v", props)
	}

	frontend := ids + "/loadBalancers/web-lb/frontendIPConfigurations/public"
	pool := ids + "/loadBalancers/web-lb/backendAddressPools/web"
	probe := ids + "/loadBalancers/web-lb/probes/http"
	resp = do(http.MethodPut, group+"/loadBalancers/web-lb"+azureNetworkTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard"},"properties":{
		"frontendIPConfigurations":[{"name":"public","properties":{"publicIPAddress":{"id":"`+ids+`/publicIPAddresses/web-ip"}}}],
		"backendAddressPools":[{"name":"web"}],
		"probes":[{"name":"http","properties":{"protocol":"Http","port":80,"requestPath":"/healthz"}}],
		"loadBalancingRules":[{"name":"http","properties":{"protocol":"Tcp","frontendPort":80,"backendPort":8080,"frontendIPConfiguration":{"id":"`+frontend+`"},"backendAddressPool":{"id":"`+pool+`"},"probe":{"id":"`+probe+`"}}}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props = providerContractJSONMap(t, resp)["properties"].(map[string]any)
	rule := props["loadBalancingRules"].([]any)[0].(map[string]any)
	if !strings.HasSuffix(rule["id"].(string), "/loadBalancers/web-lb/loadBalancingRules/http") || rule["properties"].(map[string]any)["backendPort"] != float64(8080) {
		t.Fatalf("unexpected load balancer rule: %#v", rule)
	}

	table := group + "/routeTables/spoke-routes"
	resp = do(http.MethodPut, table+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{"disableBgpRoutePropagation":true,"routes":[{"name":"to-firewall","properties":{"addressPrefix":"0.0.0.0/0","nextHopType":"VirtualAppliance","nextHopIpAddress":"10.0.0.4"}}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, table+"/routes/storage"+azureNetworkTestAPIVersion, `{"properties":{"addressPrefix":"Storage","nextHopType":"Internet"}}`), http.StatusCreated)
	if got := azureNetworkTestList(t, do(http.MethodGet, table+azureNetworkTestAPIVersion, ""), "routes"); len(got) != 2 {
		t.Fatalf("expected two routes, got %d", len(got))
	}
}

func TestAzureNetworkValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-net")
	group := sub + "/resourceGroups/rg-net/providers/Microsoft.Network"
	ids := strings.TrimPrefix(group, "/azure")
	azureResourcesExpectStatus(t, do(http.MethodPut, group+"/virtualNetworks/hub"+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, group+"/networkSecurityGroups/nsg"+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{}}`), http.StatusCreated)
	subnet := `{"id":"` + ids + `/virtualNetworks/hub/subnets/web"}`
	rule := `"protocol":"Tcp","access":"Allow","direction":"Inbound","priority":100,"sourceAddressPrefix":"*","sourcePortRange":"*","destinationAddressPrefix":"*"`
	frontend := `"frontendIPConfigurations":[{"name":"fe","properties":{"subnet":` + subnet + `}}]`
	feRef := `"frontendIPConfiguration":{"id":"` + ids + `/loadBalancers/lb/frontendIPConfigurations/fe"}`
	cases := []struct{ name, path, body string }{
		{"vnet no space", "/virtualNetworks/v1", `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":[]}}}`},
		{"vnet bad prefix", "/virtualNetworks/v1", `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.1/16"]}}}`},
		{"vnet dns server", "/virtualNetworks/v1", `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.1.0.0/16"]},"dhcpOptions":{"dnsServers":["dns"]}}}`},
		{"inline subnet name", "/virtualNetworks/v1", `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.1.0.0/16"]},"subnets":[{"properties":{"addressPrefix":"10.1.0.0/24"}}]}}`},
		{"inline subnet overlap", "/virtualNetworks/v1", `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.1.0.0/16"]},"subnets":[{"name":"a","properties":{"addressPrefix":"10.1.0.0/24"}},{"name":"b","properties":{"addressPrefix":"10.1.0.0/25"}}]}}`},
		{"inline subnet outside", "/virtualNetworks/v1", `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.1.0.0/16"]},"subnets":[{"name":"a","properties":{"addressPrefix":"10.2.0.0/24"}}]}}`},
		{"inline subnet slash", "/virtualNetworks/v1", `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.1.0.0/16"]},"subnets":[{"name":"a/b","properties":{"addressPrefix":"10.1.0.0/24"}}]}}`},
		{"inline subnet duplicate", "/virtualNetworks/v1", `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.1.0.0/16"]},"subnets":[{"name":"a","properties":{"addressPrefix":"10.1.0.0/24"}},{"name":"A","properties":{"addressPrefix":"10.1.1.0/24"}}]}}`},
		{"subnet no prefix", "/virtualNetworks/hub/subnets/s1", `{"properties":{}}`},
		{"subnet nsg ref", "/virtualNetworks/hub/subnets/s1", `{"properties":{"addressPrefix":"10.0.5.0/24","networkSecurityGroup":{"id":"nsg"}}}`},
		{"subnet delegation", "/virtualNetworks/hub/subnets/s1", `{"properties":{"addressPrefix":"10.0.5.0/24","delegations":[{"name":"d","properties":{}}]}}`},
		{"peering remote", "/virtualNetworks/hub/virtualNetworkPeerings/p1", `{"properties":{"remoteVirtualNetwork":{"id":"spoke"}}}`},
		{"peering gateways", "/virtualNetworks/hub/virtualNetworkPeerings/p1", `{"properties":{"remoteVirtualNetwork":{"id":"` + ids + `/virtualNetworks/spoke"},"useRemoteGateways":true,"allowGatewayTransit":true}}`},
		{"rule protocol", "/networkSecurityGroups/nsg/securityRules/r1", `{"properties":{` + strings.Replace(rule, "Tcp", "Http", 1) + `,"destinationPortRange":"80"}}`},
		{"rule priority", "/networkSecurityGroups/nsg/securityRules/r1", `{"properties":{` + strings.Replace(rule, "100", "99", 1) + `,"destinationPortRange":"80"}}`},
		{"rule port", "/networkSecurityGroups/nsg/securityRules/r1", `{"properties":{` + rule + `,"destinationPortRange":"80-70000"}}`},
		{"rule no port", "/networkSecurityGroups/nsg/securityRules/r1", `{"properties":{` + rule + `}}`},
		{"rule no destination", "/networkSecurityGroups/nsg/securityRules/r1", `{"properties":{"protocol":"Tcp","access":"Allow","direction":"Inbound","priority":100,"sourceAddressPrefix":"*","sourcePortRange":"*","destinationPortRange":"80"}}`},
		{"ip sku", "/publicIPAddresses/ip1", `{"location":"eastus","sku":{"name":"Premium"},"properties":{}}`},
		{"ip standard dynamic", "/publicIPAddresses/ip1", `{"location":"eastus","sku":{"name":"Standard"},"properties":{"publicIPAllocationMethod":"Dynamic"}}`},
		{"ip idle timeout", "/publicIPAddresses/ip1", `{"location":"eastus","properties":{"idleTimeoutInMinutes":60}}`},
		{"ip dns label", "/publicIPAddresses/ip1", `{"location":"eastus","properties":{"dnsSettings":{"domainNameLabel":"Web_1"}}}`},
		{"nic no configs", "/networkInterfaces/n1", `{"location":"eastus","properties":{"ipConfigurations":[]}}`},
		{"nic subnet", "/networkInterfaces/n1", `{"location":"eastus","properties":{"ipConfigurations":[{"name":"c","properties":{"subnet":{"id":"web"}}}]}}`},
		{"nic static", "/networkInterfaces/n1", `{"location":"eastus","properties":{"ipConfigurations":[{"name":"c","properties":{"subnet":` + subnet + `,"privateIPAllocationMethod":"Static"}}]}}`},
		{"nic primary", "/networkInterfaces/n1", `{"location":"eastus","properties":{"ipConfigurations":[{"name":"a","properties":{"subnet":` + subnet + `}},{"name":"b","properties":{"subnet":` + subnet + `}}]}}`},
		{"lb sku", "/loadBalancers/lb", `{"location":"eastus","sku":{"name":"Premium"},"properties":{` + frontend + `}}`},
		{"lb two targets", "/loadBalancers/lb", `{"location":"eastus","properties":{"frontendIPConfigurations":[{"name":"fe","properties":{"subnet":` + subnet + `,"publicIPAddress":{"id":"` + ids + `/publicIPAddresses/ip"}}}]}}`},
		{"lb nat range", "/loadBalancers/lb", `{"location":"eastus","properties":{` + frontend + `,"inboundNatRules":[{"name":"n","properties":{"protocol":"Tcp","frontendPortRangeStart":5000,"frontendPortRangeEnd":4000,"backendPort":22,` + feRef + `}}]}}`},
		{"lb frontend target", "/loadBalancers/lb", `{"location":"eastus","properties":{"frontendIPConfigurations":[{"name":"fe","properties":{}}]}}`},
		{"lb probe path", "/loadBalancers/lb", `{"location":"eastus","properties":{` + frontend + `,"probes":[{"name":"p","properties":{"protocol":"Http","port":80}}]}}`},
		{"lb rule frontend", "/loadBalancers/lb", `{"location":"eastus","properties":{` + frontend + `,"loadBalancingRules":[{"name":"r","properties":{"protocol":"Tcp","frontendPort":80,"backendPort":80,"frontendIPConfiguration":{"id":"` + ids + `/loadBalancers/lb/frontendIPConfigurations/other"}}}]}}`},
		{"lb rule pool", "/loadBalancers/lb", `{"location":"eastus","properties":{` + frontend + `,"loadBalancingRules":[{"name":"r","properties":{"protocol":"Tcp","frontendPort":80,"backendPort":80,` + feRef + `,"backendAddressPool":{"id":"` + ids + `/loadBalancers/lb/backendAddressPools/missing"}}}]}}`},
		{"lb nat port", "/loadBalancers/lb", `{"location":"eastus","properties":{` + frontend + `,"inboundNatRules":[{"name":"n","properties":{"protocol":"Tcp","frontendPort":70000,"backendPort":22,` + feRef + `}}]}}`},
		{"route prefix", "/routeTables/rt/routes/r1", `{"properties":{"addressPrefix":"10.0.0.0/33","nextHopType":"Internet"}}`},
		{"route hop", "/routeTables/rt/routes/r1", `{"properties":{"addressPrefix":"0.0.0.0/0","nextHopType":"Firewall"}}`},
		{"route appliance ip", "/routeTables/rt/routes/r1", `{"properties":{"addressPrefix":"0.0.0.0/0","nextHopType":"VirtualAppliance"}}`},
		{"route stray ip", "/routeTables/rt/routes/r1", `{"properties":{"addressPrefix":"0.0.0.0/0","nextHopType":"Internet","nextHopIpAddress":"10.0.0.4"}}`},
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, group+"/routeTables/rt"+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{}}`), http.StatusCreated)
	for _, tc := range cases {
		resp := do(http.MethodPut, group+tc.path+azureNetworkTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}

func TestAzureNetworkInlineEdgeCases(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-net")
	group := sub + "/resourceGroups/rg-net/providers/Microsoft.Network"
	ids := strings.TrimPrefix(group, "/azure")
	vnet := group + "/virtualNetworks/hub"
	azureResourcesExpectStatus(t, do(http.MethodPut, vnet+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16","10.1.0.0/16"]},"subnets":[{"name":"web","properties":{"addressPrefix":"10.1.0.0/24"}}]}}`), http.StatusCreated)

	// Names match without regard to case, and null keeps the children.
	azureResourcesExpectStatus(t, do(http.MethodPut, vnet+"/subnets/WEB"+azureNetworkTestAPIVersion, `{"properties":{"addressPrefix":"10.1.0.0/24"}}`), http.StatusOK)
	resp := do(http.MethodPut, vnet+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16","10.1.0.0/16"]},"subnets":null}}`)
	if got := azureNetworkTestList(t, resp, "subnets"); len(got) != 1 || got[0].(map[string]any)["name"] != "web" {
		t.Fatalf("expected the web subnet kept, got %#v", got)
	}
	// Shrinking the address space must not strand a kept subnet.
	azureResourcesExpectErrorCode(t, do(http.MethodPut, vnet+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`), http.StatusBadRequest, "InvalidRequestContent")

	// A single IP configuration is always primary.
	resp = do(http.MethodPut, group+"/networkInterfaces/nic"+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{"ipConfigurations":[{"name":"c","properties":{"primary":false,"subnet":{"id":"`+ids+`/virtualNetworks/hub/subnets/web"}}}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	config := providerContractJSONMap(t, resp)["properties"].(map[string]any)["ipConfigurations"].([]any)[0].(map[string]any)
	if config["properties"].(map[string]any)["primary"] != true {
		t.Fatalf("expected a primary configuration, got %#v", config)
	}

	// Load balancers may start without frontends, use public IP prefixes and
	// map NAT port ranges to a backend pool.
	azureResourcesExpectStatus(t, do(http.MethodPut, group+"/loadBalancers/empty"+azureNetworkTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard"},"properties":{}}`), http.StatusCreated)
	lb := ids + "/loadBalancers/nat"
	azureResourcesExpectStatus(t, do(http.MethodPut, group+"/loadBalancers/nat"+azureNetworkTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard"},"properties":{"frontendIPConfigurations":[{"name":"fe","properties":{"publicIPPrefix":{"id":"`+ids+`/publicIPPrefixes/prefix"}}}],"backendAddressPools":[{"name":"vms"}],"inboundNatRules":[{"name":"ssh","properties":{"protocol":"Tcp","frontendPortRangeStart":50000,"frontendPortRangeEnd":50100,"backendPort":22,"frontendIPConfiguration":{"id":"`+lb+`/frontendIPConfigurations/fe"},"backendAddressPool":{"id":"`+lb+`/backendAddressPools/vms"}}}]}}`), http.StatusCreated)

	// A null next hop address is the same as none.
	azureResourcesExpectStatus(t, do(http.MethodPut, group+"/routeTables/rt"+azureNetworkTestAPIVersion, `{"location":"eastus","properties":{"routes":[{"name":"out","properties":{"addressPrefix":"0.0.0.0/0","nextHopType":"Internet","nextHopIpAddress":null}}]}}`), http.StatusCreated)
}

// Every inline property must name a registered child type of its namespace.
func TestAzureARMInlineChildrenRegistered(t *testing.T) {
	for _, typ := range azureARMResourceTypes {
		for prop, childType := range typ.InlineChildren {
			child, ok := lookupAzureARMResourceType(typ.Namespace, childType)
			if !ok || !strings.HasPrefix(strings.ToLower(child.Type), strings.ToLower(typ.Type)+"/") {
				t.Errorf("%s inline property %s names %s, which is not a registered child type", typ.fullType(), prop, childType)
			}
		}
	}
}
