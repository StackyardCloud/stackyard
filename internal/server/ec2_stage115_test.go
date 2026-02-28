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

func TestEC2Stage115SDKLifecycle(t *testing.T) {
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

	createCapacityReservationOut, err := client.CreateCapacityReservation(ctx, &awsec2.CreateCapacityReservationInput{
		AvailabilityZone: aws.String("us-east-1a"),
		InstanceCount:    aws.Int32(2),
		InstancePlatform: awsec2types.CapacityReservationInstancePlatformLinuxUnix,
		InstanceType:     aws.String("m5.large"),
	})
	if err != nil {
		t.Fatalf("create capacity reservation: %v", err)
	}
	if createCapacityReservationOut.CapacityReservation == nil {
		t.Fatalf("expected created capacity reservation")
	}
	capacityReservationID := aws.ToString(createCapacityReservationOut.CapacityReservation.CapacityReservationId)
	if !strings.HasPrefix(capacityReservationID, "cr-") {
		t.Fatalf("unexpected capacity reservation id: %q", capacityReservationID)
	}

	describeCapacityBlocksOut, err := client.DescribeCapacityBlocks(ctx, &awsec2.DescribeCapacityBlocksInput{
		CapacityBlockIds: []string{"cb-" + strings.TrimPrefix(capacityReservationID, "cr-")},
		MaxResults:       aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe capacity blocks: %v", err)
	}
	if len(describeCapacityBlocksOut.CapacityBlocks) != 1 {
		t.Fatalf("expected 1 capacity block, got %d", len(describeCapacityBlocksOut.CapacityBlocks))
	}
	if aws.ToString(describeCapacityBlocksOut.CapacityBlocks[0].CapacityBlockId) != "cb-"+strings.TrimPrefix(capacityReservationID, "cr-") {
		t.Fatalf("unexpected capacity block id: %q", aws.ToString(describeCapacityBlocksOut.CapacityBlocks[0].CapacityBlockId))
	}

	describeCapacityReservationBillingRequestsOut, err := client.DescribeCapacityReservationBillingRequests(ctx, &awsec2.DescribeCapacityReservationBillingRequestsInput{
		Role:                   awsec2types.CallerRoleOdcrOwner,
		CapacityReservationIds: []string{capacityReservationID},
		MaxResults:             aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe capacity reservation billing requests: %v", err)
	}
	if len(describeCapacityReservationBillingRequestsOut.CapacityReservationBillingRequests) != 1 {
		t.Fatalf("expected 1 capacity reservation billing request, got %d", len(describeCapacityReservationBillingRequestsOut.CapacityReservationBillingRequests))
	}
	if aws.ToString(describeCapacityReservationBillingRequestsOut.CapacityReservationBillingRequests[0].CapacityReservationId) != capacityReservationID {
		t.Fatalf("unexpected capacity reservation id in billing request: %q", aws.ToString(describeCapacityReservationBillingRequestsOut.CapacityReservationBillingRequests[0].CapacityReservationId))
	}

	createCapacityReservationFleetOut, err := client.CreateCapacityReservationFleet(ctx, &awsec2.CreateCapacityReservationFleetInput{
		InstanceTypeSpecifications: []awsec2types.ReservationFleetInstanceSpecification{
			{
				AvailabilityZone: aws.String("us-east-1a"),
				InstancePlatform: awsec2types.CapacityReservationInstancePlatformLinuxUnix,
				InstanceType:     awsec2types.InstanceTypeM5Large,
			},
		},
		TotalTargetCapacity: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("create capacity reservation fleet: %v", err)
	}
	capacityReservationFleetID := aws.ToString(createCapacityReservationFleetOut.CapacityReservationFleetId)
	if !strings.HasPrefix(capacityReservationFleetID, "crf-") {
		t.Fatalf("unexpected capacity reservation fleet id: %q", capacityReservationFleetID)
	}

	describeCapacityReservationFleetsOut, err := client.DescribeCapacityReservationFleets(ctx, &awsec2.DescribeCapacityReservationFleetsInput{
		CapacityReservationFleetIds: []string{capacityReservationFleetID},
		MaxResults:                  aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe capacity reservation fleets: %v", err)
	}
	if len(describeCapacityReservationFleetsOut.CapacityReservationFleets) != 1 {
		t.Fatalf("expected 1 capacity reservation fleet, got %d", len(describeCapacityReservationFleetsOut.CapacityReservationFleets))
	}
	if aws.ToString(describeCapacityReservationFleetsOut.CapacityReservationFleets[0].CapacityReservationFleetId) != capacityReservationFleetID {
		t.Fatalf("unexpected capacity reservation fleet id: %q", aws.ToString(describeCapacityReservationFleetsOut.CapacityReservationFleets[0].CapacityReservationFleetId))
	}

	describeCapacityReservationsOut, err := client.DescribeCapacityReservations(ctx, &awsec2.DescribeCapacityReservationsInput{
		CapacityReservationIds: []string{capacityReservationID},
		MaxResults:             aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe capacity reservations: %v", err)
	}
	if len(describeCapacityReservationsOut.CapacityReservations) != 1 {
		t.Fatalf("expected 1 capacity reservation, got %d", len(describeCapacityReservationsOut.CapacityReservations))
	}
	if aws.ToString(describeCapacityReservationsOut.CapacityReservations[0].CapacityReservationId) != capacityReservationID {
		t.Fatalf("unexpected described capacity reservation id: %q", aws.ToString(describeCapacityReservationsOut.CapacityReservations[0].CapacityReservationId))
	}

	createVpcOut, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{
		CidrBlock: aws.String("10.115.0.0/16"),
	})
	if err != nil {
		t.Fatalf("create vpc: %v", err)
	}
	if createVpcOut.Vpc == nil {
		t.Fatalf("expected created vpc")
	}

	createCarrierGatewayOut, err := client.CreateCarrierGateway(ctx, &awsec2.CreateCarrierGatewayInput{
		VpcId: createVpcOut.Vpc.VpcId,
	})
	if err != nil {
		t.Fatalf("create carrier gateway: %v", err)
	}
	if createCarrierGatewayOut.CarrierGateway == nil {
		t.Fatalf("expected created carrier gateway")
	}
	carrierGatewayID := aws.ToString(createCarrierGatewayOut.CarrierGateway.CarrierGatewayId)

	describeCarrierGatewaysOut, err := client.DescribeCarrierGateways(ctx, &awsec2.DescribeCarrierGatewaysInput{
		CarrierGatewayIds: []string{carrierGatewayID},
		MaxResults:        aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe carrier gateways: %v", err)
	}
	if len(describeCarrierGatewaysOut.CarrierGateways) != 1 {
		t.Fatalf("expected 1 carrier gateway, got %d", len(describeCarrierGatewaysOut.CarrierGateways))
	}
	if aws.ToString(describeCarrierGatewaysOut.CarrierGateways[0].CarrierGatewayId) != carrierGatewayID {
		t.Fatalf("unexpected carrier gateway id: %q", aws.ToString(describeCarrierGatewaysOut.CarrierGateways[0].CarrierGatewayId))
	}

	createCoipPoolOut, err := client.CreateCoipPool(ctx, &awsec2.CreateCoipPoolInput{
		LocalGatewayRouteTableId: aws.String("lgw-rtb-00000000115"),
	})
	if err != nil {
		t.Fatalf("create coip pool: %v", err)
	}
	if createCoipPoolOut.CoipPool == nil {
		t.Fatalf("expected created coip pool")
	}
	coipPoolID := aws.ToString(createCoipPoolOut.CoipPool.PoolId)

	describeCoipPoolsOut, err := client.DescribeCoipPools(ctx, &awsec2.DescribeCoipPoolsInput{
		PoolIds:    []string{coipPoolID},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe coip pools: %v", err)
	}
	if len(describeCoipPoolsOut.CoipPools) != 1 {
		t.Fatalf("expected 1 coip pool, got %d", len(describeCoipPoolsOut.CoipPools))
	}
	if aws.ToString(describeCoipPoolsOut.CoipPools[0].PoolId) != coipPoolID {
		t.Fatalf("unexpected coip pool id: %q", aws.ToString(describeCoipPoolsOut.CoipPools[0].PoolId))
	}

	describeConversionTasksOut, err := client.DescribeConversionTasks(ctx, &awsec2.DescribeConversionTasksInput{
		ConversionTaskIds: []string{"import-i-0000000000000115"},
	})
	if err != nil {
		t.Fatalf("describe conversion tasks: %v", err)
	}
	if len(describeConversionTasksOut.ConversionTasks) != 1 {
		t.Fatalf("expected 1 conversion task, got %d", len(describeConversionTasksOut.ConversionTasks))
	}
	if aws.ToString(describeConversionTasksOut.ConversionTasks[0].ConversionTaskId) != "import-i-0000000000000115" {
		t.Fatalf("unexpected conversion task id: %q", aws.ToString(describeConversionTasksOut.ConversionTasks[0].ConversionTaskId))
	}

	describeElasticGpusOut, err := client.DescribeElasticGpus(ctx, &awsec2.DescribeElasticGpusInput{
		ElasticGpuIds: []string{"egpu-0000000000000115"},
		MaxResults:    aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe elastic gpus: %v", err)
	}
	if len(describeElasticGpusOut.ElasticGpuSet) != 1 {
		t.Fatalf("expected 1 elastic gpu, got %d", len(describeElasticGpusOut.ElasticGpuSet))
	}
	if aws.ToString(describeElasticGpusOut.ElasticGpuSet[0].ElasticGpuId) != "egpu-0000000000000115" {
		t.Fatalf("unexpected elastic gpu id: %q", aws.ToString(describeElasticGpusOut.ElasticGpuSet[0].ElasticGpuId))
	}

	describeExportImageTasksOut, err := client.DescribeExportImageTasks(ctx, &awsec2.DescribeExportImageTasksInput{
		ExportImageTaskIds: []string{"export-ami-0000000000000115"},
		MaxResults:         aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe export image tasks: %v", err)
	}
	if len(describeExportImageTasksOut.ExportImageTasks) != 1 {
		t.Fatalf("expected 1 export image task, got %d", len(describeExportImageTasksOut.ExportImageTasks))
	}
	if aws.ToString(describeExportImageTasksOut.ExportImageTasks[0].ExportImageTaskId) != "export-ami-0000000000000115" {
		t.Fatalf("unexpected export image task id: %q", aws.ToString(describeExportImageTasksOut.ExportImageTasks[0].ExportImageTaskId))
	}

	createInstanceExportTaskOut, err := client.CreateInstanceExportTask(ctx, &awsec2.CreateInstanceExportTaskInput{
		Description: aws.String("stage115-export"),
		ExportToS3Task: &awsec2types.ExportToS3TaskSpecification{
			ContainerFormat: awsec2types.ContainerFormatOva,
			DiskImageFormat: awsec2types.DiskImageFormatVmdk,
			S3Bucket:        aws.String("stage115-exports"),
			S3Prefix:        aws.String("exports"),
		},
		InstanceId:        aws.String("i-0000000000000115"),
		TargetEnvironment: awsec2types.ExportEnvironmentVmware,
	})
	if err != nil {
		t.Fatalf("create instance export task: %v", err)
	}
	if createInstanceExportTaskOut.ExportTask == nil {
		t.Fatalf("expected created export task")
	}
	exportTaskID := aws.ToString(createInstanceExportTaskOut.ExportTask.ExportTaskId)

	describeExportTasksOut, err := client.DescribeExportTasks(ctx, &awsec2.DescribeExportTasksInput{
		ExportTaskIds: []string{exportTaskID},
	})
	if err != nil {
		t.Fatalf("describe export tasks: %v", err)
	}
	if len(describeExportTasksOut.ExportTasks) != 1 {
		t.Fatalf("expected 1 export task, got %d", len(describeExportTasksOut.ExportTasks))
	}
	if aws.ToString(describeExportTasksOut.ExportTasks[0].ExportTaskId) != exportTaskID {
		t.Fatalf("unexpected export task id: %q", aws.ToString(describeExportTasksOut.ExportTasks[0].ExportTaskId))
	}
}

func TestEC2Stage115ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeCapacityBlocks",
		"DescribeCapacityReservationBillingRequests",
		"DescribeCapacityReservationFleets",
		"DescribeCapacityReservations",
		"DescribeCarrierGateways",
		"DescribeCoipPools",
		"DescribeConversionTasks",
		"DescribeElasticGpus",
		"DescribeExportImageTasks",
		"DescribeExportTasks",
	}

	paramsByAction := map[string]map[string]string{
		"DescribeCapacityBlocks": {
			"CapacityBlockId.1": "cb-0000000000000115",
			"MaxResults":        "10",
		},
		"DescribeCapacityReservationBillingRequests": {
			"Role":                    "odcr-owner",
			"CapacityReservationId.1": "cr-0000000000000115",
			"MaxResults":              "10",
		},
		"DescribeCapacityReservationFleets": {
			"CapacityReservationFleetId.1": "crf-0000000000000115",
			"MaxResults":                   "10",
		},
		"DescribeCapacityReservations": {
			"CapacityReservationId.1": "cr-0000000000000115",
			"MaxResults":              "10",
		},
		"DescribeCarrierGateways": {
			"CarrierGatewayId.1": "cagw-0000000000000115",
			"MaxResults":         "10",
		},
		"DescribeCoipPools": {
			"PoolId.1":   "coip-pool-0000000000000115",
			"MaxResults": "10",
		},
		"DescribeConversionTasks": {
			"ConversionTaskId.1": "import-i-0000000000000115",
		},
		"DescribeElasticGpus": {
			"ElasticGpuId.1": "egpu-0000000000000115",
			"MaxResults":     "10",
		},
		"DescribeExportImageTasks": {
			"ExportImageTaskId.1": "export-ami-0000000000000115",
			"MaxResults":          "10",
		},
		"DescribeExportTasks": {
			"ExportTaskId.1": "export-i-0000000000000115",
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
