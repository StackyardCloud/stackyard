package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard KMS advanced client using %s\n", endpoint)

	encryptKeyID := mustCreateKey(ctx, endpoint, region, creds, map[string]any{
		"Description": "stackyard advanced encrypt key",
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"KeySpec":     "SYMMETRIC_DEFAULT",
	}, "CreateKey(encrypt)")
	logf("CreateKey(encrypt) succeeded key-id=%s", encryptKeyID)

	plaintext := "stackyard advanced plaintext"
	plaintextB64 := base64.StdEncoding.EncodeToString([]byte(plaintext))
	encryptStatus, encryptBody, err := kmsRequest(ctx, endpoint, region, creds, "Encrypt", map[string]any{
		"KeyId":     encryptKeyID,
		"Plaintext": plaintextB64,
	})
	if err != nil {
		exitf("Encrypt request failed: %v", err)
	}
	var encryptOut struct {
		CiphertextBlob string `json:"CiphertextBlob"`
	}
	if err := expectJSONStatus("Encrypt", encryptStatus, encryptBody, http.StatusOK, &encryptOut); err != nil {
		exitf("Encrypt response validation failed: %v", err)
	}
	logf("Encrypt succeeded (%d)", encryptStatus)

	decryptStatus, decryptBody, err := kmsRequest(ctx, endpoint, region, creds, "Decrypt", map[string]any{
		"CiphertextBlob": encryptOut.CiphertextBlob,
		"KeyId":          encryptKeyID,
	})
	if err != nil {
		exitf("Decrypt request failed: %v", err)
	}
	var decryptOut struct {
		Plaintext string `json:"Plaintext"`
	}
	if err := expectJSONStatus("Decrypt", decryptStatus, decryptBody, http.StatusOK, &decryptOut); err != nil {
		exitf("Decrypt response validation failed: %v", err)
	}
	decryptedBytes, err := base64.StdEncoding.DecodeString(decryptOut.Plaintext)
	if err != nil {
		exitf("Decrypt response plaintext decode failed: %v", err)
	}
	if string(decryptedBytes) != plaintext {
		exitf("Decrypt plaintext mismatch: want=%q got=%q", plaintext, string(decryptedBytes))
	}
	logf("Decrypt succeeded (%d)", decryptStatus)

	aliasName := "alias/stackyard-advanced-" + strconv.FormatInt(time.Now().Unix(), 10)
	createAliasStatus, createAliasBody, err := kmsRequest(ctx, endpoint, region, creds, "CreateAlias", map[string]any{
		"AliasName":   aliasName,
		"TargetKeyId": encryptKeyID,
	})
	if err != nil {
		exitf("CreateAlias request failed: %v", err)
	}
	if err := expectJSONStatus("CreateAlias", createAliasStatus, createAliasBody, http.StatusOK, nil); err != nil {
		exitf("CreateAlias response validation failed: %v", err)
	}
	logf("CreateAlias succeeded (%d) alias=%s", createAliasStatus, aliasName)

	dataKeyStatus, dataKeyBody, err := kmsRequest(ctx, endpoint, region, creds, "GenerateDataKey", map[string]any{
		"KeyId":   encryptKeyID,
		"KeySpec": "AES_256",
	})
	if err != nil {
		exitf("GenerateDataKey request failed: %v", err)
	}
	var dataKeyOut struct {
		Plaintext string `json:"Plaintext"`
	}
	if err := expectJSONStatus("GenerateDataKey", dataKeyStatus, dataKeyBody, http.StatusOK, &dataKeyOut); err != nil {
		exitf("GenerateDataKey response validation failed: %v", err)
	}
	if decoded, err := base64.StdEncoding.DecodeString(dataKeyOut.Plaintext); err != nil || len(decoded) != 32 {
		exitf("GenerateDataKey plaintext validation failed")
	}
	logf("GenerateDataKey succeeded (%d)", dataKeyStatus)

	signKeyID := mustCreateKey(ctx, endpoint, region, creds, map[string]any{
		"Description": "stackyard advanced sign key",
		"KeyUsage":    "SIGN_VERIFY",
		"KeySpec":     "RSA_2048",
	}, "CreateKey(sign)")
	signMessage := base64.StdEncoding.EncodeToString([]byte("stackyard-sign-message"))
	signStatus, signBody, err := kmsRequest(ctx, endpoint, region, creds, "Sign", map[string]any{
		"KeyId":   signKeyID,
		"Message": signMessage,
	})
	if err != nil {
		exitf("Sign request failed: %v", err)
	}
	var signOut struct {
		Signature        string `json:"Signature"`
		SigningAlgorithm string `json:"SigningAlgorithm"`
	}
	if err := expectJSONStatus("Sign", signStatus, signBody, http.StatusOK, &signOut); err != nil {
		exitf("Sign response validation failed: %v", err)
	}
	logf("Sign succeeded (%d)", signStatus)

	verifyStatus, verifyBody, err := kmsRequest(ctx, endpoint, region, creds, "Verify", map[string]any{
		"KeyId":            signKeyID,
		"Message":          signMessage,
		"Signature":        signOut.Signature,
		"SigningAlgorithm": signOut.SigningAlgorithm,
	})
	if err != nil {
		exitf("Verify request failed: %v", err)
	}
	var verifyOut struct {
		SignatureValid bool `json:"SignatureValid"`
	}
	if err := expectJSONStatus("Verify", verifyStatus, verifyBody, http.StatusOK, &verifyOut); err != nil {
		exitf("Verify response validation failed: %v", err)
	}
	if !verifyOut.SignatureValid {
		exitf("Verify did not return SignatureValid=true")
	}
	logf("Verify succeeded (%d)", verifyStatus)

	macKeyID := mustCreateKey(ctx, endpoint, region, creds, map[string]any{
		"Description": "stackyard advanced mac key",
		"KeyUsage":    "GENERATE_VERIFY_MAC",
		"KeySpec":     "HMAC_256",
	}, "CreateKey(mac)")
	macMessage := base64.StdEncoding.EncodeToString([]byte("stackyard-mac-message"))
	generateMacStatus, generateMacBody, err := kmsRequest(ctx, endpoint, region, creds, "GenerateMac", map[string]any{
		"KeyId":   macKeyID,
		"Message": macMessage,
	})
	if err != nil {
		exitf("GenerateMac request failed: %v", err)
	}
	var macOut struct {
		Mac          string `json:"Mac"`
		MacAlgorithm string `json:"MacAlgorithm"`
	}
	if err := expectJSONStatus("GenerateMac", generateMacStatus, generateMacBody, http.StatusOK, &macOut); err != nil {
		exitf("GenerateMac response validation failed: %v", err)
	}
	verifyMacStatus, verifyMacBody, err := kmsRequest(ctx, endpoint, region, creds, "VerifyMac", map[string]any{
		"KeyId":        macKeyID,
		"Message":      macMessage,
		"Mac":          macOut.Mac,
		"MacAlgorithm": macOut.MacAlgorithm,
	})
	if err != nil {
		exitf("VerifyMac request failed: %v", err)
	}
	var verifyMacOut struct {
		MacValid bool `json:"MacValid"`
	}
	if err := expectJSONStatus("VerifyMac", verifyMacStatus, verifyMacBody, http.StatusOK, &verifyMacOut); err != nil {
		exitf("VerifyMac response validation failed: %v", err)
	}
	if !verifyMacOut.MacValid {
		exitf("VerifyMac did not return MacValid=true")
	}
	logf("GenerateMac/VerifyMac succeeded (%d/%d)", generateMacStatus, verifyMacStatus)

	agreementKeyID := mustCreateKey(ctx, endpoint, region, creds, map[string]any{
		"Description": "stackyard advanced agreement key",
		"KeyUsage":    "KEY_AGREEMENT",
		"KeySpec":     "ECC_NIST_P256",
	}, "CreateKey(agreement)")
	deriveStatus, deriveBody, err := kmsRequest(ctx, endpoint, region, creds, "DeriveSharedSecret", map[string]any{
		"KeyId":     agreementKeyID,
		"PublicKey": base64.StdEncoding.EncodeToString([]byte("stackyard-peer-public")),
	})
	if err != nil {
		exitf("DeriveSharedSecret request failed: %v", err)
	}
	var deriveOut struct {
		SharedSecret string `json:"SharedSecret"`
	}
	if err := expectJSONStatus("DeriveSharedSecret", deriveStatus, deriveBody, http.StatusOK, &deriveOut); err != nil {
		exitf("DeriveSharedSecret response validation failed: %v", err)
	}
	if decoded, err := base64.StdEncoding.DecodeString(deriveOut.SharedSecret); err != nil || len(decoded) == 0 {
		exitf("DeriveSharedSecret output validation failed")
	}
	logf("DeriveSharedSecret succeeded (%d)", deriveStatus)

	stage3TagStatus, stage3TagBody, err := kmsRequest(ctx, endpoint, region, creds, "TagResource", map[string]any{
		"KeyId": encryptKeyID,
		"Tags": []map[string]any{
			{"TagKey": "env", "TagValue": "advanced"},
		},
	})
	if err != nil {
		exitf("TagResource request failed: %v", err)
	}
	if err := expectJSONStatus("TagResource", stage3TagStatus, stage3TagBody, http.StatusOK, nil); err != nil {
		exitf("TagResource response validation failed: %v", err)
	}
	listTagsStatus, listTagsBody, err := kmsRequest(ctx, endpoint, region, creds, "ListResourceTags", map[string]any{
		"KeyId": encryptKeyID,
		"Limit": 10,
	})
	if err != nil {
		exitf("ListResourceTags request failed: %v", err)
	}
	var listTagsOut struct {
		Tags []struct {
			TagKey string `json:"TagKey"`
		} `json:"Tags"`
	}
	if err := expectJSONStatus("ListResourceTags", listTagsStatus, listTagsBody, http.StatusOK, &listTagsOut); err != nil {
		exitf("ListResourceTags response validation failed: %v", err)
	}
	if len(listTagsOut.Tags) == 0 {
		exitf("ListResourceTags did not return expected tag")
	}
	logf("TagResource/ListResourceTags succeeded (%d/%d)", stage3TagStatus, listTagsStatus)

	putPolicyStatus, putPolicyBody, err := kmsRequest(ctx, endpoint, region, creds, "PutKeyPolicy", map[string]any{
		"KeyId":      encryptKeyID,
		"PolicyName": "default",
		"Policy":     `{"Version":"2012-10-17","Statement":[]}`,
	})
	if err != nil {
		exitf("PutKeyPolicy request failed: %v", err)
	}
	if err := expectJSONStatus("PutKeyPolicy", putPolicyStatus, putPolicyBody, http.StatusOK, nil); err != nil {
		exitf("PutKeyPolicy response validation failed: %v", err)
	}
	getPolicyStatus, getPolicyBody, err := kmsRequest(ctx, endpoint, region, creds, "GetKeyPolicy", map[string]any{
		"KeyId":      encryptKeyID,
		"PolicyName": "default",
	})
	if err != nil {
		exitf("GetKeyPolicy request failed: %v", err)
	}
	var getPolicyOut struct {
		Policy string `json:"Policy"`
	}
	if err := expectJSONStatus("GetKeyPolicy", getPolicyStatus, getPolicyBody, http.StatusOK, &getPolicyOut); err != nil {
		exitf("GetKeyPolicy response validation failed: %v", err)
	}
	if strings.TrimSpace(getPolicyOut.Policy) == "" {
		exitf("GetKeyPolicy returned empty policy")
	}
	logf("PutKeyPolicy/GetKeyPolicy succeeded (%d/%d)", putPolicyStatus, getPolicyStatus)

	enableRotationStatus, enableRotationBody, err := kmsRequest(ctx, endpoint, region, creds, "EnableKeyRotation", map[string]any{
		"KeyId": encryptKeyID,
	})
	if err != nil {
		exitf("EnableKeyRotation request failed: %v", err)
	}
	if err := expectJSONStatus("EnableKeyRotation", enableRotationStatus, enableRotationBody, http.StatusOK, nil); err != nil {
		exitf("EnableKeyRotation response validation failed: %v", err)
	}
	rotateStatus, rotateBody, err := kmsRequest(ctx, endpoint, region, creds, "RotateKeyOnDemand", map[string]any{
		"KeyId": encryptKeyID,
	})
	if err != nil {
		exitf("RotateKeyOnDemand request failed: %v", err)
	}
	if err := expectJSONStatus("RotateKeyOnDemand", rotateStatus, rotateBody, http.StatusOK, nil); err != nil {
		exitf("RotateKeyOnDemand response validation failed: %v", err)
	}
	logf("EnableKeyRotation/RotateKeyOnDemand succeeded (%d/%d)", enableRotationStatus, rotateStatus)

	paramsStatus, paramsBody, err := kmsRequest(ctx, endpoint, region, creds, "GetParametersForImport", map[string]any{
		"KeyId":             encryptKeyID,
		"WrappingAlgorithm": "RSAES_OAEP_SHA_256",
		"WrappingKeySpec":   "RSA_2048",
	})
	if err != nil {
		exitf("GetParametersForImport request failed: %v", err)
	}
	var paramsOut struct {
		ImportToken string `json:"ImportToken"`
	}
	if err := expectJSONStatus("GetParametersForImport", paramsStatus, paramsBody, http.StatusOK, &paramsOut); err != nil {
		exitf("GetParametersForImport response validation failed: %v", err)
	}
	importStatus, importBody, err := kmsRequest(ctx, endpoint, region, creds, "ImportKeyMaterial", map[string]any{
		"KeyId":                encryptKeyID,
		"EncryptedKeyMaterial": base64.StdEncoding.EncodeToString([]byte("advanced-import-material")),
		"ImportToken":          paramsOut.ImportToken,
		"ExpirationModel":      "KEY_MATERIAL_DOES_NOT_EXPIRE",
	})
	if err != nil {
		exitf("ImportKeyMaterial request failed: %v", err)
	}
	if err := expectJSONStatus("ImportKeyMaterial", importStatus, importBody, http.StatusOK, nil); err != nil {
		exitf("ImportKeyMaterial response validation failed: %v", err)
	}
	deleteImportStatus, deleteImportBody, err := kmsRequest(ctx, endpoint, region, creds, "DeleteImportedKeyMaterial", map[string]any{
		"KeyId": encryptKeyID,
	})
	if err != nil {
		exitf("DeleteImportedKeyMaterial request failed: %v", err)
	}
	if err := expectJSONStatus("DeleteImportedKeyMaterial", deleteImportStatus, deleteImportBody, http.StatusOK, nil); err != nil {
		exitf("DeleteImportedKeyMaterial response validation failed: %v", err)
	}
	logf("GetParametersForImport/ImportKeyMaterial/DeleteImportedKeyMaterial succeeded (%d/%d/%d)", paramsStatus, importStatus, deleteImportStatus)

	customStoreStatus, customStoreBody, err := kmsRequest(ctx, endpoint, region, creds, "CreateCustomKeyStore", map[string]any{
		"CustomKeyStoreName": "stackyard-advanced-store-" + strconv.FormatInt(time.Now().Unix(), 10),
		"CloudHsmClusterId":  "cluster-advanced",
	})
	if err != nil {
		exitf("CreateCustomKeyStore request failed: %v", err)
	}
	var createStoreOut struct {
		CustomKeyStoreID string `json:"CustomKeyStoreId"`
	}
	if err := expectJSONStatus("CreateCustomKeyStore", customStoreStatus, customStoreBody, http.StatusOK, &createStoreOut); err != nil {
		exitf("CreateCustomKeyStore response validation failed: %v", err)
	}
	connectStatus, connectBody, err := kmsRequest(ctx, endpoint, region, creds, "ConnectCustomKeyStore", map[string]any{
		"CustomKeyStoreId": createStoreOut.CustomKeyStoreID,
	})
	if err != nil {
		exitf("ConnectCustomKeyStore request failed: %v", err)
	}
	if err := expectJSONStatus("ConnectCustomKeyStore", connectStatus, connectBody, http.StatusOK, nil); err != nil {
		exitf("ConnectCustomKeyStore response validation failed: %v", err)
	}
	disconnectStatus, disconnectBody, err := kmsRequest(ctx, endpoint, region, creds, "DisconnectCustomKeyStore", map[string]any{
		"CustomKeyStoreId": createStoreOut.CustomKeyStoreID,
	})
	if err != nil {
		exitf("DisconnectCustomKeyStore request failed: %v", err)
	}
	if err := expectJSONStatus("DisconnectCustomKeyStore", disconnectStatus, disconnectBody, http.StatusOK, nil); err != nil {
		exitf("DisconnectCustomKeyStore response validation failed: %v", err)
	}
	deleteStoreStatus, deleteStoreBody, err := kmsRequest(ctx, endpoint, region, creds, "DeleteCustomKeyStore", map[string]any{
		"CustomKeyStoreId": createStoreOut.CustomKeyStoreID,
	})
	if err != nil {
		exitf("DeleteCustomKeyStore request failed: %v", err)
	}
	if err := expectJSONStatus("DeleteCustomKeyStore", deleteStoreStatus, deleteStoreBody, http.StatusOK, nil); err != nil {
		exitf("DeleteCustomKeyStore response validation failed: %v", err)
	}
	logf("Create/Connect/Disconnect/DeleteCustomKeyStore succeeded (%d/%d/%d/%d)", customStoreStatus, connectStatus, disconnectStatus, deleteStoreStatus)

	fmt.Println("Done.")
}

func mustCreateKey(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	payload map[string]any,
	actionLabel string,
) string {
	status, body, err := kmsRequest(ctx, endpoint, region, creds, "CreateKey", payload)
	if err != nil {
		exitf("%s request failed: %v", actionLabel, err)
	}
	var out struct {
		KeyMetadata struct {
			KeyID string `json:"KeyId"`
		} `json:"KeyMetadata"`
	}
	if err := expectJSONStatus(actionLabel, status, body, http.StatusOK, &out); err != nil {
		exitf("%s response validation failed: %v", actionLabel, err)
	}
	if strings.TrimSpace(out.KeyMetadata.KeyID) == "" {
		exitf("%s returned empty key id", actionLabel)
	}
	logf("%s succeeded (%d) key-id=%s", actionLabel, status, out.KeyMetadata.KeyID)
	return out.KeyMetadata.KeyID
}

func kmsRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	requestURL := strings.TrimRight(endpoint, "/") + "/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "TrentService."+action)

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "kms", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, respBody, nil
}

func expectJSONStatus(action string, status int, body []byte, expectedStatus int, out any) error {
	if status != expectedStatus {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, expectedStatus, status, strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("expected JSON response for %s, got: %s", action, strings.TrimSpace(string(body)))
	}
	return nil
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
