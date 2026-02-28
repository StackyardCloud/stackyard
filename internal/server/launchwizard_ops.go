package server

type launchWizardOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Launch Wizard actions sourced from:
// https://docs.aws.amazon.com/launchwizard/latest/APIReference/API_Operations.html
var launchWizardOperations = []launchWizardOperation{
	{Name: "CreateDeployment", Method: "POST", URI: "/createDeployment"},
	{Name: "DeleteDeployment", Method: "POST", URI: "/deleteDeployment"},
	{Name: "GetDeployment", Method: "POST", URI: "/getDeployment"},
	{Name: "GetDeploymentPatternVersion", Method: "POST", URI: "/getDeploymentPatternVersion"},
	{Name: "GetWorkload", Method: "POST", URI: "/getWorkload"},
	{Name: "GetWorkloadDeploymentPattern", Method: "POST", URI: "/getWorkloadDeploymentPattern"},
	{Name: "ListDeploymentEvents", Method: "POST", URI: "/listDeploymentEvents"},
	{Name: "ListDeploymentPatternVersions", Method: "POST", URI: "/listDeploymentPatternVersions"},
	{Name: "ListDeployments", Method: "POST", URI: "/listDeployments"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/"},
	{Name: "ListWorkloadDeploymentPatterns", Method: "POST", URI: "/listWorkloadDeploymentPatterns"},
	{Name: "ListWorkloads", Method: "POST", URI: "/listWorkloads"},
	{Name: "TagResource", Method: "POST", URI: "/tags/"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/"},
	{Name: "UpdateDeployment", Method: "POST", URI: "/updateDeployment"},
}

var launchWizardOperationByName = func() map[string]launchWizardOperation {
	out := make(map[string]launchWizardOperation, len(launchWizardOperations))
	for _, op := range launchWizardOperations {
		out[op.Name] = op
	}
	return out
}()
