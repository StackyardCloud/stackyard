package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CloudFront KeyValueStore state is shared by the CloudFront control plane
// (Create/Describe/List/Update/DeleteKeyValueStore) and the
// cloudfront-keyvaluestore data plane (keys).

const (
	cfKVSAccountID       = "123456789012"
	cfKVSMaxKeyBytes     = 512
	cfKVSMaxValueBytes   = 1024
	cfKVSMaxStoreBytes   = 5 << 20
	cfKVSMaxUpdateBytes  = 3 << 20
	cfKVSDefaultListKeys = 10
	cfKVSMaxListKeys     = 50
)

var (
	errCFKVSNotFound       = errors.New("not found")
	errCFKVSExists         = errors.New("already exists")
	errCFKVSInvalid        = errors.New("invalid argument")
	errCFKVSETagMismatch   = errors.New("etag mismatch")
	errCFKVSMissingIfMatch = errors.New("missing If-Match")
	errCFKVSQuota          = errors.New("quota exceeded")

	cfKVSNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
)

type cfKVStore struct {
	Name         string
	ID           string
	ARN          string
	Comment      string
	Status       string
	Created      time.Time
	LastModified time.Time
	StoreETag    string // control-plane version (Update/DeleteKeyValueStore)
	DataETag     string // data-plane version (key mutations)
	items        map[string]string
	size         int64
}

type cfKVSStats struct {
	ItemCount int
	TotalSize int64
	ETag      string
}

type cfKVSPut struct{ Key, Value string }

type cloudFrontKVSRegistry struct {
	mu      sync.Mutex
	byName  map[string]*cfKVStore
	byARN   map[string]*cfKVStore
	version uint64
}

func newCloudFrontKVSRegistry() *cloudFrontKVSRegistry {
	return &cloudFrontKVSRegistry{byName: map[string]*cfKVStore{}, byARN: map[string]*cfKVStore{}}
}

// nextETagLocked returns a value never handed out before, so a recreated
// store or a reverted value can never match a stale If-Match.
func (g *cloudFrontKVSRegistry) nextETagLocked() string {
	g.version++
	return "KV" + strconv.FormatUint(g.version+0x1000000000, 36)
}

func (g *cloudFrontKVSRegistry) Create(name, comment string) (cfKVStore, error) {
	if !cfKVSNamePattern.MatchString(name) {
		return cfKVStore{}, fmt.Errorf("%w: Name must match [a-zA-Z0-9-_]{1,64}", errCFKVSInvalid)
	}
	if len(comment) > 128 {
		return cfKVStore{}, fmt.Errorf("%w: Comment must be at most 128 characters", errCFKVSInvalid)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.byName[name]; ok {
		return cfKVStore{}, fmt.Errorf("%w: key value store %s already exists", errCFKVSExists, name)
	}
	now := time.Now().UTC()
	id := uuid.NewString()
	kvs := &cfKVStore{
		Name: name, ID: id, Comment: comment, Status: "READY",
		ARN:     "arn:aws:cloudfront::" + cfKVSAccountID + ":key-value-store/" + id,
		Created: now, LastModified: now,
		StoreETag: g.nextETagLocked(), DataETag: g.nextETagLocked(),
		items: map[string]string{},
	}
	g.byName[name] = kvs
	g.byARN[kvs.ARN] = kvs
	return kvs.snapshot(), nil
}

func (g *cloudFrontKVSRegistry) Describe(name string) (cfKVStore, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	kvs, ok := g.byName[name]
	if !ok {
		return cfKVStore{}, fmt.Errorf("%w: key value store %s", errCFKVSNotFound, name)
	}
	return kvs.snapshot(), nil
}

// List returns stores sorted by name, starting after marker.
func (g *cloudFrontKVSRegistry) List(marker string, maxItems int, status string) ([]cfKVStore, string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	names := make([]string, 0, len(g.byName))
	for name, kvs := range g.byName {
		if name > marker && (status == "" || kvs.Status == status) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	next := ""
	if len(names) > maxItems {
		names = names[:maxItems]
		next = names[maxItems-1]
	}
	out := make([]cfKVStore, 0, len(names))
	for _, name := range names {
		out = append(out, g.byName[name].snapshot())
	}
	return out, next
}

func (g *cloudFrontKVSRegistry) Update(name, comment, ifMatch string) (cfKVStore, error) {
	if len(comment) > 128 {
		return cfKVStore{}, fmt.Errorf("%w: Comment must be at most 128 characters", errCFKVSInvalid)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	kvs, err := g.storeForControlLocked(name, ifMatch)
	if err != nil {
		return cfKVStore{}, err
	}
	kvs.Comment = comment
	kvs.LastModified = time.Now().UTC()
	kvs.StoreETag = g.nextETagLocked()
	return kvs.snapshot(), nil
}

func (g *cloudFrontKVSRegistry) Delete(name, ifMatch string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	kvs, err := g.storeForControlLocked(name, ifMatch)
	if err != nil {
		return err
	}
	delete(g.byName, name)
	delete(g.byARN, kvs.ARN)
	return nil
}

func (g *cloudFrontKVSRegistry) storeForControlLocked(name, ifMatch string) (*cfKVStore, error) {
	kvs, ok := g.byName[name]
	if !ok {
		return nil, fmt.Errorf("%w: key value store %s", errCFKVSNotFound, name)
	}
	if ifMatch == "" {
		return nil, fmt.Errorf("%w: If-Match is required", errCFKVSMissingIfMatch)
	}
	if ifMatch != kvs.StoreETag {
		return nil, fmt.Errorf("%w: If-Match does not match the current ETag", errCFKVSETagMismatch)
	}
	return kvs, nil
}

// --- data plane ---

func (g *cloudFrontKVSRegistry) DescribeByARN(arn string) (cfKVStore, cfKVSStats, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	kvs, err := g.byARNLocked(arn)
	if err != nil {
		return cfKVStore{}, cfKVSStats{}, err
	}
	return kvs.snapshot(), kvs.stats(), nil
}

func (g *cloudFrontKVSRegistry) GetKey(arn, key string) (string, cfKVSStats, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	kvs, err := g.byARNLocked(arn)
	if err != nil {
		return "", cfKVSStats{}, err
	}
	value, ok := kvs.items[key]
	if !ok {
		return "", cfKVSStats{}, fmt.Errorf("%w: key %s", errCFKVSNotFound, key)
	}
	return value, kvs.stats(), nil
}

// ListKeys pages keys in byte order; the token encodes the last key returned.
func (g *cloudFrontKVSRegistry) ListKeys(arn, token string, maxResults int) ([]cfKVSPut, string, error) {
	after := ""
	if token != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil || len(decoded) == 0 {
			return nil, "", fmt.Errorf("%w: invalid NextToken", errCFKVSInvalid)
		}
		after = string(decoded)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	kvs, err := g.byARNLocked(arn)
	if err != nil {
		return nil, "", err
	}
	keys := make([]string, 0, len(kvs.items))
	for key := range kvs.items {
		if key > after {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	next := ""
	if len(keys) > maxResults {
		keys = keys[:maxResults]
		next = base64.RawURLEncoding.EncodeToString([]byte(keys[maxResults-1]))
	}
	items := make([]cfKVSPut, 0, len(keys))
	for _, key := range keys {
		items = append(items, cfKVSPut{Key: key, Value: kvs.items[key]})
	}
	return items, next, nil
}

// UpdateKeys validates every put and delete before applying any, so a
// rejected batch leaves the store and its ETag unchanged. PutKey and
// DeleteKey are single-item batches.
func (g *cloudFrontKVSRegistry) UpdateKeys(arn, ifMatch string, puts []cfKVSPut, deletes []string) (cfKVSStats, error) {
	if ifMatch == "" {
		return cfKVSStats{}, fmt.Errorf("%w: If-Match is required", errCFKVSMissingIfMatch)
	}
	if len(puts) == 0 && len(deletes) == 0 {
		return cfKVSStats{}, fmt.Errorf("%w: at least one put or delete is required", errCFKVSInvalid)
	}
	seen := map[string]struct{}{}
	var requestBytes int64
	for _, put := range puts {
		if err := validateCFKVSKey(put.Key); err != nil {
			return cfKVSStats{}, err
		}
		if len(put.Value) > cfKVSMaxValueBytes {
			return cfKVSStats{}, fmt.Errorf("%w: value for key %s exceeds %d bytes", errCFKVSInvalid, put.Key, cfKVSMaxValueBytes)
		}
		if _, dup := seen[put.Key]; dup {
			return cfKVSStats{}, fmt.Errorf("%w: key %s appears more than once", errCFKVSInvalid, put.Key)
		}
		seen[put.Key] = struct{}{}
		requestBytes += int64(len(put.Key) + len(put.Value))
	}
	for _, key := range deletes {
		if err := validateCFKVSKey(key); err != nil {
			return cfKVSStats{}, err
		}
		if _, dup := seen[key]; dup {
			return cfKVSStats{}, fmt.Errorf("%w: key %s appears more than once", errCFKVSInvalid, key)
		}
		seen[key] = struct{}{}
		requestBytes += int64(len(key))
	}
	if requestBytes > cfKVSMaxUpdateBytes {
		return cfKVSStats{}, fmt.Errorf("%w: update payload exceeds %d bytes", errCFKVSQuota, cfKVSMaxUpdateBytes)
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	kvs, err := g.byARNLocked(arn)
	if err != nil {
		return cfKVSStats{}, err
	}
	if ifMatch != kvs.DataETag {
		return cfKVSStats{}, fmt.Errorf("%w: If-Match does not match the current ETag", errCFKVSETagMismatch)
	}
	size := kvs.size
	for _, key := range deletes {
		value, ok := kvs.items[key]
		if !ok {
			return cfKVSStats{}, fmt.Errorf("%w: key %s", errCFKVSNotFound, key)
		}
		size -= int64(len(key) + len(value))
	}
	for _, put := range puts {
		if old, ok := kvs.items[put.Key]; ok {
			size -= int64(len(put.Key) + len(old))
		}
		size += int64(len(put.Key) + len(put.Value))
	}
	if size > cfKVSMaxStoreBytes {
		return cfKVSStats{}, fmt.Errorf("%w: key value store would exceed %d bytes", errCFKVSQuota, cfKVSMaxStoreBytes)
	}

	for _, key := range deletes {
		delete(kvs.items, key)
	}
	for _, put := range puts {
		kvs.items[put.Key] = put.Value
	}
	kvs.size = size
	kvs.LastModified = time.Now().UTC()
	kvs.DataETag = g.nextETagLocked()
	return kvs.stats(), nil
}

func (g *cloudFrontKVSRegistry) byARNLocked(arn string) (*cfKVStore, error) {
	kvs, ok := g.byARN[arn]
	if !ok {
		return nil, fmt.Errorf("%w: key value store %s", errCFKVSNotFound, arn)
	}
	return kvs, nil
}

func validateCFKVSKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: Key must not be empty", errCFKVSInvalid)
	}
	if len(key) > cfKVSMaxKeyBytes {
		return fmt.Errorf("%w: Key exceeds %d bytes", errCFKVSInvalid, cfKVSMaxKeyBytes)
	}
	return nil
}

// snapshot copies the metadata without the item map.
func (k *cfKVStore) snapshot() cfKVStore {
	out := *k
	out.items = nil
	return out
}

func (k *cfKVStore) stats() cfKVSStats {
	return cfKVSStats{ItemCount: len(k.items), TotalSize: k.size, ETag: k.DataETag}
}
