package server

type keyspacesStreamsOperation struct{ Name string }

// Amazon Keyspaces Streams operations sourced from:
// https://docs.aws.amazon.com/keyspaces/latest/StreamsAPIReference/API_Operations.html
var keyspacesStreamsOperations = []keyspacesStreamsOperation{
	{Name: "GetRecords"},
	{Name: "GetShardIterator"},
	{Name: "GetStream"},
	{Name: "ListStreams"},
}
