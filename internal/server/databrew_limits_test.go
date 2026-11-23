package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDataBrewTableBudgets(t *testing.T) {
	// One JSON array can't carry more than the row cap.
	var arr []map[string]string
	for i := 0; i < dbwMaxRows+500; i++ {
		arr = append(arr, map[string]string{"a": "x"})
	}
	raw, _ := json.Marshal(arr)
	tbl := &dbwTable{}
	if err := tbl.readJSON(raw); err != nil {
		t.Fatal(err)
	}
	if len(tbl.rows) > dbwMaxRows {
		t.Fatalf("JSON array loaded %d rows, cap %d", len(tbl.rows), dbwMaxRows)
	}

	// Repeated ADD_PREFIX can't grow the table past the byte budget.
	tbl = &dbwTable{cols: []string{"a"}}
	for i := 0; i < 1000; i++ {
		if err := tbl.addRow([]string{"x"}); err != nil {
			t.Fatal(err)
		}
	}
	big := strings.Repeat("p", 1<<20)
	var err error
	for i := 0; i < 200 && err == nil; i++ {
		err = tbl.applyOne("ADD_PREFIX", map[string]string{"sourceColumn": "a", "pattern": big})
	}
	if err == nil || tbl.size > dbwMaxTableBytes+(1<<20) {
		t.Fatalf("ADD_PREFIX grew the table to %d bytes without an error", tbl.size)
	}

	// DUPLICATE can't push rows x columns past the cell cap.
	tbl = &dbwTable{cols: []string{"a"}}
	for i := 0; i < dbwMaxCells/2; i++ {
		tbl.rows = append(tbl.rows, []string{""})
	}
	if err := tbl.applyOne("DUPLICATE", map[string]string{"sourceColumn": "a", "targetColumn": "b"}); err != nil {
		t.Fatalf("first duplicate: %v", err)
	}
	if err := tbl.applyOne("DUPLICATE", map[string]string{"sourceColumn": "a", "targetColumn": "c"}); err == nil {
		t.Fatal("DUPLICATE went past the cell cap")
	}

	// A recipe can't apply unbounded steps.
	steps := make([]any, dbwMaxSteps+1)
	for i := range steps {
		steps[i] = map[string]any{"Action": map[string]any{"Operation": "UPPER_CASE", "Parameters": map[string]any{"sourceColumn": "a"}}}
	}
	if err := (&dbwTable{cols: []string{"a"}}).apply(steps); err == nil {
		t.Fatal("more than the step cap was applied")
	}
}
