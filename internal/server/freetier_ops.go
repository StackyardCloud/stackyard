package server

type ftierOperation struct{ Name string }

// AWS Free Tier operations sourced from:
// https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_Operations_AWS_Free_Tier.html
var ftierOperations = []ftierOperation{
	{Name: "GetAccountActivity"},
	{Name: "GetAccountPlanState"},
	{Name: "GetFreeTierUsage"},
	{Name: "ListAccountActivities"},
	{Name: "UpgradeAccountPlan"},
}
