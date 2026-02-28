package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage102Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateCapacityReservation":
		instanceCount, ok := parseEC2OptionalInt32(r.Form.Get("InstanceCount"))
		if !ok || instanceCount == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		ebsOptimized, hasEbsOptimized, ok := ec2OptionalBoolFromForm(r.Form, "EbsOptimized")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasEbsOptimized {
			ebsOptimized = nil
		}

		ephemeralStorage, hasEphemeralStorage, ok := ec2OptionalBoolFromForm(r.Form, "EphemeralStorage")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasEphemeralStorage {
			ephemeralStorage = nil
		}

		reservation, err := s.ec2.CreateCapacityReservation(
			*instanceCount,
			strings.TrimSpace(r.Form.Get("InstancePlatform")),
			strings.TrimSpace(r.Form.Get("InstanceType")),
			strings.TrimSpace(r.Form.Get("AvailabilityZone")),
			strings.TrimSpace(r.Form.Get("AvailabilityZoneId")),
			strings.TrimSpace(r.Form.Get("InstanceMatchCriteria")),
			strings.TrimSpace(r.Form.Get("Tenancy")),
			ebsOptimized,
			ephemeralStorage,
			parseEC2TagSpecificationsForResource(r.Form, "capacity-reservation"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2Stage102CreateCapacityReservationResponse{
			XMLName:             xml.Name{Local: "CreateCapacityReservationResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			CapacityReservation: ec2Stage102CapacityReservationItemFrom(reservation),
		})
		return true
	default:
		return false
	}
}

type ec2Stage102CreateCapacityReservationResponse struct {
	XMLName             xml.Name                           `xml:"CreateCapacityReservationResponse"`
	Xmlns               string                             `xml:"xmlns,attr"`
	RequestID           string                             `xml:"requestId"`
	CapacityReservation ec2Stage102CapacityReservationItem `xml:"capacityReservation"`
}

type ec2Stage102CapacityReservationItem struct {
	AvailabilityZone       string    `xml:"availabilityZone,omitempty"`
	AvailabilityZoneID     string    `xml:"availabilityZoneId,omitempty"`
	AvailableInstanceCount int32     `xml:"availableInstanceCount,omitempty"`
	CapacityReservationID  string    `xml:"capacityReservationId,omitempty"`
	CreateDate             string    `xml:"createDate,omitempty"`
	EbsOptimized           *bool     `xml:"ebsOptimized,omitempty"`
	EphemeralStorage       *bool     `xml:"ephemeralStorage,omitempty"`
	InstanceMatchCriteria  string    `xml:"instanceMatchCriteria,omitempty"`
	InstancePlatform       string    `xml:"instancePlatform,omitempty"`
	InstanceType           string    `xml:"instanceType,omitempty"`
	OwnerID                string    `xml:"ownerId,omitempty"`
	State                  string    `xml:"state,omitempty"`
	TagSet                 ec2TagSet `xml:"tagSet"`
	Tenancy                string    `xml:"tenancy,omitempty"`
	TotalInstanceCount     int32     `xml:"totalInstanceCount,omitempty"`
}

func ec2Stage102CapacityReservationItemFrom(in ec2svc.CapacityReservation) ec2Stage102CapacityReservationItem {
	return ec2Stage102CapacityReservationItem{
		AvailabilityZone:       in.AvailabilityZone,
		AvailabilityZoneID:     in.AvailabilityZoneID,
		AvailableInstanceCount: in.AvailableInstanceCount,
		CapacityReservationID:  in.ID,
		CreateDate:             in.CreateDate.Format(time.RFC3339),
		EbsOptimized:           in.EbsOptimized,
		EphemeralStorage:       in.EphemeralStorage,
		InstanceMatchCriteria:  in.InstanceMatchCriteria,
		InstancePlatform:       in.InstancePlatform,
		InstanceType:           in.InstanceType,
		OwnerID:                in.OwnerID,
		State:                  in.State,
		TagSet:                 ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		Tenancy:                in.Tenancy,
		TotalInstanceCount:     in.TotalInstanceCount,
	}
}
