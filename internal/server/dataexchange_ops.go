package server

type dataExchangeOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Data Exchange actions sourced from:
// https://docs.aws.amazon.com/data-exchange/latest/apireference/API_Operations.html
var dataExchangeOperations = []dataExchangeOperation{
	{Name: "AcceptDataGrant", Method: "POST", URI: "/v1/data-grants/{DataGrantArn}/accept"},
	{Name: "CancelJob", Method: "DELETE", URI: "/v1/jobs/{JobId}"},
	{Name: "CreateDataGrant", Method: "POST", URI: "/v1/data-grants"},
	{Name: "CreateDataSet", Method: "POST", URI: "/v1/data-sets"},
	{Name: "CreateEventAction", Method: "POST", URI: "/v1/event-actions"},
	{Name: "CreateJob", Method: "POST", URI: "/v1/jobs"},
	{Name: "CreateRevision", Method: "POST", URI: "/v1/data-sets/{DataSetId}/revisions"},
	{Name: "DeleteAsset", Method: "DELETE", URI: "/v1/data-sets/{DataSetId}/revisions/{RevisionId}/assets/{AssetId}"},
	{Name: "DeleteDataGrant", Method: "DELETE", URI: "/v1/data-grants/{DataGrantId}"},
	{Name: "DeleteDataSet", Method: "DELETE", URI: "/v1/data-sets/{DataSetId}"},
	{Name: "DeleteEventAction", Method: "DELETE", URI: "/v1/event-actions/{EventActionId}"},
	{Name: "DeleteRevision", Method: "DELETE", URI: "/v1/data-sets/{DataSetId}/revisions/{RevisionId}"},
	{Name: "GetAsset", Method: "GET", URI: "/v1/data-sets/{DataSetId}/revisions/{RevisionId}/assets/{AssetId}"},
	{Name: "GetDataGrant", Method: "GET", URI: "/v1/data-grants/{DataGrantId}"},
	{Name: "GetDataSet", Method: "GET", URI: "/v1/data-sets/{DataSetId}"},
	{Name: "GetEventAction", Method: "GET", URI: "/v1/event-actions/{EventActionId}"},
	{Name: "GetJob", Method: "GET", URI: "/v1/jobs/{JobId}"},
	{Name: "GetReceivedDataGrant", Method: "GET", URI: "/v1/received-data-grants/{DataGrantArn}"},
	{Name: "GetRevision", Method: "GET", URI: "/v1/data-sets/{DataSetId}/revisions/{RevisionId}"},
	{Name: "ListDataGrants", Method: "GET", URI: "/v1/data-grants?maxResults={MaxResults}&nextToken={NextToken}"},
	{Name: "ListDataSetRevisions", Method: "GET", URI: "/v1/data-sets/{DataSetId}/revisions?maxResults={MaxResults}&nextToken={NextToken}"},
	{Name: "ListDataSets", Method: "GET", URI: "/v1/data-sets?maxResults={MaxResults}&nextToken={NextToken}&origin={Origin}"},
	{Name: "ListEventActions", Method: "GET", URI: "/v1/event-actions?eventSourceId={EventSourceId}&maxResults={MaxResults}&nextToken={NextToken}"},
	{Name: "ListJobs", Method: "GET", URI: "/v1/jobs?dataSetId={DataSetId}&maxResults={MaxResults}&nextToken={NextToken}&revisionId={RevisionId}"},
	{Name: "ListReceivedDataGrants", Method: "GET", URI: "/v1/received-data-grants?acceptanceState={AcceptanceState}&maxResults={MaxResults}&nextToken={NextToken}"},
	{Name: "ListRevisionAssets", Method: "GET", URI: "/v1/data-sets/{DataSetId}/revisions/{RevisionId}/assets?maxResults={MaxResults}&nextToken={NextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "RevokeRevision", Method: "POST", URI: "/v1/data-sets/{DataSetId}/revisions/{RevisionId}/revoke"},
	{Name: "SendApiAsset", Method: "POST", URI: "/v1?{QueryStringParameters}"},
	{Name: "SendDataSetNotification", Method: "POST", URI: "/v1/data-sets/{DataSetId}/notification"},
	{Name: "StartJob", Method: "PATCH", URI: "/v1/jobs/{JobId}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}?tagKeys={TagKeys}"},
	{Name: "UpdateAsset", Method: "PATCH", URI: "/v1/data-sets/{DataSetId}/revisions/{RevisionId}/assets/{AssetId}"},
	{Name: "UpdateDataSet", Method: "PATCH", URI: "/v1/data-sets/{DataSetId}"},
	{Name: "UpdateEventAction", Method: "PATCH", URI: "/v1/event-actions/{EventActionId}"},
	{Name: "UpdateRevision", Method: "PATCH", URI: "/v1/data-sets/{DataSetId}/revisions/{RevisionId}"},
}

var dataExchangeOperationByName = func() map[string]dataExchangeOperation {
	out := make(map[string]dataExchangeOperation, len(dataExchangeOperations))
	for _, op := range dataExchangeOperations {
		out[op.Name] = op
	}
	return out
}()
