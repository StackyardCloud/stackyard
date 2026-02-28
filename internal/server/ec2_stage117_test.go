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

func TestEC2Stage117SDKLifecycle(t *testing.T) {
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

	describeHostsOut, err := client.DescribeHosts(ctx, &awsec2.DescribeHostsInput{
		HostIds:    []string{hostID},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe hosts: %v", err)
	}
	if len(describeHostsOut.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(describeHostsOut.Hosts))
	}
	if aws.ToString(describeHostsOut.Hosts[0].HostId) != hostID {
		t.Fatalf("unexpected host id: %q", aws.ToString(describeHostsOut.Hosts[0].HostId))
	}

	if _, err := client.CancelImportTask(ctx, &awsec2.CancelImportTaskInput{
		ImportTaskId: aws.String("import-ami-00000000000000117"),
		CancelReason: aws.String("stage117 import image cancel"),
	}); err != nil {
		t.Fatalf("cancel import image task: %v", err)
	}
	if _, err := client.CancelImportTask(ctx, &awsec2.CancelImportTaskInput{
		ImportTaskId: aws.String("import-snap-00000000000000117"),
		CancelReason: aws.String("stage117 import snapshot cancel"),
	}); err != nil {
		t.Fatalf("cancel import snapshot task: %v", err)
	}

	describeImportImageTasksOut, err := client.DescribeImportImageTasks(ctx, &awsec2.DescribeImportImageTasksInput{
		ImportTaskIds: []string{"import-ami-00000000000000117"},
		MaxResults:    aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe import image tasks: %v", err)
	}
	if len(describeImportImageTasksOut.ImportImageTasks) != 1 {
		t.Fatalf("expected 1 import image task, got %d", len(describeImportImageTasksOut.ImportImageTasks))
	}
	if aws.ToString(describeImportImageTasksOut.ImportImageTasks[0].ImportTaskId) != "import-ami-00000000000000117" {
		t.Fatalf("unexpected import image task id: %q", aws.ToString(describeImportImageTasksOut.ImportImageTasks[0].ImportTaskId))
	}

	describeImportSnapshotTasksOut, err := client.DescribeImportSnapshotTasks(ctx, &awsec2.DescribeImportSnapshotTasksInput{
		ImportTaskIds: []string{"import-snap-00000000000000117"},
		MaxResults:    aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe import snapshot tasks: %v", err)
	}
	if len(describeImportSnapshotTasksOut.ImportSnapshotTasks) != 1 {
		t.Fatalf("expected 1 import snapshot task, got %d", len(describeImportSnapshotTasksOut.ImportSnapshotTasks))
	}
	if aws.ToString(describeImportSnapshotTasksOut.ImportSnapshotTasks[0].ImportTaskId) != "import-snap-00000000000000117" {
		t.Fatalf("unexpected import snapshot task id: %q", aws.ToString(describeImportSnapshotTasksOut.ImportSnapshotTasks[0].ImportTaskId))
	}

	runInstancesOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-stage117"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runInstancesOut.Instances) == 0 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	createEndpointOut, err := client.CreateInstanceConnectEndpoint(ctx, &awsec2.CreateInstanceConnectEndpointInput{
		SubnetId: aws.String("subnet-00000001"),
	})
	if err != nil {
		t.Fatalf("create instance connect endpoint: %v", err)
	}
	if createEndpointOut.InstanceConnectEndpoint == nil || createEndpointOut.InstanceConnectEndpoint.InstanceConnectEndpointId == nil {
		t.Fatalf("expected created instance connect endpoint")
	}
	endpointID := aws.ToString(createEndpointOut.InstanceConnectEndpoint.InstanceConnectEndpointId)

	describeInstanceConnectEndpointsOut, err := client.DescribeInstanceConnectEndpoints(ctx, &awsec2.DescribeInstanceConnectEndpointsInput{
		InstanceConnectEndpointIds: []string{endpointID},
		MaxResults:                 aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe instance connect endpoints: %v", err)
	}
	if len(describeInstanceConnectEndpointsOut.InstanceConnectEndpoints) != 1 {
		t.Fatalf("expected 1 instance connect endpoint, got %d", len(describeInstanceConnectEndpointsOut.InstanceConnectEndpoints))
	}
	if aws.ToString(describeInstanceConnectEndpointsOut.InstanceConnectEndpoints[0].InstanceConnectEndpointId) != endpointID {
		t.Fatalf("unexpected instance connect endpoint id: %q", aws.ToString(describeInstanceConnectEndpointsOut.InstanceConnectEndpoints[0].InstanceConnectEndpointId))
	}

	describeInstanceCreditSpecificationsOut, err := client.DescribeInstanceCreditSpecifications(ctx, &awsec2.DescribeInstanceCreditSpecificationsInput{
		InstanceIds: []string{instanceID},
		MaxResults:  aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe instance credit specifications: %v", err)
	}
	if len(describeInstanceCreditSpecificationsOut.InstanceCreditSpecifications) != 1 {
		t.Fatalf("expected 1 instance credit specification, got %d", len(describeInstanceCreditSpecificationsOut.InstanceCreditSpecifications))
	}
	if aws.ToString(describeInstanceCreditSpecificationsOut.InstanceCreditSpecifications[0].InstanceId) != instanceID {
		t.Fatalf("unexpected instance id in credit specification: %q", aws.ToString(describeInstanceCreditSpecificationsOut.InstanceCreditSpecifications[0].InstanceId))
	}

	describeInstanceEventNotificationAttributesOut, err := client.DescribeInstanceEventNotificationAttributes(ctx, &awsec2.DescribeInstanceEventNotificationAttributesInput{})
	if err != nil {
		t.Fatalf("describe instance event notification attributes: %v", err)
	}
	if describeInstanceEventNotificationAttributesOut.InstanceTagAttribute == nil {
		t.Fatalf("expected instance tag attributes output")
	}

	createInstanceEventWindowOut, err := client.CreateInstanceEventWindow(ctx, &awsec2.CreateInstanceEventWindowInput{
		Name:           aws.String("stage117-window"),
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
	if createInstanceEventWindowOut.InstanceEventWindow == nil || createInstanceEventWindowOut.InstanceEventWindow.InstanceEventWindowId == nil {
		t.Fatalf("expected created instance event window")
	}
	windowID := aws.ToString(createInstanceEventWindowOut.InstanceEventWindow.InstanceEventWindowId)

	describeInstanceEventWindowsOut, err := client.DescribeInstanceEventWindows(ctx, &awsec2.DescribeInstanceEventWindowsInput{
		InstanceEventWindowIds: []string{windowID},
		MaxResults:             aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe instance event windows: %v", err)
	}
	if len(describeInstanceEventWindowsOut.InstanceEventWindows) != 1 {
		t.Fatalf("expected 1 instance event window, got %d", len(describeInstanceEventWindowsOut.InstanceEventWindows))
	}
	if aws.ToString(describeInstanceEventWindowsOut.InstanceEventWindows[0].InstanceEventWindowId) != windowID {
		t.Fatalf("unexpected instance event window id: %q", aws.ToString(describeInstanceEventWindowsOut.InstanceEventWindows[0].InstanceEventWindowId))
	}

	describeInstanceImageMetadataOut, err := client.DescribeInstanceImageMetadata(ctx, &awsec2.DescribeInstanceImageMetadataInput{
		InstanceIds: []string{instanceID},
		MaxResults:  aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe instance image metadata: %v", err)
	}
	if len(describeInstanceImageMetadataOut.InstanceImageMetadata) != 1 {
		t.Fatalf("expected 1 instance image metadata record, got %d", len(describeInstanceImageMetadataOut.InstanceImageMetadata))
	}
	if aws.ToString(describeInstanceImageMetadataOut.InstanceImageMetadata[0].InstanceId) != instanceID {
		t.Fatalf("unexpected instance id in image metadata: %q", aws.ToString(describeInstanceImageMetadataOut.InstanceImageMetadata[0].InstanceId))
	}

	describeInstanceTopologyOut, err := client.DescribeInstanceTopology(ctx, &awsec2.DescribeInstanceTopologyInput{
		InstanceIds: []string{instanceID},
		MaxResults:  aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe instance topology: %v", err)
	}
	if len(describeInstanceTopologyOut.Instances) != 1 {
		t.Fatalf("expected 1 instance topology record, got %d", len(describeInstanceTopologyOut.Instances))
	}
	if aws.ToString(describeInstanceTopologyOut.Instances[0].InstanceId) != instanceID {
		t.Fatalf("unexpected topology instance id: %q", aws.ToString(describeInstanceTopologyOut.Instances[0].InstanceId))
	}

	describeInstanceTypeOfferingsOut, err := client.DescribeInstanceTypeOfferings(ctx, &awsec2.DescribeInstanceTypeOfferingsInput{
		LocationType: awsec2types.LocationTypeRegion,
		MaxResults:   aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe instance type offerings: %v", err)
	}
	if len(describeInstanceTypeOfferingsOut.InstanceTypeOfferings) == 0 {
		t.Fatalf("expected at least one instance type offering")
	}
}

func TestEC2Stage117ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeHosts",
		"DescribeImportImageTasks",
		"DescribeImportSnapshotTasks",
		"DescribeInstanceConnectEndpoints",
		"DescribeInstanceCreditSpecifications",
		"DescribeInstanceEventNotificationAttributes",
		"DescribeInstanceEventWindows",
		"DescribeInstanceImageMetadata",
		"DescribeInstanceTopology",
		"DescribeInstanceTypeOfferings",
	}

	paramsByAction := map[string]map[string]string{
		"DescribeHosts": {
			"HostId.1":   "h-00000000117",
			"MaxResults": "10",
		},
		"DescribeImportImageTasks": {
			"ImportTaskId.1":    "import-ami-00000000000000117",
			"Filters.1.Name":    "status",
			"Filters.1.Value.1": "active",
			"MaxResults":        "10",
		},
		"DescribeImportSnapshotTasks": {
			"ImportTaskId.1":    "import-snap-00000000000000117",
			"Filters.1.Name":    "status",
			"Filters.1.Value.1": "active",
			"MaxResults":        "10",
		},
		"DescribeInstanceConnectEndpoints": {
			"InstanceConnectEndpointId.1": "eice-00000000117",
			"MaxResults":                  "10",
		},
		"DescribeInstanceCreditSpecifications": {
			"InstanceId.1": "i-00000000117",
			"MaxResults":   "10",
		},
		"DescribeInstanceEventNotificationAttributes": nil,
		"DescribeInstanceEventWindows": {
			"InstanceEventWindowId.1": "iew-00000000117",
			"MaxResults":              "10",
		},
		"DescribeInstanceImageMetadata": {
			"InstanceId.1": "i-00000000117",
			"MaxResults":   "10",
		},
		"DescribeInstanceTopology": {
			"InstanceId.1": "i-00000000117",
			"MaxResults":   "10",
		},
		"DescribeInstanceTypeOfferings": {
			"LocationType": "region",
			"MaxResults":   "10",
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
