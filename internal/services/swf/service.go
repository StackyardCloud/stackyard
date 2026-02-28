package swf

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrDomainAlreadyExists             = errors.New("domain already exists")
	ErrDomainNotFound                  = errors.New("domain not found")
	ErrActivityTypeAlreadyExists       = errors.New("activity type already exists")
	ErrActivityTypeNotFound            = errors.New("activity type not found")
	ErrWorkflowTypeAlreadyExists       = errors.New("workflow type already exists")
	ErrWorkflowTypeNotFound            = errors.New("workflow type not found")
	ErrWorkflowExecutionAlreadyStarted = errors.New("workflow execution already started")
	ErrWorkflowExecutionNotFound       = errors.New("workflow execution not found")
	ErrInvalidParameter                = errors.New("invalid parameter")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"
	StatusRegistered = "REGISTERED"
	StatusDeprecated = "DEPRECATED"
	StatusOpen       = "OPEN"
	StatusClosed     = "CLOSED"
	CloseCanceled    = "CANCELED"
	CloseTerminated  = "TERMINATED"
)

type Domain struct {
	Name        string
	Description string
	Status      string
	Retention   string
	Tags        map[string]string
}

type ActivityType struct {
	Domain      string
	Name        string
	Version     string
	Description string
	Status      string
}

type WorkflowType struct {
	Domain      string
	Name        string
	Version     string
	Description string
	Status      string
}

type WorkflowExecution struct {
	Domain              string
	WorkflowID          string
	RunID               string
	WorkflowTypeName    string
	WorkflowTypeVersion string
	TaskList            string
	Status              string
	CloseStatus         string
	StartTime           time.Time
	CloseTime           time.Time
	Tags                []string
}

type Service struct {
	mu            sync.Mutex
	seq           uint64
	domains       map[string]*Domain
	activityTypes map[string]*ActivityType
	workflowTypes map[string]*WorkflowType
	executions    map[string]*WorkflowExecution
	resourceTags  map[string]map[string]string
}

func NewService() *Service {
	return &Service{
		domains:       make(map[string]*Domain),
		activityTypes: make(map[string]*ActivityType),
		workflowTypes: make(map[string]*WorkflowType),
		executions:    make(map[string]*WorkflowExecution),
		resourceTags:  make(map[string]map[string]string),
	}
}

func (s *Service) RegisterDomain(name, description, retention string) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.TrimSpace(retention) == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.domains[name]; ok {
		return ErrDomainAlreadyExists
	}
	s.domains[name] = &Domain{
		Name:        name,
		Description: description,
		Status:      StatusRegistered,
		Retention:   retention,
		Tags:        map[string]string{},
	}
	return nil
}

func (s *Service) DescribeDomain(name string) (Domain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	domain, ok := s.domains[name]
	if !ok {
		return Domain{}, ErrDomainNotFound
	}
	return cloneDomain(domain), nil
}

func (s *Service) ListDomains(status string) []Domain {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Domain{}
	for _, domain := range s.domains {
		if status != "" && domain.Status != status {
			continue
		}
		out = append(out, cloneDomain(domain))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) DeprecateDomain(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	domain, ok := s.domains[name]
	if !ok {
		return ErrDomainNotFound
	}
	domain.Status = StatusDeprecated
	return nil
}

func (s *Service) UndeprecateDomain(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	domain, ok := s.domains[name]
	if !ok {
		return ErrDomainNotFound
	}
	domain.Status = StatusRegistered
	return nil
}

func (s *Service) RegisterActivityType(domain, name, version, description string) error {
	if strings.TrimSpace(domain) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(version) == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.domains[domain]; !ok {
		return ErrDomainNotFound
	}
	key := activityKey(domain, name, version)
	if _, ok := s.activityTypes[key]; ok {
		return ErrActivityTypeAlreadyExists
	}
	s.activityTypes[key] = &ActivityType{
		Domain:      domain,
		Name:        name,
		Version:     version,
		Description: description,
		Status:      StatusRegistered,
	}
	return nil
}

func (s *Service) DescribeActivityType(domain, name, version string) (ActivityType, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := activityKey(domain, name, version)
	activity, ok := s.activityTypes[key]
	if !ok {
		return ActivityType{}, ErrActivityTypeNotFound
	}
	return *activity, nil
}

func (s *Service) ListActivityTypes(domain, status string) []ActivityType {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ActivityType{}
	for _, activity := range s.activityTypes {
		if activity.Domain != domain {
			continue
		}
		if status != "" && activity.Status != status {
			continue
		}
		out = append(out, *activity)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Version < out[j].Version
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) DeprecateActivityType(domain, name, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := activityKey(domain, name, version)
	activity, ok := s.activityTypes[key]
	if !ok {
		return ErrActivityTypeNotFound
	}
	activity.Status = StatusDeprecated
	return nil
}

func (s *Service) UndeprecateActivityType(domain, name, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := activityKey(domain, name, version)
	activity, ok := s.activityTypes[key]
	if !ok {
		return ErrActivityTypeNotFound
	}
	activity.Status = StatusRegistered
	return nil
}

func (s *Service) DeleteActivityType(domain, name, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := activityKey(domain, name, version)
	if _, ok := s.activityTypes[key]; !ok {
		return ErrActivityTypeNotFound
	}
	delete(s.activityTypes, key)
	return nil
}

func (s *Service) RegisterWorkflowType(domain, name, version, description string) error {
	if strings.TrimSpace(domain) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(version) == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.domains[domain]; !ok {
		return ErrDomainNotFound
	}
	key := workflowKey(domain, name, version)
	if _, ok := s.workflowTypes[key]; ok {
		return ErrWorkflowTypeAlreadyExists
	}
	s.workflowTypes[key] = &WorkflowType{
		Domain:      domain,
		Name:        name,
		Version:     version,
		Description: description,
		Status:      StatusRegistered,
	}
	return nil
}

func (s *Service) DescribeWorkflowType(domain, name, version string) (WorkflowType, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := workflowKey(domain, name, version)
	workflow, ok := s.workflowTypes[key]
	if !ok {
		return WorkflowType{}, ErrWorkflowTypeNotFound
	}
	return *workflow, nil
}

func (s *Service) ListWorkflowTypes(domain, status string) []WorkflowType {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []WorkflowType{}
	for _, workflow := range s.workflowTypes {
		if workflow.Domain != domain {
			continue
		}
		if status != "" && workflow.Status != status {
			continue
		}
		out = append(out, *workflow)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Version < out[j].Version
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) DeprecateWorkflowType(domain, name, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := workflowKey(domain, name, version)
	workflow, ok := s.workflowTypes[key]
	if !ok {
		return ErrWorkflowTypeNotFound
	}
	workflow.Status = StatusDeprecated
	return nil
}

func (s *Service) UndeprecateWorkflowType(domain, name, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := workflowKey(domain, name, version)
	workflow, ok := s.workflowTypes[key]
	if !ok {
		return ErrWorkflowTypeNotFound
	}
	workflow.Status = StatusRegistered
	return nil
}

func (s *Service) DeleteWorkflowType(domain, name, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := workflowKey(domain, name, version)
	if _, ok := s.workflowTypes[key]; !ok {
		return ErrWorkflowTypeNotFound
	}
	delete(s.workflowTypes, key)
	return nil
}

func (s *Service) StartWorkflowExecution(domain, workflowID, typeName, typeVersion, taskList string, tags []string) (WorkflowExecution, error) {
	workflowID = strings.TrimSpace(workflowID)
	typeName = strings.TrimSpace(typeName)
	typeVersion = strings.TrimSpace(typeVersion)
	if strings.TrimSpace(domain) == "" || workflowID == "" || typeName == "" || typeVersion == "" {
		return WorkflowExecution{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.domains[domain]; !ok {
		return WorkflowExecution{}, ErrDomainNotFound
	}
	if _, ok := s.workflowTypes[workflowKey(domain, typeName, typeVersion)]; !ok {
		return WorkflowExecution{}, ErrWorkflowTypeNotFound
	}
	for _, exec := range s.executions {
		if exec.Domain == domain && exec.WorkflowID == workflowID && exec.Status == StatusOpen {
			return WorkflowExecution{}, ErrWorkflowExecutionAlreadyStarted
		}
	}
	runID := fmt.Sprintf("run-%d", atomic.AddUint64(&s.seq, 1))
	exec := &WorkflowExecution{
		Domain:              domain,
		WorkflowID:          workflowID,
		RunID:               runID,
		WorkflowTypeName:    typeName,
		WorkflowTypeVersion: typeVersion,
		TaskList:            taskList,
		Status:              StatusOpen,
		StartTime:           time.Now().UTC(),
		Tags:                append([]string{}, tags...),
	}
	s.executions[runID] = exec
	return *exec, nil
}

func (s *Service) DescribeWorkflowExecution(domain, workflowID, runID string) (WorkflowExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	exec, err := s.findExecution(domain, workflowID, runID)
	if err != nil {
		return WorkflowExecution{}, err
	}
	return *exec, nil
}

func (s *Service) ListWorkflowExecutions(domain, status string) []WorkflowExecution {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []WorkflowExecution{}
	for _, exec := range s.executions {
		if exec.Domain != domain {
			continue
		}
		if status != "" && exec.Status != status {
			continue
		}
		out = append(out, *exec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartTime.Before(out[j].StartTime)
	})
	return out
}

func (s *Service) CountWorkflowExecutions(domain, status string) int {
	return len(s.ListWorkflowExecutions(domain, status))
}

func (s *Service) CloseWorkflowExecution(domain, workflowID, runID, closeStatus string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	exec, err := s.findExecution(domain, workflowID, runID)
	if err != nil {
		return err
	}
	exec.Status = StatusClosed
	exec.CloseStatus = closeStatus
	exec.CloseTime = time.Now().UTC()
	return nil
}

func (s *Service) TagResource(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.resourceTags[arn]; !ok {
		s.resourceTags[arn] = map[string]string{}
	}
	for k, v := range tags {
		s.resourceTags[arn][k] = v
	}
}

func (s *Service) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tagSet, ok := s.resourceTags[arn]
	if !ok {
		return
	}
	for _, key := range keys {
		delete(tagSet, key)
	}
}

func (s *Service) ListTagsForResource(arn string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]string{}
	for k, v := range s.resourceTags[arn] {
		out[k] = v
	}
	return out
}

func (s *Service) findExecution(domain, workflowID, runID string) (*WorkflowExecution, error) {
	if runID != "" {
		exec, ok := s.executions[runID]
		if !ok || exec.Domain != domain || exec.WorkflowID != workflowID {
			return nil, ErrWorkflowExecutionNotFound
		}
		return exec, nil
	}
	for _, exec := range s.executions {
		if exec.Domain == domain && exec.WorkflowID == workflowID {
			return exec, nil
		}
	}
	return nil, ErrWorkflowExecutionNotFound
}

func cloneDomain(domain *Domain) Domain {
	out := *domain
	out.Tags = map[string]string{}
	for k, v := range domain.Tags {
		out.Tags[k] = v
	}
	return out
}

func activityKey(domain, name, version string) string {
	return fmt.Sprintf("%s|%s|%s", domain, name, version)
}

func workflowKey(domain, name, version string) string {
	return fmt.Sprintf("%s|%s|%s", domain, name, version)
}
