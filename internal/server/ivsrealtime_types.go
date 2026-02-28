package server

type ivsRealtimeDataType struct {
	Name string
}

// Amazon IVS Real-Time Streaming data types sourced from:
// https://docs.aws.amazon.com/ivs/latest/RealTimeAPIReference/API_Types.html
var ivsRealtimeDataTypes = []ivsRealtimeDataType{
	{Name: "AutoParticipantRecordingConfiguration"},
	{Name: "ChannelDestinationConfiguration"},
	{Name: "Composition"},
	{Name: "CompositionRecordingHlsConfiguration"},
	{Name: "CompositionSummary"},
	{Name: "CompositionThumbnailConfiguration"},
	{Name: "Destination"},
	{Name: "DestinationConfiguration"},
	{Name: "DestinationDetail"},
	{Name: "DestinationSummary"},
	{Name: "EncoderConfiguration"},
	{Name: "EncoderConfigurationSummary"},
	{Name: "Event"},
	{Name: "ExchangedParticipantToken"},
	{Name: "GridConfiguration"},
	{Name: "IngestConfiguration"},
	{Name: "IngestConfigurationSummary"},
	{Name: "LayoutConfiguration"},
	{Name: "Participant"},
	{Name: "ParticipantRecordingHlsConfiguration"},
	{Name: "ParticipantReplica"},
	{Name: "ParticipantSummary"},
	{Name: "ParticipantThumbnailConfiguration"},
	{Name: "ParticipantToken"},
	{Name: "ParticipantTokenConfiguration"},
	{Name: "PipConfiguration"},
	{Name: "PublicKey"},
	{Name: "PublicKeySummary"},
	{Name: "RecordingConfiguration"},
	{Name: "S3DestinationConfiguration"},
	{Name: "S3Detail"},
	{Name: "S3StorageConfiguration"},
	{Name: "Stage"},
	{Name: "StageEndpoints"},
	{Name: "StageSession"},
	{Name: "StageSessionSummary"},
	{Name: "StageSummary"},
	{Name: "StorageConfiguration"},
	{Name: "StorageConfigurationSummary"},
	{Name: "Video"},
}

var ivsRealtimeDataTypeByName = func() map[string]ivsRealtimeDataType {
	out := make(map[string]ivsRealtimeDataType, len(ivsRealtimeDataTypes))
	for _, dt := range ivsRealtimeDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
