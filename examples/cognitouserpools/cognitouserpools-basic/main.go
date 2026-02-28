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

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000)
	poolName := fmt.Sprintf("%s-%s", getenv("STACKYARD_USER_POOL_NAME", "stackyard-cognito-basic"), suffix)
	clientName := "stackyard-basic-client"

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Cognito User Pools basic client using %s\n", endpoint)

	createPoolOut, err := runCognitoAction(ctx, endpoint, region, creds, "CreateUserPool", map[string]any{
		"PoolName": poolName,
	})
	if err != nil {
		exitf("CreateUserPool failed: %v", err)
	}

	userPoolID, err := nestedString(createPoolOut, "UserPool", "Id")
	if err != nil {
		exitf("CreateUserPool response missing UserPool.Id: %v", err)
	}
	logf("created user pool: %s", userPoolID)

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "ListUserPools", map[string]any{
		"MaxResults": 10,
	}); err != nil {
		exitf("ListUserPools failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DescribeUserPool", map[string]any{
		"UserPoolId": userPoolID,
	}); err != nil {
		exitf("DescribeUserPool failed: %v", err)
	}

	createClientOut, err := runCognitoAction(ctx, endpoint, region, creds, "CreateUserPoolClient", map[string]any{
		"UserPoolId":     userPoolID,
		"ClientName":     clientName,
		"GenerateSecret": false,
	})
	if err != nil {
		exitf("CreateUserPoolClient failed: %v", err)
	}

	clientID, err := nestedString(createClientOut, "UserPoolClient", "ClientId")
	if err != nil {
		exitf("CreateUserPoolClient response missing UserPoolClient.ClientId: %v", err)
	}
	logf("created user pool client: %s", clientID)

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DescribeUserPoolClient", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
	}); err != nil {
		exitf("DescribeUserPoolClient failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "SignUp", map[string]any{
		"ClientId": clientID,
		"Username": "basic-user",
		"Password": "Basic#Password1",
		"UserAttributes": []map[string]string{
			{"Name": "email", "Value": "basic-user@stackyard.local"},
		},
	}); err != nil {
		exitf("SignUp failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "ConfirmSignUp", map[string]any{
		"ClientId":         clientID,
		"Username":         "basic-user",
		"ConfirmationCode": "",
	}); err != nil {
		exitf("ConfirmSignUp failed: %v", err)
	}

	initiateAuthOut, err := runCognitoAction(ctx, endpoint, region, creds, "InitiateAuth", map[string]any{
		"ClientId": clientID,
		"AuthFlow": "USER_PASSWORD_AUTH",
		"AuthParameters": map[string]string{
			"USERNAME": "basic-user",
			"PASSWORD": "Basic#Password1",
		},
	})
	if err != nil {
		exitf("InitiateAuth failed: %v", err)
	}

	accessToken, err := nestedString(initiateAuthOut, "AuthenticationResult", "AccessToken")
	if err != nil {
		exitf("InitiateAuth response missing AuthenticationResult.AccessToken: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "GetUser", map[string]any{
		"AccessToken": accessToken,
	}); err != nil {
		exitf("GetUser failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "GlobalSignOut", map[string]any{
		"AccessToken": accessToken,
	}); err != nil {
		exitf("GlobalSignOut failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DeleteUserPoolClient", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
	}); err != nil {
		exitf("DeleteUserPoolClient failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DeleteUserPool", map[string]any{
		"UserPoolId": userPoolID,
	}); err != nil {
		exitf("DeleteUserPool failed: %v", err)
	}

	fmt.Println("Done.")
}

func runCognitoAction(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (map[string]any, error) {
	status, body, err := cognitoRequest(ctx, endpoint, region, creds, action, payload)
	if err != nil {
		return nil, err
	}
	if err := expectOK(action, status, body); err != nil {
		return nil, err
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", action, err)
	}
	return out, nil
}

func cognitoRequest(
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
	req.Header.Set("X-Amz-Target", "AWSCognitoIdentityProviderService."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "cognito-idp", region, time.Now()); err != nil {
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

func nestedString(m map[string]any, keys ...string) (string, error) {
	var cur any = m
	for _, key := range keys {
		asMap, ok := cur.(map[string]any)
		if !ok {
			return "", fmt.Errorf("missing path %s", strings.Join(keys, "."))
		}
		cur, ok = asMap[key]
		if !ok {
			return "", fmt.Errorf("missing path %s", strings.Join(keys, "."))
		}
	}
	value, ok := cur.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("invalid path %s", strings.Join(keys, "."))
	}
	return value, nil
}

func expectOK(action string, status int, body []byte) error {
	if status >= 200 && status < 300 {
		logf("%s returned %d", action, status)
		return nil
	}
	return fmt.Errorf("expected %s to return 2xx, got %d: %s", action, status, strings.TrimSpace(string(body)))
}

func hashSHA256(body []byte) string {
	sum := sha256.Sum256(body)
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
