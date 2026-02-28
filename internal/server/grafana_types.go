package server

type grafanaDataType struct {
	Name string
}

// Amazon Managed Grafana data types sourced from:
// https://docs.aws.amazon.com/grafana/latest/APIReference/API_Types.html
var grafanaDataTypes = []grafanaDataType{
	{Name: "AssertionAttributes"},
	{Name: "AuthenticationDescription"},
	{Name: "AuthenticationSummary"},
	{Name: "AwsSsoAuthentication"},
	{Name: "IdpMetadata"},
	{Name: "NetworkAccessConfiguration"},
	{Name: "PermissionEntry"},
	{Name: "RoleValues"},
	{Name: "SamlAuthentication"},
	{Name: "SamlConfiguration"},
	{Name: "ServiceAccountSummary"},
	{Name: "ServiceAccountTokenSummary"},
	{Name: "ServiceAccountTokenSummaryWithKey"},
	{Name: "UpdateError"},
	{Name: "UpdateInstruction"},
	{Name: "UpdateWorkspaceConfiguration"},
	{Name: "User"},
	{Name: "ValidationExceptionField"},
	{Name: "VpcConfiguration"},
	{Name: "WorkspaceDescription"},
	{Name: "WorkspaceSummary"},
}

var grafanaDataTypeByName = func() map[string]grafanaDataType {
	out := make(map[string]grafanaDataType, len(grafanaDataTypes))
	for _, dt := range grafanaDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
