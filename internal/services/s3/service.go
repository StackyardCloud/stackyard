package s3

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrBucketExists                       = errors.New("bucket already exists")
	ErrBucketNotFound                     = errors.New("bucket not found")
	ErrObjectNotFound                     = errors.New("object not found")
	ErrBucketNotEmpty                     = errors.New("bucket not empty")
	ErrMetadataConfigurationNotFound      = errors.New("metadata configuration not found")
	ErrMetadataTableConfigurationNotFound = errors.New("metadata table configuration not found")
)

type Bucket struct {
	Name               string                                           `json:"name"`
	CreatedAt          time.Time                                        `json:"created_at"`
	ACL                string                                           `json:"acl"`
	Accelerate         *BucketAccelerateConfiguration                   `json:"accelerate,omitempty"`
	CORS               *CORSConfiguration                               `json:"cors,omitempty"`
	Lifecycle          *LifecycleConfiguration                          `json:"lifecycle,omitempty"`
	Tags               map[string]string                                `json:"tags,omitempty"`
	Website            *WebsiteConfiguration                            `json:"website,omitempty"`
	Logging            *LoggingConfiguration                            `json:"logging,omitempty"`
	Replication        *ReplicationConfiguration                        `json:"replication,omitempty"`
	Policy             string                                           `json:"policy,omitempty"`
	PublicAccessBlock  *PublicAccessBlockConfiguration                  `json:"public_access_block,omitempty"`
	ObjectLock         *ObjectLockConfiguration                         `json:"object_lock,omitempty"`
	Encryption         *BucketEncryptionConfiguration                   `json:"encryption,omitempty"`
	Notifications      *BucketNotificationConfiguration                 `json:"notifications,omitempty"`
	RequestPayment     *BucketRequestPaymentConfiguration               `json:"request_payment,omitempty"`
	OwnershipControls  *BucketOwnershipControls                         `json:"ownership_controls,omitempty"`
	Abac               *BucketAbacConfiguration                         `json:"abac,omitempty"`
	Analytics          map[string]BucketAnalyticsConfiguration          `json:"analytics,omitempty"`
	Metrics            map[string]BucketMetricsConfiguration            `json:"metrics,omitempty"`
	Inventory          map[string]BucketInventoryConfiguration          `json:"inventory,omitempty"`
	IntelligentTiering map[string]BucketIntelligentTieringConfiguration `json:"intelligent_tiering,omitempty"`
	Metadata           *BucketMetadataConfiguration                     `json:"metadata,omitempty"`
	MetadataTable      *BucketMetadataTableConfiguration                `json:"metadata_table,omitempty"`
}

type Object struct {
	Key                  string            `json:"key"`
	Bucket               string            `json:"bucket"`
	SizeBytes            int               `json:"size_bytes"`
	ContentType          string            `json:"content_type"`
	ETag                 string            `json:"etag"`
	StorageClass         string            `json:"storage_class,omitempty"`
	SSEAlgorithm         string            `json:"sse_algorithm,omitempty"`
	SSECustomerAlgorithm string            `json:"sse_customer_algorithm,omitempty"`
	SSECustomerKeyMD5    string            `json:"sse_customer_key_md5,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	Tags                 map[string]string `json:"tags,omitempty"`
	Retention            *ObjectRetention  `json:"retention,omitempty"`
	LegalHold            *ObjectLegalHold  `json:"legal_hold,omitempty"`
	RestoreInProgress    bool              `json:"restore_in_progress,omitempty"`
	RestoreUntil         time.Time         `json:"restore_until,omitempty"`
	PartNumber           int               `json:"part_number,omitempty"`
	UpdatedAt            time.Time         `json:"updated_at"`
	Body                 []byte            `json:"-"`
	ACL                  string            `json:"acl"`
	VersionID            string            `json:"version_id,omitempty"`
	IsLatest             bool              `json:"is_latest,omitempty"`
	DeleteMarker         bool              `json:"delete_marker,omitempty"`
}

type Service struct {
	mu            sync.RWMutex
	buckets       map[string]*bucketState
	uploads       map[string]*multipartUpload
	uploadCounter uint64
}

type bucketState struct {
	bucket     Bucket
	objects    map[string][]Object
	versioning string
}

type CORSConfiguration struct {
	Rules []CORSRule `json:"rules"`
}

type CORSRule struct {
	AllowedOrigins []string `json:"allowed_origins"`
	AllowedMethods []string `json:"allowed_methods"`
	AllowedHeaders []string `json:"allowed_headers,omitempty"`
	ExposeHeaders  []string `json:"expose_headers,omitempty"`
	MaxAgeSeconds  int      `json:"max_age_seconds,omitempty"`
}

type BucketAccelerateConfiguration struct {
	Status string `json:"status,omitempty"`
}

type LifecycleConfiguration struct {
	Rules []LifecycleRule `json:"rules"`
}

type LifecycleRule struct {
	ID     string `json:"id,omitempty"`
	Status string `json:"status"`
	Prefix string `json:"prefix,omitempty"`
}

type WebsiteConfiguration struct {
	IndexDocument string `json:"index_document,omitempty"`
	ErrorDocument string `json:"error_document,omitempty"`
}

type LoggingConfiguration struct {
	TargetBucket string `json:"target_bucket,omitempty"`
	TargetPrefix string `json:"target_prefix,omitempty"`
}

type BucketRequestPaymentConfiguration struct {
	Payer string `json:"payer,omitempty"`
}

type BucketOwnershipControls struct {
	Rules []OwnershipControlRule `json:"rules,omitempty"`
}

type OwnershipControlRule struct {
	ObjectOwnership string `json:"object_ownership,omitempty"`
}

type BucketAbacConfiguration struct {
	Status string `json:"status,omitempty"`
}

type PublicAccessBlockConfiguration struct {
	BlockPublicAcls       bool `json:"block_public_acls,omitempty"`
	IgnorePublicAcls      bool `json:"ignore_public_acls,omitempty"`
	BlockPublicPolicy     bool `json:"block_public_policy,omitempty"`
	RestrictPublicBuckets bool `json:"restrict_public_buckets,omitempty"`
}

type ObjectRetention struct {
	Mode        string    `json:"mode,omitempty"`
	RetainUntil time.Time `json:"retain_until,omitempty"`
}

type ObjectLegalHold struct {
	Status string `json:"status,omitempty"`
}

type ObjectLockConfiguration struct {
	Enabled          bool              `json:"enabled,omitempty"`
	DefaultRetention *DefaultRetention `json:"default_retention,omitempty"`
}

type DefaultRetention struct {
	Mode  string `json:"mode,omitempty"`
	Days  int    `json:"days,omitempty"`
	Years int    `json:"years,omitempty"`
}

type ReplicationConfiguration struct {
	Role  string            `json:"role,omitempty"`
	Rules []ReplicationRule `json:"rules,omitempty"`
}

type ReplicationRule struct {
	ID                      string `json:"id,omitempty"`
	Status                  string `json:"status"`
	Prefix                  string `json:"prefix,omitempty"`
	DestinationBucket       string `json:"destination_bucket"`
	DeleteMarkerReplication bool   `json:"delete_marker_replication,omitempty"`
}

type BucketEncryptionConfiguration struct {
	Rules []BucketEncryptionRule `json:"rules"`
}

type BucketEncryptionRule struct {
	SSEAlgorithm string `json:"sse_algorithm"`
}

type BucketNotificationConfiguration struct {
	QueueConfigurations []QueueNotificationConfiguration `json:"queue_configurations,omitempty"`
}

type QueueNotificationConfiguration struct {
	ID     string   `json:"id,omitempty"`
	Queue  string   `json:"queue"`
	Events []string `json:"events"`
	Prefix string   `json:"prefix,omitempty"`
	Suffix string   `json:"suffix,omitempty"`
}

type BucketAnalyticsConfiguration struct {
	ID      string `json:"id"`
	Payload string `json:"payload"`
}

type BucketMetricsConfiguration struct {
	ID      string `json:"id"`
	Payload string `json:"payload"`
}

type BucketInventoryConfiguration struct {
	ID      string `json:"id"`
	Payload string `json:"payload"`
	Enabled bool   `json:"enabled"`
}

type BucketIntelligentTieringConfiguration struct {
	ID      string `json:"id"`
	Payload string `json:"payload"`
	Status  string `json:"status"`
}

type BucketMetadataConfiguration struct {
	TableBucketArn  string `json:"table_bucket_arn,omitempty"`
	TableBucketType string `json:"table_bucket_type,omitempty"`
	TableNamespace  string `json:"table_namespace,omitempty"`
	InventoryState  string `json:"inventory_state,omitempty"`
	InventoryTable  string `json:"inventory_table,omitempty"`
	InventoryArn    string `json:"inventory_arn,omitempty"`
	InventoryStatus string `json:"inventory_status,omitempty"`
	JournalDays     int    `json:"journal_days,omitempty"`
	JournalExpireAt string `json:"journal_expiration,omitempty"`
	JournalTable    string `json:"journal_table,omitempty"`
	JournalArn      string `json:"journal_arn,omitempty"`
	JournalStatus   string `json:"journal_status,omitempty"`
}

type BucketMetadataTableConfiguration struct {
	TableBucketArn string `json:"table_bucket_arn,omitempty"`
	TableName      string `json:"table_name,omitempty"`
	TableArn       string `json:"table_arn,omitempty"`
	TableNamespace string `json:"table_namespace,omitempty"`
	Status         string `json:"status,omitempty"`
}

type multipartUpload struct {
	ID                   string
	Bucket               string
	Key                  string
	ContentType          string
	Metadata             map[string]string
	ACL                  string
	StorageClass         string
	SSEAlgorithm         string
	SSECustomerAlgorithm string
	SSECustomerKeyMD5    string
	InitiatedAt          time.Time
	Parts                map[int]Object
}

type MultipartUploadInfo struct {
	ID           string
	Bucket       string
	Key          string
	StorageClass string
	InitiatedAt  time.Time
}

func NewService() *Service {
	return &Service{
		buckets: make(map[string]*bucketState),
		uploads: make(map[string]*multipartUpload),
	}
}

func (s *Service) CreateBucket(name string) (Bucket, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Bucket{}, ErrBucketNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.buckets[name]; exists {
		return Bucket{}, ErrBucketExists
	}

	bucket := Bucket{
		Name:               name,
		CreatedAt:          time.Now().UTC(),
		ACL:                "private",
		Analytics:          map[string]BucketAnalyticsConfiguration{},
		Metrics:            map[string]BucketMetricsConfiguration{},
		Inventory:          map[string]BucketInventoryConfiguration{},
		IntelligentTiering: map[string]BucketIntelligentTieringConfiguration{},
	}
	s.buckets[name] = &bucketState{
		bucket:     bucket,
		objects:    make(map[string][]Object),
		versioning: "Suspended",
	}

	return bucket, nil
}

func (s *Service) ListBuckets() []Bucket {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Bucket, 0, len(s.buckets))
	for _, state := range s.buckets {
		out = append(out, state.bucket)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) PutObject(bucket, key, contentType string, body []byte, metadata map[string]string, acl string, storageClass string) (Object, error) {
	return s.putObjectWithVersionID(bucket, key, contentType, body, metadata, acl, storageClass, "")
}

func (s *Service) PutObjectWithVersionID(bucket, key, contentType string, body []byte, metadata map[string]string, acl string, storageClass string, versionID string) (Object, error) {
	return s.putObjectWithVersionID(bucket, key, contentType, body, metadata, acl, storageClass, versionID)
}

func (s *Service) putObjectWithVersionID(bucket, key, contentType string, body []byte, metadata map[string]string, acl string, storageClass string, versionID string) (Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.buckets[bucket]
	if !ok {
		return Object{}, ErrBucketNotFound
	}

	if acl == "" {
		acl = "private"
	}
	if storageClass == "" {
		storageClass = "STANDARD"
	}
	if versionID == "" && state.versioning == "Enabled" {
		versionID = newVersionID()
	}
	obj := Object{
		Key:               key,
		Bucket:            bucket,
		SizeBytes:         len(body),
		ContentType:       contentType,
		ETag:              etagFor(body),
		StorageClass:      storageClass,
		Metadata:          metadata,
		Tags:              nil,
		Retention:         nil,
		LegalHold:         nil,
		RestoreInProgress: false,
		RestoreUntil:      time.Time{},
		UpdatedAt:         time.Now().UTC(),
		Body:              append([]byte(nil), body...),
		ACL:               acl,
		VersionID:         versionID,
		IsLatest:          true,
	}
	if obj.Retention == nil && state.bucket.ObjectLock != nil && state.bucket.ObjectLock.Enabled {
		if retention := buildDefaultRetention(state.bucket.ObjectLock.DefaultRetention); retention != nil {
			obj.Retention = retention
		}
	}
	versions := state.objects[key]
	for i := range versions {
		versions[i].IsLatest = false
	}
	state.objects[key] = append([]Object{obj}, versions...)
	return obj, nil
}

func (s *Service) SetObjectEncryption(bucket, key, versionID, sseAlg, sseCustomerAlg, sseCustomerMD5 string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return ErrObjectNotFound
	}
	for i := range versions {
		if versionID == "" {
			if versions[i].IsLatest && !versions[i].DeleteMarker {
				versions[i].SSEAlgorithm = sseAlg
				versions[i].SSECustomerAlgorithm = sseCustomerAlg
				versions[i].SSECustomerKeyMD5 = sseCustomerMD5
				state.objects[key] = versions
				return nil
			}
			continue
		}
		if versions[i].VersionID == versionID {
			if versions[i].DeleteMarker {
				return ErrObjectNotFound
			}
			versions[i].SSEAlgorithm = sseAlg
			versions[i].SSECustomerAlgorithm = sseCustomerAlg
			versions[i].SSECustomerKeyMD5 = sseCustomerMD5
			state.objects[key] = versions
			return nil
		}
	}
	return ErrObjectNotFound
}

func (s *Service) GetObject(bucket, key string) (Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.buckets[bucket]
	if !ok {
		return Object{}, ErrBucketNotFound
	}

	versions, ok := state.objects[key]
	if !ok || len(versions) == 0 {
		return Object{}, ErrObjectNotFound
	}
	for _, obj := range versions {
		if obj.IsLatest && !obj.DeleteMarker {
			obj.Body = append([]byte(nil), obj.Body...)
			return obj, nil
		}
	}
	return Object{}, ErrObjectNotFound
}

func (s *Service) GetObjectVersion(bucket, key, versionID string) (Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return Object{}, ErrBucketNotFound
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return Object{}, ErrObjectNotFound
	}
	for _, obj := range versions {
		if obj.VersionID == versionID {
			if obj.DeleteMarker {
				return Object{}, ErrObjectNotFound
			}
			obj.Body = append([]byte(nil), obj.Body...)
			return obj, nil
		}
	}
	return Object{}, ErrObjectNotFound
}

func (s *Service) RestoreObject(bucket, key, versionID string, days int) (Object, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return Object{}, false, ErrBucketNotFound
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return Object{}, false, ErrObjectNotFound
	}
	if days <= 0 {
		days = 1
	}
	now := time.Now().UTC()
	for i := range versions {
		if versionID == "" {
			if versions[i].IsLatest && !versions[i].DeleteMarker {
				already := !versions[i].RestoreUntil.IsZero() && versions[i].RestoreUntil.After(now) && !versions[i].RestoreInProgress
				versions[i].RestoreInProgress = false
				versions[i].RestoreUntil = now.Add(time.Duration(days) * 24 * time.Hour)
				state.objects[key] = versions
				return versions[i], !already, nil
			}
			continue
		}
		if versions[i].VersionID == versionID {
			if versions[i].DeleteMarker {
				return Object{}, false, ErrObjectNotFound
			}
			already := !versions[i].RestoreUntil.IsZero() && versions[i].RestoreUntil.After(now) && !versions[i].RestoreInProgress
			versions[i].RestoreInProgress = false
			versions[i].RestoreUntil = now.Add(time.Duration(days) * 24 * time.Hour)
			state.objects[key] = versions
			return versions[i], !already, nil
		}
	}
	return Object{}, false, ErrObjectNotFound
}

func (s *Service) GetDeleteMarker(bucket, key, versionID string) (Object, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return Object{}, false
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return Object{}, false
	}
	for _, obj := range versions {
		if obj.VersionID == versionID && obj.DeleteMarker {
			return obj, true
		}
	}
	return Object{}, false
}

func (s *Service) GetLatestDeleteMarker(bucket, key string) (Object, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return Object{}, false
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return Object{}, false
	}
	for _, obj := range versions {
		if obj.IsLatest && obj.DeleteMarker {
			return obj, true
		}
	}
	return Object{}, false
}

func (s *Service) ListObjects(bucket, prefix string) ([]Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}

	out := make([]Object, 0, len(state.objects))
	for key, versions := range state.objects {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		for _, obj := range versions {
			if obj.IsLatest && !obj.DeleteMarker {
				obj.Body = nil
				out = append(out, obj)
				break
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	return out, nil
}

func (s *Service) ListObjectsPaged(bucket, prefix, token, startAfter string, maxKeys int) ([]Object, string, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.buckets[bucket]
	if !ok {
		return nil, "", false, ErrBucketNotFound
	}

	keys := make([]string, 0, len(state.objects))
	for key := range state.objects {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	startIndex := 0
	if token != "" {
		for i, key := range keys {
			if key == token {
				startIndex = i + 1
				break
			}
		}
	} else if startAfter != "" {
		for i, key := range keys {
			if key > startAfter {
				startIndex = i
				break
			}
		}
	}

	if maxKeys <= 0 {
		maxKeys = 1000
	}

	endIndex := startIndex + maxKeys
	if endIndex > len(keys) {
		endIndex = len(keys)
	}

	out := make([]Object, 0, endIndex-startIndex)
	for _, key := range keys[startIndex:endIndex] {
		versions := state.objects[key]
		for _, obj := range versions {
			if obj.IsLatest && !obj.DeleteMarker {
				obj.Body = nil
				out = append(out, obj)
				break
			}
		}
	}

	isTruncated := endIndex < len(keys)
	nextToken := ""
	if isTruncated && len(out) > 0 {
		nextToken = out[len(out)-1].Key
	}

	return out, nextToken, isTruncated, nil
}

func (s *Service) HasBucket(bucket string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.buckets[bucket]
	return ok
}

func (s *Service) GetBucket(bucket string) (Bucket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return Bucket{}, ErrBucketNotFound
	}
	return state.bucket, nil
}

func (s *Service) GetBucketAccelerate(bucket string) (*BucketAccelerateConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.Accelerate, nil
}

func (s *Service) SetBucketAccelerate(bucket string, cfg *BucketAccelerateConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Accelerate = cfg
	return nil
}

func (s *Service) DeleteBucket(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	for _, versions := range state.objects {
		if len(versions) > 0 {
			return ErrBucketNotEmpty
		}
	}
	delete(s.buckets, bucket)
	return nil
}

func (s *Service) GetBucketCORS(bucket string) (*CORSConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.CORS, nil
}

func (s *Service) SetBucketCORS(bucket string, cfg *CORSConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.CORS = cfg
	return nil
}

func (s *Service) DeleteBucketCORS(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.CORS = nil
	return nil
}

func (s *Service) GetBucketLifecycle(bucket string) (*LifecycleConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.Lifecycle, nil
}

func (s *Service) SetBucketLifecycle(bucket string, cfg *LifecycleConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Lifecycle = cfg
	return nil
}

func (s *Service) DeleteBucketLifecycle(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Lifecycle = nil
	return nil
}

func (s *Service) GetBucketTags(bucket string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return copyTags(state.bucket.Tags), nil
}

func (s *Service) SetBucketTags(bucket string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Tags = copyTags(tags)
	return nil
}

func (s *Service) DeleteBucketTags(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Tags = nil
	return nil
}

func (s *Service) GetBucketWebsite(bucket string) (*WebsiteConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.Website, nil
}

func (s *Service) SetBucketWebsite(bucket string, cfg *WebsiteConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Website = cfg
	return nil
}

func (s *Service) DeleteBucketWebsite(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Website = nil
	return nil
}

func (s *Service) GetBucketLogging(bucket string) (*LoggingConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.Logging, nil
}

func (s *Service) SetBucketLogging(bucket string, cfg *LoggingConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Logging = cfg
	return nil
}

func (s *Service) DeleteBucketLogging(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Logging = nil
	return nil
}

func (s *Service) GetBucketRequestPayment(bucket string) (*BucketRequestPaymentConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.RequestPayment, nil
}

func (s *Service) SetBucketRequestPayment(bucket string, cfg *BucketRequestPaymentConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.RequestPayment = cfg
	return nil
}

func (s *Service) GetBucketOwnershipControls(bucket string) (*BucketOwnershipControls, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.OwnershipControls, nil
}

func (s *Service) SetBucketOwnershipControls(bucket string, cfg *BucketOwnershipControls) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.OwnershipControls = cfg
	return nil
}

func (s *Service) DeleteBucketOwnershipControls(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.OwnershipControls = nil
	return nil
}

func (s *Service) GetBucketAbac(bucket string) (*BucketAbacConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.Abac, nil
}

func (s *Service) SetBucketAbac(bucket string, cfg *BucketAbacConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Abac = cfg
	return nil
}

func (s *Service) GetBucketReplication(bucket string) (*ReplicationConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.Replication, nil
}

func (s *Service) SetBucketReplication(bucket string, cfg *ReplicationConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Replication = cfg
	return nil
}

func (s *Service) GetBucketEncryption(bucket string) (*BucketEncryptionConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.Encryption, nil
}

func (s *Service) SetBucketEncryption(bucket string, cfg *BucketEncryptionConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Encryption = cfg
	return nil
}

func (s *Service) DeleteBucketEncryption(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Encryption = nil
	return nil
}

func (s *Service) GetBucketNotifications(bucket string) (*BucketNotificationConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.Notifications, nil
}

func (s *Service) SetBucketNotifications(bucket string, cfg *BucketNotificationConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Notifications = cfg
	return nil
}

func (s *Service) GetBucketAnalyticsConfig(bucket, id string) (BucketAnalyticsConfiguration, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return BucketAnalyticsConfiguration{}, false, ErrBucketNotFound
	}
	cfg, ok := state.bucket.Analytics[id]
	return cfg, ok, nil
}

func (s *Service) ListBucketAnalyticsConfigs(bucket string) ([]BucketAnalyticsConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	out := make([]BucketAnalyticsConfiguration, 0, len(state.bucket.Analytics))
	for _, cfg := range state.bucket.Analytics {
		out = append(out, cfg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Service) SetBucketAnalyticsConfig(bucket string, cfg BucketAnalyticsConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	if state.bucket.Analytics == nil {
		state.bucket.Analytics = map[string]BucketAnalyticsConfiguration{}
	}
	state.bucket.Analytics[cfg.ID] = cfg
	return nil
}

func (s *Service) DeleteBucketAnalyticsConfig(bucket, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	delete(state.bucket.Analytics, id)
	return nil
}

func (s *Service) GetBucketMetricsConfig(bucket, id string) (BucketMetricsConfiguration, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return BucketMetricsConfiguration{}, false, ErrBucketNotFound
	}
	cfg, ok := state.bucket.Metrics[id]
	return cfg, ok, nil
}

func (s *Service) ListBucketMetricsConfigs(bucket string) ([]BucketMetricsConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	out := make([]BucketMetricsConfiguration, 0, len(state.bucket.Metrics))
	for _, cfg := range state.bucket.Metrics {
		out = append(out, cfg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Service) SetBucketMetricsConfig(bucket string, cfg BucketMetricsConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	if state.bucket.Metrics == nil {
		state.bucket.Metrics = map[string]BucketMetricsConfiguration{}
	}
	state.bucket.Metrics[cfg.ID] = cfg
	return nil
}

func (s *Service) DeleteBucketMetricsConfig(bucket, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	delete(state.bucket.Metrics, id)
	return nil
}

func (s *Service) GetBucketInventoryConfig(bucket, id string) (BucketInventoryConfiguration, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return BucketInventoryConfiguration{}, false, ErrBucketNotFound
	}
	cfg, ok := state.bucket.Inventory[id]
	return cfg, ok, nil
}

func (s *Service) ListBucketInventoryConfigs(bucket string) ([]BucketInventoryConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	out := make([]BucketInventoryConfiguration, 0, len(state.bucket.Inventory))
	for _, cfg := range state.bucket.Inventory {
		out = append(out, cfg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Service) SetBucketInventoryConfig(bucket string, cfg BucketInventoryConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	if state.bucket.Inventory == nil {
		state.bucket.Inventory = map[string]BucketInventoryConfiguration{}
	}
	state.bucket.Inventory[cfg.ID] = cfg
	return nil
}

func (s *Service) DeleteBucketInventoryConfig(bucket, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	delete(state.bucket.Inventory, id)
	return nil
}

func (s *Service) GetBucketIntelligentTieringConfig(bucket, id string) (BucketIntelligentTieringConfiguration, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return BucketIntelligentTieringConfiguration{}, false, ErrBucketNotFound
	}
	cfg, ok := state.bucket.IntelligentTiering[id]
	return cfg, ok, nil
}

func (s *Service) ListBucketIntelligentTieringConfigs(bucket string) ([]BucketIntelligentTieringConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	out := make([]BucketIntelligentTieringConfiguration, 0, len(state.bucket.IntelligentTiering))
	for _, cfg := range state.bucket.IntelligentTiering {
		out = append(out, cfg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Service) SetBucketIntelligentTieringConfig(bucket string, cfg BucketIntelligentTieringConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	if state.bucket.IntelligentTiering == nil {
		state.bucket.IntelligentTiering = map[string]BucketIntelligentTieringConfiguration{}
	}
	state.bucket.IntelligentTiering[cfg.ID] = cfg
	return nil
}

func (s *Service) DeleteBucketIntelligentTieringConfig(bucket, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	delete(state.bucket.IntelligentTiering, id)
	return nil
}

func (s *Service) DeleteBucketReplication(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Replication = nil
	return nil
}

func (s *Service) GetBucketPolicy(bucket string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return "", ErrBucketNotFound
	}
	return state.bucket.Policy, nil
}

func (s *Service) SetBucketPolicy(bucket, policy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Policy = policy
	return nil
}

func (s *Service) DeleteBucketPolicy(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Policy = ""
	return nil
}

func (s *Service) GetBucketPublicAccessBlock(bucket string) (*PublicAccessBlockConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.PublicAccessBlock, nil
}

func (s *Service) SetBucketPublicAccessBlock(bucket string, cfg *PublicAccessBlockConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.PublicAccessBlock = cfg
	return nil
}

func (s *Service) DeleteBucketPublicAccessBlock(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.PublicAccessBlock = nil
	return nil
}

func (s *Service) GetBucketObjectLock(bucket string) (*ObjectLockConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	return state.bucket.ObjectLock, nil
}

func (s *Service) SetBucketObjectLock(bucket string, cfg *ObjectLockConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.ObjectLock = cfg
	return nil
}

func copyTags(tags map[string]string) map[string]string {
	if tags == nil {
		return nil
	}
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}

func (s *Service) GetBucketVersioning(bucket string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return "", ErrBucketNotFound
	}
	return state.versioning, nil
}

func (s *Service) SetBucketVersioning(bucket, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	if status != "Enabled" && status != "Suspended" {
		return errors.New("invalid versioning status")
	}
	state.versioning = status
	return nil
}

func (s *Service) SetBucketACL(bucket, acl string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	if acl == "" {
		acl = "private"
	}
	state.bucket.ACL = acl
	return nil
}

func (s *Service) DeleteObject(bucket, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	versions, ok := state.objects[key]
	if !ok || len(versions) == 0 {
		return ErrObjectNotFound
	}
	delete(state.objects, key)
	return nil
}

func (s *Service) DeleteObjectVersioned(bucket, key, versionID string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return "", false, ErrBucketNotFound
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return "", false, ErrObjectNotFound
	}
	if versionID == "" {
		// create delete marker if versioning enabled
		if state.versioning == "Enabled" {
			marker := createDeleteMarker(bucket, key, versions, "")
			state.objects[key] = append([]Object{marker}, versions...)
			return marker.VersionID, true, nil
		}
		// default behavior
		delete(state.objects, key)
		return "", false, nil
	}
	out := make([]Object, 0, len(versions))
	found := false
	for _, obj := range versions {
		if obj.VersionID == versionID {
			found = true
			continue
		}
		out = append(out, obj)
	}
	if !found {
		return "", false, ErrObjectNotFound
	}
	if len(out) > 0 {
		out[0].IsLatest = true
	}
	state.objects[key] = out
	return versionID, false, nil
}

func (s *Service) PutDeleteMarker(bucket, key, versionID string) (Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return Object{}, ErrBucketNotFound
	}
	versions := state.objects[key]
	marker := createDeleteMarker(bucket, key, versions, versionID)
	state.objects[key] = append([]Object{marker}, versions...)
	return marker, nil
}

func createDeleteMarker(bucket, key string, versions []Object, versionID string) Object {
	for i := range versions {
		versions[i].IsLatest = false
	}
	if versionID == "" {
		versionID = newVersionID()
	}
	return Object{
		Key:          key,
		Bucket:       bucket,
		UpdatedAt:    time.Now().UTC(),
		VersionID:    versionID,
		IsLatest:     true,
		DeleteMarker: true,
	}
}

func (s *Service) SetObjectACL(bucket, key, versionID, acl string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return ErrObjectNotFound
	}
	if acl == "" {
		acl = "private"
	}
	if versionID != "" {
		for i := range versions {
			if versions[i].VersionID == versionID {
				versions[i].ACL = acl
				state.objects[key] = versions
				return nil
			}
		}
		return ErrObjectNotFound
	}
	for i := range versions {
		if versions[i].IsLatest {
			versions[i].ACL = acl
			state.objects[key] = versions
			return nil
		}
	}
	return ErrObjectNotFound
}

func (s *Service) GetObjectTags(bucket, key, versionID string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return nil, ErrObjectNotFound
	}
	if versionID != "" {
		for _, obj := range versions {
			if obj.VersionID == versionID {
				if obj.DeleteMarker {
					return nil, ErrObjectNotFound
				}
				return copyTags(obj.Tags), nil
			}
		}
		return nil, ErrObjectNotFound
	}
	for _, obj := range versions {
		if obj.IsLatest && !obj.DeleteMarker {
			return copyTags(obj.Tags), nil
		}
	}
	return nil, ErrObjectNotFound
}

func (s *Service) SetObjectTags(bucket, key, versionID string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return ErrObjectNotFound
	}
	if versionID != "" {
		for i := range versions {
			if versions[i].VersionID == versionID {
				if versions[i].DeleteMarker {
					return ErrObjectNotFound
				}
				versions[i].Tags = copyTags(tags)
				state.objects[key] = versions
				return nil
			}
		}
		return ErrObjectNotFound
	}
	for i := range versions {
		if versions[i].IsLatest {
			if versions[i].DeleteMarker {
				return ErrObjectNotFound
			}
			versions[i].Tags = copyTags(tags)
			state.objects[key] = versions
			return nil
		}
	}
	return ErrObjectNotFound
}

func (s *Service) DeleteObjectTags(bucket, key, versionID string) error {
	return s.SetObjectTags(bucket, key, versionID, nil)
}

func (s *Service) GetObjectRetention(bucket, key, versionID string) (*ObjectRetention, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return nil, ErrObjectNotFound
	}
	if versionID != "" {
		for _, obj := range versions {
			if obj.VersionID == versionID {
				if obj.DeleteMarker {
					return nil, ErrObjectNotFound
				}
				return obj.Retention, nil
			}
		}
		return nil, ErrObjectNotFound
	}
	for _, obj := range versions {
		if obj.IsLatest && !obj.DeleteMarker {
			return obj.Retention, nil
		}
	}
	return nil, ErrObjectNotFound
}

func (s *Service) SetObjectRetention(bucket, key, versionID string, retention *ObjectRetention) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return ErrObjectNotFound
	}
	if versionID != "" {
		for i := range versions {
			if versions[i].VersionID == versionID {
				if versions[i].DeleteMarker {
					return ErrObjectNotFound
				}
				versions[i].Retention = retention
				state.objects[key] = versions
				return nil
			}
		}
		return ErrObjectNotFound
	}
	for i := range versions {
		if versions[i].IsLatest {
			if versions[i].DeleteMarker {
				return ErrObjectNotFound
			}
			versions[i].Retention = retention
			state.objects[key] = versions
			return nil
		}
	}
	return ErrObjectNotFound
}

func (s *Service) GetObjectLegalHold(bucket, key, versionID string) (*ObjectLegalHold, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return nil, ErrObjectNotFound
	}
	if versionID != "" {
		for _, obj := range versions {
			if obj.VersionID == versionID {
				if obj.DeleteMarker {
					return nil, ErrObjectNotFound
				}
				return obj.LegalHold, nil
			}
		}
		return nil, ErrObjectNotFound
	}
	for _, obj := range versions {
		if obj.IsLatest && !obj.DeleteMarker {
			return obj.LegalHold, nil
		}
	}
	return nil, ErrObjectNotFound
}

func (s *Service) SetObjectLegalHold(bucket, key, versionID string, hold *ObjectLegalHold) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	versions := state.objects[key]
	if len(versions) == 0 {
		return ErrObjectNotFound
	}
	if versionID != "" {
		for i := range versions {
			if versions[i].VersionID == versionID {
				if versions[i].DeleteMarker {
					return ErrObjectNotFound
				}
				versions[i].LegalHold = hold
				state.objects[key] = versions
				return nil
			}
		}
		return ErrObjectNotFound
	}
	for i := range versions {
		if versions[i].IsLatest {
			if versions[i].DeleteMarker {
				return ErrObjectNotFound
			}
			versions[i].LegalHold = hold
			state.objects[key] = versions
			return nil
		}
	}
	return ErrObjectNotFound
}

func buildDefaultRetention(cfg *DefaultRetention) *ObjectRetention {
	if cfg == nil {
		return nil
	}
	if cfg.Days <= 0 && cfg.Years <= 0 {
		return nil
	}
	retainUntil := time.Now().UTC()
	if cfg.Days > 0 {
		retainUntil = retainUntil.Add(time.Duration(cfg.Days) * 24 * time.Hour)
	}
	if cfg.Years > 0 {
		retainUntil = retainUntil.Add(time.Duration(cfg.Years*365) * 24 * time.Hour)
	}
	return &ObjectRetention{
		Mode:        cfg.Mode,
		RetainUntil: retainUntil,
	}
}

func (s *Service) DeleteObjects(bucket string, keys []string) ([]Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}

	deleted := make([]Object, 0, len(keys))
	for _, key := range keys {
		versions, ok := state.objects[key]
		if !ok || len(versions) == 0 {
			continue
		}
		if state.versioning == "Enabled" {
			for i := range versions {
				versions[i].IsLatest = false
			}
			marker := Object{
				Key:          key,
				Bucket:       bucket,
				UpdatedAt:    time.Now().UTC(),
				VersionID:    newVersionID(),
				IsLatest:     true,
				DeleteMarker: true,
			}
			state.objects[key] = append([]Object{marker}, versions...)
			deleted = append(deleted, marker)
			continue
		}
		delete(state.objects, key)
		deleted = append(deleted, Object{Key: key})
	}
	return deleted, nil
}

func (s *Service) ListObjectVersions(bucket, prefix string) ([]Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	out := make([]Object, 0, len(state.objects))
	for key, versions := range state.objects {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		for _, obj := range versions {
			obj.Body = nil
			out = append(out, obj)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key == out[j].Key {
			return out[i].UpdatedAt.After(out[j].UpdatedAt)
		}
		return out[i].Key < out[j].Key
	})
	return out, nil
}

func (s *Service) CopyObject(srcBucket, srcKey, dstBucket, dstKey string, metadata map[string]string, contentType string, acl string, storageClass string) (Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	srcState, ok := s.buckets[srcBucket]
	if !ok {
		return Object{}, ErrBucketNotFound
	}
	srcVersions := srcState.objects[srcKey]
	if len(srcVersions) == 0 {
		return Object{}, ErrObjectNotFound
	}
	var srcObj Object
	found := false
	for _, obj := range srcVersions {
		if obj.IsLatest && !obj.DeleteMarker {
			srcObj = obj
			found = true
			break
		}
	}
	if !found {
		return Object{}, ErrObjectNotFound
	}
	dstState, ok := s.buckets[dstBucket]
	if !ok {
		return Object{}, ErrBucketNotFound
	}

	if metadata == nil {
		metadata = srcObj.Metadata
	}
	if contentType == "" {
		contentType = srcObj.ContentType
	}
	if storageClass == "" {
		storageClass = srcObj.StorageClass
	}
	if acl == "" {
		acl = srcObj.ACL
	}
	if storageClass == "" {
		storageClass = "STANDARD"
	}

	versionID := ""
	if dstState.versioning == "Enabled" {
		versionID = newVersionID()
	}
	obj := Object{
		Key:                  dstKey,
		Bucket:               dstBucket,
		SizeBytes:            len(srcObj.Body),
		ContentType:          contentType,
		ETag:                 etagFor(srcObj.Body),
		StorageClass:         storageClass,
		SSEAlgorithm:         srcObj.SSEAlgorithm,
		SSECustomerAlgorithm: srcObj.SSECustomerAlgorithm,
		SSECustomerKeyMD5:    srcObj.SSECustomerKeyMD5,
		Metadata:             metadata,
		Tags:                 copyTags(srcObj.Tags),
		Retention:            srcObj.Retention,
		LegalHold:            srcObj.LegalHold,
		UpdatedAt:            time.Now().UTC(),
		Body:                 append([]byte(nil), srcObj.Body...),
		ACL:                  acl,
		VersionID:            versionID,
		IsLatest:             true,
	}
	versions := dstState.objects[dstKey]
	for i := range versions {
		versions[i].IsLatest = false
	}
	dstState.objects[dstKey] = append([]Object{obj}, versions...)
	return obj, nil
}

func (s *Service) InitiateMultipartUpload(bucket, key, contentType string, metadata map[string]string, acl string, storageClass string, sseAlg string, sseCustomerAlg string, sseCustomerMD5 string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.buckets[bucket]
	if !ok {
		return "", ErrBucketNotFound
	}

	s.uploadCounter++
	id := fmt.Sprintf("upload-%d-%d", time.Now().UTC().UnixNano(), s.uploadCounter)
	if storageClass == "" {
		storageClass = "STANDARD"
	}
	s.uploads[id] = &multipartUpload{
		ID:                   id,
		Bucket:               bucket,
		Key:                  key,
		ContentType:          contentType,
		Metadata:             metadata,
		ACL:                  acl,
		StorageClass:         storageClass,
		SSEAlgorithm:         sseAlg,
		SSECustomerAlgorithm: sseCustomerAlg,
		SSECustomerKeyMD5:    sseCustomerMD5,
		InitiatedAt:          time.Now().UTC(),
		Parts:                make(map[int]Object),
	}
	if _, ok := state.objects[key]; ok {
		// Keep existing object until completion.
	}
	return id, nil
}

func (s *Service) PutMultipartPart(uploadID string, partNumber int, body []byte) (Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	upload, ok := s.uploads[uploadID]
	if !ok {
		return Object{}, ErrObjectNotFound
	}

	obj := Object{
		Key:         upload.Key,
		Bucket:      upload.Bucket,
		PartNumber:  partNumber,
		SizeBytes:   len(body),
		ContentType: upload.ContentType,
		ETag:        etagFor(body),
		Metadata:    upload.Metadata,
		Tags:        nil,
		Retention:   nil,
		LegalHold:   nil,
		UpdatedAt:   time.Now().UTC(),
		Body:        append([]byte(nil), body...),
	}
	upload.Parts[partNumber] = obj
	return obj, nil
}

func (s *Service) CompleteMultipartUpload(uploadID string, orderedParts []int) (Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	upload, ok := s.uploads[uploadID]
	if !ok {
		return Object{}, ErrObjectNotFound
	}
	state, ok := s.buckets[upload.Bucket]
	if !ok {
		return Object{}, ErrBucketNotFound
	}

	var combined []byte
	var partEtags []string
	for _, partNumber := range orderedParts {
		part, ok := upload.Parts[partNumber]
		if !ok {
			return Object{}, ErrObjectNotFound
		}
		combined = append(combined, part.Body...)
		partEtags = append(partEtags, part.ETag)
	}

	versionID := ""
	if state.versioning == "Enabled" {
		versionID = newVersionID()
	}
	acl := upload.ACL
	if acl == "" {
		acl = "private"
	}
	obj := Object{
		Key:                  upload.Key,
		Bucket:               upload.Bucket,
		SizeBytes:            len(combined),
		ContentType:          upload.ContentType,
		ETag:                 multipartETag(partEtags),
		StorageClass:         upload.StorageClass,
		SSEAlgorithm:         upload.SSEAlgorithm,
		SSECustomerAlgorithm: upload.SSECustomerAlgorithm,
		SSECustomerKeyMD5:    upload.SSECustomerKeyMD5,
		Metadata:             upload.Metadata,
		Tags:                 nil,
		Retention:            nil,
		LegalHold:            nil,
		UpdatedAt:            time.Now().UTC(),
		Body:                 combined,
		ACL:                  acl,
		VersionID:            versionID,
		IsLatest:             true,
	}
	versions := state.objects[upload.Key]
	for i := range versions {
		versions[i].IsLatest = false
	}
	state.objects[upload.Key] = append([]Object{obj}, versions...)
	delete(s.uploads, uploadID)
	return obj, nil
}

func (s *Service) AbortMultipartUpload(uploadID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.uploads, uploadID)
}

func (s *Service) GetMultipartUpload(uploadID string) (*multipartUpload, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	upload, ok := s.uploads[uploadID]
	return upload, ok
}

func (s *Service) ListMultipartUploads(bucket, prefix string) []MultipartUploadInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MultipartUploadInfo, 0)
	for _, upload := range s.uploads {
		if upload.Bucket != bucket {
			continue
		}
		if prefix != "" && !strings.HasPrefix(upload.Key, prefix) {
			continue
		}
		out = append(out, MultipartUploadInfo{
			ID:           upload.ID,
			Bucket:       upload.Bucket,
			Key:          upload.Key,
			StorageClass: upload.StorageClass,
			InitiatedAt:  upload.InitiatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key == out[j].Key {
			return out[i].InitiatedAt.Before(out[j].InitiatedAt)
		}
		return out[i].Key < out[j].Key
	})
	return out
}

func (s *Service) ListMultipartParts(uploadID string) ([]Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	upload, ok := s.uploads[uploadID]
	if !ok {
		return nil, ErrObjectNotFound
	}
	parts := make([]Object, 0, len(upload.Parts))
	for _, part := range upload.Parts {
		part.Body = nil
		parts = append(parts, part)
	}
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber
	})
	return parts, nil

}

func newVersionID() string {
	return fmt.Sprintf("v-%d", time.Now().UTC().UnixNano())
}

func (s *Service) SetBucketMetadataConfiguration(bucket string, cfg BucketMetadataConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.Metadata = &cfg
	return nil
}

func (s *Service) GetBucketMetadataConfiguration(bucket string) (*BucketMetadataConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	if state.bucket.Metadata == nil {
		return nil, ErrMetadataConfigurationNotFound
	}
	cfg := *state.bucket.Metadata
	return &cfg, nil
}

func (s *Service) DeleteBucketMetadataConfiguration(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	if state.bucket.Metadata == nil {
		return ErrMetadataConfigurationNotFound
	}
	state.bucket.Metadata = nil
	return nil
}

func (s *Service) SetBucketMetadataInventoryTableConfiguration(bucket string, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	bucketState, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	if bucketState.bucket.Metadata == nil {
		return ErrMetadataConfigurationNotFound
	}
	bucketState.bucket.Metadata.InventoryState = state
	return nil
}

func (s *Service) SetBucketMetadataJournalTableConfiguration(bucket string, days int, expiration string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	bucketState, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	if bucketState.bucket.Metadata == nil {
		return ErrMetadataConfigurationNotFound
	}
	bucketState.bucket.Metadata.JournalDays = days
	bucketState.bucket.Metadata.JournalExpireAt = expiration
	return nil
}

func (s *Service) SetBucketMetadataTableConfiguration(bucket string, cfg BucketMetadataTableConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	state.bucket.MetadataTable = &cfg
	return nil
}

func (s *Service) GetBucketMetadataTableConfiguration(bucket string) (*BucketMetadataTableConfiguration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return nil, ErrBucketNotFound
	}
	if state.bucket.MetadataTable == nil {
		return nil, ErrMetadataTableConfigurationNotFound
	}
	cfg := *state.bucket.MetadataTable
	return &cfg, nil
}

func (s *Service) DeleteBucketMetadataTableConfiguration(bucket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.buckets[bucket]
	if !ok {
		return ErrBucketNotFound
	}
	if state.bucket.MetadataTable == nil {
		return ErrMetadataTableConfigurationNotFound
	}
	state.bucket.MetadataTable = nil
	return nil
}
