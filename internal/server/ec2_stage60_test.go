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
)

func TestEC2Stage60SDKLifecycle(t *testing.T) {
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

	createOne, err := client.CreateVpcEndpointConnectionNotification(ctx, &awsec2.CreateVpcEndpointConnectionNotificationInput{
		ServiceId:                 aws.String("vpce-svc-00000000"),
		ConnectionNotificationArn: aws.String("arn:aws:sns:us-east-1:123456789012:stage60-topic-1"),
		ConnectionEvents:          []string{"Accept"},
	})
	if err != nil {
		t.Fatalf("create first connection notification: %v", err)
	}
	createTwo, err := client.CreateVpcEndpointConnectionNotification(ctx, &awsec2.CreateVpcEndpointConnectionNotificationInput{
		ServiceId:                 aws.String("vpce-svc-00000000"),
		ConnectionNotificationArn: aws.String("arn:aws:sns:us-east-1:123456789012:stage60-topic-2"),
		ConnectionEvents:          []string{"Reject"},
	})
	if err != nil {
		t.Fatalf("create second connection notification: %v", err)
	}

	firstPage, err := client.DescribeVpcEndpointConnectionNotifications(ctx, &awsec2.DescribeVpcEndpointConnectionNotificationsInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe connection notifications page one: %v", err)
	}
	if len(firstPage.ConnectionNotificationSet) != 1 {
		t.Fatalf("expected one notification in first page, got %d", len(firstPage.ConnectionNotificationSet))
	}
	if firstPage.NextToken == nil {
		t.Fatalf("expected next token in first page")
	}

	secondPage, err := client.DescribeVpcEndpointConnectionNotifications(ctx, &awsec2.DescribeVpcEndpointConnectionNotificationsInput{
		NextToken: firstPage.NextToken,
	})
	if err != nil {
		t.Fatalf("describe connection notifications page two: %v", err)
	}
	if len(secondPage.ConnectionNotificationSet) == 0 {
		t.Fatalf("expected at least one notification in second page")
	}

	filtered, err := client.DescribeVpcEndpointConnectionNotifications(ctx, &awsec2.DescribeVpcEndpointConnectionNotificationsInput{
		ConnectionNotificationId: createTwo.ConnectionNotification.ConnectionNotificationId,
	})
	if err != nil {
		t.Fatalf("describe connection notifications by id: %v", err)
	}
	if len(filtered.ConnectionNotificationSet) != 1 {
		t.Fatalf("expected one notification by id, got %d", len(filtered.ConnectionNotificationSet))
	}
	if aws.ToString(filtered.ConnectionNotificationSet[0].ConnectionNotificationId) != aws.ToString(createTwo.ConnectionNotification.ConnectionNotificationId) {
		t.Fatalf("unexpected notification id: %q", aws.ToString(filtered.ConnectionNotificationSet[0].ConnectionNotificationId))
	}
	if aws.ToString(createOne.ConnectionNotification.ConnectionNotificationId) == aws.ToString(createTwo.ConnectionNotification.ConnectionNotificationId) {
		t.Fatalf("expected unique connection notification ids")
	}
}

func TestEC2Stage60ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeVpcEndpointConnectionNotifications",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
