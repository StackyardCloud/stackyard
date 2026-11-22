package rds

import (
	"regexp"
	"sort"
	"time"
)

func (s *Service) checkNewSnapshotIDLocked(id string) error {
	if err := validateDocDBIdentifier("DBClusterSnapshotIdentifier", id, 255); err != nil {
		return err
	}
	if _, exists := s.clusterSnapshots[id]; exists {
		return fault(ErrAlreadyExists, "DBClusterSnapshotAlreadyExistsFault", "Cluster snapshot %s already exists", id)
	}
	if len(s.clusterSnapshots) >= docdbMaxSnapshots {
		return fault(ErrQuotaExceeded, "SnapshotQuotaExceeded", "snapshot quota of %d reached", docdbMaxSnapshots)
	}
	return nil
}

func (s *Service) clusterSnapshotLocked(id string) (*DBClusterSnapshot, error) {
	snap, ok := s.clusterSnapshots[arnOrName(id, ":cluster-snapshot:")]
	if !ok {
		return nil, fault(ErrNotFound, "DBClusterSnapshotNotFoundFault", "DBClusterSnapshot %s not found.", id)
	}
	return snap, nil
}

func (s *Service) CreateClusterSnapshot(snapshotID, clusterID string) (DBClusterSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cluster, err := s.docdbClusterLocked(clusterID)
	if err != nil {
		return DBClusterSnapshot{}, err
	}
	if cluster.Status != "available" {
		return DBClusterSnapshot{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "DbCluster %s is not available", cluster.Identifier)
	}
	if err := s.checkNewSnapshotIDLocked(snapshotID); err != nil {
		return DBClusterSnapshot{}, err
	}
	snap := s.snapshotFromClusterLocked(cluster, snapshotID, "manual", time.Now().UTC())
	s.clusterSnapshots[snapshotID] = snap
	return cloneClusterSnapshot(snap), nil
}

func (s *Service) CopyClusterSnapshot(source, target, kmsKeyID string) (DBClusterSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, err := s.clusterSnapshotLocked(source)
	if err != nil {
		return DBClusterSnapshot{}, err
	}
	if src.Status != "available" {
		return DBClusterSnapshot{}, fault(ErrInvalidState, "InvalidDBClusterSnapshotStateFault", "snapshot %s is not available", src.Identifier)
	}
	if err := s.checkNewSnapshotIDLocked(target); err != nil {
		return DBClusterSnapshot{}, err
	}
	snap := cloneClusterSnapshot(src)
	snap.Identifier, snap.ARN, snap.SnapshotType, snap.SourceARN = target, clusterSnapshotARN(target), "manual", src.ARN
	snap.CreatedAt, snap.RestoreAccounts = time.Now().UTC(), nil
	if kmsKeyID != "" {
		snap.StorageEncrypted, snap.KmsKeyID = true, kmsKeyID
	}
	s.clusterSnapshots[target] = &snap
	return cloneClusterSnapshot(&snap), nil
}

func (s *Service) DeleteClusterSnapshot(id string) (DBClusterSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, err := s.clusterSnapshotLocked(id)
	if err != nil {
		return DBClusterSnapshot{}, err
	}
	if snap.SnapshotType == "automated" {
		return DBClusterSnapshot{}, fault(ErrInvalidState, "InvalidDBClusterSnapshotStateFault", "automated snapshots can't be deleted")
	}
	delete(s.clusterSnapshots, snap.Identifier)
	delete(s.tags, snap.ARN)
	out := cloneClusterSnapshot(snap)
	out.Status = "deleted"
	return out, nil
}

// ClusterSnapshotFilter selects snapshots for DescribeClusterSnapshots.
type ClusterSnapshotFilter struct {
	Identifier    string
	ClusterID     string
	SnapshotType  string // automated, manual, shared, public or empty
	IncludeShared bool
	IncludePublic bool
}

func (s *Service) DescribeClusterSnapshots(f ClusterSnapshotFilter, marker string, maxRecords int) ([]DBClusterSnapshot, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch f.SnapshotType {
	case "", "automated", "manual", "shared", "public":
	default:
		return nil, "", fault(ErrInvalidParameter, "InvalidParameterValue", "SnapshotType must be automated, manual, shared or public")
	}
	if f.Identifier != "" {
		snap, err := s.clusterSnapshotLocked(f.Identifier)
		if err != nil {
			return nil, "", err
		}
		return []DBClusterSnapshot{cloneClusterSnapshot(snap)}, "", nil
	}
	clusterID := arnOrName(f.ClusterID, ":cluster:")
	var items []*DBClusterSnapshot
	for _, snap := range s.clusterSnapshots {
		if clusterID != "" && snap.ClusterID != clusterID {
			continue
		}
		// Every snapshot here is owned by this account, so "shared" and
		// "public" select nothing.
		if f.SnapshotType == "shared" || f.SnapshotType == "public" || (f.SnapshotType != "" && snap.SnapshotType != f.SnapshotType) {
			continue
		}
		items = append(items, snap)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Identifier < items[j].Identifier })
	start, end, next, err := docdbPage(len(items), marker, maxRecords)
	if err != nil {
		return nil, "", err
	}
	out := make([]DBClusterSnapshot, 0, end-start)
	for _, snap := range items[start:end] {
		out = append(out, cloneClusterSnapshot(snap))
	}
	return out, next, nil
}

func (s *Service) ClusterSnapshotAttributes(id string) (DBClusterSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, err := s.clusterSnapshotLocked(id)
	if err != nil {
		return DBClusterSnapshot{}, err
	}
	return cloneClusterSnapshot(snap), nil
}

var docdbAccountPattern = regexp.MustCompile(`^(?:\d{12}|all)$`)

// ModifyClusterSnapshotAttribute shares a manual snapshot ("restore"
// attribute) with account ids, or with "all" to make it public.
func (s *Service) ModifyClusterSnapshotAttribute(id, name string, add, remove []string) (DBClusterSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, err := s.clusterSnapshotLocked(id)
	if err != nil {
		return DBClusterSnapshot{}, err
	}
	if name != "restore" {
		return DBClusterSnapshot{}, fault(ErrInvalidParameter, "InvalidParameterValue", "AttributeName must be restore")
	}
	if snap.SnapshotType != "manual" {
		return DBClusterSnapshot{}, fault(ErrInvalidState, "InvalidDBClusterSnapshotStateFault", "only manual snapshots can be shared")
	}
	for _, acct := range append(append([]string(nil), add...), remove...) {
		if !docdbAccountPattern.MatchString(acct) {
			return DBClusterSnapshot{}, fault(ErrInvalidParameter, "InvalidParameterValue", "%q is not an AWS account id or all", acct)
		}
	}
	if containsString(add, "all") && snap.StorageEncrypted {
		return DBClusterSnapshot{}, fault(ErrInvalidParameter, "InvalidParameterValue", "encrypted snapshots can't be made public")
	}
	accounts := append([]string(nil), snap.RestoreAccounts...)
	for _, acct := range add {
		accounts = appendUnique(accounts, acct)
	}
	kept := accounts[:0]
	for _, acct := range accounts {
		if !containsString(remove, acct) {
			kept = append(kept, acct)
		}
	}
	if len(kept) > docdbMaxSharedAccts {
		return DBClusterSnapshot{}, fault(ErrQuotaExceeded, "SharedSnapshotQuotaExceeded", "a snapshot can be shared with at most %d accounts", docdbMaxSharedAccts)
	}
	snap.RestoreAccounts = kept
	return cloneClusterSnapshot(snap), nil
}

// RestoreClusterFromSnapshot creates a cluster from a cluster snapshot.
func (s *Service) RestoreClusterFromSnapshot(snapshotID string, in DocDBClusterInput) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, err := s.clusterSnapshotLocked(snapshotID)
	if err != nil {
		return DBCluster{}, err
	}
	if snap.Status != "available" {
		return DBCluster{}, fault(ErrInvalidState, "InvalidDBClusterSnapshotStateFault", "snapshot %s is not available", snap.Identifier)
	}
	if in.Engine != snap.Engine {
		return DBCluster{}, fault(ErrInvalidParameter, "InvalidParameterCombination", "Engine %s doesn't match the snapshot's engine %s", in.Engine, snap.Engine)
	}
	in.EngineVersion = firstNonEmptyString(in.EngineVersion, snap.EngineVersion)
	in.MasterUsername = snap.MasterUsername
	if in.Port == 0 {
		in.Port = snap.Port
	}
	in.StorageEncrypted = snap.StorageEncrypted || in.KmsKeyID != ""
	in.KmsKeyID = firstNonEmptyString(in.KmsKeyID, snap.KmsKeyID)
	in.StorageType = firstNonEmptyString(in.StorageType, snap.StorageType)
	now := time.Now().UTC()
	cluster, err := s.newDocDBClusterLocked(in, now)
	if err != nil {
		return DBCluster{}, err
	}
	s.clusters[cluster.Identifier] = cluster
	s.addAutomatedSnapshotLocked(cluster, now)
	return cloneDBCluster(cluster), nil
}

// RestoreClusterToPointInTime clones a cluster as of restoreTime (or the
// latest restorable time).
func (s *Service) RestoreClusterToPointInTime(sourceID string, restoreTime time.Time, useLatest bool, restoreType string, in DocDBClusterInput) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, err := s.docdbClusterLocked(sourceID)
	if err != nil {
		return DBCluster{}, err
	}
	now := time.Now().UTC()
	switch {
	case useLatest == !restoreTime.IsZero():
		return DBCluster{}, fault(ErrInvalidParameter, "InvalidParameterCombination", "specify exactly one of RestoreToTime and UseLatestRestorableTime")
	case restoreType != "" && restoreType != "full-copy" && restoreType != "copy-on-write":
		return DBCluster{}, fault(ErrInvalidParameter, "InvalidParameterValue", "RestoreType must be full-copy or copy-on-write")
	case !useLatest && (restoreTime.Before(src.CreatedAt) || restoreTime.After(now)):
		return DBCluster{}, fault(ErrInvalidParameter, "InvalidRestoreFault", "RestoreToTime must be between the cluster's earliest and latest restorable times")
	}
	in.Engine, in.EngineVersion, in.MasterUsername = src.Engine, src.EngineVersion, src.MasterUsername
	if in.Port == 0 {
		in.Port = src.Port
	}
	in.StorageEncrypted = src.StorageEncrypted || in.KmsKeyID != ""
	in.KmsKeyID = firstNonEmptyString(in.KmsKeyID, src.KmsKeyID)
	in.StorageType = firstNonEmptyString(in.StorageType, src.StorageType)
	in.DBClusterParameterGroup = firstNonEmptyString(in.DBClusterParameterGroup, src.DBClusterParameterGroup)
	cluster, err := s.newDocDBClusterLocked(in, now)
	if err != nil {
		return DBCluster{}, err
	}
	s.clusters[cluster.Identifier] = cluster
	s.addAutomatedSnapshotLocked(cluster, now)
	return cloneDBCluster(cluster), nil
}
