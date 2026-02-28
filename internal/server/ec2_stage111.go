package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage111Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DeleteFlowLogs":
		unsuccessful, err := s.ec2.DeleteFlowLogs(parseEC2MembersWithAliases(r.Form, "FlowLogId", "FlowLogIds"))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage111DeleteFlowLogsResponse{
			XMLName:      xml.Name{Local: "DeleteFlowLogsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Unsuccessful: ec2UnsuccessfulItemSet{Items: ec2UnsuccessfulItems(unsuccessful)},
		})
		return true
	case "DeleteFpgaImage":
		ok, err := s.ec2.DeleteFpgaImage(strings.TrimSpace(r.Form.Get("FpgaImageId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteFpgaImageResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ok,
		})
		return true
	case "DeleteInstanceConnectEndpoint":
		endpoint, err := s.ec2.DeleteInstanceConnectEndpoint(strings.TrimSpace(r.Form.Get("InstanceConnectEndpointId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage111DeleteInstanceConnectEndpointResponse{
			XMLName:                 xml.Name{Local: "DeleteInstanceConnectEndpointResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			InstanceConnectEndpoint: ec2Stage107InstanceConnectEndpointItemFrom(endpoint),
		})
		return true
	case "DeleteInstanceEventWindow":
		forceDelete, hasForceDelete, ok := ec2OptionalBoolFromForm(r.Form, "ForceDelete")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasForceDelete {
			forceDelete = nil
		}
		stateChange, err := s.ec2.DeleteInstanceEventWindow(strings.TrimSpace(r.Form.Get("InstanceEventWindowId")), forceDelete)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage111DeleteInstanceEventWindowResponse{
			XMLName:                  xml.Name{Local: "DeleteInstanceEventWindowResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			InstanceEventWindowState: ec2Stage111InstanceEventWindowStateItemFrom(stateChange),
		})
		return true
	case "DeleteIpam":
		cascade, hasCascade, ok := ec2OptionalBoolFromForm(r.Form, "Cascade")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasCascade {
			cascade = nil
		}
		ipam, err := s.ec2.DeleteIpam(strings.TrimSpace(r.Form.Get("IpamId")), cascade)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage111DeleteIpamResponse{
			XMLName:   xml.Name{Local: "DeleteIpamResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Ipam:      ec2Stage107IpamItemFrom(ipam),
		})
		return true
	case "DeleteIpamExternalResourceVerificationToken":
		token, err := s.ec2.DeleteIpamExternalResourceVerificationToken(
			strings.TrimSpace(r.Form.Get("IpamExternalResourceVerificationTokenId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage111DeleteIpamExternalResourceVerificationTokenResponse{
			XMLName:                               xml.Name{Local: "DeleteIpamExternalResourceVerificationTokenResponse"},
			Xmlns:                                 ec2Namespace,
			RequestID:                             "stackyard-request",
			IpamExternalResourceVerificationToken: ec2Stage111IpamExternalResourceVerificationTokenItemFrom(token),
		})
		return true
	case "DeleteIpamPool":
		cascade, hasCascade, ok := ec2OptionalBoolFromForm(r.Form, "Cascade")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasCascade {
			cascade = nil
		}
		pool, err := s.ec2.DeleteIpamPool(strings.TrimSpace(r.Form.Get("IpamPoolId")), cascade)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage111DeleteIpamPoolResponse{
			XMLName:   xml.Name{Local: "DeleteIpamPoolResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			IpamPool:  ec2Stage108IpamPoolItemFrom(pool),
		})
		return true
	case "DeleteIpamResourceDiscovery":
		discovery, err := s.ec2.DeleteIpamResourceDiscovery(strings.TrimSpace(r.Form.Get("IpamResourceDiscoveryId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage111DeleteIpamResourceDiscoveryResponse{
			XMLName:               xml.Name{Local: "DeleteIpamResourceDiscoveryResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			IpamResourceDiscovery: ec2Stage108IpamResourceDiscoveryItemFrom(discovery),
		})
		return true
	case "DeleteIpamScope":
		scope, err := s.ec2.DeleteIpamScope(strings.TrimSpace(r.Form.Get("IpamScopeId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage111DeleteIpamScopeResponse{
			XMLName:   xml.Name{Local: "DeleteIpamScopeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			IpamScope: ec2Stage108IpamScopeItemFrom(scope),
		})
		return true
	case "DeleteLaunchTemplate":
		launchTemplate, err := s.ec2.DeleteLaunchTemplate(
			strings.TrimSpace(r.Form.Get("LaunchTemplateId")),
			strings.TrimSpace(r.Form.Get("LaunchTemplateName")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage111DeleteLaunchTemplateResponse{
			XMLName:        xml.Name{Local: "DeleteLaunchTemplateResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			LaunchTemplate: ec2Stage108LaunchTemplateItemFrom(launchTemplate),
		})
		return true
	default:
		return false
	}
}

func ec2Stage111InstanceEventWindowStateItemFrom(in ec2svc.InstanceEventWindowStateChange) ec2Stage111InstanceEventWindowStateItem {
	return ec2Stage111InstanceEventWindowStateItem{
		InstanceEventWindowID: in.InstanceEventWindowID,
		State:                 in.State,
	}
}

func ec2Stage111IpamExternalResourceVerificationTokenItemFrom(in ec2svc.IpamExternalResourceVerificationToken) ec2Stage107IpamExternalResourceVerificationTokenItem {
	return ec2Stage107IpamExternalResourceVerificationTokenItem{
		IpamARN:                                  in.IpamARN,
		IpamExternalResourceVerificationTokenARN: in.IpamExternalResourceVerificationTokenARN,
		IpamExternalResourceVerificationTokenID:  in.IpamExternalResourceVerificationTokenID,
		IpamID:                                   in.IpamID,
		IpamRegion:                               in.IpamRegion,
		NotAfter:                                 in.NotAfter.UTC().Format(time.RFC3339),
		State:                                    in.State,
		Status:                                   in.Status,
		TagSet:                                   ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TokenName:                                in.TokenName,
	}
}

type ec2Stage111DeleteFlowLogsResponse struct {
	XMLName      xml.Name               `xml:"DeleteFlowLogsResponse"`
	Xmlns        string                 `xml:"xmlns,attr"`
	RequestID    string                 `xml:"requestId"`
	Unsuccessful ec2UnsuccessfulItemSet `xml:"unsuccessful"`
}

type ec2Stage111DeleteInstanceConnectEndpointResponse struct {
	XMLName                 xml.Name                               `xml:"DeleteInstanceConnectEndpointResponse"`
	Xmlns                   string                                 `xml:"xmlns,attr"`
	RequestID               string                                 `xml:"requestId"`
	InstanceConnectEndpoint ec2Stage107InstanceConnectEndpointItem `xml:"instanceConnectEndpoint"`
}

type ec2Stage111DeleteInstanceEventWindowResponse struct {
	XMLName                  xml.Name                                `xml:"DeleteInstanceEventWindowResponse"`
	Xmlns                    string                                  `xml:"xmlns,attr"`
	RequestID                string                                  `xml:"requestId"`
	InstanceEventWindowState ec2Stage111InstanceEventWindowStateItem `xml:"instanceEventWindowState"`
}

type ec2Stage111InstanceEventWindowStateItem struct {
	InstanceEventWindowID string `xml:"instanceEventWindowId,omitempty"`
	State                 string `xml:"state,omitempty"`
}

type ec2Stage111DeleteIpamResponse struct {
	XMLName   xml.Name            `xml:"DeleteIpamResponse"`
	Xmlns     string              `xml:"xmlns,attr"`
	RequestID string              `xml:"requestId"`
	Ipam      ec2Stage107IpamItem `xml:"ipam"`
}

type ec2Stage111DeleteIpamExternalResourceVerificationTokenResponse struct {
	XMLName                               xml.Name                                             `xml:"DeleteIpamExternalResourceVerificationTokenResponse"`
	Xmlns                                 string                                               `xml:"xmlns,attr"`
	RequestID                             string                                               `xml:"requestId"`
	IpamExternalResourceVerificationToken ec2Stage107IpamExternalResourceVerificationTokenItem `xml:"ipamExternalResourceVerificationToken"`
}

type ec2Stage111DeleteIpamPoolResponse struct {
	XMLName   xml.Name                `xml:"DeleteIpamPoolResponse"`
	Xmlns     string                  `xml:"xmlns,attr"`
	RequestID string                  `xml:"requestId"`
	IpamPool  ec2Stage108IpamPoolItem `xml:"ipamPool"`
}

type ec2Stage111DeleteIpamResourceDiscoveryResponse struct {
	XMLName               xml.Name                             `xml:"DeleteIpamResourceDiscoveryResponse"`
	Xmlns                 string                               `xml:"xmlns,attr"`
	RequestID             string                               `xml:"requestId"`
	IpamResourceDiscovery ec2Stage108IpamResourceDiscoveryItem `xml:"ipamResourceDiscovery"`
}

type ec2Stage111DeleteIpamScopeResponse struct {
	XMLName   xml.Name                 `xml:"DeleteIpamScopeResponse"`
	Xmlns     string                   `xml:"xmlns,attr"`
	RequestID string                   `xml:"requestId"`
	IpamScope ec2Stage108IpamScopeItem `xml:"ipamScope"`
}

type ec2Stage111DeleteLaunchTemplateResponse struct {
	XMLName        xml.Name                      `xml:"DeleteLaunchTemplateResponse"`
	Xmlns          string                        `xml:"xmlns,attr"`
	RequestID      string                        `xml:"requestId"`
	LaunchTemplate ec2Stage108LaunchTemplateItem `xml:"launchTemplate"`
}
