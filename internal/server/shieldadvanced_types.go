package server

type shieldAdvancedType struct {
	Name string
}

// AWS Shield Advanced data types sourced from:
// https://docs.aws.amazon.com/waf/latest/DDOSAPIReference/API_Types.html
var shieldAdvancedTypes = []shieldAdvancedType{
	{Name: "ApplicationLayerAutomaticResponseConfiguration"},
	{Name: "AttackDetail"},
	{Name: "AttackProperty"},
	{Name: "AttackStatisticsDataItem"},
	{Name: "AttackSummary"},
	{Name: "AttackVectorDescription"},
	{Name: "AttackVolume"},
	{Name: "AttackVolumeStatistics"},
	{Name: "BlockAction"},
	{Name: "Contributor"},
	{Name: "CountAction"},
	{Name: "EmergencyContact"},
	{Name: "InclusionProtectionFilters"},
	{Name: "InclusionProtectionGroupFilters"},
	{Name: "Limit"},
	{Name: "Mitigation"},
	{Name: "Protection"},
	{Name: "ProtectionGroup"},
	{Name: "ProtectionGroupArbitraryPatternLimits"},
	{Name: "ProtectionGroupLimits"},
	{Name: "ProtectionGroupPatternTypeLimits"},
	{Name: "ProtectionLimits"},
	{Name: "ResponseAction"},
	{Name: "SubResourceSummary"},
	{Name: "Subscription"},
	{Name: "SubscriptionLimits"},
	{Name: "SummarizedAttackVector"},
	{Name: "SummarizedCounter"},
	{Name: "Tag"},
	{Name: "TimeRange"},
	{Name: "ValidationExceptionField"},
}

var shieldAdvancedTypeByName = func() map[string]shieldAdvancedType {
	out := make(map[string]shieldAdvancedType, len(shieldAdvancedTypes))
	for _, dt := range shieldAdvancedTypes {
		out[dt.Name] = dt
	}
	return out
}()
