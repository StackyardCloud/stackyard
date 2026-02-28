package timestreaminfluxdb

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrNotFound         = errors.New("resource not found")
	ErrConflict         = errors.New("resource conflict")
)

const (
	DefaultRegion    = "us-east-1"
	DefaultAccountID = "123456789012"
)

type DbCluster struct {
	ID                            string
	Name                          string
	Arn                           string
	Status                        string
	Endpoint                      string
	ReaderEndpoint                string
	Port                          int
	DeploymentType                string
	DbInstanceType                string
	NetworkType                   string
	DbStorageType                 string
	AllocatedStorage              int
	EngineType                    string
	PubliclyAccessible            bool
	DbParameterGroupIdentifier    string
	LogDeliveryConfiguration      map[string]any
	InfluxAuthParametersSecretArn string
	VpcSubnetIds                  []string
	VpcSecurityGroupIds           []string
	FailoverMode                  string
	InstanceIDs                   []string
}

type DbInstance struct {
	ID                            string
	Name                          string
	Arn                           string
	Status                        string
	Endpoint                      string
	Port                          int
	NetworkType                   string
	DbInstanceType                string
	DbStorageType                 string
	AllocatedStorage              int
	DeploymentType                string
	VpcSubnetIds                  []string
	PubliclyAccessible            bool
	VpcSecurityGroupIds           []string
	DbParameterGroupIdentifier    string
	AvailabilityZone              string
	SecondaryAvailabilityZone     string
	LogDeliveryConfiguration      map[string]any
	InfluxAuthParametersSecretArn string
	DbClusterID                   string
	InstanceMode                  string
	InstanceModes                 []string
}

type DbParameterGroup struct {
	ID          string
	Name        string
	Arn         string
	Description string
	Parameters  map[string]any
}

type CreateDbClusterInput struct {
	Name                       string
	DbInstanceType             string
	VpcSubnetIds               []string
	VpcSecurityGroupIds        []string
	Port                       int
	DbParameterGroupIdentifier string
	DeploymentType             string
	NetworkType                string
	DbStorageType              string
	AllocatedStorage           int
	EngineType                 string
	PubliclyAccessible         *bool
	FailoverMode               string
	LogDeliveryConfiguration   map[string]any
	Tags                       map[string]string
}

type UpdateDbClusterInput struct {
	DbClusterID                 string
	LogDeliveryConfiguration    map[string]any
	LogDeliveryConfigurationSet bool
	DbParameterGroupIdentifier  *string
	Port                        *int
	DbInstanceType              *string
	FailoverMode                *string
}

type CreateDbInstanceInput struct {
	Name                       string
	Password                   string
	DbInstanceType             string
	VpcSubnetIds               []string
	VpcSecurityGroupIds        []string
	AllocatedStorage           int
	Port                       int
	DbParameterGroupIdentifier string
	DeploymentType             string
	NetworkType                string
	DbStorageType              string
	PubliclyAccessible         *bool
	LogDeliveryConfiguration   map[string]any
	Tags                       map[string]string
}

type UpdateDbInstanceInput struct {
	Identifier                  string
	DbInstanceType              *string
	AllocatedStorage            *int
	Port                        *int
	DbParameterGroupIdentifier  *string
	DeploymentType              *string
	DbStorageType               *string
	LogDeliveryConfiguration    map[string]any
	LogDeliveryConfigurationSet bool
}

type CreateDbParameterGroupInput struct {
	Name        string
	Description string
	Parameters  map[string]any
	Tags        map[string]string
}

type Service struct {
	mu sync.Mutex

	clusterSeq  int
	instanceSeq int
	pgSeq       int

	clusters               map[string]*DbCluster
	clusterNameToID        map[string]string
	instances              map[string]*DbInstance
	instanceNameToID       map[string]string
	parameterGroups        map[string]*DbParameterGroup
	parameterGroupNameToID map[string]string
	resourceTags           map[string]map[string]string
}

func NewService() *Service {
	s := &Service{
		clusters:               map[string]*DbCluster{},
		clusterNameToID:        map[string]string{},
		instances:              map[string]*DbInstance{},
		instanceNameToID:       map[string]string{},
		parameterGroups:        map[string]*DbParameterGroup{},
		parameterGroupNameToID: map[string]string{},
		resourceTags:           map[string]map[string]string{},
	}
	s.seedDefaultParameterGroup()
	return s
}

func (s *Service) seedDefaultParameterGroup() {
	pg := &DbParameterGroup{
		ID:          "dbparamgrp-000000000001",
		Name:        "default",
		Arn:         dbParameterGroupARN("dbparamgrp-000000000001"),
		Description: "Default Timestream for InfluxDB parameter group",
		Parameters:  map[string]any{},
	}
	s.parameterGroups[pg.ID] = pg
	s.parameterGroupNameToID[pg.Name] = pg.ID
	s.pgSeq = 1
	s.resourceTags[pg.Arn] = map[string]string{}
}

func (s *Service) CreateDbCluster(input CreateDbClusterInput) (DbCluster, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	instanceType := strings.TrimSpace(input.DbInstanceType)
	if name == "" || instanceType == "" || len(input.VpcSubnetIds) == 0 || len(input.VpcSecurityGroupIds) == 0 {
		return DbCluster{}, "", ErrInvalidParameter
	}
	if !validateTags(input.Tags) {
		return DbCluster{}, "", ErrInvalidParameter
	}
	if _, exists := s.clusterNameToID[name]; exists {
		return DbCluster{}, "", ErrConflict
	}

	parameterGroupID := strings.TrimSpace(input.DbParameterGroupIdentifier)
	if parameterGroupID == "" {
		parameterGroupID = "dbparamgrp-000000000001"
	}
	if _, ok := s.lookupParameterGroupLocked(parameterGroupID); !ok {
		return DbCluster{}, "", ErrNotFound
	}

	s.clusterSeq++
	clusterID := fmt.Sprintf("dbcluster-%012d", s.clusterSeq)
	clusterARN := dbClusterARN(clusterID)
	port := input.Port
	if port == 0 {
		port = 8086
	}
	if !isValidPort(port) {
		return DbCluster{}, "", ErrInvalidParameter
	}
	publiclyAccessible := false
	if input.PubliclyAccessible != nil {
		publiclyAccessible = *input.PubliclyAccessible
	}
	deploymentType := strings.TrimSpace(input.DeploymentType)
	if deploymentType == "" {
		deploymentType = "SINGLE_AZ"
	}
	networkType := strings.TrimSpace(input.NetworkType)
	if networkType == "" {
		networkType = "IPV4"
	}
	dbStorageType := strings.TrimSpace(input.DbStorageType)
	if dbStorageType == "" {
		dbStorageType = "INFLUX_IO_INCLUDED_3000_IOPS"
	}
	allocatedStorage := input.AllocatedStorage
	if allocatedStorage == 0 {
		allocatedStorage = 20
	}
	if !isValidAllocatedStorage(allocatedStorage) {
		return DbCluster{}, "", ErrInvalidParameter
	}
	engineType := strings.TrimSpace(input.EngineType)
	if engineType == "" {
		engineType = "InfluxDBv2"
	}
	failoverMode := strings.TrimSpace(input.FailoverMode)
	if failoverMode == "" {
		failoverMode = "AUTOMATIC"
	}

	cluster := &DbCluster{
		ID:                            clusterID,
		Name:                          name,
		Arn:                           clusterARN,
		Status:                        "AVAILABLE",
		Endpoint:                      fmt.Sprintf("%s.cluster.timestream-influxdb.%s.amazonaws.com", clusterID, DefaultRegion),
		ReaderEndpoint:                fmt.Sprintf("%s.reader.timestream-influxdb.%s.amazonaws.com", clusterID, DefaultRegion),
		Port:                          port,
		DeploymentType:                deploymentType,
		DbInstanceType:                instanceType,
		NetworkType:                   networkType,
		DbStorageType:                 dbStorageType,
		AllocatedStorage:              allocatedStorage,
		EngineType:                    engineType,
		PubliclyAccessible:            publiclyAccessible,
		DbParameterGroupIdentifier:    parameterGroupID,
		LogDeliveryConfiguration:      cloneAnyMap(input.LogDeliveryConfiguration),
		InfluxAuthParametersSecretArn: fmt.Sprintf("arn:aws:secretsmanager:%s:%s:secret:timestream-influxdb/%s", DefaultRegion, DefaultAccountID, clusterID),
		VpcSubnetIds:                  cloneStrings(input.VpcSubnetIds),
		VpcSecurityGroupIds:           cloneStrings(input.VpcSecurityGroupIds),
		FailoverMode:                  failoverMode,
	}

	instance := s.newPrimaryInstanceLocked(cluster)
	cluster.InstanceIDs = []string{instance.ID}

	s.clusters[clusterID] = cluster
	s.clusterNameToID[name] = clusterID
	s.resourceTags[clusterARN] = cloneStringMap(input.Tags)
	return cloneCluster(cluster), "CREATING", nil
}

func (s *Service) UpdateDbCluster(input UpdateDbClusterInput) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.lookupClusterLocked(input.DbClusterID)
	if !ok {
		return "", lookupErr(input.DbClusterID)
	}

	if input.DbParameterGroupIdentifier != nil {
		value := strings.TrimSpace(*input.DbParameterGroupIdentifier)
		if value == "" {
			return "", ErrInvalidParameter
		}
		if _, ok := s.lookupParameterGroupLocked(value); !ok {
			return "", ErrNotFound
		}
		cluster.DbParameterGroupIdentifier = value
	}
	if input.Port != nil {
		if !isValidPort(*input.Port) {
			return "", ErrInvalidParameter
		}
		cluster.Port = *input.Port
	}
	if input.DbInstanceType != nil {
		value := strings.TrimSpace(*input.DbInstanceType)
		if value == "" {
			return "", ErrInvalidParameter
		}
		cluster.DbInstanceType = value
	}
	if input.FailoverMode != nil {
		value := strings.TrimSpace(*input.FailoverMode)
		if value == "" {
			return "", ErrInvalidParameter
		}
		cluster.FailoverMode = value
	}
	if input.LogDeliveryConfigurationSet {
		cluster.LogDeliveryConfiguration = cloneAnyMap(input.LogDeliveryConfiguration)
	}
	for _, instanceID := range cluster.InstanceIDs {
		instance, ok := s.instances[instanceID]
		if !ok {
			continue
		}
		if input.DbParameterGroupIdentifier != nil {
			instance.DbParameterGroupIdentifier = cluster.DbParameterGroupIdentifier
		}
		if input.Port != nil {
			instance.Port = cluster.Port
		}
		if input.DbInstanceType != nil {
			instance.DbInstanceType = cluster.DbInstanceType
		}
		if input.LogDeliveryConfigurationSet {
			instance.LogDeliveryConfiguration = cloneAnyMap(cluster.LogDeliveryConfiguration)
		}
	}
	cluster.Status = "UPDATING"
	return cluster.Status, nil
}

func (s *Service) RebootDbCluster(dbClusterID string, instanceIDs []string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.lookupClusterLocked(dbClusterID)
	if !ok {
		return "", lookupErr(dbClusterID)
	}

	if len(instanceIDs) > 0 {
		valid := map[string]struct{}{}
		for _, instanceID := range cluster.InstanceIDs {
			valid[instanceID] = struct{}{}
		}
		for _, requested := range instanceIDs {
			id := strings.TrimSpace(requested)
			if id == "" {
				return "", ErrInvalidParameter
			}
			if _, exists := valid[id]; !exists {
				return "", ErrNotFound
			}
		}
	}

	cluster.Status = "REBOOTING"
	for _, instanceID := range cluster.InstanceIDs {
		if instance, ok := s.instances[instanceID]; ok {
			instance.Status = "REBOOTING"
		}
	}
	return cluster.Status, nil
}

func (s *Service) DeleteDbCluster(dbClusterID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.lookupClusterLocked(dbClusterID)
	if !ok {
		return "", lookupErr(dbClusterID)
	}

	delete(s.clusterNameToID, cluster.Name)
	delete(s.resourceTags, cluster.Arn)
	for _, instanceID := range cluster.InstanceIDs {
		if instance, ok := s.instances[instanceID]; ok {
			delete(s.instanceNameToID, instance.Name)
			delete(s.resourceTags, instance.Arn)
		}
		delete(s.instances, instanceID)
	}
	delete(s.clusters, cluster.ID)
	return "DELETING", nil
}

func (s *Service) CreateDbInstance(input CreateDbInstanceInput) (DbInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	password := strings.TrimSpace(input.Password)
	dbInstanceType := strings.TrimSpace(input.DbInstanceType)
	if name == "" || password == "" || dbInstanceType == "" || len(input.VpcSubnetIds) == 0 || len(input.VpcSecurityGroupIds) == 0 {
		return DbInstance{}, ErrInvalidParameter
	}
	if !validateTags(input.Tags) {
		return DbInstance{}, ErrInvalidParameter
	}
	if _, exists := s.instanceNameToID[name]; exists {
		return DbInstance{}, ErrConflict
	}

	port := input.Port
	if port == 0 {
		port = 8086
	}
	if !isValidPort(port) {
		return DbInstance{}, ErrInvalidParameter
	}
	allocatedStorage := input.AllocatedStorage
	if allocatedStorage == 0 {
		allocatedStorage = 20
	}
	if !isValidAllocatedStorage(allocatedStorage) {
		return DbInstance{}, ErrInvalidParameter
	}

	parameterGroupID := strings.TrimSpace(input.DbParameterGroupIdentifier)
	if parameterGroupID == "" {
		parameterGroupID = "dbparamgrp-000000000001"
	}
	if _, ok := s.lookupParameterGroupLocked(parameterGroupID); !ok {
		return DbInstance{}, ErrNotFound
	}

	deploymentType := strings.TrimSpace(input.DeploymentType)
	if deploymentType == "" {
		deploymentType = "SINGLE_AZ"
	}
	networkType := strings.TrimSpace(input.NetworkType)
	if networkType == "" {
		networkType = "IPV4"
	}
	dbStorageType := strings.TrimSpace(input.DbStorageType)
	if dbStorageType == "" {
		dbStorageType = "INFLUX_IO_INCLUDED_3000_IOPS"
	}
	publiclyAccessible := false
	if input.PubliclyAccessible != nil {
		publiclyAccessible = *input.PubliclyAccessible
	}

	s.instanceSeq++
	instanceID := fmt.Sprintf("dbinstance-%012d", s.instanceSeq)
	instance := &DbInstance{
		ID:                            instanceID,
		Name:                          name,
		Arn:                           dbInstanceARN(instanceID),
		Status:                        "CREATING",
		Endpoint:                      fmt.Sprintf("%s.instance.timestream-influxdb.%s.amazonaws.com", instanceID, DefaultRegion),
		Port:                          port,
		NetworkType:                   networkType,
		DbInstanceType:                dbInstanceType,
		DbStorageType:                 dbStorageType,
		AllocatedStorage:              allocatedStorage,
		DeploymentType:                deploymentType,
		VpcSubnetIds:                  cloneStrings(input.VpcSubnetIds),
		PubliclyAccessible:            publiclyAccessible,
		VpcSecurityGroupIds:           cloneStrings(input.VpcSecurityGroupIds),
		DbParameterGroupIdentifier:    parameterGroupID,
		AvailabilityZone:              "us-east-1a",
		SecondaryAvailabilityZone:     "us-east-1b",
		LogDeliveryConfiguration:      cloneAnyMap(input.LogDeliveryConfiguration),
		InfluxAuthParametersSecretArn: fmt.Sprintf("arn:aws:secretsmanager:%s:%s:secret:timestream-influxdb/%s", DefaultRegion, DefaultAccountID, instanceID),
		InstanceMode:                  "PRIMARY",
		InstanceModes:                 []string{"PRIMARY"},
	}
	s.instances[instanceID] = instance
	s.instanceNameToID[name] = instanceID
	s.resourceTags[instance.Arn] = cloneStringMap(input.Tags)
	return cloneInstance(instance), nil
}

func (s *Service) UpdateDbInstance(input UpdateDbInstanceInput) (DbInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.lookupInstanceLocked(input.Identifier)
	if !ok {
		return DbInstance{}, lookupErr(input.Identifier)
	}

	if input.DbInstanceType != nil {
		value := strings.TrimSpace(*input.DbInstanceType)
		if value == "" {
			return DbInstance{}, ErrInvalidParameter
		}
		instance.DbInstanceType = value
	}
	if input.AllocatedStorage != nil {
		if !isValidAllocatedStorage(*input.AllocatedStorage) {
			return DbInstance{}, ErrInvalidParameter
		}
		instance.AllocatedStorage = *input.AllocatedStorage
	}
	if input.Port != nil {
		if !isValidPort(*input.Port) {
			return DbInstance{}, ErrInvalidParameter
		}
		instance.Port = *input.Port
	}
	if input.DbParameterGroupIdentifier != nil {
		value := strings.TrimSpace(*input.DbParameterGroupIdentifier)
		if value == "" {
			return DbInstance{}, ErrInvalidParameter
		}
		if _, ok := s.lookupParameterGroupLocked(value); !ok {
			return DbInstance{}, ErrNotFound
		}
		instance.DbParameterGroupIdentifier = value
	}
	if input.DeploymentType != nil {
		value := strings.TrimSpace(*input.DeploymentType)
		if value == "" {
			return DbInstance{}, ErrInvalidParameter
		}
		instance.DeploymentType = value
	}
	if input.DbStorageType != nil {
		value := strings.TrimSpace(*input.DbStorageType)
		if value == "" {
			return DbInstance{}, ErrInvalidParameter
		}
		instance.DbStorageType = value
	}
	if input.LogDeliveryConfigurationSet {
		instance.LogDeliveryConfiguration = cloneAnyMap(input.LogDeliveryConfiguration)
	}

	instance.Status = "UPDATING"
	return cloneInstance(instance), nil
}

func (s *Service) RebootDbInstance(identifier string) (DbInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.lookupInstanceLocked(identifier)
	if !ok {
		return DbInstance{}, lookupErr(identifier)
	}
	instance.Status = "REBOOTING"
	return cloneInstance(instance), nil
}

func (s *Service) DeleteDbInstance(identifier string) (DbInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.lookupInstanceLocked(identifier)
	if !ok {
		return DbInstance{}, lookupErr(identifier)
	}
	out := cloneInstance(instance)
	out.Status = "DELETING"

	if cluster, ok := s.clusters[instance.DbClusterID]; ok {
		cluster.InstanceIDs = removeString(cluster.InstanceIDs, instance.ID)
	}
	delete(s.instances, instance.ID)
	delete(s.instanceNameToID, instance.Name)
	delete(s.resourceTags, instance.Arn)
	return out, nil
}

func (s *Service) CreateDbParameterGroup(input CreateDbParameterGroupInput) (DbParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return DbParameterGroup{}, ErrInvalidParameter
	}
	if !validateTags(input.Tags) {
		return DbParameterGroup{}, ErrInvalidParameter
	}
	if _, exists := s.parameterGroupNameToID[name]; exists {
		return DbParameterGroup{}, ErrConflict
	}

	s.pgSeq++
	id := fmt.Sprintf("dbparamgrp-%012d", s.pgSeq)
	pg := &DbParameterGroup{
		ID:          id,
		Name:        name,
		Arn:         dbParameterGroupARN(id),
		Description: strings.TrimSpace(input.Description),
		Parameters:  cloneAnyMap(input.Parameters),
	}
	s.parameterGroups[pg.ID] = pg
	s.parameterGroupNameToID[pg.Name] = pg.ID
	s.resourceTags[pg.Arn] = cloneStringMap(input.Tags)
	return cloneParameterGroup(pg), nil
}

func (s *Service) GetDbCluster(dbClusterID string) (DbCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.lookupClusterLocked(dbClusterID)
	if !ok {
		return DbCluster{}, lookupErr(dbClusterID)
	}
	return cloneCluster(cluster), nil
}

func (s *Service) ListDbClusters(nextToken string, maxResults int) ([]DbCluster, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	start, limit, err := paginationWindow(nextToken, maxResults, len(s.clusters))
	if err != nil {
		return nil, "", err
	}

	keys := make([]string, 0, len(s.clusters))
	for id := range s.clusters {
		keys = append(keys, id)
	}
	sort.Strings(keys)

	end := start + limit
	if end > len(keys) {
		end = len(keys)
	}
	items := make([]DbCluster, 0, end-start)
	for _, id := range keys[start:end] {
		items = append(items, cloneCluster(s.clusters[id]))
	}

	outToken := ""
	if end < len(keys) {
		outToken = strconv.Itoa(end)
	}
	return items, outToken, nil
}

func (s *Service) GetDbInstance(identifier string) (DbInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, ok := s.lookupInstanceLocked(identifier)
	if !ok {
		return DbInstance{}, lookupErr(identifier)
	}
	return cloneInstance(instance), nil
}

func (s *Service) ListDbInstances(nextToken string, maxResults int) ([]DbInstance, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	start, limit, err := paginationWindow(nextToken, maxResults, len(s.instances))
	if err != nil {
		return nil, "", err
	}

	keys := make([]string, 0, len(s.instances))
	for id := range s.instances {
		keys = append(keys, id)
	}
	sort.Strings(keys)

	end := start + limit
	if end > len(keys) {
		end = len(keys)
	}
	items := make([]DbInstance, 0, end-start)
	for _, id := range keys[start:end] {
		items = append(items, cloneInstance(s.instances[id]))
	}

	outToken := ""
	if end < len(keys) {
		outToken = strconv.Itoa(end)
	}
	return items, outToken, nil
}

func (s *Service) ListDbInstancesForCluster(dbClusterID, nextToken string, maxResults int) ([]DbInstance, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.lookupClusterLocked(dbClusterID)
	if !ok {
		return nil, "", lookupErr(dbClusterID)
	}
	start, limit, err := paginationWindow(nextToken, maxResults, len(cluster.InstanceIDs))
	if err != nil {
		return nil, "", err
	}

	instanceIDs := cloneStrings(cluster.InstanceIDs)
	sort.Strings(instanceIDs)

	end := start + limit
	if end > len(instanceIDs) {
		end = len(instanceIDs)
	}
	items := make([]DbInstance, 0, end-start)
	for _, instanceID := range instanceIDs[start:end] {
		if instance, ok := s.instances[instanceID]; ok {
			items = append(items, cloneInstance(instance))
		}
	}

	outToken := ""
	if end < len(instanceIDs) {
		outToken = strconv.Itoa(end)
	}
	return items, outToken, nil
}

func (s *Service) GetDbParameterGroup(identifier string) (DbParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	group, ok := s.lookupParameterGroupLocked(identifier)
	if !ok {
		return DbParameterGroup{}, lookupErr(identifier)
	}
	return cloneParameterGroup(group), nil
}

func (s *Service) ListDbParameterGroups(nextToken string, maxResults int) ([]DbParameterGroup, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	start, limit, err := paginationWindow(nextToken, maxResults, len(s.parameterGroups))
	if err != nil {
		return nil, "", err
	}

	keys := make([]string, 0, len(s.parameterGroups))
	for id := range s.parameterGroups {
		keys = append(keys, id)
	}
	sort.Strings(keys)

	end := start + limit
	if end > len(keys) {
		end = len(keys)
	}
	items := make([]DbParameterGroup, 0, end-start)
	for _, id := range keys[start:end] {
		items = append(items, cloneParameterGroup(s.parameterGroups[id]))
	}

	outToken := ""
	if end < len(keys) {
		outToken = strconv.Itoa(end)
	}
	return items, outToken, nil
}

func (s *Service) ListTagsForResource(resourceARN string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	arn := strings.TrimSpace(resourceARN)
	if !isValidResourceARN(arn) {
		return nil, ErrInvalidParameter
	}
	tags, ok := s.resourceTags[arn]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneStringMap(tags), nil
}

func (s *Service) TagResource(resourceARN string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	arn := strings.TrimSpace(resourceARN)
	if !isValidResourceARN(arn) || !validateTags(tags) || len(tags) == 0 {
		return ErrInvalidParameter
	}
	resourceTags, ok := s.resourceTags[arn]
	if !ok {
		return ErrNotFound
	}
	for key, value := range tags {
		resourceTags[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return nil
}

func (s *Service) UntagResource(resourceARN string, tagKeys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	arn := strings.TrimSpace(resourceARN)
	keys := normalizeTagKeys(tagKeys)
	if !isValidResourceARN(arn) || len(keys) == 0 {
		return ErrInvalidParameter
	}
	resourceTags, ok := s.resourceTags[arn]
	if !ok {
		return ErrNotFound
	}
	for _, key := range keys {
		delete(resourceTags, key)
	}
	return nil
}

func (s *Service) newPrimaryInstanceLocked(cluster *DbCluster) *DbInstance {
	s.instanceSeq++
	instanceID := fmt.Sprintf("dbinstance-%012d", s.instanceSeq)
	instance := &DbInstance{
		ID:                            instanceID,
		Name:                          fmt.Sprintf("%s-primary", cluster.Name),
		Arn:                           dbInstanceARN(instanceID),
		Status:                        "AVAILABLE",
		Endpoint:                      fmt.Sprintf("%s.instance.timestream-influxdb.%s.amazonaws.com", instanceID, DefaultRegion),
		Port:                          cluster.Port,
		NetworkType:                   cluster.NetworkType,
		DbInstanceType:                cluster.DbInstanceType,
		DbStorageType:                 cluster.DbStorageType,
		AllocatedStorage:              cluster.AllocatedStorage,
		DeploymentType:                "SINGLE_INSTANCE",
		VpcSubnetIds:                  cloneStrings(cluster.VpcSubnetIds),
		PubliclyAccessible:            cluster.PubliclyAccessible,
		VpcSecurityGroupIds:           cloneStrings(cluster.VpcSecurityGroupIds),
		DbParameterGroupIdentifier:    cluster.DbParameterGroupIdentifier,
		AvailabilityZone:              "us-east-1a",
		SecondaryAvailabilityZone:     "us-east-1b",
		LogDeliveryConfiguration:      cloneAnyMap(cluster.LogDeliveryConfiguration),
		InfluxAuthParametersSecretArn: cluster.InfluxAuthParametersSecretArn,
		DbClusterID:                   cluster.ID,
		InstanceMode:                  "PRIMARY",
		InstanceModes:                 []string{"PRIMARY"},
	}
	s.instances[instanceID] = instance
	s.instanceNameToID[instance.Name] = instance.ID
	s.resourceTags[instance.Arn] = map[string]string{}
	return instance
}

func (s *Service) lookupClusterLocked(identifier string) (*DbCluster, bool) {
	key := strings.TrimSpace(identifier)
	if key == "" {
		return nil, false
	}
	if strings.HasPrefix(key, "arn:") {
		if parsed := parseTrailingID(key); parsed != "" {
			key = parsed
		}
	}
	if cluster, ok := s.clusters[key]; ok {
		return cluster, true
	}
	if id, ok := s.clusterNameToID[key]; ok {
		cluster, exists := s.clusters[id]
		return cluster, exists
	}
	return nil, false
}

func (s *Service) lookupInstanceLocked(identifier string) (*DbInstance, bool) {
	key := strings.TrimSpace(identifier)
	if key == "" {
		return nil, false
	}
	if strings.HasPrefix(key, "arn:") {
		if parsed := parseTrailingID(key); parsed != "" {
			key = parsed
		}
	}
	if instance, ok := s.instances[key]; ok {
		return instance, true
	}
	if id, ok := s.instanceNameToID[key]; ok {
		instance, exists := s.instances[id]
		return instance, exists
	}
	return nil, false
}

func (s *Service) lookupParameterGroupLocked(identifier string) (*DbParameterGroup, bool) {
	key := strings.TrimSpace(identifier)
	if key == "" {
		return nil, false
	}
	if strings.HasPrefix(key, "arn:") {
		if parsed := parseTrailingID(key); parsed != "" {
			key = parsed
		}
	}
	if group, ok := s.parameterGroups[key]; ok {
		return group, true
	}
	if id, ok := s.parameterGroupNameToID[key]; ok {
		group, exists := s.parameterGroups[id]
		return group, exists
	}
	return nil, false
}

func paginationWindow(nextToken string, maxResults int, total int) (int, int, error) {
	start := 0
	if token := strings.TrimSpace(nextToken); token != "" {
		value, err := strconv.Atoi(token)
		if err != nil || value < 0 || value > total {
			return 0, 0, ErrInvalidParameter
		}
		start = value
	}

	limit := maxResults
	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 100 {
		return 0, 0, ErrInvalidParameter
	}
	return start, limit, nil
}

func lookupErr(identifier string) error {
	if strings.TrimSpace(identifier) == "" {
		return ErrInvalidParameter
	}
	return ErrNotFound
}

func parseTrailingID(arn string) string {
	idx := strings.LastIndex(strings.TrimSpace(arn), "/")
	if idx == -1 || idx+1 >= len(arn) {
		return ""
	}
	return strings.TrimSpace(arn[idx+1:])
}

func dbClusterARN(id string) string {
	return fmt.Sprintf("arn:aws:timestream-influxdb:%s:%s:db-cluster/%s", DefaultRegion, DefaultAccountID, strings.TrimSpace(id))
}

func dbInstanceARN(id string) string {
	return fmt.Sprintf("arn:aws:timestream-influxdb:%s:%s:db-instance/%s", DefaultRegion, DefaultAccountID, strings.TrimSpace(id))
}

func dbParameterGroupARN(id string) string {
	return fmt.Sprintf("arn:aws:timestream-influxdb:%s:%s:db-parameter-group/%s", DefaultRegion, DefaultAccountID, strings.TrimSpace(id))
}

func cloneCluster(in *DbCluster) DbCluster {
	if in == nil {
		return DbCluster{}
	}
	return DbCluster{
		ID:                            in.ID,
		Name:                          in.Name,
		Arn:                           in.Arn,
		Status:                        in.Status,
		Endpoint:                      in.Endpoint,
		ReaderEndpoint:                in.ReaderEndpoint,
		Port:                          in.Port,
		DeploymentType:                in.DeploymentType,
		DbInstanceType:                in.DbInstanceType,
		NetworkType:                   in.NetworkType,
		DbStorageType:                 in.DbStorageType,
		AllocatedStorage:              in.AllocatedStorage,
		EngineType:                    in.EngineType,
		PubliclyAccessible:            in.PubliclyAccessible,
		DbParameterGroupIdentifier:    in.DbParameterGroupIdentifier,
		LogDeliveryConfiguration:      cloneAnyMap(in.LogDeliveryConfiguration),
		InfluxAuthParametersSecretArn: in.InfluxAuthParametersSecretArn,
		VpcSubnetIds:                  cloneStrings(in.VpcSubnetIds),
		VpcSecurityGroupIds:           cloneStrings(in.VpcSecurityGroupIds),
		FailoverMode:                  in.FailoverMode,
		InstanceIDs:                   cloneStrings(in.InstanceIDs),
	}
}

func cloneInstance(in *DbInstance) DbInstance {
	if in == nil {
		return DbInstance{}
	}
	return DbInstance{
		ID:                            in.ID,
		Name:                          in.Name,
		Arn:                           in.Arn,
		Status:                        in.Status,
		Endpoint:                      in.Endpoint,
		Port:                          in.Port,
		NetworkType:                   in.NetworkType,
		DbInstanceType:                in.DbInstanceType,
		DbStorageType:                 in.DbStorageType,
		AllocatedStorage:              in.AllocatedStorage,
		DeploymentType:                in.DeploymentType,
		VpcSubnetIds:                  cloneStrings(in.VpcSubnetIds),
		PubliclyAccessible:            in.PubliclyAccessible,
		VpcSecurityGroupIds:           cloneStrings(in.VpcSecurityGroupIds),
		DbParameterGroupIdentifier:    in.DbParameterGroupIdentifier,
		AvailabilityZone:              in.AvailabilityZone,
		SecondaryAvailabilityZone:     in.SecondaryAvailabilityZone,
		LogDeliveryConfiguration:      cloneAnyMap(in.LogDeliveryConfiguration),
		InfluxAuthParametersSecretArn: in.InfluxAuthParametersSecretArn,
		DbClusterID:                   in.DbClusterID,
		InstanceMode:                  in.InstanceMode,
		InstanceModes:                 cloneStrings(in.InstanceModes),
	}
}

func cloneParameterGroup(in *DbParameterGroup) DbParameterGroup {
	if in == nil {
		return DbParameterGroup{}
	}
	return DbParameterGroup{
		ID:          in.ID,
		Name:        in.Name,
		Arn:         in.Arn,
		Description: in.Description,
		Parameters:  cloneAnyMap(in.Parameters),
	}
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

var timestreamInfluxDBResourceARNPattern = regexp.MustCompile(`^arn:aws[a-z\-]*:timestream-influxdb:[a-z0-9\-]+:[0-9]{12}:(db-cluster|db-instance|db-parameter-group)/[A-Za-z0-9\-]{3,64}$`)

func isValidPort(port int) bool {
	return port >= 1024 && port <= 65535
}

func isValidAllocatedStorage(value int) bool {
	return value >= 20 && value <= 16384
}

func isValidResourceARN(arn string) bool {
	return timestreamInfluxDBResourceARNPattern.MatchString(strings.TrimSpace(arn))
}

func validateTags(tags map[string]string) bool {
	if len(tags) > 50 {
		return false
	}
	for key, value := range tags {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

func normalizeTagKeys(tagKeys []string) []string {
	if len(tagKeys) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tagKeys))
	keys := make([]string, 0, len(tagKeys))
	for _, key := range tagKeys {
		normalized := strings.TrimSpace(key)
		if normalized == "" {
			return nil
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		keys = append(keys, normalized)
	}
	return keys
}

func removeString(values []string, target string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == target {
			continue
		}
		out = append(out, value)
	}
	return out
}
