package server

type tsmOperation struct{ Name string }

// Amazon Timestream Write operations sourced from:
// https://docs.aws.amazon.com/timestream/latest/developerguide/API_Operations_Amazon_Timestream_Write.html
var timestreamWriteOperations = []tsmOperation{
	{Name: "CreateBatchLoadTask"},
	{Name: "CreateDatabase"},
	{Name: "CreateTable"},
	{Name: "DeleteDatabase"},
	{Name: "DeleteTable"},
	{Name: "DescribeBatchLoadTask"},
	{Name: "DescribeDatabase"},
	{Name: "DescribeEndpoints"},
	{Name: "DescribeTable"},
	{Name: "ListBatchLoadTasks"},
	{Name: "ListDatabases"},
	{Name: "ListTables"},
	{Name: "ListTagsForResource"},
	{Name: "ResumeBatchLoadTask"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateDatabase"},
	{Name: "UpdateTable"},
	{Name: "WriteRecords"},
}
