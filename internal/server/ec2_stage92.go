package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage92Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CancelExportTask":
		if err := s.ec2.CancelExportTask(strings.TrimSpace(r.Form.Get("ExportTaskId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage92CancelExportTaskResponse{
			XMLName:   xml.Name{Local: "CancelExportTaskResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	default:
		return false
	}
}

type ec2Stage92CancelExportTaskResponse struct {
	XMLName   xml.Name `xml:"CancelExportTaskResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
}
