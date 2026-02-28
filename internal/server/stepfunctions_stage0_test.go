package server

import "testing"

func TestStepFunctionsStage0CatalogCoverage(t *testing.T) {
	if len(stepFunctionsOperations) != 37 {
		t.Fatalf("expected 37 Step Functions actions from docs, got %d", len(stepFunctionsOperations))
	}
	if len(stepFunctionsOperationByName) != len(stepFunctionsOperations) {
		t.Fatalf("expected unique Step Functions action names")
	}

	requiredActions := []string{
		"CreateStateMachine",
		"StartExecution",
		"StartSyncExecution",
		"DescribeExecution",
		"GetExecutionHistory",
		"SendTaskSuccess",
		"TestState",
		"ValidateStateMachineDefinition",
	}
	for _, action := range requiredActions {
		if _, ok := stepFunctionsOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(stepFunctionsDataTypes) != 63 {
		t.Fatalf("expected 63 Step Functions data types from docs, got %d", len(stepFunctionsDataTypes))
	}
	if len(stepFunctionsDataTypeByName) != len(stepFunctionsDataTypes) {
		t.Fatalf("expected unique Step Functions data type names")
	}

	requiredTypes := []string{
		"StateMachineListItem",
		"ExecutionListItem",
		"HistoryEvent",
		"TaskScheduledEventDetails",
		"MapRunListItem",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := stepFunctionsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}
