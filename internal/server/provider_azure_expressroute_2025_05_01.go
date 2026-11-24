package server

import (
	"encoding/base64"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Azure ExpressRoute (Microsoft.Network 2025-05-01) on the shared ARM
// handler: circuits with their peerings, global-reach connections and
// authorizations, provider cross connections and their peerings,
// ExpressRoute Direct ports with links and authorizations, route filters
// and rules, and service endpoint policies and definitions, plus the
// provider, location, port and BGP community catalogs. Circuit statistics,
// ARP and route tables are in azure_expressroute_2025_05_01_tables.go. No
// traffic flows: provisioning completes at once and tables are synthesized
// from the configured peerings.

const azureExpressRoute20250501APIVersion = "2025-05-01"

// xrBandwidthsMbps are the provider circuit bandwidths on offer.
var xrBandwidthsMbps = []float64{50, 100, 200, 500, 1000, 2000, 5000, 10000}

func init() {
	ns := "Microsoft.Network"
	peering := azureARMResourceType{
		Namespace: ns, Type: "expressRouteCircuits/peerings", Proxy: true,
		Validate:           xrValidatePeering,
		ValidateInParent:   xrPeeringNameMatchesType,
		ReadOnlyProperties: xrPeeringProperties,
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "expressRouteCircuits",
		Validate:           xrValidateCircuit,
		ReadOnlyProperties: xrCircuitProperties,
		InlineChildren:     map[string]string{"peerings": "expressRouteCircuits/peerings", "authorizations": "expressRouteCircuits/authorizations"},
	})
	registerAzureARMResourceType(peering)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "expressRouteCircuits/authorizations", Proxy: true,
		ReadOnlyProperties: xrAuthorizationProperties,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "expressRouteCircuits/peerings/connections", Proxy: true,
		Validate: xrValidateCircuitConnection,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"circuitConnectionStatus": "Connected"}
		},
	})
	registerAzureARMView(ns, "expressRouteCircuits/peerings/peerConnections", xrPeerConnections)
	crossPeering := peering
	crossPeering.Type = "expressRouteCrossConnections/peerings"
	registerAzureARMResourceType(crossPeering)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "expressRouteCrossConnections",
		Validate:           xrValidateCrossConnection,
		ReadOnlyProperties: xrCrossConnectionProperties,
		InlineChildren:     map[string]string{"peerings": "expressRouteCrossConnections/peerings"},
		NoDelete:           true, // Providers' cross connections go with the customer's circuit.
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "expressRoutePorts",
		Validate:           xrValidatePort,
		ReadOnlyProperties: xrPortProperties,
		Actions:            map[string]azureARMAction{"generateloa": xrGenerateLOA},
	})
	registerAzureARMView(ns, "expressRoutePorts/links", func(s *Server, scope string) []map[string]any {
		port := s.azureARMLookup(scope)
		if port == nil {
			return nil
		}
		var out []map[string]any
		for _, link := range azureARMSlice(port.Properties["links"]) {
			out = append(out, azureARMObject(link))
		}
		return out
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "expressRoutePorts/authorizations", Proxy: true,
		ReadOnlyProperties: xrAuthorizationProperties,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "routeFilters",
		InlineChildren: map[string]string{"rules": "routeFilters/routeFilterRules"},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "routeFilters/routeFilterRules", Proxy: true, KeepLocation: true,
		Validate: xrValidateRouteFilterRule,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "serviceEndpointPolicies",
		ReadOnlyProperties: azureNetworkResourceGUID,
		InlineChildren:     map[string]string{"serviceEndpointPolicyDefinitions": "serviceEndpointPolicies/serviceEndpointPolicyDefinitions"},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "serviceEndpointPolicies/serviceEndpointPolicyDefinitions", Proxy: true,
		Validate: xrValidatePolicyDefinition,
	})
	registerAzureARMView(ns, "ExpressRoutePortsLocations", func(_ *Server, scope string) []map[string]any {
		var out []map[string]any
		for _, loc := range []struct{ name, address string }{{"Equinix-Ashburn-DC2", "21715 Filigree Ct, Ashburn, VA"}, {"Equinix-Amsterdam-AM5", "Johan Huizingalaan 759, Amsterdam"}} {
			out = append(out, map[string]any{
				"id": scope + "/providers/" + ns + "/ExpressRoutePortsLocations/" + loc.name, "name": loc.name, "type": ns + "/ExpressRoutePortsLocations",
				"properties": map[string]any{
					"address": loc.address, "contact": "support@equinix.com", "provisioningState": "Succeeded",
					"availableBandwidths": []any{map[string]any{"offerName": "10 Gbps", "valueInGbps": 10}, map[string]any{"offerName": "100 Gbps", "valueInGbps": 100}},
				},
			})
		}
		return out
	})
	registerAzureARMView(ns, "expressRouteServiceProviders", func(_ *Server, scope string) []map[string]any {
		var out []map[string]any
		for _, provider := range []string{"Equinix", "AT&T NetBond", "Megaport"} {
			var bandwidths []any
			for _, mbps := range xrBandwidthsMbps {
				bandwidths = append(bandwidths, map[string]any{"offerName": xrBandwidthName(mbps), "valueInMbps": mbps})
			}
			out = append(out, map[string]any{
				"id": scope + "/providers/" + ns + "/expressRouteServiceProviders/" + strings.ReplaceAll(provider, " ", ""), "name": provider, "type": ns + "/expressRouteServiceProviders",
				"properties": map[string]any{"peeringLocations": []any{"Silicon Valley", "Washington DC", "Amsterdam"}, "bandwidthsOffered": bandwidths, "provisioningState": "Succeeded"},
			})
		}
		return out
	})
	registerAzureARMView(ns, "bgpServiceCommunities", func(_ *Server, scope string) []map[string]any {
		var out []map[string]any
		for _, svc := range []struct{ name, community string }{{"AzureStorage", "12076:5010"}, {"AzureSQL", "12076:5030"}, {"Exchange", "12076:5110"}} {
			out = append(out, map[string]any{
				"id": scope + "/providers/" + ns + "/bgpServiceCommunities/" + svc.name, "name": svc.name, "type": ns + "/bgpServiceCommunities",
				"properties": map[string]any{"serviceName": svc.name, "bgpCommunities": []any{map[string]any{
					"serviceSupportedRegion": "Global", "communityName": svc.name, "communityValue": svc.community,
					"communityPrefixes": []any{"13.64.0.0/11"}, "isAuthorizedToUse": true, "serviceGroup": svc.name,
				}}},
			})
		}
		return out
	})
	registerAzureARMQueryView(ns, "expressRouteProviderPorts", xrProviderPorts)
}

// handleAzureExpressRoute20250501Router serves circuit and cross-connection
// statistics, ARP tables and route tables (azure_expressroute_2025_05_01_tables.go).
func (s *Server) handleAzureExpressRoute20250501Router(w http.ResponseWriter, r *http.Request) bool {
	return s.xrServeTables(w, r)
}

// xrBandwidthName names a bandwidth offer, such as 500Mbps or 10Gbps.
func xrBandwidthName(mbps float64) string {
	if mbps >= 1000 {
		return strconv.FormatFloat(mbps/1000, 'f', -1, 64) + "Gbps"
	}
	return strconv.FormatFloat(mbps, 'f', -1, 64) + "Mbps"
}

func xrValidateCircuit(payload map[string]any) string {
	sku := azureARMObject(payload["sku"])
	tier, family := azureARMString(sku["tier"]), azureARMString(sku["family"])
	switch {
	case tier != "Standard" && tier != "Premium" && tier != "Basic" && tier != "Local":
		return "sku.tier must be Standard, Premium, Basic or Local."
	case family != "MeteredData" && family != "UnlimitedData":
		return "sku.family must be MeteredData or UnlimitedData."
	case azureARMString(sku["name"]) != tier+"_"+family:
		return "sku.name must be {tier}_{family}, such as Standard_MeteredData."
	}
	props := azureARMObject(payload["properties"])
	provider := azureARMObject(props["serviceProviderProperties"])
	port := azureARMObject(props["expressRoutePort"])
	switch {
	case len(provider) != 0 && len(port) != 0:
		return "A circuit uses either serviceProviderProperties or expressRoutePort, not both."
	case len(port) != 0:
		if !azureNetworkIDOf(port, "expressrouteports") {
			return "properties.expressRoutePort.id must be an ExpressRoute port resource ID."
		}
		if gbps, ok := props["bandwidthInGbps"].(float64); !ok || gbps <= 0 || gbps > 100 {
			return "properties.bandwidthInGbps must be a number from 0 to 100."
		}
	default:
		if azureARMString(provider["serviceProviderName"]) == "" || azureARMString(provider["peeringLocation"]) == "" {
			return "properties.serviceProviderProperties needs serviceProviderName and peeringLocation."
		}
		mbps, _ := provider["bandwidthInMbps"].(float64)
		valid := false
		for _, offered := range xrBandwidthsMbps {
			valid = valid || mbps == offered
		}
		if !valid {
			return "properties.serviceProviderProperties.bandwidthInMbps must be one of 50, 100, 200, 500, 1000, 2000, 5000 or 10000."
		}
	}
	return ""
}

func xrCircuitProperties(rec *azureARMResource) map[string]any {
	state := "NotProvisioned"
	if len(azureARMObject(rec.Properties["expressRoutePort"])) != 0 {
		state = "Provisioned" // ExpressRoute Direct has no provider to wait on.
	}
	return map[string]any{
		"serviceKey":                       uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
		"circuitProvisioningState":         "Enabled",
		"serviceProviderProvisioningState": state,
		"stag":                             xrSTag(rec.ID),
	}
}

// xrSTag is a stable service tag (VLAN) in 1-4094 derived from the ID.
func xrSTag(id string) int {
	sum := 0
	for _, b := range []byte(azureARMKey(id)) {
		sum = (sum*31 + int(b)) % 4093
	}
	return sum + 1
}

func xrValidatePeering(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	switch props["peeringType"] {
	case "AzurePrivatePeering", "AzurePublicPeering", "MicrosoftPeering":
	default:
		return "properties.peeringType must be AzurePrivatePeering, AzurePublicPeering or MicrosoftPeering."
	}
	if !azureWholeNumber(props["peerASN"], 1, 4294967295) {
		return "properties.peerASN must be an integer from 1 to 4294967295."
	}
	if !azureWholeNumber(props["vlanId"], 1, 4094) {
		return "properties.vlanId must be an integer from 1 to 4094."
	}
	for _, key := range []string{"primaryPeerAddressPrefix", "secondaryPeerAddressPrefix"} {
		prefix, err := netip.ParsePrefix(azureARMString(props[key]))
		if err != nil || !prefix.Addr().Is4() || prefix.Bits() != 30 {
			return "properties." + key + " must be an IPv4 /30 prefix."
		}
	}
	if props["peeringType"] == "MicrosoftPeering" && len(azureARMSlice(azureARMObject(props["microsoftPeeringConfig"])["advertisedPublicPrefixes"])) == 0 {
		return "properties.microsoftPeeringConfig.advertisedPublicPrefixes is required for Microsoft peering."
	}
	return ""
}

// xrPeeringNameMatchesType enforces Azure's rule that a peering is named
// after its type.
func xrPeeringNameMatchesType(_ map[string]any, name string, payload map[string]any) string {
	if peeringType := azureARMString(azureARMObject(payload["properties"])["peeringType"]); !strings.EqualFold(name, peeringType) {
		return "A peering must be named after its peeringType, " + peeringType + "."
	}
	return ""
}

func xrPeeringProperties(*azureARMResource) map[string]any {
	return map[string]any{"azureASN": 12076, "primaryAzurePort": "", "secondaryAzurePort": "", "state": "Enabled", "lastModifiedBy": "Customer"}
}

func xrAuthorizationProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"authorizationKey":       uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID)+"#authorization")).String(),
		"authorizationUseStatus": "Available",
	}
}

func xrValidateCircuitConnection(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, key := range []string{"expressRouteCircuitPeering", "peerExpressRouteCircuitPeering"} {
		if !azureNetworkIDOf(props[key], "expressroutecircuits/peerings") {
			return "properties." + key + ".id must be an ExpressRoute circuit peering resource ID."
		}
	}
	prefix, err := netip.ParsePrefix(azureARMString(props["addressPrefix"]))
	if err != nil || !prefix.Addr().Is4() || prefix.Bits() != 29 {
		return "properties.addressPrefix must be an IPv4 /29 prefix."
	}
	return ""
}

// xrPeerConnections lists, under a peering, the global-reach connections
// other circuits made to it.
func xrPeerConnections(s *Server, scope string) []map[string]any {
	var out []map[string]any
	for _, conn := range s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, "Microsoft.Network/expressRouteCircuits/peerings/connections") &&
			strings.EqualFold(azureARMString(azureARMObject(rec.Properties["peerExpressRouteCircuitPeering"])["id"]), scope)
	}) {
		out = append(out, map[string]any{
			"id": scope + "/peerConnections/" + conn.Name, "name": conn.Name, "type": "Microsoft.Network/expressRouteCircuits/peerings/peerConnections",
			"properties": map[string]any{
				"expressRouteCircuitPeering":     map[string]any{"id": scope},
				"peerExpressRouteCircuitPeering": conn.Properties["expressRouteCircuitPeering"],
				"addressPrefix":                  conn.Properties["addressPrefix"],
				"circuitConnectionStatus":        "Connected",
				"connectionName":                 conn.Name,
				"provisioningState":              "Succeeded",
			},
		})
	}
	return out
}

func xrValidateCrossConnection(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if azureARMString(props["peeringLocation"]) == "" {
		return "properties.peeringLocation is required."
	}
	if mbps, present := props["bandwidthInMbps"]; present && !azureWholeNumber(mbps, 1, 100000) {
		return "properties.bandwidthInMbps must be a positive integer."
	}
	if circuit, present := props["expressRouteCircuit"]; present && !azureNetworkIDOf(circuit, "expressroutecircuits") {
		return "properties.expressRouteCircuit.id must be an ExpressRoute circuit resource ID."
	}
	switch state := props["serviceProviderProvisioningState"]; state {
	case nil, "NotProvisioned", "Provisioning", "Provisioned", "Deprovisioning":
	default:
		return "properties.serviceProviderProvisioningState must be NotProvisioned, Provisioning, Provisioned or Deprovisioning."
	}
	return ""
}

func xrCrossConnectionProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{"primaryAzurePort": "AzurePort-Primary", "secondaryAzurePort": "AzurePort-Secondary", "sTag": xrSTag(rec.ID)}
	if _, set := rec.Properties["serviceProviderProvisioningState"]; !set {
		props["serviceProviderProvisioningState"] = "NotProvisioned"
	}
	return props
}

func xrValidatePort(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if azureARMString(props["peeringLocation"]) == "" {
		return "properties.peeringLocation is required."
	}
	switch props["bandwidthInGbps"] {
	case 10.0, 100.0, 400.0:
	default:
		return "properties.bandwidthInGbps must be 10, 100 or 400."
	}
	if !egcpOneOf(props, "encapsulation", "Dot1Q", "QinQ") {
		return "properties.encapsulation must be Dot1Q or QinQ."
	}
	if !egcpOneOf(props, "billingType", "MeteredData", "UnlimitedData") {
		return "properties.billingType must be MeteredData or UnlimitedData."
	}
	for _, link := range azureARMSlice(props["links"]) {
		if !egcpOneOf(azureARMObject(azureARMObject(link)["properties"]), "adminState", "Enabled", "Disabled") {
			return "properties.links[].properties.adminState must be Enabled or Disabled."
		}
	}
	return ""
}

// xrPortProperties gives a port its two physical links, keeping the admin
// state the client set on each.
func xrPortProperties(rec *azureARMResource) map[string]any {
	sent := azureARMSlice(rec.Properties["links"])
	links := make([]any, 0, 2)
	for i, name := range []string{"link1", "link2"} {
		admin := "Disabled"
		if i < len(sent) {
			if state := azureARMString(azureARMObject(azureARMObject(sent[i])["properties"])["adminState"]); state != "" {
				admin = state
			}
		}
		links = append(links, map[string]any{
			"id": rec.ID + "/links/" + name, "name": name,
			"properties": map[string]any{
				"router": strings.ToLower(rec.Name) + "-0" + string(rune('1'+i)) + "-er", "interfaceName": "HundredGigE2/2/" + string(rune('1'+i)),
				"patchPanelId": "PP:0123:" + string(rune('1'+i)), "rackId": "RK" + string(rune('1'+i)), "connectorType": "LC",
				"adminState": admin, "provisioningState": "Succeeded",
			},
		})
	}
	return map[string]any{
		"links": links, "etherType": "0x8100", "mtu": "1500",
		"allocationDate":             rec.CreatedAt.Format(time.RFC1123),
		"provisionedBandwidthInGbps": 0, "resourceGuid": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
	}
}

func xrGenerateLOA(rec *azureARMResource, payload map[string]any) (int, any) {
	customer := strings.TrimSpace(azureARMString(payload["customerName"]))
	if customer == "" || len(customer) > 256 {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "customerName is required and must be at most 256 characters.")
	}
	letter := "Letter of Authorization\nCustomer: " + customer + "\nPort: " + rec.ID + "\nLocation: " + azureARMString(rec.Properties["peeringLocation"]) + "\n"
	return http.StatusOK, map[string]any{"encodedContent": base64.StdEncoding.EncodeToString([]byte(letter))}
}

func xrValidateRouteFilterRule(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	switch {
	case props["access"] != "Allow":
		return "properties.access must be Allow."
	case props["routeFilterRuleType"] != "Community":
		return "properties.routeFilterRuleType must be Community."
	}
	communities := azureARMSlice(props["communities"])
	if len(communities) == 0 || len(communities) > 1000 {
		return "properties.communities must hold 1 to 1000 BGP community values."
	}
	for _, community := range communities {
		value := azureARMString(community)
		asn, rest, found := strings.Cut(value, ":")
		if !found || asn == "" || rest == "" || strings.Trim(asn+rest, "0123456789") != "" {
			return "properties.communities values must look like 12076:5010."
		}
	}
	return ""
}

func xrValidatePolicyDefinition(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	service := azureARMString(props["service"])
	if service != "Microsoft.Storage" && service != "Global" {
		return "properties.service must be Microsoft.Storage or Global."
	}
	resources := azureARMSlice(props["serviceResources"])
	if len(resources) == 0 || len(resources) > 200 {
		return "properties.serviceResources must hold 1 to 200 resource scopes."
	}
	for _, scope := range resources {
		if !strings.HasPrefix(strings.ToLower(azureARMString(scope)), "/subscriptions/") {
			return "properties.serviceResources must be subscription, resource group or resource IDs."
		}
	}
	return ""
}

// xrProviderPorts is the ExpressRoute provider port catalog; $filter
// "location eq '{location}'" narrows it.
func xrProviderPorts(_ *Server, scope string, query url.Values) []map[string]any {
	location := ""
	if filter := query.Get("$filter"); filter != "" {
		_, value, _ := strings.Cut(filter, "eq ")
		location = strings.Trim(strings.TrimSpace(value), "'")
	}
	var out []map[string]any
	for _, port := range []struct{ name, location, provider string }{{"equinix-sv5-p1", "Silicon Valley", "Equinix"}, {"megaport-dc2-p1", "Washington DC", "Megaport"}} {
		if location != "" && !strings.EqualFold(location, port.location) {
			continue
		}
		out = append(out, map[string]any{
			"id": scope + "/providers/Microsoft.Network/expressRouteProviderPorts/" + port.name, "name": port.name, "type": "Microsoft.Network/expressRouteProviderPorts",
			"properties": map[string]any{"peeringLocation": port.location, "portPairDescriptor": port.provider + " port pair", "primaryAzurePort": port.name + "-primary", "secondaryAzurePort": port.name + "-secondary", "remainingBandwidthInMbps": 10000, "usedBandwidthInMbps": 0},
		})
	}
	return out
}
