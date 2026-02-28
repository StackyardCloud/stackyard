package server

import "testing"

func TestDRSStage0CatalogCoverage(t *testing.T) {
	if len(drsOperations) != 50 {
		t.Fatalf("expected 50 DRS operations from docs, got %d", len(drsOperations))
	}
	if len(drsOperationByName) != len(drsOperations) {
		t.Fatalf("expected unique DRS operation names")
	}

	requiredActions := []string{
		"CreateSourceNetwork",
		"DescribeSourceServers",
		"StartRecovery",
		"StartReplication",
		"StopReplication",
		"UpdateLaunchConfiguration",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := drsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(drsDataTypes) != 62 {
		t.Fatalf("expected 62 DRS data types from docs, got %d", len(drsDataTypes))
	}
	if len(drsDataTypeByName) != len(drsDataTypes) {
		t.Fatalf("expected unique DRS data type names")
	}

	requiredTypes := []string{
		"SourceServer",
		"RecoveryInstance",
		"LaunchConfiguration",
		"ReplicationConfigurationTemplate",
		"DataReplicationInfo",
		"ValidationExceptionField",
	}
	for _, typeName := range requiredTypes {
		if _, ok := drsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}
