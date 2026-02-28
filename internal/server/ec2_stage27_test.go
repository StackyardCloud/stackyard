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

func TestEC2Stage27SDKLifecycle(t *testing.T) {
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

	createGroupOneOut, err := client.CreateSecurityGroup(ctx, &awsec2.CreateSecurityGroupInput{
		GroupName:   aws.String("stage27-shared-sg-1"),
		Description: aws.String("stage27 security-group-for-vpc APIs #1"),
		VpcId:       aws.String("vpc-00000001"),
	})
	if err != nil || createGroupOneOut.GroupId == nil {
		t.Fatalf("create security group #1: %v", err)
	}
	groupOneID := aws.ToString(createGroupOneOut.GroupId)

	createGroupTwoOut, err := client.CreateSecurityGroup(ctx, &awsec2.CreateSecurityGroupInput{
		GroupName:   aws.String("stage27-shared-sg-2"),
		Description: aws.String("stage27 security-group-for-vpc APIs #2"),
		VpcId:       aws.String("vpc-00000001"),
	})
	if err != nil || createGroupTwoOut.GroupId == nil {
		t.Fatalf("create security group #2: %v", err)
	}
	groupTwoID := aws.ToString(createGroupTwoOut.GroupId)

	if _, err := client.AuthorizeSecurityGroupIngress(ctx, &awsec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(groupOneID),
		IpPermissions: []awsec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(8081),
				ToPort:     aws.Int32(8081),
				IpRanges:   []awsec2types.IpRange{{CidrIp: aws.String("203.0.113.0/24")}},
			},
		},
	}); err != nil {
		t.Fatalf("authorize security group ingress: %v", err)
	}

	rulesOut, err := client.DescribeSecurityGroupRules(ctx, &awsec2.DescribeSecurityGroupRulesInput{
		Filters: []awsec2types.Filter{{Name: aws.String("group-id"), Values: []string{groupOneID}}},
	})
	if err != nil {
		t.Fatalf("describe security group rules: %v", err)
	}
	var ingressRuleID string
	for _, rule := range rulesOut.SecurityGroupRules {
		if aws.ToString(rule.GroupId) != groupOneID || aws.ToBool(rule.IsEgress) {
			continue
		}
		if aws.ToString(rule.CidrIpv4) == "203.0.113.0/24" &&
			aws.ToString(rule.IpProtocol) == "tcp" &&
			aws.ToInt32(rule.FromPort) == 8081 &&
			aws.ToInt32(rule.ToPort) == 8081 {
			ingressRuleID = aws.ToString(rule.SecurityGroupRuleId)
			break
		}
	}
	if ingressRuleID == "" {
		t.Fatalf("expected ingress rule id for stage27 rule")
	}

	modifyOut, err := client.ModifySecurityGroupRules(ctx, &awsec2.ModifySecurityGroupRulesInput{
		GroupId: aws.String(groupOneID),
		SecurityGroupRules: []awsec2types.SecurityGroupRuleUpdate{
			{
				SecurityGroupRuleId: aws.String(ingressRuleID),
				SecurityGroupRule: &awsec2types.SecurityGroupRuleRequest{
					Description: aws.String("stage27 ingress updated"),
				},
			},
		},
	})
	if err != nil || modifyOut.Return == nil || !aws.ToBool(modifyOut.Return) {
		t.Fatalf("modify security group rules: %v", err)
	}

	updatedOut, err := client.DescribeSecurityGroupRules(ctx, &awsec2.DescribeSecurityGroupRulesInput{
		SecurityGroupRuleIds: []string{ingressRuleID},
	})
	if err != nil || len(updatedOut.SecurityGroupRules) != 1 {
		t.Fatalf("describe updated security group rule: %v", err)
	}
	if got := aws.ToString(updatedOut.SecurityGroupRules[0].Description); got != "stage27 ingress updated" {
		t.Fatalf("expected updated ingress description, got %q", got)
	}

	createVpcOut, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{CidrBlock: aws.String("10.227.0.0/16")})
	if err != nil || createVpcOut.Vpc == nil || createVpcOut.Vpc.VpcId == nil {
		t.Fatalf("create vpc: %v", err)
	}
	vpcID := aws.ToString(createVpcOut.Vpc.VpcId)

	if _, err := client.AssociateSecurityGroupVpc(ctx, &awsec2.AssociateSecurityGroupVpcInput{GroupId: aws.String(groupOneID), VpcId: aws.String(vpcID)}); err != nil {
		t.Fatalf("associate security group #1 with vpc: %v", err)
	}
	if _, err := client.AssociateSecurityGroupVpc(ctx, &awsec2.AssociateSecurityGroupVpcInput{GroupId: aws.String(groupTwoID), VpcId: aws.String(vpcID)}); err != nil {
		t.Fatalf("associate security group #2 with vpc: %v", err)
	}

	pageOneOut, err := client.GetSecurityGroupsForVpc(ctx, &awsec2.GetSecurityGroupsForVpcInput{
		VpcId:      aws.String(vpcID),
		MaxResults: aws.Int32(1),
	})
	if err != nil || len(pageOneOut.SecurityGroupForVpcs) != 1 || pageOneOut.NextToken == nil {
		t.Fatalf("get security groups for vpc page 1: %v", err)
	}

	pageTwoOut, err := client.GetSecurityGroupsForVpc(ctx, &awsec2.GetSecurityGroupsForVpcInput{
		VpcId:     aws.String(vpcID),
		NextToken: pageOneOut.NextToken,
	})
	if err != nil || len(pageTwoOut.SecurityGroupForVpcs) == 0 {
		t.Fatalf("get security groups for vpc page 2: %v", err)
	}

	seenGroupIDs := map[string]struct{}{}
	for _, item := range pageOneOut.SecurityGroupForVpcs {
		seenGroupIDs[aws.ToString(item.GroupId)] = struct{}{}
	}
	for _, item := range pageTwoOut.SecurityGroupForVpcs {
		seenGroupIDs[aws.ToString(item.GroupId)] = struct{}{}
	}
	if _, ok := seenGroupIDs[groupOneID]; !ok {
		t.Fatalf("expected group %s in paginated get security groups for vpc output", groupOneID)
	}
	if _, ok := seenGroupIDs[groupTwoID]; !ok {
		t.Fatalf("expected group %s in paginated get security groups for vpc output", groupTwoID)
	}

	filteredOut, err := client.GetSecurityGroupsForVpc(ctx, &awsec2.GetSecurityGroupsForVpcInput{
		VpcId: aws.String(vpcID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("group-id"), Values: []string{groupOneID}},
			{Name: aws.String("owner-id"), Values: []string{"123456789012"}},
			{Name: aws.String("primary-vpc-id"), Values: []string{"vpc-00000001"}},
		},
	})
	if err != nil || len(filteredOut.SecurityGroupForVpcs) != 1 {
		t.Fatalf("filtered get security groups for vpc: %v", err)
	}
	if got := aws.ToString(filteredOut.SecurityGroupForVpcs[0].PrimaryVpcId); got != "vpc-00000001" {
		t.Fatalf("expected primary vpc id vpc-00000001, got %q", got)
	}

	staleOut, err := client.DescribeStaleSecurityGroups(ctx, &awsec2.DescribeStaleSecurityGroupsInput{
		VpcId:      aws.String(vpcID),
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe stale security groups: %v", err)
	}
	if len(staleOut.StaleSecurityGroupSet) != 0 {
		t.Fatalf("expected no stale security groups, got %d", len(staleOut.StaleSecurityGroupSet))
	}
}

func TestEC2Stage27ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeStaleSecurityGroups",
		"GetSecurityGroupsForVpc",
		"ModifySecurityGroupRules",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "DescribeStaleSecurityGroups", "GetSecurityGroupsForVpc":
			params["VpcId"] = "vpc-00000001"
		case "ModifySecurityGroupRules":
			params["GroupId"] = "sg-00000001"
			params["SecurityGroupRule.1.SecurityGroupRuleId"] = "sgr-0000000000000000"
			params["SecurityGroupRule.1.SecurityGroupRule.Description"] = "updated"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
