package server

type novaActResource struct {
	Name string
}

// Amazon Nova Act API types sourced from:
// https://docs.aws.amazon.com/nova-act/latest/APIReference/API_Types.html
var novaActResources = []novaActResource{
	{Name: "ActError"},
	{Name: "ActSummary"},
	{Name: "Call"},
	{Name: "CallResult"},
	{Name: "CallResultContent"},
	{Name: "ClientInfo"},
	{Name: "CompatibilityInformation"},
	{Name: "ModelAlias"},
	{Name: "ModelLifecycle"},
	{Name: "ModelSummary"},
	{Name: "SessionSummary"},
	{Name: "ToolInputSchema"},
	{Name: "ToolSpec"},
	{Name: "TraceLocation"},
	{Name: "ValidationExceptionField"},
	{Name: "WorkflowDefinitionSummary"},
	{Name: "WorkflowExportConfig"},
	{Name: "WorkflowRunSummary"},
}

var novaActResourceByName = func() map[string]novaActResource {
	out := make(map[string]novaActResource, len(novaActResources))
	for _, r := range novaActResources {
		out[r.Name] = r
	}
	return out
}()
