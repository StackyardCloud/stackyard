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

func TestEC2Stage76SDKLifecycle(t *testing.T) {
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

	_, err = client.EnableAwsNetworkPerformanceMetricSubscription(ctx, &awsec2.EnableAwsNetworkPerformanceMetricSubscriptionInput{
		Source:      aws.String("us-east-1"),
		Destination: aws.String("us-west-2"),
		Metric:      awsec2types.MetricTypeAggregateLatency,
		Statistic:   awsec2types.StatisticTypeP50,
	})
	if err != nil {
		t.Fatalf("enable aws network performance metric subscription #1: %v", err)
	}

	_, err = client.EnableAwsNetworkPerformanceMetricSubscription(ctx, &awsec2.EnableAwsNetworkPerformanceMetricSubscriptionInput{
		Source:      aws.String("us-west-2"),
		Destination: aws.String("us-east-1"),
		Metric:      awsec2types.MetricTypeAggregateLatency,
		Statistic:   awsec2types.StatisticTypeP50,
	})
	if err != nil {
		t.Fatalf("enable aws network performance metric subscription #2: %v", err)
	}

	describeOut, err := client.DescribeAwsNetworkPerformanceMetricSubscriptions(ctx, &awsec2.DescribeAwsNetworkPerformanceMetricSubscriptionsInput{})
	if err != nil {
		t.Fatalf("describe aws network performance metric subscriptions: %v", err)
	}
	if len(describeOut.Subscriptions) != 2 {
		t.Fatalf("expected two subscriptions, got %d", len(describeOut.Subscriptions))
	}
	for _, subscription := range describeOut.Subscriptions {
		if subscription.Period != awsec2types.PeriodTypeFiveMinutes {
			t.Fatalf("unexpected period in subscription: %q", subscription.Period)
		}
		if subscription.Metric != awsec2types.MetricTypeAggregateLatency {
			t.Fatalf("unexpected metric in subscription: %q", subscription.Metric)
		}
		if subscription.Statistic != awsec2types.StatisticTypeP50 {
			t.Fatalf("unexpected statistic in subscription: %q", subscription.Statistic)
		}
	}

	filteredOut, err := client.DescribeAwsNetworkPerformanceMetricSubscriptions(ctx, &awsec2.DescribeAwsNetworkPerformanceMetricSubscriptionsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("source"), Values: []string{"us-east-1"}},
		},
	})
	if err != nil {
		t.Fatalf("describe aws network performance metric subscriptions with source filter: %v", err)
	}
	if len(filteredOut.Subscriptions) != 1 {
		t.Fatalf("expected one filtered subscription, got %d", len(filteredOut.Subscriptions))
	}
	if aws.ToString(filteredOut.Subscriptions[0].Source) != "us-east-1" {
		t.Fatalf("unexpected filtered subscription source: %q", aws.ToString(filteredOut.Subscriptions[0].Source))
	}

	firstPageOut, err := client.DescribeAwsNetworkPerformanceMetricSubscriptions(ctx, &awsec2.DescribeAwsNetworkPerformanceMetricSubscriptionsInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe aws network performance metric subscriptions first page: %v", err)
	}
	if len(firstPageOut.Subscriptions) != 1 {
		t.Fatalf("expected one first-page subscription, got %d", len(firstPageOut.Subscriptions))
	}
	if firstPageOut.NextToken == nil || *firstPageOut.NextToken == "" {
		t.Fatalf("expected next token on first page")
	}

	secondPageOut, err := client.DescribeAwsNetworkPerformanceMetricSubscriptions(ctx, &awsec2.DescribeAwsNetworkPerformanceMetricSubscriptionsInput{
		MaxResults: aws.Int32(1),
		NextToken:  firstPageOut.NextToken,
	})
	if err != nil {
		t.Fatalf("describe aws network performance metric subscriptions second page: %v", err)
	}
	if len(secondPageOut.Subscriptions) != 1 {
		t.Fatalf("expected one second-page subscription, got %d", len(secondPageOut.Subscriptions))
	}

	_, err = client.DisableAwsNetworkPerformanceMetricSubscription(ctx, &awsec2.DisableAwsNetworkPerformanceMetricSubscriptionInput{
		Source:      aws.String("us-east-1"),
		Destination: aws.String("us-west-2"),
		Metric:      awsec2types.MetricTypeAggregateLatency,
		Statistic:   awsec2types.StatisticTypeP50,
	})
	if err != nil {
		t.Fatalf("disable aws network performance metric subscription: %v", err)
	}

	afterDisableOut, err := client.DescribeAwsNetworkPerformanceMetricSubscriptions(ctx, &awsec2.DescribeAwsNetworkPerformanceMetricSubscriptionsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("source"), Values: []string{"us-east-1"}},
		},
	})
	if err != nil {
		t.Fatalf("describe aws network performance metric subscriptions after disable: %v", err)
	}
	if len(afterDisableOut.Subscriptions) != 0 {
		t.Fatalf("expected no subscriptions for source us-east-1 after disable, got %d", len(afterDisableOut.Subscriptions))
	}
}

func TestEC2Stage76ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeAwsNetworkPerformanceMetricSubscriptions",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		if action == "DescribeAwsNetworkPerformanceMetricSubscriptions" {
			params["Filter.1.Name"] = "source"
			params["Filter.1.Value.1"] = "us-east-1"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
