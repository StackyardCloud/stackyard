package server

type kinesisVideoStreamsOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Kinesis Video Streams API actions sourced from:
// https://docs.aws.amazon.com/kinesisvideostreams/latest/dg/API_Operations.html
var kinesisVideoStreamsOperations = []kinesisVideoStreamsOperation{
	{Name: "CreateSignalingChannel", Method: "POST", URI: "/createSignalingChannel"},
	{Name: "CreateStream", Method: "POST", URI: "/createStream"},
	{Name: "DeleteEdgeConfiguration", Method: "POST", URI: "/deleteEdgeConfiguration"},
	{Name: "DeleteSignalingChannel", Method: "POST", URI: "/deleteSignalingChannel"},
	{Name: "DeleteStream", Method: "POST", URI: "/deleteStream"},
	{Name: "DescribeEdgeConfiguration", Method: "POST", URI: "/describeEdgeConfiguration"},
	{Name: "DescribeImageGenerationConfiguration", Method: "POST", URI: "/describeImageGenerationConfiguration"},
	{Name: "DescribeMappedResourceConfiguration", Method: "POST", URI: "/describeMappedResourceConfiguration"},
	{Name: "DescribeMediaStorageConfiguration", Method: "POST", URI: "/describeMediaStorageConfiguration"},
	{Name: "DescribeNotificationConfiguration", Method: "POST", URI: "/describeNotificationConfiguration"},
	{Name: "DescribeSignalingChannel", Method: "POST", URI: "/describeSignalingChannel"},
	{Name: "DescribeStream", Method: "POST", URI: "/describeStream"},
	{Name: "DescribeStreamStorageConfiguration", Method: "POST", URI: "/describeStreamStorageConfiguration"},
	{Name: "GetDataEndpoint", Method: "POST", URI: "/getDataEndpoint"},
	{Name: "GetSignalingChannelEndpoint", Method: "POST", URI: "/getSignalingChannelEndpoint"},
	{Name: "ListEdgeAgentConfigurations", Method: "POST", URI: "/listEdgeAgentConfigurations"},
	{Name: "ListSignalingChannels", Method: "POST", URI: "/listSignalingChannels"},
	{Name: "ListStreams", Method: "POST", URI: "/listStreams"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/ListTagsForResource"},
	{Name: "ListTagsForStream", Method: "POST", URI: "/listTagsForStream"},
	{Name: "StartEdgeConfigurationUpdate", Method: "POST", URI: "/startEdgeConfigurationUpdate"},
	{Name: "TagResource", Method: "POST", URI: "/TagResource"},
	{Name: "TagStream", Method: "POST", URI: "/tagStream"},
	{Name: "UntagResource", Method: "POST", URI: "/UntagResource"},
	{Name: "UntagStream", Method: "POST", URI: "/untagStream"},
	{Name: "UpdateDataRetention", Method: "POST", URI: "/updateDataRetention"},
	{Name: "UpdateImageGenerationConfiguration", Method: "POST", URI: "/updateImageGenerationConfiguration"},
	{Name: "UpdateMediaStorageConfiguration", Method: "POST", URI: "/updateMediaStorageConfiguration"},
	{Name: "UpdateNotificationConfiguration", Method: "POST", URI: "/updateNotificationConfiguration"},
	{Name: "UpdateSignalingChannel", Method: "POST", URI: "/updateSignalingChannel"},
	{Name: "UpdateStream", Method: "POST", URI: "/updateStream"},
	{Name: "UpdateStreamStorageConfiguration", Method: "POST", URI: "/updateStreamStorageConfiguration"},
}

var kinesisVideoStreamsOperationByName = func() map[string]kinesisVideoStreamsOperation {
	out := make(map[string]kinesisVideoStreamsOperation, len(kinesisVideoStreamsOperations))
	for _, op := range kinesisVideoStreamsOperations {
		out[op.Name] = op
	}
	return out
}()
