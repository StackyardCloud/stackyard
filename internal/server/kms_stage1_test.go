package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func kmsCreateKeyForTest(t *testing.T, ts *httptest.Server, payload map[string]any) (string, string) {
	t.Helper()

	resp := kmsRequest(t, ts, "CreateKey", mustJSON(t, payload))
	assertStatus(t, resp, http.StatusOK)

	var out struct {
		KeyMetadata struct {
			KeyID string `json:"KeyId"`
			Arn   string `json:"Arn"`
		} `json:"KeyMetadata"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal create key response: %v", err)
	}
	if out.KeyMetadata.KeyID == "" || out.KeyMetadata.Arn == "" {
		t.Fatalf("expected key id and arn in create key response")
	}
	return out.KeyMetadata.KeyID, out.KeyMetadata.Arn
}

func TestKMSStage1Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	keyID, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "stage1-primary",
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"KeySpec":     "SYMMETRIC_DEFAULT",
	})
	keyID2, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "stage1-secondary",
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"KeySpec":     "SYMMETRIC_DEFAULT",
	})

	describe := func(id string) struct {
		KeyMetadata struct {
			KeyID       string `json:"KeyId"`
			Arn         string `json:"Arn"`
			Description string `json:"Description"`
			Enabled     bool   `json:"Enabled"`
			KeyState    string `json:"KeyState"`
		} `json:"KeyMetadata"`
	} {
		resp := kmsRequest(t, ts, "DescribeKey", mustJSON(t, map[string]any{"KeyId": id}))
		assertStatus(t, resp, http.StatusOK)
		var out struct {
			KeyMetadata struct {
				KeyID       string `json:"KeyId"`
				Arn         string `json:"Arn"`
				Description string `json:"Description"`
				Enabled     bool   `json:"Enabled"`
				KeyState    string `json:"KeyState"`
			} `json:"KeyMetadata"`
		}
		if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
			t.Fatalf("unmarshal describe key response: %v", err)
		}
		return out
	}

	describeOut := describe(keyID)
	if describeOut.KeyMetadata.KeyID != keyID {
		t.Fatalf("expected key id %q, got %q", keyID, describeOut.KeyMetadata.KeyID)
	}
	if describeOut.KeyMetadata.KeyState != "Enabled" || !describeOut.KeyMetadata.Enabled {
		t.Fatalf("expected key to be enabled, got state=%q enabled=%v", describeOut.KeyMetadata.KeyState, describeOut.KeyMetadata.Enabled)
	}

	resp := kmsRequest(t, ts, "ListKeys", []byte(`{"Limit":10}`))
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		Keys []struct {
			KeyID string `json:"KeyId"`
		} `json:"Keys"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal list keys response: %v", err)
	}
	if len(listOut.Keys) < 2 {
		t.Fatalf("expected at least two keys, got %d", len(listOut.Keys))
	}

	aliasName := "alias/stackyard-stage1"
	resp = kmsRequest(t, ts, "CreateAlias", mustJSON(t, map[string]any{
		"AliasName":   aliasName,
		"TargetKeyId": keyID,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "ListAliases", mustJSON(t, map[string]any{"KeyId": keyID, "Limit": 10}))
	assertStatus(t, resp, http.StatusOK)
	var aliasOut struct {
		Aliases []struct {
			AliasName   string `json:"AliasName"`
			TargetKeyID string `json:"TargetKeyId"`
		} `json:"Aliases"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &aliasOut); err != nil {
		t.Fatalf("unmarshal list aliases response: %v", err)
	}
	if len(aliasOut.Aliases) == 0 || aliasOut.Aliases[0].AliasName != aliasName || aliasOut.Aliases[0].TargetKeyID != keyID {
		t.Fatalf("expected alias %q to target key %q", aliasName, keyID)
	}

	resp = kmsRequest(t, ts, "UpdateAlias", mustJSON(t, map[string]any{
		"AliasName":   aliasName,
		"TargetKeyId": keyID2,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "ListAliases", mustJSON(t, map[string]any{"KeyId": keyID2, "Limit": 10}))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &aliasOut); err != nil {
		t.Fatalf("unmarshal list aliases response after update: %v", err)
	}
	if len(aliasOut.Aliases) == 0 || aliasOut.Aliases[0].TargetKeyID != keyID2 {
		t.Fatalf("expected alias to retarget key %q", keyID2)
	}

	resp = kmsRequest(t, ts, "UpdateKeyDescription", mustJSON(t, map[string]any{
		"KeyId":       keyID,
		"Description": "stage1-updated",
	}))
	assertStatus(t, resp, http.StatusOK)
	if got := describe(keyID).KeyMetadata.Description; got != "stage1-updated" {
		t.Fatalf("expected updated description, got %q", got)
	}

	resp = kmsRequest(t, ts, "DisableKey", mustJSON(t, map[string]any{"KeyId": keyID}))
	assertStatus(t, resp, http.StatusOK)
	if got := describe(keyID).KeyMetadata.KeyState; got != "Disabled" {
		t.Fatalf("expected key state Disabled, got %q", got)
	}

	resp = kmsRequest(t, ts, "EnableKey", mustJSON(t, map[string]any{"KeyId": keyID}))
	assertStatus(t, resp, http.StatusOK)
	if got := describe(keyID).KeyMetadata.KeyState; got != "Enabled" {
		t.Fatalf("expected key state Enabled, got %q", got)
	}

	resp = kmsRequest(t, ts, "ScheduleKeyDeletion", mustJSON(t, map[string]any{
		"KeyId":               keyID,
		"PendingWindowInDays": 7,
	}))
	assertStatus(t, resp, http.StatusOK)
	var scheduleOut struct {
		KeyID        string    `json:"KeyId"`
		DeletionDate time.Time `json:"DeletionDate"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &scheduleOut); err != nil {
		t.Fatalf("unmarshal schedule key deletion response: %v", err)
	}
	if scheduleOut.KeyID != keyID {
		t.Fatalf("expected scheduled key id %q, got %q", keyID, scheduleOut.KeyID)
	}
	if scheduleOut.DeletionDate.IsZero() {
		t.Fatalf("expected deletion date in schedule response")
	}
	if got := describe(keyID).KeyMetadata.KeyState; got != "PendingDeletion" {
		t.Fatalf("expected key state PendingDeletion, got %q", got)
	}

	resp = kmsRequest(t, ts, "CancelKeyDeletion", mustJSON(t, map[string]any{"KeyId": keyID}))
	assertStatus(t, resp, http.StatusOK)
	var cancelOut struct {
		KeyID string `json:"KeyId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &cancelOut); err != nil {
		t.Fatalf("unmarshal cancel key deletion response: %v", err)
	}
	if cancelOut.KeyID != keyID {
		t.Fatalf("expected canceled key id %q, got %q", keyID, cancelOut.KeyID)
	}
	if got := describe(keyID).KeyMetadata.KeyState; got != "Disabled" {
		t.Fatalf("expected key state Disabled after cancel, got %q", got)
	}

	resp = kmsRequest(t, ts, "DeleteAlias", mustJSON(t, map[string]any{"AliasName": aliasName}))
	assertStatus(t, resp, http.StatusOK)
	resp = kmsRequest(t, ts, "ListAliases", mustJSON(t, map[string]any{"KeyId": keyID2, "Limit": 10}))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &aliasOut); err != nil {
		t.Fatalf("unmarshal list aliases response after delete: %v", err)
	}
	if len(aliasOut.Aliases) != 0 {
		t.Fatalf("expected alias list to be empty after delete")
	}
}
