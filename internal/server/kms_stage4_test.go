package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestKMSStage4RotationImportAndReplication(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	keyID, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "stage4-key",
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"KeySpec":     "SYMMETRIC_DEFAULT",
	})

	resp := kmsRequest(t, ts, "EnableKeyRotation", mustJSON(t, map[string]any{"KeyId": keyID}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "GetKeyRotationStatus", mustJSON(t, map[string]any{"KeyId": keyID}))
	assertStatus(t, resp, http.StatusOK)
	var rotationStatus struct {
		KeyRotationEnabled bool `json:"KeyRotationEnabled"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &rotationStatus); err != nil {
		t.Fatalf("unmarshal get key rotation status response: %v", err)
	}
	if !rotationStatus.KeyRotationEnabled {
		t.Fatalf("expected KeyRotationEnabled=true")
	}

	resp = kmsRequest(t, ts, "RotateKeyOnDemand", mustJSON(t, map[string]any{"KeyId": keyID}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "ListKeyRotations", mustJSON(t, map[string]any{
		"KeyId": keyID,
		"Limit": 10,
	}))
	assertStatus(t, resp, http.StatusOK)
	var listRotationsOut struct {
		Rotations []struct {
			RotationType string `json:"RotationType"`
		} `json:"Rotations"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listRotationsOut); err != nil {
		t.Fatalf("unmarshal list key rotations response: %v", err)
	}
	if len(listRotationsOut.Rotations) == 0 {
		t.Fatalf("expected at least one key rotation entry")
	}

	resp = kmsRequest(t, ts, "DisableKeyRotation", mustJSON(t, map[string]any{"KeyId": keyID}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "GetParametersForImport", mustJSON(t, map[string]any{
		"KeyId":             keyID,
		"WrappingAlgorithm": "RSAES_OAEP_SHA_256",
		"WrappingKeySpec":   "RSA_2048",
	}))
	assertStatus(t, resp, http.StatusOK)
	var importParamsOut struct {
		ImportToken       string    `json:"ImportToken"`
		PublicKey         string    `json:"PublicKey"`
		ParametersValidTo time.Time `json:"ParametersValidTo"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &importParamsOut); err != nil {
		t.Fatalf("unmarshal get parameters for import response: %v", err)
	}
	if importParamsOut.ImportToken == "" || importParamsOut.PublicKey == "" || importParamsOut.ParametersValidTo.IsZero() {
		t.Fatalf("expected import parameters output fields to be populated")
	}

	resp = kmsRequest(t, ts, "ImportKeyMaterial", mustJSON(t, map[string]any{
		"KeyId":                keyID,
		"EncryptedKeyMaterial": base64.StdEncoding.EncodeToString([]byte("stage4-imported-material")),
		"ImportToken":          importParamsOut.ImportToken,
		"ExpirationModel":      "KEY_MATERIAL_DOES_NOT_EXPIRE",
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "DescribeKey", mustJSON(t, map[string]any{"KeyId": keyID}))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		KeyMetadata struct {
			Origin string `json:"Origin"`
		} `json:"KeyMetadata"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe key response: %v", err)
	}
	if describeOut.KeyMetadata.Origin != "EXTERNAL" {
		t.Fatalf("expected key origin EXTERNAL after import, got %q", describeOut.KeyMetadata.Origin)
	}

	resp = kmsRequest(t, ts, "DeleteImportedKeyMaterial", mustJSON(t, map[string]any{"KeyId": keyID}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "DescribeKey", mustJSON(t, map[string]any{"KeyId": keyID}))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe key response after delete imported material: %v", err)
	}
	if describeOut.KeyMetadata.Origin != "AWS_KMS" {
		t.Fatalf("expected key origin AWS_KMS after delete imported material, got %q", describeOut.KeyMetadata.Origin)
	}

	resp = kmsRequest(t, ts, "ReplicateKey", mustJSON(t, map[string]any{
		"KeyId":         keyID,
		"ReplicaRegion": "us-west-2",
	}))
	assertStatus(t, resp, http.StatusOK)
	var replicateOut struct {
		ReplicaKeyMetadata struct {
			KeyID string `json:"KeyId"`
			Arn   string `json:"Arn"`
		} `json:"ReplicaKeyMetadata"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &replicateOut); err != nil {
		t.Fatalf("unmarshal replicate key response: %v", err)
	}
	if replicateOut.ReplicaKeyMetadata.KeyID == "" || !strings.Contains(replicateOut.ReplicaKeyMetadata.Arn, ":us-west-2:") {
		t.Fatalf("expected replica key metadata with us-west-2 ARN")
	}

	resp = kmsRequest(t, ts, "UpdatePrimaryRegion", mustJSON(t, map[string]any{
		"KeyId":         keyID,
		"PrimaryRegion": "us-east-2",
	}))
	assertStatus(t, resp, http.StatusOK)
	var updatePrimaryOut struct {
		KeyMetadata struct {
			Arn           string `json:"Arn"`
			PrimaryRegion string `json:"PrimaryRegion"`
		} `json:"KeyMetadata"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updatePrimaryOut); err != nil {
		t.Fatalf("unmarshal update primary region response: %v", err)
	}
	if updatePrimaryOut.KeyMetadata.PrimaryRegion != "us-east-2" || !strings.Contains(updatePrimaryOut.KeyMetadata.Arn, ":us-east-2:") {
		t.Fatalf("expected updated primary region in response")
	}
}
