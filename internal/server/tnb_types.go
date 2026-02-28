package server

// AWS Telco Network Builder (TNB) data types sourced from:
// https://docs.aws.amazon.com/tnb/latest/APIReference/API_Types.html
var tnbDataTypes = []string{
	"ErrorInfo",
	"FunctionArtifactMeta",
	"GetSolFunctionInstanceMetadata",
	"GetSolFunctionPackageMetadata",
	"GetSolInstantiatedVnfInfo",
	"GetSolNetworkInstanceMetadata",
	"GetSolNetworkOperationMetadata",
	"GetSolNetworkOperationTaskDetails",
	"GetSolNetworkPackageMetadata",
	"GetSolVnfInfo",
	"GetSolVnfcResourceInfo",
	"GetSolVnfcResourceInfoMetadata",
	"InstantiateMetadata",
	"LcmOperationInfo",
	"ListSolFunctionInstanceInfo",
	"ListSolFunctionInstanceMetadata",
	"ListSolFunctionPackageInfo",
	"ListSolFunctionPackageMetadata",
	"ListSolNetworkInstanceInfo",
	"ListSolNetworkInstanceMetadata",
	"ListSolNetworkOperationsInfo",
	"ListSolNetworkOperationsMetadata",
	"ListSolNetworkPackageInfo",
	"ListSolNetworkPackageMetadata",
	"ModifyVnfInfoMetadata",
	"NetworkArtifactMeta",
	"ProblemDetails",
	"PutSolFunctionPackageContentMetadata",
	"PutSolNetworkPackageContentMetadata",
	"ToscaOverride",
	"UpdateNsMetadata",
	"UpdateSolNetworkModify",
	"UpdateSolNetworkServiceData",
	"ValidateSolFunctionPackageContentMetadata",
	"ValidateSolNetworkPackageContentMetadata",
}

var tnbDataTypeByName = func() map[string]struct{} {
	out := make(map[string]struct{}, len(tnbDataTypes))
	for _, typeName := range tnbDataTypes {
		out[typeName] = struct{}{}
	}
	return out
}()
