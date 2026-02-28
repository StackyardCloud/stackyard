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

func TestEC2Stage90SDKLifecycle(t *testing.T) {
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

	out, err := client.CancelCapacityReservationFleets(ctx, &awsec2.CancelCapacityReservationFleetsInput{
		CapacityReservationFleetIds: []string{"crf-00000000000000090", "crf-00000000000000091"},
	})
	if err != nil {
		t.Fatalf("cancel capacity reservation fleets: %v", err)
	}
	if len(out.SuccessfulFleetCancellations) != 2 {
		t.Fatalf("expected 2 successful fleet cancellations, got %d", len(out.SuccessfulFleetCancellations))
	}
	if len(out.FailedFleetCancellations) != 0 {
		t.Fatalf("expected 0 failed fleet cancellations, got %d", len(out.FailedFleetCancellations))
	}

	first := out.SuccessfulFleetCancellations[0]
	if aws.ToString(first.CapacityReservationFleetId) != "crf-00000000000000090" {
		t.Fatalf("unexpected first successful fleet id: %q", aws.ToString(first.CapacityReservationFleetId))
	}
	if first.CurrentFleetState != awsec2types.CapacityReservationFleetStateCancelled {
		t.Fatalf("unexpected current fleet state: %q", first.CurrentFleetState)
	}
	if first.PreviousFleetState != awsec2types.CapacityReservationFleetStateActive {
		t.Fatalf("unexpected previous fleet state: %q", first.PreviousFleetState)
	}
}

func TestEC2Stage90ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CancelCapacityReservationFleets",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"CapacityReservationFleetId.1": "crf-00000000000000090",
			"CapacityReservationFleetId.2": "crf-00000000000000091",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
