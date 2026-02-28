package cloudhsm

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrNotFound         = errors.New("resource not found")
)

const (
	defaultRegion       = "us-east-1"
	defaultAccountID    = "123456789012"
	defaultMaxResults   = int32(100)
	maxResultsUpper     = int32(100)
	defaultHSMType      = "hsm1.medium"
	defaultNetworkType  = "IPV4"
	defaultMode         = "FIPS"
	defaultBackupPolicy = "DEFAULT"
)

type Tag struct {
	Key   string
	Value string
}

type BackupRetentionPolicy struct {
	Type  string
	Value string
}

type Certificates struct {
	ClusterCsr                      string
	HsmCertificate                  string
	AwsHardwareCertificate          string
	ManufacturerHardwareCertificate string
	ClusterCertificate              string
}

type Hsm struct {
	AvailabilityZone string
	ClusterID        string
	SubnetID         string
	EniID            string
	EniIP            string
	EniIPv6          string
	HsmID            string
	State            string
	StateMessage     string
}

type Cluster struct {
	BackupPolicy          string
	BackupRetentionPolicy BackupRetentionPolicy
	ClusterID             string
	CreateTimestamp       time.Time
	Hsms                  []Hsm
	HsmType               string
	PreCoPassword         string
	SecurityGroup         string
	SourceBackupID        string
	State                 string
	StateMessage          string
	SubnetMapping         map[string]string
	VpcID                 string
	NetworkType           string
	Certificates          Certificates
	TagList               []Tag
	Mode                  string
}

type Backup struct {
	BackupID        string
	BackupARN       string
	BackupState     string
	ClusterID       string
	CreateTimestamp time.Time
	CopyTimestamp   time.Time
	NeverExpires    bool
	SourceRegion    string
	SourceBackup    string
	SourceCluster   string
	DeleteTimestamp time.Time
	TagList         []Tag
	HsmType         string
	Mode            string
}

type Service struct {
	mu               sync.Mutex
	seq              uint64
	clusters         map[string]*Cluster
	backups          map[string]*Backup
	resourcePolicies map[string]string
	resourceTags     map[string]map[string]string
}

func NewService() *Service {
	return &Service{
		clusters:         map[string]*Cluster{},
		backups:          map[string]*Backup{},
		resourcePolicies: map[string]string{},
		resourceTags:     map[string]map[string]string{},
	}
}

func (s *Service) CreateCluster(
	backupRetentionPolicy BackupRetentionPolicy,
	hsmType string,
	sourceBackupID string,
	subnetIDs []string,
	networkType string,
	tagList []Tag,
	mode string,
) (Cluster, error) {
	hsmType = strings.TrimSpace(hsmType)
	if hsmType == "" {
		return Cluster{}, ErrInvalidParameter
	}
	subnets := normalizeNonEmptyStrings(subnetIDs)
	if len(subnets) == 0 {
		return Cluster{}, ErrInvalidParameter
	}
	if networkType = strings.ToUpper(strings.TrimSpace(networkType)); networkType == "" {
		networkType = defaultNetworkType
	}
	if mode = strings.ToUpper(strings.TrimSpace(mode)); mode == "" {
		mode = defaultMode
	}
	retention := normalizeBackupRetentionPolicy(backupRetentionPolicy)

	sourceBackupID = strings.TrimSpace(sourceBackupID)

	s.mu.Lock()
	defer s.mu.Unlock()

	if sourceBackupID != "" {
		if _, ok := s.backups[sourceBackupID]; !ok {
			return Cluster{}, ErrNotFound
		}
	}

	now := time.Now().UTC()
	clusterID := s.nextIDLocked("cluster")
	cluster := &Cluster{
		BackupPolicy:          defaultBackupPolicy,
		BackupRetentionPolicy: retention,
		ClusterID:             clusterID,
		CreateTimestamp:       now,
		Hsms:                  []Hsm{},
		HsmType:               hsmType,
		PreCoPassword:         "preco-" + clusterID,
		SecurityGroup:         "sg-00000001",
		SourceBackupID:        sourceBackupID,
		State:                 "UNINITIALIZED",
		StateMessage:          "",
		SubnetMapping:         map[string]string{},
		VpcID:                 "vpc-00000001",
		NetworkType:           networkType,
		Certificates: Certificates{
			ClusterCsr: "cluster-csr-" + clusterID,
		},
		TagList: normalizeTags(tagList),
		Mode:    mode,
	}
	for i, subnet := range subnets {
		cluster.SubnetMapping[subnet] = fmt.Sprintf("%s%c", "us-east-1", 'a'+(i%3))
	}
	s.clusters[clusterID] = cluster

	clusterARN := clusterARN(clusterID)
	s.resourceTags[clusterARN] = tagsToMap(cluster.TagList)
	s.resourceTags[clusterID] = cloneStringMap(s.resourceTags[clusterARN])

	backup := s.newBackupLocked(cluster, sourceBackupID, "", "", cluster.TagList)
	s.backups[backup.BackupID] = backup
	s.resourceTags[backup.BackupARN] = tagsToMap(backup.TagList)
	s.resourceTags[backup.BackupID] = cloneStringMap(s.resourceTags[backup.BackupARN])

	return cloneCluster(*cluster), nil
}

func (s *Service) DescribeClusters(filters map[string][]string, nextToken string, maxResults int32) ([]Cluster, string, error) {
	start, limit, err := normalizeListInput(nextToken, maxResults)
	if err != nil {
		return nil, "", err
	}

	filtered := normalizeFilters(filters)

	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.clusters))
	for id := range s.clusters {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	matched := make([]Cluster, 0, len(ids))
	for _, id := range ids {
		cluster := s.clusters[id]
		if !clusterMatchesFilters(cluster, filtered) {
			continue
		}
		matched = append(matched, cloneCluster(*cluster))
	}

	return paginateClusters(matched, start, limit), nextPageToken(len(matched), start, limit), nil
}

func (s *Service) InitializeCluster(clusterID, signedCert, trustAnchor string) (string, string, error) {
	clusterID = strings.TrimSpace(clusterID)
	if clusterID == "" || strings.TrimSpace(signedCert) == "" || strings.TrimSpace(trustAnchor) == "" {
		return "", "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.clusters[clusterID]
	if !ok {
		return "", "", ErrNotFound
	}

	cluster.State = "INITIALIZED"
	cluster.StateMessage = "Cluster initialized"
	cluster.Certificates.ClusterCertificate = "cluster-cert-" + clusterID
	cluster.Certificates.HsmCertificate = "hsm-cert-" + clusterID
	cluster.Certificates.AwsHardwareCertificate = "aws-hardware-cert"
	cluster.Certificates.ManufacturerHardwareCertificate = "manufacturer-hardware-cert"

	return cluster.State, cluster.StateMessage, nil
}

func (s *Service) DeleteCluster(clusterID string) (Cluster, error) {
	clusterID = strings.TrimSpace(clusterID)
	if clusterID == "" {
		return Cluster{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.clusters[clusterID]
	if !ok {
		return Cluster{}, ErrNotFound
	}

	out := cloneCluster(*cluster)
	out.State = "DELETED"
	out.StateMessage = "Cluster deleted"
	delete(s.clusters, clusterID)
	delete(s.resourceTags, clusterID)
	delete(s.resourceTags, clusterARN(clusterID))
	delete(s.resourcePolicies, clusterARN(clusterID))
	return out, nil
}

func (s *Service) CreateHsm(clusterID, availabilityZone, ipAddress string) (Hsm, error) {
	clusterID = strings.TrimSpace(clusterID)
	availabilityZone = strings.TrimSpace(availabilityZone)
	if clusterID == "" || availabilityZone == "" {
		return Hsm{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.clusters[clusterID]
	if !ok {
		return Hsm{}, ErrNotFound
	}

	if ipAddress = strings.TrimSpace(ipAddress); ipAddress == "" {
		ipAddress = fmt.Sprintf("10.0.0.%d", (len(cluster.Hsms)%200)+10)
	}

	subnetID := ""
	for key := range cluster.SubnetMapping {
		subnetID = key
		break
	}
	hsm := Hsm{
		AvailabilityZone: availabilityZone,
		ClusterID:        clusterID,
		SubnetID:         subnetID,
		EniID:            s.nextIDLocked("eni"),
		EniIP:            ipAddress,
		EniIPv6:          "",
		HsmID:            s.nextIDLocked("hsm"),
		State:            "ACTIVE",
		StateMessage:     "HSM active",
	}
	cluster.Hsms = append(cluster.Hsms, hsm)
	cluster.State = "ACTIVE"
	cluster.StateMessage = "Cluster active"
	return hsm, nil
}

func (s *Service) DeleteHsm(clusterID, hsmID, eniID, eniIP string) (string, error) {
	clusterID = strings.TrimSpace(clusterID)
	if clusterID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.clusters[clusterID]
	if !ok {
		return "", ErrNotFound
	}

	hsmID = strings.TrimSpace(hsmID)
	eniID = strings.TrimSpace(eniID)
	eniIP = strings.TrimSpace(eniIP)

	index := -1
	for i, hsm := range cluster.Hsms {
		if hsmID != "" && hsm.HsmID == hsmID {
			index = i
			break
		}
		if eniID != "" && hsm.EniID == eniID {
			index = i
			break
		}
		if eniIP != "" && hsm.EniIP == eniIP {
			index = i
			break
		}
	}
	if index < 0 {
		if hsmID == "" && eniID == "" && eniIP == "" && len(cluster.Hsms) > 0 {
			index = 0
		} else {
			return "", ErrNotFound
		}
	}

	removedID := cluster.Hsms[index].HsmID
	cluster.Hsms = append(cluster.Hsms[:index], cluster.Hsms[index+1:]...)
	if len(cluster.Hsms) == 0 {
		if cluster.State == "ACTIVE" {
			cluster.State = "INITIALIZED"
			cluster.StateMessage = "No HSMs attached"
		}
	}
	return removedID, nil
}

func (s *Service) ModifyCluster(backupRetentionPolicy BackupRetentionPolicy, clusterID string) (Cluster, error) {
	clusterID = strings.TrimSpace(clusterID)
	if clusterID == "" {
		return Cluster{}, ErrInvalidParameter
	}
	retention := normalizeBackupRetentionPolicy(backupRetentionPolicy)

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.clusters[clusterID]
	if !ok {
		return Cluster{}, ErrNotFound
	}
	cluster.BackupRetentionPolicy = retention
	return cloneCluster(*cluster), nil
}

func (s *Service) DescribeBackups(
	nextToken string,
	maxResults int32,
	filters map[string][]string,
	_ *bool,
	sortAscending *bool,
) ([]Backup, string, error) {
	start, limit, err := normalizeListInput(nextToken, maxResults)
	if err != nil {
		return nil, "", err
	}
	filtered := normalizeFilters(filters)
	ascending := true
	if sortAscending != nil {
		ascending = *sortAscending
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.backups))
	for id := range s.backups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if !ascending {
		for i, j := 0, len(ids)-1; i < j; i, j = i+1, j-1 {
			ids[i], ids[j] = ids[j], ids[i]
		}
	}

	matched := make([]Backup, 0, len(ids))
	for _, id := range ids {
		backup := s.backups[id]
		if !backupMatchesFilters(backup, filtered) {
			continue
		}
		matched = append(matched, cloneBackup(*backup))
	}

	return paginateBackups(matched, start, limit), nextPageToken(len(matched), start, limit), nil
}

func (s *Service) CopyBackupToRegion(destinationRegion, backupID string, tagList []Tag) (Backup, error) {
	destinationRegion = strings.TrimSpace(destinationRegion)
	backupID = strings.TrimSpace(backupID)
	if destinationRegion == "" || backupID == "" {
		return Backup{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	source, ok := s.backups[backupID]
	if !ok {
		return Backup{}, ErrNotFound
	}

	copyBackup := cloneBackup(*source)
	copyBackup.BackupID = s.nextIDLocked("backup")
	copyBackup.BackupARN = backupARN(copyBackup.BackupID)
	copyBackup.SourceRegion = defaultRegion
	copyBackup.SourceBackup = backupID
	copyBackup.SourceCluster = source.ClusterID
	copyBackup.CopyTimestamp = time.Now().UTC()
	copyBackup.TagList = normalizeTags(tagList)
	s.backups[copyBackup.BackupID] = &copyBackup
	s.resourceTags[copyBackup.BackupARN] = tagsToMap(copyBackup.TagList)
	s.resourceTags[copyBackup.BackupID] = cloneStringMap(s.resourceTags[copyBackup.BackupARN])
	return cloneBackup(copyBackup), nil
}

func (s *Service) RestoreBackup(backupID string) (Backup, error) {
	backupID = strings.TrimSpace(backupID)
	if backupID == "" {
		return Backup{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	backup, ok := s.backups[backupID]
	if !ok {
		return Backup{}, ErrNotFound
	}
	backup.BackupState = "RESTORED"
	return cloneBackup(*backup), nil
}

func (s *Service) DeleteBackup(backupID string) (Backup, error) {
	backupID = strings.TrimSpace(backupID)
	if backupID == "" {
		return Backup{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	backup, ok := s.backups[backupID]
	if !ok {
		return Backup{}, ErrNotFound
	}
	out := cloneBackup(*backup)
	out.BackupState = "DELETED"
	out.DeleteTimestamp = time.Now().UTC()
	delete(s.backups, backupID)
	delete(s.resourceTags, backupID)
	delete(s.resourceTags, backup.BackupARN)
	delete(s.resourcePolicies, backup.BackupARN)
	return out, nil
}

func (s *Service) ModifyBackupAttributes(backupID string, neverExpires bool) (Backup, error) {
	backupID = strings.TrimSpace(backupID)
	if backupID == "" {
		return Backup{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	backup, ok := s.backups[backupID]
	if !ok {
		return Backup{}, ErrNotFound
	}
	backup.NeverExpires = neverExpires
	return cloneBackup(*backup), nil
}

func (s *Service) TagResource(resourceID string, tagList []Tag) error {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return ErrInvalidParameter
	}
	incoming := normalizeTags(tagList)

	s.mu.Lock()
	defer s.mu.Unlock()

	resolvedID, resolvedARN, ok := s.resolveResourceLocked(resourceID)
	if !ok {
		return ErrNotFound
	}
	existing := cloneStringMap(s.resourceTags[resolvedARN])
	for _, tag := range incoming {
		existing[tag.Key] = tag.Value
	}
	s.resourceTags[resolvedARN] = existing
	s.resourceTags[resolvedID] = cloneStringMap(existing)
	return nil
}

func (s *Service) UntagResource(resourceID string, tagKeys []string) error {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resolvedID, resolvedARN, ok := s.resolveResourceLocked(resourceID)
	if !ok {
		return ErrNotFound
	}
	existing := cloneStringMap(s.resourceTags[resolvedARN])
	for _, key := range tagKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		delete(existing, key)
	}
	s.resourceTags[resolvedARN] = existing
	s.resourceTags[resolvedID] = cloneStringMap(existing)
	return nil
}

func (s *Service) ListTags(resourceID, nextToken string, maxResults int32) ([]Tag, string, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return nil, "", ErrInvalidParameter
	}
	start, limit, err := normalizeListInput(nextToken, maxResults)
	if err != nil {
		return nil, "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, resolvedARN, ok := s.resolveResourceLocked(resourceID)
	if !ok {
		return nil, "", ErrNotFound
	}

	tags := mapToTags(s.resourceTags[resolvedARN])
	end := start + int(limit)
	if start >= len(tags) {
		return []Tag{}, "", nil
	}
	if end > len(tags) {
		end = len(tags)
	}
	out := make([]Tag, 0, end-start)
	out = append(out, tags[start:end]...)
	return out, nextPageToken(len(tags), start, limit), nil
}

func (s *Service) PutResourcePolicy(resourceARN, policy string) (string, string, error) {
	resourceARN = strings.TrimSpace(resourceARN)
	policy = strings.TrimSpace(policy)
	if resourceARN == "" || policy == "" {
		return "", "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.resourceExistsLocked(resourceARN) {
		return "", "", ErrNotFound
	}
	s.resourcePolicies[resourceARN] = policy
	return resourceARN, policy, nil
}

func (s *Service) GetResourcePolicy(resourceARN string) (string, error) {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.resourceExistsLocked(resourceARN) {
		return "", ErrNotFound
	}
	return s.resourcePolicies[resourceARN], nil
}

func (s *Service) DeleteResourcePolicy(resourceARN string) (string, string, error) {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return "", "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.resourceExistsLocked(resourceARN) {
		return "", "", ErrNotFound
	}
	policy := s.resourcePolicies[resourceARN]
	delete(s.resourcePolicies, resourceARN)
	return resourceARN, policy, nil
}

func (s *Service) newBackupLocked(cluster *Cluster, sourceBackupID, sourceRegion, sourceCluster string, tags []Tag) *Backup {
	now := time.Now().UTC()
	backupID := s.nextIDLocked("backup")
	if sourceRegion == "" {
		sourceRegion = defaultRegion
	}
	if sourceCluster == "" {
		sourceCluster = cluster.ClusterID
	}
	backup := &Backup{
		BackupID:        backupID,
		BackupARN:       backupARN(backupID),
		BackupState:     "READY",
		ClusterID:       cluster.ClusterID,
		CreateTimestamp: now,
		CopyTimestamp:   now,
		NeverExpires:    false,
		SourceRegion:    sourceRegion,
		SourceBackup:    sourceBackupID,
		SourceCluster:   sourceCluster,
		DeleteTimestamp: time.Time{},
		TagList:         normalizeTags(tags),
		HsmType:         cluster.HsmType,
		Mode:            cluster.Mode,
	}
	return backup
}

func (s *Service) resolveResourceLocked(resourceID string) (string, string, bool) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return "", "", false
	}
	if strings.HasPrefix(resourceID, "arn:aws:cloudhsm:") {
		switch {
		case strings.Contains(resourceID, ":cluster/"):
			id := resourceID[strings.LastIndex(resourceID, "/")+1:]
			if _, ok := s.clusters[id]; ok {
				return id, resourceID, true
			}
		case strings.Contains(resourceID, ":backup/"):
			id := resourceID[strings.LastIndex(resourceID, "/")+1:]
			if _, ok := s.backups[id]; ok {
				return id, resourceID, true
			}
		}
		return "", "", false
	}
	if cluster, ok := s.clusters[resourceID]; ok {
		return cluster.ClusterID, clusterARN(cluster.ClusterID), true
	}
	if backup, ok := s.backups[resourceID]; ok {
		return backup.BackupID, backup.BackupARN, true
	}
	return "", "", false
}

func (s *Service) resourceExistsLocked(resourceARN string) bool {
	if _, _, ok := s.resolveResourceLocked(resourceARN); ok {
		return true
	}
	return false
}

func (s *Service) nextIDLocked(prefix string) string {
	s.seq++
	return fmt.Sprintf("%s-%012x", prefix, s.seq)
}

func clusterARN(clusterID string) string {
	return "arn:aws:cloudhsm:" + defaultRegion + ":" + defaultAccountID + ":cluster/" + strings.TrimSpace(clusterID)
}

func backupARN(backupID string) string {
	return "arn:aws:cloudhsm:" + defaultRegion + ":" + defaultAccountID + ":backup/" + strings.TrimSpace(backupID)
}

func normalizeBackupRetentionPolicy(policy BackupRetentionPolicy) BackupRetentionPolicy {
	policyType := strings.ToUpper(strings.TrimSpace(policy.Type))
	if policyType == "" {
		policyType = "DAYS"
	}
	value := strings.TrimSpace(policy.Value)
	if value == "" {
		value = "30"
	}
	return BackupRetentionPolicy{Type: policyType, Value: value}
}

func normalizeTags(in []Tag) []Tag {
	if len(in) == 0 {
		return []Tag{}
	}
	merged := map[string]string{}
	for _, tag := range in {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			continue
		}
		merged[key] = strings.TrimSpace(tag.Value)
	}
	return mapToTags(merged)
}

func normalizeNonEmptyStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, item := range in {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func normalizeFilters(in map[string][]string) map[string][]string {
	if len(in) == 0 {
		return map[string][]string{}
	}
	out := make(map[string][]string, len(in))
	for key, values := range in {
		canonical := strings.ToLower(strings.TrimSpace(key))
		if canonical == "" {
			continue
		}
		normalizedValues := normalizeNonEmptyStrings(values)
		if len(normalizedValues) == 0 {
			continue
		}
		out[canonical] = normalizedValues
	}
	return out
}

func normalizeListInput(nextToken string, maxResults int32) (int, int32, error) {
	start, err := parseOffset(nextToken)
	if err != nil {
		return 0, 0, err
	}
	limit := maxResults
	if limit == 0 {
		limit = defaultMaxResults
	}
	if limit < 0 || limit > maxResultsUpper {
		return 0, 0, ErrInvalidParameter
	}
	return start, limit, nil
}

func parseOffset(nextToken string) (int, error) {
	nextToken = strings.TrimSpace(nextToken)
	if nextToken == "" {
		return 0, nil
	}
	offset := 0
	for _, ch := range nextToken {
		if ch < '0' || ch > '9' {
			return 0, ErrInvalidParameter
		}
		offset = offset*10 + int(ch-'0')
	}
	return offset, nil
}

func nextPageToken(total, start int, limit int32) string {
	end := start + int(limit)
	if end < total {
		return fmt.Sprintf("%d", end)
	}
	return ""
}

func clusterMatchesFilters(cluster *Cluster, filters map[string][]string) bool {
	if len(filters) == 0 {
		return true
	}
	for key, values := range filters {
		switch key {
		case "clusterids", "clusterid":
			if !containsIgnoreCase(values, cluster.ClusterID) {
				return false
			}
		case "hsmtype":
			if !containsIgnoreCase(values, cluster.HsmType) {
				return false
			}
		case "states", "state":
			if !containsIgnoreCase(values, cluster.State) {
				return false
			}
		}
	}
	return true
}

func backupMatchesFilters(backup *Backup, filters map[string][]string) bool {
	if len(filters) == 0 {
		return true
	}
	for key, values := range filters {
		switch key {
		case "backupids", "backupid":
			if !containsIgnoreCase(values, backup.BackupID) {
				return false
			}
		case "clusterids", "clusterid":
			if !containsIgnoreCase(values, backup.ClusterID) {
				return false
			}
		case "states", "state":
			if !containsIgnoreCase(values, backup.BackupState) {
				return false
			}
		case "sourcebackupids":
			if !containsIgnoreCase(values, backup.SourceBackup) {
				return false
			}
		}
	}
	return true
}

func containsIgnoreCase(values []string, value string) bool {
	value = strings.TrimSpace(value)
	for _, item := range values {
		if strings.EqualFold(strings.TrimSpace(item), value) {
			return true
		}
	}
	return false
}

func paginateClusters(in []Cluster, start int, limit int32) []Cluster {
	if start >= len(in) {
		return []Cluster{}
	}
	end := start + int(limit)
	if end > len(in) {
		end = len(in)
	}
	out := make([]Cluster, 0, end-start)
	out = append(out, in[start:end]...)
	return out
}

func paginateBackups(in []Backup, start int, limit int32) []Backup {
	if start >= len(in) {
		return []Backup{}
	}
	end := start + int(limit)
	if end > len(in) {
		end = len(in)
	}
	out := make([]Backup, 0, end-start)
	out = append(out, in[start:end]...)
	return out
}

func cloneCluster(in Cluster) Cluster {
	out := in
	out.Hsms = make([]Hsm, len(in.Hsms))
	copy(out.Hsms, in.Hsms)
	out.SubnetMapping = make(map[string]string, len(in.SubnetMapping))
	for k, v := range in.SubnetMapping {
		out.SubnetMapping[k] = v
	}
	out.TagList = make([]Tag, len(in.TagList))
	copy(out.TagList, in.TagList)
	return out
}

func cloneBackup(in Backup) Backup {
	out := in
	out.TagList = make([]Tag, len(in.TagList))
	copy(out.TagList, in.TagList)
	return out
}

func mapToTags(in map[string]string) []Tag {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]Tag, 0, len(keys))
	for _, key := range keys {
		out = append(out, Tag{Key: key, Value: in[key]})
	}
	return out
}

func tagsToMap(in []Tag) map[string]string {
	out := map[string]string{}
	for _, tag := range in {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(tag.Value)
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
