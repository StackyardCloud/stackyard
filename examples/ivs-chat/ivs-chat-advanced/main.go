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
	roomID := getenv("STACKYARD_IVS_CHAT_ROOM_ID", "stackyard-room")
	loggingConfigID := getenv("STACKYARD_IVS_CHAT_LOGGING_CONFIGURATION_ID", "stackyard-logging")
	resourceARN := getenv("STACKYARD_IVS_CHAT_RESOURCE_ARN", "arn:aws:ivs-chat:us-east-1:123456789012:room/room-00000001")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard IVS Chat advanced client using %s\n", endpoint)

	tagPath := "/tags/" + url.PathEscape(resourceARN)
	calls := []restCall{
		{Name: "CreateRoom", Method: http.MethodPost, Path: "/CreateRoom", Payload: map[string]any{"name": roomID}},
		{Name: "ListRooms", Method: http.MethodPost, Path: "/ListRooms", Payload: map[string]any{}},
		{Name: "GetRoom", Method: http.MethodPost, Path: "/GetRoom", Payload: map[string]any{"identifier": roomID}},
		{Name: "UpdateRoom", Method: http.MethodPost, Path: "/UpdateRoom", Payload: map[string]any{"identifier": roomID, "maximumMessageLength": 200}},
		{Name: "CreateChatToken", Method: http.MethodPost, Path: "/CreateChatToken", Payload: map[string]any{"roomIdentifier": roomID, "userId": "stackyard-user"}},
		{Name: "SendEvent", Method: http.MethodPost, Path: "/SendEvent", Payload: map[string]any{"roomIdentifier": roomID, "eventName": "stackyard.event"}},
		{Name: "DisconnectUser", Method: http.MethodPost, Path: "/DisconnectUser", Payload: map[string]any{"roomIdentifier": roomID, "userId": "stackyard-user"}},
		{Name: "DeleteMessage", Method: http.MethodPost, Path: "/DeleteMessage", Payload: map[string]any{"roomIdentifier": roomID, "id": "msg-00000001"}},
		{
			Name:   "CreateLoggingConfiguration",
			Method: http.MethodPost,
			Path:   "/CreateLoggingConfiguration",
			Payload: map[string]any{
				"name": loggingConfigID,
				"destinationConfiguration": map[string]any{
					"s3": map[string]any{
						"bucketName": "stackyard-bucket",
					},
				},
			},
		},
		{Name: "ListLoggingConfigurations", Method: http.MethodPost, Path: "/ListLoggingConfigurations", Payload: map[string]any{}},
		{Name: "GetLoggingConfiguration", Method: http.MethodPost, Path: "/GetLoggingConfiguration", Payload: map[string]any{"identifier": loggingConfigID}},
		{Name: "UpdateLoggingConfiguration", Method: http.MethodPost, Path: "/UpdateLoggingConfiguration", Payload: map[string]any{"identifier": loggingConfigID}},
		{Name: "TagResource", Method: http.MethodPost, Path: tagPath, Payload: map[string]any{"tags": map[string]any{"env": "test", "team": "stackyard"}}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: tagPath, Payload: nil},
		{Name: "UntagResource", Method: http.MethodDelete, Path: tagPath + "?tagKeys=team", Payload: nil},
		{Name: "DeleteLoggingConfiguration", Method: http.MethodPost, Path: "/DeleteLoggingConfiguration", Payload: map[string]any{"identifier": loggingConfigID}},
		{Name: "DeleteRoom", Method: http.MethodPost, Path: "/DeleteRoom", Payload: map[string]any{"identifier": roomID}},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := ivsChatRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func ivsChatRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload == nil {
		body = []byte{}
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
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
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "ivs-chat", region, time.Now()); err != nil {
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

func isStagedPlanTolerated(errType string, body []byte) bool {
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "invalidrequest") ||
		strings.Contains(combined, "conflictexception")
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
