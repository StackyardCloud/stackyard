package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type mpaStore struct {
	mu sync.Mutex

	nextApprovalTeam   int64
	nextIdentitySource int64
	nextSession        int64

	approvalTeams   map[string]map[string]any
	identitySources map[string]map[string]any
	sessions        map[string]map[string]any
	policies        map[string]map[string]any
	policyVersions  map[string]map[string]any
	resourcePolicy  map[string]map[string]any
	tags            map[string]map[string]string
}

func newMPAStore() *mpaStore {
	s := &mpaStore{
		nextApprovalTeam:   2,
		nextIdentitySource: 2,
		nextSession:        2,
		approvalTeams:      map[string]map[string]any{},
		identitySources:    map[string]map[string]any{},
		sessions:           map[string]map[string]any{},
		policies:           map[string]map[string]any{},
		policyVersions:     map[string]map[string]any{},
		resourcePolicy:     map[string]map[string]any{},
		tags:               map[string]map[string]string{},
	}

	teamArn := "arn:aws:mpa:us-east-1:123456789012:approval-team/stackyard"
	identitySourceArn := "arn:aws:mpa:us-east-1:123456789012:identity-source/stackyard"
	sessionArn := "arn:aws:mpa:us-east-1:123456789012:session/stackyard"
	policyArn := "arn:aws:mpa:us-east-1:123456789012:policy/stackyard"
	policyVersionArn := "arn:aws:mpa:us-east-1:123456789012:policy-version/stackyard/1"
	resourceArn := "arn:aws:mpa:us-east-1:123456789012:resource/stackyard"

	s.ensureApprovalTeamLocked(teamArn)
	s.ensureIdentitySourceLocked(identitySourceArn)
	s.ensureSessionLocked(sessionArn, teamArn)
	s.ensurePolicyLocked(policyArn)
	s.ensurePolicyVersionLocked(policyVersionArn, policyArn)
	s.resourcePolicy[resourceArn] = map[string]any{
		"ResourceArn": resourceArn,
		"Policy":      `{"Version":"2012-10-17","Statement":[]}`,
		"UpdatedAt":   time.Now().UTC().Format(time.RFC3339),
	}
	s.tags[teamArn] = map[string]string{"stackyard": "true"}
	s.tags[resourceArn] = map[string]string{"stackyard": "true"}
	return s
}

func (s *mpaStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateApprovalTeam":
		arn := mpaDefaultStringAny(payload, "Arn", "")
		if arn == "" {
			arn = fmt.Sprintf("arn:aws:mpa:us-east-1:123456789012:approval-team/stackyard-%06d", s.nextApprovalTeam)
			s.nextApprovalTeam++
		}
		team := s.ensureApprovalTeamLocked(arn)
		for key, value := range payload {
			team[key] = value
		}
		return map[string]any{
			"ApprovalTeam": mpaCloneAnyMap(team),
		}

	case "GetApprovalTeam":
		arn := mpaDefaultString(pathParams, "Arn", "arn:aws:mpa:us-east-1:123456789012:approval-team/stackyard")
		team := s.ensureApprovalTeamLocked(arn)
		return map[string]any{"ApprovalTeam": mpaCloneAnyMap(team)}

	case "ListApprovalTeams":
		items := make([]any, 0, len(s.approvalTeams))
		keys := mpaSortedKeys(s.approvalTeams)
		for _, key := range keys {
			items = append(items, mpaCloneAnyMap(s.approvalTeams[key]))
		}
		return map[string]any{"ApprovalTeams": items, "NextToken": ""}

	case "UpdateApprovalTeam":
		arn := mpaDefaultString(pathParams, "Arn", "arn:aws:mpa:us-east-1:123456789012:approval-team/stackyard")
		team := s.ensureApprovalTeamLocked(arn)
		for key, value := range payload {
			team[key] = value
		}
		team["UpdatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"ApprovalTeam": mpaCloneAnyMap(team)}

	case "DeleteInactiveApprovalTeamVersion":
		arn := mpaDefaultString(pathParams, "Arn", "arn:aws:mpa:us-east-1:123456789012:approval-team/stackyard")
		team := s.ensureApprovalTeamLocked(arn)
		team["InactiveVersionDeleted"] = true
		team["UpdatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{}

	case "StartActiveApprovalTeamDeletion":
		arn := mpaDefaultString(pathParams, "Arn", "arn:aws:mpa:us-east-1:123456789012:approval-team/stackyard")
		team := s.ensureApprovalTeamLocked(arn)
		team["Status"] = "DELETING"
		team["UpdatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"ApprovalTeam": mpaCloneAnyMap(team)}

	case "CreateIdentitySource":
		arn := mpaDefaultStringAny(payload, "IdentitySourceArn", "")
		if arn == "" {
			arn = fmt.Sprintf("arn:aws:mpa:us-east-1:123456789012:identity-source/stackyard-%06d", s.nextIdentitySource)
			s.nextIdentitySource++
		}
		source := s.ensureIdentitySourceLocked(arn)
		for key, value := range payload {
			source[key] = value
		}
		return map[string]any{"IdentitySource": mpaCloneAnyMap(source)}

	case "GetIdentitySource":
		arn := mpaDefaultString(pathParams, "IdentitySourceArn", "arn:aws:mpa:us-east-1:123456789012:identity-source/stackyard")
		source := s.ensureIdentitySourceLocked(arn)
		return map[string]any{"IdentitySource": mpaCloneAnyMap(source)}

	case "ListIdentitySources":
		items := make([]any, 0, len(s.identitySources))
		keys := mpaSortedKeys(s.identitySources)
		for _, key := range keys {
			items = append(items, mpaCloneAnyMap(s.identitySources[key]))
		}
		return map[string]any{"IdentitySources": items, "NextToken": ""}

	case "DeleteIdentitySource":
		arn := mpaDefaultString(pathParams, "IdentitySourceArn", "arn:aws:mpa:us-east-1:123456789012:identity-source/stackyard")
		delete(s.identitySources, arn)
		return map[string]any{}

	case "GetSession":
		sessionArn := mpaDefaultString(pathParams, "SessionArn", "arn:aws:mpa:us-east-1:123456789012:session/stackyard")
		session := s.ensureSessionLocked(sessionArn, "arn:aws:mpa:us-east-1:123456789012:approval-team/stackyard")
		return map[string]any{"Session": mpaCloneAnyMap(session)}

	case "ListSessions":
		approvalTeamArn := mpaDefaultString(pathParams, "ApprovalTeamArn", "arn:aws:mpa:us-east-1:123456789012:approval-team/stackyard")
		items := make([]any, 0, len(s.sessions))
		keys := mpaSortedKeys(s.sessions)
		for _, key := range keys {
			session := s.sessions[key]
			if mpaDefaultStringAny(session, "ApprovalTeamArn", "") != approvalTeamArn {
				continue
			}
			items = append(items, mpaCloneAnyMap(session))
		}
		return map[string]any{"Sessions": items, "NextToken": ""}

	case "CancelSession":
		sessionArn := mpaDefaultString(pathParams, "SessionArn", "arn:aws:mpa:us-east-1:123456789012:session/stackyard")
		session := s.ensureSessionLocked(sessionArn, "arn:aws:mpa:us-east-1:123456789012:approval-team/stackyard")
		session["Status"] = "CANCELED"
		session["UpdatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"Session": mpaCloneAnyMap(session)}

	case "ListPolicies":
		items := make([]any, 0, len(s.policies))
		keys := mpaSortedKeys(s.policies)
		for _, key := range keys {
			items = append(items, mpaCloneAnyMap(s.policies[key]))
		}
		return map[string]any{"Policies": items, "NextToken": ""}

	case "ListPolicyVersions":
		policyArn := mpaDefaultString(pathParams, "PolicyArn", "arn:aws:mpa:us-east-1:123456789012:policy/stackyard")
		items := make([]any, 0, len(s.policyVersions))
		keys := mpaSortedKeys(s.policyVersions)
		for _, key := range keys {
			version := s.policyVersions[key]
			if mpaDefaultStringAny(version, "PolicyArn", "") != policyArn {
				continue
			}
			items = append(items, mpaCloneAnyMap(version))
		}
		return map[string]any{"PolicyVersions": items, "NextToken": ""}

	case "GetPolicyVersion":
		versionArn := mpaDefaultString(pathParams, "PolicyVersionArn", "arn:aws:mpa:us-east-1:123456789012:policy-version/stackyard/1")
		version := s.ensurePolicyVersionLocked(versionArn, "arn:aws:mpa:us-east-1:123456789012:policy/stackyard")
		return map[string]any{"PolicyVersion": mpaCloneAnyMap(version)}

	case "GetResourcePolicy":
		resourceArn := mpaDefaultStringAny(payload, "ResourceArn", "arn:aws:mpa:us-east-1:123456789012:resource/stackyard")
		policy := s.ensureResourcePolicyLocked(resourceArn)
		return map[string]any{"ResourcePolicy": mpaCloneAnyMap(policy)}

	case "ListResourcePolicies":
		resourceArn := mpaDefaultString(pathParams, "ResourceArn", "arn:aws:mpa:us-east-1:123456789012:resource/stackyard")
		policy := s.ensureResourcePolicyLocked(resourceArn)
		return map[string]any{"ResourcePolicies": []any{mpaCloneAnyMap(policy)}, "NextToken": ""}

	case "ListTagsForResource":
		resourceArn := mpaDefaultString(pathParams, "ResourceArn", "arn:aws:mpa:us-east-1:123456789012:resource/stackyard")
		return map[string]any{"Tags": mpaCloneStringMap(s.tags[resourceArn])}

	case "TagResource":
		resourceArn := mpaDefaultString(pathParams, "ResourceArn", "arn:aws:mpa:us-east-1:123456789012:resource/stackyard")
		if s.tags[resourceArn] == nil {
			s.tags[resourceArn] = map[string]string{}
		}
		for key, value := range mpaMapString(payload, "Tags") {
			s.tags[resourceArn][key] = value
		}
		return map[string]any{}

	case "UntagResource":
		resourceArn := mpaDefaultString(pathParams, "ResourceArn", "arn:aws:mpa:us-east-1:123456789012:resource/stackyard")
		for _, key := range mpaStringSlice(payload, "TagKeys") {
			if s.tags[resourceArn] != nil {
				delete(s.tags[resourceArn], key)
			}
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *mpaStore) ensureApprovalTeamLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = "arn:aws:mpa:us-east-1:123456789012:approval-team/stackyard"
	}
	if team := s.approvalTeams[arn]; team != nil {
		return team
	}
	item := map[string]any{
		"Arn":               arn,
		"Name":              "stackyard-team",
		"Description":       "stackyard seeded approval team",
		"ApprovalThreshold": 1,
		"Status":            "ACTIVE",
		"CreatedAt":         time.Now().UTC().Format(time.RFC3339),
		"UpdatedAt":         time.Now().UTC().Format(time.RFC3339),
	}
	s.approvalTeams[arn] = item
	return item
}

func (s *mpaStore) ensureIdentitySourceLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = "arn:aws:mpa:us-east-1:123456789012:identity-source/stackyard"
	}
	if source := s.identitySources[arn]; source != nil {
		return source
	}
	item := map[string]any{
		"IdentitySourceArn": arn,
		"Name":              "stackyard-identity-source",
		"Type":              "IAM_IDENTITY_CENTER",
		"CreatedAt":         time.Now().UTC().Format(time.RFC3339),
	}
	s.identitySources[arn] = item
	return item
}

func (s *mpaStore) ensureSessionLocked(arn, approvalTeamArn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = "arn:aws:mpa:us-east-1:123456789012:session/stackyard"
	}
	if item := s.sessions[arn]; item != nil {
		return item
	}
	if strings.TrimSpace(approvalTeamArn) == "" {
		approvalTeamArn = "arn:aws:mpa:us-east-1:123456789012:approval-team/stackyard"
	}
	item := map[string]any{
		"SessionArn":        arn,
		"ApprovalTeamArn":   approvalTeamArn,
		"Status":            "PENDING",
		"RequesterArn":      "arn:aws:iam::123456789012:user/stackyard",
		"RequestedAt":       time.Now().UTC().Format(time.RFC3339),
		"LastUpdatedAt":     time.Now().UTC().Format(time.RFC3339),
		"RequiredApprovals": 1,
	}
	s.sessions[arn] = item
	return item
}

func (s *mpaStore) ensurePolicyLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = "arn:aws:mpa:us-east-1:123456789012:policy/stackyard"
	}
	if policy := s.policies[arn]; policy != nil {
		return policy
	}
	item := map[string]any{
		"PolicyArn":     arn,
		"Name":          "stackyard-policy",
		"Description":   "stackyard seeded policy",
		"CreatedAt":     time.Now().UTC().Format(time.RFC3339),
		"UpdatedAt":     time.Now().UTC().Format(time.RFC3339),
		"LatestVersion": "1",
	}
	s.policies[arn] = item
	return item
}

func (s *mpaStore) ensurePolicyVersionLocked(versionArn, policyArn string) map[string]any {
	versionArn = strings.TrimSpace(versionArn)
	if versionArn == "" {
		versionArn = "arn:aws:mpa:us-east-1:123456789012:policy-version/stackyard/1"
	}
	if version := s.policyVersions[versionArn]; version != nil {
		return version
	}
	if strings.TrimSpace(policyArn) == "" {
		policyArn = "arn:aws:mpa:us-east-1:123456789012:policy/stackyard"
	}
	_ = s.ensurePolicyLocked(policyArn)
	item := map[string]any{
		"PolicyVersionArn": versionArn,
		"PolicyArn":        policyArn,
		"VersionId":        "1",
		"Document":         `{"Version":"2012-10-17","Statement":[]}`,
		"CreatedAt":        time.Now().UTC().Format(time.RFC3339),
	}
	s.policyVersions[versionArn] = item
	return item
}

func (s *mpaStore) ensureResourcePolicyLocked(resourceArn string) map[string]any {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = "arn:aws:mpa:us-east-1:123456789012:resource/stackyard"
	}
	if policy := s.resourcePolicy[resourceArn]; policy != nil {
		return policy
	}
	item := map[string]any{
		"ResourceArn": resourceArn,
		"Policy":      `{"Version":"2012-10-17","Statement":[]}`,
		"UpdatedAt":   time.Now().UTC().Format(time.RFC3339),
	}
	s.resourcePolicy[resourceArn] = item
	return item
}

func mpaDefaultString(values map[string]string, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}

func mpaDefaultStringAny(values map[string]any, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if trimmed := strings.TrimSpace(fmt.Sprintf("%v", v)); trimmed != "" && trimmed != "<nil>" {
				return trimmed
			}
		}
	}
	return fallback
}

func mpaSortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func mpaCloneAnyMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mpaCloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mpaMapString(values map[string]any, key string) map[string]string {
	for k, raw := range values {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		switch tags := raw.(type) {
		case map[string]any:
			out := map[string]string{}
			for tk, tv := range tags {
				out[strings.TrimSpace(tk)] = strings.TrimSpace(fmt.Sprintf("%v", tv))
			}
			return out
		case map[string]string:
			out := map[string]string{}
			for tk, tv := range tags {
				out[strings.TrimSpace(tk)] = strings.TrimSpace(tv)
			}
			return out
		}
	}
	return map[string]string{}
}

func mpaStringSlice(values map[string]any, key string) []string {
	for k, raw := range values {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		switch list := raw.(type) {
		case []any:
			out := make([]string, 0, len(list))
			for _, value := range list {
				if item := strings.TrimSpace(fmt.Sprintf("%v", value)); item != "" && item != "<nil>" {
					out = append(out, item)
				}
			}
			return out
		case []string:
			out := make([]string, 0, len(list))
			for _, value := range list {
				if item := strings.TrimSpace(value); item != "" {
					out = append(out, item)
				}
			}
			return out
		}
	}
	return nil
}
