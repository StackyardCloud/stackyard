package server

type codeCatalystType struct {
	Name string
}

// Amazon CodeCatalyst data types sourced from:
// https://docs.aws.amazon.com/codecatalyst/latest/APIReference/API_Types.html
var codeCatalystTypes = []codeCatalystType{
	{Name: "AccessTokenSummary"},
	{Name: "DevEnvironmentAccessDetails"},
	{Name: "DevEnvironmentRepositorySummary"},
	{Name: "DevEnvironmentSessionConfiguration"},
	{Name: "DevEnvironmentSessionSummary"},
	{Name: "DevEnvironmentSummary"},
	{Name: "EmailAddress"},
	{Name: "EventLogEntry"},
	{Name: "EventPayload"},
	{Name: "ExecuteCommandSessionConfiguration"},
	{Name: "Filter"},
	{Name: "Ide"},
	{Name: "IdeConfiguration"},
	{Name: "ListSourceRepositoriesItem"},
	{Name: "ListSourceRepositoryBranchesItem"},
	{Name: "PersistentStorage"},
	{Name: "PersistentStorageConfiguration"},
	{Name: "ProjectInformation"},
	{Name: "ProjectListFilter"},
	{Name: "ProjectSummary"},
	{Name: "RepositoryInput"},
	{Name: "SpaceSummary"},
	{Name: "UserIdentity"},
	{Name: "VerifySession"},
	{Name: "WorkflowDefinition"},
	{Name: "WorkflowDefinitionSummary"},
	{Name: "WorkflowRunSortCriteria"},
	{Name: "WorkflowRunStatusReason"},
	{Name: "WorkflowRunSummary"},
	{Name: "WorkflowSortCriteria"},
	{Name: "WorkflowSummary"},
}

var codeCatalystTypeByName = func() map[string]codeCatalystType {
	out := make(map[string]codeCatalystType, len(codeCatalystTypes))
	for _, dt := range codeCatalystTypes {
		out[dt.Name] = dt
	}
	return out
}()
