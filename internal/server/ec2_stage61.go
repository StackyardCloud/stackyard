package server

import (
	"encoding/xml"
	"net/http"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage61Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeVpcEndpointServiceConfigurations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		configurations, nextToken, err := s.ec2.DescribeVpcEndpointServiceConfigurations(
			parseEC2MembersWithAliases(r.Form, "ServiceId", "ServiceIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVpcEndpointServiceConfigurationsResponse{
			XMLName:   xml.Name{Local: "DescribeVpcEndpointServiceConfigurationsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ServiceConfigurationSet: ec2ServiceConfigurationSet{
				Items: ec2ServiceConfigurationItems(configurations),
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

func ec2ServiceConfigurationItems(in []ec2svc.VpcEndpointServiceConfiguration) []ec2ServiceConfigurationItem {
	out := make([]ec2ServiceConfigurationItem, 0, len(in))
	for _, cfg := range in {
		out = append(out, ec2ServiceConfigurationItemFrom(cfg))
	}
	return out
}

type ec2DescribeVpcEndpointServiceConfigurationsResponse struct {
	XMLName                 xml.Name                   `xml:"DescribeVpcEndpointServiceConfigurationsResponse"`
	Xmlns                   string                     `xml:"xmlns,attr"`
	RequestID               string                     `xml:"requestId"`
	NextToken               string                     `xml:"nextToken,omitempty"`
	ServiceConfigurationSet ec2ServiceConfigurationSet `xml:"serviceConfigurationSet"`
}

type ec2ServiceConfigurationSet struct {
	Items []ec2ServiceConfigurationItem `xml:"item"`
}
