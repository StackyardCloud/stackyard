package server

// Amazon Pinpoint SMS and Voice (v1) operations sourced from:
// https://docs.aws.amazon.com/pinpoint-sms-voice/latest/APIReference/resources.html
var pinpointSMSVoiceOperations = []sesv2Operation{
	{Name: "CreateConfigurationSet", Method: "POST", Pattern: "/v1/sms-voice/configuration-sets"},
	{Name: "CreateConfigurationSetEventDestination", Method: "POST", Pattern: "/v1/sms-voice/configuration-sets/{ConfigurationSetName}/event-destinations"},
	{Name: "DeleteConfigurationSet", Method: "DELETE", Pattern: "/v1/sms-voice/configuration-sets/{ConfigurationSetName}"},
	{Name: "DeleteConfigurationSetEventDestination", Method: "DELETE", Pattern: "/v1/sms-voice/configuration-sets/{ConfigurationSetName}/event-destinations/{EventDestinationName}"},
	{Name: "GetConfigurationSetEventDestinations", Method: "GET", Pattern: "/v1/sms-voice/configuration-sets/{ConfigurationSetName}/event-destinations"},
	{Name: "ListConfigurationSets", Method: "GET", Pattern: "/v1/sms-voice/configuration-sets"},
	{Name: "SendVoiceMessage", Method: "POST", Pattern: "/v1/sms-voice/voice/message"},
	{Name: "UpdateConfigurationSetEventDestination", Method: "PUT", Pattern: "/v1/sms-voice/configuration-sets/{ConfigurationSetName}/event-destinations/{EventDestinationName}"},
}
