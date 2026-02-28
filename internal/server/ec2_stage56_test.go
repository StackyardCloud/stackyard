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

func TestEC2Stage56SDKLifecycle(t *testing.T) {
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

	out, err := client.CreateVpcEndpoint(ctx, &awsec2.CreateVpcEndpointInput{
		VpcId:             aws.String("vpc-00000001"),
		ServiceName:       aws.String("com.amazonaws.us-east-1.s3"),
		VpcEndpointType:   awsec2types.VpcEndpointTypeInterface,
		SubnetIds:         []string{"subnet-00000001"},
		SecurityGroupIds:  []string{"sg-00000000"},
		PrivateDnsEnabled: aws.Bool(true),
		ClientToken:       aws.String("stage56-token"),
	})
	if err != nil {
		t.Fatalf("create vpc endpoint: %v", err)
	}
	if out.VpcEndpoint == nil {
		t.Fatalf("expected vpc endpoint in response")
	}
	if aws.ToString(out.VpcEndpoint.VpcId) != "vpc-00000001" {
		t.Fatalf("unexpected vpc id: %q", aws.ToString(out.VpcEndpoint.VpcId))
	}
	if aws.ToString(out.VpcEndpoint.ServiceName) != "com.amazonaws.us-east-1.s3" {
		t.Fatalf("unexpected service name: %q", aws.ToString(out.VpcEndpoint.ServiceName))
	}
	if out.VpcEndpoint.State != awsec2types.StateAvailable {
		t.Fatalf("unexpected endpoint state: %q", out.VpcEndpoint.State)
	}
	if aws.ToString(out.ClientToken) != "stage56-token" {
		t.Fatalf("unexpected client token: %q", aws.ToString(out.ClientToken))
	}
}

func TestEC2Stage56ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateVpcEndpoint",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"VpcId":             "vpc-00000001",
			"ServiceName":       "com.amazonaws.us-east-1.s3",
			"VpcEndpointType":   "Interface",
			"SubnetId.1":        "subnet-00000001",
			"SecurityGroupId.1": "sg-00000000",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
