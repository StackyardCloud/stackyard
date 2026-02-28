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

func TestEC2Stage104SDKLifecycle(t *testing.T) {
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

	out, err := client.CreateCapacityReservationFleet(ctx, &awsec2.CreateCapacityReservationFleetInput{
		InstanceTypeSpecifications: []awsec2types.ReservationFleetInstanceSpecification{
			{
				AvailabilityZone: aws.String("us-east-1a"),
				InstancePlatform: awsec2types.CapacityReservationInstancePlatformLinuxUnix,
				InstanceType:     awsec2types.InstanceTypeM5Large,
			},
		},
		TotalTargetCapacity: aws.Int32(2),
	})
	if err != nil {
		t.Fatalf("create capacity reservation fleet: %v", err)
	}
	if !strings.HasPrefix(aws.ToString(out.CapacityReservationFleetId), "crf-") {
		t.Fatalf("unexpected capacity reservation fleet id: %q", aws.ToString(out.CapacityReservationFleetId))
	}
	if out.State != awsec2types.CapacityReservationFleetStateActive {
		t.Fatalf("unexpected capacity reservation fleet state: %q", out.State)
	}
	if out.TotalTargetCapacity == nil || *out.TotalTargetCapacity != 2 {
		t.Fatalf("unexpected total target capacity: %v", out.TotalTargetCapacity)
	}
	if out.TotalFulfilledCapacity == nil || *out.TotalFulfilledCapacity != 2 {
		t.Fatalf("unexpected total fulfilled capacity: %v", out.TotalFulfilledCapacity)
	}
	if len(out.FleetCapacityReservations) != 1 {
		t.Fatalf("expected 1 fleet capacity reservation, got %d", len(out.FleetCapacityReservations))
	}
	if !strings.HasPrefix(aws.ToString(out.FleetCapacityReservations[0].CapacityReservationId), "cr-") {
		t.Fatalf("unexpected fleet reservation id: %q", aws.ToString(out.FleetCapacityReservations[0].CapacityReservationId))
	}
}

func TestEC2Stage104ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateCapacityReservationFleet",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"InstanceTypeSpecification.1.InstanceType":     "m5.large",
			"InstanceTypeSpecification.1.InstancePlatform": "Linux/UNIX",
			"TotalTargetCapacity":                          "2",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
