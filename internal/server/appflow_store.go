package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type appFlowStore struct {
	mu sync.Mutex

	nextFlowID             int64
	nextExecutionID        int64
	nextConnectorProfileID int64

	flows             map[string]map[string]any
	flowExecutions    map[string][]map[string]any
	connectorProfiles map[string]map[string]any
	connectors        map[string]map[string]any
	tags              map[string]map[string]string
}

func newAppFlowStore() *appFlowStore {
	s := &appFlowStore{
		nextFlowID:             2,
		nextExecutionID:        2,
		nextConnectorProfileID: 2,
		flows:                  map[string]map[string]any{},
		flowExecutions:         map[string][]map[string]any{},
		connectorProfiles:      map[string]map[string]any{},
		connectors:             map[string]map[string]any{},
		tags:                   map[string]map[string]string{},
	}
	s.ensureSeedDataLocked()
	return s
}

func (s *appFlowStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSeedDataLocked()

	syncPayload := appFlowCloneMap(payload)
	for key, value := range pathParams {
		syncPayload[key] = value
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		syncPayload[key] = values[len(values)-1]
	}

	now := time.Now().UTC()
	defaultFlowName := s.defaultFlowNameLocked()
	if defaultFlowName == "" {
		defaultFlowName = "stackyard-seed-flow"
	}
	defaultFlow := s.ensureFlowLocked(defaultFlowName, now)

	switch action {
	case "CreateFlow":
		flowName := appFlowString(syncPayload, "flowName", fmt.Sprintf("stackyard-flow-%06d", s.nextFlowID))
		flow := s.ensureFlowLocked(flowName, now)
		if desc := appFlowString(syncPayload, "description", ""); desc != "" {
			flow["description"] = desc
		}
		flow["flowStatus"] = "Active"
		flow["lastUpdatedAt"] = now.Format(time.RFC3339)
		s.nextFlowID++
		s.upsertTagsLocked(appFlowString(flow, "flowArn", ""), appFlowMapString(syncPayload["tags"]))
		return map[string]any{
			"flowArn":    appFlowString(flow, "flowArn", ""),
			"flowStatus": appFlowString(flow, "flowStatus", "Active"),
		}

	case "UpdateFlow":
		flowName := appFlowString(syncPayload, "flowName", defaultFlowName)
		flow := s.ensureFlowLocked(flowName, now)
		if desc := appFlowString(syncPayload, "description", ""); desc != "" {
			flow["description"] = desc
		}
		flow["flowStatus"] = appFlowString(syncPayload, "flowStatus", appFlowString(flow, "flowStatus", "Active"))
		flow["lastUpdatedAt"] = now.Format(time.RFC3339)
		s.upsertTagsLocked(appFlowString(flow, "flowArn", ""), appFlowMapString(syncPayload["tags"]))
		return map[string]any{
			"flowArn":    appFlowString(flow, "flowArn", ""),
			"flowStatus": appFlowString(flow, "flowStatus", "Active"),
		}

	case "DeleteFlow":
		flowName := appFlowString(syncPayload, "flowName", defaultFlowName)
		flow := s.ensureFlowLocked(flowName, now)
		delete(s.flows, flowName)
		delete(s.flowExecutions, flowName)
		delete(s.tags, appFlowString(flow, "flowArn", ""))
		return map[string]any{}

	case "DescribeFlow":
		flowName := appFlowString(syncPayload, "flowName", defaultFlowName)
		flow := s.ensureFlowLocked(flowName, now)
		return appFlowCloneMap(flow)

	case "ListFlows":
		items := make([]any, 0, len(s.flows))
		names := make([]string, 0, len(s.flows))
		for name := range s.flows {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			flow := s.flows[name]
			items = append(items, map[string]any{
				"flowArn":                  appFlowString(flow, "flowArn", ""),
				"flowName":                 appFlowString(flow, "flowName", ""),
				"description":              appFlowString(flow, "description", ""),
				"flowStatus":               appFlowString(flow, "flowStatus", "Active"),
				"sourceConnectorType":      appFlowString(flow, "sourceConnectorType", "Salesforce"),
				"destinationConnectorType": appFlowString(flow, "destinationConnectorType", "S3"),
				"createdAt":                appFlowString(flow, "createdAt", ""),
				"lastUpdatedAt":            appFlowString(flow, "lastUpdatedAt", ""),
			})
		}
		return map[string]any{"flows": items, "nextToken": ""}

	case "StartFlow":
		flowName := appFlowString(syncPayload, "flowName", defaultFlowName)
		flow := s.ensureFlowLocked(flowName, now)
		flow["flowStatus"] = "Active"
		flow["lastUpdatedAt"] = now.Format(time.RFC3339)
		execID := fmt.Sprintf("exec-%06d", s.nextExecutionID)
		s.nextExecutionID++
		record := map[string]any{
			"executionId":       execID,
			"executionStatus":   "Successful",
			"startedAt":         now.Format(time.RFC3339),
			"lastUpdatedAt":     now.Format(time.RFC3339),
			"dataPullStartTime": now.Add(-1 * time.Minute).Format(time.RFC3339),
			"dataPullEndTime":   now.Format(time.RFC3339),
		}
		s.flowExecutions[flowName] = append(s.flowExecutions[flowName], record)
		return map[string]any{
			"executionId": execID,
			"flowArn":     appFlowString(flow, "flowArn", ""),
			"flowStatus":  appFlowString(flow, "flowStatus", "Active"),
		}

	case "StopFlow":
		flowName := appFlowString(syncPayload, "flowName", defaultFlowName)
		flow := s.ensureFlowLocked(flowName, now)
		flow["flowStatus"] = "Suspended"
		flow["lastUpdatedAt"] = now.Format(time.RFC3339)
		return map[string]any{
			"flowArn":    appFlowString(flow, "flowArn", ""),
			"flowStatus": appFlowString(flow, "flowStatus", "Suspended"),
		}

	case "CancelFlowExecutions":
		flowName := appFlowString(syncPayload, "flowName", defaultFlowName)
		records := s.flowExecutions[flowName]
		for _, record := range records {
			record["executionStatus"] = "Canceled"
			record["lastUpdatedAt"] = now.Format(time.RFC3339)
		}
		return map[string]any{}

	case "DescribeFlowExecutionRecords":
		flowName := appFlowString(syncPayload, "flowName", defaultFlowName)
		records := s.flowExecutions[flowName]
		out := make([]any, 0, len(records))
		for _, record := range records {
			out = append(out, appFlowCloneMap(record))
		}
		return map[string]any{
			"flowExecutions": out,
			"nextToken":      "",
		}

	case "CreateConnectorProfile", "UpdateConnectorProfile":
		name := appFlowString(syncPayload, "connectorProfileName", fmt.Sprintf("stackyard-connector-profile-%06d", s.nextConnectorProfileID))
		profile := s.ensureConnectorProfileLocked(name, now)
		if connectorType := appFlowString(syncPayload, "connectorType", ""); connectorType != "" {
			profile["connectorType"] = connectorType
		}
		profile["lastUpdatedAt"] = now.Format(time.RFC3339)
		s.nextConnectorProfileID++
		return map[string]any{"connectorProfileArn": appFlowString(profile, "connectorProfileArn", "")}

	case "DeleteConnectorProfile":
		name := appFlowString(syncPayload, "connectorProfileName", "stackyard-seed-profile")
		delete(s.connectorProfiles, name)
		return map[string]any{}

	case "DescribeConnectorProfiles":
		requested := appFlowStringSlice(syncPayload["connectorProfileNames"])
		filter := map[string]struct{}{}
		for _, name := range requested {
			filter[name] = struct{}{}
		}
		items := make([]any, 0, len(s.connectorProfiles))
		names := make([]string, 0, len(s.connectorProfiles))
		for name := range s.connectorProfiles {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if len(filter) > 0 {
				if _, ok := filter[name]; !ok {
					continue
				}
			}
			items = append(items, appFlowCloneMap(s.connectorProfiles[name]))
		}
		return map[string]any{"connectorProfileDetails": items, "nextToken": ""}

	case "DescribeConnector":
		connectorType := appFlowString(syncPayload, "connectorType", "Salesforce")
		return map[string]any{
			"connectorConfiguration": map[string]any{
				"canUseAsDestination":  true,
				"canUseAsSource":       true,
				"connectorType":        connectorType,
				"isPrivateLinkEnabled": false,
			},
		}

	case "DescribeConnectors":
		out := map[string]any{}
		keys := make([]string, 0, len(s.connectors))
		for key := range s.connectors {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			out[key] = appFlowCloneMap(s.connectors[key])
		}
		return map[string]any{"connectorConfigurations": out}

	case "ListConnectors":
		items := make([]any, 0, len(s.connectors))
		keys := make([]string, 0, len(s.connectors))
		for key := range s.connectors {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			connector := s.connectors[key]
			items = append(items, map[string]any{
				"connectorLabel": key,
				"connectorType":  appFlowString(connector, "connectorType", key),
			})
		}
		return map[string]any{"connectors": items, "nextToken": ""}

	case "DescribeConnectorEntity":
		entityName := appFlowString(syncPayload, "entityName", "Account")
		return map[string]any{
			"connectorEntityFields": []any{
				map[string]any{"identifier": "Id", "label": "Id", "isPrimaryKey": true, "type": "string"},
				map[string]any{"identifier": "Name", "label": "Name", "isPrimaryKey": false, "type": "string"},
			},
			"entities": map[string]any{entityName: map[string]any{}},
		}

	case "ListConnectorEntities":
		return map[string]any{
			"connectorEntityMap": map[string]any{
				"Account": map[string]any{},
				"Contact": map[string]any{},
			},
			"nextToken": "",
		}

	case "RegisterConnector", "UpdateConnectorRegistration":
		connectorLabel := appFlowString(syncPayload, "connectorLabel", "CustomConnector")
		connectorArn := appFlowConnectorARN(strings.ToLower(strings.ReplaceAll(connectorLabel, " ", "-")))
		s.connectors[connectorLabel] = map[string]any{
			"connectorLabel":       connectorLabel,
			"connectorType":        appFlowString(syncPayload, "connectorType", connectorLabel),
			"canUseAsDestination":  true,
			"canUseAsSource":       true,
			"isPrivateLinkEnabled": false,
			"connectorArn":         connectorArn,
		}
		return map[string]any{"connectorArn": connectorArn}

	case "UnregisterConnector":
		connectorLabel := appFlowString(syncPayload, "connectorLabel", "")
		if connectorLabel == "" {
			if connectorArn := appFlowString(syncPayload, "connectorArn", ""); connectorArn != "" {
				connectorLabel = appFlowLastToken(connectorArn)
			}
		}
		if connectorLabel != "" {
			delete(s.connectors, connectorLabel)
		}
		return map[string]any{}

	case "ResetConnectorMetadataCache":
		return map[string]any{}

	case "TagResource":
		resourceArn := appFlowResourceARN(syncPayload, pathParams, defaultFlow)
		s.upsertTagsLocked(resourceArn, appFlowMapString(syncPayload["tags"]))
		return map[string]any{}

	case "UntagResource":
		resourceArn := appFlowResourceARN(syncPayload, pathParams, defaultFlow)
		tagKeys := appFlowTagKeys(syncPayload, query)
		s.removeTagsLocked(resourceArn, tagKeys)
		return map[string]any{}

	case "ListTagsForResource":
		resourceArn := appFlowResourceARN(syncPayload, pathParams, defaultFlow)
		return map[string]any{"tags": appFlowCloneStringMap(s.tags[resourceArn])}
	}

	return map[string]any{}
}

func (s *appFlowStore) ensureSeedDataLocked() {
	if len(s.flows) == 0 {
		now := time.Now().UTC()
		s.ensureFlowLocked("stackyard-seed-flow", now)
	}
	if len(s.connectorProfiles) == 0 {
		now := time.Now().UTC()
		s.ensureConnectorProfileLocked("stackyard-seed-profile", now)
	}
	if len(s.connectors) == 0 {
		s.connectors["Salesforce"] = map[string]any{
			"canUseAsDestination": true,
			"canUseAsSource":      true,
			"connectorType":       "Salesforce",
		}
		s.connectors["S3"] = map[string]any{
			"canUseAsDestination": true,
			"canUseAsSource":      false,
			"connectorType":       "S3",
		}
	}
}

func (s *appFlowStore) ensureFlowLocked(flowName string, now time.Time) map[string]any {
	name := strings.TrimSpace(flowName)
	if name == "" {
		name = fmt.Sprintf("stackyard-flow-%06d", s.nextFlowID)
	}
	if flow := s.flows[name]; flow != nil {
		return flow
	}
	arn := appFlowFlowARN(name)
	flow := map[string]any{
		"flowName":                 name,
		"flowArn":                  arn,
		"description":              "Stackyard flow " + name,
		"flowStatus":               "Active",
		"sourceConnectorType":      "Salesforce",
		"destinationConnectorType": "S3",
		"createdAt":                now.Format(time.RFC3339),
		"lastUpdatedAt":            now.Format(time.RFC3339),
	}
	s.flows[name] = flow
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{"seed": "true"}
	}
	return flow
}

func (s *appFlowStore) ensureConnectorProfileLocked(name string, now time.Time) map[string]any {
	key := strings.TrimSpace(name)
	if key == "" {
		key = fmt.Sprintf("stackyard-connector-profile-%06d", s.nextConnectorProfileID)
	}
	if profile := s.connectorProfiles[key]; profile != nil {
		return profile
	}
	profile := map[string]any{
		"connectorProfileName": key,
		"connectorType":        "Salesforce",
		"connectorProfileArn":  appFlowConnectorProfileARN(key),
		"createdAt":            now.Format(time.RFC3339),
		"lastUpdatedAt":        now.Format(time.RFC3339),
	}
	s.connectorProfiles[key] = profile
	return profile
}

func (s *appFlowStore) defaultFlowNameLocked() string {
	if len(s.flows) == 0 {
		return ""
	}
	names := make([]string, 0, len(s.flows))
	for name := range s.flows {
		names = append(names, name)
	}
	sort.Strings(names)
	return names[0]
}

func (s *appFlowStore) upsertTagsLocked(resourceArn string, tags map[string]string) {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" || len(tags) == 0 {
		return
	}
	if s.tags[resourceArn] == nil {
		s.tags[resourceArn] = map[string]string{}
	}
	for key, value := range tags {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		s.tags[resourceArn][k] = strings.TrimSpace(value)
	}
}

func (s *appFlowStore) removeTagsLocked(resourceArn string, tagKeys []string) {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" || len(tagKeys) == 0 {
		return
	}
	tags := s.tags[resourceArn]
	if tags == nil {
		return
	}
	for _, key := range tagKeys {
		delete(tags, strings.TrimSpace(key))
	}
}

func appFlowResourceARN(payload map[string]any, pathParams map[string]string, defaultFlow map[string]any) string {
	if value := strings.TrimSpace(pathParams["resourceArn"]); value != "" {
		return value
	}
	if value := strings.TrimSpace(appFlowString(payload, "resourceArn", "")); value != "" {
		return value
	}
	if value := strings.TrimSpace(appFlowString(payload, "ResourceArn", "")); value != "" {
		return value
	}
	if defaultFlow != nil {
		return appFlowString(defaultFlow, "flowArn", "")
	}
	return ""
}

func appFlowTagKeys(payload map[string]any, query url.Values) []string {
	for _, key := range []string{"tagKeys", "TagKeys"} {
		if raw, ok := payload[key]; ok {
			if values := appFlowStringSlice(raw); len(values) > 0 {
				return values
			}
			if single := strings.TrimSpace(appFlowString(payload, key, "")); single != "" {
				return []string{single}
			}
		}
	}
	if values, ok := query["tagKeys"]; ok && len(values) > 0 {
		out := make([]string, 0, len(values))
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value != "" {
				out = append(out, value)
			}
		}
		return out
	}
	if value := strings.TrimSpace(query.Get("TagKeys")); value != "" {
		return []string{value}
	}
	return nil
}

func appFlowFlowARN(flowName string) string {
	return "arn:aws:appflow:us-east-1:123456789012:flow/" + strings.TrimSpace(flowName)
}

func appFlowConnectorProfileARN(name string) string {
	return "arn:aws:appflow:us-east-1:123456789012:connectorprofile/" + strings.TrimSpace(name)
}

func appFlowConnectorARN(name string) string {
	return "arn:aws:appflow:us-east-1:123456789012:connector/" + strings.TrimSpace(name)
}

func appFlowLastToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.LastIndex(value, "/"); idx >= 0 && idx+1 < len(value) {
		return strings.TrimSpace(value[idx+1:])
	}
	return value
}

func appFlowString(m map[string]any, key, def string) string {
	if m == nil {
		return def
	}
	raw, ok := m[key]
	if !ok || raw == nil {
		return def
	}
	if value, ok := raw.(string); ok {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
		return def
	}
	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" {
		return def
	}
	return value
}

func appFlowMapString(value any) map[string]string {
	out := map[string]string{}
	v, ok := value.(map[string]any)
	if !ok {
		if typed, ok := value.(map[string]string); ok {
			for k, v := range typed {
				k = strings.TrimSpace(k)
				if k == "" {
					continue
				}
				out[k] = strings.TrimSpace(v)
			}
		}
		return out
	}
	for k, v := range v {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(fmt.Sprint(v))
	}
	return out
}

func appFlowStringSlice(value any) []string {
	rawItems, ok := value.([]any)
	if !ok {
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
		return nil
	}
	out := make([]string, 0, len(rawItems))
	for _, item := range rawItems {
		itemStr := strings.TrimSpace(fmt.Sprint(item))
		if itemStr != "" {
			out = append(out, itemStr)
		}
	}
	return out
}

func appFlowCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = appFlowCloneMap(typed)
		case []any:
			items := make([]any, len(typed))
			for i, item := range typed {
				if itemMap, ok := item.(map[string]any); ok {
					items[i] = appFlowCloneMap(itemMap)
				} else {
					items[i] = item
				}
			}
			out[key] = items
		default:
			out[key] = value
		}
	}
	return out
}

func appFlowCloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
