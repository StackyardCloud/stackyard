package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage79SDKLifecycle(t *testing.T) {
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
	client := awsec2.NewFromConfig(cfg, func(o *awsec2.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	out, err := client.AdvertiseByoipCidr(ctx, &awsec2.AdvertiseByoipCidrInput{
		Cidr:               aws.String("198.51.100.0/24"),
		Asn:                aws.String("64512"),
		NetworkBorderGroup: aws.String("us-east-1"),
	})
	if err != nil {
		t.Fatalf("advertise byoip cidr: %v", err)
	}
	if out.ByoipCidr == nil {
		t.Fatalf("expected byoip cidr in response")
	}
	if aws.ToString(out.ByoipCidr.Cidr) != "198.51.100.0/24" {
		t.Fatalf("unexpected byoip cidr: %q", aws.ToString(out.ByoipCidr.Cidr))
	}
	if out.ByoipCidr.State != awsec2types.ByoipCidrStateAdvertised {
		t.Fatalf("unexpected byoip cidr state: %q", out.ByoipCidr.State)
	}
	if aws.ToString(out.ByoipCidr.NetworkBorderGroup) != "us-east-1" {
		t.Fatalf("unexpected network border group: %q", aws.ToString(out.ByoipCidr.NetworkBorderGroup))
	}
	if len(out.ByoipCidr.AsnAssociations) != 1 {
		t.Fatalf("expected one ASN association, got %d", len(out.ByoipCidr.AsnAssociations))
	}
	if aws.ToString(out.ByoipCidr.AsnAssociations[0].Asn) != "64512" {
		t.Fatalf("unexpected ASN association value: %q", aws.ToString(out.ByoipCidr.AsnAssociations[0].Asn))
	}
}

func TestEC2Stage79ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AdvertiseByoipCidr",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"Cidr":               "198.51.100.0/24",
			"Asn":                "64512",
			"NetworkBorderGroup": "us-east-1",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
