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
	status, created := ccCall(t, ts, "codestar-connections", csTestTarget, "CreateConnection", `{"ProviderType":"GitLab","ConnectionName":"gl"}`)
	if status != http.StatusOK {
		t.Fatalf("create via codestar-connections: %d %v", status, created)
	}
	csARN := created["ConnectionArn"].(string)
	if !strings.HasPrefix(csARN, "arn:aws:codestar-connections:") || !strings.Contains(csARN, ":connection/") {
		t.Fatalf("codestar ARN: %s", csARN)
	}
	ccARN := strings.Replace(csARN, "codestar-connections", "codeconnections", 1)

	got := ccOK(t, ts, "GetConnection", `{"ConnectionArn":"`+csARN+`"}`)["Connection"].(map[string]any)
	if got["ConnectionArn"] != ccARN || got["ProviderType"] != "GitLab" {
		t.Fatalf("codeconnections view: %v", got)
	}
	status, out := ccCall(t, ts, "codestar-connections", csTestTarget, "GetConnection", `{"ConnectionArn":"`+ccARN+`"}`)
	if status != http.StatusOK || out["Connection"].(map[string]any)["ConnectionArn"] != csARN {
		t.Fatalf("codestar view of a codeconnections ARN: %d %v", status, out)
	}
	status, out = ccCall(t, ts, "codestar-connections", csTestTarget, "GetConnection", `{"ConnectionArn":"arn:aws:codestar-connections:us-east-1:123456789012:connection/nope"}`)
	if status != http.StatusBadRequest || out["__type"] != "ResourceNotFoundException" {
		t.Fatalf("missing connection: %d %v", status, out)
	}
	status, _ = ccCall(t, ts, "codestar-connections", csTestTarget, "DeleteConnection", `{"ConnectionArn":"`+ccARN+`"}`)
	if status != http.StatusOK {
		t.Fatalf("delete via codestar-connections: %d", status)
	}
	if code := ccErrCode(t, ts, "GetConnection", `{"ConnectionArn":"`+ccARN+`"}`); code != "ResourceNotFoundException" {
		t.Fatalf("after delete: %s", code)
	}
}
