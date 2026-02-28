package server

type wickrOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Wickr operations sourced from:
// https://docs.aws.amazon.com/wickr/latest/APIReference/API_Operations.html
var wickrOperations = []wickrOperation{
	{Name: "BatchCreateUser", Method: "POST", URI: "/networks/{networkId}/users"},
	{Name: "BatchDeleteUser", Method: "POST", URI: "/networks/{networkId}/users/batch-delete"},
	{Name: "BatchLookupUserUname", Method: "POST", URI: "/networks/{networkId}/users/uname-lookup"},
	{Name: "BatchReinviteUser", Method: "PATCH", URI: "/networks/{networkId}/users/re-invite"},
	{Name: "BatchResetDevicesForUser", Method: "PATCH", URI: "/networks/{networkId}/users/{userId}/devices"},
	{Name: "BatchToggleUserSuspendStatus", Method: "PATCH", URI: "/networks/{networkId}/users/toggleSuspend?suspend={suspend}"},
	{Name: "CreateBot", Method: "POST", URI: "/networks/{networkId}/bots"},
	{Name: "CreateDataRetentionBot", Method: "POST", URI: "/networks/{networkId}/data-retention-bots"},
	{Name: "CreateDataRetentionBotChallenge", Method: "POST", URI: "/networks/{networkId}/data-retention-bots/challenge"},
	{Name: "CreateNetwork", Method: "POST", URI: "/networks"},
	{Name: "CreateSecurityGroup", Method: "POST", URI: "/networks/{networkId}/security-groups"},
	{Name: "DeleteBot", Method: "DELETE", URI: "/networks/{networkId}/bots/{botId}"},
	{Name: "DeleteDataRetentionBot", Method: "DELETE", URI: "/networks/{networkId}/data-retention-bots"},
	{Name: "DeleteNetwork", Method: "DELETE", URI: "/networks/{networkId}"},
	{Name: "DeleteSecurityGroup", Method: "DELETE", URI: "/networks/{networkId}/security-groups/{groupId}"},
	{Name: "GetBot", Method: "GET", URI: "/networks/{networkId}/bots/{botId}"},
	{Name: "GetBotsCount", Method: "GET", URI: "/networks/{networkId}/bots/count"},
	{Name: "GetDataRetentionBot", Method: "GET", URI: "/networks/{networkId}/data-retention-bots"},
	{Name: "GetGuestUserHistoryCount", Method: "GET", URI: "/networks/{networkId}/guest-users/count"},
	{Name: "GetNetwork", Method: "GET", URI: "/networks/{networkId}"},
	{Name: "GetNetworkSettings", Method: "GET", URI: "/networks/{networkId}/settings"},
	{Name: "GetOidcInfo", Method: "GET", URI: "/networks/{networkId}/oidc?certificate={certificate}&clientId={clientId}&clientSecret={clientSecret}&code={code}&codeVerifier={codeVerifier}&grantType={grantType}&redirectUri={redirectUri}&url={url}"},
	{Name: "GetSecurityGroup", Method: "GET", URI: "/networks/{networkId}/security-groups/{groupId}"},
	{Name: "GetUser", Method: "GET", URI: "/networks/{networkId}/users/{userId}?endTime={endTime}&startTime={startTime}"},
	{Name: "GetUsersCount", Method: "GET", URI: "/networks/{networkId}/users/count"},
	{Name: "ListBlockedGuestUsers", Method: "GET", URI: "/networks/{networkId}/guest-users/blocklist?admin={admin}&maxResults={maxResults}&nextToken={nextToken}&sortDirection={sortDirection}&sortFields={sortFields}&username={username}"},
	{Name: "ListBots", Method: "GET", URI: "/networks/{networkId}/bots?displayName={displayName}&groupId={groupId}&maxResults={maxResults}&nextToken={nextToken}&sortDirection={sortDirection}&sortFields={sortFields}&status={status}&username={username}"},
	{Name: "ListDevicesForUser", Method: "GET", URI: "/networks/{networkId}/users/{userId}/devices?maxResults={maxResults}&nextToken={nextToken}&sortDirection={sortDirection}&sortFields={sortFields}"},
	{Name: "ListGuestUsers", Method: "GET", URI: "/networks/{networkId}/guest-users?billingPeriod={billingPeriod}&maxResults={maxResults}&nextToken={nextToken}&sortDirection={sortDirection}&sortFields={sortFields}&username={username}"},
	{Name: "ListNetworks", Method: "GET", URI: "/networks?maxResults={maxResults}&nextToken={nextToken}&sortDirection={sortDirection}&sortFields={sortFields}"},
	{Name: "ListSecurityGroups", Method: "GET", URI: "/networks/{networkId}/security-groups?maxResults={maxResults}&nextToken={nextToken}&sortDirection={sortDirection}&sortFields={sortFields}"},
	{Name: "ListSecurityGroupUsers", Method: "GET", URI: "/networks/{networkId}/security-groups/{groupId}/users?maxResults={maxResults}&nextToken={nextToken}&sortDirection={sortDirection}&sortFields={sortFields}"},
	{Name: "ListUsers", Method: "GET", URI: "/networks/{networkId}/users?firstName={firstName}&groupId={groupId}&lastName={lastName}&maxResults={maxResults}&nextToken={nextToken}&sortDirection={sortDirection}&sortFields={sortFields}&status={status}&username={username}"},
	{Name: "RegisterOidcConfig", Method: "POST", URI: "/networks/{networkId}/oidc/save"},
	{Name: "RegisterOidcConfigTest", Method: "POST", URI: "/networks/{networkId}/oidc/test"},
	{Name: "UpdateBot", Method: "PATCH", URI: "/networks/{networkId}/bots/{botId}"},
	{Name: "UpdateDataRetention", Method: "PATCH", URI: "/networks/{networkId}/data-retention-bots"},
	{Name: "UpdateGuestUser", Method: "PATCH", URI: "/networks/{networkId}/guest-users/{usernameHash}"},
	{Name: "UpdateNetwork", Method: "PATCH", URI: "/networks/{networkId}"},
	{Name: "UpdateNetworkSettings", Method: "PATCH", URI: "/networks/{networkId}/settings"},
	{Name: "UpdateSecurityGroup", Method: "PATCH", URI: "/networks/{networkId}/security-groups/{groupId}"},
	{Name: "UpdateUser", Method: "PATCH", URI: "/networks/{networkId}/users"},
}

var wickrOperationByName = func() map[string]wickrOperation {
	out := make(map[string]wickrOperation, len(wickrOperations))
	for _, op := range wickrOperations {
		out[op.Name] = op
	}
	return out
}()
