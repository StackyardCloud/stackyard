package server

type iotEventsOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS IoT Events operations sourced from:
// https://docs.aws.amazon.com/iotevents/latest/apireference/API_Operations.html
var iotEventsOperations = []iotEventsOperation{
	{Name: "CreateAlarmModel", Method: "POST", URI: "/alarm-models"},
	{Name: "CreateDetectorModel", Method: "POST", URI: "/detector-models"},
	{Name: "CreateInput", Method: "POST", URI: "/inputs"},
	{Name: "DeleteAlarmModel", Method: "DELETE", URI: "/alarm-models/{alarmModelName}"},
	{Name: "DeleteDetectorModel", Method: "DELETE", URI: "/detector-models/{detectorModelName}"},
	{Name: "DeleteInput", Method: "DELETE", URI: "/inputs/{inputName}"},
	{Name: "DescribeAlarmModel", Method: "GET", URI: "/alarm-models/{alarmModelName}"},
	{Name: "DescribeDetectorModel", Method: "GET", URI: "/detector-models/{detectorModelName}"},
	{Name: "DescribeDetectorModelAnalysis", Method: "GET", URI: "/analysis/detector-models/{analysisId}"},
	{Name: "DescribeInput", Method: "GET", URI: "/inputs/{inputName}"},
	{Name: "DescribeLoggingOptions", Method: "GET", URI: "/logging"},
	{Name: "GetDetectorModelAnalysisResults", Method: "GET", URI: "/analysis/detector-models/{analysisId}/results"},
	{Name: "ListAlarmModelVersions", Method: "GET", URI: "/alarm-models/{alarmModelName}/versions"},
	{Name: "ListAlarmModels", Method: "GET", URI: "/alarm-models"},
	{Name: "ListDetectorModelVersions", Method: "GET", URI: "/detector-models/{detectorModelName}/versions"},
	{Name: "ListDetectorModels", Method: "GET", URI: "/detector-models"},
	{Name: "ListInputRoutings", Method: "POST", URI: "/input-routings"},
	{Name: "ListInputs", Method: "GET", URI: "/inputs"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags"},
	{Name: "PutLoggingOptions", Method: "PUT", URI: "/logging"},
	{Name: "StartDetectorModelAnalysis", Method: "POST", URI: "/analysis/detector-models/"},
	{Name: "TagResource", Method: "POST", URI: "/tags"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags"},
	{Name: "UpdateAlarmModel", Method: "POST", URI: "/alarm-models/{alarmModelName}"},
	{Name: "UpdateDetectorModel", Method: "POST", URI: "/detector-models/{detectorModelName}"},
	{Name: "UpdateInput", Method: "PUT", URI: "/inputs/{inputName}"},
}

var iotEventsOperationByName = func() map[string]iotEventsOperation {
	out := make(map[string]iotEventsOperation, len(iotEventsOperations))
	for _, op := range iotEventsOperations {
		out[op.Name] = op
	}
	return out
}()
