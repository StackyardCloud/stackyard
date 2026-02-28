package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type iotMIStore struct {
	mu sync.Mutex

	nextID int64

	accountAssociations        map[string]map[string]any
	cloudConnectors            map[string]map[string]any
	connectorDestinations      map[string]map[string]any
	credentialLockers          map[string]map[string]any
	destinations               map[string]map[string]any
	eventLogConfigurations     map[string]map[string]any
	managedThings              map[string]map[string]any
	notificationConfigurations map[string]map[string]any
	otaTaskConfigurations      map[string]map[string]any
	otaTasks                   map[string]map[string]any
	provisioningProfiles       map[string]map[string]any
	deviceDiscoveries          map[string]map[string]any
	runtimeLogConfigurations   map[string]map[string]any
	tags                       map[string]map[string]string

	managedThingAssociations []map[string]any
	hubConfiguration         map[string]any
	defaultEncryptionConfig  map[string]any
	customEndpoint           map[string]any
}

func newIoTMIStore() *iotMIStore {
	now := time.Now().UTC().Format(time.RFC3339)
	thingID := "thing-000001"
	associationID := "assoc-000001"
	connectorID := "connector-000001"
	destinationName := "stackyard-destination"
	otaTaskID := "ota-task-000001"
	otaTaskConfigID := "ota-config-000001"
	provisioningProfileID := "profile-000001"
	resourceARN := iotMIManagedThingARN(thingID)

	return &iotMIStore{
		nextID: 2,
		accountAssociations: map[string]map[string]any{
			associationID: {
				"accountAssociationId":   associationID,
				"connectorDestinationId": "connector-destination-000001",
				"status":                 "ACTIVE",
				"createdAt":              now,
				"updatedAt":              now,
			},
		},
		cloudConnectors: map[string]map[string]any{
			connectorID: {
				"identifier": connectorID,
				"type":       "LAMBDA",
				"lambdaArn":  "arn:aws:lambda:us-east-1:123456789012:function:stackyard-iotmi",
				"status":     "ACTIVE",
				"createdAt":  now,
				"updatedAt":  now,
			},
		},
		connectorDestinations: map[string]map[string]any{
			"connector-destination-000001": {
				"identifier":       "connector-destination-000001",
				"cloudConnectorId": connectorID,
				"status":           "ACTIVE",
				"createdAt":        now,
				"updatedAt":        now,
			},
		},
		credentialLockers: map[string]map[string]any{
			"locker-000001": {
				"identifier": "locker-000001",
				"type":       "SECRETS_MANAGER",
				"status":     "ACTIVE",
				"createdAt":  now,
				"updatedAt":  now,
			},
		},
		destinations: map[string]map[string]any{
			destinationName: {
				"name":      destinationName,
				"status":    "ACTIVE",
				"createdAt": now,
				"updatedAt": now,
			},
		},
		eventLogConfigurations: map[string]map[string]any{
			"event-log-000001": {
				"id":        "event-log-000001",
				"logLevel":  "INFO",
				"status":    "ENABLED",
				"createdAt": now,
				"updatedAt": now,
			},
		},
		managedThings: map[string]map[string]any{
			thingID: {
				"identifier":             thingID,
				"managedThingId":         thingID,
				"managedThingArn":        resourceARN,
				"name":                   "stackyard-managed-thing",
				"serialNumber":           "SN-000001",
				"provisioningStatus":     "COMPLETED",
				"connectorDestinationId": "connector-destination-000001",
				"createdAt":              now,
				"updatedAt":              now,
			},
		},
		notificationConfigurations: map[string]map[string]any{
			"DEVICE_CONNECTED": {
				"eventType": "DEVICE_CONNECTED",
				"status":    "ENABLED",
				"createdAt": now,
				"updatedAt": now,
			},
		},
		otaTaskConfigurations: map[string]map[string]any{
			otaTaskConfigID: {
				"identifier": otaTaskConfigID,
				"name":       "stackyard-ota-config",
				"status":     "ACTIVE",
				"createdAt":  now,
				"updatedAt":  now,
			},
		},
		otaTasks: map[string]map[string]any{
			otaTaskID: {
				"identifier":             otaTaskID,
				"name":                   "stackyard-ota-task",
				"status":                 "IN_PROGRESS",
				"otaTaskConfigurationId": otaTaskConfigID,
				"createdAt":              now,
				"updatedAt":              now,
			},
		},
		provisioningProfiles: map[string]map[string]any{
			provisioningProfileID: {
				"identifier": provisioningProfileID,
				"name":       "stackyard-profile",
				"status":     "ACTIVE",
				"createdAt":  now,
				"updatedAt":  now,
			},
		},
		deviceDiscoveries: map[string]map[string]any{
			"discovery-000001": {
				"identifier": "discovery-000001",
				"type":       "MATTER",
				"status":     "SUCCEEDED",
				"createdAt":  now,
				"updatedAt":  now,
			},
		},
		runtimeLogConfigurations: map[string]map[string]any{
			thingID: {
				"managedThingId": thingID,
				"logLevel":       "INFO",
				"updatedAt":      now,
			},
		},
		tags: map[string]map[string]string{
			resourceARN: {"stackyard": "true"},
		},
		managedThingAssociations: []map[string]any{
			{
				"accountAssociationId": associationID,
				"managedThingId":       thingID,
				"status":               "REGISTERED",
			},
		},
		hubConfiguration: map[string]any{
			"hubTokenTimerExpirySettingInSeconds": 3600,
			"updatedAt":                           now,
		},
		defaultEncryptionConfig: map[string]any{
			"kmsKeyArn": "arn:aws:kms:us-east-1:123456789012:key/stackyard-iotmi",
			"updatedAt": now,
		},
		customEndpoint: map[string]any{
			"endpointAddress": "stackyard-iot-mi.custom.iot.us-east-1.amazonaws.com",
			"status":          "ENABLED",
			"updatedAt":       now,
		},
	}
}

func (s *iotMIStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "GetCustomEndpoint":
		return iotMICloneMap(s.customEndpoint)
	case "RegisterCustomEndpoint":
		s.customEndpoint["status"] = "ENABLED"
		s.customEndpoint["updatedAt"] = now
		return iotMICloneMap(s.customEndpoint)

	case "GetHubConfiguration":
		return iotMICloneMap(s.hubConfiguration)
	case "PutHubConfiguration":
		if level := iotMIGetString(payload, "hubTokenTimerExpirySettingInSeconds"); level != "" {
			s.hubConfiguration["hubTokenTimerExpirySettingInSeconds"] = level
		}
		s.hubConfiguration["updatedAt"] = now
		return iotMICloneMap(s.hubConfiguration)

	case "GetDefaultEncryptionConfiguration":
		return iotMICloneMap(s.defaultEncryptionConfig)
	case "PutDefaultEncryptionConfiguration":
		if arn := iotMIGetString(payload, "kmsKeyArn", "KmsKeyArn"); arn != "" {
			s.defaultEncryptionConfig["kmsKeyArn"] = arn
		}
		s.defaultEncryptionConfig["updatedAt"] = now
		return iotMICloneMap(s.defaultEncryptionConfig)

	case "PutRuntimeLogConfiguration":
		thingID := iotMIFirstString(pathParams, payload, query, []string{"ManagedThingId", "managedThingId"}, "thing-000001")
		config := map[string]any{
			"managedThingId": thingID,
			"logLevel":       iotMIFirstString(pathParams, payload, query, []string{"logLevel", "LogLevel"}, "INFO"),
			"updatedAt":      now,
		}
		s.runtimeLogConfigurations[thingID] = config
		return iotMICloneMap(config)
	case "GetRuntimeLogConfiguration":
		thingID := iotMIFirstString(pathParams, payload, query, []string{"ManagedThingId", "managedThingId"}, "thing-000001")
		return iotMICloneMap(s.ensureRuntimeLogConfigLocked(thingID, now))
	case "ResetRuntimeLogConfiguration":
		thingID := iotMIFirstString(pathParams, payload, query, []string{"ManagedThingId", "managedThingId"}, "thing-000001")
		delete(s.runtimeLogConfigurations, thingID)
		return map[string]any{}

	case "TagResource":
		resourceARN := iotMIFirstString(pathParams, payload, query, []string{"ResourceArn", "resourceArn"}, iotMIManagedThingARN("thing-000001"))
		tags := s.ensureTagsLocked(resourceARN)
		for k, v := range iotMIExtractTagMap(payload) {
			tags[k] = v
		}
		return map[string]any{}
	case "UntagResource":
		resourceARN := iotMIFirstString(pathParams, payload, query, []string{"ResourceArn", "resourceArn"}, iotMIManagedThingARN("thing-000001"))
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range iotMIExtractTagKeys(payload, query) {
			delete(tags, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		resourceARN := iotMIFirstString(pathParams, payload, query, []string{"ResourceArn", "resourceArn"}, iotMIManagedThingARN("thing-000001"))
		out := map[string]any{}
		for k, v := range s.ensureTagsLocked(resourceARN) {
			out[k] = v
		}
		return map[string]any{"tags": out}

	case "RegisterAccountAssociation":
		associationID := iotMIFirstString(pathParams, payload, query, []string{"AccountAssociationId", "accountAssociationId"}, "assoc-000001")
		thingID := iotMIFirstString(pathParams, payload, query, []string{"ManagedThingId", "managedThingId"}, "thing-000001")
		s.managedThingAssociations = append(s.managedThingAssociations, map[string]any{
			"accountAssociationId": associationID,
			"managedThingId":       thingID,
			"status":               "REGISTERED",
		})
		return map[string]any{"accountAssociationId": associationID, "managedThingId": thingID, "status": "REGISTERED"}
	case "DeregisterAccountAssociation":
		associationID := iotMIFirstString(pathParams, payload, query, []string{"AccountAssociationId", "accountAssociationId"}, "assoc-000001")
		thingID := iotMIFirstString(pathParams, payload, query, []string{"ManagedThingId", "managedThingId"}, "thing-000001")
		out := make([]map[string]any, 0, len(s.managedThingAssociations))
		for _, item := range s.managedThingAssociations {
			if iotMIGetString(item, "accountAssociationId") == associationID && iotMIGetString(item, "managedThingId") == thingID {
				continue
			}
			out = append(out, item)
		}
		s.managedThingAssociations = out
		return map[string]any{"accountAssociationId": associationID, "managedThingId": thingID, "status": "DEREGISTERED"}

	case "SendConnectorEvent":
		return map[string]any{
			"connectorId": iotMIFirstString(pathParams, payload, query, []string{"ConnectorId", "connectorId"}, "connector-000001"),
			"status":      "ACCEPTED",
			"timestamp":   now,
		}
	case "SendManagedThingCommand":
		return map[string]any{
			"managedThingId": iotMIFirstString(pathParams, payload, query, []string{"ManagedThingId", "managedThingId"}, "thing-000001"),
			"commandId":      fmt.Sprintf("command-%06d", s.nextIDLocked()),
			"status":         "QUEUED",
			"timestamp":      now,
		}
	case "StartAccountAssociationRefresh":
		return map[string]any{
			"accountAssociationId": iotMIFirstString(pathParams, payload, query, []string{"AccountAssociationId", "accountAssociationId"}, "assoc-000001"),
			"status":               "IN_PROGRESS",
			"updatedAt":            now,
		}
	case "StartDeviceDiscovery":
		id := fmt.Sprintf("discovery-%06d", s.nextIDLocked())
		s.deviceDiscoveries[id] = map[string]any{
			"identifier": id,
			"type":       iotMIFirstString(pathParams, payload, query, []string{"type", "Type"}, "MATTER"),
			"status":     "IN_PROGRESS",
			"createdAt":  now,
			"updatedAt":  now,
		}
		return map[string]any{"identifier": id, "status": "IN_PROGRESS"}

	case "GetManagedThingCapabilities":
		thingID := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "ManagedThingId", "managedThingId"}, "thing-000001")
		return map[string]any{
			"managedThingId": thingID,
			"capabilities": []any{
				map[string]any{
					"capabilityId": "switch",
					"actions":      []any{"TurnOn", "TurnOff"},
				},
			},
		}
	case "GetManagedThingCertificate":
		thingID := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "ManagedThingId", "managedThingId"}, "thing-000001")
		return map[string]any{
			"managedThingId": thingID,
			"certificatePem": "-----BEGIN CERTIFICATE-----\\nSTACKYARD\\n-----END CERTIFICATE-----",
		}
	case "GetManagedThingConnectivityData":
		thingID := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "ManagedThingId", "managedThingId"}, "thing-000001")
		return map[string]any{
			"managedThingId": thingID,
			"connectivity": map[string]any{
				"connected":      true,
				"lastSeenAt":     now,
				"protocol":       "MATTER",
				"signalStrength": -48,
			},
		}
	case "GetManagedThingMetaData":
		thingID := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "ManagedThingId", "managedThingId"}, "thing-000001")
		thing := s.ensureManagedThingLocked(thingID, now)
		return map[string]any{
			"managedThingId": thingID,
			"metadata": map[string]any{
				"name":         thing["name"],
				"serialNumber": thing["serialNumber"],
				"owner":        "STACKYARD",
			},
		}
	case "GetManagedThingState":
		thingID := iotMIFirstString(pathParams, payload, query, []string{"ManagedThingId", "managedThingId"}, "thing-000001")
		return map[string]any{
			"managedThingId": thingID,
			"state": map[string]any{
				"power":     "ON",
				"updatedAt": now,
			},
		}
	case "GetSchemaVersion":
		return map[string]any{
			"type":              iotMIFirstString(pathParams, payload, query, []string{"Type", "type"}, "capability"),
			"schemaVersionedId": iotMIFirstString(pathParams, payload, query, []string{"SchemaVersionedId", "schemaVersionedId"}, "schema-000001"),
			"format":            iotMIFirstString(pathParams, payload, query, []string{"Format", "format"}, "JSON"),
			"schema":            "{}",
		}
	}

	switch action {
	case "CreateAccountAssociation":
		id := iotMIFirstString(pathParams, payload, query, []string{"AccountAssociationId", "accountAssociationId"}, fmt.Sprintf("assoc-%06d", s.nextIDLocked()))
		item := s.ensureAccountAssociationLocked(id, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "UpdateAccountAssociation":
		id := iotMIFirstString(pathParams, payload, query, []string{"AccountAssociationId", "accountAssociationId"}, "assoc-000001")
		item := s.ensureAccountAssociationLocked(id, now)
		item["status"] = "ACTIVE"
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "GetAccountAssociation":
		id := iotMIFirstString(pathParams, payload, query, []string{"AccountAssociationId", "accountAssociationId"}, "assoc-000001")
		return iotMICloneMap(s.ensureAccountAssociationLocked(id, now))
	case "DeleteAccountAssociation":
		id := iotMIFirstString(pathParams, payload, query, []string{"AccountAssociationId", "accountAssociationId"}, "assoc-000001")
		delete(s.accountAssociations, id)
		return map[string]any{}
	case "ListAccountAssociations":
		return map[string]any{"accountAssociations": iotMIListClonedMaps(s.accountAssociations), "nextToken": ""}

	case "CreateCloudConnector":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, fmt.Sprintf("connector-%06d", s.nextIDLocked()))
		item := s.ensureCloudConnectorLocked(id, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "UpdateCloudConnector":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "connector-000001")
		item := s.ensureCloudConnectorLocked(id, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "GetCloudConnector":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "connector-000001")
		return iotMICloneMap(s.ensureCloudConnectorLocked(id, now))
	case "DeleteCloudConnector":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "connector-000001")
		delete(s.cloudConnectors, id)
		return map[string]any{}
	case "ListCloudConnectors":
		return map[string]any{"cloudConnectors": iotMIListClonedMaps(s.cloudConnectors), "nextToken": ""}

	case "CreateConnectorDestination":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, fmt.Sprintf("connector-destination-%06d", s.nextIDLocked()))
		item := s.ensureConnectorDestinationLocked(id, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "UpdateConnectorDestination":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "connector-destination-000001")
		item := s.ensureConnectorDestinationLocked(id, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "GetConnectorDestination":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "connector-destination-000001")
		return iotMICloneMap(s.ensureConnectorDestinationLocked(id, now))
	case "DeleteConnectorDestination":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "connector-destination-000001")
		delete(s.connectorDestinations, id)
		return map[string]any{}
	case "ListConnectorDestinations":
		return map[string]any{"connectorDestinations": iotMIListClonedMaps(s.connectorDestinations), "nextToken": ""}

	case "CreateCredentialLocker":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, fmt.Sprintf("locker-%06d", s.nextIDLocked()))
		item := s.ensureCredentialLockerLocked(id, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "GetCredentialLocker":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "locker-000001")
		return iotMICloneMap(s.ensureCredentialLockerLocked(id, now))
	case "DeleteCredentialLocker":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "locker-000001")
		delete(s.credentialLockers, id)
		return map[string]any{}
	case "ListCredentialLockers":
		return map[string]any{"credentialLockers": iotMIListClonedMaps(s.credentialLockers), "nextToken": ""}

	case "CreateDestination":
		name := iotMIFirstString(pathParams, payload, query, []string{"Name", "name"}, "stackyard-destination")
		item := s.ensureDestinationLocked(name, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "UpdateDestination":
		name := iotMIFirstString(pathParams, payload, query, []string{"Name", "name"}, "stackyard-destination")
		item := s.ensureDestinationLocked(name, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "GetDestination":
		name := iotMIFirstString(pathParams, payload, query, []string{"Name", "name"}, "stackyard-destination")
		return iotMICloneMap(s.ensureDestinationLocked(name, now))
	case "DeleteDestination":
		name := iotMIFirstString(pathParams, payload, query, []string{"Name", "name"}, "stackyard-destination")
		delete(s.destinations, name)
		return map[string]any{}
	case "ListDestinations":
		return map[string]any{"destinations": iotMIListClonedMaps(s.destinations), "nextToken": ""}

	case "CreateEventLogConfiguration":
		id := iotMIFirstString(pathParams, payload, query, []string{"Id", "id"}, fmt.Sprintf("event-log-%06d", s.nextIDLocked()))
		item := s.ensureEventLogConfigurationLocked(id, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "UpdateEventLogConfiguration":
		id := iotMIFirstString(pathParams, payload, query, []string{"Id", "id"}, "event-log-000001")
		item := s.ensureEventLogConfigurationLocked(id, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "GetEventLogConfiguration":
		id := iotMIFirstString(pathParams, payload, query, []string{"Id", "id"}, "event-log-000001")
		return iotMICloneMap(s.ensureEventLogConfigurationLocked(id, now))
	case "DeleteEventLogConfiguration":
		id := iotMIFirstString(pathParams, payload, query, []string{"Id", "id"}, "event-log-000001")
		delete(s.eventLogConfigurations, id)
		return map[string]any{}
	case "ListEventLogConfigurations":
		return map[string]any{"eventLogConfigurations": iotMIListClonedMaps(s.eventLogConfigurations), "nextToken": ""}

	case "CreateManagedThing":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "managedThingId", "ManagedThingId"}, fmt.Sprintf("thing-%06d", s.nextIDLocked()))
		item := s.ensureManagedThingLocked(id, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "UpdateManagedThing":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "managedThingId", "ManagedThingId"}, "thing-000001")
		item := s.ensureManagedThingLocked(id, now)
		if name := iotMIGetString(payload, "name", "Name"); name != "" {
			item["name"] = name
		}
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "GetManagedThing":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "managedThingId", "ManagedThingId"}, "thing-000001")
		return iotMICloneMap(s.ensureManagedThingLocked(id, now))
	case "DeleteManagedThing":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "managedThingId", "ManagedThingId"}, "thing-000001")
		delete(s.managedThings, id)
		return map[string]any{}
	case "ListManagedThings":
		return map[string]any{"managedThings": iotMIListClonedMaps(s.managedThings), "nextToken": ""}

	case "CreateNotificationConfiguration":
		eventType := iotMIFirstString(pathParams, payload, query, []string{"EventType", "eventType"}, "DEVICE_CONNECTED")
		item := s.ensureNotificationConfigurationLocked(eventType, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "UpdateNotificationConfiguration":
		eventType := iotMIFirstString(pathParams, payload, query, []string{"EventType", "eventType"}, "DEVICE_CONNECTED")
		item := s.ensureNotificationConfigurationLocked(eventType, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "GetNotificationConfiguration":
		eventType := iotMIFirstString(pathParams, payload, query, []string{"EventType", "eventType"}, "DEVICE_CONNECTED")
		return iotMICloneMap(s.ensureNotificationConfigurationLocked(eventType, now))
	case "DeleteNotificationConfiguration":
		eventType := iotMIFirstString(pathParams, payload, query, []string{"EventType", "eventType"}, "DEVICE_CONNECTED")
		delete(s.notificationConfigurations, eventType)
		return map[string]any{}
	case "ListNotificationConfigurations":
		return map[string]any{"notificationConfigurations": iotMIListClonedMaps(s.notificationConfigurations), "nextToken": ""}

	case "CreateOtaTaskConfiguration":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, fmt.Sprintf("ota-config-%06d", s.nextIDLocked()))
		item := s.ensureOTATaskConfigurationLocked(id, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "GetOtaTaskConfiguration":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "ota-config-000001")
		return iotMICloneMap(s.ensureOTATaskConfigurationLocked(id, now))
	case "DeleteOtaTaskConfiguration":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "ota-config-000001")
		delete(s.otaTaskConfigurations, id)
		return map[string]any{}
	case "ListOtaTaskConfigurations":
		return map[string]any{"otaTaskConfigurations": iotMIListClonedMaps(s.otaTaskConfigurations), "nextToken": ""}

	case "CreateOtaTask":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, fmt.Sprintf("ota-task-%06d", s.nextIDLocked()))
		item := s.ensureOTATaskLocked(id, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "UpdateOtaTask":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "ota-task-000001")
		item := s.ensureOTATaskLocked(id, now)
		item["updatedAt"] = now
		item["status"] = "IN_PROGRESS"
		return iotMICloneMap(item)
	case "GetOtaTask":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "ota-task-000001")
		return iotMICloneMap(s.ensureOTATaskLocked(id, now))
	case "DeleteOtaTask":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "ota-task-000001")
		delete(s.otaTasks, id)
		return map[string]any{}
	case "ListOtaTasks":
		return map[string]any{"otaTasks": iotMIListClonedMaps(s.otaTasks), "nextToken": ""}
	case "ListOtaTaskExecutions":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "ota-task-000001")
		return map[string]any{
			"otaTaskExecutions": []any{
				map[string]any{
					"identifier":     id,
					"managedThingId": "thing-000001",
					"status":         "SUCCEEDED",
					"updatedAt":      now,
				},
			},
			"nextToken": "",
		}

	case "CreateProvisioningProfile":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, fmt.Sprintf("profile-%06d", s.nextIDLocked()))
		item := s.ensureProvisioningProfileLocked(id, now)
		item["updatedAt"] = now
		return iotMICloneMap(item)
	case "GetProvisioningProfile":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "profile-000001")
		return iotMICloneMap(s.ensureProvisioningProfileLocked(id, now))
	case "DeleteProvisioningProfile":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "profile-000001")
		delete(s.provisioningProfiles, id)
		return map[string]any{}
	case "ListProvisioningProfiles":
		return map[string]any{"provisioningProfiles": iotMIListClonedMaps(s.provisioningProfiles), "nextToken": ""}

	case "GetDeviceDiscovery":
		id := iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "discovery-000001")
		return iotMICloneMap(s.ensureDeviceDiscoveryLocked(id, now))
	case "ListDeviceDiscoveries":
		return map[string]any{"deviceDiscoveries": iotMIListClonedMaps(s.deviceDiscoveries), "nextToken": ""}
	case "ListDiscoveredDevices":
		return map[string]any{
			"devices": []any{
				map[string]any{
					"managedThingId": "thing-000001",
					"serialNumber":   "SN-000001",
					"type":           "MATTER",
				},
			},
			"nextToken": "",
		}

	case "ListManagedThingAccountAssociations":
		items := make([]any, 0, len(s.managedThingAssociations))
		for _, item := range s.managedThingAssociations {
			items = append(items, iotMICloneMap(item))
		}
		return map[string]any{"managedThingAccountAssociations": items, "nextToken": ""}
	case "ListManagedThingSchemas":
		return map[string]any{
			"schemas": []any{
				map[string]any{
					"identifier":      iotMIFirstString(pathParams, payload, query, []string{"Identifier", "identifier"}, "thing-000001"),
					"schemaId":        "switch",
					"semanticVersion": "1.0.0",
					"visibility":      "PUBLIC",
				},
			},
			"nextToken": "",
		}
	case "ListSchemaVersions":
		return map[string]any{
			"schemaVersions": []any{
				map[string]any{
					"type":            iotMIFirstString(pathParams, payload, query, []string{"Type", "type"}, "capability"),
					"schemaId":        "switch",
					"semanticVersion": "1.0.0",
					"visibility":      "PUBLIC",
				},
			},
			"nextToken": "",
		}
	}

	return map[string]any{
		"action":    action,
		"status":    "SUCCESS",
		"timestamp": now,
	}
}

func (s *iotMIStore) nextIDLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *iotMIStore) ensureAccountAssociationLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "assoc-000001"
	}
	if item := s.accountAssociations[id]; item != nil {
		return item
	}
	item := map[string]any{
		"accountAssociationId":   id,
		"connectorDestinationId": "connector-destination-000001",
		"status":                 "ACTIVE",
		"createdAt":              now,
		"updatedAt":              now,
	}
	s.accountAssociations[id] = item
	return item
}

func (s *iotMIStore) ensureCloudConnectorLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "connector-000001"
	}
	if item := s.cloudConnectors[id]; item != nil {
		return item
	}
	item := map[string]any{
		"identifier": id,
		"type":       "LAMBDA",
		"lambdaArn":  "arn:aws:lambda:us-east-1:123456789012:function:stackyard-iotmi",
		"status":     "ACTIVE",
		"createdAt":  now,
		"updatedAt":  now,
	}
	s.cloudConnectors[id] = item
	return item
}

func (s *iotMIStore) ensureConnectorDestinationLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "connector-destination-000001"
	}
	if item := s.connectorDestinations[id]; item != nil {
		return item
	}
	item := map[string]any{
		"identifier":       id,
		"cloudConnectorId": "connector-000001",
		"status":           "ACTIVE",
		"createdAt":        now,
		"updatedAt":        now,
	}
	s.connectorDestinations[id] = item
	return item
}

func (s *iotMIStore) ensureCredentialLockerLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "locker-000001"
	}
	if item := s.credentialLockers[id]; item != nil {
		return item
	}
	item := map[string]any{
		"identifier": id,
		"type":       "SECRETS_MANAGER",
		"status":     "ACTIVE",
		"createdAt":  now,
		"updatedAt":  now,
	}
	s.credentialLockers[id] = item
	return item
}

func (s *iotMIStore) ensureDestinationLocked(name, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-destination"
	}
	if item := s.destinations[name]; item != nil {
		return item
	}
	item := map[string]any{
		"name":      name,
		"status":    "ACTIVE",
		"createdAt": now,
		"updatedAt": now,
	}
	s.destinations[name] = item
	return item
}

func (s *iotMIStore) ensureEventLogConfigurationLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "event-log-000001"
	}
	if item := s.eventLogConfigurations[id]; item != nil {
		return item
	}
	item := map[string]any{
		"id":        id,
		"logLevel":  "INFO",
		"status":    "ENABLED",
		"createdAt": now,
		"updatedAt": now,
	}
	s.eventLogConfigurations[id] = item
	return item
}

func (s *iotMIStore) ensureManagedThingLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "thing-000001"
	}
	if item := s.managedThings[id]; item != nil {
		return item
	}
	item := map[string]any{
		"identifier":             id,
		"managedThingId":         id,
		"managedThingArn":        iotMIManagedThingARN(id),
		"name":                   "stackyard-managed-thing",
		"serialNumber":           "SN-000001",
		"provisioningStatus":     "COMPLETED",
		"connectorDestinationId": "connector-destination-000001",
		"createdAt":              now,
		"updatedAt":              now,
	}
	s.managedThings[id] = item
	return item
}

func (s *iotMIStore) ensureNotificationConfigurationLocked(eventType, now string) map[string]any {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		eventType = "DEVICE_CONNECTED"
	}
	if item := s.notificationConfigurations[eventType]; item != nil {
		return item
	}
	item := map[string]any{
		"eventType": eventType,
		"status":    "ENABLED",
		"createdAt": now,
		"updatedAt": now,
	}
	s.notificationConfigurations[eventType] = item
	return item
}

func (s *iotMIStore) ensureOTATaskConfigurationLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "ota-config-000001"
	}
	if item := s.otaTaskConfigurations[id]; item != nil {
		return item
	}
	item := map[string]any{
		"identifier": id,
		"name":       "stackyard-ota-config",
		"status":     "ACTIVE",
		"createdAt":  now,
		"updatedAt":  now,
	}
	s.otaTaskConfigurations[id] = item
	return item
}

func (s *iotMIStore) ensureOTATaskLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "ota-task-000001"
	}
	if item := s.otaTasks[id]; item != nil {
		return item
	}
	item := map[string]any{
		"identifier":             id,
		"name":                   "stackyard-ota-task",
		"status":                 "IN_PROGRESS",
		"otaTaskConfigurationId": "ota-config-000001",
		"createdAt":              now,
		"updatedAt":              now,
	}
	s.otaTasks[id] = item
	return item
}

func (s *iotMIStore) ensureProvisioningProfileLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "profile-000001"
	}
	if item := s.provisioningProfiles[id]; item != nil {
		return item
	}
	item := map[string]any{
		"identifier": id,
		"name":       "stackyard-profile",
		"status":     "ACTIVE",
		"createdAt":  now,
		"updatedAt":  now,
	}
	s.provisioningProfiles[id] = item
	return item
}

func (s *iotMIStore) ensureDeviceDiscoveryLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "discovery-000001"
	}
	if item := s.deviceDiscoveries[id]; item != nil {
		return item
	}
	item := map[string]any{
		"identifier": id,
		"type":       "MATTER",
		"status":     "SUCCEEDED",
		"createdAt":  now,
		"updatedAt":  now,
	}
	s.deviceDiscoveries[id] = item
	return item
}

func (s *iotMIStore) ensureRuntimeLogConfigLocked(thingID, now string) map[string]any {
	thingID = strings.TrimSpace(thingID)
	if thingID == "" {
		thingID = "thing-000001"
	}
	if item := s.runtimeLogConfigurations[thingID]; item != nil {
		return item
	}
	item := map[string]any{
		"managedThingId": thingID,
		"logLevel":       "INFO",
		"updatedAt":      now,
	}
	s.runtimeLogConfigurations[thingID] = item
	return item
}

func (s *iotMIStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = iotMIManagedThingARN("thing-000001")
	}
	if tags := s.tags[resourceARN]; tags != nil {
		return tags
	}
	tags := map[string]string{}
	s.tags[resourceARN] = tags
	return tags
}

func iotMIManagedThingARN(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "thing-000001"
	}
	return "arn:aws:iotmanagedintegrations:us-east-1:123456789012:managed-thing/" + id
}

func iotMIFirstString(pathParams map[string]string, payload map[string]any, query url.Values, keys []string, fallback string) string {
	for _, key := range keys {
		if pathParams != nil {
			if v := strings.TrimSpace(pathParams[key]); v != "" {
				return v
			}
			for k, v := range pathParams {
				if strings.EqualFold(k, key) {
					v = strings.TrimSpace(v)
					if v != "" {
						return v
					}
				}
			}
		}
		if payload != nil {
			if v := strings.TrimSpace(iotMIGetString(payload, key)); v != "" {
				return v
			}
		}
		if query != nil {
			if v := strings.TrimSpace(query.Get(key)); v != "" {
				return v
			}
			for k, values := range query {
				if strings.EqualFold(k, key) && len(values) > 0 {
					v := strings.TrimSpace(values[0])
					if v != "" {
						return v
					}
				}
			}
		}
	}
	return fallback
}

func iotMIGetString(payload map[string]any, keys ...string) string {
	if payload == nil {
		return ""
	}
	for _, key := range keys {
		if raw, ok := payload[key]; ok {
			if value := strings.TrimSpace(iotMIToString(raw)); value != "" {
				return value
			}
		}
		for k, raw := range payload {
			if strings.EqualFold(k, key) {
				if value := strings.TrimSpace(iotMIToString(raw)); value != "" {
					return value
				}
			}
		}
	}
	return ""
}

func iotMIToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func iotMIExtractTagMap(payload map[string]any) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	raw, ok := payload["tags"]
	if !ok {
		for k, v := range payload {
			if strings.EqualFold(k, "tags") {
				raw = v
				ok = true
				break
			}
		}
	}
	if !ok {
		return out
	}
	switch tags := raw.(type) {
	case map[string]any:
		for key, val := range tags {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(iotMIToString(val))
		}
	case map[string]string:
		for key, val := range tags {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(val)
		}
	case []any:
		for _, entry := range tags {
			tag, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			key := strings.TrimSpace(iotMIToString(tag["key"]))
			if key == "" {
				key = strings.TrimSpace(iotMIToString(tag["Key"]))
			}
			if key == "" {
				continue
			}
			value := strings.TrimSpace(iotMIToString(tag["value"]))
			if value == "" {
				value = strings.TrimSpace(iotMIToString(tag["Value"]))
			}
			out[key] = value
		}
	}
	return out
}

func iotMIExtractTagKeys(payload map[string]any, query url.Values) []string {
	set := map[string]struct{}{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v != "" {
			set[v] = struct{}{}
		}
	}

	if payload != nil {
		for _, keyName := range []string{"tagKeys", "TagKeys"} {
			raw, ok := payload[keyName]
			if !ok {
				continue
			}
			switch v := raw.(type) {
			case []any:
				for _, item := range v {
					add(iotMIToString(item))
				}
			case []string:
				for _, item := range v {
					add(item)
				}
			case string:
				for _, part := range strings.Split(v, ",") {
					add(part)
				}
			}
		}
	}

	for _, keyName := range []string{"tagKeys", "TagKeys"} {
		for _, value := range query[keyName] {
			for _, part := range strings.Split(value, ",") {
				add(part)
			}
		}
	}

	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func iotMICloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func iotMIListClonedMaps(in map[string]map[string]any) []any {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, iotMICloneMap(in[key]))
	}
	return out
}
