package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage67Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateVpcBlockPublicAccessExclusion":
		exclusion, err := s.ec2.CreateVpcBlockPublicAccessExclusion(
			strings.TrimSpace(r.Form.Get("InternetGatewayExclusionMode")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			strings.TrimSpace(r.Form.Get("SubnetId")),
			parseEC2TagSpecificationsForResource(r.Form, "vpc-block-public-access-exclusion"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateVpcBlockPublicAccessExclusionResponse{
			XMLName:                       xml.Name{Local: "CreateVpcBlockPublicAccessExclusionResponse"},
			Xmlns:                         ec2Namespace,
			RequestID:                     "stackyard-request",
			VpcBlockPublicAccessExclusion: ec2VpcBlockPublicAccessExclusionItemFrom(exclusion),
		})
		return true
	case "DeleteVpcBlockPublicAccessExclusion":
		exclusion, err := s.ec2.DeleteVpcBlockPublicAccessExclusion(strings.TrimSpace(r.Form.Get("ExclusionId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteVpcBlockPublicAccessExclusionResponse{
			XMLName:                       xml.Name{Local: "DeleteVpcBlockPublicAccessExclusionResponse"},
			Xmlns:                         ec2Namespace,
			RequestID:                     "stackyard-request",
			VpcBlockPublicAccessExclusion: ec2VpcBlockPublicAccessExclusionItemFrom(exclusion),
		})
		return true
	case "DescribeVpcBlockPublicAccessExclusions":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		exclusions, nextToken, err := s.ec2.DescribeVpcBlockPublicAccessExclusions(
			parseEC2Members(r.Form, "ExclusionId."),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVpcBlockPublicAccessExclusionsResponse{
			XMLName:   xml.Name{Local: "DescribeVpcBlockPublicAccessExclusionsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpcBlockPublicAccessExclusionSet: ec2VpcBlockPublicAccessExclusionSet{
				Items: ec2VpcBlockPublicAccessExclusionItemsFrom(exclusions),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "DescribeVpcBlockPublicAccessOptions":
		options := s.ec2.DescribeVpcBlockPublicAccessOptions()
		respondEC2XML(w, ec2DescribeVpcBlockPublicAccessOptionsResponse{
			XMLName:                     xml.Name{Local: "DescribeVpcBlockPublicAccessOptionsResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			VpcBlockPublicAccessOptions: ec2VpcBlockPublicAccessOptionsItemFrom(options),
		})
		return true
	default:
		return false
	}
}

func ec2VpcBlockPublicAccessExclusionItemsFrom(in []ec2svc.VpcBlockPublicAccessExclusion) []ec2VpcBlockPublicAccessExclusionItem {
	out := make([]ec2VpcBlockPublicAccessExclusionItem, 0, len(in))
	for _, exclusion := range in {
		out = append(out, ec2VpcBlockPublicAccessExclusionItemFrom(exclusion))
	}
	return out
}

type ec2CreateVpcBlockPublicAccessExclusionResponse struct {
	XMLName                       xml.Name                             `xml:"CreateVpcBlockPublicAccessExclusionResponse"`
	Xmlns                         string                               `xml:"xmlns,attr"`
	RequestID                     string                               `xml:"requestId"`
	VpcBlockPublicAccessExclusion ec2VpcBlockPublicAccessExclusionItem `xml:"vpcBlockPublicAccessExclusion"`
}

type ec2DeleteVpcBlockPublicAccessExclusionResponse struct {
	XMLName                       xml.Name                             `xml:"DeleteVpcBlockPublicAccessExclusionResponse"`
	Xmlns                         string                               `xml:"xmlns,attr"`
	RequestID                     string                               `xml:"requestId"`
	VpcBlockPublicAccessExclusion ec2VpcBlockPublicAccessExclusionItem `xml:"vpcBlockPublicAccessExclusion"`
}

type ec2DescribeVpcBlockPublicAccessExclusionsResponse struct {
	XMLName                          xml.Name                            `xml:"DescribeVpcBlockPublicAccessExclusionsResponse"`
	Xmlns                            string                              `xml:"xmlns,attr"`
	RequestID                        string                              `xml:"requestId"`
	NextToken                        string                              `xml:"nextToken,omitempty"`
	VpcBlockPublicAccessExclusionSet ec2VpcBlockPublicAccessExclusionSet `xml:"vpcBlockPublicAccessExclusionSet"`
}

type ec2VpcBlockPublicAccessExclusionSet struct {
	Items []ec2VpcBlockPublicAccessExclusionItem `xml:"item"`
}

type ec2DescribeVpcBlockPublicAccessOptionsResponse struct {
	XMLName                     xml.Name                           `xml:"DescribeVpcBlockPublicAccessOptionsResponse"`
	Xmlns                       string                             `xml:"xmlns,attr"`
	RequestID                   string                             `xml:"requestId"`
	VpcBlockPublicAccessOptions ec2VpcBlockPublicAccessOptionsItem `xml:"vpcBlockPublicAccessOptions"`
}
