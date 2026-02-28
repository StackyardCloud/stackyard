package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCodeArtifactStage0CatalogCoverage(t *testing.T) {
	if len(codeArtifactOperations) != 48 {
		t.Fatalf("expected 48 CodeArtifact operations from docs, got %d", len(codeArtifactOperations))
	}
	if len(codeArtifactOperationByName) != len(codeArtifactOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredOps := []string{
		"CreateDomain",
		"CreateRepository",
		"GetAuthorizationToken",
		"ListDomains",
		"PublishPackageVersion",
		"UpdateRepository",
	}
	for _, name := range requiredOps {
		if _, ok := codeArtifactOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}

	if len(codeArtifactDataTypes) != 29 {
		t.Fatalf("expected 29 CodeArtifact data types from docs, got %d", len(codeArtifactDataTypes))
	}
	if len(codeArtifactDataTypeByName) != len(codeArtifactDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"DomainDescription",
		"PackageDescription",
		"PackageVersionSummary",
		"RepositoryDescription",
		"Tag",
		"UpstreamRepositoryInfo",
	}
	for _, name := range requiredTypes {
		if _, ok := codeArtifactDataTypeByName[name]; !ok {
			t.Fatalf("missing documented data type %s", name)
		}
	}
}

func TestCodeArtifactStage0KnownRouteDoesNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/v1/domains",
		[]byte(`{}`),
		map[string]string{"Content-Type": "application/json"},
		"codeartifact",
	)
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected CodeArtifact route to be implemented")
	}
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected non-NotImplemented response body, got %q", body)
	}
}
