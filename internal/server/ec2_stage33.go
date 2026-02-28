package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage33Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeTransitGatewayMulticastDomains":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		domains, nextToken, err := s.ec2.DescribeTransitGatewayMulticastDomains(
			parseEC2MembersOrItemList(r.Form, "TransitGatewayMulticastDomainIds"),
			parseEC2FilterValues(r.Form, "state"),
			parseEC2FilterValues(r.Form, "transit-gateway-id"),
			parseEC2FilterValues(r.Form, "transit-gateway-multicast-domain-id"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeTransitGatewayMulticastDomainsResponse{
			XMLName:                        xml.Name{Local: "DescribeTransitGatewayMulticastDomainsResponse"},
			Xmlns:                          ec2Namespace,
			RequestID:                      "stackyard-request",
			TransitGatewayMulticastDomains: ec2TransitGatewayMulticastDomainSet{Items: ec2TransitGatewayMulticastDomainItems(domains)},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "DescribeTransitGatewayPolicyTables":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		tables, nextToken, err := s.ec2.DescribeTransitGatewayPolicyTables(
			parseEC2MembersOrItemList(r.Form, "TransitGatewayPolicyTableIds"),
			parseEC2FilterValues(r.Form, "state"),
			parseEC2FilterValues(r.Form, "transit-gateway-id"),
			parseEC2FilterValues(r.Form, "transit-gateway-policy-table-id"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeTransitGatewayPolicyTablesResponse{
			XMLName:                    xml.Name{Local: "DescribeTransitGatewayPolicyTablesResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			TransitGatewayPolicyTables: ec2TransitGatewayPolicyTableSet{Items: ec2TransitGatewayPolicyTableItems(tables)},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "DescribeTransitGatewayRouteTables":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		tables, nextToken, err := s.ec2.DescribeTransitGatewayRouteTables(
			parseEC2MembersOrItemList(r.Form, "TransitGatewayRouteTableIds"),
			parseEC2BoolFilterValues(parseEC2FilterValues(r.Form, "default-association-route-table")),
			parseEC2BoolFilterValues(parseEC2FilterValues(r.Form, "default-propagation-route-table")),
			parseEC2FilterValues(r.Form, "state"),
			parseEC2FilterValues(r.Form, "transit-gateway-id"),
			parseEC2FilterValues(r.Form, "transit-gateway-route-table-id"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeTransitGatewayRouteTablesResponse{
			XMLName:                   xml.Name{Local: "DescribeTransitGatewayRouteTablesResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			TransitGatewayRouteTables: ec2TransitGatewayRouteTableSet{Items: ec2TransitGatewayRouteTableItems(tables)},
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

func ec2TransitGatewayMulticastDomainItems(in []ec2svc.TransitGatewayMulticastDomain) []ec2TransitGatewayMulticastDomainItem {
	out := make([]ec2TransitGatewayMulticastDomainItem, 0, len(in))
	for _, domain := range in {
		out = append(out, ec2TransitGatewayMulticastDomainItemFrom(domain))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TransitGatewayMulticastDomainID < out[j].TransitGatewayMulticastDomainID
	})
	return out
}

func ec2TransitGatewayPolicyTableItems(in []ec2svc.TransitGatewayPolicyTable) []ec2TransitGatewayPolicyTableItem {
	out := make([]ec2TransitGatewayPolicyTableItem, 0, len(in))
	for _, table := range in {
		out = append(out, ec2TransitGatewayPolicyTableItemFrom(table))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayPolicyTableID < out[j].TransitGatewayPolicyTableID })
	return out
}

func ec2TransitGatewayRouteTableItems(in []ec2svc.TransitGatewayRouteTable) []ec2TransitGatewayRouteTableItem {
	out := make([]ec2TransitGatewayRouteTableItem, 0, len(in))
	for _, table := range in {
		out = append(out, ec2TransitGatewayRouteTableItemFrom(table))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayRouteTableID < out[j].TransitGatewayRouteTableID })
	return out
}

type ec2DescribeTransitGatewayMulticastDomainsResponse struct {
	XMLName                        xml.Name                            `xml:"DescribeTransitGatewayMulticastDomainsResponse"`
	Xmlns                          string                              `xml:"xmlns,attr"`
	RequestID                      string                              `xml:"requestId"`
	NextToken                      string                              `xml:"nextToken,omitempty"`
	TransitGatewayMulticastDomains ec2TransitGatewayMulticastDomainSet `xml:"transitGatewayMulticastDomains"`
}

type ec2TransitGatewayMulticastDomainSet struct {
	Items []ec2TransitGatewayMulticastDomainItem `xml:"item"`
}

type ec2DescribeTransitGatewayPolicyTablesResponse struct {
	XMLName                    xml.Name                        `xml:"DescribeTransitGatewayPolicyTablesResponse"`
	Xmlns                      string                          `xml:"xmlns,attr"`
	RequestID                  string                          `xml:"requestId"`
	NextToken                  string                          `xml:"nextToken,omitempty"`
	TransitGatewayPolicyTables ec2TransitGatewayPolicyTableSet `xml:"transitGatewayPolicyTables"`
}

type ec2TransitGatewayPolicyTableSet struct {
	Items []ec2TransitGatewayPolicyTableItem `xml:"item"`
}

type ec2DescribeTransitGatewayRouteTablesResponse struct {
	XMLName                   xml.Name                       `xml:"DescribeTransitGatewayRouteTablesResponse"`
	Xmlns                     string                         `xml:"xmlns,attr"`
	RequestID                 string                         `xml:"requestId"`
	NextToken                 string                         `xml:"nextToken,omitempty"`
	TransitGatewayRouteTables ec2TransitGatewayRouteTableSet `xml:"transitGatewayRouteTables"`
}

type ec2TransitGatewayRouteTableSet struct {
	Items []ec2TransitGatewayRouteTableItem `xml:"item"`
}

func parseEC2MembersOrItemList(values url.Values, key string) []string {
	out := parseEC2Members(values, key+".")
	if len(out) > 0 {
		return out
	}
	return parseEC2Members(values, key+".Item.")
}
