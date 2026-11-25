package server

type iotDeviceAdvisorOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS IoT Core Device Advisor operations sourced from:
// https://docs.aws.amazon.com/iot/latest/apireference/API_Operations_AWS_IoT_Core_Device_Advisor.html
var iotDeviceAdvisorOperations = []iotDeviceAdvisorOperation{
	{Name: "CreateSuiteDefinition", Method: "POST", URI: "/suiteDefinitions"},
	{Name: "DeleteSuiteDefinition", Method: "DELETE", URI: "/suiteDefinitions/{suiteDefinitionId}"},
	{Name: "GetEndpoint", Method: "GET", URI: "/endpoint"},
	{Name: "GetSuiteDefinition", Method: "GET", URI: "/suiteDefinitions/{suiteDefinitionId}"},
	{Name: "GetSuiteRun", Method: "GET", URI: "/suiteDefinitions/{suiteDefinitionId}/suiteRuns/{suiteRunId}"},
	{Name: "GetSuiteRunReport", Method: "GET", URI: "/suiteDefinitions/{suiteDefinitionId}/suiteRuns/{suiteRunId}/report"},
	{Name: "ListSuiteDefinitions", Method: "GET", URI: "/suiteDefinitions"},
	{Name: "ListSuiteRuns", Method: "GET", URI: "/suiteRuns"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "StartSuiteRun", Method: "POST", URI: "/suiteDefinitions/{suiteDefinitionId}/suiteRuns"},
	{Name: "StopSuiteRun", Method: "POST", URI: "/suiteDefinitions/{suiteDefinitionId}/suiteRuns/{suiteRunId}/stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateSuiteDefinition", Method: "PATCH", URI: "/suiteDefinitions/{suiteDefinitionId}"},
}

var iotDeviceAdvisorOperationByName = func() map[string]iotDeviceAdvisorOperation {
	out := make(map[string]iotDeviceAdvisorOperation, len(iotDeviceAdvisorOperations))
	for _, op := range iotDeviceAdvisorOperations {
		out[op.Name] = op
	}
	return out
}()
