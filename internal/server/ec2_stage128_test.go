package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage128SDKLifecycle(t *testing.T) {
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

	createInstanceConnectEndpointOut, err := client.CreateInstanceConnectEndpoint(ctx, &awsec2.CreateInstanceConnectEndpointInput{
		SubnetId: aws.String("subnet-00000001"),
	})
	if err != nil || createInstanceConnectEndpointOut.InstanceConnectEndpoint == nil || createInstanceConnectEndpointOut.InstanceConnectEndpoint.InstanceConnectEndpointId == nil {
		t.Fatalf("create instance connect endpoint: %v", err)
	}

	modifyInstanceConnectEndpointOut, err := client.ModifyInstanceConnectEndpoint(ctx, &awsec2.ModifyInstanceConnectEndpointInput{
		InstanceConnectEndpointId: createInstanceConnectEndpointOut.InstanceConnectEndpoint.InstanceConnectEndpointId,
		IpAddressType:             awsec2types.IpAddressTypeIpv4,
		PreserveClientIp:          aws.Bool(false),
		SecurityGroupIds:          []string{"sg-00000000"},
	})
	if err != nil {
		t.Fatalf("modify instance connect endpoint: %v", err)
	}
	if modifyInstanceConnectEndpointOut.Return == nil || !aws.ToBool(modifyInstanceConnectEndpointOut.Return) {
		t.Fatalf("expected modify instance connect endpoint return true")
	}

	runInstancesOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:  aws.String("ami-00000000000000128"),
		MinCount: aws.Int32(1),
		MaxCount: aws.Int32(1),
		SubnetId: aws.String("subnet-00000001"),
	})
	if err != nil || len(runInstancesOut.Instances) != 1 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	modifyInstanceCpuOptionsOut, err := client.ModifyInstanceCpuOptions(ctx, &awsec2.ModifyInstanceCpuOptionsInput{
		InstanceId:     aws.String(instanceID),
		CoreCount:      aws.Int32(1),
		ThreadsPerCore: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("modify instance cpu options: %v", err)
	}
	if aws.ToInt32(modifyInstanceCpuOptionsOut.CoreCount) != 1 || aws.ToInt32(modifyInstanceCpuOptionsOut.ThreadsPerCore) != 1 {
		t.Fatalf("unexpected modify instance cpu options output: %#v", modifyInstanceCpuOptionsOut)
	}

	modifyInstanceCreditSpecificationOut, err := client.ModifyInstanceCreditSpecification(ctx, &awsec2.ModifyInstanceCreditSpecificationInput{
		InstanceCreditSpecifications: []awsec2types.InstanceCreditSpecificationRequest{
			{
				InstanceId: aws.String(instanceID),
				CpuCredits: aws.String("unlimited"),
			},
		},
	})
	if err != nil {
		t.Fatalf("modify instance credit specification: %v", err)
	}
	if len(modifyInstanceCreditSpecificationOut.SuccessfulInstanceCreditSpecifications) != 1 {
		t.Fatalf("expected one successful instance credit specification, got %#v", modifyInstanceCreditSpecificationOut.SuccessfulInstanceCreditSpecifications)
	}
	if len(modifyInstanceCreditSpecificationOut.UnsuccessfulInstanceCreditSpecifications) != 0 {
		t.Fatalf("expected zero unsuccessful instance credit specifications, got %#v", modifyInstanceCreditSpecificationOut.UnsuccessfulInstanceCreditSpecifications)
	}

	describeInstanceCreditSpecificationsOut, err := client.DescribeInstanceCreditSpecifications(ctx, &awsec2.DescribeInstanceCreditSpecificationsInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		t.Fatalf("describe instance credit specifications: %v", err)
	}
	if len(describeInstanceCreditSpecificationsOut.InstanceCreditSpecifications) != 1 ||
		aws.ToString(describeInstanceCreditSpecificationsOut.InstanceCreditSpecifications[0].CpuCredits) != "unlimited" {
		t.Fatalf("unexpected describe instance credit specifications output: %#v", describeInstanceCreditSpecificationsOut.InstanceCreditSpecifications)
	}

	notBefore := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	modifyInstanceEventStartTimeOut, err := client.ModifyInstanceEventStartTime(ctx, &awsec2.ModifyInstanceEventStartTimeInput{
		InstanceId:      aws.String(instanceID),
		InstanceEventId: aws.String("evt-stage128-1"),
		NotBefore:       aws.Time(notBefore),
	})
	if err != nil {
		t.Fatalf("modify instance event start time: %v", err)
	}
	if modifyInstanceEventStartTimeOut.Event == nil || aws.ToString(modifyInstanceEventStartTimeOut.Event.InstanceEventId) != "evt-stage128-1" {
		t.Fatalf("unexpected modify instance event start time output: %#v", modifyInstanceEventStartTimeOut.Event)
	}

	createInstanceEventWindowOut, err := client.CreateInstanceEventWindow(ctx, &awsec2.CreateInstanceEventWindowInput{
		Name:           aws.String("stage128-window"),
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
	windowID := aws.ToString(createInstanceEventWindowOut.InstanceEventWindow.InstanceEventWindowId)

	modifyInstanceEventWindowOut, err := client.ModifyInstanceEventWindow(ctx, &awsec2.ModifyInstanceEventWindowInput{
		InstanceEventWindowId: aws.String(windowID),
		Name:                  aws.String("stage128-window-updated"),
		CronExpression:        aws.String("cron(0 11 ? * MON *)"),
		TimeRanges: []awsec2types.InstanceEventWindowTimeRangeRequest{
			{
				StartWeekDay: awsec2types.WeekDayMonday,
				StartHour:    aws.Int32(11),
				EndWeekDay:   awsec2types.WeekDayMonday,
				EndHour:      aws.Int32(12),
			},
		},
	})
	if err != nil {
		t.Fatalf("modify instance event window: %v", err)
	}
	if modifyInstanceEventWindowOut.InstanceEventWindow == nil || aws.ToString(modifyInstanceEventWindowOut.InstanceEventWindow.Name) != "stage128-window-updated" {
		t.Fatalf("unexpected modify instance event window output: %#v", modifyInstanceEventWindowOut.InstanceEventWindow)
	}

	modifyInstanceMaintenanceOptionsOut, err := client.ModifyInstanceMaintenanceOptions(ctx, &awsec2.ModifyInstanceMaintenanceOptionsInput{
		InstanceId:      aws.String(instanceID),
		AutoRecovery:    awsec2types.InstanceAutoRecoveryState("disabled"),
		RebootMigration: awsec2types.InstanceRebootMigrationState("disabled"),
	})
	if err != nil {
		t.Fatalf("modify instance maintenance options: %v", err)
	}
	if string(modifyInstanceMaintenanceOptionsOut.AutoRecovery) != "disabled" ||
		string(modifyInstanceMaintenanceOptionsOut.RebootMigration) != "disabled" {
		t.Fatalf("unexpected modify instance maintenance options output: %#v", modifyInstanceMaintenanceOptionsOut)
	}

	modifyInstanceMetadataDefaultsOut, err := client.ModifyInstanceMetadataDefaults(ctx, &awsec2.ModifyInstanceMetadataDefaultsInput{
		HttpEndpoint:            awsec2types.DefaultInstanceMetadataEndpointState("enabled"),
		HttpTokens:              awsec2types.MetadataDefaultHttpTokensState("required"),
		InstanceMetadataTags:    awsec2types.DefaultInstanceMetadataTagsState("enabled"),
		HttpPutResponseHopLimit: aws.Int32(4),
	})
	if err != nil {
		t.Fatalf("modify instance metadata defaults: %v", err)
	}
	if modifyInstanceMetadataDefaultsOut.Return == nil || !aws.ToBool(modifyInstanceMetadataDefaultsOut.Return) {
		t.Fatalf("expected modify instance metadata defaults return true")
	}

	getInstanceMetadataDefaultsOut, err := client.GetInstanceMetadataDefaults(ctx, &awsec2.GetInstanceMetadataDefaultsInput{})
	if err != nil {
		t.Fatalf("get instance metadata defaults: %v", err)
	}
	if string(getInstanceMetadataDefaultsOut.AccountLevel.HttpTokens) != "required" ||
		string(getInstanceMetadataDefaultsOut.AccountLevel.InstanceMetadataTags) != "enabled" {
		t.Fatalf("unexpected get instance metadata defaults output: %#v", getInstanceMetadataDefaultsOut.AccountLevel)
	}

	modifyInstanceMetadataOptionsOut, err := client.ModifyInstanceMetadataOptions(ctx, &awsec2.ModifyInstanceMetadataOptionsInput{
		InstanceId:              aws.String(instanceID),
		HttpEndpoint:            awsec2types.InstanceMetadataEndpointState("enabled"),
		HttpProtocolIpv6:        awsec2types.InstanceMetadataProtocolState("disabled"),
		HttpPutResponseHopLimit: aws.Int32(5),
		HttpTokens:              awsec2types.HttpTokensState("required"),
		InstanceMetadataTags:    awsec2types.InstanceMetadataTagsState("enabled"),
	})
	if err != nil {
		t.Fatalf("modify instance metadata options: %v", err)
	}
	if modifyInstanceMetadataOptionsOut.InstanceMetadataOptions == nil ||
		string(modifyInstanceMetadataOptionsOut.InstanceMetadataOptions.HttpTokens) != "required" ||
		aws.ToInt32(modifyInstanceMetadataOptionsOut.InstanceMetadataOptions.HttpPutResponseHopLimit) != 5 {
		t.Fatalf("unexpected modify instance metadata options output: %#v", modifyInstanceMetadataOptionsOut.InstanceMetadataOptions)
	}

	modifyInstanceNetworkPerformanceOptionsOut, err := client.ModifyInstanceNetworkPerformanceOptions(ctx, &awsec2.ModifyInstanceNetworkPerformanceOptionsInput{
		InstanceId:         aws.String(instanceID),
		BandwidthWeighting: awsec2types.InstanceBandwidthWeighting("vpc-1"),
	})
	if err != nil {
		t.Fatalf("modify instance network performance options: %v", err)
	}
	if string(modifyInstanceNetworkPerformanceOptionsOut.BandwidthWeighting) != "vpc-1" {
		t.Fatalf("unexpected modify instance network performance options output: %#v", modifyInstanceNetworkPerformanceOptionsOut)
	}

	modifyInstancePlacementOut, err := client.ModifyInstancePlacement(ctx, &awsec2.ModifyInstancePlacementInput{
		InstanceId: aws.String(instanceID),
		Affinity:   awsec2types.Affinity("default"),
		GroupName:  aws.String(""),
		Tenancy:    awsec2types.HostTenancy("default"),
	})
	if err != nil {
		t.Fatalf("modify instance placement: %v", err)
	}
	if modifyInstancePlacementOut.Return == nil || !aws.ToBool(modifyInstancePlacementOut.Return) {
		t.Fatalf("expected modify instance placement return true")
	}
}

func TestEC2Stage128ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyInstanceConnectEndpoint",
		"ModifyInstanceCpuOptions",
		"ModifyInstanceCreditSpecification",
		"ModifyInstanceEventStartTime",
		"ModifyInstanceEventWindow",
		"ModifyInstanceMaintenanceOptions",
		"ModifyInstanceMetadataDefaults",
		"ModifyInstanceMetadataOptions",
		"ModifyInstanceNetworkPerformanceOptions",
		"ModifyInstancePlacement",
	}

	paramsByAction := map[string]map[string]string{
		"ModifyInstanceConnectEndpoint": {
			"InstanceConnectEndpointId": "eice-00000000128",
		},
		"ModifyInstanceCpuOptions": {
			"InstanceId":     "i-00000000128",
			"CoreCount":      "1",
			"ThreadsPerCore": "1",
		},
		"ModifyInstanceCreditSpecification": {
			"InstanceCreditSpecification.1.InstanceId": "i-00000000128",
			"InstanceCreditSpecification.1.CpuCredits": "unlimited",
		},
		"ModifyInstanceEventStartTime": {
			"InstanceId":      "i-00000000128",
			"InstanceEventId": "evt-stage128",
			"NotBefore":       "2026-01-01T00:00:00Z",
		},
		"ModifyInstanceEventWindow": {
			"InstanceEventWindowId": "iew-00000000128",
			"CronExpression":        "cron(0 10 ? * SUN *)",
		},
		"ModifyInstanceMaintenanceOptions": {
			"InstanceId":      "i-00000000128",
			"AutoRecovery":    "disabled",
			"RebootMigration": "disabled",
		},
		"ModifyInstanceMetadataDefaults": {
			"HttpEndpoint": "enabled",
		},
		"ModifyInstanceMetadataOptions": {
			"InstanceId":   "i-00000000128",
			"HttpEndpoint": "enabled",
		},
		"ModifyInstanceNetworkPerformanceOptions": {
			"InstanceId":         "i-00000000128",
			"BandwidthWeighting": "vpc-1",
		},
		"ModifyInstancePlacement": {
			"InstanceId": "i-00000000128",
			"Affinity":   "default",
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
