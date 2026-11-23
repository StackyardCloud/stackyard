package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
	"github.com/stackyard/stackyard/internal/services/privateca"
)

func newPCAConnectorADTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// padCall sends a pca-connector-ad-signed REST call and decodes the reply.
func padCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "pca-connector-ad")
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("%s %s: %d (want %d) %s", method, path, resp.StatusCode, want, data)
	}
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("decode %s: %v", data, err)
		}
	}
	if want >= 400 && resp.Header.Get("X-Amzn-ErrorType") == "" {
		t.Fatalf("%s %s: error without X-Amzn-ErrorType", method, path)
	}
	return out
}

// padErr returns the error type (and Reason) of a call expected to fail.
func padErr(t *testing.T, ts *httptest.Server, method, path, body string) (string, string) {
	t.Helper()
	resp := signedRequestWithService(t, method, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "pca-connector-ad")
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 400 {
		t.Fatalf("%s %s: status %d, want an error (%s)", method, path, resp.StatusCode, data)
	}
	var e struct{ Reason string }
	_ = json.Unmarshal(data, &e)
	return resp.Header.Get("X-Amzn-ErrorType"), e.Reason
}

func padEsc(arn string) string { return url.PathEscape(arn) }

// padNewCA creates an ACTIVE Private CA in the server's store.
func padNewCA(t *testing.T, srv *Server) string {
	t.Helper()
	arn, err := srv.privateca.CreateCertificateAuthority(privateca.CreateCertificateAuthorityInput{
		Configuration: privateca.CertificateAuthorityConfiguration{KeyAlgorithm: "RSA_2048", SigningAlgorithm: "SHA256WITHRSA"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return arn
}

func TestPCAConnectorADStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.PCAConnectorADModel()
	if len(pcaConnectorADOperations) != len(model.Operations) || len(pcaConnectorADOperationByName) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(pcaConnectorADOperations), len(model.Operations))
	}
	for name := range model.Operations {
		if _, ok := pcaConnectorADOperationByName[name]; !ok {
			t.Fatalf("operation %s missing from the catalog", name)
		}
	}
}

func TestPCAConnectorADStage0Routing(t *testing.T) {
	_, ts := newPCAConnectorADTestServer(t)
	padCall(t, ts, http.StatusNotFound, http.MethodPut, "/connectors", "")

	// A /tags request signed for another service isn't claimed.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/connectors", nil, nil, "s3")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "Connectors") {
		t.Fatalf("an s3-signed request reached pca-connector-ad: %s", body)
	}
	// Private CA (acm-pca, JSON 1.1) keeps its traffic.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "ACMPrivateCA.ListCertificateAuthorities",
	}, "acm-pca")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "CertificateAuthorities") {
		t.Fatalf("acm-pca ListCertificateAuthorities: %d %s", resp.StatusCode, body)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/connectors", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/pca-connector-ad/aws4_request, SignedHeaders=host, Signature=00")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", r.StatusCode)
	}
}

func TestPCAConnectorADStage0TemplateChecks(t *testing.T) {
	valid := padTestDefinition()
	if schema, err := padCheckDefinition(valid); err != nil || schema != 2 {
		t.Fatalf("valid V2 definition: %d %v", schema, err)
	}
	both := padTestDefinition()
	both["TemplateV3"] = both["TemplateV2"]
	if _, err := padCheckDefinition(both); err == nil {
		t.Fatal("a definition with two versions was accepted")
	}
	longRenewal := padTestDefinition()
	cv := longRenewal["TemplateV2"].(map[string]any)["CertificateValidity"].(map[string]any)
	cv["RenewalPeriod"] = map[string]any{"Period": float64(2), "PeriodType": "YEARS"}
	if _, err := padCheckDefinition(longRenewal); err == nil {
		t.Fatal("a renewal period longer than the validity period was accepted")
	}
	badPolicy := padTestDefinition()
	ext := badPolicy["TemplateV2"].(map[string]any)["Extensions"].(map[string]any)
	ext["ApplicationPolicies"] = map[string]any{"Policies": []any{map[string]any{"PolicyType": "CODE_SIGNING", "PolicyObjectIdentifier": "1.3.6.1"}}}
	if _, err := padCheckDefinition(badPolicy); err == nil {
		t.Fatal("an application policy setting both union members was accepted")
	}
	for name, ok := range map[string]bool{"Web Server": true, "   ": false, "a+b": false, `a"b`: false, strings.Repeat("x", 65): false} {
		if err := padCheckTemplateName("Name", name); (err == nil) != ok {
			t.Fatalf("template name %q: ok=%v err=%v", name, ok, err)
		}
	}
}

// padTestDefinition is a minimal valid TemplateV2 definition.
func padTestDefinition() map[string]any {
	var def map[string]any
	_ = json.Unmarshal([]byte(padTestDefinitionJSON), &def)
	return def
}

const padTestDefinitionJSON = `{"TemplateV2":{
 "CertificateValidity":{"ValidityPeriod":{"Period":1,"PeriodType":"YEARS"},"RenewalPeriod":{"Period":6,"PeriodType":"WEEKS"}},
 "EnrollmentFlags":{"IncludeSymmetricAlgorithms":true},
 "Extensions":{"KeyUsage":{"UsageFlags":{"DigitalSignature":true,"KeyEncipherment":true}},"ApplicationPolicies":{"Policies":[{"PolicyType":"SERVER_AUTHENTICATION"}]}},
 "GeneralFlags":{"AutoEnrollment":true,"MachineType":true},
 "PrivateKeyAttributes":{"KeySpec":"KEY_EXCHANGE","MinimalKeyLength":2048},
 "PrivateKeyFlags":{"ClientVersion":"WINDOWS_SERVER_2016"},
 "SubjectNameFlags":{"SanRequireDns":true}}}`
