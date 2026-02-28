package server

type auditManagerOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Audit Manager operations sourced from:
// https://docs.aws.amazon.com/audit-manager/latest/APIReference/API_Operations.html
var auditManagerOperations = []auditManagerOperation{
	{Name: "AssociateAssessmentReportEvidenceFolder", Method: "PUT", URI: "/assessments/{assessmentId}/associateToAssessmentReport"},
	{Name: "BatchAssociateAssessmentReportEvidence", Method: "PUT", URI: "/assessments/{assessmentId}/batchAssociateToAssessmentReport"},
	{Name: "BatchCreateDelegationByAssessment", Method: "POST", URI: "/assessments/{assessmentId}/delegations"},
	{Name: "BatchDeleteDelegationByAssessment", Method: "PUT", URI: "/assessments/{assessmentId}/delegations"},
	{Name: "BatchDisassociateAssessmentReportEvidence", Method: "PUT", URI: "/assessments/{assessmentId}/batchDisassociateFromAssessmentReport"},
	{Name: "BatchImportEvidenceToAssessmentControl", Method: "POST", URI: "/assessments/{assessmentId}/controlSets/{controlSetId}/controls/{controlId}/evidence"},
	{Name: "CreateAssessment", Method: "POST", URI: "/assessments"},
	{Name: "CreateAssessmentFramework", Method: "POST", URI: "/assessmentFrameworks"},
	{Name: "CreateAssessmentReport", Method: "POST", URI: "/assessments/{assessmentId}/reports"},
	{Name: "CreateControl", Method: "POST", URI: "/controls"},
	{Name: "DeleteAssessment", Method: "DELETE", URI: "/assessments/{assessmentId}"},
	{Name: "DeleteAssessmentFramework", Method: "DELETE", URI: "/assessmentFrameworks/{frameworkId}"},
	{Name: "DeleteAssessmentFrameworkShare", Method: "DELETE", URI: "/assessmentFrameworkShareRequests/{requestId}"},
	{Name: "DeleteAssessmentReport", Method: "DELETE", URI: "/assessments/{assessmentId}/reports/{assessmentReportId}"},
	{Name: "DeleteControl", Method: "DELETE", URI: "/controls/{controlId}"},
	{Name: "DeregisterAccount", Method: "POST", URI: "/account/deregisterAccount"},
	{Name: "DeregisterOrganizationAdminAccount", Method: "POST", URI: "/account/deregisterOrganizationAdminAccount"},
	{Name: "DisassociateAssessmentReportEvidenceFolder", Method: "PUT", URI: "/assessments/{assessmentId}/disassociateFromAssessmentReport"},
	{Name: "GetAccountStatus", Method: "GET", URI: "/account/status"},
	{Name: "GetAssessment", Method: "GET", URI: "/assessments/{assessmentId}"},
	{Name: "GetAssessmentFramework", Method: "GET", URI: "/assessmentFrameworks/{frameworkId}"},
	{Name: "GetAssessmentReportUrl", Method: "GET", URI: "/assessments/{assessmentId}/reports/{assessmentReportId}/url"},
	{Name: "GetChangeLogs", Method: "GET", URI: "/assessments/{assessmentId}/changelogs"},
	{Name: "GetControl", Method: "GET", URI: "/controls/{controlId}"},
	{Name: "GetDelegations", Method: "GET", URI: "/delegations"},
	{Name: "GetEvidence", Method: "GET", URI: "/assessments/{assessmentId}/controlSets/{controlSetId}/evidenceFolders/{evidenceFolderId}/evidence/{evidenceId}"},
	{Name: "GetEvidenceByEvidenceFolder", Method: "GET", URI: "/assessments/{assessmentId}/controlSets/{controlSetId}/evidenceFolders/{evidenceFolderId}/evidence"},
	{Name: "GetEvidenceFileUploadUrl", Method: "GET", URI: "/evidenceFileUploadUrl"},
	{Name: "GetEvidenceFolder", Method: "GET", URI: "/assessments/{assessmentId}/controlSets/{controlSetId}/evidenceFolders/{evidenceFolderId}"},
	{Name: "GetEvidenceFoldersByAssessment", Method: "GET", URI: "/assessments/{assessmentId}/evidenceFolders"},
	{Name: "GetEvidenceFoldersByAssessmentControl", Method: "GET", URI: "/assessments/{assessmentId}/evidenceFolders-by-assessment-control/{controlSetId}/{controlId}"},
	{Name: "GetInsights", Method: "GET", URI: "/insights"},
	{Name: "GetInsightsByAssessment", Method: "GET", URI: "/insights/assessments/{assessmentId}"},
	{Name: "GetOrganizationAdminAccount", Method: "GET", URI: "/account/organizationAdminAccount"},
	{Name: "GetServicesInScope", Method: "GET", URI: "/services"},
	{Name: "GetSettings", Method: "GET", URI: "/settings/{attribute}"},
	{Name: "ListAssessmentControlInsightsByControlDomain", Method: "GET", URI: "/insights/controls-by-assessment"},
	{Name: "ListAssessmentFrameworkShareRequests", Method: "GET", URI: "/assessmentFrameworkShareRequests"},
	{Name: "ListAssessmentFrameworks", Method: "GET", URI: "/assessmentFrameworks"},
	{Name: "ListAssessmentReports", Method: "GET", URI: "/assessmentReports"},
	{Name: "ListAssessments", Method: "GET", URI: "/assessments"},
	{Name: "ListControlDomainInsights", Method: "GET", URI: "/insights/control-domains"},
	{Name: "ListControlDomainInsightsByAssessment", Method: "GET", URI: "/insights/control-domains-by-assessment"},
	{Name: "ListControlInsightsByControlDomain", Method: "GET", URI: "/insights/controls"},
	{Name: "ListControls", Method: "GET", URI: "/controls"},
	{Name: "ListKeywordsForDataSource", Method: "GET", URI: "/dataSourceKeywords"},
	{Name: "ListNotifications", Method: "GET", URI: "/notifications"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "RegisterAccount", Method: "POST", URI: "/account/registerAccount"},
	{Name: "RegisterOrganizationAdminAccount", Method: "POST", URI: "/account/registerOrganizationAdminAccount"},
	{Name: "StartAssessmentFrameworkShare", Method: "POST", URI: "/assessmentFrameworks/{frameworkId}/shareRequests"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateAssessment", Method: "PUT", URI: "/assessments/{assessmentId}"},
	{Name: "UpdateAssessmentControl", Method: "PUT", URI: "/assessments/{assessmentId}/controlSets/{controlSetId}/controls/{controlId}"},
	{Name: "UpdateAssessmentControlSetStatus", Method: "PUT", URI: "/assessments/{assessmentId}/controlSets/{controlSetId}/status"},
	{Name: "UpdateAssessmentFramework", Method: "PUT", URI: "/assessmentFrameworks/{frameworkId}"},
	{Name: "UpdateAssessmentFrameworkShare", Method: "PUT", URI: "/assessmentFrameworkShareRequests/{requestId}"},
	{Name: "UpdateAssessmentStatus", Method: "PUT", URI: "/assessments/{assessmentId}/status"},
	{Name: "UpdateControl", Method: "PUT", URI: "/controls/{controlId}"},
	{Name: "UpdateSettings", Method: "PUT", URI: "/settings"},
	{Name: "ValidateAssessmentReportIntegrity", Method: "POST", URI: "/assessmentReports/integrity"},
}

var auditManagerOperationByName = func() map[string]auditManagerOperation {
	out := make(map[string]auditManagerOperation, len(auditManagerOperations))
	for _, op := range auditManagerOperations {
		out[op.Name] = op
	}
	return out
}()
