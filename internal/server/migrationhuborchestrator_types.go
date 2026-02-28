package server

type migrationHubOrchestratorDataType struct {
	Name string
}

// AWS Migration Hub Orchestrator data types sourced from:
// https://docs.aws.amazon.com/migrationhub-orchestrator/latest/APIReference/API_Types.html
var migrationHubOrchestratorDataTypes = []migrationHubOrchestratorDataType{
	{Name: "MigrationWorkflowSummary"},
	{Name: "PlatformCommand"},
	{Name: "PlatformScriptKey"},
	{Name: "PluginSummary"},
	{Name: "StepAutomationConfiguration"},
	{Name: "StepInput"},
	{Name: "StepOutput"},
	{Name: "TemplateInput"},
	{Name: "TemplateSource"},
	{Name: "TemplateStepGroupSummary"},
	{Name: "TemplateStepSummary"},
	{Name: "TemplateSummary"},
	{Name: "Tool"},
	{Name: "WorkflowStepAutomationConfiguration"},
	{Name: "WorkflowStepGroupSummary"},
	{Name: "WorkflowStepOutput"},
	{Name: "WorkflowStepOutputUnion"},
	{Name: "WorkflowStepSummary"},
}

var migrationHubOrchestratorDataTypeByName = func() map[string]migrationHubOrchestratorDataType {
	out := make(map[string]migrationHubOrchestratorDataType, len(migrationHubOrchestratorDataTypes))
	for _, dt := range migrationHubOrchestratorDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
