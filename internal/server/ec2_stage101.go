package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage101Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CopySnapshot":
		completionDurationMinutes, ok := parseEC2OptionalInt32(r.Form.Get("CompletionDurationMinutes"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		encrypted, hasEncrypted, ok := ec2OptionalBoolFromForm(r.Form, "Encrypted")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasEncrypted {
			encrypted = nil
		}

		snapshot, err := s.ec2.CopySnapshot(
			strings.TrimSpace(r.Form.Get("SourceSnapshotId")),
			strings.TrimSpace(r.Form.Get("SourceRegion")),
			completionDurationMinutes,
			parseEC2OptionalString(r.Form.Get("Description")),
			encrypted,
			parseEC2OptionalString(r.Form.Get("KmsKeyId")),
			parseEC2TagSpecificationsForResource(r.Form, "snapshot"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		resp := ec2Stage101CopySnapshotResponse{
			XMLName:    xml.Name{Local: "CopySnapshotResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			SnapshotID: snapshot.ID,
		}
		if len(snapshot.Tags) > 0 {
			resp.TagSet = &ec2TagSet{Items: ec2TagItemsFromMap(snapshot.Tags)}
		}
		respondEC2XML(w, resp)
		return true
	default:
		return false
	}
}

type ec2Stage101CopySnapshotResponse struct {
	XMLName    xml.Name   `xml:"CopySnapshotResponse"`
	Xmlns      string     `xml:"xmlns,attr"`
	RequestID  string     `xml:"requestId"`
	SnapshotID string     `xml:"snapshotId,omitempty"`
	TagSet     *ec2TagSet `xml:"tagSet,omitempty"`
}
