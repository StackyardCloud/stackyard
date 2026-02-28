package server

type cloudWatchOperation struct {
	Name string
}

// Amazon CloudWatch operations sourced from:
// https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_Operations.html
var cloudWatchOperations = []cloudWatchOperation{
	{Name: "DeleteAlarmMuteRule"},
	{Name: "DeleteAlarms"},
	{Name: "DeleteAnomalyDetector"},
	{Name: "DeleteDashboards"},
	{Name: "DeleteInsightRules"},
	{Name: "DeleteMetricStream"},
	{Name: "DescribeAlarmContributors"},
	{Name: "DescribeAlarmHistory"},
	{Name: "DescribeAlarms"},
	{Name: "DescribeAlarmsForMetric"},
	{Name: "DescribeAnomalyDetectors"},
	{Name: "DescribeInsightRules"},
	{Name: "DisableAlarmActions"},
	{Name: "DisableInsightRules"},
	{Name: "EnableAlarmActions"},
	{Name: "EnableInsightRules"},
	{Name: "GetAlarmMuteRule"},
	{Name: "GetDashboard"},
	{Name: "GetInsightRuleReport"},
	{Name: "GetMetricData"},
	{Name: "GetMetricStatistics"},
	{Name: "GetMetricStream"},
	{Name: "GetMetricWidgetImage"},
	{Name: "ListAlarmMuteRules"},
	{Name: "ListDashboards"},
	{Name: "ListManagedInsightRules"},
	{Name: "ListMetricStreams"},
	{Name: "ListMetrics"},
	{Name: "ListTagsForResource"},
	{Name: "PutAlarmMuteRule"},
	{Name: "PutAnomalyDetector"},
	{Name: "PutCompositeAlarm"},
	{Name: "PutDashboard"},
	{Name: "PutInsightRule"},
	{Name: "PutManagedInsightRules"},
	{Name: "PutMetricAlarm"},
	{Name: "PutMetricData"},
	{Name: "PutMetricStream"},
	{Name: "SetAlarmState"},
	{Name: "StartMetricStreams"},
	{Name: "StopMetricStreams"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
}

var cloudWatchOperationByName = func() map[string]cloudWatchOperation {
	out := make(map[string]cloudWatchOperation, len(cloudWatchOperations))
	for _, op := range cloudWatchOperations {
		out[op.Name] = op
	}
	return out
}()
