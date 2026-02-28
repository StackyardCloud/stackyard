package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage99Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CopyFpgaImage":
		fpgaImageID, err := s.ec2.CopyFpgaImage(
			strings.TrimSpace(r.Form.Get("SourceFpgaImageId")),
			strings.TrimSpace(r.Form.Get("SourceRegion")),
			parseEC2OptionalString(r.Form.Get("Name")),
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2Stage99CopyFpgaImageResponse{
			XMLName:     xml.Name{Local: "CopyFpgaImageResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			FpgaImageID: fpgaImageID,
		})
		return true
	default:
		return false
	}
}

type ec2Stage99CopyFpgaImageResponse struct {
	XMLName     xml.Name `xml:"CopyFpgaImageResponse"`
	Xmlns       string   `xml:"xmlns,attr"`
	RequestID   string   `xml:"requestId"`
	FpgaImageID string   `xml:"fpgaImageId,omitempty"`
}
