package server

import (
	"crypto/sha256"
	"fmt"
	"net/netip"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Microsoft.Network virtual networks, subnets and peerings, network security
// groups and rules, public IP addresses, network interfaces, load balancers
// and route tables, served by the shared ARM handler. Subnets, security
// rules and routes are both inline properties and child resources, kept in
// sync by the handler. No packets flow: addresses are bookkeeping only.
func init() {
	const ns = "Microsoft.Network"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "virtualNetworks",
		Validate:           azureVNetValidate,
		ReadOnlyProperties: azureNetworkResourceGUID,
		InlineChildren:     map[string]string{"subnets": "virtualNetworks/subnets", "virtualNetworkPeerings": "virtualNetworks/virtualNetworkPeerings"},
		// DNS Private Resolver list operations (provider_azure_dns_dnsresolver_2025_05_01.go).
		ServerActions: dnsrVNetActions,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "virtualNetworks/subnets", Proxy: true,
		Validate:           azureSubnetValidate,
		ValidateInParent:   azureSubnetValidateInVNet,
		ReadOnlyProperties: azureSubnetProperties,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "virtualNetworks/virtualNetworkPeerings", Proxy: true,
		Validate:           azureVNetPeeringValidate,
		ReadOnlyProperties: azureVNetPeeringProperties,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "networkSecurityGroups",
		ReadOnlyProperties: azureNSGProperties,
		InlineChildren:     map[string]string{"securityRules": "networkSecurityGroups/securityRules"},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "networkSecurityGroups/securityRules", Proxy: true,
		Validate:         azureSecurityRuleValidate,
		ValidateInParent: azureSecurityRuleValidateInNSG,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "publicIPAddresses",
		Validate:           azurePublicIPValidate,
		ReadOnlyProperties: azurePublicIPProperties,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "networkInterfaces",
		Validate:           azureNICValidate,
		ReadOnlyProperties: azureNICProperties,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "loadBalancers",
		Validate:           azureLoadBalancerValidate,
		ReadOnlyProperties: azureLoadBalancerProperties,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "routeTables",
		ReadOnlyProperties: azureNetworkResourceGUID,
		InlineChildren:     map[string]string{"routes": "routeTables/routes"},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "routeTables/routes", Proxy: true,
		Validate: azureRouteValidate,
	})
}

func azureNetworkResourceGUID(rec *azureARMResource) map[string]any {
	return map[string]any{"resourceGuid": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String()}
}

// azureNetworkIDOf reports whether value is an object whose "id" is a
// Microsoft.Network resource ID of the given lower-cased type path, such as
// "virtualnetworks/subnets".
func azureNetworkIDOf(value any, typePath string) bool {
	id := strings.ToLower(azureARMString(azureARMObject(value)["id"]))
	parts := strings.Split(typePath, "/")
	pattern := "/providers/microsoft.network/" + parts[0] + "/[^/]+"
	for _, part := range parts[1:] {
		pattern += "/" + part + "/[^/]+"
	}
	matched, _ := regexp.MatchString(pattern+"$", id)
	return matched
}

// azureNetworkOptionalRef checks that props[key], when set, references a
// resource of typePath.
func azureNetworkOptionalRef(props map[string]any, key, typePath string) string {
	if value, present := props[key]; present && !azureNetworkIDOf(value, typePath) {
		return "properties." + key + ".id must be a " + typePath + " resource ID."
	}
	return ""
}

// azureNetworkPrefix parses a canonical CIDR prefix such as 10.0.0.0/16.
func azureNetworkPrefix(raw any) (netip.Prefix, bool) {
	prefix, err := netip.ParsePrefix(azureARMString(raw))
	return prefix, err == nil && prefix == prefix.Masked()
}

// azureNetworkPrefixes returns the addressPrefix and addressPrefixes of a
// subnet's properties, and false if any is not a canonical CIDR prefix.
func azureNetworkPrefixes(props map[string]any) ([]netip.Prefix, bool) {
	raws, _ := props["addressPrefixes"].([]any)
	if single, present := props["addressPrefix"]; present {
		raws = append([]any{single}, raws...)
	}
	out := make([]netip.Prefix, 0, len(raws))
	for _, raw := range raws {
		prefix, ok := azureNetworkPrefix(raw)
		if !ok {
			return nil, false
		}
		out = append(out, prefix)
	}
	return out, true
}

func azureVNetValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	raws, _ := azureARMObject(props["addressSpace"])["addressPrefixes"].([]any)
	if len(raws) == 0 {
		return "properties.addressSpace.addressPrefixes must list at least one CIDR prefix."
	}
	for _, raw := range raws {
		if _, ok := azureNetworkPrefix(raw); !ok {
			return "properties.addressSpace.addressPrefixes has an invalid CIDR prefix: " + azureARMString(raw) + "."
		}
	}
	servers, _ := azureARMObject(props["dhcpOptions"])["dnsServers"].([]any)
	for _, raw := range servers {
		if _, err := netip.ParseAddr(azureARMString(raw)); err != nil {
			return "properties.dhcpOptions.dnsServers has an invalid IP address: " + azureARMString(raw) + "."
		}
	}
	return ""
}

func azureSubnetValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	prefixes, ok := azureNetworkPrefixes(props)
	if !ok || len(prefixes) == 0 {
		return "properties.addressPrefix or addressPrefixes must give canonical CIDR prefixes."
	}
	for _, ref := range [][2]string{{"networkSecurityGroup", "networksecuritygroups"}, {"routeTable", "routetables"}, {"natGateway", "natgateways"}} {
		if msg := azureNetworkOptionalRef(props, ref[0], ref[1]); msg != "" {
			return msg
		}
	}
	delegations, _ := props["delegations"].([]any)
	for _, raw := range delegations {
		if azureARMString(azureARMObject(azureARMObject(raw)["properties"])["serviceName"]) == "" {
			return "Each delegation needs properties.serviceName."
		}
	}
	return azureEnumProperty("privateEndpointNetworkPolicies", "Enabled", "Disabled", "NetworkSecurityGroupEnabled", "RouteTableEnabled")(payload)
}

// azureSubnetValidateInVNet requires the subnet's prefixes to lie inside the
// network's address space and not overlap its other subnets.
func azureSubnetValidateInVNet(vnet map[string]any, name string, payload map[string]any) string {
	prefixes, _ := azureNetworkPrefixes(azureARMObject(payload["properties"]))
	space, _ := azureARMObject(vnet["addressSpace"])["addressPrefixes"].([]any)
	for _, prefix := range prefixes {
		inside := slices.ContainsFunc(space, func(raw any) bool {
			outer, ok := azureNetworkPrefix(raw)
			return ok && outer.Bits() <= prefix.Bits() && outer.Contains(prefix.Addr())
		})
		if !inside {
			return "Subnet prefix " + prefix.String() + " is not within the virtual network's address space."
		}
	}
	siblings, _ := vnet["subnets"].([]any)
	for _, raw := range siblings {
		sibling := azureARMObject(raw)
		if strings.EqualFold(azureARMString(sibling["name"]), name) {
			continue
		}
		theirs, _ := azureNetworkPrefixes(azureARMObject(sibling["properties"]))
		for _, prefix := range prefixes {
			for _, other := range theirs {
				if prefix.Overlaps(other) {
					return "Subnet prefix " + prefix.String() + " overlaps subnet " + azureARMString(sibling["name"]) + " (" + other.String() + ")."
				}
			}
		}
	}
	return ""
}

func azureSubnetProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{}
	for key, value := range map[string]any{"privateEndpointNetworkPolicies": "Disabled", "privateLinkServiceNetworkPolicies": "Enabled"} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}

func azureVNetPeeringValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !azureNetworkIDOf(props["remoteVirtualNetwork"], "virtualnetworks") {
		return "properties.remoteVirtualNetwork.id must be a virtual network ID."
	}
	if props["useRemoteGateways"] == true && props["allowGatewayTransit"] == true {
		return "A peering cannot both use remote gateways and allow gateway transit."
	}
	return ""
}

// azureVNetPeeringProperties reports peerings as connected and in sync at
// once, whether or not the remote network peers back.
func azureVNetPeeringProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{"peeringState": "Connected", "peeringSyncLevel": "FullyInSync"}
	for key, value := range map[string]any{"allowVirtualNetworkAccess": true, "allowForwardedTraffic": false, "allowGatewayTransit": false, "useRemoteGateways": false} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}

var (
	azureSecurityRuleEnums = map[string][]string{
		"protocol":  {"Tcp", "Udp", "Icmp", "Esp", "Ah", "*"},
		"access":    {"Allow", "Deny"},
		"direction": {"Inbound", "Outbound"},
	}
	azureNetworkPortPattern = regexp.MustCompile(`^(\*|\d{1,5}(-\d{1,5})?)$`)
)

func azureNetworkPortRange(raw string) bool {
	if !azureNetworkPortPattern.MatchString(raw) {
		return false
	}
	if raw == "*" {
		return true
	}
	low, high, isRange := strings.Cut(raw, "-")
	lo, _ := strconv.Atoi(low)
	hi := lo
	if isRange {
		hi, _ = strconv.Atoi(high)
	}
	return lo <= hi && hi <= 65535
}

func azureSecurityRuleValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, key := range []string{"protocol", "access", "direction"} {
		if msg := azureEnumRequired(key, azureSecurityRuleEnums[key]...)(payload); msg != "" {
			return msg
		}
	}
	if priority, ok := props["priority"].(float64); !ok || priority != float64(int(priority)) || priority < 100 || priority > 4096 {
		return "properties.priority must be an integer from 100 to 4096."
	}
	for _, side := range []string{"source", "destination"} {
		ranges, _ := props[side+"PortRanges"].([]any)
		if single, present := props[side+"PortRange"]; present {
			ranges = append([]any{single}, ranges...)
		}
		if len(ranges) == 0 {
			return "properties." + side + "PortRange or " + side + "PortRanges is required."
		}
		for _, raw := range ranges {
			if !azureNetworkPortRange(azureARMString(raw)) {
				return "properties." + side + "PortRange has an invalid port or range: " + azureARMString(raw) + "."
			}
		}
		prefixes, _ := props[side+"AddressPrefixes"].([]any)
		groups, _ := props[side+"ApplicationSecurityGroups"].([]any)
		if azureARMString(props[side+"AddressPrefix"]) == "" && len(prefixes) == 0 && len(groups) == 0 {
			return "properties." + side + "AddressPrefix, " + side + "AddressPrefixes or " + side + "ApplicationSecurityGroups is required."
		}
	}
	return ""
}

// azureSecurityRuleValidateInNSG requires each priority to be unique per
// direction within the group.
func azureSecurityRuleValidateInNSG(nsg map[string]any, name string, payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	siblings, _ := nsg["securityRules"].([]any)
	for _, raw := range siblings {
		sibling := azureARMObject(raw)
		theirs := azureARMObject(sibling["properties"])
		if !strings.EqualFold(azureARMString(sibling["name"]), name) && theirs["priority"] == props["priority"] && theirs["direction"] == props["direction"] {
			return fmt.Sprintf("Rule %s already uses priority %v for %s traffic.", azureARMString(sibling["name"]), props["priority"], azureARMString(props["direction"]))
		}
	}
	return ""
}

// azureNSGDefaultRules are the rules every network security group has.
var azureNSGDefaultRules = []struct {
	name, direction, source, destination, access string
	priority                                     int
}{
	{"AllowVnetInBound", "Inbound", "VirtualNetwork", "VirtualNetwork", "Allow", 65000},
	{"AllowAzureLoadBalancerInBound", "Inbound", "AzureLoadBalancer", "*", "Allow", 65001},
	{"DenyAllInBound", "Inbound", "*", "*", "Deny", 65500},
	{"AllowVnetOutBound", "Outbound", "VirtualNetwork", "VirtualNetwork", "Allow", 65000},
	{"AllowInternetOutBound", "Outbound", "*", "Internet", "Allow", 65001},
	{"DenyAllOutBound", "Outbound", "*", "*", "Deny", 65500},
}

func azureNSGProperties(rec *azureARMResource) map[string]any {
	rules := make([]any, 0, len(azureNSGDefaultRules))
	for _, rule := range azureNSGDefaultRules {
		rules = append(rules, map[string]any{
			"id":   rec.ID + "/defaultSecurityRules/" + rule.name,
			"name": rule.name,
			"type": "Microsoft.Network/networkSecurityGroups/defaultSecurityRules",
			"properties": map[string]any{
				"protocol": "*", "sourcePortRange": "*", "destinationPortRange": "*",
				"sourceAddressPrefix": rule.source, "destinationAddressPrefix": rule.destination,
				"access": rule.access, "priority": rule.priority, "direction": rule.direction,
				"provisioningState": "Succeeded",
			},
		})
	}
	return azureARMMerge(azureNetworkResourceGUID(rec), map[string]any{"defaultSecurityRules": rules})
}

var azureNetworkDNSLabelPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,61}[a-z0-9]$`)

func azurePublicIPValidate(payload map[string]any) string {
	sku := azureARMObject(payload["sku"])
	if name, present := sku["name"]; present && name != "Basic" && name != "Standard" {
		return "sku.name must be Basic or Standard."
	}
	if tier, present := sku["tier"]; present && tier != "Regional" && tier != "Global" {
		return "sku.tier must be Regional or Global."
	}
	props := azureARMObject(payload["properties"])
	if msg := azureEnumProperty("publicIPAllocationMethod", "Static", "Dynamic")(payload); msg != "" {
		return msg
	}
	if msg := azureEnumProperty("publicIPAddressVersion", "IPv4", "IPv6")(payload); msg != "" {
		return msg
	}
	if sku["name"] == "Standard" && props["publicIPAllocationMethod"] == "Dynamic" {
		return "Standard SKU public IP addresses must use Static allocation."
	}
	if raw, present := props["idleTimeoutInMinutes"]; present {
		if minutes, ok := raw.(float64); !ok || minutes != float64(int(minutes)) || minutes < 4 || minutes > 30 {
			return "properties.idleTimeoutInMinutes must be an integer from 4 to 30."
		}
	}
	if label, present := azureARMObject(props["dnsSettings"])["domainNameLabel"]; present && !azureNetworkDNSLabelPattern.MatchString(azureARMString(label)) {
		return "properties.dnsSettings.domainNameLabel must be 3-63 lowercase letters, digits or hyphens, starting with a letter."
	}
	return ""
}

// azurePublicIPProperties allocates a stable address to static IPs (dynamic
// ones get theirs only when attached in Azure, so here never) and reports
// the DNS name for a domain label.
func azurePublicIPProperties(rec *azureARMResource) map[string]any {
	props := azureNetworkResourceGUID(rec)
	method := azureARMString(rec.Properties["publicIPAllocationMethod"])
	if method == "" {
		method = "Dynamic"
		if rec.SKU["name"] == "Standard" {
			method = "Static"
		}
		props["publicIPAllocationMethod"] = method
	}
	version := azureARMString(rec.Properties["publicIPAddressVersion"])
	if version == "" {
		version = "IPv4"
		props["publicIPAddressVersion"] = version
	}
	if _, present := rec.Properties["idleTimeoutInMinutes"]; !present {
		props["idleTimeoutInMinutes"] = 4
	}
	if method == "Static" {
		sum := sha256.Sum256([]byte(azureARMKey(rec.ID)))
		if version == "IPv6" {
			props["ipAddress"] = fmt.Sprintf("2603:1030:%x:%x::%x", sum[0], sum[1], 1+int(sum[2])%254)
		} else {
			props["ipAddress"] = fmt.Sprintf("20.%d.%d.%d", sum[0], sum[1], 1+int(sum[2])%254)
		}
	}
	if label := azureARMString(azureARMObject(rec.Properties["dnsSettings"])["domainNameLabel"]); label != "" {
		props["dnsSettings"] = map[string]any{"fqdn": label + "." + azureLocationHost(rec.Location) + ".cloudapp.azure.com"}
	}
	return props
}

func azureNICValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	configs, _ := props["ipConfigurations"].([]any)
	if len(configs) == 0 {
		return "properties.ipConfigurations must list at least one IP configuration."
	}
	primaries := 0
	for i, raw := range configs {
		config := azureARMObject(raw)
		at := fmt.Sprintf("properties.ipConfigurations[%d]", i)
		cp := azureARMObject(config["properties"])
		switch {
		case strings.TrimSpace(azureARMString(config["name"])) == "":
			return at + ".name is required."
		case !azureNetworkIDOf(cp["subnet"], "virtualnetworks/subnets"):
			return at + ".properties.subnet.id must be a subnet ID."
		}
		if msg := azureNetworkOptionalRef(cp, "publicIPAddress", "publicipaddresses"); msg != "" {
			return at + ": " + msg
		}
		switch azureARMString(cp["privateIPAllocationMethod"]) {
		case "", "Dynamic":
		case "Static":
			if _, err := netip.ParseAddr(azureARMString(cp["privateIPAddress"])); err != nil {
				return at + ".properties.privateIPAddress must be an IP address for Static allocation."
			}
		default:
			return at + ".properties.privateIPAllocationMethod must be Dynamic or Static."
		}
		if cp["primary"] == true {
			primaries++
		}
	}
	if len(configs) > 1 && primaries != 1 {
		return "Exactly one IP configuration must be primary."
	}
	return azureNetworkOptionalRef(props, "networkSecurityGroup", "networksecuritygroups")
}

// azureNICProperties gives each IP configuration its ID and defaults, and
// the interface a stable MAC address.
func azureNICProperties(rec *azureARMResource) map[string]any {
	sum := sha256.Sum256([]byte(azureARMKey(rec.ID)))
	props := azureARMMerge(azureNetworkResourceGUID(rec), map[string]any{
		"macAddress": fmt.Sprintf("00-0D-3A-%02X-%02X-%02X", sum[0], sum[1], sum[2]),
		"nicType":    "Standard",
	})
	for _, key := range []string{"enableAcceleratedNetworking", "enableIPForwarding"} {
		if _, present := rec.Properties[key]; !present {
			props[key] = false
		}
	}
	configs, _ := rec.Properties["ipConfigurations"].([]any)
	out := make([]any, 0, len(configs))
	for _, raw := range configs {
		config := azureARMObject(raw)
		cp := azureARMMerge(map[string]any{"privateIPAllocationMethod": "Dynamic", "privateIPAddressVersion": "IPv4", "primary": false}, azureARMObject(config["properties"]))
		cp["provisioningState"] = "Succeeded"
		if len(configs) == 1 {
			cp["primary"] = true
		}
		name := azureARMString(config["name"])
		out = append(out, map[string]any{"id": rec.ID + "/ipConfigurations/" + name, "name": name, "type": "Microsoft.Network/networkInterfaces/ipConfigurations", "properties": cp})
	}
	props["ipConfigurations"] = out
	return props
}

// azureLoadBalancerCollections are the load balancer's inline collections.
var azureLoadBalancerCollections = []string{"frontendIPConfigurations", "backendAddressPools", "probes", "loadBalancingRules", "inboundNatRules"}

func azureLoadBalancerValidate(payload map[string]any) string {
	if name, present := azureARMObject(payload["sku"])["name"]; present && !slices.Contains([]string{"Basic", "Standard", "Gateway"}, azureARMString(name)) {
		return "sku.name must be Basic, Standard or Gateway."
	}
	props := azureARMObject(payload["properties"])
	names := map[string]map[string]bool{}
	for _, collection := range azureLoadBalancerCollections {
		names[collection] = map[string]bool{}
		items, _ := props[collection].([]any)
		for i, raw := range items {
			name := strings.ToLower(strings.TrimSpace(azureARMString(azureARMObject(raw)["name"])))
			if name == "" || names[collection][name] {
				return fmt.Sprintf("properties.%s[%d] needs a unique name.", collection, i)
			}
			names[collection][name] = true
		}
	}
	// ref checks that an item's reference names an item of this load balancer.
	ref := func(item map[string]any, key, collection string) bool {
		id := azureARMString(azureARMObject(item[key])["id"])
		head, name, found := strings.Cut(strings.ToLower(id), "/"+strings.ToLower(collection)+"/")
		return found && strings.Contains(head, "/providers/microsoft.network/loadbalancers/") && names[collection][name]
	}
	for i, raw := range azureARMSlice(props["frontendIPConfigurations"]) {
		fp := azureARMObject(azureARMObject(raw)["properties"])
		targets := 0
		for _, target := range [][2]string{{"publicIPAddress", "publicipaddresses"}, {"publicIPPrefix", "publicipprefixes"}, {"subnet", "virtualnetworks/subnets"}} {
			if azureNetworkIDOf(fp[target[0]], target[1]) {
				targets++
			}
		}
		if targets != 1 {
			return fmt.Sprintf("properties.frontendIPConfigurations[%d] needs exactly one of publicIPAddress, publicIPPrefix or subnet.", i)
		}
	}
	for i, raw := range azureARMSlice(props["probes"]) {
		pp := azureARMObject(azureARMObject(raw)["properties"])
		protocol := azureARMString(pp["protocol"])
		switch {
		case !slices.Contains([]string{"Tcp", "Http", "Https"}, protocol):
			return fmt.Sprintf("properties.probes[%d].properties.protocol must be Tcp, Http or Https.", i)
		case !azureNetworkPort(pp["port"], 1):
			return fmt.Sprintf("properties.probes[%d].properties.port must be from 1 to 65535.", i)
		case protocol != "Tcp" && !strings.HasPrefix(azureARMString(pp["requestPath"]), "/"):
			return fmt.Sprintf("properties.probes[%d].properties.requestPath is required for %s probes.", i, protocol)
		}
	}
	for _, collection := range []string{"loadBalancingRules", "inboundNatRules"} {
		for i, raw := range azureARMSlice(props[collection]) {
			rp := azureARMObject(azureARMObject(raw)["properties"])
			at := fmt.Sprintf("properties.%s[%d].properties", collection, i)
			switch {
			case !slices.Contains([]string{"Tcp", "Udp", "All"}, azureARMString(rp["protocol"])):
				return at + ".protocol must be Tcp, Udp or All."
			case !azureNetworkPort(rp["backendPort"], 0):
				return at + ".backendPort must be from 0 to 65535."
			case !azureLoadBalancerFrontendPorts(rp):
				return at + " needs a frontendPort, or for an inbound NAT rule on a backend pool frontendPortRangeStart and frontendPortRangeEnd, from 0 to 65535."
			case !ref(rp, "frontendIPConfiguration", "frontendIPConfigurations"):
				return at + ".frontendIPConfiguration.id must reference a frontend of this load balancer."
			}
			if _, present := rp["backendAddressPool"]; present && !ref(rp, "backendAddressPool", "backendAddressPools") {
				return at + ".backendAddressPool.id must reference a backend pool of this load balancer."
			}
			if _, present := rp["probe"]; present && !ref(rp, "probe", "probes") {
				return at + ".probe.id must reference a probe of this load balancer."
			}
		}
	}
	return ""
}

// azureLoadBalancerFrontendPorts accepts a single frontendPort or, for
// inbound NAT rules mapping a backend pool, a port range.
func azureLoadBalancerFrontendPorts(rule map[string]any) bool {
	if _, ranged := rule["frontendPortRangeStart"]; ranged {
		start, end := rule["frontendPortRangeStart"], rule["frontendPortRangeEnd"]
		return azureNetworkPort(start, 1) && azureNetworkPort(end, 1) && start.(float64) <= end.(float64)
	}
	return azureNetworkPort(rule["frontendPort"], 0)
}

func azureARMSlice(raw any) []any {
	items, _ := raw.([]any)
	return items
}

func azureNetworkPort(raw any, least float64) bool {
	port, ok := raw.(float64)
	return ok && port == float64(int(port)) && port >= least && port <= 65535
}

// azureLoadBalancerProperties gives every inline item its ID.
func azureLoadBalancerProperties(rec *azureARMResource) map[string]any {
	props := azureNetworkResourceGUID(rec)
	for _, collection := range azureLoadBalancerCollections {
		items := azureARMSlice(rec.Properties[collection])
		if items == nil {
			continue
		}
		out := make([]any, 0, len(items))
		for _, raw := range items {
			item := azureARMObject(raw)
			name := azureARMString(item["name"])
			out = append(out, azureARMMerge(item, map[string]any{
				"id":         rec.ID + "/" + collection + "/" + name,
				"type":       "Microsoft.Network/loadBalancers/" + collection,
				"properties": map[string]any{"provisioningState": "Succeeded"},
			}))
		}
		props[collection] = out
	}
	return props
}

var (
	azureRouteNextHops      = []string{"VirtualNetworkGateway", "VnetLocal", "Internet", "VirtualAppliance", "None"}
	azureNetworkServiceTags = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9.]*$`)
)

func azureRouteValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if _, ok := azureNetworkPrefix(props["addressPrefix"]); !ok && !azureNetworkServiceTags.MatchString(azureARMString(props["addressPrefix"])) {
		return "properties.addressPrefix must be a CIDR prefix or a service tag."
	}
	hop := azureARMString(props["nextHopType"])
	if !slices.Contains(azureRouteNextHops, hop) {
		return "properties.nextHopType must be one of " + strings.Join(azureRouteNextHops, ", ") + "."
	}
	_, err := netip.ParseAddr(azureARMString(props["nextHopIpAddress"]))
	present := props["nextHopIpAddress"] != nil
	switch {
	case hop == "VirtualAppliance" && err != nil:
		return "properties.nextHopIpAddress must be an IP address for a VirtualAppliance next hop."
	case hop != "VirtualAppliance" && present:
		return "properties.nextHopIpAddress is allowed only for a VirtualAppliance next hop."
	}
	return ""
}
