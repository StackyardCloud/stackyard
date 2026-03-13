package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage72Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DisableImageDeregistrationProtection":
		ret, err := s.ec2.DisableImageDeregistrationProtection(strings.TrimSpace(r.Form.Get("ImageId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ImageDeregistrationProtectionResponse{
			XMLName:   xml.Name{Local: "DisableImageDeregistrationProtectionResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "EnableImageDeregistrationProtection":
		withCooldown, hasWithCooldown, ok := ec2OptionalBoolFromForm(r.Form, "WithCooldown")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasWithCooldown {
			withCooldown = nil
		}
		ret, err := s.ec2.EnableImageDeregistrationProtection(strings.TrimSpace(r.Form.Get("ImageId")), withCooldown)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ImageDeregistrationProtectionResponse{
			XMLName:   xml.Name{Local: "EnableImageDeregistrationProtectionResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "DisableSnapshotBlockPublicAccess":
		state := s.ec2.DisableSnapshotBlockPublicAccess()
		respondEC2XML(w, ec2SnapshotBlockPublicAccessStateResponse{
			XMLName:   xml.Name{Local: "DisableSnapshotBlockPublicAccessResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			State:     state.State,
		})
		return true
	case "EnableSnapshotBlockPublicAccess":
		state, err := s.ec2.EnableSnapshotBlockPublicAccess(strings.TrimSpace(r.Form.Get("State")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SnapshotBlockPublicAccessStateResponse{
			XMLName:   xml.Name{Local: "EnableSnapshotBlockPublicAccessResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			State:     state.State,
		})
		return true
	case "GetSnapshotBlockPublicAccessState":
		state := s.ec2.GetSnapshotBlockPublicAccessState()
		respondEC2XML(w, ec2GetSnapshotBlockPublicAccessStateResponse{
			XMLName:   xml.Name{Local: "GetSnapshotBlockPublicAccessStateResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			State:     state.State,
		})
		return true
	default:
		return false
	}
}

type ec2ImageDeregistrationProtectionResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	Return    string `xml:"return,omitempty"`
}

type ec2SnapshotBlockPublicAccessStateResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
	State     string `xml:"state,omitempty"`
}

type ec2GetSnapshotBlockPublicAccessStateResponse struct {
	XMLName   xml.Name `xml:"GetSnapshotBlockPublicAccessStateResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	State     string   `xml:"state,omitempty"`
}
