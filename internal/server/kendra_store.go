package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type kendraStore struct {
	mu sync.Mutex

	nextID int64

	indexes             map[string]*kendraIndex
	dataSources         map[string]map[string]*kendraResource
	faqs                map[string]map[string]*kendraResource
	thesauri            map[string]map[string]*kendraResource
	accessControls      map[string]map[string]*kendraResource
	queryBlockLists     map[string]*kendraResource
	featuredResultsSets map[string]map[string]*kendraResource
	experiences         map[string]map[string]*kendraResource
	principalMappings   map[string]map[string]any
	tags                map[string]map[string]string
}

type kendraIndex struct {
	ID          string
	Name        string
	Description string
	ARN         string
	Status      string
	CreatedAt   string
	UpdatedAt   string
}

type kendraResource struct {
	ID        string
	IndexID   string
	Name      string
	ARN       string
	Status    string
	CreatedAt string
	UpdatedAt string
}

func newKendraStore() *kendraStore {
	now := time.Now().UTC().Format(time.RFC3339)
	index := &kendraIndex{
		ID:          "idx-000001",
		Name:        "stackyard-kendra-index",
		Description: "Default Stackyard Kendra index",
		ARN:         kendraARN("index", "idx-000001"),
		Status:      "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	dataSource := &kendraResource{
		ID:        "ds-000001",
		IndexID:   index.ID,
		Name:      "stackyard-data-source",
		ARN:       kendraARN("datasource", "ds-000001"),
		Status:    "ACTIVE",
		CreatedAt: now,
		UpdatedAt: now,
	}

	s := &kendraStore{
		nextID: 2,
		indexes: map[string]*kendraIndex{
			index.ID: index,
		},
		dataSources: map[string]map[string]*kendraResource{
			index.ID: {
				dataSource.ID: dataSource,
			},
		},
		faqs:                map[string]map[string]*kendraResource{},
		thesauri:            map[string]map[string]*kendraResource{},
		accessControls:      map[string]map[string]*kendraResource{},
		queryBlockLists:     map[string]*kendraResource{},
		featuredResultsSets: map[string]map[string]*kendraResource{},
		experiences:         map[string]map[string]*kendraResource{},
		principalMappings:   map[string]map[string]any{},
		tags: map[string]map[string]string{
			index.ARN: {
				"stackyard": "true",
			},
			dataSource.ARN: {
				"stackyard": "true",
			},
		},
	}

	return s
}

func (s *kendraStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	indexID := kendraPayloadString(payload, "IndexId", "idx-000001")
	if indexID == "" {
		indexID = "idx-000001"
	}

	s.ensureIndexLocked(indexID)

	switch action {
	case "CreateIndex":
		id := s.nextTokenLocked("idx")
		name := kendraPayloadString(payload, "Name", "stackyard-kendra-index")
		desc := kendraPayloadString(payload, "Description", "")
		index := &kendraIndex{
			ID:          id,
			Name:        name,
			Description: desc,
			ARN:         kendraARN("index", id),
			Status:      "ACTIVE",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		s.indexes[id] = index
		s.ensureTagsLocked(index.ARN)
		return map[string]any{"Id": id}

	case "DescribeIndex":
		id := kendraPayloadString(payload, "Id", indexID)
		index := s.ensureIndexLocked(id)
		return map[string]any{
			"Id":                      index.ID,
			"Name":                    index.Name,
			"Description":             index.Description,
			"Arn":                     index.ARN,
			"Status":                  index.Status,
			"CreatedAt":               index.CreatedAt,
			"UpdatedAt":               index.UpdatedAt,
			"DocumentMetadataConfigs": []any{},
		}

	case "ListIndices":
		summaryItems := make([]any, 0, len(s.indexes))
		for _, index := range s.sortedIndexesLocked() {
			summaryItems = append(summaryItems, map[string]any{
				"Id":        index.ID,
				"Name":      index.Name,
				"Status":    index.Status,
				"UpdatedAt": index.UpdatedAt,
				"CreatedAt": index.CreatedAt,
			})
		}
		return map[string]any{"IndexConfigurationSummaryItems": summaryItems, "NextToken": ""}

	case "UpdateIndex":
		id := kendraPayloadString(payload, "Id", indexID)
		index := s.ensureIndexLocked(id)
		if name := kendraPayloadString(payload, "Name", ""); name != "" {
			index.Name = name
		}
		if desc := kendraPayloadString(payload, "Description", ""); desc != "" {
			index.Description = desc
		}
		index.UpdatedAt = now
		return map[string]any{}

	case "DeleteIndex":
		id := kendraPayloadString(payload, "Id", indexID)
		index := s.ensureIndexLocked(id)
		index.Status = "DELETING"
		index.UpdatedAt = now
		return map[string]any{}

	case "CreateDataSource":
		id := s.nextTokenLocked("ds")
		name := kendraPayloadString(payload, "Name", "stackyard-data-source")
		res := &kendraResource{
			ID:        id,
			IndexID:   indexID,
			Name:      name,
			ARN:       kendraARN("datasource", id),
			Status:    "ACTIVE",
			CreatedAt: now,
			UpdatedAt: now,
		}
		s.ensureBucketLocked(s.dataSources, indexID)
		s.dataSources[indexID][id] = res
		s.ensureTagsLocked(res.ARN)
		return map[string]any{"Id": id}

	case "DescribeDataSource":
		id := kendraPayloadString(payload, "Id", "ds-000001")
		res := s.ensureResourceLocked(s.dataSources, indexID, id, "datasource", "stackyard-data-source")
		return s.describeResourcePayload(res, "DataSourceConfiguration")

	case "ListDataSources":
		return map[string]any{"SummaryItems": s.listResourcesPayload(s.dataSources, indexID), "NextToken": ""}

	case "StartDataSourceSyncJob":
		id := kendraPayloadString(payload, "Id", "ds-000001")
		_ = s.ensureResourceLocked(s.dataSources, indexID, id, "datasource", "stackyard-data-source")
		return map[string]any{"ExecutionId": s.nextTokenLocked("sync")}

	case "StopDataSourceSyncJob":
		return map[string]any{}

	case "ListDataSourceSyncJobs":
		id := kendraPayloadString(payload, "Id", "ds-000001")
		return map[string]any{
			"History": []any{
				map[string]any{
					"ExecutionId":  s.nextTokenLocked("sync"),
					"StartTime":    now,
					"EndTime":      now,
					"Status":       "SUCCEEDED",
					"DataSourceId": id,
				},
			},
			"NextToken": "",
		}

	case "CreateFaq":
		id := s.nextTokenLocked("faq")
		name := kendraPayloadString(payload, "Name", "stackyard-faq")
		res := &kendraResource{ID: id, IndexID: indexID, Name: name, ARN: kendraARN("faq", id), Status: "ACTIVE", CreatedAt: now, UpdatedAt: now}
		s.ensureBucketLocked(s.faqs, indexID)
		s.faqs[indexID][id] = res
		s.ensureTagsLocked(res.ARN)
		return map[string]any{"Id": id}

	case "DescribeFaq":
		id := kendraPayloadString(payload, "Id", "faq-000001")
		res := s.ensureResourceLocked(s.faqs, indexID, id, "faq", "stackyard-faq")
		return s.describeResourcePayload(res, "S3Path")

	case "ListFaqs":
		return map[string]any{"FaqSummaryItems": s.listResourcesPayload(s.faqs, indexID), "NextToken": ""}

	case "UpdateFaq", "DeleteFaq":
		id := kendraPayloadString(payload, "Id", "faq-000001")
		res := s.ensureResourceLocked(s.faqs, indexID, id, "faq", "stackyard-faq")
		if action == "DeleteFaq" {
			res.Status = "DELETING"
		} else {
			res.UpdatedAt = now
		}
		return map[string]any{}

	case "CreateThesaurus":
		id := s.nextTokenLocked("thes")
		name := kendraPayloadString(payload, "Name", "stackyard-thesaurus")
		res := &kendraResource{ID: id, IndexID: indexID, Name: name, ARN: kendraARN("thesaurus", id), Status: "ACTIVE", CreatedAt: now, UpdatedAt: now}
		s.ensureBucketLocked(s.thesauri, indexID)
		s.thesauri[indexID][id] = res
		s.ensureTagsLocked(res.ARN)
		return map[string]any{"Id": id}

	case "DescribeThesaurus":
		id := kendraPayloadString(payload, "Id", "thes-000001")
		res := s.ensureResourceLocked(s.thesauri, indexID, id, "thesaurus", "stackyard-thesaurus")
		return s.describeResourcePayload(res, "SourceS3Path")

	case "ListThesauri":
		return map[string]any{"ThesaurusSummaryItems": s.listResourcesPayload(s.thesauri, indexID), "NextToken": ""}

	case "UpdateThesaurus", "DeleteThesaurus":
		id := kendraPayloadString(payload, "Id", "thes-000001")
		res := s.ensureResourceLocked(s.thesauri, indexID, id, "thesaurus", "stackyard-thesaurus")
		if action == "DeleteThesaurus" {
			res.Status = "DELETING"
		} else {
			res.UpdatedAt = now
		}
		return map[string]any{}

	case "CreateAccessControlConfiguration":
		id := s.nextTokenLocked("acc")
		name := kendraPayloadString(payload, "Name", "stackyard-access-control")
		res := &kendraResource{ID: id, IndexID: indexID, Name: name, ARN: kendraARN("access-control-configuration", id), Status: "ACTIVE", CreatedAt: now, UpdatedAt: now}
		s.ensureBucketLocked(s.accessControls, indexID)
		s.accessControls[indexID][id] = res
		return map[string]any{"Id": id}

	case "DescribeAccessControlConfiguration":
		id := kendraPayloadString(payload, "Id", "acc-000001")
		res := s.ensureResourceLocked(s.accessControls, indexID, id, "access-control-configuration", "stackyard-access-control")
		return map[string]any{"Name": res.Name, "Id": res.ID, "IndexId": res.IndexID, "HierarchicalAccessControlList": []any{}}

	case "ListAccessControlConfigurations":
		return map[string]any{"AccessControlConfigurations": s.listResourcesPayload(s.accessControls, indexID), "NextToken": ""}

	case "UpdateAccessControlConfiguration", "DeleteAccessControlConfiguration":
		id := kendraPayloadString(payload, "Id", "acc-000001")
		res := s.ensureResourceLocked(s.accessControls, indexID, id, "access-control-configuration", "stackyard-access-control")
		if action == "DeleteAccessControlConfiguration" {
			res.Status = "DELETING"
		}
		res.UpdatedAt = now
		return map[string]any{}

	case "CreateFeaturedResultsSet":
		id := s.nextTokenLocked("frs")
		name := kendraPayloadString(payload, "FeaturedResultsSetName", "stackyard-featured-results")
		res := &kendraResource{ID: id, IndexID: indexID, Name: name, ARN: kendraARN("featured-results-set", id), Status: "ACTIVE", CreatedAt: now, UpdatedAt: now}
		s.ensureBucketLocked(s.featuredResultsSets, indexID)
		s.featuredResultsSets[indexID][id] = res
		return map[string]any{"FeaturedResultsSet": map[string]any{"FeaturedResultsSetId": id, "FeaturedResultsSetName": name}}

	case "DescribeFeaturedResultsSet":
		id := kendraPayloadString(payload, "FeaturedResultsSetId", "frs-000001")
		res := s.ensureResourceLocked(s.featuredResultsSets, indexID, id, "featured-results-set", "stackyard-featured-results")
		return map[string]any{"FeaturedResultsSet": map[string]any{"FeaturedResultsSetId": res.ID, "FeaturedResultsSetName": res.Name, "Status": res.Status}}

	case "ListFeaturedResultsSets":
		return map[string]any{"FeaturedResultsSetSummaryItems": s.listResourcesPayload(s.featuredResultsSets, indexID), "NextToken": ""}

	case "UpdateFeaturedResultsSet", "BatchDeleteFeaturedResultsSet":
		id := kendraPayloadString(payload, "FeaturedResultsSetId", "frs-000001")
		res := s.ensureResourceLocked(s.featuredResultsSets, indexID, id, "featured-results-set", "stackyard-featured-results")
		res.UpdatedAt = now
		return map[string]any{"Errors": []any{}}

	case "CreateExperience":
		id := s.nextTokenLocked("exp")
		name := kendraPayloadString(payload, "Name", "stackyard-experience")
		res := &kendraResource{ID: id, IndexID: indexID, Name: name, ARN: kendraARN("experience", id), Status: "ACTIVE", CreatedAt: now, UpdatedAt: now}
		s.ensureBucketLocked(s.experiences, indexID)
		s.experiences[indexID][id] = res
		return map[string]any{"Id": id}

	case "DescribeExperience":
		id := kendraPayloadString(payload, "Id", "exp-000001")
		res := s.ensureResourceLocked(s.experiences, indexID, id, "experience", "stackyard-experience")
		return map[string]any{"Id": res.ID, "IndexId": res.IndexID, "Name": res.Name, "Status": res.Status, "Endpoint": "https://stackyard.local/kendra/experience"}

	case "ListExperiences":
		return map[string]any{"SummaryItems": s.listResourcesPayload(s.experiences, indexID), "NextToken": ""}

	case "UpdateExperience", "DeleteExperience":
		id := kendraPayloadString(payload, "Id", "exp-000001")
		res := s.ensureResourceLocked(s.experiences, indexID, id, "experience", "stackyard-experience")
		if action == "DeleteExperience" {
			res.Status = "DELETING"
		}
		res.UpdatedAt = now
		return map[string]any{}

	case "AssociateEntitiesToExperience", "DisassociateEntitiesFromExperience", "AssociatePersonasToEntities", "DisassociatePersonasFromEntities":
		return map[string]any{"FailedEntityList": []any{}}

	case "ListExperienceEntities", "ListEntityPersonas":
		return map[string]any{"SummaryItems": []any{}, "NextToken": ""}

	case "BatchPutDocument", "BatchDeleteDocument":
		return map[string]any{"FailedDocuments": []any{}}

	case "BatchGetDocumentStatus":
		statusItems := []any{}
		if docsRaw, ok := kendraPayloadValue(payload, "DocumentInfoList"); ok {
			if docs, ok := docsRaw.([]any); ok {
				for _, item := range docs {
					docID := "doc-000001"
					if m, ok := item.(map[string]any); ok {
						docID = kendraPayloadString(m, "DocumentId", docID)
					}
					statusItems = append(statusItems, map[string]any{"DocumentId": docID, "Status": "SUCCEEDED"})
				}
			}
		}
		return map[string]any{"DocumentStatusList": statusItems, "Errors": []any{}}

	case "PutPrincipalMapping":
		key := s.principalMappingKey(payload)
		s.principalMappings[key] = map[string]any{
			"DataSourceId": kendraPayloadString(payload, "DataSourceId", "ds-000001"),
			"GroupId":      kendraPayloadString(payload, "GroupId", "group-000001"),
			"IndexId":      indexID,
			"OrderingId":   s.nextTokenLocked("ord"),
		}
		return map[string]any{}

	case "DescribePrincipalMapping":
		key := s.principalMappingKey(payload)
		record, ok := s.principalMappings[key]
		if !ok {
			record = map[string]any{"DataSourceId": "ds-000001", "GroupId": "group-000001", "IndexId": indexID, "OrderingId": "ord-000001"}
		}
		return map[string]any{"GroupId": record["GroupId"], "DataSourceId": record["DataSourceId"], "IndexId": record["IndexId"], "GroupMembers": []any{}, "OrderingIdSummary": map[string]any{"ReceivedOrderingId": record["OrderingId"], "LastUpdatedAt": now}}

	case "DeletePrincipalMapping":
		delete(s.principalMappings, s.principalMappingKey(payload))
		return map[string]any{}

	case "ListGroupsOlderThanOrderingId":
		return map[string]any{"GroupsSummaries": []any{}, "NextToken": ""}

	case "Query":
		result := map[string]any{
			"Id":              "result-000001",
			"Type":            "DOCUMENT",
			"DocumentId":      "doc-000001",
			"DocumentURI":     "https://example.com/stackyard-kendra",
			"ScoreAttributes": map[string]any{"ScoreConfidence": "HIGH"},
			"DocumentTitle":   map[string]any{"Text": "Stackyard Kendra Result", "Highlights": []any{}},
			"DocumentExcerpt": map[string]any{"Text": "Stackyard local emulation result.", "Highlights": []any{}},
		}
		return map[string]any{"QueryId": s.nextTokenLocked("query"), "ResultItems": []any{result}, "TotalNumberOfResults": 1}

	case "Retrieve":
		item := map[string]any{
			"Id":            "result-000001",
			"DocumentId":    "doc-000001",
			"DocumentTitle": map[string]any{"Text": "Stackyard Retrieved Result", "Highlights": []any{}},
			"Content":       "Stackyard retrieve response",
			"DocumentURI":   "https://example.com/stackyard-kendra",
		}
		return map[string]any{"QueryId": s.nextTokenLocked("query"), "ResultItems": []any{item}}

	case "GetQuerySuggestions":
		return map[string]any{"Suggestions": []any{map[string]any{"Value": map[string]any{"Text": "stackyard kendra"}, "SourceDocuments": []any{}}}}

	case "CreateQuerySuggestionsBlockList":
		id := s.nextTokenLocked("qsb")
		name := kendraPayloadString(payload, "Name", "stackyard-block-list")
		res := &kendraResource{ID: id, Name: name, ARN: kendraARN("query-suggestions-block-list", id), Status: "ACTIVE", CreatedAt: now, UpdatedAt: now}
		s.queryBlockLists[id] = res
		return map[string]any{"Id": id}

	case "DescribeQuerySuggestionsBlockList":
		id := kendraPayloadString(payload, "Id", "qsb-000001")
		res := s.ensureStandaloneResourceLocked(s.queryBlockLists, id, "query-suggestions-block-list", "stackyard-block-list")
		return map[string]any{"Id": res.ID, "Name": res.Name, "Status": res.Status, "UpdatedAt": res.UpdatedAt}

	case "ListQuerySuggestionsBlockLists":
		items := make([]any, 0, len(s.queryBlockLists))
		for _, res := range s.sortedStandaloneResourcesLocked(s.queryBlockLists) {
			items = append(items, map[string]any{"Id": res.ID, "Name": res.Name, "Status": res.Status, "UpdatedAt": res.UpdatedAt})
		}
		return map[string]any{"BlockListSummaryItems": items, "NextToken": ""}

	case "UpdateQuerySuggestionsBlockList", "DeleteQuerySuggestionsBlockList", "ClearQuerySuggestions":
		if action != "ClearQuerySuggestions" {
			id := kendraPayloadString(payload, "Id", "qsb-000001")
			res := s.ensureStandaloneResourceLocked(s.queryBlockLists, id, "query-suggestions-block-list", "stackyard-block-list")
			if action == "DeleteQuerySuggestionsBlockList" {
				res.Status = "DELETING"
			}
			res.UpdatedAt = now
		}
		return map[string]any{}

	case "DescribeQuerySuggestionsConfig":
		return map[string]any{"Mode": "ENABLED", "Status": "ACTIVE", "AttributeSuggestionsConfig": map[string]any{"SuggestableConfigList": []any{}}, "LastUpdatedAt": now}

	case "UpdateQuerySuggestionsConfig", "SubmitFeedback":
		return map[string]any{}

	case "TagResource":
		resourceARN := kendraPayloadString(payload, "ResourceARN", "")
		if resourceARN == "" {
			resourceARN = kendraPayloadString(payload, "ResourceArn", "")
		}
		if resourceARN == "" {
			resourceARN = s.ensureIndexLocked(indexID).ARN
		}
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range kendraPayloadTags(payload) {
			tags[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := kendraPayloadString(payload, "ResourceARN", "")
		if resourceARN == "" {
			resourceARN = kendraPayloadString(payload, "ResourceArn", "")
		}
		if resourceARN == "" {
			resourceARN = s.ensureIndexLocked(indexID).ARN
		}
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range kendraPayloadStringSlice(payload, "TagKeys") {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := kendraPayloadString(payload, "ResourceARN", "")
		if resourceARN == "" {
			resourceARN = kendraPayloadString(payload, "ResourceArn", "")
		}
		if resourceARN == "" {
			resourceARN = s.ensureIndexLocked(indexID).ARN
		}
		return map[string]any{"Tags": kendraTagList(s.ensureTagsLocked(resourceARN))}

	case "GetSnapshots":
		return map[string]any{"SnapshotsDataHeader": map[string]any{"SnapshotStatus": "AVAILABLE", "IndexId": indexID}, "SnapshotsDataRecord": []any{}}
	}

	switch {
	case strings.HasPrefix(action, "List"):
		return map[string]any{"SummaryItems": []any{}, "NextToken": ""}
	case strings.HasPrefix(action, "Describe"), strings.HasPrefix(action, "Get"):
		return map[string]any{"Action": action, "Status": "ACTIVE", "UpdatedAt": now}
	case strings.HasPrefix(action, "Create"), strings.HasPrefix(action, "Start"), strings.HasPrefix(action, "Update"), strings.HasPrefix(action, "Put"), strings.HasPrefix(action, "Associate"), strings.HasPrefix(action, "Disassociate"), strings.HasPrefix(action, "Batch"):
		return map[string]any{"Action": action, "Status": "OK", "RequestId": s.nextTokenLocked("req")}
	case strings.HasPrefix(action, "Delete"), strings.HasPrefix(action, "Stop"), strings.HasPrefix(action, "Untag"), strings.HasPrefix(action, "Tag"), strings.HasPrefix(action, "Clear"), strings.HasPrefix(action, "Submit"):
		return map[string]any{}
	}

	return map[string]any{"Action": action, "Status": "OK"}
}

func (s *kendraStore) ensureIndexLocked(indexID string) *kendraIndex {
	id := strings.TrimSpace(indexID)
	if id == "" {
		id = "idx-000001"
	}
	if existing, ok := s.indexes[id]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	index := &kendraIndex{
		ID:          id,
		Name:        "stackyard-kendra-index",
		Description: "Stackyard generated index",
		ARN:         kendraARN("index", id),
		Status:      "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.indexes[id] = index
	s.ensureTagsLocked(index.ARN)
	return index
}

func (s *kendraStore) ensureResourceLocked(collection map[string]map[string]*kendraResource, indexID, resourceID, resourceType, defaultName string) *kendraResource {
	index := s.ensureIndexLocked(indexID)
	id := strings.TrimSpace(resourceID)
	if id == "" {
		id = s.nextTokenLocked(resourceType)
	}
	s.ensureBucketLocked(collection, index.ID)
	if existing, ok := collection[index.ID][id]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	resource := &kendraResource{
		ID:        id,
		IndexID:   index.ID,
		Name:      defaultName,
		ARN:       kendraARN(resourceType, id),
		Status:    "ACTIVE",
		CreatedAt: now,
		UpdatedAt: now,
	}
	collection[index.ID][id] = resource
	s.ensureTagsLocked(resource.ARN)
	return resource
}

func (s *kendraStore) ensureStandaloneResourceLocked(collection map[string]*kendraResource, resourceID, resourceType, defaultName string) *kendraResource {
	id := strings.TrimSpace(resourceID)
	if id == "" {
		id = s.nextTokenLocked(resourceType)
	}
	if existing, ok := collection[id]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	resource := &kendraResource{ID: id, Name: defaultName, ARN: kendraARN(resourceType, id), Status: "ACTIVE", CreatedAt: now, UpdatedAt: now}
	collection[id] = resource
	s.ensureTagsLocked(resource.ARN)
	return resource
}

func (s *kendraStore) ensureBucketLocked(collection map[string]map[string]*kendraResource, indexID string) {
	if _, ok := collection[indexID]; !ok {
		collection[indexID] = map[string]*kendraResource{}
	}
}

func (s *kendraStore) describeResourcePayload(resource *kendraResource, configField string) map[string]any {
	payload := map[string]any{
		"Id":        resource.ID,
		"IndexId":   resource.IndexID,
		"Name":      resource.Name,
		"Arn":       resource.ARN,
		"Status":    resource.Status,
		"CreatedAt": resource.CreatedAt,
		"UpdatedAt": resource.UpdatedAt,
	}
	if strings.TrimSpace(configField) != "" {
		payload[configField] = map[string]any{}
	}
	return payload
}

func (s *kendraStore) listResourcesPayload(collection map[string]map[string]*kendraResource, indexID string) []any {
	items := []any{}
	bucket, ok := collection[indexID]
	if !ok {
		return items
	}
	keys := make([]string, 0, len(bucket))
	for id := range bucket {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	for _, id := range keys {
		res := bucket[id]
		items = append(items, map[string]any{
			"Id":        res.ID,
			"Name":      res.Name,
			"Status":    res.Status,
			"UpdatedAt": res.UpdatedAt,
			"CreatedAt": res.CreatedAt,
		})
	}
	return items
}

func (s *kendraStore) sortedIndexesLocked() []*kendraIndex {
	keys := make([]string, 0, len(s.indexes))
	for id := range s.indexes {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]*kendraIndex, 0, len(keys))
	for _, id := range keys {
		out = append(out, s.indexes[id])
	}
	return out
}

func (s *kendraStore) sortedStandaloneResourcesLocked(collection map[string]*kendraResource) []*kendraResource {
	keys := make([]string, 0, len(collection))
	for id := range collection {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]*kendraResource, 0, len(keys))
	for _, id := range keys {
		out = append(out, collection[id])
	}
	return out
}

func (s *kendraStore) ensureTagsLocked(arn string) map[string]string {
	key := strings.TrimSpace(arn)
	if key == "" {
		key = kendraARN("index", "idx-000001")
	}
	if tags, ok := s.tags[key]; ok {
		return tags
	}
	s.tags[key] = map[string]string{}
	return s.tags[key]
}

func (s *kendraStore) principalMappingKey(payload map[string]any) string {
	return strings.Join([]string{
		kendraPayloadString(payload, "IndexId", "idx-000001"),
		kendraPayloadString(payload, "DataSourceId", "ds-000001"),
		kendraPayloadString(payload, "GroupId", "group-000001"),
	}, "|")
}

func (s *kendraStore) nextTokenLocked(prefix string) string {
	id := s.nextID
	s.nextID++
	p := strings.TrimSpace(prefix)
	if p == "" {
		p = "res"
	}
	return fmt.Sprintf("%s-%06d", p, id)
}

func kendraARN(resourceType, id string) string {
	resource := strings.TrimSpace(resourceType)
	if resource == "" {
		resource = "resource"
	}
	value := strings.TrimSpace(id)
	if value == "" {
		value = "stackyard"
	}
	if strings.HasPrefix(value, "arn:") {
		return value
	}
	return fmt.Sprintf("arn:aws:kendra:us-east-1:123456789012:%s/%s", resource, value)
}

func kendraPayloadValue(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func kendraPayloadString(payload map[string]any, key, fallback string) string {
	value, ok := kendraPayloadValue(payload, key)
	if !ok {
		return fallback
	}
	text := strings.TrimSpace(fmt.Sprintf("%v", value))
	if text == "" {
		return fallback
	}
	return text
}

func kendraPayloadStringSlice(payload map[string]any, key string) []string {
	value, ok := kendraPayloadValue(payload, key)
	if !ok {
		return nil
	}
	list, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]string); ok {
			return typed
		}
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		text := strings.TrimSpace(fmt.Sprintf("%v", item))
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func kendraPayloadTags(payload map[string]any) map[string]string {
	value, ok := kendraPayloadValue(payload, "Tags")
	if !ok {
		return map[string]string{}
	}

	if rawMap, ok := value.(map[string]any); ok {
		out := make(map[string]string, len(rawMap))
		for k, v := range rawMap {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		return out
	}

	if rawList, ok := value.([]any); ok {
		out := map[string]string{}
		for _, item := range rawList {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := strings.TrimSpace(fmt.Sprintf("%v", entry["Key"]))
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprintf("%v", entry["Value"]))
		}
		return out
	}

	return map[string]string{}
}

func kendraTagList(tags map[string]string) []any {
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
