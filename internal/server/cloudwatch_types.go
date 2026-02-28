package server

type cloudWatchDataType struct {
	Name string
}

// Amazon CloudWatch data types sourced from:
// https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_Types.html
var cloudWatchDataTypes = []cloudWatchDataType{
	{Name: "AlarmContributor"},
	{Name: "AlarmHistoryItem"},
	{Name: "AlarmMuteRuleSummary"},
	{Name: "AnomalyDetector"},
	{Name: "AnomalyDetectorConfiguration"},
	{Name: "CompositeAlarm"},
	{Name: "DashboardEntry"},
	{Name: "DashboardValidationMessage"},
	{Name: "Datapoint"},
	{Name: "Dimension"},
	{Name: "DimensionFilter"},
	{Name: "Entity"},
	{Name: "EntityMetricData"},
	{Name: "InsightRule"},
	{Name: "InsightRuleContributor"},
	{Name: "InsightRuleContributorDatapoint"},
	{Name: "InsightRuleMetricDatapoint"},
	{Name: "LabelOptions"},
	{Name: "ManagedRule"},
	{Name: "ManagedRuleDescription"},
	{Name: "ManagedRuleState"},
	{Name: "MessageData"},
	{Name: "Metric"},
	{Name: "MetricAlarm"},
	{Name: "MetricCharacteristics"},
	{Name: "MetricDataQuery"},
	{Name: "MetricDataResult"},
	{Name: "MetricDatum"},
	{Name: "MetricMathAnomalyDetector"},
	{Name: "MetricStat"},
	{Name: "MetricStreamEntry"},
	{Name: "MetricStreamFilter"},
	{Name: "MetricStreamStatisticsConfiguration"},
	{Name: "MetricStreamStatisticsMetric"},
	{Name: "MuteTargets"},
	{Name: "PartialFailure"},
	{Name: "Range"},
	{Name: "Rule"},
	{Name: "Schedule"},
	{Name: "SingleMetricAnomalyDetector"},
	{Name: "StatisticSet"},
	{Name: "Tag"},
	{Name: "UntagResource"},
}

var cloudWatchDataTypeByName = func() map[string]cloudWatchDataType {
	out := make(map[string]cloudWatchDataType, len(cloudWatchDataTypes))
	for _, dt := range cloudWatchDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
