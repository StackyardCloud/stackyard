package server

type ivsDataType struct {
	Name string
}

// Amazon IVS Low Latency data types sourced from:
// https://docs.aws.amazon.com/ivs/latest/LowLatencyAPIReference/API_Types.html
var ivsDataTypes = []ivsDataType{
	{Name: "AudioConfiguration"},
	{Name: "BatchError"},
	{Name: "BatchStartViewerSessionRevocationError"},
	{Name: "BatchStartViewerSessionRevocationViewerSession"},
	{Name: "Channel"},
	{Name: "ChannelSummary"},
	{Name: "DestinationConfiguration"},
	{Name: "IngestConfiguration"},
	{Name: "IngestConfigurations"},
	{Name: "MultitrackInputConfiguration"},
	{Name: "PlaybackKeyPair"},
	{Name: "PlaybackKeyPairSummary"},
	{Name: "PlaybackRestrictionPolicy"},
	{Name: "PlaybackRestrictionPolicySummary"},
	{Name: "RecordingConfiguration"},
	{Name: "RecordingConfigurationSummary"},
	{Name: "RenditionConfiguration"},
	{Name: "S3DestinationConfiguration"},
	{Name: "Srt"},
	{Name: "Stream"},
	{Name: "StreamEvent"},
	{Name: "StreamFilters"},
	{Name: "StreamKey"},
	{Name: "StreamKeySummary"},
	{Name: "StreamSession"},
	{Name: "StreamSessionSummary"},
	{Name: "StreamSummary"},
	{Name: "ThumbnailConfiguration"},
	{Name: "UpdatePlaybackRestrictionPolicy"},
	{Name: "VideoConfiguration"},
}

var ivsDataTypeByName = func() map[string]ivsDataType {
	out := make(map[string]ivsDataType, len(ivsDataTypes))
	for _, dt := range ivsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
