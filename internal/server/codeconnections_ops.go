package server

type codeconnectionsOperation struct{ Name string }

// AWS CodeConnections operations sourced from:
// https://docs.aws.amazon.com/codeconnections/latest/APIReference/API_Operations.html
var codeconnectionsOperations = []codeconnectionsOperation{
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
