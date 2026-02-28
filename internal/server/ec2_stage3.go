package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage3Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateNetworkAclEntry":
		if err := s.ec2.CreateNetworkACLEntry(
			strings.TrimSpace(r.Form.Get("NetworkAclId")),
			parseEC2Int32(r.Form.Get("RuleNumber"), 0),
			strings.TrimSpace(r.Form.Get("Protocol")),
			strings.TrimSpace(r.Form.Get("RuleAction")),
			parseEC2Bool(r.Form.Get("Egress"), false),
			strings.TrimSpace(r.Form.Get("CidrBlock")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "CreateNetworkAclEntryResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "ReplaceNetworkAclEntry":
		if err := s.ec2.ReplaceNetworkACLEntry(
			strings.TrimSpace(r.Form.Get("NetworkAclId")),
			parseEC2Int32(r.Form.Get("RuleNumber"), 0),
			strings.TrimSpace(r.Form.Get("Protocol")),
			strings.TrimSpace(r.Form.Get("RuleAction")),
			parseEC2Bool(r.Form.Get("Egress"), false),
			strings.TrimSpace(r.Form.Get("CidrBlock")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ReplaceNetworkAclEntryResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DeleteNetworkAclEntry":
		if err := s.ec2.DeleteNetworkACLEntry(
			strings.TrimSpace(r.Form.Get("NetworkAclId")),
			parseEC2Int32(r.Form.Get("RuleNumber"), 0),
			parseEC2Bool(r.Form.Get("Egress"), false),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteNetworkAclEntryResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "ReplaceRoute":
		if err := s.ec2.ReplaceRoute(
			strings.TrimSpace(r.Form.Get("RouteTableId")),
			strings.TrimSpace(r.Form.Get("DestinationCidrBlock")),
			strings.TrimSpace(r.Form.Get("GatewayId")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ReplaceRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "ReplaceRouteTableAssociation":
		association, err := s.ec2.ReplaceRouteTableAssociation(
			strings.TrimSpace(r.Form.Get("AssociationId")),
			strings.TrimSpace(r.Form.Get("RouteTableId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ReplaceRouteTableAssociationResponse{
			XMLName:          xml.Name{Local: "ReplaceRouteTableAssociationResponse"},
			Xmlns:            ec2Namespace,
			RequestID:        "stackyard-request",
			NewAssociationID: association.ID,
		})
		return true
	case "ModifyNetworkInterfaceAttribute":
		var description *string
		if raw := strings.TrimSpace(r.Form.Get("Description.Value")); raw != "" {
			description = &raw
		}
		var sourceDestCheck *bool
		if value, ok := parseEC2OptionalBool(r.Form.Get("SourceDestCheck.Value")); ok {
			sourceDestCheck = &value
		}
		groupIDs := parseEC2Members(r.Form, "GroupId.")
		if err := s.ec2.ModifyNetworkInterfaceAttribute(strings.TrimSpace(r.Form.Get("NetworkInterfaceId")), description, sourceDestCheck, groupIDs); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyNetworkInterfaceAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	case "ModifySubnetAttribute":
		value, ok := parseEC2OptionalBool(r.Form.Get("MapPublicIpOnLaunch.Value"))
		if !ok {
			respondEC2ErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MapPublicIpOnLaunch.Value is required")
			return true
		}
		if err := s.ec2.ModifySubnetAttribute(strings.TrimSpace(r.Form.Get("SubnetId")), &value); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifySubnetAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	case "ModifyVpcAttribute":
		var dnsSupport *bool
		if value, ok := parseEC2OptionalBool(r.Form.Get("EnableDnsSupport.Value")); ok {
			dnsSupport = &value
		}
		var dnsHostnames *bool
		if value, ok := parseEC2OptionalBool(r.Form.Get("EnableDnsHostnames.Value")); ok {
			dnsHostnames = &value
		}
		if err := s.ec2.ModifyVpcAttribute(strings.TrimSpace(r.Form.Get("VpcId")), dnsSupport, dnsHostnames); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyVpcAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	case "DescribeVpcAttribute":
		attr, err := s.ec2.DescribeVpcAttribute(strings.TrimSpace(r.Form.Get("VpcId")), strings.TrimSpace(r.Form.Get("Attribute")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		resp := ec2DescribeVpcAttributeResponse{
			XMLName:   xml.Name{Local: "DescribeVpcAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpcID:     attr.VpcID,
		}
		switch attr.Attribute {
		case "enableDnsSupport":
			resp.EnableDnsSupport = &ec2AttributeBooleanValue{Value: attr.Value}
		case "enableDnsHostnames":
			resp.EnableDnsHostnames = &ec2AttributeBooleanValue{Value: attr.Value}
		}
		respondEC2XML(w, resp)
		return true
	case "DescribeAccountAttributes":
		attributes := s.ec2.DescribeAccountAttributes(parseEC2Members(r.Form, "AttributeName."))
		respondEC2XML(w, ec2DescribeAccountAttributesResponse{
			XMLName:             xml.Name{Local: "DescribeAccountAttributesResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			AccountAttributeSet: ec2AccountAttributeSet{Items: ec2AccountAttributeItems(attributes)},
		})
		return true
	default:
		return false
	}
}

func ec2AccountAttributeItems(in []ec2svc.AccountAttribute) []ec2AccountAttributeItem {
	out := make([]ec2AccountAttributeItem, 0, len(in))
	for _, attr := range in {
		values := make([]ec2AccountAttributeValueItem, 0, len(attr.Values))
		for _, value := range attr.Values {
			values = append(values, ec2AccountAttributeValueItem{AttributeValue: value})
		}
		out = append(out, ec2AccountAttributeItem{
			AttributeName:  attr.Name,
			AttributeValue: ec2AccountAttributeValueSet{Items: values},
		})
	}
	return out
}

func parseEC2OptionalBool(value string) (bool, bool) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return false, false
	}
	return parseEC2Bool(raw, false), true
}

type ec2ReplaceRouteTableAssociationResponse struct {
	XMLName          xml.Name
	Xmlns            string `xml:"xmlns,attr"`
	RequestID        string `xml:"requestId"`
	NewAssociationID string `xml:"newAssociationId"`
}

type ec2DescribeVpcAttributeResponse struct {
	XMLName            xml.Name
	Xmlns              string                    `xml:"xmlns,attr"`
	RequestID          string                    `xml:"requestId"`
	VpcID              string                    `xml:"vpcId"`
	EnableDnsSupport   *ec2AttributeBooleanValue `xml:"enableDnsSupport,omitempty"`
	EnableDnsHostnames *ec2AttributeBooleanValue `xml:"enableDnsHostnames,omitempty"`
}

type ec2AttributeBooleanValue struct {
	Value bool `xml:"value"`
}

type ec2DescribeAccountAttributesResponse struct {
	XMLName             xml.Name               `xml:"DescribeAccountAttributesResponse"`
	Xmlns               string                 `xml:"xmlns,attr"`
	RequestID           string                 `xml:"requestId"`
	AccountAttributeSet ec2AccountAttributeSet `xml:"accountAttributeSet"`
}

type ec2AccountAttributeSet struct {
	Items []ec2AccountAttributeItem `xml:"item"`
}

type ec2AccountAttributeItem struct {
	AttributeName  string                      `xml:"attributeName"`
	AttributeValue ec2AccountAttributeValueSet `xml:"attributeValueSet"`
}

type ec2AccountAttributeValueSet struct {
	Items []ec2AccountAttributeValueItem `xml:"item"`
}

type ec2AccountAttributeValueItem struct {
	AttributeValue string `xml:"attributeValue"`
}
