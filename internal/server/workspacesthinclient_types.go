package server

type workspacesThinClientType struct {
	Name string
}

// Amazon WorkSpaces Thin Client data types sourced from:

// https://docs.aws.amazon.com/workspaces-thin-client/latest/api/API_Types.html

var workspacesThinClientTypes = []workspacesThinClientType{
	{Name: "Device"},
	{Name: "DeviceSummary"},
	{Name: "Environment"},
	{Name: "EnvironmentSummary"},
	{Name: "MaintenanceWindow"},
	{Name: "Software"},
	{Name: "SoftwareSet"},
	{Name: "SoftwareSetSummary"},
	{Name: "ValidationExceptionField"},
}

var workspacesThinClientTypeByName = func() map[string]workspacesThinClientType {
	out := make(map[string]workspacesThinClientType, len(workspacesThinClientTypes))
	for _, typ := range workspacesThinClientTypes {
		out[typ.Name] = typ
	}
	return out
}()
