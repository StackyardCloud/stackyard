package server

import "testing"

func TestEBSStage0CatalogCoverage(t *testing.T) {
	if len(ebsOperations) != 6 {
		t.Fatalf("expected 6 EBS operations from docs, got %d", len(ebsOperations))
	}
	if len(ebsOperationByName) != len(ebsOperations) {
		t.Fatalf("expected unique EBS operation names")
	}

	requiredActions := []string{
		"StartSnapshot",
		"PutSnapshotBlock",
		"ListSnapshotBlocks",
		"GetSnapshotBlock",
		"ListChangedBlocks",
		"CompleteSnapshot",
	}
	for _, action := range requiredActions {
		if _, ok := ebsOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(ebsDataTypes) != 23 {
		t.Fatalf("expected 23 EBS data types from docs, got %d", len(ebsDataTypes))
	}
	if len(ebsDataTypeByName) != len(ebsDataTypes) {
		t.Fatalf("expected unique EBS data type names")
	}

	requiredTypes := []string{
		"Block",
		"ChangedBlock",
		"StartSnapshotRequest",
		"PutSnapshotBlockRequest",
		"ListSnapshotBlocksResponse",
		"CompleteSnapshotResponse",
	}
	for _, typeName := range requiredTypes {
		if _, ok := ebsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}
