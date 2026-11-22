package server

import (
	"net/http"
	"strings"
	"testing"
)

// CodeStar Connections and CodeConnections share one store: each API shows a
// resource with its own ARN prefix and accepts either prefix as input.
func TestCodeStarConnectionsStage1SharesTheCodeConnectionsStore(t *testing.T) {
	_, ts := newCodeConnectionsTestServer(t)
	status, created := cxnCall(t, ts, "codestar-connections", csTestTarget, "CreateConnection", `{"ProviderType":"GitLab","ConnectionName":"gl"}`)
	if status != http.StatusOK {
		t.Fatalf("create via codestar-connections: %d %v", status, created)
	}
	csARN := created["ConnectionArn"].(string)
	if !strings.HasPrefix(csARN, "arn:aws:codestar-connections:") || !strings.Contains(csARN, ":connection/") {
		t.Fatalf("codestar ARN: %s", csARN)
	}
	cxnARN := strings.Replace(csARN, "codestar-connections", "codeconnections", 1)

	got := cxnOK(t, ts, "GetConnection", `{"ConnectionArn":"`+csARN+`"}`)["Connection"].(map[string]any)
	if got["ConnectionArn"] != cxnARN || got["ProviderType"] != "GitLab" {
		t.Fatalf("codeconnections view: %v", got)
	}
	status, out := cxnCall(t, ts, "codestar-connections", csTestTarget, "GetConnection", `{"ConnectionArn":"`+cxnARN+`"}`)
	if status != http.StatusOK || out["Connection"].(map[string]any)["ConnectionArn"] != csARN {
		t.Fatalf("codestar view of a codeconnections ARN: %d %v", status, out)
	}
	status, out = cxnCall(t, ts, "codestar-connections", csTestTarget, "GetConnection", `{"ConnectionArn":"arn:aws:codestar-connections:us-east-1:123456789012:connection/nope"}`)
	if status != http.StatusBadRequest || out["__type"] != "ResourceNotFoundException" {
		t.Fatalf("missing connection: %d %v", status, out)
	}
	status, _ = cxnCall(t, ts, "codestar-connections", csTestTarget, "DeleteConnection", `{"ConnectionArn":"`+cxnARN+`"}`)
	if status != http.StatusOK {
		t.Fatalf("delete via codestar-connections: %d", status)
	}
	if code := cxnErrCode(t, ts, "GetConnection", `{"ConnectionArn":"`+cxnARN+`"}`); code != "ResourceNotFoundException" {
		t.Fatalf("after delete: %s", code)
	}
}
