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
	tnbDefaultRegion    = "us-east-1"
	tnbDefaultAccountID = "123456789012"
)

type tnbStore struct {
	mu sync.Mutex

	nextNsdID     int64
	nextVnfPkgID  int64
	nextNsInstID  int64
	nextNsLcmOpID int64
	nextVnfInstID int64

	networkPackages   map[string]map[string]any
	functionPackages  map[string]map[string]any
	networkInstances  map[string]map[string]any
	functionInstances map[string]map[string]any
	operations        map[string]map[string]any
	tags              map[string]map[string]string
}

func newTNBStore() *tnbStore {
	s := &tnbStore{
		nextNsdID:         2,
		nextVnfPkgID:      2,
		nextNsInstID:      2,
		nextNsLcmOpID:     2,
		nextVnfInstID:     2,
		networkPackages:   map[string]map[string]any{},
		functionPackages:  map[string]map[string]any{},
		networkInstances:  map[string]map[string]any{},
		functionInstances: map[string]map[string]any{},
		operations:        map[string]map[string]any{},
		tags:              map[string]map[string]string{},
	}

	now := time.Now().UTC()
	s.ensureNetworkPackageLocked("nsd-000001", now)
	s.ensureFunctionPackageLocked("vnfpkg-000001", now)
	s.ensureNetworkInstanceLocked("ns-instance-000001", now)
	s.ensureFunctionInstanceLocked("vnf-instance-000001", now)
	s.ensureOperationLocked("ns-lcm-op-000001", "ns-instance-000001", "INSTANTIATE", now)
	s.ensureTagsLocked("arn:aws:tnb:us-east-1:123456789012:nsd/nsd-000001")
	return s
}

func (s *tnbStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ctx := tnbMergeMaps(payload, pathParams, query)

	nsdInfoID := tnbString(ctx, []string{"nsdInfoId"}, "nsd-000001")
	vnfPkgID := tnbString(ctx, []string{"vnfPkgId"}, "vnfpkg-000001")
	nsInstanceID := tnbString(ctx, []string{"nsInstanceId"}, "ns-instance-000001")
	vnfInstanceID := tnbString(ctx, []string{"vnfInstanceId"}, "vnf-instance-000001")
	nsLcmOpOccID := tnbString(ctx, []string{"nsLcmOpOccId"}, "ns-lcm-op-000001")
	resourceARN := tnbString(ctx, []string{"resourceArn"}, "arn:aws:tnb:us-east-1:123456789012:nsd/nsd-000001")

	switch action {
	case "CreateSolNetworkPackage":
		id := fmt.Sprintf("nsd-%06d", s.nextNsdID)
		s.nextNsdID++
		pkg := s.ensureNetworkPackageLocked(id, now)
		pkg["onboardingState"] = "ONBOARDED"
		pkg["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{
			"nsdInfoId":          id,
			"arn":                tnbNetworkPackageARN(id),
			"nsdOnboardingState": "ONBOARDED",
		}

	case "GetSolNetworkPackage":
		return tnbCloneMap(s.ensureNetworkPackageLocked(nsdInfoID, now))

	case "ListSolNetworkPackages":
		return map[string]any{
			"networkPackages": s.sortedValuesLocked(s.networkPackages),
			"nextToken":       "",
		}

	case "UpdateSolNetworkPackage":
		pkg := s.ensureNetworkPackageLocked(nsdInfoID, now)
		pkg["lastModifiedTime"] = now.Format(time.RFC3339)
		pkg["onboardingState"] = "ONBOARDED"
		return map[string]any{}

	case "DeleteSolNetworkPackage":
		pkg := s.ensureNetworkPackageLocked(nsdInfoID, now)
		pkg["onboardingState"] = "DELETED"
		pkg["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "GetSolNetworkPackageDescriptor":
		s.ensureNetworkPackageLocked(nsdInfoID, now)
		return map[string]any{
			"nsdInfoId":   nsdInfoID,
			"descriptor":  "tosca_definitions_version: tosca_simple_yaml_1_2",
			"contentType": "text/yaml",
		}

	case "GetSolNetworkPackageContent":
		s.ensureNetworkPackageLocked(nsdInfoID, now)
		return map[string]any{
			"nsdInfoId": nsdInfoID,
			"content":   "UEsDBAoAAAAAA",
		}

	case "PutSolNetworkPackageContent":
		pkg := s.ensureNetworkPackageLocked(nsdInfoID, now)
		pkg["onboardingState"] = "ONBOARDED"
		pkg["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{"status": "SUCCESS"}

	case "ValidateSolNetworkPackageContent":
		s.ensureNetworkPackageLocked(nsdInfoID, now)
		return map[string]any{
			"status":      "VALIDATED",
			"nsdInfoId":   nsdInfoID,
			"description": "network package content is valid",
		}

	case "CreateSolFunctionPackage":
		id := fmt.Sprintf("vnfpkg-%06d", s.nextVnfPkgID)
		s.nextVnfPkgID++
		pkg := s.ensureFunctionPackageLocked(id, now)
		pkg["onboardingState"] = "ONBOARDED"
		pkg["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{
			"vnfPkgId":              id,
			"arn":                   tnbFunctionPackageARN(id),
			"vnfPkgOnboardingState": "ONBOARDED",
		}

	case "GetSolFunctionPackage":
		return tnbCloneMap(s.ensureFunctionPackageLocked(vnfPkgID, now))

	case "ListSolFunctionPackages":
		return map[string]any{
			"functionPackages": s.sortedValuesLocked(s.functionPackages),
			"nextToken":        "",
		}

	case "UpdateSolFunctionPackage":
		pkg := s.ensureFunctionPackageLocked(vnfPkgID, now)
		pkg["onboardingState"] = "ONBOARDED"
		pkg["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "DeleteSolFunctionPackage":
		pkg := s.ensureFunctionPackageLocked(vnfPkgID, now)
		pkg["onboardingState"] = "DELETED"
		pkg["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "GetSolFunctionPackageDescriptor":
		s.ensureFunctionPackageLocked(vnfPkgID, now)
		return map[string]any{
			"vnfPkgId":    vnfPkgID,
			"descriptor":  "tosca_definitions_version: tosca_simple_yaml_1_2",
			"contentType": "text/yaml",
		}

	case "GetSolFunctionPackageContent":
		s.ensureFunctionPackageLocked(vnfPkgID, now)
		return map[string]any{
			"vnfPkgId": vnfPkgID,
			"content":  "UEsDBAoAAAAAA",
		}

	case "PutSolFunctionPackageContent":
		pkg := s.ensureFunctionPackageLocked(vnfPkgID, now)
		pkg["onboardingState"] = "ONBOARDED"
		pkg["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{"status": "SUCCESS"}

	case "ValidateSolFunctionPackageContent":
		s.ensureFunctionPackageLocked(vnfPkgID, now)
		return map[string]any{
			"status":      "VALIDATED",
			"vnfPkgId":    vnfPkgID,
			"description": "function package content is valid",
		}

	case "CreateSolNetworkInstance":
		id := fmt.Sprintf("ns-instance-%06d", s.nextNsInstID)
		s.nextNsInstID++
		inst := s.ensureNetworkInstanceLocked(id, now)
		inst["nsState"] = "NOT_INSTANTIATED"
		inst["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{
			"nsInstanceId": id,
			"nsState":      "NOT_INSTANTIATED",
		}

	case "GetSolNetworkInstance":
		return tnbCloneMap(s.ensureNetworkInstanceLocked(nsInstanceID, now))

	case "ListSolNetworkInstances":
		return map[string]any{
			"networkInstances": s.sortedValuesLocked(s.networkInstances),
			"nextToken":        "",
		}

	case "InstantiateSolNetworkInstance":
		inst := s.ensureNetworkInstanceLocked(nsInstanceID, now)
		inst["nsState"] = "INSTANTIATED"
		inst["lastModifiedTime"] = now.Format(time.RFC3339)
		op := s.newOperationLocked(nsInstanceID, "INSTANTIATE", now)
		return map[string]any{"nsLcmOpOccId": op["nsLcmOpOccId"]}

	case "UpdateSolNetworkInstance":
		inst := s.ensureNetworkInstanceLocked(nsInstanceID, now)
		inst["nsState"] = "INSTANTIATED"
		inst["lastModifiedTime"] = now.Format(time.RFC3339)
		op := s.newOperationLocked(nsInstanceID, "UPDATE", now)
		return map[string]any{"nsLcmOpOccId": op["nsLcmOpOccId"]}

	case "TerminateSolNetworkInstance":
		inst := s.ensureNetworkInstanceLocked(nsInstanceID, now)
		inst["nsState"] = "TERMINATED"
		inst["lastModifiedTime"] = now.Format(time.RFC3339)
		op := s.newOperationLocked(nsInstanceID, "TERMINATE", now)
		return map[string]any{"nsLcmOpOccId": op["nsLcmOpOccId"]}

	case "DeleteSolNetworkInstance":
		inst := s.ensureNetworkInstanceLocked(nsInstanceID, now)
		inst["nsState"] = "DELETED"
		inst["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "GetSolFunctionInstance":
		return tnbCloneMap(s.ensureFunctionInstanceLocked(vnfInstanceID, now))

	case "ListSolFunctionInstances":
		return map[string]any{
			"functionInstances": s.sortedValuesLocked(s.functionInstances),
			"nextToken":         "",
		}

	case "GetSolNetworkOperation":
		return tnbCloneMap(s.ensureOperationLocked(nsLcmOpOccID, nsInstanceID, "INSTANTIATE", now))

	case "ListSolNetworkOperations":
		return map[string]any{
			"networkOperations": s.sortedValuesLocked(s.operations),
			"nextToken":         "",
		}

	case "CancelSolNetworkOperation":
		op := s.ensureOperationLocked(nsLcmOpOccID, nsInstanceID, "INSTANTIATE", now)
		op["operationState"] = "CANCELLED"
		op["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range tnbMapString(ctx["tags"]) {
			tags[key] = value
		}
		for key, value := range tnbMapString(ctx["Tags"]) {
			tags[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range tnbStringSlice(ctx["tagKeys"]) {
			delete(tags, key)
		}
		for _, key := range query["tagKeys"] {
			for _, split := range strings.Split(key, ",") {
				split = strings.TrimSpace(split)
				if split != "" {
					delete(tags, split)
				}
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": tnbCloneMapString(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{}
}

func (s *tnbStore) ensureNetworkPackageLocked(nsdInfoID string, now time.Time) map[string]any {
	id := strings.TrimSpace(nsdInfoID)
	if id == "" {
		id = "nsd-000001"
	}
	if existing := s.networkPackages[id]; existing != nil {
		return existing
	}
	out := map[string]any{
		"nsdInfoId":        id,
		"arn":              tnbNetworkPackageARN(id),
		"onboardingState":  "ONBOARDED",
		"operationalState": "ENABLED",
		"usageState":       "NOT_IN_USE",
		"createdTime":      now.Format(time.RFC3339),
		"lastModifiedTime": now.Format(time.RFC3339),
	}
	s.networkPackages[id] = out
	return out
}

func (s *tnbStore) ensureFunctionPackageLocked(vnfPkgID string, now time.Time) map[string]any {
	id := strings.TrimSpace(vnfPkgID)
	if id == "" {
		id = "vnfpkg-000001"
	}
	if existing := s.functionPackages[id]; existing != nil {
		return existing
	}
	out := map[string]any{
		"vnfPkgId":         id,
		"arn":              tnbFunctionPackageARN(id),
		"onboardingState":  "ONBOARDED",
		"operationalState": "ENABLED",
		"usageState":       "NOT_IN_USE",
		"createdTime":      now.Format(time.RFC3339),
		"lastModifiedTime": now.Format(time.RFC3339),
	}
	s.functionPackages[id] = out
	return out
}

func (s *tnbStore) ensureNetworkInstanceLocked(nsInstanceID string, now time.Time) map[string]any {
	id := strings.TrimSpace(nsInstanceID)
	if id == "" {
		id = "ns-instance-000001"
	}
	if existing := s.networkInstances[id]; existing != nil {
		return existing
	}
	out := map[string]any{
		"nsInstanceId":     id,
		"nsInstanceArn":    tnbNetworkInstanceARN(id),
		"nsdInfoId":        "nsd-000001",
		"nsState":          "NOT_INSTANTIATED",
		"createdTime":      now.Format(time.RFC3339),
		"lastModifiedTime": now.Format(time.RFC3339),
	}
	s.networkInstances[id] = out
	return out
}

func (s *tnbStore) ensureFunctionInstanceLocked(vnfInstanceID string, now time.Time) map[string]any {
	id := strings.TrimSpace(vnfInstanceID)
	if id == "" {
		id = "vnf-instance-000001"
	}
	if existing := s.functionInstances[id]; existing != nil {
		return existing
	}
	out := map[string]any{
		"vnfInstanceId":    id,
		"vnfInstanceArn":   tnbFunctionInstanceARN(id),
		"vnfPkgId":         "vnfpkg-000001",
		"vnfState":         "STARTED",
		"createdTime":      now.Format(time.RFC3339),
		"lastModifiedTime": now.Format(time.RFC3339),
	}
	s.functionInstances[id] = out
	return out
}

func (s *tnbStore) ensureOperationLocked(nsLcmOpOccID, nsInstanceID, opType string, now time.Time) map[string]any {
	id := strings.TrimSpace(nsLcmOpOccID)
	if id == "" {
		id = "ns-lcm-op-000001"
	}
	if existing := s.operations[id]; existing != nil {
		return existing
	}
	if strings.TrimSpace(opType) == "" {
		opType = "INSTANTIATE"
	}
	out := map[string]any{
		"nsLcmOpOccId":     id,
		"nsInstanceId":     strings.TrimSpace(nsInstanceID),
		"lcmOperationType": strings.ToUpper(strings.TrimSpace(opType)),
		"operationState":   "COMPLETED",
		"createdTime":      now.Format(time.RFC3339),
		"lastModifiedTime": now.Format(time.RFC3339),
	}
	if out["nsInstanceId"] == "" {
		out["nsInstanceId"] = "ns-instance-000001"
	}
	s.operations[id] = out
	return out
}

func (s *tnbStore) newOperationLocked(nsInstanceID, opType string, now time.Time) map[string]any {
	id := fmt.Sprintf("ns-lcm-op-%06d", s.nextNsLcmOpID)
	s.nextNsLcmOpID++
	return s.ensureOperationLocked(id, nsInstanceID, opType, now)
}

func (s *tnbStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = "arn:aws:tnb:us-east-1:123456789012:nsd/nsd-000001"
	}
	if existing := s.tags[resourceARN]; existing != nil {
		return existing
	}
	out := map[string]string{
		"env":     "local",
		"service": "tnb",
	}
	s.tags[resourceARN] = out
	return out
}

func (s *tnbStore) sortedValuesLocked(in map[string]map[string]any) []any {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, tnbCloneMap(in[key]))
	}
	return out
}

func tnbMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
		out[strings.ToLower(strings.TrimSpace(key))] = value
	}
	for key, values := range query {
		if len(values) == 1 {
			out[key] = values[0]
		} else if len(values) > 1 {
			list := make([]any, 0, len(values))
			for _, value := range values {
				list = append(list, value)
			}
			out[key] = list
		}
	}
	return out
}

func tnbString(source map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		for sourceKey, raw := range source {
			if !strings.EqualFold(strings.TrimSpace(sourceKey), strings.TrimSpace(key)) {
				continue
			}
			switch value := raw.(type) {
			case string:
				if trimmed := strings.TrimSpace(value); trimmed != "" {
					return trimmed
				}
			case []any:
				for _, entry := range value {
					if text, ok := entry.(string); ok {
						if trimmed := strings.TrimSpace(text); trimmed != "" {
							return trimmed
						}
					}
				}
			case fmt.Stringer:
				if trimmed := strings.TrimSpace(value.String()); trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return strings.TrimSpace(fallback)
}

func tnbStringSlice(raw any) []string {
	out := []string{}
	appendValue := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, existing := range out {
			if existing == value {
				return
			}
		}
		out = append(out, value)
	}
	switch value := raw.(type) {
	case string:
		for _, part := range strings.Split(value, ",") {
			appendValue(part)
		}
	case []string:
		for _, part := range value {
			appendValue(part)
		}
	case []any:
		for _, part := range value {
			if text, ok := part.(string); ok {
				appendValue(text)
			}
		}
	}
	return out
}

func tnbMapString(raw any) map[string]string {
	out := map[string]string{}
	switch value := raw.(type) {
	case map[string]string:
		for key, entry := range value {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(entry)
		}
	case map[string]any:
		for key, entry := range value {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			text, _ := entry.(string)
			out[key] = strings.TrimSpace(text)
		}
	}
	return out
}

func tnbCloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = tnbCloneAny(value)
	}
	return out
}

func tnbCloneAny(raw any) any {
	switch value := raw.(type) {
	case map[string]any:
		return tnbCloneMap(value)
	case map[string]string:
		return tnbCloneMapString(value)
	case []any:
		out := make([]any, 0, len(value))
		for _, entry := range value {
			out = append(out, tnbCloneAny(entry))
		}
		return out
	default:
		return raw
	}
}

func tnbCloneMapString(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out[key] = input[key]
	}
	return out
}

func tnbNetworkPackageARN(nsdInfoID string) string {
	return fmt.Sprintf("arn:aws:tnb:%s:%s:nsd/%s", tnbDefaultRegion, tnbDefaultAccountID, strings.TrimSpace(nsdInfoID))
}

func tnbFunctionPackageARN(vnfPkgID string) string {
	return fmt.Sprintf("arn:aws:tnb:%s:%s:vnf-package/%s", tnbDefaultRegion, tnbDefaultAccountID, strings.TrimSpace(vnfPkgID))
}

func tnbNetworkInstanceARN(nsInstanceID string) string {
	return fmt.Sprintf("arn:aws:tnb:%s:%s:ns-instance/%s", tnbDefaultRegion, tnbDefaultAccountID, strings.TrimSpace(nsInstanceID))
}

func tnbFunctionInstanceARN(vnfInstanceID string) string {
	return fmt.Sprintf("arn:aws:tnb:%s:%s:vnf-instance/%s", tnbDefaultRegion, tnbDefaultAccountID, strings.TrimSpace(vnfInstanceID))
}
