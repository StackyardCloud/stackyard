package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Core network policy limits (well above any realistic policy).
const (
	nmMaxSegments = 200
	nmMaxEdges    = 100
	nmMaxRules    = 1000
)

var nmSegmentNameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]{0,63}$`)

type nmEdge struct {
	location string
	asn      int64
}

type nmSegment struct {
	name              string
	edges             []string
	shared            []string
	requireAcceptance bool
}

type nmRule struct {
	number            int
	logic             string
	conditions        []map[string]any
	method            string
	segment           string
	tagKey            string
	requireAcceptance bool
}

// nmPolicyDoc is the parsed subset of a core network policy that Stackyard
// acts on: edges (with ASNs), segments, sharing and attachment policies.
type nmPolicyDoc struct {
	edges    []nmEdge
	segments []nmSegment
	rules    []nmRule
}

func (d *nmPolicyDoc) edge(location string) *nmEdge {
	for i := range d.edges {
		if d.edges[i].location == location {
			return &d.edges[i]
		}
	}
	return nil
}

func (d *nmPolicyDoc) segment(name string) *nmSegment {
	for i := range d.segments {
		if d.segments[i].name == name {
			return &d.segments[i]
		}
	}
	return nil
}

type nmPolicyErrors []any

func (e *nmPolicyErrors) add(path, code, format string, args ...any) {
	if len(*e) < 50 {
		*e = append(*e, map[string]any{"ErrorCode": code, "Message": fmt.Sprintf(format, args...), "Path": path})
	}
}

func nmPolicyException(errs nmPolicyErrors) *nmErr {
	return &nmErr{status: http.StatusBadRequest, code: "CoreNetworkPolicyException",
		msg: "The core network policy document is invalid.", extra: map[string]any{"Errors": []any(errs)}}
}

func nmParseASNRange(v string) (int64, int64, bool) {
	lo, hi, ok := strings.Cut(v, "-")
	if !ok {
		return 0, 0, false
	}
	a, err1 := strconv.ParseInt(strings.TrimSpace(lo), 10, 64)
	b, err2 := strconv.ParseInt(strings.TrimSpace(hi), 10, 64)
	private := func(n int64) bool { return (n >= 64512 && n <= 65534) || (n >= 4200000000 && n <= 4294967294) }
	return a, b, err1 == nil && err2 == nil && a <= b && private(a) && private(b)
}

// nmParsePolicy checks the structure AWS requires of a core network policy
// (version, core-network-configuration with asn-ranges and edge-locations,
// segments, segment-actions, attachment-policies) and returns what Stackyard
// uses from it.
func nmParsePolicy(raw string) (*nmPolicyDoc, *nmErr) {
	var errs nmPolicyErrors
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil || doc == nil {
		errs.add("", "INVALID_JSON", "The policy document isn't a JSON object.")
		return nil, nmPolicyException(errs)
	}
	if v, _ := doc["version"].(string); v != "2021.12" && v != "2025.11" {
		errs.add("version", "INVALID_VERSION", "version must be 2021.12 or 2025.11.")
	}
	out := &nmPolicyDoc{}
	cfg, _ := doc["core-network-configuration"].(map[string]any)
	if cfg == nil {
		errs.add("core-network-configuration", "MISSING_FIELD", "core-network-configuration is required.")
	}
	var asnNext int64
	ranges, _ := cfg["asn-ranges"].([]any)
	if len(ranges) == 0 {
		errs.add("core-network-configuration.asn-ranges", "MISSING_FIELD", "asn-ranges needs at least one range.")
	}
	for i, r := range ranges {
		s, _ := r.(string)
		lo, _, ok := nmParseASNRange(s)
		if !ok {
			errs.add(fmt.Sprintf("core-network-configuration.asn-ranges[%d]", i), "INVALID_ASN_RANGE", "%q isn't a private ASN range like 64512-65534.", s)
			continue
		}
		if asnNext == 0 {
			asnNext = lo
		}
	}
	edges, _ := cfg["edge-locations"].([]any)
	if len(edges) == 0 || len(edges) > nmMaxEdges {
		errs.add("core-network-configuration.edge-locations", "INVALID_EDGE_LOCATIONS", "edge-locations needs 1 to %d entries.", nmMaxEdges)
	}
	for i, e := range edges {
		em, _ := e.(map[string]any)
		loc, _ := em["location"].(string)
		path := fmt.Sprintf("core-network-configuration.edge-locations[%d]", i)
		if loc == "" {
			errs.add(path+".location", "MISSING_FIELD", "Each edge location needs a location.")
			continue
		}
		if out.edge(loc) != nil {
			errs.add(path+".location", "DUPLICATE_EDGE_LOCATION", "Edge location %s is listed twice.", loc)
			continue
		}
		asn := asnNext
		if v, ok := em["asn"].(float64); ok {
			asn = int64(v)
		} else if asnNext != 0 {
			asnNext++
		}
		out.edges = append(out.edges, nmEdge{location: loc, asn: asn})
	}
	segments, _ := doc["segments"].([]any)
	if len(segments) == 0 || len(segments) > nmMaxSegments {
		errs.add("segments", "INVALID_SEGMENTS", "segments needs 1 to %d entries.", nmMaxSegments)
	}
	for i, s := range segments {
		sm, _ := s.(map[string]any)
		name, _ := sm["name"].(string)
		path := fmt.Sprintf("segments[%d]", i)
		if !nmSegmentNameRe.MatchString(name) {
			errs.add(path+".name", "INVALID_SEGMENT_NAME", "Segment name %q must start with a letter and hold only letters and digits.", name)
			continue
		}
		if out.segment(name) != nil {
			errs.add(path+".name", "DUPLICATE_SEGMENT", "Segment %s is defined twice.", name)
			continue
		}
		seg := nmSegment{name: name, requireAcceptance: sm["require-attachment-acceptance"] == true}
		for _, e := range nmAnyStrings(sm["edge-locations"]) {
			if out.edge(e) == nil {
				errs.add(path+".edge-locations", "UNKNOWN_EDGE_LOCATION", "Edge location %s isn't in core-network-configuration.", e)
			}
			seg.edges = append(seg.edges, e)
		}
		out.segments = append(out.segments, seg)
	}
	actions, _ := doc["segment-actions"].([]any)
	for i, a := range actions {
		am, _ := a.(map[string]any)
		path := fmt.Sprintf("segment-actions[%d]", i)
		action, _ := am["action"].(string)
		if !slices.Contains([]string{"share", "create-route", "send-via", "send-to", "associate-routing-policy"}, action) {
			errs.add(path+".action", "INVALID_ACTION", "Segment action %q isn't supported.", action)
			continue
		}
		name, _ := am["segment"].(string)
		seg := out.segment(name)
		if seg == nil && action != "send-via" && action != "send-to" {
			errs.add(path+".segment", "UNKNOWN_SEGMENT", "Segment %q isn't defined.", name)
			continue
		}
		if action == "share" {
			for _, with := range nmAnyStrings(am["share-with"]) {
				if with != "*" && out.segment(with) == nil {
					errs.add(path+".share-with", "UNKNOWN_SEGMENT", "Segment %q isn't defined.", with)
					continue
				}
				seg.shared = append(seg.shared, with)
			}
		}
	}
	rules, _ := doc["attachment-policies"].([]any)
	if len(rules) > nmMaxRules {
		errs.add("attachment-policies", "TOO_MANY_RULES", "attachment-policies can hold at most %d rules.", nmMaxRules)
		rules = rules[:nmMaxRules]
	}
	seen := map[int]bool{}
	for i, r := range rules {
		rm, _ := r.(map[string]any)
		path := fmt.Sprintf("attachment-policies[%d]", i)
		n, ok := rm["rule-number"].(float64)
		if !ok || n < 1 || n > 65535 || n != float64(int(n)) {
			errs.add(path+".rule-number", "INVALID_RULE_NUMBER", "rule-number must be an integer from 1 to 65535.")
			continue
		}
		if seen[int(n)] {
			errs.add(path+".rule-number", "DUPLICATE_RULE_NUMBER", "rule-number %d is used twice.", int(n))
			continue
		}
		seen[int(n)] = true
		rule := nmRule{number: int(n), logic: "or"}
		if l, _ := rm["condition-logic"].(string); l != "" {
			rule.logic = l
		}
		conds, _ := rm["conditions"].([]any)
		if len(conds) == 0 {
			errs.add(path+".conditions", "MISSING_FIELD", "Each attachment policy needs at least one condition.")
		}
		for _, c := range conds {
			cm, _ := c.(map[string]any)
			typ, _ := cm["type"].(string)
			if !slices.Contains([]string{"any", "tag-exists", "tag-value", "attachment-type", "region", "account-id", "resource-id"}, typ) {
				errs.add(path+".conditions", "INVALID_CONDITION", "Condition type %q isn't supported.", typ)
			}
			rule.conditions = append(rule.conditions, cm)
		}
		act, _ := rm["action"].(map[string]any)
		rule.method, _ = act["association-method"].(string)
		rule.segment, _ = act["segment"].(string)
		rule.tagKey, _ = act["tag-value-of-key"].(string)
		rule.requireAcceptance = act["require-acceptance"] == true
		switch rule.method {
		case "constant":
			if out.segment(rule.segment) == nil {
				errs.add(path+".action.segment", "UNKNOWN_SEGMENT", "Segment %q isn't defined.", rule.segment)
			}
		case "tag":
			if rule.tagKey == "" {
				errs.add(path+".action.tag-value-of-key", "MISSING_FIELD", "association-method tag needs tag-value-of-key.")
			}
		default:
			errs.add(path+".action.association-method", "INVALID_ASSOCIATION_METHOD", "association-method must be constant or tag.")
		}
		out.rules = append(out.rules, rule)
	}
	if len(errs) > 0 {
		return nil, nmPolicyException(errs)
	}
	sort.Slice(out.rules, func(i, j int) bool { return out.rules[i].number < out.rules[j].number })
	return out, nil
}

func nmAnyStrings(v any) []string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// mapAttachment applies the attachment policies to an attachment: the first
// matching rule gives its segment and whether acceptance is required.
func (d *nmPolicyDoc) mapAttachment(attType, edge, resourceID string, tags map[string]string) (segment string, rule int, accept bool) {
	for _, r := range d.rules {
		matched := r.logic == "and"
		for _, c := range r.conditions {
			ok := nmConditionMatches(c, attType, edge, resourceID, tags)
			if r.logic == "and" {
				matched = matched && ok
			} else {
				matched = matched || ok
			}
		}
		if !matched {
			continue
		}
		seg := r.segment
		if r.method == "tag" {
			seg = tags[r.tagKey]
		}
		s := d.segment(seg)
		if s == nil {
			continue
		}
		return seg, r.number, r.requireAcceptance || s.requireAcceptance
	}
	return "", 0, false
}

func nmConditionMatches(c map[string]any, attType, edge, resourceID string, tags map[string]string) bool {
	key, _ := c["key"].(string)
	value, _ := c["value"].(string)
	op, _ := c["operator"].(string)
	cmp := func(got string) bool {
		switch op {
		case "not-equals":
			return got != value
		case "contains":
			return strings.Contains(got, value)
		case "begins-with":
			return strings.HasPrefix(got, value)
		}
		return got == value
	}
	switch c["type"] {
	case "any":
		return true
	case "tag-exists":
		_, ok := tags[key]
		return ok
	case "tag-value":
		got, ok := tags[key]
		return ok && cmp(got)
	case "attachment-type":
		return cmp(attType)
	case "region":
		return cmp(edge)
	case "account-id":
		return cmp(nmAccount)
	case "resource-id":
		return cmp(resourceID)
	}
	return false
}

// nmSegmentsOut and nmEdgesOut describe a LIVE policy on its core network.
func nmSegmentsOut(d *nmPolicyDoc) []any {
	out := []any{}
	if d == nil {
		return out
	}
	for _, s := range d.segments {
		edges := s.edges
		if len(edges) == 0 {
			for _, e := range d.edges {
				edges = append(edges, e.location)
			}
		}
		out = append(out, map[string]any{"Name": s.name, "EdgeLocations": nmStringsAny(edges), "SharedSegments": nmStringsAny(s.shared)})
	}
	return out
}

func nmEdgesOut(d *nmPolicyDoc) []any {
	out := []any{}
	if d == nil {
		return out
	}
	for _, e := range d.edges {
		out = append(out, map[string]any{"EdgeLocation": e.location, "Asn": float64(e.asn), "InsideCidrBlocks": []any{}})
	}
	return out
}

func nmStringsAny(v []string) []any {
	out := make([]any, 0, len(v))
	for _, s := range v {
		out = append(out, s)
	}
	return out
}

// nmChanges lists what applying next over prev would change: segments and
// edges added, removed or modified.
func nmChanges(prev, next *nmPolicyDoc) []any {
	if prev == nil {
		prev = &nmPolicyDoc{}
	}
	var out []any
	for _, s := range next.segments {
		values := map[string]any{"SegmentName": s.name, "EdgeLocations": nmStringsAny(s.edges), "SharedSegments": nmStringsAny(s.shared)}
		if p := prev.segment(s.name); p == nil {
			out = append(out, nmChange("CORE_NETWORK_SEGMENT", "ADD", s.name, "/segments/"+s.name, nil, values))
		} else if !slices.Equal(p.edges, s.edges) || !slices.Equal(p.shared, s.shared) {
			out = append(out, nmChange("CORE_NETWORK_SEGMENT", "MODIFY", s.name, "/segments/"+s.name,
				map[string]any{"SegmentName": p.name, "EdgeLocations": nmStringsAny(p.edges), "SharedSegments": nmStringsAny(p.shared)}, values))
		}
	}
	for _, p := range prev.segments {
		if next.segment(p.name) == nil {
			out = append(out, nmChange("CORE_NETWORK_SEGMENT", "REMOVE", p.name, "/segments/"+p.name, map[string]any{"SegmentName": p.name}, nil))
		}
	}
	for _, e := range next.edges {
		values := map[string]any{"EdgeLocations": []any{e.location}, "Asn": float64(e.asn)}
		if p := prev.edge(e.location); p == nil {
			out = append(out, nmChange("CORE_NETWORK_EDGE", "ADD", e.location, "/coreNetworkConfiguration/edgeLocations/"+e.location, nil, values))
		} else if p.asn != e.asn {
			out = append(out, nmChange("CORE_NETWORK_EDGE", "MODIFY", e.location, "/coreNetworkConfiguration/edgeLocations/"+e.location,
				map[string]any{"EdgeLocations": []any{p.location}, "Asn": float64(p.asn)}, values))
		}
	}
	for _, p := range prev.edges {
		if next.edge(p.location) == nil {
			out = append(out, nmChange("CORE_NETWORK_EDGE", "REMOVE", p.location, "/coreNetworkConfiguration/edgeLocations/"+p.location,
				map[string]any{"EdgeLocations": []any{p.location}}, nil))
		}
	}
	return out
}

func nmChange(typ, action, id, path string, prev, next map[string]any) map[string]any {
	c := map[string]any{"Type": typ, "Action": action, "Identifier": id, "IdentifierPath": path}
	if prev != nil {
		c["PreviousValues"] = prev
	}
	if next != nil {
		c["NewValues"] = next
	}
	return c
}
