package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage23Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AttachClassicLinkVpc":
		ret, err := s.ec2.AttachClassicLinkVpc(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2Members(r.Form, "SecurityGroupId."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "AttachClassicLinkVpcResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "DetachClassicLinkVpc":
		ret, err := s.ec2.DetachClassicLinkVpc(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("VpcId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DetachClassicLinkVpcResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "EnableVpcClassicLink":
		ret, err := s.ec2.EnableVpcClassicLink(strings.TrimSpace(r.Form.Get("VpcId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "EnableVpcClassicLinkResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "DisableVpcClassicLink":
		ret, err := s.ec2.DisableVpcClassicLink(strings.TrimSpace(r.Form.Get("VpcId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DisableVpcClassicLinkResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "EnableVpcClassicLinkDnsSupport":
		ret, err := s.ec2.EnableVpcClassicLinkDnsSupport(strings.TrimSpace(r.Form.Get("VpcId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "EnableVpcClassicLinkDnsSupportResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "DisableVpcClassicLinkDnsSupport":
		ret, err := s.ec2.DisableVpcClassicLinkDnsSupport(strings.TrimSpace(r.Form.Get("VpcId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DisableVpcClassicLinkDnsSupportResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "DescribeVpcClassicLink":
		vpcIDs := parseEC2Members(r.Form, "VpcId.")
		if len(vpcIDs) == 0 {
			vpcIDs = parseEC2FilterValues(r.Form, "vpc-id")
		}
		vpcs := s.ec2.DescribeVpcClassicLink(
			vpcIDs,
			parseEC2BoolFilterValues(parseEC2FilterValues(r.Form, "is-classic-link-enabled")),
		)
		respondEC2XML(w, ec2DescribeVpcClassicLinkResponse{
			XMLName:   xml.Name{Local: "DescribeVpcClassicLinkResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpcSet: ec2VpcClassicLinkSet{
				Items: ec2VpcClassicLinkItems(vpcs),
			},
		})
		return true
	case "DescribeVpcClassicLinkDnsSupport":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		vpcIDs := parseEC2Members(r.Form, "VpcIds.")
		if len(vpcIDs) == 0 {
			vpcIDs = parseEC2Members(r.Form, "VpcId.")
		}
		vpcs, nextToken, err := s.ec2.DescribeVpcClassicLinkDnsSupport(
			vpcIDs,
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVpcClassicLinkDnsSupportResponse{
			XMLName:   xml.Name{Local: "DescribeVpcClassicLinkDnsSupportResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Vpcs: ec2ClassicLinkDnsSupportSet{
				Items: ec2ClassicLinkDnsSupportItems(vpcs),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "DescribeClassicLinkInstances":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		instanceIDs := parseEC2Members(r.Form, "InstanceId.")
		if len(instanceIDs) == 0 {
			instanceIDs = parseEC2FilterValues(r.Form, "instance-id")
		}
		instances, nextToken, err := s.ec2.DescribeClassicLinkInstances(
			instanceIDs,
			parseEC2FilterValues(r.Form, "vpc-id"),
			parseEC2FilterValues(r.Form, "group-id"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeClassicLinkInstancesResponse{
			XMLName:   xml.Name{Local: "DescribeClassicLinkInstancesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			InstancesSet: ec2ClassicLinkInstanceSet{
				Items: ec2ClassicLinkInstanceItems(instances),
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

func ec2VpcClassicLinkItems(in []ec2svc.VpcClassicLink) []ec2VpcClassicLinkItem {
	out := make([]ec2VpcClassicLinkItem, 0, len(in))
	for _, vpc := range in {
		tags := make([]ec2TagItem, 0, len(vpc.Tags))
		for key, value := range vpc.Tags {
			tags = append(tags, ec2TagItem{Key: key, Value: value})
		}
		sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

		out = append(out, ec2VpcClassicLinkItem{
			ClassicLinkEnabled: vpc.ClassicLinkEnabled,
			TagSet:             ec2TagSet{Items: tags},
			VpcID:              vpc.VpcID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].VpcID < out[j].VpcID })
	return out
}

func ec2ClassicLinkDnsSupportItems(in []ec2svc.VpcClassicLinkDnsSupport) []ec2ClassicLinkDnsSupportItem {
	out := make([]ec2ClassicLinkDnsSupportItem, 0, len(in))
	for _, vpc := range in {
		out = append(out, ec2ClassicLinkDnsSupportItem{
			ClassicLinkDnsSupported: vpc.ClassicLinkDnsSupported,
			VpcID:                   vpc.VpcID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].VpcID < out[j].VpcID })
	return out
}

func ec2ClassicLinkInstanceItems(in []ec2svc.ClassicLinkInstance) []ec2ClassicLinkInstanceItem {
	out := make([]ec2ClassicLinkInstanceItem, 0, len(in))
	for _, instance := range in {
		groupItems := make([]ec2GroupSetItem, 0, len(instance.GroupIDs))
		for _, groupID := range instance.GroupIDs {
			groupItems = append(groupItems, ec2GroupSetItem{GroupID: groupID})
		}

		tagItems := make([]ec2TagItem, 0, len(instance.Tags))
		for key, value := range instance.Tags {
			tagItems = append(tagItems, ec2TagItem{Key: key, Value: value})
		}
		sort.Slice(tagItems, func(i, j int) bool { return tagItems[i].Key < tagItems[j].Key })

		out = append(out, ec2ClassicLinkInstanceItem{
			GroupSet:   ec2GroupSet{Items: groupItems},
			InstanceID: instance.InstanceID,
			TagSet:     ec2TagSet{Items: tagItems},
			VpcID:      instance.VpcID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].InstanceID < out[j].InstanceID })
	return out
}

type ec2DescribeVpcClassicLinkResponse struct {
	XMLName   xml.Name             `xml:"DescribeVpcClassicLinkResponse"`
	Xmlns     string               `xml:"xmlns,attr"`
	RequestID string               `xml:"requestId"`
	VpcSet    ec2VpcClassicLinkSet `xml:"vpcSet"`
}

type ec2VpcClassicLinkSet struct {
	Items []ec2VpcClassicLinkItem `xml:"item"`
}

type ec2VpcClassicLinkItem struct {
	ClassicLinkEnabled bool      `xml:"classicLinkEnabled"`
	TagSet             ec2TagSet `xml:"tagSet,omitempty"`
	VpcID              string    `xml:"vpcId,omitempty"`
}

type ec2DescribeVpcClassicLinkDnsSupportResponse struct {
	XMLName   xml.Name                    `xml:"DescribeVpcClassicLinkDnsSupportResponse"`
	Xmlns     string                      `xml:"xmlns,attr"`
	RequestID string                      `xml:"requestId"`
	NextToken string                      `xml:"nextToken,omitempty"`
	Vpcs      ec2ClassicLinkDnsSupportSet `xml:"vpcs"`
}

type ec2ClassicLinkDnsSupportSet struct {
	Items []ec2ClassicLinkDnsSupportItem `xml:"item"`
}

type ec2ClassicLinkDnsSupportItem struct {
	ClassicLinkDnsSupported bool   `xml:"classicLinkDnsSupported"`
	VpcID                   string `xml:"vpcId,omitempty"`
}

type ec2DescribeClassicLinkInstancesResponse struct {
	XMLName      xml.Name                  `xml:"DescribeClassicLinkInstancesResponse"`
	Xmlns        string                    `xml:"xmlns,attr"`
	RequestID    string                    `xml:"requestId"`
	NextToken    string                    `xml:"nextToken,omitempty"`
	InstancesSet ec2ClassicLinkInstanceSet `xml:"instancesSet"`
}

type ec2ClassicLinkInstanceSet struct {
	Items []ec2ClassicLinkInstanceItem `xml:"item"`
}

type ec2ClassicLinkInstanceItem struct {
	GroupSet   ec2GroupSet `xml:"groupSet,omitempty"`
	InstanceID string      `xml:"instanceId,omitempty"`
	TagSet     ec2TagSet   `xml:"tagSet,omitempty"`
	VpcID      string      `xml:"vpcId,omitempty"`
}
