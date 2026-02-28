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

func TestEC2Stage3SDKLifecycle(t *testing.T) {
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

	createVpcOut, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{CidrBlock: aws.String("10.40.0.0/16")})
	if err != nil || createVpcOut.Vpc == nil || createVpcOut.Vpc.VpcId == nil {
		t.Fatalf("create vpc: %v", err)
	}
	vpcID := aws.ToString(createVpcOut.Vpc.VpcId)

	createSubnetOut, err := client.CreateSubnet(ctx, &awsec2.CreateSubnetInput{
		VpcId:            aws.String(vpcID),
		CidrBlock:        aws.String("10.40.1.0/24"),
		AvailabilityZone: aws.String("us-east-1a"),
	})
	if err != nil || createSubnetOut.Subnet == nil || createSubnetOut.Subnet.SubnetId == nil {
		t.Fatalf("create subnet: %v", err)
	}
	subnetID := aws.ToString(createSubnetOut.Subnet.SubnetId)

	createIgwOut1, err := client.CreateInternetGateway(ctx, &awsec2.CreateInternetGatewayInput{})
	if err != nil || createIgwOut1.InternetGateway == nil || createIgwOut1.InternetGateway.InternetGatewayId == nil {
		t.Fatalf("create internet gateway #1: %v", err)
	}
	igwID1 := aws.ToString(createIgwOut1.InternetGateway.InternetGatewayId)
	if _, err := client.AttachInternetGateway(ctx, &awsec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(igwID1),
		VpcId:             aws.String(vpcID),
	}); err != nil {
		t.Fatalf("attach internet gateway #1: %v", err)
	}

	createIgwOut2, err := client.CreateInternetGateway(ctx, &awsec2.CreateInternetGatewayInput{})
	if err != nil || createIgwOut2.InternetGateway == nil || createIgwOut2.InternetGateway.InternetGatewayId == nil {
		t.Fatalf("create internet gateway #2: %v", err)
	}
	igwID2 := aws.ToString(createIgwOut2.InternetGateway.InternetGatewayId)
	if _, err := client.AttachInternetGateway(ctx, &awsec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(igwID2),
		VpcId:             aws.String(vpcID),
	}); err != nil {
		t.Fatalf("attach internet gateway #2: %v", err)
	}

	createRouteTableOut1, err := client.CreateRouteTable(ctx, &awsec2.CreateRouteTableInput{VpcId: aws.String(vpcID)})
	if err != nil || createRouteTableOut1.RouteTable == nil || createRouteTableOut1.RouteTable.RouteTableId == nil {
		t.Fatalf("create route table #1: %v", err)
	}
	routeTableID1 := aws.ToString(createRouteTableOut1.RouteTable.RouteTableId)
	if _, err := client.CreateRoute(ctx, &awsec2.CreateRouteInput{
		RouteTableId:         aws.String(routeTableID1),
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(igwID1),
	}); err != nil {
		t.Fatalf("create route: %v", err)
	}
	if _, err := client.ReplaceRoute(ctx, &awsec2.ReplaceRouteInput{
		RouteTableId:         aws.String(routeTableID1),
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(igwID2),
	}); err != nil {
		t.Fatalf("replace route: %v", err)
	}

	createRouteTableOut2, err := client.CreateRouteTable(ctx, &awsec2.CreateRouteTableInput{VpcId: aws.String(vpcID)})
	if err != nil || createRouteTableOut2.RouteTable == nil || createRouteTableOut2.RouteTable.RouteTableId == nil {
		t.Fatalf("create route table #2: %v", err)
	}
	routeTableID2 := aws.ToString(createRouteTableOut2.RouteTable.RouteTableId)
	associateOut, err := client.AssociateRouteTable(ctx, &awsec2.AssociateRouteTableInput{
		RouteTableId: aws.String(routeTableID1),
		SubnetId:     aws.String(subnetID),
	})
	if err != nil || associateOut.AssociationId == nil {
		t.Fatalf("associate route table: %v", err)
	}
	associationID := aws.ToString(associateOut.AssociationId)

	replaceAssocOut, err := client.ReplaceRouteTableAssociation(ctx, &awsec2.ReplaceRouteTableAssociationInput{
		AssociationId: aws.String(associationID),
		RouteTableId:  aws.String(routeTableID2),
	})
	if err != nil || replaceAssocOut.NewAssociationId == nil {
		t.Fatalf("replace route table association: %v", err)
	}

	createAclOut, err := client.CreateNetworkAcl(ctx, &awsec2.CreateNetworkAclInput{VpcId: aws.String(vpcID)})
	if err != nil || createAclOut.NetworkAcl == nil || createAclOut.NetworkAcl.NetworkAclId == nil {
		t.Fatalf("create network acl: %v", err)
	}
	aclID := aws.ToString(createAclOut.NetworkAcl.NetworkAclId)

	if _, err := client.CreateNetworkAclEntry(ctx, &awsec2.CreateNetworkAclEntryInput{
		NetworkAclId: aws.String(aclID),
		RuleNumber:   aws.Int32(110),
		Protocol:     aws.String("6"),
		RuleAction:   awsec2types.RuleActionAllow,
		Egress:       aws.Bool(false),
		CidrBlock:    aws.String("0.0.0.0/0"),
	}); err != nil {
		t.Fatalf("create network acl entry: %v", err)
	}
	if _, err := client.ReplaceNetworkAclEntry(ctx, &awsec2.ReplaceNetworkAclEntryInput{
		NetworkAclId: aws.String(aclID),
		RuleNumber:   aws.Int32(110),
		Protocol:     aws.String("6"),
		RuleAction:   awsec2types.RuleActionDeny,
		Egress:       aws.Bool(false),
		CidrBlock:    aws.String("0.0.0.0/0"),
	}); err != nil {
		t.Fatalf("replace network acl entry: %v", err)
	}
	if _, err := client.DeleteNetworkAclEntry(ctx, &awsec2.DeleteNetworkAclEntryInput{
		NetworkAclId: aws.String(aclID),
		RuleNumber:   aws.Int32(110),
		Egress:       aws.Bool(false),
	}); err != nil {
		t.Fatalf("delete network acl entry: %v", err)
	}

	createSGOut1, err := client.CreateSecurityGroup(ctx, &awsec2.CreateSecurityGroupInput{
		GroupName:   aws.String("stage3-sg-1"),
		Description: aws.String("stage3 sg 1"),
		VpcId:       aws.String(vpcID),
	})
	if err != nil || createSGOut1.GroupId == nil {
		t.Fatalf("create security group #1: %v", err)
	}
	sgID1 := aws.ToString(createSGOut1.GroupId)
	createSGOut2, err := client.CreateSecurityGroup(ctx, &awsec2.CreateSecurityGroupInput{
		GroupName:   aws.String("stage3-sg-2"),
		Description: aws.String("stage3 sg 2"),
		VpcId:       aws.String(vpcID),
	})
	if err != nil || createSGOut2.GroupId == nil {
		t.Fatalf("create security group #2: %v", err)
	}
	sgID2 := aws.ToString(createSGOut2.GroupId)

	createInterfaceOut, err := client.CreateNetworkInterface(ctx, &awsec2.CreateNetworkInterfaceInput{
		SubnetId: aws.String(subnetID),
		Groups:   []string{sgID1},
	})
	if err != nil || createInterfaceOut.NetworkInterface == nil || createInterfaceOut.NetworkInterface.NetworkInterfaceId == nil {
		t.Fatalf("create network interface: %v", err)
	}
	interfaceID := aws.ToString(createInterfaceOut.NetworkInterface.NetworkInterfaceId)

	if _, err := client.ModifyNetworkInterfaceAttribute(ctx, &awsec2.ModifyNetworkInterfaceAttributeInput{
		NetworkInterfaceId: aws.String(interfaceID),
		Description:        &awsec2types.AttributeValue{Value: aws.String("updated interface")},
		SourceDestCheck:    &awsec2types.AttributeBooleanValue{Value: aws.Bool(false)},
		Groups:             []string{sgID2},
	}); err != nil {
		t.Fatalf("modify network interface attribute: %v", err)
	}
	describeInterfaceOut, err := client.DescribeNetworkInterfaces(ctx, &awsec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []string{interfaceID},
	})
	if err != nil || len(describeInterfaceOut.NetworkInterfaces) != 1 {
		t.Fatalf("describe network interfaces: %v", err)
	}
	if describeInterfaceOut.NetworkInterfaces[0].SourceDestCheck == nil || aws.ToBool(describeInterfaceOut.NetworkInterfaces[0].SourceDestCheck) {
		t.Fatalf("expected source dest check disabled")
	}

	if _, err := client.ModifySubnetAttribute(ctx, &awsec2.ModifySubnetAttributeInput{
		SubnetId:            aws.String(subnetID),
		MapPublicIpOnLaunch: &awsec2types.AttributeBooleanValue{Value: aws.Bool(false)},
	}); err != nil {
		t.Fatalf("modify subnet attribute: %v", err)
	}
	describeSubnetsOut, err := client.DescribeSubnets(ctx, &awsec2.DescribeSubnetsInput{SubnetIds: []string{subnetID}})
	if err != nil || len(describeSubnetsOut.Subnets) != 1 {
		t.Fatalf("describe subnets: %v", err)
	}
	if describeSubnetsOut.Subnets[0].MapPublicIpOnLaunch == nil || aws.ToBool(describeSubnetsOut.Subnets[0].MapPublicIpOnLaunch) {
		t.Fatalf("expected map public ip on launch disabled")
	}

	if _, err := client.ModifyVpcAttribute(ctx, &awsec2.ModifyVpcAttributeInput{
		VpcId:            aws.String(vpcID),
		EnableDnsSupport: &awsec2types.AttributeBooleanValue{Value: aws.Bool(false)},
	}); err != nil {
		t.Fatalf("modify vpc attribute dns support: %v", err)
	}
	if _, err := client.ModifyVpcAttribute(ctx, &awsec2.ModifyVpcAttributeInput{
		VpcId:              aws.String(vpcID),
		EnableDnsHostnames: &awsec2types.AttributeBooleanValue{Value: aws.Bool(true)},
	}); err != nil {
		t.Fatalf("modify vpc attribute dns hostnames: %v", err)
	}

	describeVpcDnsSupportOut, err := client.DescribeVpcAttribute(ctx, &awsec2.DescribeVpcAttributeInput{
		VpcId:     aws.String(vpcID),
		Attribute: awsec2types.VpcAttributeNameEnableDnsSupport,
	})
	if err != nil || describeVpcDnsSupportOut.EnableDnsSupport == nil || aws.ToBool(describeVpcDnsSupportOut.EnableDnsSupport.Value) {
		t.Fatalf("describe vpc attribute dns support: %v", err)
	}
	describeVpcDnsHostnamesOut, err := client.DescribeVpcAttribute(ctx, &awsec2.DescribeVpcAttributeInput{
		VpcId:     aws.String(vpcID),
		Attribute: awsec2types.VpcAttributeNameEnableDnsHostnames,
	})
	if err != nil || describeVpcDnsHostnamesOut.EnableDnsHostnames == nil || !aws.ToBool(describeVpcDnsHostnamesOut.EnableDnsHostnames.Value) {
		t.Fatalf("describe vpc attribute dns hostnames: %v", err)
	}

	describeAccountAttributesOut, err := client.DescribeAccountAttributes(ctx, &awsec2.DescribeAccountAttributesInput{
		AttributeNames: []awsec2types.AccountAttributeName{awsec2types.AccountAttributeNameDefaultVpc},
	})
	if err != nil {
		t.Fatalf("describe account attributes: %v", err)
	}
	if len(describeAccountAttributesOut.AccountAttributes) == 0 {
		t.Fatalf("expected account attributes")
	}
}

func TestEC2Stage3ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateNetworkAclEntry",
		"ReplaceNetworkAclEntry",
		"DeleteNetworkAclEntry",
		"ReplaceRoute",
		"ReplaceRouteTableAssociation",
		"ModifyNetworkInterfaceAttribute",
		"ModifySubnetAttribute",
		"ModifyVpcAttribute",
		"DescribeVpcAttribute",
		"DescribeAccountAttributes",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for idx, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateNetworkAclEntry", "ReplaceNetworkAclEntry":
			params["NetworkAclId"] = "acl-00000001"
			params["RuleNumber"] = "120"
			params["Protocol"] = "6"
			params["RuleAction"] = "allow"
			params["Egress"] = "false"
			params["CidrBlock"] = "0.0.0.0/0"
		case "DeleteNetworkAclEntry":
			params["NetworkAclId"] = "acl-00000001"
			params["RuleNumber"] = "100"
			params["Egress"] = "false"
		case "ReplaceRoute":
			params["RouteTableId"] = "rtb-00000001"
			params["DestinationCidrBlock"] = "10.0.0.0/16"
			params["GatewayId"] = "local"
		case "ReplaceRouteTableAssociation":
			params["AssociationId"] = "rtbassoc-00000001"
			params["RouteTableId"] = "rtb-00000001"
		case "ModifyNetworkInterfaceAttribute":
			params["NetworkInterfaceId"] = "eni-0000000" + strconv.Itoa(idx+1)
			params["Description.Value"] = "updated"
		case "ModifySubnetAttribute":
			params["SubnetId"] = "subnet-00000001"
			params["MapPublicIpOnLaunch.Value"] = "false"
		case "ModifyVpcAttribute":
			params["VpcId"] = "vpc-00000001"
			params["EnableDnsSupport.Value"] = "true"
		case "DescribeVpcAttribute":
			params["VpcId"] = "vpc-00000001"
			params["Attribute"] = "enableDnsSupport"
		case "DescribeAccountAttributes":
			params["AttributeName.1"] = "default-vpc"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
