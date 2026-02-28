package server

type quickSetupDataType struct {
	Name string
}

// AWS Systems Manager Quick Setup data types sourced from:
// https://docs.aws.amazon.com/quick-setup/latest/APIReference/API_Types.html
var quickSetupDataTypes = []quickSetupDataType{
	{Name: "ConfigurationDefinition"},
	{Name: "ConfigurationDefinitionInput"},
	{Name: "ConfigurationDefinitionSummary"},
	{Name: "ConfigurationManagerSummary"},
	{Name: "ConfigurationSummary"},
	{Name: "Filter"},
	{Name: "QuickSetupTypeOutput"},
	{Name: "ServiceSettings"},
	{Name: "StatusSummary"},
	{Name: "TagEntry"},
}

var quickSetupDataTypeByName = func() map[string]quickSetupDataType {
	out := make(map[string]quickSetupDataType, len(quickSetupDataTypes))
	for _, dt := range quickSetupDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
