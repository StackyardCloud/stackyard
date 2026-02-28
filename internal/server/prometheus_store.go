package server

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type prometheusStore struct {
	mu sync.Mutex

	nextWorkspace int64
	nextScraper   int64
	nextAnomaly   int64

	workspaces        map[string]map[string]any
	workspaceConfig   map[string]map[string]any
	alertDefinitions  map[string]map[string]any
	ruleNamespaces    map[string]map[string]map[string]any
	scrapers          map[string]map[string]any
	loggingConfigs    map[string]map[string]any
	queryLogging      map[string]map[string]any
	resourcePolicies  map[string]map[string]any
	anomalyDetectors  map[string]map[string]map[string]any
	scraperLoggingCfg map[string]map[string]any
	tags              map[string]map[string]string
}

func newPrometheusStore() *prometheusStore {
	s := &prometheusStore{
		nextWorkspace:     2,
		nextScraper:       2,
		nextAnomaly:       2,
		workspaces:        map[string]map[string]any{},
		workspaceConfig:   map[string]map[string]any{},
		alertDefinitions:  map[string]map[string]any{},
		ruleNamespaces:    map[string]map[string]map[string]any{},
		scrapers:          map[string]map[string]any{},
		loggingConfigs:    map[string]map[string]any{},
		queryLogging:      map[string]map[string]any{},
		resourcePolicies:  map[string]map[string]any{},
		anomalyDetectors:  map[string]map[string]map[string]any{},
		scraperLoggingCfg: map[string]map[string]any{},
		tags:              map[string]map[string]string{},
	}

	ws := s.ensureWorkspaceLocked("ws-00000000-0000-0000-0000-000000000000")
	wsID := prometheusDefaultStringAny(ws, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
	s.ensureRuleNamespaceLocked(wsID, "stackyard-rules")
	s.ensureAlertDefinitionLocked(wsID)
	s.ensureAnomalyDetectorLocked(wsID, "ad-000001")
	scraper := s.ensureScraperLocked("scr-000001")
	scraperID := prometheusDefaultStringAny(scraper, "scraperId", "scr-000001")
	s.scraperLoggingCfg[scraperID] = map[string]any{
		"status": map[string]any{"statusCode": "ACTIVE"},
	}

	s.tags[prometheusWorkspaceARN(wsID)] = map[string]string{"stackyard": "true"}
	s.tags[prometheusScraperARN(scraperID)] = map[string]string{"stackyard": "true"}

	return s
}

func (s *prometheusStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateWorkspace":
		id := fmt.Sprintf("ws-%012d-0000-0000-0000-%012d", s.nextWorkspace, s.nextWorkspace)
		s.nextWorkspace++
		ws := s.ensureWorkspaceLocked(id)
		for k, v := range payload {
			ws[k] = v
		}
		return map[string]any{
			"arn":         prometheusWorkspaceARN(id),
			"workspaceId": id,
			"status":      map[string]any{"statusCode": "ACTIVE"},
		}

	case "DescribeWorkspace":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		return map[string]any{"workspace": prometheusCloneAnyMap(s.ensureWorkspaceLocked(wsID))}

	case "ListWorkspaces":
		keys := prometheusSortedKeys(s.workspaces)
		items := make([]any, 0, len(keys))
		for _, k := range keys {
			ws := s.ensureWorkspaceLocked(k)
			items = append(items, map[string]any{
				"alias":       ws["alias"],
				"arn":         ws["arn"],
				"createdAt":   ws["createdAt"],
				"status":      ws["status"],
				"workspaceId": ws["workspaceId"],
			})
		}
		return map[string]any{"workspaces": items, "nextToken": ""}

	case "DeleteWorkspace":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		delete(s.workspaces, wsID)
		delete(s.workspaceConfig, wsID)
		delete(s.alertDefinitions, wsID)
		delete(s.ruleNamespaces, wsID)
		delete(s.loggingConfigs, wsID)
		delete(s.queryLogging, wsID)
		delete(s.resourcePolicies, wsID)
		delete(s.anomalyDetectors, wsID)
		return map[string]any{"status": map[string]any{"statusCode": "DELETING"}}

	case "UpdateWorkspaceAlias":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		ws := s.ensureWorkspaceLocked(wsID)
		ws["alias"] = prometheusDefaultStringAny(payload, "alias", "stackyard")
		return map[string]any{}

	case "DescribeWorkspaceConfiguration":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		cfg := s.ensureWorkspaceConfigLocked(wsID)
		return map[string]any{"workspaceConfiguration": prometheusCloneAnyMap(cfg)}

	case "UpdateWorkspaceConfiguration":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		cfg := s.ensureWorkspaceConfigLocked(wsID)
		for k, v := range payload {
			cfg[k] = v
		}
		return map[string]any{"workspaceConfiguration": prometheusCloneAnyMap(cfg)}

	case "CreateRuleGroupsNamespace", "PutRuleGroupsNamespace":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		name := prometheusDefaultString(pathParams, "name", prometheusDefaultStringAny(payload, "name", "stackyard-rules"))
		ns := s.ensureRuleNamespaceLocked(wsID, name)
		for k, v := range payload {
			ns[k] = v
		}
		return map[string]any{"arn": prometheusRuleNamespaceARN(wsID, name), "name": name, "status": map[string]any{"statusCode": "ACTIVE"}}

	case "DescribeRuleGroupsNamespace":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		name := prometheusDefaultString(pathParams, "name", "stackyard-rules")
		ns := s.ensureRuleNamespaceLocked(wsID, name)
		return map[string]any{"ruleGroupsNamespace": prometheusCloneAnyMap(ns)}

	case "ListRuleGroupsNamespaces":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		if s.ruleNamespaces[wsID] == nil {
			s.ruleNamespaces[wsID] = map[string]map[string]any{}
		}
		keys := prometheusSortedKeys(s.ruleNamespaces[wsID])
		items := make([]any, 0, len(keys))
		for _, k := range keys {
			ns := s.ensureRuleNamespaceLocked(wsID, k)
			items = append(items, map[string]any{
				"arn":    ns["arn"],
				"name":   ns["name"],
				"status": ns["status"],
			})
		}
		return map[string]any{"ruleGroupsNamespaces": items, "nextToken": ""}

	case "DeleteRuleGroupsNamespace":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		name := prometheusDefaultString(pathParams, "name", "stackyard-rules")
		if s.ruleNamespaces[wsID] != nil {
			delete(s.ruleNamespaces[wsID], name)
		}
		return map[string]any{}

	case "CreateAlertManagerDefinition", "PutAlertManagerDefinition":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		ad := s.ensureAlertDefinitionLocked(wsID)
		for k, v := range payload {
			ad[k] = v
		}
		return map[string]any{"status": ad["status"]}

	case "DescribeAlertManagerDefinition":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		return map[string]any{"alertManagerDefinition": prometheusCloneAnyMap(s.ensureAlertDefinitionLocked(wsID))}

	case "DeleteAlertManagerDefinition":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		delete(s.alertDefinitions, wsID)
		return map[string]any{}

	case "CreateScraper":
		scraperID := fmt.Sprintf("scr-%06d", s.nextScraper)
		s.nextScraper++
		scr := s.ensureScraperLocked(scraperID)
		for k, v := range payload {
			scr[k] = v
		}
		return map[string]any{"arn": prometheusScraperARN(scraperID), "scraperId": scraperID, "status": scr["status"]}

	case "UpdateScraper":
		scraperID := prometheusDefaultString(pathParams, "scraperId", "scr-000001")
		scr := s.ensureScraperLocked(scraperID)
		for k, v := range payload {
			scr[k] = v
		}
		return map[string]any{"arn": prometheusScraperARN(scraperID), "scraperId": scraperID, "status": scr["status"]}

	case "DescribeScraper":
		scraperID := prometheusDefaultString(pathParams, "scraperId", "scr-000001")
		return map[string]any{"scraper": prometheusCloneAnyMap(s.ensureScraperLocked(scraperID))}

	case "ListScrapers":
		keys := prometheusSortedKeys(s.scrapers)
		items := make([]any, 0, len(keys))
		for _, k := range keys {
			scr := s.ensureScraperLocked(k)
			items = append(items, map[string]any{
				"arn":       scr["arn"],
				"createdAt": scr["createdAt"],
				"scraperId": scr["scraperId"],
				"status":    scr["status"],
			})
		}
		return map[string]any{"scrapers": items, "nextToken": ""}

	case "DeleteScraper":
		scraperID := prometheusDefaultString(pathParams, "scraperId", "scr-000001")
		delete(s.scrapers, scraperID)
		delete(s.scraperLoggingCfg, scraperID)
		return map[string]any{}

	case "GetDefaultScraperConfiguration":
		return map[string]any{"configuration": "{}"}

	case "UpdateScraperLoggingConfiguration":
		scraperID := prometheusDefaultString(pathParams, "scraperId", "scr-000001")
		cfg := map[string]any{"status": map[string]any{"statusCode": "ACTIVE"}}
		for k, v := range payload {
			cfg[k] = v
		}
		s.scraperLoggingCfg[scraperID] = cfg
		return map[string]any{"status": cfg["status"]}

	case "DescribeScraperLoggingConfiguration":
		scraperID := prometheusDefaultString(pathParams, "scraperId", "scr-000001")
		cfg := s.scraperLoggingCfg[scraperID]
		if cfg == nil {
			cfg = map[string]any{"status": map[string]any{"statusCode": "ACTIVE"}}
			s.scraperLoggingCfg[scraperID] = cfg
		}
		return map[string]any{"scraperLoggingConfiguration": prometheusCloneAnyMap(cfg)}

	case "DeleteScraperLoggingConfiguration":
		scraperID := prometheusDefaultString(pathParams, "scraperId", "scr-000001")
		delete(s.scraperLoggingCfg, scraperID)
		return map[string]any{}

	case "CreateLoggingConfiguration", "UpdateLoggingConfiguration":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		cfg := map[string]any{"status": map[string]any{"statusCode": "ACTIVE"}}
		for k, v := range payload {
			cfg[k] = v
		}
		s.loggingConfigs[wsID] = cfg
		return map[string]any{"status": cfg["status"]}

	case "DescribeLoggingConfiguration":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		cfg := s.loggingConfigs[wsID]
		if cfg == nil {
			cfg = map[string]any{"status": map[string]any{"statusCode": "ACTIVE"}}
			s.loggingConfigs[wsID] = cfg
		}
		return map[string]any{"loggingConfiguration": prometheusCloneAnyMap(cfg)}

	case "DeleteLoggingConfiguration":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		delete(s.loggingConfigs, wsID)
		return map[string]any{}

	case "CreateQueryLoggingConfiguration", "UpdateQueryLoggingConfiguration":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		cfg := map[string]any{"status": map[string]any{"statusCode": "ACTIVE"}}
		for k, v := range payload {
			cfg[k] = v
		}
		s.queryLogging[wsID] = cfg
		return map[string]any{"status": cfg["status"]}

	case "DescribeQueryLoggingConfiguration":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		cfg := s.queryLogging[wsID]
		if cfg == nil {
			cfg = map[string]any{"status": map[string]any{"statusCode": "ACTIVE"}}
			s.queryLogging[wsID] = cfg
		}
		return map[string]any{"queryLoggingConfiguration": prometheusCloneAnyMap(cfg)}

	case "DeleteQueryLoggingConfiguration":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		delete(s.queryLogging, wsID)
		return map[string]any{}

	case "PutResourcePolicy":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		policy := map[string]any{
			"policy":       prometheusDefaultStringAny(payload, "policy", "{}"),
			"policyStatus": map[string]any{"statusCode": "ACTIVE"},
		}
		s.resourcePolicies[wsID] = policy
		return map[string]any{"policyStatus": policy["policyStatus"], "revisionId": "1"}

	case "DescribeResourcePolicy":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		policy := s.resourcePolicies[wsID]
		if policy == nil {
			policy = map[string]any{
				"policy":       "{}",
				"policyStatus": map[string]any{"statusCode": "ACTIVE"},
			}
			s.resourcePolicies[wsID] = policy
		}
		return map[string]any{"policy": policy["policy"], "policyStatus": policy["policyStatus"], "revisionId": "1"}

	case "DeleteResourcePolicy":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		delete(s.resourcePolicies, wsID)
		return map[string]any{}

	case "CreateAnomalyDetector":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		id := fmt.Sprintf("ad-%06d", s.nextAnomaly)
		s.nextAnomaly++
		d := s.ensureAnomalyDetectorLocked(wsID, id)
		for k, v := range payload {
			d[k] = v
		}
		return map[string]any{"arn": prometheusAnomalyARN(wsID, id), "anomalyDetectorId": id, "status": d["status"]}

	case "PutAnomalyDetector":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		id := prometheusDefaultString(pathParams, "anomalyDetectorId", "ad-000001")
		d := s.ensureAnomalyDetectorLocked(wsID, id)
		for k, v := range payload {
			d[k] = v
		}
		return map[string]any{"arn": prometheusAnomalyARN(wsID, id), "anomalyDetectorId": id, "status": d["status"]}

	case "DescribeAnomalyDetector":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		id := prometheusDefaultString(pathParams, "anomalyDetectorId", "ad-000001")
		return map[string]any{"anomalyDetector": prometheusCloneAnyMap(s.ensureAnomalyDetectorLocked(wsID, id))}

	case "ListAnomalyDetectors":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		if s.anomalyDetectors[wsID] == nil {
			s.anomalyDetectors[wsID] = map[string]map[string]any{}
		}
		keys := prometheusSortedKeys(s.anomalyDetectors[wsID])
		items := make([]any, 0, len(keys))
		for _, k := range keys {
			d := s.ensureAnomalyDetectorLocked(wsID, k)
			items = append(items, map[string]any{
				"anomalyDetectorId": d["anomalyDetectorId"],
				"arn":               d["arn"],
				"status":            d["status"],
			})
		}
		return map[string]any{"anomalyDetectors": items, "nextToken": ""}

	case "DeleteAnomalyDetector":
		wsID := prometheusDefaultString(pathParams, "workspaceId", "ws-00000000-0000-0000-0000-000000000000")
		id := prometheusDefaultString(pathParams, "anomalyDetectorId", "ad-000001")
		if s.anomalyDetectors[wsID] != nil {
			delete(s.anomalyDetectors[wsID], id)
		}
		return map[string]any{}

	case "ListTagsForResource":
		arn := prometheusDefaultString(pathParams, "resourceArn", prometheusWorkspaceARN("ws-00000000-0000-0000-0000-000000000000"))
		return map[string]any{"tags": prometheusCloneStringMap(s.tags[arn])}

	case "TagResource":
		arn := prometheusDefaultString(pathParams, "resourceArn", prometheusWorkspaceARN("ws-00000000-0000-0000-0000-000000000000"))
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{}
		}
		switch tags := payload["tags"].(type) {
		case map[string]any:
			for k, v := range tags {
				s.tags[arn][k] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		case map[string]string:
			for k, v := range tags {
				s.tags[arn][k] = v
			}
		}
		return map[string]any{}

	case "UntagResource":
		arn := prometheusDefaultString(pathParams, "resourceArn", prometheusWorkspaceARN("ws-00000000-0000-0000-0000-000000000000"))
		if keys, ok := payload["tagKeys"].([]any); ok {
			for _, key := range keys {
				delete(s.tags[arn], strings.TrimSpace(fmt.Sprintf("%v", key)))
			}
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *prometheusStore) ensureWorkspaceLocked(workspaceID string) map[string]any {
	id := strings.TrimSpace(workspaceID)
	if id == "" {
		id = "ws-00000000-0000-0000-0000-000000000000"
	}
	if existing := s.workspaces[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"alias":       "stackyard",
		"arn":         prometheusWorkspaceARN(id),
		"createdAt":   time.Now().UTC().Format(time.RFC3339),
		"status":      map[string]any{"statusCode": "ACTIVE"},
		"workspaceId": id,
	}
	s.workspaces[id] = item
	s.ensureWorkspaceConfigLocked(id)
	return item
}

func (s *prometheusStore) ensureWorkspaceConfigLocked(workspaceID string) map[string]any {
	id := strings.TrimSpace(workspaceID)
	if id == "" {
		id = "ws-00000000-0000-0000-0000-000000000000"
	}
	_ = s.ensureWorkspaceLocked(id)
	if cfg := s.workspaceConfig[id]; cfg != nil {
		return cfg
	}
	cfg := map[string]any{
		"limitsPerLabelSet": []any{},
		"status":            map[string]any{"statusCode": "ACTIVE"},
	}
	s.workspaceConfig[id] = cfg
	return cfg
}

func (s *prometheusStore) ensureRuleNamespaceLocked(workspaceID, name string) map[string]any {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		workspaceID = "ws-00000000-0000-0000-0000-000000000000"
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-rules"
	}
	_ = s.ensureWorkspaceLocked(workspaceID)
	if s.ruleNamespaces[workspaceID] == nil {
		s.ruleNamespaces[workspaceID] = map[string]map[string]any{}
	}
	if ns := s.ruleNamespaces[workspaceID][name]; ns != nil {
		return ns
	}
	ns := map[string]any{
		"arn":    prometheusRuleNamespaceARN(workspaceID, name),
		"name":   name,
		"status": map[string]any{"statusCode": "ACTIVE"},
		"data":   "{}",
	}
	s.ruleNamespaces[workspaceID][name] = ns
	return ns
}

func (s *prometheusStore) ensureAlertDefinitionLocked(workspaceID string) map[string]any {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		workspaceID = "ws-00000000-0000-0000-0000-000000000000"
	}
	_ = s.ensureWorkspaceLocked(workspaceID)
	if ad := s.alertDefinitions[workspaceID]; ad != nil {
		return ad
	}
	ad := map[string]any{
		"status": map[string]any{"statusCode": "ACTIVE"},
		"data":   "{}",
	}
	s.alertDefinitions[workspaceID] = ad
	return ad
}

func (s *prometheusStore) ensureScraperLocked(scraperID string) map[string]any {
	id := strings.TrimSpace(scraperID)
	if id == "" {
		id = "scr-000001"
	}
	if s.scrapers[id] != nil {
		return s.scrapers[id]
	}
	scr := map[string]any{
		"arn":       prometheusScraperARN(id),
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"scraperId": id,
		"status":    map[string]any{"statusCode": "ACTIVE"},
	}
	s.scrapers[id] = scr
	return scr
}

func (s *prometheusStore) ensureAnomalyDetectorLocked(workspaceID, anomalyID string) map[string]any {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		workspaceID = "ws-00000000-0000-0000-0000-000000000000"
	}
	anomalyID = strings.TrimSpace(anomalyID)
	if anomalyID == "" {
		anomalyID = "ad-000001"
	}
	if s.anomalyDetectors[workspaceID] == nil {
		s.anomalyDetectors[workspaceID] = map[string]map[string]any{}
	}
	if d := s.anomalyDetectors[workspaceID][anomalyID]; d != nil {
		return d
	}
	d := map[string]any{
		"anomalyDetectorId": anomalyID,
		"arn":               prometheusAnomalyARN(workspaceID, anomalyID),
		"status":            map[string]any{"statusCode": "ACTIVE"},
	}
	s.anomalyDetectors[workspaceID][anomalyID] = d
	return d
}

func prometheusWorkspaceARN(workspaceID string) string {
	id := strings.TrimSpace(workspaceID)
	if id == "" {
		id = "ws-00000000-0000-0000-0000-000000000000"
	}
	return fmt.Sprintf("arn:aws:aps:us-east-1:123456789012:workspace/%s", id)
}

func prometheusRuleNamespaceARN(workspaceID, name string) string {
	return fmt.Sprintf("%s/rulegroupsnamespace/%s", prometheusWorkspaceARN(workspaceID), strings.TrimSpace(name))
}

func prometheusScraperARN(scraperID string) string {
	id := strings.TrimSpace(scraperID)
	if id == "" {
		id = "scr-000001"
	}
	return fmt.Sprintf("arn:aws:aps:us-east-1:123456789012:scraper/%s", id)
}

func prometheusAnomalyARN(workspaceID, anomalyID string) string {
	return fmt.Sprintf("%s/anomalydetector/%s", prometheusWorkspaceARN(workspaceID), strings.TrimSpace(anomalyID))
}

func prometheusDefaultString(values map[string]string, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}

func prometheusDefaultStringAny(values map[string]any, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if trimmed := strings.TrimSpace(fmt.Sprintf("%v", v)); trimmed != "" && trimmed != "<nil>" {
				return trimmed
			}
		}
	}
	return fallback
}

func prometheusDefaultInt64Any(values map[string]any, key string, fallback int64) int64 {
	for k, v := range values {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		switch n := v.(type) {
		case int:
			return int64(n)
		case int32:
			return int64(n)
		case int64:
			return n
		case float64:
			return int64(n)
		case float32:
			return int64(n)
		default:
			asString := strings.TrimSpace(fmt.Sprintf("%v", v))
			if asString == "" {
				break
			}
			parsed, err := strconv.ParseInt(asString, 10, 64)
			if err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func prometheusSortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func prometheusCloneAnyMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func prometheusCloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
