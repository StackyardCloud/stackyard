package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage128Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyInstanceConnectEndpoint":
		preserveClientIP, ok := parseEC2OptionalBoolValue(r.Form.Get("PreserveClientIp"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		okResult, err := s.ec2.ModifyInstanceConnectEndpoint(
			strings.TrimSpace(r.Form.Get("InstanceConnectEndpointId")),
			parseEC2OptionalString(r.Form.Get("IpAddressType")),
			preserveClientIP,
			parseEC2MembersOrItemList(r.Form, "SecurityGroupId"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyInstanceConnectEndpointResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    okResult,
		})
		return true
	case "ModifyInstanceCpuOptions":
		coreCount, ok := parseEC2OptionalInt32(r.Form.Get("CoreCount"))
		if !ok || coreCount == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		threadsPerCore, ok := parseEC2OptionalInt32(r.Form.Get("ThreadsPerCore"))
		if !ok || threadsPerCore == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		newCoreCount, newThreadsPerCore, err := s.ec2.ModifyInstanceCpuOptions(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			*coreCount,
			*threadsPerCore,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage128ModifyInstanceCpuOptionsResponse{
			XMLName:        xml.Name{Local: "ModifyInstanceCpuOptionsResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			CoreCount:      &newCoreCount,
			InstanceID:     parseEC2OptionalString(r.Form.Get("InstanceId")),
			ThreadsPerCore: &newThreadsPerCore,
		})
		return true
	case "ModifyInstanceCreditSpecification":
		specifications, hasSpecifications, ok := parseEC2Stage128InstanceCreditSpecifications(r.Form)
		if !ok || !hasSpecifications {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		successful, unsuccessful, err := s.ec2.ModifyInstanceCreditSpecification(specifications)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage128ModifyInstanceCreditSpecificationResponse{
			XMLName:                                  xml.Name{Local: "ModifyInstanceCreditSpecificationResponse"},
			Xmlns:                                    ec2Namespace,
			RequestID:                                "stackyard-request",
			SuccessfulInstanceCreditSpecificationSet: ec2Stage128SuccessfulInstanceCreditSpecificationSet{Items: ec2Stage128SuccessfulInstanceCreditSpecificationItemsFrom(successful)},
			UnsuccessfulInstanceCreditSpecificationSet: ec2Stage128UnsuccessfulInstanceCreditSpecificationSet{
				Items: ec2Stage128UnsuccessfulInstanceCreditSpecificationItemsFrom(unsuccessful),
			},
		})
		return true
	case "ModifyInstanceEventStartTime":
		notBeforeRaw := strings.TrimSpace(r.Form.Get("NotBefore"))
		if notBeforeRaw == "" {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		notBefore, err := parseEC2RFC3339Time(notBeforeRaw)
		if err != nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		event, err := s.ec2.ModifyInstanceEventStartTime(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("InstanceEventId")),
			notBefore,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage128ModifyInstanceEventStartTimeResponse{
			XMLName:   xml.Name{Local: "ModifyInstanceEventStartTimeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Event: ec2Stage128InstanceStatusEventItem{
				Code:              event.Code,
				Description:       parseEC2OptionalString(event.Description),
				InstanceEventID:   parseEC2OptionalString(event.InstanceEventID),
				NotAfter:          ec2TimeString(event.NotAfter),
				NotBefore:         ec2TimeString(event.NotBefore),
				NotBeforeDeadline: ec2TimeString(event.NotBeforeDeadline),
			},
		})
		return true
	case "ModifyInstanceEventWindow":
		timeRanges, hasTimeRanges, ok := parseEC2Stage128OptionalInstanceEventWindowTimeRanges(r.Form)
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		window, err := s.ec2.ModifyInstanceEventWindow(
			strings.TrimSpace(r.Form.Get("InstanceEventWindowId")),
			ec2OptionalStringPointerFromForm(r.Form, "Name"),
			ec2OptionalStringPointerFromForm(r.Form, "CronExpression"),
			timeRanges,
			hasTimeRanges,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		windowItems := ec2Stage117InstanceEventWindowItemsFrom([]ec2svc.InstanceEventWindowDescription{window})
		windowItem := ec2Stage117InstanceEventWindowItem{}
		if len(windowItems) == 1 {
			windowItem = windowItems[0]
		}
		respondEC2XML(w, ec2Stage128ModifyInstanceEventWindowResponse{
			XMLName:             xml.Name{Local: "ModifyInstanceEventWindowResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			InstanceEventWindow: windowItem,
		})
		return true
	case "ModifyInstanceMaintenanceOptions":
		options, err := s.ec2.ModifyInstanceMaintenanceOptions(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			ec2OptionalStringPointerFromForm(r.Form, "AutoRecovery"),
			ec2OptionalStringPointerFromForm(r.Form, "RebootMigration"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage128ModifyInstanceMaintenanceOptionsResponse{
			XMLName:         xml.Name{Local: "ModifyInstanceMaintenanceOptionsResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			AutoRecovery:    options.AutoRecovery,
			InstanceID:      parseEC2OptionalString(options.InstanceID),
			RebootMigration: options.RebootMigration,
		})
		return true
	case "ModifyInstanceMetadataDefaults":
		var httpPutResponseHopLimit *int32
		if hasEC2Field(r.Form, "HttpPutResponseHopLimit") {
			parsed, ok := parseEC2OptionalInt32(r.Form.Get("HttpPutResponseHopLimit"))
			if !ok || parsed == nil {
				respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
				return true
			}
			httpPutResponseHopLimit = parsed
		}
		okResult, err := s.ec2.ModifyInstanceMetadataDefaults(
			ec2OptionalStringPointerFromForm(r.Form, "HttpEndpoint"),
			ec2OptionalStringPointerFromForm(r.Form, "HttpTokens"),
			ec2OptionalStringPointerFromForm(r.Form, "InstanceMetadataTags"),
			httpPutResponseHopLimit,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyInstanceMetadataDefaultsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    okResult,
		})
		return true
	case "ModifyInstanceMetadataOptions":
		var httpPutResponseHopLimit *int32
		if hasEC2Field(r.Form, "HttpPutResponseHopLimit") {
			parsed, ok := parseEC2OptionalInt32(r.Form.Get("HttpPutResponseHopLimit"))
			if !ok || parsed == nil {
				respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
				return true
			}
			httpPutResponseHopLimit = parsed
		}
		options, err := s.ec2.ModifyInstanceMetadataOptions(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			ec2OptionalStringPointerFromForm(r.Form, "HttpEndpoint"),
			ec2OptionalStringPointerFromForm(r.Form, "HttpProtocolIpv6"),
			ec2OptionalStringPointerFromForm(r.Form, "HttpTokens"),
			ec2OptionalStringPointerFromForm(r.Form, "InstanceMetadataTags"),
			httpPutResponseHopLimit,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage128ModifyInstanceMetadataOptionsResponse{
			XMLName:    xml.Name{Local: "ModifyInstanceMetadataOptionsResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			InstanceID: parseEC2OptionalString(options.InstanceID),
			InstanceMetadataOptions: ec2Stage128InstanceMetadataOptionsItem{
				HttpEndpoint:            options.HttpEndpoint,
				HttpProtocolIpv6:        options.HttpProtocolIpv6,
				HttpPutResponseHopLimit: options.HttpPutResponseHopLimit,
				HttpTokens:              options.HttpTokens,
				InstanceMetadataTags:    options.InstanceMetadataTags,
				State:                   options.State,
			},
		})
		return true
	case "ModifyInstanceNetworkPerformanceOptions":
		bandwidthWeighting, err := s.ec2.ModifyInstanceNetworkPerformanceOptions(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("BandwidthWeighting")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage128ModifyInstanceNetworkPerformanceOptionsResponse{
			XMLName:            xml.Name{Local: "ModifyInstanceNetworkPerformanceOptionsResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			BandwidthWeighting: bandwidthWeighting,
			InstanceID:         parseEC2OptionalString(r.Form.Get("InstanceId")),
		})
		return true
	case "ModifyInstancePlacement":
		var partitionNumber *int32
		if hasEC2Field(r.Form, "PartitionNumber") {
			parsed, ok := parseEC2OptionalInt32(r.Form.Get("PartitionNumber"))
			if !ok || parsed == nil {
				respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
				return true
			}
			partitionNumber = parsed
		}
		okResult, err := s.ec2.ModifyInstancePlacement(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			ec2OptionalStringPointerFromForm(r.Form, "Affinity"),
			ec2OptionalStringPointerFromForm(r.Form, "GroupId"),
			ec2OptionalStringPointerFromForm(r.Form, "GroupName"),
			ec2OptionalStringPointerFromForm(r.Form, "HostId"),
			ec2OptionalStringPointerFromForm(r.Form, "HostResourceGroupArn"),
			ec2OptionalStringPointerFromForm(r.Form, "Tenancy"),
			partitionNumber,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyInstancePlacementResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    okResult,
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage128InstanceCreditSpecifications(values url.Values) ([]ec2svc.InstanceCreditSpecificationRequest, bool, bool) {
	if !hasEC2PrefixedField(values, "InstanceCreditSpecification.") {
		return nil, false, true
	}

	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, "InstanceCreditSpecification.") {
			continue
		}
		rest := strings.TrimPrefix(key, "InstanceCreditSpecification.")
		rest = strings.TrimPrefix(rest, "Item.")
		rest = strings.TrimPrefix(rest, "Member.")
		part := rest
		if dot := strings.IndexByte(rest, '.'); dot >= 0 {
			part = rest[:dot]
		}
		idx, err := strconv.Atoi(part)
		if err != nil || idx <= 0 {
			continue
		}
		indices[idx] = struct{}{}
	}

	ordered := make([]int, 0, len(indices))
	for idx := range indices {
		ordered = append(ordered, idx)
	}
	sort.Ints(ordered)

	out := make([]ec2svc.InstanceCreditSpecificationRequest, 0, len(ordered))
	for _, idx := range ordered {
		instanceID := ec2ReservationFleetIndexedField(values, []string{"InstanceCreditSpecification."}, idx, "InstanceId")
		cpuCredits := ec2ReservationFleetIndexedField(values, []string{"InstanceCreditSpecification."}, idx, "CpuCredits")
		if instanceID == "" || cpuCredits == "" {
			return nil, true, false
		}
		out = append(out, ec2svc.InstanceCreditSpecificationRequest{
			CpuCredits: cpuCredits,
			InstanceID: instanceID,
		})
	}

	if len(out) == 0 {
		return nil, true, false
	}
	return out, true, true
}

func parseEC2Stage128OptionalInstanceEventWindowTimeRanges(values url.Values) ([]ec2svc.InstanceEventWindowTimeRange, bool, bool) {
	if !hasEC2PrefixedField(values, "TimeRange.") {
		return nil, false, true
	}
	timeRanges, ok := parseEC2Stage107InstanceEventWindowTimeRanges(values)
	if !ok {
		return nil, true, false
	}
	return timeRanges, true, true
}

func ec2Stage128SuccessfulInstanceCreditSpecificationItemsFrom(in []string) []ec2Stage128SuccessfulInstanceCreditSpecificationItem {
	out := make([]ec2Stage128SuccessfulInstanceCreditSpecificationItem, 0, len(in))
	for _, instanceID := range in {
		instanceID = strings.TrimSpace(instanceID)
		if instanceID == "" {
			continue
		}
		out = append(out, ec2Stage128SuccessfulInstanceCreditSpecificationItem{InstanceID: instanceID})
	}
	return out
}

func ec2Stage128UnsuccessfulInstanceCreditSpecificationItemsFrom(in []ec2svc.InstanceCreditSpecificationUnsuccessful) []ec2Stage128UnsuccessfulInstanceCreditSpecificationItem {
	out := make([]ec2Stage128UnsuccessfulInstanceCreditSpecificationItem, 0, len(in))
	for _, item := range in {
		instanceID := strings.TrimSpace(item.InstanceID)
		if instanceID == "" {
			continue
		}
		out = append(out, ec2Stage128UnsuccessfulInstanceCreditSpecificationItem{
			InstanceID: instanceID,
			Error: ec2Stage128UnsuccessfulInstanceCreditSpecificationError{
				Code:    item.Code,
				Message: parseEC2OptionalString(item.Message),
			},
		})
	}
	return out
}

type ec2Stage128ModifyInstanceCpuOptionsResponse struct {
	XMLName        xml.Name `xml:"ModifyInstanceCpuOptionsResponse"`
	Xmlns          string   `xml:"xmlns,attr"`
	RequestID      string   `xml:"requestId"`
	CoreCount      *int32   `xml:"coreCount,omitempty"`
	InstanceID     *string  `xml:"instanceId,omitempty"`
	ThreadsPerCore *int32   `xml:"threadsPerCore,omitempty"`
}

type ec2Stage128ModifyInstanceCreditSpecificationResponse struct {
	XMLName                                    xml.Name                                              `xml:"ModifyInstanceCreditSpecificationResponse"`
	Xmlns                                      string                                                `xml:"xmlns,attr"`
	RequestID                                  string                                                `xml:"requestId"`
	SuccessfulInstanceCreditSpecificationSet   ec2Stage128SuccessfulInstanceCreditSpecificationSet   `xml:"successfulInstanceCreditSpecificationSet"`
	UnsuccessfulInstanceCreditSpecificationSet ec2Stage128UnsuccessfulInstanceCreditSpecificationSet `xml:"unsuccessfulInstanceCreditSpecificationSet"`
}

type ec2Stage128SuccessfulInstanceCreditSpecificationSet struct {
	Items []ec2Stage128SuccessfulInstanceCreditSpecificationItem `xml:"item"`
}

type ec2Stage128SuccessfulInstanceCreditSpecificationItem struct {
	InstanceID string `xml:"instanceId,omitempty"`
}

type ec2Stage128UnsuccessfulInstanceCreditSpecificationSet struct {
	Items []ec2Stage128UnsuccessfulInstanceCreditSpecificationItem `xml:"item"`
}

type ec2Stage128UnsuccessfulInstanceCreditSpecificationItem struct {
	Error      ec2Stage128UnsuccessfulInstanceCreditSpecificationError `xml:"error"`
	InstanceID string                                                  `xml:"instanceId,omitempty"`
}

type ec2Stage128UnsuccessfulInstanceCreditSpecificationError struct {
	Code    string  `xml:"code,omitempty"`
	Message *string `xml:"message,omitempty"`
}

type ec2Stage128ModifyInstanceEventStartTimeResponse struct {
	XMLName   xml.Name                           `xml:"ModifyInstanceEventStartTimeResponse"`
	Xmlns     string                             `xml:"xmlns,attr"`
	RequestID string                             `xml:"requestId"`
	Event     ec2Stage128InstanceStatusEventItem `xml:"event"`
}

type ec2Stage128InstanceStatusEventItem struct {
	Code              string  `xml:"code,omitempty"`
	Description       *string `xml:"description,omitempty"`
	InstanceEventID   *string `xml:"instanceEventId,omitempty"`
	NotAfter          string  `xml:"notAfter,omitempty"`
	NotBefore         string  `xml:"notBefore,omitempty"`
	NotBeforeDeadline string  `xml:"notBeforeDeadline,omitempty"`
}

type ec2Stage128ModifyInstanceEventWindowResponse struct {
	XMLName             xml.Name                           `xml:"ModifyInstanceEventWindowResponse"`
	Xmlns               string                             `xml:"xmlns,attr"`
	RequestID           string                             `xml:"requestId"`
	InstanceEventWindow ec2Stage117InstanceEventWindowItem `xml:"instanceEventWindow"`
}

type ec2Stage128ModifyInstanceMaintenanceOptionsResponse struct {
	XMLName         xml.Name `xml:"ModifyInstanceMaintenanceOptionsResponse"`
	Xmlns           string   `xml:"xmlns,attr"`
	RequestID       string   `xml:"requestId"`
	AutoRecovery    string   `xml:"autoRecovery,omitempty"`
	InstanceID      *string  `xml:"instanceId,omitempty"`
	RebootMigration string   `xml:"rebootMigration,omitempty"`
}

type ec2Stage128ModifyInstanceMetadataOptionsResponse struct {
	XMLName                 xml.Name                               `xml:"ModifyInstanceMetadataOptionsResponse"`
	Xmlns                   string                                 `xml:"xmlns,attr"`
	RequestID               string                                 `xml:"requestId"`
	InstanceID              *string                                `xml:"instanceId,omitempty"`
	InstanceMetadataOptions ec2Stage128InstanceMetadataOptionsItem `xml:"instanceMetadataOptions"`
}

type ec2Stage128InstanceMetadataOptionsItem struct {
	HttpEndpoint            string `xml:"httpEndpoint,omitempty"`
	HttpProtocolIpv6        string `xml:"httpProtocolIpv6,omitempty"`
	HttpPutResponseHopLimit *int32 `xml:"httpPutResponseHopLimit,omitempty"`
	HttpTokens              string `xml:"httpTokens,omitempty"`
	InstanceMetadataTags    string `xml:"instanceMetadataTags,omitempty"`
	State                   string `xml:"state,omitempty"`
}

type ec2Stage128ModifyInstanceNetworkPerformanceOptionsResponse struct {
	XMLName            xml.Name `xml:"ModifyInstanceNetworkPerformanceOptionsResponse"`
	Xmlns              string   `xml:"xmlns,attr"`
	RequestID          string   `xml:"requestId"`
	BandwidthWeighting string   `xml:"bandwidthWeighting,omitempty"`
	InstanceID         *string  `xml:"instanceId,omitempty"`
}
