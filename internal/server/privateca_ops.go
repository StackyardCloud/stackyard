package server

type privateCAOperation struct {
	Name string
}

// AWS Private Certificate Authority operations sourced from:
// https://docs.aws.amazon.com/privateca/latest/APIReference/API_Operations.html
var privateCAOperations = []privateCAOperation{
	{Name: "CreateCertificateAuthority"},
	{Name: "CreateCertificateAuthorityAuditReport"},
	{Name: "CreatePermission"},
	{Name: "DeleteCertificateAuthority"},
	{Name: "DeletePermission"},
	{Name: "DeletePolicy"},
	{Name: "DescribeCertificateAuthority"},
	{Name: "DescribeCertificateAuthorityAuditReport"},
	{Name: "GetCertificate"},
	{Name: "GetCertificateAuthorityCertificate"},
	{Name: "GetCertificateAuthorityCsr"},
	{Name: "GetPolicy"},
	{Name: "ImportCertificateAuthorityCertificate"},
	{Name: "IssueCertificate"},
	{Name: "ListCertificateAuthorities"},
	{Name: "ListPermissions"},
	{Name: "ListTags"},
	{Name: "PutPolicy"},
	{Name: "RestoreCertificateAuthority"},
	{Name: "RevokeCertificate"},
	{Name: "TagCertificateAuthority"},
	{Name: "UntagCertificateAuthority"},
	{Name: "UpdateCertificateAuthority"},
}

var privateCAOperationByName = func() map[string]privateCAOperation {
	out := make(map[string]privateCAOperation, len(privateCAOperations))
	for _, op := range privateCAOperations {
		out[op.Name] = op
	}
	return out
}()
