package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type lexRTTextResponse struct {
	Messages []struct {
		Content     string `json:"content"`
		ContentType string `json:"contentType"`
	} `json:"messages"`
	SessionState struct {
		DialogAction struct {
			Type         string `json:"type"`
			SlotToElicit string `json:"slotToElicit"`
		} `json:"dialogAction"`
		Intent struct {
			Name  string `json:"name"`
			State string `json:"state"`
			Slots map[string]*struct {
				Value struct {
					InterpretedValue string `json:"interpretedValue"`
				} `json:"value"`
			} `json:"slots"`
		} `json:"intent"`
		SessionAttributes map[string]string `json:"sessionAttributes"`
	} `json:"sessionState"`
	Interpretations []struct {
		NluConfidence *struct {
			Score float64 `json:"score"`
		} `json:"nluConfidence"`
		Intent struct {
			Name string `json:"name"`
		} `json:"intent"`
	} `json:"interpretations"`
	RequestAttributes map[string]string `json:"requestAttributes"`
	SessionID         string            `json:"sessionId"`
}

// lexRTHotelBot builds, through the models API, a bot with a BookHotel intent
// that needs two slots, and returns the bot ID and a created alias ID.
func lexRTHotelBot(t *testing.T, ts *httptest.Server) (string, string) {
	t.Helper()
	decode := func(resp *http.Response, field string) string {
		t.Helper()
		assertStatus(t, resp, http.StatusOK)
		var out map[string]any
		if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
			t.Fatalf("decode %s: %v", field, err)
		}
		v, _ := out[field].(string)
		if v == "" {
			t.Fatalf("missing %s in %v", field, out)
		}
		return v
	}
	botID := decode(lexRTRequest(t, ts, http.MethodPut, "/bots/", []byte(`{"botName":"hotel","idleSessionTTLInSeconds":60,"roleArn":"arn:aws:iam::123456789012:role/r","dataPrivacy":{"childDirected":false}}`), nil), "botId")
	decode(lexRTRequest(t, ts, http.MethodPut, "/bots/"+botID+"/botversions/DRAFT/botlocales/", []byte(`{"localeId":"en_US","nluIntentConfidenceThreshold":0.4}`), nil), "localeId")
	intentID := decode(lexRTRequest(t, ts, http.MethodPut, "/bots/"+botID+"/botversions/DRAFT/botlocales/en_US/intents/", []byte(`{"intentName":"BookHotel","sampleUtterances":[{"utterance":"book a hotel"},{"utterance":"book a {RoomType} room"}]}`), nil), "intentId")
	decode(lexRTRequest(t, ts, http.MethodPut, "/bots/"+botID+"/botversions/DRAFT/botlocales/en_US/intents/", []byte(`{"intentName":"CheckWeather","sampleUtterances":[{"utterance":"what is the weather"}]}`), nil), "intentId")
	for _, slot := range []struct{ name, prompt string }{{"RoomType", "Which room type?"}, {"Nights", "How many nights?"}} {
		body := `{"slotName":"` + slot.name + `","slotTypeId":"AMAZON.AlphaNumeric","valueElicitationSetting":{"slotConstraint":"Required","promptSpecification":{"maxRetries":1,"messageGroups":[{"message":{"plainTextMessage":{"value":"` + slot.prompt + `"}}}]}}}`
		decode(lexRTRequest(t, ts, http.MethodPut, "/bots/"+botID+"/botversions/DRAFT/botlocales/en_US/intents/"+intentID+"/slots/", []byte(body), nil), "slotId")
	}
	aliasID := decode(lexRTRequest(t, ts, http.MethodPut, "/bots/"+botID+"/botaliases/", []byte(`{"botAliasName":"live","botVersion":"DRAFT"}`), nil), "botAliasId")
	return botID, aliasID
}

func lexRTSay(t *testing.T, ts *httptest.Server, path, text string) lexRTTextResponse {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"text": text, "requestAttributes": map[string]string{"channel": "test"}})
	resp := lexRTRequest(t, ts, http.MethodPost, path+"/text", body, nil)
	assertStatus(t, resp, http.StatusOK)
	var out lexRTTextResponse
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("decode RecognizeText: %v", err)
	}
	return out
}

func TestLexRuntimeV2Stage1SlotElicitation(t *testing.T) {
	_, ts := newLexRTTestServer(t)
	botID, aliasID := lexRTHotelBot(t, ts)
	if len(botID) != 10 {
		t.Fatalf("CreateBot returned %q; bot IDs are 10 characters", botID)
	}
	session := "/bots/" + botID + "/botAliases/" + aliasID + "/botLocales/en_US/sessions/user-1"

	out := lexRTSay(t, ts, session, "Book a hotel!")
	if out.SessionState.Intent.Name != "BookHotel" || out.SessionState.Intent.State != "InProgress" ||
		out.SessionState.DialogAction.Type != "ElicitSlot" || out.SessionState.DialogAction.SlotToElicit != "RoomType" {
		t.Fatalf("turn 1 state = %+v", out.SessionState)
	}
	if len(out.Messages) != 1 || out.Messages[0].Content != "Which room type?" || out.Messages[0].ContentType != "PlainText" {
		t.Fatalf("turn 1 messages = %+v", out.Messages)
	}
	if out.Interpretations[0].Intent.Name != "BookHotel" || out.Interpretations[0].NluConfidence.Score != 1 {
		t.Fatalf("turn 1 interpretations = %+v", out.Interpretations)
	}
	if last := out.Interpretations[len(out.Interpretations)-1]; last.Intent.Name != "FallbackIntent" || last.NluConfidence != nil {
		t.Fatalf("FallbackIntent should close the interpretations, got %+v", last)
	}
	if out.SessionID != "user-1" || out.RequestAttributes["channel"] != "test" {
		t.Fatalf("sessionId/requestAttributes = %q %v", out.SessionID, out.RequestAttributes)
	}

	out = lexRTSay(t, ts, session, "king")
	if out.SessionState.DialogAction.SlotToElicit != "Nights" || out.SessionState.Intent.Slots["RoomType"].Value.InterpretedValue != "king" {
		t.Fatalf("turn 2 state = %+v", out.SessionState)
	}
	if out.SessionState.Intent.Slots["Nights"] != nil {
		t.Fatalf("Nights should still be null")
	}

	out = lexRTSay(t, ts, session, "3")
	if out.SessionState.DialogAction.Type != "Close" || out.SessionState.Intent.State != "ReadyForFulfillment" ||
		out.SessionState.Intent.Slots["Nights"].Value.InterpretedValue != "3" || len(out.Messages) != 0 {
		t.Fatalf("turn 3 = %+v", out)
	}

	// A sample utterance with a slot reference fills that slot directly.
	out = lexRTSay(t, ts, session, "book a suite room")
	if out.SessionState.Intent.Slots["RoomType"].Value.InterpretedValue != "suite" || out.SessionState.DialogAction.SlotToElicit != "Nights" {
		t.Fatalf("slot from utterance = %+v", out.SessionState)
	}

	// The other intent wins on its own words; nonsense falls back.
	if out = lexRTSay(t, ts, session, "what is the weather"); out.SessionState.Intent.Name != "CheckWeather" {
		t.Fatalf("intent switch = %+v", out.SessionState.Intent)
	}
	out = lexRTSay(t, ts, session, "purple elephants")
	if out.SessionState.Intent.Name != "FallbackIntent" || out.SessionState.DialogAction.Type != "Close" || out.Interpretations[0].Intent.Name != "FallbackIntent" {
		t.Fatalf("fallback = %+v", out)
	}

	// The built-in test alias resolves too; a deleted alias does not.
	lexRTSay(t, ts, "/bots/"+botID+"/botAliases/TSTALIASID/botLocales/en_US/sessions/user-2", "book a hotel")
	assertStatus(t, lexRTRequest(t, ts, http.MethodDelete, "/bots/"+botID+"/botaliases/"+aliasID+"/", nil, nil), http.StatusOK)
	resp := lexRTRequest(t, ts, http.MethodPost, session+"/text", []byte(`{"text":"hi"}`), nil)
	assertStatus(t, resp, http.StatusNotFound)
	if code := lexRTErrorCode(t, resp); code != "ResourceNotFoundException" {
		t.Fatalf("deleted alias code = %q", code)
	}
}

func TestLexRuntimeV2Stage1SessionLifecycle(t *testing.T) {
	srv, ts := newLexRTTestServer(t)
	botID, aliasID := lexRTHotelBot(t, ts)
	session := "/bots/" + botID + "/botAliases/" + aliasID + "/botLocales/en_US/sessions/user-9"

	put := `{"sessionState":{"sessionAttributes":{"tier":"gold"},"dialogAction":{"type":"ElicitSlot","slotToElicit":"Nights"},"intent":{"name":"BookHotel","state":"InProgress","slots":{"RoomType":{"value":{"interpretedValue":"queen"}}}}},"messages":[{"contentType":"PlainText","content":"How many nights?"}]}`
	resp := lexRTRequest(t, ts, http.MethodPost, session, []byte(put), map[string]string{"ResponseContentType": "text/plain; charset=utf-8"})
	assertStatus(t, resp, http.StatusOK)
	var state lexRTSessionState
	if err := decodeLexRTHeader(resp.Header.Get("x-amz-lex-session-state"), &state); err != nil || state.SessionAttributes["tier"] != "gold" {
		t.Fatalf("PutSession state header: %v %+v", err, state)
	}
	if body := string(mustBody(t, resp)); body != "How many nights?" || resp.Header.Get("x-amz-lex-session-id") != "user-9" {
		t.Fatalf("PutSession body/session = %q %q", body, resp.Header.Get("x-amz-lex-session-id"))
	}

	// The stored state drives the next turn, and attributes carry over.
	out := lexRTSay(t, ts, session, "2")
	if out.SessionState.Intent.State != "ReadyForFulfillment" || out.SessionState.Intent.Slots["RoomType"].Value.InterpretedValue != "queen" ||
		out.SessionState.SessionAttributes["tier"] != "gold" {
		t.Fatalf("turn after PutSession = %+v", out.SessionState)
	}

	resp = lexRTRequest(t, ts, http.MethodGet, session, nil, nil)
	assertStatus(t, resp, http.StatusOK)
	var got struct {
		SessionID    string            `json:"sessionId"`
		SessionState lexRTSessionState `json:"sessionState"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &got); err != nil || got.SessionID != "user-9" || got.SessionState.Intent.Name != "BookHotel" {
		t.Fatalf("GetSession = %v %+v", err, got)
	}

	assertStatus(t, lexRTRequest(t, ts, http.MethodDelete, session, nil, nil), http.StatusOK)
	assertStatus(t, lexRTRequest(t, ts, http.MethodGet, session, nil, nil), http.StatusNotFound)

	// Sessions expire after the bot's idleSessionTTLInSeconds (60s here).
	lexRTSay(t, ts, session, "book a hotel")
	srv.lexv2.mu.Lock()
	for _, sess := range srv.lexv2.sessions {
		sess.LastActive = time.Now().Add(-61 * time.Second)
	}
	srv.lexv2.mu.Unlock()
	assertStatus(t, lexRTRequest(t, ts, http.MethodGet, session, nil, nil), http.StatusNotFound)

	// Deleting the bot removes its runtime model.
	assertStatus(t, lexRTRequest(t, ts, http.MethodDelete, "/bots/"+botID+"/", nil, nil), http.StatusOK)
	assertStatus(t, lexRTRequest(t, ts, http.MethodPost, session+"/text", []byte(`{"text":"hi"}`), nil), http.StatusNotFound)
}

func TestLexRuntimeV2Stage1MatchUtterance(t *testing.T) {
	cases := []struct {
		utterance, text string
		score           float64
		slots           map[string]string
	}{
		{"book a hotel", "Book a HOTEL.", 1, map[string]string{}},
		{"book a {RoomType} room", "book a deluxe king room", 1, map[string]string{"RoomType": "deluxe king"}},
		{"book a hotel", "please book a hotel now", 0.9, nil},
		{"book a hotel", "hotel", 0.3, nil},
		{"book a hotel", "", 0, nil},
	}
	for _, tc := range cases {
		score, slots := matchLexRTUtterance(tc.utterance, tc.text)
		if score != tc.score || len(slots) != len(tc.slots) {
			t.Fatalf("%q vs %q = %v %v", tc.utterance, tc.text, score, slots)
		}
		for k, v := range tc.slots {
			if slots[k] != v {
				t.Fatalf("%q vs %q slot %s = %q", tc.utterance, tc.text, k, slots[k])
			}
		}
	}
}
