package server

type licenseManagerDataType struct {
	Name string
}

// AWS License Manager data types sourced from:
// https://docs.aws.amazon.com/license-manager/latest/APIReference/API_Types.html
var licenseManagerDataTypes = []licenseManagerDataType{
	{Name: "AndRuleStatement"},
	{Name: "Asset"},
	{Name: "AutomatedDiscoveryInformation"},
	{Name: "BorrowConfiguration"},
	{Name: "ConsumedLicenseSummary"},
	{Name: "ConsumptionConfiguration"},
	{Name: "CrossAccountDiscoveryServiceStatus"},
	{Name: "CrossRegionDiscoveryStatus"},
	{Name: "DatetimeRange"},
	{Name: "Entitlement"},
	{Name: "EntitlementData"},
	{Name: "EntitlementUsage"},
	{Name: "Filter"},
	{Name: "Grant"},
	{Name: "GrantedLicense"},
	{Name: "InstanceRuleStatement"},
	{Name: "InventoryFilter"},
	{Name: "Issuer"},
	{Name: "IssuerDetails"},
	{Name: "License"},
	{Name: "LicenseAssetGroup"},
	{Name: "LicenseAssetGroupConfiguration"},
	{Name: "LicenseAssetGroupProperty"},
	{Name: "LicenseAssetRule"},
	{Name: "LicenseAssetRuleset"},
	{Name: "LicenseConfiguration"},
	{Name: "LicenseConfigurationAssociation"},
	{Name: "LicenseConfigurationRuleStatement"},
	{Name: "LicenseConfigurationUsage"},
	{Name: "LicenseConversionContext"},
	{Name: "LicenseConversionTask"},
	{Name: "LicenseOperationFailure"},
	{Name: "LicenseRuleStatement"},
	{Name: "LicenseSpecification"},
	{Name: "LicenseUsage"},
	{Name: "ManagedResourceSummary"},
	{Name: "MatchingRuleStatement"},
	{Name: "Metadata"},
	{Name: "Options"},
	{Name: "OrRuleStatement"},
	{Name: "OrganizationConfiguration"},
	{Name: "ProductCodeListItem"},
	{Name: "ProductInformation"},
	{Name: "ProductInformationFilter"},
	{Name: "ProvisionalConfiguration"},
	{Name: "ReceivedMetadata"},
	{Name: "RegionStatus"},
	{Name: "ReportContext"},
	{Name: "ReportFrequency"},
	{Name: "ReportGenerator"},
	{Name: "ResourceInventory"},
	{Name: "RuleStatement"},
	{Name: "S3Location"},
	{Name: "ScriptRuleStatement"},
	{Name: "ServiceStatus"},
	{Name: "Tag"},
	{Name: "TokenData"},
	{Name: "UpdateServiceSettings"},
}

var licenseManagerDataTypeByName = func() map[string]licenseManagerDataType {
	out := make(map[string]licenseManagerDataType, len(licenseManagerDataTypes))
	for _, dt := range licenseManagerDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
