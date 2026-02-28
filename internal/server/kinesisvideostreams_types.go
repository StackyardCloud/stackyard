package server

type kinesisVideoStreamsDataType struct {
	Name string
}

// Amazon Kinesis Video Streams API data types sourced from:
// https://docs.aws.amazon.com/kinesisvideostreams/latest/dg/API_Types.html
var kinesisVideoStreamsDataTypes = []kinesisVideoStreamsDataType{
	{Name: "ChannelInfo"},
	{Name: "ChannelNameCondition"},
	{Name: "DeletionConfig"},
	{Name: "EdgeAgentStatus"},
	{Name: "EdgeConfig"},
	{Name: "ImageGenerationConfiguration"},
	{Name: "ImageGenerationDestinationConfig"},
	{Name: "LastRecorderStatus"},
	{Name: "LastUploaderStatus"},
	{Name: "ListEdgeAgentConfigurationsEdgeConfig"},
	{Name: "LocalSizeConfig"},
	{Name: "MappedResourceConfigurationListItem"},
	{Name: "MediaSourceConfig"},
	{Name: "MediaStorageConfiguration"},
	{Name: "NotificationConfiguration"},
	{Name: "NotificationDestinationConfig"},
	{Name: "RecorderConfig"},
	{Name: "ResourceEndpointListItem"},
	{Name: "ScheduleConfig"},
	{Name: "SingleMasterChannelEndpointConfiguration"},
	{Name: "SingleMasterConfiguration"},
	{Name: "StreamInfo"},
	{Name: "StreamNameCondition"},
	{Name: "StreamStorageConfiguration"},
	{Name: "Tag"},
	{Name: "UploaderConfig"},
}

var kinesisVideoStreamsDataTypeByName = func() map[string]kinesisVideoStreamsDataType {
	out := make(map[string]kinesisVideoStreamsDataType, len(kinesisVideoStreamsDataTypes))
	for _, dt := range kinesisVideoStreamsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
