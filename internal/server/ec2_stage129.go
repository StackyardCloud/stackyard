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

func (s *Server) handleEC2Stage129Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyIpam":
		enablePrivateGua, hasEnablePrivateGua, ok := ec2OptionalBoolFromForm(r.Form, "EnablePrivateGua")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasEnablePrivateGua {
			enablePrivateGua = nil
		}
		ipam, err := s.ec2.ModifyIpam(
			strings.TrimSpace(r.Form.Get("IpamId")),
			ec2OptionalStringPointerFromForm(r.Form, "Description"),
			enablePrivateGua,
			strings.TrimSpace(r.Form.Get("MeteredAccount")),
			parseEC2Stage129OperatingRegions(r.Form, "AddOperatingRegion.", "AddOperatingRegions."),
			parseEC2Stage129OperatingRegions(r.Form, "RemoveOperatingRegion.", "RemoveOperatingRegions."),
			strings.TrimSpace(r.Form.Get("Tier")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage129ModifyIpamResponse{
			XMLName:   xml.Name{Local: "ModifyIpamResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Ipam:      ec2Stage107IpamItemFrom(ipam),
		})
		return true
	case "ModifyIpamPool":
		allocationDefaultNetmaskLength, hasAllocationDefaultNetmaskLength, ok := ec2OptionalInt32FromForm(r.Form, "AllocationDefaultNetmaskLength")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasAllocationDefaultNetmaskLength {
			allocationDefaultNetmaskLength = nil
		}

		allocationMaxNetmaskLength, hasAllocationMaxNetmaskLength, ok := ec2OptionalInt32FromForm(r.Form, "AllocationMaxNetmaskLength")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasAllocationMaxNetmaskLength {
			allocationMaxNetmaskLength = nil
		}

		allocationMinNetmaskLength, hasAllocationMinNetmaskLength, ok := ec2OptionalInt32FromForm(r.Form, "AllocationMinNetmaskLength")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasAllocationMinNetmaskLength {
			allocationMinNetmaskLength = nil
		}

		autoImport, hasAutoImport, ok := ec2OptionalBoolFromForm(r.Form, "AutoImport")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasAutoImport {
			autoImport = nil
		}

		clearAllocationDefaultNetmaskLength, hasClearAllocationDefaultNetmaskLength, ok := ec2OptionalBoolFromForm(r.Form, "ClearAllocationDefaultNetmaskLength")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasClearAllocationDefaultNetmaskLength {
			clearAllocationDefaultNetmaskLength = nil
		}

		ipamPool, err := s.ec2.ModifyIpamPool(
			strings.TrimSpace(r.Form.Get("IpamPoolId")),
			parseEC2Stage129RequestIpamResourceTags(r.Form, "AddAllocationResourceTag.", "AddAllocationResourceTags."),
			allocationDefaultNetmaskLength,
			allocationMaxNetmaskLength,
			allocationMinNetmaskLength,
			autoImport,
			clearAllocationDefaultNetmaskLength,
			ec2OptionalStringPointerFromForm(r.Form, "Description"),
			parseEC2Stage129RequestIpamResourceTags(r.Form, "RemoveAllocationResourceTag.", "RemoveAllocationResourceTags."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage129ModifyIpamPoolResponse{
			XMLName:   xml.Name{Local: "ModifyIpamPoolResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			IpamPool:  ec2Stage108IpamPoolItemFrom(ipamPool),
		})
		return true
	case "ModifyIpamResourceCidr":
		monitored, hasMonitored, ok := ec2OptionalBoolFromForm(r.Form, "Monitored")
		if !ok || !hasMonitored || monitored == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		resourceCidr, err := s.ec2.ModifyIpamResourceCidr(
			strings.TrimSpace(r.Form.Get("CurrentIpamScopeId")),
			*monitored,
			strings.TrimSpace(r.Form.Get("ResourceCidr")),
			strings.TrimSpace(r.Form.Get("ResourceId")),
			strings.TrimSpace(r.Form.Get("ResourceRegion")),
			parseEC2OptionalString(r.Form.Get("DestinationIpamScopeId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		resourceCidrItems := ec2Stage125IpamResourceCidrItemsFrom([]ec2svc.IpamResourceCidr{resourceCidr})
		resourceCidrItem := ec2Stage125IpamResourceCidrItem{}
		if len(resourceCidrItems) == 1 {
			resourceCidrItem = resourceCidrItems[0]
		}
		respondEC2XML(w, ec2Stage129ModifyIpamResourceCidrResponse{
			XMLName:          xml.Name{Local: "ModifyIpamResourceCidrResponse"},
			Xmlns:            ec2Namespace,
			RequestID:        "stackyard-request",
			IpamResourceCidr: resourceCidrItem,
		})
		return true
	case "ModifyIpamResourceDiscovery":
		discovery, err := s.ec2.ModifyIpamResourceDiscovery(
			strings.TrimSpace(r.Form.Get("IpamResourceDiscoveryId")),
			parseEC2Stage129OperatingRegions(r.Form, "AddOperatingRegion.", "AddOperatingRegions."),
			ec2OptionalStringPointerFromForm(r.Form, "Description"),
			parseEC2Stage129OperatingRegions(r.Form, "RemoveOperatingRegion.", "RemoveOperatingRegions."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage129ModifyIpamResourceDiscoveryResponse{
			XMLName:               xml.Name{Local: "ModifyIpamResourceDiscoveryResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			IpamResourceDiscovery: ec2Stage108IpamResourceDiscoveryItemFrom(discovery),
		})
		return true
	case "ModifyIpamScope":
		scope, err := s.ec2.ModifyIpamScope(
			strings.TrimSpace(r.Form.Get("IpamScopeId")),
			ec2OptionalStringPointerFromForm(r.Form, "Description"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage129ModifyIpamScopeResponse{
			XMLName:   xml.Name{Local: "ModifyIpamScopeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			IpamScope: ec2Stage108IpamScopeItemFrom(scope),
		})
		return true
	case "ModifyLaunchTemplate":
		launchTemplate, err := s.ec2.ModifyLaunchTemplate(
			strings.TrimSpace(r.Form.Get("LaunchTemplateId")),
			strings.TrimSpace(r.Form.Get("LaunchTemplateName")),
			ec2OptionalStringPointerFromForm(r.Form, "SetDefaultVersion"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage129ModifyLaunchTemplateResponse{
			XMLName:        xml.Name{Local: "ModifyLaunchTemplateResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			LaunchTemplate: ec2Stage108LaunchTemplateItemFrom(launchTemplate),
		})
		return true
	case "ModifyLocalGatewayRoute":
		route, err := s.ec2.ModifyLocalGatewayRoute(
			strings.TrimSpace(r.Form.Get("LocalGatewayRouteTableId")),
			parseEC2OptionalString(r.Form.Get("DestinationCidrBlock")),
			parseEC2OptionalString(r.Form.Get("DestinationPrefixListId")),
			ec2OptionalStringPointerFromForm(r.Form, "LocalGatewayVirtualInterfaceGroupId"),
			ec2OptionalStringPointerFromForm(r.Form, "NetworkInterfaceId"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage129ModifyLocalGatewayRouteResponse{
			XMLName:   xml.Name{Local: "ModifyLocalGatewayRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Route:     ec2Stage108LocalGatewayRouteItemFrom(route),
		})
		return true
	case "ModifyManagedPrefixList":
		var currentVersion *int64
		if hasEC2Field(r.Form, "CurrentVersion") {
			parsed, ok := parseEC2OptionalInt64(r.Form.Get("CurrentVersion"))
			if !ok || parsed == nil {
				respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
				return true
			}
			currentVersion = parsed
		}
		maxEntries, hasMaxEntries, ok := ec2OptionalInt32FromForm(r.Form, "MaxEntries")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasMaxEntries {
			maxEntries = nil
		}

		prefixList, err := s.ec2.ModifyManagedPrefixList(
			strings.TrimSpace(r.Form.Get("PrefixListId")),
			parseEC2Stage129ModifyManagedPrefixListAddEntries(r.Form, "AddEntry."),
			currentVersion,
			maxEntries,
			ec2OptionalStringPointerFromForm(r.Form, "PrefixListName"),
			parseEC2Stage129ModifyManagedPrefixListRemoveEntries(r.Form, "RemoveEntry."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage129ModifyManagedPrefixListResponse{
			XMLName:    xml.Name{Local: "ModifyManagedPrefixListResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			PrefixList: ec2Stage109ManagedPrefixListItemFrom(prefixList),
		})
		return true
	case "ModifyPrivateDnsNameOptions":
		enableResourceNameDnsARecord, hasEnableResourceNameDnsARecord, ok := ec2OptionalBoolFromForm(r.Form, "EnableResourceNameDnsARecord")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasEnableResourceNameDnsARecord {
			enableResourceNameDnsARecord = nil
		}

		enableResourceNameDnsAAAARecord, hasEnableResourceNameDnsAAAARecord, ok := ec2OptionalBoolFromForm(r.Form, "EnableResourceNameDnsAAAARecord")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasEnableResourceNameDnsAAAARecord {
			enableResourceNameDnsAAAARecord = nil
		}

		successful, err := s.ec2.ModifyPrivateDnsNameOptions(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			enableResourceNameDnsARecord,
			enableResourceNameDnsAAAARecord,
			ec2OptionalStringPointerFromForm(r.Form, "PrivateDnsHostnameType"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyPrivateDnsNameOptionsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    successful,
		})
		return true
	case "ModifyPublicIpDnsNameOptions":
		successful, err := s.ec2.ModifyPublicIpDnsNameOptions(
			strings.TrimSpace(r.Form.Get("NetworkInterfaceId")),
			strings.TrimSpace(r.Form.Get("HostnameType")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage129ModifyPublicIpDnsNameOptionsResponse{
			XMLName:    xml.Name{Local: "ModifyPublicIpDnsNameOptionsResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			Successful: successful,
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage129OperatingRegions(values url.Values, prefixes ...string) []string {
	regionsByIndex := map[int]string{}
	for _, prefix := range prefixes {
		for key := range values {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			rest := strings.TrimPrefix(key, prefix)
			if strings.HasPrefix(rest, "Item.") {
				rest = strings.TrimPrefix(rest, "Item.")
			}
			parts := strings.SplitN(rest, ".", 2)
			if len(parts) == 0 {
				continue
			}
			idx, err := strconv.Atoi(parts[0])
			if err != nil || idx <= 0 {
				continue
			}
			region := ""
			if len(parts) == 1 {
				region = strings.TrimSpace(values.Get(key))
			} else if parts[1] == "RegionName" || parts[1] == "regionName" {
				region = strings.TrimSpace(values.Get(key))
			}
			if region == "" {
				continue
			}
			regionsByIndex[idx] = region
		}
	}
	if len(regionsByIndex) == 0 {
		return nil
	}
	indices := make([]int, 0, len(regionsByIndex))
	for idx := range regionsByIndex {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	out := make([]string, 0, len(indices))
	seen := map[string]struct{}{}
	for _, idx := range indices {
		region := strings.TrimSpace(regionsByIndex[idx])
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

func parseEC2Stage129RequestIpamResourceTags(values url.Values, prefixes ...string) []ec2svc.RequestIpamResourceTag {
	tagsByIndex := map[int]ec2svc.RequestIpamResourceTag{}
	for _, prefix := range prefixes {
		for key := range values {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			rest := strings.TrimPrefix(key, prefix)
			if strings.HasPrefix(rest, "Item.") {
				rest = strings.TrimPrefix(rest, "Item.")
			}
			parts := strings.SplitN(rest, ".", 2)
			if len(parts) != 2 {
				continue
			}
			idx, err := strconv.Atoi(parts[0])
			if err != nil || idx <= 0 {
				continue
			}
			tag := tagsByIndex[idx]
			switch parts[1] {
			case "Key", "key":
				tag.Key = strings.TrimSpace(values.Get(key))
			case "Value", "value":
				tag.Value = strings.TrimSpace(values.Get(key))
			}
			tagsByIndex[idx] = tag
		}
	}
	if len(tagsByIndex) == 0 {
		return nil
	}
	indices := make([]int, 0, len(tagsByIndex))
	for idx := range tagsByIndex {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	out := make([]ec2svc.RequestIpamResourceTag, 0, len(indices))
	for _, idx := range indices {
		tag := tagsByIndex[idx]
		if strings.TrimSpace(tag.Key) == "" {
			continue
		}
		out = append(out, ec2svc.RequestIpamResourceTag{Key: strings.TrimSpace(tag.Key), Value: strings.TrimSpace(tag.Value)})
	}
	return out
}

func parseEC2Stage129ModifyManagedPrefixListAddEntries(values url.Values, prefix string) []ec2svc.ModifyManagedPrefixListAddEntry {
	entriesByIndex := map[int]ec2svc.ModifyManagedPrefixListAddEntry{}
	for key := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
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
		case "Cidr", "cidr":
			entry.CIDR = strings.TrimSpace(values.Get(key))
		case "Description", "description":
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
	out := make([]ec2svc.ModifyManagedPrefixListAddEntry, 0, len(indices))
	for _, idx := range indices {
		entry := entriesByIndex[idx]
		if strings.TrimSpace(entry.CIDR) == "" {
			continue
		}
		out = append(out, ec2svc.ModifyManagedPrefixListAddEntry{CIDR: strings.TrimSpace(entry.CIDR), Description: strings.TrimSpace(entry.Description)})
	}
	return out
}

func parseEC2Stage129ModifyManagedPrefixListRemoveEntries(values url.Values, prefix string) []ec2svc.ModifyManagedPrefixListRemoveEntry {
	entriesByIndex := map[int]ec2svc.ModifyManagedPrefixListRemoveEntry{}
	for key := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
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
		case "Cidr", "cidr":
			entry.CIDR = strings.TrimSpace(values.Get(key))
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
	out := make([]ec2svc.ModifyManagedPrefixListRemoveEntry, 0, len(indices))
	for _, idx := range indices {
		entry := entriesByIndex[idx]
		if strings.TrimSpace(entry.CIDR) == "" {
			continue
		}
		out = append(out, ec2svc.ModifyManagedPrefixListRemoveEntry{CIDR: strings.TrimSpace(entry.CIDR)})
	}
	return out
}

type ec2Stage129ModifyIpamResponse struct {
	XMLName   xml.Name            `xml:"ModifyIpamResponse"`
	Xmlns     string              `xml:"xmlns,attr"`
	RequestID string              `xml:"requestId"`
	Ipam      ec2Stage107IpamItem `xml:"ipam"`
}

type ec2Stage129ModifyIpamPoolResponse struct {
	XMLName   xml.Name                `xml:"ModifyIpamPoolResponse"`
	Xmlns     string                  `xml:"xmlns,attr"`
	RequestID string                  `xml:"requestId"`
	IpamPool  ec2Stage108IpamPoolItem `xml:"ipamPool"`
}

type ec2Stage129ModifyIpamResourceCidrResponse struct {
	XMLName          xml.Name                        `xml:"ModifyIpamResourceCidrResponse"`
	Xmlns            string                          `xml:"xmlns,attr"`
	RequestID        string                          `xml:"requestId"`
	IpamResourceCidr ec2Stage125IpamResourceCidrItem `xml:"ipamResourceCidr"`
}

type ec2Stage129ModifyIpamResourceDiscoveryResponse struct {
	XMLName               xml.Name                             `xml:"ModifyIpamResourceDiscoveryResponse"`
	Xmlns                 string                               `xml:"xmlns,attr"`
	RequestID             string                               `xml:"requestId"`
	IpamResourceDiscovery ec2Stage108IpamResourceDiscoveryItem `xml:"ipamResourceDiscovery"`
}

type ec2Stage129ModifyIpamScopeResponse struct {
	XMLName   xml.Name                 `xml:"ModifyIpamScopeResponse"`
	Xmlns     string                   `xml:"xmlns,attr"`
	RequestID string                   `xml:"requestId"`
	IpamScope ec2Stage108IpamScopeItem `xml:"ipamScope"`
}

type ec2Stage129ModifyLaunchTemplateResponse struct {
	XMLName        xml.Name                      `xml:"ModifyLaunchTemplateResponse"`
	Xmlns          string                        `xml:"xmlns,attr"`
	RequestID      string                        `xml:"requestId"`
	LaunchTemplate ec2Stage108LaunchTemplateItem `xml:"launchTemplate"`
}

type ec2Stage129ModifyLocalGatewayRouteResponse struct {
	XMLName   xml.Name                         `xml:"ModifyLocalGatewayRouteResponse"`
	Xmlns     string                           `xml:"xmlns,attr"`
	RequestID string                           `xml:"requestId"`
	Route     ec2Stage108LocalGatewayRouteItem `xml:"route"`
}

type ec2Stage129ModifyManagedPrefixListResponse struct {
	XMLName    xml.Name                         `xml:"ModifyManagedPrefixListResponse"`
	Xmlns      string                           `xml:"xmlns,attr"`
	RequestID  string                           `xml:"requestId"`
	PrefixList ec2Stage109ManagedPrefixListItem `xml:"prefixList"`
}

type ec2Stage129ModifyPublicIpDnsNameOptionsResponse struct {
	XMLName    xml.Name `xml:"ModifyPublicIpDnsNameOptionsResponse"`
	Xmlns      string   `xml:"xmlns,attr"`
	RequestID  string   `xml:"requestId"`
	Successful bool     `xml:"successful"`
}
