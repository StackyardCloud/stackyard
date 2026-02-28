package server

type licenseManagerOperation struct {
	Name string
}

// AWS License Manager operations sourced from:
// https://docs.aws.amazon.com/license-manager/latest/APIReference/API_Operations.html
var licenseManagerOperations = []licenseManagerOperation{
	{Name: "AcceptGrant"},
	{Name: "CheckInLicense"},
	{Name: "CheckoutBorrowLicense"},
	{Name: "CheckoutLicense"},
	{Name: "CreateGrant"},
	{Name: "CreateGrantVersion"},
	{Name: "CreateLicense"},
	{Name: "CreateLicenseAssetGroup"},
	{Name: "CreateLicenseAssetRuleset"},
	{Name: "CreateLicenseConfiguration"},
	{Name: "CreateLicenseConversionTaskForResource"},
	{Name: "CreateLicenseManagerReportGenerator"},
	{Name: "CreateLicenseVersion"},
	{Name: "CreateToken"},
	{Name: "DeleteGrant"},
	{Name: "DeleteLicense"},
	{Name: "DeleteLicenseAssetGroup"},
	{Name: "DeleteLicenseAssetRuleset"},
	{Name: "DeleteLicenseConfiguration"},
	{Name: "DeleteLicenseManagerReportGenerator"},
	{Name: "DeleteToken"},
	{Name: "ExtendLicenseConsumption"},
	{Name: "GetAccessToken"},
	{Name: "GetGrant"},
	{Name: "GetLicense"},
	{Name: "GetLicenseAssetGroup"},
	{Name: "GetLicenseAssetRuleset"},
	{Name: "GetLicenseConfiguration"},
	{Name: "GetLicenseConversionTask"},
	{Name: "GetLicenseManagerReportGenerator"},
	{Name: "GetLicenseUsage"},
	{Name: "GetServiceSettings"},
	{Name: "ListAssetsForLicenseAssetGroup"},
	{Name: "ListAssociationsForLicenseConfiguration"},
	{Name: "ListDistributedGrants"},
	{Name: "ListFailuresForLicenseConfigurationOperations"},
	{Name: "ListLicenseAssetGroups"},
	{Name: "ListLicenseAssetRulesets"},
	{Name: "ListLicenseConfigurations"},
	{Name: "ListLicenseConfigurationsForOrganization"},
	{Name: "ListLicenseConversionTasks"},
	{Name: "ListLicenseManagerReportGenerators"},
	{Name: "ListLicenseSpecificationsForResource"},
	{Name: "ListLicenseVersions"},
	{Name: "ListLicenses"},
	{Name: "ListReceivedGrants"},
	{Name: "ListReceivedGrantsForOrganization"},
	{Name: "ListReceivedLicenses"},
	{Name: "ListReceivedLicensesForOrganization"},
	{Name: "ListResourceInventory"},
	{Name: "ListTagsForResource"},
	{Name: "ListTokens"},
	{Name: "ListUsageForLicenseConfiguration"},
	{Name: "RejectGrant"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateLicenseAssetGroup"},
	{Name: "UpdateLicenseAssetRuleset"},
	{Name: "UpdateLicenseConfiguration"},
	{Name: "UpdateLicenseManagerReportGenerator"},
	{Name: "UpdateLicenseSpecificationsForResource"},
	{Name: "UpdateServiceSettings"},
}

var licenseManagerOperationByName = func() map[string]licenseManagerOperation {
	out := make(map[string]licenseManagerOperation, len(licenseManagerOperations))
	for _, op := range licenseManagerOperations {
		out[op.Name] = op
	}
	return out
}()
