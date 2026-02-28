package server

type signerOperation struct {
	Name string
}

// AWS Signer operations sourced from:
// https://docs.aws.amazon.com/signer/latest/api/API_Operations.html
var signerOperations = []signerOperation{
	{Name: "AddProfilePermission"},
	{Name: "CancelSigningProfile"},
	{Name: "DescribeSigningJob"},
	{Name: "GetRevocationStatus"},
	{Name: "GetSigningPlatform"},
	{Name: "GetSigningProfile"},
	{Name: "ListProfilePermissions"},
	{Name: "ListSigningJobs"},
	{Name: "ListSigningPlatforms"},
	{Name: "ListSigningProfiles"},
	{Name: "ListTagsForResource"},
	{Name: "PutSigningProfile"},
	{Name: "RemoveProfilePermission"},
	{Name: "RevokeSignature"},
	{Name: "RevokeSigningProfile"},
	{Name: "SignPayload"},
	{Name: "StartSigningJob"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
}

var signerOperationByName = func() map[string]signerOperation {
	out := make(map[string]signerOperation, len(signerOperations))
	for _, op := range signerOperations {
		out[op.Name] = op
	}
	return out
}()
