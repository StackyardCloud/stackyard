package server

type arcRegionSwitchOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Application Recovery Controller Region Switch operations sourced from:
// https://docs.aws.amazon.com/arc-region-switch/latest/APIReference/API_Operations.html
// Mirror path currently resolves under /latest/api/API_Operations.html.
var arcRegionSwitchOperations = []arcRegionSwitchOperation{
	{Name: "ApprovePlanExecutionStep", Method: "POST", URI: "/"},
	{Name: "CancelPlanExecution", Method: "POST", URI: "/"},
	{Name: "CreatePlan", Method: "POST", URI: "/"},
	{Name: "DeletePlan", Method: "POST", URI: "/"},
	{Name: "GetPlan", Method: "POST", URI: "/"},
	{Name: "GetPlanEvaluationStatus", Method: "POST", URI: "/"},
	{Name: "GetPlanExecution", Method: "POST", URI: "/"},
	{Name: "GetPlanInRegion", Method: "POST", URI: "/"},
	{Name: "ListPlanExecutionEvents", Method: "POST", URI: "/"},
	{Name: "ListPlanExecutions", Method: "POST", URI: "/"},
	{Name: "ListPlans", Method: "POST", URI: "/"},
	{Name: "ListPlansInRegion", Method: "POST", URI: "/"},
	{Name: "ListRoute53HealthChecks", Method: "POST", URI: "/"},
	{Name: "ListRoute53HealthChecksInRegion", Method: "POST", URI: "/"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/"},
	{Name: "StartPlanExecution", Method: "POST", URI: "/"},
	{Name: "TagResource", Method: "POST", URI: "/"},
	{Name: "UntagResource", Method: "POST", URI: "/"},
	{Name: "UpdatePlan", Method: "POST", URI: "/"},
	{Name: "UpdatePlanExecution", Method: "POST", URI: "/"},
	{Name: "UpdatePlanExecutionStep", Method: "POST", URI: "/"},
}

var arcRegionSwitchOperationByName = func() map[string]arcRegionSwitchOperation {
	out := make(map[string]arcRegionSwitchOperation, len(arcRegionSwitchOperations))
	for _, op := range arcRegionSwitchOperations {
		out[op.Name] = op
	}
	return out
}()
