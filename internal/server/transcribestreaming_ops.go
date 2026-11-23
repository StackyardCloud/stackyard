package server

type transcribestreamingOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Transcribe Streaming operations sourced from:
// https://docs.aws.amazon.com/transcribe/latest/APIReference/API_Operations_Amazon_Transcribe_Streaming_Service.html
var transcribestreamingOperations = []transcribestreamingOperation{
	{Name: "GetMedicalScribeStream", Method: "GET", URI: "/medical-scribe-stream/{SessionId}"},
	{Name: "StartCallAnalyticsStreamTranscription", Method: "POST", URI: "/call-analytics-stream-transcription"},
	{Name: "StartMedicalScribeStream", Method: "POST", URI: "/medical-scribe-stream"},
	{Name: "StartMedicalStreamTranscription", Method: "POST", URI: "/medical-stream-transcription"},
	{Name: "StartStreamTranscription", Method: "POST", URI: "/stream-transcription"},
}
