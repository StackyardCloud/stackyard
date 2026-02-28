package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage44Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyVolume":
		size, ok := parseEC2OptionalInt32(r.Form.Get("Size"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		iops, ok := parseEC2OptionalInt32(r.Form.Get("Iops"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		throughput, ok := parseEC2OptionalInt32(r.Form.Get("Throughput"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		multiAttachEnabled, hasMultiAttachEnabled, ok := ec2OptionalBoolFromForm(r.Form, "MultiAttachEnabled")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasMultiAttachEnabled {
			multiAttachEnabled = nil
		}

		modification, err := s.ec2.ModifyVolume(
			strings.TrimSpace(r.Form.Get("VolumeId")),
			size,
			iops,
			throughput,
			parseEC2OptionalString(r.Form.Get("VolumeType")),
			multiAttachEnabled,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2ModifyVolumeResponse{
			XMLName:            xml.Name{Local: "ModifyVolumeResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			VolumeModification: ec2VolumeModificationItemFrom(modification),
		})
		return true
	case "ModifyVolumeAttribute":
		autoEnableIO, hasAutoEnableIO, ok := ec2OptionalBoolFromForm(r.Form, "AutoEnableIO.Value")
		if !ok || !hasAutoEnableIO {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		if err := s.ec2.ModifyVolumeAttribute(strings.TrimSpace(r.Form.Get("VolumeId")), autoEnableIO); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyVolumeAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	default:
		return false
	}
}

func ec2VolumeModificationItemFrom(modification ec2svc.VolumeModification) ec2VolumeModificationItem {
	return ec2VolumeModificationItem{
		VolumeID:                   modification.VolumeID,
		ModificationState:          modification.ModificationState,
		StartTime:                  modification.StartTime.Format(time.RFC3339),
		EndTime:                    modification.EndTime.Format(time.RFC3339),
		Progress:                   modification.Progress,
		StatusMessage:              modification.StatusMessage,
		OriginalSize:               modification.OriginalSize,
		TargetSize:                 modification.TargetSize,
		OriginalVolumeType:         modification.OriginalVolumeType,
		TargetVolumeType:           modification.TargetVolumeType,
		OriginalIops:               modification.OriginalIops,
		TargetIops:                 modification.TargetIops,
		OriginalThroughput:         modification.OriginalThroughput,
		TargetThroughput:           modification.TargetThroughput,
		OriginalMultiAttachEnabled: modification.OriginalMultiAttachEnabled,
		TargetMultiAttachEnabled:   modification.TargetMultiAttachEnabled,
	}
}

type ec2ModifyVolumeResponse struct {
	XMLName            xml.Name                  `xml:"ModifyVolumeResponse"`
	Xmlns              string                    `xml:"xmlns,attr"`
	RequestID          string                    `xml:"requestId"`
	VolumeModification ec2VolumeModificationItem `xml:"volumeModification"`
}

type ec2VolumeModificationItem struct {
	VolumeID                   string `xml:"volumeId"`
	ModificationState          string `xml:"modificationState"`
	StartTime                  string `xml:"startTime,omitempty"`
	EndTime                    string `xml:"endTime,omitempty"`
	Progress                   int64  `xml:"progress,omitempty"`
	StatusMessage              string `xml:"statusMessage,omitempty"`
	OriginalSize               int32  `xml:"originalSize,omitempty"`
	TargetSize                 int32  `xml:"targetSize,omitempty"`
	OriginalVolumeType         string `xml:"originalVolumeType,omitempty"`
	TargetVolumeType           string `xml:"targetVolumeType,omitempty"`
	OriginalIops               int32  `xml:"originalIops,omitempty"`
	TargetIops                 int32  `xml:"targetIops,omitempty"`
	OriginalThroughput         int32  `xml:"originalThroughput,omitempty"`
	TargetThroughput           int32  `xml:"targetThroughput,omitempty"`
	OriginalMultiAttachEnabled bool   `xml:"originalMultiAttachEnabled,omitempty"`
	TargetMultiAttachEnabled   bool   `xml:"targetMultiAttachEnabled,omitempty"`
}
