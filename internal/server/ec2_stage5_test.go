package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage5SDKLifecycle(t *testing.T) {
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

	createDhcpOut, err := client.CreateDhcpOptions(ctx, &awsec2.CreateDhcpOptionsInput{
		DhcpConfigurations: []awsec2types.NewDhcpConfiguration{
			{Key: aws.String("domain-name-servers"), Values: []string{"AmazonProvidedDNS"}},
			{Key: aws.String("domain-name"), Values: []string{"example.internal"}},
		},
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeDhcpOptions,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("stage5")}},
			},
		},
	})
	if err != nil || createDhcpOut.DhcpOptions == nil || createDhcpOut.DhcpOptions.DhcpOptionsId == nil {
		t.Fatalf("create dhcp options: %v", err)
	}
	dhcpOptionsID := aws.ToString(createDhcpOut.DhcpOptions.DhcpOptionsId)

	if _, err := client.DescribeDhcpOptions(ctx, &awsec2.DescribeDhcpOptionsInput{
		DhcpOptionsIds: []string{dhcpOptionsID},
	}); err != nil {
		t.Fatalf("describe dhcp options: %v", err)
	}

	if _, err := client.AssociateDhcpOptions(ctx, &awsec2.AssociateDhcpOptionsInput{
		DhcpOptionsId: aws.String(dhcpOptionsID),
		VpcId:         aws.String("vpc-00000001"),
	}); err != nil {
		t.Fatalf("associate dhcp options: %v", err)
	}

	createEgressOut, err := client.CreateEgressOnlyInternetGateway(ctx, &awsec2.CreateEgressOnlyInternetGatewayInput{
		VpcId: aws.String("vpc-00000001"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeEgressOnlyInternetGateway,
				Tags:         []awsec2types.Tag{{Key: aws.String("name"), Value: aws.String("stage5-egress")}},
			},
		},
	})
	if err != nil || createEgressOut.EgressOnlyInternetGateway == nil || createEgressOut.EgressOnlyInternetGateway.EgressOnlyInternetGatewayId == nil {
		t.Fatalf("create egress-only internet gateway: %v", err)
	}
	egressID := aws.ToString(createEgressOut.EgressOnlyInternetGateway.EgressOnlyInternetGatewayId)

	if _, err := client.DescribeEgressOnlyInternetGateways(ctx, &awsec2.DescribeEgressOnlyInternetGatewaysInput{
		EgressOnlyInternetGatewayIds: []string{egressID},
	}); err != nil {
		t.Fatalf("describe egress-only internet gateways: %v", err)
	}

	allocateOut, err := client.AllocateAddress(ctx, &awsec2.AllocateAddressInput{
		Domain: awsec2types.DomainTypeVpc,
	})
	if err != nil || allocateOut.AllocationId == nil {
		t.Fatalf("allocate address: %v", err)
	}
	allocationID := aws.ToString(allocateOut.AllocationId)
	describeAddressAttrsOut, err := client.DescribeAddressesAttribute(ctx, &awsec2.DescribeAddressesAttributeInput{
		AllocationIds: []string{allocationID},
		Attribute:     awsec2types.AddressAttributeNameDomainName,
	})
	if err != nil {
		t.Fatalf("describe addresses attribute: %v", err)
	}
	if len(describeAddressAttrsOut.Addresses) != 1 {
		t.Fatalf("expected one address attribute item")
	}
	if _, err := client.ReleaseAddress(ctx, &awsec2.ReleaseAddressInput{AllocationId: aws.String(allocationID)}); err != nil {
		t.Fatalf("release address: %v", err)
	}

	if _, err := client.DeleteEgressOnlyInternetGateway(ctx, &awsec2.DeleteEgressOnlyInternetGatewayInput{
		EgressOnlyInternetGatewayId: aws.String(egressID),
	}); err != nil {
		t.Fatalf("delete egress-only internet gateway: %v", err)
	}

	if _, err := client.AssociateDhcpOptions(ctx, &awsec2.AssociateDhcpOptionsInput{
		DhcpOptionsId: aws.String("default"),
		VpcId:         aws.String("vpc-00000001"),
	}); err != nil {
		t.Fatalf("associate default dhcp options: %v", err)
	}
	if _, err := client.DeleteDhcpOptions(ctx, &awsec2.DeleteDhcpOptionsInput{
		DhcpOptionsId: aws.String(dhcpOptionsID),
	}); err != nil {
		t.Fatalf("delete dhcp options: %v", err)
	}
}

func TestEC2Stage5ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateDhcpOptions",
		"DescribeDhcpOptions",
		"AssociateDhcpOptions",
		"DeleteDhcpOptions",
		"CreateEgressOnlyInternetGateway",
		"DescribeEgressOnlyInternetGateways",
		"DeleteEgressOnlyInternetGateway",
		"DescribeAddressesAttribute",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for idx, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateDhcpOptions":
			params["DhcpConfiguration.1.Key"] = "domain-name-servers"
			params["DhcpConfiguration.1.Value.1"] = "AmazonProvidedDNS"
		case "DescribeDhcpOptions":
			params["DhcpOptionsId.1"] = "dopt-00000000"
		case "AssociateDhcpOptions":
			params["DhcpOptionsId"] = "default"
			params["VpcId"] = "vpc-00000001"
		case "DeleteDhcpOptions":
			params["DhcpOptionsId"] = "dopt-0000000" + strconv.Itoa(idx+1)
		case "CreateEgressOnlyInternetGateway":
			params["VpcId"] = "vpc-00000001"
		case "DescribeEgressOnlyInternetGateways":
			params["EgressOnlyInternetGatewayId.1"] = "eigw-00000001"
		case "DeleteEgressOnlyInternetGateway":
			params["EgressOnlyInternetGatewayId"] = "eigw-00000001"
		case "DescribeAddressesAttribute":
			params["Attribute"] = "domain-name"
			params["AllocationId.1"] = "eipalloc-00000001"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
