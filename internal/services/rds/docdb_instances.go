package rds

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type DocDBInstanceInput struct {
	Identifier              string
	ClusterID               string
	Engine                  string
	DBInstanceClass         string
	AvailabilityZone        string
	PromotionTier           *int
	AutoMinorVersionUpgrade *bool
	MaintenanceWindow       string
	CACertificateIdentifier string
}

func (s *Service) checkInstanceSettingsLocked(in DocDBInstanceInput, cluster *DBCluster) error {
	if in.DBInstanceClass != "" && (!docdbClassPattern.MatchString(in.DBInstanceClass) || !containsString(DocDBInstanceClasses, in.DBInstanceClass)) {
		return fault(ErrInvalidParameter, "InvalidParameterValue", "DBInstanceClass %s is not supported", in.DBInstanceClass)
	}
	if in.PromotionTier != nil && (*in.PromotionTier < 0 || *in.PromotionTier > 15) {
		return fault(ErrInvalidParameter, "InvalidParameterValue", "PromotionTier must be between 0 and 15")
	}
	if in.AvailabilityZone != "" && cluster != nil && len(cluster.AvailabilityZones) > 0 && !containsString(cluster.AvailabilityZones, in.AvailabilityZone) {
		return fault(ErrInvalidParameter, "InvalidParameterCombination", "availability zone %s is not one of the cluster's", in.AvailabilityZone)
	}
	if in.MaintenanceWindow != "" && !docdbWindowPattern.MatchString(strings.ToLower(in.MaintenanceWindow)) {
		return fault(ErrInvalidParameter, "InvalidParameterValue", "invalid maintenance window %q", in.MaintenanceWindow)
	}
	if in.CACertificateIdentifier != "" {
		s.ensureCertificatesLocked()
		if _, ok := s.certificates[in.CACertificateIdentifier]; !ok {
			return fault(ErrNotFound, "CertificateNotFound", "Certificate %s not found", in.CACertificateIdentifier)
		}
	}
	return nil
}

func (s *Service) CreateDocDBInstance(in DocDBInstanceInput) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateDocDBIdentifier("DBInstanceIdentifier", in.Identifier, 63); err != nil {
		return DBInstance{}, err
	}
	if in.Engine != DocDBEngine {
		return DBInstance{}, fault(ErrInvalidParameter, "InvalidParameterValue", "Engine must be docdb")
	}
	if in.DBInstanceClass == "" {
		return DBInstance{}, fault(ErrInvalidParameter, "InvalidParameterValue", "DBInstanceClass is required")
	}
	if _, exists := s.instances[in.Identifier]; exists {
		return DBInstance{}, fault(ErrAlreadyExists, "DBInstanceAlreadyExists", "DB instance already exists")
	}
	if len(s.instances) >= docdbMaxInstances {
		return DBInstance{}, fault(ErrQuotaExceeded, "InstanceQuotaExceeded", "instance quota of %d reached", docdbMaxInstances)
	}
	cluster, err := s.docdbClusterLocked(in.ClusterID)
	if err != nil {
		return DBInstance{}, err
	}
	if cluster.Engine != DocDBEngine {
		return DBInstance{}, fault(ErrInvalidParameter, "InvalidParameterCombination", "cluster %s isn't a docdb cluster", cluster.Identifier)
	}
	if cluster.Status != "available" {
		return DBInstance{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "DbCluster %s is not available", cluster.Identifier)
	}
	if err := s.checkInstanceSettingsLocked(in, cluster); err != nil {
		return DBInstance{}, err
	}
	tier, autoMinor := 1, true
	if in.PromotionTier != nil {
		tier = *in.PromotionTier
	}
	if in.AutoMinorVersionUpgrade != nil {
		autoMinor = *in.AutoMinorVersionUpgrade
	}
	az := in.AvailabilityZone
	if az == "" {
		// A docdb cluster made through the rds API has no zone list.
		zones := cluster.AvailabilityZones
		if len(zones) == 0 {
			zones = docdbAZs
		}
		az = zones[len(cluster.Members)%len(zones)]
	}
	now := time.Now().UTC()
	inst := &DBInstance{
		Identifier: in.Identifier, ARN: dbInstanceARN(in.Identifier), Engine: DocDBEngine,
		DBInstanceClass: in.DBInstanceClass, MasterUsername: cluster.MasterUsername, Status: "available",
		EndpointAddress:            fmt.Sprintf("%s.stackyard.%s.docdb.amazonaws.com", in.Identifier, defaultRegion),
		Port:                       cluster.Port,
		BackupRetentionPeriod:      cluster.BackupRetentionPeriod,
		DBSubnetGroupName:          cluster.DBSubnetGroupName,
		DBClusterIdentifier:        cluster.Identifier,
		PromotionTier:              tier,
		AvailabilityZone:           az,
		EngineVersion:              cluster.EngineVersion,
		CACertificateIdentifier:    firstNonEmptyString(in.CACertificateIdentifier, "rds-ca-rsa2048-g1"),
		AutoMinorVersionUpgrade:    autoMinor,
		PreferredMaintenanceWindow: firstNonEmptyString(in.MaintenanceWindow, cluster.PreferredMaintenanceWindow),
		CreatedAt:                  now, UpdatedAt: now,
	}
	s.instances[inst.Identifier] = inst
	cluster.Members = append(cluster.Members, ClusterMember{InstanceID: inst.Identifier, IsWriter: !hasWriter(cluster.Members), PromotionTier: tier})
	return cloneDBInstance(inst), nil
}

func hasWriter(members []ClusterMember) bool {
	for _, m := range members {
		if m.IsWriter {
			return true
		}
	}
	return false
}

func (s *Service) docdbInstanceLocked(id string) (*DBInstance, error) {
	inst, ok := s.instances[arnOrName(id, ":db:")]
	if !ok {
		return nil, fault(ErrNotFound, "DBInstanceNotFound", "DBInstance %s not found.", id)
	}
	return inst, nil
}

// DocDBInstanceFilter selects instances for DescribeDocDBInstances.
type DocDBInstanceFilter struct {
	Identifier  string
	ClusterIDs  []string
	InstanceIDs []string
	Engines     []string
}

func (s *Service) DescribeDocDBInstances(f DocDBInstanceFilter, marker string, maxRecords int) ([]DBInstance, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.Identifier != "" {
		inst, err := s.docdbInstanceLocked(f.Identifier)
		if err != nil {
			return nil, "", err
		}
		return []DBInstance{cloneDBInstance(inst)}, "", nil
	}
	var items []*DBInstance
	for _, inst := range s.instances {
		if (len(f.Engines) == 0 || containsString(f.Engines, inst.Engine)) &&
			(len(f.InstanceIDs) == 0 || containsString(f.InstanceIDs, inst.Identifier) || containsString(f.InstanceIDs, inst.ARN)) &&
			(len(f.ClusterIDs) == 0 || containsString(f.ClusterIDs, inst.DBClusterIdentifier) || containsString(f.ClusterIDs, dbClusterARN(inst.DBClusterIdentifier))) {
			items = append(items, inst)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Identifier < items[j].Identifier })
	start, end, next, err := docdbPage(len(items), marker, maxRecords)
	if err != nil {
		return nil, "", err
	}
	out := make([]DBInstance, 0, end-start)
	for _, inst := range items[start:end] {
		out = append(out, cloneDBInstance(inst))
	}
	return out, next, nil
}

// ClusterMembers returns a copy of a cluster's members (nil if unknown).
func (s *Service) ClusterMembers(clusterID string) []ClusterMember {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.clusters[clusterID]; ok {
		return append([]ClusterMember(nil), c.Members...)
	}
	return nil
}

type DocDBModifyInstanceInput struct {
	DocDBInstanceInput
	NewIdentifier string
}

func (s *Service) ModifyDocDBInstance(in DocDBModifyInstanceInput) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inst, err := s.docdbInstanceLocked(in.Identifier)
	if err != nil {
		return DBInstance{}, err
	}
	if inst.Status != "available" {
		return DBInstance{}, fault(ErrInvalidState, "InvalidDBInstanceState", "DB instance %s is not available", inst.Identifier)
	}
	if err := s.checkInstanceSettingsLocked(in.DocDBInstanceInput, nil); err != nil {
		return DBInstance{}, err
	}
	newID := in.NewIdentifier
	if newID != "" && newID != inst.Identifier {
		if err := validateDocDBIdentifier("NewDBInstanceIdentifier", newID, 63); err != nil {
			return DBInstance{}, err
		}
		if _, exists := s.instances[newID]; exists {
			return DBInstance{}, fault(ErrAlreadyExists, "DBInstanceAlreadyExists", "DB instance %s already exists", newID)
		}
	}
	cluster := s.clusters[inst.DBClusterIdentifier]
	if in.DBInstanceClass != "" {
		inst.DBInstanceClass = in.DBInstanceClass
	}
	if in.PromotionTier != nil {
		inst.PromotionTier = *in.PromotionTier
	}
	if in.AutoMinorVersionUpgrade != nil {
		inst.AutoMinorVersionUpgrade = *in.AutoMinorVersionUpgrade
	}
	if in.MaintenanceWindow != "" {
		inst.PreferredMaintenanceWindow = in.MaintenanceWindow
	}
	if in.CACertificateIdentifier != "" {
		inst.CACertificateIdentifier = in.CACertificateIdentifier
	}
	if cluster != nil {
		for i := range cluster.Members {
			if cluster.Members[i].InstanceID == inst.Identifier {
				cluster.Members[i].PromotionTier = inst.PromotionTier
				if newID != "" {
					cluster.Members[i].InstanceID = newID
				}
			}
		}
	}
	if newID != "" && newID != inst.Identifier {
		oldARN := inst.ARN
		delete(s.instances, inst.Identifier)
		inst.Identifier, inst.ARN = newID, dbInstanceARN(newID)
		inst.EndpointAddress = fmt.Sprintf("%s.stackyard.%s.docdb.amazonaws.com", newID, defaultRegion)
		s.instances[newID] = inst
		if tags, ok := s.tags[oldARN]; ok {
			s.tags[inst.ARN] = tags
			delete(s.tags, oldARN)
		}
	}
	inst.UpdatedAt = time.Now().UTC()
	return cloneDBInstance(inst), nil
}

func (s *Service) RebootDocDBInstance(id string, forceFailover bool) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inst, err := s.docdbInstanceLocked(id)
	if err != nil {
		return DBInstance{}, err
	}
	if inst.Status != "available" {
		return DBInstance{}, fault(ErrInvalidState, "InvalidDBInstanceState", "DB instance %s is not available", inst.Identifier)
	}
	if cluster, ok := s.clusters[inst.DBClusterIdentifier]; ok && forceFailover {
		for i, m := range cluster.Members {
			if m.InstanceID == inst.Identifier && m.IsWriter {
				if best := bestReaderIndex(cluster.Members); best >= 0 {
					cluster.Members[i].IsWriter = false
					cluster.Members[best].IsWriter = true
				}
			}
		}
	}
	inst.UpdatedAt = time.Now().UTC()
	return cloneDBInstance(inst), nil
}

// DeleteDocDBInstance removes an instance; if it was the writer, the
// highest-priority reader takes over.
func (s *Service) DeleteDocDBInstance(id string) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inst, err := s.docdbInstanceLocked(id)
	if err != nil {
		return DBInstance{}, err
	}
	if inst.Status != "available" && inst.Status != "stopped" {
		return DBInstance{}, fault(ErrInvalidState, "InvalidDBInstanceState", "DB instance %s is %s", inst.Identifier, inst.Status)
	}
	if cluster, ok := s.clusters[inst.DBClusterIdentifier]; ok {
		if cluster.Status == "stopped" {
			return DBInstance{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "cluster %s is stopped", cluster.Identifier)
		}
		wasWriter := false
		kept := cluster.Members[:0]
		for _, m := range cluster.Members {
			if m.InstanceID == inst.Identifier {
				wasWriter = m.IsWriter
				continue
			}
			kept = append(kept, m)
		}
		cluster.Members = kept
		if wasWriter {
			if best := bestReaderIndex(cluster.Members); best >= 0 {
				cluster.Members[best].IsWriter = true
			}
		}
	}
	delete(s.instances, inst.Identifier)
	delete(s.tags, inst.ARN)
	delete(s.pendingMaint, inst.ARN)
	out := cloneDBInstance(inst)
	out.Status = "deleting"
	return out, nil
}
