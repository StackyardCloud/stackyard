package server

type paymentCryptographyDataPlaneType struct {
	Name string
}

// AWS Payment Cryptography Data Plane data types sourced from:
// https://docs.aws.amazon.com/payment-cryptography/latest/DataAPIReference/API_Types.html
var paymentCryptographyDataPlaneTypes = []paymentCryptographyDataPlaneType{
	{Name: "AmexAttributes"},
	{Name: "AmexCardSecurityCodeVersion1"},
	{Name: "AmexCardSecurityCodeVersion2"},
	{Name: "As2805KekValidationType"},
	{Name: "As2805PekDerivationAttributes"},
	{Name: "AsymmetricEncryptionAttributes"},
	{Name: "CardGenerationAttributes"},
	{Name: "CardHolderVerificationValue"},
	{Name: "CardVerificationAttributes"},
	{Name: "CardVerificationValue1"},
	{Name: "CardVerificationValue2"},
	{Name: "CryptogramAuthResponse"},
	{Name: "CryptogramVerificationArpcMethod1"},
	{Name: "CryptogramVerificationArpcMethod2"},
	{Name: "CurrentPinAttributes"},
	{Name: "DerivationMethodAttributes"},
	{Name: "DiffieHellmanDerivationData"},
	{Name: "DiscoverDynamicCardVerificationCode"},
	{Name: "DukptAttributes"},
	{Name: "DukptDerivationAttributes"},
	{Name: "DukptEncryptionAttributes"},
	{Name: "DynamicCardVerificationCode"},
	{Name: "DynamicCardVerificationValue"},
	{Name: "EcdhDerivationAttributes"},
	{Name: "Emv2000Attributes"},
	{Name: "EmvCommonAttributes"},
	{Name: "EmvEncryptionAttributes"},
	{Name: "EncryptionDecryptionAttributes"},
	{Name: "Ibm3624NaturalPin"},
	{Name: "Ibm3624PinFromOffset"},
	{Name: "Ibm3624PinOffset"},
	{Name: "Ibm3624PinVerification"},
	{Name: "Ibm3624RandomPin"},
	{Name: "IncomingDiffieHellmanTr31KeyBlock"},
	{Name: "IncomingKeyMaterial"},
	{Name: "KekValidationRequest"},
	{Name: "KekValidationResponse"},
	{Name: "MacAlgorithmDukpt"},
	{Name: "MacAlgorithmEmv"},
	{Name: "MacAttributes"},
	{Name: "MasterCardAttributes"},
	{Name: "OutgoingKeyMaterial"},
	{Name: "OutgoingTr31KeyBlock"},
	{Name: "PinData"},
	{Name: "PinGenerationAttributes"},
	{Name: "PinVerificationAttributes"},
	{Name: "ReEncryptionAttributes"},
	{Name: "SessionKeyAmex"},
	{Name: "SessionKeyDerivation"},
	{Name: "SessionKeyDerivationValue"},
	{Name: "SessionKeyEmv2000"},
	{Name: "SessionKeyEmvCommon"},
	{Name: "SessionKeyMastercard"},
	{Name: "SessionKeyVisa"},
	{Name: "SymmetricEncryptionAttributes"},
	{Name: "TranslationIsoFormats"},
	{Name: "TranslationPinDataAs2805Format0"},
	{Name: "TranslationPinDataIsoFormat034"},
	{Name: "TranslationPinDataIsoFormat1"},
	{Name: "ValidationExceptionField"},
	{Name: "VerifyPinData"},
	{Name: "VisaAmexDerivationOutputs"},
	{Name: "VisaAttributes"},
	{Name: "VisaPin"},
	{Name: "VisaPinVerification"},
	{Name: "VisaPinVerificationValue"},
	{Name: "WrappedKey"},
	{Name: "WrappedKeyMaterial"},
	{Name: "WrappedWorkingKey"},
}

var paymentCryptographyDataPlaneTypeByName = func() map[string]paymentCryptographyDataPlaneType {
	out := make(map[string]paymentCryptographyDataPlaneType, len(paymentCryptographyDataPlaneTypes))
	for _, dt := range paymentCryptographyDataPlaneTypes {
		out[dt.Name] = dt
	}
	return out
}()
