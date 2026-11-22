package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	glcAccountID = "123456789012"
	glcRegion    = "us-east-1"

	glcMaxBody          = 64 << 20 // one request body (UploadArchive / UploadMultipartPart)
	glcMaxStoreBytes    = 1 << 30  // archive + in-flight part bytes across vaults
	glcMaxArchives      = 100000   // archives across vaults
	glcMaxVaults        = 1000     // AWS's per-region vault quota
	glcMaxUploads       = 1000     // in-progress multipart uploads per vault
	glcMaxJobs          = 1000     // jobs kept per vault (oldest completed go first)
	glcMaxTags          = 50
	glcMaxPolicyBytes   = 20 << 10
	glcMaxDescription   = 1024
	glcMaxCapacityUnits = 10 // ponytail: AWS allows 2; higher so repeated coverage runs on one server don't exhaust it
	glcTreeChunk        = 1 << 20
	glcMinPartSize      = 1 << 20
	glcMaxPartSize      = 4 << 30
	glcLockWindow       = 24 * time.Hour
	glcTimeLayout       = "2006-01-02T15:04:05.000Z"
)

var (
	glcNow          = func() time.Time { return time.Now().UTC() }
	glcVaultPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,255}$`)
)

type glcErr struct {
	status    int
	code, msg string
}

func glcInvalid(format string, args ...any) *glcErr {
	return &glcErr{http.StatusBadRequest, "InvalidParameterValueException", fmt.Sprintf(format, args...)}
}

func glcMissing(format string, args ...any) *glcErr {
	return &glcErr{http.StatusBadRequest, "MissingParameterValueException", fmt.Sprintf(format, args...)}
}

func glcNotFound(format string, args ...any) *glcErr {
	return &glcErr{http.StatusNotFound, "ResourceNotFoundException", fmt.Sprintf(format, args...)}
}

func glcLimit(format string, args ...any) *glcErr {
	return &glcErr{http.StatusBadRequest, "LimitExceededException", fmt.Sprintf(format, args...)}
}

type glcStore struct {
	mu        sync.Mutex
	vaults    map[string]*glcVault
	bytes     int64 // archive and in-flight part bytes
	archives  int
	retrieval []glcRule
	capacity  []glcCapacity
}

type glcVault struct {
	name      string
	created   time.Time
	archives  map[string]*glcArchive
	uploads   map[string]*glcUpload
	jobs      map[string]*glcJob
	jobOrder  []string
	tags      map[string]string
	policy    string // access policy JSON, "" when unset
	notify    *glcNotify
	lock      *glcLock
	inventory *glcInventory // last completed inventory, nil before the first
}

type glcArchive struct {
	id, desc, hash string
	created        time.Time
	data           []byte // never mutated once stored, so jobs may share it
}

type glcUpload struct {
	id, desc string
	partSize int64
	created  time.Time
	parts    map[int64]*glcPart // by first byte
	bytes    int64
}

type glcPart struct {
	start int64
	data  []byte
	hash  string
}

type glcNotify struct {
	Topic  string   `json:"SNSTopic"`
	Events []string `json:"Events"`
}

type glcLock struct {
	id, policy, state  string
	created, expiresAt time.Time
}

type glcInventory struct {
	date        time.Time
	count, size int64
}

type glcRule struct {
	Strategy     string `json:"Strategy"`
	BytesPerHour *int64 `json:"BytesPerHour,omitempty"`
}

type glcCapacity struct {
	ID         string `json:"CapacityId"`
	StartDate  string `json:"StartDate"`
	Expiration string `json:"ExpirationDate"`
}

func newGlacierStore() *glcStore {
	// ponytail: AWS's default retrieval policy is Free Tier Only; free-tier
	// usage isn't metered here, so only BytesPerHour is ever enforced.
	return &glcStore{vaults: map[string]*glcVault{}, retrieval: []glcRule{{Strategy: "FreeTier"}}}
}

func glcVaultARN(name string) string {
	return "arn:aws:glacier:" + glcRegion + ":" + glcAccountID + ":vaults/" + name
}

func glcVaultPath(name string) string { return "/" + glcAccountID + "/vaults/" + name }

func glcTime(t time.Time) string { return t.UTC().Format(glcTimeLayout) }

// glcRandomID returns n URL-safe characters, the shape of Glacier archive,
// upload and job ids.
func glcRandomID(n int) string {
	raw := make([]byte, n)
	_, _ = rand.Read(raw)
	return base64.RawURLEncoding.EncodeToString(raw)[:n]
}

// glcTreeHash is the Glacier SHA-256 tree hash: hash each 1 MiB chunk, then
// hash adjacent pairs until one digest is left.
func glcTreeHash(data []byte) string {
	if len(data) == 0 {
		sum := sha256.Sum256(nil)
		return hex.EncodeToString(sum[:])
	}
	var level [][32]byte
	for off := 0; off < len(data); off += glcTreeChunk {
		level = append(level, sha256.Sum256(data[off:min(off+glcTreeChunk, len(data))]))
	}
	for len(level) > 1 {
		next := make([][32]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			if i+1 == len(level) {
				next = append(next, level[i])
				continue
			}
			next = append(next, sha256.Sum256(append(level[i][:], level[i+1][:]...)))
		}
		level = next
	}
	return hex.EncodeToString(level[0][:])
}

func glcCheckVaultName(name string) *glcErr {
	if !glcVaultPattern.MatchString(name) {
		return glcInvalid("Invalid vault name: %q. Names are 1-255 characters of a-z, A-Z, 0-9, '_', '-' and '.'.", name)
	}
	return nil
}

// glcCheckDescription enforces AWS's printable-ASCII, 1024-character rule for
// archive and job descriptions (archive descriptions travel in a header).
func glcCheckDescription(what, desc string) *glcErr {
	if len(desc) > glcMaxDescription {
		return glcInvalid("%s must be at most %d characters", what, glcMaxDescription)
	}
	for i := 0; i < len(desc); i++ {
		if desc[i] < 32 || desc[i] > 126 {
			return glcInvalid("%s must contain only printable ASCII characters", what)
		}
	}
	return nil
}

func glcCheckPolicy(policy string) *glcErr {
	if policy == "" {
		return glcMissing("Policy is required")
	}
	if len(policy) > glcMaxPolicyBytes {
		return glcInvalid("Policy must be at most %d bytes", glcMaxPolicyBytes)
	}
	var doc map[string]any
	if json.Unmarshal([]byte(policy), &doc) != nil {
		return glcInvalid("Policy is not valid JSON")
	}
	if _, ok := doc["Statement"]; !ok {
		return glcInvalid("Policy must contain a Statement")
	}
	return nil
}

func (st *glcStore) vaultLocked(name string) (*glcVault, *glcErr) {
	if err := glcCheckVaultName(name); err != nil {
		return nil, err
	}
	v, ok := st.vaults[name]
	if !ok {
		return nil, glcNotFound("Vault not found for ARN: %s", glcVaultARN(name))
	}
	return v, nil
}

// --- vaults ---

func (st *glcStore) createVault(name string) *glcErr {
	if err := glcCheckVaultName(name); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.vaults[name]; ok {
		return nil // CreateVault is idempotent
	}
	if len(st.vaults) >= glcMaxVaults {
		return glcLimit("An account can have at most %d vaults per region", glcMaxVaults)
	}
	st.vaults[name] = &glcVault{name: name, created: glcNow(), archives: map[string]*glcArchive{}, uploads: map[string]*glcUpload{}, jobs: map[string]*glcJob{}, tags: map[string]string{}}
	return nil
}

func (v *glcVault) describe() map[string]any {
	out := map[string]any{"VaultARN": glcVaultARN(v.name), "VaultName": v.name, "CreationDate": glcTime(v.created), "NumberOfArchives": int64(0), "SizeInBytes": int64(0)}
	if v.inventory != nil {
		out["LastInventoryDate"] = glcTime(v.inventory.date)
		out["NumberOfArchives"], out["SizeInBytes"] = v.inventory.count, v.inventory.size
	}
	return out
}

func (st *glcStore) describeVault(name string) (map[string]any, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return nil, err
	}
	return v.describe(), nil
}

func (st *glcStore) listVaults(marker string, limit int) (map[string]any, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	names := make([]string, 0, len(st.vaults))
	for name := range st.vaults {
		names = append(names, name)
	}
	sort.Strings(names)
	page, next, err := glcPage(names, func(n string) string { return glcVaultARN(n) }, marker, limit)
	if err != nil {
		return nil, err
	}
	list := make([]map[string]any, 0, len(page))
	for _, n := range page {
		list = append(list, st.vaults[n].describe())
	}
	out := map[string]any{"VaultList": list}
	if next != "" {
		out["Marker"] = next
	}
	return out, nil
}

// glcPage returns the items after marker (matched by key) and the marker for
// the following page.
func glcPage[T any](items []T, key func(T) string, marker string, limit int) ([]T, string, *glcErr) {
	start := 0
	if marker != "" {
		start = -1
		for i, it := range items {
			if key(it) == marker {
				start = i
				break
			}
		}
		if start < 0 {
			return nil, "", glcInvalid("Invalid marker: %s", marker)
		}
	}
	end := min(start+limit, len(items))
	next := ""
	if end < len(items) {
		next = key(items[end])
	}
	return items[start:end], next, nil
}

func (st *glcStore) deleteVault(name string) *glcErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return err
	}
	if len(v.archives) > 0 {
		return glcInvalid("Vault not empty or recently written to: %s", glcVaultARN(name))
	}
	for _, u := range v.uploads {
		st.bytes -= u.bytes
	}
	for _, j := range v.jobs {
		st.bytes -= j.held
	}
	delete(st.vaults, name)
	return nil
}

// --- archives ---

func (st *glcStore) reserveLocked(n int64) *glcErr {
	if st.bytes+n > glcMaxStoreBytes {
		return glcLimit("Stackyard's Glacier storage budget (%d bytes) is exhausted", glcMaxStoreBytes)
	}
	return nil
}

func (st *glcStore) putArchiveLocked(v *glcVault, desc string, data []byte, hash string) (*glcArchive, *glcErr) {
	if st.archives >= glcMaxArchives {
		return nil, glcLimit("Stackyard holds at most %d archives", glcMaxArchives)
	}
	if err := st.reserveLocked(int64(len(data))); err != nil {
		return nil, err
	}
	a := &glcArchive{id: glcRandomID(138), desc: desc, hash: hash, created: glcNow(), data: data}
	v.archives[a.id] = a
	st.bytes += int64(len(data))
	st.archives++
	return a, nil
}

func (st *glcStore) uploadArchive(name, desc, checksum string, data []byte) (*glcArchive, *glcErr) {
	if err := glcCheckDescription("Archive description", desc); err != nil {
		return nil, err
	}
	if checksum == "" {
		return nil, glcMissing("Missing required header: x-amz-sha256-tree-hash")
	}
	if len(data) == 0 {
		return nil, glcInvalid("Invalid Content-Length: 0. An archive must be at least 1 byte")
	}
	hash := glcTreeHash(data)
	if !strings.EqualFold(hash, checksum) {
		return nil, glcInvalid("Checksum mismatch: expected %s (calculated), got %s", hash, checksum)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return nil, err
	}
	return st.putArchiveLocked(v, desc, data, hash)
}

func (st *glcStore) deleteArchive(name, id string) *glcErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return err
	}
	a, ok := v.archives[id]
	if !ok {
		return glcNotFound("Archive not found: %s", id)
	}
	// ponytail: a Locked vault lock policy isn't evaluated, so deletes it
	// would deny still succeed.
	delete(v.archives, id)
	st.bytes -= int64(len(a.data))
	st.archives--
	// Retrieval jobs share the archive's bytes; drop them so the store budget
	// stays the bound on memory.
	for _, j := range v.jobs {
		if j.archiveID == id && j.action == "ArchiveRetrieval" {
			j.data, j.gone = nil, true
		}
	}
	return nil
}

// --- multipart uploads ---

func (st *glcStore) initiateUpload(name, desc string, partSize int64) (*glcUpload, *glcErr) {
	if err := glcCheckDescription("Archive description", desc); err != nil {
		return nil, err
	}
	if partSize < glcMinPartSize || partSize > glcMaxPartSize || partSize&(partSize-1) != 0 {
		return nil, glcInvalid("Invalid part size: %d. Part size must be a power of two from 1 MiB to 4 GiB", partSize)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return nil, err
	}
	if len(v.uploads) >= glcMaxUploads {
		return nil, glcLimit("A vault can have at most %d multipart uploads in progress", glcMaxUploads)
	}
	u := &glcUpload{id: glcRandomID(92), desc: desc, partSize: partSize, created: glcNow(), parts: map[int64]*glcPart{}}
	v.uploads[u.id] = u
	return u, nil
}

func (st *glcStore) uploadLocked(name, id string) (*glcVault, *glcUpload, *glcErr) {
	v, err := st.vaultLocked(name)
	if err != nil {
		return nil, nil, err
	}
	u, ok := v.uploads[id]
	if !ok {
		return nil, nil, glcNotFound("Multipart upload not found: %s", id)
	}
	return v, u, nil
}

func (st *glcStore) uploadPart(name, id, checksum string, start, end int64, data []byte) *glcErr {
	if checksum == "" {
		return glcMissing("Missing required header: x-amz-sha256-tree-hash")
	}
	if int64(len(data)) != end-start+1 {
		return glcInvalid("Content-Range length %d does not match the body length %d", end-start+1, len(data))
	}
	hash := glcTreeHash(data)
	if !strings.EqualFold(hash, checksum) {
		return glcInvalid("Checksum mismatch: expected %s (calculated), got %s", hash, checksum)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	_, u, err := st.uploadLocked(name, id)
	if err != nil {
		return err
	}
	if start%u.partSize != 0 || int64(len(data)) > u.partSize {
		return glcInvalid("Invalid Content-Range: bytes %d-%d/* for part size %d. Parts start on part-size boundaries and are at most one part long", start, end, u.partSize)
	}
	size := int64(len(data))
	old := u.parts[start]
	if old != nil {
		size -= int64(len(old.data))
	}
	if err := st.reserveLocked(size); err != nil {
		return err
	}
	u.parts[start] = &glcPart{start: start, data: data, hash: hash}
	u.bytes += size
	st.bytes += size
	return nil
}

func (st *glcStore) completeUpload(name, id, checksum string, archiveSize int64) (*glcArchive, *glcErr) {
	if checksum == "" {
		return nil, glcMissing("Missing required header: x-amz-sha256-tree-hash")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	v, u, err := st.uploadLocked(name, id)
	if err != nil {
		return nil, err
	}
	starts := make([]int64, 0, len(u.parts))
	for s := range u.parts {
		starts = append(starts, s)
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })
	// Parts must cover 0..archiveSize-1 with every part but the last full size.
	var total int64
	for i, s := range starts {
		p := u.parts[s]
		if s != total || (i < len(starts)-1 && int64(len(p.data)) != u.partSize) {
			return nil, glcInvalid("Parts do not form a contiguous archive starting at byte 0")
		}
		total += int64(len(p.data))
	}
	if total == 0 || total != archiveSize {
		return nil, glcInvalid("Archive size %d does not match the %d bytes uploaded", archiveSize, total)
	}
	data := make([]byte, 0, total)
	for _, s := range starts {
		data = append(data, u.parts[s].data...)
	}
	hash := glcTreeHash(data)
	if !strings.EqualFold(hash, checksum) {
		return nil, glcInvalid("Checksum mismatch: expected %s (calculated), got %s", hash, checksum)
	}
	// The parts' bytes become the archive's.
	st.bytes -= u.bytes
	delete(v.uploads, id)
	a, err := st.putArchiveLocked(v, u.desc, data, hash)
	if err != nil {
		v.uploads[id], st.bytes = u, st.bytes+u.bytes
		return nil, err
	}
	return a, nil
}

func (st *glcStore) abortUpload(name, id string) *glcErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, u, err := st.uploadLocked(name, id)
	if err != nil {
		return err
	}
	st.bytes -= u.bytes
	delete(v.uploads, id)
	return nil
}

func (u *glcUpload) describe(vault string) map[string]any {
	out := map[string]any{"MultipartUploadId": u.id, "VaultARN": glcVaultARN(vault), "PartSizeInBytes": u.partSize, "CreationDate": glcTime(u.created)}
	if u.desc != "" {
		out["ArchiveDescription"] = u.desc
	}
	return out
}

func (st *glcStore) listParts(name, id, marker string, limit int) (map[string]any, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, u, err := st.uploadLocked(name, id)
	if err != nil {
		return nil, err
	}
	starts := make([]int64, 0, len(u.parts))
	for s := range u.parts {
		starts = append(starts, s)
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })
	page, next, perr := glcPage(starts, func(s int64) string { return fmt.Sprint(s) }, marker, limit)
	if perr != nil {
		return nil, perr
	}
	parts := make([]map[string]any, 0, len(page))
	for _, s := range page {
		p := u.parts[s]
		parts = append(parts, map[string]any{"RangeInBytes": fmt.Sprintf("%d-%d", p.start, p.start+int64(len(p.data))-1), "SHA256TreeHash": p.hash})
	}
	out := u.describe(name)
	out["Parts"] = parts
	if next != "" {
		out["Marker"] = next
	}
	return out, nil
}

func (st *glcStore) listUploads(name, marker string, limit int) (map[string]any, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return nil, err
	}
	ups := make([]*glcUpload, 0, len(v.uploads))
	for _, u := range v.uploads {
		ups = append(ups, u)
	}
	sort.Slice(ups, func(i, j int) bool {
		return ups[i].created.Before(ups[j].created) || (ups[i].created.Equal(ups[j].created) && ups[i].id < ups[j].id)
	})
	page, next, perr := glcPage(ups, func(u *glcUpload) string { return u.id }, marker, limit)
	if perr != nil {
		return nil, perr
	}
	list := make([]map[string]any, 0, len(page))
	for _, u := range page {
		list = append(list, u.describe(name))
	}
	out := map[string]any{"UploadsList": list}
	if next != "" {
		out["Marker"] = next
	}
	return out, nil
}

// --- tags, access policy, notifications ---

func (st *glcStore) addTags(name string, tags map[string]string) *glcErr {
	for k, val := range tags {
		if k == "" || len(k) > 128 || len(val) > 256 {
			return glcInvalid("Tag keys must be 1-128 characters and values at most 256")
		}
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return glcInvalid("Tag keys can't start with aws:")
		}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return err
	}
	n := len(v.tags)
	for k := range tags {
		if _, ok := v.tags[k]; !ok {
			n++
		}
	}
	if n > glcMaxTags {
		return glcLimit("A vault can have at most %d tags", glcMaxTags)
	}
	for k, val := range tags {
		v.tags[k] = val
	}
	return nil
}

func (st *glcStore) removeTags(name string, keys []string) *glcErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return err
	}
	for _, k := range keys {
		delete(v.tags, k)
	}
	return nil
}

func (st *glcStore) listTags(name string) (map[string]any, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return nil, err
	}
	tags := make(map[string]string, len(v.tags))
	for k, val := range v.tags {
		tags[k] = val
	}
	return map[string]any{"Tags": tags}, nil
}

func (st *glcStore) setAccessPolicy(name, policy string) *glcErr {
	if err := glcCheckPolicy(policy); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return err
	}
	v.policy = policy
	return nil
}

func (st *glcStore) accessPolicy(name string, remove bool) (map[string]any, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return nil, err
	}
	if remove {
		v.policy = ""
		return nil, nil
	}
	if v.policy == "" {
		return nil, glcNotFound("No vault access policy is set for: %s", glcVaultARN(name))
	}
	return map[string]any{"Policy": v.policy}, nil
}

var glcNotifyEvents = map[string]bool{"ArchiveRetrievalCompleted": true, "InventoryRetrievalCompleted": true}

func glcCheckNotify(n glcNotify) *glcErr {
	if n.Topic == "" {
		return glcMissing("SNSTopic is required")
	}
	if len(n.Events) == 0 {
		return glcMissing("Events is required")
	}
	if len(n.Events) > len(glcNotifyEvents) {
		return glcInvalid("Events can list each event at most once")
	}
	for _, e := range n.Events {
		if !glcNotifyEvents[e] {
			return glcInvalid("Invalid event %q. Events are ArchiveRetrievalCompleted and InventoryRetrievalCompleted", e)
		}
	}
	return nil
}

func (st *glcStore) setNotifications(name string, n glcNotify) *glcErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return err
	}
	v.notify = &glcNotify{Topic: n.Topic, Events: append([]string(nil), n.Events...)}
	return nil
}

func (st *glcStore) notifications(name string, remove bool) (map[string]any, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return nil, err
	}
	if remove {
		v.notify = nil
		return nil, nil
	}
	if v.notify == nil {
		return nil, glcNotFound("No notification configuration is set for: %s", glcVaultARN(name))
	}
	return map[string]any{"SNSTopic": v.notify.Topic, "Events": append([]string(nil), v.notify.Events...)}, nil
}

// --- vault lock ---

// liveLockLocked drops an InProgress lock whose 24-hour window has passed,
// as AWS does, and returns what's left.
func (v *glcVault) liveLockLocked() *glcLock {
	if v.lock != nil && v.lock.state == "InProgress" && glcNow().After(v.lock.expiresAt) {
		v.lock = nil
	}
	return v.lock
}

func (st *glcStore) initiateLock(name, policy string) (string, *glcErr) {
	if err := glcCheckPolicy(policy); err != nil {
		return "", err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return "", err
	}
	switch l := v.liveLockLocked(); {
	case l != nil && l.state == "Locked":
		return "", glcInvalid("Vault is already locked: %s", glcVaultARN(name))
	case l != nil:
		return "", glcInvalid("A vault lock is already in progress for: %s. Abort it first", glcVaultARN(name))
	}
	now := glcNow()
	v.lock = &glcLock{id: glcRandomID(32), policy: policy, state: "InProgress", created: now, expiresAt: now.Add(glcLockWindow)}
	return v.lock.id, nil
}

func (st *glcStore) completeLock(name, lockID string) *glcErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return err
	}
	l := v.liveLockLocked()
	if l == nil {
		return glcInvalid("No vault lock is in progress for: %s", glcVaultARN(name))
	}
	if l.id != lockID {
		return glcInvalid("Invalid lock ID: %s", lockID)
	}
	l.state = "Locked" // completing a Locked vault with its own id is a no-op
	return nil
}

func (st *glcStore) abortLock(name string) *glcErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return err
	}
	if l := v.liveLockLocked(); l != nil && l.state == "Locked" {
		return glcInvalid("A completed vault lock can't be aborted: %s", glcVaultARN(name))
	}
	v.lock = nil
	return nil
}

func (st *glcStore) getLock(name string) (map[string]any, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.vaultLocked(name)
	if err != nil {
		return nil, err
	}
	l := v.liveLockLocked()
	if l == nil {
		return nil, glcNotFound("No vault lock policy is set for: %s", glcVaultARN(name))
	}
	out := map[string]any{"Policy": l.policy, "State": l.state, "CreationDate": glcTime(l.created)}
	if l.state == "InProgress" {
		out["ExpirationDate"] = glcTime(l.expiresAt)
	}
	return out, nil
}

// --- data retrieval policy, provisioned capacity ---

func glcCheckRules(rules []glcRule) *glcErr {
	if len(rules) > 1 {
		return glcInvalid("A data retrieval policy has at most one rule")
	}
	for _, r := range rules {
		switch r.Strategy {
		case "BytesPerHour":
			if r.BytesPerHour == nil || *r.BytesPerHour < 1 {
				return glcInvalid("BytesPerHour must be a positive number for the BytesPerHour strategy")
			}
		case "FreeTier", "None":
			if r.BytesPerHour != nil {
				return glcInvalid("BytesPerHour is only allowed with the BytesPerHour strategy")
			}
		case "":
			return glcMissing("Strategy is required")
		default:
			return glcInvalid("Invalid strategy %q. Strategies are BytesPerHour, FreeTier and None", r.Strategy)
		}
	}
	return nil
}

func (st *glcStore) setRetrievalPolicy(rules []glcRule) *glcErr {
	if err := glcCheckRules(rules); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.retrieval = append([]glcRule(nil), rules...)
	return nil
}

func (st *glcStore) retrievalPolicy() map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	return map[string]any{"Policy": map[string]any{"Rules": append([]glcRule{}, st.retrieval...)}}
}

func (st *glcStore) purchaseCapacity() (string, *glcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	now := glcNow()
	live := st.capacity[:0]
	for _, c := range st.capacity {
		if exp, err := time.Parse(glcTimeLayout, c.Expiration); err == nil && exp.After(now) {
			live = append(live, c)
		}
	}
	st.capacity = live
	if len(st.capacity) >= glcMaxCapacityUnits {
		return "", glcLimit("An account can have at most %d provisioned capacity units", glcMaxCapacityUnits)
	}
	c := glcCapacity{ID: strings.ToUpper(glcRandomID(24)), StartDate: glcTime(now), Expiration: glcTime(now.AddDate(0, 1, 0))}
	st.capacity = append(st.capacity, c)
	return c.ID, nil
}

func (st *glcStore) listCapacity() map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	return map[string]any{"ProvisionedCapacityList": append([]glcCapacity{}, st.capacity...)}
}
