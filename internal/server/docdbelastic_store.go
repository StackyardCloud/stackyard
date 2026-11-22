package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	dbeAccountID      = "123456789012"
	dbeRegion         = "us-east-1"
	dbeOwnedKMSKey    = "AWS_OWNED_KMS_KEY"
	dbeMaxClusters    = 1000 // ponytail: well above the AWS quota so repeated test runs don't hit it
	dbeMaxSnapshots   = 5000
	dbeMaxTags        = 50
	dbeMaxIDList      = 20 // subnet ids or security groups per cluster
	dbeMaxBody        = 1 << 20
	dbeMaintenanceOp  = "SYSTEM_UPDATE"
	dbeMaintenanceMsg = "A new system update is available for this cluster."
)

var (
	dbeNamePattern     = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	dbeUserPattern     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]{0,62}$`)
	dbeSecretPattern   = regexp.MustCompile(`^arn:aws[a-z-]*:secretsmanager:[a-z0-9-]+:\d{12}:secret:.+$`)
	dbeBackupWindow    = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)-([01]\d|2[0-3]):([0-5]\d)$`)
	dbeMaintWindow     = regexp.MustCompile(`^(mon|tue|wed|thu|fri|sat|sun):([01]\d|2[0-3]):([0-5]\d)-(mon|tue|wed|thu|fri|sat|sun):([01]\d|2[0-3]):([0-5]\d)$`)
	dbeShardCapacities = map[int32]bool{2: true, 4: true, 8: true, 16: true, 32: true, 64: true}
	dbeDays            = map[string]int{"mon": 0, "tue": 1, "wed": 2, "thu": 3, "fri": 4, "sat": 5, "sun": 6}
)

// dbeError is a docdb-elastic API error; status and code follow the botocore model.
type dbeError struct {
	status       int
	code         string
	msg          string
	resourceID   string
	resourceType string
}

func dbeValidation(format string, args ...any) *dbeError {
	return &dbeError{status: http.StatusBadRequest, code: "ValidationException", msg: fmt.Sprintf(format, args...)}
}

func dbeNotFound(id, kind string) *dbeError {
	return &dbeError{status: http.StatusNotFound, code: "ResourceNotFoundException", msg: fmt.Sprintf("The %s %s does not exist.", kind, id), resourceID: id, resourceType: kind}
}

func dbeConflict(id, kind, format string, args ...any) *dbeError {
	return &dbeError{status: http.StatusConflict, code: "ConflictException", msg: fmt.Sprintf(format, args...), resourceID: id, resourceType: kind}
}

func dbeQuota(format string, args ...any) *dbeError {
	return &dbeError{status: http.StatusPaymentRequired, code: "ServiceQuotaExceededException", msg: fmt.Sprintf(format, args...)}
}

type dbeShard struct {
	id      string
	created string
}

type dbeAction struct {
	optIn     string // "", "immediate", "next-maintenance", "apply-on"
	applyDate string
}

type dbeCluster struct {
	arn, name, adminUser, authType, password  string
	shardCapacity, shardCount, shardInstances int32
	backupRetention                           int32
	backupWindow, maintWindow, kmsKeyID       string
	subnets, groups                           []string
	status                                    string
	created                                   string
	shards                                    []dbeShard
	activated                                 bool       // has settled to ACTIVE at least once
	pending                                   *dbeAction // the SYSTEM_UPDATE maintenance action, if not yet applied
}

type dbeSnapshot struct {
	arn, name, clusterArn, kind, status string
	created, clusterCreated             string
	adminUser, authType, password       string
	kmsKeyID                            string
	shardCapacity, shardCount           int32
	shardInstances                      int32
	subnets, groups                     []string
}

// dbeStore holds Amazon DocumentDB Elastic Clusters state.
type dbeStore struct {
	mu        sync.Mutex
	clusters  map[string]*dbeCluster
	snapshots map[string]*dbeSnapshot
	order     []string // cluster and snapshot ARNs in creation order, for stable listing
	tags      map[string]map[string]string
	tokens    map[string]string // CreateCluster clientToken -> cluster ARN
}

func newDocDBElasticStore() *dbeStore {
	return &dbeStore{
		clusters:  map[string]*dbeCluster{},
		snapshots: map[string]*dbeSnapshot{},
		tags:      map[string]map[string]string{},
		tokens:    map[string]string{},
	}
}

func dbeNow() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }

func dbeID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func dbeClusterARN(id string) string {
	return "arn:aws:docdb-elastic:" + dbeRegion + ":" + dbeAccountID + ":cluster/" + id
}

func dbeSnapshotARN(id string) string {
	return "arn:aws:docdb-elastic:" + dbeRegion + ":" + dbeAccountID + ":cluster-snapshot/" + id
}

// dbeClusterSpec is a validated create, restore or update request.
type dbeClusterSpec struct {
	name, adminUser, authType, password      string
	shardCapacity, shardCount, shardInstance *int32
	backupRetention                          *int32
	backupWindow, maintWindow, kmsKeyID      string
	subnets, groups                          []string
	tags                                     map[string]string
}

func dbeCheckName(member, name string) *dbeError {
	if len(name) < 1 || len(name) > 63 || !dbeNamePattern.MatchString(name) {
		return dbeValidation("%s must be 1 to 63 lowercase letters, numbers or hyphens, start with a letter, and not end with or contain two consecutive hyphens.", member)
	}
	return nil
}

// dbeCheckAuth validates the admin password (PLAIN_TEXT) or secret ARN (SECRET_ARN).
func dbeCheckAuth(authType, password string, secretExists func(string) bool) *dbeError {
	switch authType {
	case "PLAIN_TEXT":
		if len(password) < 8 || len(password) > 100 || strings.ContainsAny(password, `/"@`) {
			return dbeValidation("adminUserPassword must be 8 to 100 printable ASCII characters and can't contain '/', '\"' or '@'.")
		}
		for _, c := range password {
			if c < 0x20 || c > 0x7e {
				return dbeValidation("adminUserPassword must be 8 to 100 printable ASCII characters and can't contain '/', '\"' or '@'.")
			}
		}
	case "SECRET_ARN":
		if !dbeSecretPattern.MatchString(password) {
			return dbeValidation("adminUserPassword must be a Secrets Manager secret ARN when authType is SECRET_ARN.")
		}
		if !secretExists(password) {
			return dbeValidation("The secret %s does not exist.", password)
		}
	}
	return nil
}

func dbeWindowMinutes(a, b int) int {
	if b <= a {
		b += 7 * 24 * 60
	}
	return b - a
}

func dbeCheckWindows(backup, maint string) *dbeError {
	if backup != "" {
		m := dbeBackupWindow.FindStringSubmatch(backup)
		if m == nil {
			return dbeValidation("preferredBackupWindow must be in the format hh24:mi-hh24:mi.")
		}
		start, end := dbeAtoi(m[1])*60+dbeAtoi(m[2]), dbeAtoi(m[3])*60+dbeAtoi(m[4])
		if end <= start {
			end += 24 * 60
		}
		if end-start < 30 {
			return dbeValidation("preferredBackupWindow must be at least 30 minutes.")
		}
	}
	if maint != "" {
		m := dbeMaintWindow.FindStringSubmatch(strings.ToLower(maint))
		if m == nil {
			return dbeValidation("preferredMaintenanceWindow must be in the format ddd:hh24:mi-ddd:hh24:mi.")
		}
		start := dbeDays[m[1]]*24*60 + dbeAtoi(m[2])*60 + dbeAtoi(m[3])
		end := dbeDays[m[4]]*24*60 + dbeAtoi(m[5])*60 + dbeAtoi(m[6])
		if dbeWindowMinutes(start, end) < 30 {
			return dbeValidation("preferredMaintenanceWindow must be at least 30 minutes.")
		}
	}
	return nil
}

func dbeAtoi(s string) int {
	n, _ := strconv.Atoi(s) // only called on regex-matched two-digit fields
	return n
}

func dbeCheckSizes(spec dbeClusterSpec) *dbeError {
	if spec.shardCapacity != nil && !dbeShardCapacities[*spec.shardCapacity] {
		return dbeValidation("shardCapacity must be one of 2, 4, 8, 16, 32 or 64.")
	}
	if spec.shardCount != nil && (*spec.shardCount < 1 || *spec.shardCount > 32) {
		return dbeValidation("shardCount must be between 1 and 32.")
	}
	if spec.shardInstance != nil && (*spec.shardInstance < 1 || *spec.shardInstance > 16) {
		return dbeValidation("shardInstanceCount must be between 1 and 16.")
	}
	if spec.backupRetention != nil && (*spec.backupRetention < 1 || *spec.backupRetention > 35) {
		return dbeValidation("backupRetentionPeriod must be between 1 and 35.")
	}
	if len(spec.subnets) > dbeMaxIDList || len(spec.groups) > dbeMaxIDList {
		return dbeValidation("A cluster can have at most %d subnets and %d security groups.", dbeMaxIDList, dbeMaxIDList)
	}
	return dbeCheckWindows(spec.backupWindow, spec.maintWindow)
}

func dbeCheckTags(tags map[string]string) *dbeError {
	if len(tags) > dbeMaxTags {
		return dbeValidation("A resource can have at most %d tags.", dbeMaxTags)
	}
	for k, v := range tags {
		if k == "" || len(k) > 128 || strings.HasPrefix(strings.ToLower(k), "aws:") {
			return dbeValidation("Tag keys must be 1 to 128 characters and can't start with aws:.")
		}
		if len(v) > 256 {
			return dbeValidation("Tag values can be at most 256 characters.")
		}
	}
	return nil
}

func dbeShards(n int32, created string) []dbeShard {
	out := make([]dbeShard, n)
	for i := range out {
		out[i] = dbeShard{id: dbeID()[:12], created: created}
	}
	return out
}

func (st *dbeStore) addTagsLocked(arn string, tags map[string]string) {
	if len(tags) == 0 {
		return
	}
	m := st.tags[arn]
	if m == nil {
		m = map[string]string{}
		st.tags[arn] = m
	}
	for k, v := range tags {
		m[k] = v
	}
}

// settleLocked moves a cluster out of a transitional state. Only reads (Get*,
// List*) call it, so a change made before the next read sees the transition.
// ponytail: one read per transition, a clock when tests need real timing.
func (st *dbeStore) settleLocked(c *dbeCluster) {
	switch c.status {
	case "CREATING", "UPDATING", "STARTING", "MAINTENANCE":
		if c.status == "MAINTENANCE" {
			c.pending = nil
		}
		c.status = "ACTIVE"
	case "STOPPING":
		c.status = "STOPPED"
	default:
		return
	}
	if c.status == "ACTIVE" && !c.activated {
		c.activated = true
		c.pending = &dbeAction{}
		st.addSnapshotLocked(c, "", "AUTOMATED", "ACTIVE")
	}
}

func (st *dbeStore) settleSnapshotLocked(s *dbeSnapshot) {
	if s.status == "CREATING" || s.status == "COPYING" {
		s.status = "ACTIVE"
	}
}

// addSnapshotLocked records a snapshot of c; name "" names an automated one.
func (st *dbeStore) addSnapshotLocked(c *dbeCluster, name, kind, status string) *dbeSnapshot {
	id := dbeID()
	if name == "" {
		name = "rds:" + c.name + "-" + time.Now().UTC().Format("2006-01-02-15-04")
	}
	s := &dbeSnapshot{
		arn: dbeSnapshotARN(id), name: name, clusterArn: c.arn, kind: kind, status: status,
		created: dbeNow(), clusterCreated: c.created, adminUser: c.adminUser, authType: c.authType, password: c.password,
		kmsKeyID: c.kmsKeyID, shardCapacity: c.shardCapacity, shardCount: c.shardCount, shardInstances: c.shardInstances,
		subnets: append([]string(nil), c.subnets...), groups: append([]string(nil), c.groups...),
	}
	st.snapshots[s.arn] = s
	st.order = append(st.order, s.arn)
	return s
}

func (st *dbeStore) clusterLocked(arn string) (*dbeCluster, *dbeError) {
	c, ok := st.clusters[arn]
	if !ok {
		return nil, dbeNotFound(arn, "cluster")
	}
	return c, nil
}

func (st *dbeStore) snapshotLocked(arn string) (*dbeSnapshot, *dbeError) {
	s, ok := st.snapshots[arn]
	if !ok {
		return nil, dbeNotFound(arn, "snapshot")
	}
	return s, nil
}

func (st *dbeStore) nameTakenLocked(name string) bool {
	for _, c := range st.clusters {
		if c.name == name {
			return true
		}
	}
	return false
}

// newClusterLocked creates a cluster from a validated spec; network fields are already resolved.
func (st *dbeStore) newClusterLocked(spec dbeClusterSpec, capacity, count, instances int32) (*dbeCluster, *dbeError) {
	if st.nameTakenLocked(spec.name) {
		return nil, dbeConflict(spec.name, "cluster", "A cluster named %s already exists.", spec.name)
	}
	if len(st.clusters) >= dbeMaxClusters {
		return nil, dbeQuota("You can have at most %d elastic clusters.", dbeMaxClusters)
	}
	now := dbeNow()
	c := &dbeCluster{
		arn: dbeClusterARN(dbeID()), name: spec.name, adminUser: spec.adminUser, authType: spec.authType, password: spec.password,
		shardCapacity: capacity, shardCount: count, shardInstances: instances, backupRetention: 1,
		backupWindow: "00:00-00:30", maintWindow: "sun:05:00-sun:05:30", kmsKeyID: dbeOwnedKMSKey,
		subnets: spec.subnets, groups: spec.groups, status: "CREATING", created: now, shards: dbeShards(count, now),
	}
	if spec.backupRetention != nil {
		c.backupRetention = *spec.backupRetention
	}
	if spec.backupWindow != "" {
		c.backupWindow = spec.backupWindow
	}
	if spec.maintWindow != "" {
		c.maintWindow = strings.ToLower(spec.maintWindow)
	}
	if spec.kmsKeyID != "" {
		c.kmsKeyID = spec.kmsKeyID
	}
	st.clusters[c.arn] = c
	st.order = append(st.order, c.arn)
	st.addTagsLocked(c.arn, spec.tags)
	return c, nil
}

func (st *dbeStore) createCluster(spec dbeClusterSpec, clientToken string) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if arn, ok := st.tokens[clientToken]; ok && clientToken != "" {
		if c, ok := st.clusters[arn]; ok {
			return map[string]any{"cluster": c.out()}, nil
		}
	}
	instances := int32(2)
	if spec.shardInstance != nil {
		instances = *spec.shardInstance
	}
	c, err := st.newClusterLocked(spec, *spec.shardCapacity, *spec.shardCount, instances)
	if err != nil {
		return nil, err
	}
	if clientToken != "" {
		st.tokens[clientToken] = c.arn
	}
	return map[string]any{"cluster": c.out()}, nil
}

func (st *dbeStore) getCluster(arn string) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.clusterLocked(arn)
	if err != nil {
		return nil, err
	}
	st.settleLocked(c)
	return map[string]any{"cluster": c.out()}, nil
}

func (st *dbeStore) updateCluster(arn string, spec dbeClusterSpec, secretExists func(string) bool) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.clusterLocked(arn)
	if err != nil {
		return nil, err
	}
	if c.status != "ACTIVE" {
		return nil, dbeConflict(arn, "cluster", "The cluster is %s; it must be ACTIVE to be updated.", c.status)
	}
	authType := c.authType
	if spec.authType != "" {
		authType = spec.authType
	}
	if spec.authType != "" && spec.authType != c.authType && spec.password == "" {
		return nil, dbeValidation("adminUserPassword is required when changing authType.")
	}
	if spec.password != "" {
		if err := dbeCheckAuth(authType, spec.password, secretExists); err != nil {
			return nil, err
		}
	}
	// Everything is validated above; apply the change.
	c.authType = authType
	if spec.password != "" {
		c.password = spec.password
	}
	if spec.shardCapacity != nil {
		c.shardCapacity = *spec.shardCapacity
	}
	if spec.shardInstance != nil {
		c.shardInstances = *spec.shardInstance
	}
	if spec.shardCount != nil && *spec.shardCount != c.shardCount {
		if *spec.shardCount > c.shardCount {
			c.shards = append(c.shards, dbeShards(*spec.shardCount-c.shardCount, dbeNow())...)
		} else {
			c.shards = c.shards[:*spec.shardCount]
		}
		c.shardCount = *spec.shardCount
	}
	if spec.backupRetention != nil {
		c.backupRetention = *spec.backupRetention
	}
	if spec.backupWindow != "" {
		c.backupWindow = spec.backupWindow
	}
	if spec.maintWindow != "" {
		c.maintWindow = strings.ToLower(spec.maintWindow)
	}
	if spec.subnets != nil {
		c.subnets = spec.subnets
	}
	if spec.groups != nil {
		c.groups = spec.groups
	}
	c.status = "UPDATING"
	return map[string]any{"cluster": c.out()}, nil
}

// clusterNetwork returns a cluster's subnets for an update that changes only its security groups.
func (st *dbeStore) clusterNetwork(arn string) ([]string, []string, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.clusterLocked(arn)
	if err != nil {
		return nil, nil, err
	}
	return append([]string(nil), c.subnets...), append([]string(nil), c.groups...), nil
}

func (st *dbeStore) deleteCluster(arn string) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.clusterLocked(arn)
	if err != nil {
		return nil, err
	}
	if c.status != "ACTIVE" && c.status != "STOPPED" {
		return nil, dbeConflict(arn, "cluster", "The cluster is %s; it must be ACTIVE or STOPPED to be deleted.", c.status)
	}
	c.status = "DELETING"
	out := c.out()
	// ponytail: deletion completes at once; automated snapshots go with the cluster, manual ones stay.
	delete(st.clusters, arn)
	delete(st.tags, arn)
	for sarn, s := range st.snapshots {
		if s.clusterArn == arn && s.kind == "AUTOMATED" {
			delete(st.snapshots, sarn)
			delete(st.tags, sarn)
		}
	}
	st.pruneOrderLocked()
	return map[string]any{"cluster": out}, nil
}

func (st *dbeStore) pruneOrderLocked() {
	kept := st.order[:0]
	for _, arn := range st.order {
		if st.clusters[arn] != nil || st.snapshots[arn] != nil {
			kept = append(kept, arn)
		}
	}
	st.order = kept
	for token, arn := range st.tokens {
		if st.clusters[arn] == nil {
			delete(st.tokens, token)
		}
	}
}

func (st *dbeStore) setRunning(arn string, start bool) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.clusterLocked(arn)
	if err != nil {
		return nil, err
	}
	switch {
	case start && c.status == "STOPPED":
		c.status = "STARTING"
	case !start && c.status == "ACTIVE":
		c.status = "STOPPING"
	case start:
		return nil, dbeValidation("Cluster %s is %s; only a STOPPED cluster can be started.", arn, c.status)
	default:
		return nil, dbeValidation("Cluster %s is %s; only an ACTIVE cluster can be stopped.", arn, c.status)
	}
	return map[string]any{"cluster": c.out()}, nil
}

// listPage returns ARNs (in creation order) matching keep, paged by an opaque offset token.
func (st *dbeStore) listPageLocked(keep func(string) bool, token string, limit int) ([]string, string, *dbeError) {
	start := 0
	if token != "" {
		raw, err := base64.RawURLEncoding.DecodeString(token)
		n, perr := strconv.ParseInt(strings.TrimPrefix(string(raw), "dbe:"), 10, 32)
		if err != nil || perr != nil || !strings.HasPrefix(string(raw), "dbe:") || n < 0 {
			return nil, "", dbeValidation("The nextToken is not valid.")
		}
		start = int(n)
	}
	var matched []string
	for _, arn := range st.order {
		if keep(arn) {
			matched = append(matched, arn)
		}
	}
	if start > len(matched) {
		start = len(matched)
	}
	end := min(start+limit, len(matched))
	next := ""
	if end < len(matched) {
		next = base64.RawURLEncoding.EncodeToString([]byte("dbe:" + strconv.Itoa(end)))
	}
	return matched[start:end], next, nil
}

func (st *dbeStore) listClusters(token string, limit int) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	arns, next, err := st.listPageLocked(func(arn string) bool { return st.clusters[arn] != nil }, token, limit)
	if err != nil {
		return nil, err
	}
	items := []map[string]any{}
	for _, arn := range arns {
		c := st.clusters[arn]
		st.settleLocked(c)
		items = append(items, map[string]any{"clusterArn": c.arn, "clusterName": c.name, "status": c.status})
	}
	out := map[string]any{"clusters": items}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

func (st *dbeStore) createSnapshot(clusterArn, name string, tags map[string]string) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.clusterLocked(clusterArn)
	if err != nil {
		return nil, err
	}
	if c.status != "ACTIVE" {
		return nil, dbeConflict(clusterArn, "cluster", "The cluster is %s; it must be ACTIVE to take a snapshot.", c.status)
	}
	if err := st.snapshotNameFreeLocked(name); err != nil {
		return nil, err
	}
	s := st.addSnapshotLocked(c, name, "MANUAL", "CREATING")
	st.addTagsLocked(s.arn, tags)
	return map[string]any{"snapshot": s.out()}, nil
}

func (st *dbeStore) snapshotNameFreeLocked(name string) *dbeError {
	if len(st.snapshots) >= dbeMaxSnapshots {
		return dbeQuota("You can have at most %d cluster snapshots.", dbeMaxSnapshots)
	}
	for _, s := range st.snapshots {
		if s.name == name {
			return dbeConflict(name, "snapshot", "A cluster snapshot named %s already exists.", name)
		}
	}
	return nil
}

func (st *dbeStore) getSnapshot(arn string) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s, err := st.snapshotLocked(arn)
	if err != nil {
		return nil, err
	}
	st.settleSnapshotLocked(s)
	return map[string]any{"snapshot": s.out()}, nil
}

func (st *dbeStore) deleteSnapshot(arn string) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s, err := st.snapshotLocked(arn)
	if err != nil {
		return nil, err
	}
	if s.kind == "AUTOMATED" {
		return nil, dbeValidation("Automated snapshot %s can't be deleted; it is removed with its cluster.", arn)
	}
	if s.status != "ACTIVE" {
		return nil, dbeConflict(arn, "snapshot", "The snapshot is %s; it must be ACTIVE to be deleted.", s.status)
	}
	s.status = "DELETING"
	out := s.out()
	delete(st.snapshots, arn)
	delete(st.tags, arn)
	st.pruneOrderLocked()
	return map[string]any{"snapshot": out}, nil
}

func (st *dbeStore) copySnapshot(arn, target, kmsKeyID string, copyTags bool, tags map[string]string) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	src, err := st.snapshotLocked(arn)
	if err != nil {
		return nil, err
	}
	if src.status != "ACTIVE" {
		return nil, dbeConflict(arn, "snapshot", "The snapshot is %s; it must be ACTIVE to be copied.", src.status)
	}
	if err := st.snapshotNameFreeLocked(target); err != nil {
		return nil, err
	}
	cp := *src
	cp.arn, cp.name, cp.kind, cp.status, cp.created = dbeSnapshotARN(dbeID()), target, "MANUAL", "COPYING", dbeNow()
	cp.subnets, cp.groups = append([]string(nil), src.subnets...), append([]string(nil), src.groups...)
	if kmsKeyID != "" {
		cp.kmsKeyID = kmsKeyID
	}
	st.snapshots[cp.arn] = &cp
	st.order = append(st.order, cp.arn)
	if copyTags {
		st.addTagsLocked(cp.arn, st.tags[arn])
	}
	st.addTagsLocked(cp.arn, tags)
	return map[string]any{"snapshot": cp.out()}, nil
}

// snapshotForRestore returns the settled snapshot a restore starts from.
func (st *dbeStore) snapshotForRestore(arn string) (dbeSnapshot, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s, err := st.snapshotLocked(arn)
	if err != nil {
		return dbeSnapshot{}, err
	}
	if s.status != "ACTIVE" {
		return dbeSnapshot{}, dbeConflict(arn, "snapshot", "The snapshot is %s; it must be ACTIVE to be restored.", s.status)
	}
	return *s, nil
}

func (st *dbeStore) restore(snap dbeSnapshot, spec dbeClusterSpec) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	spec.adminUser, spec.authType, spec.password = snap.adminUser, snap.authType, snap.password
	if spec.kmsKeyID == "" {
		spec.kmsKeyID = snap.kmsKeyID
	}
	capacity, instances := snap.shardCapacity, snap.shardInstances
	if spec.shardCapacity != nil {
		capacity = *spec.shardCapacity
	}
	if spec.shardInstance != nil {
		instances = *spec.shardInstance
	}
	c, err := st.newClusterLocked(spec, capacity, snap.shardCount, instances)
	if err != nil {
		return nil, err
	}
	return map[string]any{"cluster": c.out()}, nil
}

func (st *dbeStore) listSnapshots(clusterArn, kind, token string, limit int) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	arns, next, err := st.listPageLocked(func(arn string) bool {
		s := st.snapshots[arn]
		return s != nil && (clusterArn == "" || s.clusterArn == clusterArn) && (kind == "" || s.kind == kind)
	}, token, limit)
	if err != nil {
		return nil, err
	}
	items := []map[string]any{}
	for _, arn := range arns {
		s := st.snapshots[arn]
		st.settleSnapshotLocked(s)
		items = append(items, map[string]any{"clusterArn": s.clusterArn, "snapshotArn": s.arn, "snapshotCreationTime": s.created, "snapshotName": s.name, "status": s.status})
	}
	out := map[string]any{"snapshots": items}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

func (c *dbeCluster) pendingOut() map[string]any {
	details := []map[string]any{}
	if a := c.pending; a != nil {
		d := map[string]any{"action": dbeMaintenanceOp, "description": dbeMaintenanceMsg}
		if a.optIn != "" {
			d["optInStatus"] = a.optIn
		}
		if a.applyDate != "" {
			d["currentApplyDate"] = a.applyDate
		}
		details = append(details, d)
	}
	return map[string]any{"resourceArn": c.arn, "pendingMaintenanceActionDetails": details}
}

func (st *dbeStore) getPending(arn string) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.clusterLocked(arn)
	if err != nil {
		return nil, err
	}
	st.settleLocked(c)
	return map[string]any{"resourcePendingMaintenanceAction": c.pendingOut()}, nil
}

func (st *dbeStore) listPending(token string, limit int) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, c := range st.clusters {
		st.settleLocked(c)
	}
	arns, next, err := st.listPageLocked(func(arn string) bool { c := st.clusters[arn]; return c != nil && c.pending != nil }, token, limit)
	if err != nil {
		return nil, err
	}
	items := []map[string]any{}
	for _, arn := range arns {
		items = append(items, st.clusters[arn].pendingOut())
	}
	out := map[string]any{"resourcePendingMaintenanceActions": items}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

// applyPending opts a cluster's maintenance action in or out. IMMEDIATE puts the
// cluster into MAINTENANCE; the action is gone once it settles.
func (st *dbeStore) applyPending(arn, action, optIn, applyOn string) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, err := st.clusterLocked(arn)
	if err != nil {
		return nil, err
	}
	if c.pending == nil || action != dbeMaintenanceOp {
		return nil, dbeNotFound(action, "pending maintenance action")
	}
	switch optIn {
	case "IMMEDIATE":
		if c.status != "ACTIVE" {
			return nil, dbeConflict(arn, "cluster", "The cluster is %s; it must be ACTIVE to apply maintenance now.", c.status)
		}
		c.pending.optIn, c.pending.applyDate = "immediate", dbeNow()
		c.status = "MAINTENANCE"
	case "NEXT_MAINTENANCE":
		c.pending.optIn, c.pending.applyDate = "next-maintenance", ""
	case "APPLY_ON":
		if _, perr := time.Parse(time.RFC3339, applyOn); perr != nil {
			return nil, dbeValidation("applyOn must be an ISO 8601 date and time when optInType is APPLY_ON.")
		}
		c.pending.optIn, c.pending.applyDate = "apply-on", applyOn
	case "UNDO_OPT_IN":
		c.pending.optIn, c.pending.applyDate = "", ""
	}
	return map[string]any{"resourcePendingMaintenanceAction": c.pendingOut()}, nil
}

func (st *dbeStore) existsLocked(arn string) bool {
	return st.clusters[arn] != nil || st.snapshots[arn] != nil
}

func (st *dbeStore) listTags(arn string) (map[string]any, *dbeError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.existsLocked(arn) {
		return nil, dbeNotFound(arn, "resource")
	}
	tags := map[string]string{}
	for k, v := range st.tags[arn] {
		tags[k] = v
	}
	return map[string]any{"tags": tags}, nil
}

func (st *dbeStore) tag(arn string, tags map[string]string) *dbeError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.existsLocked(arn) {
		return dbeNotFound(arn, "resource")
	}
	merged := map[string]string{}
	for k, v := range st.tags[arn] {
		merged[k] = v
	}
	for k, v := range tags {
		merged[k] = v
	}
	if err := dbeCheckTags(merged); err != nil {
		return err
	}
	st.tags[arn] = merged
	return nil
}

func (st *dbeStore) untag(arn string, keys []string) *dbeError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.existsLocked(arn) {
		return dbeNotFound(arn, "resource")
	}
	for _, k := range keys {
		delete(st.tags[arn], k)
	}
	return nil
}

func (c *dbeCluster) out() map[string]any {
	shards := make([]map[string]any, 0, len(c.shards))
	for _, s := range c.shards {
		shards = append(shards, map[string]any{"shardId": s.id, "createTime": s.created, "status": c.status})
	}
	return map[string]any{
		"adminUserName": c.adminUser, "authType": c.authType, "backupRetentionPeriod": c.backupRetention,
		"clusterArn": c.arn, "clusterEndpoint": c.name + "-" + dbeAccountID + "." + dbeRegion + ".docdb-elastic.amazonaws.com",
		"clusterName": c.name, "createTime": c.created, "kmsKeyId": c.kmsKeyID,
		"preferredBackupWindow": c.backupWindow, "preferredMaintenanceWindow": c.maintWindow,
		"shardCapacity": c.shardCapacity, "shardCount": c.shardCount, "shardInstanceCount": c.shardInstances,
		"shards": shards, "status": c.status,
		"subnetIds": dbeSorted(c.subnets), "vpcSecurityGroupIds": dbeSorted(c.groups),
	}
}

func (s *dbeSnapshot) out() map[string]any {
	return map[string]any{
		"adminUserName": s.adminUser, "clusterArn": s.clusterArn, "clusterCreationTime": s.clusterCreated,
		"kmsKeyId": s.kmsKeyID, "snapshotArn": s.arn, "snapshotCreationTime": s.created, "snapshotName": s.name,
		"snapshotType": s.kind, "status": s.status,
		"subnetIds": dbeSorted(s.subnets), "vpcSecurityGroupIds": dbeSorted(s.groups),
	}
}

func dbeSorted(in []string) []string {
	out := append([]string{}, in...)
	sort.Strings(out)
	return out
}
