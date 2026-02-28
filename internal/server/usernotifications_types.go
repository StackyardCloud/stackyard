package server

type userNotificationsDataType struct {
	Name string
}

// AWS User Notifications data types sourced from:
// https://docs.aws.amazon.com/notifications/latest/APIReference/API_Types.html
var userNotificationsDataTypes = []userNotificationsDataType{
	{Name: "AggregationDetail"},
	{Name: "AggregationKey"},
	{Name: "AggregationSummary"},
	{Name: "Dimension"},
	{Name: "EventRuleStatusSummary"},
	{Name: "EventRuleStructure"},
	{Name: "ManagedNotificationChannelAssociationSummary"},
	{Name: "ManagedNotificationChildEvent"},
	{Name: "ManagedNotificationChildEventOverview"},
	{Name: "ManagedNotificationChildEventSummary"},
	{Name: "ManagedNotificationConfigurationStructure"},
	{Name: "ManagedNotificationEvent"},
	{Name: "ManagedNotificationEventOverview"},
	{Name: "ManagedNotificationEventSummary"},
	{Name: "ManagedSourceEventMetadataSummary"},
	{Name: "MediaElement"},
	{Name: "MemberAccount"},
	{Name: "MessageComponents"},
	{Name: "MessageComponentsSummary"},
	{Name: "NotificationConfigurationStructure"},
	{Name: "NotificationEvent"},
	{Name: "NotificationEventOverview"},
	{Name: "NotificationEventSummary"},
	{Name: "NotificationHubOverview"},
	{Name: "NotificationHubStatusSummary"},
	{Name: "NotificationsAccessForOrganization"},
	{Name: "Resource"},
	{Name: "SourceEventMetadata"},
	{Name: "SourceEventMetadataSummary"},
	{Name: "SummarizationDimensionDetail"},
	{Name: "SummarizationDimensionOverview"},
	{Name: "TextPartValue"},
	{Name: "ValidationExceptionField"},
	{Name: "UpdateNotificationConfiguration"},
}

var userNotificationsDataTypeByName = func() map[string]userNotificationsDataType {
	out := make(map[string]userNotificationsDataType, len(userNotificationsDataTypes))
	for _, t := range userNotificationsDataTypes {
		out[t.Name] = t
	}
	return out
}()
