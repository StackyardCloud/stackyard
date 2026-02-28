package server

import (
	"encoding/xml"
	"net/http"
)

func (s *Server) handleEC2Stage90Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CancelCapacityReservationFleets":
		successful, failed, err := s.ec2.CancelCapacityReservationFleets(
			parseEC2MembersWithAliases(r.Form, "CapacityReservationFleetId", "CapacityReservationFleetIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		successfulItems := make([]ec2Stage90CapacityReservationFleetCancellationStateItem, 0, len(successful))
		for _, state := range successful {
			successfulItems = append(successfulItems, ec2Stage90CapacityReservationFleetCancellationStateItem{
				CapacityReservationFleetID: state.CapacityReservationFleetID,
				CurrentFleetState:          state.CurrentFleetState,
				PreviousFleetState:         state.PreviousFleetState,
			})
		}

		failedItems := make([]ec2Stage90FailedCapacityReservationFleetCancellationResultItem, 0, len(failed))
		for _, item := range failed {
			out := ec2Stage90FailedCapacityReservationFleetCancellationResultItem{
				CapacityReservationFleetID: item.CapacityReservationFleetID,
			}
			if item.CancelCapacityReservationFleetError != nil {
				out.CancelCapacityReservationFleetError = &ec2Stage90CancelCapacityReservationFleetErrorItem{
					Code:    item.CancelCapacityReservationFleetError.Code,
					Message: item.CancelCapacityReservationFleetError.Message,
				}
			}
			failedItems = append(failedItems, out)
		}

		respondEC2XML(w, ec2Stage90CancelCapacityReservationFleetsResponse{
			XMLName:                      xml.Name{Local: "CancelCapacityReservationFleetsResponse"},
			Xmlns:                        ec2Namespace,
			RequestID:                    "stackyard-request",
			FailedFleetCancellations:     ec2Stage90FailedCapacityReservationFleetCancellationResultSet{Items: failedItems},
			SuccessfulFleetCancellations: ec2Stage90CapacityReservationFleetCancellationStateSet{Items: successfulItems},
		})
		return true
	default:
		return false
	}
}

type ec2Stage90CancelCapacityReservationFleetsResponse struct {
	XMLName                      xml.Name                                                      `xml:"CancelCapacityReservationFleetsResponse"`
	Xmlns                        string                                                        `xml:"xmlns,attr"`
	RequestID                    string                                                        `xml:"requestId"`
	FailedFleetCancellations     ec2Stage90FailedCapacityReservationFleetCancellationResultSet `xml:"failedFleetCancellationSet"`
	SuccessfulFleetCancellations ec2Stage90CapacityReservationFleetCancellationStateSet        `xml:"successfulFleetCancellationSet"`
}

type ec2Stage90FailedCapacityReservationFleetCancellationResultSet struct {
	Items []ec2Stage90FailedCapacityReservationFleetCancellationResultItem `xml:"item"`
}

type ec2Stage90FailedCapacityReservationFleetCancellationResultItem struct {
	CancelCapacityReservationFleetError *ec2Stage90CancelCapacityReservationFleetErrorItem `xml:"cancelCapacityReservationFleetError,omitempty"`
	CapacityReservationFleetID          string                                             `xml:"capacityReservationFleetId,omitempty"`
}

type ec2Stage90CancelCapacityReservationFleetErrorItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2Stage90CapacityReservationFleetCancellationStateSet struct {
	Items []ec2Stage90CapacityReservationFleetCancellationStateItem `xml:"item"`
}

type ec2Stage90CapacityReservationFleetCancellationStateItem struct {
	CapacityReservationFleetID string `xml:"capacityReservationFleetId,omitempty"`
	CurrentFleetState          string `xml:"currentFleetState,omitempty"`
	PreviousFleetState         string `xml:"previousFleetState,omitempty"`
}
