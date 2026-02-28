package server

type cloudWatchApplicationInsightsOperation struct {
	Name string
}

// Amazon CloudWatch Application Insights operations sourced from:
// https://docs.aws.amazon.com/cloudwatch/latest/APIReference/API_Operations.html
var cloudWatchApplicationInsightsOperations = []cloudWatchApplicationInsightsOperation{
	{Name: "AddWorkload"},
	{Name: "CreateApplication"},
	{Name: "CreateComponent"},
	{Name: "CreateLogPattern"},
	{Name: "DeleteApplication"},
	{Name: "DeleteComponent"},
	{Name: "DeleteLogPattern"},
	{Name: "DescribeApplication"},
	{Name: "DescribeComponent"},
	{Name: "DescribeComponentConfiguration"},
	{Name: "DescribeComponentConfigurationRecommendation"},
	{Name: "DescribeLogPattern"},
	{Name: "DescribeObservation"},
	{Name: "DescribeProblem"},
	{Name: "DescribeProblemObservations"},
	{Name: "DescribeWorkload"},
	{Name: "ListApplications"},
	{Name: "ListComponents"},
	{Name: "ListConfigurationHistory"},
	{Name: "ListLogPatternSets"},
	{Name: "ListLogPatterns"},
	{Name: "ListProblems"},
	{Name: "ListTagsForResource"},
	{Name: "ListWorkloads"},
	{Name: "RemoveWorkload"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateApplication"},
	{Name: "UpdateComponent"},
	{Name: "UpdateComponentConfiguration"},
	{Name: "UpdateLogPattern"},
	{Name: "UpdateProblem"},
	{Name: "UpdateWorkload"},
}

var cloudWatchApplicationInsightsOperationByName = func() map[string]cloudWatchApplicationInsightsOperation {
	out := make(map[string]cloudWatchApplicationInsightsOperation, len(cloudWatchApplicationInsightsOperations))
	for _, op := range cloudWatchApplicationInsightsOperations {
		out[op.Name] = op
	}
	return out
}()
