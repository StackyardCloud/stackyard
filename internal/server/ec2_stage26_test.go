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

func TestEC2Stage26SDKLifecycle(t *testing.T) {
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

	createGroupOut, err := client.CreateSecurityGroup(ctx, &awsec2.CreateSecurityGroupInput{
		GroupName:   aws.String("stage26-shared-sg"),
		Description: aws.String("stage26 security-group-vpc-association APIs"),
		VpcId:       aws.String("vpc-00000001"),
	})
	if err != nil || createGroupOut.GroupId == nil {
		t.Fatalf("create security group: %v", err)
	}
	groupID := aws.ToString(createGroupOut.GroupId)

	createVpcOneOut, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{
		CidrBlock: aws.String("10.126.0.0/16"),
	})
	if err != nil || createVpcOneOut.Vpc == nil || createVpcOneOut.Vpc.VpcId == nil {
		t.Fatalf("create vpc #1: %v", err)
	}
	vpcOneID := aws.ToString(createVpcOneOut.Vpc.VpcId)

	createVpcTwoOut, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{
		CidrBlock: aws.String("10.127.0.0/16"),
	})
	if err != nil || createVpcTwoOut.Vpc == nil || createVpcTwoOut.Vpc.VpcId == nil {
		t.Fatalf("create vpc #2: %v", err)
	}
	vpcTwoID := aws.ToString(createVpcTwoOut.Vpc.VpcId)

	associateOneOut, err := client.AssociateSecurityGroupVpc(ctx, &awsec2.AssociateSecurityGroupVpcInput{
		GroupId: aws.String(groupID),
		VpcId:   aws.String(vpcOneID),
	})
	if err != nil || associateOneOut.State != awsec2types.SecurityGroupVpcAssociationStateAssociated {
		t.Fatalf("associate security group with vpc #1: %v", err)
	}

	associateTwoOut, err := client.AssociateSecurityGroupVpc(ctx, &awsec2.AssociateSecurityGroupVpcInput{
		GroupId: aws.String(groupID),
		VpcId:   aws.String(vpcTwoID),
	})
	if err != nil || associateTwoOut.State != awsec2types.SecurityGroupVpcAssociationStateAssociated {
		t.Fatalf("associate security group with vpc #2: %v", err)
	}

	describePageOneOut, err := client.DescribeSecurityGroupVpcAssociations(ctx, &awsec2.DescribeSecurityGroupVpcAssociationsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("group-id"), Values: []string{groupID}},
			{Name: aws.String("state"), Values: []string{"associated"}},
		},
		MaxResults: aws.Int32(1),
	})
	if err != nil || len(describePageOneOut.SecurityGroupVpcAssociations) != 1 || describePageOneOut.NextToken == nil {
		t.Fatalf("describe security group vpc associations page 1: %v", err)
	}

	describePageTwoOut, err := client.DescribeSecurityGroupVpcAssociations(ctx, &awsec2.DescribeSecurityGroupVpcAssociationsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("group-id"), Values: []string{groupID}},
		},
		NextToken: describePageOneOut.NextToken,
	})
	if err != nil || len(describePageTwoOut.SecurityGroupVpcAssociations) == 0 {
		t.Fatalf("describe security group vpc associations page 2: %v", err)
	}

	disassociateOut, err := client.DisassociateSecurityGroupVpc(ctx, &awsec2.DisassociateSecurityGroupVpcInput{
		GroupId: aws.String(groupID),
		VpcId:   aws.String(vpcOneID),
	})
	if err != nil || disassociateOut.State != awsec2types.SecurityGroupVpcAssociationStateDisassociated {
		t.Fatalf("disassociate security group from vpc #1: %v", err)
	}

	describeAfterDisassociateOut, err := client.DescribeSecurityGroupVpcAssociations(ctx, &awsec2.DescribeSecurityGroupVpcAssociationsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("group-id"), Values: []string{groupID}},
			{Name: aws.String("vpc-id"), Values: []string{vpcOneID}},
		},
	})
	if err != nil {
		t.Fatalf("describe after disassociate: %v", err)
	}
	if len(describeAfterDisassociateOut.SecurityGroupVpcAssociations) != 0 {
		t.Fatalf("expected no associations for disassociated vpc, got %d", len(describeAfterDisassociateOut.SecurityGroupVpcAssociations))
	}
}

func TestEC2Stage26ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AssociateSecurityGroupVpc",
		"DescribeSecurityGroupVpcAssociations",
		"DisassociateSecurityGroupVpc",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "AssociateSecurityGroupVpc", "DisassociateSecurityGroupVpc":
			params["GroupId"] = "sg-00000001"
			params["VpcId"] = "vpc-00000002"
		case "DescribeSecurityGroupVpcAssociations":
			params["Filter.1.Name"] = "group-id"
			params["Filter.1.Value.1"] = "sg-00000001"
			params["MaxResults"] = "1"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
