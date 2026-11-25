package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func newGeoTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// geoCall sends a request signed for service and returns status, headers and body.
func geoCall(t *testing.T, ts *httptest.Server, service, method, path, body string) (int, http.Header, []byte) {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, service)
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, data
}

// geoJSON expects 200 and a body that satisfies the operation's output shape.
func geoJSON(t *testing.T, ts *httptest.Server, md *awsmodels.Model, service, op, method, path, body string) map[string]any {
	t.Helper()
	status, h, data := geoCall(t, ts, service, method, path, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %s", op, status, data)
	}
	if h.Get("X-Amz-Geo-Pricing-Bucket") == "" {
		t.Fatalf("%s: no pricing bucket header", op)
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("%s: decode %s: %v", op, data, err)
	}
	var problems []string
	for _, p := range md.Validate(md.Operations[op].Output, out) {
		// The model makes VehicleLegDetails.TruckRoadTypes required with a
		// minimum of one item; AWS returns an empty list for non-truck legs.
		if !strings.HasSuffix(p, "TruckRoadTypes: must have at least 1 items") {
			problems = append(problems, p)
		}
	}
	if len(problems) > 0 {
		t.Fatalf("%s output does not match the model: %v\n%s", op, problems, data)
	}
	return out
}

// geoReject expects a 400 ValidationException carrying reason.
func geoReject(t *testing.T, ts *httptest.Server, service, method, path, body, reason string) {
	t.Helper()
	status, h, data := geoCall(t, ts, service, method, path, body)
	if status != http.StatusBadRequest || h.Get("X-Amzn-ErrorType") != "ValidationException" {
		t.Fatalf("%s %s %s: %d %s %s", method, path, body, status, h.Get("X-Amzn-ErrorType"), data)
	}
	out := map[string]any{}
	_ = json.Unmarshal(data, &out)
	if reason != "" && out["reason"] != reason {
		t.Fatalf("%s %s: reason %v, want %s (%s)", method, path, out["reason"], reason, data)
	}
}

func geoCatalogMatchesModel(t *testing.T, md *awsmodels.Model, ops map[string]geosaOp, handlers map[string]func(geosaCall) (*geosaReply, *geosaError)) {
	t.Helper()
	if len(ops) != len(md.Operations) || len(handlers) != len(md.Operations) {
		t.Fatalf("catalog %d / handlers %d, model %d operations", len(ops), len(handlers), len(md.Operations))
	}
	for name := range md.Operations {
		if _, ok := ops[name]; !ok {
			t.Fatalf("model operation %s missing from the catalog", name)
		}
		if handlers[name] == nil {
			t.Fatalf("no handler for %s", name)
		}
	}
}

func TestGeoStandaloneGeometryHelpers(t *testing.T) {
	// Reference vector from github.com/heremaps/flexible-polyline (precision 5).
	pts := [][2]float64{{8.6982122, 50.1022829}, {8.6956695, 50.1020076}, {8.6914960, 50.1006313}, {8.6875156, 50.0987800}}
	if got := grteFlexEncode(pts, 5); got != "BFoz5xJ67i1B1B7PzIhaxL7Y" {
		t.Fatalf("flexible polyline: %s", got)
	}
	// Seattle to Vancouver is about 192 km.
	if d := geosaDistance([2]float64{-122.33207, 47.60621}, [2]float64{-123.11934, 49.24966}); d < 190000 || d > 195000 {
		t.Fatalf("distance %f", d)
	}
	ring := grteCircle([2]float64{0, 0}, 1000, 32)
	if len(ring) != 33 || ring[0] != ring[32] {
		t.Fatalf("ring %v", ring)
	}
	if d := geosaDistance([2]float64{0, 0}, ring[5]); d < 999 || d > 1001 {
		t.Fatalf("ring radius %f", d)
	}
	if !geosaInBBox([4]float64{170, -10, -170, 10}, [2]float64{179, 0}) || geosaInBBox([4]float64{170, -10, -170, 10}, [2]float64{0, 0}) {
		t.Fatal("antimeridian bounding box")
	}
}
