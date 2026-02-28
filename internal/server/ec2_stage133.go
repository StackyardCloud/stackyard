package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage133Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "RestoreSnapshotFromRecycleBin":
		snapshot, err := s.ec2.RestoreSnapshotFromRecycleBin(strings.TrimSpace(r.Form.Get("SnapshotId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		encrypted := false
		startTime := ""
		if !snapshot.StartTime.IsZero() {
			startTime = snapshot.StartTime.UTC().Format(timeRFC3339UTC)
		}
		respondEC2XML(w, ec2Stage133RestoreSnapshotFromRecycleBinResponse{
			XMLName:     xml.Name{Local: "RestoreSnapshotFromRecycleBinResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			Description: snapshot.Description,
			Encrypted:   &encrypted,
			OwnerID:     ec2svc.DefaultAccountID,
			Progress:    snapshot.Progress,
			SnapshotID:  snapshot.ID,
			StartTime:   startTime,
			State:       snapshot.State,
			VolumeID:    snapshot.VolumeID,
			VolumeSize:  &snapshot.VolumeSize,
		})
		return true
	case "RestoreSnapshotTier":
		permanentRestore, ok := parseEC2OptionalBoolValue(r.Form.Get("PermanentRestore"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		temporaryRestoreDays, ok := parseEC2OptionalInt32(r.Form.Get("TemporaryRestoreDays"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		snapshotID, restoreStartTime, restoreDuration, isPermanentRestore, err := s.ec2.RestoreSnapshotTier(
			strings.TrimSpace(r.Form.Get("SnapshotId")),
			permanentRestore,
			temporaryRestoreDays,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		restoreStartTimeValue := ""
		if !restoreStartTime.IsZero() {
			restoreStartTimeValue = restoreStartTime.UTC().Format(timeRFC3339UTC)
		}
		respondEC2XML(w, ec2Stage133RestoreSnapshotTierResponse{
			XMLName:            xml.Name{Local: "RestoreSnapshotTierResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			IsPermanentRestore: &isPermanentRestore,
			RestoreDuration:    &restoreDuration,
			RestoreStartTime:   restoreStartTimeValue,
			SnapshotID:         snapshotID,
		})
		return true
	case "RunScheduledInstances":
		if !hasEC2PrefixedField(r.Form, "LaunchSpecification.") && !hasEC2Field(r.Form, "LaunchSpecification") {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		instanceCount, ok := parseEC2OptionalInt32(r.Form.Get("InstanceCount"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		instanceIDs, err := s.ec2.RunScheduledInstances(
			strings.TrimSpace(r.Form.Get("ScheduledInstanceId")),
			strings.TrimSpace(r.Form.Get("LaunchSpecification.ImageId")),
			strings.TrimSpace(r.Form.Get("LaunchSpecification.InstanceType")),
			instanceCount,
			parseEC2OptionalString(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage133RunScheduledInstancesResponse{
			XMLName:       xml.Name{Local: "RunScheduledInstancesResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			InstanceIDSet: ec2StringSet{Items: append([]string(nil), instanceIDs...)},
		})
		return true
	case "SearchLocalGatewayRoutes":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		routes, nextToken, err := s.ec2.SearchLocalGatewayRoutes(
			strings.TrimSpace(r.Form.Get("LocalGatewayRouteTableId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage133SearchLocalGatewayRoutesResponse{
			XMLName:   xml.Name{Local: "SearchLocalGatewayRoutesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			NextToken: nextToken,
			Routes: ec2Stage133LocalGatewayRouteSet{
				Items: ec2Stage133LocalGatewayRouteItemsFrom(routes),
			},
		})
		return true
	case "SendDiagnosticInterrupt":
		err := s.ec2.SendDiagnosticInterrupt(strings.TrimSpace(r.Form.Get("InstanceId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage133SendDiagnosticInterruptResponse{
			XMLName:   xml.Name{Local: "SendDiagnosticInterruptResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	case "StartNetworkInsightsAccessScopeAnalysis":
		analysis, err := s.ec2.StartNetworkInsightsAccessScopeAnalysis(
			strings.TrimSpace(r.Form.Get("NetworkInsightsAccessScopeId")),
			strings.TrimSpace(r.Form.Get("ClientToken")),
			parseEC2TagSpecificationsForResource(r.Form, "network-insights-access-scope-analysis"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		analysisItem := ec2Stage120NetworkInsightsAccessScopeAnalysisItem{}
		analysisItems := ec2Stage120NetworkInsightsAccessScopeAnalysisItemsFrom([]ec2svc.NetworkInsightsAccessScopeAnalysis{analysis})
		if len(analysisItems) > 0 {
			analysisItem = analysisItems[0]
		}
		respondEC2XML(w, ec2Stage133StartNetworkInsightsAccessScopeAnalysisResponse{
			XMLName:                            xml.Name{Local: "StartNetworkInsightsAccessScopeAnalysisResponse"},
			Xmlns:                              ec2Namespace,
			RequestID:                          "stackyard-request",
			NetworkInsightsAccessScopeAnalysis: analysisItem,
		})
		return true
	case "StartNetworkInsightsAnalysis":
		analysis, err := s.ec2.StartNetworkInsightsAnalysis(
			strings.TrimSpace(r.Form.Get("NetworkInsightsPathId")),
			strings.TrimSpace(r.Form.Get("ClientToken")),
			parseEC2TagSpecificationsForResource(r.Form, "network-insights-analysis"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		analysisItem := ec2Stage120NetworkInsightsAnalysisItem{}
		analysisItems := ec2Stage120NetworkInsightsAnalysisItemsFrom([]ec2svc.NetworkInsightsAnalysis{analysis})
		if len(analysisItems) > 0 {
			analysisItem = analysisItems[0]
		}
		respondEC2XML(w, ec2Stage133StartNetworkInsightsAnalysisResponse{
			XMLName:                 xml.Name{Local: "StartNetworkInsightsAnalysisResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			NetworkInsightsAnalysis: analysisItem,
		})
		return true
	case "UnlockSnapshot":
		snapshotID, err := s.ec2.UnlockSnapshot(strings.TrimSpace(r.Form.Get("SnapshotId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage133UnlockSnapshotResponse{
			XMLName:    xml.Name{Local: "UnlockSnapshotResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			SnapshotID: snapshotID,
		})
		return true
	case "WithdrawByoipCidr":
		byoipCidr, err := s.ec2.WithdrawByoipCidr(strings.TrimSpace(r.Form.Get("Cidr")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage133WithdrawByoipCidrResponse{
			XMLName:   xml.Name{Local: "WithdrawByoipCidrResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ByoipCidr: ec2ByoipCidrItemFrom(byoipCidr),
		})
		return true
	default:
		return false
	}
}

func ec2Stage133LocalGatewayRouteItemsFrom(in []ec2svc.LocalGatewayRoute) []ec2Stage108LocalGatewayRouteItem {
	out := make([]ec2Stage108LocalGatewayRouteItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage108LocalGatewayRouteItemFrom(item))
	}
	return out
}

type ec2Stage133RestoreSnapshotFromRecycleBinResponse struct {
	XMLName     xml.Name `xml:"RestoreSnapshotFromRecycleBinResponse"`
	Xmlns       string   `xml:"xmlns,attr"`
	RequestID   string   `xml:"requestId"`
	Description string   `xml:"description,omitempty"`
	Encrypted   *bool    `xml:"encrypted,omitempty"`
	OutpostARN  string   `xml:"outpostArn,omitempty"`
	OwnerID     string   `xml:"ownerId,omitempty"`
	Progress    string   `xml:"progress,omitempty"`
	SnapshotID  string   `xml:"snapshotId,omitempty"`
	SseType     string   `xml:"sseType,omitempty"`
	StartTime   string   `xml:"startTime,omitempty"`
	State       string   `xml:"status,omitempty"`
	VolumeID    string   `xml:"volumeId,omitempty"`
	VolumeSize  *int32   `xml:"volumeSize,omitempty"`
}

type ec2Stage133RestoreSnapshotTierResponse struct {
	XMLName            xml.Name `xml:"RestoreSnapshotTierResponse"`
	Xmlns              string   `xml:"xmlns,attr"`
	RequestID          string   `xml:"requestId"`
	IsPermanentRestore *bool    `xml:"isPermanentRestore,omitempty"`
	RestoreDuration    *int32   `xml:"restoreDuration,omitempty"`
	RestoreStartTime   string   `xml:"restoreStartTime,omitempty"`
	SnapshotID         string   `xml:"snapshotId,omitempty"`
}

type ec2Stage133RunScheduledInstancesResponse struct {
	XMLName       xml.Name     `xml:"RunScheduledInstancesResponse"`
	Xmlns         string       `xml:"xmlns,attr"`
	RequestID     string       `xml:"requestId"`
	InstanceIDSet ec2StringSet `xml:"instanceIdSet"`
}

type ec2Stage133SearchLocalGatewayRoutesResponse struct {
	XMLName   xml.Name                        `xml:"SearchLocalGatewayRoutesResponse"`
	Xmlns     string                          `xml:"xmlns,attr"`
	RequestID string                          `xml:"requestId"`
	NextToken *string                         `xml:"nextToken,omitempty"`
	Routes    ec2Stage133LocalGatewayRouteSet `xml:"routeSet"`
}

type ec2Stage133LocalGatewayRouteSet struct {
	Items []ec2Stage108LocalGatewayRouteItem `xml:"item"`
}

type ec2Stage133SendDiagnosticInterruptResponse struct {
	XMLName   xml.Name `xml:"SendDiagnosticInterruptResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
}

type ec2Stage133StartNetworkInsightsAccessScopeAnalysisResponse struct {
	XMLName                            xml.Name                                          `xml:"StartNetworkInsightsAccessScopeAnalysisResponse"`
	Xmlns                              string                                            `xml:"xmlns,attr"`
	RequestID                          string                                            `xml:"requestId"`
	NetworkInsightsAccessScopeAnalysis ec2Stage120NetworkInsightsAccessScopeAnalysisItem `xml:"networkInsightsAccessScopeAnalysis"`
}

type ec2Stage133StartNetworkInsightsAnalysisResponse struct {
	XMLName                 xml.Name                               `xml:"StartNetworkInsightsAnalysisResponse"`
	Xmlns                   string                                 `xml:"xmlns,attr"`
	RequestID               string                                 `xml:"requestId"`
	NetworkInsightsAnalysis ec2Stage120NetworkInsightsAnalysisItem `xml:"networkInsightsAnalysis"`
}

type ec2Stage133UnlockSnapshotResponse struct {
	XMLName    xml.Name `xml:"UnlockSnapshotResponse"`
	Xmlns      string   `xml:"xmlns,attr"`
	RequestID  string   `xml:"requestId"`
	SnapshotID string   `xml:"snapshotId,omitempty"`
}

type ec2Stage133WithdrawByoipCidrResponse struct {
	XMLName   xml.Name         `xml:"WithdrawByoipCidrResponse"`
	Xmlns     string           `xml:"xmlns,attr"`
	RequestID string           `xml:"requestId"`
	ByoipCidr ec2ByoipCidrItem `xml:"byoipCidr"`
}
