package server

type mskConnectOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon MSK Connect actions sourced from:
// https://docs.aws.amazon.com/MSKC/latest/mskc/API_Operations.html
var mskConnectOperations = []mskConnectOperation{
	{Name: "CreateConnector", Method: "POST", URI: "/v1/connectors"},
	{Name: "CreateCustomPlugin", Method: "POST", URI: "/v1/custom-plugins"},
	{Name: "CreateWorkerConfiguration", Method: "POST", URI: "/v1/worker-configurations"},
	{Name: "DeleteConnector", Method: "DELETE", URI: "/v1/connectors/{connectorArn}?currentVersion={currentVersion}"},
	{Name: "DeleteCustomPlugin", Method: "DELETE", URI: "/v1/custom-plugins/{customPluginArn}"},
	{Name: "DeleteWorkerConfiguration", Method: "DELETE", URI: "/v1/worker-configurations/{workerConfigurationArn}"},
	{Name: "DescribeConnector", Method: "GET", URI: "/v1/connectors/{connectorArn}"},
	{Name: "DescribeConnectorOperation", Method: "GET", URI: "/v1/connectorOperations/{connectorOperationArn}"},
	{Name: "DescribeCustomPlugin", Method: "GET", URI: "/v1/custom-plugins/{customPluginArn}"},
	{Name: "DescribeWorkerConfiguration", Method: "GET", URI: "/v1/worker-configurations/{workerConfigurationArn}"},
	{Name: "ListConnectorOperations", Method: "GET", URI: "/v1/connectors/{connectorArn}/operations?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListConnectors", Method: "GET", URI: "/v1/connectors?connectorNamePrefix={connectorNamePrefix}&maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListCustomPlugins", Method: "GET", URI: "/v1/custom-plugins?maxResults={maxResults}&namePrefix={namePrefix}&nextToken={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/v1/tags/{resourceArn}"},
	{Name: "ListWorkerConfigurations", Method: "GET", URI: "/v1/worker-configurations?maxResults={maxResults}&namePrefix={namePrefix}&nextToken={nextToken}"},
	{Name: "TagResource", Method: "POST", URI: "/v1/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/v1/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateConnector", Method: "PUT", URI: "/v1/connectors/{connectorArn}?currentVersion={currentVersion}"},
}

var mskConnectOperationByName = func() map[string]mskConnectOperation {
	out := make(map[string]mskConnectOperation, len(mskConnectOperations))
	for _, op := range mskConnectOperations {
		out[op.Name] = op
	}
	return out
}()
