package server

type transferOperation struct {
	Name string
}

// AWS Transfer Family operations sourced from:
// https://docs.aws.amazon.com/transfer/latest/APIReference/API_Operations.html
var transferOperations = []transferOperation{
	{Name: "CreateAccess"},
	{Name: "CreateAgreement"},
	{Name: "CreateConnector"},
	{Name: "CreateProfile"},
	{Name: "CreateServer"},
	{Name: "CreateUser"},
	{Name: "CreateWebApp"},
	{Name: "CreateWorkflow"},
	{Name: "DeleteAccess"},
	{Name: "DeleteAgreement"},
	{Name: "DeleteCertificate"},
	{Name: "DeleteConnector"},
	{Name: "DeleteHostKey"},
	{Name: "DeleteProfile"},
	{Name: "DeleteServer"},
	{Name: "DeleteSshPublicKey"},
	{Name: "DeleteUser"},
	{Name: "DeleteWebApp"},
	{Name: "DeleteWebAppCustomization"},
	{Name: "DeleteWorkflow"},
	{Name: "DescribeAccess"},
	{Name: "DescribeAgreement"},
	{Name: "DescribeCertificate"},
	{Name: "DescribeConnector"},
	{Name: "DescribeExecution"},
	{Name: "DescribeHostKey"},
	{Name: "DescribeProfile"},
	{Name: "DescribeSecurityPolicy"},
	{Name: "DescribeServer"},
	{Name: "DescribeUser"},
	{Name: "DescribeWebApp"},
	{Name: "DescribeWebAppCustomization"},
	{Name: "DescribeWorkflow"},
	{Name: "ImportCertificate"},
	{Name: "ImportHostKey"},
	{Name: "ImportSshPublicKey"},
	{Name: "ListAccesses"},
	{Name: "ListAgreements"},
	{Name: "ListCertificates"},
	{Name: "ListConnectors"},
	{Name: "ListExecutions"},
	{Name: "ListFileTransferResults"},
	{Name: "ListHostKeys"},
	{Name: "ListProfiles"},
	{Name: "ListSecurityPolicies"},
	{Name: "ListServers"},
	{Name: "ListTagsForResource"},
	{Name: "ListUsers"},
	{Name: "ListWebApps"},
	{Name: "ListWorkflows"},
	{Name: "SendWorkflowStepState"},
	{Name: "StartDirectoryListing"},
	{Name: "StartFileTransfer"},
	{Name: "StartRemoteDelete"},
	{Name: "StartRemoteMove"},
	{Name: "StartServer"},
	{Name: "StopServer"},
	{Name: "TagResource"},
	{Name: "TestConnection"},
	{Name: "TestIdentityProvider"},
	{Name: "UntagResource"},
	{Name: "UpdateAccess"},
	{Name: "UpdateAgreement"},
	{Name: "UpdateCertificate"},
	{Name: "UpdateConnector"},
	{Name: "UpdateHostKey"},
	{Name: "UpdateProfile"},
	{Name: "UpdateServer"},
	{Name: "UpdateUser"},
	{Name: "UpdateWebApp"},
	{Name: "UpdateWebAppCustomization"},
}

var transferOperationByName = func() map[string]transferOperation {
	out := make(map[string]transferOperation, len(transferOperations))
	for _, op := range transferOperations {
		out[op.Name] = op
	}
	return out
}()
