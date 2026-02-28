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

func TestEC2Stage68SDKLifecycle(t *testing.T) {
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

	createInstanceOut, err := client.CreateVerifiedAccessInstance(ctx, &awsec2.CreateVerifiedAccessInstanceInput{
		CidrEndpointsCustomSubDomain: aws.String("stage68"),
		Description:                  aws.String("stage68 instance"),
		FIPSEnabled:                  aws.Bool(true),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeVerifiedAccessInstance,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("stage68")}},
			},
		},
	})
	if err != nil || createInstanceOut.VerifiedAccessInstance == nil || createInstanceOut.VerifiedAccessInstance.VerifiedAccessInstanceId == nil {
		t.Fatalf("create verified access instance: %v", err)
	}
	instanceID := aws.ToString(createInstanceOut.VerifiedAccessInstance.VerifiedAccessInstanceId)

	describeInstancesOut, err := client.DescribeVerifiedAccessInstances(ctx, &awsec2.DescribeVerifiedAccessInstancesInput{
		VerifiedAccessInstanceIds: []string{instanceID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("fips-enabled"), Values: []string{"true"}},
			{Name: aws.String("tag:env"), Values: []string{"stage68"}},
		},
	})
	if err != nil {
		t.Fatalf("describe verified access instances: %v", err)
	}
	if len(describeInstancesOut.VerifiedAccessInstances) != 1 {
		t.Fatalf("expected one verified access instance, got %d", len(describeInstancesOut.VerifiedAccessInstances))
	}
	if !aws.ToBool(describeInstancesOut.VerifiedAccessInstances[0].FipsEnabled) {
		t.Fatalf("expected fips enabled true")
	}

	createGroupOut, err := client.CreateVerifiedAccessGroup(ctx, &awsec2.CreateVerifiedAccessGroupInput{
		VerifiedAccessInstanceId: aws.String(instanceID),
		Description:              aws.String("stage68 group"),
		PolicyDocument:           aws.String(`{"Version":"2012-10-17","Statement":[]}`),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeVerifiedAccessGroup,
				Tags:         []awsec2types.Tag{{Key: aws.String("team"), Value: aws.String("platform")}},
			},
		},
	})
	if err != nil || createGroupOut.VerifiedAccessGroup == nil || createGroupOut.VerifiedAccessGroup.VerifiedAccessGroupId == nil {
		t.Fatalf("create verified access group: %v", err)
	}
	groupID := aws.ToString(createGroupOut.VerifiedAccessGroup.VerifiedAccessGroupId)

	describeGroupsOut, err := client.DescribeVerifiedAccessGroups(ctx, &awsec2.DescribeVerifiedAccessGroupsInput{
		VerifiedAccessInstanceId: aws.String(instanceID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("verified-access-group-id"), Values: []string{groupID}},
			{Name: aws.String("owner"), Values: []string{"123456789012"}},
		},
	})
	if err != nil {
		t.Fatalf("describe verified access groups: %v", err)
	}
	if len(describeGroupsOut.VerifiedAccessGroups) != 1 {
		t.Fatalf("expected one verified access group, got %d", len(describeGroupsOut.VerifiedAccessGroups))
	}
	if aws.ToString(describeGroupsOut.VerifiedAccessGroups[0].VerifiedAccessGroupId) != groupID {
		t.Fatalf("unexpected group id in describe: %q", aws.ToString(describeGroupsOut.VerifiedAccessGroups[0].VerifiedAccessGroupId))
	}

	createEndpointOut, err := client.CreateVerifiedAccessEndpoint(ctx, &awsec2.CreateVerifiedAccessEndpointInput{
		VerifiedAccessGroupId: aws.String(groupID),
		AttachmentType:        awsec2types.VerifiedAccessEndpointAttachmentTypeVpc,
		EndpointType:          awsec2types.VerifiedAccessEndpointTypeLoadBalancer,
		ApplicationDomain:     aws.String("app.stage68.example.com"),
		Description:           aws.String("stage68 endpoint"),
		DomainCertificateArn:  aws.String("arn:aws:acm:us-east-1:123456789012:certificate/stage68"),
		EndpointDomainPrefix:  aws.String("stage68"),
		SecurityGroupIds:      []string{"sg-00000000"},
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeVerifiedAccessEndpoint,
				Tags:         []awsec2types.Tag{{Key: aws.String("svc"), Value: aws.String("frontend")}},
			},
		},
	})
	if err != nil || createEndpointOut.VerifiedAccessEndpoint == nil || createEndpointOut.VerifiedAccessEndpoint.VerifiedAccessEndpointId == nil {
		t.Fatalf("create verified access endpoint: %v", err)
	}
	endpointID := aws.ToString(createEndpointOut.VerifiedAccessEndpoint.VerifiedAccessEndpointId)

	describeEndpointsOut, err := client.DescribeVerifiedAccessEndpoints(ctx, &awsec2.DescribeVerifiedAccessEndpointsInput{
		VerifiedAccessGroupId: aws.String(groupID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("endpoint-type"), Values: []string{"load-balancer"}},
			{Name: aws.String("status.code"), Values: []string{"active"}},
		},
	})
	if err != nil {
		t.Fatalf("describe verified access endpoints: %v", err)
	}
	if len(describeEndpointsOut.VerifiedAccessEndpoints) != 1 {
		t.Fatalf("expected one verified access endpoint, got %d", len(describeEndpointsOut.VerifiedAccessEndpoints))
	}
	if aws.ToString(describeEndpointsOut.VerifiedAccessEndpoints[0].VerifiedAccessEndpointId) != endpointID {
		t.Fatalf("unexpected endpoint id in describe: %q", aws.ToString(describeEndpointsOut.VerifiedAccessEndpoints[0].VerifiedAccessEndpointId))
	}

	createTrustProviderOneOut, err := client.CreateVerifiedAccessTrustProvider(ctx, &awsec2.CreateVerifiedAccessTrustProviderInput{
		PolicyReferenceName:   aws.String("stage68-user"),
		TrustProviderType:     awsec2types.TrustProviderTypeUser,
		UserTrustProviderType: awsec2types.UserTrustProviderTypeIamIdentityCenter,
		Description:           aws.String("stage68 trust provider one"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeVerifiedAccessTrustProvider,
				Tags:         []awsec2types.Tag{{Key: aws.String("owner"), Value: aws.String("team-a")}},
			},
		},
	})
	if err != nil || createTrustProviderOneOut.VerifiedAccessTrustProvider == nil || createTrustProviderOneOut.VerifiedAccessTrustProvider.VerifiedAccessTrustProviderId == nil {
		t.Fatalf("create verified access trust provider one: %v", err)
	}
	trustProviderIDOne := aws.ToString(createTrustProviderOneOut.VerifiedAccessTrustProvider.VerifiedAccessTrustProviderId)

	createTrustProviderTwoOut, err := client.CreateVerifiedAccessTrustProvider(ctx, &awsec2.CreateVerifiedAccessTrustProviderInput{
		PolicyReferenceName:   aws.String("stage68-user-2"),
		TrustProviderType:     awsec2types.TrustProviderTypeUser,
		UserTrustProviderType: awsec2types.UserTrustProviderTypeIamIdentityCenter,
		Description:           aws.String("stage68 trust provider two"),
	})
	if err != nil || createTrustProviderTwoOut.VerifiedAccessTrustProvider == nil || createTrustProviderTwoOut.VerifiedAccessTrustProvider.VerifiedAccessTrustProviderId == nil {
		t.Fatalf("create verified access trust provider two: %v", err)
	}
	trustProviderIDTwo := aws.ToString(createTrustProviderTwoOut.VerifiedAccessTrustProvider.VerifiedAccessTrustProviderId)

	describeTrustProvidersPageOneOut, err := client.DescribeVerifiedAccessTrustProviders(ctx, &awsec2.DescribeVerifiedAccessTrustProvidersInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe verified access trust providers page one: %v", err)
	}
	if len(describeTrustProvidersPageOneOut.VerifiedAccessTrustProviders) != 1 {
		t.Fatalf("expected one trust provider on page one, got %d", len(describeTrustProvidersPageOneOut.VerifiedAccessTrustProviders))
	}
	if describeTrustProvidersPageOneOut.NextToken == nil {
		t.Fatalf("expected next token on trust provider page one")
	}

	describeTrustProvidersPageTwoOut, err := client.DescribeVerifiedAccessTrustProviders(ctx, &awsec2.DescribeVerifiedAccessTrustProvidersInput{
		NextToken: describeTrustProvidersPageOneOut.NextToken,
	})
	if err != nil {
		t.Fatalf("describe verified access trust providers page two: %v", err)
	}
	if len(describeTrustProvidersPageTwoOut.VerifiedAccessTrustProviders) == 0 {
		t.Fatalf("expected trust providers on page two")
	}

	describeTrustProvidersFilteredOut, err := client.DescribeVerifiedAccessTrustProviders(ctx, &awsec2.DescribeVerifiedAccessTrustProvidersInput{
		VerifiedAccessTrustProviderIds: []string{trustProviderIDOne, trustProviderIDTwo},
		Filters: []awsec2types.Filter{
			{Name: aws.String("policy-reference-name"), Values: []string{"stage68-user"}},
			{Name: aws.String("trust-provider-type"), Values: []string{"user"}},
		},
	})
	if err != nil {
		t.Fatalf("describe verified access trust providers filtered: %v", err)
	}
	if len(describeTrustProvidersFilteredOut.VerifiedAccessTrustProviders) != 1 {
		t.Fatalf("expected one filtered trust provider, got %d", len(describeTrustProvidersFilteredOut.VerifiedAccessTrustProviders))
	}
	if aws.ToString(describeTrustProvidersFilteredOut.VerifiedAccessTrustProviders[0].VerifiedAccessTrustProviderId) != trustProviderIDOne {
		t.Fatalf("unexpected filtered trust provider id: %q", aws.ToString(describeTrustProvidersFilteredOut.VerifiedAccessTrustProviders[0].VerifiedAccessTrustProviderId))
	}

	deleteEndpointOut, err := client.DeleteVerifiedAccessEndpoint(ctx, &awsec2.DeleteVerifiedAccessEndpointInput{
		VerifiedAccessEndpointId: aws.String(endpointID),
	})
	if err != nil || deleteEndpointOut.VerifiedAccessEndpoint == nil {
		t.Fatalf("delete verified access endpoint: %v", err)
	}
	if deleteEndpointOut.VerifiedAccessEndpoint.DeletionTime == nil {
		t.Fatalf("expected deletion time when deleting verified access endpoint")
	}

	deleteGroupOut, err := client.DeleteVerifiedAccessGroup(ctx, &awsec2.DeleteVerifiedAccessGroupInput{
		VerifiedAccessGroupId: aws.String(groupID),
	})
	if err != nil || deleteGroupOut.VerifiedAccessGroup == nil {
		t.Fatalf("delete verified access group: %v", err)
	}
	if deleteGroupOut.VerifiedAccessGroup.DeletionTime == nil {
		t.Fatalf("expected deletion time when deleting verified access group")
	}

	if _, err := client.DeleteVerifiedAccessInstance(ctx, &awsec2.DeleteVerifiedAccessInstanceInput{
		VerifiedAccessInstanceId: aws.String(instanceID),
	}); err != nil {
		t.Fatalf("delete verified access instance: %v", err)
	}

	if _, err := client.DeleteVerifiedAccessTrustProvider(ctx, &awsec2.DeleteVerifiedAccessTrustProviderInput{
		VerifiedAccessTrustProviderId: aws.String(trustProviderIDOne),
	}); err != nil {
		t.Fatalf("delete verified access trust provider one: %v", err)
	}
	if _, err := client.DeleteVerifiedAccessTrustProvider(ctx, &awsec2.DeleteVerifiedAccessTrustProviderInput{
		VerifiedAccessTrustProviderId: aws.String(trustProviderIDTwo),
	}); err != nil {
		t.Fatalf("delete verified access trust provider two: %v", err)
	}
}

func TestEC2Stage68ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateVerifiedAccessInstance",
		"DeleteVerifiedAccessInstance",
		"DescribeVerifiedAccessInstances",
		"CreateVerifiedAccessGroup",
		"DeleteVerifiedAccessGroup",
		"DescribeVerifiedAccessGroups",
		"CreateVerifiedAccessEndpoint",
		"DeleteVerifiedAccessEndpoint",
		"DescribeVerifiedAccessEndpoints",
		"CreateVerifiedAccessTrustProvider",
		"DeleteVerifiedAccessTrustProvider",
		"DescribeVerifiedAccessTrustProviders",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "DeleteVerifiedAccessInstance":
			params["VerifiedAccessInstanceId"] = "vai-00000000"
		case "CreateVerifiedAccessGroup":
			params["VerifiedAccessInstanceId"] = "vai-00000000"
		case "DeleteVerifiedAccessGroup":
			params["VerifiedAccessGroupId"] = "vag-00000000"
		case "CreateVerifiedAccessEndpoint":
			params["VerifiedAccessGroupId"] = "vag-00000000"
			params["AttachmentType"] = "vpc"
			params["EndpointType"] = "load-balancer"
		case "DeleteVerifiedAccessEndpoint":
			params["VerifiedAccessEndpointId"] = "vae-00000000"
		case "CreateVerifiedAccessTrustProvider":
			params["PolicyReferenceName"] = "stage68"
			params["TrustProviderType"] = "user"
			params["UserTrustProviderType"] = "iam-identity-center"
		case "DeleteVerifiedAccessTrustProvider":
			params["VerifiedAccessTrustProviderId"] = "vatp-00000000"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
