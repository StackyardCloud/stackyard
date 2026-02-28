package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage62Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeVpcEndpointServicePermissions":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		allowedPrincipals, nextToken, err := s.ec2.DescribeVpcEndpointServicePermissions(
			strings.TrimSpace(r.Form.Get("ServiceId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVpcEndpointServicePermissionsResponse{
			XMLName:   xml.Name{Local: "DescribeVpcEndpointServicePermissionsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			AllowedPrincipals: ec2AllowedPrincipalSet{
				Items: ec2AddedPrincipalItems(allowedPrincipals),
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

type ec2DescribeVpcEndpointServicePermissionsResponse struct {
	XMLName           xml.Name               `xml:"DescribeVpcEndpointServicePermissionsResponse"`
	Xmlns             string                 `xml:"xmlns,attr"`
	RequestID         string                 `xml:"requestId"`
	NextToken         string                 `xml:"nextToken,omitempty"`
	AllowedPrincipals ec2AllowedPrincipalSet `xml:"allowedPrincipals"`
}

type ec2AllowedPrincipalSet struct {
	Items []ec2AddedPrincipalItem `xml:"item"`
}
