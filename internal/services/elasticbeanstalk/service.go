package elasticbeanstalk

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
	ErrAlreadyExists    = errors.New("already exists")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"
)

type S3Location struct {
	S3Bucket string
	S3Key    string
}

type Tag struct {
	Key   string
	Value string
}

type OptionSetting struct {
	Namespace    string
	OptionName   string
	ResourceName string
	Value        string
}

type OptionSpecification struct {
	Namespace    string
	OptionName   string
	ResourceName string
}

type ConfigurationOptionDescription struct {
	Namespace      string
	Name           string
	DefaultValue   string
	ValueType      string
	ChangeSeverity string
	UserDefined    bool
	Description    string
}

type ConfigurationSettingsDescription struct {
	ApplicationName   string
	TemplateName      string
	EnvironmentName   string
	Description       string
	SolutionStackName string
	DateCreated       time.Time
	DateUpdated       time.Time
	OptionSettings    []OptionSetting
}

type ValidationMessage struct {
	Message    string
	Namespace  string
	OptionName string
	Severity   string
}

type Application struct {
	ARN                    string
	Name                   string
	Description            string
	DateCreated            time.Time
	DateUpdated            time.Time
	Versions               []string
	ConfigurationTemplates []string
}

type ApplicationVersion struct {
	ARN             string
	ApplicationName string
	VersionLabel    string
	Description     string
	DateCreated     time.Time
	DateUpdated     time.Time
	SourceBundle    S3Location
	Status          string
}

type Environment struct {
	ARN                          string
	ID                           string
	Name                         string
	ApplicationName              string
	VersionLabel                 string
	Description                  string
	CNAME                        string
	EndpointURL                  string
	TemplateName                 string
	SolutionStackName            string
	DateCreated                  time.Time
	DateUpdated                  time.Time
	Status                       string
	Health                       string
	HealthStatus                 string
	AbortableOperationInProgress bool
	TierName                     string
	TierType                     string
	OperationsRole               string
	OptionSettings               []OptionSetting
}

type EnvironmentResourceDescription struct {
	EnvironmentName      string
	AutoScalingGroups    []string
	Instances            []string
	LaunchConfigurations []string
	LoadBalancers        []string
	Queues               []string
	Triggers             []string
}

type Event struct {
	ApplicationName string
	EnvironmentName string
	EventDate       time.Time
	Message         string
	Severity        string
	RequestID       string
	VersionLabel    string
}

type EnvironmentInfo struct {
	Ec2InstanceID   string
	InfoType        string
	Message         string
	SampleTimestamp time.Time
}

type ResourceQuota struct {
	Maximum int32
}

type ResourceQuotas struct {
	ApplicationQuota           ResourceQuota
	ApplicationVersionQuota    ResourceQuota
	ConfigurationTemplateQuota ResourceQuota
	CustomPlatformQuota        ResourceQuota
	EnvironmentQuota           ResourceQuota
}

type PlatformFilter struct {
	Type     string
	Operator string
	Values   []string
}

type SearchFilter struct {
	Attribute string
	Operator  string
	Values    []string
}

type PlatformVersion struct {
	PlatformARN                  string
	PlatformName                 string
	PlatformVersion              string
	PlatformOwner                string
	PlatformStatus               string
	PlatformBranchName           string
	PlatformBranchLifecycleState string
	PlatformLifecycleState       string
	PlatformCategory             string
	SupportedTierList            []string
	SupportedAddonList           []string
	OperatingSystemName          string
	OperatingSystemVersion       string
	Description                  string
	Maintainer                   string
	SolutionStackName            string
	BuilderARN                   string
	DateCreated                  time.Time
	DateUpdated                  time.Time
}

type PlatformBranch struct {
	BranchName        string
	BranchOrder       int32
	LifecycleState    string
	PlatformName      string
	SupportedTierList []string
}

type MetricsStatusCodes struct {
	Status2xx int32
	Status3xx int32
	Status4xx int32
	Status5xx int32
}

type HealthApplicationMetrics struct {
	Duration     int32
	RequestCount int32
	StatusCodes  MetricsStatusCodes
}

type HealthInstanceSummary struct {
	Degraded int32
	Info     int32
	NoData   int32
	Ok       int32
	Pending  int32
	Severe   int32
	Unknown  int32
	Warning  int32
}

type EnvironmentHealth struct {
	EnvironmentName    string
	Color              string
	HealthStatus       string
	Status             string
	Causes             []string
	ApplicationMetrics HealthApplicationMetrics
	InstancesHealth    HealthInstanceSummary
	RefreshedAt        time.Time
}

type InstanceHealth struct {
	InstanceID         string
	AvailabilityZone   string
	Color              string
	HealthStatus       string
	InstanceType       string
	LaunchedAt         time.Time
	Causes             []string
	ApplicationMetrics HealthApplicationMetrics
}

type ManagedAction struct {
	ActionID          string
	ActionDescription string
	ActionType        string
	Status            string
	WindowStartTime   time.Time
}

type ManagedActionHistoryItem struct {
	ActionID           string
	ActionDescription  string
	ActionType         string
	ExecutedTime       time.Time
	FinishedTime       time.Time
	Status             string
	FailureType        string
	FailureDescription string
}

type MaxAgeRule struct {
	Enabled            bool
	DeleteSourceFromS3 bool
	MaxAgeInDays       int32
}

type MaxCountRule struct {
	Enabled            bool
	DeleteSourceFromS3 bool
	MaxCount           int32
}

type ApplicationVersionLifecycleConfig struct {
	MaxAgeRule   *MaxAgeRule
	MaxCountRule *MaxCountRule
}

type ApplicationResourceLifecycleConfig struct {
	ServiceRole            string
	VersionLifecycleConfig *ApplicationVersionLifecycleConfig
}

type ConfigurationTemplate struct {
	ARN               string
	ApplicationName   string
	TemplateName      string
	Description       string
	SolutionStackName string
	DateCreated       time.Time
	DateUpdated       time.Time
	OptionSettings    []OptionSetting
}

type Service struct {
	mu                     sync.Mutex
	seq                    uint64
	applications           map[string]*Application
	applicationVersions    map[string]map[string]*ApplicationVersion
	environmentsByID       map[string]*Environment
	environmentIDByName    map[string]string
	platformVersionsByARN  map[string]*PlatformVersion
	managedActionsByEnvID  map[string][]ManagedAction
	managedActionHistByEnv map[string][]ManagedActionHistoryItem
	appResourceLifecycle   map[string]ApplicationResourceLifecycleConfig
	events                 []Event
	environmentInfoByKey   map[string][]EnvironmentInfo
	configurationTemplates map[string]map[string]*ConfigurationTemplate
	tagsByResourceARN      map[string]map[string]string
	solutionStacks         []string
	storageBucket          string
	accountResourceQuotas  ResourceQuotas
}

func NewService() *Service {
	return &Service{
		applications:           map[string]*Application{},
		applicationVersions:    map[string]map[string]*ApplicationVersion{},
		environmentsByID:       map[string]*Environment{},
		environmentIDByName:    map[string]string{},
		platformVersionsByARN:  map[string]*PlatformVersion{},
		managedActionsByEnvID:  map[string][]ManagedAction{},
		managedActionHistByEnv: map[string][]ManagedActionHistoryItem{},
		appResourceLifecycle:   map[string]ApplicationResourceLifecycleConfig{},
		events:                 []Event{},
		environmentInfoByKey:   map[string][]EnvironmentInfo{},
		configurationTemplates: map[string]map[string]*ConfigurationTemplate{},
		tagsByResourceARN:      map[string]map[string]string{},
		solutionStacks: []string{
			"64bit Amazon Linux 2 v3.6.3 running Go 1",
			"64bit Amazon Linux 2 v5.8.4 running Node.js 20",
			"64bit Amazon Linux 2 v3.5.10 running Python 3.11",
		},
		storageBucket: fmt.Sprintf("elasticbeanstalk-%s-%s", DefaultRegion, DefaultAccountID),
		accountResourceQuotas: ResourceQuotas{
			ApplicationQuota:           ResourceQuota{Maximum: 75},
			ApplicationVersionQuota:    ResourceQuota{Maximum: 1000},
			ConfigurationTemplateQuota: ResourceQuota{Maximum: 200},
			CustomPlatformQuota:        ResourceQuota{Maximum: 25},
			EnvironmentQuota:           ResourceQuota{Maximum: 200},
		},
	}
}

func (s *Service) CreateApplication(name, description string, tags []Tag) (Application, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Application{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.applications[name]; exists {
		return Application{}, ErrAlreadyExists
	}

	now := time.Now().UTC()
	app := &Application{
		ARN:         applicationARN(name),
		Name:        name,
		Description: strings.TrimSpace(description),
		DateCreated: now,
		DateUpdated: now,
		Versions:    []string{},
	}
	s.applications[name] = app
	s.applicationVersions[name] = map[string]*ApplicationVersion{}
	s.configurationTemplates[name] = map[string]*ConfigurationTemplate{}
	s.appResourceLifecycle[name] = ApplicationResourceLifecycleConfig{}
	s.setTagsLocked(app.ARN, tags)

	return cloneApplication(app), nil
}

func (s *Service) UpdateApplication(name, description string) (Application, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Application{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	app := s.applications[name]
	if app == nil {
		return Application{}, ErrNotFound
	}
	app.Description = strings.TrimSpace(description)
	app.DateUpdated = time.Now().UTC()
	return cloneApplication(app), nil
}

func (s *Service) DeleteApplication(name string, terminateEnvByForce bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	app := s.applications[name]
	if app == nil {
		return ErrNotFound
	}

	activeEnvIDs := make([]string, 0)
	for _, env := range s.environmentsByID {
		if env.ApplicationName != name || env.Status == "Terminated" {
			continue
		}
		activeEnvIDs = append(activeEnvIDs, env.ID)
	}
	if len(activeEnvIDs) > 0 && !terminateEnvByForce {
		return ErrConflict
	}
	for _, envID := range activeEnvIDs {
		s.terminateEnvironmentLocked(envID)
	}

	delete(s.applications, name)
	delete(s.applicationVersions, name)
	delete(s.configurationTemplates, name)
	delete(s.appResourceLifecycle, name)
	delete(s.tagsByResourceARN, app.ARN)

	for arn := range s.tagsByResourceARN {
		if strings.Contains(arn, "/"+name+"/") {
			delete(s.tagsByResourceARN, arn)
		}
	}

	return nil
}

func (s *Service) DescribeApplications(names []string) []Application {
	s.mu.Lock()
	defer s.mu.Unlock()

	nameSet := make(map[string]struct{}, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		nameSet[n] = struct{}{}
	}

	out := make([]Application, 0, len(s.applications))
	for _, app := range s.applications {
		if len(nameSet) > 0 {
			if _, ok := nameSet[app.Name]; !ok {
				continue
			}
		}
		out = append(out, cloneApplication(app))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) CreateApplicationVersion(applicationName, versionLabel, description string, source S3Location, tags []Tag) (ApplicationVersion, error) {
	applicationName = strings.TrimSpace(applicationName)
	versionLabel = strings.TrimSpace(versionLabel)
	if applicationName == "" || versionLabel == "" {
		return ApplicationVersion{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	app := s.applications[applicationName]
	if app == nil {
		return ApplicationVersion{}, ErrNotFound
	}
	versions := s.applicationVersions[applicationName]
	if versions == nil {
		versions = map[string]*ApplicationVersion{}
		s.applicationVersions[applicationName] = versions
	}
	if _, exists := versions[versionLabel]; exists {
		return ApplicationVersion{}, ErrAlreadyExists
	}

	now := time.Now().UTC()
	version := &ApplicationVersion{
		ARN:             applicationVersionARN(applicationName, versionLabel),
		ApplicationName: applicationName,
		VersionLabel:    versionLabel,
		Description:     strings.TrimSpace(description),
		DateCreated:     now,
		DateUpdated:     now,
		SourceBundle: S3Location{
			S3Bucket: strings.TrimSpace(source.S3Bucket),
			S3Key:    strings.TrimSpace(source.S3Key),
		},
		Status: "Processed",
	}
	versions[versionLabel] = version
	app.Versions = append(app.Versions, versionLabel)
	sort.Strings(app.Versions)
	app.DateUpdated = now
	s.setTagsLocked(version.ARN, tags)

	return cloneApplicationVersion(version), nil
}

func (s *Service) UpdateApplicationVersion(applicationName, versionLabel, description string) (ApplicationVersion, error) {
	applicationName = strings.TrimSpace(applicationName)
	versionLabel = strings.TrimSpace(versionLabel)
	if applicationName == "" || versionLabel == "" {
		return ApplicationVersion{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	version := s.applicationVersions[applicationName][versionLabel]
	if version == nil {
		return ApplicationVersion{}, ErrNotFound
	}
	version.Description = strings.TrimSpace(description)
	version.DateUpdated = time.Now().UTC()
	return cloneApplicationVersion(version), nil
}

func (s *Service) DeleteApplicationVersion(applicationName, versionLabel string) error {
	applicationName = strings.TrimSpace(applicationName)
	versionLabel = strings.TrimSpace(versionLabel)
	if applicationName == "" || versionLabel == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	versions := s.applicationVersions[applicationName]
	if versions == nil {
		return ErrNotFound
	}
	version := versions[versionLabel]
	if version == nil {
		return ErrNotFound
	}
	delete(versions, versionLabel)
	delete(s.tagsByResourceARN, version.ARN)

	app := s.applications[applicationName]
	if app != nil {
		filtered := make([]string, 0, len(app.Versions))
		for _, label := range app.Versions {
			if label != versionLabel {
				filtered = append(filtered, label)
			}
		}
		app.Versions = filtered
		app.DateUpdated = time.Now().UTC()
	}

	return nil
}

func (s *Service) DescribeApplicationVersions(applicationName string, labels []string) []ApplicationVersion {
	applicationName = strings.TrimSpace(applicationName)

	s.mu.Lock()
	defer s.mu.Unlock()

	labelSet := map[string]struct{}{}
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		labelSet[label] = struct{}{}
	}

	out := make([]ApplicationVersion, 0)
	for appName, versions := range s.applicationVersions {
		if applicationName != "" && appName != applicationName {
			continue
		}
		for _, version := range versions {
			if len(labelSet) > 0 {
				if _, ok := labelSet[version.VersionLabel]; !ok {
					continue
				}
			}
			out = append(out, cloneApplicationVersion(version))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ApplicationName == out[j].ApplicationName {
			return out[i].VersionLabel < out[j].VersionLabel
		}
		return out[i].ApplicationName < out[j].ApplicationName
	})
	return out
}

func (s *Service) CreateConfigurationTemplate(applicationName, templateName, description, solutionStackName, sourceTemplateName string, optionSettings []OptionSetting) (ConfigurationTemplate, error) {
	applicationName = strings.TrimSpace(applicationName)
	templateName = strings.TrimSpace(templateName)
	sourceTemplateName = strings.TrimSpace(sourceTemplateName)
	if applicationName == "" || templateName == "" {
		return ConfigurationTemplate{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.applications[applicationName] == nil {
		return ConfigurationTemplate{}, ErrNotFound
	}
	templates := s.configurationTemplates[applicationName]
	if templates == nil {
		templates = map[string]*ConfigurationTemplate{}
		s.configurationTemplates[applicationName] = templates
	}
	if _, exists := templates[templateName]; exists {
		return ConfigurationTemplate{}, ErrAlreadyExists
	}

	if sourceTemplateName != "" {
		source := templates[sourceTemplateName]
		if source == nil {
			return ConfigurationTemplate{}, ErrNotFound
		}
		if strings.TrimSpace(description) == "" {
			description = source.Description
		}
		if strings.TrimSpace(solutionStackName) == "" {
			solutionStackName = source.SolutionStackName
		}
		if len(optionSettings) == 0 {
			optionSettings = cloneOptionSettings(source.OptionSettings)
		}
	}
	if strings.TrimSpace(solutionStackName) == "" {
		solutionStackName = s.solutionStacks[0]
	}

	now := time.Now().UTC()
	template := &ConfigurationTemplate{
		ARN:               configurationTemplateARN(applicationName, templateName),
		ApplicationName:   applicationName,
		TemplateName:      templateName,
		Description:       strings.TrimSpace(description),
		SolutionStackName: strings.TrimSpace(solutionStackName),
		DateCreated:       now,
		DateUpdated:       now,
		OptionSettings:    cloneOptionSettings(optionSettings),
	}
	templates[templateName] = template

	app := s.applications[applicationName]
	app.ConfigurationTemplates = append(app.ConfigurationTemplates, templateName)
	sort.Strings(app.ConfigurationTemplates)
	app.DateUpdated = now

	return cloneConfigurationTemplate(template), nil
}

func (s *Service) UpdateConfigurationTemplate(applicationName, templateName, description, solutionStackName string, optionSettings []OptionSetting) (ConfigurationTemplate, error) {
	applicationName = strings.TrimSpace(applicationName)
	templateName = strings.TrimSpace(templateName)
	if applicationName == "" || templateName == "" {
		return ConfigurationTemplate{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	template := s.configurationTemplates[applicationName][templateName]
	if template == nil {
		return ConfigurationTemplate{}, ErrNotFound
	}

	if strings.TrimSpace(description) != "" {
		template.Description = strings.TrimSpace(description)
	}
	if strings.TrimSpace(solutionStackName) != "" {
		template.SolutionStackName = strings.TrimSpace(solutionStackName)
	}
	if len(optionSettings) > 0 {
		template.OptionSettings = cloneOptionSettings(optionSettings)
	}
	template.DateUpdated = time.Now().UTC()

	return cloneConfigurationTemplate(template), nil
}

func (s *Service) DeleteConfigurationTemplate(applicationName, templateName string) error {
	applicationName = strings.TrimSpace(applicationName)
	templateName = strings.TrimSpace(templateName)
	if applicationName == "" || templateName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	templates := s.configurationTemplates[applicationName]
	if templates == nil {
		return ErrNotFound
	}
	template := templates[templateName]
	if template == nil {
		return ErrNotFound
	}
	delete(templates, templateName)
	delete(s.tagsByResourceARN, template.ARN)

	app := s.applications[applicationName]
	if app != nil {
		filtered := make([]string, 0, len(app.ConfigurationTemplates))
		for _, name := range app.ConfigurationTemplates {
			if name != templateName {
				filtered = append(filtered, name)
			}
		}
		app.ConfigurationTemplates = filtered
		app.DateUpdated = time.Now().UTC()
	}
	return nil
}

func (s *Service) DescribeConfigurationSettings(applicationName, templateName, environmentName string) []ConfigurationSettingsDescription {
	applicationName = strings.TrimSpace(applicationName)
	templateName = strings.TrimSpace(templateName)
	environmentName = strings.TrimSpace(environmentName)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]ConfigurationSettingsDescription, 0)
	if environmentName != "" {
		env := s.environmentBySelectorLocked("", environmentName)
		if env != nil && (applicationName == "" || env.ApplicationName == applicationName) {
			out = append(out, ConfigurationSettingsDescription{
				ApplicationName:   env.ApplicationName,
				EnvironmentName:   env.Name,
				Description:       env.Description,
				SolutionStackName: env.SolutionStackName,
				TemplateName:      env.TemplateName,
				DateCreated:       env.DateCreated,
				DateUpdated:       env.DateUpdated,
				OptionSettings:    cloneOptionSettings(env.OptionSettings),
			})
		}
		return out
	}

	for appName, templates := range s.configurationTemplates {
		if applicationName != "" && appName != applicationName {
			continue
		}
		for name, template := range templates {
			if templateName != "" && name != templateName {
				continue
			}
			out = append(out, ConfigurationSettingsDescription{
				ApplicationName:   appName,
				TemplateName:      name,
				Description:       template.Description,
				SolutionStackName: template.SolutionStackName,
				DateCreated:       template.DateCreated,
				DateUpdated:       template.DateUpdated,
				OptionSettings:    cloneOptionSettings(template.OptionSettings),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].ApplicationName == out[j].ApplicationName {
			if out[i].TemplateName == out[j].TemplateName {
				return out[i].EnvironmentName < out[j].EnvironmentName
			}
			return out[i].TemplateName < out[j].TemplateName
		}
		return out[i].ApplicationName < out[j].ApplicationName
	})
	return out
}

func (s *Service) DescribeConfigurationOptions(applicationName, templateName, solutionStackName, environmentName string, optionSettings []OptionSpecification) (string, []ConfigurationOptionDescription, error) {
	applicationName = strings.TrimSpace(applicationName)
	templateName = strings.TrimSpace(templateName)
	solutionStackName = strings.TrimSpace(solutionStackName)
	environmentName = strings.TrimSpace(environmentName)

	s.mu.Lock()
	defer s.mu.Unlock()

	if applicationName != "" && s.applications[applicationName] == nil {
		return "", nil, ErrNotFound
	}
	if environmentName != "" {
		env := s.environmentBySelectorLocked("", environmentName)
		if env == nil {
			return "", nil, ErrNotFound
		}
		if solutionStackName == "" {
			solutionStackName = env.SolutionStackName
		}
	}
	if templateName != "" && applicationName != "" {
		tpl := s.configurationTemplates[applicationName][templateName]
		if tpl == nil {
			return "", nil, ErrNotFound
		}
		if solutionStackName == "" {
			solutionStackName = tpl.SolutionStackName
		}
	}
	if solutionStackName == "" {
		solutionStackName = s.solutionStacks[0]
	}

	defaults := []ConfigurationOptionDescription{
		{Namespace: "aws:autoscaling:launchconfiguration", Name: "InstanceType", DefaultValue: "t3.micro", ValueType: "Scalar", ChangeSeverity: "RestartEnvironment", Description: "EC2 instance type"},
		{Namespace: "aws:elasticbeanstalk:application:environment", Name: "ENV", DefaultValue: "dev", ValueType: "Scalar", ChangeSeverity: "NoInterruption", UserDefined: true, Description: "Environment variable"},
		{Namespace: "aws:elasticbeanstalk:command", Name: "DeploymentPolicy", DefaultValue: "Rolling", ValueType: "Scalar", ChangeSeverity: "RestartApplicationServer", Description: "Deployment policy"},
	}
	if len(optionSettings) == 0 {
		return solutionStackName, defaults, nil
	}

	out := make([]ConfigurationOptionDescription, 0, len(optionSettings))
	for _, spec := range optionSettings {
		ns := strings.TrimSpace(spec.Namespace)
		on := strings.TrimSpace(spec.OptionName)
		if ns == "" || on == "" {
			continue
		}
		out = append(out, ConfigurationOptionDescription{
			Namespace:      ns,
			Name:           on,
			DefaultValue:   "",
			ValueType:      "Scalar",
			ChangeSeverity: "NoInterruption",
			UserDefined:    true,
			Description:    "User supplied option",
		})
	}
	if len(out) == 0 {
		return solutionStackName, defaults, nil
	}
	return solutionStackName, out, nil
}

func (s *Service) ValidateConfigurationSettings(applicationName string, optionSettings []OptionSetting) []ValidationMessage {
	applicationName = strings.TrimSpace(applicationName)
	if applicationName == "" {
		return []ValidationMessage{{
			Message:  "ApplicationName is required",
			Severity: "error",
		}}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.applications[applicationName] == nil {
		return []ValidationMessage{{
			Message:  "application not found",
			Severity: "error",
		}}
	}

	messages := make([]ValidationMessage, 0)
	for _, setting := range optionSettings {
		if strings.TrimSpace(setting.Namespace) == "" || strings.TrimSpace(setting.OptionName) == "" {
			messages = append(messages, ValidationMessage{
				Message:    "Namespace and OptionName are required",
				Namespace:  setting.Namespace,
				OptionName: setting.OptionName,
				Severity:   "error",
			})
		}
	}
	if len(messages) == 0 {
		messages = append(messages, ValidationMessage{
			Message:  "Configuration validation completed",
			Severity: "warning",
		})
	}
	return messages
}

func (s *Service) CreateEnvironment(applicationName, environmentName, cnamePrefix, description, solutionStackName, templateName, versionLabel string, optionSettings []OptionSetting, tags []Tag) (Environment, error) {
	applicationName = strings.TrimSpace(applicationName)
	environmentName = strings.TrimSpace(environmentName)
	cnamePrefix = strings.TrimSpace(cnamePrefix)
	templateName = strings.TrimSpace(templateName)
	versionLabel = strings.TrimSpace(versionLabel)
	if applicationName == "" || environmentName == "" {
		return Environment{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	app := s.applications[applicationName]
	if app == nil {
		return Environment{}, ErrNotFound
	}
	if existingID := s.environmentIDByName[environmentName]; existingID != "" {
		return Environment{}, ErrAlreadyExists
	}
	if templateName != "" {
		if s.configurationTemplates[applicationName][templateName] == nil {
			return Environment{}, ErrNotFound
		}
	}
	if versionLabel != "" {
		if s.applicationVersions[applicationName][versionLabel] == nil {
			return Environment{}, ErrNotFound
		}
	}
	if solutionStackName == "" {
		solutionStackName = s.solutionStacks[0]
	}

	now := time.Now().UTC()
	seq := atomic.AddUint64(&s.seq, 1)
	envID := fmt.Sprintf("e-%08d", seq)
	if cnamePrefix == "" {
		cnamePrefix = normalizeCNAMEPrefix(environmentName)
	}
	if cnamePrefix == "" {
		cnamePrefix = fmt.Sprintf("env-%d", seq)
	}
	fqdn := uniqueCNAMELocked(s.environmentsByID, cnamePrefix)
	env := &Environment{
		ARN:                          environmentARN(applicationName, environmentName),
		ID:                           envID,
		Name:                         environmentName,
		ApplicationName:              applicationName,
		VersionLabel:                 versionLabel,
		Description:                  strings.TrimSpace(description),
		CNAME:                        fqdn,
		EndpointURL:                  fmt.Sprintf("%s.%s", normalizeCNAMEPrefix(environmentName), "elb.local"),
		TemplateName:                 templateName,
		SolutionStackName:            strings.TrimSpace(solutionStackName),
		DateCreated:                  now,
		DateUpdated:                  now,
		Status:                       "Ready",
		Health:                       "Green",
		HealthStatus:                 "Ok",
		AbortableOperationInProgress: false,
		TierName:                     "WebServer",
		TierType:                     "Standard",
		OptionSettings:               cloneOptionSettings(optionSettings),
	}
	s.environmentsByID[envID] = env
	s.environmentIDByName[environmentName] = envID
	s.managedActionsByEnvID[envID] = []ManagedAction{{
		ActionID:          fmt.Sprintf("ma-%s-1", envID),
		ActionDescription: "Apply platform updates",
		ActionType:        "PlatformUpdate",
		Status:            "Scheduled",
		WindowStartTime:   now.Add(24 * time.Hour),
	}}
	s.managedActionHistByEnv[envID] = []ManagedActionHistoryItem{}
	s.setTagsLocked(env.ARN, tags)
	s.appendEventLocked(Event{
		ApplicationName: applicationName,
		EnvironmentName: environmentName,
		EventDate:       now,
		Message:         "Environment created",
		Severity:        "INFO",
		RequestID:       s.nextRequestIDLocked(),
		VersionLabel:    versionLabel,
	})

	app.DateUpdated = now
	return cloneEnvironment(env), nil
}

func (s *Service) UpdateEnvironment(environmentID, environmentName, versionLabel, templateName, description string, optionSettings []OptionSetting) (Environment, error) {
	environmentID = strings.TrimSpace(environmentID)
	environmentName = strings.TrimSpace(environmentName)
	versionLabel = strings.TrimSpace(versionLabel)
	templateName = strings.TrimSpace(templateName)

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(environmentID, environmentName)
	if env == nil {
		return Environment{}, ErrNotFound
	}

	if versionLabel != "" {
		if s.applicationVersions[env.ApplicationName][versionLabel] == nil {
			return Environment{}, ErrNotFound
		}
		env.VersionLabel = versionLabel
	}
	if templateName != "" {
		if s.configurationTemplates[env.ApplicationName][templateName] == nil {
			return Environment{}, ErrNotFound
		}
		env.TemplateName = templateName
	}
	if strings.TrimSpace(description) != "" {
		env.Description = strings.TrimSpace(description)
	}
	if len(optionSettings) > 0 {
		env.OptionSettings = cloneOptionSettings(optionSettings)
	}
	env.Status = "Ready"
	env.Health = "Green"
	env.HealthStatus = "Ok"
	env.AbortableOperationInProgress = false
	env.DateUpdated = time.Now().UTC()
	s.appendEventLocked(Event{
		ApplicationName: env.ApplicationName,
		EnvironmentName: env.Name,
		EventDate:       env.DateUpdated,
		Message:         "Environment updated",
		Severity:        "INFO",
		RequestID:       s.nextRequestIDLocked(),
		VersionLabel:    env.VersionLabel,
	})

	return cloneEnvironment(env), nil
}

func (s *Service) AbortEnvironmentUpdate(environmentID, environmentName string) (Environment, error) {
	environmentID = strings.TrimSpace(environmentID)
	environmentName = strings.TrimSpace(environmentName)

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(environmentID, environmentName)
	if env == nil {
		return Environment{}, ErrNotFound
	}
	env.AbortableOperationInProgress = false
	env.DateUpdated = time.Now().UTC()
	s.appendEventLocked(Event{
		ApplicationName: env.ApplicationName,
		EnvironmentName: env.Name,
		EventDate:       env.DateUpdated,
		Message:         "Environment update aborted",
		Severity:        "WARN",
		RequestID:       s.nextRequestIDLocked(),
		VersionLabel:    env.VersionLabel,
	})
	return cloneEnvironment(env), nil
}

func (s *Service) RebuildEnvironment(environmentID, environmentName string) (Environment, error) {
	environmentID = strings.TrimSpace(environmentID)
	environmentName = strings.TrimSpace(environmentName)

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(environmentID, environmentName)
	if env == nil {
		return Environment{}, ErrNotFound
	}
	env.Status = "Ready"
	env.Health = "Green"
	env.HealthStatus = "Ok"
	env.DateUpdated = time.Now().UTC()
	s.appendEventLocked(Event{
		ApplicationName: env.ApplicationName,
		EnvironmentName: env.Name,
		EventDate:       env.DateUpdated,
		Message:         "Environment rebuilt",
		Severity:        "INFO",
		RequestID:       s.nextRequestIDLocked(),
		VersionLabel:    env.VersionLabel,
	})
	return cloneEnvironment(env), nil
}

func (s *Service) RestartAppServer(environmentID, environmentName string) (Environment, error) {
	environmentID = strings.TrimSpace(environmentID)
	environmentName = strings.TrimSpace(environmentName)

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(environmentID, environmentName)
	if env == nil {
		return Environment{}, ErrNotFound
	}
	env.DateUpdated = time.Now().UTC()
	s.appendEventLocked(Event{
		ApplicationName: env.ApplicationName,
		EnvironmentName: env.Name,
		EventDate:       env.DateUpdated,
		Message:         "Application server restarted",
		Severity:        "INFO",
		RequestID:       s.nextRequestIDLocked(),
		VersionLabel:    env.VersionLabel,
	})
	return cloneEnvironment(env), nil
}

func (s *Service) TerminateEnvironment(environmentID, environmentName string) (Environment, error) {
	environmentID = strings.TrimSpace(environmentID)
	environmentName = strings.TrimSpace(environmentName)

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(environmentID, environmentName)
	if env == nil {
		return Environment{}, ErrNotFound
	}
	s.terminateEnvironmentLocked(env.ID)
	return cloneEnvironment(env), nil
}

func (s *Service) terminateEnvironmentLocked(environmentID string) {
	env := s.environmentsByID[environmentID]
	if env == nil {
		return
	}
	env.Status = "Terminated"
	env.Health = "Grey"
	env.HealthStatus = "NoData"
	env.AbortableOperationInProgress = false
	env.DateUpdated = time.Now().UTC()
	s.appendEventLocked(Event{
		ApplicationName: env.ApplicationName,
		EnvironmentName: env.Name,
		EventDate:       env.DateUpdated,
		Message:         "Environment terminated",
		Severity:        "INFO",
		RequestID:       s.nextRequestIDLocked(),
		VersionLabel:    env.VersionLabel,
	})
}

func (s *Service) SwapEnvironmentCNAMEs(sourceID, sourceName, destinationID, destinationName string) error {
	sourceID = strings.TrimSpace(sourceID)
	sourceName = strings.TrimSpace(sourceName)
	destinationID = strings.TrimSpace(destinationID)
	destinationName = strings.TrimSpace(destinationName)

	s.mu.Lock()
	defer s.mu.Unlock()

	src := s.environmentBySelectorLocked(sourceID, sourceName)
	dst := s.environmentBySelectorLocked(destinationID, destinationName)
	if src == nil || dst == nil {
		return ErrNotFound
	}
	src.CNAME, dst.CNAME = dst.CNAME, src.CNAME
	now := time.Now().UTC()
	src.DateUpdated = now
	dst.DateUpdated = now
	return nil
}

func (s *Service) DescribeEnvironments(applicationName string, environmentIDs, environmentNames []string, includeDeleted bool) []Environment {
	applicationName = strings.TrimSpace(applicationName)

	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := make(map[string]struct{}, len(environmentIDs))
	for _, id := range environmentIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		idSet[id] = struct{}{}
	}
	nameSet := make(map[string]struct{}, len(environmentNames))
	for _, name := range environmentNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		nameSet[name] = struct{}{}
	}

	out := make([]Environment, 0, len(s.environmentsByID))
	for _, env := range s.environmentsByID {
		if applicationName != "" && env.ApplicationName != applicationName {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[env.ID]; !ok {
				continue
			}
		}
		if len(nameSet) > 0 {
			if _, ok := nameSet[env.Name]; !ok {
				continue
			}
		}
		if !includeDeleted && env.Status == "Terminated" {
			continue
		}
		out = append(out, cloneEnvironment(env))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) DescribeEnvironmentResources(environmentID, environmentName string) (EnvironmentResourceDescription, error) {
	environmentID = strings.TrimSpace(environmentID)
	environmentName = strings.TrimSpace(environmentName)

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(environmentID, environmentName)
	if env == nil {
		return EnvironmentResourceDescription{}, ErrNotFound
	}

	suffix := normalizeResourceToken(env.Name)
	if suffix == "" {
		suffix = "env"
	}
	return EnvironmentResourceDescription{
		EnvironmentName:      env.Name,
		AutoScalingGroups:    []string{"awseb-e-" + suffix + "-asg"},
		Instances:            []string{"i-" + fmt.Sprintf("%08x", hashToken(env.ID))},
		LaunchConfigurations: []string{"awseb-e-" + suffix + "-launchconfig"},
		LoadBalancers:        []string{"awseb-e-" + suffix + "-lb"},
		Queues:               []string{"awseb-e-" + suffix + "-queue"},
		Triggers:             []string{"awseb-e-" + suffix + "-trigger"},
	}, nil
}

func (s *Service) CheckDNSAvailability(prefix string) (bool, string, error) {
	prefix = normalizeCNAMEPrefix(prefix)
	if prefix == "" {
		return false, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	fqdn := prefix + ".elasticbeanstalk.local"
	for _, env := range s.environmentsByID {
		if env.CNAME == fqdn && env.Status != "Terminated" {
			return false, fqdn, nil
		}
	}
	return true, fqdn, nil
}

func (s *Service) DescribeEvents(applicationName, environmentName string, maxRecords int) []Event {
	applicationName = strings.TrimSpace(applicationName)
	environmentName = strings.TrimSpace(environmentName)
	if maxRecords <= 0 {
		maxRecords = 50
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]Event, 0, len(s.events))
	for _, event := range s.events {
		if applicationName != "" && event.ApplicationName != applicationName {
			continue
		}
		if environmentName != "" && event.EnvironmentName != environmentName {
			continue
		}
		filtered = append(filtered, event)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].EventDate.After(filtered[j].EventDate)
	})
	if len(filtered) > maxRecords {
		filtered = filtered[:maxRecords]
	}
	out := make([]Event, len(filtered))
	copy(out, filtered)
	return out
}

func (s *Service) RequestEnvironmentInfo(environmentID, environmentName, infoType string) error {
	infoType = strings.TrimSpace(infoType)
	if infoType == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(strings.TrimSpace(environmentID), strings.TrimSpace(environmentName))
	if env == nil {
		return ErrNotFound
	}
	entry := EnvironmentInfo{
		Ec2InstanceID:   "i-" + fmt.Sprintf("%08x", hashToken(env.ID)),
		InfoType:        infoType,
		Message:         fmt.Sprintf("https://logs.stackyard.local/%s/%s", env.ID, strings.ToLower(infoType)),
		SampleTimestamp: time.Now().UTC(),
	}
	key := env.ID + "|" + strings.ToLower(infoType)
	s.environmentInfoByKey[key] = []EnvironmentInfo{entry}
	return nil
}

func (s *Service) RetrieveEnvironmentInfo(environmentID, environmentName, infoType string) ([]EnvironmentInfo, error) {
	infoType = strings.TrimSpace(infoType)
	if infoType == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(strings.TrimSpace(environmentID), strings.TrimSpace(environmentName))
	if env == nil {
		return nil, ErrNotFound
	}
	key := env.ID + "|" + strings.ToLower(infoType)
	items := s.environmentInfoByKey[key]
	out := make([]EnvironmentInfo, len(items))
	copy(out, items)
	return out, nil
}

func (s *Service) CreateStorageLocation() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.storageBucket
}

func (s *Service) ListAvailableSolutionStacks() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.solutionStacks))
	copy(out, s.solutionStacks)
	return out
}

func (s *Service) DescribeAccountAttributes() ResourceQuotas {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneResourceQuotas(s.accountResourceQuotas)
}

func (s *Service) AssociateEnvironmentOperationsRole(environmentName, operationsRole string) error {
	environmentName = strings.TrimSpace(environmentName)
	operationsRole = strings.TrimSpace(operationsRole)
	if environmentName == "" || operationsRole == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked("", environmentName)
	if env == nil {
		return ErrNotFound
	}
	env.OperationsRole = operationsRole
	env.DateUpdated = time.Now().UTC()
	s.appendEventLocked(Event{
		ApplicationName: env.ApplicationName,
		EnvironmentName: env.Name,
		EventDate:       env.DateUpdated,
		Message:         "Environment operations role associated",
		Severity:        "INFO",
		RequestID:       s.nextRequestIDLocked(),
		VersionLabel:    env.VersionLabel,
	})
	return nil
}

func (s *Service) DisassociateEnvironmentOperationsRole(environmentName string) error {
	environmentName = strings.TrimSpace(environmentName)
	if environmentName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked("", environmentName)
	if env == nil {
		return ErrNotFound
	}
	env.OperationsRole = ""
	env.DateUpdated = time.Now().UTC()
	s.appendEventLocked(Event{
		ApplicationName: env.ApplicationName,
		EnvironmentName: env.Name,
		EventDate:       env.DateUpdated,
		Message:         "Environment operations role disassociated",
		Severity:        "INFO",
		RequestID:       s.nextRequestIDLocked(),
		VersionLabel:    env.VersionLabel,
	})
	return nil
}

func (s *Service) ComposeEnvironments(applicationName, groupName string, versionLabels []string) ([]Environment, error) {
	applicationName = strings.TrimSpace(applicationName)
	_ = strings.TrimSpace(groupName)
	if applicationName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.applications[applicationName] == nil {
		return nil, ErrNotFound
	}

	versionSet := make(map[string]struct{}, len(versionLabels))
	for _, label := range versionLabels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		if s.applicationVersions[applicationName][label] == nil {
			return nil, ErrNotFound
		}
		versionSet[label] = struct{}{}
	}

	out := make([]Environment, 0, len(s.environmentsByID))
	for _, env := range s.environmentsByID {
		if env.ApplicationName != applicationName {
			continue
		}
		if len(versionSet) > 0 {
			if _, ok := versionSet[env.VersionLabel]; !ok {
				continue
			}
		}
		out = append(out, cloneEnvironment(env))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Service) CreatePlatformVersion(platformName, platformVersion, environmentName string, bundle S3Location, optionSettings []OptionSetting, tags []Tag) (PlatformVersion, error) {
	platformName = strings.TrimSpace(platformName)
	platformVersion = strings.TrimSpace(platformVersion)
	environmentName = strings.TrimSpace(environmentName)
	bundle.S3Bucket = strings.TrimSpace(bundle.S3Bucket)
	bundle.S3Key = strings.TrimSpace(bundle.S3Key)
	if platformName == "" || platformVersion == "" || bundle.S3Bucket == "" || bundle.S3Key == "" {
		return PlatformVersion{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	arn := platformVersionARN(platformName, platformVersion)
	if s.platformVersionsByARN[arn] != nil {
		return PlatformVersion{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	builderToken := normalizeCNAMEPrefix(environmentName)
	if builderToken == "" {
		builderToken = normalizeCNAMEPrefix(platformName)
	}
	if builderToken == "" {
		builderToken = fmt.Sprintf("platform-%d", atomic.AddUint64(&s.seq, 1))
	}
	pv := &PlatformVersion{
		PlatformARN:                  arn,
		PlatformName:                 platformName,
		PlatformVersion:              platformVersion,
		PlatformOwner:                DefaultAccountID,
		PlatformStatus:               "Ready",
		PlatformBranchName:           platformName,
		PlatformBranchLifecycleState: "Supported",
		PlatformLifecycleState:       "recommended",
		PlatformCategory:             "custom",
		SupportedTierList:            []string{"WebServer/Standard"},
		SupportedAddonList:           []string{"Log/S3", "Monitoring/Healthd"},
		OperatingSystemName:          "Amazon Linux 2",
		OperatingSystemVersion:       "2",
		Description:                  "Custom platform version",
		Maintainer:                   "stackyard",
		SolutionStackName:            "64bit Amazon Linux 2",
		BuilderARN:                   fmt.Sprintf("arn:aws:elasticbeanstalk:%s:%s:environment/Builder/%s", DefaultRegion, DefaultAccountID, builderToken),
		DateCreated:                  now,
		DateUpdated:                  now,
	}
	_ = optionSettings
	s.platformVersionsByARN[arn] = pv
	s.setTagsLocked(arn, tags)
	return clonePlatformVersion(pv), nil
}

func (s *Service) DeletePlatformVersion(platformARN string) (PlatformVersion, error) {
	platformARN = strings.TrimSpace(platformARN)
	if platformARN == "" {
		return PlatformVersion{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pv := s.platformVersionsByARN[platformARN]
	if pv == nil {
		return PlatformVersion{}, ErrNotFound
	}
	out := clonePlatformVersion(pv)
	out.PlatformStatus = "Deleted"
	delete(s.platformVersionsByARN, platformARN)
	delete(s.tagsByResourceARN, platformARN)
	return out, nil
}

func (s *Service) DescribePlatformVersion(platformARN string) (PlatformVersion, error) {
	platformARN = strings.TrimSpace(platformARN)
	if platformARN == "" {
		return PlatformVersion{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pv := s.platformVersionsByARN[platformARN]
	if pv == nil {
		return PlatformVersion{}, ErrNotFound
	}
	return clonePlatformVersion(pv), nil
}

func (s *Service) ListPlatformVersions(filters []PlatformFilter, maxRecords int, nextToken string) ([]PlatformVersion, string, error) {
	if maxRecords <= 0 {
		maxRecords = 100
	}
	offset := 0
	if nextToken != "" {
		n, err := strconv.Atoi(strings.TrimSpace(nextToken))
		if err != nil || n < 0 {
			return nil, "", ErrInvalidParameter
		}
		offset = n
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]PlatformVersion, 0, len(s.platformVersionsByARN))
	for _, pv := range s.platformVersionsByARN {
		item := clonePlatformVersion(pv)
		if platformVersionMatchesFilters(item, filters) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].PlatformName == items[j].PlatformName {
			return items[i].PlatformVersion < items[j].PlatformVersion
		}
		return items[i].PlatformName < items[j].PlatformName
	})
	if offset >= len(items) {
		return []PlatformVersion{}, "", nil
	}
	end := offset + maxRecords
	if end > len(items) {
		end = len(items)
	}
	out := make([]PlatformVersion, end-offset)
	copy(out, items[offset:end])
	if end < len(items) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *Service) ListPlatformBranches(filters []SearchFilter, maxRecords int, nextToken string) ([]PlatformBranch, string, error) {
	if maxRecords <= 0 {
		maxRecords = 100
	}
	offset := 0
	if nextToken != "" {
		n, err := strconv.Atoi(strings.TrimSpace(nextToken))
		if err != nil || n < 0 {
			return nil, "", ErrInvalidParameter
		}
		offset = n
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	branchesByKey := map[string]*PlatformBranch{}
	orderCounterByPlatform := map[string]int32{}
	for _, pv := range s.platformVersionsByARN {
		branchName := strings.TrimSpace(pv.PlatformBranchName)
		if branchName == "" {
			branchName = strings.TrimSpace(pv.PlatformName)
		}
		key := pv.PlatformName + "|" + branchName
		branch := branchesByKey[key]
		if branch == nil {
			orderCounterByPlatform[pv.PlatformName]++
			branch = &PlatformBranch{
				BranchName:        branchName,
				BranchOrder:       orderCounterByPlatform[pv.PlatformName],
				LifecycleState:    strings.ToLower(strings.TrimSpace(pv.PlatformBranchLifecycleState)),
				PlatformName:      pv.PlatformName,
				SupportedTierList: append([]string(nil), pv.SupportedTierList...),
			}
			if branch.LifecycleState == "" {
				branch.LifecycleState = "supported"
			}
			branchesByKey[key] = branch
		}
	}

	items := make([]PlatformBranch, 0, len(branchesByKey))
	for _, branch := range branchesByKey {
		item := clonePlatformBranch(branch)
		if platformBranchMatchesFilters(item, filters) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].PlatformName == items[j].PlatformName {
			return items[i].BranchOrder < items[j].BranchOrder
		}
		return items[i].PlatformName < items[j].PlatformName
	})
	if offset >= len(items) {
		return []PlatformBranch{}, "", nil
	}
	end := offset + maxRecords
	if end > len(items) {
		end = len(items)
	}
	out := make([]PlatformBranch, end-offset)
	copy(out, items[offset:end])
	if end < len(items) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *Service) DescribeEnvironmentHealth(environmentID, environmentName string) (EnvironmentHealth, error) {
	environmentID = strings.TrimSpace(environmentID)
	environmentName = strings.TrimSpace(environmentName)
	if environmentID == "" && environmentName == "" {
		return EnvironmentHealth{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(environmentID, environmentName)
	if env == nil {
		return EnvironmentHealth{}, ErrNotFound
	}
	refreshed := time.Now().UTC()
	summary := HealthInstanceSummary{Unknown: 1}
	switch strings.ToLower(env.Health) {
	case "green":
		summary = HealthInstanceSummary{Ok: 1}
	case "yellow":
		summary = HealthInstanceSummary{Warning: 1}
	case "red":
		summary = HealthInstanceSummary{Degraded: 1}
	case "grey":
		summary = HealthInstanceSummary{NoData: 1}
	}
	causes := []string{"Environment is healthy"}
	if strings.ToLower(env.HealthStatus) != "ok" {
		causes = []string{fmt.Sprintf("Environment health status is %s", env.HealthStatus)}
	}
	if strings.EqualFold(env.Status, "Terminated") {
		causes = []string{"Environment is terminated"}
	}
	return EnvironmentHealth{
		EnvironmentName: env.Name,
		Color:           env.Health,
		HealthStatus:    env.HealthStatus,
		Status:          env.Status,
		Causes:          causes,
		ApplicationMetrics: HealthApplicationMetrics{
			Duration:     10,
			RequestCount: 1,
			StatusCodes: MetricsStatusCodes{
				Status2xx: 100,
				Status3xx: 0,
				Status4xx: 0,
				Status5xx: 0,
			},
		},
		InstancesHealth: summary,
		RefreshedAt:     refreshed,
	}, nil
}

func (s *Service) DescribeInstancesHealth(environmentID, environmentName, nextToken string) ([]InstanceHealth, string, time.Time, error) {
	environmentID = strings.TrimSpace(environmentID)
	environmentName = strings.TrimSpace(environmentName)
	nextToken = strings.TrimSpace(nextToken)
	if environmentID == "" && environmentName == "" {
		return nil, "", time.Time{}, ErrInvalidParameter
	}
	if nextToken != "" {
		return []InstanceHealth{}, "", time.Now().UTC(), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(environmentID, environmentName)
	if env == nil {
		return nil, "", time.Time{}, ErrNotFound
	}
	instanceID := "i-" + fmt.Sprintf("%08x", hashToken(env.ID))
	item := InstanceHealth{
		InstanceID:       instanceID,
		AvailabilityZone: DefaultRegion + "a",
		Color:            env.Health,
		HealthStatus:     env.HealthStatus,
		InstanceType:     "t3.micro",
		LaunchedAt:       env.DateCreated,
		Causes:           []string{"Instance is healthy"},
		ApplicationMetrics: HealthApplicationMetrics{
			Duration:     10,
			RequestCount: 1,
			StatusCodes: MetricsStatusCodes{
				Status2xx: 100,
			},
		},
	}
	if strings.ToLower(env.HealthStatus) != "ok" {
		item.Causes = []string{fmt.Sprintf("Instance health status is %s", env.HealthStatus)}
	}
	return []InstanceHealth{item}, "", time.Now().UTC(), nil
}

func (s *Service) DeleteEnvironmentConfiguration(applicationName, environmentName string) error {
	applicationName = strings.TrimSpace(applicationName)
	environmentName = strings.TrimSpace(environmentName)
	if applicationName == "" || environmentName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked("", environmentName)
	if env == nil || env.ApplicationName != applicationName {
		return ErrNotFound
	}
	env.OptionSettings = []OptionSetting{}
	env.DateUpdated = time.Now().UTC()
	return nil
}

func (s *Service) DescribeEnvironmentManagedActions(environmentID, environmentName, status string) ([]ManagedAction, error) {
	environmentID = strings.TrimSpace(environmentID)
	environmentName = strings.TrimSpace(environmentName)
	status = strings.TrimSpace(status)
	if environmentID == "" && environmentName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(environmentID, environmentName)
	if env == nil {
		return nil, ErrNotFound
	}
	items := s.managedActionsByEnvID[env.ID]
	out := make([]ManagedAction, 0, len(items))
	for _, item := range items {
		if status != "" && !strings.EqualFold(item.Status, status) {
			continue
		}
		out = append(out, cloneManagedAction(item))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].WindowStartTime.Before(out[j].WindowStartTime) })
	return out, nil
}

func (s *Service) ApplyEnvironmentManagedAction(actionID, environmentID, environmentName string) (ManagedAction, error) {
	actionID = strings.TrimSpace(actionID)
	environmentID = strings.TrimSpace(environmentID)
	environmentName = strings.TrimSpace(environmentName)
	if actionID == "" || (environmentID == "" && environmentName == "") {
		return ManagedAction{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(environmentID, environmentName)
	if env == nil {
		return ManagedAction{}, ErrNotFound
	}
	items := s.managedActionsByEnvID[env.ID]
	idx := -1
	var selected ManagedAction
	for i, item := range items {
		if item.ActionID == actionID {
			idx = i
			selected = item
			break
		}
	}
	if idx < 0 {
		return ManagedAction{}, ErrNotFound
	}
	if !strings.EqualFold(selected.Status, "Scheduled") {
		return ManagedAction{}, ErrConflict
	}
	selected.Status = "Running"
	now := time.Now().UTC()
	history := ManagedActionHistoryItem{
		ActionID:          selected.ActionID,
		ActionDescription: selected.ActionDescription,
		ActionType:        selected.ActionType,
		ExecutedTime:      now,
		FinishedTime:      now,
		Status:            "Completed",
	}
	s.managedActionHistByEnv[env.ID] = append([]ManagedActionHistoryItem{history}, s.managedActionHistByEnv[env.ID]...)
	remaining := append([]ManagedAction(nil), items[:idx]...)
	remaining = append(remaining, items[idx+1:]...)
	s.managedActionsByEnvID[env.ID] = remaining
	return cloneManagedAction(selected), nil
}

func (s *Service) DescribeEnvironmentManagedActionHistory(environmentID, environmentName string, maxItems int, nextToken string) ([]ManagedActionHistoryItem, string, error) {
	environmentID = strings.TrimSpace(environmentID)
	environmentName = strings.TrimSpace(environmentName)
	nextToken = strings.TrimSpace(nextToken)
	if environmentID == "" && environmentName == "" {
		return nil, "", ErrInvalidParameter
	}
	if maxItems <= 0 {
		maxItems = 100
	}
	offset := 0
	if nextToken != "" {
		n, err := strconv.Atoi(nextToken)
		if err != nil || n < 0 {
			return nil, "", ErrInvalidParameter
		}
		offset = n
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	env := s.environmentBySelectorLocked(environmentID, environmentName)
	if env == nil {
		return nil, "", ErrNotFound
	}
	items := s.managedActionHistByEnv[env.ID]
	if offset >= len(items) {
		return []ManagedActionHistoryItem{}, "", nil
	}
	end := offset + maxItems
	if end > len(items) {
		end = len(items)
	}
	out := make([]ManagedActionHistoryItem, end-offset)
	for i := range out {
		out[i] = cloneManagedActionHistoryItem(items[offset+i])
	}
	if end < len(items) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *Service) UpdateApplicationResourceLifecycle(applicationName string, cfg ApplicationResourceLifecycleConfig) (ApplicationResourceLifecycleConfig, error) {
	applicationName = strings.TrimSpace(applicationName)
	if applicationName == "" {
		return ApplicationResourceLifecycleConfig{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.applications[applicationName] == nil {
		return ApplicationResourceLifecycleConfig{}, ErrNotFound
	}
	existing := cloneApplicationResourceLifecycleConfig(s.appResourceLifecycle[applicationName])
	normalized := cloneApplicationResourceLifecycleConfig(cfg)
	if normalized.ServiceRole == "" {
		normalized.ServiceRole = existing.ServiceRole
	}
	if normalized.VersionLifecycleConfig == nil {
		normalized.VersionLifecycleConfig = existing.VersionLifecycleConfig
	} else if existing.VersionLifecycleConfig != nil {
		if normalized.VersionLifecycleConfig.MaxAgeRule == nil {
			normalized.VersionLifecycleConfig.MaxAgeRule = existing.VersionLifecycleConfig.MaxAgeRule
		}
		if normalized.VersionLifecycleConfig.MaxCountRule == nil {
			normalized.VersionLifecycleConfig.MaxCountRule = existing.VersionLifecycleConfig.MaxCountRule
		}
	}
	s.appResourceLifecycle[applicationName] = normalized
	return cloneApplicationResourceLifecycleConfig(normalized), nil
}

func (s *Service) ListTagsForResource(resourceARN string) ([]Tag, error) {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tags := s.tagsByResourceARN[resourceARN]
	if tags == nil {
		return []Tag{}, nil
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]Tag, 0, len(keys))
	for _, key := range keys {
		out = append(out, Tag{Key: key, Value: tags[key]})
	}
	return out, nil
}

func (s *Service) UpdateTagsForResource(resourceARN string, tagsToAdd []Tag, tagsToRemove []string) error {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tags := s.tagsByResourceARN[resourceARN]
	if tags == nil {
		tags = map[string]string{}
		s.tagsByResourceARN[resourceARN] = tags
	}
	for _, tag := range tagsToAdd {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			continue
		}
		tags[key] = tag.Value
	}
	for _, key := range tagsToRemove {
		delete(tags, strings.TrimSpace(key))
	}
	return nil
}

func (s *Service) environmentBySelectorLocked(environmentID, environmentName string) *Environment {
	if environmentID != "" {
		if env := s.environmentsByID[environmentID]; env != nil {
			return env
		}
	}
	if environmentName != "" {
		id := s.environmentIDByName[environmentName]
		if id != "" {
			return s.environmentsByID[id]
		}
	}
	return nil
}

func (s *Service) appendEventLocked(event Event) {
	s.events = append(s.events, event)
}

func (s *Service) nextRequestIDLocked() string {
	n := atomic.AddUint64(&s.seq, 1)
	return fmt.Sprintf("request-%d", n)
}

func (s *Service) setTagsLocked(resourceARN string, tags []Tag) {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return
	}
	if len(tags) == 0 {
		if _, ok := s.tagsByResourceARN[resourceARN]; !ok {
			s.tagsByResourceARN[resourceARN] = map[string]string{}
		}
		return
	}
	m := map[string]string{}
	for _, tag := range tags {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			continue
		}
		m[key] = tag.Value
	}
	s.tagsByResourceARN[resourceARN] = m
}

func cloneApplication(in *Application) Application {
	if in == nil {
		return Application{}
	}
	out := *in
	out.Versions = append([]string(nil), in.Versions...)
	out.ConfigurationTemplates = append([]string(nil), in.ConfigurationTemplates...)
	return out
}

func cloneApplicationVersion(in *ApplicationVersion) ApplicationVersion {
	if in == nil {
		return ApplicationVersion{}
	}
	return *in
}

func cloneEnvironment(in *Environment) Environment {
	if in == nil {
		return Environment{}
	}
	out := *in
	out.OptionSettings = cloneOptionSettings(in.OptionSettings)
	return out
}

func cloneConfigurationTemplate(in *ConfigurationTemplate) ConfigurationTemplate {
	if in == nil {
		return ConfigurationTemplate{}
	}
	out := *in
	out.OptionSettings = cloneOptionSettings(in.OptionSettings)
	return out
}

func cloneOptionSettings(in []OptionSetting) []OptionSetting {
	if len(in) == 0 {
		return []OptionSetting{}
	}
	out := make([]OptionSetting, len(in))
	copy(out, in)
	return out
}

func cloneResourceQuotas(in ResourceQuotas) ResourceQuotas {
	return ResourceQuotas{
		ApplicationQuota:           in.ApplicationQuota,
		ApplicationVersionQuota:    in.ApplicationVersionQuota,
		ConfigurationTemplateQuota: in.ConfigurationTemplateQuota,
		CustomPlatformQuota:        in.CustomPlatformQuota,
		EnvironmentQuota:           in.EnvironmentQuota,
	}
}

func cloneManagedAction(in ManagedAction) ManagedAction {
	return ManagedAction{
		ActionID:          in.ActionID,
		ActionDescription: in.ActionDescription,
		ActionType:        in.ActionType,
		Status:            in.Status,
		WindowStartTime:   in.WindowStartTime,
	}
}

func cloneManagedActionHistoryItem(in ManagedActionHistoryItem) ManagedActionHistoryItem {
	return ManagedActionHistoryItem{
		ActionID:           in.ActionID,
		ActionDescription:  in.ActionDescription,
		ActionType:         in.ActionType,
		ExecutedTime:       in.ExecutedTime,
		FinishedTime:       in.FinishedTime,
		Status:             in.Status,
		FailureType:        in.FailureType,
		FailureDescription: in.FailureDescription,
	}
}

func cloneApplicationResourceLifecycleConfig(in ApplicationResourceLifecycleConfig) ApplicationResourceLifecycleConfig {
	out := ApplicationResourceLifecycleConfig{
		ServiceRole: strings.TrimSpace(in.ServiceRole),
	}
	if in.VersionLifecycleConfig != nil {
		cfg := &ApplicationVersionLifecycleConfig{}
		if in.VersionLifecycleConfig.MaxAgeRule != nil {
			rule := *in.VersionLifecycleConfig.MaxAgeRule
			cfg.MaxAgeRule = &rule
		}
		if in.VersionLifecycleConfig.MaxCountRule != nil {
			rule := *in.VersionLifecycleConfig.MaxCountRule
			cfg.MaxCountRule = &rule
		}
		out.VersionLifecycleConfig = cfg
	}
	return out
}

func clonePlatformVersion(in *PlatformVersion) PlatformVersion {
	if in == nil {
		return PlatformVersion{}
	}
	out := *in
	out.SupportedTierList = append([]string(nil), in.SupportedTierList...)
	out.SupportedAddonList = append([]string(nil), in.SupportedAddonList...)
	return out
}

func clonePlatformBranch(in *PlatformBranch) PlatformBranch {
	if in == nil {
		return PlatformBranch{}
	}
	out := *in
	out.SupportedTierList = append([]string(nil), in.SupportedTierList...)
	return out
}

func platformVersionMatchesFilters(item PlatformVersion, filters []PlatformFilter) bool {
	for _, filter := range filters {
		if !platformVersionMatchesFilter(item, filter) {
			return false
		}
	}
	return true
}

func platformVersionMatchesFilter(item PlatformVersion, filter PlatformFilter) bool {
	if len(filter.Values) == 0 {
		return true
	}
	field, ok := platformFilterField(item, filter.Type)
	if !ok {
		return false
	}
	value := strings.TrimSpace(filter.Values[0])
	switch strings.TrimSpace(filter.Operator) {
	case "", "=":
		return field == value
	case "!=":
		return field != value
	case "contains":
		return strings.Contains(field, value)
	case "begins_with":
		return strings.HasPrefix(field, value)
	case "ends_with":
		return strings.HasSuffix(field, value)
	default:
		return false
	}
}

func platformFilterField(item PlatformVersion, filterType string) (string, bool) {
	switch strings.TrimSpace(filterType) {
	case "PlatformName":
		return item.PlatformName, true
	case "PlatformVersion":
		return item.PlatformVersion, true
	case "PlatformStatus":
		return item.PlatformStatus, true
	case "PlatformBranchName":
		return item.PlatformBranchName, true
	case "PlatformLifecycleState":
		return item.PlatformLifecycleState, true
	case "PlatformOwner":
		return item.PlatformOwner, true
	case "SupportedTier":
		if len(item.SupportedTierList) > 0 {
			return item.SupportedTierList[0], true
		}
		return "", true
	case "SupportedAddon":
		if len(item.SupportedAddonList) > 0 {
			return item.SupportedAddonList[0], true
		}
		return "", true
	case "OperatingSystemName":
		return item.OperatingSystemName, true
	default:
		return "", false
	}
}

func platformBranchMatchesFilters(item PlatformBranch, filters []SearchFilter) bool {
	for _, filter := range filters {
		if !platformBranchMatchesFilter(item, filter) {
			return false
		}
	}
	return true
}

func platformBranchMatchesFilter(item PlatformBranch, filter SearchFilter) bool {
	if len(filter.Values) == 0 {
		return true
	}
	field, ok := platformBranchFilterField(item, filter.Attribute)
	if !ok {
		return false
	}
	return matchFilterOperator(field, filter.Operator, filter.Values)
}

func platformBranchFilterField(item PlatformBranch, attribute string) (string, bool) {
	switch strings.TrimSpace(attribute) {
	case "BranchName":
		return item.BranchName, true
	case "LifecycleState":
		return strings.ToLower(item.LifecycleState), true
	case "PlatformName":
		return item.PlatformName, true
	case "TierType":
		if len(item.SupportedTierList) > 0 {
			return item.SupportedTierList[0], true
		}
		return "", true
	default:
		return "", false
	}
}

func matchFilterOperator(field, operator string, values []string) bool {
	field = strings.TrimSpace(field)
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	switch strings.TrimSpace(operator) {
	case "", "=":
		return field == values[0]
	case "!=":
		return field != values[0]
	case "contains":
		return strings.Contains(field, values[0])
	case "begins_with":
		return strings.HasPrefix(field, values[0])
	case "ends_with":
		return strings.HasSuffix(field, values[0])
	case "in":
		for _, v := range values {
			if field == v {
				return true
			}
		}
		return false
	case "not_in":
		for _, v := range values {
			if field == v {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func normalizeCNAMEPrefix(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	b := strings.Builder{}
	lastDash := false
	for _, r := range v {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 50 {
		out = out[:50]
	}
	return out
}

func uniqueCNAMELocked(environments map[string]*Environment, prefix string) string {
	candidate := prefix + ".elasticbeanstalk.local"
	if !cnameInUse(environments, candidate) {
		return candidate
	}
	for i := 2; i < 5000; i++ {
		candidate = fmt.Sprintf("%s-%d.elasticbeanstalk.local", prefix, i)
		if !cnameInUse(environments, candidate) {
			return candidate
		}
	}
	return fmt.Sprintf("%s-%d.elasticbeanstalk.local", prefix, time.Now().UnixNano())
}

func cnameInUse(environments map[string]*Environment, cname string) bool {
	for _, env := range environments {
		if env.CNAME == cname && env.Status != "Terminated" {
			return true
		}
	}
	return false
}

func normalizeResourceToken(v string) string {
	v = normalizeCNAMEPrefix(v)
	if v == "" {
		return ""
	}
	return strings.ReplaceAll(v, "-", "")
}

func hashToken(v string) uint32 {
	var h uint32
	for _, r := range v {
		h = h*33 + uint32(r)
	}
	if h == 0 {
		h = 1
	}
	return h
}

func applicationARN(name string) string {
	return fmt.Sprintf("arn:aws:elasticbeanstalk:%s:%s:application/%s", DefaultRegion, DefaultAccountID, name)
}

func applicationVersionARN(applicationName, versionLabel string) string {
	return fmt.Sprintf("arn:aws:elasticbeanstalk:%s:%s:applicationversion/%s/%s", DefaultRegion, DefaultAccountID, applicationName, versionLabel)
}

func environmentARN(applicationName, environmentName string) string {
	return fmt.Sprintf("arn:aws:elasticbeanstalk:%s:%s:environment/%s/%s", DefaultRegion, DefaultAccountID, applicationName, environmentName)
}

func configurationTemplateARN(applicationName, templateName string) string {
	return fmt.Sprintf("arn:aws:elasticbeanstalk:%s:%s:configurationtemplate/%s/%s", DefaultRegion, DefaultAccountID, applicationName, templateName)
}

func platformVersionARN(platformName, platformVersion string) string {
	return fmt.Sprintf("arn:aws:elasticbeanstalk:%s:%s:platform/%s/%s", DefaultRegion, DefaultAccountID, platformName, platformVersion)
}
