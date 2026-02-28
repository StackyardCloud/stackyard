package server

type resourceExplorer2Operation struct {
	Name   string
	Method string
	URI    string
}

// AWS Resource Explorer operations sourced from:
// https://docs.aws.amazon.com/resource-explorer/latest/apireference/API_Operations.html
var resourceExplorer2Operations = []resourceExplorer2Operation{
	{Name: "AssociateDefaultView", Method: "POST", URI: "/AssociateDefaultView"},
	{Name: "BatchGetView", Method: "POST", URI: "/BatchGetView"},
	{Name: "CreateIndex", Method: "POST", URI: "/CreateIndex"},
	{Name: "CreateResourceExplorerSetup", Method: "POST", URI: "/CreateResourceExplorerSetup"},
	{Name: "CreateView", Method: "POST", URI: "/CreateView"},
	{Name: "DeleteIndex", Method: "POST", URI: "/DeleteIndex"},
	{Name: "DeleteResourceExplorerSetup", Method: "POST", URI: "/DeleteResourceExplorerSetup"},
	{Name: "DeleteView", Method: "POST", URI: "/DeleteView"},
	{Name: "DisassociateDefaultView", Method: "POST", URI: "/DisassociateDefaultView"},
	{Name: "GetAccountLevelServiceConfiguration", Method: "POST", URI: "/GetAccountLevelServiceConfiguration"},
	{Name: "GetDefaultView", Method: "POST", URI: "/GetDefaultView"},
	{Name: "GetIndex", Method: "POST", URI: "/GetIndex"},
	{Name: "GetManagedView", Method: "POST", URI: "/GetManagedView"},
	{Name: "GetResourceExplorerSetup", Method: "POST", URI: "/GetResourceExplorerSetup"},
	{Name: "GetServiceIndex", Method: "POST", URI: "/GetServiceIndex"},
	{Name: "GetServiceView", Method: "POST", URI: "/GetServiceView"},
	{Name: "GetView", Method: "POST", URI: "/GetView"},
	{Name: "ListIndexes", Method: "POST", URI: "/ListIndexes"},
	{Name: "ListIndexesForMembers", Method: "POST", URI: "/ListIndexesForMembers"},
	{Name: "ListManagedViews", Method: "POST", URI: "/ListManagedViews"},
	{Name: "ListResources", Method: "POST", URI: "/ListResources"},
	{Name: "ListServiceIndexes", Method: "POST", URI: "/ListServiceIndexes"},
	{Name: "ListServiceViews", Method: "POST", URI: "/ListServiceViews"},
	{Name: "ListStreamingAccessForServices", Method: "POST", URI: "/ListStreamingAccessForServices"},
	{Name: "ListSupportedResourceTypes", Method: "POST", URI: "/ListSupportedResourceTypes"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListViews", Method: "POST", URI: "/ListViews"},
	{Name: "Search", Method: "POST", URI: "/Search"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateIndexType", Method: "POST", URI: "/UpdateIndexType"},
	{Name: "UpdateView", Method: "POST", URI: "/UpdateView"},
}

var resourceExplorer2OperationByName = func() map[string]resourceExplorer2Operation {
	out := make(map[string]resourceExplorer2Operation, len(resourceExplorer2Operations))
	for _, op := range resourceExplorer2Operations {
		out[op.Name] = op
	}
	return out
}()
