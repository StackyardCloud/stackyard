package server

type workspacesAppStream2Type struct {
	Name string
}

// Amazon WorkSpaces Applications data types sourced from:
// https://docs.aws.amazon.com/appstream2/latest/APIReference/API_Types.html
var workspacesAppStream2Types = []workspacesAppStream2Type{
	{Name: "AccessEndpoint"},
	{Name: "AdminAppLicenseUsageRecord"},
	{Name: "AppBlock"},
	{Name: "AppBlockBuilder"},
	{Name: "AppBlockBuilderAppBlockAssociation"},
	{Name: "AppBlockBuilderStateChangeReason"},
	{Name: "Application"},
	{Name: "ApplicationConfig"},
	{Name: "ApplicationFleetAssociation"},
	{Name: "ApplicationSettings"},
	{Name: "ApplicationSettingsResponse"},
	{Name: "CertificateBasedAuthProperties"},
	{Name: "ComputeCapacity"},
	{Name: "ComputeCapacityStatus"},
	{Name: "DirectoryConfig"},
	{Name: "DomainJoinInfo"},
	{Name: "EntitledApplication"},
	{Name: "Entitlement"},
	{Name: "EntitlementAttribute"},
	{Name: "ErrorDetails"},
	{Name: "ExportImageTask"},
	{Name: "Filter"},
	{Name: "Fleet"},
	{Name: "FleetError"},
	{Name: "Image"},
	{Name: "ImageBuilder"},
	{Name: "ImageBuilderStateChangeReason"},
	{Name: "ImagePermissions"},
	{Name: "ImageStateChangeReason"},
	{Name: "LastReportGenerationExecutionError"},
	{Name: "NetworkAccessConfiguration"},
	{Name: "ResourceError"},
	{Name: "RuntimeValidationConfig"},
	{Name: "S3Location"},
	{Name: "ScriptDetails"},
	{Name: "ServiceAccountCredentials"},
	{Name: "Session"},
	{Name: "SharedImagePermissions"},
	{Name: "SoftwareAssociations"},
	{Name: "Stack"},
	{Name: "StackError"},
	{Name: "StorageConnector"},
	{Name: "StreamingExperienceSettings"},
	{Name: "Theme"},
	{Name: "ThemeFooterLink"},
	{Name: "UsageReportSubscription"},
	{Name: "User"},
	{Name: "UserSetting"},
	{Name: "UserStackAssociation"},
	{Name: "UserStackAssociationError"},
	{Name: "VolumeConfig"},
	{Name: "VpcConfig"},
}

var workspacesAppStream2TypeByName = func() map[string]workspacesAppStream2Type {
	out := make(map[string]workspacesAppStream2Type, len(workspacesAppStream2Types))
	for _, typ := range workspacesAppStream2Types {
		out[typ.Name] = typ
	}
	return out
}()
