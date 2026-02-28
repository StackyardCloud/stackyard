package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func privateCARequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "ACMPrivateCA." + action,
		},
		"acm-pca",
	)
}

func TestPrivateCAStage1Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBody := []byte(`{
		"CertificateAuthorityConfiguration": {
			"KeyAlgorithm": "RSA_2048",
			"SigningAlgorithm": "SHA256WITHRSA",
			"Subject": {
				"Country": "US",
				"Organization": "Stackyard",
				"CommonName": "stackyard-privateca"
			}
		},
		"CertificateAuthorityType": "ROOT",
		"IdempotencyToken": "privateca-stage1"
	}`)
	resp := privateCARequest(t, ts, "CreateCertificateAuthority", createBody)
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		CertificateAuthorityArn string `json:"CertificateAuthorityArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create output: %v", err)
	}
	if strings.TrimSpace(createOut.CertificateAuthorityArn) == "" {
		t.Fatalf("expected certificate authority arn")
	}

	describePayload, _ := json.Marshal(map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn})
	resp = privateCARequest(t, ts, "DescribeCertificateAuthority", describePayload)
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		CertificateAuthority struct {
			Arn    string `json:"Arn"`
			Status string `json:"Status"`
		} `json:"CertificateAuthority"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe output: %v", err)
	}
	if describeOut.CertificateAuthority.Arn != createOut.CertificateAuthorityArn {
		t.Fatalf("expected ARN %q, got %q", createOut.CertificateAuthorityArn, describeOut.CertificateAuthority.Arn)
	}
	if describeOut.CertificateAuthority.Status != "ACTIVE" {
		t.Fatalf("expected ACTIVE status, got %q", describeOut.CertificateAuthority.Status)
	}

	resp = privateCARequest(t, ts, "ListCertificateAuthorities", []byte(`{"MaxResults":10}`))
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		CertificateAuthorities []struct {
			Arn string `json:"Arn"`
		} `json:"CertificateAuthorities"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal list output: %v", err)
	}
	if len(listOut.CertificateAuthorities) == 0 {
		t.Fatalf("expected at least one certificate authority in list")
	}

	updateBody, _ := json.Marshal(map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"Status":                  "DISABLED",
	})
	resp = privateCARequest(t, ts, "UpdateCertificateAuthority", updateBody)
	assertStatus(t, resp, http.StatusOK)

	deleteBody, _ := json.Marshal(map[string]any{
		"CertificateAuthorityArn":     createOut.CertificateAuthorityArn,
		"PermanentDeletionTimeInDays": 7,
	})
	resp = privateCARequest(t, ts, "DeleteCertificateAuthority", deleteBody)
	assertStatus(t, resp, http.StatusOK)

	restoreBody, _ := json.Marshal(map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn})
	resp = privateCARequest(t, ts, "RestoreCertificateAuthority", restoreBody)
	assertStatus(t, resp, http.StatusOK)

	resp = privateCARequest(t, ts, "DescribeCertificateAuthority", describePayload)
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe output after restore: %v", err)
	}
	if describeOut.CertificateAuthority.Status != "DISABLED" {
		t.Fatalf("expected DISABLED status after restore, got %q", describeOut.CertificateAuthority.Status)
	}
}

func TestPrivateCAStage1ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBody := []byte(`{
		"CertificateAuthorityConfiguration": {
			"KeyAlgorithm": "RSA_2048",
			"SigningAlgorithm": "SHA256WITHRSA",
			"Subject": {
				"Country": "US",
				"Organization": "Stackyard",
				"CommonName": "stackyard-privateca"
			}
		},
		"CertificateAuthorityType": "ROOT"
	}`)
	resp := privateCARequest(t, ts, "CreateCertificateAuthority", createBody)
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		CertificateAuthorityArn string `json:"CertificateAuthorityArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create output: %v", err)
	}

	actions := []struct {
		Name string
		Body []byte
	}{
		{Name: "DescribeCertificateAuthority", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn})},
		{Name: "ListCertificateAuthorities", Body: []byte(`{"MaxResults":10}`)},
		{Name: "UpdateCertificateAuthority", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "Status": "DISABLED"})},
		{Name: "DeleteCertificateAuthority", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "PermanentDeletionTimeInDays": 7})},
		{Name: "RestoreCertificateAuthority", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn})},
	}

	for _, action := range actions {
		resp = privateCARequest(t, ts, action.Name, action.Body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.Name)
		}
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	out, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return out
}
