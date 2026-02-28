package server

type iotFleetWiseOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS IoT FleetWise actions sourced from:
// https://docs.aws.amazon.com/iot-fleetwise/latest/APIReference/API_Operations.html
var iotFleetWiseOperations = []iotFleetWiseOperation{
	{Name: "AssociateVehicleFleet", Method: "POST", URI: "/"},
	{Name: "BatchCreateVehicle", Method: "POST", URI: "/"},
	{Name: "BatchUpdateVehicle", Method: "POST", URI: "/"},
	{Name: "CreateCampaign", Method: "POST", URI: "/"},
	{Name: "CreateDecoderManifest", Method: "POST", URI: "/"},
	{Name: "CreateFleet", Method: "POST", URI: "/"},
	{Name: "CreateModelManifest", Method: "POST", URI: "/"},
	{Name: "CreateSignalCatalog", Method: "POST", URI: "/"},
	{Name: "CreateStateTemplate", Method: "POST", URI: "/"},
	{Name: "CreateVehicle", Method: "POST", URI: "/"},
	{Name: "DeleteCampaign", Method: "POST", URI: "/"},
	{Name: "DeleteDecoderManifest", Method: "POST", URI: "/"},
	{Name: "DeleteFleet", Method: "POST", URI: "/"},
	{Name: "DeleteModelManifest", Method: "POST", URI: "/"},
	{Name: "DeleteSignalCatalog", Method: "POST", URI: "/"},
	{Name: "DeleteStateTemplate", Method: "POST", URI: "/"},
	{Name: "DeleteVehicle", Method: "POST", URI: "/"},
	{Name: "DisassociateVehicleFleet", Method: "POST", URI: "/"},
	{Name: "GetCampaign", Method: "POST", URI: "/"},
	{Name: "GetDecoderManifest", Method: "POST", URI: "/"},
	{Name: "GetEncryptionConfiguration", Method: "POST", URI: "/"},
	{Name: "GetFleet", Method: "POST", URI: "/"},
	{Name: "GetLoggingOptions", Method: "POST", URI: "/"},
	{Name: "GetModelManifest", Method: "POST", URI: "/"},
	{Name: "GetRegisterAccountStatus", Method: "POST", URI: "/"},
	{Name: "GetSignalCatalog", Method: "POST", URI: "/"},
	{Name: "GetStateTemplate", Method: "POST", URI: "/"},
	{Name: "GetVehicle", Method: "POST", URI: "/"},
	{Name: "GetVehicleStatus", Method: "POST", URI: "/"},
	{Name: "ImportDecoderManifest", Method: "POST", URI: "/"},
	{Name: "ImportSignalCatalog", Method: "POST", URI: "/"},
	{Name: "ListCampaigns", Method: "POST", URI: "/"},
	{Name: "ListDecoderManifestNetworkInterfaces", Method: "POST", URI: "/"},
	{Name: "ListDecoderManifests", Method: "POST", URI: "/"},
	{Name: "ListDecoderManifestSignals", Method: "POST", URI: "/"},
	{Name: "ListFleets", Method: "POST", URI: "/"},
	{Name: "ListFleetsForVehicle", Method: "POST", URI: "/"},
	{Name: "ListModelManifestNodes", Method: "POST", URI: "/"},
	{Name: "ListModelManifests", Method: "POST", URI: "/"},
	{Name: "ListSignalCatalogNodes", Method: "POST", URI: "/"},
	{Name: "ListSignalCatalogs", Method: "POST", URI: "/"},
	{Name: "ListStateTemplates", Method: "POST", URI: "/"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/"},
	{Name: "ListVehicles", Method: "POST", URI: "/"},
	{Name: "ListVehiclesInFleet", Method: "POST", URI: "/"},
	{Name: "PutEncryptionConfiguration", Method: "POST", URI: "/"},
	{Name: "PutLoggingOptions", Method: "POST", URI: "/"},
	{Name: "RegisterAccount", Method: "POST", URI: "/"},
	{Name: "TagResource", Method: "POST", URI: "/"},
	{Name: "UntagResource", Method: "POST", URI: "/"},
	{Name: "UpdateCampaign", Method: "POST", URI: "/"},
	{Name: "UpdateDecoderManifest", Method: "POST", URI: "/"},
	{Name: "UpdateFleet", Method: "POST", URI: "/"},
	{Name: "UpdateModelManifest", Method: "POST", URI: "/"},
	{Name: "UpdateSignalCatalog", Method: "POST", URI: "/"},
	{Name: "UpdateStateTemplate", Method: "POST", URI: "/"},
	{Name: "UpdateVehicle", Method: "POST", URI: "/"},
}

var iotFleetWiseOperationByName = func() map[string]iotFleetWiseOperation {
	out := make(map[string]iotFleetWiseOperation, len(iotFleetWiseOperations))
	for _, op := range iotFleetWiseOperations {
		out[op.Name] = op
	}
	return out
}()
