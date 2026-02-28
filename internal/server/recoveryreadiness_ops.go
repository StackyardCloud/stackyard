package server

type recoveryReadinessOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Route 53 Application Recovery Controller - Recovery Readiness operations sourced from:
// https://docs.aws.amazon.com/recovery-readiness/latest/APIReference/API_Operations.html
// AWS currently serves this API reference under /latest/api/.
var recoveryReadinessOperations = []recoveryReadinessOperation{
	{Name: "CreateCell", Method: "POST", URI: "/cells"},
	{Name: "CreateCrossAccountAuthorization", Method: "POST", URI: "/crossaccountauthorizations"},
	{Name: "CreateReadinessCheck", Method: "POST", URI: "/readinesschecks"},
	{Name: "CreateRecoveryGroup", Method: "POST", URI: "/recoverygroups"},
	{Name: "CreateResourceSet", Method: "POST", URI: "/resourcesets"},
	{Name: "DeleteCell", Method: "DELETE", URI: "/cells/{CellName}"},
	{Name: "DeleteCrossAccountAuthorization", Method: "DELETE", URI: "/crossaccountauthorizations/{CrossAccountAuthorization}"},
	{Name: "DeleteReadinessCheck", Method: "DELETE", URI: "/readinesschecks/{ReadinessCheckName}"},
	{Name: "DeleteRecoveryGroup", Method: "DELETE", URI: "/recoverygroups/{RecoveryGroupName}"},
	{Name: "DeleteResourceSet", Method: "DELETE", URI: "/resourcesets/{ResourceSetName}"},
	{Name: "GetArchitectureRecommendations", Method: "GET", URI: "/recoverygroups/{RecoveryGroupName}/architectureRecommendations"},
	{Name: "GetCell", Method: "GET", URI: "/cells/{CellName}"},
	{Name: "GetCellReadinessSummary", Method: "GET", URI: "/cellreadiness/{CellName}"},
	{Name: "GetReadinessCheck", Method: "GET", URI: "/readinesschecks/{ReadinessCheckName}"},
	{Name: "GetReadinessCheckResourceStatus", Method: "GET", URI: "/readinesschecks/{ReadinessCheckName}/resource/{ResourceIdentifier}/status"},
	{Name: "GetReadinessCheckStatus", Method: "GET", URI: "/readinesschecks/{ReadinessCheckName}/status"},
	{Name: "GetRecoveryGroup", Method: "GET", URI: "/recoverygroups/{RecoveryGroupName}"},
	{Name: "GetRecoveryGroupReadinessSummary", Method: "GET", URI: "/recoverygroupreadiness/{RecoveryGroupName}"},
	{Name: "GetResourceSet", Method: "GET", URI: "/resourcesets/{ResourceSetName}"},
	{Name: "ListCells", Method: "GET", URI: "/cells"},
	{Name: "ListCrossAccountAuthorizations", Method: "GET", URI: "/crossaccountauthorizations"},
	{Name: "ListReadinessChecks", Method: "GET", URI: "/readinesschecks"},
	{Name: "ListRecoveryGroups", Method: "GET", URI: "/recoverygroups"},
	{Name: "ListResourceSets", Method: "GET", URI: "/resourcesets"},
	{Name: "ListRules", Method: "GET", URI: "/rules"},
	{Name: "ListTagsForResources", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateCell", Method: "PUT", URI: "/cells/{CellName}"},
	{Name: "UpdateReadinessCheck", Method: "PUT", URI: "/readinesschecks/{ReadinessCheckName}"},
	{Name: "UpdateRecoveryGroup", Method: "PUT", URI: "/recoverygroups/{RecoveryGroupName}"},
	{Name: "UpdateResourceSet", Method: "PUT", URI: "/resourcesets/{ResourceSetName}"},
}

var recoveryReadinessOperationByName = func() map[string]recoveryReadinessOperation {
	out := make(map[string]recoveryReadinessOperation, len(recoveryReadinessOperations))
	for _, op := range recoveryReadinessOperations {
		out[op.Name] = op
	}
	return out
}()
