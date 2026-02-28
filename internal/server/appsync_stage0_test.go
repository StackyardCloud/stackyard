package server

import "testing"

func TestAppSyncStage0CatalogCoverage(t *testing.T) {
	if len(appSyncOperations) != 74 {
		t.Fatalf("expected 74 AppSync actions from docs, got %d", len(appSyncOperations))
	}
	if len(appSyncOperationByName) != len(appSyncOperations) {
		t.Fatalf("expected unique AppSync action names")
	}

	requiredActions := []string{
		"CreateGraphqlApi",
		"CreateResolver",
		"StartSchemaCreation",
		"ListGraphqlApis",
		"GetGraphqlApi",
		"UpdateGraphqlApi",
		"DeleteGraphqlApi",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := appSyncOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(appSyncDataTypes) != 57 {
		t.Fatalf("expected 57 AppSync data types from docs, got %d", len(appSyncDataTypes))
	}
	if len(appSyncDataTypeByName) != len(appSyncDataTypes) {
		t.Fatalf("expected unique AppSync data type names")
	}

	requiredTypes := []string{
		"GraphqlApi",
		"Resolver",
		"DataSource",
		"ApiKey",
		"Type",
		"AuthMode",
	}
	for _, typeName := range requiredTypes {
		if _, ok := appSyncDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}
