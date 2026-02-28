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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const (
	instanceArn = "arn:aws:sso:::instance/ssoins-0000000000000000"
	principalID = "11111111-2222-3333-4444-555555555555"
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

	fmt.Printf("Stackyard IAM Identity Center advanced client using %s\n", endpoint)

	_, createPS, err := mustCall(ctx, endpoint, region, creds, "CreatePermissionSet", map[string]any{
		"InstanceArn": instanceArn,
		"Name":        "stackyard-permission-set",
	})
	if err != nil {
		exitf("CreatePermissionSet failed: %v", err)
	}
	permissionSetArn := readString(createPS, "PermissionSet", "PermissionSetArn", instanceArn+"/ps-0000000000000001")

	_, createApp, err := mustCall(ctx, endpoint, region, creds, "CreateApplication", map[string]any{
		"ApplicationProviderArn": "arn:aws:sso::aws:applicationProvider/custom",
		"Name":                   "stackyard-application",
		"InstanceArn":            instanceArn,
	})
	if err != nil {
		exitf("CreateApplication failed: %v", err)
	}
	applicationArn := readString(createApp, "ApplicationArn", "", "arn:aws:sso::123456789012:application/ssoins-0000000000000000/apl-0000000000000001")

	calls := []struct {
		action  string
		payload map[string]any
	}{
		{action: "ListInstances", payload: map[string]any{}},
		{action: "ListPermissionSets", payload: map[string]any{"InstanceArn": instanceArn}},
		{action: "DescribePermissionSet", payload: map[string]any{"InstanceArn": instanceArn, "PermissionSetArn": permissionSetArn}},
		{action: "CreateAccountAssignment", payload: map[string]any{
			"InstanceArn":      instanceArn,
			"PermissionSetArn": permissionSetArn,
			"PrincipalId":      principalID,
			"PrincipalType":    "USER",
			"TargetId":         "123456789012",
			"TargetType":       "AWS_ACCOUNT",
		}},
		{action: "ListAccountAssignments", payload: map[string]any{"InstanceArn": instanceArn, "PermissionSetArn": permissionSetArn, "AccountId": "123456789012"}},
		{action: "ListApplications", payload: map[string]any{"InstanceArn": instanceArn}},
		{action: "PutApplicationSessionConfiguration", payload: map[string]any{"ApplicationArn": applicationArn, "SessionDuration": "PT1H"}},
		{action: "CreateTrustedTokenIssuer", payload: map[string]any{
			"InstanceArn":            instanceArn,
			"Name":                   "stackyard-tti",
			"TrustedTokenIssuerType": "OIDC_JWT",
		}},
		{action: "ListTrustedTokenIssuers", payload: map[string]any{"InstanceArn": instanceArn}},
		{action: "TagResource", payload: map[string]any{"ResourceArn": permissionSetArn, "Tags": []map[string]string{{"Key": "env", "Value": "coverage"}}}},
		{action: "ListTagsForResource", payload: map[string]any{"ResourceArn": permissionSetArn}},
	}

	for _, call := range calls {
		if _, _, err := mustCall(ctx, endpoint, region, creds, call.action, call.payload); err != nil {
			exitf("%s failed: %v", call.action, err)
		}
	}

	fmt.Println("Done.")
}

func mustCall(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, map[string]any, error) {
	status, body, err := singleSignOnRequest(ctx, endpoint, region, creds, action, payload)
	if err != nil {
		return 0, nil, err
	}
	if status < 200 || status >= 300 {
		return status, nil, fmt.Errorf("returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	decoded := map[string]any{}
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &decoded); err != nil {
			return status, nil, fmt.Errorf("invalid JSON response: %w", err)
		}
	}
	fmt.Printf("%s returned %d\n", action, status)
	return status, decoded, nil
}

func readString(payload map[string]any, key string, nestedKey string, fallback string) string {
	if nestedKey == "" {
		if v, ok := payload[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		return fallback
	}
	outer, ok := payload[key].(map[string]any)
	if !ok {
		return fallback
	}
	if v, ok := outer[nestedKey].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func singleSignOnRequest(
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

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "SWBExternalService."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "sso", region, time.Now()); err != nil {
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
