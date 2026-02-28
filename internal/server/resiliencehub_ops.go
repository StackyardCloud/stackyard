package server

type resilienceHubOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Resilience Hub operations sourced from:
// https://docs.aws.amazon.com/resilience-hub/latest/APIReference/API_Operations.html
var resilienceHubOperations = []resilienceHubOperation{
	{Name: "AcceptResourceGroupingRecommendations", Method: "POST", URI: "/accept-resource-grouping-recommendations"},
	{Name: "AddDraftAppVersionResourceMappings", Method: "POST", URI: "/add-draft-app-version-resource-mappings"},
	{Name: "BatchUpdateRecommendationStatus", Method: "POST", URI: "/batch-update-recommendation-status"},
	{Name: "CreateApp", Method: "POST", URI: "/create-app"},
	{Name: "CreateAppVersionAppComponent", Method: "POST", URI: "/create-app-version-app-component"},
	{Name: "CreateAppVersionResource", Method: "POST", URI: "/create-app-version-resource"},
	{Name: "CreateRecommendationTemplate", Method: "POST", URI: "/create-recommendation-template"},
	{Name: "CreateResiliencyPolicy", Method: "POST", URI: "/create-resiliency-policy"},
	{Name: "DeleteApp", Method: "POST", URI: "/delete-app"},
	{Name: "DeleteAppAssessment", Method: "POST", URI: "/delete-app-assessment"},
	{Name: "DeleteAppInputSource", Method: "POST", URI: "/delete-app-input-source"},
	{Name: "DeleteAppVersionAppComponent", Method: "POST", URI: "/delete-app-version-app-component"},
	{Name: "DeleteAppVersionResource", Method: "POST", URI: "/delete-app-version-resource"},
	{Name: "DeleteRecommendationTemplate", Method: "POST", URI: "/delete-recommendation-template"},
	{Name: "DeleteResiliencyPolicy", Method: "POST", URI: "/delete-resiliency-policy"},
	{Name: "DescribeApp", Method: "POST", URI: "/describe-app"},
	{Name: "DescribeAppAssessment", Method: "POST", URI: "/describe-app-assessment"},
	{Name: "DescribeAppVersion", Method: "POST", URI: "/describe-app-version"},
	{Name: "DescribeAppVersionAppComponent", Method: "POST", URI: "/describe-app-version-app-component"},
	{Name: "DescribeAppVersionResource", Method: "POST", URI: "/describe-app-version-resource"},
	{Name: "DescribeAppVersionResourcesResolutionStatus", Method: "POST", URI: "/describe-app-version-resources-resolution-status"},
	{Name: "DescribeAppVersionTemplate", Method: "POST", URI: "/describe-app-version-template"},
	{Name: "DescribeDraftAppVersionResourcesImportStatus", Method: "POST", URI: "/describe-draft-app-version-resources-import-status"},
	{Name: "DescribeMetricsExport", Method: "POST", URI: "/describe-metrics-export"},
	{Name: "DescribeResiliencyPolicy", Method: "POST", URI: "/describe-resiliency-policy"},
	{Name: "DescribeResourceGroupingRecommendationTask", Method: "POST", URI: "/describe-resource-grouping-recommendation-task"},
	{Name: "ImportResourcesToDraftAppVersion", Method: "POST", URI: "/import-resources-to-draft-app-version"},
	{Name: "ListAlarmRecommendations", Method: "POST", URI: "/list-alarm-recommendations"},
	{Name: "ListAppAssessmentComplianceDrifts", Method: "POST", URI: "/list-app-assessment-compliance-drifts"},
	{Name: "ListAppAssessmentResourceDrifts", Method: "POST", URI: "/list-app-assessment-resource-drifts"},
	{Name: "ListAppAssessments", Method: "GET", URI: "/list-app-assessments"},
	{Name: "ListAppComponentCompliances", Method: "POST", URI: "/list-app-component-compliances"},
	{Name: "ListAppComponentRecommendations", Method: "POST", URI: "/list-app-component-recommendations"},
	{Name: "ListAppInputSources", Method: "POST", URI: "/list-app-input-sources"},
	{Name: "ListAppVersionAppComponents", Method: "POST", URI: "/list-app-version-app-components"},
	{Name: "ListAppVersionResourceMappings", Method: "POST", URI: "/list-app-version-resource-mappings"},
	{Name: "ListAppVersionResources", Method: "POST", URI: "/list-app-version-resources"},
	{Name: "ListAppVersions", Method: "POST", URI: "/list-app-versions"},
	{Name: "ListApps", Method: "GET", URI: "/list-apps"},
	{Name: "ListMetrics", Method: "POST", URI: "/list-metrics"},
	{Name: "ListRecommendationTemplates", Method: "GET", URI: "/list-recommendation-templates"},
	{Name: "ListResiliencyPolicies", Method: "GET", URI: "/list-resiliency-policies"},
	{Name: "ListResourceGroupingRecommendations", Method: "GET", URI: "/list-resource-grouping-recommendations"},
	{Name: "ListSopRecommendations", Method: "POST", URI: "/list-sop-recommendations"},
	{Name: "ListSuggestedResiliencyPolicies", Method: "GET", URI: "/list-suggested-resiliency-policies"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListTestRecommendations", Method: "POST", URI: "/list-test-recommendations"},
	{Name: "ListUnsupportedAppVersionResources", Method: "POST", URI: "/list-unsupported-app-version-resources"},
	{Name: "PublishAppVersion", Method: "POST", URI: "/publish-app-version"},
	{Name: "PutDraftAppVersionTemplate", Method: "POST", URI: "/put-draft-app-version-template"},
	{Name: "RejectResourceGroupingRecommendations", Method: "POST", URI: "/reject-resource-grouping-recommendations"},
	{Name: "RemoveDraftAppVersionResourceMappings", Method: "POST", URI: "/remove-draft-app-version-resource-mappings"},
	{Name: "ResolveAppVersionResources", Method: "POST", URI: "/resolve-app-version-resources"},
	{Name: "StartAppAssessment", Method: "POST", URI: "/start-app-assessment"},
	{Name: "StartMetricsExport", Method: "POST", URI: "/start-metrics-export"},
	{Name: "StartResourceGroupingRecommendationTask", Method: "POST", URI: "/start-resource-grouping-recommendation-task"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateApp", Method: "POST", URI: "/update-app"},
	{Name: "UpdateAppVersion", Method: "POST", URI: "/update-app-version"},
	{Name: "UpdateAppVersionAppComponent", Method: "POST", URI: "/update-app-version-app-component"},
	{Name: "UpdateAppVersionResource", Method: "POST", URI: "/update-app-version-resource"},
	{Name: "UpdateResiliencyPolicy", Method: "POST", URI: "/update-resiliency-policy"},
}

var resilienceHubOperationByName = func() map[string]resilienceHubOperation {
	out := make(map[string]resilienceHubOperation, len(resilienceHubOperations))
	for _, op := range resilienceHubOperations {
		out[op.Name] = op
	}
	return out
}()
