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

func TestEC2Stage7SDKLifecycle(t *testing.T) {
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

	allocateOut1, err := client.AllocateAddress(ctx, &awsec2.AllocateAddressInput{Domain: awsec2types.DomainTypeVpc})
	if err != nil || allocateOut1.AllocationId == nil {
		t.Fatalf("allocate address #1: %v", err)
	}
	allocationID1 := aws.ToString(allocateOut1.AllocationId)

	createNatOut, err := client.CreateNatGateway(ctx, &awsec2.CreateNatGatewayInput{
		SubnetId:     aws.String("subnet-00000001"),
		AllocationId: aws.String(allocationID1),
	})
	if err != nil || createNatOut.NatGateway == nil || createNatOut.NatGateway.NatGatewayId == nil {
		t.Fatalf("create nat gateway: %v", err)
	}
	natGatewayID := aws.ToString(createNatOut.NatGateway.NatGatewayId)

	assignOut, err := client.AssignPrivateNatGatewayAddress(ctx, &awsec2.AssignPrivateNatGatewayAddressInput{
		NatGatewayId:          aws.String(natGatewayID),
		PrivateIpAddressCount: aws.Int32(1),
	})
	if err != nil || len(assignOut.NatGatewayAddresses) != 1 {
		t.Fatalf("assign private nat gateway address: %v", err)
	}
	assignedPrivateIP := aws.ToString(assignOut.NatGatewayAddresses[0].PrivateIp)

	allocateOut2, err := client.AllocateAddress(ctx, &awsec2.AllocateAddressInput{Domain: awsec2types.DomainTypeVpc})
	if err != nil || allocateOut2.AllocationId == nil {
		t.Fatalf("allocate address #2: %v", err)
	}
	allocationID2 := aws.ToString(allocateOut2.AllocationId)

	associateOut, err := client.AssociateNatGatewayAddress(ctx, &awsec2.AssociateNatGatewayAddressInput{
		NatGatewayId:  aws.String(natGatewayID),
		AllocationIds: []string{allocationID2},
	})
	if err != nil || len(associateOut.NatGatewayAddresses) != 1 || associateOut.NatGatewayAddresses[0].AssociationId == nil {
		t.Fatalf("associate nat gateway address: %v", err)
	}
	associationID := aws.ToString(associateOut.NatGatewayAddresses[0].AssociationId)

	if _, err := client.DisassociateNatGatewayAddress(ctx, &awsec2.DisassociateNatGatewayAddressInput{
		NatGatewayId:   aws.String(natGatewayID),
		AssociationIds: []string{associationID},
	}); err != nil {
		t.Fatalf("disassociate nat gateway address: %v", err)
	}

	if _, err := client.UnassignPrivateNatGatewayAddress(ctx, &awsec2.UnassignPrivateNatGatewayAddressInput{
		NatGatewayId:       aws.String(natGatewayID),
		PrivateIpAddresses: []string{assignedPrivateIP},
	}); err != nil {
		t.Fatalf("unassign private nat gateway address: %v", err)
	}

	createVpcOut1, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{CidrBlock: aws.String("10.91.0.0/16")})
	if err != nil || createVpcOut1.Vpc == nil || createVpcOut1.Vpc.VpcId == nil {
		t.Fatalf("create vpc #1: %v", err)
	}
	vpcID1 := aws.ToString(createVpcOut1.Vpc.VpcId)

	createVpcOut2, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{CidrBlock: aws.String("10.92.0.0/16")})
	if err != nil || createVpcOut2.Vpc == nil || createVpcOut2.Vpc.VpcId == nil {
		t.Fatalf("create vpc #2: %v", err)
	}
	vpcID2 := aws.ToString(createVpcOut2.Vpc.VpcId)

	createPcxOut, err := client.CreateVpcPeeringConnection(ctx, &awsec2.CreateVpcPeeringConnectionInput{
		VpcId:     aws.String(vpcID1),
		PeerVpcId: aws.String(vpcID2),
	})
	if err != nil || createPcxOut.VpcPeeringConnection == nil || createPcxOut.VpcPeeringConnection.VpcPeeringConnectionId == nil {
		t.Fatalf("create vpc peering connection: %v", err)
	}
	pcxID := aws.ToString(createPcxOut.VpcPeeringConnection.VpcPeeringConnectionId)

	modifyPcxOut, err := client.ModifyVpcPeeringConnectionOptions(ctx, &awsec2.ModifyVpcPeeringConnectionOptionsInput{
		VpcPeeringConnectionId: aws.String(pcxID),
		RequesterPeeringConnectionOptions: &awsec2types.PeeringConnectionOptionsRequest{
			AllowDnsResolutionFromRemoteVpc: aws.Bool(true),
		},
		AccepterPeeringConnectionOptions: &awsec2types.PeeringConnectionOptionsRequest{
			AllowDnsResolutionFromRemoteVpc: aws.Bool(true),
		},
	})
	if err != nil {
		t.Fatalf("modify vpc peering options: %v", err)
	}
	if modifyPcxOut.RequesterPeeringConnectionOptions == nil || modifyPcxOut.RequesterPeeringConnectionOptions.AllowDnsResolutionFromRemoteVpc == nil || !aws.ToBool(modifyPcxOut.RequesterPeeringConnectionOptions.AllowDnsResolutionFromRemoteVpc) {
		t.Fatalf("expected requester dns resolution option true")
	}

	createSubnetOut, err := client.CreateSubnet(ctx, &awsec2.CreateSubnetInput{VpcId: aws.String(vpcID1), CidrBlock: aws.String("10.91.1.0/24")})
	if err != nil || createSubnetOut.Subnet == nil || createSubnetOut.Subnet.SubnetId == nil {
		t.Fatalf("create subnet: %v", err)
	}
	subnetID := aws.ToString(createSubnetOut.Subnet.SubnetId)

	describeAclOut, err := client.DescribeNetworkAcls(ctx, &awsec2.DescribeNetworkAclsInput{
		Filters: []awsec2types.Filter{{Name: aws.String("vpc-id"), Values: []string{vpcID1}}},
	})
	if err != nil {
		t.Fatalf("describe network acls: %v", err)
	}
	associationIDToReplace := ""
	for _, acl := range describeAclOut.NetworkAcls {
		for _, assoc := range acl.Associations {
			if assoc.SubnetId != nil && aws.ToString(assoc.SubnetId) == subnetID && assoc.NetworkAclAssociationId != nil {
				associationIDToReplace = aws.ToString(assoc.NetworkAclAssociationId)
				break
			}
		}
		if associationIDToReplace != "" {
			break
		}
	}
	if associationIDToReplace == "" {
		t.Fatalf("expected network acl association for subnet")
	}

	createAclOut, err := client.CreateNetworkAcl(ctx, &awsec2.CreateNetworkAclInput{VpcId: aws.String(vpcID1)})
	if err != nil || createAclOut.NetworkAcl == nil || createAclOut.NetworkAcl.NetworkAclId == nil {
		t.Fatalf("create network acl: %v", err)
	}
	aclID := aws.ToString(createAclOut.NetworkAcl.NetworkAclId)

	replaceAclAssocOut, err := client.ReplaceNetworkAclAssociation(ctx, &awsec2.ReplaceNetworkAclAssociationInput{
		AssociationId: aws.String(associationIDToReplace),
		NetworkAclId:  aws.String(aclID),
	})
	if err != nil || replaceAclAssocOut.NewAssociationId == nil || aws.ToString(replaceAclAssocOut.NewAssociationId) == "" {
		t.Fatalf("replace network acl association: %v", err)
	}

	createIfaceOut, err := client.CreateNetworkInterface(ctx, &awsec2.CreateNetworkInterfaceInput{SubnetId: aws.String("subnet-00000001")})
	if err != nil || createIfaceOut.NetworkInterface == nil || createIfaceOut.NetworkInterface.NetworkInterfaceId == nil {
		t.Fatalf("create network interface: %v", err)
	}
	ifaceID := aws.ToString(createIfaceOut.NetworkInterface.NetworkInterfaceId)

	createPermOut, err := client.CreateNetworkInterfacePermission(ctx, &awsec2.CreateNetworkInterfacePermissionInput{
		NetworkInterfaceId: aws.String(ifaceID),
		Permission:         awsec2types.InterfacePermissionTypeInstanceAttach,
		AwsAccountId:       aws.String("123456789012"),
	})
	if err != nil || createPermOut.InterfacePermission == nil || createPermOut.InterfacePermission.NetworkInterfacePermissionId == nil {
		t.Fatalf("create network interface permission: %v", err)
	}
	permissionID := aws.ToString(createPermOut.InterfacePermission.NetworkInterfacePermissionId)

	describePermsOut, err := client.DescribeNetworkInterfacePermissions(ctx, &awsec2.DescribeNetworkInterfacePermissionsInput{
		NetworkInterfacePermissionIds: []string{permissionID},
	})
	if err != nil || len(describePermsOut.NetworkInterfacePermissions) != 1 {
		t.Fatalf("describe network interface permissions: %v", err)
	}

	if _, err := client.ModifyNetworkInterfaceAttribute(ctx, &awsec2.ModifyNetworkInterfaceAttributeInput{
		NetworkInterfaceId: aws.String(ifaceID),
		SourceDestCheck:    &awsec2types.AttributeBooleanValue{Value: aws.Bool(false)},
	}); err != nil {
		t.Fatalf("modify network interface attribute: %v", err)
	}

	describeIfaceAttrOut, err := client.DescribeNetworkInterfaceAttribute(ctx, &awsec2.DescribeNetworkInterfaceAttributeInput{
		NetworkInterfaceId: aws.String(ifaceID),
		Attribute:          awsec2types.NetworkInterfaceAttributeSourceDestCheck,
	})
	if err != nil || describeIfaceAttrOut.SourceDestCheck == nil || aws.ToBool(describeIfaceAttrOut.SourceDestCheck.Value) {
		t.Fatalf("describe network interface attribute before reset: %v", err)
	}

	if _, err := client.ResetNetworkInterfaceAttribute(ctx, &awsec2.ResetNetworkInterfaceAttributeInput{
		NetworkInterfaceId: aws.String(ifaceID),
		SourceDestCheck:    aws.String("sourceDestCheck"),
	}); err != nil {
		t.Fatalf("reset network interface attribute: %v", err)
	}

	describeIfaceAttrOutAfterReset, err := client.DescribeNetworkInterfaceAttribute(ctx, &awsec2.DescribeNetworkInterfaceAttributeInput{
		NetworkInterfaceId: aws.String(ifaceID),
		Attribute:          awsec2types.NetworkInterfaceAttributeSourceDestCheck,
	})
	if err != nil || describeIfaceAttrOutAfterReset.SourceDestCheck == nil || !aws.ToBool(describeIfaceAttrOutAfterReset.SourceDestCheck.Value) {
		t.Fatalf("describe network interface attribute after reset: %v", err)
	}

	if _, err := client.DeleteNetworkInterfacePermission(ctx, &awsec2.DeleteNetworkInterfacePermissionInput{NetworkInterfacePermissionId: aws.String(permissionID)}); err != nil {
		t.Fatalf("delete network interface permission: %v", err)
	}
}

func TestEC2Stage7ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AssignPrivateNatGatewayAddress",
		"AssociateNatGatewayAddress",
		"DisassociateNatGatewayAddress",
		"UnassignPrivateNatGatewayAddress",
		"ModifyVpcPeeringConnectionOptions",
		"ReplaceNetworkAclAssociation",
		"CreateNetworkInterfacePermission",
		"DeleteNetworkInterfacePermission",
		"DescribeNetworkInterfaceAttribute",
		"DescribeNetworkInterfacePermissions",
		"ResetNetworkInterfaceAttribute",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for idx, action := range implemented {
		params := map[string]string{}
		switch action {
		case "AssignPrivateNatGatewayAddress":
			params["NatGatewayId"] = "nat-00000001"
			params["PrivateIpAddressCount"] = "1"
		case "AssociateNatGatewayAddress":
			params["NatGatewayId"] = "nat-00000001"
			params["AllocationId.1"] = "eipalloc-00000001"
		case "DisassociateNatGatewayAddress":
			params["NatGatewayId"] = "nat-00000001"
			params["AssociationId.1"] = "eipassoc-00000001"
		case "UnassignPrivateNatGatewayAddress":
			params["NatGatewayId"] = "nat-00000001"
			params["PrivateIpAddress.1"] = "10.0.1.10"
		case "ModifyVpcPeeringConnectionOptions":
			params["VpcPeeringConnectionId"] = "pcx-00000001"
			params["RequesterPeeringConnectionOptions.AllowDnsResolutionFromRemoteVpc"] = "true"
		case "ReplaceNetworkAclAssociation":
			params["AssociationId"] = "aclassoc-00000001"
			params["NetworkAclId"] = "acl-00000001"
		case "CreateNetworkInterfacePermission":
			params["NetworkInterfaceId"] = "eni-00000001"
			params["Permission"] = "INSTANCE-ATTACH"
			params["AwsAccountId"] = "123456789012"
		case "DeleteNetworkInterfacePermission":
			params["NetworkInterfacePermissionId"] = "eni-perm-0000000" + strconv.Itoa(idx+1)
		case "DescribeNetworkInterfaceAttribute":
			params["NetworkInterfaceId"] = "eni-00000001"
			params["Attribute"] = "sourceDestCheck"
		case "DescribeNetworkInterfacePermissions":
			params["NetworkInterfacePermissionId.1"] = "eni-perm-00000001"
		case "ResetNetworkInterfaceAttribute":
			params["NetworkInterfaceId"] = "eni-00000001"
			params["SourceDestCheck"] = "sourceDestCheck"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
