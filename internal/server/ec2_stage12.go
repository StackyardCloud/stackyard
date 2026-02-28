package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage12Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeIdFormat":
		statuses, err := s.ec2.DescribeIDFormat(strings.TrimSpace(r.Form.Get("Resource")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DescribeIDFormatResponse{
			XMLName:   xml.Name{Local: "DescribeIdFormatResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			StatusSet: ec2IDFormatStatusSet{Items: ec2IDFormatStatusItems(statuses)},
		})
		return true
	case "DescribeIdentityIdFormat":
		statuses, err := s.ec2.DescribeIdentityIDFormat(
			strings.TrimSpace(r.Form.Get("PrincipalArn")),
			strings.TrimSpace(r.Form.Get("Resource")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DescribeIDFormatResponse{
			XMLName:   xml.Name{Local: "DescribeIdentityIdFormatResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			StatusSet: ec2IDFormatStatusSet{Items: ec2IDFormatStatusItems(statuses)},
		})
		return true
	case "DescribePrincipalIdFormat":
		resources := parseEC2Members(r.Form, "Resource.")
		principals := s.ec2.DescribePrincipalIDFormat(resources)
		respondEC2XML(w, ec2DescribePrincipalIDFormatResponse{
			XMLName:      xml.Name{Local: "DescribePrincipalIdFormatResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			PrincipalSet: ec2PrincipalIDFormatSet{Items: ec2PrincipalIDFormatItems(principals)},
		})
		return true
	case "ModifyIdFormat":
		useLongIDs, ok := parseEC2OptionalBool(r.Form.Get("UseLongIds"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if err := s.ec2.ModifyIDFormat(strings.TrimSpace(r.Form.Get("Resource")), useLongIDs); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyIdFormatResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	default:
		return false
	}
}

func ec2IDFormatStatusItems(in []ec2svc.IDFormatStatus) []ec2IDFormatStatusItem {
	out := make([]ec2IDFormatStatusItem, 0, len(in))
	for _, status := range in {
		out = append(out, ec2IDFormatStatusItem{
			Resource:   status.Resource,
			UseLongIDs: status.UseLongIDs,
		})
	}
	return out
}

func ec2PrincipalIDFormatItems(in []ec2svc.PrincipalIDFormat) []ec2PrincipalIDFormatItem {
	out := make([]ec2PrincipalIDFormatItem, 0, len(in))
	for _, principal := range in {
		out = append(out, ec2PrincipalIDFormatItem{
			Arn:       principal.ARN,
			StatusSet: ec2IDFormatStatusSet{Items: ec2IDFormatStatusItems(principal.Statuses)},
		})
	}
	return out
}

type ec2DescribeIDFormatResponse struct {
	XMLName   xml.Name
	Xmlns     string               `xml:"xmlns,attr"`
	RequestID string               `xml:"requestId"`
	StatusSet ec2IDFormatStatusSet `xml:"statusSet"`
}

type ec2DescribePrincipalIDFormatResponse struct {
	XMLName      xml.Name                `xml:"DescribePrincipalIdFormatResponse"`
	Xmlns        string                  `xml:"xmlns,attr"`
	RequestID    string                  `xml:"requestId"`
	NextToken    string                  `xml:"nextToken,omitempty"`
	PrincipalSet ec2PrincipalIDFormatSet `xml:"principalSet"`
}

type ec2IDFormatStatusSet struct {
	Items []ec2IDFormatStatusItem `xml:"item"`
}

type ec2IDFormatStatusItem struct {
	Deadline   string `xml:"deadline,omitempty"`
	Resource   string `xml:"resource,omitempty"`
	UseLongIDs bool   `xml:"useLongIds"`
}

type ec2PrincipalIDFormatSet struct {
	Items []ec2PrincipalIDFormatItem `xml:"item"`
}

type ec2PrincipalIDFormatItem struct {
	Arn       string               `xml:"arn,omitempty"`
	StatusSet ec2IDFormatStatusSet `xml:"statusSet"`
}
