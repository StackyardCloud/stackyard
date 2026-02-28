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

func TestEC2Stage123SDKLifecycle(t *testing.T) {
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

	createVolumeOut, err := client.CreateVolume(ctx, &awsec2.CreateVolumeInput{
		AvailabilityZone: aws.String("us-east-1a"),
		Size:             aws.Int32(8),
	})
	if err != nil || createVolumeOut.VolumeId == nil {
		t.Fatalf("create volume: %v", err)
	}
	volumeID := aws.ToString(createVolumeOut.VolumeId)

	_, err = client.ModifyVolume(ctx, &awsec2.ModifyVolumeInput{
		VolumeId: aws.String(volumeID),
		Size:     aws.Int32(9),
	})
	if err != nil {
		t.Fatalf("modify volume: %v", err)
	}

	describeVolumeAttributeOut, err := client.DescribeVolumeAttribute(ctx, &awsec2.DescribeVolumeAttributeInput{
		Attribute: awsec2types.VolumeAttributeNameAutoEnableIO,
		VolumeId:  aws.String(volumeID),
	})
	if err != nil {
		t.Fatalf("describe volume attribute: %v", err)
	}
	if aws.ToString(describeVolumeAttributeOut.VolumeId) != volumeID {
		t.Fatalf("unexpected describe volume attribute output: %#v", describeVolumeAttributeOut)
	}

	describeVolumeStatusOut, err := client.DescribeVolumeStatus(ctx, &awsec2.DescribeVolumeStatusInput{
		VolumeIds:  []string{volumeID},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe volume status: %v", err)
	}
	if len(describeVolumeStatusOut.VolumeStatuses) != 1 || aws.ToString(describeVolumeStatusOut.VolumeStatuses[0].VolumeId) != volumeID {
		t.Fatalf("unexpected describe volume status output: %#v", describeVolumeStatusOut.VolumeStatuses)
	}

	describeVolumesModificationsOut, err := client.DescribeVolumesModifications(ctx, &awsec2.DescribeVolumesModificationsInput{
		VolumeIds:  []string{volumeID},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe volumes modifications: %v", err)
	}
	if len(describeVolumesModificationsOut.VolumesModifications) != 1 || aws.ToString(describeVolumesModificationsOut.VolumesModifications[0].VolumeId) != volumeID {
		t.Fatalf("unexpected describe volumes modifications output: %#v", describeVolumesModificationsOut.VolumesModifications)
	}

	createCapacityReservationOut, err := client.CreateCapacityReservation(ctx, &awsec2.CreateCapacityReservationInput{
		AvailabilityZone: aws.String("us-east-1a"),
		InstanceCount:    aws.Int32(1),
		InstancePlatform: awsec2types.CapacityReservationInstancePlatformLinuxUnix,
		InstanceType:     aws.String("m5.large"),
	})
	if err != nil || createCapacityReservationOut.CapacityReservation == nil || createCapacityReservationOut.CapacityReservation.CapacityReservationId == nil {
		t.Fatalf("create capacity reservation: %v", err)
	}
	capacityReservationID := aws.ToString(createCapacityReservationOut.CapacityReservation.CapacityReservationId)
	unusedReservationBillingOwnerID := "210987654321"

	_, err = client.AssociateCapacityReservationBillingOwner(ctx, &awsec2.AssociateCapacityReservationBillingOwnerInput{
		CapacityReservationId:           aws.String(capacityReservationID),
		UnusedReservationBillingOwnerId: aws.String(unusedReservationBillingOwnerID),
	})
	if err != nil {
		t.Fatalf("associate capacity reservation billing owner: %v", err)
	}

	disassociateCapacityReservationBillingOwnerOut, err := client.DisassociateCapacityReservationBillingOwner(ctx, &awsec2.DisassociateCapacityReservationBillingOwnerInput{
		CapacityReservationId:           aws.String(capacityReservationID),
		UnusedReservationBillingOwnerId: aws.String(unusedReservationBillingOwnerID),
	})
	if err != nil {
		t.Fatalf("disassociate capacity reservation billing owner: %v", err)
	}
	if !aws.ToBool(disassociateCapacityReservationBillingOwnerOut.Return) {
		t.Fatalf("expected successful disassociate capacity reservation billing owner")
	}

	certificateARN := "arn:aws:acm:us-east-1:123456789012:certificate/stage123"
	roleARN := "arn:aws:iam::123456789012:role/stage123"
	_, err = client.AssociateEnclaveCertificateIamRole(ctx, &awsec2.AssociateEnclaveCertificateIamRoleInput{
		CertificateArn: aws.String(certificateARN),
		RoleArn:        aws.String(roleARN),
	})
	if err != nil {
		t.Fatalf("associate enclave certificate iam role: %v", err)
	}

	getAssociatedBeforeOut, err := client.GetAssociatedEnclaveCertificateIamRoles(ctx, &awsec2.GetAssociatedEnclaveCertificateIamRolesInput{
		CertificateArn: aws.String(certificateARN),
	})
	if err != nil {
		t.Fatalf("get associated enclave certificate iam roles (before): %v", err)
	}
	if len(getAssociatedBeforeOut.AssociatedRoles) != 1 || aws.ToString(getAssociatedBeforeOut.AssociatedRoles[0].AssociatedRoleArn) != roleARN {
		t.Fatalf("unexpected associated roles output before disassociate: %#v", getAssociatedBeforeOut.AssociatedRoles)
	}

	disassociateEnclaveCertificateIamRoleOut, err := client.DisassociateEnclaveCertificateIamRole(ctx, &awsec2.DisassociateEnclaveCertificateIamRoleInput{
		CertificateArn: aws.String(certificateARN),
		RoleArn:        aws.String(roleARN),
	})
	if err != nil {
		t.Fatalf("disassociate enclave certificate iam role: %v", err)
	}
	if !aws.ToBool(disassociateEnclaveCertificateIamRoleOut.Return) {
		t.Fatalf("expected successful disassociate enclave certificate iam role")
	}

	getAssociatedAfterOut, err := client.GetAssociatedEnclaveCertificateIamRoles(ctx, &awsec2.GetAssociatedEnclaveCertificateIamRolesInput{
		CertificateArn: aws.String(certificateARN),
	})
	if err != nil {
		t.Fatalf("get associated enclave certificate iam roles (after): %v", err)
	}
	if len(getAssociatedAfterOut.AssociatedRoles) != 0 {
		t.Fatalf("expected no associated roles after disassociate, got %#v", getAssociatedAfterOut.AssociatedRoles)
	}

	runInstancesOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-stage123"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runInstancesOut.Instances) == 0 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	createInstanceEventWindowOut, err := client.CreateInstanceEventWindow(ctx, &awsec2.CreateInstanceEventWindowInput{
		Name:           aws.String("stage123-window"),
		CronExpression: aws.String("cron(0 10 ? * SUN *)"),
		TimeRanges: []awsec2types.InstanceEventWindowTimeRangeRequest{
			{
				StartWeekDay: awsec2types.WeekDaySunday,
				StartHour:    aws.Int32(10),
				EndWeekDay:   awsec2types.WeekDaySunday,
				EndHour:      aws.Int32(11),
			},
		},
	})
	if err != nil || createInstanceEventWindowOut.InstanceEventWindow == nil || createInstanceEventWindowOut.InstanceEventWindow.InstanceEventWindowId == nil {
		t.Fatalf("create instance event window: %v", err)
	}
	instanceEventWindowID := aws.ToString(createInstanceEventWindowOut.InstanceEventWindow.InstanceEventWindowId)

	_, err = client.AssociateInstanceEventWindow(ctx, &awsec2.AssociateInstanceEventWindowInput{
		InstanceEventWindowId: aws.String(instanceEventWindowID),
		AssociationTarget: &awsec2types.InstanceEventWindowAssociationRequest{
			InstanceIds: []string{instanceID},
		},
	})
	if err != nil {
		t.Fatalf("associate instance event window: %v", err)
	}

	disassociateInstanceEventWindowOut, err := client.DisassociateInstanceEventWindow(ctx, &awsec2.DisassociateInstanceEventWindowInput{
		InstanceEventWindowId: aws.String(instanceEventWindowID),
		AssociationTarget: &awsec2types.InstanceEventWindowDisassociationRequest{
			InstanceIds: []string{instanceID},
		},
	})
	if err != nil {
		t.Fatalf("disassociate instance event window: %v", err)
	}
	if disassociateInstanceEventWindowOut.InstanceEventWindow == nil || aws.ToString(disassociateInstanceEventWindowOut.InstanceEventWindow.InstanceEventWindowId) != instanceEventWindowID {
		t.Fatalf("unexpected disassociate instance event window output: %#v", disassociateInstanceEventWindowOut.InstanceEventWindow)
	}

	createIpamOut, err := client.CreateIpam(ctx, &awsec2.CreateIpamInput{
		OperatingRegions: []awsec2types.AddIpamOperatingRegion{
			{RegionName: aws.String("us-east-1")},
		},
		Tier: awsec2types.IpamTierFree,
	})
	if err != nil || createIpamOut.Ipam == nil || createIpamOut.Ipam.IpamId == nil {
		t.Fatalf("create ipam: %v", err)
	}
	ipamID := aws.ToString(createIpamOut.Ipam.IpamId)

	createIpamResourceDiscoveryOut, err := client.CreateIpamResourceDiscovery(ctx, &awsec2.CreateIpamResourceDiscoveryInput{
		OperatingRegions: []awsec2types.AddIpamOperatingRegion{{RegionName: aws.String("us-east-1")}},
	})
	if err != nil || createIpamResourceDiscoveryOut.IpamResourceDiscovery == nil || createIpamResourceDiscoveryOut.IpamResourceDiscovery.IpamResourceDiscoveryId == nil {
		t.Fatalf("create ipam resource discovery: %v", err)
	}
	ipamResourceDiscoveryID := aws.ToString(createIpamResourceDiscoveryOut.IpamResourceDiscovery.IpamResourceDiscoveryId)

	associateIpamResourceDiscoveryOut, err := client.AssociateIpamResourceDiscovery(ctx, &awsec2.AssociateIpamResourceDiscoveryInput{
		IpamId:                  aws.String(ipamID),
		IpamResourceDiscoveryId: aws.String(ipamResourceDiscoveryID),
	})
	if err != nil || associateIpamResourceDiscoveryOut.IpamResourceDiscoveryAssociation == nil || associateIpamResourceDiscoveryOut.IpamResourceDiscoveryAssociation.IpamResourceDiscoveryAssociationId == nil {
		t.Fatalf("associate ipam resource discovery: %v", err)
	}
	ipamResourceDiscoveryAssociationID := aws.ToString(associateIpamResourceDiscoveryOut.IpamResourceDiscoveryAssociation.IpamResourceDiscoveryAssociationId)

	disassociateIpamResourceDiscoveryOut, err := client.DisassociateIpamResourceDiscovery(ctx, &awsec2.DisassociateIpamResourceDiscoveryInput{
		IpamResourceDiscoveryAssociationId: aws.String(ipamResourceDiscoveryAssociationID),
	})
	if err != nil {
		t.Fatalf("disassociate ipam resource discovery: %v", err)
	}
	if disassociateIpamResourceDiscoveryOut.IpamResourceDiscoveryAssociation == nil ||
		aws.ToString(disassociateIpamResourceDiscoveryOut.IpamResourceDiscoveryAssociation.IpamResourceDiscoveryAssociationId) != ipamResourceDiscoveryAssociationID {
		t.Fatalf("unexpected disassociate ipam resource discovery output: %#v", disassociateIpamResourceDiscoveryOut.IpamResourceDiscoveryAssociation)
	}

	_, err = client.AssociateIpamByoasn(ctx, &awsec2.AssociateIpamByoasnInput{
		Asn:  aws.String("64512"),
		Cidr: aws.String("198.51.100.0/24"),
	})
	if err != nil {
		t.Fatalf("associate ipam byoasn: %v", err)
	}

	disassociateIpamByoasnOut, err := client.DisassociateIpamByoasn(ctx, &awsec2.DisassociateIpamByoasnInput{
		Asn:  aws.String("64512"),
		Cidr: aws.String("198.51.100.0/24"),
	})
	if err != nil {
		t.Fatalf("disassociate ipam byoasn: %v", err)
	}
	if disassociateIpamByoasnOut.AsnAssociation == nil || aws.ToString(disassociateIpamByoasnOut.AsnAssociation.Asn) != "64512" {
		t.Fatalf("unexpected disassociate ipam byoasn output: %#v", disassociateIpamByoasnOut.AsnAssociation)
	}

	createImageOut, err := client.CreateImage(ctx, &awsec2.CreateImageInput{
		InstanceId: aws.String(instanceID),
		Name:       aws.String("stage123-image"),
	})
	if err != nil || createImageOut.ImageId == nil {
		t.Fatalf("create image: %v", err)
	}
	imageID := aws.ToString(createImageOut.ImageId)

	exportImageOut, err := client.ExportImage(ctx, &awsec2.ExportImageInput{
		Description:     aws.String("stage123 export"),
		DiskImageFormat: awsec2types.DiskImageFormatVmdk,
		ImageId:         aws.String(imageID),
		RoleName:        aws.String("vmimport"),
		S3ExportLocation: &awsec2types.ExportTaskS3LocationRequest{
			S3Bucket: aws.String("stage123-export-bucket"),
			S3Prefix: aws.String("exports/"),
		},
	})
	if err != nil {
		t.Fatalf("export image: %v", err)
	}
	if aws.ToString(exportImageOut.ExportImageTaskId) == "" || aws.ToString(exportImageOut.ImageId) != imageID {
		t.Fatalf("unexpected export image output: %#v", exportImageOut)
	}
	if exportImageOut.S3ExportLocation == nil || aws.ToString(exportImageOut.S3ExportLocation.S3Bucket) != "stage123-export-bucket" {
		t.Fatalf("unexpected export image S3 location output: %#v", exportImageOut.S3ExportLocation)
	}
}

func TestEC2Stage123ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeVolumeAttribute",
		"DescribeVolumeStatus",
		"DescribeVolumesModifications",
		"DisassociateCapacityReservationBillingOwner",
		"DisassociateEnclaveCertificateIamRole",
		"DisassociateInstanceEventWindow",
		"DisassociateIpamByoasn",
		"DisassociateIpamResourceDiscovery",
		"ExportImage",
		"GetAssociatedEnclaveCertificateIamRoles",
	}

	paramsByAction := map[string]map[string]string{
		"DescribeVolumeAttribute": {
			"Attribute": "autoEnableIO",
			"VolumeId":  "vol-0000000123",
		},
		"DescribeVolumeStatus": {
			"VolumeId.1": "vol-0000000123",
			"MaxResults": "10",
		},
		"DescribeVolumesModifications": {
			"VolumeId.1": "vol-0000000123",
			"MaxResults": "10",
		},
		"DisassociateCapacityReservationBillingOwner": {
			"CapacityReservationId":           "cr-0000000123",
			"UnusedReservationBillingOwnerId": "210987654321",
		},
		"DisassociateEnclaveCertificateIamRole": {
			"CertificateArn": "arn:aws:acm:us-east-1:123456789012:certificate/stage123",
			"RoleArn":        "arn:aws:iam::123456789012:role/stage123",
		},
		"DisassociateInstanceEventWindow": {
			"InstanceEventWindowId":          "iew-0000000123",
			"AssociationTarget.InstanceId.1": "i-0000000123",
		},
		"DisassociateIpamByoasn": {
			"Asn":  "64512",
			"Cidr": "198.51.100.0/24",
		},
		"DisassociateIpamResourceDiscovery": {
			"IpamResourceDiscoveryAssociationId": "ipam-rd-assoc-0000000123",
		},
		"ExportImage": {
			"DiskImageFormat":           "vmdk",
			"ImageId":                   "ami-0000000123",
			"S3ExportLocation.S3Bucket": "stage123-export-bucket",
			"S3ExportLocation.S3Prefix": "exports/",
			"RoleName":                  "vmimport",
		},
		"GetAssociatedEnclaveCertificateIamRoles": {
			"CertificateArn": "arn:aws:acm:us-east-1:123456789012:certificate/stage123",
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
