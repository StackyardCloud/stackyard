package server

type appConfigDataType struct {
	Name string
}

// AWS AppConfig data types sourced from:
// https://docs.aws.amazon.com/appconfig/2019-10-09/APIReference/API_Types.html
var appConfigDataTypes = []appConfigDataType{
	{Name: "Action"},
	{Name: "ActionInvocation"},
	{Name: "Application"},
	{Name: "AppliedExtension"},
	{Name: "BadRequestDetails"},
	{Name: "ConfigurationProfileSummary"},
	{Name: "DeletionProtectionSettings"},
	{Name: "DeploymentEvent"},
	{Name: "DeploymentStrategy"},
	{Name: "DeploymentSummary"},
	{Name: "Environment"},
	{Name: "ExtensionAssociationSummary"},
	{Name: "ExtensionSummary"},
	{Name: "HostedConfigurationVersionSummary"},
	{Name: "InvalidConfigurationDetail"},
	{Name: "Monitor"},
	{Name: "Parameter"},
	{Name: "Validator"},
}

var appConfigDataTypeByName = func() map[string]appConfigDataType {
	out := make(map[string]appConfigDataType, len(appConfigDataTypes))
	for _, dt := range appConfigDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
