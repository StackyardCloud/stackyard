package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestRedshiftStage7TagLifecycle(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resource := "arn:aws:redshift:us-east-1:123456789012:cluster:tag-demo"

	create := url.Values{
		"Action":              []string{"CreateTags"},
		"ResourceName":        []string{resource},
		"Tags.member.1.Key":   []string{"env"},
		"Tags.member.1.Value": []string{"dev"},
		"Tags.member.2.Key":   []string{"team"},
		"Tags.member.2.Value": []string{"core"},
	}
	status, body := redshiftRequest(t, ts, create)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create tags, got %d: %s", status, string(body))
	}

	describe := url.Values{
		"Action":       []string{"DescribeTags"},
		"ResourceName": []string{resource},
	}
	status, body = redshiftRequest(t, ts, describe)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe tags, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Key>env</Key>")) || !bytes.Contains(body, []byte("<Value>dev</Value>")) {
		t.Fatalf("missing env tag: %s", string(body))
	}
	if !bytes.Contains(body, []byte("<Key>team</Key>")) || !bytes.Contains(body, []byte("<Value>core</Value>")) {
		t.Fatalf("missing team tag: %s", string(body))
	}

	deleteTags := url.Values{
		"Action":           []string{"DeleteTags"},
		"ResourceName":     []string{resource},
		"TagKeys.member.1": []string{"env"},
	}
	status, body = redshiftRequest(t, ts, deleteTags)
	if status != http.StatusOK {
		t.Fatalf("expected 200 delete tags, got %d: %s", status, string(body))
	}
	status, body = redshiftRequest(t, ts, describe)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe tags, got %d: %s", status, string(body))
	}
	if bytes.Contains(body, []byte("<Key>env</Key>")) {
		t.Fatalf("expected env tag removed: %s", string(body))
	}
	if !bytes.Contains(body, []byte("<Key>team</Key>")) {
		t.Fatalf("expected team tag present: %s", string(body))
	}
}

func TestRedshiftStage7TagValidation(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resource := "arn:aws:redshift:us-east-1:123456789012:cluster:tag-bad"
	bad := url.Values{
		"Action":              []string{"CreateTags"},
		"ResourceName":        []string{resource},
		"Tags.member.1.Key":   []string{"aws:reserved"},
		"Tags.member.1.Value": []string{"bad"},
	}
	status, body := redshiftRequest(t, ts, bad)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 reserved key, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>InvalidParameterValue</Code>")) {
		t.Fatalf("expected InvalidParameterValue: %s", string(body))
	}

	tooMany := url.Values{
		"Action":       []string{"CreateTags"},
		"ResourceName": []string{resource},
	}
	for i := 1; i <= 51; i++ {
		tooMany.Set("Tags.member."+strconv.Itoa(i)+".Key", "k"+strconv.Itoa(i))
		tooMany.Set("Tags.member."+strconv.Itoa(i)+".Value", "v"+strconv.Itoa(i))
	}
	status, body = redshiftRequest(t, ts, tooMany)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 too many tags, got %d: %s", status, string(body))
	}
}

func TestRedshiftStage7ResourcePolicyLifecycle(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceArn := "arn:aws:redshift:us-east-1:123456789012:cluster:policy-demo"
	policy := `{"Version":"2012-10-17","Statement":[]}`

	put := url.Values{
		"Action":      []string{"PutResourcePolicy"},
		"ResourceArn": []string{resourceArn},
		"Policy":      []string{policy},
	}
	status, body := redshiftRequest(t, ts, put)
	if status != http.StatusOK {
		t.Fatalf("expected 200 put policy, got %d: %s", status, string(body))
	}

	get := url.Values{
		"Action":      []string{"GetResourcePolicy"},
		"ResourceArn": []string{resourceArn},
	}
	status, body = redshiftRequest(t, ts, get)
	if status != http.StatusOK {
		t.Fatalf("expected 200 get policy, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte(policy)) {
		t.Fatalf("missing policy in response: %s", string(body))
	}

	del := url.Values{
		"Action":      []string{"DeleteResourcePolicy"},
		"ResourceArn": []string{resourceArn},
	}
	status, body = redshiftRequest(t, ts, del)
	if status != http.StatusOK {
		t.Fatalf("expected 200 delete policy, got %d: %s", status, string(body))
	}

	status, body = redshiftRequest(t, ts, get)
	if status != http.StatusNotFound {
		t.Fatalf("expected 404 get after delete, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>ResourcePolicyNotFound</Code>")) {
		t.Fatalf("expected ResourcePolicyNotFound: %s", string(body))
	}
}
