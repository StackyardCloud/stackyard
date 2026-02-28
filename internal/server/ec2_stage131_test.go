package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage131SDKLifecycle(t *testing.T) {
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

	createIpamOut, err := client.CreateIpam(ctx, &awsec2.CreateIpamInput{})
	if err != nil || createIpamOut.Ipam == nil || createIpamOut.Ipam.IpamId == nil {
		t.Fatalf("create ipam: %v", err)
	}
	ipamID := aws.ToString(createIpamOut.Ipam.IpamId)

	createIpamScopeOut, err := client.CreateIpamScope(ctx, &awsec2.CreateIpamScopeInput{IpamId: aws.String(ipamID)})
	if err != nil || createIpamScopeOut.IpamScope == nil || createIpamScopeOut.IpamScope.IpamScopeId == nil {
		t.Fatalf("create ipam scope: %v", err)
	}
	ipamScopeID := aws.ToString(createIpamScopeOut.IpamScope.IpamScopeId)

	createIpamPoolOut, err := client.CreateIpamPool(ctx, &awsec2.CreateIpamPoolInput{
		AddressFamily: awsec2types.AddressFamilyIpv4,
		IpamScopeId:   aws.String(ipamScopeID),
	})
	if err != nil || createIpamPoolOut.IpamPool == nil || createIpamPoolOut.IpamPool.IpamPoolId == nil {
		t.Fatalf("create ipam pool: %v", err)
	}
	ipamPoolID := aws.ToString(createIpamPoolOut.IpamPool.IpamPoolId)

	provisionIpamByoasnOut, err := client.ProvisionIpamByoasn(ctx, &awsec2.ProvisionIpamByoasnInput{
		Asn:    aws.String("AS64512"),
		IpamId: aws.String(ipamID),
		AsnAuthorizationContext: &awsec2types.AsnAuthorizationContext{
			Message:   aws.String("stage131-message"),
			Signature: aws.String("stage131-signature"),
		},
	})
	if err != nil {
		t.Fatalf("provision ipam byoasn: %v", err)
	}
	if provisionIpamByoasnOut.Byoasn == nil || aws.ToString(provisionIpamByoasnOut.Byoasn.Asn) != "AS64512" {
		t.Fatalf("unexpected provision ipam byoasn output: %#v", provisionIpamByoasnOut.Byoasn)
	}

	provisionIpamPoolCidrOut, err := client.ProvisionIpamPoolCidr(ctx, &awsec2.ProvisionIpamPoolCidrInput{
		IpamPoolId: aws.String(ipamPoolID),
		Cidr:       aws.String("10.131.0.0/24"),
	})
	if err != nil {
		t.Fatalf("provision ipam pool cidr: %v", err)
	}
	if provisionIpamPoolCidrOut.IpamPoolCidr == nil || aws.ToString(provisionIpamPoolCidrOut.IpamPoolCidr.Cidr) != "10.131.0.0/24" {
		t.Fatalf("unexpected provision ipam pool cidr output: %#v", provisionIpamPoolCidrOut.IpamPoolCidr)
	}

	createPublicIpv4PoolOut, err := client.CreatePublicIpv4Pool(ctx, &awsec2.CreatePublicIpv4PoolInput{})
	if err != nil || createPublicIpv4PoolOut.PoolId == nil {
		t.Fatalf("create public ipv4 pool: %v", err)
	}
	publicPoolID := aws.ToString(createPublicIpv4PoolOut.PoolId)

	provisionPublicIpv4PoolCidrOut, err := client.ProvisionPublicIpv4PoolCidr(ctx, &awsec2.ProvisionPublicIpv4PoolCidrInput{
		IpamPoolId:    aws.String(ipamPoolID),
		NetmaskLength: aws.Int32(24),
		PoolId:        aws.String(publicPoolID),
	})
	if err != nil {
		t.Fatalf("provision public ipv4 pool cidr: %v", err)
	}
	if aws.ToString(provisionPublicIpv4PoolCidrOut.PoolId) != publicPoolID || provisionPublicIpv4PoolCidrOut.PoolAddressRange == nil {
		t.Fatalf("unexpected provision public ipv4 pool cidr output: %#v", provisionPublicIpv4PoolCidrOut)
	}

	purchaseCapacityBlockOut, err := client.PurchaseCapacityBlock(ctx, &awsec2.PurchaseCapacityBlockInput{
		CapacityBlockOfferingId: aws.String("cbo-stage131"),
		InstancePlatform:        awsec2types.CapacityReservationInstancePlatform("Linux/UNIX"),
	})
	if err != nil {
		t.Fatalf("purchase capacity block: %v", err)
	}
	if purchaseCapacityBlockOut.CapacityReservation == nil || purchaseCapacityBlockOut.CapacityReservation.CapacityReservationId == nil {
		t.Fatalf("unexpected purchase capacity block output: %#v", purchaseCapacityBlockOut)
	}
	capacityReservationID := aws.ToString(purchaseCapacityBlockOut.CapacityReservation.CapacityReservationId)
	if len(purchaseCapacityBlockOut.CapacityBlocks) == 0 {
		t.Fatalf("expected purchased capacity blocks")
	}

	purchaseCapacityBlockExtensionOut, err := client.PurchaseCapacityBlockExtension(ctx, &awsec2.PurchaseCapacityBlockExtensionInput{
		CapacityBlockExtensionOfferingId: aws.String("cbext-stage131"),
		CapacityReservationId:            aws.String(capacityReservationID),
	})
	if err != nil {
		t.Fatalf("purchase capacity block extension: %v", err)
	}
	if len(purchaseCapacityBlockExtensionOut.CapacityBlockExtensions) == 0 {
		t.Fatalf("expected purchased capacity block extensions")
	}

	purchaseHostReservationOut, err := client.PurchaseHostReservation(ctx, &awsec2.PurchaseHostReservationInput{
		HostIdSet:  []string{"h-00000000131"},
		OfferingId: aws.String("hro-00000000131"),
	})
	if err != nil {
		t.Fatalf("purchase host reservation: %v", err)
	}
	if len(purchaseHostReservationOut.Purchase) == 0 {
		t.Fatalf("expected purchase host reservation output")
	}

	describeReservedInstancesOfferingsOut, err := client.DescribeReservedInstancesOfferings(ctx, &awsec2.DescribeReservedInstancesOfferingsInput{})
	if err != nil || len(describeReservedInstancesOfferingsOut.ReservedInstancesOfferings) == 0 || describeReservedInstancesOfferingsOut.ReservedInstancesOfferings[0].ReservedInstancesOfferingId == nil {
		t.Fatalf("describe reserved instances offerings: %v %#v", err, describeReservedInstancesOfferingsOut.ReservedInstancesOfferings)
	}
	reservedInstancesOfferingID := aws.ToString(describeReservedInstancesOfferingsOut.ReservedInstancesOfferings[0].ReservedInstancesOfferingId)

	purchaseReservedInstancesOfferingOut, err := client.PurchaseReservedInstancesOffering(ctx, &awsec2.PurchaseReservedInstancesOfferingInput{
		InstanceCount:               aws.Int32(1),
		ReservedInstancesOfferingId: aws.String(reservedInstancesOfferingID),
	})
	if err != nil {
		t.Fatalf("purchase reserved instances offering: %v", err)
	}
	if aws.ToString(purchaseReservedInstancesOfferingOut.ReservedInstancesId) == "" {
		t.Fatalf("expected reserved instances id")
	}

	firstSlotEarliestTime := time.Now().UTC().Add(23 * time.Hour)
	firstSlotLatestTime := time.Now().UTC().Add(72 * time.Hour)
	describeScheduledInstanceAvailabilityOut, err := client.DescribeScheduledInstanceAvailability(ctx, &awsec2.DescribeScheduledInstanceAvailabilityInput{
		FirstSlotStartTimeRange: &awsec2types.SlotDateTimeRangeRequest{
			EarliestTime: aws.Time(firstSlotEarliestTime),
			LatestTime:   aws.Time(firstSlotLatestTime),
		},
		Recurrence: &awsec2types.ScheduledInstanceRecurrenceRequest{
			Frequency:      aws.String("Weekly"),
			OccurrenceDays: []int32{1},
		},
	})
	if err != nil || describeScheduledInstanceAvailabilityOut == nil || len(describeScheduledInstanceAvailabilityOut.ScheduledInstanceAvailabilitySet) == 0 || describeScheduledInstanceAvailabilityOut.ScheduledInstanceAvailabilitySet[0].PurchaseToken == nil {
		t.Fatalf("describe scheduled instance availability: %v %#v", err, describeScheduledInstanceAvailabilityOut)
	}
	purchaseToken := aws.ToString(describeScheduledInstanceAvailabilityOut.ScheduledInstanceAvailabilitySet[0].PurchaseToken)

	purchaseScheduledInstancesOut, err := client.PurchaseScheduledInstances(ctx, &awsec2.PurchaseScheduledInstancesInput{
		PurchaseRequests: []awsec2types.PurchaseRequest{{
			InstanceCount: aws.Int32(1),
			PurchaseToken: aws.String(purchaseToken),
		}},
	})
	if err != nil {
		t.Fatalf("purchase scheduled instances: %v", err)
	}
	if len(purchaseScheduledInstancesOut.ScheduledInstanceSet) == 0 {
		t.Fatalf("expected purchased scheduled instances")
	}

	registerImageOut, err := client.RegisterImage(ctx, &awsec2.RegisterImageInput{
		Name: aws.String("stage131-image"),
	})
	if err != nil {
		t.Fatalf("register image: %v", err)
	}
	if aws.ToString(registerImageOut.ImageId) == "" {
		t.Fatalf("expected image id in register image output")
	}

	registerInstanceEventNotificationAttributesOut, err := client.RegisterInstanceEventNotificationAttributes(ctx, &awsec2.RegisterInstanceEventNotificationAttributesInput{
		InstanceTagAttribute: &awsec2types.RegisterInstanceTagAttributeRequest{
			InstanceTagKeys: []string{"env", "team"},
		},
	})
	if err != nil {
		t.Fatalf("register instance event notification attributes: %v", err)
	}
	if registerInstanceEventNotificationAttributesOut.InstanceTagAttribute == nil ||
		len(registerInstanceEventNotificationAttributesOut.InstanceTagAttribute.InstanceTagKeys) != 2 {
		t.Fatalf("unexpected register instance event notification attributes output: %#v", registerInstanceEventNotificationAttributesOut.InstanceTagAttribute)
	}
}

func TestEC2Stage131ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ProvisionIpamByoasn",
		"ProvisionIpamPoolCidr",
		"ProvisionPublicIpv4PoolCidr",
		"PurchaseCapacityBlock",
		"PurchaseCapacityBlockExtension",
		"PurchaseHostReservation",
		"PurchaseReservedInstancesOffering",
		"PurchaseScheduledInstances",
		"RegisterImage",
		"RegisterInstanceEventNotificationAttributes",
	}

	paramsByAction := map[string]map[string]string{
		"ProvisionIpamByoasn": {
			"Asn":                               "AS64512",
			"IpamId":                            "ipam-00000000131",
			"AsnAuthorizationContext.Message":   "stage131-message",
			"AsnAuthorizationContext.Signature": "stage131-signature",
		},
		"ProvisionIpamPoolCidr": {
			"IpamPoolId": "ipam-pool-00000000131",
			"Cidr":       "10.131.0.0/24",
		},
		"ProvisionPublicIpv4PoolCidr": {
			"IpamPoolId":    "ipam-pool-00000000131",
			"NetmaskLength": "24",
			"PoolId":        "ipv4pool-ec2-00000000131",
		},
		"PurchaseCapacityBlock": {
			"CapacityBlockOfferingId": "cbo-stage131",
			"InstancePlatform":        "Linux/UNIX",
		},
		"PurchaseCapacityBlockExtension": {
			"CapacityBlockExtensionOfferingId": "cbext-stage131",
			"CapacityReservationId":            "cr-00000000131",
		},
		"PurchaseHostReservation": {
			"HostIdSet.1": "h-00000000131",
			"OfferingId":  "hro-00000000131",
		},
		"PurchaseReservedInstancesOffering": {
			"InstanceCount":               "1",
			"ReservedInstancesOfferingId": "off-00000000131",
		},
		"PurchaseScheduledInstances": {
			"PurchaseRequest.1.InstanceCount": "1",
			"PurchaseRequest.1.PurchaseToken": "purchase-token-131",
		},
		"RegisterImage": {
			"Name": "stage131-image",
		},
		"RegisterInstanceEventNotificationAttributes": {
			"InstanceTagAttribute.InstanceTagKey.1": "env",
		},
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, paramsByAction[action])
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
