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

func TestEC2Stage103SDKLifecycle(t *testing.T) {
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

	createOut, err := client.CreateCapacityReservation(ctx, &awsec2.CreateCapacityReservationInput{
		InstanceCount:    aws.Int32(3),
		InstancePlatform: awsec2types.CapacityReservationInstancePlatformLinuxUnix,
		InstanceType:     aws.String("m5.large"),
		AvailabilityZone: aws.String("us-east-1a"),
	})
	if err != nil {
		t.Fatalf("create capacity reservation: %v", err)
	}
	if createOut.CapacityReservation == nil || createOut.CapacityReservation.CapacityReservationId == nil {
		t.Fatalf("expected created capacity reservation id")
	}
	sourceID := aws.ToString(createOut.CapacityReservation.CapacityReservationId)

	splitOut, err := client.CreateCapacityReservationBySplitting(ctx, &awsec2.CreateCapacityReservationBySplittingInput{
		SourceCapacityReservationId: aws.String(sourceID),
		InstanceCount:               aws.Int32(1),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeCapacityReservation,
				Tags: []awsec2types.Tag{
					{Key: aws.String("team"), Value: aws.String("stage103")},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("create capacity reservation by splitting: %v", err)
	}
	if splitOut.InstanceCount == nil || *splitOut.InstanceCount != 1 {
		t.Fatalf("unexpected split instance count: %v", splitOut.InstanceCount)
	}
	if splitOut.SourceCapacityReservation == nil {
		t.Fatalf("expected source capacity reservation")
	}
	if splitOut.SourceCapacityReservation.TotalInstanceCount == nil || *splitOut.SourceCapacityReservation.TotalInstanceCount != 2 {
		t.Fatalf("unexpected source total instance count: %v", splitOut.SourceCapacityReservation.TotalInstanceCount)
	}
	if splitOut.DestinationCapacityReservation == nil {
		t.Fatalf("expected destination capacity reservation")
	}
	if !strings.HasPrefix(aws.ToString(splitOut.DestinationCapacityReservation.CapacityReservationId), "cr-") {
		t.Fatalf("unexpected destination reservation id: %q", aws.ToString(splitOut.DestinationCapacityReservation.CapacityReservationId))
	}
	if splitOut.DestinationCapacityReservation.TotalInstanceCount == nil || *splitOut.DestinationCapacityReservation.TotalInstanceCount != 1 {
		t.Fatalf("unexpected destination total instance count: %v", splitOut.DestinationCapacityReservation.TotalInstanceCount)
	}
	if len(splitOut.DestinationCapacityReservation.Tags) != 1 {
		t.Fatalf("expected 1 destination tag, got %d", len(splitOut.DestinationCapacityReservation.Tags))
	}
}

func TestEC2Stage103ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateCapacityReservationBySplitting",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"SourceCapacityReservationId": "cr-00000000000000103",
			"InstanceCount":               "1",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
