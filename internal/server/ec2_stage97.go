package server

import (
	"encoding/xml"
	"net/http"
)

func (s *Server) handleEC2Stage97Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CancelSpotInstanceRequests":
		cancelled, err := s.ec2.CancelSpotInstanceRequests(
			parseEC2MembersWithAliases(r.Form, "SpotInstanceRequestId", "SpotInstanceRequestIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		items := make([]ec2Stage97CancelledSpotInstanceRequestItem, 0, len(cancelled))
		for _, req := range cancelled {
			items = append(items, ec2Stage97CancelledSpotInstanceRequestItem{
				SpotInstanceRequestID: req.SpotInstanceRequestID,
				State:                 req.State,
			})
		}

		respondEC2XML(w, ec2Stage97CancelSpotInstanceRequestsResponse{
			XMLName:                    xml.Name{Local: "CancelSpotInstanceRequestsResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			CancelledSpotInstanceItems: ec2Stage97CancelledSpotInstanceRequestSet{Items: items},
		})
		return true
	default:
		return false
	}
}

type ec2Stage97CancelSpotInstanceRequestsResponse struct {
	XMLName                    xml.Name                                  `xml:"CancelSpotInstanceRequestsResponse"`
	Xmlns                      string                                    `xml:"xmlns,attr"`
	RequestID                  string                                    `xml:"requestId"`
	CancelledSpotInstanceItems ec2Stage97CancelledSpotInstanceRequestSet `xml:"spotInstanceRequestSet"`
}

type ec2Stage97CancelledSpotInstanceRequestSet struct {
	Items []ec2Stage97CancelledSpotInstanceRequestItem `xml:"item"`
}

type ec2Stage97CancelledSpotInstanceRequestItem struct {
	SpotInstanceRequestID string `xml:"spotInstanceRequestId,omitempty"`
	State                 string `xml:"state,omitempty"`
}
