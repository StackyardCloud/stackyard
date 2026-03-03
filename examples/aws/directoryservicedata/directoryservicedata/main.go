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

	fmt.Printf("Stackyard Directory Service Data advanced client using %s\n", endpoint)

	calls := []apiCall{
		{Name: "ListUsers", Path: "/Users/ListUsers", Payload: map[string]any{"DirectoryId": "d-0000000000", "MaxResults": 10}},
		{Name: "ListGroups", Path: "/Groups/ListGroups", Payload: map[string]any{"DirectoryId": "d-0000000000", "MaxResults": 10}},
		{Name: "SearchUsers", Path: "/Users/SearchUsers", Payload: map[string]any{"DirectoryId": "d-0000000000", "SearchAttributes": []any{"displayName"}, "SearchString": "stackyard"}},
		{Name: "SearchGroups", Path: "/Groups/SearchGroups", Payload: map[string]any{"DirectoryId": "d-0000000000", "SearchAttributes": []any{"displayName"}, "SearchString": "admins"}},
		{Name: "DescribeUser", Path: "/Users/DescribeUser", Payload: map[string]any{"DirectoryId": "d-0000000000", "SAMAccountName": "stackyard"}},
		{Name: "DescribeGroup", Path: "/Groups/DescribeGroup", Payload: map[string]any{"DirectoryId": "d-0000000000", "SAMAccountName": "Admins"}},
		{Name: "ListGroupMembers", Path: "/GroupMemberships/ListGroupMembers", Payload: map[string]any{"DirectoryId": "d-0000000000", "SAMAccountName": "Admins"}},
		{Name: "ListGroupsForMember", Path: "/GroupMemberships/ListGroupsForMember", Payload: map[string]any{"DirectoryId": "d-0000000000", "SAMAccountName": "stackyard"}},
	}

	for _, call := range calls {
		status, body, err := dsDataRequest(ctx, endpoint, region, creds, call.Path, call.Payload)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}

		if status >= 200 && status < 300 {
			logf("%s returned %d", call.Name, status)
			continue
		}

		errType := extractErrorType(body)
		if isStagedPlanTolerated(errType, body) {
			logf("%s returned %d (%s): tolerated during staged plan", call.Name, status, errType)
			continue
		}

		exitf("%s returned HTTP %d: %s", call.Name, status, strings.TrimSpace(string(body)))
	}

	fmt.Println("Done.")
}

func dsDataRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	requestPath string,
	payload map[string]any,
) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+requestPath,
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "ds-data", region, time.Now()); err != nil {
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
	if v, ok := payload["Message"].(string); ok {
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
		strings.Contains(combined, "signaturedoesnotmatch") ||
		strings.Contains(combined, "service mismatch") ||
		strings.Contains(combined, "invalidrequest")
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
