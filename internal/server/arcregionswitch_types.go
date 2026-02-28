package server

type arcRegionSwitchDataType struct {
	Name string
}

// Amazon Application Recovery Controller Region Switch data types sourced from:
// https://docs.aws.amazon.com/arc-region-switch/latest/APIReference/API_Types.html
// Mirror path currently resolves under /latest/api/API_Types.html.
var arcRegionSwitchDataTypes = []arcRegionSwitchDataType{
	{Name: "AbbreviatedExecution"},
	{Name: "AbbreviatedPlan"},
	{Name: "ArcRoutingControlConfiguration"},
	{Name: "ArcRoutingControlState"},
	{Name: "Asg"},
	{Name: "AssociatedAlarm"},
	{Name: "CustomActionLambdaConfiguration"},
	{Name: "DocumentDbConfiguration"},
	{Name: "DocumentDbUngraceful"},
	{Name: "Ec2AsgCapacityIncreaseConfiguration"},
	{Name: "Ec2Ungraceful"},
	{Name: "EcsCapacityIncreaseConfiguration"},
	{Name: "EcsUngraceful"},
	{Name: "EksCluster"},
	{Name: "EksResourceScalingConfiguration"},
	{Name: "EksResourceScalingUngraceful"},
	{Name: "ExecutionApprovalConfiguration"},
	{Name: "ExecutionBlockConfiguration"},
	{Name: "ExecutionEvent"},
	{Name: "FailedReportOutput"},
	{Name: "GeneratedReport"},
	{Name: "GlobalAuroraConfiguration"},
	{Name: "GlobalAuroraUngraceful"},
	{Name: "KubernetesResourceType"},
	{Name: "KubernetesScalingResource"},
	{Name: "LambdaUngraceful"},
	{Name: "Lambdas"},
	{Name: "MinimalWorkflow"},
	{Name: "ParallelExecutionBlockConfiguration"},
	{Name: "Plan"},
	{Name: "RegionSwitchPlanConfiguration"},
	{Name: "ReportConfiguration"},
	{Name: "ReportOutput"},
	{Name: "ReportOutputConfiguration"},
	{Name: "ResourceWarning"},
	{Name: "Route53HealthCheck"},
	{Name: "Route53HealthCheckConfiguration"},
	{Name: "Route53ResourceRecordSet"},
	{Name: "S3ReportOutput"},
	{Name: "S3ReportOutputConfiguration"},
	{Name: "Service"},
	{Name: "Step"},
	{Name: "StepState"},
	{Name: "Trigger"},
	{Name: "TriggerCondition"},
	{Name: "Workflow"},
}

var arcRegionSwitchDataTypeByName = func() map[string]arcRegionSwitchDataType {
	out := make(map[string]arcRegionSwitchDataType, len(arcRegionSwitchDataTypes))
	for _, dt := range arcRegionSwitchDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
