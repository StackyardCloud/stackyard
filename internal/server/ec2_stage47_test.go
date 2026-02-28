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

func TestEC2Stage47SDKLifecycle(t *testing.T) {
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

	modifyEgressOut, err := client.ModifyVpcBlockPublicAccessExclusion(ctx, &awsec2.ModifyVpcBlockPublicAccessExclusionInput{
		ExclusionId:                  aws.String("vpcbpa-ex-00000000"),
		InternetGatewayExclusionMode: awsec2types.InternetGatewayExclusionModeAllowEgress,
	})
	if err != nil || modifyEgressOut.VpcBlockPublicAccessExclusion == nil {
		t.Fatalf("modify vpc block public access exclusion (allow-egress): %v", err)
	}
	if modifyEgressOut.VpcBlockPublicAccessExclusion.InternetGatewayExclusionMode != awsec2types.InternetGatewayExclusionModeAllowEgress {
		t.Fatalf("unexpected exclusion mode: %q", modifyEgressOut.VpcBlockPublicAccessExclusion.InternetGatewayExclusionMode)
	}
	if modifyEgressOut.VpcBlockPublicAccessExclusion.State != awsec2types.VpcBlockPublicAccessExclusionStateUpdateComplete {
		t.Fatalf("unexpected exclusion state: %q", modifyEgressOut.VpcBlockPublicAccessExclusion.State)
	}
	if aws.ToString(modifyEgressOut.VpcBlockPublicAccessExclusion.ExclusionId) != "vpcbpa-ex-00000000" {
		t.Fatalf("unexpected exclusion id: %q", aws.ToString(modifyEgressOut.VpcBlockPublicAccessExclusion.ExclusionId))
	}

	modifyBidirectionalOut, err := client.ModifyVpcBlockPublicAccessExclusion(ctx, &awsec2.ModifyVpcBlockPublicAccessExclusionInput{
		ExclusionId:                  aws.String("vpcbpa-ex-00000000"),
		InternetGatewayExclusionMode: awsec2types.InternetGatewayExclusionModeAllowBidirectional,
	})
	if err != nil || modifyBidirectionalOut.VpcBlockPublicAccessExclusion == nil {
		t.Fatalf("modify vpc block public access exclusion (allow-bidirectional): %v", err)
	}
	if modifyBidirectionalOut.VpcBlockPublicAccessExclusion.InternetGatewayExclusionMode != awsec2types.InternetGatewayExclusionModeAllowBidirectional {
		t.Fatalf("unexpected exclusion mode after reset: %q", modifyBidirectionalOut.VpcBlockPublicAccessExclusion.InternetGatewayExclusionMode)
	}
}

func TestEC2Stage47ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyVpcBlockPublicAccessExclusion",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"ExclusionId":                  "vpcbpa-ex-00000000",
			"InternetGatewayExclusionMode": "allow-bidirectional",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
