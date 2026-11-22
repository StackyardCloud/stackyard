package server

type kinesisVideoSignalingOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Kinesis Video Signaling Channels operations sourced from:
// https://docs.aws.amazon.com/kinesisvideostreams/latest/dg/API_Operations_Amazon_Kinesis_Video_Signaling_Channels.html
var kinesisVideoSignalingOperations = []kinesisVideoSignalingOperation{
	{Name: "GetIceServerConfig", Method: "POST", URI: "/v1/get-ice-server-config"},
	{Name: "SendAlexaOfferToMaster", Method: "POST", URI: "/v1/send-alexa-offer-to-master"},
}

var kinesisVideoSignalingOperationByName = func() map[string]kinesisVideoSignalingOperation {
	out := make(map[string]kinesisVideoSignalingOperation, len(kinesisVideoSignalingOperations))
	for _, op := range kinesisVideoSignalingOperations {
		out[op.Name] = op
	}
	return out
}()
