package server

type ecrPublicOperation struct {
	Name string
}

// Amazon ECR Public operations sourced from:
// https://docs.aws.amazon.com/AmazonECRPublic/latest/APIReference/API_Operations.html
var ecrPublicOperations = []ecrPublicOperation{
	{Name: "BatchCheckLayerAvailability"},
	{Name: "BatchDeleteImage"},
	{Name: "CompleteLayerUpload"},
	{Name: "CreateRepository"},
	{Name: "DeleteRepository"},
	{Name: "DeleteRepositoryPolicy"},
	{Name: "DescribeImageTags"},
	{Name: "DescribeImages"},
	{Name: "DescribeRegistries"},
	{Name: "DescribeRepositories"},
	{Name: "GetAuthorizationToken"},
	{Name: "GetRegistryCatalogData"},
	{Name: "GetRepositoryCatalogData"},
	{Name: "GetRepositoryPolicy"},
	{Name: "InitiateLayerUpload"},
	{Name: "ListTagsForResource"},
	{Name: "PutImage"},
	{Name: "PutRegistryCatalogData"},
	{Name: "PutRepositoryCatalogData"},
	{Name: "SetRepositoryPolicy"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UploadLayerPart"},
}

var ecrPublicOperationByName = func() map[string]ecrPublicOperation {
	out := make(map[string]ecrPublicOperation, len(ecrPublicOperations))
	for _, op := range ecrPublicOperations {
		out[op.Name] = op
	}
	return out
}()
