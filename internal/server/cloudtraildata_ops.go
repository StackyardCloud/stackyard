package server

type cloudtraildataOperation struct {
	Name string
}

// AWS CloudTrail Data operations sourced from:
// https://docs.aws.amazon.com/awscloudtraildata/latest/APIReference/API_Operations.html
var cloudtraildataOperations = []cloudtraildataOperation{
	{Name: "PutAuditEvents"},
}

var cloudtraildataOperationByName = func() map[string]cloudtraildataOperation {
	out := make(map[string]cloudtraildataOperation, len(cloudtraildataOperations))
	for _, op := range cloudtraildataOperations {
		out[op.Name] = op
	}
	return out
}()
