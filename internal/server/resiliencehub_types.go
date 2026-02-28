package server

type resilienceHubDataType struct {
	Name string
}

// AWS Resilience Hub data types sourced from:
// https://docs.aws.amazon.com/resilience-hub/latest/APIReference/API_Types.html
var resilienceHubDataTypes = []resilienceHubDataType{
	{Name: "AcceptGroupingRecommendationEntry"},
	{Name: "Alarm"},
	{Name: "AlarmRecommendation"},
	{Name: "App"},
	{Name: "AppAssessment"},
	{Name: "AppAssessmentSummary"},
	{Name: "AppComponent"},
	{Name: "AppComponentCompliance"},
	{Name: "AppInputSource"},
	{Name: "AppSummary"},
	{Name: "AppVersionSummary"},
	{Name: "AssessmentRiskRecommendation"},
	{Name: "AssessmentSummary"},
	{Name: "BatchUpdateRecommendationStatusFailedEntry"},
	{Name: "BatchUpdateRecommendationStatusSuccessfulEntry"},
	{Name: "ComplianceDrift"},
	{Name: "ComponentRecommendation"},
	{Name: "Condition"},
	{Name: "ConfigRecommendation"},
	{Name: "Cost"},
	{Name: "DisruptionCompliance"},
	{Name: "EksSource"},
	{Name: "EksSourceClusterNamespace"},
	{Name: "ErrorDetail"},
	{Name: "EventSubscription"},
	{Name: "Experiment"},
	{Name: "FailedGroupingRecommendationEntry"},
	{Name: "FailurePolicy"},
	{Name: "Field"},
	{Name: "GroupingAppComponent"},
	{Name: "GroupingRecommendation"},
	{Name: "GroupingResource"},
	{Name: "LogicalResourceId"},
	{Name: "PermissionModel"},
	{Name: "PhysicalResource"},
	{Name: "PhysicalResourceId"},
	{Name: "RecommendationDisruptionCompliance"},
	{Name: "RecommendationItem"},
	{Name: "RecommendationTemplate"},
	{Name: "RejectGroupingRecommendationEntry"},
	{Name: "ResiliencyPolicy"},
	{Name: "ResiliencyScore"},
	{Name: "ResourceDrift"},
	{Name: "ResourceError"},
	{Name: "ResourceErrorsDetails"},
	{Name: "ResourceIdentifier"},
	{Name: "ResourceMapping"},
	{Name: "S3Location"},
	{Name: "ScoringComponentResiliencyScore"},
	{Name: "SopRecommendation"},
	{Name: "Sort"},
	{Name: "TerraformSource"},
	{Name: "TestRecommendation"},
	{Name: "UnsupportedResource"},
	{Name: "UpdateRecommendationStatusItem"},
	{Name: "UpdateRecommendationStatusRequestEntry"},
	{Name: "UpdateResiliencyPolicy"},
}

var resilienceHubDataTypeByName = func() map[string]resilienceHubDataType {
	out := make(map[string]resilienceHubDataType, len(resilienceHubDataTypes))
	for _, dt := range resilienceHubDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
