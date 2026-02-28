package server

type recoveryClusterDataType struct {
	Name string
}

// Amazon Route 53 Application Recovery Controller - Recovery Control Configuration data types sourced from:
// https://docs.aws.amazon.com/recovery-cluster/latest/APIReference/API_Types.html
// AWS currently serves this API guide under /latest/api/ resource pages.
var recoveryClusterDataTypes = []recoveryClusterDataType{
	{Name: "AccessDeniedException"},
	{Name: "AssertionRule"},
	{Name: "AssertionRuleUpdate"},
	{Name: "Cluster"},
	{Name: "ClusterEndpoint"},
	{Name: "ConflictException"},
	{Name: "ControlPanel"},
	{Name: "GatingRule"},
	{Name: "GatingRuleUpdate"},
	{Name: "InternalServerException"},
	{Name: "NetworkType"},
	{Name: "NewAssertionRule"},
	{Name: "NewGatingRule"},
	{Name: "ResourceNotFoundException"},
	{Name: "RoutingControl"},
	{Name: "Rule"},
	{Name: "RuleConfig"},
	{Name: "RuleType"},
	{Name: "ServiceQuotaExceededException"},
	{Name: "Status"},
	{Name: "ThrottlingException"},
	{Name: "ValidationException"},
}

var recoveryClusterDataTypeByName = func() map[string]recoveryClusterDataType {
	out := make(map[string]recoveryClusterDataType, len(recoveryClusterDataTypes))
	for _, dt := range recoveryClusterDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
