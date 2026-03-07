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

	fmt.Printf("Stackyard IVS low latency advanced client using %s\n", endpoint)

	status, body, err := apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/CreateRecordingConfiguration", map[string]any{"name": "stackyard-recording"})
	mustSuccess(status, body, err, "CreateRecordingConfiguration")
	recordingArn := extractString(body, []string{"recordingConfiguration.arn"}, "arn:aws:ivs:us-east-1:123456789012:recording-configuration/rc-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/CreateChannel", map[string]any{"name": "stackyard-channel", "recordingConfigurationArn": recordingArn})
	mustSuccess(status, body, err, "CreateChannel")
	channelArn := extractString(body, []string{"channel.arn"}, "arn:aws:ivs:us-east-1:123456789012:channel/channel-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/CreateStreamKey", map[string]any{"channelArn": channelArn})
	mustSuccess(status, body, err, "CreateStreamKey")
	streamKeyArn := extractString(body, []string{"streamKey.arn"}, "arn:aws:ivs:us-east-1:123456789012:stream-key/sk-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/CreatePlaybackRestrictionPolicy", map[string]any{"name": "stackyard-policy", "allowedCountries": []string{"US"}})
	mustSuccess(status, body, err, "CreatePlaybackRestrictionPolicy")
	policyArn := extractString(body, []string{"playbackRestrictionPolicy.arn"}, "arn:aws:ivs:us-east-1:123456789012:playback-restriction-policy/prp-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/ImportPlaybackKeyPair", map[string]any{"name": "stackyard-keypair", "publicKeyMaterial": "-----BEGIN PUBLIC KEY-----\nstackyard\n-----END PUBLIC KEY-----"})
	mustSuccess(status, body, err, "ImportPlaybackKeyPair")
	keyPairArn := extractString(body, []string{"playbackKeyPair.arn"}, "arn:aws:ivs:us-east-1:123456789012:playback-key-pair/pk-00000001")

	calls := []apiCall{
		{Method: http.MethodPost, Path: "/GetChannel", Payload: map[string]any{"arn": channelArn}},
		{Method: http.MethodPost, Path: "/BatchGetChannel", Payload: map[string]any{"arns": []string{channelArn}}},
		{Method: http.MethodPost, Path: "/ListChannels", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/UpdateChannel", Payload: map[string]any{"arn": channelArn, "name": "stackyard-channel-updated"}},
		{Method: http.MethodPost, Path: "/GetStreamKey", Payload: map[string]any{"arn": streamKeyArn}},
		{Method: http.MethodPost, Path: "/BatchGetStreamKey", Payload: map[string]any{"arns": []string{streamKeyArn}}},
		{Method: http.MethodPost, Path: "/ListStreamKeys", Payload: map[string]any{"channelArn": channelArn}},
		{Method: http.MethodPost, Path: "/GetRecordingConfiguration", Payload: map[string]any{"arn": recordingArn}},
		{Method: http.MethodPost, Path: "/ListRecordingConfigurations", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/GetPlaybackRestrictionPolicy", Payload: map[string]any{"arn": policyArn}},
		{Method: http.MethodPost, Path: "/ListPlaybackRestrictionPolicies", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/UpdatePlaybackRestrictionPolicy", Payload: map[string]any{"arn": policyArn, "allowedCountries": []string{"US", "CA"}}},
		{Method: http.MethodPost, Path: "/GetPlaybackKeyPair", Payload: map[string]any{"arn": keyPairArn}},
		{Method: http.MethodPost, Path: "/ListPlaybackKeyPairs", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/PutMetadata", Payload: map[string]any{"channelArn": channelArn, "metadata": "stackyard-metadata"}},
		{Method: http.MethodPost, Path: "/GetStream", Payload: map[string]any{"channelArn": channelArn}},
		{Method: http.MethodPost, Path: "/ListStreams", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/GetStreamSession", Payload: map[string]any{"channelArn": channelArn}},
		{Method: http.MethodPost, Path: "/ListStreamSessions", Payload: map[string]any{"channelArn": channelArn}},
		{Method: http.MethodPost, Path: "/StartViewerSessionRevocation", Payload: map[string]any{"channelArn": channelArn, "viewerId": "viewer-1", "viewerSessionVersionsLessThanOrEqualTo": 1}},
		{Method: http.MethodPost, Path: "/BatchStartViewerSessionRevocation", Payload: map[string]any{"channelArn": channelArn, "viewerSessions": []map[string]any{{"viewerId": "viewer-1", "viewerSessionVersionsLessThanOrEqualTo": 1}}}},
	}

	for _, call := range calls {
		status, body, err = apiRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
		mustSuccess(status, body, err, call.Method+" "+call.Path)
		fmt.Printf("%s %s returned %d\n", call.Method, call.Path, status)
	}

	tagPath := "/tags/" + url.PathEscape(channelArn)
	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, tagPath, map[string]any{"tags": map[string]any{"env": "test", "team": "stackyard"}})
	mustSuccess(status, body, err, "TagResource")
	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodGet, tagPath, nil)
	mustSuccess(status, body, err, "ListTagsForResource")
	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodDelete, tagPath, map[string]any{"tagKeys": []string{"team"}})
	mustSuccess(status, body, err, "UntagResource")

	teardown := []apiCall{
		{Method: http.MethodPost, Path: "/StopStream", Payload: map[string]any{"channelArn": channelArn}},
		{Method: http.MethodPost, Path: "/DeleteStreamKey", Payload: map[string]any{"arn": streamKeyArn}},
		{Method: http.MethodPost, Path: "/DeletePlaybackKeyPair", Payload: map[string]any{"arn": keyPairArn}},
		{Method: http.MethodPost, Path: "/DeletePlaybackRestrictionPolicy", Payload: map[string]any{"arn": policyArn}},
		{Method: http.MethodPost, Path: "/DeleteRecordingConfiguration", Payload: map[string]any{"arn": recordingArn}},
		{Method: http.MethodPost, Path: "/DeleteChannel", Payload: map[string]any{"arn": channelArn}},
	}
	for _, call := range teardown {
		status, body, err = apiRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
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

func extractString(body []byte, keys []string, fallback string) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fallback
	}
	for _, key := range keys {
		parts := strings.Split(key, ".")
		var cur any = payload
		ok := true
		for _, part := range parts {
			m, good := cur.(map[string]any)
			if !good {
				ok = false
				break
			}
			cur, good = m[part]
			if !good {
				ok = false
				break
			}
		}
		if ok {
			if s, good := cur.(string); good && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return fallback
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "ivs", region, time.Now()); err != nil {
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
