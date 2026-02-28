package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecretsManagerStage345RotationReplicationPolicyAndTags(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := secretsManagerRequest(t, ts, "CreateSecret", mustJSON(t, map[string]any{
		"Name":               "stage345-secret",
		"ClientRequestToken": "stage345-create-token",
		"Description":        "stage345",
		"SecretString":       "{\"token\":\"initial\"}",
		"Tags": []map[string]string{
			{"Key": "env", "Value": "dev"},
			{"Key": "team", "Value": "platform"},
		},
	}))
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		ARN string `json:"ARN"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create output: %v", err)
	}
	if strings.TrimSpace(createOut.ARN) == "" {
		t.Fatalf("expected secret arn")
	}

	resp = secretsManagerRequest(t, ts, "RotateSecret", mustJSON(t, map[string]any{
		"SecretId":           createOut.ARN,
		"ClientRequestToken": "stage345-rotate-token",
		"RotationLambdaARN":  "arn:aws:lambda:us-east-1:123456789012:function:stage345-rotate",
		"RotateImmediately":  true,
		"RotationRules": map[string]any{
			"AutomaticallyAfterDays": 30,
		},
	}))
	assertStatus(t, resp, http.StatusOK)
	var rotateOut struct {
		VersionID string `json:"VersionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &rotateOut); err != nil {
		t.Fatalf("unmarshal rotate output: %v", err)
	}
	if strings.TrimSpace(rotateOut.VersionID) == "" {
		t.Fatalf("expected rotate version id")
	}

	resp = secretsManagerRequest(t, ts, "RotateSecret", mustJSON(t, map[string]any{
		"SecretId":           createOut.ARN,
		"ClientRequestToken": "stage345-rotate-token",
		"RotationLambdaARN":  "arn:aws:lambda:us-east-1:123456789012:function:stage345-rotate",
		"RotateImmediately":  true,
		"RotationRules": map[string]any{
			"AutomaticallyAfterDays": 30,
		},
	}))
	assertStatus(t, resp, http.StatusOK)
	var rotateOut2 struct {
		VersionID string `json:"VersionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &rotateOut2); err != nil {
		t.Fatalf("unmarshal second rotate output: %v", err)
	}
	if rotateOut2.VersionID != rotateOut.VersionID {
		t.Fatalf("expected idempotent rotate version id %q, got %q", rotateOut.VersionID, rotateOut2.VersionID)
	}

	resp = secretsManagerRequest(t, ts, "ReplicateSecretToRegions", mustJSON(t, map[string]any{
		"SecretId": createOut.ARN,
		"AddReplicaRegions": []map[string]any{
			{"Region": "us-west-2", "KmsKeyId": "alias/stage345"},
			{"Region": "us-west-1"},
		},
	}))
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), "us-west-2") {
		t.Fatalf("expected replication status response to include regions")
	}

	resp = secretsManagerRequest(t, ts, "DeleteSecret", mustJSON(t, map[string]any{
		"SecretId":             createOut.ARN,
		"RecoveryWindowInDays": 7,
	}))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "InvalidRequestException") {
		t.Fatalf("expected invalid request when deleting replicated secret")
	}

	resp = secretsManagerRequest(t, ts, "ListSecrets", mustJSON(t, map[string]any{
		"Filters": []map[string]any{
			{"Key": "name", "Values": []string{"stage345"}},
		},
	}))
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), createOut.ARN) {
		t.Fatalf("expected filtered list to include created secret")
	}

	resp = secretsManagerRequest(t, ts, "BatchGetSecretValue", mustJSON(t, map[string]any{
		"SecretIdList": []string{createOut.ARN},
		"Filters": []map[string]any{
			{"Key": "name", "Values": []string{"stage345"}},
		},
	}))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ValidationException") {
		t.Fatalf("expected validation exception for invalid batch-get contract")
	}

	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"arn:aws:iam::123456789012:root"},"Action":"secretsmanager:GetSecretValue","Resource":"*"}]}`
	resp = secretsManagerRequest(t, ts, "PutResourcePolicy", mustJSON(t, map[string]any{
		"SecretId":       createOut.ARN,
		"ResourcePolicy": policy,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = secretsManagerRequest(t, ts, "GetResourcePolicy", mustJSON(t, map[string]any{
		"SecretId": createOut.ARN,
	}))
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), "ResourcePolicy") {
		t.Fatalf("expected get resource policy response")
	}

	resp = secretsManagerRequest(t, ts, "ValidateResourcePolicy", mustJSON(t, map[string]any{
		"SecretId":          createOut.ARN,
		"BlockPublicPolicy": true,
	}))
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), `"PolicyValidationPassed":true`) {
		t.Fatalf("expected successful policy validation")
	}

	resp = secretsManagerRequest(t, ts, "ValidateResourcePolicy", mustJSON(t, map[string]any{
		"ResourcePolicy":    `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"secretsmanager:GetSecretValue","Resource":"*"}]}`,
		"BlockPublicPolicy": true,
	}))
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), `"PolicyValidationPassed":false`) {
		t.Fatalf("expected failed validation for wildcard policy")
	}

	resp = secretsManagerRequest(t, ts, "TagResource", mustJSON(t, map[string]any{
		"SecretId": createOut.ARN,
		"Tags": []map[string]string{
			{"Key": "owner", "Value": "qa"},
		},
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = secretsManagerRequest(t, ts, "UntagResource", mustJSON(t, map[string]any{
		"SecretId": createOut.ARN,
		"TagKeys":  []string{"team"},
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = secretsManagerRequest(t, ts, "DescribeSecret", mustJSON(t, map[string]any{
		"SecretId": createOut.ARN,
	}))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, `"Key":"owner"`) {
		t.Fatalf("expected owner tag to be present")
	}
	if strings.Contains(body, `"Key":"team"`) {
		t.Fatalf("expected team tag to be removed")
	}

	resp = secretsManagerRequest(t, ts, "CancelRotateSecret", mustJSON(t, map[string]any{
		"SecretId": createOut.ARN,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = secretsManagerRequest(t, ts, "RemoveRegionsFromReplication", mustJSON(t, map[string]any{
		"SecretId":             createOut.ARN,
		"RemoveReplicaRegions": []string{"us-west-2"},
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = secretsManagerRequest(t, ts, "StopReplicationToReplica", mustJSON(t, map[string]any{
		"SecretId":      createOut.ARN,
		"ReplicaRegion": "us-west-1",
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = secretsManagerRequest(t, ts, "DeleteResourcePolicy", mustJSON(t, map[string]any{
		"SecretId": createOut.ARN,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = secretsManagerRequest(t, ts, "DeleteSecret", mustJSON(t, map[string]any{
		"SecretId":             createOut.ARN,
		"RecoveryWindowInDays": 7,
	}))
	assertStatus(t, resp, http.StatusOK)
}

func TestSecretsManagerStage34ActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := secretsManagerRequest(t, ts, "CreateSecret", mustJSON(t, map[string]any{
		"Name":               "stage34-implemented",
		"ClientRequestToken": "stage34-implemented-create",
		"SecretString":       "{\"k\":\"v\"}",
	}))
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		ARN string `json:"ARN"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create output: %v", err)
	}

	actions := []struct {
		Name string
		Body []byte
	}{
		{Name: "RotateSecret", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN, "RotateImmediately": true, "RotationLambdaARN": "arn:aws:lambda:us-east-1:123456789012:function:stage34-rotate"})},
		{Name: "CancelRotateSecret", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN})},
		{Name: "ReplicateSecretToRegions", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN, "AddReplicaRegions": []map[string]any{{"Region": "us-west-2"}}})},
		{Name: "RemoveRegionsFromReplication", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN, "RemoveReplicaRegions": []string{"us-west-2"}})},
		{Name: "ReplicateSecretToRegions", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN, "AddReplicaRegions": []map[string]any{{"Region": "us-west-1"}}})},
		{Name: "StopReplicationToReplica", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN, "ReplicaRegion": "us-west-1"})},
		{Name: "PutResourcePolicy", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN, "ResourcePolicy": `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"arn:aws:iam::123456789012:root"},"Action":"secretsmanager:GetSecretValue","Resource":"*"}` + `]}`})},
		{Name: "GetResourcePolicy", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN})},
		{Name: "ValidateResourcePolicy", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN})},
		{Name: "DeleteResourcePolicy", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN})},
		{Name: "TagResource", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN, "Tags": []map[string]string{{"Key": "env", "Value": "dev"}}})},
		{Name: "UntagResource", Body: mustJSON(t, map[string]any{"SecretId": createOut.ARN, "TagKeys": []string{"env"}})},
	}

	for _, action := range actions {
		resp = secretsManagerRequest(t, ts, action.Name, action.Body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.Name)
		}
	}
}
