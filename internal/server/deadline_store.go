package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type deadlineStore struct {
	mu sync.Mutex

	nextFarmID           int64
	nextFleetID          int64
	nextQueueID          int64
	nextWorkerID         int64
	nextMonitorID        int64
	nextBudgetID         int64
	nextStorageProfileID int64
	nextLicenseID        int64
	nextJobID            int64
	nextStepID           int64
	nextTaskID           int64
	nextSessionID        int64
	nextQueueEnvID       int64
	nextLimitID          int64

	farms            map[string]map[string]any
	fleets           map[string]map[string]any
	queues           map[string]map[string]any
	workers          map[string]map[string]any
	monitors         map[string]map[string]any
	budgets          map[string]map[string]any
	storageProfiles  map[string]map[string]any
	licenseEndpoints map[string]map[string]any
	jobs             map[string]map[string]any
	steps            map[string]map[string]any
	tasks            map[string]map[string]any
	sessions         map[string]map[string]any
	queueEnvs        map[string]map[string]any
	limits           map[string]map[string]any
	tags             map[string]map[string]string
}

func newDeadlineStore() *deadlineStore {
	s := &deadlineStore{
		nextFarmID:           2,
		nextFleetID:          2,
		nextQueueID:          2,
		nextWorkerID:         2,
		nextMonitorID:        2,
		nextBudgetID:         2,
		nextStorageProfileID: 2,
		nextLicenseID:        2,
		nextJobID:            2,
		nextStepID:           2,
		nextTaskID:           2,
		nextSessionID:        2,
		nextQueueEnvID:       2,
		nextLimitID:          2,
		farms:                map[string]map[string]any{},
		fleets:               map[string]map[string]any{},
		queues:               map[string]map[string]any{},
		workers:              map[string]map[string]any{},
		monitors:             map[string]map[string]any{},
		budgets:              map[string]map[string]any{},
		storageProfiles:      map[string]map[string]any{},
		licenseEndpoints:     map[string]map[string]any{},
		jobs:                 map[string]map[string]any{},
		steps:                map[string]map[string]any{},
		tasks:                map[string]map[string]any{},
		sessions:             map[string]map[string]any{},
		queueEnvs:            map[string]map[string]any{},
		limits:               map[string]map[string]any{},
		tags:                 map[string]map[string]string{},
	}

	farm := s.ensureFarmLocked("farm-00000001")
	fleet := s.ensureFleetLocked(mhFirstNonEmpty(mhStringAny(farm, "farmId"), "farm-00000001"), "fleet-00000001")
	queue := s.ensureQueueLocked(mhFirstNonEmpty(mhStringAny(farm, "farmId"), "farm-00000001"), "queue-00000001")
	worker := s.ensureWorkerLocked(
		mhFirstNonEmpty(mhStringAny(farm, "farmId"), "farm-00000001"),
		mhFirstNonEmpty(mhStringAny(fleet, "fleetId"), "fleet-00000001"),
		"worker-00000001",
	)
	job := s.ensureJobLocked(
		mhFirstNonEmpty(mhStringAny(farm, "farmId"), "farm-00000001"),
		mhFirstNonEmpty(mhStringAny(queue, "queueId"), "queue-00000001"),
		"job-00000001",
	)
	step := s.ensureStepLocked(
		mhFirstNonEmpty(mhStringAny(farm, "farmId"), "farm-00000001"),
		mhFirstNonEmpty(mhStringAny(queue, "queueId"), "queue-00000001"),
		mhFirstNonEmpty(mhStringAny(job, "jobId"), "job-00000001"),
		"step-00000001",
	)
	task := s.ensureTaskLocked(
		mhFirstNonEmpty(mhStringAny(farm, "farmId"), "farm-00000001"),
		mhFirstNonEmpty(mhStringAny(queue, "queueId"), "queue-00000001"),
		mhFirstNonEmpty(mhStringAny(job, "jobId"), "job-00000001"),
		mhFirstNonEmpty(mhStringAny(step, "stepId"), "step-00000001"),
		"task-00000001",
	)
	session := s.ensureSessionLocked(
		mhFirstNonEmpty(mhStringAny(farm, "farmId"), "farm-00000001"),
		mhFirstNonEmpty(mhStringAny(queue, "queueId"), "queue-00000001"),
		mhFirstNonEmpty(mhStringAny(job, "jobId"), "job-00000001"),
		"session-00000001",
	)

	s.ensureMonitorLocked("monitor-00000001")
	s.ensureBudgetLocked(mhFirstNonEmpty(mhStringAny(farm, "farmId"), "farm-00000001"), "budget-00000001")
	s.ensureStorageProfileLocked(mhFirstNonEmpty(mhStringAny(farm, "farmId"), "farm-00000001"), "sp-00000001")
	s.ensureLicenseEndpointLocked("lic-00000001")
	s.ensureQueueEnvironmentLocked(mhFirstNonEmpty(mhStringAny(farm, "farmId"), "farm-00000001"), mhFirstNonEmpty(mhStringAny(queue, "queueId"), "queue-00000001"), "qenv-00000001")
	s.ensureLimitLocked(mhFirstNonEmpty(mhStringAny(farm, "farmId"), "farm-00000001"), "limit-00000001")
	s.tags[mhFirstNonEmpty(mhStringAny(session, "arn"), mhDeadlineARN("session", "session-00000001"))] = map[string]string{"seed": "true"}
	s.tags[mhFirstNonEmpty(mhStringAny(task, "arn"), mhDeadlineARN("task", "task-00000001"))] = map[string]string{"seed": "true"}
	s.tags[mhFirstNonEmpty(mhStringAny(worker, "arn"), mhDeadlineARN("worker", "worker-00000001"))] = map[string]string{"seed": "true"}

	return s
}

func (s *deadlineStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	farmID := mhFirstNonEmpty(mhPathParam(pathParams, "farmId"), mhStringAny(payload, "farmId", "farmID"), "farm-00000001")
	fleetID := mhFirstNonEmpty(mhPathParam(pathParams, "fleetId"), mhStringAny(payload, "fleetId", "fleetID"), "fleet-00000001")
	queueID := mhFirstNonEmpty(mhPathParam(pathParams, "queueId"), mhStringAny(payload, "queueId", "queueID"), "queue-00000001")
	jobID := mhFirstNonEmpty(mhPathParam(pathParams, "jobId"), mhStringAny(payload, "jobId", "jobID"), "job-00000001")
	workerID := mhFirstNonEmpty(mhPathParam(pathParams, "workerId"), mhStringAny(payload, "workerId", "workerID"), "worker-00000001")
	stepID := mhFirstNonEmpty(mhPathParam(pathParams, "stepId"), mhStringAny(payload, "stepId", "stepID"), "step-00000001")
	taskID := mhFirstNonEmpty(mhPathParam(pathParams, "taskId"), mhStringAny(payload, "taskId", "taskID"), "task-00000001")
	sessionID := mhFirstNonEmpty(mhPathParam(pathParams, "sessionId"), mhStringAny(payload, "sessionId", "sessionID"), "session-00000001")
	monitorID := mhFirstNonEmpty(mhPathParam(pathParams, "monitorId"), mhStringAny(payload, "monitorId", "monitorID"), "monitor-00000001")
	budgetID := mhFirstNonEmpty(mhPathParam(pathParams, "budgetId"), mhStringAny(payload, "budgetId", "budgetID"), "budget-00000001")
	storageProfileID := mhFirstNonEmpty(mhPathParam(pathParams, "storageProfileId"), mhStringAny(payload, "storageProfileId", "storageProfileID"), "sp-00000001")
	licenseEndpointID := mhFirstNonEmpty(mhPathParam(pathParams, "licenseEndpointId"), mhStringAny(payload, "licenseEndpointId", "licenseEndpointID"), "lic-00000001")
	queueEnvID := mhFirstNonEmpty(mhPathParam(pathParams, "queueEnvironmentId"), mhStringAny(payload, "queueEnvironmentId", "queueEnvironmentID"), "qenv-00000001")
	limitID := mhFirstNonEmpty(mhPathParam(pathParams, "limitId"), mhStringAny(payload, "limitId", "limitID"), "limit-00000001")
	resourceARN := mhFirstNonEmpty(mhPathParam(pathParams, "resourceArn"), mhStringAny(payload, "resourceArn", "resourceARN"), mhDeadlineARN("farm", farmID))
	sessionActionID := mhFirstNonEmpty(mhPathParam(pathParams, "sessionActionId"), mhStringAny(payload, "sessionActionId"), "session-action-00000001")
	principalID := mhFirstNonEmpty(mhPathParam(pathParams, "principalId"), mhStringAny(payload, "principalId"), "AIDAEXAMPLE")
	productID := mhFirstNonEmpty(mhPathParam(pathParams, "productId"), mhStringAny(payload, "productId"), "product-00000001")

	switch action {
	case "CreateFarm":
		farmID = fmt.Sprintf("farm-%08d", s.nextFarmIDLocked())
		farm := s.ensureFarmLocked(farmID)
		farm["displayName"] = mhFirstNonEmpty(mhStringAny(payload, "displayName", "name"), fmt.Sprintf("stackyard-farm-%s", farmID))
		farm["description"] = mhFirstNonEmpty(mhStringAny(payload, "description"), "Stackyard Deadline farm")
		farm["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"farmId": farmID, "farmArn": mhStringAny(farm, "arn")}

	case "GetFarm":
		return mhCloneMap(s.ensureFarmLocked(farmID))
	case "ListFarms":
		return map[string]any{"farms": s.listByTypeLocked("farm"), "nextToken": ""}
	case "UpdateFarm":
		farm := s.ensureFarmLocked(farmID)
		for k, v := range payload {
			farm[k] = v
		}
		farm["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"farmId": farmID}
	case "DeleteFarm":
		delete(s.farms, farmID)
		return map[string]any{"farmId": farmID}

	case "CreateFleet":
		s.ensureFarmLocked(farmID)
		fleetID = fmt.Sprintf("fleet-%08d", s.nextFleetIDLocked())
		fleet := s.ensureFleetLocked(farmID, fleetID)
		fleet["displayName"] = mhFirstNonEmpty(mhStringAny(payload, "displayName", "name"), fmt.Sprintf("stackyard-fleet-%s", fleetID))
		fleet["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"fleetId": fleetID, "fleetArn": mhStringAny(fleet, "arn")}
	case "GetFleet":
		return mhCloneMap(s.ensureFleetLocked(farmID, fleetID))
	case "ListFleets":
		return map[string]any{"fleets": s.listByFarmTypeLocked("fleet", farmID), "nextToken": ""}
	case "UpdateFleet":
		fleet := s.ensureFleetLocked(farmID, fleetID)
		for k, v := range payload {
			fleet[k] = v
		}
		fleet["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"fleetId": fleetID}
	case "DeleteFleet":
		delete(s.fleets, fleetID)
		return map[string]any{"fleetId": fleetID}

	case "CreateQueue":
		s.ensureFarmLocked(farmID)
		queueID = fmt.Sprintf("queue-%08d", s.nextQueueIDLocked())
		queue := s.ensureQueueLocked(farmID, queueID)
		queue["displayName"] = mhFirstNonEmpty(mhStringAny(payload, "displayName", "name"), fmt.Sprintf("stackyard-queue-%s", queueID))
		queue["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"queueId": queueID, "queueArn": mhStringAny(queue, "arn")}
	case "GetQueue":
		return mhCloneMap(s.ensureQueueLocked(farmID, queueID))
	case "ListQueues":
		return map[string]any{"queues": s.listByFarmTypeLocked("queue", farmID), "nextToken": ""}
	case "UpdateQueue":
		queue := s.ensureQueueLocked(farmID, queueID)
		for k, v := range payload {
			queue[k] = v
		}
		queue["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"queueId": queueID}
	case "DeleteQueue":
		delete(s.queues, queueID)
		return map[string]any{"queueId": queueID}

	case "CreateWorker":
		s.ensureFleetLocked(farmID, fleetID)
		workerID = fmt.Sprintf("worker-%08d", s.nextWorkerIDLocked())
		worker := s.ensureWorkerLocked(farmID, fleetID, workerID)
		worker["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"workerId": workerID, "workerArn": mhStringAny(worker, "arn")}
	case "GetWorker":
		return mhCloneMap(s.ensureWorkerLocked(farmID, fleetID, workerID))
	case "ListWorkers":
		return map[string]any{"workers": s.listWorkersByFleetLocked(farmID, fleetID), "nextToken": ""}
	case "UpdateWorker", "UpdateWorkerSchedule":
		worker := s.ensureWorkerLocked(farmID, fleetID, workerID)
		for k, v := range payload {
			worker[k] = v
		}
		worker["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"workerId": workerID}
	case "DeleteWorker":
		delete(s.workers, workerID)
		return map[string]any{"workerId": workerID}

	case "CreateMonitor":
		monitorID = fmt.Sprintf("monitor-%08d", s.nextMonitorIDLocked())
		monitor := s.ensureMonitorLocked(monitorID)
		monitor["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"monitorId": monitorID, "monitorArn": mhStringAny(monitor, "arn")}
	case "GetMonitor":
		return mhCloneMap(s.ensureMonitorLocked(monitorID))
	case "ListMonitors":
		return map[string]any{"monitors": s.listByTypeLocked("monitor"), "nextToken": ""}
	case "UpdateMonitor":
		monitor := s.ensureMonitorLocked(monitorID)
		for k, v := range payload {
			monitor[k] = v
		}
		monitor["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"monitorId": monitorID}
	case "DeleteMonitor":
		delete(s.monitors, monitorID)
		return map[string]any{"monitorId": monitorID}

	case "CreateBudget":
		budgetID = fmt.Sprintf("budget-%08d", s.nextBudgetIDLocked())
		budget := s.ensureBudgetLocked(farmID, budgetID)
		budget["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"budgetId": budgetID, "budgetArn": mhStringAny(budget, "arn")}
	case "GetBudget":
		return mhCloneMap(s.ensureBudgetLocked(farmID, budgetID))
	case "ListBudgets":
		return map[string]any{"budgets": s.listByFarmTypeLocked("budget", farmID), "nextToken": ""}
	case "UpdateBudget":
		budget := s.ensureBudgetLocked(farmID, budgetID)
		for k, v := range payload {
			budget[k] = v
		}
		budget["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"budgetId": budgetID}
	case "DeleteBudget":
		delete(s.budgets, budgetID)
		return map[string]any{"budgetId": budgetID}

	case "CreateStorageProfile":
		storageProfileID = fmt.Sprintf("sp-%08d", s.nextStorageProfileIDLocked())
		sp := s.ensureStorageProfileLocked(farmID, storageProfileID)
		sp["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"storageProfileId": storageProfileID}
	case "GetStorageProfile", "GetStorageProfileForQueue":
		return mhCloneMap(s.ensureStorageProfileLocked(farmID, storageProfileID))
	case "ListStorageProfiles", "ListStorageProfilesForQueue":
		return map[string]any{"storageProfiles": s.listByFarmTypeLocked("storageProfile", farmID), "nextToken": ""}
	case "UpdateStorageProfile":
		sp := s.ensureStorageProfileLocked(farmID, storageProfileID)
		for k, v := range payload {
			sp[k] = v
		}
		sp["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"storageProfileId": storageProfileID}
	case "DeleteStorageProfile":
		delete(s.storageProfiles, storageProfileID)
		return map[string]any{"storageProfileId": storageProfileID}

	case "CreateLicenseEndpoint":
		licenseEndpointID = fmt.Sprintf("lic-%08d", s.nextLicenseIDLocked())
		lic := s.ensureLicenseEndpointLocked(licenseEndpointID)
		lic["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"licenseEndpointId": licenseEndpointID}
	case "GetLicenseEndpoint":
		return mhCloneMap(s.ensureLicenseEndpointLocked(licenseEndpointID))
	case "ListLicenseEndpoints":
		return map[string]any{"licenseEndpoints": s.listByTypeLocked("licenseEndpoint"), "nextToken": ""}
	case "DeleteLicenseEndpoint":
		delete(s.licenseEndpoints, licenseEndpointID)
		return map[string]any{"licenseEndpointId": licenseEndpointID}
	case "ListAvailableMeteredProducts":
		return map[string]any{"meteredProducts": []any{map[string]any{"productId": productID, "name": "stackyard-product"}}, "nextToken": ""}
	case "ListMeteredProducts":
		return map[string]any{"meteredProducts": []any{map[string]any{"productId": productID, "licenseEndpointId": licenseEndpointID}}, "nextToken": ""}
	case "PutMeteredProduct":
		return map[string]any{"licenseEndpointId": licenseEndpointID, "productId": productID}
	case "DeleteMeteredProduct":
		return map[string]any{"licenseEndpointId": licenseEndpointID, "productId": productID}

	case "CreateQueueEnvironment":
		queueEnvID = fmt.Sprintf("qenv-%08d", s.nextQueueEnvIDLocked())
		qe := s.ensureQueueEnvironmentLocked(farmID, queueID, queueEnvID)
		qe["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"queueEnvironmentId": queueEnvID}
	case "GetQueueEnvironment":
		return mhCloneMap(s.ensureQueueEnvironmentLocked(farmID, queueID, queueEnvID))
	case "ListQueueEnvironments":
		return map[string]any{"queueEnvironments": s.listQueueEnvironmentsLocked(farmID, queueID), "nextToken": ""}
	case "UpdateQueueEnvironment":
		qe := s.ensureQueueEnvironmentLocked(farmID, queueID, queueEnvID)
		for k, v := range payload {
			qe[k] = v
		}
		qe["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"queueEnvironmentId": queueEnvID}
	case "DeleteQueueEnvironment":
		delete(s.queueEnvs, queueEnvID)
		return map[string]any{"queueEnvironmentId": queueEnvID}

	case "CreateLimit":
		limitID = fmt.Sprintf("limit-%08d", s.nextLimitIDLocked())
		limit := s.ensureLimitLocked(farmID, limitID)
		limit["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"limitId": limitID}
	case "GetLimit":
		return mhCloneMap(s.ensureLimitLocked(farmID, limitID))
	case "ListLimits":
		return map[string]any{"limits": s.listByFarmTypeLocked("limit", farmID), "nextToken": ""}
	case "UpdateLimit":
		limit := s.ensureLimitLocked(farmID, limitID)
		for k, v := range payload {
			limit[k] = v
		}
		limit["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"limitId": limitID}
	case "DeleteLimit":
		delete(s.limits, limitID)
		return map[string]any{"limitId": limitID}

	case "CreateJob":
		s.ensureQueueLocked(farmID, queueID)
		jobID = fmt.Sprintf("job-%08d", s.nextJobIDLocked())
		job := s.ensureJobLocked(farmID, queueID, jobID)
		job["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"jobId": jobID, "jobArn": mhStringAny(job, "arn")}
	case "GetJob":
		return mhCloneMap(s.ensureJobLocked(farmID, queueID, jobID))
	case "ListJobs":
		return map[string]any{"jobs": s.listJobsLocked(farmID, queueID), "nextToken": ""}
	case "UpdateJob":
		job := s.ensureJobLocked(farmID, queueID, jobID)
		for k, v := range payload {
			job[k] = v
		}
		job["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"jobId": jobID}
	case "SearchJobs":
		return map[string]any{"jobs": s.listJobsLocked(farmID, queueID), "nextToken": ""}

	case "GetStep":
		return mhCloneMap(s.ensureStepLocked(farmID, queueID, jobID, stepID))
	case "ListSteps":
		return map[string]any{"steps": s.listStepsLocked(farmID, queueID, jobID), "nextToken": ""}
	case "UpdateStep":
		step := s.ensureStepLocked(farmID, queueID, jobID, stepID)
		for k, v := range payload {
			step[k] = v
		}
		step["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"stepId": stepID}
	case "SearchSteps":
		return map[string]any{"steps": s.listStepsLocked(farmID, queueID, jobID), "nextToken": ""}
	case "ListStepDependencies":
		return map[string]any{"dependencies": []any{}, "nextToken": ""}
	case "ListStepConsumers":
		return map[string]any{"consumers": []any{}, "nextToken": ""}

	case "GetTask":
		return mhCloneMap(s.ensureTaskLocked(farmID, queueID, jobID, stepID, taskID))
	case "ListTasks":
		return map[string]any{"tasks": s.listTasksLocked(farmID, queueID, jobID, stepID), "nextToken": ""}
	case "UpdateTask":
		task := s.ensureTaskLocked(farmID, queueID, jobID, stepID, taskID)
		for k, v := range payload {
			task[k] = v
		}
		task["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"taskId": taskID}
	case "SearchTasks":
		return map[string]any{"tasks": s.listTasksLocked(farmID, queueID, jobID, stepID), "nextToken": ""}

	case "GetSession":
		return mhCloneMap(s.ensureSessionLocked(farmID, queueID, jobID, sessionID))
	case "ListSessions":
		return map[string]any{"sessions": s.listSessionsLocked(farmID, queueID, jobID), "nextToken": ""}
	case "UpdateSession":
		session := s.ensureSessionLocked(farmID, queueID, jobID, sessionID)
		for k, v := range payload {
			session[k] = v
		}
		session["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"sessionId": sessionID}
	case "ListSessionsForWorker":
		return map[string]any{"sessions": s.listSessionsForWorkerLocked(workerID), "nextToken": ""}
	case "GetSessionAction":
		return map[string]any{"sessionActionId": sessionActionID, "status": "READY"}
	case "ListSessionActions":
		return map[string]any{"sessionActions": []any{map[string]any{"sessionActionId": sessionActionID, "status": "READY"}}, "nextToken": ""}
	case "StartSessionsStatisticsAggregation":
		return map[string]any{"status": "STARTED", "farmId": farmID}
	case "GetSessionsStatisticsAggregation":
		return map[string]any{"status": "COMPLETED", "farmId": farmID, "statistics": map[string]any{"runningSessions": 1}}

	case "AssociateMemberToFarm", "DisassociateMemberFromFarm":
		return map[string]any{"farmId": farmID, "principalId": principalID}
	case "AssociateMemberToFleet", "DisassociateMemberFromFleet":
		return map[string]any{"farmId": farmID, "fleetId": fleetID, "principalId": principalID}
	case "AssociateMemberToQueue", "DisassociateMemberFromQueue":
		return map[string]any{"farmId": farmID, "queueId": queueID, "principalId": principalID}
	case "AssociateMemberToJob", "DisassociateMemberFromJob":
		return map[string]any{"farmId": farmID, "queueId": queueID, "jobId": jobID, "principalId": principalID}

	case "ListFarmMembers":
		return map[string]any{"members": []any{map[string]any{"principalId": principalID, "membershipLevel": "VIEWER"}}, "nextToken": ""}
	case "ListFleetMembers":
		return map[string]any{"members": []any{map[string]any{"principalId": principalID, "membershipLevel": "VIEWER"}}, "nextToken": ""}
	case "ListQueueMembers":
		return map[string]any{"members": []any{map[string]any{"principalId": principalID, "membershipLevel": "VIEWER"}}, "nextToken": ""}
	case "ListJobMembers":
		return map[string]any{"members": []any{map[string]any{"principalId": principalID, "membershipLevel": "VIEWER"}}, "nextToken": ""}
	case "ListJobParameterDefinitions":
		return map[string]any{"jobParameterDefinitions": []any{}, "nextToken": ""}

	case "AssumeFleetRoleForRead", "AssumeFleetRoleForWorker", "AssumeQueueRoleForRead", "AssumeQueueRoleForUser", "AssumeQueueRoleForWorker":
		return map[string]any{
			"credentials": map[string]any{
				"accessKeyId":     "stackyard",
				"secretAccessKey": "stackyard",
				"sessionToken":    "stackyard",
				"expiration":      time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
			},
		}

	case "CreateQueueFleetAssociation", "GetQueueFleetAssociation", "UpdateQueueFleetAssociation", "DeleteQueueFleetAssociation":
		return map[string]any{"farmId": farmID, "queueId": queueID, "fleetId": fleetID}
	case "ListQueueFleetAssociations":
		return map[string]any{"queueFleetAssociations": []any{map[string]any{"farmId": farmID, "queueId": queueID, "fleetId": fleetID}}, "nextToken": ""}
	case "CreateQueueLimitAssociation", "GetQueueLimitAssociation", "UpdateQueueLimitAssociation", "DeleteQueueLimitAssociation":
		return map[string]any{"farmId": farmID, "queueId": queueID, "limitId": limitID}
	case "ListQueueLimitAssociations":
		return map[string]any{"queueLimitAssociations": []any{map[string]any{"farmId": farmID, "queueId": queueID, "limitId": limitID}}, "nextToken": ""}

	case "BatchGetJobEntity":
		return map[string]any{"entities": []any{s.ensureJobLocked(farmID, queueID, jobID)}, "errors": []any{}}
	case "CopyJobTemplate":
		return map[string]any{"jobId": jobID, "template": map[string]any{"specificationVersion": "jobtemplate-2023-09", "name": "stackyard-job-template"}}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for k, v := range mhStringMapAny(payload, "tags", "Tags") {
			tags[k] = v
		}
		return map[string]any{"resourceArn": resourceARN, "tags": mhCloneStringMap(tags)}
	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range mhStringSliceAny(payload, "tagKeys", "TagKeys") {
			delete(tags, key)
		}
		for _, key := range query["tagKeys"] {
			if strings.TrimSpace(key) != "" {
				delete(tags, strings.TrimSpace(key))
			}
		}
		return map[string]any{"resourceArn": resourceARN, "tags": mhCloneStringMap(tags)}
	case "ListTagsForResource":
		return map[string]any{"tags": mhCloneStringMap(s.ensureTagsLocked(resourceARN))}

	case "SearchWorkers":
		return map[string]any{"workers": s.listWorkersByFleetLocked(farmID, fleetID), "nextToken": ""}
	}

	return map[string]any{}
}

func (s *deadlineStore) nextFarmIDLocked() int64 {
	id := s.nextFarmID
	s.nextFarmID++
	return id
}

func (s *deadlineStore) nextFleetIDLocked() int64 {
	id := s.nextFleetID
	s.nextFleetID++
	return id
}

func (s *deadlineStore) nextQueueIDLocked() int64 {
	id := s.nextQueueID
	s.nextQueueID++
	return id
}

func (s *deadlineStore) nextWorkerIDLocked() int64 {
	id := s.nextWorkerID
	s.nextWorkerID++
	return id
}

func (s *deadlineStore) nextMonitorIDLocked() int64 {
	id := s.nextMonitorID
	s.nextMonitorID++
	return id
}

func (s *deadlineStore) nextBudgetIDLocked() int64 {
	id := s.nextBudgetID
	s.nextBudgetID++
	return id
}

func (s *deadlineStore) nextStorageProfileIDLocked() int64 {
	id := s.nextStorageProfileID
	s.nextStorageProfileID++
	return id
}

func (s *deadlineStore) nextLicenseIDLocked() int64 {
	id := s.nextLicenseID
	s.nextLicenseID++
	return id
}

func (s *deadlineStore) nextJobIDLocked() int64 {
	id := s.nextJobID
	s.nextJobID++
	return id
}

func (s *deadlineStore) nextStepIDLocked() int64 {
	id := s.nextStepID
	s.nextStepID++
	return id
}

func (s *deadlineStore) nextTaskIDLocked() int64 {
	id := s.nextTaskID
	s.nextTaskID++
	return id
}

func (s *deadlineStore) nextSessionIDLocked() int64 {
	id := s.nextSessionID
	s.nextSessionID++
	return id
}

func (s *deadlineStore) nextQueueEnvIDLocked() int64 {
	id := s.nextQueueEnvID
	s.nextQueueEnvID++
	return id
}

func (s *deadlineStore) nextLimitIDLocked() int64 {
	id := s.nextLimitID
	s.nextLimitID++
	return id
}

func mhDeadlineARN(kind, id string) string {
	return fmt.Sprintf("arn:aws:deadline:us-east-1:123456789012:%s/%s", kind, id)
}

func (s *deadlineStore) ensureFarmLocked(farmID string) map[string]any {
	farmID = mhFirstNonEmpty(strings.TrimSpace(farmID), "farm-00000001")
	if farm, ok := s.farms[farmID]; ok {
		return farm
	}
	now := time.Now().UTC().Format(time.RFC3339)
	farm := map[string]any{
		"type":        "farm",
		"farmId":      farmID,
		"arn":         mhDeadlineARN("farm", farmID),
		"displayName": fmt.Sprintf("stackyard-farm-%s", farmID),
		"createdAt":   now,
		"updatedAt":   now,
	}
	s.farms[farmID] = farm
	return farm
}

func (s *deadlineStore) ensureFleetLocked(farmID, fleetID string) map[string]any {
	s.ensureFarmLocked(farmID)
	fleetID = mhFirstNonEmpty(strings.TrimSpace(fleetID), "fleet-00000001")
	if fleet, ok := s.fleets[fleetID]; ok {
		return fleet
	}
	now := time.Now().UTC().Format(time.RFC3339)
	fleet := map[string]any{
		"type":        "fleet",
		"farmId":      farmID,
		"fleetId":     fleetID,
		"arn":         mhDeadlineARN("fleet", fleetID),
		"displayName": fmt.Sprintf("stackyard-fleet-%s", fleetID),
		"status":      "ACTIVE",
		"createdAt":   now,
		"updatedAt":   now,
	}
	s.fleets[fleetID] = fleet
	return fleet
}

func (s *deadlineStore) ensureQueueLocked(farmID, queueID string) map[string]any {
	s.ensureFarmLocked(farmID)
	queueID = mhFirstNonEmpty(strings.TrimSpace(queueID), "queue-00000001")
	if queue, ok := s.queues[queueID]; ok {
		return queue
	}
	now := time.Now().UTC().Format(time.RFC3339)
	queue := map[string]any{
		"type":        "queue",
		"farmId":      farmID,
		"queueId":     queueID,
		"arn":         mhDeadlineARN("queue", queueID),
		"displayName": fmt.Sprintf("stackyard-queue-%s", queueID),
		"status":      "ACTIVE",
		"createdAt":   now,
		"updatedAt":   now,
	}
	s.queues[queueID] = queue
	return queue
}

func (s *deadlineStore) ensureWorkerLocked(farmID, fleetID, workerID string) map[string]any {
	s.ensureFleetLocked(farmID, fleetID)
	workerID = mhFirstNonEmpty(strings.TrimSpace(workerID), "worker-00000001")
	if worker, ok := s.workers[workerID]; ok {
		return worker
	}
	now := time.Now().UTC().Format(time.RFC3339)
	worker := map[string]any{
		"type":      "worker",
		"farmId":    farmID,
		"fleetId":   fleetID,
		"workerId":  workerID,
		"arn":       mhDeadlineARN("worker", workerID),
		"status":    "ACTIVE",
		"createdAt": now,
		"updatedAt": now,
	}
	s.workers[workerID] = worker
	return worker
}

func (s *deadlineStore) ensureMonitorLocked(monitorID string) map[string]any {
	monitorID = mhFirstNonEmpty(strings.TrimSpace(monitorID), "monitor-00000001")
	if monitor, ok := s.monitors[monitorID]; ok {
		return monitor
	}
	now := time.Now().UTC().Format(time.RFC3339)
	monitor := map[string]any{
		"type":      "monitor",
		"monitorId": monitorID,
		"arn":       mhDeadlineARN("monitor", monitorID),
		"status":    "ACTIVE",
		"createdAt": now,
		"updatedAt": now,
	}
	s.monitors[monitorID] = monitor
	return monitor
}

func (s *deadlineStore) ensureBudgetLocked(farmID, budgetID string) map[string]any {
	s.ensureFarmLocked(farmID)
	budgetID = mhFirstNonEmpty(strings.TrimSpace(budgetID), "budget-00000001")
	if budget, ok := s.budgets[budgetID]; ok {
		return budget
	}
	now := time.Now().UTC().Format(time.RFC3339)
	budget := map[string]any{
		"type":      "budget",
		"farmId":    farmID,
		"budgetId":  budgetID,
		"arn":       mhDeadlineARN("budget", budgetID),
		"status":    "ACTIVE",
		"createdAt": now,
		"updatedAt": now,
	}
	s.budgets[budgetID] = budget
	return budget
}

func (s *deadlineStore) ensureStorageProfileLocked(farmID, storageProfileID string) map[string]any {
	s.ensureFarmLocked(farmID)
	storageProfileID = mhFirstNonEmpty(strings.TrimSpace(storageProfileID), "sp-00000001")
	if sp, ok := s.storageProfiles[storageProfileID]; ok {
		return sp
	}
	now := time.Now().UTC().Format(time.RFC3339)
	sp := map[string]any{
		"type":             "storageProfile",
		"farmId":           farmID,
		"storageProfileId": storageProfileID,
		"arn":              mhDeadlineARN("storage-profile", storageProfileID),
		"createdAt":        now,
		"updatedAt":        now,
	}
	s.storageProfiles[storageProfileID] = sp
	return sp
}

func (s *deadlineStore) ensureLicenseEndpointLocked(licenseEndpointID string) map[string]any {
	licenseEndpointID = mhFirstNonEmpty(strings.TrimSpace(licenseEndpointID), "lic-00000001")
	if lic, ok := s.licenseEndpoints[licenseEndpointID]; ok {
		return lic
	}
	now := time.Now().UTC().Format(time.RFC3339)
	lic := map[string]any{
		"type":              "licenseEndpoint",
		"licenseEndpointId": licenseEndpointID,
		"arn":               mhDeadlineARN("license-endpoint", licenseEndpointID),
		"status":            "ACTIVE",
		"createdAt":         now,
		"updatedAt":         now,
	}
	s.licenseEndpoints[licenseEndpointID] = lic
	return lic
}

func (s *deadlineStore) ensureJobLocked(farmID, queueID, jobID string) map[string]any {
	s.ensureQueueLocked(farmID, queueID)
	jobID = mhFirstNonEmpty(strings.TrimSpace(jobID), "job-00000001")
	if job, ok := s.jobs[jobID]; ok {
		return job
	}
	now := time.Now().UTC().Format(time.RFC3339)
	job := map[string]any{
		"type":      "job",
		"farmId":    farmID,
		"queueId":   queueID,
		"jobId":     jobID,
		"arn":       mhDeadlineARN("job", jobID),
		"status":    "READY",
		"createdAt": now,
		"updatedAt": now,
	}
	s.jobs[jobID] = job
	return job
}

func (s *deadlineStore) ensureStepLocked(farmID, queueID, jobID, stepID string) map[string]any {
	s.ensureJobLocked(farmID, queueID, jobID)
	stepID = mhFirstNonEmpty(strings.TrimSpace(stepID), "step-00000001")
	if step, ok := s.steps[stepID]; ok {
		return step
	}
	now := time.Now().UTC().Format(time.RFC3339)
	step := map[string]any{
		"type":      "step",
		"farmId":    farmID,
		"queueId":   queueID,
		"jobId":     jobID,
		"stepId":    stepID,
		"arn":       mhDeadlineARN("step", stepID),
		"status":    "READY",
		"createdAt": now,
		"updatedAt": now,
	}
	s.steps[stepID] = step
	return step
}

func (s *deadlineStore) ensureTaskLocked(farmID, queueID, jobID, stepID, taskID string) map[string]any {
	s.ensureStepLocked(farmID, queueID, jobID, stepID)
	taskID = mhFirstNonEmpty(strings.TrimSpace(taskID), "task-00000001")
	if task, ok := s.tasks[taskID]; ok {
		return task
	}
	now := time.Now().UTC().Format(time.RFC3339)
	task := map[string]any{
		"type":      "task",
		"farmId":    farmID,
		"queueId":   queueID,
		"jobId":     jobID,
		"stepId":    stepID,
		"taskId":    taskID,
		"arn":       mhDeadlineARN("task", taskID),
		"status":    "READY",
		"createdAt": now,
		"updatedAt": now,
	}
	s.tasks[taskID] = task
	return task
}

func (s *deadlineStore) ensureSessionLocked(farmID, queueID, jobID, sessionID string) map[string]any {
	s.ensureJobLocked(farmID, queueID, jobID)
	sessionID = mhFirstNonEmpty(strings.TrimSpace(sessionID), "session-00000001")
	if session, ok := s.sessions[sessionID]; ok {
		return session
	}
	now := time.Now().UTC().Format(time.RFC3339)
	session := map[string]any{
		"type":      "session",
		"farmId":    farmID,
		"queueId":   queueID,
		"jobId":     jobID,
		"sessionId": sessionID,
		"arn":       mhDeadlineARN("session", sessionID),
		"status":    "READY",
		"createdAt": now,
		"updatedAt": now,
	}
	s.sessions[sessionID] = session
	return session
}

func (s *deadlineStore) ensureQueueEnvironmentLocked(farmID, queueID, queueEnvironmentID string) map[string]any {
	s.ensureQueueLocked(farmID, queueID)
	queueEnvironmentID = mhFirstNonEmpty(strings.TrimSpace(queueEnvironmentID), "qenv-00000001")
	if qe, ok := s.queueEnvs[queueEnvironmentID]; ok {
		return qe
	}
	now := time.Now().UTC().Format(time.RFC3339)
	qe := map[string]any{
		"type":               "queueEnvironment",
		"farmId":             farmID,
		"queueId":            queueID,
		"queueEnvironmentId": queueEnvironmentID,
		"arn":                mhDeadlineARN("queue-environment", queueEnvironmentID),
		"status":             "ACTIVE",
		"createdAt":          now,
		"updatedAt":          now,
	}
	s.queueEnvs[queueEnvironmentID] = qe
	return qe
}

func (s *deadlineStore) ensureLimitLocked(farmID, limitID string) map[string]any {
	s.ensureFarmLocked(farmID)
	limitID = mhFirstNonEmpty(strings.TrimSpace(limitID), "limit-00000001")
	if limit, ok := s.limits[limitID]; ok {
		return limit
	}
	now := time.Now().UTC().Format(time.RFC3339)
	limit := map[string]any{
		"type":      "limit",
		"farmId":    farmID,
		"limitId":   limitID,
		"arn":       mhDeadlineARN("limit", limitID),
		"status":    "ACTIVE",
		"createdAt": now,
		"updatedAt": now,
	}
	s.limits[limitID] = limit
	return limit
}

func (s *deadlineStore) listByTypeLocked(kind string) []any {
	items := []any{}
	var src map[string]map[string]any
	switch kind {
	case "farm":
		src = s.farms
	case "monitor":
		src = s.monitors
	case "licenseEndpoint":
		src = s.licenseEndpoints
	}
	keys := mhSortedKeys(src)
	for _, key := range keys {
		items = append(items, mhCloneMap(src[key]))
	}
	return items
}

func (s *deadlineStore) listByFarmTypeLocked(kind, farmID string) []any {
	items := []any{}
	var src map[string]map[string]any
	switch kind {
	case "fleet":
		src = s.fleets
	case "queue":
		src = s.queues
	case "budget":
		src = s.budgets
	case "storageProfile":
		src = s.storageProfiles
	case "limit":
		src = s.limits
	}
	keys := mhSortedKeys(src)
	for _, key := range keys {
		item := src[key]
		if mhStringAny(item, "farmId") != farmID {
			continue
		}
		items = append(items, mhCloneMap(item))
	}
	return items
}

func (s *deadlineStore) listWorkersByFleetLocked(farmID, fleetID string) []any {
	items := []any{}
	keys := mhSortedKeys(s.workers)
	for _, key := range keys {
		item := s.workers[key]
		if mhStringAny(item, "farmId") != farmID || mhStringAny(item, "fleetId") != fleetID {
			continue
		}
		items = append(items, mhCloneMap(item))
	}
	return items
}

func (s *deadlineStore) listJobsLocked(farmID, queueID string) []any {
	items := []any{}
	keys := mhSortedKeys(s.jobs)
	for _, key := range keys {
		item := s.jobs[key]
		if mhStringAny(item, "farmId") != farmID || mhStringAny(item, "queueId") != queueID {
			continue
		}
		items = append(items, mhCloneMap(item))
	}
	return items
}

func (s *deadlineStore) listStepsLocked(farmID, queueID, jobID string) []any {
	items := []any{}
	keys := mhSortedKeys(s.steps)
	for _, key := range keys {
		item := s.steps[key]
		if mhStringAny(item, "farmId") != farmID || mhStringAny(item, "queueId") != queueID || mhStringAny(item, "jobId") != jobID {
			continue
		}
		items = append(items, mhCloneMap(item))
	}
	return items
}

func (s *deadlineStore) listTasksLocked(farmID, queueID, jobID, stepID string) []any {
	items := []any{}
	keys := mhSortedKeys(s.tasks)
	for _, key := range keys {
		item := s.tasks[key]
		if mhStringAny(item, "farmId") != farmID || mhStringAny(item, "queueId") != queueID || mhStringAny(item, "jobId") != jobID || mhStringAny(item, "stepId") != stepID {
			continue
		}
		items = append(items, mhCloneMap(item))
	}
	return items
}

func (s *deadlineStore) listSessionsLocked(farmID, queueID, jobID string) []any {
	items := []any{}
	keys := mhSortedKeys(s.sessions)
	for _, key := range keys {
		item := s.sessions[key]
		if mhStringAny(item, "farmId") != farmID || mhStringAny(item, "queueId") != queueID || mhStringAny(item, "jobId") != jobID {
			continue
		}
		items = append(items, mhCloneMap(item))
	}
	return items
}

func (s *deadlineStore) listSessionsForWorkerLocked(workerID string) []any {
	items := []any{}
	keys := mhSortedKeys(s.sessions)
	for _, key := range keys {
		item := s.sessions[key]
		items = append(items, mhCloneMap(item))
	}
	if len(items) == 0 {
		return []any{map[string]any{"workerId": workerID}}
	}
	return items
}

func (s *deadlineStore) listQueueEnvironmentsLocked(farmID, queueID string) []any {
	items := []any{}
	keys := mhSortedKeys(s.queueEnvs)
	for _, key := range keys {
		item := s.queueEnvs[key]
		if mhStringAny(item, "farmId") != farmID || mhStringAny(item, "queueId") != queueID {
			continue
		}
		items = append(items, mhCloneMap(item))
	}
	return items
}

func (s *deadlineStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = mhDeadlineARN("farm", "farm-00000001")
	}
	if s.tags[resourceARN] == nil {
		s.tags[resourceARN] = map[string]string{}
	}
	return s.tags[resourceARN]
}

func mhPathParam(pathParams map[string]string, keys ...string) string {
	for _, key := range keys {
		if pathParams == nil {
			break
		}
		if value, ok := pathParams[key]; ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mhStringAny(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if m == nil {
			break
		}
		value, ok := m[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		case fmt.Stringer:
			s := strings.TrimSpace(v.String())
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func mhFirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func mhCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch v := value.(type) {
		case map[string]any:
			out[key] = mhCloneMap(v)
		case []any:
			copied := make([]any, len(v))
			for i := range v {
				item := v[i]
				if m, ok := item.(map[string]any); ok {
					copied[i] = mhCloneMap(m)
					continue
				}
				copied[i] = item
			}
			out[key] = copied
		default:
			out[key] = v
		}
	}
	return out
}

func mhSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mhStringMapAny(payload map[string]any, keys ...string) map[string]string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case map[string]string:
			out := make(map[string]string, len(typed))
			for k, v := range typed {
				out[k] = v
			}
			return out
		case map[string]any:
			out := map[string]string{}
			for k, v := range typed {
				s := strings.TrimSpace(fmt.Sprint(v))
				if s != "" {
					out[k] = s
				}
			}
			return out
		}
	}
	return map[string]string{}
}

func mhStringSliceAny(payload map[string]any, keys ...string) []string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case []string:
			out := make([]string, 0, len(typed))
			for _, value := range typed {
				if trimmed := strings.TrimSpace(value); trimmed != "" {
					out = append(out, trimmed)
				}
			}
			return out
		case []any:
			out := make([]string, 0, len(typed))
			for _, value := range typed {
				trimmed := strings.TrimSpace(fmt.Sprint(value))
				if trimmed != "" {
					out = append(out, trimmed)
				}
			}
			return out
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				return []string{trimmed}
			}
		}
	}
	return nil
}

func mhCloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
