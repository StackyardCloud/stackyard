package server

type secretsManagerOperation struct {
	Name string
}

// AWS Secrets Manager operations sourced from:
// https://docs.aws.amazon.com/secretsmanager/latest/APIReference/API_Operations.html
var secretsManagerOperations = []secretsManagerOperation{
	{Name: "BatchGetSecretValue"},
	{Name: "CancelRotateSecret"},
	{Name: "CreateSecret"},
	{Name: "DeleteResourcePolicy"},
	{Name: "DeleteSecret"},
	{Name: "DescribeSecret"},
	{Name: "GetRandomPassword"},
	{Name: "GetResourcePolicy"},
	{Name: "GetSecretValue"},
	{Name: "ListSecrets"},
	{Name: "ListSecretVersionIds"},
	{Name: "PutResourcePolicy"},
	{Name: "PutSecretValue"},
	{Name: "RemoveRegionsFromReplication"},
	{Name: "ReplicateSecretToRegions"},
	{Name: "RestoreSecret"},
	{Name: "RotateSecret"},
	{Name: "StopReplicationToReplica"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateSecret"},
	{Name: "UpdateSecretVersionStage"},
	{Name: "ValidateResourcePolicy"},
}

var secretsManagerOperationByName = func() map[string]secretsManagerOperation {
	out := make(map[string]secretsManagerOperation, len(secretsManagerOperations))
	for _, op := range secretsManagerOperations {
		out[op.Name] = op
	}
	return out
}()
