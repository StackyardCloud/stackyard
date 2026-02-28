package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage48Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyVpcEndpointServicePayerResponsibility":
		ret, err := s.ec2.ModifyVpcEndpointServicePayerResponsibility(
			strings.TrimSpace(r.Form.Get("ServiceId")),
			strings.TrimSpace(r.Form.Get("PayerResponsibility")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyVpcEndpointServicePayerResponsibilityResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}
