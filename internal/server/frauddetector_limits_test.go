package server

import (
	"strings"
	"testing"
)

// Updates must re-charge a record's size: otherwise repeated updates with
// large values grow the store past frdMaxBytes.
func TestFRDUpdateRechargesStoredBytes(t *testing.T) {
	st := newFraudDetectorStore()
	if _, err := st.createVariable(map[string]any{"name": "v", "dataType": "STRING", "dataSource": "EVENT", "defaultValue": "x"}); err != nil {
		t.Fatalf("create: %v", err.msg)
	}
	before := st.bytes
	big := strings.Repeat("d", 4096)
	if _, err := st.updateVariable(map[string]any{"name": "v", "description": big}); err != nil {
		t.Fatalf("update: %v", err.msg)
	}
	if st.bytes < before+len(big) {
		t.Fatalf("update not charged: bytes %d -> %d", before, st.bytes)
	}
	st.bytes = frdMaxBytes // store full
	if _, err := st.updateVariable(map[string]any{"name": "v", "description": big + big}); err == nil {
		t.Fatal("update that grows a full store was accepted")
	}
}
