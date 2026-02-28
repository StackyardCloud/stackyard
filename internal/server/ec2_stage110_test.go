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

func TestEC2Stage110SDKLifecycle(t *testing.T) {
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

	createSpotDatafeedSubscriptionOut, err := client.CreateSpotDatafeedSubscription(ctx, &awsec2.CreateSpotDatafeedSubscriptionInput{
		Bucket: aws.String("stage110-datafeed-bucket"),
		Prefix: aws.String("spot/"),
	})
	if err != nil {
		t.Fatalf("create spot datafeed subscription: %v", err)
	}
	if createSpotDatafeedSubscriptionOut.SpotDatafeedSubscription == nil || aws.ToString(createSpotDatafeedSubscriptionOut.SpotDatafeedSubscription.Bucket) != "stage110-datafeed-bucket" {
		t.Fatalf("unexpected spot datafeed subscription output: %#v", createSpotDatafeedSubscriptionOut.SpotDatafeedSubscription)
	}

	runOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-stage110"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runOut.Instances) == 0 {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)

	createImageOut, err := client.CreateImage(ctx, &awsec2.CreateImageInput{
		InstanceId: aws.String(instanceID),
		Name:       aws.String("stage110-image"),
	})
	if err != nil {
		t.Fatalf("create image: %v", err)
	}

	createStoreImageTaskOut, err := client.CreateStoreImageTask(ctx, &awsec2.CreateStoreImageTaskInput{
		Bucket:  aws.String("stage110-store-bucket"),
		ImageId: createImageOut.ImageId,
	})
	if err != nil {
		t.Fatalf("create store image task: %v", err)
	}
	if strings.TrimSpace(aws.ToString(createStoreImageTaskOut.ObjectKey)) == "" {
		t.Fatalf("expected object key in create store image task output")
	}

	createTrafficMirrorFilterOut, err := client.CreateTrafficMirrorFilter(ctx, &awsec2.CreateTrafficMirrorFilterInput{
		Description: aws.String("stage110 filter"),
	})
	if err != nil {
		t.Fatalf("create traffic mirror filter: %v", err)
	}
	if createTrafficMirrorFilterOut.TrafficMirrorFilter == nil || !strings.HasPrefix(aws.ToString(createTrafficMirrorFilterOut.TrafficMirrorFilter.TrafficMirrorFilterId), "tmf-") {
		t.Fatalf("unexpected traffic mirror filter output: %#v", createTrafficMirrorFilterOut.TrafficMirrorFilter)
	}

	createTrafficMirrorTargetOut, err := client.CreateTrafficMirrorTarget(ctx, &awsec2.CreateTrafficMirrorTargetInput{
		NetworkInterfaceId: aws.String("eni-00000000000000110"),
	})
	if err != nil {
		t.Fatalf("create traffic mirror target: %v", err)
	}
	if createTrafficMirrorTargetOut.TrafficMirrorTarget == nil || !strings.HasPrefix(aws.ToString(createTrafficMirrorTargetOut.TrafficMirrorTarget.TrafficMirrorTargetId), "tmt-") {
		t.Fatalf("unexpected traffic mirror target output: %#v", createTrafficMirrorTargetOut.TrafficMirrorTarget)
	}

	createTrafficMirrorFilterRuleOut, err := client.CreateTrafficMirrorFilterRule(ctx, &awsec2.CreateTrafficMirrorFilterRuleInput{
		DestinationCidrBlock:  aws.String("10.110.1.0/24"),
		RuleAction:            awsec2types.TrafficMirrorRuleActionAccept,
		RuleNumber:            aws.Int32(1),
		SourceCidrBlock:       aws.String("10.110.0.0/24"),
		TrafficDirection:      awsec2types.TrafficDirectionIngress,
		TrafficMirrorFilterId: createTrafficMirrorFilterOut.TrafficMirrorFilter.TrafficMirrorFilterId,
	})
	if err != nil {
		t.Fatalf("create traffic mirror filter rule: %v", err)
	}
	if createTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule == nil || !strings.HasPrefix(aws.ToString(createTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule.TrafficMirrorFilterRuleId), "tmfr-") {
		t.Fatalf("unexpected traffic mirror filter rule output: %#v", createTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule)
	}

	createTrafficMirrorSessionOut, err := client.CreateTrafficMirrorSession(ctx, &awsec2.CreateTrafficMirrorSessionInput{
		NetworkInterfaceId:    aws.String("eni-00000000000000110"),
		SessionNumber:         aws.Int32(1),
		TrafficMirrorFilterId: createTrafficMirrorFilterOut.TrafficMirrorFilter.TrafficMirrorFilterId,
		TrafficMirrorTargetId: createTrafficMirrorTargetOut.TrafficMirrorTarget.TrafficMirrorTargetId,
	})
	if err != nil {
		t.Fatalf("create traffic mirror session: %v", err)
	}
	if createTrafficMirrorSessionOut.TrafficMirrorSession == nil || !strings.HasPrefix(aws.ToString(createTrafficMirrorSessionOut.TrafficMirrorSession.TrafficMirrorSessionId), "tms-") {
		t.Fatalf("unexpected traffic mirror session output: %#v", createTrafficMirrorSessionOut.TrafficMirrorSession)
	}

	createCarrierGatewayOut, err := client.CreateCarrierGateway(ctx, &awsec2.CreateCarrierGatewayInput{
		VpcId: aws.String("vpc-00000001"),
	})
	if err != nil {
		t.Fatalf("create carrier gateway: %v", err)
	}
	if createCarrierGatewayOut.CarrierGateway == nil {
		t.Fatalf("expected carrier gateway output")
	}

	deleteCarrierGatewayOut, err := client.DeleteCarrierGateway(ctx, &awsec2.DeleteCarrierGatewayInput{
		CarrierGatewayId: createCarrierGatewayOut.CarrierGateway.CarrierGatewayId,
	})
	if err != nil {
		t.Fatalf("delete carrier gateway: %v", err)
	}
	if deleteCarrierGatewayOut.CarrierGateway == nil || aws.ToString(deleteCarrierGatewayOut.CarrierGateway.CarrierGatewayId) != aws.ToString(createCarrierGatewayOut.CarrierGateway.CarrierGatewayId) {
		t.Fatalf("unexpected delete carrier gateway output: %#v", deleteCarrierGatewayOut.CarrierGateway)
	}

	createCoipPoolOut, err := client.CreateCoipPool(ctx, &awsec2.CreateCoipPoolInput{
		LocalGatewayRouteTableId: aws.String("lgw-rtb-00000000110"),
	})
	if err != nil {
		t.Fatalf("create coip pool: %v", err)
	}
	if createCoipPoolOut.CoipPool == nil {
		t.Fatalf("expected coip pool output")
	}

	createCoipCidrOut, err := client.CreateCoipCidr(ctx, &awsec2.CreateCoipCidrInput{
		Cidr:       aws.String("10.110.2.0/24"),
		CoipPoolId: createCoipPoolOut.CoipPool.PoolId,
	})
	if err != nil {
		t.Fatalf("create coip cidr: %v", err)
	}
	if createCoipCidrOut.CoipCidr == nil {
		t.Fatalf("expected coip cidr output")
	}

	deleteCoipCidrOut, err := client.DeleteCoipCidr(ctx, &awsec2.DeleteCoipCidrInput{
		Cidr:       aws.String("10.110.2.0/24"),
		CoipPoolId: createCoipPoolOut.CoipPool.PoolId,
	})
	if err != nil {
		t.Fatalf("delete coip cidr: %v", err)
	}
	if deleteCoipCidrOut.CoipCidr == nil || aws.ToString(deleteCoipCidrOut.CoipCidr.CoipPoolId) != aws.ToString(createCoipPoolOut.CoipPool.PoolId) {
		t.Fatalf("unexpected delete coip cidr output: %#v", deleteCoipCidrOut.CoipCidr)
	}

	deleteCoipPoolOut, err := client.DeleteCoipPool(ctx, &awsec2.DeleteCoipPoolInput{
		CoipPoolId: createCoipPoolOut.CoipPool.PoolId,
	})
	if err != nil {
		t.Fatalf("delete coip pool: %v", err)
	}
	if deleteCoipPoolOut.CoipPool == nil || aws.ToString(deleteCoipPoolOut.CoipPool.PoolId) != aws.ToString(createCoipPoolOut.CoipPool.PoolId) {
		t.Fatalf("unexpected delete coip pool output: %#v", deleteCoipPoolOut.CoipPool)
	}

	createFleetOut, err := client.CreateFleet(ctx, &awsec2.CreateFleetInput{
		LaunchTemplateConfigs: []awsec2types.FleetLaunchTemplateConfigRequest{
			{
				LaunchTemplateSpecification: &awsec2types.FleetLaunchTemplateSpecificationRequest{
					LaunchTemplateId: aws.String("lt-00000000000000110"),
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

	deleteFleetsOut, err := client.DeleteFleets(ctx, &awsec2.DeleteFleetsInput{
		FleetIds:           []string{aws.ToString(createFleetOut.FleetId)},
		TerminateInstances: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("delete fleets: %v", err)
	}
	if len(deleteFleetsOut.SuccessfulFleetDeletions) != 1 || aws.ToString(deleteFleetsOut.SuccessfulFleetDeletions[0].FleetId) != aws.ToString(createFleetOut.FleetId) {
		t.Fatalf("unexpected delete fleets output: %#v", deleteFleetsOut)
	}
}

func TestEC2Stage110ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateSpotDatafeedSubscription",
		"CreateStoreImageTask",
		"CreateTrafficMirrorFilter",
		"CreateTrafficMirrorFilterRule",
		"CreateTrafficMirrorSession",
		"CreateTrafficMirrorTarget",
		"DeleteCarrierGateway",
		"DeleteCoipCidr",
		"DeleteCoipPool",
		"DeleteFleets",
	}

	paramsByAction := map[string]map[string]string{
		"CreateSpotDatafeedSubscription": {
			"Bucket": "stage110-datafeed-bucket",
		},
		"CreateStoreImageTask": {
			"Bucket":  "stage110-store-bucket",
			"ImageId": "ami-00000000000000110",
		},
		"CreateTrafficMirrorFilter": {
			"Description": "stage110-filter",
		},
		"CreateTrafficMirrorFilterRule": {
			"DestinationCidrBlock":  "10.110.1.0/24",
			"RuleAction":            "accept",
			"RuleNumber":            "1",
			"SourceCidrBlock":       "10.110.0.0/24",
			"TrafficDirection":      "ingress",
			"TrafficMirrorFilterId": "tmf-00000000110",
		},
		"CreateTrafficMirrorSession": {
			"NetworkInterfaceId":    "eni-00000000110",
			"SessionNumber":         "1",
			"TrafficMirrorFilterId": "tmf-00000000110",
			"TrafficMirrorTargetId": "tmt-00000000110",
		},
		"CreateTrafficMirrorTarget": {
			"NetworkInterfaceId": "eni-00000000110",
		},
		"DeleteCarrierGateway": {
			"CarrierGatewayId": "cagw-00000000110",
		},
		"DeleteCoipCidr": {
			"Cidr":       "10.110.2.0/24",
			"CoipPoolId": "coip-pool-00000000110",
		},
		"DeleteCoipPool": {
			"CoipPoolId": "coip-pool-00000000110",
		},
		"DeleteFleets": {
			"FleetId.1":          "fleet-00000000110",
			"TerminateInstances": "true",
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
