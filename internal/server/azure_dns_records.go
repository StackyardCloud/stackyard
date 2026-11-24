package server

import (
	"crypto/sha256"
	"fmt"
	"net/netip"
	"strings"
)

// Record sets shared by public DNS zones (Microsoft.Network/dnsZones) and
// private DNS zones (Microsoft.Network/privateDnsZones). Each record type is
// its own child type, e.g. dnsZones/A, so a record set's ID is
// {zoneId}/{recordType}/{relativeName} as in Azure. Public zones name their
// record arrays ARecords, MXRecords...; private zones aRecords, mxRecords....

const (
	adnsMaxRecordsPerSet = 20   // Azure DNS: records per record set
	adnsMaxTXTValueLen   = 1024 // characters per TXT record string
	adnsMaxRelativeName  = 253  // characters in a relative record set name
	adnsMaxListTargets   = 100  // target resources per getDnsResourceReference call
	adnsMaxTTL           = 1<<31 - 1
)

// adnsRecordKind is one record type's payload member: an array of records,
// or (for CNAME and SOA) a single object.
type adnsRecordKind struct {
	recordType string // "A", "AAAA", ...
	member     string // properties member holding the records
	single     bool
	check      func(rec map[string]any) string
}

// adnsZoneFlavor holds what differs between public and private zones.
type adnsZoneFlavor struct {
	namespace   string
	zoneType    string // "dnsZones" or "privateDnsZones"
	ttlMember   string // "TTL" or "ttl"
	maxSets     int
	kinds       []adnsRecordKind
	allowTarget bool // public zones support alias record sets (targetResource)
}

func adnsKinds(private bool) []adnsRecordKind {
	name := func(public, priv string) string {
		if private {
			return priv
		}
		return public
	}
	kinds := []adnsRecordKind{
		{recordType: "A", member: name("ARecords", "aRecords"), check: adnsCheckIP("ipv4Address", true)},
		{recordType: "AAAA", member: name("AAAARecords", "aaaaRecords"), check: adnsCheckIP("ipv6Address", false)},
		{recordType: "CNAME", member: name("CNAMERecord", "cnameRecord"), single: true, check: adnsCheckHost("cname")},
		{recordType: "MX", member: name("MXRecords", "mxRecords"), check: adnsCheckAll(adnsCheckUint("preference", 65535), adnsCheckHost("exchange"))},
		{recordType: "PTR", member: name("PTRRecords", "ptrRecords"), check: adnsCheckHost("ptrdname")},
		{recordType: "SOA", member: name("SOARecord", "soaRecord"), single: true, check: adnsCheckSOA},
		{recordType: "SRV", member: name("SRVRecords", "srvRecords"), check: adnsCheckAll(
			adnsCheckUint("priority", 65535), adnsCheckUint("weight", 65535), adnsCheckUint("port", 65535), adnsCheckHost("target"))},
		{recordType: "TXT", member: name("TXTRecords", "txtRecords"), check: adnsCheckTXT},
	}
	if !private {
		kinds = append(kinds,
			adnsRecordKind{recordType: "NS", member: "NSRecords", check: adnsCheckHost("nsdname")},
			adnsRecordKind{recordType: "CAA", member: "caaRecords", check: adnsCheckCAA},
		)
	}
	return kinds
}

// registerAzureDNSRecordTypes registers one child type per record type under
// the zone type, plus the zone-wide record set list views.
func registerAzureDNSRecordTypes(f adnsZoneFlavor, views ...string) {
	for _, kind := range f.kinds {
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: f.namespace, Type: f.zoneType + "/" + kind.recordType, Proxy: true, ETag: true,
			ValidateName:       adnsValidateRelativeName(kind),
			Validate:           func(payload map[string]any) string { return adnsValidateRecordSet(f, kind, payload) },
			ReadOnlyProperties: adnsRecordSetProperties,
			ValidatePut: func(s *Server, existing, rec *azureARMResource) (string, string) {
				if existing != nil {
					return "", ""
				}
				if kind.recordType == "SOA" {
					return "BadRequest", "The SOA record set is created with the zone and can only be updated."
				}
				zone := rec.ID[:strings.LastIndex(rec.ID[:strings.LastIndex(rec.ID, "/")], "/")]
				if n := len(adnsRecordSetsLocked(s, f, zone)); n >= f.maxSets {
					return "QuotaExceeded", fmt.Sprintf("The zone already holds the maximum of %d record sets.", f.maxSets)
				}
				return "", ""
			},
			ValidateDelete: func(_ *Server, existing *azureARMResource) string {
				if existing.Name == "@" && (kind.recordType == "SOA" || kind.recordType == "NS") {
					return "The " + kind.recordType + " record set at the zone apex cannot be deleted."
				}
				return ""
			},
		})
	}
	for _, view := range views {
		registerAzureARMView(f.namespace, f.zoneType+"/"+view, func(s *Server, zone string) []map[string]any {
			s.providerStorageMu.Lock()
			defer s.providerStorageMu.Unlock()
			sets := adnsRecordSetsLocked(s, f, zone)
			out := make([]map[string]any, 0, len(sets))
			for _, rec := range sets {
				out = append(out, rec.fixture())
			}
			return out
		})
	}
}

// adnsRecordSetsLocked returns the record sets stored under zone. Caller
// holds providerStorageMu.
// ponytail: scans the whole ARM store; index by zone if zones grow large.
func adnsRecordSetsLocked(s *Server, f adnsZoneFlavor, zone string) []*azureARMResource {
	prefix := strings.ToLower(f.namespace + "/" + f.zoneType + "/")
	parent := azureARMKey(zone) + "/"
	return s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		t := strings.ToLower(rec.Type)
		return strings.HasPrefix(t, prefix) && strings.Count(t, "/") == 2 && !strings.HasSuffix(t, "/virtualnetworklinks") &&
			strings.HasPrefix(azureARMKey(rec.ID), parent)
	})
}

func adnsValidateRelativeName(kind adnsRecordKind) func(string) string {
	return func(name string) string {
		if len(name) > adnsMaxRelativeName {
			return "The record set name is too long."
		}
		if kind.recordType == "SOA" && name != "@" {
			return "The SOA record set must be named '@'."
		}
		if kind.recordType == "CNAME" && name == "@" {
			return "A CNAME record set cannot be created at the zone apex."
		}
		if name == "@" {
			return ""
		}
		for i, label := range strings.Split(name, ".") {
			if label == "" || len(label) > 63 || (label == "*" && i != 0) {
				return "The record set name '" + name + "' is invalid."
			}
			for _, c := range label {
				if !(c == '-' || c == '_' || c == '*' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
					return "The record set name '" + name + "' is invalid."
				}
			}
		}
		return ""
	}
}

// adnsValidateRecordSet checks a record set PUT: the TTL, that only the
// set's own record member is present, and each record.
func adnsValidateRecordSet(f adnsZoneFlavor, kind adnsRecordKind, payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if raw, ok := props[f.ttlMember]; ok && raw != nil {
		ttl, ok := raw.(float64)
		if !ok || ttl < 0 || ttl > adnsMaxTTL || ttl != float64(int64(ttl)) {
			return f.ttlMember + " must be an integer between 0 and 2147483647."
		}
	}
	for _, other := range f.kinds {
		if other.member != kind.member && props[other.member] != nil {
			return "A record set of type " + kind.recordType + " cannot contain " + other.member + "."
		}
		// Another spelling of a record member would skip validation, e.g.
		// ARecords sent to a private zone, whose member is aRecords.
		for key := range props {
			if key != other.member && strings.EqualFold(key, other.member) {
				return "Unknown property " + key + "; this zone type uses " + other.member + "."
			}
		}
	}
	if target := props["targetResource"]; target != nil {
		if !f.allowTarget {
			return "targetResource is not supported for this zone type."
		}
		if id := azureARMString(azureARMObject(target)["id"]); id != "" && !strings.HasPrefix(strings.ToLower(id), "/subscriptions/") {
			return "targetResource.id must be an Azure resource ID."
		}
	}
	raw := props[kind.member]
	if raw == nil {
		return ""
	}
	if kind.single {
		rec, ok := raw.(map[string]any)
		if !ok {
			return kind.member + " must be an object."
		}
		return kind.check(rec)
	}
	records, ok := raw.([]any)
	if !ok {
		return kind.member + " must be an array."
	}
	if len(records) > adnsMaxRecordsPerSet {
		return fmt.Sprintf("A record set can hold at most %d records.", adnsMaxRecordsPerSet)
	}
	for _, item := range records {
		rec, ok := item.(map[string]any)
		if !ok {
			return kind.member + " items must be objects."
		}
		if msg := kind.check(rec); msg != "" {
			return msg
		}
	}
	return ""
}

// adnsRecordSetProperties derives fqdn from the record set ID:
// {zoneId}/{recordType}/{relativeName}.
func adnsRecordSetProperties(rec *azureARMResource) map[string]any {
	parts := strings.Split(rec.ID, "/")
	zone := parts[len(parts)-3]
	fqdn := zone + "."
	if rec.Name != "@" {
		fqdn = rec.Name + "." + fqdn
	}
	return map[string]any{"fqdn": fqdn}
}

// adnsNameServers returns the four deterministic name servers Azure would
// assign a public zone.
func adnsNameServers(zoneID string) []any {
	sum := sha256.Sum256([]byte(azureARMKey(zoneID)))
	n := int(sum[0])%9 + 1
	return []any{
		fmt.Sprintf("ns1-0%d.azure-dns.com.", n), fmt.Sprintf("ns2-0%d.azure-dns.net.", n),
		fmt.Sprintf("ns3-0%d.azure-dns.org.", n), fmt.Sprintf("ns4-0%d.azure-dns.info.", n),
	}
}

func adnsCheckAll(checks ...func(map[string]any) string) func(map[string]any) string {
	return func(rec map[string]any) string {
		for _, check := range checks {
			if msg := check(rec); msg != "" {
				return msg
			}
		}
		return ""
	}
}

func adnsCheckIP(field string, v4 bool) func(map[string]any) string {
	return func(rec map[string]any) string {
		addr, err := netip.ParseAddr(azureARMString(rec[field]))
		if err != nil || addr.Is4() != v4 || addr.Is4In6() {
			return field + " '" + azureARMString(rec[field]) + "' is invalid."
		}
		return ""
	}
}

func adnsCheckHost(field string) func(map[string]any) string {
	return func(rec map[string]any) string {
		host := azureARMString(rec[field])
		if host == "" || len(host) > 255 || strings.ContainsAny(host, " /\\@") {
			return field + " is missing or invalid."
		}
		return ""
	}
}

func adnsCheckUint(field string, max float64) func(map[string]any) string {
	return func(rec map[string]any) string {
		v, ok := rec[field].(float64)
		if !ok || v < 0 || v > max || v != float64(int64(v)) {
			return fmt.Sprintf("%s must be an integer between 0 and %d.", field, int64(max))
		}
		return ""
	}
}

func adnsCheckTXT(rec map[string]any) string {
	values, ok := rec["value"].([]any)
	if !ok || len(values) == 0 {
		return "TXT records need a non-empty value array."
	}
	for _, v := range values {
		text, ok := v.(string)
		if !ok || len(text) > adnsMaxTXTValueLen {
			return "TXT record values must be strings of at most 1024 characters."
		}
	}
	return ""
}

func adnsCheckCAA(rec map[string]any) string {
	if msg := adnsCheckUint("flags", 255)(rec); msg != "" {
		return msg
	}
	switch azureARMString(rec["tag"]) {
	case "issue", "issuewild", "iodef":
		return ""
	}
	return "CAA tag must be issue, issuewild or iodef."
}

func adnsCheckSOA(rec map[string]any) string {
	for _, field := range []string{"serialNumber", "refreshTime", "retryTime", "expireTime"} {
		if rec[field] != nil {
			if msg := adnsCheckUint(field, adnsMaxTTL)(rec); msg != "" {
				return msg
			}
		}
	}
	return ""
}

// adnsDefaultSOA is the apex SOA record set Azure creates with a zone.
func adnsDefaultSOA(f adnsZoneFlavor, host, email string, minimumTTL int) map[string]any {
	soa := map[string]any{
		"host": host, "email": email, "serialNumber": 1, "refreshTime": 3600,
		"retryTime": 300, "expireTime": 2419200,
	}
	if f.ttlMember == "TTL" {
		soa["minimumTTL"] = minimumTTL
		return map[string]any{"TTL": 3600, "SOARecord": soa}
	}
	soa["minimumTtl"] = minimumTTL
	return map[string]any{"ttl": 3600, "soaRecord": soa}
}

func adnsKind(f adnsZoneFlavor, recordType string) adnsRecordKind {
	for _, kind := range f.kinds {
		if kind.recordType == recordType {
			return kind
		}
	}
	panic("unknown DNS record type " + recordType)
}

func adnsRecordType(f adnsZoneFlavor, recordType string) azureARMResourceType {
	typ, _ := lookupAzureARMResourceType(f.namespace, f.zoneType+"/"+adnsKind(f, recordType).recordType)
	return typ
}
