package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage88Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CancelBundleTask":
		task, err := s.ec2.CancelBundleTask(strings.TrimSpace(r.Form.Get("BundleId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage88CancelBundleTaskResponse{
			XMLName:            xml.Name{Local: "CancelBundleTaskResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			BundleInstanceTask: ec2Stage87BundleTaskItemFrom(task),
		})
		return true
	default:
		return false
	}
}

type ec2Stage88CancelBundleTaskResponse struct {
	XMLName            xml.Name                 `xml:"CancelBundleTaskResponse"`
	Xmlns              string                   `xml:"xmlns,attr"`
	RequestID          string                   `xml:"requestId"`
	BundleInstanceTask ec2Stage87BundleTaskItem `xml:"bundleInstanceTask"`
}
