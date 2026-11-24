package server

import (
	"fmt"
	"strings"
)

// Azure Private DNS (Microsoft.Network/privateDnsZones, api-version
// 2018-09-01) on the shared ARM handler: private zones with their apex SOA
// record set, typed record sets, the zone-wide record set list and virtual
// network links to existing virtual networks.
var pdnsFlavor = adnsZoneFlavor{
	namespace: "Microsoft.Network", zoneType: "privateDnsZones", ttlMember: "ttl",
	maxSets: 25000, kinds: adnsKinds(true),
}

const (
	pdnsMaxLinks             = 1000
	pdnsMaxRegistrationLinks = 100
)

func init() {
	f := pdnsFlavor
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: f.namespace, Type: f.zoneType, ETag: true,
		ValidateName: adnsValidateZoneName,
		Validate:     pdnsValidateGlobal,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{
				"maxNumberOfRecordSets": f.maxSets, "maxNumberOfVirtualNetworkLinks": pdnsMaxLinks,
				"maxNumberOfVirtualNetworkLinksWithRegistration": pdnsMaxRegistrationLinks,
			}
		},
		ChildProperties: func(s *Server, rec *azureARMResource) map[string]any {
			links, registered := pdnsLinksLocked(s, rec.ID)
			return map[string]any{
				"numberOfRecordSets": len(adnsRecordSetsLocked(s, f, rec.ID)), "numberOfVirtualNetworkLinks": links,
				"numberOfVirtualNetworkLinksWithRegistration": registered,
			}
		},
		DefaultChildren: func(zone *azureARMResource) []*azureARMResource {
			return []*azureARMResource{azureARMChild(zone, adnsRecordType(f, "SOA"), "@",
				adnsDefaultSOA(f, "azureprivatedns.net", "azureprivatedns-host.microsoft.com", 10))}
		},
	})
	registerAzureDNSRecordTypes(f, "all")
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: f.namespace, Type: f.zoneType + "/virtualNetworkLinks", ETag: true,
		Validate: func(payload map[string]any) string {
			if msg := pdnsValidateGlobal(payload); msg != "" {
				return msg
			}
			props := azureARMObject(payload["properties"])
			if !azureNetworkIDOf(props["virtualNetwork"], "virtualnetworks") {
				return "properties.virtualNetwork.id is missing or is not a virtual network ID."
			}
			if raw, ok := props["registrationEnabled"]; ok && raw != nil {
				if _, isBool := raw.(bool); !isBool {
					return "properties.registrationEnabled must be a boolean."
				}
			}
			return ""
		},
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"virtualNetworkLinkState": "Completed"}
		},
		Reconcile: func(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
			vnet := azureARMString(azureARMObject(rec.Properties["virtualNetwork"])["id"])
			if s.azureARMResources[azureARMKey(vnet)] == nil {
				return nil, "The virtual network " + vnet + " was not found."
			}
			zone := rec.ID[:strings.LastIndex(rec.ID[:strings.LastIndex(rec.ID, "/")], "/")]
			links, registered := pdnsLinksLocked(s, zone)
			if existing == nil && links >= pdnsMaxLinks {
				return nil, fmt.Sprintf("The zone already has the maximum of %d virtual network links.", pdnsMaxLinks)
			}
			wasRegistered := existing != nil && existing.Properties["registrationEnabled"] == true
			if rec.Properties["registrationEnabled"] == true && !wasRegistered && registered >= pdnsMaxRegistrationLinks {
				return nil, fmt.Sprintf("The zone already has the maximum of %d registration-enabled links.", pdnsMaxRegistrationLinks)
			}
			return nil, ""
		},
	})
}

func pdnsValidateGlobal(payload map[string]any) string {
	if !strings.EqualFold(azureARMString(payload["location"]), "global") {
		return "The location of a private DNS resource must be 'global'."
	}
	return ""
}

// pdnsLinksLocked counts zone's virtual network links and those with
// auto-registration. Caller holds providerStorageMu.
func pdnsLinksLocked(s *Server, zone string) (links, registered int) {
	parent := azureARMKey(zone) + "/virtualnetworklinks/"
	for _, rec := range s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.HasPrefix(azureARMKey(rec.ID), parent)
	}) {
		links++
		if rec.Properties["registrationEnabled"] == true {
			registered++
		}
	}
	return links, registered
}
