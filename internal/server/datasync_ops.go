package server

type datasyncOperation struct{ Name string }

// AWS DataSync operations sourced from:
// https://docs.aws.amazon.com/datasync/latest/userguide/API_Operations.html
var datasyncOperations = []datasyncOperation{
	{Name: "CancelTaskExecution"},
	{Name: "CreateAgent"},
	{Name: "CreateLocationAzureBlob"},
	{Name: "CreateLocationEfs"},
	{Name: "CreateLocationFsxLustre"},
	{Name: "CreateLocationFsxOntap"},
	{Name: "CreateLocationFsxOpenZfs"},
	{Name: "CreateLocationFsxWindows"},
	{Name: "CreateLocationHdfs"},
	{Name: "CreateLocationNfs"},
	{Name: "CreateLocationObjectStorage"},
	{Name: "CreateLocationS3"},
	{Name: "CreateLocationSmb"},
	{Name: "CreateTask"},
	{Name: "DeleteAgent"},
	{Name: "DeleteLocation"},
	{Name: "DeleteTask"},
	{Name: "DescribeAgent"},
	{Name: "DescribeLocationAzureBlob"},
	{Name: "DescribeLocationEfs"},
	{Name: "DescribeLocationFsxLustre"},
	{Name: "DescribeLocationFsxOntap"},
	{Name: "DescribeLocationFsxOpenZfs"},
	{Name: "DescribeLocationFsxWindows"},
	{Name: "DescribeLocationHdfs"},
	{Name: "DescribeLocationNfs"},
	{Name: "DescribeLocationObjectStorage"},
	{Name: "DescribeLocationS3"},
	{Name: "DescribeLocationSmb"},
	{Name: "DescribeTask"},
	{Name: "DescribeTaskExecution"},
	{Name: "ListAgents"},
	{Name: "ListLocations"},
	{Name: "ListTagsForResource"},
	{Name: "ListTaskExecutions"},
	{Name: "ListTasks"},
	{Name: "StartTaskExecution"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateAgent"},
	{Name: "UpdateLocationAzureBlob"},
	{Name: "UpdateLocationEfs"},
	{Name: "UpdateLocationFsxLustre"},
	{Name: "UpdateLocationFsxOntap"},
	{Name: "UpdateLocationFsxOpenZfs"},
	{Name: "UpdateLocationFsxWindows"},
	{Name: "UpdateLocationHdfs"},
	{Name: "UpdateLocationNfs"},
	{Name: "UpdateLocationObjectStorage"},
	{Name: "UpdateLocationS3"},
	{Name: "UpdateLocationSmb"},
	{Name: "UpdateTask"},
	{Name: "UpdateTaskExecution"},
}

var datasyncOperationByName = func() map[string]datasyncOperation {
	out := make(map[string]datasyncOperation, len(datasyncOperations))
	for _, op := range datasyncOperations {
		out[op.Name] = op
	}
	return out
}()
