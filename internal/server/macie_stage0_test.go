package server

import "testing"

func TestMacieStage0CatalogCoverage(t *testing.T) {
	if len(macieOperations) != 81 {
		t.Fatalf("expected 81 Macie operations from docs, got %d", len(macieOperations))
	}
	if len(macieOperationByName) != len(macieOperations) {
		t.Fatalf("expected unique Macie operation names")
	}

	requiredActions := []string{
		"EnableMacie",
		"DisableMacie",
		"GetMacieSession",
		"CreateClassificationJob",
		"ListClassificationJobs",
		"GetFindings",
		"ListFindings",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := macieOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(macieResources) != 53 {
		t.Fatalf("expected 53 Macie resources from docs, got %d", len(macieResources))
	}
	if len(macieResourceByName) != len(macieResources) {
		t.Fatalf("expected unique Macie resource names")
	}

	requiredResources := []string{
		"Account Administration",
		"Classification Job",
		"Findings",
		"Findings Filters",
		"Members",
		"Tags",
	}
	for _, name := range requiredResources {
		if _, ok := macieResourceByName[name]; !ok {
			t.Fatalf("missing documented resource %s", name)
		}
	}
}
