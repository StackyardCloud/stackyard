package server

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

type iamStore struct {
	mu sync.Mutex

	users    map[string]map[string]any
	groups   map[string]map[string]any
	roles    map[string]map[string]any
	policies map[string]map[string]any

	groupUsers            map[string]map[string]struct{}
	userAttachedPolicies  map[string]map[string]struct{}
	groupAttachedPolicies map[string]map[string]struct{}
	roleAttachedPolicies  map[string]map[string]struct{}

	nextID int64
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
		groupUsers: map[string]map[string]struct{}{
			"stackyard-admins": {"stackyard": {}},
		},
		userAttachedPolicies: map[string]map[string]struct{}{
			"stackyard": {"arn:aws:iam::123456789012:policy/stackyard-policy": {}},
		},
		groupAttachedPolicies: map[string]map[string]struct{}{},
		roleAttachedPolicies: map[string]map[string]struct{}{
			"stackyard-role": {"arn:aws:iam::123456789012:policy/stackyard-policy": {}},
		},
		nextID: 1,
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
		user := s.getOrCreateUser(strings.TrimSpace(form.Get("UserName")))
		return map[string]any{"User": user}
	case "GetGroup":
		group := s.getOrCreateGroup(strings.TrimSpace(form.Get("GroupName")))
		users := s.usersInGroup(group["GroupName"].(string))
		return map[string]any{
			"Group":       group,
			"Users":       map[string]any{"member": users},
			"IsTruncated": false,
		}
	case "GetRole":
		role := s.getOrCreateRole(strings.TrimSpace(form.Get("RoleName")))
		return map[string]any{"Role": role}
	case "GetPolicy":
		arn := strings.TrimSpace(form.Get("PolicyArn"))
		if arn == "" {
			return map[string]any{}
		}
		policy := s.policies[arn]
		if policy == nil {
			policy = s.createPolicy("stackyard-policy", "/")
		}
		return map[string]any{"Policy": policy}
	case "ListAccountAliases":
		return map[string]any{
			"AccountAliases": map[string]any{"member": []any{"stackyard"}},
			"IsTruncated":    false,
		}

	case "CreateUser":
		name := strings.TrimSpace(form.Get("UserName"))
		path := strings.TrimSpace(form.Get("Path"))
		if path == "" {
			path = "/"
		}
		if name == "" {
			return map[string]any{}
		}
		user := s.users[name]
		if user == nil {
			user = map[string]any{
				"Path":       path,
				"UserName":   name,
				"UserId":     s.nextIdentifier("AIDAUSER"),
				"Arn":        "arn:aws:iam::123456789012:user/" + name,
				"CreateDate": "2026-01-01T00:00:00Z",
			}
			s.users[name] = user
		}
		return map[string]any{"User": user}
	case "DeleteUser":
		name := strings.TrimSpace(form.Get("UserName"))
		delete(s.users, name)
		delete(s.userAttachedPolicies, name)
		for _, members := range s.groupUsers {
			delete(members, name)
		}
		return map[string]any{}
	case "UpdateUser":
		oldName := strings.TrimSpace(form.Get("UserName"))
		newName := strings.TrimSpace(form.Get("NewUserName"))
		if oldName == "" || newName == "" || oldName == newName {
			return map[string]any{}
		}
		user := s.users[oldName]
		if user == nil {
			return map[string]any{}
		}
		delete(s.users, oldName)
		user["UserName"] = newName
		user["Arn"] = "arn:aws:iam::123456789012:user/" + newName
		s.users[newName] = user
		if attached, ok := s.userAttachedPolicies[oldName]; ok {
			delete(s.userAttachedPolicies, oldName)
			s.userAttachedPolicies[newName] = attached
		}
		for _, members := range s.groupUsers {
			if _, ok := members[oldName]; ok {
				delete(members, oldName)
				members[newName] = struct{}{}
			}
		}
		return map[string]any{}

	case "CreateGroup":
		name := strings.TrimSpace(form.Get("GroupName"))
		path := strings.TrimSpace(form.Get("Path"))
		if path == "" {
			path = "/"
		}
		if name == "" {
			return map[string]any{}
		}
		group := s.groups[name]
		if group == nil {
			group = map[string]any{
				"Path":       path,
				"GroupName":  name,
				"GroupId":    s.nextIdentifier("AGPA"),
				"Arn":        "arn:aws:iam::123456789012:group/" + name,
				"CreateDate": "2026-01-01T00:00:00Z",
			}
			s.groups[name] = group
		}
		return map[string]any{"Group": group}
	case "DeleteGroup":
		name := strings.TrimSpace(form.Get("GroupName"))
		delete(s.groups, name)
		delete(s.groupUsers, name)
		delete(s.groupAttachedPolicies, name)
		return map[string]any{}
	case "UpdateGroup":
		oldName := strings.TrimSpace(form.Get("GroupName"))
		newName := strings.TrimSpace(form.Get("NewGroupName"))
		if oldName == "" || newName == "" || oldName == newName {
			return map[string]any{}
		}
		group := s.groups[oldName]
		if group == nil {
			return map[string]any{}
		}
		delete(s.groups, oldName)
		group["GroupName"] = newName
		group["Arn"] = "arn:aws:iam::123456789012:group/" + newName
		s.groups[newName] = group
		if members, ok := s.groupUsers[oldName]; ok {
			delete(s.groupUsers, oldName)
			s.groupUsers[newName] = members
		}
		if attached, ok := s.groupAttachedPolicies[oldName]; ok {
			delete(s.groupAttachedPolicies, oldName)
			s.groupAttachedPolicies[newName] = attached
		}
		return map[string]any{}

	case "CreateRole":
		name := strings.TrimSpace(form.Get("RoleName"))
		path := strings.TrimSpace(form.Get("Path"))
		if path == "" {
			path = "/"
		}
		if name == "" {
			return map[string]any{}
		}
		role := s.roles[name]
		if role == nil {
			role = map[string]any{
				"Path":                     path,
				"RoleName":                 name,
				"RoleId":                   s.nextIdentifier("AROA"),
				"Arn":                      "arn:aws:iam::123456789012:role/" + name,
				"CreateDate":               "2026-01-01T00:00:00Z",
				"AssumeRolePolicyDocument": defaultIfEmpty(strings.TrimSpace(form.Get("AssumeRolePolicyDocument")), "{}"),
			}
			s.roles[name] = role
		}
		return map[string]any{"Role": role}
	case "DeleteRole":
		name := strings.TrimSpace(form.Get("RoleName"))
		delete(s.roles, name)
		delete(s.roleAttachedPolicies, name)
		return map[string]any{}
	case "UpdateRole":
		name := strings.TrimSpace(form.Get("RoleName"))
		if role := s.roles[name]; role != nil {
			if desc := strings.TrimSpace(form.Get("Description")); desc != "" {
				role["Description"] = desc
			}
			if maxSession := strings.TrimSpace(form.Get("MaxSessionDuration")); maxSession != "" {
				if n, err := strconv.Atoi(maxSession); err == nil {
					role["MaxSessionDuration"] = n
				}
			}
		}
		return map[string]any{}
	case "UpdateRoleDescription":
		name := strings.TrimSpace(form.Get("RoleName"))
		if role := s.roles[name]; role != nil {
			if desc := strings.TrimSpace(form.Get("Description")); desc != "" {
				role["Description"] = desc
			}
			return map[string]any{"Role": role}
		}
		return map[string]any{}

	case "CreatePolicy":
		name := strings.TrimSpace(form.Get("PolicyName"))
		path := strings.TrimSpace(form.Get("Path"))
		if path == "" {
			path = "/"
		}
		if name == "" {
			return map[string]any{}
		}
		policy := s.createPolicy(name, path)
		return map[string]any{"Policy": policy}
	case "DeletePolicy":
		arn := strings.TrimSpace(form.Get("PolicyArn"))
		delete(s.policies, arn)
		s.detachPolicyFromAll(arn)
		return map[string]any{}

	case "AddUserToGroup":
		userName := strings.TrimSpace(form.Get("UserName"))
		groupName := strings.TrimSpace(form.Get("GroupName"))
		if userName == "" || groupName == "" {
			return map[string]any{}
		}
		s.getOrCreateUser(userName)
		s.getOrCreateGroup(groupName)
		members := s.groupUsers[groupName]
		if members == nil {
			members = map[string]struct{}{}
			s.groupUsers[groupName] = members
		}
		members[userName] = struct{}{}
		return map[string]any{}
	case "RemoveUserFromGroup":
		userName := strings.TrimSpace(form.Get("UserName"))
		groupName := strings.TrimSpace(form.Get("GroupName"))
		if members := s.groupUsers[groupName]; members != nil {
			delete(members, userName)
		}
		return map[string]any{}
	case "ListGroupsForUser":
		userName := strings.TrimSpace(form.Get("UserName"))
		groups := s.groupsForUser(userName)
		return map[string]any{
			"Groups":      map[string]any{"member": groups},
			"IsTruncated": false,
		}

	case "AttachUserPolicy":
		s.attachPolicy(s.userAttachedPolicies, strings.TrimSpace(form.Get("UserName")), strings.TrimSpace(form.Get("PolicyArn")))
		return map[string]any{}
	case "DetachUserPolicy":
		s.detachPolicy(s.userAttachedPolicies, strings.TrimSpace(form.Get("UserName")), strings.TrimSpace(form.Get("PolicyArn")))
		return map[string]any{}
	case "ListAttachedUserPolicies":
		userName := strings.TrimSpace(form.Get("UserName"))
		return map[string]any{
			"AttachedPolicies": map[string]any{"member": s.listAttachedPolicies(s.userAttachedPolicies[userName])},
			"IsTruncated":      false,
		}

	case "AttachGroupPolicy":
		s.attachPolicy(s.groupAttachedPolicies, strings.TrimSpace(form.Get("GroupName")), strings.TrimSpace(form.Get("PolicyArn")))
		return map[string]any{}
	case "DetachGroupPolicy":
		s.detachPolicy(s.groupAttachedPolicies, strings.TrimSpace(form.Get("GroupName")), strings.TrimSpace(form.Get("PolicyArn")))
		return map[string]any{}
	case "ListAttachedGroupPolicies":
		groupName := strings.TrimSpace(form.Get("GroupName"))
		return map[string]any{
			"AttachedPolicies": map[string]any{"member": s.listAttachedPolicies(s.groupAttachedPolicies[groupName])},
			"IsTruncated":      false,
		}

	case "AttachRolePolicy":
		s.attachPolicy(s.roleAttachedPolicies, strings.TrimSpace(form.Get("RoleName")), strings.TrimSpace(form.Get("PolicyArn")))
		return map[string]any{}
	case "DetachRolePolicy":
		s.detachPolicy(s.roleAttachedPolicies, strings.TrimSpace(form.Get("RoleName")), strings.TrimSpace(form.Get("PolicyArn")))
		return map[string]any{}
	case "ListAttachedRolePolicies":
		roleName := strings.TrimSpace(form.Get("RoleName"))
		return map[string]any{
			"AttachedPolicies": map[string]any{"member": s.listAttachedPolicies(s.roleAttachedPolicies[roleName])},
			"IsTruncated":      false,
		}
	}

	return iamDefaultResponse(action)
}

func (s *iamStore) getOrCreateUser(name string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard"
	}
	user := s.users[name]
	if user == nil {
		user = map[string]any{
			"Path":       "/",
			"UserName":   name,
			"UserId":     s.nextIdentifier("AIDA"),
			"Arn":        "arn:aws:iam::123456789012:user/" + name,
			"CreateDate": "2026-01-01T00:00:00Z",
		}
		s.users[name] = user
	}
	return user
}

func (s *iamStore) getOrCreateGroup(name string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-admins"
	}
	group := s.groups[name]
	if group == nil {
		group = map[string]any{
			"Path":       "/",
			"GroupName":  name,
			"GroupId":    s.nextIdentifier("AGPA"),
			"Arn":        "arn:aws:iam::123456789012:group/" + name,
			"CreateDate": "2026-01-01T00:00:00Z",
		}
		s.groups[name] = group
	}
	return group
}

func (s *iamStore) getOrCreateRole(name string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-role"
	}
	role := s.roles[name]
	if role == nil {
		role = map[string]any{
			"Path":                     "/",
			"RoleName":                 name,
			"RoleId":                   s.nextIdentifier("AROA"),
			"Arn":                      "arn:aws:iam::123456789012:role/" + name,
			"CreateDate":               "2026-01-01T00:00:00Z",
			"AssumeRolePolicyDocument": "{}",
		}
		s.roles[name] = role
	}
	return role
}

func (s *iamStore) createPolicy(name, path string) map[string]any {
	arn := "arn:aws:iam::123456789012:policy/" + name
	if policy := s.policies[arn]; policy != nil {
		return policy
	}
	policy := map[string]any{
		"PolicyName":       name,
		"PolicyId":         s.nextIdentifier("ANPA"),
		"Arn":              arn,
		"Path":             path,
		"DefaultVersionId": "v1",
	}
	s.policies[arn] = policy
	return policy
}

func (s *iamStore) usersInGroup(groupName string) []any {
	memberSet := s.groupUsers[groupName]
	if len(memberSet) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(memberSet))
	for _, userName := range mapKeysSorted(memberSet) {
		if user := s.users[userName]; user != nil {
			out = append(out, user)
		}
	}
	return out
}

func (s *iamStore) groupsForUser(userName string) []any {
	out := make([]any, 0, len(s.groupUsers))
	for _, groupName := range mapKeysSorted(s.groupUsers) {
		if _, ok := s.groupUsers[groupName][userName]; !ok {
			continue
		}
		if group := s.groups[groupName]; group != nil {
			out = append(out, group)
		}
	}
	return out
}

func (s *iamStore) attachPolicy(store map[string]map[string]struct{}, principal, policyARN string) {
	if principal == "" || policyARN == "" {
		return
	}
	attached := store[principal]
	if attached == nil {
		attached = map[string]struct{}{}
		store[principal] = attached
	}
	attached[policyARN] = struct{}{}
}

func (s *iamStore) detachPolicy(store map[string]map[string]struct{}, principal, policyARN string) {
	if principal == "" || policyARN == "" {
		return
	}
	attached := store[principal]
	if attached == nil {
		return
	}
	delete(attached, policyARN)
}

func (s *iamStore) listAttachedPolicies(attached map[string]struct{}) []any {
	if len(attached) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(attached))
	for _, arn := range mapKeysSorted(attached) {
		policy := s.policies[arn]
		if policy == nil {
			policy = map[string]any{"PolicyName": arn[strings.LastIndex(arn, "/")+1:], "Arn": arn}
		}
		out = append(out, map[string]any{
			"PolicyName": policy["PolicyName"],
			"PolicyArn":  arn,
		})
	}
	return out
}

func (s *iamStore) detachPolicyFromAll(policyARN string) {
	for _, attached := range s.userAttachedPolicies {
		delete(attached, policyARN)
	}
	for _, attached := range s.groupAttachedPolicies {
		delete(attached, policyARN)
	}
	for _, attached := range s.roleAttachedPolicies {
		delete(attached, policyARN)
	}
}

func (s *iamStore) nextIdentifier(prefix string) string {
	id := s.nextID
	s.nextID++
	return prefix + fmt.Sprintf("%012d", id)
}

func iamDefaultResponse(action string) map[string]any {
	switch action {
	case "ListAccessKeys", "ListMFADevices", "ListSSHPublicKeys", "ListSigningCertificates",
		"ListUserPolicies", "ListRolePolicies", "ListGroupPolicies",
		"ListPolicyVersions", "ListPolicyTags", "ListRoleTags", "ListUserTags", "ListInstanceProfiles",
		"ListInstanceProfilesForRole", "ListServerCertificates", "ListServiceSpecificCredentials",
		"ListOpenIDConnectProviders", "ListOpenIDConnectProviderTags", "ListSAMLProviders", "ListSAMLProviderTags",
		"ListVirtualMFADevices", "ListEntitiesForPolicy", "ListPoliciesGrantingServiceAccess",
		"ListOrganizationsFeatures", "ListDelegationRequests", "ListInstanceProfileTags",
		"ListMFADeviceTags", "ListServerCertificateTags":
		return map[string]any{"IsTruncated": false}
	case "GetAccessKeyLastUsed":
		return map[string]any{"AccessKeyLastUsed": map[string]any{}}
	case "GetAccountPasswordPolicy":
		return map[string]any{"PasswordPolicy": map[string]any{}}
	case "GetCredentialReport":
		return map[string]any{"State": "COMPLETE", "GeneratedTime": "2026-01-01T00:00:00Z", "Content": ""}
	case "GenerateCredentialReport", "GenerateOrganizationsAccessReport":
		return map[string]any{"State": "COMPLETE", "Description": ""}
	case "GetServiceLinkedRoleDeletionStatus":
		return map[string]any{"Status": "SUCCEEDED"}
	case "GetOrganizationsAccessReport":
		return map[string]any{"JobStatus": "COMPLETED", "NumberOfServicesAccessible": 0}
	case "GetPolicyVersion":
		return map[string]any{"PolicyVersion": map[string]any{"VersionId": "v1", "IsDefaultVersion": true, "Document": "{}"}}
	case "GetRolePolicy", "GetUserPolicy", "GetGroupPolicy":
		return map[string]any{"PolicyDocument": "{}"}
	case "GetMFADevice", "GetHumanReadableSummary", "GetOutboundWebIdentityFederationInfo":
		return map[string]any{}
	case "GetInstanceProfile":
		return map[string]any{"InstanceProfile": map[string]any{}}
	case "GetLoginProfile":
		return map[string]any{"LoginProfile": map[string]any{}}
	case "GetOpenIDConnectProvider":
		return map[string]any{"ClientIDList": []any{}, "ThumbprintList": []any{}}
	case "GetSAMLProvider":
		return map[string]any{"SAMLMetadataDocument": ""}
	case "GetSSHPublicKey":
		return map[string]any{"SSHPublicKey": map[string]any{}}
	case "GetServerCertificate":
		return map[string]any{"ServerCertificate": map[string]any{}}
	case "GetServiceLastAccessedDetails", "GetServiceLastAccessedDetailsWithEntities":
		return map[string]any{"JobStatus": "COMPLETED"}
	case "SimulateCustomPolicy", "SimulatePrincipalPolicy":
		return map[string]any{"IsTruncated": false, "EvaluationResults": []any{}}
	}

	switch {
	case strings.HasPrefix(action, "Create"):
		return map[string]any{}
	case strings.HasPrefix(action, "Delete"),
		strings.HasPrefix(action, "Update"),
		strings.HasPrefix(action, "Put"),
		strings.HasPrefix(action, "Attach"),
		strings.HasPrefix(action, "Detach"),
		strings.HasPrefix(action, "Add"),
		strings.HasPrefix(action, "Remove"),
		strings.HasPrefix(action, "Enable"),
		strings.HasPrefix(action, "Disable"),
		strings.HasPrefix(action, "Tag"),
		strings.HasPrefix(action, "Untag"),
		strings.HasPrefix(action, "Set"),
		strings.HasPrefix(action, "Reset"),
		strings.HasPrefix(action, "Resync"),
		strings.HasPrefix(action, "Upload"),
		strings.HasPrefix(action, "Accept"),
		strings.HasPrefix(action, "Reject"),
		strings.HasPrefix(action, "Associate"),
		strings.HasPrefix(action, "Send"),
		strings.HasPrefix(action, "Deactivate"),
		strings.HasPrefix(action, "Change"):
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func iamMapValues(src map[string]map[string]any) []any {
	keys := mapKeysSorted(src)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, src[key])
	}
	return out
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
