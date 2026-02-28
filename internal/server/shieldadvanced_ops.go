package server

type shieldAdvancedOperation struct {
	Name string
}

// AWS Shield Advanced operations sourced from:
// https://docs.aws.amazon.com/waf/latest/DDOSAPIReference/API_Operations.html
var shieldAdvancedOperations = []shieldAdvancedOperation{
	{Name: "AssociateDRTLogBucket"},
	{Name: "AssociateDRTRole"},
	{Name: "AssociateHealthCheck"},
	{Name: "AssociateProactiveEngagementDetails"},
	{Name: "CreateProtection"},
	{Name: "CreateProtectionGroup"},
	{Name: "CreateSubscription"},
	{Name: "DeleteProtection"},
	{Name: "DeleteProtectionGroup"},
	{Name: "DeleteSubscription"},
	{Name: "DescribeAttack"},
	{Name: "DescribeAttackStatistics"},
	{Name: "DescribeDRTAccess"},
	{Name: "DescribeEmergencyContactSettings"},
	{Name: "DescribeProtection"},
	{Name: "DescribeProtectionGroup"},
	{Name: "DescribeSubscription"},
	{Name: "DisableApplicationLayerAutomaticResponse"},
	{Name: "DisableProactiveEngagement"},
	{Name: "DisassociateDRTLogBucket"},
	{Name: "DisassociateDRTRole"},
	{Name: "DisassociateHealthCheck"},
	{Name: "EnableApplicationLayerAutomaticResponse"},
	{Name: "EnableProactiveEngagement"},
	{Name: "GetSubscriptionState"},
	{Name: "ListAttacks"},
	{Name: "ListProtectionGroups"},
	{Name: "ListProtections"},
	{Name: "ListResourcesInProtectionGroup"},
	{Name: "ListTagsForResource"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateApplicationLayerAutomaticResponse"},
	{Name: "UpdateEmergencyContactSettings"},
	{Name: "UpdateProtectionGroup"},
	{Name: "UpdateSubscription"},
}

var shieldAdvancedOperationByName = func() map[string]shieldAdvancedOperation {
	out := make(map[string]shieldAdvancedOperation, len(shieldAdvancedOperations))
	for _, op := range shieldAdvancedOperations {
		out[op.Name] = op
	}
	return out
}()
