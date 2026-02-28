package server

type iotGreengrassOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS IoT Greengrass (V2) operations sourced from:
// https://docs.aws.amazon.com/greengrass/v2/APIReference/API_Operations.html
var iotGreengrassOperations = []iotGreengrassOperation{
	{Name: "AssociateServiceRoleToAccount", Method: "PUT", URI: "/greengrass/servicerole"},
	{Name: "BatchAssociateClientDeviceWithCoreDevice", Method: "POST", URI: "/greengrass/v2/coreDevices/{coreDeviceThingName}/associateClientDevices"},
	{Name: "BatchDisassociateClientDeviceFromCoreDevice", Method: "POST", URI: "/greengrass/v2/coreDevices/{coreDeviceThingName}/disassociateClientDevices"},
	{Name: "CancelDeployment", Method: "POST", URI: "/greengrass/v2/deployments/{deploymentId}/cancel"},
	{Name: "CreateComponentVersion", Method: "POST", URI: "/greengrass/v2/createComponentVersion"},
	{Name: "CreateDeployment", Method: "POST", URI: "/greengrass/v2/deployments"},
	{Name: "DeleteComponent", Method: "DELETE", URI: "/greengrass/v2/components/{arn}"},
	{Name: "DeleteCoreDevice", Method: "DELETE", URI: "/greengrass/v2/coreDevices/{coreDeviceThingName}"},
	{Name: "DeleteDeployment", Method: "DELETE", URI: "/greengrass/v2/deployments/{deploymentId}"},
	{Name: "DescribeComponent", Method: "GET", URI: "/greengrass/v2/components/{arn}/metadata"},
	{Name: "DisassociateServiceRoleFromAccount", Method: "DELETE", URI: "/greengrass/servicerole"},
	{Name: "GetComponent", Method: "GET", URI: "/greengrass/v2/components/{arn}"},
	{Name: "GetComponentVersionArtifact", Method: "GET", URI: "/greengrass/v2/components/{arn}/artifacts/{artifactName+}"},
	{Name: "GetConnectivityInfo", Method: "GET", URI: "/greengrass/things/{thingName}/connectivityInfo"},
	{Name: "GetCoreDevice", Method: "GET", URI: "/greengrass/v2/coreDevices/{coreDeviceThingName}"},
	{Name: "GetDeployment", Method: "GET", URI: "/greengrass/v2/deployments/{deploymentId}"},
	{Name: "GetServiceRoleForAccount", Method: "GET", URI: "/greengrass/servicerole"},
	{Name: "ListClientDevicesAssociatedWithCoreDevice", Method: "GET", URI: "/greengrass/v2/coreDevices/{coreDeviceThingName}/associatedClientDevices"},
	{Name: "ListComponents", Method: "GET", URI: "/greengrass/v2/components"},
	{Name: "ListComponentVersions", Method: "GET", URI: "/greengrass/v2/components/{arn}/versions"},
	{Name: "ListCoreDevices", Method: "GET", URI: "/greengrass/v2/coreDevices"},
	{Name: "ListDeployments", Method: "GET", URI: "/greengrass/v2/deployments"},
	{Name: "ListEffectiveDeployments", Method: "GET", URI: "/greengrass/v2/coreDevices/{coreDeviceThingName}/effectiveDeployments"},
	{Name: "ListInstalledComponents", Method: "GET", URI: "/greengrass/v2/coreDevices/{coreDeviceThingName}/installedComponents"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ResolveComponentCandidates", Method: "POST", URI: "/greengrass/v2/resolveComponentCandidates"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateConnectivityInfo", Method: "PUT", URI: "/greengrass/things/{thingName}/connectivityInfo"},
}

var iotGreengrassOperationByName = func() map[string]iotGreengrassOperation {
	out := make(map[string]iotGreengrassOperation, len(iotGreengrassOperations))
	for _, op := range iotGreengrassOperations {
		out[op.Name] = op
	}
	return out
}()
