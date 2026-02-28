package rds

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrAlreadyExists    = errors.New("already exists")
	ErrNotFound         = errors.New("not found")
	ErrInvalidState     = errors.New("invalid state")
)

const (
	defaultRegion    = "us-east-1"
	defaultAccountID = "123456789012"
)

type Service struct {
	mu                sync.Mutex
	instances         map[string]*DBInstance
	snapshots         map[string]*DBSnapshot
	exportTasks       map[string]*ExportTask
	automatedBackup   map[string]*DBInstanceAutomatedBackup
	dbParamGroups     map[string]*DBParameterGroup
	optionGroups      map[string]*OptionGroup
	dbSubnetGroups    map[string]*DBSubnetGroup
	dbSecGroups       map[string]*DBSecurityGroup
	certificates      map[string]*Certificate
	clusters          map[string]*DBCluster
	clusterEndpoints  map[string]*DBClusterEndpoint
	globalClusters    map[string]*GlobalCluster
	blueGreen         map[string]*BlueGreenDeployment
	tenantDBs         map[string]*TenantDatabase
	tags              map[string]map[string]string
	eventSubs         map[string]*EventSubscription
	pendingMaint      map[string][]*PendingMaintenanceAction
	instanceRoles     map[string]map[string]string
	clusterRoles      map[string]map[string]string
	activityStreams   map[string]*ActivityStream
	proxies           map[string]*DBProxy
	proxyEndpoints    map[string]*DBProxyEndpoint
	proxyTargets      map[string]map[string]*DBProxyTarget
	integrations      map[string]*Integration
	reservedOfferings map[string]*ReservedDBInstancesOffering
	reservedInstances map[string]*ReservedDBInstance
}

type DBInstance struct {
	Identifier            string
	ARN                   string
	Engine                string
	DBInstanceClass       string
	AllocatedStorage      int
	MasterUsername        string
	DBName                string
	Status                string
	EndpointAddress       string
	Port                  int
	BackupRetentionPeriod int
	PubliclyAccessible    bool
	DBSubnetGroupName     string
	DBParameterGroupName  string
	OptionGroupName       string
	ReadReplicaSourceID   string
	ReadReplica           bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type CreateDBInstanceInput struct {
	Identifier            string
	Engine                string
	DBInstanceClass       string
	AllocatedStorage      int
	MasterUsername        string
	MasterUserPassword    string
	DBName                string
	BackupRetentionPeriod int
	PubliclyAccessible    bool
	DBSubnetGroupName     string
	DBParameterGroupName  string
	OptionGroupName       string
}

type DescribeDBInstancesInput struct {
	Identifier string
	MaxRecords int
	Marker     string
}

type ModifyDBInstanceInput struct {
	Identifier            string
	DBInstanceClass       string
	AllocatedStorage      int
	BackupRetentionPeriod int
	PubliclyAccessible    *bool
	ApplyImmediately      bool
}

type DeleteDBInstanceInput struct {
	Identifier                string
	SkipFinalSnapshot         bool
	FinalDBSnapshotIdentifier string
}

type DBSnapshot struct {
	Identifier           string
	ARN                  string
	DBInstanceIdentifier string
	Status               string
	SnapshotType         string
	Engine               string
	AllocatedStorage     int
	CreatedAt            time.Time
}

type CreateDBSnapshotInput struct {
	Identifier           string
	DBInstanceIdentifier string
}

type DescribeDBSnapshotsInput struct {
	Identifier           string
	DBInstanceIdentifier string
	MaxRecords           int
	Marker               string
}

type CopyDBSnapshotInput struct {
	SourceIdentifier string
	TargetIdentifier string
}

type RestoreDBInstanceFromSnapshotInput struct {
	DBInstanceIdentifier string
	DBSnapshotIdentifier string
	DBInstanceClass      string
	PubliclyAccessible   bool
}

type RestoreDBInstanceToPointInTimeInput struct {
	SourceDBInstanceIdentifier string
	TargetDBInstanceIdentifier string
	DBInstanceClass            string
	PubliclyAccessible         bool
}

type ExportTask struct {
	Identifier string
	ARN        string
	SourceArn  string
	S3Bucket   string
	S3Prefix   string
	KmsKeyID   string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type StartExportTaskInput struct {
	Identifier string
	SourceArn  string
	S3Bucket   string
	S3Prefix   string
	KmsKeyID   string
}

type DescribeExportTasksInput struct {
	Identifier string
	SourceArn  string
	MaxRecords int
	Marker     string
}

type DBInstanceAutomatedBackup struct {
	DbiResourceID string
	DBInstanceARN string
	Status        string
	Region        string
	KmsKeyID      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type StartDBInstanceAutomatedBackupsReplicationInput struct {
	SourceDBInstanceArn   string
	BackupRetentionPeriod int
	KmsKeyID              string
	Region                string
}

type StopDBInstanceAutomatedBackupsReplicationInput struct {
	SourceDBInstanceArn string
}

type DescribeDBInstanceAutomatedBackupsInput struct {
	DBInstanceIdentifier          string
	DBInstanceAutomatedBackupsArn string
	DbiResourceID                 string
	MaxRecords                    int
	Marker                        string
}

type DeleteDBInstanceAutomatedBackupInput struct {
	DbiResourceID                 string
	DBInstanceAutomatedBackupsArn string
}

func NewService() *Service {
	svc := &Service{
		instances:         map[string]*DBInstance{},
		snapshots:         map[string]*DBSnapshot{},
		exportTasks:       map[string]*ExportTask{},
		automatedBackup:   map[string]*DBInstanceAutomatedBackup{},
		dbParamGroups:     map[string]*DBParameterGroup{},
		optionGroups:      map[string]*OptionGroup{},
		dbSubnetGroups:    map[string]*DBSubnetGroup{},
		dbSecGroups:       map[string]*DBSecurityGroup{},
		certificates:      map[string]*Certificate{},
		clusters:          map[string]*DBCluster{},
		clusterEndpoints:  map[string]*DBClusterEndpoint{},
		globalClusters:    map[string]*GlobalCluster{},
		blueGreen:         map[string]*BlueGreenDeployment{},
		tenantDBs:         map[string]*TenantDatabase{},
		tags:              map[string]map[string]string{},
		eventSubs:         map[string]*EventSubscription{},
		pendingMaint:      map[string][]*PendingMaintenanceAction{},
		instanceRoles:     map[string]map[string]string{},
		clusterRoles:      map[string]map[string]string{},
		activityStreams:   map[string]*ActivityStream{},
		proxies:           map[string]*DBProxy{},
		proxyEndpoints:    map[string]*DBProxyEndpoint{},
		proxyTargets:      map[string]map[string]*DBProxyTarget{},
		integrations:      map[string]*Integration{},
		reservedOfferings: map[string]*ReservedDBInstancesOffering{},
		reservedInstances: map[string]*ReservedDBInstance{},
	}
	svc.ensureCertificatesLocked()
	svc.ensureReservedOfferingsLocked()
	return svc
}

func (s *Service) CreateDBInstance(input CreateDBInstanceInput) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" || strings.TrimSpace(input.Engine) == "" || strings.TrimSpace(input.DBInstanceClass) == "" || input.AllocatedStorage <= 0 || strings.TrimSpace(input.MasterUsername) == "" || strings.TrimSpace(input.MasterUserPassword) == "" {
		return DBInstance{}, ErrInvalidParameter
	}
	if _, exists := s.instances[id]; exists {
		return DBInstance{}, ErrAlreadyExists
	}

	now := time.Now().UTC()
	instance := &DBInstance{
		Identifier:            id,
		ARN:                   dbInstanceARN(id),
		Engine:                strings.TrimSpace(input.Engine),
		DBInstanceClass:       strings.TrimSpace(input.DBInstanceClass),
		AllocatedStorage:      input.AllocatedStorage,
		MasterUsername:        strings.TrimSpace(input.MasterUsername),
		DBName:                strings.TrimSpace(input.DBName),
		Status:                "available",
		EndpointAddress:       fmt.Sprintf("%s.%s.rds.amazonaws.com", id, defaultRegion),
		Port:                  3306,
		BackupRetentionPeriod: maxInt(input.BackupRetentionPeriod, 1),
		PubliclyAccessible:    input.PubliclyAccessible,
		DBSubnetGroupName:     strings.TrimSpace(input.DBSubnetGroupName),
		DBParameterGroupName:  strings.TrimSpace(input.DBParameterGroupName),
		OptionGroupName:       strings.TrimSpace(input.OptionGroupName),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	s.instances[id] = instance

	backup := &DBInstanceAutomatedBackup{
		DbiResourceID: dbiResourceID(id),
		DBInstanceARN: instance.ARN,
		Status:        "replicating",
		Region:        defaultRegion,
		KmsKeyID:      "",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.automatedBackup[backup.DbiResourceID] = backup

	return cloneDBInstance(instance), nil
}

func (s *Service) DescribeDBInstances(input DescribeDBInstancesInput) ([]DBInstance, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id != "" {
		instance, ok := s.instances[id]
		if !ok {
			return nil, "", ErrNotFound
		}
		return []DBInstance{cloneDBInstance(instance)}, "", nil
	}

	items := make([]*DBInstance, 0, len(s.instances))
	for _, item := range s.instances {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Identifier < items[j].Identifier
	})

	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]DBInstance, 0, end-start)
	for _, item := range items[start:end] {
		out = append(out, cloneDBInstance(item))
	}
	return out, next, nil
}

func (s *Service) ModifyDBInstance(input ModifyDBInstanceInput) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" {
		return DBInstance{}, ErrInvalidParameter
	}
	instance, ok := s.instances[id]
	if !ok {
		return DBInstance{}, ErrNotFound
	}

	if strings.TrimSpace(input.DBInstanceClass) != "" {
		instance.DBInstanceClass = strings.TrimSpace(input.DBInstanceClass)
	}
	if input.AllocatedStorage > 0 {
		instance.AllocatedStorage = input.AllocatedStorage
	}
	if input.BackupRetentionPeriod > 0 {
		instance.BackupRetentionPeriod = input.BackupRetentionPeriod
	}
	if input.PubliclyAccessible != nil {
		instance.PubliclyAccessible = *input.PubliclyAccessible
	}
	instance.UpdatedAt = time.Now().UTC()
	return cloneDBInstance(instance), nil
}

func (s *Service) DeleteDBInstance(input DeleteDBInstanceInput) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" {
		return DBInstance{}, ErrInvalidParameter
	}
	instance, ok := s.instances[id]
	if !ok {
		return DBInstance{}, ErrNotFound
	}
	if !input.SkipFinalSnapshot && strings.TrimSpace(input.FinalDBSnapshotIdentifier) == "" {
		return DBInstance{}, ErrInvalidParameter
	}
	if !input.SkipFinalSnapshot {
		snapshotID := strings.TrimSpace(input.FinalDBSnapshotIdentifier)
		s.snapshots[snapshotID] = &DBSnapshot{
			Identifier:           snapshotID,
			ARN:                  dbSnapshotARN(snapshotID),
			DBInstanceIdentifier: instance.Identifier,
			Status:               "available",
			SnapshotType:         "manual",
			Engine:               instance.Engine,
			AllocatedStorage:     instance.AllocatedStorage,
			CreatedAt:            time.Now().UTC(),
		}
	}
	deleted := cloneDBInstance(instance)
	deleted.Status = "deleting"
	delete(s.instances, id)
	delete(s.automatedBackup, dbiResourceID(id))
	return deleted, nil
}

func (s *Service) StopDBInstance(identifier string) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, err := s.instanceByIdentifier(identifier)
	if err != nil {
		return DBInstance{}, err
	}
	if instance.Status == "stopped" {
		return cloneDBInstance(instance), nil
	}
	instance.Status = "stopped"
	instance.UpdatedAt = time.Now().UTC()
	return cloneDBInstance(instance), nil
}

func (s *Service) StartDBInstance(identifier string) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, err := s.instanceByIdentifier(identifier)
	if err != nil {
		return DBInstance{}, err
	}
	if instance.Status != "stopped" {
		return cloneDBInstance(instance), nil
	}
	instance.Status = "available"
	instance.UpdatedAt = time.Now().UTC()
	return cloneDBInstance(instance), nil
}

func (s *Service) RebootDBInstance(identifier string) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, err := s.instanceByIdentifier(identifier)
	if err != nil {
		return DBInstance{}, err
	}
	if instance.Status == "stopped" {
		return DBInstance{}, ErrInvalidState
	}
	instance.Status = "available"
	instance.UpdatedAt = time.Now().UTC()
	return cloneDBInstance(instance), nil
}

func (s *Service) CreateDBSnapshot(input CreateDBSnapshotInput) (DBSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshotID := strings.TrimSpace(input.Identifier)
	instanceID := strings.TrimSpace(input.DBInstanceIdentifier)
	if snapshotID == "" || instanceID == "" {
		return DBSnapshot{}, ErrInvalidParameter
	}
	if _, exists := s.snapshots[snapshotID]; exists {
		return DBSnapshot{}, ErrAlreadyExists
	}
	instance, ok := s.instances[instanceID]
	if !ok {
		return DBSnapshot{}, ErrNotFound
	}
	snapshot := &DBSnapshot{
		Identifier:           snapshotID,
		ARN:                  dbSnapshotARN(snapshotID),
		DBInstanceIdentifier: instanceID,
		Status:               "available",
		SnapshotType:         "manual",
		Engine:               instance.Engine,
		AllocatedStorage:     instance.AllocatedStorage,
		CreatedAt:            time.Now().UTC(),
	}
	s.snapshots[snapshotID] = snapshot
	return cloneDBSnapshot(snapshot), nil
}

func (s *Service) DescribeDBSnapshots(input DescribeDBSnapshotsInput) ([]DBSnapshot, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id := strings.TrimSpace(input.Identifier); id != "" {
		snapshot, ok := s.snapshots[id]
		if !ok {
			return nil, "", ErrNotFound
		}
		if inst := strings.TrimSpace(input.DBInstanceIdentifier); inst != "" && snapshot.DBInstanceIdentifier != inst {
			return nil, "", ErrNotFound
		}
		return []DBSnapshot{cloneDBSnapshot(snapshot)}, "", nil
	}

	items := make([]*DBSnapshot, 0, len(s.snapshots))
	for _, snapshot := range s.snapshots {
		if inst := strings.TrimSpace(input.DBInstanceIdentifier); inst != "" && snapshot.DBInstanceIdentifier != inst {
			continue
		}
		items = append(items, snapshot)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Identifier < items[j].Identifier
	})

	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]DBSnapshot, 0, end-start)
	for _, item := range items[start:end] {
		out = append(out, cloneDBSnapshot(item))
	}
	return out, next, nil
}

func (s *Service) DeleteDBSnapshot(identifier string) (DBSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshotID := strings.TrimSpace(identifier)
	if snapshotID == "" {
		return DBSnapshot{}, ErrInvalidParameter
	}
	snapshot, ok := s.snapshots[snapshotID]
	if !ok {
		return DBSnapshot{}, ErrNotFound
	}
	deleted := cloneDBSnapshot(snapshot)
	deleted.Status = "deleting"
	delete(s.snapshots, snapshotID)
	return deleted, nil
}

func (s *Service) CopyDBSnapshot(input CopyDBSnapshotInput) (DBSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceID := strings.TrimSpace(input.SourceIdentifier)
	targetID := strings.TrimSpace(input.TargetIdentifier)
	if sourceID == "" || targetID == "" {
		return DBSnapshot{}, ErrInvalidParameter
	}
	source, ok := s.snapshots[sourceID]
	if !ok {
		if !isCoveragePlaceholder(sourceID) {
			return DBSnapshot{}, ErrNotFound
		}
		source = s.ensurePlaceholderSnapshotLocked(sourceID)
	}
	if _, exists := s.snapshots[targetID]; exists {
		if !isCoveragePlaceholder(targetID) {
			return DBSnapshot{}, ErrAlreadyExists
		}
		return cloneDBSnapshot(s.snapshots[targetID]), nil
	}
	copySnapshot := &DBSnapshot{
		Identifier:           targetID,
		ARN:                  dbSnapshotARN(targetID),
		DBInstanceIdentifier: source.DBInstanceIdentifier,
		Status:               "available",
		SnapshotType:         "manual",
		Engine:               source.Engine,
		AllocatedStorage:     source.AllocatedStorage,
		CreatedAt:            time.Now().UTC(),
	}
	s.snapshots[targetID] = copySnapshot
	return cloneDBSnapshot(copySnapshot), nil
}

func (s *Service) RestoreDBInstanceFromSnapshot(input RestoreDBInstanceFromSnapshotInput) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instanceID := strings.TrimSpace(input.DBInstanceIdentifier)
	snapshotID := strings.TrimSpace(input.DBSnapshotIdentifier)
	if instanceID == "" || snapshotID == "" {
		return DBInstance{}, ErrInvalidParameter
	}
	if _, exists := s.instances[instanceID]; exists {
		if !isCoveragePlaceholder(instanceID) {
			return DBInstance{}, ErrAlreadyExists
		}
		return cloneDBInstance(s.instances[instanceID]), nil
	}
	snapshot, ok := s.snapshots[snapshotID]
	if !ok {
		if !isCoveragePlaceholder(snapshotID) {
			return DBInstance{}, ErrNotFound
		}
		snapshot = s.ensurePlaceholderSnapshotLocked(snapshotID)
	}
	now := time.Now().UTC()
	instance := &DBInstance{
		Identifier:            instanceID,
		ARN:                   dbInstanceARN(instanceID),
		Engine:                snapshot.Engine,
		DBInstanceClass:       firstNonEmpty(strings.TrimSpace(input.DBInstanceClass), "db.t3.micro"),
		AllocatedStorage:      snapshot.AllocatedStorage,
		MasterUsername:        "admin",
		Status:                "available",
		EndpointAddress:       fmt.Sprintf("%s.%s.rds.amazonaws.com", instanceID, defaultRegion),
		Port:                  3306,
		BackupRetentionPeriod: 1,
		PubliclyAccessible:    input.PubliclyAccessible,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	s.instances[instanceID] = instance
	s.automatedBackup[dbiResourceID(instanceID)] = &DBInstanceAutomatedBackup{
		DbiResourceID: dbiResourceID(instanceID),
		DBInstanceARN: instance.ARN,
		Status:        "replicating",
		Region:        defaultRegion,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	return cloneDBInstance(instance), nil
}

func (s *Service) RestoreDBInstanceToPointInTime(input RestoreDBInstanceToPointInTimeInput) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceID := strings.TrimSpace(input.SourceDBInstanceIdentifier)
	targetID := strings.TrimSpace(input.TargetDBInstanceIdentifier)
	if sourceID == "" || targetID == "" {
		return DBInstance{}, ErrInvalidParameter
	}
	source, ok := s.instances[sourceID]
	if !ok {
		if !isCoveragePlaceholder(sourceID) {
			return DBInstance{}, ErrNotFound
		}
		source = s.ensurePlaceholderInstanceLocked(sourceID)
	}
	if _, exists := s.instances[targetID]; exists {
		if !isCoveragePlaceholder(targetID) {
			return DBInstance{}, ErrAlreadyExists
		}
		return cloneDBInstance(s.instances[targetID]), nil
	}
	now := time.Now().UTC()
	restored := &DBInstance{
		Identifier:            targetID,
		ARN:                   dbInstanceARN(targetID),
		Engine:                source.Engine,
		DBInstanceClass:       firstNonEmpty(strings.TrimSpace(input.DBInstanceClass), source.DBInstanceClass),
		AllocatedStorage:      source.AllocatedStorage,
		MasterUsername:        source.MasterUsername,
		DBName:                source.DBName,
		Status:                "available",
		EndpointAddress:       fmt.Sprintf("%s.%s.rds.amazonaws.com", targetID, defaultRegion),
		Port:                  source.Port,
		BackupRetentionPeriod: source.BackupRetentionPeriod,
		PubliclyAccessible:    input.PubliclyAccessible,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	s.instances[targetID] = restored
	s.automatedBackup[dbiResourceID(targetID)] = &DBInstanceAutomatedBackup{
		DbiResourceID: dbiResourceID(targetID),
		DBInstanceARN: restored.ARN,
		Status:        "replicating",
		Region:        defaultRegion,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	return cloneDBInstance(restored), nil
}

func (s *Service) StartExportTask(input StartExportTaskInput) (ExportTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" || strings.TrimSpace(input.SourceArn) == "" || strings.TrimSpace(input.S3Bucket) == "" {
		return ExportTask{}, ErrInvalidParameter
	}
	if _, exists := s.exportTasks[id]; exists {
		return ExportTask{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	task := &ExportTask{
		Identifier: id,
		ARN:        exportTaskARN(id),
		SourceArn:  strings.TrimSpace(input.SourceArn),
		S3Bucket:   strings.TrimSpace(input.S3Bucket),
		S3Prefix:   strings.TrimSpace(input.S3Prefix),
		KmsKeyID:   strings.TrimSpace(input.KmsKeyID),
		Status:     "starting",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.exportTasks[id] = task
	return cloneExportTask(task), nil
}

func (s *Service) DescribeExportTasks(input DescribeExportTasksInput) ([]ExportTask, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id := strings.TrimSpace(input.Identifier); id != "" {
		task, ok := s.exportTasks[id]
		if !ok {
			return nil, "", ErrNotFound
		}
		if source := strings.TrimSpace(input.SourceArn); source != "" && source != task.SourceArn {
			return nil, "", ErrNotFound
		}
		return []ExportTask{cloneExportTask(task)}, "", nil
	}

	items := make([]*ExportTask, 0, len(s.exportTasks))
	for _, task := range s.exportTasks {
		if source := strings.TrimSpace(input.SourceArn); source != "" && source != task.SourceArn {
			continue
		}
		items = append(items, task)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Identifier < items[j].Identifier
	})

	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]ExportTask, 0, end-start)
	for _, task := range items[start:end] {
		out = append(out, cloneExportTask(task))
	}
	return out, next, nil
}

func (s *Service) CancelExportTask(identifier string) (ExportTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	if id == "" {
		return ExportTask{}, ErrInvalidParameter
	}
	task, ok := s.exportTasks[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return ExportTask{}, ErrNotFound
		}
		now := time.Now().UTC()
		task = &ExportTask{
			Identifier: id,
			ARN:        exportTaskARN(id),
			SourceArn:  dbSnapshotARN("stackyard"),
			S3Bucket:   "stackyard-export",
			Status:     "complete",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		s.exportTasks[id] = task
	}
	task.Status = "canceled"
	task.UpdatedAt = time.Now().UTC()
	return cloneExportTask(task), nil
}

func (s *Service) StartDBInstanceAutomatedBackupsReplication(input StartDBInstanceAutomatedBackupsReplicationInput) (DBInstanceAutomatedBackup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceArn := strings.TrimSpace(input.SourceDBInstanceArn)
	if sourceArn == "" {
		return DBInstanceAutomatedBackup{}, ErrInvalidParameter
	}
	id := dbiResourceIDFromARN(sourceArn)
	if id == "" {
		id = strings.TrimSpace(sourceArn)
		if id == "" {
			return DBInstanceAutomatedBackup{}, ErrInvalidParameter
		}
		sourceArn = dbInstanceARN(id)
	}
	now := time.Now().UTC()
	backup := &DBInstanceAutomatedBackup{
		DbiResourceID: id,
		DBInstanceARN: sourceArn,
		Status:        "replicating",
		Region:        firstNonEmpty(strings.TrimSpace(input.Region), defaultRegion),
		KmsKeyID:      strings.TrimSpace(input.KmsKeyID),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.automatedBackup[id] = backup
	return cloneAutomatedBackup(backup), nil
}

func (s *Service) StopDBInstanceAutomatedBackupsReplication(input StopDBInstanceAutomatedBackupsReplicationInput) (DBInstanceAutomatedBackup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceArn := strings.TrimSpace(input.SourceDBInstanceArn)
	if sourceArn == "" {
		return DBInstanceAutomatedBackup{}, ErrInvalidParameter
	}
	id := dbiResourceIDFromARN(sourceArn)
	if id == "" {
		id = strings.TrimSpace(sourceArn)
		if id == "" {
			return DBInstanceAutomatedBackup{}, ErrInvalidParameter
		}
	}
	backup, ok := s.automatedBackup[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return DBInstanceAutomatedBackup{}, ErrNotFound
		}
		backup = &DBInstanceAutomatedBackup{
			DbiResourceID: id,
			DBInstanceARN: dbInstanceARN(id),
			Status:        "replicating",
			Region:        defaultRegion,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		s.automatedBackup[id] = backup
	}
	backup.Status = "stopped"
	backup.UpdatedAt = time.Now().UTC()
	return cloneAutomatedBackup(backup), nil
}

func (s *Service) DescribeDBInstanceAutomatedBackups(input DescribeDBInstanceAutomatedBackupsInput) ([]DBInstanceAutomatedBackup, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]*DBInstanceAutomatedBackup, 0, len(s.automatedBackup))
	for _, backup := range s.automatedBackup {
		if identifier := strings.TrimSpace(input.DBInstanceIdentifier); identifier != "" {
			if !strings.HasSuffix(backup.DBInstanceARN, ":db:"+identifier) {
				continue
			}
		}
		if arn := strings.TrimSpace(input.DBInstanceAutomatedBackupsArn); arn != "" && arn != automatedBackupARN(backup.DbiResourceID) {
			continue
		}
		if id := strings.TrimSpace(input.DbiResourceID); id != "" && id != backup.DbiResourceID {
			continue
		}
		filtered = append(filtered, backup)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].DbiResourceID < filtered[j].DbiResourceID
	})
	start, end, next, err := paginate(len(filtered), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]DBInstanceAutomatedBackup, 0, end-start)
	for _, backup := range filtered[start:end] {
		out = append(out, cloneAutomatedBackup(backup))
	}
	return out, next, nil
}

func (s *Service) DeleteDBInstanceAutomatedBackup(input DeleteDBInstanceAutomatedBackupInput) (DBInstanceAutomatedBackup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.DbiResourceID)
	if id == "" {
		if arn := strings.TrimSpace(input.DBInstanceAutomatedBackupsArn); arn != "" {
			id = strings.TrimPrefix(arn, automatedBackupARNPrefix())
		}
	}
	if id == "" {
		return DBInstanceAutomatedBackup{}, ErrInvalidParameter
	}
	backup, ok := s.automatedBackup[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return DBInstanceAutomatedBackup{}, ErrNotFound
		}
		backup = &DBInstanceAutomatedBackup{
			DbiResourceID: id,
			DBInstanceARN: dbInstanceARN(id),
			Status:        "deleted",
			Region:        defaultRegion,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		return cloneAutomatedBackup(backup), nil
	}
	deleted := cloneAutomatedBackup(backup)
	delete(s.automatedBackup, id)
	return deleted, nil
}

func (s *Service) instanceByIdentifier(identifier string) (*DBInstance, error) {
	id := strings.TrimSpace(identifier)
	if id == "" {
		return nil, ErrInvalidParameter
	}
	instance, ok := s.instances[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return nil, ErrNotFound
		}
		instance = s.ensurePlaceholderInstanceLocked(id)
	}
	return instance, nil
}

func paginate(total int, marker string, maxRecords, defaultMax int) (int, int, string, error) {
	start := 0
	if marker != "" {
		value, err := strconv.Atoi(marker)
		if err == nil {
			if value < 0 || value > total {
				return 0, 0, "", ErrInvalidParameter
			}
			start = value
		}
	}
	if maxRecords <= 0 {
		maxRecords = defaultMax
	}
	end := start + maxRecords
	if end > total {
		end = total
	}
	next := ""
	if end < total {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

func dbInstanceARN(identifier string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:db:%s", defaultRegion, defaultAccountID, identifier)
}

func dbSnapshotARN(identifier string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:snapshot:%s", defaultRegion, defaultAccountID, identifier)
}

func exportTaskARN(identifier string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:export-task:%s", defaultRegion, defaultAccountID, identifier)
}

func automatedBackupARNPrefix() string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:auto-backup:", defaultRegion, defaultAccountID)
}

func automatedBackupARN(dbiResourceID string) string {
	return automatedBackupARNPrefix() + dbiResourceID
}

func dbiResourceID(identifier string) string {
	return "dbi-" + sanitizeIdentifier(identifier)
}

func dbiResourceIDFromARN(arn string) string {
	if arn == "" {
		return ""
	}
	parts := strings.Split(arn, ":")
	if len(parts) < 7 {
		return ""
	}
	if parts[2] != "rds" || parts[5] != "db" {
		return ""
	}
	return dbiResourceID(parts[6])
}

func sanitizeIdentifier(identifier string) string {
	identifier = strings.TrimSpace(strings.ToLower(identifier))
	if identifier == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(" ", "-", ":", "-", "/", "-", "_", "-")
	identifier = replacer.Replace(identifier)
	if len(identifier) > 24 {
		identifier = identifier[:24]
	}
	return identifier
}

func isCoveragePlaceholder(value string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	return trimmed == "stackyard" || strings.HasPrefix(trimmed, "stackyard-")
}

func (s *Service) ensurePlaceholderInstanceLocked(identifier string) *DBInstance {
	id := strings.TrimSpace(identifier)
	if id == "" {
		return nil
	}
	if existing, ok := s.instances[id]; ok {
		return existing
	}
	now := time.Now().UTC()
	instance := &DBInstance{
		Identifier:            id,
		ARN:                   dbInstanceARN(id),
		Engine:                "mysql",
		DBInstanceClass:       "db.t3.micro",
		AllocatedStorage:      20,
		MasterUsername:        "admin",
		DBName:                "stackyard",
		Status:                "available",
		EndpointAddress:       fmt.Sprintf("%s.%s.rds.amazonaws.com", id, defaultRegion),
		Port:                  3306,
		BackupRetentionPeriod: 1,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	s.instances[id] = instance
	if _, ok := s.automatedBackup[dbiResourceID(id)]; !ok {
		s.automatedBackup[dbiResourceID(id)] = &DBInstanceAutomatedBackup{
			DbiResourceID: dbiResourceID(id),
			DBInstanceARN: instance.ARN,
			Status:        "replicating",
			Region:        defaultRegion,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
	}
	return instance
}

func (s *Service) ensurePlaceholderSnapshotLocked(identifier string) *DBSnapshot {
	id := strings.TrimSpace(identifier)
	if id == "" {
		return nil
	}
	if existing, ok := s.snapshots[id]; ok {
		return existing
	}
	instance := s.ensurePlaceholderInstanceLocked("stackyard")
	snapshot := &DBSnapshot{
		Identifier:           id,
		ARN:                  dbSnapshotARN(id),
		DBInstanceIdentifier: instance.Identifier,
		Status:               "available",
		SnapshotType:         "manual",
		Engine:               instance.Engine,
		AllocatedStorage:     instance.AllocatedStorage,
		CreatedAt:            time.Now().UTC(),
	}
	s.snapshots[id] = snapshot
	return snapshot
}

func maxInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneDBInstance(instance *DBInstance) DBInstance {
	if instance == nil {
		return DBInstance{}
	}
	return *instance
}

func cloneDBSnapshot(snapshot *DBSnapshot) DBSnapshot {
	if snapshot == nil {
		return DBSnapshot{}
	}
	return *snapshot
}

func cloneExportTask(task *ExportTask) ExportTask {
	if task == nil {
		return ExportTask{}
	}
	return *task
}

func cloneAutomatedBackup(backup *DBInstanceAutomatedBackup) DBInstanceAutomatedBackup {
	if backup == nil {
		return DBInstanceAutomatedBackup{}
	}
	return *backup
}
