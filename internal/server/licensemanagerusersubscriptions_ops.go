package server

type licenseManagerUserSubscriptionsOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS License Manager User Subscriptions operations sourced from:
// https://docs.aws.amazon.com/license-manager-user-subscriptions/latest/APIReference/API_Operations.html
var licenseManagerUserSubscriptionsOperations = []licenseManagerUserSubscriptionsOperation{
	{Name: "AssociateUser", Method: "POST", URI: "/user/AssociateUser"},
	{Name: "CreateLicenseServerEndpoint", Method: "POST", URI: "/license-server/CreateLicenseServerEndpoint"},
	{Name: "DeleteLicenseServerEndpoint", Method: "POST", URI: "/license-server/DeleteLicenseServerEndpoint"},
	{Name: "DeregisterIdentityProvider", Method: "POST", URI: "/identity-provider/DeregisterIdentityProvider"},
	{Name: "DisassociateUser", Method: "POST", URI: "/user/DisassociateUser"},
	{Name: "ListIdentityProviders", Method: "POST", URI: "/identity-provider/ListIdentityProviders"},
	{Name: "ListInstances", Method: "POST", URI: "/instance/ListInstances"},
	{Name: "ListLicenseServerEndpoints", Method: "POST", URI: "/license-server/ListLicenseServerEndpoints"},
	{Name: "ListProductSubscriptions", Method: "POST", URI: "/user/ListProductSubscriptions"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "ListUserAssociations", Method: "POST", URI: "/user/ListUserAssociations"},
	{Name: "RegisterIdentityProvider", Method: "POST", URI: "/identity-provider/RegisterIdentityProvider"},
	{Name: "StartProductSubscription", Method: "POST", URI: "/user/StartProductSubscription"},
	{Name: "StopProductSubscription", Method: "POST", URI: "/user/StopProductSubscription"},
	{Name: "TagResource", Method: "PUT", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateIdentityProviderSettings", Method: "POST", URI: "/identity-provider/UpdateIdentityProviderSettings"},
}

var licenseManagerUserSubscriptionsOperationByName = func() map[string]licenseManagerUserSubscriptionsOperation {
	out := make(map[string]licenseManagerUserSubscriptionsOperation, len(licenseManagerUserSubscriptionsOperations))
	for _, op := range licenseManagerUserSubscriptionsOperations {
		out[op.Name] = op
	}
	return out
}()
