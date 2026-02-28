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

func TestEC2Stage45SDKLifecycle(t *testing.T) {
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

	createVpcOut, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{
		CidrBlock: aws.String("10.45.0.0/16"),
	})
	if err != nil || createVpcOut.Vpc == nil || createVpcOut.Vpc.VpcId == nil {
		t.Fatalf("create vpc: %v", err)
	}
	vpcID := aws.ToString(createVpcOut.Vpc.VpcId)

	modifyTenancyOut, err := client.ModifyVpcTenancy(ctx, &awsec2.ModifyVpcTenancyInput{
		VpcId:           aws.String(vpcID),
		InstanceTenancy: awsec2types.VpcTenancyDefault,
	})
	if err != nil {
		t.Fatalf("modify vpc tenancy: %v", err)
	}
	if !aws.ToBool(modifyTenancyOut.ReturnValue) {
		t.Fatalf("expected modify vpc tenancy return value to be true")
	}

	describeOut, err := client.DescribeVpcs(ctx, &awsec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	})
	if err != nil || len(describeOut.Vpcs) != 1 {
		t.Fatalf("describe vpcs: %v", err)
	}
	if describeOut.Vpcs[0].InstanceTenancy != awsec2types.TenancyDefault {
		t.Fatalf("unexpected vpc instance tenancy: %q", describeOut.Vpcs[0].InstanceTenancy)
	}
}

func TestEC2Stage45ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyVpcTenancy",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"VpcId":           "vpc-00000045",
			"InstanceTenancy": "default",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
