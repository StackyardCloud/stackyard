package rds

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// DocDBClusterInput holds the CreateDBCluster / restore settings.
type DocDBClusterInput struct {
	Identifier                 string
	Engine                     string
	EngineVersion              string
	MasterUsername             string
	MasterUserPassword         string
	Port                       int
	DBSubnetGroupName          string
	DBClusterParameterGroup    string
	VpcSecurityGroupIDs        []string
	AvailabilityZones          []string
	BackupRetentionPeriod      int
	PreferredBackupWindow      string
	PreferredMaintenanceWindow string
	StorageEncrypted           bool
	KmsKeyID                   string
	DeletionProtection         bool
	LogExports                 []string
	StorageType                string
	GlobalClusterID            string
}

func validateDocDBClusterSettings(in *DocDBClusterInput) error {
	if in.EngineVersion == "" {
		in.EngineVersion = docdbDefaultVersion
	}
	if _, ok := DocDBFamilyForVersion(in.EngineVersion); !ok {
		return fault(ErrInvalidParameter, "InvalidParameterCombination", "Cannot find version %s for docdb", in.EngineVersion)
	}
	if in.Port == 0 {
		in.Port = docdbDefaultPort
	}
	if in.Port < 1150 || in.Port > 65535 {
		return fault(ErrInvalidParameter, "InvalidParameterValue", "Port must be between 1150 and 65535")
	}
	if in.BackupRetentionPeriod == 0 {
		in.BackupRetentionPeriod = 1
	}
	if in.BackupRetentionPeriod < 1 || in.BackupRetentionPeriod > 35 {
		return fault(ErrInvalidParameter, "InvalidParameterValue", "BackupRetentionPeriod must be between 1 and 35")
	}
	for _, w := range []string{in.PreferredBackupWindow, in.PreferredMaintenanceWindow} {
		if w != "" && !docdbWindowPattern.MatchString(strings.ToLower(w)) {
			return fault(ErrInvalidParameter, "InvalidParameterValue", "invalid time window %q", w)
		}
	}
	switch in.StorageType {
	case "":
		in.StorageType = "standard"
	case "standard", "iopt1":
	default:
		return fault(ErrInvalidParameter, "InvalidParameterValue", "StorageType must be standard or iopt1")
	}
	for _, log := range in.LogExports {
		if log != "audit" && log != "profiler" {
			return fault(ErrInvalidParameter, "InvalidParameterValue", "log type %q can't be exported; use audit or profiler", log)
		}
	}
	for _, az := range in.AvailabilityZones {
		if !containsString(docdbAZs, az) {
			return fault(ErrInvalidParameter, "InvalidParameterValue", "availability zone %q is not available", az)
		}
	}
	if in.KmsKeyID != "" && !in.StorageEncrypted {
		return fault(ErrInvalidParameter, "InvalidParameterCombination", "KmsKeyId requires StorageEncrypted")
	}
	return nil
}

func (s *Service) checkClusterRefsLocked(subnetGroup, paramGroup, family string) error {
	if subnetGroup != "" && subnetGroup != "default" {
		if _, ok := s.dbSubnetGroups[subnetGroup]; !ok {
			return fault(ErrNotFound, "DBSubnetGroupNotFoundFault", "DB subnet group %s not found", subnetGroup)
		}
	}
	if paramGroup != "" {
		group, ok := s.clusterParamGroups[paramGroup]
		if !ok {
			return fault(ErrNotFound, "DBClusterParameterGroupNotFound", "DBClusterParameterGroup %s not found", paramGroup)
		}
		if group.Family != family {
			return fault(ErrInvalidParameter, "InvalidParameterCombination", "parameter group family %s doesn't match engine family %s", group.Family, family)
		}
	}
	return nil
}

func (s *Service) newDocDBClusterLocked(in DocDBClusterInput, now time.Time) (*DBCluster, error) {
	if err := validateDocDBIdentifier("DBClusterIdentifier", in.Identifier, 63); err != nil {
		return nil, err
	}
	if in.Engine != DocDBEngine {
		return nil, fault(ErrInvalidParameter, "InvalidParameterValue", "Engine must be docdb")
	}
	if _, exists := s.clusters[in.Identifier]; exists {
		return nil, fault(ErrAlreadyExists, "DBClusterAlreadyExistsFault", "DB Cluster already exists")
	}
	if len(s.clusters) >= docdbMaxClusters {
		return nil, fault(ErrQuotaExceeded, "DBClusterQuotaExceededFault", "cluster quota of %d reached", docdbMaxClusters)
	}
	if err := validateDocDBClusterSettings(&in); err != nil {
		return nil, err
	}
	family, _ := DocDBFamilyForVersion(in.EngineVersion)
	s.ensureDefaultClusterParamGroupsLocked()
	if in.DBClusterParameterGroup == "" {
		in.DBClusterParameterGroup = "default." + family
	}
	if err := s.checkClusterRefsLocked(in.DBSubnetGroupName, in.DBClusterParameterGroup, family); err != nil {
		return nil, err
	}
	if in.DBSubnetGroupName == "" {
		in.DBSubnetGroupName = "default"
	}
	if len(in.AvailabilityZones) == 0 {
		in.AvailabilityZones = docdbAZs
	}
	if in.PreferredBackupWindow == "" {
		in.PreferredBackupWindow = "07:00-07:30"
	}
	if in.PreferredMaintenanceWindow == "" {
		in.PreferredMaintenanceWindow = "sun:08:00-sun:08:30"
	}
	if in.StorageEncrypted && in.KmsKeyID == "" {
		in.KmsKeyID = fmt.Sprintf("arn:aws:kms:%s:%s:key/aws-docdb-default", defaultRegion, defaultAccountID)
	}
	return &DBCluster{
		Identifier:                 in.Identifier,
		ARN:                        dbClusterARN(in.Identifier),
		Engine:                     DocDBEngine,
		EngineVersion:              in.EngineVersion,
		Status:                     "available",
		MasterUsername:             in.MasterUsername,
		DBSubnetGroupName:          in.DBSubnetGroupName,
		DBClusterParameterGroup:    in.DBClusterParameterGroup,
		Endpoint:                   fmt.Sprintf("%s.cluster-stackyard.%s.docdb.amazonaws.com", in.Identifier, defaultRegion),
		ReaderEndpoint:             fmt.Sprintf("%s.cluster-ro-stackyard.%s.docdb.amazonaws.com", in.Identifier, defaultRegion),
		Port:                       in.Port,
		BackupRetentionPeriod:      in.BackupRetentionPeriod,
		VpcSecurityGroupIDs:        compactStringSlice(in.VpcSecurityGroupIDs),
		AvailabilityZones:          append([]string(nil), in.AvailabilityZones...),
		PreferredBackupWindow:      in.PreferredBackupWindow,
		PreferredMaintenanceWindow: in.PreferredMaintenanceWindow,
		StorageEncrypted:           in.StorageEncrypted,
		KmsKeyID:                   in.KmsKeyID,
		DeletionProtection:         in.DeletionProtection,
		LogExports:                 compactStringSlice(in.LogExports),
		StorageType:                in.StorageType,
		ResourceID:                 docdbResourceID("cluster", in.Identifier),
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}, nil
}

// CreateDocDBCluster creates a cluster. A cluster joining a global cluster as
// a secondary needs no master credentials.
func (s *Service) CreateDocDBCluster(in DocDBClusterInput) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var global *GlobalCluster
	if in.GlobalClusterID != "" {
		var ok bool
		if global, ok = s.globalClusters[in.GlobalClusterID]; !ok {
			return DBCluster{}, fault(ErrNotFound, "GlobalClusterNotFoundFault", "Global cluster %s not found", in.GlobalClusterID)
		}
		if in.EngineVersion == "" {
			in.EngineVersion = global.EngineVersion
		}
		in.StorageEncrypted = in.StorageEncrypted || global.StorageEncrypted
	} else if err := validateMasterCredentials(in.MasterUsername, in.MasterUserPassword); err != nil {
		return DBCluster{}, err
	}
	now := time.Now().UTC()
	cluster, err := s.newDocDBClusterLocked(in, now)
	if err != nil {
		return DBCluster{}, err
	}
	if global != nil {
		cluster.GlobalClusterID = global.Identifier
		global.Members = appendUnique(global.Members, cluster.ARN)
		if global.SourceDBClusterArn == "" {
			global.SourceDBClusterArn = cluster.ARN
		}
	}
	s.clusters[cluster.Identifier] = cluster
	s.addAutomatedSnapshotLocked(cluster, now)
	return cloneDBCluster(cluster), nil
}

var docdbUsernamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]{0,62}$`)

func validateMasterCredentials(user, password string) error {
	if user == "" {
		return fault(ErrInvalidParameter, "InvalidParameterValue", "MasterUsername is required")
	}
	if !docdbUsernamePattern.MatchString(user) {
		return fault(ErrInvalidParameter, "InvalidParameterValue", "MasterUsername must be 1 to 63 letters or numbers and start with a letter")
	}
	if len(password) < 8 || len(password) > 100 || strings.ContainsAny(password, "/\"@") {
		return fault(ErrInvalidParameter, "InvalidParameterValue", "MasterUserPassword must be 8 to 100 printable characters, excluding /, \" and @")
	}
	return nil
}

func (s *Service) addAutomatedSnapshotLocked(cluster *DBCluster, now time.Time) {
	id := "rds:" + cluster.Identifier + "-" + now.Format("2006-01-02-15-04")
	if _, exists := s.clusterSnapshots[id]; exists {
		return
	}
	s.clusterSnapshots[id] = s.snapshotFromClusterLocked(cluster, id, "automated", now)
}

func (s *Service) snapshotFromClusterLocked(cluster *DBCluster, id, kind string, now time.Time) *DBClusterSnapshot {
	return &DBClusterSnapshot{
		Identifier: id, ARN: clusterSnapshotARN(id), ClusterID: cluster.Identifier,
		Engine: cluster.Engine, EngineVersion: cluster.EngineVersion, Status: "available", SnapshotType: kind,
		Port: cluster.Port, MasterUsername: cluster.MasterUsername, StorageEncrypted: cluster.StorageEncrypted,
		KmsKeyID: cluster.KmsKeyID, StorageType: cluster.StorageType, AZs: append([]string(nil), cluster.AvailabilityZones...),
		ClusterCreatedAt: cluster.CreatedAt, CreatedAt: now,
	}
}

func (s *Service) docdbClusterLocked(id string) (*DBCluster, error) {
	cluster, ok := s.clusters[arnOrName(id, ":cluster:")]
	if !ok {
		return nil, fault(ErrNotFound, "DBClusterNotFoundFault", "DBCluster %s not found.", id)
	}
	return cluster, nil
}

// DocDBClusterFilter selects clusters for DescribeDocDBClusters.
type DocDBClusterFilter struct {
	Identifier string
	ClusterIDs []string // db-cluster-id filter (ids or ARNs)
	Engines    []string // engine filter
}

func (s *Service) DescribeDocDBClusters(f DocDBClusterFilter, marker string, maxRecords int) ([]DBCluster, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.Identifier != "" {
		cluster, err := s.docdbClusterLocked(f.Identifier)
		if err != nil {
			return nil, "", err
		}
		return []DBCluster{cloneDBCluster(cluster)}, "", nil
	}
	var items []*DBCluster
	for _, c := range s.clusters {
		if (len(f.Engines) == 0 || containsString(f.Engines, c.Engine)) &&
			(len(f.ClusterIDs) == 0 || containsString(f.ClusterIDs, c.Identifier) || containsString(f.ClusterIDs, c.ARN)) {
			items = append(items, c)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Identifier < items[j].Identifier })
	start, end, next, err := docdbPage(len(items), marker, maxRecords)
	if err != nil {
		return nil, "", err
	}
	out := make([]DBCluster, 0, end-start)
	for _, c := range items[start:end] {
		out = append(out, cloneDBCluster(c))
	}
	return out, next, nil
}

// DocDBModifyClusterInput carries ModifyDBCluster settings; empty or nil
// fields are left unchanged.
type DocDBModifyClusterInput struct {
	Identifier                 string
	NewIdentifier              string
	BackupRetentionPeriod      int
	DBClusterParameterGroup    string
	VpcSecurityGroupIDs        []string
	Port                       int
	MasterUserPassword         string
	PreferredBackupWindow      string
	PreferredMaintenanceWindow string
	EnableLogs                 []string
	DisableLogs                []string
	EngineVersion              string
	AllowMajorVersionUpgrade   bool
	DeletionProtection         *bool
	StorageType                string
}

func (s *Service) ModifyDocDBCluster(in DocDBModifyClusterInput) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cluster, err := s.docdbClusterLocked(in.Identifier)
	if err != nil {
		return DBCluster{}, err
	}
	if cluster.Status != "available" {
		return DBCluster{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "DBCluster %s is not in the available state", cluster.Identifier)
	}
	// Validate everything first so a rejected call changes nothing.
	check := DocDBClusterInput{
		EngineVersion: firstNonEmptyString(in.EngineVersion, cluster.EngineVersion), Port: in.Port,
		BackupRetentionPeriod: in.BackupRetentionPeriod, PreferredBackupWindow: in.PreferredBackupWindow,
		PreferredMaintenanceWindow: in.PreferredMaintenanceWindow, StorageType: in.StorageType,
		LogExports: append(append([]string(nil), in.EnableLogs...), in.DisableLogs...),
	}
	if err := validateDocDBClusterSettings(&check); err != nil {
		return DBCluster{}, err
	}
	family, _ := DocDBFamilyForVersion(check.EngineVersion)
	if in.EngineVersion != "" && in.EngineVersion != cluster.EngineVersion {
		if in.EngineVersion < cluster.EngineVersion {
			return DBCluster{}, fault(ErrInvalidParameter, "InvalidParameterCombination", "cannot downgrade engine version from %s to %s", cluster.EngineVersion, in.EngineVersion)
		}
		if strings.SplitN(in.EngineVersion, ".", 2)[0] != strings.SplitN(cluster.EngineVersion, ".", 2)[0] && !in.AllowMajorVersionUpgrade {
			return DBCluster{}, fault(ErrInvalidParameter, "InvalidParameterCombination", "AllowMajorVersionUpgrade is required for a major version upgrade")
		}
	}
	paramGroup := cluster.DBClusterParameterGroup
	if in.DBClusterParameterGroup != "" {
		paramGroup = in.DBClusterParameterGroup
	} else if in.EngineVersion != "" && in.EngineVersion != cluster.EngineVersion {
		paramGroup = "default." + family
	}
	s.ensureDefaultClusterParamGroupsLocked()
	if err := s.checkClusterRefsLocked("", paramGroup, family); err != nil {
		return DBCluster{}, err
	}
	if in.MasterUserPassword != "" {
		if err := validateMasterCredentials(cluster.MasterUsername, in.MasterUserPassword); err != nil {
			return DBCluster{}, err
		}
	}
	newID := in.NewIdentifier
	if newID != "" && newID != cluster.Identifier {
		if err := validateDocDBIdentifier("NewDBClusterIdentifier", newID, 63); err != nil {
			return DBCluster{}, err
		}
		if _, exists := s.clusters[newID]; exists {
			return DBCluster{}, fault(ErrAlreadyExists, "DBClusterAlreadyExistsFault", "DB Cluster %s already exists", newID)
		}
	}

	if in.BackupRetentionPeriod > 0 {
		cluster.BackupRetentionPeriod = in.BackupRetentionPeriod
	}
	if in.Port > 0 {
		cluster.Port = in.Port
	}
	if in.VpcSecurityGroupIDs != nil {
		cluster.VpcSecurityGroupIDs = compactStringSlice(in.VpcSecurityGroupIDs)
	}
	cluster.DBClusterParameterGroup = paramGroup
	cluster.EngineVersion = check.EngineVersion
	if in.PreferredBackupWindow != "" {
		cluster.PreferredBackupWindow = in.PreferredBackupWindow
	}
	if in.PreferredMaintenanceWindow != "" {
		cluster.PreferredMaintenanceWindow = in.PreferredMaintenanceWindow
	}
	for _, log := range in.EnableLogs {
		cluster.LogExports = appendUnique(cluster.LogExports, log)
	}
	if len(in.DisableLogs) > 0 {
		kept := cluster.LogExports[:0]
		for _, log := range cluster.LogExports {
			if !containsString(in.DisableLogs, log) {
				kept = append(kept, log)
			}
		}
		cluster.LogExports = kept
	}
	if in.DeletionProtection != nil {
		cluster.DeletionProtection = *in.DeletionProtection
	}
	if in.StorageType != "" {
		cluster.StorageType = in.StorageType
	}
	for _, m := range cluster.Members {
		if inst, ok := s.instances[m.InstanceID]; ok {
			inst.EngineVersion = cluster.EngineVersion
		}
	}
	if newID != "" && newID != cluster.Identifier {
		s.renameClusterLocked(cluster, newID)
	}
	cluster.UpdatedAt = time.Now().UTC()
	return cloneDBCluster(cluster), nil
}

func (s *Service) renameClusterLocked(cluster *DBCluster, newID string) {
	oldARN := cluster.ARN
	delete(s.clusters, cluster.Identifier)
	cluster.Identifier, cluster.ARN = newID, dbClusterARN(newID)
	cluster.Endpoint = fmt.Sprintf("%s.cluster-stackyard.%s.docdb.amazonaws.com", newID, defaultRegion)
	cluster.ReaderEndpoint = fmt.Sprintf("%s.cluster-ro-stackyard.%s.docdb.amazonaws.com", newID, defaultRegion)
	s.clusters[newID] = cluster
	if tags, ok := s.tags[oldARN]; ok {
		s.tags[cluster.ARN] = tags
		delete(s.tags, oldARN)
	}
	for _, m := range cluster.Members {
		if inst, ok := s.instances[m.InstanceID]; ok {
			inst.DBClusterIdentifier = newID
		}
	}
	if g, ok := s.globalClusters[cluster.GlobalClusterID]; ok {
		for i, arn := range g.Members {
			if arn == oldARN {
				g.Members[i] = cluster.ARN
			}
		}
		if g.SourceDBClusterArn == oldARN {
			g.SourceDBClusterArn = cluster.ARN
		}
	}
}

// DeleteDocDBCluster deletes a cluster with no instances, optionally taking a
// final snapshot. Automated snapshots go with it.
func (s *Service) DeleteDocDBCluster(id string, skipFinal bool, finalSnapshotID string) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cluster, err := s.docdbClusterLocked(id)
	if err != nil {
		return DBCluster{}, err
	}
	switch {
	case cluster.DeletionProtection:
		return DBCluster{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "Cluster %s has deletion protection enabled", cluster.Identifier)
	case len(cluster.Members) > 0:
		return DBCluster{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "Cluster %s still has %d instance(s); delete them first", cluster.Identifier, len(cluster.Members))
	case cluster.GlobalClusterID != "":
		return DBCluster{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "Cluster %s is a member of global cluster %s; remove it first", cluster.Identifier, cluster.GlobalClusterID)
	case skipFinal && finalSnapshotID != "":
		return DBCluster{}, fault(ErrInvalidParameter, "InvalidParameterCombination", "FinalDBSnapshotIdentifier can't be used with SkipFinalSnapshot")
	case !skipFinal && finalSnapshotID == "":
		return DBCluster{}, fault(ErrInvalidParameter, "InvalidParameterCombination", "FinalDBSnapshotIdentifier is required unless SkipFinalSnapshot is true")
	}
	now := time.Now().UTC()
	if !skipFinal {
		if err := s.checkNewSnapshotIDLocked(finalSnapshotID); err != nil {
			return DBCluster{}, err
		}
		s.clusterSnapshots[finalSnapshotID] = s.snapshotFromClusterLocked(cluster, finalSnapshotID, "manual", now)
	}
	for snapID, snap := range s.clusterSnapshots {
		if snap.ClusterID == cluster.Identifier && snap.SnapshotType == "automated" {
			delete(s.clusterSnapshots, snapID)
		}
	}
	delete(s.clusters, cluster.Identifier)
	delete(s.tags, cluster.ARN)
	delete(s.pendingMaint, cluster.ARN)
	out := cloneDBCluster(cluster)
	out.Status = "deleting"
	return out, nil
}

func (s *Service) setClusterStateLocked(id, from, to string) (DBCluster, error) {
	cluster, err := s.docdbClusterLocked(id)
	if err != nil {
		return DBCluster{}, err
	}
	if cluster.Status != from {
		return DBCluster{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "DbCluster %s is in %s state but expected it to be %s.", cluster.Identifier, cluster.Status, from)
	}
	if cluster.GlobalClusterID != "" {
		return DBCluster{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "clusters in a global cluster can't be started or stopped")
	}
	cluster.Status = to
	for _, m := range cluster.Members {
		if inst, ok := s.instances[m.InstanceID]; ok {
			inst.Status = to
		}
	}
	cluster.UpdatedAt = time.Now().UTC()
	return cloneDBCluster(cluster), nil
}

func (s *Service) StartDocDBCluster(id string) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.setClusterStateLocked(id, "stopped", "available")
}

func (s *Service) StopDocDBCluster(id string) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.setClusterStateLocked(id, "available", "stopped")
}

// FailoverDocDBCluster promotes target (or the highest-priority reader) to
// writer.
func (s *Service) FailoverDocDBCluster(id, target string) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cluster, err := s.docdbClusterLocked(id)
	if err != nil {
		return DBCluster{}, err
	}
	if cluster.Status != "available" {
		return DBCluster{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "DbCluster %s is not available", cluster.Identifier)
	}
	if len(cluster.Members) == 0 {
		return DBCluster{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "DbCluster %s has no instances to fail over to", cluster.Identifier)
	}
	idx := -1
	if target != "" {
		for i, m := range cluster.Members {
			if m.InstanceID == target {
				idx = i
			}
		}
		if idx < 0 {
			return DBCluster{}, fault(ErrInvalidState, "InvalidDBInstanceState", "instance %s is not a member of cluster %s", target, cluster.Identifier)
		}
	} else {
		idx = bestReaderIndex(cluster.Members)
	}
	if idx >= 0 {
		for i := range cluster.Members {
			cluster.Members[i].IsWriter = i == idx
		}
	}
	cluster.UpdatedAt = time.Now().UTC()
	return cloneDBCluster(cluster), nil
}

// bestReaderIndex picks the reader with the lowest promotion tier, or -1.
func bestReaderIndex(members []ClusterMember) int {
	best := -1
	for i, m := range members {
		if !m.IsWriter && (best < 0 || m.PromotionTier < members[best].PromotionTier) {
			best = i
		}
	}
	return best
}
