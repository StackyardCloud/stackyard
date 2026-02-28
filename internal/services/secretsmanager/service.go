package secretsmanager

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrNotFound         = errors.New("resource not found")
	ErrInvalidState     = errors.New("invalid state")
	ErrThrottling       = errors.New("throttling")
	ErrLimitExceeded    = errors.New("limit exceeded")
)

const (
	DefaultRegion          = "us-east-1"
	DefaultAccountID       = "123456789012"
	defaultMaxTags         = 50
	defaultMaxFilters      = 10
	defaultMaxFilterValues = 10
	defaultMaxPolicyBytes  = 20 * 1024
	defaultThrottleLimit   = 1000
	defaultThrottleWindow  = time.Second
)

type SecretVersion struct {
	VersionID    string
	SecretString string
	SecretBinary string
	CreatedDate  time.Time
	Stages       []string
}

type Secret struct {
	ARN                    string
	Name                   string
	Description            string
	KmsKeyID               string
	OwningService          string
	PrimaryRegion          string
	CreatedDate            time.Time
	LastChangedDate        time.Time
	LastAccessedDate       *time.Time
	LastRotatedDate        *time.Time
	NextRotationDate       *time.Time
	DeletedDate            *time.Time
	RotationEnabled        bool
	RotationLambdaARN      string
	AutomaticallyAfterDays int64
	ResourcePolicy         string
	Tags                   map[string]string
	ReplicationStatus      map[string]*ReplicationStatus
	Versions               map[string]*SecretVersion
}

type ReplicationStatus struct {
	Region           string
	KmsKeyID         string
	Status           string
	StatusMessage    string
	LastAccessedDate *time.Time
}

type SecretFilter struct {
	Key    string
	Values []string
}

type SecretVersionListItem struct {
	VersionID     string
	VersionStages []string
	CreatedDate   time.Time
}

type SecretSummary struct {
	ARN               string
	Name              string
	Description       string
	KmsKeyID          string
	OwningService     string
	PrimaryRegion     string
	CreatedDate       time.Time
	LastChangedDate   time.Time
	LastAccessedDate  *time.Time
	LastRotatedDate   *time.Time
	NextRotationDate  *time.Time
	DeletedDate       *time.Time
	RotationEnabled   bool
	Tags              map[string]string
	ReplicationStatus []ReplicationStatus
}

type CreateSecretInput struct {
	Name               string
	ClientRequestToken string
	Description        string
	KmsKeyID           string
	OwningService      string
	SecretBinary       string
	SecretString       string
	Tags               map[string]string
}

type CreateSecretOutput struct {
	ARN       string
	Name      string
	VersionID string
}

type ListSecretsInput struct {
	NextToken              string
	MaxResults             int32
	IncludePlannedDeletion bool
	SortOrder              string
	Filters                []SecretFilter
}

type ListSecretsOutput struct {
	SecretList []SecretSummary
	NextToken  string
}

type UpdateSecretInput struct {
	SecretID           string
	ClientRequestToken string
	Description        string
	KmsKeyID           string
	SecretBinary       string
	SecretString       string
}

type UpdateSecretOutput struct {
	ARN       string
	Name      string
	VersionID string
}

type DeleteSecretInput struct {
	SecretID                   string
	RecoveryWindowInDays       int64
	ForceDeleteWithoutRecovery bool
}

type DeleteSecretOutput struct {
	ARN          string
	Name         string
	DeletionDate time.Time
}

type RestoreSecretOutput struct {
	ARN  string
	Name string
}

type PutSecretValueInput struct {
	SecretID           string
	ClientRequestToken string
	SecretBinary       string
	SecretString       string
	VersionStages      []string
}

type PutSecretValueOutput struct {
	ARN           string
	Name          string
	VersionID     string
	VersionStages []string
}

type GetSecretValueInput struct {
	SecretID     string
	VersionID    string
	VersionStage string
}

type GetSecretValueOutput struct {
	ARN           string
	Name          string
	VersionID     string
	VersionStages []string
	SecretString  string
	SecretBinary  string
	CreatedDate   time.Time
}

type ListSecretVersionIDsInput struct {
	SecretID          string
	NextToken         string
	MaxResults        int32
	IncludeDeprecated bool
}

type ListSecretVersionIDsOutput struct {
	ARN       string
	Name      string
	Versions  []SecretVersionListItem
	NextToken string
}

type UpdateSecretVersionStageInput struct {
	SecretID            string
	VersionStage        string
	RemoveFromVersionID string
	MoveToVersionID     string
}

type UpdateSecretVersionStageOutput struct {
	ARN  string
	Name string
}

type BatchGetSecretValueInput struct {
	SecretIDList []string
	NextToken    string
	MaxResults   int32
	Filters      []SecretFilter
}

type BatchGetSecretValueError struct {
	SecretID  string
	ErrorCode string
	Message   string
}

type BatchGetSecretValueOutput struct {
	SecretValues []GetSecretValueOutput
	Errors       []BatchGetSecretValueError
	NextToken    string
}

type GetRandomPasswordInput struct {
	PasswordLength          int64
	ExcludeCharacters       string
	ExcludeNumbers          bool
	ExcludePunctuation      bool
	ExcludeUppercase        bool
	ExcludeLowercase        bool
	IncludeSpace            bool
	RequireEachIncludedType bool
}

type RotateSecretInput struct {
	SecretID               string
	ClientRequestToken     string
	RotationLambdaARN      string
	AutomaticallyAfterDays int64
	RotateImmediately      bool
}

type RotateSecretOutput struct {
	ARN       string
	Name      string
	VersionID string
}

type CancelRotateSecretOutput struct {
	ARN       string
	Name      string
	VersionID string
}

type ReplicaRegionInput struct {
	Region   string
	KmsKeyID string
}

type ReplicateSecretToRegionsInput struct {
	SecretID                    string
	AddReplicaRegions           []ReplicaRegionInput
	ForceOverwriteReplicaSecret bool
}

type ReplicateSecretToRegionsOutput struct {
	ARN               string
	ReplicationStatus []ReplicationStatus
}

type RemoveRegionsFromReplicationInput struct {
	SecretID             string
	RemoveReplicaRegions []string
}

type RemoveRegionsFromReplicationOutput struct {
	ARN               string
	ReplicationStatus []ReplicationStatus
}

type StopReplicationToReplicaInput struct {
	SecretID      string
	ReplicaRegion string
}

type StopReplicationToReplicaOutput struct {
	ARN string
}

type PutResourcePolicyInput struct {
	SecretID          string
	ResourcePolicy    string
	BlockPublicPolicy bool
}

type PutResourcePolicyOutput struct {
	ARN  string
	Name string
}

type GetResourcePolicyOutput struct {
	ARN            string
	Name           string
	ResourcePolicy string
}

type DeleteResourcePolicyOutput struct {
	ARN  string
	Name string
}

type ValidateResourcePolicyInput struct {
	SecretID          string
	ResourcePolicy    string
	BlockPublicPolicy bool
}

type ValidateResourcePolicyError struct {
	CheckName    string
	ErrorMessage string
}

type ValidateResourcePolicyOutput struct {
	PolicyValidationPassed bool
	ValidationErrors       []ValidateResourcePolicyError
}

type TagResourceInput struct {
	SecretID string
	Tags     map[string]string
}

type UntagResourceInput struct {
	SecretID string
	TagKeys  []string
}

type Service struct {
	mu           sync.Mutex
	seq          uint64
	secrets      map[string]*Secret
	secretByName map[string]string
	createTokens map[string]createTokenRecord
	rotateTokens map[string]rotateTokenRecord
	calls        []time.Time
}

type createTokenRecord struct {
	ARN           string
	Name          string
	SecretString  string
	SecretBinary  string
	KmsKeyID      string
	OwningService string
	Tags          map[string]string
}

type rotateTokenRecord struct {
	ARN                    string
	RotationLambdaARN      string
	AutomaticallyAfterDays int64
	RotateImmediately      bool
	VersionID              string
}

func NewService() *Service {
	return &Service{
		secrets:      map[string]*Secret{},
		secretByName: map[string]string{},
		createTokens: map[string]createTokenRecord{},
		rotateTokens: map[string]rotateTokenRecord{},
		calls:        make([]time.Time, 0, defaultThrottleLimit),
	}
}

func (s *Service) RecordAPICall() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.checkThrottleLocked()
}

func (s *Service) CreateSecret(input CreateSecretInput) (CreateSecretOutput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CreateSecretOutput{}, ErrInvalidParameter
	}
	if strings.TrimSpace(input.SecretString) == "" && strings.TrimSpace(input.SecretBinary) == "" {
		return CreateSecretOutput{}, ErrInvalidParameter
	}

	tags := cloneTags(input.Tags)
	if len(tags) > defaultMaxTags {
		return CreateSecretOutput{}, ErrLimitExceeded
	}
	if err := validateTagMap(tags); err != nil {
		return CreateSecretOutput{}, err
	}

	token := strings.TrimSpace(input.ClientRequestToken)
	if token == "" {
		token = s.nextVersionID()
	}
	secretString := strings.TrimSpace(input.SecretString)
	secretBinary := strings.TrimSpace(input.SecretBinary)
	kmsKeyID := strings.TrimSpace(input.KmsKeyID)
	owningService := strings.TrimSpace(input.OwningService)

	s.mu.Lock()
	defer s.mu.Unlock()

	if record, ok := s.createTokens[token]; ok {
		if record.Name != name || record.SecretString != secretString || record.SecretBinary != secretBinary || record.KmsKeyID != kmsKeyID || record.OwningService != owningService || !stringMapEqual(record.Tags, tags) {
			return CreateSecretOutput{}, ErrInvalidParameter
		}
		if existing, ok := s.secrets[record.ARN]; ok {
			versionID := token
			if _, ok := existing.Versions[versionID]; !ok {
				versionID = currentVersionIDLocked(existing)
			}
			return CreateSecretOutput{ARN: existing.ARN, Name: existing.Name, VersionID: versionID}, nil
		}
	}

	if arn, exists := s.secretByName[name]; exists {
		if secret, ok := s.secrets[arn]; ok && secret.DeletedDate == nil {
			return CreateSecretOutput{}, ErrInvalidParameter
		}
	}

	now := time.Now().UTC()
	arn := s.nextSecretARNLocked(name)
	version := &SecretVersion{
		VersionID:    token,
		SecretString: secretString,
		SecretBinary: secretBinary,
		CreatedDate:  now,
		Stages:       []string{"AWSCURRENT"},
	}

	secret := &Secret{
		ARN:               arn,
		Name:              name,
		Description:       strings.TrimSpace(input.Description),
		KmsKeyID:          kmsKeyID,
		OwningService:     owningService,
		PrimaryRegion:     DefaultRegion,
		CreatedDate:       now,
		LastChangedDate:   now,
		Tags:              tags,
		ReplicationStatus: map[string]*ReplicationStatus{},
		Versions:          map[string]*SecretVersion{token: version},
	}

	s.secrets[arn] = secret
	s.secretByName[name] = arn
	s.createTokens[token] = createTokenRecord{
		ARN:           arn,
		Name:          name,
		SecretString:  secretString,
		SecretBinary:  secretBinary,
		KmsKeyID:      kmsKeyID,
		OwningService: owningService,
		Tags:          cloneTags(tags),
	}

	return CreateSecretOutput{
		ARN:       arn,
		Name:      name,
		VersionID: token,
	}, nil
}

func (s *Service) DescribeSecret(secretID string) (Secret, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return Secret{}, err
	}
	return cloneSecret(secret), nil
}

func (s *Service) ListSecrets(input ListSecretsInput) (ListSecretsOutput, error) {
	maxResults := input.MaxResults
	if maxResults == 0 {
		maxResults = 100
	}
	if maxResults < 0 || maxResults > 100 {
		return ListSecretsOutput{}, ErrInvalidParameter
	}

	start, err := parseNextTokenOffset(input.NextToken)
	if err != nil {
		return ListSecretsOutput{}, ErrInvalidParameter
	}

	sortOrder := strings.ToLower(strings.TrimSpace(input.SortOrder))
	if sortOrder != "" && sortOrder != "asc" && sortOrder != "desc" {
		return ListSecretsOutput{}, ErrInvalidParameter
	}
	filters, err := normalizeSecretFilters(input.Filters)
	if err != nil {
		return ListSecretsOutput{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	arns := make([]string, 0, len(s.secrets))
	for arn, secret := range s.secrets {
		if secret.DeletedDate != nil && !input.IncludePlannedDeletion {
			continue
		}
		if !matchesSecretFilters(secret, filters) {
			continue
		}
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	if sortOrder == "desc" {
		for i, j := 0, len(arns)-1; i < j; i, j = i+1, j-1 {
			arns[i], arns[j] = arns[j], arns[i]
		}
	}

	if start < 0 || start > len(arns) {
		return ListSecretsOutput{}, ErrInvalidParameter
	}

	end := start + int(maxResults)
	if end > len(arns) {
		end = len(arns)
	}

	summaries := make([]SecretSummary, 0, end-start)
	for _, arn := range arns[start:end] {
		secret := s.secrets[arn]
		summaries = append(summaries, SecretSummary{
			ARN:               secret.ARN,
			Name:              secret.Name,
			Description:       secret.Description,
			KmsKeyID:          secret.KmsKeyID,
			OwningService:     secret.OwningService,
			PrimaryRegion:     secret.PrimaryRegion,
			CreatedDate:       secret.CreatedDate,
			LastChangedDate:   secret.LastChangedDate,
			LastAccessedDate:  cloneTimePtr(secret.LastAccessedDate),
			LastRotatedDate:   cloneTimePtr(secret.LastRotatedDate),
			NextRotationDate:  cloneTimePtr(secret.NextRotationDate),
			DeletedDate:       cloneTimePtr(secret.DeletedDate),
			RotationEnabled:   secret.RotationEnabled,
			Tags:              cloneTags(secret.Tags),
			ReplicationStatus: cloneReplicationStatusList(secret.ReplicationStatus),
		})
	}

	out := ListSecretsOutput{SecretList: summaries}
	if end < len(arns) {
		out.NextToken = strconv.Itoa(end)
	}
	return out, nil
}

func (s *Service) UpdateSecret(input UpdateSecretInput) (UpdateSecretOutput, error) {
	secretID := strings.TrimSpace(input.SecretID)
	if secretID == "" {
		return UpdateSecretOutput{}, ErrInvalidParameter
	}

	clientToken := strings.TrimSpace(input.ClientRequestToken)
	if clientToken == "" && (strings.TrimSpace(input.SecretString) != "" || strings.TrimSpace(input.SecretBinary) != "") {
		clientToken = s.nextVersionID()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return UpdateSecretOutput{}, err
	}
	if secret.DeletedDate != nil {
		return UpdateSecretOutput{}, ErrInvalidState
	}

	if desc := strings.TrimSpace(input.Description); desc != "" {
		secret.Description = desc
	}
	if key := strings.TrimSpace(input.KmsKeyID); key != "" {
		secret.KmsKeyID = key
	}

	out := UpdateSecretOutput{ARN: secret.ARN, Name: secret.Name}
	if strings.TrimSpace(input.SecretString) != "" || strings.TrimSpace(input.SecretBinary) != "" {
		version, err := s.putSecretValueLocked(secret, putValueInput{
			ClientRequestToken: clientToken,
			SecretString:       strings.TrimSpace(input.SecretString),
			SecretBinary:       strings.TrimSpace(input.SecretBinary),
			VersionStages:      nil,
		})
		if err != nil {
			return UpdateSecretOutput{}, err
		}
		out.VersionID = version.VersionID
	}
	secret.LastChangedDate = time.Now().UTC()

	return out, nil
}

func (s *Service) DeleteSecret(input DeleteSecretInput) (DeleteSecretOutput, error) {
	secretID := strings.TrimSpace(input.SecretID)
	if secretID == "" {
		return DeleteSecretOutput{}, ErrInvalidParameter
	}
	if input.ForceDeleteWithoutRecovery && input.RecoveryWindowInDays > 0 {
		return DeleteSecretOutput{}, ErrInvalidParameter
	}
	if !input.ForceDeleteWithoutRecovery {
		if input.RecoveryWindowInDays == 0 {
			input.RecoveryWindowInDays = 30
		}
		if input.RecoveryWindowInDays < 7 || input.RecoveryWindowInDays > 30 {
			return DeleteSecretOutput{}, ErrInvalidParameter
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return DeleteSecretOutput{}, err
	}
	if len(secret.ReplicationStatus) > 0 {
		return DeleteSecretOutput{}, ErrInvalidState
	}

	now := time.Now().UTC()
	deletionDate := now
	if !input.ForceDeleteWithoutRecovery {
		deletionDate = now.Add(time.Duration(input.RecoveryWindowInDays) * 24 * time.Hour)
		secret.DeletedDate = &deletionDate
		secret.LastChangedDate = now
	} else {
		delete(s.secrets, secret.ARN)
		if arn, ok := s.secretByName[secret.Name]; ok && arn == secret.ARN {
			delete(s.secretByName, secret.Name)
		}
	}

	return DeleteSecretOutput{
		ARN:          secret.ARN,
		Name:         secret.Name,
		DeletionDate: deletionDate,
	}, nil
}

func (s *Service) RestoreSecret(secretID string) (RestoreSecretOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return RestoreSecretOutput{}, err
	}
	if secret.DeletedDate == nil {
		return RestoreSecretOutput{}, ErrInvalidState
	}
	secret.DeletedDate = nil
	secret.LastChangedDate = time.Now().UTC()

	return RestoreSecretOutput{ARN: secret.ARN, Name: secret.Name}, nil
}

func (s *Service) PutSecretValue(input PutSecretValueInput) (PutSecretValueOutput, error) {
	if strings.TrimSpace(input.SecretID) == "" {
		return PutSecretValueOutput{}, ErrInvalidParameter
	}
	if strings.TrimSpace(input.SecretString) == "" && strings.TrimSpace(input.SecretBinary) == "" {
		return PutSecretValueOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(input.SecretID)
	if err != nil {
		return PutSecretValueOutput{}, err
	}
	if secret.DeletedDate != nil {
		return PutSecretValueOutput{}, ErrInvalidState
	}

	version, err := s.putSecretValueLocked(secret, putValueInput{
		ClientRequestToken: strings.TrimSpace(input.ClientRequestToken),
		SecretString:       strings.TrimSpace(input.SecretString),
		SecretBinary:       strings.TrimSpace(input.SecretBinary),
		VersionStages:      dedupeStrings(input.VersionStages),
	})
	if err != nil {
		return PutSecretValueOutput{}, err
	}

	return PutSecretValueOutput{
		ARN:           secret.ARN,
		Name:          secret.Name,
		VersionID:     version.VersionID,
		VersionStages: append([]string(nil), version.Stages...),
	}, nil
}

func (s *Service) GetSecretValue(input GetSecretValueInput) (GetSecretValueOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(input.SecretID)
	if err != nil {
		return GetSecretValueOutput{}, err
	}
	if secret.DeletedDate != nil {
		return GetSecretValueOutput{}, ErrInvalidState
	}

	version, err := selectVersionLocked(secret, strings.TrimSpace(input.VersionID), strings.TrimSpace(input.VersionStage))
	if err != nil {
		return GetSecretValueOutput{}, err
	}
	now := time.Now().UTC()
	secret.LastAccessedDate = cloneTimePtr(&now)

	return GetSecretValueOutput{
		ARN:           secret.ARN,
		Name:          secret.Name,
		VersionID:     version.VersionID,
		VersionStages: append([]string(nil), version.Stages...),
		SecretString:  version.SecretString,
		SecretBinary:  version.SecretBinary,
		CreatedDate:   version.CreatedDate,
	}, nil
}

func (s *Service) ListSecretVersionIDs(input ListSecretVersionIDsInput) (ListSecretVersionIDsOutput, error) {
	maxResults := input.MaxResults
	if maxResults == 0 {
		maxResults = 100
	}
	if maxResults < 0 || maxResults > 100 {
		return ListSecretVersionIDsOutput{}, ErrInvalidParameter
	}

	start, err := parseNextTokenOffset(input.NextToken)
	if err != nil {
		return ListSecretVersionIDsOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(input.SecretID)
	if err != nil {
		return ListSecretVersionIDsOutput{}, err
	}

	versions := make([]*SecretVersion, 0, len(secret.Versions))
	for _, version := range secret.Versions {
		if !input.IncludeDeprecated && len(version.Stages) == 0 {
			continue
		}
		versions = append(versions, version)
	}
	sort.SliceStable(versions, func(i, j int) bool {
		return versions[i].CreatedDate.After(versions[j].CreatedDate)
	})

	if start < 0 || start > len(versions) {
		return ListSecretVersionIDsOutput{}, ErrInvalidParameter
	}
	end := start + int(maxResults)
	if end > len(versions) {
		end = len(versions)
	}

	items := make([]SecretVersionListItem, 0, end-start)
	for _, version := range versions[start:end] {
		items = append(items, SecretVersionListItem{
			VersionID:     version.VersionID,
			VersionStages: append([]string(nil), version.Stages...),
			CreatedDate:   version.CreatedDate,
		})
	}

	out := ListSecretVersionIDsOutput{
		ARN:      secret.ARN,
		Name:     secret.Name,
		Versions: items,
	}
	if end < len(versions) {
		out.NextToken = strconv.Itoa(end)
	}
	return out, nil
}

func (s *Service) UpdateSecretVersionStage(input UpdateSecretVersionStageInput) (UpdateSecretVersionStageOutput, error) {
	secretID := strings.TrimSpace(input.SecretID)
	versionStage := strings.TrimSpace(input.VersionStage)
	removeFrom := strings.TrimSpace(input.RemoveFromVersionID)
	moveTo := strings.TrimSpace(input.MoveToVersionID)

	if secretID == "" || versionStage == "" {
		return UpdateSecretVersionStageOutput{}, ErrInvalidParameter
	}
	if removeFrom != "" && moveTo != "" && removeFrom == moveTo {
		return UpdateSecretVersionStageOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return UpdateSecretVersionStageOutput{}, err
	}
	if secret.DeletedDate != nil {
		return UpdateSecretVersionStageOutput{}, ErrInvalidState
	}

	if removeFrom != "" {
		version, ok := secret.Versions[removeFrom]
		if !ok {
			return UpdateSecretVersionStageOutput{}, ErrNotFound
		}
		if !sliceContains(version.Stages, versionStage) {
			return UpdateSecretVersionStageOutput{}, ErrInvalidParameter
		}
		version.Stages = removeString(version.Stages, versionStage)
		sort.Strings(version.Stages)
	}

	if moveTo != "" {
		version, ok := secret.Versions[moveTo]
		if !ok {
			return UpdateSecretVersionStageOutput{}, ErrNotFound
		}
		for _, current := range secret.Versions {
			if current.VersionID == version.VersionID {
				continue
			}
			current.Stages = removeString(current.Stages, versionStage)
		}
		if !sliceContains(version.Stages, versionStage) {
			version.Stages = append(version.Stages, versionStage)
			sort.Strings(version.Stages)
		}
		secret.LastChangedDate = time.Now().UTC()
	}

	return UpdateSecretVersionStageOutput{ARN: secret.ARN, Name: secret.Name}, nil
}

func (s *Service) BatchGetSecretValue(input BatchGetSecretValueInput) (BatchGetSecretValueOutput, error) {
	maxResults := input.MaxResults
	if maxResults == 0 {
		maxResults = 20
	}
	if maxResults < 0 || maxResults > 100 {
		return BatchGetSecretValueOutput{}, ErrInvalidParameter
	}

	start, err := parseNextTokenOffset(input.NextToken)
	if err != nil {
		return BatchGetSecretValueOutput{}, ErrInvalidParameter
	}
	filters, err := normalizeSecretFilters(input.Filters)
	if err != nil {
		return BatchGetSecretValueOutput{}, err
	}
	if len(input.SecretIDList) > 0 && len(filters) > 0 {
		return BatchGetSecretValueOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	targets := dedupeStrings(input.SecretIDList)
	if len(targets) == 0 && len(filters) > 0 {
		targets = make([]string, 0, len(s.secrets))
		for _, secret := range s.secrets {
			if secret.DeletedDate != nil {
				continue
			}
			if !matchesSecretFilters(secret, filters) {
				continue
			}
			targets = append(targets, secret.ARN)
		}
		sort.Strings(targets)
	}
	if len(targets) == 0 && len(filters) == 0 {
		targets = make([]string, 0, len(s.secrets))
		for _, secret := range s.secrets {
			if secret.DeletedDate == nil {
				targets = append(targets, secret.ARN)
			}
		}
		sort.Strings(targets)
	}

	if start < 0 || start > len(targets) {
		return BatchGetSecretValueOutput{}, ErrInvalidParameter
	}
	end := start + int(maxResults)
	if end > len(targets) {
		end = len(targets)
	}

	out := BatchGetSecretValueOutput{
		SecretValues: make([]GetSecretValueOutput, 0, end-start),
		Errors:       make([]BatchGetSecretValueError, 0),
	}

	for _, id := range targets[start:end] {
		secret, err := s.lookupSecretLocked(id)
		if err != nil {
			out.Errors = append(out.Errors, BatchGetSecretValueError{
				SecretID:  id,
				ErrorCode: "ResourceNotFoundException",
				Message:   "secret not found",
			})
			continue
		}
		if secret.DeletedDate != nil {
			out.Errors = append(out.Errors, BatchGetSecretValueError{
				SecretID:  id,
				ErrorCode: "InvalidRequestException",
				Message:   "secret is deleted",
			})
			continue
		}

		version, err := selectVersionLocked(secret, "", "")
		if err != nil {
			out.Errors = append(out.Errors, BatchGetSecretValueError{
				SecretID:  id,
				ErrorCode: "ResourceNotFoundException",
				Message:   "secret version not found",
			})
			continue
		}
		now := time.Now().UTC()
		secret.LastAccessedDate = cloneTimePtr(&now)
		out.SecretValues = append(out.SecretValues, GetSecretValueOutput{
			ARN:           secret.ARN,
			Name:          secret.Name,
			VersionID:     version.VersionID,
			VersionStages: append([]string(nil), version.Stages...),
			SecretString:  version.SecretString,
			SecretBinary:  version.SecretBinary,
			CreatedDate:   version.CreatedDate,
		})
	}

	if end < len(targets) {
		out.NextToken = strconv.Itoa(end)
	}
	return out, nil
}

func (s *Service) GetRandomPassword(input GetRandomPasswordInput) (string, error) {
	length := input.PasswordLength
	if length == 0 {
		length = 32
	}
	if length < 1 || length > 4096 {
		return "", ErrInvalidParameter
	}

	lower := "abcdefghijklmnopqrstuvwxyz"
	upper := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers := "0123456789"
	punctuation := "!\"#$%&'()*+,-./:;<=>?@[\\]^_{|}~"
	space := " "

	if input.ExcludeLowercase {
		lower = ""
	}
	if input.ExcludeUppercase {
		upper = ""
	}
	if input.ExcludeNumbers {
		numbers = ""
	}
	if input.ExcludePunctuation {
		punctuation = ""
	}
	if !input.IncludeSpace {
		space = ""
	}

	excludeSet := map[rune]struct{}{}
	for _, r := range input.ExcludeCharacters {
		excludeSet[r] = struct{}{}
	}
	filter := func(chars string) string {
		var b strings.Builder
		for _, r := range chars {
			if _, excluded := excludeSet[r]; excluded {
				continue
			}
			b.WriteRune(r)
		}
		return b.String()
	}

	lower = filter(lower)
	upper = filter(upper)
	numbers = filter(numbers)
	punctuation = filter(punctuation)
	space = filter(space)

	pools := []string{}
	for _, pool := range []string{lower, upper, numbers, punctuation, space} {
		if pool != "" {
			pools = append(pools, pool)
		}
	}
	if len(pools) == 0 {
		return "", ErrInvalidParameter
	}

	all := strings.Join(pools, "")
	if all == "" {
		return "", ErrInvalidParameter
	}

	out := make([]rune, 0, int(length))
	if input.RequireEachIncludedType {
		if int64(len(pools)) > length {
			return "", ErrInvalidParameter
		}
		for _, pool := range pools {
			ch, err := randomRune(pool)
			if err != nil {
				return "", err
			}
			out = append(out, ch)
		}
	}
	for len(out) < int(length) {
		ch, err := randomRune(all)
		if err != nil {
			return "", err
		}
		out = append(out, ch)
	}

	if err := shuffleRunes(out); err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *Service) RotateSecret(input RotateSecretInput) (RotateSecretOutput, error) {
	secretID := strings.TrimSpace(input.SecretID)
	if secretID == "" {
		return RotateSecretOutput{}, ErrInvalidParameter
	}
	if input.AutomaticallyAfterDays < 0 || input.AutomaticallyAfterDays > 1000 {
		return RotateSecretOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return RotateSecretOutput{}, err
	}
	if secret.DeletedDate != nil {
		return RotateSecretOutput{}, ErrInvalidState
	}

	now := time.Now().UTC()
	lambdaARN := strings.TrimSpace(input.RotationLambdaARN)
	if lambdaARN != "" {
		secret.RotationLambdaARN = lambdaARN
	} else {
		lambdaARN = strings.TrimSpace(secret.RotationLambdaARN)
	}
	if lambdaARN == "" {
		return RotateSecretOutput{}, ErrInvalidParameter
	}
	token := strings.TrimSpace(input.ClientRequestToken)
	if token != "" {
		if existing, ok := s.rotateTokens[token]; ok {
			if existing.ARN != secret.ARN || existing.RotationLambdaARN != lambdaARN || existing.AutomaticallyAfterDays != input.AutomaticallyAfterDays || existing.RotateImmediately != input.RotateImmediately {
				return RotateSecretOutput{}, ErrInvalidParameter
			}
			return RotateSecretOutput{
				ARN:       secret.ARN,
				Name:      secret.Name,
				VersionID: existing.VersionID,
			}, nil
		}
	}
	secret.RotationEnabled = true
	if input.AutomaticallyAfterDays > 0 {
		secret.AutomaticallyAfterDays = input.AutomaticallyAfterDays
		next := now.Add(time.Duration(input.AutomaticallyAfterDays) * 24 * time.Hour)
		secret.NextRotationDate = &next
	}

	out := RotateSecretOutput{ARN: secret.ARN, Name: secret.Name}
	if input.RotateImmediately {
		current, err := selectVersionLocked(secret, "", "AWSCURRENT")
		if err != nil {
			return RotateSecretOutput{}, err
		}
		version, err := s.putSecretValueLocked(secret, putValueInput{
			ClientRequestToken: token,
			SecretString:       current.SecretString,
			SecretBinary:       current.SecretBinary,
			VersionStages:      nil,
		})
		if err != nil {
			return RotateSecretOutput{}, err
		}
		out.VersionID = version.VersionID
		secret.LastRotatedDate = cloneTimePtr(&now)
		if secret.AutomaticallyAfterDays > 0 {
			next := now.Add(time.Duration(secret.AutomaticallyAfterDays) * 24 * time.Hour)
			secret.NextRotationDate = &next
		}
	}
	secret.LastChangedDate = now
	if token != "" {
		s.rotateTokens[token] = rotateTokenRecord{
			ARN:                    secret.ARN,
			RotationLambdaARN:      lambdaARN,
			AutomaticallyAfterDays: input.AutomaticallyAfterDays,
			RotateImmediately:      input.RotateImmediately,
			VersionID:              out.VersionID,
		}
	}
	return out, nil
}

func (s *Service) CancelRotateSecret(secretID string) (CancelRotateSecretOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return CancelRotateSecretOutput{}, err
	}
	if secret.DeletedDate != nil {
		return CancelRotateSecretOutput{}, ErrInvalidState
	}
	if !secret.RotationEnabled {
		return CancelRotateSecretOutput{}, ErrInvalidState
	}

	secret.RotationEnabled = false
	secret.NextRotationDate = nil
	secret.AutomaticallyAfterDays = 0
	secret.LastChangedDate = time.Now().UTC()
	return CancelRotateSecretOutput{
		ARN:       secret.ARN,
		Name:      secret.Name,
		VersionID: currentVersionIDLocked(secret),
	}, nil
}

func (s *Service) ReplicateSecretToRegions(input ReplicateSecretToRegionsInput) (ReplicateSecretToRegionsOutput, error) {
	secretID := strings.TrimSpace(input.SecretID)
	if secretID == "" || len(input.AddReplicaRegions) == 0 {
		return ReplicateSecretToRegionsOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return ReplicateSecretToRegionsOutput{}, err
	}
	if secret.DeletedDate != nil {
		return ReplicateSecretToRegionsOutput{}, ErrInvalidState
	}
	if secret.ReplicationStatus == nil {
		secret.ReplicationStatus = map[string]*ReplicationStatus{}
	}

	now := time.Now().UTC()
	for _, replica := range input.AddReplicaRegions {
		region := strings.TrimSpace(replica.Region)
		if region == "" || strings.EqualFold(region, secret.PrimaryRegion) {
			return ReplicateSecretToRegionsOutput{}, ErrInvalidParameter
		}
		if _, exists := secret.ReplicationStatus[region]; exists && !input.ForceOverwriteReplicaSecret {
			return ReplicateSecretToRegionsOutput{}, ErrInvalidState
		}
		secret.ReplicationStatus[region] = &ReplicationStatus{
			Region:           region,
			KmsKeyID:         strings.TrimSpace(replica.KmsKeyID),
			Status:           "InSync",
			StatusMessage:    "replication configured",
			LastAccessedDate: cloneTimePtr(&now),
		}
	}
	secret.LastChangedDate = now
	return ReplicateSecretToRegionsOutput{
		ARN:               secret.ARN,
		ReplicationStatus: cloneReplicationStatusList(secret.ReplicationStatus),
	}, nil
}

func (s *Service) RemoveRegionsFromReplication(input RemoveRegionsFromReplicationInput) (RemoveRegionsFromReplicationOutput, error) {
	secretID := strings.TrimSpace(input.SecretID)
	regions := dedupeStrings(input.RemoveReplicaRegions)
	if secretID == "" || len(regions) == 0 {
		return RemoveRegionsFromReplicationOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return RemoveRegionsFromReplicationOutput{}, err
	}
	if secret.DeletedDate != nil {
		return RemoveRegionsFromReplicationOutput{}, ErrInvalidState
	}

	for _, region := range regions {
		if _, exists := secret.ReplicationStatus[region]; !exists {
			return RemoveRegionsFromReplicationOutput{}, ErrNotFound
		}
		delete(secret.ReplicationStatus, region)
	}
	secret.LastChangedDate = time.Now().UTC()
	return RemoveRegionsFromReplicationOutput{
		ARN:               secret.ARN,
		ReplicationStatus: cloneReplicationStatusList(secret.ReplicationStatus),
	}, nil
}

func (s *Service) StopReplicationToReplica(input StopReplicationToReplicaInput) (StopReplicationToReplicaOutput, error) {
	secretID := strings.TrimSpace(input.SecretID)
	if secretID == "" {
		return StopReplicationToReplicaOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return StopReplicationToReplicaOutput{}, err
	}
	if secret.DeletedDate != nil {
		return StopReplicationToReplicaOutput{}, ErrInvalidState
	}
	if len(secret.ReplicationStatus) == 0 {
		return StopReplicationToReplicaOutput{}, ErrInvalidState
	}

	region := strings.TrimSpace(input.ReplicaRegion)
	if region == "" {
		if len(secret.ReplicationStatus) != 1 {
			return StopReplicationToReplicaOutput{}, ErrInvalidParameter
		}
		for key := range secret.ReplicationStatus {
			region = key
		}
	}
	if _, exists := secret.ReplicationStatus[region]; !exists {
		return StopReplicationToReplicaOutput{}, ErrNotFound
	}
	delete(secret.ReplicationStatus, region)
	secret.LastChangedDate = time.Now().UTC()
	return StopReplicationToReplicaOutput{ARN: secret.ARN}, nil
}

func (s *Service) PutResourcePolicy(input PutResourcePolicyInput) (PutResourcePolicyOutput, error) {
	secretID := strings.TrimSpace(input.SecretID)
	policy := strings.TrimSpace(input.ResourcePolicy)
	if secretID == "" || policy == "" {
		return PutResourcePolicyOutput{}, ErrInvalidParameter
	}
	if len(policy) > defaultMaxPolicyBytes {
		return PutResourcePolicyOutput{}, ErrLimitExceeded
	}
	document, err := parsePolicyDocument(policy)
	if err != nil {
		return PutResourcePolicyOutput{}, ErrInvalidParameter
	}
	if !policyHasStatement(document) {
		return PutResourcePolicyOutput{}, ErrInvalidParameter
	}
	if input.BlockPublicPolicy && policyHasWildcardPrincipal(document) {
		return PutResourcePolicyOutput{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return PutResourcePolicyOutput{}, err
	}
	if secret.DeletedDate != nil {
		return PutResourcePolicyOutput{}, ErrInvalidState
	}
	secret.ResourcePolicy = policy
	secret.LastChangedDate = time.Now().UTC()
	return PutResourcePolicyOutput{ARN: secret.ARN, Name: secret.Name}, nil
}

func (s *Service) GetResourcePolicy(secretID string) (GetResourcePolicyOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return GetResourcePolicyOutput{}, err
	}
	if secret.DeletedDate != nil {
		return GetResourcePolicyOutput{}, ErrInvalidState
	}
	if strings.TrimSpace(secret.ResourcePolicy) == "" {
		return GetResourcePolicyOutput{}, ErrNotFound
	}
	return GetResourcePolicyOutput{
		ARN:            secret.ARN,
		Name:           secret.Name,
		ResourcePolicy: secret.ResourcePolicy,
	}, nil
}

func (s *Service) DeleteResourcePolicy(secretID string) (DeleteResourcePolicyOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return DeleteResourcePolicyOutput{}, err
	}
	if secret.DeletedDate != nil {
		return DeleteResourcePolicyOutput{}, ErrInvalidState
	}
	if strings.TrimSpace(secret.ResourcePolicy) == "" {
		return DeleteResourcePolicyOutput{}, ErrNotFound
	}
	secret.ResourcePolicy = ""
	secret.LastChangedDate = time.Now().UTC()
	return DeleteResourcePolicyOutput{ARN: secret.ARN, Name: secret.Name}, nil
}

func (s *Service) ValidateResourcePolicy(input ValidateResourcePolicyInput) (ValidateResourcePolicyOutput, error) {
	policy := strings.TrimSpace(input.ResourcePolicy)
	secretID := strings.TrimSpace(input.SecretID)
	if secretID == "" && policy == "" {
		return ValidateResourcePolicyOutput{}, ErrInvalidParameter
	}
	if secretID != "" {
		s.mu.Lock()
		secret, err := s.lookupSecretLocked(secretID)
		if err != nil {
			s.mu.Unlock()
			return ValidateResourcePolicyOutput{}, err
		}
		if secret.DeletedDate != nil {
			s.mu.Unlock()
			return ValidateResourcePolicyOutput{}, ErrInvalidState
		}
		if policy == "" {
			policy = strings.TrimSpace(secret.ResourcePolicy)
		}
		s.mu.Unlock()
	}

	out := ValidateResourcePolicyOutput{
		PolicyValidationPassed: true,
		ValidationErrors:       []ValidateResourcePolicyError{},
	}
	if policy == "" {
		out.PolicyValidationPassed = false
		out.ValidationErrors = append(out.ValidationErrors, ValidateResourcePolicyError{
			CheckName:    "NO_POLICY",
			ErrorMessage: "resource policy is empty",
		})
		return out, nil
	}
	if len(policy) > defaultMaxPolicyBytes {
		out.PolicyValidationPassed = false
		out.ValidationErrors = append(out.ValidationErrors, ValidateResourcePolicyError{
			CheckName:    "POLICY_LENGTH_EXCEEDED",
			ErrorMessage: "resource policy size exceeds supported limit",
		})
		return out, nil
	}

	document, err := parsePolicyDocument(policy)
	if err != nil {
		out.PolicyValidationPassed = false
		out.ValidationErrors = append(out.ValidationErrors, ValidateResourcePolicyError{
			CheckName:    "SYNTAX_CHECK_FAILED",
			ErrorMessage: err.Error(),
		})
		return out, nil
	}
	if !policyHasStatement(document) {
		out.PolicyValidationPassed = false
		out.ValidationErrors = append(out.ValidationErrors, ValidateResourcePolicyError{
			CheckName:    "MISSING_STATEMENT",
			ErrorMessage: "policy must include at least one statement",
		})
	}
	if input.BlockPublicPolicy && policyHasWildcardPrincipal(document) {
		out.PolicyValidationPassed = false
		out.ValidationErrors = append(out.ValidationErrors, ValidateResourcePolicyError{
			CheckName:    "PUBLIC_POLICY_CHECK",
			ErrorMessage: "policy allows wildcard principal",
		})
	}
	return out, nil
}

func (s *Service) TagResource(input TagResourceInput) error {
	secretID := strings.TrimSpace(input.SecretID)
	tags := cloneTags(input.Tags)
	if secretID == "" || len(tags) == 0 {
		return ErrInvalidParameter
	}
	if err := validateTagMap(tags); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return err
	}
	if secret.DeletedDate != nil {
		return ErrInvalidState
	}

	count := len(secret.Tags)
	for key := range tags {
		if _, exists := secret.Tags[key]; !exists {
			count++
		}
	}
	if count > defaultMaxTags {
		return ErrLimitExceeded
	}
	for key, value := range tags {
		secret.Tags[key] = value
	}
	secret.LastChangedDate = time.Now().UTC()
	return nil
}

func (s *Service) UntagResource(input UntagResourceInput) error {
	secretID := strings.TrimSpace(input.SecretID)
	tagKeys := dedupeStrings(input.TagKeys)
	if secretID == "" || len(tagKeys) == 0 {
		return ErrInvalidParameter
	}
	for _, key := range tagKeys {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(key)), "aws:") {
			return ErrInvalidParameter
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, err := s.lookupSecretLocked(secretID)
	if err != nil {
		return err
	}
	if secret.DeletedDate != nil {
		return ErrInvalidState
	}

	for _, key := range tagKeys {
		delete(secret.Tags, key)
	}
	secret.LastChangedDate = time.Now().UTC()
	return nil
}

type putValueInput struct {
	ClientRequestToken string
	SecretString       string
	SecretBinary       string
	VersionStages      []string
}

func (s *Service) putSecretValueLocked(secret *Secret, input putValueInput) (*SecretVersion, error) {
	token := strings.TrimSpace(input.ClientRequestToken)
	if token == "" {
		token = s.nextVersionID()
	}

	if existing, ok := secret.Versions[token]; ok {
		if existing.SecretString != input.SecretString || existing.SecretBinary != input.SecretBinary {
			return nil, ErrInvalidParameter
		}
		return existing, nil
	}

	now := time.Now().UTC()
	version := &SecretVersion{
		VersionID:    token,
		SecretString: input.SecretString,
		SecretBinary: input.SecretBinary,
		CreatedDate:  now,
		Stages:       []string{},
	}
	secret.Versions[token] = version

	if len(input.VersionStages) > 0 {
		for _, stage := range dedupeStrings(input.VersionStages) {
			if stage == "" {
				continue
			}
			for _, current := range secret.Versions {
				current.Stages = removeString(current.Stages, stage)
			}
			version.Stages = append(version.Stages, stage)
		}
		if !sliceContains(version.Stages, "AWSCURRENT") {
			// If caller did not provide AWSCURRENT, keep explicit stages only.
		}
		sort.Strings(version.Stages)
	} else {
		previousCurrentID := currentVersionIDLocked(secret)
		if previousCurrentID != "" {
			if previous, ok := secret.Versions[previousCurrentID]; ok {
				previous.Stages = removeString(previous.Stages, "AWSCURRENT")
				if !sliceContains(previous.Stages, "AWSPREVIOUS") {
					previous.Stages = append(previous.Stages, "AWSPREVIOUS")
					sort.Strings(previous.Stages)
				}
			}
		}
		version.Stages = []string{"AWSCURRENT"}
	}

	secret.LastChangedDate = now
	return version, nil
}

func selectVersionLocked(secret *Secret, versionID, versionStage string) (*SecretVersion, error) {
	if versionID != "" {
		version, ok := secret.Versions[versionID]
		if !ok {
			return nil, ErrNotFound
		}
		if versionStage != "" && !sliceContains(version.Stages, versionStage) {
			return nil, ErrInvalidParameter
		}
		return version, nil
	}
	stage := versionStage
	if stage == "" {
		stage = "AWSCURRENT"
	}
	for _, version := range secret.Versions {
		if sliceContains(version.Stages, stage) {
			return version, nil
		}
	}
	return nil, ErrNotFound
}

func currentVersionIDLocked(secret *Secret) string {
	for versionID, version := range secret.Versions {
		if sliceContains(version.Stages, "AWSCURRENT") {
			return versionID
		}
	}
	return ""
}

func (s *Service) lookupSecretLocked(secretID string) (*Secret, error) {
	id := strings.TrimSpace(secretID)
	if id == "" {
		return nil, ErrInvalidParameter
	}
	if secret, ok := s.secrets[id]; ok {
		return secret, nil
	}
	if arn, ok := s.secretByName[id]; ok {
		if secret, ok := s.secrets[arn]; ok {
			return secret, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) nextSecretARNLocked(name string) string {
	s.seq++
	suffix := fmt.Sprintf("%06d", s.seq)
	return fmt.Sprintf("arn:aws:secretsmanager:%s:%s:secret:%s-%s", DefaultRegion, DefaultAccountID, sanitizeName(name), suffix)
}

func (s *Service) nextVersionID() string {
	// Lightweight unique token for local emulation flows.
	n, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	return fmt.Sprintf("%016x", n.Uint64())
}

func (s *Service) checkThrottleLocked() error {
	now := time.Now().UTC()
	cutoff := now.Add(-defaultThrottleWindow)
	kept := s.calls[:0]
	for _, ts := range s.calls {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	s.calls = kept
	if len(s.calls) >= defaultThrottleLimit {
		return ErrThrottling
	}
	s.calls = append(s.calls, now)
	return nil
}

func cloneSecret(secret *Secret) Secret {
	out := Secret{
		ARN:                    secret.ARN,
		Name:                   secret.Name,
		Description:            secret.Description,
		KmsKeyID:               secret.KmsKeyID,
		OwningService:          secret.OwningService,
		PrimaryRegion:          secret.PrimaryRegion,
		CreatedDate:            secret.CreatedDate,
		LastChangedDate:        secret.LastChangedDate,
		LastAccessedDate:       cloneTimePtr(secret.LastAccessedDate),
		LastRotatedDate:        cloneTimePtr(secret.LastRotatedDate),
		NextRotationDate:       cloneTimePtr(secret.NextRotationDate),
		DeletedDate:            cloneTimePtr(secret.DeletedDate),
		RotationEnabled:        secret.RotationEnabled,
		RotationLambdaARN:      secret.RotationLambdaARN,
		AutomaticallyAfterDays: secret.AutomaticallyAfterDays,
		ResourcePolicy:         secret.ResourcePolicy,
		Tags:                   cloneTags(secret.Tags),
		ReplicationStatus:      cloneReplicationStatusMap(secret.ReplicationStatus),
		Versions:               make(map[string]*SecretVersion, len(secret.Versions)),
	}
	for versionID, version := range secret.Versions {
		out.Versions[versionID] = &SecretVersion{
			VersionID:    version.VersionID,
			SecretString: version.SecretString,
			SecretBinary: version.SecretBinary,
			CreatedDate:  version.CreatedDate,
			Stages:       append([]string(nil), version.Stages...),
		}
	}
	return out
}

func cloneTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(tags))
	for key, value := range tags {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(value)
	}
	return out
}

func validateTagMap(tags map[string]string) error {
	for key, value := range tags {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return ErrInvalidParameter
		}
		if strings.HasPrefix(strings.ToLower(trimmedKey), "aws:") {
			return ErrInvalidParameter
		}
		if len(trimmedKey) > 128 || len(value) > 256 {
			return ErrInvalidParameter
		}
	}
	return nil
}

func stringMapEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func cloneReplicationStatusMap(input map[string]*ReplicationStatus) map[string]*ReplicationStatus {
	if len(input) == 0 {
		return map[string]*ReplicationStatus{}
	}
	out := make(map[string]*ReplicationStatus, len(input))
	for region, status := range input {
		if status == nil {
			continue
		}
		out[region] = &ReplicationStatus{
			Region:           status.Region,
			KmsKeyID:         status.KmsKeyID,
			Status:           status.Status,
			StatusMessage:    status.StatusMessage,
			LastAccessedDate: cloneTimePtr(status.LastAccessedDate),
		}
	}
	return out
}

func cloneReplicationStatusList(input map[string]*ReplicationStatus) []ReplicationStatus {
	if len(input) == 0 {
		return []ReplicationStatus{}
	}
	regions := make([]string, 0, len(input))
	for region := range input {
		regions = append(regions, region)
	}
	sort.Strings(regions)
	out := make([]ReplicationStatus, 0, len(regions))
	for _, region := range regions {
		status := input[region]
		if status == nil {
			continue
		}
		out = append(out, ReplicationStatus{
			Region:           status.Region,
			KmsKeyID:         status.KmsKeyID,
			Status:           status.Status,
			StatusMessage:    status.StatusMessage,
			LastAccessedDate: cloneTimePtr(status.LastAccessedDate),
		})
	}
	return out
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func normalizeSecretFilters(filters []SecretFilter) ([]SecretFilter, error) {
	if len(filters) == 0 {
		return nil, nil
	}
	if len(filters) > defaultMaxFilters {
		return nil, ErrLimitExceeded
	}
	out := make([]SecretFilter, 0, len(filters))
	for _, filter := range filters {
		key := strings.ToLower(strings.TrimSpace(filter.Key))
		if key == "" {
			return nil, ErrInvalidParameter
		}
		switch key {
		case "all", "name", "description", "tag-key", "tag-value", "secret-id", "primary-region", "owning-service":
		default:
			return nil, ErrInvalidParameter
		}
		values := dedupeStrings(filter.Values)
		if len(values) == 0 {
			return nil, ErrInvalidParameter
		}
		if len(values) > defaultMaxFilterValues {
			return nil, ErrLimitExceeded
		}
		out = append(out, SecretFilter{Key: key, Values: values})
	}
	return out, nil
}

func matchesSecretFilters(secret *Secret, filters []SecretFilter) bool {
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		if !matchesSecretFilter(secret, filter) {
			return false
		}
	}
	return true
}

func matchesSecretFilter(secret *Secret, filter SecretFilter) bool {
	for _, value := range filter.Values {
		if secretMatchesFilterValue(secret, filter.Key, strings.ToLower(strings.TrimSpace(value))) {
			return true
		}
	}
	return false
}

func secretMatchesFilterValue(secret *Secret, key, value string) bool {
	if value == "" {
		return false
	}
	name := strings.ToLower(secret.Name)
	description := strings.ToLower(secret.Description)
	arn := strings.ToLower(secret.ARN)
	primaryRegion := strings.ToLower(secret.PrimaryRegion)
	owningService := strings.ToLower(secret.OwningService)

	switch key {
	case "name":
		return strings.Contains(name, value)
	case "description":
		return strings.Contains(description, value)
	case "secret-id":
		return strings.Contains(arn, value) || strings.Contains(name, value)
	case "primary-region":
		return strings.EqualFold(primaryRegion, value)
	case "owning-service":
		return strings.Contains(owningService, value)
	case "tag-key":
		for tagKey := range secret.Tags {
			if strings.Contains(strings.ToLower(tagKey), value) {
				return true
			}
		}
	case "tag-value":
		for _, tagValue := range secret.Tags {
			if strings.Contains(strings.ToLower(tagValue), value) {
				return true
			}
		}
	case "all":
		if strings.Contains(name, value) || strings.Contains(description, value) || strings.Contains(arn, value) {
			return true
		}
		for tagKey, tagValue := range secret.Tags {
			if strings.Contains(strings.ToLower(tagKey), value) || strings.Contains(strings.ToLower(tagValue), value) {
				return true
			}
		}
	}
	return false
}

func parsePolicyDocument(policy string) (map[string]any, error) {
	document := map[string]any{}
	if err := json.Unmarshal([]byte(policy), &document); err != nil {
		return nil, err
	}
	return document, nil
}

func policyHasStatement(document map[string]any) bool {
	statement, ok := document["Statement"]
	if !ok {
		return false
	}
	switch typed := statement.(type) {
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return false
	}
}

func policyHasWildcardPrincipal(document map[string]any) bool {
	statementRaw, ok := document["Statement"]
	if !ok {
		return false
	}

	entries := []map[string]any{}
	switch typed := statementRaw.(type) {
	case map[string]any:
		entries = append(entries, typed)
	case []any:
		for _, entry := range typed {
			if statement, ok := entry.(map[string]any); ok {
				entries = append(entries, statement)
			}
		}
	}

	for _, statement := range entries {
		principal, exists := statement["Principal"]
		if !exists {
			continue
		}
		switch typed := principal.(type) {
		case string:
			if strings.TrimSpace(typed) == "*" {
				return true
			}
		case map[string]any:
			for _, value := range typed {
				switch v := value.(type) {
				case string:
					if strings.TrimSpace(v) == "*" {
						return true
					}
				case []any:
					for _, item := range v {
						text, ok := item.(string)
						if ok && strings.TrimSpace(text) == "*" {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func removeString(values []string, target string) []string {
	if len(values) == 0 {
		return values
	}
	out := values[:0]
	for _, value := range values {
		if value == target {
			continue
		}
		out = append(out, value)
	}
	return out
}

func sliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "secret"
	}
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '/' {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('-')
	}
	return strings.Trim(b.String(), "-")
}

func parseNextTokenOffset(nextToken string) (int, error) {
	trimmed := strings.TrimSpace(nextToken)
	if trimmed == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value < 0 {
		return 0, ErrInvalidParameter
	}
	return value, nil
}

func randomRune(chars string) (rune, error) {
	if chars == "" {
		return 0, ErrInvalidParameter
	}
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
	if err != nil {
		return 0, err
	}
	return rune(chars[index.Int64()]), nil
}

func shuffleRunes(values []rune) error {
	for i := len(values) - 1; i > 0; i-- {
		jv, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(jv.Int64())
		values[i], values[j] = values[j], values[i]
	}
	return nil
}
