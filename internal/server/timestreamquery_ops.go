package server

// Amazon Timestream Query operations sourced from:
// https://docs.aws.amazon.com/timestream/latest/developerguide/API_Operations_Amazon_Timestream_Query.html
var timestreamQueryOperations = []tsmOperation{
	{Name: "CancelQuery"},
	{Name: "CreateScheduledQuery"},
	{Name: "DeleteScheduledQuery"},
	{Name: "DescribeAccountSettings"},
	{Name: "DescribeEndpoints"},
	{Name: "DescribeScheduledQuery"},
	{Name: "ExecuteScheduledQuery"},
	{Name: "ListScheduledQueries"},
	{Name: "ListTagsForResource"},
	{Name: "PrepareQuery"},
	{Name: "Query"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateAccountSettings"},
	{Name: "UpdateScheduledQuery"},
}
