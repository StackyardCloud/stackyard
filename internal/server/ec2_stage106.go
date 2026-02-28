package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage106Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateCoipCidr":
		coipCidr, err := s.ec2.CreateCoipCidr(
			strings.TrimSpace(r.Form.Get("Cidr")),
			strings.TrimSpace(r.Form.Get("CoipPoolId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2Stage106CreateCoipCidrResponse{
			XMLName:   xml.Name{Local: "CreateCoipCidrResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			CoipCidr: ec2Stage106CoipCidrItem{
				Cidr:                     coipCidr.Cidr,
				CoipPoolID:               coipCidr.CoipPoolID,
				LocalGatewayRouteTableID: coipCidr.LocalGatewayRouteTableID,
			},
		})
		return true
	default:
		return false
	}
}

type ec2Stage106CreateCoipCidrResponse struct {
	XMLName   xml.Name                `xml:"CreateCoipCidrResponse"`
	Xmlns     string                  `xml:"xmlns,attr"`
	RequestID string                  `xml:"requestId"`
	CoipCidr  ec2Stage106CoipCidrItem `xml:"coipCidr"`
}

type ec2Stage106CoipCidrItem struct {
	Cidr                     string `xml:"cidr,omitempty"`
	CoipPoolID               string `xml:"coipPoolId,omitempty"`
	LocalGatewayRouteTableID string `xml:"localGatewayRouteTableId,omitempty"`
}
