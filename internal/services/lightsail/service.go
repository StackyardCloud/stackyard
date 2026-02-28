package lightsail

import (
	"encoding/base64"
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
	DefaultKeyPair   = "LightsailDefaultKeyPair"
)

type Instance struct {
	Name             string
	ARN              string
	BlueprintID      string
	BundleID         string
	IPAddressType    string
	AvailabilityZone string
	Region           string
	PublicIPAddress  string
	PrivateIPAddress string
	StateCode        int32
	StateName        string
	CreatedAt        time.Time
	IsStaticIP       bool
	Username         string
	IPv6Addresses    []string
	PortStates       []InstancePortState
	MetadataOptions  InstanceMetadataOptions
	HostKeys         []HostKeyAttributes
	Tags             map[string]string
}

type PortInfo struct {
	FromPort        int32
	ToPort          int32
	Protocol        string
	Cidrs           []string
	Ipv6Cidrs       []string
	CidrListAliases []string
}

type InstancePortState struct {
	PortInfo
	State string
}

type HostKeyAttributes struct {
	Algorithm         string
	FingerprintSHA1   string
	FingerprintSHA256 string
	PublicKey         string
	WitnessedAt       time.Time
	NotValidBefore    time.Time
	NotValidAfter     time.Time
}

type InstanceAccessDetails struct {
	InstanceName  string
	Protocol      string
	Username      string
	IpAddress     string
	Ipv6Addresses []string
	PrivateKey    string
	CertKey       string
	Password      string
	ExpiresAt     time.Time
	HostKeys      []HostKeyAttributes
}

type InstanceMetadataOptions struct {
	HttpEndpoint            string
	HttpProtocolIpv6        string
	HttpPutResponseHopLimit int32
	HttpTokens              string
	State                   string
}

type Blueprint struct {
	AppCategory string
	BlueprintID string
	Description string
	Group       string
	IsActive    bool
	LicenseURL  string
	MinPower    int32
	Name        string
	Platform    string
	ProductURL  string
	Type        string
	Version     string
	VersionCode string
}

type Bundle struct {
	AppCategory            string
	BundleID               string
	CPUCount               int32
	DiskSizeInGb           int32
	InstanceType           string
	IsActive               bool
	Name                   string
	Power                  int32
	Price                  float32
	PublicIpv4AddressCount int32
	RAMSizeInGb            float32
	SupportedAppCategories []string
	SupportedPlatforms     []string
	TransferPerMonthInGb   int32
}

type BlueprintsPage struct {
	Blueprints    []Blueprint
	NextPageToken string
}

type BundlesPage struct {
	Bundles       []Bundle
	NextPageToken string
}

type SetupExecutionDetail struct {
	Command        string
	DateTime       time.Time
	Name           string
	StandardError  string
	StandardOutput string
	Status         string
	Version        string
}

type SetupRequest struct {
	CertificateProvider string
	DomainNames         []string
	InstanceName        string
}

type SetupHistoryResource struct {
	ARN              string
	CreatedAt        time.Time
	AvailabilityZone string
	Region           string
	Name             string
	ResourceType     string
}

type SetupHistory struct {
	ExecutionDetails []SetupExecutionDetail
	OperationID      string
	Request          *SetupRequest
	Resource         SetupHistoryResource
	Status           string
}

type SetupHistoryPage struct {
	SetupHistory  []SetupHistory
	NextPageToken string
}

type EstimateByTime struct {
	Currency    string
	PricingUnit string
	StartTime   time.Time
	EndTime     time.Time
	Unit        float64
	UsageCost   float64
}

type CostEstimate struct {
	UsageType     string
	ResultsByTime []EstimateByTime
}

type ResourceBudgetEstimate struct {
	ResourceName  string
	ResourceType  string
	StartTime     time.Time
	EndTime       time.Time
	CostEstimates []CostEstimate
}

type CloudFormationDestinationInfo struct {
	ID      string
	Service string
}

type CloudFormationStackSourceInfo struct {
	ARN          string
	Name         string
	ResourceType string
}

type CloudFormationStackRecord struct {
	ARN              string
	CreatedAt        time.Time
	DestinationInfo  CloudFormationDestinationInfo
	AvailabilityZone string
	Region           string
	Name             string
	ResourceType     string
	SourceInfo       []CloudFormationStackSourceInfo
	State            string
}

type CloudFormationStackRecordsPage struct {
	CloudFormationStackRecords []CloudFormationStackRecord
	NextPageToken              string
}

type GUISession struct {
	IsPrimary bool
	Name      string
	URL       string
}

type GUISessionAccessDetails struct {
	FailureReason      string
	PercentageComplete int32
	ResourceName       string
	Sessions           []GUISession
	Status             string
}

type InstanceEntry struct {
	AvailabilityZone string
	InstanceType     string
	PortInfoSource   string
	SourceName       string
	UserData         string
}

type InstanceMetricInput struct {
	InstanceName string
	EndTime      time.Time
	MetricName   string
	Period       int32
	StartTime    time.Time
	Statistics   []string
	Unit         string
}

type InstanceMetricDatapoint struct {
	Average     *float64
	Maximum     *float64
	Minimum     *float64
	SampleCount *float64
	Sum         *float64
	Timestamp   time.Time
	Unit        string
}

type InstanceSnapshot struct {
	Name             string
	ARN              string
	FromInstanceName string
	FromInstanceARN  string
	FromBlueprintID  string
	FromBundleID     string
	AvailabilityZone string
	Region           string
	State            string
	CreatedAt        time.Time
	Tags             map[string]string
}

type StaticIP struct {
	Name             string
	ARN              string
	IPAddress        string
	AttachedTo       string
	AvailabilityZone string
	Region           string
	CreatedAt        time.Time
	Tags             map[string]string
}

type Disk struct {
	Name             string
	ARN              string
	AvailabilityZone string
	Region           string
	CreatedAt        time.Time
	SizeInGb         int32
	Iops             int32
	IsAttached       bool
	AttachedTo       string
	Path             string
	State            string
	AutoMountStatus  string
	IsSystemDisk     bool
	Tags             map[string]string
}

type DiskSnapshot struct {
	Name               string
	ARN                string
	AvailabilityZone   string
	Region             string
	CreatedAt          time.Time
	FromDiskARN        string
	FromDiskName       string
	FromInstanceARN    string
	FromInstanceName   string
	IsFromAutoSnapshot bool
	Progress           string
	SizeInGb           int32
	State              string
	SupportCode        string
	Tags               map[string]string
}

type ExportSnapshotRecord struct {
	Name               string
	ARN                string
	AvailabilityZone   string
	Region             string
	CreatedAt          time.Time
	DestinationID      string
	DestinationService string
	SourceSnapshotARN  string
	SourceSnapshotName string
	SourceCreatedAt    time.Time
	SourceResourceARN  string
	SourceResourceName string
	SourceType         string
	SourceDiskSizeInGb int32
	State              string
}

type AlarmMonitoredResourceInfo struct {
	ARN          string
	Name         string
	ResourceType string
}

type Alarm struct {
	Name                 string
	ARN                  string
	ComparisonOperator   string
	ContactProtocols     []string
	CreatedAt            time.Time
	DatapointsToAlarm    int32
	EvaluationPeriods    int32
	MetricName           string
	MonitoredResource    AlarmMonitoredResourceInfo
	NotificationEnabled  bool
	NotificationTriggers []string
	Period               int32
	ResourceType         string
	State                string
	Statistic            string
	SupportCode          string
	Threshold            float64
	TreatMissingData     string
	Unit                 string
	AvailabilityZone     string
	Region               string
}

type AutoSnapshotAttachedDisk struct {
	Path     string
	SizeInGb int32
}

type AutoSnapshotDetails struct {
	CreatedAt         time.Time
	Date              string
	FromAttachedDisks []AutoSnapshotAttachedDisk
	Status            string
}

type AddOn struct {
	Name                  string
	Status                string
	SnapshotTimeOfDay     string
	NextSnapshotTimeOfDay string
}

type LoadBalancerInstanceHealthSummary struct {
	InstanceName         string
	InstanceHealth       string
	InstanceHealthReason string
}

type LoadBalancerTLSCertificateSummary struct {
	Name       string
	IsAttached bool
}

type LoadBalancer struct {
	Name                    string
	ARN                     string
	CreatedAt               time.Time
	DNSName                 string
	HealthCheckPath         string
	HTTPSRedirectionEnabled bool
	InstanceHealthSummary   []LoadBalancerInstanceHealthSummary
	InstancePort            int32
	IPAddressType           string
	AvailabilityZone        string
	Region                  string
	Protocol                string
	PublicPorts             []int32
	ResourceType            string
	State                   string
	SupportCode             string
	Tags                    map[string]string
	TLSCertificateSummaries []LoadBalancerTLSCertificateSummary
	TLSPolicyName           string
	ConfigurationOptions    map[string]string
	AttachedInstances       []string
}

type LoadBalancerMetricInput struct {
	LoadBalancerName string
	EndTime          time.Time
	MetricName       string
	Period           int32
	StartTime        time.Time
	Statistics       []string
	Unit             string
}

type LoadBalancerMetricDatapoint struct {
	Average     *float64
	Maximum     *float64
	Minimum     *float64
	SampleCount *float64
	Sum         *float64
	Timestamp   time.Time
	Unit        string
}

type LoadBalancersPage struct {
	LoadBalancers []LoadBalancer
	NextPageToken string
}

type LoadBalancerTLSCertificate struct {
	Name                    string
	ARN                     string
	CreatedAt               time.Time
	DomainName              string
	LoadBalancerName        string
	AvailabilityZone        string
	Region                  string
	IsAttached              bool
	IssuedAt                time.Time
	Issuer                  string
	KeyAlgorithm            string
	NotAfter                time.Time
	NotBefore               time.Time
	ResourceType            string
	Status                  string
	Subject                 string
	SubjectAlternativeNames []string
	SupportCode             string
	Tags                    map[string]string
}

type LoadBalancerTLSPolicy struct {
	Name        string
	Description string
	IsDefault   bool
	Ciphers     []string
	Protocols   []string
}

type CertificateDomainValidationRecord struct {
	DomainName            string
	ValidationStatus      string
	DNSRecordCreationCode string
	DNSRecordCreationText string
	ResourceRecordName    string
	ResourceRecordType    string
	ResourceRecordValue   string
}

type CertificateRenewalSummary struct {
	DomainValidationRecords []CertificateDomainValidationRecord
	RenewalStatus           string
	RenewalStatusReason     string
	UpdatedAt               time.Time
}

type Certificate struct {
	Name                    string
	ARN                     string
	CreatedAt               time.Time
	DomainName              string
	DomainValidationRecords []CertificateDomainValidationRecord
	EligibleToRenew         string
	InUseResourceCount      int32
	IssuedAt                time.Time
	IssuerCA                string
	KeyAlgorithm            string
	NotAfter                time.Time
	NotBefore               time.Time
	RenewalSummary          CertificateRenewalSummary
	RequestFailureReason    string
	RevocationReason        string
	RevokedAt               time.Time
	SerialNumber            string
	Status                  string
	SubjectAlternativeNames []string
	SupportCode             string
	Tags                    map[string]string
	AttachedDistributions   []string
}

type DomainEntry struct {
	ID      string
	IsAlias bool
	Name    string
	Options map[string]string
	Target  string
	Type    string
}

type Domain struct {
	Name             string
	ARN              string
	CreatedAt        time.Time
	DomainEntries    []DomainEntry
	AvailabilityZone string
	Region           string
	ResourceType     string
	SupportCode      string
	Tags             map[string]string
}

type DistributionCacheBehavior struct {
	Behavior string
}

type DistributionCacheBehaviorPerPath struct {
	Behavior string
	Path     string
}

type DistributionCookieObject struct {
	Option           string
	CookiesAllowList []string
}

type DistributionHeaderObject struct {
	Option           string
	HeadersAllowList []string
}

type DistributionQueryStringObject struct {
	Option                *bool
	QueryStringsAllowList []string
}

type DistributionCacheSettings struct {
	AllowedHTTPMethods    string
	CachedHTTPMethods     string
	DefaultTTL            int64
	ForwardedCookies      DistributionCookieObject
	ForwardedHeaders      DistributionHeaderObject
	ForwardedQueryStrings DistributionQueryStringObject
	MaximumTTL            int64
	MinimumTTL            int64
}

type DistributionOrigin struct {
	Name            string
	ProtocolPolicy  string
	RegionName      string
	ResourceType    string
	ResponseTimeout int32
}

type Distribution struct {
	AbleToUpdateBundle              bool
	AlternativeDomainNames          []string
	ARN                             string
	BundleID                        string
	CacheBehaviorSettings           DistributionCacheSettings
	CacheBehaviors                  []DistributionCacheBehaviorPerPath
	CertificateName                 string
	CreatedAt                       time.Time
	DefaultCacheBehavior            DistributionCacheBehavior
	DomainName                      string
	IPAddressType                   string
	IsEnabled                       bool
	Name                            string
	Origin                          DistributionOrigin
	OriginPublicDNS                 string
	ResourceType                    string
	Status                          string
	SupportCode                     string
	Tags                            map[string]string
	ViewerMinimumTLSProtocolVersion string
	AvailabilityZone                string
	Region                          string
}

type DistributionBundle struct {
	BundleID             string
	IsActive             bool
	Name                 string
	Price                float32
	TransferPerMonthInGb int32
}

type DistributionMetricDatapoint struct {
	Average     *float64
	Maximum     *float64
	Minimum     *float64
	SampleCount *float64
	Sum         *float64
	Timestamp   time.Time
	Unit        string
}

type DistributionCacheReset struct {
	CreateTime time.Time
	Status     string
}

type DistributionCreateInput struct {
	BundleID                        string
	DefaultCacheBehavior            DistributionCacheBehavior
	DistributionName                string
	Origin                          DistributionOrigin
	CacheBehaviorSettings           DistributionCacheSettings
	CacheBehaviors                  []DistributionCacheBehaviorPerPath
	CertificateName                 string
	IPAddressType                   string
	Tags                            map[string]string
	ViewerMinimumTLSProtocolVersion string
}

type DistributionUpdateInput struct {
	DistributionName                string
	CacheBehaviorSettings           *DistributionCacheSettings
	CacheBehaviors                  []DistributionCacheBehaviorPerPath
	HasCacheBehaviors               bool
	CertificateName                 *string
	DefaultCacheBehavior            *DistributionCacheBehavior
	IsEnabled                       *bool
	Origin                          *DistributionOrigin
	UseDefaultCertificate           *bool
	ViewerMinimumTLSProtocolVersion string
}

type DistributionMetricInput struct {
	DistributionName string
	EndTime          time.Time
	MetricName       string
	Period           int32
	StartTime        time.Time
	Statistics       []string
	Unit             string
}

type KeyPair struct {
	Name             string
	ARN              string
	Fingerprint      string
	AvailabilityZone string
	Region           string
	CreatedAt        time.Time
	PublicKeyBase64  string
	PrivateKeyBase64 string
	IsDefault        bool
	Tags             map[string]string
}

type BucketAccessLogConfig struct {
	Enabled     bool
	Destination string
	Prefix      string
}

type BucketAccessRules struct {
	AllowPublicOverrides bool
	GetObject            string
}

type BucketResourceReceivingAccess struct {
	Name         string
	ResourceType string
}

type BucketState struct {
	Code    string
	Message string
}

type Bucket struct {
	Name                     string
	ARN                      string
	BundleID                 string
	CreatedAt                time.Time
	AvailabilityZone         string
	Region                   string
	AbleToUpdateBundle       bool
	ObjectVersioning         string
	ReadonlyAccessAccounts   []string
	AccessRules              *BucketAccessRules
	AccessLogConfig          *BucketAccessLogConfig
	ResourcesReceivingAccess []BucketResourceReceivingAccess
	ResourceType             string
	State                    BucketState
	SupportCode              string
	URL                      string
	Tags                     map[string]string
}

type BucketBundle struct {
	BundleID             string
	IsActive             bool
	Name                 string
	Price                float32
	StoragePerMonthInGb  int32
	TransferPerMonthInGb int32
}

type BucketUpdateInput struct {
	BucketName                string
	AccessLogConfig           *BucketAccessLogConfig
	AccessRules               *BucketAccessRules
	ReadonlyAccessAccounts    []string
	HasReadonlyAccessAccounts bool
	Versioning                *string
}

type BucketMetricInput struct {
	BucketName string
	EndTime    time.Time
	MetricName string
	Period     int32
	StartTime  time.Time
	Statistics []string
	Unit       string
}

type BucketMetricDatapoint struct {
	Average     *float64
	Maximum     *float64
	Minimum     *float64
	SampleCount *float64
	Sum         *float64
	Timestamp   time.Time
	Unit        string
}

type ContainerService struct {
	Name              string
	ARN               string
	CreatedAt         time.Time
	AvailabilityZone  string
	Region            string
	Power             string
	PowerID           string
	Scale             int32
	IsDisabled        bool
	PrincipalARN      string
	PrivateDomainName string
	PublicDomainNames map[string][]string
	ResourceType      string
	State             string
	SupportCode       string
	URL               string
	Tags              map[string]string
}

type ContainerServiceUpdateInput struct {
	ServiceName          string
	Power                *string
	Scale                *int32
	IsDisabled           *bool
	PublicDomainNames    map[string][]string
	HasPublicDomainNames bool
}

type ContainerServiceMetricInput struct {
	ServiceName string
	EndTime     time.Time
	MetricName  string
	Period      int32
	StartTime   time.Time
	Statistics  []string
}

type ContainerServiceMetricDatapoint struct {
	Average     *float64
	Maximum     *float64
	Minimum     *float64
	SampleCount *float64
	Sum         *float64
	Timestamp   time.Time
	Unit        string
}

type ContainerServicePower struct {
	Name        string
	PowerID     string
	CPUCount    float32
	RAMSizeInGb float32
	Price       float32
	IsActive    bool
}

type ContainerServiceRegistryLogin struct {
	ExpiresAt time.Time
	Password  string
	Registry  string
	Username  string
}

type ContainerServiceContainer struct {
	Command     []string
	Environment map[string]string
	Image       string
	Ports       map[string]string
}

type ContainerServiceHealthCheckConfig struct {
	HealthyThreshold   *int32
	IntervalSeconds    *int32
	Path               *string
	SuccessCodes       *string
	TimeoutSeconds     *int32
	UnhealthyThreshold *int32
}

type ContainerServiceEndpoint struct {
	ContainerName string
	ContainerPort int32
	HealthCheck   *ContainerServiceHealthCheckConfig
}

type ContainerServiceDeployment struct {
	Containers     map[string]ContainerServiceContainer
	CreatedAt      time.Time
	PublicEndpoint *ContainerServiceEndpoint
	State          string
	Version        int32
}

type ContainerImage struct {
	CreatedAt time.Time
	Digest    string
	Image     string
}

type ContainerServiceLogEvent struct {
	CreatedAt time.Time
	Message   string
}

type ContainerLogInput struct {
	ServiceName   string
	ContainerName string
	StartTime     *time.Time
	EndTime       *time.Time
	FilterPattern string
	PageToken     string
}

type RelationalDatabase struct {
	Name                         string
	ARN                          string
	CreatedAt                    time.Time
	AvailabilityZone             string
	Region                       string
	BlueprintID                  string
	BundleID                     string
	Engine                       string
	EngineVersion                string
	MasterDatabaseName           string
	MasterUsername               string
	MasterUserPassword           string
	MasterUserPasswordCreatedAt  time.Time
	PreviousMasterUserPassword   string
	PreviousMasterPasswordAt     time.Time
	PendingMasterUserPassword    string
	PendingMasterPasswordAt      time.Time
	MasterEndpointAddress        string
	MasterEndpointPort           int32
	CPUCount                     int32
	DiskSizeInGb                 int32
	RAMSizeInGb                  float32
	LatestRestorableTime         time.Time
	PreferredBackupWindow        string
	PreferredMaintenanceWindow   string
	PubliclyAccessible           bool
	BackupRetentionEnabled       bool
	CACertificateIdentifier      string
	ParameterApplyStatus         string
	SecondaryAvailabilityZone    string
	State                        string
	SupportCode                  string
	PendingModifiedValues        *PendingModifiedRelationalDatabaseValues
	PendingMaintenanceActions    []string
	PendingMaintenanceActionCode string
	Tags                         map[string]string
}

type PendingModifiedRelationalDatabaseValues struct {
	BackupRetentionEnabled *bool
	EngineVersion          *string
	MasterUserPassword     *string
}

type RelationalDatabaseCreateInput struct {
	RelationalDatabaseName        string
	AvailabilityZone              string
	MasterDatabaseName            string
	MasterUsername                string
	MasterUserPassword            string
	RelationalDatabaseBlueprintID string
	RelationalDatabaseBundleID    string
	PreferredBackupWindow         string
	PreferredMaintenanceWindow    string
	PubliclyAccessible            *bool
	Tags                          map[string]string
}

type RelationalDatabaseUpdateInput struct {
	RelationalDatabaseName        string
	ApplyImmediately              *bool
	CACertificateIdentifier       *string
	DisableBackupRetention        *bool
	EnableBackupRetention         *bool
	MasterUserPassword            *string
	PreferredBackupWindow         *string
	PreferredMaintenanceWindow    *string
	PubliclyAccessible            *bool
	RelationalDatabaseBlueprintID *string
	RotateMasterUserPassword      *bool
}

type RelationalDatabaseDeleteInput struct {
	RelationalDatabaseName              string
	FinalRelationalDatabaseSnapshotName string
	SkipFinalSnapshot                   *bool
}

type RelationalDatabasesPage struct {
	RelationalDatabases []RelationalDatabase
	NextPageToken       string
}

type RelationalDatabaseBlueprint struct {
	BlueprintID              string
	Engine                   string
	EngineDescription        string
	EngineVersion            string
	EngineVersionDescription string
	IsEngineDefault          bool
}

type RelationalDatabaseBlueprintsPage struct {
	Blueprints    []RelationalDatabaseBlueprint
	NextPageToken string
}

type RelationalDatabaseBundle struct {
	BundleID             string
	CPUCount             int32
	DiskSizeInGb         int32
	IsActive             bool
	IsEncrypted          bool
	Name                 string
	Price                float32
	RAMSizeInGb          float32
	TransferPerMonthInGb int32
}

type RelationalDatabaseBundlesPage struct {
	Bundles       []RelationalDatabaseBundle
	NextPageToken string
}

type RelationalDatabaseSnapshot struct {
	Name                              string
	ARN                               string
	CreatedAt                         time.Time
	Engine                            string
	EngineVersion                     string
	FromRelationalDatabaseARN         string
	FromRelationalDatabaseBlueprintID string
	FromRelationalDatabaseBundleID    string
	FromRelationalDatabaseName        string
	AvailabilityZone                  string
	Region                            string
	SizeInGb                          int32
	State                             string
	SupportCode                       string
	Tags                              map[string]string
}

type RelationalDatabaseSnapshotsPage struct {
	RelationalDatabaseSnapshots []RelationalDatabaseSnapshot
	NextPageToken               string
}

type RelationalDatabaseFromSnapshotInput struct {
	RelationalDatabaseName         string
	AvailabilityZone               string
	PubliclyAccessible             *bool
	RelationalDatabaseBundleID     string
	RelationalDatabaseSnapshotName string
	RestoreTime                    *time.Time
	SourceRelationalDatabaseName   string
	Tags                           map[string]string
	UseLatestRestorableTime        *bool
}

type RelationalDatabaseEvent struct {
	CreatedAt       time.Time
	EventCategories []string
	Message         string
	Resource        string
}

type RelationalDatabaseEventsPage struct {
	RelationalDatabaseEvents []RelationalDatabaseEvent
	NextPageToken            string
}

type RelationalDatabaseLogEvent struct {
	CreatedAt time.Time
	Message   string
}

type RelationalDatabaseLogEventsInput struct {
	RelationalDatabaseName string
	LogStreamName          string
	StartTime              *time.Time
	EndTime                *time.Time
	PageToken              string
	StartFromHead          *bool
}

type RelationalDatabaseLogEventsPage struct {
	ResourceLogEvents []RelationalDatabaseLogEvent
	NextBackwardToken string
	NextForwardToken  string
}

type RelationalDatabaseMetricInput struct {
	RelationalDatabaseName string
	EndTime                time.Time
	MetricName             string
	Period                 int32
	StartTime              time.Time
	Statistics             []string
	Unit                   string
}

type RelationalDatabaseMetricDatapoint struct {
	Average     *float64
	Maximum     *float64
	Minimum     *float64
	SampleCount *float64
	Sum         *float64
	Timestamp   time.Time
	Unit        string
}

type RelationalDatabaseParameter struct {
	AllowedValues  string
	ApplyMethod    string
	ApplyType      string
	DataType       string
	Description    string
	IsModifiable   bool
	ParameterName  string
	ParameterValue string
}

type RelationalDatabaseParametersPage struct {
	Parameters    []RelationalDatabaseParameter
	NextPageToken string
}

type BucketAccessKeyLastUsed struct {
	LastUsedDate *time.Time
	Region       string
	ServiceName  string
}

type BucketAccessKey struct {
	AccessKeyID     string
	CreatedAt       time.Time
	LastUsed        *BucketAccessKeyLastUsed
	SecretAccessKey string
	Status          string
}

type ContactMethod struct {
	ARN              string
	ContactEndpoint  string
	CreatedAt        time.Time
	AvailabilityZone string
	Region           string
	Name             string
	Protocol         string
	ResourceType     string
	Status           string
	SupportCode      string
}

type Region struct {
	Name              string
	DisplayName       string
	Description       string
	AvailabilityZones []string
	DatabaseZones     []string
	ContinentCode     string
}

type Operation struct {
	ID               string
	ResourceName     string
	ResourceType     string
	OperationType    string
	Status           string
	Details          string
	AvailabilityZone string
	Region           string
	CreatedAt        time.Time
	StatusChangedAt  time.Time
	IsTerminal       bool
}

type Service struct {
	mu                           sync.Mutex
	seq                          uint64
	instances                    map[string]*Instance
	snapshots                    map[string]*InstanceSnapshot
	staticIPs                    map[string]*StaticIP
	disks                        map[string]*Disk
	diskSnapshots                map[string]*DiskSnapshot
	exportRecords                map[string]*ExportSnapshotRecord
	alarms                       map[string]*Alarm
	autoSnapshots                map[string][]AutoSnapshotDetails
	addOns                       map[string]map[string]*AddOn
	loadBalancers                map[string]*LoadBalancer
	lbTLSCerts                   map[string]map[string]*LoadBalancerTLSCertificate
	lbTLSPolicies                []LoadBalancerTLSPolicy
	certificates                 map[string]*Certificate
	domains                      map[string]*Domain
	distributions                map[string]*Distribution
	distributionBundles          []DistributionBundle
	distributionCacheResets      map[string]DistributionCacheReset
	keyPairs                     map[string]*KeyPair
	buckets                      map[string]*Bucket
	bucketAccessKeys             map[string]map[string]*BucketAccessKey
	bucketBundles                []BucketBundle
	containerServices            map[string]*ContainerService
	containerServicePowers       []ContainerServicePower
	containerDeployments         map[string][]ContainerServiceDeployment
	containerImages              map[string]map[string]*ContainerImage
	containerImageVersions       map[string]map[string]int32
	containerLogs                map[string]map[string][]ContainerServiceLogEvent
	blueprints                   []Blueprint
	bundles                      []Bundle
	setupHistory                 map[string][]SetupHistory
	vpcPeered                    bool
	cloudFormationStackRecords   map[string]*CloudFormationStackRecord
	guiSessions                  map[string][]GUISession
	relationalDatabases          map[string]*RelationalDatabase
	relationalDatabaseBlueprints []RelationalDatabaseBlueprint
	relationalDatabaseBundles    []RelationalDatabaseBundle
	relationalDatabaseParameters map[string]map[string]RelationalDatabaseParameter
	relationalDatabaseSnapshots  map[string]*RelationalDatabaseSnapshot
	relationalDatabaseEvents     map[string][]RelationalDatabaseEvent
	relationalDatabaseLogStreams map[string][]string
	relationalDatabaseLogEvents  map[string]map[string][]RelationalDatabaseLogEvent
	contactMethods               map[string]*ContactMethod
	regions                      []Region
	operations                   []Operation
	operationByID                map[string]Operation
}

func NewService() *Service {
	return &Service{
		instances:                    map[string]*Instance{},
		snapshots:                    map[string]*InstanceSnapshot{},
		staticIPs:                    map[string]*StaticIP{},
		disks:                        map[string]*Disk{},
		diskSnapshots:                map[string]*DiskSnapshot{},
		exportRecords:                map[string]*ExportSnapshotRecord{},
		alarms:                       map[string]*Alarm{},
		autoSnapshots:                map[string][]AutoSnapshotDetails{},
		addOns:                       map[string]map[string]*AddOn{},
		loadBalancers:                map[string]*LoadBalancer{},
		lbTLSCerts:                   map[string]map[string]*LoadBalancerTLSCertificate{},
		lbTLSPolicies:                defaultLoadBalancerTLSPolicies(),
		certificates:                 map[string]*Certificate{},
		domains:                      map[string]*Domain{},
		distributions:                map[string]*Distribution{},
		distributionBundles:          defaultDistributionBundles(),
		distributionCacheResets:      map[string]DistributionCacheReset{},
		keyPairs:                     map[string]*KeyPair{},
		buckets:                      map[string]*Bucket{},
		bucketAccessKeys:             map[string]map[string]*BucketAccessKey{},
		bucketBundles:                defaultBucketBundles(),
		containerServices:            map[string]*ContainerService{},
		containerServicePowers:       defaultContainerServicePowers(),
		containerDeployments:         map[string][]ContainerServiceDeployment{},
		containerImages:              map[string]map[string]*ContainerImage{},
		containerImageVersions:       map[string]map[string]int32{},
		containerLogs:                map[string]map[string][]ContainerServiceLogEvent{},
		blueprints:                   defaultBlueprints(),
		bundles:                      defaultBundles(),
		setupHistory:                 map[string][]SetupHistory{},
		vpcPeered:                    false,
		cloudFormationStackRecords:   map[string]*CloudFormationStackRecord{},
		guiSessions:                  map[string][]GUISession{},
		relationalDatabases:          map[string]*RelationalDatabase{},
		relationalDatabaseBlueprints: defaultRelationalDatabaseBlueprints(),
		relationalDatabaseBundles:    defaultRelationalDatabaseBundles(),
		relationalDatabaseParameters: map[string]map[string]RelationalDatabaseParameter{},
		relationalDatabaseSnapshots:  map[string]*RelationalDatabaseSnapshot{},
		relationalDatabaseEvents:     map[string][]RelationalDatabaseEvent{},
		relationalDatabaseLogStreams: map[string][]string{},
		relationalDatabaseLogEvents:  map[string]map[string][]RelationalDatabaseLogEvent{},
		contactMethods:               map[string]*ContactMethod{},
		operations:                   []Operation{},
		operationByID:                map[string]Operation{},
		regions: []Region{
			{
				Name:              "us-east-1",
				DisplayName:       "US East (N. Virginia)",
				Description:       "US East (N. Virginia)",
				ContinentCode:     "NA",
				AvailabilityZones: []string{"us-east-1a", "us-east-1b", "us-east-1c"},
				DatabaseZones:     []string{"us-east-1a", "us-east-1b", "us-east-1c"},
			},
			{
				Name:              "us-west-2",
				DisplayName:       "US West (Oregon)",
				Description:       "US West (Oregon)",
				ContinentCode:     "NA",
				AvailabilityZones: []string{"us-west-2a", "us-west-2b"},
				DatabaseZones:     []string{"us-west-2a", "us-west-2b"},
			},
			{
				Name:              "eu-west-1",
				DisplayName:       "Europe (Ireland)",
				Description:       "Europe (Ireland)",
				ContinentCode:     "EU",
				AvailabilityZones: []string{"eu-west-1a", "eu-west-1b"},
				DatabaseZones:     []string{"eu-west-1a", "eu-west-1b"},
			},
		},
	}
}

func (s *Service) CreateInstances(availabilityZone, blueprintID, bundleID string, names []string, tags map[string]string) ([]Operation, error) {
	availabilityZone = strings.TrimSpace(availabilityZone)
	blueprintID = strings.TrimSpace(blueprintID)
	bundleID = strings.TrimSpace(bundleID)
	if availabilityZone == "" || blueprintID == "" || bundleID == "" || len(names) == 0 {
		return nil, ErrInvalidParameter
	}

	normNames := make([]string, 0, len(names))
	seen := map[string]struct{}{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, ErrInvalidParameter
		}
		if _, ok := seen[name]; ok {
			return nil, ErrInvalidParameter
		}
		seen[name] = struct{}{}
		normNames = append(normNames, name)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, name := range normNames {
		if _, ok := s.instances[name]; ok {
			return nil, ErrAlreadyExists
		}
	}

	ops := make([]Operation, 0, len(normNames))
	for _, name := range normNames {
		region := regionFromAvailabilityZone(availabilityZone)
		seq := atomic.AddUint64(&s.seq, 1)
		now := time.Now().UTC()
		instance := &Instance{
			Name:             name,
			ARN:              instanceARN(region, name),
			BlueprintID:      blueprintID,
			BundleID:         bundleID,
			IPAddressType:    "dualstack",
			AvailabilityZone: availabilityZone,
			Region:           region,
			PublicIPAddress:  fmt.Sprintf("203.0.113.%d", (seq%250)+1),
			PrivateIPAddress: fmt.Sprintf("10.0.%d.%d", (seq/250)%250, (seq%250)+1),
			StateCode:        16,
			StateName:        "running",
			CreatedAt:        now,
			Username:         "ec2-user",
			IPv6Addresses:    []string{fmt.Sprintf("2001:db8::%x", (seq%65535)+1)},
			PortStates:       defaultInstancePortStates(),
			MetadataOptions:  defaultInstanceMetadataOptions(),
			HostKeys:         defaultInstanceHostKeys(name, now, seq),
			Tags:             cloneStringMap(tags),
		}
		s.instances[name] = instance
		ops = append(ops, newOperation(seq, name, "Instance", "CreateInstances", "Succeeded", "instance created", availabilityZone, region, now))
		s.appendSetupHistoryLocked(name, "create-instance", "instance created", "", "succeeded", "1.0.0")
	}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetBlueprints(includeInactive bool, appCategory, pageToken string) (BlueprintsPage, error) {
	appCategory = strings.TrimSpace(appCategory)
	pageToken = strings.TrimSpace(pageToken)
	if appCategory != "" && !strings.EqualFold(appCategory, "LfR") {
		return BlueprintsPage{}, ErrInvalidParameter
	}
	offset := 0
	if pageToken != "" {
		value, err := strconv.Atoi(pageToken)
		if err != nil || value < 0 {
			return BlueprintsPage{}, ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]Blueprint, 0, len(s.blueprints))
	for _, blueprint := range s.blueprints {
		if !includeInactive && !blueprint.IsActive {
			continue
		}
		if appCategory != "" && !strings.EqualFold(blueprint.AppCategory, appCategory) {
			continue
		}
		filtered = append(filtered, blueprint)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].BlueprintID < filtered[j].BlueprintID })
	if offset >= len(filtered) {
		return BlueprintsPage{Blueprints: []Blueprint{}}, nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	nextPageToken := ""
	if end < len(filtered) {
		nextPageToken = strconv.Itoa(end)
	}
	return BlueprintsPage{
		Blueprints:    append([]Blueprint(nil), filtered[offset:end]...),
		NextPageToken: nextPageToken,
	}, nil
}

func (s *Service) GetBundles(includeInactive bool, appCategory, pageToken string) (BundlesPage, error) {
	appCategory = strings.TrimSpace(appCategory)
	pageToken = strings.TrimSpace(pageToken)
	if appCategory != "" && !strings.EqualFold(appCategory, "LfR") {
		return BundlesPage{}, ErrInvalidParameter
	}
	offset := 0
	if pageToken != "" {
		value, err := strconv.Atoi(pageToken)
		if err != nil || value < 0 {
			return BundlesPage{}, ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]Bundle, 0, len(s.bundles))
	for _, bundle := range s.bundles {
		if !includeInactive && !bundle.IsActive {
			continue
		}
		if appCategory != "" {
			match := strings.EqualFold(bundle.AppCategory, appCategory)
			if !match {
				for _, supported := range bundle.SupportedAppCategories {
					if strings.EqualFold(strings.TrimSpace(supported), appCategory) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, bundle)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].BundleID < filtered[j].BundleID })
	if offset >= len(filtered) {
		return BundlesPage{Bundles: []Bundle{}}, nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	nextPageToken := ""
	if end < len(filtered) {
		nextPageToken = strconv.Itoa(end)
	}
	return BundlesPage{
		Bundles:       append([]Bundle(nil), filtered[offset:end]...),
		NextPageToken: nextPageToken,
	}, nil
}

func (s *Service) GetActiveNames(pageToken string) ([]string, string, error) {
	pageToken = strings.TrimSpace(pageToken)
	offset := 0
	if pageToken != "" {
		value, err := strconv.Atoi(pageToken)
		if err != nil || value < 0 {
			return nil, "", ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	names := make([]string, 0, len(s.instances)+len(s.snapshots)+len(s.staticIPs)+len(s.disks)+len(s.diskSnapshots)+len(s.loadBalancers)+len(s.alarms)+len(s.domains)+len(s.distributions)+len(s.keyPairs)+len(s.buckets)+len(s.containerServices)+len(s.relationalDatabases)+len(s.relationalDatabaseSnapshots)+len(s.certificates)+len(s.contactMethods)+len(s.cloudFormationStackRecords))
	for name := range s.instances {
		names = append(names, name)
	}
	for name := range s.snapshots {
		names = append(names, name)
	}
	for name := range s.staticIPs {
		names = append(names, name)
	}
	for name := range s.disks {
		names = append(names, name)
	}
	for name := range s.diskSnapshots {
		names = append(names, name)
	}
	for name := range s.loadBalancers {
		names = append(names, name)
	}
	for name := range s.alarms {
		names = append(names, name)
	}
	for name := range s.domains {
		names = append(names, name)
	}
	for name := range s.distributions {
		names = append(names, name)
	}
	for name := range s.keyPairs {
		names = append(names, name)
	}
	for name := range s.buckets {
		names = append(names, name)
	}
	for name := range s.containerServices {
		names = append(names, name)
	}
	for name := range s.relationalDatabases {
		names = append(names, name)
	}
	for name := range s.relationalDatabaseSnapshots {
		names = append(names, name)
	}
	for name := range s.certificates {
		names = append(names, name)
	}
	for name := range s.contactMethods {
		names = append(names, name)
	}
	for name := range s.cloudFormationStackRecords {
		names = append(names, name)
	}
	names = dedupeStrings(names)
	sort.Strings(names)
	if offset >= len(names) {
		return []string{}, "", nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(names) {
		end = len(names)
	}
	nextPageToken := ""
	if end < len(names) {
		nextPageToken = strconv.Itoa(end)
	}
	return append([]string(nil), names[offset:end]...), nextPageToken, nil
}

func (s *Service) GetSetupHistory(resourceName, pageToken string) (SetupHistoryPage, error) {
	resourceName = strings.TrimSpace(resourceName)
	pageToken = strings.TrimSpace(pageToken)
	if resourceName == "" {
		return SetupHistoryPage{}, ErrInvalidParameter
	}
	offset := 0
	if pageToken != "" {
		value, err := strconv.Atoi(pageToken)
		if err != nil || value < 0 {
			return SetupHistoryPage{}, ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, _, _, _, exists := s.resourceIdentityLocked(resourceName); !exists {
		if _, hadHistory := s.setupHistory[resourceName]; !hadHistory {
			return SetupHistoryPage{}, ErrNotFound
		}
	}
	items := cloneSetupHistory(s.setupHistory[resourceName])
	sort.Slice(items, func(i, j int) bool {
		if items[i].Resource.CreatedAt.Equal(items[j].Resource.CreatedAt) {
			return items[i].OperationID > items[j].OperationID
		}
		return items[i].Resource.CreatedAt.After(items[j].Resource.CreatedAt)
	})
	if offset >= len(items) {
		return SetupHistoryPage{SetupHistory: []SetupHistory{}}, nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(items) {
		end = len(items)
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	return SetupHistoryPage{
		SetupHistory:  items[offset:end],
		NextPageToken: nextPageToken,
	}, nil
}

func (s *Service) GetCostEstimate(resourceName string, startTime, endTime time.Time) ([]ResourceBudgetEstimate, error) {
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" || startTime.IsZero() || endTime.IsZero() || endTime.Before(startTime) {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resourceType, _, _, _, exists := s.resourceIdentityLocked(resourceName)
	if !exists {
		return nil, ErrNotFound
	}
	durationHours := endTime.Sub(startTime).Hours()
	if durationHours < 1 {
		durationHours = 1
	}
	unitCost := 0.01
	switch resourceType {
	case "Instance":
		unitCost = 0.02
	case "RelationalDatabase":
		unitCost = 0.03
	case "Bucket":
		unitCost = 0.005
	case "Distribution":
		unitCost = 0.015
	}
	estimate := ResourceBudgetEstimate{
		ResourceName: resourceName,
		ResourceType: resourceType,
		StartTime:    startTime.UTC(),
		EndTime:      endTime.UTC(),
		CostEstimates: []CostEstimate{
			{
				UsageType: "Cost",
				ResultsByTime: []EstimateByTime{
					{
						Currency:    "USD",
						PricingUnit: "Hrs",
						StartTime:   startTime.UTC(),
						EndTime:     endTime.UTC(),
						Unit:        durationHours,
						UsageCost:   durationHours * unitCost,
					},
				},
			},
		},
	}
	return []ResourceBudgetEstimate{estimate}, nil
}

func (s *Service) IsVpcPeered() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.vpcPeered
}

func (s *Service) PeerVpc() (Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.vpcPeered = true
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, "LightsailVpc", "Vpc", "PeerVpc", "Succeeded", "vpc peered", DefaultRegion+"a", DefaultRegion, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) UnpeerVpc() (Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.vpcPeered = false
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, "LightsailVpc", "Vpc", "UnpeerVpc", "Succeeded", "vpc unpeered", DefaultRegion+"a", DefaultRegion, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) CreateCloudFormationStack(entries []InstanceEntry) ([]Operation, error) {
	if len(entries) == 0 || len(entries) > 1 {
		return nil, ErrInvalidParameter
	}
	for idx := range entries {
		entries[idx].AvailabilityZone = strings.TrimSpace(entries[idx].AvailabilityZone)
		entries[idx].InstanceType = strings.TrimSpace(entries[idx].InstanceType)
		entries[idx].PortInfoSource = strings.TrimSpace(entries[idx].PortInfoSource)
		entries[idx].SourceName = strings.TrimSpace(entries[idx].SourceName)
		entries[idx].UserData = strings.TrimSpace(entries[idx].UserData)
		if entries[idx].AvailabilityZone == "" || entries[idx].InstanceType == "" || entries[idx].PortInfoSource == "" || entries[idx].SourceName == "" {
			return nil, ErrInvalidParameter
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	first := entries[0]
	region := regionFromAvailabilityZone(first.AvailabilityZone)
	recordName := fmt.Sprintf("CloudFormationStackRecord-%d", seq)
	record := &CloudFormationStackRecord{
		ARN:       fmt.Sprintf("arn:aws:lightsail:%s:%s:CloudFormationStackRecord/%s", region, DefaultAccountID, recordName),
		CreatedAt: now,
		DestinationInfo: CloudFormationDestinationInfo{
			ID:      fmt.Sprintf("arn:aws:cloudformation:%s:%s:stack/%s/%d", region, DefaultAccountID, recordName, seq),
			Service: "CloudFormation",
		},
		AvailabilityZone: first.AvailabilityZone,
		Region:           region,
		Name:             recordName,
		ResourceType:     "CloudFormationStackRecord",
		SourceInfo: []CloudFormationStackSourceInfo{
			{
				ARN:          fmt.Sprintf("arn:aws:lightsail:%s:%s:ExportSnapshotRecord/%s", region, DefaultAccountID, first.SourceName),
				Name:         first.SourceName,
				ResourceType: "ExportSnapshotRecord",
			},
		},
		State: "Succeeded",
	}
	if exportRecord, ok := s.exportRecords[first.SourceName]; ok {
		record.SourceInfo[0].ARN = exportRecord.ARN
	}
	s.cloudFormationStackRecords[recordName] = record

	op := newOperation(seq, recordName, "CloudFormationStackRecord", "CreateCloudFormationStack", "Succeeded", "cloudformation stack created", first.AvailabilityZone, region, now)
	s.appendOperationsLocked([]Operation{op})
	s.appendSetupHistoryLocked(first.SourceName, "create-cloudformation-stack", "cloudformation stack created", "", "succeeded", "1.0.0")
	return []Operation{op}, nil
}

func (s *Service) GetCloudFormationStackRecords(pageToken string) (CloudFormationStackRecordsPage, error) {
	pageToken = strings.TrimSpace(pageToken)
	offset := 0
	if pageToken != "" {
		value, err := strconv.Atoi(pageToken)
		if err != nil || value < 0 {
			return CloudFormationStackRecordsPage{}, ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	records := make([]CloudFormationStackRecord, 0, len(s.cloudFormationStackRecords))
	for _, record := range s.cloudFormationStackRecords {
		records = append(records, cloneCloudFormationStackRecord(record))
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].CreatedAt.Equal(records[j].CreatedAt) {
			return records[i].Name < records[j].Name
		}
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})
	if offset >= len(records) {
		return CloudFormationStackRecordsPage{CloudFormationStackRecords: []CloudFormationStackRecord{}}, nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(records) {
		end = len(records)
	}
	nextPageToken := ""
	if end < len(records) {
		nextPageToken = strconv.Itoa(end)
	}
	return CloudFormationStackRecordsPage{
		CloudFormationStackRecords: records[offset:end],
		NextPageToken:              nextPageToken,
	}, nil
}

func (s *Service) CreateGUISessionAccessDetails(resourceName string) (GUISessionAccessDetails, error) {
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" {
		return GUISessionAccessDetails{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance, exists := s.instances[resourceName]
	if !exists {
		return GUISessionAccessDetails{}, ErrNotFound
	}
	sessions := s.guiSessions[resourceName]
	if len(sessions) == 0 {
		sessions = []GUISession{
			{
				IsPrimary: true,
				Name:      "primary",
				URL:       fmt.Sprintf("https://%s.console.lightsail.aws.amazon.com/gui/%s", instance.Region, resourceName),
			},
		}
		s.guiSessions[resourceName] = sessions
	}
	s.appendSetupHistoryLocked(resourceName, "create-gui-session-access-details", "gui session access details created", "", "succeeded", "1.0.0")
	return GUISessionAccessDetails{
		FailureReason:      "",
		PercentageComplete: 100,
		ResourceName:       resourceName,
		Sessions:           append([]GUISession(nil), sessions...),
		Status:             "started",
	}, nil
}

func (s *Service) StartGUISession(resourceName string) ([]Operation, error) {
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance, exists := s.instances[resourceName]
	if !exists {
		return nil, ErrNotFound
	}
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	s.guiSessions[resourceName] = []GUISession{
		{
			IsPrimary: true,
			Name:      "primary",
			URL:       fmt.Sprintf("https://%s.console.lightsail.aws.amazon.com/gui/%s?session=%d", instance.Region, resourceName, seq),
		},
	}
	op := newOperation(seq, resourceName, "Instance", "StartGUISession", "Succeeded", "gui session started", instance.AvailabilityZone, instance.Region, now)
	s.appendOperationsLocked([]Operation{op})
	s.appendSetupHistoryLocked(resourceName, "start-gui-session", "gui session started", "", "succeeded", "1.0.0")
	return []Operation{op}, nil
}

func (s *Service) StopGUISession(resourceName string) ([]Operation, error) {
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance, exists := s.instances[resourceName]
	if !exists {
		return nil, ErrNotFound
	}
	delete(s.guiSessions, resourceName)
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, resourceName, "Instance", "StopGUISession", "Succeeded", "gui session stopped", instance.AvailabilityZone, instance.Region, now)
	s.appendOperationsLocked([]Operation{op})
	s.appendSetupHistoryLocked(resourceName, "stop-gui-session", "gui session stopped", "", "succeeded", "1.0.0")
	return []Operation{op}, nil
}

func (s *Service) GetInstanceMetricData(input InstanceMetricInput) (string, []InstanceMetricDatapoint, error) {
	input.InstanceName = strings.TrimSpace(input.InstanceName)
	input.MetricName = normalizeInstanceMetricName(input.MetricName)
	input.Unit = strings.TrimSpace(input.Unit)
	if input.InstanceName == "" || input.MetricName == "" || input.Unit == "" || input.Period <= 0 || input.StartTime.IsZero() || input.EndTime.IsZero() || input.EndTime.Before(input.StartTime) || len(input.Statistics) == 0 {
		return "", nil, ErrInvalidParameter
	}
	if !hasAnyMetricStatistic(input.Statistics) {
		return "", nil, ErrInvalidParameter
	}
	if !validInstanceMetricUnit(input.MetricName, input.Unit) {
		return "", nil, ErrInvalidParameter
	}

	s.mu.Lock()
	_, exists := s.instances[input.InstanceName]
	s.mu.Unlock()
	if !exists {
		return "", nil, ErrNotFound
	}

	step := time.Duration(input.Period) * time.Second
	count := int(input.EndTime.Sub(input.StartTime)/step) + 1
	if count < 1 {
		count = 1
	}
	if count > 1440 {
		count = 1440
	}

	scale := 1.0
	switch input.MetricName {
	case "BurstCapacityPercentage", "CPUUtilization":
		scale = 5.0
	case "BurstCapacityTime":
		scale = 30.0
	case "NetworkIn", "NetworkOut":
		scale = 1024 * 64
	case "StatusCheckFailed", "StatusCheckFailed_Instance", "StatusCheckFailed_System", "MetadataNoToken":
		scale = 1
	}

	out := make([]InstanceMetricDatapoint, 0, count)
	for i := 0; i < count; i++ {
		ts := input.StartTime.Add(time.Duration(i) * step)
		if ts.After(input.EndTime) {
			break
		}
		base := float64(i+1) * scale
		point := InstanceMetricDatapoint{
			Timestamp: ts.UTC(),
			Unit:      input.Unit,
		}
		if hasMetricStatistic(input.Statistics, "Average") {
			v := base
			point.Average = &v
		}
		if hasMetricStatistic(input.Statistics, "Maximum") {
			v := base + (0.5 * scale)
			point.Maximum = &v
		}
		if hasMetricStatistic(input.Statistics, "Minimum") {
			v := base - (0.5 * scale)
			point.Minimum = &v
		}
		if hasMetricStatistic(input.Statistics, "SampleCount") {
			v := float64(1)
			point.SampleCount = &v
		}
		if hasMetricStatistic(input.Statistics, "Sum") {
			v := base
			point.Sum = &v
		}
		out = append(out, point)
	}
	if len(out) == 0 {
		out = append(out, InstanceMetricDatapoint{
			Timestamp: input.EndTime.UTC(),
			Unit:      input.Unit,
		})
	}
	return input.MetricName, out, nil
}

func (s *Service) OpenInstancePublicPorts(instanceName string, info PortInfo) (Operation, error) {
	instanceName = strings.TrimSpace(instanceName)
	if instanceName == "" {
		return Operation{}, ErrInvalidParameter
	}
	info = normalizePortInfo(info)
	if info.Protocol == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.instances[instanceName]
	if !ok {
		return Operation{}, ErrNotFound
	}
	state := InstancePortState{PortInfo: info, State: "open"}
	replaced := false
	for i := range instance.PortStates {
		if samePortRule(instance.PortStates[i].PortInfo, info) {
			instance.PortStates[i] = state
			replaced = true
			break
		}
	}
	if !replaced {
		instance.PortStates = append(instance.PortStates, state)
	}
	sort.Slice(instance.PortStates, func(i, j int) bool {
		if instance.PortStates[i].Protocol == instance.PortStates[j].Protocol {
			if instance.PortStates[i].FromPort == instance.PortStates[j].FromPort {
				return instance.PortStates[i].ToPort < instance.PortStates[j].ToPort
			}
			return instance.PortStates[i].FromPort < instance.PortStates[j].FromPort
		}
		return instance.PortStates[i].Protocol < instance.PortStates[j].Protocol
	})
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, instanceName, "Instance", "OpenInstancePublicPorts", "Succeeded", "opened instance public ports", instance.AvailabilityZone, instance.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) CloseInstancePublicPorts(instanceName string, info PortInfo) (Operation, error) {
	instanceName = strings.TrimSpace(instanceName)
	if instanceName == "" {
		return Operation{}, ErrInvalidParameter
	}
	info = normalizePortInfo(info)
	if info.Protocol == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.instances[instanceName]
	if !ok {
		return Operation{}, ErrNotFound
	}
	filtered := make([]InstancePortState, 0, len(instance.PortStates))
	for _, state := range instance.PortStates {
		if samePortRule(state.PortInfo, info) {
			continue
		}
		filtered = append(filtered, state)
	}
	instance.PortStates = filtered
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, instanceName, "Instance", "CloseInstancePublicPorts", "Succeeded", "closed instance public ports", instance.AvailabilityZone, instance.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) PutInstancePublicPorts(instanceName string, infos []PortInfo) (Operation, error) {
	instanceName = strings.TrimSpace(instanceName)
	if instanceName == "" || len(infos) == 0 {
		return Operation{}, ErrInvalidParameter
	}

	normalized := make([]InstancePortState, 0, len(infos))
	for _, info := range infos {
		norm := normalizePortInfo(info)
		if norm.Protocol == "" {
			return Operation{}, ErrInvalidParameter
		}
		normalized = append(normalized, InstancePortState{PortInfo: norm, State: "open"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.instances[instanceName]
	if !ok {
		return Operation{}, ErrNotFound
	}
	instance.PortStates = normalized
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, instanceName, "Instance", "PutInstancePublicPorts", "Succeeded", "updated instance public ports", instance.AvailabilityZone, instance.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) GetInstancePortStates(instanceName string) ([]InstancePortState, error) {
	instanceName = strings.TrimSpace(instanceName)
	if instanceName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.instances[instanceName]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneInstancePortStates(instance.PortStates), nil
}

func (s *Service) GetInstanceAccessDetails(instanceName, protocol string) (InstanceAccessDetails, error) {
	instanceName = strings.TrimSpace(instanceName)
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if protocol == "" {
		protocol = "ssh"
	}
	if instanceName == "" {
		return InstanceAccessDetails{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.instances[instanceName]
	if !ok {
		return InstanceAccessDetails{}, ErrNotFound
	}
	now := time.Now().UTC()
	details := InstanceAccessDetails{
		InstanceName:  instance.Name,
		Protocol:      protocol,
		Username:      instance.Username,
		IpAddress:     instance.PublicIPAddress,
		Ipv6Addresses: append([]string(nil), instance.IPv6Addresses...),
		ExpiresAt:     now.Add(60 * time.Minute),
		HostKeys:      cloneHostKeyAttributes(instance.HostKeys),
	}
	if protocol == "rdp" {
		details.Username = "Administrator"
		details.Password = "Stackyard-RDP-Password-1!"
	} else {
		details.PrivateKey = "-----BEGIN PRIVATE KEY-----\nSTACKYARD-TEMP-PRIVATE-KEY\n-----END PRIVATE KEY-----"
		details.CertKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAISTACKYARDTEMPKEY stackyard@localhost"
	}
	return details, nil
}

func (s *Service) UpdateInstanceMetadataOptions(instanceName, httpEndpoint, httpProtocolIpv6, httpTokens string, httpPutResponseHopLimit *int32) (Operation, error) {
	instanceName = strings.TrimSpace(instanceName)
	httpEndpoint = strings.ToLower(strings.TrimSpace(httpEndpoint))
	httpProtocolIpv6 = strings.ToLower(strings.TrimSpace(httpProtocolIpv6))
	httpTokens = strings.ToLower(strings.TrimSpace(httpTokens))
	if instanceName == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.instances[instanceName]
	if !ok {
		return Operation{}, ErrNotFound
	}
	if httpEndpoint != "" {
		instance.MetadataOptions.HttpEndpoint = httpEndpoint
	}
	if httpProtocolIpv6 != "" {
		instance.MetadataOptions.HttpProtocolIpv6 = httpProtocolIpv6
	}
	if httpTokens != "" {
		instance.MetadataOptions.HttpTokens = httpTokens
	}
	if httpPutResponseHopLimit != nil {
		instance.MetadataOptions.HttpPutResponseHopLimit = *httpPutResponseHopLimit
	}
	instance.MetadataOptions.State = "applied"

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, instanceName, "Instance", "UpdateInstanceMetadataOptions", "Succeeded", "updated instance metadata options", instance.AvailabilityZone, instance.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) CreateInstancesFromSnapshot(availabilityZone, bundleID string, instanceNames []string, instanceSnapshotName, sourceInstanceName string, tags map[string]string) ([]Operation, error) {
	availabilityZone = strings.TrimSpace(availabilityZone)
	bundleID = strings.TrimSpace(bundleID)
	instanceSnapshotName = strings.TrimSpace(instanceSnapshotName)
	sourceInstanceName = strings.TrimSpace(sourceInstanceName)
	if availabilityZone == "" || bundleID == "" || len(instanceNames) == 0 {
		return nil, ErrInvalidParameter
	}
	if instanceSnapshotName == "" && sourceInstanceName == "" {
		return nil, ErrInvalidParameter
	}

	normNames := make([]string, 0, len(instanceNames))
	seen := map[string]struct{}{}
	for _, name := range instanceNames {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, ErrInvalidParameter
		}
		if _, ok := seen[name]; ok {
			return nil, ErrInvalidParameter
		}
		seen[name] = struct{}{}
		normNames = append(normNames, name)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var blueprintID string
	if instanceSnapshotName != "" {
		snapshot, ok := s.snapshots[instanceSnapshotName]
		if !ok {
			return nil, ErrNotFound
		}
		blueprintID = snapshot.FromBlueprintID
	} else {
		source, ok := s.instances[sourceInstanceName]
		if !ok {
			return nil, ErrNotFound
		}
		blueprintID = source.BlueprintID
	}
	if blueprintID == "" {
		blueprintID = "amazon_linux_2"
	}
	for _, name := range normNames {
		if _, exists := s.instances[name]; exists {
			return nil, ErrAlreadyExists
		}
	}

	ops := make([]Operation, 0, len(normNames))
	for _, name := range normNames {
		region := regionFromAvailabilityZone(availabilityZone)
		seq := atomic.AddUint64(&s.seq, 1)
		now := time.Now().UTC()
		instance := &Instance{
			Name:             name,
			ARN:              instanceARN(region, name),
			BlueprintID:      blueprintID,
			BundleID:         bundleID,
			AvailabilityZone: availabilityZone,
			Region:           region,
			PublicIPAddress:  fmt.Sprintf("203.0.113.%d", (seq%250)+1),
			PrivateIPAddress: fmt.Sprintf("10.0.%d.%d", (seq/250)%250, (seq%250)+1),
			StateCode:        16,
			StateName:        "running",
			CreatedAt:        now,
			Username:         "ec2-user",
			IPv6Addresses:    []string{fmt.Sprintf("2001:db8::%x", (seq%65535)+1)},
			PortStates:       defaultInstancePortStates(),
			MetadataOptions:  defaultInstanceMetadataOptions(),
			HostKeys:         defaultInstanceHostKeys(name, now, seq),
			Tags:             cloneStringMap(tags),
		}
		s.instances[name] = instance
		ops = append(ops, newOperation(seq, name, "Instance", "CreateInstancesFromSnapshot", "Succeeded", "instance created from snapshot", availabilityZone, region, now))
	}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) DeleteKnownHostKeys(instanceName string) ([]Operation, error) {
	instanceName = strings.TrimSpace(instanceName)
	if instanceName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.instances[instanceName]
	if !ok {
		return nil, ErrNotFound
	}
	now := time.Now().UTC()
	seq := atomic.AddUint64(&s.seq, 1)
	instance.HostKeys = defaultInstanceHostKeys(instanceName, now, seq)
	op := newOperation(seq, instanceName, "Instance", "DeleteKnownHostKeys", "Succeeded", "known host keys deleted", instance.AvailabilityZone, instance.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return []Operation{op}, nil
}

func (s *Service) CreateDisk(availabilityZone, diskName string, sizeInGb int32, tags map[string]string) ([]Operation, error) {
	availabilityZone = strings.TrimSpace(availabilityZone)
	diskName = strings.TrimSpace(diskName)
	if availabilityZone == "" || diskName == "" || sizeInGb <= 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.disks[diskName]; exists {
		return nil, ErrAlreadyExists
	}
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := regionFromAvailabilityZone(availabilityZone)
	disk := &Disk{
		Name:             diskName,
		ARN:              diskARN(region, diskName),
		AvailabilityZone: availabilityZone,
		Region:           region,
		CreatedAt:        now,
		SizeInGb:         sizeInGb,
		Iops:             sizeInGb * 100,
		IsAttached:       false,
		State:            "available",
		AutoMountStatus:  "NotMounted",
		IsSystemDisk:     false,
		Tags:             cloneStringMap(tags),
	}
	s.disks[diskName] = disk
	ops := []Operation{newOperation(seq, diskName, "Disk", "CreateDisk", "Succeeded", "disk created", availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetDisk(diskName string) (Disk, bool) {
	diskName = strings.TrimSpace(diskName)
	s.mu.Lock()
	defer s.mu.Unlock()
	disk, ok := s.disks[diskName]
	if !ok {
		return Disk{}, false
	}
	return cloneDisk(disk), true
}

func (s *Service) GetDisks() []Disk {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Disk, 0, len(s.disks))
	for _, disk := range s.disks {
		out = append(out, cloneDisk(disk))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) DeleteDisk(diskName string) ([]Operation, error) {
	diskName = strings.TrimSpace(diskName)
	if diskName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	disk, ok := s.disks[diskName]
	if !ok {
		return nil, ErrNotFound
	}
	delete(s.disks, diskName)
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, diskName, "Disk", "DeleteDisk", "Succeeded", "disk deleted", disk.AvailabilityZone, disk.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) AttachDisk(diskName, diskPath, instanceName string, autoMounting bool) ([]Operation, error) {
	diskName = strings.TrimSpace(diskName)
	diskPath = strings.TrimSpace(diskPath)
	instanceName = strings.TrimSpace(instanceName)
	if diskName == "" || diskPath == "" || instanceName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	disk, ok := s.disks[diskName]
	if !ok {
		return nil, ErrNotFound
	}
	instance, ok := s.instances[instanceName]
	if !ok {
		return nil, ErrNotFound
	}
	disk.AttachedTo = instanceName
	disk.Path = diskPath
	disk.IsAttached = true
	disk.State = "in-use"
	if autoMounting {
		disk.AutoMountStatus = "Mounted"
	} else {
		disk.AutoMountStatus = "NotMounted"
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, diskName, "Disk", "AttachDisk", "Succeeded", "disk attached", instance.AvailabilityZone, instance.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) DetachDisk(diskName string) ([]Operation, error) {
	diskName = strings.TrimSpace(diskName)
	if diskName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	disk, ok := s.disks[diskName]
	if !ok {
		return nil, ErrNotFound
	}
	availabilityZone := disk.AvailabilityZone
	region := disk.Region
	if disk.AttachedTo != "" {
		if instance, exists := s.instances[disk.AttachedTo]; exists {
			availabilityZone = instance.AvailabilityZone
			region = instance.Region
		}
	}
	disk.AttachedTo = ""
	disk.Path = ""
	disk.IsAttached = false
	disk.State = "available"
	disk.AutoMountStatus = "NotMounted"

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, diskName, "Disk", "DetachDisk", "Succeeded", "disk detached", availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) CreateDiskSnapshot(diskName, instanceName, diskSnapshotName string, tags map[string]string) ([]Operation, error) {
	diskName = strings.TrimSpace(diskName)
	instanceName = strings.TrimSpace(instanceName)
	diskSnapshotName = strings.TrimSpace(diskSnapshotName)
	if diskSnapshotName == "" {
		return nil, ErrInvalidParameter
	}
	if (diskName == "" && instanceName == "") || (diskName != "" && instanceName != "") {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.diskSnapshots[diskSnapshotName]; exists {
		return nil, ErrAlreadyExists
	}

	var (
		fromDiskARN      string
		fromDiskName     string
		fromInstanceARN  string
		fromInstanceName string
		availabilityZone string
		region           string
		sizeInGb         int32
	)
	if diskName != "" {
		disk, exists := s.disks[diskName]
		if !exists {
			return nil, ErrNotFound
		}
		fromDiskARN = disk.ARN
		fromDiskName = disk.Name
		availabilityZone = disk.AvailabilityZone
		region = disk.Region
		sizeInGb = disk.SizeInGb
	} else {
		instance, exists := s.instances[instanceName]
		if !exists {
			return nil, ErrNotFound
		}
		fromInstanceARN = instance.ARN
		fromInstanceName = instance.Name
		availabilityZone = instance.AvailabilityZone
		region = instance.Region
		sizeInGb = 8
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	snapshot := &DiskSnapshot{
		Name:               diskSnapshotName,
		ARN:                diskSnapshotARN(region, diskSnapshotName),
		AvailabilityZone:   availabilityZone,
		Region:             region,
		CreatedAt:          now,
		FromDiskARN:        fromDiskARN,
		FromDiskName:       fromDiskName,
		FromInstanceARN:    fromInstanceARN,
		FromInstanceName:   fromInstanceName,
		IsFromAutoSnapshot: false,
		Progress:           "100%",
		SizeInGb:           sizeInGb,
		State:              "completed",
		SupportCode:        fmt.Sprintf("%s/%d", region, seq),
		Tags:               cloneStringMap(tags),
	}
	s.diskSnapshots[diskSnapshotName] = snapshot

	ops := []Operation{newOperation(seq, diskSnapshotName, "DiskSnapshot", "CreateDiskSnapshot", "Succeeded", "disk snapshot created", availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetDiskSnapshot(diskSnapshotName string) (DiskSnapshot, bool) {
	diskSnapshotName = strings.TrimSpace(diskSnapshotName)
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.diskSnapshots[diskSnapshotName]
	if !ok {
		return DiskSnapshot{}, false
	}
	return cloneDiskSnapshot(snapshot), true
}

func (s *Service) GetDiskSnapshots() []DiskSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]DiskSnapshot, 0, len(s.diskSnapshots))
	for _, snapshot := range s.diskSnapshots {
		out = append(out, cloneDiskSnapshot(snapshot))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) DeleteDiskSnapshot(diskSnapshotName string) ([]Operation, error) {
	diskSnapshotName = strings.TrimSpace(diskSnapshotName)
	if diskSnapshotName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.diskSnapshots[diskSnapshotName]
	if !ok {
		return nil, ErrNotFound
	}
	delete(s.diskSnapshots, diskSnapshotName)
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, diskSnapshotName, "DiskSnapshot", "DeleteDiskSnapshot", "Succeeded", "disk snapshot deleted", snapshot.AvailabilityZone, snapshot.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) CreateDiskFromSnapshot(availabilityZone, diskName, diskSnapshotName, sourceDiskName string, sizeInGb int32, tags map[string]string) ([]Operation, error) {
	availabilityZone = strings.TrimSpace(availabilityZone)
	diskName = strings.TrimSpace(diskName)
	diskSnapshotName = strings.TrimSpace(diskSnapshotName)
	sourceDiskName = strings.TrimSpace(sourceDiskName)
	if availabilityZone == "" || diskName == "" || sizeInGb <= 0 {
		return nil, ErrInvalidParameter
	}
	if diskSnapshotName == "" && sourceDiskName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.disks[diskName]; exists {
		return nil, ErrAlreadyExists
	}

	var sourceSnapshot *DiskSnapshot
	if diskSnapshotName != "" {
		sourceSnapshot = s.diskSnapshots[diskSnapshotName]
	} else {
		for _, candidate := range s.diskSnapshots {
			if candidate.FromDiskName != sourceDiskName {
				continue
			}
			if sourceSnapshot == nil || candidate.CreatedAt.After(sourceSnapshot.CreatedAt) {
				sourceSnapshot = candidate
			}
		}
	}
	if sourceSnapshot == nil {
		return nil, ErrNotFound
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := regionFromAvailabilityZone(availabilityZone)
	disk := &Disk{
		Name:             diskName,
		ARN:              diskARN(region, diskName),
		AvailabilityZone: availabilityZone,
		Region:           region,
		CreatedAt:        now,
		SizeInGb:         sizeInGb,
		Iops:             sizeInGb * 100,
		IsAttached:       false,
		State:            "available",
		AutoMountStatus:  "NotMounted",
		IsSystemDisk:     false,
		Tags:             cloneStringMap(tags),
	}
	s.disks[diskName] = disk

	ops := []Operation{newOperation(seq, diskName, "Disk", "CreateDiskFromSnapshot", "Succeeded", "disk created from snapshot", availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) CopySnapshot(sourceRegion, targetSnapshotName, sourceSnapshotName, sourceResourceName string) ([]Operation, error) {
	sourceRegion = strings.TrimSpace(sourceRegion)
	targetSnapshotName = strings.TrimSpace(targetSnapshotName)
	sourceSnapshotName = strings.TrimSpace(sourceSnapshotName)
	sourceResourceName = strings.TrimSpace(sourceResourceName)
	if sourceRegion == "" || targetSnapshotName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.diskSnapshots[targetSnapshotName]; exists {
		return nil, ErrAlreadyExists
	}

	var sourceSnapshot *DiskSnapshot
	if sourceSnapshotName != "" {
		sourceSnapshot = s.diskSnapshots[sourceSnapshotName]
	}
	if sourceSnapshot == nil && sourceResourceName != "" {
		for _, candidate := range s.diskSnapshots {
			if candidate.FromDiskName != sourceResourceName && candidate.FromInstanceName != sourceResourceName {
				continue
			}
			if sourceSnapshot == nil || candidate.CreatedAt.After(sourceSnapshot.CreatedAt) {
				sourceSnapshot = candidate
			}
		}
	}
	if sourceSnapshot == nil {
		return nil, ErrNotFound
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := sourceSnapshot.Region
	if region == "" {
		region = sourceRegion
	}
	availabilityZone := sourceSnapshot.AvailabilityZone
	if availabilityZone == "" && region != "" {
		availabilityZone = region + "a"
	}
	snapshot := &DiskSnapshot{
		Name:               targetSnapshotName,
		ARN:                diskSnapshotARN(region, targetSnapshotName),
		AvailabilityZone:   availabilityZone,
		Region:             region,
		CreatedAt:          now,
		FromDiskARN:        sourceSnapshot.FromDiskARN,
		FromDiskName:       sourceSnapshot.FromDiskName,
		FromInstanceARN:    sourceSnapshot.FromInstanceARN,
		FromInstanceName:   sourceSnapshot.FromInstanceName,
		IsFromAutoSnapshot: false,
		Progress:           "100%",
		SizeInGb:           sourceSnapshot.SizeInGb,
		State:              "completed",
		SupportCode:        fmt.Sprintf("%s/%d", region, seq),
		Tags:               cloneStringMap(sourceSnapshot.Tags),
	}
	s.diskSnapshots[targetSnapshotName] = snapshot

	ops := []Operation{newOperation(seq, targetSnapshotName, "DiskSnapshot", "CopySnapshot", "Succeeded", "snapshot copied", availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) ExportSnapshot(sourceSnapshotName string) ([]Operation, error) {
	sourceSnapshotName = strings.TrimSpace(sourceSnapshotName)
	if sourceSnapshotName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, exists := s.diskSnapshots[sourceSnapshotName]
	if !exists {
		return nil, ErrNotFound
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	recordName := fmt.Sprintf("%s-export-%d", sourceSnapshotName, seq)
	record := &ExportSnapshotRecord{
		Name:               recordName,
		ARN:                exportSnapshotRecordARN(snapshot.Region, recordName),
		AvailabilityZone:   snapshot.AvailabilityZone,
		Region:             snapshot.Region,
		CreatedAt:          now,
		DestinationID:      fmt.Sprintf("snap-%08x", seq*97),
		DestinationService: "EC2",
		SourceSnapshotARN:  snapshot.ARN,
		SourceSnapshotName: snapshot.Name,
		SourceCreatedAt:    snapshot.CreatedAt,
		SourceResourceARN:  firstNonEmptyString(snapshot.FromDiskARN, snapshot.FromInstanceARN),
		SourceResourceName: firstNonEmptyString(snapshot.FromDiskName, snapshot.FromInstanceName),
		SourceType:         "DiskSnapshot",
		SourceDiskSizeInGb: snapshot.SizeInGb,
		State:              "Completed",
	}
	s.exportRecords[recordName] = record

	ops := []Operation{newOperation(seq, sourceSnapshotName, "DiskSnapshot", "ExportSnapshot", "Succeeded", "snapshot exported", snapshot.AvailabilityZone, snapshot.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetExportSnapshotRecords() []ExportSnapshotRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ExportSnapshotRecord, 0, len(s.exportRecords))
	for _, record := range s.exportRecords {
		out = append(out, cloneExportSnapshotRecord(record))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Service) PutAlarm(
	alarmName, comparisonOperator, metricName, monitoredResourceName string,
	evaluationPeriods int32,
	threshold float64,
	contactProtocols []string,
	datapointsToAlarm *int32,
	notificationEnabled *bool,
	notificationTriggers []string,
	treatMissingData string,
) ([]Operation, error) {
	alarmName = strings.TrimSpace(alarmName)
	comparisonOperator = strings.TrimSpace(comparisonOperator)
	metricName = strings.TrimSpace(metricName)
	monitoredResourceName = strings.TrimSpace(monitoredResourceName)
	treatMissingData = strings.TrimSpace(treatMissingData)
	if alarmName == "" || comparisonOperator == "" || metricName == "" || monitoredResourceName == "" || evaluationPeriods <= 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resourceType, resourceARN, availabilityZone, region, ok := s.resourceIdentityLocked(monitoredResourceName)
	if !ok {
		return nil, ErrNotFound
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()

	dp := evaluationPeriods
	if datapointsToAlarm != nil && *datapointsToAlarm > 0 {
		dp = *datapointsToAlarm
	}
	enabled := true
	if notificationEnabled != nil {
		enabled = *notificationEnabled
	}
	if len(notificationTriggers) == 0 {
		notificationTriggers = []string{"ALARM"}
	}
	if treatMissingData == "" {
		treatMissingData = "missing"
	}
	alarm := &Alarm{
		Name:               alarmName,
		ARN:                alarmARN(region, alarmName),
		ComparisonOperator: comparisonOperator,
		ContactProtocols:   dedupeStrings(contactProtocols),
		CreatedAt:          now,
		DatapointsToAlarm:  dp,
		EvaluationPeriods:  evaluationPeriods,
		MetricName:         metricName,
		MonitoredResource: AlarmMonitoredResourceInfo{
			ARN:          resourceARN,
			Name:         monitoredResourceName,
			ResourceType: resourceType,
		},
		NotificationEnabled:  enabled,
		NotificationTriggers: dedupeStrings(notificationTriggers),
		Period:               300,
		ResourceType:         "Alarm",
		State:                "INSUFFICIENT_DATA",
		Statistic:            "Average",
		SupportCode:          fmt.Sprintf("%s/%d", region, seq),
		Threshold:            threshold,
		TreatMissingData:     treatMissingData,
		Unit:                 "Percent",
		AvailabilityZone:     availabilityZone,
		Region:               region,
	}
	s.alarms[alarmName] = alarm

	ops := []Operation{newOperation(seq, alarmName, "Alarm", "PutAlarm", "Succeeded", "alarm configured", availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetAlarms(alarmName, monitoredResourceName string) []Alarm {
	alarmName = strings.TrimSpace(alarmName)
	monitoredResourceName = strings.TrimSpace(monitoredResourceName)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Alarm, 0, len(s.alarms))
	for _, alarm := range s.alarms {
		if alarmName != "" && alarm.Name != alarmName {
			continue
		}
		if monitoredResourceName != "" && alarm.MonitoredResource.Name != monitoredResourceName {
			continue
		}
		out = append(out, cloneAlarm(alarm))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) TestAlarm(alarmName, state string) ([]Operation, error) {
	alarmName = strings.TrimSpace(alarmName)
	state = strings.TrimSpace(state)
	if alarmName == "" || state == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	alarm, ok := s.alarms[alarmName]
	if !ok {
		return nil, ErrNotFound
	}
	alarm.State = state

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, alarmName, "Alarm", "TestAlarm", "Succeeded", "alarm tested", alarm.AvailabilityZone, alarm.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) DeleteAlarm(alarmName string) ([]Operation, error) {
	alarmName = strings.TrimSpace(alarmName)
	if alarmName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	alarm, ok := s.alarms[alarmName]
	if !ok {
		return nil, ErrNotFound
	}
	delete(s.alarms, alarmName)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, alarmName, "Alarm", "DeleteAlarm", "Succeeded", "alarm deleted", alarm.AvailabilityZone, alarm.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetAutoSnapshots(resourceName string) ([]AutoSnapshotDetails, string, error) {
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" {
		return nil, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resourceType, _, _, _, ok := s.resourceIdentityLocked(resourceName)
	if !ok {
		return nil, "", ErrNotFound
	}
	items := s.autoSnapshots[resourceName]
	out := make([]AutoSnapshotDetails, 0, len(items))
	for _, item := range items {
		out = append(out, cloneAutoSnapshotDetails(item))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out, resourceType, nil
}

func (s *Service) DeleteAutoSnapshot(resourceName, date string) ([]Operation, error) {
	resourceName = strings.TrimSpace(resourceName)
	date = strings.TrimSpace(date)
	if resourceName == "" || date == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	resourceType, _, availabilityZone, region, ok := s.resourceIdentityLocked(resourceName)
	if !ok {
		return nil, ErrNotFound
	}
	items := s.autoSnapshots[resourceName]
	filtered := make([]AutoSnapshotDetails, 0, len(items))
	removed := false
	for _, item := range items {
		if item.Date == date {
			removed = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !removed {
		return nil, ErrNotFound
	}
	s.autoSnapshots[resourceName] = filtered

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	details := "automatic snapshot deleted"
	ops := []Operation{newOperation(seq, resourceName, resourceType, "DeleteAutoSnapshot", "Succeeded", details, availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) EnableAddOn(resourceName, addOnType, snapshotTimeOfDay string) ([]Operation, error) {
	resourceName = strings.TrimSpace(resourceName)
	addOnType = strings.TrimSpace(addOnType)
	snapshotTimeOfDay = strings.TrimSpace(snapshotTimeOfDay)
	if resourceName == "" || addOnType == "" {
		return nil, ErrInvalidParameter
	}
	if snapshotTimeOfDay == "" {
		snapshotTimeOfDay = "01:00"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resourceType, _, availabilityZone, region, ok := s.resourceIdentityLocked(resourceName)
	if !ok {
		return nil, ErrNotFound
	}
	if s.addOns[resourceName] == nil {
		s.addOns[resourceName] = map[string]*AddOn{}
	}
	s.addOns[resourceName][addOnType] = &AddOn{
		Name:                  addOnType,
		Status:                "Enabled",
		SnapshotTimeOfDay:     snapshotTimeOfDay,
		NextSnapshotTimeOfDay: snapshotTimeOfDay,
	}
	if strings.EqualFold(addOnType, "AutoSnapshot") && len(s.autoSnapshots[resourceName]) == 0 {
		now := time.Now().UTC()
		s.autoSnapshots[resourceName] = []AutoSnapshotDetails{{
			CreatedAt:         now,
			Date:              now.Format("2006-01-02"),
			FromAttachedDisks: s.autoSnapshotAttachedDisksLocked(resourceName),
			Status:            "Success",
		}}
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, resourceName, resourceType, "EnableAddOn", "Succeeded", "add-on enabled", availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) DisableAddOn(resourceName, addOnType string) ([]Operation, error) {
	resourceName = strings.TrimSpace(resourceName)
	addOnType = strings.TrimSpace(addOnType)
	if resourceName == "" || addOnType == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resourceType, _, availabilityZone, region, ok := s.resourceIdentityLocked(resourceName)
	if !ok {
		return nil, ErrNotFound
	}
	if byType := s.addOns[resourceName]; byType != nil {
		delete(byType, addOnType)
		if len(byType) == 0 {
			delete(s.addOns, resourceName)
		}
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, resourceName, resourceType, "DisableAddOn", "Succeeded", "add-on disabled", availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) CreateLoadBalancer(loadBalancerName string, instancePort int32, certificateName, certificateDomainName string, certificateAlternativeNames []string, healthCheckPath, ipAddressType, tlsPolicyName string, tags map[string]string) ([]Operation, error) {
	loadBalancerName = strings.TrimSpace(loadBalancerName)
	certificateName = strings.TrimSpace(certificateName)
	certificateDomainName = strings.TrimSpace(certificateDomainName)
	healthCheckPath = strings.TrimSpace(healthCheckPath)
	ipAddressType = normalizeLoadBalancerIPAddressType(ipAddressType)
	tlsPolicyName = strings.TrimSpace(tlsPolicyName)
	if loadBalancerName == "" || instancePort <= 0 {
		return nil, ErrInvalidParameter
	}
	if (certificateName == "") != (certificateDomainName == "") {
		return nil, ErrInvalidParameter
	}
	if ipAddressType == "" {
		return nil, ErrInvalidParameter
	}
	if healthCheckPath == "" {
		healthCheckPath = "/"
	}
	if !strings.HasPrefix(healthCheckPath, "/") {
		healthCheckPath = "/" + healthCheckPath
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.loadBalancers[loadBalancerName]; exists {
		return nil, ErrAlreadyExists
	}
	if tlsPolicyName == "" {
		tlsPolicyName = s.defaultLoadBalancerTLSPolicyNameLocked()
	}
	if !s.loadBalancerTLSPolicyExistsLocked(tlsPolicyName) {
		return nil, ErrInvalidParameter
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := DefaultRegion
	availabilityZone := region + "a"
	lb := &LoadBalancer{
		Name:                    loadBalancerName,
		ARN:                     loadBalancerARN(region, loadBalancerName),
		CreatedAt:               now,
		DNSName:                 fmt.Sprintf("%s.%s.elb.amazonaws.com", strings.ToLower(loadBalancerName), region),
		HealthCheckPath:         healthCheckPath,
		HTTPSRedirectionEnabled: false,
		InstanceHealthSummary:   []LoadBalancerInstanceHealthSummary{},
		InstancePort:            instancePort,
		IPAddressType:           ipAddressType,
		AvailabilityZone:        availabilityZone,
		Region:                  region,
		Protocol:                "HTTP",
		PublicPorts:             []int32{80},
		ResourceType:            "LoadBalancer",
		State:                   "active",
		SupportCode:             fmt.Sprintf("%s/%d", region, seq),
		Tags:                    cloneStringMap(tags),
		TLSCertificateSummaries: []LoadBalancerTLSCertificateSummary{},
		TLSPolicyName:           tlsPolicyName,
		ConfigurationOptions: map[string]string{
			"HealthCheckPath":                            healthCheckPath,
			"SessionStickinessEnabled":                   "false",
			"SessionStickiness_LB_CookieDurationSeconds": "0",
			"HttpsRedirectionEnabled":                    "false",
			"TlsPolicyName":                              tlsPolicyName,
		},
		AttachedInstances: []string{},
	}
	s.loadBalancers[loadBalancerName] = lb

	if certificateName != "" {
		if s.lbTLSCerts[loadBalancerName] == nil {
			s.lbTLSCerts[loadBalancerName] = map[string]*LoadBalancerTLSCertificate{}
		}
		if _, exists := s.lbTLSCerts[loadBalancerName][certificateName]; !exists {
			cert := &LoadBalancerTLSCertificate{
				Name:                    certificateName,
				ARN:                     loadBalancerTLSCertificateARN(region, loadBalancerName, certificateName),
				CreatedAt:               now,
				DomainName:              certificateDomainName,
				LoadBalancerName:        loadBalancerName,
				AvailabilityZone:        availabilityZone,
				Region:                  region,
				IsAttached:              true,
				IssuedAt:                now,
				Issuer:                  "Amazon",
				KeyAlgorithm:            "RSA-2048",
				NotBefore:               now,
				NotAfter:                now.Add(365 * 24 * time.Hour),
				ResourceType:            "LoadBalancerTlsCertificate",
				Status:                  "ISSUED",
				Subject:                 certificateDomainName,
				SubjectAlternativeNames: dedupeStrings(certificateAlternativeNames),
				SupportCode:             fmt.Sprintf("%s/%d", region, seq),
				Tags:                    map[string]string{},
			}
			s.lbTLSCerts[loadBalancerName][certificateName] = cert
		}
		for name, cert := range s.lbTLSCerts[loadBalancerName] {
			cert.IsAttached = strings.EqualFold(name, certificateName)
		}
	}
	s.refreshLoadBalancerDerivedLocked(lb)

	ops := []Operation{newOperation(seq, loadBalancerName, "LoadBalancer", "CreateLoadBalancer", "Succeeded", "load balancer created", availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) DeleteLoadBalancer(loadBalancerName string) ([]Operation, error) {
	loadBalancerName = strings.TrimSpace(loadBalancerName)
	if loadBalancerName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	lb, exists := s.loadBalancers[loadBalancerName]
	if !exists {
		return nil, ErrNotFound
	}
	delete(s.loadBalancers, loadBalancerName)
	delete(s.lbTLSCerts, loadBalancerName)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, loadBalancerName, "LoadBalancer", "DeleteLoadBalancer", "Succeeded", "load balancer deleted", lb.AvailabilityZone, lb.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) AttachInstancesToLoadBalancer(loadBalancerName string, instanceNames []string) ([]Operation, error) {
	loadBalancerName = strings.TrimSpace(loadBalancerName)
	if loadBalancerName == "" || len(instanceNames) == 0 {
		return nil, ErrInvalidParameter
	}

	normalized := make([]string, 0, len(instanceNames))
	seen := map[string]struct{}{}
	for _, name := range instanceNames {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, ErrInvalidParameter
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	if len(normalized) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	lb, exists := s.loadBalancers[loadBalancerName]
	if !exists {
		return nil, ErrNotFound
	}
	for _, instanceName := range normalized {
		if _, exists := s.instances[instanceName]; !exists {
			return nil, ErrNotFound
		}
	}
	for _, instanceName := range normalized {
		if !containsString(lb.AttachedInstances, instanceName) {
			lb.AttachedInstances = append(lb.AttachedInstances, instanceName)
		}
	}
	s.refreshLoadBalancerDerivedLocked(lb)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, loadBalancerName, "LoadBalancer", "AttachInstancesToLoadBalancer", "Succeeded", "instances attached to load balancer", lb.AvailabilityZone, lb.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) DetachInstancesFromLoadBalancer(loadBalancerName string, instanceNames []string) ([]Operation, error) {
	loadBalancerName = strings.TrimSpace(loadBalancerName)
	if loadBalancerName == "" || len(instanceNames) == 0 {
		return nil, ErrInvalidParameter
	}

	normalized := make([]string, 0, len(instanceNames))
	seen := map[string]struct{}{}
	for _, name := range instanceNames {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, ErrInvalidParameter
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	if len(normalized) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	lb, exists := s.loadBalancers[loadBalancerName]
	if !exists {
		return nil, ErrNotFound
	}
	filtered := make([]string, 0, len(lb.AttachedInstances))
	for _, attached := range lb.AttachedInstances {
		if containsString(normalized, attached) {
			continue
		}
		filtered = append(filtered, attached)
	}
	lb.AttachedInstances = filtered
	s.refreshLoadBalancerDerivedLocked(lb)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, loadBalancerName, "LoadBalancer", "DetachInstancesFromLoadBalancer", "Succeeded", "instances detached from load balancer", lb.AvailabilityZone, lb.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetLoadBalancer(loadBalancerName string) (LoadBalancer, bool) {
	loadBalancerName = strings.TrimSpace(loadBalancerName)
	s.mu.Lock()
	defer s.mu.Unlock()
	lb, exists := s.loadBalancers[loadBalancerName]
	if !exists {
		return LoadBalancer{}, false
	}
	s.refreshLoadBalancerDerivedLocked(lb)
	return cloneLoadBalancer(lb), true
}

func (s *Service) GetLoadBalancers(pageToken string) (LoadBalancersPage, error) {
	pageToken = strings.TrimSpace(pageToken)
	offset := 0
	if pageToken != "" {
		value, err := strconv.Atoi(pageToken)
		if err != nil || value < 0 {
			return LoadBalancersPage{}, ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]LoadBalancer, 0, len(s.loadBalancers))
	for _, lb := range s.loadBalancers {
		s.refreshLoadBalancerDerivedLocked(lb)
		items = append(items, cloneLoadBalancer(lb))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	if offset >= len(items) {
		return LoadBalancersPage{LoadBalancers: []LoadBalancer{}}, nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(items) {
		end = len(items)
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	return LoadBalancersPage{
		LoadBalancers: append([]LoadBalancer(nil), items[offset:end]...),
		NextPageToken: nextPageToken,
	}, nil
}

func (s *Service) UpdateLoadBalancerAttribute(loadBalancerName, attributeName, attributeValue string) ([]Operation, error) {
	loadBalancerName = strings.TrimSpace(loadBalancerName)
	attributeName = normalizeLoadBalancerAttributeName(attributeName)
	attributeValue = strings.TrimSpace(attributeValue)
	if loadBalancerName == "" || attributeName == "" || attributeValue == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	lb, exists := s.loadBalancers[loadBalancerName]
	if !exists {
		return nil, ErrNotFound
	}

	switch attributeName {
	case "HealthCheckPath":
		if !strings.HasPrefix(attributeValue, "/") {
			attributeValue = "/" + attributeValue
		}
		lb.HealthCheckPath = attributeValue
	case "SessionStickinessEnabled":
		if _, err := strconv.ParseBool(attributeValue); err != nil {
			return nil, ErrInvalidParameter
		}
	case "SessionStickiness_LB_CookieDurationSeconds":
		value, err := strconv.Atoi(attributeValue)
		if err != nil || value < 0 {
			return nil, ErrInvalidParameter
		}
	case "HttpsRedirectionEnabled":
		enabled, err := strconv.ParseBool(attributeValue)
		if err != nil {
			return nil, ErrInvalidParameter
		}
		lb.HTTPSRedirectionEnabled = enabled
	case "TlsPolicyName":
		if !s.loadBalancerTLSPolicyExistsLocked(attributeValue) {
			return nil, ErrInvalidParameter
		}
		lb.TLSPolicyName = attributeValue
	default:
		return nil, ErrInvalidParameter
	}

	if lb.ConfigurationOptions == nil {
		lb.ConfigurationOptions = map[string]string{}
	}
	lb.ConfigurationOptions[attributeName] = attributeValue
	s.refreshLoadBalancerDerivedLocked(lb)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, loadBalancerName, "LoadBalancer", "UpdateLoadBalancerAttribute", "Succeeded", "load balancer attribute updated", lb.AvailabilityZone, lb.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetLoadBalancerMetricData(input LoadBalancerMetricInput) (string, []LoadBalancerMetricDatapoint, error) {
	input.LoadBalancerName = strings.TrimSpace(input.LoadBalancerName)
	input.MetricName = normalizeLoadBalancerMetricName(input.MetricName)
	input.Unit = strings.TrimSpace(input.Unit)
	if input.LoadBalancerName == "" || input.MetricName == "" || input.Unit == "" || input.Period <= 0 || input.StartTime.IsZero() || input.EndTime.IsZero() || input.EndTime.Before(input.StartTime) || len(input.Statistics) == 0 {
		return "", nil, ErrInvalidParameter
	}
	if !hasAnyMetricStatistic(input.Statistics) {
		return "", nil, ErrInvalidParameter
	}
	if !validLoadBalancerMetricUnit(input.MetricName, input.Unit) {
		return "", nil, ErrInvalidParameter
	}

	s.mu.Lock()
	lb, exists := s.loadBalancers[input.LoadBalancerName]
	if exists {
		s.refreshLoadBalancerDerivedLocked(lb)
	}
	s.mu.Unlock()
	if !exists {
		return "", nil, ErrNotFound
	}

	step := time.Duration(input.Period) * time.Second
	count := int(input.EndTime.Sub(input.StartTime)/step) + 1
	if count < 1 {
		count = 1
	}
	if count > 1440 {
		count = 1440
	}

	scale := 1.0
	switch input.MetricName {
	case "InstanceResponseTime":
		scale = 0.05
	case "RequestCount":
		scale = 10
	case "HealthyHostCount", "UnhealthyHostCount":
		scale = 1
	default:
		scale = 2
	}

	healthyHosts := 0
	unhealthyHosts := 0
	for _, summary := range lb.InstanceHealthSummary {
		if strings.EqualFold(summary.InstanceHealth, "healthy") {
			healthyHosts++
		} else {
			unhealthyHosts++
		}
	}

	out := make([]LoadBalancerMetricDatapoint, 0, count)
	for i := 0; i < count; i++ {
		ts := input.StartTime.Add(time.Duration(i) * step)
		if ts.After(input.EndTime) {
			break
		}
		base := float64(i+1) * scale
		switch input.MetricName {
		case "HealthyHostCount":
			base = float64(healthyHosts)
		case "UnhealthyHostCount":
			base = float64(unhealthyHosts)
		}
		point := LoadBalancerMetricDatapoint{
			Timestamp: ts.UTC(),
			Unit:      input.Unit,
		}
		if hasMetricStatistic(input.Statistics, "Average") {
			v := base
			point.Average = &v
		}
		if hasMetricStatistic(input.Statistics, "Maximum") {
			v := base + (0.5 * scale)
			point.Maximum = &v
		}
		if hasMetricStatistic(input.Statistics, "Minimum") {
			v := base - (0.5 * scale)
			if v < 0 {
				v = 0
			}
			point.Minimum = &v
		}
		if hasMetricStatistic(input.Statistics, "SampleCount") {
			v := float64(1)
			point.SampleCount = &v
		}
		if hasMetricStatistic(input.Statistics, "Sum") {
			v := base
			point.Sum = &v
		}
		out = append(out, point)
	}
	if len(out) == 0 {
		out = append(out, LoadBalancerMetricDatapoint{
			Timestamp: input.EndTime.UTC(),
			Unit:      input.Unit,
		})
	}
	return input.MetricName, out, nil
}

func (s *Service) CreateLoadBalancerTLSCertificate(loadBalancerName, certificateName, certificateDomainName string, certificateAlternativeNames []string, tags map[string]string) ([]Operation, error) {
	loadBalancerName = strings.TrimSpace(loadBalancerName)
	certificateName = strings.TrimSpace(certificateName)
	certificateDomainName = strings.TrimSpace(certificateDomainName)
	if loadBalancerName == "" || certificateName == "" || certificateDomainName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lbTLSCerts[loadBalancerName] == nil {
		s.lbTLSCerts[loadBalancerName] = map[string]*LoadBalancerTLSCertificate{}
	}
	if _, exists := s.lbTLSCerts[loadBalancerName][certificateName]; exists {
		return nil, ErrAlreadyExists
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := DefaultRegion
	availabilityZone := region + "a"
	cert := &LoadBalancerTLSCertificate{
		Name:                    certificateName,
		ARN:                     loadBalancerTLSCertificateARN(region, loadBalancerName, certificateName),
		CreatedAt:               now,
		DomainName:              certificateDomainName,
		LoadBalancerName:        loadBalancerName,
		AvailabilityZone:        availabilityZone,
		Region:                  region,
		IsAttached:              false,
		IssuedAt:                now,
		Issuer:                  "Amazon",
		KeyAlgorithm:            "RSA-2048",
		NotBefore:               now,
		NotAfter:                now.Add(365 * 24 * time.Hour),
		ResourceType:            "LoadBalancerTlsCertificate",
		Status:                  "ISSUED",
		Subject:                 certificateDomainName,
		SubjectAlternativeNames: dedupeStrings(certificateAlternativeNames),
		SupportCode:             fmt.Sprintf("%s/%d", region, seq),
		Tags:                    cloneStringMap(tags),
	}
	s.lbTLSCerts[loadBalancerName][certificateName] = cert

	ops := []Operation{newOperation(seq, certificateName, "LoadBalancerTlsCertificate", "CreateLoadBalancerTlsCertificate", "Succeeded", "load balancer tls certificate created", availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetLoadBalancerTLSCertificates(loadBalancerName string) []LoadBalancerTLSCertificate {
	loadBalancerName = strings.TrimSpace(loadBalancerName)
	s.mu.Lock()
	defer s.mu.Unlock()
	byName := s.lbTLSCerts[loadBalancerName]
	if len(byName) == 0 {
		return []LoadBalancerTLSCertificate{}
	}
	out := make([]LoadBalancerTLSCertificate, 0, len(byName))
	for _, cert := range byName {
		out = append(out, cloneLoadBalancerTLSCertificate(cert))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) GetLoadBalancerTLSPolicies() []LoadBalancerTLSPolicy {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]LoadBalancerTLSPolicy, len(s.lbTLSPolicies))
	copy(out, s.lbTLSPolicies)
	return out
}

func (s *Service) AttachLoadBalancerTLSCertificate(loadBalancerName, certificateName string) ([]Operation, error) {
	loadBalancerName = strings.TrimSpace(loadBalancerName)
	certificateName = strings.TrimSpace(certificateName)
	if loadBalancerName == "" || certificateName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	byName := s.lbTLSCerts[loadBalancerName]
	if len(byName) == 0 {
		return nil, ErrNotFound
	}
	cert, ok := byName[certificateName]
	if !ok {
		return nil, ErrNotFound
	}
	for _, item := range byName {
		item.IsAttached = false
	}
	cert.IsAttached = true
	if lb, exists := s.loadBalancers[loadBalancerName]; exists {
		s.refreshLoadBalancerDerivedLocked(lb)
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, certificateName, "LoadBalancerTlsCertificate", "AttachLoadBalancerTlsCertificate", "Succeeded", "load balancer tls certificate attached", cert.AvailabilityZone, cert.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) DeleteLoadBalancerTLSCertificate(loadBalancerName, certificateName string, force bool) ([]Operation, error) {
	loadBalancerName = strings.TrimSpace(loadBalancerName)
	certificateName = strings.TrimSpace(certificateName)
	if loadBalancerName == "" || certificateName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	byName := s.lbTLSCerts[loadBalancerName]
	if len(byName) == 0 {
		return nil, ErrNotFound
	}
	cert, ok := byName[certificateName]
	if !ok {
		return nil, ErrNotFound
	}
	if cert.IsAttached && !force {
		return nil, ErrInvalidParameter
	}
	delete(byName, certificateName)
	if len(byName) == 0 {
		delete(s.lbTLSCerts, loadBalancerName)
	}
	if lb, exists := s.loadBalancers[loadBalancerName]; exists {
		s.refreshLoadBalancerDerivedLocked(lb)
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, certificateName, "LoadBalancerTlsCertificate", "DeleteLoadBalancerTlsCertificate", "Succeeded", "load balancer tls certificate deleted", cert.AvailabilityZone, cert.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) SetupInstanceHTTPS(certificateProvider string, domainNames []string, emailAddress, instanceName string) ([]Operation, error) {
	certificateProvider = strings.TrimSpace(certificateProvider)
	emailAddress = strings.TrimSpace(emailAddress)
	instanceName = strings.TrimSpace(instanceName)
	domainNames = dedupeStrings(domainNames)
	if certificateProvider == "" || emailAddress == "" || instanceName == "" || len(domainNames) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[instanceName]
	if !ok {
		return nil, ErrNotFound
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, instanceName, "Instance", "SetupInstanceHttps", "Succeeded", "instance https configured", instance.AvailabilityZone, instance.Region, now)}
	s.appendOperationsLocked(ops)
	s.appendSetupHistoryLocked(instanceName, "setup-instance-https", "instance https configured", "", "succeeded", "1.0.0")
	return ops, nil
}

func (s *Service) CreateDistribution(input DistributionCreateInput) (Distribution, Operation, error) {
	input.BundleID = strings.TrimSpace(input.BundleID)
	input.DistributionName = strings.TrimSpace(input.DistributionName)
	input.DefaultCacheBehavior.Behavior = strings.TrimSpace(input.DefaultCacheBehavior.Behavior)
	input.CertificateName = strings.TrimSpace(input.CertificateName)
	input.IPAddressType = firstNonEmptyString(input.IPAddressType, "dualstack")
	input.ViewerMinimumTLSProtocolVersion = firstNonEmptyString(input.ViewerMinimumTLSProtocolVersion, "TLSv1.2_2021")
	if input.BundleID == "" || input.DistributionName == "" || input.DefaultCacheBehavior.Behavior == "" || strings.TrimSpace(input.Origin.Name) == "" {
		return Distribution{}, Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.distributions[input.DistributionName]; exists {
		return Distribution{}, Operation{}, ErrAlreadyExists
	}
	if !s.bundleExistsLocked(input.BundleID) {
		return Distribution{}, Operation{}, ErrInvalidParameter
	}

	now := time.Now().UTC()
	seq := atomic.AddUint64(&s.seq, 1)
	region := DefaultRegion
	availabilityZone := region + "a"

	origin := normalizeDistributionOrigin(input.Origin)
	cacheSettings := normalizeDistributionCacheSettings(input.CacheBehaviorSettings)
	cacheBehaviors := normalizeDistributionCacheBehaviors(input.CacheBehaviors)
	domainName := fmt.Sprintf("%s-%d.cloudfront.net", strings.ToLower(input.DistributionName), seq)
	originPublicDNS := fmt.Sprintf("%s.%s.lightsail.local", origin.Name, origin.RegionName)
	distribution := &Distribution{
		AbleToUpdateBundle:              true,
		AlternativeDomainNames:          []string{},
		ARN:                             distributionARN(region, input.DistributionName),
		BundleID:                        input.BundleID,
		CacheBehaviorSettings:           cacheSettings,
		CacheBehaviors:                  cacheBehaviors,
		CertificateName:                 input.CertificateName,
		CreatedAt:                       now,
		DefaultCacheBehavior:            DistributionCacheBehavior{Behavior: strings.TrimSpace(input.DefaultCacheBehavior.Behavior)},
		DomainName:                      domainName,
		IPAddressType:                   input.IPAddressType,
		IsEnabled:                       true,
		Name:                            input.DistributionName,
		Origin:                          origin,
		OriginPublicDNS:                 originPublicDNS,
		ResourceType:                    "Distribution",
		Status:                          "Enabled",
		SupportCode:                     fmt.Sprintf("%s/%d", region, seq),
		Tags:                            cloneStringMap(input.Tags),
		ViewerMinimumTLSProtocolVersion: input.ViewerMinimumTLSProtocolVersion,
		AvailabilityZone:                availabilityZone,
		Region:                          region,
	}
	s.distributions[input.DistributionName] = distribution

	op := newOperation(seq, input.DistributionName, "Distribution", "CreateDistribution", "Succeeded", "distribution created", availabilityZone, region, now)
	s.appendOperationsLocked([]Operation{op})
	return cloneDistribution(distribution), op, nil
}

func (s *Service) GetDistributions(distributionName string) []Distribution {
	distributionName = strings.TrimSpace(distributionName)

	s.mu.Lock()
	defer s.mu.Unlock()

	if distributionName != "" {
		dist, ok := s.distributions[distributionName]
		if !ok {
			return []Distribution{}
		}
		return []Distribution{cloneDistribution(dist)}
	}

	out := make([]Distribution, 0, len(s.distributions))
	for _, dist := range s.distributions {
		out = append(out, cloneDistribution(dist))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) DeleteDistribution(distributionName string) (Operation, error) {
	distributionName = strings.TrimSpace(distributionName)
	if distributionName == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	dist, ok := s.distributions[distributionName]
	if !ok {
		return Operation{}, ErrNotFound
	}
	delete(s.distributions, distributionName)
	delete(s.distributionCacheResets, distributionName)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, distributionName, "Distribution", "DeleteDistribution", "Succeeded", "distribution deleted", dist.AvailabilityZone, dist.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) UpdateDistribution(input DistributionUpdateInput) (Operation, error) {
	input.DistributionName = strings.TrimSpace(input.DistributionName)
	if input.DistributionName == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	dist, ok := s.distributions[input.DistributionName]
	if !ok {
		return Operation{}, ErrNotFound
	}

	if input.CacheBehaviorSettings != nil {
		dist.CacheBehaviorSettings = normalizeDistributionCacheSettings(*input.CacheBehaviorSettings)
	}
	if input.HasCacheBehaviors {
		dist.CacheBehaviors = normalizeDistributionCacheBehaviors(input.CacheBehaviors)
	}
	if input.CertificateName != nil {
		dist.CertificateName = strings.TrimSpace(*input.CertificateName)
	}
	if input.DefaultCacheBehavior != nil {
		behavior := strings.TrimSpace(input.DefaultCacheBehavior.Behavior)
		if behavior == "" {
			return Operation{}, ErrInvalidParameter
		}
		dist.DefaultCacheBehavior = DistributionCacheBehavior{Behavior: behavior}
	}
	if input.IsEnabled != nil {
		dist.IsEnabled = *input.IsEnabled
		if dist.IsEnabled {
			dist.Status = "Enabled"
		} else {
			dist.Status = "Disabled"
		}
	}
	if input.Origin != nil {
		updated := dist.Origin
		if v := strings.TrimSpace(input.Origin.Name); v != "" {
			updated.Name = v
		}
		if v := strings.TrimSpace(input.Origin.ProtocolPolicy); v != "" {
			updated.ProtocolPolicy = v
		}
		if v := strings.TrimSpace(input.Origin.RegionName); v != "" {
			updated.RegionName = v
		}
		if v := strings.TrimSpace(input.Origin.ResourceType); v != "" {
			updated.ResourceType = v
		}
		if input.Origin.ResponseTimeout > 0 {
			updated.ResponseTimeout = input.Origin.ResponseTimeout
		}
		dist.Origin = normalizeDistributionOrigin(updated)
		dist.OriginPublicDNS = fmt.Sprintf("%s.%s.lightsail.local", dist.Origin.Name, dist.Origin.RegionName)
	}
	if input.UseDefaultCertificate != nil && *input.UseDefaultCertificate {
		dist.CertificateName = ""
	}
	if v := strings.TrimSpace(input.ViewerMinimumTLSProtocolVersion); v != "" {
		dist.ViewerMinimumTLSProtocolVersion = v
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, input.DistributionName, "Distribution", "UpdateDistribution", "Succeeded", "distribution updated", dist.AvailabilityZone, dist.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) ResetDistributionCache(distributionName string) (DistributionCacheReset, Operation, error) {
	distributionName = strings.TrimSpace(distributionName)
	if distributionName == "" {
		return DistributionCacheReset{}, Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	dist, ok := s.distributions[distributionName]
	if !ok {
		return DistributionCacheReset{}, Operation{}, ErrNotFound
	}

	now := time.Now().UTC()
	reset := DistributionCacheReset{
		CreateTime: now,
		Status:     "Succeeded",
	}
	s.distributionCacheResets[distributionName] = reset

	seq := atomic.AddUint64(&s.seq, 1)
	op := newOperation(seq, distributionName, "Distribution", "ResetDistributionCache", "Succeeded", "distribution cache reset", dist.AvailabilityZone, dist.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return reset, op, nil
}

func (s *Service) GetDistributionMetricData(input DistributionMetricInput) (string, []DistributionMetricDatapoint, error) {
	input.DistributionName = strings.TrimSpace(input.DistributionName)
	input.MetricName = strings.TrimSpace(input.MetricName)
	input.Unit = strings.TrimSpace(input.Unit)
	if input.DistributionName == "" || input.MetricName == "" || input.Unit == "" || input.Period <= 0 || input.StartTime.IsZero() || input.EndTime.IsZero() || input.EndTime.Before(input.StartTime) || len(input.Statistics) == 0 {
		return "", nil, ErrInvalidParameter
	}
	for i := range input.Statistics {
		input.Statistics[i] = strings.TrimSpace(input.Statistics[i])
	}

	s.mu.Lock()
	_, ok := s.distributions[input.DistributionName]
	s.mu.Unlock()
	if !ok {
		return "", nil, ErrNotFound
	}

	step := time.Duration(input.Period) * time.Second
	count := int(input.EndTime.Sub(input.StartTime)/step) + 1
	if count < 1 {
		count = 1
	}
	if count > 1440 {
		count = 1440
	}

	out := make([]DistributionMetricDatapoint, 0, count)
	for i := 0; i < count; i++ {
		ts := input.StartTime.Add(time.Duration(i) * step)
		if ts.After(input.EndTime) {
			break
		}
		base := float64(i + 1)
		point := DistributionMetricDatapoint{
			Timestamp: ts.UTC(),
			Unit:      input.Unit,
		}
		if hasMetricStatistic(input.Statistics, "Average") {
			v := base
			point.Average = &v
		}
		if hasMetricStatistic(input.Statistics, "Maximum") {
			v := base + 0.5
			point.Maximum = &v
		}
		if hasMetricStatistic(input.Statistics, "Minimum") {
			v := base - 0.5
			point.Minimum = &v
		}
		if hasMetricStatistic(input.Statistics, "SampleCount") {
			v := float64(1)
			point.SampleCount = &v
		}
		if hasMetricStatistic(input.Statistics, "Sum") {
			v := base
			point.Sum = &v
		}
		out = append(out, point)
	}
	if len(out) == 0 {
		out = append(out, DistributionMetricDatapoint{
			Timestamp: input.EndTime.UTC(),
			Unit:      input.Unit,
		})
	}
	return input.MetricName, out, nil
}

func (s *Service) GetDistributionBundles() []DistributionBundle {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]DistributionBundle, len(s.distributionBundles))
	copy(out, s.distributionBundles)
	sort.Slice(out, func(i, j int) bool { return out[i].BundleID < out[j].BundleID })
	return out
}

func (s *Service) GetDistributionLatestCacheReset(distributionName string) (DistributionCacheReset, bool, error) {
	distributionName = strings.TrimSpace(distributionName)

	s.mu.Lock()
	defer s.mu.Unlock()

	if distributionName != "" {
		if _, exists := s.distributions[distributionName]; !exists {
			return DistributionCacheReset{}, false, ErrNotFound
		}
		reset, exists := s.distributionCacheResets[distributionName]
		return reset, exists, nil
	}

	var latest DistributionCacheReset
	found := false
	for name, reset := range s.distributionCacheResets {
		if _, exists := s.distributions[name]; !exists {
			continue
		}
		if !found || reset.CreateTime.After(latest.CreateTime) {
			latest = reset
			found = true
		}
	}
	return latest, found, nil
}

func (s *Service) UpdateDistributionBundle(distributionName, bundleID string) (Operation, error) {
	distributionName = strings.TrimSpace(distributionName)
	bundleID = strings.TrimSpace(bundleID)
	if distributionName == "" || bundleID == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	dist, exists := s.distributions[distributionName]
	if !exists {
		return Operation{}, ErrNotFound
	}
	if !s.bundleExistsLocked(bundleID) {
		return Operation{}, ErrInvalidParameter
	}

	dist.BundleID = bundleID
	dist.AbleToUpdateBundle = true
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, distributionName, "Distribution", "UpdateDistributionBundle", "Succeeded", "distribution bundle updated", dist.AvailabilityZone, dist.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) CreateCertificate(certificateName, domainName string, subjectAlternativeNames []string, tags map[string]string) (Certificate, []Operation, error) {
	certificateName = strings.TrimSpace(certificateName)
	domainName = strings.TrimSpace(domainName)
	if certificateName == "" || domainName == "" {
		return Certificate{}, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.certificates[certificateName]; exists {
		return Certificate{}, nil, ErrAlreadyExists
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := DefaultRegion
	availabilityZone := region + "a"
	certificate := &Certificate{
		Name:                    certificateName,
		ARN:                     certificateARN(region, certificateName),
		CreatedAt:               now,
		DomainName:              domainName,
		DomainValidationRecords: []CertificateDomainValidationRecord{},
		EligibleToRenew:         "Yes",
		InUseResourceCount:      0,
		IssuedAt:                now,
		IssuerCA:                "Amazon",
		KeyAlgorithm:            "RSA-2048",
		NotAfter:                now.Add(365 * 24 * time.Hour),
		NotBefore:               now,
		RenewalSummary:          CertificateRenewalSummary{},
		RequestFailureReason:    "",
		RevocationReason:        "",
		RevokedAt:               time.Time{},
		SerialNumber:            fmt.Sprintf("stackyard-%012d", seq),
		Status:                  "ISSUED",
		SubjectAlternativeNames: dedupeStrings(subjectAlternativeNames),
		SupportCode:             fmt.Sprintf("%s/%d", region, seq),
		Tags:                    cloneStringMap(tags),
		AttachedDistributions:   []string{},
	}
	s.certificates[certificateName] = certificate

	ops := []Operation{
		newOperation(seq, certificateName, "Certificate", "CreateCertificate", "Succeeded", "certificate created", availabilityZone, region, now),
	}
	s.appendOperationsLocked(ops)
	return cloneCertificate(certificate), ops, nil
}

func (s *Service) GetCertificates(certificateName string, certificateStatuses []string) []Certificate {
	certificateName = strings.TrimSpace(certificateName)
	normalizedStatuses := map[string]struct{}{}
	for _, status := range certificateStatuses {
		status = strings.ToUpper(strings.TrimSpace(status))
		if status == "" {
			continue
		}
		normalizedStatuses[status] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if certificateName != "" {
		certificate, exists := s.certificates[certificateName]
		if !exists {
			return []Certificate{}
		}
		if len(normalizedStatuses) > 0 {
			if _, ok := normalizedStatuses[strings.ToUpper(certificate.Status)]; !ok {
				return []Certificate{}
			}
		}
		return []Certificate{cloneCertificate(certificate)}
	}

	out := make([]Certificate, 0, len(s.certificates))
	for _, certificate := range s.certificates {
		if len(normalizedStatuses) > 0 {
			if _, ok := normalizedStatuses[strings.ToUpper(certificate.Status)]; !ok {
				continue
			}
		}
		out = append(out, cloneCertificate(certificate))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) DeleteCertificate(certificateName string) ([]Operation, error) {
	certificateName = strings.TrimSpace(certificateName)
	if certificateName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	certificate, exists := s.certificates[certificateName]
	if !exists {
		return nil, ErrNotFound
	}
	if len(certificate.AttachedDistributions) > 0 || certificate.InUseResourceCount > 0 {
		return nil, ErrInvalidParameter
	}
	delete(s.certificates, certificateName)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := DefaultRegion
	availabilityZone := region + "a"
	ops := []Operation{
		newOperation(seq, certificateName, "Certificate", "DeleteCertificate", "Succeeded", "certificate deleted", availabilityZone, region, now),
	}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) AttachCertificateToDistribution(certificateName, distributionName string) (Operation, error) {
	certificateName = strings.TrimSpace(certificateName)
	distributionName = strings.TrimSpace(distributionName)
	if certificateName == "" || distributionName == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	certificate, certExists := s.certificates[certificateName]
	if !certExists {
		return Operation{}, ErrNotFound
	}
	if !strings.EqualFold(strings.TrimSpace(certificate.Status), "ISSUED") {
		return Operation{}, ErrInvalidParameter
	}
	distribution, distExists := s.distributions[distributionName]
	if !distExists {
		return Operation{}, ErrNotFound
	}

	previousCertificateName := strings.TrimSpace(distribution.CertificateName)
	if previousCertificateName != "" && previousCertificateName != certificateName {
		if previous, ok := s.certificates[previousCertificateName]; ok {
			previous.AttachedDistributions = removeString(previous.AttachedDistributions, distributionName)
			previous.InUseResourceCount = int32(len(previous.AttachedDistributions))
		}
	}
	distribution.CertificateName = certificateName
	if !containsString(certificate.AttachedDistributions, distributionName) {
		certificate.AttachedDistributions = append(certificate.AttachedDistributions, distributionName)
		sort.Strings(certificate.AttachedDistributions)
	}
	certificate.InUseResourceCount = int32(len(certificate.AttachedDistributions))

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, distributionName, "Distribution", "AttachCertificateToDistribution", "Succeeded", "certificate attached to distribution", distribution.AvailabilityZone, distribution.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) DetachCertificateFromDistribution(distributionName string) (Operation, error) {
	distributionName = strings.TrimSpace(distributionName)
	if distributionName == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	distribution, distExists := s.distributions[distributionName]
	if !distExists {
		return Operation{}, ErrNotFound
	}
	currentCertificateName := strings.TrimSpace(distribution.CertificateName)
	if currentCertificateName != "" {
		if certificate, ok := s.certificates[currentCertificateName]; ok {
			certificate.AttachedDistributions = removeString(certificate.AttachedDistributions, distributionName)
			certificate.InUseResourceCount = int32(len(certificate.AttachedDistributions))
		}
	}
	distribution.CertificateName = ""

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, distributionName, "Distribution", "DetachCertificateFromDistribution", "Succeeded", "certificate detached from distribution", distribution.AvailabilityZone, distribution.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) SetIPAddressType(resourceName, resourceType, ipAddressType string, acceptBundleUpdate *bool) ([]Operation, error) {
	resourceName = strings.TrimSpace(resourceName)
	resourceType = strings.TrimSpace(resourceType)
	ipAddressType = strings.ToLower(strings.TrimSpace(ipAddressType))
	if resourceName == "" || resourceType == "" || ipAddressType == "" {
		return nil, ErrInvalidParameter
	}
	if ipAddressType != "ipv4" && ipAddressType != "ipv6" && ipAddressType != "dualstack" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()

	switch strings.ToLower(resourceType) {
	case "distribution":
		distribution, exists := s.distributions[resourceName]
		if !exists {
			return nil, ErrNotFound
		}
		distribution.IPAddressType = ipAddressType
		ops := []Operation{
			newOperation(seq, resourceName, "Distribution", "SetIpAddressType", "Succeeded", "distribution ip address type updated", distribution.AvailabilityZone, distribution.Region, now),
		}
		s.appendOperationsLocked(ops)
		return ops, nil
	case "instance":
		instance, exists := s.instances[resourceName]
		if !exists {
			return nil, ErrNotFound
		}
		current := strings.ToLower(firstNonEmptyString(instance.IPAddressType, inferInstanceIPAddressType(instance)))
		if (ipAddressType == "ipv6" || current == "ipv6") && (acceptBundleUpdate == nil || !*acceptBundleUpdate) && ipAddressType != current {
			return nil, ErrInvalidParameter
		}
		switch ipAddressType {
		case "ipv4":
			instance.IPv6Addresses = []string{}
			if strings.TrimSpace(instance.PublicIPAddress) == "" {
				instance.PublicIPAddress = fmt.Sprintf("203.0.113.%d", (seq%250)+1)
			}
		case "ipv6":
			instance.PublicIPAddress = ""
			if len(instance.IPv6Addresses) == 0 {
				instance.IPv6Addresses = []string{fmt.Sprintf("2001:db8::%x", (seq%65535)+1)}
			}
		case "dualstack":
			if strings.TrimSpace(instance.PublicIPAddress) == "" {
				instance.PublicIPAddress = fmt.Sprintf("203.0.113.%d", (seq%250)+1)
			}
			if len(instance.IPv6Addresses) == 0 {
				instance.IPv6Addresses = []string{fmt.Sprintf("2001:db8::%x", (seq%65535)+1)}
			}
		}
		instance.IPAddressType = ipAddressType
		ops := []Operation{
			newOperation(seq, resourceName, "Instance", "SetIpAddressType", "Succeeded", "instance ip address type updated", instance.AvailabilityZone, instance.Region, now),
		}
		s.appendOperationsLocked(ops)
		return ops, nil
	case "loadbalancer":
		return nil, ErrNotFound
	default:
		return nil, ErrInvalidParameter
	}
}

func (s *Service) CreateDomain(domainName string, tags map[string]string) (Operation, error) {
	domainName = strings.TrimSpace(domainName)
	if domainName == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.domains[domainName]; exists {
		return Operation{}, ErrAlreadyExists
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := DefaultRegion
	availabilityZone := region + "a"
	s.domains[domainName] = &Domain{
		Name:             domainName,
		ARN:              domainARN(region, domainName),
		CreatedAt:        now,
		DomainEntries:    []DomainEntry{},
		AvailabilityZone: availabilityZone,
		Region:           region,
		ResourceType:     "Domain",
		SupportCode:      fmt.Sprintf("%s/%d", region, seq),
		Tags:             cloneStringMap(tags),
	}

	op := newOperation(seq, domainName, "Domain", "CreateDomain", "Succeeded", "domain created", availabilityZone, region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) GetDomain(domainName string) (Domain, bool) {
	domainName = strings.TrimSpace(domainName)
	if domainName == "" {
		return Domain{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	domain, exists := s.domains[domainName]
	if !exists {
		return Domain{}, false
	}
	return cloneDomain(domain), true
}

func (s *Service) GetDomains() []Domain {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Domain, 0, len(s.domains))
	for _, domain := range s.domains {
		out = append(out, cloneDomain(domain))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) DeleteDomain(domainName string) (Operation, error) {
	domainName = strings.TrimSpace(domainName)
	if domainName == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	domain, exists := s.domains[domainName]
	if !exists {
		return Operation{}, ErrNotFound
	}
	delete(s.domains, domainName)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, domainName, "Domain", "DeleteDomain", "Succeeded", "domain deleted", domain.AvailabilityZone, domain.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) CreateDomainEntry(domainName string, entry DomainEntry) (Operation, error) {
	domainName = strings.TrimSpace(domainName)
	entry = normalizeDomainEntry(entry)
	if domainName == "" || entry.Name == "" || entry.Type == "" || entry.Target == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	domain, exists := s.domains[domainName]
	if !exists {
		return Operation{}, ErrNotFound
	}
	if idx := findDomainEntryIndex(domain.DomainEntries, entry); idx >= 0 {
		return Operation{}, ErrAlreadyExists
	}
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("de-%d", atomic.AddUint64(&s.seq, 1))
	}
	domain.DomainEntries = append(domain.DomainEntries, entry)
	sortDomainEntries(domain.DomainEntries)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, domainName, "Domain", "CreateDomainEntry", "Succeeded", "domain entry created", domain.AvailabilityZone, domain.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) UpdateDomainEntry(domainName string, entry DomainEntry) ([]Operation, error) {
	domainName = strings.TrimSpace(domainName)
	entry = normalizeDomainEntry(entry)
	if domainName == "" || entry.Name == "" || entry.Type == "" || entry.Target == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	domain, exists := s.domains[domainName]
	if !exists {
		return nil, ErrNotFound
	}
	idx := findDomainEntryIndex(domain.DomainEntries, entry)
	if idx < 0 {
		return nil, ErrNotFound
	}
	if entry.ID == "" {
		entry.ID = domain.DomainEntries[idx].ID
	}
	domain.DomainEntries[idx] = entry
	sortDomainEntries(domain.DomainEntries)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{
		newOperation(seq, domainName, "Domain", "UpdateDomainEntry", "Succeeded", "domain entry updated", domain.AvailabilityZone, domain.Region, now),
	}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) DeleteDomainEntry(domainName string, entry DomainEntry) (Operation, error) {
	domainName = strings.TrimSpace(domainName)
	entry = normalizeDomainEntry(entry)
	if domainName == "" || entry.Name == "" || entry.Type == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	domain, exists := s.domains[domainName]
	if !exists {
		return Operation{}, ErrNotFound
	}
	idx := findDomainEntryIndex(domain.DomainEntries, entry)
	if idx < 0 {
		return Operation{}, ErrNotFound
	}
	domain.DomainEntries = append(domain.DomainEntries[:idx], domain.DomainEntries[idx+1:]...)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, domainName, "Domain", "DeleteDomainEntry", "Succeeded", "domain entry deleted", domain.AvailabilityZone, domain.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) CreateBucket(bucketName, bundleID string, enableObjectVersioning *bool, tags map[string]string) (Bucket, []Operation, error) {
	bucketName = strings.TrimSpace(bucketName)
	bundleID = strings.TrimSpace(bundleID)
	if bucketName == "" || bundleID == "" {
		return Bucket{}, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.buckets[bucketName]; exists {
		return Bucket{}, nil, ErrAlreadyExists
	}
	if !s.bucketBundleExistsLocked(bundleID, false) {
		return Bucket{}, nil, ErrInvalidParameter
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := DefaultRegion
	availabilityZone := region + "a"
	objectVersioning := "NeverEnabled"
	if enableObjectVersioning != nil && *enableObjectVersioning {
		objectVersioning = "Enabled"
	}

	bucket := &Bucket{
		Name:               bucketName,
		ARN:                bucketARN(region, bucketName),
		BundleID:           bundleID,
		CreatedAt:          now,
		AvailabilityZone:   availabilityZone,
		Region:             region,
		AbleToUpdateBundle: true,
		ObjectVersioning:   objectVersioning,
		AccessRules: &BucketAccessRules{
			AllowPublicOverrides: false,
			GetObject:            "private",
		},
		ReadonlyAccessAccounts:   []string{},
		ResourcesReceivingAccess: []BucketResourceReceivingAccess{},
		ResourceType:             "Bucket",
		State: BucketState{
			Code:    "OK",
			Message: "Bucket is ready.",
		},
		SupportCode: fmt.Sprintf("%s/%d", region, seq),
		URL:         fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucketName, region),
		Tags:        cloneStringMap(tags),
	}
	s.buckets[bucketName] = bucket
	s.bucketAccessKeys[bucketName] = map[string]*BucketAccessKey{}

	ops := []Operation{
		newOperation(seq, bucketName, "Bucket", "CreateBucket", "Succeeded", "bucket created", availabilityZone, region, now),
	}
	s.appendOperationsLocked(ops)
	return cloneBucket(bucket), ops, nil
}

func (s *Service) GetBuckets(bucketName string, includeConnectedResources bool) []Bucket {
	bucketName = strings.TrimSpace(bucketName)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Bucket, 0, len(s.buckets))
	if bucketName != "" {
		bucket, exists := s.buckets[bucketName]
		if !exists {
			return []Bucket{}
		}
		cloned := cloneBucket(bucket)
		if !includeConnectedResources {
			cloned.ResourcesReceivingAccess = []BucketResourceReceivingAccess{}
		}
		return []Bucket{cloned}
	}

	for _, bucket := range s.buckets {
		cloned := cloneBucket(bucket)
		if !includeConnectedResources {
			cloned.ResourcesReceivingAccess = []BucketResourceReceivingAccess{}
		}
		out = append(out, cloned)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) UpdateBucket(input BucketUpdateInput) (Bucket, []Operation, error) {
	input.BucketName = strings.TrimSpace(input.BucketName)
	if input.BucketName == "" {
		return Bucket{}, nil, ErrInvalidParameter
	}

	var normalizedVersioning string
	if input.Versioning != nil {
		normalizedVersioning = normalizeBucketVersioning(*input.Versioning)
		if normalizedVersioning == "" {
			return Bucket{}, nil, ErrInvalidParameter
		}
	}

	var normalizedAccessRules *BucketAccessRules
	if input.AccessRules != nil {
		getObject := normalizeBucketAccessType(input.AccessRules.GetObject)
		if getObject == "" {
			return Bucket{}, nil, ErrInvalidParameter
		}
		normalizedAccessRules = &BucketAccessRules{
			AllowPublicOverrides: input.AccessRules.AllowPublicOverrides,
			GetObject:            getObject,
		}
	}

	var normalizedAccessLogConfig *BucketAccessLogConfig
	if input.AccessLogConfig != nil {
		destination := strings.TrimSpace(input.AccessLogConfig.Destination)
		prefix := strings.TrimSpace(input.AccessLogConfig.Prefix)
		if input.AccessLogConfig.Enabled && destination == "" {
			return Bucket{}, nil, ErrInvalidParameter
		}
		if !input.AccessLogConfig.Enabled {
			destination = ""
			prefix = ""
		}
		normalizedAccessLogConfig = &BucketAccessLogConfig{
			Enabled:     input.AccessLogConfig.Enabled,
			Destination: destination,
			Prefix:      prefix,
		}
	}

	var normalizedReadonlyAccounts []string
	if input.HasReadonlyAccessAccounts {
		normalizedReadonlyAccounts = dedupeStrings(input.ReadonlyAccessAccounts)
		if len(normalizedReadonlyAccounts) > 10 {
			return Bucket{}, nil, ErrInvalidParameter
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, exists := s.buckets[input.BucketName]
	if !exists {
		return Bucket{}, nil, ErrNotFound
	}

	if normalizedAccessRules != nil {
		bucket.AccessRules = normalizedAccessRules
	}
	if normalizedAccessLogConfig != nil {
		bucket.AccessLogConfig = normalizedAccessLogConfig
	}
	if input.HasReadonlyAccessAccounts {
		bucket.ReadonlyAccessAccounts = append([]string(nil), normalizedReadonlyAccounts...)
	}
	if input.Versioning != nil {
		bucket.ObjectVersioning = normalizedVersioning
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{
		newOperation(seq, input.BucketName, "Bucket", "UpdateBucket", "Succeeded", "bucket updated", bucket.AvailabilityZone, bucket.Region, now),
	}
	s.appendOperationsLocked(ops)
	return cloneBucket(bucket), ops, nil
}

func (s *Service) DeleteBucket(bucketName string, forceDelete bool) ([]Operation, error) {
	bucketName = strings.TrimSpace(bucketName)
	if bucketName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, exists := s.buckets[bucketName]
	if !exists {
		return nil, ErrNotFound
	}
	if !forceDelete && len(bucket.ResourcesReceivingAccess) > 0 {
		return nil, ErrInvalidParameter
	}

	delete(s.buckets, bucketName)
	delete(s.bucketAccessKeys, bucketName)
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{
		newOperation(seq, bucketName, "Bucket", "DeleteBucket", "Succeeded", "bucket deleted", bucket.AvailabilityZone, bucket.Region, now),
	}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) SetResourceAccessForBucket(bucketName, resourceName, access string) ([]Operation, error) {
	bucketName = strings.TrimSpace(bucketName)
	resourceName = strings.TrimSpace(resourceName)
	access = strings.ToLower(strings.TrimSpace(access))
	if bucketName == "" || resourceName == "" {
		return nil, ErrInvalidParameter
	}
	if access != "allow" && access != "deny" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, exists := s.buckets[bucketName]
	if !exists {
		return nil, ErrNotFound
	}
	instance, exists := s.instances[resourceName]
	if !exists {
		return nil, ErrNotFound
	}

	idx := -1
	for i := range bucket.ResourcesReceivingAccess {
		if bucket.ResourcesReceivingAccess[i].Name == resourceName {
			idx = i
			break
		}
	}
	if access == "allow" {
		if idx < 0 {
			bucket.ResourcesReceivingAccess = append(bucket.ResourcesReceivingAccess, BucketResourceReceivingAccess{
				Name:         resourceName,
				ResourceType: "Instance",
			})
		}
	} else if idx >= 0 {
		bucket.ResourcesReceivingAccess = append(bucket.ResourcesReceivingAccess[:idx], bucket.ResourcesReceivingAccess[idx+1:]...)
	}
	sort.Slice(bucket.ResourcesReceivingAccess, func(i, j int) bool {
		return bucket.ResourcesReceivingAccess[i].Name < bucket.ResourcesReceivingAccess[j].Name
	})

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{
		newOperation(seq, bucketName, "Bucket", "SetResourceAccessForBucket", "Succeeded", "bucket resource access updated", instance.AvailabilityZone, instance.Region, now),
	}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetBucketBundles(includeInactive bool) []BucketBundle {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]BucketBundle, 0, len(s.bucketBundles))
	for _, bundle := range s.bucketBundles {
		if !includeInactive && !bundle.IsActive {
			continue
		}
		out = append(out, bundle)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BundleID < out[j].BundleID })
	return out
}

func (s *Service) GetBucketMetricData(input BucketMetricInput) (string, []BucketMetricDatapoint, error) {
	input.BucketName = strings.TrimSpace(input.BucketName)
	input.MetricName = normalizeBucketMetricName(input.MetricName)
	input.Unit = strings.TrimSpace(input.Unit)
	if input.BucketName == "" || input.MetricName == "" || input.Unit == "" || input.Period <= 0 || input.StartTime.IsZero() || input.EndTime.IsZero() || input.EndTime.Before(input.StartTime) || len(input.Statistics) == 0 {
		return "", nil, ErrInvalidParameter
	}
	if !hasAnyMetricStatistic(input.Statistics) {
		return "", nil, ErrInvalidParameter
	}
	if !validBucketMetricUnit(input.MetricName, input.Unit) {
		return "", nil, ErrInvalidParameter
	}

	s.mu.Lock()
	_, exists := s.buckets[input.BucketName]
	s.mu.Unlock()
	if !exists {
		return "", nil, ErrNotFound
	}

	step := time.Duration(input.Period) * time.Second
	count := int(input.EndTime.Sub(input.StartTime)/step) + 1
	if count < 1 {
		count = 1
	}
	if count > 1440 {
		count = 1440
	}

	scale := float64(1)
	if strings.EqualFold(input.MetricName, "BucketSizeBytes") {
		scale = 1024 * 1024
	}
	out := make([]BucketMetricDatapoint, 0, count)
	for i := 0; i < count; i++ {
		ts := input.StartTime.Add(time.Duration(i) * step)
		if ts.After(input.EndTime) {
			break
		}
		base := float64(i+1) * scale
		point := BucketMetricDatapoint{
			Timestamp: ts.UTC(),
			Unit:      input.Unit,
		}
		if hasMetricStatistic(input.Statistics, "Average") {
			v := base
			point.Average = &v
		}
		if hasMetricStatistic(input.Statistics, "Maximum") {
			v := base + (0.5 * scale)
			point.Maximum = &v
		}
		if hasMetricStatistic(input.Statistics, "Minimum") {
			v := base - (0.5 * scale)
			point.Minimum = &v
		}
		if hasMetricStatistic(input.Statistics, "SampleCount") {
			v := float64(1)
			point.SampleCount = &v
		}
		if hasMetricStatistic(input.Statistics, "Sum") {
			v := base
			point.Sum = &v
		}
		out = append(out, point)
	}
	if len(out) == 0 {
		out = append(out, BucketMetricDatapoint{
			Timestamp: input.EndTime.UTC(),
			Unit:      input.Unit,
		})
	}
	return input.MetricName, out, nil
}

func (s *Service) UpdateBucketBundle(bucketName, bundleID string) ([]Operation, error) {
	bucketName = strings.TrimSpace(bucketName)
	bundleID = strings.TrimSpace(bundleID)
	if bucketName == "" || bundleID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, exists := s.buckets[bucketName]
	if !exists {
		return nil, ErrNotFound
	}
	if !s.bucketBundleExistsLocked(bundleID, false) {
		return nil, ErrInvalidParameter
	}

	bucket.BundleID = bundleID
	bucket.AbleToUpdateBundle = true
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{
		newOperation(seq, bucketName, "Bucket", "UpdateBucketBundle", "Succeeded", "bucket bundle updated", bucket.AvailabilityZone, bucket.Region, now),
	}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) CreateBucketAccessKey(bucketName string) (BucketAccessKey, []Operation, error) {
	bucketName = strings.TrimSpace(bucketName)
	if bucketName == "" {
		return BucketAccessKey{}, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, exists := s.buckets[bucketName]
	if !exists {
		return BucketAccessKey{}, nil, ErrNotFound
	}
	if s.bucketAccessKeys[bucketName] == nil {
		s.bucketAccessKeys[bucketName] = map[string]*BucketAccessKey{}
	}
	if len(s.bucketAccessKeys[bucketName]) >= 2 {
		return BucketAccessKey{}, nil, ErrInvalidParameter
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	accessKeyID := fmt.Sprintf("LSIA%016X", seq)
	accessKey := &BucketAccessKey{
		AccessKeyID:     accessKeyID,
		CreatedAt:       now,
		LastUsed:        nil,
		SecretAccessKey: encodeKeyMaterial(fmt.Sprintf("stackyard-bucket-%s-secret-%d", bucketName, seq)),
		Status:          "Active",
	}
	s.bucketAccessKeys[bucketName][accessKeyID] = accessKey

	ops := []Operation{
		newOperation(seq, bucketName, "BucketAccessKey", "CreateBucketAccessKey", "Succeeded", "bucket access key created", bucket.AvailabilityZone, bucket.Region, now),
	}
	s.appendOperationsLocked(ops)
	return cloneBucketAccessKey(accessKey), ops, nil
}

func (s *Service) GetBucketAccessKeys(bucketName string) ([]BucketAccessKey, error) {
	bucketName = strings.TrimSpace(bucketName)
	if bucketName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.buckets[bucketName]; !exists {
		return nil, ErrNotFound
	}
	out := make([]BucketAccessKey, 0, len(s.bucketAccessKeys[bucketName]))
	for _, accessKey := range s.bucketAccessKeys[bucketName] {
		cloned := cloneBucketAccessKey(accessKey)
		// SecretAccessKey is returned only by CreateBucketAccessKey.
		cloned.SecretAccessKey = ""
		out = append(out, cloned)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AccessKeyID < out[j].AccessKeyID })
	return out, nil
}

func (s *Service) DeleteBucketAccessKey(bucketName, accessKeyID string) ([]Operation, error) {
	bucketName = strings.TrimSpace(bucketName)
	accessKeyID = strings.TrimSpace(accessKeyID)
	if bucketName == "" || accessKeyID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, exists := s.buckets[bucketName]
	if !exists {
		return nil, ErrNotFound
	}
	if s.bucketAccessKeys[bucketName] == nil {
		return nil, ErrNotFound
	}
	if _, exists := s.bucketAccessKeys[bucketName][accessKeyID]; !exists {
		return nil, ErrNotFound
	}
	delete(s.bucketAccessKeys[bucketName], accessKeyID)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{
		newOperation(seq, bucketName, "BucketAccessKey", "DeleteBucketAccessKey", "Succeeded", "bucket access key deleted", bucket.AvailabilityZone, bucket.Region, now),
	}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) CreateContactMethod(contactEndpoint, protocol string) ([]Operation, error) {
	contactEndpoint = strings.TrimSpace(contactEndpoint)
	protocol, ok := normalizeContactProtocol(protocol)
	if contactEndpoint == "" || !ok {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.ToLower(protocol)
	if _, exists := s.contactMethods[key]; exists {
		return nil, ErrAlreadyExists
	}
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := DefaultRegion
	availabilityZone := region + "a"
	contactMethod := &ContactMethod{
		ARN:              contactMethodARN(region, key),
		ContactEndpoint:  contactEndpoint,
		CreatedAt:        now,
		AvailabilityZone: availabilityZone,
		Region:           region,
		Name:             protocol,
		Protocol:         protocol,
		ResourceType:     "ContactMethod",
		Status:           "PendingVerification",
		SupportCode:      fmt.Sprintf("%s/%d", region, seq),
	}
	s.contactMethods[key] = contactMethod

	ops := []Operation{
		newOperation(seq, protocol, "ContactMethod", "CreateContactMethod", "Succeeded", "contact method created", availabilityZone, region, now),
	}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetContactMethods(protocols []string) []ContactMethod {
	s.mu.Lock()
	defer s.mu.Unlock()

	filter := map[string]struct{}{}
	for _, protocol := range protocols {
		normalized, ok := normalizeContactProtocol(protocol)
		if !ok {
			continue
		}
		filter[strings.ToLower(normalized)] = struct{}{}
	}

	out := make([]ContactMethod, 0, len(s.contactMethods))
	for key, contactMethod := range s.contactMethods {
		if len(filter) > 0 {
			if _, keep := filter[key]; !keep {
				continue
			}
		}
		out = append(out, cloneContactMethod(contactMethod))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Protocol < out[j].Protocol })
	return out
}

func (s *Service) DeleteContactMethod(protocol string) ([]Operation, error) {
	protocol, ok := normalizeContactProtocol(protocol)
	if !ok {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.ToLower(protocol)
	contactMethod, exists := s.contactMethods[key]
	if !exists {
		return nil, ErrNotFound
	}
	delete(s.contactMethods, key)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{
		newOperation(seq, protocol, "ContactMethod", "DeleteContactMethod", "Succeeded", "contact method deleted", contactMethod.AvailabilityZone, contactMethod.Region, now),
	}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) SendContactMethodVerification(protocol string) ([]Operation, error) {
	protocol, ok := normalizeContactProtocol(protocol)
	if !ok {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.ToLower(protocol)
	contactMethod, exists := s.contactMethods[key]
	if !exists {
		return nil, ErrNotFound
	}
	contactMethod.Status = "Valid"

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{
		newOperation(seq, protocol, "ContactMethod", "SendContactMethodVerification", "Succeeded", "contact method verification sent", contactMethod.AvailabilityZone, contactMethod.Region, now),
	}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) CreateContainerService(serviceName, power string, scale int32, publicDomainNames map[string][]string, tags map[string]string) (ContainerService, error) {
	serviceName = strings.TrimSpace(serviceName)
	power, ok := normalizeContainerServicePowerName(power)
	if serviceName == "" || !ok || scale <= 0 {
		return ContainerService{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.containerServices[serviceName]; exists {
		return ContainerService{}, ErrAlreadyExists
	}
	powerSpec, ok := s.containerServicePowerByNameLocked(power)
	if !ok || !powerSpec.IsActive {
		return ContainerService{}, ErrInvalidParameter
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := DefaultRegion
	availabilityZone := region + "a"
	containerService := &ContainerService{
		Name:              serviceName,
		ARN:               containerServiceARN(region, serviceName),
		CreatedAt:         now,
		AvailabilityZone:  availabilityZone,
		Region:            region,
		Power:             power,
		PowerID:           powerSpec.PowerID,
		Scale:             scale,
		IsDisabled:        false,
		PrincipalARN:      fmt.Sprintf("arn:aws:iam::%s:role/lightsail/container-service/%s", DefaultAccountID, serviceName),
		PrivateDomainName: fmt.Sprintf("%s.service.local", serviceName),
		PublicDomainNames: cloneStringSliceMap(publicDomainNames),
		ResourceType:      "ContainerService",
		State:             "READY",
		SupportCode:       fmt.Sprintf("%s/%d", region, seq),
		URL:               fmt.Sprintf("https://%s.%s.cs.amazonlightsail.com", serviceName, region),
		Tags:              cloneStringMap(tags),
	}
	s.containerServices[serviceName] = containerService

	op := newOperation(seq, serviceName, "ContainerService", "CreateContainerService", "Succeeded", "container service created", availabilityZone, region, now)
	s.appendOperationsLocked([]Operation{op})
	return cloneContainerService(containerService), nil
}

func (s *Service) GetContainerServices(serviceName string) []ContainerService {
	serviceName = strings.TrimSpace(serviceName)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]ContainerService, 0, len(s.containerServices))
	if serviceName != "" {
		containerService, exists := s.containerServices[serviceName]
		if !exists {
			return []ContainerService{}
		}
		return []ContainerService{cloneContainerService(containerService)}
	}
	for _, containerService := range s.containerServices {
		out = append(out, cloneContainerService(containerService))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) UpdateContainerService(input ContainerServiceUpdateInput) (ContainerService, error) {
	input.ServiceName = strings.TrimSpace(input.ServiceName)
	if input.ServiceName == "" {
		return ContainerService{}, ErrInvalidParameter
	}
	if input.Power != nil {
		power, ok := normalizeContainerServicePowerName(*input.Power)
		if !ok {
			return ContainerService{}, ErrInvalidParameter
		}
		input.Power = &power
	}
	if input.Scale != nil && *input.Scale <= 0 {
		return ContainerService{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	containerService, exists := s.containerServices[input.ServiceName]
	if !exists {
		return ContainerService{}, ErrNotFound
	}
	if input.Power != nil {
		powerSpec, ok := s.containerServicePowerByNameLocked(*input.Power)
		if !ok || !powerSpec.IsActive {
			return ContainerService{}, ErrInvalidParameter
		}
		containerService.Power = *input.Power
		containerService.PowerID = powerSpec.PowerID
	}
	if input.Scale != nil {
		containerService.Scale = *input.Scale
	}
	if input.IsDisabled != nil {
		containerService.IsDisabled = *input.IsDisabled
		if *input.IsDisabled {
			containerService.State = "DISABLED"
		} else {
			containerService.State = "READY"
		}
	}
	if input.HasPublicDomainNames {
		containerService.PublicDomainNames = cloneStringSliceMap(input.PublicDomainNames)
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, input.ServiceName, "ContainerService", "UpdateContainerService", "Succeeded", "container service updated", containerService.AvailabilityZone, containerService.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return cloneContainerService(containerService), nil
}

func (s *Service) DeleteContainerService(serviceName string) error {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	containerService, exists := s.containerServices[serviceName]
	if !exists {
		return ErrNotFound
	}
	delete(s.containerServices, serviceName)
	delete(s.containerDeployments, serviceName)
	delete(s.containerImages, serviceName)
	delete(s.containerImageVersions, serviceName)
	delete(s.containerLogs, serviceName)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, serviceName, "ContainerService", "DeleteContainerService", "Succeeded", "container service deleted", containerService.AvailabilityZone, containerService.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return nil
}

func (s *Service) GetContainerAPIMetadata() []map[string]string {
	return []map[string]string{
		{"lightsailctl": "v1.0.5"},
		{"apiVersion": "2023-11-28"},
	}
}

func (s *Service) GetContainerServiceMetricData(input ContainerServiceMetricInput) (string, []ContainerServiceMetricDatapoint, error) {
	input.ServiceName = strings.TrimSpace(input.ServiceName)
	input.MetricName = normalizeContainerServiceMetricName(input.MetricName)
	if input.ServiceName == "" || input.MetricName == "" || input.Period <= 0 || input.StartTime.IsZero() || input.EndTime.IsZero() || input.EndTime.Before(input.StartTime) || len(input.Statistics) == 0 {
		return "", nil, ErrInvalidParameter
	}
	if !hasAnyMetricStatistic(input.Statistics) {
		return "", nil, ErrInvalidParameter
	}

	s.mu.Lock()
	_, exists := s.containerServices[input.ServiceName]
	s.mu.Unlock()
	if !exists {
		return "", nil, ErrNotFound
	}

	step := time.Duration(input.Period) * time.Second
	count := int(input.EndTime.Sub(input.StartTime)/step) + 1
	if count < 1 {
		count = 1
	}
	if count > 288 {
		count = 288
	}

	out := make([]ContainerServiceMetricDatapoint, 0, count)
	for i := 0; i < count; i++ {
		ts := input.StartTime.Add(time.Duration(i) * step)
		if ts.After(input.EndTime) {
			break
		}
		base := float64((i%100)+1) / 2
		point := ContainerServiceMetricDatapoint{
			Timestamp: ts.UTC(),
			Unit:      "Percent",
		}
		if hasMetricStatistic(input.Statistics, "Average") {
			v := base
			point.Average = &v
		}
		if hasMetricStatistic(input.Statistics, "Maximum") {
			v := base + 5
			point.Maximum = &v
		}
		if hasMetricStatistic(input.Statistics, "Minimum") {
			v := base - 5
			point.Minimum = &v
		}
		if hasMetricStatistic(input.Statistics, "SampleCount") {
			v := float64(1)
			point.SampleCount = &v
		}
		if hasMetricStatistic(input.Statistics, "Sum") {
			v := base
			point.Sum = &v
		}
		out = append(out, point)
	}
	if len(out) == 0 {
		out = append(out, ContainerServiceMetricDatapoint{
			Timestamp: input.EndTime.UTC(),
			Unit:      "Percent",
		})
	}
	return input.MetricName, out, nil
}

func (s *Service) GetContainerServicePowers() []ContainerServicePower {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]ContainerServicePower, len(s.containerServicePowers))
	copy(out, s.containerServicePowers)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) CreateContainerServiceRegistryLogin() ContainerServiceRegistryLogin {
	now := time.Now().UTC()
	return ContainerServiceRegistryLogin{
		ExpiresAt: now.Add(12 * time.Hour),
		Password:  encodeKeyMaterial(fmt.Sprintf("stackyard-registry-password-%d", now.UnixNano())),
		Registry:  fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/lightsail", DefaultAccountID, DefaultRegion),
		Username:  "AWS",
	}
}

func (s *Service) CreateContainerServiceDeployment(serviceName string, containers map[string]ContainerServiceContainer, publicEndpoint *ContainerServiceEndpoint) (ContainerService, error) {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" || len(containers) == 0 {
		return ContainerService{}, ErrInvalidParameter
	}

	normalizedContainers := make(map[string]ContainerServiceContainer, len(containers))
	for name, container := range containers {
		name = strings.TrimSpace(name)
		if name == "" {
			return ContainerService{}, ErrInvalidParameter
		}
		container.Image = strings.TrimSpace(container.Image)
		if container.Image == "" {
			return ContainerService{}, ErrInvalidParameter
		}
		container.Command = dedupeStrings(container.Command)
		container.Environment = cloneStringMap(container.Environment)
		normalizedPorts := make(map[string]string, len(container.Ports))
		for key, protocol := range container.Ports {
			key = strings.TrimSpace(key)
			protocol, ok := normalizeContainerServiceProtocol(protocol)
			if key == "" || !ok {
				return ContainerService{}, ErrInvalidParameter
			}
			normalizedPorts[key] = protocol
		}
		container.Ports = normalizedPorts
		normalizedContainers[name] = container
	}

	if publicEndpoint != nil {
		publicEndpoint.ContainerName = strings.TrimSpace(publicEndpoint.ContainerName)
		if publicEndpoint.ContainerName == "" || publicEndpoint.ContainerPort <= 0 {
			return ContainerService{}, ErrInvalidParameter
		}
		if _, ok := normalizedContainers[publicEndpoint.ContainerName]; !ok {
			return ContainerService{}, ErrInvalidParameter
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	containerService, exists := s.containerServices[serviceName]
	if !exists {
		return ContainerService{}, ErrNotFound
	}

	deployments := s.containerDeployments[serviceName]
	for i := range deployments {
		if deployments[i].State == "ACTIVE" || deployments[i].State == "ACTIVATING" {
			deployments[i].State = "INACTIVE"
		}
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	deployment := ContainerServiceDeployment{
		Containers:     cloneContainerServiceContainers(normalizedContainers),
		CreatedAt:      now,
		PublicEndpoint: cloneContainerServiceEndpoint(publicEndpoint),
		State:          "ACTIVE",
		Version:        int32(len(deployments) + 1),
	}
	s.containerDeployments[serviceName] = append(deployments, deployment)
	containerService.State = "READY"
	for containerName := range deployment.Containers {
		s.appendContainerLogLocked(serviceName, containerName, fmt.Sprintf("deployment %d activated", deployment.Version), now)
	}

	op := newOperation(seq, serviceName, "ContainerService", "CreateContainerServiceDeployment", "Succeeded", "container service deployment created", containerService.AvailabilityZone, containerService.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return cloneContainerService(containerService), nil
}

func (s *Service) GetContainerServiceDeployments(serviceName string) ([]ContainerServiceDeployment, error) {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.containerServices[serviceName]; !exists {
		return nil, ErrNotFound
	}
	out := cloneContainerServiceDeployments(s.containerDeployments[serviceName])
	sort.Slice(out, func(i, j int) bool {
		if out[i].Version == out[j].Version {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].Version > out[j].Version
	})
	return out, nil
}

func (s *Service) RegisterContainerImage(serviceName, label, digest string) (ContainerImage, error) {
	serviceName = strings.TrimSpace(serviceName)
	label = strings.TrimSpace(label)
	digest = strings.TrimSpace(digest)
	if serviceName == "" || label == "" || digest == "" {
		return ContainerImage{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	containerService, exists := s.containerServices[serviceName]
	if !exists {
		return ContainerImage{}, ErrNotFound
	}
	if _, ok := s.containerImages[serviceName]; !ok {
		s.containerImages[serviceName] = map[string]*ContainerImage{}
	}
	if _, ok := s.containerImageVersions[serviceName]; !ok {
		s.containerImageVersions[serviceName] = map[string]int32{}
	}

	version := s.containerImageVersions[serviceName][label] + 1
	s.containerImageVersions[serviceName][label] = version
	imageName := fmt.Sprintf(":%s.%s.%d", serviceName, label, version)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	image := &ContainerImage{
		CreatedAt: now,
		Digest:    digest,
		Image:     imageName,
	}
	s.containerImages[serviceName][imageName] = image
	s.appendContainerLogLocked(serviceName, "registry", fmt.Sprintf("registered image %s", imageName), now)

	op := newOperation(seq, serviceName, "ContainerService", "RegisterContainerImage", "Succeeded", "container image registered", containerService.AvailabilityZone, containerService.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return cloneContainerImage(image), nil
}

func (s *Service) GetContainerImages(serviceName string) ([]ContainerImage, error) {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.containerServices[serviceName]; !exists {
		return nil, ErrNotFound
	}
	imageSet := s.containerImages[serviceName]
	out := make([]ContainerImage, 0, len(imageSet))
	for _, image := range imageSet {
		out = append(out, cloneContainerImage(image))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].Image < out[j].Image
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *Service) DeleteContainerImage(serviceName, image string) error {
	serviceName = strings.TrimSpace(serviceName)
	image = strings.TrimSpace(image)
	if serviceName == "" || image == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	containerService, exists := s.containerServices[serviceName]
	if !exists {
		return ErrNotFound
	}
	imageSet := s.containerImages[serviceName]
	if len(imageSet) == 0 {
		return ErrNotFound
	}
	if _, ok := imageSet[image]; !ok {
		return ErrNotFound
	}
	delete(imageSet, image)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	s.appendContainerLogLocked(serviceName, "registry", fmt.Sprintf("deleted image %s", image), now)
	op := newOperation(seq, serviceName, "ContainerService", "DeleteContainerImage", "Succeeded", "container image deleted", containerService.AvailabilityZone, containerService.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return nil
}

func (s *Service) GetContainerLog(input ContainerLogInput) ([]ContainerServiceLogEvent, string, error) {
	input.ServiceName = strings.TrimSpace(input.ServiceName)
	input.ContainerName = strings.TrimSpace(input.ContainerName)
	input.FilterPattern = strings.TrimSpace(input.FilterPattern)
	input.PageToken = strings.TrimSpace(input.PageToken)
	if input.ServiceName == "" || input.ContainerName == "" {
		return nil, "", ErrInvalidParameter
	}
	if input.StartTime != nil && input.EndTime != nil && input.EndTime.Before(*input.StartTime) {
		return nil, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.containerServices[input.ServiceName]; !exists {
		return nil, "", ErrNotFound
	}

	logsByContainer := s.containerLogs[input.ServiceName]
	events := cloneContainerServiceLogEvents(logsByContainer[input.ContainerName])
	containerKnown := len(events) > 0
	if !containerKnown {
		for _, deployment := range s.containerDeployments[input.ServiceName] {
			if _, ok := deployment.Containers[input.ContainerName]; ok {
				containerKnown = true
				break
			}
		}
	}
	if !containerKnown {
		return nil, "", ErrNotFound
	}

	filtered := make([]ContainerServiceLogEvent, 0, len(events))
	for _, event := range events {
		if input.StartTime != nil && event.CreatedAt.Before(*input.StartTime) {
			continue
		}
		if input.EndTime != nil && event.CreatedAt.After(*input.EndTime) {
			continue
		}
		if !matchesContainerLogFilter(event.Message, input.FilterPattern) {
			continue
		}
		filtered = append(filtered, event)
	}

	offset := 0
	if input.PageToken != "" {
		value, err := strconv.Atoi(input.PageToken)
		if err != nil || value < 0 {
			return nil, "", ErrInvalidParameter
		}
		offset = value
	}
	if offset >= len(filtered) {
		return []ContainerServiceLogEvent{}, "", nil
	}

	const pageSize = 100
	end := offset + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	nextPageToken := ""
	if end < len(filtered) {
		nextPageToken = strconv.Itoa(end)
	}
	return cloneContainerServiceLogEvents(filtered[offset:end]), nextPageToken, nil
}

func (s *Service) CreateRelationalDatabase(input RelationalDatabaseCreateInput) ([]Operation, error) {
	input.RelationalDatabaseName = strings.TrimSpace(input.RelationalDatabaseName)
	input.AvailabilityZone = strings.TrimSpace(input.AvailabilityZone)
	input.MasterDatabaseName = strings.TrimSpace(input.MasterDatabaseName)
	input.MasterUsername = strings.TrimSpace(input.MasterUsername)
	input.MasterUserPassword = strings.TrimSpace(input.MasterUserPassword)
	input.RelationalDatabaseBlueprintID = strings.TrimSpace(input.RelationalDatabaseBlueprintID)
	input.RelationalDatabaseBundleID = strings.TrimSpace(input.RelationalDatabaseBundleID)
	input.PreferredBackupWindow = strings.TrimSpace(input.PreferredBackupWindow)
	input.PreferredMaintenanceWindow = strings.TrimSpace(input.PreferredMaintenanceWindow)
	if input.RelationalDatabaseName == "" || input.MasterDatabaseName == "" || input.MasterUsername == "" || input.RelationalDatabaseBlueprintID == "" || input.RelationalDatabaseBundleID == "" {
		return nil, ErrInvalidParameter
	}
	if input.AvailabilityZone == "" {
		input.AvailabilityZone = DefaultRegion + "a"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.relationalDatabases[input.RelationalDatabaseName]; exists {
		return nil, ErrAlreadyExists
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := regionFromAvailabilityZone(input.AvailabilityZone)
	engine, engineVersion, port := relationalDatabaseEngineFromBlueprint(input.RelationalDatabaseBlueprintID)
	cpu, disk, ram := relationalDatabaseHardwareFromBundle(input.RelationalDatabaseBundleID)
	publiclyAccessible := false
	if input.PubliclyAccessible != nil {
		publiclyAccessible = *input.PubliclyAccessible
	}

	db := &RelationalDatabase{
		Name:                        input.RelationalDatabaseName,
		ARN:                         relationalDatabaseARN(region, input.RelationalDatabaseName),
		CreatedAt:                   now,
		AvailabilityZone:            input.AvailabilityZone,
		Region:                      region,
		BlueprintID:                 input.RelationalDatabaseBlueprintID,
		BundleID:                    input.RelationalDatabaseBundleID,
		Engine:                      engine,
		EngineVersion:               engineVersion,
		MasterDatabaseName:          input.MasterDatabaseName,
		MasterUsername:              input.MasterUsername,
		MasterUserPassword:          firstNonEmptyString(input.MasterUserPassword, fmt.Sprintf("Stackyard!%d", seq%100000)),
		MasterUserPasswordCreatedAt: now,
		MasterEndpointAddress:       fmt.Sprintf("%s.%s.rds.amazonaws.com", input.RelationalDatabaseName, region),
		MasterEndpointPort:          port,
		CPUCount:                    cpu,
		DiskSizeInGb:                disk,
		RAMSizeInGb:                 ram,
		LatestRestorableTime:        now,
		PreferredBackupWindow:       firstNonEmptyString(input.PreferredBackupWindow, "03:00-03:30"),
		PreferredMaintenanceWindow:  firstNonEmptyString(input.PreferredMaintenanceWindow, "Sun:04:00-Sun:04:30"),
		PubliclyAccessible:          publiclyAccessible,
		BackupRetentionEnabled:      true,
		CACertificateIdentifier:     "rds-ca-rsa2048-g1",
		ParameterApplyStatus:        "in-sync",
		SecondaryAvailabilityZone:   region + "b",
		State:                       "available",
		SupportCode:                 fmt.Sprintf("%s/%d", region, seq),
		PendingModifiedValues:       nil,
		Tags:                        cloneStringMap(input.Tags),
	}
	s.relationalDatabases[input.RelationalDatabaseName] = db
	s.relationalDatabaseParameters[input.RelationalDatabaseName] = defaultRelationalDatabaseParameters(db.Engine)
	s.relationalDatabaseLogStreams[input.RelationalDatabaseName] = defaultRelationalDatabaseLogStreams()
	s.relationalDatabaseLogEvents[input.RelationalDatabaseName] = defaultRelationalDatabaseLogEvents(input.RelationalDatabaseName, now)
	s.appendRelationalDatabaseEventLocked(input.RelationalDatabaseName, "creation", "created relational database", now)

	op := newOperation(seq, input.RelationalDatabaseName, "RelationalDatabase", "CreateRelationalDatabase", "Succeeded", "relational database created", db.AvailabilityZone, db.Region, now)
	ops := []Operation{op}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetRelationalDatabase(name string) (RelationalDatabase, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return RelationalDatabase{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	db, exists := s.relationalDatabases[name]
	if !exists {
		return RelationalDatabase{}, false
	}
	return cloneRelationalDatabase(db), true
}

func (s *Service) GetRelationalDatabases(pageToken string) (RelationalDatabasesPage, error) {
	pageToken = strings.TrimSpace(pageToken)
	offset := 0
	if pageToken != "" {
		value, err := strconv.Atoi(pageToken)
		if err != nil || value < 0 {
			return RelationalDatabasesPage{}, ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]RelationalDatabase, 0, len(s.relationalDatabases))
	for _, db := range s.relationalDatabases {
		items = append(items, cloneRelationalDatabase(db))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	if offset >= len(items) {
		return RelationalDatabasesPage{RelationalDatabases: []RelationalDatabase{}}, nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(items) {
		end = len(items)
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	return RelationalDatabasesPage{
		RelationalDatabases: items[offset:end],
		NextPageToken:       nextPageToken,
	}, nil
}

func (s *Service) GetRelationalDatabaseBlueprints(pageToken string) (RelationalDatabaseBlueprintsPage, error) {
	pageToken = strings.TrimSpace(pageToken)
	offset := 0
	if pageToken != "" {
		value, err := strconv.Atoi(pageToken)
		if err != nil || value < 0 {
			return RelationalDatabaseBlueprintsPage{}, ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	blueprints := append([]RelationalDatabaseBlueprint(nil), s.relationalDatabaseBlueprints...)
	sort.Slice(blueprints, func(i, j int) bool { return blueprints[i].BlueprintID < blueprints[j].BlueprintID })
	if offset >= len(blueprints) {
		return RelationalDatabaseBlueprintsPage{Blueprints: []RelationalDatabaseBlueprint{}}, nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(blueprints) {
		end = len(blueprints)
	}
	nextPageToken := ""
	if end < len(blueprints) {
		nextPageToken = strconv.Itoa(end)
	}
	return RelationalDatabaseBlueprintsPage{
		Blueprints:    blueprints[offset:end],
		NextPageToken: nextPageToken,
	}, nil
}

func (s *Service) GetRelationalDatabaseBundles(includeInactive bool, pageToken string) (RelationalDatabaseBundlesPage, error) {
	pageToken = strings.TrimSpace(pageToken)
	offset := 0
	if pageToken != "" {
		value, err := strconv.Atoi(pageToken)
		if err != nil || value < 0 {
			return RelationalDatabaseBundlesPage{}, ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]RelationalDatabaseBundle, 0, len(s.relationalDatabaseBundles))
	for _, bundle := range s.relationalDatabaseBundles {
		if !includeInactive && !bundle.IsActive {
			continue
		}
		filtered = append(filtered, bundle)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].BundleID < filtered[j].BundleID })
	if offset >= len(filtered) {
		return RelationalDatabaseBundlesPage{Bundles: []RelationalDatabaseBundle{}}, nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	nextPageToken := ""
	if end < len(filtered) {
		nextPageToken = strconv.Itoa(end)
	}
	return RelationalDatabaseBundlesPage{
		Bundles:       filtered[offset:end],
		NextPageToken: nextPageToken,
	}, nil
}

func (s *Service) GetRelationalDatabaseMasterUserPassword(relationalDatabaseName, passwordVersion string) (time.Time, string, error) {
	relationalDatabaseName = strings.TrimSpace(relationalDatabaseName)
	passwordVersion = strings.TrimSpace(strings.ToUpper(passwordVersion))
	if relationalDatabaseName == "" {
		return time.Time{}, "", ErrInvalidParameter
	}
	if passwordVersion == "" {
		passwordVersion = "CURRENT"
	}
	if passwordVersion != "CURRENT" && passwordVersion != "PREVIOUS" && passwordVersion != "PENDING" {
		return time.Time{}, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	db, exists := s.relationalDatabases[relationalDatabaseName]
	if !exists {
		return time.Time{}, "", ErrNotFound
	}

	switch passwordVersion {
	case "PREVIOUS":
		if strings.TrimSpace(db.PreviousMasterUserPassword) != "" {
			return db.PreviousMasterPasswordAt, db.PreviousMasterUserPassword, nil
		}
	case "PENDING":
		if strings.TrimSpace(db.PendingMasterUserPassword) != "" {
			return db.PendingMasterPasswordAt, db.PendingMasterUserPassword, nil
		}
	}
	return db.MasterUserPasswordCreatedAt, db.MasterUserPassword, nil
}

func (s *Service) GetRelationalDatabaseMetricData(input RelationalDatabaseMetricInput) (string, []RelationalDatabaseMetricDatapoint, error) {
	input.RelationalDatabaseName = strings.TrimSpace(input.RelationalDatabaseName)
	input.MetricName = normalizeRelationalDatabaseMetricName(input.MetricName)
	input.Unit = strings.TrimSpace(input.Unit)
	if input.RelationalDatabaseName == "" || input.MetricName == "" || input.Unit == "" || input.Period <= 0 || input.StartTime.IsZero() || input.EndTime.IsZero() || input.EndTime.Before(input.StartTime) || len(input.Statistics) == 0 {
		return "", nil, ErrInvalidParameter
	}
	if !hasAnyMetricStatistic(input.Statistics) {
		return "", nil, ErrInvalidParameter
	}
	if !validRelationalDatabaseMetricUnit(input.MetricName, input.Unit) {
		return "", nil, ErrInvalidParameter
	}

	s.mu.Lock()
	_, exists := s.relationalDatabases[input.RelationalDatabaseName]
	s.mu.Unlock()
	if !exists {
		return "", nil, ErrNotFound
	}

	step := time.Duration(input.Period) * time.Second
	count := int(input.EndTime.Sub(input.StartTime)/step) + 1
	if count < 1 {
		count = 1
	}
	if count > 1440 {
		count = 1440
	}

	scale := 1.0
	switch input.MetricName {
	case "CPUUtilization":
		scale = 5.0
	case "DatabaseConnections":
		scale = 2.0
	case "DiskQueueDepth":
		scale = 1.0
	case "FreeStorageSpace":
		scale = 1024 * 1024 * 1024
	case "NetworkReceiveThroughput", "NetworkTransmitThroughput":
		scale = 1024 * 64
	}

	out := make([]RelationalDatabaseMetricDatapoint, 0, count)
	for i := 0; i < count; i++ {
		ts := input.StartTime.Add(time.Duration(i) * step)
		if ts.After(input.EndTime) {
			break
		}
		base := float64(i+1) * scale
		point := RelationalDatabaseMetricDatapoint{
			Timestamp: ts.UTC(),
			Unit:      input.Unit,
		}
		if hasMetricStatistic(input.Statistics, "Average") {
			v := base
			point.Average = &v
		}
		if hasMetricStatistic(input.Statistics, "Maximum") {
			v := base + (0.5 * scale)
			point.Maximum = &v
		}
		if hasMetricStatistic(input.Statistics, "Minimum") {
			v := base - (0.5 * scale)
			point.Minimum = &v
		}
		if hasMetricStatistic(input.Statistics, "SampleCount") {
			v := float64(1)
			point.SampleCount = &v
		}
		if hasMetricStatistic(input.Statistics, "Sum") {
			v := base
			point.Sum = &v
		}
		out = append(out, point)
	}
	if len(out) == 0 {
		out = append(out, RelationalDatabaseMetricDatapoint{
			Timestamp: input.EndTime.UTC(),
			Unit:      input.Unit,
		})
	}
	return input.MetricName, out, nil
}

func (s *Service) GetRelationalDatabaseParameters(relationalDatabaseName, pageToken string) (RelationalDatabaseParametersPage, error) {
	relationalDatabaseName = strings.TrimSpace(relationalDatabaseName)
	pageToken = strings.TrimSpace(pageToken)
	if relationalDatabaseName == "" {
		return RelationalDatabaseParametersPage{}, ErrInvalidParameter
	}
	offset := 0
	if pageToken != "" {
		value, err := strconv.Atoi(pageToken)
		if err != nil || value < 0 {
			return RelationalDatabaseParametersPage{}, ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	db, exists := s.relationalDatabases[relationalDatabaseName]
	if !exists {
		return RelationalDatabaseParametersPage{}, ErrNotFound
	}
	if _, ok := s.relationalDatabaseParameters[relationalDatabaseName]; !ok {
		s.relationalDatabaseParameters[relationalDatabaseName] = defaultRelationalDatabaseParameters(db.Engine)
	}
	parameters := make([]RelationalDatabaseParameter, 0, len(s.relationalDatabaseParameters[relationalDatabaseName]))
	for _, parameter := range s.relationalDatabaseParameters[relationalDatabaseName] {
		parameters = append(parameters, parameter)
	}
	sort.Slice(parameters, func(i, j int) bool { return parameters[i].ParameterName < parameters[j].ParameterName })
	if offset >= len(parameters) {
		return RelationalDatabaseParametersPage{Parameters: []RelationalDatabaseParameter{}}, nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(parameters) {
		end = len(parameters)
	}
	nextPageToken := ""
	if end < len(parameters) {
		nextPageToken = strconv.Itoa(end)
	}
	return RelationalDatabaseParametersPage{
		Parameters:    parameters[offset:end],
		NextPageToken: nextPageToken,
	}, nil
}

func (s *Service) UpdateRelationalDatabaseParameters(relationalDatabaseName string, parameters []RelationalDatabaseParameter) ([]Operation, error) {
	relationalDatabaseName = strings.TrimSpace(relationalDatabaseName)
	if relationalDatabaseName == "" || len(parameters) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	db, exists := s.relationalDatabases[relationalDatabaseName]
	if !exists {
		return nil, ErrNotFound
	}
	if _, ok := s.relationalDatabaseParameters[relationalDatabaseName]; !ok {
		s.relationalDatabaseParameters[relationalDatabaseName] = defaultRelationalDatabaseParameters(db.Engine)
	}

	hasPendingReboot := false
	for _, parameter := range parameters {
		name := strings.TrimSpace(parameter.ParameterName)
		value := strings.TrimSpace(parameter.ParameterValue)
		applyMethod := strings.TrimSpace(strings.ToLower(parameter.ApplyMethod))
		if name == "" {
			return nil, ErrInvalidParameter
		}
		if applyMethod == "" {
			applyMethod = "immediate"
		}
		if applyMethod != "immediate" && applyMethod != "pending-reboot" {
			return nil, ErrInvalidParameter
		}
		current, exists := s.relationalDatabaseParameters[relationalDatabaseName][name]
		if !exists {
			current = RelationalDatabaseParameter{
				AllowedValues: "",
				ApplyMethod:   applyMethod,
				ApplyType:     "dynamic",
				DataType:      "string",
				Description:   "",
				IsModifiable:  true,
				ParameterName: name,
			}
		}
		if !current.IsModifiable {
			return nil, ErrInvalidParameter
		}
		current.ParameterValue = value
		current.ApplyMethod = applyMethod
		s.relationalDatabaseParameters[relationalDatabaseName][name] = current
		if applyMethod == "pending-reboot" {
			hasPendingReboot = true
		}
	}

	now := time.Now().UTC()
	if hasPendingReboot {
		db.ParameterApplyStatus = "pending-reboot"
	} else {
		db.ParameterApplyStatus = "in-sync"
	}
	db.LatestRestorableTime = now
	s.appendRelationalDatabaseEventLocked(relationalDatabaseName, "configuration change", "updated relational database parameters", now)

	seq := atomic.AddUint64(&s.seq, 1)
	op := newOperation(seq, relationalDatabaseName, "RelationalDatabase", "UpdateRelationalDatabaseParameters", "Succeeded", "relational database parameters updated", db.AvailabilityZone, db.Region, now)
	ops := []Operation{op}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) UpdateRelationalDatabase(input RelationalDatabaseUpdateInput) ([]Operation, error) {
	input.RelationalDatabaseName = strings.TrimSpace(input.RelationalDatabaseName)
	if input.RelationalDatabaseName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	db, exists := s.relationalDatabases[input.RelationalDatabaseName]
	if !exists {
		return nil, ErrNotFound
	}

	if input.DisableBackupRetention != nil && input.EnableBackupRetention != nil && *input.DisableBackupRetention && *input.EnableBackupRetention {
		return nil, ErrInvalidParameter
	}

	pending := &PendingModifiedRelationalDatabaseValues{}
	changed := false
	if input.CACertificateIdentifier != nil {
		value := strings.TrimSpace(*input.CACertificateIdentifier)
		if value == "" {
			return nil, ErrInvalidParameter
		}
		db.CACertificateIdentifier = value
		changed = true
	}
	if input.DisableBackupRetention != nil && *input.DisableBackupRetention {
		db.BackupRetentionEnabled = false
		value := false
		pending.BackupRetentionEnabled = &value
		changed = true
	}
	if input.EnableBackupRetention != nil && *input.EnableBackupRetention {
		db.BackupRetentionEnabled = true
		value := true
		pending.BackupRetentionEnabled = &value
		changed = true
	}
	if input.MasterUserPassword != nil {
		value := strings.TrimSpace(*input.MasterUserPassword)
		if value == "" {
			return nil, ErrInvalidParameter
		}
		db.PreviousMasterUserPassword = db.MasterUserPassword
		db.PreviousMasterPasswordAt = db.MasterUserPasswordCreatedAt
		db.MasterUserPassword = value
		db.MasterUserPasswordCreatedAt = time.Now().UTC()
		db.PendingMasterUserPassword = ""
		db.PendingMasterPasswordAt = time.Time{}
		pending.MasterUserPassword = &value
		changed = true
	}
	if input.PreferredBackupWindow != nil {
		db.PreferredBackupWindow = strings.TrimSpace(*input.PreferredBackupWindow)
		changed = true
	}
	if input.PreferredMaintenanceWindow != nil {
		db.PreferredMaintenanceWindow = strings.TrimSpace(*input.PreferredMaintenanceWindow)
		changed = true
	}
	if input.PubliclyAccessible != nil {
		db.PubliclyAccessible = *input.PubliclyAccessible
		changed = true
	}
	if input.RelationalDatabaseBlueprintID != nil {
		blueprintID := strings.TrimSpace(*input.RelationalDatabaseBlueprintID)
		if blueprintID == "" {
			return nil, ErrInvalidParameter
		}
		db.BlueprintID = blueprintID
		engine, engineVersion, port := relationalDatabaseEngineFromBlueprint(blueprintID)
		db.Engine = engine
		db.EngineVersion = engineVersion
		db.MasterEndpointPort = port
		pending.EngineVersion = &engineVersion
		changed = true
	}
	if input.RotateMasterUserPassword != nil && *input.RotateMasterUserPassword {
		newPassword := fmt.Sprintf("Stackyard!%d", atomic.AddUint64(&s.seq, 1)%100000)
		if input.ApplyImmediately != nil && *input.ApplyImmediately {
			db.PreviousMasterUserPassword = db.MasterUserPassword
			db.PreviousMasterPasswordAt = db.MasterUserPasswordCreatedAt
			db.MasterUserPassword = newPassword
			db.MasterUserPasswordCreatedAt = time.Now().UTC()
			db.PendingMasterUserPassword = ""
			db.PendingMasterPasswordAt = time.Time{}
		} else {
			db.PendingMasterUserPassword = newPassword
			db.PendingMasterPasswordAt = time.Now().UTC()
		}
		pending.MasterUserPassword = &newPassword
		changed = true
	}
	if !changed {
		return nil, ErrInvalidParameter
	}

	if pending.BackupRetentionEnabled != nil || pending.EngineVersion != nil || pending.MasterUserPassword != nil {
		db.PendingModifiedValues = pending
	} else {
		db.PendingModifiedValues = nil
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	db.LatestRestorableTime = now
	if input.ApplyImmediately != nil && *input.ApplyImmediately {
		db.ParameterApplyStatus = "applying"
	} else {
		db.ParameterApplyStatus = "pending-reboot"
	}
	op := newOperation(seq, input.RelationalDatabaseName, "RelationalDatabase", "UpdateRelationalDatabase", "Succeeded", "relational database updated", db.AvailabilityZone, db.Region, now)
	ops := []Operation{op}
	s.appendOperationsLocked(ops)
	s.appendRelationalDatabaseEventLocked(input.RelationalDatabaseName, "configuration change", "updated relational database configuration", now)
	return ops, nil
}

func (s *Service) DeleteRelationalDatabase(input RelationalDatabaseDeleteInput) ([]Operation, error) {
	input.RelationalDatabaseName = strings.TrimSpace(input.RelationalDatabaseName)
	input.FinalRelationalDatabaseSnapshotName = strings.TrimSpace(input.FinalRelationalDatabaseSnapshotName)
	if input.RelationalDatabaseName == "" {
		return nil, ErrInvalidParameter
	}
	if input.SkipFinalSnapshot != nil && *input.SkipFinalSnapshot && input.FinalRelationalDatabaseSnapshotName != "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	db, exists := s.relationalDatabases[input.RelationalDatabaseName]
	if !exists {
		return nil, ErrNotFound
	}
	delete(s.relationalDatabases, input.RelationalDatabaseName)
	delete(s.relationalDatabaseParameters, input.RelationalDatabaseName)
	delete(s.relationalDatabaseEvents, input.RelationalDatabaseName)
	delete(s.relationalDatabaseLogStreams, input.RelationalDatabaseName)
	delete(s.relationalDatabaseLogEvents, input.RelationalDatabaseName)

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, input.RelationalDatabaseName, "RelationalDatabase", "DeleteRelationalDatabase", "Succeeded", "relational database deleted", db.AvailabilityZone, db.Region, now)
	ops := []Operation{op}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) RebootRelationalDatabase(name string) ([]Operation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	db, exists := s.relationalDatabases[name]
	if !exists {
		return nil, ErrNotFound
	}
	if strings.TrimSpace(db.PendingMasterUserPassword) != "" {
		db.PreviousMasterUserPassword = db.MasterUserPassword
		db.PreviousMasterPasswordAt = db.MasterUserPasswordCreatedAt
		db.MasterUserPassword = db.PendingMasterUserPassword
		db.MasterUserPasswordCreatedAt = firstNonZeroTime(db.PendingMasterPasswordAt, time.Now().UTC())
		db.PendingMasterUserPassword = ""
		db.PendingMasterPasswordAt = time.Time{}
	}
	if db.ParameterApplyStatus == "pending-reboot" {
		db.ParameterApplyStatus = "in-sync"
	}
	db.PendingModifiedValues = nil
	db.State = "available"

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, name, "RelationalDatabase", "RebootRelationalDatabase", "Succeeded", "relational database rebooted", db.AvailabilityZone, db.Region, now)
	ops := []Operation{op}
	s.appendOperationsLocked(ops)
	s.appendRelationalDatabaseEventLocked(name, "availability", "rebooted relational database", now)
	return ops, nil
}

func (s *Service) StartRelationalDatabase(name string) ([]Operation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	db, exists := s.relationalDatabases[name]
	if !exists {
		return nil, ErrNotFound
	}
	if strings.TrimSpace(db.PendingMasterUserPassword) != "" {
		db.PreviousMasterUserPassword = db.MasterUserPassword
		db.PreviousMasterPasswordAt = db.MasterUserPasswordCreatedAt
		db.MasterUserPassword = db.PendingMasterUserPassword
		db.MasterUserPasswordCreatedAt = firstNonZeroTime(db.PendingMasterPasswordAt, time.Now().UTC())
		db.PendingMasterUserPassword = ""
		db.PendingMasterPasswordAt = time.Time{}
	}
	if db.ParameterApplyStatus == "pending-reboot" {
		db.ParameterApplyStatus = "in-sync"
	}
	db.PendingModifiedValues = nil
	db.State = "available"
	db.LatestRestorableTime = time.Now().UTC()

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, name, "RelationalDatabase", "StartRelationalDatabase", "Succeeded", "relational database started", db.AvailabilityZone, db.Region, now)
	ops := []Operation{op}
	s.appendOperationsLocked(ops)
	s.appendRelationalDatabaseEventLocked(name, "availability", "started relational database", now)
	return ops, nil
}

func (s *Service) StopRelationalDatabase(name, snapshotName string) ([]Operation, error) {
	name = strings.TrimSpace(name)
	snapshotName = strings.TrimSpace(snapshotName)
	if name == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	db, exists := s.relationalDatabases[name]
	if !exists {
		return nil, ErrNotFound
	}
	db.State = "stopped"

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	details := "relational database stopped"
	if snapshotName != "" {
		details = fmt.Sprintf("relational database stopped with snapshot %s", snapshotName)
		snapshot := &RelationalDatabaseSnapshot{
			Name:                              snapshotName,
			ARN:                               relationalDatabaseSnapshotARN(db.Region, snapshotName),
			CreatedAt:                         now,
			Engine:                            db.Engine,
			EngineVersion:                     db.EngineVersion,
			FromRelationalDatabaseARN:         db.ARN,
			FromRelationalDatabaseBlueprintID: db.BlueprintID,
			FromRelationalDatabaseBundleID:    db.BundleID,
			FromRelationalDatabaseName:        db.Name,
			AvailabilityZone:                  db.AvailabilityZone,
			Region:                            db.Region,
			SizeInGb:                          db.DiskSizeInGb,
			State:                             "available",
			SupportCode:                       fmt.Sprintf("%s/%d", db.Region, seq),
			Tags:                              cloneStringMap(db.Tags),
		}
		s.relationalDatabaseSnapshots[snapshotName] = snapshot
	}
	op := newOperation(seq, name, "RelationalDatabase", "StopRelationalDatabase", "Succeeded", details, db.AvailabilityZone, db.Region, now)
	ops := []Operation{op}
	s.appendOperationsLocked(ops)
	s.appendRelationalDatabaseEventLocked(name, "availability", details, now)
	return ops, nil
}

func (s *Service) CreateRelationalDatabaseSnapshot(relationalDatabaseName, relationalDatabaseSnapshotName string, tags map[string]string) ([]Operation, error) {
	relationalDatabaseName = strings.TrimSpace(relationalDatabaseName)
	relationalDatabaseSnapshotName = strings.TrimSpace(relationalDatabaseSnapshotName)
	if relationalDatabaseName == "" || relationalDatabaseSnapshotName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	db, exists := s.relationalDatabases[relationalDatabaseName]
	if !exists {
		return nil, ErrNotFound
	}
	if _, exists := s.relationalDatabaseSnapshots[relationalDatabaseSnapshotName]; exists {
		return nil, ErrAlreadyExists
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	snapshot := &RelationalDatabaseSnapshot{
		Name:                              relationalDatabaseSnapshotName,
		ARN:                               relationalDatabaseSnapshotARN(db.Region, relationalDatabaseSnapshotName),
		CreatedAt:                         now,
		Engine:                            db.Engine,
		EngineVersion:                     db.EngineVersion,
		FromRelationalDatabaseARN:         db.ARN,
		FromRelationalDatabaseBlueprintID: db.BlueprintID,
		FromRelationalDatabaseBundleID:    db.BundleID,
		FromRelationalDatabaseName:        db.Name,
		AvailabilityZone:                  db.AvailabilityZone,
		Region:                            db.Region,
		SizeInGb:                          db.DiskSizeInGb,
		State:                             "available",
		SupportCode:                       fmt.Sprintf("%s/%d", db.Region, seq),
		Tags:                              cloneStringMap(tags),
	}
	s.relationalDatabaseSnapshots[relationalDatabaseSnapshotName] = snapshot
	s.appendRelationalDatabaseEventLocked(relationalDatabaseName, "backup", fmt.Sprintf("created snapshot %s", relationalDatabaseSnapshotName), now)

	op := newOperation(seq, relationalDatabaseSnapshotName, "RelationalDatabaseSnapshot", "CreateRelationalDatabaseSnapshot", "Succeeded", "relational database snapshot created", db.AvailabilityZone, db.Region, now)
	ops := []Operation{op}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetRelationalDatabaseSnapshot(relationalDatabaseSnapshotName string) (RelationalDatabaseSnapshot, bool) {
	relationalDatabaseSnapshotName = strings.TrimSpace(relationalDatabaseSnapshotName)
	if relationalDatabaseSnapshotName == "" {
		return RelationalDatabaseSnapshot{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, exists := s.relationalDatabaseSnapshots[relationalDatabaseSnapshotName]
	if !exists {
		return RelationalDatabaseSnapshot{}, false
	}
	return cloneRelationalDatabaseSnapshot(snapshot), true
}

func (s *Service) GetRelationalDatabaseSnapshots(pageToken string) (RelationalDatabaseSnapshotsPage, error) {
	pageToken = strings.TrimSpace(pageToken)
	offset := 0
	if pageToken != "" {
		value, err := strconv.Atoi(pageToken)
		if err != nil || value < 0 {
			return RelationalDatabaseSnapshotsPage{}, ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshots := make([]RelationalDatabaseSnapshot, 0, len(s.relationalDatabaseSnapshots))
	for _, snapshot := range s.relationalDatabaseSnapshots {
		snapshots = append(snapshots, cloneRelationalDatabaseSnapshot(snapshot))
	}
	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].CreatedAt.Equal(snapshots[j].CreatedAt) {
			return snapshots[i].Name < snapshots[j].Name
		}
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})

	if offset >= len(snapshots) {
		return RelationalDatabaseSnapshotsPage{RelationalDatabaseSnapshots: []RelationalDatabaseSnapshot{}}, nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(snapshots) {
		end = len(snapshots)
	}
	nextPageToken := ""
	if end < len(snapshots) {
		nextPageToken = strconv.Itoa(end)
	}
	return RelationalDatabaseSnapshotsPage{
		RelationalDatabaseSnapshots: snapshots[offset:end],
		NextPageToken:               nextPageToken,
	}, nil
}

func (s *Service) DeleteRelationalDatabaseSnapshot(relationalDatabaseSnapshotName string) ([]Operation, error) {
	relationalDatabaseSnapshotName = strings.TrimSpace(relationalDatabaseSnapshotName)
	if relationalDatabaseSnapshotName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, exists := s.relationalDatabaseSnapshots[relationalDatabaseSnapshotName]
	if !exists {
		return nil, ErrNotFound
	}
	delete(s.relationalDatabaseSnapshots, relationalDatabaseSnapshotName)
	if sourceName := strings.TrimSpace(snapshot.FromRelationalDatabaseName); sourceName != "" {
		s.appendRelationalDatabaseEventLocked(sourceName, "backup", fmt.Sprintf("deleted snapshot %s", relationalDatabaseSnapshotName), time.Now().UTC())
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, relationalDatabaseSnapshotName, "RelationalDatabaseSnapshot", "DeleteRelationalDatabaseSnapshot", "Succeeded", "relational database snapshot deleted", snapshot.AvailabilityZone, snapshot.Region, now)
	ops := []Operation{op}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) CreateRelationalDatabaseFromSnapshot(input RelationalDatabaseFromSnapshotInput) ([]Operation, error) {
	input.RelationalDatabaseName = strings.TrimSpace(input.RelationalDatabaseName)
	input.AvailabilityZone = strings.TrimSpace(input.AvailabilityZone)
	input.RelationalDatabaseBundleID = strings.TrimSpace(input.RelationalDatabaseBundleID)
	input.RelationalDatabaseSnapshotName = strings.TrimSpace(input.RelationalDatabaseSnapshotName)
	input.SourceRelationalDatabaseName = strings.TrimSpace(input.SourceRelationalDatabaseName)
	if input.RelationalDatabaseName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.relationalDatabases[input.RelationalDatabaseName]; exists {
		return nil, ErrAlreadyExists
	}

	var source *RelationalDatabase
	if input.RelationalDatabaseSnapshotName != "" {
		snapshot, exists := s.relationalDatabaseSnapshots[input.RelationalDatabaseSnapshotName]
		if !exists {
			return nil, ErrNotFound
		}
		source = &RelationalDatabase{
			Name:               input.RelationalDatabaseName,
			BlueprintID:        snapshot.FromRelationalDatabaseBlueprintID,
			BundleID:           snapshot.FromRelationalDatabaseBundleID,
			Engine:             snapshot.Engine,
			EngineVersion:      snapshot.EngineVersion,
			MasterDatabaseName: firstNonEmptyString(snapshot.FromRelationalDatabaseName, "postgres"),
			MasterUsername:     "admin",
			DiskSizeInGb:       snapshot.SizeInGb,
		}
		if input.AvailabilityZone == "" {
			input.AvailabilityZone = snapshot.AvailabilityZone
		}
	}
	if source == nil && input.SourceRelationalDatabaseName != "" {
		db, exists := s.relationalDatabases[input.SourceRelationalDatabaseName]
		if !exists {
			return nil, ErrNotFound
		}
		cloned := cloneRelationalDatabase(db)
		source = &cloned
	}
	if source == nil {
		return nil, ErrInvalidParameter
	}

	if input.AvailabilityZone == "" {
		input.AvailabilityZone = source.AvailabilityZone
	}
	if input.RelationalDatabaseBundleID == "" {
		input.RelationalDatabaseBundleID = source.BundleID
	}
	publiclyAccessible := source.PubliclyAccessible
	if input.PubliclyAccessible != nil {
		publiclyAccessible = *input.PubliclyAccessible
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := regionFromAvailabilityZone(input.AvailabilityZone)
	cpu, disk, ram := relationalDatabaseHardwareFromBundle(input.RelationalDatabaseBundleID)
	engine, engineVersion, port := relationalDatabaseEngineFromBlueprint(source.BlueprintID)
	db := &RelationalDatabase{
		Name:                        input.RelationalDatabaseName,
		ARN:                         relationalDatabaseARN(region, input.RelationalDatabaseName),
		CreatedAt:                   now,
		AvailabilityZone:            input.AvailabilityZone,
		Region:                      region,
		BlueprintID:                 source.BlueprintID,
		BundleID:                    input.RelationalDatabaseBundleID,
		Engine:                      engine,
		EngineVersion:               firstNonEmptyString(source.EngineVersion, engineVersion),
		MasterDatabaseName:          firstNonEmptyString(source.MasterDatabaseName, "postgres"),
		MasterUsername:              firstNonEmptyString(source.MasterUsername, "admin"),
		MasterUserPassword:          fmt.Sprintf("Stackyard!%d", seq%100000),
		MasterUserPasswordCreatedAt: now,
		MasterEndpointAddress:       fmt.Sprintf("%s.%s.rds.amazonaws.com", input.RelationalDatabaseName, region),
		MasterEndpointPort:          port,
		CPUCount:                    cpu,
		DiskSizeInGb:                maxInt32(disk, source.DiskSizeInGb),
		RAMSizeInGb:                 ram,
		LatestRestorableTime:        now,
		PreferredBackupWindow:       firstNonEmptyString(source.PreferredBackupWindow, "03:00-03:30"),
		PreferredMaintenanceWindow:  firstNonEmptyString(source.PreferredMaintenanceWindow, "Sun:04:00-Sun:04:30"),
		PubliclyAccessible:          publiclyAccessible,
		BackupRetentionEnabled:      true,
		CACertificateIdentifier:     firstNonEmptyString(source.CACertificateIdentifier, "rds-ca-rsa2048-g1"),
		ParameterApplyStatus:        "in-sync",
		SecondaryAvailabilityZone:   region + "b",
		State:                       "available",
		SupportCode:                 fmt.Sprintf("%s/%d", region, seq),
		Tags:                        cloneStringMap(input.Tags),
	}
	s.relationalDatabases[input.RelationalDatabaseName] = db
	s.relationalDatabaseParameters[input.RelationalDatabaseName] = defaultRelationalDatabaseParameters(db.Engine)
	s.relationalDatabaseLogStreams[input.RelationalDatabaseName] = defaultRelationalDatabaseLogStreams()
	s.relationalDatabaseLogEvents[input.RelationalDatabaseName] = defaultRelationalDatabaseLogEvents(input.RelationalDatabaseName, now)
	s.appendRelationalDatabaseEventLocked(input.RelationalDatabaseName, "creation", "created relational database from snapshot", now)

	op := newOperation(seq, input.RelationalDatabaseName, "RelationalDatabase", "CreateRelationalDatabaseFromSnapshot", "Succeeded", "relational database created from snapshot", db.AvailabilityZone, db.Region, now)
	ops := []Operation{op}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetRelationalDatabaseEvents(relationalDatabaseName string, durationInMinutes *int32, pageToken string) (RelationalDatabaseEventsPage, error) {
	relationalDatabaseName = strings.TrimSpace(relationalDatabaseName)
	pageToken = strings.TrimSpace(pageToken)
	if relationalDatabaseName == "" {
		return RelationalDatabaseEventsPage{}, ErrInvalidParameter
	}
	offset := 0
	if pageToken != "" {
		value, err := strconv.Atoi(pageToken)
		if err != nil || value < 0 {
			return RelationalDatabaseEventsPage{}, ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.relationalDatabases[relationalDatabaseName]; !exists {
		return RelationalDatabaseEventsPage{}, ErrNotFound
	}
	events := cloneRelationalDatabaseEvents(s.relationalDatabaseEvents[relationalDatabaseName])
	if durationInMinutes != nil && *durationInMinutes > 0 {
		cutoff := time.Now().UTC().Add(-time.Duration(*durationInMinutes) * time.Minute)
		filtered := make([]RelationalDatabaseEvent, 0, len(events))
		for _, event := range events {
			if event.CreatedAt.Before(cutoff) {
				continue
			}
			filtered = append(filtered, event)
		}
		events = filtered
	}
	sort.Slice(events, func(i, j int) bool { return events[i].CreatedAt.After(events[j].CreatedAt) })

	if offset >= len(events) {
		return RelationalDatabaseEventsPage{RelationalDatabaseEvents: []RelationalDatabaseEvent{}}, nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(events) {
		end = len(events)
	}
	nextPageToken := ""
	if end < len(events) {
		nextPageToken = strconv.Itoa(end)
	}
	return RelationalDatabaseEventsPage{
		RelationalDatabaseEvents: events[offset:end],
		NextPageToken:            nextPageToken,
	}, nil
}

func (s *Service) GetRelationalDatabaseLogStreams(relationalDatabaseName string) ([]string, error) {
	relationalDatabaseName = strings.TrimSpace(relationalDatabaseName)
	if relationalDatabaseName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.relationalDatabases[relationalDatabaseName]; !exists {
		return nil, ErrNotFound
	}
	if _, ok := s.relationalDatabaseLogStreams[relationalDatabaseName]; !ok {
		s.relationalDatabaseLogStreams[relationalDatabaseName] = defaultRelationalDatabaseLogStreams()
	}
	out := dedupeStrings(s.relationalDatabaseLogStreams[relationalDatabaseName])
	sort.Strings(out)
	return out, nil
}

func (s *Service) GetRelationalDatabaseLogEvents(input RelationalDatabaseLogEventsInput) (RelationalDatabaseLogEventsPage, error) {
	input.RelationalDatabaseName = strings.TrimSpace(input.RelationalDatabaseName)
	input.LogStreamName = strings.TrimSpace(input.LogStreamName)
	input.PageToken = strings.TrimSpace(input.PageToken)
	if input.RelationalDatabaseName == "" || input.LogStreamName == "" {
		return RelationalDatabaseLogEventsPage{}, ErrInvalidParameter
	}
	if input.StartTime != nil && input.EndTime != nil && input.EndTime.Before(*input.StartTime) {
		return RelationalDatabaseLogEventsPage{}, ErrInvalidParameter
	}

	offset := 0
	if input.PageToken != "" {
		value, err := strconv.Atoi(input.PageToken)
		if err != nil || value < 0 {
			return RelationalDatabaseLogEventsPage{}, ErrInvalidParameter
		}
		offset = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.relationalDatabases[input.RelationalDatabaseName]; !exists {
		return RelationalDatabaseLogEventsPage{}, ErrNotFound
	}
	if _, ok := s.relationalDatabaseLogStreams[input.RelationalDatabaseName]; !ok {
		s.relationalDatabaseLogStreams[input.RelationalDatabaseName] = defaultRelationalDatabaseLogStreams()
	}
	if _, ok := s.relationalDatabaseLogEvents[input.RelationalDatabaseName]; !ok {
		s.relationalDatabaseLogEvents[input.RelationalDatabaseName] = defaultRelationalDatabaseLogEvents(input.RelationalDatabaseName, time.Now().UTC())
	}
	streamEvents, exists := s.relationalDatabaseLogEvents[input.RelationalDatabaseName][input.LogStreamName]
	if !exists {
		return RelationalDatabaseLogEventsPage{}, ErrNotFound
	}

	events := cloneRelationalDatabaseLogEvents(streamEvents)
	filtered := make([]RelationalDatabaseLogEvent, 0, len(events))
	for _, event := range events {
		if input.StartTime != nil && event.CreatedAt.Before(*input.StartTime) {
			continue
		}
		if input.EndTime != nil && event.CreatedAt.After(*input.EndTime) {
			continue
		}
		filtered = append(filtered, event)
	}

	if input.StartFromHead == nil || !*input.StartFromHead {
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].CreatedAt.After(filtered[j].CreatedAt) })
	} else {
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].CreatedAt.Before(filtered[j].CreatedAt) })
	}

	if offset >= len(filtered) {
		return RelationalDatabaseLogEventsPage{ResourceLogEvents: []RelationalDatabaseLogEvent{}}, nil
	}
	const pageSize = 100
	end := offset + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	nextBackwardToken := ""
	nextForwardToken := ""
	if offset > 0 {
		nextBackwardToken = strconv.Itoa(maxInt(offset-pageSize, 0))
	}
	if end < len(filtered) {
		nextForwardToken = strconv.Itoa(end)
	}
	return RelationalDatabaseLogEventsPage{
		ResourceLogEvents: filtered[offset:end],
		NextBackwardToken: nextBackwardToken,
		NextForwardToken:  nextForwardToken,
	}, nil
}

func (s *Service) CreateKeyPair(keyPairName string, tags map[string]string) (KeyPair, Operation, error) {
	keyPairName = strings.TrimSpace(keyPairName)
	if keyPairName == "" {
		return KeyPair{}, Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.keyPairs[keyPairName]; exists {
		return KeyPair{}, Operation{}, ErrAlreadyExists
	}
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := DefaultRegion
	availabilityZone := region + "a"
	keyPair := &KeyPair{
		Name:             keyPairName,
		ARN:              keyPairARN(region, keyPairName),
		Fingerprint:      fingerprintFromSeed(seq),
		AvailabilityZone: availabilityZone,
		Region:           region,
		CreatedAt:        now,
		PublicKeyBase64:  encodeKeyMaterial(fmt.Sprintf("ssh-rsa STACKYARD-%s-PUBLIC", keyPairName)),
		PrivateKeyBase64: encodeKeyMaterial(fmt.Sprintf("-----BEGIN PRIVATE KEY-----\nSTACKYARD-%s-PRIVATE\n-----END PRIVATE KEY-----", keyPairName)),
		Tags:             cloneStringMap(tags),
	}
	s.keyPairs[keyPairName] = keyPair
	op := newOperation(seq, keyPairName, "KeyPair", "CreateKeyPair", "Succeeded", "key pair created", availabilityZone, region, now)
	s.appendOperationsLocked([]Operation{op})
	return cloneKeyPair(keyPair), op, nil
}

func (s *Service) ImportKeyPair(keyPairName, publicKeyBase64 string) (Operation, error) {
	keyPairName = strings.TrimSpace(keyPairName)
	publicKeyBase64 = strings.TrimSpace(publicKeyBase64)
	if keyPairName == "" || publicKeyBase64 == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.keyPairs[keyPairName]; exists {
		return Operation{}, ErrAlreadyExists
	}
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := DefaultRegion
	availabilityZone := region + "a"
	keyPair := &KeyPair{
		Name:             keyPairName,
		ARN:              keyPairARN(region, keyPairName),
		Fingerprint:      fingerprintFromSeed(seq),
		AvailabilityZone: availabilityZone,
		Region:           region,
		CreatedAt:        now,
		PublicKeyBase64:  publicKeyBase64,
		PrivateKeyBase64: "",
		Tags:             map[string]string{},
	}
	s.keyPairs[keyPairName] = keyPair
	op := newOperation(seq, keyPairName, "KeyPair", "ImportKeyPair", "Succeeded", "key pair imported", availabilityZone, region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) GetKeyPair(keyPairName string) (KeyPair, bool) {
	keyPairName = strings.TrimSpace(keyPairName)
	s.mu.Lock()
	defer s.mu.Unlock()
	keyPair, ok := s.keyPairs[keyPairName]
	if !ok {
		return KeyPair{}, false
	}
	return cloneKeyPair(keyPair), true
}

func (s *Service) GetKeyPairs(includeDefault bool) []KeyPair {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]KeyPair, 0, len(s.keyPairs))
	for _, keyPair := range s.keyPairs {
		if keyPair.IsDefault && !includeDefault {
			continue
		}
		out = append(out, cloneKeyPair(keyPair))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) DeleteKeyPair(keyPairName, expectedFingerprint string) (Operation, error) {
	keyPairName = strings.TrimSpace(keyPairName)
	expectedFingerprint = strings.TrimSpace(expectedFingerprint)
	if keyPairName == "" {
		return Operation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	keyPair, ok := s.keyPairs[keyPairName]
	if !ok {
		return Operation{}, ErrNotFound
	}
	if keyPair.IsDefault && expectedFingerprint != "" && expectedFingerprint != keyPair.Fingerprint {
		return Operation{}, ErrInvalidParameter
	}
	delete(s.keyPairs, keyPairName)
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	op := newOperation(seq, keyPairName, "KeyPair", "DeleteKeyPair", "Succeeded", "key pair deleted", keyPair.AvailabilityZone, keyPair.Region, now)
	s.appendOperationsLocked([]Operation{op})
	return op, nil
}

func (s *Service) DownloadDefaultKeyPair() (time.Time, string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyPair := s.ensureDefaultKeyPairLocked()
	return keyPair.CreatedAt, keyPair.PrivateKeyBase64, keyPair.PublicKeyBase64, nil
}

func (s *Service) GetInstances() []Instance {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Instance, 0, len(s.instances))
	for _, instance := range s.instances {
		out = append(out, cloneInstance(instance))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) GetInstance(name string) (Instance, bool) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[name]
	if !ok {
		return Instance{}, false
	}
	return cloneInstance(instance), true
}

func (s *Service) GetInstanceSnapshot(name string) (InstanceSnapshot, bool) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.snapshots[name]
	if !ok {
		return InstanceSnapshot{}, false
	}
	return cloneInstanceSnapshot(snapshot), true
}

func (s *Service) GetInstanceState(name string) (int32, string, bool) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[name]
	if !ok {
		return 0, "", false
	}
	return instance.StateCode, instance.StateName, true
}

func (s *Service) StartInstance(name string) ([]Operation, error) {
	return s.transitionInstance(name, "StartInstance", 16, "running")
}

func (s *Service) StopInstance(name string) ([]Operation, error) {
	return s.transitionInstance(name, "StopInstance", 80, "stopped")
}

func (s *Service) RebootInstance(name string) ([]Operation, error) {
	return s.transitionInstance(name, "RebootInstance", 16, "running")
}

func (s *Service) DeleteInstance(name string) ([]Operation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[name]
	if !ok {
		return nil, ErrNotFound
	}
	for _, ip := range s.staticIPs {
		if ip.AttachedTo == name {
			ip.AttachedTo = ""
		}
	}
	for _, disk := range s.disks {
		if disk.AttachedTo != name {
			continue
		}
		disk.AttachedTo = ""
		disk.Path = ""
		disk.IsAttached = false
		disk.State = "available"
		disk.AutoMountStatus = "NotMounted"
	}
	delete(s.instances, name)
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, name, "Instance", "DeleteInstance", "Succeeded", "instance deleted", instance.AvailabilityZone, instance.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) CreateInstanceSnapshot(instanceName, snapshotName string, tags map[string]string) ([]Operation, error) {
	instanceName = strings.TrimSpace(instanceName)
	snapshotName = strings.TrimSpace(snapshotName)
	if instanceName == "" || snapshotName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[instanceName]
	if !ok {
		return nil, ErrNotFound
	}
	if _, exists := s.snapshots[snapshotName]; exists {
		return nil, ErrAlreadyExists
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	snap := &InstanceSnapshot{
		Name:             snapshotName,
		ARN:              instanceSnapshotARN(instance.Region, snapshotName),
		FromInstanceName: instance.Name,
		FromInstanceARN:  instance.ARN,
		FromBlueprintID:  instance.BlueprintID,
		FromBundleID:     instance.BundleID,
		AvailabilityZone: instance.AvailabilityZone,
		Region:           instance.Region,
		State:            "available",
		CreatedAt:        now,
		Tags:             cloneStringMap(tags),
	}
	s.snapshots[snapshotName] = snap

	ops := []Operation{newOperation(seq, snapshotName, "InstanceSnapshot", "CreateInstanceSnapshot", "Succeeded", "snapshot created", snap.AvailabilityZone, snap.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetInstanceSnapshots() []InstanceSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]InstanceSnapshot, 0, len(s.snapshots))
	for _, snapshot := range s.snapshots {
		out = append(out, cloneInstanceSnapshot(snapshot))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) DeleteInstanceSnapshot(snapshotName string) ([]Operation, error) {
	snapshotName = strings.TrimSpace(snapshotName)
	if snapshotName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.snapshots[snapshotName]
	if !ok {
		return nil, ErrNotFound
	}
	delete(s.snapshots, snapshotName)
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, snapshotName, "InstanceSnapshot", "DeleteInstanceSnapshot", "Succeeded", "snapshot deleted", snapshot.AvailabilityZone, snapshot.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) AllocateStaticIP(name string) ([]Operation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.staticIPs[name]; exists {
		return nil, ErrAlreadyExists
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	region := DefaultRegion
	availabilityZone := region + "a"
	ip := &StaticIP{
		Name:             name,
		ARN:              staticIPARN(region, name),
		IPAddress:        fmt.Sprintf("198.51.100.%d", (seq%250)+1),
		AvailabilityZone: availabilityZone,
		Region:           region,
		CreatedAt:        now,
		Tags:             map[string]string{},
	}
	s.staticIPs[name] = ip
	ops := []Operation{newOperation(seq, name, "StaticIp", "AllocateStaticIp", "Succeeded", "static ip allocated", availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetStaticIPs() []StaticIP {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]StaticIP, 0, len(s.staticIPs))
	for _, staticIP := range s.staticIPs {
		out = append(out, cloneStaticIP(staticIP))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) GetStaticIP(name string) (StaticIP, bool) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	staticIP, ok := s.staticIPs[name]
	if !ok {
		return StaticIP{}, false
	}
	return cloneStaticIP(staticIP), true
}

func (s *Service) AttachStaticIP(staticIPName, instanceName string) ([]Operation, error) {
	staticIPName = strings.TrimSpace(staticIPName)
	instanceName = strings.TrimSpace(instanceName)
	if staticIPName == "" || instanceName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[instanceName]
	if !ok {
		return nil, ErrNotFound
	}
	staticIP, ok := s.staticIPs[staticIPName]
	if !ok {
		return nil, ErrNotFound
	}

	if staticIP.AttachedTo != "" {
		if prev, ok := s.instances[staticIP.AttachedTo]; ok {
			prev.IsStaticIP = false
		}
	}

	staticIP.AttachedTo = instanceName
	instance.IsStaticIP = true
	instance.PublicIPAddress = staticIP.IPAddress

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, staticIPName, "StaticIp", "AttachStaticIp", "Succeeded", "static ip attached", instance.AvailabilityZone, instance.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) DetachStaticIP(staticIPName string) ([]Operation, error) {
	staticIPName = strings.TrimSpace(staticIPName)
	if staticIPName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	staticIP, ok := s.staticIPs[staticIPName]
	if !ok {
		return nil, ErrNotFound
	}
	if staticIP.AttachedTo != "" {
		if instance, ok := s.instances[staticIP.AttachedTo]; ok {
			instance.IsStaticIP = false
		}
	}
	attachedTo := staticIP.AttachedTo
	staticIP.AttachedTo = ""

	region := staticIP.Region
	availabilityZone := staticIP.AvailabilityZone
	if attachedTo != "" {
		if instance, ok := s.instances[attachedTo]; ok {
			region = instance.Region
			availabilityZone = instance.AvailabilityZone
		}
	}

	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, staticIPName, "StaticIp", "DetachStaticIp", "Succeeded", "static ip detached", availabilityZone, region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) ReleaseStaticIP(staticIPName string) ([]Operation, error) {
	staticIPName = strings.TrimSpace(staticIPName)
	if staticIPName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	staticIP, ok := s.staticIPs[staticIPName]
	if !ok {
		return nil, ErrNotFound
	}
	if staticIP.AttachedTo != "" {
		if instance, ok := s.instances[staticIP.AttachedTo]; ok {
			instance.IsStaticIP = false
		}
	}
	delete(s.staticIPs, staticIPName)
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, staticIPName, "StaticIp", "ReleaseStaticIp", "Succeeded", "static ip released", staticIP.AvailabilityZone, staticIP.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) TagResource(resourceName string, tags map[string]string) ([]Operation, error) {
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" || len(tags) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	resourceType, locationAZ, locationRegion, existingTags, ok := s.resourceTagsLocked(resourceName)
	if !ok {
		return nil, ErrNotFound
	}
	for k, v := range tags {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		existingTags[key] = strings.TrimSpace(v)
	}
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, resourceName, resourceType, "TagResource", "Succeeded", "resource tagged", locationAZ, locationRegion, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) UntagResource(resourceName string, tagKeys []string) ([]Operation, error) {
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	resourceType, locationAZ, locationRegion, existingTags, ok := s.resourceTagsLocked(resourceName)
	if !ok {
		return nil, ErrNotFound
	}
	for _, key := range tagKeys {
		delete(existingTags, strings.TrimSpace(key))
	}
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, resourceName, resourceType, "UntagResource", "Succeeded", "resource untagged", locationAZ, locationRegion, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) GetRegions(includeAvailabilityZones, includeRelationalDatabaseAvailabilityZones bool) []Region {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Region, 0, len(s.regions))
	for _, region := range s.regions {
		item := region
		if !includeAvailabilityZones {
			item.AvailabilityZones = nil
		}
		if !includeRelationalDatabaseAvailabilityZones {
			item.DatabaseZones = nil
		}
		out = append(out, item)
	}
	return out
}

func (s *Service) ResourceNameFromARN(resourceARN string) string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return ""
	}
	idx := strings.LastIndex(resourceARN, "/")
	if idx < 0 || idx+1 >= len(resourceARN) {
		return ""
	}
	return strings.TrimSpace(resourceARN[idx+1:])
}

func (s *Service) GetOperation(id string) (Operation, bool) {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	op, ok := s.operationByID[id]
	if !ok {
		return Operation{}, false
	}
	return op, true
}

func (s *Service) GetOperations() []Operation {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneAndSortOperations(s.operations)
}

func (s *Service) GetOperationsForResource(resourceName string) []Operation {
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" {
		return []Operation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Operation, 0, len(s.operations))
	for _, op := range s.operations {
		if op.ResourceName == resourceName {
			out = append(out, op)
		}
	}
	return cloneAndSortOperations(out)
}

func (s *Service) transitionInstance(name, operationType string, code int32, state string) ([]Operation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	instance, ok := s.instances[name]
	if !ok {
		return nil, ErrNotFound
	}
	instance.StateCode = code
	instance.StateName = strings.TrimSpace(state)
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	ops := []Operation{newOperation(seq, name, "Instance", operationType, "Succeeded", strings.ToLower(operationType), instance.AvailabilityZone, instance.Region, now)}
	s.appendOperationsLocked(ops)
	return ops, nil
}

func (s *Service) resourceIdentityLocked(resourceName string) (resourceType, arn, availabilityZone, region string, ok bool) {
	if instance, exists := s.instances[resourceName]; exists {
		return "Instance", instance.ARN, instance.AvailabilityZone, instance.Region, true
	}
	if disk, exists := s.disks[resourceName]; exists {
		return "Disk", disk.ARN, disk.AvailabilityZone, disk.Region, true
	}
	if snapshot, exists := s.snapshots[resourceName]; exists {
		return "InstanceSnapshot", snapshot.ARN, snapshot.AvailabilityZone, snapshot.Region, true
	}
	if snapshot, exists := s.diskSnapshots[resourceName]; exists {
		return "DiskSnapshot", snapshot.ARN, snapshot.AvailabilityZone, snapshot.Region, true
	}
	if lb, exists := s.loadBalancers[resourceName]; exists {
		return "LoadBalancer", lb.ARN, lb.AvailabilityZone, lb.Region, true
	}
	if distribution, exists := s.distributions[resourceName]; exists {
		return "Distribution", distribution.ARN, distribution.AvailabilityZone, distribution.Region, true
	}
	if certificate, exists := s.certificates[resourceName]; exists {
		return "Certificate", certificate.ARN, DefaultRegion + "a", DefaultRegion, true
	}
	if domain, exists := s.domains[resourceName]; exists {
		return "Domain", domain.ARN, domain.AvailabilityZone, domain.Region, true
	}
	if bucket, exists := s.buckets[resourceName]; exists {
		return "Bucket", bucket.ARN, bucket.AvailabilityZone, bucket.Region, true
	}
	if containerService, exists := s.containerServices[resourceName]; exists {
		return "ContainerService", containerService.ARN, containerService.AvailabilityZone, containerService.Region, true
	}
	if relationalDatabase, exists := s.relationalDatabases[resourceName]; exists {
		return "RelationalDatabase", relationalDatabase.ARN, relationalDatabase.AvailabilityZone, relationalDatabase.Region, true
	}
	if relationalDatabaseSnapshot, exists := s.relationalDatabaseSnapshots[resourceName]; exists {
		return "RelationalDatabaseSnapshot", relationalDatabaseSnapshot.ARN, relationalDatabaseSnapshot.AvailabilityZone, relationalDatabaseSnapshot.Region, true
	}
	if cloudFormationStackRecord, exists := s.cloudFormationStackRecords[resourceName]; exists {
		return "CloudFormationStackRecord", cloudFormationStackRecord.ARN, cloudFormationStackRecord.AvailabilityZone, cloudFormationStackRecord.Region, true
	}
	if exportSnapshotRecord, exists := s.exportRecords[resourceName]; exists {
		return "ExportSnapshotRecord", exportSnapshotRecord.ARN, exportSnapshotRecord.AvailabilityZone, exportSnapshotRecord.Region, true
	}
	return "", "", "", "", false
}

func (s *Service) autoSnapshotAttachedDisksLocked(resourceName string) []AutoSnapshotAttachedDisk {
	instance, exists := s.instances[resourceName]
	if !exists {
		return nil
	}
	out := make([]AutoSnapshotAttachedDisk, 0, len(s.disks))
	for _, disk := range s.disks {
		if disk.AttachedTo != instance.Name {
			continue
		}
		path := disk.Path
		if path == "" {
			path = "/dev/xvdf"
		}
		out = append(out, AutoSnapshotAttachedDisk{
			Path:     path,
			SizeInGb: disk.SizeInGb,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func (s *Service) resourceTagsLocked(resourceName string) (resourceType string, locationAZ string, locationRegion string, tags map[string]string, ok bool) {
	if instance, exists := s.instances[resourceName]; exists {
		return "Instance", instance.AvailabilityZone, instance.Region, instance.Tags, true
	}
	if snapshot, exists := s.snapshots[resourceName]; exists {
		return "InstanceSnapshot", snapshot.AvailabilityZone, snapshot.Region, snapshot.Tags, true
	}
	if staticIP, exists := s.staticIPs[resourceName]; exists {
		if staticIP.Tags == nil {
			staticIP.Tags = map[string]string{}
		}
		return "StaticIp", staticIP.AvailabilityZone, staticIP.Region, staticIP.Tags, true
	}
	if disk, exists := s.disks[resourceName]; exists {
		if disk.Tags == nil {
			disk.Tags = map[string]string{}
		}
		return "Disk", disk.AvailabilityZone, disk.Region, disk.Tags, true
	}
	if snapshot, exists := s.diskSnapshots[resourceName]; exists {
		if snapshot.Tags == nil {
			snapshot.Tags = map[string]string{}
		}
		return "DiskSnapshot", snapshot.AvailabilityZone, snapshot.Region, snapshot.Tags, true
	}
	if lb, exists := s.loadBalancers[resourceName]; exists {
		if lb.Tags == nil {
			lb.Tags = map[string]string{}
		}
		return "LoadBalancer", lb.AvailabilityZone, lb.Region, lb.Tags, true
	}
	for _, byName := range s.lbTLSCerts {
		cert, exists := byName[resourceName]
		if !exists {
			continue
		}
		if cert.Tags == nil {
			cert.Tags = map[string]string{}
		}
		return "LoadBalancerTlsCertificate", cert.AvailabilityZone, cert.Region, cert.Tags, true
	}
	if keyPair, exists := s.keyPairs[resourceName]; exists {
		if keyPair.Tags == nil {
			keyPair.Tags = map[string]string{}
		}
		return "KeyPair", keyPair.AvailabilityZone, keyPair.Region, keyPair.Tags, true
	}
	if distribution, exists := s.distributions[resourceName]; exists {
		if distribution.Tags == nil {
			distribution.Tags = map[string]string{}
		}
		return "Distribution", distribution.AvailabilityZone, distribution.Region, distribution.Tags, true
	}
	if certificate, exists := s.certificates[resourceName]; exists {
		if certificate.Tags == nil {
			certificate.Tags = map[string]string{}
		}
		return "Certificate", DefaultRegion + "a", DefaultRegion, certificate.Tags, true
	}
	if domain, exists := s.domains[resourceName]; exists {
		if domain.Tags == nil {
			domain.Tags = map[string]string{}
		}
		return "Domain", domain.AvailabilityZone, domain.Region, domain.Tags, true
	}
	if bucket, exists := s.buckets[resourceName]; exists {
		if bucket.Tags == nil {
			bucket.Tags = map[string]string{}
		}
		return "Bucket", bucket.AvailabilityZone, bucket.Region, bucket.Tags, true
	}
	if containerService, exists := s.containerServices[resourceName]; exists {
		if containerService.Tags == nil {
			containerService.Tags = map[string]string{}
		}
		return "ContainerService", containerService.AvailabilityZone, containerService.Region, containerService.Tags, true
	}
	if relationalDatabase, exists := s.relationalDatabases[resourceName]; exists {
		if relationalDatabase.Tags == nil {
			relationalDatabase.Tags = map[string]string{}
		}
		return "RelationalDatabase", relationalDatabase.AvailabilityZone, relationalDatabase.Region, relationalDatabase.Tags, true
	}
	if relationalDatabaseSnapshot, exists := s.relationalDatabaseSnapshots[resourceName]; exists {
		if relationalDatabaseSnapshot.Tags == nil {
			relationalDatabaseSnapshot.Tags = map[string]string{}
		}
		return "RelationalDatabaseSnapshot", relationalDatabaseSnapshot.AvailabilityZone, relationalDatabaseSnapshot.Region, relationalDatabaseSnapshot.Tags, true
	}
	return "", "", "", nil, false
}

func (s *Service) defaultLoadBalancerTLSPolicyNameLocked() string {
	for _, policy := range s.lbTLSPolicies {
		if policy.IsDefault {
			return policy.Name
		}
	}
	if len(s.lbTLSPolicies) > 0 {
		return s.lbTLSPolicies[0].Name
	}
	return "TLS-1-2-2018-06"
}

func (s *Service) loadBalancerTLSPolicyExistsLocked(policyName string) bool {
	policyName = strings.TrimSpace(policyName)
	if policyName == "" {
		return false
	}
	for _, policy := range s.lbTLSPolicies {
		if strings.EqualFold(strings.TrimSpace(policy.Name), policyName) {
			return true
		}
	}
	return false
}

func (s *Service) refreshLoadBalancerDerivedLocked(lb *LoadBalancer) {
	if lb == nil {
		return
	}
	lb.Name = strings.TrimSpace(lb.Name)
	lb.AttachedInstances = dedupeStrings(lb.AttachedInstances)
	sort.Strings(lb.AttachedInstances)

	health := make([]LoadBalancerInstanceHealthSummary, 0, len(lb.AttachedInstances))
	for _, instanceName := range lb.AttachedInstances {
		summary := LoadBalancerInstanceHealthSummary{
			InstanceName:   instanceName,
			InstanceHealth: "healthy",
		}
		if instance, exists := s.instances[instanceName]; !exists || !strings.EqualFold(instance.StateName, "running") {
			summary.InstanceHealth = "unhealthy"
			summary.InstanceHealthReason = "Instance.InvalidState"
		}
		health = append(health, summary)
	}
	lb.InstanceHealthSummary = health

	certsByName := s.lbTLSCerts[lb.Name]
	if len(certsByName) == 0 {
		lb.TLSCertificateSummaries = []LoadBalancerTLSCertificateSummary{}
		lb.Protocol = "HTTP"
		lb.PublicPorts = []int32{80}
	} else {
		names := make([]string, 0, len(certsByName))
		for name := range certsByName {
			names = append(names, name)
		}
		sort.Strings(names)
		summaries := make([]LoadBalancerTLSCertificateSummary, 0, len(names))
		hasAttached := false
		for _, name := range names {
			isAttached := certsByName[name].IsAttached
			if isAttached {
				hasAttached = true
			}
			summaries = append(summaries, LoadBalancerTLSCertificateSummary{Name: name, IsAttached: isAttached})
		}
		lb.TLSCertificateSummaries = summaries
		if hasAttached {
			lb.Protocol = "HTTP_HTTPS"
			lb.PublicPorts = []int32{80, 443}
		} else {
			lb.Protocol = "HTTP"
			lb.PublicPorts = []int32{80}
		}
	}

	if lb.HealthCheckPath == "" {
		lb.HealthCheckPath = "/"
	}
	if !strings.HasPrefix(lb.HealthCheckPath, "/") {
		lb.HealthCheckPath = "/" + lb.HealthCheckPath
	}
	if lb.ConfigurationOptions == nil {
		lb.ConfigurationOptions = map[string]string{}
	}
	lb.ConfigurationOptions["HealthCheckPath"] = lb.HealthCheckPath
	if _, ok := lb.ConfigurationOptions["SessionStickinessEnabled"]; !ok {
		lb.ConfigurationOptions["SessionStickinessEnabled"] = "false"
	}
	if _, ok := lb.ConfigurationOptions["SessionStickiness_LB_CookieDurationSeconds"]; !ok {
		lb.ConfigurationOptions["SessionStickiness_LB_CookieDurationSeconds"] = "0"
	}
	lb.ConfigurationOptions["HttpsRedirectionEnabled"] = strconv.FormatBool(lb.HTTPSRedirectionEnabled)
	if lb.TLSPolicyName == "" {
		lb.TLSPolicyName = s.defaultLoadBalancerTLSPolicyNameLocked()
	}
	lb.ConfigurationOptions["TlsPolicyName"] = lb.TLSPolicyName
	if lb.ResourceType == "" {
		lb.ResourceType = "LoadBalancer"
	}
	if lb.State == "" {
		lb.State = "active"
	}
}

func normalizeDomainEntry(in DomainEntry) DomainEntry {
	out := DomainEntry{
		ID:      strings.TrimSpace(in.ID),
		IsAlias: in.IsAlias,
		Name:    strings.TrimSpace(in.Name),
		Options: cloneStringMap(in.Options),
		Target:  strings.TrimSpace(in.Target),
		Type:    strings.ToUpper(strings.TrimSpace(in.Type)),
	}
	return out
}

func findDomainEntryIndex(entries []DomainEntry, wanted DomainEntry) int {
	wanted = normalizeDomainEntry(wanted)
	for idx, entry := range entries {
		if wanted.ID != "" && strings.TrimSpace(entry.ID) == wanted.ID {
			return idx
		}
		if strings.EqualFold(strings.TrimSpace(entry.Name), wanted.Name) &&
			strings.EqualFold(strings.TrimSpace(entry.Type), wanted.Type) {
			return idx
		}
	}
	return -1
}

func sortDomainEntries(entries []DomainEntry) {
	sort.Slice(entries, func(i, j int) bool {
		leftName := strings.ToLower(strings.TrimSpace(entries[i].Name))
		rightName := strings.ToLower(strings.TrimSpace(entries[j].Name))
		if leftName == rightName {
			leftType := strings.ToUpper(strings.TrimSpace(entries[i].Type))
			rightType := strings.ToUpper(strings.TrimSpace(entries[j].Type))
			if leftType == rightType {
				return strings.ToLower(strings.TrimSpace(entries[i].Target)) < strings.ToLower(strings.TrimSpace(entries[j].Target))
			}
			return leftType < rightType
		}
		return leftName < rightName
	})
}

func normalizeDistributionOrigin(in DistributionOrigin) DistributionOrigin {
	out := DistributionOrigin{
		Name:            strings.TrimSpace(in.Name),
		ProtocolPolicy:  firstNonEmptyString(in.ProtocolPolicy, "http-only"),
		RegionName:      firstNonEmptyString(in.RegionName, DefaultRegion),
		ResourceType:    firstNonEmptyString(in.ResourceType, "Instance"),
		ResponseTimeout: in.ResponseTimeout,
	}
	if out.ResponseTimeout <= 0 {
		out.ResponseTimeout = 30
	}
	return out
}

func normalizeDistributionCacheSettings(in DistributionCacheSettings) DistributionCacheSettings {
	out := DistributionCacheSettings{
		AllowedHTTPMethods: firstNonEmptyString(in.AllowedHTTPMethods, "GET,HEAD"),
		CachedHTTPMethods:  firstNonEmptyString(in.CachedHTTPMethods, "GET,HEAD"),
		DefaultTTL:         in.DefaultTTL,
		MaximumTTL:         in.MaximumTTL,
		MinimumTTL:         in.MinimumTTL,
		ForwardedCookies: DistributionCookieObject{
			Option:           firstNonEmptyString(in.ForwardedCookies.Option, "none"),
			CookiesAllowList: dedupeStrings(in.ForwardedCookies.CookiesAllowList),
		},
		ForwardedHeaders: DistributionHeaderObject{
			Option:           firstNonEmptyString(in.ForwardedHeaders.Option, "none"),
			HeadersAllowList: dedupeStrings(in.ForwardedHeaders.HeadersAllowList),
		},
		ForwardedQueryStrings: DistributionQueryStringObject{
			Option:                in.ForwardedQueryStrings.Option,
			QueryStringsAllowList: dedupeStrings(in.ForwardedQueryStrings.QueryStringsAllowList),
		},
	}
	if out.ForwardedQueryStrings.Option == nil {
		defaultOption := false
		out.ForwardedQueryStrings.Option = &defaultOption
	}
	if out.DefaultTTL < 0 {
		out.DefaultTTL = 0
	}
	if out.MaximumTTL < 0 {
		out.MaximumTTL = 0
	}
	if out.MinimumTTL < 0 {
		out.MinimumTTL = 0
	}
	return out
}

func normalizeDistributionCacheBehaviors(in []DistributionCacheBehaviorPerPath) []DistributionCacheBehaviorPerPath {
	if len(in) == 0 {
		return []DistributionCacheBehaviorPerPath{}
	}
	seen := map[string]struct{}{}
	out := make([]DistributionCacheBehaviorPerPath, 0, len(in))
	for _, item := range in {
		behavior := strings.TrimSpace(item.Behavior)
		path := strings.TrimSpace(item.Path)
		if behavior == "" || path == "" {
			continue
		}
		key := behavior + "\x00" + path
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, DistributionCacheBehaviorPerPath{
			Behavior: behavior,
			Path:     path,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].Behavior < out[j].Behavior
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func hasMetricStatistic(stats []string, wanted string) bool {
	wanted = strings.TrimSpace(strings.ToLower(wanted))
	for _, item := range stats {
		if strings.TrimSpace(strings.ToLower(item)) == wanted {
			return true
		}
	}
	return false
}

func hasAnyMetricStatistic(stats []string) bool {
	for _, wanted := range []string{"Minimum", "Maximum", "Sum", "Average", "SampleCount"} {
		if hasMetricStatistic(stats, wanted) {
			return true
		}
	}
	return false
}

func (s *Service) appendContainerLogLocked(serviceName, containerName, message string, ts time.Time) {
	serviceName = strings.TrimSpace(serviceName)
	containerName = strings.TrimSpace(containerName)
	message = strings.TrimSpace(message)
	if serviceName == "" || containerName == "" || message == "" {
		return
	}
	if _, ok := s.containerLogs[serviceName]; !ok {
		s.containerLogs[serviceName] = map[string][]ContainerServiceLogEvent{}
	}
	s.containerLogs[serviceName][containerName] = append(
		s.containerLogs[serviceName][containerName],
		ContainerServiceLogEvent{CreatedAt: ts, Message: message},
	)
}

func (s *Service) appendRelationalDatabaseEventLocked(relationalDatabaseName, category, message string, ts time.Time) {
	relationalDatabaseName = strings.TrimSpace(relationalDatabaseName)
	category = strings.TrimSpace(category)
	message = strings.TrimSpace(message)
	if relationalDatabaseName == "" || message == "" {
		return
	}
	event := RelationalDatabaseEvent{
		CreatedAt:       ts,
		EventCategories: []string{firstNonEmptyString(category, "notification")},
		Message:         message,
		Resource:        relationalDatabaseName,
	}
	s.relationalDatabaseEvents[relationalDatabaseName] = append(s.relationalDatabaseEvents[relationalDatabaseName], event)
}

func (s *Service) appendSetupHistoryLocked(resourceName, command, standardOutput, standardError, status, version string) {
	resourceName = strings.TrimSpace(resourceName)
	command = strings.TrimSpace(command)
	status = strings.TrimSpace(status)
	version = strings.TrimSpace(version)
	if resourceName == "" || command == "" {
		return
	}
	if status == "" {
		status = "succeeded"
	}
	if version == "" {
		version = "1.0.0"
	}

	resourceType, arn, availabilityZone, region, ok := s.resourceIdentityLocked(resourceName)
	if !ok {
		region = DefaultRegion
		availabilityZone = DefaultRegion + "a"
		resourceType = "Unknown"
		arn = fmt.Sprintf("arn:aws:lightsail:%s:%s:Resource/%s", region, DefaultAccountID, resourceName)
	}
	now := time.Now().UTC()
	seq := atomic.AddUint64(&s.seq, 1)
	s.setupHistory[resourceName] = append(s.setupHistory[resourceName], SetupHistory{
		ExecutionDetails: []SetupExecutionDetail{
			{
				Command:        command,
				DateTime:       now,
				Name:           resourceName,
				StandardError:  standardError,
				StandardOutput: standardOutput,
				Status:         status,
				Version:        version,
			},
		},
		OperationID: fmt.Sprintf("setup-%d", seq),
		Request:     nil,
		Resource: SetupHistoryResource{
			ARN:              arn,
			CreatedAt:        now,
			AvailabilityZone: availabilityZone,
			Region:           region,
			Name:             resourceName,
			ResourceType:     resourceType,
		},
		Status: status,
	})
}

func defaultRelationalDatabaseLogStreams() []string {
	return []string{"error", "general", "slowquery"}
}

func defaultRelationalDatabaseLogEvents(relationalDatabaseName string, now time.Time) map[string][]RelationalDatabaseLogEvent {
	relationalDatabaseName = strings.TrimSpace(relationalDatabaseName)
	if relationalDatabaseName == "" {
		relationalDatabaseName = "database"
	}
	return map[string][]RelationalDatabaseLogEvent{
		"error": {
			{CreatedAt: now.Add(-2 * time.Minute), Message: "InnoDB: startup complete"},
			{CreatedAt: now.Add(-1 * time.Minute), Message: "ready for connections"},
		},
		"general": {
			{CreatedAt: now.Add(-90 * time.Second), Message: fmt.Sprintf("Connect root@localhost on %s", relationalDatabaseName)},
			{CreatedAt: now.Add(-30 * time.Second), Message: "Query SELECT 1"},
		},
		"slowquery": {
			{CreatedAt: now.Add(-45 * time.Second), Message: "Query_time: 2.10 Lock_time: 0.01 Rows_sent: 10 Rows_examined: 1000"},
		},
	}
}

func maxInt32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func matchesContainerLogFilter(message, filterPattern string) bool {
	messageLower := strings.ToLower(strings.TrimSpace(message))
	filterPattern = strings.TrimSpace(filterPattern)
	if filterPattern == "" {
		return true
	}
	terms := strings.Fields(filterPattern)
	if len(terms) == 0 {
		return true
	}
	optionalTerms := []string{}
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		switch {
		case strings.HasPrefix(term, "-"):
			excluded := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(term, "-")))
			if excluded != "" && strings.Contains(messageLower, excluded) {
				return false
			}
		case strings.HasPrefix(term, "?"):
			term = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(term, "?")))
			if term == "" {
				continue
			}
			optionalTerms = append(optionalTerms, term)
		default:
			term = strings.ToLower(term)
			if !strings.Contains(messageLower, term) {
				return false
			}
		}
	}
	if len(optionalTerms) > 0 {
		for _, term := range optionalTerms {
			if strings.Contains(messageLower, term) {
				return true
			}
		}
		return false
	}
	return true
}

func normalizeBucketVersioning(versioning string) string {
	switch strings.ToLower(strings.TrimSpace(versioning)) {
	case "enabled":
		return "Enabled"
	case "suspended":
		return "Suspended"
	case "neverenabled":
		return "NeverEnabled"
	default:
		return ""
	}
}

func normalizeBucketAccessType(accessType string) string {
	switch strings.ToLower(strings.TrimSpace(accessType)) {
	case "public":
		return "public"
	case "private":
		return "private"
	default:
		return ""
	}
}

func normalizeBucketMetricName(metricName string) string {
	switch strings.ToLower(strings.TrimSpace(metricName)) {
	case "bucketsizebytes":
		return "BucketSizeBytes"
	case "numberofobjects":
		return "NumberOfObjects"
	default:
		return ""
	}
}

func normalizeInstanceMetricName(metricName string) string {
	switch strings.ToLower(strings.TrimSpace(metricName)) {
	case "burstcapacitypercentage":
		return "BurstCapacityPercentage"
	case "burstcapacitytime":
		return "BurstCapacityTime"
	case "cpuutilization":
		return "CPUUtilization"
	case "networkin":
		return "NetworkIn"
	case "networkout":
		return "NetworkOut"
	case "statuscheckfailed":
		return "StatusCheckFailed"
	case "statuscheckfailed_instance":
		return "StatusCheckFailed_Instance"
	case "statuscheckfailed_system":
		return "StatusCheckFailed_System"
	case "metadatanotoken":
		return "MetadataNoToken"
	default:
		return ""
	}
}

func normalizeLoadBalancerMetricName(metricName string) string {
	switch strings.ToLower(strings.TrimSpace(metricName)) {
	case "clienttlsnegotiationerrorcount":
		return "ClientTLSNegotiationErrorCount"
	case "healthyhostcount":
		return "HealthyHostCount"
	case "unhealthyhostcount":
		return "UnhealthyHostCount"
	case "httpcode_lb_4xx_count":
		return "HTTPCode_LB_4XX_Count"
	case "httpcode_lb_5xx_count":
		return "HTTPCode_LB_5XX_Count"
	case "httpcode_instance_2xx_count":
		return "HTTPCode_Instance_2XX_Count"
	case "httpcode_instance_3xx_count":
		return "HTTPCode_Instance_3XX_Count"
	case "httpcode_instance_4xx_count":
		return "HTTPCode_Instance_4XX_Count"
	case "httpcode_instance_5xx_count":
		return "HTTPCode_Instance_5XX_Count"
	case "instanceresponsetime":
		return "InstanceResponseTime"
	case "rejectedconnectioncount":
		return "RejectedConnectionCount"
	case "requestcount":
		return "RequestCount"
	default:
		return ""
	}
}

func normalizeLoadBalancerAttributeName(attributeName string) string {
	switch strings.ToLower(strings.TrimSpace(attributeName)) {
	case "healthcheckpath":
		return "HealthCheckPath"
	case "sessionstickinessenabled":
		return "SessionStickinessEnabled"
	case "sessionstickiness_lb_cookiedurationseconds":
		return "SessionStickiness_LB_CookieDurationSeconds"
	case "httpsredirectionenabled":
		return "HttpsRedirectionEnabled"
	case "tlspolicyname":
		return "TlsPolicyName"
	default:
		return ""
	}
}

func normalizeLoadBalancerIPAddressType(ipAddressType string) string {
	switch strings.ToLower(strings.TrimSpace(ipAddressType)) {
	case "", "dualstack":
		return "dualstack"
	case "ipv4":
		return "ipv4"
	case "ipv6":
		return "ipv6"
	default:
		return ""
	}
}

func normalizeContactProtocol(protocol string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "email":
		return "Email", true
	case "sms":
		return "SMS", true
	default:
		return "", false
	}
}

func normalizeContainerServicePowerName(power string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(power)) {
	case "nano":
		return "nano", true
	case "micro":
		return "micro", true
	case "small":
		return "small", true
	case "medium":
		return "medium", true
	case "large":
		return "large", true
	case "xlarge":
		return "xlarge", true
	default:
		return "", false
	}
}

func normalizeContainerServiceMetricName(metricName string) string {
	switch strings.ToLower(strings.TrimSpace(metricName)) {
	case "cpuutilization":
		return "CPUUtilization"
	case "memoryutilization":
		return "MemoryUtilization"
	default:
		return ""
	}
}

func normalizeRelationalDatabaseMetricName(metricName string) string {
	switch strings.ToLower(strings.TrimSpace(metricName)) {
	case "cpuutilization":
		return "CPUUtilization"
	case "databaseconnections":
		return "DatabaseConnections"
	case "diskqueuedepth":
		return "DiskQueueDepth"
	case "freestoragespace":
		return "FreeStorageSpace"
	case "networkreceivethroughput":
		return "NetworkReceiveThroughput"
	case "networktransmitthroughput":
		return "NetworkTransmitThroughput"
	default:
		return ""
	}
}

func normalizeContainerServiceProtocol(protocol string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "HTTP":
		return "HTTP", true
	case "HTTPS":
		return "HTTPS", true
	case "TCP":
		return "TCP", true
	case "UDP":
		return "UDP", true
	default:
		return "", false
	}
}

func relationalDatabaseEngineFromBlueprint(blueprintID string) (engine, version string, port int32) {
	blueprint := strings.ToLower(strings.TrimSpace(blueprintID))
	if strings.Contains(blueprint, "postgres") || strings.Contains(blueprint, "pgsql") {
		return "postgres", firstNonEmptyString(relationalDatabaseVersionFromBlueprint(blueprint), "13.7"), 5432
	}
	return "mysql", firstNonEmptyString(relationalDatabaseVersionFromBlueprint(blueprint), "8.0"), 3306
}

func relationalDatabaseVersionFromBlueprint(blueprint string) string {
	parts := strings.FieldsFunc(blueprint, func(r rune) bool {
		return !(r >= '0' && r <= '9' || r == '.')
	})
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Count(part, ".") > 0 {
			return part
		}
	}
	return ""
}

func relationalDatabaseHardwareFromBundle(bundleID string) (cpuCount, diskSizeInGb int32, ramSizeInGb float32) {
	bundle := strings.ToLower(strings.TrimSpace(bundleID))
	switch {
	case strings.Contains(bundle, "nano"):
		return 1, 20, 1
	case strings.Contains(bundle, "micro"):
		return 1, 40, 1
	case strings.Contains(bundle, "small"):
		return 1, 80, 2
	case strings.Contains(bundle, "medium"):
		return 2, 120, 4
	case strings.Contains(bundle, "large"):
		return 4, 240, 8
	default:
		return 1, 40, 1
	}
}

func validBucketMetricUnit(metricName, unit string) bool {
	switch normalizeBucketMetricName(metricName) {
	case "BucketSizeBytes":
		return strings.EqualFold(strings.TrimSpace(unit), "Bytes")
	case "NumberOfObjects":
		return strings.EqualFold(strings.TrimSpace(unit), "Count")
	default:
		return false
	}
}

func validInstanceMetricUnit(metricName, unit string) bool {
	unit = strings.TrimSpace(strings.ToLower(unit))
	switch normalizeInstanceMetricName(metricName) {
	case "BurstCapacityPercentage", "CPUUtilization":
		return unit == "percent"
	case "BurstCapacityTime":
		return unit == "seconds"
	case "NetworkIn", "NetworkOut":
		return unit == "bytes"
	case "StatusCheckFailed", "StatusCheckFailed_Instance", "StatusCheckFailed_System", "MetadataNoToken":
		return unit == "count"
	default:
		return false
	}
}

func validLoadBalancerMetricUnit(metricName, unit string) bool {
	unit = strings.TrimSpace(strings.ToLower(unit))
	switch normalizeLoadBalancerMetricName(metricName) {
	case "InstanceResponseTime":
		return unit == "seconds"
	case "ClientTLSNegotiationErrorCount", "HealthyHostCount", "UnhealthyHostCount", "HTTPCode_LB_4XX_Count", "HTTPCode_LB_5XX_Count", "HTTPCode_Instance_2XX_Count", "HTTPCode_Instance_3XX_Count", "HTTPCode_Instance_4XX_Count", "HTTPCode_Instance_5XX_Count", "RejectedConnectionCount", "RequestCount":
		return unit == "count"
	default:
		return false
	}
}

func validRelationalDatabaseMetricUnit(metricName, unit string) bool {
	unit = strings.TrimSpace(strings.ToLower(unit))
	switch normalizeRelationalDatabaseMetricName(metricName) {
	case "CPUUtilization":
		return unit == "percent"
	case "DatabaseConnections", "DiskQueueDepth":
		return unit == "count"
	case "FreeStorageSpace":
		return unit == "bytes"
	case "NetworkReceiveThroughput", "NetworkTransmitThroughput":
		return unit == "bytes/second"
	default:
		return false
	}
}

func (s *Service) bundleExistsLocked(bundleID string) bool {
	for _, bundle := range s.distributionBundles {
		if strings.EqualFold(strings.TrimSpace(bundle.BundleID), strings.TrimSpace(bundleID)) && bundle.IsActive {
			return true
		}
	}
	return false
}

func (s *Service) bucketBundleExistsLocked(bundleID string, includeInactive bool) bool {
	for _, bundle := range s.bucketBundles {
		if !strings.EqualFold(strings.TrimSpace(bundle.BundleID), strings.TrimSpace(bundleID)) {
			continue
		}
		if includeInactive || bundle.IsActive {
			return true
		}
	}
	return false
}

func (s *Service) containerServicePowerByNameLocked(name string) (ContainerServicePower, bool) {
	name, ok := normalizeContainerServicePowerName(name)
	if !ok {
		return ContainerServicePower{}, false
	}
	for _, power := range s.containerServicePowers {
		if strings.EqualFold(power.Name, name) {
			return power, true
		}
	}
	return ContainerServicePower{}, false
}

func cloneInstance(in *Instance) Instance {
	if in == nil {
		return Instance{}
	}
	out := *in
	out.IPv6Addresses = append([]string(nil), in.IPv6Addresses...)
	out.PortStates = cloneInstancePortStates(in.PortStates)
	out.MetadataOptions = in.MetadataOptions
	out.HostKeys = cloneHostKeyAttributes(in.HostKeys)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneInstanceSnapshot(in *InstanceSnapshot) InstanceSnapshot {
	if in == nil {
		return InstanceSnapshot{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneStaticIP(in *StaticIP) StaticIP {
	if in == nil {
		return StaticIP{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneDisk(in *Disk) Disk {
	if in == nil {
		return Disk{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneDiskSnapshot(in *DiskSnapshot) DiskSnapshot {
	if in == nil {
		return DiskSnapshot{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneExportSnapshotRecord(in *ExportSnapshotRecord) ExportSnapshotRecord {
	if in == nil {
		return ExportSnapshotRecord{}
	}
	return *in
}

func cloneSetupHistory(in []SetupHistory) []SetupHistory {
	if len(in) == 0 {
		return []SetupHistory{}
	}
	out := make([]SetupHistory, len(in))
	for i := range in {
		out[i] = SetupHistory{
			ExecutionDetails: make([]SetupExecutionDetail, len(in[i].ExecutionDetails)),
			OperationID:      in[i].OperationID,
			Request:          nil,
			Resource:         in[i].Resource,
			Status:           in[i].Status,
		}
		copy(out[i].ExecutionDetails, in[i].ExecutionDetails)
		if in[i].Request != nil {
			req := *in[i].Request
			req.DomainNames = append([]string(nil), in[i].Request.DomainNames...)
			out[i].Request = &req
		}
	}
	return out
}

func cloneCloudFormationStackRecord(in *CloudFormationStackRecord) CloudFormationStackRecord {
	if in == nil {
		return CloudFormationStackRecord{}
	}
	out := *in
	out.SourceInfo = append([]CloudFormationStackSourceInfo(nil), in.SourceInfo...)
	return out
}

func cloneAlarm(in *Alarm) Alarm {
	if in == nil {
		return Alarm{}
	}
	out := *in
	out.ContactProtocols = append([]string(nil), in.ContactProtocols...)
	out.NotificationTriggers = append([]string(nil), in.NotificationTriggers...)
	return out
}

func cloneAutoSnapshotDetails(in AutoSnapshotDetails) AutoSnapshotDetails {
	out := in
	out.FromAttachedDisks = append([]AutoSnapshotAttachedDisk(nil), in.FromAttachedDisks...)
	return out
}

func cloneLoadBalancer(in *LoadBalancer) LoadBalancer {
	if in == nil {
		return LoadBalancer{}
	}
	out := *in
	out.InstanceHealthSummary = cloneLoadBalancerInstanceHealthSummaries(in.InstanceHealthSummary)
	out.PublicPorts = append([]int32(nil), in.PublicPorts...)
	out.Tags = cloneStringMap(in.Tags)
	out.TLSCertificateSummaries = cloneLoadBalancerTLSCertificateSummaries(in.TLSCertificateSummaries)
	out.ConfigurationOptions = cloneStringMap(in.ConfigurationOptions)
	out.AttachedInstances = append([]string(nil), in.AttachedInstances...)
	return out
}

func cloneLoadBalancerInstanceHealthSummaries(in []LoadBalancerInstanceHealthSummary) []LoadBalancerInstanceHealthSummary {
	if len(in) == 0 {
		return []LoadBalancerInstanceHealthSummary{}
	}
	out := make([]LoadBalancerInstanceHealthSummary, len(in))
	copy(out, in)
	return out
}

func cloneLoadBalancerTLSCertificateSummaries(in []LoadBalancerTLSCertificateSummary) []LoadBalancerTLSCertificateSummary {
	if len(in) == 0 {
		return []LoadBalancerTLSCertificateSummary{}
	}
	out := make([]LoadBalancerTLSCertificateSummary, len(in))
	copy(out, in)
	return out
}

func cloneLoadBalancerTLSCertificate(in *LoadBalancerTLSCertificate) LoadBalancerTLSCertificate {
	if in == nil {
		return LoadBalancerTLSCertificate{}
	}
	out := *in
	out.SubjectAlternativeNames = append([]string(nil), in.SubjectAlternativeNames...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneCertificate(in *Certificate) Certificate {
	if in == nil {
		return Certificate{}
	}
	out := *in
	out.DomainValidationRecords = cloneCertificateDomainValidationRecords(in.DomainValidationRecords)
	out.RenewalSummary = cloneCertificateRenewalSummary(in.RenewalSummary)
	out.SubjectAlternativeNames = append([]string(nil), in.SubjectAlternativeNames...)
	out.Tags = cloneStringMap(in.Tags)
	out.AttachedDistributions = append([]string(nil), in.AttachedDistributions...)
	return out
}

func cloneCertificateDomainValidationRecords(in []CertificateDomainValidationRecord) []CertificateDomainValidationRecord {
	if len(in) == 0 {
		return []CertificateDomainValidationRecord{}
	}
	out := make([]CertificateDomainValidationRecord, len(in))
	copy(out, in)
	return out
}

func cloneCertificateRenewalSummary(in CertificateRenewalSummary) CertificateRenewalSummary {
	out := in
	out.DomainValidationRecords = cloneCertificateDomainValidationRecords(in.DomainValidationRecords)
	return out
}

func cloneDomain(in *Domain) Domain {
	if in == nil {
		return Domain{}
	}
	out := *in
	out.DomainEntries = cloneDomainEntries(in.DomainEntries)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneDomainEntries(in []DomainEntry) []DomainEntry {
	if len(in) == 0 {
		return []DomainEntry{}
	}
	out := make([]DomainEntry, len(in))
	for idx := range in {
		out[idx] = DomainEntry{
			ID:      in[idx].ID,
			IsAlias: in[idx].IsAlias,
			Name:    in[idx].Name,
			Options: cloneStringMap(in[idx].Options),
			Target:  in[idx].Target,
			Type:    in[idx].Type,
		}
	}
	return out
}

func cloneDistribution(in *Distribution) Distribution {
	if in == nil {
		return Distribution{}
	}
	out := *in
	out.AlternativeDomainNames = append([]string(nil), in.AlternativeDomainNames...)
	out.CacheBehaviorSettings = cloneDistributionCacheSettings(in.CacheBehaviorSettings)
	out.CacheBehaviors = cloneDistributionCacheBehaviors(in.CacheBehaviors)
	out.Origin = cloneDistributionOrigin(in.Origin)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneDistributionOrigin(in DistributionOrigin) DistributionOrigin {
	return DistributionOrigin{
		Name:            in.Name,
		ProtocolPolicy:  in.ProtocolPolicy,
		RegionName:      in.RegionName,
		ResourceType:    in.ResourceType,
		ResponseTimeout: in.ResponseTimeout,
	}
}

func cloneDistributionCacheSettings(in DistributionCacheSettings) DistributionCacheSettings {
	return DistributionCacheSettings{
		AllowedHTTPMethods: in.AllowedHTTPMethods,
		CachedHTTPMethods:  in.CachedHTTPMethods,
		DefaultTTL:         in.DefaultTTL,
		MaximumTTL:         in.MaximumTTL,
		MinimumTTL:         in.MinimumTTL,
		ForwardedCookies: DistributionCookieObject{
			Option:           in.ForwardedCookies.Option,
			CookiesAllowList: append([]string(nil), in.ForwardedCookies.CookiesAllowList...),
		},
		ForwardedHeaders: DistributionHeaderObject{
			Option:           in.ForwardedHeaders.Option,
			HeadersAllowList: append([]string(nil), in.ForwardedHeaders.HeadersAllowList...),
		},
		ForwardedQueryStrings: DistributionQueryStringObject{
			Option:                cloneBoolPtr(in.ForwardedQueryStrings.Option),
			QueryStringsAllowList: append([]string(nil), in.ForwardedQueryStrings.QueryStringsAllowList...),
		},
	}
}

func cloneDistributionCacheBehaviors(in []DistributionCacheBehaviorPerPath) []DistributionCacheBehaviorPerPath {
	if len(in) == 0 {
		return []DistributionCacheBehaviorPerPath{}
	}
	out := make([]DistributionCacheBehaviorPerPath, len(in))
	copy(out, in)
	return out
}

func cloneKeyPair(in *KeyPair) KeyPair {
	if in == nil {
		return KeyPair{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneBucketAccessKey(in *BucketAccessKey) BucketAccessKey {
	if in == nil {
		return BucketAccessKey{}
	}
	out := *in
	out.LastUsed = cloneBucketAccessKeyLastUsed(in.LastUsed)
	return out
}

func cloneBucketAccessKeyLastUsed(in *BucketAccessKeyLastUsed) *BucketAccessKeyLastUsed {
	if in == nil {
		return nil
	}
	out := *in
	if in.LastUsedDate != nil {
		ts := *in.LastUsedDate
		out.LastUsedDate = &ts
	}
	return &out
}

func cloneContactMethod(in *ContactMethod) ContactMethod {
	if in == nil {
		return ContactMethod{}
	}
	return *in
}

func cloneBucket(in *Bucket) Bucket {
	if in == nil {
		return Bucket{}
	}
	out := *in
	out.ReadonlyAccessAccounts = append([]string(nil), in.ReadonlyAccessAccounts...)
	out.ResourcesReceivingAccess = cloneBucketResourcesReceivingAccess(in.ResourcesReceivingAccess)
	out.AccessRules = cloneBucketAccessRules(in.AccessRules)
	out.AccessLogConfig = cloneBucketAccessLogConfig(in.AccessLogConfig)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneContainerService(in *ContainerService) ContainerService {
	if in == nil {
		return ContainerService{}
	}
	out := *in
	out.PublicDomainNames = cloneStringSliceMap(in.PublicDomainNames)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneRelationalDatabase(in *RelationalDatabase) RelationalDatabase {
	if in == nil {
		return RelationalDatabase{}
	}
	out := *in
	out.PendingModifiedValues = clonePendingModifiedRelationalDatabaseValues(in.PendingModifiedValues)
	out.PendingMaintenanceActions = dedupeStrings(in.PendingMaintenanceActions)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func clonePendingModifiedRelationalDatabaseValues(in *PendingModifiedRelationalDatabaseValues) *PendingModifiedRelationalDatabaseValues {
	if in == nil {
		return nil
	}
	out := *in
	out.BackupRetentionEnabled = cloneBoolPtr(in.BackupRetentionEnabled)
	out.EngineVersion = cloneStringPtr(in.EngineVersion)
	out.MasterUserPassword = cloneStringPtr(in.MasterUserPassword)
	return &out
}

func cloneRelationalDatabaseSnapshot(in *RelationalDatabaseSnapshot) RelationalDatabaseSnapshot {
	if in == nil {
		return RelationalDatabaseSnapshot{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneRelationalDatabaseEvents(in []RelationalDatabaseEvent) []RelationalDatabaseEvent {
	if len(in) == 0 {
		return []RelationalDatabaseEvent{}
	}
	out := make([]RelationalDatabaseEvent, len(in))
	for idx := range in {
		out[idx] = RelationalDatabaseEvent{
			CreatedAt:       in[idx].CreatedAt,
			EventCategories: dedupeStrings(in[idx].EventCategories),
			Message:         in[idx].Message,
			Resource:        in[idx].Resource,
		}
	}
	return out
}

func cloneRelationalDatabaseLogEvents(in []RelationalDatabaseLogEvent) []RelationalDatabaseLogEvent {
	if len(in) == 0 {
		return []RelationalDatabaseLogEvent{}
	}
	out := make([]RelationalDatabaseLogEvent, len(in))
	copy(out, in)
	return out
}

func cloneContainerServiceDeployments(in []ContainerServiceDeployment) []ContainerServiceDeployment {
	if len(in) == 0 {
		return []ContainerServiceDeployment{}
	}
	out := make([]ContainerServiceDeployment, len(in))
	for idx := range in {
		out[idx] = cloneContainerServiceDeployment(in[idx])
	}
	return out
}

func cloneContainerServiceDeployment(in ContainerServiceDeployment) ContainerServiceDeployment {
	return ContainerServiceDeployment{
		Containers:     cloneContainerServiceContainers(in.Containers),
		CreatedAt:      in.CreatedAt,
		PublicEndpoint: cloneContainerServiceEndpoint(in.PublicEndpoint),
		State:          in.State,
		Version:        in.Version,
	}
}

func cloneContainerServiceContainers(in map[string]ContainerServiceContainer) map[string]ContainerServiceContainer {
	if len(in) == 0 {
		return map[string]ContainerServiceContainer{}
	}
	out := make(map[string]ContainerServiceContainer, len(in))
	for key, container := range in {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		out[trimmed] = cloneContainerServiceContainer(container)
	}
	return out
}

func cloneContainerServiceContainer(in ContainerServiceContainer) ContainerServiceContainer {
	return ContainerServiceContainer{
		Command:     dedupeStrings(in.Command),
		Environment: cloneStringMap(in.Environment),
		Image:       strings.TrimSpace(in.Image),
		Ports:       cloneStringMap(in.Ports),
	}
}

func cloneContainerServiceEndpoint(in *ContainerServiceEndpoint) *ContainerServiceEndpoint {
	if in == nil {
		return nil
	}
	out := *in
	out.ContainerName = strings.TrimSpace(in.ContainerName)
	out.HealthCheck = cloneContainerServiceHealthCheckConfig(in.HealthCheck)
	return &out
}

func cloneContainerServiceHealthCheckConfig(in *ContainerServiceHealthCheckConfig) *ContainerServiceHealthCheckConfig {
	if in == nil {
		return nil
	}
	out := *in
	out.HealthyThreshold = cloneInt32Ptr(in.HealthyThreshold)
	out.IntervalSeconds = cloneInt32Ptr(in.IntervalSeconds)
	out.Path = cloneStringPtr(in.Path)
	out.SuccessCodes = cloneStringPtr(in.SuccessCodes)
	out.TimeoutSeconds = cloneInt32Ptr(in.TimeoutSeconds)
	out.UnhealthyThreshold = cloneInt32Ptr(in.UnhealthyThreshold)
	return &out
}

func cloneContainerImage(in *ContainerImage) ContainerImage {
	if in == nil {
		return ContainerImage{}
	}
	return *in
}

func cloneContainerServiceLogEvents(in []ContainerServiceLogEvent) []ContainerServiceLogEvent {
	if len(in) == 0 {
		return []ContainerServiceLogEvent{}
	}
	out := make([]ContainerServiceLogEvent, len(in))
	copy(out, in)
	return out
}

func cloneBucketResourcesReceivingAccess(in []BucketResourceReceivingAccess) []BucketResourceReceivingAccess {
	if len(in) == 0 {
		return []BucketResourceReceivingAccess{}
	}
	out := make([]BucketResourceReceivingAccess, len(in))
	copy(out, in)
	return out
}

func cloneBucketAccessRules(in *BucketAccessRules) *BucketAccessRules {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneBucketAccessLogConfig(in *BucketAccessLogConfig) *BucketAccessLogConfig {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneStringSliceMap(in map[string][]string) map[string][]string {
	if len(in) == 0 {
		return map[string][]string{}
	}
	out := make(map[string][]string, len(in))
	for key, values := range in {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		out[trimmed] = dedupeStrings(values)
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneBoolPtr(in *bool) *bool {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneInt32Ptr(in *int32) *int32 {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneStringPtr(in *string) *string {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneAndSortOperations(in []Operation) []Operation {
	if len(in) == 0 {
		return []Operation{}
	}
	out := make([]Operation, len(in))
	copy(out, in)
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

func cloneInstancePortStates(in []InstancePortState) []InstancePortState {
	if len(in) == 0 {
		return []InstancePortState{}
	}
	out := make([]InstancePortState, len(in))
	for i := range in {
		out[i] = InstancePortState{
			PortInfo: PortInfo{
				FromPort:        in[i].FromPort,
				ToPort:          in[i].ToPort,
				Protocol:        in[i].Protocol,
				Cidrs:           append([]string(nil), in[i].Cidrs...),
				Ipv6Cidrs:       append([]string(nil), in[i].Ipv6Cidrs...),
				CidrListAliases: append([]string(nil), in[i].CidrListAliases...),
			},
			State: in[i].State,
		}
	}
	return out
}

func cloneHostKeyAttributes(in []HostKeyAttributes) []HostKeyAttributes {
	if len(in) == 0 {
		return []HostKeyAttributes{}
	}
	out := make([]HostKeyAttributes, len(in))
	copy(out, in)
	return out
}

func newOperation(seq uint64, resourceName, resourceType, operationType, status, details, availabilityZone, region string, ts time.Time) Operation {
	if region == "" {
		region = DefaultRegion
	}
	if availabilityZone == "" {
		availabilityZone = region + "a"
	}
	return Operation{
		ID:               fmt.Sprintf("op-%d", seq),
		ResourceName:     resourceName,
		ResourceType:     resourceType,
		OperationType:    operationType,
		Status:           status,
		Details:          details,
		AvailabilityZone: availabilityZone,
		Region:           region,
		CreatedAt:        ts,
		StatusChangedAt:  ts,
		IsTerminal:       true,
	}
}

func (s *Service) appendOperationsLocked(ops []Operation) {
	if len(ops) == 0 {
		return
	}
	for _, op := range ops {
		s.operations = append(s.operations, op)
		s.operationByID[op.ID] = op
	}
}

func regionFromAvailabilityZone(availabilityZone string) string {
	availabilityZone = strings.TrimSpace(strings.ToLower(availabilityZone))
	if availabilityZone == "" {
		return DefaultRegion
	}
	last := availabilityZone[len(availabilityZone)-1]
	if last >= 'a' && last <= 'z' {
		return availabilityZone[:len(availabilityZone)-1]
	}
	return availabilityZone
}

func instanceARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:Instance/%s", region, DefaultAccountID, name)
}

func instanceSnapshotARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:InstanceSnapshot/%s", region, DefaultAccountID, name)
}

func staticIPARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:StaticIp/%s", region, DefaultAccountID, name)
}

func diskARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:Disk/%s", region, DefaultAccountID, name)
}

func diskSnapshotARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:DiskSnapshot/%s", region, DefaultAccountID, name)
}

func exportSnapshotRecordARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:ExportSnapshotRecord/%s", region, DefaultAccountID, name)
}

func alarmARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:Alarm/%s", region, DefaultAccountID, name)
}

func loadBalancerARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:LoadBalancer/%s", region, DefaultAccountID, name)
}

func loadBalancerTLSCertificateARN(region, loadBalancerName, certificateName string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:LoadBalancerTlsCertificate/%s/%s", region, DefaultAccountID, loadBalancerName, certificateName)
}

func distributionARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:Distribution/%s", region, DefaultAccountID, name)
}

func domainARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:Domain/%s", region, DefaultAccountID, name)
}

func certificateARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:Certificate/%s", region, DefaultAccountID, name)
}

func keyPairARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:KeyPair/%s", region, DefaultAccountID, name)
}

func bucketARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:Bucket/%s", region, DefaultAccountID, name)
}

func contactMethodARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:ContactMethod/%s", region, DefaultAccountID, name)
}

func containerServiceARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:ContainerService/%s", region, DefaultAccountID, name)
}

func relationalDatabaseARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:RelationalDatabase/%s", region, DefaultAccountID, name)
}

func relationalDatabaseSnapshotARN(region, name string) string {
	if region == "" {
		region = DefaultRegion
	}
	return fmt.Sprintf("arn:aws:lightsail:%s:%s:RelationalDatabaseSnapshot/%s", region, DefaultAccountID, name)
}

func encodeKeyMaterial(material string) string {
	return base64.StdEncoding.EncodeToString([]byte(material))
}

func fingerprintFromSeed(seed uint64) string {
	hex := fmt.Sprintf("%040x", seed*104729+13)
	parts := make([]string, 0, len(hex)/2)
	for i := 0; i+2 <= len(hex); i += 2 {
		parts = append(parts, hex[i:i+2])
	}
	return strings.Join(parts, ":")
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value.UTC()
		}
	}
	return time.Time{}
}

func defaultBlueprints() []Blueprint {
	return []Blueprint{
		{
			AppCategory: "",
			BlueprintID: "amazon_linux_2",
			Description: "Amazon Linux 2",
			Group:       "linux",
			IsActive:    true,
			LicenseURL:  "https://aws.amazon.com/amazon-linux-2/",
			MinPower:    0,
			Name:        "Amazon Linux 2",
			Platform:    "Linux/Unix",
			ProductURL:  "https://aws.amazon.com/amazon-linux-2/",
			Type:        "os",
			Version:     "2",
			VersionCode: "1",
		},
		{
			AppCategory: "",
			BlueprintID: "ubuntu_22_04",
			Description: "Ubuntu 22.04 LTS",
			Group:       "ubuntu",
			IsActive:    true,
			LicenseURL:  "https://ubuntu.com/legal",
			MinPower:    0,
			Name:        "Ubuntu 22.04 LTS",
			Platform:    "Linux/Unix",
			ProductURL:  "https://ubuntu.com/",
			Type:        "os",
			Version:     "22.04",
			VersionCode: "1",
		},
		{
			AppCategory: "LfR",
			BlueprintID: "lfr-jupyter-1",
			Description: "Lightsail for Research Jupyter image",
			Group:       "research",
			IsActive:    true,
			LicenseURL:  "https://aws.amazon.com/lightsail/",
			MinPower:    1000,
			Name:        "Jupyter Notebook",
			Platform:    "Linux/Unix",
			ProductURL:  "https://aws.amazon.com/lightsail/research/",
			Type:        "app",
			Version:     "1.0",
			VersionCode: "1",
		},
		{
			AppCategory: "",
			BlueprintID: "debian_10",
			Description: "Debian 10",
			Group:       "debian",
			IsActive:    false,
			LicenseURL:  "https://www.debian.org/legal/licenses/",
			MinPower:    0,
			Name:        "Debian 10",
			Platform:    "Linux/Unix",
			ProductURL:  "https://www.debian.org/",
			Type:        "os",
			Version:     "10",
			VersionCode: "1",
		},
	}
}

func defaultBundles() []Bundle {
	return []Bundle{
		{
			AppCategory:            "",
			BundleID:               "micro_2_0",
			CPUCount:               1,
			DiskSizeInGb:           40,
			InstanceType:           "micro",
			IsActive:               true,
			Name:                   "Micro",
			Power:                  500,
			Price:                  5.0,
			PublicIpv4AddressCount: 1,
			RAMSizeInGb:            1,
			SupportedAppCategories: []string{},
			SupportedPlatforms:     []string{"LINUX_UNIX"},
			TransferPerMonthInGb:   1024,
		},
		{
			AppCategory:            "",
			BundleID:               "small_2_0",
			CPUCount:               1,
			DiskSizeInGb:           80,
			InstanceType:           "small",
			IsActive:               true,
			Name:                   "Small",
			Power:                  1000,
			Price:                  10.0,
			PublicIpv4AddressCount: 1,
			RAMSizeInGb:            2,
			SupportedAppCategories: []string{},
			SupportedPlatforms:     []string{"LINUX_UNIX"},
			TransferPerMonthInGb:   2048,
		},
		{
			AppCategory:            "LfR",
			BundleID:               "lfr-standard-1",
			CPUCount:               4,
			DiskSizeInGb:           160,
			InstanceType:           "lfr.standard",
			IsActive:               true,
			Name:                   "Research Standard",
			Power:                  2000,
			Price:                  40.0,
			PublicIpv4AddressCount: 1,
			RAMSizeInGb:            16,
			SupportedAppCategories: []string{"LfR"},
			SupportedPlatforms:     []string{"LINUX_UNIX"},
			TransferPerMonthInGb:   4096,
		},
		{
			AppCategory:            "",
			BundleID:               "legacy_2_0",
			CPUCount:               1,
			DiskSizeInGb:           20,
			InstanceType:           "legacy.micro",
			IsActive:               false,
			Name:                   "Legacy",
			Power:                  250,
			Price:                  3.5,
			PublicIpv4AddressCount: 1,
			RAMSizeInGb:            0.5,
			SupportedAppCategories: []string{},
			SupportedPlatforms:     []string{"LINUX_UNIX"},
			TransferPerMonthInGb:   512,
		},
	}
}

func defaultLoadBalancerTLSPolicies() []LoadBalancerTLSPolicy {
	return []LoadBalancerTLSPolicy{
		{
			Name:        "TLS-1-2-2018-06",
			Description: "TLS policy with modern ciphers",
			IsDefault:   true,
			Ciphers: []string{
				"ECDHE-ECDSA-AES128-GCM-SHA256",
				"ECDHE-RSA-AES128-GCM-SHA256",
				"ECDHE-ECDSA-AES256-GCM-SHA384",
				"ECDHE-RSA-AES256-GCM-SHA384",
			},
			Protocols: []string{"TLSv1.2"},
		},
		{
			Name:        "TLS-1-2-Ext-2018-06",
			Description: "Extended TLS policy",
			IsDefault:   false,
			Ciphers: []string{
				"ECDHE-ECDSA-AES128-GCM-SHA256",
				"ECDHE-RSA-AES128-GCM-SHA256",
			},
			Protocols: []string{"TLSv1.2"},
		},
	}
}

func defaultDistributionBundles() []DistributionBundle {
	return []DistributionBundle{
		{
			BundleID:             "small_1_0",
			IsActive:             true,
			Name:                 "Small",
			Price:                2.5,
			TransferPerMonthInGb: 50,
		},
		{
			BundleID:             "medium_1_0",
			IsActive:             true,
			Name:                 "Medium",
			Price:                10.0,
			TransferPerMonthInGb: 100,
		},
		{
			BundleID:             "large_1_0",
			IsActive:             true,
			Name:                 "Large",
			Price:                20.0,
			TransferPerMonthInGb: 150,
		},
	}
}

func defaultBucketBundles() []BucketBundle {
	return []BucketBundle{
		{
			BundleID:             "small_1_0",
			IsActive:             true,
			Name:                 "Small",
			Price:                1.0,
			StoragePerMonthInGb:  40,
			TransferPerMonthInGb: 100,
		},
		{
			BundleID:             "medium_1_0",
			IsActive:             true,
			Name:                 "Medium",
			Price:                2.0,
			StoragePerMonthInGb:  100,
			TransferPerMonthInGb: 250,
		},
		{
			BundleID:             "large_1_0",
			IsActive:             true,
			Name:                 "Large",
			Price:                5.0,
			StoragePerMonthInGb:  250,
			TransferPerMonthInGb: 500,
		},
		{
			BundleID:             "legacy_1_0",
			IsActive:             false,
			Name:                 "Legacy",
			Price:                0.5,
			StoragePerMonthInGb:  20,
			TransferPerMonthInGb: 50,
		},
	}
}

func defaultRelationalDatabaseBlueprints() []RelationalDatabaseBlueprint {
	return []RelationalDatabaseBlueprint{
		{
			BlueprintID:              "mysql_8_0",
			Engine:                   "mysql",
			EngineDescription:        "MySQL",
			EngineVersion:            "8.0",
			EngineVersionDescription: "MySQL 8.0",
			IsEngineDefault:          true,
		},
		{
			BlueprintID:              "mysql_5_7",
			Engine:                   "mysql",
			EngineDescription:        "MySQL",
			EngineVersion:            "5.7",
			EngineVersionDescription: "MySQL 5.7",
			IsEngineDefault:          false,
		},
		{
			BlueprintID:              "postgres_13",
			Engine:                   "postgres",
			EngineDescription:        "PostgreSQL",
			EngineVersion:            "13.7",
			EngineVersionDescription: "PostgreSQL 13",
			IsEngineDefault:          true,
		},
	}
}

func defaultRelationalDatabaseBundles() []RelationalDatabaseBundle {
	return []RelationalDatabaseBundle{
		{
			BundleID:             "micro_1_0",
			CPUCount:             1,
			DiskSizeInGb:         40,
			IsActive:             true,
			IsEncrypted:          true,
			Name:                 "Micro",
			Price:                15,
			RAMSizeInGb:          1,
			TransferPerMonthInGb: 100,
		},
		{
			BundleID:             "small_1_0",
			CPUCount:             1,
			DiskSizeInGb:         80,
			IsActive:             true,
			IsEncrypted:          true,
			Name:                 "Small",
			Price:                30,
			RAMSizeInGb:          2,
			TransferPerMonthInGb: 200,
		},
		{
			BundleID:             "medium_1_0",
			CPUCount:             2,
			DiskSizeInGb:         120,
			IsActive:             true,
			IsEncrypted:          true,
			Name:                 "Medium",
			Price:                60,
			RAMSizeInGb:          4,
			TransferPerMonthInGb: 300,
		},
		{
			BundleID:             "legacy_1_0",
			CPUCount:             1,
			DiskSizeInGb:         20,
			IsActive:             false,
			IsEncrypted:          false,
			Name:                 "Legacy",
			Price:                10,
			RAMSizeInGb:          1,
			TransferPerMonthInGb: 50,
		},
	}
}

func defaultRelationalDatabaseParameters(engine string) map[string]RelationalDatabaseParameter {
	engine = strings.ToLower(strings.TrimSpace(engine))
	out := map[string]RelationalDatabaseParameter{
		"autocommit": {
			AllowedValues:  "0,1",
			ApplyMethod:    "immediate",
			ApplyType:      "dynamic",
			DataType:       "boolean",
			Description:    "Autocommit mode",
			IsModifiable:   true,
			ParameterName:  "autocommit",
			ParameterValue: "1",
		},
		"max_connections": {
			AllowedValues:  "1-10000",
			ApplyMethod:    "pending-reboot",
			ApplyType:      "static",
			DataType:       "integer",
			Description:    "Maximum permitted number of simultaneous client connections",
			IsModifiable:   true,
			ParameterName:  "max_connections",
			ParameterValue: "150",
		},
	}
	if engine == "postgres" {
		out["log_statement"] = RelationalDatabaseParameter{
			AllowedValues:  "none,ddl,mod,all",
			ApplyMethod:    "immediate",
			ApplyType:      "dynamic",
			DataType:       "string",
			Description:    "Sets the type of statements logged",
			IsModifiable:   true,
			ParameterName:  "log_statement",
			ParameterValue: "none",
		}
	} else {
		out["slow_query_log"] = RelationalDatabaseParameter{
			AllowedValues:  "0,1",
			ApplyMethod:    "pending-reboot",
			ApplyType:      "static",
			DataType:       "boolean",
			Description:    "Enable slow query log",
			IsModifiable:   true,
			ParameterName:  "slow_query_log",
			ParameterValue: "0",
		}
	}
	return out
}

func defaultContainerServicePowers() []ContainerServicePower {
	return []ContainerServicePower{
		{Name: "nano", PowerID: "nano-1", CPUCount: 0.25, RAMSizeInGb: 0.5, Price: 7, IsActive: true},
		{Name: "micro", PowerID: "micro-1", CPUCount: 0.5, RAMSizeInGb: 1, Price: 10, IsActive: true},
		{Name: "small", PowerID: "small-1", CPUCount: 1, RAMSizeInGb: 2, Price: 20, IsActive: true},
		{Name: "medium", PowerID: "medium-1", CPUCount: 2, RAMSizeInGb: 4, Price: 40, IsActive: true},
		{Name: "large", PowerID: "large-1", CPUCount: 4, RAMSizeInGb: 8, Price: 80, IsActive: true},
		{Name: "xlarge", PowerID: "xlarge-1", CPUCount: 8, RAMSizeInGb: 16, Price: 160, IsActive: true},
	}
}

func (s *Service) ensureDefaultKeyPairLocked() *KeyPair {
	if keyPair := s.keyPairs[DefaultKeyPair]; keyPair != nil {
		return keyPair
	}
	seq := atomic.AddUint64(&s.seq, 1)
	now := time.Now().UTC()
	keyPair := &KeyPair{
		Name:             DefaultKeyPair,
		ARN:              keyPairARN(DefaultRegion, DefaultKeyPair),
		Fingerprint:      fingerprintFromSeed(seq),
		AvailabilityZone: DefaultRegion + "a",
		Region:           DefaultRegion,
		CreatedAt:        now,
		PublicKeyBase64:  encodeKeyMaterial("ssh-rsa STACKYARD-DEFAULT-PUBLIC"),
		PrivateKeyBase64: encodeKeyMaterial("-----BEGIN PRIVATE KEY-----\nSTACKYARD-DEFAULT-PRIVATE\n-----END PRIVATE KEY-----"),
		IsDefault:        true,
		Tags:             map[string]string{},
	}
	s.keyPairs[DefaultKeyPair] = keyPair
	return keyPair
}

func normalizePortInfo(in PortInfo) PortInfo {
	out := PortInfo{
		FromPort:        in.FromPort,
		ToPort:          in.ToPort,
		Protocol:        strings.ToLower(strings.TrimSpace(in.Protocol)),
		Cidrs:           dedupeStrings(in.Cidrs),
		Ipv6Cidrs:       dedupeStrings(in.Ipv6Cidrs),
		CidrListAliases: dedupeStrings(in.CidrListAliases),
	}
	return out
}

func samePortRule(a, b PortInfo) bool {
	return strings.EqualFold(strings.TrimSpace(a.Protocol), strings.TrimSpace(b.Protocol)) && a.FromPort == b.FromPort && a.ToPort == b.ToPort
}

func dedupeStrings(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func containsString(in []string, wanted string) bool {
	wanted = strings.TrimSpace(wanted)
	for _, item := range in {
		if strings.TrimSpace(item) == wanted {
			return true
		}
	}
	return false
}

func removeString(in []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || len(in) == 0 {
		return append([]string(nil), in...)
	}
	out := make([]string, 0, len(in))
	for _, item := range in {
		if strings.TrimSpace(item) == value {
			continue
		}
		out = append(out, item)
	}
	return out
}

func inferInstanceIPAddressType(instance *Instance) string {
	if instance == nil {
		return "dualstack"
	}
	hasIPv4 := strings.TrimSpace(instance.PublicIPAddress) != ""
	hasIPv6 := len(instance.IPv6Addresses) > 0
	switch {
	case hasIPv4 && hasIPv6:
		return "dualstack"
	case hasIPv6:
		return "ipv6"
	default:
		return "ipv4"
	}
}

func defaultInstanceMetadataOptions() InstanceMetadataOptions {
	return InstanceMetadataOptions{
		HttpEndpoint:            "enabled",
		HttpProtocolIpv6:        "disabled",
		HttpPutResponseHopLimit: 1,
		HttpTokens:              "optional",
		State:                   "applied",
	}
}

func defaultInstancePortStates() []InstancePortState {
	return []InstancePortState{
		{
			PortInfo: PortInfo{
				FromPort: 22,
				ToPort:   22,
				Protocol: "tcp",
				Cidrs:    []string{"0.0.0.0/0"},
			},
			State: "open",
		},
	}
}

func defaultInstanceHostKeys(instanceName string, now time.Time, seed uint64) []HostKeyAttributes {
	instanceName = strings.TrimSpace(instanceName)
	if instanceName == "" {
		instanceName = "instance"
	}
	return []HostKeyAttributes{
		{
			Algorithm:         "ssh-ed25519",
			FingerprintSHA1:   fmt.Sprintf("SHA1:%x", seed),
			FingerprintSHA256: fmt.Sprintf("SHA256:%x", seed*7919+17),
			PublicKey:         fmt.Sprintf("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI%s stackyard@%s", strings.ToUpper(fmt.Sprintf("%x", seed)), instanceName),
			WitnessedAt:       now,
		},
	}
}
