package server

type autoScalingPlansOperation struct {
	Name string
}

// AWS Auto Scaling (Scaling Plans) operations sourced from:
// https://docs.aws.amazon.com/autoscaling/plans/APIReference/API_Operations.html
var autoScalingPlansOperations = []autoScalingPlansOperation{
	{Name: "CreateScalingPlan"},
	{Name: "DeleteScalingPlan"},
	{Name: "DescribeScalingPlanResources"},
	{Name: "DescribeScalingPlans"},
	{Name: "GetScalingPlanResourceForecastData"},
	{Name: "UpdateScalingPlan"},
}

var autoScalingPlansOperationByName = func() map[string]autoScalingPlansOperation {
	out := make(map[string]autoScalingPlansOperation, len(autoScalingPlansOperations))
	for _, op := range autoScalingPlansOperations {
		out[op.Name] = op
	}
	return out
}()
