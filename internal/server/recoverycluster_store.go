package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	recoveryClusterDefaultClusterARN      = "cluster-000001"
	recoveryClusterDefaultControlPanelARN = "controlpanel-000001"
	recoveryClusterDefaultRoutingCtrlARN  = "routingcontrol-000001"
	recoveryClusterDefaultSafetyRuleARN   = "safetyrule-000001"
	recoveryClusterDefaultPolicy          = `{"Version":"2012-10-17","Statement":[]}`
)

type recoveryClusterStore struct {
	mu sync.Mutex

	nextClusterID      int64
	nextControlPanelID int64
	nextRoutingCtrlID  int64
	nextSafetyRuleID   int64

	clusters               map[string]map[string]any
	controlPanels          map[string]map[string]any
	routingControls        map[string]map[string]any
	safetyRules            map[string]map[string]any
	associatedHealthChecks map[string][]map[string]any
	resourcePolicies       map[string]string
	tags                   map[string]map[string]string
}

func newRecoveryClusterStore() *recoveryClusterStore {
	s := &recoveryClusterStore{
		nextClusterID:      2,
		nextControlPanelID: 2,
		nextRoutingCtrlID:  2,
		nextSafetyRuleID:   2,
		clusters:           map[string]map[string]any{},
		controlPanels:      map[string]map[string]any{},
		routingControls:    map[string]map[string]any{},
		safetyRules:        map[string]map[string]any{},
		associatedHealthChecks: map[string][]map[string]any{
			recoveryClusterDefaultRoutingCtrlARN: {
				{"HealthCheckArn": "arn:aws:route53:::healthcheck/hc-000001"},
			},
		},
		resourcePolicies: map[string]string{},
		tags:             map[string]map[string]string{},
	}
	s.seedDefaultsLocked(time.Now().UTC().Format(time.RFC3339))
	return s
}

func (s *recoveryClusterStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.syncPayloadWithQuery(payload, query)
	now := time.Now().UTC().Format(time.RFC3339)
	s.seedDefaultsLocked(now)

	switch action {
	case "CreateCluster":
		name := rcFirstNonEmpty(rcPayloadString(payload, "ClusterName", ""), fmt.Sprintf("stackyard-cluster-%06d", s.nextClusterID))
		clusterARN := fmt.Sprintf("cluster-%06d", s.nextClusterID)
		s.nextClusterID++
		cluster := s.ensureClusterLocked(clusterARN, now)
		cluster["ClusterName"] = name
		if networkType := rcPayloadString(payload, "NetworkType", ""); networkType != "" {
			cluster["NetworkType"] = networkType
		}
		s.mergeTagsLocked(clusterARN, payload)
		return map[string]any{"Cluster": rcCloneMap(cluster)}

	case "CreateControlPanel":
		clusterARN := rcFirstNonEmpty(rcPayloadString(payload, "ClusterArn", ""), recoveryClusterDefaultClusterARN)
		s.ensureClusterLocked(clusterARN, now)
		controlPanelARN := fmt.Sprintf("controlpanel-%06d", s.nextControlPanelID)
		s.nextControlPanelID++
		controlPanel := s.ensureControlPanelLocked(controlPanelARN, clusterARN, now)
		if name := rcPayloadString(payload, "ControlPanelName", ""); name != "" {
			controlPanel["ControlPanelName"] = name
		}
		s.mergeTagsLocked(controlPanelARN, payload)
		return map[string]any{"ControlPanel": rcCloneMap(controlPanel)}

	case "CreateRoutingControl":
		clusterARN := rcFirstNonEmpty(rcPayloadString(payload, "ClusterArn", ""), recoveryClusterDefaultClusterARN)
		controlPanelARN := rcFirstNonEmpty(rcPayloadString(payload, "ControlPanelArn", ""), recoveryClusterDefaultControlPanelARN)
		s.ensureClusterLocked(clusterARN, now)
		s.ensureControlPanelLocked(controlPanelARN, clusterARN, now)
		routingControlARN := fmt.Sprintf("routingcontrol-%06d", s.nextRoutingCtrlID)
		s.nextRoutingCtrlID++
		routingControl := s.ensureRoutingControlLocked(routingControlARN, controlPanelARN, clusterARN, now)
		if name := rcPayloadString(payload, "RoutingControlName", ""); name != "" {
			routingControl["RoutingControlName"] = name
		}
		s.mergeTagsLocked(routingControlARN, payload)
		return map[string]any{"RoutingControl": rcCloneMap(routingControl)}

	case "CreateSafetyRule":
		controlPanelARN := rcFirstNonEmpty(rcPayloadString(payload, "ControlPanelArn", ""), recoveryClusterDefaultControlPanelARN)
		clusterARN := rcFirstNonEmpty(rcPayloadString(payload, "ClusterArn", ""), recoveryClusterDefaultClusterARN)
		s.ensureClusterLocked(clusterARN, now)
		s.ensureControlPanelLocked(controlPanelARN, clusterARN, now)
		safetyRuleARN := fmt.Sprintf("safetyrule-%06d", s.nextSafetyRuleID)
		s.nextSafetyRuleID++
		safetyRule := s.ensureSafetyRuleLocked(safetyRuleARN, controlPanelARN, now)
		if name := rcPayloadString(payload, "Name", ""); name != "" {
			safetyRule["Name"] = name
		}
		if assertion, ok := rcLookupCI(payload, "AssertionRule"); ok {
			safetyRule["AssertionRule"] = rcCloneAny(assertion)
			safetyRule["RuleType"] = "ASSERTION"
		}
		if gating, ok := rcLookupCI(payload, "GatingRule"); ok {
			safetyRule["GatingRule"] = rcCloneAny(gating)
			safetyRule["RuleType"] = "GATING"
		}
		s.mergeTagsLocked(safetyRuleARN, payload)
		return map[string]any{"SafetyRule": rcCloneMap(safetyRule)}

	case "DeleteCluster":
		clusterARN := rcFirstNonEmpty(rcPathString(pathParams, "ClusterArn", ""), rcPayloadString(payload, "ClusterArn", ""), recoveryClusterDefaultClusterARN)
		delete(s.clusters, clusterARN)
		delete(s.tags, clusterARN)
		for controlPanelARN, controlPanel := range s.controlPanels {
			if rcAnyString(controlPanel["ClusterArn"]) != clusterARN {
				continue
			}
			delete(s.controlPanels, controlPanelARN)
			delete(s.tags, controlPanelARN)
			for routingControlARN, routingControl := range s.routingControls {
				if rcAnyString(routingControl["ControlPanelArn"]) != controlPanelARN {
					continue
				}
				delete(s.routingControls, routingControlARN)
				delete(s.associatedHealthChecks, routingControlARN)
				delete(s.tags, routingControlARN)
			}
			for safetyRuleARN, safetyRule := range s.safetyRules {
				if rcAnyString(safetyRule["ControlPanelArn"]) != controlPanelARN {
					continue
				}
				delete(s.safetyRules, safetyRuleARN)
				delete(s.tags, safetyRuleARN)
			}
		}
		return map[string]any{}

	case "DeleteControlPanel":
		controlPanelARN := rcFirstNonEmpty(rcPathString(pathParams, "ControlPanelArn", ""), rcPayloadString(payload, "ControlPanelArn", ""), recoveryClusterDefaultControlPanelARN)
		delete(s.controlPanels, controlPanelARN)
		delete(s.tags, controlPanelARN)
		for routingControlARN, routingControl := range s.routingControls {
			if rcAnyString(routingControl["ControlPanelArn"]) != controlPanelARN {
				continue
			}
			delete(s.routingControls, routingControlARN)
			delete(s.associatedHealthChecks, routingControlARN)
			delete(s.tags, routingControlARN)
		}
		for safetyRuleARN, safetyRule := range s.safetyRules {
			if rcAnyString(safetyRule["ControlPanelArn"]) != controlPanelARN {
				continue
			}
			delete(s.safetyRules, safetyRuleARN)
			delete(s.tags, safetyRuleARN)
		}
		return map[string]any{}

	case "DeleteRoutingControl":
		routingControlARN := rcFirstNonEmpty(rcPathString(pathParams, "RoutingControlArn", ""), rcPayloadString(payload, "RoutingControlArn", ""), recoveryClusterDefaultRoutingCtrlARN)
		delete(s.routingControls, routingControlARN)
		delete(s.associatedHealthChecks, routingControlARN)
		delete(s.tags, routingControlARN)
		return map[string]any{}

	case "DeleteSafetyRule":
		safetyRuleARN := rcFirstNonEmpty(rcPathString(pathParams, "SafetyRuleArn", ""), rcPayloadString(payload, "SafetyRuleArn", ""), recoveryClusterDefaultSafetyRuleARN)
		delete(s.safetyRules, safetyRuleARN)
		delete(s.tags, safetyRuleARN)
		return map[string]any{}

	case "DescribeCluster":
		clusterARN := rcFirstNonEmpty(rcPathString(pathParams, "ClusterArn", ""), rcPayloadString(payload, "ClusterArn", ""), recoveryClusterDefaultClusterARN)
		return map[string]any{"Cluster": rcCloneMap(s.ensureClusterLocked(clusterARN, now))}

	case "DescribeControlPanel":
		controlPanelARN := rcFirstNonEmpty(rcPathString(pathParams, "ControlPanelArn", ""), rcPayloadString(payload, "ControlPanelArn", ""), recoveryClusterDefaultControlPanelARN)
		controlPanel := s.ensureControlPanelLocked(controlPanelARN, recoveryClusterDefaultClusterARN, now)
		return map[string]any{"ControlPanel": rcCloneMap(controlPanel)}

	case "DescribeRoutingControl":
		routingControlARN := rcFirstNonEmpty(rcPathString(pathParams, "RoutingControlArn", ""), rcPayloadString(payload, "RoutingControlArn", ""), recoveryClusterDefaultRoutingCtrlARN)
		routingControl := s.ensureRoutingControlLocked(routingControlARN, recoveryClusterDefaultControlPanelARN, recoveryClusterDefaultClusterARN, now)
		return map[string]any{"RoutingControl": rcCloneMap(routingControl)}

	case "DescribeSafetyRule":
		safetyRuleARN := rcFirstNonEmpty(rcPathString(pathParams, "SafetyRuleArn", ""), rcPayloadString(payload, "SafetyRuleArn", ""), recoveryClusterDefaultSafetyRuleARN)
		safetyRule := s.ensureSafetyRuleLocked(safetyRuleARN, recoveryClusterDefaultControlPanelARN, now)
		return map[string]any{"SafetyRule": rcCloneMap(safetyRule)}

	case "GetResourcePolicy":
		resourceARN := rcFirstNonEmpty(rcPathString(pathParams, "ResourceArn", ""), rcPayloadString(payload, "ResourceArn", ""), recoveryClusterDefaultClusterARN)
		if strings.TrimSpace(s.resourcePolicies[resourceARN]) == "" {
			s.resourcePolicies[resourceARN] = recoveryClusterDefaultPolicy
		}
		return map[string]any{"ResourcePolicy": s.resourcePolicies[resourceARN]}

	case "ListAssociatedRoute53HealthChecks":
		routingControlARN := rcFirstNonEmpty(rcPathString(pathParams, "RoutingControlArn", ""), rcPayloadString(payload, "RoutingControlArn", ""), recoveryClusterDefaultRoutingCtrlARN)
		healthChecks := s.associatedHealthChecks[routingControlARN]
		items := make([]any, 0, len(healthChecks))
		for _, item := range healthChecks {
			items = append(items, rcCloneMap(item))
		}
		return map[string]any{"Route53HealthChecks": items, "NextToken": ""}

	case "ListClusters":
		items := make([]any, 0, len(s.clusters))
		for _, cluster := range s.listClustersLocked() {
			items = append(items, rcCloneMap(cluster))
		}
		return map[string]any{"Clusters": items, "NextToken": ""}

	case "ListControlPanels":
		items := make([]any, 0, len(s.controlPanels))
		for _, controlPanel := range s.listControlPanelsLocked() {
			items = append(items, rcCloneMap(controlPanel))
		}
		return map[string]any{"ControlPanels": items, "NextToken": ""}

	case "ListRoutingControls":
		controlPanelARN := rcFirstNonEmpty(rcPathString(pathParams, "ControlPanelArn", ""), rcPayloadString(payload, "ControlPanelArn", ""), recoveryClusterDefaultControlPanelARN)
		items := make([]any, 0, len(s.routingControls))
		for _, routingControl := range s.listRoutingControlsLocked() {
			if rcAnyString(routingControl["ControlPanelArn"]) != controlPanelARN {
				continue
			}
			items = append(items, rcCloneMap(routingControl))
		}
		return map[string]any{"RoutingControls": items, "NextToken": ""}

	case "ListSafetyRules":
		controlPanelARN := rcFirstNonEmpty(rcPathString(pathParams, "ControlPanelArn", ""), rcPayloadString(payload, "ControlPanelArn", ""), recoveryClusterDefaultControlPanelARN)
		items := make([]any, 0, len(s.safetyRules))
		for _, safetyRule := range s.listSafetyRulesLocked() {
			if rcAnyString(safetyRule["ControlPanelArn"]) != controlPanelARN {
				continue
			}
			items = append(items, rcCloneMap(safetyRule))
		}
		return map[string]any{"SafetyRules": items, "NextToken": ""}

	case "ListTagsForResource":
		resourceARN := rcFirstNonEmpty(rcPathString(pathParams, "ResourceArn", ""), rcPayloadString(payload, "ResourceArn", ""), recoveryClusterDefaultClusterARN)
		return map[string]any{"Tags": rcCloneStringMap(s.ensureTagMapLocked(resourceARN))}

	case "TagResource":
		resourceARN := rcFirstNonEmpty(rcPathString(pathParams, "ResourceArn", ""), rcPayloadString(payload, "ResourceArn", ""), recoveryClusterDefaultClusterARN)
		s.mergeTagsLocked(resourceARN, payload)
		return map[string]any{}

	case "UntagResource":
		resourceARN := rcFirstNonEmpty(rcPathString(pathParams, "ResourceArn", ""), rcPayloadString(payload, "ResourceArn", ""), recoveryClusterDefaultClusterARN)
		tagKeys := rcStringSlice(rcLookupAny(payload, "TagKeys"))
		if len(tagKeys) == 0 {
			tagKeys = rcStringSlice(rcLookupAny(payload, "tagKeys"))
		}
		tags := s.ensureTagMapLocked(resourceARN)
		for _, key := range tagKeys {
			delete(tags, key)
		}
		return map[string]any{}

	case "UpdateCluster":
		clusterARN := rcFirstNonEmpty(rcPayloadString(payload, "ClusterArn", ""), recoveryClusterDefaultClusterARN)
		cluster := s.ensureClusterLocked(clusterARN, now)
		if name := rcPayloadString(payload, "ClusterName", ""); name != "" {
			cluster["ClusterName"] = name
		}
		if networkType := rcPayloadString(payload, "NetworkType", ""); networkType != "" {
			cluster["NetworkType"] = networkType
		}
		cluster["UpdatedAt"] = now
		s.mergeTagsLocked(clusterARN, payload)
		return map[string]any{"Cluster": rcCloneMap(cluster)}

	case "UpdateControlPanel":
		controlPanelARN := rcFirstNonEmpty(rcPayloadString(payload, "ControlPanelArn", ""), recoveryClusterDefaultControlPanelARN)
		clusterARN := rcFirstNonEmpty(rcPayloadString(payload, "ClusterArn", ""), recoveryClusterDefaultClusterARN)
		controlPanel := s.ensureControlPanelLocked(controlPanelARN, clusterARN, now)
		if name := rcPayloadString(payload, "ControlPanelName", ""); name != "" {
			controlPanel["ControlPanelName"] = name
		}
		if owner := rcPayloadString(payload, "Owner", ""); owner != "" {
			controlPanel["Owner"] = owner
		}
		controlPanel["UpdatedAt"] = now
		s.mergeTagsLocked(controlPanelARN, payload)
		return map[string]any{"ControlPanel": rcCloneMap(controlPanel)}

	case "UpdateRoutingControl":
		routingControlARN := rcFirstNonEmpty(rcPayloadString(payload, "RoutingControlArn", ""), recoveryClusterDefaultRoutingCtrlARN)
		clusterARN := rcFirstNonEmpty(rcPayloadString(payload, "ClusterArn", ""), recoveryClusterDefaultClusterARN)
		controlPanelARN := rcFirstNonEmpty(rcPayloadString(payload, "ControlPanelArn", ""), recoveryClusterDefaultControlPanelARN)
		routingControl := s.ensureRoutingControlLocked(routingControlARN, controlPanelARN, clusterARN, now)
		if name := rcPayloadString(payload, "RoutingControlName", ""); name != "" {
			routingControl["RoutingControlName"] = name
		}
		if status := rcPayloadString(payload, "Status", ""); status != "" {
			routingControl["Status"] = status
		}
		routingControl["UpdatedAt"] = now
		s.mergeTagsLocked(routingControlARN, payload)
		return map[string]any{"RoutingControl": rcCloneMap(routingControl)}

	case "UpdateSafetyRule":
		safetyRuleARN := rcFirstNonEmpty(rcPayloadString(payload, "SafetyRuleArn", ""), recoveryClusterDefaultSafetyRuleARN)
		safetyRule := s.ensureSafetyRuleLocked(safetyRuleARN, recoveryClusterDefaultControlPanelARN, now)
		if assertionUpdate, ok := rcLookupCI(payload, "AssertionRuleUpdate"); ok {
			safetyRule["AssertionRuleUpdate"] = rcCloneAny(assertionUpdate)
			safetyRule["RuleType"] = "ASSERTION"
		}
		if gatingUpdate, ok := rcLookupCI(payload, "GatingRuleUpdate"); ok {
			safetyRule["GatingRuleUpdate"] = rcCloneAny(gatingUpdate)
			safetyRule["RuleType"] = "GATING"
		}
		safetyRule["UpdatedAt"] = now
		s.mergeTagsLocked(safetyRuleARN, payload)
		return map[string]any{"SafetyRule": rcCloneMap(safetyRule)}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"Items": []any{}, "NextToken": ""}
	}
	if strings.HasPrefix(action, "Describe") || strings.HasPrefix(action, "Get") {
		return map[string]any{"Status": "DEPLOYED"}
	}
	if strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Update") {
		return map[string]any{"Status": "PENDING"}
	}
	if strings.HasPrefix(action, "Delete") || strings.HasPrefix(action, "Tag") || strings.HasPrefix(action, "Untag") {
		return map[string]any{}
	}
	return map[string]any{}
}

func (s *recoveryClusterStore) seedDefaultsLocked(now string) {
	cluster := s.ensureClusterLocked(recoveryClusterDefaultClusterARN, now)
	controlPanel := s.ensureControlPanelLocked(recoveryClusterDefaultControlPanelARN, recoveryClusterDefaultClusterARN, now)
	routingControl := s.ensureRoutingControlLocked(recoveryClusterDefaultRoutingCtrlARN, recoveryClusterDefaultControlPanelARN, recoveryClusterDefaultClusterARN, now)
	safetyRule := s.ensureSafetyRuleLocked(recoveryClusterDefaultSafetyRuleARN, recoveryClusterDefaultControlPanelARN, now)
	if strings.TrimSpace(s.resourcePolicies[recoveryClusterDefaultClusterARN]) == "" {
		s.resourcePolicies[recoveryClusterDefaultClusterARN] = recoveryClusterDefaultPolicy
	}
	s.ensureTagMapLocked(recoveryClusterDefaultClusterARN)["seed"] = "true"
	s.ensureTagMapLocked(recoveryClusterDefaultControlPanelARN)["seed"] = "true"
	s.ensureTagMapLocked(recoveryClusterDefaultRoutingCtrlARN)["seed"] = "true"
	s.ensureTagMapLocked(recoveryClusterDefaultSafetyRuleARN)["seed"] = "true"

	if rcAnyString(controlPanel["ClusterArn"]) == "" {
		controlPanel["ClusterArn"] = rcAnyString(cluster["ClusterArn"])
	}
	if rcAnyString(routingControl["ControlPanelArn"]) == "" {
		routingControl["ControlPanelArn"] = rcAnyString(controlPanel["ControlPanelArn"])
	}
	if rcAnyString(routingControl["ClusterArn"]) == "" {
		routingControl["ClusterArn"] = rcAnyString(cluster["ClusterArn"])
	}
	if rcAnyString(safetyRule["ControlPanelArn"]) == "" {
		safetyRule["ControlPanelArn"] = rcAnyString(controlPanel["ControlPanelArn"])
	}
}

func (s *recoveryClusterStore) ensureClusterLocked(clusterARN, now string) map[string]any {
	clusterARN = rcNormalizeID(clusterARN, recoveryClusterDefaultClusterARN)
	if cluster := s.clusters[clusterARN]; cluster != nil {
		return cluster
	}
	cluster := map[string]any{
		"ClusterArn":  clusterARN,
		"ClusterName": "stackyard-cluster",
		"Status":      "DEPLOYED",
		"NetworkType": "IPV4",
		"ClusterEndpoints": []any{
			map[string]any{"Endpoint": "https://cluster.stackyard.local", "Region": "us-east-1"},
		},
		"Owner":     "123456789012",
		"CreatedAt": now,
	}
	s.clusters[clusterARN] = cluster
	return cluster
}

func (s *recoveryClusterStore) ensureControlPanelLocked(controlPanelARN, clusterARN, now string) map[string]any {
	controlPanelARN = rcNormalizeID(controlPanelARN, recoveryClusterDefaultControlPanelARN)
	if controlPanel := s.controlPanels[controlPanelARN]; controlPanel != nil {
		return controlPanel
	}
	controlPanel := map[string]any{
		"ControlPanelArn":  controlPanelARN,
		"ControlPanelName": "stackyard-control-panel",
		"ClusterArn":       rcNormalizeID(clusterARN, recoveryClusterDefaultClusterARN),
		"Status":           "DEPLOYED",
		"Owner":            "123456789012",
		"CreatedAt":        now,
	}
	s.controlPanels[controlPanelARN] = controlPanel
	return controlPanel
}

func (s *recoveryClusterStore) ensureRoutingControlLocked(routingControlARN, controlPanelARN, clusterARN, now string) map[string]any {
	routingControlARN = rcNormalizeID(routingControlARN, recoveryClusterDefaultRoutingCtrlARN)
	if routingControl := s.routingControls[routingControlARN]; routingControl != nil {
		return routingControl
	}
	routingControl := map[string]any{
		"RoutingControlArn":  routingControlARN,
		"RoutingControlName": "stackyard-routing-control",
		"ControlPanelArn":    rcNormalizeID(controlPanelARN, recoveryClusterDefaultControlPanelARN),
		"ClusterArn":         rcNormalizeID(clusterARN, recoveryClusterDefaultClusterARN),
		"Status":             "DEPLOYED",
		"Owner":              "123456789012",
		"CreatedAt":          now,
	}
	s.routingControls[routingControlARN] = routingControl
	if _, ok := s.associatedHealthChecks[routingControlARN]; !ok {
		s.associatedHealthChecks[routingControlARN] = []map[string]any{}
	}
	return routingControl
}

func (s *recoveryClusterStore) ensureSafetyRuleLocked(safetyRuleARN, controlPanelARN, now string) map[string]any {
	safetyRuleARN = rcNormalizeID(safetyRuleARN, recoveryClusterDefaultSafetyRuleARN)
	if safetyRule := s.safetyRules[safetyRuleARN]; safetyRule != nil {
		return safetyRule
	}
	safetyRule := map[string]any{
		"SafetyRuleArn":   safetyRuleARN,
		"ControlPanelArn": rcNormalizeID(controlPanelARN, recoveryClusterDefaultControlPanelARN),
		"Name":            "stackyard-safety-rule",
		"RuleType":        "ASSERTION",
		"Status":          "DEPLOYED",
		"CreatedAt":       now,
	}
	s.safetyRules[safetyRuleARN] = safetyRule
	return safetyRule
}

func (s *recoveryClusterStore) listClustersLocked() []map[string]any {
	keys := make([]string, 0, len(s.clusters))
	for key := range s.clusters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.clusters[key])
	}
	return out
}

func (s *recoveryClusterStore) listControlPanelsLocked() []map[string]any {
	keys := make([]string, 0, len(s.controlPanels))
	for key := range s.controlPanels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.controlPanels[key])
	}
	return out
}

func (s *recoveryClusterStore) listRoutingControlsLocked() []map[string]any {
	keys := make([]string, 0, len(s.routingControls))
	for key := range s.routingControls {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.routingControls[key])
	}
	return out
}

func (s *recoveryClusterStore) listSafetyRulesLocked() []map[string]any {
	keys := make([]string, 0, len(s.safetyRules))
	for key := range s.safetyRules {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.safetyRules[key])
	}
	return out
}

func (s *recoveryClusterStore) ensureTagMapLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = recoveryClusterDefaultClusterARN
	}
	if s.tags[resourceARN] == nil {
		s.tags[resourceARN] = map[string]string{}
	}
	return s.tags[resourceARN]
}

func (s *recoveryClusterStore) mergeTagsLocked(resourceARN string, payload map[string]any) {
	tags := s.ensureTagMapLocked(resourceARN)
	for key, value := range rcStringMap(rcLookupAny(payload, "Tags")) {
		tags[key] = value
	}
	for key, value := range rcStringMap(rcLookupAny(payload, "tags")) {
		tags[key] = value
	}
}

func (s *recoveryClusterStore) syncPayloadWithQuery(payload map[string]any, query url.Values) {
	if payload == nil {
		return
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if len(values) == 1 {
			payload[key] = values[0]
			continue
		}
		items := make([]any, 0, len(values))
		for _, value := range values {
			items = append(items, value)
		}
		payload[key] = items
	}
}

func rcNormalizeID(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func rcPathString(pathParams map[string]string, key, fallback string) string {
	for currentKey, value := range pathParams {
		if !strings.EqualFold(currentKey, key) {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return fallback
}

func rcPayloadString(payload map[string]any, key, fallback string) string {
	if value := rcLookupAny(payload, key); value != nil {
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return fallback
}

func rcLookupAny(payload map[string]any, key string) any {
	value, _ := rcLookupCI(payload, key)
	return value
}

func rcLookupCI(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	for currentKey, value := range payload {
		if strings.EqualFold(currentKey, key) {
			return value, true
		}
	}
	return nil, false
}

func rcFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func rcStringMap(value any) map[string]string {
	out := map[string]string{}
	if value == nil {
		return out
	}
	if typed, ok := value.(map[string]string); ok {
		for key, val := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(val)
		}
		return out
	}
	if typed, ok := value.(map[string]any); ok {
		for key, val := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprintf("%v", val))
		}
	}
	return out
}

func rcStringSlice(value any) []string {
	if value == nil {
		return nil
	}
	if typed, ok := value.([]string); ok {
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	}
	if typed, ok := value.([]any); ok {
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text != "" && text != "<nil>" {
				out = append(out, text)
			}
		}
		return out
	}
	text := strings.TrimSpace(fmt.Sprintf("%v", value))
	if text == "" || text == "<nil>" {
		return nil
	}
	return []string{text}
}

func rcAnyString(value any) string {
	text := strings.TrimSpace(fmt.Sprintf("%v", value))
	if text == "<nil>" {
		return ""
	}
	return text
}

func rcCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = rcCloneAny(value)
	}
	return out
}

func rcCloneSlice(in []any) []any {
	out := make([]any, 0, len(in))
	for _, value := range in {
		out = append(out, rcCloneAny(value))
	}
	return out
}

func rcCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return rcCloneMap(typed)
	case []any:
		return rcCloneSlice(typed)
	case map[string]string:
		return rcCloneStringMap(typed)
	default:
		return value
	}
}

func rcCloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}
