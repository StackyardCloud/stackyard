package server

import (
	"encoding/xml"
	"net/http"
)

func (s *Server) handleEC2Stage14Action(w http.ResponseWriter, _ *http.Request, action string) bool {
	switch action {
	case "DescribeAggregateIdFormat":
		statuses, aggregated := s.ec2.DescribeAggregateIDFormat()
		respondEC2XML(w, ec2DescribeAggregateIDFormatResponse{
			XMLName:              xml.Name{Local: "DescribeAggregateIdFormatResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			StatusSet:            ec2IDFormatStatusSet{Items: ec2IDFormatStatusItems(statuses)},
			UseLongIDsAggregated: aggregated,
		})
		return true
	default:
		return false
	}
}

type ec2DescribeAggregateIDFormatResponse struct {
	XMLName              xml.Name
	Xmlns                string               `xml:"xmlns,attr"`
	RequestID            string               `xml:"requestId"`
	StatusSet            ec2IDFormatStatusSet `xml:"statusSet"`
	UseLongIDsAggregated bool                 `xml:"useLongIdsAggregated"`
}
