package server

type eksauthOperation struct{ Name string }

// Amazon EKS Auth operations sourced from:
// https://docs.aws.amazon.com/eks/latest/APIReference/API_Operations_Amazon_EKS_Auth.html
var eksauthOperations = []eksauthOperation{
	{Name: "AssumeRoleForPodIdentity"},
}

var eksauthOperationByName = func() map[string]eksauthOperation {
	out := make(map[string]eksauthOperation, len(eksauthOperations))
	for _, op := range eksauthOperations {
		out[op.Name] = op
	}
	return out
}()
