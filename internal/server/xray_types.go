package server

type xrayDataType struct {
	Name string
}

// AWS X-Ray data types sourced from:
// https://docs.aws.amazon.com/xray/latest/api/API_Types.html
var xrayDataTypes = []xrayDataType{
	{Name: "Alias"},
	{Name: "AnnotationValue"},
	{Name: "AnomalousService"},
	{Name: "AvailabilityZoneDetail"},
	{Name: "BackendConnectionErrors"},
	{Name: "Edge"},
	{Name: "EdgeStatistics"},
	{Name: "EncryptionConfig"},
	{Name: "ErrorRootCause"},
	{Name: "ErrorRootCauseEntity"},
	{Name: "ErrorRootCauseService"},
	{Name: "ErrorStatistics"},
	{Name: "FaultRootCause"},
	{Name: "FaultRootCauseEntity"},
	{Name: "FaultRootCauseService"},
	{Name: "FaultStatistics"},
	{Name: "ForecastStatistics"},
	{Name: "GraphLink"},
	{Name: "Group"},
	{Name: "GroupSummary"},
	{Name: "HistogramEntry"},
	{Name: "Http"},
	{Name: "IndexingRule"},
	{Name: "IndexingRuleValue"},
	{Name: "IndexingRuleValueUpdate"},
	{Name: "Insight"},
	{Name: "InsightEvent"},
	{Name: "InsightImpactGraphEdge"},
	{Name: "InsightImpactGraphService"},
	{Name: "InsightSummary"},
	{Name: "InsightsConfiguration"},
	{Name: "InstanceIdDetail"},
	{Name: "ProbabilisticRuleValue"},
	{Name: "ProbabilisticRuleValueUpdate"},
	{Name: "RequestImpactStatistics"},
	{Name: "ResourceARNDetail"},
	{Name: "ResourcePolicy"},
	{Name: "ResponseTimeRootCause"},
	{Name: "ResponseTimeRootCauseEntity"},
	{Name: "ResponseTimeRootCauseService"},
	{Name: "RetrievedService"},
	{Name: "RetrievedTrace"},
	{Name: "RootCauseException"},
	{Name: "SamplingBoost"},
	{Name: "SamplingBoostStatisticsDocument"},
	{Name: "SamplingRateBoost"},
	{Name: "SamplingRule"},
	{Name: "SamplingRuleRecord"},
	{Name: "SamplingRuleUpdate"},
	{Name: "SamplingStatisticSummary"},
	{Name: "SamplingStatisticsDocument"},
	{Name: "SamplingStrategy"},
	{Name: "SamplingTargetDocument"},
	{Name: "Segment"},
	{Name: "Service"},
	{Name: "ServiceId"},
	{Name: "ServiceStatistics"},
	{Name: "Span"},
	{Name: "Tag"},
	{Name: "TelemetryRecord"},
	{Name: "TimeSeriesServiceStatistics"},
	{Name: "Trace"},
	{Name: "TraceSummary"},
	{Name: "TraceUser"},
	{Name: "UnprocessedStatistics"},
	{Name: "UnprocessedTraceSegment"},
	{Name: "UpdateTraceSegmentDestination"},
	{Name: "ValueWithServiceIds"},
}

var xrayDataTypeByName = func() map[string]xrayDataType {
	out := make(map[string]xrayDataType, len(xrayDataTypes))
	for _, dt := range xrayDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
