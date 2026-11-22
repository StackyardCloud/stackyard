package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The models API hands out distinct, correctly sized IDs, which the runtime
// relies on to tell intents, slots and aliases apart.
func TestLexV2Stage1CreatedIDs(t *testing.T) {
	_, ts := newLexRTTestServer(t)
	create := func(path, body, field string) string {
		t.Helper()
		resp := lexRTRequest(t, ts, http.MethodPut, path, []byte(body), nil)
		assertStatus(t, resp, http.StatusOK)
		var out map[string]any
		if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		id, _ := out[field].(string)
		if len(id) != 10 {
			t.Fatalf("%s %s = %q, want 10 characters", path, field, id)
		}
		return id
	}
	botID := create("/bots/", `{"botName":"ids"}`, "botId")
	intents := botID + "/botversions/DRAFT/botlocales/en_US/intents/"
	a := create("/bots/"+intents, `{"intentName":"A"}`, "intentId")
	b := create("/bots/"+intents, `{"intentName":"B"}`, "intentId")
	if a == b {
		t.Fatalf("two intents share ID %s", a)
	}
	if s1, s2 := create("/bots/"+intents+a+"/slots/", `{"slotName":"x"}`, "slotId"), create("/bots/"+intents+a+"/slots/", `{"slotName":"y"}`, "slotId"); s1 == s2 {
		t.Fatalf("two slots share ID %s", s1)
	}
	if a1, a2 := create("/bots/"+botID+"/botaliases/", `{"botAliasName":"one"}`, "botAliasId"), create("/bots/"+botID+"/botaliases/", `{"botAliasName":"two"}`, "botAliasId"); a1 == a2 {
		t.Fatalf("two aliases share ID %s", a1)
	}

	resp := lexRTRequest(t, ts, http.MethodPut, "/bots/", []byte(`{`), nil)
	assertStatus(t, resp, http.StatusBadRequest)
	if code := lexRTErrorCode(t, resp); code != "ValidationException" {
		t.Fatalf("bad JSON code = %q", code)
	}
}
