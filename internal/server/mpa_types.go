package server

type mpaDataType struct {
	Name string
}

// Amazon Multi-party Approval data types sourced from:
// https://docs.aws.amazon.com/mpa/latest/APIReference/API_Types.html
var mpaDataTypes = []mpaDataType{
	{Name: "ApprovalStrategy"},
	{Name: "ApprovalStrategyResponse"},
	{Name: "ApprovalTeamRequestApprover"},
	{Name: "Filter"},
	{Name: "GetApprovalTeamResponseApprover"},
	{Name: "GetSessionResponseApproverResponse"},
	{Name: "IamIdentityCenter"},
	{Name: "IamIdentityCenterForGet"},
	{Name: "IamIdentityCenterForList"},
	{Name: "IdentitySourceForList"},
	{Name: "IdentitySourceParameters"},
	{Name: "IdentitySourceParametersForGet"},
	{Name: "IdentitySourceParametersForList"},
	{Name: "ListApprovalTeamsResponseApprovalTeam"},
	{Name: "ListResourcePoliciesResponseResourcePolicy"},
	{Name: "ListSessionsResponseSession"},
	{Name: "MofNApprovalStrategy"},
	{Name: "PendingUpdate"},
	{Name: "Policy"},
	{Name: "PolicyReference"},
	{Name: "PolicyVersion"},
	{Name: "PolicyVersionSummary"},
	{Name: "UpdateApprovalTeam"},
}

var mpaDataTypeByName = func() map[string]mpaDataType {
	out := make(map[string]mpaDataType, len(mpaDataTypes))
	for _, dt := range mpaDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
