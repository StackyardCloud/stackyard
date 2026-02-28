package server

type storageGatewayDataType struct {
	Name string
}

// AWS Storage Gateway data types sourced from:
// https://docs.aws.amazon.com/storagegateway/latest/APIReference/API_Types.html
var storageGatewayDataTypes = []storageGatewayDataType{
	{Name: "AutomaticTapeCreationPolicyInfo"},
	{Name: "AutomaticTapeCreationRule"},
	{Name: "BandwidthRateLimitInterval"},
	{Name: "CacheAttributes"},
	{Name: "CacheReportFilter"},
	{Name: "CacheReportInfo"},
	{Name: "CachediSCSIVolume"},
	{Name: "ChapInfo"},
	{Name: "DeviceiSCSIAttributes"},
	{Name: "Disk"},
	{Name: "EndpointNetworkConfiguration"},
	{Name: "FileShareInfo"},
	{Name: "FileSystemAssociationInfo"},
	{Name: "FileSystemAssociationStatusDetail"},
	{Name: "FileSystemAssociationSummary"},
	{Name: "GatewayInfo"},
	{Name: "NFSFileShareDefaults"},
	{Name: "NFSFileShareInfo"},
	{Name: "NetworkInterface"},
	{Name: "PoolInfo"},
	{Name: "SMBFileShareInfo"},
	{Name: "SMBLocalGroups"},
	{Name: "SoftwareUpdatePreferences"},
	{Name: "StorageGatewayError"},
	{Name: "StorediSCSIVolume"},
	{Name: "Tag"},
	{Name: "Tape"},
	{Name: "TapeArchive"},
	{Name: "TapeInfo"},
	{Name: "TapeRecoveryPointInfo"},
	{Name: "UpdateVTLDeviceType"},
	{Name: "VTLDevice"},
	{Name: "VolumeInfo"},
	{Name: "VolumeRecoveryPointInfo"},
	{Name: "VolumeiSCSIAttributes"},
}

var storageGatewayDataTypeByName = func() map[string]storageGatewayDataType {
	out := make(map[string]storageGatewayDataType, len(storageGatewayDataTypes))
	for _, dt := range storageGatewayDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
