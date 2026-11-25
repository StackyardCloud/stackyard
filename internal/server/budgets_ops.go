package server

type budOperation struct{ Name string }

// AWS Budgets operations sourced from:
// https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_Operations_AWS_Budgets.html
var budOperations = []budOperation{
	{Name: "CreateBudget"},
	{Name: "CreateBudgetAction"},
	{Name: "CreateNotification"},
	{Name: "CreateSubscriber"},
	{Name: "DeleteBudget"},
	{Name: "DeleteBudgetAction"},
	{Name: "DeleteNotification"},
	{Name: "DeleteSubscriber"},
	{Name: "DescribeBudget"},
	{Name: "DescribeBudgetAction"},
	{Name: "DescribeBudgetActionHistories"},
	{Name: "DescribeBudgetActionsForAccount"},
	{Name: "DescribeBudgetActionsForBudget"},
	{Name: "DescribeBudgetNotificationsForAccount"},
	{Name: "DescribeBudgetPerformanceHistory"},
	{Name: "DescribeBudgets"},
	{Name: "DescribeNotificationsForBudget"},
	{Name: "DescribeSubscribersForNotification"},
	{Name: "ExecuteBudgetAction"},
	{Name: "ListTagsForResource"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateBudget"},
	{Name: "UpdateBudgetAction"},
	{Name: "UpdateNotification"},
	{Name: "UpdateSubscriber"},
}

var budOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range budOperations {
		out[op.Name] = true
	}
	return out
}()
