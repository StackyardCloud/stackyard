package server

import "testing"

func TestShieldAdvancedStage0CatalogCoverage(t *testing.T) {
	if len(shieldAdvancedOperations) != 36 {
		t.Fatalf("expected 36 Shield Advanced operations from docs, got %d", len(shieldAdvancedOperations))
	}
	if len(shieldAdvancedOperationByName) != len(shieldAdvancedOperations) {
		t.Fatalf("expected unique Shield Advanced operation names")
	}

	requiredActions := []string{
		"CreateProtection",
		"DescribeProtection",
		"ListProtections",
		"CreateProtectionGroup",
		"ListAttacks",
		"TagResource",
		"UntagResource",
		"UpdateSubscription",
	}
	for _, action := range requiredActions {
		if _, ok := shieldAdvancedOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(shieldAdvancedTypes) != 31 {
		t.Fatalf("expected 31 Shield Advanced data types from docs, got %d", len(shieldAdvancedTypes))
	}
	if len(shieldAdvancedTypeByName) != len(shieldAdvancedTypes) {
		t.Fatalf("expected unique Shield Advanced data type names")
	}

	requiredTypes := []string{
		"Protection",
		"ProtectionGroup",
		"AttackDetail",
		"AttackSummary",
		"Subscription",
	}
	for _, typeName := range requiredTypes {
		if _, ok := shieldAdvancedTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}
