package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type translateStore struct {
	mu sync.Mutex

	nextID int64

	terminologies map[string]*translateTerminology
	parallelData  map[string]*translateParallelData
	jobs          map[string]*translateTextTranslationJob
	jobByName     map[string]string
	jobByToken    map[string]string
	tags          map[string]map[string]string
}

type translateTerminology struct {
	Name        string
	ARN         string
	Description string
	Format      string
	Direction   string
	CreatedAt   string
	UpdatedAt   string
}

type translateParallelData struct {
	Name        string
	ARN         string
	Description string
	Status      string
	CreatedAt   string
	UpdatedAt   string
}

type translateTextTranslationJob struct {
	JobID             string
	JobName           string
	JobArn            string
	JobStatus         string
	SourceLanguage    string
	TargetLanguage    string
	SubmittedTime     string
	EndTime           string
	Message           string
	InputS3URI        string
	OutputS3URI       string
	ClientToken       string
	DataAccessRoleArn string
}

func newTranslateStore() *translateStore {
	now := time.Now().UTC().Format(time.RFC3339)

	terminology := &translateTerminology{
		Name:        "stackyard-terminology",
		ARN:         translateARN("terminology", "stackyard-terminology"),
		Description: "Stackyard seeded terminology",
		Format:      "CSV",
		Direction:   "UNI",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	parallel := &translateParallelData{
		Name:        "stackyard-parallel-data",
		ARN:         translateARN("parallel-data", "stackyard-parallel-data"),
		Description: "Stackyard seeded parallel data",
		Status:      "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return &translateStore{
		nextID: 2,
		terminologies: map[string]*translateTerminology{
			terminology.Name: terminology,
		},
		parallelData: map[string]*translateParallelData{
			parallel.Name: parallel,
		},
		jobs:       map[string]*translateTextTranslationJob{},
		jobByName:  map[string]string{},
		jobByToken: map[string]string{},
		tags: map[string]map[string]string{
			terminology.ARN: {"stackyard": "true"},
			parallel.ARN:    {"stackyard": "true"},
		},
	}
}

func (s *translateStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	sourceLang := translatePayloadString(payload, "SourceLanguageCode", "en")
	targetLang := translatePayloadString(payload, "TargetLanguageCode", "es")
	if strings.EqualFold(sourceLang, "auto") {
		sourceLang = "en"
	}

	switch action {
	case "ListLanguages":
		return map[string]any{
			"DisplayLanguageCode": "en",
			"Languages": []any{
				map[string]any{"LanguageCode": "en", "LanguageName": "English"},
				map[string]any{"LanguageCode": "es", "LanguageName": "Spanish"},
				map[string]any{"LanguageCode": "fr", "LanguageName": "French"},
				map[string]any{"LanguageCode": "de", "LanguageName": "German"},
			},
		}

	case "TranslateText":
		text := translatePayloadString(payload, "Text", "hello from stackyard")
		translated := fmt.Sprintf("[%s] %s", targetLang, text)
		return map[string]any{
			"TranslatedText":       translated,
			"SourceLanguageCode":   sourceLang,
			"TargetLanguageCode":   targetLang,
			"AppliedTerminologies": s.appliedTerminologiesPayloadLocked(payload),
		}

	case "TranslateDocument":
		doc := translatePayloadMap(payload, "Document")
		content := translateAnyToString(doc["Content"], "c3RhY2t5YXJkLXRyYW5zbGF0ZS1kb2N1bWVudA==")
		contentType := translateAnyToString(doc["ContentType"], "text/plain")
		return map[string]any{
			"TranslatedDocument": map[string]any{
				"Content":     content,
				"ContentType": contentType,
			},
			"SourceLanguageCode":   sourceLang,
			"TargetLanguageCode":   targetLang,
			"AppliedTerminologies": s.appliedTerminologiesPayloadLocked(payload),
		}

	case "ImportTerminology":
		name := translatePayloadString(payload, "Name", "stackyard-terminology")
		description := translatePayloadString(payload, "Description", "")
		term := s.ensureTerminologyLocked(name, now)
		if description != "" {
			term.Description = description
		}
		term.UpdatedAt = now
		s.ensureTagsLocked(term.ARN)
		return map[string]any{
			"TerminologyProperties": s.terminologyPropertiesPayload(term),
			"AuxiliaryDataLocation": map[string]any{"Location": "s3://stackyard-translate/auxiliary-data.tmx"},
		}

	case "GetTerminology":
		name := translatePayloadString(payload, "Name", "stackyard-terminology")
		term := s.ensureTerminologyLocked(name, now)
		return map[string]any{
			"TerminologyProperties": s.terminologyPropertiesPayload(term),
			"TerminologyData": map[string]any{
				"Format": term.Format,
				"File":   "source,target\nhello,hola",
			},
		}

	case "ListTerminologies":
		items := make([]any, 0, len(s.terminologies))
		for _, term := range s.sortedTerminologiesLocked() {
			items = append(items, s.terminologyPropertiesPayload(term))
		}
		return map[string]any{"TerminologyPropertiesList": items, "NextToken": ""}

	case "DeleteTerminology":
		name := translatePayloadString(payload, "Name", "stackyard-terminology")
		if term, ok := s.terminologies[name]; ok {
			delete(s.tags, term.ARN)
		}
		delete(s.terminologies, name)
		return map[string]any{}

	case "CreateParallelData":
		name := translatePayloadString(payload, "Name", s.nextTokenLocked("parallel-data"))
		pd := s.ensureParallelDataLocked(name, now)
		pd.Description = translatePayloadString(payload, "Description", pd.Description)
		pd.Status = "ACTIVE"
		pd.UpdatedAt = now
		s.ensureTagsLocked(pd.ARN)
		return map[string]any{"Name": pd.Name, "Status": pd.Status, "LatestUpdateAttemptStatus": "SUCCESS"}

	case "UpdateParallelData":
		name := translatePayloadString(payload, "Name", "stackyard-parallel-data")
		pd := s.ensureParallelDataLocked(name, now)
		pd.Description = translatePayloadString(payload, "Description", pd.Description)
		pd.Status = "ACTIVE"
		pd.UpdatedAt = now
		return map[string]any{"Name": pd.Name, "Status": pd.Status, "LatestUpdateAttemptStatus": "SUCCESS"}

	case "GetParallelData":
		name := translatePayloadString(payload, "Name", "stackyard-parallel-data")
		pd := s.ensureParallelDataLocked(name, now)
		return map[string]any{
			"ParallelDataProperties": s.parallelDataPropertiesPayload(pd),
			"DataLocation":           map[string]any{"Location": "s3://stackyard-translate/parallel-data.tmx"},
		}

	case "ListParallelData":
		items := make([]any, 0, len(s.parallelData))
		for _, pd := range s.sortedParallelDataLocked() {
			items = append(items, s.parallelDataPropertiesPayload(pd))
		}
		return map[string]any{"ParallelDataPropertiesList": items, "NextToken": ""}

	case "DeleteParallelData":
		name := translatePayloadString(payload, "Name", "stackyard-parallel-data")
		if pd, ok := s.parallelData[name]; ok {
			delete(s.tags, pd.ARN)
		}
		delete(s.parallelData, name)
		return map[string]any{}

	case "StartTextTranslationJob":
		jobName := translatePayloadString(payload, "JobName", s.nextTokenLocked("translate-job"))
		clientToken := translatePayloadString(payload, "ClientToken", "")
		input := translatePayloadMap(payload, "InputDataConfig")
		output := translatePayloadMap(payload, "OutputDataConfig")

		if clientToken != "" {
			if jobID, ok := s.jobByToken[clientToken]; ok {
				job := s.ensureJobLocked(jobID, now)
				return map[string]any{"JobId": job.JobID, "JobStatus": job.JobStatus}
			}
		}
		if existingJobID, ok := s.jobByName[jobName]; ok {
			job := s.ensureJobLocked(existingJobID, now)
			if clientToken != "" {
				s.jobByToken[clientToken] = job.JobID
			}
			return map[string]any{"JobId": job.JobID, "JobStatus": job.JobStatus}
		}

		jobID := s.nextTokenLocked("ttj")
		job := &translateTextTranslationJob{
			JobID:             jobID,
			JobName:           jobName,
			JobArn:            translateARN("text-translation-job", jobID),
			JobStatus:         "IN_PROGRESS",
			SourceLanguage:    sourceLang,
			TargetLanguage:    targetLang,
			SubmittedTime:     now,
			InputS3URI:        translateAnyToString(input["S3Uri"], "s3://stackyard-translate/input/"),
			OutputS3URI:       translateAnyToString(output["S3Uri"], "s3://stackyard-translate/output/"),
			ClientToken:       clientToken,
			DataAccessRoleArn: translatePayloadString(payload, "DataAccessRoleArn", "arn:aws:iam::123456789012:role/stackyard-translate"),
		}
		s.jobs[job.JobID] = job
		s.jobByName[job.JobName] = job.JobID
		if clientToken != "" {
			s.jobByToken[clientToken] = job.JobID
		}
		s.ensureTagsLocked(job.JobArn)
		return map[string]any{"JobId": job.JobID, "JobStatus": job.JobStatus}

	case "DescribeTextTranslationJob":
		jobID := translatePayloadString(payload, "JobId", "")
		job := s.jobForPayloadLocked(jobID, now)
		s.maybeCompleteJobLocked(job, now)
		return map[string]any{"TextTranslationJobProperties": s.textTranslationJobPropertiesPayload(job)}

	case "ListTextTranslationJobs":
		items := make([]any, 0, len(s.jobs))
		for _, job := range s.sortedJobsLocked() {
			s.maybeCompleteJobLocked(job, now)
			items = append(items, s.textTranslationJobPropertiesPayload(job))
		}
		return map[string]any{"TextTranslationJobPropertiesList": items, "NextToken": ""}

	case "StopTextTranslationJob":
		jobID := translatePayloadString(payload, "JobId", "")
		job := s.jobForPayloadLocked(jobID, now)
		job.JobStatus = "STOPPED"
		job.EndTime = now
		return map[string]any{"JobId": job.JobID, "JobStatus": job.JobStatus}

	case "TagResource":
		resourceARN := translatePayloadString(payload, "ResourceArn", translateARN("terminology", "stackyard-terminology"))
		tagMap := s.ensureTagsLocked(resourceARN)
		for _, tag := range translatePayloadTagList(payload, "Tags") {
			if tag.Key == "" {
				continue
			}
			tagMap[tag.Key] = tag.Value
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := translatePayloadString(payload, "ResourceArn", translateARN("terminology", "stackyard-terminology"))
		tagMap := s.ensureTagsLocked(resourceARN)
		for _, key := range translatePayloadStringSlice(payload, "TagKeys") {
			delete(tagMap, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := translatePayloadString(payload, "ResourceArn", translateARN("terminology", "stackyard-terminology"))
		tagMap := s.ensureTagsLocked(resourceARN)
		items := make([]any, 0, len(tagMap))
		keys := make([]string, 0, len(tagMap))
		for key := range tagMap {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			items = append(items, map[string]any{"Key": key, "Value": tagMap[key]})
		}
		return map[string]any{"Tags": items}
	}

	return map[string]any{}
}

func (s *translateStore) terminologyPropertiesPayload(term *translateTerminology) map[string]any {
	return map[string]any{
		"Name":                  term.Name,
		"Arn":                   term.ARN,
		"Description":           term.Description,
		"TerminologyDataFormat": term.Format,
		"Directionality":        term.Direction,
		"CreatedAt":             term.CreatedAt,
		"LastUpdatedAt":         term.UpdatedAt,
	}
}

func (s *translateStore) parallelDataPropertiesPayload(pd *translateParallelData) map[string]any {
	return map[string]any{
		"Name":                      pd.Name,
		"Arn":                       pd.ARN,
		"Description":               pd.Description,
		"Status":                    pd.Status,
		"CreatedAt":                 pd.CreatedAt,
		"LastUpdatedAt":             pd.UpdatedAt,
		"LatestUpdateAttemptStatus": "SUCCESS",
	}
}

func (s *translateStore) textTranslationJobPropertiesPayload(job *translateTextTranslationJob) map[string]any {
	return map[string]any{
		"JobId":              job.JobID,
		"JobName":            job.JobName,
		"JobArn":             job.JobArn,
		"JobStatus":          job.JobStatus,
		"SubmittedTime":      job.SubmittedTime,
		"EndTime":            job.EndTime,
		"Message":            job.Message,
		"SourceLanguageCode": job.SourceLanguage,
		"TargetLanguageCodes": []any{
			job.TargetLanguage,
		},
		"InputDataConfig":   map[string]any{"S3Uri": job.InputS3URI, "ContentType": "text/plain"},
		"OutputDataConfig":  map[string]any{"S3Uri": job.OutputS3URI},
		"DataAccessRoleArn": job.DataAccessRoleArn,
		"JobDetails": map[string]any{
			"InputDocumentsCount":      1,
			"TranslatedDocumentsCount": 1,
			"DocumentsWithErrorsCount": 0,
		},
	}
}

func (s *translateStore) appliedTerminologiesPayloadLocked(payload map[string]any) []any {
	names := translatePayloadStringSlice(payload, "TerminologyNames")
	if len(names) == 0 {
		return []any{}
	}
	items := make([]any, 0, len(names))
	for _, name := range names {
		term := s.ensureTerminologyLocked(name, time.Now().UTC().Format(time.RFC3339))
		items = append(items, map[string]any{"Name": term.Name, "Terms": []any{map[string]any{"SourceText": "hello", "TargetText": "hola"}}})
	}
	return items
}

func (s *translateStore) ensureTerminologyLocked(name, now string) *translateTerminology {
	if t, ok := s.terminologies[name]; ok {
		return t
	}
	t := &translateTerminology{
		Name:        name,
		ARN:         translateARN("terminology", name),
		Description: "",
		Format:      "CSV",
		Direction:   "UNI",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.terminologies[name] = t
	return t
}

func (s *translateStore) ensureParallelDataLocked(name, now string) *translateParallelData {
	if p, ok := s.parallelData[name]; ok {
		return p
	}
	p := &translateParallelData{
		Name:        name,
		ARN:         translateARN("parallel-data", name),
		Description: "",
		Status:      "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.parallelData[name] = p
	return p
}

func (s *translateStore) ensureJobLocked(jobID, now string) *translateTextTranslationJob {
	if j, ok := s.jobs[jobID]; ok {
		return j
	}
	job := &translateTextTranslationJob{
		JobID:             jobID,
		JobName:           "stackyard-translate-job",
		JobArn:            translateARN("text-translation-job", jobID),
		JobStatus:         "COMPLETED",
		SourceLanguage:    "en",
		TargetLanguage:    "es",
		SubmittedTime:     now,
		EndTime:           now,
		InputS3URI:        "s3://stackyard-translate/input/",
		OutputS3URI:       "s3://stackyard-translate/output/",
		DataAccessRoleArn: "arn:aws:iam::123456789012:role/stackyard-translate",
	}
	s.jobs[jobID] = job
	s.jobByName[job.JobName] = jobID
	return job
}

func (s *translateStore) jobForPayloadLocked(jobID, now string) *translateTextTranslationJob {
	if jobID != "" {
		return s.ensureJobLocked(jobID, now)
	}
	if len(s.jobs) == 0 {
		return s.ensureJobLocked(s.nextTokenLocked("ttj"), now)
	}
	ids := make([]string, 0, len(s.jobs))
	for id := range s.jobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return s.jobs[ids[0]]
}

func (s *translateStore) maybeCompleteJobLocked(job *translateTextTranslationJob, now string) {
	if strings.EqualFold(job.JobStatus, "IN_PROGRESS") {
		job.JobStatus = "COMPLETED"
		job.EndTime = now
	}
}

func (s *translateStore) ensureTagsLocked(resourceARN string) map[string]string {
	if tagMap, ok := s.tags[resourceARN]; ok {
		return tagMap
	}
	tagMap := map[string]string{}
	s.tags[resourceARN] = tagMap
	return tagMap
}

func (s *translateStore) sortedTerminologiesLocked() []*translateTerminology {
	items := make([]*translateTerminology, 0, len(s.terminologies))
	for _, item := range s.terminologies {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (s *translateStore) sortedParallelDataLocked() []*translateParallelData {
	items := make([]*translateParallelData, 0, len(s.parallelData))
	for _, item := range s.parallelData {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (s *translateStore) sortedJobsLocked() []*translateTextTranslationJob {
	items := make([]*translateTextTranslationJob, 0, len(s.jobs))
	for _, item := range s.jobs {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].JobID < items[j].JobID })
	return items
}

func (s *translateStore) nextTokenLocked(prefix string) string {
	token := fmt.Sprintf("%s-%06d", prefix, s.nextID)
	s.nextID++
	return token
}

func translateARN(resourceType, name string) string {
	resourceType = strings.Trim(strings.TrimSpace(resourceType), "/")
	name = strings.Trim(strings.TrimSpace(name), "/")
	if resourceType == "" {
		resourceType = "resource"
	}
	if name == "" {
		name = "stackyard"
	}
	return fmt.Sprintf("arn:aws:translate:us-east-1:123456789012:%s/%s", resourceType, name)
}

type translateTag struct {
	Key   string
	Value string
}

func translatePayloadTagList(payload map[string]any, key string) []translateTag {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]translateTag, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, translateTag{Key: translateAnyToString(entry["Key"], ""), Value: translateAnyToString(entry["Value"], "")})
	}
	return out
}

func translatePayloadMap(payload map[string]any, key string) map[string]any {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return map[string]any{}
	}
	asMap, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return asMap
}

func translatePayloadString(payload map[string]any, key, def string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return def
	}
	value := translateAnyToString(raw, def)
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

func translatePayloadStringSlice(payload map[string]any, key string) []string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		value := strings.TrimSpace(translateAnyToString(item, ""))
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func translateAnyToString(raw any, def string) string {
	switch value := raw.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	case nil:
		return def
	default:
		return fmt.Sprintf("%v", raw)
	}
}
