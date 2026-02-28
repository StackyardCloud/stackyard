package ec2

import (
	"strings"
	"testing"
)

func TestServiceLifecycle(t *testing.T) {
	svc := NewService()

	res, err := svc.RunInstances("ami-test", "t3.micro", "", "", "", nil, 1, 1, []Tag{{Key: "env", Value: "test"}})
	if err != nil {
		t.Fatalf("run instances: %v", err)
	}
	if len(res.Instances) != 1 {
		t.Fatalf("expected 1 instance")
	}
	instanceID := res.Instances[0].ID

	described := svc.DescribeInstances([]string{instanceID})
	if len(described) != 1 {
		t.Fatalf("expected describe instances to return one reservation")
	}

	if _, err := svc.StopInstances([]string{instanceID}); err != nil {
		t.Fatalf("stop instances: %v", err)
	}
	if _, err := svc.StartInstances([]string{instanceID}); err != nil {
		t.Fatalf("start instances: %v", err)
	}
	if err := svc.RebootInstances([]string{instanceID}); err != nil {
		t.Fatalf("reboot instances: %v", err)
	}

	if err := svc.CreateTags([]string{instanceID}, []Tag{{Key: "team", Value: "platform"}}); err != nil {
		t.Fatalf("create tags: %v", err)
	}
	tags := svc.DescribeTags([]string{instanceID})
	if len(tags) == 0 {
		t.Fatalf("expected tags")
	}
	if err := svc.DeleteTags([]string{instanceID}, []Tag{{Key: "team"}}); err != nil {
		t.Fatalf("delete tags: %v", err)
	}

	sg, err := svc.CreateSecurityGroup("sg-app", "app group", "", nil)
	if err != nil {
		t.Fatalf("create security group: %v", err)
	}
	if err := svc.AuthorizeSecurityGroupIngress(sg.ID, "", "", []IPPermission{{Protocol: "tcp", FromPort: 80, ToPort: 80, CidrIP: "0.0.0.0/0"}}); err != nil {
		t.Fatalf("authorize ingress: %v", err)
	}
	if err := svc.RevokeSecurityGroupIngress(sg.ID, "", "", []IPPermission{{Protocol: "tcp", FromPort: 80, ToPort: 80, CidrIP: "0.0.0.0/0"}}); err != nil {
		t.Fatalf("revoke ingress: %v", err)
	}

	vol, err := svc.CreateVolume(10, "us-east-1a", "gp3", "", []Tag{{Key: "name", Value: "data"}})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	if _, err := svc.AttachVolume(vol.ID, instanceID, "/dev/xvdf"); err != nil {
		t.Fatalf("attach volume: %v", err)
	}
	if _, err := svc.DetachVolume(vol.ID, instanceID, "/dev/xvdf", false); err != nil {
		t.Fatalf("detach volume: %v", err)
	}
	snap, err := svc.CreateSnapshot(vol.ID, "snapshot", nil)
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if err := svc.DeleteSnapshot(snap.ID); err != nil {
		t.Fatalf("delete snapshot: %v", err)
	}
	if err := svc.DeleteVolume(vol.ID); err != nil {
		t.Fatalf("delete volume: %v", err)
	}

	if _, err := svc.TerminateInstances([]string{instanceID}); err != nil {
		t.Fatalf("terminate instances: %v", err)
	}
	if err := svc.DeleteSecurityGroup(sg.ID, "", ""); err != nil {
		t.Fatalf("delete security group: %v", err)
	}
}

func TestServiceStage1NetworkingLifecycle(t *testing.T) {
	svc := NewService()

	vpc, err := svc.CreateVpc("10.2.0.0/16", []Tag{{Key: "env", Value: "test"}})
	if err != nil {
		t.Fatalf("create vpc: %v", err)
	}

	subnet, err := svc.CreateSubnet(vpc.ID, "10.2.1.0/24", "us-east-1a", nil)
	if err != nil {
		t.Fatalf("create subnet: %v", err)
	}

	igw, err := svc.CreateInternetGateway(nil)
	if err != nil {
		t.Fatalf("create internet gateway: %v", err)
	}
	if err := svc.AttachInternetGateway(igw.ID, vpc.ID); err != nil {
		t.Fatalf("attach internet gateway: %v", err)
	}

	rtb, err := svc.CreateRouteTable(vpc.ID, nil)
	if err != nil {
		t.Fatalf("create route table: %v", err)
	}
	if err := svc.CreateRoute(rtb.ID, "0.0.0.0/0", igw.ID); err != nil {
		t.Fatalf("create route: %v", err)
	}
	assoc, err := svc.AssociateRouteTable(rtb.ID, subnet.ID)
	if err != nil {
		t.Fatalf("associate route table: %v", err)
	}

	acl, err := svc.CreateNetworkACL(vpc.ID, nil)
	if err != nil {
		t.Fatalf("create network acl: %v", err)
	}

	sg, err := svc.CreateSecurityGroup("stage1-sg", "stage1 sg", vpc.ID, nil)
	if err != nil {
		t.Fatalf("create security group: %v", err)
	}

	res, err := svc.RunInstances("ami-test", "t3.micro", "", subnet.ID, "", []string{sg.ID}, 1, 1, nil)
	if err != nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := res.Instances[0].ID

	iface, err := svc.CreateNetworkInterface(subnet.ID, "secondary", "10.2.1.50", []string{sg.ID}, nil)
	if err != nil {
		t.Fatalf("create network interface: %v", err)
	}
	attachment, err := svc.AttachNetworkInterface(iface.ID, instanceID, 1)
	if err != nil {
		t.Fatalf("attach network interface: %v", err)
	}
	if err := svc.DetachNetworkInterface(attachment.ID, false); err != nil {
		t.Fatalf("detach network interface: %v", err)
	}
	if err := svc.DeleteNetworkInterface(iface.ID); err != nil {
		t.Fatalf("delete network interface: %v", err)
	}

	if err := svc.DisassociateRouteTable(assoc.ID); err != nil {
		t.Fatalf("disassociate route table: %v", err)
	}
	if err := svc.DeleteRoute(rtb.ID, "0.0.0.0/0"); err != nil {
		t.Fatalf("delete route: %v", err)
	}
	if err := svc.DeleteRouteTable(rtb.ID); err != nil {
		t.Fatalf("delete route table: %v", err)
	}

	if err := svc.DetachInternetGateway(igw.ID, vpc.ID); err != nil {
		t.Fatalf("detach internet gateway: %v", err)
	}
	if err := svc.DeleteInternetGateway(igw.ID); err != nil {
		t.Fatalf("delete internet gateway: %v", err)
	}

	if _, err := svc.TerminateInstances([]string{instanceID}); err != nil {
		t.Fatalf("terminate instance: %v", err)
	}
	if err := svc.DeleteSecurityGroup(sg.ID, "", ""); err != nil {
		t.Fatalf("delete security group: %v", err)
	}
	if err := svc.DeleteSubnet(subnet.ID); err != nil {
		t.Fatalf("delete subnet: %v", err)
	}
	if err := svc.DeleteNetworkACL(acl.ID); err != nil {
		t.Fatalf("delete network acl: %v", err)
	}
	if err := svc.DeleteVpc(vpc.ID); err != nil {
		t.Fatalf("delete vpc: %v", err)
	}
}

func TestServiceStage2SecurityAndIdentityLifecycle(t *testing.T) {
	svc := NewService()

	sg, err := svc.CreateSecurityGroup("stage2-sg", "stage2", "vpc-00000001", nil)
	if err != nil {
		t.Fatalf("create security group: %v", err)
	}
	if err := svc.AuthorizeSecurityGroupEgress(sg.ID, "", "", []IPPermission{{Protocol: "tcp", FromPort: 443, ToPort: 443, CidrIP: "0.0.0.0/0"}}); err != nil {
		t.Fatalf("authorize egress: %v", err)
	}
	if err := svc.RevokeSecurityGroupEgress(sg.ID, "", "", []IPPermission{{Protocol: "tcp", FromPort: 443, ToPort: 443, CidrIP: "0.0.0.0/0"}}); err != nil {
		t.Fatalf("revoke egress: %v", err)
	}

	keyPair, err := svc.CreateKeyPair("stage2-key")
	if err != nil {
		t.Fatalf("create key pair: %v", err)
	}
	if keyPair.Name != "stage2-key" {
		t.Fatalf("unexpected key pair name: %s", keyPair.Name)
	}
	if _, err := svc.ImportKeyPair("stage2-import", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDcstage2 test@example.com"); err != nil {
		t.Fatalf("import key pair: %v", err)
	}
	if got := svc.DescribeKeyPairs([]string{"stage2-key"}, nil); len(got) != 1 {
		t.Fatalf("expected one described key pair, got %d", len(got))
	}

	res, err := svc.RunInstances("ami-test", "t3.micro", "stage2-key", "subnet-00000001", "", []string{sg.ID}, 1, 1, nil)
	if err != nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := res.Instances[0].ID

	association, err := svc.AssociateIamInstanceProfile(instanceID, "stage2-profile", "")
	if err != nil {
		t.Fatalf("associate iam profile: %v", err)
	}
	if association.State != "associated" {
		t.Fatalf("expected associated state")
	}

	if got := svc.DescribeIamInstanceProfileAssociations([]string{association.AssociationID}, nil); len(got) != 1 {
		t.Fatalf("expected one profile association, got %d", len(got))
	}

	replaced, err := svc.ReplaceIamInstanceProfileAssociation(association.AssociationID, "stage2-profile-replaced", "")
	if err != nil {
		t.Fatalf("replace iam profile association: %v", err)
	}
	if replaced.ProfileName != "stage2-profile-replaced" {
		t.Fatalf("unexpected replaced profile name: %s", replaced.ProfileName)
	}

	disassociated, err := svc.DisassociateIamInstanceProfile(association.AssociationID)
	if err != nil {
		t.Fatalf("disassociate iam profile association: %v", err)
	}
	if disassociated.State != "disassociated" {
		t.Fatalf("expected disassociated state")
	}

	if _, err := svc.TerminateInstances([]string{instanceID}); err != nil {
		t.Fatalf("terminate instance: %v", err)
	}
	if err := svc.DeleteKeyPair("stage2-key", ""); err != nil {
		t.Fatalf("delete key pair: %v", err)
	}
	if err := svc.DeleteKeyPair("stage2-import", ""); err != nil {
		t.Fatalf("delete imported key pair: %v", err)
	}
	if err := svc.DeleteSecurityGroup(sg.ID, "", ""); err != nil {
		t.Fatalf("delete security group: %v", err)
	}
}

func TestServiceStage3NetworkAndAttributeLifecycle(t *testing.T) {
	svc := NewService()

	vpc, err := svc.CreateVpc("10.50.0.0/16", nil)
	if err != nil {
		t.Fatalf("create vpc: %v", err)
	}
	subnet, err := svc.CreateSubnet(vpc.ID, "10.50.1.0/24", "us-east-1a", nil)
	if err != nil {
		t.Fatalf("create subnet: %v", err)
	}

	igw1, err := svc.CreateInternetGateway(nil)
	if err != nil {
		t.Fatalf("create internet gateway #1: %v", err)
	}
	if err := svc.AttachInternetGateway(igw1.ID, vpc.ID); err != nil {
		t.Fatalf("attach internet gateway #1: %v", err)
	}
	igw2, err := svc.CreateInternetGateway(nil)
	if err != nil {
		t.Fatalf("create internet gateway #2: %v", err)
	}
	if err := svc.AttachInternetGateway(igw2.ID, vpc.ID); err != nil {
		t.Fatalf("attach internet gateway #2: %v", err)
	}

	rtb1, err := svc.CreateRouteTable(vpc.ID, nil)
	if err != nil {
		t.Fatalf("create route table #1: %v", err)
	}
	rtb2, err := svc.CreateRouteTable(vpc.ID, nil)
	if err != nil {
		t.Fatalf("create route table #2: %v", err)
	}
	if err := svc.CreateRoute(rtb1.ID, "0.0.0.0/0", igw1.ID); err != nil {
		t.Fatalf("create route: %v", err)
	}
	if err := svc.ReplaceRoute(rtb1.ID, "0.0.0.0/0", igw2.ID); err != nil {
		t.Fatalf("replace route: %v", err)
	}
	assoc, err := svc.AssociateRouteTable(rtb1.ID, subnet.ID)
	if err != nil {
		t.Fatalf("associate route table: %v", err)
	}
	replacedAssoc, err := svc.ReplaceRouteTableAssociation(assoc.ID, rtb2.ID)
	if err != nil {
		t.Fatalf("replace route table association: %v", err)
	}
	if replacedAssoc.ID == "" || replacedAssoc.SubnetID != subnet.ID {
		t.Fatalf("unexpected replacement association: %+v", replacedAssoc)
	}

	acl, err := svc.CreateNetworkACL(vpc.ID, nil)
	if err != nil {
		t.Fatalf("create network acl: %v", err)
	}
	if err := svc.CreateNetworkACLEntry(acl.ID, 110, "6", "allow", false, "0.0.0.0/0"); err != nil {
		t.Fatalf("create network acl entry: %v", err)
	}
	if err := svc.ReplaceNetworkACLEntry(acl.ID, 110, "6", "deny", false, "0.0.0.0/0"); err != nil {
		t.Fatalf("replace network acl entry: %v", err)
	}
	if err := svc.DeleteNetworkACLEntry(acl.ID, 110, false); err != nil {
		t.Fatalf("delete network acl entry: %v", err)
	}

	sg1, err := svc.CreateSecurityGroup("stage3-sg-1", "stage3 sg 1", vpc.ID, nil)
	if err != nil {
		t.Fatalf("create security group #1: %v", err)
	}
	sg2, err := svc.CreateSecurityGroup("stage3-sg-2", "stage3 sg 2", vpc.ID, nil)
	if err != nil {
		t.Fatalf("create security group #2: %v", err)
	}
	iface, err := svc.CreateNetworkInterface(subnet.ID, "stage3", "10.50.1.44", []string{sg1.ID}, nil)
	if err != nil {
		t.Fatalf("create network interface: %v", err)
	}
	desc := "stage3-updated"
	sourceDestCheck := false
	if err := svc.ModifyNetworkInterfaceAttribute(iface.ID, &desc, &sourceDestCheck, []string{sg2.ID}); err != nil {
		t.Fatalf("modify network interface attribute: %v", err)
	}
	ifaces := svc.DescribeNetworkInterfaces([]string{iface.ID}, "", "")
	if len(ifaces) != 1 {
		t.Fatalf("expected one network interface")
	}
	if ifaces[0].Description != desc || ifaces[0].SourceDestCheck || len(ifaces[0].GroupIDs) != 1 || ifaces[0].GroupIDs[0] != sg2.ID {
		t.Fatalf("unexpected network interface attributes: %+v", ifaces[0])
	}

	mapPublicIPOnLaunch := false
	if err := svc.ModifySubnetAttribute(subnet.ID, &mapPublicIPOnLaunch); err != nil {
		t.Fatalf("modify subnet attribute: %v", err)
	}
	subnets := svc.DescribeSubnets([]string{subnet.ID}, nil)
	if len(subnets) != 1 || subnets[0].MapPublicIPOnLaunch {
		t.Fatalf("expected map public ip on launch to be false")
	}

	enableDnsSupport := false
	enableDnsHostnames := true
	if err := svc.ModifyVpcAttribute(vpc.ID, &enableDnsSupport, nil); err != nil {
		t.Fatalf("modify vpc dns support: %v", err)
	}
	if err := svc.ModifyVpcAttribute(vpc.ID, nil, &enableDnsHostnames); err != nil {
		t.Fatalf("modify vpc dns hostnames: %v", err)
	}
	vpcSupport, err := svc.DescribeVpcAttribute(vpc.ID, "enableDnsSupport")
	if err != nil || vpcSupport.Value {
		t.Fatalf("describe vpc dns support: %v", err)
	}
	vpcHostnames, err := svc.DescribeVpcAttribute(vpc.ID, "enableDnsHostnames")
	if err != nil || !vpcHostnames.Value {
		t.Fatalf("describe vpc dns hostnames: %v", err)
	}

	accountAttrs := svc.DescribeAccountAttributes([]string{"default-vpc"})
	if len(accountAttrs) != 1 || accountAttrs[0].Name != "default-vpc" || len(accountAttrs[0].Values) != 1 {
		t.Fatalf("unexpected account attrs: %+v", accountAttrs)
	}
}

func TestServiceStage4AddressingAndDefaultsLifecycle(t *testing.T) {
	svc := NewService()

	addr, err := svc.AllocateAddress("", []Tag{{Key: "env", Value: "stage4"}})
	if err != nil {
		t.Fatalf("allocate address: %v", err)
	}
	if addr.AllocationID == "" || addr.PublicIP == "" {
		t.Fatalf("expected allocation id and public ip")
	}

	res, err := svc.RunInstances("ami-stage4", "t3.micro", "", "subnet-00000001", "", nil, 1, 1, nil)
	if err != nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := res.Instances[0].ID

	associationID, err := svc.AssociateAddress(addr.AllocationID, "", instanceID, "", "", true)
	if err != nil {
		t.Fatalf("associate address: %v", err)
	}
	if associationID == "" {
		t.Fatalf("expected association id")
	}

	addresses := svc.DescribeAddresses([]string{addr.AllocationID}, nil, nil, nil)
	if len(addresses) != 1 || addresses[0].AssociationID == "" || addresses[0].InstanceID != instanceID {
		t.Fatalf("unexpected described addresses: %+v", addresses)
	}

	if err := svc.DisassociateAddress(associationID, ""); err != nil {
		t.Fatalf("disassociate address: %v", err)
	}
	if err := svc.ReleaseAddress(addr.AllocationID, ""); err != nil {
		t.Fatalf("release address: %v", err)
	}

	vpc, err := svc.CreateDefaultVpc()
	if err != nil {
		t.Fatalf("create default vpc: %v", err)
	}
	if !vpc.IsDefault {
		t.Fatalf("expected default vpc")
	}

	defaultSubnet, err := svc.CreateDefaultSubnet("us-east-1b", "")
	if err != nil {
		t.Fatalf("create default subnet: %v", err)
	}
	if defaultSubnet.VpcID != vpc.ID {
		t.Fatalf("expected default subnet in default vpc")
	}
}

func TestServiceStage5DhcpAndEgressLifecycle(t *testing.T) {
	svc := NewService()

	dhcp, err := svc.CreateDhcpOptions([]DHCPConfiguration{
		{Key: "domain-name-servers", Values: []string{"AmazonProvidedDNS"}},
		{Key: "domain-name", Values: []string{"example.internal"}},
	}, []Tag{{Key: "env", Value: "stage5"}})
	if err != nil {
		t.Fatalf("create dhcp options: %v", err)
	}
	if dhcp.ID == "" {
		t.Fatalf("expected dhcp options id")
	}

	if err := svc.AssociateDhcpOptions(dhcp.ID, defaultVPCID); err != nil {
		t.Fatalf("associate dhcp options: %v", err)
	}
	describedDhcp := svc.DescribeDhcpOptions([]string{dhcp.ID}, nil)
	if len(describedDhcp) != 1 {
		t.Fatalf("expected one dhcp options set")
	}

	gateway, err := svc.CreateEgressOnlyInternetGateway(defaultVPCID, []Tag{{Key: "name", Value: "stage5-egress"}})
	if err != nil {
		t.Fatalf("create egress-only internet gateway: %v", err)
	}
	if gateway.ID == "" {
		t.Fatalf("expected egress gateway id")
	}
	describedGateways := svc.DescribeEgressOnlyInternetGateways([]string{gateway.ID}, nil)
	if len(describedGateways) != 1 {
		t.Fatalf("expected one egress gateway")
	}

	address, err := svc.AllocateAddress("", nil)
	if err != nil {
		t.Fatalf("allocate address: %v", err)
	}
	addressAttrs, err := svc.DescribeAddressesAttribute([]string{address.AllocationID}, "domain-name")
	if err != nil {
		t.Fatalf("describe addresses attribute: %v", err)
	}
	if len(addressAttrs) != 1 || addressAttrs[0].PtrRecord == "" {
		t.Fatalf("expected one address attribute with ptr record")
	}
	if err := svc.ReleaseAddress(address.AllocationID, ""); err != nil {
		t.Fatalf("release address: %v", err)
	}

	if err := svc.DeleteEgressOnlyInternetGateway(gateway.ID); err != nil {
		t.Fatalf("delete egress gateway: %v", err)
	}
	if err := svc.AssociateDhcpOptions("default", defaultVPCID); err != nil {
		t.Fatalf("associate default dhcp options: %v", err)
	}
	if err := svc.DeleteDhcpOptions(dhcp.ID); err != nil {
		t.Fatalf("delete dhcp options: %v", err)
	}
}

func TestServiceStage6NatAndPeeringLifecycle(t *testing.T) {
	svc := NewService()

	address, err := svc.AllocateAddress("", nil)
	if err != nil {
		t.Fatalf("allocate address: %v", err)
	}

	nat, err := svc.CreateNatGateway(defaultSubnetID, address.AllocationID, "public", []Tag{{Key: "env", Value: "stage6"}})
	if err != nil {
		t.Fatalf("create nat gateway: %v", err)
	}
	if nat.ID == "" {
		t.Fatalf("expected nat gateway id")
	}
	describedNat := svc.DescribeNatGateways([]string{nat.ID}, nil, nil)
	if len(describedNat) != 1 {
		t.Fatalf("expected one nat gateway")
	}
	if err := svc.DeleteNatGateway(nat.ID); err != nil {
		t.Fatalf("delete nat gateway: %v", err)
	}
	if err := svc.ReleaseAddress(address.AllocationID, ""); err != nil {
		t.Fatalf("release address: %v", err)
	}

	vpcA, err := svc.CreateVpc("10.71.0.0/16", nil)
	if err != nil {
		t.Fatalf("create vpc A: %v", err)
	}
	vpcB, err := svc.CreateVpc("10.72.0.0/16", nil)
	if err != nil {
		t.Fatalf("create vpc B: %v", err)
	}

	connection, err := svc.CreateVpcPeeringConnection(vpcA.ID, vpcB.ID, []Tag{{Key: "name", Value: "stage6"}})
	if err != nil {
		t.Fatalf("create vpc peering connection: %v", err)
	}
	if connection.ID == "" || connection.StatusCode != "pending-acceptance" {
		t.Fatalf("unexpected vpc peering connection: %+v", connection)
	}
	describedConnections := svc.DescribeVpcPeeringConnections([]string{connection.ID}, nil, nil, []string{"pending-acceptance"})
	if len(describedConnections) != 1 {
		t.Fatalf("expected one pending vpc peering connection")
	}
	accepted, err := svc.AcceptVpcPeeringConnection(connection.ID)
	if err != nil {
		t.Fatalf("accept vpc peering connection: %v", err)
	}
	if accepted.StatusCode != "active" {
		t.Fatalf("expected active status, got %s", accepted.StatusCode)
	}
	deleted, err := svc.DeleteVpcPeeringConnection(connection.ID)
	if err != nil {
		t.Fatalf("delete vpc peering connection: %v", err)
	}
	if deleted.StatusCode != "deleted" {
		t.Fatalf("expected deleted status, got %s", deleted.StatusCode)
	}

	connection2, err := svc.CreateVpcPeeringConnection(vpcA.ID, vpcB.ID, nil)
	if err != nil {
		t.Fatalf("create second vpc peering connection: %v", err)
	}
	rejected, err := svc.RejectVpcPeeringConnection(connection2.ID)
	if err != nil {
		t.Fatalf("reject vpc peering connection: %v", err)
	}
	if rejected.StatusCode != "rejected" {
		t.Fatalf("expected rejected status, got %s", rejected.StatusCode)
	}
}

func TestServiceStage7NatAclAndNetworkInterfacePermissionLifecycle(t *testing.T) {
	svc := NewService()

	eip1, err := svc.AllocateAddress("", nil)
	if err != nil {
		t.Fatalf("allocate address #1: %v", err)
	}
	nat, err := svc.CreateNatGateway(defaultSubnetID, eip1.AllocationID, "public", nil)
	if err != nil {
		t.Fatalf("create nat gateway: %v", err)
	}
	if nat.ID == "" {
		t.Fatalf("expected nat gateway id")
	}
	assigned, err := svc.AssignPrivateNatGatewayAddress(nat.ID, nil, 2)
	if err != nil {
		t.Fatalf("assign private nat gateway address: %v", err)
	}
	if len(assigned) != 2 {
		t.Fatalf("expected two assigned private nat gateway addresses")
	}

	eip2, err := svc.AllocateAddress("", nil)
	if err != nil {
		t.Fatalf("allocate address #2: %v", err)
	}
	associated, err := svc.AssociateNatGatewayAddress(nat.ID, []string{eip2.AllocationID}, nil)
	if err != nil {
		t.Fatalf("associate nat gateway address: %v", err)
	}
	if len(associated) != 1 || associated[0].AssociationID == "" {
		t.Fatalf("expected associated nat gateway address")
	}
	disassociated, err := svc.DisassociateNatGatewayAddress(nat.ID, []string{associated[0].AssociationID})
	if err != nil {
		t.Fatalf("disassociate nat gateway address: %v", err)
	}
	if len(disassociated) != 1 {
		t.Fatalf("expected one disassociated nat gateway address")
	}

	unassigned, err := svc.UnassignPrivateNatGatewayAddress(nat.ID, []string{assigned[0].PrivateIP})
	if err != nil {
		t.Fatalf("unassign private nat gateway address: %v", err)
	}
	if len(unassigned) != 1 {
		t.Fatalf("expected one unassigned private nat gateway address")
	}

	vpcA, err := svc.CreateVpc("10.81.0.0/16", nil)
	if err != nil {
		t.Fatalf("create vpc A: %v", err)
	}
	vpcB, err := svc.CreateVpc("10.82.0.0/16", nil)
	if err != nil {
		t.Fatalf("create vpc B: %v", err)
	}
	connection, err := svc.CreateVpcPeeringConnection(vpcA.ID, vpcB.ID, nil)
	if err != nil {
		t.Fatalf("create peering connection: %v", err)
	}
	trueValue := true
	accepter, requester, err := svc.ModifyVpcPeeringConnectionOptions(connection.ID,
		&PeeringConnectionOptionsPatch{AllowDNSResolutionFromRemoteVPC: &trueValue},
		&PeeringConnectionOptionsPatch{AllowDNSResolutionFromRemoteVPC: &trueValue},
	)
	if err != nil {
		t.Fatalf("modify vpc peering connection options: %v", err)
	}
	if !accepter.AllowDNSResolutionFromRemoteVPC || !requester.AllowDNSResolutionFromRemoteVPC {
		t.Fatalf("expected dns resolution enabled for both sides")
	}

	subnet, err := svc.CreateSubnet(vpcA.ID, "10.81.1.0/24", "us-east-1a", nil)
	if err != nil {
		t.Fatalf("create subnet: %v", err)
	}
	var currentAssocID string
	networkACLs := svc.DescribeNetworkACLs(nil, []string{vpcA.ID})
	for _, acl := range networkACLs {
		if !acl.IsDefault {
			continue
		}
		for _, assoc := range acl.Associations {
			if assoc.SubnetID == subnet.ID {
				currentAssocID = assoc.ID
				break
			}
		}
		if currentAssocID != "" {
			break
		}
	}
	if currentAssocID == "" {
		t.Fatalf("expected subnet association in default acl")
	}
	replacementACL, err := svc.CreateNetworkACL(vpcA.ID, nil)
	if err != nil {
		t.Fatalf("create replacement acl: %v", err)
	}
	replacementAssoc, err := svc.ReplaceNetworkACLAssociation(currentAssocID, replacementACL.ID)
	if err != nil {
		t.Fatalf("replace network acl association: %v", err)
	}
	if replacementAssoc.ID == "" || replacementAssoc.SubnetID != subnet.ID {
		t.Fatalf("unexpected replacement association: %+v", replacementAssoc)
	}

	iface, err := svc.CreateNetworkInterface(defaultSubnetID, "stage7", "10.0.0.77", []string{"sg-00000000"}, nil)
	if err != nil {
		t.Fatalf("create network interface: %v", err)
	}
	perm, err := svc.CreateNetworkInterfacePermission(iface.ID, "INSTANCE-ATTACH", DefaultAccountID, "")
	if err != nil {
		t.Fatalf("create network interface permission: %v", err)
	}
	if perm.ID == "" {
		t.Fatalf("expected network interface permission id")
	}
	describedPerms := svc.DescribeNetworkInterfacePermissions([]string{perm.ID}, nil, nil, nil, nil)
	if len(describedPerms) != 1 {
		t.Fatalf("expected one network interface permission")
	}
	falseValue := false
	if err := svc.ModifyNetworkInterfaceAttribute(iface.ID, nil, &falseValue, nil); err != nil {
		t.Fatalf("modify network interface source-dest-check: %v", err)
	}
	attr, err := svc.DescribeNetworkInterfaceAttribute(iface.ID, "sourceDestCheck")
	if err != nil {
		t.Fatalf("describe network interface attribute: %v", err)
	}
	if attr.SourceDestCheck {
		t.Fatalf("expected source-dest-check false before reset")
	}
	if err := svc.ResetNetworkInterfaceAttribute(iface.ID, "sourceDestCheck"); err != nil {
		t.Fatalf("reset network interface attribute: %v", err)
	}
	attrAfterReset, err := svc.DescribeNetworkInterfaceAttribute(iface.ID, "sourceDestCheck")
	if err != nil {
		t.Fatalf("describe network interface attribute after reset: %v", err)
	}
	if !attrAfterReset.SourceDestCheck {
		t.Fatalf("expected source-dest-check true after reset")
	}
	if err := svc.DeleteNetworkInterfacePermission(perm.ID); err != nil {
		t.Fatalf("delete network interface permission: %v", err)
	}
}

func TestServiceStage8ImageLifecycle(t *testing.T) {
	svc := NewService()

	res, err := svc.RunInstances("ami-base", "t3.micro", "", defaultSubnetID, "", nil, 1, 1, nil)
	if err != nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := res.Instances[0].ID

	image, err := svc.CreateImage(instanceID, "stage8-image", "stage8 description", false, []Tag{{Key: "env", Value: "stage8"}})
	if err != nil {
		t.Fatalf("create image: %v", err)
	}
	if image.ID == "" {
		t.Fatalf("expected image id")
	}

	described := svc.DescribeImages([]string{image.ID}, []string{"self"})
	if len(described) != 1 || described[0].ID != image.ID {
		t.Fatalf("expected described image %s", image.ID)
	}

	attr, err := svc.DescribeImageAttribute(image.ID, "description")
	if err != nil {
		t.Fatalf("describe image attribute description: %v", err)
	}
	if attr.Description != "stage8 description" {
		t.Fatalf("unexpected image description: %s", attr.Description)
	}

	updatedDescription := "stage8 updated"
	if err := svc.ModifyImageAttribute(image.ID, "description", &updatedDescription, nil); err != nil {
		t.Fatalf("modify image description: %v", err)
	}
	if err := svc.ModifyImageAttribute(image.ID, "launchPermission", nil, &LaunchPermissionModifications{
		Add: []LaunchPermission{{UserID: "111122223333"}},
	}); err != nil {
		t.Fatalf("modify image launch permission: %v", err)
	}

	launchAttr, err := svc.DescribeImageAttribute(image.ID, "launchPermission")
	if err != nil {
		t.Fatalf("describe image launch permission: %v", err)
	}
	if len(launchAttr.LaunchPermissions) != 1 || launchAttr.LaunchPermissions[0].UserID != "111122223333" {
		t.Fatalf("unexpected launch permissions: %+v", launchAttr.LaunchPermissions)
	}

	if err := svc.ResetImageAttribute(image.ID, "launchPermission"); err != nil {
		t.Fatalf("reset image launch permission: %v", err)
	}
	launchAttrAfterReset, err := svc.DescribeImageAttribute(image.ID, "launchPermission")
	if err != nil {
		t.Fatalf("describe image launch permission after reset: %v", err)
	}
	if len(launchAttrAfterReset.LaunchPermissions) != 0 {
		t.Fatalf("expected no launch permissions after reset")
	}

	if err := svc.CreateTags([]string{image.ID}, []Tag{{Key: "team", Value: "platform"}}); err != nil {
		t.Fatalf("create image tags: %v", err)
	}
	tags := svc.DescribeTags([]string{image.ID})
	if len(tags) == 0 {
		t.Fatalf("expected image tags")
	}

	if err := svc.DeregisterImage(image.ID); err != nil {
		t.Fatalf("deregister image: %v", err)
	}
	describedAfterDelete := svc.DescribeImages([]string{image.ID}, nil)
	if len(describedAfterDelete) != 0 {
		t.Fatalf("expected no images after deregister")
	}
}

func TestServiceStage9InstanceAttributeAndMonitoringLifecycle(t *testing.T) {
	svc := NewService()

	res, err := svc.RunInstances("ami-stage9", "t3.micro", "", defaultSubnetID, "", nil, 1, 1, nil)
	if err != nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := res.Instances[0].ID

	attr, err := svc.DescribeInstanceAttribute(instanceID, "instanceType")
	if err != nil {
		t.Fatalf("describe instance attribute: %v", err)
	}
	if attr.InstanceType != "t3.micro" {
		t.Fatalf("unexpected instance type: %s", attr.InstanceType)
	}

	newType := "t3.small"
	if err := svc.ModifyInstanceAttribute(instanceID, "instanceType", InstanceAttributePatch{InstanceType: &newType}); err != nil {
		t.Fatalf("modify instance type: %v", err)
	}
	updatedAttr, err := svc.DescribeInstanceAttribute(instanceID, "instanceType")
	if err != nil {
		t.Fatalf("describe instance type after modify: %v", err)
	}
	if updatedAttr.InstanceType != "t3.small" {
		t.Fatalf("expected updated instance type")
	}

	falseValue := false
	if err := svc.ModifyInstanceAttribute(instanceID, "sourceDestCheck", InstanceAttributePatch{SourceDestCheck: &falseValue}); err != nil {
		t.Fatalf("modify source dest check: %v", err)
	}
	sourceDestAttr, err := svc.DescribeInstanceAttribute(instanceID, "sourceDestCheck")
	if err != nil {
		t.Fatalf("describe source dest check: %v", err)
	}
	if sourceDestAttr.SourceDestCheck {
		t.Fatalf("expected sourceDestCheck false before reset")
	}
	if err := svc.ResetInstanceAttribute(instanceID, "sourceDestCheck"); err != nil {
		t.Fatalf("reset instance attribute: %v", err)
	}
	sourceDestAfterReset, err := svc.DescribeInstanceAttribute(instanceID, "sourceDestCheck")
	if err != nil {
		t.Fatalf("describe source dest check after reset: %v", err)
	}
	if !sourceDestAfterReset.SourceDestCheck {
		t.Fatalf("expected sourceDestCheck true after reset")
	}

	monitoring, err := svc.MonitorInstances([]string{instanceID})
	if err != nil {
		t.Fatalf("monitor instances: %v", err)
	}
	if len(monitoring) != 1 || monitoring[0].State != "enabled" {
		t.Fatalf("expected enabled monitoring state")
	}
	unmonitoring, err := svc.UnmonitorInstances([]string{instanceID})
	if err != nil {
		t.Fatalf("unmonitor instances: %v", err)
	}
	if len(unmonitoring) != 1 || unmonitoring[0].State != "disabled" {
		t.Fatalf("expected disabled monitoring state")
	}
}

func TestServiceStage10ConsoleAndPasswordData(t *testing.T) {
	svc := NewService()

	res, err := svc.RunInstances("ami-stage10", "t3.micro", "", defaultSubnetID, "", nil, 1, 1, nil)
	if err != nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := res.Instances[0].ID

	consoleOut, err := svc.GetConsoleOutput(instanceID, true)
	if err != nil {
		t.Fatalf("get console output: %v", err)
	}
	if consoleOut.InstanceID != instanceID || consoleOut.Output == "" || consoleOut.Timestamp.IsZero() {
		t.Fatalf("unexpected console output payload: %+v", consoleOut)
	}

	screenshot, err := svc.GetConsoleScreenshot(instanceID, false)
	if err != nil {
		t.Fatalf("get console screenshot: %v", err)
	}
	if screenshot.InstanceID != instanceID || screenshot.ImageData == "" {
		t.Fatalf("unexpected console screenshot payload: %+v", screenshot)
	}

	passwordData, err := svc.GetPasswordData(instanceID)
	if err != nil {
		t.Fatalf("get password data: %v", err)
	}
	if passwordData.InstanceID != instanceID || passwordData.PasswordData == "" || passwordData.Timestamp.IsZero() {
		t.Fatalf("unexpected password data payload: %+v", passwordData)
	}
}

func TestServiceStage11EbsEncryptionDefaultsLifecycle(t *testing.T) {
	svc := NewService()

	if svc.GetEbsEncryptionByDefault() {
		t.Fatalf("expected EBS encryption by default to start disabled")
	}
	originalKMS := svc.GetEbsDefaultKmsKeyID()
	if originalKMS == "" {
		t.Fatalf("expected default EBS KMS key id")
	}

	enabled := svc.EnableEbsEncryptionByDefault()
	if !enabled || !svc.GetEbsEncryptionByDefault() {
		t.Fatalf("expected EBS encryption by default to be enabled")
	}

	updatedKMS, err := svc.ModifyEbsDefaultKmsKeyID("arn:aws:kms:us-east-1:123456789012:key/stage11")
	if err != nil {
		t.Fatalf("modify default EBS KMS key id: %v", err)
	}
	if updatedKMS != "arn:aws:kms:us-east-1:123456789012:key/stage11" {
		t.Fatalf("unexpected updated KMS key id: %s", updatedKMS)
	}
	if svc.GetEbsDefaultKmsKeyID() != updatedKMS {
		t.Fatalf("expected updated KMS key id from getter")
	}

	resetKMS := svc.ResetEbsDefaultKmsKeyID()
	if resetKMS == "" || svc.GetEbsDefaultKmsKeyID() != resetKMS {
		t.Fatalf("expected reset KMS key id to be applied")
	}
	if resetKMS == updatedKMS {
		t.Fatalf("expected reset KMS key id to differ from custom key")
	}

	disabled := svc.DisableEbsEncryptionByDefault()
	if disabled || svc.GetEbsEncryptionByDefault() {
		t.Fatalf("expected EBS encryption by default to be disabled")
	}
}

func TestServiceStage12IDFormatLifecycle(t *testing.T) {
	svc := NewService()

	statuses, err := svc.DescribeIDFormat("instance")
	if err != nil {
		t.Fatalf("describe id format: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Resource != "instance" || !statuses[0].UseLongIDs {
		t.Fatalf("unexpected id format statuses: %+v", statuses)
	}

	if err := svc.ModifyIDFormat("instance", false); err != nil {
		t.Fatalf("modify id format: %v", err)
	}

	statusesAfterModify, err := svc.DescribeIDFormat("instance")
	if err != nil {
		t.Fatalf("describe id format after modify: %v", err)
	}
	if len(statusesAfterModify) != 1 || statusesAfterModify[0].UseLongIDs {
		t.Fatalf("expected instance useLongIds false after modify")
	}

	identityStatuses, err := svc.DescribeIdentityIDFormat("arn:aws:iam::123456789012:user/stage12", "instance")
	if err != nil {
		t.Fatalf("describe identity id format: %v", err)
	}
	if len(identityStatuses) != 1 || identityStatuses[0].Resource != "instance" || identityStatuses[0].UseLongIDs {
		t.Fatalf("unexpected identity statuses: %+v", identityStatuses)
	}

	principals := svc.DescribePrincipalIDFormat([]string{"instance"})
	if len(principals) == 0 || principals[0].ARN == "" || len(principals[0].Statuses) != 1 || principals[0].Statuses[0].Resource != "instance" || principals[0].Statuses[0].UseLongIDs {
		t.Fatalf("unexpected principal id format output: %+v", principals)
	}
}

func TestServiceStage13ModifyIdentityIDFormat(t *testing.T) {
	svc := NewService()

	principalARN := "arn:aws:iam::123456789012:user/stage13"
	if err := svc.ModifyIdentityIDFormat(principalARN, "instance", false); err != nil {
		t.Fatalf("modify identity id format: %v", err)
	}

	statuses, err := svc.DescribeIdentityIDFormat(principalARN, "instance")
	if err != nil {
		t.Fatalf("describe identity id format after modify: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Resource != "instance" || statuses[0].UseLongIDs {
		t.Fatalf("unexpected identity id format statuses: %+v", statuses)
	}

	if err := svc.ModifyIdentityIDFormat(principalARN, "instance", true); err != nil {
		t.Fatalf("modify identity id format to true: %v", err)
	}
	statusesAfterSecondModify, err := svc.DescribeIdentityIDFormat(principalARN, "instance")
	if err != nil {
		t.Fatalf("describe identity id format after second modify: %v", err)
	}
	if len(statusesAfterSecondModify) != 1 || !statusesAfterSecondModify[0].UseLongIDs {
		t.Fatalf("expected identity id format to be restored to true")
	}
}

func TestServiceStage14DescribeAggregateIDFormat(t *testing.T) {
	svc := NewService()

	beforeStatuses, beforeAggregated := svc.DescribeAggregateIDFormat()
	if len(beforeStatuses) == 0 || !beforeAggregated {
		t.Fatalf("expected aggregate id format to start fully enabled")
	}

	if err := svc.ModifyIdentityIDFormat("arn:aws:iam::123456789012:user/stage14", "instance", false); err != nil {
		t.Fatalf("modify identity id format: %v", err)
	}

	afterStatuses, afterAggregated := svc.DescribeAggregateIDFormat()
	if len(afterStatuses) == 0 || afterAggregated {
		t.Fatalf("expected aggregate id format to be not fully enabled after override")
	}

	var foundInstance bool
	for _, status := range afterStatuses {
		if status.Resource != "instance" {
			continue
		}
		foundInstance = true
		if status.UseLongIDs {
			t.Fatalf("expected aggregate instance useLongIds to be false")
		}
	}
	if !foundInstance {
		t.Fatalf("expected instance resource in aggregate status list")
	}
}

func TestServiceStage15AddressTransferLifecycle(t *testing.T) {
	svc := NewService()

	address, err := svc.AllocateAddress("", nil)
	if err != nil {
		t.Fatalf("allocate address: %v", err)
	}

	enabledTransfer, err := svc.EnableAddressTransfer(address.AllocationID, "210987654321")
	if err != nil {
		t.Fatalf("enable address transfer: %v", err)
	}
	if enabledTransfer.AddressTransferStatus != "pending" || enabledTransfer.AllocationID != address.AllocationID || enabledTransfer.PublicIP != address.PublicIP || enabledTransfer.TransferAccountID != "210987654321" || enabledTransfer.TransferOfferExpirationTime == nil {
		t.Fatalf("unexpected enabled transfer payload: %+v", enabledTransfer)
	}

	addressTransfers := svc.DescribeAddressTransfers([]string{address.AllocationID})
	if len(addressTransfers) != 1 || addressTransfers[0].AddressTransferStatus != "pending" {
		t.Fatalf("unexpected described transfers: %+v", addressTransfers)
	}

	acceptedTransfer, err := svc.AcceptAddressTransfer(address.PublicIP)
	if err != nil {
		t.Fatalf("accept address transfer: %v", err)
	}
	if acceptedTransfer.AddressTransferStatus != "accepted" || acceptedTransfer.TransferOfferAcceptedTimestamp == nil {
		t.Fatalf("unexpected accepted transfer payload: %+v", acceptedTransfer)
	}

	disabledTransfer, err := svc.DisableAddressTransfer(address.AllocationID)
	if err != nil {
		t.Fatalf("disable address transfer: %v", err)
	}
	if disabledTransfer.AddressTransferStatus != "disabled" {
		t.Fatalf("unexpected disabled transfer payload: %+v", disabledTransfer)
	}

	movingStatuses := svc.DescribeMovingAddresses([]string{address.PublicIP})
	if len(movingStatuses) != 1 || movingStatuses[0].PublicIP != address.PublicIP || movingStatuses[0].MoveStatus != "restoringToClassic" {
		t.Fatalf("unexpected moving address statuses: %+v", movingStatuses)
	}
}

func TestServiceStage16PlacementGroupLifecycle(t *testing.T) {
	svc := NewService()

	group, err := svc.CreatePlacementGroup("stage16-group", "partition", 3, "rack", []Tag{{Key: "env", Value: "stage16"}})
	if err != nil {
		t.Fatalf("create placement group: %v", err)
	}
	if group.GroupName != "stage16-group" || group.GroupID == "" || group.GroupARN == "" || group.State != "available" || group.Strategy != "partition" || group.PartitionCount != 3 {
		t.Fatalf("unexpected placement group payload: %+v", group)
	}

	describedByName := svc.DescribePlacementGroups([]string{"stage16-group"}, nil)
	if len(describedByName) != 1 || describedByName[0].GroupName != "stage16-group" {
		t.Fatalf("describe placement groups by name: %+v", describedByName)
	}
	describedByID := svc.DescribePlacementGroups(nil, []string{group.GroupID})
	if len(describedByID) != 1 || describedByID[0].GroupID != group.GroupID {
		t.Fatalf("describe placement groups by id: %+v", describedByID)
	}

	if err := svc.DeletePlacementGroup("stage16-group"); err != nil {
		t.Fatalf("delete placement group: %v", err)
	}
	if got := svc.DescribePlacementGroups([]string{"stage16-group"}, nil); len(got) != 0 {
		t.Fatalf("expected no placement groups after delete")
	}
}

func TestServiceStage17CustomerGatewayLifecycle(t *testing.T) {
	svc := NewService()

	bgpASN := int64(65010)
	gateway, err := svc.CreateCustomerGateway(
		"ipsec.1",
		"198.51.100.10",
		&bgpASN,
		nil,
		"",
		"stage17-device",
		[]Tag{{Key: "env", Value: "stage17"}},
	)
	if err != nil {
		t.Fatalf("create customer gateway: %v", err)
	}
	if gateway.ID == "" || gateway.IPAddress != "198.51.100.10" || gateway.Type != "ipsec.1" || gateway.BgpASN != "65010" || gateway.State != "available" {
		t.Fatalf("unexpected customer gateway payload: %+v", gateway)
	}

	described := svc.DescribeCustomerGateways(
		[]string{gateway.ID},
		nil,
		[]string{"198.51.100.10"},
		[]string{"available"},
		[]string{"ipsec.1"},
		[]string{"65010"},
		[]string{"env"},
		map[string][]string{"env": []string{"stage17"}},
	)
	if len(described) != 1 || described[0].ID != gateway.ID {
		t.Fatalf("describe customer gateways with filters: %+v", described)
	}

	if err := svc.DeleteCustomerGateway(gateway.ID); err != nil {
		t.Fatalf("delete customer gateway: %v", err)
	}

	describedDeleted := svc.DescribeCustomerGateways(
		[]string{gateway.ID},
		nil,
		nil,
		[]string{"deleted"},
		nil,
		nil,
		nil,
		nil,
	)
	if len(describedDeleted) != 1 || describedDeleted[0].State != "deleted" {
		t.Fatalf("expected deleted customer gateway in describe output")
	}
}

func TestServiceStage18VpnGatewayLifecycle(t *testing.T) {
	svc := NewService()

	amazonSideASN := int64(65020)
	gateway, err := svc.CreateVpnGateway("ipsec.1", &amazonSideASN, "us-east-1a", []Tag{{Key: "env", Value: "stage18"}})
	if err != nil {
		t.Fatalf("create vpn gateway: %v", err)
	}
	if gateway.ID == "" || gateway.AmazonSideASN != 65020 || gateway.Type != "ipsec.1" || gateway.State != "available" {
		t.Fatalf("unexpected vpn gateway payload: %+v", gateway)
	}

	attachment, err := svc.AttachVpnGateway(gateway.ID, "vpc-00000001")
	if err != nil {
		t.Fatalf("attach vpn gateway: %v", err)
	}
	if attachment.VpcID != "vpc-00000001" || attachment.State != "attached" {
		t.Fatalf("unexpected vpn gateway attachment: %+v", attachment)
	}

	described := svc.DescribeVpnGateways(
		[]string{gateway.ID},
		nil,
		[]string{"vpc-00000001"},
		[]string{"attached"},
		[]string{"available"},
		[]string{"ipsec.1"},
		[]string{"us-east-1a"},
		[]string{"65020"},
		[]string{"env"},
		map[string][]string{"env": []string{"stage18"}},
	)
	if len(described) != 1 || described[0].ID != gateway.ID {
		t.Fatalf("describe vpn gateways with filters: %+v", described)
	}

	if err := svc.DetachVpnGateway(gateway.ID, "vpc-00000001"); err != nil {
		t.Fatalf("detach vpn gateway: %v", err)
	}

	if err := svc.DeleteVpnGateway(gateway.ID); err != nil {
		t.Fatalf("delete vpn gateway: %v", err)
	}

	describedDeleted := svc.DescribeVpnGateways(
		[]string{gateway.ID},
		nil,
		nil,
		nil,
		[]string{"deleted"},
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	if len(describedDeleted) != 1 || describedDeleted[0].State != "deleted" {
		t.Fatalf("expected deleted vpn gateway in describe output")
	}
}

func TestServiceStage19VpnConnectionLifecycle(t *testing.T) {
	svc := NewService()

	bgpASN := int64(65030)
	customerGateway, err := svc.CreateCustomerGateway(
		"ipsec.1",
		"198.51.100.30",
		&bgpASN,
		nil,
		"",
		"stage19-cgw",
		nil,
	)
	if err != nil {
		t.Fatalf("create customer gateway: %v", err)
	}

	vpnGateway, err := svc.CreateVpnGateway("ipsec.1", nil, "us-east-1a", nil)
	if err != nil {
		t.Fatalf("create vpn gateway: %v", err)
	}

	staticRoutesOnly := true
	connection, err := svc.CreateVpnConnection(
		customerGateway.ID,
		"ipsec.1",
		vpnGateway.ID,
		"",
		&staticRoutesOnly,
		[]Tag{{Key: "env", Value: "stage19"}},
	)
	if err != nil {
		t.Fatalf("create vpn connection: %v", err)
	}
	if connection.ID == "" || connection.CustomerGatewayID != customerGateway.ID || connection.VpnGatewayID != vpnGateway.ID || !connection.Options.StaticRoutesOnly || connection.State != "available" {
		t.Fatalf("unexpected vpn connection payload: %+v", connection)
	}

	if err := svc.CreateVpnConnectionRoute(connection.ID, "10.200.0.0/16"); err != nil {
		t.Fatalf("create vpn connection route: %v", err)
	}

	described := svc.DescribeVpnConnections(
		[]string{connection.ID},
		nil,
		[]string{customerGateway.ID},
		[]string{"available"},
		[]string{"ipsec.1"},
		[]string{vpnGateway.ID},
		nil,
		nil,
		[]string{"10.200.0.0/16"},
		[]string{"65030"},
		[]string{"env"},
		[]bool{true},
		map[string][]string{"env": []string{"stage19"}},
	)
	if len(described) != 1 || described[0].ID != connection.ID || len(described[0].Routes) != 1 || described[0].Routes[0].DestinationCIDRBlock != "10.200.0.0/16" {
		t.Fatalf("unexpected described vpn connections: %+v", described)
	}

	if err := svc.DeleteVpnConnectionRoute(connection.ID, "10.200.0.0/16"); err != nil {
		t.Fatalf("delete vpn connection route: %v", err)
	}
	if err := svc.DeleteVpnConnection(connection.ID); err != nil {
		t.Fatalf("delete vpn connection: %v", err)
	}

	describedDeleted := svc.DescribeVpnConnections(
		[]string{connection.ID},
		nil,
		nil,
		[]string{"deleted"},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	if len(describedDeleted) != 1 || describedDeleted[0].State != "deleted" {
		t.Fatalf("expected deleted vpn connection in describe output")
	}
}

func TestServiceStage20ModifyVpnConnectionLifecycle(t *testing.T) {
	svc := NewService()

	bgpASN1 := int64(65050)
	customerGateway1, err := svc.CreateCustomerGateway("ipsec.1", "198.51.100.50", &bgpASN1, nil, "", "stage20-cgw-1", nil)
	if err != nil {
		t.Fatalf("create customer gateway #1: %v", err)
	}
	bgpASN2 := int64(65051)
	customerGateway2, err := svc.CreateCustomerGateway("ipsec.1", "198.51.100.51", &bgpASN2, nil, "", "stage20-cgw-2", nil)
	if err != nil {
		t.Fatalf("create customer gateway #2: %v", err)
	}
	vpnGateway, err := svc.CreateVpnGateway("ipsec.1", nil, "us-east-1a", nil)
	if err != nil {
		t.Fatalf("create vpn gateway: %v", err)
	}

	connection, err := svc.CreateVpnConnection(customerGateway1.ID, "ipsec.1", vpnGateway.ID, "", nil, nil)
	if err != nil {
		t.Fatalf("create vpn connection: %v", err)
	}

	modifiedConnection, err := svc.ModifyVpnConnection(connection.ID, customerGateway2.ID, "", "")
	if err != nil {
		t.Fatalf("modify vpn connection: %v", err)
	}
	if modifiedConnection.CustomerGatewayID != customerGateway2.ID {
		t.Fatalf("expected modified customer gateway id %s, got %s", customerGateway2.ID, modifiedConnection.CustomerGatewayID)
	}

	localIPv4 := "10.250.0.0/16"
	remoteIPv4 := "10.251.0.0/16"
	modifiedOptionsConnection, err := svc.ModifyVpnConnectionOptions(connection.ID, &localIPv4, nil, &remoteIPv4, nil)
	if err != nil {
		t.Fatalf("modify vpn connection options: %v", err)
	}
	if modifiedOptionsConnection.Options.LocalIpv4NetworkCidr != localIPv4 || modifiedOptionsConnection.Options.RemoteIpv4NetworkCidr != remoteIPv4 {
		t.Fatalf("unexpected modified vpn connection options: %+v", modifiedOptionsConnection.Options)
	}
}

func TestServiceStage21ClientVpnLifecycle(t *testing.T) {
	svc := NewService()

	vpnPort := int32(443)
	splitTunnel := true
	sessionTimeoutHours := int32(24)
	disconnectOnSessionTimeout := false
	endpoint, err := svc.CreateClientVpnEndpoint(
		"certificate-authentication",
		"172.16.0.0/22",
		"arn:aws:acm:us-east-1:123456789012:certificate/stage21",
		"stage21-endpoint",
		"udp",
		&vpnPort,
		&splitTunnel,
		nil,
		defaultVPCID,
		[]string{"10.0.0.2"},
		&sessionTimeoutHours,
		&disconnectOnSessionTimeout,
		"enabled",
		[]Tag{{Key: "env", Value: "stage21"}},
	)
	if err != nil {
		t.Fatalf("create client vpn endpoint: %v", err)
	}
	if endpoint.ID == "" || endpoint.Status.Code != "pending-associate" || endpoint.TransportProtocol != "udp" {
		t.Fatalf("unexpected endpoint payload: %+v", endpoint)
	}

	describedEndpoints := svc.DescribeClientVpnEndpoints([]string{endpoint.ID}, []string{endpoint.ID}, []string{"udp"})
	if len(describedEndpoints) != 1 || describedEndpoints[0].ID != endpoint.ID {
		t.Fatalf("describe client vpn endpoints: %+v", describedEndpoints)
	}

	targetNetwork, err := svc.AssociateClientVpnTargetNetwork(endpoint.ID, defaultSubnetID)
	if err != nil {
		t.Fatalf("associate client vpn target network: %v", err)
	}
	if targetNetwork.AssociationID == "" || targetNetwork.Status.Code != "associated" || targetNetwork.TargetNetworkID != defaultSubnetID {
		t.Fatalf("unexpected target network: %+v", targetNetwork)
	}

	appliedSecurityGroupIDs, err := svc.ApplySecurityGroupsToClientVpnTargetNetwork(endpoint.ID, defaultVPCID, []string{"sg-00000000"})
	if err != nil {
		t.Fatalf("apply security groups: %v", err)
	}
	if len(appliedSecurityGroupIDs) != 1 || appliedSecurityGroupIDs[0] != "sg-00000000" {
		t.Fatalf("unexpected applied security groups: %+v", appliedSecurityGroupIDs)
	}

	routeStatus, err := svc.CreateClientVpnRoute(endpoint.ID, "10.240.0.0/16", defaultSubnetID, "stage21-route")
	if err != nil {
		t.Fatalf("create client vpn route: %v", err)
	}
	if routeStatus.Code != "active" {
		t.Fatalf("unexpected create route status: %+v", routeStatus)
	}

	routes, err := svc.DescribeClientVpnRoutes(endpoint.ID, []string{"10.240.0.0/16"}, []string{"add-route"}, []string{defaultSubnetID})
	if err != nil {
		t.Fatalf("describe client vpn routes: %v", err)
	}
	if len(routes) != 1 || routes[0].DestinationCIDR != "10.240.0.0/16" || routes[0].TargetSubnet != defaultSubnetID {
		t.Fatalf("unexpected described routes: %+v", routes)
	}

	authStatus, err := svc.AuthorizeClientVpnIngress(endpoint.ID, "10.240.0.0/16", "grp-stage21", false, "stage21-rule")
	if err != nil {
		t.Fatalf("authorize client vpn ingress: %v", err)
	}
	if authStatus.Code != "active" {
		t.Fatalf("unexpected authorize status: %+v", authStatus)
	}

	authorizationRules, err := svc.DescribeClientVpnAuthorizationRules(endpoint.ID, []string{"stage21-rule"}, []string{"10.240.0.0/16"}, []string{"grp-stage21"})
	if err != nil {
		t.Fatalf("describe client vpn authorization rules: %v", err)
	}
	if len(authorizationRules) != 1 || authorizationRules[0].DestinationCIDR != "10.240.0.0/16" {
		t.Fatalf("unexpected authorization rules: %+v", authorizationRules)
	}

	connections, err := svc.DescribeClientVpnConnections(endpoint.ID, nil, nil)
	if err != nil {
		t.Fatalf("describe client vpn connections: %v", err)
	}
	if len(connections) == 0 {
		t.Fatalf("expected seeded connection for client vpn endpoint")
	}

	terminated, err := svc.TerminateClientVpnConnections(endpoint.ID, connections[0].ConnectionID, "")
	if err != nil {
		t.Fatalf("terminate client vpn connections: %v", err)
	}
	if terminated.ClientVpnEndpointID != endpoint.ID || len(terminated.ConnectionStatuses) != 1 || terminated.ConnectionStatuses[0].CurrentStatus.Code != "terminated" {
		t.Fatalf("unexpected terminate response: %+v", terminated)
	}

	describedTargetNetworks, err := svc.DescribeClientVpnTargetNetworks(endpoint.ID, []string{targetNetwork.AssociationID}, []string{targetNetwork.AssociationID}, []string{defaultSubnetID}, []string{defaultVPCID})
	if err != nil {
		t.Fatalf("describe client vpn target networks: %v", err)
	}
	if len(describedTargetNetworks) != 1 || describedTargetNetworks[0].AssociationID != targetNetwork.AssociationID {
		t.Fatalf("unexpected target network describe response: %+v", describedTargetNetworks)
	}

	clientConfiguration, err := svc.ExportClientVpnClientConfiguration(endpoint.ID)
	if err != nil {
		t.Fatalf("export client vpn client configuration: %v", err)
	}
	if !strings.Contains(clientConfiguration, endpoint.DnsName) {
		t.Fatalf("expected exported client configuration to contain endpoint dns name")
	}

	certificateRevocationList, certificateRevocationListStatus, err := svc.ExportClientVpnClientCertificateRevocationList(endpoint.ID)
	if err != nil {
		t.Fatalf("export client vpn certificate revocation list: %v", err)
	}
	if strings.TrimSpace(certificateRevocationList) == "" || certificateRevocationListStatus.Code != "active" {
		t.Fatalf("unexpected certificate revocation list export response")
	}

	if ret, err := svc.ImportClientVpnClientCertificateRevocationList(endpoint.ID, "-----BEGIN X509 CRL-----\nSTAGE21\n-----END X509 CRL-----"); err != nil {
		t.Fatalf("import client vpn certificate revocation list: %v", err)
	} else if !ret {
		t.Fatalf("expected import client vpn certificate revocation list to return true")
	}

	revokedStatus, err := svc.RevokeClientVpnIngress(endpoint.ID, "10.240.0.0/16", "grp-stage21", false)
	if err != nil {
		t.Fatalf("revoke client vpn ingress: %v", err)
	}
	if revokedStatus.Code != "revoking" {
		t.Fatalf("unexpected revoke status: %+v", revokedStatus)
	}

	deletedRouteStatus, err := svc.DeleteClientVpnRoute(endpoint.ID, "10.240.0.0/16", defaultSubnetID)
	if err != nil {
		t.Fatalf("delete client vpn route: %v", err)
	}
	if deletedRouteStatus.Code != "deleting" {
		t.Fatalf("unexpected delete route status: %+v", deletedRouteStatus)
	}

	disassociatedTargetNetwork, err := svc.DisassociateClientVpnTargetNetwork(endpoint.ID, targetNetwork.AssociationID)
	if err != nil {
		t.Fatalf("disassociate client vpn target network: %v", err)
	}
	if disassociatedTargetNetwork.Status.Code != "disassociated" {
		t.Fatalf("unexpected disassociate status: %+v", disassociatedTargetNetwork)
	}

	deletedEndpointStatus, err := svc.DeleteClientVpnEndpoint(endpoint.ID)
	if err != nil {
		t.Fatalf("delete client vpn endpoint: %v", err)
	}
	if deletedEndpointStatus.Code != "deleted" {
		t.Fatalf("unexpected delete endpoint status: %+v", deletedEndpointStatus)
	}

	describedAfterDelete := svc.DescribeClientVpnEndpoints([]string{endpoint.ID}, nil, nil)
	if len(describedAfterDelete) != 1 || describedAfterDelete[0].Status.Code != "deleted" {
		t.Fatalf("expected deleted endpoint in describe output")
	}
}

func TestServiceStage22VpnTunnelAndRoutePropagationLifecycle(t *testing.T) {
	svc := NewService()

	bgpASN := int64(65070)
	customerGateway, err := svc.CreateCustomerGateway("ipsec.1", "198.51.100.70", &bgpASN, nil, "", "stage22-cgw", nil)
	if err != nil {
		t.Fatalf("create customer gateway: %v", err)
	}
	vpnGateway, err := svc.CreateVpnGateway("ipsec.1", nil, "", nil)
	if err != nil {
		t.Fatalf("create vpn gateway: %v", err)
	}
	if _, err := svc.AttachVpnGateway(vpnGateway.ID, defaultVPCID); err != nil {
		t.Fatalf("attach vpn gateway: %v", err)
	}

	connection, err := svc.CreateVpnConnection(customerGateway.ID, "ipsec.1", vpnGateway.ID, "", nil, nil)
	if err != nil {
		t.Fatalf("create vpn connection: %v", err)
	}
	if len(connection.VgwTelemetry) == 0 {
		t.Fatalf("expected vpn telemetry to be initialized")
	}
	outsideIP := connection.VgwTelemetry[0].OutsideIPAddress
	if outsideIP == "" {
		t.Fatalf("expected telemetry outside ip address")
	}

	if err := svc.EnableVgwRoutePropagation(defaultRouteTableID, vpnGateway.ID); err != nil {
		t.Fatalf("enable vgw route propagation: %v", err)
	}

	activeStatus, err := svc.GetActiveVpnTunnelStatus(connection.ID, outsideIP)
	if err != nil {
		t.Fatalf("get active vpn tunnel status: %v", err)
	}
	if activeStatus.IkeVersion == "" || activeStatus.Phase1EncryptionAlgorithm == "" {
		t.Fatalf("unexpected active vpn tunnel status: %+v", activeStatus)
	}

	deviceTypes, nextToken, err := svc.GetVpnConnectionDeviceTypes(func() *int32 {
		value := int32(1)
		return &value
	}(), nil)
	if err != nil {
		t.Fatalf("get vpn connection device types: %v", err)
	}
	if len(deviceTypes) != 1 || nextToken == nil || strings.TrimSpace(*nextToken) == "" {
		t.Fatalf("unexpected vpn connection device types page: %+v nextToken=%v", deviceTypes, nextToken)
	}

	sampleConfiguration, err := svc.GetVpnConnectionDeviceSampleConfiguration(connection.ID, deviceTypes[0].ID, "ikev2", "recommended")
	if err != nil {
		t.Fatalf("get vpn connection device sample configuration: %v", err)
	}
	if !strings.Contains(sampleConfiguration, "vpn_connection_id="+connection.ID) {
		t.Fatalf("unexpected sample configuration: %q", sampleConfiguration)
	}

	replacementStatus, err := svc.GetVpnTunnelReplacementStatus(connection.ID, outsideIP)
	if err != nil {
		t.Fatalf("get vpn tunnel replacement status: %v", err)
	}
	if replacementStatus.VpnConnectionID != connection.ID || replacementStatus.VpnTunnelOutsideIPAddress != outsideIP {
		t.Fatalf("unexpected replacement status: %+v", replacementStatus)
	}

	modifiedCertificateConnection, err := svc.ModifyVpnTunnelCertificate(connection.ID, outsideIP)
	if err != nil {
		t.Fatalf("modify vpn tunnel certificate: %v", err)
	}
	if len(modifiedCertificateConnection.VgwTelemetry) == 0 || modifiedCertificateConnection.VgwTelemetry[0].CertificateARN == "" {
		t.Fatalf("expected vpn telemetry certificate arn to be updated")
	}

	tunnelInsideCidr := "169.254.21.0/30"
	modifiedOptionsConnection, err := svc.ModifyVpnTunnelOptions(connection.ID, outsideIP, ModifyVpnTunnelOptionsRequest{
		HasTunnelOptions: true,
		PreSharedKey: func() *string {
			value := "stackyardpsk"
			return &value
		}(),
		TunnelInsideCidr: &tunnelInsideCidr,
	})
	if err != nil {
		t.Fatalf("modify vpn tunnel options: %v", err)
	}
	if modifiedOptionsConnection.Options.LocalIpv4NetworkCidr != tunnelInsideCidr {
		t.Fatalf("expected tunnel inside cidr to update local ipv4 network cidr")
	}

	ret, err := svc.ReplaceVpnTunnel(connection.ID, outsideIP, true)
	if err != nil {
		t.Fatalf("replace vpn tunnel: %v", err)
	}
	if !ret {
		t.Fatalf("expected replace vpn tunnel to return true")
	}

	replacementStatusAfterReplace, err := svc.GetVpnTunnelReplacementStatus(connection.ID, outsideIP)
	if err != nil {
		t.Fatalf("get vpn tunnel replacement status after replace: %v", err)
	}
	if replacementStatusAfterReplace.MaintenanceDetails.LastMaintenanceApplied == nil {
		t.Fatalf("expected replacement maintenance details to include last maintenance applied")
	}

	if err := svc.DisableVgwRoutePropagation(defaultRouteTableID, vpnGateway.ID); err != nil {
		t.Fatalf("disable vgw route propagation: %v", err)
	}
}

func TestServiceStage23ClassicLinkLifecycle(t *testing.T) {
	svc := NewService()

	vpc, err := svc.CreateVpc("10.23.0.0/16", nil)
	if err != nil {
		t.Fatalf("create vpc: %v", err)
	}

	sg, err := svc.CreateSecurityGroup("stage23", "stage23 classic link", vpc.ID, nil)
	if err != nil {
		t.Fatalf("create security group: %v", err)
	}

	runInstancesOut, err := svc.RunInstances("ami-stage23", "t3.micro", "", defaultSubnetID, "", nil, 1, 1, nil)
	if err != nil || len(runInstancesOut.Instances) != 1 {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := runInstancesOut.Instances[0].ID

	ret, err := svc.EnableVpcClassicLink(vpc.ID)
	if err != nil || !ret {
		t.Fatalf("enable vpc classic link: %v", err)
	}

	ret, err = svc.EnableVpcClassicLinkDnsSupport(vpc.ID)
	if err != nil || !ret {
		t.Fatalf("enable vpc classic link dns support: %v", err)
	}

	vpcClassicLinks := svc.DescribeVpcClassicLink([]string{vpc.ID}, nil)
	if len(vpcClassicLinks) != 1 || !vpcClassicLinks[0].ClassicLinkEnabled {
		t.Fatalf("unexpected classic link describe output: %+v", vpcClassicLinks)
	}

	dnsSupportVpcs, nextToken, err := svc.DescribeVpcClassicLinkDnsSupport(nil, func() *int32 {
		value := int32(1)
		return &value
	}(), nil)
	if err != nil {
		t.Fatalf("describe vpc classic link dns support: %v", err)
	}
	if len(dnsSupportVpcs) != 1 || nextToken == nil || strings.TrimSpace(*nextToken) == "" {
		t.Fatalf("unexpected dns support describe page: %+v nextToken=%v", dnsSupportVpcs, nextToken)
	}

	ret, err = svc.AttachClassicLinkVpc(instanceID, vpc.ID, []string{sg.ID})
	if err != nil || !ret {
		t.Fatalf("attach classic link vpc: %v", err)
	}

	instances, nextToken, err := svc.DescribeClassicLinkInstances([]string{instanceID}, []string{vpc.ID}, []string{sg.ID}, nil, nil)
	if err != nil {
		t.Fatalf("describe classic link instances: %v", err)
	}
	if len(instances) != 1 || nextToken != nil || instances[0].InstanceID != instanceID {
		t.Fatalf("unexpected classic link instances output: %+v nextToken=%v", instances, nextToken)
	}

	if ret, err = svc.DisableVpcClassicLink(vpc.ID); err == nil || ret {
		t.Fatalf("expected disable vpc classic link to fail while attachments exist")
	}

	ret, err = svc.DetachClassicLinkVpc(instanceID, vpc.ID)
	if err != nil || !ret {
		t.Fatalf("detach classic link vpc: %v", err)
	}

	ret, err = svc.DisableVpcClassicLinkDnsSupport(vpc.ID)
	if err != nil || !ret {
		t.Fatalf("disable vpc classic link dns support: %v", err)
	}

	ret, err = svc.DisableVpcClassicLink(vpc.ID)
	if err != nil || !ret {
		t.Fatalf("disable vpc classic link: %v", err)
	}
}

func TestServiceStage24AddressTransitionsAndAttributes(t *testing.T) {
	svc := NewService()

	address, err := svc.AllocateAddress("198.51.100.24", nil)
	if err != nil {
		t.Fatalf("allocate address: %v", err)
	}

	attr, err := svc.ModifyAddressAttribute(address.AllocationID, "mail.example.com")
	if err != nil {
		t.Fatalf("modify address attribute: %v", err)
	}
	if attr.PtrRecord != "mail.example.com" {
		t.Fatalf("unexpected ptr record: %q", attr.PtrRecord)
	}

	attr, err = svc.ResetAddressAttribute(address.AllocationID, "domain-name")
	if err != nil {
		t.Fatalf("reset address attribute: %v", err)
	}
	if attr.PtrRecord == "" || strings.Contains(attr.PtrRecord, "mail.example.com") {
		t.Fatalf("unexpected ptr record after reset: %q", attr.PtrRecord)
	}

	allocationID, status, err := svc.MoveAddressToVpc(address.PublicIP)
	if err != nil {
		t.Fatalf("move address to vpc: %v", err)
	}
	if allocationID != address.AllocationID || status != "InVpc" {
		t.Fatalf("unexpected move address to vpc output: allocation=%q status=%q", allocationID, status)
	}

	publicIP, status, err := svc.RestoreAddressToClassic(address.PublicIP)
	if err != nil {
		t.Fatalf("restore address to classic: %v", err)
	}
	if publicIP != address.PublicIP || status != "InClassic" {
		t.Fatalf("unexpected restore address output: publicIp=%q status=%q", publicIP, status)
	}
}
