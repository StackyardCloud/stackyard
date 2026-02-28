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

func TestEC2Stage62SDKLifecycle(t *testing.T) {
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

	createOut, err := client.CreateVpcEndpointServiceConfiguration(ctx, &awsec2.CreateVpcEndpointServiceConfigurationInput{
		NetworkLoadBalancerArns: []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/stage62/1234567890abcdef"},
	})
	if err != nil {
		t.Fatalf("create vpc endpoint service configuration: %v", err)
	}
	if createOut.ServiceConfiguration == nil || createOut.ServiceConfiguration.ServiceId == nil {
		t.Fatalf("expected service configuration id")
	}
	serviceID := aws.ToString(createOut.ServiceConfiguration.ServiceId)

	_, err = client.ModifyVpcEndpointServicePermissions(ctx, &awsec2.ModifyVpcEndpointServicePermissionsInput{
		ServiceId:            aws.String(serviceID),
		AddAllowedPrincipals: []string{"123456789012", "arn:aws:iam::123456789012:role/Stage62Role"},
	})
	if err != nil {
		t.Fatalf("modify vpc endpoint service permissions: %v", err)
	}

	pageOne, err := client.DescribeVpcEndpointServicePermissions(ctx, &awsec2.DescribeVpcEndpointServicePermissionsInput{
		ServiceId:  aws.String(serviceID),
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe service permissions page one: %v", err)
	}
	if len(pageOne.AllowedPrincipals) != 1 {
		t.Fatalf("expected one allowed principal in first page, got %d", len(pageOne.AllowedPrincipals))
	}
	if pageOne.NextToken == nil {
		t.Fatalf("expected next token in first page")
	}

	pageTwo, err := client.DescribeVpcEndpointServicePermissions(ctx, &awsec2.DescribeVpcEndpointServicePermissionsInput{
		ServiceId: aws.String(serviceID),
		NextToken: pageOne.NextToken,
	})
	if err != nil {
		t.Fatalf("describe service permissions page two: %v", err)
	}
	if len(pageTwo.AllowedPrincipals) == 0 {
		t.Fatalf("expected at least one allowed principal in second page")
	}

	filteredByType, err := client.DescribeVpcEndpointServicePermissions(ctx, &awsec2.DescribeVpcEndpointServicePermissionsInput{
		ServiceId: aws.String(serviceID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("principal-type"), Values: []string{"Role"}},
		},
	})
	if err != nil {
		t.Fatalf("describe service permissions by principal-type: %v", err)
	}
	if len(filteredByType.AllowedPrincipals) != 1 {
		t.Fatalf("expected one role principal, got %d", len(filteredByType.AllowedPrincipals))
	}
	if filteredByType.AllowedPrincipals[0].PrincipalType != awsec2types.PrincipalTypeRole {
		t.Fatalf("expected principal type Role, got %q", filteredByType.AllowedPrincipals[0].PrincipalType)
	}
	if aws.ToString(filteredByType.AllowedPrincipals[0].ServiceId) != serviceID {
		t.Fatalf("unexpected service id for filtered principal: %q", aws.ToString(filteredByType.AllowedPrincipals[0].ServiceId))
	}
}

func TestEC2Stage62ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeVpcEndpointServicePermissions",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"ServiceId": "vpce-svc-00000000",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
