package server

import (
	"fmt"
	"net/http"
	"net/netip"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Microsoft.Network NAT gateways, application security groups, private
// endpoints with their DNS zone groups, and private link services, served
// by the shared ARM handler. A private endpoint gets a network interface
// with the next free address in its subnet, created and deleted with it,
// and its connections are approved at once (manual ones stay pending).
func init() {
	const ns = "Microsoft.Network"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "natGateways",
		Validate:           azureNATGatewayValidate,
		ReadOnlyProperties: azureNetworkResourceGUID,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "applicationSecurityGroups",
		ReadOnlyProperties: azureNetworkResourceGUID,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "privateEndpoints",
		Validate:           azurePrivateEndpointValidate,
		ReadOnlyProperties: azurePrivateEndpointProperties,
		Linked:             azurePrivateEndpointLinked,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "privateEndpoints/privateDnsZoneGroups", Proxy: true,
		Validate: azurePrivateDNSZoneGroupValidate,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "privateLinkServices",
		Validate:           azurePrivateLinkServiceValidate,
		ReadOnlyProperties: azurePrivateLinkServiceProperties,
	})
	registerAzureARMProviderAction(ns, "locations/checkPrivateLinkServiceVisibility", func(s *Server, payload map[string]any) (int, any) {
		return s.azurePrivateLinkServiceVisibility(payload)
	})
	registerAzureARMGroupAction(ns, "checkPrivateLinkServiceVisibility", func(s *Server, _, _ string, payload map[string]any) (int, any) {
		return s.azurePrivateLinkServiceVisibility(payload)
	})
}

// azureNetworkRefs checks that every item of props[key], when set,
// references a resource of typePath.
func azureNetworkRefs(props map[string]any, key, typePath string) string {
	for i, item := range azureARMSlice(props[key]) {
		if !azureNetworkIDOf(item, typePath) {
			return fmt.Sprintf("properties.%s[%d].id must be a %s resource ID.", key, i, typePath)
		}
	}
	return ""
}

func azureNATGatewayValidate(payload map[string]any) string {
	if sku, present := azureARMObject(payload["sku"])["name"]; present && sku != "Standard" {
		return "sku.name must be Standard."
	}
	if len(azureARMSlice(payload["zones"])) > 1 {
		return "A NAT gateway can be deployed to only one availability zone."
	}
	props := azureARMObject(payload["properties"])
	if raw, present := props["idleTimeoutInMinutes"]; present && !azureWholeNumber(raw, 4, 120) {
		return "properties.idleTimeoutInMinutes must be an integer from 4 to 120."
	}
	if msg := azureNetworkRefs(props, "publicIpAddresses", "publicipaddresses"); msg != "" {
		return msg
	}
	return azureNetworkRefs(props, "publicIpPrefixes", "publicipprefixes")
}

var azureNICNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,78}[a-zA-Z0-9_]$`)

// azurePrivateEndpointConnections returns the automatic and manual
// connections of a private endpoint's properties.
func azurePrivateEndpointConnections(props map[string]any) (auto, manual []any) {
	return azureARMSlice(props["privateLinkServiceConnections"]), azureARMSlice(props["manualPrivateLinkServiceConnections"])
}

func azurePrivateEndpointValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !azureNetworkIDOf(props["subnet"], "virtualnetworks/subnets") {
		return "properties.subnet.id must be a subnet ID."
	}
	auto, manual := azurePrivateEndpointConnections(props)
	if len(auto)+len(manual) == 0 {
		return "properties.privateLinkServiceConnections or manualPrivateLinkServiceConnections must list a connection."
	}
	for i, raw := range slices.Concat(auto, manual) {
		conn := azureARMObject(raw)
		cp := azureARMObject(conn["properties"])
		target := strings.ToLower(azureARMString(cp["privateLinkServiceId"]))
		at := fmt.Sprintf("connection %d", i)
		switch {
		case strings.TrimSpace(azureARMString(conn["name"])) == "":
			return at + " needs a name."
		case strings.HasSuffix(target, ".azure.privatelinkservice"):
			// A private link service alias.
		case !strings.Contains(target, "/providers/"):
			return at + ": properties.privateLinkServiceId must be a resource ID or private link service alias."
		case !strings.Contains(target, "/providers/microsoft.network/privatelinkservices/") && len(azureARMSlice(cp["groupIds"])) == 0:
			return at + ": properties.groupIds must name the target sub-resource, such as blob or vault."
		}
	}
	if name, present := props["customNetworkInterfaceName"]; present && !azureNICNamePattern.MatchString(azureARMString(name)) {
		return "properties.customNetworkInterfaceName must be 2 to 80 letters, digits, periods, underscores or hyphens."
	}
	for i, raw := range azureARMSlice(props["ipConfigurations"]) {
		config := azureARMObject(raw)
		cp := azureARMObject(config["properties"])
		if _, err := netip.ParseAddr(azureARMString(cp["privateIPAddress"])); err != nil || azureARMString(config["name"]) == "" || azureARMString(cp["groupId"]) == "" || azureARMString(cp["memberName"]) == "" {
			return fmt.Sprintf("properties.ipConfigurations[%d] needs a name, groupId, memberName and privateIPAddress.", i)
		}
	}
	return azureNetworkRefs(props, "applicationSecurityGroups", "applicationsecuritygroups")
}

// azurePrivateEndpointProperties gives each connection its ID and state:
// automatic connections are approved at once, manual ones wait.
func azurePrivateEndpointProperties(rec *azureARMResource) map[string]any {
	auto, manual := azurePrivateEndpointConnections(rec.Properties)
	decorate := func(key string, conns []any, state map[string]any) []any {
		out := make([]any, 0, len(conns))
		for _, raw := range conns {
			conn := azureARMObject(raw)
			name := azureARMString(conn["name"])
			cp := azureARMMerge(azureARMObject(conn["properties"]), map[string]any{"provisioningState": "Succeeded"})
			if _, present := cp["privateLinkServiceConnectionState"]; !present || key == "privateLinkServiceConnections" {
				cp["privateLinkServiceConnectionState"] = state
			}
			out = append(out, map[string]any{"id": rec.ID + "/" + key + "/" + name, "name": name, "type": "Microsoft.Network/privateEndpoints/" + key, "properties": cp})
		}
		return out
	}
	return map[string]any{
		"privateLinkServiceConnections":       decorate("privateLinkServiceConnections", auto, map[string]any{"status": "Approved", "description": "Auto-Approved", "actionsRequired": "None"}),
		"manualPrivateLinkServiceConnections": decorate("manualPrivateLinkServiceConnections", manual, map[string]any{"status": "Pending", "description": "Awaiting approval", "actionsRequired": "None"}),
	}
}

// azurePrivateEndpointDomains maps a target sub-resource to the domain its
// private name is in.
// ponytail: common targets only; others get no customDnsConfigs entry.
var azurePrivateEndpointDomains = map[string]string{
	"blob": "blob.core.windows.net", "file": "file.core.windows.net", "queue": "queue.core.windows.net",
	"table": "table.core.windows.net", "dfs": "dfs.core.windows.net", "vault": "vault.azure.net",
	"sqlserver": "database.windows.net", "registry": "azurecr.io", "namespace": "servicebus.windows.net",
	"rediscache": "redis.cache.windows.net", "sites": "azurewebsites.net", "configurationstores": "azconfig.io",
	"sql": "documents.azure.com", "postgresqlserver": "postgres.database.azure.com",
}

// azurePrivateEndpointLinked creates the endpoint's network interface,
// keeping its address across PUTs, and reports the address for each
// target name.
func azurePrivateEndpointLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	subnetID := azureARMString(azureARMObject(rec.Properties["subnet"])["id"])
	subnet := s.azureARMResources[azureARMKey(subnetID)]
	if subnet == nil {
		return nil, nil, "Subnet " + subnetID + " was not found."
	}
	nicName := azureARMString(rec.Properties["customNetworkInterfaceName"])
	if nicName == "" {
		nicName = rec.Name + ".nic." + uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String()
	}
	nicID := "/subscriptions/" + rec.SubscriptionID + "/resourceGroups/" + rec.ResourceGroup + "/providers/Microsoft.Network/networkInterfaces/" + nicName
	nicKey := azureARMKey(nicID)
	// The interface from the previous PUT, even if it is being renamed.
	var old *azureARMResource
	if existing := s.azureARMResources[azureARMKey(rec.ID)]; existing != nil && len(existing.LinkedIDs) > 0 {
		old = s.azureARMResources[azureARMKey(existing.LinkedIDs[0])]
	}
	if other := s.azureARMResources[nicKey]; other != nil && other != old {
		return nil, nil, "Network interface " + nicName + " already exists."
	}
	ownKey := ""
	if old != nil {
		ownKey = azureARMKey(old.ID)
	}
	used := s.azureSubnetUsedLocked(subnet, ownKey)

	method, address := "Dynamic", ""
	if configs := azureARMSlice(rec.Properties["ipConfigurations"]); len(configs) > 0 {
		method, address = "Static", azureARMString(azureARMObject(azureARMObject(configs[0])["properties"])["privateIPAddress"])
		switch {
		case !azureSubnetUsable(subnet, address):
			return nil, nil, "Private IP address " + address + " is not a usable address of subnet " + subnetID + "."
		case used[address]:
			return nil, nil, "Private IP address " + address + " is already in use."
		}
	} else if old != nil && azureNICSubnet(old, subnetID) && azureSubnetUsable(subnet, azureNICAddress(old)) {
		address = azureNICAddress(old)
	}
	if address == "" {
		var msg string
		if address, msg = azureSubnetFreeAddress(subnet, used); msg != "" {
			return nil, nil, msg
		}
	}

	auto, manual := azurePrivateEndpointConnections(rec.Properties)
	var groupID string
	var dns []any
	for _, raw := range slices.Concat(auto, manual) {
		cp := azureARMObject(azureARMObject(raw)["properties"])
		target := azureARMString(cp["privateLinkServiceId"])
		for _, group := range azureARMSlice(cp["groupIds"]) {
			groupID = azureARMString(group)
			if domain, ok := azurePrivateEndpointDomains[strings.ToLower(groupID)]; ok {
				name := strings.ToLower(target[strings.LastIndex(target, "/")+1:])
				dns = append(dns, map[string]any{"fqdn": name + "." + domain, "ipAddresses": []any{address}})
			}
		}
	}
	fqdns := make([]any, 0, len(dns))
	for _, entry := range dns {
		fqdns = append(fqdns, azureARMObject(entry)["fqdn"])
	}

	created := rec.CreatedAt
	if old != nil {
		created = old.CreatedAt
	}
	nicType, _ := lookupAzureARMResourceType("Microsoft.Network", "networkInterfaces")
	nic := &azureARMResource{
		ID: nicID, SubscriptionID: rec.SubscriptionID, ResourceGroup: rec.ResourceGroup, Name: nicName,
		Type: nicType.fullType(), Location: rec.Location,
		Properties: map[string]any{
			"ipConfigurations": []any{map[string]any{
				"name": "privateEndpointIpConfig." + uuid.NewSHA1(uuid.NameSpaceURL, []byte(nicKey)).String(),
				"properties": map[string]any{
					"privateIPAddress": address, "privateIPAllocationMethod": method, "subnet": map[string]any{"id": subnetID}, "primary": true,
					"privateLinkConnectionProperties": map[string]any{"groupId": groupID, "requiredMemberName": groupID, "fqdns": fqdns},
				},
			}},
			"privateEndpoint": map[string]any{"id": rec.ID},
		},
		CreatedAt: created, UpdatedAt: rec.UpdatedAt,
	}
	nic.Properties = azureARMWithState(nicType, azureARMProperties(nicType, nic), nil)

	props := map[string]any{"networkInterfaces": []any{map[string]any{"id": nicID}}}
	if dns != nil {
		props["customDnsConfigs"] = dns
	}
	return props, []*azureARMResource{nic}, ""
}

func azureNICSubnet(nic *azureARMResource, subnetID string) bool {
	for _, raw := range azureARMSlice(nic.Properties["ipConfigurations"]) {
		if strings.EqualFold(azureARMString(azureARMObject(azureARMObject(azureARMObject(raw)["properties"])["subnet"])["id"]), subnetID) {
			return true
		}
	}
	return false
}

func azureNICAddress(nic *azureARMResource) string {
	configs := azureARMSlice(nic.Properties["ipConfigurations"])
	if len(configs) == 0 {
		return ""
	}
	return azureARMString(azureARMObject(azureARMObject(configs[0])["properties"])["privateIPAddress"])
}

// azureSubnetUsable reports whether address is in the subnet and not one
// of the five Azure reserves: the first four and the last of each prefix.
func azureSubnetUsable(subnet *azureARMResource, address string) bool {
	addr, err := netip.ParseAddr(address)
	if err != nil {
		return false
	}
	prefixes, _ := azureNetworkPrefixes(subnet.Properties)
	for _, prefix := range prefixes {
		first := prefix.Addr().Next().Next().Next().Next()
		if prefix.Contains(addr) && addr.Compare(first) >= 0 && prefix.Contains(addr.Next()) {
			return true
		}
	}
	return false
}

// azureSubnetUsedLocked returns the addresses stored network interfaces
// use in the subnet, other than the one stored at skipKey.
// ponytail: scans every stored record; index NICs by subnet if that gets slow.
func (s *Server) azureSubnetUsedLocked(subnet *azureARMResource, skipKey string) map[string]bool {
	used := map[string]bool{}
	for key, rec := range s.azureARMResources {
		if key == skipKey || !strings.EqualFold(rec.Type, "Microsoft.Network/networkInterfaces") {
			continue
		}
		for _, raw := range azureARMSlice(rec.Properties["ipConfigurations"]) {
			cp := azureARMObject(azureARMObject(raw)["properties"])
			if strings.EqualFold(azureARMString(azureARMObject(cp["subnet"])["id"]), subnet.ID) {
				used[azureARMString(cp["privateIPAddress"])] = true
			}
		}
	}
	return used
}

// azureSubnetFreeAddress returns the lowest usable IPv4 address of the
// subnet not in used.
func azureSubnetFreeAddress(subnet *azureARMResource, used map[string]bool) (string, string) {
	prefixes, _ := azureNetworkPrefixes(subnet.Properties)
	for _, prefix := range prefixes {
		if !prefix.Addr().Is4() || prefix.Bits() > 29 {
			continue
		}
		addr := prefix.Addr().Next().Next().Next().Next()
		for ; prefix.Contains(addr.Next()); addr = addr.Next() {
			if !used[addr.String()] {
				return addr.String(), ""
			}
		}
	}
	return "", "Subnet " + subnet.ID + " has no free IPv4 addresses."
}

func azurePrivateDNSZoneGroupValidate(payload map[string]any) string {
	configs := azureARMSlice(azureARMObject(payload["properties"])["privateDnsZoneConfigs"])
	if len(configs) == 0 || len(configs) > 5 {
		return "properties.privateDnsZoneConfigs must list 1 to 5 private DNS zones."
	}
	for i, raw := range configs {
		config := azureARMObject(raw)
		zone := strings.ToLower(azureARMString(azureARMObject(config["properties"])["privateDnsZoneId"]))
		if azureARMString(config["name"]) == "" || !strings.Contains(zone, "/providers/microsoft.network/privatednszones/") {
			return fmt.Sprintf("properties.privateDnsZoneConfigs[%d] needs a name and a private DNS zone privateDnsZoneId.", i)
		}
	}
	return ""
}

func azurePrivateLinkServiceValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	frontends := azureARMSlice(props["loadBalancerFrontendIpConfigurations"])
	if len(frontends) == 0 {
		return "properties.loadBalancerFrontendIpConfigurations must list a load balancer frontend."
	}
	if msg := azureNetworkRefs(props, "loadBalancerFrontendIpConfigurations", "loadbalancers/frontendipconfigurations"); msg != "" {
		return msg
	}
	configs := azureARMSlice(props["ipConfigurations"])
	if len(configs) == 0 || len(configs) > 8 {
		return "properties.ipConfigurations must list 1 to 8 IP configurations."
	}
	primaries := 0
	for i, raw := range configs {
		config := azureARMObject(raw)
		cp := azureARMObject(config["properties"])
		if azureARMString(config["name"]) == "" || !azureNetworkIDOf(cp["subnet"], "virtualnetworks/subnets") {
			return fmt.Sprintf("properties.ipConfigurations[%d] needs a name and a subnet ID.", i)
		}
		if method, present := cp["privateIPAllocationMethod"]; present && method != "Dynamic" && method != "Static" {
			return fmt.Sprintf("properties.ipConfigurations[%d].properties.privateIPAllocationMethod must be Dynamic or Static.", i)
		}
		if cp["primary"] == true {
			primaries++
		}
	}
	if len(configs) > 1 && primaries != 1 {
		return "Exactly one IP configuration must be primary."
	}
	for _, key := range []string{"visibility", "autoApproval"} {
		for _, sub := range azureARMSlice(azureARMObject(props[key])["subscriptions"]) {
			if _, err := uuid.Parse(azureARMString(sub)); err != nil && !(key == "visibility" && sub == "*") {
				return "properties." + key + ".subscriptions must list subscription IDs."
			}
		}
	}
	return ""
}

// azurePrivateLinkServiceProperties gives the service its alias and each IP
// configuration its ID.
func azurePrivateLinkServiceProperties(rec *azureARMResource) map[string]any {
	guid := uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String()
	configs := azureARMSlice(rec.Properties["ipConfigurations"])
	out := make([]any, 0, len(configs))
	for _, raw := range configs {
		config := azureARMObject(raw)
		cp := azureARMMerge(map[string]any{"privateIPAllocationMethod": "Dynamic", "privateIPAddressVersion": "IPv4", "primary": len(configs) == 1}, azureARMObject(config["properties"]))
		cp["provisioningState"] = "Succeeded"
		name := azureARMString(config["name"])
		out = append(out, map[string]any{"id": rec.ID + "/ipConfigurations/" + name, "name": name, "type": "Microsoft.Network/privateLinkServices/ipConfigurations", "properties": cp})
	}
	return map[string]any{
		"alias":                      rec.Name + "." + guid + "." + azureLocationHost(rec.Location) + ".azure.privatelinkservice",
		"ipConfigurations":           out,
		"privateEndpointConnections": []any{},
	}
}

// azurePrivateLinkServiceVisibility reports whether a private link
// service has the alias.
func (s *Server) azurePrivateLinkServiceVisibility(payload map[string]any) (int, any) {
	alias := azureARMString(payload["privateLinkServiceAlias"])
	if alias == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "privateLinkServiceAlias is required.")
	}
	found := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, "Microsoft.Network/privateLinkServices") && strings.EqualFold(azureARMString(rec.Properties["alias"]), alias)
	})
	return http.StatusOK, map[string]any{"visible": len(found) > 0}
}
