package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage115Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeCapacityBlocks":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		capacityBlocks, nextToken, err := s.ec2.DescribeCapacityBlocks(
			parseEC2MembersOrItemList(r.Form, "CapacityBlockId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage115DescribeCapacityBlocksResponse{
			XMLName:          xml.Name{Local: "DescribeCapacityBlocksResponse"},
			Xmlns:            ec2Namespace,
			RequestID:        "stackyard-request",
			CapacityBlockSet: ec2Stage115CapacityBlockSet{Items: ec2Stage115CapacityBlockItemsFrom(capacityBlocks)},
			NextToken:        nextToken,
		})
		return true
	case "DescribeCapacityReservationBillingRequests":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		requests, nextToken, err := s.ec2.DescribeCapacityReservationBillingRequests(
			strings.TrimSpace(r.Form.Get("Role")),
			parseEC2MembersOrItemList(r.Form, "CapacityReservationId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage115DescribeCapacityReservationBillingRequestsResponse{
			XMLName:                              xml.Name{Local: "DescribeCapacityReservationBillingRequestsResponse"},
			Xmlns:                                ec2Namespace,
			RequestID:                            "stackyard-request",
			CapacityReservationBillingRequestSet: ec2Stage115CapacityReservationBillingRequestSet{Items: ec2Stage115CapacityReservationBillingRequestItemsFrom(requests)},
			NextToken:                            nextToken,
		})
		return true
	case "DescribeCapacityReservationFleets":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		fleets, nextToken, err := s.ec2.DescribeCapacityReservationFleets(
			parseEC2MembersOrItemList(r.Form, "CapacityReservationFleetId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage115DescribeCapacityReservationFleetsResponse{
			XMLName:                     xml.Name{Local: "DescribeCapacityReservationFleetsResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			CapacityReservationFleetSet: ec2Stage115CapacityReservationFleetSet{Items: ec2Stage115CapacityReservationFleetItemsFrom(fleets)},
			NextToken:                   nextToken,
		})
		return true
	case "DescribeCapacityReservations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		reservations, nextToken, err := s.ec2.DescribeCapacityReservations(
			parseEC2MembersOrItemList(r.Form, "CapacityReservationId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage115DescribeCapacityReservationsResponse{
			XMLName:                xml.Name{Local: "DescribeCapacityReservationsResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			CapacityReservationSet: ec2Stage115CapacityReservationSet{Items: ec2Stage115CapacityReservationItemsFrom(reservations)},
			NextToken:              nextToken,
		})
		return true
	case "DescribeCarrierGateways":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		gateways, nextToken, err := s.ec2.DescribeCarrierGateways(
			parseEC2MembersOrItemList(r.Form, "CarrierGatewayId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage115DescribeCarrierGatewaysResponse{
			XMLName:           xml.Name{Local: "DescribeCarrierGatewaysResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			CarrierGatewaySet: ec2Stage115CarrierGatewaySet{Items: ec2Stage115CarrierGatewayItemsFrom(gateways)},
			NextToken:         nextToken,
		})
		return true
	case "DescribeCoipPools":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		coipPools, nextToken, err := s.ec2.DescribeCoipPools(
			parseEC2MembersOrItemList(r.Form, "PoolId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage115DescribeCoipPoolsResponse{
			XMLName:     xml.Name{Local: "DescribeCoipPoolsResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			CoipPoolSet: ec2Stage115CoipPoolSet{Items: ec2Stage115CoipPoolItemsFrom(coipPools)},
			NextToken:   nextToken,
		})
		return true
	case "DescribeConversionTasks":
		conversionTasks := s.ec2.DescribeConversionTasks(
			parseEC2MembersOrItemList(r.Form, "ConversionTaskId"),
		)
		respondEC2XML(w, ec2Stage115DescribeConversionTasksResponse{
			XMLName:         xml.Name{Local: "DescribeConversionTasksResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			ConversionTasks: ec2Stage115ConversionTaskSet{Items: ec2Stage115ConversionTaskItemsFrom(conversionTasks)},
		})
		return true
	case "DescribeElasticGpus":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		elasticGpus, nextToken, err := s.ec2.DescribeElasticGpus(
			parseEC2MembersOrItemList(r.Form, "ElasticGpuId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage115DescribeElasticGpusResponse{
			XMLName:       xml.Name{Local: "DescribeElasticGpusResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			ElasticGpuSet: ec2Stage115ElasticGpuSet{Items: ec2Stage115ElasticGpuItemsFrom(elasticGpus)},
			MaxResults:    maxResults,
			NextToken:     nextToken,
		})
		return true
	case "DescribeExportImageTasks":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		exportImageTasks, nextToken, err := s.ec2.DescribeExportImageTasks(
			parseEC2MembersOrItemList(r.Form, "ExportImageTaskId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage115DescribeExportImageTasksResponse{
			XMLName:            xml.Name{Local: "DescribeExportImageTasksResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			ExportImageTaskSet: ec2Stage115ExportImageTaskSet{Items: ec2Stage115ExportImageTaskItemsFrom(exportImageTasks)},
			NextToken:          nextToken,
		})
		return true
	case "DescribeExportTasks":
		exportTasks := s.ec2.DescribeExportTasks(
			parseEC2MembersOrItemList(r.Form, "ExportTaskId"),
			parseEC2Filters(r.Form),
		)
		respondEC2XML(w, ec2Stage115DescribeExportTasksResponse{
			XMLName:       xml.Name{Local: "DescribeExportTasksResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			ExportTaskSet: ec2Stage115ExportTaskSet{Items: ec2Stage115ExportTaskItemsFrom(exportTasks)},
		})
		return true
	default:
		return false
	}
}

func ec2Stage115CapacityBlockItemsFrom(in []ec2svc.CapacityBlock) []ec2Stage115CapacityBlockItem {
	out := make([]ec2Stage115CapacityBlockItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage115CapacityBlockItem{
			AvailabilityZone:       item.AvailabilityZone,
			AvailabilityZoneID:     item.AvailabilityZoneID,
			CapacityBlockID:        item.CapacityBlockID,
			CapacityReservationIDs: ec2StringSet{Items: append([]string(nil), item.CapacityReservationIDs...)},
			CreateDate:             ec2Stage115RFC3339(item.CreateDate),
			EndDate:                ec2Stage115RFC3339(item.EndDate),
			StartDate:              ec2Stage115RFC3339(item.StartDate),
			State:                  item.State,
			TagSet:                 ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			UltraserverType:        item.UltraserverType,
		})
	}
	return out
}

func ec2Stage115CapacityReservationBillingRequestItemsFrom(in []ec2svc.CapacityReservationBillingRequest) []ec2Stage115CapacityReservationBillingRequestItem {
	out := make([]ec2Stage115CapacityReservationBillingRequestItem, 0, len(in))
	for _, item := range in {
		request := ec2Stage115CapacityReservationBillingRequestItem{
			CapacityReservationID:           item.CapacityReservationID,
			LastUpdateTime:                  ec2Stage115RFC3339(item.LastUpdateTime),
			RequestedBy:                     item.RequestedBy,
			Status:                          item.Status,
			StatusMessage:                   item.StatusMessage,
			UnusedReservationBillingOwnerID: item.UnusedReservationBillingOwnerID,
		}
		if item.CapacityReservationInfo != nil {
			request.CapacityReservationInfo = &ec2Stage115CapacityReservationInfoItem{
				AvailabilityZone:   item.CapacityReservationInfo.AvailabilityZone,
				AvailabilityZoneID: item.CapacityReservationInfo.AvailabilityZoneID,
				InstanceType:       item.CapacityReservationInfo.InstanceType,
				Tenancy:            item.CapacityReservationInfo.Tenancy,
			}
		}
		out = append(out, request)
	}
	return out
}

func ec2Stage115CapacityReservationFleetItemsFrom(in []ec2svc.CapacityReservationFleet) []ec2Stage115CapacityReservationFleetItem {
	out := make([]ec2Stage115CapacityReservationFleetItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage115CapacityReservationFleetItem{
			AllocationStrategy:         item.AllocationStrategy,
			CapacityReservationFleetID: item.ID,
			CreateTime:                 ec2Stage115RFC3339(item.CreateTime),
			EndDate:                    ec2OptionalRFC3339(item.EndDate),
			InstanceMatchCriteria:      item.InstanceMatchCriteria,
			InstanceTypeSpecificationSet: ec2Stage115FleetCapacityReservationSet{
				Items: ec2Stage115FleetCapacityReservationItemsFrom(item.FleetCapacityReservations),
			},
			State:                  item.State,
			TagSet:                 ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			Tenancy:                item.Tenancy,
			TotalFulfilledCapacity: item.TotalFulfilledCapacity,
			TotalTargetCapacity:    item.TotalTargetCapacity,
		})
	}
	return out
}

func ec2Stage115FleetCapacityReservationItemsFrom(in []ec2svc.FleetCapacityReservation) []ec2Stage115FleetCapacityReservationItem {
	out := make([]ec2Stage115FleetCapacityReservationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage115FleetCapacityReservationItem{
			AvailabilityZone:      item.AvailabilityZone,
			AvailabilityZoneID:    item.AvailabilityZoneID,
			CapacityReservationID: item.CapacityReservationID,
			CreateDate:            ec2Stage115RFC3339(item.CreateDate),
			EbsOptimized:          item.EbsOptimized,
			FulfilledCapacity:     item.FulfilledCapacity,
			InstancePlatform:      item.InstancePlatform,
			InstanceType:          item.InstanceType,
			Priority:              item.Priority,
			TotalInstanceCount:    item.TotalInstanceCount,
			Weight:                item.Weight,
		})
	}
	return out
}

func ec2Stage115CapacityReservationItemsFrom(in []ec2svc.CapacityReservation) []ec2Stage102CapacityReservationItem {
	out := make([]ec2Stage102CapacityReservationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage102CapacityReservationItemFrom(item))
	}
	return out
}

func ec2Stage115CarrierGatewayItemsFrom(in []ec2svc.CarrierGateway) []ec2Stage105CarrierGatewayItem {
	out := make([]ec2Stage105CarrierGatewayItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage105CarrierGatewayItem{
			CarrierGatewayID: item.ID,
			OwnerID:          item.OwnerID,
			State:            item.State,
			TagSet:           ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			VpcID:            item.VpcID,
		})
	}
	return out
}

func ec2Stage115CoipPoolItemsFrom(in []ec2svc.CoipPool) []ec2Stage107CoipPoolItem {
	out := make([]ec2Stage107CoipPoolItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage107CoipPoolItemFrom(item))
	}
	return out
}

func ec2Stage115ConversionTaskItemsFrom(in []ec2svc.ConversionTask) []ec2Stage115ConversionTaskItem {
	out := make([]ec2Stage115ConversionTaskItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage115ConversionTaskItem{
			ConversionTaskID: item.ConversionTaskID,
			ExpirationTime:   item.ExpirationTime,
			State:            item.State,
			StatusMessage:    item.StatusMessage,
			TagSet:           ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage115ElasticGpuItemsFrom(in []ec2svc.ElasticGpu) []ec2Stage115ElasticGpuItem {
	out := make([]ec2Stage115ElasticGpuItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage115ElasticGpuItem{
			AvailabilityZone: item.AvailabilityZone,
			ElasticGpuHealth: ec2Stage115ElasticGpuHealthItem{Status: item.ElasticGpuHealth},
			ElasticGpuID:     item.ElasticGpuID,
			ElasticGpuState:  item.ElasticGpuState,
			ElasticGpuType:   item.ElasticGpuType,
			InstanceID:       item.InstanceID,
			TagSet:           ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage115ExportImageTaskItemsFrom(in []ec2svc.ExportImageTask) []ec2Stage115ExportImageTaskItem {
	out := make([]ec2Stage115ExportImageTaskItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage115ExportImageTaskItem{
			Description:       item.Description,
			ExportImageTaskID: item.ExportImageTaskID,
			ImageID:           item.ImageID,
			Progress:          item.Progress,
			S3ExportLocation: ec2Stage115ExportTaskS3LocationItem{
				S3Bucket: item.S3ExportLocation.S3Bucket,
				S3Prefix: item.S3ExportLocation.S3Prefix,
			},
			Status:        item.Status,
			StatusMessage: item.StatusMessage,
			TagSet:        ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage115ExportTaskItemsFrom(in []ec2svc.InstanceExportTask) []ec2Stage107ExportTaskItem {
	out := make([]ec2Stage107ExportTaskItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage107ExportTaskItem{
			Description:  item.Description,
			ExportTaskID: item.ExportTaskID,
			ExportToS3: ec2Stage107ExportToS3TaskItem{
				ContainerFormat: item.ContainerFormat,
				DiskImageFormat: item.DiskImageFormat,
				S3Bucket:        item.S3Bucket,
				S3Key:           item.S3Key,
			},
			InstanceExport: ec2Stage107InstanceExportDetailsItem{
				InstanceID:        item.InstanceID,
				TargetEnvironment: item.TargetEnvironment,
			},
			State:         item.State,
			StatusMessage: item.StatusMessage,
			TagSet:        ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage115RFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

type ec2Stage115DescribeCapacityBlocksResponse struct {
	XMLName          xml.Name                    `xml:"DescribeCapacityBlocksResponse"`
	Xmlns            string                      `xml:"xmlns,attr"`
	RequestID        string                      `xml:"requestId"`
	CapacityBlockSet ec2Stage115CapacityBlockSet `xml:"capacityBlockSet"`
	NextToken        *string                     `xml:"nextToken,omitempty"`
}

type ec2Stage115CapacityBlockSet struct {
	Items []ec2Stage115CapacityBlockItem `xml:"item"`
}

type ec2Stage115CapacityBlockItem struct {
	AvailabilityZone       string       `xml:"availabilityZone,omitempty"`
	AvailabilityZoneID     string       `xml:"availabilityZoneId,omitempty"`
	CapacityBlockID        string       `xml:"capacityBlockId,omitempty"`
	CapacityReservationIDs ec2StringSet `xml:"capacityReservationIdSet"`
	CreateDate             string       `xml:"createDate,omitempty"`
	EndDate                string       `xml:"endDate,omitempty"`
	StartDate              string       `xml:"startDate,omitempty"`
	State                  string       `xml:"state,omitempty"`
	TagSet                 ec2TagSet    `xml:"tagSet"`
	UltraserverType        string       `xml:"ultraserverType,omitempty"`
}

type ec2Stage115DescribeCapacityReservationBillingRequestsResponse struct {
	XMLName                              xml.Name                                        `xml:"DescribeCapacityReservationBillingRequestsResponse"`
	Xmlns                                string                                          `xml:"xmlns,attr"`
	RequestID                            string                                          `xml:"requestId"`
	CapacityReservationBillingRequestSet ec2Stage115CapacityReservationBillingRequestSet `xml:"capacityReservationBillingRequestSet"`
	NextToken                            *string                                         `xml:"nextToken,omitempty"`
}

type ec2Stage115CapacityReservationBillingRequestSet struct {
	Items []ec2Stage115CapacityReservationBillingRequestItem `xml:"item"`
}

type ec2Stage115CapacityReservationBillingRequestItem struct {
	CapacityReservationID           string                                  `xml:"capacityReservationId,omitempty"`
	CapacityReservationInfo         *ec2Stage115CapacityReservationInfoItem `xml:"capacityReservationInfo,omitempty"`
	LastUpdateTime                  string                                  `xml:"lastUpdateTime,omitempty"`
	RequestedBy                     string                                  `xml:"requestedBy,omitempty"`
	Status                          string                                  `xml:"status,omitempty"`
	StatusMessage                   string                                  `xml:"statusMessage,omitempty"`
	UnusedReservationBillingOwnerID string                                  `xml:"unusedReservationBillingOwnerId,omitempty"`
}

type ec2Stage115CapacityReservationInfoItem struct {
	AvailabilityZone   string `xml:"availabilityZone,omitempty"`
	AvailabilityZoneID string `xml:"availabilityZoneId,omitempty"`
	InstanceType       string `xml:"instanceType,omitempty"`
	Tenancy            string `xml:"tenancy,omitempty"`
}

type ec2Stage115DescribeCapacityReservationFleetsResponse struct {
	XMLName                     xml.Name                               `xml:"DescribeCapacityReservationFleetsResponse"`
	Xmlns                       string                                 `xml:"xmlns,attr"`
	RequestID                   string                                 `xml:"requestId"`
	CapacityReservationFleetSet ec2Stage115CapacityReservationFleetSet `xml:"capacityReservationFleetSet"`
	NextToken                   *string                                `xml:"nextToken,omitempty"`
}

type ec2Stage115CapacityReservationFleetSet struct {
	Items []ec2Stage115CapacityReservationFleetItem `xml:"item"`
}

type ec2Stage115CapacityReservationFleetItem struct {
	AllocationStrategy           string                                 `xml:"allocationStrategy,omitempty"`
	CapacityReservationFleetArn  string                                 `xml:"capacityReservationFleetArn,omitempty"`
	CapacityReservationFleetID   string                                 `xml:"capacityReservationFleetId,omitempty"`
	CreateTime                   string                                 `xml:"createTime,omitempty"`
	EndDate                      string                                 `xml:"endDate,omitempty"`
	InstanceMatchCriteria        string                                 `xml:"instanceMatchCriteria,omitempty"`
	InstanceTypeSpecificationSet ec2Stage115FleetCapacityReservationSet `xml:"instanceTypeSpecificationSet"`
	State                        string                                 `xml:"state,omitempty"`
	TagSet                       ec2TagSet                              `xml:"tagSet"`
	Tenancy                      string                                 `xml:"tenancy,omitempty"`
	TotalFulfilledCapacity       float64                                `xml:"totalFulfilledCapacity,omitempty"`
	TotalTargetCapacity          int32                                  `xml:"totalTargetCapacity,omitempty"`
}

type ec2Stage115FleetCapacityReservationSet struct {
	Items []ec2Stage115FleetCapacityReservationItem `xml:"item"`
}

type ec2Stage115FleetCapacityReservationItem struct {
	AvailabilityZone      string   `xml:"availabilityZone,omitempty"`
	AvailabilityZoneID    string   `xml:"availabilityZoneId,omitempty"`
	CapacityReservationID string   `xml:"capacityReservationId,omitempty"`
	CreateDate            string   `xml:"createDate,omitempty"`
	EbsOptimized          *bool    `xml:"ebsOptimized,omitempty"`
	FulfilledCapacity     float64  `xml:"fulfilledCapacity,omitempty"`
	InstancePlatform      string   `xml:"instancePlatform,omitempty"`
	InstanceType          string   `xml:"instanceType,omitempty"`
	Priority              *int32   `xml:"priority,omitempty"`
	TotalInstanceCount    int32    `xml:"totalInstanceCount,omitempty"`
	Weight                *float64 `xml:"weight,omitempty"`
}

type ec2Stage115DescribeCapacityReservationsResponse struct {
	XMLName                xml.Name                          `xml:"DescribeCapacityReservationsResponse"`
	Xmlns                  string                            `xml:"xmlns,attr"`
	RequestID              string                            `xml:"requestId"`
	CapacityReservationSet ec2Stage115CapacityReservationSet `xml:"capacityReservationSet"`
	NextToken              *string                           `xml:"nextToken,omitempty"`
}

type ec2Stage115CapacityReservationSet struct {
	Items []ec2Stage102CapacityReservationItem `xml:"item"`
}

type ec2Stage115DescribeCarrierGatewaysResponse struct {
	XMLName           xml.Name                     `xml:"DescribeCarrierGatewaysResponse"`
	Xmlns             string                       `xml:"xmlns,attr"`
	RequestID         string                       `xml:"requestId"`
	CarrierGatewaySet ec2Stage115CarrierGatewaySet `xml:"carrierGatewaySet"`
	NextToken         *string                      `xml:"nextToken,omitempty"`
}

type ec2Stage115CarrierGatewaySet struct {
	Items []ec2Stage105CarrierGatewayItem `xml:"item"`
}

type ec2Stage115DescribeCoipPoolsResponse struct {
	XMLName     xml.Name               `xml:"DescribeCoipPoolsResponse"`
	Xmlns       string                 `xml:"xmlns,attr"`
	RequestID   string                 `xml:"requestId"`
	CoipPoolSet ec2Stage115CoipPoolSet `xml:"coipPoolSet"`
	NextToken   *string                `xml:"nextToken,omitempty"`
}

type ec2Stage115CoipPoolSet struct {
	Items []ec2Stage107CoipPoolItem `xml:"item"`
}

type ec2Stage115DescribeConversionTasksResponse struct {
	XMLName         xml.Name                     `xml:"DescribeConversionTasksResponse"`
	Xmlns           string                       `xml:"xmlns,attr"`
	RequestID       string                       `xml:"requestId"`
	ConversionTasks ec2Stage115ConversionTaskSet `xml:"conversionTasks"`
}

type ec2Stage115ConversionTaskSet struct {
	Items []ec2Stage115ConversionTaskItem `xml:"item"`
}

type ec2Stage115ConversionTaskItem struct {
	ConversionTaskID string    `xml:"conversionTaskId,omitempty"`
	ExpirationTime   string    `xml:"expirationTime,omitempty"`
	State            string    `xml:"state,omitempty"`
	StatusMessage    string    `xml:"statusMessage,omitempty"`
	TagSet           ec2TagSet `xml:"tagSet"`
}

type ec2Stage115DescribeElasticGpusResponse struct {
	XMLName       xml.Name                 `xml:"DescribeElasticGpusResponse"`
	Xmlns         string                   `xml:"xmlns,attr"`
	RequestID     string                   `xml:"requestId"`
	ElasticGpuSet ec2Stage115ElasticGpuSet `xml:"elasticGpuSet"`
	MaxResults    *int32                   `xml:"maxResults,omitempty"`
	NextToken     *string                  `xml:"nextToken,omitempty"`
}

type ec2Stage115ElasticGpuSet struct {
	Items []ec2Stage115ElasticGpuItem `xml:"item"`
}

type ec2Stage115ElasticGpuItem struct {
	AvailabilityZone string                          `xml:"availabilityZone,omitempty"`
	ElasticGpuHealth ec2Stage115ElasticGpuHealthItem `xml:"elasticGpuHealth"`
	ElasticGpuID     string                          `xml:"elasticGpuId,omitempty"`
	ElasticGpuState  string                          `xml:"elasticGpuState,omitempty"`
	ElasticGpuType   string                          `xml:"elasticGpuType,omitempty"`
	InstanceID       string                          `xml:"instanceId,omitempty"`
	TagSet           ec2TagSet                       `xml:"tagSet"`
}

type ec2Stage115ElasticGpuHealthItem struct {
	Status string `xml:"status,omitempty"`
}

type ec2Stage115DescribeExportImageTasksResponse struct {
	XMLName            xml.Name                      `xml:"DescribeExportImageTasksResponse"`
	Xmlns              string                        `xml:"xmlns,attr"`
	RequestID          string                        `xml:"requestId"`
	ExportImageTaskSet ec2Stage115ExportImageTaskSet `xml:"exportImageTaskSet"`
	NextToken          *string                       `xml:"nextToken,omitempty"`
}

type ec2Stage115ExportImageTaskSet struct {
	Items []ec2Stage115ExportImageTaskItem `xml:"item"`
}

type ec2Stage115ExportImageTaskItem struct {
	Description       string                              `xml:"description,omitempty"`
	ExportImageTaskID string                              `xml:"exportImageTaskId,omitempty"`
	ImageID           string                              `xml:"imageId,omitempty"`
	Progress          string                              `xml:"progress,omitempty"`
	S3ExportLocation  ec2Stage115ExportTaskS3LocationItem `xml:"s3ExportLocation"`
	Status            string                              `xml:"status,omitempty"`
	StatusMessage     string                              `xml:"statusMessage,omitempty"`
	TagSet            ec2TagSet                           `xml:"tagSet"`
}

type ec2Stage115ExportTaskS3LocationItem struct {
	S3Bucket string `xml:"s3Bucket,omitempty"`
	S3Prefix string `xml:"s3Prefix,omitempty"`
}

type ec2Stage115DescribeExportTasksResponse struct {
	XMLName       xml.Name                 `xml:"DescribeExportTasksResponse"`
	Xmlns         string                   `xml:"xmlns,attr"`
	RequestID     string                   `xml:"requestId"`
	ExportTaskSet ec2Stage115ExportTaskSet `xml:"exportTaskSet"`
}

type ec2Stage115ExportTaskSet struct {
	Items []ec2Stage107ExportTaskItem `xml:"item"`
}
