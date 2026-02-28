package server

type billingConductorOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Billing Conductor actions sourced from:
// https://docs.aws.amazon.com/billingconductor/latest/APIReference/API_Operations.html
var billingConductorOperations = []billingConductorOperation{
	{Name: "AssociateAccounts", Method: "POST", URI: "/associate-accounts"},
	{Name: "AssociatePricingRules", Method: "PUT", URI: "/associate-pricing-rules"},
	{Name: "BatchAssociateResourcesToCustomLineItem", Method: "PUT", URI: "/batch-associate-resources-to-custom-line-item"},
	{Name: "BatchDisassociateResourcesFromCustomLineItem", Method: "PUT", URI: "/batch-disassociate-resources-from-custom-line-item"},
	{Name: "CreateBillingGroup", Method: "POST", URI: "/create-billing-group"},
	{Name: "CreateCustomLineItem", Method: "POST", URI: "/create-custom-line-item"},
	{Name: "CreatePricingPlan", Method: "POST", URI: "/create-pricing-plan"},
	{Name: "CreatePricingRule", Method: "POST", URI: "/create-pricing-rule"},
	{Name: "DeleteBillingGroup", Method: "POST", URI: "/delete-billing-group"},
	{Name: "DeleteCustomLineItem", Method: "POST", URI: "/delete-custom-line-item"},
	{Name: "DeletePricingPlan", Method: "POST", URI: "/delete-pricing-plan"},
	{Name: "DeletePricingRule", Method: "POST", URI: "/delete-pricing-rule"},
	{Name: "DisassociateAccounts", Method: "POST", URI: "/disassociate-accounts"},
	{Name: "DisassociatePricingRules", Method: "PUT", URI: "/disassociate-pricing-rules"},
	{Name: "GetBillingGroupCostReport", Method: "POST", URI: "/get-billing-group-cost-report"},
	{Name: "ListAccountAssociations", Method: "POST", URI: "/list-account-associations"},
	{Name: "ListBillingGroupCostReports", Method: "POST", URI: "/list-billing-group-cost-reports"},
	{Name: "ListBillingGroups", Method: "POST", URI: "/list-billing-groups"},
	{Name: "ListCustomLineItemVersions", Method: "POST", URI: "/list-custom-line-item-versions"},
	{Name: "ListCustomLineItems", Method: "POST", URI: "/list-custom-line-items"},
	{Name: "ListPricingPlans", Method: "POST", URI: "/list-pricing-plans"},
	{Name: "ListPricingPlansAssociatedWithPricingRule", Method: "POST", URI: "/list-pricing-plans-associated-with-pricing-rule"},
	{Name: "ListPricingRules", Method: "POST", URI: "/list-pricing-rules"},
	{Name: "ListPricingRulesAssociatedToPricingPlan", Method: "POST", URI: "/list-pricing-rules-associated-to-pricing-plan"},
	{Name: "ListResourcesAssociatedToCustomLineItem", Method: "POST", URI: "/list-resources-associated-to-custom-line-item"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateBillingGroup", Method: "POST", URI: "/update-billing-group"},
	{Name: "UpdateCustomLineItem", Method: "POST", URI: "/update-custom-line-item"},
	{Name: "UpdatePricingPlan", Method: "PUT", URI: "/update-pricing-plan"},
	{Name: "UpdatePricingRule", Method: "PUT", URI: "/update-pricing-rule"},
}

var billingConductorOperationByName = func() map[string]billingConductorOperation {
	out := make(map[string]billingConductorOperation, len(billingConductorOperations))
	for _, op := range billingConductorOperations {
		out[op.Name] = op
	}
	return out
}()
