package dsql

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrNotFound         = errors.New("resource not found")
	ErrAlreadyExists    = errors.New("resource already exists")
	ErrConflict         = errors.New("resource conflict")
	ErrPolicyNotFound   = errors.New("cluster policy not found")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"
)

var clusterIDPattern = regexp.MustCompile(`^[a-z0-9]{26}$`)

type EncryptionDetails struct {
	EncryptionStatus string `json:"encryptionStatus,omitempty"`
	EncryptionType   string `json:"encryptionType,omitempty"`
	KmsKeyARN        string `json:"kmsKeyArn,omitempty"`
}

type MultiRegionProperties struct {
	WitnessRegion string `json:"witnessRegion,omitempty"`
}

type Cluster struct {
	ARN                       string                 `json:"arn"`
	CreationTime              time.Time              `json:"creationTime"`
	DeletionProtectionEnabled bool                   `json:"deletionProtectionEnabled"`
	Endpoint                  string                 `json:"endpoint,omitempty"`
	Identifier                string                 `json:"identifier"`
	KmsEncryptionKey          string                 `json:"kmsEncryptionKey,omitempty"`
	EncryptionDetails         *EncryptionDetails     `json:"encryptionDetails,omitempty"`
	MultiRegionProperties     *MultiRegionProperties `json:"multiRegionProperties,omitempty"`
	Status                    string                 `json:"status"`
	Tags                      map[string]string      `json:"tags,omitempty"`
}

type ClusterSummary struct {
	ARN        string `json:"arn"`
	Identifier string `json:"identifier"`
}

type CreateClusterInput struct {
	Identifier                string
	ClientToken               string
	DeletionProtectionEnabled *bool
	KmsEncryptionKey          string
	MultiRegionProperties     *MultiRegionProperties
	Tags                      map[string]string
}

type ListClustersInput struct {
	MaxResults int
	NextToken  string
}

type UpdateClusterInput struct {
	Identifier                string
	ClientToken               string
	DeletionProtectionEnabled *bool
}

type DeleteClusterInput struct {
	Identifier  string
	ClientToken string
}

type ClusterPolicy struct {
	Policy        string `json:"policy"`
	PolicyVersion string `json:"policyVersion"`
}

type PutClusterPolicyInput struct {
	Identifier  string
	ClientToken string
	Policy      string
}

type GetClusterPolicyInput struct {
	Identifier string
}

type DeleteClusterPolicyInput struct {
	Identifier            string
	ClientToken           string
	ExpectedPolicyVersion string
}

type TagResourceInput struct {
	ResourceARN string
	Tags        map[string]string
}

type ListTagsForResourceInput struct {
	ResourceARN string
}

type UntagResourceInput struct {
	ResourceARN string
	TagKeys     []string
}

type Service struct {
	mu sync.Mutex

	sequence             uint64
	clusters             map[string]*Cluster
	createTokenToCluster map[string]string
	clusterPolicies      map[string]*ClusterPolicy
}

func NewService() *Service {
	return &Service{
		clusters:             map[string]*Cluster{},
		createTokenToCluster: map[string]string{},
		clusterPolicies:      map[string]*ClusterPolicy{},
	}
}

func (s *Service) CreateCluster(input CreateClusterInput) (Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clientToken := strings.TrimSpace(input.ClientToken)
	if clientToken != "" {
		if identifier, ok := s.createTokenToCluster[clientToken]; ok {
			if existing, ok := s.clusters[identifier]; ok {
				return cloneCluster(existing), nil
			}
		}
	}

	identifier := strings.TrimSpace(input.Identifier)
	if identifier == "" {
		s.sequence++
		identifier = generatedClusterIdentifier(s.sequence)
	}
	if !clusterIDPattern.MatchString(identifier) {
		return Cluster{}, ErrInvalidParameter
	}
	if _, exists := s.clusters[identifier]; exists {
		return Cluster{}, ErrAlreadyExists
	}

	deletionProtection := true
	if input.DeletionProtectionEnabled != nil {
		deletionProtection = *input.DeletionProtectionEnabled
	}

	now := time.Now().UTC()
	cluster := &Cluster{
		ARN:                       clusterARN(identifier),
		CreationTime:              now,
		DeletionProtectionEnabled: deletionProtection,
		Endpoint:                  fmt.Sprintf("%s.%s.dsql.amazonaws.com", identifier, DefaultRegion),
		Identifier:                identifier,
		KmsEncryptionKey:          strings.TrimSpace(input.KmsEncryptionKey),
		MultiRegionProperties:     cloneMultiRegion(input.MultiRegionProperties),
		Status:                    "ACTIVE",
		Tags:                      cloneStringMap(input.Tags),
	}
	if cluster.KmsEncryptionKey != "" {
		cluster.EncryptionDetails = &EncryptionDetails{
			EncryptionStatus: "ENABLED",
			EncryptionType:   "CUSTOMER_MANAGED_KMS_KEY",
			KmsKeyARN:        cluster.KmsEncryptionKey,
		}
	} else {
		cluster.EncryptionDetails = &EncryptionDetails{
			EncryptionStatus: "ENABLED",
			EncryptionType:   "AWS_OWNED_KMS_KEY",
		}
	}

	s.clusters[identifier] = cluster
	if clientToken != "" {
		s.createTokenToCluster[clientToken] = identifier
	}
	return cloneCluster(cluster), nil
}

func (s *Service) GetCluster(identifier string) (Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	identifier = strings.TrimSpace(identifier)
	if !clusterIDPattern.MatchString(identifier) {
		return Cluster{}, ErrInvalidParameter
	}
	cluster, ok := s.clusters[identifier]
	if !ok {
		return Cluster{}, ErrNotFound
	}
	return cloneCluster(cluster), nil
}

func (s *Service) ListClusters(input ListClustersInput) ([]ClusterSummary, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	maxResults := input.MaxResults
	if maxResults == 0 {
		maxResults = 100
	}
	if maxResults < 1 || maxResults > 100 {
		return nil, "", ErrInvalidParameter
	}

	identifiers := make([]string, 0, len(s.clusters))
	for identifier := range s.clusters {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)

	start := 0
	nextToken := strings.TrimSpace(input.NextToken)
	if nextToken != "" {
		offset, err := strconv.Atoi(nextToken)
		if err != nil || offset < 0 || offset > len(identifiers) {
			return nil, "", ErrInvalidParameter
		}
		start = offset
	}

	end := start + maxResults
	if end > len(identifiers) {
		end = len(identifiers)
	}

	summaries := make([]ClusterSummary, 0, end-start)
	for _, identifier := range identifiers[start:end] {
		cluster := s.clusters[identifier]
		summaries = append(summaries, ClusterSummary{
			ARN:        cluster.ARN,
			Identifier: cluster.Identifier,
		})
	}

	outNextToken := ""
	if end < len(identifiers) {
		outNextToken = strconv.Itoa(end)
	}
	return summaries, outNextToken, nil
}

func (s *Service) UpdateCluster(input UpdateClusterInput) (Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	identifier := strings.TrimSpace(input.Identifier)
	if !clusterIDPattern.MatchString(identifier) {
		return Cluster{}, ErrInvalidParameter
	}
	cluster, ok := s.clusters[identifier]
	if !ok {
		return Cluster{}, ErrNotFound
	}

	if input.DeletionProtectionEnabled != nil {
		cluster.DeletionProtectionEnabled = *input.DeletionProtectionEnabled
	}
	cluster.Status = "ACTIVE"
	return cloneCluster(cluster), nil
}

func (s *Service) DeleteCluster(input DeleteClusterInput) (Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	identifier := strings.TrimSpace(input.Identifier)
	if !clusterIDPattern.MatchString(identifier) {
		return Cluster{}, ErrInvalidParameter
	}
	cluster, ok := s.clusters[identifier]
	if !ok {
		return Cluster{}, ErrNotFound
	}
	if cluster.DeletionProtectionEnabled {
		return Cluster{}, ErrConflict
	}

	deleted := cloneCluster(cluster)
	deleted.Status = "DELETING"
	delete(s.clusters, identifier)
	delete(s.clusterPolicies, identifier)
	return deleted, nil
}

func (s *Service) GetVpcEndpointServiceName(identifier string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	identifier = strings.TrimSpace(identifier)
	if !clusterIDPattern.MatchString(identifier) {
		return "", ErrInvalidParameter
	}
	if _, ok := s.clusters[identifier]; !ok {
		return "", ErrNotFound
	}
	return fmt.Sprintf("com.amazonaws.%s.dsql", DefaultRegion), nil
}

func (s *Service) PutClusterPolicy(input PutClusterPolicyInput) (ClusterPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	identifier := strings.TrimSpace(input.Identifier)
	if !clusterIDPattern.MatchString(identifier) {
		return ClusterPolicy{}, ErrInvalidParameter
	}
	if _, ok := s.clusters[identifier]; !ok {
		return ClusterPolicy{}, ErrNotFound
	}

	policyStr := strings.TrimSpace(input.Policy)
	if policyStr == "" {
		return ClusterPolicy{}, ErrInvalidParameter
	}

	current := s.clusterPolicies[identifier]
	versionNum := 1
	if current != nil {
		parsed, err := strconv.Atoi(strings.TrimPrefix(current.PolicyVersion, "v"))
		if err != nil || parsed < 1 {
			versionNum = 1
		} else {
			versionNum = parsed + 1
		}
	}
	policy := &ClusterPolicy{
		Policy:        policyStr,
		PolicyVersion: fmt.Sprintf("v%d", versionNum),
	}
	s.clusterPolicies[identifier] = policy
	return cloneClusterPolicy(policy), nil
}

func (s *Service) GetClusterPolicy(input GetClusterPolicyInput) (ClusterPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	identifier := strings.TrimSpace(input.Identifier)
	if !clusterIDPattern.MatchString(identifier) {
		return ClusterPolicy{}, ErrInvalidParameter
	}
	if _, ok := s.clusters[identifier]; !ok {
		return ClusterPolicy{}, ErrNotFound
	}
	policy := s.clusterPolicies[identifier]
	if policy == nil {
		return ClusterPolicy{}, ErrPolicyNotFound
	}
	return cloneClusterPolicy(policy), nil
}

func (s *Service) DeleteClusterPolicy(input DeleteClusterPolicyInput) (ClusterPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	identifier := strings.TrimSpace(input.Identifier)
	if !clusterIDPattern.MatchString(identifier) {
		return ClusterPolicy{}, ErrInvalidParameter
	}
	if _, ok := s.clusters[identifier]; !ok {
		return ClusterPolicy{}, ErrNotFound
	}
	current := s.clusterPolicies[identifier]
	if current == nil {
		return ClusterPolicy{}, ErrPolicyNotFound
	}
	expectedVersion := strings.TrimSpace(input.ExpectedPolicyVersion)
	if expectedVersion != "" && expectedVersion != current.PolicyVersion {
		return ClusterPolicy{}, ErrConflict
	}
	deleted := cloneClusterPolicy(current)
	delete(s.clusterPolicies, identifier)
	return deleted, nil
}

func (s *Service) TagResource(input TagResourceInput) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, err := s.resolveClusterByARN(input.ResourceARN)
	if err != nil {
		return nil, err
	}
	if cluster.Tags == nil {
		cluster.Tags = map[string]string{}
	}
	for k, v := range input.Tags {
		key := strings.TrimSpace(k)
		if key == "" {
			return nil, ErrInvalidParameter
		}
		cluster.Tags[key] = v
	}
	return cloneStringMap(cluster.Tags), nil
}

func (s *Service) ListTagsForResource(input ListTagsForResourceInput) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, err := s.resolveClusterByARN(input.ResourceARN)
	if err != nil {
		return nil, err
	}
	return cloneStringMap(cluster.Tags), nil
}

func (s *Service) UntagResource(input UntagResourceInput) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, err := s.resolveClusterByARN(input.ResourceARN)
	if err != nil {
		return nil, err
	}
	for _, rawKey := range input.TagKeys {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			return nil, ErrInvalidParameter
		}
		delete(cluster.Tags, key)
	}
	return cloneStringMap(cluster.Tags), nil
}

func generatedClusterIdentifier(sequence uint64) string {
	return fmt.Sprintf("dsql%022x", sequence)
}

func clusterARN(identifier string) string {
	return fmt.Sprintf("arn:aws:dsql:%s:%s:cluster/%s", DefaultRegion, DefaultAccountID, identifier)
}

func cloneCluster(in *Cluster) Cluster {
	if in == nil {
		return Cluster{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	out.MultiRegionProperties = cloneMultiRegion(in.MultiRegionProperties)
	if in.EncryptionDetails != nil {
		enc := *in.EncryptionDetails
		out.EncryptionDetails = &enc
	}
	return out
}

func cloneMultiRegion(in *MultiRegionProperties) *MultiRegionProperties {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if strings.TrimSpace(k) == "" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneClusterPolicy(in *ClusterPolicy) ClusterPolicy {
	if in == nil {
		return ClusterPolicy{}
	}
	return *in
}

func (s *Service) resolveClusterByARN(resourceARN string) (*Cluster, error) {
	arn := strings.TrimSpace(resourceARN)
	if arn == "" {
		return nil, ErrInvalidParameter
	}
	prefix := "arn:aws:dsql:"
	if !strings.HasPrefix(arn, prefix) {
		return nil, ErrInvalidParameter
	}
	parts := strings.SplitN(arn, ":cluster/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return nil, ErrInvalidParameter
	}
	identifier := strings.TrimSpace(parts[1])
	if !clusterIDPattern.MatchString(identifier) {
		return nil, ErrInvalidParameter
	}
	cluster := s.clusters[identifier]
	if cluster == nil {
		return nil, ErrNotFound
	}
	return cluster, nil
}
