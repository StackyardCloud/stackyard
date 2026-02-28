package server

type auditManagerDataType struct {
	Name string
}

// AWS Audit Manager data types sourced from:
// https://docs.aws.amazon.com/audit-manager/latest/APIReference/API_Types.html
var auditManagerDataTypes = []auditManagerDataType{
	{Name: "AWSAccount"},
	{Name: "AWSService"},
	{Name: "Assessment"},
	{Name: "AssessmentControl"},
	{Name: "AssessmentControlSet"},
	{Name: "AssessmentEvidenceFolder"},
	{Name: "AssessmentFramework"},
	{Name: "AssessmentFrameworkMetadata"},
	{Name: "AssessmentFrameworkShareRequest"},
	{Name: "AssessmentMetadata"},
	{Name: "AssessmentMetadataItem"},
	{Name: "AssessmentReport"},
	{Name: "AssessmentReportEvidenceError"},
	{Name: "AssessmentReportMetadata"},
	{Name: "AssessmentReportsDestination"},
	{Name: "BatchCreateDelegationByAssessmentError"},
	{Name: "BatchDeleteDelegationByAssessmentError"},
	{Name: "BatchImportEvidenceToAssessmentControlError"},
	{Name: "ChangeLog"},
	{Name: "Control"},
	{Name: "ControlComment"},
	{Name: "ControlDomainInsights"},
	{Name: "ControlInsightsMetadataByAssessmentItem"},
	{Name: "ControlInsightsMetadataItem"},
	{Name: "ControlMappingSource"},
	{Name: "ControlMetadata"},
	{Name: "ControlSet"},
	{Name: "CreateAssessmentFrameworkControl"},
	{Name: "CreateAssessmentFrameworkControlSet"},
	{Name: "CreateControlMappingSource"},
	{Name: "CreateDelegationRequest"},
	{Name: "DefaultExportDestination"},
	{Name: "Delegation"},
	{Name: "DelegationMetadata"},
	{Name: "DeregistrationPolicy"},
	{Name: "Evidence"},
	{Name: "EvidenceFinderEnablement"},
	{Name: "EvidenceInsights"},
	{Name: "Framework"},
	{Name: "FrameworkMetadata"},
	{Name: "Insights"},
	{Name: "InsightsByAssessment"},
	{Name: "ManualEvidence"},
	{Name: "Notification"},
	{Name: "Resource"},
	{Name: "Role"},
	{Name: "Scope"},
	{Name: "ServiceMetadata"},
	{Name: "Settings"},
	{Name: "SourceKeyword"},
	{Name: "URL"},
	{Name: "UpdateAssessmentFrameworkControlSet"},
	{Name: "ValidateAssessmentReportIntegrity"},
	{Name: "ValidationExceptionField"},
}

var auditManagerDataTypeByName = func() map[string]auditManagerDataType {
	out := make(map[string]auditManagerDataType, len(auditManagerDataTypes))
	for _, dt := range auditManagerDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
