package server

type dynamodbstreamsOperation struct {
	Name string
}

// Amazon DynamoDB Streams operations sourced from:
// https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_Operations_Amazon_DynamoDB_Streams.html
var dynamodbstreamsOperations = []dynamodbstreamsOperation{
	{Name: "DescribeStream"},
	{Name: "GetRecords"},
	{Name: "GetShardIterator"},
	{Name: "ListStreams"},
}

var dynamodbstreamsOperationByName = func() map[string]dynamodbstreamsOperation {
	out := make(map[string]dynamodbstreamsOperation, len(dynamodbstreamsOperations))
	for _, op := range dynamodbstreamsOperations {
		out[op.Name] = op
	}
	return out
}()
