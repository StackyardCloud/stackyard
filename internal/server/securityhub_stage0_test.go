package server

import "testing"

func TestSecurityHubStage0CatalogCoverage(t *testing.T) {
	if len(securityHubOperations) != 107 {
		t.Fatalf("expected 107 Security Hub operations from docs, got %d", len(securityHubOperations))
	}
	if len(securityHubOperationByName) != len(securityHubOperations) {
		t.Fatalf("expected unique Security Hub operation names")
	}

	requiredActions := []string{
		"DescribeHub",
		"EnableSecurityHub",
		"DisableSecurityHub",
		"GetFindings",
		"BatchImportFindings",
		"ListEnabledProductsForImport",
		"ListMembers",
		"ListStandardsControlAssociations",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := securityHubOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(securityHubTypes) != 725 {
		t.Fatalf("expected 725 Security Hub data types from docs, got %d", len(securityHubTypes))
	}
	if len(securityHubTypeByName) != len(securityHubTypes) {
		t.Fatalf("expected unique Security Hub data type names")
	}

	requiredTypes := []string{
		"AutomationRulesConfig",
		"AwsEc2InstanceDetails",
		"AccountDetails",
		"ActionTarget",
		"StandardsSubscription",
		"StringFilter",
	}
	for _, typeName := range requiredTypes {
		if _, ok := securityHubTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}
