package server

type savingsPlansOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Savings Plans actions sourced from:
// https://docs.aws.amazon.com/savingsplans/latest/APIReference/API_Operations.html
var savingsPlansOperations = []savingsPlansOperation{
	{Name: "CreateSavingsPlan", Method: "POST", URI: "/CreateSavingsPlan"},
	{Name: "DeleteQueuedSavingsPlan", Method: "POST", URI: "/DeleteQueuedSavingsPlan"},
	{Name: "DescribeSavingsPlanRates", Method: "POST", URI: "/DescribeSavingsPlanRates"},
	{Name: "DescribeSavingsPlans", Method: "POST", URI: "/DescribeSavingsPlans"},
	{Name: "DescribeSavingsPlansOfferingRates", Method: "POST", URI: "/DescribeSavingsPlansOfferingRates"},
	{Name: "DescribeSavingsPlansOfferings", Method: "POST", URI: "/DescribeSavingsPlansOfferings"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/ListTagsForResource"},
	{Name: "ReturnSavingsPlan", Method: "POST", URI: "/ReturnSavingsPlan"},
	{Name: "TagResource", Method: "POST", URI: "/TagResource"},
	{Name: "UntagResource", Method: "POST", URI: "/UntagResource"},
}

var savingsPlansOperationByName = func() map[string]savingsPlansOperation {
	out := make(map[string]savingsPlansOperation, len(savingsPlansOperations))
	for _, op := range savingsPlansOperations {
		out[op.Name] = op
	}
	return out
}()
