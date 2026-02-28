package memorydb

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultRegion    = "us-east-1"
	defaultAccountID = "123456789012"
)

type FaultError struct {
	Code    string
	Message string
}

func (e *FaultError) Error() string {
	if strings.TrimSpace(e.Message) == "" {
		return e.Code
	}
	return e.Message
}

func fault(code, message string) error {
	return &FaultError{Code: strings.TrimSpace(code), Message: strings.TrimSpace(message)}
}

type ACL struct {
	Name                 string
	Status               string
	UserNames            []string
	MinimumEngineVersion string
	Clusters             []string
	ARN                  string
}

type User struct {
	Name                 string
	Status               string
	AccessString         string
	ACLNames             []string
	MinimumEngineVersion string
	AuthenticationType   string
	PasswordCount        int
	ARN                  string
}

type SubnetGroup struct {
	Name        string
	Description string
	VpcID       string
	SubnetIDs   []string
	ARN         string
}

type Cluster struct {
	Name                    string
	Description             string
	Status                  string
	MultiRegionClusterName  string
	NumberOfShards          int
	NodeType                string
	Engine                  string
	EngineVersion           string
	ParameterGroupName      string
	SecurityGroupIDs        []string
	SubnetGroupName         string
	TLSEnabled              bool
	KmsKeyID                string
	ARN                     string
	SnsTopicArn             string
	SnsTopicStatus          string
	SnapshotRetentionLimit  int
	MaintenanceWindow       string
	SnapshotWindow          string
	ACLName                 string
	AutoMinorVersionUpgrade bool
	DataTiering             string
	Port                    int
}

type ParameterGroup struct {
	Name        string
	Family      string
	Description string
	Parameters  map[string]string
	ARN         string
}

type Parameter struct {
	Name                 string
	Value                string
	Description          string
	DataType             string
	AllowedValues        string
	MinimumEngineVersion string
}

type Snapshot struct {
	Name        string
	Status      string
	Source      string
	KmsKeyID    string
	ARN         string
	ClusterName string
	DataTiering string
}

type RegionalCluster struct {
	ClusterName string
	Region      string
	Status      string
	ARN         string
}

type MultiRegionCluster struct {
	Name                          string
	Description                   string
	Status                        string
	NodeType                      string
	Engine                        string
	EngineVersion                 string
	NumberOfShards                int
	Clusters                      []RegionalCluster
	MultiRegionParameterGroupName string
	TLSEnabled                    bool
	ARN                           string
}

type UnprocessedCluster struct {
	ClusterName  string
	ErrorType    string
	ErrorMessage string
}

type EngineVersion struct {
	Engine               string
	EngineVersion        string
	EnginePatchVersion   string
	ParameterGroupFamily string
	Default              bool
}

type RecurringCharge struct {
	Amount    float64
	Frequency string
}

type ReservedNode struct {
	ReservationID           string
	ReservedNodesOfferingID string
	NodeType                string
	StartTime               time.Time
	Duration                int
	FixedPrice              float64
	NodeCount               int
	OfferingType            string
	State                   string
	RecurringCharges        []RecurringCharge
	ARN                     string
}

type ReservedNodeOffering struct {
	ReservedNodesOfferingID string
	NodeType                string
	Duration                int
	FixedPrice              float64
	OfferingType            string
	RecurringCharges        []RecurringCharge
}

type Event struct {
	SourceName string
	SourceType string
	Message    string
	Date       time.Time
}

type ServiceUpdate struct {
	ClusterName         string
	ServiceUpdateName   string
	ReleaseDate         time.Time
	Description         string
	Status              string
	Type                string
	Engine              string
	NodesUpdated        string
	AutoUpdateStartDate time.Time
}

type CreateUserInput struct {
	UserName           string
	AccessString       string
	AuthenticationType string
	PasswordCount      int
	Tags               map[string]string
}

type UpdateUserInput struct {
	UserName           string
	AccessString       *string
	AuthenticationType *string
	PasswordCount      *int
}

type CreateClusterInput struct {
	ClusterName             string
	NodeType                string
	MultiRegionClusterName  string
	ParameterGroupName      string
	Description             string
	NumShards               int
	SubnetGroupName         string
	SecurityGroupIDs        []string
	MaintenanceWindow       string
	Port                    int
	SnsTopicArn             string
	TLSEnabled              *bool
	KmsKeyID                string
	SnapshotRetentionLimit  int
	SnapshotWindow          string
	ACLName                 string
	Engine                  string
	EngineVersion           string
	AutoMinorVersionUpgrade *bool
	DataTiering             *bool
	Tags                    map[string]string
}

type UpdateClusterInput struct {
	ClusterName            string
	Description            *string
	SecurityGroupIDs       []string
	SecurityGroupIDsSet    bool
	MaintenanceWindow      *string
	SnsTopicArn            *string
	SnsTopicStatus         *string
	ParameterGroupName     *string
	SnapshotWindow         *string
	SnapshotRetentionLimit *int
	NodeType               *string
	Engine                 *string
	EngineVersion          *string
	ShardCount             *int
	ACLName                *string
}

type CreateParameterGroupInput struct {
	Name        string
	Family      string
	Description string
	Tags        map[string]string
}

type CreateSnapshotInput struct {
	ClusterName  string
	SnapshotName string
	KmsKeyID     string
	Tags         map[string]string
}

type CopySnapshotInput struct {
	SourceSnapshotName string
	TargetSnapshotName string
	KmsKeyID           string
	Tags               map[string]string
}

type CreateMultiRegionClusterInput struct {
	NameSuffix                    string
	Description                   string
	Engine                        string
	EngineVersion                 string
	NodeType                      string
	MultiRegionParameterGroupName string
	NumShards                     int
	TLSEnabled                    *bool
	Tags                          map[string]string
}

type UpdateMultiRegionClusterInput struct {
	Name                          string
	NodeType                      *string
	Description                   *string
	EngineVersion                 *string
	ShardCount                    *int
	MultiRegionParameterGroupName *string
}

type Service struct {
	mu                  sync.Mutex
	acls                map[string]*ACL
	users               map[string]*User
	subnetGroups        map[string]*SubnetGroup
	clusters            map[string]*Cluster
	parameterGroups     map[string]*ParameterGroup
	snapshots           map[string]*Snapshot
	multiRegionClusters map[string]*MultiRegionCluster
	reservedNodes       map[string]*ReservedNode
	tags                map[string]map[string]string
}

func NewService() *Service {
	return &Service{
		acls:                map[string]*ACL{},
		users:               map[string]*User{},
		subnetGroups:        map[string]*SubnetGroup{},
		clusters:            map[string]*Cluster{},
		parameterGroups:     map[string]*ParameterGroup{},
		snapshots:           map[string]*Snapshot{},
		multiRegionClusters: map[string]*MultiRegionCluster{},
		reservedNodes:       map[string]*ReservedNode{},
		tags:                map[string]map[string]string{},
	}
}

func (s *Service) CreateACL(name string, userNames []string, tags map[string]string) (ACL, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return ACL{}, fault("InvalidParameterValueException", "ACLName is required")
	}
	if _, exists := s.acls[name]; exists {
		return ACL{}, fault("ACLAlreadyExistsFault", "acl already exists")
	}

	acl := &ACL{
		Name:                 name,
		Status:               "active",
		UserNames:            dedupeStrings(userNames),
		MinimumEngineVersion: "6.2",
		Clusters:             []string{},
		ARN:                  aclARN(name),
	}
	s.acls[name] = acl
	s.tags[acl.ARN] = cloneStringMap(tags)
	for _, userName := range acl.UserNames {
		if user := s.users[userName]; user != nil {
			user.ACLNames = addUnique(user.ACLNames, acl.Name)
		}
	}
	return cloneACL(*acl), nil
}

func (s *Service) DescribeACLs(name, nextToken string, maxResults int) ([]ACL, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]ACL, 0)
	name = strings.TrimSpace(name)
	if name != "" {
		if acl := s.acls[name]; acl != nil {
			items = append(items, cloneACL(*acl))
		}
		return items, "", nil
	}

	names := make([]string, 0, len(s.acls))
	for key := range s.acls {
		names = append(names, key)
	}
	sort.Strings(names)
	for _, key := range names {
		items = append(items, cloneACL(*s.acls[key]))
	}
	return paginateSlice(items, nextToken, maxResults)
}

func (s *Service) UpdateACL(name string, userNamesToAdd, userNamesToRemove []string) (ACL, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return ACL{}, fault("InvalidParameterValueException", "ACLName is required")
	}

	acl := s.ensureACLLocked(name)
	removeSet := setOf(userNamesToRemove)
	kept := make([]string, 0, len(acl.UserNames))
	for _, userName := range acl.UserNames {
		if _, drop := removeSet[userName]; !drop {
			kept = append(kept, userName)
		}
	}
	for _, userName := range userNamesToAdd {
		kept = addUnique(kept, userName)
		if user := s.users[userName]; user != nil {
			user.ACLNames = addUnique(user.ACLNames, acl.Name)
		}
	}
	acl.UserNames = kept
	return cloneACL(*acl), nil
}

func (s *Service) DeleteACL(name string) (ACL, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return ACL{}, fault("InvalidParameterValueException", "ACLName is required")
	}

	if acl := s.acls[name]; acl != nil {
		deleted := cloneACL(*acl)
		for _, user := range s.users {
			user.ACLNames = removeValue(user.ACLNames, name)
		}
		delete(s.tags, acl.ARN)
		delete(s.acls, name)
		deleted.Status = "deleting"
		return deleted, nil
	}

	return ACL{Name: name, Status: "deleting", ARN: aclARN(name)}, nil
}

func (s *Service) CreateUser(input CreateUserInput) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.UserName)
	if name == "" {
		return User{}, fault("InvalidParameterValueException", "UserName is required")
	}
	if strings.TrimSpace(input.AccessString) == "" {
		return User{}, fault("InvalidParameterValueException", "AccessString is required")
	}
	if strings.TrimSpace(input.AuthenticationType) == "" {
		return User{}, fault("InvalidParameterValueException", "AuthenticationMode.Type is required")
	}
	if _, exists := s.users[name]; exists {
		return User{}, fault("UserAlreadyExistsFault", "user already exists")
	}

	user := &User{
		Name:                 name,
		Status:               "active",
		AccessString:         strings.TrimSpace(input.AccessString),
		ACLNames:             []string{},
		MinimumEngineVersion: "6.2",
		AuthenticationType:   strings.TrimSpace(input.AuthenticationType),
		PasswordCount:        maxInt(input.PasswordCount, 1),
		ARN:                  userARN(name),
	}
	s.users[name] = user
	s.tags[user.ARN] = cloneStringMap(input.Tags)
	return cloneUser(*user), nil
}

func (s *Service) DescribeUsers(name, nextToken string, maxResults int) ([]User, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]User, 0)
	name = strings.TrimSpace(name)
	if name != "" {
		if user := s.users[name]; user != nil {
			items = append(items, cloneUser(*user))
		}
		return items, "", nil
	}

	names := make([]string, 0, len(s.users))
	for key := range s.users {
		names = append(names, key)
	}
	sort.Strings(names)
	for _, key := range names {
		items = append(items, cloneUser(*s.users[key]))
	}
	return paginateSlice(items, nextToken, maxResults)
}

func (s *Service) UpdateUser(input UpdateUserInput) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.UserName)
	if name == "" {
		return User{}, fault("InvalidParameterValueException", "UserName is required")
	}

	user := s.users[name]
	if user == nil {
		user = &User{
			Name:                 name,
			Status:               "active",
			AccessString:         "on ~* +@all",
			ACLNames:             []string{},
			MinimumEngineVersion: "6.2",
			AuthenticationType:   "password",
			PasswordCount:        1,
			ARN:                  userARN(name),
		}
		s.users[name] = user
	}

	if input.AccessString != nil {
		if strings.TrimSpace(*input.AccessString) != "" {
			user.AccessString = strings.TrimSpace(*input.AccessString)
		}
	}
	if input.AuthenticationType != nil {
		if strings.TrimSpace(*input.AuthenticationType) != "" {
			user.AuthenticationType = strings.TrimSpace(*input.AuthenticationType)
		}
	}
	if input.PasswordCount != nil {
		user.PasswordCount = maxInt(*input.PasswordCount, 1)
	}

	return cloneUser(*user), nil
}

func (s *Service) DeleteUser(name string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return User{}, fault("InvalidParameterValueException", "UserName is required")
	}

	if user := s.users[name]; user != nil {
		deleted := cloneUser(*user)
		delete(s.tags, user.ARN)
		delete(s.users, name)
		deleted.Status = "deleting"
		return deleted, nil
	}

	return User{Name: name, Status: "deleting", ARN: userARN(name)}, nil
}

func (s *Service) CreateSubnetGroup(name, description string, subnetIDs []string, tags map[string]string) (SubnetGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return SubnetGroup{}, fault("InvalidParameterValueException", "SubnetGroupName is required")
	}
	if len(subnetIDs) == 0 {
		return SubnetGroup{}, fault("InvalidParameterValueException", "SubnetIds is required")
	}
	if _, exists := s.subnetGroups[name]; exists {
		return SubnetGroup{}, fault("SubnetGroupAlreadyExistsFault", "subnet group already exists")
	}

	group := &SubnetGroup{
		Name:        name,
		Description: strings.TrimSpace(description),
		VpcID:       "vpc-00000001",
		SubnetIDs:   dedupeStrings(subnetIDs),
		ARN:         subnetGroupARN(name),
	}
	if group.Description == "" {
		group.Description = name
	}
	s.subnetGroups[name] = group
	s.tags[group.ARN] = cloneStringMap(tags)
	return cloneSubnetGroup(*group), nil
}

func (s *Service) DescribeSubnetGroups(name, nextToken string, maxResults int) ([]SubnetGroup, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]SubnetGroup, 0)
	name = strings.TrimSpace(name)
	if name != "" {
		if group := s.subnetGroups[name]; group != nil {
			items = append(items, cloneSubnetGroup(*group))
		}
		return items, "", nil
	}

	names := make([]string, 0, len(s.subnetGroups))
	for key := range s.subnetGroups {
		names = append(names, key)
	}
	sort.Strings(names)
	for _, key := range names {
		items = append(items, cloneSubnetGroup(*s.subnetGroups[key]))
	}
	return paginateSlice(items, nextToken, maxResults)
}

func (s *Service) UpdateSubnetGroup(name, description string, subnetIDs []string) (SubnetGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return SubnetGroup{}, fault("InvalidParameterValueException", "SubnetGroupName is required")
	}

	group := s.subnetGroups[name]
	if group == nil {
		group = &SubnetGroup{
			Name:      name,
			VpcID:     "vpc-00000001",
			SubnetIDs: []string{"subnet-00000001"},
			ARN:       subnetGroupARN(name),
		}
		s.subnetGroups[name] = group
	}
	if strings.TrimSpace(description) != "" {
		group.Description = strings.TrimSpace(description)
	}
	if len(subnetIDs) > 0 {
		group.SubnetIDs = dedupeStrings(subnetIDs)
	}
	if strings.TrimSpace(group.Description) == "" {
		group.Description = name
	}

	return cloneSubnetGroup(*group), nil
}

func (s *Service) DeleteSubnetGroup(name string) (SubnetGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return SubnetGroup{}, fault("InvalidParameterValueException", "SubnetGroupName is required")
	}

	if group := s.subnetGroups[name]; group != nil {
		deleted := cloneSubnetGroup(*group)
		delete(s.tags, group.ARN)
		delete(s.subnetGroups, name)
		return deleted, nil
	}

	return SubnetGroup{Name: name, VpcID: "vpc-00000001", SubnetIDs: []string{}, ARN: subnetGroupARN(name)}, nil
}

func (s *Service) CreateCluster(input CreateClusterInput) (Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.ClusterName)
	if name == "" {
		return Cluster{}, fault("InvalidParameterValueException", "ClusterName is required")
	}
	if strings.TrimSpace(input.NodeType) == "" {
		return Cluster{}, fault("InvalidParameterValueException", "NodeType is required")
	}
	if strings.TrimSpace(input.ACLName) == "" {
		return Cluster{}, fault("InvalidParameterValueException", "ACLName is required")
	}
	if _, exists := s.clusters[name]; exists {
		return Cluster{}, fault("ClusterAlreadyExistsFault", "cluster already exists")
	}

	cluster := s.newClusterLocked(name)
	cluster.NodeType = strings.TrimSpace(input.NodeType)
	cluster.ACLName = strings.TrimSpace(input.ACLName)
	if strings.TrimSpace(input.Description) != "" {
		cluster.Description = strings.TrimSpace(input.Description)
	}
	if strings.TrimSpace(input.MultiRegionClusterName) != "" {
		cluster.MultiRegionClusterName = strings.TrimSpace(input.MultiRegionClusterName)
	}
	if strings.TrimSpace(input.ParameterGroupName) != "" {
		cluster.ParameterGroupName = strings.TrimSpace(input.ParameterGroupName)
	}
	if input.NumShards > 0 {
		cluster.NumberOfShards = input.NumShards
	}
	if strings.TrimSpace(input.SubnetGroupName) != "" {
		cluster.SubnetGroupName = strings.TrimSpace(input.SubnetGroupName)
	}
	if len(input.SecurityGroupIDs) > 0 {
		cluster.SecurityGroupIDs = dedupeStrings(input.SecurityGroupIDs)
	}
	if strings.TrimSpace(input.MaintenanceWindow) != "" {
		cluster.MaintenanceWindow = strings.TrimSpace(input.MaintenanceWindow)
	}
	if input.Port > 0 {
		cluster.Port = input.Port
	}
	if strings.TrimSpace(input.SnsTopicArn) != "" {
		cluster.SnsTopicArn = strings.TrimSpace(input.SnsTopicArn)
	}
	if input.TLSEnabled != nil {
		cluster.TLSEnabled = *input.TLSEnabled
	}
	if strings.TrimSpace(input.KmsKeyID) != "" {
		cluster.KmsKeyID = strings.TrimSpace(input.KmsKeyID)
	}
	if input.SnapshotRetentionLimit > 0 {
		cluster.SnapshotRetentionLimit = input.SnapshotRetentionLimit
	}
	if strings.TrimSpace(input.SnapshotWindow) != "" {
		cluster.SnapshotWindow = strings.TrimSpace(input.SnapshotWindow)
	}
	if strings.TrimSpace(input.Engine) != "" {
		cluster.Engine = strings.TrimSpace(input.Engine)
	}
	if strings.TrimSpace(input.EngineVersion) != "" {
		cluster.EngineVersion = strings.TrimSpace(input.EngineVersion)
	}
	if input.AutoMinorVersionUpgrade != nil {
		cluster.AutoMinorVersionUpgrade = *input.AutoMinorVersionUpgrade
	}
	if input.DataTiering != nil {
		cluster.DataTiering = boolToDataTiering(*input.DataTiering)
	}
	cluster.Status = "available"
	s.addClusterToACLLocked(cluster.ACLName, cluster.Name)
	if input.Tags != nil {
		s.tags[cluster.ARN] = cloneStringMap(input.Tags)
	}

	return cloneCluster(*cluster), nil
}

func (s *Service) DescribeClusters(name, nextToken string, maxResults int) ([]Cluster, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]Cluster, 0)
	name = strings.TrimSpace(name)
	if name != "" {
		if cluster := s.clusters[name]; cluster != nil {
			items = append(items, cloneCluster(*cluster))
		}
		return items, "", nil
	}

	names := make([]string, 0, len(s.clusters))
	for key := range s.clusters {
		names = append(names, key)
	}
	sort.Strings(names)
	for _, key := range names {
		items = append(items, cloneCluster(*s.clusters[key]))
	}
	return paginateSlice(items, nextToken, maxResults)
}

func (s *Service) UpdateCluster(input UpdateClusterInput) (Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.ClusterName)
	if name == "" {
		return Cluster{}, fault("InvalidParameterValueException", "ClusterName is required")
	}
	cluster := s.ensureClusterLocked(name)

	if input.Description != nil && strings.TrimSpace(*input.Description) != "" {
		cluster.Description = strings.TrimSpace(*input.Description)
	}
	if input.SecurityGroupIDsSet {
		cluster.SecurityGroupIDs = dedupeStrings(input.SecurityGroupIDs)
	}
	if input.MaintenanceWindow != nil && strings.TrimSpace(*input.MaintenanceWindow) != "" {
		cluster.MaintenanceWindow = strings.TrimSpace(*input.MaintenanceWindow)
	}
	if input.SnsTopicArn != nil && strings.TrimSpace(*input.SnsTopicArn) != "" {
		cluster.SnsTopicArn = strings.TrimSpace(*input.SnsTopicArn)
	}
	if input.SnsTopicStatus != nil && strings.TrimSpace(*input.SnsTopicStatus) != "" {
		cluster.SnsTopicStatus = strings.TrimSpace(*input.SnsTopicStatus)
	}
	if input.ParameterGroupName != nil && strings.TrimSpace(*input.ParameterGroupName) != "" {
		cluster.ParameterGroupName = strings.TrimSpace(*input.ParameterGroupName)
	}
	if input.SnapshotWindow != nil && strings.TrimSpace(*input.SnapshotWindow) != "" {
		cluster.SnapshotWindow = strings.TrimSpace(*input.SnapshotWindow)
	}
	if input.SnapshotRetentionLimit != nil && *input.SnapshotRetentionLimit >= 0 {
		cluster.SnapshotRetentionLimit = *input.SnapshotRetentionLimit
	}
	if input.NodeType != nil && strings.TrimSpace(*input.NodeType) != "" {
		cluster.NodeType = strings.TrimSpace(*input.NodeType)
	}
	if input.Engine != nil && strings.TrimSpace(*input.Engine) != "" {
		cluster.Engine = strings.TrimSpace(*input.Engine)
	}
	if input.EngineVersion != nil && strings.TrimSpace(*input.EngineVersion) != "" {
		cluster.EngineVersion = strings.TrimSpace(*input.EngineVersion)
	}
	if input.ShardCount != nil && *input.ShardCount > 0 {
		cluster.NumberOfShards = *input.ShardCount
	}
	if input.ACLName != nil && strings.TrimSpace(*input.ACLName) != "" {
		s.removeClusterFromACLLocked(cluster.ACLName, cluster.Name)
		cluster.ACLName = strings.TrimSpace(*input.ACLName)
		s.addClusterToACLLocked(cluster.ACLName, cluster.Name)
	}
	cluster.Status = "modifying"
	return cloneCluster(*cluster), nil
}

func (s *Service) DeleteCluster(name string) (Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return Cluster{}, fault("InvalidParameterValueException", "ClusterName is required")
	}
	if cluster := s.clusters[name]; cluster != nil {
		deleted := cloneCluster(*cluster)
		s.removeClusterFromACLLocked(cluster.ACLName, cluster.Name)
		delete(s.tags, cluster.ARN)
		delete(s.clusters, name)
		deleted.Status = "deleting"
		return deleted, nil
	}

	cluster := s.newClusterLocked(name)
	cluster.Status = "deleting"
	delete(s.clusters, name)
	return cloneCluster(*cluster), nil
}

func (s *Service) CreateParameterGroup(input CreateParameterGroupInput) (ParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return ParameterGroup{}, fault("InvalidParameterValueException", "ParameterGroupName is required")
	}
	family := strings.TrimSpace(input.Family)
	if family == "" {
		return ParameterGroup{}, fault("InvalidParameterValueException", "Family is required")
	}
	if _, exists := s.parameterGroups[name]; exists {
		return ParameterGroup{}, fault("ParameterGroupAlreadyExistsFault", "parameter group already exists")
	}

	pg := &ParameterGroup{
		Name:        name,
		Family:      family,
		Description: strings.TrimSpace(input.Description),
		Parameters: map[string]string{
			"activedefrag":    "yes",
			"cluster-enabled": "yes",
		},
		ARN: parameterGroupARN(name),
	}
	if pg.Description == "" {
		pg.Description = name
	}
	s.parameterGroups[name] = pg
	s.tags[pg.ARN] = cloneStringMap(input.Tags)
	return cloneParameterGroup(*pg), nil
}

func (s *Service) DescribeParameterGroups(name, nextToken string, maxResults int) ([]ParameterGroup, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]ParameterGroup, 0)
	name = strings.TrimSpace(name)
	if name != "" {
		if pg := s.parameterGroups[name]; pg != nil {
			items = append(items, cloneParameterGroup(*pg))
		}
		return items, "", nil
	}

	names := make([]string, 0, len(s.parameterGroups))
	for key := range s.parameterGroups {
		names = append(names, key)
	}
	sort.Strings(names)
	for _, key := range names {
		items = append(items, cloneParameterGroup(*s.parameterGroups[key]))
	}
	return paginateSlice(items, nextToken, maxResults)
}

func (s *Service) UpdateParameterGroup(name string, values map[string]string) (ParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return ParameterGroup{}, fault("InvalidParameterValueException", "ParameterGroupName is required")
	}
	pg := s.parameterGroups[name]
	if pg == nil {
		pg = &ParameterGroup{
			Name:        name,
			Family:      "memorydb_redis7",
			Description: name,
			Parameters:  map[string]string{},
			ARN:         parameterGroupARN(name),
		}
		s.parameterGroups[name] = pg
	}
	for key, value := range values {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		pg.Parameters[k] = strings.TrimSpace(value)
	}
	return cloneParameterGroup(*pg), nil
}

func (s *Service) DeleteParameterGroup(name string) (ParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return ParameterGroup{}, fault("InvalidParameterValueException", "ParameterGroupName is required")
	}

	if pg := s.parameterGroups[name]; pg != nil {
		deleted := cloneParameterGroup(*pg)
		delete(s.tags, pg.ARN)
		delete(s.parameterGroups, name)
		return deleted, nil
	}
	return ParameterGroup{Name: name, Family: "memorydb_redis7", Description: name, Parameters: map[string]string{}, ARN: parameterGroupARN(name)}, nil
}

func (s *Service) DescribeParameters(parameterGroupName, nextToken string, maxResults int) ([]Parameter, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(parameterGroupName)
	if name == "" {
		return nil, "", fault("InvalidParameterValueException", "ParameterGroupName is required")
	}

	pg := s.parameterGroups[name]
	if pg == nil {
		pg = &ParameterGroup{
			Name:        name,
			Family:      "memorydb_redis7",
			Description: name,
			Parameters:  map[string]string{},
			ARN:         parameterGroupARN(name),
		}
		s.parameterGroups[name] = pg
	}

	keys := make([]string, 0, len(pg.Parameters))
	for key := range pg.Parameters {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]Parameter, 0, len(keys))
	for _, key := range keys {
		items = append(items, Parameter{
			Name:                 key,
			Value:                pg.Parameters[key],
			Description:          "",
			DataType:             "string",
			AllowedValues:        "",
			MinimumEngineVersion: "6.2",
		})
	}

	return paginateSlice(items, nextToken, maxResults)
}

func (s *Service) ResetParameterGroup(name string, allParameters bool, parameterNames []string) (ParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return ParameterGroup{}, fault("InvalidParameterValueException", "ParameterGroupName is required")
	}
	pg := s.parameterGroups[name]
	if pg == nil {
		pg = &ParameterGroup{
			Name:        name,
			Family:      "memorydb_redis7",
			Description: name,
			Parameters:  map[string]string{},
			ARN:         parameterGroupARN(name),
		}
		s.parameterGroups[name] = pg
	}

	if allParameters {
		pg.Parameters = map[string]string{}
	} else {
		for _, raw := range parameterNames {
			delete(pg.Parameters, strings.TrimSpace(raw))
		}
	}

	return cloneParameterGroup(*pg), nil
}

func (s *Service) CreateSnapshot(input CreateSnapshotInput) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterName := strings.TrimSpace(input.ClusterName)
	snapshotName := strings.TrimSpace(input.SnapshotName)
	if clusterName == "" {
		return Snapshot{}, fault("InvalidParameterValueException", "ClusterName is required")
	}
	if snapshotName == "" {
		return Snapshot{}, fault("InvalidParameterValueException", "SnapshotName is required")
	}
	if _, exists := s.snapshots[snapshotName]; exists {
		return Snapshot{}, fault("SnapshotAlreadyExistsFault", "snapshot already exists")
	}
	if s.clusters[clusterName] == nil {
		s.newClusterLocked(clusterName)
	}

	snapshot := &Snapshot{
		Name:        snapshotName,
		Status:      "available",
		Source:      "manual",
		KmsKeyID:    strings.TrimSpace(input.KmsKeyID),
		ARN:         snapshotARN(snapshotName),
		ClusterName: clusterName,
		DataTiering: s.clusters[clusterName].DataTiering,
	}
	s.snapshots[snapshotName] = snapshot
	s.tags[snapshot.ARN] = cloneStringMap(input.Tags)
	return cloneSnapshot(*snapshot), nil
}

func (s *Service) CopySnapshot(input CopySnapshotInput) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceName := strings.TrimSpace(input.SourceSnapshotName)
	targetName := strings.TrimSpace(input.TargetSnapshotName)
	if sourceName == "" {
		return Snapshot{}, fault("InvalidParameterValueException", "SourceSnapshotName is required")
	}
	if targetName == "" {
		return Snapshot{}, fault("InvalidParameterValueException", "TargetSnapshotName is required")
	}

	source := s.snapshots[sourceName]
	if source == nil {
		source = &Snapshot{
			Name:        sourceName,
			Status:      "available",
			Source:      "system",
			KmsKeyID:    "",
			ARN:         snapshotARN(sourceName),
			ClusterName: "stackyard-cluster",
			DataTiering: "false",
		}
		s.snapshots[sourceName] = source
	}

	copied := &Snapshot{
		Name:        targetName,
		Status:      "available",
		Source:      sourceName,
		KmsKeyID:    firstNonEmpty(strings.TrimSpace(input.KmsKeyID), source.KmsKeyID),
		ARN:         snapshotARN(targetName),
		ClusterName: source.ClusterName,
		DataTiering: source.DataTiering,
	}
	s.snapshots[targetName] = copied
	s.tags[copied.ARN] = cloneStringMap(input.Tags)
	return cloneSnapshot(*copied), nil
}

func (s *Service) DescribeSnapshots(clusterName, snapshotName, source, nextToken string, maxResults int) ([]Snapshot, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterName = strings.TrimSpace(clusterName)
	snapshotName = strings.TrimSpace(snapshotName)
	source = strings.TrimSpace(source)

	names := make([]string, 0, len(s.snapshots))
	for key := range s.snapshots {
		names = append(names, key)
	}
	sort.Strings(names)

	items := make([]Snapshot, 0)
	for _, key := range names {
		snap := s.snapshots[key]
		if snap == nil {
			continue
		}
		if snapshotName != "" && snap.Name != snapshotName {
			continue
		}
		if clusterName != "" && snap.ClusterName != clusterName {
			continue
		}
		if source != "" && !strings.EqualFold(snap.Source, source) {
			continue
		}
		items = append(items, cloneSnapshot(*snap))
	}

	return paginateSlice(items, nextToken, maxResults)
}

func (s *Service) DeleteSnapshot(name string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return Snapshot{}, fault("InvalidParameterValueException", "SnapshotName is required")
	}
	if snapshot := s.snapshots[name]; snapshot != nil {
		deleted := cloneSnapshot(*snapshot)
		delete(s.tags, snapshot.ARN)
		delete(s.snapshots, name)
		deleted.Status = "deleting"
		return deleted, nil
	}

	return Snapshot{Name: name, Status: "deleting", Source: "manual", ARN: snapshotARN(name), DataTiering: "false"}, nil
}

func (s *Service) CreateMultiRegionCluster(input CreateMultiRegionClusterInput) (MultiRegionCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.NameSuffix)
	if name == "" {
		return MultiRegionCluster{}, fault("InvalidParameterValueException", "MultiRegionClusterNameSuffix is required")
	}
	if strings.TrimSpace(input.NodeType) == "" {
		return MultiRegionCluster{}, fault("InvalidParameterValueException", "NodeType is required")
	}
	if _, exists := s.multiRegionClusters[name]; exists {
		return MultiRegionCluster{}, fault("MultiRegionClusterAlreadyExistsFault", "multi-region cluster already exists")
	}

	mrc := &MultiRegionCluster{
		Name:           name,
		Description:    strings.TrimSpace(input.Description),
		Status:         "available",
		NodeType:       strings.TrimSpace(input.NodeType),
		Engine:         firstNonEmpty(strings.TrimSpace(input.Engine), "redis"),
		EngineVersion:  firstNonEmpty(strings.TrimSpace(input.EngineVersion), "7.1"),
		NumberOfShards: maxInt(input.NumShards, 1),
		Clusters: []RegionalCluster{{
			ClusterName: name + "-use1",
			Region:      defaultRegion,
			Status:      "available",
			ARN:         clusterARN(name + "-use1"),
		}},
		MultiRegionParameterGroupName: strings.TrimSpace(input.MultiRegionParameterGroupName),
		TLSEnabled:                    true,
		ARN:                           multiRegionClusterARN(name),
	}
	if mrc.Description == "" {
		mrc.Description = name
	}
	if input.TLSEnabled != nil {
		mrc.TLSEnabled = *input.TLSEnabled
	}

	s.multiRegionClusters[name] = mrc
	s.tags[mrc.ARN] = cloneStringMap(input.Tags)
	return cloneMultiRegionCluster(*mrc), nil
}

func (s *Service) DescribeMultiRegionClusters(name, nextToken string, maxResults int) ([]MultiRegionCluster, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]MultiRegionCluster, 0)
	name = strings.TrimSpace(name)
	if name != "" {
		if entry := s.multiRegionClusters[name]; entry != nil {
			items = append(items, cloneMultiRegionCluster(*entry))
		}
		return items, "", nil
	}

	names := make([]string, 0, len(s.multiRegionClusters))
	for key := range s.multiRegionClusters {
		names = append(names, key)
	}
	sort.Strings(names)
	for _, key := range names {
		items = append(items, cloneMultiRegionCluster(*s.multiRegionClusters[key]))
	}
	return paginateSlice(items, nextToken, maxResults)
}

func (s *Service) UpdateMultiRegionCluster(input UpdateMultiRegionClusterInput) (MultiRegionCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return MultiRegionCluster{}, fault("InvalidParameterValueException", "MultiRegionClusterName is required")
	}

	entry := s.multiRegionClusters[name]
	if entry == nil {
		entry = &MultiRegionCluster{
			Name:                          name,
			Description:                   name,
			Status:                        "available",
			NodeType:                      "db.r6g.large",
			Engine:                        "redis",
			EngineVersion:                 "7.1",
			NumberOfShards:                1,
			Clusters:                      []RegionalCluster{},
			MultiRegionParameterGroupName: "default.memorydb-redis7",
			TLSEnabled:                    true,
			ARN:                           multiRegionClusterARN(name),
		}
		s.multiRegionClusters[name] = entry
	}
	if input.NodeType != nil && strings.TrimSpace(*input.NodeType) != "" {
		entry.NodeType = strings.TrimSpace(*input.NodeType)
	}
	if input.Description != nil && strings.TrimSpace(*input.Description) != "" {
		entry.Description = strings.TrimSpace(*input.Description)
	}
	if input.EngineVersion != nil && strings.TrimSpace(*input.EngineVersion) != "" {
		entry.EngineVersion = strings.TrimSpace(*input.EngineVersion)
	}
	if input.ShardCount != nil && *input.ShardCount > 0 {
		entry.NumberOfShards = *input.ShardCount
	}
	if input.MultiRegionParameterGroupName != nil && strings.TrimSpace(*input.MultiRegionParameterGroupName) != "" {
		entry.MultiRegionParameterGroupName = strings.TrimSpace(*input.MultiRegionParameterGroupName)
	}
	entry.Status = "modifying"

	return cloneMultiRegionCluster(*entry), nil
}

func (s *Service) DeleteMultiRegionCluster(name string) (MultiRegionCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		return MultiRegionCluster{}, fault("InvalidParameterValueException", "MultiRegionClusterName is required")
	}

	if entry := s.multiRegionClusters[name]; entry != nil {
		deleted := cloneMultiRegionCluster(*entry)
		delete(s.tags, entry.ARN)
		delete(s.multiRegionClusters, name)
		deleted.Status = "deleting"
		return deleted, nil
	}

	return MultiRegionCluster{Name: name, Status: "deleting", ARN: multiRegionClusterARN(name)}, nil
}

func (s *Service) ListAllowedMultiRegionClusterUpdates(_ string) ([]string, []string, error) {
	return []string{"db.r6g.xlarge", "db.r7g.xlarge"}, []string{"db.r6g.large"}, nil
}

func (s *Service) BatchUpdateCluster(clusterNames []string, _ string) ([]Cluster, []UnprocessedCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(clusterNames) == 0 {
		return nil, nil, fault("InvalidParameterValueException", "ClusterNames is required")
	}

	out := make([]Cluster, 0, len(clusterNames))
	for _, raw := range clusterNames {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		cluster := s.ensureClusterLocked(name)
		cluster.Status = "modifying"
		out = append(out, cloneCluster(*cluster))
	}
	if len(out) == 0 {
		return nil, nil, fault("InvalidParameterValueException", "ClusterNames is required")
	}
	return out, []UnprocessedCluster{}, nil
}

func (s *Service) FailoverShard(clusterName, shardName string) (Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterName = strings.TrimSpace(clusterName)
	shardName = strings.TrimSpace(shardName)
	if clusterName == "" {
		return Cluster{}, fault("InvalidParameterValueException", "ClusterName is required")
	}
	if shardName == "" {
		return Cluster{}, fault("InvalidParameterValueException", "ShardName is required")
	}

	cluster := s.ensureClusterLocked(clusterName)
	cluster.Status = "failing-over"
	return cloneCluster(*cluster), nil
}

func (s *Service) DescribeEngineVersions(engine, engineVersion, parameterGroupFamily, nextToken string, maxResults int, defaultOnly bool) ([]EngineVersion, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := []EngineVersion{
		{
			Engine:               "redis",
			EngineVersion:        "7.1",
			EnginePatchVersion:   "7.1.0",
			ParameterGroupFamily: "memorydb_redis7",
			Default:              true,
		},
		{
			Engine:               "redis",
			EngineVersion:        "6.2",
			EnginePatchVersion:   "6.2.6",
			ParameterGroupFamily: "memorydb_redis6",
			Default:              false,
		},
	}

	engine = strings.TrimSpace(engine)
	engineVersion = strings.TrimSpace(engineVersion)
	parameterGroupFamily = strings.TrimSpace(parameterGroupFamily)

	filtered := make([]EngineVersion, 0, len(items))
	for _, entry := range items {
		if engine != "" && !strings.EqualFold(entry.Engine, engine) {
			continue
		}
		if engineVersion != "" && !strings.EqualFold(entry.EngineVersion, engineVersion) {
			continue
		}
		if parameterGroupFamily != "" && !strings.EqualFold(entry.ParameterGroupFamily, parameterGroupFamily) {
			continue
		}
		if defaultOnly && !entry.Default {
			continue
		}
		filtered = append(filtered, entry)
	}

	return paginateSlice(filtered, nextToken, maxResults)
}

func (s *Service) ListAllowedNodeTypeUpdates(clusterName string) ([]string, []string, error) {
	clusterName = strings.TrimSpace(clusterName)
	if clusterName == "" {
		return nil, nil, fault("InvalidParameterValueException", "ClusterName is required")
	}
	return []string{"db.r6g.xlarge", "db.r7g.xlarge"}, []string{"db.r6g.large", "db.t4g.small"}, nil
}

func (s *Service) DescribeReservedNodes(reservationID, offeringID, nodeType, duration, offeringType, nextToken string, maxResults int) ([]ReservedNode, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reservationID = strings.TrimSpace(reservationID)
	offeringID = strings.TrimSpace(offeringID)
	nodeType = strings.TrimSpace(nodeType)
	duration = strings.TrimSpace(duration)
	offeringType = strings.TrimSpace(offeringType)

	keys := make([]string, 0, len(s.reservedNodes))
	for key := range s.reservedNodes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]ReservedNode, 0, len(keys))
	for _, key := range keys {
		entry := s.reservedNodes[key]
		if entry == nil {
			continue
		}
		if reservationID != "" && !strings.EqualFold(entry.ReservationID, reservationID) {
			continue
		}
		if offeringID != "" && !strings.EqualFold(entry.ReservedNodesOfferingID, offeringID) {
			continue
		}
		if nodeType != "" && !strings.EqualFold(entry.NodeType, nodeType) {
			continue
		}
		if duration != "" && !durationMatches(duration, entry.Duration) {
			continue
		}
		if offeringType != "" && !strings.EqualFold(entry.OfferingType, offeringType) {
			continue
		}
		items = append(items, cloneReservedNode(*entry))
	}

	return paginateSlice(items, nextToken, maxResults)
}

func (s *Service) DescribeReservedNodesOfferings(offeringID, nodeType, duration, offeringType, nextToken string, maxResults int) ([]ReservedNodeOffering, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	offeringID = strings.TrimSpace(offeringID)
	nodeType = strings.TrimSpace(nodeType)
	duration = strings.TrimSpace(duration)
	offeringType = strings.TrimSpace(offeringType)

	all := defaultReservedNodeOfferings()
	items := make([]ReservedNodeOffering, 0, len(all))
	for _, entry := range all {
		if offeringID != "" && !strings.EqualFold(entry.ReservedNodesOfferingID, offeringID) {
			continue
		}
		if nodeType != "" && !strings.EqualFold(entry.NodeType, nodeType) {
			continue
		}
		if duration != "" && !durationMatches(duration, entry.Duration) {
			continue
		}
		if offeringType != "" && !strings.EqualFold(entry.OfferingType, offeringType) {
			continue
		}
		items = append(items, cloneReservedNodeOffering(entry))
	}
	return paginateSlice(items, nextToken, maxResults)
}

func (s *Service) PurchaseReservedNodesOffering(offeringID, reservationID string, nodeCount int, tags map[string]string) (ReservedNode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	offeringID = strings.TrimSpace(offeringID)
	if offeringID == "" {
		return ReservedNode{}, fault("InvalidParameterValueException", "ReservedNodesOfferingId is required")
	}
	reservationID = strings.TrimSpace(reservationID)
	if reservationID == "" {
		reservationID = fmt.Sprintf("reserved-node-%06d", len(s.reservedNodes)+1)
	}
	if existing := s.reservedNodes[reservationID]; existing != nil {
		return cloneReservedNode(*existing), nil
	}
	if nodeCount <= 0 {
		nodeCount = 1
	}

	offering := defaultReservedNodeOfferingByID(offeringID)
	now := time.Now().UTC()
	entry := &ReservedNode{
		ReservationID:           reservationID,
		ReservedNodesOfferingID: offering.ReservedNodesOfferingID,
		NodeType:                offering.NodeType,
		StartTime:               now,
		Duration:                offering.Duration,
		FixedPrice:              offering.FixedPrice,
		NodeCount:               nodeCount,
		OfferingType:            offering.OfferingType,
		State:                   "active",
		RecurringCharges:        cloneRecurringCharges(offering.RecurringCharges),
		ARN:                     reservedNodeARN(reservationID),
	}
	s.reservedNodes[reservationID] = entry
	s.tags[entry.ARN] = cloneStringMap(tags)
	return cloneReservedNode(*entry), nil
}

func (s *Service) TagResource(resourceARN string, tags map[string]string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return nil, fault("InvalidParameterValueException", "ResourceArn is required")
	}
	current := cloneStringMap(s.tags[resourceARN])
	if current == nil {
		current = map[string]string{}
	}
	for key, value := range tags {
		tagKey := strings.TrimSpace(key)
		if tagKey == "" {
			continue
		}
		current[tagKey] = strings.TrimSpace(value)
	}
	s.tags[resourceARN] = current
	return cloneStringMap(current), nil
}

func (s *Service) UntagResource(resourceARN string, tagKeys []string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return nil, fault("InvalidParameterValueException", "ResourceArn is required")
	}
	current := cloneStringMap(s.tags[resourceARN])
	if current == nil {
		current = map[string]string{}
	}
	for _, raw := range tagKeys {
		tagKey := strings.TrimSpace(raw)
		if tagKey == "" {
			continue
		}
		delete(current, tagKey)
	}
	s.tags[resourceARN] = current
	return cloneStringMap(current), nil
}

func (s *Service) ListTags(resourceARN, nextToken string, maxResults int) (map[string]string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return nil, "", fault("InvalidParameterValueException", "ResourceArn is required")
	}
	all := cloneStringMap(s.tags[resourceARN])
	if all == nil {
		all = map[string]string{}
		s.tags[resourceARN] = all
	}
	keys := make([]string, 0, len(all))
	for key := range all {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	window, outToken, err := paginateSlice(keys, nextToken, maxResults)
	if err != nil {
		return nil, "", err
	}
	out := make(map[string]string, len(window))
	for _, key := range window {
		out[key] = all[key]
	}
	return out, outToken, nil
}

func (s *Service) DescribeEvents(sourceName, sourceType, startTimeRaw, endTimeRaw string, duration int, nextToken string, maxResults int) ([]Event, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceName = strings.TrimSpace(sourceName)
	sourceType = strings.TrimSpace(sourceType)
	if sourceType == "" {
		sourceType = "cluster"
	}

	now := time.Now().UTC()
	names := make([]string, 0, len(s.clusters))
	for name := range s.clusters {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]Event, 0, maxInt(len(names), 1))
	if len(names) == 0 {
		eventSource := firstNonEmpty(sourceName, "stackyard-cluster")
		items = append(items, Event{
			SourceName: eventSource,
			SourceType: sourceType,
			Message:    "MemoryDB event",
			Date:       now,
		})
	} else {
		for _, name := range names {
			items = append(items, Event{
				SourceName: name,
				SourceType: "cluster",
				Message:    fmt.Sprintf("Cluster %s is available", name),
				Date:       now,
			})
		}
	}

	filtered := make([]Event, 0, len(items))
	for _, item := range items {
		if sourceName != "" && !strings.EqualFold(item.SourceName, sourceName) {
			continue
		}
		if sourceType != "" && !strings.EqualFold(item.SourceType, sourceType) {
			continue
		}
		filtered = append(filtered, item)
	}

	start, hasStart := parseFlexibleTime(startTimeRaw)
	end, hasEnd := parseFlexibleTime(endTimeRaw)
	if duration > 0 && !hasStart && !hasEnd {
		start = now.Add(-time.Duration(duration) * time.Minute)
		hasStart = true
	}
	if hasStart || hasEnd {
		timeFiltered := make([]Event, 0, len(filtered))
		for _, item := range filtered {
			if hasStart && item.Date.Before(start) {
				continue
			}
			if hasEnd && item.Date.After(end) {
				continue
			}
			timeFiltered = append(timeFiltered, item)
		}
		filtered = timeFiltered
	}

	return paginateSlice(filtered, nextToken, maxResults)
}

func (s *Service) DescribeServiceUpdates(serviceUpdateName string, clusterNames, statuses []string, nextToken string, maxResults int) ([]ServiceUpdate, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	serviceUpdateName = strings.TrimSpace(serviceUpdateName)
	clusterNames = dedupeStrings(clusterNames)
	statuses = dedupeStrings(statuses)
	statusSet := setOf(statuses)

	targetClusters := make([]string, 0)
	if len(clusterNames) > 0 {
		targetClusters = append(targetClusters, clusterNames...)
	} else {
		for name := range s.clusters {
			targetClusters = append(targetClusters, name)
		}
		sort.Strings(targetClusters)
		if len(targetClusters) == 0 {
			targetClusters = append(targetClusters, "stackyard-cluster")
		}
	}

	releaseDate := time.Now().UTC().Add(-24 * time.Hour)
	autoUpdateStart := releaseDate.Add(7 * 24 * time.Hour)
	defaultName := "memorydb-update-2026-01"

	items := make([]ServiceUpdate, 0, len(targetClusters))
	for _, clusterName := range targetClusters {
		update := ServiceUpdate{
			ClusterName:         clusterName,
			ServiceUpdateName:   defaultName,
			ReleaseDate:         releaseDate,
			Description:         "Security and stability update",
			Status:              "available",
			Type:                "security-update",
			Engine:              "redis",
			NodesUpdated:        "0/1",
			AutoUpdateStartDate: autoUpdateStart,
		}
		if serviceUpdateName != "" && !strings.EqualFold(update.ServiceUpdateName, serviceUpdateName) {
			continue
		}
		if len(statusSet) > 0 {
			if _, ok := statusSet[update.Status]; !ok {
				lower := strings.ToLower(update.Status)
				if _, ok := statusSet[lower]; !ok {
					continue
				}
			}
		}
		items = append(items, update)
	}

	return paginateSlice(items, nextToken, maxResults)
}

func (s *Service) ensureACLLocked(name string) *ACL {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}
	if acl := s.acls[name]; acl != nil {
		return acl
	}
	acl := &ACL{
		Name:                 name,
		Status:               "active",
		UserNames:            []string{},
		MinimumEngineVersion: "6.2",
		Clusters:             []string{},
		ARN:                  aclARN(name),
	}
	s.acls[name] = acl
	return acl
}

func (s *Service) newClusterLocked(name string) *Cluster {
	cluster := &Cluster{
		Name:                    name,
		Description:             name,
		Status:                  "creating",
		MultiRegionClusterName:  "",
		NumberOfShards:          1,
		NodeType:                "db.r6g.large",
		Engine:                  "redis",
		EngineVersion:           "7.1",
		ParameterGroupName:      "default.memorydb-redis7",
		SecurityGroupIDs:        []string{"sg-00000001"},
		SubnetGroupName:         "default",
		TLSEnabled:              true,
		KmsKeyID:                "",
		ARN:                     clusterARN(name),
		SnsTopicArn:             "",
		SnsTopicStatus:          "active",
		SnapshotRetentionLimit:  0,
		MaintenanceWindow:       "sun:23:00-mon:01:30",
		SnapshotWindow:          "03:00-04:00",
		ACLName:                 "open-access",
		AutoMinorVersionUpgrade: true,
		DataTiering:             "false",
		Port:                    6379,
	}
	s.clusters[name] = cluster
	s.addClusterToACLLocked(cluster.ACLName, cluster.Name)
	return cluster
}

func (s *Service) ensureClusterLocked(name string) *Cluster {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "cluster"
	}
	if cluster := s.clusters[name]; cluster != nil {
		return cluster
	}
	return s.newClusterLocked(name)
}

func (s *Service) addClusterToACLLocked(aclName, clusterName string) {
	aclName = strings.TrimSpace(aclName)
	clusterName = strings.TrimSpace(clusterName)
	if aclName == "" || clusterName == "" {
		return
	}
	acl := s.ensureACLLocked(aclName)
	acl.Clusters = addUnique(acl.Clusters, clusterName)
}

func (s *Service) removeClusterFromACLLocked(aclName, clusterName string) {
	aclName = strings.TrimSpace(aclName)
	clusterName = strings.TrimSpace(clusterName)
	if acl := s.acls[aclName]; acl != nil {
		acl.Clusters = removeValue(acl.Clusters, clusterName)
	}
}

func cloneACL(in ACL) ACL {
	return ACL{
		Name:                 in.Name,
		Status:               in.Status,
		UserNames:            cloneStrings(in.UserNames),
		MinimumEngineVersion: in.MinimumEngineVersion,
		Clusters:             cloneStrings(in.Clusters),
		ARN:                  in.ARN,
	}
}

func cloneUser(in User) User {
	return User{
		Name:                 in.Name,
		Status:               in.Status,
		AccessString:         in.AccessString,
		ACLNames:             cloneStrings(in.ACLNames),
		MinimumEngineVersion: in.MinimumEngineVersion,
		AuthenticationType:   in.AuthenticationType,
		PasswordCount:        in.PasswordCount,
		ARN:                  in.ARN,
	}
}

func cloneSubnetGroup(in SubnetGroup) SubnetGroup {
	return SubnetGroup{
		Name:        in.Name,
		Description: in.Description,
		VpcID:       in.VpcID,
		SubnetIDs:   cloneStrings(in.SubnetIDs),
		ARN:         in.ARN,
	}
}

func cloneCluster(in Cluster) Cluster {
	return Cluster{
		Name:                    in.Name,
		Description:             in.Description,
		Status:                  in.Status,
		MultiRegionClusterName:  in.MultiRegionClusterName,
		NumberOfShards:          in.NumberOfShards,
		NodeType:                in.NodeType,
		Engine:                  in.Engine,
		EngineVersion:           in.EngineVersion,
		ParameterGroupName:      in.ParameterGroupName,
		SecurityGroupIDs:        cloneStrings(in.SecurityGroupIDs),
		SubnetGroupName:         in.SubnetGroupName,
		TLSEnabled:              in.TLSEnabled,
		KmsKeyID:                in.KmsKeyID,
		ARN:                     in.ARN,
		SnsTopicArn:             in.SnsTopicArn,
		SnsTopicStatus:          in.SnsTopicStatus,
		SnapshotRetentionLimit:  in.SnapshotRetentionLimit,
		MaintenanceWindow:       in.MaintenanceWindow,
		SnapshotWindow:          in.SnapshotWindow,
		ACLName:                 in.ACLName,
		AutoMinorVersionUpgrade: in.AutoMinorVersionUpgrade,
		DataTiering:             in.DataTiering,
		Port:                    in.Port,
	}
}

func cloneParameterGroup(in ParameterGroup) ParameterGroup {
	return ParameterGroup{
		Name:        in.Name,
		Family:      in.Family,
		Description: in.Description,
		Parameters:  cloneStringMap(in.Parameters),
		ARN:         in.ARN,
	}
}

func cloneSnapshot(in Snapshot) Snapshot {
	return Snapshot{
		Name:        in.Name,
		Status:      in.Status,
		Source:      in.Source,
		KmsKeyID:    in.KmsKeyID,
		ARN:         in.ARN,
		ClusterName: in.ClusterName,
		DataTiering: in.DataTiering,
	}
}

func cloneMultiRegionCluster(in MultiRegionCluster) MultiRegionCluster {
	clusters := make([]RegionalCluster, 0, len(in.Clusters))
	for _, c := range in.Clusters {
		clusters = append(clusters, c)
	}
	return MultiRegionCluster{
		Name:                          in.Name,
		Description:                   in.Description,
		Status:                        in.Status,
		NodeType:                      in.NodeType,
		Engine:                        in.Engine,
		EngineVersion:                 in.EngineVersion,
		NumberOfShards:                in.NumberOfShards,
		Clusters:                      clusters,
		MultiRegionParameterGroupName: in.MultiRegionParameterGroupName,
		TLSEnabled:                    in.TLSEnabled,
		ARN:                           in.ARN,
	}
}

func cloneRecurringCharges(values []RecurringCharge) []RecurringCharge {
	if len(values) == 0 {
		return []RecurringCharge{}
	}
	out := make([]RecurringCharge, len(values))
	copy(out, values)
	return out
}

func cloneReservedNode(in ReservedNode) ReservedNode {
	return ReservedNode{
		ReservationID:           in.ReservationID,
		ReservedNodesOfferingID: in.ReservedNodesOfferingID,
		NodeType:                in.NodeType,
		StartTime:               in.StartTime,
		Duration:                in.Duration,
		FixedPrice:              in.FixedPrice,
		NodeCount:               in.NodeCount,
		OfferingType:            in.OfferingType,
		State:                   in.State,
		RecurringCharges:        cloneRecurringCharges(in.RecurringCharges),
		ARN:                     in.ARN,
	}
}

func cloneReservedNodeOffering(in ReservedNodeOffering) ReservedNodeOffering {
	return ReservedNodeOffering{
		ReservedNodesOfferingID: in.ReservedNodesOfferingID,
		NodeType:                in.NodeType,
		Duration:                in.Duration,
		FixedPrice:              in.FixedPrice,
		OfferingType:            in.OfferingType,
		RecurringCharges:        cloneRecurringCharges(in.RecurringCharges),
	}
}

func paginateSlice[T any](items []T, nextToken string, maxResults int) ([]T, string, error) {
	start := 0
	nextToken = strings.TrimSpace(nextToken)
	if nextToken != "" {
		if isPlaceholderPaginationToken(nextToken) {
			nextToken = ""
		}
	}
	if nextToken != "" {
		offset, err := strconv.Atoi(nextToken)
		if err != nil || offset < 0 {
			return nil, "", fault("InvalidParameterValueException", "invalid NextToken")
		}
		start = offset
	}
	if maxResults <= 0 {
		maxResults = 100
	}
	if start > len(items) {
		return []T{}, "", nil
	}
	end := start + maxResults
	if end > len(items) {
		end = len(items)
	}
	out := make([]T, 0, end-start)
	out = append(out, items[start:end]...)
	if end < len(items) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func addUnique(list []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return list
	}
	for _, existing := range list {
		if existing == value {
			return list
		}
	}
	return append(list, value)
}

func removeValue(list []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return cloneStrings(list)
	}
	out := make([]string, 0, len(list))
	for _, existing := range list {
		if existing != value {
			out = append(out, existing)
		}
	}
	return out
}

func setOf(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out
}

func maxInt(value, floor int) int {
	if value < floor {
		return floor
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

func boolToDataTiering(enabled bool) string {
	if enabled {
		return "true"
	}
	return "false"
}

func isPlaceholderPaginationToken(token string) bool {
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "stackyard", "x", "xx", "xxx", "xxxx":
		return true
	default:
		return false
	}
}

func parseFlexibleTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC(), true
		}
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if len(raw) > 10 {
			return time.UnixMilli(seconds).UTC(), true
		}
		return time.Unix(seconds, 0).UTC(), true
	}
	return time.Time{}, false
}

func durationMatches(filter string, seconds int) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	if parsed, err := strconv.Atoi(filter); err == nil {
		if parsed == seconds {
			return true
		}
		if parsed < 1000 {
			return parsed*31536000 == seconds
		}
	}
	switch strings.ToLower(filter) {
	case "1", "1 year", "1year", "oneyear":
		return seconds == 31536000
	case "3", "3 years", "3years", "threeyears":
		return seconds == 94608000
	default:
		return false
	}
}

func defaultReservedNodeOfferings() []ReservedNodeOffering {
	return []ReservedNodeOffering{
		{
			ReservedNodesOfferingID: "memorydb-r6g-large-1yr-no-upfront",
			NodeType:                "db.r6g.large",
			Duration:                31536000,
			FixedPrice:              0,
			OfferingType:            "No Upfront",
			RecurringCharges: []RecurringCharge{
				{Amount: 0.083, Frequency: "Hourly"},
			},
		},
		{
			ReservedNodesOfferingID: "memorydb-r7g-xlarge-3yr-partial-upfront",
			NodeType:                "db.r7g.xlarge",
			Duration:                94608000,
			FixedPrice:              500,
			OfferingType:            "Partial Upfront",
			RecurringCharges: []RecurringCharge{
				{Amount: 0.055, Frequency: "Hourly"},
			},
		},
	}
}

func defaultReservedNodeOfferingByID(offeringID string) ReservedNodeOffering {
	offeringID = strings.TrimSpace(offeringID)
	for _, entry := range defaultReservedNodeOfferings() {
		if strings.EqualFold(entry.ReservedNodesOfferingID, offeringID) {
			return entry
		}
	}
	return ReservedNodeOffering{
		ReservedNodesOfferingID: offeringID,
		NodeType:                "db.r6g.large",
		Duration:                31536000,
		FixedPrice:              0,
		OfferingType:            "No Upfront",
		RecurringCharges: []RecurringCharge{
			{Amount: 0.083, Frequency: "Hourly"},
		},
	}
}

func aclARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:acl/%s", defaultRegion, defaultAccountID, strings.TrimSpace(name))
}

func userARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:user/%s", defaultRegion, defaultAccountID, strings.TrimSpace(name))
}

func subnetGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:subnetgroup/%s", defaultRegion, defaultAccountID, strings.TrimSpace(name))
}

func clusterARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:cluster/%s", defaultRegion, defaultAccountID, strings.TrimSpace(name))
}

func parameterGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:parametergroup/%s", defaultRegion, defaultAccountID, strings.TrimSpace(name))
}

func snapshotARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:snapshot/%s", defaultRegion, defaultAccountID, strings.TrimSpace(name))
}

func multiRegionClusterARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:multiregioncluster/%s", defaultRegion, defaultAccountID, strings.TrimSpace(name))
}

func reservedNodeARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:reservednode/%s", defaultRegion, defaultAccountID, strings.TrimSpace(name))
}
