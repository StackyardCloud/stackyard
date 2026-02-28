package server

type mediaTailorDataType struct {
	Name string
}

// AWS Elemental MediaTailor data types sourced from:
// https://docs.aws.amazon.com/mediatailor/latest/apireference/API_Types.html
var mediaTailorDataTypes = []mediaTailorDataType{
	{Name: "AccessConfiguration"},
	{Name: "AdBreak"},
	{Name: "AdBreakOpportunity"},
	{Name: "AdConditioningConfiguration"},
	{Name: "AdMarkerPassthrough"},
	{Name: "AdsInteractionLog"},
	{Name: "Alert"},
	{Name: "AlternateMedia"},
	{Name: "AudienceMedia"},
	{Name: "AvailMatchingCriteria"},
	{Name: "AvailSuppression"},
	{Name: "Bumper"},
	{Name: "CdnConfiguration"},
	{Name: "Channel"},
	{Name: "ClipRange"},
	{Name: "DashConfiguration"},
	{Name: "DashConfigurationForPut"},
	{Name: "DashPlaylistSettings"},
	{Name: "DefaultSegmentDeliveryConfiguration"},
	{Name: "HlsConfiguration"},
	{Name: "HlsPlaylistSettings"},
	{Name: "HttpConfiguration"},
	{Name: "HttpPackageConfiguration"},
	{Name: "KeyValuePair"},
	{Name: "LivePreRollConfiguration"},
	{Name: "LiveSource"},
	{Name: "LogConfiguration"},
	{Name: "LogConfigurationForChannel"},
	{Name: "ManifestProcessingRules"},
	{Name: "ManifestServiceInteractionLog"},
	{Name: "PlaybackConfiguration"},
	{Name: "PrefetchConsumption"},
	{Name: "PrefetchRetrieval"},
	{Name: "PrefetchSchedule"},
	{Name: "RecurringConsumption"},
	{Name: "RecurringPrefetchConfiguration"},
	{Name: "RecurringRetrieval"},
	{Name: "RequestOutputItem"},
	{Name: "ResponseOutputItem"},
	{Name: "ScheduleAdBreak"},
	{Name: "ScheduleConfiguration"},
	{Name: "ScheduleEntry"},
	{Name: "SecretsManagerAccessTokenConfiguration"},
	{Name: "SegmentDeliveryConfiguration"},
	{Name: "SegmentationDescriptor"},
	{Name: "SlateSource"},
	{Name: "SourceLocation"},
	{Name: "SpliceInsertMessage"},
	{Name: "TimeShiftConfiguration"},
	{Name: "TimeSignalMessage"},
	{Name: "TrafficShapingRetrievalWindow"},
	{Name: "Transition"},
	{Name: "UpdateProgramScheduleConfiguration"},
	{Name: "UpdateProgramTransition"},
	{Name: "UpdateVodSource"},
	{Name: "VodSource"},
}

var mediaTailorDataTypeByName = func() map[string]mediaTailorDataType {
	out := make(map[string]mediaTailorDataType, len(mediaTailorDataTypes))
	for _, dt := range mediaTailorDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
