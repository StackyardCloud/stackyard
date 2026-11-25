package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// sgeoCall sends a signed SageMaker geospatial REST request.
func sgeoCall(t *testing.T, ts *httptest.Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	return sgeoCallAs(t, ts, method, path, body, sgeoSigningName)
}

func sgeoCallAs(t *testing.T, ts *httptest.Server, method, path, body, service string) (int, map[string]any) {
	t.Helper()
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, map[string]string{"Content-Type": "application/json"}, service)
	data := mustBody(t, resp)
	out := map[string]any{}
	if len(data) > 0 && strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("%s %s: decode %q: %v", method, path, data, err)
		}
	}
	if code := resp.Header.Get("X-Amzn-ErrorType"); code != "" {
		out["__type"] = code
	}
	return resp.StatusCode, out
}

func sgeoOK(t *testing.T, ts *httptest.Server, method, path, body string) map[string]any {
	t.Helper()
	status, out := sgeoCall(t, ts, method, path, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %v", method, path, status, out)
	}
	return out
}

func sgeoPath(prefix, arn string) string { return prefix + url.PathEscape(arn) }

func TestSageMakerGeospatialStage0CatalogMatchesModel(t *testing.T) {
	m := awsmodels.SageMakerGeospatialModel()
	if len(sageMakerGeospatialOperations) != len(m.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(sageMakerGeospatialOperations), len(m.Operations))
	}
	st := newSageMakerGeospatialStore()
	for _, op := range sageMakerGeospatialOperations {
		if _, ok := m.Operations[op.Name]; !ok {
			t.Errorf("%s is not in the model", op.Name)
		}
		if op.Name == "GetTile" {
			continue // served by tileSeed
		}
		if _, err := st.dispatch(op.Name, map[string]any{}, &sgeoPre{}); err != nil && err.code == "UnknownOperationException" {
			t.Errorf("%s is not dispatched", op.Name)
		}
	}
}

func TestSageMakerGeospatialStage0Routing(t *testing.T) {
	ts := newForecastTestServer(t)
	if out := sgeoOK(t, ts, http.MethodGet, "/raster-data-collections", ""); len(out["RasterDataCollectionSummaries"].([]any)) != 3 {
		t.Fatalf("collections: %v", out)
	}
	if status, out := sgeoCall(t, ts, http.MethodGet, sgeoPath("/earth-observation-jobs/", "arn:aws:sagemaker-geospatial:us-west-2:123456789012:earth-observation-job/none00000000"), ""); status != http.StatusNotFound || out["__type"] != "ResourceNotFoundException" {
		t.Fatalf("missing job: %d %v", status, out)
	}
	if status, out := sgeoCall(t, ts, http.MethodPost, "/list-earth-observation-jobs", `{"MaxResults":99}`); status != http.StatusBadRequest || out["__type"] != "ValidationException" {
		t.Fatalf("bad MaxResults: %d %v", status, out)
	}
	// SageMaker and SageMaker Runtime sign as sagemaker and keep their traffic.
	if status, out := jsonCall(t, ts, "SageMaker.ListModels", "sagemaker", `{}`); status != http.StatusOK || out["Models"] == nil {
		t.Fatalf("SageMaker ListModels: %d %v", status, out)
	}
	if status, _ := sgeoCallAs(t, ts, http.MethodGet, "/raster-data-collections", "", "sagemaker"); status == http.StatusOK {
		t.Fatal("a sagemaker-signed request reached the geospatial router")
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/endpoints/none/invocations", []byte(`{}`), map[string]string{"Content-Type": "application/json"}, "sagemaker")
	if body := string(mustBody(t, resp)); resp.Header.Get("X-Amzn-ErrorType") == "UnknownOperationException" || strings.Contains(body, "geospatial") {
		t.Fatalf("SageMaker Runtime InvokeEndpoint was claimed: %d %s", resp.StatusCode, body)
	}
	// Geospatial paths overlapping other REST services (tags) stay with their signer.
	if status, out := sgeoCall(t, ts, http.MethodGet, sgeoPath("/tags/", "arn:aws:sagemaker-geospatial:us-west-2:123456789012:earth-observation-job/none00000000"), ""); status != http.StatusNotFound {
		t.Fatalf("tags of a missing job: %d %v", status, out)
	}
}
