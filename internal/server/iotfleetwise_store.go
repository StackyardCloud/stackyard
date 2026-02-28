package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	iotFleetWiseDefaultRegion    = "us-east-1"
	iotFleetWiseDefaultAccountID = "123456789012"
)

type iotFleetWiseStore struct {
	mu sync.Mutex

	nextID int64

	registered bool

	loggingOptions          map[string]any
	encryptionConfiguration map[string]any

	campaigns        map[string]map[string]any
	decoderManifests map[string]map[string]any
	fleets           map[string]map[string]any
	modelManifests   map[string]map[string]any
	signalCatalogs   map[string]map[string]any
	stateTemplates   map[string]map[string]any
	vehicles         map[string]map[string]any

	fleetVehicles map[string]map[string]struct{}
	tags          map[string]map[string]string

	createTokens map[string]string
}

func newIoTFleetWiseStore() *iotFleetWiseStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &iotFleetWiseStore{
		nextID: 1,

		loggingOptions: map[string]any{
			"logType": "ERROR",
		},
		encryptionConfiguration: map[string]any{
			"encryptionStatus": "ENABLED",
			"kmsKeyId":         "alias/aws/iotfleetwise",
		},

		campaigns:        map[string]map[string]any{},
		decoderManifests: map[string]map[string]any{},
		fleets:           map[string]map[string]any{},
		modelManifests:   map[string]map[string]any{},
		signalCatalogs:   map[string]map[string]any{},
		stateTemplates:   map[string]map[string]any{},
		vehicles:         map[string]map[string]any{},

		fleetVehicles: map[string]map[string]struct{}{},
		tags:          map[string]map[string]string{},

		createTokens: map[string]string{},
	}

	signalCatalog := s.ensureSignalCatalogLocked("stackyard-signal-catalog", now)
	modelManifest := s.ensureModelManifestLocked("stackyard-model-manifest", iotFleetWisePayloadString(signalCatalog, "arn"), now)
	decoderManifest := s.ensureDecoderManifestLocked("stackyard-decoder-manifest", iotFleetWisePayloadString(modelManifest, "arn"), now)
	fleet := s.ensureFleetLocked("stackyard-fleet", iotFleetWisePayloadString(signalCatalog, "arn"), now)
	vehicle := s.ensureVehicleLocked(
		"stackyard-vehicle",
		iotFleetWisePayloadString(fleet, "arn"),
		iotFleetWisePayloadString(modelManifest, "arn"),
		iotFleetWisePayloadString(decoderManifest, "arn"),
		now,
	)
	campaign := s.ensureCampaignLocked(
		"stackyard-campaign",
		iotFleetWisePayloadString(fleet, "arn"),
		iotFleetWisePayloadString(signalCatalog, "arn"),
		now,
	)
	stateTemplate := s.ensureStateTemplateLocked("stackyard-state-template", iotFleetWisePayloadString(signalCatalog, "arn"), now)

	s.associateVehicleFleetLocked(iotFleetWisePayloadString(vehicle, "vehicleName"), iotFleetWisePayloadString(fleet, "id"))
	s.ensureTagsLocked(iotFleetWisePayloadString(campaign, "arn"))["stackyard"] = "true"
	s.ensureTagsLocked(iotFleetWisePayloadString(decoderManifest, "arn"))["stackyard"] = "true"
	s.ensureTagsLocked(iotFleetWisePayloadString(fleet, "arn"))["stackyard"] = "true"
	s.ensureTagsLocked(iotFleetWisePayloadString(modelManifest, "arn"))["stackyard"] = "true"
	s.ensureTagsLocked(iotFleetWisePayloadString(signalCatalog, "arn"))["stackyard"] = "true"
	s.ensureTagsLocked(iotFleetWisePayloadString(stateTemplate, "arn"))["stackyard"] = "true"
	s.ensureTagsLocked(iotFleetWisePayloadString(vehicle, "arn"))["stackyard"] = "true"

	return s
}

func (s *iotFleetWiseStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "RegisterAccount":
		s.registered = true
		if cfg := iotFleetWisePayloadMap(payload, "loggingOptions", "cloudWatchLogDeliveryOptions"); len(cfg) > 0 {
			s.loggingOptions = iotFleetWiseCloneMap(cfg)
		}
		if cfg := iotFleetWisePayloadMap(payload, "encryptionConfiguration"); len(cfg) > 0 {
			s.encryptionConfiguration = iotFleetWiseCloneMap(cfg)
		}
		return s.accountStatusLocked()
	case "GetRegisterAccountStatus":
		return s.accountStatusLocked()
	case "PutLoggingOptions":
		if cfg := iotFleetWisePayloadMap(payload, "cloudWatchLogDeliveryOptions", "loggingOptions"); len(cfg) > 0 {
			s.loggingOptions = iotFleetWiseCloneMap(cfg)
		}
		return map[string]any{}
	case "GetLoggingOptions":
		return map[string]any{
			"cloudWatchLogDeliveryOptions": iotFleetWiseCloneMap(s.loggingOptions),
		}
	case "PutEncryptionConfiguration":
		if cfg := iotFleetWisePayloadMap(payload, "encryptionConfiguration"); len(cfg) > 0 {
			s.encryptionConfiguration = iotFleetWiseCloneMap(cfg)
		} else {
			status := iotFleetWiseFirstNonEmpty(
				iotFleetWisePayloadString(payload, "encryptionStatus", "status"),
				iotFleetWisePayloadString(s.encryptionConfiguration, "encryptionStatus"),
			)
			kmsKeyID := iotFleetWiseFirstNonEmpty(
				iotFleetWisePayloadString(payload, "kmsKeyId", "kmsKeyID"),
				iotFleetWisePayloadString(s.encryptionConfiguration, "kmsKeyId"),
			)
			s.encryptionConfiguration = map[string]any{
				"encryptionStatus": iotFleetWiseFirstNonEmpty(status, "ENABLED"),
				"kmsKeyId":         iotFleetWiseFirstNonEmpty(kmsKeyID, "alias/aws/iotfleetwise"),
			}
		}
		return map[string]any{}
	case "GetEncryptionConfiguration":
		return iotFleetWiseCloneMap(s.encryptionConfiguration)

	case "CreateSignalCatalog", "ImportSignalCatalog":
		resource := s.createSignalCatalogLocked(action, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "GetSignalCatalog":
		resource := s.resolveSignalCatalogLocked(s.signalCatalogIdentifier(payload), now)
		return iotFleetWiseCloneMap(resource)
	case "ListSignalCatalogs":
		return map[string]any{"summaries": s.listSignalCatalogSummariesLocked(), "nextToken": ""}
	case "UpdateSignalCatalog":
		resource := s.resolveSignalCatalogLocked(s.signalCatalogIdentifier(payload), now)
		s.updateCommonResourceLocked(resource, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "DeleteSignalCatalog":
		resource := s.resolveSignalCatalogLocked(s.signalCatalogIdentifier(payload), now)
		resource["status"] = "DELETING"
		resource["lastModificationTime"] = now
		resource["status"] = "DELETED"
		return map[string]any{"name": iotFleetWisePayloadString(resource, "name"), "arn": iotFleetWisePayloadString(resource, "arn")}
	case "ListSignalCatalogNodes":
		resource := s.resolveSignalCatalogLocked(s.signalCatalogIdentifier(payload), now)
		nodes := iotFleetWisePayloadList(resource, "nodes")
		if len(nodes) == 0 {
			nodes = []any{
				map[string]any{"fullyQualifiedName": "Vehicle.Speed", "nodeType": "SENSOR", "dataType": "DOUBLE"},
				map[string]any{"fullyQualifiedName": "Vehicle.Engine", "nodeType": "BRANCH"},
			}
			resource["nodes"] = nodes
		}
		return map[string]any{"nodes": iotFleetWiseCloneList(nodes), "nextToken": ""}

	case "CreateModelManifest":
		resource := s.createModelManifestLocked(action, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "GetModelManifest":
		resource := s.resolveModelManifestLocked(s.modelManifestIdentifier(payload), now)
		return iotFleetWiseCloneMap(resource)
	case "ListModelManifests":
		return map[string]any{"summaries": s.listModelManifestSummariesLocked(), "nextToken": ""}
	case "UpdateModelManifest":
		resource := s.resolveModelManifestLocked(s.modelManifestIdentifier(payload), now)
		s.updateCommonResourceLocked(resource, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "DeleteModelManifest":
		resource := s.resolveModelManifestLocked(s.modelManifestIdentifier(payload), now)
		resource["status"] = "DELETING"
		resource["lastModificationTime"] = now
		resource["status"] = "DELETED"
		return map[string]any{"name": iotFleetWisePayloadString(resource, "name"), "arn": iotFleetWisePayloadString(resource, "arn")}
	case "ListModelManifestNodes":
		resource := s.resolveModelManifestLocked(s.modelManifestIdentifier(payload), now)
		nodes := iotFleetWisePayloadList(resource, "nodes")
		if len(nodes) == 0 {
			nodes = []any{
				map[string]any{"fullyQualifiedName": "Vehicle.Speed", "nodeType": "SENSOR"},
				map[string]any{"fullyQualifiedName": "Vehicle.Engine.RPM", "nodeType": "SENSOR"},
			}
			resource["nodes"] = nodes
		}
		return map[string]any{"nodes": iotFleetWiseCloneList(nodes), "nextToken": ""}

	case "CreateDecoderManifest", "ImportDecoderManifest":
		resource := s.createDecoderManifestLocked(action, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "GetDecoderManifest":
		resource := s.resolveDecoderManifestLocked(s.decoderManifestIdentifier(payload), now)
		return iotFleetWiseCloneMap(resource)
	case "ListDecoderManifests":
		return map[string]any{"summaries": s.listDecoderManifestSummariesLocked(), "nextToken": ""}
	case "UpdateDecoderManifest":
		resource := s.resolveDecoderManifestLocked(s.decoderManifestIdentifier(payload), now)
		s.updateCommonResourceLocked(resource, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "DeleteDecoderManifest":
		resource := s.resolveDecoderManifestLocked(s.decoderManifestIdentifier(payload), now)
		resource["status"] = "DELETING"
		resource["lastModificationTime"] = now
		resource["status"] = "DELETED"
		return map[string]any{"name": iotFleetWisePayloadString(resource, "name"), "arn": iotFleetWisePayloadString(resource, "arn")}
	case "ListDecoderManifestNetworkInterfaces":
		resource := s.resolveDecoderManifestLocked(s.decoderManifestIdentifier(payload), now)
		interfaces := iotFleetWisePayloadList(resource, "networkInterfaces")
		if len(interfaces) == 0 {
			interfaces = []any{
				map[string]any{"interfaceId": "can0", "type": "CAN_INTERFACE"},
			}
			resource["networkInterfaces"] = interfaces
		}
		return map[string]any{"networkInterfaces": iotFleetWiseCloneList(interfaces), "nextToken": ""}
	case "ListDecoderManifestSignals":
		resource := s.resolveDecoderManifestLocked(s.decoderManifestIdentifier(payload), now)
		signals := iotFleetWisePayloadList(resource, "signalDecoders")
		if len(signals) == 0 {
			signals = []any{
				map[string]any{"fullyQualifiedName": "Vehicle.Speed", "interfaceId": "can0", "type": "CAN_SIGNAL"},
			}
			resource["signalDecoders"] = signals
		}
		return map[string]any{"signalDecoders": iotFleetWiseCloneList(signals), "nextToken": ""}

	case "CreateFleet":
		resource := s.createFleetLocked(action, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "GetFleet":
		resource := s.resolveFleetLocked(s.fleetIdentifier(payload), now)
		return iotFleetWiseCloneMap(resource)
	case "ListFleets":
		return map[string]any{"fleetSummaries": s.listFleetSummariesLocked(), "nextToken": ""}
	case "UpdateFleet":
		resource := s.resolveFleetLocked(s.fleetIdentifier(payload), now)
		s.updateCommonResourceLocked(resource, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "DeleteFleet":
		resource := s.resolveFleetLocked(s.fleetIdentifier(payload), now)
		resource["status"] = "DELETING"
		resource["lastModificationTime"] = now
		resource["status"] = "DELETED"
		return map[string]any{"name": iotFleetWisePayloadString(resource, "name"), "arn": iotFleetWisePayloadString(resource, "arn")}

	case "CreateVehicle":
		resource := s.createVehicleLocked(action, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "BatchCreateVehicle":
		return s.batchCreateVehiclesLocked(payload, now)
	case "GetVehicle":
		resource := s.resolveVehicleLocked(s.vehicleIdentifier(payload), now)
		return iotFleetWiseCloneMap(resource)
	case "ListVehicles":
		return map[string]any{"vehicleSummaries": s.listVehicleSummariesLocked(), "nextToken": ""}
	case "UpdateVehicle":
		resource := s.resolveVehicleLocked(s.vehicleIdentifier(payload), now)
		s.updateCommonResourceLocked(resource, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "BatchUpdateVehicle":
		return s.batchUpdateVehiclesLocked(payload, now)
	case "DeleteVehicle":
		resource := s.resolveVehicleLocked(s.vehicleIdentifier(payload), now)
		resource["status"] = "DELETING"
		resource["lastModificationTime"] = now
		resource["status"] = "DELETED"
		for _, assoc := range s.fleetVehicles {
			delete(assoc, iotFleetWisePayloadString(resource, "vehicleName"))
		}
		return map[string]any{"vehicleName": iotFleetWisePayloadString(resource, "vehicleName"), "arn": iotFleetWisePayloadString(resource, "arn")}
	case "GetVehicleStatus":
		vehicle := s.resolveVehicleLocked(s.vehicleIdentifier(payload), now)
		return map[string]any{
			"vehicleName": iotFleetWisePayloadString(vehicle, "vehicleName"),
			"campaigns":   s.vehicleCampaignStatusesLocked(vehicle),
		}
	case "AssociateVehicleFleet":
		vehicle := s.resolveVehicleLocked(s.vehicleIdentifier(payload), now)
		fleet := s.resolveFleetLocked(s.fleetIdentifier(payload), now)
		s.associateVehicleFleetLocked(iotFleetWisePayloadString(vehicle, "vehicleName"), iotFleetWisePayloadString(fleet, "id"))
		return map[string]any{}
	case "DisassociateVehicleFleet":
		vehicle := s.resolveVehicleLocked(s.vehicleIdentifier(payload), now)
		fleet := s.resolveFleetLocked(s.fleetIdentifier(payload), now)
		s.disassociateVehicleFleetLocked(iotFleetWisePayloadString(vehicle, "vehicleName"), iotFleetWisePayloadString(fleet, "id"))
		return map[string]any{}
	case "ListFleetsForVehicle":
		vehicle := s.resolveVehicleLocked(s.vehicleIdentifier(payload), now)
		return map[string]any{"fleets": s.listFleetsForVehicleLocked(iotFleetWisePayloadString(vehicle, "vehicleName")), "nextToken": ""}
	case "ListVehiclesInFleet":
		fleet := s.resolveFleetLocked(s.fleetIdentifier(payload), now)
		return map[string]any{"vehicles": s.listVehiclesInFleetLocked(iotFleetWisePayloadString(fleet, "id")), "nextToken": ""}

	case "CreateCampaign":
		resource := s.createCampaignLocked(action, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "GetCampaign":
		resource := s.resolveCampaignLocked(s.campaignIdentifier(payload), now)
		return iotFleetWiseCloneMap(resource)
	case "ListCampaigns":
		return map[string]any{"campaignSummaries": s.listCampaignSummariesLocked(), "nextToken": ""}
	case "UpdateCampaign":
		resource := s.resolveCampaignLocked(s.campaignIdentifier(payload), now)
		s.updateCommonResourceLocked(resource, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "DeleteCampaign":
		resource := s.resolveCampaignLocked(s.campaignIdentifier(payload), now)
		resource["status"] = "DELETING"
		resource["lastModificationTime"] = now
		resource["status"] = "DELETED"
		return map[string]any{"name": iotFleetWisePayloadString(resource, "name"), "arn": iotFleetWisePayloadString(resource, "arn")}

	case "CreateStateTemplate":
		resource := s.createStateTemplateLocked(action, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "GetStateTemplate":
		resource := s.resolveStateTemplateLocked(s.stateTemplateIdentifier(payload), now)
		return iotFleetWiseCloneMap(resource)
	case "ListStateTemplates":
		return map[string]any{"summaries": s.listStateTemplateSummariesLocked(), "nextToken": ""}
	case "UpdateStateTemplate":
		resource := s.resolveStateTemplateLocked(s.stateTemplateIdentifier(payload), now)
		s.updateCommonResourceLocked(resource, payload, now)
		return iotFleetWiseCloneMap(resource)
	case "DeleteStateTemplate":
		resource := s.resolveStateTemplateLocked(s.stateTemplateIdentifier(payload), now)
		resource["status"] = "DELETING"
		resource["lastModificationTime"] = now
		resource["status"] = "DELETED"
		return map[string]any{"name": iotFleetWisePayloadString(resource, "name"), "arn": iotFleetWisePayloadString(resource, "arn")}

	case "TagResource":
		resourceARN := iotFleetWiseFirstNonEmpty(
			iotFleetWisePayloadString(payload, "resourceARN", "resourceArn", "arn"),
			s.defaultResourceARNLocked(),
		)
		target := s.ensureTagsLocked(resourceARN)
		for key, value := range iotFleetWisePayloadStringMap(payload, "tags", "Tags") {
			target[key] = value
		}
		return map[string]any{}
	case "UntagResource":
		resourceARN := iotFleetWiseFirstNonEmpty(
			iotFleetWisePayloadString(payload, "resourceARN", "resourceArn", "arn"),
			s.defaultResourceARNLocked(),
		)
		target := s.ensureTagsLocked(resourceARN)
		for _, key := range iotFleetWisePayloadStringSlice(payload, "tagKeys", "TagKeys") {
			delete(target, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		resourceARN := iotFleetWiseFirstNonEmpty(
			iotFleetWisePayloadString(payload, "resourceARN", "resourceArn", "arn"),
			s.defaultResourceARNLocked(),
		)
		return map[string]any{"tags": iotFleetWiseTagsList(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{"action": action}
}

func (s *iotFleetWiseStore) createSignalCatalogLocked(action string, payload map[string]any, now string) map[string]any {
	name := iotFleetWiseFirstNonEmpty(
		s.idempotentNameLocked(action, payload),
		iotFleetWisePayloadString(payload, "name", "signalCatalogName", "id"),
	)
	if name == "" {
		name = s.nextNameLocked("signal-catalog")
	}
	resource := s.ensureSignalCatalogLocked(name, now)
	s.applyIdempotencyTokenLocked(action, payload, name)
	return resource
}

func (s *iotFleetWiseStore) createModelManifestLocked(action string, payload map[string]any, now string) map[string]any {
	name := iotFleetWiseFirstNonEmpty(
		s.idempotentNameLocked(action, payload),
		iotFleetWisePayloadString(payload, "name", "modelManifestName", "id"),
	)
	if name == "" {
		name = s.nextNameLocked("model-manifest")
	}
	signalCatalog := s.resolveSignalCatalogLocked(s.signalCatalogIdentifier(payload), now)
	resource := s.ensureModelManifestLocked(name, iotFleetWisePayloadString(signalCatalog, "arn"), now)
	s.applyIdempotencyTokenLocked(action, payload, name)
	return resource
}

func (s *iotFleetWiseStore) createDecoderManifestLocked(action string, payload map[string]any, now string) map[string]any {
	name := iotFleetWiseFirstNonEmpty(
		s.idempotentNameLocked(action, payload),
		iotFleetWisePayloadString(payload, "name", "decoderManifestName", "id"),
	)
	if name == "" {
		name = s.nextNameLocked("decoder-manifest")
	}
	modelManifest := s.resolveModelManifestLocked(s.modelManifestIdentifier(payload), now)
	resource := s.ensureDecoderManifestLocked(name, iotFleetWisePayloadString(modelManifest, "arn"), now)
	s.applyIdempotencyTokenLocked(action, payload, name)
	return resource
}

func (s *iotFleetWiseStore) createFleetLocked(action string, payload map[string]any, now string) map[string]any {
	name := iotFleetWiseFirstNonEmpty(
		s.idempotentNameLocked(action, payload),
		iotFleetWisePayloadString(payload, "name", "fleetName", "fleetId", "id"),
	)
	if name == "" {
		name = s.nextNameLocked("fleet")
	}
	signalCatalog := s.resolveSignalCatalogLocked(s.signalCatalogIdentifier(payload), now)
	resource := s.ensureFleetLocked(name, iotFleetWisePayloadString(signalCatalog, "arn"), now)
	s.applyIdempotencyTokenLocked(action, payload, name)
	return resource
}

func (s *iotFleetWiseStore) createVehicleLocked(action string, payload map[string]any, now string) map[string]any {
	name := iotFleetWiseFirstNonEmpty(
		s.idempotentNameLocked(action, payload),
		iotFleetWisePayloadString(payload, "vehicleName", "name", "id"),
	)
	if name == "" {
		name = s.nextNameLocked("vehicle")
	}
	fleet := s.resolveFleetLocked(s.fleetIdentifier(payload), now)
	modelManifest := s.resolveModelManifestLocked(s.modelManifestIdentifier(payload), now)
	decoderManifest := s.resolveDecoderManifestLocked(s.decoderManifestIdentifier(payload), now)
	resource := s.ensureVehicleLocked(
		name,
		iotFleetWisePayloadString(fleet, "arn"),
		iotFleetWisePayloadString(modelManifest, "arn"),
		iotFleetWisePayloadString(decoderManifest, "arn"),
		now,
	)
	s.applyIdempotencyTokenLocked(action, payload, name)
	return resource
}

func (s *iotFleetWiseStore) createCampaignLocked(action string, payload map[string]any, now string) map[string]any {
	name := iotFleetWiseFirstNonEmpty(
		s.idempotentNameLocked(action, payload),
		iotFleetWisePayloadString(payload, "name", "campaignName", "id"),
	)
	if name == "" {
		name = s.nextNameLocked("campaign")
	}
	fleet := s.resolveFleetLocked(s.fleetIdentifier(payload), now)
	signalCatalog := s.resolveSignalCatalogLocked(s.signalCatalogIdentifier(payload), now)
	targetARN := iotFleetWiseFirstNonEmpty(
		iotFleetWisePayloadString(payload, "targetArn", "targetARN"),
		iotFleetWisePayloadString(fleet, "arn"),
	)
	resource := s.ensureCampaignLocked(name, targetARN, iotFleetWisePayloadString(signalCatalog, "arn"), now)
	s.applyIdempotencyTokenLocked(action, payload, name)
	return resource
}

func (s *iotFleetWiseStore) createStateTemplateLocked(action string, payload map[string]any, now string) map[string]any {
	name := iotFleetWiseFirstNonEmpty(
		s.idempotentNameLocked(action, payload),
		iotFleetWisePayloadString(payload, "name", "stateTemplateName", "id"),
	)
	if name == "" {
		name = s.nextNameLocked("state-template")
	}
	signalCatalog := s.resolveSignalCatalogLocked(s.signalCatalogIdentifier(payload), now)
	resource := s.ensureStateTemplateLocked(name, iotFleetWisePayloadString(signalCatalog, "arn"), now)
	s.applyIdempotencyTokenLocked(action, payload, name)
	return resource
}

func (s *iotFleetWiseStore) batchCreateVehiclesLocked(payload map[string]any, now string) map[string]any {
	items := iotFleetWisePayloadList(payload, "vehicles", "createVehicleRequestItems", "createVehicleRequestItem")
	if len(items) == 0 {
		items = []any{map[string]any{}}
	}

	out := make([]any, 0, len(items))
	for _, item := range items {
		itemPayload, _ := item.(map[string]any)
		created := s.createVehicleLocked("BatchCreateVehicle", itemPayload, now)
		out = append(out, map[string]any{
			"vehicleName": iotFleetWisePayloadString(created, "vehicleName"),
			"arn":         iotFleetWisePayloadString(created, "arn"),
		})
	}
	return map[string]any{"vehicles": out, "errors": []any{}}
}

func (s *iotFleetWiseStore) batchUpdateVehiclesLocked(payload map[string]any, now string) map[string]any {
	items := iotFleetWisePayloadList(payload, "vehicles", "updateVehicleRequestItems", "updateVehicleRequestItem")
	if len(items) == 0 {
		items = []any{map[string]any{"vehicleName": "stackyard-vehicle"}}
	}

	out := make([]any, 0, len(items))
	for _, item := range items {
		itemPayload, _ := item.(map[string]any)
		resource := s.resolveVehicleLocked(s.vehicleIdentifier(itemPayload), now)
		s.updateCommonResourceLocked(resource, itemPayload, now)
		out = append(out, map[string]any{
			"vehicleName": iotFleetWisePayloadString(resource, "vehicleName"),
			"arn":         iotFleetWisePayloadString(resource, "arn"),
		})
	}
	return map[string]any{"vehicles": out, "errors": []any{}}
}

func (s *iotFleetWiseStore) ensureSignalCatalogLocked(name, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = s.nextNameLocked("signal-catalog")
	}
	if existing, ok := s.signalCatalogs[name]; ok {
		return existing
	}
	resource := map[string]any{
		"id":                   s.nextIdentifierLocked("signal-catalog"),
		"name":                 name,
		"arn":                  s.resourceARN("signal-catalog", name),
		"description":          "Stackyard signal catalog",
		"status":               "ACTIVE",
		"creationTime":         now,
		"lastModificationTime": now,
		"nodes": []any{
			map[string]any{"fullyQualifiedName": "Vehicle.Speed", "nodeType": "SENSOR"},
		},
	}
	s.signalCatalogs[name] = resource
	return resource
}

func (s *iotFleetWiseStore) ensureModelManifestLocked(name, signalCatalogARN, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = s.nextNameLocked("model-manifest")
	}
	if existing, ok := s.modelManifests[name]; ok {
		return existing
	}
	resource := map[string]any{
		"id":                   s.nextIdentifierLocked("model-manifest"),
		"name":                 name,
		"arn":                  s.resourceARN("model-manifest", name),
		"description":          "Stackyard model manifest",
		"status":               "ACTIVE",
		"signalCatalogArn":     signalCatalogARN,
		"creationTime":         now,
		"lastModificationTime": now,
		"nodes": []any{
			map[string]any{"fullyQualifiedName": "Vehicle.Speed", "nodeType": "SENSOR"},
		},
	}
	s.modelManifests[name] = resource
	return resource
}

func (s *iotFleetWiseStore) ensureDecoderManifestLocked(name, modelManifestARN, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = s.nextNameLocked("decoder-manifest")
	}
	if existing, ok := s.decoderManifests[name]; ok {
		return existing
	}
	resource := map[string]any{
		"id":                   s.nextIdentifierLocked("decoder-manifest"),
		"name":                 name,
		"arn":                  s.resourceARN("decoder-manifest", name),
		"description":          "Stackyard decoder manifest",
		"status":               "ACTIVE",
		"modelManifestArn":     modelManifestARN,
		"creationTime":         now,
		"lastModificationTime": now,
		"networkInterfaces": []any{
			map[string]any{"interfaceId": "can0", "type": "CAN_INTERFACE"},
		},
		"signalDecoders": []any{
			map[string]any{"fullyQualifiedName": "Vehicle.Speed", "interfaceId": "can0", "type": "CAN_SIGNAL"},
		},
	}
	s.decoderManifests[name] = resource
	return resource
}

func (s *iotFleetWiseStore) ensureFleetLocked(name, signalCatalogARN, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = s.nextNameLocked("fleet")
	}
	if existing, ok := s.fleets[name]; ok {
		return existing
	}
	resource := map[string]any{
		"id":                   s.nextIdentifierLocked("fleet"),
		"name":                 name,
		"arn":                  s.resourceARN("fleet", name),
		"description":          "Stackyard fleet",
		"status":               "ACTIVE",
		"signalCatalogArn":     signalCatalogARN,
		"creationTime":         now,
		"lastModificationTime": now,
	}
	s.fleets[name] = resource
	return resource
}

func (s *iotFleetWiseStore) ensureVehicleLocked(name, fleetARN, modelManifestARN, decoderManifestARN, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = s.nextNameLocked("vehicle")
	}
	if existing, ok := s.vehicles[name]; ok {
		return existing
	}
	resource := map[string]any{
		"id":                   s.nextIdentifierLocked("vehicle"),
		"vehicleName":          name,
		"name":                 name,
		"arn":                  s.resourceARN("vehicle", name),
		"modelManifestArn":     modelManifestARN,
		"decoderManifestArn":   decoderManifestARN,
		"associatedFleetArn":   fleetARN,
		"status":               "ACTIVE",
		"creationTime":         now,
		"lastModificationTime": now,
	}
	s.vehicles[name] = resource
	return resource
}

func (s *iotFleetWiseStore) ensureCampaignLocked(name, targetARN, signalCatalogARN, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = s.nextNameLocked("campaign")
	}
	if existing, ok := s.campaigns[name]; ok {
		return existing
	}
	resource := map[string]any{
		"id":                   s.nextIdentifierLocked("campaign"),
		"name":                 name,
		"arn":                  s.resourceARN("campaign", name),
		"targetArn":            targetARN,
		"signalCatalogArn":     signalCatalogARN,
		"description":          "Stackyard campaign",
		"status":               "ACTIVE",
		"creationTime":         now,
		"lastModificationTime": now,
	}
	s.campaigns[name] = resource
	return resource
}

func (s *iotFleetWiseStore) ensureStateTemplateLocked(name, signalCatalogARN, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = s.nextNameLocked("state-template")
	}
	if existing, ok := s.stateTemplates[name]; ok {
		return existing
	}
	resource := map[string]any{
		"id":                   s.nextIdentifierLocked("state-template"),
		"name":                 name,
		"arn":                  s.resourceARN("state-template", name),
		"signalCatalogArn":     signalCatalogARN,
		"description":          "Stackyard state template",
		"status":               "ACTIVE",
		"creationTime":         now,
		"lastModificationTime": now,
	}
	s.stateTemplates[name] = resource
	return resource
}

func (s *iotFleetWiseStore) resolveSignalCatalogLocked(identifier, now string) map[string]any {
	if resource := s.lookupResourceLocked(s.signalCatalogs, identifier); resource != nil {
		return resource
	}
	name := iotFleetWiseFirstNonEmpty(strings.TrimSpace(identifier), "stackyard-signal-catalog")
	return s.ensureSignalCatalogLocked(name, now)
}

func (s *iotFleetWiseStore) resolveModelManifestLocked(identifier, now string) map[string]any {
	if resource := s.lookupResourceLocked(s.modelManifests, identifier); resource != nil {
		return resource
	}
	name := iotFleetWiseFirstNonEmpty(strings.TrimSpace(identifier), "stackyard-model-manifest")
	signalCatalog := s.resolveSignalCatalogLocked("", now)
	return s.ensureModelManifestLocked(name, iotFleetWisePayloadString(signalCatalog, "arn"), now)
}

func (s *iotFleetWiseStore) resolveDecoderManifestLocked(identifier, now string) map[string]any {
	if resource := s.lookupResourceLocked(s.decoderManifests, identifier); resource != nil {
		return resource
	}
	name := iotFleetWiseFirstNonEmpty(strings.TrimSpace(identifier), "stackyard-decoder-manifest")
	modelManifest := s.resolveModelManifestLocked("", now)
	return s.ensureDecoderManifestLocked(name, iotFleetWisePayloadString(modelManifest, "arn"), now)
}

func (s *iotFleetWiseStore) resolveFleetLocked(identifier, now string) map[string]any {
	if resource := s.lookupResourceLocked(s.fleets, identifier); resource != nil {
		return resource
	}
	name := iotFleetWiseFirstNonEmpty(strings.TrimSpace(identifier), "stackyard-fleet")
	signalCatalog := s.resolveSignalCatalogLocked("", now)
	return s.ensureFleetLocked(name, iotFleetWisePayloadString(signalCatalog, "arn"), now)
}

func (s *iotFleetWiseStore) resolveVehicleLocked(identifier, now string) map[string]any {
	if resource := s.lookupResourceLocked(s.vehicles, identifier); resource != nil {
		return resource
	}
	name := iotFleetWiseFirstNonEmpty(strings.TrimSpace(identifier), "stackyard-vehicle")
	fleet := s.resolveFleetLocked("", now)
	modelManifest := s.resolveModelManifestLocked("", now)
	decoderManifest := s.resolveDecoderManifestLocked("", now)
	return s.ensureVehicleLocked(
		name,
		iotFleetWisePayloadString(fleet, "arn"),
		iotFleetWisePayloadString(modelManifest, "arn"),
		iotFleetWisePayloadString(decoderManifest, "arn"),
		now,
	)
}

func (s *iotFleetWiseStore) resolveCampaignLocked(identifier, now string) map[string]any {
	if resource := s.lookupResourceLocked(s.campaigns, identifier); resource != nil {
		return resource
	}
	name := iotFleetWiseFirstNonEmpty(strings.TrimSpace(identifier), "stackyard-campaign")
	fleet := s.resolveFleetLocked("", now)
	signalCatalog := s.resolveSignalCatalogLocked("", now)
	return s.ensureCampaignLocked(name, iotFleetWisePayloadString(fleet, "arn"), iotFleetWisePayloadString(signalCatalog, "arn"), now)
}

func (s *iotFleetWiseStore) resolveStateTemplateLocked(identifier, now string) map[string]any {
	if resource := s.lookupResourceLocked(s.stateTemplates, identifier); resource != nil {
		return resource
	}
	name := iotFleetWiseFirstNonEmpty(strings.TrimSpace(identifier), "stackyard-state-template")
	signalCatalog := s.resolveSignalCatalogLocked("", now)
	return s.ensureStateTemplateLocked(name, iotFleetWisePayloadString(signalCatalog, "arn"), now)
}

func (s *iotFleetWiseStore) lookupResourceLocked(resources map[string]map[string]any, identifier string) map[string]any {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		if len(resources) == 0 {
			return nil
		}
		names := make([]string, 0, len(resources))
		for name := range resources {
			names = append(names, name)
		}
		sort.Strings(names)
		return resources[names[0]]
	}
	if resource, ok := resources[identifier]; ok {
		return resource
	}
	if name := iotFleetWiseExtractNameFromARN(identifier); name != "" {
		if resource, ok := resources[name]; ok {
			return resource
		}
	}
	for _, resource := range resources {
		if iotFleetWisePayloadString(resource, "id") == identifier {
			return resource
		}
		if iotFleetWisePayloadString(resource, "arn") == identifier {
			return resource
		}
		if iotFleetWisePayloadString(resource, "name") == identifier || iotFleetWisePayloadString(resource, "vehicleName") == identifier {
			return resource
		}
	}
	return nil
}

func (s *iotFleetWiseStore) listSignalCatalogSummariesLocked() []any {
	return s.resourceSummariesLocked(s.signalCatalogs)
}

func (s *iotFleetWiseStore) listModelManifestSummariesLocked() []any {
	return s.resourceSummariesLocked(s.modelManifests)
}

func (s *iotFleetWiseStore) listDecoderManifestSummariesLocked() []any {
	return s.resourceSummariesLocked(s.decoderManifests)
}

func (s *iotFleetWiseStore) listFleetSummariesLocked() []any {
	return s.resourceSummariesLocked(s.fleets)
}

func (s *iotFleetWiseStore) listVehicleSummariesLocked() []any {
	return s.resourceSummariesLocked(s.vehicles)
}

func (s *iotFleetWiseStore) listCampaignSummariesLocked() []any {
	return s.resourceSummariesLocked(s.campaigns)
}

func (s *iotFleetWiseStore) listStateTemplateSummariesLocked() []any {
	return s.resourceSummariesLocked(s.stateTemplates)
}

func (s *iotFleetWiseStore) resourceSummariesLocked(resources map[string]map[string]any) []any {
	names := make([]string, 0, len(resources))
	for name := range resources {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		resource := resources[name]
		out = append(out, map[string]any{
			"id":     iotFleetWisePayloadString(resource, "id"),
			"name":   iotFleetWiseFirstNonEmpty(iotFleetWisePayloadString(resource, "name"), iotFleetWisePayloadString(resource, "vehicleName")),
			"arn":    iotFleetWisePayloadString(resource, "arn"),
			"status": iotFleetWiseFirstNonEmpty(iotFleetWisePayloadString(resource, "status"), "ACTIVE"),
		})
	}
	return out
}

func (s *iotFleetWiseStore) associateVehicleFleetLocked(vehicleName, fleetIdentifier string) {
	vehicleName = strings.TrimSpace(vehicleName)
	if vehicleName == "" {
		return
	}
	fleet := s.lookupResourceLocked(s.fleets, fleetIdentifier)
	if fleet == nil {
		fleet = s.lookupResourceLocked(s.fleets, "")
	}
	if fleet == nil {
		return
	}
	fleetID := iotFleetWisePayloadString(fleet, "id")
	if fleetID == "" {
		return
	}
	if _, ok := s.fleetVehicles[fleetID]; !ok {
		s.fleetVehicles[fleetID] = map[string]struct{}{}
	}
	s.fleetVehicles[fleetID][vehicleName] = struct{}{}
}

func (s *iotFleetWiseStore) disassociateVehicleFleetLocked(vehicleName, fleetIdentifier string) {
	vehicleName = strings.TrimSpace(vehicleName)
	if vehicleName == "" {
		return
	}
	fleet := s.lookupResourceLocked(s.fleets, fleetIdentifier)
	if fleet == nil {
		return
	}
	fleetID := iotFleetWisePayloadString(fleet, "id")
	if fleetID == "" {
		return
	}
	if assoc, ok := s.fleetVehicles[fleetID]; ok {
		delete(assoc, vehicleName)
	}
}

func (s *iotFleetWiseStore) listFleetsForVehicleLocked(vehicleName string) []any {
	vehicleName = strings.TrimSpace(vehicleName)
	out := []any{}
	if vehicleName == "" {
		return out
	}
	for fleetID, assoc := range s.fleetVehicles {
		if _, ok := assoc[vehicleName]; !ok {
			continue
		}
		for _, fleet := range s.fleets {
			if iotFleetWisePayloadString(fleet, "id") == fleetID {
				out = append(out, map[string]any{
					"id":     iotFleetWisePayloadString(fleet, "id"),
					"name":   iotFleetWisePayloadString(fleet, "name"),
					"arn":    iotFleetWisePayloadString(fleet, "arn"),
					"status": iotFleetWisePayloadString(fleet, "status"),
				})
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := out[i].(map[string]any)
		right, _ := out[j].(map[string]any)
		return iotFleetWisePayloadString(left, "name") < iotFleetWisePayloadString(right, "name")
	})
	return out
}

func (s *iotFleetWiseStore) listVehiclesInFleetLocked(fleetIdentifier string) []any {
	fleet := s.lookupResourceLocked(s.fleets, fleetIdentifier)
	if fleet == nil {
		return []any{}
	}
	fleetID := iotFleetWisePayloadString(fleet, "id")
	assoc := s.fleetVehicles[fleetID]
	if len(assoc) == 0 {
		return []any{}
	}
	names := make([]string, 0, len(assoc))
	for name := range assoc {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		vehicle := s.lookupResourceLocked(s.vehicles, name)
		if vehicle == nil {
			continue
		}
		out = append(out, map[string]any{
			"vehicleName": iotFleetWisePayloadString(vehicle, "vehicleName"),
			"arn":         iotFleetWisePayloadString(vehicle, "arn"),
			"status":      iotFleetWisePayloadString(vehicle, "status"),
		})
	}
	return out
}

func (s *iotFleetWiseStore) vehicleCampaignStatusesLocked(vehicle map[string]any) []any {
	fleetARN := iotFleetWisePayloadString(vehicle, "associatedFleetArn")
	campaigns := make([]any, 0, len(s.campaigns))
	for _, campaign := range s.campaigns {
		if fleetARN == "" || iotFleetWisePayloadString(campaign, "targetArn") == fleetARN {
			campaigns = append(campaigns, map[string]any{
				"name":   iotFleetWisePayloadString(campaign, "name"),
				"status": iotFleetWiseFirstNonEmpty(iotFleetWisePayloadString(campaign, "status"), "ACTIVE"),
			})
		}
	}
	if len(campaigns) == 0 {
		campaigns = append(campaigns, map[string]any{"name": "stackyard-campaign", "status": "ACTIVE"})
	}
	return campaigns
}

func (s *iotFleetWiseStore) updateCommonResourceLocked(resource map[string]any, payload map[string]any, now string) {
	if resource == nil {
		return
	}
	if name := iotFleetWisePayloadString(payload, "name", "description", "vehicleName"); name != "" {
		if _, hasVehicleName := resource["vehicleName"]; hasVehicleName {
			resource["vehicleName"] = name
			resource["name"] = name
		} else if _, hasName := resource["name"]; hasName {
			resource["name"] = name
		}
	}
	if description := iotFleetWisePayloadString(payload, "description"); description != "" {
		resource["description"] = description
	}
	if status := iotFleetWisePayloadString(payload, "status"); status != "" {
		resource["status"] = status
	}
	if targetArn := iotFleetWisePayloadString(payload, "targetArn", "targetARN"); targetArn != "" {
		resource["targetArn"] = targetArn
	}
	if signalCatalogArn := iotFleetWisePayloadString(payload, "signalCatalogArn"); signalCatalogArn != "" {
		resource["signalCatalogArn"] = signalCatalogArn
	}
	resource["lastModificationTime"] = now
}

func (s *iotFleetWiseStore) accountStatusLocked() map[string]any {
	status := "NOT_REGISTERED"
	if s.registered {
		status = "REGISTRATION_SUCCESS"
	}
	return map[string]any{
		"registerAccountStatus": status,
		"iamRegistrationResponse": map[string]any{
			"roleArn": "arn:aws:iam::123456789012:role/stackyard-iotfleetwise",
		},
		"timestreamRegistrationResponse": map[string]any{
			"timestreamDatabaseName": "stackyard_iotfleetwise",
			"timestreamTableName":    "vehicle_data",
		},
	}
}

func (s *iotFleetWiseStore) resourceARN(resourceType, name string) string {
	return fmt.Sprintf("arn:aws:iotfleetwise:%s:%s:%s/%s", iotFleetWiseDefaultRegion, iotFleetWiseDefaultAccountID, resourceType, name)
}

func (s *iotFleetWiseStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = s.defaultResourceARNLocked()
	}
	if _, ok := s.tags[resourceARN]; !ok {
		s.tags[resourceARN] = map[string]string{}
	}
	return s.tags[resourceARN]
}

func (s *iotFleetWiseStore) defaultResourceARNLocked() string {
	if fleet := s.lookupResourceLocked(s.fleets, ""); fleet != nil {
		if arn := iotFleetWisePayloadString(fleet, "arn"); arn != "" {
			return arn
		}
	}
	return s.resourceARN("fleet", "stackyard-fleet")
}

func (s *iotFleetWiseStore) idempotentNameLocked(action string, payload map[string]any) string {
	token := strings.TrimSpace(iotFleetWisePayloadString(payload, "clientToken"))
	if token == "" {
		return ""
	}
	return strings.TrimSpace(s.createTokens[action+"|"+token])
}

func (s *iotFleetWiseStore) applyIdempotencyTokenLocked(action string, payload map[string]any, name string) {
	token := strings.TrimSpace(iotFleetWisePayloadString(payload, "clientToken"))
	if token == "" || strings.TrimSpace(name) == "" {
		return
	}
	s.createTokens[action+"|"+token] = name
}

func (s *iotFleetWiseStore) nextNameLocked(prefix string) string {
	name := fmt.Sprintf("stackyard-%s-%06d", prefix, s.nextID)
	s.nextID++
	return name
}

func (s *iotFleetWiseStore) nextIdentifierLocked(prefix string) string {
	id := fmt.Sprintf("%s-%06d", prefix, s.nextID)
	s.nextID++
	return id
}

func (s *iotFleetWiseStore) signalCatalogIdentifier(payload map[string]any) string {
	return iotFleetWisePayloadString(payload, "signalCatalogName", "name", "signalCatalogArn", "arn", "id")
}

func (s *iotFleetWiseStore) modelManifestIdentifier(payload map[string]any) string {
	return iotFleetWisePayloadString(payload, "modelManifestName", "name", "modelManifestArn", "arn", "id")
}

func (s *iotFleetWiseStore) decoderManifestIdentifier(payload map[string]any) string {
	return iotFleetWisePayloadString(payload, "decoderManifestName", "name", "decoderManifestArn", "arn", "id")
}

func (s *iotFleetWiseStore) fleetIdentifier(payload map[string]any) string {
	return iotFleetWisePayloadString(payload, "fleetId", "fleetName", "name", "fleetArn", "arn", "id", "targetArn")
}

func (s *iotFleetWiseStore) vehicleIdentifier(payload map[string]any) string {
	return iotFleetWisePayloadString(payload, "vehicleName", "name", "vehicleArn", "arn", "id")
}

func (s *iotFleetWiseStore) campaignIdentifier(payload map[string]any) string {
	return iotFleetWisePayloadString(payload, "campaignName", "name", "campaignArn", "arn", "id")
}

func (s *iotFleetWiseStore) stateTemplateIdentifier(payload map[string]any) string {
	return iotFleetWisePayloadString(payload, "stateTemplateName", "name", "stateTemplateArn", "arn", "id")
}

func iotFleetWiseExtractNameFromARN(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if !strings.Contains(identifier, "arn:") || !strings.Contains(identifier, "/") {
		return ""
	}
	parts := strings.Split(identifier, "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func iotFleetWisePayloadString(payload map[string]any, keys ...string) string {
	if payload == nil {
		return ""
	}
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func iotFleetWisePayloadMap(payload map[string]any, keys ...string) map[string]any {
	if payload == nil {
		return nil
	}
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		value, ok := raw.(map[string]any)
		if ok && len(value) > 0 {
			return value
		}
	}
	return nil
}

func iotFleetWisePayloadList(payload map[string]any, keys ...string) []any {
	if payload == nil {
		return nil
	}
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		value, ok := raw.([]any)
		if ok {
			return value
		}
	}
	return nil
}

func iotFleetWisePayloadStringSlice(payload map[string]any, keys ...string) []string {
	out := []string{}
	values := iotFleetWisePayloadList(payload, keys...)
	for _, item := range values {
		value, ok := item.(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func iotFleetWisePayloadStringMap(payload map[string]any, keys ...string) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}

	for _, key := range keys {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case map[string]any:
			for k, v := range value {
				trimmedKey := strings.TrimSpace(k)
				if trimmedKey == "" {
					continue
				}
				out[trimmedKey] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		case []any:
			for _, item := range value {
				tagMap, ok := item.(map[string]any)
				if !ok {
					continue
				}
				tagKey := iotFleetWiseFirstNonEmpty(
					iotFleetWisePayloadString(tagMap, "Key", "key"),
					iotFleetWisePayloadString(tagMap, "TagKey"),
				)
				if tagKey == "" {
					continue
				}
				tagValue := iotFleetWiseFirstNonEmpty(
					iotFleetWisePayloadString(tagMap, "Value", "value"),
					iotFleetWisePayloadString(tagMap, "TagValue"),
				)
				out[tagKey] = tagValue
			}
		}
	}
	return out
}

func iotFleetWiseTagsList(tags map[string]string) []any {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}

func iotFleetWiseFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func iotFleetWiseCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return iotFleetWiseCloneMap(typed)
	case []any:
		return iotFleetWiseCloneList(typed)
	default:
		return typed
	}
}

func iotFleetWiseCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = iotFleetWiseCloneAny(v)
	}
	return out
}

func iotFleetWiseCloneList(in []any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, iotFleetWiseCloneAny(item))
	}
	return out
}
