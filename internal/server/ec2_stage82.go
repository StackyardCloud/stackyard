package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage82Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AssociateCapacityReservationBillingOwner":
		ret, err := s.ec2.AssociateCapacityReservationBillingOwner(
			strings.TrimSpace(r.Form.Get("CapacityReservationId")),
			strings.TrimSpace(r.Form.Get("UnusedReservationBillingOwnerId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "AssociateCapacityReservationBillingOwnerResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}
