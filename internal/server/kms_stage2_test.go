package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func decodeBase64Field(t *testing.T, encoded string) []byte {
	t.Helper()
	out, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode base64 field: %v", err)
	}
	return out
}

func TestKMSStage2CryptoFlows(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	symKeyID, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "stage2-encrypt-source",
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"KeySpec":     "SYMMETRIC_DEFAULT",
	})
	destKeyID, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "stage2-encrypt-destination",
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"KeySpec":     "SYMMETRIC_DEFAULT",
	})

	const plaintext = "stackyard-kms-stage2"
	plaintextB64 := base64.StdEncoding.EncodeToString([]byte(plaintext))

	resp := kmsRequest(t, ts, "Encrypt", mustJSON(t, map[string]any{
		"KeyId":     symKeyID,
		"Plaintext": plaintextB64,
	}))
	assertStatus(t, resp, http.StatusOK)
	var encryptOut struct {
		CiphertextBlob      string `json:"CiphertextBlob"`
		KeyID               string `json:"KeyId"`
		EncryptionAlgorithm string `json:"EncryptionAlgorithm"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &encryptOut); err != nil {
		t.Fatalf("unmarshal encrypt response: %v", err)
	}
	if encryptOut.CiphertextBlob == "" || encryptOut.KeyID == "" || encryptOut.EncryptionAlgorithm == "" {
		t.Fatalf("expected encrypt output fields to be populated")
	}

	resp = kmsRequest(t, ts, "Decrypt", mustJSON(t, map[string]any{
		"CiphertextBlob": encryptOut.CiphertextBlob,
	}))
	assertStatus(t, resp, http.StatusOK)
	var decryptOut struct {
		Plaintext           string `json:"Plaintext"`
		KeyID               string `json:"KeyId"`
		EncryptionAlgorithm string `json:"EncryptionAlgorithm"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &decryptOut); err != nil {
		t.Fatalf("unmarshal decrypt response: %v", err)
	}
	if got := string(decodeBase64Field(t, decryptOut.Plaintext)); got != plaintext {
		t.Fatalf("expected decrypted plaintext %q, got %q", plaintext, got)
	}

	resp = kmsRequest(t, ts, "ReEncrypt", mustJSON(t, map[string]any{
		"CiphertextBlob":   encryptOut.CiphertextBlob,
		"SourceKeyId":      symKeyID,
		"DestinationKeyId": destKeyID,
	}))
	assertStatus(t, resp, http.StatusOK)
	var reEncryptOut struct {
		CiphertextBlob string `json:"CiphertextBlob"`
		KeyID          string `json:"KeyId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &reEncryptOut); err != nil {
		t.Fatalf("unmarshal re-encrypt response: %v", err)
	}
	if reEncryptOut.CiphertextBlob == "" || reEncryptOut.KeyID == "" {
		t.Fatalf("expected re-encrypt output fields to be populated")
	}

	resp = kmsRequest(t, ts, "Decrypt", mustJSON(t, map[string]any{
		"CiphertextBlob": reEncryptOut.CiphertextBlob,
		"KeyId":          destKeyID,
	}))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &decryptOut); err != nil {
		t.Fatalf("unmarshal decrypt re-encrypt response: %v", err)
	}
	if got := string(decodeBase64Field(t, decryptOut.Plaintext)); got != plaintext {
		t.Fatalf("expected decrypted re-encrypted plaintext %q, got %q", plaintext, got)
	}

	resp = kmsRequest(t, ts, "GenerateDataKey", mustJSON(t, map[string]any{
		"KeyId":   symKeyID,
		"KeySpec": "AES_256",
	}))
	assertStatus(t, resp, http.StatusOK)
	var dataKeyOut struct {
		CiphertextBlob string `json:"CiphertextBlob"`
		Plaintext      string `json:"Plaintext"`
		KeyID          string `json:"KeyId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &dataKeyOut); err != nil {
		t.Fatalf("unmarshal generate data key response: %v", err)
	}
	if dataKeyOut.CiphertextBlob == "" || dataKeyOut.Plaintext == "" || dataKeyOut.KeyID == "" {
		t.Fatalf("expected generate data key output fields to be populated")
	}
	if gotLen := len(decodeBase64Field(t, dataKeyOut.Plaintext)); gotLen != 32 {
		t.Fatalf("expected 32-byte data key plaintext, got %d", gotLen)
	}

	resp = kmsRequest(t, ts, "GenerateDataKeyWithoutPlaintext", mustJSON(t, map[string]any{
		"KeyId":   symKeyID,
		"KeySpec": "AES_128",
	}))
	assertStatus(t, resp, http.StatusOK)
	var dataKeyNoPlain map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &dataKeyNoPlain); err != nil {
		t.Fatalf("unmarshal generate data key without plaintext response: %v", err)
	}
	if _, ok := dataKeyNoPlain["Plaintext"]; ok {
		t.Fatalf("expected no Plaintext field for GenerateDataKeyWithoutPlaintext")
	}
	if text, _ := dataKeyNoPlain["CiphertextBlob"].(string); text == "" {
		t.Fatalf("expected CiphertextBlob in GenerateDataKeyWithoutPlaintext response")
	}

	resp = kmsRequest(t, ts, "GenerateDataKeyPair", mustJSON(t, map[string]any{
		"KeyId":       symKeyID,
		"KeyPairSpec": "RSA_2048",
	}))
	assertStatus(t, resp, http.StatusOK)
	var pairOut struct {
		PublicKey                string `json:"PublicKey"`
		PrivateKeyPlaintext      string `json:"PrivateKeyPlaintext"`
		PrivateKeyCiphertextBlob string `json:"PrivateKeyCiphertextBlob"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &pairOut); err != nil {
		t.Fatalf("unmarshal generate data key pair response: %v", err)
	}
	if pairOut.PublicKey == "" || pairOut.PrivateKeyPlaintext == "" || pairOut.PrivateKeyCiphertextBlob == "" {
		t.Fatalf("expected key pair output fields to be populated")
	}

	resp = kmsRequest(t, ts, "GenerateDataKeyPairWithoutPlaintext", mustJSON(t, map[string]any{
		"KeyId":       symKeyID,
		"KeyPairSpec": "RSA_2048",
	}))
	assertStatus(t, resp, http.StatusOK)
	var pairNoPlain map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &pairNoPlain); err != nil {
		t.Fatalf("unmarshal generate data key pair without plaintext response: %v", err)
	}
	if _, ok := pairNoPlain["PrivateKeyPlaintext"]; ok {
		t.Fatalf("expected no PrivateKeyPlaintext field for GenerateDataKeyPairWithoutPlaintext")
	}

	resp = kmsRequest(t, ts, "GenerateRandom", mustJSON(t, map[string]any{"NumberOfBytes": 48}))
	assertStatus(t, resp, http.StatusOK)
	var randomOut struct {
		Plaintext string `json:"Plaintext"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &randomOut); err != nil {
		t.Fatalf("unmarshal generate random response: %v", err)
	}
	if gotLen := len(decodeBase64Field(t, randomOut.Plaintext)); gotLen != 48 {
		t.Fatalf("expected 48-byte random output, got %d", gotLen)
	}

	signKeyID, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "stage2-sign-key",
		"KeyUsage":    "SIGN_VERIFY",
		"KeySpec":     "RSA_2048",
	})

	resp = kmsRequest(t, ts, "GetPublicKey", mustJSON(t, map[string]any{"KeyId": signKeyID}))
	assertStatus(t, resp, http.StatusOK)
	var publicKeyOut struct {
		PublicKey string `json:"PublicKey"`
		KeyUsage  string `json:"KeyUsage"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &publicKeyOut); err != nil {
		t.Fatalf("unmarshal get public key response: %v", err)
	}
	if publicKeyOut.PublicKey == "" || publicKeyOut.KeyUsage != "SIGN_VERIFY" {
		t.Fatalf("expected public key output for sign key")
	}

	signMessage := base64.StdEncoding.EncodeToString([]byte("stackyard-sign-message"))
	resp = kmsRequest(t, ts, "Sign", mustJSON(t, map[string]any{
		"KeyId":   signKeyID,
		"Message": signMessage,
	}))
	assertStatus(t, resp, http.StatusOK)
	var signOut struct {
		Signature        string `json:"Signature"`
		SigningAlgorithm string `json:"SigningAlgorithm"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &signOut); err != nil {
		t.Fatalf("unmarshal sign response: %v", err)
	}
	if signOut.Signature == "" || signOut.SigningAlgorithm == "" {
		t.Fatalf("expected signature output fields to be populated")
	}

	resp = kmsRequest(t, ts, "Verify", mustJSON(t, map[string]any{
		"KeyId":            signKeyID,
		"Message":          signMessage,
		"Signature":        signOut.Signature,
		"SigningAlgorithm": signOut.SigningAlgorithm,
	}))
	assertStatus(t, resp, http.StatusOK)
	var verifyOut struct {
		SignatureValid bool `json:"SignatureValid"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &verifyOut); err != nil {
		t.Fatalf("unmarshal verify response: %v", err)
	}
	if !verifyOut.SignatureValid {
		t.Fatalf("expected SignatureValid to be true")
	}

	macKeyID, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "stage2-mac-key",
		"KeyUsage":    "GENERATE_VERIFY_MAC",
		"KeySpec":     "HMAC_256",
	})

	macMessage := base64.StdEncoding.EncodeToString([]byte("stackyard-mac-message"))
	resp = kmsRequest(t, ts, "GenerateMac", mustJSON(t, map[string]any{
		"KeyId":   macKeyID,
		"Message": macMessage,
	}))
	assertStatus(t, resp, http.StatusOK)
	var macOut struct {
		Mac          string `json:"Mac"`
		MacAlgorithm string `json:"MacAlgorithm"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &macOut); err != nil {
		t.Fatalf("unmarshal generate mac response: %v", err)
	}
	if macOut.Mac == "" || macOut.MacAlgorithm == "" {
		t.Fatalf("expected MAC output fields to be populated")
	}

	resp = kmsRequest(t, ts, "VerifyMac", mustJSON(t, map[string]any{
		"KeyId":        macKeyID,
		"Message":      macMessage,
		"Mac":          macOut.Mac,
		"MacAlgorithm": macOut.MacAlgorithm,
	}))
	assertStatus(t, resp, http.StatusOK)
	var verifyMacOut struct {
		MacValid bool `json:"MacValid"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &verifyMacOut); err != nil {
		t.Fatalf("unmarshal verify mac response: %v", err)
	}
	if !verifyMacOut.MacValid {
		t.Fatalf("expected MacValid to be true")
	}

	agreementKeyID, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "stage2-key-agreement",
		"KeyUsage":    "KEY_AGREEMENT",
		"KeySpec":     "ECC_NIST_P256",
	})
	peerPublic := base64.StdEncoding.EncodeToString([]byte("stackyard-peer-public"))
	resp = kmsRequest(t, ts, "DeriveSharedSecret", mustJSON(t, map[string]any{
		"KeyId":     agreementKeyID,
		"PublicKey": peerPublic,
	}))
	assertStatus(t, resp, http.StatusOK)
	var sharedSecretOut struct {
		SharedSecret string `json:"SharedSecret"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &sharedSecretOut); err != nil {
		t.Fatalf("unmarshal derive shared secret response: %v", err)
	}
	if len(decodeBase64Field(t, sharedSecretOut.SharedSecret)) == 0 {
		t.Fatalf("expected non-empty shared secret")
	}
}

func TestKMSStage2VerifyRejectsInvalidSignature(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	signKeyID, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "stage2-invalid-signature",
		"KeyUsage":    "SIGN_VERIFY",
		"KeySpec":     "RSA_2048",
	})

	message := base64.StdEncoding.EncodeToString([]byte("stackyard-invalid-signature-message"))
	resp := kmsRequest(t, ts, "Sign", mustJSON(t, map[string]any{
		"KeyId":   signKeyID,
		"Message": message,
	}))
	assertStatus(t, resp, http.StatusOK)
	var signOut struct {
		Signature string `json:"Signature"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &signOut); err != nil {
		t.Fatalf("unmarshal sign response: %v", err)
	}
	signature := decodeBase64Field(t, signOut.Signature)
	signature[0] ^= 0x01

	resp = kmsRequest(t, ts, "Verify", mustJSON(t, map[string]any{
		"KeyId":     signKeyID,
		"Message":   message,
		"Signature": base64.StdEncoding.EncodeToString(signature),
	}))
	assertStatus(t, resp, http.StatusOK)
	var verifyOut struct {
		SignatureValid bool `json:"SignatureValid"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &verifyOut); err != nil {
		t.Fatalf("unmarshal verify response: %v", err)
	}
	if verifyOut.SignatureValid {
		t.Fatalf("expected SignatureValid to be false for tampered signature")
	}

	// Also verify a mismatched message fails.
	resp = kmsRequest(t, ts, "Verify", mustJSON(t, map[string]any{
		"KeyId":     signKeyID,
		"Message":   base64.StdEncoding.EncodeToString([]byte("different")),
		"Signature": signOut.Signature,
	}))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &verifyOut); err != nil {
		t.Fatalf("unmarshal verify response for mismatched message: %v", err)
	}
	if verifyOut.SignatureValid {
		t.Fatalf("expected SignatureValid to be false for mismatched message")
	}

	// Ensure decrypt validation catches wrong key reference for ciphertext.
	symKeyID, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "stage2-decrypt-wrong-key-source",
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"KeySpec":     "SYMMETRIC_DEFAULT",
	})
	otherSymKeyID, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "stage2-decrypt-wrong-key-target",
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"KeySpec":     "SYMMETRIC_DEFAULT",
	})
	resp = kmsRequest(t, ts, "Encrypt", mustJSON(t, map[string]any{
		"KeyId":     symKeyID,
		"Plaintext": base64.StdEncoding.EncodeToString([]byte("stackyard-decrypt-key-check")),
	}))
	assertStatus(t, resp, http.StatusOK)
	var encryptOut struct {
		CiphertextBlob string `json:"CiphertextBlob"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &encryptOut); err != nil {
		t.Fatalf("unmarshal encrypt response: %v", err)
	}

	resp = kmsRequest(t, ts, "Decrypt", mustJSON(t, map[string]any{
		"KeyId":          otherSymKeyID,
		"CiphertextBlob": encryptOut.CiphertextBlob,
	}))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected decrypt with wrong key to return 400, got %d", resp.StatusCode)
	}
	if !bytes.Contains(mustBody(t, resp), []byte("IncorrectKeyException")) {
		t.Fatalf("expected IncorrectKeyException for decrypt with wrong key")
	}
}
