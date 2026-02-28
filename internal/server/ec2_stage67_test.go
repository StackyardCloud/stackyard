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

func TestEC2Stage67SDKLifecycle(t *testing.T) {
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

	describeOptionsOut, err := client.DescribeVpcBlockPublicAccessOptions(ctx, &awsec2.DescribeVpcBlockPublicAccessOptionsInput{})
	if err != nil {
		t.Fatalf("describe vpc block public access options: %v", err)
	}
	if describeOptionsOut.VpcBlockPublicAccessOptions == nil {
		t.Fatalf("expected vpc block public access options")
	}
	if describeOptionsOut.VpcBlockPublicAccessOptions.InternetGatewayBlockMode != awsec2types.InternetGatewayBlockModeOff {
		t.Fatalf("unexpected internet gateway block mode: %q", describeOptionsOut.VpcBlockPublicAccessOptions.InternetGatewayBlockMode)
	}

	createSubnetOneOut, err := client.CreateSubnet(ctx, &awsec2.CreateSubnetInput{
		VpcId:     aws.String("vpc-00000001"),
		CidrBlock: aws.String("10.67.1.0/24"),
	})
	if err != nil || createSubnetOneOut.Subnet == nil || createSubnetOneOut.Subnet.SubnetId == nil {
		t.Fatalf("create subnet one: %v", err)
	}
	subnetIDOne := aws.ToString(createSubnetOneOut.Subnet.SubnetId)

	createSubnetTwoOut, err := client.CreateSubnet(ctx, &awsec2.CreateSubnetInput{
		VpcId:     aws.String("vpc-00000001"),
		CidrBlock: aws.String("10.67.2.0/24"),
	})
	if err != nil || createSubnetTwoOut.Subnet == nil || createSubnetTwoOut.Subnet.SubnetId == nil {
		t.Fatalf("create subnet two: %v", err)
	}
	subnetIDTwo := aws.ToString(createSubnetTwoOut.Subnet.SubnetId)

	createOneOut, err := client.CreateVpcBlockPublicAccessExclusion(ctx, &awsec2.CreateVpcBlockPublicAccessExclusionInput{
		InternetGatewayExclusionMode: awsec2types.InternetGatewayExclusionModeAllowEgress,
		SubnetId:                     aws.String(subnetIDOne),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeVpcBlockPublicAccessExclusion,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("stage67")}},
			},
		},
	})
	if err != nil || createOneOut.VpcBlockPublicAccessExclusion == nil || createOneOut.VpcBlockPublicAccessExclusion.ExclusionId == nil {
		t.Fatalf("create vpc block public access exclusion one: %v", err)
	}
	exclusionIDOne := aws.ToString(createOneOut.VpcBlockPublicAccessExclusion.ExclusionId)
	if createOneOut.VpcBlockPublicAccessExclusion.InternetGatewayExclusionMode != awsec2types.InternetGatewayExclusionModeAllowEgress {
		t.Fatalf("unexpected exclusion mode for one: %q", createOneOut.VpcBlockPublicAccessExclusion.InternetGatewayExclusionMode)
	}
	resourceARNOne := aws.ToString(createOneOut.VpcBlockPublicAccessExclusion.ResourceArn)

	createTwoOut, err := client.CreateVpcBlockPublicAccessExclusion(ctx, &awsec2.CreateVpcBlockPublicAccessExclusionInput{
		InternetGatewayExclusionMode: awsec2types.InternetGatewayExclusionModeAllowBidirectional,
		SubnetId:                     aws.String(subnetIDTwo),
	})
	if err != nil || createTwoOut.VpcBlockPublicAccessExclusion == nil || createTwoOut.VpcBlockPublicAccessExclusion.ExclusionId == nil {
		t.Fatalf("create vpc block public access exclusion two: %v", err)
	}
	exclusionIDTwo := aws.ToString(createTwoOut.VpcBlockPublicAccessExclusion.ExclusionId)

	describePageOneOut, err := client.DescribeVpcBlockPublicAccessExclusions(ctx, &awsec2.DescribeVpcBlockPublicAccessExclusionsInput{
		ExclusionIds: []string{exclusionIDOne, exclusionIDTwo},
		MaxResults:   aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe vpc block public access exclusions page one: %v", err)
	}
	if len(describePageOneOut.VpcBlockPublicAccessExclusions) != 1 {
		t.Fatalf("expected one exclusion in page one, got %d", len(describePageOneOut.VpcBlockPublicAccessExclusions))
	}
	if describePageOneOut.NextToken == nil {
		t.Fatalf("expected next token in exclusions page one")
	}

	describePageTwoOut, err := client.DescribeVpcBlockPublicAccessExclusions(ctx, &awsec2.DescribeVpcBlockPublicAccessExclusionsInput{
		ExclusionIds: []string{exclusionIDOne, exclusionIDTwo},
		NextToken:    describePageOneOut.NextToken,
	})
	if err != nil {
		t.Fatalf("describe vpc block public access exclusions page two: %v", err)
	}
	if len(describePageTwoOut.VpcBlockPublicAccessExclusions) == 0 {
		t.Fatalf("expected exclusions in page two")
	}

	describeFilteredOut, err := client.DescribeVpcBlockPublicAccessExclusions(ctx, &awsec2.DescribeVpcBlockPublicAccessExclusionsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("resource-arn"), Values: []string{resourceARNOne}},
			{Name: aws.String("internet-gateway-exclusion-mode"), Values: []string{"allow-egress"}},
		},
	})
	if err != nil {
		t.Fatalf("describe vpc block public access exclusions filtered: %v", err)
	}
	if len(describeFilteredOut.VpcBlockPublicAccessExclusions) != 1 {
		t.Fatalf("expected one filtered exclusion, got %d", len(describeFilteredOut.VpcBlockPublicAccessExclusions))
	}
	if aws.ToString(describeFilteredOut.VpcBlockPublicAccessExclusions[0].ExclusionId) != exclusionIDOne {
		t.Fatalf("unexpected filtered exclusion id: %q", aws.ToString(describeFilteredOut.VpcBlockPublicAccessExclusions[0].ExclusionId))
	}

	deleteOut, err := client.DeleteVpcBlockPublicAccessExclusion(ctx, &awsec2.DeleteVpcBlockPublicAccessExclusionInput{
		ExclusionId: aws.String(exclusionIDTwo),
	})
	if err != nil || deleteOut.VpcBlockPublicAccessExclusion == nil {
		t.Fatalf("delete vpc block public access exclusion: %v", err)
	}
	if deleteOut.VpcBlockPublicAccessExclusion.State != awsec2types.VpcBlockPublicAccessExclusionStateDeleteComplete {
		t.Fatalf("unexpected exclusion state after delete: %q", deleteOut.VpcBlockPublicAccessExclusion.State)
	}
	if deleteOut.VpcBlockPublicAccessExclusion.DeletionTimestamp == nil {
		t.Fatalf("expected deletion timestamp after delete")
	}
}

func TestEC2Stage67ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateVpcBlockPublicAccessExclusion",
		"DeleteVpcBlockPublicAccessExclusion",
		"DescribeVpcBlockPublicAccessExclusions",
		"DescribeVpcBlockPublicAccessOptions",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateVpcBlockPublicAccessExclusion":
			params["InternetGatewayExclusionMode"] = "allow-egress"
			params["SubnetId"] = "subnet-00000001"
		case "DeleteVpcBlockPublicAccessExclusion":
			params["ExclusionId"] = "vpcbpa-ex-00000000"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
