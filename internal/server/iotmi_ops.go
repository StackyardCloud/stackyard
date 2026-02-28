package server

type iotMIAction struct {
	Name   string
	Method string
	URI    string
}

// AWS IoT Managed Integrations actions sourced from:
// https://docs.aws.amazon.com/iot-mi/latest/APIReference/API_Operations.html
var iotMIActions = []iotMIAction{
	{Name: "CreateAccountAssociation", Method: "POST", URI: "/account-associations"},
	{Name: "CreateCloudConnector", Method: "POST", URI: "/cloud-connectors"},
	{Name: "CreateConnectorDestination", Method: "POST", URI: "/connector-destinations"},
	{Name: "CreateCredentialLocker", Method: "POST", URI: "/credential-lockers"},
	{Name: "CreateDestination", Method: "POST", URI: "/destinations"},
	{Name: "CreateEventLogConfiguration", Method: "POST", URI: "/event-log-configurations"},
	{Name: "CreateManagedThing", Method: "POST", URI: "/managed-things"},
	{Name: "CreateNotificationConfiguration", Method: "POST", URI: "/notification-configurations"},
	{Name: "CreateOtaTask", Method: "POST", URI: "/ota-tasks"},
	{Name: "CreateOtaTaskConfiguration", Method: "POST", URI: "/ota-task-configurations"},
	{Name: "CreateProvisioningProfile", Method: "POST", URI: "/provisioning-profiles"},
	{Name: "DeleteAccountAssociation", Method: "DELETE", URI: "/account-associations/{AccountAssociationId}"},
	{Name: "DeleteCloudConnector", Method: "DELETE", URI: "/cloud-connectors/{Identifier}"},
	{Name: "DeleteConnectorDestination", Method: "DELETE", URI: "/connector-destinations/{Identifier}"},
	{Name: "DeleteCredentialLocker", Method: "DELETE", URI: "/credential-lockers/{Identifier}"},
	{Name: "DeleteDestination", Method: "DELETE", URI: "/destinations/{Name}"},
	{Name: "DeleteEventLogConfiguration", Method: "DELETE", URI: "/event-log-configurations/{Id}"},
	{Name: "DeleteManagedThing", Method: "DELETE", URI: "/managed-things/{Identifier}?Force={Force}"},
	{Name: "DeleteNotificationConfiguration", Method: "DELETE", URI: "/notification-configurations/{EventType}"},
	{Name: "DeleteOtaTask", Method: "DELETE", URI: "/ota-tasks/{Identifier}"},
	{Name: "DeleteOtaTaskConfiguration", Method: "DELETE", URI: "/ota-task-configurations/{Identifier}"},
	{Name: "DeleteProvisioningProfile", Method: "DELETE", URI: "/provisioning-profiles/{Identifier}"},
	{Name: "DeregisterAccountAssociation", Method: "PUT", URI: "/managed-thing-associations/deregister"},
	{Name: "GetAccountAssociation", Method: "GET", URI: "/account-associations/{AccountAssociationId}"},
	{Name: "GetCloudConnector", Method: "GET", URI: "/cloud-connectors/{Identifier}"},
	{Name: "GetConnectorDestination", Method: "GET", URI: "/connector-destinations/{Identifier}"},
	{Name: "GetCredentialLocker", Method: "GET", URI: "/credential-lockers/{Identifier}"},
	{Name: "GetCustomEndpoint", Method: "GET", URI: "/custom-endpoint"},
	{Name: "GetDefaultEncryptionConfiguration", Method: "GET", URI: "/configuration/account/encryption"},
	{Name: "GetDestination", Method: "GET", URI: "/destinations/{Name}"},
	{Name: "GetDeviceDiscovery", Method: "GET", URI: "/device-discoveries/{Identifier}"},
	{Name: "GetEventLogConfiguration", Method: "GET", URI: "/event-log-configurations/{Id}"},
	{Name: "GetHubConfiguration", Method: "GET", URI: "/hub-configuration"},
	{Name: "GetManagedThing", Method: "GET", URI: "/managed-things/{Identifier}"},
	{Name: "GetManagedThingCapabilities", Method: "GET", URI: "/managed-things-capabilities/{Identifier}"},
	{Name: "GetManagedThingCertificate", Method: "GET", URI: "/managed-things-certificate/{Identifier}"},
	{Name: "GetManagedThingConnectivityData", Method: "POST", URI: "/managed-things-connectivity-data/{Identifier}"},
	{Name: "GetManagedThingMetaData", Method: "GET", URI: "/managed-things-metadata/{Identifier}"},
	{Name: "GetManagedThingState", Method: "GET", URI: "/managed-thing-states/{ManagedThingId}"},
	{Name: "GetNotificationConfiguration", Method: "GET", URI: "/notification-configurations/{EventType}"},
	{Name: "GetOtaTask", Method: "GET", URI: "/ota-tasks/{Identifier}"},
	{Name: "GetOtaTaskConfiguration", Method: "GET", URI: "/ota-task-configurations/{Identifier}"},
	{Name: "GetProvisioningProfile", Method: "GET", URI: "/provisioning-profiles/{Identifier}"},
	{Name: "GetRuntimeLogConfiguration", Method: "GET", URI: "/runtime-log-configurations/{ManagedThingId}"},
	{Name: "GetSchemaVersion", Method: "GET", URI: "/schema-versions/{Type}/{SchemaVersionedId}?Format={Format}"},
	{Name: "ListAccountAssociations", Method: "GET", URI: "/account-associations?ConnectorDestinationId={ConnectorDestinationId}&MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListCloudConnectors", Method: "GET", URI: "/cloud-connectors?LambdaArn={LambdaArn}&MaxResults={MaxResults}&NextToken={NextToken}&Type={Type}"},
	{Name: "ListConnectorDestinations", Method: "GET", URI: "/connector-destinations?CloudConnectorId={CloudConnectorId}&MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListCredentialLockers", Method: "GET", URI: "/credential-lockers?MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListDestinations", Method: "GET", URI: "/destinations?MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListDeviceDiscoveries", Method: "GET", URI: "/device-discoveries?MaxResults={MaxResults}&NextToken={NextToken}&StatusFilter={StatusFilter}&TypeFilter={TypeFilter}"},
	{Name: "ListDiscoveredDevices", Method: "GET", URI: "/device-discoveries/{Identifier}/devices?MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListEventLogConfigurations", Method: "GET", URI: "/event-log-configurations?MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListManagedThingAccountAssociations", Method: "GET", URI: "/managed-thing-associations?AccountAssociationId={AccountAssociationId}&ManagedThingId={ManagedThingId}&MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListManagedThingSchemas", Method: "GET", URI: "/managed-thing-schemas/{Identifier}?CapabilityIdFilter={CapabilityIdFilter}&EndpointIdFilter={EndpointIdFilter}&MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListManagedThings", Method: "GET", URI: "/managed-things?ConnectorDestinationIdFilter={ConnectorDestinationIdFilter}&ConnectorDeviceIdFilter={ConnectorDeviceIdFilter}&ConnectorPolicyIdFilter={ConnectorPolicyIdFilter}&CredentialLockerFilter={CredentialLockerFilter}&MaxResults={MaxResults}&NextToken={NextToken}&OwnerFilter={OwnerFilter}&ParentControllerIdentifierFilter={ParentControllerIdentifierFilter}&ProvisioningStatusFilter={ProvisioningStatusFilter}&RoleFilter={RoleFilter}&SerialNumberFilter={SerialNumberFilter}"},
	{Name: "ListNotificationConfigurations", Method: "GET", URI: "/notification-configurations?MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListOtaTaskConfigurations", Method: "GET", URI: "/ota-task-configurations?MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListOtaTaskExecutions", Method: "GET", URI: "/ota-tasks/{Identifier}/devices?MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListOtaTasks", Method: "GET", URI: "/ota-tasks?MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListProvisioningProfiles", Method: "GET", URI: "/provisioning-profiles?MaxResults={MaxResults}&NextToken={NextToken}"},
	{Name: "ListSchemaVersions", Method: "GET", URI: "/schema-versions/{Type}?MaxResults={MaxResults}&NamespaceFilter={Namespace}&NextToken={NextToken}&SchemaIdFilter={SchemaId}&SemanticVersionFilter={SemanticVersion}&VisibilityFilter={Visibility}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "PutDefaultEncryptionConfiguration", Method: "POST", URI: "/configuration/account/encryption"},
	{Name: "PutHubConfiguration", Method: "PUT", URI: "/hub-configuration"},
	{Name: "PutRuntimeLogConfiguration", Method: "PUT", URI: "/runtime-log-configurations/{ManagedThingId}"},
	{Name: "RegisterAccountAssociation", Method: "PUT", URI: "/managed-thing-associations/register"},
	{Name: "RegisterCustomEndpoint", Method: "POST", URI: "/custom-endpoint"},
	{Name: "ResetRuntimeLogConfiguration", Method: "DELETE", URI: "/runtime-log-configurations/{ManagedThingId}"},
	{Name: "SendConnectorEvent", Method: "POST", URI: "/connector-event/{ConnectorId}"},
	{Name: "SendManagedThingCommand", Method: "POST", URI: "/managed-things-command/{ManagedThingId}"},
	{Name: "StartAccountAssociationRefresh", Method: "POST", URI: "/account-associations/{AccountAssociationId}/refresh"},
	{Name: "StartDeviceDiscovery", Method: "POST", URI: "/device-discoveries"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}?tagKeys={TagKeys}"},
	{Name: "UpdateAccountAssociation", Method: "PUT", URI: "/account-associations/{AccountAssociationId}"},
	{Name: "UpdateCloudConnector", Method: "PUT", URI: "/cloud-connectors/{Identifier}"},
	{Name: "UpdateConnectorDestination", Method: "PUT", URI: "/connector-destinations/{Identifier}"},
	{Name: "UpdateDestination", Method: "PUT", URI: "/destinations/{Name}"},
	{Name: "UpdateEventLogConfiguration", Method: "PATCH", URI: "/event-log-configurations/{Id}"},
	{Name: "UpdateManagedThing", Method: "PUT", URI: "/managed-things/{Identifier}"},
	{Name: "UpdateNotificationConfiguration", Method: "PUT", URI: "/notification-configurations/{EventType}"},
	{Name: "UpdateOtaTask", Method: "PUT", URI: "/ota-tasks/{Identifier}"},
}

var iotMIActionByName = func() map[string]iotMIAction {
	out := make(map[string]iotMIAction, len(iotMIActions))
	for _, action := range iotMIActions {
		out[action.Name] = action
	}
	return out
}()
