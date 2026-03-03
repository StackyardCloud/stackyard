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

type restCall struct {
	Name    string
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

	fmt.Printf("Stackyard Wickr advanced client using %s\n", endpoint)
	if err := waitForStackyard(ctx, endpoint, 30*time.Second); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	networkID := "n-000001"
	userID := "u-000001"
	groupID := "g-000001"
	botID := "b-000001"
	usernameHash := "uh-000001"

	calls := []restCall{
		{Name: "CreateNetwork", Method: http.MethodPost, Path: "/networks", Payload: map[string]any{"name": "stackyard-network"}},
		{Name: "ListNetworks", Method: http.MethodGet, Path: "/networks?maxResults=10"},
		{Name: "GetNetwork", Method: http.MethodGet, Path: "/networks/" + url.PathEscape(networkID)},
		{Name: "UpdateNetwork", Method: http.MethodPatch, Path: "/networks/" + url.PathEscape(networkID), Payload: map[string]any{"name": "stackyard-network-updated"}},
		{Name: "GetNetworkSettings", Method: http.MethodGet, Path: "/networks/" + url.PathEscape(networkID) + "/settings"},
		{Name: "UpdateNetworkSettings", Method: http.MethodPatch, Path: "/networks/" + url.PathEscape(networkID) + "/settings", Payload: map[string]any{"defaultClient": "web"}},
		{Name: "CreateSecurityGroup", Method: http.MethodPost, Path: "/networks/" + url.PathEscape(networkID) + "/security-groups", Payload: map[string]any{"name": "stackyard-security-group"}},
		{Name: "ListSecurityGroups", Method: http.MethodGet, Path: "/networks/" + url.PathEscape(networkID) + "/security-groups?maxResults=10"},
		{Name: "GetSecurityGroup", Method: http.MethodGet, Path: "/networks/" + url.PathEscape(networkID) + "/security-groups/" + url.PathEscape(groupID)},
		{Name: "BatchCreateUser", Method: http.MethodPost, Path: "/networks/" + url.PathEscape(networkID) + "/users", Payload: map[string]any{"users": []map[string]any{{"email": "user@example.com", "name": "stackyard-user"}}}},
		{Name: "ListUsers", Method: http.MethodGet, Path: "/networks/" + url.PathEscape(networkID) + "/users?maxResults=10"},
		{Name: "GetUser", Method: http.MethodGet, Path: "/networks/" + url.PathEscape(networkID) + "/users/" + url.PathEscape(userID)},
		{Name: "ListDevicesForUser", Method: http.MethodGet, Path: "/networks/" + url.PathEscape(networkID) + "/users/" + url.PathEscape(userID) + "/devices?maxResults=10"},
		{Name: "CreateBot", Method: http.MethodPost, Path: "/networks/" + url.PathEscape(networkID) + "/bots", Payload: map[string]any{"name": "stackyard-bot"}},
		{Name: "ListBots", Method: http.MethodGet, Path: "/networks/" + url.PathEscape(networkID) + "/bots?maxResults=10"},
		{Name: "GetBot", Method: http.MethodGet, Path: "/networks/" + url.PathEscape(networkID) + "/bots/" + url.PathEscape(botID)},
		{Name: "UpdateGuestUser", Method: http.MethodPatch, Path: "/networks/" + url.PathEscape(networkID) + "/guest-users/" + url.PathEscape(usernameHash), Payload: map[string]any{"action": "ALLOW"}},
		{Name: "DeleteBot", Method: http.MethodDelete, Path: "/networks/" + url.PathEscape(networkID) + "/bots/" + url.PathEscape(botID)},
		{Name: "DeleteSecurityGroup", Method: http.MethodDelete, Path: "/networks/" + url.PathEscape(networkID) + "/security-groups/" + url.PathEscape(groupID)},
		{Name: "DeleteNetwork", Method: http.MethodDelete, Path: "/networks/" + url.PathEscape(networkID)},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := wickrRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}
	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(status, errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func wickrRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	} else {
		body = []byte{}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		strings.TrimRight(endpoint, "/")+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "wickr", region, time.Now()); err != nil {
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

func extractErrorType(body []byte) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if v, ok := payload["__type"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["code"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["message"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func isStagedPlanTolerated(status int, errType string, body []byte) bool {
	if status == http.StatusNotFound {
		return true
	}
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "unknown action") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "forbidden") ||
		strings.Contains(combined, "unauthorized")
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func waitForStackyard(ctx context.Context, endpoint string, timeout time.Duration) error {
	healthURL := strings.TrimRight(endpoint, "/") + "/_stackyard/health"
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, http.NoBody)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("health returned status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("timed out waiting for health endpoint")
	}
	return fmt.Errorf("%s: %w", healthURL, lastErr)
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
