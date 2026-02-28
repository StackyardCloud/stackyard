package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type healthImagingStore struct {
	mu         sync.Mutex
	next       int64
	datastores map[string]*healthImagingDatastore
	importJobs map[string]*healthImagingImportJob
	imageSets  map[string]*healthImagingImageSet
	tags       map[string]map[string]string
}

type healthImagingDatastore struct {
	ID        string
	Name      string
	Arn       string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type healthImagingImportJob struct {
	ID          string
	DatastoreID string
	Arn         string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type healthImagingImageSet struct {
	DatastoreID string
	ID          string
	VersionID   string
	Arn         string
	State       string
	Metadata    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func newHealthImagingStore() *healthImagingStore {
	now := time.Now().UTC()

	ds := &healthImagingDatastore{
		ID:        "stackyard-datastore",
		Name:      "stackyard",
		Arn:       healthImagingDatastoreARN("stackyard-datastore"),
		Status:    "ACTIVE",
		CreatedAt: now,
		UpdatedAt: now,
	}
	job := &healthImagingImportJob{
		ID:          "job-000001",
		DatastoreID: ds.ID,
		Arn:         healthImagingDICOMImportJobARN(ds.ID, "job-000001"),
		Status:      "COMPLETED",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	imageSet := &healthImagingImageSet{
		DatastoreID: ds.ID,
		ID:          "imageset-000001",
		VersionID:   "1",
		Arn:         healthImagingImageSetARN(ds.ID, "imageset-000001"),
		State:       "ACTIVE",
		Metadata:    `{"PatientID":"stackyard"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return &healthImagingStore{
		next: 2,
		datastores: map[string]*healthImagingDatastore{
			ds.ID: ds,
		},
		importJobs: map[string]*healthImagingImportJob{
			job.ID: job,
		},
		imageSets: map[string]*healthImagingImageSet{
			healthImagingImageSetKey(ds.ID, imageSet.ID): imageSet,
		},
		tags: map[string]map[string]string{
			ds.Arn:       {"seed": "true"},
			imageSet.Arn: {"seed": "true"},
		},
	}
}

func (s *healthImagingStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "CreateDatastore":
		name := healthImagingDefaultString(payload, "datastoreName", fmt.Sprintf("stackyard-datastore-%06d", s.nextID()))
		id := healthImagingDefaultString(payload, "datastoreId", fmt.Sprintf("datastore-%06d", s.nextID()))
		ds := s.ensureDatastore(id)
		ds.Name = name
		ds.Status = "CREATING"
		ds.UpdatedAt = now
		ds.Status = "ACTIVE"
		return map[string]any{
			"datastoreId":     ds.ID,
			"datastoreName":   ds.Name,
			"datastoreArn":    ds.Arn,
			"datastoreStatus": ds.Status,
		}

	case "DeleteDatastore":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		if ds, ok := s.datastores[datastoreID]; ok {
			ds.Status = "DELETING"
			ds.UpdatedAt = now
			delete(s.datastores, datastoreID)
		}
		for key, imageSet := range s.imageSets {
			if imageSet.DatastoreID == datastoreID {
				delete(s.imageSets, key)
			}
		}
		for key, job := range s.importJobs {
			if job.DatastoreID == datastoreID {
				delete(s.importJobs, key)
			}
		}
		return map[string]any{"datastoreId": datastoreID, "datastoreStatus": "DELETING"}

	case "GetDatastore":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		ds := s.ensureDatastore(datastoreID)
		props := healthImagingDatastorePayload(ds)
		return map[string]any{
			"datastoreId":         ds.ID,
			"datastoreName":       ds.Name,
			"datastoreArn":        ds.Arn,
			"datastoreStatus":     ds.Status,
			"datastoreProperties": props,
		}

	case "ListDatastores":
		ids := make([]string, 0, len(s.datastores))
		for id := range s.datastores {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		items := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			items = append(items, healthImagingDatastorePayload(s.datastores[id]))
		}
		return map[string]any{
			"datastorePropertiesList": items,
			"nextToken":               "",
		}

	case "StartDICOMImportJob":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		s.ensureDatastore(datastoreID)
		jobID := healthImagingDefaultString(payload, "jobId", fmt.Sprintf("job-%06d", s.nextID()))
		job := &healthImagingImportJob{
			ID:          jobID,
			DatastoreID: datastoreID,
			Arn:         healthImagingDICOMImportJobARN(datastoreID, jobID),
			Status:      "SUBMITTED",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		s.importJobs[job.ID] = job
		return map[string]any{
			"jobId":       job.ID,
			"jobStatus":   job.Status,
			"datastoreId": datastoreID,
			"jobArn":      job.Arn,
		}

	case "GetDICOMImportJob":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		jobID := healthImagingPathValue(pathParams, "jobId", healthImagingDefaultString(payload, "jobId", "job-000001"))
		job := s.ensureImportJob(datastoreID, jobID)
		props := healthImagingImportJobPayload(job)
		return map[string]any{
			"jobProperties": props,
			"jobId":         job.ID,
			"jobStatus":     job.Status,
		}

	case "ListDICOMImportJobs":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		s.ensureDatastore(datastoreID)
		items := make([]map[string]any, 0)
		for _, job := range s.importJobs {
			if job.DatastoreID != datastoreID {
				continue
			}
			items = append(items, healthImagingImportJobPayload(job))
		}
		sort.Slice(items, func(i, j int) bool {
			return healthImagingDefaultString(items[i], "jobId", "") < healthImagingDefaultString(items[j], "jobId", "")
		})
		return map[string]any{
			"jobSummaries": items,
			"nextToken":    "",
		}

	case "GetImageSet":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		imageSetID := healthImagingPathValue(pathParams, "imageSetId", healthImagingDefaultString(payload, "imageSetId", "imageset-000001"))
		imageSet := s.ensureImageSet(datastoreID, imageSetID)
		return map[string]any{
			"imageSetId":         imageSet.ID,
			"datastoreId":        imageSet.DatastoreID,
			"versionId":          imageSet.VersionID,
			"imageSetState":      imageSet.State,
			"imageSetArn":        imageSet.Arn,
			"imageSetProperties": healthImagingImageSetPayload(imageSet),
			"updatedAt":          imageSet.UpdatedAt,
			"createdAt":          imageSet.CreatedAt,
		}

	case "GetImageSetMetadata":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		imageSetID := healthImagingPathValue(pathParams, "imageSetId", healthImagingDefaultString(payload, "imageSetId", "imageset-000001"))
		imageSet := s.ensureImageSet(datastoreID, imageSetID)
		return map[string]any{
			"imageSetId":           imageSet.ID,
			"datastoreId":          imageSet.DatastoreID,
			"versionId":            imageSet.VersionID,
			"imageSetMetadataBlob": imageSet.Metadata,
			"contentEncoding":      "JSON",
		}

	case "GetImageFrame":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		imageSetID := healthImagingPathValue(pathParams, "imageSetId", healthImagingDefaultString(payload, "imageSetId", "imageset-000001"))
		imageSet := s.ensureImageSet(datastoreID, imageSetID)
		return map[string]any{
			"imageSetId":      imageSet.ID,
			"datastoreId":     imageSet.DatastoreID,
			"versionId":       imageSet.VersionID,
			"imageFrameBlob":  "c3RhY2t5YXJk",
			"contentEncoding": "BASE64",
		}

	case "ListImageSetVersions":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		imageSetID := healthImagingPathValue(pathParams, "imageSetId", healthImagingDefaultString(payload, "imageSetId", "imageset-000001"))
		imageSet := s.ensureImageSet(datastoreID, imageSetID)
		return map[string]any{
			"imageSetPropertiesList": []any{healthImagingImageSetPayload(imageSet)},
			"nextToken":              "",
		}

	case "SearchImageSets":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		items := make([]map[string]any, 0)
		for _, imageSet := range s.imageSets {
			if imageSet.DatastoreID != datastoreID {
				continue
			}
			items = append(items, map[string]any{
				"imageSetId":  imageSet.ID,
				"versionId":   imageSet.VersionID,
				"updatedAt":   imageSet.UpdatedAt,
				"createdAt":   imageSet.CreatedAt,
				"datastoreId": imageSet.DatastoreID,
			})
		}
		sort.Slice(items, func(i, j int) bool {
			return healthImagingDefaultString(items[i], "imageSetId", "") < healthImagingDefaultString(items[j], "imageSetId", "")
		})
		return map[string]any{
			"imageSetsMetadataSummaries": items,
			"nextToken":                  "",
		}

	case "CopyImageSet":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		sourceImageSetID := healthImagingPathValue(pathParams, "sourceImageSetId", healthImagingDefaultString(payload, "sourceImageSetId", "imageset-000001"))
		source := s.ensureImageSet(datastoreID, sourceImageSetID)
		destinationImageSetID := healthImagingDefaultString(
			healthImagingMapValue(payload, "destinationImageSet"),
			"destinationImageSetId",
			fmt.Sprintf("imageset-%06d", s.nextID()),
		)
		destination := s.ensureImageSet(datastoreID, destinationImageSetID)
		destination.Metadata = source.Metadata
		destination.VersionID = fmt.Sprintf("%d", s.nextID())
		destination.UpdatedAt = now
		return map[string]any{
			"datastoreId":                   datastoreID,
			"sourceImageSetId":              source.ID,
			"destinationImageSetId":         destination.ID,
			"sourceImageSetProperties":      healthImagingImageSetPayload(source),
			"destinationImageSetProperties": healthImagingImageSetPayload(destination),
		}

	case "DeleteImageSet":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		imageSetID := healthImagingPathValue(pathParams, "imageSetId", healthImagingDefaultString(payload, "imageSetId", "imageset-000001"))
		key := healthImagingImageSetKey(datastoreID, imageSetID)
		if imageSet, ok := s.imageSets[key]; ok {
			imageSet.State = "DELETED"
			imageSet.UpdatedAt = now
			delete(s.imageSets, key)
		}
		return map[string]any{"datastoreId": datastoreID, "imageSetId": imageSetID, "imageSetState": "DELETED"}

	case "UpdateImageSetMetadata":
		datastoreID := healthImagingPathValue(pathParams, "datastoreId", healthImagingDefaultString(payload, "datastoreId", "stackyard-datastore"))
		imageSetID := healthImagingPathValue(pathParams, "imageSetId", healthImagingDefaultString(payload, "imageSetId", "imageset-000001"))
		imageSet := s.ensureImageSet(datastoreID, imageSetID)
		metadataUpdates := healthImagingMapValue(payload, "metadataUpdates")
		if len(metadataUpdates) > 0 {
			imageSet.Metadata = healthImagingDefaultString(metadataUpdates, "DICOMUpdates", imageSet.Metadata)
		}
		imageSet.VersionID = fmt.Sprintf("%d", s.nextID())
		imageSet.UpdatedAt = now
		return map[string]any{
			"datastoreId":        datastoreID,
			"imageSetId":         imageSet.ID,
			"versionId":          imageSet.VersionID,
			"updatedAt":          imageSet.UpdatedAt,
			"imageSetProperties": healthImagingImageSetPayload(imageSet),
		}

	case "TagResource":
		resourceARN := healthImagingPathValue(pathParams, "resourceArn", healthImagingDefaultString(payload, "resourceArn", ""))
		if strings.TrimSpace(resourceARN) == "" {
			resourceARN = healthImagingDatastoreARN("stackyard-datastore")
		}
		current := s.tags[resourceARN]
		if current == nil {
			current = map[string]string{}
		}
		for k, v := range healthImagingExtractTagMap(healthImagingValue(payload, "tags")) {
			current[k] = v
		}
		s.tags[resourceARN] = current
		return map[string]any{}

	case "UntagResource":
		resourceARN := healthImagingPathValue(pathParams, "resourceArn", healthImagingDefaultString(payload, "resourceArn", ""))
		if strings.TrimSpace(resourceARN) == "" {
			resourceARN = healthImagingDatastoreARN("stackyard-datastore")
		}
		current := s.tags[resourceARN]
		if current == nil {
			current = map[string]string{}
		}
		for _, key := range healthImagingStringSlice(healthImagingValue(payload, "tagKeys")) {
			delete(current, key)
		}
		for _, key := range query["tagKeys"] {
			if trimmed := strings.TrimSpace(key); trimmed != "" {
				delete(current, trimmed)
			}
		}
		s.tags[resourceARN] = current
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := healthImagingPathValue(pathParams, "resourceArn", healthImagingDefaultString(payload, "resourceArn", ""))
		if strings.TrimSpace(resourceARN) == "" {
			resourceARN = healthImagingDatastoreARN("stackyard-datastore")
		}
		return map[string]any{"tags": healthImagingCloneStringMap(s.tags[resourceARN])}
	}

	return map[string]any{}
}

func (s *healthImagingStore) nextID() int64 {
	s.next++
	return s.next
}

func (s *healthImagingStore) ensureDatastore(id string) *healthImagingDatastore {
	if id == "" {
		id = "stackyard-datastore"
	}
	if ds, ok := s.datastores[id]; ok {
		return ds
	}
	now := time.Now().UTC()
	ds := &healthImagingDatastore{
		ID:        id,
		Name:      id,
		Arn:       healthImagingDatastoreARN(id),
		Status:    "ACTIVE",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.datastores[id] = ds
	return ds
}

func (s *healthImagingStore) ensureImportJob(datastoreID, jobID string) *healthImagingImportJob {
	if datastoreID == "" {
		datastoreID = "stackyard-datastore"
	}
	if jobID == "" {
		jobID = "job-000001"
	}
	if job, ok := s.importJobs[jobID]; ok {
		return job
	}
	now := time.Now().UTC()
	job := &healthImagingImportJob{
		ID:          jobID,
		DatastoreID: datastoreID,
		Arn:         healthImagingDICOMImportJobARN(datastoreID, jobID),
		Status:      "COMPLETED",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.importJobs[jobID] = job
	return job
}

func (s *healthImagingStore) ensureImageSet(datastoreID, imageSetID string) *healthImagingImageSet {
	if datastoreID == "" {
		datastoreID = "stackyard-datastore"
	}
	if imageSetID == "" {
		imageSetID = "imageset-000001"
	}
	key := healthImagingImageSetKey(datastoreID, imageSetID)
	if imageSet, ok := s.imageSets[key]; ok {
		return imageSet
	}
	now := time.Now().UTC()
	imageSet := &healthImagingImageSet{
		DatastoreID: datastoreID,
		ID:          imageSetID,
		VersionID:   "1",
		Arn:         healthImagingImageSetARN(datastoreID, imageSetID),
		State:       "ACTIVE",
		Metadata:    `{"PatientID":"stackyard"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.imageSets[key] = imageSet
	return imageSet
}

func healthImagingDatastoreARN(datastoreID string) string {
	return fmt.Sprintf("arn:aws:medical-imaging:us-east-1:123456789012:datastore/%s", datastoreID)
}

func healthImagingDICOMImportJobARN(datastoreID, jobID string) string {
	return fmt.Sprintf("arn:aws:medical-imaging:us-east-1:123456789012:datastore/%s/job/%s", datastoreID, jobID)
}

func healthImagingImageSetARN(datastoreID, imageSetID string) string {
	return fmt.Sprintf("arn:aws:medical-imaging:us-east-1:123456789012:datastore/%s/imageSet/%s", datastoreID, imageSetID)
}

func healthImagingImageSetKey(datastoreID, imageSetID string) string {
	return datastoreID + "::" + imageSetID
}

func healthImagingDatastorePayload(ds *healthImagingDatastore) map[string]any {
	return map[string]any{
		"datastoreId":     ds.ID,
		"datastoreName":   ds.Name,
		"datastoreArn":    ds.Arn,
		"datastoreStatus": ds.Status,
		"createdAt":       ds.CreatedAt,
		"updatedAt":       ds.UpdatedAt,
	}
}

func healthImagingImportJobPayload(job *healthImagingImportJob) map[string]any {
	return map[string]any{
		"jobId":       job.ID,
		"jobStatus":   job.Status,
		"jobArn":      job.Arn,
		"datastoreId": job.DatastoreID,
		"submittedAt": job.CreatedAt,
		"updatedAt":   job.UpdatedAt,
	}
}

func healthImagingImageSetPayload(imageSet *healthImagingImageSet) map[string]any {
	return map[string]any{
		"imageSetId":    imageSet.ID,
		"versionId":     imageSet.VersionID,
		"imageSetArn":   imageSet.Arn,
		"datastoreId":   imageSet.DatastoreID,
		"imageSetState": imageSet.State,
		"createdAt":     imageSet.CreatedAt,
		"updatedAt":     imageSet.UpdatedAt,
	}
}

func healthImagingPathValue(pathParams map[string]string, key, fallback string) string {
	if pathParams == nil {
		return fallback
	}
	if value, ok := pathParams[key]; ok {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	for k, value := range pathParams {
		if strings.EqualFold(k, key) {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}

func healthImagingMapValue(payload map[string]any, key string) map[string]any {
	value := healthImagingValue(payload, key)
	if mapped, ok := value.(map[string]any); ok {
		return mapped
	}
	return map[string]any{}
}

func healthImagingValue(payload map[string]any, key string) any {
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

func healthImagingDefaultString(payload map[string]any, key, fallback string) string {
	value := healthImagingValue(payload, key)
	text := strings.TrimSpace(healthImagingToString(value))
	if text == "" {
		return fallback
	}
	return text
}

func healthImagingToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case []byte:
		return string(v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", value)
	}
}

func healthImagingStringSlice(value any) []string {
	switch v := value.(type) {
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
			text := strings.TrimSpace(healthImagingToString(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := strings.TrimSpace(healthImagingToString(value))
		if text == "" {
			return nil
		}
		return []string{text}
	}
}

func healthImagingExtractTagMap(value any) map[string]string {
	tags := map[string]string{}
	switch v := value.(type) {
	case map[string]string:
		for key, val := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			tags[key] = strings.TrimSpace(val)
		}
	case map[string]any:
		for rawKey, rawValue := range v {
			key := strings.TrimSpace(rawKey)
			if key == "" {
				continue
			}
			tags[key] = strings.TrimSpace(healthImagingToString(rawValue))
		}
	}
	return tags
}

func healthImagingCloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}
