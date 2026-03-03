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
	groupID := getenv("STACKYARD_INVESTIGATION_GROUP_ID", "stackyard-investigation-group")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	calls := []call{
		{
			Name:   "CreateInvestigationGroup",
			Method: http.MethodPost,
			Path:   "/investigationGroups",
			Payload: map[string]any{
				"name":        groupID,
				"description": "stackyard investigation group",
			},
		},
		{
			Name:   "PutInvestigationGroupPolicy",
			Method: http.MethodPost,
			Path:   "/investigationGroups/" + groupID + "/policy",
			Payload: map[string]any{
				"policy": `{"Version":"2012-10-17","Statement":[]}`,
			},
		},
		{
			Name:   "GetInvestigationGroup",
			Method: http.MethodGet,
			Path:   "/investigationGroups/" + groupID,
		},
		{
			Name:   "ListInvestigationGroups",
			Method: http.MethodGet,
			Path:   "/investigationGroups",
		},
	}

	fmt.Printf("Stackyard CloudWatch Investigations advanced client using %s\n", endpoint)
	for _, c := range calls {
		status, body, err := cloudWatchInvestigationsRequest(ctx, endpoint, region, creds, c.Method, c.Path, c.Payload)
		if err != nil {
			exitf("%s failed: %v", c.Name, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", c.Name, status, strings.TrimSpace(string(body)))
		}
		fmt.Printf("%s succeeded\n", c.Name)
	}

	fmt.Println("Done.")
}

func cloudWatchInvestigationsRequest(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, method, path string, payload map[string]any) (int, []byte, error) {
	var body []byte
	if payload == nil {
		body = []byte{}
	} else {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = encoded
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(endpoint, "/")+path, bytes.NewReader(body))
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "aiops", region, time.Now()); err != nil {
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
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
