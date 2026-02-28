package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage121SDKLifecycle(t *testing.T) {
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

	createReservedInstancesListingOut, err := client.CreateReservedInstancesListing(ctx, &awsec2.CreateReservedInstancesListingInput{
		ClientToken:         aws.String("stage121-ril-token"),
		InstanceCount:       aws.Int32(1),
		ReservedInstancesId: aws.String("ri-stage121"),
		PriceSchedules: []awsec2types.PriceScheduleSpecification{
			{Price: aws.Float64(1.25), Term: aws.Int64(1), CurrencyCode: awsec2types.CurrencyCodeValuesUsd},
		},
	})
	if err != nil {
		t.Fatalf("create reserved instances listing: %v", err)
	}
	if len(createReservedInstancesListingOut.ReservedInstancesListings) != 1 || createReservedInstancesListingOut.ReservedInstancesListings[0].ReservedInstancesListingId == nil {
		t.Fatalf("expected created reserved instances listing")
	}
	listingID := aws.ToString(createReservedInstancesListingOut.ReservedInstancesListings[0].ReservedInstancesListingId)
	reservedInstancesID := "ri-" + strings.TrimPrefix(listingID, "ril-")
	reservedInstancesModificationID := "rimod-" + strings.TrimPrefix(listingID, "ril-")

	describeReservedInstancesListingsOut, err := client.DescribeReservedInstancesListings(ctx, &awsec2.DescribeReservedInstancesListingsInput{
		ReservedInstancesId:        aws.String(reservedInstancesID),
		ReservedInstancesListingId: aws.String(listingID),
	})
	if err != nil {
		t.Fatalf("describe reserved instances listings: %v", err)
	}
	if len(describeReservedInstancesListingsOut.ReservedInstancesListings) != 1 ||
		aws.ToString(describeReservedInstancesListingsOut.ReservedInstancesListings[0].ReservedInstancesListingId) != listingID {
		t.Fatalf("unexpected reserved instances listings output: %#v", describeReservedInstancesListingsOut.ReservedInstancesListings)
	}

	describeReservedInstancesModificationsOut, err := client.DescribeReservedInstancesModifications(ctx, &awsec2.DescribeReservedInstancesModificationsInput{
		ReservedInstancesModificationIds: []string{reservedInstancesModificationID},
	})
	if err != nil {
		t.Fatalf("describe reserved instances modifications: %v", err)
	}
	if len(describeReservedInstancesModificationsOut.ReservedInstancesModifications) != 1 ||
		aws.ToString(describeReservedInstancesModificationsOut.ReservedInstancesModifications[0].ReservedInstancesModificationId) != reservedInstancesModificationID {
		t.Fatalf("unexpected reserved instances modifications output: %#v", describeReservedInstancesModificationsOut.ReservedInstancesModifications)
	}

	describeReservedInstancesOfferingsOut, err := client.DescribeReservedInstancesOfferings(ctx, &awsec2.DescribeReservedInstancesOfferingsInput{
		IncludeMarketplace: aws.Bool(false),
		InstanceType:       awsec2types.InstanceTypeT3Micro,
		MaxResults:         aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe reserved instances offerings: %v", err)
	}
	if len(describeReservedInstancesOfferingsOut.ReservedInstancesOfferings) == 0 ||
		describeReservedInstancesOfferingsOut.ReservedInstancesOfferings[0].ReservedInstancesOfferingId == nil {
		t.Fatalf("expected reserved instances offerings output")
	}

	firstSlotEarliestTime := time.Now().UTC().Add(23 * time.Hour)
	firstSlotLatestTime := time.Now().UTC().Add(72 * time.Hour)
	describeScheduledInstanceAvailabilityOut, err := client.DescribeScheduledInstanceAvailability(ctx, &awsec2.DescribeScheduledInstanceAvailabilityInput{
		FirstSlotStartTimeRange: &awsec2types.SlotDateTimeRangeRequest{
			EarliestTime: aws.Time(firstSlotEarliestTime),
			LatestTime:   aws.Time(firstSlotLatestTime),
		},
		MaxResults: aws.Int32(10),
		Recurrence: &awsec2types.ScheduledInstanceRecurrenceRequest{
			Frequency:      aws.String("Weekly"),
			OccurrenceDays: []int32{1},
		},
	})
	if err != nil {
		t.Fatalf("describe scheduled instance availability: %v", err)
	}
	if len(describeScheduledInstanceAvailabilityOut.ScheduledInstanceAvailabilitySet) == 0 {
		t.Fatalf("expected scheduled instance availability output")
	}

	scheduledInstanceID := "sci-00000000121"
	slotStartEarliestTime := time.Now().UTC().Add(-1 * time.Hour)
	slotStartLatestTime := time.Now().UTC().Add(72 * time.Hour)
	describeScheduledInstancesOut, err := client.DescribeScheduledInstances(ctx, &awsec2.DescribeScheduledInstancesInput{
		MaxResults:           aws.Int32(10),
		ScheduledInstanceIds: []string{scheduledInstanceID},
		SlotStartTimeRange: &awsec2types.SlotStartTimeRangeRequest{
			EarliestTime: aws.Time(slotStartEarliestTime),
			LatestTime:   aws.Time(slotStartLatestTime),
		},
	})
	if err != nil {
		t.Fatalf("describe scheduled instances: %v", err)
	}
	if len(describeScheduledInstancesOut.ScheduledInstanceSet) != 1 ||
		aws.ToString(describeScheduledInstancesOut.ScheduledInstanceSet[0].ScheduledInstanceId) != scheduledInstanceID {
		t.Fatalf("unexpected scheduled instances output: %#v", describeScheduledInstancesOut.ScheduledInstanceSet)
	}

	localGatewayID := "lgw-00000000121"
	createLocalGatewayVirtualInterfaceGroupOut, err := client.CreateLocalGatewayVirtualInterfaceGroup(ctx, &awsec2.CreateLocalGatewayVirtualInterfaceGroupInput{
		LocalGatewayId: aws.String(localGatewayID),
	})
	if err != nil {
		t.Fatalf("create local gateway virtual interface group: %v", err)
	}
	if createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup == nil ||
		createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup.LocalGatewayVirtualInterfaceGroupId == nil {
		t.Fatalf("expected created local gateway virtual interface group")
	}
	localGatewayVirtualInterfaceGroupID := aws.ToString(createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup.LocalGatewayVirtualInterfaceGroupId)

	createLocalGatewayVirtualInterfaceOut, err := client.CreateLocalGatewayVirtualInterface(ctx, &awsec2.CreateLocalGatewayVirtualInterfaceInput{
		LocalAddress:                        aws.String("169.254.121.1"),
		LocalGatewayVirtualInterfaceGroupId: aws.String(localGatewayVirtualInterfaceGroupID),
		OutpostLagId:                        aws.String("lag-121"),
		PeerAddress:                         aws.String("169.254.121.2"),
		Vlan:                                aws.Int32(121),
	})
	if err != nil {
		t.Fatalf("create local gateway virtual interface: %v", err)
	}
	if createLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface == nil ||
		createLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface.LocalGatewayVirtualInterfaceId == nil {
		t.Fatalf("expected created local gateway virtual interface")
	}
	localGatewayVirtualInterfaceID := aws.ToString(createLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface.LocalGatewayVirtualInterfaceId)
	serviceLinkVirtualInterfaceID := "slvi-" + strings.TrimPrefix(localGatewayVirtualInterfaceID, "lgw-vif-")

	describeServiceLinkVirtualInterfacesOut, err := client.DescribeServiceLinkVirtualInterfaces(ctx, &awsec2.DescribeServiceLinkVirtualInterfacesInput{
		MaxResults:                     aws.Int32(10),
		ServiceLinkVirtualInterfaceIds: []string{serviceLinkVirtualInterfaceID},
	})
	if err != nil {
		t.Fatalf("describe service link virtual interfaces: %v", err)
	}
	if len(describeServiceLinkVirtualInterfacesOut.ServiceLinkVirtualInterfaces) != 1 ||
		aws.ToString(describeServiceLinkVirtualInterfacesOut.ServiceLinkVirtualInterfaces[0].ServiceLinkVirtualInterfaceId) != serviceLinkVirtualInterfaceID {
		t.Fatalf("unexpected service link virtual interfaces output: %#v", describeServiceLinkVirtualInterfacesOut.ServiceLinkVirtualInterfaces)
	}

	createVolumeOut, err := client.CreateVolume(ctx, &awsec2.CreateVolumeInput{
		AvailabilityZone: aws.String("us-east-1a"),
		Size:             aws.Int32(8),
	})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	if createVolumeOut.VolumeId == nil {
		t.Fatalf("expected created volume")
	}

	createSnapshotOut, err := client.CreateSnapshot(ctx, &awsec2.CreateSnapshotInput{VolumeId: createVolumeOut.VolumeId})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if createSnapshotOut.SnapshotId == nil {
		t.Fatalf("expected created snapshot")
	}
	snapshotID := aws.ToString(createSnapshotOut.SnapshotId)

	describeSnapshotAttributeOut, err := client.DescribeSnapshotAttribute(ctx, &awsec2.DescribeSnapshotAttributeInput{
		Attribute:  awsec2types.SnapshotAttributeNameCreateVolumePermission,
		SnapshotId: aws.String(snapshotID),
	})
	if err != nil {
		t.Fatalf("describe snapshot attribute: %v", err)
	}
	if aws.ToString(describeSnapshotAttributeOut.SnapshotId) != snapshotID {
		t.Fatalf("unexpected snapshot attribute output: %#v", describeSnapshotAttributeOut)
	}

	describeSnapshotTierStatusOut, err := client.DescribeSnapshotTierStatus(ctx, &awsec2.DescribeSnapshotTierStatusInput{
		Filters:    []awsec2types.Filter{{Name: aws.String("snapshot-id"), Values: []string{snapshotID}}},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe snapshot tier status: %v", err)
	}
	if len(describeSnapshotTierStatusOut.SnapshotTierStatuses) != 1 ||
		aws.ToString(describeSnapshotTierStatusOut.SnapshotTierStatuses[0].SnapshotId) != snapshotID {
		t.Fatalf("unexpected snapshot tier status output: %#v", describeSnapshotTierStatusOut.SnapshotTierStatuses)
	}

	_, err = client.CreateSpotDatafeedSubscription(ctx, &awsec2.CreateSpotDatafeedSubscriptionInput{
		Bucket: aws.String("stage121-bucket"),
		Prefix: aws.String("logs/"),
	})
	if err != nil {
		t.Fatalf("create spot datafeed subscription: %v", err)
	}

	describeSpotDatafeedSubscriptionOut, err := client.DescribeSpotDatafeedSubscription(ctx, &awsec2.DescribeSpotDatafeedSubscriptionInput{})
	if err != nil {
		t.Fatalf("describe spot datafeed subscription: %v", err)
	}
	if describeSpotDatafeedSubscriptionOut.SpotDatafeedSubscription == nil ||
		aws.ToString(describeSpotDatafeedSubscriptionOut.SpotDatafeedSubscription.Bucket) != "stage121-bucket" {
		t.Fatalf("unexpected spot datafeed subscription output: %#v", describeSpotDatafeedSubscriptionOut.SpotDatafeedSubscription)
	}

	spotFleetRequestID := "sfr-stage121"
	describeSpotFleetInstancesOut, err := client.DescribeSpotFleetInstances(ctx, &awsec2.DescribeSpotFleetInstancesInput{
		MaxResults:         aws.Int32(10),
		SpotFleetRequestId: aws.String(spotFleetRequestID),
	})
	if err != nil {
		t.Fatalf("describe spot fleet instances: %v", err)
	}
	if aws.ToString(describeSpotFleetInstancesOut.SpotFleetRequestId) != spotFleetRequestID || len(describeSpotFleetInstancesOut.ActiveInstances) == 0 {
		t.Fatalf("unexpected spot fleet instances output: %#v", describeSpotFleetInstancesOut)
	}
}

func TestEC2Stage121ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeReservedInstancesListings",
		"DescribeReservedInstancesModifications",
		"DescribeReservedInstancesOfferings",
		"DescribeScheduledInstanceAvailability",
		"DescribeScheduledInstances",
		"DescribeServiceLinkVirtualInterfaces",
		"DescribeSnapshotAttribute",
		"DescribeSnapshotTierStatus",
		"DescribeSpotDatafeedSubscription",
		"DescribeSpotFleetInstances",
	}

	paramsByAction := map[string]map[string]string{
		"DescribeReservedInstancesListings": {
			"ReservedInstancesListingId": "ril-0000000121",
		},
		"DescribeReservedInstancesModifications": {
			"ReservedInstancesModificationId.1": "rimod-0000000121",
		},
		"DescribeReservedInstancesOfferings": {
			"MaxResults": "10",
		},
		"DescribeScheduledInstanceAvailability": {
			"MaxResults": "10",
		},
		"DescribeScheduledInstances": {
			"MaxResults": "10",
		},
		"DescribeServiceLinkVirtualInterfaces": {
			"MaxResults": "10",
		},
		"DescribeSnapshotAttribute": {
			"Attribute":  "createVolumePermission",
			"SnapshotId": "snap-0000000121",
		},
		"DescribeSnapshotTierStatus": {
			"MaxResults": "10",
		},
		"DescribeSpotDatafeedSubscription": {},
		"DescribeSpotFleetInstances": {
			"SpotFleetRequestId": "sfr-0000000121",
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
