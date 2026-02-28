package server

type workspacesWebOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon WorkSpaces Secure Browser actions sourced from:

// https://docs.aws.amazon.com/workspaces-web/latest/APIReference/API_Operations.html

var workspacesWebOperations = []workspacesWebOperation{
	{Name: "AssociateBrowserSettings", Method: "PUT", URI: "/portals/{portalArn+}/browserSettings?browserSettingsArn={browserSettingsArn}"},
	{Name: "AssociateDataProtectionSettings", Method: "PUT", URI: "/portals/{portalArn+}/dataProtectionSettings?dataProtectionSettingsArn={dataProtectionSettingsArn}"},
	{Name: "AssociateIpAccessSettings", Method: "PUT", URI: "/portals/{portalArn+}/ipAccessSettings?ipAccessSettingsArn={ipAccessSettingsArn}"},
	{Name: "AssociateNetworkSettings", Method: "PUT", URI: "/portals/{portalArn+}/networkSettings?networkSettingsArn={networkSettingsArn}"},
	{Name: "AssociateSessionLogger", Method: "PUT", URI: "/portals/{portalArn+}/sessionLogger?sessionLoggerArn={sessionLoggerArn}"},
	{Name: "AssociateTrustStore", Method: "PUT", URI: "/portals/{portalArn+}/trustStores?trustStoreArn={trustStoreArn}"},
	{Name: "AssociateUserAccessLoggingSettings", Method: "PUT", URI: "/portals/{portalArn+}/userAccessLoggingSettings?userAccessLoggingSettingsArn={userAccessLoggingSettingsArn}"},
	{Name: "AssociateUserSettings", Method: "PUT", URI: "/portals/{portalArn+}/userSettings?userSettingsArn={userSettingsArn}"},
	{Name: "CreateBrowserSettings", Method: "POST", URI: "/browserSettings"},
	{Name: "CreateDataProtectionSettings", Method: "POST", URI: "/dataProtectionSettings"},
	{Name: "CreateIdentityProvider", Method: "POST", URI: "/identityProviders"},
	{Name: "CreateIpAccessSettings", Method: "POST", URI: "/ipAccessSettings"},
	{Name: "CreateNetworkSettings", Method: "POST", URI: "/networkSettings"},
	{Name: "CreatePortal", Method: "POST", URI: "/portals"},
	{Name: "CreateSessionLogger", Method: "POST", URI: "/sessionLoggers"},
	{Name: "CreateTrustStore", Method: "POST", URI: "/trustStores"},
	{Name: "CreateUserAccessLoggingSettings", Method: "POST", URI: "/userAccessLoggingSettings"},
	{Name: "CreateUserSettings", Method: "POST", URI: "/userSettings"},
	{Name: "DeleteBrowserSettings", Method: "DELETE", URI: "/browserSettings/{browserSettingsArn+}"},
	{Name: "DeleteDataProtectionSettings", Method: "DELETE", URI: "/dataProtectionSettings/{dataProtectionSettingsArn+}"},
	{Name: "DeleteIdentityProvider", Method: "DELETE", URI: "/identityProviders/{identityProviderArn+}"},
	{Name: "DeleteIpAccessSettings", Method: "DELETE", URI: "/ipAccessSettings/{ipAccessSettingsArn+}"},
	{Name: "DeleteNetworkSettings", Method: "DELETE", URI: "/networkSettings/{networkSettingsArn+}"},
	{Name: "DeletePortal", Method: "DELETE", URI: "/portals/{portalArn+}"},
	{Name: "DeleteSessionLogger", Method: "DELETE", URI: "/sessionLoggers/{sessionLoggerArn+}"},
	{Name: "DeleteTrustStore", Method: "DELETE", URI: "/trustStores/{trustStoreArn+}"},
	{Name: "DeleteUserAccessLoggingSettings", Method: "DELETE", URI: "/userAccessLoggingSettings/{userAccessLoggingSettingsArn+}"},
	{Name: "DeleteUserSettings", Method: "DELETE", URI: "/userSettings/{userSettingsArn+}"},
	{Name: "DisassociateBrowserSettings", Method: "DELETE", URI: "/portals/{portalArn+}/browserSettings"},
	{Name: "DisassociateDataProtectionSettings", Method: "DELETE", URI: "/portals/{portalArn+}/dataProtectionSettings"},
	{Name: "DisassociateIpAccessSettings", Method: "DELETE", URI: "/portals/{portalArn+}/ipAccessSettings"},
	{Name: "DisassociateNetworkSettings", Method: "DELETE", URI: "/portals/{portalArn+}/networkSettings"},
	{Name: "DisassociateSessionLogger", Method: "DELETE", URI: "/portals/{portalArn+}/sessionLogger"},
	{Name: "DisassociateTrustStore", Method: "DELETE", URI: "/portals/{portalArn+}/trustStores"},
	{Name: "DisassociateUserAccessLoggingSettings", Method: "DELETE", URI: "/portals/{portalArn+}/userAccessLoggingSettings"},
	{Name: "DisassociateUserSettings", Method: "DELETE", URI: "/portals/{portalArn+}/userSettings"},
	{Name: "ExpireSession", Method: "DELETE", URI: "/portals/{portalId}/sessions/{sessionId}"},
	{Name: "GetBrowserSettings", Method: "GET", URI: "/browserSettings/{browserSettingsArn+}"},
	{Name: "GetDataProtectionSettings", Method: "GET", URI: "/dataProtectionSettings/{dataProtectionSettingsArn+}"},
	{Name: "GetIdentityProvider", Method: "GET", URI: "/identityProviders/{identityProviderArn+}"},
	{Name: "GetIpAccessSettings", Method: "GET", URI: "/ipAccessSettings/{ipAccessSettingsArn+}"},
	{Name: "GetNetworkSettings", Method: "GET", URI: "/networkSettings/{networkSettingsArn+}"},
	{Name: "GetPortal", Method: "GET", URI: "/portals/{portalArn+}"},
	{Name: "GetPortalServiceProviderMetadata", Method: "GET", URI: "/portalIdp/{portalArn+}"},
	{Name: "GetSession", Method: "GET", URI: "/portals/{portalId}/sessions/{sessionId}"},
	{Name: "GetSessionLogger", Method: "GET", URI: "/sessionLoggers/{sessionLoggerArn+}"},
	{Name: "GetTrustStore", Method: "GET", URI: "/trustStores/{trustStoreArn+}"},
	{Name: "GetTrustStoreCertificate", Method: "GET", URI: "/trustStores/{trustStoreArn+}/certificate?thumbprint={thumbprint}"},
	{Name: "GetUserAccessLoggingSettings", Method: "GET", URI: "/userAccessLoggingSettings/{userAccessLoggingSettingsArn+}"},
	{Name: "GetUserSettings", Method: "GET", URI: "/userSettings/{userSettingsArn+}"},
	{Name: "ListBrowserSettings", Method: "GET", URI: "/browserSettings?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListDataProtectionSettings", Method: "GET", URI: "/dataProtectionSettings?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListIdentityProviders", Method: "GET", URI: "/portals/{portalArn+}/identityProviders?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListIpAccessSettings", Method: "GET", URI: "/ipAccessSettings?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListNetworkSettings", Method: "GET", URI: "/networkSettings?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListPortals", Method: "GET", URI: "/portals?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListSessionLoggers", Method: "GET", URI: "/sessionLoggers?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListSessions", Method: "GET", URI: "/portals/{portalId}/sessions?maxResults={maxResults}&nextToken={nextToken}&sessionId={sessionId}&sortBy={sortBy}&status={status}&username={username}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn+}"},
	{Name: "ListTrustStoreCertificates", Method: "GET", URI: "/trustStores/{trustStoreArn+}/certificates?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTrustStores", Method: "GET", URI: "/trustStores?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListUserAccessLoggingSettings", Method: "GET", URI: "/userAccessLoggingSettings?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListUserSettings", Method: "GET", URI: "/userSettings?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn+}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn+}?tagKeys={tagKeys}"},
	{Name: "UpdateBrowserSettings", Method: "PATCH", URI: "/browserSettings/{browserSettingsArn+}"},
	{Name: "UpdateDataProtectionSettings", Method: "PATCH", URI: "/dataProtectionSettings/{dataProtectionSettingsArn+}"},
	{Name: "UpdateIdentityProvider", Method: "PATCH", URI: "/identityProviders/{identityProviderArn+}"},
	{Name: "UpdateIpAccessSettings", Method: "PATCH", URI: "/ipAccessSettings/{ipAccessSettingsArn+}"},
	{Name: "UpdateNetworkSettings", Method: "PATCH", URI: "/networkSettings/{networkSettingsArn+}"},
	{Name: "UpdatePortal", Method: "PUT", URI: "/portals/{portalArn+}"},
	{Name: "UpdateSessionLogger", Method: "POST", URI: "/sessionLoggers/{sessionLoggerArn+}"},
	{Name: "UpdateTrustStore", Method: "PATCH", URI: "/trustStores/{trustStoreArn+}"},
	{Name: "UpdateUserAccessLoggingSettings", Method: "PATCH", URI: "/userAccessLoggingSettings/{userAccessLoggingSettingsArn+}"},
	{Name: "UpdateUserSettings", Method: "PATCH", URI: "/userSettings/{userSettingsArn+}"},
}

var workspacesWebOperationByName = func() map[string]workspacesWebOperation {
	out := make(map[string]workspacesWebOperation, len(workspacesWebOperations))
	for _, op := range workspacesWebOperations {
		out[op.Name] = op
	}
	return out
}()
