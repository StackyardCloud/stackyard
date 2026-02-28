package server

type firehoseOperation struct {
	Name string
}

// Amazon Data Firehose actions sourced from:
// https://docs.aws.amazon.com/firehose/latest/APIReference/API_Operations.html
var firehoseOperations = []firehoseOperation{
	{Name: "CreateDeliveryStream"},
	{Name: "DeleteDeliveryStream"},
	{Name: "DescribeDeliveryStream"},
	{Name: "ListDeliveryStreams"},
	{Name: "ListTagsForDeliveryStream"},
	{Name: "PutRecord"},
	{Name: "PutRecordBatch"},
	{Name: "StartDeliveryStreamEncryption"},
	{Name: "StopDeliveryStreamEncryption"},
	{Name: "TagDeliveryStream"},
	{Name: "UntagDeliveryStream"},
	{Name: "UpdateDestination"},
}

var firehoseOperationByName = func() map[string]firehoseOperation {
	out := make(map[string]firehoseOperation, len(firehoseOperations))
	for _, op := range firehoseOperations {
		out[op.Name] = op
	}
	return out
}()
