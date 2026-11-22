package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEKSAuthStage0Catalog(t *testing.T) {
	if len(eksauthOperations) != 1 || len(eksauthOperationByName) != 1 {
		t.Fatalf("operations: %d / %d", len(eksauthOperations), len(eksauthOperationByName))
	}
	if _, ok := eksauthOperationByName["AssumeRoleForPodIdentity"]; !ok {
		t.Fatal("missing AssumeRoleForPodIdentity")
	}
}

// eks-auth and eks share the /clusters/ path space; routing is by signing name.
func TestEKSAuthStage0RoutingByService(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// An EKS call still reaches the EKS router.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/clusters", nil, nil, "eks")
	assertStatus(t, resp, http.StatusOK)
	var list struct {
		Clusters []string `json:"clusters"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &list); err != nil {
		t.Fatalf("eks list-clusters: %v", err)
	}

	// An unknown eks-auth route is EKS Auth's 404, not EKS's.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/clusters", nil, nil, "eks-auth")
	assertStatus(t, resp, http.StatusNotFound)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "UnknownOperationException") {
		t.Fatalf("unknown eks-auth route: %s", body)
	}

	// The eks-auth operation path signed for eks stays with EKS.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/clusters/demo/assume-role-for-pod-identity", []byte(`{"token":"a.b.c"}`), map[string]string{"Content-Type": "application/json"}, "eks")
	if body := string(mustBody(t, resp)); strings.Contains(body, "InvalidTokenException") {
		t.Fatalf("eks-signed request reached EKS Auth: %s", body)
	}
}

func TestEKSAuthStage0RefusesBodiesWithoutDeclaredLength(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/clusters/demo/assume-role-for-pod-identity", io.MultiReader(strings.NewReader(`{"token":"a.b.c"}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/eks-auth/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}
