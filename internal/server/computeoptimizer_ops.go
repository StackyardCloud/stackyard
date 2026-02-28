package server

type computeOptimizerOperation struct {
	Name string
}

// AWS Compute Optimizer operations sourced from:
// https://docs.aws.amazon.com/compute-optimizer/latest/APIReference/API_Operations.html
var computeOptimizerOperations = []computeOptimizerOperation{
	{Name: "DeleteRecommendationPreferences"},
	{Name: "DescribeRecommendationExportJobs"},
	{Name: "ExportAutoScalingGroupRecommendations"},
	{Name: "ExportEBSVolumeRecommendations"},
	{Name: "ExportEC2InstanceRecommendations"},
	{Name: "ExportECSServiceRecommendations"},
	{Name: "ExportIdleRecommendations"},
	{Name: "ExportLambdaFunctionRecommendations"},
	{Name: "ExportLicenseRecommendations"},
	{Name: "ExportRDSDatabaseRecommendations"},
	{Name: "GetAutoScalingGroupRecommendations"},
	{Name: "GetEBSVolumeRecommendations"},
	{Name: "GetEC2InstanceRecommendations"},
	{Name: "GetEC2RecommendationProjectedMetrics"},
	{Name: "GetECSServiceRecommendationProjectedMetrics"},
	{Name: "GetECSServiceRecommendations"},
	{Name: "GetEffectiveRecommendationPreferences"},
	{Name: "GetEnrollmentStatus"},
	{Name: "GetEnrollmentStatusesForOrganization"},
	{Name: "GetIdleRecommendations"},
	{Name: "GetLambdaFunctionRecommendations"},
	{Name: "GetLicenseRecommendations"},
	{Name: "GetRDSDatabaseRecommendationProjectedMetrics"},
	{Name: "GetRDSDatabaseRecommendations"},
	{Name: "GetRecommendationPreferences"},
	{Name: "GetRecommendationSummaries"},
	{Name: "PutRecommendationPreferences"},
	{Name: "UpdateEnrollmentStatus"},
	{Name: "automation_AssociateAccounts"},
	{Name: "automation_CreateAutomationRule"},
	{Name: "automation_DeleteAutomationRule"},
	{Name: "automation_DisassociateAccounts"},
	{Name: "automation_GetAutomationEvent"},
	{Name: "automation_GetAutomationRule"},
	{Name: "automation_GetEnrollmentConfiguration"},
	{Name: "automation_ListAccounts"},
	{Name: "automation_ListAutomationEvents"},
	{Name: "automation_ListAutomationEventSteps"},
	{Name: "automation_ListAutomationEventSummaries"},
	{Name: "automation_ListAutomationRulePreview"},
	{Name: "automation_ListAutomationRulePreviewSummaries"},
	{Name: "automation_ListAutomationRules"},
	{Name: "automation_ListRecommendedActions"},
	{Name: "automation_ListRecommendedActionSummaries"},
	{Name: "automation_ListTagsForResource"},
	{Name: "automation_RollbackAutomationEvent"},
	{Name: "automation_StartAutomationEvent"},
	{Name: "automation_TagResource"},
	{Name: "automation_UntagResource"},
	{Name: "automation_UpdateAutomationRule"},
	{Name: "automation_UpdateEnrollmentConfiguration"},
}

var computeOptimizerOperationByName = func() map[string]computeOptimizerOperation {
	out := make(map[string]computeOptimizerOperation, len(computeOptimizerOperations))
	for _, op := range computeOptimizerOperations {
		out[op.Name] = op
	}
	return out
}()
