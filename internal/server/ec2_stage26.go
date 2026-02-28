package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage26Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AssociateSecurityGroupVpc":
		state, err := s.ec2.AssociateSecurityGroupVpc(
			strings.TrimSpace(r.Form.Get("GroupId")),
			strings.TrimSpace(r.Form.Get("VpcId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssociateSecurityGroupVpcResponse{
			XMLName:   xml.Name{Local: "AssociateSecurityGroupVpcResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			State:     state,
		})
		return true
	case "DisassociateSecurityGroupVpc":
		state, err := s.ec2.DisassociateSecurityGroupVpc(
			strings.TrimSpace(r.Form.Get("GroupId")),
			strings.TrimSpace(r.Form.Get("VpcId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DisassociateSecurityGroupVpcResponse{
			XMLName:   xml.Name{Local: "DisassociateSecurityGroupVpcResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			State:     state,
		})
		return true
	case "DescribeSecurityGroupVpcAssociations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		states := parseEC2FilterValues(r.Form, "state")
		for i := range states {
			states[i] = strings.ToLower(strings.TrimSpace(states[i]))
		}
		associations, nextToken, err := s.ec2.DescribeSecurityGroupVpcAssociations(
			parseEC2FilterValues(r.Form, "group-id"),
			parseEC2FilterValues(r.Form, "group-owner-id"),
			states,
			parseEC2FilterValues(r.Form, "vpc-id"),
			parseEC2FilterValues(r.Form, "vpc-owner-id"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeSecurityGroupVpcAssociationsResponse{
			XMLName:   xml.Name{Local: "DescribeSecurityGroupVpcAssociationsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			SecurityGroupVpcAssociationSet: ec2SecurityGroupVpcAssociationSet{
				Items: ec2SecurityGroupVpcAssociationItems(associations),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	default:
		return false
	}
}

func ec2SecurityGroupVpcAssociationItems(in []ec2svc.SecurityGroupVpcAssociation) []ec2SecurityGroupVpcAssociationItem {
	out := make([]ec2SecurityGroupVpcAssociationItem, 0, len(in))
	for _, association := range in {
		out = append(out, ec2SecurityGroupVpcAssociationItem{
			GroupID:      association.GroupID,
			GroupOwnerID: association.GroupOwnerID,
			State:        association.State,
			StateReason:  association.StateReason,
			VpcID:        association.VpcID,
			VpcOwnerID:   association.VpcOwnerID,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GroupID != out[j].GroupID {
			return out[i].GroupID < out[j].GroupID
		}
		return out[i].VpcID < out[j].VpcID
	})
	return out
}

type ec2AssociateSecurityGroupVpcResponse struct {
	XMLName   xml.Name `xml:"AssociateSecurityGroupVpcResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	State     string   `xml:"state,omitempty"`
}

type ec2DisassociateSecurityGroupVpcResponse struct {
	XMLName   xml.Name `xml:"DisassociateSecurityGroupVpcResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	State     string   `xml:"state,omitempty"`
}

type ec2DescribeSecurityGroupVpcAssociationsResponse struct {
	XMLName                        xml.Name                          `xml:"DescribeSecurityGroupVpcAssociationsResponse"`
	Xmlns                          string                            `xml:"xmlns,attr"`
	RequestID                      string                            `xml:"requestId"`
	NextToken                      string                            `xml:"nextToken,omitempty"`
	SecurityGroupVpcAssociationSet ec2SecurityGroupVpcAssociationSet `xml:"securityGroupVpcAssociationSet"`
}

type ec2SecurityGroupVpcAssociationSet struct {
	Items []ec2SecurityGroupVpcAssociationItem `xml:"item"`
}

type ec2SecurityGroupVpcAssociationItem struct {
	GroupID      string `xml:"groupId,omitempty"`
	GroupOwnerID string `xml:"groupOwnerId,omitempty"`
	State        string `xml:"state,omitempty"`
	StateReason  string `xml:"stateReason,omitempty"`
	VpcID        string `xml:"vpcId,omitempty"`
	VpcOwnerID   string `xml:"vpcOwnerId,omitempty"`
}
