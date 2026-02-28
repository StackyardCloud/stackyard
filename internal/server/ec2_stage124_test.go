package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage124SDKLifecycle(t *testing.T) {
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

	associateVpcCidrBlockOut, err := client.AssociateVpcCidrBlock(ctx, &awsec2.AssociateVpcCidrBlockInput{
		VpcId:    aws.String("vpc-00000001"),
		Ipv6Pool: aws.String("ipv6pool-stage124"),
	})
	if err != nil || associateVpcCidrBlockOut.Ipv6CidrBlockAssociation == nil {
		t.Fatalf("associate vpc cidr block: %v", err)
	}

	getAssociatedIpv6PoolCidrsOut, err := client.GetAssociatedIpv6PoolCidrs(ctx, &awsec2.GetAssociatedIpv6PoolCidrsInput{
		PoolId:     aws.String("ipv6pool-stage124"),
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get associated ipv6 pool cidrs: %v", err)
	}
	if len(getAssociatedIpv6PoolCidrsOut.Ipv6CidrAssociations) == 0 {
		t.Fatalf("expected ipv6 cidr associations")
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

	getCapacityReservationUsageOut, err := client.GetCapacityReservationUsage(ctx, &awsec2.GetCapacityReservationUsageInput{
		CapacityReservationId: aws.String(capacityReservationID),
		MaxResults:            aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get capacity reservation usage: %v", err)
	}
	if aws.ToString(getCapacityReservationUsageOut.CapacityReservationId) != capacityReservationID {
		t.Fatalf("unexpected get capacity reservation usage output: %#v", getCapacityReservationUsageOut)
	}

	createCoipPoolOut, err := client.CreateCoipPool(ctx, &awsec2.CreateCoipPoolInput{
		LocalGatewayRouteTableId: aws.String("lgw-rtb-stage124"),
	})
	if err != nil || createCoipPoolOut.CoipPool == nil || createCoipPoolOut.CoipPool.PoolId == nil {
		t.Fatalf("create coip pool: %v", err)
	}
	coipPoolID := aws.ToString(createCoipPoolOut.CoipPool.PoolId)

	_, err = client.CreateCoipCidr(ctx, &awsec2.CreateCoipCidrInput{
		Cidr:       aws.String("192.0.2.0/24"),
		CoipPoolId: aws.String(coipPoolID),
	})
	if err != nil {
		t.Fatalf("create coip cidr: %v", err)
	}

	getCoipPoolUsageOut, err := client.GetCoipPoolUsage(ctx, &awsec2.GetCoipPoolUsageInput{
		PoolId:     aws.String(coipPoolID),
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get coip pool usage: %v", err)
	}
	if aws.ToString(getCoipPoolUsageOut.CoipPoolId) != coipPoolID {
		t.Fatalf("unexpected get coip pool usage output: %#v", getCoipPoolUsageOut)
	}

	getDefaultCreditSpecificationOut, err := client.GetDefaultCreditSpecification(ctx, &awsec2.GetDefaultCreditSpecificationInput{
		InstanceFamily: awsec2types.UnlimitedSupportedInstanceFamilyT3,
	})
	if err != nil {
		t.Fatalf("get default credit specification: %v", err)
	}
	if getDefaultCreditSpecificationOut.InstanceFamilyCreditSpecification == nil ||
		string(getDefaultCreditSpecificationOut.InstanceFamilyCreditSpecification.InstanceFamily) != "t3" {
		t.Fatalf("unexpected get default credit specification output: %#v", getDefaultCreditSpecificationOut.InstanceFamilyCreditSpecification)
	}

	createFlowLogsOut, err := client.CreateFlowLogs(ctx, &awsec2.CreateFlowLogsInput{
		ResourceIds:  []string{"vpc-00000001"},
		ResourceType: awsec2types.FlowLogsResourceTypeVpc,
		TrafficType:  awsec2types.TrafficTypeAll,
	})
	if err != nil || len(createFlowLogsOut.FlowLogIds) == 0 {
		t.Fatalf("create flow logs: %v", err)
	}
	flowLogID := createFlowLogsOut.FlowLogIds[0]

	getFlowLogsIntegrationTemplateOut, err := client.GetFlowLogsIntegrationTemplate(ctx, &awsec2.GetFlowLogsIntegrationTemplateInput{
		ConfigDeliveryS3DestinationArn: aws.String("arn:aws:s3:::stage124-bucket/integrations"),
		FlowLogId:                      aws.String(flowLogID),
		IntegrateServices: &awsec2types.IntegrateServices{
			AthenaIntegrations: []awsec2types.AthenaIntegration{
				{
					IntegrationResultS3DestinationArn: aws.String("arn:aws:s3:::stage124-bucket/athena"),
					PartitionLoadFrequency:            awsec2types.PartitionLoadFrequencyDaily,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("get flow logs integration template: %v", err)
	}
	if aws.ToString(getFlowLogsIntegrationTemplateOut.Result) == "" || !strings.Contains(aws.ToString(getFlowLogsIntegrationTemplateOut.Result), flowLogID) {
		t.Fatalf("unexpected get flow logs integration template output: %#v", getFlowLogsIntegrationTemplateOut)
	}

	getGroupsForCapacityReservationOut, err := client.GetGroupsForCapacityReservation(ctx, &awsec2.GetGroupsForCapacityReservationInput{
		CapacityReservationId: aws.String(capacityReservationID),
		MaxResults:            aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get groups for capacity reservation: %v", err)
	}
	if len(getGroupsForCapacityReservationOut.CapacityReservationGroups) == 0 {
		t.Fatalf("expected capacity reservation groups")
	}

	allocateHostsOut, err := client.AllocateHosts(ctx, &awsec2.AllocateHostsInput{
		AvailabilityZone: aws.String("us-east-1a"),
		InstanceType:     aws.String("m5.large"),
		Quantity:         aws.Int32(1),
	})
	if err != nil || len(allocateHostsOut.HostIds) == 0 {
		t.Fatalf("allocate hosts: %v", err)
	}
	hostID := allocateHostsOut.HostIds[0]

	describeHostReservationOfferingsOut, err := client.DescribeHostReservationOfferings(ctx, &awsec2.DescribeHostReservationOfferingsInput{
		MaxResults: aws.Int32(10),
	})
	if err != nil || len(describeHostReservationOfferingsOut.OfferingSet) == 0 || describeHostReservationOfferingsOut.OfferingSet[0].OfferingId == nil {
		t.Fatalf("describe host reservation offerings: %v", err)
	}
	offeringID := aws.ToString(describeHostReservationOfferingsOut.OfferingSet[0].OfferingId)

	getHostReservationPurchasePreviewOut, err := client.GetHostReservationPurchasePreview(ctx, &awsec2.GetHostReservationPurchasePreviewInput{
		HostIdSet:  []string{hostID},
		OfferingId: aws.String(offeringID),
	})
	if err != nil {
		t.Fatalf("get host reservation purchase preview: %v", err)
	}
	if len(getHostReservationPurchasePreviewOut.Purchase) == 0 {
		t.Fatalf("expected host reservation purchase preview entries")
	}

	getInstanceMetadataDefaultsOut, err := client.GetInstanceMetadataDefaults(ctx, &awsec2.GetInstanceMetadataDefaultsInput{})
	if err != nil {
		t.Fatalf("get instance metadata defaults: %v", err)
	}
	if getInstanceMetadataDefaultsOut.AccountLevel == nil || string(getInstanceMetadataDefaultsOut.AccountLevel.HttpEndpoint) == "" {
		t.Fatalf("unexpected instance metadata defaults output: %#v", getInstanceMetadataDefaultsOut.AccountLevel)
	}

	runInstancesOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-stage124"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runInstancesOut.Instances) == 0 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	getInstanceTpmEkPubOut, err := client.GetInstanceTpmEkPub(ctx, &awsec2.GetInstanceTpmEkPubInput{
		InstanceId: aws.String(instanceID),
		KeyFormat:  awsec2types.EkPubKeyFormatDer,
		KeyType:    awsec2types.EkPubKeyTypeRsa2048,
	})
	if err != nil {
		t.Fatalf("get instance tpm ek pub: %v", err)
	}
	if aws.ToString(getInstanceTpmEkPubOut.InstanceId) != instanceID || aws.ToString(getInstanceTpmEkPubOut.KeyValue) == "" {
		t.Fatalf("unexpected get instance tpm ek pub output: %#v", getInstanceTpmEkPubOut)
	}

	getInstanceTypesFromInstanceRequirementsOut, err := client.GetInstanceTypesFromInstanceRequirements(ctx, &awsec2.GetInstanceTypesFromInstanceRequirementsInput{
		ArchitectureTypes:   []awsec2types.ArchitectureType{awsec2types.ArchitectureTypeX8664},
		VirtualizationTypes: []awsec2types.VirtualizationType{awsec2types.VirtualizationTypeHvm},
		InstanceRequirements: &awsec2types.InstanceRequirementsRequest{
			VCpuCount: &awsec2types.VCpuCountRangeRequest{Min: aws.Int32(1)},
			MemoryMiB: &awsec2types.MemoryMiBRequest{Min: aws.Int32(1024)},
		},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get instance types from instance requirements: %v", err)
	}
	if len(getInstanceTypesFromInstanceRequirementsOut.InstanceTypes) == 0 {
		t.Fatalf("expected instance types from requirements")
	}
}

func TestEC2Stage124ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"GetAssociatedIpv6PoolCidrs",
		"GetCapacityReservationUsage",
		"GetCoipPoolUsage",
		"GetDefaultCreditSpecification",
		"GetFlowLogsIntegrationTemplate",
		"GetGroupsForCapacityReservation",
		"GetHostReservationPurchasePreview",
		"GetInstanceMetadataDefaults",
		"GetInstanceTpmEkPub",
		"GetInstanceTypesFromInstanceRequirements",
	}

	paramsByAction := map[string]map[string]string{
		"GetAssociatedIpv6PoolCidrs": {
			"PoolId": "ipv6pool-stage124",
		},
		"GetCapacityReservationUsage": {
			"CapacityReservationId": "cr-0000000124",
		},
		"GetCoipPoolUsage": {
			"PoolId": "coip-pool-0000000124",
		},
		"GetDefaultCreditSpecification": {
			"InstanceFamily": "t3",
		},
		"GetFlowLogsIntegrationTemplate": {
			"ConfigDeliveryS3DestinationArn": "arn:aws:s3:::stage124-bucket/integrations",
			"FlowLogId":                      "fl-0000000124",
			"IntegrateService.AthenaIntegrations.1.IntegrationResultS3DestinationArn": "arn:aws:s3:::stage124-bucket/athena",
			"IntegrateService.AthenaIntegrations.1.PartitionLoadFrequency":            "daily",
		},
		"GetGroupsForCapacityReservation": {
			"CapacityReservationId": "cr-0000000124",
		},
		"GetHostReservationPurchasePreview": {
			"HostIdSet.1": "h-0000000124",
			"OfferingId":  "hro-001",
		},
		"GetInstanceMetadataDefaults": {},
		"GetInstanceTpmEkPub": {
			"InstanceId": "i-0000000124",
			"KeyFormat":  "der",
			"KeyType":    "rsa-2048",
		},
		"GetInstanceTypesFromInstanceRequirements": {
			"ArchitectureType.1":                 "x86_64",
			"VirtualizationType.1":               "hvm",
			"InstanceRequirements.VCpuCount.Min": "1",
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
