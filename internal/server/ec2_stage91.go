package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage91Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CancelConversionTask":
		if err := s.ec2.CancelConversionTask(
			strings.TrimSpace(r.Form.Get("ConversionTaskId")),
			parseEC2OptionalString(r.Form.Get("ReasonMessage")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage91CancelConversionTaskResponse{
			XMLName:   xml.Name{Local: "CancelConversionTaskResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	default:
		return false
	}
}

type ec2Stage91CancelConversionTaskResponse struct {
	XMLName   xml.Name `xml:"CancelConversionTaskResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
}
