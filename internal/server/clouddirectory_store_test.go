package server

import (
	"testing"
)

func TestCloudDirectoryStoreReturnsModeledResponseShapes(t *testing.T) {
	store := newCloudDirectoryStore()

	createIndex := store.Handle(
		"CreateIndex",
		map[string]any{"DirectoryArn": cloudDirectoryDirectoryARN("d-00000001")},
		nil,
		nil,
	)
	if _, ok := createIndex["ObjectIdentifier"].(string); !ok {
		t.Fatalf("expected CreateIndex to return ObjectIdentifier, got %#v", createIndex)
	}

	typedLinkSpecifier := map[string]any{
		"TypedLinkFacet": map[string]any{
			"SchemaArn": cloudDirectorySchemaARN("s-00000001"),
			"Name":      "stackyard-typed-link",
		},
		"SourceObjectReference": map[string]any{"Selector": "root"},
		"TargetObjectReference": map[string]any{"Selector": "root"},
		"IdentityAttributeValues": []any{
			map[string]any{"AttributeName": "id", "Value": map[string]any{"StringValue": "1"}},
		},
	}
	attachTypedLink := store.Handle(
		"AttachTypedLink",
		map[string]any{"TypedLinkSpecifier": typedLinkSpecifier},
		nil,
		nil,
	)
	gotSpecifier, ok := attachTypedLink["TypedLinkSpecifier"].(map[string]any)
	if !ok {
		t.Fatalf("expected AttachTypedLink to return TypedLinkSpecifier, got %#v", attachTypedLink)
	}
	if _, ok := gotSpecifier["TypedLinkFacet"].(map[string]any); !ok {
		t.Fatalf("expected AttachTypedLink to preserve TypedLinkFacet, got %#v", gotSpecifier)
	}

	listOutgoing := store.Handle("ListOutgoingTypedLinks", map[string]any{}, nil, nil)
	if _, ok := listOutgoing["TypedLinkSpecifiers"].([]any); !ok {
		t.Fatalf("expected ListOutgoingTypedLinks to return TypedLinkSpecifiers, got %#v", listOutgoing)
	}

	applySchema := store.Handle(
		"ApplySchema",
		map[string]any{
			"DirectoryArn": cloudDirectoryDirectoryARN("d-00000001"),
			"SchemaArn":    cloudDirectorySchemaARN("s-00000001"),
		},
		nil,
		nil,
	)
	if applySchema["AppliedSchemaArn"] != cloudDirectorySchemaARN("s-00000001") {
		t.Fatalf("expected ApplySchema to return AppliedSchemaArn, got %#v", applySchema)
	}
	if applySchema["DirectoryArn"] != cloudDirectoryDirectoryARN("d-00000001") {
		t.Fatalf("expected ApplySchema to return DirectoryArn, got %#v", applySchema)
	}
}
