// Command geo-routes runs the Amazon Location Service Routes V2 API against
// Stackyard: a route through a waypoint, a route matrix, an isoline, an
// optimized delivery order and a snapped GPS trace.
//
// Stackyard computes straight great-circle geometry at a fixed speed per
// travel mode. The SDK has no geo-routes module in this repository's
// go.mod, so requests are signed by hand.
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

const signingName = "geo-routes"

var (
	seattle   = []float64{-122.33207, 47.60621}
	everett   = []float64{-122.20208, 47.97898}
	vancouver = []float64{-123.11934, 49.24966}
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard geo-routes client using %s\n", c.endpoint)

	routes := c.post("/v2/routes", map[string]any{
		"Origin": seattle, "Destination": vancouver, "Waypoints": []any{map[string]any{"Position": everett}},
		"DepartureTime": "2026-01-02T08:00:00-08:00",
	})
	route := routes["Routes"].([]any)[0].(map[string]any)
	sum := route["Summary"].(map[string]any)
	fmt.Printf("route: %d leg(s), %.0f m, %.0f s\n", len(route["Legs"].([]any)), sum["Distance"], sum["Duration"])

	matrix := c.post("/v2/route-matrix", map[string]any{
		"Origins":         []any{map[string]any{"Position": seattle}, map[string]any{"Position": everett}},
		"Destinations":    []any{map[string]any{"Position": vancouver}},
		"RoutingBoundary": map[string]any{"Unbounded": true},
	})
	for i, row := range matrix["RouteMatrix"].([]any) {
		e := row.([]any)[0].(map[string]any)
		fmt.Printf("matrix origin %d -> Vancouver: %.0f m\n", i, e["Distance"])
	}

	iso := c.post("/v2/isolines", map[string]any{"Origin": seattle, "Thresholds": map[string]any{"Time": []int{600, 1200}}})
	fmt.Printf("isolines: %d (format %v)\n", len(iso["Isolines"].([]any)), iso["IsolineGeometryFormat"])

	opt := c.post("/v2/optimize-waypoints", map[string]any{
		"Origin": seattle, "Destination": seattle,
		"Waypoints": []any{
			map[string]any{"Id": "vancouver", "Position": vancouver},
			map[string]any{"Id": "everett", "Position": everett, "ServiceDuration": 300},
		},
	})
	var order []string
	for _, w := range opt["OptimizedWaypoints"].([]any) {
		order = append(order, fmt.Sprint(w.(map[string]any)["Id"]))
	}
	fmt.Println("optimized order:", strings.Join(order, " -> "))

	snap := c.post("/v2/snap-to-roads", map[string]any{"TracePoints": []any{
		map[string]any{"Position": seattle}, map[string]any{"Position": []float64{-122.3321, 47.6070}},
	}})
	fmt.Printf("snapped %d trace point(s)\n", len(snap["SnappedTracePoints"].([]any)))
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
