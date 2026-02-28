package server

type prometheusOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Managed Service for Prometheus operations sourced from:
// https://docs.aws.amazon.com/prometheus/latest/APIReference/API_Operations.html
var prometheusOperations = []prometheusOperation{
	{Name: "CreateAlertManagerDefinition", Method: "POST", URI: "/workspaces/{workspaceId}/alertmanager/definition"},
	{Name: "CreateAnomalyDetector", Method: "POST", URI: "/workspaces/{workspaceId}/anomalydetectors"},
	{Name: "CreateLoggingConfiguration", Method: "POST", URI: "/workspaces/{workspaceId}/logging"},
	{Name: "CreateQueryLoggingConfiguration", Method: "POST", URI: "/workspaces/{workspaceId}/logging/query"},
	{Name: "CreateRuleGroupsNamespace", Method: "POST", URI: "/workspaces/{workspaceId}/rulegroupsnamespaces"},
	{Name: "CreateScraper", Method: "POST", URI: "/scrapers"},
	{Name: "CreateWorkspace", Method: "POST", URI: "/workspaces"},
	{Name: "DeleteAlertManagerDefinition", Method: "DELETE", URI: "/workspaces/{workspaceId}/alertmanager/definition"},
	{Name: "DeleteAnomalyDetector", Method: "DELETE", URI: "/workspaces/{workspaceId}/anomalydetectors/{anomalyDetectorId}"},
	{Name: "DeleteLoggingConfiguration", Method: "DELETE", URI: "/workspaces/{workspaceId}/logging"},
	{Name: "DeleteQueryLoggingConfiguration", Method: "DELETE", URI: "/workspaces/{workspaceId}/logging/query"},
	{Name: "DeleteResourcePolicy", Method: "DELETE", URI: "/workspaces/{workspaceId}/policy"},
	{Name: "DeleteRuleGroupsNamespace", Method: "DELETE", URI: "/workspaces/{workspaceId}/rulegroupsnamespaces/{name}"},
	{Name: "DeleteScraper", Method: "DELETE", URI: "/scrapers/{scraperId}"},
	{Name: "DeleteScraperLoggingConfiguration", Method: "DELETE", URI: "/scrapers/{scraperId}/logging-configuration"},
	{Name: "DeleteWorkspace", Method: "DELETE", URI: "/workspaces/{workspaceId}"},
	{Name: "DescribeAlertManagerDefinition", Method: "GET", URI: "/workspaces/{workspaceId}/alertmanager/definition"},
	{Name: "DescribeAnomalyDetector", Method: "GET", URI: "/workspaces/{workspaceId}/anomalydetectors/{anomalyDetectorId}"},
	{Name: "DescribeLoggingConfiguration", Method: "GET", URI: "/workspaces/{workspaceId}/logging"},
	{Name: "DescribeQueryLoggingConfiguration", Method: "GET", URI: "/workspaces/{workspaceId}/logging/query"},
	{Name: "DescribeResourcePolicy", Method: "GET", URI: "/workspaces/{workspaceId}/policy"},
	{Name: "DescribeRuleGroupsNamespace", Method: "GET", URI: "/workspaces/{workspaceId}/rulegroupsnamespaces/{name}"},
	{Name: "DescribeScraper", Method: "GET", URI: "/scrapers/{scraperId}"},
	{Name: "DescribeScraperLoggingConfiguration", Method: "GET", URI: "/scrapers/{scraperId}/logging-configuration"},
	{Name: "DescribeWorkspace", Method: "GET", URI: "/workspaces/{workspaceId}"},
	{Name: "DescribeWorkspaceConfiguration", Method: "GET", URI: "/workspaces/{workspaceId}/configuration"},
	{Name: "GetDefaultScraperConfiguration", Method: "GET", URI: "/scraperconfiguration"},
	{Name: "ListAnomalyDetectors", Method: "GET", URI: "/workspaces/{workspaceId}/anomalydetectors"},
	{Name: "ListRuleGroupsNamespaces", Method: "GET", URI: "/workspaces/{workspaceId}/rulegroupsnamespaces"},
	{Name: "ListScrapers", Method: "GET", URI: "/scrapers"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListWorkspaces", Method: "GET", URI: "/workspaces"},
	{Name: "PutAlertManagerDefinition", Method: "PUT", URI: "/workspaces/{workspaceId}/alertmanager/definition"},
	{Name: "PutAnomalyDetector", Method: "PUT", URI: "/workspaces/{workspaceId}/anomalydetectors/{anomalyDetectorId}"},
	{Name: "PutResourcePolicy", Method: "PUT", URI: "/workspaces/{workspaceId}/policy"},
	{Name: "PutRuleGroupsNamespace", Method: "PUT", URI: "/workspaces/{workspaceId}/rulegroupsnamespaces/{name}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateLoggingConfiguration", Method: "PUT", URI: "/workspaces/{workspaceId}/logging"},
	{Name: "UpdateQueryLoggingConfiguration", Method: "PUT", URI: "/workspaces/{workspaceId}/logging/query"},
	{Name: "UpdateScraper", Method: "PUT", URI: "/scrapers/{scraperId}"},
	{Name: "UpdateScraperLoggingConfiguration", Method: "PUT", URI: "/scrapers/{scraperId}/logging-configuration"},
	{Name: "UpdateWorkspaceAlias", Method: "POST", URI: "/workspaces/{workspaceId}/alias"},
	{Name: "UpdateWorkspaceConfiguration", Method: "PATCH", URI: "/workspaces/{workspaceId}/configuration"},
}

var prometheusOperationByName = func() map[string]prometheusOperation {
	out := make(map[string]prometheusOperation, len(prometheusOperations))
	for _, op := range prometheusOperations {
		out[op.Name] = op
	}
	return out
}()
