package server

type invcOperation struct{ Name string }

// AWS Invoicing operations sourced from:
// https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_Operations_AWS_Invoicing.html
var invcOperations = []invcOperation{
	{Name: "BatchGetInvoiceProfile"},
	{Name: "CreateInvoiceUnit"},
	{Name: "CreateProcurementPortalPreference"},
	{Name: "DeleteInvoiceUnit"},
	{Name: "DeleteProcurementPortalPreference"},
	{Name: "GetInvoicePDF"},
	{Name: "GetInvoiceUnit"},
	{Name: "GetProcurementPortalPreference"},
	{Name: "ListInvoiceSummaries"},
	{Name: "ListInvoiceUnits"},
	{Name: "ListProcurementPortalPreferences"},
	{Name: "ListProcurementPortalSuppliers"},
	{Name: "ListProcurementPortals"},
	{Name: "ListTagsForResource"},
	{Name: "PutProcurementPortalPreference"},
	{Name: "SendProcurementPortalValidation"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateInvoiceUnit"},
	{Name: "UpdateProcurementPortalPreferenceStatus"},
	{Name: "VerifyProcurementPortalValidation"},
}

var invcOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range invcOperations {
		out[op.Name] = true
	}
	return out
}()
