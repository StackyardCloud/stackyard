package server

type apiGatewayManagementOperation struct {
	Name   string
	Method string
}

// Amazon API Gateway Management API operations sourced from:
// https://docs.aws.amazon.com/apigateway/latest/api/API_Operations_Amazon_API_Gateway_Management_API.html
var apiGatewayManagementOperations = []apiGatewayManagementOperation{
	{Name: "DeleteConnection", Method: "DELETE"},
	{Name: "GetConnection", Method: "GET"},
	{Name: "PostToConnection", Method: "POST"},
}

var apiGatewayManagementOperationByName = func() map[string]apiGatewayManagementOperation {
	out := make(map[string]apiGatewayManagementOperation, len(apiGatewayManagementOperations))
	for _, op := range apiGatewayManagementOperations {
		out[op.Name] = op
	}
	return out
}()

var apiGatewayManagementOperationByMethod = func() map[string]string {
	out := make(map[string]string, len(apiGatewayManagementOperations))
	for _, op := range apiGatewayManagementOperations {
		out[op.Method] = op.Name
	}
	return out
}()
