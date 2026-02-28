package eventbridge

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
	ErrEventBusNotFound           = errors.New("event bus not found")
	ErrRuleNotFound               = errors.New("rule not found")
	ErrTargetNotFound             = errors.New("target not found")
	ErrArchiveNotFound            = errors.New("archive not found")
	ErrConnectionNotFound         = errors.New("connection not found")
	ErrApiDestinationNotFound     = errors.New("api destination not found")
	ErrEndpointNotFound           = errors.New("endpoint not found")
	ErrEventSourceNotFound        = errors.New("event source not found")
	ErrPartnerEventSourceNotFound = errors.New("partner event source not found")
	ErrReplayNotFound             = errors.New("replay not found")
	ErrPermissionNotFound         = errors.New("permission not found")
	ErrInvalidParameter           = errors.New("invalid parameter")
	ErrRuleHasTargets             = errors.New("rule has targets")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"
)

type EventBus struct {
	Name             string
	ARN              string
	EventSourceName  string
	Description      string
	KmsKeyIdentifier string
	DeadLetterArn    string
	Policy           string
	Tags             map[string]string
	CreatedAt        time.Time
	LastModifiedAt   time.Time
}

type Rule struct {
	Name               string
	ARN                string
	EventBusName       string
	EventPattern       string
	ScheduleExpression string
	State              string
	Description        string
	RoleArn            string
	ManagedBy          string
	CreatedAt          time.Time
}

type Target struct {
	ID      string
	Arn     string
	RoleArn string
	Input   string
}

type EventEntry struct {
	ID         string
	EventBus   string
	Source     string
	DetailType string
	Detail     string
	Resources  []string
	Time       time.Time
}

type Archive struct {
	Name           string
	ARN            string
	EventSourceArn string
	EventPattern   string
	Description    string
	State          string
	RetentionDays  int32
	CreatedAt      time.Time
}

type Connection struct {
	Name              string
	ARN               string
	AuthorizationType string
	Description       string
	State             string
	CreatedAt         time.Time
	LastAuthorizedAt  time.Time
}

type ApiDestination struct {
	Name                         string
	ARN                          string
	ConnectionArn                string
	InvocationEndpoint           string
	HttpMethod                   string
	InvocationRateLimitPerSecond int32
	Description                  string
	CreatedAt                    time.Time
	LastModifiedAt               time.Time
}

type Endpoint struct {
	Name        string
	ARN         string
	Description string
	EventBuses  []string
	State       string
	CreatedAt   time.Time
}

type EventSource struct {
	Name      string
	ARN       string
	State     string
	CreatedAt time.Time
}

type PartnerEventSource struct {
	Name            string
	ARN             string
	Account         string
	EventSourceName string
	State           string
	CreatedAt       time.Time
}

type Replay struct {
	Name           string
	ARN            string
	State          string
	EventSourceArn string
	Description    string
	EventStartTime time.Time
	EventEndTime   time.Time
	CreatedAt      time.Time
	LastModifiedAt time.Time
}

type Permission struct {
	Action      string
	Principal   string
	StatementID string
}

type Service struct {
	mu                  sync.Mutex
	seq                 uint64
	eventBuses          map[string]*EventBus
	rules               map[string]*Rule
	targets             map[string]map[string]map[string]Target
	events              map[string][]EventEntry
	archives            map[string]*Archive
	connections         map[string]*Connection
	apiDests            map[string]*ApiDestination
	endpoints           map[string]*Endpoint
	eventSources        map[string]*EventSource
	partnerEventSources map[string]*PartnerEventSource
	replays             map[string]*Replay
	permissions         map[string]map[string]Permission
	resourceTags        map[string]map[string]string
}

func NewService() *Service {
	s := &Service{
		eventBuses:          make(map[string]*EventBus),
		rules:               make(map[string]*Rule),
		targets:             make(map[string]map[string]map[string]Target),
		events:              make(map[string][]EventEntry),
		archives:            make(map[string]*Archive),
		connections:         make(map[string]*Connection),
		apiDests:            make(map[string]*ApiDestination),
		endpoints:           make(map[string]*Endpoint),
		eventSources:        make(map[string]*EventSource),
		partnerEventSources: make(map[string]*PartnerEventSource),
		replays:             make(map[string]*Replay),
		permissions:         make(map[string]map[string]Permission),
		resourceTags:        make(map[string]map[string]string),
	}
	_, _ = s.CreateEventBus("default", "", "")
	return s
}

func (s *Service) CreateEventBus(name, eventSourceName, description string) (EventBus, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return EventBus{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if bus, ok := s.eventBuses[name]; ok {
		return cloneBus(bus), nil
	}
	bus := &EventBus{
		Name:            name,
		ARN:             eventBusARN(name),
		EventSourceName: strings.TrimSpace(eventSourceName),
		Description:     strings.TrimSpace(description),
		Tags:            map[string]string{},
		CreatedAt:       time.Now().UTC(),
		LastModifiedAt:  time.Now().UTC(),
	}
	s.eventBuses[name] = bus
	return cloneBus(bus), nil
}

func (s *Service) UpdateEventBus(nameOrArn, kmsKeyIdentifier, description, deadLetterArn string) (EventBus, error) {
	name := strings.TrimSpace(nameOrArn)
	if name == "" {
		return EventBus{}, ErrInvalidParameter
	}
	if strings.HasPrefix(name, "arn:") {
		if parsed := parseEventBusName(name); parsed != "" {
			name = parsed
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	bus, ok := s.eventBuses[name]
	if !ok {
		return EventBus{}, ErrEventBusNotFound
	}
	bus.KmsKeyIdentifier = strings.TrimSpace(kmsKeyIdentifier)
	bus.Description = strings.TrimSpace(description)
	bus.DeadLetterArn = strings.TrimSpace(deadLetterArn)
	bus.LastModifiedAt = time.Now().UTC()
	return cloneBus(bus), nil
}

func (s *Service) DeleteEventBus(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.eventBuses[name]; !ok {
		return ErrEventBusNotFound
	}
	delete(s.eventBuses, name)
	delete(s.events, name)
	for key, rule := range s.rules {
		if rule.EventBusName == name {
			delete(s.rules, key)
		}
	}
	delete(s.targets, name)
	return nil
}

func (s *Service) DescribeEventBus(nameOrArn string) (EventBus, error) {
	name := strings.TrimSpace(nameOrArn)
	if name == "" || name == "default" {
		name = "default"
	}
	if strings.HasPrefix(name, "arn:") {
		if parsed := parseEventBusName(name); parsed != "" {
			name = parsed
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	bus, ok := s.eventBuses[name]
	if !ok {
		return EventBus{}, ErrEventBusNotFound
	}
	return cloneBus(bus), nil
}

func (s *Service) ListEventBuses() []EventBus {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]EventBus, 0, len(s.eventBuses))
	for _, bus := range s.eventBuses {
		out = append(out, cloneBus(bus))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) PutRule(rule Rule) (Rule, error) {
	if strings.TrimSpace(rule.Name) == "" {
		return Rule{}, ErrInvalidParameter
	}
	if rule.EventBusName == "" {
		rule.EventBusName = "default"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.eventBuses[rule.EventBusName]; !ok {
		return Rule{}, ErrEventBusNotFound
	}
	key := ruleKey(rule.EventBusName, rule.Name)
	if existing, ok := s.rules[key]; ok {
		existing.EventPattern = rule.EventPattern
		existing.ScheduleExpression = rule.ScheduleExpression
		existing.Description = rule.Description
		existing.RoleArn = rule.RoleArn
		existing.ManagedBy = rule.ManagedBy
		if rule.State != "" {
			existing.State = rule.State
		}
		return cloneRule(existing), nil
	}
	rule.ARN = ruleARN(rule.EventBusName, rule.Name)
	if rule.State == "" {
		rule.State = "ENABLED"
	}
	rule.CreatedAt = time.Now().UTC()
	s.rules[key] = &rule
	return cloneRule(&rule), nil
}

func (s *Service) DescribeRule(eventBusName, ruleName string) (Rule, error) {
	if ruleName == "" {
		return Rule{}, ErrInvalidParameter
	}
	if eventBusName == "" {
		eventBusName = "default"
	}
	key := ruleKey(eventBusName, ruleName)
	s.mu.Lock()
	defer s.mu.Unlock()
	rule, ok := s.rules[key]
	if !ok {
		return Rule{}, ErrRuleNotFound
	}
	return cloneRule(rule), nil
}

func (s *Service) ListRules(eventBusName, namePrefix string) []Rule {
	if eventBusName == "" {
		eventBusName = "default"
	}
	prefix := strings.TrimSpace(namePrefix)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Rule, 0)
	for _, rule := range s.rules {
		if rule.EventBusName != eventBusName {
			continue
		}
		if prefix != "" && !strings.HasPrefix(rule.Name, prefix) {
			continue
		}
		out = append(out, cloneRule(rule))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) SetRuleState(eventBusName, ruleName, state string) error {
	if eventBusName == "" {
		eventBusName = "default"
	}
	key := ruleKey(eventBusName, ruleName)
	s.mu.Lock()
	defer s.mu.Unlock()
	rule, ok := s.rules[key]
	if !ok {
		return ErrRuleNotFound
	}
	rule.State = state
	return nil
}

func (s *Service) DeleteRule(eventBusName, ruleName string, force bool) error {
	if eventBusName == "" {
		eventBusName = "default"
	}
	key := ruleKey(eventBusName, ruleName)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[key]; !ok {
		return ErrRuleNotFound
	}
	if !force {
		if ruleTargets, ok := s.targets[eventBusName]; ok {
			if targets, ok := ruleTargets[ruleName]; ok && len(targets) > 0 {
				return ErrRuleHasTargets
			}
		}
	}
	delete(s.rules, key)
	if ruleTargets, ok := s.targets[eventBusName]; ok {
		delete(ruleTargets, ruleName)
	}
	return nil
}

func (s *Service) PutTargets(eventBusName, ruleName string, targets []Target) ([]Target, error) {
	if eventBusName == "" {
		eventBusName = "default"
	}
	if ruleName == "" {
		return nil, ErrInvalidParameter
	}
	key := ruleKey(eventBusName, ruleName)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[key]; !ok {
		return nil, ErrRuleNotFound
	}
	if _, ok := s.targets[eventBusName]; !ok {
		s.targets[eventBusName] = make(map[string]map[string]Target)
	}
	if _, ok := s.targets[eventBusName][ruleName]; !ok {
		s.targets[eventBusName][ruleName] = make(map[string]Target)
	}
	failed := make([]Target, 0)
	for _, target := range targets {
		if strings.TrimSpace(target.ID) == "" || strings.TrimSpace(target.Arn) == "" {
			failed = append(failed, target)
			continue
		}
		s.targets[eventBusName][ruleName][target.ID] = target
	}
	return failed, nil
}

func (s *Service) RemoveTargets(eventBusName, ruleName string, targetIDs []string, force bool) ([]string, error) {
	if eventBusName == "" {
		eventBusName = "default"
	}
	key := ruleKey(eventBusName, ruleName)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[key]; !ok {
		return nil, ErrRuleNotFound
	}
	if !force && len(targetIDs) == 0 {
		return nil, ErrInvalidParameter
	}
	failed := make([]string, 0)
	targets := s.targets[eventBusName][ruleName]
	for _, id := range targetIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := targets[id]; !ok {
			failed = append(failed, id)
			continue
		}
		delete(targets, id)
	}
	return failed, nil
}

func (s *Service) ListTargetsByRule(eventBusName, ruleName string) ([]Target, error) {
	if eventBusName == "" {
		eventBusName = "default"
	}
	key := ruleKey(eventBusName, ruleName)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[key]; !ok {
		return nil, ErrRuleNotFound
	}
	targets := s.targets[eventBusName][ruleName]
	out := make([]Target, 0, len(targets))
	for _, target := range targets {
		out = append(out, target)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Service) ListRuleNamesByTarget(eventBusName, targetArn string) []string {
	if eventBusName == "" {
		eventBusName = "default"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, 0)
	for _, rule := range s.rules {
		if rule.EventBusName != eventBusName {
			continue
		}
		targets := s.targets[eventBusName][rule.Name]
		for _, target := range targets {
			if target.Arn == targetArn {
				names = append(names, rule.Name)
				break
			}
		}
	}
	sort.Strings(names)
	return names
}

func (s *Service) PutEvents(entries []EventEntry) ([]EventEntry, []EventEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	success := make([]EventEntry, 0, len(entries))
	failed := make([]EventEntry, 0)
	for _, entry := range entries {
		busName := entry.EventBus
		if busName == "" {
			busName = "default"
		}
		if _, ok := s.eventBuses[busName]; !ok {
			entry.Detail = "event bus not found"
			failed = append(failed, entry)
			continue
		}
		if strings.TrimSpace(entry.Source) == "" || strings.TrimSpace(entry.DetailType) == "" || strings.TrimSpace(entry.Detail) == "" {
			entry.Detail = "missing required fields"
			failed = append(failed, entry)
			continue
		}
		entry.ID = s.nextID("evt")
		entry.EventBus = busName
		if entry.Time.IsZero() {
			entry.Time = time.Now().UTC()
		}
		s.events[busName] = append(s.events[busName], entry)
		success = append(success, entry)
	}
	return success, failed
}

func (s *Service) TagResource(arn string, tags map[string]string) {
	if arn == "" || len(tags) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := s.resourceTags[arn]
	if existing == nil {
		existing = map[string]string{}
		s.resourceTags[arn] = existing
	}
	for k, v := range tags {
		existing[k] = v
	}
}

func (s *Service) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if arn == "" {
		return
	}
	existing := s.resourceTags[arn]
	for _, key := range keys {
		delete(existing, key)
	}
}

func (s *Service) ListTags(arn string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := s.resourceTags[arn]
	out := map[string]string{}
	for k, v := range existing {
		out[k] = v
	}
	return out
}

func (s *Service) CreateArchive(archive Archive) (Archive, error) {
	archive.Name = strings.TrimSpace(archive.Name)
	if archive.Name == "" {
		return Archive{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.eventBuses[archive.EventSourceArn]; ok {
		archive.EventSourceArn = eventBusARN(archive.EventSourceArn)
	}
	if archive.ARN == "" {
		archive.ARN = archiveARN(archive.Name)
	}
	if archive.State == "" {
		archive.State = "ENABLED"
	}
	archive.CreatedAt = time.Now().UTC()
	s.archives[archive.Name] = &archive
	return cloneArchive(&archive), nil
}

func (s *Service) DescribeArchive(name string) (Archive, error) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	archive, ok := s.archives[name]
	if !ok {
		return Archive{}, ErrArchiveNotFound
	}
	return cloneArchive(archive), nil
}

func (s *Service) UpdateArchive(name, description, eventPattern string, retentionDays int32) (Archive, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	archive, ok := s.archives[name]
	if !ok {
		return Archive{}, ErrArchiveNotFound
	}
	if description != "" {
		archive.Description = description
	}
	if eventPattern != "" {
		archive.EventPattern = eventPattern
	}
	if retentionDays != 0 {
		archive.RetentionDays = retentionDays
	}
	return cloneArchive(archive), nil
}

func (s *Service) DeleteArchive(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.archives[name]; !ok {
		return ErrArchiveNotFound
	}
	delete(s.archives, name)
	return nil
}

func (s *Service) ListArchives() []Archive {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Archive, 0, len(s.archives))
	for _, archive := range s.archives {
		out = append(out, cloneArchive(archive))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) CreateConnection(conn Connection) (Connection, error) {
	conn.Name = strings.TrimSpace(conn.Name)
	if conn.Name == "" {
		return Connection{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if conn.ARN == "" {
		conn.ARN = connectionARN(conn.Name)
	}
	if conn.State == "" {
		conn.State = "AUTHORIZED"
	}
	conn.CreatedAt = time.Now().UTC()
	conn.LastAuthorizedAt = conn.CreatedAt
	s.connections[conn.Name] = &conn
	return cloneConnection(&conn), nil
}

func (s *Service) DescribeConnection(name string) (Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn, ok := s.connections[name]
	if !ok {
		return Connection{}, ErrConnectionNotFound
	}
	return cloneConnection(conn), nil
}

func (s *Service) UpdateConnection(name, description string) (Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn, ok := s.connections[name]
	if !ok {
		return Connection{}, ErrConnectionNotFound
	}
	if description != "" {
		conn.Description = description
	}
	return cloneConnection(conn), nil
}

func (s *Service) DeleteConnection(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.connections[name]; !ok {
		return ErrConnectionNotFound
	}
	delete(s.connections, name)
	return nil
}

func (s *Service) ListConnections() []Connection {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Connection, 0, len(s.connections))
	for _, conn := range s.connections {
		out = append(out, cloneConnection(conn))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) DeauthorizeConnection(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn, ok := s.connections[name]
	if !ok {
		return ErrConnectionNotFound
	}
	conn.State = "DEAUTHORIZED"
	return nil
}

func (s *Service) CreateApiDestination(dest ApiDestination) (ApiDestination, error) {
	dest.Name = strings.TrimSpace(dest.Name)
	if dest.Name == "" {
		return ApiDestination{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if dest.ARN == "" {
		dest.ARN = apiDestinationARN(dest.Name)
	}
	dest.CreatedAt = time.Now().UTC()
	dest.LastModifiedAt = dest.CreatedAt
	s.apiDests[dest.Name] = &dest
	return cloneApiDestination(&dest), nil
}

func (s *Service) DescribeApiDestination(name string) (ApiDestination, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dest, ok := s.apiDests[name]
	if !ok {
		return ApiDestination{}, ErrApiDestinationNotFound
	}
	return cloneApiDestination(dest), nil
}

func (s *Service) UpdateApiDestination(name, endpoint string, rateLimit int32) (ApiDestination, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dest, ok := s.apiDests[name]
	if !ok {
		return ApiDestination{}, ErrApiDestinationNotFound
	}
	if endpoint != "" {
		dest.InvocationEndpoint = endpoint
	}
	if rateLimit != 0 {
		dest.InvocationRateLimitPerSecond = rateLimit
	}
	dest.LastModifiedAt = time.Now().UTC()
	return cloneApiDestination(dest), nil
}

func (s *Service) DeleteApiDestination(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apiDests[name]; !ok {
		return ErrApiDestinationNotFound
	}
	delete(s.apiDests, name)
	return nil
}

func (s *Service) ListApiDestinations() []ApiDestination {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ApiDestination, 0, len(s.apiDests))
	for _, dest := range s.apiDests {
		out = append(out, cloneApiDestination(dest))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) CreateEndpoint(ep Endpoint) (Endpoint, error) {
	ep.Name = strings.TrimSpace(ep.Name)
	if ep.Name == "" {
		return Endpoint{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if ep.ARN == "" {
		ep.ARN = endpointARN(ep.Name)
	}
	if ep.State == "" {
		ep.State = "ACTIVE"
	}
	ep.CreatedAt = time.Now().UTC()
	s.endpoints[ep.Name] = &ep
	return cloneEndpoint(&ep), nil
}

func (s *Service) DescribeEndpoint(name string) (Endpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ep, ok := s.endpoints[name]
	if !ok {
		return Endpoint{}, ErrEndpointNotFound
	}
	return cloneEndpoint(ep), nil
}

func (s *Service) UpdateEndpoint(name, description string) (Endpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ep, ok := s.endpoints[name]
	if !ok {
		return Endpoint{}, ErrEndpointNotFound
	}
	if description != "" {
		ep.Description = description
	}
	return cloneEndpoint(ep), nil
}

func (s *Service) DeleteEndpoint(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.endpoints[name]; !ok {
		return ErrEndpointNotFound
	}
	delete(s.endpoints, name)
	return nil
}

func (s *Service) ListEndpoints() []Endpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Endpoint, 0, len(s.endpoints))
	for _, ep := range s.endpoints {
		out = append(out, cloneEndpoint(ep))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) ActivateEventSource(name string) (EventSource, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return EventSource{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.eventSources[name]
	if !ok {
		return EventSource{}, ErrEventSourceNotFound
	}
	src.State = "ACTIVE"
	return cloneEventSource(src), nil
}

func (s *Service) DeactivateEventSource(name string) (EventSource, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return EventSource{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.eventSources[name]
	if !ok {
		return EventSource{}, ErrEventSourceNotFound
	}
	src.State = "INACTIVE"
	return cloneEventSource(src), nil
}

func (s *Service) DescribeEventSource(name string) (EventSource, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return EventSource{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.eventSources[name]
	if !ok {
		return EventSource{}, ErrEventSourceNotFound
	}
	return cloneEventSource(src), nil
}

func (s *Service) ListEventSources(prefix string) []EventSource {
	prefix = strings.TrimSpace(prefix)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]EventSource, 0, len(s.eventSources))
	for _, src := range s.eventSources {
		if prefix != "" && !strings.HasPrefix(src.Name, prefix) {
			continue
		}
		out = append(out, cloneEventSource(src))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) CreatePartnerEventSource(name, account, eventSourceName string) (PartnerEventSource, error) {
	name = strings.TrimSpace(name)
	account = strings.TrimSpace(account)
	eventSourceName = strings.TrimSpace(eventSourceName)
	if eventSourceName == "" {
		eventSourceName = name
	}
	if name == "" || account == "" || eventSourceName == "" {
		return PartnerEventSource{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if src, ok := s.partnerEventSources[name]; ok {
		return clonePartnerEventSource(src), nil
	}
	src := &PartnerEventSource{
		Name:            name,
		Account:         account,
		EventSourceName: eventSourceName,
		ARN:             partnerEventSourceARN(name),
		State:           "ACTIVE",
		CreatedAt:       time.Now().UTC(),
	}
	s.partnerEventSources[name] = src
	if _, ok := s.eventSources[eventSourceName]; !ok {
		s.eventSources[eventSourceName] = &EventSource{
			Name:      eventSourceName,
			ARN:       eventSourceARN(eventSourceName),
			State:     "ACTIVE",
			CreatedAt: time.Now().UTC(),
		}
	}
	return clonePartnerEventSource(src), nil
}

func (s *Service) DeletePartnerEventSource(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.partnerEventSources[name]; !ok {
		return ErrPartnerEventSourceNotFound
	}
	delete(s.partnerEventSources, name)
	return nil
}

func (s *Service) DescribePartnerEventSource(name string) (PartnerEventSource, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return PartnerEventSource{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.partnerEventSources[name]
	if !ok {
		return PartnerEventSource{}, ErrPartnerEventSourceNotFound
	}
	return clonePartnerEventSource(src), nil
}

func (s *Service) ListPartnerEventSources(prefix string) []PartnerEventSource {
	prefix = strings.TrimSpace(prefix)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PartnerEventSource, 0, len(s.partnerEventSources))
	for _, src := range s.partnerEventSources {
		if prefix != "" && !strings.HasPrefix(src.Name, prefix) {
			continue
		}
		out = append(out, clonePartnerEventSource(src))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) ListPartnerEventSourceAccounts(eventSourceName string) []string {
	eventSourceName = strings.TrimSpace(eventSourceName)
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts := map[string]struct{}{}
	for _, src := range s.partnerEventSources {
		if eventSourceName != "" && src.EventSourceName != eventSourceName {
			continue
		}
		accounts[src.Account] = struct{}{}
	}
	out := make([]string, 0, len(accounts))
	for account := range accounts {
		out = append(out, account)
	}
	sort.Strings(out)
	return out
}

func (s *Service) StartReplay(name, eventSourceArn, description string, startTime, endTime time.Time) (Replay, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.TrimSpace(eventSourceArn) == "" {
		return Replay{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if replay, ok := s.replays[name]; ok {
		return cloneReplay(replay), nil
	}
	replay := &Replay{
		Name:           name,
		ARN:            replayARN(name),
		State:          "RUNNING",
		EventSourceArn: eventSourceArn,
		Description:    strings.TrimSpace(description),
		EventStartTime: startTime,
		EventEndTime:   endTime,
		CreatedAt:      time.Now().UTC(),
		LastModifiedAt: time.Now().UTC(),
	}
	s.replays[name] = replay
	return cloneReplay(replay), nil
}

func (s *Service) CancelReplay(name string) (Replay, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Replay{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	replay, ok := s.replays[name]
	if !ok {
		return Replay{}, ErrReplayNotFound
	}
	replay.State = "CANCELLED"
	replay.LastModifiedAt = time.Now().UTC()
	return cloneReplay(replay), nil
}

func (s *Service) DescribeReplay(name string) (Replay, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Replay{}, ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	replay, ok := s.replays[name]
	if !ok {
		return Replay{}, ErrReplayNotFound
	}
	return cloneReplay(replay), nil
}

func (s *Service) ListReplays(prefix string) []Replay {
	prefix = strings.TrimSpace(prefix)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Replay, 0, len(s.replays))
	for _, replay := range s.replays {
		if prefix != "" && !strings.HasPrefix(replay.Name, prefix) {
			continue
		}
		out = append(out, cloneReplay(replay))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) PutPermission(eventBusName, statementID, action, principal string) error {
	eventBusName = strings.TrimSpace(eventBusName)
	if eventBusName == "" {
		eventBusName = "default"
	}
	statementID = strings.TrimSpace(statementID)
	if statementID == "" {
		return ErrInvalidParameter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.eventBuses[eventBusName]; !ok {
		return ErrEventBusNotFound
	}
	if _, ok := s.permissions[eventBusName]; !ok {
		s.permissions[eventBusName] = map[string]Permission{}
	}
	s.permissions[eventBusName][statementID] = Permission{
		Action:      strings.TrimSpace(action),
		Principal:   strings.TrimSpace(principal),
		StatementID: statementID,
	}
	return nil
}

func (s *Service) RemovePermission(eventBusName, statementID string, removeAll bool) error {
	eventBusName = strings.TrimSpace(eventBusName)
	if eventBusName == "" {
		eventBusName = "default"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.eventBuses[eventBusName]; !ok {
		return ErrEventBusNotFound
	}
	if removeAll {
		delete(s.permissions, eventBusName)
		return nil
	}
	if _, ok := s.permissions[eventBusName][statementID]; !ok {
		return ErrPermissionNotFound
	}
	delete(s.permissions[eventBusName], statementID)
	return nil
}

func (s *Service) nextID(prefix string) string {
	val := atomic.AddUint64(&s.seq, 1)
	return fmt.Sprintf("%s-%d", prefix, val)
}

func eventBusARN(name string) string {
	return fmt.Sprintf("arn:aws:events:%s:%s:event-bus/%s", DefaultRegion, DefaultAccountID, name)
}

func ruleARN(eventBusName, ruleName string) string {
	return fmt.Sprintf("arn:aws:events:%s:%s:rule/%s/%s", DefaultRegion, DefaultAccountID, eventBusName, ruleName)
}

func archiveARN(name string) string {
	return fmt.Sprintf("arn:aws:events:%s:%s:archive/%s", DefaultRegion, DefaultAccountID, name)
}

func connectionARN(name string) string {
	return fmt.Sprintf("arn:aws:events:%s:%s:connection/%s", DefaultRegion, DefaultAccountID, name)
}

func apiDestinationARN(name string) string {
	return fmt.Sprintf("arn:aws:events:%s:%s:api-destination/%s", DefaultRegion, DefaultAccountID, name)
}

func endpointARN(name string) string {
	return fmt.Sprintf("arn:aws:events:%s:%s:endpoint/%s", DefaultRegion, DefaultAccountID, name)
}

func eventSourceARN(name string) string {
	return fmt.Sprintf("arn:aws:events:%s:%s:event-source/%s", DefaultRegion, DefaultAccountID, name)
}

func partnerEventSourceARN(name string) string {
	return fmt.Sprintf("arn:aws:events:%s:%s:partner-event-source/%s", DefaultRegion, DefaultAccountID, name)
}

func replayARN(name string) string {
	return fmt.Sprintf("arn:aws:events:%s:%s:replay/%s", DefaultRegion, DefaultAccountID, name)
}

func ruleKey(busName, ruleName string) string {
	return busName + "/" + ruleName
}

func parseEventBusName(arn string) string {
	parts := strings.Split(arn, ":event-bus/")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func cloneBus(bus *EventBus) EventBus {
	clone := *bus
	clone.Tags = cloneStringMap(bus.Tags)
	return clone
}

func cloneRule(rule *Rule) Rule {
	clone := *rule
	return clone
}

func cloneArchive(archive *Archive) Archive {
	clone := *archive
	return clone
}

func cloneConnection(conn *Connection) Connection {
	clone := *conn
	return clone
}

func cloneApiDestination(dest *ApiDestination) ApiDestination {
	clone := *dest
	return clone
}

func cloneEndpoint(ep *Endpoint) Endpoint {
	clone := *ep
	clone.EventBuses = append([]string{}, ep.EventBuses...)
	return clone
}

func cloneEventSource(src *EventSource) EventSource {
	clone := *src
	return clone
}

func clonePartnerEventSource(src *PartnerEventSource) PartnerEventSource {
	clone := *src
	return clone
}

func cloneReplay(replay *Replay) Replay {
	clone := *replay
	return clone
}

func cloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
