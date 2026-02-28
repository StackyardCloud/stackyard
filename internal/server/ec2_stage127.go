package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage127Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ListSnapshotsInRecycleBin":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		snapshots, nextToken, err := s.ec2.ListSnapshotsInRecycleBin(
			parseEC2MembersOrItemList(r.Form, "SnapshotId"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage127ListSnapshotsInRecycleBinResponse{
			XMLName:     xml.Name{Local: "ListSnapshotsInRecycleBinResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			SnapshotSet: ec2Stage127SnapshotRecycleBinInfoSet{Items: ec2Stage127SnapshotRecycleBinInfoItemsFrom(snapshots)},
			NextToken:   nextToken,
		})
		return true
	case "LockSnapshot":
		lockDuration, ok := parseEC2OptionalInt32(r.Form.Get("LockDuration"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		coolOffPeriod, ok := parseEC2OptionalInt32(r.Form.Get("CoolOffPeriod"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		var expirationDate *time.Time
		if hasEC2Field(r.Form, "ExpirationDate") {
			raw := strings.TrimSpace(r.Form.Get("ExpirationDate"))
			if raw != "" {
				parsed, err := parseEC2RFC3339Time(raw)
				if err != nil {
					respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
					return true
				}
				expirationDate = &parsed
			}
		}

		lockedSnapshot, err := s.ec2.LockSnapshot(
			strings.TrimSpace(r.Form.Get("SnapshotId")),
			strings.TrimSpace(r.Form.Get("LockMode")),
			lockDuration,
			coolOffPeriod,
			expirationDate,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage127LockSnapshotResponse{
			XMLName:                xml.Name{Local: "LockSnapshotResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			CoolOffPeriod:          lockedSnapshot.CoolOffPeriod,
			CoolOffPeriodExpiresOn: ec2TimeString(lockedSnapshot.CoolOffPeriodExpiresOn),
			LockCreatedOn:          ec2TimeString(lockedSnapshot.LockCreatedOn),
			LockDuration:           lockedSnapshot.LockDuration,
			LockDurationStartTime:  ec2TimeString(lockedSnapshot.LockDurationStartTime),
			LockExpiresOn:          ec2TimeString(lockedSnapshot.LockExpiresOn),
			LockState:              lockedSnapshot.LockState,
			SnapshotID:             lockedSnapshot.SnapshotID,
		})
		return true
	case "ModifyAvailabilityZoneGroup":
		okResult, err := s.ec2.ModifyAvailabilityZoneGroup(
			strings.TrimSpace(r.Form.Get("GroupName")),
			strings.TrimSpace(r.Form.Get("OptInStatus")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyAvailabilityZoneGroupResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    okResult,
		})
		return true
	case "ModifyCapacityReservation":
		accept, ok := parseEC2OptionalBoolValue(r.Form.Get("Accept"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		instanceCount, ok := parseEC2OptionalInt32(r.Form.Get("InstanceCount"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		var endDate *time.Time
		if hasEC2Field(r.Form, "EndDate") {
			raw := strings.TrimSpace(r.Form.Get("EndDate"))
			if raw != "" {
				parsed, err := parseEC2RFC3339Time(raw)
				if err != nil {
					respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
					return true
				}
				endDate = &parsed
			}
		}

		okResult, err := s.ec2.ModifyCapacityReservation(
			strings.TrimSpace(r.Form.Get("CapacityReservationId")),
			accept,
			parseEC2OptionalString(r.Form.Get("AdditionalInfo")),
			endDate,
			strings.TrimSpace(r.Form.Get("EndDateType")),
			instanceCount,
			strings.TrimSpace(r.Form.Get("InstanceMatchCriteria")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyCapacityReservationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    okResult,
		})
		return true
	case "ModifyCapacityReservationFleet":
		totalTargetCapacity, ok := parseEC2OptionalInt32(r.Form.Get("TotalTargetCapacity"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		removeEndDate, ok := parseEC2OptionalBoolValue(r.Form.Get("RemoveEndDate"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		var endDate *time.Time
		if hasEC2Field(r.Form, "EndDate") {
			raw := strings.TrimSpace(r.Form.Get("EndDate"))
			if raw != "" {
				parsed, err := parseEC2RFC3339Time(raw)
				if err != nil {
					respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
					return true
				}
				endDate = &parsed
			}
		}

		okResult, err := s.ec2.ModifyCapacityReservationFleet(
			strings.TrimSpace(r.Form.Get("CapacityReservationFleetId")),
			endDate,
			removeEndDate,
			totalTargetCapacity,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyCapacityReservationFleetResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    okResult,
		})
		return true
	case "ModifyDefaultCreditSpecification":
		specification, err := s.ec2.ModifyDefaultCreditSpecification(
			strings.TrimSpace(r.Form.Get("InstanceFamily")),
			strings.TrimSpace(r.Form.Get("CpuCredits")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage127ModifyDefaultCreditSpecificationResponse{
			XMLName:   xml.Name{Local: "ModifyDefaultCreditSpecificationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			InstanceFamilyCreditSpecification: ec2Stage124InstanceFamilyCreditSpecificationItem{
				CpuCredits:     specification.CpuCredits,
				InstanceFamily: specification.InstanceFamily,
			},
		})
		return true
	case "ModifyFleet":
		totalTargetCapacity, ok := parseEC2OptionalInt32(r.Form.Get("TargetCapacitySpecification.TotalTargetCapacity"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		onDemandTargetCapacity, ok := parseEC2OptionalInt32(r.Form.Get("TargetCapacitySpecification.OnDemandTargetCapacity"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		spotTargetCapacity, ok := parseEC2OptionalInt32(r.Form.Get("TargetCapacitySpecification.SpotTargetCapacity"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		okResult, err := s.ec2.ModifyFleet(
			strings.TrimSpace(r.Form.Get("FleetId")),
			hasEC2PrefixedField(r.Form, "LaunchTemplateConfig."),
			totalTargetCapacity,
			onDemandTargetCapacity,
			spotTargetCapacity,
			parseEC2OptionalString(r.Form.Get("Context")),
			strings.TrimSpace(r.Form.Get("ExcessCapacityTerminationPolicy")),
			parseEC2Stage127FleetInstanceType(r.Form),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyFleetResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    okResult,
		})
		return true
	case "ModifyFpgaImageAttribute":
		attribute, err := s.ec2.ModifyFpgaImageAttribute(
			strings.TrimSpace(r.Form.Get("FpgaImageId")),
			strings.TrimSpace(r.Form.Get("Attribute")),
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2OptionalString(r.Form.Get("Name")),
			strings.TrimSpace(r.Form.Get("OperationType")),
			parseEC2MembersOrItemList(r.Form, "ProductCode"),
			parseEC2MembersOrItemList(r.Form, "UserGroup"),
			parseEC2MembersOrItemList(r.Form, "UserId"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage127ModifyFpgaImageAttributeResponse{
			XMLName:            xml.Name{Local: "ModifyFpgaImageAttributeResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			FpgaImageAttribute: ec2Stage116FpgaImageAttributeItemFrom(attribute),
		})
		return true
	case "ModifyHosts":
		successful, unsuccessful, err := s.ec2.ModifyHosts(
			parseEC2MembersOrItemList(r.Form, "HostId"),
			parseEC2OptionalString(r.Form.Get("AutoPlacement")),
			parseEC2OptionalString(r.Form.Get("HostMaintenance")),
			parseEC2OptionalString(r.Form.Get("HostRecovery")),
			parseEC2OptionalString(r.Form.Get("InstanceFamily")),
			parseEC2OptionalString(r.Form.Get("InstanceType")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage127ModifyHostsResponse{
			XMLName:      xml.Name{Local: "ModifyHostsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Successful:   ec2StringSet{Items: successful},
			Unsuccessful: ec2UnsuccessfulItemSet{Items: ec2UnsuccessfulItems(unsuccessful)},
		})
		return true
	case "ModifyInstanceCapacityReservationAttributes":
		okResult, err := s.ec2.ModifyInstanceCapacityReservationAttributes(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("CapacityReservationSpecification.CapacityReservationPreference")),
			strings.TrimSpace(r.Form.Get("CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationId")),
			strings.TrimSpace(r.Form.Get("CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupArn")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyInstanceCapacityReservationAttributesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    okResult,
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage127FleetInstanceType(values url.Values) string {
	keys := make([]string, 0)
	for key := range values {
		if !strings.HasPrefix(key, "LaunchTemplateConfig.") {
			continue
		}
		if !strings.HasSuffix(key, ".InstanceType") {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := strings.TrimSpace(values.Get(key))
		if value != "" {
			return value
		}
	}
	return ""
}

func ec2Stage127SnapshotRecycleBinInfoItemsFrom(in []ec2svc.SnapshotRecycleBinInfo) []ec2Stage127SnapshotRecycleBinInfoItem {
	out := make([]ec2Stage127SnapshotRecycleBinInfoItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage127SnapshotRecycleBinInfoItem{
			Description:         item.Description,
			RecycleBinEnterTime: ec2TimeString(item.RecycleBinEnterTime),
			RecycleBinExitTime:  ec2TimeString(item.RecycleBinExitTime),
			SnapshotID:          item.SnapshotID,
		})
	}
	return out
}

type ec2Stage127ListSnapshotsInRecycleBinResponse struct {
	XMLName     xml.Name                             `xml:"ListSnapshotsInRecycleBinResponse"`
	Xmlns       string                               `xml:"xmlns,attr"`
	RequestID   string                               `xml:"requestId"`
	SnapshotSet ec2Stage127SnapshotRecycleBinInfoSet `xml:"snapshotSet"`
	NextToken   *string                              `xml:"nextToken,omitempty"`
}

type ec2Stage127SnapshotRecycleBinInfoSet struct {
	Items []ec2Stage127SnapshotRecycleBinInfoItem `xml:"item"`
}

type ec2Stage127SnapshotRecycleBinInfoItem struct {
	Description         string `xml:"description,omitempty"`
	RecycleBinEnterTime string `xml:"recycleBinEnterTime,omitempty"`
	RecycleBinExitTime  string `xml:"recycleBinExitTime,omitempty"`
	SnapshotID          string `xml:"snapshotId,omitempty"`
}

type ec2Stage127LockSnapshotResponse struct {
	XMLName                xml.Name `xml:"LockSnapshotResponse"`
	Xmlns                  string   `xml:"xmlns,attr"`
	RequestID              string   `xml:"requestId"`
	CoolOffPeriod          *int32   `xml:"coolOffPeriod,omitempty"`
	CoolOffPeriodExpiresOn string   `xml:"coolOffPeriodExpiresOn,omitempty"`
	LockCreatedOn          string   `xml:"lockCreatedOn,omitempty"`
	LockDuration           *int32   `xml:"lockDuration,omitempty"`
	LockDurationStartTime  string   `xml:"lockDurationStartTime,omitempty"`
	LockExpiresOn          string   `xml:"lockExpiresOn,omitempty"`
	LockState              string   `xml:"lockState,omitempty"`
	SnapshotID             string   `xml:"snapshotId,omitempty"`
}

type ec2Stage127ModifyDefaultCreditSpecificationResponse struct {
	XMLName                           xml.Name                                         `xml:"ModifyDefaultCreditSpecificationResponse"`
	Xmlns                             string                                           `xml:"xmlns,attr"`
	RequestID                         string                                           `xml:"requestId"`
	InstanceFamilyCreditSpecification ec2Stage124InstanceFamilyCreditSpecificationItem `xml:"instanceFamilyCreditSpecification"`
}

type ec2Stage127ModifyFpgaImageAttributeResponse struct {
	XMLName            xml.Name                          `xml:"ModifyFpgaImageAttributeResponse"`
	Xmlns              string                            `xml:"xmlns,attr"`
	RequestID          string                            `xml:"requestId"`
	FpgaImageAttribute ec2Stage116FpgaImageAttributeItem `xml:"fpgaImageAttribute"`
}

type ec2Stage127ModifyHostsResponse struct {
	XMLName      xml.Name               `xml:"ModifyHostsResponse"`
	Xmlns        string                 `xml:"xmlns,attr"`
	RequestID    string                 `xml:"requestId"`
	Successful   ec2StringSet           `xml:"successful"`
	Unsuccessful ec2UnsuccessfulItemSet `xml:"unsuccessful"`
}
