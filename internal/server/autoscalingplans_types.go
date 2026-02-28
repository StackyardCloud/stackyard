package server

type autoScalingPlansDataType struct {
	Name string
}

// AWS Auto Scaling (Scaling Plans) data types sourced from:
// https://docs.aws.amazon.com/autoscaling/plans/APIReference/API_Types.html
var autoScalingPlansDataTypes = []autoScalingPlansDataType{
	{Name: "ApplicationSource"},
	{Name: "CustomizedLoadMetricSpecification"},
	{Name: "CustomizedScalingMetricSpecification"},
	{Name: "Datapoint"},
	{Name: "MetricDimension"},
	{Name: "PredefinedLoadMetricSpecification"},
	{Name: "PredefinedScalingMetricSpecification"},
	{Name: "ScalingInstruction"},
	{Name: "ScalingPlan"},
	{Name: "ScalingPlanResource"},
	{Name: "ScalingPolicy"},
	{Name: "TagFilter"},
	{Name: "TargetTrackingConfiguration"},
	{Name: "UpdateScalingPlan"},
}

var autoScalingPlansDataTypeByName = func() map[string]autoScalingPlansDataType {
	out := make(map[string]autoScalingPlansDataType, len(autoScalingPlansDataTypes))
	for _, dt := range autoScalingPlansDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
