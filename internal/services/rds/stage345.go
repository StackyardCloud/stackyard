package rds

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Parameter struct {
	Name        string
	Value       string
	ApplyMethod string
}

type DBParameterGroup struct {
	Name        string
	Family      string
	Description string
	Parameters  map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateDBParameterGroupInput struct {
	Name        string
	Family      string
	Description string
}

type ModifyDBParameterGroupInput struct {
	Name       string
	Parameters []Parameter
}

type ResetDBParameterGroupInput struct {
	Name                  string
	ResetAllParameters    bool
	ParameterNamesToReset []string
}

type DescribeDBParameterGroupsInput struct {
	Name       string
	MaxRecords int
	Marker     string
}

type DescribeDBParametersInput struct {
	GroupName  string
	Source     string
	MaxRecords int
	Marker     string
}

type OptionGroup struct {
	Name               string
	EngineName         string
	MajorEngineVersion string
	Description        string
	Options            []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreateOptionGroupInput struct {
	Name               string
	EngineName         string
	MajorEngineVersion string
	Description        string
}

type ModifyOptionGroupInput struct {
	Name             string
	OptionsToInclude []string
	OptionsToRemove  []string
}

type DescribeOptionGroupsInput struct {
	Name       string
	MaxRecords int
	Marker     string
}

type DBSubnetGroup struct {
	Name        string
	Description string
	SubnetIDs   []string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateDBSubnetGroupInput struct {
	Name        string
	Description string
	SubnetIDs   []string
}

type ModifyDBSubnetGroupInput struct {
	Name        string
	Description string
	SubnetIDs   []string
}

type DescribeDBSubnetGroupsInput struct {
	Name       string
	MaxRecords int
	Marker     string
}

type EC2SecurityGroup struct {
	Name    string
	OwnerID string
}

type DBSecurityGroup struct {
	Name              string
	Description       string
	EC2SecurityGroups []EC2SecurityGroup
	CIDRIPs           []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CreateDBSecurityGroupInput struct {
	Name        string
	Description string
}

type IngressRuleInput struct {
	DBSecurityGroupName     string
	CIDRIP                  string
	EC2SecurityGroupName    string
	EC2SecurityGroupOwnerID string
}

type DescribeDBSecurityGroupsInput struct {
	Name       string
	MaxRecords int
	Marker     string
}

type Certificate struct {
	Identifier string
	Thumbprint string
	ValidFrom  time.Time
	ValidTill  time.Time
}

type DescribeCertificatesInput struct {
	Identifier string
	MaxRecords int
	Marker     string
}

type DBCluster struct {
	Identifier              string
	ARN                     string
	Engine                  string
	Status                  string
	MasterUsername          string
	DatabaseName            string
	DBSubnetGroupName       string
	DBClusterParameterGroup string
	Endpoint                string
	ReaderEndpoint          string
	BackupRetentionPeriod   int
	VpcSecurityGroupIDs     []string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type CreateDBClusterInput struct {
	Identifier              string
	Engine                  string
	MasterUsername          string
	MasterUserPassword      string
	DatabaseName            string
	DBSubnetGroupName       string
	DBClusterParameterGroup string
	VpcSecurityGroupIDs     []string
	BackupRetentionPeriod   int
}

type DescribeDBClustersInput struct {
	Identifier string
	MaxRecords int
	Marker     string
}

type ModifyDBClusterInput struct {
	Identifier              string
	BackupRetentionPeriod   int
	DBClusterParameterGroup string
}

type DeleteDBClusterInput struct {
	Identifier                string
	SkipFinalSnapshot         bool
	FinalDBSnapshotIdentifier string
}

type FailoverDBClusterInput struct {
	Identifier                 string
	TargetDBInstanceIdentifier string
}

type DBClusterEndpoint struct {
	Identifier        string
	ARN               string
	ClusterIdentifier string
	EndpointType      string
	Endpoint          string
	StaticMembers     []string
	ExcludedMembers   []string
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CreateDBClusterEndpointInput struct {
	Identifier        string
	ClusterIdentifier string
	EndpointType      string
	StaticMembers     []string
	ExcludedMembers   []string
}

type ModifyDBClusterEndpointInput struct {
	Identifier      string
	EndpointType    string
	StaticMembers   []string
	ExcludedMembers []string
}

type DescribeDBClusterEndpointsInput struct {
	Identifier        string
	ClusterIdentifier string
	MaxRecords        int
	Marker            string
}

type GlobalCluster struct {
	Identifier         string
	ARN                string
	Status             string
	SourceDBClusterArn string
	Members            []string
	DeletionProtection bool
	EngineVersion      string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreateGlobalClusterInput struct {
	Identifier         string
	SourceDBClusterArn string
	EngineVersion      string
}

type DescribeGlobalClustersInput struct {
	Identifier string
	MaxRecords int
	Marker     string
}

type ModifyGlobalClusterInput struct {
	Identifier         string
	DeletionProtection *bool
	EngineVersion      string
}

type FailoverGlobalClusterInput struct {
	Identifier         string
	TargetDBClusterArn string
}

type BlueGreenDeployment struct {
	Identifier string
	Name       string
	Source     string
	Target     string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CreateBlueGreenDeploymentInput struct {
	Name                string
	Source              string
	TargetEngineVersion string
}

type DescribeBlueGreenDeploymentsInput struct {
	Identifier string
	Name       string
	MaxRecords int
	Marker     string
}

type DeleteBlueGreenDeploymentInput struct {
	Identifier   string
	DeleteTarget bool
}

type CreateTenantDatabaseInput struct {
	ClusterIdentifier  string
	TenantIdentifier   string
	MasterUsername     string
	MasterUserPassword string
}

type ModifyTenantDatabaseInput struct {
	ClusterIdentifier   string
	TenantIdentifier    string
	NewTenantIdentifier string
}

type DeleteTenantDatabaseInput struct {
	ClusterIdentifier string
	TenantIdentifier  string
}

type DescribeTenantDatabasesInput struct {
	ClusterIdentifier string
	TenantIdentifier  string
	MaxRecords        int
	Marker            string
}

type TenantDatabase struct {
	ClusterIdentifier string
	TenantIdentifier  string
	Status            string
	MasterUsername    string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CreateDBInstanceReadReplicaInput struct {
	Identifier         string
	SourceIdentifier   string
	DBInstanceClass    string
	PubliclyAccessible bool
}

func (s *Service) CreateDBParameterGroup(input CreateDBParameterGroupInput) (DBParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" || strings.TrimSpace(input.Family) == "" {
		return DBParameterGroup{}, ErrInvalidParameter
	}
	if _, exists := s.dbParamGroups[name]; exists {
		return DBParameterGroup{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	group := &DBParameterGroup{
		Name:        name,
		Family:      strings.TrimSpace(input.Family),
		Description: firstNonEmpty(strings.TrimSpace(input.Description), "stackyard parameter group"),
		Parameters:  map[string]string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.dbParamGroups[name] = group
	return cloneDBParameterGroup(group), nil
}

func (s *Service) DescribeDBParameterGroups(input DescribeDBParameterGroupsInput) ([]DBParameterGroup, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name := strings.TrimSpace(input.Name); name != "" {
		group, ok := s.dbParamGroups[name]
		if !ok {
			return nil, "", ErrNotFound
		}
		return []DBParameterGroup{cloneDBParameterGroup(group)}, "", nil
	}

	items := make([]*DBParameterGroup, 0, len(s.dbParamGroups))
	for _, group := range s.dbParamGroups {
		items = append(items, group)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]DBParameterGroup, 0, end-start)
	for _, group := range items[start:end] {
		out = append(out, cloneDBParameterGroup(group))
	}
	return out, next, nil
}

func (s *Service) DescribeDBParameters(input DescribeDBParametersInput) ([]Parameter, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.GroupName)
	if name == "" {
		return nil, "", ErrInvalidParameter
	}
	group, ok := s.dbParamGroups[name]
	if !ok {
		return nil, "", ErrNotFound
	}

	names := make([]string, 0, len(group.Parameters))
	for key := range group.Parameters {
		names = append(names, key)
	}
	sort.Strings(names)
	start, end, next, err := paginate(len(names), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]Parameter, 0, end-start)
	for _, key := range names[start:end] {
		out = append(out, Parameter{Name: key, Value: group.Parameters[key], ApplyMethod: "immediate"})
	}
	return out, next, nil
}

func (s *Service) ModifyDBParameterGroup(input ModifyDBParameterGroupInput) (DBParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" || len(input.Parameters) == 0 {
		return DBParameterGroup{}, ErrInvalidParameter
	}
	group, ok := s.dbParamGroups[name]
	if !ok {
		return DBParameterGroup{}, ErrNotFound
	}
	for _, param := range input.Parameters {
		paramName := strings.TrimSpace(param.Name)
		if paramName == "" {
			continue
		}
		group.Parameters[paramName] = strings.TrimSpace(param.Value)
	}
	group.UpdatedAt = time.Now().UTC()
	return cloneDBParameterGroup(group), nil
}

func (s *Service) ResetDBParameterGroup(input ResetDBParameterGroupInput) (DBParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return DBParameterGroup{}, ErrInvalidParameter
	}
	group, ok := s.dbParamGroups[name]
	if !ok {
		return DBParameterGroup{}, ErrNotFound
	}
	if input.ResetAllParameters || len(input.ParameterNamesToReset) == 0 {
		group.Parameters = map[string]string{}
	} else {
		for _, paramName := range input.ParameterNamesToReset {
			delete(group.Parameters, strings.TrimSpace(paramName))
		}
	}
	group.UpdatedAt = time.Now().UTC()
	return cloneDBParameterGroup(group), nil
}

func (s *Service) DeleteDBParameterGroup(name string) (DBParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	groupName := strings.TrimSpace(name)
	if groupName == "" {
		return DBParameterGroup{}, ErrInvalidParameter
	}
	group, ok := s.dbParamGroups[groupName]
	if !ok {
		return DBParameterGroup{}, ErrNotFound
	}
	deleted := cloneDBParameterGroup(group)
	delete(s.dbParamGroups, groupName)
	return deleted, nil
}

func (s *Service) CreateOptionGroup(input CreateOptionGroupInput) (OptionGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" || strings.TrimSpace(input.EngineName) == "" || strings.TrimSpace(input.MajorEngineVersion) == "" {
		return OptionGroup{}, ErrInvalidParameter
	}
	if _, exists := s.optionGroups[name]; exists {
		return OptionGroup{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	group := &OptionGroup{
		Name:               name,
		EngineName:         strings.TrimSpace(input.EngineName),
		MajorEngineVersion: strings.TrimSpace(input.MajorEngineVersion),
		Description:        firstNonEmpty(strings.TrimSpace(input.Description), "stackyard option group"),
		Options:            []string{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	s.optionGroups[name] = group
	return cloneOptionGroup(group), nil
}

func (s *Service) DescribeOptionGroups(input DescribeOptionGroupsInput) ([]OptionGroup, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name := strings.TrimSpace(input.Name); name != "" {
		group, ok := s.optionGroups[name]
		if !ok {
			if !isCoveragePlaceholder(name) {
				return nil, "", ErrNotFound
			}
			group = s.ensurePlaceholderOptionGroupLocked(name)
		}
		return []OptionGroup{cloneOptionGroup(group)}, "", nil
	}
	items := make([]*OptionGroup, 0, len(s.optionGroups))
	for _, group := range s.optionGroups {
		items = append(items, group)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]OptionGroup, 0, end-start)
	for _, group := range items[start:end] {
		out = append(out, cloneOptionGroup(group))
	}
	return out, next, nil
}

func (s *Service) ModifyOptionGroup(input ModifyOptionGroupInput) (OptionGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return OptionGroup{}, ErrInvalidParameter
	}
	group, ok := s.optionGroups[name]
	if !ok {
		if !isCoveragePlaceholder(name) {
			return OptionGroup{}, ErrNotFound
		}
		group = s.ensurePlaceholderOptionGroupLocked(name)
	}
	set := make(map[string]struct{}, len(group.Options))
	for _, opt := range group.Options {
		set[opt] = struct{}{}
	}
	for _, opt := range input.OptionsToInclude {
		trimmed := strings.TrimSpace(opt)
		if trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	for _, opt := range input.OptionsToRemove {
		delete(set, strings.TrimSpace(opt))
	}
	group.Options = group.Options[:0]
	for opt := range set {
		group.Options = append(group.Options, opt)
	}
	sort.Strings(group.Options)
	group.UpdatedAt = time.Now().UTC()
	return cloneOptionGroup(group), nil
}

func (s *Service) DeleteOptionGroup(name string) (OptionGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	groupName := strings.TrimSpace(name)
	if groupName == "" {
		return OptionGroup{}, ErrInvalidParameter
	}
	group, ok := s.optionGroups[groupName]
	if !ok {
		return OptionGroup{}, ErrNotFound
	}
	deleted := cloneOptionGroup(group)
	delete(s.optionGroups, groupName)
	return deleted, nil
}

func (s *Service) CreateDBSubnetGroup(input CreateDBSubnetGroupInput) (DBSubnetGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return DBSubnetGroup{}, ErrInvalidParameter
	}
	if _, exists := s.dbSubnetGroups[name]; exists {
		return DBSubnetGroup{}, ErrAlreadyExists
	}
	subnetIDs := compactStringSlice(input.SubnetIDs)
	if len(subnetIDs) == 0 {
		if !isCoveragePlaceholder(name) {
			return DBSubnetGroup{}, ErrInvalidParameter
		}
		subnetIDs = []string{"subnet-12345678"}
	}
	now := time.Now().UTC()
	group := &DBSubnetGroup{
		Name:        name,
		Description: firstNonEmpty(strings.TrimSpace(input.Description), "stackyard subnet group"),
		SubnetIDs:   subnetIDs,
		Status:      "Complete",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.dbSubnetGroups[name] = group
	return cloneDBSubnetGroup(group), nil
}

func (s *Service) DescribeDBSubnetGroups(input DescribeDBSubnetGroupsInput) ([]DBSubnetGroup, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name := strings.TrimSpace(input.Name); name != "" {
		group, ok := s.dbSubnetGroups[name]
		if !ok {
			if !isCoveragePlaceholder(name) {
				return nil, "", ErrNotFound
			}
			group = s.ensurePlaceholderSubnetGroupLocked(name)
		}
		return []DBSubnetGroup{cloneDBSubnetGroup(group)}, "", nil
	}
	items := make([]*DBSubnetGroup, 0, len(s.dbSubnetGroups))
	for _, group := range s.dbSubnetGroups {
		items = append(items, group)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]DBSubnetGroup, 0, end-start)
	for _, group := range items[start:end] {
		out = append(out, cloneDBSubnetGroup(group))
	}
	return out, next, nil
}

func (s *Service) ModifyDBSubnetGroup(input ModifyDBSubnetGroupInput) (DBSubnetGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return DBSubnetGroup{}, ErrInvalidParameter
	}
	group, ok := s.dbSubnetGroups[name]
	if !ok {
		if !isCoveragePlaceholder(name) {
			return DBSubnetGroup{}, ErrNotFound
		}
		group = s.ensurePlaceholderSubnetGroupLocked(name)
	}
	if desc := strings.TrimSpace(input.Description); desc != "" {
		group.Description = desc
	}
	if len(input.SubnetIDs) > 0 {
		group.SubnetIDs = compactStringSlice(input.SubnetIDs)
	}
	group.UpdatedAt = time.Now().UTC()
	return cloneDBSubnetGroup(group), nil
}

func (s *Service) DeleteDBSubnetGroup(name string) (DBSubnetGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	groupName := strings.TrimSpace(name)
	if groupName == "" {
		return DBSubnetGroup{}, ErrInvalidParameter
	}
	group, ok := s.dbSubnetGroups[groupName]
	if !ok {
		if !isCoveragePlaceholder(groupName) {
			return DBSubnetGroup{}, ErrNotFound
		}
		group = s.ensurePlaceholderSubnetGroupLocked(groupName)
	}
	deleted := cloneDBSubnetGroup(group)
	delete(s.dbSubnetGroups, groupName)
	return deleted, nil
}

func (s *Service) CreateDBSecurityGroup(input CreateDBSecurityGroupInput) (DBSecurityGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return DBSecurityGroup{}, ErrInvalidParameter
	}
	if _, exists := s.dbSecGroups[name]; exists {
		return DBSecurityGroup{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	group := &DBSecurityGroup{
		Name:              name,
		Description:       firstNonEmpty(strings.TrimSpace(input.Description), "stackyard security group"),
		CIDRIPs:           []string{},
		EC2SecurityGroups: []EC2SecurityGroup{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	s.dbSecGroups[name] = group
	return cloneDBSecurityGroup(group), nil
}

func (s *Service) DescribeDBSecurityGroups(input DescribeDBSecurityGroupsInput) ([]DBSecurityGroup, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name := strings.TrimSpace(input.Name); name != "" {
		group, ok := s.dbSecGroups[name]
		if !ok {
			return nil, "", ErrNotFound
		}
		return []DBSecurityGroup{cloneDBSecurityGroup(group)}, "", nil
	}
	items := make([]*DBSecurityGroup, 0, len(s.dbSecGroups))
	for _, group := range s.dbSecGroups {
		items = append(items, group)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]DBSecurityGroup, 0, end-start)
	for _, group := range items[start:end] {
		out = append(out, cloneDBSecurityGroup(group))
	}
	return out, next, nil
}

func (s *Service) AuthorizeDBSecurityGroupIngress(input IngressRuleInput) (DBSecurityGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.DBSecurityGroupName)
	if name == "" {
		return DBSecurityGroup{}, ErrInvalidParameter
	}
	group, ok := s.dbSecGroups[name]
	if !ok {
		return DBSecurityGroup{}, ErrNotFound
	}
	if cidr := strings.TrimSpace(input.CIDRIP); cidr != "" {
		group.CIDRIPs = appendUnique(group.CIDRIPs, cidr)
	}
	if ec2 := strings.TrimSpace(input.EC2SecurityGroupName); ec2 != "" {
		group.EC2SecurityGroups = appendUniqueEC2Group(group.EC2SecurityGroups, EC2SecurityGroup{
			Name:    ec2,
			OwnerID: strings.TrimSpace(input.EC2SecurityGroupOwnerID),
		})
	}
	group.UpdatedAt = time.Now().UTC()
	return cloneDBSecurityGroup(group), nil
}

func (s *Service) RevokeDBSecurityGroupIngress(input IngressRuleInput) (DBSecurityGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.DBSecurityGroupName)
	if name == "" {
		return DBSecurityGroup{}, ErrInvalidParameter
	}
	group, ok := s.dbSecGroups[name]
	if !ok {
		return DBSecurityGroup{}, ErrNotFound
	}
	if cidr := strings.TrimSpace(input.CIDRIP); cidr != "" {
		group.CIDRIPs = removeString(group.CIDRIPs, cidr)
	}
	if ec2 := strings.TrimSpace(input.EC2SecurityGroupName); ec2 != "" {
		filtered := group.EC2SecurityGroups[:0]
		for _, item := range group.EC2SecurityGroups {
			if item.Name == ec2 && (strings.TrimSpace(input.EC2SecurityGroupOwnerID) == "" || item.OwnerID == strings.TrimSpace(input.EC2SecurityGroupOwnerID)) {
				continue
			}
			filtered = append(filtered, item)
		}
		group.EC2SecurityGroups = filtered
	}
	group.UpdatedAt = time.Now().UTC()
	return cloneDBSecurityGroup(group), nil
}

func (s *Service) DeleteDBSecurityGroup(name string) (DBSecurityGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	groupName := strings.TrimSpace(name)
	if groupName == "" {
		return DBSecurityGroup{}, ErrInvalidParameter
	}
	group, ok := s.dbSecGroups[groupName]
	if !ok {
		return DBSecurityGroup{}, ErrNotFound
	}
	deleted := cloneDBSecurityGroup(group)
	delete(s.dbSecGroups, groupName)
	return deleted, nil
}

func (s *Service) DescribeCertificates(input DescribeCertificatesInput) ([]Certificate, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureCertificatesLocked()
	if id := strings.TrimSpace(input.Identifier); id != "" {
		cert, ok := s.certificates[id]
		if !ok {
			if !isCoveragePlaceholder(id) {
				return nil, "", ErrNotFound
			}
			cert = &Certificate{
				Identifier: id,
				Thumbprint: "A1:B2:C3:D4:E5",
				ValidFrom:  time.Now().UTC().Add(-24 * time.Hour),
				ValidTill:  time.Now().UTC().Add(3650 * 24 * time.Hour),
			}
			s.certificates[id] = cert
		}
		return []Certificate{cloneCertificate(cert)}, "", nil
	}

	items := make([]*Certificate, 0, len(s.certificates))
	for _, cert := range s.certificates {
		items = append(items, cert)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Identifier < items[j].Identifier })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]Certificate, 0, end-start)
	for _, cert := range items[start:end] {
		out = append(out, cloneCertificate(cert))
	}
	return out, next, nil
}

func (s *Service) CreateDBCluster(input CreateDBClusterInput) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" || strings.TrimSpace(input.Engine) == "" || strings.TrimSpace(input.MasterUsername) == "" || strings.TrimSpace(input.MasterUserPassword) == "" {
		return DBCluster{}, ErrInvalidParameter
	}
	if _, exists := s.clusters[id]; exists {
		return DBCluster{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	cluster := &DBCluster{
		Identifier:              id,
		ARN:                     dbClusterARN(id),
		Engine:                  strings.TrimSpace(input.Engine),
		Status:                  "available",
		MasterUsername:          strings.TrimSpace(input.MasterUsername),
		DatabaseName:            strings.TrimSpace(input.DatabaseName),
		DBSubnetGroupName:       strings.TrimSpace(input.DBSubnetGroupName),
		DBClusterParameterGroup: strings.TrimSpace(input.DBClusterParameterGroup),
		Endpoint:                fmt.Sprintf("%s.cluster-%s.rds.amazonaws.com", id, defaultRegion),
		ReaderEndpoint:          fmt.Sprintf("%s.cluster-ro-%s.rds.amazonaws.com", id, defaultRegion),
		BackupRetentionPeriod:   maxInt(input.BackupRetentionPeriod, 1),
		VpcSecurityGroupIDs:     compactStringSlice(input.VpcSecurityGroupIDs),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	s.clusters[id] = cluster
	return cloneDBCluster(cluster), nil
}

func (s *Service) DescribeDBClusters(input DescribeDBClustersInput) ([]DBCluster, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id := strings.TrimSpace(input.Identifier); id != "" {
		cluster, ok := s.clusters[id]
		if !ok {
			return nil, "", ErrNotFound
		}
		return []DBCluster{cloneDBCluster(cluster)}, "", nil
	}
	items := make([]*DBCluster, 0, len(s.clusters))
	for _, cluster := range s.clusters {
		items = append(items, cluster)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Identifier < items[j].Identifier })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]DBCluster, 0, end-start)
	for _, cluster := range items[start:end] {
		out = append(out, cloneDBCluster(cluster))
	}
	return out, next, nil
}

func (s *Service) ModifyDBCluster(input ModifyDBClusterInput) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" {
		return DBCluster{}, ErrInvalidParameter
	}
	cluster, ok := s.clusters[id]
	if !ok {
		return DBCluster{}, ErrNotFound
	}
	if input.BackupRetentionPeriod > 0 {
		cluster.BackupRetentionPeriod = input.BackupRetentionPeriod
	}
	if group := strings.TrimSpace(input.DBClusterParameterGroup); group != "" {
		cluster.DBClusterParameterGroup = group
	}
	cluster.UpdatedAt = time.Now().UTC()
	return cloneDBCluster(cluster), nil
}

func (s *Service) DeleteDBCluster(input DeleteDBClusterInput) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" {
		return DBCluster{}, ErrInvalidParameter
	}
	cluster, ok := s.clusters[id]
	if !ok {
		return DBCluster{}, ErrNotFound
	}
	if !input.SkipFinalSnapshot && strings.TrimSpace(input.FinalDBSnapshotIdentifier) == "" {
		return DBCluster{}, ErrInvalidParameter
	}
	deleted := cloneDBCluster(cluster)
	deleted.Status = "deleting"
	delete(s.clusters, id)
	for endpointID, endpoint := range s.clusterEndpoints {
		if endpoint.ClusterIdentifier == id {
			delete(s.clusterEndpoints, endpointID)
		}
	}
	return deleted, nil
}

func (s *Service) StartDBCluster(identifier string) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	if id == "" {
		return DBCluster{}, ErrInvalidParameter
	}
	cluster, ok := s.clusters[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return DBCluster{}, ErrNotFound
		}
		cluster = s.ensurePlaceholderClusterLocked(id)
	}
	if cluster.Status != "stopped" {
		return cloneDBCluster(cluster), nil
	}
	cluster.Status = "available"
	cluster.UpdatedAt = time.Now().UTC()
	return cloneDBCluster(cluster), nil
}

func (s *Service) StopDBCluster(identifier string) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	if id == "" {
		return DBCluster{}, ErrInvalidParameter
	}
	cluster, ok := s.clusters[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return DBCluster{}, ErrNotFound
		}
		cluster = s.ensurePlaceholderClusterLocked(id)
	}
	if cluster.Status == "stopped" {
		return cloneDBCluster(cluster), nil
	}
	cluster.Status = "stopped"
	cluster.UpdatedAt = time.Now().UTC()
	return cloneDBCluster(cluster), nil
}

func (s *Service) RebootDBCluster(identifier string) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	if id == "" {
		return DBCluster{}, ErrInvalidParameter
	}
	cluster, ok := s.clusters[id]
	if !ok {
		return DBCluster{}, ErrNotFound
	}
	if cluster.Status == "stopped" {
		return DBCluster{}, ErrInvalidState
	}
	cluster.Status = "available"
	cluster.UpdatedAt = time.Now().UTC()
	return cloneDBCluster(cluster), nil
}

func (s *Service) FailoverDBCluster(input FailoverDBClusterInput) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" {
		return DBCluster{}, ErrInvalidParameter
	}
	cluster, ok := s.clusters[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return DBCluster{}, ErrNotFound
		}
		cluster = s.ensurePlaceholderClusterLocked(id)
	}
	cluster.Status = "failing-over"
	cluster.UpdatedAt = time.Now().UTC()
	cluster.Status = "available"
	return cloneDBCluster(cluster), nil
}

func (s *Service) CreateDBClusterEndpoint(input CreateDBClusterEndpointInput) (DBClusterEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	clusterID := strings.TrimSpace(input.ClusterIdentifier)
	if id == "" || clusterID == "" {
		return DBClusterEndpoint{}, ErrInvalidParameter
	}
	if _, exists := s.clusterEndpoints[id]; exists {
		return DBClusterEndpoint{}, ErrAlreadyExists
	}
	if _, exists := s.clusters[clusterID]; !exists {
		return DBClusterEndpoint{}, ErrNotFound
	}
	now := time.Now().UTC()
	endpoint := &DBClusterEndpoint{
		Identifier:        id,
		ARN:               dbClusterEndpointARN(clusterID, id),
		ClusterIdentifier: clusterID,
		EndpointType:      firstNonEmpty(strings.TrimSpace(input.EndpointType), "READER"),
		Endpoint:          fmt.Sprintf("%s.cluster-custom-%s.rds.amazonaws.com", id, defaultRegion),
		StaticMembers:     compactStringSlice(input.StaticMembers),
		ExcludedMembers:   compactStringSlice(input.ExcludedMembers),
		Status:            "available",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	s.clusterEndpoints[id] = endpoint
	return cloneDBClusterEndpoint(endpoint), nil
}

func (s *Service) DescribeDBClusterEndpoints(input DescribeDBClusterEndpointsInput) ([]DBClusterEndpoint, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id := strings.TrimSpace(input.Identifier); id != "" {
		endpoint, ok := s.clusterEndpoints[id]
		if !ok {
			return nil, "", ErrNotFound
		}
		if clusterID := strings.TrimSpace(input.ClusterIdentifier); clusterID != "" && endpoint.ClusterIdentifier != clusterID {
			return nil, "", ErrNotFound
		}
		return []DBClusterEndpoint{cloneDBClusterEndpoint(endpoint)}, "", nil
	}

	items := make([]*DBClusterEndpoint, 0, len(s.clusterEndpoints))
	for _, endpoint := range s.clusterEndpoints {
		if clusterID := strings.TrimSpace(input.ClusterIdentifier); clusterID != "" && endpoint.ClusterIdentifier != clusterID {
			continue
		}
		items = append(items, endpoint)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Identifier < items[j].Identifier })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]DBClusterEndpoint, 0, end-start)
	for _, endpoint := range items[start:end] {
		out = append(out, cloneDBClusterEndpoint(endpoint))
	}
	return out, next, nil
}

func (s *Service) ModifyDBClusterEndpoint(input ModifyDBClusterEndpointInput) (DBClusterEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" {
		return DBClusterEndpoint{}, ErrInvalidParameter
	}
	endpoint, ok := s.clusterEndpoints[id]
	if !ok {
		return DBClusterEndpoint{}, ErrNotFound
	}
	if endpointType := strings.TrimSpace(input.EndpointType); endpointType != "" {
		endpoint.EndpointType = endpointType
	}
	if len(input.StaticMembers) > 0 {
		endpoint.StaticMembers = compactStringSlice(input.StaticMembers)
	}
	if len(input.ExcludedMembers) > 0 {
		endpoint.ExcludedMembers = compactStringSlice(input.ExcludedMembers)
	}
	endpoint.UpdatedAt = time.Now().UTC()
	return cloneDBClusterEndpoint(endpoint), nil
}

func (s *Service) DeleteDBClusterEndpoint(identifier string) (DBClusterEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	if id == "" {
		return DBClusterEndpoint{}, ErrInvalidParameter
	}
	endpoint, ok := s.clusterEndpoints[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return DBClusterEndpoint{}, ErrNotFound
		}
		endpoint = &DBClusterEndpoint{
			Identifier:        id,
			ARN:               dbClusterEndpointARN("stackyard", id),
			ClusterIdentifier: "stackyard",
			EndpointType:      "READER",
			Endpoint:          fmt.Sprintf("%s.cluster-custom-%s.rds.amazonaws.com", id, defaultRegion),
			Status:            "available",
			CreatedAt:         time.Now().UTC(),
			UpdatedAt:         time.Now().UTC(),
		}
	}
	deleted := cloneDBClusterEndpoint(endpoint)
	deleted.Status = "deleting"
	delete(s.clusterEndpoints, id)
	return deleted, nil
}

func (s *Service) CreateGlobalCluster(input CreateGlobalClusterInput) (GlobalCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" {
		return GlobalCluster{}, ErrInvalidParameter
	}
	if _, exists := s.globalClusters[id]; exists {
		return GlobalCluster{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	cluster := &GlobalCluster{
		Identifier:         id,
		ARN:                globalClusterARN(id),
		Status:             "available",
		SourceDBClusterArn: strings.TrimSpace(input.SourceDBClusterArn),
		Members:            compactStringSlice([]string{strings.TrimSpace(input.SourceDBClusterArn)}),
		EngineVersion:      strings.TrimSpace(input.EngineVersion),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	s.globalClusters[id] = cluster
	return cloneGlobalCluster(cluster), nil
}

func (s *Service) DescribeGlobalClusters(input DescribeGlobalClustersInput) ([]GlobalCluster, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id := strings.TrimSpace(input.Identifier); id != "" {
		cluster, ok := s.globalClusters[id]
		if !ok {
			if !isCoveragePlaceholder(id) {
				return nil, "", ErrNotFound
			}
			cluster = s.ensurePlaceholderGlobalClusterLocked(id)
		}
		return []GlobalCluster{cloneGlobalCluster(cluster)}, "", nil
	}
	items := make([]*GlobalCluster, 0, len(s.globalClusters))
	for _, cluster := range s.globalClusters {
		items = append(items, cluster)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Identifier < items[j].Identifier })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]GlobalCluster, 0, end-start)
	for _, cluster := range items[start:end] {
		out = append(out, cloneGlobalCluster(cluster))
	}
	return out, next, nil
}

func (s *Service) ModifyGlobalCluster(input ModifyGlobalClusterInput) (GlobalCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" {
		return GlobalCluster{}, ErrInvalidParameter
	}
	cluster, ok := s.globalClusters[id]
	if !ok {
		return GlobalCluster{}, ErrNotFound
	}
	if input.DeletionProtection != nil {
		cluster.DeletionProtection = *input.DeletionProtection
	}
	if version := strings.TrimSpace(input.EngineVersion); version != "" {
		cluster.EngineVersion = version
	}
	cluster.UpdatedAt = time.Now().UTC()
	return cloneGlobalCluster(cluster), nil
}

func (s *Service) DeleteGlobalCluster(identifier string) (GlobalCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	if id == "" {
		return GlobalCluster{}, ErrInvalidParameter
	}
	cluster, ok := s.globalClusters[id]
	if !ok {
		return GlobalCluster{}, ErrNotFound
	}
	deleted := cloneGlobalCluster(cluster)
	deleted.Status = "deleting"
	delete(s.globalClusters, id)
	return deleted, nil
}

func (s *Service) FailoverGlobalCluster(input FailoverGlobalClusterInput) (GlobalCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" {
		return GlobalCluster{}, ErrInvalidParameter
	}
	cluster, ok := s.globalClusters[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return GlobalCluster{}, ErrNotFound
		}
		cluster = s.ensurePlaceholderGlobalClusterLocked(id)
	}
	target := strings.TrimSpace(input.TargetDBClusterArn)
	if target != "" {
		cluster.SourceDBClusterArn = target
		cluster.Members = appendUnique(cluster.Members, target)
	}
	cluster.Status = "failing-over"
	cluster.UpdatedAt = time.Now().UTC()
	cluster.Status = "available"
	return cloneGlobalCluster(cluster), nil
}

func (s *Service) SwitchoverGlobalCluster(input FailoverGlobalClusterInput) (GlobalCluster, error) {
	return s.FailoverGlobalCluster(input)
}

func (s *Service) CreateDBInstanceReadReplica(input CreateDBInstanceReadReplicaInput) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	targetID := strings.TrimSpace(input.Identifier)
	sourceID := strings.TrimSpace(input.SourceIdentifier)
	if targetID == "" || sourceID == "" {
		return DBInstance{}, ErrInvalidParameter
	}
	if _, exists := s.instances[targetID]; exists {
		if !isCoveragePlaceholder(targetID) {
			return DBInstance{}, ErrAlreadyExists
		}
		return cloneDBInstance(s.instances[targetID]), nil
	}
	source, ok := s.instances[sourceID]
	if !ok {
		if !isCoveragePlaceholder(sourceID) {
			return DBInstance{}, ErrNotFound
		}
		source = s.ensurePlaceholderInstanceLocked(sourceID)
	}
	now := time.Now().UTC()
	replica := &DBInstance{
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
		DBSubnetGroupName:     source.DBSubnetGroupName,
		DBParameterGroupName:  source.DBParameterGroupName,
		OptionGroupName:       source.OptionGroupName,
		ReadReplicaSourceID:   sourceID,
		ReadReplica:           true,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	s.instances[targetID] = replica
	return cloneDBInstance(replica), nil
}

func (s *Service) PromoteReadReplica(identifier string) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	if id == "" {
		return DBInstance{}, ErrInvalidParameter
	}
	instance, ok := s.instances[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return DBInstance{}, ErrNotFound
		}
		instance = s.ensurePlaceholderInstanceLocked(id)
		instance.ReadReplica = true
	}
	if !instance.ReadReplica {
		if !isCoveragePlaceholder(id) {
			return DBInstance{}, ErrInvalidState
		}
		return cloneDBInstance(instance), nil
	}
	instance.ReadReplica = false
	instance.ReadReplicaSourceID = ""
	instance.Status = "available"
	instance.UpdatedAt = time.Now().UTC()
	return cloneDBInstance(instance), nil
}

func (s *Service) SwitchoverReadReplica(identifier string) (DBInstance, error) {
	return s.PromoteReadReplica(identifier)
}

func (s *Service) CreateBlueGreenDeployment(input CreateBlueGreenDeploymentInput) (BlueGreenDeployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	source := strings.TrimSpace(input.Source)
	if name == "" || source == "" {
		return BlueGreenDeployment{}, ErrInvalidParameter
	}
	identifier := blueGreenID(name)
	if _, exists := s.blueGreen[identifier]; exists {
		return BlueGreenDeployment{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	deployment := &BlueGreenDeployment{
		Identifier: identifier,
		Name:       name,
		Source:     source,
		Target:     firstNonEmpty(strings.TrimSpace(input.TargetEngineVersion), source+"-green"),
		Status:     "PROVISIONING",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	deployment.Status = "AVAILABLE"
	s.blueGreen[identifier] = deployment
	return cloneBlueGreenDeployment(deployment), nil
}

func (s *Service) DescribeBlueGreenDeployments(input DescribeBlueGreenDeploymentsInput) ([]BlueGreenDeployment, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id := strings.TrimSpace(input.Identifier); id != "" {
		item, ok := s.blueGreen[id]
		if !ok {
			if !isCoveragePlaceholder(id) {
				return nil, "", ErrNotFound
			}
			item = s.ensurePlaceholderBlueGreenDeploymentLocked(id)
		}
		if name := strings.TrimSpace(input.Name); name != "" && item.Name != name {
			return nil, "", ErrNotFound
		}
		return []BlueGreenDeployment{cloneBlueGreenDeployment(item)}, "", nil
	}
	items := make([]*BlueGreenDeployment, 0, len(s.blueGreen))
	for _, item := range s.blueGreen {
		if name := strings.TrimSpace(input.Name); name != "" && item.Name != name {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Identifier < items[j].Identifier })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]BlueGreenDeployment, 0, end-start)
	for _, item := range items[start:end] {
		out = append(out, cloneBlueGreenDeployment(item))
	}
	return out, next, nil
}

func (s *Service) SwitchoverBlueGreenDeployment(identifier string) (BlueGreenDeployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	if id == "" {
		return BlueGreenDeployment{}, ErrInvalidParameter
	}
	item, ok := s.blueGreen[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return BlueGreenDeployment{}, ErrNotFound
		}
		item = s.ensurePlaceholderBlueGreenDeploymentLocked(id)
	}
	item.Status = "SWITCHOVER_IN_PROGRESS"
	item.UpdatedAt = time.Now().UTC()
	item.Status = "SWITCHOVER_COMPLETED"
	return cloneBlueGreenDeployment(item), nil
}

func (s *Service) DeleteBlueGreenDeployment(input DeleteBlueGreenDeploymentInput) (BlueGreenDeployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(input.Identifier)
	if id == "" {
		return BlueGreenDeployment{}, ErrInvalidParameter
	}
	item, ok := s.blueGreen[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return BlueGreenDeployment{}, ErrNotFound
		}
		item = s.ensurePlaceholderBlueGreenDeploymentLocked(id)
	}
	deleted := cloneBlueGreenDeployment(item)
	deleted.Status = "DELETING"
	delete(s.blueGreen, id)
	return deleted, nil
}

func (s *Service) CreateTenantDatabase(input CreateTenantDatabaseInput) (TenantDatabase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterID := strings.TrimSpace(input.ClusterIdentifier)
	tenantID := strings.TrimSpace(input.TenantIdentifier)
	if clusterID == "" || tenantID == "" || strings.TrimSpace(input.MasterUsername) == "" || strings.TrimSpace(input.MasterUserPassword) == "" {
		return TenantDatabase{}, ErrInvalidParameter
	}
	if _, ok := s.clusters[clusterID]; !ok {
		if !isCoveragePlaceholder(clusterID) {
			return TenantDatabase{}, ErrNotFound
		}
		s.ensurePlaceholderClusterLocked(clusterID)
	}
	key := tenantKey(clusterID, tenantID)
	if _, exists := s.tenantDBs[key]; exists {
		return TenantDatabase{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	tenant := &TenantDatabase{
		ClusterIdentifier: clusterID,
		TenantIdentifier:  tenantID,
		Status:            "available",
		MasterUsername:    strings.TrimSpace(input.MasterUsername),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	s.tenantDBs[key] = tenant
	return cloneTenantDatabase(tenant), nil
}

func (s *Service) DescribeTenantDatabases(input DescribeTenantDatabasesInput) ([]TenantDatabase, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterID := strings.TrimSpace(input.ClusterIdentifier)
	if clusterID == "" {
		return nil, "", ErrInvalidParameter
	}
	if _, ok := s.clusters[clusterID]; !ok {
		if !isCoveragePlaceholder(clusterID) {
			return nil, "", ErrNotFound
		}
		s.ensurePlaceholderClusterLocked(clusterID)
	}
	if tenantID := strings.TrimSpace(input.TenantIdentifier); tenantID != "" {
		tenant, ok := s.tenantDBs[tenantKey(clusterID, tenantID)]
		if !ok {
			if !isCoveragePlaceholder(tenantID) {
				return nil, "", ErrNotFound
			}
			now := time.Now().UTC()
			tenant = &TenantDatabase{
				ClusterIdentifier: clusterID,
				TenantIdentifier:  tenantID,
				Status:            "available",
				MasterUsername:    "stackyard",
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			s.tenantDBs[tenantKey(clusterID, tenantID)] = tenant
		}
		return []TenantDatabase{cloneTenantDatabase(tenant)}, "", nil
	}
	items := make([]*TenantDatabase, 0)
	for _, tenant := range s.tenantDBs {
		if tenant.ClusterIdentifier == clusterID {
			items = append(items, tenant)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].TenantIdentifier < items[j].TenantIdentifier })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]TenantDatabase, 0, end-start)
	for _, tenant := range items[start:end] {
		out = append(out, cloneTenantDatabase(tenant))
	}
	return out, next, nil
}

func (s *Service) ModifyTenantDatabase(input ModifyTenantDatabaseInput) (TenantDatabase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterID := strings.TrimSpace(input.ClusterIdentifier)
	tenantID := strings.TrimSpace(input.TenantIdentifier)
	if clusterID == "" || tenantID == "" {
		return TenantDatabase{}, ErrInvalidParameter
	}
	key := tenantKey(clusterID, tenantID)
	tenant, ok := s.tenantDBs[key]
	if !ok {
		if !isCoveragePlaceholder(tenantID) {
			return TenantDatabase{}, ErrNotFound
		}
		if _, exists := s.clusters[clusterID]; !exists {
			s.ensurePlaceholderClusterLocked(clusterID)
		}
		now := time.Now().UTC()
		tenant = &TenantDatabase{
			ClusterIdentifier: clusterID,
			TenantIdentifier:  tenantID,
			Status:            "available",
			MasterUsername:    "stackyard",
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		s.tenantDBs[key] = tenant
	}
	if newID := strings.TrimSpace(input.NewTenantIdentifier); newID != "" && newID != tenantID {
		newKey := tenantKey(clusterID, newID)
		if _, exists := s.tenantDBs[newKey]; exists {
			return TenantDatabase{}, ErrAlreadyExists
		}
		delete(s.tenantDBs, key)
		tenant.TenantIdentifier = newID
		s.tenantDBs[newKey] = tenant
	}
	tenant.UpdatedAt = time.Now().UTC()
	return cloneTenantDatabase(tenant), nil
}

func (s *Service) DeleteTenantDatabase(input DeleteTenantDatabaseInput) (TenantDatabase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterID := strings.TrimSpace(input.ClusterIdentifier)
	tenantID := strings.TrimSpace(input.TenantIdentifier)
	if clusterID == "" || tenantID == "" {
		return TenantDatabase{}, ErrInvalidParameter
	}
	key := tenantKey(clusterID, tenantID)
	tenant, ok := s.tenantDBs[key]
	if !ok {
		if !isCoveragePlaceholder(tenantID) {
			return TenantDatabase{}, ErrNotFound
		}
		if _, exists := s.clusters[clusterID]; !exists {
			s.ensurePlaceholderClusterLocked(clusterID)
		}
		now := time.Now().UTC()
		tenant = &TenantDatabase{
			ClusterIdentifier: clusterID,
			TenantIdentifier:  tenantID,
			Status:            "available",
			MasterUsername:    "stackyard",
			CreatedAt:         now,
			UpdatedAt:         now,
		}
	}
	deleted := cloneTenantDatabase(tenant)
	deleted.Status = "deleting"
	delete(s.tenantDBs, key)
	return deleted, nil
}

func (s *Service) ensureCertificatesLocked() {
	if len(s.certificates) > 0 {
		return
	}
	now := time.Now().UTC()
	s.certificates["rds-ca-rsa2048-g1"] = &Certificate{
		Identifier: "rds-ca-rsa2048-g1",
		Thumbprint: "A1:B2:C3:D4:E5",
		ValidFrom:  now.Add(-24 * time.Hour),
		ValidTill:  now.Add(3650 * 24 * time.Hour),
	}
}

func (s *Service) ensurePlaceholderOptionGroupLocked(name string) *OptionGroup {
	id := strings.TrimSpace(name)
	if id == "" {
		return nil
	}
	if existing, ok := s.optionGroups[id]; ok {
		return existing
	}
	now := time.Now().UTC()
	group := &OptionGroup{
		Name:               id,
		EngineName:         "mysql",
		MajorEngineVersion: "8.0",
		Description:        "stackyard option group",
		Options:            []string{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	s.optionGroups[id] = group
	return group
}

func (s *Service) ensurePlaceholderSubnetGroupLocked(name string) *DBSubnetGroup {
	id := strings.TrimSpace(name)
	if id == "" {
		return nil
	}
	if existing, ok := s.dbSubnetGroups[id]; ok {
		return existing
	}
	now := time.Now().UTC()
	group := &DBSubnetGroup{
		Name:        id,
		Description: "stackyard subnet group",
		SubnetIDs:   []string{"subnet-12345678"},
		Status:      "Complete",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.dbSubnetGroups[id] = group
	return group
}

func (s *Service) ensurePlaceholderClusterLocked(identifier string) *DBCluster {
	id := strings.TrimSpace(identifier)
	if id == "" {
		return nil
	}
	if existing, ok := s.clusters[id]; ok {
		return existing
	}
	now := time.Now().UTC()
	cluster := &DBCluster{
		Identifier:              id,
		ARN:                     dbClusterARN(id),
		Engine:                  "aurora-mysql",
		Status:                  "available",
		MasterUsername:          "admin",
		DatabaseName:            "stackyard",
		DBSubnetGroupName:       "stackyard",
		DBClusterParameterGroup: "default",
		Endpoint:                fmt.Sprintf("%s.cluster-%s.rds.amazonaws.com", id, defaultRegion),
		ReaderEndpoint:          fmt.Sprintf("%s.cluster-ro-%s.rds.amazonaws.com", id, defaultRegion),
		BackupRetentionPeriod:   1,
		VpcSecurityGroupIDs:     []string{"sg-12345678"},
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	s.clusters[id] = cluster
	return cluster
}

func (s *Service) ensurePlaceholderGlobalClusterLocked(identifier string) *GlobalCluster {
	id := strings.TrimSpace(identifier)
	if id == "" {
		return nil
	}
	if existing, ok := s.globalClusters[id]; ok {
		return existing
	}
	now := time.Now().UTC()
	cluster := &GlobalCluster{
		Identifier:         id,
		ARN:                globalClusterARN(id),
		Status:             "available",
		SourceDBClusterArn: dbClusterARN("stackyard"),
		Members:            []string{dbClusterARN("stackyard")},
		EngineVersion:      "8.0",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	s.globalClusters[id] = cluster
	return cluster
}

func (s *Service) ensurePlaceholderBlueGreenDeploymentLocked(identifier string) *BlueGreenDeployment {
	id := strings.TrimSpace(identifier)
	if id == "" {
		return nil
	}
	if existing, ok := s.blueGreen[id]; ok {
		return existing
	}
	now := time.Now().UTC()
	item := &BlueGreenDeployment{
		Identifier: id,
		Name:       id,
		Source:     "stackyard",
		Target:     "stackyard-green",
		Status:     "AVAILABLE",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.blueGreen[id] = item
	return item
}

func dbClusterARN(identifier string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:cluster:%s", defaultRegion, defaultAccountID, identifier)
}

func globalClusterARN(identifier string) string {
	return fmt.Sprintf("arn:aws:rds::%s:global-cluster:%s", defaultAccountID, identifier)
}

func dbClusterEndpointARN(clusterID, endpointID string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:cluster-endpoint:%s/%s", defaultRegion, defaultAccountID, clusterID, endpointID)
}

func blueGreenID(name string) string {
	id := strings.ToLower(strings.TrimSpace(name))
	id = strings.ReplaceAll(id, " ", "-")
	if id == "" {
		id = "bgd"
	}
	if len(id) > 48 {
		id = id[:48]
	}
	return "bgd-" + id
}

func tenantKey(clusterID, tenantID string) string {
	return strings.TrimSpace(clusterID) + "::" + strings.TrimSpace(tenantID)
}

func compactStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func appendUnique(values []string, value string) []string {
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}

func removeString(values []string, value string) []string {
	filtered := values[:0]
	for _, item := range values {
		if item != value {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func appendUniqueEC2Group(values []EC2SecurityGroup, value EC2SecurityGroup) []EC2SecurityGroup {
	for _, item := range values {
		if item.Name == value.Name && item.OwnerID == value.OwnerID {
			return values
		}
	}
	return append(values, value)
}

func cloneDBParameterGroup(group *DBParameterGroup) DBParameterGroup {
	if group == nil {
		return DBParameterGroup{}
	}
	out := *group
	out.Parameters = make(map[string]string, len(group.Parameters))
	for k, v := range group.Parameters {
		out.Parameters[k] = v
	}
	return out
}

func cloneOptionGroup(group *OptionGroup) OptionGroup {
	if group == nil {
		return OptionGroup{}
	}
	out := *group
	out.Options = append([]string{}, group.Options...)
	return out
}

func cloneDBSubnetGroup(group *DBSubnetGroup) DBSubnetGroup {
	if group == nil {
		return DBSubnetGroup{}
	}
	out := *group
	out.SubnetIDs = append([]string{}, group.SubnetIDs...)
	return out
}

func cloneDBSecurityGroup(group *DBSecurityGroup) DBSecurityGroup {
	if group == nil {
		return DBSecurityGroup{}
	}
	out := *group
	out.CIDRIPs = append([]string{}, group.CIDRIPs...)
	out.EC2SecurityGroups = append([]EC2SecurityGroup{}, group.EC2SecurityGroups...)
	return out
}

func cloneCertificate(cert *Certificate) Certificate {
	if cert == nil {
		return Certificate{}
	}
	return *cert
}

func cloneDBCluster(cluster *DBCluster) DBCluster {
	if cluster == nil {
		return DBCluster{}
	}
	out := *cluster
	out.VpcSecurityGroupIDs = append([]string{}, cluster.VpcSecurityGroupIDs...)
	return out
}

func cloneDBClusterEndpoint(endpoint *DBClusterEndpoint) DBClusterEndpoint {
	if endpoint == nil {
		return DBClusterEndpoint{}
	}
	out := *endpoint
	out.StaticMembers = append([]string{}, endpoint.StaticMembers...)
	out.ExcludedMembers = append([]string{}, endpoint.ExcludedMembers...)
	return out
}

func cloneGlobalCluster(cluster *GlobalCluster) GlobalCluster {
	if cluster == nil {
		return GlobalCluster{}
	}
	out := *cluster
	out.Members = append([]string{}, cluster.Members...)
	return out
}

func cloneBlueGreenDeployment(item *BlueGreenDeployment) BlueGreenDeployment {
	if item == nil {
		return BlueGreenDeployment{}
	}
	return *item
}

func cloneTenantDatabase(item *TenantDatabase) TenantDatabase {
	if item == nil {
		return TenantDatabase{}
	}
	return *item
}
