package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage120Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeManagedPrefixLists":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		prefixLists, nextToken, err := s.ec2.DescribeManagedPrefixLists(
			parseEC2MembersOrItemList(r.Form, "PrefixListId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage120DescribeManagedPrefixListsResponse{
			XMLName:     xml.Name{Local: "DescribeManagedPrefixListsResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			PrefixLists: ec2Stage120ManagedPrefixListSet{Items: ec2Stage120ManagedPrefixListItemsFrom(prefixLists)},
			NextToken:   nextToken,
		})
		return true
	case "DescribeNetworkInsightsAccessScopeAnalyses":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		analysisStartTimeBegin, ok := parseEC2Stage120OptionalRFC3339Time(r.Form, "AnalysisStartTimeBegin")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		analysisStartTimeEnd, ok := parseEC2Stage120OptionalRFC3339Time(r.Form, "AnalysisStartTimeEnd")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		analyses, nextToken, err := s.ec2.DescribeNetworkInsightsAccessScopeAnalyses(
			parseEC2MembersOrItemList(r.Form, "NetworkInsightsAccessScopeAnalysisId"),
			parseEC2OptionalString(r.Form.Get("NetworkInsightsAccessScopeId")),
			analysisStartTimeBegin,
			analysisStartTimeEnd,
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage120DescribeNetworkInsightsAccessScopeAnalysesResponse{
			XMLName:                            xml.Name{Local: "DescribeNetworkInsightsAccessScopeAnalysesResponse"},
			Xmlns:                              ec2Namespace,
			RequestID:                          "stackyard-request",
			NetworkInsightsAccessScopeAnalyses: ec2Stage120NetworkInsightsAccessScopeAnalysisSet{Items: ec2Stage120NetworkInsightsAccessScopeAnalysisItemsFrom(analyses)},
			NextToken:                          nextToken,
		})
		return true
	case "DescribeNetworkInsightsAccessScopes":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		scopes, nextToken, err := s.ec2.DescribeNetworkInsightsAccessScopes(
			parseEC2MembersOrItemList(r.Form, "NetworkInsightsAccessScopeId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage120DescribeNetworkInsightsAccessScopesResponse{
			XMLName:                     xml.Name{Local: "DescribeNetworkInsightsAccessScopesResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			NetworkInsightsAccessScopes: ec2Stage120NetworkInsightsAccessScopeSet{Items: ec2Stage120NetworkInsightsAccessScopeItemsFrom(scopes)},
			NextToken:                   nextToken,
		})
		return true
	case "DescribeNetworkInsightsAnalyses":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		analysisStartTime, ok := parseEC2Stage120OptionalRFC3339Time(r.Form, "AnalysisStartTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		analysisEndTime, ok := parseEC2Stage120OptionalRFC3339Time(r.Form, "AnalysisEndTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		analyses, nextToken, err := s.ec2.DescribeNetworkInsightsAnalyses(
			parseEC2MembersOrItemList(r.Form, "NetworkInsightsAnalysisId"),
			parseEC2OptionalString(r.Form.Get("NetworkInsightsPathId")),
			analysisStartTime,
			analysisEndTime,
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage120DescribeNetworkInsightsAnalysesResponse{
			XMLName:                 xml.Name{Local: "DescribeNetworkInsightsAnalysesResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			NetworkInsightsAnalyses: ec2Stage120NetworkInsightsAnalysisSet{Items: ec2Stage120NetworkInsightsAnalysisItemsFrom(analyses)},
			NextToken:               nextToken,
		})
		return true
	case "DescribeNetworkInsightsPaths":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		paths, nextToken, err := s.ec2.DescribeNetworkInsightsPaths(
			parseEC2MembersOrItemList(r.Form, "NetworkInsightsPathId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage120DescribeNetworkInsightsPathsResponse{
			XMLName:              xml.Name{Local: "DescribeNetworkInsightsPathsResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			NetworkInsightsPaths: ec2Stage120NetworkInsightsPathSet{Items: ec2Stage120NetworkInsightsPathItemsFrom(paths)},
			NextToken:            nextToken,
		})
		return true
	case "DescribeOutpostLags":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		outpostLags, nextToken, err := s.ec2.DescribeOutpostLags(
			parseEC2MembersOrItemList(r.Form, "OutpostLagId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage120DescribeOutpostLagsResponse{
			XMLName:     xml.Name{Local: "DescribeOutpostLagsResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			OutpostLags: ec2Stage120OutpostLagSet{Items: ec2Stage120OutpostLagItemsFrom(outpostLags)},
			NextToken:   nextToken,
		})
		return true
	case "DescribePrefixLists":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		prefixLists, nextToken, err := s.ec2.DescribePrefixLists(
			parseEC2MembersOrItemList(r.Form, "PrefixListId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage120DescribePrefixListsResponse{
			XMLName:     xml.Name{Local: "DescribePrefixListsResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			PrefixLists: ec2Stage120PrefixListSet{Items: ec2Stage120PrefixListItemsFrom(prefixLists)},
			NextToken:   nextToken,
		})
		return true
	case "DescribePublicIpv4Pools":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		pools, nextToken, err := s.ec2.DescribePublicIpv4Pools(
			parseEC2MembersOrItemList(r.Form, "PoolId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage120DescribePublicIpv4PoolsResponse{
			XMLName:         xml.Name{Local: "DescribePublicIpv4PoolsResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			PublicIpv4Pools: ec2Stage120PublicIpv4PoolSet{Items: ec2Stage120PublicIpv4PoolItemsFrom(pools)},
			NextToken:       nextToken,
		})
		return true
	case "DescribeReplaceRootVolumeTasks":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		tasks, nextToken, err := s.ec2.DescribeReplaceRootVolumeTasks(
			parseEC2MembersOrItemList(r.Form, "ReplaceRootVolumeTaskId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage120DescribeReplaceRootVolumeTasksResponse{
			XMLName:                xml.Name{Local: "DescribeReplaceRootVolumeTasksResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			ReplaceRootVolumeTasks: ec2Stage120ReplaceRootVolumeTaskSet{Items: ec2Stage120ReplaceRootVolumeTaskItemsFrom(tasks)},
			NextToken:              nextToken,
		})
		return true
	case "DescribeReservedInstances":
		reservedInstances, err := s.ec2.DescribeReservedInstances(
			parseEC2MembersOrItemList(r.Form, "ReservedInstancesId"),
			parseEC2Filters(r.Form),
			strings.TrimSpace(r.Form.Get("OfferingClass")),
			strings.TrimSpace(r.Form.Get("OfferingType")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage120DescribeReservedInstancesResponse{
			XMLName:           xml.Name{Local: "DescribeReservedInstancesResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			ReservedInstances: ec2Stage120ReservedInstanceSet{Items: ec2Stage120ReservedInstanceItemsFrom(reservedInstances)},
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage120OptionalRFC3339Time(values url.Values, key string) (*time.Time, bool) {
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

func ec2Stage120ManagedPrefixListItemsFrom(in []ec2svc.ManagedPrefixList) []ec2Stage109ManagedPrefixListItem {
	out := make([]ec2Stage109ManagedPrefixListItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage109ManagedPrefixListItemFrom(item))
	}
	return out
}

func ec2Stage120NetworkInsightsAccessScopeAnalysisItemsFrom(in []ec2svc.NetworkInsightsAccessScopeAnalysis) []ec2Stage120NetworkInsightsAccessScopeAnalysisItem {
	out := make([]ec2Stage120NetworkInsightsAccessScopeAnalysisItem, 0, len(in))
	for _, item := range in {
		entry := ec2Stage120NetworkInsightsAccessScopeAnalysisItem{
			AnalyzedENICount:                      item.AnalyzedENICount,
			FindingsFound:                         item.FindingsFound,
			NetworkInsightsAccessScopeAnalysisARN: item.NetworkInsightsAccessScopeAnalysisARN,
			NetworkInsightsAccessScopeAnalysisID:  item.NetworkInsightsAccessScopeAnalysisID,
			NetworkInsightsAccessScopeID:          item.NetworkInsightsAccessScopeID,
			Status:                                item.Status,
			StatusMessage:                         item.StatusMessage,
			TagSet:                                ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			WarningMessage:                        item.WarningMessage,
		}
		if item.StartDate != nil && !item.StartDate.IsZero() {
			entry.StartDate = item.StartDate.UTC().Format(time.RFC3339)
		}
		if item.EndDate != nil && !item.EndDate.IsZero() {
			entry.EndDate = item.EndDate.UTC().Format(time.RFC3339)
		}
		out = append(out, entry)
	}
	return out
}

func ec2Stage120NetworkInsightsAccessScopeItemsFrom(in []ec2svc.NetworkInsightsAccessScope) []ec2Stage109NetworkInsightsAccessScopeItem {
	out := make([]ec2Stage109NetworkInsightsAccessScopeItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage109NetworkInsightsAccessScopeItemFrom(item))
	}
	return out
}

func ec2Stage120NetworkInsightsAnalysisItemsFrom(in []ec2svc.NetworkInsightsAnalysis) []ec2Stage120NetworkInsightsAnalysisItem {
	out := make([]ec2Stage120NetworkInsightsAnalysisItem, 0, len(in))
	for _, item := range in {
		entry := ec2Stage120NetworkInsightsAnalysisItem{
			NetworkInsightsAnalysisARN: item.NetworkInsightsAnalysisARN,
			NetworkInsightsAnalysisID:  item.NetworkInsightsAnalysisID,
			NetworkInsightsPathID:      item.NetworkInsightsPathID,
			NetworkPathFound:           item.NetworkPathFound,
			Status:                     item.Status,
			StatusMessage:              item.StatusMessage,
			TagSet:                     ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			WarningMessage:             item.WarningMessage,
		}
		if item.StartDate != nil && !item.StartDate.IsZero() {
			entry.StartDate = item.StartDate.UTC().Format(time.RFC3339)
		}
		out = append(out, entry)
	}
	return out
}

func ec2Stage120NetworkInsightsPathItemsFrom(in []ec2svc.NetworkInsightsPath) []ec2Stage109NetworkInsightsPathItem {
	out := make([]ec2Stage109NetworkInsightsPathItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage109NetworkInsightsPathItemFrom(item))
	}
	return out
}

func ec2Stage120OutpostLagItemsFrom(in []ec2svc.OutpostLag) []ec2Stage120OutpostLagItem {
	out := make([]ec2Stage120OutpostLagItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage120OutpostLagItem{
			LocalGatewayVirtualInterfaceIDs: ec2StringSet{Items: append([]string(nil), item.LocalGatewayVirtualInterfaceIDs...)},
			OutpostARN:                      item.OutpostARN,
			OutpostLagID:                    item.OutpostLagID,
			OwnerID:                         item.OwnerID,
			ServiceLinkVirtualInterfaceIDs:  ec2StringSet{Items: append([]string(nil), item.ServiceLinkVirtualInterfaceIDs...)},
			State:                           item.State,
			TagSet:                          ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage120PrefixListItemsFrom(in []ec2svc.PrefixList) []ec2Stage120PrefixListItem {
	out := make([]ec2Stage120PrefixListItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage120PrefixListItem{
			CIDRSet:        ec2StringSet{Items: append([]string(nil), item.CIDRs...)},
			PrefixListID:   item.PrefixListID,
			PrefixListName: item.PrefixListName,
		})
	}
	return out
}

func ec2Stage120PublicIpv4PoolItemsFrom(in []ec2svc.PublicIpv4Pool) []ec2Stage120PublicIpv4PoolItem {
	out := make([]ec2Stage120PublicIpv4PoolItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage120PublicIpv4PoolItem{
			NetworkBorderGroup: item.NetworkBorderGroup,
			PoolAddressRanges:  ec2Stage120PublicIpv4PoolRangeSet{},
			PoolID:             item.PoolID,
			TagSet:             ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage120ReplaceRootVolumeTaskItemsFrom(in []ec2svc.ReplaceRootVolumeTask) []ec2Stage109ReplaceRootVolumeTaskItem {
	out := make([]ec2Stage109ReplaceRootVolumeTaskItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage109ReplaceRootVolumeTaskItemFrom(item))
	}
	return out
}

func ec2Stage120ReservedInstanceItemsFrom(in []ec2svc.ReservedInstance) []ec2Stage120ReservedInstanceItem {
	out := make([]ec2Stage120ReservedInstanceItem, 0, len(in))
	for _, item := range in {
		entry := ec2Stage120ReservedInstanceItem{
			AvailabilityZone:    item.AvailabilityZone,
			AvailabilityZoneID:  item.AvailabilityZoneID,
			CurrencyCode:        item.CurrencyCode,
			Duration:            item.Duration,
			FixedPrice:          item.FixedPrice,
			InstanceCount:       item.InstanceCount,
			InstanceTenancy:     item.InstanceTenancy,
			InstanceType:        item.InstanceType,
			OfferingClass:       item.OfferingClass,
			OfferingType:        item.OfferingType,
			ProductDescription:  item.ProductDescription,
			RecurringCharges:    ec2Stage120RecurringChargeSet{Items: ec2Stage120RecurringChargeItemsFrom(item.RecurringCharges)},
			ReservedInstancesID: item.ReservedInstancesID,
			Scope:               item.Scope,
			State:               item.State,
			TagSet:              ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			UsagePrice:          item.UsagePrice,
		}
		if item.Start != nil && !item.Start.IsZero() {
			entry.Start = item.Start.UTC().Format(time.RFC3339)
		}
		if item.End != nil && !item.End.IsZero() {
			entry.End = item.End.UTC().Format(time.RFC3339)
		}
		out = append(out, entry)
	}
	return out
}

func ec2Stage120RecurringChargeItemsFrom(in []ec2svc.ReservedInstanceRecurringCharge) []ec2Stage120RecurringChargeItem {
	out := make([]ec2Stage120RecurringChargeItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage120RecurringChargeItem{
			Amount:    item.Amount,
			Frequency: item.Frequency,
		})
	}
	return out
}

type ec2Stage120DescribeManagedPrefixListsResponse struct {
	XMLName     xml.Name                        `xml:"DescribeManagedPrefixListsResponse"`
	Xmlns       string                          `xml:"xmlns,attr"`
	RequestID   string                          `xml:"requestId"`
	PrefixLists ec2Stage120ManagedPrefixListSet `xml:"prefixListSet"`
	NextToken   *string                         `xml:"nextToken,omitempty"`
}

type ec2Stage120ManagedPrefixListSet struct {
	Items []ec2Stage109ManagedPrefixListItem `xml:"item"`
}

type ec2Stage120DescribeNetworkInsightsAccessScopeAnalysesResponse struct {
	XMLName                            xml.Name                                         `xml:"DescribeNetworkInsightsAccessScopeAnalysesResponse"`
	Xmlns                              string                                           `xml:"xmlns,attr"`
	RequestID                          string                                           `xml:"requestId"`
	NetworkInsightsAccessScopeAnalyses ec2Stage120NetworkInsightsAccessScopeAnalysisSet `xml:"networkInsightsAccessScopeAnalysisSet"`
	NextToken                          *string                                          `xml:"nextToken,omitempty"`
}

type ec2Stage120NetworkInsightsAccessScopeAnalysisSet struct {
	Items []ec2Stage120NetworkInsightsAccessScopeAnalysisItem `xml:"item"`
}

type ec2Stage120NetworkInsightsAccessScopeAnalysisItem struct {
	AnalyzedENICount                      *int32    `xml:"analyzedEniCount,omitempty"`
	EndDate                               string    `xml:"endDate,omitempty"`
	FindingsFound                         string    `xml:"findingsFound,omitempty"`
	NetworkInsightsAccessScopeAnalysisARN string    `xml:"networkInsightsAccessScopeAnalysisArn,omitempty"`
	NetworkInsightsAccessScopeAnalysisID  string    `xml:"networkInsightsAccessScopeAnalysisId,omitempty"`
	NetworkInsightsAccessScopeID          string    `xml:"networkInsightsAccessScopeId,omitempty"`
	StartDate                             string    `xml:"startDate,omitempty"`
	Status                                string    `xml:"status,omitempty"`
	StatusMessage                         string    `xml:"statusMessage,omitempty"`
	TagSet                                ec2TagSet `xml:"tagSet"`
	WarningMessage                        string    `xml:"warningMessage,omitempty"`
}

type ec2Stage120DescribeNetworkInsightsAccessScopesResponse struct {
	XMLName                     xml.Name                                 `xml:"DescribeNetworkInsightsAccessScopesResponse"`
	Xmlns                       string                                   `xml:"xmlns,attr"`
	RequestID                   string                                   `xml:"requestId"`
	NetworkInsightsAccessScopes ec2Stage120NetworkInsightsAccessScopeSet `xml:"networkInsightsAccessScopeSet"`
	NextToken                   *string                                  `xml:"nextToken,omitempty"`
}

type ec2Stage120NetworkInsightsAccessScopeSet struct {
	Items []ec2Stage109NetworkInsightsAccessScopeItem `xml:"item"`
}

type ec2Stage120DescribeNetworkInsightsAnalysesResponse struct {
	XMLName                 xml.Name                              `xml:"DescribeNetworkInsightsAnalysesResponse"`
	Xmlns                   string                                `xml:"xmlns,attr"`
	RequestID               string                                `xml:"requestId"`
	NetworkInsightsAnalyses ec2Stage120NetworkInsightsAnalysisSet `xml:"networkInsightsAnalysisSet"`
	NextToken               *string                               `xml:"nextToken,omitempty"`
}

type ec2Stage120NetworkInsightsAnalysisSet struct {
	Items []ec2Stage120NetworkInsightsAnalysisItem `xml:"item"`
}

type ec2Stage120NetworkInsightsAnalysisItem struct {
	NetworkInsightsAnalysisARN string    `xml:"networkInsightsAnalysisArn,omitempty"`
	NetworkInsightsAnalysisID  string    `xml:"networkInsightsAnalysisId,omitempty"`
	NetworkInsightsPathID      string    `xml:"networkInsightsPathId,omitempty"`
	NetworkPathFound           *bool     `xml:"networkPathFound,omitempty"`
	StartDate                  string    `xml:"startDate,omitempty"`
	Status                     string    `xml:"status,omitempty"`
	StatusMessage              string    `xml:"statusMessage,omitempty"`
	TagSet                     ec2TagSet `xml:"tagSet"`
	WarningMessage             string    `xml:"warningMessage,omitempty"`
}

type ec2Stage120DescribeNetworkInsightsPathsResponse struct {
	XMLName              xml.Name                          `xml:"DescribeNetworkInsightsPathsResponse"`
	Xmlns                string                            `xml:"xmlns,attr"`
	RequestID            string                            `xml:"requestId"`
	NetworkInsightsPaths ec2Stage120NetworkInsightsPathSet `xml:"networkInsightsPathSet"`
	NextToken            *string                           `xml:"nextToken,omitempty"`
}

type ec2Stage120NetworkInsightsPathSet struct {
	Items []ec2Stage109NetworkInsightsPathItem `xml:"item"`
}

type ec2Stage120DescribeOutpostLagsResponse struct {
	XMLName     xml.Name                 `xml:"DescribeOutpostLagsResponse"`
	Xmlns       string                   `xml:"xmlns,attr"`
	RequestID   string                   `xml:"requestId"`
	OutpostLags ec2Stage120OutpostLagSet `xml:"outpostLagSet"`
	NextToken   *string                  `xml:"nextToken,omitempty"`
}

type ec2Stage120OutpostLagSet struct {
	Items []ec2Stage120OutpostLagItem `xml:"item"`
}

type ec2Stage120OutpostLagItem struct {
	LocalGatewayVirtualInterfaceIDs ec2StringSet `xml:"localGatewayVirtualInterfaceIdSet"`
	OutpostARN                      string       `xml:"outpostArn,omitempty"`
	OutpostLagID                    string       `xml:"outpostLagId,omitempty"`
	OwnerID                         string       `xml:"ownerId,omitempty"`
	ServiceLinkVirtualInterfaceIDs  ec2StringSet `xml:"serviceLinkVirtualInterfaceIdSet"`
	State                           string       `xml:"state,omitempty"`
	TagSet                          ec2TagSet    `xml:"tagSet"`
}

type ec2Stage120DescribePrefixListsResponse struct {
	XMLName     xml.Name                 `xml:"DescribePrefixListsResponse"`
	Xmlns       string                   `xml:"xmlns,attr"`
	RequestID   string                   `xml:"requestId"`
	PrefixLists ec2Stage120PrefixListSet `xml:"prefixListSet"`
	NextToken   *string                  `xml:"nextToken,omitempty"`
}

type ec2Stage120PrefixListSet struct {
	Items []ec2Stage120PrefixListItem `xml:"item"`
}

type ec2Stage120PrefixListItem struct {
	CIDRSet        ec2StringSet `xml:"cidrSet"`
	PrefixListID   string       `xml:"prefixListId,omitempty"`
	PrefixListName string       `xml:"prefixListName,omitempty"`
}

type ec2Stage120DescribePublicIpv4PoolsResponse struct {
	XMLName         xml.Name                     `xml:"DescribePublicIpv4PoolsResponse"`
	Xmlns           string                       `xml:"xmlns,attr"`
	RequestID       string                       `xml:"requestId"`
	PublicIpv4Pools ec2Stage120PublicIpv4PoolSet `xml:"publicIpv4PoolSet"`
	NextToken       *string                      `xml:"nextToken,omitempty"`
}

type ec2Stage120PublicIpv4PoolSet struct {
	Items []ec2Stage120PublicIpv4PoolItem `xml:"item"`
}

type ec2Stage120PublicIpv4PoolItem struct {
	Description                string                            `xml:"description,omitempty"`
	NetworkBorderGroup         string                            `xml:"networkBorderGroup,omitempty"`
	PoolAddressRanges          ec2Stage120PublicIpv4PoolRangeSet `xml:"poolAddressRangeSet"`
	PoolID                     string                            `xml:"poolId,omitempty"`
	TagSet                     ec2TagSet                         `xml:"tagSet"`
	TotalAddressCount          *int32                            `xml:"totalAddressCount,omitempty"`
	TotalAvailableAddressCount *int32                            `xml:"totalAvailableAddressCount,omitempty"`
}

type ec2Stage120PublicIpv4PoolRangeSet struct {
	Items []ec2Stage120PublicIpv4PoolRangeItem `xml:"item"`
}

type ec2Stage120PublicIpv4PoolRangeItem struct {
	AddressCount          *int32 `xml:"addressCount,omitempty"`
	AvailableAddressCount *int32 `xml:"availableAddressCount,omitempty"`
	FirstAddress          string `xml:"firstAddress,omitempty"`
	LastAddress           string `xml:"lastAddress,omitempty"`
}

type ec2Stage120DescribeReplaceRootVolumeTasksResponse struct {
	XMLName                xml.Name                            `xml:"DescribeReplaceRootVolumeTasksResponse"`
	Xmlns                  string                              `xml:"xmlns,attr"`
	RequestID              string                              `xml:"requestId"`
	ReplaceRootVolumeTasks ec2Stage120ReplaceRootVolumeTaskSet `xml:"replaceRootVolumeTaskSet"`
	NextToken              *string                             `xml:"nextToken,omitempty"`
}

type ec2Stage120ReplaceRootVolumeTaskSet struct {
	Items []ec2Stage109ReplaceRootVolumeTaskItem `xml:"item"`
}

type ec2Stage120DescribeReservedInstancesResponse struct {
	XMLName           xml.Name                       `xml:"DescribeReservedInstancesResponse"`
	Xmlns             string                         `xml:"xmlns,attr"`
	RequestID         string                         `xml:"requestId"`
	ReservedInstances ec2Stage120ReservedInstanceSet `xml:"reservedInstancesSet"`
}

type ec2Stage120ReservedInstanceSet struct {
	Items []ec2Stage120ReservedInstanceItem `xml:"item"`
}

type ec2Stage120ReservedInstanceItem struct {
	AvailabilityZone    string                        `xml:"availabilityZone,omitempty"`
	AvailabilityZoneID  string                        `xml:"availabilityZoneId,omitempty"`
	CurrencyCode        string                        `xml:"currencyCode,omitempty"`
	Duration            *int64                        `xml:"duration,omitempty"`
	End                 string                        `xml:"end,omitempty"`
	FixedPrice          *float32                      `xml:"fixedPrice,omitempty"`
	InstanceCount       *int32                        `xml:"instanceCount,omitempty"`
	InstanceTenancy     string                        `xml:"instanceTenancy,omitempty"`
	InstanceType        string                        `xml:"instanceType,omitempty"`
	OfferingClass       string                        `xml:"offeringClass,omitempty"`
	OfferingType        string                        `xml:"offeringType,omitempty"`
	ProductDescription  string                        `xml:"productDescription,omitempty"`
	RecurringCharges    ec2Stage120RecurringChargeSet `xml:"recurringCharges"`
	ReservedInstancesID string                        `xml:"reservedInstancesId,omitempty"`
	Scope               string                        `xml:"scope,omitempty"`
	Start               string                        `xml:"start,omitempty"`
	State               string                        `xml:"state,omitempty"`
	TagSet              ec2TagSet                     `xml:"tagSet"`
	UsagePrice          *float32                      `xml:"usagePrice,omitempty"`
}

type ec2Stage120RecurringChargeSet struct {
	Items []ec2Stage120RecurringChargeItem `xml:"item"`
}

type ec2Stage120RecurringChargeItem struct {
	Amount    float32 `xml:"amount,omitempty"`
	Frequency string  `xml:"frequency,omitempty"`
}
