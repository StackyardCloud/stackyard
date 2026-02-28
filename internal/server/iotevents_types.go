package server

type iotEventsDataType struct {
	Name string
}

// AWS IoT Events data types sourced from:
// https://docs.aws.amazon.com/iotevents/latest/apireference/API_Types.html
var iotEventsDataTypes = []iotEventsDataType{
	{Name: "AcknowledgeFlow"},
	{Name: "Action"},
	{Name: "AlarmAction"},
	{Name: "AlarmCapabilities"},
	{Name: "AlarmEventActions"},
	{Name: "AlarmModelSummary"},
	{Name: "AlarmModelVersionSummary"},
	{Name: "AlarmNotification"},
	{Name: "AlarmRule"},
	{Name: "AnalysisResult"},
	{Name: "AnalysisResultLocation"},
	{Name: "AssetPropertyTimestamp"},
	{Name: "AssetPropertyValue"},
	{Name: "AssetPropertyVariant"},
	{Name: "Attribute"},
	{Name: "ClearTimerAction"},
	{Name: "DetectorDebugOption"},
	{Name: "DetectorModel"},
	{Name: "DetectorModelConfiguration"},
	{Name: "DetectorModelDefinition"},
	{Name: "DetectorModelSummary"},
	{Name: "DetectorModelVersionSummary"},
	{Name: "DynamoDBAction"},
	{Name: "DynamoDBv2Action"},
	{Name: "EmailConfiguration"},
	{Name: "EmailContent"},
	{Name: "EmailRecipients"},
	{Name: "Event"},
	{Name: "FirehoseAction"},
	{Name: "InitializationConfiguration"},
	{Name: "Input"},
	{Name: "InputConfiguration"},
	{Name: "InputDefinition"},
	{Name: "InputIdentifier"},
	{Name: "InputSummary"},
	{Name: "IotEventsAction"},
	{Name: "IotEventsInputIdentifier"},
	{Name: "IotSiteWiseAction"},
	{Name: "IotSiteWiseAssetModelPropertyIdentifier"},
	{Name: "IotSiteWiseInputIdentifier"},
	{Name: "IotTopicPublishAction"},
	{Name: "LambdaAction"},
	{Name: "LoggingOptions"},
	{Name: "NotificationAction"},
	{Name: "NotificationTargetActions"},
	{Name: "OnEnterLifecycle"},
	{Name: "OnExitLifecycle"},
	{Name: "OnInputLifecycle"},
	{Name: "Payload"},
	{Name: "RecipientDetail"},
	{Name: "ResetTimerAction"},
	{Name: "RoutedResource"},
	{Name: "SMSConfiguration"},
	{Name: "SNSTopicPublishAction"},
	{Name: "SSOIdentity"},
	{Name: "SetTimerAction"},
	{Name: "SetVariableAction"},
	{Name: "SimpleRule"},
	{Name: "SqsAction"},
	{Name: "State"},
	{Name: "Tag"},
	{Name: "TransitionEvent"},
}

var iotEventsDataTypeByName = func() map[string]iotEventsDataType {
	out := make(map[string]iotEventsDataType, len(iotEventsDataTypes))
	for _, dt := range iotEventsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
