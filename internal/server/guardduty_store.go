package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

type guardDutyStore struct {
	mu sync.Mutex

	defaultDetectorID string
	defaultAccountID  string
	nextID            int64

	detectors map[string]map[string]any
}

func newGuardDutyStore() *guardDutyStore {
	s := &guardDutyStore{
		defaultDetectorID: "detector-00000000000000000000000000000001",
		defaultAccountID:  "123456789012",
		nextID:            1,
		detectors:         map[string]map[string]any{},
	}
	s.ensureDetectorLocked(s.defaultDetectorID)
	return s
}

func (s *guardDutyStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	detectorID := s.resolveDetectorID(payload, pathParams)
	resourceARN := s.resolveResourceARN(payload, pathParams)

	switch action {
	case "CreateDetector":
		if strings.TrimSpace(detectorID) == "" || detectorID == s.defaultDetectorID {
			s.nextID++
			detectorID = fmt.Sprintf("detector-%032d", s.nextID)
		}
		s.ensureDetectorLocked(detectorID)
		return map[string]any{"DetectorId": detectorID}
	case "DeleteDetector":
		delete(s.detectors, detectorID)
		return map[string]any{}
	case "ListDetectors":
		ids := s.listDetectorIDsLocked()
		return map[string]any{"DetectorIds": ids}
	case "GetDetector":
		return s.ensureDetectorLocked(detectorID)
	case "GetInvitationsCount":
		return map[string]any{"InvitationsCount": 0}
	case "ListInvitations":
		return map[string]any{"Invitations": []any{}, "NextToken": ""}
	case "ListOrganizationAdminAccounts":
		return map[string]any{
			"AdminAccounts": []any{
				map[string]any{
					"AdminAccountId": s.defaultAccountID,
					"Status":         "ENABLED",
				},
			},
			"NextToken": "",
		}
	case "GetOrganizationStatistics":
		return map[string]any{
			"OrganizationStatistics": map[string]any{
				"TotalAccountsCount":  1,
				"MemberAccountsCount": 1,
			},
		}
	case "CreateIPSet":
		return map[string]any{"IpSetId": "ipset-00000001"}
	case "CreateThreatIntelSet":
		return map[string]any{"ThreatIntelSetId": "threatintelset-00000001"}
	case "CreateThreatEntitySet":
		return map[string]any{"ThreatEntitySetId": "threatentityset-00000001"}
	case "CreateTrustedEntitySet":
		return map[string]any{"TrustedEntitySetId": "trustedentityset-00000001"}
	case "CreatePublishingDestination":
		return map[string]any{"DestinationId": "destination-00000001"}
	case "CreateFilter":
		name := strings.TrimSpace(guardDutyPayloadString(payload, "Name"))
		if name == "" {
			name = strings.TrimSpace(guardDutyPathParam(pathParams, "filterName"))
		}
		if name == "" {
			name = "stackyard-filter"
		}
		return map[string]any{"Name": name}
	case "CreateMalwareProtectionPlan":
		return map[string]any{"MalwareProtectionPlanId": "mp-00000001"}
	case "GetMalwareScan":
		return map[string]any{"ScanId": "scan-00000001", "Status": "COMPLETED"}
	case "TagResource":
		return map[string]any{"ResourceArn": resourceARN}
	default:
		return map[string]any{}
	}
}

func (s *guardDutyStore) resolveDetectorID(payload map[string]any, pathParams map[string]string) string {
	if v := guardDutyPathParam(pathParams, "detectorId"); v != "" {
		return v
	}
	for _, key := range []string{"DetectorId", "detectorId"} {
		if v := guardDutyPayloadString(payload, key); v != "" {
			return v
		}
	}
	return s.defaultDetectorID
}

func (s *guardDutyStore) resolveResourceARN(payload map[string]any, pathParams map[string]string) string {
	if v := guardDutyPathParam(pathParams, "resourceArn"); v != "" {
		return v
	}
	for _, key := range []string{"ResourceArn", "resourceArn"} {
		if v := guardDutyPayloadString(payload, key); v != "" {
			return v
		}
	}
	return fmt.Sprintf("arn:aws:guardduty:us-east-1:%s:detector/%s", s.defaultAccountID, s.defaultDetectorID)
}

func (s *guardDutyStore) ensureDetectorLocked(detectorID string) map[string]any {
	detectorID = strings.TrimSpace(detectorID)
	if detectorID == "" {
		detectorID = s.defaultDetectorID
	}
	if existing, ok := s.detectors[detectorID]; ok {
		return existing
	}
	detector := map[string]any{
		"DetectorId": detectorID,
		"Status":     "ENABLED",
		"ServiceRole": fmt.Sprintf(
			"arn:aws:iam::%s:role/service-role/AmazonGuardDutyServiceRolePolicy",
			s.defaultAccountID,
		),
		"DataSources": map[string]any{},
		"Features":    []any{},
	}
	s.detectors[detectorID] = detector
	return detector
}

func (s *guardDutyStore) listDetectorIDsLocked() []string {
	ids := make([]string, 0, len(s.detectors))
	for detectorID := range s.detectors {
		ids = append(ids, detectorID)
	}
	sort.Strings(ids)
	return ids
}

func guardDutyPathParam(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	if v, ok := pathParams[key]; ok {
		return strings.TrimSpace(v)
	}
	for k, v := range pathParams {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func guardDutyPayloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload[key]; ok {
		return strings.TrimSpace(guardDutyAnyString(v))
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(guardDutyAnyString(v))
		}
	}
	return ""
}

func guardDutyAnyString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}
