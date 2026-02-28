package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage103Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateCapacityReservationBySplitting":
		instanceCount, ok := parseEC2OptionalInt32(r.Form.Get("InstanceCount"))
		if !ok || instanceCount == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		out, err := s.ec2.CreateCapacityReservationBySplitting(
			*instanceCount,
			strings.TrimSpace(r.Form.Get("SourceCapacityReservationId")),
			parseEC2TagSpecificationsForResource(r.Form, "capacity-reservation"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2Stage103CreateCapacityReservationBySplittingResponse{
			XMLName:                        xml.Name{Local: "CreateCapacityReservationBySplittingResponse"},
			Xmlns:                          ec2Namespace,
			RequestID:                      "stackyard-request",
			InstanceCount:                  out.InstanceCount,
			SourceCapacityReservation:      ec2Stage102CapacityReservationItemFrom(out.SourceCapacityReservation),
			DestinationCapacityReservation: ec2Stage102CapacityReservationItemFrom(out.DestinationCapacityReservation),
		})
		return true
	default:
		return false
	}
}

type ec2Stage103CreateCapacityReservationBySplittingResponse struct {
	XMLName                        xml.Name                           `xml:"CreateCapacityReservationBySplittingResponse"`
	Xmlns                          string                             `xml:"xmlns,attr"`
	RequestID                      string                             `xml:"requestId"`
	DestinationCapacityReservation ec2Stage102CapacityReservationItem `xml:"destinationCapacityReservation"`
	InstanceCount                  int32                              `xml:"instanceCount,omitempty"`
	SourceCapacityReservation      ec2Stage102CapacityReservationItem `xml:"sourceCapacityReservation"`
}
