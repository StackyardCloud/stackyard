package server

type ecrOperation struct {
	Name string
}

var ecrOperations = []ecrOperation{
	{Name: "BatchCheckLayerAvailability"},
	{Name: "BatchDeleteImage"},
	{Name: "BatchGetImage"},
	{Name: "BatchGetRepositoryScanningConfiguration"},
	{Name: "CompleteLayerUpload"},
	{Name: "CreatePullThroughCacheRule"},
	{Name: "CreateRepository"},
	{Name: "CreateRepositoryCreationTemplate"},
	{Name: "DeleteLifecyclePolicy"},
	{Name: "DeletePullThroughCacheRule"},
	{Name: "DeleteRegistryPolicy"},
	{Name: "DeleteRepository"},
	{Name: "DeleteRepositoryCreationTemplate"},
	{Name: "DeleteRepositoryPolicy"},
	{Name: "DeleteSigningConfiguration"},
	{Name: "DeregisterPullTimeUpdateExclusion"},
	{Name: "DescribeImageReplicationStatus"},
	{Name: "DescribeImageScanFindings"},
	{Name: "DescribeImages"},
	{Name: "DescribeImageSigningStatus"},
	{Name: "DescribePullThroughCacheRules"},
	{Name: "DescribeRegistry"},
	{Name: "DescribeRepositories"},
	{Name: "DescribeRepositoryCreationTemplates"},
	{Name: "GetAccountSetting"},
	{Name: "GetAuthorizationToken"},
	{Name: "GetDownloadUrlForLayer"},
	{Name: "GetLifecyclePolicy"},
	{Name: "GetLifecyclePolicyPreview"},
	{Name: "GetRegistryPolicy"},
	{Name: "GetRegistryScanningConfiguration"},
	{Name: "GetRepositoryPolicy"},
	{Name: "GetSigningConfiguration"},
	{Name: "InitiateLayerUpload"},
	{Name: "ListImageReferrers"},
	{Name: "ListImages"},
	{Name: "ListPullTimeUpdateExclusions"},
	{Name: "ListTagsForResource"},
	{Name: "PutAccountSetting"},
	{Name: "PutImage"},
	{Name: "PutImageScanningConfiguration"},
	{Name: "PutImageTagMutability"},
	{Name: "PutLifecyclePolicy"},
	{Name: "PutRegistryPolicy"},
	{Name: "PutRegistryScanningConfiguration"},
	{Name: "PutReplicationConfiguration"},
	{Name: "PutSigningConfiguration"},
	{Name: "RegisterPullTimeUpdateExclusion"},
	{Name: "SetRepositoryPolicy"},
	{Name: "StartImageScan"},
	{Name: "StartLifecyclePolicyPreview"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateImageStorageClass"},
	{Name: "UpdatePullThroughCacheRule"},
	{Name: "UpdateRepositoryCreationTemplate"},
	{Name: "UploadLayerPart"},
	{Name: "ValidatePullThroughCacheRule"},
}

var ecrOperationByName = func() map[string]ecrOperation {
	out := make(map[string]ecrOperation, len(ecrOperations))
	for _, op := range ecrOperations {
		out[op.Name] = op
	}
	return out
}()
