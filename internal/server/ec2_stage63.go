package server

import (
	"encoding/xml"
	"net/http"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage63Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeVpcEndpointServices":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		serviceDetails, serviceNames, nextToken, err := s.ec2.DescribeVpcEndpointServices(
			parseEC2MembersWithAliases(r.Form, "ServiceName", "ServiceNames"),
			parseEC2MembersWithAliases(r.Form, "ServiceRegion", "ServiceRegions"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVpcEndpointServicesResponse{
			XMLName:   xml.Name{Local: "DescribeVpcEndpointServicesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ServiceNameSet: ec2Stage63StringSet{
				Items: append([]string(nil), serviceNames...),
			},
			ServiceDetailSet: ec2VpcEndpointServiceDetailSet{
				Items: ec2VpcEndpointServiceDetailItems(serviceDetails),
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

func ec2VpcEndpointServiceDetailItems(in []ec2svc.VpcEndpointServiceDetail) []ec2VpcEndpointServiceDetailItem {
	out := make([]ec2VpcEndpointServiceDetailItem, 0, len(in))
	for _, detail := range in {
		out = append(out, ec2VpcEndpointServiceDetailItem{
			AcceptanceRequired:         &detail.AcceptanceRequired,
			Owner:                      detail.Owner,
			PayerResponsibility:        detail.PayerResponsibility,
			PrivateDNSName:             detail.PrivateDNSName,
			ServiceID:                  detail.ServiceID,
			ServiceName:                detail.ServiceName,
			ServiceRegion:              detail.ServiceRegion,
			ServiceType:                ec2ServiceTypeDetailSet{Items: ec2ServiceTypeItems(detail.ServiceTypes)},
			SupportedIPAddressTypes:    ec2Stage55StringSet{Items: append([]string(nil), detail.SupportedIPAddressTypes...)},
			VpcEndpointPolicySupported: &detail.VpcEndpointPolicySupported,
		})
	}
	return out
}

func ec2ServiceTypeItems(in []string) []ec2ServiceTypeDetailItem {
	out := make([]ec2ServiceTypeDetailItem, 0, len(in))
	for _, serviceType := range in {
		out = append(out, ec2ServiceTypeDetailItem{ServiceType: serviceType})
	}
	return out
}

type ec2DescribeVpcEndpointServicesResponse struct {
	XMLName          xml.Name                       `xml:"DescribeVpcEndpointServicesResponse"`
	Xmlns            string                         `xml:"xmlns,attr"`
	RequestID        string                         `xml:"requestId"`
	NextToken        string                         `xml:"nextToken,omitempty"`
	ServiceNameSet   ec2Stage63StringSet            `xml:"serviceNameSet"`
	ServiceDetailSet ec2VpcEndpointServiceDetailSet `xml:"serviceDetailSet"`
}

type ec2Stage63StringSet struct {
	Items []string `xml:"item"`
}

type ec2VpcEndpointServiceDetailSet struct {
	Items []ec2VpcEndpointServiceDetailItem `xml:"item"`
}

type ec2VpcEndpointServiceDetailItem struct {
	AcceptanceRequired         *bool                   `xml:"acceptanceRequired,omitempty"`
	Owner                      string                  `xml:"owner,omitempty"`
	PayerResponsibility        string                  `xml:"payerResponsibility,omitempty"`
	PrivateDNSName             string                  `xml:"privateDnsName,omitempty"`
	ServiceID                  string                  `xml:"serviceId,omitempty"`
	ServiceName                string                  `xml:"serviceName,omitempty"`
	ServiceRegion              string                  `xml:"serviceRegion,omitempty"`
	ServiceType                ec2ServiceTypeDetailSet `xml:"serviceType"`
	SupportedIPAddressTypes    ec2Stage55StringSet     `xml:"supportedIpAddressTypeSet"`
	VpcEndpointPolicySupported *bool                   `xml:"vpcEndpointPolicySupported,omitempty"`
}
