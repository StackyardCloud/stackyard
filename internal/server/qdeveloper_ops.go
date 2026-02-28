package server

type qDeveloperOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Q Developer in chat applications (AWS Chatbot API) operations sourced from:
// https://docs.aws.amazon.com/chatbot/latest/APIReference/API_Operations.html
var qDeveloperOperations = []qDeveloperOperation{
	{Name: "AssociateToConfiguration", Method: "POST", URI: "/associate-to-configuration"},
	{Name: "CreateChimeWebhookConfiguration", Method: "POST", URI: "/create-chime-webhook-configuration"},
	{Name: "CreateCustomAction", Method: "POST", URI: "/create-custom-action"},
	{Name: "CreateMicrosoftTeamsChannelConfiguration", Method: "POST", URI: "/create-ms-teams-channel-configuration"},
	{Name: "CreateSlackChannelConfiguration", Method: "POST", URI: "/create-slack-channel-configuration"},
	{Name: "DeleteChimeWebhookConfiguration", Method: "POST", URI: "/delete-chime-webhook-configuration"},
	{Name: "DeleteCustomAction", Method: "POST", URI: "/delete-custom-action"},
	{Name: "DeleteMicrosoftTeamsChannelConfiguration", Method: "POST", URI: "/delete-ms-teams-channel-configuration"},
	{Name: "DeleteMicrosoftTeamsConfiguredTeam", Method: "POST", URI: "/delete-ms-teams-configured-teams"},
	{Name: "DeleteMicrosoftTeamsUserIdentity", Method: "POST", URI: "/delete-ms-teams-user-identity"},
	{Name: "DeleteSlackChannelConfiguration", Method: "POST", URI: "/delete-slack-channel-configuration"},
	{Name: "DeleteSlackUserIdentity", Method: "POST", URI: "/delete-slack-user-identity"},
	{Name: "DeleteSlackWorkspaceAuthorization", Method: "POST", URI: "/delete-slack-workspace-authorization"},
	{Name: "DescribeChimeWebhookConfigurations", Method: "POST", URI: "/describe-chime-webhook-configurations"},
	{Name: "DescribeSlackChannelConfigurations", Method: "POST", URI: "/describe-slack-channel-configurations"},
	{Name: "DescribeSlackUserIdentities", Method: "POST", URI: "/describe-slack-user-identities"},
	{Name: "DescribeSlackWorkspaces", Method: "POST", URI: "/describe-slack-workspaces"},
	{Name: "DisassociateFromConfiguration", Method: "POST", URI: "/disassociate-from-configuration"},
	{Name: "GetAccountPreferences", Method: "POST", URI: "/get-account-preferences"},
	{Name: "GetCustomAction", Method: "POST", URI: "/get-custom-action"},
	{Name: "GetMicrosoftTeamsChannelConfiguration", Method: "POST", URI: "/get-ms-teams-channel-configuration"},
	{Name: "ListAssociations", Method: "POST", URI: "/list-associations"},
	{Name: "ListCustomActions", Method: "POST", URI: "/list-custom-actions"},
	{Name: "ListMicrosoftTeamsChannelConfigurations", Method: "POST", URI: "/list-ms-teams-channel-configurations"},
	{Name: "ListMicrosoftTeamsConfiguredTeams", Method: "POST", URI: "/list-ms-teams-configured-teams"},
	{Name: "ListMicrosoftTeamsUserIdentities", Method: "POST", URI: "/list-ms-teams-user-identities"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/list-tags-for-resource"},
	{Name: "TagResource", Method: "POST", URI: "/tag-resource"},
	{Name: "UntagResource", Method: "POST", URI: "/untag-resource"},
	{Name: "UpdateAccountPreferences", Method: "POST", URI: "/update-account-preferences"},
	{Name: "UpdateChimeWebhookConfiguration", Method: "POST", URI: "/update-chime-webhook-configuration"},
	{Name: "UpdateCustomAction", Method: "POST", URI: "/update-custom-action"},
	{Name: "UpdateMicrosoftTeamsChannelConfiguration", Method: "POST", URI: "/update-ms-teams-channel-configuration"},
	{Name: "UpdateSlackChannelConfiguration", Method: "POST", URI: "/update-slack-channel-configuration"},
}

var qDeveloperOperationByName = func() map[string]qDeveloperOperation {
	out := make(map[string]qDeveloperOperation, len(qDeveloperOperations))
	for _, op := range qDeveloperOperations {
		out[op.Name] = op
	}
	return out
}()
