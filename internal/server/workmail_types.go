package server

type workmailDataType struct {
	Name string
}

// Amazon WorkMail data types sourced from:
// https://docs.aws.amazon.com/workmail/latest/APIReference/API_Types.html
var workmailDataTypes = []workmailDataType{
	{Name: "AccessControlRule"},
	{Name: "AvailabilityConfiguration"},
	{Name: "BookingOptions"},
	{Name: "Delegate"},
	{Name: "DnsRecord"},
	{Name: "Domain"},
	{Name: "EwsAvailabilityProvider"},
	{Name: "FolderConfiguration"},
	{Name: "Group"},
	{Name: "GroupIdentifier"},
	{Name: "IdentityCenterConfiguration"},
	{Name: "ImpersonationMatchedRule"},
	{Name: "ImpersonationRole"},
	{Name: "ImpersonationRule"},
	{Name: "LambdaAvailabilityProvider"},
	{Name: "ListGroupsFilters"},
	{Name: "ListGroupsForEntityFilters"},
	{Name: "ListResourcesFilters"},
	{Name: "ListUsersFilters"},
	{Name: "MailDomainSummary"},
	{Name: "MailboxExportJob"},
	{Name: "Member"},
	{Name: "MobileDeviceAccessMatchedRule"},
	{Name: "MobileDeviceAccessOverride"},
	{Name: "MobileDeviceAccessRule"},
	{Name: "OrganizationSummary"},
	{Name: "Permission"},
	{Name: "PersonalAccessTokenConfiguration"},
	{Name: "PersonalAccessTokenSummary"},
	{Name: "RedactedEwsAvailabilityProvider"},
	{Name: "Resource"},
	{Name: "Tag"},
	{Name: "User"},
}

var workmailDataTypeByName = func() map[string]workmailDataType {
	out := make(map[string]workmailDataType, len(workmailDataTypes))
	for _, dt := range workmailDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
