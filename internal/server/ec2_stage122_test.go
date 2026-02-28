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

func TestEC2Stage122SDKLifecycle(t *testing.T) {
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

	spotFleetRequestID := "sfr-stage122"
	_, err = client.CancelSpotFleetRequests(ctx, &awsec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: []string{spotFleetRequestID},
		TerminateInstances:  aws.Bool(false),
	})
	if err != nil {
		t.Fatalf("cancel spot fleet requests: %v", err)
	}

	describeSpotFleetRequestHistoryOut, err := client.DescribeSpotFleetRequestHistory(ctx, &awsec2.DescribeSpotFleetRequestHistoryInput{
		SpotFleetRequestId: aws.String(spotFleetRequestID),
		StartTime:          aws.Time(time.Now().UTC().Add(-1 * time.Hour)),
		EventType:          awsec2types.EventTypeInstanceChange,
		MaxResults:         aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe spot fleet request history: %v", err)
	}
	if aws.ToString(describeSpotFleetRequestHistoryOut.SpotFleetRequestId) != spotFleetRequestID || len(describeSpotFleetRequestHistoryOut.HistoryRecords) == 0 {
		t.Fatalf("unexpected describe spot fleet request history output: %#v", describeSpotFleetRequestHistoryOut)
	}

	describeSpotFleetRequestsOut, err := client.DescribeSpotFleetRequests(ctx, &awsec2.DescribeSpotFleetRequestsInput{
		SpotFleetRequestIds: []string{spotFleetRequestID},
		MaxResults:          aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe spot fleet requests: %v", err)
	}
	if len(describeSpotFleetRequestsOut.SpotFleetRequestConfigs) != 1 ||
		aws.ToString(describeSpotFleetRequestsOut.SpotFleetRequestConfigs[0].SpotFleetRequestId) != spotFleetRequestID {
		t.Fatalf("unexpected describe spot fleet requests output: %#v", describeSpotFleetRequestsOut.SpotFleetRequestConfigs)
	}

	spotInstanceRequestID := "sir-stage122"
	_, err = client.CancelSpotInstanceRequests(ctx, &awsec2.CancelSpotInstanceRequestsInput{
		SpotInstanceRequestIds: []string{spotInstanceRequestID},
	})
	if err != nil {
		t.Fatalf("cancel spot instance requests: %v", err)
	}

	describeSpotInstanceRequestsOut, err := client.DescribeSpotInstanceRequests(ctx, &awsec2.DescribeSpotInstanceRequestsInput{
		SpotInstanceRequestIds: []string{spotInstanceRequestID},
		MaxResults:             aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe spot instance requests: %v", err)
	}
	if len(describeSpotInstanceRequestsOut.SpotInstanceRequests) != 1 ||
		aws.ToString(describeSpotInstanceRequestsOut.SpotInstanceRequests[0].SpotInstanceRequestId) != spotInstanceRequestID {
		t.Fatalf("unexpected describe spot instance requests output: %#v", describeSpotInstanceRequestsOut.SpotInstanceRequests)
	}

	describeSpotPriceHistoryOut, err := client.DescribeSpotPriceHistory(ctx, &awsec2.DescribeSpotPriceHistoryInput{
		InstanceTypes:       []awsec2types.InstanceType{awsec2types.InstanceTypeT3Micro},
		ProductDescriptions: []string{string(awsec2types.RIProductDescriptionLinuxUnix)},
		StartTime:           aws.Time(time.Now().UTC().Add(-1 * time.Hour)),
		EndTime:             aws.Time(time.Now().UTC()),
		MaxResults:          aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe spot price history: %v", err)
	}
	if len(describeSpotPriceHistoryOut.SpotPriceHistory) == 0 {
		t.Fatalf("expected spot price history entries")
	}

	runInstancesOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-stage122"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runInstancesOut.Instances) == 0 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	createImageOut, err := client.CreateImage(ctx, &awsec2.CreateImageInput{
		InstanceId: aws.String(instanceID),
		Name:       aws.String("stage122-image"),
	})
	if err != nil || createImageOut.ImageId == nil {
		t.Fatalf("create image: %v", err)
	}
	imageID := aws.ToString(createImageOut.ImageId)

	_, err = client.CreateStoreImageTask(ctx, &awsec2.CreateStoreImageTaskInput{
		Bucket:  aws.String("stage122-bucket"),
		ImageId: aws.String(imageID),
	})
	if err != nil {
		t.Fatalf("create store image task: %v", err)
	}

	describeStoreImageTasksOut, err := client.DescribeStoreImageTasks(ctx, &awsec2.DescribeStoreImageTasksInput{
		ImageIds:   []string{imageID},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe store image tasks: %v", err)
	}
	if len(describeStoreImageTasksOut.StoreImageTaskResults) == 0 ||
		aws.ToString(describeStoreImageTasksOut.StoreImageTaskResults[0].AmiId) != imageID {
		t.Fatalf("unexpected describe store image tasks output: %#v", describeStoreImageTasksOut.StoreImageTaskResults)
	}

	createTrafficMirrorFilterOut, err := client.CreateTrafficMirrorFilter(ctx, &awsec2.CreateTrafficMirrorFilterInput{})
	if err != nil || createTrafficMirrorFilterOut.TrafficMirrorFilter == nil || createTrafficMirrorFilterOut.TrafficMirrorFilter.TrafficMirrorFilterId == nil {
		t.Fatalf("create traffic mirror filter: %v", err)
	}
	trafficMirrorFilterID := aws.ToString(createTrafficMirrorFilterOut.TrafficMirrorFilter.TrafficMirrorFilterId)

	createTrafficMirrorFilterRuleOut, err := client.CreateTrafficMirrorFilterRule(ctx, &awsec2.CreateTrafficMirrorFilterRuleInput{
		DestinationCidrBlock:  aws.String("0.0.0.0/0"),
		RuleAction:            awsec2types.TrafficMirrorRuleActionAccept,
		RuleNumber:            aws.Int32(1),
		SourceCidrBlock:       aws.String("0.0.0.0/0"),
		TrafficDirection:      awsec2types.TrafficDirectionIngress,
		TrafficMirrorFilterId: aws.String(trafficMirrorFilterID),
	})
	if err != nil || createTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule == nil || createTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule.TrafficMirrorFilterRuleId == nil {
		t.Fatalf("create traffic mirror filter rule: %v", err)
	}
	trafficMirrorFilterRuleID := aws.ToString(createTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule.TrafficMirrorFilterRuleId)

	createTrafficMirrorTargetOut, err := client.CreateTrafficMirrorTarget(ctx, &awsec2.CreateTrafficMirrorTargetInput{
		NetworkLoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/stage122/1234567890abcdef"),
	})
	if err != nil || createTrafficMirrorTargetOut.TrafficMirrorTarget == nil || createTrafficMirrorTargetOut.TrafficMirrorTarget.TrafficMirrorTargetId == nil {
		t.Fatalf("create traffic mirror target: %v", err)
	}
	trafficMirrorTargetID := aws.ToString(createTrafficMirrorTargetOut.TrafficMirrorTarget.TrafficMirrorTargetId)

	createTrafficMirrorSessionOut, err := client.CreateTrafficMirrorSession(ctx, &awsec2.CreateTrafficMirrorSessionInput{
		NetworkInterfaceId:    aws.String("eni-stage122"),
		SessionNumber:         aws.Int32(1),
		TrafficMirrorFilterId: aws.String(trafficMirrorFilterID),
		TrafficMirrorTargetId: aws.String(trafficMirrorTargetID),
	})
	if err != nil || createTrafficMirrorSessionOut.TrafficMirrorSession == nil || createTrafficMirrorSessionOut.TrafficMirrorSession.TrafficMirrorSessionId == nil {
		t.Fatalf("create traffic mirror session: %v", err)
	}
	trafficMirrorSessionID := aws.ToString(createTrafficMirrorSessionOut.TrafficMirrorSession.TrafficMirrorSessionId)

	describeTrafficMirrorFilterRulesOut, err := client.DescribeTrafficMirrorFilterRules(ctx, &awsec2.DescribeTrafficMirrorFilterRulesInput{
		TrafficMirrorFilterId:      aws.String(trafficMirrorFilterID),
		TrafficMirrorFilterRuleIds: []string{trafficMirrorFilterRuleID},
		MaxResults:                 aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe traffic mirror filter rules: %v", err)
	}
	if len(describeTrafficMirrorFilterRulesOut.TrafficMirrorFilterRules) != 1 ||
		aws.ToString(describeTrafficMirrorFilterRulesOut.TrafficMirrorFilterRules[0].TrafficMirrorFilterRuleId) != trafficMirrorFilterRuleID {
		t.Fatalf("unexpected describe traffic mirror filter rules output: %#v", describeTrafficMirrorFilterRulesOut.TrafficMirrorFilterRules)
	}

	describeTrafficMirrorFiltersOut, err := client.DescribeTrafficMirrorFilters(ctx, &awsec2.DescribeTrafficMirrorFiltersInput{
		TrafficMirrorFilterIds: []string{trafficMirrorFilterID},
		MaxResults:             aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe traffic mirror filters: %v", err)
	}
	if len(describeTrafficMirrorFiltersOut.TrafficMirrorFilters) != 1 ||
		aws.ToString(describeTrafficMirrorFiltersOut.TrafficMirrorFilters[0].TrafficMirrorFilterId) != trafficMirrorFilterID {
		t.Fatalf("unexpected describe traffic mirror filters output: %#v", describeTrafficMirrorFiltersOut.TrafficMirrorFilters)
	}

	describeTrafficMirrorSessionsOut, err := client.DescribeTrafficMirrorSessions(ctx, &awsec2.DescribeTrafficMirrorSessionsInput{
		TrafficMirrorSessionIds: []string{trafficMirrorSessionID},
		MaxResults:              aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe traffic mirror sessions: %v", err)
	}
	if len(describeTrafficMirrorSessionsOut.TrafficMirrorSessions) != 1 ||
		aws.ToString(describeTrafficMirrorSessionsOut.TrafficMirrorSessions[0].TrafficMirrorSessionId) != trafficMirrorSessionID {
		t.Fatalf("unexpected describe traffic mirror sessions output: %#v", describeTrafficMirrorSessionsOut.TrafficMirrorSessions)
	}

	describeTrafficMirrorTargetsOut, err := client.DescribeTrafficMirrorTargets(ctx, &awsec2.DescribeTrafficMirrorTargetsInput{
		TrafficMirrorTargetIds: []string{trafficMirrorTargetID},
		MaxResults:             aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe traffic mirror targets: %v", err)
	}
	if len(describeTrafficMirrorTargetsOut.TrafficMirrorTargets) != 1 ||
		aws.ToString(describeTrafficMirrorTargetsOut.TrafficMirrorTargets[0].TrafficMirrorTargetId) != trafficMirrorTargetID {
		t.Fatalf("unexpected describe traffic mirror targets output: %#v", describeTrafficMirrorTargetsOut.TrafficMirrorTargets)
	}

	createBranchInterfaceOut, err := client.CreateNetworkInterface(ctx, &awsec2.CreateNetworkInterfaceInput{SubnetId: aws.String("subnet-00000001")})
	if err != nil || createBranchInterfaceOut.NetworkInterface == nil || createBranchInterfaceOut.NetworkInterface.NetworkInterfaceId == nil {
		t.Fatalf("create branch network interface: %v", err)
	}
	branchInterfaceID := aws.ToString(createBranchInterfaceOut.NetworkInterface.NetworkInterfaceId)

	createTrunkInterfaceOut, err := client.CreateNetworkInterface(ctx, &awsec2.CreateNetworkInterfaceInput{SubnetId: aws.String("subnet-00000001")})
	if err != nil || createTrunkInterfaceOut.NetworkInterface == nil || createTrunkInterfaceOut.NetworkInterface.NetworkInterfaceId == nil {
		t.Fatalf("create trunk network interface: %v", err)
	}
	trunkInterfaceID := aws.ToString(createTrunkInterfaceOut.NetworkInterface.NetworkInterfaceId)

	associateTrunkInterfaceOut, err := client.AssociateTrunkInterface(ctx, &awsec2.AssociateTrunkInterfaceInput{
		BranchInterfaceId: aws.String(branchInterfaceID),
		TrunkInterfaceId:  aws.String(trunkInterfaceID),
		VlanId:            aws.Int32(122),
	})
	if err != nil || associateTrunkInterfaceOut.InterfaceAssociation == nil || associateTrunkInterfaceOut.InterfaceAssociation.AssociationId == nil {
		t.Fatalf("associate trunk interface: %v", err)
	}
	associationID := aws.ToString(associateTrunkInterfaceOut.InterfaceAssociation.AssociationId)

	describeTrunkInterfaceAssociationsOut, err := client.DescribeTrunkInterfaceAssociations(ctx, &awsec2.DescribeTrunkInterfaceAssociationsInput{
		AssociationIds: []string{associationID},
		MaxResults:     aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe trunk interface associations: %v", err)
	}
	if len(describeTrunkInterfaceAssociationsOut.InterfaceAssociations) != 1 ||
		aws.ToString(describeTrunkInterfaceAssociationsOut.InterfaceAssociations[0].AssociationId) != associationID {
		t.Fatalf("unexpected describe trunk interface associations output: %#v", describeTrunkInterfaceAssociationsOut.InterfaceAssociations)
	}
}

func TestEC2Stage122ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeSpotFleetRequestHistory",
		"DescribeSpotFleetRequests",
		"DescribeSpotInstanceRequests",
		"DescribeSpotPriceHistory",
		"DescribeStoreImageTasks",
		"DescribeTrafficMirrorFilterRules",
		"DescribeTrafficMirrorFilters",
		"DescribeTrafficMirrorSessions",
		"DescribeTrafficMirrorTargets",
		"DescribeTrunkInterfaceAssociations",
	}

	paramsByAction := map[string]map[string]string{
		"DescribeSpotFleetRequestHistory": {
			"SpotFleetRequestId": "sfr-0000000122",
			"StartTime":          "2026-02-15T00:00:00Z",
			"MaxResults":         "10",
		},
		"DescribeSpotFleetRequests": {
			"SpotFleetRequestId.1": "sfr-0000000122",
			"MaxResults":           "10",
		},
		"DescribeSpotInstanceRequests": {
			"SpotInstanceRequestId.1": "sir-0000000122",
			"MaxResults":              "10",
		},
		"DescribeSpotPriceHistory": {
			"MaxResults": "10",
		},
		"DescribeStoreImageTasks": {
			"MaxResults": "10",
		},
		"DescribeTrafficMirrorFilterRules": {
			"TrafficMirrorFilterId": "tmf-0000000122",
			"MaxResults":            "10",
		},
		"DescribeTrafficMirrorFilters": {
			"TrafficMirrorFilterId.1": "tmf-0000000122",
			"MaxResults":              "10",
		},
		"DescribeTrafficMirrorSessions": {
			"TrafficMirrorSessionId.1": "tms-0000000122",
			"MaxResults":               "10",
		},
		"DescribeTrafficMirrorTargets": {
			"TrafficMirrorTargetId.1": "tmt-0000000122",
			"MaxResults":              "10",
		},
		"DescribeTrunkInterfaceAssociations": {
			"AssociationId.1": "trunk-assoc-0000000122",
			"MaxResults":      "10",
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
