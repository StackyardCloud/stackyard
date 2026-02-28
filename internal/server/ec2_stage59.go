package server

import (
	"encoding/xml"
	"net/http"
)

func (s *Server) handleEC2Stage59Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DeleteVpcEndpoints":
		unsuccessful, err := s.ec2.DeleteVpcEndpoints(
			parseEC2MembersWithAliases(r.Form, "VpcEndpointId", "VpcEndpointIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteVpcEndpointsResponse{
			XMLName:      xml.Name{Local: "DeleteVpcEndpointsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Unsuccessful: ec2UnsuccessfulItemSet{Items: ec2UnsuccessfulItems(unsuccessful)},
		})
		return true
	default:
		return false
	}
}

type ec2DeleteVpcEndpointsResponse struct {
	XMLName      xml.Name               `xml:"DeleteVpcEndpointsResponse"`
	Xmlns        string                 `xml:"xmlns,attr"`
	RequestID    string                 `xml:"requestId"`
	Unsuccessful ec2UnsuccessfulItemSet `xml:"unsuccessful"`
}
