package server

type eksOperation struct {
	Name string
}

// Amazon EKS operations sourced from:
// https://docs.aws.amazon.com/eks/latest/APIReference/API_Operations.html
var eksOperations = []eksOperation{
	{Name: "AssociateAccessPolicy"},
	{Name: "AssociateEncryptionConfig"},
	{Name: "AssociateIdentityProviderConfig"},
	{Name: "CreateAccessEntry"},
	{Name: "CreateAddon"},
	{Name: "CreateCapability"},
	{Name: "CreateCluster"},
	{Name: "CreateEksAnywhereSubscription"},
	{Name: "CreateFargateProfile"},
	{Name: "CreateNodegroup"},
	{Name: "CreatePodIdentityAssociation"},
	{Name: "DeleteAccessEntry"},
	{Name: "DeleteAddon"},
	{Name: "DeleteCapability"},
	{Name: "DeleteCluster"},
	{Name: "DeleteEksAnywhereSubscription"},
	{Name: "DeleteFargateProfile"},
	{Name: "DeleteNodegroup"},
	{Name: "DeletePodIdentityAssociation"},
	{Name: "DeregisterCluster"},
	{Name: "DescribeAccessEntry"},
	{Name: "DescribeAddon"},
	{Name: "DescribeAddonConfiguration"},
	{Name: "DescribeAddonVersions"},
	{Name: "DescribeCapability"},
	{Name: "DescribeCluster"},
	{Name: "DescribeClusterVersions"},
	{Name: "DescribeEksAnywhereSubscription"},
	{Name: "DescribeFargateProfile"},
	{Name: "DescribeIdentityProviderConfig"},
	{Name: "DescribeInsight"},
	{Name: "DescribeInsightsRefresh"},
	{Name: "DescribeNodegroup"},
	{Name: "DescribePodIdentityAssociation"},
	{Name: "DescribeUpdate"},
	{Name: "DisassociateAccessPolicy"},
	{Name: "DisassociateIdentityProviderConfig"},
	{Name: "ListAccessEntries"},
	{Name: "ListAccessPolicies"},
	{Name: "ListAddons"},
	{Name: "ListAssociatedAccessPolicies"},
	{Name: "ListCapabilities"},
	{Name: "ListClusters"},
	{Name: "ListEksAnywhereSubscriptions"},
	{Name: "ListFargateProfiles"},
	{Name: "ListIdentityProviderConfigs"},
	{Name: "ListInsights"},
	{Name: "ListNodegroups"},
	{Name: "ListPodIdentityAssociations"},
	{Name: "ListTagsForResource"},
	{Name: "ListUpdates"},
	{Name: "RegisterCluster"},
	{Name: "StartInsightsRefresh"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateAccessEntry"},
	{Name: "UpdateAddon"},
	{Name: "UpdateCapability"},
	{Name: "UpdateClusterConfig"},
	{Name: "UpdateClusterVersion"},
	{Name: "UpdateEksAnywhereSubscription"},
	{Name: "UpdateNodegroupConfig"},
	{Name: "UpdateNodegroupVersion"},
	{Name: "UpdatePodIdentityAssociation"},
}

var eksOperationByName = func() map[string]eksOperation {
	out := make(map[string]eksOperation, len(eksOperations))
	for _, op := range eksOperations {
		out[op.Name] = op
	}
	return out
}()
