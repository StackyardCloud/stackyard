package server

type ivsRealtimeOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon IVS Real-Time Streaming operations sourced from:
// https://docs.aws.amazon.com/ivs/latest/RealTimeAPIReference/API_Operations.html
var ivsRealtimeOperations = []ivsRealtimeOperation{
	{Name: "CreateEncoderConfiguration", Method: "POST", URI: "/CreateEncoderConfiguration"},
	{Name: "CreateIngestConfiguration", Method: "POST", URI: "/CreateIngestConfiguration"},
	{Name: "CreateParticipantToken", Method: "POST", URI: "/CreateParticipantToken"},
	{Name: "CreateStage", Method: "POST", URI: "/CreateStage"},
	{Name: "CreateStorageConfiguration", Method: "POST", URI: "/CreateStorageConfiguration"},
	{Name: "DeleteEncoderConfiguration", Method: "POST", URI: "/DeleteEncoderConfiguration"},
	{Name: "DeleteIngestConfiguration", Method: "POST", URI: "/DeleteIngestConfiguration"},
	{Name: "DeletePublicKey", Method: "POST", URI: "/DeletePublicKey"},
	{Name: "DeleteStage", Method: "POST", URI: "/DeleteStage"},
	{Name: "DeleteStorageConfiguration", Method: "POST", URI: "/DeleteStorageConfiguration"},
	{Name: "DisconnectParticipant", Method: "POST", URI: "/DisconnectParticipant"},
	{Name: "GetComposition", Method: "POST", URI: "/GetComposition"},
	{Name: "GetEncoderConfiguration", Method: "POST", URI: "/GetEncoderConfiguration"},
	{Name: "GetIngestConfiguration", Method: "POST", URI: "/GetIngestConfiguration"},
	{Name: "GetParticipant", Method: "POST", URI: "/GetParticipant"},
	{Name: "GetPublicKey", Method: "POST", URI: "/GetPublicKey"},
	{Name: "GetStage", Method: "POST", URI: "/GetStage"},
	{Name: "GetStageSession", Method: "POST", URI: "/GetStageSession"},
	{Name: "GetStorageConfiguration", Method: "POST", URI: "/GetStorageConfiguration"},
	{Name: "ImportPublicKey", Method: "POST", URI: "/ImportPublicKey"},
	{Name: "ListCompositions", Method: "POST", URI: "/ListCompositions"},
	{Name: "ListEncoderConfigurations", Method: "POST", URI: "/ListEncoderConfigurations"},
	{Name: "ListIngestConfigurations", Method: "POST", URI: "/ListIngestConfigurations"},
	{Name: "ListParticipantEvents", Method: "POST", URI: "/ListParticipantEvents"},
	{Name: "ListParticipantReplicas", Method: "POST", URI: "/ListParticipantReplicas"},
	{Name: "ListParticipants", Method: "POST", URI: "/ListParticipants"},
	{Name: "ListPublicKeys", Method: "POST", URI: "/ListPublicKeys"},
	{Name: "ListStages", Method: "POST", URI: "/ListStages"},
	{Name: "ListStageSessions", Method: "POST", URI: "/ListStageSessions"},
	{Name: "ListStorageConfigurations", Method: "POST", URI: "/ListStorageConfigurations"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "StartComposition", Method: "POST", URI: "/StartComposition"},
	{Name: "StartParticipantReplication", Method: "POST", URI: "/StartParticipantReplication"},
	{Name: "StopComposition", Method: "POST", URI: "/StopComposition"},
	{Name: "StopParticipantReplication", Method: "POST", URI: "/StopParticipantReplication"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateIngestConfiguration", Method: "POST", URI: "/UpdateIngestConfiguration"},
	{Name: "UpdateStage", Method: "POST", URI: "/UpdateStage"},
}

var ivsRealtimeOperationByName = func() map[string]ivsRealtimeOperation {
	out := make(map[string]ivsRealtimeOperation, len(ivsRealtimeOperations))
	for _, op := range ivsRealtimeOperations {
		out[op.Name] = op
	}
	return out
}()
