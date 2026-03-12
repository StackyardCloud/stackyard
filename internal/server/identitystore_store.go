package server

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type identityStoreStore struct {
	mu sync.Mutex

	identityStoreID  string
	nextUserID       int64
	nextGroupID      int64
	nextMembershipID int64

	users       map[string]map[string]any
	groups      map[string]map[string]any
	memberships map[string]map[string]any
	userByName  map[string]string
	groupByName map[string]string
}

const (
	identityStoreSeedUserID       = "0000000011-00000000-0000-0000-0000-000000000011"
	identityStoreSeedGroupID      = "0000000012-00000000-0000-0000-0000-000000000012"
	identityStoreSeedMembershipID = "0000000013-00000000-0000-0000-0000-000000000013"
)

func newIdentityStoreStore() *identityStoreStore {
	s := &identityStoreStore{
		identityStoreID:  "d-1234567890",
		nextUserID:       2,
		nextGroupID:      2,
		nextMembershipID: 2,
		users:            map[string]map[string]any{},
		groups:           map[string]map[string]any{},
		memberships:      map[string]map[string]any{},
		userByName:       map[string]string{},
		groupByName:      map[string]string{},
	}

	userID := identityStoreSeedUserID
	groupID := identityStoreSeedGroupID
	membershipID := identityStoreSeedMembershipID

	s.users[userID] = map[string]any{
		"UserName":          "stackyard.user",
		"UserId":            userID,
		"ExternalIds":       []any{},
		"Name":              map[string]any{"GivenName": "Stackyard", "FamilyName": "User"},
		"DisplayName":       "Stackyard User",
		"NickName":          "",
		"ProfileUrl":        "",
		"Emails":            []any{},
		"Addresses":         []any{},
		"PhoneNumbers":      []any{},
		"UserType":          "",
		"Title":             "",
		"PreferredLanguage": "en-US",
		"Locale":            "en-US",
		"Timezone":          "UTC",
		"IdentityStoreId":   s.identityStoreID,
	}
	s.userByName[strings.ToLower("stackyard.user")] = userID

	s.groups[groupID] = map[string]any{
		"GroupId":         groupID,
		"DisplayName":     "stackyard-group",
		"ExternalIds":     []any{},
		"Description":     "seed group",
		"IdentityStoreId": s.identityStoreID,
	}
	s.groupByName[strings.ToLower("stackyard-group")] = groupID

	s.memberships[membershipID] = map[string]any{
		"IdentityStoreId": s.identityStoreID,
		"MembershipId":    membershipID,
		"GroupId":         groupID,
		"MemberId":        map[string]any{"UserId": userID},
	}

	return s
}

func (s *identityStoreStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	identityStoreID := identityStorePayloadString(payload, "IdentityStoreId", s.identityStoreID)
	groupID := identityStorePayloadString(payload, "GroupId", identityStoreSeedGroupID)
	userID := identityStorePayloadString(payload, "UserId", identityStoreSeedUserID)
	membershipID := identityStorePayloadString(payload, "MembershipId", identityStoreSeedMembershipID)
	memberUserID := identityStoreMemberUserID(payload, "MemberId", userID)

	switch action {
	case "CreateGroup":
		displayName := identityStorePayloadString(payload, "DisplayName", "stackyard-group")
		description := identityStorePayloadString(payload, "Description", "")
		id := s.groupByName[strings.ToLower(displayName)]
		if id == "" {
			id = s.nextGroupIdentifierLocked()
		}
		group := s.ensureGroupLocked(id, displayName)
		group["DisplayName"] = displayName
		group["Description"] = description
		group["IdentityStoreId"] = identityStoreID
		s.groupByName[strings.ToLower(displayName)] = id
		return map[string]any{"GroupId": id, "IdentityStoreId": identityStoreID}

	case "CreateGroupMembership":
		group := s.ensureGroupLocked(groupID, "")
		groupID = identityStorePayloadString(group, "GroupId", groupID)
		user := s.ensureUserLocked(memberUserID, "")
		memberUserID = identityStorePayloadString(user, "UserId", memberUserID)
		if existing := s.findMembershipByGroupUserLocked(groupID, memberUserID); existing != "" {
			return map[string]any{"MembershipId": existing, "IdentityStoreId": identityStoreID}
		}
		id := s.nextMembershipIdentifierLocked()
		s.memberships[id] = map[string]any{
			"IdentityStoreId": identityStoreID,
			"MembershipId":    id,
			"GroupId":         groupID,
			"MemberId":        map[string]any{"UserId": memberUserID},
		}
		return map[string]any{"MembershipId": id, "IdentityStoreId": identityStoreID}

	case "CreateUser":
		userName := identityStorePayloadString(payload, "UserName", fmt.Sprintf("stackyard.user.%d", s.nextUserID))
		displayName := identityStorePayloadString(payload, "DisplayName", userName)
		id := s.userByName[strings.ToLower(userName)]
		if id == "" {
			id = s.nextUserIdentifierLocked()
		}
		user := s.ensureUserLocked(id, userName)
		user["UserName"] = userName
		user["DisplayName"] = displayName
		user["IdentityStoreId"] = identityStoreID
		if name, ok := payloadCaseInsensitiveMap(payload, "Name"); ok && len(name) > 0 {
			user["Name"] = cloneAnyMap(name)
		}
		s.userByName[strings.ToLower(userName)] = id
		return map[string]any{"UserId": id, "IdentityStoreId": identityStoreID}

	case "DeleteGroup":
		delete(s.groups, groupID)
		for key, value := range s.groupByName {
			if value == groupID {
				delete(s.groupByName, key)
			}
		}
		for id, membership := range s.memberships {
			if identityStorePayloadString(membership, "GroupId", "") == groupID {
				delete(s.memberships, id)
			}
		}
		return map[string]any{}

	case "DeleteGroupMembership":
		delete(s.memberships, membershipID)
		return map[string]any{}

	case "DeleteUser":
		delete(s.users, userID)
		for key, value := range s.userByName {
			if value == userID {
				delete(s.userByName, key)
			}
		}
		for id, membership := range s.memberships {
			if identityStoreMemberUserID(membership, "MemberId", "") == userID {
				delete(s.memberships, id)
			}
		}
		return map[string]any{}

	case "DescribeGroup":
		group := s.ensureGroupLocked(groupID, "")
		group["IdentityStoreId"] = identityStoreID
		return identityStoreCloneMap(group)

	case "DescribeGroupMembership":
		membership := s.ensureMembershipLocked(membershipID, groupID, memberUserID)
		membership["IdentityStoreId"] = identityStoreID
		return identityStoreCloneMap(membership)

	case "DescribeUser":
		user := s.ensureUserLocked(userID, "")
		user["IdentityStoreId"] = identityStoreID
		return identityStoreCloneMap(user)

	case "GetGroupId":
		value := s.alternateIdentifierValue(payload, "displayName")
		if strings.TrimSpace(value) == "" {
			value = "stackyard-group"
		}
		id := s.groupByName[strings.ToLower(value)]
		if id == "" {
			id = s.nextGroupIdentifierLocked()
			group := s.ensureGroupLocked(id, value)
			group["DisplayName"] = value
		}
		return map[string]any{"GroupId": id, "IdentityStoreId": identityStoreID}

	case "GetGroupMembershipId":
		s.ensureGroupLocked(groupID, "")
		s.ensureUserLocked(memberUserID, "")
		id := s.findMembershipByGroupUserLocked(groupID, memberUserID)
		if id == "" {
			id = s.nextMembershipIdentifierLocked()
			s.memberships[id] = map[string]any{
				"IdentityStoreId": identityStoreID,
				"MembershipId":    id,
				"GroupId":         groupID,
				"MemberId":        map[string]any{"UserId": memberUserID},
			}
		}
		return map[string]any{"MembershipId": id, "IdentityStoreId": identityStoreID}

	case "GetUserId":
		value := s.alternateIdentifierValue(payload, "userName")
		if strings.TrimSpace(value) == "" {
			value = "stackyard.user"
		}
		id := s.userByName[strings.ToLower(value)]
		if id == "" {
			id = s.nextUserIdentifierLocked()
			user := s.ensureUserLocked(id, value)
			user["UserName"] = value
		}
		return map[string]any{"UserId": id, "IdentityStoreId": identityStoreID}

	case "IsMemberInGroups":
		groupIDs := identityStoreGroupIDs(payload)
		if len(groupIDs) == 0 {
			groupIDs = []string{groupID}
		}
		results := make([]any, 0, len(groupIDs))
		for _, gid := range groupIDs {
			exists := s.findMembershipByGroupUserLocked(gid, memberUserID) != ""
			results = append(results, map[string]any{
				"GroupId":          gid,
				"MemberId":         map[string]any{"UserId": memberUserID},
				"MembershipExists": exists,
			})
		}
		return map[string]any{"Results": results}

	case "ListGroupMemberships":
		items := make([]any, 0)
		for _, membership := range s.sortedMembershipsLocked() {
			if identityStorePayloadString(membership, "GroupId", "") != groupID {
				continue
			}
			items = append(items, identityStoreCloneMap(membership))
		}
		return map[string]any{"GroupMemberships": items, "NextToken": ""}

	case "ListGroupMembershipsForMember":
		items := make([]any, 0)
		for _, membership := range s.sortedMembershipsLocked() {
			if identityStoreMemberUserID(membership, "MemberId", "") != memberUserID {
				continue
			}
			items = append(items, identityStoreCloneMap(membership))
		}
		return map[string]any{"GroupMemberships": items, "NextToken": ""}

	case "ListGroups":
		items := make([]any, 0, len(s.groups))
		for _, group := range s.sortedGroupsLocked() {
			items = append(items, identityStoreCloneMap(group))
		}
		return map[string]any{"Groups": items, "NextToken": ""}

	case "ListUsers":
		items := make([]any, 0, len(s.users))
		for _, user := range s.sortedUsersLocked() {
			items = append(items, identityStoreCloneMap(user))
		}
		return map[string]any{"Users": items, "NextToken": ""}

	case "UpdateGroup":
		group := s.ensureGroupLocked(groupID, "")
		for _, op := range identityStoreOperationsList(payload) {
			path := strings.TrimSpace(identityStorePayloadString(op, "AttributePath", ""))
			if strings.EqualFold(path, "DisplayName") {
				oldName := identityStorePayloadString(group, "DisplayName", "")
				newName := identityStoreStringify(op["AttributeValue"])
				if strings.TrimSpace(newName) != "" {
					group["DisplayName"] = newName
					if oldName != "" {
						delete(s.groupByName, strings.ToLower(oldName))
					}
					s.groupByName[strings.ToLower(newName)] = groupID
				}
				continue
			}
			if strings.EqualFold(path, "Description") {
				group["Description"] = identityStoreStringify(op["AttributeValue"])
			}
		}
		group["IdentityStoreId"] = identityStoreID
		return map[string]any{}

	case "UpdateUser":
		user := s.ensureUserLocked(userID, "")
		for _, op := range identityStoreOperationsList(payload) {
			path := strings.TrimSpace(identityStorePayloadString(op, "AttributePath", ""))
			value := op["AttributeValue"]
			switch {
			case strings.EqualFold(path, "DisplayName"):
				user["DisplayName"] = identityStoreStringify(value)
			case strings.EqualFold(path, "UserName"):
				oldName := identityStorePayloadString(user, "UserName", "")
				newName := identityStoreStringify(value)
				if strings.TrimSpace(newName) != "" {
					user["UserName"] = newName
					if oldName != "" {
						delete(s.userByName, strings.ToLower(oldName))
					}
					s.userByName[strings.ToLower(newName)] = userID
				}
			case strings.EqualFold(path, "Name.GivenName"):
				name, _ := user["Name"].(map[string]any)
				if name == nil {
					name = map[string]any{}
				}
				name["GivenName"] = identityStoreStringify(value)
				user["Name"] = name
			case strings.EqualFold(path, "Name.FamilyName"):
				name, _ := user["Name"].(map[string]any)
				if name == nil {
					name = map[string]any{}
				}
				name["FamilyName"] = identityStoreStringify(value)
				user["Name"] = name
			}
		}
		user["IdentityStoreId"] = identityStoreID
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *identityStoreStore) nextUserIdentifierLocked() string {
	id := identityStoreResourceID(1, s.nextUserID)
	s.nextUserID++
	return id
}

func (s *identityStoreStore) nextGroupIdentifierLocked() string {
	id := identityStoreResourceID(2, s.nextGroupID)
	s.nextGroupID++
	return id
}

func (s *identityStoreStore) nextMembershipIdentifierLocked() string {
	id := identityStoreResourceID(3, s.nextMembershipID)
	s.nextMembershipID++
	return id
}

func (s *identityStoreStore) ensureUserLocked(userID, userName string) map[string]any {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		userID = identityStoreSeedUserID
	}
	if user, ok := s.users[userID]; ok {
		return user
	}
	if strings.TrimSpace(userName) == "" {
		userName = "stackyard.user"
	}
	user := map[string]any{
		"UserName":          userName,
		"UserId":            userID,
		"ExternalIds":       []any{},
		"Name":              map[string]any{"GivenName": "Stackyard", "FamilyName": "User"},
		"DisplayName":       userName,
		"NickName":          "",
		"ProfileUrl":        "",
		"Emails":            []any{},
		"Addresses":         []any{},
		"PhoneNumbers":      []any{},
		"UserType":          "",
		"Title":             "",
		"PreferredLanguage": "en-US",
		"Locale":            "en-US",
		"Timezone":          "UTC",
		"IdentityStoreId":   s.identityStoreID,
	}
	s.users[userID] = user
	s.userByName[strings.ToLower(userName)] = userID
	return user
}

func (s *identityStoreStore) ensureGroupLocked(groupID, displayName string) map[string]any {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		groupID = identityStoreSeedGroupID
	}
	if group, ok := s.groups[groupID]; ok {
		return group
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = "stackyard-group"
	}
	group := map[string]any{
		"GroupId":         groupID,
		"DisplayName":     displayName,
		"ExternalIds":     []any{},
		"Description":     "",
		"IdentityStoreId": s.identityStoreID,
	}
	s.groups[groupID] = group
	s.groupByName[strings.ToLower(displayName)] = groupID
	return group
}

func (s *identityStoreStore) ensureMembershipLocked(membershipID, groupID, userID string) map[string]any {
	membershipID = strings.TrimSpace(membershipID)
	if membershipID == "" {
		membershipID = identityStoreSeedMembershipID
	}
	if membership, ok := s.memberships[membershipID]; ok {
		return membership
	}
	group := s.ensureGroupLocked(groupID, "")
	user := s.ensureUserLocked(userID, "")
	membership := map[string]any{
		"IdentityStoreId": s.identityStoreID,
		"MembershipId":    membershipID,
		"GroupId":         identityStorePayloadString(group, "GroupId", groupID),
		"MemberId":        map[string]any{"UserId": identityStorePayloadString(user, "UserId", userID)},
	}
	s.memberships[membershipID] = membership
	return membership
}

func identityStoreResourceID(namespace, counter int64) string {
	token := (counter << 4) | (namespace & 0xf)
	return fmt.Sprintf("%010x-00000000-0000-0000-0000-%012x", token, token)
}

func (s *identityStoreStore) findMembershipByGroupUserLocked(groupID, userID string) string {
	for membershipID, membership := range s.memberships {
		if identityStorePayloadString(membership, "GroupId", "") == groupID &&
			identityStoreMemberUserID(membership, "MemberId", "") == userID {
			return membershipID
		}
	}
	return ""
}

func (s *identityStoreStore) sortedUsersLocked() []map[string]any {
	keys := make([]string, 0, len(s.users))
	for key := range s.users {
		keys = append(keys, key)
	}
	identityStoreSortStrings(keys)
	items := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		items = append(items, s.users[key])
	}
	return items
}

func (s *identityStoreStore) sortedGroupsLocked() []map[string]any {
	keys := make([]string, 0, len(s.groups))
	for key := range s.groups {
		keys = append(keys, key)
	}
	identityStoreSortStrings(keys)
	items := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		items = append(items, s.groups[key])
	}
	return items
}

func (s *identityStoreStore) sortedMembershipsLocked() []map[string]any {
	keys := make([]string, 0, len(s.memberships))
	for key := range s.memberships {
		keys = append(keys, key)
	}
	identityStoreSortStrings(keys)
	items := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		items = append(items, s.memberships[key])
	}
	return items
}

func (s *identityStoreStore) alternateIdentifierValue(payload map[string]any, defaultPath string) string {
	alternate, ok := payloadCaseInsensitiveMap(payload, "AlternateIdentifier")
	if !ok {
		return ""
	}
	uniqueAttribute, ok := payloadCaseInsensitiveMap(alternate, "UniqueAttribute")
	if !ok {
		return ""
	}
	path := identityStorePayloadString(uniqueAttribute, "AttributePath", defaultPath)
	value := uniqueAttribute["AttributeValue"]
	stringValue := identityStoreStringify(value)
	if strings.TrimSpace(path) == "" {
		return stringValue
	}
	return stringValue
}

func identityStorePayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, value := range payload {
		if strings.EqualFold(k, key) {
			return identityStoreStringify(value)
		}
	}
	return fallback
}

func identityStoreMemberUserID(payload map[string]any, key, fallback string) string {
	memberID, ok := payloadCaseInsensitiveMap(payload, key)
	if !ok {
		return fallback
	}
	return identityStorePayloadString(memberID, "UserId", fallback)
}

func identityStoreGroupIDs(payload map[string]any) []string {
	for key, value := range payload {
		if !strings.EqualFold(key, "GroupIds") {
			continue
		}
		switch list := value.(type) {
		case []any:
			out := make([]string, 0, len(list))
			for _, item := range list {
				id := strings.TrimSpace(identityStoreStringify(item))
				if id != "" {
					out = append(out, id)
				}
			}
			return out
		case []string:
			out := make([]string, 0, len(list))
			for _, item := range list {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
			return out
		}
	}
	return nil
}

func identityStoreOperationsList(payload map[string]any) []map[string]any {
	for key, value := range payload {
		if !strings.EqualFold(key, "Operations") {
			continue
		}
		list, ok := value.([]any)
		if !ok {
			return nil
		}
		out := make([]map[string]any, 0, len(list))
		for _, item := range list {
			if op, ok := item.(map[string]any); ok {
				out = append(out, op)
			}
		}
		return out
	}
	return nil
}

func identityStoreStringify(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func identityStoreCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch v := value.(type) {
		case map[string]any:
			out[key] = identityStoreCloneMap(v)
		case []any:
			out[key] = identityStoreCloneSlice(v)
		default:
			out[key] = value
		}
	}
	return out
}

func identityStoreCloneSlice(in []any) []any {
	out := make([]any, 0, len(in))
	for _, value := range in {
		switch v := value.(type) {
		case map[string]any:
			out = append(out, identityStoreCloneMap(v))
		case []any:
			out = append(out, identityStoreCloneSlice(v))
		default:
			out = append(out, value)
		}
	}
	return out
}

func identityStoreSortStrings(values []string) {
	if len(values) < 2 {
		return
	}
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}
