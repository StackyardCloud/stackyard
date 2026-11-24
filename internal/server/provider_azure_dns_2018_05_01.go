package server

import (
	"net/http"
	"strings"
)

// Azure DNS (Microsoft.Network/dnsZones, api-version 2018-05-01) on the shared
// ARM handler: public zones with their apex SOA and NS record sets, typed
// record sets per zone, zone-wide record set lists and the DNS resource
// reference lookup for alias record sets. No queries are answered.
var adnsPublic = adnsZoneFlavor{
	namespace: "Microsoft.Network", zoneType: "dnsZones", ttlMember: "TTL",
	maxSets: 10000, kinds: adnsKinds(false), allowTarget: true,
}

func init() {
	f := adnsPublic
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: f.namespace, Type: f.zoneType, ETag: true,
		ValidateName: adnsValidateZoneName,
		Validate: func(payload map[string]any) string {
			if !strings.EqualFold(azureARMString(payload["location"]), "global") {
				return "The location of a DNS zone must be 'global'."
			}
			switch azureARMString(azureARMObject(payload["properties"])["zoneType"]) {
			case "", "Public":
				return ""
			case "Private":
				return "Private zones are served by Microsoft.Network/privateDnsZones."
			}
			return "properties.zoneType must be Public."
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"nameServers": adnsNameServers(rec.ID), "maxNumberOfRecordSets": f.maxSets, "zoneType": "Public"}
		},
		ChildProperties: func(s *Server, rec *azureARMResource) map[string]any {
			return map[string]any{"numberOfRecordSets": len(adnsRecordSetsLocked(s, f, rec.ID))}
		},
		DefaultChildren: func(zone *azureARMResource) []*azureARMResource {
			servers := adnsNameServers(zone.ID)
			ns := []any{}
			for _, server := range servers {
				ns = append(ns, map[string]any{"nsdname": server})
			}
			return []*azureARMResource{
				azureARMChild(zone, adnsRecordType(f, "SOA"), "@", adnsDefaultSOA(f, strings.TrimSuffix(azureARMString(servers[0]), "."), "azuredns-hostmaster.microsoft.com", 300)),
				azureARMChild(zone, adnsRecordType(f, "NS"), "@", map[string]any{"TTL": 172800, "NSRecords": ns}),
			}
		},
	})
	registerAzureDNSRecordTypes(f, "recordsets", "all")
	registerAzureARMProviderAction(f.namespace, "getDnsResourceReference", adnsGetResourceReference)
}

// adnsValidateZoneName accepts a fully qualified domain name of at least
// two labels, e.g. contoso.com.
func adnsValidateZoneName(name string) string {
	labels := strings.Split(strings.TrimSuffix(name, "."), ".")
	if len(name) > 253 || len(labels) < 2 {
		return "The zone name '" + name + "' is invalid: it must have at least two labels."
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "The zone name '" + name + "' is invalid."
		}
		for _, c := range label {
			if !(c == '-' || c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
				return "The zone name '" + name + "' is invalid."
			}
		}
	}
	return ""
}

// adnsGetResourceReference lists the alias record sets pointing at each
// requested target resource.
func adnsGetResourceReference(s *Server, payload map[string]any) (int, any) {
	targets, ok := azureARMObject(payload["properties"])["targetResources"].([]any)
	if !ok || len(targets) == 0 {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.targetResources is missing or empty.")
	}
	if len(targets) > adnsMaxListTargets {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.targetResources holds too many items.")
	}
	refs := make([]any, 0, len(targets))
	for _, item := range targets {
		target := azureARMString(azureARMObject(item)["id"])
		if target == "" {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "Each target resource needs an id.")
		}
		// ponytail: provider actions get no subscription, so this matches by target ID alone.
		sets := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
			return strings.HasPrefix(strings.ToLower(rec.Type), "microsoft.network/dnszones/") &&
				strings.EqualFold(azureARMString(azureARMObject(rec.Properties["targetResource"])["id"]), target)
		})
		resources := make([]any, 0, len(sets))
		for _, rec := range sets {
			resources = append(resources, map[string]any{"id": rec.ID})
		}
		refs = append(refs, map[string]any{"dnsResources": resources, "targetResource": map[string]any{"id": target}})
	}
	return http.StatusOK, map[string]any{"properties": map[string]any{"dnsResourceReferences": refs}}
}
