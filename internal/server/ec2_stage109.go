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

func (s *Server) handleEC2Stage109Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateLocalGatewayVirtualInterfaceGroup":
		localBgpASN, ok := parseEC2OptionalInt32(r.Form.Get("LocalBgpAsn"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		localBgpASNExtended, ok := parseEC2OptionalInt64(r.Form.Get("LocalBgpAsnExtended"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		group, err := s.ec2.CreateLocalGatewayVirtualInterfaceGroup(
			strings.TrimSpace(r.Form.Get("LocalGatewayId")),
			localBgpASN,
			localBgpASNExtended,
			parseEC2TagSpecificationsForResource(r.Form, "local-gateway-virtual-interface-group"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage109CreateLocalGatewayVirtualInterfaceGroupResponse{
			XMLName:                           xml.Name{Local: "CreateLocalGatewayVirtualInterfaceGroupResponse"},
			Xmlns:                             ec2Namespace,
			RequestID:                         "stackyard-request",
			LocalGatewayVirtualInterfaceGroup: ec2Stage109LocalGatewayVirtualInterfaceGroupItemFrom(group),
		})
		return true
	case "CreateMacSystemIntegrityProtectionModificationTask":
		task, err := s.ec2.CreateMacSystemIntegrityProtectionModificationTask(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("MacSystemIntegrityProtectionStatus")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			parseEC2OptionalString(r.Form.Get("MacCredentials")),
			parseEC2TagSpecificationsForResource(r.Form, "mac-modification-task"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage109CreateMacSystemIntegrityProtectionModificationTaskResponse{
			XMLName:             xml.Name{Local: "CreateMacSystemIntegrityProtectionModificationTaskResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			MacModificationTask: ec2Stage107MacModificationTaskItemFrom(task),
		})
		return true
	case "CreateManagedPrefixList":
		maxEntries, ok := parseEC2OptionalInt32(r.Form.Get("MaxEntries"))
		if !ok || maxEntries == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		prefixList, err := s.ec2.CreateManagedPrefixList(
			strings.TrimSpace(r.Form.Get("AddressFamily")),
			*maxEntries,
			strings.TrimSpace(r.Form.Get("PrefixListName")),
			parseEC2Stage109ManagedPrefixListEntries(r.Form),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			parseEC2TagSpecificationsForResource(r.Form, "prefix-list"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage109CreateManagedPrefixListResponse{
			XMLName:    xml.Name{Local: "CreateManagedPrefixListResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			PrefixList: ec2Stage109ManagedPrefixListItemFrom(prefixList),
		})
		return true
	case "CreateNetworkInsightsAccessScope":
		scope, err := s.ec2.CreateNetworkInsightsAccessScope(
			strings.TrimSpace(r.Form.Get("ClientToken")),
			parseEC2TagSpecificationsForResource(r.Form, "network-insights-access-scope"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage109CreateNetworkInsightsAccessScopeResponse{
			XMLName:                    xml.Name{Local: "CreateNetworkInsightsAccessScopeResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			NetworkInsightsAccessScope: ec2Stage109NetworkInsightsAccessScopeItemFrom(scope),
		})
		return true
	case "CreateNetworkInsightsPath":
		destinationPort, ok := parseEC2OptionalInt32(r.Form.Get("DestinationPort"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		path, err := s.ec2.CreateNetworkInsightsPath(
			strings.TrimSpace(r.Form.Get("ClientToken")),
			strings.TrimSpace(r.Form.Get("Protocol")),
			strings.TrimSpace(r.Form.Get("Source")),
			parseEC2OptionalString(r.Form.Get("Destination")),
			parseEC2OptionalString(r.Form.Get("DestinationIp")),
			destinationPort,
			parseEC2OptionalString(r.Form.Get("SourceIp")),
			parseEC2TagSpecificationsForResource(r.Form, "network-insights-path"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage109CreateNetworkInsightsPathResponse{
			XMLName:             xml.Name{Local: "CreateNetworkInsightsPathResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			NetworkInsightsPath: ec2Stage109NetworkInsightsPathItemFrom(path),
		})
		return true
	case "CreatePublicIpv4Pool":
		pool, err := s.ec2.CreatePublicIpv4Pool(
			parseEC2OptionalString(r.Form.Get("NetworkBorderGroup")),
			parseEC2TagSpecificationsForResource(r.Form, "public-ipv4-pool"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage109CreatePublicIpv4PoolResponse{
			XMLName:   xml.Name{Local: "CreatePublicIpv4PoolResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			PoolID:    pool.PoolID,
		})
		return true
	case "CreateReplaceRootVolumeTask":
		deleteReplacedRootVolume, hasDeleteReplacedRootVolume, ok := ec2OptionalBoolFromForm(r.Form, "DeleteReplacedRootVolume")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasDeleteReplacedRootVolume {
			deleteReplacedRootVolume = nil
		}
		volumeInitializationRate, ok := parseEC2OptionalInt64(r.Form.Get("VolumeInitializationRate"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		task, err := s.ec2.CreateReplaceRootVolumeTask(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			deleteReplacedRootVolume,
			parseEC2OptionalString(r.Form.Get("ImageId")),
			parseEC2OptionalString(r.Form.Get("SnapshotId")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			volumeInitializationRate,
			parseEC2TagSpecificationsForResource(r.Form, "replace-root-volume-task"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage109CreateReplaceRootVolumeTaskResponse{
			XMLName:               xml.Name{Local: "CreateReplaceRootVolumeTaskResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			ReplaceRootVolumeTask: ec2Stage109ReplaceRootVolumeTaskItemFrom(task),
		})
		return true
	case "CreateReservedInstancesListing":
		instanceCount, ok := parseEC2OptionalInt32(r.Form.Get("InstanceCount"))
		if !ok || instanceCount == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		priceSchedules, ok := parseEC2Stage109PriceSchedules(r.Form)
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		listings, err := s.ec2.CreateReservedInstancesListing(
			strings.TrimSpace(r.Form.Get("ClientToken")),
			*instanceCount,
			strings.TrimSpace(r.Form.Get("ReservedInstancesId")),
			priceSchedules,
			parseEC2TagSpecificationsForResource(r.Form, "reserved-instances-listing"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage109CreateReservedInstancesListingResponse{
			XMLName:                   xml.Name{Local: "CreateReservedInstancesListingResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			ReservedInstancesListings: ec2Stage95ReservedInstancesListingSet{Items: ec2Stage95ReservedInstancesListingsFrom(listings)},
		})
		return true
	case "CreateRestoreImageTask":
		imageID, err := s.ec2.CreateRestoreImageTask(
			strings.TrimSpace(r.Form.Get("Bucket")),
			strings.TrimSpace(r.Form.Get("ObjectKey")),
			parseEC2OptionalString(r.Form.Get("Name")),
			parseEC2TagSpecificationsForResource(r.Form, "image"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage109CreateRestoreImageTaskResponse{
			XMLName:   xml.Name{Local: "CreateRestoreImageTaskResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ImageID:   imageID,
		})
		return true
	case "CreateSnapshots":
		snapshots, err := s.ec2.CreateSnapshots(
			strings.TrimSpace(r.Form.Get("InstanceSpecification.InstanceId")),
			parseEC2OptionalString(r.Form.Get("Description")),
			strings.TrimSpace(r.Form.Get("CopyTagsFromSource")),
			parseEC2TagSpecificationsForResource(r.Form, "snapshot"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage109CreateSnapshotsResponse{
			XMLName:     xml.Name{Local: "CreateSnapshotsResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			SnapshotSet: ec2SnapshotSet{Items: ec2SnapshotItems(snapshots)},
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage109ManagedPrefixListEntries(values url.Values) []ec2svc.ManagedPrefixListEntry {
	entriesByIndex := map[int]ec2svc.ManagedPrefixListEntry{}
	for key := range values {
		if !strings.HasPrefix(key, "Entry.") {
			continue
		}
		rest := strings.TrimPrefix(key, "Entry.")
		parts := strings.SplitN(rest, ".", 2)
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil || idx <= 0 {
			continue
		}
		entry := entriesByIndex[idx]
		switch parts[1] {
		case "Cidr":
			entry.CIDR = strings.TrimSpace(values.Get(key))
		case "Description":
			entry.Description = strings.TrimSpace(values.Get(key))
		}
		entriesByIndex[idx] = entry
	}

	if len(entriesByIndex) == 0 {
		return nil
	}
	indices := make([]int, 0, len(entriesByIndex))
	for idx := range entriesByIndex {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	out := make([]ec2svc.ManagedPrefixListEntry, 0, len(indices))
	for _, idx := range indices {
		entry := entriesByIndex[idx]
		if strings.TrimSpace(entry.CIDR) == "" {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func parseEC2Stage109PriceSchedules(values url.Values) ([]ec2svc.ReservedInstancesPriceSchedule, bool) {
	type rawPriceSchedule struct {
		currencyCode string
		price        *float64
		term         *int64
	}
	byIndex := map[int]rawPriceSchedule{}
	for key := range values {
		if !strings.HasPrefix(key, "PriceSchedules.") {
			continue
		}
		rest := strings.TrimPrefix(key, "PriceSchedules.")
		parts := strings.SplitN(rest, ".", 2)
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil || idx <= 0 {
			continue
		}
		raw := byIndex[idx]
		switch parts[1] {
		case "CurrencyCode":
			raw.currencyCode = strings.TrimSpace(values.Get(key))
		case "Price":
			rawValue := strings.TrimSpace(values.Get(key))
			if rawValue == "" {
				break
			}
			v, err := strconv.ParseFloat(rawValue, 64)
			if err != nil {
				return nil, false
			}
			raw.price = &v
		case "Term":
			rawValue := strings.TrimSpace(values.Get(key))
			if rawValue == "" {
				break
			}
			v, err := strconv.ParseInt(rawValue, 10, 64)
			if err != nil {
				return nil, false
			}
			raw.term = &v
		}
		byIndex[idx] = raw
	}

	if len(byIndex) == 0 {
		return nil, false
	}
	indices := make([]int, 0, len(byIndex))
	for idx := range byIndex {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	out := make([]ec2svc.ReservedInstancesPriceSchedule, 0, len(indices))
	for _, idx := range indices {
		raw := byIndex[idx]
		if raw.price == nil || raw.term == nil {
			return nil, false
		}
		out = append(out, ec2svc.ReservedInstancesPriceSchedule{
			CurrencyCode: raw.currencyCode,
			Price:        *raw.price,
			Term:         *raw.term,
		})
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func ec2Stage109LocalGatewayVirtualInterfaceGroupItemFrom(in ec2svc.LocalGatewayVirtualInterfaceGroup) ec2Stage109LocalGatewayVirtualInterfaceGroupItem {
	return ec2Stage109LocalGatewayVirtualInterfaceGroupItem{
		LocalGatewayID:                      in.LocalGatewayID,
		LocalGatewayVirtualInterfaceGroupID: in.LocalGatewayVirtualInterfaceGroupID,
		LocalGatewayVirtualInterfaceIDSet:   ec2StringSet{Items: in.LocalGatewayVirtualInterfaceIDs},
		OwnerID:                             in.OwnerID,
		TagSet:                              ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
	}
}

func ec2Stage109ManagedPrefixListItemFrom(in ec2svc.ManagedPrefixList) ec2Stage109ManagedPrefixListItem {
	return ec2Stage109ManagedPrefixListItem{
		AddressFamily:  in.AddressFamily,
		MaxEntries:     in.MaxEntries,
		OwnerID:        in.OwnerID,
		PrefixListARN:  in.PrefixListARN,
		PrefixListID:   in.PrefixListID,
		PrefixListName: in.PrefixListName,
		State:          in.State,
		StateMessage:   in.StateMessage,
		TagSet:         ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		Version:        in.Version,
	}
}

func ec2Stage109NetworkInsightsAccessScopeItemFrom(in ec2svc.NetworkInsightsAccessScope) ec2Stage109NetworkInsightsAccessScopeItem {
	item := ec2Stage109NetworkInsightsAccessScopeItem{
		NetworkInsightsAccessScopeARN: in.NetworkInsightsAccessScopeARN,
		NetworkInsightsAccessScopeID:  in.NetworkInsightsAccessScopeID,
		TagSet:                        ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
	}
	if !in.CreatedDate.IsZero() {
		item.CreatedDate = in.CreatedDate.UTC().Format(time.RFC3339)
	}
	if !in.UpdatedDate.IsZero() {
		item.UpdatedDate = in.UpdatedDate.UTC().Format(time.RFC3339)
	}
	return item
}

func ec2Stage109NetworkInsightsPathItemFrom(in ec2svc.NetworkInsightsPath) ec2Stage109NetworkInsightsPathItem {
	item := ec2Stage109NetworkInsightsPathItem{
		Destination:            in.Destination,
		DestinationIP:          in.DestinationIP,
		DestinationPort:        in.DestinationPort,
		NetworkInsightsPathARN: in.NetworkInsightsPathARN,
		NetworkInsightsPathID:  in.NetworkInsightsPathID,
		Protocol:               in.Protocol,
		Source:                 in.Source,
		SourceIP:               in.SourceIP,
		TagSet:                 ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
	}
	if !in.CreatedDate.IsZero() {
		item.CreatedDate = in.CreatedDate.UTC().Format(time.RFC3339)
	}
	return item
}

func ec2Stage109ReplaceRootVolumeTaskItemFrom(in ec2svc.ReplaceRootVolumeTask) ec2Stage109ReplaceRootVolumeTaskItem {
	return ec2Stage109ReplaceRootVolumeTaskItem{
		CompleteTime:             in.CompleteTime,
		DeleteReplacedRootVolume: in.DeleteReplacedRootVolume,
		ImageID:                  in.ImageID,
		InstanceID:               in.InstanceID,
		ReplaceRootVolumeTaskID:  in.ReplaceRootVolumeTaskID,
		SnapshotID:               in.SnapshotID,
		StartTime:                in.StartTime,
		TagSet:                   ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TaskState:                in.TaskState,
	}
}

type ec2Stage109CreateLocalGatewayVirtualInterfaceGroupResponse struct {
	XMLName                           xml.Name                                         `xml:"CreateLocalGatewayVirtualInterfaceGroupResponse"`
	Xmlns                             string                                           `xml:"xmlns,attr"`
	RequestID                         string                                           `xml:"requestId"`
	LocalGatewayVirtualInterfaceGroup ec2Stage109LocalGatewayVirtualInterfaceGroupItem `xml:"localGatewayVirtualInterfaceGroup"`
}

type ec2Stage109CreateMacSystemIntegrityProtectionModificationTaskResponse struct {
	XMLName             xml.Name                           `xml:"CreateMacSystemIntegrityProtectionModificationTaskResponse"`
	Xmlns               string                             `xml:"xmlns,attr"`
	RequestID           string                             `xml:"requestId"`
	MacModificationTask ec2Stage107MacModificationTaskItem `xml:"macModificationTask"`
}

type ec2Stage109CreateManagedPrefixListResponse struct {
	XMLName    xml.Name                         `xml:"CreateManagedPrefixListResponse"`
	Xmlns      string                           `xml:"xmlns,attr"`
	RequestID  string                           `xml:"requestId"`
	PrefixList ec2Stage109ManagedPrefixListItem `xml:"prefixList"`
}

type ec2Stage109CreateNetworkInsightsAccessScopeResponse struct {
	XMLName                    xml.Name                                  `xml:"CreateNetworkInsightsAccessScopeResponse"`
	Xmlns                      string                                    `xml:"xmlns,attr"`
	RequestID                  string                                    `xml:"requestId"`
	NetworkInsightsAccessScope ec2Stage109NetworkInsightsAccessScopeItem `xml:"networkInsightsAccessScope"`
}

type ec2Stage109CreateNetworkInsightsPathResponse struct {
	XMLName             xml.Name                           `xml:"CreateNetworkInsightsPathResponse"`
	Xmlns               string                             `xml:"xmlns,attr"`
	RequestID           string                             `xml:"requestId"`
	NetworkInsightsPath ec2Stage109NetworkInsightsPathItem `xml:"networkInsightsPath"`
}

type ec2Stage109CreatePublicIpv4PoolResponse struct {
	XMLName   xml.Name `xml:"CreatePublicIpv4PoolResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	PoolID    string   `xml:"poolId"`
}

type ec2Stage109CreateReplaceRootVolumeTaskResponse struct {
	XMLName               xml.Name                             `xml:"CreateReplaceRootVolumeTaskResponse"`
	Xmlns                 string                               `xml:"xmlns,attr"`
	RequestID             string                               `xml:"requestId"`
	ReplaceRootVolumeTask ec2Stage109ReplaceRootVolumeTaskItem `xml:"replaceRootVolumeTask"`
}

type ec2Stage109CreateReservedInstancesListingResponse struct {
	XMLName                   xml.Name                              `xml:"CreateReservedInstancesListingResponse"`
	Xmlns                     string                                `xml:"xmlns,attr"`
	RequestID                 string                                `xml:"requestId"`
	ReservedInstancesListings ec2Stage95ReservedInstancesListingSet `xml:"reservedInstancesListingsSet"`
}

type ec2Stage109CreateRestoreImageTaskResponse struct {
	XMLName   xml.Name `xml:"CreateRestoreImageTaskResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	ImageID   string   `xml:"imageId"`
}

type ec2Stage109CreateSnapshotsResponse struct {
	XMLName     xml.Name       `xml:"CreateSnapshotsResponse"`
	Xmlns       string         `xml:"xmlns,attr"`
	RequestID   string         `xml:"requestId"`
	SnapshotSet ec2SnapshotSet `xml:"snapshotSet"`
}

type ec2Stage109LocalGatewayVirtualInterfaceGroupItem struct {
	LocalGatewayID                      string       `xml:"localGatewayId,omitempty"`
	LocalGatewayVirtualInterfaceGroupID string       `xml:"localGatewayVirtualInterfaceGroupId,omitempty"`
	LocalGatewayVirtualInterfaceIDSet   ec2StringSet `xml:"localGatewayVirtualInterfaceIdSet"`
	OwnerID                             string       `xml:"ownerId,omitempty"`
	TagSet                              ec2TagSet    `xml:"tagSet"`
}

type ec2Stage109ManagedPrefixListItem struct {
	AddressFamily  string    `xml:"addressFamily,omitempty"`
	MaxEntries     int32     `xml:"maxEntries,omitempty"`
	OwnerID        string    `xml:"ownerId,omitempty"`
	PrefixListARN  string    `xml:"prefixListArn,omitempty"`
	PrefixListID   string    `xml:"prefixListId,omitempty"`
	PrefixListName string    `xml:"prefixListName,omitempty"`
	State          string    `xml:"state,omitempty"`
	StateMessage   string    `xml:"stateMessage,omitempty"`
	TagSet         ec2TagSet `xml:"tagSet"`
	Version        int64     `xml:"version,omitempty"`
}

type ec2Stage109NetworkInsightsAccessScopeItem struct {
	CreatedDate                   string    `xml:"createdDate,omitempty"`
	NetworkInsightsAccessScopeARN string    `xml:"networkInsightsAccessScopeArn,omitempty"`
	NetworkInsightsAccessScopeID  string    `xml:"networkInsightsAccessScopeId,omitempty"`
	TagSet                        ec2TagSet `xml:"tagSet"`
	UpdatedDate                   string    `xml:"updatedDate,omitempty"`
}

type ec2Stage109NetworkInsightsPathItem struct {
	CreatedDate            string    `xml:"createdDate,omitempty"`
	Destination            string    `xml:"destination,omitempty"`
	DestinationIP          string    `xml:"destinationIp,omitempty"`
	DestinationPort        *int32    `xml:"destinationPort,omitempty"`
	NetworkInsightsPathARN string    `xml:"networkInsightsPathArn,omitempty"`
	NetworkInsightsPathID  string    `xml:"networkInsightsPathId,omitempty"`
	Protocol               string    `xml:"protocol,omitempty"`
	Source                 string    `xml:"source,omitempty"`
	SourceIP               string    `xml:"sourceIp,omitempty"`
	TagSet                 ec2TagSet `xml:"tagSet"`
}

type ec2Stage109ReplaceRootVolumeTaskItem struct {
	CompleteTime             string    `xml:"completeTime,omitempty"`
	DeleteReplacedRootVolume *bool     `xml:"deleteReplacedRootVolume,omitempty"`
	ImageID                  string    `xml:"imageId,omitempty"`
	InstanceID               string    `xml:"instanceId,omitempty"`
	ReplaceRootVolumeTaskID  string    `xml:"replaceRootVolumeTaskId,omitempty"`
	SnapshotID               string    `xml:"snapshotId,omitempty"`
	StartTime                string    `xml:"startTime,omitempty"`
	TagSet                   ec2TagSet `xml:"tagSet"`
	TaskState                string    `xml:"taskState,omitempty"`
}
