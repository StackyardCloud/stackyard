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

func TestEC2Stage96SDKLifecycle(t *testing.T) {
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

	out, err := client.CancelSpotFleetRequests(ctx, &awsec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: []string{"sfr-00000000000000096", "sfr-missing00000000000000096"},
		TerminateInstances:  aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("cancel spot fleet requests: %v", err)
	}

	if len(out.SuccessfulFleetRequests) != 1 {
		t.Fatalf("expected 1 successful fleet request, got %d", len(out.SuccessfulFleetRequests))
	}
	if len(out.UnsuccessfulFleetRequests) != 1 {
		t.Fatalf("expected 1 unsuccessful fleet request, got %d", len(out.UnsuccessfulFleetRequests))
	}

	success := out.SuccessfulFleetRequests[0]
	if aws.ToString(success.SpotFleetRequestId) != "sfr-00000000000000096" {
		t.Fatalf("unexpected successful spot fleet request id: %q", aws.ToString(success.SpotFleetRequestId))
	}
	if success.CurrentSpotFleetRequestState != awsec2types.BatchStateCancelledTerminatingInstances {
		t.Fatalf("unexpected current spot fleet request state: %q", success.CurrentSpotFleetRequestState)
	}
	if success.PreviousSpotFleetRequestState != awsec2types.BatchStateActive {
		t.Fatalf("unexpected previous spot fleet request state: %q", success.PreviousSpotFleetRequestState)
	}

	failed := out.UnsuccessfulFleetRequests[0]
	if aws.ToString(failed.SpotFleetRequestId) != "sfr-missing00000000000000096" {
		t.Fatalf("unexpected failed spot fleet request id: %q", aws.ToString(failed.SpotFleetRequestId))
	}
	if failed.Error == nil {
		t.Fatalf("expected unsuccessful request error")
	}
	if failed.Error.Code != awsec2types.CancelBatchErrorCodeFleetRequestIdDoesNotExist {
		t.Fatalf("unexpected unsuccessful error code: %q", failed.Error.Code)
	}
}

func TestEC2Stage96ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CancelSpotFleetRequests",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"SpotFleetRequestId.1": "sfr-00000000000000096",
			"TerminateInstances":   "true",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
