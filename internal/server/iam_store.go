package server

import (
	"net/url"
	"strings"
	"sync"
)

type iamStore struct {
	mu       sync.Mutex
	users    map[string]map[string]any
	groups   map[string]map[string]any
	roles    map[string]map[string]any
	policies map[string]map[string]any
}

func newIAMStore() *iamStore {
	return &iamStore{
		users: map[string]map[string]any{
			"stackyard": {
				"Path":       "/",
				"UserName":   "stackyard",
				"UserId":     "AIDASTACKYARDUSER",
				"Arn":        "arn:aws:iam::123456789012:user/stackyard",
				"CreateDate": "2026-01-01T00:00:00Z",
			},
		},
		groups: map[string]map[string]any{
			"stackyard-admins": {
				"Path":       "/",
				"GroupName":  "stackyard-admins",
				"GroupId":    "AGPASTACKYARDGRP",
				"Arn":        "arn:aws:iam::123456789012:group/stackyard-admins",
				"CreateDate": "2026-01-01T00:00:00Z",
			},
		},
		roles: map[string]map[string]any{
			"stackyard-role": {
				"Path":                     "/",
				"RoleName":                 "stackyard-role",
				"RoleId":                   "AROASTACKYARDROLE",
				"Arn":                      "arn:aws:iam::123456789012:role/stackyard-role",
				"CreateDate":               "2026-01-01T00:00:00Z",
				"AssumeRolePolicyDocument": "{}",
			},
		},
		policies: map[string]map[string]any{
			"arn:aws:iam::123456789012:policy/stackyard-policy": {
				"PolicyName":       "stackyard-policy",
				"PolicyId":         "ANPASTACKYARDPOL",
				"Arn":              "arn:aws:iam::123456789012:policy/stackyard-policy",
				"Path":             "/",
				"DefaultVersionId": "v1",
			},
		},
	}
}

func (s *iamStore) Handle(action string, form url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "GetAccountSummary":
		return map[string]any{
			"SummaryMap": map[string]any{
				"Users":             len(s.users),
				"Groups":            len(s.groups),
				"Roles":             len(s.roles),
				"Policies":          len(s.policies),
				"AccountMFAEnabled": 1,
			},
		}
	case "ListUsers":
		return map[string]any{
			"Users":       map[string]any{"member": iamMapValues(s.users)},
			"IsTruncated": false,
		}
	case "ListGroups":
		return map[string]any{
			"Groups":      map[string]any{"member": iamMapValues(s.groups)},
			"IsTruncated": false,
		}
	case "ListRoles":
		return map[string]any{
			"Roles":       map[string]any{"member": iamMapValues(s.roles)},
			"IsTruncated": false,
		}
	case "ListPolicies":
		return map[string]any{
			"Policies":    map[string]any{"member": iamMapValues(s.policies)},
			"IsTruncated": false,
		}
	case "GetUser":
		name := strings.TrimSpace(form.Get("UserName"))
		if name == "" {
			name = "stackyard"
		}
		user := s.users[name]
		if user == nil {
			user = map[string]any{
				"Path":       "/",
				"UserName":   name,
				"UserId":     "AIDASTACKYARDUSER",
				"Arn":        "arn:aws:iam::123456789012:user/" + name,
				"CreateDate": "2026-01-01T00:00:00Z",
			}
			s.users[name] = user
		}
		return map[string]any{"User": user}
	case "GetGroup":
		name := strings.TrimSpace(form.Get("GroupName"))
		if name == "" {
			name = "stackyard-admins"
		}
		group := s.groups[name]
		if group == nil {
			group = map[string]any{
				"Path":       "/",
				"GroupName":  name,
				"GroupId":    "AGPASTACKYARDGRP",
				"Arn":        "arn:aws:iam::123456789012:group/" + name,
				"CreateDate": "2026-01-01T00:00:00Z",
			}
			s.groups[name] = group
		}
		return map[string]any{
			"Group":       group,
			"Users":       map[string]any{"member": []any{}},
			"IsTruncated": false,
		}
	case "GetRole":
		name := strings.TrimSpace(form.Get("RoleName"))
		if name == "" {
			name = "stackyard-role"
		}
		role := s.roles[name]
		if role == nil {
			role = map[string]any{
				"Path":                     "/",
				"RoleName":                 name,
				"RoleId":                   "AROASTACKYARDROLE",
				"Arn":                      "arn:aws:iam::123456789012:role/" + name,
				"CreateDate":               "2026-01-01T00:00:00Z",
				"AssumeRolePolicyDocument": "{}",
			}
			s.roles[name] = role
		}
		return map[string]any{"Role": role}
	case "ListAccountAliases":
		return map[string]any{
			"AccountAliases": map[string]any{"member": []any{"stackyard"}},
			"IsTruncated":    false,
		}
	default:
		return map[string]any{}
	}
}

func iamMapValues(src map[string]map[string]any) []any {
	out := make([]any, 0, len(src))
	for _, item := range src {
		out = append(out, item)
	}
	return out
}
