package server

type savingsPlansType struct {
	Name string
}

// AWS Savings Plans data types sourced from:
// https://docs.aws.amazon.com/savingsplans/latest/APIReference/API_Types.html
var savingsPlansTypes = []savingsPlansType{
	{Name: "ParentSavingsPlanOffering"},
	{Name: "SavingsPlan"},
	{Name: "SavingsPlanFilter"},
	{Name: "SavingsPlanOffering"},
	{Name: "SavingsPlanOfferingFilterElement"},
	{Name: "SavingsPlanOfferingProperty"},
	{Name: "SavingsPlanOfferingRate"},
	{Name: "SavingsPlanOfferingRateFilterElement"},
	{Name: "SavingsPlanOfferingRateProperty"},
	{Name: "SavingsPlanRate"},
	{Name: "SavingsPlanRateFilter"},
	{Name: "SavingsPlanRateProperty"},
}

var savingsPlansTypeByName = func() map[string]savingsPlansType {
	out := make(map[string]savingsPlansType, len(savingsPlansTypes))
	for _, dt := range savingsPlansTypes {
		out[dt.Name] = dt
	}
	return out
}()
