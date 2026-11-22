package server

type kinesisVideoMediaOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Kinesis Video Streams Media operations sourced from:
// https://docs.aws.amazon.com/kinesisvideostreams/latest/dg/API_Operations_Amazon_Kinesis_Video_Streams_Media.html
var kinesisVideoMediaOperations = []kinesisVideoMediaOperation{
	{Name: "GetMedia", Method: "POST", URI: "/getMedia"},
	{Name: "PutMedia", Method: "POST", URI: "/putMedia"},
}

var kinesisVideoMediaOperationByName = func() map[string]kinesisVideoMediaOperation {
	out := make(map[string]kinesisVideoMediaOperation, len(kinesisVideoMediaOperations))
	for _, op := range kinesisVideoMediaOperations {
		out[op.Name] = op
	}
	return out
}()
