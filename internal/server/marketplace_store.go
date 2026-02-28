package server

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type marketplaceStore struct {
	mu sync.Mutex

	nextID int

	entities             map[string]map[string]any
	changeSets           map[string]map[string]any
	resourcePolicies     map[string]string
	agreements           map[string]map[string]any
	deploymentParameters map[string]map[string]string
	tags                 map[string]map[string]string
	entitlements         []map[string]any
}

func newMarketplaceStore() *marketplaceStore {
	s := &marketplaceStore{
		nextID: 2,

		entities: map[string]map[string]any{
			"entity-000001": {
				"EntityId":         "entity-000001",
				"EntityType":       "AmiProduct",
				"Name":             "stackyard-sample-product",
				"LastModifiedDate": "2026-01-01T00:00:00Z",
			},
		},
		changeSets: map[string]map[string]any{
			"cs-000001": {
				"ChangeSetId":   "cs-000001",
				"ChangeSetArn":  "arn:aws:aws-marketplace:us-east-1:123456789012:change-set/cs-000001",
				"ChangeSetName": "stackyard-initial-change-set",
				"Status":        "SUCCEEDED",
			},
		},
		resourcePolicies: map[string]string{
			"arn:aws:aws-marketplace:us-east-1:123456789012:resource/default": `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`,
		},
		agreements: map[string]map[string]any{
			"agr-000001": {
				"AgreementId": "agr-000001",
				"Status":      "ACTIVE",
				"Acceptor": map[string]any{
					"AccountId": "123456789012",
				},
				"Proposer": map[string]any{
					"AccountId": "210987654321",
				},
			},
		},
		deploymentParameters: map[string]map[string]string{},
		tags: map[string]map[string]string{
			"arn:aws:aws-marketplace:us-east-1:123456789012:resource/default": {
				"stackyard": "true",
			},
		},
		entitlements: []map[string]any{
			{
				"Dimension":    "users",
				"ProductCode":  "prod-000001",
				"IntegerValue": 10,
			},
		},
	}
	return s
}

func (s *marketplaceStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "BatchDescribeEntities":
		entities := make([]any, 0, len(s.entities))
		for _, entity := range marketplaceSortedMaps(s.entities) {
			entities = append(entities, marketplaceCloneMap(entity))
		}
		return map[string]any{"EntityDetails": entities}

	case "CancelChangeSet":
		changeSetID := marketplacePayloadString(payload, "ChangeSetId", s.firstChangeSetIDLocked())
		changeSet := s.ensureChangeSetLocked(changeSetID)
		changeSet["Status"] = "CANCELLED"
		return map[string]any{
			"ChangeSetId":  changeSet["ChangeSetId"],
			"ChangeSetArn": changeSet["ChangeSetArn"],
			"Status":       changeSet["Status"],
		}

	case "DeleteResourcePolicy":
		resourceArn := marketplacePayloadString(payload, "ResourceArn", s.defaultResourceARN())
		delete(s.resourcePolicies, resourceArn)
		return map[string]any{}

	case "DescribeChangeSet":
		changeSetID := marketplacePayloadString(payload, "ChangeSetId", s.firstChangeSetIDLocked())
		return marketplaceCloneMap(s.ensureChangeSetLocked(changeSetID))

	case "DescribeEntity":
		entityID := marketplacePayloadString(payload, "EntityId", s.firstEntityIDLocked())
		entity := s.ensureEntityLocked(entityID)
		return map[string]any{
			"EntityId":        entity["EntityId"],
			"EntityType":      entity["EntityType"],
			"DetailsDocument": marketplaceCloneMap(entity),
		}

	case "GetResourcePolicy":
		resourceArn := marketplacePayloadString(payload, "ResourceArn", s.defaultResourceARN())
		policy := s.resourcePolicies[resourceArn]
		if strings.TrimSpace(policy) == "" {
			policy = `{"Version":"2012-10-17","Statement":[]}`
			s.resourcePolicies[resourceArn] = policy
		}
		return map[string]any{
			"ResourceArn": resourceArn,
			"Policy":      policy,
		}

	case "ListChangeSets":
		items := make([]any, 0, len(s.changeSets))
		for _, item := range marketplaceSortedMaps(s.changeSets) {
			items = append(items, marketplaceCloneMap(item))
		}
		return map[string]any{
			"ChangeSetSummaryList": items,
			"NextToken":            "",
		}

	case "ListEntities":
		items := make([]any, 0, len(s.entities))
		for _, item := range marketplaceSortedMaps(s.entities) {
			items = append(items, marketplaceCloneMap(item))
		}
		return map[string]any{
			"EntitySummaryList": items,
			"NextToken":         "",
		}

	case "ListTagsForResource":
		resourceArn := marketplacePayloadString(payload, "ResourceArn", s.defaultResourceARN())
		tags := marketplaceCloneStringMap(s.tags[resourceArn])
		if tags == nil {
			tags = map[string]string{}
		}
		return map[string]any{"Tags": tags}

	case "PutResourcePolicy":
		resourceArn := marketplacePayloadString(payload, "ResourceArn", s.defaultResourceARN())
		policy := marketplacePayloadString(payload, "Policy", `{"Version":"2012-10-17","Statement":[]}`)
		s.resourcePolicies[resourceArn] = policy
		return map[string]any{
			"ResourceArn": resourceArn,
			"Policy":      policy,
		}

	case "StartChangeSet":
		changeSetID := s.nextChangeSetIDLocked()
		changeSet := s.ensureChangeSetLocked(changeSetID)
		changeSet["ChangeSetName"] = marketplacePayloadString(payload, "ChangeSetName", "stackyard-change-set")
		changeSet["Status"] = "PREPARING"
		return map[string]any{
			"ChangeSetId":  changeSet["ChangeSetId"],
			"ChangeSetArn": changeSet["ChangeSetArn"],
		}

	case "TagResource":
		resourceArn := marketplacePayloadString(payload, "ResourceArn", s.defaultResourceARN())
		incoming := marketplacePayloadTags(payload, "Tags")
		if len(incoming) == 0 {
			incoming = map[string]string{"stackyard": "true"}
		}
		tags := s.tags[resourceArn]
		if tags == nil {
			tags = map[string]string{}
			s.tags[resourceArn] = tags
		}
		for k, v := range incoming {
			tags[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		resourceArn := marketplacePayloadString(payload, "ResourceArn", s.defaultResourceARN())
		tagKeys := marketplacePayloadStringSlice(payload, "TagKeys")
		tags := s.tags[resourceArn]
		if tags == nil {
			return map[string]any{}
		}
		if len(tagKeys) == 0 {
			delete(tags, "stackyard")
			if len(tags) == 0 {
				delete(s.tags, resourceArn)
			}
			return map[string]any{}
		}
		for _, key := range tagKeys {
			delete(tags, key)
		}
		if len(tags) == 0 {
			delete(s.tags, resourceArn)
		}
		return map[string]any{}

	case "DescribeAgreement":
		agreementID := marketplacePayloadString(payload, "AgreementId", s.firstAgreementIDLocked())
		agreement := s.ensureAgreementLocked(agreementID)
		return map[string]any{"AgreementView": marketplaceCloneMap(agreement)}

	case "GetAgreementTerms":
		return map[string]any{
			"AcceptedTerms": []any{
				map[string]any{"Type": "UsageBasedPricingTerm"},
				map[string]any{"Type": "ConfigurableUpfrontPricingTerm"},
			},
			"NextToken": "",
		}

	case "SearchAgreements":
		items := make([]any, 0, len(s.agreements))
		for _, agreement := range marketplaceSortedMaps(s.agreements) {
			items = append(items, marketplaceCloneMap(agreement))
		}
		return map[string]any{
			"AgreementViewSummaries": items,
			"NextToken":              "",
		}

	case "BatchMeterUsage":
		records := marketplacePayloadSlice(payload, "UsageRecords")
		results := make([]any, 0, len(records))
		for idx := range records {
			results = append(results, map[string]any{
				"MeteringRecordId": fmt.Sprintf("mru-%06d", idx+1),
				"Status":           "Success",
			})
		}
		if len(results) == 0 {
			results = append(results, map[string]any{
				"MeteringRecordId": "mru-000001",
				"Status":           "Success",
			})
		}
		return map[string]any{
			"Results":            results,
			"UnprocessedRecords": []any{},
		}

	case "MeterUsage":
		return map[string]any{
			"MeteringRecordId": fmt.Sprintf("mr-%06d", s.nextID),
			"Status":           "Success",
		}

	case "RegisterUsage":
		return map[string]any{
			"PublicKeyRotationTimestamp": "2026-01-01T00:00:00Z",
			"Signature":                  "stackyard-signature",
		}

	case "ResolveCustomer":
		return map[string]any{
			"CustomerIdentifier":   "cust-000001",
			"ProductCode":          "prod-000001",
			"CustomerAWSAccountId": "123456789012",
		}

	case "GetEntitlements":
		items := make([]any, 0, len(s.entitlements))
		for _, item := range s.entitlements {
			items = append(items, marketplaceCloneMap(item))
		}
		return map[string]any{
			"Entitlements": items,
			"NextToken":    "",
		}

	case "PutDeploymentParameter":
		resourceArn := marketplacePayloadString(payload, "ResourceArn", s.defaultResourceARN())
		parameterName := marketplacePayloadString(payload, "ParameterName", marketplacePayloadString(payload, "Name", "stackyardParameter"))
		parameterValue := marketplacePayloadString(payload, "ParameterValue", marketplacePayloadString(payload, "Value", "stackyardValue"))
		if s.deploymentParameters[resourceArn] == nil {
			s.deploymentParameters[resourceArn] = map[string]string{}
		}
		s.deploymentParameters[resourceArn][parameterName] = parameterValue
		return map[string]any{
			"ResourceArn":    resourceArn,
			"ParameterName":  parameterName,
			"ParameterValue": parameterValue,
		}

	case "GetBuyerDashboard":
		return map[string]any{
			"Url":       "https://example.com/marketplace/dashboard",
			"Dashboard": "stackyard",
		}
	}

	return map[string]any{}
}

func (s *marketplaceStore) defaultResourceARN() string {
	return "arn:aws:aws-marketplace:us-east-1:123456789012:resource/default"
}

func (s *marketplaceStore) nextChangeSetIDLocked() string {
	id := fmt.Sprintf("cs-%06d", s.nextID)
	s.nextID++
	return id
}

func (s *marketplaceStore) firstEntityIDLocked() string {
	if len(s.entities) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.entities))
	for key := range s.entities {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *marketplaceStore) firstChangeSetIDLocked() string {
	if len(s.changeSets) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.changeSets))
	for key := range s.changeSets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *marketplaceStore) firstAgreementIDLocked() string {
	if len(s.agreements) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.agreements))
	for key := range s.agreements {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *marketplaceStore) ensureEntityLocked(entityID string) map[string]any {
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		entityID = "entity-000001"
	}
	if entity, ok := s.entities[entityID]; ok {
		return entity
	}
	entity := map[string]any{
		"EntityId":         entityID,
		"EntityType":       "AmiProduct",
		"Name":             "stackyard-" + strings.ToLower(entityID),
		"LastModifiedDate": "2026-01-01T00:00:00Z",
	}
	s.entities[entityID] = entity
	return entity
}

func (s *marketplaceStore) ensureChangeSetLocked(changeSetID string) map[string]any {
	changeSetID = strings.TrimSpace(changeSetID)
	if changeSetID == "" {
		changeSetID = "cs-000001"
	}
	if changeSet, ok := s.changeSets[changeSetID]; ok {
		return changeSet
	}
	changeSet := map[string]any{
		"ChangeSetId":   changeSetID,
		"ChangeSetArn":  "arn:aws:aws-marketplace:us-east-1:123456789012:change-set/" + changeSetID,
		"ChangeSetName": "stackyard-change-set",
		"Status":        "SUCCEEDED",
	}
	s.changeSets[changeSetID] = changeSet
	return changeSet
}

func (s *marketplaceStore) ensureAgreementLocked(agreementID string) map[string]any {
	agreementID = strings.TrimSpace(agreementID)
	if agreementID == "" {
		agreementID = "agr-000001"
	}
	if agreement, ok := s.agreements[agreementID]; ok {
		return agreement
	}
	agreement := map[string]any{
		"AgreementId": agreementID,
		"Status":      "ACTIVE",
		"Acceptor": map[string]any{
			"AccountId": "123456789012",
		},
		"Proposer": map[string]any{
			"AccountId": "210987654321",
		},
	}
	s.agreements[agreementID] = agreement
	return agreement
}

func marketplacePayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return strings.TrimSpace(def)
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return strings.TrimSpace(def)
	}
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return strings.TrimSpace(def)
		}
		return strings.TrimSpace(v)
	default:
		str := strings.TrimSpace(fmt.Sprintf("%v", value))
		if str == "" {
			return strings.TrimSpace(def)
		}
		return str
	}
}

func marketplacePayloadSlice(payload map[string]any, key string) []any {
	if payload == nil {
		return nil
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return nil
	}
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]any, 0, len(raw))
	for _, item := range raw {
		out = append(out, item)
	}
	return out
}

func marketplacePayloadStringSlice(payload map[string]any, key string) []string {
	items := marketplacePayloadSlice(payload, key)
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				out = append(out, strings.TrimSpace(v))
			}
		default:
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func marketplacePayloadTags(payload map[string]any, key string) map[string]string {
	if payload == nil {
		return nil
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return nil
	}
	out := map[string]string{}
	switch tags := value.(type) {
	case map[string]any:
		for k, v := range tags {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	case map[string]string:
		for k, v := range tags {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(v)
		}
	case []any:
		for _, item := range tags {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			k := marketplacePayloadString(m, "Key", "")
			if k == "" {
				k = marketplacePayloadString(m, "key", "")
			}
			if k == "" {
				continue
			}
			v := marketplacePayloadString(m, "Value", "")
			if v == "" {
				v = marketplacePayloadString(m, "value", "")
			}
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func marketplaceSortedMaps(in map[string]map[string]any) []map[string]any {
	if len(in) == 0 {
		return nil
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, marketplaceCloneMap(in[key]))
	}
	return out
}

func marketplaceCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch typed := v.(type) {
		case map[string]any:
			out[k] = marketplaceCloneMap(typed)
		case map[string]string:
			out[k] = marketplaceCloneStringMap(typed)
		case []any:
			copied := make([]any, 0, len(typed))
			for _, item := range typed {
				if m, ok := item.(map[string]any); ok {
					copied = append(copied, marketplaceCloneMap(m))
					continue
				}
				copied = append(copied, item)
			}
			out[k] = copied
		default:
			out[k] = typed
		}
	}
	return out
}

func marketplaceCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func marketplacePayloadInt(payload map[string]any, key string, def int) int {
	if payload == nil {
		return def
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return def
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
			return parsed
		}
	}
	return def
}
