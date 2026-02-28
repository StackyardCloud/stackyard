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

func TestEC2Stage107SDKLifecycle(t *testing.T) {
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

	createCoipPoolOut, err := client.CreateCoipPool(ctx, &awsec2.CreateCoipPoolInput{
		LocalGatewayRouteTableId: aws.String("lgw-rtb-00000000000000107"),
	})
	if err != nil {
		t.Fatalf("create coip pool: %v", err)
	}
	if createCoipPoolOut.CoipPool == nil || !strings.HasPrefix(aws.ToString(createCoipPoolOut.CoipPool.PoolId), "coip-pool-") {
		t.Fatalf("unexpected coip pool output: %#v", createCoipPoolOut.CoipPool)
	}

	createDelegateTaskOut, err := client.CreateDelegateMacVolumeOwnershipTask(ctx, &awsec2.CreateDelegateMacVolumeOwnershipTaskInput{
		InstanceId:     aws.String("i-00000000000000107"),
		MacCredentials: aws.String("mac-credentials"),
	})
	if err != nil {
		t.Fatalf("create delegate mac volume ownership task: %v", err)
	}
	if createDelegateTaskOut.MacModificationTask == nil || !strings.HasPrefix(aws.ToString(createDelegateTaskOut.MacModificationTask.MacModificationTaskId), "mmt-") {
		t.Fatalf("unexpected mac modification task: %#v", createDelegateTaskOut.MacModificationTask)
	}

	createFleetOut, err := client.CreateFleet(ctx, &awsec2.CreateFleetInput{
		LaunchTemplateConfigs: []awsec2types.FleetLaunchTemplateConfigRequest{
			{
				LaunchTemplateSpecification: &awsec2types.FleetLaunchTemplateSpecificationRequest{
					LaunchTemplateId: aws.String("lt-00000000000000107"),
					Version:          aws.String("1"),
				},
			},
		},
		TargetCapacitySpecification: &awsec2types.TargetCapacitySpecificationRequest{
			DefaultTargetCapacityType: awsec2types.DefaultTargetCapacityTypeOnDemand,
			TotalTargetCapacity:       aws.Int32(1),
		},
	})
	if err != nil {
		t.Fatalf("create fleet: %v", err)
	}
	if !strings.HasPrefix(aws.ToString(createFleetOut.FleetId), "fleet-") {
		t.Fatalf("unexpected fleet id: %q", aws.ToString(createFleetOut.FleetId))
	}
	if len(createFleetOut.Instances) == 0 {
		t.Fatalf("expected at least one fleet instance")
	}

	createFlowLogsOut, err := client.CreateFlowLogs(ctx, &awsec2.CreateFlowLogsInput{
		ResourceIds:  []string{"vpc-00000001"},
		ResourceType: awsec2types.FlowLogsResourceTypeVpc,
		TrafficType:  awsec2types.TrafficTypeAll,
	})
	if err != nil {
		t.Fatalf("create flow logs: %v", err)
	}
	if len(createFlowLogsOut.FlowLogIds) != 1 || !strings.HasPrefix(createFlowLogsOut.FlowLogIds[0], "fl-") {
		t.Fatalf("unexpected flow log ids: %#v", createFlowLogsOut.FlowLogIds)
	}

	createFpgaImageOut, err := client.CreateFpgaImage(ctx, &awsec2.CreateFpgaImageInput{
		InputStorageLocation: &awsec2types.StorageLocation{
			Bucket: aws.String("stage107-bucket"),
			Key:    aws.String("stage107/input.xclbin"),
		},
	})
	if err != nil {
		t.Fatalf("create fpga image: %v", err)
	}
	if !strings.HasPrefix(aws.ToString(createFpgaImageOut.FpgaImageId), "afi-") {
		t.Fatalf("unexpected fpga image id: %q", aws.ToString(createFpgaImageOut.FpgaImageId))
	}
	if !strings.HasPrefix(aws.ToString(createFpgaImageOut.FpgaImageGlobalId), "agfi-") {
		t.Fatalf("unexpected fpga image global id: %q", aws.ToString(createFpgaImageOut.FpgaImageGlobalId))
	}

	createInstanceConnectEndpointOut, err := client.CreateInstanceConnectEndpoint(ctx, &awsec2.CreateInstanceConnectEndpointInput{
		SubnetId: aws.String("subnet-00000001"),
	})
	if err != nil {
		t.Fatalf("create instance connect endpoint: %v", err)
	}
	if createInstanceConnectEndpointOut.InstanceConnectEndpoint == nil {
		t.Fatalf("expected instance connect endpoint")
	}
	if !strings.HasPrefix(aws.ToString(createInstanceConnectEndpointOut.InstanceConnectEndpoint.InstanceConnectEndpointId), "eice-") {
		t.Fatalf("unexpected instance connect endpoint id: %q", aws.ToString(createInstanceConnectEndpointOut.InstanceConnectEndpoint.InstanceConnectEndpointId))
	}

	createInstanceEventWindowOut, err := client.CreateInstanceEventWindow(ctx, &awsec2.CreateInstanceEventWindowInput{
		Name:           aws.String("stage107-window"),
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
	if err != nil {
		t.Fatalf("create instance event window: %v", err)
	}
	if createInstanceEventWindowOut.InstanceEventWindow == nil {
		t.Fatalf("expected instance event window")
	}
	if !strings.HasPrefix(aws.ToString(createInstanceEventWindowOut.InstanceEventWindow.InstanceEventWindowId), "iew-") {
		t.Fatalf("unexpected instance event window id: %q", aws.ToString(createInstanceEventWindowOut.InstanceEventWindow.InstanceEventWindowId))
	}

	createInstanceExportTaskOut, err := client.CreateInstanceExportTask(ctx, &awsec2.CreateInstanceExportTaskInput{
		Description: aws.String("stage107-export"),
		ExportToS3Task: &awsec2types.ExportToS3TaskSpecification{
			ContainerFormat: awsec2types.ContainerFormatOva,
			DiskImageFormat: awsec2types.DiskImageFormatVmdk,
			S3Bucket:        aws.String("stage107-exports"),
			S3Prefix:        aws.String("exports"),
		},
		InstanceId:        aws.String("i-00000000000000107"),
		TargetEnvironment: awsec2types.ExportEnvironmentVmware,
	})
	if err != nil {
		t.Fatalf("create instance export task: %v", err)
	}
	if createInstanceExportTaskOut.ExportTask == nil || !strings.HasPrefix(aws.ToString(createInstanceExportTaskOut.ExportTask.ExportTaskId), "export-i-") {
		t.Fatalf("unexpected export task: %#v", createInstanceExportTaskOut.ExportTask)
	}

	createIpamOut, err := client.CreateIpam(ctx, &awsec2.CreateIpamInput{
		Description: aws.String("stage107-ipam"),
		OperatingRegions: []awsec2types.AddIpamOperatingRegion{
			{RegionName: aws.String("us-east-1")},
		},
	})
	if err != nil {
		t.Fatalf("create ipam: %v", err)
	}
	if createIpamOut.Ipam == nil || !strings.HasPrefix(aws.ToString(createIpamOut.Ipam.IpamId), "ipam-") {
		t.Fatalf("unexpected ipam output: %#v", createIpamOut.Ipam)
	}

	createIpamExternalResourceVerificationTokenOut, err := client.CreateIpamExternalResourceVerificationToken(ctx, &awsec2.CreateIpamExternalResourceVerificationTokenInput{
		IpamId: createIpamOut.Ipam.IpamId,
	})
	if err != nil {
		t.Fatalf("create ipam external resource verification token: %v", err)
	}
	if createIpamExternalResourceVerificationTokenOut.IpamExternalResourceVerificationToken == nil || !strings.HasPrefix(
		aws.ToString(createIpamExternalResourceVerificationTokenOut.IpamExternalResourceVerificationToken.IpamExternalResourceVerificationTokenId),
		"ipam-ervt-",
	) {
		t.Fatalf("unexpected ipam external resource verification token: %#v", createIpamExternalResourceVerificationTokenOut.IpamExternalResourceVerificationToken)
	}
}

func TestEC2Stage107ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateCoipPool",
		"CreateDelegateMacVolumeOwnershipTask",
		"CreateFleet",
		"CreateFlowLogs",
		"CreateFpgaImage",
		"CreateInstanceConnectEndpoint",
		"CreateInstanceEventWindow",
		"CreateInstanceExportTask",
		"CreateIpam",
		"CreateIpamExternalResourceVerificationToken",
	}

	paramsByAction := map[string]map[string]string{
		"CreateCoipPool": {
			"LocalGatewayRouteTableId": "lgw-rtb-00000000000000107",
		},
		"CreateDelegateMacVolumeOwnershipTask": {
			"InstanceId":     "i-00000000000000107",
			"MacCredentials": "mac-credentials",
		},
		"CreateFleet": {
			"LaunchTemplateConfigs.1.LaunchTemplateSpecification.LaunchTemplateId": "lt-00000000000000107",
			"TargetCapacitySpecification.TotalTargetCapacity":                      "1",
		},
		"CreateFlowLogs": {
			"ResourceId.1": "vpc-00000001",
			"ResourceType": "VPC",
		},
		"CreateFpgaImage": {
			"InputStorageLocation.Bucket": "stage107-bucket",
			"InputStorageLocation.Key":    "stage107/input.xclbin",
		},
		"CreateInstanceConnectEndpoint": {
			"SubnetId": "subnet-00000001",
		},
		"CreateInstanceEventWindow": {
			"Name": "stage107-window",
		},
		"CreateInstanceExportTask": {
			"InstanceId":          "i-00000000000000107",
			"TargetEnvironment":   "vmware",
			"ExportToS3.S3Bucket": "stage107-exports",
		},
		"CreateIpam": {
			"Description": "stage107-ipam",
		},
		"CreateIpamExternalResourceVerificationToken": {
			"IpamId": "ipam-00000000107",
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
