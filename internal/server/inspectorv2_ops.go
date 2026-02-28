package server

type inspectorV2Operation struct {
	Name   string
	Method string
	URI    string
}

// AWS Inspector V2 operations sourced from:
// https://docs.aws.amazon.com/inspector/v2/APIReference/API_Operations.html
var inspectorV2Operations = []inspectorV2Operation{
	{Name: "AssociateMember", Method: "POST", URI: "/members/associate"},
	{Name: "BatchAssociateCodeSecurityScanConfiguration", Method: "POST", URI: "/codesecurity/scan-configuration/batch/associate"},
	{Name: "BatchDisassociateCodeSecurityScanConfiguration", Method: "POST", URI: "/codesecurity/scan-configuration/batch/disassociate"},
	{Name: "BatchGetAccountStatus", Method: "POST", URI: "/status/batch/get"},
	{Name: "BatchGetCodeSnippet", Method: "POST", URI: "/codesnippet/batchget"},
	{Name: "BatchGetFindingDetails", Method: "POST", URI: "/findings/details/batch/get"},
	{Name: "BatchGetFreeTrialInfo", Method: "POST", URI: "/freetrialinfo/batchget"},
	{Name: "BatchGetMemberEc2DeepInspectionStatus", Method: "POST", URI: "/ec2deepinspectionstatus/member/batch/get"},
	{Name: "BatchUpdateMemberEc2DeepInspectionStatus", Method: "POST", URI: "/ec2deepinspectionstatus/member/batch/update"},
	{Name: "CancelFindingsReport", Method: "POST", URI: "/reporting/cancel"},
	{Name: "CancelSbomExport", Method: "POST", URI: "/sbomexport/cancel"},
	{Name: "CreateCisScanConfiguration", Method: "POST", URI: "/cis/scan-configuration/create"},
	{Name: "CreateCodeSecurityIntegration", Method: "POST", URI: "/codesecurity/integration/create"},
	{Name: "CreateCodeSecurityScanConfiguration", Method: "POST", URI: "/codesecurity/scan-configuration/create"},
	{Name: "CreateFilter", Method: "POST", URI: "/filters/create"},
	{Name: "CreateFindingsReport", Method: "POST", URI: "/reporting/create"},
	{Name: "CreateSbomExport", Method: "POST", URI: "/sbomexport/create"},
	{Name: "DeleteCisScanConfiguration", Method: "POST", URI: "/cis/scan-configuration/delete"},
	{Name: "DeleteCodeSecurityIntegration", Method: "POST", URI: "/codesecurity/integration/delete"},
	{Name: "DeleteCodeSecurityScanConfiguration", Method: "POST", URI: "/codesecurity/scan-configuration/delete"},
	{Name: "DeleteFilter", Method: "POST", URI: "/filters/delete"},
	{Name: "DescribeOrganizationConfiguration", Method: "POST", URI: "/organizationconfiguration/describe"},
	{Name: "Disable", Method: "POST", URI: "/disable"},
	{Name: "DisableDelegatedAdminAccount", Method: "POST", URI: "/delegatedadminaccounts/disable"},
	{Name: "DisassociateMember", Method: "POST", URI: "/members/disassociate"},
	{Name: "Enable", Method: "POST", URI: "/enable"},
	{Name: "EnableDelegatedAdminAccount", Method: "POST", URI: "/delegatedadminaccounts/enable"},
	{Name: "GetCisScanReport", Method: "POST", URI: "/cis/scan/report/get"},
	{Name: "GetCisScanResultDetails", Method: "POST", URI: "/cis/scan-result/details/get"},
	{Name: "GetClustersForImage", Method: "POST", URI: "/cluster/get"},
	{Name: "GetCodeSecurityIntegration", Method: "POST", URI: "/codesecurity/integration/get"},
	{Name: "GetCodeSecurityScan", Method: "POST", URI: "/codesecurity/scan/get"},
	{Name: "GetCodeSecurityScanConfiguration", Method: "POST", URI: "/codesecurity/scan-configuration/get"},
	{Name: "GetConfiguration", Method: "POST", URI: "/configuration/get"},
	{Name: "GetDelegatedAdminAccount", Method: "POST", URI: "/delegatedadminaccounts/get"},
	{Name: "GetEc2DeepInspectionConfiguration", Method: "POST", URI: "/ec2deepinspectionconfiguration/get"},
	{Name: "GetEncryptionKey", Method: "GET", URI: "/encryptionkey/get?resourceType={resourceType}&scanType={scanType}"},
	{Name: "GetFindingsReportStatus", Method: "POST", URI: "/reporting/status/get"},
	{Name: "GetMember", Method: "POST", URI: "/members/get"},
	{Name: "GetSbomExport", Method: "POST", URI: "/sbomexport/get"},
	{Name: "ListAccountPermissions", Method: "POST", URI: "/accountpermissions/list"},
	{Name: "ListCisScanConfigurations", Method: "POST", URI: "/cis/scan-configuration/list"},
	{Name: "ListCisScanResultsAggregatedByChecks", Method: "POST", URI: "/cis/scan-result/check/list"},
	{Name: "ListCisScanResultsAggregatedByTargetResource", Method: "POST", URI: "/cis/scan-result/resource/list"},
	{Name: "ListCisScans", Method: "POST", URI: "/cis/scan/list"},
	{Name: "ListCodeSecurityIntegrations", Method: "POST", URI: "/codesecurity/integration/list?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListCodeSecurityScanConfigurationAssociations", Method: "POST", URI: "/codesecurity/scan-configuration/associations/list?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListCodeSecurityScanConfigurations", Method: "POST", URI: "/codesecurity/scan-configuration/list?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListCoverage", Method: "POST", URI: "/coverage/list"},
	{Name: "ListCoverageStatistics", Method: "POST", URI: "/coverage/statistics/list"},
	{Name: "ListDelegatedAdminAccounts", Method: "POST", URI: "/delegatedadminaccounts/list"},
	{Name: "ListFilters", Method: "POST", URI: "/filters/list"},
	{Name: "ListFindingAggregations", Method: "POST", URI: "/findings/aggregation/list"},
	{Name: "ListFindings", Method: "POST", URI: "/findings/list"},
	{Name: "ListMembers", Method: "POST", URI: "/members/list"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListUsageTotals", Method: "POST", URI: "/usage/list"},
	{Name: "ResetEncryptionKey", Method: "PUT", URI: "/encryptionkey/reset"},
	{Name: "SearchVulnerabilities", Method: "POST", URI: "/vulnerabilities/search"},
	{Name: "SendCisSessionHealth", Method: "PUT", URI: "/cissession/health/send"},
	{Name: "SendCisSessionTelemetry", Method: "PUT", URI: "/cissession/telemetry/send"},
	{Name: "StartCisSession", Method: "PUT", URI: "/cissession/start"},
	{Name: "StartCodeSecurityScan", Method: "POST", URI: "/codesecurity/scan/start"},
	{Name: "StopCisSession", Method: "PUT", URI: "/cissession/stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateCisScanConfiguration", Method: "POST", URI: "/cis/scan-configuration/update"},
	{Name: "UpdateCodeSecurityIntegration", Method: "POST", URI: "/codesecurity/integration/update"},
	{Name: "UpdateCodeSecurityScanConfiguration", Method: "POST", URI: "/codesecurity/scan-configuration/update"},
	{Name: "UpdateConfiguration", Method: "POST", URI: "/configuration/update"},
	{Name: "UpdateEc2DeepInspectionConfiguration", Method: "POST", URI: "/ec2deepinspectionconfiguration/update"},
	{Name: "UpdateEncryptionKey", Method: "PUT", URI: "/encryptionkey/update"},
	{Name: "UpdateFilter", Method: "POST", URI: "/filters/update"},
	{Name: "UpdateOrganizationConfiguration", Method: "POST", URI: "/organizationconfiguration/update"},
	{Name: "UpdateOrgEc2DeepInspectionConfiguration", Method: "POST", URI: "/ec2deepinspectionconfiguration/org/update"},
	{Name: "ScanSbom", Method: "POST", URI: "/scan/sbom"},
}

var inspectorV2OperationByName = func() map[string]inspectorV2Operation {
	out := make(map[string]inspectorV2Operation, len(inspectorV2Operations))
	for _, op := range inspectorV2Operations {
		out[op.Name] = op
	}
	return out
}()
