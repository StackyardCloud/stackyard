package server

import "testing"

func TestNetworkFirewallStage0CatalogCoverage(t *testing.T) {
	if len(networkFirewallOperations) != 36 {
		t.Fatalf("expected 36 Network Firewall operations from docs, got %d", len(networkFirewallOperations))
	}
	if len(networkFirewallOperationByName) != len(networkFirewallOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateFirewall",
		"DescribeFirewall",
		"ListFirewalls",
		"CreateFirewallPolicy",
		"CreateRuleGroup",
		"UpdateLoggingConfiguration",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := networkFirewallOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(networkFirewallDataTypes) != 55 {
		t.Fatalf("expected 55 Network Firewall data types from docs, got %d", len(networkFirewallDataTypes))
	}
	if len(networkFirewallDataTypeByName) != len(networkFirewallDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Firewall",
		"FirewallPolicy",
		"RuleGroup",
		"TLSInspectionConfiguration",
		"LoggingConfiguration",
		"SubnetMapping",
	}
	for _, typeName := range requiredTypes {
		if _, ok := networkFirewallDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}
