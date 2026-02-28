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

type call struct {
	Name    string
	Method  string
	Path    string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	workspaceID := getenv("IOTTWINMAKER_WORKSPACE_ID", "stackyard-workspace")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard IoT TwinMaker advanced client using %s\n", endpoint)

	calls := []call{
		{
			Name:    "ListWorkspaces",
			Method:  http.MethodPost,
			Path:    "/workspaces-list",
			Payload: map[string]any{"maxResults": 10},
		},
		{
			Name:   "GetPricingPlan",
			Method: http.MethodGet,
			Path:   "/pricingplan",
		},
		{
			Name:    "ListMetadataTransferJobs",
			Method:  http.MethodPost,
			Path:    "/metadata-transfer-jobs-list",
			Payload: map[string]any{"maxResults": 10},
		},
		{
			Name:    "ListEntities",
			Method:  http.MethodPost,
			Path:    "/workspaces/" + workspaceID + "/entities-list",
			Payload: map[string]any{"maxResults": 10},
		},
		{
			Name:    "ExecuteQuery",
			Method:  http.MethodPost,
			Path:    "/queries/execution",
			Payload: map[string]any{"workspaceId": workspaceID, "queryStatement": "SELECT * FROM Entity"},
		},
	}

	for _, c := range calls {
		status, body, err := iotTwinMakerRequest(ctx, endpoint, region, creds, c.Method, c.Path, c.Payload)
		if err != nil {
			exitf("%s failed: %v", c.Name, err)
		}
		if err := expectOK(c.Name, status, body); err != nil {
			exitf("%v", err)
		}
	}

	fmt.Println("Done.")
}

func iotTwinMakerRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
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

	url := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "iottwinmaker", region, time.Now()); err != nil {
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

func expectOK(action string, status int, body []byte) error {
	if status >= 200 && status < 300 {
		logf("%s returned %d", action, status)
		return nil
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("%s returned HTTP %d: %s", action, status, trimmed)
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
