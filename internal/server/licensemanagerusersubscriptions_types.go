package server

type licenseManagerUserSubscriptionsDataType struct {
	Name string
}

// AWS License Manager User Subscriptions data types sourced from:
// https://docs.aws.amazon.com/license-manager-user-subscriptions/latest/APIReference/API_Types.html
var licenseManagerUserSubscriptionsDataTypes = []licenseManagerUserSubscriptionsDataType{
	{Name: "ActiveDirectoryIdentityProvider"},
	{Name: "ActiveDirectorySettings"},
	{Name: "CredentialsProvider"},
	{Name: "DomainNetworkSettings"},
	{Name: "Filter"},
	{Name: "IdentityProvider"},
	{Name: "IdentityProviderSummary"},
	{Name: "InstanceSummary"},
	{Name: "InstanceUserSummary"},
	{Name: "LicenseServer"},
	{Name: "LicenseServerEndpoint"},
	{Name: "LicenseServerSettings"},
	{Name: "ProductUserSummary"},
	{Name: "RdsSalSettings"},
	{Name: "SecretsManagerCredentialsProvider"},
	{Name: "ServerEndpoint"},
	{Name: "ServerSettings"},
	{Name: "Settings"},
	{Name: "UpdateIdentityProviderSettings"},
	{Name: "UpdateSettings"},
}

var licenseManagerUserSubscriptionsDataTypeByName = func() map[string]licenseManagerUserSubscriptionsDataType {
	out := make(map[string]licenseManagerUserSubscriptionsDataType, len(licenseManagerUserSubscriptionsDataTypes))
	for _, dt := range licenseManagerUserSubscriptionsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
