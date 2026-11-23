package server

type pollyOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Polly operations sourced from:
// https://docs.aws.amazon.com/polly/latest/dg/API_Operations.html
var pollyOperations = []pollyOperation{
	{Name: "CreateVoice", Method: "POST", URI: "/v1/voices"},
	{Name: "DeleteLexicon", Method: "DELETE", URI: "/v1/lexicons/{LexiconName}"},
	{Name: "DeleteVoice", Method: "DELETE", URI: "/v1/voices/{VoiceId}"},
	{Name: "DescribeVoices", Method: "GET", URI: "/v1/voices"},
	{Name: "GetLexicon", Method: "GET", URI: "/v1/lexicons/{LexiconName}"},
	{Name: "GetSpeechSynthesisTask", Method: "GET", URI: "/v1/synthesisTasks/{TaskId}"},
	{Name: "GetVoice", Method: "GET", URI: "/v1/voices/{VoiceId}"},
	{Name: "ListLexicons", Method: "GET", URI: "/v1/lexicons"},
	{Name: "ListSpeechSynthesisTasks", Method: "GET", URI: "/v1/synthesisTasks"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/v1/listTagsForResource/"},
	{Name: "PutLexicon", Method: "PUT", URI: "/v1/lexicons/{LexiconName}"},
	{Name: "StartSpeechSynthesisStream", Method: "POST", URI: "/v1/synthesisStream"},
	{Name: "StartSpeechSynthesisTask", Method: "POST", URI: "/v1/synthesisTasks"},
	{Name: "SynthesizeSpeech", Method: "POST", URI: "/v1/speech"},
	{Name: "TagResource", Method: "POST", URI: "/v1/tagResource"},
	{Name: "UntagResource", Method: "POST", URI: "/v1/untagResource"},
}
