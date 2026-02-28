package rds

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type EventSubscription struct {
	Name            string
	Arn             string
	SnsTopicArn     string
	SourceType      string
	SourceIDs       []string
	EventCategories []string
	Enabled         bool
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CreateEventSubscriptionInput struct {
	Name            string
	SnsTopicArn     string
	SourceType      string
	SourceIDs       []string
	EventCategories []string
	Enabled         bool
}

type DescribeEventSubscriptionsInput struct {
	Name       string
	MaxRecords int
	Marker     string
}

type ModifyEventSubscriptionInput struct {
	Name            string
	SnsTopicArn     string
	SourceType      string
	SourceIDs       []string
	EventCategories []string
	Enabled         *bool
}

type PendingMaintenanceAction struct {
	ResourceIdentifier string
	ApplyAction        string
	Description        string
	OptInStatus        string
	CurrentApplyDate   time.Time
}

type DescribePendingMaintenanceActionsInput struct {
	ResourceIdentifier string
	MaxRecords         int
	Marker             string
}

type ApplyPendingMaintenanceActionInput struct {
	ResourceIdentifier string
	ApplyAction        string
	OptInType          string
}

type EventRecord struct {
	SourceIdentifier string
	SourceType       string
	Date             time.Time
	Message          string
	EventCategories  []string
}

type DescribeEventsInput struct {
	SourceIdentifier string
	SourceType       string
	Duration         int
	MaxRecords       int
	Marker           string
}

type AccountAttribute struct {
	Name   string
	Values []string
}

type DBEngineVersion struct {
	Engine                 string
	EngineVersion          string
	DBParameterGroupFamily string
	Status                 string
}

type DescribeDBEngineVersionsInput struct {
	Engine        string
	EngineVersion string
	MaxRecords    int
	Marker        string
}

type OrderableDBInstanceOption struct {
	Engine          string
	EngineVersion   string
	DBInstanceClass string
	LicenseModel    string
	Vpc             bool
}

type DescribeOrderableDBInstanceOptionsInput struct {
	Engine          string
	EngineVersion   string
	DBInstanceClass string
	LicenseModel    string
	Vpc             *bool
	MaxRecords      int
	Marker          string
}

type SourceRegion struct {
	RegionName         string
	Endpoint           string
	Status             string
	SupportsDBInstance bool
}

type DescribeSourceRegionsInput struct {
	RegionName string
	MaxRecords int
	Marker     string
}

type ValidDBInstanceModification struct {
	DBInstanceClass string
	Storage         int
	StorageType     string
}

type ActivityStream struct {
	ResourceArn string
	Mode        string
	KmsKeyID    string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type StartActivityStreamInput struct {
	ResourceArn string
	Mode        string
	KmsKeyID    string
}

type StopActivityStreamInput struct {
	ResourceArn      string
	ApplyImmediately bool
}

type DBProxyAuth struct {
	AuthScheme string
	SecretArn  string
	IAMAuth    string
}

type DBProxy struct {
	Name                string
	Arn                 string
	EngineFamily        string
	RoleArn             string
	VpcSubnetIDs        []string
	VpcSecurityGroupIDs []string
	RequireTLS          bool
	IdleClientTimeout   int
	DebugLogging        bool
	Status              string
	Auth                []DBProxyAuth
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type CreateDBProxyInput struct {
	Name                string
	EngineFamily        string
	RoleArn             string
	VpcSubnetIDs        []string
	VpcSecurityGroupIDs []string
	RequireTLS          bool
	IdleClientTimeout   int
	DebugLogging        bool
	Auth                []DBProxyAuth
}

type DescribeDBProxiesInput struct {
	Name       string
	MaxRecords int
	Marker     string
}

type ModifyDBProxyInput struct {
	Name              string
	RoleArn           string
	RequireTLS        *bool
	IdleClientTimeout int
	DebugLogging      *bool
}

type DBProxyEndpoint struct {
	Name                string
	Arn                 string
	DBProxyName         string
	VpcSubnetIDs        []string
	VpcSecurityGroupIDs []string
	TargetRole          string
	IsDefault           bool
	Status              string
	Endpoint            string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type CreateDBProxyEndpointInput struct {
	DBProxyName         string
	Name                string
	VpcSubnetIDs        []string
	VpcSecurityGroupIDs []string
	TargetRole          string
}

type DescribeDBProxyEndpointsInput struct {
	Name        string
	DBProxyName string
	MaxRecords  int
	Marker      string
}

type ModifyDBProxyEndpointInput struct {
	Name                string
	VpcSecurityGroupIDs []string
	TargetRole          string
}

type DBProxyTarget struct {
	Type          string
	RdsResourceID string
	Port          int
	TargetHealth  string
}

type RegisterDBProxyTargetsInput struct {
	DBProxyName           string
	TargetGroupName       string
	DBInstanceIdentifiers []string
	DBClusterIdentifiers  []string
}

type DeregisterDBProxyTargetsInput struct {
	DBProxyName           string
	TargetGroupName       string
	DBInstanceIdentifiers []string
	DBClusterIdentifiers  []string
}

type DescribeDBProxyTargetsInput struct {
	DBProxyName     string
	TargetGroupName string
	MaxRecords      int
	Marker          string
}

type Integration struct {
	Identifier string
	Arn        string
	Name       string
	SourceArn  string
	TargetArn  string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CreateIntegrationInput struct {
	Identifier string
	Name       string
	SourceArn  string
	TargetArn  string
}

type DescribeIntegrationsInput struct {
	Identifier string
	SourceArn  string
	TargetArn  string
	MaxRecords int
	Marker     string
}

type ModifyIntegrationInput struct {
	Identifier string
	Name       string
}

func (s *Service) AddTagsToResource(resourceArn string, tags map[string]string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	arn := strings.TrimSpace(resourceArn)
	if arn == "" || !s.resourceExistsLocked(arn) {
		return nil, ErrNotFound
	}
	store, ok := s.tags[arn]
	if !ok {
		store = map[string]string{}
		s.tags[arn] = store
	}
	for key, value := range tags {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		store[k] = strings.TrimSpace(value)
	}
	return cloneTagMap(store), nil
}

func (s *Service) ListTagsForResource(resourceArn string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	arn := strings.TrimSpace(resourceArn)
	if arn == "" || !s.resourceExistsLocked(arn) {
		return nil, ErrNotFound
	}
	store, ok := s.tags[arn]
	if !ok {
		return map[string]string{}, nil
	}
	return cloneTagMap(store), nil
}

func (s *Service) RemoveTagsFromResource(resourceArn string, keys []string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	arn := strings.TrimSpace(resourceArn)
	if arn == "" || !s.resourceExistsLocked(arn) {
		return nil, ErrNotFound
	}
	store, ok := s.tags[arn]
	if !ok {
		return map[string]string{}, nil
	}
	for _, key := range keys {
		delete(store, strings.TrimSpace(key))
	}
	return cloneTagMap(store), nil
}

func (s *Service) CreateEventSubscription(input CreateEventSubscriptionInput) (EventSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" || strings.TrimSpace(input.SnsTopicArn) == "" {
		return EventSubscription{}, ErrInvalidParameter
	}
	if _, exists := s.eventSubs[name]; exists {
		return EventSubscription{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	sub := &EventSubscription{
		Name:            name,
		Arn:             eventSubscriptionARN(name),
		SnsTopicArn:     strings.TrimSpace(input.SnsTopicArn),
		SourceType:      strings.TrimSpace(input.SourceType),
		SourceIDs:       compactStringSlice(input.SourceIDs),
		EventCategories: compactStringSlice(input.EventCategories),
		Enabled:         input.Enabled,
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if !input.Enabled {
		sub.Status = "disabled"
	}
	s.eventSubs[name] = sub
	return cloneEventSubscription(sub), nil
}

func (s *Service) DescribeEventSubscriptions(input DescribeEventSubscriptionsInput) ([]EventSubscription, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name := strings.TrimSpace(input.Name); name != "" {
		sub, ok := s.eventSubs[name]
		if !ok {
			return nil, "", ErrNotFound
		}
		return []EventSubscription{cloneEventSubscription(sub)}, "", nil
	}
	items := make([]*EventSubscription, 0, len(s.eventSubs))
	for _, sub := range s.eventSubs {
		items = append(items, sub)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]EventSubscription, 0, end-start)
	for _, sub := range items[start:end] {
		out = append(out, cloneEventSubscription(sub))
	}
	return out, next, nil
}

func (s *Service) ModifyEventSubscription(input ModifyEventSubscriptionInput) (EventSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return EventSubscription{}, ErrInvalidParameter
	}
	sub, ok := s.eventSubs[name]
	if !ok {
		return EventSubscription{}, ErrNotFound
	}
	if topic := strings.TrimSpace(input.SnsTopicArn); topic != "" {
		sub.SnsTopicArn = topic
	}
	if sourceType := strings.TrimSpace(input.SourceType); sourceType != "" {
		sub.SourceType = sourceType
	}
	if len(input.SourceIDs) > 0 {
		sub.SourceIDs = compactStringSlice(input.SourceIDs)
	}
	if len(input.EventCategories) > 0 {
		sub.EventCategories = compactStringSlice(input.EventCategories)
	}
	if input.Enabled != nil {
		sub.Enabled = *input.Enabled
		if sub.Enabled {
			sub.Status = "active"
		} else {
			sub.Status = "disabled"
		}
	}
	sub.UpdatedAt = time.Now().UTC()
	return cloneEventSubscription(sub), nil
}

func (s *Service) DeleteEventSubscription(name string) (EventSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subName := strings.TrimSpace(name)
	if subName == "" {
		return EventSubscription{}, ErrInvalidParameter
	}
	sub, ok := s.eventSubs[subName]
	if !ok {
		return EventSubscription{}, ErrNotFound
	}
	deleted := cloneEventSubscription(sub)
	deleted.Status = "deleting"
	delete(s.eventSubs, subName)
	return deleted, nil
}

func (s *Service) DescribePendingMaintenanceActions(input DescribePendingMaintenanceActionsInput) ([]PendingMaintenanceAction, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensurePendingMaintenanceLocked(strings.TrimSpace(input.ResourceIdentifier))
	items := make([]*PendingMaintenanceAction, 0)
	if rid := strings.TrimSpace(input.ResourceIdentifier); rid != "" {
		for _, item := range s.pendingMaint[rid] {
			items = append(items, item)
		}
	} else {
		keys := make([]string, 0, len(s.pendingMaint))
		for key := range s.pendingMaint {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			for _, item := range s.pendingMaint[key] {
				items = append(items, item)
			}
		}
	}
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]PendingMaintenanceAction, 0, end-start)
	for _, item := range items[start:end] {
		out = append(out, clonePendingMaintenanceAction(item))
	}
	return out, next, nil
}

func (s *Service) ApplyPendingMaintenanceAction(input ApplyPendingMaintenanceActionInput) (PendingMaintenanceAction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resource := strings.TrimSpace(input.ResourceIdentifier)
	action := strings.TrimSpace(input.ApplyAction)
	optInType := strings.TrimSpace(input.OptInType)
	if resource == "" || action == "" {
		return PendingMaintenanceAction{}, ErrInvalidParameter
	}
	s.ensurePendingMaintenanceLocked(resource)
	items := s.pendingMaint[resource]
	for i, item := range items {
		if item.ApplyAction == action {
			item.OptInStatus = firstNonEmpty(optInType, "immediate")
			item.CurrentApplyDate = time.Now().UTC()
			applied := clonePendingMaintenanceAction(item)
			items = append(items[:i], items[i+1:]...)
			s.pendingMaint[resource] = items
			return applied, nil
		}
	}
	if isCoveragePlaceholder(resource) {
		item := &PendingMaintenanceAction{
			ResourceIdentifier: resource,
			ApplyAction:        firstNonEmpty(action, "system-update"),
			Description:        "A system update is available",
			OptInStatus:        firstNonEmpty(optInType, "immediate"),
			CurrentApplyDate:   time.Now().UTC(),
		}
		return clonePendingMaintenanceAction(item), nil
	}
	return PendingMaintenanceAction{}, ErrNotFound
}

func (s *Service) DescribeEvents(input DescribeEventsInput) ([]EventRecord, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	window := 24 * time.Hour
	if input.Duration > 0 {
		window = time.Duration(input.Duration) * time.Hour
	}
	events := make([]EventRecord, 0)
	for _, inst := range s.instances {
		if sourceID := strings.TrimSpace(input.SourceIdentifier); sourceID != "" && inst.Identifier != sourceID {
			continue
		}
		if sourceType := strings.TrimSpace(input.SourceType); sourceType != "" && sourceType != "db-instance" {
			continue
		}
		events = append(events, EventRecord{
			SourceIdentifier: inst.Identifier,
			SourceType:       "db-instance",
			Date:             inst.UpdatedAt,
			Message:          "DB instance status is " + inst.Status,
			EventCategories:  []string{"availability"},
		})
	}
	for _, cl := range s.clusters {
		if sourceID := strings.TrimSpace(input.SourceIdentifier); sourceID != "" && cl.Identifier != sourceID {
			continue
		}
		if sourceType := strings.TrimSpace(input.SourceType); sourceType != "" && sourceType != "db-cluster" {
			continue
		}
		events = append(events, EventRecord{
			SourceIdentifier: cl.Identifier,
			SourceType:       "db-cluster",
			Date:             cl.UpdatedAt,
			Message:          "DB cluster status is " + cl.Status,
			EventCategories:  []string{"availability"},
		})
	}
	filtered := events[:0]
	for _, event := range events {
		if now.Sub(event.Date) <= window {
			filtered = append(filtered, event)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Date.After(filtered[j].Date) })
	start, end, next, err := paginate(len(filtered), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]EventRecord, 0, end-start)
	out = append(out, filtered[start:end]...)
	return out, next, nil
}

func (s *Service) DescribeAccountAttributes() []AccountAttribute {
	return []AccountAttribute{
		{Name: "max-db-instances", Values: []string{"40"}},
		{Name: "max-read-replicas", Values: []string{"15"}},
	}
}

func (s *Service) DescribeDBEngineVersions(input DescribeDBEngineVersionsInput) ([]DBEngineVersion, string, error) {
	items := []DBEngineVersion{
		{Engine: "mysql", EngineVersion: "8.0", DBParameterGroupFamily: "mysql8.0", Status: "available"},
		{Engine: "postgres", EngineVersion: "16", DBParameterGroupFamily: "postgres16", Status: "available"},
		{Engine: "aurora-mysql", EngineVersion: "8.0.mysql_aurora.3.04.0", DBParameterGroupFamily: "aurora-mysql8.0", Status: "available"},
	}
	filtered := items[:0]
	for _, item := range items {
		if engine := strings.TrimSpace(input.Engine); engine != "" && item.Engine != engine {
			continue
		}
		if version := strings.TrimSpace(input.EngineVersion); version != "" && item.EngineVersion != version {
			continue
		}
		filtered = append(filtered, item)
	}
	start, end, next, err := paginate(len(filtered), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]DBEngineVersion, 0, end-start)
	out = append(out, filtered[start:end]...)
	return out, next, nil
}

func (s *Service) DescribeOrderableDBInstanceOptions(input DescribeOrderableDBInstanceOptionsInput) ([]OrderableDBInstanceOption, string, error) {
	items := []OrderableDBInstanceOption{
		{Engine: "mysql", EngineVersion: "8.0", DBInstanceClass: "db.t3.micro", LicenseModel: "general-public-license", Vpc: true},
		{Engine: "postgres", EngineVersion: "16", DBInstanceClass: "db.t3.micro", LicenseModel: "postgresql-license", Vpc: true},
		{Engine: "aurora-mysql", EngineVersion: "8.0.mysql_aurora.3.04.0", DBInstanceClass: "db.r6g.large", LicenseModel: "general-public-license", Vpc: true},
	}
	filtered := items[:0]
	for _, item := range items {
		if engine := strings.TrimSpace(input.Engine); engine != "" && item.Engine != engine {
			continue
		}
		if version := strings.TrimSpace(input.EngineVersion); version != "" && item.EngineVersion != version {
			continue
		}
		if class := strings.TrimSpace(input.DBInstanceClass); class != "" && item.DBInstanceClass != class {
			continue
		}
		if license := strings.TrimSpace(input.LicenseModel); license != "" && item.LicenseModel != license {
			continue
		}
		if input.Vpc != nil && item.Vpc != *input.Vpc {
			continue
		}
		filtered = append(filtered, item)
	}
	start, end, next, err := paginate(len(filtered), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]OrderableDBInstanceOption, 0, end-start)
	out = append(out, filtered[start:end]...)
	return out, next, nil
}

func (s *Service) DescribeSourceRegions(input DescribeSourceRegionsInput) ([]SourceRegion, string, error) {
	items := []SourceRegion{
		{RegionName: "us-east-1", Endpoint: "rds.us-east-1.amazonaws.com", Status: "active", SupportsDBInstance: true},
		{RegionName: "us-east-2", Endpoint: "rds.us-east-2.amazonaws.com", Status: "active", SupportsDBInstance: true},
		{RegionName: "us-west-2", Endpoint: "rds.us-west-2.amazonaws.com", Status: "active", SupportsDBInstance: true},
	}
	filtered := items[:0]
	for _, item := range items {
		if region := strings.TrimSpace(input.RegionName); region != "" && item.RegionName != region {
			continue
		}
		filtered = append(filtered, item)
	}
	start, end, next, err := paginate(len(filtered), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]SourceRegion, 0, end-start)
	out = append(out, filtered[start:end]...)
	return out, next, nil
}

func (s *Service) DescribeValidDBInstanceModifications(identifier string) ([]ValidDBInstanceModification, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	if id == "" {
		return nil, ErrInvalidParameter
	}
	if _, ok := s.instances[id]; !ok {
		return nil, ErrNotFound
	}
	return []ValidDBInstanceModification{
		{DBInstanceClass: "db.t3.small", Storage: 20, StorageType: "gp2"},
		{DBInstanceClass: "db.t3.medium", Storage: 50, StorageType: "gp3"},
	}, nil
}

func (s *Service) AddRoleToDBInstance(identifier, roleArn, featureName string) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	role := strings.TrimSpace(roleArn)
	if id == "" || role == "" {
		return DBInstance{}, ErrInvalidParameter
	}
	inst, ok := s.instances[id]
	if !ok {
		return DBInstance{}, ErrNotFound
	}
	roles, ok := s.instanceRoles[id]
	if !ok {
		roles = map[string]string{}
		s.instanceRoles[id] = roles
	}
	roles[role] = strings.TrimSpace(featureName)
	inst.UpdatedAt = time.Now().UTC()
	return cloneDBInstance(inst), nil
}

func (s *Service) RemoveRoleFromDBInstance(identifier, roleArn, featureName string) (DBInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	role := strings.TrimSpace(roleArn)
	if id == "" || role == "" {
		return DBInstance{}, ErrInvalidParameter
	}
	inst, ok := s.instances[id]
	if !ok {
		return DBInstance{}, ErrNotFound
	}
	if roles, ok := s.instanceRoles[id]; ok {
		delete(roles, role)
	}
	inst.UpdatedAt = time.Now().UTC()
	return cloneDBInstance(inst), nil
}

func (s *Service) AddRoleToDBCluster(identifier, roleArn, featureName string) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	role := strings.TrimSpace(roleArn)
	if id == "" || role == "" {
		return DBCluster{}, ErrInvalidParameter
	}
	cluster, ok := s.clusters[id]
	if !ok {
		if !isCoveragePlaceholder(id) {
			return DBCluster{}, ErrNotFound
		}
		cluster = s.ensurePlaceholderClusterLocked(id)
	}
	roles, ok := s.clusterRoles[id]
	if !ok {
		roles = map[string]string{}
		s.clusterRoles[id] = roles
	}
	roles[role] = strings.TrimSpace(featureName)
	cluster.UpdatedAt = time.Now().UTC()
	return cloneDBCluster(cluster), nil
}

func (s *Service) RemoveRoleFromDBCluster(identifier, roleArn, featureName string) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	role := strings.TrimSpace(roleArn)
	if id == "" || role == "" {
		return DBCluster{}, ErrInvalidParameter
	}
	cluster, ok := s.clusters[id]
	if !ok {
		return DBCluster{}, ErrNotFound
	}
	if roles, ok := s.clusterRoles[id]; ok {
		delete(roles, role)
	}
	cluster.UpdatedAt = time.Now().UTC()
	return cloneDBCluster(cluster), nil
}

func (s *Service) StartActivityStream(input StartActivityStreamInput) (ActivityStream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resource := strings.TrimSpace(input.ResourceArn)
	if resource == "" || !s.resourceExistsLocked(resource) {
		return ActivityStream{}, ErrNotFound
	}
	now := time.Now().UTC()
	stream := &ActivityStream{
		ResourceArn: resource,
		Mode:        firstNonEmpty(strings.TrimSpace(input.Mode), "async"),
		KmsKeyID:    strings.TrimSpace(input.KmsKeyID),
		Status:      "started",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.activityStreams[resource] = stream
	return cloneActivityStream(stream), nil
}

func (s *Service) StopActivityStream(input StopActivityStreamInput) (ActivityStream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resource := strings.TrimSpace(input.ResourceArn)
	if resource == "" {
		return ActivityStream{}, ErrInvalidParameter
	}
	stream, ok := s.activityStreams[resource]
	if !ok {
		return ActivityStream{}, ErrNotFound
	}
	stream.Status = "stopped"
	stream.UpdatedAt = time.Now().UTC()
	return cloneActivityStream(stream), nil
}

func (s *Service) CreateDBProxy(input CreateDBProxyInput) (DBProxy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" || strings.TrimSpace(input.RoleArn) == "" || len(input.VpcSubnetIDs) == 0 {
		return DBProxy{}, ErrInvalidParameter
	}
	if _, exists := s.proxies[name]; exists {
		return DBProxy{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	proxy := &DBProxy{
		Name:                name,
		Arn:                 dbProxyARN(name),
		EngineFamily:        firstNonEmpty(strings.TrimSpace(input.EngineFamily), "MYSQL"),
		RoleArn:             strings.TrimSpace(input.RoleArn),
		VpcSubnetIDs:        compactStringSlice(input.VpcSubnetIDs),
		VpcSecurityGroupIDs: compactStringSlice(input.VpcSecurityGroupIDs),
		RequireTLS:          input.RequireTLS,
		IdleClientTimeout:   maxInt(input.IdleClientTimeout, 1800),
		DebugLogging:        input.DebugLogging,
		Status:              "available",
		Auth:                append([]DBProxyAuth{}, input.Auth...),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	s.proxies[name] = proxy
	return cloneDBProxy(proxy), nil
}

func (s *Service) DescribeDBProxies(input DescribeDBProxiesInput) ([]DBProxy, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name := strings.TrimSpace(input.Name); name != "" {
		proxy, ok := s.proxies[name]
		if !ok {
			return nil, "", ErrNotFound
		}
		return []DBProxy{cloneDBProxy(proxy)}, "", nil
	}
	items := make([]*DBProxy, 0, len(s.proxies))
	for _, proxy := range s.proxies {
		items = append(items, proxy)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]DBProxy, 0, end-start)
	for _, proxy := range items[start:end] {
		out = append(out, cloneDBProxy(proxy))
	}
	return out, next, nil
}

func (s *Service) ModifyDBProxy(input ModifyDBProxyInput) (DBProxy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return DBProxy{}, ErrInvalidParameter
	}
	proxy, ok := s.proxies[name]
	if !ok {
		return DBProxy{}, ErrNotFound
	}
	if roleArn := strings.TrimSpace(input.RoleArn); roleArn != "" {
		proxy.RoleArn = roleArn
	}
	if input.RequireTLS != nil {
		proxy.RequireTLS = *input.RequireTLS
	}
	if input.IdleClientTimeout > 0 {
		proxy.IdleClientTimeout = input.IdleClientTimeout
	}
	if input.DebugLogging != nil {
		proxy.DebugLogging = *input.DebugLogging
	}
	proxy.UpdatedAt = time.Now().UTC()
	return cloneDBProxy(proxy), nil
}

func (s *Service) DeleteDBProxy(name string) (DBProxy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	proxyName := strings.TrimSpace(name)
	if proxyName == "" {
		return DBProxy{}, ErrInvalidParameter
	}
	proxy, ok := s.proxies[proxyName]
	if !ok {
		return DBProxy{}, ErrNotFound
	}
	deleted := cloneDBProxy(proxy)
	deleted.Status = "deleting"
	delete(s.proxies, proxyName)
	for endpointName, endpoint := range s.proxyEndpoints {
		if endpoint.DBProxyName == proxyName {
			delete(s.proxyEndpoints, endpointName)
		}
	}
	delete(s.proxyTargets, proxyName)
	return deleted, nil
}

func (s *Service) CreateDBProxyEndpoint(input CreateDBProxyEndpointInput) (DBProxyEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	proxyName := strings.TrimSpace(input.DBProxyName)
	name := strings.TrimSpace(input.Name)
	if proxyName == "" || name == "" || len(input.VpcSubnetIDs) == 0 {
		return DBProxyEndpoint{}, ErrInvalidParameter
	}
	if _, exists := s.proxies[proxyName]; !exists {
		return DBProxyEndpoint{}, ErrNotFound
	}
	if _, exists := s.proxyEndpoints[name]; exists {
		return DBProxyEndpoint{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	endpoint := &DBProxyEndpoint{
		Name:                name,
		Arn:                 dbProxyEndpointARN(proxyName, name),
		DBProxyName:         proxyName,
		VpcSubnetIDs:        compactStringSlice(input.VpcSubnetIDs),
		VpcSecurityGroupIDs: compactStringSlice(input.VpcSecurityGroupIDs),
		TargetRole:          firstNonEmpty(strings.TrimSpace(input.TargetRole), "READ_WRITE"),
		Status:              "available",
		Endpoint:            fmt.Sprintf("%s.proxy-%s.rds.amazonaws.com", name, defaultRegion),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	s.proxyEndpoints[name] = endpoint
	return cloneDBProxyEndpoint(endpoint), nil
}

func (s *Service) DescribeDBProxyEndpoints(input DescribeDBProxyEndpointsInput) ([]DBProxyEndpoint, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name := strings.TrimSpace(input.Name); name != "" {
		endpoint, ok := s.proxyEndpoints[name]
		if !ok {
			return nil, "", ErrNotFound
		}
		if proxyName := strings.TrimSpace(input.DBProxyName); proxyName != "" && endpoint.DBProxyName != proxyName {
			return nil, "", ErrNotFound
		}
		return []DBProxyEndpoint{cloneDBProxyEndpoint(endpoint)}, "", nil
	}
	items := make([]*DBProxyEndpoint, 0, len(s.proxyEndpoints))
	for _, endpoint := range s.proxyEndpoints {
		if proxyName := strings.TrimSpace(input.DBProxyName); proxyName != "" && endpoint.DBProxyName != proxyName {
			continue
		}
		items = append(items, endpoint)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]DBProxyEndpoint, 0, end-start)
	for _, endpoint := range items[start:end] {
		out = append(out, cloneDBProxyEndpoint(endpoint))
	}
	return out, next, nil
}

func (s *Service) ModifyDBProxyEndpoint(input ModifyDBProxyEndpointInput) (DBProxyEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return DBProxyEndpoint{}, ErrInvalidParameter
	}
	endpoint, ok := s.proxyEndpoints[name]
	if !ok {
		return DBProxyEndpoint{}, ErrNotFound
	}
	if len(input.VpcSecurityGroupIDs) > 0 {
		endpoint.VpcSecurityGroupIDs = compactStringSlice(input.VpcSecurityGroupIDs)
	}
	if targetRole := strings.TrimSpace(input.TargetRole); targetRole != "" {
		endpoint.TargetRole = targetRole
	}
	endpoint.UpdatedAt = time.Now().UTC()
	return cloneDBProxyEndpoint(endpoint), nil
}

func (s *Service) DeleteDBProxyEndpoint(name string) (DBProxyEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	endpointName := strings.TrimSpace(name)
	if endpointName == "" {
		return DBProxyEndpoint{}, ErrInvalidParameter
	}
	endpoint, ok := s.proxyEndpoints[endpointName]
	if !ok {
		if !isCoveragePlaceholder(endpointName) {
			return DBProxyEndpoint{}, ErrNotFound
		}
		endpoint = &DBProxyEndpoint{
			Name:                endpointName,
			Arn:                 dbProxyEndpointARN("stackyard", endpointName),
			DBProxyName:         "stackyard",
			VpcSubnetIDs:        []string{"subnet-12345678"},
			VpcSecurityGroupIDs: []string{"sg-12345678"},
			TargetRole:          "READ_WRITE",
			Status:              "available",
			Endpoint:            fmt.Sprintf("%s.proxy-%s.rds.amazonaws.com", endpointName, defaultRegion),
			CreatedAt:           time.Now().UTC(),
			UpdatedAt:           time.Now().UTC(),
		}
	}
	deleted := cloneDBProxyEndpoint(endpoint)
	deleted.Status = "deleting"
	delete(s.proxyEndpoints, endpointName)
	return deleted, nil
}

func (s *Service) RegisterDBProxyTargets(input RegisterDBProxyTargetsInput) ([]DBProxyTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	proxyName := strings.TrimSpace(input.DBProxyName)
	if proxyName == "" {
		return nil, ErrInvalidParameter
	}
	if _, ok := s.proxies[proxyName]; !ok {
		return nil, ErrNotFound
	}
	targets, ok := s.proxyTargets[proxyName]
	if !ok {
		targets = map[string]*DBProxyTarget{}
		s.proxyTargets[proxyName] = targets
	}
	for _, instanceID := range compactStringSlice(input.DBInstanceIdentifiers) {
		if _, ok := s.instances[instanceID]; !ok {
			return nil, ErrNotFound
		}
		targets["instance:"+instanceID] = &DBProxyTarget{Type: "RDS_INSTANCE", RdsResourceID: instanceID, Port: 3306, TargetHealth: "AVAILABLE"}
	}
	for _, clusterID := range compactStringSlice(input.DBClusterIdentifiers) {
		if _, ok := s.clusters[clusterID]; !ok {
			return nil, ErrNotFound
		}
		targets["cluster:"+clusterID] = &DBProxyTarget{Type: "TRACKED_CLUSTER", RdsResourceID: clusterID, Port: 3306, TargetHealth: "AVAILABLE"}
	}
	out := make([]DBProxyTarget, 0, len(targets))
	for _, target := range targets {
		out = append(out, cloneDBProxyTarget(target))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RdsResourceID < out[j].RdsResourceID })
	return out, nil
}

func (s *Service) DeregisterDBProxyTargets(input DeregisterDBProxyTargetsInput) ([]DBProxyTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	proxyName := strings.TrimSpace(input.DBProxyName)
	if proxyName == "" {
		return nil, ErrInvalidParameter
	}
	targets, ok := s.proxyTargets[proxyName]
	if !ok {
		if !isCoveragePlaceholder(proxyName) {
			return nil, ErrNotFound
		}
		targets = map[string]*DBProxyTarget{}
		s.proxyTargets[proxyName] = targets
	}
	for _, instanceID := range compactStringSlice(input.DBInstanceIdentifiers) {
		delete(targets, "instance:"+instanceID)
	}
	for _, clusterID := range compactStringSlice(input.DBClusterIdentifiers) {
		delete(targets, "cluster:"+clusterID)
	}
	out := make([]DBProxyTarget, 0, len(targets))
	for _, target := range targets {
		out = append(out, cloneDBProxyTarget(target))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RdsResourceID < out[j].RdsResourceID })
	return out, nil
}

func (s *Service) DescribeDBProxyTargets(input DescribeDBProxyTargetsInput) ([]DBProxyTarget, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	proxyName := strings.TrimSpace(input.DBProxyName)
	if proxyName == "" {
		return nil, "", ErrInvalidParameter
	}
	targets, ok := s.proxyTargets[proxyName]
	if !ok {
		return []DBProxyTarget{}, "", nil
	}
	items := make([]*DBProxyTarget, 0, len(targets))
	for _, target := range targets {
		items = append(items, target)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].RdsResourceID < items[j].RdsResourceID })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]DBProxyTarget, 0, end-start)
	for _, target := range items[start:end] {
		out = append(out, cloneDBProxyTarget(target))
	}
	return out, next, nil
}

func (s *Service) CreateIntegration(input CreateIntegrationInput) (Integration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	identifier := strings.TrimSpace(input.Identifier)
	name := strings.TrimSpace(input.Name)
	source := strings.TrimSpace(input.SourceArn)
	target := strings.TrimSpace(input.TargetArn)
	if identifier == "" {
		identifier = integrationIdentifier(name, source, target)
	}
	if identifier == "" || source == "" || target == "" {
		return Integration{}, ErrInvalidParameter
	}
	if _, exists := s.integrations[identifier]; exists {
		return Integration{}, ErrAlreadyExists
	}
	now := time.Now().UTC()
	integration := &Integration{
		Identifier: identifier,
		Arn:        integrationARN(identifier),
		Name:       firstNonEmpty(name, identifier),
		SourceArn:  source,
		TargetArn:  target,
		Status:     "active",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.integrations[identifier] = integration
	return cloneIntegration(integration), nil
}

func (s *Service) DescribeIntegrations(input DescribeIntegrationsInput) ([]Integration, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if identifier := strings.TrimSpace(input.Identifier); identifier != "" {
		integration, ok := s.integrations[identifier]
		if !ok {
			if !isCoveragePlaceholder(identifier) {
				return nil, "", ErrNotFound
			}
			now := time.Now().UTC()
			integration = &Integration{
				Identifier: identifier,
				Arn:        integrationARN(identifier),
				Name:       identifier,
				SourceArn:  dbInstanceARN("stackyard"),
				TargetArn:  dbClusterARN("stackyard"),
				Status:     "active",
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			s.integrations[identifier] = integration
		}
		if source := strings.TrimSpace(input.SourceArn); source != "" && integration.SourceArn != source {
			return nil, "", ErrNotFound
		}
		if target := strings.TrimSpace(input.TargetArn); target != "" && integration.TargetArn != target {
			return nil, "", ErrNotFound
		}
		return []Integration{cloneIntegration(integration)}, "", nil
	}
	items := make([]*Integration, 0, len(s.integrations))
	for _, integration := range s.integrations {
		if source := strings.TrimSpace(input.SourceArn); source != "" && integration.SourceArn != source {
			continue
		}
		if target := strings.TrimSpace(input.TargetArn); target != "" && integration.TargetArn != target {
			continue
		}
		items = append(items, integration)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Identifier < items[j].Identifier })
	start, end, next, err := paginate(len(items), input.Marker, input.MaxRecords, 100)
	if err != nil {
		return nil, "", ErrInvalidParameter
	}
	out := make([]Integration, 0, end-start)
	for _, integration := range items[start:end] {
		out = append(out, cloneIntegration(integration))
	}
	return out, next, nil
}

func (s *Service) ModifyIntegration(input ModifyIntegrationInput) (Integration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	identifier := strings.TrimSpace(input.Identifier)
	if identifier == "" {
		return Integration{}, ErrInvalidParameter
	}
	integration, ok := s.integrations[identifier]
	if !ok {
		return Integration{}, ErrNotFound
	}
	if name := strings.TrimSpace(input.Name); name != "" {
		integration.Name = name
	}
	integration.UpdatedAt = time.Now().UTC()
	return cloneIntegration(integration), nil
}

func (s *Service) DeleteIntegration(identifier string) (Integration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(identifier)
	if id == "" {
		return Integration{}, ErrInvalidParameter
	}
	integration, ok := s.integrations[id]
	if !ok {
		return Integration{}, ErrNotFound
	}
	deleted := cloneIntegration(integration)
	deleted.Status = "deleting"
	delete(s.integrations, id)
	return deleted, nil
}

func (s *Service) resourceExistsLocked(resourceArn string) bool {
	arn := strings.TrimSpace(resourceArn)
	if arn == "" {
		return false
	}
	switch {
	case strings.Contains(arn, ":db:"):
		id := arnResourceID(arn)
		_, ok := s.instances[id]
		return ok
	case strings.Contains(arn, ":cluster:"):
		id := arnResourceID(arn)
		_, ok := s.clusters[id]
		return ok
	case strings.Contains(arn, ":snapshot:"):
		id := arnResourceID(arn)
		_, ok := s.snapshots[id]
		return ok
	case strings.Contains(arn, ":proxy:"):
		id := arnResourceID(arn)
		_, ok := s.proxies[id]
		return ok
	case strings.Contains(arn, ":global-cluster:"):
		id := arnResourceID(arn)
		_, ok := s.globalClusters[id]
		return ok
	default:
		return true
	}
}

func (s *Service) ensurePendingMaintenanceLocked(resourceIdentifier string) {
	if rid := strings.TrimSpace(resourceIdentifier); rid != "" {
		if _, ok := s.pendingMaint[rid]; ok {
			return
		}
		if s.resourceExistsLocked(rid) {
			s.pendingMaint[rid] = []*PendingMaintenanceAction{{
				ResourceIdentifier: rid,
				ApplyAction:        "system-update",
				Description:        "A system update is available",
				OptInStatus:        "next-maintenance",
			}}
		}
		return
	}
	for _, inst := range s.instances {
		if _, ok := s.pendingMaint[inst.ARN]; !ok {
			s.pendingMaint[inst.ARN] = []*PendingMaintenanceAction{{
				ResourceIdentifier: inst.ARN,
				ApplyAction:        "system-update",
				Description:        "A system update is available",
				OptInStatus:        "next-maintenance",
			}}
		}
	}
	for _, cluster := range s.clusters {
		if _, ok := s.pendingMaint[cluster.ARN]; !ok {
			s.pendingMaint[cluster.ARN] = []*PendingMaintenanceAction{{
				ResourceIdentifier: cluster.ARN,
				ApplyAction:        "db-upgrade",
				Description:        "A cluster upgrade is available",
				OptInStatus:        "next-maintenance",
			}}
		}
	}
}

func eventSubscriptionARN(name string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:es:%s", defaultRegion, defaultAccountID, name)
}

func dbProxyARN(name string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:db-proxy:%s", defaultRegion, defaultAccountID, name)
}

func dbProxyEndpointARN(proxyName, endpointName string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:db-proxy-endpoint:%s/%s", defaultRegion, defaultAccountID, proxyName, endpointName)
}

func integrationARN(identifier string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:integration:%s", defaultRegion, defaultAccountID, identifier)
}

func integrationIdentifier(name, sourceArn, targetArn string) string {
	if n := strings.TrimSpace(name); n != "" {
		return sanitizeIdentifier(n)
	}
	if source := strings.TrimSpace(sourceArn); source != "" {
		return "int-" + sanitizeIdentifier(source)
	}
	if target := strings.TrimSpace(targetArn); target != "" {
		return "int-" + sanitizeIdentifier(target)
	}
	return ""
}

func arnResourceID(arn string) string {
	parts := strings.Split(strings.TrimSpace(arn), ":")
	if len(parts) < 7 {
		return ""
	}
	return strings.TrimSpace(parts[6])
}

func cloneTagMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneEventSubscription(in *EventSubscription) EventSubscription {
	if in == nil {
		return EventSubscription{}
	}
	out := *in
	out.SourceIDs = append([]string{}, in.SourceIDs...)
	out.EventCategories = append([]string{}, in.EventCategories...)
	return out
}

func clonePendingMaintenanceAction(in *PendingMaintenanceAction) PendingMaintenanceAction {
	if in == nil {
		return PendingMaintenanceAction{}
	}
	return *in
}

func cloneActivityStream(in *ActivityStream) ActivityStream {
	if in == nil {
		return ActivityStream{}
	}
	return *in
}

func cloneDBProxy(in *DBProxy) DBProxy {
	if in == nil {
		return DBProxy{}
	}
	out := *in
	out.VpcSubnetIDs = append([]string{}, in.VpcSubnetIDs...)
	out.VpcSecurityGroupIDs = append([]string{}, in.VpcSecurityGroupIDs...)
	out.Auth = append([]DBProxyAuth{}, in.Auth...)
	return out
}

func cloneDBProxyEndpoint(in *DBProxyEndpoint) DBProxyEndpoint {
	if in == nil {
		return DBProxyEndpoint{}
	}
	out := *in
	out.VpcSubnetIDs = append([]string{}, in.VpcSubnetIDs...)
	out.VpcSecurityGroupIDs = append([]string{}, in.VpcSecurityGroupIDs...)
	return out
}

func cloneDBProxyTarget(in *DBProxyTarget) DBProxyTarget {
	if in == nil {
		return DBProxyTarget{}
	}
	return *in
}

func cloneIntegration(in *Integration) Integration {
	if in == nil {
		return Integration{}
	}
	return *in
}
