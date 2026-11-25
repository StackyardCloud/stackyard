// Command chime-sdk-messaging runs Amazon Chime SDK Messaging against
// Stackyard: a moderator creates a private channel and adds members, a
// member chats (send, edit, list), the moderator redacts and bans, and a
// channel flow holds messages until its processor calls back.
//
// Calls act as an app instance user through the x-amz-chime-bearer header.
// The SDK has no Chime SDK Messaging module in this repository's go.mod, so
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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// The Chime SDK APIs all sign as chime.
const signingName = "chime"

func main() {
	c := newClient()
	fmt.Printf("Stackyard Chime SDK Messaging client using %s\n", c.endpoint)
	run := fmt.Sprint(time.Now().UnixNano())
	app := "arn:aws:chime:us-east-1:123456789012:app-instance/support-" + run
	alice, bob, troll := app+"/user/alice", app+"/user/bob", app+"/user/troll"

	arn := c.rest("CreateChannel", http.MethodPost, "/channels", alice, map[string]any{
		"AppInstanceArn": app, "Name": "incident-42", "Mode": "RESTRICTED", "Privacy": "PRIVATE",
		"ClientRequestToken": "create-" + run, "MemberArns": []string{bob, troll},
	})["ChannelArn"].(string)
	ch := "/channels/" + url.PathEscape(arn)
	fmt.Printf("alice created %s\n", arn)

	sent := c.rest("SendChannelMessage", http.MethodPost, ch+"/messages", bob, map[string]any{
		"Content": "Database failover started", "Type": "STANDARD", "Persistence": "PERSISTENT", "ClientRequestToken": "m1-" + run,
	})
	id := sent["MessageId"].(string)
	c.rest("UpdateChannelMessage", http.MethodPut, ch+"/messages/"+id, bob, map[string]any{"Content": "Database failover complete"})
	spam := c.rest("SendChannelMessage", http.MethodPost, ch+"/messages", troll, map[string]any{
		"Content": "buy now", "Type": "STANDARD", "Persistence": "PERSISTENT", "ClientRequestToken": "m2-" + run,
	})["MessageId"].(string)
	c.rest("RedactChannelMessage", http.MethodPost, ch+"/messages/"+spam+"?operation=redact", alice, nil)
	c.rest("CreateChannelBan", http.MethodPost, ch+"/bans", alice, map[string]any{"MemberArn": troll})
	msgs := c.rest("ListChannelMessages", http.MethodGet, ch+"/messages?sort-order=ASCENDING", bob, nil)["ChannelMessages"].([]any)
	for _, m := range msgs {
		m := m.(map[string]any)
		fmt.Printf("  %v: %v (redacted %v)\n", m["Sender"].(map[string]any)["Arn"], m["Content"], m["Redacted"])
	}
	members := c.rest("ListChannelMemberships", http.MethodGet, ch+"/memberships", alice, nil)["ChannelMemberships"].([]any)
	fmt.Printf("%d members after banning troll\n", len(members))
	c.rest("UpdateChannelReadMarker", http.MethodPut, ch+"/readMarker", bob, nil)

	flow := c.rest("CreateChannelFlow", http.MethodPost, "/channel-flows", "", map[string]any{
		"AppInstanceArn": app, "Name": "redact-secrets", "ClientRequestToken": "flow-" + run,
		"Processors": []any{map[string]any{"Name": "secrets", "ExecutionOrder": 1, "FallbackAction": "ABORT",
			"Configuration": map[string]any{"Lambda": map[string]any{"ResourceArn": "arn:aws:lambda:us-east-1:123456789012:function:redact", "InvocationType": "ASYNC"}}}},
	})["ChannelFlowArn"].(string)
	c.rest("AssociateChannelFlow", http.MethodPut, ch+"/channel-flow", alice, map[string]any{"ChannelFlowArn": flow})
	pending := c.rest("SendChannelMessage", http.MethodPost, ch+"/messages", bob, map[string]any{
		"Content": "password is hunter2", "Type": "STANDARD", "Persistence": "PERSISTENT", "ClientRequestToken": "m3-" + run,
	})
	pid := pending["MessageId"].(string)
	fmt.Printf("flowed message is %v\n", pending["Status"].(map[string]any)["Value"])
	c.rest("ChannelFlowCallback", http.MethodPost, ch+"?operation=channel-flow-callback", "", map[string]any{
		"CallbackId": fmt.Sprintf("callback-%032d", time.Now().UnixNano()), "ChannelMessage": map[string]any{"MessageId": pid, "Content": "password is *******"},
	})
	got := c.rest("GetChannelMessage", http.MethodGet, ch+"/messages/"+pid, bob, nil)["ChannelMessage"].(map[string]any)
	fmt.Printf("after the processor: %q (%v)\n", got["Content"], got["Status"].(map[string]any)["Value"])

	c.rest("DisassociateChannelFlow", http.MethodDelete, ch+"/channel-flow/"+url.PathEscape(flow), alice, nil)
	c.rest("DeleteChannelFlow", http.MethodDelete, "/channel-flows/"+url.PathEscape(flow), "", nil)
	c.rest("DeleteChannel", http.MethodDelete, ch, alice, nil)
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

// rest sends one REST JSON operation signed for chime, acting as bearer
// when it is set, and exits unless it returns 2xx.
func (c client) rest(op, method, path, bearer string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	headers := map[string]string{"Content-Type": "application/json"}
	if bearer != "" {
		headers["X-Amz-Chime-Bearer"] = bearer
	}
	raw := c.send(op, method, c.endpoint+path, body, headers)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request and exits unless it returns 2xx.
func (c client) send(op, method, target string, body []byte, headers map[string]string) []byte {
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
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, signingName, c.region, time.Now()); err != nil {
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
