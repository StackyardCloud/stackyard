package server

type appFlowOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon AppFlow actions sourced from:
// https://docs.aws.amazon.com/appflow/1.0/APIReference/API_Operations.html
var appFlowOperations = []appFlowOperation{
	{Name: "CancelFlowExecutions", Method: "POST", URI: "/cancel-flow-executions"},
	{Name: "CreateConnectorProfile", Method: "POST", URI: "/create-connector-profile"},
	{Name: "CreateFlow", Method: "POST", URI: "/create-flow"},
	{Name: "DeleteConnectorProfile", Method: "POST", URI: "/delete-connector-profile"},
	{Name: "DeleteFlow", Method: "POST", URI: "/delete-flow"},
	{Name: "DescribeConnector", Method: "POST", URI: "/describe-connector"},
	{Name: "DescribeConnectorEntity", Method: "POST", URI: "/describe-connector-entity"},
	{Name: "DescribeConnectorProfiles", Method: "POST", URI: "/describe-connector-profiles"},
	{Name: "DescribeConnectors", Method: "POST", URI: "/describe-connectors"},
	{Name: "DescribeFlow", Method: "POST", URI: "/describe-flow"},
	{Name: "DescribeFlowExecutionRecords", Method: "POST", URI: "/describe-flow-execution-records"},
	{Name: "ListConnectorEntities", Method: "POST", URI: "/list-connector-entities"},
	{Name: "ListConnectors", Method: "POST", URI: "/list-connectors"},
	{Name: "ListFlows", Method: "POST", URI: "/list-flows"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "RegisterConnector", Method: "POST", URI: "/register-connector"},
	{Name: "ResetConnectorMetadataCache", Method: "POST", URI: "/reset-connector-metadata-cache"},
	{Name: "StartFlow", Method: "POST", URI: "/start-flow"},
	{Name: "StopFlow", Method: "POST", URI: "/stop-flow"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UnregisterConnector", Method: "POST", URI: "/unregister-connector"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateConnectorProfile", Method: "POST", URI: "/update-connector-profile"},
	{Name: "UpdateConnectorRegistration", Method: "POST", URI: "/update-connector-registration"},
	{Name: "UpdateFlow", Method: "POST", URI: "/update-flow"},
}

var appFlowOperationByName = func() map[string]appFlowOperation {
	out := make(map[string]appFlowOperation, len(appFlowOperations))
	for _, op := range appFlowOperations {
		out[op.Name] = op
	}
	return out
}()
