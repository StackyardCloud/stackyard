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

func newVPTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func vpCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": vpTargetPrefix + op,
	}, "verifiedpermissions")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func vpOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := vpCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func vpErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := vpCall(t, ts, op, body)
	if status == http.StatusOK {
		t.Fatalf("%s: want an error, got 200 %v", op, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestVerifiedPermissionsStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.VerifiedPermissionsModel()
	if len(verifiedPermissionsOperations) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(verifiedPermissionsOperations), len(model.Operations))
	}
	for _, op := range verifiedPermissionsOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Fatalf("%s is not in the model", op.Name)
		}
		if vpHandlers[op.Name] == nil {
			t.Fatalf("%s has no handler", op.Name)
		}
	}
}

func TestVerifiedPermissionsStage0Routing(t *testing.T) {
	_, ts := newVPTestServer(t)
	vpOK(t, ts, "ListPolicyStores", `{}`)
	// A request signed for another service with our target isn't claimed.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": vpTargetPrefix + "ListPolicyStores",
	}, "s3")
	if strings.Contains(string(mustBody(t, resp)), "policyStores") {
		t.Fatal("an s3-signed request reached Verified Permissions")
	}
	// Neighbouring JSON services keep their traffic.
	aossOK(t, ts, "GetAccountSettings", `{}`)
}

func TestVerifiedPermissionsStage0RefusesUndeclaredBodies(t *testing.T) {
	_, ts := newVPTestServer(t)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Amz-Target", vpTargetPrefix+"ListPolicyStores")
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/verifiedpermissions/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}
