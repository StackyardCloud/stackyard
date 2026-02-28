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

func TestEC2Stage54SDKLifecycle(t *testing.T) {
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

	out, err := client.CreateVpcEndpointConnectionNotification(ctx, &awsec2.CreateVpcEndpointConnectionNotificationInput{
		ServiceId:                 aws.String("vpce-svc-00000000"),
		ConnectionNotificationArn: aws.String("arn:aws:sns:us-east-1:123456789012:stage54-topic"),
		ConnectionEvents:          []string{"Accept", "Reject"},
		ClientToken:               aws.String("stage54-token"),
	})
	if err != nil {
		t.Fatalf("create vpc endpoint connection notification: %v", err)
	}
	if out.ConnectionNotification == nil {
		t.Fatalf("expected connection notification in response")
	}
	if aws.ToString(out.ConnectionNotification.ConnectionNotificationArn) != "arn:aws:sns:us-east-1:123456789012:stage54-topic" {
		t.Fatalf("unexpected connection notification arn: %q", aws.ToString(out.ConnectionNotification.ConnectionNotificationArn))
	}
	if out.ConnectionNotification.ConnectionNotificationState != awsec2types.ConnectionNotificationStateEnabled {
		t.Fatalf("unexpected connection notification state: %q", out.ConnectionNotification.ConnectionNotificationState)
	}
	if out.ConnectionNotification.ConnectionNotificationType != awsec2types.ConnectionNotificationTypeTopic {
		t.Fatalf("unexpected connection notification type: %q", out.ConnectionNotification.ConnectionNotificationType)
	}
	if aws.ToString(out.ClientToken) != "stage54-token" {
		t.Fatalf("unexpected client token: %q", aws.ToString(out.ClientToken))
	}
}

func TestEC2Stage54ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateVpcEndpointConnectionNotification",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"ServiceId":                 "vpce-svc-00000000",
			"ConnectionNotificationArn": "arn:aws:sns:us-east-1:123456789012:stage54-topic",
			"ConnectionEvents.1":        "Accept",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
