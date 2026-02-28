package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type healthStore struct {
	mu        sync.Mutex
	nextID    int64
	orgAccess bool
	events    map[string]map[string]any
	entities  map[string]map[string]any
}

func newHealthStore() *healthStore {
	now := time.Now().UTC()
	eventArn := "arn:aws:health:us-east-1::event/stackyard/STACKYARD_EVENT/stackyard"
	event := map[string]any{
		"arn":               eventArn,
		"service":           "EC2",
		"eventTypeCode":     "AWS_EC2_INSTANCE_RETIREMENT_SCHEDULED",
		"eventTypeCategory": "issue",
		"region":            "us-east-1",
		"availabilityZone":  "us-east-1a",
		"startTime":         now.Add(-1 * time.Hour).Format(time.RFC3339),
		"endTime":           now.Add(24 * time.Hour).Format(time.RFC3339),
		"lastUpdatedTime":   now.Format(time.RFC3339),
		"statusCode":        "open",
		"eventScopeCode":    "ACCOUNT_SPECIFIC",
	}
	entityArn := "arn:aws:ec2:us-east-1:123456789012:instance/i-00000000000000001"
	entity := map[string]any{
		"entityArn":       entityArn,
		"eventArn":        eventArn,
		"entityValue":     "i-00000000000000001",
		"entityUrl":       "https://console.aws.amazon.com/ec2/home",
		"awsAccountId":    "123456789012",
		"lastUpdatedTime": now.Format(time.RFC3339),
		"statusCode":      "IMPAIRED",
		"tags":            map[string]any{"stackyard": "true"},
	}
	return &healthStore{
		nextID:    2,
		orgAccess: true,
		events: map[string]map[string]any{
			eventArn: event,
		},
		entities: map[string]map[string]any{
			entityArn: entity,
		},
	}
}

func (s *healthStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	eventArn := healthPayloadString(payload, "eventArn", "")
	if eventArn == "" {
		eventArn = healthFirstMapKey(s.events)
	}
	event := s.ensureEventLocked(eventArn)

	switch action {
	case "DescribeEvents":
		return map[string]any{
			"events":    healthSortedValues(s.events),
			"nextToken": "",
		}
	case "DescribeEventsForOrganization":
		items := make([]any, 0, len(s.events))
		for _, v := range healthSortedValues(s.events) {
			item, _ := healthCloneAny(v).(map[string]any)
			if item == nil {
				item = map[string]any{}
			}
			item["awsAccountId"] = "123456789012"
			items = append(items, item)
		}
		return map[string]any{"events": items, "nextToken": ""}
	case "DescribeEventTypes":
		return map[string]any{
			"eventTypes": []any{
				map[string]any{"service": "EC2", "code": "AWS_EC2_INSTANCE_RETIREMENT_SCHEDULED", "category": "issue"},
			},
			"nextToken": "",
		}
	case "DescribeEventAggregates":
		return map[string]any{
			"eventAggregates": []any{
				map[string]any{"aggregateValue": "issue", "count": len(s.events)},
			},
			"nextToken": "",
		}
	case "DescribeEntityAggregates":
		return map[string]any{
			"entityAggregates": []any{
				map[string]any{"eventArn": eventArn, "count": len(s.entities)},
			},
		}
	case "DescribeEntityAggregatesForOrganization":
		return map[string]any{
			"organizationEntityAggregates": []any{
				map[string]any{"eventArn": eventArn, "count": len(s.entities)},
			},
		}
	case "DescribeAffectedEntities":
		return map[string]any{
			"entities":  healthSortedValues(s.entities),
			"nextToken": "",
		}
	case "DescribeAffectedEntitiesForOrganization":
		return map[string]any{
			"entities":  healthSortedValues(s.entities),
			"failedSet": []any{},
			"nextToken": "",
		}
	case "DescribeAffectedAccountsForOrganization":
		return map[string]any{
			"affectedAccounts": []any{"123456789012"},
			"eventScopeCode":   healthPayloadString(payload, "eventScopeCode", "ACCOUNT_SPECIFIC"),
			"nextToken":        "",
		}
	case "DescribeEventDetails":
		return map[string]any{
			"successfulSet": []any{
				map[string]any{
					"event":            healthCloneMap(event),
					"eventDescription": map[string]any{"latestDescription": "Stackyard seeded AWS Health event detail."},
				},
			},
			"failedSet": []any{},
		}
	case "DescribeEventDetailsForOrganization":
		return map[string]any{
			"successfulSet": []any{
				map[string]any{
					"awsAccountId":     "123456789012",
					"event":            healthCloneMap(event),
					"eventDescription": map[string]any{"latestDescription": "Stackyard seeded organization event detail."},
				},
			},
			"failedSet": []any{},
		}
	case "DescribeHealthServiceStatusForOrganization":
		status := "DISABLED"
		if s.orgAccess {
			status = "ENABLED"
		}
		return map[string]any{"healthServiceAccessStatusForOrganization": status}
	case "EnableHealthServiceAccessForOrganization":
		s.orgAccess = true
		return map[string]any{}
	case "DisableHealthServiceAccessForOrganization":
		s.orgAccess = false
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *healthStore) ensureEventLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = healthFirstMapKey(s.events)
	}
	if ev := s.events[arn]; ev != nil {
		return ev
	}
	id := s.nextID
	s.nextID++
	now := time.Now().UTC()
	ev := map[string]any{
		"arn":               arn,
		"service":           "EC2",
		"eventTypeCode":     fmt.Sprintf("STACKYARD_EVENT_%06d", id),
		"eventTypeCategory": "issue",
		"region":            "us-east-1",
		"availabilityZone":  "us-east-1a",
		"startTime":         now.Add(-1 * time.Hour).Format(time.RFC3339),
		"endTime":           now.Add(24 * time.Hour).Format(time.RFC3339),
		"lastUpdatedTime":   now.Format(time.RFC3339),
		"statusCode":        "open",
		"eventScopeCode":    "ACCOUNT_SPECIFIC",
	}
	s.events[arn] = ev
	return ev
}

func healthPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" && s != "%!v(<nil>)" {
				return s
			}
		}
	}
	return fallback
}

func healthFirstMapKey(in map[string]map[string]any) string {
	if len(in) == 0 {
		return ""
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func healthSortedValues(in map[string]map[string]any) []any {
	if len(in) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, healthCloneMap(in[key]))
	}
	return out
}

func healthCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = healthCloneAny(value)
	}
	return out
}

func healthCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return healthCloneMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, healthCloneAny(item))
		}
		return out
	default:
		return typed
	}
}
