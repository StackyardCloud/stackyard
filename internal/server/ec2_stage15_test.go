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

func TestEC2Stage15SDKLifecycle(t *testing.T) {
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

	allocateOut, err := client.AllocateAddress(ctx, &awsec2.AllocateAddressInput{Domain: awsec2types.DomainTypeVpc})
	if err != nil || allocateOut.AllocationId == nil || allocateOut.PublicIp == nil {
		t.Fatalf("allocate address: %v", err)
	}
	allocationID := aws.ToString(allocateOut.AllocationId)
	publicIP := aws.ToString(allocateOut.PublicIp)

	enableOut, err := client.EnableAddressTransfer(ctx, &awsec2.EnableAddressTransferInput{
		AllocationId:      aws.String(allocationID),
		TransferAccountId: aws.String("210987654321"),
	})
	if err != nil || enableOut.AddressTransfer == nil || enableOut.AddressTransfer.AddressTransferStatus != awsec2types.AddressTransferStatusPending {
		t.Fatalf("enable address transfer: %v", err)
	}

	describeTransfersOut, err := client.DescribeAddressTransfers(ctx, &awsec2.DescribeAddressTransfersInput{
		AllocationIds: []string{allocationID},
	})
	if err != nil || len(describeTransfersOut.AddressTransfers) != 1 || describeTransfersOut.AddressTransfers[0].AddressTransferStatus != awsec2types.AddressTransferStatusPending {
		t.Fatalf("describe address transfers: %v", err)
	}

	acceptOut, err := client.AcceptAddressTransfer(ctx, &awsec2.AcceptAddressTransferInput{
		Address: aws.String(publicIP),
	})
	if err != nil || acceptOut.AddressTransfer == nil || acceptOut.AddressTransfer.AddressTransferStatus != awsec2types.AddressTransferStatusAccepted || acceptOut.AddressTransfer.TransferOfferAcceptedTimestamp == nil {
		t.Fatalf("accept address transfer: %v", err)
	}

	disableOut, err := client.DisableAddressTransfer(ctx, &awsec2.DisableAddressTransferInput{
		AllocationId: aws.String(allocationID),
	})
	if err != nil || disableOut.AddressTransfer == nil || disableOut.AddressTransfer.AddressTransferStatus != awsec2types.AddressTransferStatusDisabled {
		t.Fatalf("disable address transfer: %v", err)
	}

	describeMovingOut, err := client.DescribeMovingAddresses(ctx, &awsec2.DescribeMovingAddressesInput{
		PublicIps: []string{publicIP},
	})
	if err != nil || len(describeMovingOut.MovingAddressStatuses) != 1 || describeMovingOut.MovingAddressStatuses[0].MoveStatus != awsec2types.MoveStatusRestoringToClassic {
		t.Fatalf("describe moving addresses: %v", err)
	}
}

func TestEC2Stage15ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"EnableAddressTransfer",
		"DisableAddressTransfer",
		"AcceptAddressTransfer",
		"DescribeAddressTransfers",
		"DescribeMovingAddresses",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "EnableAddressTransfer":
			params["AllocationId"] = "eipalloc-00000001"
			params["TransferAccountId"] = "210987654321"
		case "DisableAddressTransfer":
			params["AllocationId"] = "eipalloc-00000001"
		case "AcceptAddressTransfer":
			params["Address"] = "203.0.113.10"
		case "DescribeAddressTransfers":
			params["AllocationId.1"] = "eipalloc-00000001"
		case "DescribeMovingAddresses":
			params["PublicIp.1"] = "203.0.113.10"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
