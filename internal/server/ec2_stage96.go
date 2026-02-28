package server

import (
	"encoding/xml"
	"net/http"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage96Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CancelSpotFleetRequests":
		terminateInstances, hasTerminateInstances, ok := ec2OptionalBoolFromForm(r.Form, "TerminateInstances")
		if !ok || !hasTerminateInstances || terminateInstances == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		successful, unsuccessful, err := s.ec2.CancelSpotFleetRequests(
			parseEC2MembersWithAliases(r.Form, "SpotFleetRequestId", "SpotFleetRequestIds"),
			*terminateInstances,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		successfulItems := make([]ec2Stage96CancelSpotFleetRequestsSuccessItem, 0, len(successful))
		for _, item := range successful {
			successfulItems = append(successfulItems, ec2Stage96CancelSpotFleetRequestsSuccessItem{
				SpotFleetRequestID:            item.SpotFleetRequestID,
				CurrentSpotFleetRequestState:  item.CurrentSpotFleetRequestState,
				PreviousSpotFleetRequestState: item.PreviousSpotFleetRequestState,
			})
		}

		unsuccessfulItems := make([]ec2Stage96CancelSpotFleetRequestsErrorItem, 0, len(unsuccessful))
		for _, item := range unsuccessful {
			out := ec2Stage96CancelSpotFleetRequestsErrorItem{
				SpotFleetRequestID: item.SpotFleetRequestID,
			}
			if item.Error != nil {
				out.Error = &ec2Stage96CancelSpotFleetRequestsError{
					Code:    item.Error.Code,
					Message: item.Error.Message,
				}
			}
			unsuccessfulItems = append(unsuccessfulItems, out)
		}

		respondEC2XML(w, ec2Stage96CancelSpotFleetRequestsResponse{
			XMLName:                   xml.Name{Local: "CancelSpotFleetRequestsResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			SuccessfulFleetRequests:   ec2Stage96CancelSpotFleetRequestsSuccessSet{Items: successfulItems},
			UnsuccessfulFleetRequests: ec2Stage96CancelSpotFleetRequestsErrorSet{Items: unsuccessfulItems},
		})
		return true
	default:
		return false
	}
}

type ec2Stage96CancelSpotFleetRequestsResponse struct {
	XMLName                   xml.Name                                    `xml:"CancelSpotFleetRequestsResponse"`
	Xmlns                     string                                      `xml:"xmlns,attr"`
	RequestID                 string                                      `xml:"requestId"`
	SuccessfulFleetRequests   ec2Stage96CancelSpotFleetRequestsSuccessSet `xml:"successfulFleetRequestSet"`
	UnsuccessfulFleetRequests ec2Stage96CancelSpotFleetRequestsErrorSet   `xml:"unsuccessfulFleetRequestSet"`
}

type ec2Stage96CancelSpotFleetRequestsSuccessSet struct {
	Items []ec2Stage96CancelSpotFleetRequestsSuccessItem `xml:"item"`
}

type ec2Stage96CancelSpotFleetRequestsSuccessItem struct {
	CurrentSpotFleetRequestState  string `xml:"currentSpotFleetRequestState,omitempty"`
	PreviousSpotFleetRequestState string `xml:"previousSpotFleetRequestState,omitempty"`
	SpotFleetRequestID            string `xml:"spotFleetRequestId,omitempty"`
}

type ec2Stage96CancelSpotFleetRequestsErrorSet struct {
	Items []ec2Stage96CancelSpotFleetRequestsErrorItem `xml:"item"`
}

type ec2Stage96CancelSpotFleetRequestsErrorItem struct {
	Error              *ec2Stage96CancelSpotFleetRequestsError `xml:"error,omitempty"`
	SpotFleetRequestID string                                  `xml:"spotFleetRequestId,omitempty"`
}

type ec2Stage96CancelSpotFleetRequestsError struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}
