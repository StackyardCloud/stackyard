package server

type singleSignOnDataType struct {
	Name string
}

// AWS IAM Identity Center (sso-admin) data types sourced from:
// https://docs.aws.amazon.com/singlesignon/latest/APIReference/API_Types.html
var singleSignOnDataTypes = []singleSignOnDataType{
	{Name: "AccessControlAttribute"},
	{Name: "AccessControlAttributeValue"},
	{Name: "AccountAssignment"},
	{Name: "AccountAssignmentForPrincipal"},
	{Name: "AccountAssignmentOperationStatus"},
	{Name: "AccountAssignmentOperationStatusMetadata"},
	{Name: "Application"},
	{Name: "ApplicationAssignment"},
	{Name: "ApplicationAssignmentForPrincipal"},
	{Name: "ApplicationProvider"},
	{Name: "AttachedManagedPolicy"},
	{Name: "AuthenticationMethod"},
	{Name: "AuthenticationMethodItem"},
	{Name: "AuthorizationCodeGrant"},
	{Name: "AuthorizedTokenIssuer"},
	{Name: "CustomerManagedPolicyReference"},
	{Name: "DisplayData"},
	{Name: "EncryptionConfiguration"},
	{Name: "EncryptionConfigurationDetails"},
	{Name: "Grant"},
	{Name: "GrantItem"},
	{Name: "IamAuthenticationMethod"},
	{Name: "InstanceAccessControlAttributeConfiguration"},
	{Name: "InstanceMetadata"},
	{Name: "JwtBearerGrant"},
	{Name: "ListAccountAssignmentsFilter"},
	{Name: "ListApplicationAssignmentsFilter"},
	{Name: "ListApplicationsFilter"},
	{Name: "OidcJwtConfiguration"},
	{Name: "OidcJwtUpdateConfiguration"},
	{Name: "OperationStatusFilter"},
	{Name: "PermissionSet"},
	{Name: "PermissionSetProvisioningStatus"},
	{Name: "PermissionSetProvisioningStatusMetadata"},
	{Name: "PermissionsBoundary"},
	{Name: "PortalOptions"},
	{Name: "RefreshTokenGrant"},
	{Name: "RegionMetadata"},
	{Name: "ResourceServerConfig"},
	{Name: "ResourceServerScopeDetails"},
	{Name: "ScopeDetails"},
	{Name: "SignInOptions"},
	{Name: "Tag"},
	{Name: "TokenExchangeGrant"},
	{Name: "TrustedTokenIssuerConfiguration"},
	{Name: "TrustedTokenIssuerMetadata"},
	{Name: "TrustedTokenIssuerUpdateConfiguration"},
	{Name: "UpdateApplicationPortalOptions"},
	{Name: "UpdateTrustedTokenIssuer"},
}

var singleSignOnDataTypeByName = func() map[string]singleSignOnDataType {
	out := make(map[string]singleSignOnDataType, len(singleSignOnDataTypes))
	for _, dt := range singleSignOnDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
