package server

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	rdssvc "github.com/stackyard/stackyard/internal/services/rds"
)

// DocumentDB is a Query API on the RDS endpoint: same signing name (rds), API
// version and XML namespace. Requests are told apart by the client's
// User-Agent (or a /docdb path or .docdb. host) and their state lives in the
// shared RDS store, so the rds CLI sees DocumentDB clusters too, as in AWS.

const docdbMaxBodyBytes = 1 << 20

var docdbHandlers = map[string]func(*Server, http.ResponseWriter, *http.Request){
	"CreateDBCluster":                        (*Server).handleDocDBCreateDBCluster,
	"DescribeDBClusters":                     (*Server).handleDocDBDescribeDBClusters,
	"ModifyDBCluster":                        (*Server).handleDocDBModifyDBCluster,
	"DeleteDBCluster":                        (*Server).handleDocDBDeleteDBCluster,
	"StartDBCluster":                         (*Server).handleDocDBStartDBCluster,
	"StopDBCluster":                          (*Server).handleDocDBStopDBCluster,
	"FailoverDBCluster":                      (*Server).handleDocDBFailoverDBCluster,
	"CreateDBInstance":                       (*Server).handleDocDBCreateDBInstance,
	"DescribeDBInstances":                    (*Server).handleDocDBDescribeDBInstances,
	"ModifyDBInstance":                       (*Server).handleDocDBModifyDBInstance,
	"RebootDBInstance":                       (*Server).handleDocDBRebootDBInstance,
	"DeleteDBInstance":                       (*Server).handleDocDBDeleteDBInstance,
	"CreateDBClusterParameterGroup":          (*Server).handleDocDBCreateDBClusterParameterGroup,
	"CopyDBClusterParameterGroup":            (*Server).handleDocDBCopyDBClusterParameterGroup,
	"DeleteDBClusterParameterGroup":          (*Server).handleDocDBDeleteDBClusterParameterGroup,
	"DescribeDBClusterParameterGroups":       (*Server).handleDocDBDescribeDBClusterParameterGroups,
	"DescribeDBClusterParameters":            (*Server).handleDocDBDescribeDBClusterParameters,
	"ModifyDBClusterParameterGroup":          (*Server).handleDocDBModifyDBClusterParameterGroup,
	"ResetDBClusterParameterGroup":           (*Server).handleDocDBResetDBClusterParameterGroup,
	"DescribeEngineDefaultClusterParameters": (*Server).handleDocDBDescribeEngineDefaultClusterParameters,
	"CreateDBClusterSnapshot":                (*Server).handleDocDBCreateDBClusterSnapshot,
	"CopyDBClusterSnapshot":                  (*Server).handleDocDBCopyDBClusterSnapshot,
	"DeleteDBClusterSnapshot":                (*Server).handleDocDBDeleteDBClusterSnapshot,
	"DescribeDBClusterSnapshots":             (*Server).handleDocDBDescribeDBClusterSnapshots,
	"DescribeDBClusterSnapshotAttributes":    (*Server).handleDocDBDescribeDBClusterSnapshotAttributes,
	"ModifyDBClusterSnapshotAttribute":       (*Server).handleDocDBModifyDBClusterSnapshotAttribute,
	"RestoreDBClusterFromSnapshot":           (*Server).handleDocDBRestoreDBClusterFromSnapshot,
	"RestoreDBClusterToPointInTime":          (*Server).handleDocDBRestoreDBClusterToPointInTime,
	"CreateGlobalCluster":                    (*Server).handleDocDBCreateGlobalCluster,
	"DescribeGlobalClusters":                 (*Server).handleDocDBDescribeGlobalClusters,
	"ModifyGlobalCluster":                    (*Server).handleDocDBModifyGlobalCluster,
	"DeleteGlobalCluster":                    (*Server).handleDocDBDeleteGlobalCluster,
	"RemoveFromGlobalCluster":                (*Server).handleDocDBRemoveFromGlobalCluster,
	"FailoverGlobalCluster":                  (*Server).handleDocDBFailoverGlobalCluster,
	"SwitchoverGlobalCluster":                (*Server).handleDocDBSwitchoverGlobalCluster,
	"AddSourceIdentifierToSubscription":      (*Server).handleDocDBAddSourceIdentifierToSubscription,
	"RemoveSourceIdentifierFromSubscription": (*Server).handleDocDBRemoveSourceIdentifierFromSubscription,
	"AddTagsToResource":                      (*Server).handleDocDBAddTagsToResource,
	"ListTagsForResource":                    (*Server).handleDocDBListTagsForResource,
	"RemoveTagsFromResource":                 (*Server).handleDocDBRemoveTagsFromResource,
	"DeleteDBSubnetGroup":                    (*Server).handleDocDBDeleteDBSubnetGroup,
	"DescribeDBEngineVersions":               (*Server).handleDocDBDescribeDBEngineVersions,
	"DescribeOrderableDBInstanceOptions":     (*Server).handleDocDBDescribeOrderableDBInstanceOptions,
	"DescribeEventCategories":                (*Server).handleDocDBDescribeEventCategories,
	"DescribeServerlessV2PlatformVersions":   (*Server).handleDocDBDescribeServerlessV2PlatformVersions,
}

// docdbDelegatedToRDS are DocumentDB operations whose RDS implementation is
// already right for DocumentDB (same store, same shapes).
var docdbDelegatedToRDS = []string{
	"CreateDBSubnetGroup", "DescribeDBSubnetGroups", "ModifyDBSubnetGroup",
	"CreateEventSubscription", "DescribeEventSubscriptions", "ModifyEventSubscription", "DeleteEventSubscription",
	"DescribePendingMaintenanceActions", "ApplyPendingMaintenanceAction", "DescribeCertificates", "DescribeEvents",
}

func (s *Server) handleDocDBQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDocDBQueryCandidate(r) {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "rds"); !ok {
		respondRDSErrorXML(w, status, code, msg)
		return true
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		respondRDSErrorXML(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}
	// The body is already buffered (SigV4 read it, capped at 25 MiB); Query
	// requests are small, so refuse anything past 1 MiB before parsing.
	if body, err := readBodyBytes(r); err != nil || len(body) > docdbMaxBodyBytes {
		respondRDSErrorXML(w, http.StatusRequestEntityTooLarge, "RequestEntityTooLarge", "request body exceeds 1 MiB")
		return true
	}
	if err := r.ParseForm(); err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid form body")
		return true
	}
	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondRDSErrorXML(w, http.StatusBadRequest, "MissingAction", "Action is required")
		return true
	}
	if _, ok := docdbOperationByName[action]; !ok {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidAction", "The action "+action+" is not valid for this web service.")
		return true
	}
	if version := strings.TrimSpace(r.Form.Get("Version")); version != "" && version != "2014-10-31" {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "unsupported Version")
		return true
	}
	if handler, ok := docdbHandlers[action]; ok {
		handler(s, w, r)
		return true
	}
	if handler, ok := rdsQueryHandlers[action]; ok {
		handler(s, w, r)
		return true
	}
	respondRDSErrorXML(w, http.StatusNotImplemented, "NotImplemented", "operation not implemented")
	return true
}

// isDocDBQueryCandidate claims only requests that say they're DocumentDB:
// everything else signed rds belongs to Neptune or RDS.
func isDocDBQueryCandidate(r *http.Request) bool {
	if !hasDocDBRequestHint(r) {
		return false
	}
	if service := strings.TrimSpace(sigV4ServiceHint(r)); service != "" && service != "rds" {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		return false
	}
	if r.URL.Query().Get("Action") != "" {
		return true
	}
	if r.Method != http.MethodPost || !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/x-www-form-urlencoded") {
		return false
	}
	body, err := readBodyBytes(r)
	if err != nil {
		return false
	}
	values, err := url.ParseQuery(string(body))
	return err == nil && values.Get("Action") != ""
}

func hasDocDBRequestHint(r *http.Request) bool {
	host := strings.ToLower(r.Host)
	if strings.Contains(host, ".docdb.") || strings.HasPrefix(host, "docdb.") || strings.HasPrefix(r.URL.Path, "/docdb") {
		return true
	}
	for _, ua := range []string{r.Header.Get("User-Agent"), r.Header.Get("X-Amz-User-Agent")} {
		ua = strings.ToLower(ua)
		// "command#docdb." excludes the separate docdb-elastic CLI.
		if strings.Contains(ua, "command#docdb.") || strings.Contains(ua, "api/docdb#") || strings.Contains(ua, "api/docdb/") {
			return true
		}
	}
	return false
}

// respondDocDBError reports a store error with its AWS code and status.
func respondDocDBError(w http.ResponseWriter, action string, err error) {
	var f *rdssvc.Fault
	if !errors.As(err, &f) {
		respondRDSServiceError(w, action, err)
		return
	}
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, rdssvc.ErrNotFound):
		status = http.StatusNotFound
	case f.Code == "DBClusterQuotaExceededFault":
		status = http.StatusForbidden
	}
	respondRDSErrorXML(w, status, f.Code, f.Message)
}

func docdbTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// ---- XML shapes (docdb service-2.json) ----

type docdbVpcSGXML struct {
	VpcSecurityGroupID string `xml:"VpcSecurityGroupId"`
	Status             string `xml:"Status"`
}

type docdbClusterMemberXML struct {
	DBInstanceIdentifier          string `xml:"DBInstanceIdentifier"`
	IsClusterWriter               bool   `xml:"IsClusterWriter"`
	DBClusterParameterGroupStatus string `xml:"DBClusterParameterGroupStatus"`
	PromotionTier                 int    `xml:"PromotionTier"`
}

type docdbClusterXML struct {
	AvailabilityZones          []string                `xml:"AvailabilityZones>AvailabilityZone"`
	BackupRetentionPeriod      int                     `xml:"BackupRetentionPeriod"`
	DBClusterIdentifier        string                  `xml:"DBClusterIdentifier"`
	DBClusterParameterGroup    string                  `xml:"DBClusterParameterGroup,omitempty"`
	DBSubnetGroup              string                  `xml:"DBSubnetGroup,omitempty"`
	Status                     string                  `xml:"Status"`
	EarliestRestorableTime     string                  `xml:"EarliestRestorableTime,omitempty"`
	Endpoint                   string                  `xml:"Endpoint,omitempty"`
	ReaderEndpoint             string                  `xml:"ReaderEndpoint,omitempty"`
	MultiAZ                    bool                    `xml:"MultiAZ"`
	Engine                     string                  `xml:"Engine"`
	EngineVersion              string                  `xml:"EngineVersion,omitempty"`
	LatestRestorableTime       string                  `xml:"LatestRestorableTime,omitempty"`
	Port                       int                     `xml:"Port,omitempty"`
	MasterUsername             string                  `xml:"MasterUsername,omitempty"`
	PreferredBackupWindow      string                  `xml:"PreferredBackupWindow,omitempty"`
	PreferredMaintenanceWindow string                  `xml:"PreferredMaintenanceWindow,omitempty"`
	DBClusterMembers           []docdbClusterMemberXML `xml:"DBClusterMembers>DBClusterMember"`
	VpcSecurityGroups          []docdbVpcSGXML         `xml:"VpcSecurityGroups>VpcSecurityGroupMembership"`
	HostedZoneID               string                  `xml:"HostedZoneId"`
	StorageEncrypted           bool                    `xml:"StorageEncrypted"`
	KmsKeyID                   string                  `xml:"KmsKeyId,omitempty"`
	DbClusterResourceID        string                  `xml:"DbClusterResourceId,omitempty"`
	DBClusterArn               string                  `xml:"DBClusterArn"`
	ClusterCreateTime          string                  `xml:"ClusterCreateTime,omitempty"`
	EnabledLogs                []string                `xml:"EnabledCloudwatchLogsExports>member"`
	DeletionProtection         bool                    `xml:"DeletionProtection"`
	StorageType                string                  `xml:"StorageType,omitempty"`
	CopyTagsToSnapshot         bool                    `xml:"CopyTagsToSnapshot"`
}

const docdbHostedZoneID = "ZNKXH85TT8WVW"

func docdbClusterToXML(c rdssvc.DBCluster) docdbClusterXML {
	out := docdbClusterXML{
		AvailabilityZones: c.AvailabilityZones, BackupRetentionPeriod: c.BackupRetentionPeriod,
		DBClusterIdentifier: c.Identifier, DBClusterParameterGroup: c.DBClusterParameterGroup,
		DBSubnetGroup: c.DBSubnetGroupName, Status: c.Status, Endpoint: c.Endpoint, ReaderEndpoint: c.ReaderEndpoint,
		MultiAZ: len(c.Members) > 1, Engine: c.Engine, EngineVersion: c.EngineVersion, Port: c.Port,
		MasterUsername: c.MasterUsername, PreferredBackupWindow: c.PreferredBackupWindow,
		PreferredMaintenanceWindow: c.PreferredMaintenanceWindow, HostedZoneID: docdbHostedZoneID,
		StorageEncrypted: c.StorageEncrypted, KmsKeyID: c.KmsKeyID, DbClusterResourceID: c.ResourceID,
		DBClusterArn: c.ARN, ClusterCreateTime: docdbTime(c.CreatedAt), EnabledLogs: c.LogExports,
		DeletionProtection: c.DeletionProtection, StorageType: c.StorageType, CopyTagsToSnapshot: c.CopyTagsToSnapshot,
	}
	if !c.CreatedAt.IsZero() {
		out.EarliestRestorableTime = docdbTime(c.CreatedAt)
		out.LatestRestorableTime = docdbTime(time.Now())
	}
	for _, m := range c.Members {
		out.DBClusterMembers = append(out.DBClusterMembers, docdbClusterMemberXML{
			DBInstanceIdentifier: m.InstanceID, IsClusterWriter: m.IsWriter, DBClusterParameterGroupStatus: "in-sync", PromotionTier: m.PromotionTier,
		})
	}
	for _, sg := range c.VpcSecurityGroupIDs {
		out.VpcSecurityGroups = append(out.VpcSecurityGroups, docdbVpcSGXML{VpcSecurityGroupID: sg, Status: "active"})
	}
	return out
}

type docdbEndpointXML struct {
	Address      string `xml:"Address"`
	Port         int    `xml:"Port"`
	HostedZoneID string `xml:"HostedZoneId"`
}

type docdbSubnetGroupRefXML struct {
	DBSubnetGroupName string `xml:"DBSubnetGroupName"`
	SubnetGroupStatus string `xml:"SubnetGroupStatus"`
}

type docdbInstanceXML struct {
	DBInstanceIdentifier       string                  `xml:"DBInstanceIdentifier"`
	DBInstanceClass            string                  `xml:"DBInstanceClass"`
	Engine                     string                  `xml:"Engine"`
	DBInstanceStatus           string                  `xml:"DBInstanceStatus"`
	Endpoint                   *docdbEndpointXML       `xml:"Endpoint,omitempty"`
	InstanceCreateTime         string                  `xml:"InstanceCreateTime,omitempty"`
	BackupRetentionPeriod      int                     `xml:"BackupRetentionPeriod"`
	AvailabilityZone           string                  `xml:"AvailabilityZone,omitempty"`
	DBSubnetGroup              *docdbSubnetGroupRefXML `xml:"DBSubnetGroup,omitempty"`
	PreferredMaintenanceWindow string                  `xml:"PreferredMaintenanceWindow,omitempty"`
	EngineVersion              string                  `xml:"EngineVersion,omitempty"`
	AutoMinorVersionUpgrade    bool                    `xml:"AutoMinorVersionUpgrade"`
	PubliclyAccessible         bool                    `xml:"PubliclyAccessible"`
	DBClusterIdentifier        string                  `xml:"DBClusterIdentifier,omitempty"`
	StorageEncrypted           bool                    `xml:"StorageEncrypted"`
	DbiResourceID              string                  `xml:"DbiResourceId,omitempty"`
	CACertificateIdentifier    string                  `xml:"CACertificateIdentifier,omitempty"`
	PromotionTier              int                     `xml:"PromotionTier"`
	DBInstanceArn              string                  `xml:"DBInstanceArn"`
}

func (s *Server) docdbInstanceToXML(i rdssvc.DBInstance) docdbInstanceXML {
	out := docdbInstanceXML{
		DBInstanceIdentifier: i.Identifier, DBInstanceClass: i.DBInstanceClass, Engine: i.Engine,
		DBInstanceStatus: i.Status, InstanceCreateTime: docdbTime(i.CreatedAt), BackupRetentionPeriod: i.BackupRetentionPeriod,
		AvailabilityZone: i.AvailabilityZone, PreferredMaintenanceWindow: i.PreferredMaintenanceWindow,
		EngineVersion: i.EngineVersion, AutoMinorVersionUpgrade: i.AutoMinorVersionUpgrade,
		PubliclyAccessible: i.PubliclyAccessible, DBClusterIdentifier: i.DBClusterIdentifier,
		DbiResourceID: rdssvc.ResourceID(i.Identifier), CACertificateIdentifier: i.CACertificateIdentifier,
		PromotionTier: i.PromotionTier, DBInstanceArn: i.ARN,
	}
	if i.EndpointAddress != "" {
		out.Endpoint = &docdbEndpointXML{Address: i.EndpointAddress, Port: i.Port, HostedZoneID: docdbHostedZoneID}
	}
	if i.DBSubnetGroupName != "" {
		out.DBSubnetGroup = &docdbSubnetGroupRefXML{DBSubnetGroupName: i.DBSubnetGroupName, SubnetGroupStatus: "Complete"}
	}
	if i.DBClusterIdentifier != "" {
		if c, _, err := s.rds.DescribeDocDBClusters(rdssvc.DocDBClusterFilter{Identifier: i.DBClusterIdentifier}, "", 0); err == nil && len(c) == 1 {
			out.StorageEncrypted = c[0].StorageEncrypted
		}
	}
	return out
}

type docdbSnapshotXML struct {
	AvailabilityZones           []string `xml:"AvailabilityZones>AvailabilityZone"`
	DBClusterSnapshotIdentifier string   `xml:"DBClusterSnapshotIdentifier"`
	DBClusterIdentifier         string   `xml:"DBClusterIdentifier"`
	SnapshotCreateTime          string   `xml:"SnapshotCreateTime"`
	Engine                      string   `xml:"Engine"`
	Status                      string   `xml:"Status"`
	Port                        int      `xml:"Port"`
	ClusterCreateTime           string   `xml:"ClusterCreateTime,omitempty"`
	MasterUsername              string   `xml:"MasterUsername,omitempty"`
	EngineVersion               string   `xml:"EngineVersion,omitempty"`
	SnapshotType                string   `xml:"SnapshotType"`
	PercentProgress             int      `xml:"PercentProgress"`
	StorageEncrypted            bool     `xml:"StorageEncrypted"`
	KmsKeyID                    string   `xml:"KmsKeyId,omitempty"`
	DBClusterSnapshotArn        string   `xml:"DBClusterSnapshotArn"`
	SourceDBClusterSnapshotArn  string   `xml:"SourceDBClusterSnapshotArn,omitempty"`
	StorageType                 string   `xml:"StorageType,omitempty"`
}

func docdbSnapshotToXML(s rdssvc.DBClusterSnapshot) docdbSnapshotXML {
	return docdbSnapshotXML{
		AvailabilityZones: s.AZs, DBClusterSnapshotIdentifier: s.Identifier, DBClusterIdentifier: s.ClusterID,
		SnapshotCreateTime: docdbTime(s.CreatedAt), Engine: s.Engine, Status: s.Status, Port: s.Port,
		ClusterCreateTime: docdbTime(s.ClusterCreatedAt), MasterUsername: s.MasterUsername, EngineVersion: s.EngineVersion,
		SnapshotType: s.SnapshotType, PercentProgress: 100, StorageEncrypted: s.StorageEncrypted, KmsKeyID: s.KmsKeyID,
		DBClusterSnapshotArn: s.ARN, SourceDBClusterSnapshotArn: s.SourceARN, StorageType: s.StorageType,
	}
}

type docdbParamGroupXML struct {
	DBClusterParameterGroupName string `xml:"DBClusterParameterGroupName"`
	DBParameterGroupFamily      string `xml:"DBParameterGroupFamily"`
	Description                 string `xml:"Description"`
	DBClusterParameterGroupArn  string `xml:"DBClusterParameterGroupArn"`
}

func docdbParamGroupToXML(g rdssvc.DBClusterParameterGroup) docdbParamGroupXML {
	return docdbParamGroupXML{DBClusterParameterGroupName: g.Name, DBParameterGroupFamily: g.Family, Description: g.Description, DBClusterParameterGroupArn: g.ARN}
}

type docdbParameterXML struct {
	ParameterName        string `xml:"ParameterName"`
	ParameterValue       string `xml:"ParameterValue"`
	Description          string `xml:"Description"`
	Source               string `xml:"Source"`
	ApplyType            string `xml:"ApplyType"`
	DataType             string `xml:"DataType"`
	AllowedValues        string `xml:"AllowedValues"`
	IsModifiable         bool   `xml:"IsModifiable"`
	MinimumEngineVersion string `xml:"MinimumEngineVersion"`
	ApplyMethod          string `xml:"ApplyMethod"`
}

func docdbParametersToXML(params []rdssvc.ClusterParameter) []docdbParameterXML {
	out := make([]docdbParameterXML, 0, len(params))
	for _, p := range params {
		out = append(out, docdbParameterXML{
			ParameterName: p.Name, ParameterValue: p.Value, Description: p.Description, Source: p.Source,
			ApplyType: p.ApplyType, DataType: p.DataType, AllowedValues: p.AllowedValues, IsModifiable: true,
			MinimumEngineVersion: p.MinVersion, ApplyMethod: p.ApplyMethod,
		})
	}
	return out
}

type docdbGlobalMemberXML struct {
	DBClusterArn          string   `xml:"DBClusterArn"`
	Readers               []string `xml:"Readers>member"`
	IsWriter              bool     `xml:"IsWriter"`
	SynchronizationStatus string   `xml:"SynchronizationStatus,omitempty"`
}

type docdbGlobalXML struct {
	GlobalClusterIdentifier string                 `xml:"GlobalClusterIdentifier"`
	GlobalClusterResourceID string                 `xml:"GlobalClusterResourceId"`
	GlobalClusterArn        string                 `xml:"GlobalClusterArn"`
	Status                  string                 `xml:"Status"`
	Engine                  string                 `xml:"Engine"`
	EngineVersion           string                 `xml:"EngineVersion"`
	DatabaseName            string                 `xml:"DatabaseName,omitempty"`
	StorageEncrypted        bool                   `xml:"StorageEncrypted"`
	DeletionProtection      bool                   `xml:"DeletionProtection"`
	GlobalClusterMembers    []docdbGlobalMemberXML `xml:"GlobalClusterMembers>GlobalClusterMember"`
}

func docdbGlobalToXML(g rdssvc.GlobalCluster) docdbGlobalXML {
	out := docdbGlobalXML{
		GlobalClusterIdentifier: g.Identifier, GlobalClusterResourceID: "cluster-" + strings.ToUpper(strings.ReplaceAll(g.Identifier, "-", "")),
		GlobalClusterArn: g.ARN, Status: g.Status, Engine: firstNonEmpty(g.Engine, rdssvc.DocDBEngine),
		EngineVersion: g.EngineVersion, DatabaseName: g.DatabaseName, StorageEncrypted: g.StorageEncrypted,
		DeletionProtection: g.DeletionProtection,
	}
	for _, arn := range g.Members {
		m := docdbGlobalMemberXML{DBClusterArn: arn, IsWriter: arn == g.SourceDBClusterArn}
		if m.IsWriter {
			for _, other := range g.Members {
				if other != arn {
					m.Readers = append(m.Readers, other)
				}
			}
		} else {
			m.SynchronizationStatus = "connected"
		}
		out.GlobalClusterMembers = append(out.GlobalClusterMembers, m)
	}
	return out
}

// ---- form helpers ----

func docdbFormString(r *http.Request, name string) string {
	return strings.TrimSpace(r.Form.Get(name))
}

func docdbFormList(r *http.Request, prefix string) []string {
	return parseRDSListMembers(r.Form, prefix)
}

// docdbFilters reads Filters.Filter.N.{Name,Values.Value.M} into a map.
func docdbFilters(r *http.Request) map[string][]string {
	out := map[string][]string{}
	for i := 1; i <= 50; i++ {
		base := "Filters.Filter." + strconv.Itoa(i)
		name := docdbFormString(r, base+".Name")
		if name == "" {
			if i > 1 {
				break
			}
			continue
		}
		out[name] = append(out[name], parseRDSListMembers(r.Form, base+".Values.Value")...)
	}
	return out
}

// docdbCheckFilters rejects filter names the operation doesn't support.
func docdbCheckFilters(w http.ResponseWriter, filters map[string][]string, allowed ...string) bool {
	for name, values := range filters {
		ok := false
		for _, a := range allowed {
			ok = ok || a == name
		}
		if !ok {
			respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Unrecognized filter name: "+name)
			return false
		}
		if len(values) == 0 {
			respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "filter "+name+" needs at least one value")
			return false
		}
	}
	return true
}

// docdbIntParam parses an optional non-negative integer field, writing the
// error response itself when the value is bad.
func docdbIntParam(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	n, err := parseOptionalRDSInt(r.Form.Get(name))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", name+" must be a non-negative integer")
		return 0, false
	}
	return n, true
}

func docdbBoolParam(w http.ResponseWriter, r *http.Request, name string) (*bool, bool) {
	b, err := parseOptionalRDSBoolPtr(r.Form.Get(name))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", name+" must be true or false")
		return nil, false
	}
	return b, true
}

func docdbRequire(w http.ResponseWriter, r *http.Request, names ...string) bool {
	for _, name := range names {
		if docdbFormString(r, name) == "" {
			respondRDSErrorXML(w, http.StatusBadRequest, "MissingParameter", "The request must contain the parameter "+name)
			return false
		}
	}
	return true
}

func docdbPagingParams(w http.ResponseWriter, r *http.Request) (string, int, bool) {
	n, ok := docdbIntParam(w, r, "MaxRecords")
	return docdbFormString(r, "Marker"), n, ok
}
