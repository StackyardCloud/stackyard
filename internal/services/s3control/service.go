package s3control

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	ErrAccessPointNotFound                = errors.New("access point not found")
	ErrAccessPointAlreadyExists           = errors.New("access point already exists")
	ErrAccessPointPolicyNotFound          = errors.New("access point policy not found")
	ErrAccessPointPublicAccessBlockAbsent = errors.New("access point public access block not found")
	ErrAccessPointScopeNotFound           = errors.New("access point scope not found")
	ErrAccountPublicAccessBlockAbsent     = errors.New("account public access block not found")
	ErrMultiRegionAccessPointNotFound     = errors.New("multi-region access point not found")
	ErrMultiRegionAccessPointExists       = errors.New("multi-region access point already exists")
	ErrMultiRegionAccessPointPolicyAbsent = errors.New("multi-region access point policy not found")
	ErrMultiRegionAccessPointOpNotFound   = errors.New("multi-region access point operation not found")
	ErrJobNotFound                        = errors.New("job not found")
	ErrJobTaggingNotFound                 = errors.New("job tagging not found")
	ErrStorageLensConfigNotFound          = errors.New("storage lens configuration not found")
	ErrStorageLensTaggingNotFound         = errors.New("storage lens configuration tagging not found")
	ErrStorageLensGroupNotFound           = errors.New("storage lens group not found")
	ErrAccessGrantsInstanceNotFound       = errors.New("access grants instance not found")
	ErrAccessGrantsInstanceAlreadyExists  = errors.New("access grants instance already exists")
	ErrAccessGrantsResourcePolicyNotFound = errors.New("access grants instance resource policy not found")
	ErrAccessGrantsLocationNotFound       = errors.New("access grants location not found")
	ErrAccessGrantNotFound                = errors.New("access grant not found")
	ErrOutpostsBucketNotFound             = errors.New("outposts bucket not found")
	ErrOutpostsBucketAlreadyExists        = errors.New("outposts bucket already exists")
	ErrOutpostsBucketTaggingNotFound      = errors.New("outposts bucket tagging not found")
	ErrOutpostsBucketLifecycleNotFound    = errors.New("outposts bucket lifecycle configuration not found")
	ErrOutpostsBucketPolicyNotFound       = errors.New("outposts bucket policy not found")
	ErrOutpostsBucketReplicationNotFound  = errors.New("outposts bucket replication not found")
)

type AccessPointPublicAccessBlockConfiguration struct {
	BlockPublicAcls       bool
	IgnorePublicAcls      bool
	BlockPublicPolicy     bool
	RestrictPublicBuckets bool
}

type AccountPublicAccessBlockConfiguration struct {
	BlockPublicAcls       bool
	IgnorePublicAcls      bool
	BlockPublicPolicy     bool
	RestrictPublicBuckets bool
}

type MultiRegionAccessPointPublicAccessBlock struct {
	BlockPublicAcls       bool
	BlockPublicPolicy     bool
	IgnorePublicAcls      bool
	RestrictPublicBuckets bool
}

type MultiRegionAccessPointRegion struct {
	Bucket        string
	BucketAccount string
	Region        string
}

type MultiRegionAccessPointRoute struct {
	Bucket                string
	Region                string
	TrafficDialPercentage int
}

type MultiRegionAccessPoint struct {
	Name              string
	Alias             string
	CreatedAt         time.Time
	Status            string
	PublicAccessBlock *MultiRegionAccessPointPublicAccessBlock
	Regions           []MultiRegionAccessPointRegion
	Policy            string
	Routes            []MultiRegionAccessPointRoute
}

type MultiRegionAccessPointRequestParameters struct {
	CreateRequest *MultiRegionAccessPoint
	DeleteName    string
	PutPolicyName string
	PutPolicyBody string
}

type MultiRegionAccessPointRegionStatus struct {
	Name          string
	RequestStatus string
}

type MultiRegionAccessPointOperation struct {
	TokenARN       string
	CreationTime   time.Time
	Operation      string
	RequestStatus  string
	RequestParams  MultiRegionAccessPointRequestParameters
	RegionStatuses []MultiRegionAccessPointRegionStatus
}

type Job struct {
	ID                   string
	Status               string
	Priority             int
	RoleArn              string
	ClientRequestToken   string
	ConfirmationRequired bool
	CreationTime         time.Time
	ManifestXML          string
	OperationXML         string
	ReportXML            string
	Tags                 map[string]string
}

type StorageLensConfiguration struct {
	ID        string
	ConfigXML string
	Tags      map[string]string
	CreatedAt time.Time
}

type StorageLensGroup struct {
	Name      string
	GroupXML  string
	CreatedAt time.Time
}

type AccessGrantsInstance struct {
	ID                           string
	Arn                          string
	IdentityCenterArn            string
	IdentityCenterInstanceArn    string
	IdentityCenterApplicationArn string
	CreatedAt                    time.Time
}

type AccessGrantsResourcePolicy struct {
	Policy       string
	Organization string
	CreatedAt    time.Time
}

type AccessGrantsLocation struct {
	ID            string
	Arn           string
	LocationScope string
	IAMRoleArn    string
	Tags          map[string]string
	CreatedAt     time.Time
}

type AccessGrantGrantee struct {
	Type       string
	Identifier string
}

type AccessGrant struct {
	ID                        string
	Arn                       string
	AccessGrantsLocationID    string
	AccessGrantsLocationScope string
	Permission                string
	ApplicationArn            string
	S3PrefixType              string
	S3SubPrefix               string
	Grantee                   AccessGrantGrantee
	Tags                      map[string]string
	CreatedAt                 time.Time
}

type OutpostsBucket struct {
	Name             string
	OutpostID        string
	Arn              string
	CreatedAt        time.Time
	Tags             map[string]string
	LifecycleXML     string
	Policy           string
	ReplicationXML   string
	VersioningStatus string
}

type AccessPointTag struct {
	Key   string
	Value string
}

type AccessPointScope struct {
	Permissions []string
	Prefixes    []string
}

type AccessPoint struct {
	Name              string
	Bucket            string
	BucketAccount     string
	VpcID             string
	CreatedAt         time.Time
	Alias             string
	NetworkOrigin     string
	AccessPointArn    string
	DataSourceID      string
	DataSourceType    string
	PublicAccessBlock *AccessPointPublicAccessBlockConfiguration
	Policy            string
	Scope             *AccessPointScope
	Tags              []AccessPointTag
}

type AccountState struct {
	AccountID               string
	AccessPoints            map[string]*AccessPoint
	PublicAccessBlock       *AccountPublicAccessBlockConfiguration
	MultiRegionAccessPoints map[string]*MultiRegionAccessPoint
	MultiRegionOperations   map[string]*MultiRegionAccessPointOperation
	Jobs                    map[string]*Job
	StorageLensConfigs      map[string]*StorageLensConfiguration
	StorageLensGroups       map[string]*StorageLensGroup
	AccessGrantsInstance    *AccessGrantsInstance
	AccessGrantsPolicy      *AccessGrantsResourcePolicy
	AccessGrantsLocations   map[string]*AccessGrantsLocation
	AccessGrantsGrants      map[string]*AccessGrant
	OutpostsBuckets         map[string]*OutpostsBucket
	ResourceTags            map[string]map[string]string
}

type Service struct {
	mu       sync.RWMutex
	accounts map[string]*AccountState
}

func NewService() *Service {
	return &Service{
		accounts: make(map[string]*AccountState),
	}
}

func (s *Service) GetOrCreateAccount(accountID string) *AccountState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if account, ok := s.accounts[accountID]; ok {
		return account
	}
	account := &AccountState{
		AccountID:               accountID,
		AccessPoints:            make(map[string]*AccessPoint),
		MultiRegionAccessPoints: make(map[string]*MultiRegionAccessPoint),
		MultiRegionOperations:   make(map[string]*MultiRegionAccessPointOperation),
		Jobs:                    make(map[string]*Job),
		StorageLensConfigs:      make(map[string]*StorageLensConfiguration),
		StorageLensGroups:       make(map[string]*StorageLensGroup),
		AccessGrantsLocations:   make(map[string]*AccessGrantsLocation),
		AccessGrantsGrants:      make(map[string]*AccessGrant),
		OutpostsBuckets:         make(map[string]*OutpostsBucket),
		ResourceTags:            make(map[string]map[string]string),
	}
	s.accounts[accountID] = account
	return account
}

func cloneTagMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func (s *Service) TagResource(accountID, resourceArn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		account = &AccountState{
			AccountID:               accountID,
			AccessPoints:            make(map[string]*AccessPoint),
			MultiRegionAccessPoints: make(map[string]*MultiRegionAccessPoint),
			MultiRegionOperations:   make(map[string]*MultiRegionAccessPointOperation),
			Jobs:                    make(map[string]*Job),
			StorageLensConfigs:      make(map[string]*StorageLensConfiguration),
			StorageLensGroups:       make(map[string]*StorageLensGroup),
			AccessGrantsLocations:   make(map[string]*AccessGrantsLocation),
			AccessGrantsGrants:      make(map[string]*AccessGrant),
			OutpostsBuckets:         make(map[string]*OutpostsBucket),
			ResourceTags:            make(map[string]map[string]string),
		}
		s.accounts[accountID] = account
	}
	existing := account.ResourceTags[resourceArn]
	if existing == nil {
		existing = make(map[string]string)
		account.ResourceTags[resourceArn] = existing
	}
	for key, value := range tags {
		if key == "" {
			continue
		}
		existing[key] = value
	}
}

func (s *Service) UntagResource(accountID, resourceArn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return
	}
	existing := account.ResourceTags[resourceArn]
	if existing == nil {
		return
	}
	for _, key := range keys {
		delete(existing, key)
	}
	if len(existing) == 0 {
		delete(account.ResourceTags, resourceArn)
	}
}

func (s *Service) ListResourceTags(accountID, resourceArn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil
	}
	return cloneTagMap(account.ResourceTags[resourceArn])
}

func (s *Service) CreateAccessPoint(accountID string, ap *AccessPoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		account = &AccountState{
			AccountID:               accountID,
			AccessPoints:            make(map[string]*AccessPoint),
			MultiRegionAccessPoints: make(map[string]*MultiRegionAccessPoint),
			MultiRegionOperations:   make(map[string]*MultiRegionAccessPointOperation),
			Jobs:                    make(map[string]*Job),
			StorageLensConfigs:      make(map[string]*StorageLensConfiguration),
			StorageLensGroups:       make(map[string]*StorageLensGroup),
			AccessGrantsLocations:   make(map[string]*AccessGrantsLocation),
			AccessGrantsGrants:      make(map[string]*AccessGrant),
			OutpostsBuckets:         make(map[string]*OutpostsBucket),
		}
		s.accounts[accountID] = account
	}
	if _, exists := account.AccessPoints[ap.Name]; exists {
		return ErrAccessPointAlreadyExists
	}
	account.AccessPoints[ap.Name] = ap
	return nil
}

func (s *Service) GetAccessPoint(accountID, name string) (*AccessPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil, ErrAccessPointNotFound
	}
	ap, ok := account.AccessPoints[name]
	if !ok {
		return nil, ErrAccessPointNotFound
	}
	return ap, nil
}

func (s *Service) DeleteAccessPoint(accountID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrAccessPointNotFound
	}
	if _, ok := account.AccessPoints[name]; !ok {
		return ErrAccessPointNotFound
	}
	delete(account.AccessPoints, name)
	return nil
}

func (s *Service) ListAccessPoints(accountID string) []*AccessPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil
	}
	list := make([]*AccessPoint, 0, len(account.AccessPoints))
	for _, ap := range account.AccessPoints {
		list = append(list, ap)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}

func (s *Service) SetAccessPointPolicy(accountID, name, policy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrAccessPointNotFound
	}
	ap, ok := account.AccessPoints[name]
	if !ok {
		return ErrAccessPointNotFound
	}
	ap.Policy = policy
	return nil
}

func (s *Service) GetAccessPointPolicy(accountID, name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return "", ErrAccessPointNotFound
	}
	ap, ok := account.AccessPoints[name]
	if !ok {
		return "", ErrAccessPointNotFound
	}
	if ap.Policy == "" {
		return "", ErrAccessPointPolicyNotFound
	}
	return ap.Policy, nil
}

func (s *Service) DeleteAccessPointPolicy(accountID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrAccessPointNotFound
	}
	ap, ok := account.AccessPoints[name]
	if !ok {
		return ErrAccessPointNotFound
	}
	if ap.Policy == "" {
		return ErrAccessPointPolicyNotFound
	}
	ap.Policy = ""
	return nil
}

func (s *Service) SetAccessPointScope(accountID, name string, scope *AccessPointScope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrAccessPointNotFound
	}
	ap, ok := account.AccessPoints[name]
	if !ok {
		return ErrAccessPointNotFound
	}
	ap.Scope = scope
	return nil
}

func (s *Service) GetAccessPointScope(accountID, name string) (*AccessPointScope, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil, ErrAccessPointNotFound
	}
	ap, ok := account.AccessPoints[name]
	if !ok {
		return nil, ErrAccessPointNotFound
	}
	if ap.Scope == nil {
		return nil, ErrAccessPointScopeNotFound
	}
	scope := *ap.Scope
	scope.Permissions = append([]string(nil), ap.Scope.Permissions...)
	scope.Prefixes = append([]string(nil), ap.Scope.Prefixes...)
	return &scope, nil
}

func (s *Service) DeleteAccessPointScope(accountID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrAccessPointNotFound
	}
	ap, ok := account.AccessPoints[name]
	if !ok {
		return ErrAccessPointNotFound
	}
	if ap.Scope == nil {
		return ErrAccessPointScopeNotFound
	}
	ap.Scope = nil
	return nil
}

func (s *Service) SetAccessPointPublicAccessBlock(accountID, name string, cfg *AccessPointPublicAccessBlockConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrAccessPointNotFound
	}
	ap, ok := account.AccessPoints[name]
	if !ok {
		return ErrAccessPointNotFound
	}
	ap.PublicAccessBlock = cfg
	return nil
}

func (s *Service) GetAccessPointPublicAccessBlock(accountID, name string) (*AccessPointPublicAccessBlockConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil, ErrAccessPointNotFound
	}
	ap, ok := account.AccessPoints[name]
	if !ok {
		return nil, ErrAccessPointNotFound
	}
	if ap.PublicAccessBlock == nil {
		return nil, ErrAccessPointPublicAccessBlockAbsent
	}
	cfg := *ap.PublicAccessBlock
	return &cfg, nil
}

func (s *Service) DeleteAccessPointPublicAccessBlock(accountID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrAccessPointNotFound
	}
	ap, ok := account.AccessPoints[name]
	if !ok {
		return ErrAccessPointNotFound
	}
	if ap.PublicAccessBlock == nil {
		return ErrAccessPointPublicAccessBlockAbsent
	}
	ap.PublicAccessBlock = nil
	return nil
}

func (s *Service) SetAccountPublicAccessBlock(accountID string, cfg *AccountPublicAccessBlockConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		account = &AccountState{
			AccountID:               accountID,
			AccessPoints:            make(map[string]*AccessPoint),
			MultiRegionAccessPoints: make(map[string]*MultiRegionAccessPoint),
			MultiRegionOperations:   make(map[string]*MultiRegionAccessPointOperation),
			Jobs:                    make(map[string]*Job),
			StorageLensConfigs:      make(map[string]*StorageLensConfiguration),
			StorageLensGroups:       make(map[string]*StorageLensGroup),
			AccessGrantsLocations:   make(map[string]*AccessGrantsLocation),
			AccessGrantsGrants:      make(map[string]*AccessGrant),
			OutpostsBuckets:         make(map[string]*OutpostsBucket),
		}
		s.accounts[accountID] = account
	}
	account.PublicAccessBlock = cfg
	return nil
}

func (s *Service) GetAccountPublicAccessBlock(accountID string) (*AccountPublicAccessBlockConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil, ErrAccountPublicAccessBlockAbsent
	}
	if account.PublicAccessBlock == nil {
		return nil, ErrAccountPublicAccessBlockAbsent
	}
	cfg := *account.PublicAccessBlock
	return &cfg, nil
}

func (s *Service) DeleteAccountPublicAccessBlock(accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrAccountPublicAccessBlockAbsent
	}
	if account.PublicAccessBlock == nil {
		return ErrAccountPublicAccessBlockAbsent
	}
	account.PublicAccessBlock = nil
	return nil
}

func (s *Service) CreateMultiRegionAccessPoint(accountID string, mrap *MultiRegionAccessPoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		account = &AccountState{
			AccountID:               accountID,
			AccessPoints:            make(map[string]*AccessPoint),
			MultiRegionAccessPoints: make(map[string]*MultiRegionAccessPoint),
			MultiRegionOperations:   make(map[string]*MultiRegionAccessPointOperation),
			Jobs:                    make(map[string]*Job),
			StorageLensConfigs:      make(map[string]*StorageLensConfiguration),
			StorageLensGroups:       make(map[string]*StorageLensGroup),
			AccessGrantsLocations:   make(map[string]*AccessGrantsLocation),
			AccessGrantsGrants:      make(map[string]*AccessGrant),
			OutpostsBuckets:         make(map[string]*OutpostsBucket),
		}
		s.accounts[accountID] = account
	}
	if _, exists := account.MultiRegionAccessPoints[mrap.Name]; exists {
		return ErrMultiRegionAccessPointExists
	}
	account.MultiRegionAccessPoints[mrap.Name] = mrap
	return nil
}

func (s *Service) GetMultiRegionAccessPoint(accountID, name string) (*MultiRegionAccessPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil, ErrMultiRegionAccessPointNotFound
	}
	mrap, ok := account.MultiRegionAccessPoints[name]
	if !ok {
		return nil, ErrMultiRegionAccessPointNotFound
	}
	return mrap, nil
}

func (s *Service) ListMultiRegionAccessPoints(accountID string) []*MultiRegionAccessPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil
	}
	list := make([]*MultiRegionAccessPoint, 0, len(account.MultiRegionAccessPoints))
	for _, mrap := range account.MultiRegionAccessPoints {
		list = append(list, mrap)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}

func (s *Service) DeleteMultiRegionAccessPoint(accountID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrMultiRegionAccessPointNotFound
	}
	if _, ok := account.MultiRegionAccessPoints[name]; !ok {
		return ErrMultiRegionAccessPointNotFound
	}
	delete(account.MultiRegionAccessPoints, name)
	return nil
}

func (s *Service) SetMultiRegionAccessPointPolicy(accountID, name, policy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrMultiRegionAccessPointNotFound
	}
	mrap, ok := account.MultiRegionAccessPoints[name]
	if !ok {
		return ErrMultiRegionAccessPointNotFound
	}
	mrap.Policy = policy
	return nil
}

func (s *Service) GetMultiRegionAccessPointPolicy(accountID, name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return "", ErrMultiRegionAccessPointNotFound
	}
	mrap, ok := account.MultiRegionAccessPoints[name]
	if !ok {
		return "", ErrMultiRegionAccessPointNotFound
	}
	if mrap.Policy == "" {
		return "", ErrMultiRegionAccessPointPolicyAbsent
	}
	return mrap.Policy, nil
}

func (s *Service) SetMultiRegionAccessPointRoutes(accountID, name string, routes []MultiRegionAccessPointRoute) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrMultiRegionAccessPointNotFound
	}
	mrap, ok := account.MultiRegionAccessPoints[name]
	if !ok {
		return ErrMultiRegionAccessPointNotFound
	}
	mrap.Routes = routes
	return nil
}

func (s *Service) CreateMultiRegionOperation(accountID string, op *MultiRegionAccessPointOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		account = &AccountState{
			AccountID:               accountID,
			AccessPoints:            make(map[string]*AccessPoint),
			MultiRegionAccessPoints: make(map[string]*MultiRegionAccessPoint),
			MultiRegionOperations:   make(map[string]*MultiRegionAccessPointOperation),
		}
		s.accounts[accountID] = account
	}
	account.MultiRegionOperations[op.TokenARN] = op
}

func (s *Service) GetMultiRegionOperation(accountID, token string) (*MultiRegionAccessPointOperation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil, ErrMultiRegionAccessPointOpNotFound
	}
	op, ok := account.MultiRegionOperations[token]
	if !ok {
		return nil, ErrMultiRegionAccessPointOpNotFound
	}
	return op, nil
}

func (s *Service) CreateJob(accountID string, job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		account = &AccountState{
			AccountID:               accountID,
			AccessPoints:            make(map[string]*AccessPoint),
			MultiRegionAccessPoints: make(map[string]*MultiRegionAccessPoint),
			MultiRegionOperations:   make(map[string]*MultiRegionAccessPointOperation),
			Jobs:                    make(map[string]*Job),
		}
		s.accounts[accountID] = account
	}
	account.Jobs[job.ID] = job
	return nil
}

func (s *Service) GetJob(accountID, jobID string) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil, ErrJobNotFound
	}
	job, ok := account.Jobs[jobID]
	if !ok {
		return nil, ErrJobNotFound
	}
	return job, nil
}

func (s *Service) ListJobs(accountID string) []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil
	}
	list := make([]*Job, 0, len(account.Jobs))
	for _, job := range account.Jobs {
		list = append(list, job)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	return list
}

func (s *Service) UpdateJobPriority(accountID, jobID string, priority int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrJobNotFound
	}
	job, ok := account.Jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}
	job.Priority = priority
	return nil
}

func (s *Service) UpdateJobStatus(accountID, jobID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrJobNotFound
	}
	job, ok := account.Jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}
	job.Status = status
	return nil
}

func (s *Service) PutJobTagging(accountID, jobID string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrJobNotFound
	}
	job, ok := account.Jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}
	if job.Tags == nil {
		job.Tags = make(map[string]string)
	}
	for k, v := range tags {
		job.Tags[k] = v
	}
	return nil
}

func (s *Service) GetJobTagging(accountID, jobID string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil, ErrJobNotFound
	}
	job, ok := account.Jobs[jobID]
	if !ok {
		return nil, ErrJobNotFound
	}
	if len(job.Tags) == 0 {
		return nil, ErrJobTaggingNotFound
	}
	out := make(map[string]string, len(job.Tags))
	for k, v := range job.Tags {
		out[k] = v
	}
	return out, nil
}

func (s *Service) DeleteJobTagging(accountID, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrJobNotFound
	}
	job, ok := account.Jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}
	if len(job.Tags) == 0 {
		return ErrJobTaggingNotFound
	}
	job.Tags = nil
	return nil
}

func (s *Service) PutStorageLensConfiguration(accountID, configID, xmlBody string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		account = &AccountState{
			AccountID:               accountID,
			AccessPoints:            make(map[string]*AccessPoint),
			MultiRegionAccessPoints: make(map[string]*MultiRegionAccessPoint),
			MultiRegionOperations:   make(map[string]*MultiRegionAccessPointOperation),
			Jobs:                    make(map[string]*Job),
			StorageLensConfigs:      make(map[string]*StorageLensConfiguration),
			StorageLensGroups:       make(map[string]*StorageLensGroup),
			AccessGrantsLocations:   make(map[string]*AccessGrantsLocation),
			AccessGrantsGrants:      make(map[string]*AccessGrant),
			OutpostsBuckets:         make(map[string]*OutpostsBucket),
		}
		s.accounts[accountID] = account
	}
	cfg, ok := account.StorageLensConfigs[configID]
	if !ok {
		cfg = &StorageLensConfiguration{
			ID:        configID,
			CreatedAt: time.Now().UTC(),
		}
		account.StorageLensConfigs[configID] = cfg
	}
	cfg.ConfigXML = xmlBody
	return nil
}

func (s *Service) GetStorageLensConfiguration(accountID, configID string) (*StorageLensConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil, ErrStorageLensConfigNotFound
	}
	cfg, ok := account.StorageLensConfigs[configID]
	if !ok {
		return nil, ErrStorageLensConfigNotFound
	}
	return cfg, nil
}

func (s *Service) DeleteStorageLensConfiguration(accountID, configID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrStorageLensConfigNotFound
	}
	if _, ok := account.StorageLensConfigs[configID]; !ok {
		return ErrStorageLensConfigNotFound
	}
	delete(account.StorageLensConfigs, configID)
	return nil
}

func (s *Service) ListStorageLensConfigurations(accountID string) []*StorageLensConfiguration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil
	}
	list := make([]*StorageLensConfiguration, 0, len(account.StorageLensConfigs))
	for _, cfg := range account.StorageLensConfigs {
		list = append(list, cfg)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	return list
}

func (s *Service) PutStorageLensTagging(accountID, configID string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrStorageLensConfigNotFound
	}
	cfg, ok := account.StorageLensConfigs[configID]
	if !ok {
		return ErrStorageLensConfigNotFound
	}
	if cfg.Tags == nil {
		cfg.Tags = make(map[string]string)
	}
	for k, v := range tags {
		cfg.Tags[k] = v
	}
	return nil
}

func (s *Service) GetStorageLensTagging(accountID, configID string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil, ErrStorageLensConfigNotFound
	}
	cfg, ok := account.StorageLensConfigs[configID]
	if !ok {
		return nil, ErrStorageLensConfigNotFound
	}
	if len(cfg.Tags) == 0 {
		return nil, ErrStorageLensTaggingNotFound
	}
	out := make(map[string]string, len(cfg.Tags))
	for k, v := range cfg.Tags {
		out[k] = v
	}
	return out, nil
}

func (s *Service) DeleteStorageLensTagging(accountID, configID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrStorageLensConfigNotFound
	}
	cfg, ok := account.StorageLensConfigs[configID]
	if !ok {
		return ErrStorageLensConfigNotFound
	}
	if len(cfg.Tags) == 0 {
		return ErrStorageLensTaggingNotFound
	}
	cfg.Tags = nil
	return nil
}

func (s *Service) CreateStorageLensGroup(accountID, name, xmlBody string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		account = &AccountState{
			AccountID:               accountID,
			AccessPoints:            make(map[string]*AccessPoint),
			MultiRegionAccessPoints: make(map[string]*MultiRegionAccessPoint),
			MultiRegionOperations:   make(map[string]*MultiRegionAccessPointOperation),
			Jobs:                    make(map[string]*Job),
			StorageLensConfigs:      make(map[string]*StorageLensConfiguration),
			StorageLensGroups:       make(map[string]*StorageLensGroup),
			AccessGrantsLocations:   make(map[string]*AccessGrantsLocation),
			AccessGrantsGrants:      make(map[string]*AccessGrant),
			OutpostsBuckets:         make(map[string]*OutpostsBucket),
		}
		s.accounts[accountID] = account
	}
	group, ok := account.StorageLensGroups[name]
	if !ok {
		group = &StorageLensGroup{
			Name:      name,
			CreatedAt: time.Now().UTC(),
		}
		account.StorageLensGroups[name] = group
	}
	group.GroupXML = xmlBody
	return nil
}

func (s *Service) GetStorageLensGroup(accountID, name string) (*StorageLensGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil, ErrStorageLensGroupNotFound
	}
	group, ok := account.StorageLensGroups[name]
	if !ok {
		return nil, ErrStorageLensGroupNotFound
	}
	return group, nil
}

func (s *Service) DeleteStorageLensGroup(accountID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrStorageLensGroupNotFound
	}
	if _, ok := account.StorageLensGroups[name]; !ok {
		return ErrStorageLensGroupNotFound
	}
	delete(account.StorageLensGroups, name)
	return nil
}

func (s *Service) ListStorageLensGroups(accountID string) []*StorageLensGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return nil
	}
	list := make([]*StorageLensGroup, 0, len(account.StorageLensGroups))
	for _, group := range account.StorageLensGroups {
		list = append(list, group)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}

func (s *Service) CreateAccessGrantsInstance(accountID string, instance *AccessGrantsInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		account = &AccountState{
			AccountID:               accountID,
			AccessPoints:            make(map[string]*AccessPoint),
			MultiRegionAccessPoints: make(map[string]*MultiRegionAccessPoint),
			MultiRegionOperations:   make(map[string]*MultiRegionAccessPointOperation),
			Jobs:                    make(map[string]*Job),
			StorageLensConfigs:      make(map[string]*StorageLensConfiguration),
			StorageLensGroups:       make(map[string]*StorageLensGroup),
			AccessGrantsLocations:   make(map[string]*AccessGrantsLocation),
			AccessGrantsGrants:      make(map[string]*AccessGrant),
			OutpostsBuckets:         make(map[string]*OutpostsBucket),
		}
		s.accounts[accountID] = account
	}
	if account.AccessGrantsInstance != nil {
		return ErrAccessGrantsInstanceAlreadyExists
	}
	account.AccessGrantsInstance = instance
	return nil
}

func (s *Service) GetAccessGrantsInstance(accountID string) (*AccessGrantsInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return nil, ErrAccessGrantsInstanceNotFound
	}
	copy := *account.AccessGrantsInstance
	return &copy, nil
}

func (s *Service) DeleteAccessGrantsInstance(accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return ErrAccessGrantsInstanceNotFound
	}
	account.AccessGrantsInstance = nil
	account.AccessGrantsPolicy = nil
	account.AccessGrantsLocations = make(map[string]*AccessGrantsLocation)
	account.AccessGrantsGrants = make(map[string]*AccessGrant)
	return nil
}

func (s *Service) ListAccessGrantsInstances(accountID string) []*AccessGrantsInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return nil
	}
	instance := *account.AccessGrantsInstance
	return []*AccessGrantsInstance{&instance}
}

func (s *Service) PutAccessGrantsInstanceResourcePolicy(accountID, policy, organization string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return ErrAccessGrantsInstanceNotFound
	}
	account.AccessGrantsPolicy = &AccessGrantsResourcePolicy{
		Policy:       policy,
		Organization: organization,
		CreatedAt:    time.Now().UTC(),
	}
	return nil
}

func (s *Service) GetAccessGrantsInstanceResourcePolicy(accountID string) (*AccessGrantsResourcePolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return nil, ErrAccessGrantsInstanceNotFound
	}
	if account.AccessGrantsPolicy == nil {
		return nil, ErrAccessGrantsResourcePolicyNotFound
	}
	copy := *account.AccessGrantsPolicy
	return &copy, nil
}

func (s *Service) DeleteAccessGrantsInstanceResourcePolicy(accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return ErrAccessGrantsInstanceNotFound
	}
	if account.AccessGrantsPolicy == nil {
		return ErrAccessGrantsResourcePolicyNotFound
	}
	account.AccessGrantsPolicy = nil
	return nil
}

func (s *Service) AssociateAccessGrantsIdentityCenter(accountID, identityCenterArn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return ErrAccessGrantsInstanceNotFound
	}
	account.AccessGrantsInstance.IdentityCenterArn = identityCenterArn
	account.AccessGrantsInstance.IdentityCenterInstanceArn = identityCenterArn
	return nil
}

func (s *Service) CreateAccessGrantsLocation(accountID string, location *AccessGrantsLocation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return ErrAccessGrantsInstanceNotFound
	}
	account.AccessGrantsLocations[location.ID] = location
	return nil
}

func (s *Service) UpdateAccessGrantsLocation(accountID, locationID, iamRoleArn string) (*AccessGrantsLocation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return nil, ErrAccessGrantsInstanceNotFound
	}
	location, ok := account.AccessGrantsLocations[locationID]
	if !ok {
		return nil, ErrAccessGrantsLocationNotFound
	}
	location.IAMRoleArn = iamRoleArn
	copy := *location
	return &copy, nil
}

func (s *Service) GetAccessGrantsLocation(accountID, locationID string) (*AccessGrantsLocation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return nil, ErrAccessGrantsInstanceNotFound
	}
	location, ok := account.AccessGrantsLocations[locationID]
	if !ok {
		return nil, ErrAccessGrantsLocationNotFound
	}
	copy := *location
	return &copy, nil
}

func (s *Service) DeleteAccessGrantsLocation(accountID, locationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return ErrAccessGrantsInstanceNotFound
	}
	if _, ok := account.AccessGrantsLocations[locationID]; !ok {
		return ErrAccessGrantsLocationNotFound
	}
	delete(account.AccessGrantsLocations, locationID)
	return nil
}

func (s *Service) ListAccessGrantsLocations(accountID string) []*AccessGrantsLocation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return nil
	}
	list := make([]*AccessGrantsLocation, 0, len(account.AccessGrantsLocations))
	for _, location := range account.AccessGrantsLocations {
		list = append(list, location)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	return list
}

func (s *Service) CreateAccessGrant(accountID string, grant *AccessGrant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return ErrAccessGrantsInstanceNotFound
	}
	if _, ok := account.AccessGrantsLocations[grant.AccessGrantsLocationID]; !ok {
		return ErrAccessGrantsLocationNotFound
	}
	account.AccessGrantsGrants[grant.ID] = grant
	return nil
}

func (s *Service) GetAccessGrant(accountID, grantID string) (*AccessGrant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return nil, ErrAccessGrantsInstanceNotFound
	}
	grant, ok := account.AccessGrantsGrants[grantID]
	if !ok {
		return nil, ErrAccessGrantNotFound
	}
	copy := *grant
	return &copy, nil
}

func (s *Service) DeleteAccessGrant(accountID, grantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return ErrAccessGrantsInstanceNotFound
	}
	if _, ok := account.AccessGrantsGrants[grantID]; !ok {
		return ErrAccessGrantNotFound
	}
	delete(account.AccessGrantsGrants, grantID)
	return nil
}

func (s *Service) ListAccessGrants(accountID string) []*AccessGrant {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[accountID]
	if !ok || account.AccessGrantsInstance == nil {
		return nil
	}
	list := make([]*AccessGrant, 0, len(account.AccessGrantsGrants))
	for _, grant := range account.AccessGrantsGrants {
		list = append(list, grant)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	return list
}

func (s *Service) CreateOutpostsBucket(accountID string, bucket *OutpostsBucket) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		account = &AccountState{
			AccountID:               accountID,
			AccessPoints:            make(map[string]*AccessPoint),
			MultiRegionAccessPoints: make(map[string]*MultiRegionAccessPoint),
			MultiRegionOperations:   make(map[string]*MultiRegionAccessPointOperation),
			Jobs:                    make(map[string]*Job),
			StorageLensConfigs:      make(map[string]*StorageLensConfiguration),
			StorageLensGroups:       make(map[string]*StorageLensGroup),
			AccessGrantsLocations:   make(map[string]*AccessGrantsLocation),
			AccessGrantsGrants:      make(map[string]*AccessGrant),
			OutpostsBuckets:         make(map[string]*OutpostsBucket),
		}
		s.accounts[accountID] = account
	}
	if _, exists := account.OutpostsBuckets[bucket.Name]; exists {
		return ErrOutpostsBucketAlreadyExists
	}
	account.OutpostsBuckets[bucket.Name] = bucket
	return nil
}

func (s *Service) resolveOutpostsBucketLocked(accountID, name string) (*AccountState, *OutpostsBucket) {
	if account, ok := s.accounts[accountID]; ok {
		if bucket, exists := account.OutpostsBuckets[name]; exists {
			return account, bucket
		}
	}
	for _, account := range s.accounts {
		if bucket, exists := account.OutpostsBuckets[name]; exists {
			return account, bucket
		}
	}
	return nil, nil
}

func (s *Service) GetOutpostsBucket(accountID, name string) (*OutpostsBucket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return nil, ErrOutpostsBucketNotFound
	}
	copy := *bucket
	return &copy, nil
}

func (s *Service) DeleteOutpostsBucket(accountID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if account == nil || bucket == nil {
		return ErrOutpostsBucketNotFound
	}
	delete(account.OutpostsBuckets, bucket.Name)
	return nil
}

func (s *Service) ListOutpostsBuckets(accountID, outpostID string) []*OutpostsBucket {
	s.mu.RLock()
	defer s.mu.RUnlock()
	candidates := make([]*OutpostsBucket, 0)
	if account, ok := s.accounts[accountID]; ok && len(account.OutpostsBuckets) > 0 {
		for _, bucket := range account.OutpostsBuckets {
			candidates = append(candidates, bucket)
		}
	} else {
		for _, account := range s.accounts {
			for _, bucket := range account.OutpostsBuckets {
				candidates = append(candidates, bucket)
			}
		}
	}
	list := make([]*OutpostsBucket, 0, len(candidates))
	for _, bucket := range candidates {
		if outpostID != "" && bucket.OutpostID != outpostID {
			continue
		}
		list = append(list, bucket)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}

func (s *Service) PutOutpostsBucketTagging(accountID, name string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return ErrOutpostsBucketNotFound
	}
	bucket.Tags = tags
	return nil
}

func (s *Service) GetOutpostsBucketTagging(accountID, name string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return nil, ErrOutpostsBucketNotFound
	}
	if len(bucket.Tags) == 0 {
		return nil, ErrOutpostsBucketTaggingNotFound
	}
	out := make(map[string]string, len(bucket.Tags))
	for k, v := range bucket.Tags {
		out[k] = v
	}
	return out, nil
}

func (s *Service) DeleteOutpostsBucketTagging(accountID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return ErrOutpostsBucketNotFound
	}
	if len(bucket.Tags) == 0 {
		return ErrOutpostsBucketTaggingNotFound
	}
	bucket.Tags = nil
	return nil
}

func (s *Service) PutOutpostsBucketLifecycle(accountID, name, xmlBody string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return ErrOutpostsBucketNotFound
	}
	bucket.LifecycleXML = xmlBody
	return nil
}

func (s *Service) GetOutpostsBucketLifecycle(accountID, name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return "", ErrOutpostsBucketNotFound
	}
	if bucket.LifecycleXML == "" {
		return "", ErrOutpostsBucketLifecycleNotFound
	}
	return bucket.LifecycleXML, nil
}

func (s *Service) DeleteOutpostsBucketLifecycle(accountID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return ErrOutpostsBucketNotFound
	}
	if bucket.LifecycleXML == "" {
		return ErrOutpostsBucketLifecycleNotFound
	}
	bucket.LifecycleXML = ""
	return nil
}

func (s *Service) PutOutpostsBucketPolicy(accountID, name, policy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return ErrOutpostsBucketNotFound
	}
	bucket.Policy = policy
	return nil
}

func (s *Service) GetOutpostsBucketPolicy(accountID, name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return "", ErrOutpostsBucketNotFound
	}
	if bucket.Policy == "" {
		return "", ErrOutpostsBucketPolicyNotFound
	}
	return bucket.Policy, nil
}

func (s *Service) DeleteOutpostsBucketPolicy(accountID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return ErrOutpostsBucketNotFound
	}
	if bucket.Policy == "" {
		return ErrOutpostsBucketPolicyNotFound
	}
	bucket.Policy = ""
	return nil
}

func (s *Service) PutOutpostsBucketReplication(accountID, name, xmlBody string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return ErrOutpostsBucketNotFound
	}
	bucket.ReplicationXML = xmlBody
	return nil
}

func (s *Service) SetOutpostsBucketVersioning(accountID, name, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return ErrOutpostsBucketNotFound
	}
	bucket.VersioningStatus = status
	return nil
}

func (s *Service) GetOutpostsBucketReplication(accountID, name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return "", ErrOutpostsBucketNotFound
	}
	if bucket.ReplicationXML == "" {
		return "", ErrOutpostsBucketReplicationNotFound
	}
	return bucket.ReplicationXML, nil
}

func (s *Service) DeleteOutpostsBucketReplication(accountID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, bucket := s.resolveOutpostsBucketLocked(accountID, name)
	if bucket == nil {
		return ErrOutpostsBucketNotFound
	}
	if bucket.ReplicationXML == "" {
		return ErrOutpostsBucketReplicationNotFound
	}
	bucket.ReplicationXML = ""
	return nil
}
