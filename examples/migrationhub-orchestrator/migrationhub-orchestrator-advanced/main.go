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

	fmt.Printf("Stackyard Migration Hub Orchestrator advanced client using %s\n", endpoint)

	status, body, err := apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/template", map[string]any{
		"name":        "stackyard-template",
		"description": "stackyard orchestrator template",
	})
	if err != nil {
		exitf("CreateTemplate request failed: %v", err)
	}
	if status < 200 || status >= 300 {
		exitf("CreateTemplate returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	templateID := extractString(body, "id", "tmpl-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/migrationworkflow/", map[string]any{
		"name":       "stackyard-workflow",
		"templateId": templateID,
	})
	if err != nil {
		exitf("CreateWorkflow request failed: %v", err)
	}
	if status < 200 || status >= 300 {
		exitf("CreateWorkflow returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	workflowID := extractString(body, "id", "mwf-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/workflowstepgroups", map[string]any{
		"workflowId": workflowID,
		"name":       "stackyard-group",
	})
	if err != nil {
		exitf("CreateWorkflowStepGroup request failed: %v", err)
	}
	if status < 200 || status >= 300 {
		exitf("CreateWorkflowStepGroup returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	stepGroupID := extractString(body, "id", "wsg-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/workflowstep", map[string]any{
		"workflowId":  workflowID,
		"stepGroupId": stepGroupID,
		"name":        "stackyard-step",
	})
	if err != nil {
		exitf("CreateWorkflowStep request failed: %v", err)
	}
	if status < 200 || status >= 300 {
		exitf("CreateWorkflowStep returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	stepID := extractString(body, "id", "wstep-00000001")

	workflowARN := "arn:aws:migrationhub-orchestrator:us-east-1:123456789012:workflow/" + workflowID
	resourcePath := "/tags/" + url.PathEscape(workflowARN)

	calls := []apiCall{
		{Method: http.MethodGet, Path: "/migrationworkflowtemplate/" + templateID},
		{Method: http.MethodGet, Path: "/templatestepgroups/" + templateID},
		{Method: http.MethodGet, Path: "/templatesteps?templateId=" + url.QueryEscape(templateID)},
		{Method: http.MethodGet, Path: "/migrationworkflow/" + workflowID},
		{Method: http.MethodGet, Path: "/workflowstepgroup/" + stepGroupID},
		{Method: http.MethodGet, Path: "/workflowstep/" + stepID},
		{Method: http.MethodGet, Path: "/workflow/" + workflowID + "/workflowstepgroups/" + stepGroupID + "/workflowsteps"},
		{Method: http.MethodPost, Path: "/retryworkflowstep/" + stepID, Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/migrationworkflow/" + workflowID + "/start", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/migrationworkflow/" + workflowID + "/stop", Payload: map[string]any{}},
		{Method: http.MethodGet, Path: "/plugins"},
		{Method: http.MethodPost, Path: resourcePath, Payload: map[string]any{"tags": map[string]any{"env": "test", "team": "stackyard"}}},
		{Method: http.MethodGet, Path: resourcePath},
		{Method: http.MethodDelete, Path: resourcePath + "?tagKeys=team"},
		{Method: http.MethodDelete, Path: "/workflowstep/" + stepID},
		{Method: http.MethodDelete, Path: "/workflowstepgroup/" + stepGroupID},
		{Method: http.MethodDelete, Path: "/migrationworkflow/" + workflowID},
		{Method: http.MethodDelete, Path: "/template/" + templateID},
	}

	for _, call := range calls {
		status, body, err = apiRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
		if err != nil {
			exitf("%s %s request failed: %v", call.Method, call.Path, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s %s returned HTTP %d: %s", call.Method, call.Path, status, strings.TrimSpace(string(body)))
		}
		fmt.Printf("%s %s returned %d\n", call.Method, call.Path, status)
	}

	fmt.Println("Done.")
}

func extractString(body []byte, key, fallback string) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fallback
	}
	if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func apiRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, requestPath string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	url := strings.TrimRight(endpoint, "/") + requestPath
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "migrationhub-orchestrator", region, time.Now()); err != nil {
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
