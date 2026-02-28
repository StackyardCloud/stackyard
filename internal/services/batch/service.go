package batch

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrAlreadyExists    = errors.New("resource already exists")
	ErrNotFound         = errors.New("resource not found")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"
)

type ComputeEnvironmentOrder struct {
	Order              int32
	ComputeEnvironment string
}

type ComputeEnvironment struct {
	Name           string
	ARN            string
	Type           string
	State          string
	Status         string
	StatusReason   string
	ServiceRole    string
	UnmanagedVCPUs int32
	Tags           map[string]string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type JobQueue struct {
	Name                    string
	ARN                     string
	State                   string
	Status                  string
	StatusReason            string
	Priority                int32
	ComputeEnvironmentOrder []ComputeEnvironmentOrder
	SchedulingPolicyARN     string
	Tags                    map[string]string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type SchedulingPolicy struct {
	Name      string
	ARN       string
	Tags      map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type JobDefinition struct {
	Name       string
	ARN        string
	Revision   int32
	Type       string
	Status     string
	Parameters map[string]string
	Tags       map[string]string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Job struct {
	ID           string
	ARN          string
	Name         string
	Queue        string
	Definition   string
	Status       string
	StatusReason string
	Parameters   map[string]string
	Consumables  []ConsumableResourceRequirement
	Quantity     int64
	ShareID      string
	Tags         map[string]string
	CreatedAt    time.Time
	StartedAt    *time.Time
	StoppedAt    *time.Time
}

type ConsumableResourceRequirement struct {
	ConsumableResource string
	Quantity           int64
}

type ConsumableResource struct {
	Name          string
	ARN           string
	ResourceType  string
	TotalQuantity int64
	InUseQuantity int64
	Tags          map[string]string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ServiceEnvironmentCapacity struct {
	CapacityUnit string
	MaxCapacity  int64
}

type ServiceEnvironment struct {
	Name           string
	ARN            string
	Type           string
	State          string
	Status         string
	CapacityLimits []ServiceEnvironmentCapacity
	Tags           map[string]string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ServiceJobEvaluateOnExit struct {
	Action         string
	OnStatusReason string
}

type ServiceJobRetryStrategy struct {
	Attempts       int32
	EvaluateOnExit []ServiceJobEvaluateOnExit
}

type ServiceJobTimeout struct {
	AttemptDurationSeconds int64
}

type ServiceJobAttempt struct {
	ServiceResourceName  string
	ServiceResourceValue string
	StartedAt            *time.Time
	StoppedAt            *time.Time
	StatusReason         string
}

type ServiceJob struct {
	ID                string
	ARN               string
	Name              string
	Queue             string
	Status            string
	StatusReason      string
	ServiceJobType    string
	ServiceReqPayload string
	ShareID           string
	SchedulingPrio    int32
	RetryStrategy     ServiceJobRetryStrategy
	TimeoutConfig     ServiceJobTimeout
	Attempts          []ServiceJobAttempt
	IsTerminated      bool
	Tags              map[string]string
	CreatedAt         time.Time
	StartedAt         *time.Time
	StoppedAt         *time.Time
}

type Service struct {
	mu                 sync.Mutex
	seq                uint64
	computeEnvs        map[string]*ComputeEnvironment
	jobQueues          map[string]*JobQueue
	schedulingPolicies map[string]*SchedulingPolicy
	jobDefsByName      map[string][]*JobDefinition
	jobDefsByARN       map[string]*JobDefinition
	jobs               map[string]*Job
	consumables        map[string]*ConsumableResource
	serviceEnvs        map[string]*ServiceEnvironment
	serviceJobs        map[string]*ServiceJob
}

func NewService() *Service {
	return &Service{
		computeEnvs:        map[string]*ComputeEnvironment{},
		jobQueues:          map[string]*JobQueue{},
		schedulingPolicies: map[string]*SchedulingPolicy{},
		jobDefsByName:      map[string][]*JobDefinition{},
		jobDefsByARN:       map[string]*JobDefinition{},
		jobs:               map[string]*Job{},
		consumables:        map[string]*ConsumableResource{},
		serviceEnvs:        map[string]*ServiceEnvironment{},
		serviceJobs:        map[string]*ServiceJob{},
	}
}

func (s *Service) CreateComputeEnvironment(name, ceType, state string, unmanagedVCPUs int32, serviceRole string, tags map[string]string) (ComputeEnvironment, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ComputeEnvironment{}, ErrInvalidParameter
	}
	ceType = normalizeType(ceType, "MANAGED")
	state = normalizeType(state, "ENABLED")

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.computeEnvs[name]; ok {
		return ComputeEnvironment{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	ce := &ComputeEnvironment{
		Name:           name,
		ARN:            computeEnvironmentARN(name),
		Type:           ceType,
		State:          state,
		Status:         "VALID",
		StatusReason:   "",
		ServiceRole:    strings.TrimSpace(serviceRole),
		UnmanagedVCPUs: unmanagedVCPUs,
		Tags:           cloneStringMap(tags),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.computeEnvs[name] = ce
	return cloneComputeEnvironment(ce), nil
}

func (s *Service) DescribeComputeEnvironments(ids []string) []ComputeEnvironment {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(ids) == 0 {
		out := make([]ComputeEnvironment, 0, len(s.computeEnvs))
		for _, ce := range s.computeEnvs {
			out = append(out, cloneComputeEnvironment(ce))
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out
	}
	seen := map[string]struct{}{}
	out := make([]ComputeEnvironment, 0, len(ids))
	for _, id := range ids {
		if ce := s.resolveComputeEnvironmentLocked(id); ce != nil {
			if _, ok := seen[ce.Name]; ok {
				continue
			}
			seen[ce.Name] = struct{}{}
			out = append(out, cloneComputeEnvironment(ce))
		}
	}
	return out
}

func (s *Service) UpdateComputeEnvironment(id string, state *string, unmanagedVCPUs *int32, serviceRole *string) (ComputeEnvironment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ce := s.resolveComputeEnvironmentLocked(id)
	if ce == nil {
		return ComputeEnvironment{}, ErrNotFound
	}
	if state != nil && strings.TrimSpace(*state) != "" {
		ce.State = normalizeType(*state, ce.State)
	}
	if unmanagedVCPUs != nil {
		ce.UnmanagedVCPUs = *unmanagedVCPUs
	}
	if serviceRole != nil {
		ce.ServiceRole = strings.TrimSpace(*serviceRole)
	}
	ce.UpdatedAt = time.Now().UTC()
	return cloneComputeEnvironment(ce), nil
}

func (s *Service) DeleteComputeEnvironment(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ce := s.resolveComputeEnvironmentLocked(id)
	if ce == nil {
		return ErrNotFound
	}
	delete(s.computeEnvs, ce.Name)
	return nil
}

func (s *Service) CreateJobQueue(name string, priority int32, state string, ceOrder []ComputeEnvironmentOrder, schedulingPolicyARN string, tags map[string]string) (JobQueue, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(ceOrder) == 0 {
		return JobQueue{}, ErrInvalidParameter
	}
	state = normalizeType(state, "ENABLED")
	if priority == 0 {
		priority = 1
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobQueues[name]; ok {
		return JobQueue{}, ErrAlreadyExists
	}
	resolvedOrder := make([]ComputeEnvironmentOrder, 0, len(ceOrder))
	for _, ord := range ceOrder {
		ce := s.resolveComputeEnvironmentLocked(ord.ComputeEnvironment)
		if ce == nil {
			return JobQueue{}, ErrNotFound
		}
		resolvedOrder = append(resolvedOrder, ComputeEnvironmentOrder{Order: ord.Order, ComputeEnvironment: ce.ARN})
	}
	sort.Slice(resolvedOrder, func(i, j int) bool {
		if resolvedOrder[i].Order == resolvedOrder[j].Order {
			return resolvedOrder[i].ComputeEnvironment < resolvedOrder[j].ComputeEnvironment
		}
		return resolvedOrder[i].Order < resolvedOrder[j].Order
	})
	if schedulingPolicyARN != "" && s.resolveSchedulingPolicyLocked(schedulingPolicyARN) == nil {
		return JobQueue{}, ErrNotFound
	}
	now := time.Now().UTC()
	jq := &JobQueue{
		Name:                    name,
		ARN:                     jobQueueARN(name),
		State:                   state,
		Status:                  "VALID",
		StatusReason:            "",
		Priority:                priority,
		ComputeEnvironmentOrder: resolvedOrder,
		SchedulingPolicyARN:     strings.TrimSpace(schedulingPolicyARN),
		Tags:                    cloneStringMap(tags),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	s.jobQueues[name] = jq
	return cloneJobQueue(jq), nil
}

func (s *Service) DescribeJobQueues(ids []string) []JobQueue {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(ids) == 0 {
		out := make([]JobQueue, 0, len(s.jobQueues))
		for _, jq := range s.jobQueues {
			out = append(out, cloneJobQueue(jq))
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out
	}
	seen := map[string]struct{}{}
	out := make([]JobQueue, 0, len(ids))
	for _, id := range ids {
		if jq := s.resolveJobQueueLocked(id); jq != nil {
			if _, ok := seen[jq.Name]; ok {
				continue
			}
			seen[jq.Name] = struct{}{}
			out = append(out, cloneJobQueue(jq))
		}
	}
	return out
}

func (s *Service) UpdateJobQueue(id string, state *string, priority *int32, ceOrder []ComputeEnvironmentOrder, schedulingPolicyARN *string) (JobQueue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	jq := s.resolveJobQueueLocked(id)
	if jq == nil {
		return JobQueue{}, ErrNotFound
	}
	if state != nil && strings.TrimSpace(*state) != "" {
		jq.State = normalizeType(*state, jq.State)
	}
	if priority != nil {
		jq.Priority = *priority
	}
	if ceOrder != nil {
		if len(ceOrder) == 0 {
			return JobQueue{}, ErrInvalidParameter
		}
		resolvedOrder := make([]ComputeEnvironmentOrder, 0, len(ceOrder))
		for _, ord := range ceOrder {
			ce := s.resolveComputeEnvironmentLocked(ord.ComputeEnvironment)
			if ce == nil {
				return JobQueue{}, ErrNotFound
			}
			resolvedOrder = append(resolvedOrder, ComputeEnvironmentOrder{Order: ord.Order, ComputeEnvironment: ce.ARN})
		}
		sort.Slice(resolvedOrder, func(i, j int) bool {
			if resolvedOrder[i].Order == resolvedOrder[j].Order {
				return resolvedOrder[i].ComputeEnvironment < resolvedOrder[j].ComputeEnvironment
			}
			return resolvedOrder[i].Order < resolvedOrder[j].Order
		})
		jq.ComputeEnvironmentOrder = resolvedOrder
	}
	if schedulingPolicyARN != nil {
		if strings.TrimSpace(*schedulingPolicyARN) == "" {
			jq.SchedulingPolicyARN = ""
		} else if sp := s.resolveSchedulingPolicyLocked(*schedulingPolicyARN); sp == nil {
			return JobQueue{}, ErrNotFound
		} else {
			jq.SchedulingPolicyARN = sp.ARN
		}
	}
	jq.UpdatedAt = time.Now().UTC()
	return cloneJobQueue(jq), nil
}

func (s *Service) DeleteJobQueue(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	jq := s.resolveJobQueueLocked(id)
	if jq == nil {
		return ErrNotFound
	}
	delete(s.jobQueues, jq.Name)
	return nil
}

func (s *Service) CreateSchedulingPolicy(name string, tags map[string]string) (SchedulingPolicy, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return SchedulingPolicy{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.schedulingPolicies[name]; ok {
		return SchedulingPolicy{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	sp := &SchedulingPolicy{
		Name:      name,
		ARN:       schedulingPolicyARN(name),
		Tags:      cloneStringMap(tags),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.schedulingPolicies[name] = sp
	return cloneSchedulingPolicy(sp), nil
}

func (s *Service) DescribeSchedulingPolicies(ids []string) []SchedulingPolicy {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(ids) == 0 {
		out := make([]SchedulingPolicy, 0, len(s.schedulingPolicies))
		for _, sp := range s.schedulingPolicies {
			out = append(out, cloneSchedulingPolicy(sp))
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out
	}
	seen := map[string]struct{}{}
	out := make([]SchedulingPolicy, 0, len(ids))
	for _, id := range ids {
		if sp := s.resolveSchedulingPolicyLocked(id); sp != nil {
			if _, ok := seen[sp.Name]; ok {
				continue
			}
			seen[sp.Name] = struct{}{}
			out = append(out, cloneSchedulingPolicy(sp))
		}
	}
	return out
}

func (s *Service) UpdateSchedulingPolicy(id string, tags map[string]string) (SchedulingPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp := s.resolveSchedulingPolicyLocked(id)
	if sp == nil {
		return SchedulingPolicy{}, ErrNotFound
	}
	if tags != nil {
		for k, v := range tags {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			sp.Tags[k] = strings.TrimSpace(v)
		}
	}
	sp.UpdatedAt = time.Now().UTC()
	return cloneSchedulingPolicy(sp), nil
}

func (s *Service) DeleteSchedulingPolicy(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp := s.resolveSchedulingPolicyLocked(id)
	if sp == nil {
		return ErrNotFound
	}
	delete(s.schedulingPolicies, sp.Name)
	return nil
}

func (s *Service) RegisterJobDefinition(name, jdType string, parameters, tags map[string]string) (JobDefinition, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return JobDefinition{}, ErrInvalidParameter
	}
	jdType = normalizeType(jdType, "container")

	s.mu.Lock()
	defer s.mu.Unlock()
	revision := int32(len(s.jobDefsByName[name]) + 1)
	now := time.Now().UTC()
	jd := &JobDefinition{
		Name:       name,
		Revision:   revision,
		ARN:        jobDefinitionARN(name, revision),
		Type:       jdType,
		Status:     "ACTIVE",
		Parameters: cloneStringMap(parameters),
		Tags:       cloneStringMap(tags),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.jobDefsByName[name] = append(s.jobDefsByName[name], jd)
	s.jobDefsByARN[jd.ARN] = jd
	return cloneJobDefinition(jd), nil
}

func (s *Service) DescribeJobDefinitions(ids []string, status string) []JobDefinition {
	s.mu.Lock()
	defer s.mu.Unlock()
	status = normalizeType(status, "")
	include := func(jd *JobDefinition) bool {
		return status == "" || strings.EqualFold(jd.Status, status)
	}

	out := make([]JobDefinition, 0)
	if len(ids) == 0 {
		for _, list := range s.jobDefsByName {
			for _, jd := range list {
				if include(jd) {
					out = append(out, cloneJobDefinition(jd))
				}
			}
		}
	} else {
		seen := map[string]struct{}{}
		for _, id := range ids {
			if jd := s.resolveJobDefinitionLocked(id); jd != nil {
				if _, ok := seen[jd.ARN]; ok {
					continue
				}
				if include(jd) {
					seen[jd.ARN] = struct{}{}
					out = append(out, cloneJobDefinition(jd))
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Revision < out[j].Revision
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) DeregisterJobDefinition(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	jd := s.resolveJobDefinitionLocked(id)
	if jd == nil {
		return ErrNotFound
	}
	jd.Status = "INACTIVE"
	jd.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Service) SubmitJob(name, queue, definition string, parameters, tags map[string]string) (Job, error) {
	return s.SubmitJobWithOptions(name, queue, definition, parameters, tags, nil, "", 0)
}

func (s *Service) SubmitJobWithOptions(name, queue, definition string, parameters, tags map[string]string, consumables []ConsumableResourceRequirement, shareID string, quantity int64) (Job, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.TrimSpace(queue) == "" || strings.TrimSpace(definition) == "" {
		return Job{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	jq := s.resolveJobQueueLocked(queue)
	if jq == nil {
		return Job{}, ErrNotFound
	}
	jd := s.resolveJobDefinitionLocked(definition)
	if jd == nil || !strings.EqualFold(jd.Status, "ACTIVE") {
		return Job{}, ErrNotFound
	}
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	jobID := fmt.Sprintf("job-%06d", seq)
	job := &Job{
		ID:           jobID,
		ARN:          jobARN(jobID),
		Name:         name,
		Queue:        jq.ARN,
		Definition:   jd.ARN,
		Status:       "SUBMITTED",
		StatusReason: "",
		Parameters:   cloneStringMap(parameters),
		Consumables:  cloneConsumableRequirements(consumables),
		Quantity:     quantity,
		ShareID:      strings.TrimSpace(shareID),
		Tags:         cloneStringMap(tags),
		CreatedAt:    now,
	}
	s.jobs[jobID] = job
	return cloneJob(job), nil
}

func (s *Service) DescribeJobs(ids []string) []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(ids) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]Job, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		job := s.jobs[id]
		if job == nil {
			continue
		}
		if _, ok := seen[job.ID]; ok {
			continue
		}
		seen[job.ID] = struct{}{}
		out = append(out, cloneJob(job))
	}
	return out
}

func (s *Service) ListJobs(queue, status string) []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	queue = strings.TrimSpace(queue)
	status = normalizeType(status, "")

	queueARN := queue
	if jq := s.resolveJobQueueLocked(queue); jq != nil {
		queueARN = jq.ARN
	}

	out := make([]Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		if queue != "" && !strings.EqualFold(job.Queue, queue) && !strings.EqualFold(job.Queue, queueARN) {
			continue
		}
		if status != "" && !strings.EqualFold(job.Status, status) {
			continue
		}
		out = append(out, cloneJob(job))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *Service) CancelJob(id, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[strings.TrimSpace(id)]
	if job == nil {
		return ErrNotFound
	}
	if isTerminalJobStatus(job.Status) {
		return nil
	}
	now := time.Now().UTC()
	job.Status = "FAILED"
	job.StatusReason = strings.TrimSpace(reason)
	job.StoppedAt = &now
	return nil
}

func (s *Service) TerminateJob(id, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[strings.TrimSpace(id)]
	if job == nil {
		return ErrNotFound
	}
	if isTerminalJobStatus(job.Status) {
		return nil
	}
	now := time.Now().UTC()
	job.Status = "FAILED"
	job.StatusReason = strings.TrimSpace(reason)
	job.StoppedAt = &now
	return nil
}

func (s *Service) CreateConsumableResource(name, resourceType string, totalQuantity int64, tags map[string]string) (ConsumableResource, error) {
	name = strings.TrimSpace(name)
	if name == "" || totalQuantity < 0 {
		return ConsumableResource{}, ErrInvalidParameter
	}
	resourceType = normalizeType(resourceType, "REPLENISHABLE")

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.consumables[name]; ok {
		return ConsumableResource{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	cr := &ConsumableResource{
		Name:          name,
		ARN:           consumableResourceARN(name),
		ResourceType:  resourceType,
		TotalQuantity: totalQuantity,
		InUseQuantity: 0,
		Tags:          cloneStringMap(tags),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.consumables[name] = cr
	return cloneConsumableResource(cr), nil
}

func (s *Service) DeleteConsumableResource(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cr := s.resolveConsumableResourceLocked(id)
	if cr == nil {
		return ErrNotFound
	}
	delete(s.consumables, cr.Name)
	return nil
}

func (s *Service) DescribeConsumableResource(id string) (ConsumableResource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cr := s.resolveConsumableResourceLocked(id)
	if cr == nil {
		return ConsumableResource{}, ErrNotFound
	}
	return cloneConsumableResource(cr), nil
}

func (s *Service) ListConsumableResources(maxResults, nextToken int) ([]ConsumableResource, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ConsumableResource, 0, len(s.consumables))
	for _, cr := range s.consumables {
		out = append(out, cloneConsumableResource(cr))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return paginateBatchSlice(out, maxResults, nextToken)
}

func (s *Service) UpdateConsumableResource(id, operation string, quantity int64) (ConsumableResource, error) {
	if quantity < 0 {
		return ConsumableResource{}, ErrInvalidParameter
	}
	operation = normalizeType(operation, "SET")

	s.mu.Lock()
	defer s.mu.Unlock()
	cr := s.resolveConsumableResourceLocked(id)
	if cr == nil {
		return ConsumableResource{}, ErrNotFound
	}
	switch operation {
	case "SET":
		cr.TotalQuantity = quantity
	case "ADD":
		cr.TotalQuantity += quantity
	case "REMOVE", "SUBTRACT":
		if cr.TotalQuantity < quantity {
			return ConsumableResource{}, ErrInvalidParameter
		}
		cr.TotalQuantity -= quantity
	default:
		return ConsumableResource{}, ErrInvalidParameter
	}
	if cr.InUseQuantity > cr.TotalQuantity {
		cr.InUseQuantity = cr.TotalQuantity
	}
	cr.UpdatedAt = time.Now().UTC()
	return cloneConsumableResource(cr), nil
}

func (s *Service) CreateServiceEnvironment(name, envType, state string, capacity []ServiceEnvironmentCapacity, tags map[string]string) (ServiceEnvironment, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ServiceEnvironment{}, ErrInvalidParameter
	}
	envType = normalizeType(envType, "ECS")
	state = normalizeType(state, "ENABLED")

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.serviceEnvs[name]; ok {
		return ServiceEnvironment{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	se := &ServiceEnvironment{
		Name:           name,
		ARN:            serviceEnvironmentARN(name),
		Type:           envType,
		State:          state,
		Status:         "VALID",
		CapacityLimits: cloneServiceEnvironmentCapacities(capacity),
		Tags:           cloneStringMap(tags),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.serviceEnvs[name] = se
	return cloneServiceEnvironment(se), nil
}

func (s *Service) DeleteServiceEnvironment(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	se := s.resolveServiceEnvironmentLocked(id)
	if se == nil {
		return ErrNotFound
	}
	delete(s.serviceEnvs, se.Name)
	return nil
}

func (s *Service) DescribeServiceEnvironments(ids []string, maxResults, nextToken int) ([]ServiceEnvironment, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ServiceEnvironment, 0)
	if len(ids) == 0 {
		out = make([]ServiceEnvironment, 0, len(s.serviceEnvs))
		for _, se := range s.serviceEnvs {
			out = append(out, cloneServiceEnvironment(se))
		}
	} else {
		seen := map[string]struct{}{}
		for _, id := range ids {
			if se := s.resolveServiceEnvironmentLocked(id); se != nil {
				if _, ok := seen[se.Name]; ok {
					continue
				}
				seen[se.Name] = struct{}{}
				out = append(out, cloneServiceEnvironment(se))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return paginateBatchSlice(out, maxResults, nextToken)
}

func (s *Service) UpdateServiceEnvironment(id string, state *string, capacity []ServiceEnvironmentCapacity) (ServiceEnvironment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	se := s.resolveServiceEnvironmentLocked(id)
	if se == nil {
		return ServiceEnvironment{}, ErrNotFound
	}
	if state != nil && strings.TrimSpace(*state) != "" {
		se.State = normalizeType(*state, se.State)
	}
	if capacity != nil {
		se.CapacityLimits = cloneServiceEnvironmentCapacities(capacity)
	}
	se.UpdatedAt = time.Now().UTC()
	return cloneServiceEnvironment(se), nil
}

func (s *Service) SubmitServiceJob(name, queue, serviceJobType, serviceRequestPayload string, schedulingPrio int32, shareID string, retry ServiceJobRetryStrategy, timeout ServiceJobTimeout, tags map[string]string) (ServiceJob, error) {
	name = strings.TrimSpace(name)
	queue = strings.TrimSpace(queue)
	if name == "" || queue == "" || strings.TrimSpace(serviceJobType) == "" {
		return ServiceJob{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	jq := s.resolveJobQueueLocked(queue)
	if jq == nil {
		return ServiceJob{}, ErrNotFound
	}
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	jobID := fmt.Sprintf("svc-job-%06d", seq)
	sj := &ServiceJob{
		ID:                jobID,
		ARN:               serviceJobARN(jobID),
		Name:              name,
		Queue:             jq.ARN,
		Status:            "SUBMITTED",
		StatusReason:      "",
		ServiceJobType:    normalizeType(serviceJobType, ""),
		ServiceReqPayload: strings.TrimSpace(serviceRequestPayload),
		ShareID:           strings.TrimSpace(shareID),
		SchedulingPrio:    schedulingPrio,
		RetryStrategy:     cloneServiceJobRetryStrategy(retry),
		TimeoutConfig:     timeout,
		Tags:              cloneStringMap(tags),
		CreatedAt:         now,
	}
	s.serviceJobs[jobID] = sj
	return cloneServiceJob(sj), nil
}

func (s *Service) DescribeServiceJob(jobID string) (ServiceJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.serviceJobs[strings.TrimSpace(jobID)]
	if job == nil {
		return ServiceJob{}, ErrNotFound
	}
	return cloneServiceJob(job), nil
}

func (s *Service) ListServiceJobs(queue, status string, maxResults, nextToken int) ([]ServiceJob, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	queue = strings.TrimSpace(queue)
	status = normalizeType(status, "")

	queueARN := queue
	if jq := s.resolveJobQueueLocked(queue); jq != nil {
		queueARN = jq.ARN
	}

	out := make([]ServiceJob, 0, len(s.serviceJobs))
	for _, job := range s.serviceJobs {
		if queue != "" && !strings.EqualFold(job.Queue, queue) && !strings.EqualFold(job.Queue, queueARN) {
			continue
		}
		if status != "" && !strings.EqualFold(job.Status, status) {
			continue
		}
		out = append(out, cloneServiceJob(job))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return paginateBatchSlice(out, maxResults, nextToken)
}

func (s *Service) TerminateServiceJob(jobID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.serviceJobs[strings.TrimSpace(jobID)]
	if job == nil {
		return ErrNotFound
	}
	if isTerminalServiceJobStatus(job.Status) {
		return nil
	}
	now := time.Now().UTC()
	job.Status = "TERMINATED"
	job.StatusReason = strings.TrimSpace(reason)
	job.IsTerminated = true
	job.StoppedAt = &now
	return nil
}

type JobQueueSnapshot struct {
	Jobs          []JobQueueSnapshotJob
	LastUpdatedAt time.Time
}

type JobQueueSnapshotJob struct {
	JobARN                 string
	EarliestTimeAtPosition time.Time
}

func (s *Service) GetJobQueueSnapshot(queue string) (JobQueueSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	jq := s.resolveJobQueueLocked(queue)
	if jq == nil {
		return JobQueueSnapshot{}, ErrNotFound
	}
	queueARN := jq.ARN
	summaries := make([]JobQueueSnapshotJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		if !strings.EqualFold(job.Queue, queueARN) {
			continue
		}
		if !strings.EqualFold(job.Status, "SUBMITTED") && !strings.EqualFold(job.Status, "RUNNABLE") {
			continue
		}
		summaries = append(summaries, JobQueueSnapshotJob{
			JobARN:                 job.ARN,
			EarliestTimeAtPosition: job.CreatedAt,
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].EarliestTimeAtPosition.Before(summaries[j].EarliestTimeAtPosition)
	})
	if len(summaries) > 100 {
		summaries = summaries[:100]
	}
	return JobQueueSnapshot{
		Jobs:          summaries,
		LastUpdatedAt: time.Now().UTC(),
	}, nil
}

type ListJobsByConsumableResourceSummary struct {
	Consumables  []ConsumableResourceRequirement
	CreatedAt    time.Time
	JobARN       string
	JobName      string
	JobQueueARN  string
	JobDefARN    string
	JobStatus    string
	Quantity     int64
	ShareID      string
	StartedAt    *time.Time
	StatusReason string
}

func (s *Service) ListJobsByConsumableResource(consumable string, maxResults, nextToken int) ([]ListJobsByConsumableResourceSummary, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cr := s.resolveConsumableResourceLocked(consumable)
	if cr == nil {
		return nil, "", ErrNotFound
	}
	targets := map[string]struct{}{cr.Name: {}, cr.ARN: {}}
	out := make([]ListJobsByConsumableResourceSummary, 0)
	for _, job := range s.jobs {
		matched := false
		totalQty := int64(0)
		for _, req := range job.Consumables {
			if _, ok := targets[req.ConsumableResource]; !ok {
				continue
			}
			matched = true
			totalQty += req.Quantity
		}
		if !matched {
			continue
		}
		out = append(out, ListJobsByConsumableResourceSummary{
			Consumables:  cloneConsumableRequirements(job.Consumables),
			CreatedAt:    job.CreatedAt,
			JobARN:       job.ARN,
			JobName:      job.Name,
			JobQueueARN:  job.Queue,
			JobDefARN:    job.Definition,
			JobStatus:    job.Status,
			Quantity:     totalQty,
			ShareID:      job.ShareID,
			StartedAt:    cloneTimePtr(job.StartedAt),
			StatusReason: job.StatusReason,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	page, token := paginateBatchSlice(out, maxResults, nextToken)
	return page, token, nil
}

func (s *Service) TagResource(resourceARN string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	resourceTags := s.resolveResourceTagsLocked(resourceARN)
	if resourceTags == nil {
		return ErrNotFound
	}
	for k, v := range tags {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		resourceTags[k] = strings.TrimSpace(v)
	}
	return nil
}

func (s *Service) UntagResource(resourceARN string, keys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	resourceTags := s.resolveResourceTagsLocked(resourceARN)
	if resourceTags == nil {
		return ErrNotFound
	}
	for _, key := range keys {
		delete(resourceTags, strings.TrimSpace(key))
	}
	return nil
}

func (s *Service) ListTagsForResource(resourceARN string) (map[string]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resourceTags := s.resolveResourceTagsLocked(resourceARN)
	if resourceTags == nil {
		return nil, false
	}
	return cloneStringMap(resourceTags), true
}

func (s *Service) resolveComputeEnvironmentLocked(id string) *ComputeEnvironment {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if ce, ok := s.computeEnvs[id]; ok {
		return ce
	}
	if strings.HasPrefix(id, "arn:") {
		for _, ce := range s.computeEnvs {
			if ce.ARN == id {
				return ce
			}
		}
	}
	return nil
}

func (s *Service) resolveJobQueueLocked(id string) *JobQueue {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if jq, ok := s.jobQueues[id]; ok {
		return jq
	}
	if strings.HasPrefix(id, "arn:") {
		for _, jq := range s.jobQueues {
			if jq.ARN == id {
				return jq
			}
		}
	}
	return nil
}

func (s *Service) resolveSchedulingPolicyLocked(id string) *SchedulingPolicy {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if sp, ok := s.schedulingPolicies[id]; ok {
		return sp
	}
	if strings.HasPrefix(id, "arn:") {
		for _, sp := range s.schedulingPolicies {
			if sp.ARN == id {
				return sp
			}
		}
	}
	return nil
}

func (s *Service) resolveJobDefinitionLocked(id string) *JobDefinition {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if jd, ok := s.jobDefsByARN[id]; ok {
		return jd
	}
	if strings.HasPrefix(id, "arn:") {
		return nil
	}

	name := id
	revision := int32(-1)
	if idx := strings.LastIndex(id, ":"); idx > 0 {
		if rev, err := strconv.Atoi(id[idx+1:]); err == nil {
			name = id[:idx]
			revision = int32(rev)
		}
	}
	list := s.jobDefsByName[name]
	if len(list) == 0 {
		return nil
	}
	if revision > 0 {
		for _, jd := range list {
			if jd.Revision == revision {
				return jd
			}
		}
		return nil
	}
	for i := len(list) - 1; i >= 0; i-- {
		if strings.EqualFold(list[i].Status, "ACTIVE") {
			return list[i]
		}
	}
	return list[len(list)-1]
}

func (s *Service) resolveConsumableResourceLocked(id string) *ConsumableResource {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if cr, ok := s.consumables[id]; ok {
		return cr
	}
	if strings.HasPrefix(id, "arn:") {
		for _, cr := range s.consumables {
			if cr.ARN == id {
				return cr
			}
		}
	}
	return nil
}

func (s *Service) resolveServiceEnvironmentLocked(id string) *ServiceEnvironment {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if se, ok := s.serviceEnvs[id]; ok {
		return se
	}
	if strings.HasPrefix(id, "arn:") {
		for _, se := range s.serviceEnvs {
			if se.ARN == id {
				return se
			}
		}
	}
	return nil
}

func (s *Service) resolveResourceTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return nil
	}
	if ce := s.resolveComputeEnvironmentLocked(resourceARN); ce != nil {
		return ce.Tags
	}
	if jq := s.resolveJobQueueLocked(resourceARN); jq != nil {
		return jq.Tags
	}
	if sp := s.resolveSchedulingPolicyLocked(resourceARN); sp != nil {
		return sp.Tags
	}
	if jd, ok := s.jobDefsByARN[resourceARN]; ok {
		return jd.Tags
	}
	for _, job := range s.jobs {
		if job.ARN == resourceARN {
			return job.Tags
		}
	}
	if cr := s.resolveConsumableResourceLocked(resourceARN); cr != nil {
		return cr.Tags
	}
	if se := s.resolveServiceEnvironmentLocked(resourceARN); se != nil {
		return se.Tags
	}
	for _, job := range s.serviceJobs {
		if job.ARN == resourceARN {
			return job.Tags
		}
	}
	return nil
}

func isTerminalJobStatus(status string) bool {
	status = normalizeType(status, "")
	switch status {
	case "SUCCEEDED", "FAILED":
		return true
	default:
		return false
	}
}

func isTerminalServiceJobStatus(status string) bool {
	status = normalizeType(status, "")
	switch status {
	case "SUCCEEDED", "FAILED", "TERMINATED":
		return true
	default:
		return false
	}
}

func normalizeType(value, def string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return strings.ToUpper(value)
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(v)
	}
	return out
}

func cloneComputeEnvironment(in *ComputeEnvironment) ComputeEnvironment {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneJobQueue(in *JobQueue) JobQueue {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	out.ComputeEnvironmentOrder = append([]ComputeEnvironmentOrder(nil), in.ComputeEnvironmentOrder...)
	return out
}

func cloneSchedulingPolicy(in *SchedulingPolicy) SchedulingPolicy {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneJobDefinition(in *JobDefinition) JobDefinition {
	out := *in
	out.Parameters = cloneStringMap(in.Parameters)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneJob(in *Job) Job {
	out := *in
	out.Parameters = cloneStringMap(in.Parameters)
	out.Consumables = cloneConsumableRequirements(in.Consumables)
	out.Tags = cloneStringMap(in.Tags)
	if in.StartedAt != nil {
		v := *in.StartedAt
		out.StartedAt = &v
	}
	if in.StoppedAt != nil {
		v := *in.StoppedAt
		out.StoppedAt = &v
	}
	return out
}

func cloneConsumableRequirements(in []ConsumableResourceRequirement) []ConsumableResourceRequirement {
	if len(in) == 0 {
		return nil
	}
	out := make([]ConsumableResourceRequirement, 0, len(in))
	for _, req := range in {
		name := strings.TrimSpace(req.ConsumableResource)
		if name == "" {
			continue
		}
		out = append(out, ConsumableResourceRequirement{
			ConsumableResource: name,
			Quantity:           req.Quantity,
		})
	}
	return out
}

func cloneConsumableResource(in *ConsumableResource) ConsumableResource {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneServiceEnvironmentCapacities(in []ServiceEnvironmentCapacity) []ServiceEnvironmentCapacity {
	if len(in) == 0 {
		return nil
	}
	out := make([]ServiceEnvironmentCapacity, 0, len(in))
	for _, item := range in {
		unit := strings.TrimSpace(item.CapacityUnit)
		if unit == "" {
			continue
		}
		out = append(out, ServiceEnvironmentCapacity{
			CapacityUnit: unit,
			MaxCapacity:  item.MaxCapacity,
		})
	}
	return out
}

func cloneServiceEnvironment(in *ServiceEnvironment) ServiceEnvironment {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	out.CapacityLimits = cloneServiceEnvironmentCapacities(in.CapacityLimits)
	return out
}

func cloneServiceJobRetryStrategy(in ServiceJobRetryStrategy) ServiceJobRetryStrategy {
	out := ServiceJobRetryStrategy{Attempts: in.Attempts}
	if len(in.EvaluateOnExit) > 0 {
		out.EvaluateOnExit = make([]ServiceJobEvaluateOnExit, 0, len(in.EvaluateOnExit))
		for _, rule := range in.EvaluateOnExit {
			out.EvaluateOnExit = append(out.EvaluateOnExit, ServiceJobEvaluateOnExit{
				Action:         strings.TrimSpace(rule.Action),
				OnStatusReason: strings.TrimSpace(rule.OnStatusReason),
			})
		}
	}
	return out
}

func cloneServiceJobAttempts(in []ServiceJobAttempt) []ServiceJobAttempt {
	if len(in) == 0 {
		return nil
	}
	out := make([]ServiceJobAttempt, 0, len(in))
	for _, attempt := range in {
		out = append(out, ServiceJobAttempt{
			ServiceResourceName:  strings.TrimSpace(attempt.ServiceResourceName),
			ServiceResourceValue: strings.TrimSpace(attempt.ServiceResourceValue),
			StartedAt:            cloneTimePtr(attempt.StartedAt),
			StoppedAt:            cloneTimePtr(attempt.StoppedAt),
			StatusReason:         strings.TrimSpace(attempt.StatusReason),
		})
	}
	return out
}

func cloneServiceJob(in *ServiceJob) ServiceJob {
	out := *in
	out.RetryStrategy = cloneServiceJobRetryStrategy(in.RetryStrategy)
	out.Attempts = cloneServiceJobAttempts(in.Attempts)
	out.Tags = cloneStringMap(in.Tags)
	out.StartedAt = cloneTimePtr(in.StartedAt)
	out.StoppedAt = cloneTimePtr(in.StoppedAt)
	return out
}

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := *t
	return &v
}

func paginateBatchSlice[T any](in []T, maxResults, nextToken int) ([]T, string) {
	if nextToken < 0 {
		nextToken = 0
	}
	if nextToken > len(in) {
		return []T{}, ""
	}
	end := len(in)
	if maxResults > 0 && nextToken+maxResults < end {
		end = nextToken + maxResults
	}
	out := append([]T(nil), in[nextToken:end]...)
	if end >= len(in) {
		return out, ""
	}
	return out, fmt.Sprintf("token-%d", end)
}

func computeEnvironmentARN(name string) string {
	return fmt.Sprintf("arn:aws:batch:%s:%s:compute-environment/%s", DefaultRegion, DefaultAccountID, name)
}

func jobQueueARN(name string) string {
	return fmt.Sprintf("arn:aws:batch:%s:%s:job-queue/%s", DefaultRegion, DefaultAccountID, name)
}

func schedulingPolicyARN(name string) string {
	return fmt.Sprintf("arn:aws:batch:%s:%s:scheduling-policy/%s", DefaultRegion, DefaultAccountID, name)
}

func jobDefinitionARN(name string, revision int32) string {
	return fmt.Sprintf("arn:aws:batch:%s:%s:job-definition/%s:%d", DefaultRegion, DefaultAccountID, name, revision)
}

func jobARN(jobID string) string {
	return fmt.Sprintf("arn:aws:batch:%s:%s:job/%s", DefaultRegion, DefaultAccountID, jobID)
}

func consumableResourceARN(name string) string {
	return fmt.Sprintf("arn:aws:batch:%s:%s:consumable-resource/%s", DefaultRegion, DefaultAccountID, name)
}

func serviceEnvironmentARN(name string) string {
	return fmt.Sprintf("arn:aws:batch:%s:%s:service-environment/%s", DefaultRegion, DefaultAccountID, name)
}

func serviceJobARN(jobID string) string {
	return fmt.Sprintf("arn:aws:batch:%s:%s:service-job/%s", DefaultRegion, DefaultAccountID, jobID)
}
