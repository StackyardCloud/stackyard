package server

type ec2AutoScalingOperation struct {
	Name string
}

// Amazon EC2 Auto Scaling operations sourced from:
// https://docs.aws.amazon.com/autoscaling/ec2/APIReference/API_Operations.html
var ec2AutoScalingOperations = []ec2AutoScalingOperation{
	{Name: "AttachInstances"},
	{Name: "AttachLoadBalancerTargetGroups"},
	{Name: "AttachLoadBalancers"},
	{Name: "AttachTrafficSources"},
	{Name: "BatchDeleteScheduledAction"},
	{Name: "BatchPutScheduledUpdateGroupAction"},
	{Name: "CancelInstanceRefresh"},
	{Name: "CompleteLifecycleAction"},
	{Name: "CreateAutoScalingGroup"},
	{Name: "CreateLaunchConfiguration"},
	{Name: "CreateOrUpdateTags"},
	{Name: "DeleteAutoScalingGroup"},
	{Name: "DeleteLaunchConfiguration"},
	{Name: "DeleteLifecycleHook"},
	{Name: "DeleteNotificationConfiguration"},
	{Name: "DeletePolicy"},
	{Name: "DeleteScheduledAction"},
	{Name: "DeleteTags"},
	{Name: "DeleteWarmPool"},
	{Name: "DescribeAccountLimits"},
	{Name: "DescribeAdjustmentTypes"},
	{Name: "DescribeAutoScalingGroups"},
	{Name: "DescribeAutoScalingInstances"},
	{Name: "DescribeAutoScalingNotificationTypes"},
	{Name: "DescribeInstanceRefreshes"},
	{Name: "DescribeLaunchConfigurations"},
	{Name: "DescribeLifecycleHookTypes"},
	{Name: "DescribeLifecycleHooks"},
	{Name: "DescribeLoadBalancerTargetGroups"},
	{Name: "DescribeLoadBalancers"},
	{Name: "DescribeMetricCollectionTypes"},
	{Name: "DescribeNotificationConfigurations"},
	{Name: "DescribePolicies"},
	{Name: "DescribeScalingActivities"},
	{Name: "DescribeScalingProcessTypes"},
	{Name: "DescribeScheduledActions"},
	{Name: "DescribeTags"},
	{Name: "DescribeTerminationPolicyTypes"},
	{Name: "DescribeTrafficSources"},
	{Name: "DescribeWarmPool"},
	{Name: "DetachInstances"},
	{Name: "DetachLoadBalancerTargetGroups"},
	{Name: "DetachLoadBalancers"},
	{Name: "DetachTrafficSources"},
	{Name: "DisableMetricsCollection"},
	{Name: "EnableMetricsCollection"},
	{Name: "EnterStandby"},
	{Name: "ExecutePolicy"},
	{Name: "ExitStandby"},
	{Name: "GetPredictiveScalingForecast"},
	{Name: "LaunchInstances"},
	{Name: "PutLifecycleHook"},
	{Name: "PutNotificationConfiguration"},
	{Name: "PutScalingPolicy"},
	{Name: "PutScheduledUpdateGroupAction"},
	{Name: "PutWarmPool"},
	{Name: "RecordLifecycleActionHeartbeat"},
	{Name: "ResumeProcesses"},
	{Name: "RollbackInstanceRefresh"},
	{Name: "SetDesiredCapacity"},
	{Name: "SetInstanceHealth"},
	{Name: "SetInstanceProtection"},
	{Name: "StartInstanceRefresh"},
	{Name: "SuspendProcesses"},
	{Name: "TerminateInstanceInAutoScalingGroup"},
	{Name: "UpdateAutoScalingGroup"},
}

var ec2AutoScalingOperationByName = func() map[string]ec2AutoScalingOperation {
	out := make(map[string]ec2AutoScalingOperation, len(ec2AutoScalingOperations))
	for _, op := range ec2AutoScalingOperations {
		out[op.Name] = op
	}
	return out
}()
