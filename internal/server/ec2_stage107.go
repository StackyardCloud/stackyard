package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage107Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateCoipPool":
		coipPool, err := s.ec2.CreateCoipPool(
			strings.TrimSpace(r.Form.Get("LocalGatewayRouteTableId")),
			parseEC2TagSpecificationsForResource(r.Form, "coip-pool"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage107CreateCoipPoolResponse{
			XMLName:   xml.Name{Local: "CreateCoipPoolResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			CoipPool:  ec2Stage107CoipPoolItemFrom(coipPool),
		})
		return true
	case "CreateDelegateMacVolumeOwnershipTask":
		task, err := s.ec2.CreateDelegateMacVolumeOwnershipTask(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("MacCredentials")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			parseEC2TagSpecificationsForResource(r.Form, "mac-modification-task"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage107CreateDelegateMacVolumeOwnershipTaskResponse{
			XMLName:             xml.Name{Local: "CreateDelegateMacVolumeOwnershipTaskResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			MacModificationTask: ec2Stage107MacModificationTaskItemFrom(task),
		})
		return true
	case "CreateFleet":
		if !hasEC2PrefixedField(r.Form, "LaunchTemplateConfigs.") {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		totalTargetCapacity, ok := parseEC2Stage107FleetTotalTargetCapacity(r.Form)
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		fleet, err := s.ec2.CreateFleet(
			true,
			totalTargetCapacity,
			parseEC2Stage107FleetInstanceType(r.Form),
			parseEC2TagSpecificationsForResource(r.Form, "fleet"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage107CreateFleetResponse{
			XMLName:   xml.Name{Local: "CreateFleetResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			FleetID:   fleet.FleetID,
			FleetInstanceSet: ec2Stage107CreateFleetInstanceSet{
				Items: ec2Stage107CreateFleetInstanceItemsFrom(fleet.Instances),
			},
		})
		return true
	case "CreateFlowLogs":
		flowLogs, err := s.ec2.CreateFlowLogs(
			parseEC2MembersOrItemList(r.Form, "ResourceId"),
			strings.TrimSpace(r.Form.Get("ResourceType")),
			strings.TrimSpace(r.Form.Get("TrafficType")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			parseEC2TagSpecificationsForResource(r.Form, "vpc-flow-log"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage107CreateFlowLogsResponse{
			XMLName:      xml.Name{Local: "CreateFlowLogsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			ClientToken:  flowLogs.ClientToken,
			FlowLogIDSet: ec2StringSet{Items: flowLogs.FlowLogIDs},
		})
		return true
	case "CreateFpgaImage":
		fpgaImage, err := s.ec2.CreateFpgaImage(
			strings.TrimSpace(r.Form.Get("InputStorageLocation.Bucket")),
			strings.TrimSpace(r.Form.Get("InputStorageLocation.Key")),
			parseEC2OptionalString(r.Form.Get("Name")),
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2TagSpecificationsForResource(r.Form, "fpga-image"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage107CreateFpgaImageResponse{
			XMLName:           xml.Name{Local: "CreateFpgaImageResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			FpgaImageID:       fpgaImage.FpgaImageID,
			FpgaImageGlobalID: fpgaImage.FpgaImageGlobalID,
		})
		return true
	case "CreateInstanceConnectEndpoint":
		preserveClientIP, hasPreserveClientIP, ok := ec2OptionalBoolFromForm(r.Form, "PreserveClientIp")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasPreserveClientIP {
			preserveClientIP = nil
		}
		endpoint, err := s.ec2.CreateInstanceConnectEndpoint(
			strings.TrimSpace(r.Form.Get("SubnetId")),
			parseEC2MembersOrItemList(r.Form, "SecurityGroupId"),
			strings.TrimSpace(r.Form.Get("IpAddressType")),
			preserveClientIP,
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			parseEC2TagSpecificationsForResource(r.Form, "instance-connect-endpoint"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage107CreateInstanceConnectEndpointResponse{
			XMLName:                 xml.Name{Local: "CreateInstanceConnectEndpointResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			ClientToken:             endpoint.ClientToken,
			InstanceConnectEndpoint: ec2Stage107InstanceConnectEndpointItemFrom(endpoint),
		})
		return true
	case "CreateInstanceEventWindow":
		timeRanges, ok := parseEC2Stage107InstanceEventWindowTimeRanges(r.Form)
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		eventWindow, err := s.ec2.CreateInstanceEventWindow(
			strings.TrimSpace(r.Form.Get("Name")),
			strings.TrimSpace(r.Form.Get("CronExpression")),
			timeRanges,
			parseEC2TagSpecificationsForResource(r.Form, "instance-event-window"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage107CreateInstanceEventWindowResponse{
			XMLName:             xml.Name{Local: "CreateInstanceEventWindowResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			InstanceEventWindow: ec2Stage107InstanceEventWindowItemFrom(eventWindow),
		})
		return true
	case "CreateInstanceExportTask":
		exportTask, err := s.ec2.CreateInstanceExportTask(
			parseEC2OptionalString(r.Form.Get("Description")),
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("TargetEnvironment")),
			strings.TrimSpace(r.Form.Get("ExportToS3.S3Bucket")),
			strings.TrimSpace(r.Form.Get("ExportToS3.S3Prefix")),
			strings.TrimSpace(r.Form.Get("ExportToS3.ContainerFormat")),
			strings.TrimSpace(r.Form.Get("ExportToS3.DiskImageFormat")),
			parseEC2TagSpecificationsForResource(r.Form, "export-image-task"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage107CreateInstanceExportTaskResponse{
			XMLName:   xml.Name{Local: "CreateInstanceExportTaskResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ExportTask: ec2Stage107ExportTaskItem{
				Description:  exportTask.Description,
				ExportTaskID: exportTask.ExportTaskID,
				ExportToS3: ec2Stage107ExportToS3TaskItem{
					ContainerFormat: exportTask.ContainerFormat,
					DiskImageFormat: exportTask.DiskImageFormat,
					S3Bucket:        exportTask.S3Bucket,
					S3Key:           exportTask.S3Key,
				},
				InstanceExport: ec2Stage107InstanceExportDetailsItem{
					InstanceID:        exportTask.InstanceID,
					TargetEnvironment: exportTask.TargetEnvironment,
				},
				State:         exportTask.State,
				StatusMessage: exportTask.StatusMessage,
				TagSet:        ec2TagSet{Items: ec2TagItemsFromMap(exportTask.Tags)},
			},
		})
		return true
	case "CreateIpam":
		enablePrivateGua, hasEnablePrivateGua, ok := ec2OptionalBoolFromForm(r.Form, "EnablePrivateGua")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasEnablePrivateGua {
			enablePrivateGua = nil
		}
		ipam, err := s.ec2.CreateIpam(
			parseEC2OptionalString(r.Form.Get("Description")),
			enablePrivateGua,
			strings.TrimSpace(r.Form.Get("MeteredAccount")),
			parseEC2Stage107OperatingRegions(r.Form),
			strings.TrimSpace(r.Form.Get("Tier")),
			parseEC2TagSpecificationsForResource(r.Form, "ipam"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage107CreateIpamResponse{
			XMLName:   xml.Name{Local: "CreateIpamResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Ipam:      ec2Stage107IpamItemFrom(ipam),
		})
		return true
	case "CreateIpamExternalResourceVerificationToken":
		token, err := s.ec2.CreateIpamExternalResourceVerificationToken(
			strings.TrimSpace(r.Form.Get("IpamId")),
			parseEC2OptionalString(r.Form.Get("TokenName")),
			parseEC2TagSpecificationsForResource(r.Form, "ipam-external-resource-verification-token"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage107CreateIpamExternalResourceVerificationTokenResponse{
			XMLName:   xml.Name{Local: "CreateIpamExternalResourceVerificationTokenResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			IpamExternalResourceVerificationToken: ec2Stage107IpamExternalResourceVerificationTokenItem{
				IpamARN:                                  token.IpamARN,
				IpamExternalResourceVerificationTokenARN: token.IpamExternalResourceVerificationTokenARN,
				IpamExternalResourceVerificationTokenID:  token.IpamExternalResourceVerificationTokenID,
				IpamID:                                   token.IpamID,
				IpamRegion:                               token.IpamRegion,
				NotAfter:                                 token.NotAfter.UTC().Format(time.RFC3339),
				State:                                    token.State,
				Status:                                   token.Status,
				TagSet:                                   ec2TagSet{Items: ec2TagItemsFromMap(token.Tags)},
				TokenName:                                token.TokenName,
			},
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage107FleetTotalTargetCapacity(values url.Values) (int32, bool) {
	total, hasTotal, ok := ec2OptionalInt32FromForm(values, "TargetCapacitySpecification.TotalTargetCapacity")
	if !ok {
		return 0, false
	}
	if hasTotal && total != nil && *total > 0 {
		return *total, true
	}

	onDemand, hasOnDemand, ok := ec2OptionalInt32FromForm(values, "TargetCapacitySpecification.OnDemandTargetCapacity")
	if !ok {
		return 0, false
	}
	spot, hasSpot, ok := ec2OptionalInt32FromForm(values, "TargetCapacitySpecification.SpotTargetCapacity")
	if !ok {
		return 0, false
	}
	if !hasOnDemand && !hasSpot {
		return 0, false
	}

	var sum int32
	if onDemand != nil {
		sum += *onDemand
	}
	if spot != nil {
		sum += *spot
	}
	if sum <= 0 {
		return 0, false
	}
	return sum, true
}

func parseEC2Stage107FleetInstanceType(values url.Values) string {
	keys := make([]string, 0)
	for key := range values {
		if !strings.HasPrefix(key, "LaunchTemplateConfigs.") {
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

func parseEC2Stage107InstanceEventWindowTimeRanges(values url.Values) ([]ec2svc.InstanceEventWindowTimeRange, bool) {
	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, "TimeRange.") {
			continue
		}
		rest := strings.TrimPrefix(key, "TimeRange.")
		rest = strings.TrimPrefix(rest, "Member.")
		rest = strings.TrimPrefix(rest, "Item.")
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

	out := make([]ec2svc.InstanceEventWindowTimeRange, 0, len(ordered))
	for _, idx := range ordered {
		baseCandidates := []string{
			"TimeRange." + strconv.Itoa(idx) + ".",
			"TimeRange.Member." + strconv.Itoa(idx) + ".",
			"TimeRange.Item." + strconv.Itoa(idx) + ".",
		}
		startHour, ok := parseEC2Stage107OptionalInt32FromPrefixes(values, baseCandidates, "StartHour")
		if !ok {
			return nil, false
		}
		endHour, ok := parseEC2Stage107OptionalInt32FromPrefixes(values, baseCandidates, "EndHour")
		if !ok {
			return nil, false
		}
		out = append(out, ec2svc.InstanceEventWindowTimeRange{
			EndHour:      endHour,
			EndWeekDay:   parseEC2Stage107StringFromPrefixes(values, baseCandidates, "EndWeekDay"),
			StartHour:    startHour,
			StartWeekDay: parseEC2Stage107StringFromPrefixes(values, baseCandidates, "StartWeekDay"),
		})
	}
	return out, true
}

func parseEC2Stage107OptionalInt32FromPrefixes(values url.Values, prefixes []string, field string) (*int32, bool) {
	for _, prefix := range prefixes {
		key := prefix + field
		if !hasEC2Field(values, key) {
			continue
		}
		parsed, ok := parseEC2OptionalInt32(values.Get(key))
		if !ok {
			return nil, false
		}
		return parsed, true
	}
	return nil, true
}

func parseEC2Stage107StringFromPrefixes(values url.Values, prefixes []string, field string) string {
	for _, prefix := range prefixes {
		key := prefix + field
		if !hasEC2Field(values, key) {
			continue
		}
		return strings.TrimSpace(values.Get(key))
	}
	return ""
}

func parseEC2Stage107OperatingRegions(values url.Values) []string {
	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, "OperatingRegion.") {
			continue
		}
		rest := strings.TrimPrefix(key, "OperatingRegion.")
		rest = strings.TrimPrefix(rest, "Member.")
		rest = strings.TrimPrefix(rest, "Item.")
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
	out := make([]string, 0, len(ordered))
	seen := map[string]struct{}{}
	for _, idx := range ordered {
		candidates := []string{
			"OperatingRegion." + strconv.Itoa(idx) + ".RegionName",
			"OperatingRegion.Member." + strconv.Itoa(idx) + ".RegionName",
			"OperatingRegion.Item." + strconv.Itoa(idx) + ".RegionName",
		}
		region := ""
		for _, key := range candidates {
			if !hasEC2Field(values, key) {
				continue
			}
			region = strings.TrimSpace(values.Get(key))
			break
		}
		if region == "" {
			continue
		}
		if _, ok := seen[region]; ok {
			continue
		}
		seen[region] = struct{}{}
		out = append(out, region)
	}
	return out
}

func ec2Stage107CoipPoolItemFrom(in ec2svc.CoipPool) ec2Stage107CoipPoolItem {
	return ec2Stage107CoipPoolItem{
		LocalGatewayRouteTableID: in.LocalGatewayRouteTableID,
		PoolARN:                  in.PoolARN,
		PoolCidrSet:              ec2StringSet{Items: append([]string(nil), in.PoolCIDRs...)},
		PoolID:                   in.PoolID,
		TagSet:                   ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
	}
}

func ec2Stage107MacModificationTaskItemFrom(in ec2svc.MacModificationTask) ec2Stage107MacModificationTaskItem {
	return ec2Stage107MacModificationTaskItem{
		InstanceID:            in.InstanceID,
		MacModificationTaskID: in.MacModificationTaskID,
		StartTime:             in.StartTime.UTC().Format(time.RFC3339),
		TagSet:                ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TaskState:             in.TaskState,
		TaskType:              in.TaskType,
	}
}

func ec2Stage107CreateFleetInstanceItemsFrom(in []ec2svc.FleetInstance) []ec2Stage107CreateFleetInstanceItem {
	out := make([]ec2Stage107CreateFleetInstanceItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage107CreateFleetInstanceItem{
			InstanceIDs:  ec2StringSet{Items: append([]string(nil), item.InstanceIDs...)},
			InstanceType: item.InstanceType,
			Lifecycle:    item.Lifecycle,
		})
	}
	return out
}

func ec2Stage107InstanceConnectEndpointItemFrom(in ec2svc.InstanceConnectEndpoint) ec2Stage107InstanceConnectEndpointItem {
	out := ec2Stage107InstanceConnectEndpointItem{
		AvailabilityZone:           in.AvailabilityZone,
		CreatedAt:                  in.CreatedAt.UTC().Format(time.RFC3339),
		DNSName:                    in.DNSName,
		FipsDNSName:                in.FipsDNSName,
		InstanceConnectEndpointARN: in.InstanceConnectEndpointARN,
		InstanceConnectEndpointID:  in.InstanceConnectEndpointID,
		OwnerID:                    in.OwnerID,
		SecurityGroupIDSet:         ec2StringSet{Items: append([]string(nil), in.SecurityGroupIDs...)},
		State:                      in.State,
		StateMessage:               in.StateMessage,
		SubnetID:                   in.SubnetID,
		TagSet:                     ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		VpcID:                      in.VpcID,
	}
	out.PreserveClientIP = &in.PreserveClientIP
	return out
}

func ec2Stage107InstanceEventWindowItemFrom(in ec2svc.InstanceEventWindow) ec2Stage107InstanceEventWindowItem {
	out := ec2Stage107InstanceEventWindowItem{
		CronExpression:        in.CronExpression,
		InstanceEventWindowID: in.InstanceEventWindowID,
		Name:                  in.Name,
		State:                 in.State,
		TagSet:                ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TimeRangeSet: ec2Stage107InstanceEventWindowTimeRangeSet{
			Items: ec2Stage107InstanceEventWindowTimeRangeItemsFrom(in.TimeRanges),
		},
	}
	return out
}

func ec2Stage107InstanceEventWindowTimeRangeItemsFrom(in []ec2svc.InstanceEventWindowTimeRange) []ec2Stage107InstanceEventWindowTimeRangeItem {
	out := make([]ec2Stage107InstanceEventWindowTimeRangeItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage107InstanceEventWindowTimeRangeItem{
			EndHour:      item.EndHour,
			EndWeekDay:   item.EndWeekDay,
			StartHour:    item.StartHour,
			StartWeekDay: item.StartWeekDay,
		})
	}
	return out
}

func ec2Stage107IpamItemFrom(in ec2svc.Ipam) ec2Stage107IpamItem {
	out := ec2Stage107IpamItem{
		Description: in.Description,
		IpamARN:     in.IpamARN,
		IpamID:      in.IpamID,
		IpamRegion:  in.IpamRegion,
		OperatingRegionSet: ec2Stage107IpamOperatingRegionSet{
			Items: ec2Stage107IpamOperatingRegionItemsFrom(in.OperatingRegions),
		},
		OwnerID: in.OwnerID,
		State:   in.State,
		TagSet:  ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		Tier:    in.Tier,
	}
	return out
}

func ec2Stage107IpamOperatingRegionItemsFrom(in []string) []ec2Stage107IpamOperatingRegionItem {
	out := make([]ec2Stage107IpamOperatingRegionItem, 0, len(in))
	for _, region := range in {
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}
		out = append(out, ec2Stage107IpamOperatingRegionItem{RegionName: region})
	}
	return out
}

type ec2Stage107CreateCoipPoolResponse struct {
	XMLName   xml.Name                `xml:"CreateCoipPoolResponse"`
	Xmlns     string                  `xml:"xmlns,attr"`
	RequestID string                  `xml:"requestId"`
	CoipPool  ec2Stage107CoipPoolItem `xml:"coipPool"`
}

type ec2Stage107CoipPoolItem struct {
	LocalGatewayRouteTableID string       `xml:"localGatewayRouteTableId,omitempty"`
	PoolARN                  string       `xml:"poolArn,omitempty"`
	PoolCidrSet              ec2StringSet `xml:"poolCidrSet"`
	PoolID                   string       `xml:"poolId,omitempty"`
	TagSet                   ec2TagSet    `xml:"tagSet"`
}

type ec2Stage107CreateDelegateMacVolumeOwnershipTaskResponse struct {
	XMLName             xml.Name                           `xml:"CreateDelegateMacVolumeOwnershipTaskResponse"`
	Xmlns               string                             `xml:"xmlns,attr"`
	RequestID           string                             `xml:"requestId"`
	MacModificationTask ec2Stage107MacModificationTaskItem `xml:"macModificationTask"`
}

type ec2Stage107MacModificationTaskItem struct {
	InstanceID            string    `xml:"instanceId,omitempty"`
	MacModificationTaskID string    `xml:"macModificationTaskId,omitempty"`
	StartTime             string    `xml:"startTime,omitempty"`
	TagSet                ec2TagSet `xml:"tagSet"`
	TaskState             string    `xml:"taskState,omitempty"`
	TaskType              string    `xml:"taskType,omitempty"`
}

type ec2Stage107CreateFleetResponse struct {
	XMLName          xml.Name                          `xml:"CreateFleetResponse"`
	Xmlns            string                            `xml:"xmlns,attr"`
	RequestID        string                            `xml:"requestId"`
	ErrorSet         ec2Stage107CreateFleetErrorSet    `xml:"errorSet"`
	FleetID          string                            `xml:"fleetId,omitempty"`
	FleetInstanceSet ec2Stage107CreateFleetInstanceSet `xml:"fleetInstanceSet"`
}

type ec2Stage107CreateFleetErrorSet struct {
	Items []ec2Stage107CreateFleetErrorItem `xml:"item"`
}

type ec2Stage107CreateFleetErrorItem struct {
	ErrorCode    string `xml:"errorCode,omitempty"`
	ErrorMessage string `xml:"errorMessage,omitempty"`
}

type ec2Stage107CreateFleetInstanceSet struct {
	Items []ec2Stage107CreateFleetInstanceItem `xml:"item"`
}

type ec2Stage107CreateFleetInstanceItem struct {
	InstanceIDs  ec2StringSet `xml:"instanceIds"`
	InstanceType string       `xml:"instanceType,omitempty"`
	Lifecycle    string       `xml:"lifecycle,omitempty"`
}

type ec2Stage107CreateFlowLogsResponse struct {
	XMLName      xml.Name               `xml:"CreateFlowLogsResponse"`
	Xmlns        string                 `xml:"xmlns,attr"`
	RequestID    string                 `xml:"requestId"`
	ClientToken  string                 `xml:"clientToken,omitempty"`
	FlowLogIDSet ec2StringSet           `xml:"flowLogIdSet"`
	Unsuccessful ec2UnsuccessfulItemSet `xml:"unsuccessful"`
}

type ec2Stage107CreateFpgaImageResponse struct {
	XMLName           xml.Name `xml:"CreateFpgaImageResponse"`
	Xmlns             string   `xml:"xmlns,attr"`
	RequestID         string   `xml:"requestId"`
	FpgaImageGlobalID string   `xml:"fpgaImageGlobalId,omitempty"`
	FpgaImageID       string   `xml:"fpgaImageId,omitempty"`
}

type ec2Stage107CreateInstanceConnectEndpointResponse struct {
	XMLName                 xml.Name                               `xml:"CreateInstanceConnectEndpointResponse"`
	Xmlns                   string                                 `xml:"xmlns,attr"`
	RequestID               string                                 `xml:"requestId"`
	ClientToken             string                                 `xml:"clientToken,omitempty"`
	InstanceConnectEndpoint ec2Stage107InstanceConnectEndpointItem `xml:"instanceConnectEndpoint"`
}

type ec2Stage107InstanceConnectEndpointItem struct {
	AvailabilityZone           string       `xml:"availabilityZone,omitempty"`
	CreatedAt                  string       `xml:"createdAt,omitempty"`
	DNSName                    string       `xml:"dnsName,omitempty"`
	FipsDNSName                string       `xml:"fipsDnsName,omitempty"`
	InstanceConnectEndpointARN string       `xml:"instanceConnectEndpointArn,omitempty"`
	InstanceConnectEndpointID  string       `xml:"instanceConnectEndpointId,omitempty"`
	OwnerID                    string       `xml:"ownerId,omitempty"`
	PreserveClientIP           *bool        `xml:"preserveClientIp,omitempty"`
	SecurityGroupIDSet         ec2StringSet `xml:"securityGroupIdSet"`
	State                      string       `xml:"state,omitempty"`
	StateMessage               string       `xml:"stateMessage,omitempty"`
	SubnetID                   string       `xml:"subnetId,omitempty"`
	TagSet                     ec2TagSet    `xml:"tagSet"`
	VpcID                      string       `xml:"vpcId,omitempty"`
}

type ec2Stage107CreateInstanceEventWindowResponse struct {
	XMLName             xml.Name                           `xml:"CreateInstanceEventWindowResponse"`
	Xmlns               string                             `xml:"xmlns,attr"`
	RequestID           string                             `xml:"requestId"`
	InstanceEventWindow ec2Stage107InstanceEventWindowItem `xml:"instanceEventWindow"`
}

type ec2Stage107InstanceEventWindowItem struct {
	CronExpression        string                                     `xml:"cronExpression,omitempty"`
	InstanceEventWindowID string                                     `xml:"instanceEventWindowId,omitempty"`
	Name                  string                                     `xml:"name,omitempty"`
	State                 string                                     `xml:"state,omitempty"`
	TagSet                ec2TagSet                                  `xml:"tagSet"`
	TimeRangeSet          ec2Stage107InstanceEventWindowTimeRangeSet `xml:"timeRangeSet"`
}

type ec2Stage107InstanceEventWindowTimeRangeSet struct {
	Items []ec2Stage107InstanceEventWindowTimeRangeItem `xml:"item"`
}

type ec2Stage107InstanceEventWindowTimeRangeItem struct {
	EndHour      *int32 `xml:"endHour,omitempty"`
	EndWeekDay   string `xml:"endWeekDay,omitempty"`
	StartHour    *int32 `xml:"startHour,omitempty"`
	StartWeekDay string `xml:"startWeekDay,omitempty"`
}

type ec2Stage107CreateInstanceExportTaskResponse struct {
	XMLName    xml.Name                  `xml:"CreateInstanceExportTaskResponse"`
	Xmlns      string                    `xml:"xmlns,attr"`
	RequestID  string                    `xml:"requestId"`
	ExportTask ec2Stage107ExportTaskItem `xml:"exportTask"`
}

type ec2Stage107ExportTaskItem struct {
	Description    string                               `xml:"description,omitempty"`
	ExportTaskID   string                               `xml:"exportTaskId,omitempty"`
	ExportToS3     ec2Stage107ExportToS3TaskItem        `xml:"exportToS3"`
	InstanceExport ec2Stage107InstanceExportDetailsItem `xml:"instanceExport"`
	State          string                               `xml:"state,omitempty"`
	StatusMessage  string                               `xml:"statusMessage,omitempty"`
	TagSet         ec2TagSet                            `xml:"tagSet"`
}

type ec2Stage107ExportToS3TaskItem struct {
	ContainerFormat string `xml:"containerFormat,omitempty"`
	DiskImageFormat string `xml:"diskImageFormat,omitempty"`
	S3Bucket        string `xml:"s3Bucket,omitempty"`
	S3Key           string `xml:"s3Key,omitempty"`
}

type ec2Stage107InstanceExportDetailsItem struct {
	InstanceID        string `xml:"instanceId,omitempty"`
	TargetEnvironment string `xml:"targetEnvironment,omitempty"`
}

type ec2Stage107CreateIpamResponse struct {
	XMLName   xml.Name            `xml:"CreateIpamResponse"`
	Xmlns     string              `xml:"xmlns,attr"`
	RequestID string              `xml:"requestId"`
	Ipam      ec2Stage107IpamItem `xml:"ipam"`
}

type ec2Stage107IpamItem struct {
	Description        string                            `xml:"description,omitempty"`
	IpamARN            string                            `xml:"ipamArn,omitempty"`
	IpamID             string                            `xml:"ipamId,omitempty"`
	IpamRegion         string                            `xml:"ipamRegion,omitempty"`
	OperatingRegionSet ec2Stage107IpamOperatingRegionSet `xml:"operatingRegionSet"`
	OwnerID            string                            `xml:"ownerId,omitempty"`
	State              string                            `xml:"state,omitempty"`
	TagSet             ec2TagSet                         `xml:"tagSet"`
	Tier               string                            `xml:"tier,omitempty"`
}

type ec2Stage107IpamOperatingRegionSet struct {
	Items []ec2Stage107IpamOperatingRegionItem `xml:"item"`
}

type ec2Stage107IpamOperatingRegionItem struct {
	RegionName string `xml:"regionName,omitempty"`
}

type ec2Stage107CreateIpamExternalResourceVerificationTokenResponse struct {
	XMLName                               xml.Name                                             `xml:"CreateIpamExternalResourceVerificationTokenResponse"`
	Xmlns                                 string                                               `xml:"xmlns,attr"`
	RequestID                             string                                               `xml:"requestId"`
	IpamExternalResourceVerificationToken ec2Stage107IpamExternalResourceVerificationTokenItem `xml:"ipamExternalResourceVerificationToken"`
}

type ec2Stage107IpamExternalResourceVerificationTokenItem struct {
	IpamARN                                  string    `xml:"ipamArn,omitempty"`
	IpamExternalResourceVerificationTokenARN string    `xml:"ipamExternalResourceVerificationTokenArn,omitempty"`
	IpamExternalResourceVerificationTokenID  string    `xml:"ipamExternalResourceVerificationTokenId,omitempty"`
	IpamID                                   string    `xml:"ipamId,omitempty"`
	IpamRegion                               string    `xml:"ipamRegion,omitempty"`
	NotAfter                                 string    `xml:"notAfter,omitempty"`
	State                                    string    `xml:"state,omitempty"`
	Status                                   string    `xml:"status,omitempty"`
	TagSet                                   ec2TagSet `xml:"tagSet"`
	TokenName                                string    `xml:"tokenName,omitempty"`
}
