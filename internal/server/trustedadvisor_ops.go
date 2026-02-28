package server

type trustedAdvisorOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Trusted Advisor operations sourced from:
// https://docs.aws.amazon.com/trustedadvisor/latest/APIReference/API_Operations.html
var trustedAdvisorOperations = []trustedAdvisorOperation{
	{Name: "BatchUpdateRecommendationResourceExclusion", Method: "PUT", URI: "/v1/batch-update-recommendation-resource-exclusion"},
	{Name: "GetOrganizationRecommendation", Method: "GET", URI: "/v1/organization-recommendations/{organizationRecommendationIdentifier}"},
	{Name: "GetRecommendation", Method: "GET", URI: "/v1/recommendations/{recommendationIdentifier}"},
	{Name: "ListChecks", Method: "GET", URI: "/v1/checks"},
	{Name: "ListOrganizationRecommendationAccounts", Method: "GET", URI: "/v1/organization-recommendations/{organizationRecommendationIdentifier}/accounts"},
	{Name: "ListOrganizationRecommendationResources", Method: "GET", URI: "/v1/organization-recommendations/{organizationRecommendationIdentifier}/resources"},
	{Name: "ListOrganizationRecommendations", Method: "GET", URI: "/v1/organization-recommendations"},
	{Name: "ListRecommendationResources", Method: "GET", URI: "/v1/recommendations/{recommendationIdentifier}/resources"},
	{Name: "ListRecommendations", Method: "GET", URI: "/v1/recommendations"},
	{Name: "UpdateOrganizationRecommendationLifecycle", Method: "PUT", URI: "/v1/organization-recommendations/{organizationRecommendationIdentifier}/lifecycle"},
	{Name: "UpdateRecommendationLifecycle", Method: "PUT", URI: "/v1/recommendations/{recommendationIdentifier}/lifecycle"},
}

var trustedAdvisorOperationByName = func() map[string]trustedAdvisorOperation {
	out := make(map[string]trustedAdvisorOperation, len(trustedAdvisorOperations))
	for _, op := range trustedAdvisorOperations {
		out[op.Name] = op
	}
	return out
}()
