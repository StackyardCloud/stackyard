package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage18Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateVpnGateway":
		amazonSideASN, ok := parseEC2OptionalInt64(r.Form.Get("AmazonSideAsn"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		gateway, err := s.ec2.CreateVpnGateway(
			strings.TrimSpace(r.Form.Get("Type")),
			amazonSideASN,
			strings.TrimSpace(r.Form.Get("AvailabilityZone")),
			parseEC2TagSpecificationsForResource(r.Form, "vpn-gateway"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateVpnGatewayResponse{
			XMLName:    xml.Name{Local: "CreateVpnGatewayResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			VpnGateway: ec2VpnGatewayItemFrom(gateway),
		})
		return true
	case "DescribeVpnGateways":
		gateways := s.ec2.DescribeVpnGateways(
			parseEC2Members(r.Form, "VpnGatewayId."),
			parseEC2FilterValues(r.Form, "vpn-gateway-id"),
			parseEC2FilterValues(r.Form, "attachment.vpc-id"),
			parseEC2FilterValues(r.Form, "attachment.state"),
			parseEC2FilterValues(r.Form, "state"),
			parseEC2FilterValues(r.Form, "type"),
			parseEC2FilterValues(r.Form, "availability-zone"),
			parseEC2FilterValues(r.Form, "amazon-side-asn"),
			parseEC2FilterValues(r.Form, "tag-key"),
			parseEC2TagValueFilters(r.Form),
		)
		respondEC2XML(w, ec2DescribeVpnGatewaysResponse{
			XMLName:   xml.Name{Local: "DescribeVpnGatewaysResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpnGatewaySet: ec2VpnGatewaySet{
				Items: ec2VpnGatewayItems(gateways),
			},
		})
		return true
	case "AttachVpnGateway":
		attachment, err := s.ec2.AttachVpnGateway(
			strings.TrimSpace(r.Form.Get("VpnGatewayId")),
			strings.TrimSpace(r.Form.Get("VpcId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AttachVpnGatewayResponse{
			XMLName:       xml.Name{Local: "AttachVpnGatewayResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			VpcAttachment: ec2VpcAttachmentItemFrom(attachment),
		})
		return true
	case "DetachVpnGateway":
		if err := s.ec2.DetachVpnGateway(strings.TrimSpace(r.Form.Get("VpnGatewayId")), strings.TrimSpace(r.Form.Get("VpcId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DetachVpnGatewayResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DeleteVpnGateway":
		if err := s.ec2.DeleteVpnGateway(strings.TrimSpace(r.Form.Get("VpnGatewayId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteVpnGatewayResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	default:
		return false
	}
}

func ec2VpnGatewayItems(in []ec2svc.VpnGateway) []ec2VpnGatewayItem {
	out := make([]ec2VpnGatewayItem, 0, len(in))
	for _, gateway := range in {
		out = append(out, ec2VpnGatewayItemFrom(gateway))
	}
	return out
}

func ec2VpnGatewayItemFrom(in ec2svc.VpnGateway) ec2VpnGatewayItem {
	attachments := make([]ec2VpcAttachmentItem, 0, len(in.Attachments))
	for _, attachment := range in.Attachments {
		attachments = append(attachments, ec2VpcAttachmentItemFrom(attachment))
	}
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	return ec2VpnGatewayItem{
		AmazonSideASN:    in.AmazonSideASN,
		AvailabilityZone: in.AvailabilityZone,
		State:            in.State,
		Type:             in.Type,
		VpcAttachmentSet: ec2VpcAttachmentSet{Items: attachments},
		TagSet:           ec2TagSet{Items: tags},
		VpnGatewayID:     in.ID,
	}
}

func ec2VpcAttachmentItemFrom(in ec2svc.VpnGatewayAttachment) ec2VpcAttachmentItem {
	return ec2VpcAttachmentItem{
		State: in.State,
		VpcID: in.VpcID,
	}
}

type ec2CreateVpnGatewayResponse struct {
	XMLName    xml.Name
	Xmlns      string            `xml:"xmlns,attr"`
	RequestID  string            `xml:"requestId"`
	VpnGateway ec2VpnGatewayItem `xml:"vpnGateway"`
}

type ec2DescribeVpnGatewaysResponse struct {
	XMLName       xml.Name         `xml:"DescribeVpnGatewaysResponse"`
	Xmlns         string           `xml:"xmlns,attr"`
	RequestID     string           `xml:"requestId"`
	VpnGatewaySet ec2VpnGatewaySet `xml:"vpnGatewaySet"`
}

type ec2AttachVpnGatewayResponse struct {
	XMLName       xml.Name             `xml:"AttachVpnGatewayResponse"`
	Xmlns         string               `xml:"xmlns,attr"`
	RequestID     string               `xml:"requestId"`
	VpcAttachment ec2VpcAttachmentItem `xml:"attachment"`
}

type ec2VpnGatewaySet struct {
	Items []ec2VpnGatewayItem `xml:"item"`
}

type ec2VpnGatewayItem struct {
	AmazonSideASN    int64               `xml:"amazonSideAsn,omitempty"`
	AvailabilityZone string              `xml:"availabilityZone,omitempty"`
	State            string              `xml:"state,omitempty"`
	Type             string              `xml:"type,omitempty"`
	VpcAttachmentSet ec2VpcAttachmentSet `xml:"attachments,omitempty"`
	TagSet           ec2TagSet           `xml:"tagSet,omitempty"`
	VpnGatewayID     string              `xml:"vpnGatewayId,omitempty"`
}

type ec2VpcAttachmentSet struct {
	Items []ec2VpcAttachmentItem `xml:"item"`
}

type ec2VpcAttachmentItem struct {
	State string `xml:"state,omitempty"`
	VpcID string `xml:"vpcId,omitempty"`
}
