package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	entityResolutionDefaultRegion    = "us-east-1"
	entityResolutionDefaultAccountID = "123456789012"
)

type entityResolutionStore struct {
	mu sync.Mutex

	nextID int64

	idNamespaces       map[string]map[string]any
	schemaMappings     map[string]map[string]any
	matchingWorkflows  map[string]map[string]any
	idMappingWorkflows map[string]map[string]any
	matchingJobs       map[string]map[string]map[string]any // workflow -> jobID -> job
	idMappingJobs      map[string]map[string]map[string]any // workflow -> jobID -> job
	providerServices   map[string]map[string]any            // provider/service -> summary
	policies           map[string]map[string]any            // arn -> policy object
	policyStatements   map[string]map[string]map[string]any // arn -> statementID -> statement
	tags               map[string]map[string]string
}

func newEntityResolutionStore() *entityResolutionStore {
	now := time.Now().UTC().Format(time.RFC3339)

	namespaceName := "stackyard-id-namespace"
	schemaName := "stackyard-schema"
	matchingWorkflow := "stackyard-matching-workflow"
	idMappingWorkflow := "stackyard-idmapping-workflow"
	providerName := "default-provider"
	providerServiceName := "default-service"
	resourceARN := entityResolutionNamespaceARN(namespaceName)

	s := &entityResolutionStore{
		nextID: 2,
		idNamespaces: map[string]map[string]any{
			namespaceName: {
				"idNamespaceName": namespaceName,
				"idNamespaceArn":  resourceARN,
				"description":     "Seeded id namespace for deterministic Stackyard responses",
				"createdAt":       now,
				"updatedAt":       now,
			},
		},
		schemaMappings: map[string]map[string]any{
			schemaName: {
				"schemaName": schemaName,
				"status":     "ACTIVE",
				"createdAt":  now,
				"updatedAt":  now,
			},
		},
		matchingWorkflows: map[string]map[string]any{
			matchingWorkflow: {
				"workflowName": matchingWorkflow,
				"status":       "ACTIVE",
				"createdAt":    now,
				"updatedAt":    now,
			},
		},
		idMappingWorkflows: map[string]map[string]any{
			idMappingWorkflow: {
				"workflowName": idMappingWorkflow,
				"status":       "ACTIVE",
				"createdAt":    now,
				"updatedAt":    now,
			},
		},
		matchingJobs: map[string]map[string]map[string]any{
			matchingWorkflow: {
				"job-000001": {
					"jobId":        "job-000001",
					"workflowName": matchingWorkflow,
					"status":       "SUCCEEDED",
					"startedAt":    now,
				},
			},
		},
		idMappingJobs: map[string]map[string]map[string]any{
			idMappingWorkflow: {
				"job-000001": {
					"jobId":        "job-000001",
					"workflowName": idMappingWorkflow,
					"status":       "SUCCEEDED",
					"startedAt":    now,
				},
			},
		},
		providerServices: map[string]map[string]any{
			providerName + "/" + providerServiceName: {
				"providerName":        providerName,
				"providerServiceName": providerServiceName,
				"providerServiceArn":  "arn:aws:entityresolution:us-east-1:aws:providerservice/default-provider/default-service",
				"description":         "Seeded provider service",
				"providerType":        "AWS_MARKETPLACE",
			},
		},
		policies: map[string]map[string]any{
			resourceARN: {
				"arn":       resourceARN,
				"policy":    "{\"Version\":\"2012-10-17\",\"Statement\":[]}",
				"updatedAt": now,
			},
		},
		policyStatements: map[string]map[string]map[string]any{
			resourceARN: {
				"default": {
					"statementId": "default",
					"effect":      "Allow",
					"action":      "entityresolution:*",
					"createdAt":   now,
				},
			},
		},
		tags: map[string]map[string]string{
			resourceARN: {"stackyard": "true"},
		},
	}

	return s
}

func (s *entityResolutionStore) Handle(
	action string,
	payload map[string]any,
	pathParams map[string]string,
	query url.Values,
) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateIdNamespace":
		name := entityResolutionLookupString(pathParams, payload, query, "idNamespaceName")
		if name == "" {
			name = fmt.Sprintf("id-namespace-%06d", s.nextID)
			s.nextID++
		}
		item := s.ensureIdNamespaceLocked(name, now)
		item["updatedAt"] = now
		return map[string]any{"idNamespaceName": name, "idNamespaceArn": item["idNamespaceArn"]}
	case "GetIdNamespace":
		name := entityResolutionLookupString(pathParams, payload, query, "idNamespaceName")
		item := s.ensureIdNamespaceLocked(name, now)
		return map[string]any{"idNamespace": entityResolutionCloneMap(item)}
	case "ListIdNamespaces":
		items := make([]any, 0, len(s.idNamespaces))
		for _, item := range entityResolutionSortedMapValues(s.idNamespaces, "idNamespaceName") {
			items = append(items, map[string]any{
				"idNamespaceName": item["idNamespaceName"],
				"idNamespaceArn":  item["idNamespaceArn"],
			})
		}
		return map[string]any{"idNamespaces": items, "nextToken": ""}
	case "UpdateIdNamespace":
		name := entityResolutionLookupString(pathParams, payload, query, "idNamespaceName")
		item := s.ensureIdNamespaceLocked(name, now)
		item["updatedAt"] = now
		return map[string]any{"idNamespaceName": item["idNamespaceName"], "idNamespaceArn": item["idNamespaceArn"]}
	case "DeleteIdNamespace":
		name := entityResolutionLookupString(pathParams, payload, query, "idNamespaceName")
		item := s.ensureIdNamespaceLocked(name, now)
		arn := entityResolutionMapString(item, "idNamespaceArn", entityResolutionNamespaceARN(name))
		delete(s.idNamespaces, name)
		delete(s.tags, arn)
		delete(s.policies, arn)
		delete(s.policyStatements, arn)
		return map[string]any{}

	case "CreateSchemaMapping":
		name := entityResolutionLookupString(pathParams, payload, query, "schemaName")
		if name == "" {
			name = fmt.Sprintf("schema-%06d", s.nextID)
			s.nextID++
		}
		item := s.ensureSchemaMappingLocked(name, now)
		item["updatedAt"] = now
		return map[string]any{"schemaName": item["schemaName"]}
	case "GetSchemaMapping":
		name := entityResolutionLookupString(pathParams, payload, query, "schemaName")
		item := s.ensureSchemaMappingLocked(name, now)
		return map[string]any{"schemaMapping": entityResolutionCloneMap(item)}
	case "ListSchemaMappings":
		items := make([]any, 0, len(s.schemaMappings))
		for _, item := range entityResolutionSortedMapValues(s.schemaMappings, "schemaName") {
			items = append(items, map[string]any{
				"schemaName": item["schemaName"],
				"status":     item["status"],
			})
		}
		return map[string]any{"schemaList": items, "nextToken": ""}
	case "UpdateSchemaMapping":
		name := entityResolutionLookupString(pathParams, payload, query, "schemaName")
		item := s.ensureSchemaMappingLocked(name, now)
		item["updatedAt"] = now
		return map[string]any{"schemaName": item["schemaName"]}
	case "DeleteSchemaMapping":
		name := entityResolutionLookupString(pathParams, payload, query, "schemaName")
		delete(s.schemaMappings, name)
		return map[string]any{}

	case "CreateMatchingWorkflow":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		if name == "" {
			name = fmt.Sprintf("matching-workflow-%06d", s.nextID)
			s.nextID++
		}
		item := s.ensureMatchingWorkflowLocked(name, now)
		item["updatedAt"] = now
		return map[string]any{"workflowName": item["workflowName"], "workflowArn": entityResolutionMatchingWorkflowARN(name)}
	case "GetMatchingWorkflow":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		item := s.ensureMatchingWorkflowLocked(name, now)
		return map[string]any{"workflow": entityResolutionCloneMap(item)}
	case "ListMatchingWorkflows":
		items := make([]any, 0, len(s.matchingWorkflows))
		for _, item := range entityResolutionSortedMapValues(s.matchingWorkflows, "workflowName") {
			items = append(items, map[string]any{
				"workflowName": item["workflowName"],
				"status":       item["status"],
			})
		}
		return map[string]any{"matchingWorkflows": items, "nextToken": ""}
	case "UpdateMatchingWorkflow":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		item := s.ensureMatchingWorkflowLocked(name, now)
		item["updatedAt"] = now
		return map[string]any{"workflowName": item["workflowName"], "workflowArn": entityResolutionMatchingWorkflowARN(name)}
	case "DeleteMatchingWorkflow":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		delete(s.matchingWorkflows, name)
		delete(s.matchingJobs, name)
		return map[string]any{}

	case "StartMatchingJob":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		s.ensureMatchingWorkflowLocked(name, now)
		jobs := s.ensureMatchingJobsLocked(name)
		jobID := entityResolutionLookupString(pathParams, payload, query, "jobId")
		if jobID == "" {
			jobID = fmt.Sprintf("job-%06d", s.nextID)
			s.nextID++
		}
		jobs[jobID] = map[string]any{
			"jobId":        jobID,
			"workflowName": name,
			"status":       "RUNNING",
			"startedAt":    now,
		}
		return map[string]any{"jobId": jobID}
	case "GetMatchingJob":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		jobs := s.ensureMatchingJobsLocked(name)
		jobID := entityResolutionLookupString(pathParams, payload, query, "jobId")
		job := s.ensureJobLocked(jobs, jobID, name, now)
		return map[string]any{"job": entityResolutionCloneMap(job)}
	case "ListMatchingJobs":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		jobs := s.ensureMatchingJobsLocked(name)
		items := make([]any, 0, len(jobs))
		for _, item := range entityResolutionSortedMapValues(jobs, "jobId") {
			items = append(items, map[string]any{
				"jobId":  item["jobId"],
				"status": item["status"],
			})
		}
		return map[string]any{"jobs": items, "nextToken": ""}
	case "GenerateMatchId":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		s.ensureMatchingWorkflowLocked(name, now)
		return map[string]any{
			"matchGroups": []any{
				map[string]any{
					"matchId": fmt.Sprintf("match-%06d", s.nextID),
					"records": []any{
						map[string]any{"recordId": "record-000001"},
					},
				},
			},
		}
	case "GetMatchId":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		s.ensureMatchingWorkflowLocked(name, now)
		return map[string]any{
			"matchGroups": []any{
				map[string]any{
					"matchId": "match-000001",
					"records": []any{
						map[string]any{"recordId": "record-000001"},
					},
				},
			},
		}
	case "BatchDeleteUniqueId":
		return map[string]any{
			"deleted": []any{
				map[string]any{"uniqueId": "unique-000001"},
			},
			"errors": []any{},
		}

	case "CreateIdMappingWorkflow":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		if name == "" {
			name = fmt.Sprintf("idmapping-workflow-%06d", s.nextID)
			s.nextID++
		}
		item := s.ensureIdMappingWorkflowLocked(name, now)
		item["updatedAt"] = now
		return map[string]any{"workflowName": item["workflowName"], "workflowArn": entityResolutionIdMappingWorkflowARN(name)}
	case "GetIdMappingWorkflow":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		item := s.ensureIdMappingWorkflowLocked(name, now)
		return map[string]any{"workflow": entityResolutionCloneMap(item)}
	case "ListIdMappingWorkflows":
		items := make([]any, 0, len(s.idMappingWorkflows))
		for _, item := range entityResolutionSortedMapValues(s.idMappingWorkflows, "workflowName") {
			items = append(items, map[string]any{
				"workflowName": item["workflowName"],
				"status":       item["status"],
			})
		}
		return map[string]any{"idMappingWorkflows": items, "nextToken": ""}
	case "UpdateIdMappingWorkflow":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		item := s.ensureIdMappingWorkflowLocked(name, now)
		item["updatedAt"] = now
		return map[string]any{"workflowName": item["workflowName"], "workflowArn": entityResolutionIdMappingWorkflowARN(name)}
	case "DeleteIdMappingWorkflow":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		delete(s.idMappingWorkflows, name)
		delete(s.idMappingJobs, name)
		return map[string]any{}

	case "StartIdMappingJob":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		s.ensureIdMappingWorkflowLocked(name, now)
		jobs := s.ensureIdMappingJobsLocked(name)
		jobID := entityResolutionLookupString(pathParams, payload, query, "jobId")
		if jobID == "" {
			jobID = fmt.Sprintf("job-%06d", s.nextID)
			s.nextID++
		}
		jobs[jobID] = map[string]any{
			"jobId":        jobID,
			"workflowName": name,
			"status":       "RUNNING",
			"startedAt":    now,
		}
		return map[string]any{"jobId": jobID}
	case "GetIdMappingJob":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		jobs := s.ensureIdMappingJobsLocked(name)
		jobID := entityResolutionLookupString(pathParams, payload, query, "jobId")
		job := s.ensureJobLocked(jobs, jobID, name, now)
		return map[string]any{"job": entityResolutionCloneMap(job)}
	case "ListIdMappingJobs":
		name := entityResolutionLookupString(pathParams, payload, query, "workflowName")
		jobs := s.ensureIdMappingJobsLocked(name)
		items := make([]any, 0, len(jobs))
		for _, item := range entityResolutionSortedMapValues(jobs, "jobId") {
			items = append(items, map[string]any{
				"jobId":  item["jobId"],
				"status": item["status"],
			})
		}
		return map[string]any{"jobs": items, "nextToken": ""}

	case "GetProviderService":
		providerName := entityResolutionLookupString(pathParams, payload, query, "providerName")
		providerServiceName := entityResolutionLookupString(pathParams, payload, query, "providerServiceName")
		item := s.ensureProviderServiceLocked(providerName, providerServiceName)
		return map[string]any{"providerService": entityResolutionCloneMap(item)}
	case "ListProviderServices":
		items := make([]any, 0, len(s.providerServices))
		for _, item := range entityResolutionSortedMapValues(s.providerServices, "providerServiceName") {
			items = append(items, entityResolutionCloneMap(item))
		}
		return map[string]any{"providerServiceSummaries": items, "nextToken": ""}

	case "PutPolicy":
		arn := entityResolutionLookupString(pathParams, payload, query, "arn")
		if arn == "" {
			arn = entityResolutionNamespaceARN("stackyard-id-namespace")
		}
		policyText := entityResolutionLookupString(pathParams, payload, query, "policy", "Policy")
		if policyText == "" {
			policyText = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
		}
		p := s.ensurePolicyLocked(arn, now)
		p["policy"] = policyText
		p["updatedAt"] = now
		return map[string]any{}
	case "GetPolicy":
		arn := entityResolutionLookupString(pathParams, payload, query, "arn")
		if arn == "" {
			arn = entityResolutionNamespaceARN("stackyard-id-namespace")
		}
		p := s.ensurePolicyLocked(arn, now)
		statements := s.ensurePolicyStatementsLocked(arn, now)
		outStatements := make([]any, 0, len(statements))
		for _, item := range entityResolutionSortedMapValues(statements, "statementId") {
			outStatements = append(outStatements, entityResolutionCloneMap(item))
		}
		return map[string]any{
			"arn":              arn,
			"policy":           entityResolutionMapString(p, "policy", "{}"),
			"policyStatements": outStatements,
		}
	case "AddPolicyStatement":
		arn := entityResolutionLookupString(pathParams, payload, query, "arn")
		if arn == "" {
			arn = entityResolutionNamespaceARN("stackyard-id-namespace")
		}
		statementID := entityResolutionLookupString(pathParams, payload, query, "statementId")
		if statementID == "" {
			statementID = fmt.Sprintf("stmt-%06d", s.nextID)
			s.nextID++
		}
		statements := s.ensurePolicyStatementsLocked(arn, now)
		statements[statementID] = map[string]any{
			"statementId": statementID,
			"effect":      "Allow",
			"action":      "entityresolution:Get*",
			"createdAt":   now,
		}
		return map[string]any{}
	case "DeletePolicyStatement":
		arn := entityResolutionLookupString(pathParams, payload, query, "arn")
		if arn == "" {
			arn = entityResolutionNamespaceARN("stackyard-id-namespace")
		}
		statementID := entityResolutionLookupString(pathParams, payload, query, "statementId")
		if statements, ok := s.policyStatements[arn]; ok {
			delete(statements, statementID)
		}
		return map[string]any{}

	case "TagResource":
		resourceARN := entityResolutionLookupString(pathParams, payload, query, "resourceArn")
		if resourceARN == "" {
			resourceARN = entityResolutionNamespaceARN("stackyard-id-namespace")
		}
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range entityResolutionExtractTags(payload) {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			tags[key] = value
		}
		return map[string]any{}
	case "ListTagsForResource":
		resourceARN := entityResolutionLookupString(pathParams, payload, query, "resourceArn")
		if resourceARN == "" {
			resourceARN = entityResolutionNamespaceARN("stackyard-id-namespace")
		}
		return map[string]any{"tags": entityResolutionCloneStringMap(s.ensureTagsLocked(resourceARN))}
	case "UntagResource":
		resourceARN := entityResolutionLookupString(pathParams, payload, query, "resourceArn")
		if resourceARN == "" {
			resourceARN = entityResolutionNamespaceARN("stackyard-id-namespace")
		}
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range entityResolutionExtractTagKeys(payload, query) {
			delete(tags, key)
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *entityResolutionStore) ensureIdNamespaceLocked(name, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-id-namespace"
	}
	if item, ok := s.idNamespaces[name]; ok {
		return item
	}
	item := map[string]any{
		"idNamespaceName": name,
		"idNamespaceArn":  entityResolutionNamespaceARN(name),
		"description":     "Generated id namespace",
		"createdAt":       now,
		"updatedAt":       now,
	}
	s.idNamespaces[name] = item
	s.ensureTagsLocked(entityResolutionNamespaceARN(name))["stackyard"] = "true"
	return item
}

func (s *entityResolutionStore) ensureSchemaMappingLocked(name, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-schema"
	}
	if item, ok := s.schemaMappings[name]; ok {
		return item
	}
	item := map[string]any{
		"schemaName": name,
		"status":     "ACTIVE",
		"createdAt":  now,
		"updatedAt":  now,
	}
	s.schemaMappings[name] = item
	return item
}

func (s *entityResolutionStore) ensureMatchingWorkflowLocked(name, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-matching-workflow"
	}
	if item, ok := s.matchingWorkflows[name]; ok {
		return item
	}
	item := map[string]any{
		"workflowName": name,
		"workflowArn":  entityResolutionMatchingWorkflowARN(name),
		"status":       "ACTIVE",
		"createdAt":    now,
		"updatedAt":    now,
	}
	s.matchingWorkflows[name] = item
	return item
}

func (s *entityResolutionStore) ensureIdMappingWorkflowLocked(name, now string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-idmapping-workflow"
	}
	if item, ok := s.idMappingWorkflows[name]; ok {
		return item
	}
	item := map[string]any{
		"workflowName": name,
		"workflowArn":  entityResolutionIdMappingWorkflowARN(name),
		"status":       "ACTIVE",
		"createdAt":    now,
		"updatedAt":    now,
	}
	s.idMappingWorkflows[name] = item
	return item
}

func (s *entityResolutionStore) ensureMatchingJobsLocked(workflowName string) map[string]map[string]any {
	workflowName = strings.TrimSpace(workflowName)
	if workflowName == "" {
		workflowName = "stackyard-matching-workflow"
	}
	if jobs, ok := s.matchingJobs[workflowName]; ok {
		return jobs
	}
	jobs := map[string]map[string]any{}
	s.matchingJobs[workflowName] = jobs
	return jobs
}

func (s *entityResolutionStore) ensureIdMappingJobsLocked(workflowName string) map[string]map[string]any {
	workflowName = strings.TrimSpace(workflowName)
	if workflowName == "" {
		workflowName = "stackyard-idmapping-workflow"
	}
	if jobs, ok := s.idMappingJobs[workflowName]; ok {
		return jobs
	}
	jobs := map[string]map[string]any{}
	s.idMappingJobs[workflowName] = jobs
	return jobs
}

func (s *entityResolutionStore) ensureJobLocked(jobs map[string]map[string]any, jobID, workflowName, now string) map[string]any {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		jobID = "job-000001"
	}
	if job, ok := jobs[jobID]; ok {
		return job
	}
	job := map[string]any{
		"jobId":        jobID,
		"workflowName": workflowName,
		"status":       "SUCCEEDED",
		"startedAt":    now,
	}
	jobs[jobID] = job
	return job
}

func (s *entityResolutionStore) ensureProviderServiceLocked(providerName, providerServiceName string) map[string]any {
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		providerName = "default-provider"
	}
	providerServiceName = strings.TrimSpace(providerServiceName)
	if providerServiceName == "" {
		providerServiceName = "default-service"
	}
	key := providerName + "/" + providerServiceName
	if item, ok := s.providerServices[key]; ok {
		return item
	}
	item := map[string]any{
		"providerName":        providerName,
		"providerServiceName": providerServiceName,
		"providerServiceArn":  "arn:aws:entityresolution:us-east-1:aws:providerservice/" + providerName + "/" + providerServiceName,
		"description":         "Generated provider service",
		"providerType":        "AWS_MARKETPLACE",
	}
	s.providerServices[key] = item
	return item
}

func (s *entityResolutionStore) ensurePolicyLocked(arn, now string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = entityResolutionNamespaceARN("stackyard-id-namespace")
	}
	if item, ok := s.policies[arn]; ok {
		return item
	}
	item := map[string]any{
		"arn":       arn,
		"policy":    "{\"Version\":\"2012-10-17\",\"Statement\":[]}",
		"updatedAt": now,
	}
	s.policies[arn] = item
	return item
}

func (s *entityResolutionStore) ensurePolicyStatementsLocked(arn, now string) map[string]map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = entityResolutionNamespaceARN("stackyard-id-namespace")
	}
	if statements, ok := s.policyStatements[arn]; ok {
		return statements
	}
	statements := map[string]map[string]any{
		"default": {
			"statementId": "default",
			"effect":      "Allow",
			"action":      "entityresolution:*",
			"createdAt":   now,
		},
	}
	s.policyStatements[arn] = statements
	return statements
}

func (s *entityResolutionStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = entityResolutionNamespaceARN("stackyard-id-namespace")
	}
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{}
	s.tags[resourceARN] = tags
	return tags
}

func entityResolutionNamespaceARN(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-id-namespace"
	}
	return fmt.Sprintf(
		"arn:aws:entityresolution:%s:%s:idnamespace/%s",
		entityResolutionDefaultRegion,
		entityResolutionDefaultAccountID,
		name,
	)
}

func entityResolutionMatchingWorkflowARN(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-matching-workflow"
	}
	return fmt.Sprintf(
		"arn:aws:entityresolution:%s:%s:matchingworkflow/%s",
		entityResolutionDefaultRegion,
		entityResolutionDefaultAccountID,
		name,
	)
}

func entityResolutionIdMappingWorkflowARN(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-idmapping-workflow"
	}
	return fmt.Sprintf(
		"arn:aws:entityresolution:%s:%s:idmappingworkflow/%s",
		entityResolutionDefaultRegion,
		entityResolutionDefaultAccountID,
		name,
	)
}

func entityResolutionLookupString(pathParams map[string]string, payload map[string]any, query url.Values, keys ...string) string {
	for _, key := range keys {
		for _, candidate := range []string{key, entityResolutionLowerFirst(key), entityResolutionUpperFirst(key)} {
			if pathParams != nil {
				if value := strings.TrimSpace(pathParams[candidate]); value != "" {
					return value
				}
			}
			if payload != nil {
				if raw, ok := payload[candidate]; ok {
					if value, ok := raw.(string); ok && strings.TrimSpace(value) != "" {
						return strings.TrimSpace(value)
					}
				}
			}
			if query != nil {
				if value := strings.TrimSpace(query.Get(candidate)); value != "" {
					return value
				}
			}
		}
	}
	return ""
}

func entityResolutionMapString(item map[string]any, key, def string) string {
	if item == nil {
		return def
	}
	raw, ok := item[key]
	if !ok || raw == nil {
		return def
	}
	if value, ok := raw.(string); ok {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return def
}

func entityResolutionExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}

	for _, field := range []string{"tags", "Tags"} {
		raw, ok := payload[field]
		if !ok || raw == nil {
			continue
		}
		switch typed := raw.(type) {
		case map[string]any:
			for key, value := range typed {
				if s, ok := value.(string); ok {
					out[strings.TrimSpace(key)] = strings.TrimSpace(s)
				}
			}
		case map[string]string:
			for key, value := range typed {
				out[strings.TrimSpace(key)] = strings.TrimSpace(value)
			}
		case []any:
			for _, item := range typed {
				entry, ok := item.(map[string]any)
				if !ok {
					continue
				}
				key, _ := entry["key"].(string)
				if key == "" {
					key, _ = entry["Key"].(string)
				}
				value, _ := entry["value"].(string)
				if value == "" {
					value, _ = entry["Value"].(string)
				}
				key = strings.TrimSpace(key)
				if key != "" {
					out[key] = strings.TrimSpace(value)
				}
			}
		}
	}

	return out
}

func entityResolutionExtractTagKeys(payload map[string]any, query url.Values) []string {
	keys := map[string]struct{}{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		keys[value] = struct{}{}
	}

	if query != nil {
		for _, field := range []string{"tagKeys", "TagKeys", "tagKey", "TagKey"} {
			for _, value := range query[field] {
				for _, split := range strings.Split(value, ",") {
					add(split)
				}
			}
		}
	}

	if payload != nil {
		for _, field := range []string{"tagKeys", "TagKeys"} {
			raw, ok := payload[field]
			if !ok || raw == nil {
				continue
			}
			switch typed := raw.(type) {
			case string:
				for _, split := range strings.Split(typed, ",") {
					add(split)
				}
			case []any:
				for _, value := range typed {
					if s, ok := value.(string); ok {
						add(s)
					}
				}
			case []string:
				for _, value := range typed {
					add(value)
				}
			}
		}
	}

	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func entityResolutionSortedMapValues[T ~map[string]any](items map[string]T, key string) []map[string]any {
	keys := make([]string, 0, len(items))
	for itemKey := range items {
		keys = append(keys, itemKey)
	}
	sort.Strings(keys)

	out := make([]map[string]any, 0, len(keys))
	for _, itemKey := range keys {
		value := map[string]any(items[itemKey])
		if _, ok := value[key]; !ok {
			value[key] = itemKey
		}
		out = append(out, value)
	}
	return out
}

func entityResolutionCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = entityResolutionCloneMap(typed)
		case []any:
			copied := make([]any, len(typed))
			copy(copied, typed)
			out[key] = copied
		default:
			out[key] = value
		}
	}
	return out
}

func entityResolutionCloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func entityResolutionLowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func entityResolutionUpperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
