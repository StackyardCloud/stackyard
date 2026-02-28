package server

type recycleBinDataType struct {
	Name string
}

// AWS Recycle Bin data types sourced from:
// https://docs.aws.amazon.com/recyclebin/latest/APIReference/API_Types.html
var recycleBinDataTypes = []recycleBinDataType{
	{Name: "ConflictException"},
	{Name: "CreateRuleRequest"},
	{Name: "CreateRuleResponse"},
	{Name: "DeleteRuleRequest"},
	{Name: "DeleteRuleResponse"},
	{Name: "GetRuleRequest"},
	{Name: "GetRuleResponse"},
	{Name: "InternalServerException"},
	{Name: "ListRulesRequest"},
	{Name: "ListRulesResponse"},
	{Name: "ListTagsForResourceRequest"},
	{Name: "ListTagsForResourceResponse"},
	{Name: "LockConfiguration"},
	{Name: "LockRuleRequest"},
	{Name: "LockRuleResponse"},
	{Name: "ResourceNotFoundException"},
	{Name: "ResourceTag"},
	{Name: "RetentionPeriod"},
	{Name: "RuleSummary"},
	{Name: "ServiceQuotaExceededException"},
	{Name: "Tag"},
	{Name: "TagResourceRequest"},
	{Name: "TagResourceResponse"},
	{Name: "UnlockDelay"},
	{Name: "UnlockRuleRequest"},
	{Name: "UnlockRuleResponse"},
	{Name: "UntagResourceRequest"},
	{Name: "UntagResourceResponse"},
	{Name: "UpdateRuleRequest"},
	{Name: "UpdateRuleResponse"},
	{Name: "ValidationException"},
}

var recycleBinDataTypeByName = func() map[string]recycleBinDataType {
	out := make(map[string]recycleBinDataType, len(recycleBinDataTypes))
	for _, dt := range recycleBinDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
