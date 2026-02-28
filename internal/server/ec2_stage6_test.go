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

func TestEC2Stage6SDKLifecycle(t *testing.T) {
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

	allocateOut, err := client.AllocateAddress(ctx, &awsec2.AllocateAddressInput{Domain: awsec2types.DomainTypeVpc})
	if err != nil || allocateOut.AllocationId == nil {
		t.Fatalf("allocate address: %v", err)
	}
	allocationID := aws.ToString(allocateOut.AllocationId)

	createNatOut, err := client.CreateNatGateway(ctx, &awsec2.CreateNatGatewayInput{
		SubnetId:     aws.String("subnet-00000001"),
		AllocationId: aws.String(allocationID),
		TagSpecifications: []awsec2types.TagSpecification{{
			ResourceType: awsec2types.ResourceTypeNatgateway,
			Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("stage6")}},
		}},
	})
	if err != nil || createNatOut.NatGateway == nil || createNatOut.NatGateway.NatGatewayId == nil {
		t.Fatalf("create nat gateway: %v", err)
	}
	natGatewayID := aws.ToString(createNatOut.NatGateway.NatGatewayId)

	describeNatOut, err := client.DescribeNatGateways(ctx, &awsec2.DescribeNatGatewaysInput{NatGatewayIds: []string{natGatewayID}})
	if err != nil {
		t.Fatalf("describe nat gateways: %v", err)
	}
	if len(describeNatOut.NatGateways) != 1 {
		t.Fatalf("expected one nat gateway")
	}

	deleteNatOut, err := client.DeleteNatGateway(ctx, &awsec2.DeleteNatGatewayInput{NatGatewayId: aws.String(natGatewayID)})
	if err != nil || deleteNatOut.NatGatewayId == nil || aws.ToString(deleteNatOut.NatGatewayId) != natGatewayID {
		t.Fatalf("delete nat gateway: %v", err)
	}

	if _, err := client.ReleaseAddress(ctx, &awsec2.ReleaseAddressInput{AllocationId: aws.String(allocationID)}); err != nil {
		t.Fatalf("release address: %v", err)
	}

	createVpcOut1, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{CidrBlock: aws.String("10.61.0.0/16")})
	if err != nil || createVpcOut1.Vpc == nil || createVpcOut1.Vpc.VpcId == nil {
		t.Fatalf("create vpc #1: %v", err)
	}
	vpcID1 := aws.ToString(createVpcOut1.Vpc.VpcId)

	createVpcOut2, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{CidrBlock: aws.String("10.62.0.0/16")})
	if err != nil || createVpcOut2.Vpc == nil || createVpcOut2.Vpc.VpcId == nil {
		t.Fatalf("create vpc #2: %v", err)
	}
	vpcID2 := aws.ToString(createVpcOut2.Vpc.VpcId)

	createPcxOut, err := client.CreateVpcPeeringConnection(ctx, &awsec2.CreateVpcPeeringConnectionInput{
		VpcId:     aws.String(vpcID1),
		PeerVpcId: aws.String(vpcID2),
		TagSpecifications: []awsec2types.TagSpecification{{
			ResourceType: awsec2types.ResourceTypeVpcPeeringConnection,
			Tags:         []awsec2types.Tag{{Key: aws.String("name"), Value: aws.String("stage6")}},
		}},
	})
	if err != nil || createPcxOut.VpcPeeringConnection == nil || createPcxOut.VpcPeeringConnection.VpcPeeringConnectionId == nil {
		t.Fatalf("create vpc peering connection: %v", err)
	}
	pcxID := aws.ToString(createPcxOut.VpcPeeringConnection.VpcPeeringConnectionId)

	describePcxOut, err := client.DescribeVpcPeeringConnections(ctx, &awsec2.DescribeVpcPeeringConnectionsInput{
		VpcPeeringConnectionIds: []string{pcxID},
	})
	if err != nil {
		t.Fatalf("describe vpc peering connections: %v", err)
	}
	if len(describePcxOut.VpcPeeringConnections) != 1 {
		t.Fatalf("expected one vpc peering connection")
	}

	if _, err := client.AcceptVpcPeeringConnection(ctx, &awsec2.AcceptVpcPeeringConnectionInput{VpcPeeringConnectionId: aws.String(pcxID)}); err != nil {
		t.Fatalf("accept vpc peering connection: %v", err)
	}

	if _, err := client.DeleteVpcPeeringConnection(ctx, &awsec2.DeleteVpcPeeringConnectionInput{VpcPeeringConnectionId: aws.String(pcxID)}); err != nil {
		t.Fatalf("delete vpc peering connection: %v", err)
	}

	createPcxOut2, err := client.CreateVpcPeeringConnection(ctx, &awsec2.CreateVpcPeeringConnectionInput{
		VpcId:     aws.String(vpcID1),
		PeerVpcId: aws.String(vpcID2),
	})
	if err != nil || createPcxOut2.VpcPeeringConnection == nil || createPcxOut2.VpcPeeringConnection.VpcPeeringConnectionId == nil {
		t.Fatalf("create second vpc peering connection: %v", err)
	}
	pcxID2 := aws.ToString(createPcxOut2.VpcPeeringConnection.VpcPeeringConnectionId)

	if _, err := client.RejectVpcPeeringConnection(ctx, &awsec2.RejectVpcPeeringConnectionInput{VpcPeeringConnectionId: aws.String(pcxID2)}); err != nil {
		t.Fatalf("reject vpc peering connection: %v", err)
	}
}

func TestEC2Stage6ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateNatGateway",
		"DescribeNatGateways",
		"DeleteNatGateway",
		"CreateVpcPeeringConnection",
		"DescribeVpcPeeringConnections",
		"AcceptVpcPeeringConnection",
		"RejectVpcPeeringConnection",
		"DeleteVpcPeeringConnection",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for idx, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateNatGateway":
			params["SubnetId"] = "subnet-00000001"
			params["AllocationId"] = "eipalloc-00000001"
		case "DescribeNatGateways":
			params["NatGatewayId.1"] = "nat-00000001"
		case "DeleteNatGateway":
			params["NatGatewayId"] = "nat-0000000" + strconv.Itoa(idx+1)
		case "CreateVpcPeeringConnection":
			params["VpcId"] = "vpc-00000001"
			params["PeerVpcId"] = "vpc-00000002"
		case "DescribeVpcPeeringConnections":
			params["VpcPeeringConnectionId.1"] = "pcx-00000001"
		case "AcceptVpcPeeringConnection", "RejectVpcPeeringConnection", "DeleteVpcPeeringConnection":
			params["VpcPeeringConnectionId"] = "pcx-00000001"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
