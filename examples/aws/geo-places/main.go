// Command geo-places runs the Amazon Location Service Places V2 API against
// Stackyard: geocode a landmark, look it up by id, reverse-geocode its
// position, search nearby and by text, and autocomplete a city name.
//
// Stackyard answers from a small built-in gazetteer. The SDK has no
// geo-places module in this repository's go.mod, so requests are signed by
// hand.
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

const signingName = "geo-places"

func main() {
	c := newClient()
	fmt.Printf("Stackyard geo-places client using %s\n", c.endpoint)

	hit := first(c.post("/v2/geocode", map[string]any{"QueryText": "Space Needle", "AdditionalFeatures": []string{"TimeZone"}}))
	fmt.Printf("geocode: %v at %v (%v)\n", hit["Title"], hit["Position"], hit["TimeZone"].(map[string]any)["Name"])

	raw, _ := c.send(http.MethodGet, "/v2/place/"+hit["PlaceId"].(string), nil)
	place := map[string]any{}
	if err := json.Unmarshal(raw, &place); err != nil {
		exitf("get place: %v", err)
	}
	fmt.Println("get place:", place["Address"].(map[string]any)["Label"])

	rev := first(c.post("/v2/reverse-geocode", map[string]any{"QueryPosition": hit["Position"], "QueryRadius": 1000}))
	fmt.Printf("reverse geocode: %v (%v m away)\n", rev["Title"], rev["Distance"])

	near := c.post("/v2/search-nearby", map[string]any{"QueryPosition": []float64{-122.34, 47.61}, "QueryRadius": 5000})
	for _, it := range near["ResultItems"].([]any) {
		m := it.(map[string]any)
		fmt.Printf("nearby: %-20v %v m\n", m["Title"], m["Distance"])
	}

	text := c.post("/v2/search-text", map[string]any{"QueryText": "tower", "BiasPosition": []float64{2.35, 48.85}})
	for _, it := range text["ResultItems"].([]any) {
		fmt.Println("search text:", it.(map[string]any)["Title"])
	}

	auto := first(c.post("/v2/autocomplete", map[string]any{"QueryText": "Vanc"}))
	fmt.Println("autocomplete:", auto["Title"])
}

func first(out map[string]any) map[string]any {
	items, _ := out["ResultItems"].([]any)
	if len(items) == 0 {
		exitf("no results: %v", out)
	}
	return items[0].(map[string]any)
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
