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

	channelGroupName := "stackyard-group"
	channelName := "stackyard-channel"
	originEndpointName := "stackyard-endpoint"
	harvestJobName := "stackyard-harvest"
	originEndpointARN := fmt.Sprintf(
		"arn:aws:mediapackagev2:us-east-1:123456789012:channelGroup/%s/channel/%s/originEndpoint/%s",
		channelGroupName,
		channelName,
		originEndpointName,
	)

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard MediaPackage advanced client using %s\n", endpoint)

	calls := []apiCall{
		{
			Method: http.MethodPost,
			Path:   "/channelGroup",
			Payload: map[string]any{
				"ChannelGroupName": channelGroupName,
				"Description":      "stackyard advanced group",
			},
		},
		{
			Method: http.MethodPost,
			Path:   "/channelGroup/" + channelGroupName + "/channel",
			Payload: map[string]any{
				"ChannelName": channelName,
				"InputType":   "HLS",
			},
		},
		{
			Method: http.MethodPost,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/originEndpoint",
			Payload: map[string]any{
				"OriginEndpointName": originEndpointName,
				"ContainerType":      "TS",
			},
		},
		{
			Method: http.MethodPost,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/originEndpoint/" + originEndpointName + "/harvestJob",
			Payload: map[string]any{
				"HarvestJobName": harvestJobName,
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/channelGroup/" + channelGroupName,
		},
		{
			Method: http.MethodGet,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName,
		},
		{
			Method: http.MethodGet,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/originEndpoint/" + originEndpointName,
		},
		{
			Method: http.MethodGet,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/originEndpoint/" + originEndpointName + "/harvestJob/" + harvestJobName,
		},
		{
			Method: http.MethodGet,
			Path:   "/channelGroup",
		},
		{
			Method: http.MethodGet,
			Path:   "/channelGroup/" + channelGroupName + "/channel",
		},
		{
			Method: http.MethodGet,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/originEndpoint",
		},
		{
			Method: http.MethodGet,
			Path:   "/channelGroup/" + channelGroupName + "/harvestJob",
		},
		{
			Method: http.MethodPut,
			Path:   "/channelGroup/" + channelGroupName,
			Payload: map[string]any{
				"Description": "updated group description",
			},
		},
		{
			Method: http.MethodPut,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/",
			Payload: map[string]any{
				"InputType": "CMAF",
			},
		},
		{
			Method: http.MethodPut,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/originEndpoint/" + originEndpointName,
			Payload: map[string]any{
				"ContainerType": "TS",
			},
		},
		{
			Method: http.MethodPut,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/policy",
			Payload: map[string]any{
				"Policy": `{"Version":"2012-10-17","Statement":[]}`,
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/policy",
		},
		{
			Method: http.MethodDelete,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/policy",
		},
		{
			Method: http.MethodPost,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/originEndpoint/" + originEndpointName + "/policy",
			Payload: map[string]any{
				"Policy": `{"Version":"2012-10-17","Statement":[]}`,
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/originEndpoint/" + originEndpointName + "/policy",
		},
		{
			Method: http.MethodDelete,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/originEndpoint/" + originEndpointName + "/policy",
		},
		{
			Method:  http.MethodPost,
			Path:    "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/reset",
			Payload: map[string]any{},
		},
		{
			Method:  http.MethodPost,
			Path:    "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/originEndpoint/" + originEndpointName + "/reset",
			Payload: map[string]any{},
		},
		{
			Method:  http.MethodPut,
			Path:    "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/originEndpoint/" + originEndpointName + "/harvestJob/" + harvestJobName,
			Payload: map[string]any{},
		},
		{
			Method: http.MethodPost,
			Path:   "/tags/" + url.PathEscape(originEndpointARN),
			Payload: map[string]any{
				"Tags": map[string]any{"env": "test", "owner": "stackyard"},
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/tags/" + url.PathEscape(originEndpointARN),
		},
		{
			Method: http.MethodDelete,
			Path:   "/tags/" + url.PathEscape(originEndpointARN) + "?tagKeys=owner",
		},
		{
			Method: http.MethodDelete,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/originEndpoint/" + originEndpointName,
		},
		{
			Method: http.MethodDelete,
			Path:   "/channelGroup/" + channelGroupName + "/channel/" + channelName + "/",
		},
		{
			Method: http.MethodDelete,
			Path:   "/channelGroup/" + channelGroupName,
		},
	}

	for _, call := range calls {
		status, body, err := apiRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "mediapackagev2", region, time.Now()); err != nil {
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
