package server

type supportAppOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Support App in Slack operations sourced from:
// https://docs.aws.amazon.com/supportapp/latest/APIReference/API_Operations.html
var supportAppOperations = []supportAppOperation{
	{Name: "CreateSlackChannelConfiguration", Method: "POST", URI: "/control/create-slack-channel-configuration"},
	{Name: "DeleteAccountAlias", Method: "POST", URI: "/control/delete-account-alias"},
	{Name: "DeleteSlackChannelConfiguration", Method: "POST", URI: "/control/delete-slack-channel-configuration"},
	{Name: "DeleteSlackWorkspaceConfiguration", Method: "POST", URI: "/control/delete-slack-workspace-configuration"},
	{Name: "GetAccountAlias", Method: "POST", URI: "/control/get-account-alias"},
	{Name: "ListSlackChannelConfigurations", Method: "POST", URI: "/control/list-slack-channel-configurations"},
	{Name: "ListSlackWorkspaceConfigurations", Method: "POST", URI: "/control/list-slack-workspace-configurations"},
	{Name: "PutAccountAlias", Method: "POST", URI: "/control/put-account-alias"},
	{Name: "RegisterSlackWorkspaceForOrganization", Method: "POST", URI: "/control/register-slack-workspace-for-organization"},
	{Name: "UpdateSlackChannelConfiguration", Method: "POST", URI: "/control/update-slack-channel-configuration"},
}

var supportAppOperationByName = func() map[string]supportAppOperation {
	out := make(map[string]supportAppOperation, len(supportAppOperations))
	for _, op := range supportAppOperations {
		out[op.Name] = op
	}
	return out
}()
