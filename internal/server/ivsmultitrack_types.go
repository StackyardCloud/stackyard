package server

type ivsMultitrackDataType struct {
	Name string
}

// Amazon IVS Multitrack Video Integration data types sourced from:
// https://docs.aws.amazon.com/ivs/latest/BroadcastSWIntegAPIReference/structures.html
var ivsMultitrackDataTypes = []ivsMultitrackDataType{
	{Name: "AudioConfiguration"},
	{Name: "AudioTrackConfiguration"},
	{Name: "AudioTrackSettings"},
	{Name: "CapabilitiesDescription"},
	{Name: "Client"},
	{Name: "ClientConfigurationStatus"},
	{Name: "ClientDescription"},
	{Name: "ConfigurationMetadata"},
	{Name: "CpuDescription"},
	{Name: "EncoderConfiguration"},
	{Name: "Framerate"},
	{Name: "GamingFeaturesDescription"},
	{Name: "GpuDescription"},
	{Name: "Ingest"},
	{Name: "IngestEndpoint"},
	{Name: "MemoryDescription"},
	{Name: "PreferencesDescription"},
	{Name: "SystemDescription"},
	{Name: "VideoTrackSettings"},
}

var ivsMultitrackDataTypeByName = func() map[string]ivsMultitrackDataType {
	out := make(map[string]ivsMultitrackDataType, len(ivsMultitrackDataTypes))
	for _, dt := range ivsMultitrackDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
