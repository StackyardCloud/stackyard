package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage100Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CopyImage":
		copyImageTags, hasCopyImageTags, ok := ec2OptionalBoolFromForm(r.Form, "CopyImageTags")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasCopyImageTags {
			copyImageTags = nil
		}

		encrypted, hasEncrypted, ok := ec2OptionalBoolFromForm(r.Form, "Encrypted")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasEncrypted {
			encrypted = nil
		}

		imageID, err := s.ec2.CopyImage(
			strings.TrimSpace(r.Form.Get("Name")),
			strings.TrimSpace(r.Form.Get("SourceImageId")),
			strings.TrimSpace(r.Form.Get("SourceRegion")),
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			copyImageTags,
			encrypted,
			parseEC2OptionalString(r.Form.Get("KmsKeyId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2Stage100CopyImageResponse{
			XMLName:   xml.Name{Local: "CopyImageResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ImageID:   imageID,
		})
		return true
	default:
		return false
	}
}

type ec2Stage100CopyImageResponse struct {
	XMLName   xml.Name `xml:"CopyImageResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	ImageID   string   `xml:"imageId,omitempty"`
}
