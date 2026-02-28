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

func TestEC2Stage118SDKLifecycle(t *testing.T) {
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

	createIpamOut, err := client.CreateIpam(ctx, &awsec2.CreateIpamInput{
		OperatingRegions: []awsec2types.AddIpamOperatingRegion{
			{RegionName: aws.String("us-east-1")},
		},
		Tier: awsec2types.IpamTierFree,
	})
	if err != nil {
		t.Fatalf("create ipam: %v", err)
	}
	if createIpamOut.Ipam == nil || createIpamOut.Ipam.IpamId == nil {
		t.Fatalf("expected created ipam")
	}
	ipamID := aws.ToString(createIpamOut.Ipam.IpamId)

	createIpamScopeOut, err := client.CreateIpamScope(ctx, &awsec2.CreateIpamScopeInput{
		IpamId: aws.String(ipamID),
	})
	if err != nil {
		t.Fatalf("create ipam scope: %v", err)
	}
	if createIpamScopeOut.IpamScope == nil || createIpamScopeOut.IpamScope.IpamScopeId == nil {
		t.Fatalf("expected created ipam scope")
	}
	ipamScopeID := aws.ToString(createIpamScopeOut.IpamScope.IpamScopeId)

	createIpamPoolOut, err := client.CreateIpamPool(ctx, &awsec2.CreateIpamPoolInput{
		AddressFamily: awsec2types.AddressFamilyIpv4,
		IpamScopeId:   aws.String(ipamScopeID),
	})
	if err != nil {
		t.Fatalf("create ipam pool: %v", err)
	}
	if createIpamPoolOut.IpamPool == nil || createIpamPoolOut.IpamPool.IpamPoolId == nil {
		t.Fatalf("expected created ipam pool")
	}
	ipamPoolID := aws.ToString(createIpamPoolOut.IpamPool.IpamPoolId)

	createDiscoveryOut, err := client.CreateIpamResourceDiscovery(ctx, &awsec2.CreateIpamResourceDiscoveryInput{
		OperatingRegions: []awsec2types.AddIpamOperatingRegion{
			{RegionName: aws.String("us-east-1")},
		},
	})
	if err != nil {
		t.Fatalf("create ipam resource discovery: %v", err)
	}
	if createDiscoveryOut.IpamResourceDiscovery == nil || createDiscoveryOut.IpamResourceDiscovery.IpamResourceDiscoveryId == nil {
		t.Fatalf("expected created ipam resource discovery")
	}
	discoveryID := aws.ToString(createDiscoveryOut.IpamResourceDiscovery.IpamResourceDiscoveryId)

	associateDiscoveryOut, err := client.AssociateIpamResourceDiscovery(ctx, &awsec2.AssociateIpamResourceDiscoveryInput{
		IpamId:                  aws.String(ipamID),
		IpamResourceDiscoveryId: aws.String(discoveryID),
	})
	if err != nil {
		t.Fatalf("associate ipam resource discovery: %v", err)
	}
	if associateDiscoveryOut.IpamResourceDiscoveryAssociation == nil || associateDiscoveryOut.IpamResourceDiscoveryAssociation.IpamResourceDiscoveryAssociationId == nil {
		t.Fatalf("expected associated ipam resource discovery")
	}
	associationID := aws.ToString(associateDiscoveryOut.IpamResourceDiscoveryAssociation.IpamResourceDiscoveryAssociationId)

	createTokenOut, err := client.CreateIpamExternalResourceVerificationToken(ctx, &awsec2.CreateIpamExternalResourceVerificationTokenInput{
		IpamId: aws.String(ipamID),
	})
	if err != nil {
		t.Fatalf("create ipam external resource verification token: %v", err)
	}
	if createTokenOut.IpamExternalResourceVerificationToken == nil || createTokenOut.IpamExternalResourceVerificationToken.IpamExternalResourceVerificationTokenId == nil {
		t.Fatalf("expected created ipam external resource verification token")
	}
	tokenID := aws.ToString(createTokenOut.IpamExternalResourceVerificationToken.IpamExternalResourceVerificationTokenId)

	if _, err := client.AssociateIpamByoasn(ctx, &awsec2.AssociateIpamByoasnInput{
		Asn:  aws.String("64512"),
		Cidr: aws.String("198.51.100.0/24"),
	}); err != nil {
		t.Fatalf("associate ipam byoasn: %v", err)
	}

	describeInstanceTypesOut, err := client.DescribeInstanceTypes(ctx, &awsec2.DescribeInstanceTypesInput{
		InstanceTypes: []awsec2types.InstanceType{awsec2types.InstanceTypeM5Large},
		MaxResults:    aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe instance types: %v", err)
	}
	if len(describeInstanceTypesOut.InstanceTypes) == 0 {
		t.Fatalf("expected at least one instance type")
	}

	describeIpamByoasnOut, err := client.DescribeIpamByoasn(ctx, &awsec2.DescribeIpamByoasnInput{MaxResults: aws.Int32(10)})
	if err != nil {
		t.Fatalf("describe ipam byoasn: %v", err)
	}
	if len(describeIpamByoasnOut.Byoasns) == 0 {
		t.Fatalf("expected at least one ipam byoasn entry")
	}

	describeTokensOut, err := client.DescribeIpamExternalResourceVerificationTokens(ctx, &awsec2.DescribeIpamExternalResourceVerificationTokensInput{
		IpamExternalResourceVerificationTokenIds: []string{tokenID},
		MaxResults:                               aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe ipam external resource verification tokens: %v", err)
	}
	if len(describeTokensOut.IpamExternalResourceVerificationTokens) != 1 {
		t.Fatalf("expected 1 verification token, got %d", len(describeTokensOut.IpamExternalResourceVerificationTokens))
	}

	describeIpamPoolsOut, err := client.DescribeIpamPools(ctx, &awsec2.DescribeIpamPoolsInput{
		IpamPoolIds: []string{ipamPoolID},
		MaxResults:  aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe ipam pools: %v", err)
	}
	if len(describeIpamPoolsOut.IpamPools) != 1 {
		t.Fatalf("expected 1 ipam pool, got %d", len(describeIpamPoolsOut.IpamPools))
	}

	describeDiscoveriesOut, err := client.DescribeIpamResourceDiscoveries(ctx, &awsec2.DescribeIpamResourceDiscoveriesInput{
		IpamResourceDiscoveryIds: []string{discoveryID},
		MaxResults:               aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe ipam resource discoveries: %v", err)
	}
	if len(describeDiscoveriesOut.IpamResourceDiscoveries) != 1 {
		t.Fatalf("expected 1 ipam resource discovery, got %d", len(describeDiscoveriesOut.IpamResourceDiscoveries))
	}

	describeAssociationsOut, err := client.DescribeIpamResourceDiscoveryAssociations(ctx, &awsec2.DescribeIpamResourceDiscoveryAssociationsInput{
		IpamResourceDiscoveryAssociationIds: []string{associationID},
		MaxResults:                          aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe ipam resource discovery associations: %v", err)
	}
	if len(describeAssociationsOut.IpamResourceDiscoveryAssociations) != 1 {
		t.Fatalf("expected 1 ipam resource discovery association, got %d", len(describeAssociationsOut.IpamResourceDiscoveryAssociations))
	}

	describeIpamScopesOut, err := client.DescribeIpamScopes(ctx, &awsec2.DescribeIpamScopesInput{
		IpamScopeIds: []string{ipamScopeID},
		MaxResults:   aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe ipam scopes: %v", err)
	}
	if len(describeIpamScopesOut.IpamScopes) != 1 {
		t.Fatalf("expected 1 ipam scope, got %d", len(describeIpamScopesOut.IpamScopes))
	}

	describeIpamsOut, err := client.DescribeIpams(ctx, &awsec2.DescribeIpamsInput{
		IpamIds:    []string{ipamID},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe ipams: %v", err)
	}
	if len(describeIpamsOut.Ipams) != 1 {
		t.Fatalf("expected 1 ipam, got %d", len(describeIpamsOut.Ipams))
	}

	createVpcOut, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{CidrBlock: aws.String("10.118.0.0/16")})
	if err != nil || createVpcOut.Vpc == nil || createVpcOut.Vpc.VpcId == nil {
		t.Fatalf("create vpc: %v", err)
	}
	if _, err := client.AssociateVpcCidrBlock(ctx, &awsec2.AssociateVpcCidrBlockInput{
		VpcId:    createVpcOut.Vpc.VpcId,
		Ipv6Pool: aws.String("ipv6pool-stage118"),
	}); err != nil {
		t.Fatalf("associate vpc cidr block (ipv6 pool): %v", err)
	}

	describeIpv6PoolsOut, err := client.DescribeIpv6Pools(ctx, &awsec2.DescribeIpv6PoolsInput{
		PoolIds:    []string{"ipv6pool-stage118"},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe ipv6 pools: %v", err)
	}
	if len(describeIpv6PoolsOut.Ipv6Pools) != 1 {
		t.Fatalf("expected 1 ipv6 pool, got %d", len(describeIpv6PoolsOut.Ipv6Pools))
	}

	createLaunchTemplateOut, err := client.CreateLaunchTemplate(ctx, &awsec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String("stage118-template"),
		LaunchTemplateData: &awsec2types.RequestLaunchTemplateData{ImageId: aws.String("ami-stage118")},
	})
	if err != nil {
		t.Fatalf("create launch template: %v", err)
	}
	if createLaunchTemplateOut.LaunchTemplate == nil || createLaunchTemplateOut.LaunchTemplate.LaunchTemplateId == nil {
		t.Fatalf("expected created launch template")
	}
	launchTemplateID := aws.ToString(createLaunchTemplateOut.LaunchTemplate.LaunchTemplateId)

	if _, err := client.CreateLaunchTemplateVersion(ctx, &awsec2.CreateLaunchTemplateVersionInput{
		LaunchTemplateId: aws.String(launchTemplateID),
		LaunchTemplateData: &awsec2types.RequestLaunchTemplateData{
			ImageId: aws.String("ami-stage118-v2"),
		},
		VersionDescription: aws.String("stage118-version-2"),
	}); err != nil {
		t.Fatalf("create launch template version: %v", err)
	}

	describeLaunchTemplateVersionsOut, err := client.DescribeLaunchTemplateVersions(ctx, &awsec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateId: aws.String(launchTemplateID),
		Versions:         []string{"$Latest"},
		MaxResults:       aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe launch template versions: %v", err)
	}
	if len(describeLaunchTemplateVersionsOut.LaunchTemplateVersions) == 0 {
		t.Fatalf("expected launch template versions")
	}
}

func TestEC2Stage118ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeInstanceTypes",
		"DescribeIpamByoasn",
		"DescribeIpamExternalResourceVerificationTokens",
		"DescribeIpamPools",
		"DescribeIpamResourceDiscoveries",
		"DescribeIpamResourceDiscoveryAssociations",
		"DescribeIpamScopes",
		"DescribeIpams",
		"DescribeIpv6Pools",
		"DescribeLaunchTemplateVersions",
	}

	paramsByAction := map[string]map[string]string{
		"DescribeInstanceTypes": {
			"InstanceType.1": "m5.large",
			"MaxResults":     "10",
		},
		"DescribeIpamByoasn": {
			"MaxResults": "10",
		},
		"DescribeIpamExternalResourceVerificationTokens": {
			"IpamExternalResourceVerificationTokenId.1": "ipam-ert-0000000118",
			"MaxResults": "10",
		},
		"DescribeIpamPools": {
			"IpamPoolId.1": "ipam-pool-0000000118",
			"MaxResults":   "10",
		},
		"DescribeIpamResourceDiscoveries": {
			"IpamResourceDiscoveryId.1": "ipam-rd-0000000118",
			"MaxResults":                "10",
		},
		"DescribeIpamResourceDiscoveryAssociations": {
			"IpamResourceDiscoveryAssociationId.1": "ipam-rd-assoc-0000000118",
			"MaxResults":                           "10",
		},
		"DescribeIpamScopes": {
			"IpamScopeId.1": "ipam-scope-0000000118",
			"MaxResults":    "10",
		},
		"DescribeIpams": {
			"IpamId.1":   "ipam-0000000118",
			"MaxResults": "10",
		},
		"DescribeIpv6Pools": {
			"PoolId.1":   "ipv6pool-stage118",
			"MaxResults": "10",
		},
		"DescribeLaunchTemplateVersions": {
			"LaunchTemplateId": "lt-0000000118",
			"MaxResults":       "10",
		},
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, paramsByAction[action])
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
