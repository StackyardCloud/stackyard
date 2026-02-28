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

func TestEC2Stage102SDKLifecycle(t *testing.T) {
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

	out, err := client.CreateCapacityReservation(ctx, &awsec2.CreateCapacityReservationInput{
		InstanceCount:    aws.Int32(2),
		InstancePlatform: awsec2types.CapacityReservationInstancePlatformLinuxUnix,
		InstanceType:     aws.String("m5.large"),
		AvailabilityZone: aws.String("us-east-1a"),
	})
	if err != nil {
		t.Fatalf("create capacity reservation: %v", err)
	}
	if out.CapacityReservation == nil {
		t.Fatalf("expected capacity reservation")
	}
	if !strings.HasPrefix(aws.ToString(out.CapacityReservation.CapacityReservationId), "cr-") {
		t.Fatalf("unexpected capacity reservation id: %q", aws.ToString(out.CapacityReservation.CapacityReservationId))
	}
	if out.CapacityReservation.State != awsec2types.CapacityReservationStateActive {
		t.Fatalf("unexpected capacity reservation state: %q", out.CapacityReservation.State)
	}
	if out.CapacityReservation.TotalInstanceCount == nil || *out.CapacityReservation.TotalInstanceCount != 2 {
		t.Fatalf("unexpected total instance count: %v", out.CapacityReservation.TotalInstanceCount)
	}
	if out.CapacityReservation.AvailableInstanceCount == nil || *out.CapacityReservation.AvailableInstanceCount != 2 {
		t.Fatalf("unexpected available instance count: %v", out.CapacityReservation.AvailableInstanceCount)
	}
}

func TestEC2Stage102ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateCapacityReservation",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"InstanceCount":    "2",
			"InstancePlatform": "Linux/UNIX",
			"InstanceType":     "m5.large",
			"AvailabilityZone": "us-east-1a",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
