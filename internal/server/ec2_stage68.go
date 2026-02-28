package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage68Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateVerifiedAccessInstance":
		fipsEnabled, hasFIPSEnabled, ok := ec2OptionalBoolFromForm(r.Form, "FIPSEnabled")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasFIPSEnabled {
			fipsEnabled = nil
		}

		instance, err := s.ec2.CreateVerifiedAccessInstance(
			parseEC2OptionalString(r.Form.Get("CidrEndpointsCustomSubDomain")),
			parseEC2OptionalString(r.Form.Get("Description")),
			fipsEnabled,
			parseEC2TagSpecificationsForResource(r.Form, "verified-access-instance"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateVerifiedAccessInstanceResponse{
			XMLName:                xml.Name{Local: "CreateVerifiedAccessInstanceResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			VerifiedAccessInstance: ec2VerifiedAccessInstanceItemFrom(instance),
		})
		return true
	case "DeleteVerifiedAccessInstance":
		instance, err := s.ec2.DeleteVerifiedAccessInstance(strings.TrimSpace(r.Form.Get("VerifiedAccessInstanceId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteVerifiedAccessInstanceResponse{
			XMLName:                xml.Name{Local: "DeleteVerifiedAccessInstanceResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			VerifiedAccessInstance: ec2VerifiedAccessInstanceItemFrom(instance),
		})
		return true
	case "DescribeVerifiedAccessInstances":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		instances, nextToken, err := s.ec2.DescribeVerifiedAccessInstances(
			parseEC2Members(r.Form, "VerifiedAccessInstanceId."),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVerifiedAccessInstancesResponse{
			XMLName:   xml.Name{Local: "DescribeVerifiedAccessInstancesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VerifiedAccessInstanceSet: ec2VerifiedAccessInstanceSet{
				Items: ec2VerifiedAccessInstanceItemsFrom(instances),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "CreateVerifiedAccessGroup":
		group, err := s.ec2.CreateVerifiedAccessGroup(
			strings.TrimSpace(r.Form.Get("VerifiedAccessInstanceId")),
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2OptionalString(r.Form.Get("PolicyDocument")),
			parseEC2TagSpecificationsForResource(r.Form, "verified-access-group"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateVerifiedAccessGroupResponse{
			XMLName:             xml.Name{Local: "CreateVerifiedAccessGroupResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			VerifiedAccessGroup: ec2VerifiedAccessGroupItemFrom(group),
		})
		return true
	case "DeleteVerifiedAccessGroup":
		group, err := s.ec2.DeleteVerifiedAccessGroup(strings.TrimSpace(r.Form.Get("VerifiedAccessGroupId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteVerifiedAccessGroupResponse{
			XMLName:             xml.Name{Local: "DeleteVerifiedAccessGroupResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			VerifiedAccessGroup: ec2VerifiedAccessGroupItemFrom(group),
		})
		return true
	case "DescribeVerifiedAccessGroups":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		groups, nextToken, err := s.ec2.DescribeVerifiedAccessGroups(
			parseEC2Members(r.Form, "VerifiedAccessGroupId."),
			parseEC2OptionalString(r.Form.Get("VerifiedAccessInstanceId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVerifiedAccessGroupsResponse{
			XMLName:   xml.Name{Local: "DescribeVerifiedAccessGroupsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VerifiedAccessGroupSet: ec2VerifiedAccessGroupSet{
				Items: ec2VerifiedAccessGroupItemsFrom(groups),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "CreateVerifiedAccessEndpoint":
		endpoint, err := s.ec2.CreateVerifiedAccessEndpoint(
			strings.TrimSpace(r.Form.Get("VerifiedAccessGroupId")),
			strings.TrimSpace(r.Form.Get("AttachmentType")),
			strings.TrimSpace(r.Form.Get("EndpointType")),
			parseEC2OptionalString(r.Form.Get("ApplicationDomain")),
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2OptionalString(r.Form.Get("DomainCertificateArn")),
			parseEC2OptionalString(r.Form.Get("EndpointDomainPrefix")),
			parseEC2Members(r.Form, "SecurityGroupId."),
			parseEC2TagSpecificationsForResource(r.Form, "verified-access-endpoint"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateVerifiedAccessEndpointResponse{
			XMLName:                xml.Name{Local: "CreateVerifiedAccessEndpointResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			VerifiedAccessEndpoint: ec2VerifiedAccessEndpointItemFrom(endpoint),
		})
		return true
	case "DeleteVerifiedAccessEndpoint":
		endpoint, err := s.ec2.DeleteVerifiedAccessEndpoint(strings.TrimSpace(r.Form.Get("VerifiedAccessEndpointId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteVerifiedAccessEndpointResponse{
			XMLName:                xml.Name{Local: "DeleteVerifiedAccessEndpointResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			VerifiedAccessEndpoint: ec2VerifiedAccessEndpointItemFrom(endpoint),
		})
		return true
	case "DescribeVerifiedAccessEndpoints":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		endpoints, nextToken, err := s.ec2.DescribeVerifiedAccessEndpoints(
			parseEC2Members(r.Form, "VerifiedAccessEndpointId."),
			parseEC2OptionalString(r.Form.Get("VerifiedAccessGroupId")),
			parseEC2OptionalString(r.Form.Get("VerifiedAccessInstanceId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVerifiedAccessEndpointsResponse{
			XMLName:   xml.Name{Local: "DescribeVerifiedAccessEndpointsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VerifiedAccessEndpointSet: ec2VerifiedAccessEndpointSet{
				Items: ec2VerifiedAccessEndpointItemsFrom(endpoints),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "CreateVerifiedAccessTrustProvider":
		trustProvider, err := s.ec2.CreateVerifiedAccessTrustProvider(
			strings.TrimSpace(r.Form.Get("PolicyReferenceName")),
			strings.TrimSpace(r.Form.Get("TrustProviderType")),
			strings.TrimSpace(r.Form.Get("UserTrustProviderType")),
			strings.TrimSpace(r.Form.Get("DeviceTrustProviderType")),
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2TagSpecificationsForResource(r.Form, "verified-access-trust-provider"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateVerifiedAccessTrustProviderResponse{
			XMLName:                     xml.Name{Local: "CreateVerifiedAccessTrustProviderResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			VerifiedAccessTrustProvider: ec2VerifiedAccessTrustProviderItemFrom(trustProvider),
		})
		return true
	case "DeleteVerifiedAccessTrustProvider":
		trustProvider, err := s.ec2.DeleteVerifiedAccessTrustProvider(strings.TrimSpace(r.Form.Get("VerifiedAccessTrustProviderId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteVerifiedAccessTrustProviderResponse{
			XMLName:                     xml.Name{Local: "DeleteVerifiedAccessTrustProviderResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			VerifiedAccessTrustProvider: ec2VerifiedAccessTrustProviderItemFrom(trustProvider),
		})
		return true
	case "DescribeVerifiedAccessTrustProviders":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		trustProviders, nextToken, err := s.ec2.DescribeVerifiedAccessTrustProviders(
			parseEC2Members(r.Form, "VerifiedAccessTrustProviderId."),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVerifiedAccessTrustProvidersResponse{
			XMLName:   xml.Name{Local: "DescribeVerifiedAccessTrustProvidersResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VerifiedAccessTrustProviderSet: ec2VerifiedAccessTrustProviderSet{
				Items: ec2VerifiedAccessTrustProviderItemsFrom(trustProviders),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	default:
		return false
	}
}

func ec2VerifiedAccessInstanceItemsFrom(in []ec2svc.VerifiedAccessInstance) []ec2VerifiedAccessInstanceItem {
	out := make([]ec2VerifiedAccessInstanceItem, 0, len(in))
	for _, instance := range in {
		out = append(out, ec2VerifiedAccessInstanceItemFrom(instance))
	}
	return out
}

func ec2VerifiedAccessInstanceItemFrom(in ec2svc.VerifiedAccessInstance) ec2VerifiedAccessInstanceItem {
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	return ec2VerifiedAccessInstanceItem{
		CidrEndpointsCustomSubDomain: in.CidrEndpointsCustomSubDomain,
		CreationTime:                 in.CreationTime,
		Description:                  in.Description,
		FIPSEnabled:                  &in.FIPSEnabled,
		LastUpdatedTime:              in.LastUpdatedTime,
		TagSet:                       ec2TagSet{Items: tags},
		VerifiedAccessInstanceID:     in.ID,
	}
}

func ec2VerifiedAccessGroupItemsFrom(in []ec2svc.VerifiedAccessGroup) []ec2VerifiedAccessGroupItem {
	out := make([]ec2VerifiedAccessGroupItem, 0, len(in))
	for _, group := range in {
		out = append(out, ec2VerifiedAccessGroupItemFrom(group))
	}
	return out
}

func ec2VerifiedAccessGroupItemFrom(in ec2svc.VerifiedAccessGroup) ec2VerifiedAccessGroupItem {
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	out := ec2VerifiedAccessGroupItem{
		CreationTime:             in.CreationTime,
		Description:              in.Description,
		LastUpdatedTime:          in.LastUpdatedTime,
		Owner:                    in.Owner,
		TagSet:                   ec2TagSet{Items: tags},
		VerifiedAccessGroupARN:   in.ARN,
		VerifiedAccessGroupID:    in.ID,
		VerifiedAccessInstanceID: in.VerifiedInstance,
	}
	if in.DeletionTime != nil {
		out.DeletionTime = *in.DeletionTime
	}
	return out
}

func ec2VerifiedAccessEndpointItemsFrom(in []ec2svc.VerifiedAccessEndpoint) []ec2VerifiedAccessEndpointItem {
	out := make([]ec2VerifiedAccessEndpointItem, 0, len(in))
	for _, endpoint := range in {
		out = append(out, ec2VerifiedAccessEndpointItemFrom(endpoint))
	}
	return out
}

func ec2VerifiedAccessEndpointItemFrom(in ec2svc.VerifiedAccessEndpoint) ec2VerifiedAccessEndpointItem {
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	out := ec2VerifiedAccessEndpointItem{
		ApplicationDomain:        in.ApplicationDomain,
		AttachmentType:           in.AttachmentType,
		CreationTime:             in.CreationTime,
		Description:              in.Description,
		DomainCertificateARN:     in.DomainCertARN,
		EndpointDomain:           in.EndpointDomain,
		EndpointType:             in.EndpointType,
		LastUpdatedTime:          in.LastUpdatedTime,
		SecurityGroupIDSet:       ec2StringSet{Items: append([]string(nil), in.SecurityGroupIDs...)},
		Status:                   ec2VerifiedAccessEndpointStatusItem{Code: in.StatusCode, Message: in.StatusMessage},
		TagSet:                   ec2TagSet{Items: tags},
		VerifiedAccessEndpointID: in.ID,
		VerifiedAccessGroupID:    in.VerifiedGroup,
		VerifiedAccessInstanceID: in.VerifiedInstance,
	}
	if in.DeletionTime != nil {
		out.DeletionTime = *in.DeletionTime
	}
	return out
}

func ec2VerifiedAccessTrustProviderItemsFrom(in []ec2svc.VerifiedAccessTrustProvider) []ec2VerifiedAccessTrustProviderItem {
	out := make([]ec2VerifiedAccessTrustProviderItem, 0, len(in))
	for _, trustProvider := range in {
		out = append(out, ec2VerifiedAccessTrustProviderItemFrom(trustProvider))
	}
	return out
}

func ec2VerifiedAccessTrustProviderItemFrom(in ec2svc.VerifiedAccessTrustProvider) ec2VerifiedAccessTrustProviderItem {
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	return ec2VerifiedAccessTrustProviderItem{
		CreationTime:                  in.CreationTime,
		Description:                   in.Description,
		DeviceTrustProviderType:       in.DeviceTrustProviderType,
		LastUpdatedTime:               in.LastUpdatedTime,
		PolicyReferenceName:           in.PolicyReferenceName,
		TagSet:                        ec2TagSet{Items: tags},
		TrustProviderType:             in.TrustProviderType,
		UserTrustProviderType:         in.UserTrustProviderType,
		VerifiedAccessTrustProviderID: in.ID,
	}
}

type ec2CreateVerifiedAccessInstanceResponse struct {
	XMLName                xml.Name                      `xml:"CreateVerifiedAccessInstanceResponse"`
	Xmlns                  string                        `xml:"xmlns,attr"`
	RequestID              string                        `xml:"requestId"`
	VerifiedAccessInstance ec2VerifiedAccessInstanceItem `xml:"verifiedAccessInstance"`
}

type ec2DeleteVerifiedAccessInstanceResponse struct {
	XMLName                xml.Name                      `xml:"DeleteVerifiedAccessInstanceResponse"`
	Xmlns                  string                        `xml:"xmlns,attr"`
	RequestID              string                        `xml:"requestId"`
	VerifiedAccessInstance ec2VerifiedAccessInstanceItem `xml:"verifiedAccessInstance"`
}

type ec2DescribeVerifiedAccessInstancesResponse struct {
	XMLName                   xml.Name                     `xml:"DescribeVerifiedAccessInstancesResponse"`
	Xmlns                     string                       `xml:"xmlns,attr"`
	RequestID                 string                       `xml:"requestId"`
	NextToken                 string                       `xml:"nextToken,omitempty"`
	VerifiedAccessInstanceSet ec2VerifiedAccessInstanceSet `xml:"verifiedAccessInstanceSet"`
}

type ec2VerifiedAccessInstanceSet struct {
	Items []ec2VerifiedAccessInstanceItem `xml:"item"`
}

type ec2VerifiedAccessInstanceItem struct {
	CidrEndpointsCustomSubDomain string    `xml:"cidrEndpointsCustomSubDomain,omitempty"`
	CreationTime                 string    `xml:"creationTime,omitempty"`
	Description                  string    `xml:"description,omitempty"`
	FIPSEnabled                  *bool     `xml:"fipsEnabled,omitempty"`
	LastUpdatedTime              string    `xml:"lastUpdatedTime,omitempty"`
	TagSet                       ec2TagSet `xml:"tagSet"`
	VerifiedAccessInstanceID     string    `xml:"verifiedAccessInstanceId,omitempty"`
}

type ec2CreateVerifiedAccessGroupResponse struct {
	XMLName             xml.Name                   `xml:"CreateVerifiedAccessGroupResponse"`
	Xmlns               string                     `xml:"xmlns,attr"`
	RequestID           string                     `xml:"requestId"`
	VerifiedAccessGroup ec2VerifiedAccessGroupItem `xml:"verifiedAccessGroup"`
}

type ec2DeleteVerifiedAccessGroupResponse struct {
	XMLName             xml.Name                   `xml:"DeleteVerifiedAccessGroupResponse"`
	Xmlns               string                     `xml:"xmlns,attr"`
	RequestID           string                     `xml:"requestId"`
	VerifiedAccessGroup ec2VerifiedAccessGroupItem `xml:"verifiedAccessGroup"`
}

type ec2DescribeVerifiedAccessGroupsResponse struct {
	XMLName                xml.Name                  `xml:"DescribeVerifiedAccessGroupsResponse"`
	Xmlns                  string                    `xml:"xmlns,attr"`
	RequestID              string                    `xml:"requestId"`
	NextToken              string                    `xml:"nextToken,omitempty"`
	VerifiedAccessGroupSet ec2VerifiedAccessGroupSet `xml:"verifiedAccessGroupSet"`
}

type ec2VerifiedAccessGroupSet struct {
	Items []ec2VerifiedAccessGroupItem `xml:"item"`
}

type ec2VerifiedAccessGroupItem struct {
	CreationTime             string    `xml:"creationTime,omitempty"`
	DeletionTime             string    `xml:"deletionTime,omitempty"`
	Description              string    `xml:"description,omitempty"`
	LastUpdatedTime          string    `xml:"lastUpdatedTime,omitempty"`
	Owner                    string    `xml:"owner,omitempty"`
	TagSet                   ec2TagSet `xml:"tagSet"`
	VerifiedAccessGroupARN   string    `xml:"verifiedAccessGroupArn,omitempty"`
	VerifiedAccessGroupID    string    `xml:"verifiedAccessGroupId,omitempty"`
	VerifiedAccessInstanceID string    `xml:"verifiedAccessInstanceId,omitempty"`
}

type ec2CreateVerifiedAccessEndpointResponse struct {
	XMLName                xml.Name                      `xml:"CreateVerifiedAccessEndpointResponse"`
	Xmlns                  string                        `xml:"xmlns,attr"`
	RequestID              string                        `xml:"requestId"`
	VerifiedAccessEndpoint ec2VerifiedAccessEndpointItem `xml:"verifiedAccessEndpoint"`
}

type ec2DeleteVerifiedAccessEndpointResponse struct {
	XMLName                xml.Name                      `xml:"DeleteVerifiedAccessEndpointResponse"`
	Xmlns                  string                        `xml:"xmlns,attr"`
	RequestID              string                        `xml:"requestId"`
	VerifiedAccessEndpoint ec2VerifiedAccessEndpointItem `xml:"verifiedAccessEndpoint"`
}

type ec2DescribeVerifiedAccessEndpointsResponse struct {
	XMLName                   xml.Name                     `xml:"DescribeVerifiedAccessEndpointsResponse"`
	Xmlns                     string                       `xml:"xmlns,attr"`
	RequestID                 string                       `xml:"requestId"`
	NextToken                 string                       `xml:"nextToken,omitempty"`
	VerifiedAccessEndpointSet ec2VerifiedAccessEndpointSet `xml:"verifiedAccessEndpointSet"`
}

type ec2VerifiedAccessEndpointSet struct {
	Items []ec2VerifiedAccessEndpointItem `xml:"item"`
}

type ec2VerifiedAccessEndpointItem struct {
	ApplicationDomain        string                              `xml:"applicationDomain,omitempty"`
	AttachmentType           string                              `xml:"attachmentType,omitempty"`
	CreationTime             string                              `xml:"creationTime,omitempty"`
	DeletionTime             string                              `xml:"deletionTime,omitempty"`
	Description              string                              `xml:"description,omitempty"`
	DomainCertificateARN     string                              `xml:"domainCertificateArn,omitempty"`
	EndpointDomain           string                              `xml:"endpointDomain,omitempty"`
	EndpointType             string                              `xml:"endpointType,omitempty"`
	LastUpdatedTime          string                              `xml:"lastUpdatedTime,omitempty"`
	SecurityGroupIDSet       ec2StringSet                        `xml:"securityGroupIdSet"`
	Status                   ec2VerifiedAccessEndpointStatusItem `xml:"status"`
	TagSet                   ec2TagSet                           `xml:"tagSet"`
	VerifiedAccessEndpointID string                              `xml:"verifiedAccessEndpointId,omitempty"`
	VerifiedAccessGroupID    string                              `xml:"verifiedAccessGroupId,omitempty"`
	VerifiedAccessInstanceID string                              `xml:"verifiedAccessInstanceId,omitempty"`
}

type ec2VerifiedAccessEndpointStatusItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2CreateVerifiedAccessTrustProviderResponse struct {
	XMLName                     xml.Name                           `xml:"CreateVerifiedAccessTrustProviderResponse"`
	Xmlns                       string                             `xml:"xmlns,attr"`
	RequestID                   string                             `xml:"requestId"`
	VerifiedAccessTrustProvider ec2VerifiedAccessTrustProviderItem `xml:"verifiedAccessTrustProvider"`
}

type ec2DeleteVerifiedAccessTrustProviderResponse struct {
	XMLName                     xml.Name                           `xml:"DeleteVerifiedAccessTrustProviderResponse"`
	Xmlns                       string                             `xml:"xmlns,attr"`
	RequestID                   string                             `xml:"requestId"`
	VerifiedAccessTrustProvider ec2VerifiedAccessTrustProviderItem `xml:"verifiedAccessTrustProvider"`
}

type ec2DescribeVerifiedAccessTrustProvidersResponse struct {
	XMLName                        xml.Name                          `xml:"DescribeVerifiedAccessTrustProvidersResponse"`
	Xmlns                          string                            `xml:"xmlns,attr"`
	RequestID                      string                            `xml:"requestId"`
	NextToken                      string                            `xml:"nextToken,omitempty"`
	VerifiedAccessTrustProviderSet ec2VerifiedAccessTrustProviderSet `xml:"verifiedAccessTrustProviderSet"`
}

type ec2VerifiedAccessTrustProviderSet struct {
	Items []ec2VerifiedAccessTrustProviderItem `xml:"item"`
}

type ec2VerifiedAccessTrustProviderItem struct {
	CreationTime                  string    `xml:"creationTime,omitempty"`
	Description                   string    `xml:"description,omitempty"`
	DeviceTrustProviderType       string    `xml:"deviceTrustProviderType,omitempty"`
	LastUpdatedTime               string    `xml:"lastUpdatedTime,omitempty"`
	PolicyReferenceName           string    `xml:"policyReferenceName,omitempty"`
	TagSet                        ec2TagSet `xml:"tagSet"`
	TrustProviderType             string    `xml:"trustProviderType,omitempty"`
	UserTrustProviderType         string    `xml:"userTrustProviderType,omitempty"`
	VerifiedAccessTrustProviderID string    `xml:"verifiedAccessTrustProviderId,omitempty"`
}
