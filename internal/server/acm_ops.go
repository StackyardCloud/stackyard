package server

type acmOperation struct {
	Name string
}

// AWS Certificate Manager (ACM) operations sourced from:
// https://docs.aws.amazon.com/acm/latest/APIReference/API_Operations.html
var acmOperations = []acmOperation{
	{Name: "AddTagsToCertificate"},
	{Name: "DeleteCertificate"},
	{Name: "DescribeCertificate"},
	{Name: "ExportCertificate"},
	{Name: "GetAccountConfiguration"},
	{Name: "GetCertificate"},
	{Name: "ImportCertificate"},
	{Name: "ListCertificates"},
	{Name: "ListTagsForCertificate"},
	{Name: "PutAccountConfiguration"},
	{Name: "RemoveTagsFromCertificate"},
	{Name: "RevokeCertificate"},
	{Name: "RenewCertificate"},
	{Name: "RequestCertificate"},
	{Name: "ResendValidationEmail"},
	{Name: "UpdateCertificateOptions"},
}

var acmOperationByName = func() map[string]acmOperation {
	out := make(map[string]acmOperation, len(acmOperations))
	for _, op := range acmOperations {
		out[op.Name] = op
	}
	return out
}()
