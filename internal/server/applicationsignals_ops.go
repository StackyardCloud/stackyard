package server

type applicationSignalsOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Application Signals operations sourced from:
// https://docs.aws.amazon.com/applicationsignals/latest/APIReference/API_Operations.html
var applicationSignalsOperations = []applicationSignalsOperation{
	{Name: "BatchGetServiceLevelObjectiveBudgetReport", Method: "POST", URI: "/budget-report"},
	{Name: "BatchUpdateExclusionWindows", Method: "PATCH", URI: "/exclusion-windows"},
	{Name: "CreateServiceLevelObjective", Method: "POST", URI: "/slo"},
	{Name: "DeleteGroupingConfiguration", Method: "DELETE", URI: "/grouping-configuration"},
	{Name: "DeleteServiceLevelObjective", Method: "DELETE", URI: "/slo/{Id}"},
	{Name: "GetService", Method: "POST", URI: "/service"},
	{Name: "GetServiceLevelObjective", Method: "GET", URI: "/slo/{Id}"},
	{Name: "ListAuditFindings", Method: "POST", URI: "/auditFindings"},
	{Name: "ListEntityEvents", Method: "POST", URI: "/events"},
	{Name: "ListGroupingAttributeDefinitions", Method: "POST", URI: "/grouping-attribute-definitions"},
	{Name: "ListServiceDependencies", Method: "POST", URI: "/service-dependencies"},
	{Name: "ListServiceDependents", Method: "POST", URI: "/service-dependents"},
	{Name: "ListServiceLevelObjectiveExclusionWindows", Method: "GET", URI: "/slo/{Id}/exclusion-windows"},
	{Name: "ListServiceLevelObjectives", Method: "POST", URI: "/slos"},
	{Name: "ListServiceOperations", Method: "POST", URI: "/service-operations"},
	{Name: "ListServices", Method: "GET", URI: "/services"},
	{Name: "ListServiceStates", Method: "POST", URI: "/service/states"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags"},
	{Name: "PutGroupingConfiguration", Method: "PUT", URI: "/grouping-configuration"},
	{Name: "StartDiscovery", Method: "POST", URI: "/start-discovery"},
	{Name: "TagResource", Method: "POST", URI: "/tag-resource"},
	{Name: "UntagResource", Method: "POST", URI: "/untag-resource"},
	{Name: "UpdateServiceLevelObjective", Method: "PATCH", URI: "/slo/{Id}"},
}

var applicationSignalsOperationByName = func() map[string]applicationSignalsOperation {
	out := make(map[string]applicationSignalsOperation, len(applicationSignalsOperations))
	for _, op := range applicationSignalsOperations {
		out[op.Name] = op
	}
	return out
}()
