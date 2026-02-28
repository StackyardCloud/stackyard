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
	Name    string
	Method  string
	Path    string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	graphArn := "arn:aws:detective:us-east-1:123456789012:graph:graph-00000001"
	investigationID := "investigation-00000001"

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Detective advanced client using %s\n", endpoint)

	calls := []apiCall{
		{
			Name:   "CreateMembers",
			Method: http.MethodPost,
			Path:   "/graph/members",
			Payload: map[string]any{
				"GraphArn": graphArn,
				"Accounts": []any{
					map[string]any{"AccountId": "111122223333", "EmailAddress": "member@example.com"},
				},
			},
		},
		{Name: "ListMembers", Method: http.MethodPost, Path: "/graph/members/list", Payload: map[string]any{"GraphArn": graphArn}},
		{
			Name:   "StartInvestigation",
			Method: http.MethodPost,
			Path:   "/investigations/startInvestigation",
			Payload: map[string]any{
				"GraphArn":       graphArn,
				"EntityArn":      "arn:aws:iam::123456789012:user/stackyard-user",
				"ScopeStartTime": "2024-01-01T00:00:00Z",
				"ScopeEndTime":   "2024-01-01T01:00:00Z",
			},
		},
		{Name: "GetInvestigation", Method: http.MethodPost, Path: "/investigations/getInvestigation", Payload: map[string]any{"GraphArn": graphArn, "InvestigationId": investigationID}},
		{Name: "ListIndicators", Method: http.MethodPost, Path: "/investigations/listIndicators", Payload: map[string]any{"GraphArn": graphArn, "InvestigationId": investigationID}},
		{Name: "UpdateDatasourcePackages", Method: http.MethodPost, Path: "/graph/datasources/update", Payload: map[string]any{"GraphArn": graphArn, "DatasourcePackages": []any{"DETECTIVE_CORE"}}},
		{Name: "ListDatasourcePackages", Method: http.MethodPost, Path: "/graph/datasources/list", Payload: map[string]any{"GraphArn": graphArn}},
		{Name: "TagResource", Method: http.MethodPost, Path: "/tags/" + url.PathEscape(graphArn), Payload: map[string]any{"Tags": map[string]any{"env": "advanced"}}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + url.PathEscape(graphArn), Payload: nil},
		{Name: "UntagResource", Method: http.MethodDelete, Path: "/tags/" + url.PathEscape(graphArn), Payload: map[string]any{"TagKeys": []any{"env"}}},
	}

	for _, call := range calls {
		payload := []byte{}
		var err error
		if call.Payload != nil {
			payload, err = json.Marshal(call.Payload)
			if err != nil {
				exitf("%s payload marshal failed: %v", call.Name, err)
			}
		}

		status, respBody, err := detectiveRequest(ctx, endpoint, region, creds, call.Method, call.Path, payload)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", call.Name, status, strings.TrimSpace(string(respBody)))
		}
		fmt.Printf("%s returned %d\n", call.Name, status)
	}

	fmt.Println("Done.")
}

func detectiveRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
) (int, []byte, error) {
	url := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "detective", region, time.Now()); err != nil {
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
