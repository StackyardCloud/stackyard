// Command geo-maps runs the Amazon Location Service Maps V2 API against
// Stackyard: fetch a style descriptor, a vector tile, sprites, a glyph range
// and a static map image, and report what came back.
//
// Stackyard serves small, valid placeholder payloads. The SDK has no geo-maps
// module in this repository's go.mod, so requests are signed by hand.
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

const signingName = "geo-maps"

func main() {
	c := newClient()
	fmt.Printf("Stackyard geo-maps client using %s\n", c.endpoint)

	raw, _ := c.send(http.MethodGet, "/v2/styles/Standard/descriptor?color-scheme=Light", nil)
	style := map[string]any{}
	if err := json.Unmarshal(raw, &style); err != nil {
		exitf("style descriptor: %v", err)
	}
	fmt.Printf("style %v (version %v), %d layer(s)\n", style["name"], style["version"], len(style["layers"].([]any)))

	for _, path := range []string{
		"/v2/tiles/vector.basemap/2/1/1",
		"/v2/tiles/raster.satellite/2/1/1",
		"/v2/styles/Standard/Light/Default/sprites/sprites.png",
		"/v2/styles/Standard/Light/Default/sprites/sprites.json",
		"/v2/glyphs/Amazon%20Ember%20Regular/0-255.pbf",
		"/v2/static/map?center=-122.3493,47.6205&zoom=12&width=320&height=240",
	} {
		body, contentType := c.send(http.MethodGet, path, nil)
		fmt.Printf("%-60s %-36s %d bytes\n", path, contentType, len(body))
	}
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

// send signs a request and exits unless it returns 2xx; it returns the body
// and its Content-Type.
func (c client) send(method, path string, body []byte) ([]byte, string) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, signingName, c.region, time.Now()); err != nil {
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
	return raw, resp.Header.Get("Content-Type")
}

// post sends a JSON operation and decodes the reply.
func (c client) post(path string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", path, err)
	}
	raw, _ := c.send(http.MethodPost, path, body)
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		exitf("%s: decode: %v", path, err)
	}
	return out
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
