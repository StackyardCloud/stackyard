package server

type pinpointResource struct {
	Name string
}

// Amazon Pinpoint resources sourced from:
// https://docs.aws.amazon.com/pinpoint/latest/apireference/resources.html
var pinpointResources = []pinpointResource{
	{Name: "Active Template Version"},
	{Name: "ADM Channel"},
	{Name: "APNs Channel"},
	{Name: "APNs Sandbox Channel"},
	{Name: "APNs VoIP Channel"},
	{Name: "APNs VoIP Sandbox Channel"},
	{Name: "App"},
	{Name: "Application Metrics"},
	{Name: "Apps"},
	{Name: "Attributes"},
	{Name: "Baidu Channel"},
	{Name: "Campaign"},
	{Name: "Campaign Activities"},
	{Name: "Campaign Metrics"},
	{Name: "Campaign Version"},
	{Name: "Campaign Versions"},
	{Name: "Campaigns"},
	{Name: "Channels"},
	{Name: "Email Channel"},
	{Name: "Email Template"},
	{Name: "Endpoint"},
	{Name: "Endpoints"},
	{Name: "Event Stream"},
	{Name: "Events"},
	{Name: "Export Job"},
	{Name: "Export Jobs"},
	{Name: "GCM Channel"},
	{Name: "Import Job"},
	{Name: "Import Jobs"},
	{Name: "In-App Messages"},
	{Name: "In-App Template"},
	{Name: "Journey"},
	{Name: "Journey Activity Execution Metrics"},
	{Name: "Journey Engagement Metrics"},
	{Name: "Journey Execution Metrics"},
	{Name: "Journey Run Execution Activity Metrics"},
	{Name: "Journey Run Execution Metrics"},
	{Name: "Journey Runs"},
	{Name: "Journey State"},
	{Name: "Journeys"},
	{Name: "Messages"},
	{Name: "OTP Message"},
	{Name: "Phone Number Validate"},
	{Name: "Push Notification Template"},
	{Name: "Recommender Model"},
	{Name: "Recommender Models"},
	{Name: "Segment"},
	{Name: "Segment Export Jobs"},
	{Name: "Segment Import Jobs"},
	{Name: "Segment Version"},
	{Name: "Segment Versions"},
	{Name: "Segments"},
	{Name: "Settings"},
	{Name: "SMS Channel"},
	{Name: "SMS Template"},
	{Name: "Tags"},
	{Name: "Template Versions"},
	{Name: "Templates"},
	{Name: "User"},
	{Name: "Users Messages"},
	{Name: "Verify OTP"},
	{Name: "Voice Channel"},
	{Name: "Voice Template"},
}

var pinpointResourceByName = func() map[string]pinpointResource {
	out := make(map[string]pinpointResource, len(pinpointResources))
	for _, r := range pinpointResources {
		out[r.Name] = r
	}
	return out
}()
