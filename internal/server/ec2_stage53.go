package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage53Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "StartVpcEndpointServicePrivateDnsVerification":
		ret, err := s.ec2.StartVpcEndpointServicePrivateDnsVerification(strings.TrimSpace(r.Form.Get("ServiceId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "StartVpcEndpointServicePrivateDnsVerificationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}
