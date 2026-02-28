package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage105Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateCarrierGateway":
		tags := parseEC2TagSpecificationsForResource(r.Form, "carrier-gateway")
		gateway, err := s.ec2.CreateCarrierGateway(
			strings.TrimSpace(r.Form.Get("VpcId")),
			tags,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2Stage105CreateCarrierGatewayResponse{
			XMLName:   xml.Name{Local: "CreateCarrierGatewayResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			CarrierGateway: ec2Stage105CarrierGatewayItem{
				CarrierGatewayID: gateway.ID,
				OwnerID:          gateway.OwnerID,
				State:            gateway.State,
				TagSet:           ec2TagSet{Items: ec2TagItemsFromMap(gateway.Tags)},
				VpcID:            gateway.VpcID,
			},
		})
		return true
	default:
		return false
	}
}

type ec2Stage105CreateCarrierGatewayResponse struct {
	XMLName        xml.Name                      `xml:"CreateCarrierGatewayResponse"`
	Xmlns          string                        `xml:"xmlns,attr"`
	RequestID      string                        `xml:"requestId"`
	CarrierGateway ec2Stage105CarrierGatewayItem `xml:"carrierGateway"`
}

type ec2Stage105CarrierGatewayItem struct {
	CarrierGatewayID string    `xml:"carrierGatewayId,omitempty"`
	OwnerID          string    `xml:"ownerId,omitempty"`
	State            string    `xml:"state,omitempty"`
	TagSet           ec2TagSet `xml:"tagSet"`
	VpcID            string    `xml:"vpcId,omitempty"`
}
