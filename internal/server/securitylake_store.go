package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type securityLakeStore struct {
	mu sync.Mutex

	delegatedAdmin string

	dataLakes                 map[string]map[string]any
	awsLogSources             map[string]map[string]any
	customLogSources          map[string]map[string]any
	subscribers               map[string]map[string]any
	subscriberNotifications   map[string]map[string]any
	dataLakeExceptionSub      map[string]any
	orgConfiguration          map[string]any
	dataLakeExceptions        []map[string]any
	tags                      map[string]map[string]string
	nextSubscriberSequenceNum int64
}

func newSecurityLakeStore() *securityLakeStore {
	now := time.Now().UTC().Format(time.RFC3339)
	region := "us-east-1"

	s := &securityLakeStore{
		delegatedAdmin: "123456789012",
		dataLakes:      map[string]map[string]any{},
		awsLogSources:  map[string]map[string]any{},
		customLogSources: map[string]map[string]any{
			"stackyard-custom-source": {
				"sourceName":       "stackyard-custom-source",
				"providerName":     "stackyard",
				"sourceVersion":    "1.0",
				"creationTime":     now,
				"lastUpdateTime":   now,
				"sourceCollection": "Security Lake",
			},
		},
		subscribers: map[string]map[string]any{
			"subscriber-00000001": {
				"subscriberId":   "subscriber-00000001",
				"subscriberName": "stackyard-subscriber",
				"subscriberArn":  securityLakeSubscriberARN("subscriber-00000001"),
				"createdAt":      now,
				"updatedAt":      now,
				"status":         "ACTIVE",
			},
		},
		subscriberNotifications: map[string]map[string]any{
			"subscriber-00000001": {
				"subscriberId": "subscriber-00000001",
				"endpoint":     "https://example.invalid/securitylake/subscriber",
				"status":       "ENABLED",
			},
		},
		dataLakeExceptionSub: map[string]any{
			"protocol": "SQS",
			"endpoint": "arn:aws:sqs:us-east-1:123456789012:securitylake-exceptions",
			"status":   "ENABLED",
		},
		orgConfiguration: map[string]any{
			"autoEnableNewAccount": map[string]any{
				"securityLake": true,
			},
			"region": region,
		},
		dataLakeExceptions: []map[string]any{
			{
				"region":        region,
				"exceptionName": "SeededException",
				"exception":     "seeded example",
				"createdAt":     now,
			},
		},
		tags:                      map[string]map[string]string{},
		nextSubscriberSequenceNum: 2,
	}

	s.dataLakes[region] = map[string]any{
		"region": region,
		"s3BucketArn": fmt.Sprintf(
			"arn:aws:s3:::aws-security-data-lake-%s-123456789012",
			region,
		),
		"createdAt":      now,
		"updatedAt":      now,
		"encryptionType": "SSE-S3",
		"status":         "COMPLETED",
	}
	s.awsLogSources["ROUTE53"] = map[string]any{
		"sourceName":     "ROUTE53",
		"sourceVersion":  "2.0",
		"creationTime":   now,
		"lastUpdateTime": now,
	}
	s.tags[securityLakeDataLakeARN(region)] = map[string]string{"seed": "true"}

	return s
}

func (s *securityLakeStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	region := securityLakeFirstNonEmpty(
		securityLakeStringAny(payload, "region"),
		securityLakeQueryString(query, "region"),
		"us-east-1",
	)
	subscriberID := securityLakeFirstNonEmpty(
		securityLakePathParam(pathParams, "subscriberId"),
		securityLakeStringAny(payload, "subscriberId"),
		"subscriber-00000001",
	)
	sourceName := securityLakeFirstNonEmpty(
		securityLakePathParam(pathParams, "sourceName"),
		securityLakeStringAny(payload, "sourceName"),
		"stackyard-custom-source",
	)
	resourceARN := securityLakeFirstNonEmpty(
		securityLakePathParam(pathParams, "resourceArn"),
		securityLakeStringAny(payload, "resourceArn"),
		securityLakeDataLakeARN(region),
	)

	switch action {
	case "CreateDataLake":
		s.ensureDataLakeLocked(region)["updatedAt"] = now
		return map[string]any{"regions": []any{region}}
	case "ListDataLakes":
		return map[string]any{"dataLakes": s.listDataLakesLocked(), "nextToken": ""}
	case "UpdateDataLake":
		dl := s.ensureDataLakeLocked(region)
		for k, v := range payload {
			dl[k] = v
		}
		dl["updatedAt"] = now
		return map[string]any{"region": region}
	case "DeleteDataLake":
		delete(s.dataLakes, region)
		return map[string]any{"regions": []any{region}}

	case "CreateAwsLogSource":
		name := securityLakeFirstNonEmpty(securityLakeStringAny(payload, "sourceName"), "ROUTE53")
		s.awsLogSources[name] = map[string]any{
			"sourceName":     name,
			"sourceVersion":  "2.0",
			"creationTime":   now,
			"lastUpdateTime": now,
		}
		return map[string]any{"sourceName": name}
	case "DeleteAwsLogSource":
		name := securityLakeFirstNonEmpty(securityLakeStringAny(payload, "sourceName"), "ROUTE53")
		delete(s.awsLogSources, name)
		return map[string]any{"sourceName": name}
	case "CreateCustomLogSource":
		sourceName = securityLakeFirstNonEmpty(securityLakeStringAny(payload, "sourceName"), sourceName)
		s.customLogSources[sourceName] = map[string]any{
			"sourceName":       sourceName,
			"providerName":     securityLakeFirstNonEmpty(securityLakeStringAny(payload, "providerName"), "stackyard"),
			"sourceVersion":    securityLakeFirstNonEmpty(securityLakeStringAny(payload, "sourceVersion"), "1.0"),
			"creationTime":     now,
			"lastUpdateTime":   now,
			"sourceCollection": "Security Lake",
		}
		return map[string]any{"sourceName": sourceName}
	case "DeleteCustomLogSource":
		delete(s.customLogSources, sourceName)
		return map[string]any{"sourceName": sourceName}
	case "ListLogSources":
		return map[string]any{
			"sources":   append(s.listAwsLogSourcesLocked(), s.listCustomLogSourcesLocked()...),
			"nextToken": "",
		}
	case "GetDataLakeSources":
		return map[string]any{
			"sources":   append(s.listAwsLogSourcesLocked(), s.listCustomLogSourcesLocked()...),
			"nextToken": "",
		}

	case "CreateDataLakeExceptionSubscription":
		for k, v := range payload {
			s.dataLakeExceptionSub[k] = v
		}
		s.dataLakeExceptionSub["status"] = "ENABLED"
		return map[string]any{}
	case "GetDataLakeExceptionSubscription":
		return securityLakeCloneMap(s.dataLakeExceptionSub)
	case "UpdateDataLakeExceptionSubscription":
		for k, v := range payload {
			s.dataLakeExceptionSub[k] = v
		}
		s.dataLakeExceptionSub["status"] = "ENABLED"
		return map[string]any{}
	case "DeleteDataLakeExceptionSubscription":
		s.dataLakeExceptionSub = map[string]any{}
		return map[string]any{}
	case "ListDataLakeExceptions":
		return map[string]any{"exceptions": securityLakeCloneListOfMaps(s.dataLakeExceptions), "nextToken": ""}

	case "CreateDataLakeOrganizationConfiguration":
		for k, v := range payload {
			s.orgConfiguration[k] = v
		}
		return map[string]any{}
	case "GetDataLakeOrganizationConfiguration":
		return securityLakeCloneMap(s.orgConfiguration)
	case "DeleteDataLakeOrganizationConfiguration":
		s.orgConfiguration = map[string]any{}
		return map[string]any{}
	case "RegisterDataLakeDelegatedAdministrator":
		s.delegatedAdmin = securityLakeFirstNonEmpty(securityLakeStringAny(payload, "accountId"), "123456789012")
		return map[string]any{"accountId": s.delegatedAdmin}
	case "DeregisterDataLakeDelegatedAdministrator":
		s.delegatedAdmin = ""
		return map[string]any{}

	case "CreateSubscriber":
		subscriberID = fmt.Sprintf("subscriber-%08d", s.nextSubscriberSequenceNum)
		s.nextSubscriberSequenceNum++
		s.subscribers[subscriberID] = map[string]any{
			"subscriberId":   subscriberID,
			"subscriberName": securityLakeFirstNonEmpty(securityLakeStringAny(payload, "subscriberName", "name"), subscriberID),
			"subscriberArn":  securityLakeSubscriberARN(subscriberID),
			"createdAt":      now,
			"updatedAt":      now,
			"status":         "ACTIVE",
		}
		return map[string]any{"subscriber": securityLakeCloneMap(s.subscribers[subscriberID])}
	case "GetSubscriber":
		return map[string]any{"subscriber": securityLakeCloneMap(s.ensureSubscriberLocked(subscriberID))}
	case "ListSubscribers":
		return map[string]any{"subscribers": s.listSubscribersLocked(), "nextToken": ""}
	case "UpdateSubscriber":
		sub := s.ensureSubscriberLocked(subscriberID)
		for k, v := range payload {
			sub[k] = v
		}
		sub["updatedAt"] = now
		return map[string]any{"subscriber": securityLakeCloneMap(sub)}
	case "DeleteSubscriber":
		delete(s.subscribers, subscriberID)
		delete(s.subscriberNotifications, subscriberID)
		return map[string]any{}
	case "CreateSubscriberNotification":
		s.subscriberNotifications[subscriberID] = map[string]any{
			"subscriberId": subscriberID,
			"endpoint":     securityLakeFirstNonEmpty(securityLakeStringAny(payload, "endpoint"), "https://example.invalid/securitylake/subscriber"),
			"status":       "ENABLED",
		}
		return map[string]any{}
	case "UpdateSubscriberNotification":
		notification := s.ensureSubscriberNotificationLocked(subscriberID)
		for k, v := range payload {
			notification[k] = v
		}
		notification["status"] = "ENABLED"
		return map[string]any{}
	case "DeleteSubscriberNotification":
		delete(s.subscriberNotifications, subscriberID)
		return map[string]any{}

	case "ListTagsForResource":
		out := map[string]any{}
		for k, v := range s.ensureTagsLocked(resourceARN) {
			out[k] = v
		}
		return map[string]any{"tags": out}
	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		if raw, ok := payload["tags"].(map[string]any); ok {
			for k, v := range raw {
				key := strings.TrimSpace(k)
				if key == "" {
					continue
				}
				tags[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
		if raw, ok := payload["Tags"].(map[string]any); ok {
			for k, v := range raw {
				key := strings.TrimSpace(k)
				if key == "" {
					continue
				}
				tags[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
		return map[string]any{}
	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range securityLakeStringSlicePayload(payload, "tagKeys", "TagKeys") {
			delete(tags, key)
		}
		for _, key := range strings.Split(securityLakeQueryString(query, "tagKeys"), ",") {
			key = strings.TrimSpace(key)
			if key != "" {
				delete(tags, key)
			}
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *securityLakeStore) ensureDataLakeLocked(region string) map[string]any {
	region = strings.TrimSpace(region)
	if region == "" {
		region = "us-east-1"
	}
	if existing, ok := s.dataLakes[region]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := map[string]any{
		"region":         region,
		"s3BucketArn":    fmt.Sprintf("arn:aws:s3:::aws-security-data-lake-%s-123456789012", region),
		"createdAt":      now,
		"updatedAt":      now,
		"encryptionType": "SSE-S3",
		"status":         "COMPLETED",
	}
	s.dataLakes[region] = created
	return created
}

func (s *securityLakeStore) ensureSubscriberLocked(subscriberID string) map[string]any {
	subscriberID = strings.TrimSpace(subscriberID)
	if subscriberID == "" {
		subscriberID = "subscriber-00000001"
	}
	if existing, ok := s.subscribers[subscriberID]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := map[string]any{
		"subscriberId":   subscriberID,
		"subscriberName": subscriberID,
		"subscriberArn":  securityLakeSubscriberARN(subscriberID),
		"createdAt":      now,
		"updatedAt":      now,
		"status":         "ACTIVE",
	}
	s.subscribers[subscriberID] = created
	return created
}

func (s *securityLakeStore) ensureSubscriberNotificationLocked(subscriberID string) map[string]any {
	subscriberID = strings.TrimSpace(subscriberID)
	if subscriberID == "" {
		subscriberID = "subscriber-00000001"
	}
	if existing, ok := s.subscriberNotifications[subscriberID]; ok {
		return existing
	}
	created := map[string]any{
		"subscriberId": subscriberID,
		"endpoint":     "https://example.invalid/securitylake/subscriber",
		"status":       "ENABLED",
	}
	s.subscriberNotifications[subscriberID] = created
	return created
}

func (s *securityLakeStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = securityLakeDataLakeARN("us-east-1")
	}
	if existing, ok := s.tags[resourceARN]; ok {
		return existing
	}
	created := map[string]string{}
	s.tags[resourceARN] = created
	return created
}

func (s *securityLakeStore) listDataLakesLocked() []any {
	keys := make([]string, 0, len(s.dataLakes))
	for k := range s.dataLakes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, securityLakeCloneMap(s.dataLakes[k]))
	}
	return out
}

func (s *securityLakeStore) listAwsLogSourcesLocked() []any {
	keys := make([]string, 0, len(s.awsLogSources))
	for k := range s.awsLogSources {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, securityLakeCloneMap(s.awsLogSources[k]))
	}
	return out
}

func (s *securityLakeStore) listCustomLogSourcesLocked() []any {
	keys := make([]string, 0, len(s.customLogSources))
	for k := range s.customLogSources {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, securityLakeCloneMap(s.customLogSources[k]))
	}
	return out
}

func (s *securityLakeStore) listSubscribersLocked() []any {
	keys := make([]string, 0, len(s.subscribers))
	for k := range s.subscribers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, securityLakeCloneMap(s.subscribers[k]))
	}
	return out
}

func securityLakePathParam(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	if v := strings.TrimSpace(pathParams[key]); v != "" {
		return v
	}
	for k, v := range pathParams {
		if strings.EqualFold(strings.TrimSpace(k), key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func securityLakeStringAny(payload map[string]any, keys ...string) string {
	if payload == nil {
		return ""
	}
	for _, key := range keys {
		for k, v := range payload {
			if !strings.EqualFold(strings.TrimSpace(k), key) {
				continue
			}
			if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" {
				return s
			}
		}
	}
	return ""
}

func securityLakeStringSlicePayload(payload map[string]any, keys ...string) []string {
	if payload == nil {
		return nil
	}
	for _, key := range keys {
		for k, v := range payload {
			if !strings.EqualFold(strings.TrimSpace(k), key) {
				continue
			}
			switch vv := v.(type) {
			case []string:
				out := make([]string, 0, len(vv))
				for _, item := range vv {
					if s := strings.TrimSpace(item); s != "" {
						out = append(out, s)
					}
				}
				return out
			case []any:
				out := make([]string, 0, len(vv))
				for _, item := range vv {
					if s := strings.TrimSpace(fmt.Sprintf("%v", item)); s != "" {
						out = append(out, s)
					}
				}
				return out
			case string:
				raw := strings.TrimSpace(vv)
				if raw == "" {
					return nil
				}
				parts := strings.Split(raw, ",")
				out := make([]string, 0, len(parts))
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						out = append(out, p)
					}
				}
				return out
			}
		}
	}
	return nil
}

func securityLakeQueryString(values url.Values, key string) string {
	if values == nil {
		return ""
	}
	if v := strings.TrimSpace(values.Get(key)); v != "" {
		return v
	}
	for k, vals := range values {
		if !strings.EqualFold(k, key) || len(vals) == 0 {
			continue
		}
		if v := strings.TrimSpace(vals[0]); v != "" {
			return v
		}
	}
	return ""
}

func securityLakeFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func securityLakeCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func securityLakeCloneListOfMaps(in []map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, securityLakeCloneMap(item))
	}
	return out
}

func securityLakeDataLakeARN(region string) string {
	region = strings.TrimSpace(region)
	if region == "" {
		region = "us-east-1"
	}
	return fmt.Sprintf("arn:aws:securitylake:%s:123456789012:datalake/default", region)
}

func securityLakeSubscriberARN(subscriberID string) string {
	subscriberID = strings.TrimSpace(subscriberID)
	if subscriberID == "" {
		subscriberID = "subscriber-00000001"
	}
	return fmt.Sprintf("arn:aws:securitylake:us-east-1:123456789012:subscriber/%s", subscriberID)
}
