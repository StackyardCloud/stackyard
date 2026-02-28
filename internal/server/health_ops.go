package server

type healthOperation struct {
	Name string
}

// AWS Health operations sourced from:
// https://docs.aws.amazon.com/health/latest/APIReference/API_Operations.html
var healthOperations = []healthOperation{
	{Name: "DescribeAffectedAccountsForOrganization"},
	{Name: "DescribeAffectedEntities"},
	{Name: "DescribeAffectedEntitiesForOrganization"},
	{Name: "DescribeEntityAggregates"},
	{Name: "DescribeEntityAggregatesForOrganization"},
	{Name: "DescribeEventAggregates"},
	{Name: "DescribeEventDetails"},
	{Name: "DescribeEventDetailsForOrganization"},
	{Name: "DescribeEvents"},
	{Name: "DescribeEventsForOrganization"},
	{Name: "DescribeEventTypes"},
	{Name: "DescribeHealthServiceStatusForOrganization"},
	{Name: "DisableHealthServiceAccessForOrganization"},
	{Name: "EnableHealthServiceAccessForOrganization"},
}

var healthOperationByName = func() map[string]healthOperation {
	out := make(map[string]healthOperation, len(healthOperations))
	for _, op := range healthOperations {
		out[op.Name] = op
	}
	return out
}()
