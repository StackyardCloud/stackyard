package server

type codeGuruProfilerDataType struct {
	Name string
}

// Amazon CodeGuru Profiler data types sourced from:
// https://docs.aws.amazon.com/codeguru/latest/profiler-api/API_Types.html
var codeGuruProfilerDataTypes = []codeGuruProfilerDataType{
	{Name: "AgentConfiguration"},
	{Name: "AgentOrchestrationConfig"},
	{Name: "AggregatedProfileTime"},
	{Name: "Anomaly"},
	{Name: "AnomalyInstance"},
	{Name: "Channel"},
	{Name: "FindingsReportSummary"},
	{Name: "FrameMetric"},
	{Name: "FrameMetricDatum"},
	{Name: "Match"},
	{Name: "Metric"},
	{Name: "NotificationConfiguration"},
	{Name: "Pattern"},
	{Name: "ProfileTime"},
	{Name: "ProfilingGroupDescription"},
	{Name: "ProfilingStatus"},
	{Name: "Recommendation"},
	{Name: "TimestampStructure"},
	{Name: "UserFeedback"},
}

var codeGuruProfilerDataTypeByName = func() map[string]codeGuruProfilerDataType {
	out := make(map[string]codeGuruProfilerDataType, len(codeGuruProfilerDataTypes))
	for _, dt := range codeGuruProfilerDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
