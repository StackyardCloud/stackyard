// Command iot-secure-tunneling runs AWS IoT Secure Tunneling against
// Stackyard: open a tunnel to a device's SSH service, read it back, rotate
// the destination token, list and tag it, then close and delete it.
//
// Access tokens are returned only by OpenTunnel and RotateTunnelAccessToken;
// this example prints their lengths, never the tokens. The SDK has no Secure
// Tunneling module in this repository's go.mod, so requests are signed by hand.
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

const (
	signingName  = "IoTSecuredTunneling"
	targetPrefix = "IoTSecuredTunneling."
	contentType  = "application/x-amz-json-1.1"
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard IoT Secure Tunneling client using %s\n", c.endpoint)

	thing := fmt.Sprintf("pump-%d", time.Now().UnixNano())
	open := c.call("OpenTunnel", map[string]any{
		"description":       "maintenance SSH",
		"destinationConfig": map[string]any{"thingName": thing, "services": []any{"SSH"}},
		"timeoutConfig":     map[string]any{"maxLifetimeTimeoutMinutes": 60},
		"tags":              []any{map[string]any{"key": "team", "value": "field-ops"}},
	})
	id := open["tunnelId"].(string)
	fmt.Printf("opened tunnel %s (source token %d chars, destination token %d chars)\n",
		id, len(open["sourceAccessToken"].(string)), len(open["destinationAccessToken"].(string)))

	tunnel := c.call("DescribeTunnel", map[string]any{"tunnelId": id})["tunnel"].(map[string]any)
	fmt.Printf("tunnel %v to %v, source %v\n", tunnel["status"],
		tunnel["destinationConfig"].(map[string]any)["thingName"], tunnel["sourceConnectionState"].(map[string]any)["status"])

	rot := c.call("RotateTunnelAccessToken", map[string]any{"tunnelId": id, "clientMode": "DESTINATION"})
	fmt.Printf("rotated destination token (%d chars)\n", len(rot["destinationAccessToken"].(string)))

	c.call("TagResource", map[string]any{"resourceArn": open["tunnelArn"], "tags": []any{map[string]any{"key": "env", "value": "dev"}}})
	for _, t := range c.call("ListTagsForResource", map[string]any{"resourceArn": open["tunnelArn"]})["tags"].([]any) {
		fmt.Printf("tag %v=%v\n", t.(map[string]any)["key"], t.(map[string]any)["value"])
	}
	for _, s := range c.call("ListTunnels", map[string]any{"thingName": thing})["tunnelSummaries"].([]any) {
		fmt.Printf("listed %v: %v\n", s.(map[string]any)["tunnelId"], s.(map[string]any)["status"])
	}

	c.call("CloseTunnel", map[string]any{"tunnelId": id})
	fmt.Println("after close:", c.call("DescribeTunnel", map[string]any{"tunnelId": id})["tunnel"].(map[string]any)["status"])
	c.call("CloseTunnel", map[string]any{"tunnelId": id, "delete": true})
	fmt.Println("deleted tunnel", id)
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

// call sends one awsJson operation and exits unless it returns 2xx.
func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, http.MethodPost, signingName, c.endpoint+"/", body, map[string]string{
		"Content-Type": contentType, "X-Amz-Target": targetPrefix + op,
	})
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
