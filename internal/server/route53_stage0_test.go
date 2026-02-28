package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func route53Request(t *testing.T, ts *httptest.Server, method, requestPath, action string, body []byte, headers map[string]string) *http.Response {
	t.Helper()
	headersCopy := map[string]string{}
	for k, v := range headers {
		headersCopy[k] = v
	}
	if action != "" {
		headersCopy["X-Stackyard-Operation"] = action
	}
	if method == "" {
		method = http.MethodPost
	}
	if requestPath == "" {
		requestPath = "/2013-04-01/hostedzone"
	}
	return signedRequestWithService(t, method, ts.URL+requestPath, body, headersCopy, "route53")
}

func TestRoute53Stage0CatalogCoverage(t *testing.T) {
	if len(route53Operations) != 71 {
		t.Fatalf("expected 71 Route 53 operations from docs, got %d", len(route53Operations))
	}
	if len(route53OperationByName) != len(route53Operations) {
		t.Fatalf("expected unique operation names")
	}

	requiredOps := []string{
		"CreateHostedZone",
		"ChangeResourceRecordSets",
		"GetChange",
		"ListHostedZones",
		"ListResourceRecordSets",
		"CreateHealthCheck",
		"GetCheckerIpRanges",
	}
	for _, name := range requiredOps {
		if _, ok := route53OperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}

	if len(route53DataTypes) != 43 {
		t.Fatalf("expected 43 Route 53 data types from docs, got %d", len(route53DataTypes))
	}
	if len(route53DataTypeByName) != len(route53DataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"HostedZone",
		"ResourceRecordSet",
		"ChangeInfo",
		"HealthCheck",
		"TrafficPolicy",
		"VPC",
	}
	for _, name := range requiredTypes {
		if _, ok := route53DataTypeByName[name]; !ok {
			t.Fatalf("missing documented data type %s", name)
		}
	}
}

func TestRoute53Stage0UserAgentOperationParsing(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := route53Request(t, ts, http.MethodGet, "/2013-04-01/hostedzone", "", nil, map[string]string{
		"User-Agent": "aws-cli/2.22.22 md/command#route53.list-hosted-zones",
	})
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ListHostedZonesResponse") {
		t.Fatalf("expected ListHostedZonesResponse in body, got %q", body)
	}
}

func TestRoute53Stage0UnknownActionReturnsError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := route53Request(t, ts, http.MethodPost, "/2013-04-01/hostedzone", "TotallyUnknownAction", nil, nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "NoSuchAction") {
		t.Fatalf("expected NoSuchAction response body, got %q", body)
	}
}

func TestRoute53Stage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range route53Operations {
		resp := route53Request(t, ts, http.MethodPost, "/2013-04-01/hostedzone", op.Name, nil, nil)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
