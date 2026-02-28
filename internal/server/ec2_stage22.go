package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage22Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "EnableVgwRoutePropagation":
		if err := s.ec2.EnableVgwRoutePropagation(
			strings.TrimSpace(r.Form.Get("RouteTableId")),
			strings.TrimSpace(r.Form.Get("GatewayId")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2EnableVgwRoutePropagationResponse{
			XMLName:   xml.Name{Local: "EnableVgwRoutePropagationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	case "DisableVgwRoutePropagation":
		if err := s.ec2.DisableVgwRoutePropagation(
			strings.TrimSpace(r.Form.Get("RouteTableId")),
			strings.TrimSpace(r.Form.Get("GatewayId")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DisableVgwRoutePropagationResponse{
			XMLName:   xml.Name{Local: "DisableVgwRoutePropagationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	case "GetActiveVpnTunnelStatus":
		status, err := s.ec2.GetActiveVpnTunnelStatus(
			strings.TrimSpace(r.Form.Get("VpnConnectionId")),
			strings.TrimSpace(r.Form.Get("VpnTunnelOutsideIpAddress")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2GetActiveVpnTunnelStatusResponse{
			XMLName:               xml.Name{Local: "GetActiveVpnTunnelStatusResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			ActiveVpnTunnelStatus: ec2ActiveVpnTunnelStatusItemFrom(status),
		})
		return true
	case "GetVpnConnectionDeviceSampleConfiguration":
		config, err := s.ec2.GetVpnConnectionDeviceSampleConfiguration(
			strings.TrimSpace(r.Form.Get("VpnConnectionId")),
			strings.TrimSpace(r.Form.Get("VpnConnectionDeviceTypeId")),
			strings.TrimSpace(r.Form.Get("InternetKeyExchangeVersion")),
			strings.TrimSpace(r.Form.Get("SampleType")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2GetVpnConnectionDeviceSampleConfigurationResponse{
			XMLName:                                xml.Name{Local: "GetVpnConnectionDeviceSampleConfigurationResponse"},
			Xmlns:                                  ec2Namespace,
			RequestID:                              "stackyard-request",
			VpnConnectionDeviceSampleConfiguration: config,
		})
		return true
	case "GetVpnConnectionDeviceTypes":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		deviceTypes, nextToken, err := s.ec2.GetVpnConnectionDeviceTypes(
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2GetVpnConnectionDeviceTypesResponse{
			XMLName:   xml.Name{Local: "GetVpnConnectionDeviceTypesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpnConnectionDeviceTypeSet: ec2VpnConnectionDeviceTypeSet{
				Items: ec2VpnConnectionDeviceTypeItems(deviceTypes),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "GetVpnTunnelReplacementStatus":
		status, err := s.ec2.GetVpnTunnelReplacementStatus(
			strings.TrimSpace(r.Form.Get("VpnConnectionId")),
			strings.TrimSpace(r.Form.Get("VpnTunnelOutsideIpAddress")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2GetVpnTunnelReplacementStatusResponse{
			XMLName:                   xml.Name{Local: "GetVpnTunnelReplacementStatusResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			CustomerGatewayID:         status.CustomerGatewayID,
			MaintenanceDetails:        ec2MaintenanceDetailsItemFrom(status.MaintenanceDetails),
			TransitGatewayID:          status.TransitGatewayID,
			VpnConnectionID:           status.VpnConnectionID,
			VpnGatewayID:              status.VpnGatewayID,
			VpnTunnelOutsideIPAddress: status.VpnTunnelOutsideIPAddress,
		})
		return true
	case "ModifyVpnTunnelCertificate":
		connection, err := s.ec2.ModifyVpnTunnelCertificate(
			strings.TrimSpace(r.Form.Get("VpnConnectionId")),
			strings.TrimSpace(r.Form.Get("VpnTunnelOutsideIpAddress")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVpnTunnelCertificateResponse{
			XMLName:       xml.Name{Local: "ModifyVpnTunnelCertificateResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			VpnConnection: ec2VpnConnectionItemFrom(connection),
		})
		return true
	case "ModifyVpnTunnelOptions":
		skipTunnelReplacement, hasSkipTunnelReplacement, ok := ec2OptionalBoolFromForm(r.Form, "SkipTunnelReplacement")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasSkipTunnelReplacement {
			skipTunnelReplacement = nil
		}

		connection, err := s.ec2.ModifyVpnTunnelOptions(
			strings.TrimSpace(r.Form.Get("VpnConnectionId")),
			strings.TrimSpace(r.Form.Get("VpnTunnelOutsideIpAddress")),
			ec2svc.ModifyVpnTunnelOptionsRequest{
				HasTunnelOptions:      hasEC2PrefixedField(r.Form, "TunnelOptions."),
				PreSharedKey:          parseEC2OptionalString(r.Form.Get("TunnelOptions.PreSharedKey")),
				TunnelInsideCidr:      parseEC2OptionalString(r.Form.Get("TunnelOptions.TunnelInsideCidr")),
				TunnelInsideIpv6Cidr:  parseEC2OptionalString(r.Form.Get("TunnelOptions.TunnelInsideIpv6Cidr")),
				PreSharedKeyStorage:   parseEC2OptionalString(r.Form.Get("PreSharedKeyStorage")),
				SkipTunnelReplacement: skipTunnelReplacement,
			},
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVpnTunnelOptionsResponse{
			XMLName:       xml.Name{Local: "ModifyVpnTunnelOptionsResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			VpnConnection: ec2VpnConnectionItemFrom(connection),
		})
		return true
	case "ReplaceVpnTunnel":
		ret, err := s.ec2.ReplaceVpnTunnel(
			strings.TrimSpace(r.Form.Get("VpnConnectionId")),
			strings.TrimSpace(r.Form.Get("VpnTunnelOutsideIpAddress")),
			parseEC2Bool(r.Form.Get("ApplyPendingMaintenance"), false),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ReplaceVpnTunnelResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}

func ec2ActiveVpnTunnelStatusItemFrom(in ec2svc.ActiveVpnTunnelStatus) ec2ActiveVpnTunnelStatusItem {
	return ec2ActiveVpnTunnelStatusItem{
		IkeVersion:                in.IkeVersion,
		Phase1DHGroup:             in.Phase1DHGroup,
		Phase1EncryptionAlgorithm: in.Phase1EncryptionAlgorithm,
		Phase1IntegrityAlgorithm:  in.Phase1IntegrityAlgorithm,
		Phase2DHGroup:             in.Phase2DHGroup,
		Phase2EncryptionAlgorithm: in.Phase2EncryptionAlgorithm,
		Phase2IntegrityAlgorithm:  in.Phase2IntegrityAlgorithm,
		ProvisioningStatus:        in.ProvisioningStatus,
		ProvisioningStatusReason:  in.ProvisioningStatusReason,
	}
}

func ec2VpnConnectionDeviceTypeItems(in []ec2svc.VpnConnectionDeviceType) []ec2VpnConnectionDeviceTypeItem {
	out := make([]ec2VpnConnectionDeviceTypeItem, 0, len(in))
	for _, deviceType := range in {
		out = append(out, ec2VpnConnectionDeviceTypeItem{
			Platform:                  deviceType.Platform,
			Software:                  deviceType.Software,
			Vendor:                    deviceType.Vendor,
			VpnConnectionDeviceTypeID: deviceType.ID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].VpnConnectionDeviceTypeID < out[j].VpnConnectionDeviceTypeID })
	return out
}

func ec2MaintenanceDetailsItemFrom(in ec2svc.MaintenanceDetails) ec2MaintenanceDetailsItem {
	item := ec2MaintenanceDetailsItem{PendingMaintenance: in.PendingMaintenance}
	if in.LastMaintenanceApplied != nil {
		item.LastMaintenanceApplied = in.LastMaintenanceApplied.UTC().Format(timeRFC3339UTC)
	}
	if in.MaintenanceAutoAppliedAfter != nil {
		item.MaintenanceAutoAppliedAfter = in.MaintenanceAutoAppliedAfter.UTC().Format(timeRFC3339UTC)
	}
	return item
}

type ec2EnableVgwRoutePropagationResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
}

type ec2DisableVgwRoutePropagationResponse struct {
	XMLName   xml.Name
	Xmlns     string `xml:"xmlns,attr"`
	RequestID string `xml:"requestId"`
}

type ec2GetActiveVpnTunnelStatusResponse struct {
	XMLName               xml.Name
	Xmlns                 string                       `xml:"xmlns,attr"`
	RequestID             string                       `xml:"requestId"`
	ActiveVpnTunnelStatus ec2ActiveVpnTunnelStatusItem `xml:"activeVpnTunnelStatus,omitempty"`
}

type ec2GetVpnConnectionDeviceSampleConfigurationResponse struct {
	XMLName                                xml.Name
	Xmlns                                  string `xml:"xmlns,attr"`
	RequestID                              string `xml:"requestId"`
	VpnConnectionDeviceSampleConfiguration string `xml:"vpnConnectionDeviceSampleConfiguration,omitempty"`
}

type ec2GetVpnConnectionDeviceTypesResponse struct {
	XMLName                    xml.Name
	Xmlns                      string                        `xml:"xmlns,attr"`
	RequestID                  string                        `xml:"requestId"`
	NextToken                  string                        `xml:"nextToken,omitempty"`
	VpnConnectionDeviceTypeSet ec2VpnConnectionDeviceTypeSet `xml:"vpnConnectionDeviceTypeSet"`
}

type ec2GetVpnTunnelReplacementStatusResponse struct {
	XMLName                   xml.Name                  `xml:"GetVpnTunnelReplacementStatusResponse"`
	Xmlns                     string                    `xml:"xmlns,attr"`
	RequestID                 string                    `xml:"requestId"`
	CustomerGatewayID         string                    `xml:"customerGatewayId,omitempty"`
	MaintenanceDetails        ec2MaintenanceDetailsItem `xml:"maintenanceDetails,omitempty"`
	TransitGatewayID          string                    `xml:"transitGatewayId,omitempty"`
	VpnConnectionID           string                    `xml:"vpnConnectionId,omitempty"`
	VpnGatewayID              string                    `xml:"vpnGatewayId,omitempty"`
	VpnTunnelOutsideIPAddress string                    `xml:"vpnTunnelOutsideIpAddress,omitempty"`
}

type ec2ModifyVpnTunnelCertificateResponse struct {
	XMLName       xml.Name
	Xmlns         string               `xml:"xmlns,attr"`
	RequestID     string               `xml:"requestId"`
	VpnConnection ec2VpnConnectionItem `xml:"vpnConnection"`
}

type ec2ModifyVpnTunnelOptionsResponse struct {
	XMLName       xml.Name
	Xmlns         string               `xml:"xmlns,attr"`
	RequestID     string               `xml:"requestId"`
	VpnConnection ec2VpnConnectionItem `xml:"vpnConnection"`
}

type ec2ActiveVpnTunnelStatusItem struct {
	IkeVersion                string `xml:"ikeVersion,omitempty"`
	Phase1DHGroup             int32  `xml:"phase1DHGroup,omitempty"`
	Phase1EncryptionAlgorithm string `xml:"phase1EncryptionAlgorithm,omitempty"`
	Phase1IntegrityAlgorithm  string `xml:"phase1IntegrityAlgorithm,omitempty"`
	Phase2DHGroup             int32  `xml:"phase2DHGroup,omitempty"`
	Phase2EncryptionAlgorithm string `xml:"phase2EncryptionAlgorithm,omitempty"`
	Phase2IntegrityAlgorithm  string `xml:"phase2IntegrityAlgorithm,omitempty"`
	ProvisioningStatus        string `xml:"provisioningStatus,omitempty"`
	ProvisioningStatusReason  string `xml:"provisioningStatusReason,omitempty"`
}

type ec2VpnConnectionDeviceTypeSet struct {
	Items []ec2VpnConnectionDeviceTypeItem `xml:"item"`
}

type ec2VpnConnectionDeviceTypeItem struct {
	Platform                  string `xml:"platform,omitempty"`
	Software                  string `xml:"software,omitempty"`
	Vendor                    string `xml:"vendor,omitempty"`
	VpnConnectionDeviceTypeID string `xml:"vpnConnectionDeviceTypeId,omitempty"`
}

type ec2MaintenanceDetailsItem struct {
	LastMaintenanceApplied      string `xml:"lastMaintenanceApplied,omitempty"`
	MaintenanceAutoAppliedAfter string `xml:"maintenanceAutoAppliedAfter,omitempty"`
	PendingMaintenance          string `xml:"pendingMaintenance,omitempty"`
}
