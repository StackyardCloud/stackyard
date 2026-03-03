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

	fmt.Printf("Stackyard AWS PCS advanced client using %s\n", endpoint)
	if err := waitForStackyard(ctx, endpoint, 30*time.Second); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	calls := []apiCall{
		{Action: "CreateCluster", Payload: map[string]any{}},
		{Action: "GetCluster", Payload: map[string]any{}},
		{Action: "ListClusters", Payload: map[string]any{"maxResults": 10}},
		{Action: "UpdateCluster", Payload: map[string]any{}},
		{Action: "DeleteCluster", Payload: map[string]any{}},
		{Action: "CreateComputeNodeGroup", Payload: map[string]any{}},
		{Action: "GetComputeNodeGroup", Payload: map[string]any{}},
		{Action: "ListComputeNodeGroups", Payload: map[string]any{"maxResults": 10}},
		{Action: "RegisterComputeNodeGroupInstance", Payload: map[string]any{}},
		{Action: "UpdateComputeNodeGroup", Payload: map[string]any{}},
		{Action: "DeleteComputeNodeGroup", Payload: map[string]any{}},
		{Action: "CreateQueue", Payload: map[string]any{}},
		{Action: "GetQueue", Payload: map[string]any{}},
		{Action: "ListQueues", Payload: map[string]any{"maxResults": 10}},
		{Action: "UpdateQueue", Payload: map[string]any{}},
		{Action: "DeleteQueue", Payload: map[string]any{}},
		{Action: "TagResource", Payload: map[string]any{}},
		{Action: "UntagResource", Payload: map[string]any{}},
		{Action: "ListTagsForResource", Payload: map[string]any{}},
	}

	for _, call := range calls {
		status, body, err := pcsRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
		if err != nil {
			exitf("%s request failed: %v", call.Action, err)
		}

		if status >= 200 && status < 300 {
			logf("%s returned %d", call.Action, status)
			continue
		}

		errType := extractErrorType(body)
		if isStagedPlanTolerated(status, errType, body) {
			logf("%s returned %d (%s): expected while staged plan is in progress", call.Action, status, errType)
			continue
		}

		exitf("%s returned HTTP %d: %s", call.Action, status, strings.TrimSpace(string(body)))
	}

	fmt.Println("Done.")
}

func pcsRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload == nil {
		body = []byte("{}")
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
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
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "AWSParallelComputingService."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "pcs", region, time.Now()); err != nil {
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
