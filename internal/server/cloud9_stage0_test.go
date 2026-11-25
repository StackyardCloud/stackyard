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

func newC9TestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// c9Call sends a signed awsJson request for api and decodes the reply.
func c9Call(t *testing.T, ts *httptest.Server, api *c9JSONAPI, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": api.contentType, "X-Amz-Target": api.prefix + op,
	}, api.signing)
	raw := mustBody(t, resp)
	if ct := resp.Header.Get("Content-Type"); ct != api.contentType {
		t.Fatalf("%s: content type %q (%s)", op, ct, raw)
	}
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func c9OK(t *testing.T, ts *httptest.Server, api *c9JSONAPI, op, body string) map[string]any {
	t.Helper()
	status, out := c9Call(t, ts, api, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

// c9Fail expects wantStatus and returns the error code.
func c9Fail(t *testing.T, ts *httptest.Server, api *c9JSONAPI, op, body string, wantStatus int) string {
	t.Helper()
	status, out := c9Call(t, ts, api, op, body)
	if status != wantStatus {
		t.Fatalf("%s: want %d, got %d %v", op, wantStatus, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestCloud9Stage0CatalogsMatchModels(t *testing.T) {
	for _, c := range []struct {
		name  string
		ops   map[string]bool
		model *awsmodels.Model
		api   *c9JSONAPI
	}{
		{"cloud9", c9OperationByName, awsmodels.Cloud9Model(), c9API},
		{"invoicing", invcOperationByName, awsmodels.InvoicingModel(), invcAPI},
	} {
		if len(c.ops) != len(c.model.Operations) || len(c.api.handlers) != len(c.ops) {
			t.Fatalf("%s: catalog %d, model %d, handlers %d", c.name, len(c.ops), len(c.model.Operations), len(c.api.handlers))
		}
		for op := range c.ops {
			if _, ok := c.model.Operations[op]; !ok || c.api.handlers[op] == nil {
				t.Errorf("%s: %s is not in the model or has no handler", c.name, op)
			}
		}
	}
}

func TestCloud9Stage0Routing(t *testing.T) {
	_, ts := newC9TestServer(t)
	c9OK(t, ts, c9API, "ListEnvironments", `{}`)

	// Signed for another service: never served as Cloud9.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": c9TargetPrefix + "ListEnvironments",
	}, "s3")
	if body := mustBody(t, resp); resp.StatusCode == http.StatusOK && strings.Contains(string(body), "environmentIds") {
		t.Fatalf("s3-signed request was served as Cloud9: %s", body)
	}

	// EC2 keeps its query traffic.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte("Action=DescribeVpcs&Version=2016-11-15"), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "ec2")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "DescribeVpcsResponse") {
		t.Fatalf("ec2 DescribeVpcs: %d %s", resp.StatusCode, body)
	}

	// A chunked body of unknown length is refused.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", c9TargetPrefix+"ListEnvironments")
	req.Host = "cloud9.us-east-1.amazonaws.com"
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", r.StatusCode)
	}

	// Oversized bodies and unknown members are refused.
	big := `{"nextToken":"` + strings.Repeat("x", c9MaxBody) + `"}`
	if code := c9Fail(t, ts, c9API, "ListEnvironments", big, http.StatusBadRequest); code != "BadRequestException" {
		t.Fatalf("oversized body: %s", code)
	}
	if code := c9Fail(t, ts, c9API, "ListEnvironments", `{"nope":1}`, http.StatusBadRequest); code != "BadRequestException" {
		t.Fatalf("unknown member: %s", code)
	}
	if code := c9Fail(t, ts, c9API, "ListEnvironments", `{"nextToken":"zzz"}`, http.StatusBadRequest); code != "BadRequestException" {
		t.Fatalf("bad token: %s", code)
	}
}
