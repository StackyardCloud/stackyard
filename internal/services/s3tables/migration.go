package s3tables

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// MaxTableMigrations bounds stored migrations; StartTableMigration fails once reached.
const MaxTableMigrations = 1000

// Migration error codes, mapped to HTTP statuses by the server.
const (
	MigrationNotFound = "NotFoundException"
	MigrationConflict = "ConflictException"
	MigrationLimit    = "TooManyRequestsException"
)

// TableMigration is a Glue-to-S3-Tables migration. Request sub-documents are kept as JSON.
type TableMigration struct {
	MigrationID         string
	TableBucketARN      string
	Namespace           string
	Name                string
	TableARN            string
	Type                string
	State               string
	RoleARN             string
	ReviewMigrationPlan bool
	CreatedAt           time.Time
	StateUpdatedAt      time.Time
	SourceType          string
	Source              json.RawMessage
	DataMigration       json.RawMessage
	MigrationOutput     json.RawMessage
	seq                 int
}

// MigrationFilter narrows ListTableMigrations; empty fields match everything.
type MigrationFilter struct {
	TableBucketARN, Namespace, Name, TableARN, State string
}

func migrationInitialState(review bool) string {
	if review {
		return "pendingReview"
	}
	return "queued"
}

// StartTableMigration creates the destination table and records the migration.
func (s *Service) StartTableMigration(m TableMigration) (TableMigration, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.migrations) >= MaxTableMigrations {
		return TableMigration{}, MigrationLimit
	}
	if _, ok := s.tableBuckets[m.TableBucketARN]; !ok {
		return TableMigration{}, MigrationNotFound
	}
	if !s.namespaceExistsLocked(m.TableBucketARN, m.Namespace) {
		return TableMigration{}, MigrationNotFound
	}
	if _, exists := s.tables[m.TableBucketARN][m.Namespace][m.Name]; exists {
		return TableMigration{}, MigrationConflict
	}
	table, _ := s.createTableLocked(m.TableBucketARN, []string{m.Namespace}, m.Name)
	m.TableARN = table.TableArn
	m.Type = "start"
	return s.recordMigrationLocked(m), ""
}

// SyncTableMigration starts an incremental sync for a table that was migrated before.
func (s *Service) SyncTableMigration(tableARN, roleARN string, review bool, output json.RawMessage) (TableMigration, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.migrations) >= MaxTableMigrations {
		return TableMigration{}, MigrationLimit
	}
	prev, ok := s.latestMigrationLocked(tableARN)
	if !ok || !s.tableExists(tableARN) {
		return TableMigration{}, MigrationNotFound
	}
	m := prev
	m.Type = "sync"
	m.RoleARN = roleARN
	m.ReviewMigrationPlan = review
	m.MigrationOutput = output
	return s.recordMigrationLocked(m), ""
}

func (s *Service) recordMigrationLocked(m TableMigration) TableMigration {
	now := time.Now().UTC()
	m.seq = s.nextMigrationID
	m.MigrationID = fmt.Sprintf("tm-%016x", s.nextMigrationID)
	s.nextMigrationID++
	m.State = migrationInitialState(m.ReviewMigrationPlan)
	m.CreatedAt, m.StateUpdatedAt = now, now
	s.migrations[m.MigrationID] = m
	return m
}

// GetTableMigration looks a migration up by ID, or the latest one for tableARN.
func (s *Service) GetTableMigration(id, tableARN string) (TableMigration, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.findMigrationLocked(id, tableARN)
}

// UpdateTableMigration approves a reviewed plan (queued) or cancels a migration.
func (s *Service) UpdateTableMigration(id, tableARN, requested string) (TableMigration, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.findMigrationLocked(id, tableARN)
	if !ok {
		return TableMigration{}, MigrationNotFound
	}
	switch {
	case m.State == requested:
	case requested == "queued" && m.State == "pendingReview",
		requested == "cancelled" && !migrationTerminal(m.State):
		m.State = requested
		m.StateUpdatedAt = time.Now().UTC()
		s.migrations[m.MigrationID] = m
	default:
		return TableMigration{}, MigrationConflict
	}
	return m, ""
}

// CompleteTableMigration marks the table's latest migration as succeeded.
func (s *Service) CompleteTableMigration(tableARN string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.latestMigrationLocked(tableARN)
	if !ok {
		return MigrationNotFound
	}
	if m.State == "cancelled" || m.State == "failed" {
		return MigrationConflict
	}
	m.State = "succeeded"
	m.StateUpdatedAt = time.Now().UTC()
	s.migrations[m.MigrationID] = m
	return ""
}

func (s *Service) ListTableMigrations(f MigrationFilter) []TableMigration {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]TableMigration, 0, len(s.migrations))
	for _, m := range s.migrations {
		if (f.TableBucketARN == "" || f.TableBucketARN == m.TableBucketARN) &&
			(f.Namespace == "" || f.Namespace == m.Namespace) &&
			(f.Name == "" || f.Name == m.Name) &&
			(f.TableARN == "" || f.TableARN == m.TableARN) &&
			(f.State == "" || f.State == m.State) {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].seq < out[j].seq })
	return out
}

func (s *Service) findMigrationLocked(id, tableARN string) (TableMigration, bool) {
	if id != "" {
		m, ok := s.migrations[id]
		return m, ok && (tableARN == "" || m.TableARN == tableARN)
	}
	return s.latestMigrationLocked(tableARN)
}

func (s *Service) latestMigrationLocked(tableARN string) (TableMigration, bool) {
	var best TableMigration
	found := false
	for _, m := range s.migrations {
		if tableARN != "" && m.TableARN == tableARN && (!found || m.seq > best.seq) {
			best, found = m, true
		}
	}
	return best, found
}

func migrationTerminal(state string) bool {
	return state == "succeeded" || state == "failed" || state == "cancelled"
}

func (s *Service) namespaceExistsLocked(bucketARN, namespace string) bool {
	for _, ns := range s.namespaces[bucketARN] {
		if strings.Join(ns.Namespace, ".") == namespace {
			return true
		}
	}
	return false
}

// PutTableBucketLogging stores the bucket's logging status document.
func (s *Service) PutTableBucketLogging(bucketArn string, status json.RawMessage) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return false
	}
	s.bucketLogging[bucketArn] = append(json.RawMessage(nil), status...)
	return true
}

// GetTableBucketLogging returns the stored status, or nil when logging was never configured.
func (s *Service) GetTableBucketLogging(bucketArn string) (json.RawMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tableBuckets[bucketArn]; !ok {
		return nil, false
	}
	return append(json.RawMessage(nil), s.bucketLogging[bucketArn]...), true
}

// DisassociateTableReplication detaches a table from replication; false if the table is unknown.
func (s *Service) DisassociateTableReplication(tableArn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tableExists(tableArn) {
		return false
	}
	delete(s.tableReplication, tableArn)
	delete(s.tableReplicationStatus, tableArn)
	return true
}
