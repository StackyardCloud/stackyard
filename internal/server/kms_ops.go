package server

type kmsOperation struct {
	Name string
}

// AWS Key Management Service (KMS) operations sourced from:
// https://docs.aws.amazon.com/kms/latest/APIReference/API_Operations.html
var kmsOperations = []kmsOperation{
	{Name: "CancelKeyDeletion"},
	{Name: "ConnectCustomKeyStore"},
	{Name: "CreateAlias"},
	{Name: "CreateCustomKeyStore"},
	{Name: "CreateGrant"},
	{Name: "CreateKey"},
	{Name: "Decrypt"},
	{Name: "DeleteAlias"},
	{Name: "DeleteCustomKeyStore"},
	{Name: "DeleteImportedKeyMaterial"},
	{Name: "DeriveSharedSecret"},
	{Name: "DescribeCustomKeyStores"},
	{Name: "DescribeKey"},
	{Name: "DisableKey"},
	{Name: "DisableKeyRotation"},
	{Name: "DisconnectCustomKeyStore"},
	{Name: "EnableKey"},
	{Name: "EnableKeyRotation"},
	{Name: "Encrypt"},
	{Name: "GenerateDataKey"},
	{Name: "GenerateDataKeyPair"},
	{Name: "GenerateDataKeyPairWithoutPlaintext"},
	{Name: "GenerateDataKeyWithoutPlaintext"},
	{Name: "GenerateMac"},
	{Name: "GenerateRandom"},
	{Name: "GetKeyPolicy"},
	{Name: "GetKeyRotationStatus"},
	{Name: "GetParametersForImport"},
	{Name: "GetPublicKey"},
	{Name: "ImportKeyMaterial"},
	{Name: "ListAliases"},
	{Name: "ListGrants"},
	{Name: "ListKeyPolicies"},
	{Name: "ListKeyRotations"},
	{Name: "ListKeys"},
	{Name: "ListResourceTags"},
	{Name: "ListRetirableGrants"},
	{Name: "PutKeyPolicy"},
	{Name: "ReEncrypt"},
	{Name: "ReplicateKey"},
	{Name: "RetireGrant"},
	{Name: "RevokeGrant"},
	{Name: "RotateKeyOnDemand"},
	{Name: "ScheduleKeyDeletion"},
	{Name: "Sign"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateAlias"},
	{Name: "UpdateCustomKeyStore"},
	{Name: "UpdateKeyDescription"},
	{Name: "UpdatePrimaryRegion"},
	{Name: "Verify"},
	{Name: "VerifyMac"},
}

var kmsOperationByName = func() map[string]kmsOperation {
	out := make(map[string]kmsOperation, len(kmsOperations))
	for _, op := range kmsOperations {
		out[op.Name] = op
	}
	return out
}()
