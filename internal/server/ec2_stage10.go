package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage10Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "GetConsoleOutput":
		output, err := s.ec2.GetConsoleOutput(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			parseEC2Bool(r.Form.Get("Latest"), false),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2GetConsoleOutputResponse{
			XMLName:    xml.Name{Local: "GetConsoleOutputResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			InstanceID: output.InstanceID,
			Output:     output.Output,
			Timestamp:  output.Timestamp.Format(timeRFC3339UTC),
		})
		return true
	case "GetConsoleScreenshot":
		screenshot, err := s.ec2.GetConsoleScreenshot(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			parseEC2Bool(r.Form.Get("WakeUp"), false),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2GetConsoleScreenshotResponse{
			XMLName:    xml.Name{Local: "GetConsoleScreenshotResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			InstanceID: screenshot.InstanceID,
			ImageData:  screenshot.ImageData,
		})
		return true
	case "GetPasswordData":
		passwordData, err := s.ec2.GetPasswordData(strings.TrimSpace(r.Form.Get("InstanceId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2GetPasswordDataResponse{
			XMLName:      xml.Name{Local: "GetPasswordDataResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			InstanceID:   passwordData.InstanceID,
			PasswordData: passwordData.PasswordData,
			Timestamp:    passwordData.Timestamp.Format(timeRFC3339UTC),
		})
		return true
	default:
		return false
	}
}

type ec2GetConsoleOutputResponse struct {
	XMLName    xml.Name
	Xmlns      string `xml:"xmlns,attr"`
	RequestID  string `xml:"requestId"`
	InstanceID string `xml:"instanceId,omitempty"`
	Output     string `xml:"output,omitempty"`
	Timestamp  string `xml:"timestamp,omitempty"`
}

type ec2GetConsoleScreenshotResponse struct {
	XMLName    xml.Name
	Xmlns      string `xml:"xmlns,attr"`
	RequestID  string `xml:"requestId"`
	InstanceID string `xml:"instanceId,omitempty"`
	ImageData  string `xml:"imageData,omitempty"`
}

type ec2GetPasswordDataResponse struct {
	XMLName      xml.Name
	Xmlns        string `xml:"xmlns,attr"`
	RequestID    string `xml:"requestId"`
	InstanceID   string `xml:"instanceId,omitempty"`
	PasswordData string `xml:"passwordData,omitempty"`
	Timestamp    string `xml:"timestamp,omitempty"`
}
