package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type controlTowerStore struct {
	mu sync.Mutex

	next int64

	landingZones          map[string]map[string]any
	landingZoneOperations map[string]map[string]any

	baselines          map[string]map[string]any
	enabledBaselines   map[string]map[string]any
	baselineOperations map[string]map[string]any

	enabledControls   map[string]map[string]any
	controlOperations map[string]map[string]any

	tags map[string]map[string]string
}

func newControlTowerStore() *controlTowerStore {
	s := &controlTowerStore{
		next:                  2,
		landingZones:          map[string]map[string]any{},
		landingZoneOperations: map[string]map[string]any{},
		baselines:             map[string]map[string]any{},
		enabledBaselines:      map[string]map[string]any{},
		baselineOperations:    map[string]map[string]any{},
		enabledControls:       map[string]map[string]any{},
		controlOperations:     map[string]map[string]any{},
		tags:                  map[string]map[string]string{},
	}

	lzARN := "arn:aws:controltower:us-east-1:123456789012:landingzone/lz-000001"
	s.landingZones[lzARN] = map[string]any{
		"arn":                    lzARN,
		"driftStatus":            map[string]any{"status": "IN_SYNC"},
		"latestAvailableVersion": "3.2",
		"manifest":               map[string]any{"governedRegions": []any{"us-east-1"}},
		"status":                 "ACTIVE",
		"version":                "3.2",
	}

	baselineARN := "arn:aws:controltower:us-east-1::baseline/aws-baseline/default"
	s.baselines[baselineARN] = map[string]any{
		"arn":         baselineARN,
		"name":        "AWSControlTowerBaseline",
		"description": "Stackyard default baseline",
	}

	enabledBaselineARN := "arn:aws:controltower:us-east-1:123456789012:enabledbaseline/ebl-000001"
	s.enabledBaselines[enabledBaselineARN] = map[string]any{
		"arn":                enabledBaselineARN,
		"baselineIdentifier": baselineARN,
		"baselineVersion":    "1.0",
		"driftStatusSummary": ctEnabledBaselineDriftStatusSummary(),
		"parameters":         []any{},
		"statusSummary":      ctEnablementStatusSummary(""),
		"targetIdentifier":   "ou-0000-example",
	}

	enabledControlARN := "arn:aws:controltower:us-east-1:123456789012:enabledcontrol/ec-000001"
	s.enabledControls[enabledControlARN] = map[string]any{
		"arn":                enabledControlARN,
		"controlIdentifier":  ctDefaultControlIdentifier(),
		"driftStatusSummary": ctEnabledControlDriftStatusSummary(),
		"parameters":         []any{},
		"statusSummary":      ctEnablementStatusSummary(""),
		"targetIdentifier":   "ou-0000-example",
		"targetRegions":      ctDefaultTargetRegions(),
	}

	s.tags[lzARN] = map[string]string{"stackyard": "true"}
	s.tags[baselineARN] = map[string]string{"stackyard": "true"}
	s.tags[enabledBaselineARN] = map[string]string{"stackyard": "true"}
	s.tags[enabledControlARN] = map[string]string{"stackyard": "true"}
	return s
}

func (s *controlTowerStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateLandingZone":
		lzARN := ctLandingZoneIdentifier(payload, "landingZoneIdentifier")
		if lzARN == "" {
			lzARN = ctLandingZoneARN(fmt.Sprintf("lz-%06d", s.nextIDLocked()))
		}
		s.ensureLandingZoneLocked(lzARN)
		op := s.newLandingZoneOperationLocked("CREATE")
		return map[string]any{
			"arn":                 lzARN,
			"operationIdentifier": ctString(op["operationIdentifier"], ""),
		}
	case "DeleteLandingZone":
		lzARN := ctLandingZoneIdentifier(payload, "landingZoneIdentifier")
		if lzARN == "" {
			lzARN = ctFirstLandingZoneARN(s.landingZones)
		}
		delete(s.landingZones, lzARN)
		op := s.newLandingZoneOperationLocked("DELETE")
		return map[string]any{"operationIdentifier": ctString(op["operationIdentifier"], "")}
	case "UpdateLandingZone":
		lzARN := ctLandingZoneIdentifier(payload, "landingZoneIdentifier")
		if lzARN == "" {
			lzARN = ctFirstLandingZoneARN(s.landingZones)
		}
		lz := s.ensureLandingZoneLocked(lzARN)
		if version := ctStringAny(payload, "version", ""); version != "" {
			lz["version"] = version
		}
		if manifest := ctMapAny(payload, "manifest"); len(manifest) > 0 {
			lz["manifest"] = manifest
		}
		op := s.newLandingZoneOperationLocked("UPDATE")
		return map[string]any{"operationIdentifier": ctString(op["operationIdentifier"], "")}
	case "ResetLandingZone":
		lzARN := ctLandingZoneIdentifier(payload, "landingZoneIdentifier")
		if lzARN == "" {
			lzARN = ctFirstLandingZoneARN(s.landingZones)
		}
		lz := s.ensureLandingZoneLocked(lzARN)
		lz["driftStatus"] = map[string]any{"status": "IN_SYNC"}
		op := s.newLandingZoneOperationLocked("RESET")
		return map[string]any{"operationIdentifier": ctString(op["operationIdentifier"], "")}
	case "GetLandingZone":
		lzARN := ctLandingZoneIdentifier(payload, "landingZoneIdentifier")
		if lzARN == "" {
			lzARN = ctFirstLandingZoneARN(s.landingZones)
		}
		return map[string]any{"landingZone": ctCloneMap(s.ensureLandingZoneLocked(lzARN))}
	case "GetLandingZoneOperation":
		id := ctStringAny(payload, "operationIdentifier", "")
		if id == "" {
			id = ctFirstMapKey(s.landingZoneOperations)
		}
		op := s.ensureLandingZoneOperationLocked(id)
		return map[string]any{"operationDetails": ctCloneMap(op)}
	case "ListLandingZones":
		summaries := make([]any, 0, len(s.landingZones))
		for _, lz := range ctSortedMapValues(s.landingZones) {
			summaries = append(summaries, map[string]any{"arn": ctString(lz["arn"], "")})
		}
		return map[string]any{"landingZones": summaries, "nextToken": ""}
	case "ListLandingZoneOperations":
		ops := make([]any, 0, len(s.landingZoneOperations))
		for _, op := range ctSortedMapValues(s.landingZoneOperations) {
			ops = append(ops, map[string]any{
				"operationIdentifier": ctString(op["operationIdentifier"], ""),
				"operationType":       ctString(op["operationType"], "UPDATE"),
				"status":              ctString(op["status"], "SUCCEEDED"),
			})
		}
		return map[string]any{"landingZoneOperations": ops, "nextToken": ""}

	case "GetBaseline":
		baselineARN := ctBaselineIdentifier(payload, "baselineIdentifier")
		if baselineARN == "" {
			baselineARN = ctFirstMapKey(s.baselines)
		}
		return ctCloneMap(s.ensureBaselineLocked(baselineARN))
	case "ListBaselines":
		items := make([]any, 0, len(s.baselines))
		for _, baseline := range ctSortedMapValues(s.baselines) {
			items = append(items, map[string]any{
				"arn":         ctString(baseline["arn"], ""),
				"name":        ctString(baseline["name"], ""),
				"description": ctString(baseline["description"], ""),
			})
		}
		return map[string]any{"baselines": items, "nextToken": ""}
	case "EnableBaseline":
		baselineARN := ctBaselineIdentifier(payload, "baselineIdentifier")
		if baselineARN == "" {
			baselineARN = ctFirstMapKey(s.baselines)
		}
		target := ctStringAny(payload, "targetIdentifier", "ou-0000-example")
		enabledARN := ctEnabledBaselineARN(fmt.Sprintf("ebl-%06d", s.nextIDLocked()))
		s.enabledBaselines[enabledARN] = map[string]any{
			"arn":                enabledARN,
			"baselineIdentifier": baselineARN,
			"baselineVersion":    "1.0",
			"driftStatusSummary": ctEnabledBaselineDriftStatusSummary(),
			"parameters":         ctSliceAny(payload, "parameters"),
			"targetIdentifier":   target,
		}
		op := s.newBaselineOperationLocked("ENABLE", enabledARN)
		s.enabledBaselines[enabledARN]["statusSummary"] = ctEnablementStatusSummary(ctString(op["operationIdentifier"], ""))
		return map[string]any{
			"arn":                 enabledARN,
			"operationIdentifier": ctString(op["operationIdentifier"], ""),
		}
	case "DisableBaseline":
		enabledARN := ctEnabledBaselineIdentifier(payload, "enabledBaselineIdentifier")
		if enabledARN == "" {
			enabledARN = ctFirstMapKey(s.enabledBaselines)
		}
		delete(s.enabledBaselines, enabledARN)
		op := s.newBaselineOperationLocked("DISABLE", enabledARN)
		return map[string]any{"operationIdentifier": ctString(op["operationIdentifier"], "")}
	case "ResetEnabledBaseline":
		enabledARN := ctEnabledBaselineIdentifier(payload, "enabledBaselineIdentifier")
		if enabledARN == "" {
			enabledARN = ctFirstMapKey(s.enabledBaselines)
		}
		enabled := s.ensureEnabledBaselineLocked(enabledARN)
		enabled["driftStatusSummary"] = ctEnabledBaselineDriftStatusSummary()
		op := s.newBaselineOperationLocked("RESET", enabledARN)
		enabled["statusSummary"] = ctEnablementStatusSummary(ctString(op["operationIdentifier"], ""))
		return map[string]any{"operationIdentifier": ctString(op["operationIdentifier"], "")}
	case "UpdateEnabledBaseline":
		enabledARN := ctEnabledBaselineIdentifier(payload, "enabledBaselineIdentifier")
		if enabledARN == "" {
			enabledARN = ctFirstMapKey(s.enabledBaselines)
		}
		enabled := s.ensureEnabledBaselineLocked(enabledARN)
		if params := ctSliceAny(payload, "parameters"); len(params) > 0 {
			enabled["parameters"] = params
		}
		op := s.newBaselineOperationLocked("UPDATE", enabledARN)
		enabled["statusSummary"] = ctEnablementStatusSummary(ctString(op["operationIdentifier"], ""))
		return map[string]any{"operationIdentifier": ctString(op["operationIdentifier"], "")}
	case "GetEnabledBaseline":
		enabledARN := ctEnabledBaselineIdentifier(payload, "enabledBaselineIdentifier")
		if enabledARN == "" {
			enabledARN = ctFirstMapKey(s.enabledBaselines)
		}
		enabled := ctCloneMap(s.ensureEnabledBaselineLocked(enabledARN))
		return map[string]any{"enabledBaselineDetails": enabled}
	case "ListEnabledBaselines":
		items := make([]any, 0, len(s.enabledBaselines))
		for _, eb := range ctSortedMapValues(s.enabledBaselines) {
			items = append(items, map[string]any{
				"arn":                ctString(eb["arn"], ""),
				"baselineIdentifier": ctString(eb["baselineIdentifier"], ""),
				"targetIdentifier":   ctString(eb["targetIdentifier"], ""),
				"statusSummary":      ctCloneAny(eb["statusSummary"]),
			})
		}
		return map[string]any{"enabledBaselines": items, "nextToken": ""}
	case "GetBaselineOperation":
		opID := ctStringAny(payload, "operationIdentifier", "")
		if opID == "" {
			opID = ctFirstMapKey(s.baselineOperations)
		}
		return map[string]any{"baselineOperation": ctCloneMap(s.ensureBaselineOperationLocked(opID))}

	case "EnableControl":
		controlIdentifier := ctStringAny(payload, "controlIdentifier", ctDefaultControlIdentifier())
		target := ctStringAny(payload, "targetIdentifier", "ou-0000-example")
		enabledARN := ctEnabledControlARN(fmt.Sprintf("ec-%06d", s.nextIDLocked()))
		s.enabledControls[enabledARN] = map[string]any{
			"arn":                enabledARN,
			"controlIdentifier":  controlIdentifier,
			"driftStatusSummary": ctEnabledControlDriftStatusSummary(),
			"parameters":         ctSliceAny(payload, "parameters"),
			"targetIdentifier":   target,
			"targetRegions":      ctDefaultTargetRegions(),
		}
		op := s.newControlOperationLocked("ENABLE", enabledARN)
		s.enabledControls[enabledARN]["statusSummary"] = ctEnablementStatusSummary(ctString(op["operationIdentifier"], ""))
		return map[string]any{
			"arn":                 enabledARN,
			"operationIdentifier": ctString(op["operationIdentifier"], ""),
		}
	case "DisableControl":
		enabledARN := ctEnabledControlIdentifier(payload, "enabledControlIdentifier")
		if enabledARN == "" {
			enabledARN = ctFirstMapKey(s.enabledControls)
		}
		delete(s.enabledControls, enabledARN)
		op := s.newControlOperationLocked("DISABLE", enabledARN)
		return map[string]any{"operationIdentifier": ctString(op["operationIdentifier"], "")}
	case "ResetEnabledControl":
		enabledARN := ctEnabledControlIdentifier(payload, "enabledControlIdentifier")
		if enabledARN == "" {
			enabledARN = ctFirstMapKey(s.enabledControls)
		}
		enabled := s.ensureEnabledControlLocked(enabledARN)
		enabled["driftStatusSummary"] = ctEnabledControlDriftStatusSummary()
		op := s.newControlOperationLocked("RESET", enabledARN)
		enabled["statusSummary"] = ctEnablementStatusSummary(ctString(op["operationIdentifier"], ""))
		return map[string]any{"operationIdentifier": ctString(op["operationIdentifier"], "")}
	case "UpdateEnabledControl":
		enabledARN := ctEnabledControlIdentifier(payload, "enabledControlIdentifier")
		if enabledARN == "" {
			enabledARN = ctFirstMapKey(s.enabledControls)
		}
		enabled := s.ensureEnabledControlLocked(enabledARN)
		if params := ctSliceAny(payload, "parameters"); len(params) > 0 {
			enabled["parameters"] = params
		}
		op := s.newControlOperationLocked("UPDATE", enabledARN)
		enabled["statusSummary"] = ctEnablementStatusSummary(ctString(op["operationIdentifier"], ""))
		return map[string]any{"operationIdentifier": ctString(op["operationIdentifier"], "")}
	case "GetEnabledControl":
		enabledARN := ctEnabledControlIdentifier(payload, "enabledControlIdentifier")
		if enabledARN == "" {
			enabledARN = ctFirstMapKey(s.enabledControls)
		}
		enabled := ctCloneMap(s.ensureEnabledControlLocked(enabledARN))
		return map[string]any{"enabledControlDetails": enabled}
	case "ListEnabledControls":
		items := make([]any, 0, len(s.enabledControls))
		for _, ec := range ctSortedMapValues(s.enabledControls) {
			items = append(items, map[string]any{
				"arn":               ctString(ec["arn"], ""),
				"controlIdentifier": ctString(ec["controlIdentifier"], ""),
				"targetIdentifier":  ctString(ec["targetIdentifier"], ""),
				"statusSummary":     ctCloneAny(ec["statusSummary"]),
			})
		}
		return map[string]any{"enabledControls": items, "nextToken": ""}
	case "GetControlOperation":
		opID := ctStringAny(payload, "operationIdentifier", "")
		if opID == "" {
			opID = ctFirstMapKey(s.controlOperations)
		}
		return map[string]any{"controlOperation": ctCloneMap(s.ensureControlOperationLocked(opID))}
	case "ListControlOperations":
		items := make([]any, 0, len(s.controlOperations))
		for _, op := range ctSortedMapValues(s.controlOperations) {
			items = append(items, map[string]any{
				"operationIdentifier": ctString(op["operationIdentifier"], ""),
				"operationType":       ctString(op["operationType"], "UPDATE"),
				"status":              ctString(op["status"], "SUCCEEDED"),
			})
		}
		return map[string]any{"controlOperations": items, "nextToken": ""}

	case "TagResource":
		resourceARN := ctResourceARN(payload, pathParams)
		s.ensureTagsLocked(resourceARN)
		for key, value := range ctMapString(payload, "tags") {
			s.tags[resourceARN][key] = value
		}
		return map[string]any{}
	case "UntagResource":
		resourceARN := ctResourceARN(payload, pathParams)
		s.ensureTagsLocked(resourceARN)
		for _, key := range ctStringSlice(payload, "tagKeys") {
			delete(s.tags[resourceARN], key)
		}
		for _, key := range query["tagKeys"] {
			key = strings.TrimSpace(key)
			if key != "" {
				delete(s.tags[resourceARN], key)
			}
		}
		return map[string]any{}
	case "ListTagsForResource":
		resourceARN := ctResourceARN(payload, pathParams)
		s.ensureTagsLocked(resourceARN)
		return map[string]any{"tags": ctCloneStringMap(s.tags[resourceARN])}
	}

	return map[string]any{}
}

func (s *controlTowerStore) ensureLandingZoneLocked(arn string) map[string]any {
	normalized := ctLandingZoneIdentifier(map[string]any{"landingZoneIdentifier": arn}, "landingZoneIdentifier")
	if normalized == "" {
		normalized = ctLandingZoneARN("lz-000001")
	}
	if existing := s.landingZones[normalized]; existing != nil {
		return existing
	}
	out := map[string]any{
		"arn":                    normalized,
		"driftStatus":            map[string]any{"status": "IN_SYNC"},
		"latestAvailableVersion": "3.2",
		"manifest": map[string]any{
			"governedRegions": []any{"us-east-1"},
		},
		"status":  "ACTIVE",
		"version": "3.2",
	}
	s.landingZones[normalized] = out
	return out
}

func (s *controlTowerStore) ensureBaselineLocked(arn string) map[string]any {
	normalized := ctBaselineIdentifier(map[string]any{"baselineIdentifier": arn}, "baselineIdentifier")
	if normalized == "" {
		normalized = "arn:aws:controltower:us-east-1::baseline/aws-baseline/default"
	}
	if existing := s.baselines[normalized]; existing != nil {
		return existing
	}
	out := map[string]any{
		"arn":         normalized,
		"name":        "StackyardBaseline",
		"description": "Stackyard baseline",
	}
	s.baselines[normalized] = out
	return out
}

func (s *controlTowerStore) ensureEnabledBaselineLocked(arn string) map[string]any {
	normalized := ctEnabledBaselineIdentifier(map[string]any{"enabledBaselineIdentifier": arn}, "enabledBaselineIdentifier")
	if normalized == "" {
		normalized = ctEnabledBaselineARN("ebl-000001")
	}
	if existing := s.enabledBaselines[normalized]; existing != nil {
		return existing
	}
	out := map[string]any{
		"arn":                normalized,
		"baselineIdentifier": ctFirstMapKey(s.baselines),
		"baselineVersion":    "1.0",
		"driftStatusSummary": ctEnabledBaselineDriftStatusSummary(),
		"parameters":         []any{},
		"statusSummary":      ctEnablementStatusSummary(""),
		"targetIdentifier":   "ou-0000-example",
	}
	s.enabledBaselines[normalized] = out
	return out
}

func (s *controlTowerStore) ensureEnabledControlLocked(arn string) map[string]any {
	normalized := ctEnabledControlIdentifier(map[string]any{"enabledControlIdentifier": arn}, "enabledControlIdentifier")
	if normalized == "" {
		normalized = ctEnabledControlARN("ec-000001")
	}
	if existing := s.enabledControls[normalized]; existing != nil {
		return existing
	}
	out := map[string]any{
		"arn":                normalized,
		"controlIdentifier":  ctDefaultControlIdentifier(),
		"driftStatusSummary": ctEnabledControlDriftStatusSummary(),
		"parameters":         []any{},
		"statusSummary":      ctEnablementStatusSummary(""),
		"targetIdentifier":   "ou-0000-example",
		"targetRegions":      ctDefaultTargetRegions(),
	}
	s.enabledControls[normalized] = out
	return out
}

func (s *controlTowerStore) ensureLandingZoneOperationLocked(id string) map[string]any {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		normalized = fmt.Sprintf("lzop-%06d", s.nextIDLocked())
	}
	if existing := s.landingZoneOperations[normalized]; existing != nil {
		return existing
	}
	now := time.Now().UTC()
	out := map[string]any{
		"operationIdentifier": normalized,
		"operationType":       "UPDATE_ENABLED_BASELINE",
		"status":              "SUCCEEDED",
		"startTime":           now.Format(time.RFC3339),
		"endTime":             now.Format(time.RFC3339),
	}
	s.landingZoneOperations[normalized] = out
	return out
}

func (s *controlTowerStore) ensureBaselineOperationLocked(id string) map[string]any {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		normalized = fmt.Sprintf("bop-%06d", s.nextIDLocked())
	}
	if existing := s.baselineOperations[normalized]; existing != nil {
		return existing
	}
	now := time.Now().UTC()
	out := map[string]any{
		"operationIdentifier": normalized,
		"operationType":       "UPDATE_ENABLED_CONTROL",
		"status":              "SUCCEEDED",
		"startTime":           now.Format(time.RFC3339),
		"endTime":             now.Format(time.RFC3339),
	}
	s.baselineOperations[normalized] = out
	return out
}

func (s *controlTowerStore) ensureControlOperationLocked(id string) map[string]any {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		normalized = fmt.Sprintf("cop-%06d", s.nextIDLocked())
	}
	if existing := s.controlOperations[normalized]; existing != nil {
		return existing
	}
	now := time.Now().UTC()
	out := map[string]any{
		"operationIdentifier": normalized,
		"operationType":       "UPDATE",
		"status":              "SUCCEEDED",
		"startTime":           now.Format(time.RFC3339),
		"endTime":             now.Format(time.RFC3339),
	}
	s.controlOperations[normalized] = out
	return out
}

func (s *controlTowerStore) newLandingZoneOperationLocked(opType string) map[string]any {
	id := fmt.Sprintf("lzop-%06d", s.nextIDLocked())
	now := time.Now().UTC()
	op := map[string]any{
		"operationIdentifier": id,
		"operationType":       strings.ToUpper(strings.TrimSpace(opType)),
		"status":              "SUCCEEDED",
		"startTime":           now.Format(time.RFC3339),
		"endTime":             now.Format(time.RFC3339),
	}
	s.landingZoneOperations[id] = op
	return op
}

func (s *controlTowerStore) newBaselineOperationLocked(opType, _ string) map[string]any {
	id := fmt.Sprintf("bop-%06d", s.nextIDLocked())
	now := time.Now().UTC()
	op := map[string]any{
		"operationIdentifier": id,
		"operationType":       ctBaselineOperationType(opType),
		"status":              "SUCCEEDED",
		"startTime":           now.Format(time.RFC3339),
		"endTime":             now.Format(time.RFC3339),
	}
	s.baselineOperations[id] = op
	return op
}

func (s *controlTowerStore) newControlOperationLocked(opType, _ string) map[string]any {
	id := fmt.Sprintf("cop-%06d", s.nextIDLocked())
	now := time.Now().UTC()
	op := map[string]any{
		"operationIdentifier": id,
		"operationType":       ctControlOperationType(opType),
		"status":              "SUCCEEDED",
		"startTime":           now.Format(time.RFC3339),
		"endTime":             now.Format(time.RFC3339),
	}
	s.controlOperations[id] = op
	return op
}

func (s *controlTowerStore) ensureTagsLocked(arn string) {
	if strings.TrimSpace(arn) == "" {
		return
	}
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{}
	}
}

func (s *controlTowerStore) nextIDLocked() int64 {
	id := s.next
	s.next++
	return id
}

func ctDefaultControlIdentifier() string {
	return "arn:aws:controltower:us-east-1::control/AWS-GR_ENCRYPTED_VOLUMES"
}

func ctDefaultTargetRegions() []any {
	return []any{map[string]any{"name": "us-east-1"}}
}

func ctEnablementStatusSummary(operationID string) map[string]any {
	out := map[string]any{"status": "SUCCEEDED"}
	if strings.TrimSpace(operationID) != "" {
		out["lastOperationIdentifier"] = strings.TrimSpace(operationID)
	}
	return out
}

func ctEnabledBaselineDriftStatusSummary() map[string]any {
	return map[string]any{
		"types": map[string]any{
			"inheritance": map[string]any{"status": "IN_SYNC"},
		},
	}
}

func ctEnabledControlDriftStatusSummary() map[string]any {
	return map[string]any{
		"driftStatus": "IN_SYNC",
	}
}

func ctBaselineOperationType(opType string) string {
	switch strings.ToUpper(strings.TrimSpace(opType)) {
	case "ENABLE":
		return "ENABLE_BASELINE"
	case "DISABLE":
		return "DISABLE_BASELINE"
	case "RESET":
		return "RESET_ENABLED_BASELINE"
	default:
		return "UPDATE_ENABLED_BASELINE"
	}
}

func ctControlOperationType(opType string) string {
	switch strings.ToUpper(strings.TrimSpace(opType)) {
	case "ENABLE":
		return "ENABLE_CONTROL"
	case "DISABLE":
		return "DISABLE_CONTROL"
	case "RESET":
		return "RESET_ENABLED_CONTROL"
	default:
		return "UPDATE_ENABLED_CONTROL"
	}
}

func ctLandingZoneIdentifier(payload map[string]any, key string) string {
	raw := ctStringAny(payload, key, "")
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "arn:") {
		return raw
	}
	return ctLandingZoneARN(raw)
}

func ctBaselineIdentifier(payload map[string]any, key string) string {
	raw := ctStringAny(payload, key, "")
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "arn:") {
		return raw
	}
	return fmt.Sprintf("arn:aws:controltower:us-east-1::baseline/%s", strings.TrimSpace(raw))
}

func ctEnabledBaselineIdentifier(payload map[string]any, key string) string {
	raw := ctStringAny(payload, key, "")
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "arn:") {
		return raw
	}
	return ctEnabledBaselineARN(raw)
}

func ctEnabledControlIdentifier(payload map[string]any, key string) string {
	raw := ctStringAny(payload, key, "")
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "arn:") {
		return raw
	}
	return ctEnabledControlARN(raw)
}

func ctLandingZoneARN(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "lz-000001"
	}
	return fmt.Sprintf("arn:aws:controltower:us-east-1:123456789012:landingzone/%s", id)
}

func ctEnabledBaselineARN(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "ebl-000001"
	}
	return fmt.Sprintf("arn:aws:controltower:us-east-1:123456789012:enabledbaseline/%s", id)
}

func ctEnabledControlARN(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "ec-000001"
	}
	return fmt.Sprintf("arn:aws:controltower:us-east-1:123456789012:enabledcontrol/%s", id)
}

func ctResourceARN(payload map[string]any, pathParams map[string]string) string {
	if value := ctString(pathParams["resourceArn"], ""); value != "" {
		return value
	}
	if value := ctStringAny(payload, "resourceArn", ""); value != "" {
		return value
	}
	return ctFirstMapKey(pathParams)
}

func ctStringAny(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if out := strings.TrimSpace(ctString(v, "")); out != "" {
				return out
			}
		}
	}
	return fallback
}

func ctMapAny(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		if cast, ok := v.(map[string]any); ok {
			return cast
		}
	}
	return map[string]any{}
}

func ctSliceAny(payload map[string]any, key string) []any {
	if payload == nil {
		return nil
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		if cast, ok := v.([]any); ok {
			return cast
		}
	}
	return nil
}

func ctMapString(payload map[string]any, key string) map[string]string {
	raw := ctMapAny(payload, key)
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		tagKey := strings.TrimSpace(k)
		if tagKey == "" {
			continue
		}
		out[tagKey] = strings.TrimSpace(ctString(v, ""))
	}
	return out
}

func ctStringSlice(payload map[string]any, key string) []string {
	raw := ctSliceAny(payload, key)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		value := strings.TrimSpace(ctString(item, ""))
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func ctString(v any, fallback string) string {
	switch t := v.(type) {
	case string:
		if value := strings.TrimSpace(t); value != "" {
			return value
		}
	case fmt.Stringer:
		if value := strings.TrimSpace(t.String()); value != "" {
			return value
		}
	default:
		if v != nil {
			if value := strings.TrimSpace(fmt.Sprintf("%v", v)); value != "" {
				return value
			}
		}
	}
	return fallback
}

func ctFirstMapKey[V any](in map[string]V) string {
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

func ctFirstLandingZoneARN(in map[string]map[string]any) string {
	if len(in) == 0 {
		return ctLandingZoneARN("lz-000001")
	}
	return ctFirstMapKey(in)
}

func ctSortedMapValues(in map[string]map[string]any) []map[string]any {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, in[key])
	}
	return out
}

func ctCloneAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return ctCloneMap(t)
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, ctCloneAny(item))
		}
		return out
	case map[string]string:
		return ctCloneStringMap(t)
	default:
		return t
	}
}

func ctCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = ctCloneAny(v)
	}
	return out
}

func ctCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
