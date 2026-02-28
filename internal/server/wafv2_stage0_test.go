package server

import "testing"

func TestWAFV2Stage0CatalogCoverage(t *testing.T) {
	if len(wafv2Operations) != 55 {
		t.Fatalf("expected 55 WAFV2 operations from docs, got %d", len(wafv2Operations))
	}
	if len(wafv2OperationByName) != len(wafv2Operations) {
		t.Fatalf("expected unique WAFV2 operation names")
	}

	requiredActions := []string{
		"CreateWebACL",
		"GetWebACL",
		"ListWebACLs",
		"UpdateWebACL",
		"DeleteWebACL",
		"AssociateWebACL",
		"DisassociateWebACL",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := wafv2OperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(wafv2Types) != 142 {
		t.Fatalf("expected 142 WAFV2 data types from docs, got %d", len(wafv2Types))
	}
	if len(wafv2TypeByName) != len(wafv2Types) {
		t.Fatalf("expected unique WAFV2 data type names")
	}

	requiredTypes := []string{
		"WebACL",
		"Rule",
		"Statement",
		"DefaultAction",
		"VisibilityConfig",
		"IPSet",
	}
	for _, typeName := range requiredTypes {
		if _, ok := wafv2TypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}
