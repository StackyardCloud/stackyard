package server

type kinesisOperation struct {
	Name string
}

// Amazon Kinesis Data Streams actions sourced from:
// https://docs.aws.amazon.com/kinesis/latest/APIReference/API_Operations.html
var kinesisOperations = []kinesisOperation{
	{Name: "AddTagsToStream"},
	{Name: "CreateStream"},
	{Name: "DecreaseStreamRetentionPeriod"},
	{Name: "DeleteResourcePolicy"},
	{Name: "DeleteStream"},
	{Name: "DeregisterStreamConsumer"},
	{Name: "DescribeAccountSettings"},
	{Name: "DescribeLimits"},
	{Name: "DescribeStream"},
	{Name: "DescribeStreamConsumer"},
	{Name: "DescribeStreamSummary"},
	{Name: "DisableEnhancedMonitoring"},
	{Name: "EnableEnhancedMonitoring"},
	{Name: "GetRecords"},
	{Name: "GetResourcePolicy"},
	{Name: "GetShardIterator"},
	{Name: "IncreaseStreamRetentionPeriod"},
	{Name: "ListShards"},
	{Name: "ListStreamConsumers"},
	{Name: "ListStreams"},
	{Name: "ListTagsForResource"},
	{Name: "ListTagsForStream"},
	{Name: "MergeShards"},
	{Name: "PutRecord"},
	{Name: "PutRecords"},
	{Name: "PutResourcePolicy"},
	{Name: "RegisterStreamConsumer"},
	{Name: "RemoveTagsFromStream"},
	{Name: "SplitShard"},
	{Name: "StartStreamEncryption"},
	{Name: "StopStreamEncryption"},
	{Name: "SubscribeToShard"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateAccountSettings"},
	{Name: "UpdateMaxRecordSize"},
	{Name: "UpdateShardCount"},
	{Name: "UpdateStreamMode"},
	{Name: "UpdateStreamWarmThroughput"},
}

var kinesisOperationByName = func() map[string]kinesisOperation {
	out := make(map[string]kinesisOperation, len(kinesisOperations))
	for _, op := range kinesisOperations {
		out[op.Name] = op
	}
	return out
}()
