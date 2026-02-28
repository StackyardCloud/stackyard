package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRedshiftStage8HsmLifecycle(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createCert := url.Values{
		"Action":                         []string{"CreateHsmClientCertificate"},
		"HsmClientCertificateIdentifier": []string{"cert-1"},
	}
	status, body := redshiftRequest(t, ts, createCert)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create cert, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<HsmClientCertificateIdentifier>cert-1</HsmClientCertificateIdentifier>")) {
		t.Fatalf("missing cert identifier: %s", string(body))
	}

	describeCerts := url.Values{
		"Action": []string{"DescribeHsmClientCertificates"},
	}
	status, body = redshiftRequest(t, ts, describeCerts)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe certs, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<HsmClientCertificateIdentifier>cert-1</HsmClientCertificateIdentifier>")) {
		t.Fatalf("expected cert in describe: %s", string(body))
	}

	deleteCert := url.Values{
		"Action":                         []string{"DeleteHsmClientCertificate"},
		"HsmClientCertificateIdentifier": []string{"cert-1"},
	}
	status, body = redshiftRequest(t, ts, deleteCert)
	if status != http.StatusOK {
		t.Fatalf("expected 200 delete cert, got %d: %s", status, string(body))
	}

	status, body = redshiftRequest(t, ts, describeCerts)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe certs after delete, got %d: %s", status, string(body))
	}
	if bytes.Contains(body, []byte("<HsmClientCertificateIdentifier>cert-1</HsmClientCertificateIdentifier>")) {
		t.Fatalf("expected cert removed: %s", string(body))
	}

	createConfig := url.Values{
		"Action":                     []string{"CreateHsmConfiguration"},
		"HsmConfigurationIdentifier": []string{"cfg-1"},
		"HsmIpAddress":               []string{"10.0.0.10"},
		"HsmPartitionName":           []string{"p1"},
		"HsmPartitionPassword":       []string{"Secret123"},
		"HsmServerPublicCertificate": []string{"cert-data"},
	}
	status, body = redshiftRequest(t, ts, createConfig)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create config, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<HsmConfigurationIdentifier>cfg-1</HsmConfigurationIdentifier>")) {
		t.Fatalf("missing config identifier: %s", string(body))
	}

	describeConfigs := url.Values{
		"Action": []string{"DescribeHsmConfigurations"},
	}
	status, body = redshiftRequest(t, ts, describeConfigs)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe configs, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<HsmConfigurationIdentifier>cfg-1</HsmConfigurationIdentifier>")) {
		t.Fatalf("expected config in describe: %s", string(body))
	}

	deleteConfig := url.Values{
		"Action":                     []string{"DeleteHsmConfiguration"},
		"HsmConfigurationIdentifier": []string{"cfg-1"},
	}
	status, body = redshiftRequest(t, ts, deleteConfig)
	if status != http.StatusOK {
		t.Fatalf("expected 200 delete config, got %d: %s", status, string(body))
	}

	status, body = redshiftRequest(t, ts, describeConfigs)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe configs after delete, got %d: %s", status, string(body))
	}
	if bytes.Contains(body, []byte("<HsmConfigurationIdentifier>cfg-1</HsmConfigurationIdentifier>")) {
		t.Fatalf("expected config removed: %s", string(body))
	}
}

func TestRedshiftStage8LoggingLifecycle(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createCluster := url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"log-demo"},
		"NodeType":           []string{"dc2.large"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret123"},
		"DBName":             []string{"dev"},
	}
	status, body := redshiftRequest(t, ts, createCluster)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create cluster, got %d: %s", status, string(body))
	}

	enable := url.Values{
		"Action":            []string{"EnableLogging"},
		"ClusterIdentifier": []string{"log-demo"},
		"BucketName":        []string{"logs-bucket"},
		"S3KeyPrefix":       []string{"redshift/"},
	}
	status, body = redshiftRequest(t, ts, enable)
	if status != http.StatusOK {
		t.Fatalf("expected 200 enable logging, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<LoggingEnabled>true</LoggingEnabled>")) {
		t.Fatalf("expected logging enabled: %s", string(body))
	}
	if !bytes.Contains(body, []byte("<BucketName>logs-bucket</BucketName>")) {
		t.Fatalf("missing bucket name: %s", string(body))
	}

	describe := url.Values{
		"Action":            []string{"DescribeLoggingStatus"},
		"ClusterIdentifier": []string{"log-demo"},
	}
	status, body = redshiftRequest(t, ts, describe)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe logging, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<LoggingEnabled>true</LoggingEnabled>")) {
		t.Fatalf("expected logging enabled in describe: %s", string(body))
	}

	disable := url.Values{
		"Action":            []string{"DisableLogging"},
		"ClusterIdentifier": []string{"log-demo"},
	}
	status, body = redshiftRequest(t, ts, disable)
	if status != http.StatusOK {
		t.Fatalf("expected 200 disable logging, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<LoggingEnabled>false</LoggingEnabled>")) {
		t.Fatalf("expected logging disabled: %s", string(body))
	}

	status, body = redshiftRequest(t, ts, describe)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe logging after disable, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<LoggingEnabled>false</LoggingEnabled>")) {
		t.Fatalf("expected logging disabled in describe: %s", string(body))
	}
}

func TestRedshiftStage8EventSubscriptionLifecycle(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createCluster := url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"event-demo"},
		"NodeType":           []string{"dc2.large"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret123"},
		"DBName":             []string{"dev"},
	}
	status, body := redshiftRequest(t, ts, createCluster)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create cluster, got %d: %s", status, string(body))
	}

	create := url.Values{
		"Action":             []string{"CreateEventSubscription"},
		"SubscriptionName":   []string{"sub-1"},
		"SnsTopicArn":        []string{"arn:aws:sns:us-east-1:123456789012:topic-1"},
		"SourceType":         []string{"cluster"},
		"SourceIds.member.1": []string{"event-demo"},
	}
	status, body = redshiftRequest(t, ts, create)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create subscription, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<CustSubscriptionId>sub-1</CustSubscriptionId>")) {
		t.Fatalf("missing subscription id: %s", string(body))
	}
	if !bytes.Contains(body, []byte("<Severity>INFO</Severity>")) {
		t.Fatalf("expected default severity: %s", string(body))
	}

	modify := url.Values{
		"Action":           []string{"ModifyEventSubscription"},
		"SubscriptionName": []string{"sub-1"},
		"Enabled":          []string{"false"},
	}
	status, body = redshiftRequest(t, ts, modify)
	if status != http.StatusOK {
		t.Fatalf("expected 200 modify subscription, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Enabled>false</Enabled>")) {
		t.Fatalf("expected enabled false: %s", string(body))
	}

	describe := url.Values{
		"Action": []string{"DescribeEventSubscriptions"},
	}
	status, body = redshiftRequest(t, ts, describe)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe subscriptions, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<CustSubscriptionId>sub-1</CustSubscriptionId>")) {
		t.Fatalf("missing subscription in describe: %s", string(body))
	}

	deleteSub := url.Values{
		"Action":           []string{"DeleteEventSubscription"},
		"SubscriptionName": []string{"sub-1"},
	}
	status, body = redshiftRequest(t, ts, deleteSub)
	if status != http.StatusOK {
		t.Fatalf("expected 200 delete subscription, got %d: %s", status, string(body))
	}

	status, body = redshiftRequest(t, ts, describe)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe after delete, got %d: %s", status, string(body))
	}
	if bytes.Contains(body, []byte("<CustSubscriptionId>sub-1</CustSubscriptionId>")) {
		t.Fatalf("expected subscription removed: %s", string(body))
	}
}

func TestRedshiftStage8EventSubscriptionInvalidSourceType(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{
		"Action":           []string{"CreateEventSubscription"},
		"SubscriptionName": []string{"bad-sub"},
		"SnsTopicArn":      []string{"arn:aws:sns:us-east-1:123456789012:topic-1"},
		"SourceType":       []string{"not-a-type"},
	}
	status, body := redshiftRequest(t, ts, create)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid source type, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>InvalidParameterValue</Code>")) {
		t.Fatalf("expected InvalidParameterValue: %s", string(body))
	}
}

func TestRedshiftStage8ResourcePolicyArnValidation(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	badRegion := url.Values{
		"Action":      []string{"PutResourcePolicy"},
		"ResourceArn": []string{"arn:aws:redshift:us-west-2:123456789012:cluster:bad"},
		"Policy":      []string{`{"Version":"2012-10-17","Statement":[]}`},
	}
	status, body := redshiftRequest(t, ts, badRegion)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid region, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>InvalidParameterValue</Code>")) {
		t.Fatalf("expected InvalidParameterValue: %s", string(body))
	}

	badArn := url.Values{
		"Action":      []string{"PutResourcePolicy"},
		"ResourceArn": []string{"not-an-arn"},
		"Policy":      []string{`{"Version":"2012-10-17","Statement":[]}`},
	}
	status, body = redshiftRequest(t, ts, badArn)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid arn, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>InvalidParameterValue</Code>")) {
		t.Fatalf("expected InvalidParameterValue: %s", string(body))
	}
}
