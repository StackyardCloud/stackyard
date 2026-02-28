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

func TestEC2Stage85SDKLifecycle(t *testing.T) {
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

	out, err := client.AssociateIpamByoasn(ctx, &awsec2.AssociateIpamByoasnInput{
		Asn:  aws.String("64585"),
		Cidr: aws.String("198.51.85.0/24"),
	})
	if err != nil {
		t.Fatalf("associate ipam byoasn: %v", err)
	}
	if out.AsnAssociation == nil {
		t.Fatalf("expected asn association in response")
	}
	if aws.ToString(out.AsnAssociation.Asn) != "64585" {
		t.Fatalf("unexpected asn: %q", aws.ToString(out.AsnAssociation.Asn))
	}
	if aws.ToString(out.AsnAssociation.Cidr) != "198.51.85.0/24" {
		t.Fatalf("unexpected cidr: %q", aws.ToString(out.AsnAssociation.Cidr))
	}
	if out.AsnAssociation.State != awsec2types.AsnAssociationStateAssociated {
		t.Fatalf("unexpected asn association state: %q", out.AsnAssociation.State)
	}
}

func TestEC2Stage85ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AssociateIpamByoasn",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"Asn":  "64585",
			"Cidr": "198.51.85.0/24",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
