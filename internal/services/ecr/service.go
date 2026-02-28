package ecr

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidParameter         = errors.New("invalid parameter")
	ErrRepositoryAlreadyExists  = errors.New("repository already exists")
	ErrRepositoryNotFound       = errors.New("repository not found")
	ErrRepositoryPolicyNotFound = errors.New("repository policy not found")
	ErrRegistryPolicyNotFound   = errors.New("registry policy not found")
	ErrUploadNotFound           = errors.New("upload not found")
	ErrLayerNotFound            = errors.New("layer not found")
	ErrImageTagImmutable        = errors.New("image tag immutable")
	ErrImageDigestDoesNotMatch  = errors.New("image digest does not match")
	ErrImageNotFound            = errors.New("image not found")
	ErrScanNotFound             = errors.New("scan not found")
	ErrLifecyclePolicyNotFound  = errors.New("lifecycle policy not found")
	ErrLifecyclePreviewNotFound = errors.New("lifecycle preview not found")
	ErrPullThroughRuleExists    = errors.New("pull through cache rule already exists")
	ErrPullThroughRuleNotFound  = errors.New("pull through cache rule not found")
	ErrTemplateAlreadyExists    = errors.New("repository creation template already exists")
	ErrTemplateNotFound         = errors.New("repository creation template not found")
	ErrSigningConfigNotFound    = errors.New("signing configuration not found")
)

const (
	DefaultRegion        = "us-east-1"
	DefaultAccountID     = "123456789012"
	defaultLayerPartSize = int64(20 * 1024 * 1024)
)

var (
	repositoryNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:[._/-][a-z0-9]+)*$`)
	digestPattern         = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
)

type Repository struct {
	RepositoryName          string
	RepositoryArn           string
	RepositoryURI           string
	RegistryID              string
	CreatedAt               time.Time
	ImageTagMutability      string
	ImageScanningScanOnPush bool
	EncryptionType          string
	KMSKey                  string
	PolicyText              string
	Tags                    map[string]string
}

type AuthorizationData struct {
	AuthorizationToken string
	ExpiresAt          time.Time
	ProxyEndpoint      string
}

type RepositoryScanningConfiguration struct {
	RepositoryName string
	RepositoryArn  string
	ScanFrequency  string
	ScanOnPush     bool
}

type RepositoryScanningConfigurationFailure struct {
	RepositoryName string
	FailureCode    string
	FailureReason  string
}

type LayerUpload struct {
	UploadID       string
	RepositoryName string
	Data           []byte
	NextByte       int64
	CreatedAt      time.Time
}

type Layer struct {
	LayerDigest       string
	LayerAvailability string
	LayerSize         int64
	MediaType         string
	Data              []byte
	CreatedAt         time.Time
}

type LayerFailure struct {
	LayerDigest   string
	FailureCode   string
	FailureReason string
}

type ImageIdentifier struct {
	ImageDigest string
	ImageTag    string
}

type Image struct {
	ImageID                ImageIdentifier
	ImageManifest          string
	ImageManifestMediaType string
	RegistryID             string
	RepositoryName         string
}

type ImageFailure struct {
	ImageID       ImageIdentifier
	FailureCode   string
	FailureReason string
}

type ImageDetail struct {
	ImageDigest            string
	ImageTags              []string
	ImageManifestMediaType string
	ImageSizeInBytes       int64
	ImagePushedAt          time.Time
	RegistryID             string
	RepositoryName         string
}

type LifecyclePolicy struct {
	RegistryID          string
	RepositoryName      string
	LifecyclePolicyText string
	LastEvaluatedAt     time.Time
}

type LifecyclePolicyPreviewResult struct {
	ImageDigest         string
	ImageTags           []string
	ImagePushedAt       time.Time
	AppliedRulePriority int32
	ActionType          string
}

type LifecyclePolicyPreviewSummary struct {
	ExpiringImageTotalCount int32
}

type ScanningRepositoryFilter struct {
	Filter     string
	FilterType string
}

type RegistryScanningRule struct {
	RepositoryFilters []ScanningRepositoryFilter
	ScanFrequency     string
}

type RegistryScanningConfiguration struct {
	Rules    []RegistryScanningRule
	ScanType string
}

type RepositoryFilter struct {
	Filter     string
	FilterType string
}

type ReplicationDestination struct {
	Region     string
	RegistryID string
}

type ReplicationRule struct {
	Destinations      []ReplicationDestination
	RepositoryFilters []RepositoryFilter
}

type ReplicationConfiguration struct {
	Rules []ReplicationRule
}

type ImageReplicationStatus struct {
	FailureCode string
	Region      string
	RegistryID  string
	Status      string
}

type PullThroughCacheRule struct {
	CreatedAt           time.Time
	CredentialArn       string
	ECRRepositoryPrefix string
	RegistryID          string
	UpdatedAt           time.Time
	UpstreamRegistry    string
	UpstreamRegistryURL string
}

type AccountSetting struct {
	Name  string
	Value string
}

type RepositoryCreationTemplate struct {
	AppliedFor         []string
	CreatedAt          time.Time
	CustomRoleArn      string
	Description        string
	EncryptionType     string
	KMSKey             string
	ImageTagMutability string
	LifecyclePolicy    string
	Prefix             string
	RepositoryPolicy   string
	ResourceTags       map[string]string
	UpdatedAt          time.Time
}

type RepositoryCreationTemplateInput struct {
	Prefix             string
	AppliedFor         []string
	AppliedForSet      bool
	CustomRoleArn      *string
	Description        *string
	EncryptionType     *string
	KMSKey             *string
	ImageTagMutability *string
	LifecyclePolicy    *string
	RepositoryPolicy   *string
	ResourceTags       map[string]string
	ResourceTagsSet    bool
}

type SigningRepositoryFilter struct {
	Filter     string
	FilterType string
}

type SigningRule struct {
	SigningProfileArn string
	RepositoryFilters []SigningRepositoryFilter
}

type SigningConfiguration struct {
	Rules []SigningRule
}

type ImageSigningStatus struct {
	FailureCode       string
	FailureReason     string
	SigningProfileArn string
	Status            string
}

type ImageScanAttribute struct {
	Key   string
	Value string
}

type ImageScanFinding struct {
	Name        string
	Description string
	Severity    string
	URI         string
	Attributes  []ImageScanAttribute
}

type ImageScanFindings struct {
	FindingSeverityCounts        map[string]int32
	Findings                     []ImageScanFinding
	ImageScanCompletedAt         time.Time
	VulnerabilitySourceUpdatedAt time.Time
}

type ImageScanStatus struct {
	Description string
	Status      string
}

type PullTimeUpdateExclusion struct {
	CreatedAt    time.Time
	PrincipalArn string
}

type ImageReferrer struct {
	Annotations  map[string]string
	ArtifactType string
	Digest       string
	MediaType    string
	Size         int64
	Status       string
}

type Service struct {
	mu                   sync.Mutex
	seq                  uint64
	repositoriesByName   map[string]*Repository
	repositoriesByARN    map[string]*Repository
	uploads              map[string]*LayerUpload
	layersByRepository   map[string]map[string]*Layer
	imagesByRepository   map[string]map[string]*storedImage
	imageTagsByRepo      map[string]map[string]string
	lifecyclePolicies    map[string]*lifecyclePolicyState
	lifecyclePreviews    map[string]*lifecyclePreviewState
	registryPolicyText   string
	registryScanning     RegistryScanningConfiguration
	replicationConfig    ReplicationConfiguration
	accountSettings      map[string]string
	pullThroughRules     map[string]*PullThroughCacheRule
	creationTemplates    map[string]*RepositoryCreationTemplate
	signingConfiguration *SigningConfiguration
	scanStateByRepo      map[string]map[string]*imageScanState
	pullTimeExclusions   map[string]*PullTimeUpdateExclusion
}

type storedImage struct {
	Digest            string
	Manifest          string
	ManifestMediaType string
	ArtifactType      string
	SubjectDigest     string
	ImageStatus       string
	Tags              map[string]struct{}
	PushedAt          time.Time
}

type lifecyclePolicyState struct {
	Text            string
	LastEvaluatedAt time.Time
}

type lifecyclePreviewState struct {
	PolicyText string
	Status     string
	Results    []LifecyclePolicyPreviewResult
	UpdatedAt  time.Time
}

type imageScanState struct {
	Findings    ImageScanFindings
	Status      ImageScanStatus
	LastScanned time.Time
}

func NewService() *Service {
	return &Service{
		repositoriesByName: map[string]*Repository{},
		repositoriesByARN:  map[string]*Repository{},
		uploads:            map[string]*LayerUpload{},
		layersByRepository: map[string]map[string]*Layer{},
		imagesByRepository: map[string]map[string]*storedImage{},
		imageTagsByRepo:    map[string]map[string]string{},
		lifecyclePolicies:  map[string]*lifecyclePolicyState{},
		lifecyclePreviews:  map[string]*lifecyclePreviewState{},
		registryScanning: RegistryScanningConfiguration{
			ScanType: "BASIC",
			Rules:    []RegistryScanningRule{},
		},
		replicationConfig: ReplicationConfiguration{
			Rules: []ReplicationRule{},
		},
		accountSettings:      map[string]string{},
		pullThroughRules:     map[string]*PullThroughCacheRule{},
		creationTemplates:    map[string]*RepositoryCreationTemplate{},
		signingConfiguration: nil,
		scanStateByRepo:      map[string]map[string]*imageScanState{},
		pullTimeExclusions:   map[string]*PullTimeUpdateExclusion{},
	}
}

func (s *Service) GetAuthorizationToken(registryIDs []string) ([]AuthorizationData, error) {
	normalized, err := normalizeRegistryIDs(registryIDs)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	token := base64.StdEncoding.EncodeToString([]byte("AWS:stackyard-token"))
	out := make([]AuthorizationData, 0, len(normalized))
	for _, registryID := range normalized {
		out = append(out, AuthorizationData{
			AuthorizationToken: token,
			ExpiresAt:          now.Add(12 * time.Hour),
			ProxyEndpoint:      fmt.Sprintf("https://%s.dkr.ecr.%s.amazonaws.com", registryID, DefaultRegion),
		})
	}
	return out, nil
}

func (s *Service) CreateRepository(repositoryName, imageTagMutability string, imageScanningScanOnPush *bool, encryptionType, kmsKey string, tags map[string]string) (Repository, error) {
	repositoryName = strings.TrimSpace(repositoryName)
	if !repositoryNamePattern.MatchString(repositoryName) {
		return Repository{}, ErrInvalidParameter
	}

	normalizedMutability, err := normalizeImageTagMutability(imageTagMutability)
	if err != nil {
		return Repository{}, err
	}

	normalizedEncryptionType, err := normalizeEncryptionType(encryptionType)
	if err != nil {
		return Repository{}, err
	}

	scanOnPush := false
	if imageScanningScanOnPush != nil {
		scanOnPush = *imageScanningScanOnPush
	}

	cleanTags, err := normalizeTags(tags)
	if err != nil {
		return Repository{}, err
	}

	now := time.Now().UTC()
	repository := &Repository{
		RepositoryName:          repositoryName,
		RepositoryArn:           repositoryARN(repositoryName),
		RepositoryURI:           repositoryURI(repositoryName),
		RegistryID:              DefaultAccountID,
		CreatedAt:               now,
		ImageTagMutability:      normalizedMutability,
		ImageScanningScanOnPush: scanOnPush,
		EncryptionType:          normalizedEncryptionType,
		KMSKey:                  strings.TrimSpace(kmsKey),
		Tags:                    cleanTags,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.repositoriesByName[repositoryName]; exists {
		return Repository{}, ErrRepositoryAlreadyExists
	}
	s.repositoriesByName[repositoryName] = repository
	s.repositoriesByARN[repository.RepositoryArn] = repository
	s.layersByRepository[repositoryName] = map[string]*Layer{}
	s.imagesByRepository[repositoryName] = map[string]*storedImage{}
	s.imageTagsByRepo[repositoryName] = map[string]string{}

	return cloneRepository(repository), nil
}

func (s *Service) DescribeRepositories(repositoryNames []string, nextToken string, maxResults int32) ([]Repository, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(repositoryNames) > 0 {
		out := make([]Repository, 0, len(repositoryNames))
		for _, name := range repositoryNames {
			repository, err := s.repositoryForNameLocked(name)
			if err != nil {
				return nil, "", err
			}
			out = append(out, cloneRepository(repository))
		}
		return out, "", nil
	}

	all := make([]*Repository, 0, len(s.repositoriesByName))
	for _, repository := range s.repositoriesByName {
		all = append(all, repository)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].RepositoryName < all[j].RepositoryName
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
	if limit > 1000 {
		limit = 1000
	}

	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	selected := all[offset:end]
	out := make([]Repository, 0, len(selected))
	for _, repository := range selected {
		out = append(out, cloneRepository(repository))
	}

	if end < len(all) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *Service) DeleteRepository(repositoryName string, force bool) (Repository, error) {
	_ = force

	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return Repository{}, err
	}

	delete(s.repositoriesByName, repository.RepositoryName)
	delete(s.repositoriesByARN, repository.RepositoryArn)
	delete(s.layersByRepository, repository.RepositoryName)
	delete(s.imagesByRepository, repository.RepositoryName)
	delete(s.imageTagsByRepo, repository.RepositoryName)
	delete(s.lifecyclePolicies, repository.RepositoryName)
	delete(s.lifecyclePreviews, repository.RepositoryName)
	for uploadID, upload := range s.uploads {
		if upload.RepositoryName == repository.RepositoryName {
			delete(s.uploads, uploadID)
		}
	}

	return cloneRepository(repository), nil
}

func (s *Service) ListTagsForResource(resourceArn string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForARNLocked(resourceArn)
	if err != nil {
		return nil, err
	}
	return cloneStringMap(repository.Tags), nil
}

func (s *Service) TagResource(resourceArn string, tags map[string]string) error {
	cleanTags, err := normalizeTags(tags)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForARNLocked(resourceArn)
	if err != nil {
		return err
	}
	for key, value := range cleanTags {
		repository.Tags[key] = value
	}
	return nil
}

func (s *Service) UntagResource(resourceArn string, tagKeys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForARNLocked(resourceArn)
	if err != nil {
		return err
	}
	for _, key := range tagKeys {
		delete(repository.Tags, strings.TrimSpace(key))
	}
	return nil
}

func (s *Service) SetRepositoryPolicy(repositoryName, policyText string, force bool) (Repository, string, error) {
	_ = force
	policyText = strings.TrimSpace(policyText)
	if policyText == "" {
		return Repository{}, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return Repository{}, "", err
	}
	repository.PolicyText = policyText
	return cloneRepository(repository), repository.PolicyText, nil
}

func (s *Service) GetRepositoryPolicy(repositoryName string) (Repository, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return Repository{}, "", err
	}
	if strings.TrimSpace(repository.PolicyText) == "" {
		return Repository{}, "", ErrRepositoryPolicyNotFound
	}
	return cloneRepository(repository), repository.PolicyText, nil
}

func (s *Service) DeleteRepositoryPolicy(repositoryName string) (Repository, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return Repository{}, "", err
	}
	if strings.TrimSpace(repository.PolicyText) == "" {
		return Repository{}, "", ErrRepositoryPolicyNotFound
	}
	policyText := repository.PolicyText
	repository.PolicyText = ""
	return cloneRepository(repository), policyText, nil
}

func (s *Service) PutImageTagMutability(repositoryName, imageTagMutability string) (Repository, string, error) {
	normalizedMutability, err := normalizeImageTagMutability(imageTagMutability)
	if err != nil {
		return Repository{}, "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return Repository{}, "", err
	}
	repository.ImageTagMutability = normalizedMutability
	return cloneRepository(repository), repository.ImageTagMutability, nil
}

func (s *Service) PutImageScanningConfiguration(repositoryName string, scanOnPush bool) (Repository, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return Repository{}, false, err
	}
	repository.ImageScanningScanOnPush = scanOnPush
	return cloneRepository(repository), scanOnPush, nil
}

func (s *Service) BatchGetRepositoryScanningConfiguration(repositoryNames []string) ([]RepositoryScanningConfiguration, []RepositoryScanningConfigurationFailure, error) {
	if len(repositoryNames) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	configs := make([]RepositoryScanningConfiguration, 0, len(repositoryNames))
	failures := make([]RepositoryScanningConfigurationFailure, 0)
	for _, name := range repositoryNames {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			return nil, nil, ErrInvalidParameter
		}
		repository, ok := s.repositoriesByName[trimmed]
		if !ok {
			failures = append(failures, RepositoryScanningConfigurationFailure{
				RepositoryName: trimmed,
				FailureCode:    "REPOSITORY_NOT_FOUND",
				FailureReason:  "repository not found",
			})
			continue
		}
		scanFrequency := "MANUAL"
		if repository.ImageScanningScanOnPush {
			scanFrequency = "SCAN_ON_PUSH"
		}
		configs = append(configs, RepositoryScanningConfiguration{
			RepositoryName: repository.RepositoryName,
			RepositoryArn:  repository.RepositoryArn,
			ScanFrequency:  scanFrequency,
			ScanOnPush:     repository.ImageScanningScanOnPush,
		})
	}
	return configs, failures, nil
}

func (s *Service) InitiateLayerUpload(repositoryName string) (string, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.repositoryForNameLocked(repositoryName); err != nil {
		return "", 0, err
	}

	s.seq++
	uploadID := fmt.Sprintf("upload-%d", s.seq)
	s.uploads[uploadID] = &LayerUpload{
		UploadID:       uploadID,
		RepositoryName: strings.TrimSpace(repositoryName),
		Data:           []byte{},
		NextByte:       0,
		CreatedAt:      time.Now().UTC(),
	}
	return uploadID, defaultLayerPartSize, nil
}

func (s *Service) UploadLayerPart(repositoryName, uploadID string, partFirstByte, partLastByte int64, layerPartBlob []byte) (int64, error) {
	if partFirstByte < 0 || partLastByte < partFirstByte {
		return 0, ErrInvalidParameter
	}
	if int64(len(layerPartBlob)) != partLastByte-partFirstByte+1 {
		return 0, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.repositoryForNameLocked(repositoryName); err != nil {
		return 0, err
	}

	upload, ok := s.uploads[strings.TrimSpace(uploadID)]
	if !ok {
		return 0, ErrUploadNotFound
	}
	if upload.RepositoryName != strings.TrimSpace(repositoryName) {
		return 0, ErrUploadNotFound
	}
	if upload.NextByte != partFirstByte {
		return 0, ErrInvalidParameter
	}

	upload.Data = append(upload.Data, layerPartBlob...)
	upload.NextByte = partLastByte + 1
	return partLastByte, nil
}

func (s *Service) CompleteLayerUpload(repositoryName, uploadID string, layerDigests []string) (string, error) {
	if len(layerDigests) == 0 {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return "", err
	}

	upload, ok := s.uploads[strings.TrimSpace(uploadID)]
	if !ok || upload.RepositoryName != repository.RepositoryName {
		return "", ErrUploadNotFound
	}

	computedDigest := sha256Digest(upload.Data)
	matched := false
	for _, digest := range layerDigests {
		if strings.TrimSpace(digest) == computedDigest {
			matched = true
			break
		}
	}
	if !matched {
		return "", ErrInvalidParameter
	}

	layers := s.layersByRepository[repository.RepositoryName]
	if layers == nil {
		layers = map[string]*Layer{}
		s.layersByRepository[repository.RepositoryName] = layers
	}
	if _, exists := layers[computedDigest]; !exists {
		layers[computedDigest] = &Layer{
			LayerDigest:       computedDigest,
			LayerAvailability: "AVAILABLE",
			LayerSize:         int64(len(upload.Data)),
			MediaType:         "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Data:              append([]byte(nil), upload.Data...),
			CreatedAt:         time.Now().UTC(),
		}
	}

	delete(s.uploads, upload.UploadID)
	return computedDigest, nil
}

func (s *Service) PutImage(repositoryName, imageManifest, imageManifestMediaType, imageTag, imageDigest string) (Image, error) {
	repositoryName = strings.TrimSpace(repositoryName)
	imageManifest = strings.TrimSpace(imageManifest)
	imageManifestMediaType = strings.TrimSpace(imageManifestMediaType)
	imageTag = strings.TrimSpace(imageTag)
	imageDigest = strings.TrimSpace(imageDigest)

	if imageManifest == "" {
		return Image{}, ErrInvalidParameter
	}

	computedDigest := sha256Digest([]byte(imageManifest))
	if imageDigest != "" && imageDigest != computedDigest {
		return Image{}, ErrImageDigestDoesNotMatch
	}
	if imageDigest == "" {
		imageDigest = computedDigest
	}
	if !isDigest(imageDigest) {
		return Image{}, ErrInvalidParameter
	}
	if imageManifestMediaType == "" {
		imageManifestMediaType = "application/vnd.docker.distribution.manifest.v2+json"
	}
	artifactType, subjectDigest := parseManifestMetadata(imageManifest)

	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return Image{}, err
	}

	images := s.imagesByRepository[repository.RepositoryName]
	if images == nil {
		images = map[string]*storedImage{}
		s.imagesByRepository[repository.RepositoryName] = images
	}
	tagIndex := s.imageTagsByRepo[repository.RepositoryName]
	if tagIndex == nil {
		tagIndex = map[string]string{}
		s.imageTagsByRepo[repository.RepositoryName] = tagIndex
	}

	if imageTag != "" {
		if existingDigest, exists := tagIndex[imageTag]; exists && existingDigest != imageDigest && !isImageTagMutable(repository.ImageTagMutability) {
			return Image{}, ErrImageTagImmutable
		}
		if oldDigest, exists := tagIndex[imageTag]; exists && oldDigest != imageDigest {
			if previous := images[oldDigest]; previous != nil {
				delete(previous.Tags, imageTag)
			}
		}
		tagIndex[imageTag] = imageDigest
	}

	stored := images[imageDigest]
	if stored == nil {
		stored = &storedImage{
			Digest:      imageDigest,
			Tags:        map[string]struct{}{},
			PushedAt:    time.Now().UTC(),
			ImageStatus: "ACTIVE",
		}
		images[imageDigest] = stored
	}
	stored.Manifest = imageManifest
	stored.ManifestMediaType = imageManifestMediaType
	stored.ArtifactType = artifactType
	stored.SubjectDigest = subjectDigest
	stored.PushedAt = time.Now().UTC()
	if imageTag != "" {
		stored.Tags[imageTag] = struct{}{}
	}

	outTag := imageTag
	if outTag == "" {
		outTag = firstTag(stored.Tags)
	}

	return Image{
		ImageID:                ImageIdentifier{ImageDigest: imageDigest, ImageTag: outTag},
		ImageManifest:          stored.Manifest,
		ImageManifestMediaType: stored.ManifestMediaType,
		RegistryID:             repository.RegistryID,
		RepositoryName:         repository.RepositoryName,
	}, nil
}

func (s *Service) BatchCheckLayerAvailability(repositoryName string, layerDigests []string) ([]Layer, []LayerFailure, error) {
	if len(layerDigests) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return nil, nil, err
	}

	layersByDigest := s.layersByRepository[repository.RepositoryName]
	layers := make([]Layer, 0, len(layerDigests))
	failures := make([]LayerFailure, 0)
	for _, digest := range layerDigests {
		trimmed := strings.TrimSpace(digest)
		if trimmed == "" {
			failures = append(failures, LayerFailure{FailureCode: "MissingLayerDigest", FailureReason: "layer digest is required"})
			continue
		}
		if !isDigest(trimmed) {
			failures = append(failures, LayerFailure{LayerDigest: trimmed, FailureCode: "InvalidLayerDigest", FailureReason: "invalid layer digest"})
			continue
		}
		if layer := layersByDigest[trimmed]; layer != nil {
			layers = append(layers, cloneLayer(layer))
			continue
		}
		layers = append(layers, Layer{LayerDigest: trimmed, LayerAvailability: "UNAVAILABLE"})
	}
	return layers, failures, nil
}

func (s *Service) BatchGetImage(repositoryName string, imageIDs []ImageIdentifier, acceptedMediaTypes []string) ([]Image, []ImageFailure, error) {
	if len(imageIDs) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return nil, nil, err
	}

	imagesByDigest := s.imagesByRepository[repository.RepositoryName]
	tagIndex := s.imageTagsByRepo[repository.RepositoryName]
	images := make([]Image, 0, len(imageIDs))
	failures := make([]ImageFailure, 0)
	for _, id := range imageIDs {
		normalized := ImageIdentifier{ImageDigest: strings.TrimSpace(id.ImageDigest), ImageTag: strings.TrimSpace(id.ImageTag)}
		if normalized.ImageDigest == "" && normalized.ImageTag == "" {
			failures = append(failures, ImageFailure{ImageID: normalized, FailureCode: "MissingDigestAndTag", FailureReason: "image digest or tag is required"})
			continue
		}

		resolvedDigest := normalized.ImageDigest
		if normalized.ImageTag != "" {
			tagDigest, ok := tagIndex[normalized.ImageTag]
			if !ok {
				failures = append(failures, ImageFailure{ImageID: normalized, FailureCode: "ImageNotFound", FailureReason: "image not found"})
				continue
			}
			if resolvedDigest != "" && resolvedDigest != tagDigest {
				failures = append(failures, ImageFailure{ImageID: normalized, FailureCode: "ImageTagDoesNotMatchDigest", FailureReason: "image tag does not match digest"})
				continue
			}
			resolvedDigest = tagDigest
		}

		stored := imagesByDigest[resolvedDigest]
		if stored == nil {
			failures = append(failures, ImageFailure{ImageID: normalized, FailureCode: "ImageNotFound", FailureReason: "image not found"})
			continue
		}
		if len(acceptedMediaTypes) > 0 && !containsString(acceptedMediaTypes, stored.ManifestMediaType) {
			failures = append(failures, ImageFailure{ImageID: normalized, FailureCode: "ImageNotFound", FailureReason: "image not found"})
			continue
		}

		outTag := normalized.ImageTag
		if outTag == "" {
			outTag = firstTag(stored.Tags)
		}

		images = append(images, Image{
			ImageID: ImageIdentifier{
				ImageDigest: resolvedDigest,
				ImageTag:    outTag,
			},
			ImageManifest:          stored.Manifest,
			ImageManifestMediaType: stored.ManifestMediaType,
			RegistryID:             repository.RegistryID,
			RepositoryName:         repository.RepositoryName,
		})
	}
	return images, failures, nil
}

func (s *Service) GetDownloadURLForLayer(repositoryName, layerDigest string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return "", "", err
	}

	layerDigest = strings.TrimSpace(layerDigest)
	if !isDigest(layerDigest) {
		return "", "", ErrInvalidParameter
	}

	layersByDigest := s.layersByRepository[repository.RepositoryName]
	if layersByDigest[layerDigest] == nil {
		return "", "", ErrLayerNotFound
	}

	downloadURL := fmt.Sprintf(
		"https://%s.dkr.ecr.%s.amazonaws.com/v2/%s/blobs/%s",
		DefaultAccountID,
		DefaultRegion,
		repository.RepositoryName,
		url.PathEscape(layerDigest),
	)
	return downloadURL, layerDigest, nil
}

func (s *Service) ListImages(repositoryName, tagStatus, nextToken string, maxResults int32) ([]ImageIdentifier, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return nil, "", err
	}

	tagStatus = strings.ToUpper(strings.TrimSpace(tagStatus))
	if tagStatus == "" {
		tagStatus = "ANY"
	}
	if tagStatus != "ANY" && tagStatus != "TAGGED" && tagStatus != "UNTAGGED" {
		return nil, "", ErrInvalidParameter
	}

	identifiers := make([]ImageIdentifier, 0)
	tagIndex := s.imageTagsByRepo[repository.RepositoryName]
	imagesByDigest := s.imagesByRepository[repository.RepositoryName]
	taggedDigests := map[string]struct{}{}

	if tagStatus == "ANY" || tagStatus == "TAGGED" {
		tags := make([]string, 0, len(tagIndex))
		for tag := range tagIndex {
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		for _, tag := range tags {
			digest := tagIndex[tag]
			taggedDigests[digest] = struct{}{}
			identifiers = append(identifiers, ImageIdentifier{ImageDigest: digest, ImageTag: tag})
		}
	}

	if tagStatus == "ANY" || tagStatus == "UNTAGGED" {
		digests := make([]string, 0, len(imagesByDigest))
		for digest := range imagesByDigest {
			digests = append(digests, digest)
		}
		sort.Strings(digests)
		for _, digest := range digests {
			if _, tagged := taggedDigests[digest]; tagged {
				continue
			}
			identifiers = append(identifiers, ImageIdentifier{ImageDigest: digest})
		}
	}

	offset := 0
	nextToken = strings.TrimSpace(nextToken)
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 0 || parsed > len(identifiers) {
			return nil, "", ErrInvalidParameter
		}
		offset = parsed
	}

	limit := int(maxResults)
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	end := offset + limit
	if end > len(identifiers) {
		end = len(identifiers)
	}
	out := append([]ImageIdentifier(nil), identifiers[offset:end]...)
	if end < len(identifiers) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *Service) DescribeImages(repositoryName string, imageIDs []ImageIdentifier, tagStatus, nextToken string, maxResults int32) ([]ImageDetail, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return nil, "", err
	}

	if len(imageIDs) > 0 && (strings.TrimSpace(nextToken) != "" || maxResults > 0) {
		return nil, "", ErrInvalidParameter
	}

	imagesByDigest := s.imagesByRepository[repository.RepositoryName]
	tagIndex := s.imageTagsByRepo[repository.RepositoryName]

	if len(imageIDs) > 0 {
		out := make([]ImageDetail, 0, len(imageIDs))
		for _, imageID := range imageIDs {
			digest, err := resolveDigestFromIdentifier(imageID, imagesByDigest, tagIndex)
			if err != nil {
				return nil, "", err
			}
			stored := imagesByDigest[digest]
			if stored == nil {
				return nil, "", ErrImageNotFound
			}
			out = append(out, storedImageToDetail(repository, stored))
		}
		return out, "", nil
	}

	filterTagStatus := strings.ToUpper(strings.TrimSpace(tagStatus))
	if filterTagStatus == "" {
		filterTagStatus = "ANY"
	}
	if filterTagStatus != "ANY" && filterTagStatus != "TAGGED" && filterTagStatus != "UNTAGGED" {
		return nil, "", ErrInvalidParameter
	}

	digests := make([]string, 0, len(imagesByDigest))
	for digest := range imagesByDigest {
		digests = append(digests, digest)
	}
	sort.Strings(digests)

	all := make([]ImageDetail, 0, len(digests))
	for _, digest := range digests {
		stored := imagesByDigest[digest]
		if stored == nil {
			continue
		}
		tagged := len(stored.Tags) > 0
		if filterTagStatus == "TAGGED" && !tagged {
			continue
		}
		if filterTagStatus == "UNTAGGED" && tagged {
			continue
		}
		all = append(all, storedImageToDetail(repository, stored))
	}

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
	if limit > 1000 {
		limit = 1000
	}

	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	out := append([]ImageDetail(nil), all[offset:end]...)
	if end < len(all) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *Service) BatchDeleteImage(repositoryName string, imageIDs []ImageIdentifier) ([]ImageIdentifier, []ImageFailure, error) {
	if len(imageIDs) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return nil, nil, err
	}

	imagesByDigest := s.imagesByRepository[repository.RepositoryName]
	tagIndex := s.imageTagsByRepo[repository.RepositoryName]
	deleted := make([]ImageIdentifier, 0, len(imageIDs))
	failures := make([]ImageFailure, 0)

	for _, rawID := range imageIDs {
		imageID := ImageIdentifier{
			ImageDigest: strings.TrimSpace(rawID.ImageDigest),
			ImageTag:    strings.TrimSpace(rawID.ImageTag),
		}
		if imageID.ImageDigest == "" && imageID.ImageTag == "" {
			failures = append(failures, ImageFailure{
				ImageID:       imageID,
				FailureCode:   "MissingDigestAndTag",
				FailureReason: "image digest or tag is required",
			})
			continue
		}

		digest, err := resolveDigestFromIdentifier(imageID, imagesByDigest, tagIndex)
		if err != nil {
			failures = append(failures, ImageFailure{
				ImageID:       imageID,
				FailureCode:   imageFailureCodeForErr(err),
				FailureReason: err.Error(),
			})
			continue
		}

		stored := imagesByDigest[digest]
		if stored == nil {
			failures = append(failures, ImageFailure{
				ImageID:       imageID,
				FailureCode:   "ImageNotFound",
				FailureReason: "image not found",
			})
			continue
		}

		if imageID.ImageTag != "" {
			delete(tagIndex, imageID.ImageTag)
			delete(stored.Tags, imageID.ImageTag)
			if len(stored.Tags) == 0 {
				delete(imagesByDigest, digest)
			}
			deleted = append(deleted, ImageIdentifier{ImageDigest: digest, ImageTag: imageID.ImageTag})
			continue
		}

		for tag := range stored.Tags {
			delete(tagIndex, tag)
		}
		delete(imagesByDigest, digest)
		deleted = append(deleted, ImageIdentifier{ImageDigest: digest})
	}

	return deleted, failures, nil
}

func (s *Service) PutLifecyclePolicy(repositoryName, lifecyclePolicyText string) (LifecyclePolicy, error) {
	lifecyclePolicyText = strings.TrimSpace(lifecyclePolicyText)
	if lifecyclePolicyText == "" || !json.Valid([]byte(lifecyclePolicyText)) {
		return LifecyclePolicy{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return LifecyclePolicy{}, err
	}

	now := time.Now().UTC()
	s.lifecyclePolicies[repository.RepositoryName] = &lifecyclePolicyState{
		Text:            lifecyclePolicyText,
		LastEvaluatedAt: now,
	}

	return LifecyclePolicy{
		RegistryID:          repository.RegistryID,
		RepositoryName:      repository.RepositoryName,
		LifecyclePolicyText: lifecyclePolicyText,
		LastEvaluatedAt:     now,
	}, nil
}

func (s *Service) GetLifecyclePolicy(repositoryName string) (LifecyclePolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return LifecyclePolicy{}, err
	}
	state := s.lifecyclePolicies[repository.RepositoryName]
	if state == nil || strings.TrimSpace(state.Text) == "" {
		return LifecyclePolicy{}, ErrLifecyclePolicyNotFound
	}

	return LifecyclePolicy{
		RegistryID:          repository.RegistryID,
		RepositoryName:      repository.RepositoryName,
		LifecyclePolicyText: state.Text,
		LastEvaluatedAt:     state.LastEvaluatedAt,
	}, nil
}

func (s *Service) DeleteLifecyclePolicy(repositoryName string) (LifecyclePolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return LifecyclePolicy{}, err
	}
	state := s.lifecyclePolicies[repository.RepositoryName]
	if state == nil || strings.TrimSpace(state.Text) == "" {
		return LifecyclePolicy{}, ErrLifecyclePolicyNotFound
	}

	out := LifecyclePolicy{
		RegistryID:          repository.RegistryID,
		RepositoryName:      repository.RepositoryName,
		LifecyclePolicyText: state.Text,
		LastEvaluatedAt:     state.LastEvaluatedAt,
	}
	delete(s.lifecyclePolicies, repository.RepositoryName)
	return out, nil
}

func (s *Service) StartLifecyclePolicyPreview(repositoryName, lifecyclePolicyText string) (LifecyclePolicy, string, error) {
	lifecyclePolicyText = strings.TrimSpace(lifecyclePolicyText)

	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return LifecyclePolicy{}, "", err
	}

	if lifecyclePolicyText != "" {
		if !json.Valid([]byte(lifecyclePolicyText)) {
			return LifecyclePolicy{}, "", ErrInvalidParameter
		}
	} else {
		state := s.lifecyclePolicies[repository.RepositoryName]
		if state == nil || strings.TrimSpace(state.Text) == "" {
			return LifecyclePolicy{}, "", ErrLifecyclePolicyNotFound
		}
		lifecyclePolicyText = state.Text
	}

	now := time.Now().UTC()
	results := buildLifecyclePreviewResults(repository, s.imagesByRepository[repository.RepositoryName])
	s.lifecyclePreviews[repository.RepositoryName] = &lifecyclePreviewState{
		PolicyText: lifecyclePolicyText,
		Status:     "COMPLETE",
		Results:    results,
		UpdatedAt:  now,
	}

	if state := s.lifecyclePolicies[repository.RepositoryName]; state != nil && state.Text == lifecyclePolicyText {
		state.LastEvaluatedAt = now
	}

	return LifecyclePolicy{
		RegistryID:          repository.RegistryID,
		RepositoryName:      repository.RepositoryName,
		LifecyclePolicyText: lifecyclePolicyText,
		LastEvaluatedAt:     now,
	}, "COMPLETE", nil
}

func (s *Service) GetLifecyclePolicyPreview(repositoryName string, imageIDs []ImageIdentifier, tagStatus, nextToken string, maxResults int32) (LifecyclePolicy, string, []LifecyclePolicyPreviewResult, LifecyclePolicyPreviewSummary, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return LifecyclePolicy{}, "", nil, LifecyclePolicyPreviewSummary{}, "", err
	}
	preview := s.lifecyclePreviews[repository.RepositoryName]
	if preview == nil {
		return LifecyclePolicy{}, "", nil, LifecyclePolicyPreviewSummary{}, "", ErrLifecyclePreviewNotFound
	}
	if len(imageIDs) > 0 && (strings.TrimSpace(nextToken) != "" || maxResults > 0) {
		return LifecyclePolicy{}, "", nil, LifecyclePolicyPreviewSummary{}, "", ErrInvalidParameter
	}

	filterTagStatus := strings.ToUpper(strings.TrimSpace(tagStatus))
	if filterTagStatus == "" {
		filterTagStatus = "ANY"
	}
	if filterTagStatus != "ANY" && filterTagStatus != "TAGGED" && filterTagStatus != "UNTAGGED" {
		return LifecyclePolicy{}, "", nil, LifecyclePolicyPreviewSummary{}, "", ErrInvalidParameter
	}

	selected := make([]LifecyclePolicyPreviewResult, 0, len(preview.Results))
	if len(imageIDs) > 0 {
		for _, imageID := range imageIDs {
			candidate := ImageIdentifier{ImageDigest: strings.TrimSpace(imageID.ImageDigest), ImageTag: strings.TrimSpace(imageID.ImageTag)}
			if candidate.ImageDigest == "" && candidate.ImageTag == "" {
				return LifecyclePolicy{}, "", nil, LifecyclePolicyPreviewSummary{}, "", ErrInvalidParameter
			}
			for _, result := range preview.Results {
				if previewMatchesIdentifier(result, candidate) {
					selected = append(selected, clonePreviewResult(result))
					break
				}
			}
		}
	} else {
		for _, result := range preview.Results {
			tagged := len(result.ImageTags) > 0
			if filterTagStatus == "TAGGED" && !tagged {
				continue
			}
			if filterTagStatus == "UNTAGGED" && tagged {
				continue
			}
			selected = append(selected, clonePreviewResult(result))
		}
		sort.Slice(selected, func(i, j int) bool {
			return selected[i].ImageDigest < selected[j].ImageDigest
		})
	}

	total := len(selected)
	offset := 0
	nextToken = strings.TrimSpace(nextToken)
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 0 || parsed > total {
			return LifecyclePolicy{}, "", nil, LifecyclePolicyPreviewSummary{}, "", ErrInvalidParameter
		}
		offset = parsed
	}

	limit := int(maxResults)
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	end := offset + limit
	if end > total {
		end = total
	}
	page := append([]LifecyclePolicyPreviewResult(nil), selected[offset:end]...)
	next := ""
	if end < total {
		next = strconv.Itoa(end)
	}

	policy := LifecyclePolicy{
		RegistryID:          repository.RegistryID,
		RepositoryName:      repository.RepositoryName,
		LifecyclePolicyText: preview.PolicyText,
		LastEvaluatedAt:     preview.UpdatedAt,
	}
	summary := LifecyclePolicyPreviewSummary{ExpiringImageTotalCount: int32(total)}
	return policy, preview.Status, page, summary, next, nil
}

func (s *Service) DescribeRegistry() (string, ReplicationConfiguration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return DefaultAccountID, cloneReplicationConfiguration(s.replicationConfig), nil
}

func (s *Service) GetRegistryPolicy() (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(s.registryPolicyText) == "" {
		return "", "", ErrRegistryPolicyNotFound
	}
	return DefaultAccountID, s.registryPolicyText, nil
}

func (s *Service) PutRegistryPolicy(policyText string) (string, string, error) {
	policyText = strings.TrimSpace(policyText)
	if policyText == "" || !json.Valid([]byte(policyText)) {
		return "", "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.registryPolicyText = policyText
	return DefaultAccountID, s.registryPolicyText, nil
}

func (s *Service) DeleteRegistryPolicy() (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(s.registryPolicyText) == "" {
		return "", "", ErrRegistryPolicyNotFound
	}
	policyText := s.registryPolicyText
	s.registryPolicyText = ""
	return DefaultAccountID, policyText, nil
}

func (s *Service) GetRegistryScanningConfiguration() (string, RegistryScanningConfiguration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return DefaultAccountID, cloneRegistryScanningConfiguration(s.registryScanning), nil
}

func (s *Service) PutRegistryScanningConfiguration(scanType string, rules []RegistryScanningRule) (RegistryScanningConfiguration, error) {
	normalizedScanType, err := normalizeRegistryScanType(scanType)
	if err != nil {
		return RegistryScanningConfiguration{}, err
	}
	normalizedRules, err := normalizeRegistryScanningRules(rules)
	if err != nil {
		return RegistryScanningConfiguration{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.registryScanning = RegistryScanningConfiguration{
		ScanType: normalizedScanType,
		Rules:    normalizedRules,
	}
	return cloneRegistryScanningConfiguration(s.registryScanning), nil
}

func (s *Service) PutReplicationConfiguration(configuration ReplicationConfiguration) (ReplicationConfiguration, error) {
	normalized, err := normalizeReplicationConfiguration(configuration)
	if err != nil {
		return ReplicationConfiguration{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.replicationConfig = normalized
	return cloneReplicationConfiguration(s.replicationConfig), nil
}

func (s *Service) GetAccountSetting(name string) (AccountSetting, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return AccountSetting{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return AccountSetting{
		Name:  name,
		Value: s.accountSettings[name],
	}, nil
}

func (s *Service) PutAccountSetting(name, value string) (AccountSetting, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return AccountSetting{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountSettings[name] = value
	return AccountSetting{Name: name, Value: value}, nil
}

func (s *Service) DescribeImageReplicationStatus(repositoryName string, imageID ImageIdentifier) (ImageIdentifier, []ImageReplicationStatus, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return ImageIdentifier{}, nil, "", err
	}

	imagesByDigest := s.imagesByRepository[repository.RepositoryName]
	tagIndex := s.imageTagsByRepo[repository.RepositoryName]
	digest, err := resolveDigestFromIdentifier(imageID, imagesByDigest, tagIndex)
	if err != nil {
		return ImageIdentifier{}, nil, "", err
	}
	stored := imagesByDigest[digest]
	if stored == nil {
		return ImageIdentifier{}, nil, "", ErrImageNotFound
	}

	outImageID := ImageIdentifier{
		ImageDigest: digest,
		ImageTag:    strings.TrimSpace(imageID.ImageTag),
	}
	if outImageID.ImageTag == "" {
		outImageID.ImageTag = firstTag(stored.Tags)
	}

	statuses := []ImageReplicationStatus{{
		Region:     DefaultRegion,
		RegistryID: DefaultAccountID,
		Status:     "COMPLETE",
	}}

	return outImageID, statuses, repository.RepositoryName, nil
}

func (s *Service) CreatePullThroughCacheRule(prefix, upstreamRegistryURL, credentialArn, upstreamRegistry string) (PullThroughCacheRule, error) {
	prefix = strings.TrimSpace(prefix)
	upstreamRegistryURL = strings.TrimSpace(upstreamRegistryURL)
	credentialArn = strings.TrimSpace(credentialArn)
	upstreamRegistry = strings.TrimSpace(upstreamRegistry)

	if !repositoryNamePattern.MatchString(prefix) || upstreamRegistryURL == "" {
		return PullThroughCacheRule{}, ErrInvalidParameter
	}
	if upstreamRegistry == "" {
		upstreamRegistry = inferUpstreamRegistry(upstreamRegistryURL)
	}
	if !isValidUpstreamRegistry(upstreamRegistry) {
		return PullThroughCacheRule{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.pullThroughRules[prefix]; exists {
		return PullThroughCacheRule{}, ErrPullThroughRuleExists
	}

	now := time.Now().UTC()
	rule := &PullThroughCacheRule{
		CreatedAt:           now,
		CredentialArn:       credentialArn,
		ECRRepositoryPrefix: prefix,
		RegistryID:          DefaultAccountID,
		UpdatedAt:           now,
		UpstreamRegistry:    upstreamRegistry,
		UpstreamRegistryURL: upstreamRegistryURL,
	}
	s.pullThroughRules[prefix] = rule
	return clonePullThroughRule(rule), nil
}

func (s *Service) DescribePullThroughCacheRules(prefixes []string, nextToken string, maxResults int32) ([]PullThroughCacheRule, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	all := make([]*PullThroughCacheRule, 0, len(s.pullThroughRules))
	if len(prefixes) > 0 {
		seen := map[string]struct{}{}
		for _, raw := range prefixes {
			prefix := strings.TrimSpace(raw)
			if prefix == "" {
				continue
			}
			if _, exists := seen[prefix]; exists {
				continue
			}
			seen[prefix] = struct{}{}
			if rule := s.pullThroughRules[prefix]; rule != nil {
				all = append(all, rule)
			}
		}
	} else {
		for _, rule := range s.pullThroughRules {
			all = append(all, rule)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].ECRRepositoryPrefix < all[j].ECRRepositoryPrefix
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
	if limit > 1000 {
		limit = 1000
	}

	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	selected := all[offset:end]
	out := make([]PullThroughCacheRule, 0, len(selected))
	for _, rule := range selected {
		out = append(out, clonePullThroughRule(rule))
	}

	if end < len(all) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *Service) UpdatePullThroughCacheRule(prefix, credentialArn string) (PullThroughCacheRule, error) {
	prefix = strings.TrimSpace(prefix)
	credentialArn = strings.TrimSpace(credentialArn)
	if prefix == "" || credentialArn == "" {
		return PullThroughCacheRule{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	rule := s.pullThroughRules[prefix]
	if rule == nil {
		return PullThroughCacheRule{}, ErrPullThroughRuleNotFound
	}
	rule.CredentialArn = credentialArn
	rule.UpdatedAt = time.Now().UTC()
	return clonePullThroughRule(rule), nil
}

func (s *Service) DeletePullThroughCacheRule(prefix string) (PullThroughCacheRule, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return PullThroughCacheRule{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	rule := s.pullThroughRules[prefix]
	if rule == nil {
		return PullThroughCacheRule{}, ErrPullThroughRuleNotFound
	}
	out := clonePullThroughRule(rule)
	delete(s.pullThroughRules, prefix)
	return out, nil
}

func (s *Service) ValidatePullThroughCacheRule(prefix string) (PullThroughCacheRule, bool, string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return PullThroughCacheRule{}, false, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	rule := s.pullThroughRules[prefix]
	if rule == nil {
		return PullThroughCacheRule{}, false, "", ErrPullThroughRuleNotFound
	}

	isValid := strings.TrimSpace(rule.UpstreamRegistryURL) != ""
	failure := ""
	if !isValid {
		failure = "upstream registry URL is required"
	}
	return clonePullThroughRule(rule), isValid, failure, nil
}

func (s *Service) CreateRepositoryCreationTemplate(input RepositoryCreationTemplateInput) (RepositoryCreationTemplate, error) {
	normalized, err := normalizeRepositoryCreationTemplateInput(input, true)
	if err != nil {
		return RepositoryCreationTemplate{}, err
	}

	now := time.Now().UTC()
	template := &RepositoryCreationTemplate{
		AppliedFor:         normalized.AppliedFor,
		CreatedAt:          now,
		ImageTagMutability: "MUTABLE",
		Prefix:             normalized.Prefix,
		ResourceTags:       map[string]string{},
		UpdatedAt:          now,
	}
	if normalized.CustomRoleArn != nil {
		template.CustomRoleArn = *normalized.CustomRoleArn
	}
	if normalized.Description != nil {
		template.Description = *normalized.Description
	}
	if normalized.EncryptionType != nil {
		template.EncryptionType = *normalized.EncryptionType
	}
	if normalized.KMSKey != nil {
		template.KMSKey = *normalized.KMSKey
	}
	if normalized.ImageTagMutability != nil {
		template.ImageTagMutability = *normalized.ImageTagMutability
	}
	if normalized.LifecyclePolicy != nil {
		template.LifecyclePolicy = *normalized.LifecyclePolicy
	}
	if normalized.RepositoryPolicy != nil {
		template.RepositoryPolicy = *normalized.RepositoryPolicy
	}
	if normalized.ResourceTagsSet {
		template.ResourceTags = cloneStringMap(normalized.ResourceTags)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.creationTemplates[template.Prefix]; exists {
		return RepositoryCreationTemplate{}, ErrTemplateAlreadyExists
	}
	s.creationTemplates[template.Prefix] = template
	return cloneRepositoryCreationTemplate(template), nil
}

func (s *Service) DescribeRepositoryCreationTemplates(prefixes []string, nextToken string, maxResults int32) ([]RepositoryCreationTemplate, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	all := make([]*RepositoryCreationTemplate, 0, len(s.creationTemplates))
	if len(prefixes) > 0 {
		seen := map[string]struct{}{}
		for _, raw := range prefixes {
			prefix, err := normalizeTemplatePrefix(raw)
			if err != nil {
				return nil, "", err
			}
			if _, exists := seen[prefix]; exists {
				continue
			}
			seen[prefix] = struct{}{}
			if template := s.creationTemplates[prefix]; template != nil {
				all = append(all, template)
			}
		}
	} else {
		for _, template := range s.creationTemplates {
			all = append(all, template)
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Prefix < all[j].Prefix
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
	if limit > 1000 {
		limit = 1000
	}

	end := offset + limit
	if end > len(all) {
		end = len(all)
	}

	out := make([]RepositoryCreationTemplate, 0, end-offset)
	for _, template := range all[offset:end] {
		out = append(out, cloneRepositoryCreationTemplate(template))
	}
	if end < len(all) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *Service) UpdateRepositoryCreationTemplate(input RepositoryCreationTemplateInput) (RepositoryCreationTemplate, error) {
	normalized, err := normalizeRepositoryCreationTemplateInput(input, false)
	if err != nil {
		return RepositoryCreationTemplate{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	template := s.creationTemplates[normalized.Prefix]
	if template == nil {
		return RepositoryCreationTemplate{}, ErrTemplateNotFound
	}

	if normalized.AppliedForSet {
		template.AppliedFor = append([]string(nil), normalized.AppliedFor...)
	}
	if normalized.CustomRoleArn != nil {
		template.CustomRoleArn = *normalized.CustomRoleArn
	}
	if normalized.Description != nil {
		template.Description = *normalized.Description
	}
	if normalized.EncryptionType != nil {
		template.EncryptionType = *normalized.EncryptionType
	}
	if normalized.KMSKey != nil {
		template.KMSKey = *normalized.KMSKey
	}
	if normalized.ImageTagMutability != nil {
		template.ImageTagMutability = *normalized.ImageTagMutability
	}
	if normalized.LifecyclePolicy != nil {
		template.LifecyclePolicy = *normalized.LifecyclePolicy
	}
	if normalized.RepositoryPolicy != nil {
		template.RepositoryPolicy = *normalized.RepositoryPolicy
	}
	if normalized.ResourceTagsSet {
		template.ResourceTags = cloneStringMap(normalized.ResourceTags)
	}
	template.UpdatedAt = time.Now().UTC()
	return cloneRepositoryCreationTemplate(template), nil
}

func (s *Service) DeleteRepositoryCreationTemplate(prefix string) (RepositoryCreationTemplate, error) {
	normalizedPrefix, err := normalizeTemplatePrefix(prefix)
	if err != nil {
		return RepositoryCreationTemplate{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	template := s.creationTemplates[normalizedPrefix]
	if template == nil {
		return RepositoryCreationTemplate{}, ErrTemplateNotFound
	}
	out := cloneRepositoryCreationTemplate(template)
	delete(s.creationTemplates, normalizedPrefix)
	return out, nil
}

func (s *Service) PutSigningConfiguration(configuration SigningConfiguration) (SigningConfiguration, error) {
	normalized, err := normalizeSigningConfiguration(configuration)
	if err != nil {
		return SigningConfiguration{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.signingConfiguration = &normalized
	return cloneSigningConfiguration(s.signingConfiguration), nil
}

func (s *Service) GetSigningConfiguration() (string, SigningConfiguration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.signingConfiguration == nil {
		return "", SigningConfiguration{}, ErrSigningConfigNotFound
	}
	return DefaultAccountID, cloneSigningConfiguration(s.signingConfiguration), nil
}

func (s *Service) DeleteSigningConfiguration() (string, SigningConfiguration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.signingConfiguration == nil {
		return "", SigningConfiguration{}, ErrSigningConfigNotFound
	}
	out := cloneSigningConfiguration(s.signingConfiguration)
	s.signingConfiguration = nil
	return DefaultAccountID, out, nil
}

func (s *Service) DescribeImageSigningStatus(repositoryName string, imageID ImageIdentifier) (ImageIdentifier, string, string, []ImageSigningStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return ImageIdentifier{}, "", "", nil, err
	}

	imagesByDigest := s.imagesByRepository[repository.RepositoryName]
	tagIndex := s.imageTagsByRepo[repository.RepositoryName]
	digest, err := resolveDigestFromIdentifier(imageID, imagesByDigest, tagIndex)
	if err != nil {
		return ImageIdentifier{}, "", "", nil, err
	}
	stored := imagesByDigest[digest]
	if stored == nil {
		return ImageIdentifier{}, "", "", nil, ErrImageNotFound
	}

	outImageID := ImageIdentifier{
		ImageDigest: digest,
		ImageTag:    strings.TrimSpace(imageID.ImageTag),
	}
	if outImageID.ImageTag == "" {
		outImageID.ImageTag = firstTag(stored.Tags)
	}

	statuses := []ImageSigningStatus{}
	if s.signingConfiguration != nil {
		for _, rule := range s.signingConfiguration.Rules {
			if signingRuleMatchesRepository(rule, repository.RepositoryName) {
				statuses = append(statuses, ImageSigningStatus{
					SigningProfileArn: rule.SigningProfileArn,
					Status:            "COMPLETE",
				})
			}
		}
		sort.Slice(statuses, func(i, j int) bool {
			return statuses[i].SigningProfileArn < statuses[j].SigningProfileArn
		})
	}

	return outImageID, repository.RepositoryName, DefaultAccountID, statuses, nil
}

func (s *Service) StartImageScan(repositoryName string, imageID ImageIdentifier) (ImageIdentifier, ImageScanStatus, string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return ImageIdentifier{}, ImageScanStatus{}, "", "", err
	}

	imagesByDigest := s.imagesByRepository[repository.RepositoryName]
	tagIndex := s.imageTagsByRepo[repository.RepositoryName]
	digest, err := resolveDigestFromIdentifier(imageID, imagesByDigest, tagIndex)
	if err != nil {
		return ImageIdentifier{}, ImageScanStatus{}, "", "", err
	}
	stored := imagesByDigest[digest]
	if stored == nil {
		return ImageIdentifier{}, ImageScanStatus{}, "", "", ErrImageNotFound
	}

	outImageID := ImageIdentifier{
		ImageDigest: digest,
		ImageTag:    strings.TrimSpace(imageID.ImageTag),
	}
	if outImageID.ImageTag == "" {
		outImageID.ImageTag = firstTag(stored.Tags)
	}

	now := time.Now().UTC()
	status := ImageScanStatus{
		Description: "scan completed",
		Status:      "COMPLETE",
	}
	scanFindings := ImageScanFindings{
		FindingSeverityCounts: map[string]int32{
			"HIGH": 1,
		},
		Findings: []ImageScanFinding{
			{
				Name:        "CVE-0000-0000",
				Description: "simulated scan finding",
				Severity:    "HIGH",
				URI:         "https://example.com/security/CVE-0000-0000",
				Attributes: []ImageScanAttribute{
					{Key: "package_name", Value: "openssl"},
					{Key: "package_version", Value: "1.1.1"},
				},
			},
		},
		ImageScanCompletedAt:         now,
		VulnerabilitySourceUpdatedAt: now,
	}
	repoScans := s.scanStateByRepo[repository.RepositoryName]
	if repoScans == nil {
		repoScans = map[string]*imageScanState{}
		s.scanStateByRepo[repository.RepositoryName] = repoScans
	}
	repoScans[digest] = &imageScanState{
		Findings:    scanFindings,
		Status:      status,
		LastScanned: now,
	}
	return outImageID, status, DefaultAccountID, repository.RepositoryName, nil
}

func (s *Service) DescribeImageScanFindings(repositoryName string, imageID ImageIdentifier, nextToken string, maxResults int32) (ImageIdentifier, ImageScanFindings, ImageScanStatus, string, string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return ImageIdentifier{}, ImageScanFindings{}, ImageScanStatus{}, "", "", "", err
	}

	imagesByDigest := s.imagesByRepository[repository.RepositoryName]
	tagIndex := s.imageTagsByRepo[repository.RepositoryName]
	digest, err := resolveDigestFromIdentifier(imageID, imagesByDigest, tagIndex)
	if err != nil {
		return ImageIdentifier{}, ImageScanFindings{}, ImageScanStatus{}, "", "", "", err
	}
	stored := imagesByDigest[digest]
	if stored == nil {
		return ImageIdentifier{}, ImageScanFindings{}, ImageScanStatus{}, "", "", "", ErrImageNotFound
	}

	repoScans := s.scanStateByRepo[repository.RepositoryName]
	if repoScans == nil || repoScans[digest] == nil {
		return ImageIdentifier{}, ImageScanFindings{}, ImageScanStatus{}, "", "", "", ErrScanNotFound
	}
	scan := repoScans[digest]

	outImageID := ImageIdentifier{
		ImageDigest: digest,
		ImageTag:    strings.TrimSpace(imageID.ImageTag),
	}
	if outImageID.ImageTag == "" {
		outImageID.ImageTag = firstTag(stored.Tags)
	}

	allFindings := scan.Findings.Findings
	offset := 0
	nextToken = strings.TrimSpace(nextToken)
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 0 || parsed > len(allFindings) {
			return ImageIdentifier{}, ImageScanFindings{}, ImageScanStatus{}, "", "", "", ErrInvalidParameter
		}
		offset = parsed
	}
	limit := int(maxResults)
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	end := offset + limit
	if end > len(allFindings) {
		end = len(allFindings)
	}

	pagedFindings := make([]ImageScanFinding, 0, end-offset)
	for _, finding := range allFindings[offset:end] {
		pagedFindings = append(pagedFindings, cloneImageScanFinding(finding))
	}

	newNextToken := ""
	if end < len(allFindings) {
		newNextToken = strconv.Itoa(end)
	}

	return outImageID, ImageScanFindings{
		FindingSeverityCounts:        cloneSeverityCounts(scan.Findings.FindingSeverityCounts),
		Findings:                     pagedFindings,
		ImageScanCompletedAt:         scan.Findings.ImageScanCompletedAt,
		VulnerabilitySourceUpdatedAt: scan.Findings.VulnerabilitySourceUpdatedAt,
	}, scan.Status, DefaultAccountID, repository.RepositoryName, newNextToken, nil
}

func (s *Service) RegisterPullTimeUpdateExclusion(principalArn string) (PullTimeUpdateExclusion, error) {
	principalArn = strings.TrimSpace(principalArn)
	if principalArn == "" {
		return PullTimeUpdateExclusion{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.pullTimeExclusions[principalArn]; existing != nil {
		return PullTimeUpdateExclusion{
			CreatedAt:    existing.CreatedAt,
			PrincipalArn: existing.PrincipalArn,
		}, nil
	}
	exclusion := &PullTimeUpdateExclusion{
		CreatedAt:    time.Now().UTC(),
		PrincipalArn: principalArn,
	}
	s.pullTimeExclusions[principalArn] = exclusion
	return PullTimeUpdateExclusion{
		CreatedAt:    exclusion.CreatedAt,
		PrincipalArn: exclusion.PrincipalArn,
	}, nil
}

func (s *Service) DeregisterPullTimeUpdateExclusion(principalArn string) (string, error) {
	principalArn = strings.TrimSpace(principalArn)
	if principalArn == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pullTimeExclusions, principalArn)
	return principalArn, nil
}

func (s *Service) ListPullTimeUpdateExclusions(nextToken string, maxResults int32) ([]string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	all := make([]string, 0, len(s.pullTimeExclusions))
	for principalArn := range s.pullTimeExclusions {
		all = append(all, principalArn)
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
	if limit > 1000 {
		limit = 1000
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

func (s *Service) ListImageReferrers(repositoryName, subjectDigest, artifactStatus string, artifactTypes []string, nextToken string, maxResults int32) ([]ImageReferrer, string, error) {
	subjectDigest = strings.TrimSpace(subjectDigest)
	if !isDigest(subjectDigest) {
		return nil, "", ErrInvalidParameter
	}

	artifactStatus = strings.ToUpper(strings.TrimSpace(artifactStatus))
	if artifactStatus == "" {
		artifactStatus = "ACTIVE"
	}
	switch artifactStatus {
	case "ACTIVE", "ARCHIVED", "ACTIVATING", "ANY":
	default:
		return nil, "", ErrInvalidParameter
	}

	allowedArtifactTypes := map[string]struct{}{}
	for _, raw := range artifactTypes {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		allowedArtifactTypes[item] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return nil, "", err
	}

	imagesByDigest := s.imagesByRepository[repository.RepositoryName]
	referrers := make([]ImageReferrer, 0)
	for _, image := range imagesByDigest {
		if image == nil || image.SubjectDigest != subjectDigest {
			continue
		}
		status := image.ImageStatus
		if status == "" {
			status = "ACTIVE"
		}
		if artifactStatus != "ANY" && status != artifactStatus {
			continue
		}
		if len(allowedArtifactTypes) > 0 {
			if _, ok := allowedArtifactTypes[image.ArtifactType]; !ok {
				continue
			}
		}
		referrers = append(referrers, ImageReferrer{
			Annotations:  map[string]string{},
			ArtifactType: image.ArtifactType,
			Digest:       image.Digest,
			MediaType:    image.ManifestMediaType,
			Size:         int64(len(image.Manifest)),
			Status:       status,
		})
	}
	sort.Slice(referrers, func(i, j int) bool {
		return referrers[i].Digest < referrers[j].Digest
	})

	offset := 0
	nextToken = strings.TrimSpace(nextToken)
	if nextToken != "" {
		parsed, err := strconv.Atoi(nextToken)
		if err != nil || parsed < 0 || parsed > len(referrers) {
			return nil, "", ErrInvalidParameter
		}
		offset = parsed
	}
	limit := int(maxResults)
	if limit <= 0 {
		limit = 50
	}
	if limit > 50 {
		limit = 50
	}
	end := offset + limit
	if end > len(referrers) {
		end = len(referrers)
	}
	out := make([]ImageReferrer, 0, end-offset)
	for _, item := range referrers[offset:end] {
		out = append(out, cloneImageReferrer(item))
	}
	newNextToken := ""
	if end < len(referrers) {
		newNextToken = strconv.Itoa(end)
	}
	return out, newNextToken, nil
}

func (s *Service) UpdateImageStorageClass(repositoryName string, imageID ImageIdentifier, targetStorageClass string) (ImageIdentifier, string, string, string, error) {
	targetStorageClass = strings.ToUpper(strings.TrimSpace(targetStorageClass))
	switch targetStorageClass {
	case "STANDARD", "ARCHIVE":
	default:
		return ImageIdentifier{}, "", "", "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	repository, err := s.repositoryForNameLocked(repositoryName)
	if err != nil {
		return ImageIdentifier{}, "", "", "", err
	}
	imagesByDigest := s.imagesByRepository[repository.RepositoryName]
	tagIndex := s.imageTagsByRepo[repository.RepositoryName]
	digest, err := resolveDigestFromIdentifier(imageID, imagesByDigest, tagIndex)
	if err != nil {
		return ImageIdentifier{}, "", "", "", err
	}
	stored := imagesByDigest[digest]
	if stored == nil {
		return ImageIdentifier{}, "", "", "", ErrImageNotFound
	}

	if targetStorageClass == "STANDARD" {
		stored.ImageStatus = "ACTIVE"
	} else {
		stored.ImageStatus = "ARCHIVED"
	}
	outImageID := ImageIdentifier{
		ImageDigest: digest,
		ImageTag:    strings.TrimSpace(imageID.ImageTag),
	}
	if outImageID.ImageTag == "" {
		outImageID.ImageTag = firstTag(stored.Tags)
	}
	return outImageID, stored.ImageStatus, DefaultAccountID, repository.RepositoryName, nil
}

func (s *Service) repositoryForNameLocked(repositoryName string) (*Repository, error) {
	repositoryName = strings.TrimSpace(repositoryName)
	if repositoryName == "" {
		return nil, ErrInvalidParameter
	}
	repository, ok := s.repositoriesByName[repositoryName]
	if !ok {
		return nil, ErrRepositoryNotFound
	}
	return repository, nil
}

func (s *Service) repositoryForARNLocked(resourceArn string) (*Repository, error) {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		return nil, ErrInvalidParameter
	}
	repository, ok := s.repositoriesByARN[resourceArn]
	if !ok {
		return nil, ErrRepositoryNotFound
	}
	return repository, nil
}

func normalizeRegistryIDs(registryIDs []string) ([]string, error) {
	if len(registryIDs) == 0 {
		return []string{DefaultAccountID}, nil
	}
	out := make([]string, 0, len(registryIDs))
	seen := map[string]struct{}{}
	for _, raw := range registryIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if len(id) != 12 {
			return nil, ErrInvalidParameter
		}
		for _, r := range id {
			if r < '0' || r > '9' {
				return nil, ErrInvalidParameter
			}
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, ErrInvalidParameter
	}
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

func normalizeImageTagMutability(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "MUTABLE", nil
	}
	switch value {
	case "MUTABLE", "IMMUTABLE", "IMMUTABLE_WITH_EXCLUSION", "MUTABLE_WITH_EXCLUSION":
		return value, nil
	default:
		return "", ErrInvalidParameter
	}
}

func normalizeEncryptionType(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "AES256", nil
	}
	switch value {
	case "AES256", "KMS", "KMS_DSSE":
		return value, nil
	default:
		return "", ErrInvalidParameter
	}
}

func isImageTagMutable(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "MUTABLE") || strings.EqualFold(strings.TrimSpace(value), "MUTABLE_WITH_EXCLUSION")
}

func repositoryARN(repositoryName string) string {
	return fmt.Sprintf("arn:aws:ecr:%s:%s:repository/%s", DefaultRegion, DefaultAccountID, repositoryName)
}

func repositoryURI(repositoryName string) string {
	return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s", DefaultAccountID, DefaultRegion, repositoryName)
}

func sha256Digest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func isDigest(value string) bool {
	return digestPattern.MatchString(strings.TrimSpace(value))
}

func firstTag(tags map[string]struct{}) string {
	if len(tags) == 0 {
		return ""
	}
	out := make([]string, 0, len(tags))
	for tag := range tags {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out[0]
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func parseManifestMetadata(manifest string) (artifactType string, subjectDigest string) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(manifest), &payload); err != nil {
		return "", ""
	}

	artifactType = strings.TrimSpace(asString(payload["artifactType"]))
	if artifactType == "" {
		artifactType = strings.TrimSpace(asString(payload["configMediaType"]))
	}
	subject, ok := payload["subject"].(map[string]any)
	if !ok {
		return artifactType, ""
	}
	digest := strings.TrimSpace(asString(subject["digest"]))
	if !isDigest(digest) {
		return artifactType, ""
	}
	return artifactType, digest
}

func asString(v any) string {
	value, _ := v.(string)
	return value
}

func cloneSeverityCounts(in map[string]int32) map[string]int32 {
	if len(in) == 0 {
		return map[string]int32{}
	}
	out := make(map[string]int32, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneImageScanFinding(finding ImageScanFinding) ImageScanFinding {
	attributes := make([]ImageScanAttribute, 0, len(finding.Attributes))
	for _, attribute := range finding.Attributes {
		attributes = append(attributes, ImageScanAttribute{
			Key:   attribute.Key,
			Value: attribute.Value,
		})
	}
	return ImageScanFinding{
		Name:        finding.Name,
		Description: finding.Description,
		Severity:    finding.Severity,
		URI:         finding.URI,
		Attributes:  attributes,
	}
}

func cloneImageReferrer(referrer ImageReferrer) ImageReferrer {
	return ImageReferrer{
		Annotations:  cloneStringMap(referrer.Annotations),
		ArtifactType: referrer.ArtifactType,
		Digest:       referrer.Digest,
		MediaType:    referrer.MediaType,
		Size:         referrer.Size,
		Status:       referrer.Status,
	}
}

func storedImageToDetail(repository *Repository, stored *storedImage) ImageDetail {
	tags := make([]string, 0, len(stored.Tags))
	for tag := range stored.Tags {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return ImageDetail{
		ImageDigest:            stored.Digest,
		ImageTags:              tags,
		ImageManifestMediaType: stored.ManifestMediaType,
		ImageSizeInBytes:       int64(len(stored.Manifest)),
		ImagePushedAt:          stored.PushedAt,
		RegistryID:             repository.RegistryID,
		RepositoryName:         repository.RepositoryName,
	}
}

func resolveDigestFromIdentifier(imageID ImageIdentifier, imagesByDigest map[string]*storedImage, tagIndex map[string]string) (string, error) {
	resolvedDigest := strings.TrimSpace(imageID.ImageDigest)
	tag := strings.TrimSpace(imageID.ImageTag)

	if resolvedDigest == "" && tag == "" {
		return "", ErrInvalidParameter
	}
	if tag != "" {
		tagDigest, ok := tagIndex[tag]
		if !ok {
			return "", ErrImageNotFound
		}
		if resolvedDigest != "" && resolvedDigest != tagDigest {
			return "", ErrInvalidParameter
		}
		resolvedDigest = tagDigest
	}
	if resolvedDigest == "" {
		return "", ErrImageNotFound
	}
	if !isDigest(resolvedDigest) {
		return "", ErrInvalidParameter
	}
	if imagesByDigest[resolvedDigest] == nil {
		return "", ErrImageNotFound
	}
	return resolvedDigest, nil
}

func imageFailureCodeForErr(err error) string {
	switch {
	case errors.Is(err, ErrImageNotFound):
		return "ImageNotFound"
	case errors.Is(err, ErrInvalidParameter):
		return "ImageTagDoesNotMatchDigest"
	default:
		return "ImageNotFound"
	}
}

func buildLifecyclePreviewResults(repository *Repository, imagesByDigest map[string]*storedImage) []LifecyclePolicyPreviewResult {
	digests := make([]string, 0, len(imagesByDigest))
	for digest := range imagesByDigest {
		digests = append(digests, digest)
	}
	sort.Strings(digests)

	out := make([]LifecyclePolicyPreviewResult, 0, len(digests))
	for _, digest := range digests {
		stored := imagesByDigest[digest]
		if stored == nil {
			continue
		}
		tags := make([]string, 0, len(stored.Tags))
		for tag := range stored.Tags {
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		out = append(out, LifecyclePolicyPreviewResult{
			ImageDigest:         stored.Digest,
			ImageTags:           tags,
			ImagePushedAt:       stored.PushedAt,
			AppliedRulePriority: 1,
			ActionType:          "EXPIRE",
		})
	}
	return out
}

func previewMatchesIdentifier(result LifecyclePolicyPreviewResult, imageID ImageIdentifier) bool {
	if strings.TrimSpace(imageID.ImageDigest) != "" && strings.TrimSpace(imageID.ImageDigest) != result.ImageDigest {
		return false
	}
	if strings.TrimSpace(imageID.ImageTag) == "" {
		return true
	}
	for _, tag := range result.ImageTags {
		if tag == imageID.ImageTag {
			return true
		}
	}
	return false
}

func clonePreviewResult(result LifecyclePolicyPreviewResult) LifecyclePolicyPreviewResult {
	return LifecyclePolicyPreviewResult{
		ImageDigest:         result.ImageDigest,
		ImageTags:           append([]string(nil), result.ImageTags...),
		ImagePushedAt:       result.ImagePushedAt,
		AppliedRulePriority: result.AppliedRulePriority,
		ActionType:          result.ActionType,
	}
}

func normalizeRegistryScanType(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "BASIC", nil
	}
	switch value {
	case "BASIC", "ENHANCED":
		return value, nil
	default:
		return "", ErrInvalidParameter
	}
}

func normalizeRegistryScanningRules(rules []RegistryScanningRule) ([]RegistryScanningRule, error) {
	if len(rules) == 0 {
		return []RegistryScanningRule{}, nil
	}

	out := make([]RegistryScanningRule, 0, len(rules))
	for _, rule := range rules {
		scanFrequency := strings.ToUpper(strings.TrimSpace(rule.ScanFrequency))
		if scanFrequency == "" {
			scanFrequency = "MANUAL"
		}
		switch scanFrequency {
		case "MANUAL", "SCAN_ON_PUSH", "CONTINUOUS_SCAN":
		default:
			return nil, ErrInvalidParameter
		}

		if len(rule.RepositoryFilters) == 0 {
			return nil, ErrInvalidParameter
		}
		repositoryFilters := make([]ScanningRepositoryFilter, 0, len(rule.RepositoryFilters))
		for _, filter := range rule.RepositoryFilters {
			name := strings.TrimSpace(filter.Filter)
			if name == "" {
				return nil, ErrInvalidParameter
			}
			filterType := strings.ToUpper(strings.TrimSpace(filter.FilterType))
			if filterType == "" {
				filterType = "WILDCARD"
			}
			if filterType != "WILDCARD" {
				return nil, ErrInvalidParameter
			}
			repositoryFilters = append(repositoryFilters, ScanningRepositoryFilter{
				Filter:     name,
				FilterType: filterType,
			})
		}

		out = append(out, RegistryScanningRule{
			RepositoryFilters: repositoryFilters,
			ScanFrequency:     scanFrequency,
		})
	}
	return out, nil
}

func normalizeReplicationConfiguration(configuration ReplicationConfiguration) (ReplicationConfiguration, error) {
	if len(configuration.Rules) == 0 {
		return ReplicationConfiguration{Rules: []ReplicationRule{}}, nil
	}

	normalizedRules := make([]ReplicationRule, 0, len(configuration.Rules))
	for _, rule := range configuration.Rules {
		if len(rule.Destinations) == 0 {
			return ReplicationConfiguration{}, ErrInvalidParameter
		}

		destinations := make([]ReplicationDestination, 0, len(rule.Destinations))
		for _, destination := range rule.Destinations {
			region := strings.TrimSpace(destination.Region)
			registryID := strings.TrimSpace(destination.RegistryID)
			if region == "" || registryID == "" {
				return ReplicationConfiguration{}, ErrInvalidParameter
			}
			destinations = append(destinations, ReplicationDestination{
				Region:     region,
				RegistryID: registryID,
			})
		}

		repositoryFilters := make([]RepositoryFilter, 0, len(rule.RepositoryFilters))
		for _, filter := range rule.RepositoryFilters {
			name := strings.TrimSpace(filter.Filter)
			if name == "" {
				return ReplicationConfiguration{}, ErrInvalidParameter
			}
			filterType := strings.ToUpper(strings.TrimSpace(filter.FilterType))
			if filterType == "" {
				filterType = "PREFIX_MATCH"
			}
			if filterType != "PREFIX_MATCH" {
				return ReplicationConfiguration{}, ErrInvalidParameter
			}
			repositoryFilters = append(repositoryFilters, RepositoryFilter{
				Filter:     name,
				FilterType: filterType,
			})
		}

		normalizedRules = append(normalizedRules, ReplicationRule{
			Destinations:      destinations,
			RepositoryFilters: repositoryFilters,
		})
	}
	return ReplicationConfiguration{Rules: normalizedRules}, nil
}

func cloneRegistryScanningConfiguration(configuration RegistryScanningConfiguration) RegistryScanningConfiguration {
	rules := make([]RegistryScanningRule, 0, len(configuration.Rules))
	for _, rule := range configuration.Rules {
		filters := make([]ScanningRepositoryFilter, 0, len(rule.RepositoryFilters))
		for _, filter := range rule.RepositoryFilters {
			filters = append(filters, ScanningRepositoryFilter{
				Filter:     filter.Filter,
				FilterType: filter.FilterType,
			})
		}
		rules = append(rules, RegistryScanningRule{
			RepositoryFilters: filters,
			ScanFrequency:     rule.ScanFrequency,
		})
	}
	return RegistryScanningConfiguration{
		ScanType: configuration.ScanType,
		Rules:    rules,
	}
}

func cloneReplicationConfiguration(configuration ReplicationConfiguration) ReplicationConfiguration {
	rules := make([]ReplicationRule, 0, len(configuration.Rules))
	for _, rule := range configuration.Rules {
		destinations := make([]ReplicationDestination, 0, len(rule.Destinations))
		for _, destination := range rule.Destinations {
			destinations = append(destinations, ReplicationDestination{
				Region:     destination.Region,
				RegistryID: destination.RegistryID,
			})
		}
		repositoryFilters := make([]RepositoryFilter, 0, len(rule.RepositoryFilters))
		for _, filter := range rule.RepositoryFilters {
			repositoryFilters = append(repositoryFilters, RepositoryFilter{
				Filter:     filter.Filter,
				FilterType: filter.FilterType,
			})
		}
		rules = append(rules, ReplicationRule{
			Destinations:      destinations,
			RepositoryFilters: repositoryFilters,
		})
	}
	return ReplicationConfiguration{Rules: rules}
}

func clonePullThroughRule(rule *PullThroughCacheRule) PullThroughCacheRule {
	if rule == nil {
		return PullThroughCacheRule{}
	}
	return PullThroughCacheRule{
		CreatedAt:           rule.CreatedAt,
		CredentialArn:       rule.CredentialArn,
		ECRRepositoryPrefix: rule.ECRRepositoryPrefix,
		RegistryID:          rule.RegistryID,
		UpdatedAt:           rule.UpdatedAt,
		UpstreamRegistry:    rule.UpstreamRegistry,
		UpstreamRegistryURL: rule.UpstreamRegistryURL,
	}
}

func inferUpstreamRegistry(upstreamRegistryURL string) string {
	host := strings.TrimSpace(upstreamRegistryURL)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimSuffix(host, "/")
	host = strings.ToLower(host)

	switch {
	case host == "public.ecr.aws":
		return "ecr-public"
	case host == "registry-1.docker.io" || host == "index.docker.io" || host == "docker.io":
		return "docker-hub"
	case host == "quay.io":
		return "quay"
	case host == "registry.k8s.io":
		return "k8s"
	case host == "ghcr.io":
		return "github-container-registry"
	case strings.HasSuffix(host, ".azurecr.io"):
		return "azure-container-registry"
	case host == "registry.gitlab.com":
		return "gitlab-container-registry"
	default:
		return ""
	}
}

func isValidUpstreamRegistry(upstreamRegistry string) bool {
	switch strings.TrimSpace(upstreamRegistry) {
	case "ecr-public", "docker-hub", "quay", "k8s", "github-container-registry", "azure-container-registry", "gitlab-container-registry":
		return true
	default:
		return false
	}
}

func normalizeTemplatePrefix(prefix string) (string, error) {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return "", ErrInvalidParameter
	}
	if strings.EqualFold(trimmed, "ROOT") {
		return "ROOT", nil
	}
	if !repositoryNamePattern.MatchString(trimmed) {
		return "", ErrInvalidParameter
	}
	return trimmed, nil
}

func normalizeAppliedFor(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, ErrInvalidParameter
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		value := strings.ToUpper(strings.TrimSpace(raw))
		switch value {
		case "REPLICATION", "PULL_THROUGH_CACHE":
		default:
			return nil, ErrInvalidParameter
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil, ErrInvalidParameter
	}
	return out, nil
}

func normalizeRepositoryCreationTemplateInput(input RepositoryCreationTemplateInput, requireAppliedFor bool) (RepositoryCreationTemplateInput, error) {
	prefix, err := normalizeTemplatePrefix(input.Prefix)
	if err != nil {
		return RepositoryCreationTemplateInput{}, err
	}
	input.Prefix = prefix

	if requireAppliedFor && !input.AppliedForSet {
		return RepositoryCreationTemplateInput{}, ErrInvalidParameter
	}
	if input.AppliedForSet {
		appliedFor, err := normalizeAppliedFor(input.AppliedFor)
		if err != nil {
			return RepositoryCreationTemplateInput{}, err
		}
		input.AppliedFor = appliedFor
	}

	if input.CustomRoleArn != nil {
		value := strings.TrimSpace(*input.CustomRoleArn)
		input.CustomRoleArn = &value
	}
	if input.Description != nil {
		value := strings.TrimSpace(*input.Description)
		input.Description = &value
	}
	if input.EncryptionType != nil {
		value, err := normalizeEncryptionType(*input.EncryptionType)
		if err != nil {
			return RepositoryCreationTemplateInput{}, err
		}
		input.EncryptionType = &value
	}
	if input.KMSKey != nil {
		value := strings.TrimSpace(*input.KMSKey)
		input.KMSKey = &value
	}
	if input.ImageTagMutability != nil {
		value, err := normalizeImageTagMutability(*input.ImageTagMutability)
		if err != nil {
			return RepositoryCreationTemplateInput{}, err
		}
		input.ImageTagMutability = &value
	}
	if input.LifecyclePolicy != nil {
		value := strings.TrimSpace(*input.LifecyclePolicy)
		input.LifecyclePolicy = &value
	}
	if input.RepositoryPolicy != nil {
		value := strings.TrimSpace(*input.RepositoryPolicy)
		input.RepositoryPolicy = &value
	}
	if input.ResourceTagsSet {
		tags, err := normalizeTags(input.ResourceTags)
		if err != nil {
			return RepositoryCreationTemplateInput{}, err
		}
		input.ResourceTags = tags
	}
	return input, nil
}

func normalizeSigningConfiguration(configuration SigningConfiguration) (SigningConfiguration, error) {
	if len(configuration.Rules) == 0 || len(configuration.Rules) > 10 {
		return SigningConfiguration{}, ErrInvalidParameter
	}

	rules := make([]SigningRule, 0, len(configuration.Rules))
	for _, rule := range configuration.Rules {
		signingProfileArn := strings.TrimSpace(rule.SigningProfileArn)
		if signingProfileArn == "" {
			return SigningConfiguration{}, ErrInvalidParameter
		}

		repositoryFilters := make([]SigningRepositoryFilter, 0, len(rule.RepositoryFilters))
		for _, filter := range rule.RepositoryFilters {
			value := strings.TrimSpace(filter.Filter)
			if value == "" {
				return SigningConfiguration{}, ErrInvalidParameter
			}
			filterType := strings.ToUpper(strings.TrimSpace(filter.FilterType))
			if filterType == "" {
				filterType = "WILDCARD_MATCH"
			}
			if filterType != "WILDCARD_MATCH" {
				return SigningConfiguration{}, ErrInvalidParameter
			}
			repositoryFilters = append(repositoryFilters, SigningRepositoryFilter{
				Filter:     value,
				FilterType: filterType,
			})
		}

		rules = append(rules, SigningRule{
			SigningProfileArn: signingProfileArn,
			RepositoryFilters: repositoryFilters,
		})
	}
	return SigningConfiguration{Rules: rules}, nil
}

func signingRuleMatchesRepository(rule SigningRule, repositoryName string) bool {
	if len(rule.RepositoryFilters) == 0 {
		return true
	}
	for _, filter := range rule.RepositoryFilters {
		if wildcardMatch(filter.Filter, repositoryName) {
			return true
		}
	}
	return false
}

func wildcardMatch(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	quoted := regexp.QuoteMeta(pattern)
	quoted = strings.ReplaceAll(quoted, `\*`, ".*")
	re, err := regexp.Compile("^" + quoted + "$")
	if err != nil {
		return false
	}
	return re.MatchString(value)
}

func cloneRepositoryCreationTemplate(template *RepositoryCreationTemplate) RepositoryCreationTemplate {
	if template == nil {
		return RepositoryCreationTemplate{}
	}
	return RepositoryCreationTemplate{
		AppliedFor:         append([]string(nil), template.AppliedFor...),
		CreatedAt:          template.CreatedAt,
		CustomRoleArn:      template.CustomRoleArn,
		Description:        template.Description,
		EncryptionType:     template.EncryptionType,
		KMSKey:             template.KMSKey,
		ImageTagMutability: template.ImageTagMutability,
		LifecyclePolicy:    template.LifecyclePolicy,
		Prefix:             template.Prefix,
		RepositoryPolicy:   template.RepositoryPolicy,
		ResourceTags:       cloneStringMap(template.ResourceTags),
		UpdatedAt:          template.UpdatedAt,
	}
}

func cloneSigningConfiguration(configuration *SigningConfiguration) SigningConfiguration {
	if configuration == nil {
		return SigningConfiguration{}
	}
	rules := make([]SigningRule, 0, len(configuration.Rules))
	for _, rule := range configuration.Rules {
		repositoryFilters := make([]SigningRepositoryFilter, 0, len(rule.RepositoryFilters))
		for _, filter := range rule.RepositoryFilters {
			repositoryFilters = append(repositoryFilters, SigningRepositoryFilter{
				Filter:     filter.Filter,
				FilterType: filter.FilterType,
			})
		}
		rules = append(rules, SigningRule{
			SigningProfileArn: rule.SigningProfileArn,
			RepositoryFilters: repositoryFilters,
		})
	}
	return SigningConfiguration{Rules: rules}
}

func cloneRepository(repository *Repository) Repository {
	if repository == nil {
		return Repository{}
	}
	return Repository{
		RepositoryName:          repository.RepositoryName,
		RepositoryArn:           repository.RepositoryArn,
		RepositoryURI:           repository.RepositoryURI,
		RegistryID:              repository.RegistryID,
		CreatedAt:               repository.CreatedAt,
		ImageTagMutability:      repository.ImageTagMutability,
		ImageScanningScanOnPush: repository.ImageScanningScanOnPush,
		EncryptionType:          repository.EncryptionType,
		KMSKey:                  repository.KMSKey,
		PolicyText:              repository.PolicyText,
		Tags:                    cloneStringMap(repository.Tags),
	}
}

func cloneLayer(layer *Layer) Layer {
	if layer == nil {
		return Layer{}
	}
	return Layer{
		LayerDigest:       layer.LayerDigest,
		LayerAvailability: layer.LayerAvailability,
		LayerSize:         layer.LayerSize,
		MediaType:         layer.MediaType,
		Data:              append([]byte(nil), layer.Data...),
		CreatedAt:         layer.CreatedAt,
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
