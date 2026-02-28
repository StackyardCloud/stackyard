package server

type recoveryReadinessDataType struct {
	Name string
}

// Amazon Route 53 Application Recovery Controller - Recovery Readiness data types sourced from:
// https://docs.aws.amazon.com/recovery-readiness/latest/APIReference/API_Types.html
// AWS currently serves this API reference under /latest/api/.
var recoveryReadinessDataTypes = []recoveryReadinessDataType{
	{Name: "AccessDeniedException"},
	{Name: "CellOutput"},
	{Name: "ConflictException"},
	{Name: "DNSTargetResource"},
	{Name: "InternalServerException"},
	{Name: "ListRulesOutput"},
	{Name: "Message"},
	{Name: "NLBResource"},
	{Name: "R53ResourceRecord"},
	{Name: "Readiness"},
	{Name: "ReadinessCheckOutput"},
	{Name: "ReadinessCheckSummary"},
	{Name: "Recommendation"},
	{Name: "RecoveryGroupOutput"},
	{Name: "Resource"},
	{Name: "ResourceNotFoundException"},
	{Name: "ResourceResult"},
	{Name: "ResourceSetOutput"},
	{Name: "RuleResult"},
	{Name: "TargetResource"},
	{Name: "ThrottlingException"},
	{Name: "ValidationException"},
}

var recoveryReadinessDataTypeByName = func() map[string]recoveryReadinessDataType {
	out := make(map[string]recoveryReadinessDataType, len(recoveryReadinessDataTypes))
	for _, dt := range recoveryReadinessDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
