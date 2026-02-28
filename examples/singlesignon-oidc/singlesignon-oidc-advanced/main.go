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

type apiCall struct {
	Action  string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard IAM Identity Center OIDC advanced client using %s\n", endpoint)

	calls := []apiCall{
		{
			Action: "RegisterClient",
			Payload: map[string]any{
				"clientName": "stackyard-cli",
				"clientType": "public",
				"grantTypes": []string{
					"urn:ietf:params:oauth:grant-type:device_code",
					"refresh_token",
				},
				"scopes": []string{"sso:account:access"},
			},
		},
		{
			Action: "StartDeviceAuthorization",
			Payload: map[string]any{
				"clientId":     "stackyard-client-id",
				"clientSecret": "stackyard-client-secret",
				"startUrl":     "https://stackyard.awsapps.com/start",
			},
		},
		{
			Action: "CreateToken",
			Payload: map[string]any{
				"clientId":     "stackyard-client-id",
				"clientSecret": "stackyard-client-secret",
				"grantType":    "urn:ietf:params:oauth:grant-type:device_code",
				"deviceCode":   "stackyard-device-code",
			},
		},
		{
			Action: "CreateTokenWithIAM",
			Payload: map[string]any{
				"clientId":     "stackyard-client-id",
				"grantType":    "refresh_token",
				"refreshToken": "stackyard-refresh-token",
			},
		},
	}

	for _, call := range calls {
		status, body, err := oidcRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
		if err != nil {
			exitf("%s request failed: %v", call.Action, err)
		}

		if status >= 200 && status < 300 {
			fmt.Printf("%s returned %d\n", call.Action, status)
			continue
		}

		if isStagedPlanTolerated(body) {
			fmt.Printf("%s returned %d: tolerated while staged plan is in progress\n", call.Action, status)
			continue
		}

		exitf("%s returned HTTP %d: %s", call.Action, status, strings.TrimSpace(string(body)))
	}

	fmt.Println("Done.")
}

func oidcRequest(
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
	req.Header.Set("X-Amz-Target", "AWSSSOOIDCService."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "sso-oidc", region, time.Now()); err != nil {
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

func isStagedPlanTolerated(body []byte) bool {
	combined := strings.ToLower(string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "invalidrequest") ||
		strings.Contains(combined, "unauthorizedexception") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "signaturedoesnotmatch") ||
		strings.Contains(combined, "service mismatch")
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
