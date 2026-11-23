package server

type pcaConnectorADOperation struct {
	Name   string
	Method string
	URI    string
	Status int
}

// Private CA Connector for Active Directory operations sourced from:
// https://docs.aws.amazon.com/pca-connector-ad/latest/APIReference/API_Operations.html
var pcaConnectorADOperations = []pcaConnectorADOperation{
	{Name: "CreateConnector", Method: "POST", URI: "/connectors", Status: 202},
	{Name: "CreateDirectoryRegistration", Method: "POST", URI: "/directoryRegistrations", Status: 202},
	{Name: "CreateServicePrincipalName", Method: "POST", URI: "/directoryRegistrations/{DirectoryRegistrationArn}/servicePrincipalNames/{ConnectorArn}", Status: 202},
	{Name: "CreateTemplate", Method: "POST", URI: "/templates", Status: 200},
	{Name: "CreateTemplateGroupAccessControlEntry", Method: "POST", URI: "/templates/{TemplateArn}/accessControlEntries", Status: 200},
	{Name: "DeleteConnector", Method: "DELETE", URI: "/connectors/{ConnectorArn}", Status: 202},
	{Name: "DeleteDirectoryRegistration", Method: "DELETE", URI: "/directoryRegistrations/{DirectoryRegistrationArn}", Status: 202},
	{Name: "DeleteServicePrincipalName", Method: "DELETE", URI: "/directoryRegistrations/{DirectoryRegistrationArn}/servicePrincipalNames/{ConnectorArn}", Status: 202},
	{Name: "DeleteTemplate", Method: "DELETE", URI: "/templates/{TemplateArn}", Status: 202},
	{Name: "DeleteTemplateGroupAccessControlEntry", Method: "DELETE", URI: "/templates/{TemplateArn}/accessControlEntries/{GroupSecurityIdentifier}", Status: 200},
	{Name: "GetConnector", Method: "GET", URI: "/connectors/{ConnectorArn}", Status: 200},
	{Name: "GetDirectoryRegistration", Method: "GET", URI: "/directoryRegistrations/{DirectoryRegistrationArn}", Status: 200},
	{Name: "GetServicePrincipalName", Method: "GET", URI: "/directoryRegistrations/{DirectoryRegistrationArn}/servicePrincipalNames/{ConnectorArn}", Status: 200},
	{Name: "GetTemplate", Method: "GET", URI: "/templates/{TemplateArn}", Status: 200},
	{Name: "GetTemplateGroupAccessControlEntry", Method: "GET", URI: "/templates/{TemplateArn}/accessControlEntries/{GroupSecurityIdentifier}", Status: 200},
	{Name: "ListConnectors", Method: "GET", URI: "/connectors", Status: 200},
	{Name: "ListDirectoryRegistrations", Method: "GET", URI: "/directoryRegistrations", Status: 200},
	{Name: "ListServicePrincipalNames", Method: "GET", URI: "/directoryRegistrations/{DirectoryRegistrationArn}/servicePrincipalNames", Status: 200},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}", Status: 200},
	{Name: "ListTemplateGroupAccessControlEntries", Method: "GET", URI: "/templates/{TemplateArn}/accessControlEntries", Status: 200},
	{Name: "ListTemplates", Method: "GET", URI: "/templates", Status: 200},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}", Status: 204},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}", Status: 204},
	{Name: "UpdateTemplate", Method: "PATCH", URI: "/templates/{TemplateArn}", Status: 200},
	{Name: "UpdateTemplateGroupAccessControlEntry", Method: "PATCH", URI: "/templates/{TemplateArn}/accessControlEntries/{GroupSecurityIdentifier}", Status: 200},
}

var pcaConnectorADOperationByName = func() map[string]pcaConnectorADOperation {
	out := make(map[string]pcaConnectorADOperation, len(pcaConnectorADOperations))
	for _, op := range pcaConnectorADOperations {
		out[op.Name] = op
	}
	return out
}()
