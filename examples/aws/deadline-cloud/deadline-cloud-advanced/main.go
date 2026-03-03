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

	fmt.Printf("Stackyard Deadline Cloud advanced client using %s\n", endpoint)

	status, body, err := apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/2023-10-12/farms", map[string]any{
		"displayName": "stackyard-farm",
		"description": "stackyard deadline farm",
	})
	mustSuccess(status, body, err, "CreateFarm")
	farmID := extractString(body, "farmId", "farm-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/2023-10-12/farms/"+farmID+"/fleets", map[string]any{
		"displayName": "stackyard-fleet",
	})
	mustSuccess(status, body, err, "CreateFleet")
	fleetID := extractString(body, "fleetId", "fleet-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/2023-10-12/farms/"+farmID+"/queues", map[string]any{
		"displayName": "stackyard-queue",
	})
	mustSuccess(status, body, err, "CreateQueue")
	queueID := extractString(body, "queueId", "queue-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/2023-10-12/farms/"+farmID+"/fleets/"+fleetID+"/workers", map[string]any{})
	mustSuccess(status, body, err, "CreateWorker")
	workerID := extractString(body, "workerId", "worker-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/2023-10-12/farms/"+farmID+"/queues/"+queueID+"/jobs", map[string]any{})
	mustSuccess(status, body, err, "CreateJob")
	jobID := extractString(body, "jobId", "job-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/2023-10-12/farms/"+farmID+"/queues/"+queueID+"/environments", map[string]any{})
	mustSuccess(status, body, err, "CreateQueueEnvironment")
	queueEnvID := extractString(body, "queueEnvironmentId", "qenv-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/2023-10-12/farms/"+farmID+"/limits", map[string]any{})
	mustSuccess(status, body, err, "CreateLimit")
	limitID := extractString(body, "limitId", "limit-00000001")

	calls := []apiCall{
		{Method: http.MethodPut, Path: "/2023-10-12/farms/" + farmID + "/queue-fleet-associations", Payload: map[string]any{"queueId": queueID, "fleetId": fleetID}},
		{Method: http.MethodPut, Path: "/2023-10-12/farms/" + farmID + "/queue-limit-associations", Payload: map[string]any{"queueId": queueID, "limitId": limitID}},
		{Method: http.MethodGet, Path: "/2023-10-12/farms/" + farmID},
		{Method: http.MethodGet, Path: "/2023-10-12/farms/" + farmID + "/fleets/" + fleetID},
		{Method: http.MethodGet, Path: "/2023-10-12/farms/" + farmID + "/queues/" + queueID},
		{Method: http.MethodGet, Path: "/2023-10-12/farms/" + farmID + "/queues/" + queueID + "/jobs/" + jobID},
		{Method: http.MethodGet, Path: "/2023-10-12/farms/" + farmID + "/limits"},
		{Method: http.MethodGet, Path: "/2023-10-12/farms/" + farmID + "/queue-fleet-associations"},
		{Method: http.MethodGet, Path: "/2023-10-12/farms/" + farmID + "/queue-limit-associations"},
		{Method: http.MethodPost, Path: "/2023-10-12/farms/" + farmID + "/search/jobs", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/2023-10-12/farms/" + farmID + "/search/steps", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/2023-10-12/farms/" + farmID + "/search/tasks", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/2023-10-12/farms/" + farmID + "/search/workers", Payload: map[string]any{}},
		{Method: http.MethodPatch, Path: "/2023-10-12/farms/" + farmID + "/queues/" + queueID, Payload: map[string]any{"displayName": "stackyard-queue-updated"}},
		{Method: http.MethodPatch, Path: "/2023-10-12/farms/" + farmID + "/fleets/" + fleetID + "/workers/" + workerID, Payload: map[string]any{"status": "ACTIVE"}},
		{Method: http.MethodDelete, Path: "/2023-10-12/farms/" + farmID + "/queue-limit-associations/" + queueID + "/" + limitID},
		{Method: http.MethodDelete, Path: "/2023-10-12/farms/" + farmID + "/queue-fleet-associations/" + queueID + "/" + fleetID},
		{Method: http.MethodDelete, Path: "/2023-10-12/farms/" + farmID + "/queues/" + queueID + "/environments/" + queueEnvID},
		{Method: http.MethodDelete, Path: "/2023-10-12/farms/" + farmID + "/fleets/" + fleetID + "/workers/" + workerID},
		{Method: http.MethodDelete, Path: "/2023-10-12/farms/" + farmID + "/queues/" + queueID},
		{Method: http.MethodDelete, Path: "/2023-10-12/farms/" + farmID + "/fleets/" + fleetID},
		{Method: http.MethodDelete, Path: "/2023-10-12/farms/" + farmID + "/limits/" + limitID},
		{Method: http.MethodDelete, Path: "/2023-10-12/farms/" + farmID},
	}

	farmARN := "arn:aws:deadline:us-east-1:123456789012:farm/" + farmID
	tagPath := "/2023-10-12/tags/" + url.PathEscape(farmARN)

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, tagPath, map[string]any{
		"tags": map[string]any{"env": "test", "team": "stackyard"},
	})
	mustSuccess(status, body, err, "TagResource")
	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodGet, tagPath, nil)
	mustSuccess(status, body, err, "ListTagsForResource")
	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodDelete, tagPath+"?tagKeys=team", nil)
	mustSuccess(status, body, err, "UntagResource")

	for _, call := range calls {
		status, body, err = apiRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
		mustSuccess(status, body, err, call.Method+" "+call.Path)
		fmt.Printf("%s %s returned %d\n", call.Method, call.Path, status)
	}

	fmt.Println("Done.")
}

func mustSuccess(status int, body []byte, err error, action string) {
	if err != nil {
		exitf("%s request failed: %v", action, err)
	}
	if status < 200 || status >= 300 {
		exitf("%s returned HTTP %d: %s", action, status, strings.TrimSpace(string(body)))
	}
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "deadline", region, time.Now()); err != nil {
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
