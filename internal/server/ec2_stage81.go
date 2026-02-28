package server

import (
	"encoding/xml"
	"net/http"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage81Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AllocateIpamPoolCidr":
		netmaskLength, ok := parseEC2OptionalInt32(r.Form.Get("NetmaskLength"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		previewNextCidr, hasPreviewNextCidr, ok := ec2OptionalBoolFromForm(r.Form, "PreviewNextCidr")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasPreviewNextCidr {
			previewNextCidr = nil
		}

		allocation, err := s.ec2.AllocateIpamPoolCidr(
			r.Form.Get("IpamPoolId"),
			parseEC2MembersWithAliases(r.Form, "AllowedCidr", "AllowedCidrs"),
			parseEC2OptionalString(r.Form.Get("Cidr")),
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2MembersWithAliases(r.Form, "DisallowedCidr", "DisallowedCidrs"),
			netmaskLength,
			previewNextCidr,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2AllocateIpamPoolCidrResponse{
			XMLName:   xml.Name{Local: "AllocateIpamPoolCidrResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			IpamPoolAllocation: ec2IpamPoolAllocationItem{
				Cidr:                 allocation.Cidr,
				Description:          allocation.Description,
				IpamPoolAllocationID: allocation.IpamPoolAllocationID,
				ResourceID:           allocation.ResourceID,
				ResourceOwner:        allocation.ResourceOwner,
				ResourceRegion:       allocation.ResourceRegion,
				ResourceType:         allocation.ResourceType,
			},
		})
		return true
	default:
		return false
	}
}

type ec2AllocateIpamPoolCidrResponse struct {
	XMLName            xml.Name                  `xml:"AllocateIpamPoolCidrResponse"`
	Xmlns              string                    `xml:"xmlns,attr"`
	RequestID          string                    `xml:"requestId"`
	IpamPoolAllocation ec2IpamPoolAllocationItem `xml:"ipamPoolAllocation"`
}

type ec2IpamPoolAllocationItem struct {
	Cidr                 string `xml:"cidr,omitempty"`
	Description          string `xml:"description,omitempty"`
	IpamPoolAllocationID string `xml:"ipamPoolAllocationId,omitempty"`
	ResourceID           string `xml:"resourceId,omitempty"`
	ResourceOwner        string `xml:"resourceOwner,omitempty"`
	ResourceRegion       string `xml:"resourceRegion,omitempty"`
	ResourceType         string `xml:"resourceType,omitempty"`
}
