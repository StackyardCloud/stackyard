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

func TestLightsailStage13CertificatesAndDistributionAttach(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateDistribution", []byte(`{
		"distributionName":"stage13-dist",
		"bundleId":"small_1_0",
		"defaultCacheBehavior":{"behavior":"cache"},
		"origin":{"name":"stage13-origin","protocolPolicy":"http-only","regionName":"us-east-1"}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateCertificate", []byte(`{
		"certificateName":"stage13-cert",
		"domainName":"example.com",
		"subjectAlternativeNames":["www.example.com"]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetCertificates", []byte(`{"certificateName":"stage13-cert","includeCertificateDetails":true}`))
	assertStatus(t, resp, http.StatusOK)
	var getOut struct {
		Certificates []struct {
			CertificateName string `json:"certificateName"`
			DomainName      string `json:"domainName"`
		} `json:"certificates"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getOut); err != nil {
		t.Fatalf("unmarshal GetCertificates: %v", err)
	}
	if len(getOut.Certificates) != 1 || getOut.Certificates[0].CertificateName != "stage13-cert" {
		t.Fatalf("unexpected GetCertificates output: %+v", getOut)
	}

	resp = lightsailRequest(t, ts, "AttachCertificateToDistribution", []byte(`{"certificateName":"stage13-cert","distributionName":"stage13-dist"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "SetIpAddressType", []byte(`{"resourceName":"stage13-dist","resourceType":"Distribution","ipAddressType":"ipv4"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DetachCertificateFromDistribution", []byte(`{"distributionName":"stage13-dist"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DeleteCertificate", []byte(`{"certificateName":"stage13-cert"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage13SDKClientCertificatesAndDistributionAttach(t *testing.T) {
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

	if _, err := client.CreateDistribution(ctx, &awslightsail.CreateDistributionInput{
		DistributionName:     aws.String("sdk-stage13-dist"),
		BundleId:             aws.String("small_1_0"),
		DefaultCacheBehavior: &awslightsailtypes.CacheBehavior{Behavior: awslightsailtypes.BehaviorEnum("cache")},
		Origin: &awslightsailtypes.InputOrigin{
			Name:           aws.String("sdk-stage13-origin"),
			ProtocolPolicy: awslightsailtypes.OriginProtocolPolicyEnum("http-only"),
			RegionName:     awslightsailtypes.RegionName("us-east-1"),
		},
	}); err != nil {
		t.Fatalf("create distribution: %v", err)
	}

	createOut, err := client.CreateCertificate(ctx, &awslightsail.CreateCertificateInput{
		CertificateName:         aws.String("sdk-stage13-cert"),
		DomainName:              aws.String("example.com"),
		SubjectAlternativeNames: []string{"www.example.com"},
	})
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	if createOut.Certificate == nil || createOut.Certificate.CertificateName == nil || *createOut.Certificate.CertificateName != "sdk-stage13-cert" {
		t.Fatalf("unexpected create certificate output: %+v", createOut.Certificate)
	}

	certsOut, err := client.GetCertificates(ctx, &awslightsail.GetCertificatesInput{
		CertificateName:           aws.String("sdk-stage13-cert"),
		IncludeCertificateDetails: true,
	})
	if err != nil {
		t.Fatalf("get certificates: %v", err)
	}
	if len(certsOut.Certificates) != 1 {
		t.Fatalf("expected one certificate, got %d", len(certsOut.Certificates))
	}

	if _, err := client.AttachCertificateToDistribution(ctx, &awslightsail.AttachCertificateToDistributionInput{
		CertificateName:  aws.String("sdk-stage13-cert"),
		DistributionName: aws.String("sdk-stage13-dist"),
	}); err != nil {
		t.Fatalf("attach certificate to distribution: %v", err)
	}

	if _, err := client.SetIpAddressType(ctx, &awslightsail.SetIpAddressTypeInput{
		ResourceName:  aws.String("sdk-stage13-dist"),
		ResourceType:  awslightsailtypes.ResourceTypeDistribution,
		IpAddressType: awslightsailtypes.IpAddressTypeIpv4,
	}); err != nil {
		t.Fatalf("set distribution ip address type: %v", err)
	}

	if _, err := client.CreateInstances(ctx, &awslightsail.CreateInstancesInput{
		AvailabilityZone: aws.String("us-east-1a"),
		BlueprintId:      aws.String("amazon_linux_2"),
		BundleId:         aws.String("micro_2_0"),
		InstanceNames:    []string{"sdk-stage13-instance"},
	}); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if _, err := client.SetIpAddressType(ctx, &awslightsail.SetIpAddressTypeInput{
		ResourceName:       aws.String("sdk-stage13-instance"),
		ResourceType:       awslightsailtypes.ResourceTypeInstance,
		IpAddressType:      awslightsailtypes.IpAddressTypeIpv6,
		AcceptBundleUpdate: aws.Bool(true),
	}); err != nil {
		t.Fatalf("set instance ip address type: %v", err)
	}

	if _, err := client.DetachCertificateFromDistribution(ctx, &awslightsail.DetachCertificateFromDistributionInput{
		DistributionName: aws.String("sdk-stage13-dist"),
	}); err != nil {
		t.Fatalf("detach certificate from distribution: %v", err)
	}

	if _, err := client.DeleteCertificate(ctx, &awslightsail.DeleteCertificateInput{
		CertificateName: aws.String("sdk-stage13-cert"),
	}); err != nil {
		t.Fatalf("delete certificate: %v", err)
	}
}
