package server

type launchWizardDataType struct {
	Name string
}

// AWS Launch Wizard data types sourced from:
// https://docs.aws.amazon.com/launchwizard/latest/APIReference/API_Types.html
var launchWizardDataTypes = []launchWizardDataType{
	{Name: "DeploymentConditionalField"},
	{Name: "DeploymentData"},
	{Name: "DeploymentDataSummary"},
	{Name: "DeploymentEventDataSummary"},
	{Name: "DeploymentFilter"},
	{Name: "DeploymentPatternVersionDataSummary"},
	{Name: "DeploymentPatternVersionFilter"},
	{Name: "DeploymentSpecificationsField"},
	{Name: "UpdateDeployment"},
	{Name: "WorkloadData"},
	{Name: "WorkloadDataSummary"},
	{Name: "WorkloadDeploymentPatternData"},
	{Name: "WorkloadDeploymentPatternDataSummary"},
}

var launchWizardDataTypeByName = func() map[string]launchWizardDataType {
	out := make(map[string]launchWizardDataType, len(launchWizardDataTypes))
	for _, t := range launchWizardDataTypes {
		out[t.Name] = t
	}
	return out
}()
