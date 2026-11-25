package server

type cloudSearchOperation struct{ Name string }

// Amazon CloudSearch operations sourced from:
// https://docs.aws.amazon.com/cloudsearch/latest/developerguide/API_Operations.html
var cloudSearchOperations = []cloudSearchOperation{
	{Name: "BuildSuggesters"},
	{Name: "CreateDomain"},
	{Name: "DefineAnalysisScheme"},
	{Name: "DefineExpression"},
	{Name: "DefineIndexField"},
	{Name: "DefineSuggester"},
	{Name: "DeleteAnalysisScheme"},
	{Name: "DeleteDomain"},
	{Name: "DeleteExpression"},
	{Name: "DeleteIndexField"},
	{Name: "DeleteSuggester"},
	{Name: "DescribeAnalysisSchemes"},
	{Name: "DescribeAvailabilityOptions"},
	{Name: "DescribeDomainEndpointOptions"},
	{Name: "DescribeDomains"},
	{Name: "DescribeExpressions"},
	{Name: "DescribeIndexFields"},
	{Name: "DescribeScalingParameters"},
	{Name: "DescribeServiceAccessPolicies"},
	{Name: "DescribeSuggesters"},
	{Name: "IndexDocuments"},
	{Name: "ListDomainNames"},
	{Name: "UpdateAvailabilityOptions"},
	{Name: "UpdateDomainEndpointOptions"},
	{Name: "UpdateScalingParameters"},
	{Name: "UpdateServiceAccessPolicies"},
}

var cloudSearchOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range cloudSearchOperations {
		out[op.Name] = true
	}
	return out
}()
