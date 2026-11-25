// Command pinpoint-sms-voice runs Amazon Pinpoint SMS and Voice (v1) against
// Stackyard: create a configuration set with a CloudWatch Logs event
// destination, switch it to SNS, send a voice message with SSML, and clean
// up.
//
// The SDK has no Pinpoint SMS Voice module in this repository's go.mod, so
// requests are signed by hand.
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
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const signingName = "sms-voice"

func main() {
	c := newClient()
	fmt.Printf("Stackyard Pinpoint SMS Voice client using %s\n", c.endpoint)
	set := fmt.Sprintf("otp-calls-%d", time.Now().UnixNano())
	base := "/v1/sms-voice/configuration-sets"

	c.rest("CreateConfigurationSet", http.MethodPost, base, map[string]any{"ConfigurationSetName": set})
	c.rest("CreateConfigurationSetEventDestination", http.MethodPost, base+"/"+set+"/event-destinations", map[string]any{
		"EventDestinationName": "call-log",
		"EventDestination": map[string]any{
			"Enabled": true, "MatchingEventTypes": []string{"INITIATED_CALL", "ANSWERED", "COMPLETED_CALL", "FAILED"},
			"CloudWatchLogsDestination": map[string]any{
				"IamRoleArn":  "arn:aws:iam::123456789012:role/voice-logs",
				"LogGroupArn": "arn:aws:logs:us-east-1:123456789012:log-group:voice-calls",
			},
		},
	})
	c.rest("UpdateConfigurationSetEventDestination", http.MethodPut, base+"/"+set+"/event-destinations/call-log", map[string]any{
		"EventDestination": map[string]any{
			"Enabled": true, "MatchingEventTypes": []string{"BUSY", "NO_ANSWER", "FAILED"},
			"SnsDestination": map[string]any{"TopicArn": "arn:aws:sns:us-east-1:123456789012:voice-failures"},
		},
	})
	dests := c.rest("GetConfigurationSetEventDestinations", http.MethodGet, base+"/"+set+"/event-destinations", nil)["EventDestinations"].([]any)
	for _, item := range dests {
		d := item.(map[string]any)
		fmt.Printf("event destination %v: %v -> SNS %v\n", d["Name"], d["MatchingEventTypes"], d["SnsDestination"].(map[string]any)["TopicArn"])
	}

	msg := c.rest("SendVoiceMessage", http.MethodPost, "/v1/sms-voice/voice/message", map[string]any{
		"OriginationPhoneNumber": "+12065550100", "DestinationPhoneNumber": "+12065550199", "ConfigurationSetName": set,
		"Content": map[string]any{"SSMLMessage": map[string]any{
			"LanguageCode": "en-US", "VoiceId": "Joanna",
			"Text": `<speak>Your verification code is <say-as interpret-as="digits">482913</say-as>.</speak>`,
		}},
	})
	fmt.Printf("voice message %v queued\n", msg["MessageId"])

	sets := c.rest("ListConfigurationSets", http.MethodGet, base, nil)["ConfigurationSets"].([]any)
	fmt.Printf("%d configuration set(s)\n", len(sets))
	c.rest("DeleteConfigurationSetEventDestination", http.MethodDelete, base+"/"+set+"/event-destinations/call-log", nil)
	c.rest("DeleteConfigurationSet", http.MethodDelete, base+"/"+set, nil)
	fmt.Println("cleaned up")
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

// rest sends one REST JSON operation and exits unless it returns 2xx.
func (c client) rest(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	return decode(op, c.send(op, method, signingName, c.endpoint+path, body, map[string]string{"Content-Type": "application/json"}))
}

func decode(op string, raw []byte) map[string]any {
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, method, service, target string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	return raw
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
