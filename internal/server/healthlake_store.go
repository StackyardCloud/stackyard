package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type healthLakeStore struct {
	mu         sync.Mutex
	next       int64
	datastores map[string]*healthLakeDatastore
	importJobs map[string]*healthLakeJob
	exportJobs map[string]*healthLakeJob
	tags       map[string]map[string]string
}

type healthLakeDatastore struct {
	ID          string
	Name        string
	Arn         string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Endpoint    string
	Error       string
	FHIRVersion string
}

type healthLakeJob struct {
	ID          string
	DatastoreID string
	JobType     string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Message     string
}

func newHealthLakeStore() *healthLakeStore {
	now := time.Now().UTC()
	ds := &healthLakeDatastore{
		ID:          "stackyard-datastore",
		Name:        "stackyard",
		Arn:         healthLakeDatastoreARN("stackyard-datastore"),
		Status:      "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
		Endpoint:    "https://healthlake.stackyard.local/r4/",
		FHIRVersion: "R4",
	}
	ij := &healthLakeJob{
		ID:          "import-000001",
		DatastoreID: ds.ID,
		JobType:     "IMPORT",
		Status:      "COMPLETED",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	ej := &healthLakeJob{
		ID:          "export-000001",
		DatastoreID: ds.ID,
		JobType:     "EXPORT",
		Status:      "COMPLETED",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return &healthLakeStore{
		next: 2,
		datastores: map[string]*healthLakeDatastore{
			ds.ID: ds,
		},
		importJobs: map[string]*healthLakeJob{
			ij.ID: ij,
		},
		exportJobs: map[string]*healthLakeJob{
			ej.ID: ej,
		},
		tags: map[string]map[string]string{
			ds.Arn: {"seed": "true"},
		},
	}
}

func (s *healthLakeStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "CreateFHIRDatastore":
		id := healthLakeDefaultString(payload, "DatastoreId", fmt.Sprintf("datastore-%06d", s.nextID()))
		name := healthLakeDefaultString(payload, "DatastoreName", id)
		ds := s.ensureDatastore(id)
		ds.Name = name
		ds.Status = "CREATING"
		ds.UpdatedAt = now
		ds.Status = "ACTIVE"
		return map[string]any{
			"DatastoreId":       ds.ID,
			"DatastoreArn":      ds.Arn,
			"DatastoreStatus":   ds.Status,
			"DatastoreEndpoint": ds.Endpoint,
		}

	case "DeleteFHIRDatastore":
		id := healthLakeResolveDatastoreID(payload)
		if ds, ok := s.datastores[id]; ok {
			ds.Status = "DELETING"
			ds.UpdatedAt = now
			delete(s.datastores, id)
		}
		return map[string]any{"DatastoreId": id, "DatastoreStatus": "DELETING"}

	case "DescribeFHIRDatastore":
		id := healthLakeResolveDatastoreID(payload)
		ds := s.ensureDatastore(id)
		return map[string]any{"DatastoreProperties": healthLakeDatastoreProperties(ds)}

	case "ListFHIRDatastores":
		items := make([]map[string]any, 0, len(s.datastores))
		ids := make([]string, 0, len(s.datastores))
		for id := range s.datastores {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			items = append(items, healthLakeDatastoreProperties(s.datastores[id]))
		}
		return map[string]any{"DatastorePropertiesList": items, "NextToken": ""}

	case "StartFHIRImportJob":
		dsID := healthLakeResolveDatastoreID(payload)
		jobID := healthLakeDefaultString(payload, "JobId", fmt.Sprintf("import-%06d", s.nextID()))
		job := &healthLakeJob{
			ID:          jobID,
			DatastoreID: dsID,
			JobType:     "IMPORT",
			Status:      "SUBMITTED",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		s.importJobs[job.ID] = job
		return map[string]any{
			"JobId":       job.ID,
			"JobStatus":   "SUBMITTED",
			"DatastoreId": dsID,
		}

	case "StartFHIRExportJob":
		dsID := healthLakeResolveDatastoreID(payload)
		jobID := healthLakeDefaultString(payload, "JobId", fmt.Sprintf("export-%06d", s.nextID()))
		job := &healthLakeJob{
			ID:          jobID,
			DatastoreID: dsID,
			JobType:     "EXPORT",
			Status:      "SUBMITTED",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		s.exportJobs[job.ID] = job
		return map[string]any{
			"JobId":       job.ID,
			"JobStatus":   "SUBMITTED",
			"DatastoreId": dsID,
		}

	case "DescribeFHIRImportJob":
		jobID := healthLakeDefaultString(payload, "JobId", "import-000001")
		job := s.ensureJob(s.importJobs, jobID, "IMPORT")
		return map[string]any{"ImportJobProperties": healthLakeImportJobProperties(job)}

	case "DescribeFHIRExportJob":
		jobID := healthLakeDefaultString(payload, "JobId", "export-000001")
		job := s.ensureJob(s.exportJobs, jobID, "EXPORT")
		return map[string]any{"ExportJobProperties": healthLakeExportJobProperties(job)}

	case "ListFHIRImportJobs":
		items := make([]map[string]any, 0, len(s.importJobs))
		ids := make([]string, 0, len(s.importJobs))
		for id := range s.importJobs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			items = append(items, healthLakeImportJobProperties(s.importJobs[id]))
		}
		return map[string]any{"ImportJobPropertiesList": items, "NextToken": ""}

	case "ListFHIRExportJobs":
		items := make([]map[string]any, 0, len(s.exportJobs))
		ids := make([]string, 0, len(s.exportJobs))
		for id := range s.exportJobs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			items = append(items, healthLakeExportJobProperties(s.exportJobs[id]))
		}
		return map[string]any{"ExportJobPropertiesList": items, "NextToken": ""}

	case "TagResource":
		resourceARN := healthLakeDefaultString(payload, "ResourceARN", "")
		if resourceARN == "" {
			resourceARN = healthLakeDatastoreARN(healthLakeResolveDatastoreID(payload))
		}
		current := s.tags[resourceARN]
		if current == nil {
			current = map[string]string{}
		}
		for k, v := range healthLakeExtractTagMap(payload["Tags"]) {
			current[k] = v
		}
		s.tags[resourceARN] = current
		return map[string]any{}

	case "UntagResource":
		resourceARN := healthLakeDefaultString(payload, "ResourceARN", "")
		if resourceARN == "" {
			resourceARN = healthLakeDatastoreARN(healthLakeResolveDatastoreID(payload))
		}
		current := s.tags[resourceARN]
		for _, key := range healthLakeStringSlice(payload["TagKeys"]) {
			delete(current, key)
		}
		s.tags[resourceARN] = current
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := healthLakeDefaultString(payload, "ResourceARN", "")
		if resourceARN == "" {
			resourceARN = healthLakeDatastoreARN(healthLakeResolveDatastoreID(payload))
		}
		return map[string]any{"Tags": healthLakeCloneStringMap(s.tags[resourceARN])}
	}

	return map[string]any{}
}

func (s *healthLakeStore) nextID() int64 {
	s.next++
	return s.next
}

func (s *healthLakeStore) ensureDatastore(id string) *healthLakeDatastore {
	if id == "" {
		id = "stackyard-datastore"
	}
	if ds, ok := s.datastores[id]; ok {
		return ds
	}
	now := time.Now().UTC()
	ds := &healthLakeDatastore{
		ID:          id,
		Name:        id,
		Arn:         healthLakeDatastoreARN(id),
		Status:      "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
		Endpoint:    "https://healthlake.stackyard.local/r4/",
		FHIRVersion: "R4",
	}
	s.datastores[id] = ds
	return ds
}

func (s *healthLakeStore) ensureJob(jobs map[string]*healthLakeJob, id, jobType string) *healthLakeJob {
	if id == "" {
		id = strings.ToLower(jobType) + "-000001"
	}
	if job, ok := jobs[id]; ok {
		return job
	}
	now := time.Now().UTC()
	job := &healthLakeJob{
		ID:          id,
		DatastoreID: "stackyard-datastore",
		JobType:     jobType,
		Status:      "COMPLETED",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	jobs[id] = job
	return job
}

func healthLakeDatastoreARN(id string) string {
	return fmt.Sprintf("arn:aws:healthlake:us-east-1:123456789012:datastore/%s", id)
}

func healthLakeDatastoreProperties(ds *healthLakeDatastore) map[string]any {
	return map[string]any{
		"DatastoreId":          ds.ID,
		"DatastoreName":        ds.Name,
		"DatastoreArn":         ds.Arn,
		"DatastoreStatus":      ds.Status,
		"DatastoreTypeVersion": ds.FHIRVersion,
		"CreatedAt":            ds.CreatedAt.Unix(),
		"Endpoint":             ds.Endpoint,
	}
}

func healthLakeImportJobProperties(job *healthLakeJob) map[string]any {
	return map[string]any{
		"JobId":       job.ID,
		"JobStatus":   job.Status,
		"SubmitTime":  job.CreatedAt.Unix(),
		"DatastoreId": job.DatastoreID,
		"InputDataConfig": map[string]any{
			"S3Uri": "s3://stackyard-healthlake/import",
		},
		"JobProgressReport": map[string]any{
			"TotalNumberOfScannedFiles":   1,
			"TotalSizeOfScannedFilesInMB": 1.0,
		},
	}
}

func healthLakeExportJobProperties(job *healthLakeJob) map[string]any {
	return map[string]any{
		"JobId":       job.ID,
		"JobStatus":   job.Status,
		"SubmitTime":  job.CreatedAt.Unix(),
		"DatastoreId": job.DatastoreID,
		"OutputDataConfig": map[string]any{
			"S3Uri": "s3://stackyard-healthlake/export",
		},
	}
}

func healthLakeResolveDatastoreID(payload map[string]any) string {
	if payload == nil {
		return "stackyard-datastore"
	}
	for _, key := range []string{"DatastoreId", "datastoreId"} {
		if value, ok := payload[key]; ok {
			text := strings.TrimSpace(fmt.Sprintf("%v", value))
			if text != "" {
				return text
			}
		}
	}
	return "stackyard-datastore"
}

func healthLakeDefaultString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, value := range payload {
		if strings.EqualFold(k, key) {
			text := strings.TrimSpace(fmt.Sprintf("%v", value))
			if text != "" {
				return text
			}
			break
		}
	}
	return fallback
}

func healthLakeStringSlice(value any) []string {
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
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "" || text == "<nil>" {
			return nil
		}
		return []string{text}
	}
}

func healthLakeExtractTagMap(value any) map[string]string {
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]string:
		for k, v := range typed {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(v)
		}
	case map[string]any:
		for k, v := range typed {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	return out
}

func healthLakeCloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
