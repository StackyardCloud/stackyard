package server

type applicationautoscalingOperation struct {
	Name string
}

// Application Auto Scaling operations sourced from:
// https://docs.aws.amazon.com/autoscaling/application/APIReference/API_Operations.html
var applicationautoscalingOperations = []applicationautoscalingOperation{
	{Name: "DeleteScalingPolicy"},
	{Name: "DeleteScheduledAction"},
	{Name: "DeregisterScalableTarget"},
	{Name: "DescribeScalableTargets"},
	{Name: "DescribeScalingActivities"},
	{Name: "DescribeScalingPolicies"},
	{Name: "DescribeScheduledActions"},
	{Name: "GetPredictiveScalingForecast"},
	{Name: "ListTagsForResource"},
	{Name: "PutScalingPolicy"},
	{Name: "PutScheduledAction"},
	{Name: "RegisterScalableTarget"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
}

var applicationautoscalingOperationByName = func() map[string]applicationautoscalingOperation {
	out := make(map[string]applicationautoscalingOperation, len(applicationautoscalingOperations))
	for _, op := range applicationautoscalingOperations {
		out[op.Name] = op
	}
	return out
}()
