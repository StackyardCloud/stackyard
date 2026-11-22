package server

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// osDomain is a domain shared by the OpenSearch (2021-01-01) and legacy
// Elasticsearch Service (2015-01-01) APIs. Option blocks are stored under the
// member names both APIs share; the cluster config is kept in its OpenSearch
// form (".search" instance types) and translated for the es API.
type osDomain struct {
	name     string
	engine   string         // EngineVersion form: Elasticsearch_7.10 or OpenSearch_2.11
	cluster  map[string]any // OpenSearch ClusterConfig
	options  map[string]any // EBSOptions, AccessPolicies, SnapshotOptions, ...
	created  time.Time
	updated  time.Time
	version  int // config UpdateVersion
	reads    int // describes since the last change; Processing until the first
	changeID string
	tags     map[string]string
	upgrades []map[string]any // newest last
	access   []string         // VPC endpoint access accounts
	packages map[string]time.Time
}

// osDomainOptionKeys are the option blocks both APIs use under the same name.
var osDomainOptionKeys = []string{
	"EBSOptions", "AccessPolicies", "SnapshotOptions", "VPCOptions", "CognitoOptions",
	"EncryptionAtRestOptions", "NodeToNodeEncryptionOptions", "AdvancedOptions",
	"LogPublishingOptions", "DomainEndpointOptions", "AdvancedSecurityOptions", "AutoTuneOptions",
}

const (
	osMaxDomains        = 100
	osDomainAccountID   = "123456789012"
	osDomainRegion      = "us-east-1"
	osDefaultOSEngine   = "OpenSearch_2.11"
	osMaxUpgradeHistory = 20
)

func osDomainARN(name string) string {
	return fmt.Sprintf("arn:aws:es:%s:%s:domain/%s", osDomainRegion, osDomainAccountID, name)
}

// newOSDomainLocked registers a domain. options are copied, and any master
// user credentials in AdvancedSecurityOptions are dropped (never stored).
func (s *openSearchStore) newOSDomainLocked(name, engine string, cluster map[string]any, options map[string]any) *osDomain {
	if s.domains == nil {
		s.domains = map[string]*osDomain{}
	}
	now := time.Now().UTC()
	d := &osDomain{
		name: name, engine: engine, cluster: osCopyMap(cluster), options: map[string]any{},
		created: now, updated: now, version: 1, tags: map[string]string{}, packages: map[string]time.Time{},
	}
	d.setOptions(options)
	d.changeID = osNewChangeID()
	s.domains[name] = d
	return d
}

func (d *osDomain) setOptions(options map[string]any) {
	for _, key := range osDomainOptionKeys {
		v, ok := options[key]
		if !ok || v == nil {
			continue
		}
		if key == "AdvancedSecurityOptions" {
			if m, ok := v.(map[string]any); ok {
				m = osCopyMap(m)
				delete(m, "MasterUserOptions")
				v = m
			}
		}
		d.options[key] = v
	}
}

// touch records a configuration change: a new UpdateVersion and change id, and
// Processing until the next describe.
func (d *osDomain) touch() {
	d.updated = time.Now().UTC()
	d.version++
	d.reads = 0
	d.changeID = osNewChangeID()
}

// read reports whether the domain is still processing, then settles it.
// ponytail: changes settle on the next describe; no background processing.
func (d *osDomain) read() bool {
	processing := d.reads == 0
	d.reads++
	return processing
}

func (d *osDomain) engineType() string {
	if strings.HasPrefix(d.engine, "OpenSearch_") {
		return "OpenSearch"
	}
	return "Elasticsearch"
}

// endpointFields returns Endpoint, or Endpoints for a VPC domain.
func (d *osDomain) endpointFields(out map[string]any) {
	host := fmt.Sprintf("search-%s-%s.%s.es.amazonaws.com", d.name, osShortID(d.name), osDomainRegion)
	if vpc, ok := d.options["VPCOptions"].(map[string]any); ok && len(vpc) > 0 {
		out["Endpoints"] = map[string]any{"vpc": "vpc-" + strings.TrimPrefix(host, "search-")}
		return
	}
	out["Endpoint"] = host
}

func (d *osDomain) vpcDerived() map[string]any {
	vpc, ok := d.options["VPCOptions"].(map[string]any)
	if !ok {
		return nil
	}
	out := osCopyMap(vpc)
	out["VPCId"] = "vpc-0" + osShortID(d.name)[:7]
	out["AvailabilityZones"] = []any{osDomainRegion + "a"}
	return out
}

// osStatus is the OpenSearch API DomainStatus.
func (d *osDomain) osStatus(processing bool) map[string]any {
	out := map[string]any{
		"DomainId": osDomainAccountID + "/" + d.name, "DomainName": d.name, "ARN": osDomainARN(d.name),
		"Created": true, "Deleted": false, "Processing": processing, "UpgradeProcessing": false,
		"EngineVersion": d.engine, "ClusterConfig": osCopyMap(d.cluster),
	}
	for k, v := range d.options {
		out[k] = v
	}
	if vpc := d.vpcDerived(); vpc != nil {
		out["VPCOptions"] = vpc
	}
	d.endpointFields(out)
	return out
}

func osShortID(name string) string {
	var h uint32 = 2166136261
	for i := 0; i < len(name); i++ {
		h = (h ^ uint32(name[i])) * 16777619
	}
	return fmt.Sprintf("%08x%08x", h, h*2654435761)[:12]
}

func osNewChangeID() string { return uuid.NewString() }

func osCopyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (s *openSearchStore) sortedDomainsLocked() []*osDomain {
	out := make([]*osDomain, 0, len(s.domains))
	for _, d := range s.domains {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// osHandleDomainAction serves the OpenSearch domain operations from the shared
// registry. Names that were never registered keep the original stub answer so
// existing OpenSearch behaviour is unchanged.
func (s *openSearchStore) osHandleDomainAction(action, domainName string, payload map[string]any) (map[string]any, bool) {
	switch action {
	case "CreateDomain":
		d := s.domains[domainName]
		if d == nil && len(s.domains) < osMaxDomains {
			cluster, _ := payload["ClusterConfig"].(map[string]any)
			engine := opensearchFirstNonEmpty(opensearchPayloadString(payload, "EngineVersion"), osDefaultOSEngine)
			d = s.newOSDomainLocked(domainName, engine, cluster, payload)
		}
		if d == nil {
			return nil, false
		}
		return map[string]any{"DomainStatus": d.osStatus(true)}, true
	case "DescribeDomain":
		if d := s.domains[domainName]; d != nil {
			return map[string]any{"DomainStatus": d.osStatus(d.read())}, true
		}
	case "DeleteDomain":
		if d := s.domains[domainName]; d != nil {
			status := d.osStatus(true)
			status["Deleted"] = true
			delete(s.domains, domainName)
			return map[string]any{"DomainStatus": status}, true
		}
	case "DescribeDomains":
		names, _ := payload["DomainNames"].([]any)
		if len(names) == 0 || len(s.domains) == 0 {
			return nil, false
		}
		list := []any{}
		for _, n := range names {
			name, _ := n.(string)
			if d := s.domains[name]; d != nil {
				list = append(list, d.osStatus(d.read()))
			} else {
				list = append(list, s.domainStatus(name))
			}
		}
		return map[string]any{"DomainStatusList": list}, true
	case "ListDomainNames":
		if len(s.domains) == 0 {
			return nil, false
		}
		list := []any{}
		for _, d := range s.sortedDomainsLocked() {
			list = append(list, map[string]any{"DomainName": d.name, "EngineType": d.engineType()})
		}
		return map[string]any{"DomainNames": list}, true
	}
	return nil, false
}
