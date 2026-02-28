package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awslightsail "github.com/aws/aws-sdk-go-v2/service/lightsail"
	awslightsailtypes "github.com/aws/aws-sdk-go-v2/service/lightsail/types"
)

func TestLightsailStage11LoadBalancerTLS(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateInstances", []byte(`{"availabilityZone":"us-east-1a","blueprintId":"amazon_linux_2","bundleId":"micro_2_0","instanceNames":["stage11-instance"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateLoadBalancerTlsCertificate", []byte(`{"loadBalancerName":"stage11-lb","certificateName":"stage11-cert","certificateDomainName":"example.com","certificateAlternativeNames":["www.example.com"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetLoadBalancerTlsCertificates", []byte(`{"loadBalancerName":"stage11-lb"}`))
	assertStatus(t, resp, http.StatusOK)
	var getCertsOut struct {
		TLSCertificates []struct {
			Name       string `json:"name"`
			IsAttached bool   `json:"isAttached"`
		} `json:"tlsCertificates"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getCertsOut); err != nil {
		t.Fatalf("unmarshal GetLoadBalancerTlsCertificates: %v", err)
	}
	if len(getCertsOut.TLSCertificates) != 1 || getCertsOut.TLSCertificates[0].Name != "stage11-cert" || getCertsOut.TLSCertificates[0].IsAttached {
		t.Fatalf("unexpected GetLoadBalancerTlsCertificates output: %+v", getCertsOut)
	}

	resp = lightsailRequest(t, ts, "AttachLoadBalancerTlsCertificate", []byte(`{"loadBalancerName":"stage11-lb","certificateName":"stage11-cert"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetLoadBalancerTlsPolicies", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var getPoliciesOut struct {
		TLSPolicies []struct {
			Name      string `json:"name"`
			IsDefault bool   `json:"isDefault"`
		} `json:"tlsPolicies"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getPoliciesOut); err != nil {
		t.Fatalf("unmarshal GetLoadBalancerTlsPolicies: %v", err)
	}
	if len(getPoliciesOut.TLSPolicies) == 0 {
		t.Fatalf("expected at least one tls policy")
	}

	resp = lightsailRequest(t, ts, "DeleteLoadBalancerTlsCertificate", []byte(`{"loadBalancerName":"stage11-lb","certificateName":"stage11-cert","force":true}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "SetupInstanceHttps", []byte(`{"certificateProvider":"LetsEncrypt","domainNames":["example.com"],"emailAddress":"admin@example.com","instanceName":"stage11-instance"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage11SDKClientLoadBalancerTLS(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}

	client := awslightsail.NewFromConfig(cfg, func(o *awslightsail.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	if _, err := client.CreateInstances(ctx, &awslightsail.CreateInstancesInput{
		AvailabilityZone: aws.String("us-east-1a"),
		BlueprintId:      aws.String("amazon_linux_2"),
		BundleId:         aws.String("micro_2_0"),
		InstanceNames:    []string{"sdk-stage11-instance"},
	}); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	if _, err := client.CreateLoadBalancerTlsCertificate(ctx, &awslightsail.CreateLoadBalancerTlsCertificateInput{
		LoadBalancerName:            aws.String("sdk-stage11-lb"),
		CertificateName:             aws.String("sdk-stage11-cert"),
		CertificateDomainName:       aws.String("example.com"),
		CertificateAlternativeNames: []string{"www.example.com"},
	}); err != nil {
		t.Fatalf("create load balancer tls certificate: %v", err)
	}

	certsOut, err := client.GetLoadBalancerTlsCertificates(ctx, &awslightsail.GetLoadBalancerTlsCertificatesInput{
		LoadBalancerName: aws.String("sdk-stage11-lb"),
	})
	if err != nil {
		t.Fatalf("get load balancer tls certificates: %v", err)
	}
	if len(certsOut.TlsCertificates) != 1 {
		t.Fatalf("expected one tls certificate, got %d", len(certsOut.TlsCertificates))
	}

	if _, err := client.AttachLoadBalancerTlsCertificate(ctx, &awslightsail.AttachLoadBalancerTlsCertificateInput{
		LoadBalancerName: aws.String("sdk-stage11-lb"),
		CertificateName:  aws.String("sdk-stage11-cert"),
	}); err != nil {
		t.Fatalf("attach load balancer tls certificate: %v", err)
	}

	policiesOut, err := client.GetLoadBalancerTlsPolicies(ctx, &awslightsail.GetLoadBalancerTlsPoliciesInput{})
	if err != nil {
		t.Fatalf("get load balancer tls policies: %v", err)
	}
	if len(policiesOut.TlsPolicies) == 0 {
		t.Fatalf("expected tls policies")
	}

	if _, err := client.DeleteLoadBalancerTlsCertificate(ctx, &awslightsail.DeleteLoadBalancerTlsCertificateInput{
		LoadBalancerName: aws.String("sdk-stage11-lb"),
		CertificateName:  aws.String("sdk-stage11-cert"),
		Force:            aws.Bool(true),
	}); err != nil {
		t.Fatalf("delete load balancer tls certificate: %v", err)
	}

	if _, err := client.SetupInstanceHttps(ctx, &awslightsail.SetupInstanceHttpsInput{
		CertificateProvider: awslightsailtypes.CertificateProviderLetsEncrypt,
		DomainNames:         []string{"example.com"},
		EmailAddress:        aws.String("admin@example.com"),
		InstanceName:        aws.String("sdk-stage11-instance"),
	}); err != nil {
		t.Fatalf("setup instance https: %v", err)
	}
}
