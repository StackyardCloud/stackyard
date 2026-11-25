package server

type kendraRankingOperation struct{ Name string }

// Amazon Kendra Intelligent Ranking operations sourced from:
// https://docs.aws.amazon.com/kendra/latest/APIReference/API_Operations_Amazon_Kendra_Intelligent_Ranking.html
var kendraRankingOperations = []kendraRankingOperation{
	{Name: "CreateRescoreExecutionPlan"},
	{Name: "DeleteRescoreExecutionPlan"},
	{Name: "DescribeRescoreExecutionPlan"},
	{Name: "ListRescoreExecutionPlans"},
	{Name: "ListTagsForResource"},
	{Name: "Rescore"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateRescoreExecutionPlan"},
}

var kendraRankingOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range kendraRankingOperations {
		out[op.Name] = true
	}
	return out
}()
