package server

type iotTwinMakerOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS IoT TwinMaker operations sourced from:
// https://docs.aws.amazon.com/iot-twinmaker/latest/apireference/API_Operations.html
var iotTwinMakerOperations = []iotTwinMakerOperation{
	{Name: "BatchPutPropertyValues", Method: "POST", URI: "/workspaces/{workspaceId}/entity-properties"},
	{Name: "CancelMetadataTransferJob", Method: "PUT", URI: "/metadata-transfer-jobs/{metadataTransferJobId}/cancel"},
	{Name: "CreateComponentType", Method: "POST", URI: "/workspaces/{workspaceId}/component-types/{componentTypeId}"},
	{Name: "CreateEntity", Method: "POST", URI: "/workspaces/{workspaceId}/entities"},
	{Name: "CreateMetadataTransferJob", Method: "POST", URI: "/metadata-transfer-jobs"},
	{Name: "CreateScene", Method: "POST", URI: "/workspaces/{workspaceId}/scenes"},
	{Name: "CreateSyncJob", Method: "POST", URI: "/workspaces/{workspaceId}/sync-jobs/{syncSource}"},
	{Name: "CreateWorkspace", Method: "POST", URI: "/workspaces/{workspaceId}"},
	{Name: "DeleteComponentType", Method: "DELETE", URI: "/workspaces/{workspaceId}/component-types/{componentTypeId}"},
	{Name: "DeleteEntity", Method: "DELETE", URI: "/workspaces/{workspaceId}/entities/{entityId}"},
	{Name: "DeleteScene", Method: "DELETE", URI: "/workspaces/{workspaceId}/scenes/{sceneId}"},
	{Name: "DeleteSyncJob", Method: "DELETE", URI: "/workspaces/{workspaceId}/sync-jobs/{syncSource}"},
	{Name: "DeleteWorkspace", Method: "DELETE", URI: "/workspaces/{workspaceId}"},
	{Name: "ExecuteQuery", Method: "POST", URI: "/queries/execution"},
	{Name: "GetComponentType", Method: "GET", URI: "/workspaces/{workspaceId}/component-types/{componentTypeId}"},
	{Name: "GetEntity", Method: "GET", URI: "/workspaces/{workspaceId}/entities/{entityId}"},
	{Name: "GetMetadataTransferJob", Method: "GET", URI: "/metadata-transfer-jobs/{metadataTransferJobId}"},
	{Name: "GetPricingPlan", Method: "GET", URI: "/pricingplan"},
	{Name: "GetPropertyValue", Method: "POST", URI: "/workspaces/{workspaceId}/entity-properties/value"},
	{Name: "GetPropertyValueHistory", Method: "POST", URI: "/workspaces/{workspaceId}/entity-properties/history"},
	{Name: "GetScene", Method: "GET", URI: "/workspaces/{workspaceId}/scenes/{sceneId}"},
	{Name: "GetSyncJob", Method: "GET", URI: "/sync-jobs/{syncSource}"},
	{Name: "GetWorkspace", Method: "GET", URI: "/workspaces/{workspaceId}"},
	{Name: "ListComponentTypes", Method: "POST", URI: "/workspaces/{workspaceId}/component-types-list"},
	{Name: "ListComponents", Method: "POST", URI: "/workspaces/{workspaceId}/entities/{entityId}/components-list"},
	{Name: "ListEntities", Method: "POST", URI: "/workspaces/{workspaceId}/entities-list"},
	{Name: "ListMetadataTransferJobs", Method: "POST", URI: "/metadata-transfer-jobs-list"},
	{Name: "ListProperties", Method: "POST", URI: "/workspaces/{workspaceId}/properties-list"},
	{Name: "ListScenes", Method: "POST", URI: "/workspaces/{workspaceId}/scenes-list"},
	{Name: "ListSyncJobs", Method: "POST", URI: "/workspaces/{workspaceId}/sync-jobs-list"},
	{Name: "ListSyncResources", Method: "POST", URI: "/workspaces/{workspaceId}/sync-jobs/{syncSource}/resources-list"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/tags-list"},
	{Name: "ListWorkspaces", Method: "POST", URI: "/workspaces-list"},
	{Name: "TagResource", Method: "POST", URI: "/tags"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags"},
	{Name: "UpdateComponentType", Method: "PUT", URI: "/workspaces/{workspaceId}/component-types/{componentTypeId}"},
	{Name: "UpdateEntity", Method: "PUT", URI: "/workspaces/{workspaceId}/entities/{entityId}"},
	{Name: "UpdatePricingPlan", Method: "POST", URI: "/pricingplan"},
	{Name: "UpdateScene", Method: "PUT", URI: "/workspaces/{workspaceId}/scenes/{sceneId}"},
	{Name: "UpdateWorkspace", Method: "PUT", URI: "/workspaces/{workspaceId}"},
}

var iotTwinMakerOperationByName = func() map[string]iotTwinMakerOperation {
	out := make(map[string]iotTwinMakerOperation, len(iotTwinMakerOperations))
	for _, op := range iotTwinMakerOperations {
		out[op.Name] = op
	}
	return out
}()
