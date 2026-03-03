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
	name    string
	method  string
	path    string
	payload []byte
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

	fmt.Printf("Stackyard FIS advanced client using %s\n", endpoint)
	if err := waitForStackyard(ctx, endpoint, 30*time.Second); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	templateID := createTemplate(ctx, endpoint, region, creds)
	experimentID := startExperiment(ctx, endpoint, region, creds, templateID)
	resourceARN := "arn:aws:fis:us-east-1:123456789012:experiment-template/" + templateID
	resourceEscaped := url.PathEscape(resourceARN)

	calls := []apiCall{
		{name: "GetAction", method: http.MethodGet, path: "/actions/aws%3Aec2%3Astop-instances"},
		{name: "ListActions", method: http.MethodGet, path: "/actions?maxResults=10"},
		{name: "GetTemplate", method: http.MethodGet, path: "/experimentTemplates/" + url.PathEscape(templateID)},
		{name: "UpdateTemplate", method: http.MethodPatch, path: "/experimentTemplates/" + url.PathEscape(templateID), payload: []byte(`{"description":"stage-fis-template-updated"}`)},
		{name: "CreateTargetAccountConfiguration", method: http.MethodPost, path: "/experimentTemplates/" + url.PathEscape(templateID) + "/targetAccountConfigurations", payload: []byte(`{"accountId":"111122223333","roleArn":"arn:aws:iam::111122223333:role/stackyard-fis-role"}`)},
		{name: "ListTargetAccountConfigurations", method: http.MethodGet, path: "/experimentTemplates/" + url.PathEscape(templateID) + "/targetAccountConfigurations?maxResults=10"},
		{name: "GetExperiment", method: http.MethodGet, path: "/experiments/" + url.PathEscape(experimentID)},
		{name: "ListExperimentResolvedTargets", method: http.MethodGet, path: "/experiments/" + url.PathEscape(experimentID) + "/resolvedTargets?maxResults=10"},
		{name: "ListExperimentTargetAccountConfigurations", method: http.MethodGet, path: "/experiments/" + url.PathEscape(experimentID) + "/targetAccountConfigurations?maxResults=10"},
		{name: "GetSafetyLever", method: http.MethodGet, path: "/safetyLevers/default"},
		{name: "UpdateSafetyLeverState", method: http.MethodPatch, path: "/safetyLevers/default/state", payload: []byte(`{"state":"DISABLED","reason":"advanced-example"}`)},
		{name: "TagResource", method: http.MethodPost, path: "/tags/" + resourceEscaped, payload: []byte(`{"tags":{"env":"stage","owner":"qa"}}`)},
		{name: "ListTagsForResource", method: http.MethodGet, path: "/tags/" + resourceEscaped},
		{name: "UntagResource", method: http.MethodDelete, path: "/tags/" + resourceEscaped + "?tagKeys=owner"},
		{name: "StopExperiment", method: http.MethodDelete, path: "/experiments/" + url.PathEscape(experimentID)},
		{name: "DeleteTargetAccountConfiguration", method: http.MethodDelete, path: "/experimentTemplates/" + url.PathEscape(templateID) + "/targetAccountConfigurations/111122223333"},
		{name: "DeleteExperimentTemplate", method: http.MethodDelete, path: "/experimentTemplates/" + url.PathEscape(templateID)},
	}

	for _, call := range calls {
		status, body, err := fisRequest(ctx, endpoint, region, creds, call.method, call.path, call.payload)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", call.name, status, strings.TrimSpace(string(body)))
		}
		fmt.Printf("%s returned %d\n", call.name, status)
	}

	fmt.Println("Done.")
}

func createTemplate(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider) string {
	status, body, err := fisRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodPost,
		"/experimentTemplates",
		[]byte(`{"description":"stage-fis-template","clientToken":"stage-fis-advanced-create-template-token-000001"}`),
	)
	if err != nil {
		exitf("CreateExperimentTemplate failed: %v", err)
	}
	if status < 200 || status >= 300 {
		exitf("CreateExperimentTemplate returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		exitf("decode CreateExperimentTemplate response: %v", err)
	}
	template := mapValue(payload, "experimentTemplate")
	id := stringValue(template, "id")
	if id == "" {
		exitf("CreateExperimentTemplate response missing experimentTemplate.id: %s", strings.TrimSpace(string(body)))
	}
	return id
}

func startExperiment(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, templateID string) string {
	status, body, err := fisRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodPost,
		"/experiments",
		[]byte(`{"experimentTemplateId":"`+templateID+`","clientToken":"stage-fis-advanced-start-experiment-token-000001"}`),
	)
	if err != nil {
		exitf("StartExperiment failed: %v", err)
	}
	if status < 200 || status >= 300 {
		exitf("StartExperiment returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		exitf("decode StartExperiment response: %v", err)
	}
	experiment := mapValue(payload, "experiment")
	id := stringValue(experiment, "id")
	if id == "" {
		exitf("StartExperiment response missing experiment.id: %s", strings.TrimSpace(string(body)))
	}
	return id
}

func fisRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
) (int, []byte, error) {
	if payload == nil {
		payload = []byte{}
	}
	req, err := http.NewRequestWithContext(
		ctx,
		method,
		strings.TrimRight(endpoint, "/")+path,
		bytes.NewReader(payload),
	)
	if err != nil {
		return 0, nil, err
	}
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "fis", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, body, nil
}

func mapValue(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	raw, ok := payload[key]
	if !ok {
		return map[string]any{}
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

func stringValue(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
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
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
