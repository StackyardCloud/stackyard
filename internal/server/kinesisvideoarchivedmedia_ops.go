package server

type kinesisVideoArchivedMediaOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Kinesis Video Streams Archived Media operations sourced from:
// https://docs.aws.amazon.com/kinesisvideostreams/latest/dg/API_Operations_Amazon_Kinesis_Video_Streams_Archived_Media.html
var kinesisVideoArchivedMediaOperations = []kinesisVideoArchivedMediaOperation{
	{Name: "GetClip", Method: "POST", URI: "/getClip"},
	{Name: "GetDASHStreamingSessionURL", Method: "POST", URI: "/getDASHStreamingSessionURL"},
	{Name: "GetHLSStreamingSessionURL", Method: "POST", URI: "/getHLSStreamingSessionURL"},
	{Name: "GetImages", Method: "POST", URI: "/getImages"},
	{Name: "GetMediaForFragmentList", Method: "POST", URI: "/getMediaForFragmentList"},
	{Name: "ListFragments", Method: "POST", URI: "/listFragments"},
}

var kinesisVideoArchivedMediaOperationByName = func() map[string]kinesisVideoArchivedMediaOperation {
	out := make(map[string]kinesisVideoArchivedMediaOperation, len(kinesisVideoArchivedMediaOperations))
	for _, op := range kinesisVideoArchivedMediaOperations {
		out[op.Name] = op
	}
	return out
}()
