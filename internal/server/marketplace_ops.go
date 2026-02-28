package server

type marketplaceOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Marketplace operations sourced from:
// https://docs.aws.amazon.com/marketplace/latest/APIReference/API_Operations.html
var marketplaceOperations = []marketplaceOperation{
	{Name: "BatchDescribeEntities", Method: "POST", URI: "/marketplace/batch-describe-entities"},
	{Name: "CancelChangeSet", Method: "POST", URI: "/marketplace/cancel-change-set"},
	{Name: "DeleteResourcePolicy", Method: "POST", URI: "/marketplace/delete-resource-policy"},
	{Name: "DescribeChangeSet", Method: "POST", URI: "/marketplace/describe-change-set"},
	{Name: "DescribeEntity", Method: "POST", URI: "/marketplace/describe-entity"},
	{Name: "GetResourcePolicy", Method: "POST", URI: "/marketplace/get-resource-policy"},
	{Name: "ListChangeSets", Method: "POST", URI: "/marketplace/list-change-sets"},
	{Name: "ListEntities", Method: "POST", URI: "/marketplace/list-entities"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/marketplace/list-tags-for-resource"},
	{Name: "PutResourcePolicy", Method: "POST", URI: "/marketplace/put-resource-policy"},
	{Name: "StartChangeSet", Method: "POST", URI: "/marketplace/start-change-set"},
	{Name: "TagResource", Method: "POST", URI: "/marketplace/tag-resource"},
	{Name: "UntagResource", Method: "POST", URI: "/marketplace/untag-resource"},
	{Name: "DescribeAgreement", Method: "POST", URI: "/marketplace/describe-agreement"},
	{Name: "GetAgreementTerms", Method: "POST", URI: "/marketplace/get-agreement-terms"},
	{Name: "SearchAgreements", Method: "POST", URI: "/marketplace/search-agreements"},
	{Name: "BatchMeterUsage", Method: "POST", URI: "/marketplace/batch-meter-usage"},
	{Name: "MeterUsage", Method: "POST", URI: "/marketplace/meter-usage"},
	{Name: "RegisterUsage", Method: "POST", URI: "/marketplace/register-usage"},
	{Name: "ResolveCustomer", Method: "POST", URI: "/marketplace/resolve-customer"},
	{Name: "GetEntitlements", Method: "POST", URI: "/marketplace/get-entitlements"},
	{Name: "PutDeploymentParameter", Method: "POST", URI: "/marketplace/put-deployment-parameter"},
	{Name: "GetBuyerDashboard", Method: "POST", URI: "/marketplace/get-buyer-dashboard"},
}

var marketplaceOperationByName = func() map[string]marketplaceOperation {
	out := make(map[string]marketplaceOperation, len(marketplaceOperations))
	for _, op := range marketplaceOperations {
		out[op.Name] = op
	}
	return out
}()
