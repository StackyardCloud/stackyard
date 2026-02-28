package ecs

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidParameter          = errors.New("invalid parameter")
	ErrAlreadyExists             = errors.New("resource already exists")
	ErrClusterNotFound           = errors.New("cluster not found")
	ErrCapacityProviderNotFound  = errors.New("capacity provider not found")
	ErrContainerInstanceNotFound = errors.New("container instance not found")
	ErrTaskDefinitionNotFound    = errors.New("task definition not found")
	ErrServiceNotFound           = errors.New("service not found")
	ErrResourceNotFound          = errors.New("resource not found")
	ErrServiceDeploymentNotFound = errors.New("service deployment not found")
	ErrServiceRevisionNotFound   = errors.New("service revision not found")
	ErrTaskSetNotFound           = errors.New("task set not found")
	ErrTaskNotFound              = errors.New("task not found")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"
)

type AccountSetting struct {
	Name         string
	Value        string
	PrincipalArn string
	Type         string
}

type ClusterSetting struct {
	Name  string
	Value string
}

type CapacityProviderStrategyItem struct {
	CapacityProvider string
	Base             int32
	Weight           int32
}

type Cluster struct {
	ClusterArn                        string
	ClusterName                       string
	Status                            string
	Settings                          []ClusterSetting
	Statistics                        map[string]string
	Configuration                     map[string]any
	ServiceConnectDefaultsNamespace   string
	Tags                              map[string]string
	RegisteredContainerInstancesCount int32
	RunningTasksCount                 int32
	PendingTasksCount                 int32
	ActiveServicesCount               int32
	CapacityProviders                 []string
	DefaultCapacityProviderStrategy   []CapacityProviderStrategyItem
	CreatedAt                         time.Time
	UpdatedAt                         time.Time
}

type Failure struct {
	Arn    string
	Reason string
	Detail string
}

// Placeholder resources for upcoming stages.
type CapacityProvider struct {
	Name                         string
	Arn                          string
	Status                       string
	UpdateStatus                 string
	AutoScalingGroupArn          string
	ManagedScalingStatus         string
	ManagedTerminationProtection string
	Tags                         map[string]string
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
}

type ContainerDefinition struct {
	Name  string
	Image string
}

type TaskDefinitionInput struct {
	Family                  string
	NetworkMode             string
	Cpu                     string
	Memory                  string
	ExecutionRoleArn        string
	TaskRoleArn             string
	RequiresCompatibilities []string
	ContainerDefinitions    []ContainerDefinition
	Tags                    map[string]string
}

type TaskDefinition struct {
	Family                  string
	Revision                int64
	Arn                     string
	Status                  string
	NetworkMode             string
	Cpu                     string
	Memory                  string
	ExecutionRoleArn        string
	TaskRoleArn             string
	RequiresCompatibilities []string
	ContainerDefinitions    []ContainerDefinition
	Tags                    map[string]string
	RegisteredAt            time.Time
	DeregisteredAt          *time.Time
}

type ServiceDefinition struct {
	Name              string
	Arn               string
	ClusterArn        string
	TaskDefinitionArn string
	PrimaryTaskSetArn string
	DesiredCount      int32
	LaunchType        string
	Status            string
	Tags              map[string]string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Task struct {
	Arn                  string
	ClusterArn           string
	TaskDefinitionArn    string
	ServiceArn           string
	ContainerInstanceArn string
	Group                string
	StartedBy            string
	LaunchType           string
	LastStatus           string
	DesiredStatus        string
	CreatedAt            time.Time
	StartedAt            time.Time
	StoppedAt            *time.Time
	StoppedReason        string
}

type TaskSet struct {
	Arn               string
	ID                string
	ClusterArn        string
	ServiceArn        string
	TaskDefinitionArn string
	ComputedDesired   int32
	PendingCount      int32
	RunningCount      int32
	Status            string
	LaunchType        string
	ScaleValue        float64
	ScaleUnit         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ContainerInstance struct {
	Arn               string
	ClusterArn        string
	Ec2InstanceID     string
	Status            string
	AgentConnected    bool
	AgentUpdateStatus string
	Version           int64
	VersionInfo       VersionInfo
	RegisteredAt      time.Time
	UpdatedAt         time.Time
	Attributes        []Attribute
}

type VersionInfo struct {
	AgentHash     string
	AgentVersion  string
	DockerVersion string
}

type ExpressGatewayServiceConfiguration struct {
	ServiceRevisionArn   string
	Cpu                  string
	ExecutionRoleArn     string
	HealthCheckPath      string
	Memory               string
	TaskRoleArn          string
	PrimaryContainer     map[string]any
	NetworkConfiguration map[string]any
	ScalingTarget        map[string]any
	CreatedAt            time.Time
}

type ExpressGatewayService struct {
	ServiceArn            string
	ServiceName           string
	Cluster               string
	InfrastructureRoleArn string
	StatusCode            string
	StatusReason          string
	CurrentDeployment     string
	ActiveConfigurations  []ExpressGatewayServiceConfiguration
	Tags                  map[string]string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type UpdatedExpressGatewayService struct {
	ServiceArn          string
	ServiceName         string
	Cluster             string
	StatusCode          string
	StatusReason        string
	TargetConfiguration ExpressGatewayServiceConfiguration
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type TaskProtection struct {
	TaskArn           string
	ProtectionEnabled bool
	ExpirationDate    *time.Time
}

type ExecuteCommandResult struct {
	ClusterArn    string
	TaskArn       string
	ContainerName string
	Interactive   bool
	SessionID     string
	StreamURL     string
	TokenValue    string
}

type DiscoverPollEndpointResult struct {
	Endpoint          string
	TelemetryEndpoint string
}

type Attribute struct {
	Name       string
	Value      string
	TargetType string
	TargetID   string
}

type AttachmentStateChange struct {
	AttachmentArn string
	Status        string
}

type ServiceDeploymentSnapshot struct {
	Arn                string
	ServiceArn         string
	ServiceRevisionArn string
	Status             string
	StatusReason       string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	StoppedAt          *time.Time
}

type ServiceRevisionSnapshot struct {
	Arn               string
	ServiceArn        string
	TaskDefinitionArn string
	DesiredCount      int32
	CreatedAt         time.Time
}

type Service struct {
	mu sync.Mutex

	accountSettings map[string]string
	accountDefaults map[string]string

	clustersByName map[string]*Cluster
	clustersByARN  map[string]*Cluster

	capacityProviders    map[string]*CapacityProvider
	taskDefinitions      map[string][]*TaskDefinition
	taskDefinitionsByARN map[string]*TaskDefinition
	services             map[string]*ServiceDefinition
	expressServices      map[string]*ExpressGatewayService
	tasks                map[string]*Task
	taskSets             map[string]*TaskSet
	containerInstances   map[string]*ContainerInstance
	tagsByARN            map[string]map[string]string
	taskProtectionByTask map[string]TaskProtection
	attributesByKey      map[string]Attribute

	serviceDeployments map[string][]ServiceDeploymentSnapshot
	serviceRevisions   map[string][]ServiceRevisionSnapshot
	taskSetSeqBySvcARN map[string]int
	taskSeq            int64
	containerInstSeq   int64
	execSeq            int64
	expressServiceSeq  int64
}

func NewService() *Service {
	return &Service{
		accountSettings:      map[string]string{},
		accountDefaults:      map[string]string{},
		clustersByName:       map[string]*Cluster{},
		clustersByARN:        map[string]*Cluster{},
		capacityProviders:    map[string]*CapacityProvider{},
		taskDefinitions:      map[string][]*TaskDefinition{},
		taskDefinitionsByARN: map[string]*TaskDefinition{},
		services:             map[string]*ServiceDefinition{},
		expressServices:      map[string]*ExpressGatewayService{},
		tasks:                map[string]*Task{},
		taskSets:             map[string]*TaskSet{},
		containerInstances:   map[string]*ContainerInstance{},
		tagsByARN:            map[string]map[string]string{},
		taskProtectionByTask: map[string]TaskProtection{},
		attributesByKey:      map[string]Attribute{},
		serviceDeployments:   map[string][]ServiceDeploymentSnapshot{},
		serviceRevisions:     map[string][]ServiceRevisionSnapshot{},
		taskSetSeqBySvcARN:   map[string]int{},
	}
}

func (s *Service) PutAccountSetting(name, value, principalArn string) (AccountSetting, error) {
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)
	principalArn = strings.TrimSpace(principalArn)
	if name == "" || value == "" {
		return AccountSetting{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountSettings[accountSettingKey(principalArn, name)] = value
	return AccountSetting{
		Name:         name,
		Value:        value,
		PrincipalArn: principalArn,
		Type:         "user",
	}, nil
}

func (s *Service) PutAccountSettingDefault(name, value string) (AccountSetting, error) {
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)
	if name == "" || value == "" {
		return AccountSetting{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountDefaults[name] = value
	return AccountSetting{
		Name:         name,
		Value:        value,
		PrincipalArn: fmt.Sprintf("arn:aws:iam::%s:root", DefaultAccountID),
		Type:         "aws_managed",
	}, nil
}

func (s *Service) ListAccountSettings(names []string, effectiveSettings bool, principalArn string) ([]AccountSetting, error) {
	principalArn = strings.TrimSpace(principalArn)
	filter := map[string]struct{}{}
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		filter[trimmed] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	values := map[string]AccountSetting{}
	for key, value := range s.accountSettings {
		p, name := splitAccountSettingKey(key)
		if p != principalArn {
			continue
		}
		if len(filter) > 0 {
			if _, ok := filter[name]; !ok {
				continue
			}
		}
		values[name] = AccountSetting{
			Name:         name,
			Value:        value,
			PrincipalArn: principalArn,
			Type:         "user",
		}
	}

	if effectiveSettings {
		for name, value := range s.accountDefaults {
			if len(filter) > 0 {
				if _, ok := filter[name]; !ok {
					continue
				}
			}
			if _, exists := values[name]; exists {
				continue
			}
			values[name] = AccountSetting{
				Name:         name,
				Value:        value,
				PrincipalArn: fmt.Sprintf("arn:aws:iam::%s:root", DefaultAccountID),
				Type:         "aws_managed",
			}
		}
	}

	out := make([]AccountSetting, 0, len(values))
	for _, setting := range values {
		out = append(out, setting)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (s *Service) DeleteAccountSetting(name, principalArn string) error {
	name = strings.TrimSpace(name)
	principalArn = strings.TrimSpace(principalArn)
	if name == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.accountSettings, accountSettingKey(principalArn, name))
	return nil
}

func (s *Service) CreateCluster(clusterName string, settings []ClusterSetting, tags map[string]string) (Cluster, error) {
	clusterName = strings.TrimSpace(clusterName)
	if clusterName == "" {
		return Cluster{}, ErrInvalidParameter
	}

	normalizedSettings, err := normalizeClusterSettings(settings)
	if err != nil {
		return Cluster{}, err
	}
	normalizedTags, err := normalizeTags(tags)
	if err != nil {
		return Cluster{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.clustersByName[clusterName]; exists {
		return Cluster{}, ErrAlreadyExists
	}

	now := time.Now().UTC()
	cluster := &Cluster{
		ClusterArn:                        clusterARN(clusterName),
		ClusterName:                       clusterName,
		Status:                            "ACTIVE",
		Settings:                          normalizedSettings,
		Statistics:                        map[string]string{},
		Configuration:                     map[string]any{},
		ServiceConnectDefaultsNamespace:   "",
		Tags:                              normalizedTags,
		RegisteredContainerInstancesCount: 0,
		RunningTasksCount:                 0,
		PendingTasksCount:                 0,
		ActiveServicesCount:               0,
		CapacityProviders:                 []string{},
		DefaultCapacityProviderStrategy:   []CapacityProviderStrategyItem{},
		CreatedAt:                         now,
		UpdatedAt:                         now,
	}
	s.clustersByName[cluster.ClusterName] = cluster
	s.clustersByARN[cluster.ClusterArn] = cluster
	if len(cluster.Tags) > 0 {
		s.tagsByARN[cluster.ClusterArn] = cloneStringMap(cluster.Tags)
	}
	return cloneCluster(cluster), nil
}

func (s *Service) DescribeClusters(clusters []string) ([]Cluster, []Failure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(clusters) == 0 {
		out := make([]Cluster, 0, len(s.clustersByName))
		for _, cluster := range s.clustersByName {
			out = append(out, cloneCluster(cluster))
		}
		sort.Slice(out, func(i, j int) bool {
			return out[i].ClusterName < out[j].ClusterName
		})
		return out, nil, nil
	}

	seen := map[string]struct{}{}
	out := make([]Cluster, 0, len(clusters))
	failures := make([]Failure, 0)
	for _, ref := range clusters {
		cluster := s.resolveClusterLocked(ref)
		if cluster == nil {
			failures = append(failures, Failure{
				Arn:    strings.TrimSpace(ref),
				Reason: "MISSING",
				Detail: "cluster not found",
			})
			continue
		}
		if _, exists := seen[cluster.ClusterArn]; exists {
			continue
		}
		seen[cluster.ClusterArn] = struct{}{}
		out = append(out, cloneCluster(cluster))
	}
	return out, failures, nil
}

func (s *Service) ListClusters(nextToken string, maxResults int32) ([]string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	all := make([]string, 0, len(s.clustersByName))
	for _, cluster := range s.clustersByName {
		all = append(all, cluster.ClusterArn)
	}
	sort.Strings(all)

	offset := 0
	nextToken = strings.TrimSpace(nextToken)
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 0 || parsed > len(all) {
			return nil, "", ErrInvalidParameter
		}
		offset = parsed
	}
	limit := int(maxResults)
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}

	out := append([]string(nil), all[offset:end]...)
	newNextToken := ""
	if end < len(all) {
		newNextToken = strconv.Itoa(end)
	}
	return out, newNextToken, nil
}

func (s *Service) UpdateClusterSettings(clusterRef string, settings []ClusterSetting) (Cluster, error) {
	normalizedSettings, err := normalizeClusterSettings(settings)
	if err != nil {
		return Cluster{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	cluster := s.resolveClusterLocked(clusterRef)
	if cluster == nil {
		return Cluster{}, ErrClusterNotFound
	}

	settingMap := map[string]string{}
	for _, setting := range cluster.Settings {
		settingMap[setting.Name] = setting.Value
	}
	for _, setting := range normalizedSettings {
		settingMap[setting.Name] = setting.Value
	}
	merged := make([]ClusterSetting, 0, len(settingMap))
	for name, value := range settingMap {
		merged = append(merged, ClusterSetting{Name: name, Value: value})
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Name < merged[j].Name
	})

	cluster.Settings = merged
	cluster.UpdatedAt = time.Now().UTC()
	return cloneCluster(cluster), nil
}

func (s *Service) UpdateCluster(clusterRef string, settings []ClusterSetting, configuration map[string]any, serviceConnectNamespace string, hasServiceConnectDefaults bool) (Cluster, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	if clusterRef == "" {
		return Cluster{}, ErrInvalidParameter
	}

	var normalizedSettings []ClusterSetting
	var err error
	if settings != nil {
		normalizedSettings, err = normalizeClusterSettings(settings)
		if err != nil {
			return Cluster{}, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster := s.resolveClusterLocked(clusterRef)
	if cluster == nil {
		return Cluster{}, ErrClusterNotFound
	}

	if settings != nil {
		settingMap := map[string]string{}
		for _, setting := range cluster.Settings {
			settingMap[setting.Name] = setting.Value
		}
		for _, setting := range normalizedSettings {
			settingMap[setting.Name] = setting.Value
		}
		merged := make([]ClusterSetting, 0, len(settingMap))
		for name, value := range settingMap {
			merged = append(merged, ClusterSetting{Name: name, Value: value})
		}
		sort.Slice(merged, func(i, j int) bool { return merged[i].Name < merged[j].Name })
		cluster.Settings = merged
	}

	if configuration != nil {
		cluster.Configuration = cloneAnyMap(configuration)
	}

	if hasServiceConnectDefaults {
		cluster.ServiceConnectDefaultsNamespace = strings.TrimSpace(serviceConnectNamespace)
	}

	cluster.UpdatedAt = time.Now().UTC()
	return cloneCluster(cluster), nil
}

func (s *Service) DeleteCluster(clusterRef string) (Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster := s.resolveClusterLocked(clusterRef)
	if cluster == nil {
		return Cluster{}, ErrClusterNotFound
	}
	out := cloneCluster(cluster)
	out.Status = "INACTIVE"
	out.UpdatedAt = time.Now().UTC()

	delete(s.tagsByARN, cluster.ClusterArn)
	delete(s.clustersByARN, cluster.ClusterArn)
	delete(s.clustersByName, cluster.ClusterName)
	return out, nil
}

func (s *Service) CreateCapacityProvider(name, autoScalingGroupArn, managedScalingStatus, managedTerminationProtection string, tags map[string]string) (CapacityProvider, error) {
	name = strings.TrimSpace(name)
	autoScalingGroupArn = strings.TrimSpace(autoScalingGroupArn)
	managedScalingStatus = strings.TrimSpace(managedScalingStatus)
	managedTerminationProtection = strings.TrimSpace(managedTerminationProtection)
	if name == "" {
		return CapacityProvider{}, ErrInvalidParameter
	}
	normalizedTags, err := normalizeTags(tags)
	if err != nil {
		return CapacityProvider{}, err
	}
	if managedScalingStatus == "" {
		managedScalingStatus = "DISABLED"
	}
	if managedTerminationProtection == "" {
		managedTerminationProtection = "DISABLED"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.capacityProviders[name]; exists {
		return CapacityProvider{}, ErrAlreadyExists
	}

	now := time.Now().UTC()
	cp := &CapacityProvider{
		Name:                         name,
		Arn:                          capacityProviderARN(name),
		Status:                       "ACTIVE",
		UpdateStatus:                 "UPDATE_COMPLETE",
		AutoScalingGroupArn:          autoScalingGroupArn,
		ManagedScalingStatus:         managedScalingStatus,
		ManagedTerminationProtection: managedTerminationProtection,
		Tags:                         normalizedTags,
		CreatedAt:                    now,
		UpdatedAt:                    now,
	}
	s.capacityProviders[cp.Name] = cp
	if len(cp.Tags) > 0 {
		s.tagsByARN[cp.Arn] = cloneStringMap(cp.Tags)
	}
	return cloneCapacityProvider(cp), nil
}

func (s *Service) DescribeCapacityProviders(refs []string) ([]CapacityProvider, []Failure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(refs) == 0 {
		out := make([]CapacityProvider, 0, len(s.capacityProviders))
		for _, cp := range s.capacityProviders {
			out = append(out, cloneCapacityProvider(cp))
		}
		sort.Slice(out, func(i, j int) bool {
			return out[i].Name < out[j].Name
		})
		return out, nil, nil
	}

	out := make([]CapacityProvider, 0, len(refs))
	failures := make([]Failure, 0)
	seen := map[string]struct{}{}
	for _, ref := range refs {
		cp := s.resolveCapacityProviderLocked(ref)
		if cp == nil {
			failures = append(failures, Failure{
				Arn:    strings.TrimSpace(ref),
				Reason: "MISSING",
				Detail: "capacity provider not found",
			})
			continue
		}
		if _, exists := seen[cp.Arn]; exists {
			continue
		}
		seen[cp.Arn] = struct{}{}
		out = append(out, cloneCapacityProvider(cp))
	}
	return out, failures, nil
}

func (s *Service) UpdateCapacityProvider(ref, autoScalingGroupArn, managedScalingStatus, managedTerminationProtection string) (CapacityProvider, error) {
	autoScalingGroupArn = strings.TrimSpace(autoScalingGroupArn)
	managedScalingStatus = strings.TrimSpace(managedScalingStatus)
	managedTerminationProtection = strings.TrimSpace(managedTerminationProtection)

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := s.resolveCapacityProviderLocked(ref)
	if cp == nil {
		return CapacityProvider{}, ErrCapacityProviderNotFound
	}
	if autoScalingGroupArn != "" {
		cp.AutoScalingGroupArn = autoScalingGroupArn
	}
	if managedScalingStatus != "" {
		cp.ManagedScalingStatus = managedScalingStatus
	}
	if managedTerminationProtection != "" {
		cp.ManagedTerminationProtection = managedTerminationProtection
	}
	cp.UpdatedAt = time.Now().UTC()
	cp.UpdateStatus = "UPDATE_COMPLETE"
	return cloneCapacityProvider(cp), nil
}

func (s *Service) DeleteCapacityProvider(ref string) (CapacityProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := s.resolveCapacityProviderLocked(ref)
	if cp == nil {
		return CapacityProvider{}, ErrCapacityProviderNotFound
	}
	out := cloneCapacityProvider(cp)

	for _, cluster := range s.clustersByName {
		filtered := make([]string, 0, len(cluster.CapacityProviders))
		for _, provider := range cluster.CapacityProviders {
			if provider != cp.Name {
				filtered = append(filtered, provider)
			}
		}
		cluster.CapacityProviders = filtered

		strategy := make([]CapacityProviderStrategyItem, 0, len(cluster.DefaultCapacityProviderStrategy))
		for _, item := range cluster.DefaultCapacityProviderStrategy {
			if item.CapacityProvider != cp.Name {
				strategy = append(strategy, item)
			}
		}
		cluster.DefaultCapacityProviderStrategy = strategy
	}

	delete(s.tagsByARN, cp.Arn)
	delete(s.capacityProviders, cp.Name)
	return out, nil
}

func (s *Service) PutClusterCapacityProviders(clusterRef string, capacityProviders []string, strategy []CapacityProviderStrategyItem) (Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster := s.resolveClusterLocked(clusterRef)
	if cluster == nil {
		return Cluster{}, ErrClusterNotFound
	}

	dedupedProviders := make([]string, 0, len(capacityProviders))
	seenProviders := map[string]struct{}{}
	for _, ref := range capacityProviders {
		cp := s.resolveCapacityProviderLocked(ref)
		if cp == nil {
			return Cluster{}, ErrCapacityProviderNotFound
		}
		if _, exists := seenProviders[cp.Name]; exists {
			continue
		}
		seenProviders[cp.Name] = struct{}{}
		dedupedProviders = append(dedupedProviders, cp.Name)
	}

	normalizedStrategy := make([]CapacityProviderStrategyItem, 0, len(strategy))
	for _, item := range strategy {
		cp := s.resolveCapacityProviderLocked(item.CapacityProvider)
		if cp == nil {
			return Cluster{}, ErrCapacityProviderNotFound
		}
		normalizedStrategy = append(normalizedStrategy, CapacityProviderStrategyItem{
			CapacityProvider: cp.Name,
			Base:             item.Base,
			Weight:           item.Weight,
		})
	}

	cluster.CapacityProviders = dedupedProviders
	cluster.DefaultCapacityProviderStrategy = normalizedStrategy
	cluster.UpdatedAt = time.Now().UTC()
	return cloneCluster(cluster), nil
}

func (s *Service) RegisterTaskDefinition(input TaskDefinitionInput) (TaskDefinition, error) {
	family := strings.TrimSpace(input.Family)
	if family == "" {
		return TaskDefinition{}, ErrInvalidParameter
	}
	tags, err := normalizeTags(input.Tags)
	if err != nil {
		return TaskDefinition{}, err
	}
	requires := make([]string, 0, len(input.RequiresCompatibilities))
	seenReq := map[string]struct{}{}
	for _, raw := range input.RequiresCompatibilities {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		if _, exists := seenReq[item]; exists {
			continue
		}
		seenReq[item] = struct{}{}
		requires = append(requires, item)
	}

	containers := make([]ContainerDefinition, 0, len(input.ContainerDefinitions))
	for _, def := range input.ContainerDefinitions {
		name := strings.TrimSpace(def.Name)
		image := strings.TrimSpace(def.Image)
		if name == "" && image == "" {
			continue
		}
		if name == "" {
			return TaskDefinition{}, ErrInvalidParameter
		}
		containers = append(containers, ContainerDefinition{Name: name, Image: image})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	revision := int64(1)
	if defs := s.taskDefinitions[family]; len(defs) > 0 {
		revision = defs[len(defs)-1].Revision + 1
	}
	now := time.Now().UTC()
	def := &TaskDefinition{
		Family:                  family,
		Revision:                revision,
		Arn:                     taskDefinitionARN(family, revision),
		Status:                  "ACTIVE",
		NetworkMode:             strings.TrimSpace(input.NetworkMode),
		Cpu:                     strings.TrimSpace(input.Cpu),
		Memory:                  strings.TrimSpace(input.Memory),
		ExecutionRoleArn:        strings.TrimSpace(input.ExecutionRoleArn),
		TaskRoleArn:             strings.TrimSpace(input.TaskRoleArn),
		RequiresCompatibilities: requires,
		ContainerDefinitions:    containers,
		Tags:                    tags,
		RegisteredAt:            now,
	}
	s.taskDefinitions[family] = append(s.taskDefinitions[family], def)
	s.taskDefinitionsByARN[def.Arn] = def
	if len(def.Tags) > 0 {
		s.tagsByARN[def.Arn] = cloneStringMap(def.Tags)
	}
	return cloneTaskDefinition(def), nil
}

func (s *Service) DescribeTaskDefinition(ref string) (TaskDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	def := s.resolveTaskDefinitionLocked(ref)
	if def == nil {
		return TaskDefinition{}, ErrTaskDefinitionNotFound
	}
	return cloneTaskDefinition(def), nil
}

func (s *Service) ListTaskDefinitions(familyPrefix, status, sortOrder, nextToken string, maxResults int32) ([]string, string, error) {
	familyPrefix = strings.TrimSpace(familyPrefix)
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "" {
		status = "ACTIVE"
	}
	if status != "ACTIVE" && status != "INACTIVE" && status != "ALL" {
		return nil, "", ErrInvalidParameter
	}
	sortOrder = strings.ToUpper(strings.TrimSpace(sortOrder))
	if sortOrder == "" {
		sortOrder = "DESC"
	}
	if sortOrder != "ASC" && sortOrder != "DESC" {
		return nil, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	all := make([]TaskDefinition, 0)
	for family, defs := range s.taskDefinitions {
		if familyPrefix != "" && !strings.HasPrefix(family, familyPrefix) {
			continue
		}
		for _, def := range defs {
			if def == nil {
				continue
			}
			if status != "ALL" && def.Status != status {
				continue
			}
			all = append(all, cloneTaskDefinition(def))
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Family == all[j].Family {
			if sortOrder == "ASC" {
				return all[i].Revision < all[j].Revision
			}
			return all[i].Revision > all[j].Revision
		}
		if sortOrder == "ASC" {
			return all[i].Family < all[j].Family
		}
		return all[i].Family > all[j].Family
	})

	offset := 0
	nextToken = strings.TrimSpace(nextToken)
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 0 || parsed > len(all) {
			return nil, "", ErrInvalidParameter
		}
		offset = parsed
	}
	limit := int(maxResults)
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}

	out := make([]string, 0, end-offset)
	for _, def := range all[offset:end] {
		out = append(out, def.Arn)
	}
	newNextToken := ""
	if end < len(all) {
		newNextToken = strconv.Itoa(end)
	}
	return out, newNextToken, nil
}

func (s *Service) ListTaskDefinitionFamilies(familyPrefix, status, nextToken string, maxResults int32) ([]string, string, error) {
	familyPrefix = strings.TrimSpace(familyPrefix)
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "" {
		status = "ACTIVE"
	}
	if status != "ACTIVE" && status != "INACTIVE" && status != "ALL" {
		return nil, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	familiesSet := map[string]struct{}{}
	for family, defs := range s.taskDefinitions {
		if familyPrefix != "" && !strings.HasPrefix(family, familyPrefix) {
			continue
		}
		include := false
		for _, def := range defs {
			if def == nil {
				continue
			}
			if status == "ALL" || def.Status == status {
				include = true
				break
			}
		}
		if include {
			familiesSet[family] = struct{}{}
		}
	}

	families := make([]string, 0, len(familiesSet))
	for family := range familiesSet {
		families = append(families, family)
	}
	sort.Strings(families)

	offset := 0
	nextToken = strings.TrimSpace(nextToken)
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 0 || parsed > len(families) {
			return nil, "", ErrInvalidParameter
		}
		offset = parsed
	}
	limit := int(maxResults)
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	end := offset + limit
	if end > len(families) {
		end = len(families)
	}
	out := append([]string(nil), families[offset:end]...)
	newNextToken := ""
	if end < len(families) {
		newNextToken = strconv.Itoa(end)
	}
	return out, newNextToken, nil
}

func (s *Service) DeregisterTaskDefinition(ref string) (TaskDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	def := s.resolveTaskDefinitionLocked(ref)
	if def == nil {
		return TaskDefinition{}, ErrTaskDefinitionNotFound
	}
	if def.Status != "INACTIVE" {
		def.Status = "INACTIVE"
		now := time.Now().UTC()
		def.DeregisteredAt = &now
	}
	return cloneTaskDefinition(def), nil
}

func (s *Service) DeleteTaskDefinitions(refs []string) ([]TaskDefinition, []Failure, error) {
	if len(refs) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	deleted := make([]TaskDefinition, 0, len(refs))
	failures := make([]Failure, 0)
	for _, ref := range refs {
		def := s.resolveTaskDefinitionLocked(ref)
		if def == nil {
			failures = append(failures, Failure{
				Arn:    strings.TrimSpace(ref),
				Reason: "MISSING",
				Detail: "task definition not found",
			})
			continue
		}
		delete(s.taskDefinitionsByARN, def.Arn)
		delete(s.tagsByARN, def.Arn)
		familyDefs := s.taskDefinitions[def.Family]
		filtered := make([]*TaskDefinition, 0, len(familyDefs))
		for _, item := range familyDefs {
			if item != nil && item.Arn != def.Arn {
				filtered = append(filtered, item)
			}
		}
		if len(filtered) == 0 {
			delete(s.taskDefinitions, def.Family)
		} else {
			s.taskDefinitions[def.Family] = filtered
		}
		deleted = append(deleted, cloneTaskDefinition(def))
	}
	return deleted, failures, nil
}

func (s *Service) CreateService(clusterRef, serviceName, taskDefinitionRef, launchType string, desiredCount int32, hasDesiredCount bool, tags map[string]string) (ServiceDefinition, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	serviceName = strings.TrimSpace(serviceName)
	taskDefinitionRef = strings.TrimSpace(taskDefinitionRef)
	launchType = strings.TrimSpace(launchType)
	if clusterRef == "" || serviceName == "" || taskDefinitionRef == "" {
		return ServiceDefinition{}, ErrInvalidParameter
	}
	if !hasDesiredCount {
		desiredCount = 1
	}
	if desiredCount < 0 {
		return ServiceDefinition{}, ErrInvalidParameter
	}
	if launchType == "" {
		launchType = "EC2"
	}
	normalizedTags, err := normalizeTags(tags)
	if err != nil {
		return ServiceDefinition{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster := s.resolveClusterLocked(clusterRef)
	if cluster == nil {
		return ServiceDefinition{}, ErrClusterNotFound
	}
	if existing := s.resolveServiceLocked(cluster.ClusterArn, serviceName); existing != nil {
		return ServiceDefinition{}, ErrAlreadyExists
	}

	taskDefinition := s.resolveTaskDefinitionLocked(taskDefinitionRef)
	if taskDefinition == nil {
		return ServiceDefinition{}, ErrTaskDefinitionNotFound
	}

	now := time.Now().UTC()
	service := &ServiceDefinition{
		Name:              serviceName,
		Arn:               serviceARN(cluster.ClusterName, serviceName),
		ClusterArn:        cluster.ClusterArn,
		TaskDefinitionArn: taskDefinition.Arn,
		DesiredCount:      desiredCount,
		LaunchType:        launchType,
		Status:            "ACTIVE",
		Tags:              normalizedTags,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	s.services[service.Arn] = service
	if len(service.Tags) > 0 {
		s.tagsByARN[service.Arn] = cloneStringMap(service.Tags)
	}

	cluster.ActiveServicesCount++
	cluster.UpdatedAt = now

	revision := s.createServiceRevisionLocked(service, now)
	s.createServiceDeploymentLocked(service, revision.Arn, "COMPLETED", "", now)
	return cloneServiceDefinition(service), nil
}

func (s *Service) DescribeServices(clusterRef string, refs []string) ([]ServiceDefinition, []Failure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterRef = strings.TrimSpace(clusterRef)
	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, nil, ErrClusterNotFound
	}

	if len(refs) == 0 {
		out := make([]ServiceDefinition, 0, len(s.services))
		for _, service := range s.services {
			if clusterRef != "" && !serviceInCluster(service, clusterRef, s) {
				continue
			}
			out = append(out, cloneServiceDefinition(service))
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].ClusterArn == out[j].ClusterArn {
				return out[i].Name < out[j].Name
			}
			return out[i].ClusterArn < out[j].ClusterArn
		})
		return out, nil, nil
	}

	seen := map[string]struct{}{}
	out := make([]ServiceDefinition, 0, len(refs))
	failures := make([]Failure, 0)
	for _, ref := range refs {
		service := s.resolveServiceLocked(clusterRef, ref)
		if service == nil {
			failures = append(failures, Failure{
				Arn:    strings.TrimSpace(ref),
				Reason: "MISSING",
				Detail: "service not found",
			})
			continue
		}
		if _, exists := seen[service.Arn]; exists {
			continue
		}
		seen[service.Arn] = struct{}{}
		out = append(out, cloneServiceDefinition(service))
	}
	return out, failures, nil
}

func (s *Service) ListServices(clusterRef, nextToken string, maxResults int32) ([]string, string, error) {
	clusterRef = strings.TrimSpace(clusterRef)

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, "", ErrClusterNotFound
	}

	arns := make([]string, 0, len(s.services))
	for _, service := range s.services {
		if clusterRef != "" && !serviceInCluster(service, clusterRef, s) {
			continue
		}
		arns = append(arns, service.Arn)
	}
	sort.Strings(arns)

	offset := 0
	nextToken = strings.TrimSpace(nextToken)
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 0 || parsed > len(arns) {
			return nil, "", ErrInvalidParameter
		}
		offset = parsed
	}
	limit := int(maxResults)
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	end := offset + limit
	if end > len(arns) {
		end = len(arns)
	}
	out := append([]string(nil), arns[offset:end]...)

	newNextToken := ""
	if end < len(arns) {
		newNextToken = strconv.Itoa(end)
	}
	return out, newNextToken, nil
}

func (s *Service) ListServicesByLaunchType(clusterRef, launchType, nextToken string, maxResults int32) ([]string, string, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	launchType = strings.TrimSpace(launchType)
	if launchType == "" {
		return nil, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, "", ErrClusterNotFound
	}

	arns := make([]string, 0, len(s.services))
	for _, service := range s.services {
		if clusterRef != "" && !serviceInCluster(service, clusterRef, s) {
			continue
		}
		if service.LaunchType != launchType {
			continue
		}
		arns = append(arns, service.Arn)
	}
	sort.Strings(arns)
	return paginateStringList(arns, nextToken, maxResults)
}

func (s *Service) ListServicesByNamespace(clusterRef, namespace, nextToken string, maxResults int32) ([]string, string, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, "", ErrClusterNotFound
	}

	arns := make([]string, 0, len(s.services))
	for _, service := range s.services {
		if clusterRef != "" && !serviceInCluster(service, clusterRef, s) {
			continue
		}
		if strings.TrimSpace(service.Tags["namespace"]) != namespace {
			continue
		}
		arns = append(arns, service.Arn)
	}
	sort.Strings(arns)
	return paginateStringList(arns, nextToken, maxResults)
}

func (s *Service) UpdateService(clusterRef, serviceRef, taskDefinitionRef, launchType string, desiredCount int32, hasDesiredCount bool) (ServiceDefinition, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	serviceRef = strings.TrimSpace(serviceRef)
	taskDefinitionRef = strings.TrimSpace(taskDefinitionRef)
	launchType = strings.TrimSpace(launchType)
	if serviceRef == "" {
		return ServiceDefinition{}, ErrInvalidParameter
	}
	if hasDesiredCount && desiredCount < 0 {
		return ServiceDefinition{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return ServiceDefinition{}, ErrClusterNotFound
	}
	service := s.resolveServiceLocked(clusterRef, serviceRef)
	if service == nil {
		return ServiceDefinition{}, ErrServiceNotFound
	}

	var taskDefinition *TaskDefinition
	revisionChange := false
	if taskDefinitionRef != "" {
		taskDefinition = s.resolveTaskDefinitionLocked(taskDefinitionRef)
		if taskDefinition == nil {
			return ServiceDefinition{}, ErrTaskDefinitionNotFound
		}
		if service.TaskDefinitionArn != taskDefinition.Arn {
			service.TaskDefinitionArn = taskDefinition.Arn
			revisionChange = true
		}
	}
	if hasDesiredCount && service.DesiredCount != desiredCount {
		service.DesiredCount = desiredCount
		revisionChange = true
	}
	if launchType != "" {
		service.LaunchType = launchType
	}
	now := time.Now().UTC()
	service.UpdatedAt = now

	if revisionChange {
		revision := s.createServiceRevisionLocked(service, now)
		s.createServiceDeploymentLocked(service, revision.Arn, "COMPLETED", "", now)
	}
	return cloneServiceDefinition(service), nil
}

func (s *Service) DeleteService(clusterRef, serviceRef string) (ServiceDefinition, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	serviceRef = strings.TrimSpace(serviceRef)
	if serviceRef == "" {
		return ServiceDefinition{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return ServiceDefinition{}, ErrClusterNotFound
	}
	service := s.resolveServiceLocked(clusterRef, serviceRef)
	if service == nil {
		return ServiceDefinition{}, ErrServiceNotFound
	}

	now := time.Now().UTC()
	out := cloneServiceDefinition(service)
	out.Status = "INACTIVE"
	out.UpdatedAt = now

	if cluster := s.resolveClusterLocked(service.ClusterArn); cluster != nil {
		if cluster.ActiveServicesCount > 0 {
			cluster.ActiveServicesCount--
		}
		cluster.UpdatedAt = now
	}

	delete(s.tagsByARN, service.Arn)
	delete(s.services, service.Arn)
	delete(s.serviceDeployments, service.Arn)
	delete(s.serviceRevisions, service.Arn)
	delete(s.taskSetSeqBySvcARN, service.Arn)
	for taskSetARN, taskSet := range s.taskSets {
		if taskSet.ServiceArn == service.Arn {
			delete(s.taskSets, taskSetARN)
		}
	}
	for taskARN, task := range s.tasks {
		if task.ServiceArn == service.Arn {
			delete(s.taskProtectionByTask, taskARN)
			delete(s.tasks, taskARN)
		}
	}
	return out, nil
}

func (s *Service) CreateTaskSet(clusterRef, serviceRef, taskDefinitionRef, launchType string, scaleValue float64, hasScale bool) (TaskSet, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	serviceRef = strings.TrimSpace(serviceRef)
	taskDefinitionRef = strings.TrimSpace(taskDefinitionRef)
	launchType = strings.TrimSpace(launchType)
	if clusterRef == "" || serviceRef == "" || taskDefinitionRef == "" {
		return TaskSet{}, ErrInvalidParameter
	}
	if !hasScale {
		scaleValue = 100.0
	}
	if scaleValue < 0 {
		return TaskSet{}, ErrInvalidParameter
	}
	if launchType == "" {
		launchType = "EC2"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster := s.resolveClusterLocked(clusterRef)
	if cluster == nil {
		return TaskSet{}, ErrClusterNotFound
	}
	service := s.resolveServiceLocked(cluster.ClusterArn, serviceRef)
	if service == nil {
		return TaskSet{}, ErrServiceNotFound
	}
	taskDefinition := s.resolveTaskDefinitionLocked(taskDefinitionRef)
	if taskDefinition == nil {
		return TaskSet{}, ErrTaskDefinitionNotFound
	}

	now := time.Now().UTC()
	seq := s.taskSetSeqBySvcARN[service.Arn] + 1
	s.taskSetSeqBySvcARN[service.Arn] = seq
	id := strconv.Itoa(seq)
	taskSet := &TaskSet{
		Arn:               taskSetARN(cluster.ClusterName, service.Name, id),
		ID:                id,
		ClusterArn:        cluster.ClusterArn,
		ServiceArn:        service.Arn,
		TaskDefinitionArn: taskDefinition.Arn,
		ComputedDesired:   service.DesiredCount,
		PendingCount:      0,
		RunningCount:      service.DesiredCount,
		Status:            "ACTIVE",
		LaunchType:        launchType,
		ScaleValue:        scaleValue,
		ScaleUnit:         "PERCENT",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	s.taskSets[taskSet.Arn] = taskSet
	if service.PrimaryTaskSetArn == "" {
		service.PrimaryTaskSetArn = taskSet.Arn
		service.UpdatedAt = now
	}
	return cloneTaskSet(taskSet), nil
}

func (s *Service) DescribeTaskSets(clusterRef, serviceRef string, refs []string) ([]TaskSet, []Failure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	service, err := s.resolveServiceForTaskSetOpsLocked(clusterRef, serviceRef)
	if err != nil {
		return nil, nil, err
	}

	if len(refs) == 0 {
		out := s.listTaskSetsForServiceLocked(service.Arn)
		return out, nil, nil
	}

	seen := map[string]struct{}{}
	out := make([]TaskSet, 0, len(refs))
	failures := make([]Failure, 0)
	for _, ref := range refs {
		taskSet := s.resolveTaskSetLocked(service.Arn, ref)
		if taskSet == nil {
			failures = append(failures, Failure{
				Arn:    strings.TrimSpace(ref),
				Reason: "MISSING",
				Detail: "task set not found",
			})
			continue
		}
		if _, exists := seen[taskSet.Arn]; exists {
			continue
		}
		seen[taskSet.Arn] = struct{}{}
		out = append(out, cloneTaskSet(taskSet))
	}
	return out, failures, nil
}

func (s *Service) UpdateTaskSet(clusterRef, serviceRef, taskSetRef string, scaleValue float64, hasScale bool) (TaskSet, error) {
	taskSetRef = strings.TrimSpace(taskSetRef)
	if taskSetRef == "" {
		return TaskSet{}, ErrInvalidParameter
	}
	if hasScale && scaleValue < 0 {
		return TaskSet{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	service, err := s.resolveServiceForTaskSetOpsLocked(clusterRef, serviceRef)
	if err != nil {
		return TaskSet{}, err
	}
	taskSet := s.resolveTaskSetLocked(service.Arn, taskSetRef)
	if taskSet == nil {
		return TaskSet{}, ErrTaskSetNotFound
	}
	if hasScale {
		taskSet.ScaleValue = scaleValue
	}
	taskSet.UpdatedAt = time.Now().UTC()
	return cloneTaskSet(taskSet), nil
}

func (s *Service) UpdateServicePrimaryTaskSet(clusterRef, serviceRef, primaryTaskSetRef string) (ServiceDefinition, error) {
	primaryTaskSetRef = strings.TrimSpace(primaryTaskSetRef)
	if primaryTaskSetRef == "" {
		return ServiceDefinition{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	service, err := s.resolveServiceForTaskSetOpsLocked(clusterRef, serviceRef)
	if err != nil {
		return ServiceDefinition{}, err
	}
	taskSet := s.resolveTaskSetLocked(service.Arn, primaryTaskSetRef)
	if taskSet == nil {
		return ServiceDefinition{}, ErrTaskSetNotFound
	}
	service.PrimaryTaskSetArn = taskSet.Arn
	service.UpdatedAt = time.Now().UTC()
	return cloneServiceDefinition(service), nil
}

func (s *Service) DeleteTaskSet(clusterRef, serviceRef, taskSetRef string) (TaskSet, error) {
	taskSetRef = strings.TrimSpace(taskSetRef)
	if taskSetRef == "" {
		return TaskSet{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	service, err := s.resolveServiceForTaskSetOpsLocked(clusterRef, serviceRef)
	if err != nil {
		return TaskSet{}, err
	}
	taskSet := s.resolveTaskSetLocked(service.Arn, taskSetRef)
	if taskSet == nil {
		return TaskSet{}, ErrTaskSetNotFound
	}
	out := cloneTaskSet(taskSet)
	out.Status = "INACTIVE"
	out.UpdatedAt = time.Now().UTC()

	delete(s.taskSets, taskSet.Arn)
	if service.PrimaryTaskSetArn == taskSet.Arn {
		service.PrimaryTaskSetArn = ""
		remaining := s.listTaskSetsForServiceLocked(service.Arn)
		if len(remaining) > 0 {
			service.PrimaryTaskSetArn = remaining[0].Arn
		}
		service.UpdatedAt = out.UpdatedAt
	}
	return out, nil
}

func (s *Service) RunTask(clusterRef, taskDefinitionRef, launchType, startedBy, group, serviceRef string, count int32, hasCount bool) ([]Task, []Failure, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	taskDefinitionRef = strings.TrimSpace(taskDefinitionRef)
	launchType = strings.TrimSpace(launchType)
	startedBy = strings.TrimSpace(startedBy)
	group = strings.TrimSpace(group)
	serviceRef = strings.TrimSpace(serviceRef)
	if clusterRef == "" || taskDefinitionRef == "" {
		return nil, nil, ErrInvalidParameter
	}
	if !hasCount {
		count = 1
	}
	if count <= 0 {
		return nil, nil, ErrInvalidParameter
	}
	if count > 10 {
		count = 10
	}
	if launchType == "" {
		launchType = "EC2"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster := s.resolveClusterLocked(clusterRef)
	if cluster == nil {
		return nil, nil, ErrClusterNotFound
	}
	taskDefinition := s.resolveTaskDefinitionLocked(taskDefinitionRef)
	if taskDefinition == nil {
		return nil, nil, ErrTaskDefinitionNotFound
	}
	serviceArn := ""
	if serviceRef != "" {
		service := s.resolveServiceLocked(cluster.ClusterArn, serviceRef)
		if service == nil {
			return nil, nil, ErrServiceNotFound
		}
		serviceArn = service.Arn
		if group == "" {
			group = "service:" + service.Name
		}
	}
	if group == "" {
		group = "family:" + taskDefinition.Family
	}

	now := time.Now().UTC()
	out := make([]Task, 0, count)
	for idx := int32(0); idx < count; idx++ {
		s.taskSeq++
		task := &Task{
			Arn:               taskARN(cluster.ClusterName, s.taskSeq),
			ClusterArn:        cluster.ClusterArn,
			TaskDefinitionArn: taskDefinition.Arn,
			ServiceArn:        serviceArn,
			Group:             group,
			StartedBy:         startedBy,
			LaunchType:        launchType,
			LastStatus:        "RUNNING",
			DesiredStatus:     "RUNNING",
			CreatedAt:         now,
			StartedAt:         now,
		}
		s.tasks[task.Arn] = task
		out = append(out, cloneTask(task))
		cluster.RunningTasksCount++
		cluster.UpdatedAt = now
	}
	return out, nil, nil
}

func (s *Service) StartTask(clusterRef, taskDefinitionRef, startedBy, group, serviceRef string, containerInstanceRefs []string) ([]Task, []Failure, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	taskDefinitionRef = strings.TrimSpace(taskDefinitionRef)
	startedBy = strings.TrimSpace(startedBy)
	group = strings.TrimSpace(group)
	serviceRef = strings.TrimSpace(serviceRef)
	if clusterRef == "" || taskDefinitionRef == "" {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster := s.resolveClusterLocked(clusterRef)
	if cluster == nil {
		return nil, nil, ErrClusterNotFound
	}
	taskDefinition := s.resolveTaskDefinitionLocked(taskDefinitionRef)
	if taskDefinition == nil {
		return nil, nil, ErrTaskDefinitionNotFound
	}
	serviceArn := ""
	if serviceRef != "" {
		service := s.resolveServiceLocked(cluster.ClusterArn, serviceRef)
		if service == nil {
			return nil, nil, ErrServiceNotFound
		}
		serviceArn = service.Arn
	}

	containerInstanceArn := ""
	if len(containerInstanceRefs) > 0 {
		containerInstanceArn = strings.TrimSpace(containerInstanceRefs[0])
	}
	if group == "" {
		group = "family:" + taskDefinition.Family
	}

	now := time.Now().UTC()
	s.taskSeq++
	task := &Task{
		Arn:                  taskARN(cluster.ClusterName, s.taskSeq),
		ClusterArn:           cluster.ClusterArn,
		TaskDefinitionArn:    taskDefinition.Arn,
		ServiceArn:           serviceArn,
		ContainerInstanceArn: containerInstanceArn,
		Group:                group,
		StartedBy:            startedBy,
		LaunchType:           "EC2",
		LastStatus:           "RUNNING",
		DesiredStatus:        "RUNNING",
		CreatedAt:            now,
		StartedAt:            now,
	}
	s.tasks[task.Arn] = task
	cluster.RunningTasksCount++
	cluster.UpdatedAt = now
	return []Task{cloneTask(task)}, nil, nil
}

func (s *Service) StopTask(clusterRef, taskRef, reason string) (Task, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	taskRef = strings.TrimSpace(taskRef)
	reason = strings.TrimSpace(reason)
	if taskRef == "" {
		return Task{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return Task{}, ErrClusterNotFound
	}
	task := s.resolveTaskLocked(clusterRef, taskRef)
	if task == nil {
		return Task{}, ErrTaskNotFound
	}

	now := time.Now().UTC()
	task.DesiredStatus = "STOPPED"
	task.LastStatus = "STOPPED"
	task.StoppedReason = reason
	task.StoppedAt = &now
	if cluster := s.resolveClusterLocked(task.ClusterArn); cluster != nil {
		if cluster.RunningTasksCount > 0 {
			cluster.RunningTasksCount--
		}
		cluster.UpdatedAt = now
	}
	return cloneTask(task), nil
}

func (s *Service) DescribeTasks(clusterRef string, refs []string) ([]Task, []Failure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterRef = strings.TrimSpace(clusterRef)
	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, nil, ErrClusterNotFound
	}

	if len(refs) == 0 {
		out := make([]Task, 0, len(s.tasks))
		for _, task := range s.tasks {
			if clusterRef != "" && !taskInCluster(task, clusterRef, s) {
				continue
			}
			out = append(out, cloneTask(task))
		}
		sort.Slice(out, func(i, j int) bool {
			return out[i].StartedAt.Before(out[j].StartedAt)
		})
		return out, nil, nil
	}

	seen := map[string]struct{}{}
	out := make([]Task, 0, len(refs))
	failures := make([]Failure, 0)
	for _, ref := range refs {
		task := s.resolveTaskLocked(clusterRef, ref)
		if task == nil {
			failures = append(failures, Failure{
				Arn:    strings.TrimSpace(ref),
				Reason: "MISSING",
				Detail: "task not found",
			})
			continue
		}
		if _, exists := seen[task.Arn]; exists {
			continue
		}
		seen[task.Arn] = struct{}{}
		out = append(out, cloneTask(task))
	}
	return out, failures, nil
}

func (s *Service) ListTasks(clusterRef, serviceRef, family, desiredStatus, launchType, startedBy, containerInstanceRef, nextToken string, maxResults int32) ([]string, string, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	serviceRef = strings.TrimSpace(serviceRef)
	family = strings.TrimSpace(family)
	desiredStatus = strings.ToUpper(strings.TrimSpace(desiredStatus))
	launchType = strings.TrimSpace(launchType)
	startedBy = strings.TrimSpace(startedBy)
	containerInstanceRef = strings.TrimSpace(containerInstanceRef)

	if desiredStatus == "" {
		desiredStatus = "RUNNING"
	}
	if desiredStatus != "RUNNING" && desiredStatus != "STOPPED" {
		return nil, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, "", ErrClusterNotFound
	}

	serviceArn := ""
	if serviceRef != "" {
		service := s.resolveServiceLocked(clusterRef, serviceRef)
		if service == nil {
			return nil, "", ErrServiceNotFound
		}
		serviceArn = service.Arn
	}

	out := make([]string, 0, len(s.tasks))
	for _, task := range s.tasks {
		if clusterRef != "" && !taskInCluster(task, clusterRef, s) {
			continue
		}
		if serviceArn != "" && task.ServiceArn != serviceArn {
			continue
		}
		if family != "" {
			def := s.resolveTaskDefinitionLocked(task.TaskDefinitionArn)
			if def == nil || def.Family != family {
				continue
			}
		}
		if desiredStatus != "" && task.DesiredStatus != desiredStatus {
			continue
		}
		if launchType != "" && task.LaunchType != launchType {
			continue
		}
		if startedBy != "" && task.StartedBy != startedBy {
			continue
		}
		if containerInstanceRef != "" && task.ContainerInstanceArn != containerInstanceRef {
			continue
		}
		out = append(out, task.Arn)
	}
	sort.Strings(out)
	return paginateStringList(out, nextToken, maxResults)
}

func (s *Service) ExecuteCommand(clusterRef, taskRef, containerName, command string, interactive bool) (ExecuteCommandResult, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	taskRef = strings.TrimSpace(taskRef)
	containerName = strings.TrimSpace(containerName)
	command = strings.TrimSpace(command)
	if taskRef == "" || command == "" {
		return ExecuteCommandResult{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return ExecuteCommandResult{}, ErrClusterNotFound
	}
	task := s.resolveTaskLocked(clusterRef, taskRef)
	if task == nil {
		return ExecuteCommandResult{}, ErrTaskNotFound
	}
	if task.LastStatus != "RUNNING" {
		return ExecuteCommandResult{}, ErrInvalidParameter
	}
	if containerName == "" {
		containerName = "default"
	}

	s.execSeq++
	return ExecuteCommandResult{
		ClusterArn:    task.ClusterArn,
		TaskArn:       task.Arn,
		ContainerName: containerName,
		Interactive:   interactive,
		SessionID:     fmt.Sprintf("ecs-exec-%d", s.execSeq),
		StreamURL:     fmt.Sprintf("wss://ssmmessages.%s.amazonaws.com/v1/data-channel/ecs-exec-%d", DefaultRegion, s.execSeq),
		TokenValue:    fmt.Sprintf("token-%d", s.execSeq),
	}, nil
}

func (s *Service) GetTaskProtection(clusterRef string, taskRefs []string) ([]TaskProtection, []Failure, error) {
	clusterRef = strings.TrimSpace(clusterRef)

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, nil, ErrClusterNotFound
	}

	if len(taskRefs) == 0 {
		out := make([]TaskProtection, 0)
		for _, task := range s.tasks {
			if clusterRef != "" && !taskInCluster(task, clusterRef, s) {
				continue
			}
			out = append(out, s.taskProtectionForTaskLocked(task.Arn))
		}
		sort.Slice(out, func(i, j int) bool { return out[i].TaskArn < out[j].TaskArn })
		return out, nil, nil
	}

	out := make([]TaskProtection, 0, len(taskRefs))
	failures := make([]Failure, 0)
	seen := map[string]struct{}{}
	for _, ref := range taskRefs {
		task := s.resolveTaskLocked(clusterRef, ref)
		if task == nil {
			failures = append(failures, Failure{
				Arn:    strings.TrimSpace(ref),
				Reason: "MISSING",
				Detail: "task not found",
			})
			continue
		}
		if _, exists := seen[task.Arn]; exists {
			continue
		}
		seen[task.Arn] = struct{}{}
		out = append(out, s.taskProtectionForTaskLocked(task.Arn))
	}
	return out, failures, nil
}

func (s *Service) UpdateTaskProtection(clusterRef string, taskRefs []string, protectionEnabled bool, expiresInMinutes int64, hasExpires bool) ([]TaskProtection, []Failure, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	if len(taskRefs) == 0 {
		return nil, nil, ErrInvalidParameter
	}
	if hasExpires && expiresInMinutes <= 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, nil, ErrClusterNotFound
	}

	out := make([]TaskProtection, 0, len(taskRefs))
	failures := make([]Failure, 0)
	seen := map[string]struct{}{}
	for _, ref := range taskRefs {
		task := s.resolveTaskLocked(clusterRef, ref)
		if task == nil {
			failures = append(failures, Failure{
				Arn:    strings.TrimSpace(ref),
				Reason: "MISSING",
				Detail: "task not found",
			})
			continue
		}
		if _, exists := seen[task.Arn]; exists {
			continue
		}
		seen[task.Arn] = struct{}{}

		tp := TaskProtection{
			TaskArn:           task.Arn,
			ProtectionEnabled: protectionEnabled,
		}
		if protectionEnabled && hasExpires {
			exp := time.Now().UTC().Add(time.Duration(expiresInMinutes) * time.Minute)
			tp.ExpirationDate = &exp
		}
		s.taskProtectionByTask[task.Arn] = tp
		out = append(out, cloneTaskProtection(tp))
	}
	return out, failures, nil
}

func (s *Service) DiscoverPollEndpoint(clusterRef, containerInstanceRef string) (DiscoverPollEndpointResult, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	containerInstanceRef = strings.TrimSpace(containerInstanceRef)

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return DiscoverPollEndpointResult{}, ErrClusterNotFound
	}
	if containerInstanceRef != "" {
		if s.resolveContainerInstanceLocked(clusterRef, containerInstanceRef) == nil {
			return DiscoverPollEndpointResult{}, ErrContainerInstanceNotFound
		}
	}
	return DiscoverPollEndpointResult{
		Endpoint:          fmt.Sprintf("https://ecs-a-1.%s.amazonaws.com", DefaultRegion),
		TelemetryEndpoint: fmt.Sprintf("https://ecs-t-1.%s.amazonaws.com", DefaultRegion),
	}, nil
}

func (s *Service) SubmitAttachmentStateChanges(clusterRef, containerInstanceRef string, attachments []AttachmentStateChange) error {
	clusterRef = strings.TrimSpace(clusterRef)
	containerInstanceRef = strings.TrimSpace(containerInstanceRef)
	if len(attachments) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return ErrClusterNotFound
	}
	if containerInstanceRef != "" && s.resolveContainerInstanceLocked(clusterRef, containerInstanceRef) == nil {
		return ErrContainerInstanceNotFound
	}
	return nil
}

func (s *Service) SubmitContainerStateChange(clusterRef, containerInstanceRef, taskRef, containerName, status, reason string) error {
	clusterRef = strings.TrimSpace(clusterRef)
	containerInstanceRef = strings.TrimSpace(containerInstanceRef)
	taskRef = strings.TrimSpace(taskRef)
	containerName = strings.TrimSpace(containerName)
	status = strings.ToUpper(strings.TrimSpace(status))
	reason = strings.TrimSpace(reason)
	if taskRef == "" || containerName == "" || status == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return ErrClusterNotFound
	}
	if containerInstanceRef != "" && s.resolveContainerInstanceLocked(clusterRef, containerInstanceRef) == nil {
		return ErrContainerInstanceNotFound
	}
	task := s.resolveTaskLocked(clusterRef, taskRef)
	if task == nil {
		return ErrTaskNotFound
	}
	task.LastStatus = status
	if status == "STOPPED" {
		task.DesiredStatus = "STOPPED"
		now := time.Now().UTC()
		task.StoppedAt = &now
		task.StoppedReason = reason
	}
	return nil
}

func (s *Service) SubmitTaskStateChange(clusterRef, taskRef, status, reason string) error {
	clusterRef = strings.TrimSpace(clusterRef)
	taskRef = strings.TrimSpace(taskRef)
	status = strings.ToUpper(strings.TrimSpace(status))
	reason = strings.TrimSpace(reason)
	if taskRef == "" || status == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return ErrClusterNotFound
	}
	task := s.resolveTaskLocked(clusterRef, taskRef)
	if task == nil {
		return ErrTaskNotFound
	}
	task.LastStatus = status
	task.DesiredStatus = status
	if status == "STOPPED" {
		now := time.Now().UTC()
		task.StoppedAt = &now
		task.StoppedReason = reason
	}
	return nil
}

func (s *Service) SubmitTaskStateChangeByAgent(clusterRef, taskRef, status, reason string) error {
	return s.SubmitTaskStateChange(clusterRef, taskRef, status, reason)
}

func (s *Service) SubmitTaskStateChangeByManagedAgents(clusterRef, taskRef, status, reason string) error {
	return s.SubmitTaskStateChange(clusterRef, taskRef, status, reason)
}

func (s *Service) StartTelemetrySession(clusterRef, containerInstanceRef string) error {
	clusterRef = strings.TrimSpace(clusterRef)
	containerInstanceRef = strings.TrimSpace(containerInstanceRef)

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return ErrClusterNotFound
	}
	if containerInstanceRef != "" && s.resolveContainerInstanceLocked(clusterRef, containerInstanceRef) == nil {
		return ErrContainerInstanceNotFound
	}
	return nil
}

func (s *Service) UpdateContainerAgent(clusterRef, containerInstanceRef string) (ContainerInstance, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	containerInstanceRef = strings.TrimSpace(containerInstanceRef)
	if containerInstanceRef == "" {
		return ContainerInstance{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return ContainerInstance{}, ErrClusterNotFound
	}
	ci := s.resolveContainerInstanceLocked(clusterRef, containerInstanceRef)
	if ci == nil {
		return ContainerInstance{}, ErrContainerInstanceNotFound
	}

	now := time.Now().UTC()
	ci.AgentConnected = true
	ci.AgentUpdateStatus = "UPDATED"
	ci.Version++
	if strings.TrimSpace(ci.VersionInfo.AgentVersion) == "" {
		ci.VersionInfo.AgentVersion = "1.0.0"
	}
	if strings.TrimSpace(ci.VersionInfo.DockerVersion) == "" {
		ci.VersionInfo.DockerVersion = "unknown"
	}
	ci.UpdatedAt = now
	return cloneContainerInstance(ci), nil
}

func (s *Service) CreateExpressGatewayService(
	clusterRef string,
	serviceName string,
	executionRoleArn string,
	infrastructureRoleArn string,
	taskRoleArn string,
	cpu string,
	memory string,
	healthCheckPath string,
	primaryContainer map[string]any,
	networkConfiguration map[string]any,
	scalingTarget map[string]any,
	tags map[string]string,
) (ExpressGatewayService, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	serviceName = strings.TrimSpace(serviceName)
	executionRoleArn = strings.TrimSpace(executionRoleArn)
	infrastructureRoleArn = strings.TrimSpace(infrastructureRoleArn)
	taskRoleArn = strings.TrimSpace(taskRoleArn)
	cpu = strings.TrimSpace(cpu)
	memory = strings.TrimSpace(memory)
	healthCheckPath = strings.TrimSpace(healthCheckPath)
	if executionRoleArn == "" || infrastructureRoleArn == "" {
		return ExpressGatewayService{}, ErrInvalidParameter
	}
	if strings.TrimSpace(anyString(primaryContainer["image"])) == "" {
		return ExpressGatewayService{}, ErrInvalidParameter
	}
	normalizedTags, err := normalizeTags(tags)
	if err != nil {
		return ExpressGatewayService{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster := s.resolveClusterLocked(clusterRef)
	if cluster == nil {
		if clusterRef == "" || clusterRef == "default" {
			cluster = s.ensureDefaultClusterLocked()
		} else {
			return ExpressGatewayService{}, ErrClusterNotFound
		}
	}
	clusterName := cluster.ClusterName

	if serviceName == "" {
		s.expressServiceSeq++
		serviceName = fmt.Sprintf("express-gateway-service-%d", s.expressServiceSeq)
	}
	for _, existing := range s.expressServices {
		if existing == nil {
			continue
		}
		if existing.Cluster == cluster.ClusterArn && existing.ServiceName == serviceName {
			return ExpressGatewayService{}, ErrAlreadyExists
		}
	}

	now := time.Now().UTC()
	revisionArn := expressGatewayServiceRevisionARN(clusterName, serviceName, 1)
	cfg := ExpressGatewayServiceConfiguration{
		ServiceRevisionArn:   revisionArn,
		Cpu:                  cpu,
		ExecutionRoleArn:     executionRoleArn,
		HealthCheckPath:      healthCheckPath,
		Memory:               memory,
		TaskRoleArn:          taskRoleArn,
		PrimaryContainer:     cloneAnyMap(primaryContainer),
		NetworkConfiguration: cloneAnyMap(networkConfiguration),
		ScalingTarget:        cloneAnyMap(scalingTarget),
		CreatedAt:            now,
	}
	service := &ExpressGatewayService{
		ServiceArn:            expressGatewayServiceARN(clusterName, serviceName),
		ServiceName:           serviceName,
		Cluster:               cluster.ClusterArn,
		InfrastructureRoleArn: infrastructureRoleArn,
		StatusCode:            "ACTIVE",
		StatusReason:          "",
		CurrentDeployment:     revisionArn,
		ActiveConfigurations:  []ExpressGatewayServiceConfiguration{cfg},
		Tags:                  normalizedTags,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	s.expressServices[service.ServiceArn] = service
	if len(service.Tags) > 0 {
		s.tagsByARN[service.ServiceArn] = cloneStringMap(service.Tags)
	}
	return cloneExpressGatewayService(service), nil
}

func (s *Service) DescribeExpressGatewayService(serviceRef string, includeTags bool) (ExpressGatewayService, error) {
	serviceRef = strings.TrimSpace(serviceRef)
	if serviceRef == "" {
		return ExpressGatewayService{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	service := s.resolveExpressGatewayServiceLocked(serviceRef)
	if service == nil {
		return ExpressGatewayService{}, ErrServiceNotFound
	}
	out := cloneExpressGatewayService(service)
	if !includeTags {
		out.Tags = map[string]string{}
	}
	return out, nil
}

func (s *Service) UpdateExpressGatewayService(
	serviceRef string,
	executionRoleArn string,
	taskRoleArn string,
	cpu string,
	memory string,
	healthCheckPath string,
	primaryContainer map[string]any,
	networkConfiguration map[string]any,
	scalingTarget map[string]any,
) (UpdatedExpressGatewayService, error) {
	serviceRef = strings.TrimSpace(serviceRef)
	executionRoleArn = strings.TrimSpace(executionRoleArn)
	taskRoleArn = strings.TrimSpace(taskRoleArn)
	cpu = strings.TrimSpace(cpu)
	memory = strings.TrimSpace(memory)
	healthCheckPath = strings.TrimSpace(healthCheckPath)
	if serviceRef == "" {
		return UpdatedExpressGatewayService{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	service := s.resolveExpressGatewayServiceLocked(serviceRef)
	if service == nil {
		return UpdatedExpressGatewayService{}, ErrServiceNotFound
	}

	var latest ExpressGatewayServiceConfiguration
	if n := len(service.ActiveConfigurations); n > 0 {
		latest = cloneExpressGatewayServiceConfiguration(service.ActiveConfigurations[n-1])
	}
	if executionRoleArn != "" {
		latest.ExecutionRoleArn = executionRoleArn
	}
	if taskRoleArn != "" {
		latest.TaskRoleArn = taskRoleArn
	}
	if cpu != "" {
		latest.Cpu = cpu
	}
	if memory != "" {
		latest.Memory = memory
	}
	if healthCheckPath != "" {
		latest.HealthCheckPath = healthCheckPath
	}
	if primaryContainer != nil {
		if image := strings.TrimSpace(anyString(primaryContainer["image"])); image == "" {
			return UpdatedExpressGatewayService{}, ErrInvalidParameter
		}
		latest.PrimaryContainer = cloneAnyMap(primaryContainer)
	}
	if networkConfiguration != nil {
		latest.NetworkConfiguration = cloneAnyMap(networkConfiguration)
	}
	if scalingTarget != nil {
		latest.ScalingTarget = cloneAnyMap(scalingTarget)
	}

	now := time.Now().UTC()
	nextRevision := len(service.ActiveConfigurations) + 1
	clusterName, parsedServiceName, ok := parseExpressGatewayServiceARN(service.ServiceArn)
	if !ok {
		clusterName = "default"
		parsedServiceName = service.ServiceName
	}
	latest.ServiceRevisionArn = expressGatewayServiceRevisionARN(clusterName, parsedServiceName, nextRevision)
	latest.CreatedAt = now
	service.ActiveConfigurations = append(service.ActiveConfigurations, latest)
	service.CurrentDeployment = latest.ServiceRevisionArn
	service.UpdatedAt = now
	service.StatusCode = "ACTIVE"
	service.StatusReason = ""

	return UpdatedExpressGatewayService{
		ServiceArn:          service.ServiceArn,
		ServiceName:         service.ServiceName,
		Cluster:             service.Cluster,
		StatusCode:          service.StatusCode,
		StatusReason:        service.StatusReason,
		TargetConfiguration: cloneExpressGatewayServiceConfiguration(latest),
		CreatedAt:           service.CreatedAt,
		UpdatedAt:           service.UpdatedAt,
	}, nil
}

func (s *Service) DeleteExpressGatewayService(serviceRef string) (ExpressGatewayService, error) {
	serviceRef = strings.TrimSpace(serviceRef)
	if serviceRef == "" {
		return ExpressGatewayService{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	service := s.resolveExpressGatewayServiceLocked(serviceRef)
	if service == nil {
		return ExpressGatewayService{}, ErrServiceNotFound
	}
	now := time.Now().UTC()
	out := cloneExpressGatewayService(service)
	out.StatusCode = "INACTIVE"
	out.StatusReason = "deleted"
	out.UpdatedAt = now

	delete(s.tagsByARN, service.ServiceArn)
	delete(s.expressServices, service.ServiceArn)
	return out, nil
}

func (s *Service) RegisterContainerInstance(clusterRef, containerInstanceArn, ec2InstanceID string, attributes []Attribute) (ContainerInstance, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	containerInstanceArn = strings.TrimSpace(containerInstanceArn)
	ec2InstanceID = strings.TrimSpace(ec2InstanceID)
	if clusterRef == "" {
		return ContainerInstance{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster := s.resolveClusterLocked(clusterRef)
	if cluster == nil {
		return ContainerInstance{}, ErrClusterNotFound
	}

	if containerInstanceArn == "" {
		s.containerInstSeq++
		containerInstanceArn = containerInstanceARN(cluster.ClusterName, s.containerInstSeq)
	}
	if _, exists := s.containerInstances[containerInstanceArn]; exists {
		return ContainerInstance{}, ErrAlreadyExists
	}
	if ec2InstanceID == "" {
		ec2InstanceID = fmt.Sprintf("i-%08x", s.containerInstSeq+1)
	}

	now := time.Now().UTC()
	ci := &ContainerInstance{
		Arn:               containerInstanceArn,
		ClusterArn:        cluster.ClusterArn,
		Ec2InstanceID:     ec2InstanceID,
		Status:            "ACTIVE",
		AgentConnected:    true,
		AgentUpdateStatus: "",
		Version:           1,
		VersionInfo: VersionInfo{
			AgentVersion:  "1.0.0",
			DockerVersion: "unknown",
		},
		RegisteredAt: now,
		UpdatedAt:    now,
		Attributes:   []Attribute{},
	}
	s.containerInstances[ci.Arn] = ci
	cluster.RegisteredContainerInstancesCount++
	cluster.UpdatedAt = now

	if len(attributes) > 0 {
		for _, attr := range attributes {
			attr.Name = strings.TrimSpace(attr.Name)
			if attr.Name == "" {
				return ContainerInstance{}, ErrInvalidParameter
			}
			if strings.TrimSpace(attr.TargetType) == "" {
				attr.TargetType = "container-instance"
			}
			if strings.TrimSpace(attr.TargetID) == "" {
				attr.TargetID = ci.Arn
			}
			if attr.TargetType == "container-instance" {
				attr.TargetID = ci.Arn
			}
			s.attributesByKey[attributeKey(attr.TargetType, attr.TargetID, attr.Name)] = Attribute{
				Name:       attr.Name,
				Value:      attr.Value,
				TargetType: attr.TargetType,
				TargetID:   attr.TargetID,
			}
		}
		ci.Attributes = s.collectAttributesLocked("container-instance", ci.Arn)
	}
	return cloneContainerInstance(ci), nil
}

func (s *Service) DeregisterContainerInstance(clusterRef, containerInstanceRef string) (ContainerInstance, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	containerInstanceRef = strings.TrimSpace(containerInstanceRef)
	if containerInstanceRef == "" {
		return ContainerInstance{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return ContainerInstance{}, ErrClusterNotFound
	}
	ci := s.resolveContainerInstanceLocked(clusterRef, containerInstanceRef)
	if ci == nil {
		return ContainerInstance{}, ErrContainerInstanceNotFound
	}

	now := time.Now().UTC()
	out := cloneContainerInstance(ci)
	out.Status = "DEREGISTERED"
	out.UpdatedAt = now

	for key, attr := range s.attributesByKey {
		if attr.TargetType == "container-instance" && attr.TargetID == ci.Arn {
			delete(s.attributesByKey, key)
		}
	}

	for _, task := range s.tasks {
		if task.ContainerInstanceArn == ci.Arn {
			task.ContainerInstanceArn = ""
		}
	}

	delete(s.containerInstances, ci.Arn)
	if cluster := s.resolveClusterLocked(ci.ClusterArn); cluster != nil {
		if cluster.RegisteredContainerInstancesCount > 0 {
			cluster.RegisteredContainerInstancesCount--
		}
		cluster.UpdatedAt = now
	}
	return out, nil
}

func (s *Service) DescribeContainerInstances(clusterRef string, refs []string) ([]ContainerInstance, []Failure, error) {
	clusterRef = strings.TrimSpace(clusterRef)

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, nil, ErrClusterNotFound
	}

	if len(refs) == 0 {
		out := make([]ContainerInstance, 0, len(s.containerInstances))
		for _, ci := range s.containerInstances {
			if clusterRef != "" && !containerInstanceInCluster(ci, clusterRef, s) {
				continue
			}
			out = append(out, cloneContainerInstance(ci))
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Arn < out[j].Arn })
		return out, nil, nil
	}

	seen := map[string]struct{}{}
	out := make([]ContainerInstance, 0, len(refs))
	failures := make([]Failure, 0)
	for _, ref := range refs {
		ci := s.resolveContainerInstanceLocked(clusterRef, ref)
		if ci == nil {
			failures = append(failures, Failure{
				Arn:    strings.TrimSpace(ref),
				Reason: "MISSING",
				Detail: "container instance not found",
			})
			continue
		}
		if _, exists := seen[ci.Arn]; exists {
			continue
		}
		seen[ci.Arn] = struct{}{}
		out = append(out, cloneContainerInstance(ci))
	}
	return out, failures, nil
}

func (s *Service) ListContainerInstances(clusterRef, status, nextToken string, maxResults int32) ([]string, string, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	status = strings.ToUpper(strings.TrimSpace(status))

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, "", ErrClusterNotFound
	}

	out := make([]string, 0, len(s.containerInstances))
	for _, ci := range s.containerInstances {
		if clusterRef != "" && !containerInstanceInCluster(ci, clusterRef, s) {
			continue
		}
		if status != "" && ci.Status != status {
			continue
		}
		out = append(out, ci.Arn)
	}
	sort.Strings(out)
	return paginateStringList(out, nextToken, maxResults)
}

func (s *Service) UpdateContainerInstancesState(clusterRef string, refs []string, status string) ([]ContainerInstance, []Failure, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "" {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, nil, ErrClusterNotFound
	}

	out := make([]ContainerInstance, 0, len(refs))
	failures := make([]Failure, 0)
	for _, ref := range refs {
		ci := s.resolveContainerInstanceLocked(clusterRef, ref)
		if ci == nil {
			failures = append(failures, Failure{
				Arn:    strings.TrimSpace(ref),
				Reason: "MISSING",
				Detail: "container instance not found",
			})
			continue
		}
		ci.Status = status
		ci.UpdatedAt = time.Now().UTC()
		out = append(out, cloneContainerInstance(ci))
	}
	return out, failures, nil
}

func (s *Service) PutAttributes(clusterRef string, attrs []Attribute) ([]Attribute, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	if len(attrs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, ErrClusterNotFound
	}

	out := make([]Attribute, 0, len(attrs))
	for _, attr := range attrs {
		name := strings.TrimSpace(attr.Name)
		if name == "" {
			return nil, ErrInvalidParameter
		}
		targetType := strings.TrimSpace(attr.TargetType)
		if targetType == "" {
			targetType = "container-instance"
		}
		targetID := strings.TrimSpace(attr.TargetID)
		if targetID == "" {
			return nil, ErrInvalidParameter
		}
		if targetType == "container-instance" {
			ci := s.resolveContainerInstanceLocked(clusterRef, targetID)
			if ci == nil {
				return nil, ErrContainerInstanceNotFound
			}
			targetID = ci.Arn
		}
		outAttr := Attribute{
			Name:       name,
			Value:      attr.Value,
			TargetType: targetType,
			TargetID:   targetID,
		}
		s.attributesByKey[attributeKey(targetType, targetID, name)] = outAttr
		out = append(out, outAttr)
	}

	for _, ci := range s.containerInstances {
		ci.Attributes = s.collectAttributesLocked("container-instance", ci.Arn)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TargetType == out[j].TargetType {
			if out[i].TargetID == out[j].TargetID {
				return out[i].Name < out[j].Name
			}
			return out[i].TargetID < out[j].TargetID
		}
		return out[i].TargetType < out[j].TargetType
	})
	return out, nil
}

func (s *Service) TagResource(resourceARN string, tags map[string]string) error {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return ErrInvalidParameter
	}
	normalizedTags, err := normalizeTags(tags)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.resourceExistsLocked(resourceARN) {
		return ErrResourceNotFound
	}
	existing := cloneStringMap(s.tagsByARN[resourceARN])
	for key, value := range normalizedTags {
		existing[key] = value
	}
	s.tagsByARN[resourceARN] = existing

	if service := s.services[resourceARN]; service != nil {
		service.Tags = cloneStringMap(existing)
		service.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (s *Service) UntagResource(resourceARN string, tagKeys []string) error {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.resourceExistsLocked(resourceARN) {
		return ErrResourceNotFound
	}
	existing := cloneStringMap(s.tagsByARN[resourceARN])
	for _, raw := range tagKeys {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}
		delete(existing, key)
	}
	s.tagsByARN[resourceARN] = existing
	if len(existing) == 0 {
		delete(s.tagsByARN, resourceARN)
	}

	if service := s.services[resourceARN]; service != nil {
		service.Tags = cloneStringMap(existing)
		service.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (s *Service) ListTagsForResource(resourceARN string) (map[string]string, error) {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.resourceExistsLocked(resourceARN) {
		return nil, ErrResourceNotFound
	}
	return cloneStringMap(s.tagsByARN[resourceARN]), nil
}

func (s *Service) ListAttributes(clusterRef, targetType, targetID, attributeName, nextToken string, maxResults int32) ([]Attribute, string, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	targetType = strings.TrimSpace(targetType)
	targetID = strings.TrimSpace(targetID)
	attributeName = strings.TrimSpace(attributeName)

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, "", ErrClusterNotFound
	}
	if targetType == "container-instance" && targetID != "" {
		ci := s.resolveContainerInstanceLocked(clusterRef, targetID)
		if ci != nil {
			targetID = ci.Arn
		}
	}

	out := make([]Attribute, 0, len(s.attributesByKey))
	for _, attr := range s.attributesByKey {
		if targetType != "" && attr.TargetType != targetType {
			continue
		}
		if targetID != "" && attr.TargetID != targetID {
			continue
		}
		if attributeName != "" && attr.Name != attributeName {
			continue
		}
		out = append(out, cloneAttribute(attr))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TargetType == out[j].TargetType {
			if out[i].TargetID == out[j].TargetID {
				return out[i].Name < out[j].Name
			}
			return out[i].TargetID < out[j].TargetID
		}
		return out[i].TargetType < out[j].TargetType
	})
	return paginateAttributes(out, nextToken, maxResults)
}

func (s *Service) DeleteAttributes(clusterRef string, attrs []Attribute) ([]Attribute, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	if len(attrs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, ErrClusterNotFound
	}

	deleted := make([]Attribute, 0, len(attrs))
	for _, attr := range attrs {
		name := strings.TrimSpace(attr.Name)
		targetType := strings.TrimSpace(attr.TargetType)
		targetID := strings.TrimSpace(attr.TargetID)
		if name == "" || targetType == "" || targetID == "" {
			return nil, ErrInvalidParameter
		}
		if targetType == "container-instance" {
			ci := s.resolveContainerInstanceLocked(clusterRef, targetID)
			if ci != nil {
				targetID = ci.Arn
			}
		}
		key := attributeKey(targetType, targetID, name)
		existing, exists := s.attributesByKey[key]
		if !exists {
			continue
		}
		delete(s.attributesByKey, key)
		deleted = append(deleted, cloneAttribute(existing))
	}
	for _, ci := range s.containerInstances {
		ci.Attributes = s.collectAttributesLocked("container-instance", ci.Arn)
	}
	sort.Slice(deleted, func(i, j int) bool {
		if deleted[i].TargetType == deleted[j].TargetType {
			if deleted[i].TargetID == deleted[j].TargetID {
				return deleted[i].Name < deleted[j].Name
			}
			return deleted[i].TargetID < deleted[j].TargetID
		}
		return deleted[i].TargetType < deleted[j].TargetType
	})
	return deleted, nil
}

func (s *Service) ListServiceDeployments(clusterRef, serviceRef, nextToken string, maxResults int32) ([]ServiceDeploymentSnapshot, string, error) {
	return s.listServiceDeployments(clusterRef, serviceRef, "DESC", nextToken, maxResults)
}

func (s *Service) DescribeServiceDeployments(clusterRef, serviceRef string, refs []string) ([]ServiceDeploymentSnapshot, []Failure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	service, err := s.resolveServiceForStage5Locked(clusterRef, serviceRef)
	if err != nil {
		return nil, nil, err
	}

	if len(refs) == 0 {
		out := cloneServiceDeploymentSnapshots(s.serviceDeployments[service.Arn])
		sort.Slice(out, func(i, j int) bool {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		})
		return out, nil, nil
	}

	seen := map[string]struct{}{}
	out := make([]ServiceDeploymentSnapshot, 0, len(refs))
	failures := make([]Failure, 0)
	for _, ref := range refs {
		deployment := s.resolveServiceDeploymentLocked(service.Arn, ref)
		if deployment == nil {
			failures = append(failures, Failure{
				Arn:    strings.TrimSpace(ref),
				Reason: "MISSING",
				Detail: "service deployment not found",
			})
			continue
		}
		if _, exists := seen[deployment.Arn]; exists {
			continue
		}
		seen[deployment.Arn] = struct{}{}
		out = append(out, cloneServiceDeploymentSnapshot(*deployment))
	}
	return out, failures, nil
}

func (s *Service) StopServiceDeployment(clusterRef, serviceRef, deploymentRef, stopReason string) (ServiceDeploymentSnapshot, error) {
	deploymentRef = strings.TrimSpace(deploymentRef)
	stopReason = strings.TrimSpace(stopReason)
	if deploymentRef == "" {
		return ServiceDeploymentSnapshot{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	service, err := s.resolveServiceForStage5Locked(clusterRef, serviceRef)
	if err != nil {
		return ServiceDeploymentSnapshot{}, err
	}
	deployment := s.resolveServiceDeploymentLocked(service.Arn, deploymentRef)
	if deployment == nil {
		return ServiceDeploymentSnapshot{}, ErrServiceDeploymentNotFound
	}

	now := time.Now().UTC()
	deployment.Status = "STOPPED"
	if stopReason == "" {
		stopReason = "stopped by request"
	}
	deployment.StatusReason = stopReason
	deployment.UpdatedAt = now
	deployment.StoppedAt = &now
	return cloneServiceDeploymentSnapshot(*deployment), nil
}

func (s *Service) ListServiceDeploymentsByCreatedAt(clusterRef, serviceRef, order, nextToken string, maxResults int32) ([]ServiceDeploymentSnapshot, string, error) {
	return s.listServiceDeployments(clusterRef, serviceRef, order, nextToken, maxResults)
}

func (s *Service) ListServiceDeploymentsByServiceRevision(clusterRef, serviceRef, serviceRevisionRef, nextToken string, maxResults int32) ([]ServiceDeploymentSnapshot, string, error) {
	serviceRevisionRef = strings.TrimSpace(serviceRevisionRef)
	if serviceRevisionRef == "" {
		return nil, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	service, err := s.resolveServiceForStage5Locked(clusterRef, serviceRef)
	if err != nil {
		return nil, "", err
	}
	revision := s.resolveServiceRevisionLocked(service.Arn, serviceRevisionRef)
	if revision == nil {
		return nil, "", ErrServiceRevisionNotFound
	}

	filtered := make([]ServiceDeploymentSnapshot, 0, len(s.serviceDeployments[service.Arn]))
	for _, deployment := range s.serviceDeployments[service.Arn] {
		if deployment.ServiceRevisionArn == revision.Arn {
			filtered = append(filtered, cloneServiceDeploymentSnapshot(deployment))
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})
	return paginateServiceDeployments(filtered, nextToken, maxResults)
}

func (s *Service) DescribeServiceRevisions(clusterRef, serviceRef string, refs []string) ([]ServiceRevisionSnapshot, []Failure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	service, err := s.resolveServiceForStage5Locked(clusterRef, serviceRef)
	if err != nil {
		return nil, nil, err
	}

	if len(refs) == 0 {
		out := cloneServiceRevisionSnapshots(s.serviceRevisions[service.Arn])
		sort.Slice(out, func(i, j int) bool {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		})
		return out, nil, nil
	}

	seen := map[string]struct{}{}
	out := make([]ServiceRevisionSnapshot, 0, len(refs))
	failures := make([]Failure, 0)
	for _, ref := range refs {
		revision := s.resolveServiceRevisionLocked(service.Arn, ref)
		if revision == nil {
			failures = append(failures, Failure{
				Arn:    strings.TrimSpace(ref),
				Reason: "MISSING",
				Detail: "service revision not found",
			})
			continue
		}
		if _, exists := seen[revision.Arn]; exists {
			continue
		}
		seen[revision.Arn] = struct{}{}
		out = append(out, cloneServiceRevisionSnapshot(*revision))
	}
	return out, failures, nil
}

func (s *Service) listServiceDeployments(clusterRef, serviceRef, order, nextToken string, maxResults int32) ([]ServiceDeploymentSnapshot, string, error) {
	order = strings.ToUpper(strings.TrimSpace(order))
	if order == "" {
		order = "DESC"
	}
	if order != "ASC" && order != "DESC" {
		return nil, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	service, err := s.resolveServiceForStage5Locked(clusterRef, serviceRef)
	if err != nil {
		return nil, "", err
	}
	out := cloneServiceDeploymentSnapshots(s.serviceDeployments[service.Arn])
	sort.Slice(out, func(i, j int) bool {
		if order == "ASC" {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return paginateServiceDeployments(out, nextToken, maxResults)
}

func (s *Service) resolveClusterLocked(ref string) *Cluster {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil
	}
	if cluster := s.clustersByARN[ref]; cluster != nil {
		return cluster
	}
	return s.clustersByName[ref]
}

func (s *Service) resolveCapacityProviderLocked(ref string) *CapacityProvider {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil
	}
	if cp := s.capacityProviders[ref]; cp != nil {
		return cp
	}
	for _, cp := range s.capacityProviders {
		if cp != nil && cp.Arn == ref {
			return cp
		}
	}
	return nil
}

func (s *Service) resolveTaskDefinitionLocked(ref string) *TaskDefinition {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil
	}
	if def := s.taskDefinitionsByARN[ref]; def != nil {
		return def
	}
	family, revision, hasRevision := parseTaskDefinitionRef(ref)
	if !hasRevision || family == "" {
		return nil
	}
	for _, def := range s.taskDefinitions[family] {
		if def != nil && def.Revision == revision {
			return def
		}
	}
	return nil
}

func (s *Service) resolveServiceLocked(clusterRef, serviceRef string) *ServiceDefinition {
	serviceRef = strings.TrimSpace(serviceRef)
	if serviceRef == "" {
		return nil
	}

	clusterRef = strings.TrimSpace(clusterRef)
	var cluster *Cluster
	if clusterRef != "" {
		cluster = s.resolveClusterLocked(clusterRef)
		if cluster == nil {
			return nil
		}
	}

	if service := s.services[serviceRef]; service != nil {
		if cluster != nil && service.ClusterArn != cluster.ClusterArn {
			return nil
		}
		return service
	}

	serviceNameRef := serviceRef
	if strings.HasPrefix(serviceNameRef, "arn:") {
		_, parsedName, ok := parseServiceARN(serviceNameRef)
		if ok {
			serviceNameRef = parsedName
		}
	}
	if idx := strings.LastIndex(serviceNameRef, "/"); idx >= 0 && idx+1 < len(serviceNameRef) {
		serviceNameRef = serviceNameRef[idx+1:]
	}

	for _, service := range s.services {
		if cluster != nil && service.ClusterArn != cluster.ClusterArn {
			continue
		}
		if service.Arn == serviceRef || service.Name == serviceNameRef {
			return service
		}
	}
	return nil
}

func (s *Service) resolveServiceForStage5Locked(clusterRef, serviceRef string) (*ServiceDefinition, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, ErrClusterNotFound
	}
	service := s.resolveServiceLocked(clusterRef, serviceRef)
	if service == nil {
		return nil, ErrServiceNotFound
	}
	return service, nil
}

func (s *Service) resolveServiceForTaskSetOpsLocked(clusterRef, serviceRef string) (*ServiceDefinition, error) {
	clusterRef = strings.TrimSpace(clusterRef)
	if clusterRef != "" && s.resolveClusterLocked(clusterRef) == nil {
		return nil, ErrClusterNotFound
	}
	service := s.resolveServiceLocked(clusterRef, serviceRef)
	if service == nil {
		return nil, ErrServiceNotFound
	}
	return service, nil
}

func (s *Service) resolveServiceDeploymentLocked(serviceARNRef, deploymentRef string) *ServiceDeploymentSnapshot {
	deploymentRef = strings.TrimSpace(deploymentRef)
	if deploymentRef == "" {
		return nil
	}

	deployments := s.serviceDeployments[serviceARNRef]
	for idx := range deployments {
		if deployments[idx].Arn == deploymentRef {
			return &deployments[idx]
		}
		if strings.HasSuffix(deployments[idx].Arn, "/"+deploymentRef) {
			return &deployments[idx]
		}
	}
	return nil
}

func (s *Service) resolveServiceRevisionLocked(serviceARNRef, revisionRef string) *ServiceRevisionSnapshot {
	revisionRef = strings.TrimSpace(revisionRef)
	if revisionRef == "" {
		return nil
	}

	revisions := s.serviceRevisions[serviceARNRef]
	for idx := range revisions {
		if revisions[idx].Arn == revisionRef {
			return &revisions[idx]
		}
		if strings.HasSuffix(revisions[idx].Arn, "/"+revisionRef) {
			return &revisions[idx]
		}
	}
	return nil
}

func (s *Service) resolveTaskSetLocked(serviceARNRef, taskSetRef string) *TaskSet {
	taskSetRef = strings.TrimSpace(taskSetRef)
	if taskSetRef == "" {
		return nil
	}

	if taskSet := s.taskSets[taskSetRef]; taskSet != nil {
		if serviceARNRef == "" || taskSet.ServiceArn == serviceARNRef {
			return taskSet
		}
		return nil
	}

	if strings.HasPrefix(taskSetRef, "arn:") {
		if _, _, id, ok := parseTaskSetARN(taskSetRef); ok {
			taskSetRef = id
		}
	}
	for _, taskSet := range s.taskSets {
		if serviceARNRef != "" && taskSet.ServiceArn != serviceARNRef {
			continue
		}
		if taskSet.ID == taskSetRef || strings.HasSuffix(taskSet.Arn, "/"+taskSetRef) {
			return taskSet
		}
	}
	return nil
}

func (s *Service) resolveTaskLocked(clusterRef, taskRef string) *Task {
	taskRef = strings.TrimSpace(taskRef)
	if taskRef == "" {
		return nil
	}

	clusterRef = strings.TrimSpace(clusterRef)
	var cluster *Cluster
	if clusterRef != "" {
		cluster = s.resolveClusterLocked(clusterRef)
		if cluster == nil {
			return nil
		}
	}

	if task := s.tasks[taskRef]; task != nil {
		if cluster != nil && task.ClusterArn != cluster.ClusterArn {
			return nil
		}
		return task
	}
	ref := taskRef
	if strings.HasPrefix(ref, "arn:") {
		if _, id, ok := parseTaskARN(ref); ok {
			ref = id
		}
	}
	for _, task := range s.tasks {
		if cluster != nil && task.ClusterArn != cluster.ClusterArn {
			continue
		}
		if strings.HasSuffix(task.Arn, "/"+ref) {
			return task
		}
	}
	return nil
}

func (s *Service) resolveContainerInstanceLocked(clusterRef, containerInstanceRef string) *ContainerInstance {
	containerInstanceRef = strings.TrimSpace(containerInstanceRef)
	if containerInstanceRef == "" {
		return nil
	}

	clusterRef = strings.TrimSpace(clusterRef)
	var cluster *Cluster
	if clusterRef != "" {
		cluster = s.resolveClusterLocked(clusterRef)
		if cluster == nil {
			return nil
		}
	}

	if ci := s.containerInstances[containerInstanceRef]; ci != nil {
		if cluster != nil && ci.ClusterArn != cluster.ClusterArn {
			return nil
		}
		return ci
	}

	ref := containerInstanceRef
	if strings.HasPrefix(ref, "arn:") {
		if _, id, ok := parseContainerInstanceARN(ref); ok {
			ref = id
		}
	}
	for _, ci := range s.containerInstances {
		if cluster != nil && ci.ClusterArn != cluster.ClusterArn {
			continue
		}
		if strings.HasSuffix(ci.Arn, "/"+ref) {
			return ci
		}
	}
	return nil
}

func (s *Service) resolveExpressGatewayServiceLocked(serviceRef string) *ExpressGatewayService {
	serviceRef = strings.TrimSpace(serviceRef)
	if serviceRef == "" {
		return nil
	}
	if service := s.expressServices[serviceRef]; service != nil {
		return service
	}

	serviceNameRef := serviceRef
	if strings.HasPrefix(serviceNameRef, "arn:") {
		_, parsedName, ok := parseExpressGatewayServiceARN(serviceNameRef)
		if ok {
			serviceNameRef = parsedName
		}
	}
	if idx := strings.LastIndex(serviceNameRef, "/"); idx >= 0 && idx+1 < len(serviceNameRef) {
		serviceNameRef = serviceNameRef[idx+1:]
	}
	for _, service := range s.expressServices {
		if service.ServiceArn == serviceRef || service.ServiceName == serviceNameRef {
			return service
		}
	}
	return nil
}

func (s *Service) ensureDefaultClusterLocked() *Cluster {
	if cluster := s.clustersByName["default"]; cluster != nil {
		return cluster
	}
	now := time.Now().UTC()
	cluster := &Cluster{
		ClusterArn:                        clusterARN("default"),
		ClusterName:                       "default",
		Status:                            "ACTIVE",
		Settings:                          []ClusterSetting{},
		Statistics:                        map[string]string{},
		Configuration:                     map[string]any{},
		ServiceConnectDefaultsNamespace:   "",
		Tags:                              map[string]string{},
		RegisteredContainerInstancesCount: 0,
		RunningTasksCount:                 0,
		PendingTasksCount:                 0,
		ActiveServicesCount:               0,
		CapacityProviders:                 []string{},
		DefaultCapacityProviderStrategy:   []CapacityProviderStrategyItem{},
		CreatedAt:                         now,
		UpdatedAt:                         now,
	}
	s.clustersByName[cluster.ClusterName] = cluster
	s.clustersByARN[cluster.ClusterArn] = cluster
	return cluster
}

func (s *Service) resourceExistsLocked(resourceARN string) bool {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return false
	}
	if _, ok := s.clustersByARN[resourceARN]; ok {
		return true
	}
	if _, ok := s.services[resourceARN]; ok {
		return true
	}
	if _, ok := s.expressServices[resourceARN]; ok {
		return true
	}
	if _, ok := s.taskDefinitionsByARN[resourceARN]; ok {
		return true
	}
	if _, ok := s.capacityProviders[resourceARN]; ok {
		return true
	}
	for _, cp := range s.capacityProviders {
		if cp != nil && cp.Arn == resourceARN {
			return true
		}
	}
	if _, ok := s.tasks[resourceARN]; ok {
		return true
	}
	if _, ok := s.taskSets[resourceARN]; ok {
		return true
	}
	if _, ok := s.containerInstances[resourceARN]; ok {
		return true
	}
	return false
}

func (s *Service) collectAttributesLocked(targetType, targetID string) []Attribute {
	targetType = strings.TrimSpace(targetType)
	targetID = strings.TrimSpace(targetID)
	out := make([]Attribute, 0)
	for _, attr := range s.attributesByKey {
		if attr.TargetType == targetType && attr.TargetID == targetID {
			out = append(out, cloneAttribute(attr))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) taskProtectionForTaskLocked(taskARN string) TaskProtection {
	if tp, exists := s.taskProtectionByTask[taskARN]; exists {
		return cloneTaskProtection(tp)
	}
	return TaskProtection{
		TaskArn:           taskARN,
		ProtectionEnabled: false,
	}
}

func (s *Service) createServiceRevisionLocked(service *ServiceDefinition, now time.Time) ServiceRevisionSnapshot {
	sequence := len(s.serviceRevisions[service.Arn]) + 1
	clusterName, _, _ := parseServiceARN(service.Arn)
	revision := ServiceRevisionSnapshot{
		Arn:               serviceRevisionARN(clusterName, service.Name, sequence),
		ServiceArn:        service.Arn,
		TaskDefinitionArn: service.TaskDefinitionArn,
		DesiredCount:      service.DesiredCount,
		CreatedAt:         now,
	}
	s.serviceRevisions[service.Arn] = append(s.serviceRevisions[service.Arn], revision)
	return revision
}

func (s *Service) createServiceDeploymentLocked(service *ServiceDefinition, serviceRevisionARNRef, status, statusReason string, now time.Time) ServiceDeploymentSnapshot {
	sequence := len(s.serviceDeployments[service.Arn]) + 1
	clusterName, _, _ := parseServiceARN(service.Arn)
	deployment := ServiceDeploymentSnapshot{
		Arn:                serviceDeploymentARN(clusterName, service.Name, sequence),
		ServiceArn:         service.Arn,
		ServiceRevisionArn: serviceRevisionARNRef,
		Status:             status,
		StatusReason:       statusReason,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	s.serviceDeployments[service.Arn] = append(s.serviceDeployments[service.Arn], deployment)
	return deployment
}

func serviceInCluster(service *ServiceDefinition, clusterRef string, s *Service) bool {
	clusterRef = strings.TrimSpace(clusterRef)
	if clusterRef == "" {
		return true
	}
	cluster := s.resolveClusterLocked(clusterRef)
	return cluster != nil && service.ClusterArn == cluster.ClusterArn
}

func taskInCluster(task *Task, clusterRef string, s *Service) bool {
	clusterRef = strings.TrimSpace(clusterRef)
	if clusterRef == "" {
		return true
	}
	cluster := s.resolveClusterLocked(clusterRef)
	return cluster != nil && task.ClusterArn == cluster.ClusterArn
}

func containerInstanceInCluster(containerInstance *ContainerInstance, clusterRef string, s *Service) bool {
	clusterRef = strings.TrimSpace(clusterRef)
	if clusterRef == "" {
		return true
	}
	cluster := s.resolveClusterLocked(clusterRef)
	return cluster != nil && containerInstance.ClusterArn == cluster.ClusterArn
}

func paginateServiceDeployments(all []ServiceDeploymentSnapshot, nextToken string, maxResults int32) ([]ServiceDeploymentSnapshot, string, error) {
	offset := 0
	nextToken = strings.TrimSpace(nextToken)
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 0 || parsed > len(all) {
			return nil, "", ErrInvalidParameter
		}
		offset = parsed
	}

	limit := int(maxResults)
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	out := append([]ServiceDeploymentSnapshot(nil), all[offset:end]...)

	newNextToken := ""
	if end < len(all) {
		newNextToken = strconv.Itoa(end)
	}
	return out, newNextToken, nil
}

func paginateStringList(all []string, nextToken string, maxResults int32) ([]string, string, error) {
	offset := 0
	nextToken = strings.TrimSpace(nextToken)
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 0 || parsed > len(all) {
			return nil, "", ErrInvalidParameter
		}
		offset = parsed
	}

	limit := int(maxResults)
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	out := append([]string(nil), all[offset:end]...)

	newNextToken := ""
	if end < len(all) {
		newNextToken = strconv.Itoa(end)
	}
	return out, newNextToken, nil
}

func paginateAttributes(all []Attribute, nextToken string, maxResults int32) ([]Attribute, string, error) {
	offset := 0
	nextToken = strings.TrimSpace(nextToken)
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 0 || parsed > len(all) {
			return nil, "", ErrInvalidParameter
		}
		offset = parsed
	}

	limit := int(maxResults)
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	out := append([]Attribute(nil), all[offset:end]...)

	newNextToken := ""
	if end < len(all) {
		newNextToken = strconv.Itoa(end)
	}
	return out, newNextToken, nil
}

func (s *Service) listTaskSetsForServiceLocked(serviceARNRef string) []TaskSet {
	out := make([]TaskSet, 0)
	for _, taskSet := range s.taskSets {
		if taskSet.ServiceArn == serviceARNRef {
			out = append(out, cloneTaskSet(taskSet))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

func normalizeClusterSettings(settings []ClusterSetting) ([]ClusterSetting, error) {
	merged := map[string]string{}
	for _, setting := range settings {
		name := strings.TrimSpace(setting.Name)
		value := strings.TrimSpace(setting.Value)
		if name == "" || value == "" {
			return nil, ErrInvalidParameter
		}
		merged[name] = value
	}
	out := make([]ClusterSetting, 0, len(merged))
	for name, value := range merged {
		out = append(out, ClusterSetting{Name: name, Value: value})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func normalizeTags(tags map[string]string) (map[string]string, error) {
	if len(tags) == 0 {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(tags))
	for rawKey, rawValue := range tags {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			return nil, ErrInvalidParameter
		}
		out[key] = rawValue
	}
	return out, nil
}

func cloneCluster(cluster *Cluster) Cluster {
	if cluster == nil {
		return Cluster{}
	}
	settings := make([]ClusterSetting, 0, len(cluster.Settings))
	for _, setting := range cluster.Settings {
		settings = append(settings, ClusterSetting{
			Name:  setting.Name,
			Value: setting.Value,
		})
	}
	strategy := make([]CapacityProviderStrategyItem, 0, len(cluster.DefaultCapacityProviderStrategy))
	for _, item := range cluster.DefaultCapacityProviderStrategy {
		strategy = append(strategy, CapacityProviderStrategyItem{
			CapacityProvider: item.CapacityProvider,
			Base:             item.Base,
			Weight:           item.Weight,
		})
	}
	return Cluster{
		ClusterArn:                        cluster.ClusterArn,
		ClusterName:                       cluster.ClusterName,
		Status:                            cluster.Status,
		Settings:                          settings,
		Statistics:                        cloneStringMap(cluster.Statistics),
		Configuration:                     cloneAnyMap(cluster.Configuration),
		ServiceConnectDefaultsNamespace:   cluster.ServiceConnectDefaultsNamespace,
		Tags:                              cloneStringMap(cluster.Tags),
		RegisteredContainerInstancesCount: cluster.RegisteredContainerInstancesCount,
		RunningTasksCount:                 cluster.RunningTasksCount,
		PendingTasksCount:                 cluster.PendingTasksCount,
		ActiveServicesCount:               cluster.ActiveServicesCount,
		CapacityProviders:                 append([]string(nil), cluster.CapacityProviders...),
		DefaultCapacityProviderStrategy:   strategy,
		CreatedAt:                         cluster.CreatedAt,
		UpdatedAt:                         cluster.UpdatedAt,
	}
}

func cloneCapacityProvider(cp *CapacityProvider) CapacityProvider {
	if cp == nil {
		return CapacityProvider{}
	}
	return CapacityProvider{
		Name:                         cp.Name,
		Arn:                          cp.Arn,
		Status:                       cp.Status,
		UpdateStatus:                 cp.UpdateStatus,
		AutoScalingGroupArn:          cp.AutoScalingGroupArn,
		ManagedScalingStatus:         cp.ManagedScalingStatus,
		ManagedTerminationProtection: cp.ManagedTerminationProtection,
		Tags:                         cloneStringMap(cp.Tags),
		CreatedAt:                    cp.CreatedAt,
		UpdatedAt:                    cp.UpdatedAt,
	}
}

func cloneTaskDefinition(def *TaskDefinition) TaskDefinition {
	if def == nil {
		return TaskDefinition{}
	}
	containers := make([]ContainerDefinition, 0, len(def.ContainerDefinitions))
	for _, container := range def.ContainerDefinitions {
		containers = append(containers, ContainerDefinition{
			Name:  container.Name,
			Image: container.Image,
		})
	}
	var deregisteredAt *time.Time
	if def.DeregisteredAt != nil {
		t := *def.DeregisteredAt
		deregisteredAt = &t
	}
	return TaskDefinition{
		Family:                  def.Family,
		Revision:                def.Revision,
		Arn:                     def.Arn,
		Status:                  def.Status,
		NetworkMode:             def.NetworkMode,
		Cpu:                     def.Cpu,
		Memory:                  def.Memory,
		ExecutionRoleArn:        def.ExecutionRoleArn,
		TaskRoleArn:             def.TaskRoleArn,
		RequiresCompatibilities: append([]string(nil), def.RequiresCompatibilities...),
		ContainerDefinitions:    containers,
		Tags:                    cloneStringMap(def.Tags),
		RegisteredAt:            def.RegisteredAt,
		DeregisteredAt:          deregisteredAt,
	}
}

func cloneServiceDefinition(service *ServiceDefinition) ServiceDefinition {
	if service == nil {
		return ServiceDefinition{}
	}
	return ServiceDefinition{
		Name:              service.Name,
		Arn:               service.Arn,
		ClusterArn:        service.ClusterArn,
		TaskDefinitionArn: service.TaskDefinitionArn,
		PrimaryTaskSetArn: service.PrimaryTaskSetArn,
		DesiredCount:      service.DesiredCount,
		LaunchType:        service.LaunchType,
		Status:            service.Status,
		Tags:              cloneStringMap(service.Tags),
		CreatedAt:         service.CreatedAt,
		UpdatedAt:         service.UpdatedAt,
	}
}

func cloneTaskSet(taskSet *TaskSet) TaskSet {
	if taskSet == nil {
		return TaskSet{}
	}
	return TaskSet{
		Arn:               taskSet.Arn,
		ID:                taskSet.ID,
		ClusterArn:        taskSet.ClusterArn,
		ServiceArn:        taskSet.ServiceArn,
		TaskDefinitionArn: taskSet.TaskDefinitionArn,
		ComputedDesired:   taskSet.ComputedDesired,
		PendingCount:      taskSet.PendingCount,
		RunningCount:      taskSet.RunningCount,
		Status:            taskSet.Status,
		LaunchType:        taskSet.LaunchType,
		ScaleValue:        taskSet.ScaleValue,
		ScaleUnit:         taskSet.ScaleUnit,
		CreatedAt:         taskSet.CreatedAt,
		UpdatedAt:         taskSet.UpdatedAt,
	}
}

func cloneTask(task *Task) Task {
	if task == nil {
		return Task{}
	}
	out := Task{
		Arn:                  task.Arn,
		ClusterArn:           task.ClusterArn,
		TaskDefinitionArn:    task.TaskDefinitionArn,
		ServiceArn:           task.ServiceArn,
		ContainerInstanceArn: task.ContainerInstanceArn,
		Group:                task.Group,
		StartedBy:            task.StartedBy,
		LaunchType:           task.LaunchType,
		LastStatus:           task.LastStatus,
		DesiredStatus:        task.DesiredStatus,
		CreatedAt:            task.CreatedAt,
		StartedAt:            task.StartedAt,
		StoppedReason:        task.StoppedReason,
	}
	if task.StoppedAt != nil {
		t := *task.StoppedAt
		out.StoppedAt = &t
	}
	return out
}

func cloneContainerInstance(containerInstance *ContainerInstance) ContainerInstance {
	if containerInstance == nil {
		return ContainerInstance{}
	}
	return ContainerInstance{
		Arn:               containerInstance.Arn,
		ClusterArn:        containerInstance.ClusterArn,
		Ec2InstanceID:     containerInstance.Ec2InstanceID,
		Status:            containerInstance.Status,
		AgentConnected:    containerInstance.AgentConnected,
		AgentUpdateStatus: containerInstance.AgentUpdateStatus,
		Version:           containerInstance.Version,
		VersionInfo: VersionInfo{
			AgentHash:     containerInstance.VersionInfo.AgentHash,
			AgentVersion:  containerInstance.VersionInfo.AgentVersion,
			DockerVersion: containerInstance.VersionInfo.DockerVersion,
		},
		RegisteredAt: containerInstance.RegisteredAt,
		UpdatedAt:    containerInstance.UpdatedAt,
		Attributes:   cloneAttributes(containerInstance.Attributes),
	}
}

func cloneExpressGatewayService(service *ExpressGatewayService) ExpressGatewayService {
	if service == nil {
		return ExpressGatewayService{}
	}
	configurations := make([]ExpressGatewayServiceConfiguration, 0, len(service.ActiveConfigurations))
	for _, cfg := range service.ActiveConfigurations {
		configurations = append(configurations, cloneExpressGatewayServiceConfiguration(cfg))
	}
	return ExpressGatewayService{
		ServiceArn:            service.ServiceArn,
		ServiceName:           service.ServiceName,
		Cluster:               service.Cluster,
		InfrastructureRoleArn: service.InfrastructureRoleArn,
		StatusCode:            service.StatusCode,
		StatusReason:          service.StatusReason,
		CurrentDeployment:     service.CurrentDeployment,
		ActiveConfigurations:  configurations,
		Tags:                  cloneStringMap(service.Tags),
		CreatedAt:             service.CreatedAt,
		UpdatedAt:             service.UpdatedAt,
	}
}

func cloneExpressGatewayServiceConfiguration(cfg ExpressGatewayServiceConfiguration) ExpressGatewayServiceConfiguration {
	return ExpressGatewayServiceConfiguration{
		ServiceRevisionArn:   cfg.ServiceRevisionArn,
		Cpu:                  cfg.Cpu,
		ExecutionRoleArn:     cfg.ExecutionRoleArn,
		HealthCheckPath:      cfg.HealthCheckPath,
		Memory:               cfg.Memory,
		TaskRoleArn:          cfg.TaskRoleArn,
		PrimaryContainer:     cloneAnyMap(cfg.PrimaryContainer),
		NetworkConfiguration: cloneAnyMap(cfg.NetworkConfiguration),
		ScalingTarget:        cloneAnyMap(cfg.ScalingTarget),
		CreatedAt:            cfg.CreatedAt,
	}
}

func cloneTaskProtection(taskProtection TaskProtection) TaskProtection {
	out := TaskProtection{
		TaskArn:           taskProtection.TaskArn,
		ProtectionEnabled: taskProtection.ProtectionEnabled,
	}
	if taskProtection.ExpirationDate != nil {
		exp := *taskProtection.ExpirationDate
		out.ExpirationDate = &exp
	}
	return out
}

func cloneAttribute(attribute Attribute) Attribute {
	return Attribute{
		Name:       attribute.Name,
		Value:      attribute.Value,
		TargetType: attribute.TargetType,
		TargetID:   attribute.TargetID,
	}
}

func cloneAttributes(attributes []Attribute) []Attribute {
	if len(attributes) == 0 {
		return []Attribute{}
	}
	out := make([]Attribute, len(attributes))
	for idx, attribute := range attributes {
		out[idx] = cloneAttribute(attribute)
	}
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneAnyValue(value)
	}
	return out
}

func cloneAnyValue(in any) any {
	switch v := in.(type) {
	case map[string]any:
		return cloneAnyMap(v)
	case []any:
		out := make([]any, len(v))
		for idx, item := range v {
			out[idx] = cloneAnyValue(item)
		}
		return out
	case []string:
		return append([]string(nil), v...)
	case []int:
		return append([]int(nil), v...)
	case []int32:
		return append([]int32(nil), v...)
	case []int64:
		return append([]int64(nil), v...)
	case []float64:
		return append([]float64(nil), v...)
	case []bool:
		return append([]bool(nil), v...)
	default:
		return in
	}
}

func anyString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func cloneServiceDeploymentSnapshot(in ServiceDeploymentSnapshot) ServiceDeploymentSnapshot {
	out := ServiceDeploymentSnapshot{
		Arn:                in.Arn,
		ServiceArn:         in.ServiceArn,
		ServiceRevisionArn: in.ServiceRevisionArn,
		Status:             in.Status,
		StatusReason:       in.StatusReason,
		CreatedAt:          in.CreatedAt,
		UpdatedAt:          in.UpdatedAt,
	}
	if in.StoppedAt != nil {
		t := *in.StoppedAt
		out.StoppedAt = &t
	}
	return out
}

func cloneServiceDeploymentSnapshots(in []ServiceDeploymentSnapshot) []ServiceDeploymentSnapshot {
	if len(in) == 0 {
		return []ServiceDeploymentSnapshot{}
	}
	out := make([]ServiceDeploymentSnapshot, len(in))
	for idx, item := range in {
		out[idx] = cloneServiceDeploymentSnapshot(item)
	}
	return out
}

func cloneServiceRevisionSnapshot(in ServiceRevisionSnapshot) ServiceRevisionSnapshot {
	return ServiceRevisionSnapshot{
		Arn:               in.Arn,
		ServiceArn:        in.ServiceArn,
		TaskDefinitionArn: in.TaskDefinitionArn,
		DesiredCount:      in.DesiredCount,
		CreatedAt:         in.CreatedAt,
	}
}

func cloneServiceRevisionSnapshots(in []ServiceRevisionSnapshot) []ServiceRevisionSnapshot {
	if len(in) == 0 {
		return []ServiceRevisionSnapshot{}
	}
	out := make([]ServiceRevisionSnapshot, len(in))
	for idx, item := range in {
		out[idx] = cloneServiceRevisionSnapshot(item)
	}
	return out
}

func cloneStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for key, value := range m {
		out[key] = value
	}
	return out
}

func accountSettingKey(principalArn, name string) string {
	return strings.TrimSpace(principalArn) + "\x00" + strings.TrimSpace(name)
}

func splitAccountSettingKey(key string) (principalArn string, name string) {
	parts := strings.SplitN(key, "\x00", 2)
	if len(parts) != 2 {
		return "", strings.TrimSpace(key)
	}
	return parts[0], parts[1]
}

func clusterARN(name string) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", DefaultRegion, DefaultAccountID, name)
}

func capacityProviderARN(name string) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:capacity-provider/%s", DefaultRegion, DefaultAccountID, name)
}

func taskDefinitionARN(family string, revision int64) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s:%d", DefaultRegion, DefaultAccountID, family, revision)
}

func serviceARN(clusterName, serviceName string) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", DefaultRegion, DefaultAccountID, clusterName, serviceName)
}

func expressGatewayServiceARN(clusterName, serviceName string) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:express-gateway-service/%s/%s", DefaultRegion, DefaultAccountID, clusterName, serviceName)
}

func expressGatewayServiceRevisionARN(clusterName, serviceName string, revision int) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:express-gateway-service-revision/%s/%s/%d", DefaultRegion, DefaultAccountID, clusterName, serviceName, revision)
}

func serviceRevisionARN(clusterName, serviceName string, revision int) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:service-revision/%s/%s/%d", DefaultRegion, DefaultAccountID, clusterName, serviceName, revision)
}

func serviceDeploymentARN(clusterName, serviceName string, deployment int) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:service-deployment/%s/%s/%d", DefaultRegion, DefaultAccountID, clusterName, serviceName, deployment)
}

func taskSetARN(clusterName, serviceName, taskSetID string) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:task-set/%s/%s/%s", DefaultRegion, DefaultAccountID, clusterName, serviceName, taskSetID)
}

func taskARN(clusterName string, taskSeq int64) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%012d", DefaultRegion, DefaultAccountID, clusterName, taskSeq)
}

func containerInstanceARN(clusterName string, seq int64) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:container-instance/%s/%012d", DefaultRegion, DefaultAccountID, clusterName, seq)
}

func parseServiceARN(ref string) (clusterName string, serviceName string, ok bool) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, "arn:") {
		return "", "", false
	}
	idx := strings.Index(ref, ":service/")
	if idx == -1 || idx+len(":service/") >= len(ref) {
		return "", "", false
	}
	rest := ref[idx+len(":service/"):]
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	clusterName = strings.TrimSpace(parts[0])
	serviceName = strings.TrimSpace(parts[1])
	if clusterName == "" || serviceName == "" {
		return "", "", false
	}
	return clusterName, serviceName, true
}

func parseExpressGatewayServiceARN(ref string) (clusterName string, serviceName string, ok bool) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, "arn:") {
		return "", "", false
	}
	idx := strings.Index(ref, ":express-gateway-service/")
	if idx == -1 || idx+len(":express-gateway-service/") >= len(ref) {
		return "", "", false
	}
	rest := ref[idx+len(":express-gateway-service/"):]
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	clusterName = strings.TrimSpace(parts[0])
	serviceName = strings.TrimSpace(parts[1])
	if clusterName == "" || serviceName == "" {
		return "", "", false
	}
	return clusterName, serviceName, true
}

func parseTaskSetARN(ref string) (clusterName string, serviceName string, taskSetID string, ok bool) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, "arn:") {
		return "", "", "", false
	}
	idx := strings.Index(ref, ":task-set/")
	if idx == -1 || idx+len(":task-set/") >= len(ref) {
		return "", "", "", false
	}
	rest := ref[idx+len(":task-set/"):]
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) != 3 {
		return "", "", "", false
	}
	clusterName = strings.TrimSpace(parts[0])
	serviceName = strings.TrimSpace(parts[1])
	taskSetID = strings.TrimSpace(parts[2])
	if clusterName == "" || serviceName == "" || taskSetID == "" {
		return "", "", "", false
	}
	return clusterName, serviceName, taskSetID, true
}

func parseTaskARN(ref string) (clusterName string, taskID string, ok bool) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, "arn:") {
		return "", "", false
	}
	idx := strings.Index(ref, ":task/")
	if idx == -1 || idx+len(":task/") >= len(ref) {
		return "", "", false
	}
	rest := ref[idx+len(":task/"):]
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	clusterName = strings.TrimSpace(parts[0])
	taskID = strings.TrimSpace(parts[1])
	if clusterName == "" || taskID == "" {
		return "", "", false
	}
	return clusterName, taskID, true
}

func parseContainerInstanceARN(ref string) (clusterName string, instanceID string, ok bool) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, "arn:") {
		return "", "", false
	}
	idx := strings.Index(ref, ":container-instance/")
	if idx == -1 || idx+len(":container-instance/") >= len(ref) {
		return "", "", false
	}
	rest := ref[idx+len(":container-instance/"):]
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	clusterName = strings.TrimSpace(parts[0])
	instanceID = strings.TrimSpace(parts[1])
	if clusterName == "" || instanceID == "" {
		return "", "", false
	}
	return clusterName, instanceID, true
}

func attributeKey(targetType, targetID, name string) string {
	return strings.TrimSpace(targetType) + "\x00" + strings.TrimSpace(targetID) + "\x00" + strings.TrimSpace(name)
}

func parseTaskDefinitionRef(ref string) (family string, revision int64, ok bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", 0, false
	}
	if strings.HasPrefix(ref, "arn:") {
		i := strings.LastIndex(ref, "/")
		if i == -1 || i+1 >= len(ref) {
			return "", 0, false
		}
		ref = ref[i+1:]
	}
	parts := strings.Split(ref, ":")
	if len(parts) < 2 {
		return "", 0, false
	}
	revStr := strings.TrimSpace(parts[len(parts)-1])
	family = strings.TrimSpace(strings.Join(parts[:len(parts)-1], ":"))
	if family == "" || revStr == "" {
		return "", 0, false
	}
	rev, err := strconv.ParseInt(revStr, 10, 64)
	if err != nil || rev <= 0 {
		return "", 0, false
	}
	return family, rev, true
}
