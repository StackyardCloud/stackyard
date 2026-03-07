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

	fmt.Printf("Stackyard MediaTailor advanced client using %s\n", endpoint)

	channelName := "stackyard-channel"
	sourceLocationName := "stackyard-source-location"
	liveSourceName := "stackyard-live-source"
	vodSourceName := "stackyard-vod-source"
	playbackName := "stackyard-playback"
	programName := "stackyard-program"
	prefetchName := "stackyard-prefetch"

	calls := []apiCall{
		{Method: http.MethodPost, Path: "/channel/" + channelName, Payload: map[string]any{"Tier": "STANDARD"}},
		{Method: http.MethodPut, Path: "/channel/" + channelName + "/start", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/channel/" + channelName + "/program/" + programName, Payload: map[string]any{"AdBreaks": []any{}}},
		{Method: http.MethodGet, Path: "/channel/" + channelName + "/program/" + programName},
		{Method: http.MethodGet, Path: "/channel/" + channelName + "/schedule"},
		{Method: http.MethodPut, Path: "/configureLogs/channel", Payload: map[string]any{"ChannelName": channelName, "LogTypes": []string{"AS_RUN"}}},
		{Method: http.MethodPut, Path: "/channel/" + channelName + "/policy", Payload: map[string]any{"Policy": "{}"}},
		{Method: http.MethodGet, Path: "/channel/" + channelName + "/policy"},
		{Method: http.MethodPost, Path: "/sourceLocation/" + sourceLocationName, Payload: map[string]any{"AccessConfiguration": map[string]any{"AccessType": "S3_SIGV4"}}},
		{Method: http.MethodPost, Path: "/sourceLocation/" + sourceLocationName + "/liveSource/" + liveSourceName, Payload: map[string]any{"HttpPackageConfigurations": []any{}}},
		{Method: http.MethodPost, Path: "/sourceLocation/" + sourceLocationName + "/vodSource/" + vodSourceName, Payload: map[string]any{"HttpPackageConfigurations": []any{}}},
		{Method: http.MethodGet, Path: "/sourceLocation/" + sourceLocationName},
		{Method: http.MethodGet, Path: "/sourceLocation/" + sourceLocationName + "/liveSource/" + liveSourceName},
		{Method: http.MethodGet, Path: "/sourceLocation/" + sourceLocationName + "/vodSource/" + vodSourceName},
		{Method: http.MethodGet, Path: "/sourceLocation/" + sourceLocationName + "/liveSources"},
		{Method: http.MethodGet, Path: "/sourceLocation/" + sourceLocationName + "/vodSources"},
		{Method: http.MethodPut, Path: "/playbackConfiguration", Payload: map[string]any{"Name": playbackName}},
		{Method: http.MethodGet, Path: "/playbackConfiguration/" + playbackName},
		{Method: http.MethodPut, Path: "/configureLogs/playbackConfiguration", Payload: map[string]any{"Name": playbackName, "PercentEnabled": 100}},
		{Method: http.MethodPost, Path: "/prefetchSchedule/" + playbackName + "/" + prefetchName, Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/prefetchSchedule/" + playbackName, Payload: map[string]any{}},
		{Method: http.MethodGet, Path: "/prefetchSchedule/" + playbackName + "/" + prefetchName},
		{Method: http.MethodGet, Path: "/channels"},
		{Method: http.MethodGet, Path: "/sourceLocations"},
		{Method: http.MethodGet, Path: "/playbackConfigurations"},
		{Method: http.MethodGet, Path: "/alerts"},
	}

	for _, call := range calls {
		status, body, err := apiRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
		mustSuccess(status, body, err, call.Method+" "+call.Path)
		fmt.Printf("%s %s returned %d\n", call.Method, call.Path, status)
	}

	resourceARN := "arn:aws:mediatailor:us-east-1:123456789012:channel/" + channelName
	tagPath := "/tags/" + url.PathEscape(resourceARN)
	status, body, err := apiRequest(ctx, endpoint, region, creds, http.MethodPost, tagPath, map[string]any{"Tags": map[string]any{"env": "test", "team": "stackyard"}})
	mustSuccess(status, body, err, "TagResource")
	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodGet, tagPath, nil)
	mustSuccess(status, body, err, "ListTagsForResource")
	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodDelete, tagPath, map[string]any{"TagKeys": []string{"team"}})
	mustSuccess(status, body, err, "UntagResource")

	teardown := []apiCall{
		{Method: http.MethodDelete, Path: "/prefetchSchedule/" + playbackName + "/" + prefetchName},
		{Method: http.MethodDelete, Path: "/playbackConfiguration/" + playbackName},
		{Method: http.MethodDelete, Path: "/sourceLocation/" + sourceLocationName + "/vodSource/" + vodSourceName},
		{Method: http.MethodDelete, Path: "/sourceLocation/" + sourceLocationName + "/liveSource/" + liveSourceName},
		{Method: http.MethodDelete, Path: "/sourceLocation/" + sourceLocationName},
		{Method: http.MethodDelete, Path: "/channel/" + channelName + "/program/" + programName},
		{Method: http.MethodDelete, Path: "/channel/" + channelName + "/policy"},
		{Method: http.MethodPut, Path: "/channel/" + channelName + "/stop", Payload: map[string]any{}},
		{Method: http.MethodDelete, Path: "/channel/" + channelName},
	}

	for _, call := range teardown {
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "mediatailor", region, time.Now()); err != nil {
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
