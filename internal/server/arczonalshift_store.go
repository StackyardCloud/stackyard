package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type arcZonalShiftStore struct {
	mu sync.Mutex

	nextZonalShiftID int64
	nextPracticeID   int64

	observerStatus       string
	managedResources     map[string]map[string]any
	zonalShifts          map[string]map[string]any
	practiceRunConfig    map[string]map[string]any
	practiceRuns         map[string]map[string]any
	autoshifts           map[string]map[string]any
	zonalAutoShiftConfig map[string]map[string]any
}

func newARCZonalShiftStore() *arcZonalShiftStore {
	now := time.Now().UTC()
	resourceID := defaultARCZonalShiftResourceIdentifier()

	resource := map[string]any{
		"resourceIdentifier": resourceID,
		"name":               "stackyard-app-lb",
		"type":               "AWS::ElasticLoadBalancingV2::LoadBalancer",
		"appliedWeights": map[string]any{
			"us-east-1a": 0.5,
			"us-east-1b": 0.5,
		},
		"zonalShifts": []any{},
		"autoshifts": []any{
			map[string]any{
				"awayFrom":      "us-east-1a",
				"startTime":     now.Add(-30 * time.Minute).Format(time.RFC3339),
				"appliedStatus": "APPLIED",
			},
		},
	}

	return &arcZonalShiftStore{
		nextZonalShiftID: 1,
		nextPracticeID:   1,
		observerStatus:   "ENABLED",
		managedResources: map[string]map[string]any{
			resourceID: resource,
		},
		zonalShifts:       map[string]map[string]any{},
		practiceRunConfig: map[string]map[string]any{},
		practiceRuns:      map[string]map[string]any{},
		autoshifts: map[string]map[string]any{
			"autoshift-000001": {
				"awayFrom":           "us-east-1a",
				"resourceIdentifier": resourceID,
				"startTime":          now.Add(-30 * time.Minute).Format(time.RFC3339),
				"endTime":            now.Add(30 * time.Minute).Format(time.RFC3339),
				"status":             "ACTIVE",
			},
		},
		zonalAutoShiftConfig: map[string]map[string]any{
			resourceID: {
				"resourceIdentifier":   resourceID,
				"zonalAutoshiftStatus": "ENABLED",
			},
		},
	}
}

func (s *arcZonalShiftStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	resourceID := arcZonalShiftString(payload, "resourceIdentifier", s.firstResourceIdentifierLocked())
	if resourceID == "" {
		resourceID = defaultARCZonalShiftResourceIdentifier()
	}

	switch action {
	case "ListManagedResources":
		return map[string]any{
			"items":     s.listManagedResourcesLocked(),
			"nextToken": "",
		}

	case "GetManagedResource":
		return map[string]any{
			"managedResource": arcZonalShiftCloneMap(s.ensureManagedResourceLocked(resourceID, now)),
		}

	case "ListAutoshifts":
		return map[string]any{
			"items":     s.listAutoshiftsLocked(),
			"nextToken": "",
		}

	case "ListZonalShifts":
		return map[string]any{
			"items":     s.listZonalShiftsLocked(),
			"nextToken": "",
		}

	case "StartZonalShift":
		awayFrom := arcZonalShiftString(payload, "awayFrom", "us-east-1a")
		comment := arcZonalShiftString(payload, "comment", "started by stackyard")
		shiftID := fmt.Sprintf("zs-%06d", s.nextZonalShiftID)
		s.nextZonalShiftID++
		shift := map[string]any{
			"zonalShiftId":        shiftID,
			"resourceIdentifier":  resourceID,
			"awayFrom":            awayFrom,
			"comment":             comment,
			"startTime":           now.Format(time.RFC3339),
			"expiryTime":          now.Add(2 * time.Hour).Format(time.RFC3339),
			"status":              "ACTIVE",
			"practiceRunOutcome":  "PENDING",
			"zonalShiftARN":       arcZonalShiftARN(shiftID),
			"zonalShiftDirection": "AWAY_FROM",
		}
		s.zonalShifts[shiftID] = shift
		return map[string]any{"zonalShift": arcZonalShiftCloneMap(shift)}

	case "UpdateZonalShift":
		shift := s.resolveZonalShiftLocked(payload, resourceID, now)
		if comment := arcZonalShiftString(payload, "comment", ""); comment != "" {
			shift["comment"] = comment
		}
		if awayFrom := arcZonalShiftString(payload, "awayFrom", ""); awayFrom != "" {
			shift["awayFrom"] = awayFrom
		}
		shift["status"] = arcZonalShiftString(payload, "status", arcZonalShiftString(shift, "status", "ACTIVE"))
		shift["expiryTime"] = now.Add(3 * time.Hour).Format(time.RFC3339)
		return map[string]any{"zonalShift": arcZonalShiftCloneMap(shift)}

	case "CancelZonalShift":
		shift := s.resolveZonalShiftLocked(payload, resourceID, now)
		shift["status"] = "CANCELED"
		shift["cancelledAt"] = now.Format(time.RFC3339)
		return map[string]any{"zonalShift": arcZonalShiftCloneMap(shift)}

	case "CreatePracticeRunConfiguration":
		cfg := map[string]any{
			"resourceIdentifier": resourceID,
			"blockedDates":       []any{},
			"outcomeAlarms": []any{
				map[string]any{"alarmIdentifier": "arn:aws:cloudwatch:us-east-1:123456789012:alarm:stackyard"},
			},
			"crossAccountRoleArn": "arn:aws:iam::123456789012:role/stackyard-arc-practice-run",
		}
		s.practiceRunConfig[resourceID] = cfg
		return map[string]any{"practiceRunConfiguration": arcZonalShiftCloneMap(cfg)}

	case "UpdatePracticeRunConfiguration":
		cfg := s.practiceRunConfig[resourceID]
		if cfg == nil {
			cfg = map[string]any{
				"resourceIdentifier": resourceID,
				"blockedDates":       []any{},
				"outcomeAlarms":      []any{},
			}
			s.practiceRunConfig[resourceID] = cfg
		}
		if blocked, ok := payload["blockedDates"]; ok {
			cfg["blockedDates"] = arcZonalShiftToAnySlice(blocked)
		}
		if alarms, ok := payload["outcomeAlarms"]; ok {
			cfg["outcomeAlarms"] = arcZonalShiftToAnySlice(alarms)
		}
		return map[string]any{"practiceRunConfiguration": arcZonalShiftCloneMap(cfg)}

	case "DeletePracticeRunConfiguration":
		delete(s.practiceRunConfig, resourceID)
		return map[string]any{}

	case "StartPracticeRun":
		prID := fmt.Sprintf("pr-%06d", s.nextPracticeID)
		s.nextPracticeID++
		run := map[string]any{
			"practiceRunId":       prID,
			"resourceIdentifier":  resourceID,
			"awayFrom":            arcZonalShiftString(payload, "awayFrom", "us-east-1a"),
			"startTime":           now.Format(time.RFC3339),
			"expiryTime":          now.Add(1 * time.Hour).Format(time.RFC3339),
			"status":              "ACTIVE",
			"zonalShiftId":        fmt.Sprintf("zs-pr-%06d", s.nextPracticeID),
			"practiceRunArn":      arcPracticeRunARN(prID),
			"practiceRunMetadata": map[string]any{"initiatedBy": "stackyard"},
		}
		s.practiceRuns[prID] = run
		return map[string]any{"practiceRun": arcZonalShiftCloneMap(run)}

	case "CancelPracticeRun":
		run := s.resolvePracticeRunLocked(payload, resourceID, now)
		run["status"] = "CANCELED"
		run["cancelledAt"] = now.Format(time.RFC3339)
		return map[string]any{"practiceRun": arcZonalShiftCloneMap(run)}

	case "UpdateZonalAutoshiftConfiguration":
		cfg := s.zonalAutoShiftConfig[resourceID]
		if cfg == nil {
			cfg = map[string]any{"resourceIdentifier": resourceID}
			s.zonalAutoShiftConfig[resourceID] = cfg
		}
		cfg["zonalAutoshiftStatus"] = arcZonalShiftString(payload, "zonalAutoshiftStatus", "ENABLED")
		return map[string]any{"zonalAutoshiftConfiguration": arcZonalShiftCloneMap(cfg)}

	case "GetAutoshiftObserverNotificationStatus":
		return map[string]any{"status": s.observerStatus}

	case "UpdateAutoshiftObserverNotificationStatus":
		s.observerStatus = arcZonalShiftString(payload, "status", s.observerStatus)
		if s.observerStatus == "" {
			s.observerStatus = "ENABLED"
		}
		return map[string]any{"status": s.observerStatus}
	}

	return map[string]any{}
}

func (s *arcZonalShiftStore) firstResourceIdentifierLocked() string {
	keys := make([]string, 0, len(s.managedResources))
	for key := range s.managedResources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func (s *arcZonalShiftStore) ensureManagedResourceLocked(resourceID string, now time.Time) map[string]any {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		resourceID = defaultARCZonalShiftResourceIdentifier()
	}
	if current, ok := s.managedResources[resourceID]; ok {
		return current
	}
	resource := map[string]any{
		"resourceIdentifier": resourceID,
		"name":               "stackyard-managed-resource",
		"type":               "AWS::ElasticLoadBalancingV2::LoadBalancer",
		"appliedWeights": map[string]any{
			"us-east-1a": 1.0,
		},
		"zonalShifts": []any{},
		"autoshifts": []any{
			map[string]any{
				"awayFrom":      "us-east-1a",
				"startTime":     now.Add(-5 * time.Minute).Format(time.RFC3339),
				"appliedStatus": "APPLIED",
			},
		},
	}
	s.managedResources[resourceID] = resource
	return resource
}

func (s *arcZonalShiftStore) listManagedResourcesLocked() []any {
	keys := make([]string, 0, len(s.managedResources))
	for key := range s.managedResources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, arcZonalShiftCloneMap(s.managedResources[key]))
	}
	return out
}

func (s *arcZonalShiftStore) listAutoshiftsLocked() []any {
	keys := make([]string, 0, len(s.autoshifts))
	for key := range s.autoshifts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, arcZonalShiftCloneMap(s.autoshifts[key]))
	}
	return out
}

func (s *arcZonalShiftStore) listZonalShiftsLocked() []any {
	keys := make([]string, 0, len(s.zonalShifts))
	for key := range s.zonalShifts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, arcZonalShiftCloneMap(s.zonalShifts[key]))
	}
	return out
}

func (s *arcZonalShiftStore) resolveZonalShiftLocked(payload map[string]any, resourceID string, now time.Time) map[string]any {
	zonalShiftID := arcZonalShiftString(payload, "zonalShiftId", "")
	if zonalShiftID != "" {
		if shift := s.zonalShifts[zonalShiftID]; shift != nil {
			return shift
		}
	}
	if len(s.zonalShifts) > 0 {
		keys := make([]string, 0, len(s.zonalShifts))
		for key := range s.zonalShifts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return s.zonalShifts[keys[len(keys)-1]]
	}

	shiftID := fmt.Sprintf("zs-%06d", s.nextZonalShiftID)
	s.nextZonalShiftID++
	shift := map[string]any{
		"zonalShiftId":       shiftID,
		"resourceIdentifier": resourceID,
		"awayFrom":           "us-east-1a",
		"comment":            "auto-created shift",
		"startTime":          now.Format(time.RFC3339),
		"expiryTime":         now.Add(1 * time.Hour).Format(time.RFC3339),
		"status":             "ACTIVE",
		"zonalShiftARN":      arcZonalShiftARN(shiftID),
	}
	s.zonalShifts[shiftID] = shift
	return shift
}

func (s *arcZonalShiftStore) resolvePracticeRunLocked(payload map[string]any, resourceID string, now time.Time) map[string]any {
	practiceRunID := arcZonalShiftString(payload, "practiceRunId", "")
	if practiceRunID != "" {
		if run := s.practiceRuns[practiceRunID]; run != nil {
			return run
		}
	}
	if len(s.practiceRuns) > 0 {
		keys := make([]string, 0, len(s.practiceRuns))
		for key := range s.practiceRuns {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return s.practiceRuns[keys[len(keys)-1]]
	}

	prID := fmt.Sprintf("pr-%06d", s.nextPracticeID)
	s.nextPracticeID++
	run := map[string]any{
		"practiceRunId":      prID,
		"resourceIdentifier": resourceID,
		"awayFrom":           "us-east-1a",
		"startTime":          now.Format(time.RFC3339),
		"expiryTime":         now.Add(1 * time.Hour).Format(time.RFC3339),
		"status":             "ACTIVE",
		"practiceRunArn":     arcPracticeRunARN(prID),
	}
	s.practiceRuns[prID] = run
	return run
}

func defaultARCZonalShiftResourceIdentifier() string {
	return "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/stackyard/50dc6c495c0c9188"
}

func arcZonalShiftARN(zonalShiftID string) string {
	return fmt.Sprintf("arn:aws:arc-zonal-shift:us-east-1:123456789012:zonal-shift/%s", strings.TrimSpace(zonalShiftID))
}

func arcPracticeRunARN(practiceRunID string) string {
	return fmt.Sprintf("arn:aws:arc-zonal-shift:us-east-1:123456789012:practice-run/%s", strings.TrimSpace(practiceRunID))
}

func arcZonalShiftString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return def
	}
	str, ok := raw.(string)
	if !ok {
		return def
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return def
	}
	return str
}

func arcZonalShiftCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = arcZonalShiftCloneMap(typed)
		case []any:
			out[key] = arcZonalShiftCloneSlice(typed)
		default:
			out[key] = typed
		}
	}
	return out
}

func arcZonalShiftCloneSlice(in []any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		switch typed := item.(type) {
		case map[string]any:
			out = append(out, arcZonalShiftCloneMap(typed))
		case []any:
			out = append(out, arcZonalShiftCloneSlice(typed))
		default:
			out = append(out, typed)
		}
	}
	return out
}

func arcZonalShiftToAnySlice(raw any) []any {
	if raw == nil {
		return []any{}
	}
	list, ok := raw.([]any)
	if ok {
		out := make([]any, 0, len(list))
		out = append(out, list...)
		return out
	}
	if typed, ok := raw.([]string); ok {
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	}
	return []any{}
}
