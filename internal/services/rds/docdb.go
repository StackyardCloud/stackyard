package rds

import (
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// DocumentDB clusters, instances, global clusters, subnet groups, event
// subscriptions and tags live in the shared RDS maps, as they do in AWS (the
// rds CLI lists DocumentDB clusters too). The docdb*.go files add what the RDS side
// doesn't model yet: cluster parameter groups, cluster snapshots, cluster
// membership and the cluster-family lifecycle rules.

// Fault is a store error with the AWS error code to report. It unwraps to one
// of the package sentinels, so errors.Is keeps working for callers that only
// care about the kind.
type Fault struct {
	Kind    error
	Code    string
	Message string
}

func (f *Fault) Error() string { return f.Code + ": " + f.Message }
func (f *Fault) Unwrap() error { return f.Kind }

func fault(kind error, code, format string, args ...any) error {
	return &Fault{Kind: kind, Code: code, Message: fmt.Sprintf(format, args...)}
}

// ErrQuotaExceeded is the kind for *QuotaExceeded faults.
var ErrQuotaExceeded = errors.New("quota exceeded")

const (
	DocDBEngine         = "docdb"
	docdbDefaultVersion = "5.0.0"
	docdbDefaultPort    = 27017

	// ponytail: AWS's per-region quotas are lower (40 clusters, 100
	// snapshots); these only bound memory on a long-lived emulator that
	// repeated test runs keep adding to.
	docdbMaxClusters     = 500
	docdbMaxInstances    = 500
	docdbMaxSnapshots    = 2000
	docdbMaxParamGroups  = 500
	docdbMaxSharedAccts  = 20
	docdbMaxTagsPerARN   = 50
	docdbMaxModifyParams = 20
)

// DocDBEngineVersions maps each supported engine version to its parameter
// group family.
var DocDBEngineVersions = []struct{ Version, Family string }{
	{"3.6.0", "docdb3.6"}, {"4.0.0", "docdb4.0"}, {"5.0.0", "docdb5.0"}, {"8.0.0", "docdb8.0"},
}

// DocDBInstanceClasses are the orderable instance classes.
var DocDBInstanceClasses = []string{
	"db.r5.large", "db.r5.xlarge", "db.r6g.large", "db.r6g.xlarge", "db.r6g.2xlarge",
	"db.serverless", "db.t3.medium", "db.t4g.medium",
}

var docdbAZs = []string{defaultRegion + "a", defaultRegion + "b", defaultRegion + "c"}

// ClusterParameter is one entry of a cluster parameter group.
type ClusterParameter struct {
	Name          string
	Value         string
	Description   string
	Source        string // "system" or "user"
	ApplyType     string // "dynamic" or "static"
	DataType      string
	AllowedValues string
	ApplyMethod   string
	MinVersion    string
}

// docdbParameterCatalog is the parameter set every docdb family has.
// ponytail: one catalog for all families; split per family if versions diverge.
var docdbParameterCatalog = []ClusterParameter{
	{Name: "audit_logs", Value: "disabled", Description: "Enables auditing on cluster.", ApplyType: "dynamic", DataType: "string", AllowedValues: "enabled,disabled,ddl,dml_read,dml_write,all,none"},
	{Name: "change_stream_log_retention_duration", Value: "10800", Description: "Duration of time in seconds that the change stream log is retained and can be consumed.", ApplyType: "dynamic", DataType: "integer", AllowedValues: "3600-604800"},
	{Name: "profiler", Value: "disabled", Description: "Enables profiling for slow operations.", ApplyType: "dynamic", DataType: "string", AllowedValues: "enabled,disabled"},
	{Name: "profiler_sampling_rate", Value: "1.0", Description: "Rate for sampling profiled operations.", ApplyType: "dynamic", DataType: "float", AllowedValues: "0.0-1.0"},
	{Name: "profiler_threshold_ms", Value: "100", Description: "Operations longer than profiler_threshold_ms will be logged.", ApplyType: "dynamic", DataType: "integer", AllowedValues: "50-2147483646"},
	{Name: "tls", Value: "enabled", Description: "Config to enable/disable TLS.", ApplyType: "static", DataType: "string", AllowedValues: "disabled,enabled,fips-140-3"},
	{Name: "ttl_monitor", Value: "enabled", Description: "Enables TTL Monitoring.", ApplyType: "dynamic", DataType: "string", AllowedValues: "disabled,enabled"},
}

type DBClusterParameterGroup struct {
	Name        string
	ARN         string
	Family      string
	Description string
	Values      map[string]ClusterParameter // user-modified parameters
}

type DBClusterSnapshot struct {
	Identifier       string
	ARN              string
	ClusterID        string
	Engine           string
	EngineVersion    string
	Status           string
	SnapshotType     string // manual, automated
	Port             int
	MasterUsername   string
	StorageEncrypted bool
	KmsKeyID         string
	StorageType      string
	SourceARN        string
	AZs              []string
	ClusterCreatedAt time.Time
	CreatedAt        time.Time
	RestoreAccounts  []string // "restore" attribute values
}

func cloneClusterSnapshot(in *DBClusterSnapshot) DBClusterSnapshot {
	out := *in
	out.AZs = append([]string(nil), in.AZs...)
	out.RestoreAccounts = append([]string(nil), in.RestoreAccounts...)
	return out
}

func cloneClusterParamGroup(in *DBClusterParameterGroup) DBClusterParameterGroup {
	out := *in
	out.Values = make(map[string]ClusterParameter, len(in.Values))
	for k, v := range in.Values {
		out.Values[k] = v
	}
	return out
}

func clusterSnapshotARN(id string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:cluster-snapshot:%s", defaultRegion, defaultAccountID, id)
}

func clusterParamGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:cluster-pg:%s", defaultRegion, defaultAccountID, name)
}

// docdbResourceID derives a stable resource id like AWS's "cluster-ABC...".
func docdbResourceID(prefix, id string) string {
	sum := sha256.Sum256([]byte(prefix + "/" + id + "/" + time.Now().UTC().Format(time.RFC3339Nano)))
	return prefix + "-" + base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])[:26]
}

var (
	docdbIdentifierPattern = regexp.MustCompile(`^[A-Za-z](?:[A-Za-z0-9]|-[A-Za-z0-9])*$`)
	docdbClassPattern      = regexp.MustCompile(`^db\.[a-z0-9]+\.[a-z0-9]+$`)
	docdbWindowPattern     = regexp.MustCompile(`^(?:(mon|tue|wed|thu|fri|sat|sun):)?([01]\d|2[0-3]):[0-5]\d-(?:(mon|tue|wed|thu|fri|sat|sun):)?([01]\d|2[0-3]):[0-5]\d$`)
)

func validateDocDBIdentifier(field, value string, maxLen int) error {
	if value == "" {
		return fault(ErrInvalidParameter, "InvalidParameterValue", "%s is required", field)
	}
	if len(value) > maxLen || !docdbIdentifierPattern.MatchString(value) {
		return fault(ErrInvalidParameter, "InvalidParameterValue",
			"%s must be 1 to %d letters, numbers or hyphens, start with a letter, and not end with a hyphen or contain two consecutive hyphens", field, maxLen)
	}
	return nil
}

// DocDBFamilyForVersion returns the parameter group family of an engine
// version such as "5.0.0".
func DocDBFamilyForVersion(version string) (string, bool) {
	for _, v := range DocDBEngineVersions {
		if v.Version == version {
			return v.Family, true
		}
	}
	return "", false
}

func docdbKnownFamily(family string) bool {
	for _, v := range DocDBEngineVersions {
		if v.Family == family {
			return true
		}
	}
	return false
}

// arnOrName returns the resource id from an ARN of the given type marker
// (e.g. ":cluster:"), or the value unchanged.
func arnOrName(value, marker string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:") {
		if i := strings.Index(value, marker); i >= 0 {
			return value[i+len(marker):]
		}
	}
	return value
}

// docdbPage paginates with MaxRecords 20-100 (default 100), as the API does.
func docdbPage(total int, marker string, maxRecords int) (int, int, string, error) {
	if maxRecords != 0 && (maxRecords < 20 || maxRecords > 100) {
		return 0, 0, "", fault(ErrInvalidParameter, "InvalidParameterValue", "MaxRecords must be between 20 and 100")
	}
	start, end, next, err := paginate(total, marker, maxRecords, 100)
	if err != nil {
		return 0, 0, "", fault(ErrInvalidParameter, "InvalidParameterValue", "invalid Marker")
	}
	return start, end, next, nil
}

func containsString(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// DocDBPage exposes the DocumentDB pagination rules to the server's static
// catalogs.
func DocDBPage(total int, marker string, maxRecords int) (int, int, string, error) {
	return docdbPage(total, marker, maxRecords)
}
