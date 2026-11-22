package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ekssvc "github.com/stackyard/stackyard/internal/services/eks"
)

// ekaTestToken builds an unsigned projected service-account token.
func ekaTestToken(t *testing.T, claims map[string]any) string {
	t.Helper()
	enc := func(v any) string {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(b)
	}
	return enc(map[string]string{"alg": "RS256", "typ": "JWT"}) + "." + enc(claims) + ".c2ln"
}

func ekaClaimsFor(ns, sa string, exp time.Time) map[string]any {
	return map[string]any{
		"aud":           []string{"pods.eks.amazonaws.com"},
		"exp":           exp.Unix(),
		"sub":           "system:serviceaccount:" + ns + ":" + sa,
		"kubernetes.io": map[string]any{"pod": map[string]any{"name": "web-1"}},
	}
}

func ekaCall(t *testing.T, ts *httptest.Server, cluster, token string) (int, map[string]any) {
	t.Helper()
	body := []byte(fmt.Sprintf(`{"token":%q}`, token))
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/clusters/"+cluster+"/assume-role-for-pod-identity", body, map[string]string{"Content-Type": "application/json"}, "eks-auth")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("content type %q", resp.Header.Get("Content-Type"))
	}
	return resp.StatusCode, out
}

func TestEKSAuthStage1AssumeRoleForPodIdentity(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	if _, err := srv.eks.CreateCluster(ekssvc.CreateClusterInput{Name: "demo", RoleArn: "arn:aws:iam::123456789012:role/eks"}); err != nil {
		t.Fatal(err)
	}
	assoc, err := srv.eks.CreatePodIdentityAssociation("demo", ekssvc.CreatePodIdentityAssociationInput{Namespace: "apps", ServiceAccount: "web", RoleArn: "arn:aws:iam::123456789012:role/team/web-role"})
	if err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Hour)

	status, out := ekaCall(t, ts, "demo", ekaTestToken(t, ekaClaimsFor("apps", "web", future)))
	if status != http.StatusOK {
		t.Fatalf("assume: %d %v", status, out)
	}
	subject := out["subject"].(map[string]any)
	pia := out["podIdentityAssociation"].(map[string]any)
	user := out["assumedRoleUser"].(map[string]any)
	creds := out["credentials"].(map[string]any)
	if subject["namespace"] != "apps" || subject["serviceAccount"] != "web" || out["audience"] != "pods.eks.amazonaws.com" {
		t.Fatalf("subject/audience: %v", out)
	}
	if pia["associationId"] != assoc.AssociationID || pia["associationArn"] != assoc.AssociationArn {
		t.Fatalf("association: %v", pia)
	}
	if user["arn"] != "arn:aws:sts::123456789012:assumed-role/web-role/eks-demo-web-1" {
		t.Fatalf("assumed role user: %v", user)
	}
	if exp, _ := creds["expiration"].(float64); exp <= float64(time.Now().Unix()) || creds["accessKeyId"] == "" || creds["sessionToken"] == "" {
		t.Fatalf("credentials: %v", creds)
	}

	for name, tc := range map[string]struct {
		cluster, token, code string
		status               int
	}{
		"missing cluster":  {"nope", ekaTestToken(t, ekaClaimsFor("apps", "web", future)), "ResourceNotFoundException", http.StatusNotFound},
		"bad cluster name": {"-bad", ekaTestToken(t, ekaClaimsFor("apps", "web", future)), "InvalidParameterException", http.StatusBadRequest},
		"expired":          {"demo", ekaTestToken(t, ekaClaimsFor("apps", "web", time.Now().Add(-time.Minute))), "ExpiredTokenException", http.StatusBadRequest},
		"wrong audience":   {"demo", ekaTestToken(t, map[string]any{"aud": "sts.amazonaws.com", "exp": future.Unix(), "sub": "system:serviceaccount:apps:web"}), "InvalidTokenException", http.StatusBadRequest},
		"bad subject":      {"demo", ekaTestToken(t, map[string]any{"aud": "pods.eks.amazonaws.com", "exp": future.Unix(), "sub": "user:alice"}), "InvalidTokenException", http.StatusBadRequest},
		"no expiry":        {"demo", ekaTestToken(t, map[string]any{"aud": "pods.eks.amazonaws.com", "sub": "system:serviceaccount:apps:web"}), "InvalidTokenException", http.StatusBadRequest},
		"not a jwt":        {"demo", "not-a-jwt", "InvalidTokenException", http.StatusBadRequest},
		"oversized":        {"demo", strings.Repeat("a", 17<<10) + ".b.c", "InvalidTokenException", http.StatusBadRequest},
		"no association":   {"demo", ekaTestToken(t, ekaClaimsFor("apps", "other", future)), "AccessDeniedException", http.StatusBadRequest},
	} {
		status, out := ekaCall(t, ts, tc.cluster, tc.token)
		if status != tc.status || out["__type"] != tc.code {
			t.Errorf("%s: %d %v (want %d %s)", name, status, out, tc.status, tc.code)
		}
	}

	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/clusters/demo/assume-role-for-pod-identity", []byte(`{"token":"a.b.c","extra":1}`), map[string]string{"Content-Type": "application/json"}, "eks-auth")
	assertStatus(t, resp, http.StatusBadRequest)
	_ = mustBody(t, resp)
}
