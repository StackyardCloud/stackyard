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

func TestEC2Stage57SDKLifecycle(t *testing.T) {
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

	createOut, err := client.CreateVpcEndpointConnectionNotification(ctx, &awsec2.CreateVpcEndpointConnectionNotificationInput{
		ServiceId:                 aws.String("vpce-svc-00000000"),
		ConnectionNotificationArn: aws.String("arn:aws:sns:us-east-1:123456789012:stage57-topic"),
		ConnectionEvents:          []string{"Accept"},
	})
	if err != nil {
		t.Fatalf("create vpc endpoint connection notification: %v", err)
	}
	if createOut.ConnectionNotification == nil || createOut.ConnectionNotification.ConnectionNotificationId == nil {
		t.Fatalf("expected created connection notification id")
	}

	deleteOut, err := client.DeleteVpcEndpointConnectionNotifications(ctx, &awsec2.DeleteVpcEndpointConnectionNotificationsInput{
		ConnectionNotificationIds: []string{
			aws.ToString(createOut.ConnectionNotification.ConnectionNotificationId),
			"vpce-nfn-does-not-exist",
		},
	})
	if err != nil {
		t.Fatalf("delete vpc endpoint connection notifications: %v", err)
	}
	if len(deleteOut.Unsuccessful) != 1 {
		t.Fatalf("expected one unsuccessful item, got %d", len(deleteOut.Unsuccessful))
	}
	if aws.ToString(deleteOut.Unsuccessful[0].ResourceId) != "vpce-nfn-does-not-exist" {
		t.Fatalf("unexpected unsuccessful resource id: %q", aws.ToString(deleteOut.Unsuccessful[0].ResourceId))
	}
}

func TestEC2Stage57ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DeleteVpcEndpointConnectionNotifications",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"ConnectionNotificationId.1": "vpce-nfn-00000000",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
