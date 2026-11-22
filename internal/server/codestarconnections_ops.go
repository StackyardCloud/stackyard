package server

type codestarconnectionsOperation struct{ Name string }

// AWS CodeStar Connections operations sourced from:
// https://docs.aws.amazon.com/codestar-connections/latest/APIReference/API_Operations.html
var codestarconnectionsOperations = []codestarconnectionsOperation{
	{Name: "CreateConnection"},
	{Name: "CreateHost"},
	{Name: "CreateRepositoryLink"},
	{Name: "CreateSyncConfiguration"},
	{Name: "DeleteConnection"},
	{Name: "DeleteHost"},
	{Name: "DeleteRepositoryLink"},
	{Name: "DeleteSyncConfiguration"},
	{Name: "GetConnection"},
	{Name: "GetHost"},
	{Name: "GetRepositoryLink"},
	{Name: "GetRepositorySyncStatus"},
	{Name: "GetResourceSyncStatus"},
	{Name: "GetSyncBlockerSummary"},
	{Name: "GetSyncConfiguration"},
	{Name: "ListConnections"},
	{Name: "ListHosts"},
	{Name: "ListRepositoryLinks"},
	{Name: "ListRepositorySyncDefinitions"},
	{Name: "ListSyncConfigurations"},
	{Name: "ListTagsForResource"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateHost"},
	{Name: "UpdateRepositoryLink"},
	{Name: "UpdateSyncBlocker"},
	{Name: "UpdateSyncConfiguration"},
}
