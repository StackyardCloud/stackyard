package server

type bcmpcOperation struct{ Name string }

// AWS Billing and Cost Management Pricing Calculator operations sourced from:
// https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_Operations_AWS_Billing_and_Cost_Management_Pricing_Calculator.html
// (GetUsageLinesEstimatedCost from the AWS CLI 2.42.1 model).
var bcmpcOperations = []bcmpcOperation{
	{Name: "BatchCreateBillScenarioCommitmentModification"},
	{Name: "BatchCreateBillScenarioUsageModification"},
	{Name: "BatchCreateWorkloadEstimateUsage"},
	{Name: "BatchDeleteBillScenarioCommitmentModification"},
	{Name: "BatchDeleteBillScenarioUsageModification"},
	{Name: "BatchDeleteWorkloadEstimateUsage"},
	{Name: "BatchUpdateBillScenarioCommitmentModification"},
	{Name: "BatchUpdateBillScenarioUsageModification"},
	{Name: "BatchUpdateWorkloadEstimateUsage"},
	{Name: "CreateBillEstimate"},
	{Name: "CreateBillScenario"},
	{Name: "CreateWorkloadEstimate"},
	{Name: "DeleteBillEstimate"},
	{Name: "DeleteBillScenario"},
	{Name: "DeleteWorkloadEstimate"},
	{Name: "GetBillEstimate"},
	{Name: "GetBillScenario"},
	{Name: "GetPreferences"},
	{Name: "GetUsageLinesEstimatedCost"},
	{Name: "GetWorkloadEstimate"},
	{Name: "ListBillEstimateCommitments"},
	{Name: "ListBillEstimateInputCommitmentModifications"},
	{Name: "ListBillEstimateInputUsageModifications"},
	{Name: "ListBillEstimateLineItems"},
	{Name: "ListBillEstimates"},
	{Name: "ListBillScenarioCommitmentModifications"},
	{Name: "ListBillScenarioUsageModifications"},
	{Name: "ListBillScenarios"},
	{Name: "ListTagsForResource"},
	{Name: "ListWorkloadEstimateUsage"},
	{Name: "ListWorkloadEstimates"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateBillEstimate"},
	{Name: "UpdateBillScenario"},
	{Name: "UpdatePreferences"},
	{Name: "UpdateWorkloadEstimate"},
}

var bcmpcOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range bcmpcOperations {
		out[op.Name] = true
	}
	return out
}()
