package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage122Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeSpotFleetRequestHistory":
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
		spotFleetRequestID, historyRecords, lastEvaluatedTime, nextToken, err := s.ec2.DescribeSpotFleetRequestHistory(
			strings.TrimSpace(r.Form.Get("SpotFleetRequestId")),
			startTime,
			strings.TrimSpace(r.Form.Get("EventType")),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage122DescribeSpotFleetRequestHistoryResponse{
			XMLName:            xml.Name{Local: "DescribeSpotFleetRequestHistoryResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			HistoryRecordSet:   ec2Stage116HistoryRecordSet{Items: ec2Stage116HistoryRecordItemsFrom(historyRecords)},
			LastEvaluatedTime:  ec2Stage116TimeString(lastEvaluatedTime),
			NextToken:          nextToken,
			SpotFleetRequestID: spotFleetRequestID,
			StartTime:          startTime.UTC().Format(time.RFC3339),
		})
		return true
	case "DescribeSpotFleetRequests":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		spotFleetRequestConfigs, nextToken, err := s.ec2.DescribeSpotFleetRequests(
			parseEC2MembersOrItemList(r.Form, "SpotFleetRequestId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage122DescribeSpotFleetRequestsResponse{
			XMLName:                   xml.Name{Local: "DescribeSpotFleetRequestsResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			SpotFleetRequestConfigSet: ec2Stage122SpotFleetRequestConfigSet{Items: ec2Stage122SpotFleetRequestConfigItemsFrom(spotFleetRequestConfigs)},
			NextToken:                 nextToken,
		})
		return true
	case "DescribeSpotInstanceRequests":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		spotInstanceRequests, nextToken, err := s.ec2.DescribeSpotInstanceRequests(
			parseEC2MembersOrItemList(r.Form, "SpotInstanceRequestId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage122DescribeSpotInstanceRequestsResponse{
			XMLName:                xml.Name{Local: "DescribeSpotInstanceRequestsResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			SpotInstanceRequestSet: ec2Stage122SpotInstanceRequestSet{Items: ec2Stage122SpotInstanceRequestItemsFrom(spotInstanceRequests)},
			NextToken:              nextToken,
		})
		return true
	case "DescribeSpotPriceHistory":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		startTime, ok := parseEC2Stage122OptionalRFC3339Time(r.Form, "StartTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		endTime, ok := parseEC2Stage122OptionalRFC3339Time(r.Form, "EndTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		spotPriceHistory, nextToken, err := s.ec2.DescribeSpotPriceHistory(
			strings.TrimSpace(r.Form.Get("AvailabilityZone")),
			strings.TrimSpace(r.Form.Get("AvailabilityZoneId")),
			endTime,
			startTime,
			parseEC2MembersOrItemList(r.Form, "InstanceType"),
			parseEC2MembersOrItemList(r.Form, "ProductDescription"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage122DescribeSpotPriceHistoryResponse{
			XMLName:             xml.Name{Local: "DescribeSpotPriceHistoryResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			SpotPriceHistorySet: ec2Stage122SpotPriceSet{Items: ec2Stage122SpotPriceItemsFrom(spotPriceHistory)},
			NextToken:           nextToken,
		})
		return true
	case "DescribeStoreImageTasks":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		storeImageTaskResults, nextToken, err := s.ec2.DescribeStoreImageTasks(
			parseEC2MembersOrItemList(r.Form, "ImageId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage122DescribeStoreImageTasksResponse{
			XMLName:                 xml.Name{Local: "DescribeStoreImageTasksResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			StoreImageTaskResultSet: ec2Stage122StoreImageTaskResultSet{Items: ec2Stage122StoreImageTaskResultItemsFrom(storeImageTaskResults)},
			NextToken:               nextToken,
		})
		return true
	case "DescribeTrafficMirrorFilterRules":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		trafficMirrorFilterRules, nextToken, err := s.ec2.DescribeTrafficMirrorFilterRules(
			strings.TrimSpace(r.Form.Get("TrafficMirrorFilterId")),
			parseEC2MembersOrItemList(r.Form, "TrafficMirrorFilterRuleId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage122DescribeTrafficMirrorFilterRulesResponse{
			XMLName:                    xml.Name{Local: "DescribeTrafficMirrorFilterRulesResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			TrafficMirrorFilterRuleSet: ec2Stage122TrafficMirrorFilterRuleSet{Items: ec2Stage122TrafficMirrorFilterRuleItemsFrom(trafficMirrorFilterRules)},
			NextToken:                  nextToken,
		})
		return true
	case "DescribeTrafficMirrorFilters":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		trafficMirrorFilters, nextToken, err := s.ec2.DescribeTrafficMirrorFilters(
			parseEC2MembersOrItemList(r.Form, "TrafficMirrorFilterId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage122DescribeTrafficMirrorFiltersResponse{
			XMLName:                xml.Name{Local: "DescribeTrafficMirrorFiltersResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			TrafficMirrorFilterSet: ec2Stage122TrafficMirrorFilterSet{Items: ec2Stage122TrafficMirrorFilterItemsFrom(trafficMirrorFilters)},
			NextToken:              nextToken,
		})
		return true
	case "DescribeTrafficMirrorSessions":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		trafficMirrorSessions, nextToken, err := s.ec2.DescribeTrafficMirrorSessions(
			parseEC2MembersOrItemList(r.Form, "TrafficMirrorSessionId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage122DescribeTrafficMirrorSessionsResponse{
			XMLName:                 xml.Name{Local: "DescribeTrafficMirrorSessionsResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			TrafficMirrorSessionSet: ec2Stage122TrafficMirrorSessionSet{Items: ec2Stage122TrafficMirrorSessionItemsFrom(trafficMirrorSessions)},
			NextToken:               nextToken,
		})
		return true
	case "DescribeTrafficMirrorTargets":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		trafficMirrorTargets, nextToken, err := s.ec2.DescribeTrafficMirrorTargets(
			parseEC2MembersOrItemList(r.Form, "TrafficMirrorTargetId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage122DescribeTrafficMirrorTargetsResponse{
			XMLName:                xml.Name{Local: "DescribeTrafficMirrorTargetsResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			TrafficMirrorTargetSet: ec2Stage122TrafficMirrorTargetSet{Items: ec2Stage122TrafficMirrorTargetItemsFrom(trafficMirrorTargets)},
			NextToken:              nextToken,
		})
		return true
	case "DescribeTrunkInterfaceAssociations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		interfaceAssociations, nextToken, err := s.ec2.DescribeTrunkInterfaceAssociations(
			parseEC2MembersOrItemList(r.Form, "AssociationId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage122DescribeTrunkInterfaceAssociationsResponse{
			XMLName:                 xml.Name{Local: "DescribeTrunkInterfaceAssociationsResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			InterfaceAssociationSet: ec2Stage122TrunkInterfaceAssociationSet{Items: ec2Stage122TrunkInterfaceAssociationItemsFrom(interfaceAssociations)},
			NextToken:               nextToken,
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage122OptionalRFC3339Time(values url.Values, key string) (*time.Time, bool) {
	if !hasEC2Field(values, key) {
		return nil, true
	}
	value := strings.TrimSpace(values.Get(key))
	if value == "" {
		return nil, true
	}
	parsed, err := parseEC2RFC3339Time(value)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func ec2Stage122SpotFleetRequestConfigItemsFrom(in []ec2svc.SpotFleetRequestConfig) []ec2Stage122SpotFleetRequestConfigItem {
	out := make([]ec2Stage122SpotFleetRequestConfigItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage122SpotFleetRequestConfigItem{
			ActivityStatus:        item.ActivityStatus,
			CreateTime:            item.CreateTime.UTC().Format(time.RFC3339),
			SpotFleetRequestID:    item.SpotFleetRequestID,
			SpotFleetRequestState: item.SpotFleetRequestState,
			TagSet:                ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage122SpotInstanceRequestItemsFrom(in []ec2svc.SpotInstanceRequest) []ec2Stage122SpotInstanceRequestItem {
	out := make([]ec2Stage122SpotInstanceRequestItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage122SpotInstanceRequestItem{
			AvailabilityZoneGroup: item.AvailabilityZoneGroup,
			CreateTime:            item.CreateTime.UTC().Format(time.RFC3339),
			InstanceID:            item.InstanceID,
			LaunchGroup:           item.LaunchGroup,
			ProductDescription:    item.ProductDescription,
			SpotInstanceRequestID: item.SpotInstanceRequestID,
			SpotPrice:             item.SpotPrice,
			State:                 item.State,
			Status: ec2Stage122SpotInstanceStatusItem{
				Code:       item.Status.Code,
				Message:    item.Status.Message,
				UpdateTime: item.Status.UpdateTime.UTC().Format(time.RFC3339),
			},
			TagSet: ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			Type:   item.Type,
		})
	}
	return out
}

func ec2Stage122SpotPriceItemsFrom(in []ec2svc.SpotPriceHistoryItem) []ec2Stage122SpotPriceItem {
	out := make([]ec2Stage122SpotPriceItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage122SpotPriceItem{
			AvailabilityZone:   item.AvailabilityZone,
			InstanceType:       item.InstanceType,
			ProductDescription: item.ProductDescription,
			SpotPrice:          item.SpotPrice,
			Timestamp:          item.Timestamp.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func ec2Stage122StoreImageTaskResultItemsFrom(in []ec2svc.StoreImageTaskResult) []ec2Stage122StoreImageTaskResultItem {
	out := make([]ec2Stage122StoreImageTaskResultItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage122StoreImageTaskResultItem{
			AmiID:                  item.AmiID,
			Bucket:                 item.Bucket,
			ProgressPercentage:     &item.ProgressPercentage,
			S3ObjectKey:            item.S3ObjectKey,
			StoreTaskFailureReason: item.StoreTaskFailureReason,
			StoreTaskState:         item.StoreTaskState,
			TaskStartTime:          item.TaskStartTime.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func ec2Stage122TrafficMirrorFilterRuleItemsFrom(in []ec2svc.TrafficMirrorFilterRule) []ec2Stage110TrafficMirrorFilterRuleItem {
	out := make([]ec2Stage110TrafficMirrorFilterRuleItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage110TrafficMirrorFilterRuleItemFrom(item))
	}
	return out
}

func ec2Stage122TrafficMirrorFilterItemsFrom(in []ec2svc.TrafficMirrorFilter) []ec2Stage110TrafficMirrorFilterItem {
	out := make([]ec2Stage110TrafficMirrorFilterItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage110TrafficMirrorFilterItemFrom(item))
	}
	return out
}

func ec2Stage122TrafficMirrorSessionItemsFrom(in []ec2svc.TrafficMirrorSession) []ec2Stage110TrafficMirrorSessionItem {
	out := make([]ec2Stage110TrafficMirrorSessionItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage110TrafficMirrorSessionItemFrom(item))
	}
	return out
}

func ec2Stage122TrafficMirrorTargetItemsFrom(in []ec2svc.TrafficMirrorTarget) []ec2Stage110TrafficMirrorTargetItem {
	out := make([]ec2Stage110TrafficMirrorTargetItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage110TrafficMirrorTargetItemFrom(item))
	}
	return out
}

func ec2Stage122TrunkInterfaceAssociationItemsFrom(in []ec2svc.TrunkInterfaceAssociation) []ec2TrunkInterfaceAssociationItem {
	out := make([]ec2TrunkInterfaceAssociationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2TrunkInterfaceAssociationItemFrom(item))
	}
	return out
}

type ec2Stage122DescribeSpotFleetRequestHistoryResponse struct {
	XMLName            xml.Name                    `xml:"DescribeSpotFleetRequestHistoryResponse"`
	Xmlns              string                      `xml:"xmlns,attr"`
	RequestID          string                      `xml:"requestId"`
	HistoryRecordSet   ec2Stage116HistoryRecordSet `xml:"historyRecordSet"`
	LastEvaluatedTime  string                      `xml:"lastEvaluatedTime,omitempty"`
	NextToken          *string                     `xml:"nextToken,omitempty"`
	SpotFleetRequestID string                      `xml:"spotFleetRequestId,omitempty"`
	StartTime          string                      `xml:"startTime,omitempty"`
}

type ec2Stage122DescribeSpotFleetRequestsResponse struct {
	XMLName                   xml.Name                             `xml:"DescribeSpotFleetRequestsResponse"`
	Xmlns                     string                               `xml:"xmlns,attr"`
	RequestID                 string                               `xml:"requestId"`
	SpotFleetRequestConfigSet ec2Stage122SpotFleetRequestConfigSet `xml:"spotFleetRequestConfigSet"`
	NextToken                 *string                              `xml:"nextToken,omitempty"`
}

type ec2Stage122SpotFleetRequestConfigSet struct {
	Items []ec2Stage122SpotFleetRequestConfigItem `xml:"item"`
}

type ec2Stage122SpotFleetRequestConfigItem struct {
	ActivityStatus        string    `xml:"activityStatus,omitempty"`
	CreateTime            string    `xml:"createTime,omitempty"`
	SpotFleetRequestID    string    `xml:"spotFleetRequestId,omitempty"`
	SpotFleetRequestState string    `xml:"spotFleetRequestState,omitempty"`
	TagSet                ec2TagSet `xml:"tagSet"`
}

type ec2Stage122DescribeSpotInstanceRequestsResponse struct {
	XMLName                xml.Name                          `xml:"DescribeSpotInstanceRequestsResponse"`
	Xmlns                  string                            `xml:"xmlns,attr"`
	RequestID              string                            `xml:"requestId"`
	SpotInstanceRequestSet ec2Stage122SpotInstanceRequestSet `xml:"spotInstanceRequestSet"`
	NextToken              *string                           `xml:"nextToken,omitempty"`
}

type ec2Stage122SpotInstanceRequestSet struct {
	Items []ec2Stage122SpotInstanceRequestItem `xml:"item"`
}

type ec2Stage122SpotInstanceRequestItem struct {
	AvailabilityZoneGroup string                            `xml:"availabilityZoneGroup,omitempty"`
	CreateTime            string                            `xml:"createTime,omitempty"`
	InstanceID            string                            `xml:"instanceId,omitempty"`
	LaunchGroup           string                            `xml:"launchGroup,omitempty"`
	ProductDescription    string                            `xml:"productDescription,omitempty"`
	SpotInstanceRequestID string                            `xml:"spotInstanceRequestId,omitempty"`
	SpotPrice             string                            `xml:"spotPrice,omitempty"`
	State                 string                            `xml:"state,omitempty"`
	Status                ec2Stage122SpotInstanceStatusItem `xml:"status"`
	TagSet                ec2TagSet                         `xml:"tagSet"`
	Type                  string                            `xml:"type,omitempty"`
}

type ec2Stage122SpotInstanceStatusItem struct {
	Code       string `xml:"code,omitempty"`
	Message    string `xml:"message,omitempty"`
	UpdateTime string `xml:"updateTime,omitempty"`
}

type ec2Stage122DescribeSpotPriceHistoryResponse struct {
	XMLName             xml.Name                `xml:"DescribeSpotPriceHistoryResponse"`
	Xmlns               string                  `xml:"xmlns,attr"`
	RequestID           string                  `xml:"requestId"`
	SpotPriceHistorySet ec2Stage122SpotPriceSet `xml:"spotPriceHistorySet"`
	NextToken           *string                 `xml:"nextToken,omitempty"`
}

type ec2Stage122SpotPriceSet struct {
	Items []ec2Stage122SpotPriceItem `xml:"item"`
}

type ec2Stage122SpotPriceItem struct {
	AvailabilityZone   string `xml:"availabilityZone,omitempty"`
	InstanceType       string `xml:"instanceType,omitempty"`
	ProductDescription string `xml:"productDescription,omitempty"`
	SpotPrice          string `xml:"spotPrice,omitempty"`
	Timestamp          string `xml:"timestamp,omitempty"`
}

type ec2Stage122DescribeStoreImageTasksResponse struct {
	XMLName                 xml.Name                           `xml:"DescribeStoreImageTasksResponse"`
	Xmlns                   string                             `xml:"xmlns,attr"`
	RequestID               string                             `xml:"requestId"`
	StoreImageTaskResultSet ec2Stage122StoreImageTaskResultSet `xml:"storeImageTaskResultSet"`
	NextToken               *string                            `xml:"nextToken,omitempty"`
}

type ec2Stage122StoreImageTaskResultSet struct {
	Items []ec2Stage122StoreImageTaskResultItem `xml:"item"`
}

type ec2Stage122StoreImageTaskResultItem struct {
	AmiID                  string `xml:"amiId,omitempty"`
	Bucket                 string `xml:"bucket,omitempty"`
	ProgressPercentage     *int32 `xml:"progressPercentage,omitempty"`
	S3ObjectKey            string `xml:"s3objectKey,omitempty"`
	StoreTaskFailureReason string `xml:"storeTaskFailureReason,omitempty"`
	StoreTaskState         string `xml:"storeTaskState,omitempty"`
	TaskStartTime          string `xml:"taskStartTime,omitempty"`
}

type ec2Stage122DescribeTrafficMirrorFilterRulesResponse struct {
	XMLName                    xml.Name                              `xml:"DescribeTrafficMirrorFilterRulesResponse"`
	Xmlns                      string                                `xml:"xmlns,attr"`
	RequestID                  string                                `xml:"requestId"`
	TrafficMirrorFilterRuleSet ec2Stage122TrafficMirrorFilterRuleSet `xml:"trafficMirrorFilterRuleSet"`
	NextToken                  *string                               `xml:"nextToken,omitempty"`
}

type ec2Stage122TrafficMirrorFilterRuleSet struct {
	Items []ec2Stage110TrafficMirrorFilterRuleItem `xml:"item"`
}

type ec2Stage122DescribeTrafficMirrorFiltersResponse struct {
	XMLName                xml.Name                          `xml:"DescribeTrafficMirrorFiltersResponse"`
	Xmlns                  string                            `xml:"xmlns,attr"`
	RequestID              string                            `xml:"requestId"`
	TrafficMirrorFilterSet ec2Stage122TrafficMirrorFilterSet `xml:"trafficMirrorFilterSet"`
	NextToken              *string                           `xml:"nextToken,omitempty"`
}

type ec2Stage122TrafficMirrorFilterSet struct {
	Items []ec2Stage110TrafficMirrorFilterItem `xml:"item"`
}

type ec2Stage122DescribeTrafficMirrorSessionsResponse struct {
	XMLName                 xml.Name                           `xml:"DescribeTrafficMirrorSessionsResponse"`
	Xmlns                   string                             `xml:"xmlns,attr"`
	RequestID               string                             `xml:"requestId"`
	TrafficMirrorSessionSet ec2Stage122TrafficMirrorSessionSet `xml:"trafficMirrorSessionSet"`
	NextToken               *string                            `xml:"nextToken,omitempty"`
}

type ec2Stage122TrafficMirrorSessionSet struct {
	Items []ec2Stage110TrafficMirrorSessionItem `xml:"item"`
}

type ec2Stage122DescribeTrafficMirrorTargetsResponse struct {
	XMLName                xml.Name                          `xml:"DescribeTrafficMirrorTargetsResponse"`
	Xmlns                  string                            `xml:"xmlns,attr"`
	RequestID              string                            `xml:"requestId"`
	TrafficMirrorTargetSet ec2Stage122TrafficMirrorTargetSet `xml:"trafficMirrorTargetSet"`
	NextToken              *string                           `xml:"nextToken,omitempty"`
}

type ec2Stage122TrafficMirrorTargetSet struct {
	Items []ec2Stage110TrafficMirrorTargetItem `xml:"item"`
}

type ec2Stage122DescribeTrunkInterfaceAssociationsResponse struct {
	XMLName                 xml.Name                                `xml:"DescribeTrunkInterfaceAssociationsResponse"`
	Xmlns                   string                                  `xml:"xmlns,attr"`
	RequestID               string                                  `xml:"requestId"`
	InterfaceAssociationSet ec2Stage122TrunkInterfaceAssociationSet `xml:"interfaceAssociationSet"`
	NextToken               *string                                 `xml:"nextToken,omitempty"`
}

type ec2Stage122TrunkInterfaceAssociationSet struct {
	Items []ec2TrunkInterfaceAssociationItem `xml:"item"`
}
