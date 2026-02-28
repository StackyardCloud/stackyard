package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRedshiftStage5DataShares(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	authorize := url.Values{
		"Action":             []string{"AuthorizeDataShare"},
		"DataShareName":      []string{"share-1"},
		"ConsumerIdentifier": []string{"consumer-1"},
	}
	status, body := redshiftRequest(t, ts, authorize)
	if status != http.StatusOK {
		t.Fatalf("expected 200 authorize, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DataShareName>share-1</DataShareName>")) {
		t.Fatalf("missing data share name: %s", string(body))
	}

	associate := url.Values{
		"Action":             []string{"AssociateDataShareConsumer"},
		"DataShareName":      []string{"share-1"},
		"ConsumerIdentifier": []string{"consumer-1"},
	}
	status, body = redshiftRequest(t, ts, associate)
	if status != http.StatusOK {
		t.Fatalf("expected 200 associate, got %d: %s", status, string(body))
	}

	describeConsumer := url.Values{
		"Action":             []string{"DescribeDataSharesForConsumer"},
		"ConsumerIdentifier": []string{"consumer-1"},
	}
	status, body = redshiftRequest(t, ts, describeConsumer)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe consumer, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DataShareName>share-1</DataShareName>")) {
		t.Fatalf("missing share in consumer view: %s", string(body))
	}

	describeProducer := url.Values{
		"Action": []string{"DescribeDataSharesForProducer"},
	}
	status, body = redshiftRequest(t, ts, describeProducer)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe producer, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DataShareName>share-1</DataShareName>")) {
		t.Fatalf("missing share in producer view: %s", string(body))
	}

	describeShare := url.Values{
		"Action":        []string{"DescribeDataShares"},
		"DataShareName": []string{"share-1"},
	}
	status, body = redshiftRequest(t, ts, describeShare)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe share, got %d: %s", status, string(body))
	}

	deauthorize := url.Values{
		"Action":             []string{"DeauthorizeDataShare"},
		"DataShareName":      []string{"share-1"},
		"ConsumerIdentifier": []string{"consumer-1"},
	}
	status, body = redshiftRequest(t, ts, deauthorize)
	if status != http.StatusOK {
		t.Fatalf("expected 200 deauthorize, got %d: %s", status, string(body))
	}

	reject := url.Values{
		"Action":             []string{"RejectDataShare"},
		"DataShareName":      []string{"share-1"},
		"ConsumerIdentifier": []string{"consumer-2"},
	}
	status, body = redshiftRequest(t, ts, reject)
	if status != http.StatusOK {
		t.Fatalf("expected 200 reject, got %d: %s", status, string(body))
	}

	unauthorizedAssociate := url.Values{
		"Action":             []string{"AssociateDataShareConsumer"},
		"DataShareName":      []string{"share-1"},
		"ConsumerIdentifier": []string{"consumer-3"},
	}
	status, body = redshiftRequest(t, ts, unauthorizedAssociate)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 unauthorized associate, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>InvalidDataShareState</Code>")) {
		t.Fatalf("expected InvalidDataShareState error: %s", string(body))
	}
}

func TestRedshiftStage5Integrations(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{
		"Action":          []string{"CreateIntegration"},
		"IntegrationName": []string{"int-1"},
		"SourceArn":       []string{"arn:aws:redshift:us-east-1:123456789012:cluster/source"},
		"TargetArn":       []string{"arn:aws:redshift:us-east-1:123456789012:namespace/target"},
		"Description":     []string{"demo"},
	}
	status, body := redshiftRequest(t, ts, create)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create integration, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<IntegrationName>int-1</IntegrationName>")) {
		t.Fatalf("missing integration name: %s", string(body))
	}
	if !bytes.Contains(body, []byte("<IntegrationArn>")) {
		t.Fatalf("missing integration arn: %s", string(body))
	}

	describe := url.Values{
		"Action":          []string{"DescribeIntegrations"},
		"IntegrationName": []string{"int-1"},
	}
	status, body = redshiftRequest(t, ts, describe)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe, got %d: %s", status, string(body))
	}

	modify := url.Values{
		"Action":          []string{"ModifyIntegration"},
		"IntegrationName": []string{"int-1"},
		"Description":     []string{"updated"},
	}
	status, body = redshiftRequest(t, ts, modify)
	if status != http.StatusOK {
		t.Fatalf("expected 200 modify, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Description>updated</Description>")) {
		t.Fatalf("missing updated description: %s", string(body))
	}

	inbound := url.Values{
		"Action": []string{"DescribeInboundIntegrations"},
	}
	status, body = redshiftRequest(t, ts, inbound)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe inbound, got %d: %s", status, string(body))
	}

	dup := url.Values{
		"Action":          []string{"CreateIntegration"},
		"IntegrationName": []string{"int-1"},
		"SourceArn":       []string{"arn:aws:redshift:us-east-1:123456789012:cluster/source"},
		"TargetArn":       []string{"arn:aws:redshift:us-east-1:123456789012:namespace/target"},
	}
	status, body = redshiftRequest(t, ts, dup)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 duplicate integration, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>IntegrationAlreadyExists</Code>")) {
		t.Fatalf("expected IntegrationAlreadyExists: %s", string(body))
	}

	deleteParams := url.Values{
		"Action":          []string{"DeleteIntegration"},
		"IntegrationName": []string{"int-1"},
	}
	status, body = redshiftRequest(t, ts, deleteParams)
	if status != http.StatusOK {
		t.Fatalf("expected 200 delete, got %d: %s", status, string(body))
	}

	status, body = redshiftRequest(t, ts, describe)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 describe deleted integration, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>IntegrationNotFound</Code>")) {
		t.Fatalf("expected IntegrationNotFound: %s", string(body))
	}
}
