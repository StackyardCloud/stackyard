package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage73Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DisableImage":
		ret, err := s.ec2.DisableImage(strings.TrimSpace(r.Form.Get("ImageId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DisableImageResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "EnableImage":
		ret, err := s.ec2.EnableImage(strings.TrimSpace(r.Form.Get("ImageId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "EnableImageResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "EnableFastLaunch":
		maxParallelLaunches, ok := parseEC2OptionalInt32(r.Form.Get("MaxParallelLaunches"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		targetResourceCount, ok := parseEC2OptionalInt32(r.Form.Get("SnapshotConfiguration.TargetResourceCount"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		launchTemplate := parseEC2FastLaunchLaunchTemplate(r.Form)
		var snapshotConfiguration *ec2svc.FastLaunchSnapshotConfiguration
		if targetResourceCount != nil {
			snapshotConfiguration = &ec2svc.FastLaunchSnapshotConfiguration{TargetResourceCount: *targetResourceCount}
		}
		out, err := s.ec2.EnableFastLaunch(
			strings.TrimSpace(r.Form.Get("ImageId")),
			launchTemplate,
			maxParallelLaunches,
			parseEC2OptionalString(r.Form.Get("ResourceType")),
			snapshotConfiguration,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2FastLaunchResponse{
			XMLName:               xml.Name{Local: "EnableFastLaunchResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			ImageID:               out.ImageID,
			LaunchTemplate:        ec2FastLaunchLaunchTemplateFrom(out.LaunchTemplate),
			MaxParallelLaunches:   out.MaxParallelLaunches,
			OwnerID:               out.OwnerID,
			ResourceType:          out.ResourceType,
			SnapshotConfiguration: ec2FastLaunchSnapshotConfigurationFrom(out.SnapshotConfiguration),
			State:                 out.State,
			StateTransitionReason: out.StateTransitionReason,
			StateTransitionTime:   out.StateTransitionTime.Format(time.RFC3339),
		})
		return true
	case "DisableFastLaunch":
		out, err := s.ec2.DisableFastLaunch(
			strings.TrimSpace(r.Form.Get("ImageId")),
			parseEC2Bool(r.Form.Get("Force"), false),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2FastLaunchResponse{
			XMLName:               xml.Name{Local: "DisableFastLaunchResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			ImageID:               out.ImageID,
			LaunchTemplate:        ec2FastLaunchLaunchTemplateFrom(out.LaunchTemplate),
			MaxParallelLaunches:   out.MaxParallelLaunches,
			OwnerID:               out.OwnerID,
			ResourceType:          out.ResourceType,
			SnapshotConfiguration: ec2FastLaunchSnapshotConfigurationFrom(out.SnapshotConfiguration),
			State:                 out.State,
			StateTransitionReason: out.StateTransitionReason,
			StateTransitionTime:   out.StateTransitionTime.Format(time.RFC3339),
		})
		return true
	case "EnableFastSnapshotRestores":
		successful, unsuccessful, err := s.ec2.EnableFastSnapshotRestores(
			parseEC2MembersWithAliases(r.Form, "SourceSnapshotId", "SourceSnapshotIds"),
			parseEC2MembersWithAliases(r.Form, "AvailabilityZone", "AvailabilityZones", "AvailabilityZoneId", "AvailabilityZoneIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2FastSnapshotRestoresResponse{
			XMLName:   xml.Name{Local: "EnableFastSnapshotRestoresResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Successful: ec2FastSnapshotRestoreSuccessSet{
				Items: ec2FastSnapshotRestoreSuccessItemsFrom(successful),
			},
			Unsuccessful: ec2FastSnapshotRestoreErrorSet{
				Items: ec2FastSnapshotRestoreErrorItemsFrom(unsuccessful),
			},
		})
		return true
	case "DisableFastSnapshotRestores":
		successful, unsuccessful, err := s.ec2.DisableFastSnapshotRestores(
			parseEC2MembersWithAliases(r.Form, "SourceSnapshotId", "SourceSnapshotIds"),
			parseEC2MembersWithAliases(r.Form, "AvailabilityZone", "AvailabilityZones", "AvailabilityZoneId", "AvailabilityZoneIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2FastSnapshotRestoresResponse{
			XMLName:   xml.Name{Local: "DisableFastSnapshotRestoresResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Successful: ec2FastSnapshotRestoreSuccessSet{
				Items: ec2FastSnapshotRestoreSuccessItemsFrom(successful),
			},
			Unsuccessful: ec2FastSnapshotRestoreErrorSet{
				Items: ec2FastSnapshotRestoreErrorItemsFrom(unsuccessful),
			},
		})
		return true
	default:
		return false
	}
}

func parseEC2FastLaunchLaunchTemplate(values map[string][]string) *ec2svc.FastLaunchLaunchTemplate {
	launchTemplateID := strings.TrimSpace(firstFormValue(values, "LaunchTemplate.LaunchTemplateId"))
	launchTemplateName := strings.TrimSpace(firstFormValue(values, "LaunchTemplate.LaunchTemplateName"))
	version := strings.TrimSpace(firstFormValue(values, "LaunchTemplate.Version"))
	if launchTemplateID == "" && launchTemplateName == "" && version == "" {
		return nil
	}
	return &ec2svc.FastLaunchLaunchTemplate{
		LaunchTemplateID:   launchTemplateID,
		LaunchTemplateName: launchTemplateName,
		Version:            version,
	}
}

func firstFormValue(values map[string][]string, key string) string {
	all := values[key]
	if len(all) == 0 {
		return ""
	}
	return all[0]
}

func ec2FastLaunchLaunchTemplateFrom(in *ec2svc.FastLaunchLaunchTemplate) *ec2FastLaunchLaunchTemplate {
	if in == nil {
		return nil
	}
	return &ec2FastLaunchLaunchTemplate{
		LaunchTemplateID:   in.LaunchTemplateID,
		LaunchTemplateName: in.LaunchTemplateName,
		Version:            in.Version,
	}
}

func ec2FastLaunchSnapshotConfigurationFrom(in *ec2svc.FastLaunchSnapshotConfiguration) *ec2FastLaunchSnapshotConfiguration {
	if in == nil {
		return nil
	}
	return &ec2FastLaunchSnapshotConfiguration{
		TargetResourceCount: in.TargetResourceCount,
	}
}

func ec2FastSnapshotRestoreSuccessItemsFrom(in []ec2svc.FastSnapshotRestoreSuccess) []ec2FastSnapshotRestoreSuccessItem {
	out := make([]ec2FastSnapshotRestoreSuccessItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2FastSnapshotRestoreSuccessItem{
			AvailabilityZone:      item.AvailabilityZone,
			DisabledTime:          ec2TimeString(item.DisabledTime),
			DisablingTime:         ec2TimeString(item.DisablingTime),
			EnabledTime:           ec2TimeString(item.EnabledTime),
			EnablingTime:          ec2TimeString(item.EnablingTime),
			OptimizingTime:        ec2TimeString(item.OptimizingTime),
			OwnerAlias:            item.OwnerAlias,
			OwnerID:               item.OwnerID,
			SnapshotID:            item.SnapshotID,
			State:                 item.State,
			StateTransitionReason: item.StateTransitionReason,
		})
	}
	return out
}

func ec2FastSnapshotRestoreErrorItemsFrom(in []ec2svc.FastSnapshotRestoreError) []ec2FastSnapshotRestoreErrorItem {
	out := make([]ec2FastSnapshotRestoreErrorItem, 0, len(in))
	for _, item := range in {
		errors := make([]ec2FastSnapshotRestoreStateErrorItem, 0, len(item.FastSnapshotRestoreStateErrors))
		for _, stateError := range item.FastSnapshotRestoreStateErrors {
			errors = append(errors, ec2FastSnapshotRestoreStateErrorItem{
				AvailabilityZone: stateError.AvailabilityZone,
				Error: ec2FastSnapshotRestoreStateError{
					Code:    stateError.Code,
					Message: stateError.Message,
				},
			})
		}
		out = append(out, ec2FastSnapshotRestoreErrorItem{
			SnapshotID: item.SnapshotID,
			FastSnapshotRestoreStateErrorSet: ec2FastSnapshotRestoreStateErrorSet{
				Items: errors,
			},
		})
	}
	return out
}

type ec2FastLaunchResponse struct {
	XMLName               xml.Name
	Xmlns                 string                              `xml:"xmlns,attr"`
	RequestID             string                              `xml:"requestId"`
	ImageID               string                              `xml:"imageId,omitempty"`
	LaunchTemplate        *ec2FastLaunchLaunchTemplate        `xml:"launchTemplate,omitempty"`
	MaxParallelLaunches   *int32                              `xml:"maxParallelLaunches,omitempty"`
	OwnerID               string                              `xml:"ownerId,omitempty"`
	ResourceType          string                              `xml:"resourceType,omitempty"`
	SnapshotConfiguration *ec2FastLaunchSnapshotConfiguration `xml:"snapshotConfiguration,omitempty"`
	State                 string                              `xml:"state,omitempty"`
	StateTransitionReason string                              `xml:"stateTransitionReason,omitempty"`
	StateTransitionTime   string                              `xml:"stateTransitionTime,omitempty"`
}

type ec2FastLaunchLaunchTemplate struct {
	LaunchTemplateID   string `xml:"launchTemplateId,omitempty"`
	LaunchTemplateName string `xml:"launchTemplateName,omitempty"`
	Version            string `xml:"version,omitempty"`
}

type ec2FastLaunchSnapshotConfiguration struct {
	TargetResourceCount int32 `xml:"targetResourceCount,omitempty"`
}

type ec2FastSnapshotRestoresResponse struct {
	XMLName      xml.Name
	Xmlns        string                           `xml:"xmlns,attr"`
	RequestID    string                           `xml:"requestId"`
	Successful   ec2FastSnapshotRestoreSuccessSet `xml:"successful"`
	Unsuccessful ec2FastSnapshotRestoreErrorSet   `xml:"unsuccessful"`
}

type ec2FastSnapshotRestoreSuccessSet struct {
	Items []ec2FastSnapshotRestoreSuccessItem `xml:"item"`
}

type ec2FastSnapshotRestoreSuccessItem struct {
	AvailabilityZone      string `xml:"availabilityZone,omitempty"`
	DisabledTime          string `xml:"disabledTime,omitempty"`
	DisablingTime         string `xml:"disablingTime,omitempty"`
	EnabledTime           string `xml:"enabledTime,omitempty"`
	EnablingTime          string `xml:"enablingTime,omitempty"`
	OptimizingTime        string `xml:"optimizingTime,omitempty"`
	OwnerAlias            string `xml:"ownerAlias,omitempty"`
	OwnerID               string `xml:"ownerId,omitempty"`
	SnapshotID            string `xml:"snapshotId,omitempty"`
	State                 string `xml:"state,omitempty"`
	StateTransitionReason string `xml:"stateTransitionReason,omitempty"`
}

type ec2FastSnapshotRestoreErrorSet struct {
	Items []ec2FastSnapshotRestoreErrorItem `xml:"item"`
}

type ec2FastSnapshotRestoreErrorItem struct {
	SnapshotID                       string                              `xml:"snapshotId,omitempty"`
	FastSnapshotRestoreStateErrorSet ec2FastSnapshotRestoreStateErrorSet `xml:"fastSnapshotRestoreStateErrorSet"`
}

type ec2FastSnapshotRestoreStateErrorSet struct {
	Items []ec2FastSnapshotRestoreStateErrorItem `xml:"item"`
}

type ec2FastSnapshotRestoreStateErrorItem struct {
	AvailabilityZone string                           `xml:"availabilityZone,omitempty"`
	Error            ec2FastSnapshotRestoreStateError `xml:"error"`
}

type ec2FastSnapshotRestoreStateError struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}
