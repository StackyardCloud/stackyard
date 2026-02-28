package server

type paymentCryptographyDataType struct {
	Name string
}

// AWS Payment Cryptography Control Plane data types sourced from:
// https://docs.aws.amazon.com/payment-cryptography/latest/APIReference/API_Types.html
var paymentCryptographyDataTypes = []paymentCryptographyDataType{
	{Name: "Alias"},
	{Name: "CreateAliasInput"},
	{Name: "CreateAliasOutput"},
	{Name: "CreateKeyInput"},
	{Name: "CreateKeyOutput"},
	{Name: "DeleteAliasInput"},
	{Name: "DeleteAliasOutput"},
	{Name: "DeleteKeyInput"},
	{Name: "DeleteKeyOutput"},
	{Name: "ExportAttributes"},
	{Name: "ExportDukptInitialKey"},
	{Name: "ExportKeyCryptogram"},
	{Name: "ExportKeyInput"},
	{Name: "ExportKeyMaterial"},
	{Name: "ExportKeyOutput"},
	{Name: "ExportTr31KeyBlock"},
	{Name: "ExportTr34KeyBlock"},
	{Name: "GetAliasInput"},
	{Name: "GetAliasOutput"},
	{Name: "GetKeyInput"},
	{Name: "GetKeyOutput"},
	{Name: "GetParametersForExportInput"},
	{Name: "GetParametersForExportOutput"},
	{Name: "GetParametersForImportInput"},
	{Name: "GetParametersForImportOutput"},
	{Name: "GetPublicKeyCertificateInput"},
	{Name: "GetPublicKeyCertificateOutput"},
	{Name: "ImportKeyCryptogram"},
	{Name: "ImportKeyInput"},
	{Name: "ImportKeyMaterial"},
	{Name: "ImportKeyOutput"},
	{Name: "ImportTr31KeyBlock"},
	{Name: "ImportTr34KeyBlock"},
	{Name: "Key"},
	{Name: "KeyAttributes"},
	{Name: "KeyBlockHeaders"},
	{Name: "KeyModesOfUse"},
	{Name: "KeySummary"},
	{Name: "ListAliasesInput"},
	{Name: "ListAliasesOutput"},
	{Name: "ListKeysInput"},
	{Name: "ListKeysOutput"},
	{Name: "ListTagsForResourceInput"},
	{Name: "ListTagsForResourceOutput"},
	{Name: "RestoreKeyInput"},
	{Name: "RestoreKeyOutput"},
	{Name: "RootCertificatePublicKey"},
	{Name: "StartKeyUsageInput"},
	{Name: "StartKeyUsageOutput"},
	{Name: "StopKeyUsageInput"},
	{Name: "StopKeyUsageOutput"},
	{Name: "Tag"},
	{Name: "TagResourceInput"},
	{Name: "TagResourceOutput"},
	{Name: "TrustedCertificatePublicKey"},
	{Name: "UntagResourceInput"},
	{Name: "UntagResourceOutput"},
	{Name: "UpdateAliasInput"},
	{Name: "UpdateAliasOutput"},
	{Name: "WrappedKey"},
}

var paymentCryptographyDataTypeByName = func() map[string]paymentCryptographyDataType {
	out := make(map[string]paymentCryptographyDataType, len(paymentCryptographyDataTypes))
	for _, dt := range paymentCryptographyDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
