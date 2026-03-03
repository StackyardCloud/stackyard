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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type apiCall struct {
	Method  string
	Path    string
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

	fmt.Printf("Stackyard License Manager User Subscriptions advanced client using %s\n", endpoint)

	identityProviderArn := "arn:aws:license-manager-user-subscriptions:us-east-1:123456789012:identity-provider/stackyard-idp"
	resourceArn := "arn:aws:license-manager-user-subscriptions:us-east-1:123456789012:resource/stackyard-resource"

	calls := []apiCall{
		{Method: http.MethodPost, Path: "/identity-provider/RegisterIdentityProvider", Payload: map[string]any{"IdentityProviderArn": identityProviderArn, "Product": "VISUAL_STUDIO_ENTERPRISE"}},
		{Method: http.MethodPost, Path: "/identity-provider/ListIdentityProviders", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/license-server/CreateLicenseServerEndpoint", Payload: map[string]any{"ServerType": "RDS_SAL", "IdentityProviderArn": identityProviderArn}},
		{Method: http.MethodPost, Path: "/license-server/ListLicenseServerEndpoints", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/instance/ListInstances", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/user/AssociateUser", Payload: map[string]any{"Username": "jdoe", "InstanceId": "i-00000000000000001", "Product": "VISUAL_STUDIO_ENTERPRISE"}},
		{Method: http.MethodPost, Path: "/user/ListUserAssociations", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/user/StartProductSubscription", Payload: map[string]any{"Username": "jdoe", "Product": "VISUAL_STUDIO_ENTERPRISE", "InstanceId": "i-00000000000000001"}},
		{Method: http.MethodPut, Path: "/tags/" + url.PathEscape(resourceArn), Payload: map[string]any{"Tags": map[string]string{"env": "dev", "stackyard": "true"}}},
		{Method: http.MethodGet, Path: "/tags/" + url.PathEscape(resourceArn), Payload: nil},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s %s failed: %v", call.Method, call.Path, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call apiCall) error {
	status, body, err := request(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}
	if status >= 200 && status < 300 {
		logf("%s %s returned %d", call.Method, call.Path, status)
		return nil
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func request(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var bodyBytes []byte
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		bodyBytes = data
	}
	if (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch) && len(bodyBytes) == 0 {
		bodyBytes = []byte(`{}`)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		strings.TrimRight(endpoint, "/")+path,
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return 0, nil, err
	}
	if len(bodyBytes) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(bodyBytes), "license-manager-user-subscriptions", region, time.Now()); err != nil {
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
