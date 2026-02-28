package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecretsManagerStage12LifecycleAndValues(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := secretsManagerRequest(t, ts, "CreateSecret", mustJSON(t, map[string]any{
		"Name":               "stage12-secret",
		"ClientRequestToken": "stage12-create-token",
		"Description":        "stage12",
		"SecretString":       "{\"username\":\"stackyard\",\"password\":\"initial\"}",
		"Tags":               []map[string]string{{"Key": "env", "Value": "dev"}},
	}))
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		ARN       string `json:"ARN"`
		Name      string `json:"Name"`
		VersionID string `json:"VersionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if strings.TrimSpace(createOut.ARN) == "" || strings.TrimSpace(createOut.VersionID) == "" {
		t.Fatalf("expected create response to include arn and version id")
	}

	resp = secretsManagerRequest(t, ts, "DescribeSecret", mustJSON(t, map[string]any{
		"SecretId": createOut.ARN,
	}))
	assertStatus(t, resp, http.StatusOK)
	describeBody := string(mustBody(t, resp))
	if !strings.Contains(describeBody, `"Name":"stage12-secret"`) {
		t.Fatalf("expected describe response to include secret name, got %s", describeBody)
	}

	resp = secretsManagerRequest(t, ts, "ListSecrets", mustJSON(t, map[string]any{
		"MaxResults": 10,
	}))
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), createOut.ARN) {
		t.Fatalf("expected list response to include created arn")
	}

	resp = secretsManagerRequest(t, ts, "UpdateSecret", mustJSON(t, map[string]any{
		"SecretId":           createOut.ARN,
		"ClientRequestToken": "stage12-update-token",
		"Description":        "stage12-updated",
		"SecretString":       "{\"username\":\"stackyard\",\"password\":\"updated\"}",
	}))
	assertStatus(t, resp, http.StatusOK)
	var updateOut struct {
		VersionID string `json:"VersionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updateOut); err != nil {
		t.Fatalf("unmarshal update response: %v", err)
	}
	if strings.TrimSpace(updateOut.VersionID) == "" {
		t.Fatalf("expected update response to include version id")
	}

	resp = secretsManagerRequest(t, ts, "PutSecretValue", mustJSON(t, map[string]any{
		"SecretId":           createOut.ARN,
		"ClientRequestToken": "stage12-put-token",
		"SecretString":       "{\"username\":\"stackyard\",\"password\":\"put\"}",
		"VersionStages":      []string{"AWSCURRENT"},
	}))
	assertStatus(t, resp, http.StatusOK)
	var putOut struct {
		VersionID string `json:"VersionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &putOut); err != nil {
		t.Fatalf("unmarshal put response: %v", err)
	}
	if strings.TrimSpace(putOut.VersionID) == "" {
		t.Fatalf("expected put response to include version id")
	}

	resp = secretsManagerRequest(t, ts, "GetSecretValue", mustJSON(t, map[string]any{
		"SecretId":     createOut.ARN,
		"VersionStage": "AWSCURRENT",
	}))
	assertStatus(t, resp, http.StatusOK)
	var getOut struct {
		SecretString string `json:"SecretString"`
		VersionID    string `json:"VersionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getOut); err != nil {
		t.Fatalf("unmarshal get response: %v", err)
	}
	if getOut.VersionID != putOut.VersionID {
		t.Fatalf("expected get version %q, got %q", putOut.VersionID, getOut.VersionID)
	}
	if !strings.Contains(getOut.SecretString, `"password":"put"`) {
		t.Fatalf("expected get response to include latest secret value, got %s", getOut.SecretString)
	}

	resp = secretsManagerRequest(t, ts, "ListSecretVersionIds", mustJSON(t, map[string]any{
		"SecretId":   createOut.ARN,
		"MaxResults": 10,
	}))
	assertStatus(t, resp, http.StatusOK)
	var versionsOut struct {
		Versions []struct {
			VersionID string `json:"VersionId"`
		} `json:"Versions"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &versionsOut); err != nil {
		t.Fatalf("unmarshal versions response: %v", err)
	}
	if len(versionsOut.Versions) < 2 {
		t.Fatalf("expected at least two versions, got %d", len(versionsOut.Versions))
	}

	resp = secretsManagerRequest(t, ts, "UpdateSecretVersionStage", mustJSON(t, map[string]any{
		"SecretId":        createOut.ARN,
		"VersionStage":    "AWSPREVIOUS",
		"MoveToVersionId": versionsOut.Versions[0].VersionID,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = secretsManagerRequest(t, ts, "BatchGetSecretValue", mustJSON(t, map[string]any{
		"SecretIdList": []string{createOut.ARN, "missing-secret"},
		"MaxResults":   10,
	}))
	assertStatus(t, resp, http.StatusOK)
	batchBody := string(mustBody(t, resp))
	if !strings.Contains(batchBody, `"SecretValues"`) || !strings.Contains(batchBody, `"Errors"`) {
		t.Fatalf("expected batch response to include values and errors, got %s", batchBody)
	}

	resp = secretsManagerRequest(t, ts, "GetRandomPassword", mustJSON(t, map[string]any{
		"PasswordLength":          20,
		"ExcludePunctuation":      true,
		"RequireEachIncludedType": true,
	}))
	assertStatus(t, resp, http.StatusOK)
	var passwordOut struct {
		RandomPassword string `json:"RandomPassword"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &passwordOut); err != nil {
		t.Fatalf("unmarshal random password response: %v", err)
	}
	if len(passwordOut.RandomPassword) != 20 {
		t.Fatalf("expected random password length 20, got %d", len(passwordOut.RandomPassword))
	}

	resp = secretsManagerRequest(t, ts, "DeleteSecret", mustJSON(t, map[string]any{
		"SecretId":             createOut.ARN,
		"RecoveryWindowInDays": 7,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = secretsManagerRequest(t, ts, "RestoreSecret", mustJSON(t, map[string]any{
		"SecretId": createOut.ARN,
	}))
	assertStatus(t, resp, http.StatusOK)
}

func TestSecretsManagerStage12ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := secretsManagerRequest(t, ts, "CreateSecret", mustJSON(t, map[string]any{
		"Name":               "stage12-implemented",
		"ClientRequestToken": "stage12-implemented-create",
		"SecretString":       "{\"k\":\"v\"}",
	}))
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		ARN string `json:"ARN"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}

	resp = secretsManagerRequest(t, ts, "PutSecretValue", mustJSON(t, map[string]any{
		"SecretId":           createOut.ARN,
		"ClientRequestToken": "stage12-implemented-put",
		"SecretString":       "{\"k\":\"v2\"}",
	}))
	assertStatus(t, resp, http.StatusOK)
	var putOut struct {
		VersionID string `json:"VersionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &putOut); err != nil {
		t.Fatalf("unmarshal put response: %v", err)
	}

	resp = secretsManagerRequest(t, ts, "ListSecretVersionIds", mustJSON(t, map[string]any{
		"SecretId": createOut.ARN,
	}))
	assertStatus(t, resp, http.StatusOK)
	var versionsOut struct {
		Versions []struct {
			VersionID string `json:"VersionId"`
		} `json:"Versions"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &versionsOut); err != nil {
		t.Fatalf("unmarshal versions response: %v", err)
	}

	actions := []struct {
		Name string
		Body []byte
	}{
		{Name: "DescribeSecret", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN})},
		{Name: "ListSecrets", Body: mustJSON(t, map[string]any{"MaxResults": 10})},
		{Name: "UpdateSecret", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN, "Description": "updated"})},
		{Name: "GetSecretValue", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN})},
		{Name: "ListSecretVersionIds", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN})},
		{Name: "UpdateSecretVersionStage", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN, "VersionStage": "AWSPREVIOUS", "MoveToVersionId": putOut.VersionID})},
		{Name: "BatchGetSecretValue", Body: mustJSON(t, map[string]any{"SecretIdList": []string{createOut.ARN}})},
		{Name: "GetRandomPassword", Body: mustJSON(t, map[string]any{"PasswordLength": 16})},
		{Name: "DeleteSecret", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN, "RecoveryWindowInDays": 7})},
		{Name: "RestoreSecret", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN})},
	}

	for _, action := range actions {
		resp = secretsManagerRequest(t, ts, action.Name, action.Body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.Name)
		}
	}
}
