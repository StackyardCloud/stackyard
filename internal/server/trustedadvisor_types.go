package server

type trustedAdvisorDataType struct {
	Name string
}

// AWS Trusted Advisor data types sourced from:
// https://docs.aws.amazon.com/trustedadvisor/latest/APIReference/API_Types.html
var trustedAdvisorDataTypes = []trustedAdvisorDataType{
	{Name: "AccountRecommendationLifecycleSummary"},
	{Name: "CheckSummary"},
	{Name: "OrganizationRecommendation"},
	{Name: "OrganizationRecommendationResourceSummary"},
	{Name: "OrganizationRecommendationSummary"},
	{Name: "Recommendation"},
	{Name: "RecommendationCostOptimizingAggregates"},
	{Name: "RecommendationPillarSpecificAggregates"},
	{Name: "RecommendationResourceExclusion"},
	{Name: "RecommendationResourcesAggregates"},
	{Name: "RecommendationResourceSummary"},
	{Name: "RecommendationSummary"},
	{Name: "UpdateRecommendationResourceExclusionError"},
}

var trustedAdvisorDataTypeByName = func() map[string]trustedAdvisorDataType {
	out := make(map[string]trustedAdvisorDataType, len(trustedAdvisorDataTypes))
	for _, dt := range trustedAdvisorDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
