package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage116Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeFastLaunchImages":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		images, nextToken, err := s.ec2.DescribeFastLaunchImages(
			parseEC2MembersOrItemList(r.Form, "ImageId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage116DescribeFastLaunchImagesResponse{
			XMLName:          xml.Name{Local: "DescribeFastLaunchImagesResponse"},
			Xmlns:            ec2Namespace,
			RequestID:        "stackyard-request",
			FastLaunchImages: ec2Stage116FastLaunchImageSet{Items: ec2Stage116FastLaunchImageItemsFrom(images)},
			NextToken:        nextToken,
		})
		return true
	case "DescribeFastSnapshotRestores":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		restores, nextToken, err := s.ec2.DescribeFastSnapshotRestores(
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage116DescribeFastSnapshotRestoresResponse{
			XMLName:              xml.Name{Local: "DescribeFastSnapshotRestoresResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			FastSnapshotRestores: ec2Stage116FastSnapshotRestoreSet{Items: ec2FastSnapshotRestoreSuccessItemsFrom(restores)},
			NextToken:            nextToken,
		})
		return true
	case "DescribeFleetHistory":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		startTimeRaw := strings.TrimSpace(r.Form.Get("StartTime"))
		if startTimeRaw == "" {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		startTime, err := parseEC2RFC3339Time(startTimeRaw)
		if err != nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		fleetID, records, lastEvaluatedTime, nextToken, err := s.ec2.DescribeFleetHistory(
			strings.TrimSpace(r.Form.Get("FleetId")),
			startTime,
			strings.TrimSpace(r.Form.Get("EventType")),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage116DescribeFleetHistoryResponse{
			XMLName:           xml.Name{Local: "DescribeFleetHistoryResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			FleetID:           fleetID,
			HistoryRecordSet:  ec2Stage116HistoryRecordSet{Items: ec2Stage116HistoryRecordItemsFrom(records)},
			LastEvaluatedTime: ec2Stage116TimeString(lastEvaluatedTime),
			NextToken:         nextToken,
			StartTime:         startTime.UTC().Format(time.RFC3339),
		})
		return true
	case "DescribeFleetInstances":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		fleetID, instances, nextToken, err := s.ec2.DescribeFleetInstances(
			strings.TrimSpace(r.Form.Get("FleetId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage116DescribeFleetInstancesResponse{
			XMLName:           xml.Name{Local: "DescribeFleetInstancesResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			ActiveInstanceSet: ec2Stage116ActiveInstanceSet{Items: ec2Stage116ActiveInstanceItemsFrom(instances)},
			FleetID:           fleetID,
			NextToken:         nextToken,
		})
		return true
	case "DescribeFleets":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		fleets, nextToken, err := s.ec2.DescribeFleets(
			parseEC2MembersOrItemList(r.Form, "FleetId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage116DescribeFleetsResponse{
			XMLName:   xml.Name{Local: "DescribeFleetsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			FleetSet:  ec2Stage116FleetSet{Items: ec2Stage116FleetItemsFrom(fleets)},
			NextToken: nextToken,
		})
		return true
	case "DescribeFlowLogs":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		flowLogs, nextToken, err := s.ec2.DescribeFlowLogs(
			parseEC2MembersOrItemList(r.Form, "FlowLogId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage116DescribeFlowLogsResponse{
			XMLName:    xml.Name{Local: "DescribeFlowLogsResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			FlowLogSet: ec2Stage116FlowLogSet{Items: ec2Stage116FlowLogItemsFrom(flowLogs)},
			NextToken:  nextToken,
		})
		return true
	case "DescribeFpgaImageAttribute":
		attribute := strings.TrimSpace(r.Form.Get("Attribute"))
		if attribute == "" {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		fpgaImageAttribute, err := s.ec2.DescribeFpgaImageAttribute(
			strings.TrimSpace(r.Form.Get("FpgaImageId")),
			attribute,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage116DescribeFpgaImageAttributeResponse{
			XMLName:            xml.Name{Local: "DescribeFpgaImageAttributeResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			FpgaImageAttribute: ec2Stage116FpgaImageAttributeItemFrom(fpgaImageAttribute),
		})
		return true
	case "DescribeFpgaImages":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		fpgaImages, nextToken, err := s.ec2.DescribeFpgaImages(
			parseEC2MembersOrItemList(r.Form, "FpgaImageId"),
			parseEC2MembersOrItemList(r.Form, "Owner"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage116DescribeFpgaImagesResponse{
			XMLName:      xml.Name{Local: "DescribeFpgaImagesResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			FpgaImageSet: ec2Stage116FpgaImageSet{Items: ec2Stage116FpgaImageItemsFrom(fpgaImages)},
			NextToken:    nextToken,
		})
		return true
	case "DescribeHostReservationOfferings":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		minDuration, ok := parseEC2OptionalInt32(r.Form.Get("MinDuration"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		maxDuration, ok := parseEC2OptionalInt32(r.Form.Get("MaxDuration"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		offerings, nextToken, err := s.ec2.DescribeHostReservationOfferings(
			parseEC2OptionalString(r.Form.Get("OfferingId")),
			minDuration,
			maxDuration,
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage116DescribeHostReservationOfferingsResponse{
			XMLName:     xml.Name{Local: "DescribeHostReservationOfferingsResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			OfferingSet: ec2Stage116HostOfferingSet{Items: ec2Stage116HostOfferingItemsFrom(offerings)},
			NextToken:   nextToken,
		})
		return true
	case "DescribeHostReservations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		reservations, nextToken, err := s.ec2.DescribeHostReservations(
			parseEC2MembersOrItemList(r.Form, "HostReservationIdSet"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage116DescribeHostReservationsResponse{
			XMLName:            xml.Name{Local: "DescribeHostReservationsResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			HostReservationSet: ec2Stage116HostReservationSet{Items: ec2Stage116HostReservationItemsFrom(reservations)},
			NextToken:          nextToken,
		})
		return true
	default:
		return false
	}
}

func ec2Stage116FastLaunchImageItemsFrom(in []ec2svc.FastLaunchConfiguration) []ec2Stage116FastLaunchImageItem {
	out := make([]ec2Stage116FastLaunchImageItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage116FastLaunchImageItem{
			ImageID:               item.ImageID,
			LaunchTemplate:        ec2FastLaunchLaunchTemplateFrom(item.LaunchTemplate),
			MaxParallelLaunches:   item.MaxParallelLaunches,
			OwnerID:               item.OwnerID,
			ResourceType:          item.ResourceType,
			SnapshotConfiguration: ec2FastLaunchSnapshotConfigurationFrom(item.SnapshotConfiguration),
			State:                 item.State,
			StateTransitionReason: item.StateTransitionReason,
			StateTransitionTime:   ec2Stage116TimeString(&item.StateTransitionTime),
		})
	}
	return out
}

func ec2Stage116HistoryRecordItemsFrom(in []ec2svc.FleetHistoryRecord) []ec2Stage116HistoryRecordItem {
	out := make([]ec2Stage116HistoryRecordItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage116HistoryRecordItem{
			EventInformation: ec2Stage116EventInformationItem{
				EventDescription: item.EventDescription,
				EventSubType:     item.EventSubType,
				InstanceID:       item.InstanceID,
			},
			EventType: item.EventType,
			Timestamp: item.Timestamp.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func ec2Stage116ActiveInstanceItemsFrom(in []ec2svc.FleetActiveInstance) []ec2Stage116ActiveInstanceItem {
	out := make([]ec2Stage116ActiveInstanceItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage116ActiveInstanceItem{
			InstanceHealth:        item.InstanceHealth,
			InstanceID:            item.InstanceID,
			InstanceType:          item.InstanceType,
			SpotInstanceRequestID: item.SpotInstanceRequestID,
		})
	}
	return out
}

func ec2Stage116FleetItemsFrom(in []ec2svc.Fleet) []ec2Stage116FleetItem {
	out := make([]ec2Stage116FleetItem, 0, len(in))
	for _, item := range in {
		fulfilledCapacity := 0.0
		for _, fleetInstance := range item.Instances {
			fulfilledCapacity += float64(len(fleetInstance.InstanceIDs))
		}
		out = append(out, ec2Stage116FleetItem{
			ActivityStatus:    "fulfilled",
			FleetID:           item.FleetID,
			FleetState:        "active",
			FulfilledCapacity: fulfilledCapacity,
			FleetInstanceSet:  ec2Stage116DescribeFleetsInstancesSet{Items: ec2Stage116DescribeFleetsInstancesItemsFrom(item.Instances)},
			TagSet:            ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage116DescribeFleetsInstancesItemsFrom(in []ec2svc.FleetInstance) []ec2Stage116DescribeFleetsInstancesItem {
	out := make([]ec2Stage116DescribeFleetsInstancesItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage116DescribeFleetsInstancesItem{
			InstanceIDs:  ec2StringSet{Items: append([]string(nil), item.InstanceIDs...)},
			InstanceType: item.InstanceType,
			Lifecycle:    item.Lifecycle,
		})
	}
	return out
}

func ec2Stage116FlowLogItemsFrom(in []ec2svc.FlowLog) []ec2Stage116FlowLogItem {
	out := make([]ec2Stage116FlowLogItem, 0, len(in))
	now := time.Now().UTC().Format(time.RFC3339)
	for _, item := range in {
		out = append(out, ec2Stage116FlowLogItem{
			CreationTime:  now,
			FlowLogID:     item.FlowLogID,
			FlowLogStatus: "ACTIVE",
			ResourceID:    item.ResourceID,
			TagSet:        ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			TrafficType:   item.TrafficType,
		})
	}
	return out
}

func ec2Stage116FpgaImageAttributeItemFrom(in ec2svc.FpgaImageAttributeView) ec2Stage116FpgaImageAttributeItem {
	out := ec2Stage116FpgaImageAttributeItem{
		Description: in.Description,
		FpgaImageID: in.FpgaImageID,
		Name:        in.Name,
	}
	if len(in.LoadPermissionUserIDs) > 0 {
		loadPermissions := make([]ec2Stage116LoadPermissionItem, 0, len(in.LoadPermissionUserIDs))
		for _, userID := range in.LoadPermissionUserIDs {
			loadPermissions = append(loadPermissions, ec2Stage116LoadPermissionItem{UserID: userID})
		}
		out.LoadPermissions = &ec2Stage116LoadPermissionSet{Items: loadPermissions}
	}
	if len(in.ProductCodes) > 0 {
		productCodes := make([]ec2Stage116ProductCodeItem, 0, len(in.ProductCodes))
		for _, code := range in.ProductCodes {
			productCodes = append(productCodes, ec2Stage116ProductCodeItem{
				ProductCode: code,
				Type:        "marketplace",
			})
		}
		out.ProductCodes = &ec2Stage116ProductCodeSet{Items: productCodes}
	}
	return out
}

func ec2Stage116FpgaImageItemsFrom(in []ec2svc.FpgaImage) []ec2Stage116FpgaImageItem {
	out := make([]ec2Stage116FpgaImageItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage116FpgaImageItem{
			Description:       item.Description,
			FpgaImageGlobalID: item.FpgaImageGlobalID,
			FpgaImageID:       item.FpgaImageID,
			Name:              item.Name,
			OwnerID:           ec2svc.DefaultAccountID,
			Public:            false,
			State: ec2Stage116FpgaImageStateItem{
				Code:    "available",
				Message: "available",
			},
			Tags: ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage116HostOfferingItemsFrom(in []ec2svc.HostOffering) []ec2Stage116HostOfferingItem {
	out := make([]ec2Stage116HostOfferingItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage116HostOfferingItem{
			CurrencyCode:   item.CurrencyCode,
			Duration:       item.Duration,
			HourlyPrice:    item.HourlyPrice,
			InstanceFamily: item.InstanceFamily,
			OfferingID:     item.OfferingID,
			PaymentOption:  item.PaymentOption,
			UpfrontPrice:   item.UpfrontPrice,
		})
	}
	return out
}

func ec2Stage116HostReservationItemsFrom(in []ec2svc.HostReservation) []ec2Stage116HostReservationItem {
	out := make([]ec2Stage116HostReservationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage116HostReservationItem{
			Count:             item.Count,
			CurrencyCode:      item.CurrencyCode,
			Duration:          item.Duration,
			End:               item.End.UTC().Format(time.RFC3339),
			HostIDSet:         ec2StringSet{Items: append([]string(nil), item.HostIDs...)},
			HostReservationID: item.HostReservationID,
			HourlyPrice:       item.HourlyPrice,
			InstanceFamily:    item.InstanceFamily,
			OfferingID:        item.OfferingID,
			PaymentOption:     item.PaymentOption,
			Start:             item.Start.UTC().Format(time.RFC3339),
			State:             item.State,
			TagSet:            ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			UpfrontPrice:      item.UpfrontPrice,
		})
	}
	return out
}

func ec2Stage116TimeString(in *time.Time) string {
	if in == nil || in.IsZero() {
		return ""
	}
	return in.UTC().Format(time.RFC3339)
}

type ec2Stage116DescribeFastLaunchImagesResponse struct {
	XMLName          xml.Name                      `xml:"DescribeFastLaunchImagesResponse"`
	Xmlns            string                        `xml:"xmlns,attr"`
	RequestID        string                        `xml:"requestId"`
	FastLaunchImages ec2Stage116FastLaunchImageSet `xml:"fastLaunchImageSet"`
	NextToken        *string                       `xml:"nextToken,omitempty"`
}

type ec2Stage116FastLaunchImageSet struct {
	Items []ec2Stage116FastLaunchImageItem `xml:"item"`
}

type ec2Stage116FastLaunchImageItem struct {
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

type ec2Stage116DescribeFastSnapshotRestoresResponse struct {
	XMLName              xml.Name                          `xml:"DescribeFastSnapshotRestoresResponse"`
	Xmlns                string                            `xml:"xmlns,attr"`
	RequestID            string                            `xml:"requestId"`
	FastSnapshotRestores ec2Stage116FastSnapshotRestoreSet `xml:"fastSnapshotRestoreSet"`
	NextToken            *string                           `xml:"nextToken,omitempty"`
}

type ec2Stage116FastSnapshotRestoreSet struct {
	Items []ec2FastSnapshotRestoreSuccessItem `xml:"item"`
}

type ec2Stage116DescribeFleetHistoryResponse struct {
	XMLName           xml.Name                    `xml:"DescribeFleetHistoryResponse"`
	Xmlns             string                      `xml:"xmlns,attr"`
	RequestID         string                      `xml:"requestId"`
	FleetID           string                      `xml:"fleetId,omitempty"`
	HistoryRecordSet  ec2Stage116HistoryRecordSet `xml:"historyRecordSet"`
	LastEvaluatedTime string                      `xml:"lastEvaluatedTime,omitempty"`
	NextToken         *string                     `xml:"nextToken,omitempty"`
	StartTime         string                      `xml:"startTime,omitempty"`
}

type ec2Stage116HistoryRecordSet struct {
	Items []ec2Stage116HistoryRecordItem `xml:"item"`
}

type ec2Stage116HistoryRecordItem struct {
	EventInformation ec2Stage116EventInformationItem `xml:"eventInformation"`
	EventType        string                          `xml:"eventType,omitempty"`
	Timestamp        string                          `xml:"timestamp,omitempty"`
}

type ec2Stage116EventInformationItem struct {
	EventDescription string `xml:"eventDescription,omitempty"`
	EventSubType     string `xml:"eventSubType,omitempty"`
	InstanceID       string `xml:"instanceId,omitempty"`
}

type ec2Stage116DescribeFleetInstancesResponse struct {
	XMLName           xml.Name                     `xml:"DescribeFleetInstancesResponse"`
	Xmlns             string                       `xml:"xmlns,attr"`
	RequestID         string                       `xml:"requestId"`
	ActiveInstanceSet ec2Stage116ActiveInstanceSet `xml:"activeInstanceSet"`
	FleetID           string                       `xml:"fleetId,omitempty"`
	NextToken         *string                      `xml:"nextToken,omitempty"`
}

type ec2Stage116ActiveInstanceSet struct {
	Items []ec2Stage116ActiveInstanceItem `xml:"item"`
}

type ec2Stage116ActiveInstanceItem struct {
	InstanceHealth        string `xml:"instanceHealth,omitempty"`
	InstanceID            string `xml:"instanceId,omitempty"`
	InstanceType          string `xml:"instanceType,omitempty"`
	SpotInstanceRequestID string `xml:"spotInstanceRequestId,omitempty"`
}

type ec2Stage116DescribeFleetsResponse struct {
	XMLName   xml.Name            `xml:"DescribeFleetsResponse"`
	Xmlns     string              `xml:"xmlns,attr"`
	RequestID string              `xml:"requestId"`
	FleetSet  ec2Stage116FleetSet `xml:"fleetSet"`
	NextToken *string             `xml:"nextToken,omitempty"`
}

type ec2Stage116FleetSet struct {
	Items []ec2Stage116FleetItem `xml:"item"`
}

type ec2Stage116FleetItem struct {
	ActivityStatus    string                                `xml:"activityStatus,omitempty"`
	FleetID           string                                `xml:"fleetId,omitempty"`
	FleetState        string                                `xml:"fleetState,omitempty"`
	FulfilledCapacity float64                               `xml:"fulfilledCapacity,omitempty"`
	FleetInstanceSet  ec2Stage116DescribeFleetsInstancesSet `xml:"fleetInstanceSet"`
	TagSet            ec2TagSet                             `xml:"tagSet"`
}

type ec2Stage116DescribeFleetsInstancesSet struct {
	Items []ec2Stage116DescribeFleetsInstancesItem `xml:"item"`
}

type ec2Stage116DescribeFleetsInstancesItem struct {
	InstanceIDs  ec2StringSet `xml:"instanceIds"`
	InstanceType string       `xml:"instanceType,omitempty"`
	Lifecycle    string       `xml:"lifecycle,omitempty"`
}

type ec2Stage116DescribeFlowLogsResponse struct {
	XMLName    xml.Name              `xml:"DescribeFlowLogsResponse"`
	Xmlns      string                `xml:"xmlns,attr"`
	RequestID  string                `xml:"requestId"`
	FlowLogSet ec2Stage116FlowLogSet `xml:"flowLogSet"`
	NextToken  *string               `xml:"nextToken,omitempty"`
}

type ec2Stage116FlowLogSet struct {
	Items []ec2Stage116FlowLogItem `xml:"item"`
}

type ec2Stage116FlowLogItem struct {
	CreationTime  string    `xml:"creationTime,omitempty"`
	FlowLogID     string    `xml:"flowLogId,omitempty"`
	FlowLogStatus string    `xml:"flowLogStatus,omitempty"`
	ResourceID    string    `xml:"resourceId,omitempty"`
	TagSet        ec2TagSet `xml:"tagSet"`
	TrafficType   string    `xml:"trafficType,omitempty"`
}

type ec2Stage116DescribeFpgaImageAttributeResponse struct {
	XMLName            xml.Name                          `xml:"DescribeFpgaImageAttributeResponse"`
	Xmlns              string                            `xml:"xmlns,attr"`
	RequestID          string                            `xml:"requestId"`
	FpgaImageAttribute ec2Stage116FpgaImageAttributeItem `xml:"fpgaImageAttribute"`
}

type ec2Stage116FpgaImageAttributeItem struct {
	Description     string                        `xml:"description,omitempty"`
	FpgaImageID     string                        `xml:"fpgaImageId,omitempty"`
	LoadPermissions *ec2Stage116LoadPermissionSet `xml:"loadPermissions,omitempty"`
	Name            string                        `xml:"name,omitempty"`
	ProductCodes    *ec2Stage116ProductCodeSet    `xml:"productCodes,omitempty"`
}

type ec2Stage116LoadPermissionSet struct {
	Items []ec2Stage116LoadPermissionItem `xml:"item"`
}

type ec2Stage116LoadPermissionItem struct {
	Group  string `xml:"group,omitempty"`
	UserID string `xml:"userId,omitempty"`
}

type ec2Stage116ProductCodeSet struct {
	Items []ec2Stage116ProductCodeItem `xml:"item"`
}

type ec2Stage116ProductCodeItem struct {
	ProductCode string `xml:"productCode,omitempty"`
	Type        string `xml:"type,omitempty"`
}

type ec2Stage116DescribeFpgaImagesResponse struct {
	XMLName      xml.Name                `xml:"DescribeFpgaImagesResponse"`
	Xmlns        string                  `xml:"xmlns,attr"`
	RequestID    string                  `xml:"requestId"`
	FpgaImageSet ec2Stage116FpgaImageSet `xml:"fpgaImageSet"`
	NextToken    *string                 `xml:"nextToken,omitempty"`
}

type ec2Stage116FpgaImageSet struct {
	Items []ec2Stage116FpgaImageItem `xml:"item"`
}

type ec2Stage116FpgaImageItem struct {
	Description       string                        `xml:"description,omitempty"`
	FpgaImageGlobalID string                        `xml:"fpgaImageGlobalId,omitempty"`
	FpgaImageID       string                        `xml:"fpgaImageId,omitempty"`
	Name              string                        `xml:"name,omitempty"`
	OwnerID           string                        `xml:"ownerId,omitempty"`
	Public            bool                          `xml:"public,omitempty"`
	State             ec2Stage116FpgaImageStateItem `xml:"state"`
	Tags              ec2TagSet                     `xml:"tags"`
}

type ec2Stage116FpgaImageStateItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2Stage116DescribeHostReservationOfferingsResponse struct {
	XMLName     xml.Name                   `xml:"DescribeHostReservationOfferingsResponse"`
	Xmlns       string                     `xml:"xmlns,attr"`
	RequestID   string                     `xml:"requestId"`
	OfferingSet ec2Stage116HostOfferingSet `xml:"offeringSet"`
	NextToken   *string                    `xml:"nextToken,omitempty"`
}

type ec2Stage116HostOfferingSet struct {
	Items []ec2Stage116HostOfferingItem `xml:"item"`
}

type ec2Stage116HostOfferingItem struct {
	CurrencyCode   string `xml:"currencyCode,omitempty"`
	Duration       int32  `xml:"duration,omitempty"`
	HourlyPrice    string `xml:"hourlyPrice,omitempty"`
	InstanceFamily string `xml:"instanceFamily,omitempty"`
	OfferingID     string `xml:"offeringId,omitempty"`
	PaymentOption  string `xml:"paymentOption,omitempty"`
	UpfrontPrice   string `xml:"upfrontPrice,omitempty"`
}

type ec2Stage116DescribeHostReservationsResponse struct {
	XMLName            xml.Name                      `xml:"DescribeHostReservationsResponse"`
	Xmlns              string                        `xml:"xmlns,attr"`
	RequestID          string                        `xml:"requestId"`
	HostReservationSet ec2Stage116HostReservationSet `xml:"hostReservationSet"`
	NextToken          *string                       `xml:"nextToken,omitempty"`
}

type ec2Stage116HostReservationSet struct {
	Items []ec2Stage116HostReservationItem `xml:"item"`
}

type ec2Stage116HostReservationItem struct {
	Count             int32        `xml:"count,omitempty"`
	CurrencyCode      string       `xml:"currencyCode,omitempty"`
	Duration          int32        `xml:"duration,omitempty"`
	End               string       `xml:"end,omitempty"`
	HostIDSet         ec2StringSet `xml:"hostIdSet"`
	HostReservationID string       `xml:"hostReservationId,omitempty"`
	HourlyPrice       string       `xml:"hourlyPrice,omitempty"`
	InstanceFamily    string       `xml:"instanceFamily,omitempty"`
	OfferingID        string       `xml:"offeringId,omitempty"`
	PaymentOption     string       `xml:"paymentOption,omitempty"`
	Start             string       `xml:"start,omitempty"`
	State             string       `xml:"state,omitempty"`
	TagSet            ec2TagSet    `xml:"tagSet"`
	UpfrontPrice      string       `xml:"upfrontPrice,omitempty"`
}
