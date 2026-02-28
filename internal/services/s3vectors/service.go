package s3vectors

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrVectorBucketExists   = errors.New("vector bucket already exists")
	ErrVectorBucketNotFound = errors.New("vector bucket not found")
	ErrIndexExists          = errors.New("index already exists")
	ErrIndexNotFound        = errors.New("index not found")
	ErrIndexAmbiguous       = errors.New("index name is ambiguous")
	ErrPolicyNotFound       = errors.New("policy not found")
	ErrInvalidNextToken     = errors.New("invalid next token")
)

type VectorBucket struct {
	Arn                     string
	Name                    string
	CreatedAt               time.Time
	Region                  string
	EncryptionConfiguration string
	MetadataConfiguration   string
}

type VectorIndex struct {
	Arn                   string
	Name                  string
	VectorBucketName      string
	Dimension             int
	DistanceMetric        string
	CreatedAt             time.Time
	MetadataConfiguration string
	Vectors               map[string]*VectorEntry
}

type VectorEntry struct {
	Key       string
	Data      []float32
	Metadata  map[string]string
	CreatedAt time.Time
}

type Service struct {
	mu       sync.RWMutex
	buckets  map[string]*VectorBucket
	indexes  map[string]map[string]*VectorIndex
	policies map[string]string
	tags     map[string]map[string]string
}

func NewService() *Service {
	return &Service{
		buckets:  map[string]*VectorBucket{},
		indexes:  map[string]map[string]*VectorIndex{},
		policies: map[string]string{},
		tags:     map[string]map[string]string{},
	}
}

func (s *Service) CreateVectorBucket(bucket *VectorBucket) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.buckets[bucket.Name]; exists {
		return ErrVectorBucketExists
	}
	s.buckets[bucket.Name] = bucket
	if _, ok := s.indexes[bucket.Name]; !ok {
		s.indexes[bucket.Name] = map[string]*VectorIndex{}
	}
	return nil
}

func (s *Service) GetVectorBucket(name string) (*VectorBucket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bucket, ok := s.buckets[name]
	if !ok {
		return nil, ErrVectorBucketNotFound
	}
	return bucket, nil
}

func (s *Service) DeleteVectorBucket(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.buckets[name]; !ok {
		return ErrVectorBucketNotFound
	}
	delete(s.buckets, name)
	delete(s.indexes, name)
	delete(s.policies, name)
	return nil
}

func (s *Service) ListVectorBuckets() []*VectorBucket {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*VectorBucket, 0, len(s.buckets))
	for _, bucket := range s.buckets {
		out = append(out, bucket)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) CreateIndex(bucketName string, index *VectorIndex) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.buckets[bucketName]; !ok {
		return ErrVectorBucketNotFound
	}
	bucketIndexes, ok := s.indexes[bucketName]
	if !ok {
		bucketIndexes = map[string]*VectorIndex{}
		s.indexes[bucketName] = bucketIndexes
	}
	if _, exists := bucketIndexes[index.Name]; exists {
		return ErrIndexExists
	}
	if index.Vectors == nil {
		index.Vectors = map[string]*VectorEntry{}
	}
	bucketIndexes[index.Name] = index
	return nil
}

func (s *Service) GetIndex(bucketName, name string) (*VectorIndex, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bucketIndexes, ok := s.indexes[bucketName]
	if !ok {
		return nil, ErrVectorBucketNotFound
	}
	index, ok := bucketIndexes[name]
	if !ok {
		return nil, ErrIndexNotFound
	}
	return index, nil
}

func (s *Service) DeleteIndex(bucketName, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	bucketIndexes, ok := s.indexes[bucketName]
	if !ok {
		return ErrVectorBucketNotFound
	}
	if _, ok := bucketIndexes[name]; !ok {
		return ErrIndexNotFound
	}
	delete(bucketIndexes, name)
	return nil
}

func (s *Service) ListIndexes(bucketName string) ([]*VectorIndex, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bucketIndexes, ok := s.indexes[bucketName]
	if !ok {
		return nil, ErrVectorBucketNotFound
	}
	out := make([]*VectorIndex, 0, len(bucketIndexes))
	for _, index := range bucketIndexes {
		out = append(out, index)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (s *Service) PutVectorBucketPolicy(bucketName, policy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.buckets[bucketName]; !ok {
		return ErrVectorBucketNotFound
	}
	s.policies[bucketName] = policy
	return nil
}

func (s *Service) GetVectorBucketPolicy(bucketName string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.buckets[bucketName]; !ok {
		return "", ErrVectorBucketNotFound
	}
	policy, ok := s.policies[bucketName]
	if !ok {
		return "", ErrPolicyNotFound
	}
	return policy, nil
}

func (s *Service) DeleteVectorBucketPolicy(bucketName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.buckets[bucketName]; !ok {
		return ErrVectorBucketNotFound
	}
	if _, ok := s.policies[bucketName]; !ok {
		return ErrPolicyNotFound
	}
	delete(s.policies, bucketName)
	return nil
}

func (s *Service) TagResource(resourceArn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tags[resourceArn]
	if !ok {
		existing = map[string]string{}
		s.tags[resourceArn] = existing
	}
	for key, value := range tags {
		existing[key] = value
	}
}

func (s *Service) UntagResource(resourceArn string, tagKeys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tags[resourceArn]
	if !ok {
		return
	}
	for _, key := range tagKeys {
		delete(existing, key)
	}
}

func (s *Service) ListTags(resourceArn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	existing, ok := s.tags[resourceArn]
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(existing))
	for key, value := range existing {
		out[key] = value
	}
	return out
}

func (s *Service) FindIndexByName(indexName string) (string, *VectorIndex, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var bucketName string
	var found *VectorIndex
	for name, bucketIndexes := range s.indexes {
		index, ok := bucketIndexes[indexName]
		if !ok {
			continue
		}
		if found != nil {
			return "", nil, ErrIndexAmbiguous
		}
		bucketName = name
		found = index
	}
	if found == nil {
		return "", nil, ErrIndexNotFound
	}
	return bucketName, found, nil
}

func (s *Service) PutVectors(bucketName, indexName string, entries []*VectorEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	bucketIndexes, ok := s.indexes[bucketName]
	if !ok {
		return ErrVectorBucketNotFound
	}
	index, ok := bucketIndexes[indexName]
	if !ok {
		return ErrIndexNotFound
	}
	if index.Vectors == nil {
		index.Vectors = map[string]*VectorEntry{}
	}
	for _, entry := range entries {
		index.Vectors[entry.Key] = entry
	}
	return nil
}

func (s *Service) GetVectors(bucketName, indexName string, keys []string) ([]*VectorEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bucketIndexes, ok := s.indexes[bucketName]
	if !ok {
		return nil, ErrVectorBucketNotFound
	}
	index, ok := bucketIndexes[indexName]
	if !ok {
		return nil, ErrIndexNotFound
	}
	out := make([]*VectorEntry, 0, len(keys))
	for _, key := range keys {
		entry, ok := index.Vectors[key]
		if !ok {
			continue
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	return out, nil
}

func (s *Service) DeleteVectors(bucketName, indexName string, keys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	bucketIndexes, ok := s.indexes[bucketName]
	if !ok {
		return ErrVectorBucketNotFound
	}
	index, ok := bucketIndexes[indexName]
	if !ok {
		return ErrIndexNotFound
	}
	for _, key := range keys {
		delete(index.Vectors, key)
	}
	return nil
}

func (s *Service) ListVectors(bucketName, indexName string) ([]*VectorEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bucketIndexes, ok := s.indexes[bucketName]
	if !ok {
		return nil, ErrVectorBucketNotFound
	}
	index, ok := bucketIndexes[indexName]
	if !ok {
		return nil, ErrIndexNotFound
	}
	out := make([]*VectorEntry, 0, len(index.Vectors))
	for _, entry := range index.Vectors {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	return out, nil
}

func ParseNextToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	if strings.HasPrefix(token, "token-") {
		token = strings.TrimPrefix(token, "token-")
		if token == "" {
			return 0, ErrInvalidNextToken
		}
	}
	var idx int
	for _, ch := range token {
		if ch < '0' || ch > '9' {
			return 0, ErrInvalidNextToken
		}
		idx = idx*10 + int(ch-'0')
	}
	return idx, nil
}
