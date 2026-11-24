package server

import (
	"net/http"
	"net/netip"
	"net/url"
	"strings"
)

// Azure DNS Private Resolver (Microsoft.Network dnsResolvers,
// dnsForwardingRulesets, dnsResolverPolicies and dnsResolverDomainLists,
// api-version 2025-05-01) on the shared ARM handler. Resolvers, links and
// endpoints must reference existing virtual networks and subnets. No queries
// are resolved or forwarded.
const (
	dnsrMaxTargetServers = 6
	dnsrMaxIPConfigs     = 5
	dnsrMaxDomains       = 10000
	dnsrMaxRefs          = 50
)

func init() {
	const ns = "Microsoft.Network"
	vnetLink := func(typ string, proxy bool) azureARMResourceType {
		return azureARMResourceType{
			Namespace: ns, Type: typ, Proxy: proxy, ETag: true,
			Validate:  dnsrRequireRef("virtualNetwork", "virtualnetworks"),
			Reconcile: dnsrRequireStored("virtualNetwork"),
		}
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "dnsResolvers", ETag: true,
		Validate:           dnsrRequireRef("virtualNetwork", "virtualnetworks"),
		Reconcile:          dnsrRequireStored("virtualNetwork"),
		ReadOnlyProperties: dnsrState("dnsResolverState", "Connected"),
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "dnsResolvers/inboundEndpoints", ETag: true,
		Validate:           dnsrValidateInbound,
		Reconcile:          dnsrInboundSubnetsStored,
		ReadOnlyProperties: azureNetworkResourceGUID,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "dnsResolvers/outboundEndpoints", ETag: true,
		Validate:           dnsrRequireRef("subnet", "virtualnetworks/subnets"),
		Reconcile:          dnsrRequireStored("subnet"),
		ReadOnlyProperties: azureNetworkResourceGUID,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "dnsForwardingRulesets", ETag: true,
		Validate:           dnsrValidateRuleset,
		Reconcile:          dnsrRulesetEndpointsStored,
		ReadOnlyProperties: azureNetworkResourceGUID,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "dnsForwardingRulesets/forwardingRules", Proxy: true, ETag: true,
		Validate: dnsrValidateForwardingRule,
	})
	registerAzureARMResourceType(vnetLink("dnsForwardingRulesets/virtualNetworkLinks", true))
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "dnsResolverPolicies", ETag: true,
		ReadOnlyProperties: azureNetworkResourceGUID,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "dnsResolverPolicies/dnsSecurityRules", ETag: true,
		Validate: dnsrValidateSecurityRule,
	})
	registerAzureARMResourceType(vnetLink("dnsResolverPolicies/virtualNetworkLinks", false))
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "dnsResolverDomainLists", ETag: true,
		Validate:           dnsrValidateDomainList,
		ReadOnlyProperties: azureNetworkResourceGUID,
		Actions:            map[string]azureARMAction{"bulk": dnsrDomainListBulk},
	})
}

// dnsrVNetActions are the list operations Azure serves on a virtual network;
// provider_azure_network.go attaches them to virtualNetworks.
var dnsrVNetActions = map[string]func(s *Server, rec *azureARMResource, payload map[string]any) (int, any){
	"listdnsresolvers": func(s *Server, vnet *azureARMResource, _ map[string]any) (int, any) {
		return dnsrSubResources(s, "microsoft.network/dnsresolvers", vnet.ID, false)
	},
	"listdnsforwardingrulesets": func(s *Server, vnet *azureARMResource, _ map[string]any) (int, any) {
		return dnsrSubResources(s, "microsoft.network/dnsforwardingrulesets/virtualnetworklinks", vnet.ID, true)
	},
	"listdnsresolverpolicies": func(s *Server, vnet *azureARMResource, _ map[string]any) (int, any) {
		return dnsrSubResources(s, "microsoft.network/dnsresolverpolicies/virtualnetworklinks", vnet.ID, true)
	},
}

// dnsrSubResources lists the resources of type whose virtualNetwork is vnet.
// For links it returns the link's parent, as Azure does.
func dnsrSubResources(s *Server, typ, vnet string, viaLink bool) (int, any) {
	recs := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, typ) && strings.EqualFold(azureARMString(azureARMObject(rec.Properties["virtualNetwork"])["id"]), vnet)
	})
	out := make([]any, 0, len(recs))
	for _, rec := range recs {
		if !viaLink {
			out = append(out, map[string]any{"id": rec.ID})
			continue
		}
		parent := rec.ID[:strings.LastIndex(rec.ID[:strings.LastIndex(rec.ID, "/")], "/")]
		out = append(out, map[string]any{"id": parent, "properties": map[string]any{"virtualNetworkLink": map[string]any{"id": rec.ID}}})
	}
	return http.StatusOK, map[string]any{"value": out}
}

func dnsrState(key, value string) func(*azureARMResource) map[string]any {
	return func(rec *azureARMResource) map[string]any {
		props := azureNetworkResourceGUID(rec)
		props[key] = value
		return props
	}
}

func dnsrRequireRef(member, typePath string) func(map[string]any) string {
	return func(payload map[string]any) string {
		if !azureNetworkIDOf(azureARMObject(payload["properties"])[member], typePath) {
			return "properties." + member + ".id is missing or invalid."
		}
		return ""
	}
}

// dnsrRequireStored rejects a write whose properties.{member}.id names a
// resource that does not exist. Runs under the store lock.
func dnsrRequireStored(member string) func(*Server, *azureARMResource, *azureARMResource) (map[string]any, string) {
	return func(s *Server, _, rec *azureARMResource) (map[string]any, string) {
		return nil, dnsrStoredLocked(s, azureARMString(azureARMObject(rec.Properties[member])["id"]))
	}
}

func dnsrStoredLocked(s *Server, id string) string {
	if s.azureARMResources[azureARMKey(id)] == nil {
		return "The referenced resource " + id + " was not found."
	}
	return ""
}

func dnsrValidateInbound(payload map[string]any) string {
	configs, ok := azureARMObject(payload["properties"])["ipConfigurations"].([]any)
	if !ok || len(configs) == 0 || len(configs) > dnsrMaxIPConfigs {
		return "properties.ipConfigurations must hold 1 to 5 IP configurations."
	}
	for _, item := range configs {
		cfg := azureARMObject(item)
		if !azureNetworkIDOf(cfg["subnet"], "virtualnetworks/subnets") {
			return "ipConfigurations[].subnet.id is missing or invalid."
		}
		switch azureARMString(cfg["privateIpAllocationMethod"]) {
		case "", "Dynamic":
		case "Static":
			if addr, err := netip.ParseAddr(azureARMString(cfg["privateIpAddress"])); err != nil || !addr.Is4() {
				return "A Static ipConfiguration needs a valid IPv4 privateIpAddress."
			}
		default:
			return "privateIpAllocationMethod must be Dynamic or Static."
		}
	}
	return ""
}

func dnsrInboundSubnetsStored(s *Server, _, rec *azureARMResource) (map[string]any, string) {
	for _, item := range azureARMSlice(rec.Properties["ipConfigurations"]) {
		if msg := dnsrStoredLocked(s, azureARMString(azureARMObject(azureARMObject(item)["subnet"])["id"])); msg != "" {
			return nil, msg
		}
	}
	return nil, ""
}

func dnsrValidateRuleset(payload map[string]any) string {
	endpoints, ok := azureARMObject(payload["properties"])["dnsResolverOutboundEndpoints"].([]any)
	if !ok || len(endpoints) == 0 || len(endpoints) > dnsrMaxRefs {
		return "properties.dnsResolverOutboundEndpoints is missing or empty."
	}
	for _, ep := range endpoints {
		if !azureNetworkIDOf(ep, "dnsresolvers/outboundendpoints") {
			return "dnsResolverOutboundEndpoints[].id must be an outbound endpoint ID."
		}
	}
	return ""
}

func dnsrRulesetEndpointsStored(s *Server, _, rec *azureARMResource) (map[string]any, string) {
	for _, ep := range azureARMSlice(rec.Properties["dnsResolverOutboundEndpoints"]) {
		if msg := dnsrStoredLocked(s, azureARMString(azureARMObject(ep)["id"])); msg != "" {
			return nil, msg
		}
	}
	return nil, ""
}

func dnsrValidateForwardingRule(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	domain := azureARMString(props["domainName"])
	if domain == "" || !strings.HasSuffix(domain, ".") || len(domain) > 254 {
		return "properties.domainName must be a fully qualified domain name ending in '.'."
	}
	servers, ok := props["targetDnsServers"].([]any)
	if !ok || len(servers) == 0 || len(servers) > dnsrMaxTargetServers {
		return "properties.targetDnsServers must hold 1 to 6 servers."
	}
	for _, item := range servers {
		server := azureARMObject(item)
		if _, err := netip.ParseAddr(azureARMString(server["ipAddress"])); err != nil {
			return "targetDnsServers[].ipAddress is missing or invalid."
		}
		if port, ok := server["port"]; ok && port != nil {
			if p, isNum := port.(float64); !isNum || p < 1 || p > 65535 || p != float64(int64(p)) {
				return "targetDnsServers[].port must be between 1 and 65535."
			}
		}
	}
	return dnsrEnabledState(props["forwardingRuleState"], "forwardingRuleState")
}

func dnsrEnabledState(raw any, name string) string {
	switch azureARMString(raw) {
	case "", "Enabled", "Disabled":
		return ""
	}
	return name + " must be Enabled or Disabled."
}

func dnsrValidateSecurityRule(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	priority, ok := props["priority"].(float64)
	if !ok || priority < 100 || priority > 65000 || priority != float64(int64(priority)) {
		return "properties.priority must be an integer between 100 and 65000."
	}
	switch azureARMString(azureARMObject(props["action"])["actionType"]) {
	case "Allow", "Alert", "Block":
	default:
		return "properties.action.actionType must be Allow, Alert or Block."
	}
	lists, ok := props["dnsResolverDomainLists"].([]any)
	if !ok || len(lists) == 0 || len(lists) > dnsrMaxRefs {
		return "properties.dnsResolverDomainLists is missing or empty."
	}
	for _, list := range lists {
		if !azureNetworkIDOf(list, "dnsresolverdomainlists") {
			return "dnsResolverDomainLists[].id must be a DNS resolver domain list ID."
		}
	}
	return dnsrEnabledState(props["dnsSecurityRuleState"], "dnsSecurityRuleState")
}

func dnsrValidateDomainList(payload map[string]any) string {
	raw, ok := azureARMObject(payload["properties"])["domains"]
	if !ok || raw == nil {
		return ""
	}
	domains, ok := raw.([]any)
	if !ok || len(domains) > dnsrMaxDomains {
		return "properties.domains must be an array of at most 10000 domains."
	}
	for _, d := range domains {
		domain, ok := d.(string)
		if !ok || domain == "" || len(domain) > 254 || !strings.HasSuffix(domain, ".") {
			return "Each domain must be a fully qualified name ending in '.'."
		}
	}
	return ""
}

// dnsrDomainListBulk validates a bulk upload or download request and returns
// the domain list.
// ponytail: no storage account is read or written; the list keeps its domains.
func dnsrDomainListBulk(rec *azureARMResource, payload map[string]any) (int, any) {
	props := azureARMObject(payload["properties"])
	switch azureARMString(props["action"]) {
	case "Upload", "Download":
	default:
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.action must be Upload or Download.")
	}
	if u, err := url.Parse(azureARMString(props["storageUrl"])); err != nil || u.Scheme != "https" || u.Host == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.storageUrl must be an https URL.")
	}
	return http.StatusOK, rec.fixture()
}
