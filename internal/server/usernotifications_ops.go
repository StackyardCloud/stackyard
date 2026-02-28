package server

type userNotificationsOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS User Notifications operations sourced from:
// https://docs.aws.amazon.com/notifications/latest/APIReference/API_Operations.html
var userNotificationsOperations = []userNotificationsOperation{
	{Name: "AssociateChannel", Method: "POST", URI: "/channels/associate/{arn}"},
	{Name: "AssociateManagedNotificationAccountContact", Method: "PUT", URI: "/contacts/associate-managed-notification/{contactIdentifier}"},
	{Name: "AssociateManagedNotificationAdditionalChannel", Method: "PUT", URI: "/channels/associate-managed-notification/{channelArn}"},
	{Name: "AssociateOrganizationalUnit", Method: "POST", URI: "/organizational-units/associate/{organizationalUnitId}"},
	{Name: "CreateEventRule", Method: "POST", URI: "/event-rules"},
	{Name: "CreateNotificationConfiguration", Method: "POST", URI: "/notification-configurations"},
	{Name: "DeleteEventRule", Method: "DELETE", URI: "/event-rules/{arn}"},
	{Name: "DeleteNotificationConfiguration", Method: "DELETE", URI: "/notification-configurations/{arn}"},
	{Name: "DeregisterNotificationHub", Method: "DELETE", URI: "/notification-hubs/{notificationHubRegion}"},
	{Name: "DisableNotificationsAccessForOrganization", Method: "DELETE", URI: "/organization/access"},
	{Name: "DisassociateChannel", Method: "POST", URI: "/channels/disassociate/{arn}"},
	{Name: "DisassociateManagedNotificationAccountContact", Method: "PUT", URI: "/contacts/disassociate-managed-notification/{contactIdentifier}"},
	{Name: "DisassociateManagedNotificationAdditionalChannel", Method: "PUT", URI: "/channels/disassociate-managed-notification/{channelArn}"},
	{Name: "DisassociateOrganizationalUnit", Method: "POST", URI: "/organizational-units/disassociate/{organizationalUnitId}"},
	{Name: "EnableNotificationsAccessForOrganization", Method: "POST", URI: "/organization/access"},
	{Name: "GetEventRule", Method: "GET", URI: "/event-rules/{arn}"},
	{Name: "GetManagedNotificationChildEvent", Method: "GET", URI: "/managed-notification-child-events/{arn}"},
	{Name: "GetManagedNotificationConfiguration", Method: "GET", URI: "/managed-notification-configurations/{arn}"},
	{Name: "GetManagedNotificationEvent", Method: "GET", URI: "/managed-notification-events/{arn}"},
	{Name: "GetNotificationConfiguration", Method: "GET", URI: "/notification-configurations/{arn}"},
	{Name: "GetNotificationEvent", Method: "GET", URI: "/notification-events/{arn}"},
	{Name: "GetNotificationsAccessForOrganization", Method: "GET", URI: "/organization/access"},
	{Name: "ListChannels", Method: "GET", URI: "/channels"},
	{Name: "ListEventRules", Method: "GET", URI: "/event-rules"},
	{Name: "ListManagedNotificationChannelAssociations", Method: "GET", URI: "/channels/list-managed-notification-channel-associations"},
	{Name: "ListManagedNotificationChildEvents", Method: "GET", URI: "/list-managed-notification-child-events/{aggregateManagedNotificationEventArn}"},
	{Name: "ListManagedNotificationConfigurations", Method: "GET", URI: "/managed-notification-configurations"},
	{Name: "ListManagedNotificationEvents", Method: "GET", URI: "/managed-notification-events"},
	{Name: "ListMemberAccounts", Method: "GET", URI: "/list-member-accounts"},
	{Name: "ListNotificationConfigurations", Method: "GET", URI: "/notification-configurations"},
	{Name: "ListNotificationEvents", Method: "GET", URI: "/notification-events"},
	{Name: "ListNotificationHubs", Method: "GET", URI: "/notification-hubs"},
	{Name: "ListOrganizationalUnits", Method: "GET", URI: "/organizational-units"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{arn}"},
	{Name: "RegisterNotificationHub", Method: "POST", URI: "/notification-hubs"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{arn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{arn}"},
	{Name: "UpdateEventRule", Method: "PUT", URI: "/event-rules/{arn}"},
	{Name: "UpdateNotificationConfiguration", Method: "PUT", URI: "/notification-configurations/{arn}"},
}

var userNotificationsOperationByName = func() map[string]userNotificationsOperation {
	out := make(map[string]userNotificationsOperation, len(userNotificationsOperations))
	for _, op := range userNotificationsOperations {
		out[op.Name] = op
	}
	return out
}()
