package server

type ssmSAPOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Systems Manager for SAP operations sourced from:
// https://docs.aws.amazon.com/ssmsap/latest/APIReference/API_Operations.html
var ssmSAPOperations = []ssmSAPOperation{
	{Name: "DeleteResourcePermission", Method: "POST", URI: "/delete-resource-permission"},
	{Name: "DeregisterApplication", Method: "POST", URI: "/deregister-application"},
	{Name: "GetApplication", Method: "POST", URI: "/get-application"},
	{Name: "GetComponent", Method: "POST", URI: "/get-component"},
	{Name: "GetConfigurationCheckOperation", Method: "POST", URI: "/get-configuration-check-operation"},
	{Name: "GetDatabase", Method: "POST", URI: "/get-database"},
	{Name: "GetOperation", Method: "POST", URI: "/get-operation"},
	{Name: "GetResourcePermission", Method: "POST", URI: "/get-resource-permission"},
	{Name: "ListApplications", Method: "POST", URI: "/list-applications"},
	{Name: "ListComponents", Method: "POST", URI: "/list-components"},
	{Name: "ListConfigurationCheckDefinitions", Method: "POST", URI: "/list-configuration-check-definitions"},
	{Name: "ListConfigurationCheckOperations", Method: "POST", URI: "/list-configuration-check-operations"},
	{Name: "ListDatabases", Method: "POST", URI: "/list-databases"},
	{Name: "ListOperationEvents", Method: "POST", URI: "/list-operation-events"},
	{Name: "ListOperations", Method: "POST", URI: "/list-operations"},
	{Name: "ListSubCheckResults", Method: "POST", URI: "/list-sub-check-results"},
	{Name: "ListSubCheckRuleResults", Method: "POST", URI: "/list-sub-check-rule-results"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "PutResourcePermission", Method: "POST", URI: "/put-resource-permission"},
	{Name: "RegisterApplication", Method: "POST", URI: "/register-application"},
	{Name: "StartApplication", Method: "POST", URI: "/start-application"},
	{Name: "StartApplicationRefresh", Method: "POST", URI: "/start-application-refresh"},
	{Name: "StartConfigurationChecks", Method: "POST", URI: "/start-configuration-checks"},
	{Name: "StopApplication", Method: "POST", URI: "/stop-application"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateApplicationSettings", Method: "POST", URI: "/update-application-settings"},
}

var ssmSAPOperationByName = func() map[string]ssmSAPOperation {
	out := make(map[string]ssmSAPOperation, len(ssmSAPOperations))
	for _, op := range ssmSAPOperations {
		out[op.Name] = op
	}
	return out
}()
