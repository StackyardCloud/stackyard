package server

type appConfigOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS AppConfig operations sourced from:
// https://docs.aws.amazon.com/appconfig/2019-10-09/APIReference/API_Operations.html
var appConfigOperations = []appConfigOperation{
	{Name: "CreateApplication", Method: "POST", URI: "/applications"},
	{Name: "CreateConfigurationProfile", Method: "POST", URI: "/applications/{ApplicationId}/configurationprofiles"},
	{Name: "CreateDeploymentStrategy", Method: "POST", URI: "/deploymentstrategies"},
	{Name: "CreateEnvironment", Method: "POST", URI: "/applications/{ApplicationId}/environments"},
	{Name: "CreateExtension", Method: "POST", URI: "/extensions"},
	{Name: "CreateExtensionAssociation", Method: "POST", URI: "/extensionassociations"},
	{Name: "CreateHostedConfigurationVersion", Method: "POST", URI: "/applications/{ApplicationId}/configurationprofiles/{ConfigurationProfileId}/hostedconfigurationversions"},
	{Name: "DeleteApplication", Method: "DELETE", URI: "/applications/{ApplicationId}"},
	{Name: "DeleteConfigurationProfile", Method: "DELETE", URI: "/applications/{ApplicationId}/configurationprofiles/{ConfigurationProfileId}"},
	{Name: "DeleteDeploymentStrategy", Method: "DELETE", URI: "/deployementstrategies/{DeploymentStrategyId}"},
	{Name: "DeleteEnvironment", Method: "DELETE", URI: "/applications/{ApplicationId}/environments/{EnvironmentId}"},
	{Name: "DeleteExtension", Method: "DELETE", URI: "/extensions/{ExtensionIdentifier}"},
	{Name: "DeleteExtensionAssociation", Method: "DELETE", URI: "/extensionassociations/{ExtensionAssociationId}"},
	{Name: "DeleteHostedConfigurationVersion", Method: "DELETE", URI: "/applications/{ApplicationId}/configurationprofiles/{ConfigurationProfileId}/hostedconfigurationversions/{VersionNumber}"},
	{Name: "GetAccountSettings", Method: "GET", URI: "/settings"},
	{Name: "GetApplication", Method: "GET", URI: "/applications/{ApplicationId}"},
	{Name: "GetConfiguration", Method: "GET", URI: "/applications/{Application}/environments/{Environment}/configurations/{Configuration}"},
	{Name: "GetConfigurationProfile", Method: "GET", URI: "/applications/{ApplicationId}/configurationprofiles/{ConfigurationProfileId}"},
	{Name: "GetDeployment", Method: "GET", URI: "/applications/{ApplicationId}/environments/{EnvironmentId}/deployments/{DeploymentNumber}"},
	{Name: "GetDeploymentStrategy", Method: "GET", URI: "/deploymentstrategies/{DeploymentStrategyId}"},
	{Name: "GetEnvironment", Method: "GET", URI: "/applications/{ApplicationId}/environments/{EnvironmentId}"},
	{Name: "GetExtension", Method: "GET", URI: "/extensions/{ExtensionIdentifier}"},
	{Name: "GetExtensionAssociation", Method: "GET", URI: "/extensionassociations/{ExtensionAssociationId}"},
	{Name: "GetHostedConfigurationVersion", Method: "GET", URI: "/applications/{ApplicationId}/configurationprofiles/{ConfigurationProfileId}/hostedconfigurationversions/{VersionNumber}"},
	{Name: "ListApplications", Method: "GET", URI: "/applications"},
	{Name: "ListConfigurationProfiles", Method: "GET", URI: "/applications/{ApplicationId}/configurationprofiles"},
	{Name: "ListDeploymentStrategies", Method: "GET", URI: "/deploymentstrategies"},
	{Name: "ListDeployments", Method: "GET", URI: "/applications/{ApplicationId}/environments/{EnvironmentId}/deployments"},
	{Name: "ListEnvironments", Method: "GET", URI: "/applications/{ApplicationId}/environments"},
	{Name: "ListExtensionAssociations", Method: "GET", URI: "/extensionassociations"},
	{Name: "ListExtensions", Method: "GET", URI: "/extensions"},
	{Name: "ListHostedConfigurationVersions", Method: "GET", URI: "/applications/{ApplicationId}/configurationprofiles/{ConfigurationProfileId}/hostedconfigurationversions"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "StartDeployment", Method: "POST", URI: "/applications/{ApplicationId}/environments/{EnvironmentId}/deployments"},
	{Name: "StopDeployment", Method: "DELETE", URI: "/applications/{ApplicationId}/environments/{EnvironmentId}/deployments/{DeploymentNumber}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateAccountSettings", Method: "PATCH", URI: "/settings"},
	{Name: "UpdateApplication", Method: "PATCH", URI: "/applications/{ApplicationId}"},
	{Name: "UpdateConfigurationProfile", Method: "PATCH", URI: "/applications/{ApplicationId}/configurationprofiles/{ConfigurationProfileId}"},
	{Name: "UpdateDeploymentStrategy", Method: "PATCH", URI: "/deploymentstrategies/{DeploymentStrategyId}"},
	{Name: "UpdateEnvironment", Method: "PATCH", URI: "/applications/{ApplicationId}/environments/{EnvironmentId}"},
	{Name: "UpdateExtension", Method: "PATCH", URI: "/extensions/{ExtensionIdentifier}"},
	{Name: "UpdateExtensionAssociation", Method: "PATCH", URI: "/extensionassociations/{ExtensionAssociationId}"},
	{Name: "ValidateConfiguration", Method: "POST", URI: "/applications/{ApplicationId}/configurationprofiles/{ConfigurationProfileId}/validators"},
}

var appConfigOperationByName = func() map[string]appConfigOperation {
	out := make(map[string]appConfigOperation, len(appConfigOperations))
	for _, op := range appConfigOperations {
		out[op.Name] = op
	}
	return out
}()
