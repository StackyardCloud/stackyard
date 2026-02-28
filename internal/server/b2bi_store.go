package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	b2biDefaultRegion    = "us-east-1"
	b2biDefaultAccountID = "123456789012"
)

type b2biStore struct {
	mu sync.Mutex

	nextCapabilityID     int64
	nextProfileID        int64
	nextPartnershipID    int64
	nextTransformerID    int64
	nextTransformerJobID int64

	capabilities    map[string]map[string]any
	profiles        map[string]map[string]any
	partnerships    map[string]map[string]any
	transformers    map[string]map[string]any
	transformerJobs map[string]map[string]any
	tags            map[string]map[string]string
}

func newB2BIStore() *b2biStore {
	s := &b2biStore{
		nextCapabilityID:     2,
		nextProfileID:        2,
		nextPartnershipID:    2,
		nextTransformerID:    2,
		nextTransformerJobID: 2,
		capabilities:         map[string]map[string]any{},
		profiles:             map[string]map[string]any{},
		partnerships:         map[string]map[string]any{},
		transformers:         map[string]map[string]any{},
		transformerJobs:      map[string]map[string]any{},
		tags:                 map[string]map[string]string{},
	}
	s.seedLocked(time.Now().UTC())
	return s
}

func (s *b2biStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	nowRFC3339 := now.Format(time.RFC3339)

	capabilityID := b2biFirstString(payload, []string{"capabilityId", "CapabilityId"}, "cap-000001")
	profileID := b2biFirstString(payload, []string{"profileId", "ProfileId"}, "prof-000001")
	partnershipID := b2biFirstString(payload, []string{"partnershipId", "PartnershipId"}, "part-000001")
	transformerID := b2biFirstString(payload, []string{"transformerId", "TransformerId"}, "trf-000001")
	transformerJobID := b2biFirstString(payload, []string{"transformerJobId", "TransformerJobId"}, "job-000001")

	switch action {
	case "CreateCapability":
		if capabilityID == "cap-000001" {
			capabilityID = b2biFirstString(payload, []string{"name", "Name"}, "")
			if capabilityID == "" {
				capabilityID = fmt.Sprintf("cap-%06d", s.nextCapabilityID)
				s.nextCapabilityID++
			}
		}
		item := s.ensureCapabilityLocked(capabilityID, nowRFC3339)
		if name := b2biFirstString(payload, []string{"name", "Name"}, ""); name != "" {
			item["name"] = name
		}
		item["updatedAt"] = nowRFC3339
		return map[string]any{"capability": b2biCloneMap(item)}
	case "GetCapability":
		return map[string]any{"capability": b2biCloneMap(s.ensureCapabilityLocked(capabilityID, nowRFC3339))}
	case "ListCapabilities":
		return map[string]any{"capabilities": b2biSortedListByKey(s.capabilities, "capabilityId"), "nextToken": ""}
	case "UpdateCapability":
		item := s.ensureCapabilityLocked(capabilityID, nowRFC3339)
		if name := b2biFirstString(payload, []string{"name", "Name"}, ""); name != "" {
			item["name"] = name
		}
		item["updatedAt"] = nowRFC3339
		return map[string]any{"capability": b2biCloneMap(item)}
	case "DeleteCapability":
		if item := s.capabilities[capabilityID]; item != nil {
			delete(s.tags, b2biAnyToString(item["capabilityArn"]))
		}
		delete(s.capabilities, capabilityID)
		return map[string]any{}

	case "CreateProfile":
		if profileID == "prof-000001" {
			profileID = b2biFirstString(payload, []string{"name", "Name"}, "")
			if profileID == "" {
				profileID = fmt.Sprintf("prof-%06d", s.nextProfileID)
				s.nextProfileID++
			}
		}
		item := s.ensureProfileLocked(profileID, nowRFC3339)
		if name := b2biFirstString(payload, []string{"name", "Name"}, ""); name != "" {
			item["name"] = name
		}
		item["updatedAt"] = nowRFC3339
		return map[string]any{"profile": b2biCloneMap(item)}
	case "GetProfile":
		return map[string]any{"profile": b2biCloneMap(s.ensureProfileLocked(profileID, nowRFC3339))}
	case "ListProfiles":
		return map[string]any{"profiles": b2biSortedListByKey(s.profiles, "profileId"), "nextToken": ""}
	case "UpdateProfile":
		item := s.ensureProfileLocked(profileID, nowRFC3339)
		if name := b2biFirstString(payload, []string{"name", "Name"}, ""); name != "" {
			item["name"] = name
		}
		item["updatedAt"] = nowRFC3339
		return map[string]any{"profile": b2biCloneMap(item)}
	case "DeleteProfile":
		if item := s.profiles[profileID]; item != nil {
			delete(s.tags, b2biAnyToString(item["profileArn"]))
		}
		delete(s.profiles, profileID)
		return map[string]any{}

	case "CreatePartnership":
		if partnershipID == "part-000001" {
			partnershipID = b2biFirstString(payload, []string{"name", "Name"}, "")
			if partnershipID == "" {
				partnershipID = fmt.Sprintf("part-%06d", s.nextPartnershipID)
				s.nextPartnershipID++
			}
		}
		item := s.ensurePartnershipLocked(partnershipID, profileID, capabilityID, nowRFC3339)
		if name := b2biFirstString(payload, []string{"name", "Name"}, ""); name != "" {
			item["name"] = name
		}
		item["updatedAt"] = nowRFC3339
		return map[string]any{"partnership": b2biCloneMap(item)}
	case "GetPartnership":
		return map[string]any{"partnership": b2biCloneMap(s.ensurePartnershipLocked(partnershipID, profileID, capabilityID, nowRFC3339))}
	case "ListPartnerships":
		return map[string]any{"partnerships": b2biSortedListByKey(s.partnerships, "partnershipId"), "nextToken": ""}
	case "UpdatePartnership":
		item := s.ensurePartnershipLocked(partnershipID, profileID, capabilityID, nowRFC3339)
		if name := b2biFirstString(payload, []string{"name", "Name"}, ""); name != "" {
			item["name"] = name
		}
		item["updatedAt"] = nowRFC3339
		return map[string]any{"partnership": b2biCloneMap(item)}
	case "DeletePartnership":
		if item := s.partnerships[partnershipID]; item != nil {
			delete(s.tags, b2biAnyToString(item["partnershipArn"]))
		}
		delete(s.partnerships, partnershipID)
		return map[string]any{}

	case "CreateTransformer":
		if transformerID == "trf-000001" {
			transformerID = b2biFirstString(payload, []string{"name", "Name"}, "")
			if transformerID == "" {
				transformerID = fmt.Sprintf("trf-%06d", s.nextTransformerID)
				s.nextTransformerID++
			}
		}
		item := s.ensureTransformerLocked(transformerID, nowRFC3339)
		if name := b2biFirstString(payload, []string{"name", "Name"}, ""); name != "" {
			item["name"] = name
		}
		item["updatedAt"] = nowRFC3339
		return map[string]any{"transformer": b2biCloneMap(item)}
	case "GetTransformer":
		return map[string]any{"transformer": b2biCloneMap(s.ensureTransformerLocked(transformerID, nowRFC3339))}
	case "ListTransformers":
		return map[string]any{"transformers": b2biSortedListByKey(s.transformers, "transformerId"), "nextToken": ""}
	case "UpdateTransformer":
		item := s.ensureTransformerLocked(transformerID, nowRFC3339)
		if name := b2biFirstString(payload, []string{"name", "Name"}, ""); name != "" {
			item["name"] = name
		}
		item["updatedAt"] = nowRFC3339
		return map[string]any{"transformer": b2biCloneMap(item)}
	case "DeleteTransformer":
		if item := s.transformers[transformerID]; item != nil {
			delete(s.tags, b2biAnyToString(item["transformerArn"]))
		}
		delete(s.transformers, transformerID)
		for id, job := range s.transformerJobs {
			if b2biAnyToString(job["transformerId"]) == transformerID {
				delete(s.transformerJobs, id)
			}
		}
		return map[string]any{}

	case "StartTransformerJob":
		if transformerJobID == "job-000001" {
			transformerJobID = fmt.Sprintf("job-%06d", s.nextTransformerJobID)
			s.nextTransformerJobID++
		}
		item := s.ensureTransformerJobLocked(transformerJobID, transformerID, nowRFC3339)
		item["status"] = "SUCCEEDED"
		item["updatedAt"] = nowRFC3339
		return map[string]any{"transformerJob": b2biCloneMap(item)}
	case "GetTransformerJob":
		return map[string]any{"transformerJob": b2biCloneMap(s.ensureTransformerJobLocked(transformerJobID, transformerID, nowRFC3339))}

	case "CreateStarterMappingTemplate":
		return map[string]any{
			"templateDetails": map[string]any{
				"name":      "starter-template",
				"version":   "1",
				"createdAt": nowRFC3339,
			},
			"mapping": "{}",
		}
	case "GenerateMapping":
		return map[string]any{
			"mapping": map[string]any{
				"content": "{}",
				"format":  "JSON",
			},
		}
	case "TestConversion":
		return map[string]any{
			"conversionResult": map[string]any{
				"status": "SUCCESS",
				"output": map[string]any{
					"bucketName": "stackyard",
					"key":        "conversion-output.edi",
				},
			},
		}
	case "TestMapping":
		return map[string]any{
			"mappingResult": map[string]any{
				"status":        "SUCCESS",
				"mappedContent": "{}",
			},
		}
	case "TestParsing":
		return map[string]any{
			"parsingResult": map[string]any{
				"status":   "SUCCESS",
				"segments": []any{"ISA", "GS", "ST"},
			},
		}

	case "TagResource":
		resourceARN := b2biFirstString(
			payload,
			[]string{"resourceArn", "resourceARN", "ResourceArn", "ResourceARN"},
			b2biTransformerARN("trf-000001"),
		)
		s.upsertTagsLocked(resourceARN, b2biExtractTags(payload))
		return map[string]any{}
	case "UntagResource":
		resourceARN := b2biFirstString(
			payload,
			[]string{"resourceArn", "resourceARN", "ResourceArn", "ResourceARN"},
			b2biTransformerARN("trf-000001"),
		)
		s.removeTagsLocked(resourceARN, b2biExtractTagKeys(payload))
		return map[string]any{}
	case "ListTagsForResource":
		resourceARN := b2biFirstString(
			payload,
			[]string{"resourceArn", "resourceARN", "ResourceArn", "ResourceARN"},
			b2biTransformerARN("trf-000001"),
		)
		return map[string]any{"tags": b2biCloneStringMap(s.tags[resourceARN])}
	}

	return map[string]any{}
}

func (s *b2biStore) seedLocked(now time.Time) {
	nowRFC3339 := now.Format(time.RFC3339)
	s.ensureCapabilityLocked("cap-000001", nowRFC3339)
	s.ensureProfileLocked("prof-000001", nowRFC3339)
	s.ensurePartnershipLocked("part-000001", "prof-000001", "cap-000001", nowRFC3339)
	s.ensureTransformerLocked("trf-000001", nowRFC3339)
	s.ensureTransformerJobLocked("job-000001", "trf-000001", nowRFC3339)
	s.tags[b2biTransformerARN("trf-000001")] = map[string]string{"stackyard": "true"}
}

func (s *b2biStore) ensureCapabilityLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "cap-000001"
	}
	if item := s.capabilities[id]; item != nil {
		return item
	}
	item := map[string]any{
		"capabilityId":  id,
		"capabilityArn": b2biCapabilityARN(id),
		"name":          "stackyard-capability-" + id,
		"state":         "ACTIVE",
		"createdAt":     now,
		"updatedAt":     now,
	}
	s.capabilities[id] = item
	return item
}

func (s *b2biStore) ensureProfileLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "prof-000001"
	}
	if item := s.profiles[id]; item != nil {
		return item
	}
	item := map[string]any{
		"profileId":  id,
		"profileArn": b2biProfileARN(id),
		"name":       "stackyard-profile-" + id,
		"state":      "ACTIVE",
		"createdAt":  now,
		"updatedAt":  now,
	}
	s.profiles[id] = item
	return item
}

func (s *b2biStore) ensurePartnershipLocked(id, profileID, capabilityID, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "part-000001"
	}
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		profileID = "prof-000001"
	}
	capabilityID = strings.TrimSpace(capabilityID)
	if capabilityID == "" {
		capabilityID = "cap-000001"
	}
	_ = s.ensureProfileLocked(profileID, now)
	_ = s.ensureCapabilityLocked(capabilityID, now)
	if item := s.partnerships[id]; item != nil {
		return item
	}
	item := map[string]any{
		"partnershipId":  id,
		"partnershipArn": b2biPartnershipARN(id),
		"profileId":      profileID,
		"capabilityId":   capabilityID,
		"name":           "stackyard-partnership-" + id,
		"state":          "ACTIVE",
		"createdAt":      now,
		"updatedAt":      now,
	}
	s.partnerships[id] = item
	return item
}

func (s *b2biStore) ensureTransformerLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "trf-000001"
	}
	if item := s.transformers[id]; item != nil {
		return item
	}
	item := map[string]any{
		"transformerId":  id,
		"transformerArn": b2biTransformerARN(id),
		"name":           "stackyard-transformer-" + id,
		"status":         "ACTIVE",
		"createdAt":      now,
		"updatedAt":      now,
	}
	s.transformers[id] = item
	return item
}

func (s *b2biStore) ensureTransformerJobLocked(id, transformerID, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "job-000001"
	}
	transformerID = strings.TrimSpace(transformerID)
	if transformerID == "" {
		transformerID = "trf-000001"
	}
	_ = s.ensureTransformerLocked(transformerID, now)
	if item := s.transformerJobs[id]; item != nil {
		return item
	}
	item := map[string]any{
		"transformerJobId":  id,
		"transformerJobArn": b2biTransformerJobARN(id),
		"transformerId":     transformerID,
		"status":            "SUCCEEDED",
		"createdAt":         now,
		"updatedAt":         now,
	}
	s.transformerJobs[id] = item
	return item
}

func (s *b2biStore) upsertTagsLocked(resourceARN string, tags map[string]string) {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" || len(tags) == 0 {
		return
	}
	current := s.tags[resourceARN]
	if current == nil {
		current = map[string]string{}
		s.tags[resourceARN] = current
	}
	for key, value := range tags {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		current[key] = value
	}
}

func (s *b2biStore) removeTagsLocked(resourceARN string, keys []string) {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" || len(keys) == 0 {
		return
	}
	current := s.tags[resourceARN]
	if len(current) == 0 {
		return
	}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		delete(current, key)
	}
}

func b2biCapabilityARN(id string) string {
	return fmt.Sprintf("arn:aws:b2bi:%s:%s:capability/%s", b2biDefaultRegion, b2biDefaultAccountID, strings.TrimSpace(id))
}

func b2biProfileARN(id string) string {
	return fmt.Sprintf("arn:aws:b2bi:%s:%s:profile/%s", b2biDefaultRegion, b2biDefaultAccountID, strings.TrimSpace(id))
}

func b2biPartnershipARN(id string) string {
	return fmt.Sprintf("arn:aws:b2bi:%s:%s:partnership/%s", b2biDefaultRegion, b2biDefaultAccountID, strings.TrimSpace(id))
}

func b2biTransformerARN(id string) string {
	return fmt.Sprintf("arn:aws:b2bi:%s:%s:transformer/%s", b2biDefaultRegion, b2biDefaultAccountID, strings.TrimSpace(id))
}

func b2biTransformerJobARN(id string) string {
	return fmt.Sprintf("arn:aws:b2bi:%s:%s:transformer-job/%s", b2biDefaultRegion, b2biDefaultAccountID, strings.TrimSpace(id))
}

func b2biFirstString(payload map[string]any, keys []string, def string) string {
	if payload == nil {
		return def
	}
	for _, key := range keys {
		if raw, ok := b2biMapLookupInsensitive(payload, key); ok {
			if value := b2biAnyToString(raw); value != "" {
				return value
			}
		}
	}
	return def
}

func b2biMapLookupInsensitive(payload map[string]any, key string) (any, bool) {
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return v, true
		}
	}
	return nil, false
}

func b2biAnyToString(raw any) string {
	switch v := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func b2biExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	raw, ok := b2biMapLookupInsensitive(payload, "tags")
	if !ok || raw == nil {
		return out
	}

	switch typed := raw.(type) {
	case map[string]any:
		for k, v := range typed {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			out[key] = b2biAnyToString(v)
		}
	case []any:
		for _, item := range typed {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := b2biFirstString(entry, []string{"key", "Key"}, "")
			if key == "" {
				continue
			}
			out[key] = b2biFirstString(entry, []string{"value", "Value"}, "")
		}
	}
	return out
}

func b2biExtractTagKeys(payload map[string]any) []string {
	raw, ok := b2biMapLookupInsensitive(payload, "tagKeys")
	if !ok || raw == nil {
		raw, ok = b2biMapLookupInsensitive(payload, "TagKeys")
		if !ok || raw == nil {
			return nil
		}
	}

	keys := make([]string, 0)
	switch typed := raw.(type) {
	case string:
		for _, part := range strings.Split(typed, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				keys = append(keys, part)
			}
		}
	case []any:
		for _, item := range typed {
			key := b2biAnyToString(item)
			if key != "" {
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func b2biCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch typed := v.(type) {
		case map[string]any:
			out[k] = b2biCloneMap(typed)
		case []any:
			items := make([]any, 0, len(typed))
			for _, item := range typed {
				if m, ok := item.(map[string]any); ok {
					items = append(items, b2biCloneMap(m))
				} else {
					items = append(items, item)
				}
			}
			out[k] = items
		default:
			out[k] = v
		}
	}
	return out
}

func b2biCloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func b2biSortedListByKey(items map[string]map[string]any, key string) []any {
	ids := make([]string, 0, len(items))
	for id := range items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		item := b2biCloneMap(items[id])
		if b2biAnyToString(item[key]) == "" {
			item[key] = id
		}
		out = append(out, item)
	}
	return out
}
