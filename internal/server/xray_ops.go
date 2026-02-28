package server

type xrayOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS X-Ray actions sourced from:
// https://docs.aws.amazon.com/xray/latest/api/API_Operations.html
var xrayOperations = []xrayOperation{
	{Name: "BatchGetTraces", Method: "POST", URI: "/Traces"},
	{Name: "CancelTraceRetrieval", Method: "POST", URI: "/CancelTraceRetrieval"},
	{Name: "CreateGroup", Method: "POST", URI: "/CreateGroup"},
	{Name: "CreateSamplingRule", Method: "POST", URI: "/CreateSamplingRule"},
	{Name: "DeleteGroup", Method: "POST", URI: "/DeleteGroup"},
	{Name: "DeleteResourcePolicy", Method: "POST", URI: "/DeleteResourcePolicy"},
	{Name: "DeleteSamplingRule", Method: "POST", URI: "/DeleteSamplingRule"},
	{Name: "GetEncryptionConfig", Method: "POST", URI: "/EncryptionConfig"},
	{Name: "GetGroup", Method: "POST", URI: "/GetGroup"},
	{Name: "GetGroups", Method: "POST", URI: "/Groups"},
	{Name: "GetIndexingRules", Method: "POST", URI: "/GetIndexingRules"},
	{Name: "GetInsight", Method: "POST", URI: "/Insight"},
	{Name: "GetInsightEvents", Method: "POST", URI: "/InsightEvents"},
	{Name: "GetInsightImpactGraph", Method: "POST", URI: "/InsightImpactGraph"},
	{Name: "GetInsightSummaries", Method: "POST", URI: "/InsightSummaries"},
	{Name: "GetRetrievedTracesGraph", Method: "POST", URI: "/GetRetrievedTracesGraph"},
	{Name: "GetSamplingRules", Method: "POST", URI: "/GetSamplingRules"},
	{Name: "GetSamplingStatisticSummaries", Method: "POST", URI: "/SamplingStatisticSummaries"},
	{Name: "GetSamplingTargets", Method: "POST", URI: "/SamplingTargets"},
	{Name: "GetServiceGraph", Method: "POST", URI: "/ServiceGraph"},
	{Name: "GetTimeSeriesServiceStatistics", Method: "POST", URI: "/TimeSeriesServiceStatistics"},
	{Name: "GetTraceGraph", Method: "POST", URI: "/TraceGraph"},
	{Name: "GetTraceSegmentDestination", Method: "POST", URI: "/GetTraceSegmentDestination"},
	{Name: "GetTraceSummaries", Method: "POST", URI: "/TraceSummaries"},
	{Name: "ListResourcePolicies", Method: "POST", URI: "/ListResourcePolicies"},
	{Name: "ListRetrievedTraces", Method: "POST", URI: "/ListRetrievedTraces"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/ListTagsForResource"},
	{Name: "PutEncryptionConfig", Method: "POST", URI: "/PutEncryptionConfig"},
	{Name: "PutResourcePolicy", Method: "POST", URI: "/PutResourcePolicy"},
	{Name: "PutTelemetryRecords", Method: "POST", URI: "/TelemetryRecords"},
	{Name: "PutTraceSegments", Method: "POST", URI: "/TraceSegments"},
	{Name: "StartTraceRetrieval", Method: "POST", URI: "/StartTraceRetrieval"},
	{Name: "TagResource", Method: "POST", URI: "/TagResource"},
	{Name: "UntagResource", Method: "POST", URI: "/UntagResource"},
	{Name: "UpdateGroup", Method: "POST", URI: "/UpdateGroup"},
	{Name: "UpdateIndexingRule", Method: "POST", URI: "/UpdateIndexingRule"},
	{Name: "UpdateSamplingRule", Method: "POST", URI: "/UpdateSamplingRule"},
	{Name: "UpdateTraceSegmentDestination", Method: "POST", URI: "/UpdateTraceSegmentDestination"},
}

var xrayOperationByName = func() map[string]xrayOperation {
	out := make(map[string]xrayOperation, len(xrayOperations))
	for _, op := range xrayOperations {
		out[op.Name] = op
	}
	return out
}()
