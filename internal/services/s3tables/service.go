package s3tables

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type TableBucket struct {
	Arn            string
	CreatedAt      string
	Name           string
	OwnerAccountId string
	TableBucketId  string
	Type           string
}

type Namespace struct {
	CreatedAt      string
	CreatedBy      string
	Namespace      []string
	NamespaceId    string
	OwnerAccountId string
	TableBucketId  string
	TableBucketArn string
}

type Table struct {
	CreatedAt        string
	CreatedBy        string
	Namespace        []string
	Name             string
	TableArn         string
	TableBucketArn   string
	TableBucketId    string
	TableId          string
	MetadataLocation string
	Format           string
}

type Service struct {
	mu                     sync.Mutex
	tableBuckets           map[string]TableBucket
	namespaces             map[string][]Namespace
	tables                 map[string]map[string]map[string]Table
	resourceTags           map[string]map[string]string
	tablePolicies          map[string]string
	bucketPolicies         map[string]string
	tableEncryption        map[string]string
	bucketEncryption       map[string]string
	tableStorageClass      map[string]string
	bucketStorageClass     map[string]string
	tableReplication       map[string]string
	bucketReplication      map[string]string
	tableReplicationStatus map[string]string
	tableMaintenance       map[string]string
	bucketMaintenance      map[string]string
	tableMaintenanceStatus map[string]string
	tableRecordExpiration  map[string]string
	recordExpirationStatus map[string]string
	bucketMetrics          map[string]string
	nextBucketID           int
	nextNamespaceID        int
	nextTableID            int
}

func NewService() *Service {
	s := &Service{
		tableBuckets:           map[string]TableBucket{},
		namespaces:             map[string][]Namespace{},
		tables:                 map[string]map[string]map[string]Table{},
		resourceTags:           map[string]map[string]string{},
		tablePolicies:          map[string]string{},
		bucketPolicies:         map[string]string{},
		tableEncryption:        map[string]string{},
		bucketEncryption:       map[string]string{},
		tableStorageClass:      map[string]string{},
		bucketStorageClass:     map[string]string{},
		tableReplication:       map[string]string{},
		bucketReplication:      map[string]string{},
		tableReplicationStatus: map[string]string{},
		tableMaintenance:       map[string]string{},
		bucketMaintenance:      map[string]string{},
		tableMaintenanceStatus: map[string]string{},
		tableRecordExpiration:  map[string]string{},
		recordExpirationStatus: map[string]string{},
		bucketMetrics:          map[string]string{},
		nextBucketID:           2,
		nextNamespaceID:        2,
		nextTableID:            2,
	}
	seedBucket := TableBucket{
		Arn:            "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket",
		CreatedAt:      "2024-01-01T00:00:00Z",
		Name:           "demo-bucket",
		OwnerAccountId: "123456789012",
		TableBucketId:  "tb-1234567890abcdef",
		Type:           "customer",
	}
	s.tableBuckets[seedBucket.Arn] = seedBucket
	seedNamespace := Namespace{
		CreatedAt:      "2024-01-02T00:00:00Z",
		CreatedBy:      "arn:aws:iam::123456789012:user/test",
		Namespace:      []string{"analytics"},
		NamespaceId:    "ns-1234567890abcdef",
		OwnerAccountId: "123456789012",
		TableBucketId:  seedBucket.TableBucketId,
		TableBucketArn: seedBucket.Arn,
	}
	s.namespaces[seedBucket.Arn] = []Namespace{seedNamespace}
	s.tables[seedBucket.Arn] = map[string]map[string]Table{}
	seedTable := Table{
		CreatedAt:        "2024-01-03T00:00:00Z",
		CreatedBy:        "arn:aws:iam::123456789012:user/test",
		Namespace:        []string{"analytics"},
		Name:             "events",
		TableArn:         "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket/table/analytics/events",
		TableBucketArn:   seedBucket.Arn,
		TableBucketId:    seedBucket.TableBucketId,
		TableId:          "tbtable-1234567890abcdef",
		MetadataLocation: "s3://demo-bucket/analytics/events/metadata.json",
		Format:           "ICEBERG",
	}
	s.tables[seedBucket.Arn]["analytics"] = map[string]Table{"events": seedTable}
	return s
}

func (s *Service) CreateTableBucket(name string) TableBucket {
	s.mu.Lock()
	defer s.mu.Unlock()
	arn := "arn:aws:s3tables:us-east-1:123456789012:bucket/" + name
	if existing, ok := s.tableBuckets[arn]; ok {
		return existing
	}
	id := fmt.Sprintf("tb-1234567890abcde%d", s.nextBucketID)
	s.nextBucketID++
	bucket := TableBucket{
		Arn:            arn,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		Name:           name,
		OwnerAccountId: "123456789012",
		TableBucketId:  id,
		Type:           "customer",
	}
	s.tableBuckets[arn] = bucket
	return bucket
}

func (s *Service) GetTableBucket(arn string) (TableBucket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket, ok := s.tableBuckets[arn]
	return bucket, ok
}

func (s *Service) DeleteTableBucket(arn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[arn]; !ok {
		return false
	}
	delete(s.tableBuckets, arn)
	delete(s.namespaces, arn)
	delete(s.tables, arn)
	return true
}

func (s *Service) ListTableBuckets(prefix, typ string) []TableBucket {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]TableBucket, 0, len(s.tableBuckets))
	for _, bucket := range s.tableBuckets {
		if prefix != "" && !strings.HasPrefix(bucket.Name, prefix) {
			continue
		}
		if typ != "" && bucket.Type != typ {
			continue
		}
		out = append(out, bucket)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Service) CreateNamespace(tableBucketArn string, namespace []string) Namespace {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("ns-1234567890abcde%d", s.nextNamespaceID)
	s.nextNamespaceID++
	bucket := s.tableBuckets[tableBucketArn]
	ns := Namespace{
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		CreatedBy:      "arn:aws:iam::123456789012:user/test",
		Namespace:      namespace,
		NamespaceId:    id,
		OwnerAccountId: "123456789012",
		TableBucketId:  bucket.TableBucketId,
		TableBucketArn: tableBucketArn,
	}
	s.namespaces[tableBucketArn] = append(s.namespaces[tableBucketArn], ns)
	return ns
}

func (s *Service) GetNamespace(tableBucketArn string, namespace []string) (Namespace, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ns := range s.namespaces[tableBucketArn] {
		if strings.Join(ns.Namespace, ".") == strings.Join(namespace, ".") {
			return ns, true
		}
	}
	return Namespace{}, false
}

func (s *Service) DeleteNamespace(tableBucketArn string, namespace []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.namespaces[tableBucketArn]
	if len(items) == 0 {
		return false
	}
	target := strings.Join(namespace, ".")
	out := items[:0]
	removed := false
	for _, ns := range items {
		if strings.Join(ns.Namespace, ".") == target {
			removed = true
			continue
		}
		out = append(out, ns)
	}
	if removed {
		s.namespaces[tableBucketArn] = out
	}
	return removed
}

func (s *Service) ListNamespaces(tableBucketArn, prefix string) []Namespace {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.namespaces[tableBucketArn]
	out := make([]Namespace, 0, len(items))
	for _, ns := range items {
		if prefix != "" && (len(ns.Namespace) == 0 || !strings.HasPrefix(ns.Namespace[0], prefix)) {
			continue
		}
		out = append(out, ns)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.Join(out[i].Namespace, ".") < strings.Join(out[j].Namespace, ".")
	})
	return out
}

func (s *Service) CreateTable(tableBucketArn string, namespace []string, name string) (Table, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket, ok := s.tableBuckets[tableBucketArn]
	if !ok {
		return Table{}, false
	}
	key := strings.Join(namespace, ".")
	if s.tables[tableBucketArn] == nil {
		s.tables[tableBucketArn] = map[string]map[string]Table{}
	}
	if s.tables[tableBucketArn][key] == nil {
		s.tables[tableBucketArn][key] = map[string]Table{}
	}
	if existing, ok := s.tables[tableBucketArn][key][name]; ok {
		return existing, true
	}
	id := fmt.Sprintf("tbtable-1234567890abcde%d", s.nextTableID)
	s.nextTableID++
	table := Table{
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		CreatedBy:        "arn:aws:iam::123456789012:user/test",
		Namespace:        namespace,
		Name:             name,
		TableArn:         "arn:aws:s3tables:us-east-1:123456789012:bucket/" + bucket.Name + "/table/" + key + "/" + name,
		TableBucketArn:   tableBucketArn,
		TableBucketId:    bucket.TableBucketId,
		TableId:          id,
		MetadataLocation: "s3://" + bucket.Name + "/" + key + "/" + name + "/metadata.json",
		Format:           "ICEBERG",
	}
	s.tables[tableBucketArn][key][name] = table
	return table, true
}

func (s *Service) GetTable(tableBucketArn string, namespace []string, name string) (Table, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.Join(namespace, ".")
	nsMap, ok := s.tables[tableBucketArn]
	if !ok {
		return Table{}, false
	}
	table, ok := nsMap[key][name]
	return table, ok
}

func (s *Service) DeleteTable(tableBucketArn string, namespace []string, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.Join(namespace, ".")
	nsMap, ok := s.tables[tableBucketArn]
	if !ok {
		return false
	}
	if _, ok := nsMap[key][name]; !ok {
		return false
	}
	delete(nsMap[key], name)
	return true
}

func (s *Service) ListTables(tableBucketArn string, namespace []string) []Table {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.Join(namespace, ".")
	nsMap, ok := s.tables[tableBucketArn]
	if !ok {
		return nil
	}
	tables := make([]Table, 0, len(nsMap[key]))
	for _, table := range nsMap[key] {
		tables = append(tables, table)
	}
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})
	return tables
}

func (s *Service) RenameTable(tableBucketArn string, namespace []string, name, newName string) (Table, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.Join(namespace, ".")
	nsMap, ok := s.tables[tableBucketArn]
	if !ok {
		return Table{}, false
	}
	table, ok := nsMap[key][name]
	if !ok {
		return Table{}, false
	}
	if _, exists := nsMap[key][newName]; exists {
		return Table{}, false
	}
	delete(nsMap[key], name)
	table.Name = newName
	table.TableArn = "arn:aws:s3tables:us-east-1:123456789012:bucket/" + extractBucketName(table.TableBucketArn) + "/table/" + key + "/" + newName
	table.MetadataLocation = "s3://" + extractBucketName(table.TableBucketArn) + "/" + key + "/" + newName + "/metadata.json"
	nsMap[key][newName] = table
	return table, true
}

func (s *Service) GetTableMetadataLocation(tableBucketArn string, namespace []string, name string) (string, bool) {
	table, ok := s.GetTable(tableBucketArn, namespace, name)
	if !ok {
		return "", false
	}
	return table.MetadataLocation, true
}

func (s *Service) UpdateTableMetadataLocation(tableBucketArn string, namespace []string, name, location string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.Join(namespace, ".")
	nsMap, ok := s.tables[tableBucketArn]
	if !ok {
		return false
	}
	table, ok := nsMap[key][name]
	if !ok {
		return false
	}
	table.MetadataLocation = location
	nsMap[key][name] = table
	return true
}

func (s *Service) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.resourceExists(arn) {
		return false
	}
	if s.resourceTags[arn] == nil {
		s.resourceTags[arn] = map[string]string{}
	}
	for key, value := range tags {
		s.resourceTags[arn][key] = value
	}
	return true
}

func (s *Service) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.resourceExists(arn) {
		return false
	}
	tags := s.resourceTags[arn]
	for _, key := range keys {
		delete(tags, key)
	}
	return true
}

func (s *Service) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.resourceExists(arn) {
		return nil, false
	}
	out := map[string]string{}
	for key, value := range s.resourceTags[arn] {
		out[key] = value
	}
	return out, true
}

func (s *Service) PutTablePolicy(tableArn, policy string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return false
	}
	s.tablePolicies[tableArn] = policy
	return true
}

func (s *Service) GetTablePolicy(tableArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return "", false
	}
	policy := s.tablePolicies[tableArn]
	return policy, policy != ""
}

func (s *Service) DeleteTablePolicy(tableArn string) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return false, false
	}
	if _, ok := s.tablePolicies[tableArn]; !ok || s.tablePolicies[tableArn] == "" {
		return true, false
	}
	delete(s.tablePolicies, tableArn)
	return true, true
}

func (s *Service) PutTableBucketPolicy(bucketArn, policy string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return false
	}
	s.bucketPolicies[bucketArn] = policy
	return true
}

func (s *Service) GetTableBucketPolicy(bucketArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return "", false
	}
	policy := s.bucketPolicies[bucketArn]
	return policy, policy != ""
}

func (s *Service) DeleteTableBucketPolicy(bucketArn string) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return false, false
	}
	if _, ok := s.bucketPolicies[bucketArn]; !ok || s.bucketPolicies[bucketArn] == "" {
		return true, false
	}
	delete(s.bucketPolicies, bucketArn)
	return true, true
}

func (s *Service) PutTableEncryption(tableArn, config string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return false
	}
	s.tableEncryption[tableArn] = config
	return true
}

func (s *Service) GetTableEncryption(tableArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return "", false
	}
	config := s.tableEncryption[tableArn]
	return config, config != ""
}

func (s *Service) DeleteTableEncryption(tableArn string) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return false, false
	}
	if _, ok := s.tableEncryption[tableArn]; !ok || s.tableEncryption[tableArn] == "" {
		return true, false
	}
	delete(s.tableEncryption, tableArn)
	return true, true
}

func (s *Service) PutTableBucketEncryption(bucketArn, config string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return false
	}
	s.bucketEncryption[bucketArn] = config
	return true
}

func (s *Service) GetTableBucketEncryption(bucketArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return "", false
	}
	config := s.bucketEncryption[bucketArn]
	return config, config != ""
}

func (s *Service) DeleteTableBucketEncryption(bucketArn string) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return false, false
	}
	if _, ok := s.bucketEncryption[bucketArn]; !ok || s.bucketEncryption[bucketArn] == "" {
		return true, false
	}
	delete(s.bucketEncryption, bucketArn)
	return true, true
}

func (s *Service) PutTableStorageClass(tableArn, class string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return false
	}
	s.tableStorageClass[tableArn] = class
	return true
}

func (s *Service) GetTableStorageClass(tableArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return "", false
	}
	class := s.tableStorageClass[tableArn]
	return class, class != ""
}

func (s *Service) PutTableBucketStorageClass(bucketArn, class string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return false
	}
	s.bucketStorageClass[bucketArn] = class
	return true
}

func (s *Service) GetTableBucketStorageClass(bucketArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return "", false
	}
	class := s.bucketStorageClass[bucketArn]
	return class, class != ""
}

func (s *Service) PutTableReplication(tableArn, config string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return false
	}
	s.tableReplication[tableArn] = config
	if _, ok := s.tableReplicationStatus[tableArn]; !ok {
		s.tableReplicationStatus[tableArn] = "ACTIVE"
	}
	return true
}

func (s *Service) GetTableReplication(tableArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return "", false
	}
	config := s.tableReplication[tableArn]
	return config, config != ""
}

func (s *Service) DeleteTableReplication(tableArn string) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return false, false
	}
	if _, ok := s.tableReplication[tableArn]; !ok || s.tableReplication[tableArn] == "" {
		return true, false
	}
	delete(s.tableReplication, tableArn)
	delete(s.tableReplicationStatus, tableArn)
	return true, true
}

func (s *Service) GetTableReplicationStatus(tableArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return "", false
	}
	status := s.tableReplicationStatus[tableArn]
	if status == "" {
		status = "PENDING"
	}
	return status, true
}

func (s *Service) PutTableBucketReplication(bucketArn, config string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return false
	}
	s.bucketReplication[bucketArn] = config
	return true
}

func (s *Service) GetTableBucketReplication(bucketArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return "", false
	}
	config := s.bucketReplication[bucketArn]
	return config, config != ""
}

func (s *Service) DeleteTableBucketReplication(bucketArn string) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return false, false
	}
	if _, ok := s.bucketReplication[bucketArn]; !ok || s.bucketReplication[bucketArn] == "" {
		return true, false
	}
	delete(s.bucketReplication, bucketArn)
	return true, true
}

func (s *Service) PutTableMaintenance(tableArn, config string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return false
	}
	s.tableMaintenance[tableArn] = config
	if _, ok := s.tableMaintenanceStatus[tableArn]; !ok {
		s.tableMaintenanceStatus[tableArn] = "ACTIVE"
	}
	return true
}

func (s *Service) GetTableMaintenance(tableArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return "", false
	}
	config := s.tableMaintenance[tableArn]
	return config, config != ""
}

func (s *Service) PutTableBucketMaintenance(bucketArn, config string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return false
	}
	s.bucketMaintenance[bucketArn] = config
	return true
}

func (s *Service) GetTableBucketMaintenance(bucketArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return "", false
	}
	config := s.bucketMaintenance[bucketArn]
	return config, config != ""
}

func (s *Service) GetTableMaintenanceStatus(tableArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return "", false
	}
	status := s.tableMaintenanceStatus[tableArn]
	if status == "" {
		status = "PENDING"
	}
	return status, true
}

func (s *Service) PutTableRecordExpiration(tableArn, config string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return false
	}
	s.tableRecordExpiration[tableArn] = config
	if _, ok := s.recordExpirationStatus[tableArn]; !ok {
		s.recordExpirationStatus[tableArn] = "ACTIVE"
	}
	return true
}

func (s *Service) GetTableRecordExpiration(tableArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return "", false
	}
	config := s.tableRecordExpiration[tableArn]
	return config, config != ""
}

func (s *Service) GetTableRecordExpirationStatus(tableArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return "", false
	}
	status := s.recordExpirationStatus[tableArn]
	if status == "" {
		status = "PENDING"
	}
	return status, true
}

func (s *Service) PutTableBucketMetricsConfig(bucketArn, config string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return false
	}
	s.bucketMetrics[bucketArn] = config
	return true
}

func (s *Service) GetTableBucketMetricsConfig(bucketArn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return "", false
	}
	config := s.bucketMetrics[bucketArn]
	return config, config != ""
}

func (s *Service) DeleteTableBucketMetricsConfig(bucketArn string) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return false, false
	}
	if _, ok := s.bucketMetrics[bucketArn]; !ok || s.bucketMetrics[bucketArn] == "" {
		return true, false
	}
	delete(s.bucketMetrics, bucketArn)
	return true, true
}

func (s *Service) resourceExists(arn string) bool {
	if _, ok := s.tableBuckets[arn]; ok {
		return true
	}
	return s.tableExists(arn)
}

func (s *Service) tableExists(tableArn string) bool {
	for _, nsMap := range s.tables {
		for _, tableMap := range nsMap {
			for _, table := range tableMap {
				if table.TableArn == tableArn {
					return true
				}
			}
		}
	}
	return false
}

func extractBucketName(tableBucketArn string) string {
	const prefix = "arn:aws:s3tables:us-east-1:123456789012:bucket/"
	return strings.TrimPrefix(tableBucketArn, prefix)
}
