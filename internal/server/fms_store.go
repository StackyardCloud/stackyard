package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type fmsStore struct {
	mu                 sync.Mutex
	nextID             int64
	adminAccount       string
	notification       map[string]any
	policies           map[string]map[string]any
	appsLists          map[string]map[string]any
	protocolsLists     map[string]map[string]any
	resourceSets       map[string]map[string]any
	resourceSetMembers map[string][]any
	tags               map[string]map[string]string
	thirdPartyStatus   string
}

func newFMSStore() *fmsStore {
	now := time.Now().UTC().Format(time.RFC3339)
	policyID := "policy-00000001"
	appsListID := "apps-00000001"
	protocolsListID := "protocols-00000001"
	resourceSetID := "rs-00000001"

	s := &fmsStore{
		nextID:       2,
		adminAccount: "123456789012",
		notification: map[string]any{
			"SnsTopicArn": "arn:aws:sns:us-east-1:123456789012:stackyard-fms",
			"SnsRoleName": "stackyard-fms-role",
		},
		policies: map[string]map[string]any{
			policyID: {
				"PolicyId":           policyID,
				"PolicyName":         "stackyard-default-policy",
				"PolicyDescription":  "seed policy",
				"PolicyUpdateToken":  "seed-token",
				"RemediationEnabled": true,
				"ResourceType":       "AWS::EC2::Instance",
				"SecurityServicePolicyData": map[string]any{
					"Type":               "WAF",
					"ManagedServiceData": "{}",
				},
				"PolicyStatus": "ACTIVE",
			},
		},
		appsLists: map[string]map[string]any{
			appsListID: {
				"ListId":          appsListID,
				"ListName":        "stackyard-apps",
				"ListUpdateToken": "seed-token",
				"CreateTime":      now,
				"LastUpdateTime":  now,
				"AppsList": []any{
					map[string]any{"AppName": "example", "Protocol": "TCP", "Port": 443},
				},
			},
		},
		protocolsLists: map[string]map[string]any{
			protocolsListID: {
				"ListId":          protocolsListID,
				"ListName":        "stackyard-protocols",
				"ListUpdateToken": "seed-token",
				"CreateTime":      now,
				"LastUpdateTime":  now,
				"ProtocolsList":   []any{"TCP", "UDP"},
			},
		},
		resourceSets: map[string]map[string]any{
			resourceSetID: {
				"Id":               resourceSetID,
				"Name":             "stackyard-resource-set",
				"Description":      "seed resource set",
				"ResourceTypeList": []any{"AWS::EC2::Instance"},
				"UpdateToken":      "seed-token",
				"LastUpdateTime":   now,
			},
		},
		resourceSetMembers: map[string][]any{
			resourceSetID: {
				map[string]any{
					"URI":                 "arn:aws:ec2:us-east-1:123456789012:instance/i-00000000000000001",
					"AccountId":           "123456789012",
					"Region":              "us-east-1",
					"ResourceType":        "AWS::EC2::Instance",
					"ResourceDescription": "seed instance",
				},
			},
		},
		tags:             map[string]map[string]string{},
		thirdPartyStatus: "NOT_ASSOCIATED",
	}

	s.tags[fmsPolicyARN(policyID)] = map[string]string{"seed": "true"}
	return s
}

func (s *fmsStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "AssociateAdminAccount", "PutAdminAccount":
		s.adminAccount = fmsString(payload, "AdminAccount", s.adminAccount)
		if strings.TrimSpace(s.adminAccount) == "" {
			s.adminAccount = "123456789012"
		}
		return map[string]any{}

	case "DisassociateAdminAccount":
		s.adminAccount = ""
		return map[string]any{}

	case "GetAdminAccount":
		acct := s.adminAccount
		if acct == "" {
			acct = "123456789012"
		}
		return map[string]any{"AdminAccount": acct, "RoleStatus": "READY"}

	case "GetAdminScope":
		return map[string]any{
			"AdminScope": map[string]any{
				"AccountScope": map[string]any{
					"Accounts":                 []any{fmsDefaultAdmin(s.adminAccount)},
					"AllAccountsEnabled":       true,
					"ExcludeSpecifiedAccounts": false,
				},
				"OrganizationalUnitScope": map[string]any{
					"OrganizationalUnits":                 []any{"ou-abcd-12345678"},
					"AllOrganizationalUnitsEnabled":       true,
					"ExcludeSpecifiedOrganizationalUnits": false,
				},
				"RegionScope": map[string]any{
					"Regions":           []any{"us-east-1"},
					"AllRegionsEnabled": true,
				},
				"PolicyTypeScope": map[string]any{
					"PolicyTypes":           []any{"WAF"},
					"AllPolicyTypesEnabled": true,
				},
			},
		}

	case "PutNotificationChannel":
		s.notification = map[string]any{
			"SnsTopicArn": fmsString(payload, "SnsTopicArn", "arn:aws:sns:us-east-1:123456789012:stackyard-fms"),
			"SnsRoleName": fmsString(payload, "SnsRoleName", "stackyard-fms-role"),
		}
		return map[string]any{}

	case "GetNotificationChannel":
		if s.notification == nil {
			s.notification = map[string]any{
				"SnsTopicArn": "arn:aws:sns:us-east-1:123456789012:stackyard-fms",
				"SnsRoleName": "stackyard-fms-role",
			}
		}
		return map[string]any{
			"SnsTopicArn": s.notification["SnsTopicArn"],
			"SnsRoleName": s.notification["SnsRoleName"],
		}

	case "DeleteNotificationChannel":
		s.notification = nil
		return map[string]any{}

	case "PutPolicy":
		policy := fmsMap(payload, "Policy")
		policyID := fmsString(policy, "PolicyId", "")
		if policyID == "" {
			policyID = fmt.Sprintf("policy-%08d", s.nextID)
			s.nextID++
		}
		if fmsString(policy, "PolicyName", "") == "" {
			policy["PolicyName"] = fmt.Sprintf("stackyard-policy-%s", policyID)
		}
		policy["PolicyId"] = policyID
		policy["PolicyUpdateToken"] = fmt.Sprintf("token-%08d", s.nextID)
		policy["PolicyStatus"] = fmsString(policy, "PolicyStatus", "ACTIVE")
		s.policies[policyID] = fmsCloneMap(policy)
		return map[string]any{
			"Policy":    fmsCloneMap(s.policies[policyID]),
			"PolicyArn": fmsPolicyARN(policyID),
		}

	case "GetPolicy":
		policyID := fmsString(payload, "PolicyId", "")
		policy := s.findPolicyLocked(policyID)
		return map[string]any{
			"Policy":    fmsCloneMap(policy),
			"PolicyArn": fmsPolicyARN(fmsString(policy, "PolicyId", "policy-00000001")),
		}

	case "DeletePolicy":
		policyID := fmsString(payload, "PolicyId", "")
		delete(s.policies, policyID)
		return map[string]any{}

	case "ListPolicies":
		ids := fmsSortedKeys(s.policies)
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			p := s.policies[id]
			out = append(out, map[string]any{
				"PolicyArn":           fmsPolicyARN(id),
				"PolicyId":            id,
				"PolicyName":          fmsString(p, "PolicyName", "stackyard-policy"),
				"ResourceType":        fmsString(p, "ResourceType", "AWS::EC2::Instance"),
				"SecurityServiceType": fmsString(fmsMap(p, "SecurityServicePolicyData"), "Type", "WAF"),
				"RemediationEnabled":  true,
			})
		}
		return map[string]any{"PolicyList": out, "NextToken": ""}

	case "ListComplianceStatus":
		policyID := fmsString(payload, "PolicyId", "")
		if policyID == "" {
			policyID = "policy-00000001"
		}
		return map[string]any{
			"PolicyComplianceStatusList": []any{
				map[string]any{
					"PolicyOwner":   "123456789012",
					"PolicyId":      policyID,
					"PolicyName":    fmsString(s.findPolicyLocked(policyID), "PolicyName", "stackyard-policy"),
					"MemberAccount": "111122223333",
					"EvaluationResults": []any{
						map[string]any{"ComplianceStatus": true, "ViolatorCount": 0},
					},
					"IssueInfoMap": map[string]any{},
				},
			},
			"NextToken": "",
		}

	case "GetComplianceDetail":
		policyID := fmsString(payload, "PolicyId", "policy-00000001")
		memberAccount := fmsString(payload, "MemberAccount", "111122223333")
		return map[string]any{
			"PolicyComplianceDetail": map[string]any{
				"PolicyOwner":             "123456789012",
				"PolicyId":                policyID,
				"MemberAccount":           memberAccount,
				"Violators":               []any{},
				"EvaluationLimitExceeded": false,
				"ExpiredAt":               now,
				"IssueInfoMap":            map[string]any{},
			},
		}

	case "GetViolationDetails":
		return map[string]any{
			"ViolationDetail": map[string]any{
				"PolicyId":      fmsString(payload, "PolicyId", "policy-00000001"),
				"MemberAccount": fmsString(payload, "MemberAccount", "111122223333"),
				"ResourceId":    fmsString(payload, "ResourceId", "arn:aws:ec2:us-east-1:123456789012:instance/i-00000000000000001"),
				"ResourceType":  fmsString(payload, "ResourceType", "AWS::EC2::Instance"),
			},
		}

	case "GetProtectionStatus":
		return map[string]any{"AdminAccountId": fmsDefaultAdmin(s.adminAccount), "Data": []any{}, "NextToken": ""}

	case "PutAppsList":
		data := fmsMap(payload, "AppsList")
		id := fmsString(data, "ListId", "")
		if id == "" {
			id = fmt.Sprintf("apps-%08d", s.nextID)
			s.nextID++
		}
		if fmsString(data, "ListName", "") == "" {
			data["ListName"] = fmt.Sprintf("stackyard-apps-%s", id)
		}
		data["ListId"] = id
		data["ListUpdateToken"] = fmt.Sprintf("token-%08d", s.nextID)
		data["CreateTime"] = fmsString(data, "CreateTime", now)
		data["LastUpdateTime"] = now
		s.appsLists[id] = fmsCloneMap(data)
		return map[string]any{"AppsList": fmsCloneMap(s.appsLists[id]), "AppsListArn": fmsAppsListARN(id)}

	case "GetAppsList":
		id := fmsString(payload, "ListId", "")
		item := s.findAppsListLocked(id)
		return map[string]any{"AppsList": fmsCloneMap(item), "AppsListArn": fmsAppsListARN(fmsString(item, "ListId", "apps-00000001"))}

	case "DeleteAppsList":
		delete(s.appsLists, fmsString(payload, "ListId", ""))
		return map[string]any{}

	case "ListAppsLists":
		ids := fmsSortedKeys(s.appsLists)
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			item := s.appsLists[id]
			out = append(out, map[string]any{
				"ListArn":        fmsAppsListARN(id),
				"ListId":         id,
				"ListName":       fmsString(item, "ListName", "stackyard-apps"),
				"CreateTime":     fmsString(item, "CreateTime", now),
				"LastUpdateTime": fmsString(item, "LastUpdateTime", now),
			})
		}
		return map[string]any{"AppsLists": out, "NextToken": ""}

	case "PutProtocolsList":
		data := fmsMap(payload, "ProtocolsList")
		id := fmsString(data, "ListId", "")
		if id == "" {
			id = fmt.Sprintf("protocols-%08d", s.nextID)
			s.nextID++
		}
		if fmsString(data, "ListName", "") == "" {
			data["ListName"] = fmt.Sprintf("stackyard-protocols-%s", id)
		}
		data["ListId"] = id
		data["ListUpdateToken"] = fmt.Sprintf("token-%08d", s.nextID)
		data["CreateTime"] = fmsString(data, "CreateTime", now)
		data["LastUpdateTime"] = now
		s.protocolsLists[id] = fmsCloneMap(data)
		return map[string]any{"ProtocolsList": fmsCloneMap(s.protocolsLists[id]), "ProtocolsListArn": fmsProtocolsListARN(id)}

	case "GetProtocolsList":
		id := fmsString(payload, "ListId", "")
		item := s.findProtocolsListLocked(id)
		return map[string]any{"ProtocolsList": fmsCloneMap(item), "ProtocolsListArn": fmsProtocolsListARN(fmsString(item, "ListId", "protocols-00000001"))}

	case "DeleteProtocolsList":
		delete(s.protocolsLists, fmsString(payload, "ListId", ""))
		return map[string]any{}

	case "ListProtocolsLists":
		ids := fmsSortedKeys(s.protocolsLists)
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			item := s.protocolsLists[id]
			out = append(out, map[string]any{
				"ListArn":        fmsProtocolsListARN(id),
				"ListId":         id,
				"ListName":       fmsString(item, "ListName", "stackyard-protocols"),
				"CreateTime":     fmsString(item, "CreateTime", now),
				"LastUpdateTime": fmsString(item, "LastUpdateTime", now),
			})
		}
		return map[string]any{"ProtocolsLists": out, "NextToken": ""}

	case "PutResourceSet":
		data := fmsMap(payload, "ResourceSet")
		id := fmsString(data, "Id", "")
		if id == "" {
			id = fmt.Sprintf("rs-%08d", s.nextID)
			s.nextID++
		}
		if fmsString(data, "Name", "") == "" {
			data["Name"] = fmt.Sprintf("stackyard-resource-set-%s", id)
		}
		if _, ok := data["ResourceTypeList"]; !ok {
			data["ResourceTypeList"] = []any{"AWS::EC2::Instance"}
		}
		data["Id"] = id
		data["UpdateToken"] = fmt.Sprintf("token-%08d", s.nextID)
		data["LastUpdateTime"] = now
		s.resourceSets[id] = fmsCloneMap(data)
		if _, ok := s.resourceSetMembers[id]; !ok {
			s.resourceSetMembers[id] = []any{}
		}
		return map[string]any{"ResourceSet": fmsCloneMap(s.resourceSets[id]), "ResourceSetArn": fmsResourceSetARN(id)}

	case "GetResourceSet":
		id := fmsString(payload, "Identifier", "")
		if id == "" {
			id = fmsString(payload, "ResourceSetId", "")
		}
		if id == "" {
			id = strings.TrimPrefix(fmsString(payload, "ResourceSetArn", ""), "arn:aws:fms:us-east-1:123456789012:resource-set/")
		}
		item := s.findResourceSetLocked(id)
		return map[string]any{"ResourceSet": fmsCloneMap(item), "ResourceSetArn": fmsResourceSetARN(fmsString(item, "Id", "rs-00000001"))}

	case "DeleteResourceSet":
		id := fmsString(payload, "Identifier", "")
		if id == "" {
			id = fmsString(payload, "ResourceSetId", "")
		}
		delete(s.resourceSets, id)
		delete(s.resourceSetMembers, id)
		return map[string]any{}

	case "ListResourceSets":
		ids := fmsSortedKeys(s.resourceSets)
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			item := s.resourceSets[id]
			out = append(out, map[string]any{
				"Id":                id,
				"Name":              fmsString(item, "Name", "stackyard-resource-set"),
				"ResourceSetStatus": "ACTIVE",
				"Description":       fmsString(item, "Description", ""),
				"LastUpdateTime":    fmsString(item, "LastUpdateTime", now),
			})
		}
		return map[string]any{"ResourceSets": out, "NextToken": ""}

	case "ListResourceSetResources":
		id := fmsString(payload, "Identifier", "")
		if id == "" {
			id = "rs-00000001"
		}
		items := s.resourceSetMembers[id]
		if items == nil {
			items = []any{}
		}
		return map[string]any{"Items": items, "NextToken": ""}

	case "BatchAssociateResource", "BatchDisassociateResource":
		return map[string]any{"FailedItems": []any{}}

	case "ListDiscoveredResources":
		return map[string]any{
			"Items": []any{
				map[string]any{
					"URI":                 "arn:aws:ec2:us-east-1:123456789012:instance/i-00000000000000001",
					"AccountId":           "123456789012",
					"Region":              "us-east-1",
					"ResourceType":        "AWS::EC2::Instance",
					"ResourceDescription": "seed instance",
				},
			},
			"NextToken": "",
		}

	case "ListAdminAccountsForOrganization":
		acct := fmsDefaultAdmin(s.adminAccount)
		return map[string]any{
			"AdminAccounts": []any{
				map[string]any{"AdminAccount": acct, "Status": "ACTIVE", "DefaultAdmin": true},
			},
			"NextToken": "",
		}

	case "ListAdminsManagingAccount":
		acct := fmsDefaultAdmin(s.adminAccount)
		return map[string]any{"AdminAccounts": []any{acct}, "NextToken": ""}

	case "ListMemberAccounts":
		return map[string]any{"MemberAccounts": []any{"111122223333"}, "NextToken": ""}

	case "AssociateThirdPartyFirewall":
		s.thirdPartyStatus = "ASSOCIATED"
		return map[string]any{}

	case "DisassociateThirdPartyFirewall":
		s.thirdPartyStatus = "NOT_ASSOCIATED"
		return map[string]any{}

	case "GetThirdPartyFirewallAssociationStatus":
		return map[string]any{"ThirdPartyFirewallAssociationStatus": s.thirdPartyStatus}

	case "ListThirdPartyFirewallFirewallPolicies":
		return map[string]any{
			"ThirdPartyFirewallFirewallPolicies": []any{
				map[string]any{
					"FirewallVendorName": "PALO_ALTO_NETWORKS",
					"FirewallPolicyName": "stackyard-third-party-policy",
					"PolicyStatus":       s.thirdPartyStatus,
				},
			},
			"NextToken": "",
		}

	case "TagResource":
		arn := fmsString(payload, "ResourceArn", fmsPolicyARN("policy-00000001"))
		tags := fmsTagMap(payload)
		curr := s.ensureTagsLocked(arn)
		for k, v := range tags {
			curr[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		arn := fmsString(payload, "ResourceArn", fmsPolicyARN("policy-00000001"))
		keys := fmsStringSlice(payload, "TagKeys")
		curr := s.ensureTagsLocked(arn)
		for _, k := range keys {
			delete(curr, k)
		}
		return map[string]any{}

	case "ListTagsForResource":
		arn := fmsString(payload, "ResourceArn", fmsPolicyARN("policy-00000001"))
		curr := s.ensureTagsLocked(arn)
		keys := make([]string, 0, len(curr))
		for k := range curr {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]any, 0, len(keys))
		for _, k := range keys {
			out = append(out, map[string]any{"Key": k, "Value": curr[k]})
		}
		return map[string]any{"TagList": out}
	}

	if strings.HasPrefix(action, "Delete") || strings.HasPrefix(action, "Update") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "List") {
		return map[string]any{"NextToken": ""}
	}
	if strings.HasPrefix(action, "Get") {
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *fmsStore) findPolicyLocked(policyID string) map[string]any {
	if policyID != "" {
		if item := s.policies[policyID]; item != nil {
			return item
		}
	}
	for _, id := range fmsSortedKeys(s.policies) {
		return s.policies[id]
	}
	return map[string]any{"PolicyId": "policy-00000001", "PolicyName": "stackyard-default-policy"}
}

func (s *fmsStore) findAppsListLocked(id string) map[string]any {
	if id != "" {
		if item := s.appsLists[id]; item != nil {
			return item
		}
	}
	for _, key := range fmsSortedKeys(s.appsLists) {
		return s.appsLists[key]
	}
	return map[string]any{"ListId": "apps-00000001", "ListName": "stackyard-apps", "AppsList": []any{}}
}

func (s *fmsStore) findProtocolsListLocked(id string) map[string]any {
	if id != "" {
		if item := s.protocolsLists[id]; item != nil {
			return item
		}
	}
	for _, key := range fmsSortedKeys(s.protocolsLists) {
		return s.protocolsLists[key]
	}
	return map[string]any{"ListId": "protocols-00000001", "ListName": "stackyard-protocols", "ProtocolsList": []any{"TCP"}}
}

func (s *fmsStore) findResourceSetLocked(id string) map[string]any {
	if id != "" {
		if item := s.resourceSets[id]; item != nil {
			return item
		}
	}
	for _, key := range fmsSortedKeys(s.resourceSets) {
		return s.resourceSets[key]
	}
	return map[string]any{"Id": "rs-00000001", "Name": "stackyard-resource-set", "ResourceTypeList": []any{"AWS::EC2::Instance"}}
}

func (s *fmsStore) ensureTagsLocked(resourceARN string) map[string]string {
	if strings.TrimSpace(resourceARN) == "" {
		resourceARN = fmsPolicyARN("policy-00000001")
	}
	if s.tags[resourceARN] == nil {
		s.tags[resourceARN] = map[string]string{}
	}
	return s.tags[resourceARN]
}

func fmsDefaultAdmin(account string) string {
	if strings.TrimSpace(account) == "" {
		return "123456789012"
	}
	return account
}

func fmsMap(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			if out, ok := v.(map[string]any); ok && out != nil {
				return out
			}
			break
		}
	}
	return map[string]any{}
}

func fmsString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s == "" || s == "<nil>" {
			return fallback
		}
		return s
	}
	return fallback
}

func fmsStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		if list, ok := v.([]any); ok {
			out := make([]string, 0, len(list))
			for _, item := range list {
				s := strings.TrimSpace(fmt.Sprintf("%v", item))
				if s != "" && s != "<nil>" {
					out = append(out, s)
				}
			}
			return out
		}
		break
	}
	return nil
}

func fmsTagMap(payload map[string]any) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	for k, v := range payload {
		if !strings.EqualFold(k, "TagList") {
			continue
		}
		if list, ok := v.([]any); ok {
			for _, item := range list {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				key := fmsString(m, "Key", "")
				val := fmsString(m, "Value", "")
				if key != "" {
					out[key] = val
				}
			}
			return out
		}
	}
	return out
}

func fmsCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func fmsSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func fmsPolicyARN(policyID string) string {
	return "arn:aws:fms:us-east-1:123456789012:policy/" + policyID
}

func fmsAppsListARN(id string) string {
	return "arn:aws:fms:us-east-1:123456789012:apps-list/" + id
}

func fmsProtocolsListARN(id string) string {
	return "arn:aws:fms:us-east-1:123456789012:protocols-list/" + id
}

func fmsResourceSetARN(id string) string {
	return "arn:aws:fms:us-east-1:123456789012:resource-set/" + id
}
