package server

type iamDataType struct {
	Name string
}

// AWS Identity and Access Management (IAM) data types sourced from:
// https://docs.aws.amazon.com/IAM/latest/APIReference/API_Types.html
var iamDataTypes = []iamDataType{
	{Name: "AccessDetail"},
	{Name: "AccessKey"},
	{Name: "AccessKeyLastUsed"},
	{Name: "AccessKeyMetadata"},
	{Name: "AttachedPermissionsBoundary"},
	{Name: "AttachedPolicy"},
	{Name: "ContextEntry"},
	{Name: "DelegationPermission"},
	{Name: "DelegationRequest"},
	{Name: "DeletionTaskFailureReasonType"},
	{Name: "EntityDetails"},
	{Name: "EntityInfo"},
	{Name: "ErrorDetails"},
	{Name: "EvaluationResult"},
	{Name: "Group"},
	{Name: "GroupDetail"},
	{Name: "InstanceProfile"},
	{Name: "ListPoliciesGrantingServiceAccessEntry"},
	{Name: "LoginProfile"},
	{Name: "MFADevice"},
	{Name: "ManagedPolicyDetail"},
	{Name: "OpenIDConnectProviderListEntry"},
	{Name: "OrganizationsDecisionDetail"},
	{Name: "PasswordPolicy"},
	{Name: "PermissionsBoundaryDecisionDetail"},
	{Name: "Policy"},
	{Name: "PolicyDetail"},
	{Name: "PolicyGrantingServiceAccess"},
	{Name: "PolicyGroup"},
	{Name: "PolicyParameter"},
	{Name: "PolicyRole"},
	{Name: "PolicyUser"},
	{Name: "PolicyVersion"},
	{Name: "Position"},
	{Name: "ResourceSpecificResult"},
	{Name: "Role"},
	{Name: "RoleDetail"},
	{Name: "RoleLastUsed"},
	{Name: "RoleUsageType"},
	{Name: "SAMLPrivateKey"},
	{Name: "SAMLProviderListEntry"},
	{Name: "SSHPublicKey"},
	{Name: "SSHPublicKeyMetadata"},
	{Name: "ServerCertificate"},
	{Name: "ServerCertificateMetadata"},
	{Name: "ServiceLastAccessed"},
	{Name: "ServiceSpecificCredential"},
	{Name: "ServiceSpecificCredentialMetadata"},
	{Name: "SigningCertificate"},
	{Name: "Statement"},
	{Name: "Tag"},
	{Name: "TrackedActionLastAccessed"},
	{Name: "Types"},
	{Name: "UploadSSHPublicKey"},
	{Name: "User"},
	{Name: "UserDetail"},
	{Name: "VirtualMFADevice"},
}

var iamDataTypeByName = func() map[string]iamDataType {
	out := make(map[string]iamDataType, len(iamDataTypes))
	for _, dt := range iamDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
