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

func TestEC2Stage75SDKLifecycle(t *testing.T) {
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
		VolumeType:       awsec2types.VolumeTypeGp3,
	})
	if err != nil || createVolumeOut.VolumeId == nil {
		t.Fatalf("create volume: %v", err)
	}
	volumeID := aws.ToString(createVolumeOut.VolumeId)

	if _, err := client.ModifyVolumeAttribute(ctx, &awsec2.ModifyVolumeAttributeInput{
		VolumeId:     aws.String(volumeID),
		AutoEnableIO: &awsec2types.AttributeBooleanValue{Value: aws.Bool(false)},
	}); err != nil {
		t.Fatalf("modify volume attribute before enable volume io: %v", err)
	}

	if _, err := client.EnableVolumeIO(ctx, &awsec2.EnableVolumeIOInput{VolumeId: aws.String(volumeID)}); err != nil {
		t.Fatalf("enable volume io: %v", err)
	}

	enableReachabilityOut, err := client.EnableReachabilityAnalyzerOrganizationSharing(ctx, &awsec2.EnableReachabilityAnalyzerOrganizationSharingInput{})
	if err != nil {
		t.Fatalf("enable reachability analyzer organization sharing: %v", err)
	}
	if !aws.ToBool(enableReachabilityOut.ReturnValue) {
		t.Fatalf("expected enable reachability analyzer organization sharing return value true")
	}

	enableSubscriptionOut, err := client.EnableAwsNetworkPerformanceMetricSubscription(ctx, &awsec2.EnableAwsNetworkPerformanceMetricSubscriptionInput{
		Source:      aws.String("us-east-1"),
		Destination: aws.String("us-west-2"),
		Metric:      awsec2types.MetricTypeAggregateLatency,
		Statistic:   awsec2types.StatisticTypeP50,
	})
	if err != nil {
		t.Fatalf("enable aws network performance metric subscription: %v", err)
	}
	if !aws.ToBool(enableSubscriptionOut.Output) {
		t.Fatalf("expected enable aws network performance metric subscription output true")
	}

	startTime := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Second)
	endTime := startTime.Add(5 * time.Minute)

	getDataOut, err := client.GetAwsNetworkPerformanceData(ctx, &awsec2.GetAwsNetworkPerformanceDataInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
		DataQueries: []awsec2types.DataQuery{
			{
				Id:          aws.String("query-1"),
				Source:      aws.String("us-east-1"),
				Destination: aws.String("us-west-2"),
				Metric:      awsec2types.MetricTypeAggregateLatency,
				Period:      awsec2types.PeriodTypeFiveMinutes,
				Statistic:   awsec2types.StatisticTypeP50,
			},
		},
	})
	if err != nil {
		t.Fatalf("get aws network performance data: %v", err)
	}
	if len(getDataOut.DataResponses) != 1 {
		t.Fatalf("expected one data response, got %d", len(getDataOut.DataResponses))
	}
	if aws.ToString(getDataOut.DataResponses[0].Id) != "query-1" {
		t.Fatalf("unexpected data response id: %q", aws.ToString(getDataOut.DataResponses[0].Id))
	}
	if len(getDataOut.DataResponses[0].MetricPoints) != 1 {
		t.Fatalf("expected one metric point, got %d", len(getDataOut.DataResponses[0].MetricPoints))
	}
	if aws.ToString(getDataOut.DataResponses[0].MetricPoints[0].Status) != "healthy" {
		t.Fatalf("unexpected metric point status before disable: %q", aws.ToString(getDataOut.DataResponses[0].MetricPoints[0].Status))
	}

	disableSubscriptionOut, err := client.DisableAwsNetworkPerformanceMetricSubscription(ctx, &awsec2.DisableAwsNetworkPerformanceMetricSubscriptionInput{
		Source:      aws.String("us-east-1"),
		Destination: aws.String("us-west-2"),
		Metric:      awsec2types.MetricTypeAggregateLatency,
		Statistic:   awsec2types.StatisticTypeP50,
	})
	if err != nil {
		t.Fatalf("disable aws network performance metric subscription: %v", err)
	}
	if !aws.ToBool(disableSubscriptionOut.Output) {
		t.Fatalf("expected disable aws network performance metric subscription output true")
	}

	getDataAfterDisableOut, err := client.GetAwsNetworkPerformanceData(ctx, &awsec2.GetAwsNetworkPerformanceDataInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
		DataQueries: []awsec2types.DataQuery{
			{
				Id:          aws.String("query-1"),
				Source:      aws.String("us-east-1"),
				Destination: aws.String("us-west-2"),
				Metric:      awsec2types.MetricTypeAggregateLatency,
				Period:      awsec2types.PeriodTypeFiveMinutes,
				Statistic:   awsec2types.StatisticTypeP50,
			},
		},
	})
	if err != nil {
		t.Fatalf("get aws network performance data after disable: %v", err)
	}
	if len(getDataAfterDisableOut.DataResponses) != 1 || len(getDataAfterDisableOut.DataResponses[0].MetricPoints) != 1 {
		t.Fatalf("unexpected data response shape after disable")
	}
	if aws.ToString(getDataAfterDisableOut.DataResponses[0].MetricPoints[0].Status) != "degraded" {
		t.Fatalf("unexpected metric point status after disable: %q", aws.ToString(getDataAfterDisableOut.DataResponses[0].MetricPoints[0].Status))
	}
}

func TestEC2Stage75ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DisableAwsNetworkPerformanceMetricSubscription",
		"EnableAwsNetworkPerformanceMetricSubscription",
		"EnableReachabilityAnalyzerOrganizationSharing",
		"EnableVolumeIO",
		"GetAwsNetworkPerformanceData",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "DisableAwsNetworkPerformanceMetricSubscription", "EnableAwsNetworkPerformanceMetricSubscription":
			params["Source"] = "us-east-1"
			params["Destination"] = "us-west-2"
			params["Metric"] = "aggregate-latency"
			params["Statistic"] = "p50"
		case "EnableVolumeIO":
			params["VolumeId"] = "vol-00000000"
		case "GetAwsNetworkPerformanceData":
			params["StartTime"] = "2026-01-01T00:00:00Z"
			params["EndTime"] = "2026-01-01T00:05:00Z"
			params["DataQuery.1.Id"] = "query-1"
			params["DataQuery.1.Source"] = "us-east-1"
			params["DataQuery.1.Destination"] = "us-west-2"
			params["DataQuery.1.Metric"] = "aggregate-latency"
			params["DataQuery.1.Statistic"] = "p50"
			params["DataQuery.1.Period"] = "five-minutes"
		}

		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
