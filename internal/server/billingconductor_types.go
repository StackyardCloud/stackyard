package server

type billingConductorDataType struct {
	Name string
}

// AWS Billing Conductor data types sourced from:
// https://docs.aws.amazon.com/billingconductor/latest/APIReference/API_Types.html
var billingConductorDataTypes = []billingConductorDataType{
	{Name: "AccountAssociationsListElement"},
	{Name: "AccountGrouping"},
	{Name: "AssociateResourceError"},
	{Name: "AssociateResourceResponseElement"},
	{Name: "Attribute"},
	{Name: "BillingGroupCostReportElement"},
	{Name: "BillingGroupCostReportResultElement"},
	{Name: "BillingGroupListElement"},
	{Name: "BillingPeriodRange"},
	{Name: "ComputationPreference"},
	{Name: "CreateFreeTierConfig"},
	{Name: "CreateTieringInput"},
	{Name: "CustomLineItemBillingPeriodRange"},
	{Name: "CustomLineItemChargeDetails"},
	{Name: "CustomLineItemFlatChargeDetails"},
	{Name: "CustomLineItemListElement"},
	{Name: "CustomLineItemPercentageChargeDetails"},
	{Name: "CustomLineItemVersionListElement"},
	{Name: "DisassociateResourceResponseElement"},
	{Name: "FreeTierConfig"},
	{Name: "LineItemFilter"},
	{Name: "ListAccountAssociationsFilter"},
	{Name: "ListBillingGroupAccountGrouping"},
	{Name: "ListBillingGroupCostReportsFilter"},
	{Name: "ListBillingGroupsFilter"},
	{Name: "ListCustomLineItemChargeDetails"},
	{Name: "ListCustomLineItemFlatChargeDetails"},
	{Name: "ListCustomLineItemPercentageChargeDetails"},
	{Name: "ListCustomLineItemVersionsBillingPeriodRangeFilter"},
	{Name: "ListCustomLineItemVersionsFilter"},
	{Name: "ListCustomLineItemsFilter"},
	{Name: "ListPricingPlansFilter"},
	{Name: "ListPricingRulesFilter"},
	{Name: "ListResourcesAssociatedToCustomLineItemFilter"},
	{Name: "ListResourcesAssociatedToCustomLineItemResponseElement"},
	{Name: "PresentationObject"},
	{Name: "PricingPlanListElement"},
	{Name: "PricingRuleListElement"},
	{Name: "StringSearch"},
	{Name: "Tiering"},
	{Name: "UpdateBillingGroupAccountGrouping"},
	{Name: "UpdateCustomLineItemChargeDetails"},
	{Name: "UpdateCustomLineItemFlatChargeDetails"},
	{Name: "UpdateCustomLineItemPercentageChargeDetails"},
	{Name: "UpdateFreeTierConfig"},
	{Name: "UpdatePricingRule"},
	{Name: "UpdateTieringInput"},
	{Name: "ValidationExceptionField"},
}

var billingConductorDataTypeByName = func() map[string]billingConductorDataType {
	out := make(map[string]billingConductorDataType, len(billingConductorDataTypes))
	for _, dataType := range billingConductorDataTypes {
		out[dataType.Name] = dataType
	}
	return out
}()
