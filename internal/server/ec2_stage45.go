package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage45Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyVpcTenancy":
		if err := s.ec2.ModifyVpcTenancy(
			strings.TrimSpace(r.Form.Get("VpcId")),
			strings.TrimSpace(r.Form.Get("InstanceTenancy")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyVpcTenancyResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	default:
		return false
	}
}
