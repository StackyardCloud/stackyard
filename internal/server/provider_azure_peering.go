package server

import (
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Microsoft.Peering: peer ASNs, Direct and Exchange peerings with their
// registered ASNs and prefixes, and Peering Service resources with their
// prefixes and connection monitor tests, served by the shared ARM handler.
// Microsoft approves ASNs and connections by hand; the emulator approves
// them at once, and no BGP session ever runs. Locations, facilities and
// Peering Service providers come from a small sample catalog.
func init() {
	const ns = "Microsoft.Peering"
	// PATCH takes ResourceTags: it changes tags only.
	tagsOnly := func(payload map[string]any) (map[string]any, map[string]any, string) {
		if tags, present := payload["tags"]; present {
			return map[string]any{"tags": tags}, nil, ""
		}
		return map[string]any{}, nil, ""
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "peerAsns", Subscription: true, NoPatch: true,
		Validate: azurePeerAsnValidate,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"validationState": "Approved"}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "peerings",
		Validate:           azurePeeringValidate,
		ReadOnlySKU:        azurePeeringSKU,
		ReadOnlyProperties: azurePeeringProperties,
		Linked:             azurePeeringLinked,
		PatchBody:          tagsOnly,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "peerings/registeredAsns", Proxy: true, NoPatch: true,
		Validate: func(payload map[string]any) string {
			if !azureWholeNumber(azureARMObject(payload["properties"])["asn"], 1, 4294967295) {
				return "properties.asn must be an ASN from 1 to 4294967295."
			}
			return ""
		},
		ReadOnlyProperties: azurePeeringPrefixKey,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "peerings/registeredPrefixes", Proxy: true, NoPatch: true,
		Validate: azurePeeringPrefixValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return azureARMMerge(azurePeeringPrefixKey(rec), map[string]any{"prefixValidationState": "Verified"})
		},
		Actions: map[string]azureARMAction{"validate": func(rec *azureARMResource, _ map[string]any) (int, any) {
			return http.StatusOK, rec.fixture()
		}},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "peeringServices",
		Validate:  azurePeeringServiceValidate,
		PatchBody: tagsOnly,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "peeringServices/prefixes", Proxy: true, NoPatch: true,
		Validate: azurePeeringPrefixValidate,
		Linked:   azurePeeringServicePrefixLinked,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "peeringServices/connectionMonitorTests", Proxy: true, NoPatch: true,
		Validate: azurePeeringMonitorTestValidate,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"isTestSuccessful": false, "path": []any{}}
		},
	})
	registerAzureARMView(ns, "peeringLocations", azurePeeringLocations)
	registerAzureARMView(ns, "peeringServiceLocations", azurePeeringServiceLocations)
	registerAzureARMView(ns, "peeringServiceProviders", azurePeeringServiceProviders)
	registerAzureARMView(ns, "peeringServiceCountries", func(_ *Server, scope string) []map[string]any {
		return azurePeeringCatalogItems(scope, "peeringServiceCountries", []string{"Canada", "United States"}, func(string) map[string]any { return map[string]any{} })
	})
	empty := func(*Server, string) []map[string]any { return nil }
	registerAzureARMView(ns, "legacyPeerings", empty)
	registerAzureARMView(ns, "cdnPeeringPrefixes", empty)
	registerAzureARMView(ns, "peerings/receivedRoutes", empty)
	registerAzureARMView(ns, "peerings/rpUnbilledPrefixes", empty)
	registerAzureARMProviderAction(ns, "checkServiceProviderAvailability", func(_ *Server, payload map[string]any) (int, any) {
		if azurePeeringProviderServes(azureARMString(payload["peeringServiceProvider"]), azureARMString(payload["peeringServiceLocation"])) {
			return http.StatusOK, "Available"
		}
		return http.StatusOK, "Unavailable"
	})
	registerAzureARMProviderAction(ns, "initializeConnectionMonitor", func(*Server, map[string]any) (int, any) { return http.StatusOK, nil })
}

// The sample catalog: Seattle, as the peering walkthroughs on Microsoft
// Learn describe it, and one Peering Service provider.
type azurePeeringExchangeFacility struct {
	Name, MicrosoftIPv4, MicrosoftIPv6 string
	PrefixV4, PrefixV6                 netip.Prefix
	ID                                 int
}

var (
	azurePeeringLocationName     = "Seattle"
	azurePeeringDirectFacilities = []int{71}
	azurePeeringExchanges        = []azurePeeringExchangeFacility{{
		Name: "Equinix Seattle", ID: 11, MicrosoftIPv4: "198.32.134.152", MicrosoftIPv6: "2001:504:12::15",
		PrefixV4: netip.MustParsePrefix("198.32.134.0/24"), PrefixV6: netip.MustParsePrefix("2001:504:12::/64"),
	}}
	azurePeeringServiceProvider = "Contoso Networks"
	// azurePeeringServiceAreas maps each Peering Service location to its
	// state, country and region; the provider serves only Washington.
	azurePeeringServiceAreas = map[string][3]string{
		"Washington": {"Washington", "United States", "West US 2"},
		"Ontario":    {"Ontario", "Canada", "Canada Central"},
	}
	azurePeeringSKUs = map[string][]string{
		"Direct":   {"Basic_Direct_Free", "Premium_Direct_Free", "Premium_Direct_Metered", "Premium_Direct_Unlimited"},
		"Exchange": {"Basic_Exchange_Free", "Premium_Exchange_Metered"},
	}
	azurePeeringDirectTypes = []string{"Edge", "Transit", "Cdn", "Internal", "Ix", "IxRs", "Voice", "EdgeZoneForOperators", "PeerProp"}
	azurePeeringRoles       = []string{"Noc", "Policy", "Technical", "Service", "Escalation", "Other"}
)

func azurePeeringProviderServes(provider, location string) bool {
	return provider == azurePeeringServiceProvider && location == "Washington"
}

func azurePeerAsnValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !azureWholeNumber(props["peerAsn"], 1, 4294967295) {
		return "properties.peerAsn must be an ASN from 1 to 4294967295."
	}
	if strings.TrimSpace(azureARMString(props["peerName"])) == "" {
		return "properties.peerName is required."
	}
	contacts := azureARMSlice(props["peerContactDetail"])
	if len(contacts) == 0 {
		return "properties.peerContactDetail needs at least one contact."
	}
	for _, raw := range contacts {
		contact := azureARMObject(raw)
		if !slices.Contains(azurePeeringRoles, azureARMString(contact["role"])) || !strings.Contains(azureARMString(contact["email"]), "@") {
			return "properties.peerContactDetail items need a role of " + strings.Join(azurePeeringRoles, ", ") + " and an email address."
		}
	}
	return ""
}

// azurePeeringSKU derives tier, family and size from a name such as
// Basic_Direct_Free.
func azurePeeringSKU(sku map[string]any) map[string]any {
	parts := strings.Split(azureARMString(sku["name"]), "_")
	if len(parts) != 3 {
		return nil
	}
	return map[string]any{"tier": parts[0], "family": parts[1], "size": parts[2]}
}

func azurePeeringValidate(payload map[string]any) string {
	kind := azureARMString(payload["kind"])
	skus, ok := azurePeeringSKUs[kind]
	if !ok {
		return "kind must be Direct or Exchange."
	}
	if !slices.Contains(skus, azureARMString(azureARMObject(payload["sku"])["name"])) {
		return "sku.name must be one of " + strings.Join(skus, ", ") + " for a " + kind + " peering."
	}
	props := azureARMObject(payload["properties"])
	if azureARMString(props["peeringLocation"]) != azurePeeringLocationName {
		return "properties.peeringLocation must be a peering location; the emulator offers " + azurePeeringLocationName + "."
	}
	section := azureARMObject(props[strings.ToLower(kind)])
	if !strings.Contains(strings.ToLower(azureARMString(azureARMObject(section["peerAsn"])["id"])), "/providers/microsoft.peering/peerasns/") {
		return "properties." + strings.ToLower(kind) + ".peerAsn.id must be the ID of a peer ASN."
	}
	connections := azureARMSlice(section["connections"])
	if len(connections) == 0 {
		return "properties." + strings.ToLower(kind) + ".connections needs at least one connection."
	}
	if kind == "Direct" {
		if raw, present := section["directPeeringType"]; present && !slices.Contains(azurePeeringDirectTypes, azureARMString(raw)) {
			return "properties.direct.directPeeringType must be one of " + strings.Join(azurePeeringDirectTypes, ", ") + "."
		}
	}
	for _, raw := range connections {
		check := azurePeeringExchangeConnectionValidate
		if kind == "Direct" {
			check = azurePeeringDirectConnectionValidate
		}
		if msg := check(azureARMObject(raw)); msg != "" {
			return msg
		}
	}
	return ""
}

// azurePeeringDirectConnectionValidate checks a Direct connection: a
// facility at the location, bandwidth in 10 Gbps steps, and, when the peer
// provides the session addresses, a /31 IPv4 or /127 IPv6 session prefix.
func azurePeeringDirectConnectionValidate(conn map[string]any) string {
	if !slices.ContainsFunc(azurePeeringDirectFacilities, func(id int) bool { return conn["peeringDBFacilityId"] == float64(id) }) {
		return "Direct connections need a peeringDBFacilityId at the peering location, such as 71."
	}
	bandwidth, _ := conn["bandwidthInMbps"].(float64)
	if !azureWholeNumber(conn["bandwidthInMbps"], 10000, 2147483647) || int64(bandwidth)%10000 != 0 {
		return "Direct connections need a bandwidthInMbps in steps of 10000 (10 Gbps)."
	}
	provider := azureARMString(conn["sessionAddressProvider"])
	if provider != "" && provider != "Peer" && provider != "Microsoft" {
		return "sessionAddressProvider must be Peer or Microsoft."
	}
	session := azureARMObject(conn["bgpSession"])
	v4, hasV4 := session["sessionPrefixV4"]
	v6, hasV6 := session["sessionPrefixV6"]
	if provider != "Microsoft" && !hasV4 && !hasV6 {
		return "Connections whose session addresses the peer provides need bgpSession.sessionPrefixV4 or sessionPrefixV6."
	}
	if p, err := netip.ParsePrefix(azureARMString(v4)); hasV4 && (err != nil || !p.Addr().Is4() || p.Bits() != 31) {
		return "bgpSession.sessionPrefixV4 must be an IPv4 /31 prefix."
	}
	if p, err := netip.ParsePrefix(azureARMString(v6)); hasV6 && (err != nil || !p.Addr().Is6() || p.Bits() != 127) {
		return "bgpSession.sessionPrefixV6 must be an IPv6 /127 prefix."
	}
	return azurePeeringMaxPrefixes(session)
}

// azurePeeringExchangeConnectionValidate checks an Exchange connection: an
// exchange at the location, and peer session addresses in its prefixes.
func azurePeeringExchangeConnectionValidate(conn map[string]any) string {
	facility := azurePeeringExchangeOf(conn["peeringDBFacilityId"])
	if facility == nil {
		return "Exchange connections need the peeringDBFacilityId of an exchange at the peering location, such as 11."
	}
	session := azureARMObject(conn["bgpSession"])
	v4, hasV4 := session["peerSessionIPv4Address"]
	v6, hasV6 := session["peerSessionIPv6Address"]
	if !hasV4 && !hasV6 {
		return "Exchange connections need bgpSession.peerSessionIPv4Address or peerSessionIPv6Address."
	}
	if a, err := netip.ParseAddr(azureARMString(v4)); hasV4 && (err != nil || !facility.PrefixV4.Contains(a)) {
		return "bgpSession.peerSessionIPv4Address must be in the exchange's prefix, " + facility.PrefixV4.String() + "."
	}
	if a, err := netip.ParseAddr(azureARMString(v6)); hasV6 && (err != nil || !facility.PrefixV6.Contains(a)) {
		return "bgpSession.peerSessionIPv6Address must be in the exchange's prefix, " + facility.PrefixV6.String() + "."
	}
	return azurePeeringMaxPrefixes(session)
}

func azurePeeringMaxPrefixes(session map[string]any) string {
	for _, key := range []string{"maxPrefixesAdvertisedV4", "maxPrefixesAdvertisedV6"} {
		if raw, present := session[key]; present && !azureWholeNumber(raw, 0, 2147483647) {
			return "bgpSession." + key + " must be a whole number of at least 0."
		}
	}
	return ""
}

func azurePeeringExchangeOf(raw any) *azurePeeringExchangeFacility {
	for i := range azurePeeringExchanges {
		if raw == float64(azurePeeringExchanges[i].ID) {
			return &azurePeeringExchanges[i]
		}
	}
	return nil
}

// azurePeeringProperties reports each connection approved and its BGP
// sessions established, since the emulator approves them at once.
func azurePeeringProperties(rec *azureARMResource) map[string]any {
	key := strings.ToLower(rec.Kind)
	section := azureARMObject(rec.Properties[key])
	connections := make([]any, 0)
	forService := false
	for i, raw := range azureARMSlice(section["connections"]) {
		conn := azureARMMerge(azureARMObject(raw), map[string]any{
			"connectionState": "Active", "microsoftTrackingId": "stackyard-" + rec.Name + "-" + strconv.Itoa(i),
		})
		if azureARMString(conn["connectionIdentifier"]) == "" {
			conn["connectionIdentifier"] = uuid.NewSHA1(uuid.NameSpaceURL, []byte(rec.ID+"#"+strconv.Itoa(i))).String()
		}
		if rec.Kind == "Direct" {
			conn["provisionedBandwidthInMbps"] = conn["bandwidthInMbps"]
		}
		forService = forService || conn["useForPeeringService"] == true
		if session := azurePeeringSession(rec.Kind, conn); session != nil {
			conn["bgpSession"] = session
		}
		connections = append(connections, conn)
	}
	out := map[string]any{"connections": connections}
	if rec.Kind == "Direct" {
		out["useForPeeringService"] = forService
	}
	return map[string]any{key: out}
}

// azurePeeringSession fills in Microsoft's session addresses and the
// session states: Direct sessions take the first address of each peer
// prefix, and Exchange sessions use the exchange's Microsoft addresses.
func azurePeeringSession(kind string, conn map[string]any) map[string]any {
	raw, ok := conn["bgpSession"].(map[string]any)
	if !ok {
		return nil
	}
	session := azureARMMerge(raw, map[string]any{"sessionStateV4": "None", "sessionStateV6": "None"})
	if kind == "Exchange" {
		facility := azurePeeringExchangeOf(conn["peeringDBFacilityId"])
		if _, ok := raw["peerSessionIPv4Address"]; ok && facility != nil {
			session["microsoftSessionIPv4Address"], session["sessionStateV4"] = facility.MicrosoftIPv4, "Established"
		}
		if _, ok := raw["peerSessionIPv6Address"]; ok && facility != nil {
			session["microsoftSessionIPv6Address"], session["sessionStateV6"] = facility.MicrosoftIPv6, "Established"
		}
		return session
	}
	for _, v := range []struct{ prefix, microsoft, peer, state string }{
		{"sessionPrefixV4", "microsoftSessionIPv4Address", "peerSessionIPv4Address", "sessionStateV4"},
		{"sessionPrefixV6", "microsoftSessionIPv6Address", "peerSessionIPv6Address", "sessionStateV6"},
	} {
		if p, err := netip.ParsePrefix(azureARMString(raw[v.prefix])); err == nil {
			first := p.Masked().Addr()
			session[v.microsoft], session[v.peer], session[v.state] = first.String(), first.Next().String(), "Established"
		}
	}
	return session
}

// azurePeeringLinked requires the peering's peer ASN to exist in its
// subscription. Caller holds providerStorageMu.
func azurePeeringLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	asn := azureARMString(azureARMObject(azureARMObject(rec.Properties[strings.ToLower(rec.Kind)])["peerAsn"])["id"])
	if !strings.HasPrefix(strings.ToLower(asn), strings.ToLower("/subscriptions/"+rec.SubscriptionID+"/")) || s.azureARMResources[azureARMKey(asn)] == nil {
		return nil, nil, "The peer ASN " + asn + " doesn't exist in this subscription. Associate the ASN first."
	}
	return nil, nil, ""
}

func azurePeeringPrefixKey(rec *azureARMResource) map[string]any {
	return map[string]any{"peeringServicePrefixKey": uuid.NewSHA1(uuid.NameSpaceURL, []byte(strings.ToLower(rec.ID))).String()}
}

func azurePeeringPrefixValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if p, err := netip.ParsePrefix(azureARMString(props["prefix"])); err != nil || !p.Addr().Is4() || p.Masked() != p {
		return "properties.prefix must be an IPv4 prefix such as 192.168.1.0/24."
	}
	if raw, present := props["peeringServicePrefixKey"]; present {
		if _, err := uuid.Parse(azureARMString(raw)); err != nil {
			return "properties.peeringServicePrefixKey must be the GUID the provider's registered prefix gives."
		}
	}
	return ""
}

// azurePeeringServicePrefixLinked verifies a Peering Service prefix whose
// key and prefix match a prefix a provider registered on a peering. Caller
// holds providerStorageMu.
// ponytail: checked only when the prefix is written; Azure re-validates in
// the background, so a prefix written before the provider registers it
// stays Failed until it is written again.
func azurePeeringServicePrefixLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	key, prefix := azureARMString(rec.Properties["peeringServicePrefixKey"]), azureARMString(rec.Properties["prefix"])
	for _, other := range s.azureARMResources {
		if strings.EqualFold(other.Type, "Microsoft.Peering/peerings/registeredPrefixes") && key != "" &&
			strings.EqualFold(azureARMString(other.Properties["peeringServicePrefixKey"]), key) && other.Properties["prefix"] == prefix {
			return map[string]any{"prefixValidationState": "Verified", "learnedType": "ViaServiceProvider", "events": []any{}}, nil, ""
		}
	}
	return map[string]any{
		"prefixValidationState": "Failed", "learnedType": "None", "events": []any{},
		"errorMessage": "Prefix is not announced by the service provider to Microsoft.",
	}, nil, ""
}

func azurePeeringServiceValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	location, provider := azureARMString(props["peeringServiceLocation"]), azureARMString(props["peeringServiceProvider"])
	if _, ok := azurePeeringServiceAreas[location]; !ok {
		return "properties.peeringServiceLocation must be a Peering Service location, such as Washington."
	}
	if provider != azurePeeringServiceProvider {
		return "properties.peeringServiceProvider must be a Peering Service provider, such as " + azurePeeringServiceProvider + "."
	}
	if !azurePeeringProviderServes(provider, location) {
		return provider + " isn't available in " + location + "."
	}
	primary, backup := azureARMString(props["providerPrimaryPeeringLocation"]), azureARMString(props["providerBackupPeeringLocation"])
	for _, peering := range []string{primary, backup} {
		if peering != "" && peering != azurePeeringLocationName {
			return "The provider's peering locations must be ones it peers at, such as " + azurePeeringLocationName + "."
		}
	}
	if primary != "" && primary == backup {
		return "properties.providerBackupPeeringLocation must differ from providerPrimaryPeeringLocation."
	}
	return ""
}

func azurePeeringMonitorTestValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(props["sourceAgent"])) == "" || strings.TrimSpace(azureARMString(props["destination"])) == "" {
		return "properties.sourceAgent and destination are required."
	}
	if !azureWholeNumber(props["destinationPort"], 1, 65535) {
		return "properties.destinationPort must be a port from 1 to 65535."
	}
	if raw, present := props["testFrequencyInSec"]; present && !azureWholeNumber(raw, 1, 2147483647) {
		return "properties.testFrequencyInSec must be a whole number of seconds."
	}
	return ""
}

// azurePeeringCatalogItems lists catalog entries under a subscription.
func azurePeeringCatalogItems(scope, collection string, names []string, props func(string) map[string]any) []map[string]any {
	items := make([]map[string]any, 0, len(names))
	for _, name := range names {
		items = append(items, map[string]any{
			"id": scope + "/providers/Microsoft.Peering/" + collection + "/" + name, "name": name,
			"type": "Microsoft.Peering/" + collection, "properties": props(name),
		})
	}
	return items
}

// azurePeeringLocations lists Seattle once for each kind. The view can't
// read the kind query parameter, so it lists both.
func azurePeeringLocations(_ *Server, scope string) []map[string]any {
	facilities := make([]any, 0, len(azurePeeringExchanges))
	for _, f := range azurePeeringExchanges {
		facilities = append(facilities, map[string]any{
			"exchangeName": f.Name, "bandwidthInMbps": 10000, "peeringDBFacilityId": f.ID,
			"peeringDBFacilityLink": "https://www.peeringdb.com/ix/" + strconv.Itoa(f.ID),
			"microsoftIPv4Address":  f.MicrosoftIPv4, "microsoftIPv6Address": f.MicrosoftIPv6,
			"facilityIPv4Prefix": f.PrefixV4.String(), "facilityIPv6Prefix": f.PrefixV6.String(),
		})
	}
	common := map[string]any{"peeringLocation": azurePeeringLocationName, "country": "US", "azureRegion": "West US 2"}
	direct := azurePeeringCatalogItems(scope, "peeringLocations", []string{azurePeeringLocationName}, func(string) map[string]any {
		return azureARMMerge(common, map[string]any{"direct": map[string]any{
			"peeringFacilities": []any{map[string]any{"address": "2001 Sixth Avenue", "directPeeringType": "Edge", "peeringDBFacilityId": 71, "peeringDBFacilityLink": "https://www.peeringdb.com/fac/71"}},
			"bandwidthOffers":   []any{map[string]any{"offerName": "10Gbps", "valueInMbps": 10000}, map[string]any{"offerName": "100Gbps", "valueInMbps": 100000}},
		}})
	})
	exchange := azurePeeringCatalogItems(scope, "peeringLocations", []string{azurePeeringLocationName}, func(string) map[string]any {
		return azureARMMerge(common, map[string]any{"exchange": map[string]any{"peeringFacilities": facilities}})
	})
	direct[0]["kind"], exchange[0]["kind"] = "Direct", "Exchange"
	return append(direct, exchange...)
}

func azurePeeringServiceLocations(_ *Server, scope string) []map[string]any {
	names := make([]string, 0, len(azurePeeringServiceAreas))
	for name := range azurePeeringServiceAreas {
		names = append(names, name)
	}
	slices.Sort(names)
	return azurePeeringCatalogItems(scope, "peeringServiceLocations", names, func(name string) map[string]any {
		area := azurePeeringServiceAreas[name]
		return map[string]any{"state": area[0], "country": area[1], "azureRegion": area[2]}
	})
}

func azurePeeringServiceProviders(_ *Server, scope string) []map[string]any {
	return azurePeeringCatalogItems(scope, "peeringServiceProviders", []string{azurePeeringServiceProvider}, func(name string) map[string]any {
		return map[string]any{"serviceProviderName": name, "peeringLocations": []any{azurePeeringLocationName}}
	})
}
