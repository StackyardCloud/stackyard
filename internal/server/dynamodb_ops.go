package server

type dynamodbOperation struct {
	Name string
}

// Amazon DynamoDB operations sourced from:
// https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_Operations.html
var dynamodbOperations = []dynamodbOperation{
	{Name: "BatchExecuteStatement"},
	{Name: "BatchGetItem"},
	{Name: "BatchWriteItem"},
	{Name: "CreateBackup"},
	{Name: "CreateGlobalTable"},
	{Name: "CreateTable"},
	{Name: "DeleteBackup"},
	{Name: "DeleteItem"},
	{Name: "DeleteResourcePolicy"},
	{Name: "DeleteTable"},
	{Name: "DescribeBackup"},
	{Name: "DescribeContinuousBackups"},
	{Name: "DescribeContributorInsights"},
	{Name: "DescribeEndpoints"},
	{Name: "DescribeExport"},
	{Name: "DescribeGlobalTable"},
	{Name: "DescribeGlobalTableSettings"},
	{Name: "DescribeImport"},
	{Name: "DescribeKinesisStreamingDestination"},
	{Name: "DescribeLimits"},
	{Name: "DescribeTable"},
	{Name: "DescribeTableReplicaAutoScaling"},
	{Name: "DescribeTimeToLive"},
	{Name: "DisableKinesisStreamingDestination"},
	{Name: "EnableKinesisStreamingDestination"},
	{Name: "ExecuteStatement"},
	{Name: "ExecuteTransaction"},
	{Name: "ExportTableToPointInTime"},
	{Name: "GetItem"},
	{Name: "GetResourcePolicy"},
	{Name: "ImportTable"},
	{Name: "ListBackups"},
	{Name: "ListContributorInsights"},
	{Name: "ListExports"},
	{Name: "ListGlobalTables"},
	{Name: "ListImports"},
	{Name: "ListTables"},
	{Name: "ListTagsOfResource"},
	{Name: "PutItem"},
	{Name: "PutResourcePolicy"},
	{Name: "Query"},
	{Name: "RestoreTableFromBackup"},
	{Name: "RestoreTableToPointInTime"},
	{Name: "Scan"},
	{Name: "TagResource"},
	{Name: "TransactGetItems"},
	{Name: "TransactWriteItems"},
	{Name: "UntagResource"},
	{Name: "UpdateContinuousBackups"},
	{Name: "UpdateContributorInsights"},
	{Name: "UpdateGlobalTable"},
	{Name: "UpdateGlobalTableSettings"},
	{Name: "UpdateItem"},
	{Name: "UpdateKinesisStreamingDestination"},
	{Name: "UpdateTable"},
	{Name: "UpdateTableReplicaAutoScaling"},
	{Name: "UpdateTimeToLive"},
}

var dynamodbOperationByName = func() map[string]dynamodbOperation {
	out := make(map[string]dynamodbOperation, len(dynamodbOperations))
	for _, op := range dynamodbOperations {
		out[op.Name] = op
	}
	return out
}()
