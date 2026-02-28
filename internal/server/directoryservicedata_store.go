package server

import (
	"net/url"
	"sort"
	"strings"
	"sync"
)

type directoryServiceDataStore struct {
	mu sync.Mutex

	users      map[string]map[string]any
	groups     map[string]map[string]any
	groupUsers map[string]map[string]struct{}
	userGroups map[string]map[string]struct{}
}

func newDirectoryServiceDataStore() *directoryServiceDataStore {
	s := &directoryServiceDataStore{
		users:      map[string]map[string]any{},
		groups:     map[string]map[string]any{},
		groupUsers: map[string]map[string]struct{}{},
		userGroups: map[string]map[string]struct{}{},
	}

	s.users["stackyard"] = map[string]any{
		"SAMAccountName": "stackyard",
		"EmailAddress":   "stackyard@example.com",
		"GivenName":      "Stack",
		"Surname":        "Yard",
		"Enabled":        true,
	}
	s.groups["Admins"] = map[string]any{
		"SAMAccountName": "Admins",
		"DisplayName":    "Administrators",
		"Description":    "Seed admin group",
	}
	s.groupUsers["Admins"] = map[string]struct{}{"stackyard": {}}
	s.userGroups["stackyard"] = map[string]struct{}{"Admins": {}}

	return s
}

func (s *directoryServiceDataStore) Handle(action string, payload map[string]any, _ map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	directoryID := dsdFirstNonEmpty(dsdStringAny(payload, "DirectoryId"), "d-0000000000")
	user := dsdFirstNonEmpty(
		dsdStringAny(payload, "SAMAccountName"),
		dsdStringAny(payload, "UserName"),
		dsdStringAny(payload, "MemberName"),
		"stackyard",
	)
	group := dsdFirstNonEmpty(
		dsdStringAny(payload, "GroupName"),
		dsdStringAny(payload, "SAMAccountName"),
		dsdStringAny(payload, "ParentGroupName"),
		"Admins",
	)

	s.ensureUserLocked(user)
	s.ensureGroupLocked(group)

	switch action {
	case "CreateUser":
		u := s.ensureUserLocked(user)
		for k, v := range payload {
			u[k] = v
		}
		u["SAMAccountName"] = user
		u["Enabled"] = true
		return map[string]any{"DirectoryId": directoryID, "User": dsdCloneMap(u)}

	case "UpdateUser":
		u := s.ensureUserLocked(user)
		for k, v := range payload {
			u[k] = v
		}
		u["SAMAccountName"] = user
		return map[string]any{"DirectoryId": directoryID, "User": dsdCloneMap(u)}

	case "DeleteUser":
		delete(s.users, user)
		for g := range s.groups {
			delete(s.groupUsers[g], user)
		}
		delete(s.userGroups, user)
		return map[string]any{}

	case "DisableUser":
		u := s.ensureUserLocked(user)
		u["Enabled"] = false
		return map[string]any{"DirectoryId": directoryID, "User": dsdCloneMap(u)}

	case "DescribeUser":
		return map[string]any{"DirectoryId": directoryID, "User": dsdCloneMap(s.ensureUserLocked(user))}

	case "ListUsers":
		items := make([]any, 0, len(s.users))
		for _, u := range s.sortedUsersLocked() {
			items = append(items, map[string]any{
				"SAMAccountName": dsdStringAny(u, "SAMAccountName"),
				"GivenName":      dsdStringAny(u, "GivenName"),
				"Surname":        dsdStringAny(u, "Surname"),
				"Enabled":        dsdBoolAny(u, "Enabled", true),
			})
		}
		return map[string]any{"DirectoryId": directoryID, "Users": items, "NextToken": ""}

	case "SearchUsers":
		items := make([]any, 0, len(s.users))
		needle := strings.ToLower(strings.TrimSpace(dsdStringAny(payload, "SearchString")))
		for _, u := range s.sortedUsersLocked() {
			sam := strings.ToLower(dsdStringAny(u, "SAMAccountName"))
			given := strings.ToLower(dsdStringAny(u, "GivenName"))
			surname := strings.ToLower(dsdStringAny(u, "Surname"))
			if needle != "" && !strings.Contains(sam, needle) && !strings.Contains(given, needle) && !strings.Contains(surname, needle) {
				continue
			}
			items = append(items, dsdCloneMap(u))
		}
		return map[string]any{"DirectoryId": directoryID, "Users": items, "NextToken": ""}

	case "CreateGroup":
		g := s.ensureGroupLocked(group)
		for k, v := range payload {
			g[k] = v
		}
		g["SAMAccountName"] = group
		return map[string]any{"DirectoryId": directoryID, "Group": dsdCloneMap(g)}

	case "UpdateGroup":
		g := s.ensureGroupLocked(group)
		for k, v := range payload {
			g[k] = v
		}
		g["SAMAccountName"] = group
		return map[string]any{"DirectoryId": directoryID, "Group": dsdCloneMap(g)}

	case "DeleteGroup":
		delete(s.groups, group)
		for u := range s.users {
			delete(s.userGroups[u], group)
		}
		delete(s.groupUsers, group)
		return map[string]any{}

	case "DescribeGroup":
		return map[string]any{"DirectoryId": directoryID, "Group": dsdCloneMap(s.ensureGroupLocked(group))}

	case "ListGroups":
		items := make([]any, 0, len(s.groups))
		for _, g := range s.sortedGroupsLocked() {
			items = append(items, map[string]any{
				"SAMAccountName": dsdStringAny(g, "SAMAccountName"),
				"DisplayName":    dsdStringAny(g, "DisplayName"),
			})
		}
		return map[string]any{"DirectoryId": directoryID, "Groups": items, "NextToken": ""}

	case "SearchGroups":
		items := make([]any, 0, len(s.groups))
		needle := strings.ToLower(strings.TrimSpace(dsdStringAny(payload, "SearchString")))
		for _, g := range s.sortedGroupsLocked() {
			sam := strings.ToLower(dsdStringAny(g, "SAMAccountName"))
			display := strings.ToLower(dsdStringAny(g, "DisplayName"))
			if needle != "" && !strings.Contains(sam, needle) && !strings.Contains(display, needle) {
				continue
			}
			items = append(items, dsdCloneMap(g))
		}
		return map[string]any{"DirectoryId": directoryID, "Groups": items, "NextToken": ""}

	case "AddGroupMember":
		s.ensureMembershipLocked(group, user)
		return map[string]any{}

	case "RemoveGroupMember":
		if members, ok := s.groupUsers[group]; ok {
			delete(members, user)
		}
		if groups, ok := s.userGroups[user]; ok {
			delete(groups, group)
		}
		return map[string]any{}

	case "ListGroupMembers":
		items := make([]any, 0)
		for _, u := range s.sortedGroupMembersLocked(group) {
			items = append(items, map[string]any{"SAMAccountName": u})
		}
		return map[string]any{"DirectoryId": directoryID, "Members": items, "NextToken": ""}

	case "ListGroupsForMember":
		items := make([]any, 0)
		for _, g := range s.sortedUserGroupsLocked(user) {
			items = append(items, map[string]any{"SAMAccountName": g})
		}
		return map[string]any{"DirectoryId": directoryID, "Groups": items, "NextToken": ""}
	}

	return map[string]any{}
}

func (s *directoryServiceDataStore) ensureUserLocked(user string) map[string]any {
	user = strings.TrimSpace(user)
	if user == "" {
		user = "stackyard"
	}
	if u, ok := s.users[user]; ok {
		return u
	}
	u := map[string]any{
		"SAMAccountName": user,
		"GivenName":      "User",
		"Surname":        user,
		"Enabled":        true,
	}
	s.users[user] = u
	return u
}

func (s *directoryServiceDataStore) ensureGroupLocked(group string) map[string]any {
	group = strings.TrimSpace(group)
	if group == "" {
		group = "Admins"
	}
	if g, ok := s.groups[group]; ok {
		return g
	}
	g := map[string]any{
		"SAMAccountName": group,
		"DisplayName":    group,
	}
	s.groups[group] = g
	return g
}

func (s *directoryServiceDataStore) ensureMembershipLocked(group, user string) {
	if _, ok := s.groupUsers[group]; !ok {
		s.groupUsers[group] = map[string]struct{}{}
	}
	s.groupUsers[group][user] = struct{}{}

	if _, ok := s.userGroups[user]; !ok {
		s.userGroups[user] = map[string]struct{}{}
	}
	s.userGroups[user][group] = struct{}{}
}

func (s *directoryServiceDataStore) sortedUsersLocked() []map[string]any {
	keys := make([]string, 0, len(s.users))
	for k := range s.users {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, s.users[k])
	}
	return out
}

func (s *directoryServiceDataStore) sortedGroupsLocked() []map[string]any {
	keys := make([]string, 0, len(s.groups))
	for k := range s.groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, s.groups[k])
	}
	return out
}

func (s *directoryServiceDataStore) sortedGroupMembersLocked(group string) []string {
	set := s.groupUsers[group]
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (s *directoryServiceDataStore) sortedUserGroupsLocked(user string) []string {
	set := s.userGroups[user]
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func dsdStringAny(m map[string]any, keys ...string) string {
	for _, key := range keys {
		for k, v := range m {
			if !strings.EqualFold(k, key) {
				continue
			}
			if s, ok := v.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func dsdBoolAny(m map[string]any, key string, def bool) bool {
	for k, v := range m {
		if !strings.EqualFold(k, key) {
			continue
		}
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func dsdFirstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func dsdCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
