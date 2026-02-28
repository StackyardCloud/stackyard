package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type quickSetupStore struct {
	mu sync.Mutex

	nextManager int64
	nextConfig  int64

	managers          map[string]map[string]any
	configurations    map[string]map[string]any
	tags              map[string]map[string]string
	serviceSettings   map[string]any
	quickSetupTypeOut []map[string]any
}

func newQuickSetupStore() *quickSetupStore {
	s := &quickSetupStore{
		nextManager:    2,
		nextConfig:     2,
		managers:       map[string]map[string]any{},
		configurations: map[string]map[string]any{},
		tags:           map[string]map[string]string{},
		serviceSettings: map[string]any{
			"ExplorerEnablingRoleArn": "arn:aws:iam::123456789012:role/stackyard-quicksetup",
		},
		quickSetupTypeOut: []map[string]any{
			{
				"Type":          "AWSQuickSetupType-SSMHostMgmt",
				"LatestVersion": "1.0",
			},
		},
	}

	manager := s.ensureManagerLocked("arn:aws:ssm-quicksetup:us-east-1:123456789012:configuration-manager/cm-000001")
	manager["Name"] = "stackyard-manager"
	manager["Description"] = "stackyard seeded quick setup manager"
	config := s.ensureConfigurationLocked("cfg-000001", qsStringAny(manager, "ManagerArn", ""))
	config["Type"] = "AWSQuickSetupType-SSMHostMgmt"

	managerARN := qsStringAny(manager, "ManagerArn", "")
	s.tags[managerARN] = map[string]string{"stackyard": "true"}
	return s
}

func (s *quickSetupStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateConfigurationManager":
		managerARN := qsStringAny(payload, "ManagerArn", "")
		if managerARN == "" {
			managerARN = fmt.Sprintf(
				"arn:aws:ssm-quicksetup:us-east-1:123456789012:configuration-manager/cm-%06d",
				s.nextManagerIDLocked(),
			)
		}
		manager := s.ensureManagerLocked(managerARN)
		for key, value := range payload {
			manager[key] = value
		}
		manager["LastModifiedAt"] = time.Now().UTC()
		s.ensureTagsLocked(managerARN)
		return map[string]any{"ManagerArn": managerARN}

	case "DeleteConfigurationManager":
		managerARN := qsPathString(pathParams, "ManagerArn", "")
		delete(s.managers, managerARN)
		delete(s.tags, managerARN)
		return map[string]any{}

	case "GetConfiguration":
		configID := qsPathString(pathParams, "ConfigurationId", "cfg-000001")
		cfg := s.ensureConfigurationLocked(configID, s.firstManagerARNLocked())
		return map[string]any{"Configuration": qsCloneMap(cfg)}

	case "GetConfigurationManager":
		managerARN := qsPathString(pathParams, "ManagerArn", s.firstManagerARNLocked())
		manager := s.ensureManagerLocked(managerARN)
		return map[string]any{
			"ManagerArn":             managerARN,
			"ConfigurationManager":   qsCloneMap(manager),
			"StatusSummaries":        qsCloneAnySlice(qsAnySliceAny(manager, "StatusSummaries")),
			"ConfigurationSummaries": s.configurationSummariesForManagerLocked(managerARN),
		}

	case "GetServiceSettings":
		return map[string]any{"ServiceSettings": qsCloneMap(s.serviceSettings)}

	case "ListConfigurationManagers":
		arns := make([]string, 0, len(s.managers))
		for arn := range s.managers {
			arns = append(arns, arn)
		}
		sort.Strings(arns)
		items := make([]any, 0, len(arns))
		for _, arn := range arns {
			items = append(items, qsCloneMap(s.managerSummaryLocked(arn)))
		}
		return map[string]any{
			"ConfigurationManagers": items,
			"NextToken":             "",
		}

	case "ListConfigurations":
		ids := make([]string, 0, len(s.configurations))
		for id := range s.configurations {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		items := make([]any, 0, len(ids))
		for _, id := range ids {
			items = append(items, qsCloneMap(s.configurationSummaryLocked(id)))
		}
		return map[string]any{
			"Configurations": items,
			"NextToken":      "",
		}

	case "ListQuickSetupTypes":
		items := make([]any, 0, len(s.quickSetupTypeOut))
		for _, item := range s.quickSetupTypeOut {
			items = append(items, qsCloneMap(item))
		}
		return map[string]any{
			"QuickSetupTypeList": items,
			"NextToken":          "",
		}

	case "ListTagsForResource":
		resourceARN := qsPathString(pathParams, "ResourceArn", s.firstManagerARNLocked())
		s.ensureTagsLocked(resourceARN)
		return map[string]any{"Tags": qsTagEntries(s.tags[resourceARN])}

	case "TagResource":
		resourceARN := qsPathString(pathParams, "ResourceArn", s.firstManagerARNLocked())
		s.ensureTagsLocked(resourceARN)
		for key, value := range qsMapString(payload, "Tags") {
			s.tags[resourceARN][key] = value
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := qsPathString(pathParams, "ResourceArn", s.firstManagerARNLocked())
		s.ensureTagsLocked(resourceARN)
		for _, key := range qsQueryOrPayloadStrings(query, payload, "tagKeys", "TagKeys") {
			delete(s.tags[resourceARN], key)
		}
		return map[string]any{}

	case "UpdateConfigurationDefinition":
		managerARN := qsPathString(pathParams, "ManagerArn", s.firstManagerARNLocked())
		definitionID := qsPathString(pathParams, "Id", "def-000001")
		manager := s.ensureManagerLocked(managerARN)
		manager["LastModifiedAt"] = time.Now().UTC()
		manager["LastUpdatedDefinitionId"] = definitionID
		manager["LastUpdatedDefinitionPayload"] = qsCloneMap(payload)
		return map[string]any{}

	case "UpdateConfigurationManager":
		managerARN := qsPathString(pathParams, "ManagerArn", s.firstManagerARNLocked())
		manager := s.ensureManagerLocked(managerARN)
		for key, value := range payload {
			manager[key] = value
		}
		manager["LastModifiedAt"] = time.Now().UTC()
		return map[string]any{"ManagerArn": managerARN}

	case "UpdateServiceSettings":
		for key, value := range payload {
			s.serviceSettings[key] = value
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *quickSetupStore) ensureManagerLocked(managerARN string) map[string]any {
	arn := strings.TrimSpace(managerARN)
	if arn == "" {
		arn = s.firstManagerARNLocked()
	}
	if existing := s.managers[arn]; existing != nil {
		return existing
	}
	now := time.Now().UTC()
	manager := map[string]any{
		"ManagerArn": arn,
		"StatusSummaries": []any{
			map[string]any{"StatusType": "Deployment", "Status": "SUCCEEDED", "LastUpdatedAt": now},
		},
		"CreatedAt":      now,
		"LastModifiedAt": now,
	}
	s.managers[arn] = manager
	return manager
}

func (s *quickSetupStore) ensureConfigurationLocked(configID, managerARN string) map[string]any {
	id := strings.TrimSpace(configID)
	if id == "" {
		id = fmt.Sprintf("cfg-%06d", s.nextConfigIDLocked())
	}
	if existing := s.configurations[id]; existing != nil {
		return existing
	}
	if managerARN == "" {
		managerARN = s.firstManagerARNLocked()
	}
	now := time.Now().UTC()
	cfg := map[string]any{
		"ConfigurationId": id,
		"ManagerArn":      managerARN,
		"StatusSummaries": []any{
			map[string]any{"StatusType": "Deployment", "Status": "SUCCEEDED", "LastUpdatedAt": now},
		},
		"CreatedAt":      now,
		"LastModifiedAt": now,
	}
	s.configurations[id] = cfg
	return cfg
}

func (s *quickSetupStore) managerSummaryLocked(managerARN string) map[string]any {
	manager := s.ensureManagerLocked(managerARN)
	return map[string]any{
		"ManagerArn":      qsStringAny(manager, "ManagerArn", managerARN),
		"Name":            qsStringAny(manager, "Name", "stackyard-manager"),
		"Description":     qsStringAny(manager, "Description", "stackyard quick setup manager"),
		"StatusSummaries": qsCloneAnySlice(qsAnySliceAny(manager, "StatusSummaries")),
	}
}

func (s *quickSetupStore) configurationSummaryLocked(configID string) map[string]any {
	cfg := s.ensureConfigurationLocked(configID, s.firstManagerARNLocked())
	return map[string]any{
		"ConfigurationId": qsStringAny(cfg, "ConfigurationId", configID),
		"ManagerArn":      qsStringAny(cfg, "ManagerArn", s.firstManagerARNLocked()),
		"Type":            qsStringAny(cfg, "Type", "AWSQuickSetupType-SSMHostMgmt"),
		"StatusSummaries": qsCloneAnySlice(qsAnySliceAny(cfg, "StatusSummaries")),
	}
}

func (s *quickSetupStore) configurationSummariesForManagerLocked(managerARN string) []any {
	ids := make([]string, 0, len(s.configurations))
	for id, cfg := range s.configurations {
		if strings.TrimSpace(qsStringAny(cfg, "ManagerArn", "")) == strings.TrimSpace(managerARN) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.configurationSummaryLocked(id))
	}
	return out
}

func (s *quickSetupStore) ensureTagsLocked(resourceARN string) {
	arn := strings.TrimSpace(resourceARN)
	if arn == "" {
		arn = s.firstManagerARNLocked()
	}
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{}
	}
}

func (s *quickSetupStore) firstManagerARNLocked() string {
	if len(s.managers) == 0 {
		return "arn:aws:ssm-quicksetup:us-east-1:123456789012:configuration-manager/cm-000001"
	}
	arns := make([]string, 0, len(s.managers))
	for arn := range s.managers {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	return arns[0]
}

func (s *quickSetupStore) nextManagerIDLocked() int64 {
	id := s.nextManager
	s.nextManager++
	return id
}

func (s *quickSetupStore) nextConfigIDLocked() int64 {
	id := s.nextConfig
	s.nextConfig++
	return id
}

func qsPathString(values map[string]string, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			value := strings.TrimSpace(v)
			if value != "" {
				return value
			}
			break
		}
	}
	return fallback
}

func qsStringAny(values map[string]any, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if value, ok := v.(string); ok {
				value = strings.TrimSpace(value)
				if value != "" {
					return value
				}
			}
			break
		}
	}
	return fallback
}

func qsAnySliceAny(values map[string]any, key string) []any {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if out, ok := v.([]any); ok {
				return out
			}
			break
		}
	}
	return nil
}

func qsMapString(payload map[string]any, key string) map[string]string {
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		if raw, ok := v.([]any); ok {
			out := map[string]string{}
			for _, item := range raw {
				entry, ok := item.(map[string]any)
				if !ok {
					continue
				}
				tagKey := qsStringAny(entry, "Key", "")
				tagVal := qsStringAny(entry, "Value", "")
				if tagKey != "" {
					out[tagKey] = tagVal
				}
			}
			return out
		}
		if raw, ok := v.(map[string]any); ok {
			out := map[string]string{}
			for rk, rv := range raw {
				if str, ok := rv.(string); ok {
					out[rk] = str
				}
			}
			return out
		}
		if raw, ok := v.(map[string]string); ok {
			return qsCloneStringMap(raw)
		}
	}
	return map[string]string{}
}

func qsQueryOrPayloadStrings(query url.Values, payload map[string]any, queryKey, payloadKey string) []string {
	seen := map[string]struct{}{}
	out := []string{}

	for key, values := range query {
		if !strings.EqualFold(strings.TrimSpace(key), strings.TrimSpace(queryKey)) {
			continue
		}
		for _, value := range values {
			for _, token := range strings.Split(value, ",") {
				token = strings.TrimSpace(token)
				if token == "" {
					continue
				}
				if _, ok := seen[token]; ok {
					continue
				}
				seen[token] = struct{}{}
				out = append(out, token)
			}
		}
	}

	for _, value := range payload {
		items, ok := value.([]any)
		if !ok {
			continue
		}
		for _, item := range items {
			token, ok := item.(string)
			if !ok {
				continue
			}
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			out = append(out, token)
		}
	}

	for key, value := range payload {
		if !strings.EqualFold(strings.TrimSpace(key), strings.TrimSpace(payloadKey)) {
			continue
		}
		items, ok := value.([]any)
		if !ok {
			continue
		}
		for _, item := range items {
			token, ok := item.(string)
			if !ok {
				continue
			}
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			out = append(out, token)
		}
	}
	return out
}

func qsCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func qsCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func qsCloneAnySlice(in []any) []any {
	if in == nil {
		return []any{}
	}
	out := make([]any, 0, len(in))
	out = append(out, in...)
	return out
}

func qsTagEntries(tags map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}
