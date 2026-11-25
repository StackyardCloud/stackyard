// Command connect-participant runs a customer chat against Stackyard: start a
// chat contact with Amazon Connect, connect as the customer through the
// Connect Participant Service, send messages and a typing event, upload an
// attachment, page through the transcript and disconnect.
//
// Connect calls are signed (service connect); participant calls carry the
// participant or connection token in X-Amz-Bearer, as the service expects.
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
	fmt.Printf("Stackyard Connect Participant client using %s\n", c.endpoint)
	chat := c.call("connect", http.MethodPut, "/contact/chat", map[string]any{
		"InstanceId": "instance-000001", "ContactFlowId": "contact-flow-000001",
		"ParticipantDetails": map[string]any{"DisplayName": "Ann"},
		"InitialMessage":     map[string]any{"ContentType": "text/plain", "Content": "Hi, my order hasn't arrived"},
	}, "")
	fmt.Println("contact:", chat["ContactId"])
	p := func(path string, body any, token string) map[string]any {
		return c.call("execute-api", http.MethodPost, "/participant/"+path, body, token)
	}
	conn := p("connection", map[string]any{"Type": []any{"CONNECTION_CREDENTIALS"}, "ConnectParticipant": true}, chat["ParticipantToken"].(string))
	token := conn["ConnectionCredentials"].(map[string]any)["ConnectionToken"].(string)

	p("event", map[string]any{"ContentType": "application/vnd.amazonaws.connect.event.typing"}, token)
	p("message", map[string]any{"ContentType": "text/plain", "Content": "Order #1234, placed last week"}, token)

	receipt := []byte("order 1234")
	up := p("start-attachment-upload", map[string]any{"ContentType": "text/plain", "AttachmentSizeInBytes": len(receipt),
		"AttachmentName": "receipt.txt", "ClientToken": fmt.Sprint(time.Now().UnixNano())}, token)
	req, _ := http.NewRequest(http.MethodPut, up["UploadMetadata"].(map[string]any)["Url"].(string), bytes.NewReader(receipt))
	req.Header.Set("Content-Type", "text/plain")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		exitf("upload: %v %v", err, resp)
	}
	resp.Body.Close()
	p("complete-attachment-upload", map[string]any{"AttachmentIds": []any{up["AttachmentId"]}, "ClientToken": "done"}, token)
	fmt.Println("attachment:", p("attachment", map[string]any{"AttachmentId": up["AttachmentId"]}, token)["AttachmentSizeInBytes"], "bytes")

	var next any
	for page := 1; ; page++ {
		body := map[string]any{"ScanDirection": "FORWARD", "SortOrder": "ASCENDING", "MaxResults": 3}
		if next != nil {
			body["NextToken"] = next
		}
		t := p("transcript", body, token)
		for _, item := range t["Transcript"].([]any) {
			it := item.(map[string]any)
			fmt.Printf("page %d: %-18v %v\n", page, it["Type"], it["Content"])
		}
		if next = t["NextToken"]; next == nil {
			break
		}
	}
	p("disconnect", map[string]any{}, token)
	fmt.Println("disconnected")
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
