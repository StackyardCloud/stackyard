package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage24Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "MoveAddressToVpc":
		allocationID, status, err := s.ec2.MoveAddressToVpc(strings.TrimSpace(r.Form.Get("PublicIp")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2MoveAddressToVpcResponse{
			XMLName:      xml.Name{Local: "MoveAddressToVpcResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			AllocationID: allocationID,
			Status:       status,
		})
		return true
	case "RestoreAddressToClassic":
		publicIP, status, err := s.ec2.RestoreAddressToClassic(strings.TrimSpace(r.Form.Get("PublicIp")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2RestoreAddressToClassicResponse{
			XMLName:   xml.Name{Local: "RestoreAddressToClassicResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			PublicIP:  publicIP,
			Status:    status,
		})
		return true
	case "ModifyAddressAttribute":
		attr, err := s.ec2.ModifyAddressAttribute(
			strings.TrimSpace(r.Form.Get("AllocationId")),
			strings.TrimSpace(r.Form.Get("DomainName")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyAddressAttributeResponse{
			XMLName:   xml.Name{Local: "ModifyAddressAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Address:   ec2AddressAttributeItemFrom(attr),
		})
		return true
	case "ResetAddressAttribute":
		attr, err := s.ec2.ResetAddressAttribute(
			strings.TrimSpace(r.Form.Get("AllocationId")),
			strings.TrimSpace(r.Form.Get("Attribute")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ResetAddressAttributeResponse{
			XMLName:   xml.Name{Local: "ResetAddressAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Address:   ec2AddressAttributeItemFrom(attr),
		})
		return true
	default:
		return false
	}
}

func ec2AddressAttributeItemFrom(in ec2svc.AddressAttribute) ec2AddressAttributeItem {
	return ec2AddressAttributeItem{
		AllocationID: in.AllocationID,
		PublicIP:     in.PublicIP,
		PtrRecord:    in.PtrRecord,
	}
}

type ec2MoveAddressToVpcResponse struct {
	XMLName      xml.Name
	Xmlns        string `xml:"xmlns,attr"`
	RequestID    string `xml:"requestId"`
	AllocationID string `xml:"allocationId,omitempty"`
	Status       string `xml:"status,omitempty"`
}

type ec2RestoreAddressToClassicResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	PublicIP  string `xml:"publicIp,omitempty"`
	Status    string `xml:"status,omitempty"`
}

type ec2ModifyAddressAttributeResponse struct {
	XMLName   xml.Name                `xml:"ModifyAddressAttributeResponse"`
	Xmlns     string                  `xml:"xmlns,attr"`
	RequestID string                  `xml:"requestId"`
	Address   ec2AddressAttributeItem `xml:"address"`
}

type ec2ResetAddressAttributeResponse struct {
	XMLName   xml.Name                `xml:"ResetAddressAttributeResponse"`
	Xmlns     string                  `xml:"xmlns,attr"`
	RequestID string                  `xml:"requestId"`
	Address   ec2AddressAttributeItem `xml:"address"`
}
