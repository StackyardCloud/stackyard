package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRedshiftStage6ClusterCredentials(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"cred-demo"},
		"NodeType":           []string{"dc2.large"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret123"},
	}
	status, body := redshiftRequest(t, ts, create)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create, got %d: %s", status, string(body))
	}

	creds := url.Values{
		"Action":            []string{"GetClusterCredentials"},
		"ClusterIdentifier": []string{"cred-demo"},
		"DbUser":            []string{"appuser"},
		"DbName":            []string{"dev"},
	}
	status, body = redshiftRequest(t, ts, creds)
	if status != http.StatusOK {
		t.Fatalf("expected 200 get credentials, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DbUser>appuser</DbUser>")) {
		t.Fatalf("missing DbUser: %s", string(body))
	}
	if !bytes.Contains(body, []byte("<DbPassword>")) {
		t.Fatalf("missing DbPassword: %s", string(body))
	}

	badDuration := url.Values{
		"Action":            []string{"GetClusterCredentials"},
		"ClusterIdentifier": []string{"cred-demo"},
		"DbUser":            []string{"appuser"},
		"DurationSeconds":   []string{"120"},
	}
	status, body = redshiftRequest(t, ts, badDuration)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 bad duration, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>InvalidParameterValue</Code>")) {
		t.Fatalf("expected InvalidParameterValue: %s", string(body))
	}

	missingUser := url.Values{
		"Action":            []string{"GetClusterCredentials"},
		"ClusterIdentifier": []string{"cred-demo"},
	}
	status, body = redshiftRequest(t, ts, missingUser)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 missing DbUser, got %d: %s", status, string(body))
	}
}

func TestRedshiftStage6ClusterCredentialsWithIAM(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"iam-demo"},
		"NodeType":           []string{"dc2.large"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret123"},
	}
	status, body := redshiftRequest(t, ts, create)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create, got %d: %s", status, string(body))
	}

	creds := url.Values{
		"Action":            []string{"GetClusterCredentialsWithIAM"},
		"ClusterIdentifier": []string{"iam-demo"},
		"DbUser":            []string{"iamuser"},
	}
	status, body = redshiftRequest(t, ts, creds)
	if status != http.StatusOK {
		t.Fatalf("expected 200 get credentials iam, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DbUser>iamuser</DbUser>")) {
		t.Fatalf("missing DbUser: %s", string(body))
	}

	missingCluster := url.Values{
		"Action":            []string{"GetClusterCredentialsWithIAM"},
		"ClusterIdentifier": []string{"missing"},
		"DbUser":            []string{"iamuser"},
	}
	status, body = redshiftRequest(t, ts, missingCluster)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 missing cluster, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>ClusterNotFound</Code>")) {
		t.Fatalf("expected ClusterNotFound: %s", string(body))
	}
}
