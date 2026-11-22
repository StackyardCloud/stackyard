package server

import (
	"crypto/rand"
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
	efsRegion    = "us-east-1"
	efsAccountID = "123456789012"
	// efsEmptySizeBytes is what EFS reports for a new, empty file system.
	efsEmptySizeBytes = 6144
	// ponytail: per-account quotas from the EFS docs, not configurable.
	efsMaxFileSystems         = 1000
	efsMaxAccessPointsPerFS   = 10000
	efsMaxTags                = 50
	efsMaxSecurityGroups      = 5
	efsMaxProvisionedMibps    = 3414
	efsMaxPolicyBytes         = 20000
	efsMaxLifecyclePolicies   = 3
	efsDefaultKMSKeyID        = "8f2c7a10-5b3e-4c4d-9e6f-0a1b2c3d4e5f"
	efsDefaultResourceIDType  = "LONG_ID"
	efsReplicationEnabled     = "ENABLED"
	efsReplicationEnabling    = "ENABLING"
	efsProtectionReplicating  = "REPLICATING"
	efsProtectionEnabled      = "ENABLED"
	efsProtectionDisabled     = "DISABLED"
	efsLifeCycleAvailable     = "available"
	efsThroughputProvisioned  = "provisioned"
	efsThroughputElastic      = "elastic"
	efsPerformanceMaxIO       = "maxIO"
	efsPerformanceGeneral     = "generalPurpose"
	efsThroughputBursting     = "bursting"
	efsBackupEnabled          = "ENABLED"
	efsBackupDisabled         = "DISABLED"
	efsDefaultMountTargetPage = 10
	efsDefaultPage            = 100
)

var (
	efsFileSystemIDPattern  = regexp.MustCompile(`^(arn:aws[-a-z]*:elasticfilesystem:[0-9a-z-:]+:file-system/fs-[0-9a-f]{8,40}|fs-[0-9a-f]{8,40})$`)
	efsAccessPointIDPattern = regexp.MustCompile(`^(arn:aws[-a-z]*:elasticfilesystem:[0-9a-z-:]+:access-point/fsap-[0-9a-f]{8,40}|fsap-[0-9a-f]{8,40})$`)
	efsMountTargetIDPattern = regexp.MustCompile(`^fsmt-[0-9a-f]{8,40}$`)
	efsKMSKeyIDPattern      = regexp.MustCompile(`^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|mrk-[0-9a-f]{32}|alias/[a-zA-Z0-9/_-]+|(arn:aws[-a-z]*:kms:[a-z0-9-]+:\d{12}:((key/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|mrk-[0-9a-f]{32}))|(alias/[a-zA-Z0-9/_-]+))))$`)
	efsPermissionsPattern   = regexp.MustCompile(`^[0-7]{3,4}$`)
)

// efsError is an EFS API error: the ErrorCode, its HTTP status and any extra
// members the error shape carries (FileSystemId, AccessPointId).
type efsError struct {
	status int
	code   string
	msg    string
	extra  map[string]string
}

func (e *efsError) Error() string { return e.code + ": " + e.msg }

func efsErr(status int, code, format string, args ...any) *efsError {
	return &efsError{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func efsBadRequest(format string, args ...any) *efsError {
	return efsErr(http.StatusBadRequest, "BadRequest", format, args...)
}

type efsTag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type efsFileSystem struct {
	seq                  int
	ID                   string
	ARN                  string
	CreationToken        string
	CreationTime         time.Time
	PerformanceMode      string
	ThroughputMode       string
	ProvisionedMibps     *float64
	Encrypted            bool
	KmsKeyID             string
	AvailabilityZoneName string
	Tags                 []efsTag
	Protection           string
	BackupStatus         string
	Policy               string
	Lifecycle            []map[string]string
	// ReplicationSource is the source file system ID of the replication this
	// file system takes part in (its own ID when it is the source).
	ReplicationSource string
}

type efsStore struct {
	mu             sync.Mutex
	seq            int
	fileSystems    map[string]*efsFileSystem
	mountTargets   map[string]*efsMountTarget
	accessPoints   map[string]*efsAccessPoint
	replications   map[string]*efsReplication // keyed by source file system ID
	resourceIDType string
}

func newEFSStore() *efsStore {
	return &efsStore{
		fileSystems:    map[string]*efsFileSystem{},
		mountTargets:   map[string]*efsMountTarget{},
		accessPoints:   map[string]*efsAccessPoint{},
		replications:   map[string]*efsReplication{},
		resourceIDType: efsDefaultResourceIDType,
	}
}

func efsTime(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

func efsRandomHex(n int) string {
	b := make([]byte, (n+1)/2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

// newIDLocked returns prefix-<hex>, 17 hex digits for LONG_ID and 8 for
// SHORT_ID (the account preference only affects file systems and mount
// targets; access points always get long IDs).
func (s *efsStore) newIDLocked(prefix string, honorPreference bool) string {
	n := 17
	if honorPreference && s.resourceIDType == "SHORT_ID" {
		n = 8
	}
	return prefix + "-" + efsRandomHex(n)
}

func efsFileSystemARN(id string) string {
	return fmt.Sprintf("arn:aws:elasticfilesystem:%s:%s:file-system/%s", efsRegion, efsAccountID, id)
}

func efsAccessPointARN(id string) string {
	return fmt.Sprintf("arn:aws:elasticfilesystem:%s:%s:access-point/%s", efsRegion, efsAccountID, id)
}

// efsResourceID strips an ARN down to its trailing resource ID.
func efsResourceID(idOrARN string) string {
	if i := strings.LastIndex(idOrARN, "/"); strings.HasPrefix(idOrARN, "arn:") && i >= 0 {
		return idOrARN[i+1:]
	}
	return idOrARN
}

// efsAZID maps an AZ name to an AZ ID the way AWS spells them for this
// region family: us-east-1a -> use1-az1.
func efsAZID(az string) string {
	if len(az) < 2 {
		return ""
	}
	region, letter := az[:len(az)-1], az[len(az)-1]
	parts := strings.Split(region, "-")
	if len(parts) != 3 || parts[1] == "" || letter < 'a' || letter > 'z' {
		return ""
	}
	return fmt.Sprintf("%s%c%s-az%d", parts[0], parts[1][0], parts[2], letter-'a'+1)
}

func efsNormalizeKMSKey(key string) string {
	switch {
	case key == "":
		return fmt.Sprintf("arn:aws:kms:%s:%s:key/%s", efsRegion, efsAccountID, efsDefaultKMSKeyID)
	case strings.HasPrefix(key, "arn:"):
		return key
	case strings.HasPrefix(key, "alias/"):
		return fmt.Sprintf("arn:aws:kms:%s:%s:%s", efsRegion, efsAccountID, key)
	default:
		return fmt.Sprintf("arn:aws:kms:%s:%s:key/%s", efsRegion, efsAccountID, key)
	}
}

// fileSystemLocked resolves an ID or ARN, validating the format first.
func (s *efsStore) fileSystemLocked(idOrARN string) (*efsFileSystem, *efsError) {
	if !efsFileSystemIDPattern.MatchString(idOrARN) {
		return nil, efsBadRequest("FileSystemId %q is not a valid file system ID or ARN", idOrARN)
	}
	fs, ok := s.fileSystems[efsResourceID(idOrARN)]
	if !ok {
		return nil, efsErr(http.StatusNotFound, "FileSystemNotFound", "File system '%s' does not exist.", efsResourceID(idOrARN))
	}
	return fs, nil
}

func (s *efsStore) mountTargetCountLocked(fsID string) int {
	n := 0
	for _, mt := range s.mountTargets {
		if mt.FileSystemID == fsID {
			n++
		}
	}
	return n
}

// describeFileSystemLocked renders a FileSystemDescription.
func (s *efsStore) describeFileSystemLocked(fs *efsFileSystem, state string) map[string]any {
	now := efsTime(time.Now())
	out := map[string]any{
		"OwnerId":              efsAccountID,
		"CreationToken":        fs.CreationToken,
		"FileSystemId":         fs.ID,
		"FileSystemArn":        fs.ARN,
		"CreationTime":         efsTime(fs.CreationTime),
		"LifeCycleState":       state,
		"NumberOfMountTargets": s.mountTargetCountLocked(fs.ID),
		"SizeInBytes": map[string]any{
			"Value": efsEmptySizeBytes, "Timestamp": now,
			"ValueInStandard": efsEmptySizeBytes, "ValueInIA": 0, "ValueInArchive": 0,
		},
		"PerformanceMode":      fs.PerformanceMode,
		"Encrypted":            fs.Encrypted,
		"ThroughputMode":       fs.ThroughputMode,
		"Tags":                 efsCopyTags(fs.Tags),
		"FileSystemProtection": map[string]any{"ReplicationOverwriteProtection": fs.Protection},
	}
	if name, ok := efsTagValue(fs.Tags, "Name"); ok {
		out["Name"] = name
	}
	if fs.Encrypted {
		out["KmsKeyId"] = fs.KmsKeyID
	}
	if fs.ProvisionedMibps != nil {
		out["ProvisionedThroughputInMibps"] = *fs.ProvisionedMibps
	}
	if fs.AvailabilityZoneName != "" {
		out["AvailabilityZoneName"] = fs.AvailabilityZoneName
		if id := efsAZID(fs.AvailabilityZoneName); id != "" {
			out["AvailabilityZoneId"] = id
		}
	}
	return out
}

type efsCreateFileSystemInput struct {
	CreationToken                *string  `json:"CreationToken"`
	PerformanceMode              *string  `json:"PerformanceMode"`
	Encrypted                    *bool    `json:"Encrypted"`
	KmsKeyId                     *string  `json:"KmsKeyId"`
	ThroughputMode               *string  `json:"ThroughputMode"`
	ProvisionedThroughputInMibps *float64 `json:"ProvisionedThroughputInMibps"`
	AvailabilityZoneName         *string  `json:"AvailabilityZoneName"`
	Backup                       *bool    `json:"Backup"`
	Tags                         []efsTag `json:"Tags"`
}

func efsValidateThroughput(perf, mode string, mibps *float64, oneZone bool) *efsError {
	switch mode {
	case efsThroughputBursting, efsThroughputElastic, efsThroughputProvisioned:
	default:
		return efsBadRequest("ThroughputMode must be one of bursting, provisioned or elastic.")
	}
	if mode == efsThroughputProvisioned && mibps == nil {
		return efsBadRequest("ProvisionedThroughputInMibps is required when ThroughputMode is provisioned.")
	}
	if mode != efsThroughputProvisioned && mibps != nil {
		return efsBadRequest("ProvisionedThroughputInMibps can only be set when ThroughputMode is provisioned.")
	}
	if mibps != nil {
		if *mibps < 1 {
			return efsBadRequest("ProvisionedThroughputInMibps must be at least 1.0.")
		}
		if *mibps > efsMaxProvisionedMibps {
			return efsErr(http.StatusBadRequest, "ThroughputLimitExceeded", "ProvisionedThroughputInMibps exceeds the account limit of %d MiB/s.", efsMaxProvisionedMibps)
		}
	}
	if perf == efsPerformanceMaxIO && mode == efsThroughputElastic {
		return efsBadRequest("Max I/O performance mode is not supported with Elastic throughput.")
	}
	if perf == efsPerformanceMaxIO && oneZone {
		return efsBadRequest("Max I/O performance mode is not supported for One Zone file systems.")
	}
	return nil
}

func (s *efsStore) createFileSystem(in efsCreateFileSystemInput) (map[string]any, *efsError) {
	token := strings.TrimSpace(efsDeref(in.CreationToken))
	if token == "" || len(token) > 64 {
		return nil, efsBadRequest("CreationToken is required and must be 1-64 characters.")
	}
	perf := efsDefault(in.PerformanceMode, efsPerformanceGeneral)
	if perf != efsPerformanceGeneral && perf != efsPerformanceMaxIO {
		return nil, efsBadRequest("PerformanceMode must be generalPurpose or maxIO.")
	}
	az := efsDeref(in.AvailabilityZoneName)
	if az != "" && (!strings.HasPrefix(az, efsRegion) || len(az) != len(efsRegion)+1 || efsAZID(az) == "") {
		return nil, efsErr(http.StatusBadRequest, "UnsupportedAvailabilityZone", "Availability Zone '%s' is not supported.", az)
	}
	mode := efsDefault(in.ThroughputMode, efsThroughputBursting)
	if err := efsValidateThroughput(perf, mode, in.ProvisionedThroughputInMibps, az != ""); err != nil {
		return nil, err
	}
	encrypted := in.Encrypted != nil && *in.Encrypted
	kms := efsDeref(in.KmsKeyId)
	if kms != "" && !encrypted {
		return nil, efsBadRequest("KmsKeyId can only be specified when Encrypted is true.")
	}
	if kms != "" && !efsKMSKeyIDPattern.MatchString(kms) {
		return nil, efsBadRequest("KmsKeyId %q is not a valid KMS key ID, ARN or alias.", kms)
	}
	tags, err := efsValidateTags(in.Tags)
	if err != nil {
		return nil, err
	}
	// Automatic backups default on for One Zone file systems only.
	backup := az != ""
	if in.Backup != nil {
		backup = *in.Backup
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, fs := range s.fileSystems {
		if fs.CreationToken == token {
			e := efsErr(http.StatusConflict, "FileSystemAlreadyExists", "File system '%s' already exists with creation token '%s'.", fs.ID, token)
			e.extra = map[string]string{"FileSystemId": fs.ID}
			return nil, e
		}
	}
	if len(s.fileSystems) >= efsMaxFileSystems {
		return nil, efsErr(http.StatusForbidden, "FileSystemLimitExceeded", "You have reached the maximum number of file systems allowed for your account.")
	}
	fs := s.addFileSystemLocked(token, perf, mode, in.ProvisionedThroughputInMibps, encrypted, kms, az, tags)
	if backup {
		fs.BackupStatus = efsBackupEnabled
	}
	return s.describeFileSystemLocked(fs, "creating"), nil
}

func (s *efsStore) addFileSystemLocked(token, perf, mode string, mibps *float64, encrypted bool, kms, az string, tags []efsTag) *efsFileSystem {
	s.seq++
	id := s.newIDLocked("fs", true)
	fs := &efsFileSystem{
		seq: s.seq, ID: id, ARN: efsFileSystemARN(id), CreationToken: token, CreationTime: time.Now().UTC(),
		PerformanceMode: perf, ThroughputMode: mode, Encrypted: encrypted, AvailabilityZoneName: az,
		Tags: tags, Protection: efsProtectionEnabled, BackupStatus: efsBackupDisabled, Lifecycle: []map[string]string{},
	}
	if mibps != nil {
		v := *mibps
		fs.ProvisionedMibps = &v
	}
	if encrypted {
		fs.KmsKeyID = efsNormalizeKMSKey(kms)
	}
	s.fileSystems[id] = fs
	return fs
}

func (s *efsStore) describeFileSystems(idOrARN, token, marker string, limit int) (map[string]any, *efsError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []*efsFileSystem
	if idOrARN != "" {
		fs, err := s.fileSystemLocked(idOrARN)
		if err != nil {
			return nil, err
		}
		list = []*efsFileSystem{fs}
	} else {
		list = s.sortedFileSystemsLocked()
	}
	out := make([]*efsFileSystem, 0, len(list))
	for _, fs := range list {
		if token == "" || fs.CreationToken == token {
			out = append(out, fs)
		}
	}
	page, next, err := efsPage(out, func(fs *efsFileSystem) string { return fs.ID }, marker, efsDefaultLimit(limit, efsDefaultPage))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(page))
	for _, fs := range page {
		items = append(items, s.describeFileSystemLocked(fs, efsLifeCycleAvailable))
	}
	return efsMarkerResponse("FileSystems", items, marker, next), nil
}

func (s *efsStore) sortedFileSystemsLocked() []*efsFileSystem {
	list := make([]*efsFileSystem, 0, len(s.fileSystems))
	for _, fs := range s.fileSystems {
		list = append(list, fs)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].seq < list[j].seq })
	return list
}

func (s *efsStore) deleteFileSystem(idOrARN string) *efsError {
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, err := s.fileSystemLocked(idOrARN)
	if err != nil {
		return err
	}
	if n := s.mountTargetCountLocked(fs.ID); n > 0 {
		return efsErr(http.StatusConflict, "FileSystemInUse", "File system '%s' has %d mount target(s). Delete them first.", fs.ID, n)
	}
	if fs.ReplicationSource != "" {
		return efsErr(http.StatusConflict, "FileSystemInUse", "File system '%s' is part of a replication configuration. Delete the replication configuration first.", fs.ID)
	}
	for id, ap := range s.accessPoints {
		if ap.FileSystemID == fs.ID {
			delete(s.accessPoints, id)
		}
	}
	delete(s.fileSystems, fs.ID)
	return nil
}

type efsUpdateFileSystemInput struct {
	ThroughputMode               *string  `json:"ThroughputMode"`
	ProvisionedThroughputInMibps *float64 `json:"ProvisionedThroughputInMibps"`
}

func (s *efsStore) updateFileSystem(idOrARN string, in efsUpdateFileSystemInput) (map[string]any, *efsError) {
	if in.ThroughputMode == nil && in.ProvisionedThroughputInMibps == nil {
		return nil, efsBadRequest("Specify ThroughputMode or ProvisionedThroughputInMibps.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, err := s.fileSystemLocked(idOrARN)
	if err != nil {
		return nil, err
	}
	mode := efsDefault(in.ThroughputMode, fs.ThroughputMode)
	mibps := in.ProvisionedThroughputInMibps
	if mibps == nil && mode == efsThroughputProvisioned && fs.ThroughputMode == efsThroughputProvisioned {
		mibps = fs.ProvisionedMibps
	}
	if err := efsValidateThroughput(fs.PerformanceMode, mode, mibps, fs.AvailabilityZoneName != ""); err != nil {
		return nil, err
	}
	fs.ThroughputMode = mode
	fs.ProvisionedMibps = nil
	if mibps != nil {
		v := *mibps
		fs.ProvisionedMibps = &v
	}
	return s.describeFileSystemLocked(fs, "updating"), nil
}

func (s *efsStore) updateProtection(idOrARN, value string) (map[string]any, *efsError) {
	if value != efsProtectionEnabled && value != efsProtectionDisabled {
		return nil, efsBadRequest("ReplicationOverwriteProtection must be ENABLED or DISABLED.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, err := s.fileSystemLocked(idOrARN)
	if err != nil {
		return nil, err
	}
	if fs.Protection == efsProtectionReplicating {
		return nil, efsErr(http.StatusConflict, "ReplicationAlreadyExists", "File system '%s' is a replication destination; its protection can't change.", fs.ID)
	}
	fs.Protection = value
	return map[string]any{"ReplicationOverwriteProtection": fs.Protection}, nil
}

func (s *efsStore) backupPolicy(idOrARN string, set string) (map[string]any, *efsError) {
	if set != "" && set != efsBackupEnabled && set != efsBackupDisabled {
		return nil, efsErr(http.StatusBadRequest, "ValidationException", "BackupPolicy.Status must be ENABLED or DISABLED.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, err := s.fileSystemLocked(idOrARN)
	if err != nil {
		return nil, err
	}
	if set != "" {
		fs.BackupStatus = set
	}
	return map[string]any{"BackupPolicy": map[string]any{"Status": fs.BackupStatus}}, nil
}

// efsValidatePolicy checks the policy is a JSON IAM-style document.
func efsValidatePolicy(policy string) *efsError {
	if policy == "" || len(policy) > efsMaxPolicyBytes {
		return efsBadRequest("Policy is required and must be at most %d characters.", efsMaxPolicyBytes)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(policy), &doc); err != nil {
		return efsErr(http.StatusBadRequest, "InvalidPolicyException", "Policy is not valid JSON: %v", err)
	}
	switch stmt := doc["Statement"].(type) {
	case []any:
		if len(stmt) == 0 {
			return efsErr(http.StatusBadRequest, "InvalidPolicyException", "Policy must contain at least one statement.")
		}
	case map[string]any:
	default:
		return efsErr(http.StatusBadRequest, "InvalidPolicyException", "Policy must contain a Statement.")
	}
	return nil
}

func (s *efsStore) fileSystemPolicy(idOrARN string, put *string, remove bool) (map[string]any, *efsError) {
	if put != nil {
		if err := efsValidatePolicy(*put); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, err := s.fileSystemLocked(idOrARN)
	if err != nil {
		return nil, err
	}
	switch {
	case remove:
		fs.Policy = ""
		return map[string]any{}, nil
	case put != nil:
		fs.Policy = *put
	case fs.Policy == "":
		return nil, efsErr(http.StatusNotFound, "PolicyNotFound", "File system '%s' has no policy; the default policy is in effect.", fs.ID)
	}
	return map[string]any{"FileSystemId": fs.ID, "Policy": fs.Policy}, nil
}

var efsLifecycleRules = map[string]map[string]bool{
	"TransitionToIA":                  efsSet("AFTER_1_DAY", "AFTER_7_DAYS", "AFTER_14_DAYS", "AFTER_30_DAYS", "AFTER_60_DAYS", "AFTER_90_DAYS", "AFTER_180_DAYS", "AFTER_270_DAYS", "AFTER_365_DAYS"),
	"TransitionToPrimaryStorageClass": efsSet("AFTER_1_ACCESS"),
	"TransitionToArchive":             efsSet("AFTER_1_DAY", "AFTER_7_DAYS", "AFTER_14_DAYS", "AFTER_30_DAYS", "AFTER_60_DAYS", "AFTER_90_DAYS", "AFTER_180_DAYS", "AFTER_270_DAYS", "AFTER_365_DAYS"),
}

func (s *efsStore) lifecycle(idOrARN string, put []map[string]string, isPut bool) (map[string]any, *efsError) {
	if isPut {
		if len(put) > efsMaxLifecyclePolicies {
			return nil, efsBadRequest("LifecyclePolicies can hold at most %d policies.", efsMaxLifecyclePolicies)
		}
		seen := map[string]bool{}
		for _, policy := range put {
			if len(policy) != 1 {
				return nil, efsBadRequest("Each lifecycle policy must set exactly one transition.")
			}
			for key, value := range policy {
				if !efsLifecycleRules[key][value] {
					return nil, efsBadRequest("Invalid lifecycle policy %s=%q.", key, value)
				}
				if seen[key] {
					return nil, efsBadRequest("Lifecycle policy %s is specified more than once.", key)
				}
				seen[key] = true
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	fs, err := s.fileSystemLocked(idOrARN)
	if err != nil {
		return nil, err
	}
	if isPut {
		for _, policy := range put {
			if _, ok := policy["TransitionToArchive"]; ok && fs.ThroughputMode != efsThroughputElastic {
				return nil, efsBadRequest("TransitionToArchive requires a file system with Elastic throughput.")
			}
		}
		fs.Lifecycle = put
	}
	policies := make([]map[string]string, 0, len(fs.Lifecycle))
	for _, p := range fs.Lifecycle {
		cp := map[string]string{}
		for k, v := range p {
			cp[k] = v
		}
		policies = append(policies, cp)
	}
	return map[string]any{"LifecyclePolicies": policies}, nil
}

func (s *efsStore) accountPreferences(set string) (map[string]any, *efsError) {
	if set != "" && set != "LONG_ID" && set != "SHORT_ID" {
		return nil, efsBadRequest("ResourceIdType must be LONG_ID or SHORT_ID.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if set != "" {
		s.resourceIDType = set
	}
	return map[string]any{"ResourceIdPreference": map[string]any{
		"ResourceIdType": s.resourceIDType,
		"Resources":      []string{"FILE_SYSTEM", "MOUNT_TARGET"},
	}}, nil
}

func efsSet(values ...string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, v := range values {
		out[v] = true
	}
	return out
}

func efsDeref(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func efsDefault(v *string, def string) string {
	if s := efsDeref(v); s != "" {
		return s
	}
	return def
}

func efsDefaultLimit(limit, def int) int {
	if limit <= 0 {
		return def
	}
	return limit
}

// efsPage returns the page that starts at marker (an item ID) and the marker
// of the next page.
func efsPage[T any](items []T, id func(T) string, marker string, limit int) ([]T, string, *efsError) {
	start := 0
	if marker != "" {
		start = -1
		for i, item := range items {
			if id(item) == marker {
				start = i
				break
			}
		}
		if start < 0 {
			return nil, "", efsBadRequest("Marker/NextToken %q is not valid.", marker)
		}
	}
	end := min(start+limit, len(items))
	next := ""
	if end < len(items) {
		next = id(items[end])
	}
	return items[start:end], next, nil
}

func efsMarkerResponse(key string, items any, marker, next string) map[string]any {
	out := map[string]any{key: items}
	if marker != "" {
		out["Marker"] = marker
	}
	if next != "" {
		out["NextMarker"] = next
	}
	return out
}
