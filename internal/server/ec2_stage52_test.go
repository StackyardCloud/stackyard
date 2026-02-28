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

func TestEC2Stage52SDKLifecycle(t *testing.T) {
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

	modifyOut, err := client.ModifyVpcEndpoint(ctx, &awsec2.ModifyVpcEndpointInput{
		VpcEndpointId:       aws.String("vpce-00000000"),
		AddRouteTableIds:    []string{"rtb-00000002"},
		AddSecurityGroupIds: []string{"sg-00000001"},
		AddSubnetIds:        []string{"subnet-00000002"},
		SubnetConfigurations: []awsec2types.SubnetConfiguration{
			{SubnetId: aws.String("subnet-00000003")},
		},
		IpAddressType:     awsec2types.IpAddressTypeDualstack,
		PolicyDocument:    aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"*","Resource":"*"}]}`),
		PrivateDnsEnabled: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("modify vpc endpoint: %v", err)
	}
	if !aws.ToBool(modifyOut.Return) {
		t.Fatalf("expected modify vpc endpoint return value true")
	}

	removeOut, err := client.ModifyVpcEndpoint(ctx, &awsec2.ModifyVpcEndpointInput{
		VpcEndpointId:          aws.String("vpce-00000000"),
		RemoveRouteTableIds:    []string{"rtb-00000002"},
		RemoveSecurityGroupIds: []string{"sg-00000001"},
		RemoveSubnetIds:        []string{"subnet-00000002"},
		ResetPolicy:            aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("modify vpc endpoint remove values: %v", err)
	}
	if !aws.ToBool(removeOut.Return) {
		t.Fatalf("expected modify vpc endpoint remove values return value true")
	}
}

func TestEC2Stage52ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyVpcEndpoint",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"VpcEndpointId":     "vpce-00000000",
			"AddRouteTableId.1": "rtb-00000002",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
