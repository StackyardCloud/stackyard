package server

// Amazon Chime SDK Meetings operations sourced from:
// https://docs.aws.amazon.com/chime-sdk/latest/APIReference/API_Operations_Amazon_Chime_SDK_Meetings.html
var chimeSDKMeetingsOperations = []chimeOperation{
	{Name: "BatchCreateAttendee", Method: "POST", URI: "/meetings/{MeetingId}/attendees?operation=batch-create"},
	{Name: "BatchUpdateAttendeeCapabilitiesExcept", Method: "PUT", URI: "/meetings/{MeetingId}/attendees/capabilities?operation=batch-update-except"},
	{Name: "CreateAttendee", Method: "POST", URI: "/meetings/{MeetingId}/attendees"},
	{Name: "CreateMeeting", Method: "POST", URI: "/meetings"},
	{Name: "CreateMeetingWithAttendees", Method: "POST", URI: "/meetings?operation=create-attendees"},
	{Name: "DeleteAttendee", Method: "DELETE", URI: "/meetings/{MeetingId}/attendees/{AttendeeId}", Code: 204},
	{Name: "DeleteMeeting", Method: "DELETE", URI: "/meetings/{MeetingId}", Code: 204},
	{Name: "GetAttendee", Method: "GET", URI: "/meetings/{MeetingId}/attendees/{AttendeeId}"},
	{Name: "GetMeeting", Method: "GET", URI: "/meetings/{MeetingId}"},
	{Name: "ListAttendees", Method: "GET", URI: "/meetings/{MeetingId}/attendees"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags"},
	{Name: "StartMeetingTranscription", Method: "POST", URI: "/meetings/{MeetingId}/transcription?operation=start"},
	{Name: "StopMeetingTranscription", Method: "POST", URI: "/meetings/{MeetingId}/transcription?operation=stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags?operation=tag-resource", Code: 204},
	{Name: "UntagResource", Method: "POST", URI: "/tags?operation=untag-resource", Code: 204},
	{Name: "UpdateAttendeeCapabilities", Method: "PUT", URI: "/meetings/{MeetingId}/attendees/{AttendeeId}/capabilities"},
}
