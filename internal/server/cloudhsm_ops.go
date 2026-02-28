package server

type cloudhsmOperation struct {
	Name string
}

// AWS CloudHSM operations sourced from:
// https://docs.aws.amazon.com/cloudhsm/latest/APIReference/API_Operations.html
var cloudhsmOperations = []cloudhsmOperation{
	{Name: "CopyBackupToRegion"},
	{Name: "CreateCluster"},
	{Name: "CreateHsm"},
	{Name: "DeleteBackup"},
	{Name: "DeleteCluster"},
	{Name: "DeleteHsm"},
	{Name: "DeleteResourcePolicy"},
	{Name: "DescribeBackups"},
	{Name: "DescribeClusters"},
	{Name: "GetResourcePolicy"},
	{Name: "InitializeCluster"},
	{Name: "ListTags"},
	{Name: "ModifyBackupAttributes"},
	{Name: "ModifyCluster"},
	{Name: "PutResourcePolicy"},
	{Name: "RestoreBackup"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
}

var cloudhsmOperationByName = func() map[string]cloudhsmOperation {
	out := make(map[string]cloudhsmOperation, len(cloudhsmOperations))
	for _, op := range cloudhsmOperations {
		out[op.Name] = op
	}
	return out
}()
