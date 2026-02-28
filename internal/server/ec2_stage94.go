package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage94Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CancelImportTask":
		out, err := s.ec2.CancelImportTask(
			strings.TrimSpace(r.Form.Get("ImportTaskId")),
			parseEC2OptionalString(r.Form.Get("CancelReason")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage94CancelImportTaskResponse{
			XMLName:       xml.Name{Local: "CancelImportTaskResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			ImportTaskID:  out.ImportTaskID,
			PreviousState: out.PreviousState,
			State:         out.State,
		})
		return true
	default:
		return false
	}
}

type ec2Stage94CancelImportTaskResponse struct {
	XMLName       xml.Name `xml:"CancelImportTaskResponse"`
	Xmlns         string   `xml:"xmlns,attr"`
	RequestID     string   `xml:"requestId"`
	ImportTaskID  string   `xml:"importTaskId,omitempty"`
	PreviousState string   `xml:"previousState,omitempty"`
	State         string   `xml:"state,omitempty"`
}
