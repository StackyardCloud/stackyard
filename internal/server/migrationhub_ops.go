package server

type migrationHubOperation struct{ Name string }

// AWS Migration Hub operations sourced from:
// https://docs.aws.amazon.com/migrationhub/latest/ug/API_Operations.html
var migrationHubOperations = []migrationHubOperation{
	{Name: "AssociateCreatedArtifact"},
	{Name: "AssociateDiscoveredResource"},
	{Name: "AssociateSourceResource"},
	{Name: "CreateProgressUpdateStream"},
	{Name: "DeleteProgressUpdateStream"},
	{Name: "DescribeApplicationState"},
	{Name: "DescribeMigrationTask"},
	{Name: "DisassociateCreatedArtifact"},
	{Name: "DisassociateDiscoveredResource"},
	{Name: "DisassociateSourceResource"},
	{Name: "ImportMigrationTask"},
	{Name: "ListApplicationStates"},
	{Name: "ListCreatedArtifacts"},
	{Name: "ListDiscoveredResources"},
	{Name: "ListMigrationTaskUpdates"},
	{Name: "ListMigrationTasks"},
	{Name: "ListProgressUpdateStreams"},
	{Name: "ListSourceResources"},
	{Name: "NotifyApplicationState"},
	{Name: "NotifyMigrationTaskState"},
	{Name: "PutResourceAttributes"},
}

var migrationHubOperationByName = func() map[string]migrationHubOperation {
	out := make(map[string]migrationHubOperation, len(migrationHubOperations))
	for _, op := range migrationHubOperations {
		out[op.Name] = op
	}
	return out
}()
