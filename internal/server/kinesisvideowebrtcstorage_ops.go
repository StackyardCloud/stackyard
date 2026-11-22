package server

type kinesisVideoWebRTCStorageOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Kinesis Video WebRTC Storage operations sourced from:
// https://docs.aws.amazon.com/kinesisvideostreams/latest/dg/API_Operations_Amazon_Kinesis_Video_WebRTC_Storage.html
var kinesisVideoWebRTCStorageOperations = []kinesisVideoWebRTCStorageOperation{
	{Name: "JoinStorageSession", Method: "POST", URI: "/joinStorageSession"},
	{Name: "JoinStorageSessionAsViewer", Method: "POST", URI: "/joinStorageSessionAsViewer"},
}

var kinesisVideoWebRTCStorageOperationByName = func() map[string]kinesisVideoWebRTCStorageOperation {
	out := make(map[string]kinesisVideoWebRTCStorageOperation, len(kinesisVideoWebRTCStorageOperations))
	for _, op := range kinesisVideoWebRTCStorageOperations {
		out[op.Name] = op
	}
	return out
}()
