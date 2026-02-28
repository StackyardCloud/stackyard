package server

import "testing"

func TestInspectorV2Stage0CatalogCoverage(t *testing.T) {
	if len(inspectorV2Operations) != 76 {
		t.Fatalf("expected 76 Inspector V2 operations from docs, got %d", len(inspectorV2Operations))
	}
	if len(inspectorV2OperationByName) != len(inspectorV2Operations) {
		t.Fatalf("expected unique Inspector V2 operation names")
	}

	requiredActions := []string{
		"ListFindings",
		"ListCoverage",
		"ListMembers",
		"CreateFilter",
		"TagResource",
		"ScanSbom",
	}
	for _, action := range requiredActions {
		if _, ok := inspectorV2OperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(inspectorV2DataTypes) != 183 {
		t.Fatalf("expected 183 Inspector V2 data types from docs, got %d", len(inspectorV2DataTypes))
	}
	if len(inspectorV2DataTypeByName) != len(inspectorV2DataTypes) {
		t.Fatalf("expected unique Inspector V2 data type names")
	}

	requiredTypes := []string{
		"Finding",
		"FilterCriteria",
		"CoverageFilterCriteria",
		"ResourceFilterCriteria",
		"ValidationExceptionField",
		"Vulnerability",
	}
	for _, name := range requiredTypes {
		if _, ok := inspectorV2DataTypeByName[name]; !ok {
			t.Fatalf("missing documented data type %s", name)
		}
	}
}
