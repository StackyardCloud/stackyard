package server

import "testing"

func TestSecurityLakeStage0CatalogCoverage(t *testing.T) {
	if len(securityLakeOperations) != 31 {
		t.Fatalf("expected 31 Security Lake operations from docs, got %d", len(securityLakeOperations))
	}
	if len(securityLakeOperationByName) != len(securityLakeOperations) {
		t.Fatalf("expected unique Security Lake operation names")
	}

	requiredActions := []string{
		"CreateDataLake",
		"ListDataLakes",
		"GetDataLakeSources",
		"CreateSubscriber",
		"ListSubscribers",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := securityLakeOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(securityLakeTypes) != 29 {
		t.Fatalf("expected 29 Security Lake data types from docs, got %d", len(securityLakeTypes))
	}
	if len(securityLakeTypeByName) != len(securityLakeTypes) {
		t.Fatalf("expected unique Security Lake data type names")
	}

	requiredTypes := []string{
		"DataLakeConfiguration",
		"DataLakeResource",
		"DataLakeSource",
		"LogSource",
		"SubscriberResource",
	}
	for _, typeName := range requiredTypes {
		if _, ok := securityLakeTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}
