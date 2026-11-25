// Command iot-data runs the AWS IoT Data Plane against Stackyard: update and
// read a device shadow (desired vs reported, with the delta), keep a named
// shadow, connect a simulated MQTT client, publish a retained message and a
// live one, read what the client received, send it a direct message, then
// disconnect it and delete the shadow.
//
// Stackyard runs no MQTT broker; clients are simulated through
// POST /_stackyard/iot-data/connections. The SDK has no IoT Data module in
// this repository's go.mod, so requests are signed by hand.
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

const signingName = "iotdata"

func main() {
	c := newClient()
	fmt.Printf("Stackyard IoT Data client using %s\n", c.endpoint)
	run := time.Now().UnixNano()
	thing := fmt.Sprintf("pump-%d", run)
	shadow := "/things/" + thing + "/shadow"

	up := c.call("UpdateThingShadow", signingName, http.MethodPost, shadow, mustJSON(map[string]any{
		"state": map[string]any{"desired": map[string]any{"speed": 1200}, "reported": map[string]any{"speed": 900}},
	}), nil)
	fmt.Println("shadow version", up["version"])
	got := c.call("GetThingShadow", signingName, http.MethodGet, shadow, nil, nil)
	fmt.Println("delta:", got["state"].(map[string]any)["delta"])
	c.call("UpdateThingShadow", signingName, http.MethodPost, shadow+"?name=firmware", mustJSON(map[string]any{
		"state": map[string]any{"reported": map[string]any{"version": "1.4.2"}},
	}), nil)
	names := c.call("ListNamedShadowsForThing", signingName, http.MethodGet, "/api/things/shadow/ListNamedShadowsForThing/"+thing, nil, nil)
	fmt.Println("named shadows:", names["results"])

	clientID := fmt.Sprintf("dev-%d", run)
	topic := fmt.Sprintf("plant/%d/speed", run)
	c.call("Publish", signingName, http.MethodPost, "/topics/"+escape(topic)+"?qos=1&retain=true", []byte(`{"rpm":900}`), nil)
	connect(c, clientID, topic)
	c.call("Publish", signingName, http.MethodPost, "/topics/"+escape(topic)+"?qos=1", []byte(`{"rpm":950}`), nil)
	retained := c.call("GetRetainedMessage", signingName, http.MethodGet, "/retainedMessage/"+escape(topic), nil, nil)
	fmt.Printf("retained on %v (qos %v)\n", retained["topic"], retained["qos"])
	conn := c.call("GetConnection", signingName, http.MethodGet, "/connections/"+clientID, nil, nil)
	fmt.Printf("client %v connected=%v\n", conn["clientId"], conn["connected"])
	subs := c.call("ListSubscriptions", signingName, http.MethodGet, "/connections/"+clientID+"/subscriptions", nil, nil)
	fmt.Println("subscriptions:", len(subs["subscriptions"].([]any)))
	c.call("SendDirectMessage", signingName, http.MethodPost, "/connections/"+clientID+"/messages?topic=cmd%2Fping", []byte("ping"), nil)
	for _, m := range inbox(c, clientID) {
		fmt.Println("client received on", m.(map[string]any)["topic"])
	}

	c.call("DeleteConnection", signingName, http.MethodDelete, "/connections/"+clientID+"?cleanSession=true", nil, nil)
	del := c.call("DeleteThingShadow", signingName, http.MethodDelete, shadow, nil, nil)
	fmt.Println("deleted shadow at version", del["version"])
}

func escape(topic string) string { return strings.ReplaceAll(topic, "/", "%2F") }

// connect opens a simulated MQTT client subscribed to topic.
func connect(c client, id, topic string) {
	body := mustJSON(map[string]any{"clientId": id, "cleanSession": false,
		"subscriptions": []any{map[string]any{"topicFilter": topic, "qos": 1}}})
	resp, err := http.Post(c.endpoint+"/_stackyard/iot-data/connections", "application/json", bytes.NewReader(body))
	if err != nil {
		exitf("connect: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		exitf("connect: HTTP %d: %s", resp.StatusCode, raw)
	}
}

// inbox drains the messages delivered to a simulated client.
func inbox(c client, id string) []any {
	resp, err := http.Get(c.endpoint + "/_stackyard/iot-data/connections/" + id + "/messages")
	if err != nil {
		exitf("inbox: %v", err)
	}
	defer resp.Body.Close()
	var out struct {
		Messages []any `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		exitf("inbox: %v", err)
	}
	return out.Messages
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

// call sends one signed request and exits unless it returns 2xx; a JSON
// reply is decoded.
func (c client) call(op, service, method, path string, body []byte, headers map[string]string) map[string]any {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
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
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		exitf("marshal: %v", err)
	}
	return b
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
