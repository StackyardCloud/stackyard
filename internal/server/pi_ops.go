package server

type piOperation struct{ Name string }

// Amazon RDS Performance Insights operations sourced from:
// https://docs.aws.amazon.com/performance-insights/latest/APIReference/API_Operations.html
var piOperations = []piOperation{
	{Name: "CreatePerformanceAnalysisReport"},
	{Name: "DeletePerformanceAnalysisReport"},
	{Name: "DescribeDimensionKeys"},
	{Name: "GetDimensionKeyDetails"},
	{Name: "GetPerformanceAnalysisReport"},
	{Name: "GetResourceMetadata"},
	{Name: "GetResourceMetrics"},
	{Name: "ListAvailableResourceDimensions"},
	{Name: "ListAvailableResourceMetrics"},
	{Name: "ListPerformanceAnalysisReportRecommendations"},
	{Name: "ListPerformanceAnalysisReports"},
	{Name: "ListTagsForResource"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
}

var piOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range piOperations {
		out[op.Name] = true
	}
	return out
}()
