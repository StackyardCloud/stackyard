package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage69Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AttachVerifiedAccessTrustProvider":
		instance, trustProvider, err := s.ec2.AttachVerifiedAccessTrustProvider(
			strings.TrimSpace(r.Form.Get("VerifiedAccessInstanceId")),
			strings.TrimSpace(r.Form.Get("VerifiedAccessTrustProviderId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AttachVerifiedAccessTrustProviderResponse{
			XMLName:                     xml.Name{Local: "AttachVerifiedAccessTrustProviderResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			VerifiedAccessInstance:      ec2VerifiedAccessInstanceItemFrom(instance),
			VerifiedAccessTrustProvider: ec2VerifiedAccessTrustProviderItemFrom(trustProvider),
		})
		return true
	case "DescribeVerifiedAccessInstanceLoggingConfigurations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		loggingConfigurations, nextToken, err := s.ec2.DescribeVerifiedAccessInstanceLoggingConfigurations(
			parseEC2Members(r.Form, "VerifiedAccessInstanceId."),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVerifiedAccessInstanceLoggingConfigurationsResponse{
			XMLName:   xml.Name{Local: "DescribeVerifiedAccessInstanceLoggingConfigurationsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			LoggingConfigurationSet: ec2VerifiedAccessInstanceLoggingConfigurationSet{
				Items: ec2VerifiedAccessInstanceLoggingConfigurationItemsFrom(loggingConfigurations),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "DetachVerifiedAccessTrustProvider":
		instance, trustProvider, err := s.ec2.DetachVerifiedAccessTrustProvider(
			strings.TrimSpace(r.Form.Get("VerifiedAccessInstanceId")),
			strings.TrimSpace(r.Form.Get("VerifiedAccessTrustProviderId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DetachVerifiedAccessTrustProviderResponse{
			XMLName:                     xml.Name{Local: "DetachVerifiedAccessTrustProviderResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			VerifiedAccessInstance:      ec2VerifiedAccessInstanceItemFrom(instance),
			VerifiedAccessTrustProvider: ec2VerifiedAccessTrustProviderItemFrom(trustProvider),
		})
		return true
	case "ExportVerifiedAccessInstanceClientConfiguration":
		config, err := s.ec2.ExportVerifiedAccessInstanceClientConfiguration(strings.TrimSpace(r.Form.Get("VerifiedAccessInstanceId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ExportVerifiedAccessInstanceClientConfigurationResponse{
			XMLName:                  xml.Name{Local: "ExportVerifiedAccessInstanceClientConfigurationResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			DeviceTrustProviderSet:   ec2StringSet{Items: append([]string(nil), config.DeviceTrustProviders...)},
			OpenVPNConfigurationSet:  ec2VerifiedAccessInstanceOpenVPNClientConfigurationSet{Items: ec2VerifiedAccessInstanceOpenVPNClientConfigurationItemsFrom(config.OpenVPNConfigurations)},
			Region:                   config.Region,
			UserTrustProvider:        ec2VerifiedAccessInstanceUserTrustProviderClientConfigurationItemFrom(config.UserTrustProvider),
			VerifiedAccessInstanceID: config.VerifiedAccessInstanceID,
			Version:                  config.Version,
		})
		return true
	case "GetVerifiedAccessEndpointPolicy":
		policy, err := s.ec2.GetVerifiedAccessEndpointPolicy(strings.TrimSpace(r.Form.Get("VerifiedAccessEndpointId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2GetVerifiedAccessEndpointPolicyResponse{
			XMLName:        xml.Name{Local: "GetVerifiedAccessEndpointPolicyResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			PolicyDocument: policy.PolicyDocument,
			PolicyEnabled:  policy.PolicyEnabled,
		})
		return true
	case "GetVerifiedAccessEndpointTargets":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		targets, nextToken, err := s.ec2.GetVerifiedAccessEndpointTargets(
			strings.TrimSpace(r.Form.Get("VerifiedAccessEndpointId")),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2GetVerifiedAccessEndpointTargetsResponse{
			XMLName:   xml.Name{Local: "GetVerifiedAccessEndpointTargetsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VerifiedAccessEndpointTargetSet: ec2VerifiedAccessEndpointTargetSet{
				Items: ec2VerifiedAccessEndpointTargetItemsFrom(targets),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "GetVerifiedAccessGroupPolicy":
		policy, err := s.ec2.GetVerifiedAccessGroupPolicy(strings.TrimSpace(r.Form.Get("VerifiedAccessGroupId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2GetVerifiedAccessGroupPolicyResponse{
			XMLName:        xml.Name{Local: "GetVerifiedAccessGroupPolicyResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			PolicyDocument: policy.PolicyDocument,
			PolicyEnabled:  policy.PolicyEnabled,
		})
		return true
	case "ModifyVerifiedAccessEndpoint":
		endpoint, err := s.ec2.ModifyVerifiedAccessEndpoint(
			strings.TrimSpace(r.Form.Get("VerifiedAccessEndpointId")),
			parseEC2OptionalString(r.Form.Get("VerifiedAccessGroupId")),
			parseEC2OptionalString(r.Form.Get("Description")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVerifiedAccessEndpointResponse{
			XMLName:                xml.Name{Local: "ModifyVerifiedAccessEndpointResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			VerifiedAccessEndpoint: ec2VerifiedAccessEndpointItemFrom(endpoint),
		})
		return true
	case "ModifyVerifiedAccessEndpointPolicy":
		policyEnabled, hasPolicyEnabled, ok := ec2OptionalBoolFromForm(r.Form, "PolicyEnabled")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasPolicyEnabled {
			policyEnabled = nil
		}
		policy, err := s.ec2.ModifyVerifiedAccessEndpointPolicy(
			strings.TrimSpace(r.Form.Get("VerifiedAccessEndpointId")),
			parseEC2OptionalString(r.Form.Get("PolicyDocument")),
			policyEnabled,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVerifiedAccessEndpointPolicyResponse{
			XMLName:        xml.Name{Local: "ModifyVerifiedAccessEndpointPolicyResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			PolicyDocument: policy.PolicyDocument,
			PolicyEnabled:  policy.PolicyEnabled,
		})
		return true
	case "ModifyVerifiedAccessGroup":
		group, err := s.ec2.ModifyVerifiedAccessGroup(
			strings.TrimSpace(r.Form.Get("VerifiedAccessGroupId")),
			parseEC2OptionalString(r.Form.Get("VerifiedAccessInstanceId")),
			parseEC2OptionalString(r.Form.Get("Description")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVerifiedAccessGroupResponse{
			XMLName:             xml.Name{Local: "ModifyVerifiedAccessGroupResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			VerifiedAccessGroup: ec2VerifiedAccessGroupItemFrom(group),
		})
		return true
	case "ModifyVerifiedAccessGroupPolicy":
		policyEnabled, hasPolicyEnabled, ok := ec2OptionalBoolFromForm(r.Form, "PolicyEnabled")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasPolicyEnabled {
			policyEnabled = nil
		}
		policy, err := s.ec2.ModifyVerifiedAccessGroupPolicy(
			strings.TrimSpace(r.Form.Get("VerifiedAccessGroupId")),
			parseEC2OptionalString(r.Form.Get("PolicyDocument")),
			policyEnabled,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVerifiedAccessGroupPolicyResponse{
			XMLName:        xml.Name{Local: "ModifyVerifiedAccessGroupPolicyResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			PolicyDocument: policy.PolicyDocument,
			PolicyEnabled:  policy.PolicyEnabled,
		})
		return true
	case "ModifyVerifiedAccessInstance":
		instance, err := s.ec2.ModifyVerifiedAccessInstance(
			strings.TrimSpace(r.Form.Get("VerifiedAccessInstanceId")),
			parseEC2OptionalString(r.Form.Get("CidrEndpointsCustomSubDomain")),
			parseEC2OptionalString(r.Form.Get("Description")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVerifiedAccessInstanceResponse{
			XMLName:                xml.Name{Local: "ModifyVerifiedAccessInstanceResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			VerifiedAccessInstance: ec2VerifiedAccessInstanceItemFrom(instance),
		})
		return true
	case "ModifyVerifiedAccessInstanceLoggingConfiguration":
		accessLogs, ok := parseEC2VerifiedAccessLogOptions(r.Form)
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		loggingConfiguration, err := s.ec2.ModifyVerifiedAccessInstanceLoggingConfiguration(
			strings.TrimSpace(r.Form.Get("VerifiedAccessInstanceId")),
			accessLogs,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVerifiedAccessInstanceLoggingConfigurationResponse{
			XMLName:              xml.Name{Local: "ModifyVerifiedAccessInstanceLoggingConfigurationResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			LoggingConfiguration: ec2VerifiedAccessInstanceLoggingConfigurationItemFrom(loggingConfiguration),
		})
		return true
	case "ModifyVerifiedAccessTrustProvider":
		trustProvider, err := s.ec2.ModifyVerifiedAccessTrustProvider(
			strings.TrimSpace(r.Form.Get("VerifiedAccessTrustProviderId")),
			parseEC2OptionalString(r.Form.Get("Description")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVerifiedAccessTrustProviderResponse{
			XMLName:                     xml.Name{Local: "ModifyVerifiedAccessTrustProviderResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			VerifiedAccessTrustProvider: ec2VerifiedAccessTrustProviderItemFrom(trustProvider),
		})
		return true
	default:
		return false
	}
}

func parseEC2VerifiedAccessLogOptions(form url.Values) (ec2svc.VerifiedAccessLoggingOptions, bool) {
	accessLogs := ec2svc.VerifiedAccessLoggingOptions{}
	hasAccessLogs := false
	for key := range form {
		if strings.HasPrefix(key, "AccessLogs.") {
			hasAccessLogs = true
			break
		}
	}
	if !hasAccessLogs {
		return ec2svc.VerifiedAccessLoggingOptions{}, false
	}

	includeTrustContext, hasIncludeTrustContext, ok := ec2OptionalBoolFromForm(form, "AccessLogs.IncludeTrustContext")
	if !ok {
		return ec2svc.VerifiedAccessLoggingOptions{}, false
	}
	if hasIncludeTrustContext && includeTrustContext != nil {
		accessLogs.IncludeTrustContext = *includeTrustContext
	}
	if logVersion := strings.TrimSpace(form.Get("AccessLogs.LogVersion")); logVersion != "" {
		accessLogs.LogVersion = logVersion
	}

	cloudWatchEnabled, hasCloudWatchEnabled, ok := ec2OptionalBoolFromForm(form, "AccessLogs.CloudWatchLogs.Enabled")
	if !ok {
		return ec2svc.VerifiedAccessLoggingOptions{}, false
	}
	if hasCloudWatchEnabled && cloudWatchEnabled != nil {
		accessLogs.CloudWatchLogsEnabled = *cloudWatchEnabled
	}
	if cloudWatchLogGroup := strings.TrimSpace(form.Get("AccessLogs.CloudWatchLogs.LogGroup")); cloudWatchLogGroup != "" {
		accessLogs.CloudWatchLogGroup = cloudWatchLogGroup
	}

	kinesisEnabled, hasKinesisEnabled, ok := ec2OptionalBoolFromForm(form, "AccessLogs.KinesisDataFirehose.Enabled")
	if !ok {
		return ec2svc.VerifiedAccessLoggingOptions{}, false
	}
	if hasKinesisEnabled && kinesisEnabled != nil {
		accessLogs.KinesisFirehoseEnabled = *kinesisEnabled
	}
	if deliveryStream := strings.TrimSpace(form.Get("AccessLogs.KinesisDataFirehose.DeliveryStream")); deliveryStream != "" {
		accessLogs.KinesisDeliveryStream = deliveryStream
	}

	s3Enabled, hasS3Enabled, ok := ec2OptionalBoolFromForm(form, "AccessLogs.S3.Enabled")
	if !ok {
		return ec2svc.VerifiedAccessLoggingOptions{}, false
	}
	if hasS3Enabled && s3Enabled != nil {
		accessLogs.S3Enabled = *s3Enabled
	}
	if bucketName := strings.TrimSpace(form.Get("AccessLogs.S3.BucketName")); bucketName != "" {
		accessLogs.S3BucketName = bucketName
	}
	if bucketOwner := strings.TrimSpace(form.Get("AccessLogs.S3.BucketOwner")); bucketOwner != "" {
		accessLogs.S3BucketOwner = bucketOwner
	}
	if prefix := strings.TrimSpace(form.Get("AccessLogs.S3.Prefix")); prefix != "" {
		accessLogs.S3Prefix = prefix
	}

	return accessLogs, true
}

func ec2VerifiedAccessInstanceLoggingConfigurationItemsFrom(in []ec2svc.VerifiedAccessInstanceLoggingConfiguration) []ec2VerifiedAccessInstanceLoggingConfigurationItem {
	out := make([]ec2VerifiedAccessInstanceLoggingConfigurationItem, 0, len(in))
	for _, cfg := range in {
		out = append(out, ec2VerifiedAccessInstanceLoggingConfigurationItemFrom(cfg))
	}
	return out
}

func ec2VerifiedAccessInstanceLoggingConfigurationItemFrom(in ec2svc.VerifiedAccessInstanceLoggingConfiguration) ec2VerifiedAccessInstanceLoggingConfigurationItem {
	return ec2VerifiedAccessInstanceLoggingConfigurationItem{
		AccessLogs:               ec2VerifiedAccessLogsItemFrom(in.AccessLogs),
		VerifiedAccessInstanceID: in.VerifiedAccessInstanceID,
	}
}

func ec2VerifiedAccessLogsItemFrom(in ec2svc.VerifiedAccessLoggingOptions) ec2VerifiedAccessLogsItem {
	return ec2VerifiedAccessLogsItem{
		CloudWatchLogs: ec2VerifiedAccessLogCloudWatchLogsDestinationItem{
			Enabled:  in.CloudWatchLogsEnabled,
			LogGroup: in.CloudWatchLogGroup,
		},
		IncludeTrustContext: in.IncludeTrustContext,
		KinesisDataFirehose: ec2VerifiedAccessLogKinesisDataFirehoseDestinationItem{
			DeliveryStream: in.KinesisDeliveryStream,
			Enabled:        in.KinesisFirehoseEnabled,
		},
		LogVersion: in.LogVersion,
		S3: ec2VerifiedAccessLogS3DestinationItem{
			BucketName:  in.S3BucketName,
			BucketOwner: in.S3BucketOwner,
			Enabled:     in.S3Enabled,
			Prefix:      in.S3Prefix,
		},
	}
}

func ec2VerifiedAccessEndpointTargetItemsFrom(in []ec2svc.VerifiedAccessEndpointTarget) []ec2VerifiedAccessEndpointTargetItem {
	out := make([]ec2VerifiedAccessEndpointTargetItem, 0, len(in))
	for _, target := range in {
		out = append(out, ec2VerifiedAccessEndpointTargetItem{
			VerifiedAccessEndpointID:              target.VerifiedAccessEndpointID,
			VerifiedAccessEndpointTargetDNS:       target.VerifiedAccessEndpointTargetDNS,
			VerifiedAccessEndpointTargetIPAddress: target.VerifiedAccessEndpointTargetIPAddress,
		})
	}
	return out
}

func ec2VerifiedAccessInstanceOpenVPNClientConfigurationItemsFrom(in []ec2svc.VerifiedAccessInstanceOpenVPNClientConfiguration) []ec2VerifiedAccessInstanceOpenVPNClientConfigurationItem {
	out := make([]ec2VerifiedAccessInstanceOpenVPNClientConfigurationItem, 0, len(in))
	for _, cfg := range in {
		routes := make([]ec2VerifiedAccessInstanceOpenVPNClientConfigurationRouteItem, 0, len(cfg.Routes))
		for _, route := range cfg.Routes {
			routes = append(routes, ec2VerifiedAccessInstanceOpenVPNClientConfigurationRouteItem{CIDR: route.CIDR})
		}
		out = append(out, ec2VerifiedAccessInstanceOpenVPNClientConfigurationItem{
			Config:   cfg.Config,
			RouteSet: ec2VerifiedAccessInstanceOpenVPNClientConfigurationRouteSet{Items: routes},
		})
	}
	return out
}

func ec2VerifiedAccessInstanceUserTrustProviderClientConfigurationItemFrom(in *ec2svc.VerifiedAccessInstanceUserTrustProviderClientConfiguration) *ec2VerifiedAccessInstanceUserTrustProviderClientConfigurationItem {
	if in == nil {
		return nil
	}
	out := &ec2VerifiedAccessInstanceUserTrustProviderClientConfigurationItem{
		AuthorizationEndpoint:    in.AuthorizationEndpoint,
		ClientID:                 in.ClientID,
		ClientSecret:             in.ClientSecret,
		Issuer:                   in.Issuer,
		PKCEEnabled:              in.PKCEEnabled,
		PublicSigningKeyEndpoint: in.PublicSigningKeyEndpoint,
		Scopes:                   in.Scopes,
		TokenEndpoint:            in.TokenEndpoint,
		Type:                     in.Type,
		UserInfoEndpoint:         in.UserInfoEndpoint,
	}
	return out
}

type ec2AttachVerifiedAccessTrustProviderResponse struct {
	XMLName                     xml.Name                           `xml:"AttachVerifiedAccessTrustProviderResponse"`
	Xmlns                       string                             `xml:"xmlns,attr"`
	RequestID                   string                             `xml:"requestId"`
	VerifiedAccessInstance      ec2VerifiedAccessInstanceItem      `xml:"verifiedAccessInstance"`
	VerifiedAccessTrustProvider ec2VerifiedAccessTrustProviderItem `xml:"verifiedAccessTrustProvider"`
}

type ec2DescribeVerifiedAccessInstanceLoggingConfigurationsResponse struct {
	XMLName                 xml.Name                                         `xml:"DescribeVerifiedAccessInstanceLoggingConfigurationsResponse"`
	Xmlns                   string                                           `xml:"xmlns,attr"`
	RequestID               string                                           `xml:"requestId"`
	LoggingConfigurationSet ec2VerifiedAccessInstanceLoggingConfigurationSet `xml:"loggingConfigurationSet"`
	NextToken               string                                           `xml:"nextToken,omitempty"`
}

type ec2DetachVerifiedAccessTrustProviderResponse struct {
	XMLName                     xml.Name                           `xml:"DetachVerifiedAccessTrustProviderResponse"`
	Xmlns                       string                             `xml:"xmlns,attr"`
	RequestID                   string                             `xml:"requestId"`
	VerifiedAccessInstance      ec2VerifiedAccessInstanceItem      `xml:"verifiedAccessInstance"`
	VerifiedAccessTrustProvider ec2VerifiedAccessTrustProviderItem `xml:"verifiedAccessTrustProvider"`
}

type ec2ExportVerifiedAccessInstanceClientConfigurationResponse struct {
	XMLName                  xml.Name                                                           `xml:"ExportVerifiedAccessInstanceClientConfigurationResponse"`
	Xmlns                    string                                                             `xml:"xmlns,attr"`
	RequestID                string                                                             `xml:"requestId"`
	DeviceTrustProviderSet   ec2StringSet                                                       `xml:"deviceTrustProviderSet"`
	OpenVPNConfigurationSet  ec2VerifiedAccessInstanceOpenVPNClientConfigurationSet             `xml:"openVpnConfigurationSet"`
	Region                   string                                                             `xml:"region,omitempty"`
	UserTrustProvider        *ec2VerifiedAccessInstanceUserTrustProviderClientConfigurationItem `xml:"userTrustProvider,omitempty"`
	VerifiedAccessInstanceID string                                                             `xml:"verifiedAccessInstanceId,omitempty"`
	Version                  string                                                             `xml:"version,omitempty"`
}

type ec2GetVerifiedAccessEndpointPolicyResponse struct {
	XMLName        xml.Name `xml:"GetVerifiedAccessEndpointPolicyResponse"`
	Xmlns          string   `xml:"xmlns,attr"`
	RequestID      string   `xml:"requestId"`
	PolicyDocument string   `xml:"policyDocument,omitempty"`
	PolicyEnabled  bool     `xml:"policyEnabled"`
}

type ec2GetVerifiedAccessEndpointTargetsResponse struct {
	XMLName                         xml.Name                           `xml:"GetVerifiedAccessEndpointTargetsResponse"`
	Xmlns                           string                             `xml:"xmlns,attr"`
	RequestID                       string                             `xml:"requestId"`
	NextToken                       string                             `xml:"nextToken,omitempty"`
	VerifiedAccessEndpointTargetSet ec2VerifiedAccessEndpointTargetSet `xml:"verifiedAccessEndpointTargetSet"`
}

type ec2GetVerifiedAccessGroupPolicyResponse struct {
	XMLName        xml.Name `xml:"GetVerifiedAccessGroupPolicyResponse"`
	Xmlns          string   `xml:"xmlns,attr"`
	RequestID      string   `xml:"requestId"`
	PolicyDocument string   `xml:"policyDocument,omitempty"`
	PolicyEnabled  bool     `xml:"policyEnabled"`
}

type ec2ModifyVerifiedAccessEndpointResponse struct {
	XMLName                xml.Name                      `xml:"ModifyVerifiedAccessEndpointResponse"`
	Xmlns                  string                        `xml:"xmlns,attr"`
	RequestID              string                        `xml:"requestId"`
	VerifiedAccessEndpoint ec2VerifiedAccessEndpointItem `xml:"verifiedAccessEndpoint"`
}

type ec2ModifyVerifiedAccessEndpointPolicyResponse struct {
	XMLName        xml.Name `xml:"ModifyVerifiedAccessEndpointPolicyResponse"`
	Xmlns          string   `xml:"xmlns,attr"`
	RequestID      string   `xml:"requestId"`
	PolicyDocument string   `xml:"policyDocument,omitempty"`
	PolicyEnabled  bool     `xml:"policyEnabled"`
}

type ec2ModifyVerifiedAccessGroupResponse struct {
	XMLName             xml.Name                   `xml:"ModifyVerifiedAccessGroupResponse"`
	Xmlns               string                     `xml:"xmlns,attr"`
	RequestID           string                     `xml:"requestId"`
	VerifiedAccessGroup ec2VerifiedAccessGroupItem `xml:"verifiedAccessGroup"`
}

type ec2ModifyVerifiedAccessGroupPolicyResponse struct {
	XMLName        xml.Name `xml:"ModifyVerifiedAccessGroupPolicyResponse"`
	Xmlns          string   `xml:"xmlns,attr"`
	RequestID      string   `xml:"requestId"`
	PolicyDocument string   `xml:"policyDocument,omitempty"`
	PolicyEnabled  bool     `xml:"policyEnabled"`
}

type ec2ModifyVerifiedAccessInstanceResponse struct {
	XMLName                xml.Name                      `xml:"ModifyVerifiedAccessInstanceResponse"`
	Xmlns                  string                        `xml:"xmlns,attr"`
	RequestID              string                        `xml:"requestId"`
	VerifiedAccessInstance ec2VerifiedAccessInstanceItem `xml:"verifiedAccessInstance"`
}

type ec2ModifyVerifiedAccessInstanceLoggingConfigurationResponse struct {
	XMLName              xml.Name                                          `xml:"ModifyVerifiedAccessInstanceLoggingConfigurationResponse"`
	Xmlns                string                                            `xml:"xmlns,attr"`
	RequestID            string                                            `xml:"requestId"`
	LoggingConfiguration ec2VerifiedAccessInstanceLoggingConfigurationItem `xml:"loggingConfiguration"`
}

type ec2ModifyVerifiedAccessTrustProviderResponse struct {
	XMLName                     xml.Name                           `xml:"ModifyVerifiedAccessTrustProviderResponse"`
	Xmlns                       string                             `xml:"xmlns,attr"`
	RequestID                   string                             `xml:"requestId"`
	VerifiedAccessTrustProvider ec2VerifiedAccessTrustProviderItem `xml:"verifiedAccessTrustProvider"`
}

type ec2VerifiedAccessEndpointTargetSet struct {
	Items []ec2VerifiedAccessEndpointTargetItem `xml:"item"`
}

type ec2VerifiedAccessEndpointTargetItem struct {
	VerifiedAccessEndpointID              string `xml:"verifiedAccessEndpointId,omitempty"`
	VerifiedAccessEndpointTargetDNS       string `xml:"verifiedAccessEndpointTargetDns,omitempty"`
	VerifiedAccessEndpointTargetIPAddress string `xml:"verifiedAccessEndpointTargetIpAddress,omitempty"`
}

type ec2VerifiedAccessInstanceLoggingConfigurationSet struct {
	Items []ec2VerifiedAccessInstanceLoggingConfigurationItem `xml:"item"`
}

type ec2VerifiedAccessInstanceLoggingConfigurationItem struct {
	AccessLogs               ec2VerifiedAccessLogsItem `xml:"accessLogs"`
	VerifiedAccessInstanceID string                    `xml:"verifiedAccessInstanceId,omitempty"`
}

type ec2VerifiedAccessLogsItem struct {
	CloudWatchLogs      ec2VerifiedAccessLogCloudWatchLogsDestinationItem      `xml:"cloudWatchLogs"`
	IncludeTrustContext bool                                                   `xml:"includeTrustContext"`
	KinesisDataFirehose ec2VerifiedAccessLogKinesisDataFirehoseDestinationItem `xml:"kinesisDataFirehose"`
	LogVersion          string                                                 `xml:"logVersion,omitempty"`
	S3                  ec2VerifiedAccessLogS3DestinationItem                  `xml:"s3"`
}

type ec2VerifiedAccessLogCloudWatchLogsDestinationItem struct {
	Enabled  bool   `xml:"enabled"`
	LogGroup string `xml:"logGroup,omitempty"`
}

type ec2VerifiedAccessLogKinesisDataFirehoseDestinationItem struct {
	DeliveryStream string `xml:"deliveryStream,omitempty"`
	Enabled        bool   `xml:"enabled"`
}

type ec2VerifiedAccessLogS3DestinationItem struct {
	BucketName  string `xml:"bucketName,omitempty"`
	BucketOwner string `xml:"bucketOwner,omitempty"`
	Enabled     bool   `xml:"enabled"`
	Prefix      string `xml:"prefix,omitempty"`
}

type ec2VerifiedAccessInstanceOpenVPNClientConfigurationSet struct {
	Items []ec2VerifiedAccessInstanceOpenVPNClientConfigurationItem `xml:"item"`
}

type ec2VerifiedAccessInstanceOpenVPNClientConfigurationItem struct {
	Config   string                                                      `xml:"config,omitempty"`
	RouteSet ec2VerifiedAccessInstanceOpenVPNClientConfigurationRouteSet `xml:"routeSet"`
}

type ec2VerifiedAccessInstanceOpenVPNClientConfigurationRouteSet struct {
	Items []ec2VerifiedAccessInstanceOpenVPNClientConfigurationRouteItem `xml:"item"`
}

type ec2VerifiedAccessInstanceOpenVPNClientConfigurationRouteItem struct {
	CIDR string `xml:"cidr,omitempty"`
}

type ec2VerifiedAccessInstanceUserTrustProviderClientConfigurationItem struct {
	AuthorizationEndpoint    string `xml:"authorizationEndpoint,omitempty"`
	ClientID                 string `xml:"clientId,omitempty"`
	ClientSecret             string `xml:"clientSecret,omitempty"`
	Issuer                   string `xml:"issuer,omitempty"`
	PKCEEnabled              bool   `xml:"pkceEnabled"`
	PublicSigningKeyEndpoint string `xml:"publicSigningKeyEndpoint,omitempty"`
	Scopes                   string `xml:"scopes,omitempty"`
	TokenEndpoint            string `xml:"tokenEndpoint,omitempty"`
	Type                     string `xml:"type,omitempty"`
	UserInfoEndpoint         string `xml:"userInfoEndpoint,omitempty"`
}
