package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	websecurityscannerpb "cloud.google.com/go/websecurityscanner/apiv1/websecurityscannerpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpWebSecurityScannerCreateScanConfigMethod     = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/CreateScanConfig"
	gcpWebSecurityScannerDeleteScanConfigMethod     = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/DeleteScanConfig"
	gcpWebSecurityScannerGetScanConfigMethod        = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/GetScanConfig"
	gcpWebSecurityScannerListScanConfigsMethod      = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/ListScanConfigs"
	gcpWebSecurityScannerUpdateScanConfigMethod     = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/UpdateScanConfig"
	gcpWebSecurityScannerStartScanRunMethod         = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/StartScanRun"
	gcpWebSecurityScannerGetScanRunMethod           = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/GetScanRun"
	gcpWebSecurityScannerListScanRunsMethod         = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/ListScanRuns"
	gcpWebSecurityScannerStopScanRunMethod          = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/StopScanRun"
	gcpWebSecurityScannerListCrawledURLsMethod      = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/ListCrawledUrls"
	gcpWebSecurityScannerGetFindingMethod           = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/GetFinding"
	gcpWebSecurityScannerListFindingsMethod         = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/ListFindings"
	gcpWebSecurityScannerListFindingTypeStatsMethod = "/google.cloud.websecurityscanner.v1.WebSecurityScanner/ListFindingTypeStats"
)

func gcpStage4GRPCWebSecurityScanner(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpWebSecurityScannerCreateScanConfigMethod:
		return gcpStage4GRPCWebSecurityScannerCreateScanConfig(grpcReqBody)
	case gcpWebSecurityScannerDeleteScanConfigMethod:
		return gcpStage4GRPCWebSecurityScannerDeleteScanConfig(grpcReqBody)
	case gcpWebSecurityScannerGetScanConfigMethod:
		return gcpStage4GRPCWebSecurityScannerGetScanConfig(grpcReqBody)
	case gcpWebSecurityScannerListScanConfigsMethod:
		return gcpStage4GRPCWebSecurityScannerListScanConfigs(grpcReqBody)
	case gcpWebSecurityScannerUpdateScanConfigMethod:
		return gcpStage4GRPCWebSecurityScannerUpdateScanConfig(grpcReqBody)
	case gcpWebSecurityScannerStartScanRunMethod:
		return gcpStage4GRPCWebSecurityScannerStartScanRun(grpcReqBody)
	case gcpWebSecurityScannerGetScanRunMethod:
		return gcpStage4GRPCWebSecurityScannerGetScanRun(grpcReqBody)
	case gcpWebSecurityScannerListScanRunsMethod:
		return gcpStage4GRPCWebSecurityScannerListScanRuns(grpcReqBody)
	case gcpWebSecurityScannerStopScanRunMethod:
		return gcpStage4GRPCWebSecurityScannerStopScanRun(grpcReqBody)
	case gcpWebSecurityScannerListCrawledURLsMethod:
		return gcpStage4GRPCWebSecurityScannerListCrawledURLs(grpcReqBody)
	case gcpWebSecurityScannerGetFindingMethod:
		return gcpStage4GRPCWebSecurityScannerGetFinding(grpcReqBody)
	case gcpWebSecurityScannerListFindingsMethod:
		return gcpStage4GRPCWebSecurityScannerListFindings(grpcReqBody)
	case gcpWebSecurityScannerListFindingTypeStatsMethod:
		return gcpStage4GRPCWebSecurityScannerListFindingTypeStats(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCWebSecurityScannerCreateScanConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.CreateScanConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPWebSecurityScannerProjectParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	scanConfig := req.GetScanConfig()
	if scanConfig == nil {
		return grpcInvalidArgument("scan_config-required")
	}

	scanConfigID, reason, valid := gcpStage4ValidateWebSecurityScannerScanConfig(scanConfig, false, project)
	if !valid {
		return grpcInvalidArgument(reason)
	}
	if scanConfigID == "" {
		scanConfigID = "scan-config-1"
	}
	if isGCPWebSecurityScannerMissingID(scanConfigID) {
		return grpcAlreadyExists("scan_config-already-exists")
	}

	resp := gcpStage4WebSecurityScannerScanConfig(project, scanConfigID)
	gcpStage4ApplyWebSecurityScannerScanConfigOverrides(resp, scanConfig)
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCWebSecurityScannerDeleteScanConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.DeleteScanConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, scanConfigID, ok := parseGCPWebSecurityScannerScanConfigName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPWebSecurityScannerMissingID(scanConfigID) {
		return grpcNotFound("scan_config-not-found")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCWebSecurityScannerGetScanConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.GetScanConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, scanConfigID, ok := parseGCPWebSecurityScannerScanConfigName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPWebSecurityScannerMissingID(scanConfigID) {
		return grpcNotFound("scan_config-not-found")
	}
	return grpcProtoSuccess(gcpStage4WebSecurityScannerScanConfig(project, scanConfigID))
}

func gcpStage4GRPCWebSecurityScannerListScanConfigs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.ListScanConfigsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPWebSecurityScannerProjectParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}

	items := []*websecurityscannerpb.ScanConfig{
		gcpStage4WebSecurityScannerScanConfig(project, "scan-config-1"),
		gcpStage4WebSecurityScannerScanConfig(project, "scan-config-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}

	return grpcProtoSuccess(&websecurityscannerpb.ListScanConfigsResponse{
		ScanConfigs:   items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCWebSecurityScannerUpdateScanConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.UpdateScanConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scanConfig := req.GetScanConfig()
	if scanConfig == nil {
		return grpcInvalidArgument("scan_config-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}

	scanConfigID, reason, valid := gcpStage4ValidateWebSecurityScannerScanConfig(scanConfig, true, "")
	if !valid {
		return grpcInvalidArgument(reason)
	}
	project, _, ok := parseGCPWebSecurityScannerScanConfigName(strings.TrimSpace(scanConfig.GetName()))
	if !ok {
		return grpcInvalidArgument("scan_config.name-required")
	}
	if isGCPWebSecurityScannerMissingID(scanConfigID) {
		return grpcNotFound("scan_config-not-found")
	}

	resp := gcpStage4WebSecurityScannerScanConfig(project, scanConfigID)
	gcpStage4ApplyWebSecurityScannerScanConfigOverrides(resp, scanConfig)
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCWebSecurityScannerStartScanRun(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.StartScanRunRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, scanConfigID, ok := parseGCPWebSecurityScannerScanConfigName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPWebSecurityScannerMissingID(scanConfigID) {
		return grpcNotFound("scan_config-not-found")
	}
	return grpcProtoSuccess(gcpStage4WebSecurityScannerScanRun(project, scanConfigID, "scan-run-1", websecurityscannerpb.ScanRun_SCANNING, websecurityscannerpb.ScanRun_RESULT_STATE_UNSPECIFIED))
}

func gcpStage4GRPCWebSecurityScannerGetScanRun(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.GetScanRunRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, scanConfigID, scanRunID, ok := parseGCPWebSecurityScannerScanRunName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
		return grpcNotFound("scan_run-not-found")
	}
	executionState, resultState := gcpStage4WebSecurityScannerRunStates(scanRunID, false)
	return grpcProtoSuccess(gcpStage4WebSecurityScannerScanRun(project, scanConfigID, scanRunID, executionState, resultState))
}

func gcpStage4GRPCWebSecurityScannerListScanRuns(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.ListScanRunsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, scanConfigID, ok := parseGCPWebSecurityScannerScanRunParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if isGCPWebSecurityScannerMissingID(scanConfigID) {
		return grpcNotFound("scan_config-not-found")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}

	items := []*websecurityscannerpb.ScanRun{
		gcpStage4WebSecurityScannerScanRun(project, scanConfigID, "scan-run-1", websecurityscannerpb.ScanRun_SCANNING, websecurityscannerpb.ScanRun_RESULT_STATE_UNSPECIFIED),
		gcpStage4WebSecurityScannerScanRun(project, scanConfigID, "scan-run-finished-2", websecurityscannerpb.ScanRun_FINISHED, websecurityscannerpb.ScanRun_SUCCESS),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}

	return grpcProtoSuccess(&websecurityscannerpb.ListScanRunsResponse{
		ScanRuns:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCWebSecurityScannerStopScanRun(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.StopScanRunRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, scanConfigID, scanRunID, ok := parseGCPWebSecurityScannerScanRunName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
		return grpcNotFound("scan_run-not-found")
	}
	return grpcProtoSuccess(gcpStage4WebSecurityScannerScanRun(project, scanConfigID, scanRunID, websecurityscannerpb.ScanRun_FINISHED, websecurityscannerpb.ScanRun_SUCCESS))
}

func gcpStage4GRPCWebSecurityScannerListCrawledURLs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.ListCrawledUrlsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, scanConfigID, scanRunID, ok := parseGCPWebSecurityScannerScanRunName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
		return grpcNotFound("scan_run-not-found")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}

	scanRunName := gcpWebSecurityScannerScanRunName(project, scanConfigID, scanRunID)
	items := []*websecurityscannerpb.CrawledUrl{
		gcpStage4WebSecurityScannerCrawledURL(scanRunName, 1),
		gcpStage4WebSecurityScannerCrawledURL(scanRunName, 2),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}

	return grpcProtoSuccess(&websecurityscannerpb.ListCrawledUrlsResponse{
		CrawledUrls:   items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCWebSecurityScannerGetFinding(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.GetFindingRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, scanConfigID, scanRunID, findingID, ok := parseGCPWebSecurityScannerFindingName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) || isGCPWebSecurityScannerMissingID(findingID) {
		return grpcNotFound("finding-not-found")
	}
	return grpcProtoSuccess(gcpStage4WebSecurityScannerFinding(project, scanConfigID, scanRunID, findingID, "MIXED_CONTENT"))
}

func gcpStage4GRPCWebSecurityScannerListFindings(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.ListFindingsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, scanConfigID, scanRunID, ok := parseGCPWebSecurityScannerScanRunName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
		return grpcNotFound("scan_run-not-found")
	}
	filter, valid := gcpStage4ValidateWebSecurityScannerFilter(req.GetFilter())
	if !valid {
		return grpcInvalidArgument("filter-invalid")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}

	items := []*websecurityscannerpb.Finding{
		gcpStage4WebSecurityScannerFinding(project, scanConfigID, scanRunID, "finding-1", "MIXED_CONTENT"),
		gcpStage4WebSecurityScannerFinding(project, scanConfigID, scanRunID, "finding-2", "OUTDATED_LIBRARY"),
	}
	if filter != "" {
		filtered := make([]*websecurityscannerpb.Finding, 0, len(items))
		for _, item := range items {
			if strings.EqualFold(strings.TrimSpace(item.GetFindingType()), filter) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}

	return grpcProtoSuccess(&websecurityscannerpb.ListFindingsResponse{
		Findings:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCWebSecurityScannerListFindingTypeStats(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &websecurityscannerpb.ListFindingTypeStatsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, scanConfigID, scanRunID, ok := parseGCPWebSecurityScannerScanRunName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
		return grpcNotFound("scan_run-not-found")
	}
	return grpcProtoSuccess(&websecurityscannerpb.ListFindingTypeStatsResponse{
		FindingTypeStats: []*websecurityscannerpb.FindingTypeStats{
			gcpStage4WebSecurityScannerFindingTypeStat("MIXED_CONTENT", 1),
			gcpStage4WebSecurityScannerFindingTypeStat("OUTDATED_LIBRARY", 1),
		},
	})
}

func gcpStage4ValidateWebSecurityScannerScanConfig(scanConfig *websecurityscannerpb.ScanConfig, requireName bool, expectedProject string) (scanConfigID, reason string, ok bool) {
	name := strings.TrimSpace(scanConfig.GetName())
	if requireName && name == "" {
		return "", "scan_config.name-required", false
	}

	if name != "" {
		project, parsedID, parsed := parseGCPWebSecurityScannerScanConfigName(name)
		if !parsed {
			return "", "scan_config.name-invalid", false
		}
		if expectedProject != "" && project != expectedProject {
			return "", "scan_config.name-mismatch", false
		}
		scanConfigID = parsedID
	}

	if strings.TrimSpace(scanConfig.GetDisplayName()) == "" {
		return "", "scan_config.display_name-required", false
	}
	if len(scanConfig.GetStartingUrls()) == 0 {
		return "", "scan_config.starting_urls-required", false
	}
	for _, rawURL := range scanConfig.GetStartingUrls() {
		if !isGCPWebSecurityScannerURI(rawURL) {
			return "", "scan_config.starting_urls-invalid", false
		}
	}
	maxQPS := scanConfig.GetMaxQps()
	if maxQPS != 0 && (maxQPS < 5 || maxQPS > 20) {
		return "", "scan_config.max_qps-invalid", false
	}
	return scanConfigID, "", true
}

func gcpStage4ValidateWebSecurityScannerFilter(rawFilter string) (string, bool) {
	filter := strings.TrimSpace(rawFilter)
	if filter == "" {
		return "", true
	}
	matches := gcpWebSecurityScannerFilterRe.FindStringSubmatch(filter)
	if len(matches) != 2 {
		return "", false
	}
	return strings.ToUpper(strings.TrimSpace(matches[1])), true
}

func gcpStage4WebSecurityScannerScanConfig(project, scanConfigID string) *websecurityscannerpb.ScanConfig {
	return &websecurityscannerpb.ScanConfig{
		Name:         gcpWebSecurityScannerScanConfigName(project, scanConfigID),
		DisplayName:  "Stackyard Scan Config " + scanConfigID,
		MaxQps:       15,
		StartingUrls: []string{fmt.Sprintf("https://%s.stackyard.test", scanConfigID)},
		UserAgent:    websecurityscannerpb.ScanConfig_CHROME_LINUX,
		RiskLevel:    websecurityscannerpb.ScanConfig_NORMAL,
		Schedule: &websecurityscannerpb.ScanConfig_Schedule{
			ScheduleTime:         timestamppb.New(gcpStage4ReferenceTime),
			IntervalDurationDays: 1,
		},
		ManagedScan:            false,
		StaticIpScan:           false,
		IgnoreHttpStatusErrors: false,
	}
}

func gcpStage4ApplyWebSecurityScannerScanConfigOverrides(out, in *websecurityscannerpb.ScanConfig) {
	if strings.TrimSpace(in.GetDisplayName()) != "" {
		out.DisplayName = in.GetDisplayName()
	}
	if len(in.GetStartingUrls()) > 0 {
		out.StartingUrls = append([]string(nil), in.GetStartingUrls()...)
	}
	if in.GetMaxQps() != 0 {
		out.MaxQps = in.GetMaxQps()
	}
	if in.GetAuthentication() != nil {
		out.Authentication = in.GetAuthentication()
	}
	if in.GetUserAgent() != websecurityscannerpb.ScanConfig_USER_AGENT_UNSPECIFIED {
		out.UserAgent = in.GetUserAgent()
	}
	if len(in.GetBlacklistPatterns()) > 0 {
		out.BlacklistPatterns = append([]string(nil), in.GetBlacklistPatterns()...)
	}
	if in.GetSchedule() != nil {
		out.Schedule = in.GetSchedule()
	}
	if in.GetRiskLevel() != websecurityscannerpb.ScanConfig_RISK_LEVEL_UNSPECIFIED {
		out.RiskLevel = in.GetRiskLevel()
	}
	out.ManagedScan = in.GetManagedScan()
	out.StaticIpScan = in.GetStaticIpScan()
	out.IgnoreHttpStatusErrors = in.GetIgnoreHttpStatusErrors()
}

func gcpStage4WebSecurityScannerScanRun(project, scanConfigID, scanRunID string, executionState websecurityscannerpb.ScanRun_ExecutionState, resultState websecurityscannerpb.ScanRun_ResultState) *websecurityscannerpb.ScanRun {
	return &websecurityscannerpb.ScanRun{
		Name:               gcpWebSecurityScannerScanRunName(project, scanConfigID, scanRunID),
		ExecutionState:     executionState,
		ResultState:        resultState,
		StartTime:          timestamppb.New(gcpStage4ReferenceTime.Add(5 * time.Minute)),
		EndTime:            timestamppb.New(gcpStage4ReferenceTime.Add(8 * time.Minute)),
		UrlsCrawledCount:   12,
		UrlsTestedCount:    36,
		HasVulnerabilities: true,
		ProgressPercent:    gcpStage4WebSecurityScannerProgress(executionState),
	}
}

func gcpStage4WebSecurityScannerRunStates(scanRunID string, stopped bool) (websecurityscannerpb.ScanRun_ExecutionState, websecurityscannerpb.ScanRun_ResultState) {
	executionStateRaw, resultStateRaw := gcpWebSecurityScannerScanRunStates(scanRunID, stopped)
	return gcpStage4WebSecurityScannerExecutionState(executionStateRaw), gcpStage4WebSecurityScannerResultState(resultStateRaw)
}

func gcpStage4WebSecurityScannerExecutionState(value string) websecurityscannerpb.ScanRun_ExecutionState {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "FINISHED":
		return websecurityscannerpb.ScanRun_FINISHED
	case "QUEUED":
		return websecurityscannerpb.ScanRun_QUEUED
	default:
		return websecurityscannerpb.ScanRun_SCANNING
	}
}

func gcpStage4WebSecurityScannerResultState(value string) websecurityscannerpb.ScanRun_ResultState {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "SUCCESS":
		return websecurityscannerpb.ScanRun_SUCCESS
	case "ERROR":
		return websecurityscannerpb.ScanRun_ERROR
	case "KILLED":
		return websecurityscannerpb.ScanRun_KILLED
	default:
		return websecurityscannerpb.ScanRun_RESULT_STATE_UNSPECIFIED
	}
}

func gcpStage4WebSecurityScannerProgress(executionState websecurityscannerpb.ScanRun_ExecutionState) int32 {
	switch executionState {
	case websecurityscannerpb.ScanRun_FINISHED:
		return 100
	case websecurityscannerpb.ScanRun_QUEUED:
		return 0
	default:
		return 45
	}
}

func gcpStage4WebSecurityScannerCrawledURL(scanRunName string, index int) *websecurityscannerpb.CrawledUrl {
	return &websecurityscannerpb.CrawledUrl{
		HttpMethod: "GET",
		Url:        fmt.Sprintf("https://scan-%s.stackyard.test/page-%d", strings.ReplaceAll(scanRunName, "/", "-"), index),
		Body:       "",
	}
}

func gcpStage4WebSecurityScannerFinding(project, scanConfigID, scanRunID, findingID, findingType string) *websecurityscannerpb.Finding {
	return &websecurityscannerpb.Finding{
		Name:            gcpWebSecurityScannerFindingName(project, scanConfigID, scanRunID, findingID),
		FindingType:     strings.ToUpper(strings.TrimSpace(findingType)),
		Severity:        websecurityscannerpb.Finding_MEDIUM,
		HttpMethod:      "GET",
		FuzzedUrl:       fmt.Sprintf("https://%s.stackyard.test/vuln/%s", scanConfigID, findingID),
		Description:     "Stackyard staged finding fixture",
		ReproductionUrl: fmt.Sprintf("https://%s.stackyard.test/repro/%s", scanConfigID, findingID),
		FrameUrl:        fmt.Sprintf("https://%s.stackyard.test/frame", scanConfigID),
		FinalUrl:        fmt.Sprintf("https://%s.stackyard.test/final", scanConfigID),
		TrackingId:      fmt.Sprintf("tracking-%s", findingID),
	}
}

func gcpStage4WebSecurityScannerFindingTypeStat(findingType string, count int32) *websecurityscannerpb.FindingTypeStats {
	return &websecurityscannerpb.FindingTypeStats{
		FindingType:  strings.ToUpper(strings.TrimSpace(findingType)),
		FindingCount: count,
	}
}
