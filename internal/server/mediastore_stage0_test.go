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

func newMediaStoreTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// mstCall sends a MediaStore JSON call and returns status, error type and body.
func mstCall(t *testing.T, ts *httptest.Server, op, body string) (int, string, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body),
		map[string]string{"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": mstTargetPrefix + op}, "mediastore")
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %s: %v", op, raw, err)
		}
	}
	return resp.StatusCode, resp.Header.Get("X-Amzn-ErrorType"), out
}

func mstOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, code, out := mstCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %s %v", op, body, status, code, out)
	}
	return out
}

func mstFail(t *testing.T, ts *httptest.Server, op, body, want string) {
	t.Helper()
	status, code, out := mstCall(t, ts, op, body)
	if status == http.StatusOK || code != want {
		t.Fatalf("%s %s: %d %s (want %s) %v", op, body, status, code, want, out)
	}
}

func TestMediaStoreStage0CatalogMatchesModel(t *testing.T) {
	for _, c := range []struct {
		model *awsmodels.Model
		names map[string]bool
	}{
		{awsmodels.MediaStoreModel(), func() map[string]bool {
			m := map[string]bool{}
			for n := range mstOperationByName {
				m[n] = mstHandlers[n] != nil
			}
			return m
		}()},
		{awsmodels.MediaStoreDataModel(), func() map[string]bool {
			m := map[string]bool{}
			for n := range mstdOperationByName {
				m[n] = true
			}
			return m
		}()},
	} {
		if len(c.names) != len(c.model.Operations) {
			t.Fatalf("catalog has %d operations, model has %d", len(c.names), len(c.model.Operations))
		}
		for name := range c.model.Operations {
			if !c.names[name] {
				t.Fatalf("model operation %s missing from the catalog or handlers", name)
			}
		}
	}
}

func TestMediaStoreStage0Routing(t *testing.T) {
	ts := newMediaStoreTestServer(t)
	mstFail(t, ts, "NoSuchOperation", "{}", "UnknownOperationException")
	mstFail(t, ts, "CreateContainer", `{"ContainerName":"bad name!"}`, "ValidationException")

	// S3 keeps its path-style traffic, even under a would-be data-plane path.
	resp := signedRequestWithService(t, http.MethodPut, ts.URL+"/mst-routing-bucket", nil, nil, "s3")
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("s3 CreateBucket: %d", resp.StatusCode)
	}
	// A data-plane path signed for another service isn't claimed.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+mstdPathPrefix+"abc/x", nil, nil, "s3")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "ContainerNotFoundException") {
		t.Fatalf("an s3-signed request reached MediaStore Data: %s", body)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPut, ts.URL+mstdPathPrefix+"abc/x", io.MultiReader(strings.NewReader("data")))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/mediastore/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}
