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

	fmt.Printf("Stackyard AWS Managed Services CM advanced client using %s\n", endpoint)
	if err := waitForStackyard(ctx, endpoint, 30*time.Second); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	readCalls := []apiCall{
		{Action: "ListChangeTypeCategories", Payload: map[string]any{}},
		{Action: "ListChangeTypeSubcategories", Payload: map[string]any{"Category": "Infrastructure"}},
		{Action: "ListChangeTypeItems", Payload: map[string]any{}},
		{Action: "ListChangeTypeOperations", Payload: map[string]any{}},
		{Action: "ListChangeTypeClassificationSummaries", Payload: map[string]any{}},
		{Action: "ListChangeTypeVersionSummaries", Payload: map[string]any{}},
		{Action: "GetChangeTypeVersion", Payload: map[string]any{"ChangeTypeId": "ct-ec2-patch", "Version": "1.0"}},
	}
	for _, call := range readCalls {
		mustCall(ctx, endpoint, region, creds, call)
	}

	createRfc := mustCall(ctx, endpoint, region, creds, apiCall{
		Action:  "CreateRfc",
		Payload: map[string]any{"Title": "example-rfc", "ChangeTypeId": "ct-ec2-patch", "Version": "1.0", "ClientToken": "example-managedservicescm-create-token-000001"},
	})
	rfcID := stringValue(createRfc, "RfcId")
	if rfcID == "" {
		exitf("CreateRfc response missing RfcId: %s", mustJSON(createRfc))
	}

	lifecycleCalls := []apiCall{
		{Action: "UpdateRfc", Payload: map[string]any{"RfcId": rfcID, "Title": "example-rfc-updated", "Impact": "MEDIUM"}},
		{Action: "GetRfc", Payload: map[string]any{"RfcId": rfcID}},
		{Action: "SubmitRfc", Payload: map[string]any{"RfcId": rfcID}},
		{Action: "ApproveRfc", Payload: map[string]any{"RfcId": rfcID}},
	}
	for _, call := range lifecycleCalls {
		mustCall(ctx, endpoint, region, creds, call)
	}

	attachmentResp := mustCall(ctx, endpoint, region, creds, apiCall{
		Action:  "CreateRfcAttachment",
		Payload: map[string]any{"RfcId": rfcID, "FileName": "change-plan.txt", "Description": "advanced example attachment"},
	})
	attachmentID := stringValue(attachmentResp, "AttachmentId")
	if attachmentID == "" {
		exitf("CreateRfcAttachment response missing AttachmentId: %s", mustJSON(attachmentResp))
	}

	attachmentCalls := []apiCall{
		{Action: "GetRfcAttachment", Payload: map[string]any{"RfcId": rfcID, "AttachmentId": attachmentID}},
		{Action: "ListRfcAttachmentSummaries", Payload: map[string]any{"RfcId": rfcID}},
		{Action: "CreateRfcCorrespondence", Payload: map[string]any{"RfcId": rfcID, "Message": "please proceed"}},
		{Action: "ListRfcCorrespondences", Payload: map[string]any{"RfcId": rfcID}},
	}
	for _, call := range attachmentCalls {
		mustCall(ctx, endpoint, region, creds, call)
	}

	restrictedCalls := []apiCall{
		{Action: "ListRestrictedExecutionTimes", Payload: map[string]any{}},
		{Action: "UpdateRestrictedExecutionTimes", Payload: map[string]any{"RestrictedExecutionTimes": []any{map[string]any{"StartTime": "2026-01-01T00:00:00Z", "EndTime": "2026-01-01T01:00:00Z", "Reason": "example-window"}}}},
		{Action: "ListRestrictedExecutionTimes", Payload: map[string]any{}},
		{Action: "RejectRfc", Payload: map[string]any{"RfcId": rfcID}},
		{Action: "CancelRfc", Payload: map[string]any{"RfcId": rfcID}},
		{Action: "ListRfcSummaries", Payload: map[string]any{}},
	}
	for _, call := range restrictedCalls {
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
	status, body, err := managedServicesCMRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
	if err != nil {
		exitf("%s request failed: %v", call.Action, err)
	}
	if status < 200 || status >= 300 {
		exitf("%s returned HTTP %d: %s", call.Action, status, strings.TrimSpace(string(body)))
	}
	logf("%s returned %d", call.Action, status)

	out := map[string]any{}
	if len(bytes.TrimSpace(body)) == 0 {
		return out
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func managedServicesCMRequest(
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSManagedServicesChangeManagement."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "amscm", region, time.Now()); err != nil {
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

func stringValue(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	value, _ := raw.(string)
	return strings.TrimSpace(value)
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
