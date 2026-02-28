package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage36Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateTransitGatewayPrefixListReference":
		blackhole, _ := parseEC2OptionalBool(r.Form.Get("Blackhole"))
		reference, err := s.ec2.CreateTransitGatewayPrefixListReference(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("PrefixListId")),
			blackhole,
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateTransitGatewayPrefixListReferenceResponse{
			XMLName:                           xml.Name{Local: "CreateTransitGatewayPrefixListReferenceResponse"},
			Xmlns:                             ec2Namespace,
			RequestID:                         "stackyard-request",
			TransitGatewayPrefixListReference: ec2TransitGatewayPrefixListReferenceItemFrom(reference),
		})
		return true
	case "DeleteTransitGatewayPrefixListReference":
		reference, err := s.ec2.DeleteTransitGatewayPrefixListReference(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("PrefixListId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteTransitGatewayPrefixListReferenceResponse{
			XMLName:                           xml.Name{Local: "DeleteTransitGatewayPrefixListReferenceResponse"},
			Xmlns:                             ec2Namespace,
			RequestID:                         "stackyard-request",
			TransitGatewayPrefixListReference: ec2TransitGatewayPrefixListReferenceItemFrom(reference),
		})
		return true
	case "GetTransitGatewayPrefixListReferences":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		references, nextToken, err := s.ec2.GetTransitGatewayPrefixListReferences(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			parseEC2FilterValues(r.Form, "attachment.resource-id"),
			parseEC2FilterValues(r.Form, "attachment.resource-type"),
			parseEC2FilterValues(r.Form, "attachment.transit-gateway-attachment-id"),
			parseEC2BoolFilterValues(parseEC2FilterValues(r.Form, "is-blackhole")),
			parseEC2FilterValues(r.Form, "prefix-list-id"),
			parseEC2FilterValues(r.Form, "prefix-list-owner-id"),
			parseEC2FilterValues(r.Form, "state"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2GetTransitGatewayPrefixListReferencesResponse{
			XMLName:                              xml.Name{Local: "GetTransitGatewayPrefixListReferencesResponse"},
			Xmlns:                                ec2Namespace,
			RequestID:                            "stackyard-request",
			TransitGatewayPrefixListReferenceSet: ec2TransitGatewayPrefixListReferenceSet{Items: ec2TransitGatewayPrefixListReferenceItems(references)},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "GetTransitGatewayPolicyTableEntries":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		entries, err := s.ec2.GetTransitGatewayPolicyTableEntries(
			strings.TrimSpace(r.Form.Get("TransitGatewayPolicyTableId")),
			parseEC2FilterValues(r.Form, "policy-rule-number"),
			parseEC2FilterValues(r.Form, "target-route-table-id"),
			parseEC2FilterValues(r.Form, "source-cidr-block"),
			parseEC2FilterValues(r.Form, "destination-cidr-block"),
			parseEC2FilterValues(r.Form, "protocol"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2GetTransitGatewayPolicyTableEntriesResponse{
			XMLName:                          xml.Name{Local: "GetTransitGatewayPolicyTableEntriesResponse"},
			Xmlns:                            ec2Namespace,
			RequestID:                        "stackyard-request",
			TransitGatewayPolicyTableEntries: ec2TransitGatewayPolicyTableEntrySet{Items: ec2TransitGatewayPolicyTableEntryItems(entries)},
		})
		return true
	default:
		return false
	}
}

func ec2TransitGatewayPrefixListReferenceItems(in []ec2svc.TransitGatewayPrefixListReference) []ec2TransitGatewayPrefixListReferenceItem {
	out := make([]ec2TransitGatewayPrefixListReferenceItem, 0, len(in))
	for _, reference := range in {
		out = append(out, ec2TransitGatewayPrefixListReferenceItemFrom(reference))
	}
	return out
}

func ec2TransitGatewayPrefixListReferenceItemFrom(in ec2svc.TransitGatewayPrefixListReference) ec2TransitGatewayPrefixListReferenceItem {
	item := ec2TransitGatewayPrefixListReferenceItem{
		Blackhole:                  in.Blackhole,
		PrefixListID:               in.PrefixListID,
		PrefixListOwnerID:          in.PrefixListOwnerID,
		State:                      in.State,
		TransitGatewayRouteTableID: in.TransitGatewayRouteTableID,
	}
	if in.TransitGatewayAttachment != nil {
		item.TransitGatewayAttachment = &ec2TransitGatewayPrefixListAttachmentItem{
			ResourceID:                 in.TransitGatewayAttachment.ResourceID,
			ResourceType:               in.TransitGatewayAttachment.ResourceType,
			TransitGatewayAttachmentID: in.TransitGatewayAttachment.TransitGatewayAttachmentID,
		}
	}
	return item
}

func ec2TransitGatewayPolicyTableEntryItems(in []ec2svc.TransitGatewayPolicyTableEntry) []ec2TransitGatewayPolicyTableEntryItem {
	out := make([]ec2TransitGatewayPolicyTableEntryItem, 0, len(in))
	for _, entry := range in {
		item := ec2TransitGatewayPolicyTableEntryItem{
			PolicyRuleNumber:   entry.PolicyRuleNumber,
			TargetRouteTableID: entry.TargetRouteTableID,
			PolicyRule: ec2TransitGatewayPolicyRuleItem{
				DestinationCidrBlock: entry.PolicyRule.DestinationCidrBlock,
				DestinationPortRange: entry.PolicyRule.DestinationPortRange,
				Protocol:             entry.PolicyRule.Protocol,
				SourceCidrBlock:      entry.PolicyRule.SourceCidrBlock,
				SourcePortRange:      entry.PolicyRule.SourcePortRange,
			},
		}
		if entry.PolicyRule.MetaData != nil {
			item.PolicyRule.MetaData = &ec2TransitGatewayPolicyRuleMetaDataItem{
				MetaDataKey:   entry.PolicyRule.MetaData.MetaDataKey,
				MetaDataValue: entry.PolicyRule.MetaData.MetaDataValue,
			}
		}
		out = append(out, item)
	}
	return out
}

type ec2CreateTransitGatewayPrefixListReferenceResponse struct {
	XMLName                           xml.Name                                 `xml:"CreateTransitGatewayPrefixListReferenceResponse"`
	Xmlns                             string                                   `xml:"xmlns,attr"`
	RequestID                         string                                   `xml:"requestId"`
	TransitGatewayPrefixListReference ec2TransitGatewayPrefixListReferenceItem `xml:"transitGatewayPrefixListReference"`
}

type ec2DeleteTransitGatewayPrefixListReferenceResponse struct {
	XMLName                           xml.Name                                 `xml:"DeleteTransitGatewayPrefixListReferenceResponse"`
	Xmlns                             string                                   `xml:"xmlns,attr"`
	RequestID                         string                                   `xml:"requestId"`
	TransitGatewayPrefixListReference ec2TransitGatewayPrefixListReferenceItem `xml:"transitGatewayPrefixListReference"`
}

type ec2TransitGatewayPrefixListReferenceItem struct {
	Blackhole                  bool                                       `xml:"blackhole"`
	PrefixListID               string                                     `xml:"prefixListId,omitempty"`
	PrefixListOwnerID          string                                     `xml:"prefixListOwnerId,omitempty"`
	State                      string                                     `xml:"state,omitempty"`
	TransitGatewayAttachment   *ec2TransitGatewayPrefixListAttachmentItem `xml:"transitGatewayAttachment,omitempty"`
	TransitGatewayRouteTableID string                                     `xml:"transitGatewayRouteTableId,omitempty"`
}

type ec2TransitGatewayPrefixListAttachmentItem struct {
	ResourceID                 string `xml:"resourceId,omitempty"`
	ResourceType               string `xml:"resourceType,omitempty"`
	TransitGatewayAttachmentID string `xml:"transitGatewayAttachmentId,omitempty"`
}

type ec2GetTransitGatewayPrefixListReferencesResponse struct {
	XMLName                              xml.Name                                `xml:"GetTransitGatewayPrefixListReferencesResponse"`
	Xmlns                                string                                  `xml:"xmlns,attr"`
	RequestID                            string                                  `xml:"requestId"`
	TransitGatewayPrefixListReferenceSet ec2TransitGatewayPrefixListReferenceSet `xml:"transitGatewayPrefixListReferenceSet"`
	NextToken                            string                                  `xml:"nextToken,omitempty"`
}

type ec2TransitGatewayPrefixListReferenceSet struct {
	Items []ec2TransitGatewayPrefixListReferenceItem `xml:"item"`
}

type ec2GetTransitGatewayPolicyTableEntriesResponse struct {
	XMLName                          xml.Name                             `xml:"GetTransitGatewayPolicyTableEntriesResponse"`
	Xmlns                            string                               `xml:"xmlns,attr"`
	RequestID                        string                               `xml:"requestId"`
	TransitGatewayPolicyTableEntries ec2TransitGatewayPolicyTableEntrySet `xml:"transitGatewayPolicyTableEntries"`
}

type ec2TransitGatewayPolicyTableEntrySet struct {
	Items []ec2TransitGatewayPolicyTableEntryItem `xml:"item"`
}

type ec2TransitGatewayPolicyTableEntryItem struct {
	PolicyRule         ec2TransitGatewayPolicyRuleItem `xml:"policyRule"`
	PolicyRuleNumber   string                          `xml:"policyRuleNumber,omitempty"`
	TargetRouteTableID string                          `xml:"targetRouteTableId,omitempty"`
}

type ec2TransitGatewayPolicyRuleItem struct {
	DestinationCidrBlock string                                   `xml:"destinationCidrBlock,omitempty"`
	DestinationPortRange string                                   `xml:"destinationPortRange,omitempty"`
	MetaData             *ec2TransitGatewayPolicyRuleMetaDataItem `xml:"metaData,omitempty"`
	Protocol             string                                   `xml:"protocol,omitempty"`
	SourceCidrBlock      string                                   `xml:"sourceCidrBlock,omitempty"`
	SourcePortRange      string                                   `xml:"sourcePortRange,omitempty"`
}

type ec2TransitGatewayPolicyRuleMetaDataItem struct {
	MetaDataKey   string `xml:"metaDataKey,omitempty"`
	MetaDataValue string `xml:"metaDataValue,omitempty"`
}
