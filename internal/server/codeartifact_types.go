package server

type codeArtifactDataType struct {
	Name string
}

// AWS CodeArtifact data types sourced from:
// https://docs.aws.amazon.com/codeartifact/latest/APIReference/API_Types.html
var codeArtifactDataTypes = []codeArtifactDataType{
	{Name: "AssetSummary"},
	{Name: "AssociatedPackage"},
	{Name: "DomainDescription"},
	{Name: "DomainEntryPoint"},
	{Name: "DomainSummary"},
	{Name: "LicenseInfo"},
	{Name: "PackageDependency"},
	{Name: "PackageDescription"},
	{Name: "PackageGroupAllowedRepository"},
	{Name: "PackageGroupDescription"},
	{Name: "PackageGroupOriginConfiguration"},
	{Name: "PackageGroupOriginRestriction"},
	{Name: "PackageGroupReference"},
	{Name: "PackageGroupSummary"},
	{Name: "PackageOriginConfiguration"},
	{Name: "PackageOriginRestrictions"},
	{Name: "PackageSummary"},
	{Name: "PackageVersionDescription"},
	{Name: "PackageVersionError"},
	{Name: "PackageVersionOrigin"},
	{Name: "PackageVersionSummary"},
	{Name: "RepositoryDescription"},
	{Name: "RepositoryExternalConnectionInfo"},
	{Name: "RepositorySummary"},
	{Name: "ResourcePolicy"},
	{Name: "SuccessfulPackageVersionInfo"},
	{Name: "Tag"},
	{Name: "UpstreamRepository"},
	{Name: "UpstreamRepositoryInfo"},
}

var codeArtifactDataTypeByName = func() map[string]codeArtifactDataType {
	out := make(map[string]codeArtifactDataType, len(codeArtifactDataTypes))
	for _, dt := range codeArtifactDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
