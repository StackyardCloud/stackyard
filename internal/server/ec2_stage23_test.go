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
)

func TestEC2Stage23SDKLifecycle(t *testing.T) {
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
		CidrBlock: aws.String("10.23.0.0/16"),
	})
	if err != nil || createVpcOut.Vpc == nil || createVpcOut.Vpc.VpcId == nil {
		t.Fatalf("create vpc: %v", err)
	}
	vpcID := aws.ToString(createVpcOut.Vpc.VpcId)

	createSecurityGroupOut, err := client.CreateSecurityGroup(ctx, &awsec2.CreateSecurityGroupInput{
		GroupName:   aws.String("stage23-sg"),
		Description: aws.String("stage23 classic link"),
		VpcId:       aws.String(vpcID),
	})
	if err != nil || createSecurityGroupOut.GroupId == nil {
		t.Fatalf("create security group: %v", err)
	}
	groupID := aws.ToString(createSecurityGroupOut.GroupId)

	runInstancesOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-stage23"),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		InstanceType: "t3.micro",
	})
	if err != nil || len(runInstancesOut.Instances) != 1 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	enableVpcClassicLinkOut, err := client.EnableVpcClassicLink(ctx, &awsec2.EnableVpcClassicLinkInput{
		VpcId: aws.String(vpcID),
	})
	if err != nil || enableVpcClassicLinkOut.Return == nil || !aws.ToBool(enableVpcClassicLinkOut.Return) {
		t.Fatalf("enable vpc classic link: %v", err)
	}

	enableDnsSupportOut, err := client.EnableVpcClassicLinkDnsSupport(ctx, &awsec2.EnableVpcClassicLinkDnsSupportInput{
		VpcId: aws.String(vpcID),
	})
	if err != nil || enableDnsSupportOut.Return == nil || !aws.ToBool(enableDnsSupportOut.Return) {
		t.Fatalf("enable vpc classic link dns support: %v", err)
	}

	describeVpcClassicLinkOut, err := client.DescribeVpcClassicLink(ctx, &awsec2.DescribeVpcClassicLinkInput{
		VpcIds: []string{vpcID},
	})
	if err != nil || len(describeVpcClassicLinkOut.Vpcs) != 1 || !aws.ToBool(describeVpcClassicLinkOut.Vpcs[0].ClassicLinkEnabled) {
		t.Fatalf("describe vpc classic link: %v", err)
	}

	describeVpcClassicLinkDnsSupportOut, err := client.DescribeVpcClassicLinkDnsSupport(ctx, &awsec2.DescribeVpcClassicLinkDnsSupportInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil || len(describeVpcClassicLinkDnsSupportOut.Vpcs) != 1 || describeVpcClassicLinkDnsSupportOut.NextToken == nil {
		t.Fatalf("describe vpc classic link dns support: %v", err)
	}

	attachClassicLinkVpcOut, err := client.AttachClassicLinkVpc(ctx, &awsec2.AttachClassicLinkVpcInput{
		InstanceId: aws.String(instanceID),
		VpcId:      aws.String(vpcID),
		Groups:     []string{groupID},
	})
	if err != nil || attachClassicLinkVpcOut.Return == nil || !aws.ToBool(attachClassicLinkVpcOut.Return) {
		t.Fatalf("attach classic link vpc: %v", err)
	}

	describeClassicLinkInstancesOut, err := client.DescribeClassicLinkInstances(ctx, &awsec2.DescribeClassicLinkInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil || len(describeClassicLinkInstancesOut.Instances) != 1 {
		t.Fatalf("describe classic link instances: %v", err)
	}

	detachClassicLinkVpcOut, err := client.DetachClassicLinkVpc(ctx, &awsec2.DetachClassicLinkVpcInput{
		InstanceId: aws.String(instanceID),
		VpcId:      aws.String(vpcID),
	})
	if err != nil || detachClassicLinkVpcOut.Return == nil || !aws.ToBool(detachClassicLinkVpcOut.Return) {
		t.Fatalf("detach classic link vpc: %v", err)
	}

	disableDnsSupportOut, err := client.DisableVpcClassicLinkDnsSupport(ctx, &awsec2.DisableVpcClassicLinkDnsSupportInput{
		VpcId: aws.String(vpcID),
	})
	if err != nil || disableDnsSupportOut.Return == nil || !aws.ToBool(disableDnsSupportOut.Return) {
		t.Fatalf("disable vpc classic link dns support: %v", err)
	}

	disableVpcClassicLinkOut, err := client.DisableVpcClassicLink(ctx, &awsec2.DisableVpcClassicLinkInput{
		VpcId: aws.String(vpcID),
	})
	if err != nil || disableVpcClassicLinkOut.Return == nil || !aws.ToBool(disableVpcClassicLinkOut.Return) {
		t.Fatalf("disable vpc classic link: %v", err)
	}
}

func TestEC2Stage23ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AttachClassicLinkVpc",
		"DescribeClassicLinkInstances",
		"DescribeVpcClassicLink",
		"DescribeVpcClassicLinkDnsSupport",
		"DetachClassicLinkVpc",
		"DisableVpcClassicLink",
		"DisableVpcClassicLinkDnsSupport",
		"EnableVpcClassicLink",
		"EnableVpcClassicLinkDnsSupport",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "AttachClassicLinkVpc":
			params["InstanceId"] = "i-00000001"
			params["VpcId"] = "vpc-00000001"
			params["SecurityGroupId.1"] = "sg-00000000"
		case "DescribeClassicLinkInstances":
			params["MaxResults"] = "1"
		case "DescribeVpcClassicLink":
			params["VpcId.1"] = "vpc-00000001"
		case "DescribeVpcClassicLinkDnsSupport":
			params["MaxResults"] = "1"
			params["VpcIds.1"] = "vpc-00000001"
		case "DetachClassicLinkVpc":
			params["InstanceId"] = "i-00000001"
			params["VpcId"] = "vpc-00000001"
		case "DisableVpcClassicLink", "DisableVpcClassicLinkDnsSupport", "EnableVpcClassicLink", "EnableVpcClassicLinkDnsSupport":
			params["VpcId"] = "vpc-00000001"
		}

		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
