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

func TestEC2Stage81SDKLifecycle(t *testing.T) {
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

	out, err := client.AllocateIpamPoolCidr(ctx, &awsec2.AllocateIpamPoolCidrInput{
		IpamPoolId:  aws.String("ipam-pool-00000081"),
		Cidr:        aws.String("10.81.0.0/24"),
		Description: aws.String("stage81"),
	})
	if err != nil {
		t.Fatalf("allocate ipam pool cidr: %v", err)
	}
	if out.IpamPoolAllocation == nil {
		t.Fatalf("expected ipam pool allocation in response")
	}
	if aws.ToString(out.IpamPoolAllocation.Cidr) != "10.81.0.0/24" {
		t.Fatalf("unexpected ipam pool allocation cidr: %q", aws.ToString(out.IpamPoolAllocation.Cidr))
	}
	if aws.ToString(out.IpamPoolAllocation.Description) != "stage81" {
		t.Fatalf("unexpected ipam pool allocation description: %q", aws.ToString(out.IpamPoolAllocation.Description))
	}
	if aws.ToString(out.IpamPoolAllocation.IpamPoolAllocationId) == "" {
		t.Fatalf("expected non-empty ipam pool allocation id")
	}
	if aws.ToString(out.IpamPoolAllocation.ResourceId) != "ipam-pool-00000081" {
		t.Fatalf("unexpected ipam pool allocation resource id: %q", aws.ToString(out.IpamPoolAllocation.ResourceId))
	}
	if out.IpamPoolAllocation.ResourceType != awsec2types.IpamPoolAllocationResourceTypeIpamPool {
		t.Fatalf("unexpected ipam pool allocation resource type: %q", out.IpamPoolAllocation.ResourceType)
	}
}

func TestEC2Stage81ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AllocateIpamPoolCidr",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"IpamPoolId":  "ipam-pool-00000081",
			"Cidr":        "10.81.0.0/24",
			"Description": "stage81",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
