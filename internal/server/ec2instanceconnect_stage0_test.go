package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newEC2InstanceConnectTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func eicRequest(t *testing.T, ts *httptest.Server, op, body string) *http.Response {
	t.Helper()
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": eicTargetPrefix + op,
	}, eicSigningName)
}

func TestEC2InstanceConnectStage0Catalog(t *testing.T) {
	if len(ec2instanceconnectOperations) != 2 || len(ec2instanceconnectOperationByName) != 2 {
		t.Fatalf("operations: %v", ec2instanceconnectOperations)
	}
	for _, op := range []string{"SendSSHPublicKey", "SendSerialConsoleSSHPublicKey"} {
		if _, ok := ec2instanceconnectOperationByName[op]; !ok {
			t.Fatalf("missing %s", op)
		}
	}
}

func TestEC2InstanceConnectStage0Routing(t *testing.T) {
	_, ts := newEC2InstanceConnectTestServer(t)
	resp := eicRequest(t, ts, "Nope", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "UnknownOperationException") {
		t.Fatalf("unknown op: %s", body)
	}
	// Signed for another service: refused, not served.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": eicTargetPrefix + "SendSSHPublicKey",
	}, "ec2")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d", resp.StatusCode)
	}
	resp.Body.Close()
	// The keys endpoint needs a signature too.
	plain, err := http.Get(ts.URL + eicKeysPathPrefix + "i-0123456789abcdef0/keys")
	if err != nil {
		t.Fatal(err)
	}
	plain.Body.Close()
	if plain.StatusCode != http.StatusForbidden {
		t.Fatalf("unsigned keys endpoint: %d", plain.StatusCode)
	}
}
