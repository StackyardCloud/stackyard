package server

type securityLakeOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Security Lake operations sourced from:
// https://docs.aws.amazon.com/security-lake/latest/APIReference/API_Operations.html
var securityLakeOperations = []securityLakeOperation{
	{Name: "CreateAwsLogSource", Method: "POST", URI: "/v1/datalake/logsources/aws"},
	{Name: "CreateCustomLogSource", Method: "POST", URI: "/v1/datalake/logsources/custom"},
	{Name: "CreateDataLake", Method: "POST", URI: "/v1/datalake"},
	{Name: "CreateDataLakeExceptionSubscription", Method: "POST", URI: "/v1/datalake/exceptions/subscription"},
	{Name: "CreateDataLakeOrganizationConfiguration", Method: "POST", URI: "/v1/datalake/organization/configuration"},
	{Name: "CreateSubscriber", Method: "POST", URI: "/v1/subscribers"},
	{Name: "CreateSubscriberNotification", Method: "POST", URI: "/v1/subscribers/{subscriberId}/notification"},
	{Name: "DeleteAwsLogSource", Method: "POST", URI: "/v1/datalake/logsources/aws/delete"},
	{Name: "DeleteCustomLogSource", Method: "DELETE", URI: "/v1/datalake/logsources/custom/{sourceName}"},
	{Name: "DeleteDataLake", Method: "POST", URI: "/v1/datalake/delete"},
	{Name: "DeleteDataLakeExceptionSubscription", Method: "DELETE", URI: "/v1/datalake/exceptions/subscription"},
	{Name: "DeleteDataLakeOrganizationConfiguration", Method: "POST", URI: "/v1/datalake/organization/configuration/delete"},
	{Name: "DeleteSubscriber", Method: "DELETE", URI: "/v1/subscribers/{subscriberId}"},
	{Name: "DeleteSubscriberNotification", Method: "DELETE", URI: "/v1/subscribers/{subscriberId}/notification"},
	{Name: "DeregisterDataLakeDelegatedAdministrator", Method: "DELETE", URI: "/v1/datalake/delegate"},
	{Name: "GetDataLakeExceptionSubscription", Method: "GET", URI: "/v1/datalake/exceptions/subscription"},
	{Name: "GetDataLakeOrganizationConfiguration", Method: "GET", URI: "/v1/datalake/organization/configuration"},
	{Name: "GetDataLakeSources", Method: "POST", URI: "/v1/datalake/sources"},
	{Name: "GetSubscriber", Method: "GET", URI: "/v1/subscribers/{subscriberId}"},
	{Name: "ListDataLakeExceptions", Method: "POST", URI: "/v1/datalake/exceptions"},
	{Name: "ListDataLakes", Method: "GET", URI: "/v1/datalakes"},
	{Name: "ListLogSources", Method: "POST", URI: "/v1/datalake/logsources/list"},
	{Name: "ListSubscribers", Method: "GET", URI: "/v1/subscribers"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/v1/tags/{resourceArn}"},
	{Name: "RegisterDataLakeDelegatedAdministrator", Method: "POST", URI: "/v1/datalake/delegate"},
	{Name: "TagResource", Method: "POST", URI: "/v1/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/v1/tags/{resourceArn}"},
	{Name: "UpdateDataLake", Method: "PUT", URI: "/v1/datalake"},
	{Name: "UpdateDataLakeExceptionSubscription", Method: "PUT", URI: "/v1/datalake/exceptions/subscription"},
	{Name: "UpdateSubscriber", Method: "PUT", URI: "/v1/subscribers/{subscriberId}"},
	{Name: "UpdateSubscriberNotification", Method: "PUT", URI: "/v1/subscribers/{subscriberId}/notification"},
}

var securityLakeOperationByName = func() map[string]securityLakeOperation {
	out := make(map[string]securityLakeOperation, len(securityLakeOperations))
	for _, op := range securityLakeOperations {
		out[op.Name] = op
	}
	return out
}()
