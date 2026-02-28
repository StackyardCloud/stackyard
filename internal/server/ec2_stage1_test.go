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

func TestEC2Stage1SDKNetworkingLifecycle(t *testing.T) {
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

	createVpcOut, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{
		CidrBlock: aws.String("10.20.0.0/16"),
		TagSpecifications: []awsec2types.TagSpecification{
			{ResourceType: awsec2types.ResourceTypeVpc, Tags: []awsec2types.Tag{{Key: aws.String("name"), Value: aws.String("stage1")}}},
		},
	})
	if err != nil || createVpcOut.Vpc == nil || createVpcOut.Vpc.VpcId == nil {
		t.Fatalf("create vpc: %v", err)
	}
	vpcID := aws.ToString(createVpcOut.Vpc.VpcId)

	createSubnetOut, err := client.CreateSubnet(ctx, &awsec2.CreateSubnetInput{
		VpcId:            aws.String(vpcID),
		CidrBlock:        aws.String("10.20.1.0/24"),
		AvailabilityZone: aws.String("us-east-1a"),
	})
	if err != nil || createSubnetOut.Subnet == nil || createSubnetOut.Subnet.SubnetId == nil {
		t.Fatalf("create subnet: %v", err)
	}
	subnetID := aws.ToString(createSubnetOut.Subnet.SubnetId)

	createIgwOut, err := client.CreateInternetGateway(ctx, &awsec2.CreateInternetGatewayInput{})
	if err != nil || createIgwOut.InternetGateway == nil || createIgwOut.InternetGateway.InternetGatewayId == nil {
		t.Fatalf("create internet gateway: %v", err)
	}
	igwID := aws.ToString(createIgwOut.InternetGateway.InternetGatewayId)
	if _, err := client.AttachInternetGateway(ctx, &awsec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(igwID),
		VpcId:             aws.String(vpcID),
	}); err != nil {
		t.Fatalf("attach internet gateway: %v", err)
	}

	createRouteTableOut, err := client.CreateRouteTable(ctx, &awsec2.CreateRouteTableInput{VpcId: aws.String(vpcID)})
	if err != nil || createRouteTableOut.RouteTable == nil || createRouteTableOut.RouteTable.RouteTableId == nil {
		t.Fatalf("create route table: %v", err)
	}
	routeTableID := aws.ToString(createRouteTableOut.RouteTable.RouteTableId)
	associateRouteTableOut, err := client.AssociateRouteTable(ctx, &awsec2.AssociateRouteTableInput{
		RouteTableId: aws.String(routeTableID),
		SubnetId:     aws.String(subnetID),
	})
	if err != nil || associateRouteTableOut.AssociationId == nil {
		t.Fatalf("associate route table: %v", err)
	}
	associationID := aws.ToString(associateRouteTableOut.AssociationId)
	if _, err := client.CreateRoute(ctx, &awsec2.CreateRouteInput{
		RouteTableId:         aws.String(routeTableID),
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(igwID),
	}); err != nil {
		t.Fatalf("create route: %v", err)
	}

	if _, err := client.DescribeRouteTables(ctx, &awsec2.DescribeRouteTablesInput{RouteTableIds: []string{routeTableID}}); err != nil {
		t.Fatalf("describe route tables: %v", err)
	}

	createAclOut, err := client.CreateNetworkAcl(ctx, &awsec2.CreateNetworkAclInput{VpcId: aws.String(vpcID)})
	if err != nil || createAclOut.NetworkAcl == nil || createAclOut.NetworkAcl.NetworkAclId == nil {
		t.Fatalf("create network acl: %v", err)
	}
	aclID := aws.ToString(createAclOut.NetworkAcl.NetworkAclId)
	if _, err := client.DescribeNetworkAcls(ctx, &awsec2.DescribeNetworkAclsInput{NetworkAclIds: []string{aclID}}); err != nil {
		t.Fatalf("describe network acls: %v", err)
	}

	createSGOut, err := client.CreateSecurityGroup(ctx, &awsec2.CreateSecurityGroupInput{
		GroupName:   aws.String("stage1-sg"),
		Description: aws.String("stage1"),
		VpcId:       aws.String(vpcID),
	})
	if err != nil || createSGOut.GroupId == nil {
		t.Fatalf("create security group: %v", err)
	}
	sgID := aws.ToString(createSGOut.GroupId)

	runOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:          aws.String("ami-12345678"),
		MinCount:         aws.Int32(1),
		MaxCount:         aws.Int32(1),
		InstanceType:     awsec2types.InstanceTypeT3Micro,
		SubnetId:         aws.String(subnetID),
		SecurityGroupIds: []string{sgID},
	})
	if err != nil || len(runOut.Instances) == 0 || runOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)

	createInterfaceOut, err := client.CreateNetworkInterface(ctx, &awsec2.CreateNetworkInterfaceInput{
		SubnetId:         aws.String(subnetID),
		Groups:           []string{sgID},
		Description:      aws.String("secondary"),
		PrivateIpAddress: aws.String("10.20.1.50"),
	})
	if err != nil || createInterfaceOut.NetworkInterface == nil || createInterfaceOut.NetworkInterface.NetworkInterfaceId == nil {
		t.Fatalf("create network interface: %v", err)
	}
	interfaceID := aws.ToString(createInterfaceOut.NetworkInterface.NetworkInterfaceId)

	attachInterfaceOut, err := client.AttachNetworkInterface(ctx, &awsec2.AttachNetworkInterfaceInput{
		NetworkInterfaceId: aws.String(interfaceID),
		InstanceId:         aws.String(instanceID),
		DeviceIndex:        aws.Int32(1),
	})
	if err != nil || attachInterfaceOut.AttachmentId == nil {
		t.Fatalf("attach network interface: %v", err)
	}
	attachmentID := aws.ToString(attachInterfaceOut.AttachmentId)
	if _, err := client.DescribeNetworkInterfaces(ctx, &awsec2.DescribeNetworkInterfacesInput{NetworkInterfaceIds: []string{interfaceID}}); err != nil {
		t.Fatalf("describe network interfaces: %v", err)
	}
	if _, err := client.DetachNetworkInterface(ctx, &awsec2.DetachNetworkInterfaceInput{AttachmentId: aws.String(attachmentID)}); err != nil {
		t.Fatalf("detach network interface: %v", err)
	}
	if _, err := client.DeleteNetworkInterface(ctx, &awsec2.DeleteNetworkInterfaceInput{NetworkInterfaceId: aws.String(interfaceID)}); err != nil {
		t.Fatalf("delete network interface: %v", err)
	}

	if _, err := client.DisassociateRouteTable(ctx, &awsec2.DisassociateRouteTableInput{AssociationId: aws.String(associationID)}); err != nil {
		t.Fatalf("disassociate route table: %v", err)
	}
	if _, err := client.DeleteRoute(ctx, &awsec2.DeleteRouteInput{RouteTableId: aws.String(routeTableID), DestinationCidrBlock: aws.String("0.0.0.0/0")}); err != nil {
		t.Fatalf("delete route: %v", err)
	}
	if _, err := client.DeleteRouteTable(ctx, &awsec2.DeleteRouteTableInput{RouteTableId: aws.String(routeTableID)}); err != nil {
		t.Fatalf("delete route table: %v", err)
	}
	if _, err := client.DetachInternetGateway(ctx, &awsec2.DetachInternetGatewayInput{InternetGatewayId: aws.String(igwID), VpcId: aws.String(vpcID)}); err != nil {
		t.Fatalf("detach internet gateway: %v", err)
	}
	if _, err := client.DeleteInternetGateway(ctx, &awsec2.DeleteInternetGatewayInput{InternetGatewayId: aws.String(igwID)}); err != nil {
		t.Fatalf("delete internet gateway: %v", err)
	}

	if _, err := client.TerminateInstances(ctx, &awsec2.TerminateInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		t.Fatalf("terminate instances: %v", err)
	}
	if _, err := client.DeleteSecurityGroup(ctx, &awsec2.DeleteSecurityGroupInput{GroupId: aws.String(sgID)}); err != nil {
		t.Fatalf("delete security group: %v", err)
	}
	if _, err := client.DeleteSubnet(ctx, &awsec2.DeleteSubnetInput{SubnetId: aws.String(subnetID)}); err != nil {
		t.Fatalf("delete subnet: %v", err)
	}
	if _, err := client.DeleteNetworkAcl(ctx, &awsec2.DeleteNetworkAclInput{NetworkAclId: aws.String(aclID)}); err != nil {
		t.Fatalf("delete network acl: %v", err)
	}
	if _, err := client.DeleteVpc(ctx, &awsec2.DeleteVpcInput{VpcId: aws.String(vpcID)}); err != nil {
		t.Fatalf("delete vpc: %v", err)
	}
}

func TestEC2Stage1ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateVpc",
		"DescribeVpcs",
		"DeleteVpc",
		"CreateSubnet",
		"DescribeSubnets",
		"DeleteSubnet",
		"CreateInternetGateway",
		"DescribeInternetGateways",
		"AttachInternetGateway",
		"DetachInternetGateway",
		"DeleteInternetGateway",
		"CreateRouteTable",
		"DescribeRouteTables",
		"AssociateRouteTable",
		"DisassociateRouteTable",
		"CreateRoute",
		"DeleteRoute",
		"DeleteRouteTable",
		"CreateNetworkAcl",
		"DescribeNetworkAcls",
		"DeleteNetworkAcl",
		"CreateNetworkInterface",
		"DescribeNetworkInterfaces",
		"AttachNetworkInterface",
		"DetachNetworkInterface",
		"DeleteNetworkInterface",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for idx, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateVpc":
			params["CidrBlock"] = "10.30.0.0/16"
		case "DescribeVpcs", "DeleteVpc":
			params["VpcId.1"] = "vpc-00000001"
			params["VpcId"] = "vpc-00000001"
		case "CreateSubnet":
			params["VpcId"] = "vpc-00000001"
			params["CidrBlock"] = "10.0.9.0/24"
		case "DescribeSubnets", "DeleteSubnet":
			params["SubnetId.1"] = "subnet-00000001"
			params["SubnetId"] = "subnet-00000001"
		case "DescribeInternetGateways", "DeleteInternetGateway":
			params["InternetGatewayId.1"] = "igw-00000001"
			params["InternetGatewayId"] = "igw-00000001"
		case "AttachInternetGateway", "DetachInternetGateway":
			params["InternetGatewayId"] = "igw-00000001"
			params["VpcId"] = "vpc-00000001"
		case "CreateRouteTable":
			params["VpcId"] = "vpc-00000001"
		case "DescribeRouteTables", "DeleteRouteTable":
			params["RouteTableId.1"] = "rtb-00000001"
			params["RouteTableId"] = "rtb-00000001"
		case "AssociateRouteTable":
			params["RouteTableId"] = "rtb-00000001"
			params["SubnetId"] = "subnet-00000001"
		case "DisassociateRouteTable":
			params["AssociationId"] = "rtbassoc-00000001"
		case "CreateRoute", "DeleteRoute":
			params["RouteTableId"] = "rtb-00000001"
			params["DestinationCidrBlock"] = "0.0.0.0/0"
			params["GatewayId"] = "igw-00000001"
		case "CreateNetworkAcl":
			params["VpcId"] = "vpc-00000001"
		case "DescribeNetworkAcls", "DeleteNetworkAcl":
			params["NetworkAclId.1"] = "acl-00000001"
			params["NetworkAclId"] = "acl-00000001"
		case "CreateNetworkInterface":
			params["SubnetId"] = "subnet-00000001"
			params["GroupId.1"] = "sg-00000000"
		case "DescribeNetworkInterfaces", "DeleteNetworkInterface":
			params["NetworkInterfaceId.1"] = "eni-00000001"
			params["NetworkInterfaceId"] = "eni-00000001"
		case "AttachNetworkInterface":
			params["NetworkInterfaceId"] = "eni-00000001"
			params["InstanceId"] = "i-00000001"
			params["DeviceIndex"] = "1"
		case "DetachNetworkInterface":
			params["AttachmentId"] = "eniattach-00000001"
		}
		if action == "CreateNetworkInterface" {
			params["Description"] = "eni-" + strconv.Itoa(idx)
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
