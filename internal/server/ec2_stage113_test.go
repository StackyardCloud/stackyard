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

func TestEC2Stage113SDKLifecycle(t *testing.T) {
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

	createNetworkInsightsPathOut, err := client.CreateNetworkInsightsPath(ctx, &awsec2.CreateNetworkInsightsPathInput{
		ClientToken: aws.String("stage113-path-token"),
		Protocol:    awsec2types.ProtocolTcp,
		Source:      aws.String("eni-00000000000000113"),
	})
	if err != nil {
		t.Fatalf("create network insights path: %v", err)
	}
	deleteNetworkInsightsPathOut, err := client.DeleteNetworkInsightsPath(ctx, &awsec2.DeleteNetworkInsightsPathInput{
		NetworkInsightsPathId: createNetworkInsightsPathOut.NetworkInsightsPath.NetworkInsightsPathId,
	})
	if err != nil {
		t.Fatalf("delete network insights path: %v", err)
	}
	if aws.ToString(deleteNetworkInsightsPathOut.NetworkInsightsPathId) != aws.ToString(createNetworkInsightsPathOut.NetworkInsightsPath.NetworkInsightsPathId) {
		t.Fatalf("unexpected delete network insights path output: %#v", deleteNetworkInsightsPathOut.NetworkInsightsPathId)
	}

	deleteNetworkInsightsAnalysisOut, err := client.DeleteNetworkInsightsAnalysis(ctx, &awsec2.DeleteNetworkInsightsAnalysisInput{
		NetworkInsightsAnalysisId: aws.String("nia-00000000113"),
	})
	if err != nil {
		t.Fatalf("delete network insights analysis: %v", err)
	}
	if aws.ToString(deleteNetworkInsightsAnalysisOut.NetworkInsightsAnalysisId) != "nia-00000000113" {
		t.Fatalf("unexpected delete network insights analysis output: %#v", deleteNetworkInsightsAnalysisOut.NetworkInsightsAnalysisId)
	}

	createPublicIpv4PoolOut, err := client.CreatePublicIpv4Pool(ctx, &awsec2.CreatePublicIpv4PoolInput{})
	if err != nil {
		t.Fatalf("create public ipv4 pool: %v", err)
	}
	deletePublicIpv4PoolOut, err := client.DeletePublicIpv4Pool(ctx, &awsec2.DeletePublicIpv4PoolInput{PoolId: createPublicIpv4PoolOut.PoolId})
	if err != nil {
		t.Fatalf("delete public ipv4 pool: %v", err)
	}
	if !aws.ToBool(deletePublicIpv4PoolOut.ReturnValue) {
		t.Fatalf("expected delete public ipv4 pool returnValue=true")
	}

	deleteQueuedReservedInstancesOut, err := client.DeleteQueuedReservedInstances(ctx, &awsec2.DeleteQueuedReservedInstancesInput{
		ReservedInstancesIds: []string{"ri-00000000113", "invalid-id"},
	})
	if err != nil {
		t.Fatalf("delete queued reserved instances: %v", err)
	}
	if len(deleteQueuedReservedInstancesOut.SuccessfulQueuedPurchaseDeletions) != 1 ||
		aws.ToString(deleteQueuedReservedInstancesOut.SuccessfulQueuedPurchaseDeletions[0].ReservedInstancesId) != "ri-00000000113" {
		t.Fatalf("unexpected successful queued purchase deletions: %#v", deleteQueuedReservedInstancesOut.SuccessfulQueuedPurchaseDeletions)
	}
	if len(deleteQueuedReservedInstancesOut.FailedQueuedPurchaseDeletions) != 1 ||
		deleteQueuedReservedInstancesOut.FailedQueuedPurchaseDeletions[0].Error == nil ||
		deleteQueuedReservedInstancesOut.FailedQueuedPurchaseDeletions[0].Error.Code != awsec2types.DeleteQueuedReservedInstancesErrorCodeReservedInstancesIdInvalid {
		t.Fatalf("unexpected failed queued purchase deletions: %#v", deleteQueuedReservedInstancesOut.FailedQueuedPurchaseDeletions)
	}

	if _, err := client.CreateSpotDatafeedSubscription(ctx, &awsec2.CreateSpotDatafeedSubscriptionInput{
		Bucket: aws.String("stage113-datafeed-bucket"),
	}); err != nil {
		t.Fatalf("create spot datafeed subscription: %v", err)
	}
	if _, err := client.DeleteSpotDatafeedSubscription(ctx, &awsec2.DeleteSpotDatafeedSubscriptionInput{}); err != nil {
		t.Fatalf("delete spot datafeed subscription: %v", err)
	}

	createTrafficMirrorFilterOut, err := client.CreateTrafficMirrorFilter(ctx, &awsec2.CreateTrafficMirrorFilterInput{Description: aws.String("stage113-filter")})
	if err != nil {
		t.Fatalf("create traffic mirror filter: %v", err)
	}
	createTrafficMirrorTargetOut, err := client.CreateTrafficMirrorTarget(ctx, &awsec2.CreateTrafficMirrorTargetInput{NetworkInterfaceId: aws.String("eni-00000000000000113")})
	if err != nil {
		t.Fatalf("create traffic mirror target: %v", err)
	}
	createTrafficMirrorFilterRuleOut, err := client.CreateTrafficMirrorFilterRule(ctx, &awsec2.CreateTrafficMirrorFilterRuleInput{
		DestinationCidrBlock:  aws.String("10.113.1.0/24"),
		RuleAction:            awsec2types.TrafficMirrorRuleActionAccept,
		RuleNumber:            aws.Int32(1),
		SourceCidrBlock:       aws.String("10.113.0.0/24"),
		TrafficDirection:      awsec2types.TrafficDirectionIngress,
		TrafficMirrorFilterId: createTrafficMirrorFilterOut.TrafficMirrorFilter.TrafficMirrorFilterId,
	})
	if err != nil {
		t.Fatalf("create traffic mirror filter rule: %v", err)
	}
	createTrafficMirrorSessionOut, err := client.CreateTrafficMirrorSession(ctx, &awsec2.CreateTrafficMirrorSessionInput{
		NetworkInterfaceId:    aws.String("eni-00000000000000113"),
		SessionNumber:         aws.Int32(1),
		TrafficMirrorFilterId: createTrafficMirrorFilterOut.TrafficMirrorFilter.TrafficMirrorFilterId,
		TrafficMirrorTargetId: createTrafficMirrorTargetOut.TrafficMirrorTarget.TrafficMirrorTargetId,
	})
	if err != nil {
		t.Fatalf("create traffic mirror session: %v", err)
	}

	deleteTrafficMirrorFilterRuleOut, err := client.DeleteTrafficMirrorFilterRule(ctx, &awsec2.DeleteTrafficMirrorFilterRuleInput{
		TrafficMirrorFilterRuleId: createTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule.TrafficMirrorFilterRuleId,
	})
	if err != nil {
		t.Fatalf("delete traffic mirror filter rule: %v", err)
	}
	if aws.ToString(deleteTrafficMirrorFilterRuleOut.TrafficMirrorFilterRuleId) != aws.ToString(createTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule.TrafficMirrorFilterRuleId) {
		t.Fatalf("unexpected delete traffic mirror filter rule output: %#v", deleteTrafficMirrorFilterRuleOut.TrafficMirrorFilterRuleId)
	}

	deleteTrafficMirrorSessionOut, err := client.DeleteTrafficMirrorSession(ctx, &awsec2.DeleteTrafficMirrorSessionInput{
		TrafficMirrorSessionId: createTrafficMirrorSessionOut.TrafficMirrorSession.TrafficMirrorSessionId,
	})
	if err != nil {
		t.Fatalf("delete traffic mirror session: %v", err)
	}
	if aws.ToString(deleteTrafficMirrorSessionOut.TrafficMirrorSessionId) != aws.ToString(createTrafficMirrorSessionOut.TrafficMirrorSession.TrafficMirrorSessionId) {
		t.Fatalf("unexpected delete traffic mirror session output: %#v", deleteTrafficMirrorSessionOut.TrafficMirrorSessionId)
	}

	deleteTrafficMirrorTargetOut, err := client.DeleteTrafficMirrorTarget(ctx, &awsec2.DeleteTrafficMirrorTargetInput{
		TrafficMirrorTargetId: createTrafficMirrorTargetOut.TrafficMirrorTarget.TrafficMirrorTargetId,
	})
	if err != nil {
		t.Fatalf("delete traffic mirror target: %v", err)
	}
	if aws.ToString(deleteTrafficMirrorTargetOut.TrafficMirrorTargetId) != aws.ToString(createTrafficMirrorTargetOut.TrafficMirrorTarget.TrafficMirrorTargetId) {
		t.Fatalf("unexpected delete traffic mirror target output: %#v", deleteTrafficMirrorTargetOut.TrafficMirrorTargetId)
	}

	deleteTrafficMirrorFilterOut, err := client.DeleteTrafficMirrorFilter(ctx, &awsec2.DeleteTrafficMirrorFilterInput{
		TrafficMirrorFilterId: createTrafficMirrorFilterOut.TrafficMirrorFilter.TrafficMirrorFilterId,
	})
	if err != nil {
		t.Fatalf("delete traffic mirror filter: %v", err)
	}
	if aws.ToString(deleteTrafficMirrorFilterOut.TrafficMirrorFilterId) != aws.ToString(createTrafficMirrorFilterOut.TrafficMirrorFilter.TrafficMirrorFilterId) {
		t.Fatalf("unexpected delete traffic mirror filter output: %#v", deleteTrafficMirrorFilterOut.TrafficMirrorFilterId)
	}

	if _, err := client.AdvertiseByoipCidr(ctx, &awsec2.AdvertiseByoipCidrInput{Cidr: aws.String("198.51.113.0/24")}); err != nil {
		t.Fatalf("advertise byoip cidr: %v", err)
	}
	deprovisionByoipCidrOut, err := client.DeprovisionByoipCidr(ctx, &awsec2.DeprovisionByoipCidrInput{Cidr: aws.String("198.51.113.0/24")})
	if err != nil {
		t.Fatalf("deprovision byoip cidr: %v", err)
	}
	if deprovisionByoipCidrOut.ByoipCidr == nil ||
		aws.ToString(deprovisionByoipCidrOut.ByoipCidr.Cidr) != "198.51.113.0/24" ||
		string(deprovisionByoipCidrOut.ByoipCidr.State) != "deprovisioned" {
		t.Fatalf("unexpected deprovision byoip cidr output: %#v", deprovisionByoipCidrOut.ByoipCidr)
	}
}

func TestEC2Stage113ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DeleteNetworkInsightsAnalysis",
		"DeleteNetworkInsightsPath",
		"DeletePublicIpv4Pool",
		"DeleteQueuedReservedInstances",
		"DeleteSpotDatafeedSubscription",
		"DeleteTrafficMirrorFilter",
		"DeleteTrafficMirrorFilterRule",
		"DeleteTrafficMirrorSession",
		"DeleteTrafficMirrorTarget",
		"DeprovisionByoipCidr",
	}

	paramsByAction := map[string]map[string]string{
		"DeleteNetworkInsightsAnalysis": {
			"NetworkInsightsAnalysisId": "nia-00000000113",
		},
		"DeleteNetworkInsightsPath": {
			"NetworkInsightsPathId": "nip-00000000113",
		},
		"DeletePublicIpv4Pool": {
			"PoolId": "ipv4pool-ec2-00000000113",
		},
		"DeleteQueuedReservedInstances": {
			"ReservedInstancesId.1": "ri-00000000113",
		},
		"DeleteSpotDatafeedSubscription": {},
		"DeleteTrafficMirrorFilter": {
			"TrafficMirrorFilterId": "tmf-00000000113",
		},
		"DeleteTrafficMirrorFilterRule": {
			"TrafficMirrorFilterRuleId": "tmfr-00000000113",
		},
		"DeleteTrafficMirrorSession": {
			"TrafficMirrorSessionId": "tms-00000000113",
		},
		"DeleteTrafficMirrorTarget": {
			"TrafficMirrorTargetId": "tmt-00000000113",
		},
		"DeprovisionByoipCidr": {
			"Cidr": "198.51.113.0/24",
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
