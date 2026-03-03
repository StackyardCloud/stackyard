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

type requestCase struct {
	Action  string
	Method  string
	Path    string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	gatewayID := getenv("STACKYARD_RTB_FABRIC_GATEWAY_ID", "stackyard-gateway")
	linkID := getenv("STACKYARD_RTB_FABRIC_LINK_ID", "stackyard-link")
	resourceARN := getenv("STACKYARD_RTB_FABRIC_RESOURCE_ARN", "arn:aws:rtb-fabric:us-east-1:123456789012:requester-gateway/stackyard-gateway")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard RTB Fabric advanced client using %s\n", endpoint)

	escapedARN := url.PathEscape(resourceARN)
	requests := []requestCase{
		{
			Action: "CreateRequesterGateway",
			Method: http.MethodPost,
			Path:   "/requester-gateway",
			Payload: map[string]any{
				"name": "stackyard-requester",
			},
		},
		{
			Action: "CreateResponderGateway",
			Method: http.MethodPost,
			Path:   "/responder-gateway",
			Payload: map[string]any{
				"name": "stackyard-responder",
			},
		},
		{
			Action: "ListRequesterGateways",
			Method: http.MethodGet,
			Path:   "/requester-gateways?maxResults=10",
		},
		{
			Action: "ListResponderGateways",
			Method: http.MethodGet,
			Path:   "/responder-gateways?maxResults=10",
		},
		{
			Action: "CreateLink",
			Method: http.MethodPost,
			Path:   "/gateway/" + gatewayID + "/create-link",
			Payload: map[string]any{
				"name": "stackyard-link",
			},
		},
		{
			Action: "UpdateLink",
			Method: http.MethodPatch,
			Path:   "/gateway/" + gatewayID + "/link/" + linkID,
			Payload: map[string]any{
				"name": "stackyard-link-updated",
			},
		},
		{
			Action: "GetLink",
			Method: http.MethodGet,
			Path:   "/gateway/" + gatewayID + "/link/" + linkID,
		},
		{
			Action: "TagResource",
			Method: http.MethodPost,
			Path:   "/tags/" + escapedARN,
			Payload: map[string]any{
				"tags": map[string]string{"env": "dev", "stack": "stackyard"},
			},
		},
		{
			Action: "ListTagsForResource",
			Method: http.MethodGet,
			Path:   "/tags/" + escapedARN,
		},
		{
			Action: "DeleteLink",
			Method: http.MethodDelete,
			Path:   "/gateway/" + gatewayID + "/link/" + linkID,
		},
		{
			Action: "DeleteRequesterGateway",
			Method: http.MethodDelete,
			Path:   "/requester-gateway/" + gatewayID,
		},
		{
			Action: "DeleteResponderGateway",
			Method: http.MethodDelete,
			Path:   "/responder-gateway/" + gatewayID,
		},
	}

	for _, reqCase := range requests {
		status, body, err := rtbFabricRequest(ctx, endpoint, region, creds, reqCase.Method, reqCase.Path, reqCase.Payload)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if status <= 0 || status >= 500 {
			exitf("%s returned HTTP %d: %s", reqCase.Action, status, strings.TrimSpace(string(body)))
		}
		logf("%s returned %d", reqCase.Action, status)
	}

	fmt.Println("Done.")
}

func rtbFabricRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = encoded
	}

	url := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "rtb-fabric", region, time.Now()); err != nil {
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
