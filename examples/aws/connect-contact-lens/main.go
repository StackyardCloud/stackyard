// Command connect-contact-lens reads Amazon Connect Contact Lens real-time
// analysis for a chat Stackyard is running: start a chat contact, exchange
// messages through the Connect Participant Service, then list the analyzed
// transcript segments with their sentiment and detected issues.
//
// Requests are signed by hand (service connect for Connect and Contact Lens).
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

func main() {
	c := newClient()
	fmt.Printf("Stackyard Contact Lens client using %s\n", c.endpoint)
	chat := c.call("connect", http.MethodPut, "/contact/chat", map[string]any{
		"InstanceId": "instance-000001", "ContactFlowId": "contact-flow-000001",
		"ParticipantDetails": map[string]any{"DisplayName": "Ann"},
		"InitialMessage":     map[string]any{"ContentType": "text/plain", "Content": "My router is broken and I'm frustrated"},
	}, "")
	contact := chat["ContactId"].(string)
	conn := c.call("execute-api", http.MethodPost, "/participant/connection",
		map[string]any{"Type": []any{"CONNECTION_CREDENTIALS"}}, chat["ParticipantToken"].(string))
	token := conn["ConnectionCredentials"].(map[string]any)["ConnectionToken"].(string)
	c.call("execute-api", http.MethodPost, "/participant/message",
		map[string]any{"ContentType": "text/plain", "Content": "Restarting it fixed it, thanks, great help!"}, token)

	out := c.call("connect", http.MethodPost, "/realtime-contact-analysis/analysis-segments",
		map[string]any{"InstanceId": "instance-000001", "ContactId": contact, "MaxResults": 10}, "")
	for _, s := range out["Segments"].([]any) {
		tr := s.(map[string]any)["Transcript"].(map[string]any)
		fmt.Printf("[%v ms] %v %-8v issues=%d %q\n", tr["BeginOffsetMillis"], tr["ParticipantRole"], tr["Sentiment"],
			len(tr["IssuesDetected"].([]any)), tr["Content"])
	}
}

// call sends a JSON request signed for service (or unsigned with a bearer
// token) and exits unless it returns 2xx.
func (c client) call(service, method, path string, payload any, bearer string) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s %s: %v", method, path, err)
		}
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("X-Amz-Bearer", bearer)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, raw)
	}
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s %s: decode: %v", method, path, err)
		}
	}
	return out
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
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
