package server

type mediaPackageDataType struct {
	Name string
}

// AWS Elemental MediaPackage V2 Live API data types sourced from:
// https://docs.aws.amazon.com/mediapackage/latest/APIReference/API_Types.html
var mediaPackageDataTypes = []mediaPackageDataType{
	{Name: "CdnAuthConfiguration"},
	{Name: "ChannelGroupListConfiguration"},
	{Name: "ChannelListConfiguration"},
	{Name: "CreateDashManifestConfiguration"},
	{Name: "CreateHlsManifestConfiguration"},
	{Name: "CreateLowLatencyHlsManifestConfiguration"},
	{Name: "CreateMssManifestConfiguration"},
	{Name: "DashBaseUrl"},
	{Name: "DashDvbFontDownload"},
	{Name: "DashDvbMetricsReporting"},
	{Name: "DashDvbSettings"},
	{Name: "DashProgramInformation"},
	{Name: "DashSubtitleConfiguration"},
	{Name: "DashTtmlConfiguration"},
	{Name: "DashUtcTiming"},
	{Name: "Destination"},
	{Name: "Encryption"},
	{Name: "EncryptionContractConfiguration"},
	{Name: "EncryptionMethod"},
	{Name: "FilterConfiguration"},
	{Name: "ForceEndpointErrorConfiguration"},
	{Name: "GetDashManifestConfiguration"},
	{Name: "GetHlsManifestConfiguration"},
	{Name: "GetLowLatencyHlsManifestConfiguration"},
	{Name: "GetMssManifestConfiguration"},
	{Name: "HarvestedDashManifest"},
	{Name: "HarvestedHlsManifest"},
	{Name: "HarvestedLowLatencyHlsManifest"},
	{Name: "HarvestedManifests"},
	{Name: "HarvesterScheduleConfiguration"},
	{Name: "HarvestJob"},
	{Name: "IngestEndpoint"},
	{Name: "InputSwitchConfiguration"},
	{Name: "ListDashManifestConfiguration"},
	{Name: "ListHlsManifestConfiguration"},
	{Name: "ListLowLatencyHlsManifestConfiguration"},
	{Name: "ListMssManifestConfiguration"},
	{Name: "OriginEndpointListConfiguration"},
	{Name: "OutputHeaderConfiguration"},
	{Name: "S3DestinationConfig"},
	{Name: "Scte"},
	{Name: "ScteDash"},
	{Name: "ScteHls"},
	{Name: "Segment"},
	{Name: "SpekeKeyProvider"},
	{Name: "StartTag"},
}

var mediaPackageDataTypeByName = func() map[string]mediaPackageDataType {
	out := make(map[string]mediaPackageDataType, len(mediaPackageDataTypes))
	for _, dt := range mediaPackageDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
