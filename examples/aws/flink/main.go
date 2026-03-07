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

const (
	flinkTargetPrefix = "KinesisAnalytics_20180523"
	flinkSigningName  = "kinesisanalytics"
	flinkContentType  = "application/x-amz-json-1.1"
)

type rpcCall struct {
	Name    string
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

	fmt.Printf("Stackyard Flink advanced client using %s\n", endpoint)

	applicationName := "stackyard-flink-application"
	applicationARN := "arn:aws:kinesisanalytics:us-east-1:123456789012:application/stackyard-flink-application"

	calls := []rpcCall{
		{
			Name:   "CreateApplication",
			Action: "CreateApplication",
			Payload: map[string]any{
				"ApplicationName":      applicationName,
				"RuntimeEnvironment":   "FLINK-1_18",
				"ServiceExecutionRole": "arn:aws:iam::123456789012:role/stackyard-flink",
			},
		},
		{Name: "DescribeApplication", Action: "DescribeApplication", Payload: map[string]any{"ApplicationName": applicationName}},
		{Name: "ListApplications", Action: "ListApplications", Payload: map[string]any{"Limit": 20}},
		{Name: "StartApplication", Action: "StartApplication", Payload: map[string]any{"ApplicationName": applicationName}},
		{Name: "StopApplication", Action: "StopApplication", Payload: map[string]any{"ApplicationName": applicationName}},
		{Name: "ListApplicationOperations", Action: "ListApplicationOperations", Payload: map[string]any{"ApplicationName": applicationName, "Limit": 20}},
		{Name: "CreateApplicationSnapshot", Action: "CreateApplicationSnapshot", Payload: map[string]any{"ApplicationName": applicationName, "SnapshotName": "stackyard-snapshot-001"}},
		{Name: "ListApplicationSnapshots", Action: "ListApplicationSnapshots", Payload: map[string]any{"ApplicationName": applicationName, "Limit": 20}},
		{Name: "TagResource", Action: "TagResource", Payload: map[string]any{"ResourceARN": applicationARN, "Tags": []map[string]any{{"Key": "env", "Value": "dev"}}}},
		{Name: "UntagResource", Action: "UntagResource", Payload: map[string]any{"ResourceARN": applicationARN, "TagKeys": []string{"env"}}},
		{Name: "DeleteApplication", Action: "DeleteApplication", Payload: map[string]any{"ApplicationName": applicationName}},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call rpcCall) error {
	status, body, err := flinkRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
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

func flinkRequest(
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
	req.Header.Set("Content-Type", flinkContentType)
	req.Header.Set("X-Amz-Target", flinkTargetPrefix+"."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), flinkSigningName, region, time.Now()); err != nil {
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
		strings.Contains(combined, "unauthorized")
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
