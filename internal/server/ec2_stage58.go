package server

import (
	"encoding/xml"
	"net/http"
)

func (s *Server) handleEC2Stage58Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DeleteVpcEndpointServiceConfigurations":
		unsuccessful, err := s.ec2.DeleteVpcEndpointServiceConfigurations(
			parseEC2MembersWithAliases(r.Form, "ServiceId", "ServiceIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteVpcEndpointServiceConfigurationsResponse{
			XMLName:      xml.Name{Local: "DeleteVpcEndpointServiceConfigurationsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Unsuccessful: ec2UnsuccessfulItemSet{Items: ec2UnsuccessfulItems(unsuccessful)},
		})
		return true
	default:
		return false
	}
}

type ec2DeleteVpcEndpointServiceConfigurationsResponse struct {
	XMLName      xml.Name               `xml:"DeleteVpcEndpointServiceConfigurationsResponse"`
	Xmlns        string                 `xml:"xmlns,attr"`
	RequestID    string                 `xml:"requestId"`
	Unsuccessful ec2UnsuccessfulItemSet `xml:"unsuccessful"`
}
