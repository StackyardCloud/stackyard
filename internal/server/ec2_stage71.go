package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage71Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DisableImageBlockPublicAccess":
		state := s.ec2.DisableImageBlockPublicAccess()
		respondEC2XML(w, ec2DisableImageBlockPublicAccessResponse{
			XMLName:                     xml.Name{Local: "DisableImageBlockPublicAccessResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			ImageBlockPublicAccessState: state.ImageBlockPublicAccessState,
		})
		return true
	case "EnableImageBlockPublicAccess":
		state, err := s.ec2.EnableImageBlockPublicAccess(strings.TrimSpace(r.Form.Get("ImageBlockPublicAccessState")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2EnableImageBlockPublicAccessResponse{
			XMLName:                     xml.Name{Local: "EnableImageBlockPublicAccessResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			ImageBlockPublicAccessState: state.ImageBlockPublicAccessState,
		})
		return true
	case "GetImageBlockPublicAccessState":
		state := s.ec2.GetImageBlockPublicAccessState()
		respondEC2XML(w, ec2GetImageBlockPublicAccessStateResponse{
			XMLName:                     xml.Name{Local: "GetImageBlockPublicAccessStateResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			ImageBlockPublicAccessState: state.ImageBlockPublicAccessState,
			ManagedBy:                   state.ManagedBy,
		})
		return true
	case "DisableImageDeprecation":
		ret, err := s.ec2.DisableImageDeprecation(strings.TrimSpace(r.Form.Get("ImageId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DisableImageDeprecationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "EnableImageDeprecation":
		deprecateAt, err := parseEC2RFC3339Time(r.Form.Get("DeprecateAt"))
		if err != nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		ret, err := s.ec2.EnableImageDeprecation(
			strings.TrimSpace(r.Form.Get("ImageId")),
			deprecateAt,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "EnableImageDeprecationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}

func parseEC2RFC3339Time(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, ec2svc.ErrInvalidParameter
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

type ec2DisableImageBlockPublicAccessResponse struct {
	XMLName                     xml.Name `xml:"DisableImageBlockPublicAccessResponse"`
	Xmlns                       string   `xml:"xmlns,attr"`
	RequestID                   string   `xml:"requestId"`
	ImageBlockPublicAccessState string   `xml:"imageBlockPublicAccessState,omitempty"`
}

type ec2EnableImageBlockPublicAccessResponse struct {
	XMLName                     xml.Name `xml:"EnableImageBlockPublicAccessResponse"`
	Xmlns                       string   `xml:"xmlns,attr"`
	RequestID                   string   `xml:"requestId"`
	ImageBlockPublicAccessState string   `xml:"imageBlockPublicAccessState,omitempty"`
}

type ec2GetImageBlockPublicAccessStateResponse struct {
	XMLName                     xml.Name `xml:"GetImageBlockPublicAccessStateResponse"`
	Xmlns                       string   `xml:"xmlns,attr"`
	RequestID                   string   `xml:"requestId"`
	ImageBlockPublicAccessState string   `xml:"imageBlockPublicAccessState,omitempty"`
	ManagedBy                   string   `xml:"managedBy,omitempty"`
}
