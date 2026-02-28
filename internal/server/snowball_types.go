package server

type snowballDataType struct {
	Name string
}

// AWS Snowball Edge data types sourced from:
// https://docs.aws.amazon.com/snowball/latest/api-reference/API_Types.html
var snowballDataTypes = []snowballDataType{
	{Name: "Address"},
	{Name: "ClusterListEntry"},
	{Name: "ClusterMetadata"},
	{Name: "CompatibleImage"},
	{Name: "DataTransfer"},
	{Name: "DependentService"},
	{Name: "DeviceConfiguration"},
	{Name: "EKSOnDeviceServiceConfiguration"},
	{Name: "Ec2AmiResource"},
	{Name: "EventTriggerDefinition"},
	{Name: "INDTaxDocuments"},
	{Name: "JobListEntry"},
	{Name: "JobLogs"},
	{Name: "JobMetadata"},
	{Name: "JobResource"},
	{Name: "KeyRange"},
	{Name: "LambdaResource"},
	{Name: "LongTermPricingListEntry"},
	{Name: "NFSOnDeviceServiceConfiguration"},
	{Name: "Notification"},
	{Name: "OnDeviceServiceConfiguration"},
	{Name: "PickupDetails"},
	{Name: "S3OnDeviceServiceConfiguration"},
	{Name: "S3Resource"},
	{Name: "ServiceVersion"},
	{Name: "Shipment"},
	{Name: "ShippingDetails"},
	{Name: "SnowconeDeviceConfiguration"},
	{Name: "TGWOnDeviceServiceConfiguration"},
	{Name: "TargetOnDeviceService"},
	{Name: "TaxDocuments"},
	{Name: "WirelessConnection"},
}

var snowballDataTypeByName = func() map[string]snowballDataType {
	out := make(map[string]snowballDataType, len(snowballDataTypes))
	for _, dt := range snowballDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
