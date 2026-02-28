package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type detectiveStore struct {
	mu sync.Mutex

	nextGraphID         int64
	nextInvestigationID int64

	graphs              map[string]map[string]any
	graphMembers        map[string]map[string]map[string]any
	graphInvitations    map[string]map[string]map[string]any
	graphInvestigations map[string]map[string]map[string]any
	graphDatasources    map[string][]string
	orgAdmins           map[string]map[string]any
	orgConfig           map[string]map[string]any
	tags                map[string]map[string]string
}

func newDetectiveStore() *detectiveStore {
	s := &detectiveStore{
		nextGraphID:         2,
		nextInvestigationID: 2,
		graphs:              map[string]map[string]any{},
		graphMembers:        map[string]map[string]map[string]any{},
		graphInvitations:    map[string]map[string]map[string]any{},
		graphInvestigations: map[string]map[string]map[string]any{},
		graphDatasources:    map[string][]string{},
		orgAdmins:           map[string]map[string]any{},
		orgConfig:           map[string]map[string]any{},
		tags:                map[string]map[string]string{},
	}

	graphArn := detectiveGraphARN("graph-00000001")
	now := time.Now().UTC().Format(time.RFC3339)
	s.graphs[graphArn] = map[string]any{
		"Arn":         graphArn,
		"CreatedTime": now,
	}
	s.graphMembers[graphArn] = map[string]map[string]any{
		"123456789012": {
			"AccountId":          "123456789012",
			"EmailAddress":       "owner@example.com",
			"Status":             "ENABLED",
			"InvitedTime":        now,
			"UpdatedTime":        now,
			"DisabledReason":     "",
			"VolumeUsageInBytes": int64(0),
		},
	}
	s.graphInvitations[graphArn] = map[string]map[string]any{
		"111122223333": {
			"AccountId":    "111122223333",
			"EmailAddress": "member@example.com",
			"GraphArn":     graphArn,
			"InvitedTime":  now,
			"Message":      "Stackyard invitation",
		},
	}
	s.graphInvestigations[graphArn] = map[string]map[string]any{
		"investigation-00000001": {
			"GraphArn":        graphArn,
			"InvestigationId": "investigation-00000001",
			"EntityArn":       "arn:aws:iam::123456789012:user/stackyard-user",
			"EntityType":      "IAM_USER",
			"CreatedTime":     now,
			"ScopeStartTime":  now,
			"ScopeEndTime":    now,
			"Status":          "SUCCESSFUL",
			"Severity":        "LOW",
			"State":           "ACTIVE",
		},
	}
	s.graphDatasources[graphArn] = []string{"DETECTIVE_CORE"}
	s.orgAdmins["123456789012"] = map[string]any{
		"AccountId": "123456789012",
		"GraphArn":  graphArn,
		"Status":    "ENABLED",
	}
	s.orgConfig[graphArn] = map[string]any{"AutoEnable": true}
	s.tags[graphArn] = map[string]string{"seed": "true"}
	return s
}

func (s *detectiveStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	graphArn := detectiveFirstNonEmpty(
		detectiveString(payload, "GraphArn", ""),
		detectiveGraphARN("graph-00000001"),
	)
	resourceArn := detectiveFirstNonEmpty(
		detectivePathParam(pathParams, "ResourceArn", ""),
		detectiveString(payload, "ResourceArn", ""),
		graphArn,
	)
	accountID := detectiveFirstNonEmpty(
		detectiveString(payload, "AccountId", ""),
		"111122223333",
	)
	investigationID := detectiveFirstNonEmpty(
		detectiveString(payload, "InvestigationId", ""),
		"investigation-00000001",
	)
	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateGraph":
		graphID := fmt.Sprintf("graph-%08d", s.nextGraphID)
		s.nextGraphID++
		graphArn = detectiveGraphARN(graphID)
		s.graphs[graphArn] = map[string]any{
			"Arn":         graphArn,
			"CreatedTime": now,
		}
		s.graphMembers[graphArn] = map[string]map[string]any{}
		s.graphInvitations[graphArn] = map[string]map[string]any{}
		s.graphInvestigations[graphArn] = map[string]map[string]any{}
		s.graphDatasources[graphArn] = []string{"DETECTIVE_CORE"}
		s.orgConfig[graphArn] = map[string]any{"AutoEnable": true}
		s.ensureTagsLocked(graphArn)
		return map[string]any{"GraphArn": graphArn}

	case "DeleteGraph":
		delete(s.graphs, graphArn)
		delete(s.graphMembers, graphArn)
		delete(s.graphInvitations, graphArn)
		delete(s.graphInvestigations, graphArn)
		delete(s.graphDatasources, graphArn)
		delete(s.orgConfig, graphArn)
		delete(s.tags, graphArn)
		return map[string]any{}

	case "ListGraphs":
		items := make([]any, 0, len(s.graphs))
		keys := make([]string, 0, len(s.graphs))
		for arn := range s.graphs {
			keys = append(keys, arn)
		}
		sort.Strings(keys)
		for _, arn := range keys {
			items = append(items, detectiveCloneMap(s.graphs[arn]))
		}
		return map[string]any{"GraphList": items, "NextToken": ""}

	case "CreateMembers":
		accounts := detectiveAccountList(payload, "Accounts")
		memberMap := s.ensureMembersLocked(graphArn)
		invitations := s.ensureInvitationsLocked(graphArn)
		created := make([]any, 0, len(accounts))
		for _, acct := range accounts {
			id := detectiveFirstNonEmpty(acct["AccountId"], "111122223333")
			email := detectiveFirstNonEmpty(acct["EmailAddress"], "member@example.com")
			memberMap[id] = map[string]any{
				"AccountId":    id,
				"EmailAddress": email,
				"Status":       "INVITED",
				"InvitedTime":  now,
				"UpdatedTime":  now,
			}
			invitations[id] = map[string]any{
				"AccountId":    id,
				"EmailAddress": email,
				"GraphArn":     graphArn,
				"InvitedTime":  now,
				"Message":      detectiveString(payload, "Message", "Stackyard invitation"),
			}
			created = append(created, detectiveCloneMap(memberMap[id]))
		}
		return map[string]any{"Members": created, "UnprocessedAccounts": []any{}}

	case "DeleteMembers":
		accountIDs := detectiveStringSlice(payload, "AccountIds")
		if len(accountIDs) == 0 {
			accountIDs = []string{accountID}
		}
		memberMap := s.ensureMembersLocked(graphArn)
		invitations := s.ensureInvitationsLocked(graphArn)
		for _, id := range accountIDs {
			delete(memberMap, id)
			delete(invitations, id)
		}
		out := make([]any, 0, len(accountIDs))
		for _, id := range accountIDs {
			out = append(out, id)
		}
		return map[string]any{"AccountIds": out, "UnprocessedAccounts": []any{}}

	case "GetMembers":
		accountIDs := detectiveStringSlice(payload, "AccountIds")
		if len(accountIDs) == 0 {
			accountIDs = []string{accountID}
		}
		memberMap := s.ensureMembersLocked(graphArn)
		out := make([]any, 0, len(accountIDs))
		for _, id := range accountIDs {
			member := memberMap[id]
			if member == nil {
				member = map[string]any{
					"AccountId":    id,
					"EmailAddress": "member@example.com",
					"Status":       "INVITED",
					"InvitedTime":  now,
					"UpdatedTime":  now,
				}
				memberMap[id] = member
			}
			out = append(out, detectiveCloneMap(member))
		}
		return map[string]any{"MemberDetails": out, "UnprocessedAccounts": []any{}}

	case "ListMembers":
		memberMap := s.ensureMembersLocked(graphArn)
		keys := make([]string, 0, len(memberMap))
		for id := range memberMap {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		out := make([]any, 0, len(keys))
		for _, id := range keys {
			out = append(out, detectiveCloneMap(memberMap[id]))
		}
		return map[string]any{"MemberDetails": out, "NextToken": ""}

	case "DisassociateMembership":
		return map[string]any{}

	case "AcceptInvitation":
		memberMap := s.ensureMembersLocked(graphArn)
		member := memberMap["111122223333"]
		if member == nil {
			member = map[string]any{"AccountId": "111122223333", "EmailAddress": "member@example.com"}
			memberMap["111122223333"] = member
		}
		member["Status"] = "ENABLED"
		member["UpdatedTime"] = now
		return map[string]any{}

	case "RejectInvitation":
		delete(s.ensureInvitationsLocked(graphArn), "111122223333")
		return map[string]any{}

	case "ListInvitations":
		keys := make([]string, 0, len(s.graphInvitations))
		for arn := range s.graphInvitations {
			keys = append(keys, arn)
		}
		sort.Strings(keys)
		out := make([]any, 0)
		for _, arn := range keys {
			invMap := s.ensureInvitationsLocked(arn)
			ids := make([]string, 0, len(invMap))
			for id := range invMap {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			for _, id := range ids {
				out = append(out, detectiveCloneMap(invMap[id]))
			}
		}
		return map[string]any{"Invitations": out, "NextToken": ""}

	case "EnableOrganizationAdminAccount":
		s.orgAdmins[accountID] = map[string]any{
			"AccountId": accountID,
			"GraphArn":  graphArn,
			"Status":    "ENABLED",
		}
		return map[string]any{}

	case "DisableOrganizationAdminAccount":
		delete(s.orgAdmins, accountID)
		return map[string]any{}

	case "ListOrganizationAdminAccounts":
		keys := make([]string, 0, len(s.orgAdmins))
		for id := range s.orgAdmins {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		out := make([]any, 0, len(keys))
		for _, id := range keys {
			out = append(out, detectiveCloneMap(s.orgAdmins[id]))
		}
		return map[string]any{"Administrators": out, "NextToken": ""}

	case "DescribeOrganizationConfiguration":
		cfg := s.orgConfig[graphArn]
		if cfg == nil {
			cfg = map[string]any{"AutoEnable": true}
			s.orgConfig[graphArn] = cfg
		}
		return map[string]any{"AutoEnable": cfg["AutoEnable"]}

	case "UpdateOrganizationConfiguration":
		cfg := s.orgConfig[graphArn]
		if cfg == nil {
			cfg = map[string]any{}
			s.orgConfig[graphArn] = cfg
		}
		autoEnable := true
		if v, ok := payload["AutoEnable"]; ok {
			if b, ok := v.(bool); ok {
				autoEnable = b
			}
		}
		cfg["AutoEnable"] = autoEnable
		return map[string]any{}

	case "ListDatasourcePackages":
		pkgs := detectiveCloneStringSlice(s.ensureDatasourcePackagesLocked(graphArn))
		out := make(map[string]any, len(pkgs))
		for _, pkg := range pkgs {
			out[pkg] = map[string]any{
				"DatasourcePackageIngestState": "STARTED",
				"LastIngestStateChange": map[string]any{
					pkg: map[string]any{
						"Timestamp": now,
					},
				},
			}
		}
		return map[string]any{"DatasourcePackages": out, "NextToken": ""}

	case "UpdateDatasourcePackages":
		pkgs := detectiveStringSlice(payload, "DatasourcePackages")
		if len(pkgs) == 0 {
			pkgs = []string{"DETECTIVE_CORE"}
		}
		s.graphDatasources[graphArn] = detectiveCloneStringSlice(pkgs)
		return map[string]any{}

	case "BatchGetGraphMemberDatasources":
		accountIDs := detectiveStringSlice(payload, "AccountIds")
		if len(accountIDs) == 0 {
			accountIDs = []string{"111122223333"}
		}
		memberDatasources := make([]any, 0, len(accountIDs))
		pkgs := detectiveCloneStringSlice(s.ensureDatasourcePackagesLocked(graphArn))
		for _, id := range accountIDs {
			ingest := make([]any, 0, len(pkgs))
			for _, pkg := range pkgs {
				ingest = append(ingest, map[string]any{
					"DatasourcePackage":     pkg,
					"IngestState":           "STARTED",
					"LastIngestStateChange": now,
				})
			}
			memberDatasources = append(memberDatasources, map[string]any{
				"AccountId":                     id,
				"GraphArn":                      graphArn,
				"DatasourcePackageIngestStates": ingest,
			})
		}
		return map[string]any{"MemberDatasources": memberDatasources, "UnprocessedAccounts": []any{}}

	case "BatchGetMembershipDatasources":
		graphArns := detectiveStringSlice(payload, "GraphArns")
		if len(graphArns) == 0 {
			graphArns = []string{graphArn}
		}
		out := make([]any, 0, len(graphArns))
		for _, arn := range graphArns {
			pkgs := detectiveCloneStringSlice(s.ensureDatasourcePackagesLocked(arn))
			ingest := make([]any, 0, len(pkgs))
			for _, pkg := range pkgs {
				ingest = append(ingest, map[string]any{
					"DatasourcePackage":     pkg,
					"IngestState":           "STARTED",
					"LastIngestStateChange": now,
				})
			}
			out = append(out, map[string]any{
				"GraphArn":                      arn,
				"DatasourcePackageIngestStates": ingest,
			})
		}
		return map[string]any{"MembershipDatasources": out, "UnprocessedGraphs": []any{}}

	case "StartMonitoringMember":
		member := s.ensureMembersLocked(graphArn)[accountID]
		if member == nil {
			member = map[string]any{
				"AccountId":    accountID,
				"EmailAddress": "member@example.com",
				"Status":       "ENABLED",
			}
			s.ensureMembersLocked(graphArn)[accountID] = member
		}
		member["Status"] = "ENABLED"
		member["UpdatedTime"] = now
		return map[string]any{}

	case "StartInvestigation":
		investigationID = fmt.Sprintf("investigation-%08d", s.nextInvestigationID)
		s.nextInvestigationID++
		entityArn := detectiveFirstNonEmpty(detectiveString(payload, "EntityArn", ""), "arn:aws:iam::123456789012:user/stackyard-user")
		scopeStart := detectiveFirstNonEmpty(detectiveString(payload, "ScopeStartTime", ""), now)
		scopeEnd := detectiveFirstNonEmpty(detectiveString(payload, "ScopeEndTime", ""), now)
		item := map[string]any{
			"GraphArn":        graphArn,
			"InvestigationId": investigationID,
			"EntityArn":       entityArn,
			"EntityType":      detectiveEntityTypeFromARN(entityArn),
			"CreatedTime":     now,
			"ScopeStartTime":  scopeStart,
			"ScopeEndTime":    scopeEnd,
			"Status":          "RUNNING",
			"Severity":        "LOW",
			"State":           "ACTIVE",
		}
		s.ensureInvestigationsLocked(graphArn)[investigationID] = item
		return map[string]any{"InvestigationId": investigationID}

	case "GetInvestigation":
		item := s.ensureInvestigationLocked(graphArn, investigationID, now)
		return detectiveCloneMap(item)

	case "ListInvestigations":
		items := s.ensureInvestigationsLocked(graphArn)
		keys := make([]string, 0, len(items))
		for id := range items {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		out := make([]any, 0, len(keys))
		for _, id := range keys {
			out = append(out, detectiveCloneMap(items[id]))
		}
		return map[string]any{"InvestigationDetails": out, "NextToken": ""}

	case "ListIndicators":
		item := s.ensureInvestigationLocked(graphArn, investigationID, now)
		indicators := []any{
			map[string]any{
				"Type":        "TTP_OBSERVED",
				"Title":       "Credential access observed",
				"Severity":    item["Severity"],
				"CreatedTime": now,
				"State":       item["State"],
			},
		}
		return map[string]any{
			"GraphArn":        graphArn,
			"InvestigationId": investigationID,
			"Indicators":      indicators,
			"NextToken":       "",
		}

	case "UpdateInvestigationState":
		item := s.ensureInvestigationLocked(graphArn, investigationID, now)
		state := detectiveFirstNonEmpty(detectiveString(payload, "State", ""), "ACTIVE")
		item["State"] = state
		item["Status"] = "SUCCESSFUL"
		return map[string]any{}

	case "TagResource":
		tagMap := s.ensureTagsLocked(resourceArn)
		for k, v := range detectiveMapString(payload, "Tags") {
			tagMap[k] = v
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"Tags": detectiveCloneStringMap(s.ensureTagsLocked(resourceArn))}

	case "UntagResource":
		tagMap := s.ensureTagsLocked(resourceArn)
		for _, key := range detectiveStringSlice(payload, "TagKeys") {
			delete(tagMap, key)
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *detectiveStore) ensureMembersLocked(graphArn string) map[string]map[string]any {
	members := s.graphMembers[graphArn]
	if members == nil {
		members = map[string]map[string]any{}
		s.graphMembers[graphArn] = members
	}
	return members
}

func (s *detectiveStore) ensureInvitationsLocked(graphArn string) map[string]map[string]any {
	invitations := s.graphInvitations[graphArn]
	if invitations == nil {
		invitations = map[string]map[string]any{}
		s.graphInvitations[graphArn] = invitations
	}
	return invitations
}

func (s *detectiveStore) ensureInvestigationsLocked(graphArn string) map[string]map[string]any {
	items := s.graphInvestigations[graphArn]
	if items == nil {
		items = map[string]map[string]any{}
		s.graphInvestigations[graphArn] = items
	}
	return items
}

func (s *detectiveStore) ensureInvestigationLocked(graphArn, investigationID, now string) map[string]any {
	items := s.ensureInvestigationsLocked(graphArn)
	item := items[investigationID]
	if item == nil {
		item = map[string]any{
			"GraphArn":        graphArn,
			"InvestigationId": investigationID,
			"EntityArn":       "arn:aws:iam::123456789012:user/stackyard-user",
			"EntityType":      "IAM_USER",
			"CreatedTime":     now,
			"ScopeStartTime":  now,
			"ScopeEndTime":    now,
			"Status":          "RUNNING",
			"Severity":        "LOW",
			"State":           "ACTIVE",
		}
		items[investigationID] = item
	}
	return item
}

func (s *detectiveStore) ensureDatasourcePackagesLocked(graphArn string) []string {
	pkgs := s.graphDatasources[graphArn]
	if len(pkgs) == 0 {
		pkgs = []string{"DETECTIVE_CORE"}
		s.graphDatasources[graphArn] = pkgs
	}
	return pkgs
}

func (s *detectiveStore) ensureTagsLocked(resourceArn string) map[string]string {
	tags := s.tags[resourceArn]
	if tags == nil {
		tags = map[string]string{}
		s.tags[resourceArn] = tags
	}
	return tags
}

func detectiveGraphARN(id string) string {
	return "arn:aws:detective:us-east-1:123456789012:graph:" + id
}

func detectiveEntityTypeFromARN(entityArn string) string {
	arn := strings.ToLower(strings.TrimSpace(entityArn))
	if strings.Contains(arn, ":role/") {
		return "IAM_ROLE"
	}
	return "IAM_USER"
}

func detectiveString(payload map[string]any, key, def string) string {
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		if s, ok := v.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return def
}

func detectivePathParam(pathParams map[string]string, key, def string) string {
	for k, v := range pathParams {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			v = strings.TrimSpace(v)
			if v != "" {
				return v
			}
		}
	}
	return def
}

func detectiveStringSlice(payload map[string]any, key string) []string {
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		switch vals := v.(type) {
		case []any:
			out := make([]string, 0, len(vals))
			for _, item := range vals {
				if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
					out = append(out, strings.TrimSpace(s))
				}
			}
			return out
		case []string:
			out := make([]string, 0, len(vals))
			for _, s := range vals {
				if strings.TrimSpace(s) != "" {
					out = append(out, strings.TrimSpace(s))
				}
			}
			return out
		}
	}
	return nil
}

func detectiveMapString(payload map[string]any, key string) map[string]string {
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		switch raw := v.(type) {
		case map[string]any:
			out := make(map[string]string, len(raw))
			for rk, rv := range raw {
				if s, ok := rv.(string); ok {
					out[rk] = s
				}
			}
			return out
		case map[string]string:
			out := make(map[string]string, len(raw))
			for rk, rv := range raw {
				out[rk] = rv
			}
			return out
		}
	}
	return map[string]string{}
}

func detectiveAccountList(payload map[string]any, key string) []map[string]string {
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		items, ok := v.([]any)
		if !ok || len(items) == 0 {
			break
		}
		out := make([]map[string]string, 0, len(items))
		for _, item := range items {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			out = append(out, map[string]string{
				"AccountId":    detectiveFirstNonEmpty(detectiveString(entry, "AccountId", ""), "111122223333"),
				"EmailAddress": detectiveFirstNonEmpty(detectiveString(entry, "EmailAddress", ""), "member@example.com"),
			})
		}
		if len(out) > 0 {
			return out
		}
	}
	return []map[string]string{{"AccountId": "111122223333", "EmailAddress": "member@example.com"}}
}

func detectiveFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func detectiveCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func detectiveCloneStringSlice(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func detectiveCloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
