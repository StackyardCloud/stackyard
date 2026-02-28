package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type braketStore struct {
	mu sync.Mutex

	nextJobID           int64
	nextQuantumTaskID   int64
	nextSpendingLimitID int64

	jobs           map[string]map[string]any
	quantumTasks   map[string]map[string]any
	spendingLimits map[string]map[string]any
	devices        map[string]map[string]any
	tags           map[string]map[string]string
}

func newBraketStore() *braketStore {
	s := &braketStore{
		nextJobID:           2,
		nextQuantumTaskID:   2,
		nextSpendingLimitID: 2,
		jobs:                map[string]map[string]any{},
		quantumTasks:        map[string]map[string]any{},
		spendingLimits:      map[string]map[string]any{},
		devices:             map[string]map[string]any{},
		tags:                map[string]map[string]string{},
	}
	s.seedLocked(time.Now().UTC())
	return s
}

func (s *braketStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.seedLocked(now)
	ctx := braketMergeMaps(payload, pathParams, query)

	jobArn := braketString(ctx, "jobArn", "arn:aws:braket:us-east-1:123456789012:job/job-000001")
	quantumTaskArn := braketString(ctx, "quantumTaskArn", "arn:aws:braket:us-east-1:123456789012:quantum-task/task-000001")
	spendingLimitArn := braketString(ctx, "spendingLimitArn", "arn:aws:braket:us-east-1:123456789012:spending-limit/limit-000001")
	deviceArn := braketString(ctx, "deviceArn", "arn:aws:braket:us-east-1::device/qpu/test-device")
	resourceArn := braketString(ctx, "resourceArn", jobArn)

	s.ensureJobLocked(jobArn, now)
	s.ensureQuantumTaskLocked(quantumTaskArn, now)
	s.ensureSpendingLimitLocked(spendingLimitArn, now)
	s.ensureDeviceLocked(deviceArn, now)
	s.ensureTagsLocked(resourceArn)

	switch action {
	case "CreateJob":
		arn := fmt.Sprintf("arn:aws:braket:us-east-1:123456789012:job/job-%06d", s.nextJobID)
		s.nextJobID++
		job := s.ensureJobLocked(arn, now)
		return map[string]any{"jobArn": arn, "jobName": braketString(job, "jobName", "stackyard-job")}

	case "GetJob":
		return braketCloneMap(s.ensureJobLocked(jobArn, now))

	case "CancelJob":
		job := s.ensureJobLocked(jobArn, now)
		job["status"] = "CANCELLED"
		job["endedAt"] = now.Format(time.RFC3339)
		return map[string]any{"cancellationStatus": "CANCELLING", "jobArn": jobArn}

	case "SearchJobs":
		items := make([]any, 0, len(s.jobs))
		keys := make([]string, 0, len(s.jobs))
		for key := range s.jobs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			job := s.jobs[key]
			items = append(items, map[string]any{
				"jobArn":       braketString(job, "jobArn", ""),
				"jobName":      braketString(job, "jobName", ""),
				"status":       braketString(job, "status", "QUEUED"),
				"createdAt":    braketString(job, "createdAt", now.Format(time.RFC3339)),
				"device":       braketString(job, "device", ""),
				"jobType":      braketString(job, "jobType", "QUANTUM_JOB"),
				"queueInfo":    map[string]any{"position": "1", "queue": "JOBS_QUEUE"},
				"endedAt":      braketString(job, "endedAt", ""),
				"startedAt":    braketString(job, "startedAt", ""),
				"instanceType": "ml.m5.large",
			})
		}
		return map[string]any{"jobs": items, "nextToken": ""}

	case "CreateQuantumTask":
		arn := fmt.Sprintf("arn:aws:braket:us-east-1:123456789012:quantum-task/task-%06d", s.nextQuantumTaskID)
		s.nextQuantumTaskID++
		task := s.ensureQuantumTaskLocked(arn, now)
		return map[string]any{"quantumTaskArn": arn, "deviceArn": braketString(task, "deviceArn", deviceArn)}

	case "GetQuantumTask":
		return braketCloneMap(s.ensureQuantumTaskLocked(quantumTaskArn, now))

	case "CancelQuantumTask":
		task := s.ensureQuantumTaskLocked(quantumTaskArn, now)
		task["status"] = "CANCELLED"
		return map[string]any{"cancellationStatus": "CANCELLING", "quantumTaskArn": quantumTaskArn}

	case "SearchQuantumTasks":
		items := make([]any, 0, len(s.quantumTasks))
		keys := make([]string, 0, len(s.quantumTasks))
		for key := range s.quantumTasks {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			task := s.quantumTasks[key]
			items = append(items, map[string]any{
				"quantumTaskArn":    braketString(task, "quantumTaskArn", ""),
				"deviceArn":         braketString(task, "deviceArn", ""),
				"outputS3Bucket":    braketString(task, "outputS3Bucket", "stackyard-braket"),
				"outputS3Directory": braketString(task, "outputS3Directory", "quantum-tasks"),
				"shots":             10,
				"status":            braketString(task, "status", "COMPLETED"),
				"createdAt":         braketString(task, "createdAt", now.Format(time.RFC3339)),
			})
		}
		return map[string]any{"quantumTasks": items, "nextToken": ""}

	case "GetDevice":
		return braketCloneMap(s.ensureDeviceLocked(deviceArn, now))

	case "SearchDevices":
		items := make([]any, 0, len(s.devices))
		keys := make([]string, 0, len(s.devices))
		for key := range s.devices {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			device := s.devices[key]
			items = append(items, map[string]any{
				"deviceArn":          braketString(device, "deviceArn", ""),
				"deviceName":         braketString(device, "deviceName", ""),
				"deviceType":         braketString(device, "deviceType", "QPU"),
				"providerName":       braketString(device, "providerName", "Stackyard"),
				"deviceStatus":       braketString(device, "deviceStatus", "ONLINE"),
				"deviceCapabilities": braketString(device, "deviceCapabilities", "{}"),
			})
		}
		return map[string]any{"devices": items, "nextToken": ""}

	case "CreateSpendingLimit":
		arn := fmt.Sprintf("arn:aws:braket:us-east-1:123456789012:spending-limit/limit-%06d", s.nextSpendingLimitID)
		s.nextSpendingLimitID++
		limit := s.ensureSpendingLimitLocked(arn, now)
		return map[string]any{"spendingLimitArn": arn, "amount": limit["amount"]}

	case "UpdateSpendingLimit":
		limit := s.ensureSpendingLimitLocked(spendingLimitArn, now)
		if amount, ok := payload["amount"]; ok {
			limit["amount"] = amount
		}
		limit["lastUpdatedAt"] = now.Format(time.RFC3339)
		return map[string]any{"spendingLimitArn": spendingLimitArn, "amount": limit["amount"]}

	case "DeleteSpendingLimit":
		delete(s.spendingLimits, spendingLimitArn)
		return map[string]any{}

	case "SearchSpendingLimits":
		items := make([]any, 0, len(s.spendingLimits))
		keys := make([]string, 0, len(s.spendingLimits))
		for key := range s.spendingLimits {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			limit := s.spendingLimits[key]
			items = append(items, map[string]any{
				"spendingLimitArn": braketString(limit, "spendingLimitArn", ""),
				"amount":           limit["amount"],
				"currency":         braketString(limit, "currency", "USD"),
				"status":           braketString(limit, "status", "ACTIVE"),
			})
		}
		return map[string]any{"spendingLimits": items, "nextToken": ""}

	case "TagResource":
		tags := braketMapString(payload["tags"])
		if len(tags) > 0 {
			existing := s.ensureTagsLocked(resourceArn)
			for k, v := range tags {
				existing[k] = v
			}
		}
		return map[string]any{}

	case "UntagResource":
		tagKeys := braketString(ctx, "tagKeys", "env")
		existing := s.ensureTagsLocked(resourceArn)
		for _, raw := range strings.Split(tagKeys, ",") {
			key := strings.TrimSpace(raw)
			if key == "" {
				continue
			}
			delete(existing, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": braketCloneMapString(s.ensureTagsLocked(resourceArn))}
	}

	return map[string]any{}
}

func (s *braketStore) seedLocked(now time.Time) {
	s.ensureJobLocked("arn:aws:braket:us-east-1:123456789012:job/job-000001", now)
	s.ensureQuantumTaskLocked("arn:aws:braket:us-east-1:123456789012:quantum-task/task-000001", now)
	s.ensureSpendingLimitLocked("arn:aws:braket:us-east-1:123456789012:spending-limit/limit-000001", now)
	s.ensureDeviceLocked("arn:aws:braket:us-east-1::device/qpu/test-device", now)
	s.ensureTagsLocked("arn:aws:braket:us-east-1:123456789012:job/job-000001")
}

func (s *braketStore) ensureJobLocked(jobArn string, now time.Time) map[string]any {
	if job := s.jobs[jobArn]; job != nil {
		return job
	}
	job := map[string]any{
		"jobArn":           jobArn,
		"jobName":          "stackyard-job",
		"status":           "RUNNING",
		"createdAt":        now.Add(-2 * time.Minute).Format(time.RFC3339),
		"startedAt":        now.Add(-1 * time.Minute).Format(time.RFC3339),
		"endedAt":          "",
		"device":           "arn:aws:braket:us-east-1::device/qpu/test-device",
		"jobType":          "QUANTUM_JOB",
		"queueInfo":        map[string]any{"position": "1", "queue": "JOBS_QUEUE"},
		"outputDataConfig": map[string]any{"s3Path": "s3://stackyard-braket/jobs"},
	}
	s.jobs[jobArn] = job
	return job
}

func (s *braketStore) ensureQuantumTaskLocked(taskArn string, now time.Time) map[string]any {
	if task := s.quantumTasks[taskArn]; task != nil {
		return task
	}
	task := map[string]any{
		"quantumTaskArn":    taskArn,
		"deviceArn":         "arn:aws:braket:us-east-1::device/qpu/test-device",
		"status":            "COMPLETED",
		"createdAt":         now.Add(-1 * time.Minute).Format(time.RFC3339),
		"shots":             10,
		"outputS3Bucket":    "stackyard-braket",
		"outputS3Directory": "quantum-tasks",
	}
	s.quantumTasks[taskArn] = task
	return task
}

func (s *braketStore) ensureSpendingLimitLocked(limitArn string, now time.Time) map[string]any {
	if limit := s.spendingLimits[limitArn]; limit != nil {
		return limit
	}
	limit := map[string]any{
		"spendingLimitArn": limitArn,
		"amount":           1000,
		"currency":         "USD",
		"status":           "ACTIVE",
		"createdAt":        now.Add(-24 * time.Hour).Format(time.RFC3339),
		"lastUpdatedAt":    now.Format(time.RFC3339),
	}
	s.spendingLimits[limitArn] = limit
	return limit
}

func (s *braketStore) ensureDeviceLocked(deviceArn string, now time.Time) map[string]any {
	if device := s.devices[deviceArn]; device != nil {
		return device
	}
	device := map[string]any{
		"deviceArn":          deviceArn,
		"deviceName":         "stackyard-test-qpu",
		"deviceType":         "QPU",
		"providerName":       "Stackyard",
		"deviceStatus":       "ONLINE",
		"deviceCapabilities": "{}",
	}
	s.devices[deviceArn] = device
	return device
}

func (s *braketStore) ensureTagsLocked(resourceArn string) map[string]string {
	if tags, ok := s.tags[resourceArn]; ok {
		return tags
	}
	tags := map[string]string{"env": "local", "service": "braket"}
	s.tags[resourceArn] = tags
	return tags
}

func braketMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for k, v := range payload {
		out[k] = v
	}
	for k, v := range pathParams {
		out[k] = v
	}
	for k, values := range query {
		if len(values) > 0 {
			out[k] = values[len(values)-1]
		}
	}
	return out
}

func braketString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		value := strings.TrimSpace(fmt.Sprint(v))
		if value != "" {
			return value
		}
	}
	return def
}

func braketMapString(value any) map[string]string {
	out := map[string]string{}
	input, ok := value.(map[string]any)
	if !ok {
		return out
	}
	for k, raw := range input {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(fmt.Sprint(raw))
	}
	return out
}

func braketCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]any, len(keys))
	for _, key := range keys {
		out[key] = in[key]
	}
	return out
}

func braketCloneMapString(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = in[key]
	}
	return out
}
