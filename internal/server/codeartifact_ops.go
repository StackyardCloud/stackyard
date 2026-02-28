package server

type codeArtifactOperation struct {
	Name string
}

// AWS CodeArtifact operations sourced from:
// https://docs.aws.amazon.com/codeartifact/latest/APIReference/API_Operations.html
var codeArtifactOperations = []codeArtifactOperation{
	{Name: "AssociateExternalConnection"},
	{Name: "CopyPackageVersions"},
	{Name: "CreateDomain"},
	{Name: "CreatePackageGroup"},
	{Name: "CreateRepository"},
	{Name: "DeleteDomain"},
	{Name: "DeleteDomainPermissionsPolicy"},
	{Name: "DeletePackage"},
	{Name: "DeletePackageGroup"},
	{Name: "DeletePackageVersions"},
	{Name: "DeleteRepository"},
	{Name: "DeleteRepositoryPermissionsPolicy"},
	{Name: "DescribeDomain"},
	{Name: "DescribePackage"},
	{Name: "DescribePackageGroup"},
	{Name: "DescribePackageVersion"},
	{Name: "DescribeRepository"},
	{Name: "DisassociateExternalConnection"},
	{Name: "DisposePackageVersions"},
	{Name: "GetAssociatedPackageGroup"},
	{Name: "GetAuthorizationToken"},
	{Name: "GetDomainPermissionsPolicy"},
	{Name: "GetPackageVersionAsset"},
	{Name: "GetPackageVersionReadme"},
	{Name: "GetRepositoryEndpoint"},
	{Name: "GetRepositoryPermissionsPolicy"},
	{Name: "ListAllowedRepositoriesForGroup"},
	{Name: "ListAssociatedPackages"},
	{Name: "ListDomains"},
	{Name: "ListPackageGroups"},
	{Name: "ListPackages"},
	{Name: "ListPackageVersionAssets"},
	{Name: "ListPackageVersionDependencies"},
	{Name: "ListPackageVersions"},
	{Name: "ListRepositories"},
	{Name: "ListRepositoriesInDomain"},
	{Name: "ListSubPackageGroups"},
	{Name: "ListTagsForResource"},
	{Name: "PublishPackageVersion"},
	{Name: "PutDomainPermissionsPolicy"},
	{Name: "PutPackageOriginConfiguration"},
	{Name: "PutRepositoryPermissionsPolicy"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdatePackageGroup"},
	{Name: "UpdatePackageGroupOriginConfiguration"},
	{Name: "UpdatePackageVersionsStatus"},
	{Name: "UpdateRepository"},
}

var codeArtifactOperationByName = func() map[string]codeArtifactOperation {
	out := make(map[string]codeArtifactOperation, len(codeArtifactOperations))
	for _, op := range codeArtifactOperations {
		out[op.Name] = op
	}
	return out
}()
