package server

type braketOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Braket actions sourced from:
// https://docs.aws.amazon.com/braket/latest/APIReference/API_Operations.html
var braketOperations = []braketOperation{
	{Name: "CancelJob", Method: "PUT", URI: "/job/{jobArn}/cancel"},
	{Name: "CancelQuantumTask", Method: "PUT", URI: "/quantum-task/{quantumTaskArn}/cancel"},
	{Name: "CreateJob", Method: "POST", URI: "/job"},
	{Name: "CreateQuantumTask", Method: "POST", URI: "/quantum-task"},
	{Name: "CreateSpendingLimit", Method: "POST", URI: "/spending-limit"},
	{Name: "DeleteSpendingLimit", Method: "DELETE", URI: "/spending-limit/{spendingLimitArn}/delete"},
	{Name: "GetDevice", Method: "GET", URI: "/device/{deviceArn}"},
	{Name: "GetJob", Method: "GET", URI: "/job/{jobArn}?additionalAttributeNames={additionalAttributeNames}"},
	{Name: "GetQuantumTask", Method: "GET", URI: "/quantum-task/{quantumTaskArn}?additionalAttributeNames={additionalAttributeNames}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "SearchDevices", Method: "POST", URI: "/devices"},
	{Name: "SearchJobs", Method: "POST", URI: "/jobs"},
	{Name: "SearchQuantumTasks", Method: "POST", URI: "/quantum-tasks"},
	{Name: "SearchSpendingLimits", Method: "POST", URI: "/spending-limits"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateSpendingLimit", Method: "PATCH", URI: "/spending-limit/{spendingLimitArn}/update"},
}

var braketOperationByName = func() map[string]braketOperation {
	out := make(map[string]braketOperation, len(braketOperations))
	for _, op := range braketOperations {
		out[op.Name] = op
	}
	return out
}()
