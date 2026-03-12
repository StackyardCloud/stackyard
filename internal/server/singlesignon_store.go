package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type singleSignOnStore struct {
	mu                       sync.Mutex
	nextID                   int64
	instance                 map[string]any
	regions                  map[string]bool
	permissionSets           map[string]map[string]any
	inlinePolicies           map[string]string
	permissionsBoundaries    map[string]map[string]any
	managedPolicies          map[string][]map[string]any
	customerManagedPolicies  map[string][]map[string]any
	accountAssignments       []map[string]any
	assignmentCreateStatuses map[string]map[string]any
	assignmentDeleteStatuses map[string]map[string]any
	provisionStatuses        map[string]map[string]any
	applications             map[string]map[string]any
	appAssignments           []map[string]any
	appAccessScopes          map[string]map[string]any
	appAuthMethods           map[string]map[string]any
	appGrants                map[string]map[string]any
	appSessionConfigs        map[string]map[string]any
	trustedTokenIssuers      map[string]map[string]any
	tags                     map[string]map[string]string
}

func newSingleSignOnStore() *singleSignOnStore {
	instArn := "arn:aws:sso:::instance/ssoins-0000000000000000"
	t := time.Now().UTC().Format(time.RFC3339)
	psArn := instArn + "/ps-0000000000000001"
	appArn := "arn:aws:sso::123456789012:application/ssoins-0000000000000000/apl-0000000000000001"
	ttiArn := "arn:aws:sso::123456789012:trustedTokenIssuer/ssoins-0000000000000000/tti-0000000000000001"

	s := &singleSignOnStore{
		nextID: 2,
		instance: map[string]any{
			"InstanceArn":     instArn,
			"IdentityStoreId": "d-1234567890",
			"OwnerAccountId":  "123456789012",
			"CreatedDate":     t,
			"Name":            "stackyard",
			"Status":          "ACTIVE",
		},
		regions:                  map[string]bool{"us-east-1": true},
		permissionSets:           map[string]map[string]any{},
		inlinePolicies:           map[string]string{},
		permissionsBoundaries:    map[string]map[string]any{},
		managedPolicies:          map[string][]map[string]any{},
		customerManagedPolicies:  map[string][]map[string]any{},
		accountAssignments:       []map[string]any{},
		assignmentCreateStatuses: map[string]map[string]any{},
		assignmentDeleteStatuses: map[string]map[string]any{},
		provisionStatuses:        map[string]map[string]any{},
		applications:             map[string]map[string]any{},
		appAssignments:           []map[string]any{},
		appAccessScopes:          map[string]map[string]any{},
		appAuthMethods:           map[string]map[string]any{},
		appGrants:                map[string]map[string]any{},
		appSessionConfigs:        map[string]map[string]any{},
		trustedTokenIssuers:      map[string]map[string]any{},
		tags:                     map[string]map[string]string{},
	}

	s.permissionSets[psArn] = map[string]any{
		"PermissionSetArn": psArn,
		"Name":             "stackyard-permission-set",
		"Description":      "seed permission set",
		"CreatedDate":      t,
		"SessionDuration":  "PT1H",
		"RelayState":       "",
		"InstanceArn":      instArn,
	}
	s.managedPolicies[psArn] = []map[string]any{{"Name": "ReadOnlyAccess", "Arn": "arn:aws:iam::aws:policy/ReadOnlyAccess"}}
	s.customerManagedPolicies[psArn] = []map[string]any{{"Name": "stackyard-managed", "Path": "/"}}
	s.inlinePolicies[psArn] = "{}"
	s.permissionsBoundaries[psArn] = map[string]any{"ManagedPolicyArn": "arn:aws:iam::aws:policy/ReadOnlyAccess"}

	s.applications[appArn] = map[string]any{
		"ApplicationArn":         appArn,
		"ApplicationAccount":     "123456789012",
		"ApplicationProviderArn": "arn:aws:sso::aws:applicationProvider/custom",
		"CreatedDate":            t,
		"InstanceArn":            instArn,
		"Name":                   "stackyard-application",
		"Description":            "seed application",
		"Status":                 "ENABLED",
		"PortalOptions":          map[string]any{"SignInOptions": map[string]any{"Origin": "APPLICATION"}},
	}
	s.appAccessScopes[appArn] = map[string]any{"Scope": "all", "AuthorizedTargets": []any{}}
	s.appAuthMethods[appArn] = ssoDefaultAuthenticationMethod()
	s.appGrants[appArn] = map[string]any{"AuthorizationCode": map[string]any{"RedirectUris": []any{"https://example.com/callback"}}}
	s.appSessionConfigs[appArn] = map[string]any{"SessionDuration": "PT1H"}

	s.trustedTokenIssuers[ttiArn] = map[string]any{
		"TrustedTokenIssuerArn":  ttiArn,
		"Name":                   "stackyard-tti",
		"TrustedTokenIssuerType": "OIDC_JWT",
		"TrustedTokenIssuerConfiguration": map[string]any{
			"OidcJwtConfiguration": map[string]any{
				"IssuerUrl":                  "https://issuer.example.com",
				"ClaimAttributePath":         "sub",
				"IdentityStoreAttributePath": "userName",
				"JwksRetrievalOption":        "OPEN_ID_DISCOVERY",
			},
		},
	}

	s.accountAssignments = append(s.accountAssignments, map[string]any{
		"AccountId":        "123456789012",
		"PermissionSetArn": psArn,
		"PrincipalId":      "11111111-2222-3333-4444-555555555555",
		"PrincipalType":    "USER",
	})

	s.tags[psArn] = map[string]string{"seed": "true"}
	return s
}

func (s *singleSignOnStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	instanceArn := s.instanceArn()

	switch action {
	case "CreateInstance":
		return map[string]any{"InstanceArn": instanceArn}
	case "DescribeInstance":
		return ssoInstanceMetadata(s.instance)
	case "ListInstances":
		return map[string]any{"Instances": []any{ssoInstanceMetadata(s.instance)}, "NextToken": ""}
	case "UpdateInstance", "DeleteInstance":
		return map[string]any{}
	case "AddRegion":
		region := ssoPayloadString(payload, "Region", "us-east-1")
		s.regions[region] = true
		return map[string]any{}
	case "RemoveRegion":
		region := ssoPayloadString(payload, "Region", "us-east-1")
		delete(s.regions, region)
		if len(s.regions) == 0 {
			s.regions["us-east-1"] = true
		}
		return map[string]any{}
	case "DescribeRegion":
		region := ssoPayloadString(payload, "Region", "us-east-1")
		return map[string]any{"Region": map[string]any{"Region": region, "Status": "ACTIVE"}}
	case "ListRegions":
		items := make([]any, 0, len(s.regions))
		for _, r := range sortedStringKeysBool(s.regions) {
			items = append(items, map[string]any{"Region": r, "Status": "ACTIVE"})
		}
		return map[string]any{"Regions": items, "NextToken": ""}
	case "CreateInstanceAccessControlAttributeConfiguration", "UpdateInstanceAccessControlAttributeConfiguration", "DeleteInstanceAccessControlAttributeConfiguration":
		return map[string]any{}
	case "DescribeInstanceAccessControlAttributeConfiguration":
		return map[string]any{"InstanceAccessControlAttributeConfiguration": map[string]any{"AccessControlAttributes": []any{}}}

	case "CreatePermissionSet":
		name := ssoPayloadString(payload, "Name", fmt.Sprintf("stackyard-ps-%06d", s.nextID))
		arn := instanceArn + "/ps-" + s.nextTokenLocked(16)
		ps := map[string]any{
			"PermissionSetArn": arn,
			"Name":             name,
			"Description":      ssoPayloadString(payload, "Description", ""),
			"CreatedDate":      now,
			"SessionDuration":  ssoPayloadString(payload, "SessionDuration", "PT1H"),
			"RelayState":       ssoPayloadString(payload, "RelayState", ""),
			"InstanceArn":      ssoPayloadString(payload, "InstanceArn", instanceArn),
		}
		s.permissionSets[arn] = ps
		s.inlinePolicies[arn] = "{}"
		if s.permissionsBoundaries[arn] == nil {
			s.permissionsBoundaries[arn] = map[string]any{"ManagedPolicyArn": "arn:aws:iam::aws:policy/ReadOnlyAccess"}
		}
		return map[string]any{"PermissionSet": cloneAnyMap(ps)}
	case "DescribePermissionSet":
		arn := ssoPayloadString(payload, "PermissionSetArn", s.anyPermissionSetArn())
		return map[string]any{"PermissionSet": cloneAnyMap(s.ensurePermissionSetLocked(arn))}
	case "ListPermissionSets":
		arns := sortedStringKeysAnyMap(s.permissionSets)
		out := make([]any, 0, len(arns))
		for _, arn := range arns {
			out = append(out, arn)
		}
		return map[string]any{"PermissionSets": out, "NextToken": ""}
	case "UpdatePermissionSet", "DeletePermissionSet":
		if action == "DeletePermissionSet" {
			delete(s.permissionSets, ssoPayloadString(payload, "PermissionSetArn", ""))
		}
		return map[string]any{}

	case "AttachManagedPolicyToPermissionSet":
		psArn := ssoPayloadString(payload, "PermissionSetArn", s.anyPermissionSetArn())
		policyArn := ssoPayloadString(payload, "ManagedPolicyArn", "arn:aws:iam::aws:policy/ReadOnlyAccess")
		s.managedPolicies[psArn] = append(s.managedPolicies[psArn], map[string]any{"Name": nameFromARN(policyArn), "Arn": policyArn})
		return map[string]any{}
	case "DetachManagedPolicyFromPermissionSet":
		return map[string]any{}
	case "ListManagedPoliciesInPermissionSet":
		psArn := ssoPayloadString(payload, "PermissionSetArn", s.anyPermissionSetArn())
		items := toAnySliceFromMapSlice(s.managedPolicies[psArn])
		return map[string]any{"AttachedManagedPolicies": items, "NextToken": ""}

	case "AttachCustomerManagedPolicyReferenceToPermissionSet":
		psArn := ssoPayloadString(payload, "PermissionSetArn", s.anyPermissionSetArn())
		m := map[string]any{"Name": "stackyard-managed", "Path": "/"}
		if ref, ok := payloadCaseInsensitiveMap(payload, "CustomerManagedPolicyReference"); ok {
			m = map[string]any{"Name": ssoPayloadString(ref, "Name", "stackyard-managed"), "Path": ssoPayloadString(ref, "Path", "/")}
		}
		s.customerManagedPolicies[psArn] = append(s.customerManagedPolicies[psArn], m)
		return map[string]any{}
	case "DetachCustomerManagedPolicyReferenceFromPermissionSet":
		return map[string]any{}
	case "ListCustomerManagedPolicyReferencesInPermissionSet":
		psArn := ssoPayloadString(payload, "PermissionSetArn", s.anyPermissionSetArn())
		items := toAnySliceFromMapSlice(s.customerManagedPolicies[psArn])
		return map[string]any{"CustomerManagedPolicyReferences": items, "NextToken": ""}

	case "PutInlinePolicyToPermissionSet":
		psArn := ssoPayloadString(payload, "PermissionSetArn", s.anyPermissionSetArn())
		s.inlinePolicies[psArn] = ssoPayloadString(payload, "InlinePolicy", "{}")
		return map[string]any{}
	case "GetInlinePolicyForPermissionSet":
		psArn := ssoPayloadString(payload, "PermissionSetArn", s.anyPermissionSetArn())
		return map[string]any{"InlinePolicy": defaultString(s.inlinePolicies[psArn], "{}")}
	case "DeleteInlinePolicyFromPermissionSet":
		delete(s.inlinePolicies, ssoPayloadString(payload, "PermissionSetArn", ""))
		return map[string]any{}

	case "PutPermissionsBoundaryToPermissionSet":
		psArn := ssoPayloadString(payload, "PermissionSetArn", s.anyPermissionSetArn())
		if pb, ok := payloadCaseInsensitiveMap(payload, "PermissionsBoundary"); ok {
			s.permissionsBoundaries[psArn] = pb
		}
		return map[string]any{}
	case "GetPermissionsBoundaryForPermissionSet":
		psArn := ssoPayloadString(payload, "PermissionSetArn", s.anyPermissionSetArn())
		pb := s.permissionsBoundaries[psArn]
		if pb == nil {
			pb = map[string]any{"ManagedPolicyArn": "arn:aws:iam::aws:policy/ReadOnlyAccess"}
		}
		return map[string]any{"PermissionsBoundary": cloneAnyMap(pb)}
	case "DeletePermissionsBoundaryFromPermissionSet":
		delete(s.permissionsBoundaries, ssoPayloadString(payload, "PermissionSetArn", ""))
		return map[string]any{}

	case "ProvisionPermissionSet":
		id := s.nextTokenLocked(36)
		st := map[string]any{
			"Status":           "SUCCEEDED",
			"RequestId":        id,
			"PermissionSetArn": ssoPayloadString(payload, "PermissionSetArn", s.anyPermissionSetArn()),
			"CreatedDate":      now,
			"AccountId":        ssoPayloadString(payload, "TargetId", "123456789012"),
		}
		s.provisionStatuses[id] = st
		return map[string]any{"PermissionSetProvisioningStatus": cloneAnyMap(st)}
	case "DescribePermissionSetProvisioningStatus":
		id := ssoPayloadString(payload, "ProvisionPermissionSetRequestId", "")
		if id == "" {
			id = firstKeyAnyMap(s.provisionStatuses)
		}
		st := s.provisionStatuses[id]
		if st == nil {
			st = map[string]any{"Status": "SUCCEEDED", "RequestId": id, "PermissionSetArn": s.anyPermissionSetArn(), "CreatedDate": now, "AccountId": "123456789012"}
		}
		return map[string]any{"PermissionSetProvisioningStatus": cloneAnyMap(st)}
	case "ListPermissionSetProvisioningStatus":
		items := make([]any, 0, len(s.provisionStatuses))
		for _, id := range sortedStringKeysAnyMap(s.provisionStatuses) {
			items = append(items, cloneAnyMap(s.provisionStatuses[id]))
		}
		if len(items) == 0 {
			items = append(items, map[string]any{"Status": "SUCCEEDED", "RequestId": s.nextTokenLocked(36), "PermissionSetArn": s.anyPermissionSetArn(), "CreatedDate": now, "AccountId": "123456789012"})
		}
		return map[string]any{"PermissionSetsProvisioningStatus": items, "NextToken": ""}

	case "CreateAccountAssignment":
		status := map[string]any{
			"Status":           "SUCCEEDED",
			"RequestId":        s.nextTokenLocked(36),
			"CreatedDate":      now,
			"PermissionSetArn": ssoPayloadString(payload, "PermissionSetArn", s.anyPermissionSetArn()),
			"PrincipalId":      ssoPayloadString(payload, "PrincipalId", "11111111-2222-3333-4444-555555555555"),
			"PrincipalType":    ssoPayloadString(payload, "PrincipalType", "USER"),
			"TargetId":         ssoPayloadString(payload, "TargetId", "123456789012"),
			"TargetType":       ssoPayloadString(payload, "TargetType", "AWS_ACCOUNT"),
		}
		s.assignmentCreateStatuses[fmt.Sprintf("%v", status["RequestId"])] = status
		s.accountAssignments = append(s.accountAssignments, map[string]any{
			"AccountId":        status["TargetId"],
			"PermissionSetArn": status["PermissionSetArn"],
			"PrincipalId":      status["PrincipalId"],
			"PrincipalType":    status["PrincipalType"],
		})
		return map[string]any{"AccountAssignmentCreationStatus": cloneAnyMap(status)}
	case "DeleteAccountAssignment":
		status := map[string]any{
			"Status":           "SUCCEEDED",
			"RequestId":        s.nextTokenLocked(36),
			"CreatedDate":      now,
			"PermissionSetArn": ssoPayloadString(payload, "PermissionSetArn", s.anyPermissionSetArn()),
			"PrincipalId":      ssoPayloadString(payload, "PrincipalId", "11111111-2222-3333-4444-555555555555"),
			"PrincipalType":    ssoPayloadString(payload, "PrincipalType", "USER"),
			"TargetId":         ssoPayloadString(payload, "TargetId", "123456789012"),
			"TargetType":       ssoPayloadString(payload, "TargetType", "AWS_ACCOUNT"),
		}
		s.assignmentDeleteStatuses[fmt.Sprintf("%v", status["RequestId"])] = status
		return map[string]any{"AccountAssignmentDeletionStatus": cloneAnyMap(status)}
	case "DescribeAccountAssignmentCreationStatus":
		id := ssoPayloadString(payload, "AccountAssignmentCreationRequestId", firstKeyAnyMap(s.assignmentCreateStatuses))
		st := s.assignmentCreateStatuses[id]
		if st == nil {
			st = map[string]any{"Status": "SUCCEEDED", "RequestId": id, "CreatedDate": now}
		}
		return map[string]any{"AccountAssignmentCreationStatus": cloneAnyMap(st)}
	case "DescribeAccountAssignmentDeletionStatus":
		id := ssoPayloadString(payload, "AccountAssignmentDeletionRequestId", firstKeyAnyMap(s.assignmentDeleteStatuses))
		st := s.assignmentDeleteStatuses[id]
		if st == nil {
			st = map[string]any{"Status": "SUCCEEDED", "RequestId": id, "CreatedDate": now}
		}
		return map[string]any{"AccountAssignmentDeletionStatus": cloneAnyMap(st)}
	case "ListAccountAssignmentCreationStatus":
		items := make([]any, 0, len(s.assignmentCreateStatuses))
		for _, id := range sortedStringKeysAnyMap(s.assignmentCreateStatuses) {
			items = append(items, cloneAnyMap(s.assignmentCreateStatuses[id]))
		}
		return map[string]any{"AccountAssignmentsCreationStatus": items, "NextToken": ""}
	case "ListAccountAssignmentDeletionStatus":
		items := make([]any, 0, len(s.assignmentDeleteStatuses))
		for _, id := range sortedStringKeysAnyMap(s.assignmentDeleteStatuses) {
			items = append(items, cloneAnyMap(s.assignmentDeleteStatuses[id]))
		}
		return map[string]any{"AccountAssignmentsDeletionStatus": items, "NextToken": ""}
	case "ListAccountAssignments", "ListAccountAssignmentsForPrincipal":
		items := make([]any, 0, len(s.accountAssignments))
		for _, a := range s.accountAssignments {
			items = append(items, cloneAnyMap(a))
		}
		if action == "ListAccountAssignmentsForPrincipal" {
			return map[string]any{"AccountAssignments": items, "NextToken": ""}
		}
		return map[string]any{"AccountAssignments": items, "NextToken": ""}
	case "ListAccountsForProvisionedPermissionSet":
		return map[string]any{"AccountIds": []any{"123456789012"}, "NextToken": ""}
	case "ListPermissionSetsProvisionedToAccount":
		ps := sortedStringKeysAnyMap(s.permissionSets)
		out := make([]any, 0, len(ps))
		for _, arn := range ps {
			out = append(out, arn)
		}
		return map[string]any{"PermissionSets": out, "NextToken": ""}

	case "CreateApplication":
		arn := fmt.Sprintf("arn:aws:sso::123456789012:application/ssoins-0000000000000000/apl-%s", s.nextTokenLocked(16))
		app := map[string]any{
			"ApplicationArn":         arn,
			"ApplicationAccount":     "123456789012",
			"ApplicationProviderArn": ssoPayloadString(payload, "ApplicationProviderArn", "arn:aws:sso::aws:applicationProvider/custom"),
			"CreatedDate":            now,
			"InstanceArn":            ssoPayloadString(payload, "InstanceArn", instanceArn),
			"Name":                   ssoPayloadString(payload, "Name", "stackyard-application"),
			"Description":            ssoPayloadString(payload, "Description", ""),
			"Status":                 "ENABLED",
			"PortalOptions":          map[string]any{"SignInOptions": map[string]any{"Origin": "APPLICATION"}},
		}
		s.applications[arn] = app
		s.appAccessScopes[arn] = map[string]any{"Scope": "all", "AuthorizedTargets": []any{}}
		s.appAuthMethods[arn] = ssoDefaultAuthenticationMethod()
		s.appGrants[arn] = map[string]any{"AuthorizationCode": map[string]any{"RedirectUris": []any{"https://example.com/callback"}}}
		s.appSessionConfigs[arn] = map[string]any{"SessionDuration": "PT1H"}
		return map[string]any{"ApplicationArn": arn}
	case "DescribeApplication":
		arn := ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications))
		return cloneAnyMap(s.ensureApplicationLocked(arn))
	case "ListApplications":
		items := make([]any, 0, len(s.applications))
		for _, arn := range sortedStringKeysAnyMap(s.applications) {
			items = append(items, cloneAnyMap(s.applications[arn]))
		}
		return map[string]any{"Applications": items, "NextToken": ""}
	case "UpdateApplication", "DeleteApplication":
		if action == "DeleteApplication" {
			arn := ssoPayloadString(payload, "ApplicationArn", "")
			delete(s.applications, arn)
		}
		return map[string]any{}
	case "CreateApplicationAssignment":
		assignment := map[string]any{
			"ApplicationArn": ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications)),
			"PrincipalId":    ssoPayloadString(payload, "PrincipalId", "11111111-2222-3333-4444-555555555555"),
			"PrincipalType":  ssoPayloadString(payload, "PrincipalType", "USER"),
		}
		s.appAssignments = append(s.appAssignments, assignment)
		return map[string]any{}
	case "DescribeApplicationAssignment":
		if len(s.appAssignments) > 0 {
			return cloneAnyMap(s.appAssignments[0])
		}
		return map[string]any{
			"ApplicationArn": firstKeyAnyMap(s.applications),
			"PrincipalId":    "11111111-2222-3333-4444-555555555555",
			"PrincipalType":  "USER",
		}
	case "DeleteApplicationAssignment":
		return map[string]any{}
	case "ListApplicationAssignments", "ListApplicationAssignmentsForPrincipal":
		items := make([]any, 0, len(s.appAssignments))
		for _, a := range s.appAssignments {
			items = append(items, cloneAnyMap(a))
		}
		if action == "ListApplicationAssignmentsForPrincipal" {
			return map[string]any{"ApplicationAssignments": items, "NextToken": ""}
		}
		return map[string]any{"ApplicationAssignments": items, "NextToken": ""}
	case "PutApplicationAccessScope":
		arn := ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications))
		s.appAccessScopes[arn] = map[string]any{"Scope": ssoPayloadString(payload, "Scope", "all"), "AuthorizedTargets": []any{}}
		return map[string]any{}
	case "GetApplicationAccessScope":
		arn := ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications))
		m := s.appAccessScopes[arn]
		if m == nil {
			m = map[string]any{"Scope": "all", "AuthorizedTargets": []any{}}
		}
		return cloneAnyMap(m)
	case "DeleteApplicationAccessScope":
		return map[string]any{}
	case "ListApplicationAccessScopes":
		arn := ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications))
		m := s.appAccessScopes[arn]
		if m == nil {
			m = map[string]any{"Scope": "all", "AuthorizedTargets": []any{}}
		}
		return map[string]any{"Scopes": []any{cloneAnyMap(m)}, "NextToken": ""}
	case "PutApplicationAuthenticationMethod":
		arn := ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications))
		method, ok := payloadCaseInsensitiveMap(payload, "AuthenticationMethod")
		if !ok || len(method) == 0 {
			method = ssoDefaultAuthenticationMethod()
		}
		s.appAuthMethods[arn] = cloneAnyMap(method)
		return map[string]any{}
	case "GetApplicationAuthenticationMethod":
		arn := ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications))
		m := s.appAuthMethods[arn]
		if m == nil {
			m = ssoDefaultAuthenticationMethod()
		}
		return map[string]any{"AuthenticationMethod": cloneAnyMap(m)}
	case "DeleteApplicationAuthenticationMethod":
		return map[string]any{}
	case "ListApplicationAuthenticationMethods":
		arn := ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications))
		m := s.appAuthMethods[arn]
		if m == nil {
			m = ssoDefaultAuthenticationMethod()
		}
		return map[string]any{
			"AuthenticationMethods": []any{
				map[string]any{
					"AuthenticationMethodType": "IAM",
					"AuthenticationMethod":     cloneAnyMap(m),
				},
			},
			"NextToken": "",
		}
	case "PutApplicationGrant":
		arn := ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications))
		grant := map[string]any{
			"AuthorizationCode": map[string]any{
				"RedirectUris": []any{"https://example.com/callback"},
			},
		}
		if raw, ok := payloadCaseInsensitiveMap(payload, "Grant"); ok && len(raw) > 0 {
			grant = raw
		}
		s.appGrants[arn] = grant
		return map[string]any{}
	case "GetApplicationGrant":
		arn := ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications))
		m := s.appGrants[arn]
		if m == nil {
			m = map[string]any{"AuthorizationCode": map[string]any{"RedirectUris": []any{"https://example.com/callback"}}}
		}
		return map[string]any{"Grant": cloneAnyMap(m)}
	case "DeleteApplicationGrant":
		return map[string]any{}
	case "ListApplicationGrants":
		arn := ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications))
		m := s.appGrants[arn]
		if m == nil {
			m = map[string]any{"AuthorizationCode": map[string]any{"RedirectUris": []any{"https://example.com/callback"}}}
		}
		return map[string]any{
			"Grants": []any{
				map[string]any{
					"GrantType": "authorization_code",
					"Grant":     cloneAnyMap(m),
				},
			},
			"NextToken": "",
		}
	case "PutApplicationSessionConfiguration":
		arn := ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications))
		s.appSessionConfigs[arn] = map[string]any{"SessionDuration": ssoPayloadString(payload, "SessionDuration", "PT1H")}
		return map[string]any{}
	case "GetApplicationSessionConfiguration":
		arn := ssoPayloadString(payload, "ApplicationArn", firstKeyAnyMap(s.applications))
		m := s.appSessionConfigs[arn]
		if m == nil {
			m = map[string]any{"SessionDuration": "PT1H"}
		}
		return map[string]any{"SessionConfiguration": cloneAnyMap(m)}
	case "PutApplicationAssignmentConfiguration":
		return map[string]any{}
	case "GetApplicationAssignmentConfiguration":
		return map[string]any{"AssignmentRequired": false}
	case "ListApplicationProviders":
		return map[string]any{"ApplicationProviders": []any{s.defaultApplicationProvider()}, "NextToken": ""}
	case "DescribeApplicationProvider":
		return s.defaultApplicationProvider()

	case "CreateTrustedTokenIssuer":
		arn := fmt.Sprintf("arn:aws:sso::123456789012:trustedTokenIssuer/ssoins-0000000000000000/tti-%s", s.nextTokenLocked(16))
		config, ok := payloadCaseInsensitiveMap(payload, "TrustedTokenIssuerConfiguration")
		if !ok || len(config) == 0 {
			config = map[string]any{
				"OidcJwtConfiguration": map[string]any{
					"IssuerUrl":                  "https://issuer.example.com",
					"ClaimAttributePath":         "sub",
					"IdentityStoreAttributePath": "userName",
					"JwksRetrievalOption":        "OPEN_ID_DISCOVERY",
				},
			}
		}
		tti := map[string]any{
			"TrustedTokenIssuerArn":           arn,
			"Name":                            ssoPayloadString(payload, "Name", "stackyard-tti"),
			"TrustedTokenIssuerType":          ssoPayloadString(payload, "TrustedTokenIssuerType", "OIDC_JWT"),
			"TrustedTokenIssuerConfiguration": cloneAnyMap(config),
		}
		s.trustedTokenIssuers[arn] = tti
		return map[string]any{"TrustedTokenIssuerArn": arn}
	case "DescribeTrustedTokenIssuer":
		arn := ssoPayloadString(payload, "TrustedTokenIssuerArn", firstKeyAnyMap(s.trustedTokenIssuers))
		return cloneAnyMap(s.ensureTrustedTokenIssuerLocked(arn))
	case "UpdateTrustedTokenIssuer", "DeleteTrustedTokenIssuer":
		if action == "DeleteTrustedTokenIssuer" {
			delete(s.trustedTokenIssuers, ssoPayloadString(payload, "TrustedTokenIssuerArn", ""))
		}
		return map[string]any{}
	case "ListTrustedTokenIssuers":
		items := make([]any, 0, len(s.trustedTokenIssuers))
		for _, arn := range sortedStringKeysAnyMap(s.trustedTokenIssuers) {
			items = append(items, cloneAnyMap(s.trustedTokenIssuers[arn]))
		}
		return map[string]any{"TrustedTokenIssuers": items, "NextToken": ""}

	case "TagResource":
		arn := ssoPayloadString(payload, "ResourceArn", s.anyPermissionSetArn())
		tags := ssoTagMap(payload)
		m := s.ensureTagsLocked(arn)
		for k, v := range tags {
			m[k] = v
		}
		return map[string]any{}
	case "UntagResource":
		arn := ssoPayloadString(payload, "ResourceArn", s.anyPermissionSetArn())
		keys := ssoStringSlice(payload, "TagKeys")
		m := s.ensureTagsLocked(arn)
		for _, k := range keys {
			delete(m, k)
		}
		return map[string]any{}
	case "ListTagsForResource":
		arn := ssoPayloadString(payload, "ResourceArn", s.anyPermissionSetArn())
		m := s.ensureTagsLocked(arn)
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]any, 0, len(keys))
		for _, k := range keys {
			out = append(out, map[string]any{"Key": k, "Value": m[k]})
		}
		return map[string]any{"Tags": out}
	}

	if strings.HasPrefix(action, "Delete") || strings.HasPrefix(action, "Update") || strings.HasPrefix(action, "Put") || strings.HasPrefix(action, "Attach") || strings.HasPrefix(action, "Detach") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "List") {
		return map[string]any{"NextToken": ""}
	}
	if strings.HasPrefix(action, "Describe") || strings.HasPrefix(action, "Get") || strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Provision") {
		return map[string]any{}
	}
	return map[string]any{}
}

func (s *singleSignOnStore) instanceArn() string {
	if arn, ok := s.instance["InstanceArn"].(string); ok && strings.TrimSpace(arn) != "" {
		return arn
	}
	return "arn:aws:sso:::instance/ssoins-0000000000000000"
}

func (s *singleSignOnStore) anyPermissionSetArn() string {
	if arn := firstKeyAnyMap(s.permissionSets); arn != "" {
		return arn
	}
	return s.instanceArn() + "/ps-0000000000000001"
}

func (s *singleSignOnStore) ensurePermissionSetLocked(arn string) map[string]any {
	if ps := s.permissionSets[arn]; ps != nil {
		return ps
	}
	ps := map[string]any{
		"PermissionSetArn": arn,
		"Name":             "stackyard-permission-set",
		"Description":      "",
		"CreatedDate":      time.Now().UTC().Format(time.RFC3339),
		"SessionDuration":  "PT1H",
		"RelayState":       "",
		"InstanceArn":      s.instanceArn(),
	}
	s.permissionSets[arn] = ps
	return ps
}

func (s *singleSignOnStore) ensureApplicationLocked(arn string) map[string]any {
	if app := s.applications[arn]; app != nil {
		return app
	}
	app := map[string]any{
		"ApplicationArn":         arn,
		"ApplicationAccount":     "123456789012",
		"ApplicationProviderArn": "arn:aws:sso::aws:applicationProvider/custom",
		"CreatedDate":            time.Now().UTC().Format(time.RFC3339),
		"InstanceArn":            s.instanceArn(),
		"Name":                   "stackyard-application",
		"Description":            "",
		"Status":                 "ENABLED",
		"PortalOptions":          map[string]any{"SignInOptions": map[string]any{"Origin": "APPLICATION"}},
	}
	s.applications[arn] = app
	return app
}

func (s *singleSignOnStore) ensureTrustedTokenIssuerLocked(arn string) map[string]any {
	if tti := s.trustedTokenIssuers[arn]; tti != nil {
		return tti
	}
	tti := map[string]any{
		"TrustedTokenIssuerArn":           arn,
		"Name":                            "stackyard-tti",
		"TrustedTokenIssuerType":          "OIDC_JWT",
		"TrustedTokenIssuerConfiguration": map[string]any{},
	}
	s.trustedTokenIssuers[arn] = tti
	return tti
}

func (s *singleSignOnStore) ensureTagsLocked(resourceARN string) map[string]string {
	if strings.TrimSpace(resourceARN) == "" {
		resourceARN = s.anyPermissionSetArn()
	}
	if s.tags[resourceARN] == nil {
		s.tags[resourceARN] = map[string]string{}
	}
	return s.tags[resourceARN]
}

func (s *singleSignOnStore) nextTokenLocked(width int) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%0*d", width, id)
}

func (s *singleSignOnStore) defaultApplicationProvider() map[string]any {
	return map[string]any{
		"ApplicationProviderArn": "arn:aws:sso::aws:applicationProvider/custom",
		"DisplayData":            map[string]any{"DisplayName": "Custom", "IconUrl": "https://example.com/icon.png"},
		"FederationProtocol":     "SAML",
	}
}

func ssoDefaultAuthenticationMethod() map[string]any {
	return map[string]any{
		"Iam": map[string]any{
			"ActorPolicy": map[string]any{
				"Version":   "2012-10-17",
				"Statement": []any{},
			},
		},
	}
}

func ssoInstanceMetadata(instance map[string]any) map[string]any {
	return map[string]any{
		"CreatedDate":     instance["CreatedDate"],
		"IdentityStoreId": instance["IdentityStoreId"],
		"InstanceArn":     instance["InstanceArn"],
		"Name":            instance["Name"],
		"OwnerAccountId":  instance["OwnerAccountId"],
		"Status":          instance["Status"],
	}
}

func sortedStringKeysBool(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedStringKeysAnyMap(m map[string]map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func firstKeyAnyMap(m map[string]map[string]any) string {
	keys := sortedStringKeysAnyMap(m)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func toAnySliceFromMapSlice(in []map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, cloneAnyMap(item))
	}
	return out
}

func payloadCaseInsensitiveMap(payload map[string]any, key string) (map[string]any, bool) {
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			if m, ok := v.(map[string]any); ok {
				return m, true
			}
		}
	}
	return nil, false
}

func ssoPayloadString(payload map[string]any, key, fallback string) string {
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s == "" || s == "<nil>" {
				return fallback
			}
			return s
		}
	}
	return fallback
}

func ssoStringSlice(payload map[string]any, key string) []string {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		if arr, ok := v.([]any); ok {
			out := make([]string, 0, len(arr))
			for _, item := range arr {
				s := strings.TrimSpace(fmt.Sprintf("%v", item))
				if s != "" && s != "<nil>" {
					out = append(out, s)
				}
			}
			return out
		}
	}
	return nil
}

func ssoTagMap(payload map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range payload {
		if !strings.EqualFold(k, "Tags") {
			continue
		}
		if tags, ok := v.([]any); ok {
			for _, item := range tags {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				key := ssoPayloadString(m, "Key", "")
				val := ssoPayloadString(m, "Value", "")
				if key != "" {
					out[key] = val
				}
			}
		}
	}
	return out
}

func nameFromARN(arn string) string {
	if idx := strings.LastIndex(arn, "/"); idx >= 0 && idx+1 < len(arn) {
		return arn[idx+1:]
	}
	return arn
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
