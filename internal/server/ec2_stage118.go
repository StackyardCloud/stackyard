package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage118Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeInstanceTypes":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		instanceTypes, nextToken, err := s.ec2.DescribeInstanceTypes(
			parseEC2MembersOrItemList(r.Form, "InstanceType"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage118DescribeInstanceTypesResponse{
			XMLName:         xml.Name{Local: "DescribeInstanceTypesResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			InstanceTypeSet: ec2Stage118InstanceTypeInfoSet{Items: ec2Stage118InstanceTypeInfoItemsFrom(instanceTypes)},
			NextToken:       nextToken,
		})
		return true
	case "DescribeIpamByoasn":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		byoasns, nextToken, err := s.ec2.DescribeIpamByoasn(
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage118DescribeIpamByoasnResponse{
			XMLName:   xml.Name{Local: "DescribeIpamByoasnResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ByoasnSet: ec2Stage118ByoasnSet{Items: ec2Stage118ByoasnItemsFrom(byoasns)},
			NextToken: nextToken,
		})
		return true
	case "DescribeIpamExternalResourceVerificationTokens":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		tokens, nextToken, err := s.ec2.DescribeIpamExternalResourceVerificationTokens(
			parseEC2MembersOrItemList(r.Form, "IpamExternalResourceVerificationTokenId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage118DescribeIpamExternalResourceVerificationTokensResponse{
			XMLName:                                  xml.Name{Local: "DescribeIpamExternalResourceVerificationTokensResponse"},
			Xmlns:                                    ec2Namespace,
			RequestID:                                "stackyard-request",
			IpamExternalResourceVerificationTokenSet: ec2Stage118IpamExternalResourceVerificationTokenSet{Items: ec2Stage118IpamExternalResourceVerificationTokenItemsFrom(tokens)},
			NextToken:                                nextToken,
		})
		return true
	case "DescribeIpamPools":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		pools, nextToken, err := s.ec2.DescribeIpamPools(
			parseEC2MembersOrItemList(r.Form, "IpamPoolId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage118DescribeIpamPoolsResponse{
			XMLName:     xml.Name{Local: "DescribeIpamPoolsResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			IpamPoolSet: ec2Stage118IpamPoolSet{Items: ec2Stage118IpamPoolItemsFrom(pools)},
			NextToken:   nextToken,
		})
		return true
	case "DescribeIpamResourceDiscoveries":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		discoveries, nextToken, err := s.ec2.DescribeIpamResourceDiscoveries(
			parseEC2MembersOrItemList(r.Form, "IpamResourceDiscoveryId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage118DescribeIpamResourceDiscoveriesResponse{
			XMLName:                  xml.Name{Local: "DescribeIpamResourceDiscoveriesResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			IpamResourceDiscoverySet: ec2Stage118IpamResourceDiscoverySet{Items: ec2Stage118IpamResourceDiscoveryItemsFrom(discoveries)},
			NextToken:                nextToken,
		})
		return true
	case "DescribeIpamResourceDiscoveryAssociations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		associations, nextToken, err := s.ec2.DescribeIpamResourceDiscoveryAssociations(
			parseEC2MembersOrItemList(r.Form, "IpamResourceDiscoveryAssociationId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage118DescribeIpamResourceDiscoveryAssociationsResponse{
			XMLName:                             xml.Name{Local: "DescribeIpamResourceDiscoveryAssociationsResponse"},
			Xmlns:                               ec2Namespace,
			RequestID:                           "stackyard-request",
			IpamResourceDiscoveryAssociationSet: ec2Stage118IpamResourceDiscoveryAssociationSet{Items: ec2Stage118IpamResourceDiscoveryAssociationItemsFrom(associations)},
			NextToken:                           nextToken,
		})
		return true
	case "DescribeIpamScopes":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		scopes, nextToken, err := s.ec2.DescribeIpamScopes(
			parseEC2MembersOrItemList(r.Form, "IpamScopeId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage118DescribeIpamScopesResponse{
			XMLName:      xml.Name{Local: "DescribeIpamScopesResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			IpamScopeSet: ec2Stage118IpamScopeSet{Items: ec2Stage118IpamScopeItemsFrom(scopes)},
			NextToken:    nextToken,
		})
		return true
	case "DescribeIpams":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		ipams, nextToken, err := s.ec2.DescribeIpams(
			parseEC2MembersOrItemList(r.Form, "IpamId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage118DescribeIpamsResponse{
			XMLName:   xml.Name{Local: "DescribeIpamsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			IpamSet:   ec2Stage118IpamSet{Items: ec2Stage118IpamItemsFrom(ipams)},
			NextToken: nextToken,
		})
		return true
	case "DescribeIpv6Pools":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		ipv6Pools, nextToken, err := s.ec2.DescribeIpv6Pools(
			parseEC2MembersOrItemList(r.Form, "PoolId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage118DescribeIpv6PoolsResponse{
			XMLName:     xml.Name{Local: "DescribeIpv6PoolsResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			Ipv6PoolSet: ec2Stage118Ipv6PoolSet{Items: ec2Stage118Ipv6PoolItemsFrom(ipv6Pools)},
			NextToken:   nextToken,
		})
		return true
	case "DescribeLaunchTemplateVersions":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		resolveAlias, hasResolveAlias, ok := ec2OptionalBoolFromForm(r.Form, "ResolveAlias")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasResolveAlias {
			resolveAlias = nil
		}
		launchTemplateVersions, nextToken, err := s.ec2.DescribeLaunchTemplateVersions(
			strings.TrimSpace(r.Form.Get("LaunchTemplateId")),
			strings.TrimSpace(r.Form.Get("LaunchTemplateName")),
			parseEC2MembersOrItemList(r.Form, "LaunchTemplateVersion"),
			parseEC2OptionalString(r.Form.Get("MinVersion")),
			parseEC2OptionalString(r.Form.Get("MaxVersion")),
			resolveAlias,
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage118DescribeLaunchTemplateVersionsResponse{
			XMLName:                  xml.Name{Local: "DescribeLaunchTemplateVersionsResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			LaunchTemplateVersionSet: ec2Stage118LaunchTemplateVersionSet{Items: ec2Stage118LaunchTemplateVersionItemsFrom(launchTemplateVersions)},
			NextToken:                nextToken,
		})
		return true
	default:
		return false
	}
}

func ec2Stage118InstanceTypeInfoItemsFrom(in []ec2svc.InstanceTypeInfo) []ec2Stage118InstanceTypeInfoItem {
	out := make([]ec2Stage118InstanceTypeInfoItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage118InstanceTypeInfoItem{InstanceType: item.InstanceType})
	}
	return out
}

func ec2Stage118ByoasnItemsFrom(in []ec2svc.Byoasn) []ec2Stage118ByoasnItem {
	out := make([]ec2Stage118ByoasnItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage118ByoasnItem{
			ASN:           item.Asn,
			IpamID:        item.IpamID,
			State:         item.State,
			StatusMessage: item.StatusMessage,
		})
	}
	return out
}

func ec2Stage118IpamExternalResourceVerificationTokenItemsFrom(in []ec2svc.IpamExternalResourceVerificationToken) []ec2Stage107IpamExternalResourceVerificationTokenItem {
	out := make([]ec2Stage107IpamExternalResourceVerificationTokenItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage111IpamExternalResourceVerificationTokenItemFrom(item))
	}
	return out
}

func ec2Stage118IpamPoolItemsFrom(in []ec2svc.IpamPool) []ec2Stage108IpamPoolItem {
	out := make([]ec2Stage108IpamPoolItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage108IpamPoolItemFrom(item))
	}
	return out
}

func ec2Stage118IpamResourceDiscoveryItemsFrom(in []ec2svc.IpamResourceDiscovery) []ec2Stage108IpamResourceDiscoveryItem {
	out := make([]ec2Stage108IpamResourceDiscoveryItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage108IpamResourceDiscoveryItemFrom(item))
	}
	return out
}

func ec2Stage118IpamResourceDiscoveryAssociationItemsFrom(in []ec2svc.IpamResourceDiscoveryAssociation) []ec2IpamResourceDiscoveryAssociationItem {
	out := make([]ec2IpamResourceDiscoveryAssociationItem, 0, len(in))
	for _, item := range in {
		entry := ec2IpamResourceDiscoveryAssociationItem{
			IpamARN:                             item.IpamARN,
			IpamID:                              item.IpamID,
			IpamRegion:                          item.IpamRegion,
			IpamResourceDiscoveryAssociationARN: item.IpamResourceDiscoveryAssociationARN,
			IpamResourceDiscoveryAssociationID:  item.IpamResourceDiscoveryAssociationID,
			IpamResourceDiscoveryID:             item.IpamResourceDiscoveryID,
			OwnerID:                             item.OwnerID,
			ResourceDiscoveryStatus:             item.ResourceDiscoveryStatus,
			State:                               item.State,
		}
		isDefault := item.IsDefault
		entry.IsDefault = &isDefault
		tagMap := map[string]string{}
		for _, tag := range item.Tags {
			key := strings.TrimSpace(tag.Key)
			if key == "" {
				continue
			}
			tagMap[key] = strings.TrimSpace(tag.Value)
		}
		if len(tagMap) > 0 {
			tagSet := ec2TagSet{Items: ec2TagItemsFromMap(tagMap)}
			entry.TagSet = &tagSet
		}
		out = append(out, entry)
	}
	return out
}

func ec2Stage118IpamScopeItemsFrom(in []ec2svc.IpamScope) []ec2Stage108IpamScopeItem {
	out := make([]ec2Stage108IpamScopeItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage108IpamScopeItemFrom(item))
	}
	return out
}

func ec2Stage118IpamItemsFrom(in []ec2svc.Ipam) []ec2Stage107IpamItem {
	out := make([]ec2Stage107IpamItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage107IpamItemFrom(item))
	}
	return out
}

func ec2Stage118Ipv6PoolItemsFrom(in []ec2svc.Ipv6Pool) []ec2Stage118Ipv6PoolItem {
	out := make([]ec2Stage118Ipv6PoolItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage118Ipv6PoolItem{
			Description: item.Description,
			PoolID:      item.PoolID,
			PoolCidrBlockSet: ec2Stage118PoolCidrBlockSet{
				Items: ec2Stage118PoolCidrBlockItemsFrom(item.PoolCIDRBlocks),
			},
			TagSet: ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
		})
	}
	return out
}

func ec2Stage118PoolCidrBlockItemsFrom(in []string) []ec2Stage118PoolCidrBlockItem {
	out := make([]ec2Stage118PoolCidrBlockItem, 0, len(in))
	for _, cidr := range in {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		out = append(out, ec2Stage118PoolCidrBlockItem{PoolCIDRBlock: cidr})
	}
	return out
}

func ec2Stage118LaunchTemplateVersionItemsFrom(in []ec2svc.LaunchTemplateVersion) []ec2Stage108LaunchTemplateVersionItem {
	out := make([]ec2Stage108LaunchTemplateVersionItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage108LaunchTemplateVersionItemFrom(item))
	}
	return out
}

type ec2Stage118DescribeInstanceTypesResponse struct {
	XMLName         xml.Name                       `xml:"DescribeInstanceTypesResponse"`
	Xmlns           string                         `xml:"xmlns,attr"`
	RequestID       string                         `xml:"requestId"`
	InstanceTypeSet ec2Stage118InstanceTypeInfoSet `xml:"instanceTypeSet"`
	NextToken       *string                        `xml:"nextToken,omitempty"`
}

type ec2Stage118InstanceTypeInfoSet struct {
	Items []ec2Stage118InstanceTypeInfoItem `xml:"item"`
}

type ec2Stage118InstanceTypeInfoItem struct {
	InstanceType string `xml:"instanceType,omitempty"`
}

type ec2Stage118DescribeIpamByoasnResponse struct {
	XMLName   xml.Name             `xml:"DescribeIpamByoasnResponse"`
	Xmlns     string               `xml:"xmlns,attr"`
	RequestID string               `xml:"requestId"`
	ByoasnSet ec2Stage118ByoasnSet `xml:"byoasnSet"`
	NextToken *string              `xml:"nextToken,omitempty"`
}

type ec2Stage118ByoasnSet struct {
	Items []ec2Stage118ByoasnItem `xml:"item"`
}

type ec2Stage118ByoasnItem struct {
	ASN           string `xml:"asn,omitempty"`
	IpamID        string `xml:"ipamId,omitempty"`
	State         string `xml:"state,omitempty"`
	StatusMessage string `xml:"statusMessage,omitempty"`
}

type ec2Stage118DescribeIpamExternalResourceVerificationTokensResponse struct {
	XMLName                                  xml.Name                                            `xml:"DescribeIpamExternalResourceVerificationTokensResponse"`
	Xmlns                                    string                                              `xml:"xmlns,attr"`
	RequestID                                string                                              `xml:"requestId"`
	IpamExternalResourceVerificationTokenSet ec2Stage118IpamExternalResourceVerificationTokenSet `xml:"ipamExternalResourceVerificationTokenSet"`
	NextToken                                *string                                             `xml:"nextToken,omitempty"`
}

type ec2Stage118IpamExternalResourceVerificationTokenSet struct {
	Items []ec2Stage107IpamExternalResourceVerificationTokenItem `xml:"item"`
}

type ec2Stage118DescribeIpamPoolsResponse struct {
	XMLName     xml.Name               `xml:"DescribeIpamPoolsResponse"`
	Xmlns       string                 `xml:"xmlns,attr"`
	RequestID   string                 `xml:"requestId"`
	IpamPoolSet ec2Stage118IpamPoolSet `xml:"ipamPoolSet"`
	NextToken   *string                `xml:"nextToken,omitempty"`
}

type ec2Stage118IpamPoolSet struct {
	Items []ec2Stage108IpamPoolItem `xml:"item"`
}

type ec2Stage118DescribeIpamResourceDiscoveriesResponse struct {
	XMLName                  xml.Name                            `xml:"DescribeIpamResourceDiscoveriesResponse"`
	Xmlns                    string                              `xml:"xmlns,attr"`
	RequestID                string                              `xml:"requestId"`
	IpamResourceDiscoverySet ec2Stage118IpamResourceDiscoverySet `xml:"ipamResourceDiscoverySet"`
	NextToken                *string                             `xml:"nextToken,omitempty"`
}

type ec2Stage118IpamResourceDiscoverySet struct {
	Items []ec2Stage108IpamResourceDiscoveryItem `xml:"item"`
}

type ec2Stage118DescribeIpamResourceDiscoveryAssociationsResponse struct {
	XMLName                             xml.Name                                       `xml:"DescribeIpamResourceDiscoveryAssociationsResponse"`
	Xmlns                               string                                         `xml:"xmlns,attr"`
	RequestID                           string                                         `xml:"requestId"`
	IpamResourceDiscoveryAssociationSet ec2Stage118IpamResourceDiscoveryAssociationSet `xml:"ipamResourceDiscoveryAssociationSet"`
	NextToken                           *string                                        `xml:"nextToken,omitempty"`
}

type ec2Stage118IpamResourceDiscoveryAssociationSet struct {
	Items []ec2IpamResourceDiscoveryAssociationItem `xml:"item"`
}

type ec2Stage118DescribeIpamScopesResponse struct {
	XMLName      xml.Name                `xml:"DescribeIpamScopesResponse"`
	Xmlns        string                  `xml:"xmlns,attr"`
	RequestID    string                  `xml:"requestId"`
	IpamScopeSet ec2Stage118IpamScopeSet `xml:"ipamScopeSet"`
	NextToken    *string                 `xml:"nextToken,omitempty"`
}

type ec2Stage118IpamScopeSet struct {
	Items []ec2Stage108IpamScopeItem `xml:"item"`
}

type ec2Stage118DescribeIpamsResponse struct {
	XMLName   xml.Name           `xml:"DescribeIpamsResponse"`
	Xmlns     string             `xml:"xmlns,attr"`
	RequestID string             `xml:"requestId"`
	IpamSet   ec2Stage118IpamSet `xml:"ipamSet"`
	NextToken *string            `xml:"nextToken,omitempty"`
}

type ec2Stage118IpamSet struct {
	Items []ec2Stage107IpamItem `xml:"item"`
}

type ec2Stage118DescribeIpv6PoolsResponse struct {
	XMLName     xml.Name               `xml:"DescribeIpv6PoolsResponse"`
	Xmlns       string                 `xml:"xmlns,attr"`
	RequestID   string                 `xml:"requestId"`
	Ipv6PoolSet ec2Stage118Ipv6PoolSet `xml:"ipv6PoolSet"`
	NextToken   *string                `xml:"nextToken,omitempty"`
}

type ec2Stage118Ipv6PoolSet struct {
	Items []ec2Stage118Ipv6PoolItem `xml:"item"`
}

type ec2Stage118Ipv6PoolItem struct {
	Description      string                      `xml:"description,omitempty"`
	PoolCidrBlockSet ec2Stage118PoolCidrBlockSet `xml:"poolCidrBlockSet"`
	PoolID           string                      `xml:"poolId,omitempty"`
	TagSet           ec2TagSet                   `xml:"tagSet"`
}

type ec2Stage118PoolCidrBlockSet struct {
	Items []ec2Stage118PoolCidrBlockItem `xml:"item"`
}

type ec2Stage118PoolCidrBlockItem struct {
	PoolCIDRBlock string `xml:"poolCidrBlock,omitempty"`
}

type ec2Stage118DescribeLaunchTemplateVersionsResponse struct {
	XMLName                  xml.Name                            `xml:"DescribeLaunchTemplateVersionsResponse"`
	Xmlns                    string                              `xml:"xmlns,attr"`
	RequestID                string                              `xml:"requestId"`
	LaunchTemplateVersionSet ec2Stage118LaunchTemplateVersionSet `xml:"launchTemplateVersionSet"`
	NextToken                *string                             `xml:"nextToken,omitempty"`
}

type ec2Stage118LaunchTemplateVersionSet struct {
	Items []ec2Stage108LaunchTemplateVersionItem `xml:"item"`
}
