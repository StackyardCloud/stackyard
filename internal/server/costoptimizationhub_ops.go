package server

type cohOperation struct{ Name string }

// AWS Cost Optimization Hub operations sourced from:
// https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_Operations_Cost_Optimization_Hub.html
var cohOperations = []cohOperation{
	{Name: "GetPreferences"},
	{Name: "GetRecommendation"},
	{Name: "ListEfficiencyMetrics"},
	{Name: "ListEnrollmentStatuses"},
	{Name: "ListRecommendationSummaries"},
	{Name: "ListRecommendations"},
	{Name: "UpdateEnrollmentStatus"},
	{Name: "UpdatePreferences"},
}

var cohOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range cohOperations {
		out[op.Name] = true
	}
	return out
}()
