package server

type quickSetupOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Systems Manager Quick Setup operations sourced from:
// https://docs.aws.amazon.com/quick-setup/latest/APIReference/API_Operations.html
var quickSetupOperations = []quickSetupOperation{
	{Name: "CreateConfigurationManager", Method: "POST", URI: "/configurationManager"},
	{Name: "DeleteConfigurationManager", Method: "DELETE", URI: "/configurationManager/{ManagerArn}"},
	{Name: "GetConfiguration", Method: "GET", URI: "/getConfiguration/{ConfigurationId}"},
	{Name: "GetConfigurationManager", Method: "GET", URI: "/configurationManager/{ManagerArn}"},
	{Name: "GetServiceSettings", Method: "GET", URI: "/serviceSettings"},
	{Name: "ListConfigurationManagers", Method: "POST", URI: "/listConfigurationManagers"},
	{Name: "ListConfigurations", Method: "POST", URI: "/listConfigurations"},
	{Name: "ListQuickSetupTypes", Method: "GET", URI: "/listQuickSetupTypes"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "TagResource", Method: "PUT", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateConfigurationDefinition", Method: "PUT", URI: "/configurationDefinition/{ManagerArn}/{Id}"},
	{Name: "UpdateConfigurationManager", Method: "PUT", URI: "/configurationManager/{ManagerArn}"},
	{Name: "UpdateServiceSettings", Method: "PUT", URI: "/serviceSettings"},
}

var quickSetupOperationByName = func() map[string]quickSetupOperation {
	out := make(map[string]quickSetupOperation, len(quickSetupOperations))
	for _, op := range quickSetupOperations {
		out[op.Name] = op
	}
	return out
}()
