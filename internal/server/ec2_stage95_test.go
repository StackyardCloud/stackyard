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

func TestEC2Stage95SDKLifecycle(t *testing.T) {
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

	out, err := client.CancelReservedInstancesListing(ctx, &awsec2.CancelReservedInstancesListingInput{
		ReservedInstancesListingId: aws.String("ril-00000000000000095"),
	})
	if err != nil {
		t.Fatalf("cancel reserved instances listing: %v", err)
	}
	if len(out.ReservedInstancesListings) != 1 {
		t.Fatalf("expected 1 reserved instances listing, got %d", len(out.ReservedInstancesListings))
	}
	listing := out.ReservedInstancesListings[0]
	if aws.ToString(listing.ReservedInstancesListingId) != "ril-00000000000000095" {
		t.Fatalf("unexpected reserved instances listing id: %q", aws.ToString(listing.ReservedInstancesListingId))
	}
	if listing.Status != awsec2types.ListingStatusCancelled {
		t.Fatalf("unexpected reserved instances listing status: %q", listing.Status)
	}
}

func TestEC2Stage95ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CancelReservedInstancesListing",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"ReservedInstancesListingId": "ril-00000000000000095",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
