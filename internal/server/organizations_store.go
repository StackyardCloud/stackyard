package server

import (
	"fmt"
	"strings"
	"sync"
)

type organizationsStore struct {
	mu     sync.Mutex
	nextID int64
}

func newOrganizationsStore() *organizationsStore {
	return &organizationsStore{nextID: 1}
}

func (s *organizationsStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateOrganization":
		return map[string]any{
			"Organization": map[string]any{
				"Id":                 "o-" + s.nextTokenLocked(10),
				"Arn":                "arn:aws:organizations::123456789012:organization/o-" + s.nextTokenLocked(10),
				"FeatureSet":         organizationsPayloadString(payload, "FeatureSet", "ALL"),
				"MasterAccountArn":   "arn:aws:organizations::123456789012:account/o-root/123456789012",
				"MasterAccountId":    "123456789012",
				"MasterAccountEmail": "owner@example.com",
			},
		}
	case "DescribeOrganization":
		return map[string]any{
			"Organization": map[string]any{
				"Id":                 "o-stackyard",
				"Arn":                "arn:aws:organizations::123456789012:organization/o-stackyard",
				"FeatureSet":         "ALL",
				"MasterAccountArn":   "arn:aws:organizations::123456789012:account/o-stackyard/123456789012",
				"MasterAccountId":    "123456789012",
				"MasterAccountEmail": "owner@example.com",
			},
		}
	case "DescribeAccount":
		accountID := organizationsPayloadString(payload, "AccountId", "123456789012")
		return map[string]any{
			"Account": map[string]any{
				"Id":     accountID,
				"Arn":    "arn:aws:organizations::123456789012:account/o-stackyard/" + accountID,
				"Email":  "member@example.com",
				"Name":   "stackyard-account",
				"Status": "ACTIVE",
			},
		}
	case "DescribeOrganizationalUnit":
		ouID := organizationsPayloadString(payload, "OrganizationalUnitId", "ou-stackyard-001")
		return map[string]any{
			"OrganizationalUnit": map[string]any{
				"Id":   ouID,
				"Arn":  "arn:aws:organizations::123456789012:ou/o-stackyard/" + ouID,
				"Name": "stackyard-ou",
			},
		}
	case "DescribePolicy":
		policyID := organizationsPayloadString(payload, "PolicyId", "p-stackyard")
		return map[string]any{
			"Policy": map[string]any{
				"PolicySummary": map[string]any{
					"Id":   policyID,
					"Arn":  "arn:aws:organizations::123456789012:policy/o-stackyard/service_control_policy/" + policyID,
					"Name": "stackyard-policy",
					"Type": "SERVICE_CONTROL_POLICY",
				},
				"Content": "{\"Version\":\"2012-10-17\",\"Statement\":[]}",
			},
		}
	case "DescribeResourcePolicy":
		return map[string]any{
			"ResourcePolicy": map[string]any{
				"Content": "{\"Version\":\"2012-10-17\",\"Statement\":[]}",
			},
		}
	case "PutResourcePolicy":
		return map[string]any{
			"ResourcePolicy": map[string]any{
				"Content": organizationsPayloadString(payload, "Content", "{\"Version\":\"2012-10-17\",\"Statement\":[]}"),
			},
		}
	case "DescribeCreateAccountStatus":
		return map[string]any{
			"CreateAccountStatus": map[string]any{
				"Id":                 organizationsPayloadString(payload, "CreateAccountRequestId", "car-"+s.nextTokenLocked(8)),
				"State":              "SUCCEEDED",
				"AccountId":          "210987654321",
				"AccountName":        "stackyard-created-account",
				"RequestedTimestamp": 1.0,
				"CompletedTimestamp": 2.0,
			},
		}
	case "DescribeEffectivePolicy":
		return map[string]any{
			"EffectivePolicy": map[string]any{
				"PolicyType":           organizationsPayloadString(payload, "PolicyType", "SERVICE_CONTROL_POLICY"),
				"TargetId":             organizationsPayloadString(payload, "TargetId", "r-root"),
				"PolicyContent":        "{\"Version\":\"2012-10-17\",\"Statement\":[]}",
				"LastUpdatedTimestamp": 1.0,
			},
		}
	case "DescribeHandshake":
		return map[string]any{"Handshake": organizationsDefaultHandshake(s.nextTokenLocked(8))}
	case "DescribeResponsibilityTransfer":
		return map[string]any{
			"ResponsibilityTransfer": map[string]any{
				"Id":            organizationsPayloadString(payload, "ResponsibilityTransferId", "rt-"+s.nextTokenLocked(8)),
				"Status":        "ACTIVE",
				"TransferType":  "BILLING_RESPONSIBILITY",
				"InitiatedTime": 1.0,
			},
		}
	}

	if strings.HasPrefix(action, "Create") {
		resource := strings.TrimPrefix(action, "Create")
		switch resource {
		case "Account", "GovCloudAccount":
			return map[string]any{"CreateAccountStatus": map[string]any{"Id": "car-" + s.nextTokenLocked(8), "State": "IN_PROGRESS"}}
		case "OrganizationalUnit":
			ouID := "ou-" + s.nextTokenLocked(8)
			return map[string]any{"OrganizationalUnit": map[string]any{"Id": ouID, "Arn": "arn:aws:organizations::123456789012:ou/o-stackyard/" + ouID, "Name": organizationsPayloadString(payload, "Name", "stackyard-ou")}}
		case "Policy":
			pID := "p-" + s.nextTokenLocked(8)
			return map[string]any{"Policy": map[string]any{"PolicySummary": map[string]any{"Id": pID, "Arn": "arn:aws:organizations::123456789012:policy/o-stackyard/service_control_policy/" + pID, "Name": organizationsPayloadString(payload, "Name", "stackyard-policy"), "Type": organizationsPayloadString(payload, "Type", "SERVICE_CONTROL_POLICY")}, "Content": organizationsPayloadString(payload, "Content", "{\"Version\":\"2012-10-17\",\"Statement\":[]}")}}
		}
	}

	if strings.HasPrefix(action, "Invite") || strings.HasPrefix(action, "Accept") || strings.HasPrefix(action, "Decline") || strings.HasPrefix(action, "Cancel") {
		return map[string]any{"Handshake": organizationsDefaultHandshake(s.nextTokenLocked(8))}
	}

	if strings.HasPrefix(action, "List") {
		switch action {
		case "ListAccounts", "ListAccountsForParent", "ListAccountsWithInvalidEffectivePolicy":
			return map[string]any{"Accounts": []any{map[string]any{"Id": "123456789012", "Arn": "arn:aws:organizations::123456789012:account/o-stackyard/123456789012", "Name": "stackyard-management", "Email": "owner@example.com", "Status": "ACTIVE"}}, "NextToken": ""}
		case "ListRoots":
			return map[string]any{"Roots": []any{map[string]any{"Id": "r-root", "Arn": "arn:aws:organizations::123456789012:root/o-stackyard/r-root", "Name": "Root"}}, "NextToken": ""}
		case "ListOrganizationalUnitsForParent":
			return map[string]any{"OrganizationalUnits": []any{map[string]any{"Id": "ou-stackyard-001", "Arn": "arn:aws:organizations::123456789012:ou/o-stackyard/ou-stackyard-001", "Name": "Engineering"}}, "NextToken": ""}
		case "ListPolicies", "ListPoliciesForTarget":
			return map[string]any{"Policies": []any{map[string]any{"Id": "p-stackyard", "Arn": "arn:aws:organizations::123456789012:policy/o-stackyard/service_control_policy/p-stackyard", "Name": "stackyard-policy", "Type": "SERVICE_CONTROL_POLICY"}}, "NextToken": ""}
		case "ListTargetsForPolicy":
			return map[string]any{"Targets": []any{map[string]any{"TargetId": "ou-stackyard-001", "Name": "Engineering", "Type": "ORGANIZATIONAL_UNIT"}}, "NextToken": ""}
		case "ListChildren":
			return map[string]any{"Children": []any{map[string]any{"Id": "ou-stackyard-001", "Type": organizationsPayloadString(payload, "ChildType", "ORGANIZATIONAL_UNIT")}}, "NextToken": ""}
		case "ListParents":
			return map[string]any{"Parents": []any{map[string]any{"Id": "r-root", "Type": "ROOT"}}, "NextToken": ""}
		case "ListTagsForResource":
			return map[string]any{"Tags": []any{map[string]any{"Key": "stackyard", "Value": "true"}}}
		case "ListCreateAccountStatus":
			return map[string]any{"CreateAccountStatuses": []any{map[string]any{"Id": "car-00000001", "State": "SUCCEEDED", "AccountId": "210987654321"}}, "NextToken": ""}
		case "ListHandshakesForAccount", "ListHandshakesForOrganization":
			return map[string]any{"Handshakes": []any{organizationsDefaultHandshake("h-00000001")}, "NextToken": ""}
		case "ListDelegatedAdministrators":
			return map[string]any{"DelegatedAdministrators": []any{map[string]any{"Id": "123456789012", "Arn": "arn:aws:organizations::123456789012:account/o-stackyard/123456789012", "Email": "owner@example.com", "Name": "stackyard-admin", "Status": "ACTIVE", "DelegationEnabledDate": 1.0}}, "NextToken": ""}
		case "ListDelegatedServicesForAccount":
			return map[string]any{"DelegatedServices": []any{map[string]any{"ServicePrincipal": "config.amazonaws.com", "DelegationEnabledDate": 1.0}}, "NextToken": ""}
		case "ListAWSServiceAccessForOrganization":
			return map[string]any{"EnabledServicePrincipals": []any{map[string]any{"ServicePrincipal": "config.amazonaws.com", "DateEnabled": 1.0}}, "NextToken": ""}
		case "ListEffectivePolicyValidationErrors":
			return map[string]any{"EffectivePolicyValidationErrors": []any{}, "NextToken": ""}
		case "ListInboundResponsibilityTransfers", "ListOutboundResponsibilityTransfers":
			return map[string]any{"ResponsibilityTransfers": []any{}, "NextToken": ""}
		default:
			return map[string]any{"NextToken": ""}
		}
	}

	if strings.HasPrefix(action, "Delete") ||
		strings.HasPrefix(action, "Update") ||
		strings.HasPrefix(action, "Enable") ||
		strings.HasPrefix(action, "Disable") ||
		strings.HasPrefix(action, "Attach") ||
		strings.HasPrefix(action, "Detach") ||
		strings.HasPrefix(action, "Register") ||
		strings.HasPrefix(action, "Deregister") ||
		strings.HasPrefix(action, "Move") ||
		strings.HasPrefix(action, "Remove") ||
		strings.HasPrefix(action, "Tag") ||
		strings.HasPrefix(action, "Untag") ||
		strings.HasPrefix(action, "Terminate") ||
		action == "CloseAccount" ||
		action == "LeaveOrganization" {
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *organizationsStore) nextTokenLocked(width int) string {
	id := s.nextID
	s.nextID++
	format := fmt.Sprintf("%%0%dd", width)
	return fmt.Sprintf(format, id)
}

func organizationsPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" {
				return s
			}
			break
		}
	}
	return fallback
}

func organizationsDefaultHandshake(idSuffix string) map[string]any {
	id := "h-" + idSuffix
	if strings.HasPrefix(idSuffix, "h-") {
		id = idSuffix
	}
	return map[string]any{
		"Id":                  id,
		"Arn":                 "arn:aws:organizations::123456789012:handshake/o-stackyard/" + id,
		"State":               "OPEN",
		"Action":              "INVITE",
		"RequestedTimestamp":  1.0,
		"ExpirationTimestamp": 2.0,
		"Parties": []any{
			map[string]any{"Id": "123456789012", "Type": "ACCOUNT"},
		},
		"Resources": []any{},
	}
}
