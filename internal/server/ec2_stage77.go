package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage77Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AcceptCapacityReservationBillingOwnership":
		ret, err := s.ec2.AcceptCapacityReservationBillingOwnership(strings.TrimSpace(r.Form.Get("CapacityReservationId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "AcceptCapacityReservationBillingOwnershipResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}
