package server

type kinesisDataType struct {
	Name string
}

// Amazon Kinesis Data Streams data types sourced from:
// https://docs.aws.amazon.com/kinesis/latest/APIReference/API_Types.html
var kinesisDataTypes = []kinesisDataType{
	{Name: "ChildShard"},
	{Name: "Consumer"},
	{Name: "ConsumerDescription"},
	{Name: "EnhancedMetrics"},
	{Name: "HashKeyRange"},
	{Name: "MinimumThroughputBillingCommitmentInput"},
	{Name: "MinimumThroughputBillingCommitmentOutput"},
	{Name: "PutRecordsRequestEntry"},
	{Name: "PutRecordsResultEntry"},
	{Name: "Record"},
	{Name: "SequenceNumberRange"},
	{Name: "Shard"},
	{Name: "ShardFilter"},
	{Name: "StartingPosition"},
	{Name: "StreamDescription"},
	{Name: "StreamDescriptionSummary"},
	{Name: "StreamModeDetails"},
	{Name: "StreamSummary"},
	{Name: "SubscribeToShardEvent"},
	{Name: "SubscribeToShardEventStream"},
	{Name: "Tag"},
	{Name: "WarmThroughputObject"},
	{Name: "UpdateStreamWarmThroughput"},
}

var kinesisDataTypeByName = func() map[string]kinesisDataType {
	out := make(map[string]kinesisDataType, len(kinesisDataTypes))
	for _, dt := range kinesisDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
