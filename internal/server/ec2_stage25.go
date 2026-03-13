package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage25Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeSecurityGroupReferences":
		references, err := s.ec2.DescribeSecurityGroupReferences(parseEC2Members(r.Form, "GroupId."))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DescribeSecurityGroupReferencesResponse{
			XMLName:   xml.Name{Local: "DescribeSecurityGroupReferencesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			SecurityGroupReferenceSet: ec2SecurityGroupReferenceSet{
				Items: ec2SecurityGroupReferenceItems(references),
			},
		})
		return true
	case "DescribeSecurityGroupRules":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		ruleIDs := dedupeStrings(append(
			parseEC2Members(r.Form, "SecurityGroupRuleId."),
			parseEC2FilterValues(r.Form, "security-group-rule-id")...,
		))
		groupIDs := parseEC2FilterValues(r.Form, "group-id")
		rules, nextToken, err := s.ec2.DescribeSecurityGroupRules(ruleIDs, groupIDs, maxResults, parseEC2OptionalString(r.Form.Get("NextToken")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeSecurityGroupRulesResponse{
			XMLName:   xml.Name{Local: "DescribeSecurityGroupRulesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			SecurityGroupRuleSet: ec2SecurityGroupRuleSet{
				Items: ec2SecurityGroupRuleItems(rules),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "UpdateSecurityGroupRuleDescriptionsIngress":
		ret, err := s.ec2.UpdateSecurityGroupRuleDescriptionsIngress(
			strings.TrimSpace(r.Form.Get("GroupId")),
			strings.TrimSpace(r.Form.Get("GroupName")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2IPPermissions(r.Form, "IpPermissions."),
			parseEC2SecurityGroupRuleDescriptions(r.Form, "SecurityGroupRuleDescription."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "UpdateSecurityGroupRuleDescriptionsIngressResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "UpdateSecurityGroupRuleDescriptionsEgress":
		ret, err := s.ec2.UpdateSecurityGroupRuleDescriptionsEgress(
			strings.TrimSpace(r.Form.Get("GroupId")),
			strings.TrimSpace(r.Form.Get("GroupName")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2IPPermissions(r.Form, "IpPermissions."),
			parseEC2SecurityGroupRuleDescriptions(r.Form, "SecurityGroupRuleDescription."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "UpdateSecurityGroupRuleDescriptionsEgressResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}

func dedupeStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func parseEC2SecurityGroupRuleDescriptions(values url.Values, prefix string) []ec2svc.SecurityGroupRuleDescription {
	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		suffix := strings.TrimPrefix(key, prefix)
		part := suffix
		if dot := strings.IndexByte(part, '.'); dot >= 0 {
			part = part[:dot]
		}
		idx, err := strconv.Atoi(part)
		if err != nil || idx <= 0 {
			continue
		}
		indices[idx] = struct{}{}
	}
	ordered := make([]int, 0, len(indices))
	for idx := range indices {
		ordered = append(ordered, idx)
	}
	sort.Ints(ordered)

	out := make([]ec2svc.SecurityGroupRuleDescription, 0, len(ordered))
	for _, idx := range ordered {
		base := prefix + strconv.Itoa(idx) + "."
		ruleID := strings.TrimSpace(values.Get(base + "SecurityGroupRuleId"))
		if ruleID == "" {
			continue
		}
		out = append(out, ec2svc.SecurityGroupRuleDescription{
			SecurityGroupRuleID: ruleID,
			Description:         strings.TrimSpace(values.Get(base + "Description")),
		})
	}
	return out
}

func ec2SecurityGroupReferenceItems(in []ec2svc.SecurityGroupReference) []ec2SecurityGroupReferenceItem {
	out := make([]ec2SecurityGroupReferenceItem, 0, len(in))
	for _, reference := range in {
		out = append(out, ec2SecurityGroupReferenceItem{
			GroupID:                reference.GroupID,
			ReferencingVpcID:       reference.ReferencingVpcID,
			TransitGatewayID:       reference.TransitGatewayID,
			VpcPeeringConnectionID: reference.VpcPeeringConnectionID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GroupID < out[j].GroupID })
	return out
}

func ec2SecurityGroupRuleItems(in []ec2svc.SecurityGroupRule) []ec2SecurityGroupRuleItem {
	out := make([]ec2SecurityGroupRuleItem, 0, len(in))
	for _, rule := range in {
		out = append(out, ec2SecurityGroupRuleItem{
			CidrIPv4:             rule.CidrIPv4,
			Description:          rule.Description,
			FromPort:             rule.FromPort,
			GroupID:              rule.GroupID,
			GroupOwnerID:         rule.GroupOwnerID,
			IPProtocol:           rule.IPProtocol,
			IsEgress:             rule.IsEgress,
			SecurityGroupRuleID:  rule.SecurityGroupRuleID,
			ToPort:               rule.ToPort,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SecurityGroupRuleID < out[j].SecurityGroupRuleID })
	return out
}

type ec2DescribeSecurityGroupReferencesResponse struct {
	XMLName                   xml.Name                     `xml:"DescribeSecurityGroupReferencesResponse"`
	Xmlns                     string                       `xml:"xmlns,attr"`
	RequestID                 string                       `xml:"requestId"`
	SecurityGroupReferenceSet ec2SecurityGroupReferenceSet `xml:"securityGroupReferenceSet"`
}

type ec2SecurityGroupReferenceSet struct {
	Items []ec2SecurityGroupReferenceItem `xml:"item"`
}

type ec2SecurityGroupReferenceItem struct {
	GroupID                string `xml:"groupId,omitempty"`
	ReferencingVpcID       string `xml:"referencingVpcId,omitempty"`
	TransitGatewayID       string `xml:"transitGatewayId,omitempty"`
	VpcPeeringConnectionID string `xml:"vpcPeeringConnectionId,omitempty"`
}

type ec2DescribeSecurityGroupRulesResponse struct {
	XMLName              xml.Name                `xml:"DescribeSecurityGroupRulesResponse"`
	Xmlns                string                  `xml:"xmlns,attr"`
	RequestID            string                  `xml:"requestId"`
	NextToken            string                  `xml:"nextToken,omitempty"`
	SecurityGroupRuleSet ec2SecurityGroupRuleSet `xml:"securityGroupRuleSet"`
}

type ec2SecurityGroupRuleSet struct {
	Items []ec2SecurityGroupRuleItem `xml:"item"`
}

type ec2SecurityGroupRuleItem struct {
	CidrIPv4             string `xml:"cidrIpv4,omitempty"`
	CidrIPv6             string `xml:"cidrIpv6,omitempty"`
	Description          string `xml:"description,omitempty"`
	FromPort             int32  `xml:"fromPort,omitempty"`
	GroupID              string `xml:"groupId,omitempty"`
	GroupOwnerID         string `xml:"groupOwnerId,omitempty"`
	IPProtocol           string `xml:"ipProtocol,omitempty"`
	IsEgress             bool   `xml:"isEgress"`
	PrefixListID         string `xml:"prefixListId,omitempty"`
	SecurityGroupRuleID  string `xml:"securityGroupRuleId,omitempty"`
	ToPort               int32  `xml:"toPort,omitempty"`
}
