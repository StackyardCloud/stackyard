package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCloudFrontKeyValueStoreStage0Operations(t *testing.T) {
	if len(cloudfrontkeyvaluestoreOperations) != 6 {
		t.Fatalf("expected 6 operations, got %d", len(cloudfrontkeyvaluestoreOperations))
	}
	if len(cloudfrontkeyvaluestoreOperationByName) != len(cloudfrontkeyvaluestoreOperations) {
		t.Fatal("operation names are not unique")
	}
	for _, name := range []string{"DescribeKeyValueStore", "GetKey", "PutKey", "DeleteKey", "ListKeys", "UpdateKeys"} {
		if _, ok := cloudfrontkeyvaluestoreOperationByName[name]; !ok {
			t.Fatalf("missing operation %s", name)
		}
	}
}

func newCFKVSTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func TestCloudFrontKeyValueStoreStage0Routing(t *testing.T) {
	ts := newCFKVSTestServer(t)

	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/key-value-stores/x/other", nil, nil, cfKVSSigningName)
	assertStatus(t, resp, http.StatusNotImplemented)
	if body := mustBody(t, resp); !strings.Contains(string(body), "NotImplementedException") {
		t.Fatalf("unexpected body %s", body)
	}

	// Unsigned requests are rejected.
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/key-value-stores/x", nil)
	req.Host = "123456789012.cloudfront-kvs.global.api.aws"
	unsigned, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, unsigned, http.StatusForbidden)
	unsigned.Body.Close()

	// The CloudFront control plane still works next to the KVS router.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/2020-05-31/distribution", nil, nil, "cloudfront")
	assertStatus(t, resp, http.StatusOK)
	if body := mustBody(t, resp); !strings.Contains(string(body), "ListDistributionsResponse") {
		t.Fatalf("unexpected body %s", body)
	}
}

func TestCloudFrontKeyValueStoreStage0SigV4AOverProxy(t *testing.T) {
	ts := newCFKVSTestServer(t)
	arn := cfKVSTestCreate(t, ts, "sigv4a")

	// Shaped like the AWS CLI: SigV4A, account-prefixed host, absolute-form target.
	target := "http://123456789012.localhost/key-value-stores/" + strings.ReplaceAll(strings.ReplaceAll(arn, ":", "%3A"), "/", "%2F")
	req, _ := http.NewRequest(http.MethodGet, target, nil)
	sigV4ATestSign(t, req, nil, cfKVSSigningName)
	proxyURL := ts.URL
	client := &http.Client{Transport: &http.Transport{Proxy: func(*http.Request) (*url.URL, error) { return url.Parse(proxyURL) }}}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, resp, http.StatusOK)
	var out struct{ KvsARN string }
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.KvsARN != arn {
		t.Fatalf("unexpected describe: %+v %v", out, err)
	}
	if resp.Header.Get("ETag") == "" {
		t.Fatal("missing ETag header")
	}
}
