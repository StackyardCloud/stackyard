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
	Method  string
	Path    string
	Payload map[string]any
	Name    string
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

	fmt.Printf("Stackyard AWS Launch Wizard advanced client using %s\n", endpoint)
	if err := waitForStackyard(ctx, endpoint, 30*time.Second); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	baseReadCalls := []apiCall{
		{Name: "ListWorkloads", Method: http.MethodPost, Path: "/listWorkloads", Payload: map[string]any{}},
		{Name: "GetWorkload", Method: http.MethodPost, Path: "/getWorkload", Payload: map[string]any{"workloadName": "SAP_HANA_SINGLE"}},
		{Name: "ListWorkloadDeploymentPatterns", Method: http.MethodPost, Path: "/listWorkloadDeploymentPatterns", Payload: map[string]any{"workloadName": "SAP_HANA_SINGLE"}},
		{Name: "GetWorkloadDeploymentPattern", Method: http.MethodPost, Path: "/getWorkloadDeploymentPattern", Payload: map[string]any{"workloadName": "SAP_HANA_SINGLE", "deploymentPatternName": "single-node-hana"}},
		{Name: "ListDeploymentPatternVersions", Method: http.MethodPost, Path: "/listDeploymentPatternVersions", Payload: map[string]any{"workloadName": "SAP_HANA_SINGLE", "deploymentPatternName": "single-node-hana"}},
		{Name: "GetDeploymentPatternVersion", Method: http.MethodPost, Path: "/getDeploymentPatternVersion", Payload: map[string]any{"workloadName": "SAP_HANA_SINGLE", "deploymentPatternName": "single-node-hana", "deploymentPatternVersionName": "v1"}},
	}
	for _, call := range baseReadCalls {
		mustCall(ctx, endpoint, region, creds, call)
	}

	createResp := mustCall(ctx, endpoint, region, creds, apiCall{
		Name:   "CreateDeployment",
		Method: http.MethodPost,
		Path:   "/createDeployment",
		Payload: map[string]any{
			"name":                          "example-launchwizard-deployment",
			"workloadName":                  "SAP_HANA_SINGLE",
			"workloadDeploymentPatternName": "single-node-hana",
			"workloadVersionName":           "v1",
			"clientToken":                   "example-launchwizard-create-token-000001",
		},
	})
	deployment := mapValue(createResp, "deployment")
	deploymentID := stringValue(deployment, "deploymentId")
	deploymentARN := stringValue(deployment, "deploymentArn")
	if deploymentID == "" || deploymentARN == "" {
		exitf("CreateDeployment response missing deploymentId/deploymentArn: %s", mustJSON(createResp))
	}

	lifecycleCalls := []apiCall{
		{Name: "GetDeployment", Method: http.MethodPost, Path: "/getDeployment", Payload: map[string]any{"deploymentId": deploymentID}},
		{Name: "ListDeployments", Method: http.MethodPost, Path: "/listDeployments", Payload: map[string]any{}},
		{Name: "UpdateDeployment", Method: http.MethodPost, Path: "/updateDeployment", Payload: map[string]any{"deploymentId": deploymentID, "status": "DEPLOYED", "name": "example-launchwizard-deployment-updated"}},
		{Name: "ListDeploymentEvents", Method: http.MethodPost, Path: "/listDeploymentEvents", Payload: map[string]any{"deploymentId": deploymentID}},
		{Name: "TagResource", Method: http.MethodPost, Path: "/tags/", Payload: map[string]any{"resourceArn": deploymentARN, "tags": map[string]any{"env": "example", "owner": "qa"}}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/?resourceArn=" + deploymentARN},
		{Name: "UntagResource", Method: http.MethodDelete, Path: "/tags/?resourceArn=" + deploymentARN + "&tagKeys=owner"},
		{Name: "DeleteDeployment", Method: http.MethodPost, Path: "/deleteDeployment", Payload: map[string]any{"deploymentId": deploymentID}},
	}
	for _, call := range lifecycleCalls {
		mustCall(ctx, endpoint, region, creds, call)
	}

	fmt.Println("Done.")
}

func mustCall(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	call apiCall,
) map[string]any {
	status, body, err := launchWizardRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		exitf("%s request failed: %v", call.Name, err)
	}
	if status < 200 || status >= 300 {
		exitf("%s returned HTTP %d: %s", call.Name, status, strings.TrimSpace(string(body)))
	}
	logf("%s returned %d", call.Name, status)

	out := map[string]any{}
	if len(bytes.TrimSpace(body)) == 0 {
		return out
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func launchWizardRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload == nil || (method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch) {
		body = nil
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	var reader io.Reader = http.NoBody
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(endpoint, "/")+path, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	payloadHash := hashSHA256(body)
	if body == nil {
		payloadHash = hashSHA256([]byte{})
	}
	if err := signer.SignHTTP(ctx, credValue, req, payloadHash, "launchwizard", region, time.Now()); err != nil {
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

func mapValue(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if raw, ok := payload[key]; ok {
		if m, ok := raw.(map[string]any); ok {
			return m
		}
	}
	return map[string]any{}
}

func stringValue(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if raw, ok := payload[key]; ok {
		if s, ok := raw.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
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
