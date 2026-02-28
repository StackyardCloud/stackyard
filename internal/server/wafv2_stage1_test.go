package server

import (
	"net/http/httptest"
	"testing"
)

func TestWAFV2Stage1StoreHandlesAllCatalogActions(t *testing.T) {
	store := newWAFV2Store()
	for _, op := range wafv2Operations {
		resp := store.Handle(op.Name, map[string]any{})
		if resp == nil {
			t.Fatalf("expected non-nil response for action %s", op.Name)
		}
	}
}

func TestWAFV2Stage1TargetParsingAndCandidate(t *testing.T) {
	req := httptest.NewRequest("POST", "http://localhost:4566/", nil)
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSWAF_20190729.ListWebACLs")
	if !isWAFV2JSONCandidate(req) {
		t.Fatalf("expected WAFV2 candidate detection for JSON-RPC request")
	}
	if got := parseWAFV2Target(req.Header.Get("X-Amz-Target")); got != "ListWebACLs" {
		t.Fatalf("expected parsed action ListWebACLs, got %q", got)
	}
}
