package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage98Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ConfirmProductInstance":
		result, err := s.ec2.ConfirmProductInstance(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("ProductCode")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		resp := ec2Stage98ConfirmProductInstanceResponse{
			XMLName:   xml.Name{Local: "ConfirmProductInstanceResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    result.Return,
		}
		if result.OwnerID != "" {
			resp.OwnerID = result.OwnerID
		}
		respondEC2XML(w, resp)
		return true
	default:
		return false
	}
}

type ec2Stage98ConfirmProductInstanceResponse struct {
	XMLName   xml.Name `xml:"ConfirmProductInstanceResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	OwnerID   string   `xml:"ownerId,omitempty"`
	Return    bool     `xml:"return"`
}
