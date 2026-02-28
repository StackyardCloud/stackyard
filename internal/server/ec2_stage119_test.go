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

func TestEC2Stage119SDKLifecycle(t *testing.T) {
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

	createLaunchTemplateOut, err := client.CreateLaunchTemplate(ctx, &awsec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String("stage119-template"),
		LaunchTemplateData: &awsec2types.RequestLaunchTemplateData{ImageId: aws.String("ami-stage119")},
	})
	if err != nil {
		t.Fatalf("create launch template: %v", err)
	}
	if createLaunchTemplateOut.LaunchTemplate == nil || createLaunchTemplateOut.LaunchTemplate.LaunchTemplateId == nil {
		t.Fatalf("expected created launch template")
	}
	launchTemplateID := aws.ToString(createLaunchTemplateOut.LaunchTemplate.LaunchTemplateId)

	localGatewayID := "lgw-00000000119"
	createLocalGatewayRouteTableOut, err := client.CreateLocalGatewayRouteTable(ctx, &awsec2.CreateLocalGatewayRouteTableInput{
		LocalGatewayId: aws.String(localGatewayID),
	})
	if err != nil {
		t.Fatalf("create local gateway route table: %v", err)
	}
	if createLocalGatewayRouteTableOut.LocalGatewayRouteTable == nil || createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId == nil {
		t.Fatalf("expected created local gateway route table")
	}
	routeTableID := aws.ToString(createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId)

	createLocalGatewayVirtualInterfaceGroupOut, err := client.CreateLocalGatewayVirtualInterfaceGroup(ctx, &awsec2.CreateLocalGatewayVirtualInterfaceGroupInput{
		LocalGatewayId: aws.String(localGatewayID),
	})
	if err != nil {
		t.Fatalf("create local gateway virtual interface group: %v", err)
	}
	if createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup == nil || createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup.LocalGatewayVirtualInterfaceGroupId == nil {
		t.Fatalf("expected created local gateway virtual interface group")
	}
	groupID := aws.ToString(createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup.LocalGatewayVirtualInterfaceGroupId)

	createLocalGatewayRouteTableVirtualInterfaceGroupAssociationOut, err := client.CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociation(ctx, &awsec2.CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociationInput{
		LocalGatewayRouteTableId:            aws.String(routeTableID),
		LocalGatewayVirtualInterfaceGroupId: aws.String(groupID),
	})
	if err != nil {
		t.Fatalf("create local gateway route table virtual interface group association: %v", err)
	}
	if createLocalGatewayRouteTableVirtualInterfaceGroupAssociationOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociation == nil ||
		createLocalGatewayRouteTableVirtualInterfaceGroupAssociationOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociation.LocalGatewayRouteTableVirtualInterfaceGroupAssociationId == nil {
		t.Fatalf("expected created local gateway route table virtual interface group association")
	}
	vifAssociationID := aws.ToString(createLocalGatewayRouteTableVirtualInterfaceGroupAssociationOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociation.LocalGatewayRouteTableVirtualInterfaceGroupAssociationId)

	createVpcOut, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{CidrBlock: aws.String("10.119.0.0/16")})
	if err != nil {
		t.Fatalf("create vpc: %v", err)
	}
	if createVpcOut.Vpc == nil || createVpcOut.Vpc.VpcId == nil {
		t.Fatalf("expected created vpc")
	}
	vpcID := aws.ToString(createVpcOut.Vpc.VpcId)

	createLocalGatewayRouteTableVpcAssociationOut, err := client.CreateLocalGatewayRouteTableVpcAssociation(ctx, &awsec2.CreateLocalGatewayRouteTableVpcAssociationInput{
		LocalGatewayRouteTableId: aws.String(routeTableID),
		VpcId:                    aws.String(vpcID),
	})
	if err != nil {
		t.Fatalf("create local gateway route table vpc association: %v", err)
	}
	if createLocalGatewayRouteTableVpcAssociationOut.LocalGatewayRouteTableVpcAssociation == nil ||
		createLocalGatewayRouteTableVpcAssociationOut.LocalGatewayRouteTableVpcAssociation.LocalGatewayRouteTableVpcAssociationId == nil {
		t.Fatalf("expected created local gateway route table vpc association")
	}
	vpcAssociationID := aws.ToString(createLocalGatewayRouteTableVpcAssociationOut.LocalGatewayRouteTableVpcAssociation.LocalGatewayRouteTableVpcAssociationId)

	createLocalGatewayVirtualInterfaceOut, err := client.CreateLocalGatewayVirtualInterface(ctx, &awsec2.CreateLocalGatewayVirtualInterfaceInput{
		LocalAddress:                        aws.String("169.254.119.1"),
		LocalGatewayVirtualInterfaceGroupId: aws.String(groupID),
		OutpostLagId:                        aws.String("lag-119"),
		PeerAddress:                         aws.String("169.254.119.2"),
		Vlan:                                aws.Int32(119),
	})
	if err != nil {
		t.Fatalf("create local gateway virtual interface: %v", err)
	}
	if createLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface == nil || createLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface.LocalGatewayVirtualInterfaceId == nil {
		t.Fatalf("expected created local gateway virtual interface")
	}
	virtualInterfaceID := aws.ToString(createLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface.LocalGatewayVirtualInterfaceId)

	createVolumeOut, err := client.CreateVolume(ctx, &awsec2.CreateVolumeInput{
		AvailabilityZone: aws.String("us-east-1a"),
		Size:             aws.Int32(8),
	})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	if createVolumeOut.VolumeId == nil {
		t.Fatalf("expected created volume")
	}

	createSnapshotOut, err := client.CreateSnapshot(ctx, &awsec2.CreateSnapshotInput{VolumeId: createVolumeOut.VolumeId})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if createSnapshotOut.SnapshotId == nil {
		t.Fatalf("expected created snapshot")
	}
	snapshotID := aws.ToString(createSnapshotOut.SnapshotId)

	allocateHostsOut, err := client.AllocateHosts(ctx, &awsec2.AllocateHostsInput{
		AvailabilityZone: aws.String("us-east-1a"),
		InstanceFamily:   aws.String("m5"),
		Quantity:         aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("allocate hosts: %v", err)
	}
	if len(allocateHostsOut.HostIds) != 1 {
		t.Fatalf("expected 1 host id, got %d", len(allocateHostsOut.HostIds))
	}
	hostID := allocateHostsOut.HostIds[0]

	runInstancesOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-stage119"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runInstancesOut.Instances) == 0 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	createMacSystemIntegrityProtectionModificationTaskOut, err := client.CreateMacSystemIntegrityProtectionModificationTask(ctx, &awsec2.CreateMacSystemIntegrityProtectionModificationTaskInput{
		InstanceId:                         aws.String(instanceID),
		MacSystemIntegrityProtectionStatus: awsec2types.MacSystemIntegrityProtectionSettingStatusEnabled,
	})
	if err != nil {
		t.Fatalf("create mac system integrity protection modification task: %v", err)
	}
	if createMacSystemIntegrityProtectionModificationTaskOut.MacModificationTask == nil || createMacSystemIntegrityProtectionModificationTaskOut.MacModificationTask.MacModificationTaskId == nil {
		t.Fatalf("expected created mac modification task")
	}
	macModificationTaskID := aws.ToString(createMacSystemIntegrityProtectionModificationTaskOut.MacModificationTask.MacModificationTaskId)

	describeLaunchTemplatesOut, err := client.DescribeLaunchTemplates(ctx, &awsec2.DescribeLaunchTemplatesInput{
		LaunchTemplateIds: []string{launchTemplateID},
		MaxResults:        aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe launch templates: %v", err)
	}
	if len(describeLaunchTemplatesOut.LaunchTemplates) != 1 || aws.ToString(describeLaunchTemplatesOut.LaunchTemplates[0].LaunchTemplateId) != launchTemplateID {
		t.Fatalf("unexpected launch templates output: %#v", describeLaunchTemplatesOut.LaunchTemplates)
	}

	describeLocalGatewayRouteTableVirtualInterfaceGroupAssociationsOut, err := client.DescribeLocalGatewayRouteTableVirtualInterfaceGroupAssociations(ctx, &awsec2.DescribeLocalGatewayRouteTableVirtualInterfaceGroupAssociationsInput{
		LocalGatewayRouteTableVirtualInterfaceGroupAssociationIds: []string{vifAssociationID},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe local gateway route table virtual interface group associations: %v", err)
	}
	if len(describeLocalGatewayRouteTableVirtualInterfaceGroupAssociationsOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociations) != 1 ||
		aws.ToString(describeLocalGatewayRouteTableVirtualInterfaceGroupAssociationsOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociations[0].LocalGatewayRouteTableVirtualInterfaceGroupAssociationId) != vifAssociationID {
		t.Fatalf("unexpected local gateway route table virtual interface group associations output: %#v", describeLocalGatewayRouteTableVirtualInterfaceGroupAssociationsOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociations)
	}

	describeLocalGatewayRouteTableVpcAssociationsOut, err := client.DescribeLocalGatewayRouteTableVpcAssociations(ctx, &awsec2.DescribeLocalGatewayRouteTableVpcAssociationsInput{
		LocalGatewayRouteTableVpcAssociationIds: []string{vpcAssociationID},
		MaxResults:                              aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe local gateway route table vpc associations: %v", err)
	}
	if len(describeLocalGatewayRouteTableVpcAssociationsOut.LocalGatewayRouteTableVpcAssociations) != 1 ||
		aws.ToString(describeLocalGatewayRouteTableVpcAssociationsOut.LocalGatewayRouteTableVpcAssociations[0].LocalGatewayRouteTableVpcAssociationId) != vpcAssociationID {
		t.Fatalf("unexpected local gateway route table vpc associations output: %#v", describeLocalGatewayRouteTableVpcAssociationsOut.LocalGatewayRouteTableVpcAssociations)
	}

	describeLocalGatewayRouteTablesOut, err := client.DescribeLocalGatewayRouteTables(ctx, &awsec2.DescribeLocalGatewayRouteTablesInput{
		LocalGatewayRouteTableIds: []string{routeTableID},
		MaxResults:                aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe local gateway route tables: %v", err)
	}
	if len(describeLocalGatewayRouteTablesOut.LocalGatewayRouteTables) != 1 || aws.ToString(describeLocalGatewayRouteTablesOut.LocalGatewayRouteTables[0].LocalGatewayRouteTableId) != routeTableID {
		t.Fatalf("unexpected local gateway route tables output: %#v", describeLocalGatewayRouteTablesOut.LocalGatewayRouteTables)
	}

	describeLocalGatewayVirtualInterfaceGroupsOut, err := client.DescribeLocalGatewayVirtualInterfaceGroups(ctx, &awsec2.DescribeLocalGatewayVirtualInterfaceGroupsInput{
		LocalGatewayVirtualInterfaceGroupIds: []string{groupID},
		MaxResults:                           aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe local gateway virtual interface groups: %v", err)
	}
	if len(describeLocalGatewayVirtualInterfaceGroupsOut.LocalGatewayVirtualInterfaceGroups) != 1 ||
		aws.ToString(describeLocalGatewayVirtualInterfaceGroupsOut.LocalGatewayVirtualInterfaceGroups[0].LocalGatewayVirtualInterfaceGroupId) != groupID {
		t.Fatalf("unexpected local gateway virtual interface groups output: %#v", describeLocalGatewayVirtualInterfaceGroupsOut.LocalGatewayVirtualInterfaceGroups)
	}

	describeLocalGatewayVirtualInterfacesOut, err := client.DescribeLocalGatewayVirtualInterfaces(ctx, &awsec2.DescribeLocalGatewayVirtualInterfacesInput{
		LocalGatewayVirtualInterfaceIds: []string{virtualInterfaceID},
		MaxResults:                      aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe local gateway virtual interfaces: %v", err)
	}
	if len(describeLocalGatewayVirtualInterfacesOut.LocalGatewayVirtualInterfaces) != 1 ||
		aws.ToString(describeLocalGatewayVirtualInterfacesOut.LocalGatewayVirtualInterfaces[0].LocalGatewayVirtualInterfaceId) != virtualInterfaceID {
		t.Fatalf("unexpected local gateway virtual interfaces output: %#v", describeLocalGatewayVirtualInterfacesOut.LocalGatewayVirtualInterfaces)
	}

	describeLocalGatewaysOut, err := client.DescribeLocalGateways(ctx, &awsec2.DescribeLocalGatewaysInput{
		LocalGatewayIds: []string{localGatewayID},
		MaxResults:      aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe local gateways: %v", err)
	}
	if len(describeLocalGatewaysOut.LocalGateways) != 1 || aws.ToString(describeLocalGatewaysOut.LocalGateways[0].LocalGatewayId) != localGatewayID {
		t.Fatalf("unexpected local gateways output: %#v", describeLocalGatewaysOut.LocalGateways)
	}

	describeLockedSnapshotsOut, err := client.DescribeLockedSnapshots(ctx, &awsec2.DescribeLockedSnapshotsInput{
		SnapshotIds: []string{snapshotID},
		MaxResults:  aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe locked snapshots: %v", err)
	}
	if len(describeLockedSnapshotsOut.Snapshots) != 1 || aws.ToString(describeLockedSnapshotsOut.Snapshots[0].SnapshotId) != snapshotID {
		t.Fatalf("unexpected locked snapshots output: %#v", describeLockedSnapshotsOut.Snapshots)
	}

	describeMacHostsOut, err := client.DescribeMacHosts(ctx, &awsec2.DescribeMacHostsInput{
		HostIds:    []string{hostID},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe mac hosts: %v", err)
	}
	if len(describeMacHostsOut.MacHosts) != 1 || aws.ToString(describeMacHostsOut.MacHosts[0].HostId) != hostID {
		t.Fatalf("unexpected mac hosts output: %#v", describeMacHostsOut.MacHosts)
	}

	describeMacModificationTasksOut, err := client.DescribeMacModificationTasks(ctx, &awsec2.DescribeMacModificationTasksInput{
		MacModificationTaskIds: []string{macModificationTaskID},
		MaxResults:             aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe mac modification tasks: %v", err)
	}
	if len(describeMacModificationTasksOut.MacModificationTasks) != 1 ||
		aws.ToString(describeMacModificationTasksOut.MacModificationTasks[0].MacModificationTaskId) != macModificationTaskID {
		t.Fatalf("unexpected mac modification tasks output: %#v", describeMacModificationTasksOut.MacModificationTasks)
	}
}

func TestEC2Stage119ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeLaunchTemplates",
		"DescribeLocalGatewayRouteTableVirtualInterfaceGroupAssociations",
		"DescribeLocalGatewayRouteTableVpcAssociations",
		"DescribeLocalGatewayRouteTables",
		"DescribeLocalGatewayVirtualInterfaceGroups",
		"DescribeLocalGatewayVirtualInterfaces",
		"DescribeLocalGateways",
		"DescribeLockedSnapshots",
		"DescribeMacHosts",
		"DescribeMacModificationTasks",
	}

	paramsByAction := map[string]map[string]string{
		"DescribeLaunchTemplates": {
			"LaunchTemplateId.1": "lt-0000000119",
			"MaxResults":         "10",
		},
		"DescribeLocalGatewayRouteTableVirtualInterfaceGroupAssociations": {
			"LocalGatewayRouteTableVirtualInterfaceGroupAssociationId.1": "lgw-vif-assoc-0000000119",
			"MaxResults": "10",
		},
		"DescribeLocalGatewayRouteTableVpcAssociations": {
			"LocalGatewayRouteTableVpcAssociationId.1": "lgw-vpc-assoc-0000000119",
			"MaxResults": "10",
		},
		"DescribeLocalGatewayRouteTables": {
			"LocalGatewayRouteTableId.1": "lgw-rtb-0000000119",
			"MaxResults":                 "10",
		},
		"DescribeLocalGatewayVirtualInterfaceGroups": {
			"LocalGatewayVirtualInterfaceGroupId.1": "lgw-vifgrp-0000000119",
			"MaxResults":                            "10",
		},
		"DescribeLocalGatewayVirtualInterfaces": {
			"LocalGatewayVirtualInterfaceId.1": "lgw-vif-0000000119",
			"MaxResults":                       "10",
		},
		"DescribeLocalGateways": {
			"LocalGatewayId.1": "lgw-0000000119",
			"MaxResults":       "10",
		},
		"DescribeLockedSnapshots": {
			"SnapshotId.1": "snap-0000000119",
			"MaxResults":   "10",
		},
		"DescribeMacHosts": {
			"HostId.1":   "h-0000000119",
			"MaxResults": "10",
		},
		"DescribeMacModificationTasks": {
			"MacModificationTaskId.1": "mmt-0000000119",
			"MaxResults":              "10",
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
