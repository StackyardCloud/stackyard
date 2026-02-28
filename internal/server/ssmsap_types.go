package server

type ssmSAPDataType struct {
	Name string
}

// AWS Systems Manager for SAP data types sourced from:
// https://docs.aws.amazon.com/ssmsap/latest/APIReference/API_Types.html
var ssmSAPDataTypes = []ssmSAPDataType{
	{Name: "Application"},
	{Name: "ApplicationCredential"},
	{Name: "ApplicationSummary"},
	{Name: "AssociatedHost"},
	{Name: "BackintConfig"},
	{Name: "Component"},
	{Name: "ComponentInfo"},
	{Name: "ComponentSummary"},
	{Name: "ConfigurationCheckDefinition"},
	{Name: "ConfigurationCheckOperation"},
	{Name: "Database"},
	{Name: "DatabaseConnection"},
	{Name: "DatabaseSummary"},
	{Name: "Filter"},
	{Name: "Host"},
	{Name: "IpAddressMember"},
	{Name: "Operation"},
	{Name: "OperationEvent"},
	{Name: "Resilience"},
	{Name: "Resource"},
	{Name: "RuleResult"},
	{Name: "RuleStatusCounts"},
	{Name: "SubCheckResult"},
}

var ssmSAPDataTypeByName = func() map[string]ssmSAPDataType {
	out := make(map[string]ssmSAPDataType, len(ssmSAPDataTypes))
	for _, dt := range ssmSAPDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
