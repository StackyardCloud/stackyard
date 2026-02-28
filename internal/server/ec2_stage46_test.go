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

func TestEC2Stage46SDKLifecycle(t *testing.T) {
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

	modifyIngressOut, err := client.ModifyVpcBlockPublicAccessOptions(ctx, &awsec2.ModifyVpcBlockPublicAccessOptionsInput{
		InternetGatewayBlockMode: awsec2types.InternetGatewayBlockModeBlockIngress,
	})
	if err != nil || modifyIngressOut.VpcBlockPublicAccessOptions == nil {
		t.Fatalf("modify vpc block public access options (block-ingress): %v", err)
	}
	if modifyIngressOut.VpcBlockPublicAccessOptions.InternetGatewayBlockMode != awsec2types.InternetGatewayBlockModeBlockIngress {
		t.Fatalf("unexpected internet gateway block mode: %q", modifyIngressOut.VpcBlockPublicAccessOptions.InternetGatewayBlockMode)
	}
	if modifyIngressOut.VpcBlockPublicAccessOptions.State != awsec2types.VpcBlockPublicAccessStateUpdateComplete {
		t.Fatalf("unexpected vpc block public access state: %q", modifyIngressOut.VpcBlockPublicAccessOptions.State)
	}
	if modifyIngressOut.VpcBlockPublicAccessOptions.ManagedBy != awsec2types.ManagedByAccount {
		t.Fatalf("unexpected managed-by value: %q", modifyIngressOut.VpcBlockPublicAccessOptions.ManagedBy)
	}

	modifyOffOut, err := client.ModifyVpcBlockPublicAccessOptions(ctx, &awsec2.ModifyVpcBlockPublicAccessOptionsInput{
		InternetGatewayBlockMode: awsec2types.InternetGatewayBlockModeOff,
	})
	if err != nil || modifyOffOut.VpcBlockPublicAccessOptions == nil {
		t.Fatalf("modify vpc block public access options (off): %v", err)
	}
	if modifyOffOut.VpcBlockPublicAccessOptions.InternetGatewayBlockMode != awsec2types.InternetGatewayBlockModeOff {
		t.Fatalf("unexpected internet gateway block mode after reset: %q", modifyOffOut.VpcBlockPublicAccessOptions.InternetGatewayBlockMode)
	}
}

func TestEC2Stage46ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyVpcBlockPublicAccessOptions",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"InternetGatewayBlockMode": "off",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
