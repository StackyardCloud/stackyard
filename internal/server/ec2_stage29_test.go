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

func TestEC2Stage29SDKLifecycle(t *testing.T) {
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

	createVpcOut, err := client.CreateVpc(ctx, &awsec2.CreateVpcInput{CidrBlock: aws.String("10.220.0.0/16")})
	if err != nil || createVpcOut.Vpc == nil || createVpcOut.Vpc.VpcId == nil {
		t.Fatalf("create vpc: %v", err)
	}
	vpcID := aws.ToString(createVpcOut.Vpc.VpcId)

	associateVpcOut, err := client.AssociateVpcCidrBlock(ctx, &awsec2.AssociateVpcCidrBlockInput{
		VpcId:     aws.String(vpcID),
		CidrBlock: aws.String("10.221.0.0/16"),
	})
	if err != nil || associateVpcOut.CidrBlockAssociation == nil || associateVpcOut.CidrBlockAssociation.AssociationId == nil {
		t.Fatalf("associate vpc cidr block: %v", err)
	}
	vpcAssociationID := aws.ToString(associateVpcOut.CidrBlockAssociation.AssociationId)

	disassociateVpcOut, err := client.DisassociateVpcCidrBlock(ctx, &awsec2.DisassociateVpcCidrBlockInput{
		AssociationId: aws.String(vpcAssociationID),
	})
	if err != nil || disassociateVpcOut.CidrBlockAssociation == nil || disassociateVpcOut.CidrBlockAssociation.CidrBlockState == nil || disassociateVpcOut.CidrBlockAssociation.CidrBlockState.State != awsec2types.VpcCidrBlockStateCodeDisassociated {
		t.Fatalf("disassociate vpc cidr block: %v", err)
	}

	createSubnetOut, err := client.CreateSubnet(ctx, &awsec2.CreateSubnetInput{
		VpcId:     aws.String(vpcID),
		CidrBlock: aws.String("10.220.1.0/24"),
	})
	if err != nil || createSubnetOut.Subnet == nil || createSubnetOut.Subnet.SubnetId == nil {
		t.Fatalf("create subnet: %v", err)
	}
	subnetID := aws.ToString(createSubnetOut.Subnet.SubnetId)

	associateSubnetOut, err := client.AssociateSubnetCidrBlock(ctx, &awsec2.AssociateSubnetCidrBlockInput{
		SubnetId:      aws.String(subnetID),
		Ipv6CidrBlock: aws.String("2001:db8:220::/64"),
	})
	if err != nil || associateSubnetOut.Ipv6CidrBlockAssociation == nil || associateSubnetOut.Ipv6CidrBlockAssociation.AssociationId == nil {
		t.Fatalf("associate subnet cidr block: %v", err)
	}
	subnetAssociationID := aws.ToString(associateSubnetOut.Ipv6CidrBlockAssociation.AssociationId)

	disassociateSubnetOut, err := client.DisassociateSubnetCidrBlock(ctx, &awsec2.DisassociateSubnetCidrBlockInput{
		AssociationId: aws.String(subnetAssociationID),
	})
	if err != nil || disassociateSubnetOut.Ipv6CidrBlockAssociation == nil || disassociateSubnetOut.Ipv6CidrBlockAssociation.Ipv6CidrBlockState == nil || disassociateSubnetOut.Ipv6CidrBlockAssociation.Ipv6CidrBlockState.State != awsec2types.SubnetCidrBlockStateCodeDisassociated {
		t.Fatalf("disassociate subnet cidr block: %v", err)
	}

	createReservationOneOut, err := client.CreateSubnetCidrReservation(ctx, &awsec2.CreateSubnetCidrReservationInput{
		SubnetId:        aws.String(subnetID),
		Cidr:            aws.String("10.220.1.16/28"),
		ReservationType: awsec2types.SubnetCidrReservationTypeExplicit,
		Description:     aws.String("stage29-reservation-1"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeSubnetCidrReservation,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("test")}},
			},
		},
	})
	if err != nil || createReservationOneOut.SubnetCidrReservation == nil || createReservationOneOut.SubnetCidrReservation.SubnetCidrReservationId == nil {
		t.Fatalf("create subnet cidr reservation #1: %v", err)
	}
	reservationOneID := aws.ToString(createReservationOneOut.SubnetCidrReservation.SubnetCidrReservationId)

	createReservationTwoOut, err := client.CreateSubnetCidrReservation(ctx, &awsec2.CreateSubnetCidrReservationInput{
		SubnetId:        aws.String(subnetID),
		Cidr:            aws.String("2001:db8:220:1::/80"),
		ReservationType: awsec2types.SubnetCidrReservationTypePrefix,
		Description:     aws.String("stage29-reservation-2"),
	})
	if err != nil || createReservationTwoOut.SubnetCidrReservation == nil || createReservationTwoOut.SubnetCidrReservation.SubnetCidrReservationId == nil {
		t.Fatalf("create subnet cidr reservation #2: %v", err)
	}
	reservationTwoID := aws.ToString(createReservationTwoOut.SubnetCidrReservation.SubnetCidrReservationId)

	getReservationsPageOneOut, err := client.GetSubnetCidrReservations(ctx, &awsec2.GetSubnetCidrReservationsInput{
		SubnetId:   aws.String(subnetID),
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("get subnet cidr reservations page 1: %v", err)
	}
	if getReservationsPageOneOut.NextToken == nil {
		t.Fatalf("expected next token on page 1")
	}
	if len(getReservationsPageOneOut.SubnetIpv4CidrReservations)+len(getReservationsPageOneOut.SubnetIpv6CidrReservations) != 1 {
		t.Fatalf("expected one reservation on page 1")
	}

	getReservationsPageTwoOut, err := client.GetSubnetCidrReservations(ctx, &awsec2.GetSubnetCidrReservationsInput{
		SubnetId:   aws.String(subnetID),
		NextToken:  getReservationsPageOneOut.NextToken,
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("get subnet cidr reservations page 2: %v", err)
	}
	if len(getReservationsPageTwoOut.SubnetIpv4CidrReservations)+len(getReservationsPageTwoOut.SubnetIpv6CidrReservations) != 1 {
		t.Fatalf("expected one reservation on page 2")
	}

	getFilteredReservationsOut, err := client.GetSubnetCidrReservations(ctx, &awsec2.GetSubnetCidrReservationsInput{
		SubnetId: aws.String(subnetID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("reservation-type"), Values: []string{"explicit"}},
			{Name: aws.String("subnet-cidr-reservation-id"), Values: []string{reservationOneID}},
		},
	})
	if err != nil {
		t.Fatalf("get subnet cidr reservations with filters: %v", err)
	}
	if len(getFilteredReservationsOut.SubnetIpv4CidrReservations) != 1 || aws.ToString(getFilteredReservationsOut.SubnetIpv4CidrReservations[0].SubnetCidrReservationId) != reservationOneID {
		t.Fatalf("expected filtered result to include reservation %s", reservationOneID)
	}

	deleteReservationOneOut, err := client.DeleteSubnetCidrReservation(ctx, &awsec2.DeleteSubnetCidrReservationInput{
		SubnetCidrReservationId: aws.String(reservationOneID),
	})
	if err != nil || deleteReservationOneOut.DeletedSubnetCidrReservation == nil || aws.ToString(deleteReservationOneOut.DeletedSubnetCidrReservation.SubnetCidrReservationId) != reservationOneID {
		t.Fatalf("delete subnet cidr reservation #1: %v", err)
	}

	deleteReservationTwoOut, err := client.DeleteSubnetCidrReservation(ctx, &awsec2.DeleteSubnetCidrReservationInput{
		SubnetCidrReservationId: aws.String(reservationTwoID),
	})
	if err != nil || deleteReservationTwoOut.DeletedSubnetCidrReservation == nil || aws.ToString(deleteReservationTwoOut.DeletedSubnetCidrReservation.SubnetCidrReservationId) != reservationTwoID {
		t.Fatalf("delete subnet cidr reservation #2: %v", err)
	}
}

func TestEC2Stage29ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AssociateSubnetCidrBlock",
		"AssociateVpcCidrBlock",
		"CreateSubnetCidrReservation",
		"DeleteSubnetCidrReservation",
		"DisassociateSubnetCidrBlock",
		"DisassociateVpcCidrBlock",
		"GetSubnetCidrReservations",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "AssociateVpcCidrBlock":
			params["VpcId"] = "vpc-00000001"
			params["CidrBlock"] = "10.222.0.0/16"
		case "DisassociateVpcCidrBlock", "DisassociateSubnetCidrBlock":
			params["AssociationId"] = "assoc-123"
		case "AssociateSubnetCidrBlock":
			params["SubnetId"] = "subnet-00000001"
			params["Ipv6CidrBlock"] = "2001:db8:1::/64"
		case "CreateSubnetCidrReservation":
			params["SubnetId"] = "subnet-00000001"
			params["Cidr"] = "10.0.0.128/28"
			params["ReservationType"] = "explicit"
		case "DeleteSubnetCidrReservation":
			params["SubnetCidrReservationId"] = "scr-123"
		case "GetSubnetCidrReservations":
			params["SubnetId"] = "subnet-00000001"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
