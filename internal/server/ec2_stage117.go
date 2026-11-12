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

func (s *Server) handleEC2Stage117Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeHosts":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		hosts, nextToken, err := s.ec2.DescribeHosts(
			parseEC2MembersOrItemList(r.Form, "HostId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage117DescribeHostsResponse{
			XMLName:   xml.Name{Local: "DescribeHostsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			HostSet:   ec2Stage117HostSet{Items: ec2Stage117HostItemsFrom(hosts)},
			NextToken: nextToken,
		})
		return true
	case "DescribeImportImageTasks":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		importImageTasks, nextToken, err := s.ec2.DescribeImportImageTasks(
			parseEC2MembersOrItemList(r.Form, "ImportTaskId"),
			parseEC2Stage117Filters(r.Form, "Filters.", "Filters.Filter.", "Filter."),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage117DescribeImportImageTasksResponse{
			XMLName:            xml.Name{Local: "DescribeImportImageTasksResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			ImportImageTaskSet: ec2Stage117ImportImageTaskSet{Items: ec2Stage117ImportImageTaskItemsFrom(importImageTasks)},
			NextToken:          nextToken,
		})
		return true
	case "DescribeImportSnapshotTasks":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		importSnapshotTasks, nextToken, err := s.ec2.DescribeImportSnapshotTasks(
			parseEC2MembersOrItemList(r.Form, "ImportTaskId"),
			parseEC2Stage117Filters(r.Form, "Filters.", "Filters.Filter.", "Filter."),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage117DescribeImportSnapshotTasksResponse{
			XMLName:               xml.Name{Local: "DescribeImportSnapshotTasksResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			ImportSnapshotTaskSet: ec2Stage117ImportSnapshotTaskSet{Items: ec2Stage117ImportSnapshotTaskItemsFrom(importSnapshotTasks)},
			NextToken:             nextToken,
		})
		return true
	case "DescribeInstanceConnectEndpoints":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		endpoints, nextToken, err := s.ec2.DescribeInstanceConnectEndpoints(
			parseEC2MembersOrItemList(r.Form, "InstanceConnectEndpointId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage117DescribeInstanceConnectEndpointsResponse{
			XMLName:                    xml.Name{Local: "DescribeInstanceConnectEndpointsResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			InstanceConnectEndpointSet: ec2Stage117InstanceConnectEndpointSet{Items: ec2Stage117InstanceConnectEndpointItemsFrom(endpoints)},
			NextToken:                  nextToken,
		})
		return true
	case "DescribeInstanceCreditSpecifications":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		specifications, nextToken, err := s.ec2.DescribeInstanceCreditSpecifications(
			parseEC2MembersOrItemList(r.Form, "InstanceId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage117DescribeInstanceCreditSpecificationsResponse{
			XMLName:                        xml.Name{Local: "DescribeInstanceCreditSpecificationsResponse"},
			Xmlns:                          ec2Namespace,
			RequestID:                      "stackyard-request",
			InstanceCreditSpecificationSet: ec2Stage117InstanceCreditSpecificationSet{Items: ec2Stage117InstanceCreditSpecificationItemsFrom(specifications)},
			NextToken:                      nextToken,
		})
		return true
	case "DescribeInstanceEventNotificationAttributes":
		attrs := s.ec2.DescribeInstanceEventNotificationAttributes()
		respondEC2XML(w, ec2Stage117DescribeInstanceEventNotificationAttributesResponse{
			XMLName:              xml.Name{Local: "DescribeInstanceEventNotificationAttributesResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			InstanceTagAttribute: ec2Stage117InstanceTagNotificationAttributeItemFrom(attrs),
		})
		return true
	case "DescribeInstanceEventWindows":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		windows, nextToken, err := s.ec2.DescribeInstanceEventWindows(
			parseEC2MembersOrItemList(r.Form, "InstanceEventWindowId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage117DescribeInstanceEventWindowsResponse{
			XMLName:                xml.Name{Local: "DescribeInstanceEventWindowsResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			InstanceEventWindowSet: ec2Stage117InstanceEventWindowSet{Items: ec2Stage117InstanceEventWindowItemsFrom(windows)},
			NextToken:              nextToken,
		})
		return true
	case "DescribeInstanceImageMetadata":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		metadata, nextToken, err := s.ec2.DescribeInstanceImageMetadata(
			parseEC2MembersOrItemList(r.Form, "InstanceId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage117DescribeInstanceImageMetadataResponse{
			XMLName:                  xml.Name{Local: "DescribeInstanceImageMetadataResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			InstanceImageMetadataSet: ec2Stage117InstanceImageMetadataSet{Items: ec2Stage117InstanceImageMetadataItemsFrom(metadata)},
			NextToken:                nextToken,
		})
		return true
	case "DescribeInstanceTopology":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		topology, nextToken, err := s.ec2.DescribeInstanceTopology(
			parseEC2MembersOrItemList(r.Form, "InstanceId"),
			parseEC2MembersOrItemList(r.Form, "GroupName"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage117DescribeInstanceTopologyResponse{
			XMLName:     xml.Name{Local: "DescribeInstanceTopologyResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			InstanceSet: ec2Stage117InstanceTopologySet{Items: ec2Stage117InstanceTopologyItemsFrom(topology)},
			NextToken:   nextToken,
		})
		return true
	case "DescribeInstanceTypeOfferings":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		offerings, nextToken, err := s.ec2.DescribeInstanceTypeOfferings(
			strings.TrimSpace(r.Form.Get("LocationType")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage117DescribeInstanceTypeOfferingsResponse{
			XMLName:                 xml.Name{Local: "DescribeInstanceTypeOfferingsResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			InstanceTypeOfferingSet: ec2Stage117InstanceTypeOfferingSet{Items: ec2Stage117InstanceTypeOfferingItemsFrom(offerings)},
			NextToken:               nextToken,
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage117Filters(values url.Values, prefixes ...string) map[string][]string {
	if len(prefixes) == 0 {
		prefixes = []string{"Filter."}
	}

	out := map[string][]string{}
	for _, prefix := range prefixes {
		parsed := parseEC2Stage117FiltersWithPrefix(values, prefix)
		for key, parsedValues := range parsed {
			out[key] = append(out[key], parsedValues...)
		}
	}
	for key, parsedValues := range out {
		out[key] = dedupeTrimmedStrings(parsedValues)
	}
	return out
}

func parseEC2Stage117FiltersWithPrefix(values url.Values, prefix string) map[string][]string {
	indexByName := map[int]string{}
	for key := range values {
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, ".Name") {
			continue
		}
		indexText := strings.TrimPrefix(key, prefix)
		indexText = strings.TrimSuffix(indexText, ".Name")
		index, err := strconv.Atoi(indexText)
		if err != nil || index <= 0 {
			continue
		}
		name := strings.TrimSpace(values.Get(key))
		if name == "" {
			continue
		}
		indexByName[index] = name
	}

	ordered := make([]int, 0, len(indexByName))
	for index := range indexByName {
		ordered = append(ordered, index)
	}
	sort.Ints(ordered)

	out := map[string][]string{}
	for _, index := range ordered {
		name := indexByName[index]
		valuePrefix := prefix + strconv.Itoa(index) + ".Value."
		out[name] = append(out[name], parseEC2Members(values, valuePrefix)...)
	}
	return out
}

func dedupeTrimmedStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func ec2Stage117HostItemsFrom(in []ec2svc.HostDescription) []ec2Stage117HostItem {
	out := make([]ec2Stage117HostItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage117HostItem{
			AllocationTime:     ec2Stage115RFC3339(item.AllocationTime),
			AutoPlacement:      item.AutoPlacement,
			AvailabilityZone:   item.AvailabilityZone,
			AvailabilityZoneID: item.AvailabilityZoneID,
			HostID:             item.HostID,
			HostMaintenance:    item.HostMaintenance,
			HostProperties: ec2Stage117HostPropertiesItem{
				Cores:          item.Cores,
				InstanceFamily: item.InstanceFamily,
				InstanceType:   item.InstanceType,
				Sockets:        item.Sockets,
				TotalVCpus:     item.TotalVCpus,
			},
			HostRecovery: item.HostRecovery,
			Instances:    ec2Stage117HostInstanceSet{Items: ec2Stage117HostInstanceItemsFrom(item.Instances)},
			OwnerID:      item.OwnerID,
			State:        item.State,
			TagSet:       ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage117HostInstanceItemsFrom(in []ec2svc.HostInstanceDescription) []ec2Stage117HostInstanceItem {
	out := make([]ec2Stage117HostInstanceItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage117HostInstanceItem{
			InstanceID:   item.InstanceID,
			InstanceType: item.InstanceType,
			OwnerID:      item.OwnerID,
		})
	}
	return out
}

func ec2Stage117ImportImageTaskItemsFrom(in []ec2svc.ImportImageTaskDescription) []ec2Stage117ImportImageTaskItem {
	out := make([]ec2Stage117ImportImageTaskItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage117ImportImageTaskItem{
			Description:       item.Description,
			ImageID:           item.ImageID,
			ImportTaskID:      item.ImportTaskID,
			Progress:          item.Progress,
			SnapshotDetailSet: ec2Stage117ImportImageSnapshotDetailSet{Items: ec2Stage117ImportImageSnapshotDetailItemsFrom(item.SnapshotDetails)},
			Status:            item.Status,
			StatusMessage:     item.StatusMessage,
			TagSet:            ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage117ImportImageSnapshotDetailItemsFrom(in []ec2svc.ImportImageSnapshotDetail) []ec2Stage117ImportImageSnapshotDetailItem {
	out := make([]ec2Stage117ImportImageSnapshotDetailItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage117ImportImageSnapshotDetailItem{
			Progress:   item.Progress,
			SnapshotID: item.SnapshotID,
			Status:     item.Status,
		})
	}
	return out
}

func ec2Stage117ImportSnapshotTaskItemsFrom(in []ec2svc.ImportSnapshotTaskDescription) []ec2Stage117ImportSnapshotTaskItem {
	out := make([]ec2Stage117ImportSnapshotTaskItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage117ImportSnapshotTaskItem{
			Description:  item.Description,
			ImportTaskID: item.ImportTaskID,
			SnapshotTaskDetail: ec2Stage117SnapshotTaskDetailItem{
				Description:   item.SnapshotTaskDetail.Description,
				Progress:      item.SnapshotTaskDetail.Progress,
				SnapshotID:    item.SnapshotTaskDetail.SnapshotID,
				Status:        item.SnapshotTaskDetail.Status,
				StatusMessage: item.SnapshotTaskDetail.StatusMessage,
			},
			TagSet: ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage117InstanceConnectEndpointItemsFrom(in []ec2svc.InstanceConnectEndpoint) []ec2Stage117InstanceConnectEndpointItem {
	out := make([]ec2Stage117InstanceConnectEndpointItem, 0, len(in))
	for _, item := range in {
		endpointID := item.InstanceConnectEndpointID
		out = append(out, ec2Stage117InstanceConnectEndpointItem{
			AvailabilityZone:           item.AvailabilityZone,
			CreatedAt:                  ec2Stage115RFC3339(item.CreatedAt),
			DNSName:                    item.DNSName,
			FipsDNSName:                item.FipsDNSName,
			InstanceConnectEndpointARN: item.InstanceConnectEndpointARN,
			InstanceConnectEndpointID:  endpointID,
			NetworkInterfaceIDSet:      ec2StringSet{Items: []string{ec2Stage117EndpointNetworkInterfaceID(endpointID)}},
			OwnerID:                    item.OwnerID,
			PreserveClientIP:           &item.PreserveClientIP,
			SecurityGroupIDSet:         ec2StringSet{Items: append([]string(nil), item.SecurityGroupIDs...)},
			State:                      item.State,
			StateMessage:               item.StateMessage,
			SubnetID:                   item.SubnetID,
			TagSet:                     ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			VpcID:                      item.VpcID,
		})
	}
	return out
}

func ec2Stage117EndpointNetworkInterfaceID(endpointID string) string {
	endpointID = strings.TrimSpace(strings.TrimPrefix(endpointID, "eice-"))
	if endpointID == "" {
		return "eni-00000000117"
	}
	if len(endpointID) > 8 {
		endpointID = endpointID[len(endpointID)-8:]
	}
	return "eni-" + endpointID
}

func ec2Stage117InstanceCreditSpecificationItemsFrom(in []ec2svc.InstanceCreditSpecificationDescription) []ec2Stage117InstanceCreditSpecificationItem {
	out := make([]ec2Stage117InstanceCreditSpecificationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage117InstanceCreditSpecificationItem{
			CpuCredits: item.CpuCredits,
			InstanceID: item.InstanceID,
		})
	}
	return out
}

func ec2Stage117InstanceTagNotificationAttributeItemFrom(in ec2svc.InstanceTagNotificationAttribute) ec2Stage117InstanceTagNotificationAttributeItem {
	return ec2Stage117InstanceTagNotificationAttributeItem{
		IncludeAllTagsOfInstance: in.IncludeAllTagsOfInstance,
		InstanceTagKeySet:        ec2StringSet{Items: append([]string(nil), in.InstanceTagKeys...)},
	}
}

func ec2Stage117InstanceEventWindowItemsFrom(in []ec2svc.InstanceEventWindowDescription) []ec2Stage117InstanceEventWindowItem {
	out := make([]ec2Stage117InstanceEventWindowItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage117InstanceEventWindowItem{
			AssociationTarget: ec2Stage117InstanceEventWindowAssociationTargetItem{
				DedicatedHostIDSet: ec2StringSet{Items: append([]string(nil), item.AssociationDedicatedHostIDs...)},
				InstanceIDSet:      ec2StringSet{Items: append([]string(nil), item.AssociationInstanceIDs...)},
				TagSet:             ec2TagSet{Items: ec2Stage117TagItemsFromEC2Tags(item.AssociationInstanceTags)},
			},
			CronExpression:        item.CronExpression,
			InstanceEventWindowID: item.InstanceEventWindowID,
			Name:                  item.Name,
			State:                 item.State,
			TagSet:                ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			TimeRangeSet:          ec2Stage117InstanceEventWindowTimeRangeSet{Items: ec2Stage117InstanceEventWindowTimeRangeItemsFrom(item.TimeRanges)},
		})
	}
	return out
}

func ec2Stage117TagItemsFromEC2Tags(in []ec2svc.Tag) []ec2TagItem {
	out := make([]ec2TagItem, 0, len(in))
	for _, item := range in {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		out = append(out, ec2TagItem{
			Key:   key,
			Value: strings.TrimSpace(item.Value),
		})
	}
	return out
}

func ec2Stage117InstanceEventWindowTimeRangeItemsFrom(in []ec2svc.InstanceEventWindowTimeRange) []ec2Stage117InstanceEventWindowTimeRangeItem {
	out := make([]ec2Stage117InstanceEventWindowTimeRangeItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage117InstanceEventWindowTimeRangeItem{
			EndHour:      item.EndHour,
			EndWeekDay:   item.EndWeekDay,
			StartHour:    item.StartHour,
			StartWeekDay: item.StartWeekDay,
		})
	}
	return out
}

func ec2Stage117InstanceImageMetadataItemsFrom(in []ec2svc.InstanceImageMetadataDescription) []ec2Stage117InstanceImageMetadataItem {
	out := make([]ec2Stage117InstanceImageMetadataItem, 0, len(in))
	for _, item := range in {
		imageAllowed := item.ImageMetadata.ImageAllowed
		isPublic := item.ImageMetadata.IsPublic

		out = append(out, ec2Stage117InstanceImageMetadataItem{
			AvailabilityZone: item.AvailabilityZone,
			ImageMetadata: ec2Stage117ImageMetadataItem{
				CreationDate:    item.ImageMetadata.CreationDate,
				DeprecationTime: item.ImageMetadata.DeprecationTime,
				ImageAllowed:    &imageAllowed,
				ImageID:         item.ImageMetadata.ImageID,
				ImageOwnerAlias: item.ImageMetadata.ImageOwnerAlias,
				IsPublic:        &isPublic,
				Name:            item.ImageMetadata.Name,
				ImageOwnerID:    item.ImageMetadata.OwnerID,
				ImageState:      item.ImageMetadata.State,
			},
			InstanceID:      item.InstanceID,
			InstanceType:    item.InstanceType,
			LaunchTime:      ec2Stage115RFC3339(item.LaunchTime),
			InstanceOwnerID: item.OwnerID,
			InstanceState: ec2InstanceStateItem{
				Code: item.StateCode,
				Name: item.StateName,
			},
			TagSet: ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			ZoneID: item.ZoneID,
		})
	}
	return out
}

func ec2Stage117InstanceTopologyItemsFrom(in []ec2svc.InstanceTopologyDescription) []ec2Stage117InstanceTopologyItem {
	out := make([]ec2Stage117InstanceTopologyItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage117InstanceTopologyItem{
			AvailabilityZone: item.AvailabilityZone,
			CapacityBlockID:  item.CapacityBlockID,
			GroupName:        item.GroupName,
			InstanceID:       item.InstanceID,
			InstanceType:     item.InstanceType,
			NetworkNodeSet:   ec2StringSet{Items: append([]string(nil), item.NetworkNodes...)},
			ZoneID:           item.ZoneID,
		})
	}
	return out
}

func ec2Stage117InstanceTypeOfferingItemsFrom(in []ec2svc.InstanceTypeOfferingDescription) []ec2Stage117InstanceTypeOfferingItem {
	out := make([]ec2Stage117InstanceTypeOfferingItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage117InstanceTypeOfferingItem{
			InstanceType: item.InstanceType,
			Location:     item.Location,
			LocationType: item.LocationType,
		})
	}
	return out
}

type ec2Stage117DescribeHostsResponse struct {
	XMLName   xml.Name           `xml:"DescribeHostsResponse"`
	Xmlns     string             `xml:"xmlns,attr"`
	RequestID string             `xml:"requestId"`
	HostSet   ec2Stage117HostSet `xml:"hostSet"`
	NextToken *string            `xml:"nextToken,omitempty"`
}

type ec2Stage117HostSet struct {
	Items []ec2Stage117HostItem `xml:"item"`
}

type ec2Stage117HostItem struct {
	AllocationTime     string                        `xml:"allocationTime,omitempty"`
	AutoPlacement      string                        `xml:"autoPlacement,omitempty"`
	AvailabilityZone   string                        `xml:"availabilityZone,omitempty"`
	AvailabilityZoneID string                        `xml:"availabilityZoneId,omitempty"`
	HostID             string                        `xml:"hostId,omitempty"`
	HostMaintenance    string                        `xml:"hostMaintenance,omitempty"`
	HostProperties     ec2Stage117HostPropertiesItem `xml:"hostProperties"`
	HostRecovery       string                        `xml:"hostRecovery,omitempty"`
	Instances          ec2Stage117HostInstanceSet    `xml:"instances"`
	OwnerID            string                        `xml:"ownerId,omitempty"`
	State              string                        `xml:"state,omitempty"`
	TagSet             ec2TagSet                     `xml:"tagSet"`
}

type ec2Stage117HostPropertiesItem struct {
	Cores          int32  `xml:"cores,omitempty"`
	InstanceFamily string `xml:"instanceFamily,omitempty"`
	InstanceType   string `xml:"instanceType,omitempty"`
	Sockets        int32  `xml:"sockets,omitempty"`
	TotalVCpus     int32  `xml:"totalVCpus,omitempty"`
}

type ec2Stage117HostInstanceSet struct {
	Items []ec2Stage117HostInstanceItem `xml:"item"`
}

type ec2Stage117HostInstanceItem struct {
	InstanceID   string `xml:"instanceId,omitempty"`
	InstanceType string `xml:"instanceType,omitempty"`
	OwnerID      string `xml:"ownerId,omitempty"`
}

type ec2Stage117DescribeImportImageTasksResponse struct {
	XMLName            xml.Name                      `xml:"DescribeImportImageTasksResponse"`
	Xmlns              string                        `xml:"xmlns,attr"`
	RequestID          string                        `xml:"requestId"`
	ImportImageTaskSet ec2Stage117ImportImageTaskSet `xml:"importImageTaskSet"`
	NextToken          *string                       `xml:"nextToken,omitempty"`
}

type ec2Stage117ImportImageTaskSet struct {
	Items []ec2Stage117ImportImageTaskItem `xml:"item"`
}

type ec2Stage117ImportImageTaskItem struct {
	Description       string                                  `xml:"description,omitempty"`
	ImageID           string                                  `xml:"imageId,omitempty"`
	ImportTaskID      string                                  `xml:"importTaskId,omitempty"`
	Progress          string                                  `xml:"progress,omitempty"`
	SnapshotDetailSet ec2Stage117ImportImageSnapshotDetailSet `xml:"snapshotDetailSet"`
	Status            string                                  `xml:"status,omitempty"`
	StatusMessage     string                                  `xml:"statusMessage,omitempty"`
	TagSet            ec2TagSet                               `xml:"tagSet"`
}

type ec2Stage117ImportImageSnapshotDetailSet struct {
	Items []ec2Stage117ImportImageSnapshotDetailItem `xml:"item"`
}

type ec2Stage117ImportImageSnapshotDetailItem struct {
	Progress   string `xml:"progress,omitempty"`
	SnapshotID string `xml:"snapshotId,omitempty"`
	Status     string `xml:"status,omitempty"`
}

type ec2Stage117DescribeImportSnapshotTasksResponse struct {
	XMLName               xml.Name                         `xml:"DescribeImportSnapshotTasksResponse"`
	Xmlns                 string                           `xml:"xmlns,attr"`
	RequestID             string                           `xml:"requestId"`
	ImportSnapshotTaskSet ec2Stage117ImportSnapshotTaskSet `xml:"importSnapshotTaskSet"`
	NextToken             *string                          `xml:"nextToken,omitempty"`
}

type ec2Stage117ImportSnapshotTaskSet struct {
	Items []ec2Stage117ImportSnapshotTaskItem `xml:"item"`
}

type ec2Stage117ImportSnapshotTaskItem struct {
	Description        string                            `xml:"description,omitempty"`
	ImportTaskID       string                            `xml:"importTaskId,omitempty"`
	SnapshotTaskDetail ec2Stage117SnapshotTaskDetailItem `xml:"snapshotTaskDetail"`
	TagSet             ec2TagSet                         `xml:"tagSet"`
}

type ec2Stage117SnapshotTaskDetailItem struct {
	Description   string `xml:"description,omitempty"`
	Progress      string `xml:"progress,omitempty"`
	SnapshotID    string `xml:"snapshotId,omitempty"`
	Status        string `xml:"status,omitempty"`
	StatusMessage string `xml:"statusMessage,omitempty"`
}

type ec2Stage117DescribeInstanceConnectEndpointsResponse struct {
	XMLName                    xml.Name                              `xml:"DescribeInstanceConnectEndpointsResponse"`
	Xmlns                      string                                `xml:"xmlns,attr"`
	RequestID                  string                                `xml:"requestId"`
	InstanceConnectEndpointSet ec2Stage117InstanceConnectEndpointSet `xml:"instanceConnectEndpointSet"`
	NextToken                  *string                               `xml:"nextToken,omitempty"`
}

type ec2Stage117InstanceConnectEndpointSet struct {
	Items []ec2Stage117InstanceConnectEndpointItem `xml:"item"`
}

type ec2Stage117InstanceConnectEndpointItem struct {
	AvailabilityZone           string       `xml:"availabilityZone,omitempty"`
	CreatedAt                  string       `xml:"createdAt,omitempty"`
	DNSName                    string       `xml:"dnsName,omitempty"`
	FipsDNSName                string       `xml:"fipsDnsName,omitempty"`
	InstanceConnectEndpointARN string       `xml:"instanceConnectEndpointArn,omitempty"`
	InstanceConnectEndpointID  string       `xml:"instanceConnectEndpointId,omitempty"`
	NetworkInterfaceIDSet      ec2StringSet `xml:"networkInterfaceIdSet"`
	OwnerID                    string       `xml:"ownerId,omitempty"`
	PreserveClientIP           *bool        `xml:"preserveClientIp,omitempty"`
	SecurityGroupIDSet         ec2StringSet `xml:"securityGroupIdSet"`
	State                      string       `xml:"state,omitempty"`
	StateMessage               string       `xml:"stateMessage,omitempty"`
	SubnetID                   string       `xml:"subnetId,omitempty"`
	TagSet                     ec2TagSet    `xml:"tagSet"`
	VpcID                      string       `xml:"vpcId,omitempty"`
}

type ec2Stage117DescribeInstanceCreditSpecificationsResponse struct {
	XMLName                        xml.Name                                  `xml:"DescribeInstanceCreditSpecificationsResponse"`
	Xmlns                          string                                    `xml:"xmlns,attr"`
	RequestID                      string                                    `xml:"requestId"`
	InstanceCreditSpecificationSet ec2Stage117InstanceCreditSpecificationSet `xml:"instanceCreditSpecificationSet"`
	NextToken                      *string                                   `xml:"nextToken,omitempty"`
}

type ec2Stage117InstanceCreditSpecificationSet struct {
	Items []ec2Stage117InstanceCreditSpecificationItem `xml:"item"`
}

type ec2Stage117InstanceCreditSpecificationItem struct {
	CpuCredits string `xml:"cpuCredits,omitempty"`
	InstanceID string `xml:"instanceId,omitempty"`
}

type ec2Stage117DescribeInstanceEventNotificationAttributesResponse struct {
	XMLName              xml.Name                                        `xml:"DescribeInstanceEventNotificationAttributesResponse"`
	Xmlns                string                                          `xml:"xmlns,attr"`
	RequestID            string                                          `xml:"requestId"`
	InstanceTagAttribute ec2Stage117InstanceTagNotificationAttributeItem `xml:"instanceTagAttribute"`
}

type ec2Stage117InstanceTagNotificationAttributeItem struct {
	IncludeAllTagsOfInstance *bool        `xml:"includeAllTagsOfInstance,omitempty"`
	InstanceTagKeySet        ec2StringSet `xml:"instanceTagKeySet"`
}

type ec2Stage117DescribeInstanceEventWindowsResponse struct {
	XMLName                xml.Name                          `xml:"DescribeInstanceEventWindowsResponse"`
	Xmlns                  string                            `xml:"xmlns,attr"`
	RequestID              string                            `xml:"requestId"`
	InstanceEventWindowSet ec2Stage117InstanceEventWindowSet `xml:"instanceEventWindowSet"`
	NextToken              *string                           `xml:"nextToken,omitempty"`
}

type ec2Stage117InstanceEventWindowSet struct {
	Items []ec2Stage117InstanceEventWindowItem `xml:"item"`
}

type ec2Stage117InstanceEventWindowItem struct {
	AssociationTarget     ec2Stage117InstanceEventWindowAssociationTargetItem `xml:"associationTarget"`
	CronExpression        string                                              `xml:"cronExpression,omitempty"`
	InstanceEventWindowID string                                              `xml:"instanceEventWindowId,omitempty"`
	Name                  string                                              `xml:"name,omitempty"`
	State                 string                                              `xml:"state,omitempty"`
	TagSet                ec2TagSet                                           `xml:"tagSet"`
	TimeRangeSet          ec2Stage117InstanceEventWindowTimeRangeSet          `xml:"timeRangeSet"`
}

type ec2Stage117InstanceEventWindowAssociationTargetItem struct {
	DedicatedHostIDSet ec2StringSet `xml:"dedicatedHostIdSet"`
	InstanceIDSet      ec2StringSet `xml:"instanceIdSet"`
	TagSet             ec2TagSet    `xml:"tagSet"`
}

type ec2Stage117InstanceEventWindowTimeRangeSet struct {
	Items []ec2Stage117InstanceEventWindowTimeRangeItem `xml:"item"`
}

type ec2Stage117InstanceEventWindowTimeRangeItem struct {
	EndHour      *int32 `xml:"endHour,omitempty"`
	EndWeekDay   string `xml:"endWeekDay,omitempty"`
	StartHour    *int32 `xml:"startHour,omitempty"`
	StartWeekDay string `xml:"startWeekDay,omitempty"`
}

type ec2Stage117DescribeInstanceImageMetadataResponse struct {
	XMLName                  xml.Name                            `xml:"DescribeInstanceImageMetadataResponse"`
	Xmlns                    string                              `xml:"xmlns,attr"`
	RequestID                string                              `xml:"requestId"`
	InstanceImageMetadataSet ec2Stage117InstanceImageMetadataSet `xml:"instanceImageMetadataSet"`
	NextToken                *string                             `xml:"nextToken,omitempty"`
}

type ec2Stage117InstanceImageMetadataSet struct {
	Items []ec2Stage117InstanceImageMetadataItem `xml:"item"`
}

type ec2Stage117InstanceImageMetadataItem struct {
	AvailabilityZone string                       `xml:"availabilityZone,omitempty"`
	ImageMetadata    ec2Stage117ImageMetadataItem `xml:"imageMetadata"`
	InstanceID       string                       `xml:"instanceId,omitempty"`
	InstanceType     string                       `xml:"instanceType,omitempty"`
	LaunchTime       string                       `xml:"launchTime,omitempty"`
	InstanceOwnerID  string                       `xml:"instanceOwnerId,omitempty"`
	InstanceState    ec2InstanceStateItem         `xml:"instanceState"`
	TagSet           ec2TagSet                    `xml:"tagSet"`
	ZoneID           string                       `xml:"zoneId,omitempty"`
}

type ec2Stage117ImageMetadataItem struct {
	CreationDate    string `xml:"creationDate,omitempty"`
	DeprecationTime string `xml:"deprecationTime,omitempty"`
	ImageAllowed    *bool  `xml:"imageAllowed,omitempty"`
	ImageID         string `xml:"imageId,omitempty"`
	ImageOwnerAlias string `xml:"imageOwnerAlias,omitempty"`
	IsPublic        *bool  `xml:"isPublic,omitempty"`
	Name            string `xml:"name,omitempty"`
	ImageOwnerID    string `xml:"imageOwnerId,omitempty"`
	ImageState      string `xml:"imageState,omitempty"`
}

type ec2Stage117DescribeInstanceTopologyResponse struct {
	XMLName     xml.Name                       `xml:"DescribeInstanceTopologyResponse"`
	Xmlns       string                         `xml:"xmlns,attr"`
	RequestID   string                         `xml:"requestId"`
	InstanceSet ec2Stage117InstanceTopologySet `xml:"instanceSet"`
	NextToken   *string                        `xml:"nextToken,omitempty"`
}

type ec2Stage117InstanceTopologySet struct {
	Items []ec2Stage117InstanceTopologyItem `xml:"item"`
}

type ec2Stage117InstanceTopologyItem struct {
	AvailabilityZone string       `xml:"availabilityZone,omitempty"`
	CapacityBlockID  string       `xml:"capacityBlockId,omitempty"`
	GroupName        string       `xml:"groupName,omitempty"`
	InstanceID       string       `xml:"instanceId,omitempty"`
	InstanceType     string       `xml:"instanceType,omitempty"`
	NetworkNodeSet   ec2StringSet `xml:"networkNodeSet"`
	ZoneID           string       `xml:"zoneId,omitempty"`
}

type ec2Stage117DescribeInstanceTypeOfferingsResponse struct {
	XMLName                 xml.Name                           `xml:"DescribeInstanceTypeOfferingsResponse"`
	Xmlns                   string                             `xml:"xmlns,attr"`
	RequestID               string                             `xml:"requestId"`
	InstanceTypeOfferingSet ec2Stage117InstanceTypeOfferingSet `xml:"instanceTypeOfferingSet"`
	NextToken               *string                            `xml:"nextToken,omitempty"`
}

type ec2Stage117InstanceTypeOfferingSet struct {
	Items []ec2Stage117InstanceTypeOfferingItem `xml:"item"`
}

type ec2Stage117InstanceTypeOfferingItem struct {
	InstanceType string `xml:"instanceType,omitempty"`
	Location     string `xml:"location,omitempty"`
	LocationType string `xml:"locationType,omitempty"`
}
