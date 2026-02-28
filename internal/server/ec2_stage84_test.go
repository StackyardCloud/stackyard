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
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage84SDKLifecycle(t *testing.T) {
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

	eventWindowID := "iew-00000000000000084"
	instanceID := "i-00000000000000084"

	out, err := client.AssociateInstanceEventWindow(ctx, &awsec2.AssociateInstanceEventWindowInput{
		InstanceEventWindowId: aws.String(eventWindowID),
		AssociationTarget: &types.InstanceEventWindowAssociationRequest{
			InstanceIds: []string{instanceID},
		},
	})
	if err != nil {
		t.Fatalf("associate instance event window: %v", err)
	}
	if out.InstanceEventWindow == nil {
		t.Fatalf("expected instanceEventWindow")
	}
	if aws.ToString(out.InstanceEventWindow.InstanceEventWindowId) != eventWindowID {
		t.Fatalf("expected instance event window id %q, got %q", eventWindowID, aws.ToString(out.InstanceEventWindow.InstanceEventWindowId))
	}
	if got := out.InstanceEventWindow.AssociationTarget; got == nil || len(got.InstanceIds) != 1 || got.InstanceIds[0] != instanceID {
		t.Fatalf("expected associated instance id %q, got %#v", instanceID, got)
	}
}

func TestEC2Stage84ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AssociateInstanceEventWindow",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"InstanceEventWindowId":          "iew-00000000000000084",
			"AssociationTarget.InstanceId.1": "i-00000000000000084",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
