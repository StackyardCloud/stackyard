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

func TestEC2Stage17SDKLifecycle(t *testing.T) {
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

	createOut, err := client.CreateCustomerGateway(ctx, &awsec2.CreateCustomerGatewayInput{
		Type:       awsec2types.GatewayTypeIpsec1,
		IpAddress:  aws.String("198.51.100.10"),
		BgpAsn:     aws.Int32(65010),
		DeviceName: aws.String("stage17-device"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeCustomerGateway,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("stage17")}},
			},
		},
	})
	if err != nil || createOut.CustomerGateway == nil || aws.ToString(createOut.CustomerGateway.CustomerGatewayId) == "" || aws.ToString(createOut.CustomerGateway.State) != "available" {
		t.Fatalf("create customer gateway: %v", err)
	}
	customerGatewayID := aws.ToString(createOut.CustomerGateway.CustomerGatewayId)

	describeOut, err := client.DescribeCustomerGateways(ctx, &awsec2.DescribeCustomerGatewaysInput{
		CustomerGatewayIds: []string{customerGatewayID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("ip-address"), Values: []string{"198.51.100.10"}},
			{Name: aws.String("tag:env"), Values: []string{"stage17"}},
			{Name: aws.String("tag-key"), Values: []string{"env"}},
		},
	})
	if err != nil || len(describeOut.CustomerGateways) != 1 || aws.ToString(describeOut.CustomerGateways[0].CustomerGatewayId) != customerGatewayID {
		t.Fatalf("describe customer gateways: %v", err)
	}

	if _, err := client.DeleteCustomerGateway(ctx, &awsec2.DeleteCustomerGatewayInput{
		CustomerGatewayId: aws.String(customerGatewayID),
	}); err != nil {
		t.Fatalf("delete customer gateway: %v", err)
	}

	describeAfterDeleteOut, err := client.DescribeCustomerGateways(ctx, &awsec2.DescribeCustomerGatewaysInput{
		CustomerGatewayIds: []string{customerGatewayID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("state"), Values: []string{"deleted"}},
		},
	})
	if err != nil {
		t.Fatalf("describe customer gateways after delete: %v", err)
	}
	if len(describeAfterDeleteOut.CustomerGateways) != 1 || aws.ToString(describeAfterDeleteOut.CustomerGateways[0].State) != "deleted" {
		t.Fatalf("expected deleted customer gateway after delete")
	}
}

func TestEC2Stage17ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateCustomerGateway",
		"DescribeCustomerGateways",
		"DeleteCustomerGateway",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateCustomerGateway":
			params["Type"] = "ipsec.1"
			params["IpAddress"] = "198.51.100.10"
			params["BgpAsn"] = "65010"
		case "DescribeCustomerGateways":
			params["Filter.1.Name"] = "type"
			params["Filter.1.Value.1"] = "ipsec.1"
		case "DeleteCustomerGateway":
			params["CustomerGatewayId"] = "cgw-00000001"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
