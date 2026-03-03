package main

import (
	"bytes"
	"context"
	"crypto/sha256"
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

	fmt.Printf("Stackyard KMS basic client using %s\n", endpoint)

	createStatus, createBody, err := kmsRequest(ctx, endpoint, region, creds, "CreateKey", map[string]any{
		"Description": "stackyard basic key",
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"KeySpec":     "SYMMETRIC_DEFAULT",
	})
	if err != nil {
		exitf("create key request failed: %v", err)
	}
	var createOut struct {
		KeyMetadata struct {
			KeyID string `json:"KeyId"`
		} `json:"KeyMetadata"`
	}
	if err := expectJSONStatus("CreateKey", createStatus, createBody, http.StatusOK, &createOut); err != nil {
		exitf("create key response validation failed: %v", err)
	}
	if strings.TrimSpace(createOut.KeyMetadata.KeyID) == "" {
		exitf("create key response did not include a key id")
	}
	logf("CreateKey succeeded (%d) key-id=%s", createStatus, createOut.KeyMetadata.KeyID)

	aliasName := "alias/stackyard-basic-" + strconv.FormatInt(time.Now().Unix(), 10)
	createAliasStatus, createAliasBody, err := kmsRequest(ctx, endpoint, region, creds, "CreateAlias", map[string]any{
		"AliasName":   aliasName,
		"TargetKeyId": createOut.KeyMetadata.KeyID,
	})
	if err != nil {
		exitf("create alias request failed: %v", err)
	}
	if err := expectJSONStatus("CreateAlias", createAliasStatus, createAliasBody, http.StatusOK, nil); err != nil {
		exitf("create alias response validation failed: %v", err)
	}
	logf("CreateAlias succeeded (%d) alias=%s", createAliasStatus, aliasName)

	describeStatus, describeBody, err := kmsRequest(ctx, endpoint, region, creds, "DescribeKey", map[string]any{"KeyId": createOut.KeyMetadata.KeyID})
	if err != nil {
		exitf("describe key request failed: %v", err)
	}
	var describeOut struct {
		KeyMetadata struct {
			KeyID    string `json:"KeyId"`
			KeyState string `json:"KeyState"`
		} `json:"KeyMetadata"`
	}
	if err := expectJSONStatus("DescribeKey", describeStatus, describeBody, http.StatusOK, &describeOut); err != nil {
		exitf("describe key response validation failed: %v", err)
	}
	if describeOut.KeyMetadata.KeyID != createOut.KeyMetadata.KeyID {
		exitf("describe key returned unexpected key id: want=%s got=%s", createOut.KeyMetadata.KeyID, describeOut.KeyMetadata.KeyID)
	}
	logf("DescribeKey succeeded (%d) state=%s", describeStatus, describeOut.KeyMetadata.KeyState)

	listKeysStatus, listKeysBody, err := kmsRequest(ctx, endpoint, region, creds, "ListKeys", map[string]any{"Limit": 10})
	if err != nil {
		exitf("list keys request failed: %v", err)
	}
	var listKeysOut struct {
		Keys []struct {
			KeyID string `json:"KeyId"`
		} `json:"Keys"`
	}
	if err := expectJSONStatus("ListKeys", listKeysStatus, listKeysBody, http.StatusOK, &listKeysOut); err != nil {
		exitf("list keys response validation failed: %v", err)
	}
	logf("ListKeys succeeded (%d) keys=%d", listKeysStatus, len(listKeysOut.Keys))

	listAliasesStatus, listAliasesBody, err := kmsRequest(ctx, endpoint, region, creds, "ListAliases", map[string]any{
		"KeyId": createOut.KeyMetadata.KeyID,
		"Limit": 10,
	})
	if err != nil {
		exitf("list aliases request failed: %v", err)
	}
	var listAliasesOut struct {
		Aliases []struct {
			AliasName string `json:"AliasName"`
		} `json:"Aliases"`
	}
	if err := expectJSONStatus("ListAliases", listAliasesStatus, listAliasesBody, http.StatusOK, &listAliasesOut); err != nil {
		exitf("list aliases response validation failed: %v", err)
	}
	if len(listAliasesOut.Aliases) == 0 {
		exitf("list aliases response did not include expected alias")
	}
	logf("ListAliases succeeded (%d) aliases=%d", listAliasesStatus, len(listAliasesOut.Aliases))

	fmt.Println("Done.")
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
		return fmt.Errorf("expected %s response to be JSON, got: %s", action, strings.TrimSpace(string(body)))
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
