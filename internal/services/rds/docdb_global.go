package rds

import (
	"sort"
	"strings"
	"time"
)

func (s *Service) docdbGlobalLocked(id string) (*GlobalCluster, error) {
	g, ok := s.globalClusters[strings.TrimSpace(id)]
	if !ok {
		return nil, fault(ErrNotFound, "GlobalClusterNotFoundFault", "Global cluster %s not found", id)
	}
	return g, nil
}

type DocDBGlobalInput struct {
	Identifier         string
	SourceClusterID    string
	EngineVersion      string
	DatabaseName       string
	StorageEncrypted   *bool
	DeletionProtection bool
}

// CreateDocDBGlobalCluster creates a global cluster, optionally around an
// existing primary cluster.
func (s *Service) CreateDocDBGlobalCluster(in DocDBGlobalInput) (GlobalCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateDocDBIdentifier("GlobalClusterIdentifier", in.Identifier, 255); err != nil {
		return GlobalCluster{}, err
	}
	if _, exists := s.globalClusters[in.Identifier]; exists {
		return GlobalCluster{}, fault(ErrAlreadyExists, "GlobalClusterAlreadyExistsFault", "Global cluster %s already exists", in.Identifier)
	}
	g := &GlobalCluster{Identifier: in.Identifier, ARN: globalClusterARN(in.Identifier), Status: "available", Engine: DocDBEngine,
		EngineVersion: in.EngineVersion, DatabaseName: in.DatabaseName, DeletionProtection: in.DeletionProtection}
	if in.StorageEncrypted != nil {
		g.StorageEncrypted = *in.StorageEncrypted
	}
	if in.SourceClusterID != "" {
		src, err := s.docdbClusterLocked(in.SourceClusterID)
		if err != nil {
			return GlobalCluster{}, err
		}
		if src.GlobalClusterID != "" {
			return GlobalCluster{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "cluster %s is already in global cluster %s", src.Identifier, src.GlobalClusterID)
		}
		if in.EngineVersion != "" && in.EngineVersion != src.EngineVersion {
			return GlobalCluster{}, fault(ErrInvalidParameter, "InvalidParameterCombination", "EngineVersion must match the source cluster")
		}
		g.EngineVersion, g.StorageEncrypted = src.EngineVersion, src.StorageEncrypted
		g.SourceDBClusterArn, g.Members = src.ARN, []string{src.ARN}
		src.GlobalClusterID = g.Identifier
	}
	if g.EngineVersion == "" {
		g.EngineVersion = docdbDefaultVersion
	}
	if _, ok := DocDBFamilyForVersion(g.EngineVersion); !ok {
		return GlobalCluster{}, fault(ErrInvalidParameter, "InvalidParameterCombination", "Cannot find version %s for docdb", g.EngineVersion)
	}
	g.CreatedAt, g.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	s.globalClusters[g.Identifier] = g
	return cloneGlobalCluster(g), nil
}

func (s *Service) DescribeDocDBGlobalClusters(id, marker string, maxRecords int) ([]GlobalCluster, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id != "" {
		g, err := s.docdbGlobalLocked(id)
		if err != nil {
			return nil, "", err
		}
		return []GlobalCluster{cloneGlobalCluster(g)}, "", nil
	}
	ids := make([]string, 0, len(s.globalClusters))
	for gid := range s.globalClusters {
		ids = append(ids, gid)
	}
	sort.Strings(ids)
	start, end, next, err := docdbPage(len(ids), marker, maxRecords)
	if err != nil {
		return nil, "", err
	}
	out := make([]GlobalCluster, 0, end-start)
	for _, gid := range ids[start:end] {
		out = append(out, cloneGlobalCluster(s.globalClusters[gid]))
	}
	return out, next, nil
}

func (s *Service) ModifyDocDBGlobalCluster(id, newID string, deletionProtection *bool) (GlobalCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.docdbGlobalLocked(id)
	if err != nil {
		return GlobalCluster{}, err
	}
	if newID != "" && newID != g.Identifier {
		if err := validateDocDBIdentifier("NewGlobalClusterIdentifier", newID, 255); err != nil {
			return GlobalCluster{}, err
		}
		if _, exists := s.globalClusters[newID]; exists {
			return GlobalCluster{}, fault(ErrAlreadyExists, "GlobalClusterAlreadyExistsFault", "Global cluster %s already exists", newID)
		}
		delete(s.globalClusters, g.Identifier)
		for _, c := range s.clusters {
			if c.GlobalClusterID == g.Identifier {
				c.GlobalClusterID = newID
			}
		}
		g.Identifier, g.ARN = newID, globalClusterARN(newID)
		s.globalClusters[newID] = g
	}
	if deletionProtection != nil {
		g.DeletionProtection = *deletionProtection
	}
	g.UpdatedAt = time.Now().UTC()
	return cloneGlobalCluster(g), nil
}

func (s *Service) DeleteDocDBGlobalCluster(id string) (GlobalCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.docdbGlobalLocked(id)
	if err != nil {
		return GlobalCluster{}, err
	}
	if g.DeletionProtection {
		return GlobalCluster{}, fault(ErrInvalidState, "InvalidGlobalClusterStateFault", "global cluster %s has deletion protection enabled", g.Identifier)
	}
	if len(g.Members) > 0 {
		return GlobalCluster{}, fault(ErrInvalidState, "InvalidGlobalClusterStateFault", "global cluster %s still has member clusters", g.Identifier)
	}
	delete(s.globalClusters, g.Identifier)
	out := cloneGlobalCluster(g)
	out.Status = "deleting"
	return out, nil
}

func (s *Service) globalMemberLocked(g *GlobalCluster, clusterRef string) (*DBCluster, error) {
	cluster, err := s.docdbClusterLocked(clusterRef)
	if err != nil {
		return nil, err
	}
	if !containsString(g.Members, cluster.ARN) {
		return nil, fault(ErrNotFound, "DBClusterNotFoundFault", "cluster %s is not a member of global cluster %s", cluster.Identifier, g.Identifier)
	}
	return cluster, nil
}

// RemoveFromDocDBGlobalCluster detaches a secondary cluster. The primary can
// only leave once it's the last member.
func (s *Service) RemoveFromDocDBGlobalCluster(id, clusterRef string) (GlobalCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.docdbGlobalLocked(id)
	if err != nil {
		return GlobalCluster{}, err
	}
	cluster, err := s.globalMemberLocked(g, clusterRef)
	if err != nil {
		return GlobalCluster{}, err
	}
	if cluster.ARN == g.SourceDBClusterArn && len(g.Members) > 1 {
		return GlobalCluster{}, fault(ErrInvalidState, "InvalidGlobalClusterStateFault", "the primary cluster can't be removed while secondaries remain")
	}
	kept := g.Members[:0]
	for _, arn := range g.Members {
		if arn != cluster.ARN {
			kept = append(kept, arn)
		}
	}
	g.Members = kept
	if g.SourceDBClusterArn == cluster.ARN {
		g.SourceDBClusterArn = ""
	}
	cluster.GlobalClusterID = ""
	g.UpdatedAt = time.Now().UTC()
	return cloneGlobalCluster(g), nil
}

// FailoverDocDBGlobalCluster makes target the primary. A switchover requires
// matching engine versions; a failover doesn't.
func (s *Service) FailoverDocDBGlobalCluster(id, target string, switchover bool) (GlobalCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.docdbGlobalLocked(id)
	if err != nil {
		return GlobalCluster{}, err
	}
	cluster, err := s.globalMemberLocked(g, target)
	if err != nil {
		return GlobalCluster{}, err
	}
	if cluster.ARN == g.SourceDBClusterArn {
		return GlobalCluster{}, fault(ErrInvalidState, "InvalidGlobalClusterStateFault", "cluster %s is already the primary", cluster.Identifier)
	}
	if switchover && cluster.EngineVersion != g.EngineVersion {
		return GlobalCluster{}, fault(ErrInvalidState, "InvalidDBClusterStateFault", "switchover needs matching engine versions")
	}
	g.SourceDBClusterArn = cluster.ARN
	g.UpdatedAt = time.Now().UTC()
	return cloneGlobalCluster(g), nil
}

// docdbSourceExistsLocked reports whether sourceID names a resource of
// sourceType ("" means any).
func (s *Service) docdbSourceExistsLocked(sourceType, sourceID string) bool {
	_, inst := s.instances[sourceID]
	_, cluster := s.clusters[sourceID]
	_, snap := s.clusterSnapshots[sourceID]
	_, pg := s.clusterParamGroups[sourceID]
	switch sourceType {
	case "db-instance":
		return inst
	case "db-cluster":
		return cluster
	case "db-cluster-snapshot":
		return snap
	case "db-parameter-group":
		return pg
	default:
		return inst || cluster || snap || pg
	}
}

// UpdateSubscriptionSource adds or removes a source identifier.
func (s *Service) UpdateSubscriptionSource(name, sourceID string, add bool) (EventSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.eventSubs[strings.TrimSpace(name)]
	if !ok {
		return EventSubscription{}, fault(ErrNotFound, "SubscriptionNotFound", "Subscription %s not found", name)
	}
	if add {
		s.ensureDefaultClusterParamGroupsLocked()
		if !s.docdbSourceExistsLocked(sub.SourceType, sourceID) {
			return EventSubscription{}, fault(ErrNotFound, "SourceNotFound", "Source %s not found", sourceID)
		}
		sub.SourceIDs = appendUnique(sub.SourceIDs, sourceID)
	} else {
		if !containsString(sub.SourceIDs, sourceID) {
			return EventSubscription{}, fault(ErrNotFound, "SourceNotFound", "Source %s is not in subscription %s", sourceID, name)
		}
		kept := sub.SourceIDs[:0]
		for _, id := range sub.SourceIDs {
			if id != sourceID {
				kept = append(kept, id)
			}
		}
		sub.SourceIDs = kept
	}
	sub.UpdatedAt = time.Now().UTC()
	out := *sub
	out.SourceIDs = append([]string(nil), sub.SourceIDs...)
	out.EventCategories = append([]string(nil), sub.EventCategories...)
	return out, nil
}

// docdbTaggableLocked checks that resourceARN names an existing resource and
// returns the not-found fault to report otherwise.
func (s *Service) docdbTaggableLocked(resourceARN string) error {
	parts := strings.Split(resourceARN, ":")
	if len(parts) < 7 || parts[0] != "arn" || parts[2] != "rds" {
		return fault(ErrInvalidParameter, "InvalidParameterValue", "%q is not a valid resource ARN", resourceARN)
	}
	id := strings.Join(parts[6:], ":")
	s.ensureDefaultClusterParamGroupsLocked()
	var ok bool
	code := "DBClusterNotFoundFault"
	switch parts[5] {
	case "db":
		_, ok = s.instances[id]
		code = "DBInstanceNotFound"
	case "cluster":
		_, ok = s.clusters[id]
	case "cluster-snapshot":
		_, ok = s.clusterSnapshots[id]
		code = "DBSnapshotNotFound"
	case "cluster-pg":
		_, ok = s.clusterParamGroups[id]
		code = "DBParameterGroupNotFound"
	case "subgrp":
		_, ok = s.dbSubnetGroups[id]
		code = "DBSubnetGroupNotFoundFault"
	case "es":
		_, ok = s.eventSubs[id]
		code = "SubscriptionNotFound"
	default:
		return fault(ErrInvalidParameter, "InvalidParameterValue", "resource type %q can't be tagged", parts[5])
	}
	if !ok {
		return fault(ErrNotFound, code, "resource %s not found", resourceARN)
	}
	return nil
}

func (s *Service) DocDBTags(resourceARN string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.docdbTaggableLocked(resourceARN); err != nil {
		return nil, err
	}
	return cloneTagMap(s.tags[resourceARN]), nil
}

func (s *Service) DocDBTagResource(resourceARN string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.docdbTaggableLocked(resourceARN); err != nil {
		return err
	}
	if err := ValidateDocDBTags(tags); err != nil {
		return err
	}
	store := cloneTagMap(s.tags[resourceARN])
	for k, v := range tags {
		store[k] = v
	}
	if len(store) > docdbMaxTagsPerARN {
		return fault(ErrInvalidParameter, "InvalidParameterValue", "a resource can have at most %d tags", docdbMaxTagsPerARN)
	}
	s.tags[resourceARN] = store
	return nil
}

func (s *Service) DocDBUntagResource(resourceARN string, keys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.docdbTaggableLocked(resourceARN); err != nil {
		return err
	}
	for _, k := range keys {
		delete(s.tags[resourceARN], k)
	}
	return nil
}

// ValidateDocDBTags checks tag keys and values against the API's limits.
func ValidateDocDBTags(tags map[string]string) error {
	if len(tags) > docdbMaxTagsPerARN {
		return fault(ErrInvalidParameter, "InvalidParameterValue", "a resource can have at most %d tags", docdbMaxTagsPerARN)
	}
	for k, v := range tags {
		if k == "" || len(k) > 128 || len(v) > 256 || strings.HasPrefix(strings.ToLower(k), "aws:") {
			return fault(ErrInvalidParameter, "InvalidParameterValue", "invalid tag key %q", k)
		}
	}
	return nil
}
