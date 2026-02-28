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

	fmt.Printf("Stackyard IVS Real-Time Streaming advanced client using %s\n", endpoint)

	status, body, err := apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/CreateStage", map[string]any{"name": "stackyard-rt-stage"})
	mustSuccess(status, body, err, "CreateStage")
	stageArn := extractString(body, []string{"stage.arn"}, "arn:aws:ivs-realtime:us-east-1:123456789012:stage/stage-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/CreateParticipantToken", map[string]any{"stageArn": stageArn, "participantId": "participant-advanced-1", "userId": "user-advanced-1"})
	mustSuccess(status, body, err, "CreateParticipantToken")
	participantID := extractString(body, []string{"participantToken.participantId"}, "participant-advanced-1")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/CreateEncoderConfiguration", map[string]any{"name": "stackyard-encoder"})
	mustSuccess(status, body, err, "CreateEncoderConfiguration")
	encoderArn := extractString(body, []string{"encoderConfiguration.arn"}, "arn:aws:ivs-realtime:us-east-1:123456789012:encoder-configuration/encoder-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/CreateIngestConfiguration", map[string]any{"name": "stackyard-ingest"})
	mustSuccess(status, body, err, "CreateIngestConfiguration")
	ingestArn := extractString(body, []string{"ingestConfiguration.arn"}, "arn:aws:ivs-realtime:us-east-1:123456789012:ingest-configuration/ingest-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/CreateStorageConfiguration", map[string]any{"name": "stackyard-storage"})
	mustSuccess(status, body, err, "CreateStorageConfiguration")
	storageArn := extractString(body, []string{"storageConfiguration.arn"}, "arn:aws:ivs-realtime:us-east-1:123456789012:storage-configuration/storage-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/ImportPublicKey", map[string]any{"name": "stackyard-public-key", "publicKeyMaterial": "-----BEGIN PUBLIC KEY-----\nstackyard\n-----END PUBLIC KEY-----"})
	mustSuccess(status, body, err, "ImportPublicKey")
	publicKeyArn := extractString(body, []string{"publicKey.arn"}, "arn:aws:ivs-realtime:us-east-1:123456789012:public-key/pk-00000001")

	calls := []apiCall{
		{Method: http.MethodPost, Path: "/GetStage", Payload: map[string]any{"stageArn": stageArn}},
		{Method: http.MethodPost, Path: "/ListStages", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/UpdateStage", Payload: map[string]any{"stageArn": stageArn, "name": "stackyard-rt-stage-updated"}},
		{Method: http.MethodPost, Path: "/GetStageSession", Payload: map[string]any{"stageArn": stageArn, "sessionId": "session-00000001"}},
		{Method: http.MethodPost, Path: "/ListStageSessions", Payload: map[string]any{"stageArn": stageArn}},
		{Method: http.MethodPost, Path: "/GetParticipant", Payload: map[string]any{"stageArn": stageArn, "participantId": participantID}},
		{Method: http.MethodPost, Path: "/ListParticipants", Payload: map[string]any{"stageArn": stageArn}},
		{Method: http.MethodPost, Path: "/ListParticipantEvents", Payload: map[string]any{"participantId": participantID}},
		{Method: http.MethodPost, Path: "/StartParticipantReplication", Payload: map[string]any{"stageArn": stageArn, "participantId": participantID}},
		{Method: http.MethodPost, Path: "/ListParticipantReplicas", Payload: map[string]any{"participantId": participantID}},
		{Method: http.MethodPost, Path: "/StopParticipantReplication", Payload: map[string]any{"participantId": participantID}},
		{Method: http.MethodPost, Path: "/DisconnectParticipant", Payload: map[string]any{"stageArn": stageArn, "participantId": participantID}},
		{Method: http.MethodPost, Path: "/StartComposition", Payload: map[string]any{"stageArn": stageArn}},
		{Method: http.MethodPost, Path: "/ListCompositions", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/GetComposition", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/StopComposition", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/GetEncoderConfiguration", Payload: map[string]any{"arn": encoderArn}},
		{Method: http.MethodPost, Path: "/ListEncoderConfigurations", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/GetIngestConfiguration", Payload: map[string]any{"arn": ingestArn}},
		{Method: http.MethodPost, Path: "/ListIngestConfigurations", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/UpdateIngestConfiguration", Payload: map[string]any{"arn": ingestArn, "name": "stackyard-ingest-updated"}},
		{Method: http.MethodPost, Path: "/GetStorageConfiguration", Payload: map[string]any{"arn": storageArn}},
		{Method: http.MethodPost, Path: "/ListStorageConfigurations", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/GetPublicKey", Payload: map[string]any{"arn": publicKeyArn}},
		{Method: http.MethodPost, Path: "/ListPublicKeys", Payload: map[string]any{}},
	}

	for _, call := range calls {
		status, body, err = apiRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
		mustSuccess(status, body, err, call.Method+" "+call.Path)
		fmt.Printf("%s %s returned %d\n", call.Method, call.Path, status)
	}

	tagPath := "/tags/" + url.PathEscape(stageArn)
	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, tagPath, map[string]any{"tags": map[string]any{"env": "test", "team": "stackyard"}})
	mustSuccess(status, body, err, "TagResource")
	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodGet, tagPath, nil)
	mustSuccess(status, body, err, "ListTagsForResource")
	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodDelete, tagPath, map[string]any{"tagKeys": []string{"team"}})
	mustSuccess(status, body, err, "UntagResource")

	teardown := []apiCall{
		{Method: http.MethodPost, Path: "/DeleteEncoderConfiguration", Payload: map[string]any{"arn": encoderArn}},
		{Method: http.MethodPost, Path: "/DeleteIngestConfiguration", Payload: map[string]any{"arn": ingestArn}},
		{Method: http.MethodPost, Path: "/DeleteStorageConfiguration", Payload: map[string]any{"arn": storageArn}},
		{Method: http.MethodPost, Path: "/DeletePublicKey", Payload: map[string]any{"arn": publicKeyArn}},
		{Method: http.MethodPost, Path: "/DeleteStage", Payload: map[string]any{"stageArn": stageArn}},
	}
	for _, call := range teardown {
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

func extractString(body []byte, keys []string, fallback string) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fallback
	}
	for _, key := range keys {
		parts := strings.Split(key, ".")
		var cur any = payload
		ok := true
		for _, part := range parts {
			m, good := cur.(map[string]any)
			if !good {
				ok = false
				break
			}
			cur, good = m[part]
			if !good {
				ok = false
				break
			}
		}
		if ok {
			if s, good := cur.(string); good && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
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

	requestURL := strings.TrimRight(endpoint, "/") + requestPath
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "ivs-realtime", region, time.Now()); err != nil {
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
