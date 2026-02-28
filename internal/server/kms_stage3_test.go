package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKMSStage3PoliciesGrantsAndTags(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	keyID, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "stage3-key",
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"KeySpec":     "SYMMETRIC_DEFAULT",
	})

	resp := kmsRequest(t, ts, "PutKeyPolicy", mustJSON(t, map[string]any{
		"KeyId":      keyID,
		"PolicyName": "default",
		"Policy":     `{"Version":"2012-10-17","Statement":[]}`,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "GetKeyPolicy", mustJSON(t, map[string]any{
		"KeyId":      keyID,
		"PolicyName": "default",
	}))
	assertStatus(t, resp, http.StatusOK)
	var getPolicyOut struct {
		PolicyName string `json:"PolicyName"`
		Policy     string `json:"Policy"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getPolicyOut); err != nil {
		t.Fatalf("unmarshal get key policy response: %v", err)
	}
	if getPolicyOut.PolicyName != "default" || getPolicyOut.Policy == "" {
		t.Fatalf("expected default policy response")
	}

	resp = kmsRequest(t, ts, "ListKeyPolicies", mustJSON(t, map[string]any{
		"KeyId": keyID,
		"Limit": 10,
	}))
	assertStatus(t, resp, http.StatusOK)
	var listPolicyOut struct {
		PolicyNames []string `json:"PolicyNames"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listPolicyOut); err != nil {
		t.Fatalf("unmarshal list key policies response: %v", err)
	}
	if len(listPolicyOut.PolicyNames) == 0 {
		t.Fatalf("expected at least one key policy")
	}

	resp = kmsRequest(t, ts, "TagResource", mustJSON(t, map[string]any{
		"KeyId": keyID,
		"Tags": []map[string]any{
			{"TagKey": "env", "TagValue": "test"},
			{"TagKey": "team", "TagValue": "stackyard"},
		},
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "ListResourceTags", mustJSON(t, map[string]any{
		"KeyId": keyID,
		"Limit": 10,
	}))
	assertStatus(t, resp, http.StatusOK)
	var listTagsOut struct {
		Tags []struct {
			TagKey   string `json:"TagKey"`
			TagValue string `json:"TagValue"`
		} `json:"Tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listTagsOut); err != nil {
		t.Fatalf("unmarshal list resource tags response: %v", err)
	}
	if len(listTagsOut.Tags) < 2 {
		t.Fatalf("expected at least two tags")
	}

	resp = kmsRequest(t, ts, "UntagResource", mustJSON(t, map[string]any{
		"KeyId":   keyID,
		"TagKeys": []string{"team"},
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "CreateGrant", mustJSON(t, map[string]any{
		"KeyId":             keyID,
		"Name":              "stage3-grant",
		"GranteePrincipal":  "arn:aws:iam::123456789012:role/stackyard-grantee",
		"RetiringPrincipal": "arn:aws:iam::123456789012:role/stackyard-retirer",
		"Operations":        []string{"Encrypt", "Decrypt"},
	}))
	assertStatus(t, resp, http.StatusOK)
	var createGrantOut struct {
		GrantID    string `json:"GrantId"`
		GrantToken string `json:"GrantToken"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createGrantOut); err != nil {
		t.Fatalf("unmarshal create grant response: %v", err)
	}
	if createGrantOut.GrantID == "" || createGrantOut.GrantToken == "" {
		t.Fatalf("expected grant id and token")
	}

	resp = kmsRequest(t, ts, "ListGrants", mustJSON(t, map[string]any{
		"KeyId": keyID,
		"Limit": 10,
	}))
	assertStatus(t, resp, http.StatusOK)
	var listGrantsOut struct {
		Grants []struct {
			GrantID string `json:"GrantId"`
		} `json:"Grants"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listGrantsOut); err != nil {
		t.Fatalf("unmarshal list grants response: %v", err)
	}
	if len(listGrantsOut.Grants) == 0 {
		t.Fatalf("expected at least one grant")
	}

	resp = kmsRequest(t, ts, "ListRetirableGrants", mustJSON(t, map[string]any{
		"RetiringPrincipal": "arn:aws:iam::123456789012:role/stackyard-retirer",
		"Limit":             10,
	}))
	assertStatus(t, resp, http.StatusOK)
	var retirableOut struct {
		Grants []struct {
			GrantID string `json:"GrantId"`
		} `json:"Grants"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &retirableOut); err != nil {
		t.Fatalf("unmarshal list retirable grants response: %v", err)
	}
	if len(retirableOut.Grants) == 0 {
		t.Fatalf("expected at least one retirable grant")
	}

	resp = kmsRequest(t, ts, "RetireGrant", mustJSON(t, map[string]any{
		"GrantId": createGrantOut.GrantID,
		"KeyId":   keyID,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "CreateGrant", mustJSON(t, map[string]any{
		"KeyId":            keyID,
		"GranteePrincipal": "arn:aws:iam::123456789012:role/stackyard-grantee",
		"Operations":       []string{"Encrypt"},
	}))
	assertStatus(t, resp, http.StatusOK)
	var createGrantOut2 struct {
		GrantID string `json:"GrantId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createGrantOut2); err != nil {
		t.Fatalf("unmarshal second create grant response: %v", err)
	}
	if createGrantOut2.GrantID == "" {
		t.Fatalf("expected second grant id")
	}

	resp = kmsRequest(t, ts, "RevokeGrant", mustJSON(t, map[string]any{
		"KeyId":   keyID,
		"GrantId": createGrantOut2.GrantID,
	}))
	assertStatus(t, resp, http.StatusOK)
}
