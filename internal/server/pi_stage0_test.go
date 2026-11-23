package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
	rdssvc "github.com/stackyard/stackyard/internal/services/rds"
)

func newPITestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// piInstance creates an RDS instance and returns its DbiResourceId.
func piInstance(t *testing.T, srv *Server, id, engine string) string {
	t.Helper()
	if _, err := srv.rds.CreateDBInstance(rdssvc.CreateDBInstanceInput{Identifier: id, Engine: engine, DBInstanceClass: "db.r6g.large", AllocatedStorage: 20, MasterUsername: "admin", MasterUserPassword: "secret123"}); err != nil {
		t.Fatal(err)
	}
	return rdssvc.ResourceID(id)
}

func piCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": piTargetPrefix + op,
	}, "pi")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func piOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := piCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func piErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := piCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestPIStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.PIModel()
	if len(piOperations) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(piOperations), len(model.Operations))
	}
	for _, op := range piOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Errorf("%s is not in the model", op.Name)
		}
		if piHandlers[op.Name] == nil {
			t.Errorf("%s has no handler", op.Name)
		}
	}
}

func TestPIStage0RoutingAndBodies(t *testing.T) {
	srv, ts := newPITestServer(t)
	id := piInstance(t, srv, "route-db", "postgres")
	piOK(t, ts, "GetResourceMetadata", `{"ServiceType":"RDS","Identifier":"`+id+`"}`)

	// Signed for another service: refused, never served as pi.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{"ServiceType":"RDS","Identifier":"`+id+`"}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": piTargetPrefix + "GetResourceMetadata",
	}, "s3")
	if body := mustBody(t, resp); resp.StatusCode == http.StatusOK {
		t.Fatalf("s3-signed request was served: %s", body)
	}

	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", piTargetPrefix+"GetResourceMetadata")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", r.StatusCode)
	}

	// RDS's own query API is untouched.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte("Action=DescribeDBInstances&Version=2014-10-31"), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "rds")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "route-db") {
		t.Fatalf("rds DescribeDBInstances: %d %s", resp.StatusCode, body)
	}
}
