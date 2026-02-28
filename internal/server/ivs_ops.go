package server

type ivsOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon IVS Low Latency operations sourced from:
// https://docs.aws.amazon.com/ivs/latest/LowLatencyAPIReference/API_Operations.html
var ivsOperations = []ivsOperation{
	{Name: "BatchGetChannel", Method: "POST", URI: "/BatchGetChannel"},
	{Name: "BatchGetStreamKey", Method: "POST", URI: "/BatchGetStreamKey"},
	{Name: "BatchStartViewerSessionRevocation", Method: "POST", URI: "/BatchStartViewerSessionRevocation"},
	{Name: "CreateChannel", Method: "POST", URI: "/CreateChannel"},
	{Name: "CreatePlaybackRestrictionPolicy", Method: "POST", URI: "/CreatePlaybackRestrictionPolicy"},
	{Name: "CreateRecordingConfiguration", Method: "POST", URI: "/CreateRecordingConfiguration"},
	{Name: "CreateStreamKey", Method: "POST", URI: "/CreateStreamKey"},
	{Name: "DeleteChannel", Method: "POST", URI: "/DeleteChannel"},
	{Name: "DeletePlaybackKeyPair", Method: "POST", URI: "/DeletePlaybackKeyPair"},
	{Name: "DeletePlaybackRestrictionPolicy", Method: "POST", URI: "/DeletePlaybackRestrictionPolicy"},
	{Name: "DeleteRecordingConfiguration", Method: "POST", URI: "/DeleteRecordingConfiguration"},
	{Name: "DeleteStreamKey", Method: "POST", URI: "/DeleteStreamKey"},
	{Name: "GetChannel", Method: "POST", URI: "/GetChannel"},
	{Name: "GetPlaybackKeyPair", Method: "POST", URI: "/GetPlaybackKeyPair"},
	{Name: "GetPlaybackRestrictionPolicy", Method: "POST", URI: "/GetPlaybackRestrictionPolicy"},
	{Name: "GetRecordingConfiguration", Method: "POST", URI: "/GetRecordingConfiguration"},
	{Name: "GetStream", Method: "POST", URI: "/GetStream"},
	{Name: "GetStreamKey", Method: "POST", URI: "/GetStreamKey"},
	{Name: "GetStreamSession", Method: "POST", URI: "/GetStreamSession"},
	{Name: "ImportPlaybackKeyPair", Method: "POST", URI: "/ImportPlaybackKeyPair"},
	{Name: "ListChannels", Method: "POST", URI: "/ListChannels"},
	{Name: "ListPlaybackKeyPairs", Method: "POST", URI: "/ListPlaybackKeyPairs"},
	{Name: "ListPlaybackRestrictionPolicies", Method: "POST", URI: "/ListPlaybackRestrictionPolicies"},
	{Name: "ListRecordingConfigurations", Method: "POST", URI: "/ListRecordingConfigurations"},
	{Name: "ListStreamKeys", Method: "POST", URI: "/ListStreamKeys"},
	{Name: "ListStreamSessions", Method: "POST", URI: "/ListStreamSessions"},
	{Name: "ListStreams", Method: "POST", URI: "/ListStreams"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "PutMetadata", Method: "POST", URI: "/PutMetadata"},
	{Name: "StartViewerSessionRevocation", Method: "POST", URI: "/StartViewerSessionRevocation"},
	{Name: "StopStream", Method: "POST", URI: "/StopStream"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateChannel", Method: "POST", URI: "/UpdateChannel"},
	{Name: "UpdatePlaybackRestrictionPolicy", Method: "POST", URI: "/UpdatePlaybackRestrictionPolicy"},
}

var ivsOperationByName = func() map[string]ivsOperation {
	out := make(map[string]ivsOperation, len(ivsOperations))
	for _, op := range ivsOperations {
		out[op.Name] = op
	}
	return out
}()
