package server

type connectParticipantOperation struct{ Name string }

// Amazon Connect Participant Service operations sourced from:
// https://docs.aws.amazon.com/connect/latest/APIReference/API_Operations_Amazon_Connect_Participant_Service.html
var connectParticipantOperations = []connectParticipantOperation{
	{Name: "CancelParticipantAuthentication"},
	{Name: "CompleteAttachmentUpload"},
	{Name: "CreateParticipantConnection"},
	{Name: "DescribeView"},
	{Name: "DisconnectParticipant"},
	{Name: "GetAttachment"},
	{Name: "GetAuthenticationUrl"},
	{Name: "GetTranscript"},
	{Name: "SendEvent"},
	{Name: "SendMessage"},
	{Name: "StartAttachmentUpload"},
}
