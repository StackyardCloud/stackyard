package server

type taxsOperation struct{ Name string }

// Tax Settings operations sourced from:
// https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_Operations_Tax_Settings.html
var taxsOperations = []taxsOperation{
	{Name: "BatchDeleteTaxRegistration"},
	{Name: "BatchGetTaxExemptions"},
	{Name: "BatchPutTaxRegistration"},
	{Name: "DeleteSupplementalTaxRegistration"},
	{Name: "DeleteTaxRegistration"},
	{Name: "GetTaxExemptionTypes"},
	{Name: "GetTaxInheritance"},
	{Name: "GetTaxRegistration"},
	{Name: "GetTaxRegistrationDocument"},
	{Name: "ListSupplementalTaxRegistrations"},
	{Name: "ListTaxExemptions"},
	{Name: "ListTaxRegistrations"},
	{Name: "PutSupplementalTaxRegistration"},
	{Name: "PutTaxExemption"},
	{Name: "PutTaxInheritance"},
	{Name: "PutTaxRegistration"},
}
