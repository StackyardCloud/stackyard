package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type transcribeStore struct {
	mu sync.Mutex

	nextID int64

	transcriptionJobs        map[string]*transcribeJob
	medicalTranscriptionJobs map[string]*transcribeJob
	medicalScribeJobs        map[string]*transcribeJob
	callAnalyticsJobs        map[string]*transcribeJob
	callAnalyticsCategories  map[string]*transcribeCategory
	languageModels           map[string]*transcribeLanguageModel
	vocabularies             map[string]*transcribeVocabulary
	vocabularyFilters        map[string]*transcribeVocabularyFilter
	medicalVocabularies      map[string]*transcribeVocabulary
	tags                     map[string]map[string]string
}

type transcribeJob struct {
	Name         string
	ARN          string
	Status       string
	LanguageCode string
	CreatedAt    string
	UpdatedAt    string
	CompletedAt  string
}

type transcribeCategory struct {
	Name      string
	ARN       string
	CreatedAt string
	UpdatedAt string
}

type transcribeLanguageModel struct {
	Name      string
	ARN       string
	Status    string
	CreatedAt string
	UpdatedAt string
}

type transcribeVocabulary struct {
	Name         string
	ARN          string
	LanguageCode string
	State        string
	LastModified string
}

type transcribeVocabularyFilter struct {
	Name         string
	ARN          string
	LanguageCode string
	LastModified string
}

func newTranscribeStore() *transcribeStore {
	now := time.Now().UTC().Format(time.RFC3339)

	tJob := &transcribeJob{
		Name:         "stackyard-transcription-job",
		ARN:          transcribeARN("transcription-job", "stackyard-transcription-job"),
		Status:       "COMPLETED",
		LanguageCode: "en-US",
		CreatedAt:    now,
		UpdatedAt:    now,
		CompletedAt:  now,
	}
	mJob := &transcribeJob{
		Name:         "stackyard-medical-transcription-job",
		ARN:          transcribeARN("medical-transcription-job", "stackyard-medical-transcription-job"),
		Status:       "COMPLETED",
		LanguageCode: "en-US",
		CreatedAt:    now,
		UpdatedAt:    now,
		CompletedAt:  now,
	}
	scribeJob := &transcribeJob{
		Name:         "stackyard-medical-scribe-job",
		ARN:          transcribeARN("medical-scribe-job", "stackyard-medical-scribe-job"),
		Status:       "COMPLETED",
		LanguageCode: "en-US",
		CreatedAt:    now,
		UpdatedAt:    now,
		CompletedAt:  now,
	}
	caJob := &transcribeJob{
		Name:         "stackyard-call-analytics-job",
		ARN:          transcribeARN("call-analytics-job", "stackyard-call-analytics-job"),
		Status:       "COMPLETED",
		LanguageCode: "en-US",
		CreatedAt:    now,
		UpdatedAt:    now,
		CompletedAt:  now,
	}
	category := &transcribeCategory{
		Name:      "stackyard-call-analytics-category",
		ARN:       transcribeARN("call-analytics-category", "stackyard-call-analytics-category"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	model := &transcribeLanguageModel{
		Name:      "stackyard-language-model",
		ARN:       transcribeARN("language-model", "stackyard-language-model"),
		Status:    "COMPLETED",
		CreatedAt: now,
		UpdatedAt: now,
	}
	vocab := &transcribeVocabulary{
		Name:         "stackyard-vocabulary",
		ARN:          transcribeARN("vocabulary", "stackyard-vocabulary"),
		LanguageCode: "en-US",
		State:        "READY",
		LastModified: now,
	}
	vocabFilter := &transcribeVocabularyFilter{
		Name:         "stackyard-vocabulary-filter",
		ARN:          transcribeARN("vocabulary-filter", "stackyard-vocabulary-filter"),
		LanguageCode: "en-US",
		LastModified: now,
	}
	medicalVocab := &transcribeVocabulary{
		Name:         "stackyard-medical-vocabulary",
		ARN:          transcribeARN("medical-vocabulary", "stackyard-medical-vocabulary"),
		LanguageCode: "en-US",
		State:        "READY",
		LastModified: now,
	}

	return &transcribeStore{
		nextID: 2,
		transcriptionJobs: map[string]*transcribeJob{
			tJob.Name: tJob,
		},
		medicalTranscriptionJobs: map[string]*transcribeJob{
			mJob.Name: mJob,
		},
		medicalScribeJobs: map[string]*transcribeJob{
			scribeJob.Name: scribeJob,
		},
		callAnalyticsJobs: map[string]*transcribeJob{
			caJob.Name: caJob,
		},
		callAnalyticsCategories: map[string]*transcribeCategory{
			category.Name: category,
		},
		languageModels: map[string]*transcribeLanguageModel{
			model.Name: model,
		},
		vocabularies: map[string]*transcribeVocabulary{
			vocab.Name: vocab,
		},
		vocabularyFilters: map[string]*transcribeVocabularyFilter{
			vocabFilter.Name: vocabFilter,
		},
		medicalVocabularies: map[string]*transcribeVocabulary{
			medicalVocab.Name: medicalVocab,
		},
		tags: map[string]map[string]string{
			tJob.ARN:      {"stackyard": "true"},
			mJob.ARN:      {"stackyard": "true"},
			scribeJob.ARN: {"stackyard": "true"},
			caJob.ARN:     {"stackyard": "true"},
			category.ARN:  {"stackyard": "true"},
			model.ARN:     {"stackyard": "true"},
			vocab.ARN:     {"stackyard": "true"},
		},
	}
}

func (s *transcribeStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateVocabulary":
		name := transcribePayloadString(payload, "VocabularyName", s.nextTokenLocked("vocabulary"))
		languageCode := transcribePayloadString(payload, "LanguageCode", "en-US")
		v := s.ensureVocabularyLocked(name)
		v.LanguageCode = languageCode
		v.State = "READY"
		v.LastModified = now
		s.ensureTagsLocked(v.ARN)
		return map[string]any{"VocabularyName": v.Name, "LanguageCode": v.LanguageCode, "VocabularyState": v.State, "LastModifiedTime": v.LastModified}

	case "GetVocabulary":
		name := transcribePayloadString(payload, "VocabularyName", "stackyard-vocabulary")
		v := s.ensureVocabularyLocked(name)
		return map[string]any{"VocabularyName": v.Name, "LanguageCode": v.LanguageCode, "VocabularyState": v.State, "LastModifiedTime": v.LastModified}

	case "ListVocabularies":
		items := make([]any, 0, len(s.vocabularies))
		for _, v := range s.sortedVocabulariesLocked(s.vocabularies) {
			items = append(items, map[string]any{"VocabularyName": v.Name, "LanguageCode": v.LanguageCode, "VocabularyState": v.State, "LastModifiedTime": v.LastModified})
		}
		return map[string]any{"Vocabularies": items, "NextToken": ""}

	case "UpdateVocabulary":
		name := transcribePayloadString(payload, "VocabularyName", "stackyard-vocabulary")
		languageCode := transcribePayloadString(payload, "LanguageCode", "en-US")
		v := s.ensureVocabularyLocked(name)
		v.LanguageCode = languageCode
		v.State = "READY"
		v.LastModified = now
		return map[string]any{"VocabularyName": v.Name, "LanguageCode": v.LanguageCode, "VocabularyState": v.State, "LastModifiedTime": v.LastModified}

	case "DeleteVocabulary":
		name := transcribePayloadString(payload, "VocabularyName", "stackyard-vocabulary")
		if v, ok := s.vocabularies[name]; ok {
			delete(s.tags, v.ARN)
		}
		delete(s.vocabularies, name)
		return map[string]any{}

	case "CreateVocabularyFilter":
		name := transcribePayloadString(payload, "VocabularyFilterName", s.nextTokenLocked("vocabulary-filter"))
		languageCode := transcribePayloadString(payload, "LanguageCode", "en-US")
		vf := s.ensureVocabularyFilterLocked(name)
		vf.LanguageCode = languageCode
		vf.LastModified = now
		s.ensureTagsLocked(vf.ARN)
		return map[string]any{"VocabularyFilterName": vf.Name, "LanguageCode": vf.LanguageCode, "LastModifiedTime": vf.LastModified}

	case "GetVocabularyFilter":
		name := transcribePayloadString(payload, "VocabularyFilterName", "stackyard-vocabulary-filter")
		vf := s.ensureVocabularyFilterLocked(name)
		return map[string]any{"VocabularyFilterName": vf.Name, "LanguageCode": vf.LanguageCode, "LastModifiedTime": vf.LastModified}

	case "ListVocabularyFilters":
		items := make([]any, 0, len(s.vocabularyFilters))
		for _, vf := range s.sortedVocabularyFiltersLocked() {
			items = append(items, map[string]any{"VocabularyFilterName": vf.Name, "LanguageCode": vf.LanguageCode, "LastModifiedTime": vf.LastModified})
		}
		return map[string]any{"VocabularyFilters": items, "NextToken": ""}

	case "UpdateVocabularyFilter":
		name := transcribePayloadString(payload, "VocabularyFilterName", "stackyard-vocabulary-filter")
		languageCode := transcribePayloadString(payload, "LanguageCode", "en-US")
		vf := s.ensureVocabularyFilterLocked(name)
		vf.LanguageCode = languageCode
		vf.LastModified = now
		return map[string]any{"VocabularyFilterName": vf.Name, "LanguageCode": vf.LanguageCode, "LastModifiedTime": vf.LastModified}

	case "DeleteVocabularyFilter":
		name := transcribePayloadString(payload, "VocabularyFilterName", "stackyard-vocabulary-filter")
		if vf, ok := s.vocabularyFilters[name]; ok {
			delete(s.tags, vf.ARN)
		}
		delete(s.vocabularyFilters, name)
		return map[string]any{}

	case "CreateMedicalVocabulary":
		name := transcribePayloadString(payload, "VocabularyName", s.nextTokenLocked("medical-vocabulary"))
		languageCode := transcribePayloadString(payload, "LanguageCode", "en-US")
		v := s.ensureMedicalVocabularyLocked(name)
		v.LanguageCode = languageCode
		v.State = "READY"
		v.LastModified = now
		s.ensureTagsLocked(v.ARN)
		return map[string]any{"VocabularyName": v.Name, "LanguageCode": v.LanguageCode, "VocabularyState": v.State, "LastModifiedTime": v.LastModified}

	case "GetMedicalVocabulary":
		name := transcribePayloadString(payload, "VocabularyName", "stackyard-medical-vocabulary")
		v := s.ensureMedicalVocabularyLocked(name)
		return map[string]any{"VocabularyName": v.Name, "LanguageCode": v.LanguageCode, "VocabularyState": v.State, "LastModifiedTime": v.LastModified}

	case "ListMedicalVocabularies":
		items := make([]any, 0, len(s.medicalVocabularies))
		for _, v := range s.sortedVocabulariesLocked(s.medicalVocabularies) {
			items = append(items, map[string]any{"VocabularyName": v.Name, "LanguageCode": v.LanguageCode, "VocabularyState": v.State, "LastModifiedTime": v.LastModified})
		}
		return map[string]any{"Vocabularies": items, "NextToken": ""}

	case "UpdateMedicalVocabulary":
		name := transcribePayloadString(payload, "VocabularyName", "stackyard-medical-vocabulary")
		languageCode := transcribePayloadString(payload, "LanguageCode", "en-US")
		v := s.ensureMedicalVocabularyLocked(name)
		v.LanguageCode = languageCode
		v.State = "READY"
		v.LastModified = now
		return map[string]any{"VocabularyName": v.Name, "LanguageCode": v.LanguageCode, "VocabularyState": v.State, "LastModifiedTime": v.LastModified}

	case "DeleteMedicalVocabulary":
		name := transcribePayloadString(payload, "VocabularyName", "stackyard-medical-vocabulary")
		if v, ok := s.medicalVocabularies[name]; ok {
			delete(s.tags, v.ARN)
		}
		delete(s.medicalVocabularies, name)
		return map[string]any{}

	case "StartTranscriptionJob":
		name := transcribePayloadString(payload, "TranscriptionJobName", s.nextTokenLocked("transcription-job"))
		languageCode := transcribePayloadString(payload, "LanguageCode", "en-US")
		job := s.ensureJobLocked(s.transcriptionJobs, "transcription-job", name, languageCode)
		job.Status = "IN_PROGRESS"
		job.UpdatedAt = now
		job.CompletedAt = ""
		s.ensureTagsLocked(job.ARN)
		return map[string]any{"TranscriptionJob": s.transcriptionJobPayload(job)}

	case "GetTranscriptionJob":
		name := transcribePayloadString(payload, "TranscriptionJobName", "stackyard-transcription-job")
		job := s.ensureJobLocked(s.transcriptionJobs, "transcription-job", name, "en-US")
		s.maybeCompleteJob(job, now)
		return map[string]any{"TranscriptionJob": s.transcriptionJobPayload(job)}

	case "ListTranscriptionJobs":
		items := make([]any, 0, len(s.transcriptionJobs))
		for _, job := range s.sortedJobsLocked(s.transcriptionJobs) {
			s.maybeCompleteJob(job, now)
			items = append(items, s.transcriptionJobSummary(job))
		}
		return map[string]any{"TranscriptionJobSummaries": items, "NextToken": ""}

	case "DeleteTranscriptionJob":
		name := transcribePayloadString(payload, "TranscriptionJobName", "stackyard-transcription-job")
		if job, ok := s.transcriptionJobs[name]; ok {
			delete(s.tags, job.ARN)
		}
		delete(s.transcriptionJobs, name)
		return map[string]any{}

	case "StartMedicalTranscriptionJob":
		name := transcribePayloadString(payload, "MedicalTranscriptionJobName", s.nextTokenLocked("medical-transcription-job"))
		languageCode := transcribePayloadString(payload, "LanguageCode", "en-US")
		job := s.ensureJobLocked(s.medicalTranscriptionJobs, "medical-transcription-job", name, languageCode)
		job.Status = "IN_PROGRESS"
		job.UpdatedAt = now
		job.CompletedAt = ""
		s.ensureTagsLocked(job.ARN)
		return map[string]any{"MedicalTranscriptionJob": s.medicalTranscriptionJobPayload(job)}

	case "GetMedicalTranscriptionJob":
		name := transcribePayloadString(payload, "MedicalTranscriptionJobName", "stackyard-medical-transcription-job")
		job := s.ensureJobLocked(s.medicalTranscriptionJobs, "medical-transcription-job", name, "en-US")
		s.maybeCompleteJob(job, now)
		return map[string]any{"MedicalTranscriptionJob": s.medicalTranscriptionJobPayload(job)}

	case "ListMedicalTranscriptionJobs":
		items := make([]any, 0, len(s.medicalTranscriptionJobs))
		for _, job := range s.sortedJobsLocked(s.medicalTranscriptionJobs) {
			s.maybeCompleteJob(job, now)
			items = append(items, s.medicalTranscriptionJobSummary(job))
		}
		return map[string]any{"MedicalTranscriptionJobSummaries": items, "NextToken": ""}

	case "DeleteMedicalTranscriptionJob":
		name := transcribePayloadString(payload, "MedicalTranscriptionJobName", "stackyard-medical-transcription-job")
		if job, ok := s.medicalTranscriptionJobs[name]; ok {
			delete(s.tags, job.ARN)
		}
		delete(s.medicalTranscriptionJobs, name)
		return map[string]any{}

	case "StartMedicalScribeJob":
		name := transcribePayloadString(payload, "MedicalScribeJobName", s.nextTokenLocked("medical-scribe-job"))
		languageCode := transcribePayloadString(payload, "LanguageCode", "en-US")
		job := s.ensureJobLocked(s.medicalScribeJobs, "medical-scribe-job", name, languageCode)
		job.Status = "IN_PROGRESS"
		job.UpdatedAt = now
		job.CompletedAt = ""
		s.ensureTagsLocked(job.ARN)
		return map[string]any{"MedicalScribeJob": s.medicalScribeJobPayload(job)}

	case "GetMedicalScribeJob":
		name := transcribePayloadString(payload, "MedicalScribeJobName", "stackyard-medical-scribe-job")
		job := s.ensureJobLocked(s.medicalScribeJobs, "medical-scribe-job", name, "en-US")
		s.maybeCompleteJob(job, now)
		return map[string]any{"MedicalScribeJob": s.medicalScribeJobPayload(job)}

	case "ListMedicalScribeJobs":
		items := make([]any, 0, len(s.medicalScribeJobs))
		for _, job := range s.sortedJobsLocked(s.medicalScribeJobs) {
			s.maybeCompleteJob(job, now)
			items = append(items, s.medicalScribeJobSummary(job))
		}
		return map[string]any{"MedicalScribeJobSummaries": items, "NextToken": ""}

	case "DeleteMedicalScribeJob":
		name := transcribePayloadString(payload, "MedicalScribeJobName", "stackyard-medical-scribe-job")
		if job, ok := s.medicalScribeJobs[name]; ok {
			delete(s.tags, job.ARN)
		}
		delete(s.medicalScribeJobs, name)
		return map[string]any{}

	case "CreateCallAnalyticsCategory":
		name := transcribePayloadString(payload, "CategoryName", s.nextTokenLocked("call-analytics-category"))
		category := s.ensureCallAnalyticsCategoryLocked(name)
		category.UpdatedAt = now
		s.ensureTagsLocked(category.ARN)
		return map[string]any{"CategoryProperties": s.callAnalyticsCategoryPayload(category)}

	case "GetCallAnalyticsCategory":
		name := transcribePayloadString(payload, "CategoryName", "stackyard-call-analytics-category")
		category := s.ensureCallAnalyticsCategoryLocked(name)
		return map[string]any{"CategoryProperties": s.callAnalyticsCategoryPayload(category)}

	case "ListCallAnalyticsCategories":
		items := make([]any, 0, len(s.callAnalyticsCategories))
		for _, category := range s.sortedCallAnalyticsCategoriesLocked() {
			items = append(items, s.callAnalyticsCategoryPayload(category))
		}
		return map[string]any{"Categories": items, "NextToken": ""}

	case "UpdateCallAnalyticsCategory":
		name := transcribePayloadString(payload, "CategoryName", "stackyard-call-analytics-category")
		category := s.ensureCallAnalyticsCategoryLocked(name)
		category.UpdatedAt = now
		return map[string]any{"CategoryProperties": s.callAnalyticsCategoryPayload(category)}

	case "DeleteCallAnalyticsCategory":
		name := transcribePayloadString(payload, "CategoryName", "stackyard-call-analytics-category")
		if category, ok := s.callAnalyticsCategories[name]; ok {
			delete(s.tags, category.ARN)
		}
		delete(s.callAnalyticsCategories, name)
		return map[string]any{}

	case "StartCallAnalyticsJob":
		name := transcribePayloadString(payload, "CallAnalyticsJobName", s.nextTokenLocked("call-analytics-job"))
		languageCode := transcribePayloadString(payload, "LanguageCode", "en-US")
		job := s.ensureJobLocked(s.callAnalyticsJobs, "call-analytics-job", name, languageCode)
		job.Status = "IN_PROGRESS"
		job.UpdatedAt = now
		job.CompletedAt = ""
		s.ensureTagsLocked(job.ARN)
		return map[string]any{"CallAnalyticsJob": s.callAnalyticsJobPayload(job)}

	case "GetCallAnalyticsJob":
		name := transcribePayloadString(payload, "CallAnalyticsJobName", "stackyard-call-analytics-job")
		job := s.ensureJobLocked(s.callAnalyticsJobs, "call-analytics-job", name, "en-US")
		s.maybeCompleteJob(job, now)
		return map[string]any{"CallAnalyticsJob": s.callAnalyticsJobPayload(job)}

	case "ListCallAnalyticsJobs":
		items := make([]any, 0, len(s.callAnalyticsJobs))
		for _, job := range s.sortedJobsLocked(s.callAnalyticsJobs) {
			s.maybeCompleteJob(job, now)
			items = append(items, s.callAnalyticsJobSummary(job))
		}
		return map[string]any{"CallAnalyticsJobSummaries": items, "NextToken": ""}

	case "DeleteCallAnalyticsJob":
		name := transcribePayloadString(payload, "CallAnalyticsJobName", "stackyard-call-analytics-job")
		if job, ok := s.callAnalyticsJobs[name]; ok {
			delete(s.tags, job.ARN)
		}
		delete(s.callAnalyticsJobs, name)
		return map[string]any{}

	case "CreateLanguageModel":
		name := transcribePayloadString(payload, "ModelName", s.nextTokenLocked("language-model"))
		model := s.ensureLanguageModelLocked(name)
		model.Status = "COMPLETED"
		model.UpdatedAt = now
		s.ensureTagsLocked(model.ARN)
		return map[string]any{"LanguageModel": s.languageModelPayload(model)}

	case "DescribeLanguageModel":
		name := transcribePayloadString(payload, "ModelName", "stackyard-language-model")
		model := s.ensureLanguageModelLocked(name)
		return map[string]any{"LanguageModel": s.languageModelPayload(model)}

	case "ListLanguageModels":
		items := make([]any, 0, len(s.languageModels))
		for _, model := range s.sortedLanguageModelsLocked() {
			items = append(items, s.languageModelPayload(model))
		}
		return map[string]any{"Models": items, "NextToken": ""}

	case "DeleteLanguageModel":
		name := transcribePayloadString(payload, "ModelName", "stackyard-language-model")
		if model, ok := s.languageModels[name]; ok {
			delete(s.tags, model.ARN)
		}
		delete(s.languageModels, name)
		return map[string]any{}

	case "TagResource":
		resourceARN := transcribePayloadString(payload, "ResourceArn", transcribeARN("transcription-job", "stackyard-transcription-job"))
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range transcribePayloadTags(payload, "Tags") {
			tags[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := transcribePayloadString(payload, "ResourceArn", transcribeARN("transcription-job", "stackyard-transcription-job"))
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range transcribePayloadStringSlice(payload, "TagKeys") {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := transcribePayloadString(payload, "ResourceArn", transcribeARN("transcription-job", "stackyard-transcription-job"))
		tags := s.ensureTagsLocked(resourceARN)
		return map[string]any{"ResourceArn": resourceARN, "Tags": s.tagsList(tags)}
	}

	return map[string]any{}
}

func (s *transcribeStore) ensureJobLocked(target map[string]*transcribeJob, resourceType, name, languageCode string) *transcribeJob {
	name = strings.TrimSpace(name)
	if name == "" {
		name = s.nextTokenLocked(resourceType)
	}
	if job, ok := target[name]; ok {
		return job
	}
	now := time.Now().UTC().Format(time.RFC3339)
	job := &transcribeJob{
		Name:         name,
		ARN:          transcribeARN(resourceType, name),
		Status:       "COMPLETED",
		LanguageCode: languageCode,
		CreatedAt:    now,
		UpdatedAt:    now,
		CompletedAt:  now,
	}
	target[name] = job
	return job
}

func (s *transcribeStore) ensureCallAnalyticsCategoryLocked(name string) *transcribeCategory {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-call-analytics-category"
	}
	if category, ok := s.callAnalyticsCategories[name]; ok {
		return category
	}
	now := time.Now().UTC().Format(time.RFC3339)
	category := &transcribeCategory{Name: name, ARN: transcribeARN("call-analytics-category", name), CreatedAt: now, UpdatedAt: now}
	s.callAnalyticsCategories[name] = category
	return category
}

func (s *transcribeStore) ensureLanguageModelLocked(name string) *transcribeLanguageModel {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-language-model"
	}
	if model, ok := s.languageModels[name]; ok {
		return model
	}
	now := time.Now().UTC().Format(time.RFC3339)
	model := &transcribeLanguageModel{Name: name, ARN: transcribeARN("language-model", name), Status: "COMPLETED", CreatedAt: now, UpdatedAt: now}
	s.languageModels[name] = model
	return model
}

func (s *transcribeStore) ensureVocabularyLocked(name string) *transcribeVocabulary {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-vocabulary"
	}
	if v, ok := s.vocabularies[name]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	v := &transcribeVocabulary{Name: name, ARN: transcribeARN("vocabulary", name), LanguageCode: "en-US", State: "READY", LastModified: now}
	s.vocabularies[name] = v
	return v
}

func (s *transcribeStore) ensureMedicalVocabularyLocked(name string) *transcribeVocabulary {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-medical-vocabulary"
	}
	if v, ok := s.medicalVocabularies[name]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	v := &transcribeVocabulary{Name: name, ARN: transcribeARN("medical-vocabulary", name), LanguageCode: "en-US", State: "READY", LastModified: now}
	s.medicalVocabularies[name] = v
	return v
}

func (s *transcribeStore) ensureVocabularyFilterLocked(name string) *transcribeVocabularyFilter {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-vocabulary-filter"
	}
	if v, ok := s.vocabularyFilters[name]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	v := &transcribeVocabularyFilter{Name: name, ARN: transcribeARN("vocabulary-filter", name), LanguageCode: "en-US", LastModified: now}
	s.vocabularyFilters[name] = v
	return v
}

func (s *transcribeStore) maybeCompleteJob(job *transcribeJob, now string) {
	if job == nil {
		return
	}
	if strings.EqualFold(job.Status, "IN_PROGRESS") {
		job.Status = "COMPLETED"
		job.UpdatedAt = now
		job.CompletedAt = now
	}
}

func (s *transcribeStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = transcribeARN("transcription-job", "stackyard-transcription-job")
	}
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{"stackyard": "true"}
	s.tags[resourceARN] = tags
	return tags
}

func (s *transcribeStore) tagsList(tags map[string]string) []any {
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

func (s *transcribeStore) transcriptionJobPayload(job *transcribeJob) map[string]any {
	return map[string]any{
		"TranscriptionJobName":   job.Name,
		"TranscriptionJobStatus": job.Status,
		"LanguageCode":           job.LanguageCode,
		"CreationTime":           job.CreatedAt,
		"CompletionTime":         job.CompletedAt,
		"Transcript":             map[string]any{"TranscriptFileUri": "s3://stackyard-transcribe/output/" + job.Name + ".json"},
	}
}

func (s *transcribeStore) transcriptionJobSummary(job *transcribeJob) map[string]any {
	return map[string]any{
		"TranscriptionJobName":   job.Name,
		"TranscriptionJobStatus": job.Status,
		"LanguageCode":           job.LanguageCode,
		"CreationTime":           job.CreatedAt,
		"CompletionTime":         job.CompletedAt,
	}
}

func (s *transcribeStore) medicalTranscriptionJobPayload(job *transcribeJob) map[string]any {
	return map[string]any{
		"MedicalTranscriptionJobName":    job.Name,
		"TranscriptionJobStatus":         job.Status,
		"LanguageCode":                   job.LanguageCode,
		"CreationTime":                   job.CreatedAt,
		"CompletionTime":                 job.CompletedAt,
		"Transcript":                     map[string]any{"TranscriptFileUri": "s3://stackyard-transcribe/output/" + job.Name + ".json"},
		"MedicalTranscriptionJobDetails": map[string]any{"MedicalTranscriptionJobName": job.Name},
	}
}

func (s *transcribeStore) medicalTranscriptionJobSummary(job *transcribeJob) map[string]any {
	return map[string]any{
		"MedicalTranscriptionJobName": job.Name,
		"TranscriptionJobStatus":      job.Status,
		"LanguageCode":                job.LanguageCode,
		"CreationTime":                job.CreatedAt,
		"CompletionTime":              job.CompletedAt,
	}
}

func (s *transcribeStore) medicalScribeJobPayload(job *transcribeJob) map[string]any {
	return map[string]any{
		"MedicalScribeJobName":   job.Name,
		"MedicalScribeJobStatus": job.Status,
		"LanguageCode":           job.LanguageCode,
		"CreationTime":           job.CreatedAt,
		"CompletionTime":         job.CompletedAt,
		"MedicalScribeOutput":    map[string]any{"TranscriptFileUri": "s3://stackyard-transcribe/output/" + job.Name + ".json"},
	}
}

func (s *transcribeStore) medicalScribeJobSummary(job *transcribeJob) map[string]any {
	return map[string]any{
		"MedicalScribeJobName":   job.Name,
		"MedicalScribeJobStatus": job.Status,
		"LanguageCode":           job.LanguageCode,
		"CreationTime":           job.CreatedAt,
		"CompletionTime":         job.CompletedAt,
	}
}

func (s *transcribeStore) callAnalyticsJobPayload(job *transcribeJob) map[string]any {
	return map[string]any{
		"CallAnalyticsJobName":   job.Name,
		"CallAnalyticsJobStatus": job.Status,
		"LanguageCode":           job.LanguageCode,
		"CreationTime":           job.CreatedAt,
		"CompletionTime":         job.CompletedAt,
		"Transcript":             map[string]any{"TranscriptFileUri": "s3://stackyard-transcribe/output/" + job.Name + ".json"},
	}
}

func (s *transcribeStore) callAnalyticsJobSummary(job *transcribeJob) map[string]any {
	return map[string]any{
		"CallAnalyticsJobName":   job.Name,
		"CallAnalyticsJobStatus": job.Status,
		"LanguageCode":           job.LanguageCode,
		"CreationTime":           job.CreatedAt,
		"CompletionTime":         job.CompletedAt,
	}
}

func (s *transcribeStore) callAnalyticsCategoryPayload(category *transcribeCategory) map[string]any {
	return map[string]any{
		"CategoryName": category.Name,
		"CreateTime":   category.CreatedAt,
		"UpdateTime":   category.UpdatedAt,
		"Rules":        []any{},
	}
}

func (s *transcribeStore) languageModelPayload(model *transcribeLanguageModel) map[string]any {
	return map[string]any{
		"ModelName":        model.Name,
		"ModelStatus":      model.Status,
		"CreateTime":       model.CreatedAt,
		"LastModifiedTime": model.UpdatedAt,
	}
}

func (s *transcribeStore) sortedJobsLocked(items map[string]*transcribeJob) []*transcribeJob {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*transcribeJob, 0, len(keys))
	for _, key := range keys {
		out = append(out, items[key])
	}
	return out
}

func (s *transcribeStore) sortedCallAnalyticsCategoriesLocked() []*transcribeCategory {
	keys := make([]string, 0, len(s.callAnalyticsCategories))
	for key := range s.callAnalyticsCategories {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*transcribeCategory, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.callAnalyticsCategories[key])
	}
	return out
}

func (s *transcribeStore) sortedLanguageModelsLocked() []*transcribeLanguageModel {
	keys := make([]string, 0, len(s.languageModels))
	for key := range s.languageModels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*transcribeLanguageModel, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.languageModels[key])
	}
	return out
}

func (s *transcribeStore) sortedVocabulariesLocked(items map[string]*transcribeVocabulary) []*transcribeVocabulary {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*transcribeVocabulary, 0, len(keys))
	for _, key := range keys {
		out = append(out, items[key])
	}
	return out
}

func (s *transcribeStore) sortedVocabularyFiltersLocked() []*transcribeVocabularyFilter {
	keys := make([]string, 0, len(s.vocabularyFilters))
	for key := range s.vocabularyFilters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*transcribeVocabularyFilter, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.vocabularyFilters[key])
	}
	return out
}

func (s *transcribeStore) nextTokenLocked(prefix string) string {
	if strings.TrimSpace(prefix) == "" {
		prefix = "id"
	}
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%06d", prefix, id)
}

func transcribePayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return def
	}
	value, ok := raw.(string)
	if !ok {
		return def
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
}

func transcribePayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
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

func transcribePayloadTags(payload map[string]any, key string) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return out
	}

	switch typed := raw.(type) {
	case map[string]any:
		for k, v := range typed {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			if str, ok := v.(string); ok {
				out[k] = strings.TrimSpace(str)
			}
		}
	case []any:
		for _, item := range typed {
			tag, ok := item.(map[string]any)
			if !ok {
				continue
			}
			k := strings.TrimSpace(transcribePayloadString(tag, "Key", ""))
			if k == "" {
				continue
			}
			out[k] = transcribePayloadString(tag, "Value", "")
		}
	}

	return out
}

func transcribeARN(resourceType, name string) string {
	return fmt.Sprintf("arn:aws:transcribe:us-east-1:123456789012:%s/%s", strings.TrimSpace(resourceType), strings.TrimSpace(name))
}
