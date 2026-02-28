package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type iotTwinMakerStore struct {
	mu                   sync.Mutex
	nextID               int64
	workspaces           map[string]*iotTwinMakerWorkspace
	componentTypes       map[string]map[string]*iotTwinMakerComponentType
	entities             map[string]map[string]*iotTwinMakerEntity
	scenes               map[string]map[string]*iotTwinMakerScene
	syncJobs             map[string]map[string]*iotTwinMakerSyncJob
	metadataTransferJobs map[string]*iotTwinMakerMetadataTransferJob
	tags                 map[string]map[string]string
	pricingCurrent       map[string]any
	pricingPending       map[string]any
}

type iotTwinMakerWorkspace struct {
	ID          string
	Arn         string
	Description string
	S3Location  string
	Role        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type iotTwinMakerComponentType struct {
	WorkspaceID string
	ID          string
	Arn         string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type iotTwinMakerEntity struct {
	WorkspaceID   string
	ID            string
	Name          string
	Arn           string
	Description   string
	ParentEntity  string
	Components    map[string]map[string]any
	CreatedAt     time.Time
	UpdatedAt     time.Time
	HasChildItems bool
}

type iotTwinMakerScene struct {
	WorkspaceID     string
	ID              string
	Arn             string
	Description     string
	ContentLocation string
	Capabilities    []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type iotTwinMakerSyncJob struct {
	WorkspaceID string
	SyncSource  string
	Arn         string
	Role        string
	State       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type iotTwinMakerMetadataTransferJob struct {
	ID          string
	Arn         string
	Description string
	State       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func newIoTTwinMakerStore() *iotTwinMakerStore {
	now := time.Now().UTC()
	workspaceID := "stackyard-workspace"
	workspace := &iotTwinMakerWorkspace{
		ID:          workspaceID,
		Arn:         iotTwinMakerWorkspaceARN(workspaceID),
		Description: "seed workspace",
		S3Location:  "s3://stackyard-iottwinmaker/workspaces/stackyard-workspace",
		Role:        "arn:aws:iam::123456789012:role/stackyard-iottwinmaker",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	entityID := "stackyard-entity"
	entity := &iotTwinMakerEntity{
		WorkspaceID: workspaceID,
		ID:          entityID,
		Name:        "stackyard-entity",
		Arn:         iotTwinMakerEntityARN(workspaceID, entityID),
		Description: "seed entity",
		Components: map[string]map[string]any{
			"sensor": {
				"componentTypeId": "com.example.Sensor",
				"description":     "seed component",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	componentTypeID := "com.example.Sensor"
	componentType := &iotTwinMakerComponentType{
		WorkspaceID: workspaceID,
		ID:          componentTypeID,
		Arn:         iotTwinMakerComponentTypeARN(workspaceID, componentTypeID),
		Description: "seed component type",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	sceneID := "stackyard-scene"
	scene := &iotTwinMakerScene{
		WorkspaceID:     workspaceID,
		ID:              sceneID,
		Arn:             iotTwinMakerSceneARN(workspaceID, sceneID),
		Description:     "seed scene",
		ContentLocation: "s3://stackyard-iottwinmaker/scenes/stackyard-scene.json",
		Capabilities:    []string{"public-read"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	syncSource := "SITEWISE"
	sync := &iotTwinMakerSyncJob{
		WorkspaceID: workspaceID,
		SyncSource:  syncSource,
		Arn:         iotTwinMakerSyncJobARN(workspaceID, syncSource),
		Role:        "arn:aws:iam::123456789012:role/stackyard-iottwinmaker-sync",
		State:       "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	transferID := "mtj-000001"
	transfer := &iotTwinMakerMetadataTransferJob{
		ID:          transferID,
		Arn:         iotTwinMakerMetadataTransferJobARN(transferID),
		Description: "seed metadata transfer",
		State:       "COMPLETED",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return &iotTwinMakerStore{
		nextID: 2,
		workspaces: map[string]*iotTwinMakerWorkspace{
			workspaceID: workspace,
		},
		componentTypes: map[string]map[string]*iotTwinMakerComponentType{
			workspaceID: {componentTypeID: componentType},
		},
		entities: map[string]map[string]*iotTwinMakerEntity{
			workspaceID: {entityID: entity},
		},
		scenes: map[string]map[string]*iotTwinMakerScene{
			workspaceID: {sceneID: scene},
		},
		syncJobs: map[string]map[string]*iotTwinMakerSyncJob{
			workspaceID: {syncSource: sync},
		},
		metadataTransferJobs: map[string]*iotTwinMakerMetadataTransferJob{
			transferID: transfer,
		},
		tags: map[string]map[string]string{
			workspace.Arn: {"seed": "true"},
		},
		pricingCurrent: map[string]any{
			"pricingMode":         "STANDARD",
			"billableEntityCount": int64(1),
			"effectiveDateTime":   now,
			"updateDateTime":      now,
			"updateReason":        "INITIAL",
			"bundleInformation": map[string]any{
				"bundleNames": []any{"TwinMakerStandard"},
			},
		},
		pricingPending: map[string]any{},
	}
}

func (s *iotTwinMakerStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "BatchPutPropertyValues":
		return map[string]any{"errorEntries": []any{}}

	case "CancelMetadataTransferJob":
		jobID := strings.TrimSpace(pathParams["metadataTransferJobId"])
		if jobID == "" {
			jobID = iotTwinMakerDefaultString(payload, "metadataTransferJobId", "")
		}
		job := s.ensureMetadataTransferJobLocked(jobID)
		job.State = "CANCELED"
		job.UpdatedAt = now
		return map[string]any{
			"metadataTransferJobId": job.ID,
			"arn":                   job.Arn,
			"updateDateTime":        job.UpdatedAt,
			"status":                iotTwinMakerMetadataTransferStatus(job.State),
			"progress":              iotTwinMakerMetadataTransferProgress(job.State),
		}

	case "CreateComponentType":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		componentTypeID := strings.TrimSpace(pathParams["componentTypeId"])
		if componentTypeID == "" {
			componentTypeID = iotTwinMakerDefaultString(payload, "componentTypeId", fmt.Sprintf("com.stackyard.Component%06d", s.nextLocked()))
		}
		ct := s.ensureComponentTypeLocked(workspaceID, componentTypeID)
		ct.Description = iotTwinMakerDefaultString(payload, "description", ct.Description)
		ct.UpdatedAt = now
		return map[string]any{"arn": ct.Arn, "creationDateTime": ct.CreatedAt, "state": "ACTIVE"}

	case "CreateEntity":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		entityID := iotTwinMakerDefaultString(payload, "entityId", fmt.Sprintf("entity-%06d", s.nextLocked()))
		entity := s.ensureEntityLocked(workspaceID, entityID)
		entity.Name = iotTwinMakerDefaultString(payload, "entityName", entity.Name)
		entity.Description = iotTwinMakerDefaultString(payload, "description", entity.Description)
		entity.ParentEntity = iotTwinMakerDefaultString(payload, "parentEntityId", entity.ParentEntity)
		entity.UpdatedAt = now
		return map[string]any{"entityId": entity.ID, "arn": entity.Arn, "creationDateTime": entity.CreatedAt, "state": "ACTIVE"}

	case "CreateMetadataTransferJob":
		jobID := iotTwinMakerDefaultString(payload, "metadataTransferJobId", fmt.Sprintf("mtj-%06d", s.nextLocked()))
		job := s.ensureMetadataTransferJobLocked(jobID)
		job.Description = iotTwinMakerDefaultString(payload, "description", job.Description)
		job.State = "PENDING"
		job.UpdatedAt = now
		return map[string]any{"metadataTransferJobId": job.ID, "arn": job.Arn, "creationDateTime": job.CreatedAt, "status": iotTwinMakerMetadataTransferStatus(job.State)}

	case "CreateScene":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		sceneID := iotTwinMakerDefaultString(payload, "sceneId", fmt.Sprintf("scene-%06d", s.nextLocked()))
		scene := s.ensureSceneLocked(workspaceID, sceneID)
		scene.Description = iotTwinMakerDefaultString(payload, "description", scene.Description)
		scene.ContentLocation = iotTwinMakerDefaultString(payload, "contentLocation", scene.ContentLocation)
		scene.UpdatedAt = now
		return map[string]any{"arn": scene.Arn, "creationDateTime": scene.CreatedAt}

	case "CreateSyncJob":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		syncSource := iotTwinMakerResolveSyncSource(payload, pathParams)
		job := s.ensureSyncJobLocked(workspaceID, syncSource)
		job.Role = iotTwinMakerDefaultString(payload, "syncRole", job.Role)
		job.State = "ACTIVE"
		job.UpdatedAt = now
		return map[string]any{"arn": job.Arn, "creationDateTime": job.CreatedAt, "state": job.State}

	case "CreateWorkspace":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		ws := s.ensureWorkspaceLocked(workspaceID)
		ws.Description = iotTwinMakerDefaultString(payload, "description", ws.Description)
		ws.Role = iotTwinMakerDefaultString(payload, "role", ws.Role)
		ws.S3Location = iotTwinMakerDefaultString(payload, "s3Location", ws.S3Location)
		ws.UpdatedAt = now
		return map[string]any{"arn": ws.Arn, "creationDateTime": ws.CreatedAt}

	case "DeleteComponentType":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		componentTypeID := strings.TrimSpace(pathParams["componentTypeId"])
		if componentTypeID == "" {
			componentTypeID = iotTwinMakerDefaultString(payload, "componentTypeId", "")
		}
		if componentTypeID != "" {
			delete(s.componentTypes[workspaceID], componentTypeID)
		}
		return map[string]any{"state": "DELETING"}

	case "DeleteEntity":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		entityID := strings.TrimSpace(pathParams["entityId"])
		if entityID == "" {
			entityID = iotTwinMakerDefaultString(payload, "entityId", "")
		}
		if entityID != "" {
			delete(s.entities[workspaceID], entityID)
		}
		return map[string]any{"state": "DELETING"}

	case "DeleteScene":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		sceneID := strings.TrimSpace(pathParams["sceneId"])
		if sceneID == "" {
			sceneID = iotTwinMakerDefaultString(payload, "sceneId", "")
		}
		if sceneID != "" {
			delete(s.scenes[workspaceID], sceneID)
		}
		return map[string]any{}

	case "DeleteSyncJob":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		syncSource := iotTwinMakerResolveSyncSource(payload, pathParams)
		if syncSource != "" {
			delete(s.syncJobs[workspaceID], syncSource)
		}
		return map[string]any{"state": "DELETING"}

	case "DeleteWorkspace":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		if workspaceID != "" {
			delete(s.workspaces, workspaceID)
			delete(s.componentTypes, workspaceID)
			delete(s.entities, workspaceID)
			delete(s.scenes, workspaceID)
			delete(s.syncJobs, workspaceID)
		}
		return map[string]any{"message": "Workspace deleted"}

	case "ExecuteQuery":
		return map[string]any{"columnDescriptions": []any{}, "rows": []any{}, "nextToken": ""}

	case "GetComponentType":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		componentTypeID := strings.TrimSpace(pathParams["componentTypeId"])
		if componentTypeID == "" {
			componentTypeID = iotTwinMakerDefaultString(payload, "componentTypeId", "com.stackyard.Component")
		}
		ct := s.ensureComponentTypeLocked(workspaceID, componentTypeID)
		return map[string]any{
			"workspaceId":             workspaceID,
			"componentTypeId":         ct.ID,
			"componentTypeName":       ct.ID,
			"description":             ct.Description,
			"arn":                     ct.Arn,
			"creationDateTime":        ct.CreatedAt,
			"updateDateTime":          ct.UpdatedAt,
			"isSingleton":             false,
			"isAbstract":              false,
			"isSchemaInitialized":     true,
			"propertyDefinitions":     map[string]any{},
			"extendsFrom":             []any{},
			"functions":               map[string]any{},
			"propertyGroups":          map[string]any{},
			"compositeComponentTypes": map[string]any{},
			"status":                  iotTwinMakerStatus("ACTIVE"),
		}

	case "GetEntity":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		entityID := strings.TrimSpace(pathParams["entityId"])
		if entityID == "" {
			entityID = iotTwinMakerDefaultString(payload, "entityId", "stackyard-entity")
		}
		entity := s.ensureEntityLocked(workspaceID, entityID)
		return map[string]any{
			"entityId":                 entity.ID,
			"entityName":               entity.Name,
			"arn":                      entity.Arn,
			"status":                   iotTwinMakerStatus("ACTIVE"),
			"workspaceId":              workspaceID,
			"description":              entity.Description,
			"components":               iotTwinMakerCloneNestedMap(entity.Components),
			"parentEntityId":           entity.ParentEntity,
			"hasChildEntities":         entity.HasChildItems,
			"creationDateTime":         entity.CreatedAt,
			"updateDateTime":           entity.UpdatedAt,
			"areAllComponentsReturned": true,
		}

	case "GetMetadataTransferJob":
		jobID := strings.TrimSpace(pathParams["metadataTransferJobId"])
		if jobID == "" {
			jobID = iotTwinMakerDefaultString(payload, "metadataTransferJobId", "mtj-000001")
		}
		job := s.ensureMetadataTransferJobLocked(jobID)
		return map[string]any{
			"metadataTransferJobId":   job.ID,
			"arn":                     job.Arn,
			"description":             job.Description,
			"sources":                 []any{},
			"destination":             map[string]any{},
			"metadataTransferJobRole": "arn:aws:iam::123456789012:role/stackyard-iottwinmaker-transfer",
			"reportUrl":               "",
			"creationDateTime":        job.CreatedAt,
			"updateDateTime":          job.UpdatedAt,
			"status":                  iotTwinMakerMetadataTransferStatus(job.State),
			"progress":                iotTwinMakerMetadataTransferProgress(job.State),
		}

	case "GetPricingPlan":
		return map[string]any{
			"currentPricingPlan": iotTwinMakerCloneMapAny(s.pricingCurrent),
			"pendingPricingPlan": iotTwinMakerCloneMapAny(s.pricingPending),
		}

	case "GetPropertyValue":
		return map[string]any{"propertyValues": map[string]any{}, "nextToken": "", "tabularPropertyValues": []any{}}

	case "GetPropertyValueHistory":
		return map[string]any{"propertyValues": []any{}, "nextToken": ""}

	case "GetScene":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		sceneID := strings.TrimSpace(pathParams["sceneId"])
		if sceneID == "" {
			sceneID = iotTwinMakerDefaultString(payload, "sceneId", "stackyard-scene")
		}
		scene := s.ensureSceneLocked(workspaceID, sceneID)
		return map[string]any{
			"workspaceId":            workspaceID,
			"sceneId":                scene.ID,
			"contentLocation":        scene.ContentLocation,
			"arn":                    scene.Arn,
			"creationDateTime":       scene.CreatedAt,
			"updateDateTime":         scene.UpdatedAt,
			"description":            scene.Description,
			"capabilities":           iotTwinMakerToAnySlice(scene.Capabilities),
			"sceneMetadata":          map[string]any{},
			"generatedSceneMetadata": map[string]any{},
		}

	case "GetSyncJob":
		syncSource := iotTwinMakerResolveSyncSource(payload, pathParams)
		job := s.findOrCreateSyncJobBySourceLocked(syncSource)
		return map[string]any{
			"arn":              job.Arn,
			"workspaceId":      job.WorkspaceID,
			"syncSource":       job.SyncSource,
			"syncRole":         job.Role,
			"status":           iotTwinMakerSyncJobStatus(job.State),
			"creationDateTime": job.CreatedAt,
			"updateDateTime":   job.UpdatedAt,
		}

	case "GetWorkspace":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		ws := s.ensureWorkspaceLocked(workspaceID)
		return map[string]any{
			"workspaceId":      ws.ID,
			"arn":              ws.Arn,
			"description":      ws.Description,
			"linkedServices":   []any{"iotsitewise"},
			"s3Location":       ws.S3Location,
			"role":             ws.Role,
			"creationDateTime": ws.CreatedAt,
			"updateDateTime":   ws.UpdatedAt,
		}

	case "ListComponentTypes":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		cts := s.componentTypes[workspaceID]
		keys := make([]string, 0, len(cts))
		for id := range cts {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		summaries := make([]any, 0, len(keys))
		for _, id := range keys {
			ct := cts[id]
			summaries = append(summaries, map[string]any{
				"arn":              ct.Arn,
				"componentTypeId":  ct.ID,
				"description":      ct.Description,
				"creationDateTime": ct.CreatedAt,
				"updateDateTime":   ct.UpdatedAt,
				"status":           iotTwinMakerStatus("ACTIVE"),
			})
		}
		return map[string]any{"workspaceId": workspaceID, "componentTypeSummaries": summaries, "nextToken": "", "maxResults": len(summaries)}

	case "ListComponents":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		entityID := strings.TrimSpace(pathParams["entityId"])
		if entityID == "" {
			entityID = iotTwinMakerDefaultString(payload, "entityId", "stackyard-entity")
		}
		entity := s.ensureEntityLocked(workspaceID, entityID)
		names := make([]string, 0, len(entity.Components))
		for name := range entity.Components {
			names = append(names, name)
		}
		sort.Strings(names)
		items := make([]any, 0, len(names))
		for _, name := range names {
			component := entity.Components[name]
			items = append(items, map[string]any{
				"componentName":   name,
				"componentTypeId": iotTwinMakerDefaultString(component, "componentTypeId", "com.stackyard.Component"),
				"description":     iotTwinMakerDefaultString(component, "description", ""),
				"propertyGroups":  map[string]any{},
				"status":          iotTwinMakerStatus("ACTIVE"),
			})
		}
		return map[string]any{"componentSummaries": items, "nextToken": ""}

	case "ListEntities":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		entities := s.entities[workspaceID]
		keys := make([]string, 0, len(entities))
		for id := range entities {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		items := make([]any, 0, len(keys))
		for _, id := range keys {
			entity := entities[id]
			items = append(items, map[string]any{
				"entityId":         entity.ID,
				"entityName":       entity.Name,
				"arn":              entity.Arn,
				"parentEntityId":   entity.ParentEntity,
				"status":           iotTwinMakerStatus("ACTIVE"),
				"description":      entity.Description,
				"hasChildEntities": entity.HasChildItems,
				"creationDateTime": entity.CreatedAt,
				"updateDateTime":   entity.UpdatedAt,
			})
		}
		return map[string]any{"entitySummaries": items, "nextToken": ""}

	case "ListMetadataTransferJobs":
		keys := make([]string, 0, len(s.metadataTransferJobs))
		for id := range s.metadataTransferJobs {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		items := make([]any, 0, len(keys))
		for _, id := range keys {
			job := s.metadataTransferJobs[id]
			items = append(items, map[string]any{
				"metadataTransferJobId": job.ID,
				"arn":                   job.Arn,
				"creationDateTime":      job.CreatedAt,
				"updateDateTime":        job.UpdatedAt,
				"status":                iotTwinMakerMetadataTransferStatus(job.State),
				"progress":              iotTwinMakerMetadataTransferProgress(job.State),
			})
		}
		return map[string]any{"metadataTransferJobSummaries": items, "nextToken": ""}

	case "ListProperties":
		return map[string]any{"propertySummaries": []any{}, "nextToken": ""}

	case "ListScenes":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		scenes := s.scenes[workspaceID]
		keys := make([]string, 0, len(scenes))
		for id := range scenes {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		items := make([]any, 0, len(keys))
		for _, id := range keys {
			scene := scenes[id]
			items = append(items, map[string]any{
				"sceneId":          scene.ID,
				"contentLocation":  scene.ContentLocation,
				"arn":              scene.Arn,
				"creationDateTime": scene.CreatedAt,
				"updateDateTime":   scene.UpdatedAt,
				"description":      scene.Description,
			})
		}
		return map[string]any{"sceneSummaries": items, "nextToken": ""}

	case "ListSyncJobs":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		jobs := s.syncJobs[workspaceID]
		keys := make([]string, 0, len(jobs))
		for source := range jobs {
			keys = append(keys, source)
		}
		sort.Strings(keys)
		items := make([]any, 0, len(keys))
		for _, source := range keys {
			job := jobs[source]
			items = append(items, map[string]any{
				"arn":              job.Arn,
				"workspaceId":      job.WorkspaceID,
				"syncSource":       job.SyncSource,
				"status":           iotTwinMakerSyncJobStatus(job.State),
				"creationDateTime": job.CreatedAt,
				"updateDateTime":   job.UpdatedAt,
			})
		}
		return map[string]any{"syncJobSummaries": items, "nextToken": ""}

	case "ListSyncResources":
		syncSource := iotTwinMakerResolveSyncSource(payload, pathParams)
		job := s.findOrCreateSyncJobBySourceLocked(syncSource)
		return map[string]any{
			"syncResources": []any{map[string]any{
				"resourceType":   "ENTITY",
				"externalId":     "stackyard-external",
				"resourceId":     "stackyard-resource",
				"status":         map[string]any{"state": "IN_SYNC", "error": map[string]any{}},
				"updateDateTime": job.UpdatedAt,
			}},
			"nextToken": "",
		}

	case "ListTagsForResource":
		resourceArn := iotTwinMakerResolveResourceARN(payload, query)
		if resourceArn == "" {
			resourceArn = iotTwinMakerWorkspaceARN("stackyard-workspace")
		}
		tags := iotTwinMakerCloneTags(s.tags[resourceArn])
		return map[string]any{"tags": tags, "nextToken": ""}

	case "ListWorkspaces":
		keys := make([]string, 0, len(s.workspaces))
		for id := range s.workspaces {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		items := make([]any, 0, len(keys))
		for _, id := range keys {
			ws := s.workspaces[id]
			items = append(items, map[string]any{
				"workspaceId":      ws.ID,
				"arn":              ws.Arn,
				"description":      ws.Description,
				"linkedServices":   []any{"iotsitewise"},
				"creationDateTime": ws.CreatedAt,
				"updateDateTime":   ws.UpdatedAt,
			})
		}
		return map[string]any{"workspaceSummaries": items, "nextToken": ""}

	case "TagResource":
		resourceArn := iotTwinMakerResolveResourceARN(payload, query)
		if resourceArn == "" {
			resourceArn = iotTwinMakerWorkspaceARN("stackyard-workspace")
		}
		if _, ok := s.tags[resourceArn]; !ok {
			s.tags[resourceArn] = map[string]string{}
		}
		tagMap := iotTwinMakerExtractTagMap(iotTwinMakerValue(payload, "tags"))
		for key, value := range tagMap {
			s.tags[resourceArn][key] = value
		}
		return map[string]any{}

	case "UntagResource":
		resourceArn := iotTwinMakerResolveResourceARN(payload, query)
		if resourceArn == "" {
			resourceArn = iotTwinMakerWorkspaceARN("stackyard-workspace")
		}
		tagKeys := iotTwinMakerStringSlice(iotTwinMakerValue(payload, "tagKeys"))
		if len(tagKeys) == 0 {
			tagKeys = iotTwinMakerStringSliceFromCSV(query.Get("tagKeys"))
		}
		for _, key := range tagKeys {
			delete(s.tags[resourceArn], key)
		}
		return map[string]any{}

	case "UpdateComponentType":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		componentTypeID := strings.TrimSpace(pathParams["componentTypeId"])
		if componentTypeID == "" {
			componentTypeID = iotTwinMakerDefaultString(payload, "componentTypeId", "com.stackyard.Component")
		}
		ct := s.ensureComponentTypeLocked(workspaceID, componentTypeID)
		ct.Description = iotTwinMakerDefaultString(payload, "description", ct.Description)
		ct.UpdatedAt = now
		return map[string]any{"workspaceId": workspaceID, "arn": ct.Arn, "componentTypeId": ct.ID, "state": "UPDATING"}

	case "UpdateEntity":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		entityID := strings.TrimSpace(pathParams["entityId"])
		if entityID == "" {
			entityID = iotTwinMakerDefaultString(payload, "entityId", "stackyard-entity")
		}
		entity := s.ensureEntityLocked(workspaceID, entityID)
		entity.Name = iotTwinMakerDefaultString(payload, "entityName", entity.Name)
		entity.Description = iotTwinMakerDefaultString(payload, "description", entity.Description)
		entity.ParentEntity = iotTwinMakerDefaultString(payload, "parentEntityId", entity.ParentEntity)
		entity.UpdatedAt = now
		return map[string]any{"updateDateTime": entity.UpdatedAt, "state": "UPDATING"}

	case "UpdatePricingPlan":
		if pricing, ok := iotTwinMakerValue(payload, "pricingMode").(string); ok && strings.TrimSpace(pricing) != "" {
			s.pricingPending = map[string]any{
				"pricingMode":         strings.TrimSpace(pricing),
				"billableEntityCount": int64(1),
				"effectiveDateTime":   now,
				"updateDateTime":      now,
				"updateReason":        "USER_ACTION",
				"bundleInformation":   map[string]any{"bundleNames": []any{"TwinMakerStandard"}},
			}
		} else {
			s.pricingPending = iotTwinMakerCloneMapAny(s.pricingCurrent)
		}
		return map[string]any{"currentPricingPlan": iotTwinMakerCloneMapAny(s.pricingCurrent), "pendingPricingPlan": iotTwinMakerCloneMapAny(s.pricingPending)}

	case "UpdateScene":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		sceneID := strings.TrimSpace(pathParams["sceneId"])
		if sceneID == "" {
			sceneID = iotTwinMakerDefaultString(payload, "sceneId", "stackyard-scene")
		}
		scene := s.ensureSceneLocked(workspaceID, sceneID)
		scene.Description = iotTwinMakerDefaultString(payload, "description", scene.Description)
		scene.ContentLocation = iotTwinMakerDefaultString(payload, "contentLocation", scene.ContentLocation)
		scene.UpdatedAt = now
		return map[string]any{"updateDateTime": scene.UpdatedAt}

	case "UpdateWorkspace":
		workspaceID := iotTwinMakerResolveWorkspaceID(payload, pathParams)
		ws := s.ensureWorkspaceLocked(workspaceID)
		ws.Description = iotTwinMakerDefaultString(payload, "description", ws.Description)
		ws.S3Location = iotTwinMakerDefaultString(payload, "s3Location", ws.S3Location)
		ws.Role = iotTwinMakerDefaultString(payload, "role", ws.Role)
		ws.UpdatedAt = now
		return map[string]any{"updateDateTime": ws.UpdatedAt}
	}

	return map[string]any{}
}

func (s *iotTwinMakerStore) ensureWorkspaceLocked(workspaceID string) *iotTwinMakerWorkspace {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		workspaceID = "stackyard-workspace"
	}
	if ws, ok := s.workspaces[workspaceID]; ok {
		return ws
	}
	now := time.Now().UTC()
	ws := &iotTwinMakerWorkspace{
		ID:          workspaceID,
		Arn:         iotTwinMakerWorkspaceARN(workspaceID),
		Description: "stackyard workspace",
		S3Location:  fmt.Sprintf("s3://stackyard-iottwinmaker/workspaces/%s", workspaceID),
		Role:        "arn:aws:iam::123456789012:role/stackyard-iottwinmaker",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.workspaces[workspaceID] = ws
	if _, ok := s.componentTypes[workspaceID]; !ok {
		s.componentTypes[workspaceID] = map[string]*iotTwinMakerComponentType{}
	}
	if _, ok := s.entities[workspaceID]; !ok {
		s.entities[workspaceID] = map[string]*iotTwinMakerEntity{}
	}
	if _, ok := s.scenes[workspaceID]; !ok {
		s.scenes[workspaceID] = map[string]*iotTwinMakerScene{}
	}
	if _, ok := s.syncJobs[workspaceID]; !ok {
		s.syncJobs[workspaceID] = map[string]*iotTwinMakerSyncJob{}
	}
	return ws
}

func (s *iotTwinMakerStore) ensureComponentTypeLocked(workspaceID, componentTypeID string) *iotTwinMakerComponentType {
	ws := s.ensureWorkspaceLocked(workspaceID)
	componentTypeID = strings.TrimSpace(componentTypeID)
	if componentTypeID == "" {
		componentTypeID = fmt.Sprintf("com.stackyard.Component%06d", s.nextLocked())
	}
	bucket := s.componentTypes[ws.ID]
	if ct, ok := bucket[componentTypeID]; ok {
		return ct
	}
	now := time.Now().UTC()
	ct := &iotTwinMakerComponentType{
		WorkspaceID: ws.ID,
		ID:          componentTypeID,
		Arn:         iotTwinMakerComponentTypeARN(ws.ID, componentTypeID),
		Description: "stackyard component type",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	bucket[componentTypeID] = ct
	return ct
}

func (s *iotTwinMakerStore) ensureEntityLocked(workspaceID, entityID string) *iotTwinMakerEntity {
	ws := s.ensureWorkspaceLocked(workspaceID)
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		entityID = fmt.Sprintf("entity-%06d", s.nextLocked())
	}
	bucket := s.entities[ws.ID]
	if entity, ok := bucket[entityID]; ok {
		return entity
	}
	now := time.Now().UTC()
	entity := &iotTwinMakerEntity{
		WorkspaceID: ws.ID,
		ID:          entityID,
		Name:        entityID,
		Arn:         iotTwinMakerEntityARN(ws.ID, entityID),
		Description: "stackyard entity",
		Components:  map[string]map[string]any{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	bucket[entityID] = entity
	return entity
}

func (s *iotTwinMakerStore) ensureSceneLocked(workspaceID, sceneID string) *iotTwinMakerScene {
	ws := s.ensureWorkspaceLocked(workspaceID)
	sceneID = strings.TrimSpace(sceneID)
	if sceneID == "" {
		sceneID = fmt.Sprintf("scene-%06d", s.nextLocked())
	}
	bucket := s.scenes[ws.ID]
	if scene, ok := bucket[sceneID]; ok {
		return scene
	}
	now := time.Now().UTC()
	scene := &iotTwinMakerScene{
		WorkspaceID:     ws.ID,
		ID:              sceneID,
		Arn:             iotTwinMakerSceneARN(ws.ID, sceneID),
		Description:     "stackyard scene",
		ContentLocation: fmt.Sprintf("s3://stackyard-iottwinmaker/scenes/%s.json", sceneID),
		Capabilities:    []string{"public-read"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	bucket[sceneID] = scene
	return scene
}

func (s *iotTwinMakerStore) ensureSyncJobLocked(workspaceID, syncSource string) *iotTwinMakerSyncJob {
	ws := s.ensureWorkspaceLocked(workspaceID)
	syncSource = strings.TrimSpace(syncSource)
	if syncSource == "" {
		syncSource = "SITEWISE"
	}
	bucket := s.syncJobs[ws.ID]
	if job, ok := bucket[syncSource]; ok {
		return job
	}
	now := time.Now().UTC()
	job := &iotTwinMakerSyncJob{
		WorkspaceID: ws.ID,
		SyncSource:  syncSource,
		Arn:         iotTwinMakerSyncJobARN(ws.ID, syncSource),
		Role:        "arn:aws:iam::123456789012:role/stackyard-iottwinmaker-sync",
		State:       "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	bucket[syncSource] = job
	return job
}

func (s *iotTwinMakerStore) findOrCreateSyncJobBySourceLocked(syncSource string) *iotTwinMakerSyncJob {
	syncSource = strings.TrimSpace(syncSource)
	if syncSource == "" {
		syncSource = "SITEWISE"
	}
	for _, jobs := range s.syncJobs {
		if job, ok := jobs[syncSource]; ok {
			return job
		}
	}
	return s.ensureSyncJobLocked("stackyard-workspace", syncSource)
}

func (s *iotTwinMakerStore) ensureMetadataTransferJobLocked(jobID string) *iotTwinMakerMetadataTransferJob {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		jobID = fmt.Sprintf("mtj-%06d", s.nextLocked())
	}
	if job, ok := s.metadataTransferJobs[jobID]; ok {
		return job
	}
	now := time.Now().UTC()
	job := &iotTwinMakerMetadataTransferJob{
		ID:          jobID,
		Arn:         iotTwinMakerMetadataTransferJobARN(jobID),
		Description: "stackyard metadata transfer",
		State:       "PENDING",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.metadataTransferJobs[jobID] = job
	return job
}

func (s *iotTwinMakerStore) nextLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func iotTwinMakerResolveWorkspaceID(payload map[string]any, pathParams map[string]string) string {
	if id := strings.TrimSpace(pathParams["workspaceId"]); id != "" {
		return id
	}
	return iotTwinMakerDefaultString(payload, "workspaceId", "stackyard-workspace")
}

func iotTwinMakerResolveSyncSource(payload map[string]any, pathParams map[string]string) string {
	if source := strings.TrimSpace(pathParams["syncSource"]); source != "" {
		return source
	}
	return iotTwinMakerDefaultString(payload, "syncSource", "SITEWISE")
}

func iotTwinMakerResolveResourceARN(payload map[string]any, query url.Values) string {
	if arn := strings.TrimSpace(iotTwinMakerDefaultString(payload, "resourceARN", "")); arn != "" {
		return arn
	}
	if arn := strings.TrimSpace(iotTwinMakerDefaultString(payload, "resourceArn", "")); arn != "" {
		return arn
	}
	if arn := strings.TrimSpace(query.Get("resourceARN")); arn != "" {
		return arn
	}
	if arn := strings.TrimSpace(query.Get("resourceArn")); arn != "" {
		return arn
	}
	return ""
}

func iotTwinMakerStatus(state string) map[string]any {
	return map[string]any{"state": state, "error": map[string]any{}}
}

func iotTwinMakerSyncJobStatus(state string) map[string]any {
	return map[string]any{"state": state, "error": map[string]any{}}
}

func iotTwinMakerMetadataTransferStatus(state string) map[string]any {
	return map[string]any{"state": state, "error": map[string]any{}, "queuedPosition": 0}
}

func iotTwinMakerMetadataTransferProgress(state string) map[string]any {
	if state == "COMPLETED" {
		return map[string]any{"totalCount": 1, "succeededCount": 1, "skippedCount": 0, "failedCount": 0}
	}
	return map[string]any{"totalCount": 1, "succeededCount": 0, "skippedCount": 0, "failedCount": 0}
}

func iotTwinMakerWorkspaceARN(workspaceID string) string {
	return fmt.Sprintf("arn:aws:iottwinmaker:us-east-1:123456789012:workspace/%s", workspaceID)
}

func iotTwinMakerComponentTypeARN(workspaceID, componentTypeID string) string {
	return fmt.Sprintf("arn:aws:iottwinmaker:us-east-1:123456789012:workspace/%s/component-type/%s", workspaceID, componentTypeID)
}

func iotTwinMakerEntityARN(workspaceID, entityID string) string {
	return fmt.Sprintf("arn:aws:iottwinmaker:us-east-1:123456789012:workspace/%s/entity/%s", workspaceID, entityID)
}

func iotTwinMakerSceneARN(workspaceID, sceneID string) string {
	return fmt.Sprintf("arn:aws:iottwinmaker:us-east-1:123456789012:workspace/%s/scene/%s", workspaceID, sceneID)
}

func iotTwinMakerSyncJobARN(workspaceID, syncSource string) string {
	return fmt.Sprintf("arn:aws:iottwinmaker:us-east-1:123456789012:workspace/%s/sync-job/%s", workspaceID, strings.ToLower(syncSource))
}

func iotTwinMakerMetadataTransferJobARN(jobID string) string {
	return fmt.Sprintf("arn:aws:iottwinmaker:us-east-1:123456789012:metadata-transfer-job/%s", jobID)
}

func iotTwinMakerValue(payload map[string]any, key string) any {
	if payload == nil {
		return nil
	}
	if value, ok := payload[key]; ok {
		return value
	}
	for k, value := range payload {
		if strings.EqualFold(k, key) {
			return value
		}
	}
	return nil
}

func iotTwinMakerDefaultString(payload map[string]any, key, fallback string) string {
	value := iotTwinMakerValue(payload, key)
	text := strings.TrimSpace(iotTwinMakerToString(value))
	if text == "" {
		return fallback
	}
	return text
}

func iotTwinMakerToString(value any) string {
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

func iotTwinMakerExtractTagMap(value any) map[string]string {
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]any:
		for key, val := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(iotTwinMakerToString(val))
		}
	case map[string]string:
		for key, val := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(val)
		}
	}
	return out
}

func iotTwinMakerCloneMapAny(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func iotTwinMakerCloneNestedMap(input map[string]map[string]any) map[string]map[string]any {
	if input == nil {
		return map[string]map[string]any{}
	}
	out := make(map[string]map[string]any, len(input))
	for key, value := range input {
		clone := make(map[string]any, len(value))
		for k, v := range value {
			clone[k] = v
		}
		out[key] = clone
	}
	return out
}

func iotTwinMakerCloneTags(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func iotTwinMakerStringSlice(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			itemStr := strings.TrimSpace(iotTwinMakerToString(item))
			if itemStr != "" {
				out = append(out, itemStr)
			}
		}
		return out
	default:
		only := strings.TrimSpace(iotTwinMakerToString(v))
		if only == "" {
			return nil
		}
		return []string{only}
	}
}

func iotTwinMakerStringSliceFromCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func iotTwinMakerToAnySlice(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}
