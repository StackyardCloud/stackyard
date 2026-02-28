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

func TestEC2Stage25SDKLifecycle(t *testing.T) {
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
		GroupName:   aws.String("stage25-sg"),
		Description: aws.String("stage25 security-group-rule APIs"),
		VpcId:       aws.String("vpc-00000001"),
	})
	if err != nil || createGroupOut.GroupId == nil {
		t.Fatalf("create security group: %v", err)
	}
	groupID := aws.ToString(createGroupOut.GroupId)

	if _, err := client.AuthorizeSecurityGroupIngress(ctx, &awsec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(groupID),
		IpPermissions: []awsec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(8080),
				ToPort:     aws.Int32(8080),
				IpRanges: []awsec2types.IpRange{
					{CidrIp: aws.String("203.0.113.0/24")},
				},
			},
		},
	}); err != nil {
		t.Fatalf("authorize ingress rule: %v", err)
	}

	if _, err := client.AuthorizeSecurityGroupEgress(ctx, &awsec2.AuthorizeSecurityGroupEgressInput{
		GroupId: aws.String(groupID),
		IpPermissions: []awsec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(8443),
				ToPort:     aws.Int32(8443),
				IpRanges: []awsec2types.IpRange{
					{CidrIp: aws.String("198.51.100.0/24")},
				},
			},
		},
	}); err != nil {
		t.Fatalf("authorize egress rule: %v", err)
	}

	firstPageOut, err := client.DescribeSecurityGroupRules(ctx, &awsec2.DescribeSecurityGroupRulesInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("group-id"), Values: []string{groupID}},
		},
		MaxResults: aws.Int32(1),
	})
	if err != nil || len(firstPageOut.SecurityGroupRules) != 1 || firstPageOut.NextToken == nil {
		t.Fatalf("describe security group rules page 1: %v", err)
	}

	secondPageOut, err := client.DescribeSecurityGroupRules(ctx, &awsec2.DescribeSecurityGroupRulesInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("group-id"), Values: []string{groupID}},
		},
		MaxResults: aws.Int32(10),
		NextToken:  firstPageOut.NextToken,
	})
	if err != nil || len(secondPageOut.SecurityGroupRules) == 0 {
		t.Fatalf("describe security group rules page 2: %v", err)
	}

	allRulesOut, err := client.DescribeSecurityGroupRules(ctx, &awsec2.DescribeSecurityGroupRulesInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("group-id"), Values: []string{groupID}},
		},
	})
	if err != nil || len(allRulesOut.SecurityGroupRules) < 2 {
		t.Fatalf("describe security group rules: %v", err)
	}

	var ingressRuleID string
	var egressRuleID string
	for _, rule := range allRulesOut.SecurityGroupRules {
		if aws.ToString(rule.GroupId) != groupID {
			continue
		}
		if !aws.ToBool(rule.IsEgress) &&
			aws.ToString(rule.CidrIpv4) == "203.0.113.0/24" &&
			aws.ToString(rule.IpProtocol) == "tcp" &&
			aws.ToInt32(rule.FromPort) == 8080 &&
			aws.ToInt32(rule.ToPort) == 8080 {
			ingressRuleID = aws.ToString(rule.SecurityGroupRuleId)
		}
		if aws.ToBool(rule.IsEgress) &&
			aws.ToString(rule.CidrIpv4) == "198.51.100.0/24" &&
			aws.ToString(rule.IpProtocol) == "tcp" &&
			aws.ToInt32(rule.FromPort) == 8443 &&
			aws.ToInt32(rule.ToPort) == 8443 {
			egressRuleID = aws.ToString(rule.SecurityGroupRuleId)
		}
	}
	if ingressRuleID == "" || egressRuleID == "" {
		t.Fatalf("expected ingress and egress rule IDs, got ingress=%q egress=%q", ingressRuleID, egressRuleID)
	}

	ingressUpdateOut, err := client.UpdateSecurityGroupRuleDescriptionsIngress(ctx, &awsec2.UpdateSecurityGroupRuleDescriptionsIngressInput{
		GroupId: aws.String(groupID),
		SecurityGroupRuleDescriptions: []awsec2types.SecurityGroupRuleDescription{
			{
				SecurityGroupRuleId: aws.String(ingressRuleID),
				Description:         aws.String("stage25 ingress"),
			},
		},
	})
	if err != nil || ingressUpdateOut.Return == nil || !aws.ToBool(ingressUpdateOut.Return) {
		t.Fatalf("update ingress rule description: %v", err)
	}

	egressUpdateOut, err := client.UpdateSecurityGroupRuleDescriptionsEgress(ctx, &awsec2.UpdateSecurityGroupRuleDescriptionsEgressInput{
		GroupId: aws.String(groupID),
		IpPermissions: []awsec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(8443),
				ToPort:     aws.Int32(8443),
				IpRanges: []awsec2types.IpRange{
					{CidrIp: aws.String("198.51.100.0/24"), Description: aws.String("stage25 egress")},
				},
			},
		},
	})
	if err != nil || egressUpdateOut.Return == nil || !aws.ToBool(egressUpdateOut.Return) {
		t.Fatalf("update egress rule description: %v", err)
	}

	updatedOut, err := client.DescribeSecurityGroupRules(ctx, &awsec2.DescribeSecurityGroupRulesInput{
		SecurityGroupRuleIds: []string{ingressRuleID, egressRuleID},
	})
	if err != nil || len(updatedOut.SecurityGroupRules) != 2 {
		t.Fatalf("describe updated security group rules: %v", err)
	}
	seenIngress := false
	seenEgress := false
	for _, rule := range updatedOut.SecurityGroupRules {
		ruleID := aws.ToString(rule.SecurityGroupRuleId)
		switch ruleID {
		case ingressRuleID:
			seenIngress = aws.ToString(rule.Description) == "stage25 ingress"
		case egressRuleID:
			seenEgress = aws.ToString(rule.Description) == "stage25 egress"
		}
	}
	if !seenIngress || !seenEgress {
		t.Fatalf("expected updated descriptions, got ingress=%v egress=%v", seenIngress, seenEgress)
	}

	referencesOut, err := client.DescribeSecurityGroupReferences(ctx, &awsec2.DescribeSecurityGroupReferencesInput{
		GroupId: []string{groupID},
	})
	if err != nil || len(referencesOut.SecurityGroupReferenceSet) != 1 || aws.ToString(referencesOut.SecurityGroupReferenceSet[0].GroupId) != groupID {
		t.Fatalf("describe security group references: %v", err)
	}
}

func TestEC2Stage25ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeSecurityGroupReferences",
		"DescribeSecurityGroupRules",
		"UpdateSecurityGroupRuleDescriptionsEgress",
		"UpdateSecurityGroupRuleDescriptionsIngress",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "DescribeSecurityGroupReferences":
			params["GroupId.1"] = "sg-00000000"
		case "DescribeSecurityGroupRules":
			params["Filter.1.Name"] = "group-id"
			params["Filter.1.Value.1"] = "sg-00000000"
		case "UpdateSecurityGroupRuleDescriptionsEgress", "UpdateSecurityGroupRuleDescriptionsIngress":
			params["GroupId"] = "sg-00000000"
			params["SecurityGroupRuleDescription.1.SecurityGroupRuleId"] = "sgr-0000000000000000"
			params["SecurityGroupRuleDescription.1.Description"] = "desc"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
