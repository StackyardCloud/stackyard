package server

type paymentCryptographyDataOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Payment Cryptography Data Plane operations sourced from:
// https://docs.aws.amazon.com/payment-cryptography/latest/DataAPIReference/API_Operations.html
var paymentCryptographyDataOperations = []paymentCryptographyDataOperation{
	{Name: "DecryptData", Method: "POST", URI: "/keys/{KeyIdentifier}/decrypt"},
	{Name: "EncryptData", Method: "POST", URI: "/keys/{KeyIdentifier}/encrypt"},
	{Name: "GenerateAs2805KekValidation", Method: "POST", URI: "/as2805kekvalidation/generate"},
	{Name: "GenerateCardValidationData", Method: "POST", URI: "/cardvalidationdata/generate"},
	{Name: "GenerateMac", Method: "POST", URI: "/mac/generate"},
	{Name: "GenerateMacEmvPinChange", Method: "POST", URI: "/macemvpinchange/generate"},
	{Name: "GeneratePinData", Method: "POST", URI: "/pindata/generate"},
	{Name: "ReEncryptData", Method: "POST", URI: "/keys/{IncomingKeyIdentifier}/reencrypt"},
	{Name: "TranslateKeyMaterial", Method: "POST", URI: "/keymaterial/translate"},
	{Name: "TranslatePinData", Method: "POST", URI: "/pindata/translate"},
	{Name: "VerifyAuthRequestCryptogram", Method: "POST", URI: "/cryptogram/verify"},
	{Name: "VerifyCardValidationData", Method: "POST", URI: "/cardvalidationdata/verify"},
	{Name: "VerifyMac", Method: "POST", URI: "/mac/verify"},
	{Name: "VerifyPinData", Method: "POST", URI: "/pindata/verify"},
}

var paymentCryptographyDataOperationByName = func() map[string]paymentCryptographyDataOperation {
	out := make(map[string]paymentCryptographyDataOperation, len(paymentCryptographyDataOperations))
	for _, op := range paymentCryptographyDataOperations {
		out[op.Name] = op
	}
	return out
}()
