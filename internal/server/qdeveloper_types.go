package server

type qDeveloperDataType struct {
	Name string
}

// AWS Q Developer in chat applications (AWS Chatbot API) data types sourced from:
// https://docs.aws.amazon.com/chatbot/latest/APIReference/API_Types.html
var qDeveloperDataTypes = []qDeveloperDataType{
	{Name: "AccountPreferences"},
	{Name: "AssociationListing"},
	{Name: "ChimeWebhookConfiguration"},
	{Name: "ConfiguredTeam"},
	{Name: "CustomAction"},
	{Name: "CustomActionAttachment"},
	{Name: "CustomActionAttachmentCriteria"},
	{Name: "CustomActionDefinition"},
	{Name: "SlackChannelConfiguration"},
	{Name: "SlackUserIdentity"},
	{Name: "SlackWorkspace"},
	{Name: "Tag"},
	{Name: "TeamsChannelConfiguration"},
	{Name: "TeamsUserIdentity"},
	{Name: "UpdateSlackChannelConfiguration"},
}

var qDeveloperDataTypeByName = func() map[string]qDeveloperDataType {
	out := make(map[string]qDeveloperDataType, len(qDeveloperDataTypes))
	for _, dt := range qDeveloperDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
