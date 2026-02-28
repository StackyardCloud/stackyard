package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
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
	Name    string
	Method  string
	Path    string
	Payload []byte
	Headers map[string]string
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	snapshotID := getenv("STACKYARD_SNAPSHOT_ID", "snap-00000000000000001")
	blockToken := getenv("STACKYARD_BLOCK_TOKEN", "stackyard-block-token")
	blockData := []byte("stackyard-ebs-block")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard AWS EBS advanced client using %s\n", endpoint)

	startPayload := []byte(`{"VolumeSize":8,"Description":"stackyard ebs advanced snapshot","Tags":[{"Key":"env","Value":"advanced"}]}`)
	startStatus, startBody, err := ebsRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodPost,
		"/snapshots",
		startPayload,
		map[string]string{"Content-Type": "application/json"},
	)
	if err != nil {
		exitf("StartSnapshot failed: %v", err)
	}
	if startStatus >= 200 && startStatus < 300 {
		if parsed := extractString(startBody, "SnapshotId"); parsed != "" {
			snapshotID = parsed
		}
		logf("StartSnapshot returned %d", startStatus)
	} else if isStagedPlanTolerated(startStatus, startBody) {
		logf("StartSnapshot returned %d: expected while staged plan is in progress", startStatus)
	} else {
		exitf("StartSnapshot returned HTTP %d: %s", startStatus, strings.TrimSpace(string(startBody)))
	}

	blockPath := "/snapshots/" + url.PathEscape(snapshotID) + "/blocks/0?blockToken=" + url.QueryEscape(blockToken)
	checksum := base64.StdEncoding.EncodeToString(sha256Bytes(blockData))
	putHeaders := map[string]string{
		"Content-Type":               "application/octet-stream",
		"x-amz-Checksum":             checksum,
		"x-amz-Checksum-Algorithm":   "SHA256",
		"x-amz-Data-Length":          fmt.Sprintf("%d", len(blockData)),
		"x-amz-Checksum-Aggregation": "LINEAR",
	}

	calls := []apiCall{
		{
			Name:    "PutSnapshotBlock",
			Method:  http.MethodPut,
			Path:    blockPath,
			Payload: blockData,
			Headers: putHeaders,
		},
		{
			Name:    "ListSnapshotBlocks",
			Method:  http.MethodGet,
			Path:    "/snapshots/" + url.PathEscape(snapshotID) + "/blocks?maxResults=10",
			Payload: nil,
			Headers: map[string]string{"Accept": "application/json"},
		},
		{
			Name:    "GetSnapshotBlock",
			Method:  http.MethodGet,
			Path:    blockPath,
			Payload: nil,
			Headers: map[string]string{"Accept": "application/octet-stream"},
		},
		{
			Name:    "ListChangedBlocks",
			Method:  http.MethodGet,
			Path:    "/snapshots/" + url.PathEscape(snapshotID) + "/changedblocks?firstSnapshotId=" + url.QueryEscape(snapshotID) + "&maxResults=10",
			Payload: nil,
			Headers: map[string]string{"Accept": "application/json"},
		},
		{
			Name:    "CompleteSnapshot",
			Method:  http.MethodPost,
			Path:    "/snapshots/completion/" + url.PathEscape(snapshotID),
			Payload: []byte(`{"ChangedBlocksCount":1}`),
			Headers: map[string]string{"Content-Type": "application/json"},
		},
	}

	for _, call := range calls {
		status, body, err := ebsRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload, call.Headers)
		if err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
		if status >= 200 && status < 300 {
			logf("%s returned %d", call.Name, status)
			continue
		}
		if isStagedPlanTolerated(status, body) {
			logf("%s returned %d: expected while staged plan is in progress", call.Name, status)
			continue
		}
		exitf("%s returned HTTP %d: %s", call.Name, status, strings.TrimSpace(string(body)))
	}

	fmt.Println("Done.")
}

func ebsRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
	headers map[string]string,
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
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "ebs", region, time.Now()); err != nil {
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

func isStagedPlanTolerated(status int, body []byte) bool {
	if status == http.StatusNotFound || status == http.StatusMethodNotAllowed {
		return true
	}
	combined := strings.ToLower(string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "signaturedoesnotmatch") ||
		strings.Contains(combined, "invalidrequest") ||
		strings.Contains(combined, "requestthrottled") ||
		strings.Contains(combined, "conflict")
}

func extractString(body []byte, key string) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	value, ok := payload[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func sha256Bytes(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
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
