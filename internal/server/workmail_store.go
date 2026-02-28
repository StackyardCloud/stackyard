package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type workmailStore struct {
	mu sync.Mutex

	nextID int64

	organizations         map[string]*workmailOrganization
	users                 map[string]*workmailUser
	groups                map[string]*workmailGroup
	resources             map[string]*workmailResource
	membersByGroup        map[string]map[string]struct{}
	delegatesByResource   map[string]map[string]struct{}
	mailDomainsByOrg      map[string]map[string]*workmailDomain
	aliasesByEntity       map[string]map[string]struct{}
	mailboxPermsByEntity  map[string]map[string][]string
	accessControlRules    map[string]*workmailAccessControlRule
	mobileAccessRules     map[string]*workmailMobileAccessRule
	mobileAccessOverrides map[string]*workmailMobileDeviceAccessOverride
	availabilityConfigs   map[string]*workmailAvailabilityConfiguration
	impersonationRoles    map[string]*workmailImpersonationRole
	mailboxExportJobs     map[string]*workmailMailboxExportJob
	identityProviderByOrg map[string]map[string]any
	emailMonitoringByOrg  map[string]bool
	dmarcByOrg            map[string]bool
	identityCenterByOrg   map[string]string
	retentionByOrgDays    map[string]int
	personalTokens        map[string]*workmailPersonalAccessToken
	tags                  map[string]map[string]string
}

type workmailOrganization struct {
	ID                string
	Alias             string
	State             string
	DefaultMailDomain string
	CreatedAt         string
	Registered        bool
}

type workmailUser struct {
	ID             string
	OrganizationID string
	Name           string
	DisplayName    string
	Email          string
	State          string
	MailboxQuota   int
	Enabled        bool
	PasswordReset  bool
}

type workmailGroup struct {
	ID             string
	OrganizationID string
	Name           string
	Email          string
	State          string
}

type workmailResource struct {
	ID             string
	OrganizationID string
	Name           string
	Type           string
	Email          string
	State          string
}

type workmailDomain struct {
	DomainName string
	HostedZone string
	DkimStatus string
	IsDefault  bool
}

type workmailAccessControlRule struct {
	Name        string
	Effect      string
	Description string
}

type workmailMobileAccessRule struct {
	ID            string
	Name          string
	Description   string
	Effect        string
	DeviceType    string
	NotDeviceType string
}

type workmailMobileDeviceAccessOverride struct {
	ID             string
	UserID         string
	DeviceID       string
	Effect         string
	Description    string
	DateCreatedUTC string
}

type workmailAvailabilityConfiguration struct {
	DomainName string
	Provider   string
	Enabled    bool
}

type workmailImpersonationRole struct {
	ID          string
	Name        string
	Type        string
	Description string
	DateCreated string
}

type workmailMailboxExportJob struct {
	ID             string
	OrganizationID string
	EntityID       string
	Status         string
	Description    string
	CreatedAt      string
	EndedAt        string
}

type workmailPersonalAccessToken struct {
	ID          string
	UserID      string
	Name        string
	ExpiresTime string
}

func newWorkMailStore() *workmailStore {
	now := time.Now().UTC().Format(time.RFC3339)

	org := &workmailOrganization{
		ID:                "m-000001",
		Alias:             "stackyard",
		State:             "Active",
		DefaultMailDomain: "stackyard.example.com",
		CreatedAt:         now,
		Registered:        true,
	}
	user := &workmailUser{
		ID:             "u-000001",
		OrganizationID: org.ID,
		Name:           "stackyard-user",
		DisplayName:    "Stackyard User",
		Email:          "stackyard-user@stackyard.example.com",
		State:          "ENABLED",
		MailboxQuota:   5120,
		Enabled:        true,
	}
	group := &workmailGroup{
		ID:             "g-000001",
		OrganizationID: org.ID,
		Name:           "stackyard-group",
		Email:          "stackyard-group@stackyard.example.com",
		State:          "ENABLED",
	}
	resource := &workmailResource{
		ID:             "r-000001",
		OrganizationID: org.ID,
		Name:           "stackyard-room",
		Type:           "ROOM",
		Email:          "stackyard-room@stackyard.example.com",
		State:          "ENABLED",
	}
	pat := &workmailPersonalAccessToken{ID: "pat-000001", UserID: user.ID, Name: "stackyard-token", ExpiresTime: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)}

	return &workmailStore{
		nextID: 2,
		organizations: map[string]*workmailOrganization{
			org.ID: org,
		},
		users: map[string]*workmailUser{user.ID: user},
		groups: map[string]*workmailGroup{
			group.ID: group,
		},
		resources: map[string]*workmailResource{resource.ID: resource},
		membersByGroup: map[string]map[string]struct{}{
			group.ID: {user.ID: {}},
		},
		delegatesByResource: map[string]map[string]struct{}{
			resource.ID: {},
		},
		mailDomainsByOrg: map[string]map[string]*workmailDomain{
			org.ID: {
				"stackyard.example.com": {
					DomainName: "stackyard.example.com",
					HostedZone: "ZSTACKYARD0001",
					DkimStatus: "VERIFIED",
					IsDefault:  true,
				},
			},
		},
		aliasesByEntity: map[string]map[string]struct{}{},
		mailboxPermsByEntity: map[string]map[string][]string{
			user.ID: {},
		},
		accessControlRules: map[string]*workmailAccessControlRule{},
		mobileAccessRules: map[string]*workmailMobileAccessRule{
			"mdar-000001": {ID: "mdar-000001", Name: "stackyard-mobile-rule", Description: "seed", Effect: "ALLOW", DeviceType: "IOS"},
		},
		mobileAccessOverrides: map[string]*workmailMobileDeviceAccessOverride{},
		availabilityConfigs: map[string]*workmailAvailabilityConfiguration{
			"stackyard.example.com": {DomainName: "stackyard.example.com", Provider: "EWS", Enabled: true},
		},
		impersonationRoles: map[string]*workmailImpersonationRole{},
		mailboxExportJobs:  map[string]*workmailMailboxExportJob{},
		identityProviderByOrg: map[string]map[string]any{
			org.ID: {},
		},
		emailMonitoringByOrg: map[string]bool{org.ID: false},
		dmarcByOrg:           map[string]bool{org.ID: false},
		identityCenterByOrg:  map[string]string{},
		retentionByOrgDays:   map[string]int{org.ID: 30},
		personalTokens: map[string]*workmailPersonalAccessToken{
			pat.ID: pat,
		},
		tags: map[string]map[string]string{
			workmailOrgARN(org.ID):      {"stackyard": "true"},
			workmailEntityARN(user.ID):  {"stackyard": "true"},
			workmailEntityARN(group.ID): {"stackyard": "true"},
		},
	}
}

func (s *workmailStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	orgID := s.payloadOrganizationID(payload)
	entityID := workmailPayloadString(payload, "EntityId", s.firstUserID())
	groupID := workmailPayloadString(payload, "GroupId", s.firstGroupID())
	userID := workmailPayloadString(payload, "UserId", s.firstUserID())
	resourceID := workmailPayloadString(payload, "ResourceId", s.firstResourceID())
	roleID := workmailPayloadString(payload, "ImpersonationRoleId", "")
	ruleID := workmailPayloadString(payload, "MobileDeviceAccessRuleId", "")
	overrideID := workmailPayloadString(payload, "MobileDeviceAccessOverrideId", "")
	domainName := workmailPayloadString(payload, "DomainName", "stackyard.example.com")
	name := workmailPayloadString(payload, "Name", "stackyard-item")

	s.ensureOrganization(orgID, now)

	switch action {
	case "CreateOrganization":
		alias := workmailPayloadString(payload, "Alias", s.nextToken("org"))
		id := workmailPayloadString(payload, "OrganizationId", fmt.Sprintf("m-%06d", s.nextID))
		if !strings.HasPrefix(id, "m-") {
			id = fmt.Sprintf("m-%06d", s.nextID)
		}
		s.nextID++
		org := &workmailOrganization{ID: id, Alias: alias, State: "Active", DefaultMailDomain: alias + ".example.com", CreatedAt: now, Registered: true}
		s.organizations[id] = org
		s.ensureDomains(org.ID)
		s.mailDomainsByOrg[org.ID][org.DefaultMailDomain] = &workmailDomain{DomainName: org.DefaultMailDomain, HostedZone: "Z" + strings.ToUpper(strings.ReplaceAll(alias, "-", "")) + "0001", DkimStatus: "VERIFIED", IsDefault: true}
		return map[string]any{"OrganizationId": org.ID, "State": org.State}
	case "DeleteOrganization":
		delete(s.organizations, orgID)
		delete(s.mailDomainsByOrg, orgID)
		return map[string]any{}
	case "DescribeOrganization":
		org := s.ensureOrganization(orgID, now)
		return map[string]any{"OrganizationId": org.ID, "Alias": org.Alias, "State": org.State, "DefaultMailDomain": org.DefaultMailDomain, "CompletedDate": org.CreatedAt}
	case "ListOrganizations":
		items := make([]any, 0, len(s.organizations))
		for _, org := range s.sortedOrganizations() {
			items = append(items, map[string]any{"OrganizationId": org.ID, "Alias": org.Alias, "State": org.State, "DefaultMailDomain": org.DefaultMailDomain, "CompletedDate": org.CreatedAt})
		}
		return map[string]any{"OrganizationSummaries": items, "NextToken": ""}
	case "RegisterToWorkMail":
		org := s.ensureOrganization(orgID, now)
		org.Registered = true
		return map[string]any{}
	case "DeregisterFromWorkMail":
		org := s.ensureOrganization(orgID, now)
		org.Registered = false
		return map[string]any{}

	case "RegisterMailDomain":
		org := s.ensureOrganization(orgID, now)
		s.ensureDomains(org.ID)
		domain := &workmailDomain{DomainName: domainName, HostedZone: "Z" + strings.ToUpper(strings.ReplaceAll(domainName, ".", "")), DkimStatus: "PENDING", IsDefault: false}
		s.mailDomainsByOrg[org.ID][domainName] = domain
		return map[string]any{}
	case "DeregisterMailDomain":
		s.ensureDomains(orgID)
		delete(s.mailDomainsByOrg[orgID], domainName)
		return map[string]any{}
	case "UpdateDefaultMailDomain":
		org := s.ensureOrganization(orgID, now)
		s.ensureDomains(org.ID)
		if _, ok := s.mailDomainsByOrg[org.ID][domainName]; !ok {
			s.mailDomainsByOrg[org.ID][domainName] = &workmailDomain{DomainName: domainName, HostedZone: "ZDEFAULT0001", DkimStatus: "VERIFIED"}
		}
		for _, domain := range s.mailDomainsByOrg[org.ID] {
			domain.IsDefault = false
		}
		s.mailDomainsByOrg[org.ID][domainName].IsDefault = true
		org.DefaultMailDomain = domainName
		return map[string]any{}
	case "GetMailDomain":
		s.ensureDomains(orgID)
		domain := s.ensureDomain(orgID, domainName)
		return map[string]any{"IsTestDomain": false, "IsDefault": domain.IsDefault, "IsOwned": true, "DomainName": domain.DomainName, "DkimVerificationStatus": domain.DkimStatus}
	case "ListMailDomains":
		s.ensureDomains(orgID)
		domains := s.sortedDomains(orgID)
		items := make([]any, 0, len(domains))
		for _, domain := range domains {
			items = append(items, map[string]any{"DomainName": domain.DomainName, "DefaultDomain": domain.IsDefault})
		}
		return map[string]any{"MailDomains": items, "NextToken": ""}

	case "CreateUser":
		id := fmt.Sprintf("u-%06d", s.nextID)
		s.nextID++
		user := &workmailUser{ID: id, OrganizationID: orgID, Name: name, DisplayName: workmailPayloadString(payload, "DisplayName", name), Email: workmailPayloadString(payload, "Email", name+"@"+s.ensureOrganization(orgID, now).DefaultMailDomain), State: "ENABLED", MailboxQuota: 5120, Enabled: true}
		s.users[id] = user
		return map[string]any{"UserId": user.ID}
	case "DescribeUser":
		user := s.ensureUser(userID, orgID)
		return map[string]any{"UserId": user.ID, "Name": user.Name, "DisplayName": user.DisplayName, "Email": user.Email, "State": user.State, "EnabledDate": now, "DisabledDate": ""}
	case "UpdateUser":
		user := s.ensureUser(userID, orgID)
		if v := workmailPayloadString(payload, "DisplayName", ""); v != "" {
			user.DisplayName = v
		}
		if v := workmailPayloadString(payload, "Name", ""); v != "" {
			user.Name = v
		}
		if v := workmailPayloadString(payload, "Email", ""); v != "" {
			user.Email = v
		}
		return map[string]any{}
	case "DeleteUser":
		delete(s.users, userID)
		return map[string]any{}
	case "ResetPassword":
		user := s.ensureUser(userID, orgID)
		user.PasswordReset = true
		return map[string]any{}
	case "UpdateMailboxQuota":
		user := s.ensureUser(userID, orgID)
		user.MailboxQuota = int(workmailPayloadInt(payload, "MailboxQuota", int64(user.MailboxQuota)))
		return map[string]any{}
	case "UpdatePrimaryEmailAddress":
		if u, ok := s.users[entityID]; ok {
			u.Email = workmailPayloadString(payload, "Email", u.Email)
		}
		if g, ok := s.groups[entityID]; ok {
			g.Email = workmailPayloadString(payload, "Email", g.Email)
		}
		if r, ok := s.resources[entityID]; ok {
			r.Email = workmailPayloadString(payload, "Email", r.Email)
		}
		return map[string]any{}
	case "GetMailboxDetails":
		user := s.ensureUser(entityID, orgID)
		return map[string]any{"MailboxQuota": user.MailboxQuota, "MailboxSize": 128}

	case "CreateGroup":
		id := fmt.Sprintf("g-%06d", s.nextID)
		s.nextID++
		group := &workmailGroup{ID: id, OrganizationID: orgID, Name: name, Email: workmailPayloadString(payload, "Email", name+"@"+s.ensureOrganization(orgID, now).DefaultMailDomain), State: "ENABLED"}
		s.groups[id] = group
		s.membersByGroup[id] = map[string]struct{}{}
		return map[string]any{"GroupId": group.ID}
	case "DescribeGroup":
		group := s.ensureGroup(groupID, orgID)
		return map[string]any{"GroupId": group.ID, "Name": group.Name, "Email": group.Email, "State": group.State, "EnabledDate": now, "DisabledDate": ""}
	case "UpdateGroup":
		group := s.ensureGroup(groupID, orgID)
		if v := workmailPayloadString(payload, "Name", ""); v != "" {
			group.Name = v
		}
		if v := workmailPayloadString(payload, "Email", ""); v != "" {
			group.Email = v
		}
		return map[string]any{}
	case "DeleteGroup":
		delete(s.groups, groupID)
		delete(s.membersByGroup, groupID)
		return map[string]any{}

	case "CreateResource":
		id := fmt.Sprintf("r-%06d", s.nextID)
		s.nextID++
		resource := &workmailResource{ID: id, OrganizationID: orgID, Name: name, Type: workmailPayloadString(payload, "Type", "ROOM"), Email: workmailPayloadString(payload, "Email", name+"@"+s.ensureOrganization(orgID, now).DefaultMailDomain), State: "ENABLED"}
		s.resources[id] = resource
		s.delegatesByResource[id] = map[string]struct{}{}
		return map[string]any{"ResourceId": resource.ID}
	case "DescribeResource":
		resource := s.ensureResource(resourceID, orgID)
		return map[string]any{"ResourceId": resource.ID, "Name": resource.Name, "Type": resource.Type, "Email": resource.Email, "State": resource.State, "EnabledDate": now, "DisabledDate": ""}
	case "UpdateResource":
		resource := s.ensureResource(resourceID, orgID)
		if v := workmailPayloadString(payload, "Name", ""); v != "" {
			resource.Name = v
		}
		if v := workmailPayloadString(payload, "Email", ""); v != "" {
			resource.Email = v
		}
		if v := workmailPayloadString(payload, "Type", ""); v != "" {
			resource.Type = v
		}
		return map[string]any{}
	case "DeleteResource":
		delete(s.resources, resourceID)
		delete(s.delegatesByResource, resourceID)
		return map[string]any{}

	case "DescribeEntity":
		if u, ok := s.users[entityID]; ok {
			return map[string]any{"EntityId": u.ID, "Name": u.Name, "Type": "USER", "State": u.State, "EnabledDate": now}
		}
		if g, ok := s.groups[entityID]; ok {
			return map[string]any{"EntityId": g.ID, "Name": g.Name, "Type": "GROUP", "State": g.State, "EnabledDate": now}
		}
		if r, ok := s.resources[entityID]; ok {
			return map[string]any{"EntityId": r.ID, "Name": r.Name, "Type": "RESOURCE", "State": r.State, "EnabledDate": now}
		}
		return map[string]any{"EntityId": entityID, "Name": "unknown", "Type": "USER", "State": "ENABLED", "EnabledDate": now}

	case "ListUsers":
		items := make([]any, 0)
		for _, user := range s.sortedUsers(orgID) {
			items = append(items, map[string]any{"Id": user.ID, "Name": user.Name, "Email": user.Email, "State": user.State, "DisplayName": user.DisplayName})
		}
		return map[string]any{"Users": items, "NextToken": ""}
	case "ListGroups":
		items := make([]any, 0)
		for _, group := range s.sortedGroups(orgID) {
			items = append(items, map[string]any{"Id": group.ID, "Name": group.Name, "Email": group.Email, "State": group.State})
		}
		return map[string]any{"Groups": items, "NextToken": ""}
	case "ListResources":
		items := make([]any, 0)
		for _, resource := range s.sortedResources(orgID) {
			items = append(items, map[string]any{"Id": resource.ID, "Name": resource.Name, "Email": resource.Email, "Type": resource.Type, "State": resource.State})
		}
		return map[string]any{"Resources": items, "NextToken": ""}
	case "AssociateMemberToGroup":
		group := s.ensureGroup(groupID, orgID)
		s.ensureMemberSet(group.ID)[entityID] = struct{}{}
		return map[string]any{}
	case "DisassociateMemberFromGroup":
		delete(s.ensureMemberSet(groupID), entityID)
		return map[string]any{}
	case "ListGroupMembers":
		group := s.ensureGroup(groupID, orgID)
		members := s.ensureMemberSet(group.ID)
		items := make([]any, 0, len(members))
		for memberID := range members {
			items = append(items, s.memberPayload(memberID))
		}
		sort.Slice(items, func(i, j int) bool {
			li := workmailPayloadString(items[i].(map[string]any), "Id", "")
			lj := workmailPayloadString(items[j].(map[string]any), "Id", "")
			return li < lj
		})
		return map[string]any{"Members": items, "NextToken": ""}
	case "ListGroupsForEntity":
		items := make([]any, 0)
		for _, group := range s.sortedGroups(orgID) {
			if _, ok := s.ensureMemberSet(group.ID)[entityID]; ok {
				items = append(items, map[string]any{"Id": group.ID, "Name": group.Name})
			}
		}
		return map[string]any{"Groups": items, "NextToken": ""}

	case "AssociateDelegateToResource":
		resource := s.ensureResource(resourceID, orgID)
		s.ensureDelegateSet(resource.ID)[entityID] = struct{}{}
		return map[string]any{}
	case "DisassociateDelegateFromResource":
		delete(s.ensureDelegateSet(resourceID), entityID)
		return map[string]any{}
	case "ListResourceDelegates":
		set := s.ensureDelegateSet(resourceID)
		items := make([]any, 0, len(set))
		for delegateID := range set {
			items = append(items, map[string]any{"Id": delegateID, "Type": s.entityType(delegateID)})
		}
		return map[string]any{"Delegates": items, "NextToken": ""}

	case "CreateAlias":
		alias := workmailPayloadString(payload, "Alias", workmailPayloadString(payload, "Entity", "alias@example.com"))
		s.ensureAliasSet(entityID)[alias] = struct{}{}
		return map[string]any{}
	case "DeleteAlias":
		alias := workmailPayloadString(payload, "Alias", "")
		delete(s.ensureAliasSet(entityID), alias)
		return map[string]any{}
	case "ListAliases":
		set := s.ensureAliasSet(entityID)
		aliases := make([]string, 0, len(set))
		for alias := range set {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)
		items := make([]any, 0, len(aliases))
		for _, alias := range aliases {
			items = append(items, alias)
		}
		return map[string]any{"Aliases": items}

	case "PutMailboxPermissions":
		grantee := workmailPayloadString(payload, "GranteeId", userID)
		permissions := workmailPayloadStringSlice(payload, "PermissionValues")
		if len(permissions) == 0 {
			permissions = []string{"FULL_ACCESS"}
		}
		s.ensureMailboxPermSet(entityID)[grantee] = permissions
		return map[string]any{}
	case "ListMailboxPermissions":
		items := make([]any, 0)
		for grantee, perms := range s.ensureMailboxPermSet(entityID) {
			items = append(items, map[string]any{"GranteeId": grantee, "GranteeType": s.entityType(grantee), "PermissionValues": perms})
		}
		return map[string]any{"Permissions": items, "NextToken": ""}
	case "DeleteMailboxPermissions":
		grantee := workmailPayloadString(payload, "GranteeId", userID)
		delete(s.ensureMailboxPermSet(entityID), grantee)
		return map[string]any{}

	case "PutAccessControlRule":
		ruleName := workmailPayloadString(payload, "Name", fmt.Sprintf("acr-%06d", s.nextID))
		rule := &workmailAccessControlRule{Name: ruleName, Effect: workmailPayloadString(payload, "Effect", "ALLOW"), Description: workmailPayloadString(payload, "Description", "")}
		s.accessControlRules[rule.Name] = rule
		s.nextID++
		return map[string]any{}
	case "ListAccessControlRules":
		items := make([]any, 0, len(s.accessControlRules))
		for _, rule := range s.sortedAccessRules() {
			items = append(items, map[string]any{"Name": rule.Name, "Effect": rule.Effect, "Description": rule.Description})
		}
		return map[string]any{"Rules": items}
	case "GetAccessControlEffect":
		matched := make([]any, 0)
		for _, rule := range s.sortedAccessRules() {
			matched = append(matched, map[string]any{"Name": rule.Name, "Effect": rule.Effect})
		}
		return map[string]any{"Effect": "ALLOW", "MatchedRules": matched}
	case "DeleteAccessControlRule":
		ruleName := workmailPayloadString(payload, "Name", name)
		delete(s.accessControlRules, ruleName)
		return map[string]any{}

	case "PutRetentionPolicy":
		s.retentionByOrgDays[orgID] = int(workmailPayloadInt(payload, "FolderConfigurations.0.Period", 30))
		return map[string]any{}
	case "GetDefaultRetentionPolicy":
		return map[string]any{"Id": "drp-000001", "Name": "DefaultRetentionPolicy", "Description": "Stackyard default retention", "FolderConfigurations": []any{map[string]any{"Name": "INBOX", "Action": "DELETE", "Period": s.retentionByOrgDays[orgID]}}}
	case "DeleteRetentionPolicy":
		delete(s.retentionByOrgDays, orgID)
		return map[string]any{}

	case "CreateMobileDeviceAccessRule":
		id := fmt.Sprintf("mdar-%06d", s.nextID)
		s.nextID++
		rule := &workmailMobileAccessRule{ID: id, Name: name, Description: workmailPayloadString(payload, "Description", ""), Effect: workmailPayloadString(payload, "Effect", "ALLOW"), DeviceType: workmailPayloadString(payload, "DeviceType", "IOS")}
		s.mobileAccessRules[id] = rule
		return map[string]any{"MobileDeviceAccessRuleId": id}
	case "UpdateMobileDeviceAccessRule":
		rule := s.ensureMobileRule(ruleID)
		if v := workmailPayloadString(payload, "Name", ""); v != "" {
			rule.Name = v
		}
		if v := workmailPayloadString(payload, "Description", ""); v != "" {
			rule.Description = v
		}
		if v := workmailPayloadString(payload, "Effect", ""); v != "" {
			rule.Effect = v
		}
		return map[string]any{}
	case "DeleteMobileDeviceAccessRule":
		delete(s.mobileAccessRules, ruleID)
		return map[string]any{}
	case "ListMobileDeviceAccessRules":
		items := make([]any, 0, len(s.mobileAccessRules))
		for _, rule := range s.sortedMobileRules() {
			items = append(items, map[string]any{"MobileDeviceAccessRuleId": rule.ID, "Name": rule.Name, "Description": rule.Description, "Effect": rule.Effect})
		}
		return map[string]any{"Rules": items}
	case "PutMobileDeviceAccessOverride":
		id := fmt.Sprintf("mdao-%06d", s.nextID)
		s.nextID++
		o := &workmailMobileDeviceAccessOverride{ID: id, UserID: userID, DeviceID: workmailPayloadString(payload, "DeviceId", "device-000001"), Effect: workmailPayloadString(payload, "Effect", "ALLOW"), Description: workmailPayloadString(payload, "Description", ""), DateCreatedUTC: now}
		s.mobileAccessOverrides[id] = o
		return map[string]any{}
	case "GetMobileDeviceAccessOverride":
		o := s.ensureMobileOverride(overrideID, userID, now)
		return map[string]any{"MobileDeviceAccessOverride": map[string]any{"MobileDeviceAccessOverrideId": o.ID, "UserId": o.UserID, "DeviceId": o.DeviceID, "Effect": o.Effect, "Description": o.Description, "DateCreated": o.DateCreatedUTC}}
	case "DeleteMobileDeviceAccessOverride":
		delete(s.mobileAccessOverrides, overrideID)
		return map[string]any{}
	case "ListMobileDeviceAccessOverrides":
		items := make([]any, 0, len(s.mobileAccessOverrides))
		for _, o := range s.sortedMobileOverrides() {
			items = append(items, map[string]any{"MobileDeviceAccessOverrideId": o.ID, "UserId": o.UserID, "DeviceId": o.DeviceID, "Effect": o.Effect, "DateCreated": o.DateCreatedUTC})
		}
		return map[string]any{"Overrides": items}
	case "GetMobileDeviceAccessEffect":
		matched := make([]any, 0)
		for _, rule := range s.sortedMobileRules() {
			matched = append(matched, map[string]any{"MobileDeviceAccessRuleId": rule.ID, "Name": rule.Name, "Effect": rule.Effect})
		}
		return map[string]any{"Effect": "ALLOW", "MatchedRules": matched}

	case "CreateAvailabilityConfiguration":
		s.availabilityConfigs[domainName] = &workmailAvailabilityConfiguration{DomainName: domainName, Provider: workmailPayloadString(payload, "EwsProvider.Endpoint", "EWS"), Enabled: true}
		return map[string]any{}
	case "UpdateAvailabilityConfiguration":
		cfg := s.ensureAvailability(domainName)
		cfg.Enabled = workmailPayloadBool(payload, "Enabled", true)
		return map[string]any{}
	case "DeleteAvailabilityConfiguration":
		delete(s.availabilityConfigs, domainName)
		return map[string]any{}
	case "DescribeAvailabilityConfiguration":
		cfg := s.ensureAvailability(domainName)
		return map[string]any{"AvailabilityConfiguration": map[string]any{"DomainName": cfg.DomainName, "EwsProvider": map[string]any{"Endpoint": "https://ews.example.com"}, "LambdaProvider": map[string]any{"LambdaArn": "arn:aws:lambda:us-east-1:123456789012:function:stackyard-workmail"}, "DateCreated": now, "DateModified": now}}
	case "ListAvailabilityConfigurations":
		items := make([]any, 0, len(s.availabilityConfigs))
		for _, cfg := range s.sortedAvailability() {
			items = append(items, map[string]any{"DomainName": cfg.DomainName, "DateCreated": now, "DateModified": now})
		}
		return map[string]any{"AvailabilityConfigurations": items, "NextToken": ""}
	case "TestAvailabilityConfiguration":
		cfg := s.ensureAvailability(domainName)
		return map[string]any{"TestPassed": cfg.Enabled, "FailureReason": ""}

	case "CreateImpersonationRole":
		id := fmt.Sprintf("imp-%06d", s.nextID)
		s.nextID++
		role := &workmailImpersonationRole{ID: id, Name: name, Type: workmailPayloadString(payload, "Type", "FULL_ACCESS"), Description: workmailPayloadString(payload, "Description", ""), DateCreated: now}
		s.impersonationRoles[id] = role
		return map[string]any{"ImpersonationRoleId": id}
	case "UpdateImpersonationRole":
		role := s.ensureImpersonationRole(roleID, now)
		if v := workmailPayloadString(payload, "Name", ""); v != "" {
			role.Name = v
		}
		if v := workmailPayloadString(payload, "Type", ""); v != "" {
			role.Type = v
		}
		if v := workmailPayloadString(payload, "Description", ""); v != "" {
			role.Description = v
		}
		return map[string]any{}
	case "DeleteImpersonationRole":
		delete(s.impersonationRoles, roleID)
		return map[string]any{}
	case "GetImpersonationRole":
		role := s.ensureImpersonationRole(roleID, now)
		return map[string]any{"ImpersonationRole": map[string]any{"ImpersonationRoleId": role.ID, "Name": role.Name, "Type": role.Type, "Description": role.Description, "DateCreated": role.DateCreated}}
	case "ListImpersonationRoles":
		items := make([]any, 0, len(s.impersonationRoles))
		for _, role := range s.sortedImpersonationRoles() {
			items = append(items, map[string]any{"ImpersonationRoleId": role.ID, "Name": role.Name, "Type": role.Type, "DateCreated": role.DateCreated})
		}
		return map[string]any{"Roles": items, "NextToken": ""}
	case "GetImpersonationRoleEffect":
		matched := make([]any, 0)
		for _, role := range s.sortedImpersonationRoles() {
			matched = append(matched, map[string]any{"ImpersonationRoleId": role.ID, "Name": role.Name, "Effect": "ALLOW"})
		}
		return map[string]any{"Effect": "ALLOW", "MatchedRules": matched}
	case "AssumeImpersonationRole":
		role := s.ensureImpersonationRole(roleID, now)
		return map[string]any{"Token": "wm-imp-token-" + role.ID, "ExpiresIn": 3600}

	case "PutIdentityProviderConfiguration":
		s.identityProviderByOrg[orgID] = map[string]any{"AuthenticationMode": workmailPayloadString(payload, "AuthenticationMode", "IDENTITY_PROVIDER_ONLY")}
		return map[string]any{}
	case "DescribeIdentityProviderConfiguration":
		cfg := s.identityProviderByOrg[orgID]
		if cfg == nil {
			cfg = map[string]any{"AuthenticationMode": "IDENTITY_PROVIDER_ONLY"}
		}
		return cfg
	case "DeleteIdentityProviderConfiguration":
		delete(s.identityProviderByOrg, orgID)
		return map[string]any{}
	case "CreateIdentityCenterApplication":
		arn := fmt.Sprintf("arn:aws:sso::123456789012:application/wm-%06d", s.nextID)
		s.nextID++
		s.identityCenterByOrg[orgID] = arn
		return map[string]any{"IdentityCenterApplicationArn": arn}
	case "DeleteIdentityCenterApplication":
		delete(s.identityCenterByOrg, orgID)
		return map[string]any{}

	case "PutEmailMonitoringConfiguration":
		s.emailMonitoringByOrg[orgID] = true
		return map[string]any{}
	case "DescribeEmailMonitoringConfiguration":
		return map[string]any{"RoleArn": "arn:aws:iam::123456789012:role/stackyard-workmail-monitor", "LogGroupArn": "arn:aws:logs:us-east-1:123456789012:log-group:/stackyard/workmail", "EmailMonitoringState": map[string]any{"Journaling": s.emailMonitoringByOrg[orgID]}}
	case "DeleteEmailMonitoringConfiguration":
		s.emailMonitoringByOrg[orgID] = false
		return map[string]any{}
	case "PutInboundDmarcSettings":
		s.dmarcByOrg[orgID] = workmailPayloadBool(payload, "Enforced", true)
		return map[string]any{}
	case "DescribeInboundDmarcSettings":
		return map[string]any{"Enforced": s.dmarcByOrg[orgID]}

	case "StartMailboxExportJob":
		id := fmt.Sprintf("mbej-%06d", s.nextID)
		s.nextID++
		job := &workmailMailboxExportJob{ID: id, OrganizationID: orgID, EntityID: entityID, Status: "RUNNING", Description: "Stackyard export", CreatedAt: now}
		s.mailboxExportJobs[id] = job
		return map[string]any{"JobId": id}
	case "DescribeMailboxExportJob":
		jobID := workmailPayloadString(payload, "JobId", s.firstMailboxExportJobID())
		job := s.ensureMailboxExportJob(jobID, orgID, entityID, now)
		if job.Status == "RUNNING" {
			job.Status = "COMPLETED"
			job.EndedAt = now
		}
		return map[string]any{"EntityId": job.EntityID, "Description": job.Description, "S3BucketName": "stackyard-workmail-exports", "S3Path": "exports/", "EstimatedProgress": 100, "State": job.Status, "ErrorInfo": "", "StartTime": job.CreatedAt, "EndTime": job.EndedAt}
	case "ListMailboxExportJobs":
		items := make([]any, 0, len(s.mailboxExportJobs))
		for _, job := range s.sortedExportJobs() {
			items = append(items, map[string]any{"JobId": job.ID, "EntityId": job.EntityID, "State": job.Status, "StartTime": job.CreatedAt, "EndTime": job.EndedAt})
		}
		return map[string]any{"Jobs": items, "NextToken": ""}
	case "CancelMailboxExportJob":
		jobID := workmailPayloadString(payload, "JobId", s.firstMailboxExportJobID())
		job := s.ensureMailboxExportJob(jobID, orgID, entityID, now)
		job.Status = "CANCELLED"
		job.EndedAt = now
		return map[string]any{}

	case "TagResource":
		resourceARN := workmailPayloadString(payload, "ResourceARN", workmailPayloadString(payload, "ResourceArn", workmailOrgARN(orgID)))
		tagMap := s.ensureTags(resourceARN)
		for key, value := range workmailPayloadTags(payload) {
			tagMap[key] = value
		}
		return map[string]any{}
	case "UntagResource":
		resourceARN := workmailPayloadString(payload, "ResourceARN", workmailPayloadString(payload, "ResourceArn", workmailOrgARN(orgID)))
		tagMap := s.ensureTags(resourceARN)
		for _, key := range workmailPayloadStringSlice(payload, "TagKeys") {
			delete(tagMap, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		resourceARN := workmailPayloadString(payload, "ResourceARN", workmailPayloadString(payload, "ResourceArn", workmailOrgARN(orgID)))
		tagMap := s.ensureTags(resourceARN)
		out := map[string]any{}
		for key, value := range tagMap {
			out[key] = value
		}
		return map[string]any{"Tags": out}

	case "ListPersonalAccessTokens":
		items := make([]any, 0, len(s.personalTokens))
		for _, tok := range s.sortedPATs() {
			items = append(items, map[string]any{"PersonalAccessTokenId": tok.ID, "Name": tok.Name, "DateCreated": now, "DateLastUsed": now, "ExpiresTime": tok.ExpiresTime})
		}
		return map[string]any{"PersonalAccessTokenSummaries": items, "NextToken": ""}
	case "GetPersonalAccessTokenMetadata":
		tokenID := workmailPayloadString(payload, "PersonalAccessTokenId", "pat-000001")
		tok := s.ensurePAT(tokenID, userID)
		return map[string]any{"PersonalAccessTokenId": tok.ID, "Name": tok.Name, "UserId": tok.UserID, "DateCreated": now, "DateLastUsed": now, "ExpiresTime": tok.ExpiresTime}
	case "DeletePersonalAccessToken":
		tokenID := workmailPayloadString(payload, "PersonalAccessTokenId", "pat-000001")
		delete(s.personalTokens, tokenID)
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *workmailStore) payloadOrganizationID(payload map[string]any) string {
	orgID := workmailPayloadString(payload, "OrganizationId", "")
	if orgID != "" {
		return orgID
	}
	for id := range s.organizations {
		return id
	}
	return "m-000001"
}

func (s *workmailStore) ensureOrganization(id, now string) *workmailOrganization {
	if org, ok := s.organizations[id]; ok {
		return org
	}
	org := &workmailOrganization{ID: id, Alias: "stackyard", State: "Active", DefaultMailDomain: "stackyard.example.com", CreatedAt: now, Registered: true}
	s.organizations[id] = org
	return org
}

func (s *workmailStore) ensureDomains(orgID string) {
	if _, ok := s.mailDomainsByOrg[orgID]; !ok {
		s.mailDomainsByOrg[orgID] = map[string]*workmailDomain{}
	}
}

func (s *workmailStore) ensureDomain(orgID, domainName string) *workmailDomain {
	s.ensureDomains(orgID)
	if domain, ok := s.mailDomainsByOrg[orgID][domainName]; ok {
		return domain
	}
	domain := &workmailDomain{DomainName: domainName, HostedZone: "ZDEFAULT0001", DkimStatus: "VERIFIED", IsDefault: false}
	s.mailDomainsByOrg[orgID][domainName] = domain
	return domain
}

func (s *workmailStore) sortedDomains(orgID string) []*workmailDomain {
	s.ensureDomains(orgID)
	items := make([]*workmailDomain, 0, len(s.mailDomainsByOrg[orgID]))
	for _, domain := range s.mailDomainsByOrg[orgID] {
		items = append(items, domain)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].DomainName < items[j].DomainName })
	return items
}

func (s *workmailStore) ensureUser(id, orgID string) *workmailUser {
	if user, ok := s.users[id]; ok {
		return user
	}
	org := s.ensureOrganization(orgID, time.Now().UTC().Format(time.RFC3339))
	user := &workmailUser{ID: id, OrganizationID: org.ID, Name: "stackyard-user", DisplayName: "Stackyard User", Email: "stackyard-user@" + org.DefaultMailDomain, State: "ENABLED", MailboxQuota: 5120, Enabled: true}
	s.users[id] = user
	return user
}

func (s *workmailStore) ensureGroup(id, orgID string) *workmailGroup {
	if group, ok := s.groups[id]; ok {
		return group
	}
	org := s.ensureOrganization(orgID, time.Now().UTC().Format(time.RFC3339))
	group := &workmailGroup{ID: id, OrganizationID: org.ID, Name: "stackyard-group", Email: "stackyard-group@" + org.DefaultMailDomain, State: "ENABLED"}
	s.groups[id] = group
	s.membersByGroup[id] = map[string]struct{}{}
	return group
}

func (s *workmailStore) ensureResource(id, orgID string) *workmailResource {
	if resource, ok := s.resources[id]; ok {
		return resource
	}
	org := s.ensureOrganization(orgID, time.Now().UTC().Format(time.RFC3339))
	resource := &workmailResource{ID: id, OrganizationID: org.ID, Name: "stackyard-room", Type: "ROOM", Email: "stackyard-room@" + org.DefaultMailDomain, State: "ENABLED"}
	s.resources[id] = resource
	s.delegatesByResource[id] = map[string]struct{}{}
	return resource
}

func (s *workmailStore) ensureMemberSet(groupID string) map[string]struct{} {
	if set, ok := s.membersByGroup[groupID]; ok {
		return set
	}
	set := map[string]struct{}{}
	s.membersByGroup[groupID] = set
	return set
}

func (s *workmailStore) ensureDelegateSet(resourceID string) map[string]struct{} {
	if set, ok := s.delegatesByResource[resourceID]; ok {
		return set
	}
	set := map[string]struct{}{}
	s.delegatesByResource[resourceID] = set
	return set
}

func (s *workmailStore) ensureAliasSet(entityID string) map[string]struct{} {
	if set, ok := s.aliasesByEntity[entityID]; ok {
		return set
	}
	set := map[string]struct{}{}
	s.aliasesByEntity[entityID] = set
	return set
}

func (s *workmailStore) ensureMailboxPermSet(entityID string) map[string][]string {
	if set, ok := s.mailboxPermsByEntity[entityID]; ok {
		return set
	}
	set := map[string][]string{}
	s.mailboxPermsByEntity[entityID] = set
	return set
}

func (s *workmailStore) ensureMobileRule(ruleID string) *workmailMobileAccessRule {
	if rule, ok := s.mobileAccessRules[ruleID]; ok {
		return rule
	}
	rule := &workmailMobileAccessRule{ID: ruleID, Name: "stackyard-mobile-rule", Effect: "ALLOW", DeviceType: "IOS"}
	s.mobileAccessRules[ruleID] = rule
	return rule
}

func (s *workmailStore) ensureMobileOverride(overrideID, userID, now string) *workmailMobileDeviceAccessOverride {
	if o, ok := s.mobileAccessOverrides[overrideID]; ok {
		return o
	}
	o := &workmailMobileDeviceAccessOverride{ID: overrideID, UserID: userID, DeviceID: "device-000001", Effect: "ALLOW", Description: "", DateCreatedUTC: now}
	s.mobileAccessOverrides[overrideID] = o
	return o
}

func (s *workmailStore) ensureAvailability(domainName string) *workmailAvailabilityConfiguration {
	if cfg, ok := s.availabilityConfigs[domainName]; ok {
		return cfg
	}
	cfg := &workmailAvailabilityConfiguration{DomainName: domainName, Provider: "EWS", Enabled: true}
	s.availabilityConfigs[domainName] = cfg
	return cfg
}

func (s *workmailStore) ensureImpersonationRole(roleID, now string) *workmailImpersonationRole {
	if role, ok := s.impersonationRoles[roleID]; ok {
		return role
	}
	role := &workmailImpersonationRole{ID: roleID, Name: "stackyard-role", Type: "FULL_ACCESS", Description: "", DateCreated: now}
	s.impersonationRoles[roleID] = role
	return role
}

func (s *workmailStore) ensureMailboxExportJob(jobID, orgID, entityID, now string) *workmailMailboxExportJob {
	if job, ok := s.mailboxExportJobs[jobID]; ok {
		return job
	}
	job := &workmailMailboxExportJob{ID: jobID, OrganizationID: orgID, EntityID: entityID, Status: "RUNNING", Description: "Stackyard export", CreatedAt: now}
	s.mailboxExportJobs[jobID] = job
	return job
}

func (s *workmailStore) ensurePAT(tokenID, userID string) *workmailPersonalAccessToken {
	if tok, ok := s.personalTokens[tokenID]; ok {
		return tok
	}
	tok := &workmailPersonalAccessToken{ID: tokenID, UserID: userID, Name: "stackyard-token", ExpiresTime: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)}
	s.personalTokens[tokenID] = tok
	return tok
}

func (s *workmailStore) sortedOrganizations() []*workmailOrganization {
	items := make([]*workmailOrganization, 0, len(s.organizations))
	for _, org := range s.organizations {
		items = append(items, org)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workmailStore) sortedUsers(orgID string) []*workmailUser {
	items := make([]*workmailUser, 0)
	for _, user := range s.users {
		if user.OrganizationID == orgID {
			items = append(items, user)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workmailStore) sortedGroups(orgID string) []*workmailGroup {
	items := make([]*workmailGroup, 0)
	for _, group := range s.groups {
		if group.OrganizationID == orgID {
			items = append(items, group)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workmailStore) sortedResources(orgID string) []*workmailResource {
	items := make([]*workmailResource, 0)
	for _, resource := range s.resources {
		if resource.OrganizationID == orgID {
			items = append(items, resource)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workmailStore) sortedAccessRules() []*workmailAccessControlRule {
	items := make([]*workmailAccessControlRule, 0, len(s.accessControlRules))
	for _, rule := range s.accessControlRules {
		items = append(items, rule)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (s *workmailStore) sortedMobileRules() []*workmailMobileAccessRule {
	items := make([]*workmailMobileAccessRule, 0, len(s.mobileAccessRules))
	for _, rule := range s.mobileAccessRules {
		items = append(items, rule)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workmailStore) sortedMobileOverrides() []*workmailMobileDeviceAccessOverride {
	items := make([]*workmailMobileDeviceAccessOverride, 0, len(s.mobileAccessOverrides))
	for _, o := range s.mobileAccessOverrides {
		items = append(items, o)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workmailStore) sortedAvailability() []*workmailAvailabilityConfiguration {
	items := make([]*workmailAvailabilityConfiguration, 0, len(s.availabilityConfigs))
	for _, cfg := range s.availabilityConfigs {
		items = append(items, cfg)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].DomainName < items[j].DomainName })
	return items
}

func (s *workmailStore) sortedImpersonationRoles() []*workmailImpersonationRole {
	items := make([]*workmailImpersonationRole, 0, len(s.impersonationRoles))
	for _, role := range s.impersonationRoles {
		items = append(items, role)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workmailStore) sortedExportJobs() []*workmailMailboxExportJob {
	items := make([]*workmailMailboxExportJob, 0, len(s.mailboxExportJobs))
	for _, job := range s.mailboxExportJobs {
		items = append(items, job)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workmailStore) sortedPATs() []*workmailPersonalAccessToken {
	items := make([]*workmailPersonalAccessToken, 0, len(s.personalTokens))
	for _, tok := range s.personalTokens {
		items = append(items, tok)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *workmailStore) firstUserID() string {
	for id := range s.users {
		return id
	}
	return "u-000001"
}

func (s *workmailStore) firstGroupID() string {
	for id := range s.groups {
		return id
	}
	return "g-000001"
}

func (s *workmailStore) firstResourceID() string {
	for id := range s.resources {
		return id
	}
	return "r-000001"
}

func (s *workmailStore) firstMailboxExportJobID() string {
	for id := range s.mailboxExportJobs {
		return id
	}
	return "mbej-000001"
}

func (s *workmailStore) ensureTags(resourceARN string) map[string]string {
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{}
	s.tags[resourceARN] = tags
	return tags
}

func (s *workmailStore) memberPayload(memberID string) map[string]any {
	if user, ok := s.users[memberID]; ok {
		return map[string]any{"Id": user.ID, "Name": user.Name, "Type": "USER", "State": user.State}
	}
	if group, ok := s.groups[memberID]; ok {
		return map[string]any{"Id": group.ID, "Name": group.Name, "Type": "GROUP", "State": group.State}
	}
	if resource, ok := s.resources[memberID]; ok {
		return map[string]any{"Id": resource.ID, "Name": resource.Name, "Type": "RESOURCE", "State": resource.State}
	}
	return map[string]any{"Id": memberID, "Name": memberID, "Type": "USER", "State": "ENABLED"}
}

func (s *workmailStore) entityType(entityID string) string {
	if _, ok := s.users[entityID]; ok {
		return "USER"
	}
	if _, ok := s.groups[entityID]; ok {
		return "GROUP"
	}
	if _, ok := s.resources[entityID]; ok {
		return "RESOURCE"
	}
	return "USER"
}

func (s *workmailStore) nextToken(prefix string) string {
	token := fmt.Sprintf("%s-%06d", prefix, s.nextID)
	s.nextID++
	return token
}

func workmailPayloadString(payload map[string]any, key, def string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return def
	}
	value := strings.TrimSpace(fmt.Sprintf("%v", raw))
	if value == "" {
		return def
	}
	return value
}

func workmailPayloadInt(payload map[string]any, key string, def int64) int64 {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return def
	}
	switch value := raw.(type) {
	case int:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return def
		}
		var out int64
		if _, err := fmt.Sscan(value, &out); err == nil {
			return out
		}
	}
	return def
}

func workmailPayloadBool(payload map[string]any, key string, def bool) bool {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return def
	}
	value, ok := raw.(bool)
	if ok {
		return value
	}
	text := strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", raw)))
	switch text {
	case "true", "1", "yes", "y":
		return true
	case "false", "0", "no", "n":
		return false
	default:
		return def
	}
}

func workmailPayloadStringSlice(payload map[string]any, key string) []string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		value := strings.TrimSpace(fmt.Sprintf("%v", item))
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func workmailPayloadTags(payload map[string]any) map[string]string {
	result := map[string]string{}
	raw, ok := payload["Tags"]
	if !ok || raw == nil {
		raw = payload["tags"]
	}
	switch tags := raw.(type) {
	case map[string]any:
		for key, value := range tags {
			result[key] = fmt.Sprintf("%v", value)
		}
	case map[string]string:
		for key, value := range tags {
			result[key] = value
		}
	case []any:
		for _, item := range tags {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := workmailPayloadString(entry, "Key", "")
			if key == "" {
				continue
			}
			result[key] = workmailPayloadString(entry, "Value", "")
		}
	}
	return result
}

func workmailOrgARN(orgID string) string {
	return fmt.Sprintf("arn:aws:workmail:us-east-1:123456789012:organization/%s", strings.TrimSpace(orgID))
}

func workmailEntityARN(entityID string) string {
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		entityID = "entity-000001"
	}
	return fmt.Sprintf("arn:aws:workmail:us-east-1:123456789012:entity/%s", entityID)
}
