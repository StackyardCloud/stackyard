package server

type cognitoSyncOperation struct {
	Name string
}

// Amazon Cognito Sync operations sourced from:
// https://docs.aws.amazon.com/cognitosync/latest/APIReference/API_Operations.html
var cognitoSyncOperations = []cognitoSyncOperation{
	{Name: "BulkPublish"},
	{Name: "DeleteDataset"},
	{Name: "DescribeDataset"},
	{Name: "DescribeIdentityPoolUsage"},
	{Name: "DescribeIdentityUsage"},
	{Name: "GetBulkPublishDetails"},
	{Name: "GetCognitoEvents"},
	{Name: "GetIdentityPoolConfiguration"},
	{Name: "ListDatasets"},
	{Name: "ListIdentityPoolUsage"},
	{Name: "ListRecords"},
	{Name: "RegisterDevice"},
	{Name: "SetCognitoEvents"},
	{Name: "SetIdentityPoolConfiguration"},
	{Name: "SubscribeToDataset"},
	{Name: "UnsubscribeFromDataset"},
	{Name: "UpdateRecords"},
}

var cognitoSyncOperationByName = func() map[string]cognitoSyncOperation {
	out := make(map[string]cognitoSyncOperation, len(cognitoSyncOperations))
	for _, op := range cognitoSyncOperations {
		out[op.Name] = op
	}
	return out
}()
