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

func (s *Server) handleEC2Stage27Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "GetSecurityGroupsForVpc":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		groups, nextToken, err := s.ec2.GetSecurityGroupsForVpc(
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2FilterValues(r.Form, "group-id"),
			parseEC2FilterValues(r.Form, "description"),
			parseEC2FilterValues(r.Form, "group-name"),
			parseEC2FilterValues(r.Form, "owner-id"),
			parseEC2FilterValues(r.Form, "primary-vpc-id"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2GetSecurityGroupsForVpcResponse{
			XMLName:   xml.Name{Local: "GetSecurityGroupsForVpcResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			SecurityGroupForVpcSet: ec2SecurityGroupForVPCSet{
				Items: ec2SecurityGroupForVPCItems(groups),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "DescribeStaleSecurityGroups":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		groups, nextToken, err := s.ec2.DescribeStaleSecurityGroups(
			strings.TrimSpace(r.Form.Get("VpcId")),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeStaleSecurityGroupsResponse{
			XMLName:   xml.Name{Local: "DescribeStaleSecurityGroupsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			StaleSecurityGroupSet: ec2StaleSecurityGroupSet{
				Items: ec2StaleSecurityGroupItems(groups),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "ModifySecurityGroupRules":
		updates, ok := parseEC2SecurityGroupRuleUpdates(r.Form, "SecurityGroupRule.")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		ret, err := s.ec2.ModifySecurityGroupRules(
			strings.TrimSpace(r.Form.Get("GroupId")),
			updates,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifySecurityGroupRulesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}

func parseEC2SecurityGroupRuleUpdates(values url.Values, prefix string) ([]ec2svc.SecurityGroupRuleUpdateRequest, bool) {
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

	out := make([]ec2svc.SecurityGroupRuleUpdateRequest, 0, len(ordered))
	for _, idx := range ordered {
		base := prefix + strconv.Itoa(idx) + "."
		update := ec2svc.SecurityGroupRuleUpdateRequest{
			SecurityGroupRuleID: strings.TrimSpace(values.Get(base + "SecurityGroupRuleId")),
			CidrIPv4:            ec2OptionalStringPointerFromForm(values, base+"SecurityGroupRule.CidrIpv4"),
			CidrIPv6:            ec2OptionalStringPointerFromForm(values, base+"SecurityGroupRule.CidrIpv6"),
			Description:         ec2OptionalStringPointerFromForm(values, base+"SecurityGroupRule.Description"),
			IPProtocol:          ec2OptionalStringPointerFromForm(values, base+"SecurityGroupRule.IpProtocol"),
			PrefixListID:        ec2OptionalStringPointerFromForm(values, base+"SecurityGroupRule.PrefixListId"),
			ReferencedGroupID:   ec2OptionalStringPointerFromForm(values, base+"SecurityGroupRule.ReferencedGroupId"),
		}

		if fromPort, has, valid := ec2OptionalInt32FromForm(values, base+"SecurityGroupRule.FromPort"); has {
			if !valid {
				return nil, false
			}
			update.FromPort = fromPort
		}
		if toPort, has, valid := ec2OptionalInt32FromForm(values, base+"SecurityGroupRule.ToPort"); has {
			if !valid {
				return nil, false
			}
			update.ToPort = toPort
		}

		out = append(out, update)
	}

	return out, true
}

func ec2SecurityGroupForVPCItems(in []ec2svc.SecurityGroupForVPC) []ec2SecurityGroupForVPCItem {
	out := make([]ec2SecurityGroupForVPCItem, 0, len(in))
	for _, group := range in {
		tags := make([]ec2TagItem, 0, len(group.Tags))
		for key, value := range group.Tags {
			tags = append(tags, ec2TagItem{Key: key, Value: value})
		}
		sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

		out = append(out, ec2SecurityGroupForVPCItem{
			Description:  group.Description,
			GroupID:      group.GroupID,
			GroupName:    group.GroupName,
			OwnerID:      group.OwnerID,
			PrimaryVpcID: group.PrimaryVpcID,
			TagSet:       ec2TagSet{Items: tags},
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GroupID < out[j].GroupID })
	return out
}

func ec2StaleSecurityGroupItems(in []ec2svc.StaleSecurityGroup) []ec2StaleSecurityGroupItem {
	out := make([]ec2StaleSecurityGroupItem, 0, len(in))
	for _, group := range in {
		out = append(out, ec2StaleSecurityGroupItem{
			Description:              group.Description,
			GroupID:                  group.GroupID,
			GroupName:                group.GroupName,
			StaleIPPermissions:       ec2StaleIPPermissionSet{Items: ec2StaleIPPermissionItems(group.StaleIPPermissions)},
			StaleIPPermissionsEgress: ec2StaleIPPermissionSet{Items: ec2StaleIPPermissionItems(group.StaleIPPermissionsEgress)},
			VpcID:                    group.VpcID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GroupID < out[j].GroupID })
	return out
}

func ec2StaleIPPermissionItems(in []ec2svc.StaleIPPermission) []ec2StaleIPPermissionItem {
	out := make([]ec2StaleIPPermissionItem, 0, len(in))
	for _, perm := range in {
		out = append(out, ec2StaleIPPermissionItem{
			FromPort:      perm.FromPort,
			IPProtocol:    perm.IPProtocol,
			IPRanges:      ec2ValueStringSet{Items: append([]string(nil), perm.IPRanges...)},
			PrefixListIDs: ec2ValueStringSet{Items: append([]string(nil), perm.PrefixListIDs...)},
			ToPort:        perm.ToPort,
			Groups: ec2UserIDGroupPairSet{
				Items: ec2UserIDGroupPairItems(perm.UserIDGroupPairs),
			},
		})
	}
	return out
}

func ec2UserIDGroupPairItems(in []ec2svc.StaleUserIDGroupPair) []ec2UserIDGroupPairItem {
	out := make([]ec2UserIDGroupPairItem, 0, len(in))
	for _, pair := range in {
		out = append(out, ec2UserIDGroupPairItem{
			GroupID:                pair.GroupID,
			GroupName:              pair.GroupName,
			PeeringStatus:          pair.PeeringStatus,
			UserID:                 pair.UserID,
			VpcID:                  pair.VpcID,
			VpcPeeringConnectionID: pair.VpcPeeringConnectionID,
		})
	}
	return out
}

type ec2GetSecurityGroupsForVpcResponse struct {
	XMLName                xml.Name                  `xml:"GetSecurityGroupsForVpcResponse"`
	Xmlns                  string                    `xml:"xmlns,attr"`
	RequestID              string                    `xml:"requestId"`
	NextToken              string                    `xml:"nextToken,omitempty"`
	SecurityGroupForVpcSet ec2SecurityGroupForVPCSet `xml:"securityGroupForVpcSet"`
}

type ec2SecurityGroupForVPCSet struct {
	Items []ec2SecurityGroupForVPCItem `xml:"item"`
}

type ec2SecurityGroupForVPCItem struct {
	Description  string    `xml:"description,omitempty"`
	GroupID      string    `xml:"groupId,omitempty"`
	GroupName    string    `xml:"groupName,omitempty"`
	OwnerID      string    `xml:"ownerId,omitempty"`
	PrimaryVpcID string    `xml:"primaryVpcId,omitempty"`
	TagSet       ec2TagSet `xml:"tagSet"`
}

type ec2DescribeStaleSecurityGroupsResponse struct {
	XMLName               xml.Name                 `xml:"DescribeStaleSecurityGroupsResponse"`
	Xmlns                 string                   `xml:"xmlns,attr"`
	RequestID             string                   `xml:"requestId"`
	NextToken             string                   `xml:"nextToken,omitempty"`
	StaleSecurityGroupSet ec2StaleSecurityGroupSet `xml:"staleSecurityGroupSet"`
}

type ec2StaleSecurityGroupSet struct {
	Items []ec2StaleSecurityGroupItem `xml:"item"`
}

type ec2StaleSecurityGroupItem struct {
	Description              string                  `xml:"description,omitempty"`
	GroupID                  string                  `xml:"groupId,omitempty"`
	GroupName                string                  `xml:"groupName,omitempty"`
	StaleIPPermissions       ec2StaleIPPermissionSet `xml:"staleIpPermissions"`
	StaleIPPermissionsEgress ec2StaleIPPermissionSet `xml:"staleIpPermissionsEgress"`
	VpcID                    string                  `xml:"vpcId,omitempty"`
}

type ec2StaleIPPermissionSet struct {
	Items []ec2StaleIPPermissionItem `xml:"item"`
}

type ec2StaleIPPermissionItem struct {
	FromPort      *int32                `xml:"fromPort,omitempty"`
	IPProtocol    *string               `xml:"ipProtocol,omitempty"`
	IPRanges      ec2ValueStringSet     `xml:"ipRanges"`
	PrefixListIDs ec2ValueStringSet     `xml:"prefixListIds"`
	ToPort        *int32                `xml:"toPort,omitempty"`
	Groups        ec2UserIDGroupPairSet `xml:"groups"`
}

type ec2UserIDGroupPairSet struct {
	Items []ec2UserIDGroupPairItem `xml:"item"`
}

type ec2UserIDGroupPairItem struct {
	GroupID                string `xml:"groupId,omitempty"`
	GroupName              string `xml:"groupName,omitempty"`
	PeeringStatus          string `xml:"peeringStatus,omitempty"`
	UserID                 string `xml:"userId,omitempty"`
	VpcID                  string `xml:"vpcId,omitempty"`
	VpcPeeringConnectionID string `xml:"vpcPeeringConnectionId,omitempty"`
}
