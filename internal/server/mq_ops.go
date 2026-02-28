package server

type mqOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon MQ operations sourced from:
// https://docs.aws.amazon.com/amazon-mq/latest/api-reference/resources.html
var mqOperations = []mqOperation{
	{Name: "CreateBroker", Method: "POST", URI: "/v1/brokers"},
	{Name: "CreateConfiguration", Method: "POST", URI: "/v1/configurations"},
	{Name: "CreateTags", Method: "POST", URI: "/v1/tags/{resource-arn}"},
	{Name: "CreateUser", Method: "POST", URI: "/v1/brokers/{broker-id}/users/{username}"},
	{Name: "DeleteBroker", Method: "DELETE", URI: "/v1/brokers/{broker-id}"},
	{Name: "DeleteConfiguration", Method: "DELETE", URI: "/v1/configurations/{configuration-id}"},
	{Name: "DeleteTags", Method: "DELETE", URI: "/v1/tags/{resource-arn}"},
	{Name: "DeleteUser", Method: "DELETE", URI: "/v1/brokers/{broker-id}/users/{username}"},
	{Name: "DescribeBroker", Method: "GET", URI: "/v1/brokers/{broker-id}"},
	{Name: "DescribeBrokerEngineTypes", Method: "GET", URI: "/v1/broker-engine-types"},
	{Name: "DescribeBrokerInstanceOptions", Method: "GET", URI: "/v1/broker-instance-options"},
	{Name: "DescribeConfiguration", Method: "GET", URI: "/v1/configurations/{configuration-id}"},
	{Name: "DescribeConfigurationRevision", Method: "GET", URI: "/v1/configurations/{configuration-id}/revisions/{configuration-revision}"},
	{Name: "DescribeUser", Method: "GET", URI: "/v1/brokers/{broker-id}/users/{username}"},
	{Name: "ListBrokers", Method: "GET", URI: "/v1/brokers"},
	{Name: "ListConfigurationRevisions", Method: "GET", URI: "/v1/configurations/{configuration-id}/revisions"},
	{Name: "ListConfigurations", Method: "GET", URI: "/v1/configurations"},
	{Name: "ListTags", Method: "GET", URI: "/v1/tags/{resource-arn}"},
	{Name: "ListUsers", Method: "GET", URI: "/v1/brokers/{broker-id}/users"},
	{Name: "Promote", Method: "POST", URI: "/v1/brokers/{broker-id}/promote"},
	{Name: "RebootBroker", Method: "POST", URI: "/v1/brokers/{broker-id}/reboot"},
	{Name: "UpdateBroker", Method: "PUT", URI: "/v1/brokers/{broker-id}"},
	{Name: "UpdateConfiguration", Method: "PUT", URI: "/v1/configurations/{configuration-id}"},
	{Name: "UpdateUser", Method: "PUT", URI: "/v1/brokers/{broker-id}/users/{username}"},
}

var mqOperationByName = func() map[string]mqOperation {
	out := make(map[string]mqOperation, len(mqOperations))
	for _, op := range mqOperations {
		out[op.Name] = op
	}
	return out
}()
