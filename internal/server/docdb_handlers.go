package server

import (
	"encoding/xml"
	"net/http"
	"time"

	rdssvc "github.com/stackyard/stackyard/internal/services/rds"
)

// ---- clusters ----

func docdbClusterInput(w http.ResponseWriter, r *http.Request) (rdssvc.DocDBClusterInput, bool) {
	port, ok := docdbIntParam(w, r, "Port")
	if !ok {
		return rdssvc.DocDBClusterInput{}, false
	}
	retention, ok := docdbIntParam(w, r, "BackupRetentionPeriod")
	if !ok {
		return rdssvc.DocDBClusterInput{}, false
	}
	encrypted, ok := docdbBoolParam(w, r, "StorageEncrypted")
	if !ok {
		return rdssvc.DocDBClusterInput{}, false
	}
	protect, ok := docdbBoolParam(w, r, "DeletionProtection")
	if !ok {
		return rdssvc.DocDBClusterInput{}, false
	}
	return rdssvc.DocDBClusterInput{
		Identifier:                 docdbFormString(r, "DBClusterIdentifier"),
		Engine:                     docdbFormString(r, "Engine"),
		EngineVersion:              docdbFormString(r, "EngineVersion"),
		MasterUsername:             docdbFormString(r, "MasterUsername"),
		MasterUserPassword:         r.Form.Get("MasterUserPassword"),
		Port:                       port,
		DBSubnetGroupName:          docdbFormString(r, "DBSubnetGroupName"),
		DBClusterParameterGroup:    docdbFormString(r, "DBClusterParameterGroupName"),
		VpcSecurityGroupIDs:        docdbFormList(r, "VpcSecurityGroupIds.VpcSecurityGroupId"),
		AvailabilityZones:          docdbFormList(r, "AvailabilityZones.AvailabilityZone"),
		BackupRetentionPeriod:      retention,
		PreferredBackupWindow:      docdbFormString(r, "PreferredBackupWindow"),
		PreferredMaintenanceWindow: docdbFormString(r, "PreferredMaintenanceWindow"),
		StorageEncrypted:           encrypted != nil && *encrypted,
		KmsKeyID:                   docdbFormString(r, "KmsKeyId"),
		DeletionProtection:         protect != nil && *protect,
		LogExports:                 docdbFormList(r, "EnableCloudwatchLogsExports.member"),
		StorageType:                docdbFormString(r, "StorageType"),
		GlobalClusterID:            docdbFormString(r, "GlobalClusterIdentifier"),
	}, true
}

func (s *Server) respondDocDBCluster(w http.ResponseWriter, action string, c rdssvc.DBCluster) {
	respondRDSXML(w, action, struct {
		XMLName   xml.Name        `xml:""`
		DBCluster docdbClusterXML `xml:"DBCluster"`
	}{XMLName: xml.Name{Local: action + "Result"}, DBCluster: docdbClusterToXML(c)})
}

// docdbCreateTags reads and validates create-time Tags before anything is
// created, so a bad tag can't leave a half-made resource behind.
func docdbCreateTags(w http.ResponseWriter, r *http.Request, action string) (map[string]string, bool) {
	tags := parseRDSTagMembers(r.Form, "Tags.Tag")
	if err := rdssvc.ValidateDocDBTags(tags); err != nil {
		respondDocDBError(w, action, err)
		return nil, false
	}
	return tags, true
}

// docdbApplyTags tags a resource that was just created; validation already
// passed, so a failure here is a store invariant breaking.
func (s *Server) docdbApplyTags(w http.ResponseWriter, action, arn string, tags map[string]string) bool {
	if len(tags) == 0 {
		return true
	}
	if err := s.rds.DocDBTagResource(arn, tags); err != nil {
		respondDocDBError(w, action, err)
		return false
	}
	return true
}

func (s *Server) handleDocDBCreateDBCluster(w http.ResponseWriter, r *http.Request) {
	tags, ok := docdbCreateTags(w, r, "CreateDBCluster")
	if !ok {
		return
	}
	if !docdbRequire(w, r, "DBClusterIdentifier", "Engine") {
		return
	}
	in, ok := docdbClusterInput(w, r)
	if !ok {
		return
	}
	cluster, err := s.rds.CreateDocDBCluster(in)
	if err != nil {
		respondDocDBError(w, "CreateDBCluster", err)
		return
	}
	if !s.docdbApplyTags(w, "CreateDBCluster", cluster.ARN, tags) {
		return
	}
	s.respondDocDBCluster(w, "CreateDBCluster", cluster)
}

func (s *Server) handleDocDBDescribeDBClusters(w http.ResponseWriter, r *http.Request) {
	marker, max, ok := docdbPagingParams(w, r)
	if !ok {
		return
	}
	filters := docdbFilters(r)
	if !docdbCheckFilters(w, filters, "db-cluster-id", "engine") {
		return
	}
	clusters, next, err := s.rds.DescribeDocDBClusters(rdssvc.DocDBClusterFilter{
		Identifier: docdbFormString(r, "DBClusterIdentifier"),
		ClusterIDs: filters["db-cluster-id"], Engines: filters["engine"],
	}, marker, max)
	if err != nil {
		respondDocDBError(w, "DescribeDBClusters", err)
		return
	}
	out := make([]docdbClusterXML, 0, len(clusters))
	for _, c := range clusters {
		out = append(out, docdbClusterToXML(c))
	}
	respondRDSXML(w, "DescribeDBClusters", struct {
		XMLName    xml.Name          `xml:"DescribeDBClustersResult"`
		Marker     string            `xml:"Marker,omitempty"`
		DBClusters []docdbClusterXML `xml:"DBClusters>DBCluster"`
	}{Marker: next, DBClusters: out})
}

func (s *Server) handleDocDBModifyDBCluster(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBClusterIdentifier") {
		return
	}
	port, ok := docdbIntParam(w, r, "Port")
	if !ok {
		return
	}
	retention, ok := docdbIntParam(w, r, "BackupRetentionPeriod")
	if !ok {
		return
	}
	protect, ok := docdbBoolParam(w, r, "DeletionProtection")
	if !ok {
		return
	}
	major, ok := docdbBoolParam(w, r, "AllowMajorVersionUpgrade")
	if !ok {
		return
	}
	var sgs []string
	if list := docdbFormList(r, "VpcSecurityGroupIds.VpcSecurityGroupId"); len(list) > 0 {
		sgs = list
	}
	cluster, err := s.rds.ModifyDocDBCluster(rdssvc.DocDBModifyClusterInput{
		Identifier:                 docdbFormString(r, "DBClusterIdentifier"),
		NewIdentifier:              docdbFormString(r, "NewDBClusterIdentifier"),
		BackupRetentionPeriod:      retention,
		DBClusterParameterGroup:    docdbFormString(r, "DBClusterParameterGroupName"),
		VpcSecurityGroupIDs:        sgs,
		Port:                       port,
		MasterUserPassword:         r.Form.Get("MasterUserPassword"),
		PreferredBackupWindow:      docdbFormString(r, "PreferredBackupWindow"),
		PreferredMaintenanceWindow: docdbFormString(r, "PreferredMaintenanceWindow"),
		EnableLogs:                 docdbFormList(r, "CloudwatchLogsExportConfiguration.EnableLogTypes.member"),
		DisableLogs:                docdbFormList(r, "CloudwatchLogsExportConfiguration.DisableLogTypes.member"),
		EngineVersion:              docdbFormString(r, "EngineVersion"),
		AllowMajorVersionUpgrade:   major != nil && *major,
		DeletionProtection:         protect,
		StorageType:                docdbFormString(r, "StorageType"),
	})
	if err != nil {
		respondDocDBError(w, "ModifyDBCluster", err)
		return
	}
	s.respondDocDBCluster(w, "ModifyDBCluster", cluster)
}

func (s *Server) handleDocDBDeleteDBCluster(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBClusterIdentifier") {
		return
	}
	skip, ok := docdbBoolParam(w, r, "SkipFinalSnapshot")
	if !ok {
		return
	}
	cluster, err := s.rds.DeleteDocDBCluster(docdbFormString(r, "DBClusterIdentifier"), skip != nil && *skip, docdbFormString(r, "FinalDBSnapshotIdentifier"))
	if err != nil {
		respondDocDBError(w, "DeleteDBCluster", err)
		return
	}
	s.respondDocDBCluster(w, "DeleteDBCluster", cluster)
}

func (s *Server) handleDocDBStartDBCluster(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBClusterIdentifier") {
		return
	}
	cluster, err := s.rds.StartDocDBCluster(docdbFormString(r, "DBClusterIdentifier"))
	if err != nil {
		respondDocDBError(w, "StartDBCluster", err)
		return
	}
	s.respondDocDBCluster(w, "StartDBCluster", cluster)
}

func (s *Server) handleDocDBStopDBCluster(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBClusterIdentifier") {
		return
	}
	cluster, err := s.rds.StopDocDBCluster(docdbFormString(r, "DBClusterIdentifier"))
	if err != nil {
		respondDocDBError(w, "StopDBCluster", err)
		return
	}
	s.respondDocDBCluster(w, "StopDBCluster", cluster)
}

func (s *Server) handleDocDBFailoverDBCluster(w http.ResponseWriter, r *http.Request) {
	// DBClusterIdentifier is optional in the model but every real call names one.
	if !docdbRequire(w, r, "DBClusterIdentifier") {
		return
	}
	cluster, err := s.rds.FailoverDocDBCluster(docdbFormString(r, "DBClusterIdentifier"), docdbFormString(r, "TargetDBInstanceIdentifier"))
	if err != nil {
		respondDocDBError(w, "FailoverDBCluster", err)
		return
	}
	s.respondDocDBCluster(w, "FailoverDBCluster", cluster)
}

// ---- instances ----

func docdbInstanceInput(w http.ResponseWriter, r *http.Request) (rdssvc.DocDBInstanceInput, bool) {
	var tier *int
	if raw := docdbFormString(r, "PromotionTier"); raw != "" {
		n, ok := docdbIntParam(w, r, "PromotionTier")
		if !ok {
			return rdssvc.DocDBInstanceInput{}, false
		}
		tier = &n
	}
	autoMinor, ok := docdbBoolParam(w, r, "AutoMinorVersionUpgrade")
	if !ok {
		return rdssvc.DocDBInstanceInput{}, false
	}
	return rdssvc.DocDBInstanceInput{
		Identifier:              docdbFormString(r, "DBInstanceIdentifier"),
		ClusterID:               docdbFormString(r, "DBClusterIdentifier"),
		Engine:                  docdbFormString(r, "Engine"),
		DBInstanceClass:         docdbFormString(r, "DBInstanceClass"),
		AvailabilityZone:        docdbFormString(r, "AvailabilityZone"),
		PromotionTier:           tier,
		AutoMinorVersionUpgrade: autoMinor,
		MaintenanceWindow:       docdbFormString(r, "PreferredMaintenanceWindow"),
		CACertificateIdentifier: docdbFormString(r, "CACertificateIdentifier"),
	}, true
}

func (s *Server) respondDocDBInstance(w http.ResponseWriter, action string, i rdssvc.DBInstance) {
	respondRDSXML(w, action, struct {
		XMLName    xml.Name         `xml:""`
		DBInstance docdbInstanceXML `xml:"DBInstance"`
	}{XMLName: xml.Name{Local: action + "Result"}, DBInstance: s.docdbInstanceToXML(i)})
}

func (s *Server) handleDocDBCreateDBInstance(w http.ResponseWriter, r *http.Request) {
	tags, ok := docdbCreateTags(w, r, "CreateDBInstance")
	if !ok {
		return
	}
	if !docdbRequire(w, r, "DBInstanceIdentifier", "DBInstanceClass", "Engine", "DBClusterIdentifier") {
		return
	}
	in, ok := docdbInstanceInput(w, r)
	if !ok {
		return
	}
	inst, err := s.rds.CreateDocDBInstance(in)
	if err != nil {
		respondDocDBError(w, "CreateDBInstance", err)
		return
	}
	if !s.docdbApplyTags(w, "CreateDBInstance", inst.ARN, tags) {
		return
	}
	s.respondDocDBInstance(w, "CreateDBInstance", inst)
}

func (s *Server) handleDocDBDescribeDBInstances(w http.ResponseWriter, r *http.Request) {
	marker, max, ok := docdbPagingParams(w, r)
	if !ok {
		return
	}
	filters := docdbFilters(r)
	if !docdbCheckFilters(w, filters, "db-cluster-id", "db-instance-id", "engine") {
		return
	}
	instances, next, err := s.rds.DescribeDocDBInstances(rdssvc.DocDBInstanceFilter{
		Identifier: docdbFormString(r, "DBInstanceIdentifier"),
		ClusterIDs: filters["db-cluster-id"], InstanceIDs: filters["db-instance-id"], Engines: filters["engine"],
	}, marker, max)
	if err != nil {
		respondDocDBError(w, "DescribeDBInstances", err)
		return
	}
	out := make([]docdbInstanceXML, 0, len(instances))
	for _, i := range instances {
		out = append(out, s.docdbInstanceToXML(i))
	}
	respondRDSXML(w, "DescribeDBInstances", struct {
		XMLName     xml.Name           `xml:"DescribeDBInstancesResult"`
		Marker      string             `xml:"Marker,omitempty"`
		DBInstances []docdbInstanceXML `xml:"DBInstances>DBInstance"`
	}{Marker: next, DBInstances: out})
}

func (s *Server) handleDocDBModifyDBInstance(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBInstanceIdentifier") {
		return
	}
	in, ok := docdbInstanceInput(w, r)
	if !ok {
		return
	}
	inst, err := s.rds.ModifyDocDBInstance(rdssvc.DocDBModifyInstanceInput{DocDBInstanceInput: in, NewIdentifier: docdbFormString(r, "NewDBInstanceIdentifier")})
	if err != nil {
		respondDocDBError(w, "ModifyDBInstance", err)
		return
	}
	s.respondDocDBInstance(w, "ModifyDBInstance", inst)
}

func (s *Server) handleDocDBRebootDBInstance(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBInstanceIdentifier") {
		return
	}
	force, ok := docdbBoolParam(w, r, "ForceFailover")
	if !ok {
		return
	}
	inst, err := s.rds.RebootDocDBInstance(docdbFormString(r, "DBInstanceIdentifier"), force != nil && *force)
	if err != nil {
		respondDocDBError(w, "RebootDBInstance", err)
		return
	}
	s.respondDocDBInstance(w, "RebootDBInstance", inst)
}

func (s *Server) handleDocDBDeleteDBInstance(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBInstanceIdentifier") {
		return
	}
	inst, err := s.rds.DeleteDocDBInstance(docdbFormString(r, "DBInstanceIdentifier"))
	if err != nil {
		respondDocDBError(w, "DeleteDBInstance", err)
		return
	}
	s.respondDocDBInstance(w, "DeleteDBInstance", inst)
}

// ---- cluster parameter groups ----

func (s *Server) respondDocDBParamGroup(w http.ResponseWriter, action string, g rdssvc.DBClusterParameterGroup) {
	respondRDSXML(w, action, struct {
		XMLName xml.Name           `xml:""`
		Group   docdbParamGroupXML `xml:"DBClusterParameterGroup"`
	}{XMLName: xml.Name{Local: action + "Result"}, Group: docdbParamGroupToXML(g)})
}

func (s *Server) handleDocDBCreateDBClusterParameterGroup(w http.ResponseWriter, r *http.Request) {
	tags, ok := docdbCreateTags(w, r, "CreateDBClusterParameterGroup")
	if !ok {
		return
	}
	if !docdbRequire(w, r, "DBClusterParameterGroupName", "DBParameterGroupFamily", "Description") {
		return
	}
	group, err := s.rds.CreateClusterParameterGroup(docdbFormString(r, "DBClusterParameterGroupName"), docdbFormString(r, "DBParameterGroupFamily"), docdbFormString(r, "Description"))
	if err != nil {
		respondDocDBError(w, "CreateDBClusterParameterGroup", err)
		return
	}
	if !s.docdbApplyTags(w, "CreateDBClusterParameterGroup", group.ARN, tags) {
		return
	}
	s.respondDocDBParamGroup(w, "CreateDBClusterParameterGroup", group)
}

func (s *Server) handleDocDBCopyDBClusterParameterGroup(w http.ResponseWriter, r *http.Request) {
	tags, ok := docdbCreateTags(w, r, "CopyDBClusterParameterGroup")
	if !ok {
		return
	}
	if !docdbRequire(w, r, "SourceDBClusterParameterGroupIdentifier", "TargetDBClusterParameterGroupIdentifier", "TargetDBClusterParameterGroupDescription") {
		return
	}
	group, err := s.rds.CopyClusterParameterGroup(docdbFormString(r, "SourceDBClusterParameterGroupIdentifier"),
		docdbFormString(r, "TargetDBClusterParameterGroupIdentifier"), docdbFormString(r, "TargetDBClusterParameterGroupDescription"))
	if err != nil {
		respondDocDBError(w, "CopyDBClusterParameterGroup", err)
		return
	}
	if !s.docdbApplyTags(w, "CopyDBClusterParameterGroup", group.ARN, tags) {
		return
	}
	s.respondDocDBParamGroup(w, "CopyDBClusterParameterGroup", group)
}

func (s *Server) handleDocDBDeleteDBClusterParameterGroup(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBClusterParameterGroupName") {
		return
	}
	if err := s.rds.DeleteClusterParameterGroup(docdbFormString(r, "DBClusterParameterGroupName")); err != nil {
		respondDocDBError(w, "DeleteDBClusterParameterGroup", err)
		return
	}
	respondRDSXML(w, "DeleteDBClusterParameterGroup", struct {
		XMLName xml.Name `xml:"DeleteDBClusterParameterGroupResult"`
	}{})
}

func (s *Server) handleDocDBDescribeDBClusterParameterGroups(w http.ResponseWriter, r *http.Request) {
	marker, max, ok := docdbPagingParams(w, r)
	if !ok {
		return
	}
	groups, next, err := s.rds.DescribeClusterParameterGroups(docdbFormString(r, "DBClusterParameterGroupName"), marker, max)
	if err != nil {
		respondDocDBError(w, "DescribeDBClusterParameterGroups", err)
		return
	}
	out := make([]docdbParamGroupXML, 0, len(groups))
	for _, g := range groups {
		out = append(out, docdbParamGroupToXML(g))
	}
	respondRDSXML(w, "DescribeDBClusterParameterGroups", struct {
		XMLName xml.Name             `xml:"DescribeDBClusterParameterGroupsResult"`
		Marker  string               `xml:"Marker,omitempty"`
		Groups  []docdbParamGroupXML `xml:"DBClusterParameterGroups>DBClusterParameterGroup"`
	}{Marker: next, Groups: out})
}

func (s *Server) handleDocDBDescribeDBClusterParameters(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBClusterParameterGroupName") {
		return
	}
	marker, max, ok := docdbPagingParams(w, r)
	if !ok {
		return
	}
	params, next, err := s.rds.ClusterParameters(docdbFormString(r, "DBClusterParameterGroupName"), docdbFormString(r, "Source"), marker, max)
	if err != nil {
		respondDocDBError(w, "DescribeDBClusterParameters", err)
		return
	}
	respondRDSXML(w, "DescribeDBClusterParameters", struct {
		XMLName    xml.Name            `xml:"DescribeDBClusterParametersResult"`
		Parameters []docdbParameterXML `xml:"Parameters>Parameter"`
		Marker     string              `xml:"Marker,omitempty"`
	}{Parameters: docdbParametersToXML(params), Marker: next})
}

func respondDocDBParamGroupName(w http.ResponseWriter, action, name string) {
	respondRDSXML(w, action, struct {
		XMLName xml.Name `xml:""`
		Name    string   `xml:"DBClusterParameterGroupName"`
	}{XMLName: xml.Name{Local: action + "Result"}, Name: name})
}

func (s *Server) handleDocDBModifyDBClusterParameterGroup(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBClusterParameterGroupName") {
		return
	}
	var params []rdssvc.ClusterParameter
	for _, p := range parseRDSParameterMembers(r.Form, "Parameters.Parameter") {
		params = append(params, rdssvc.ClusterParameter{Name: p.Name, Value: p.Value, ApplyMethod: firstNonEmpty(p.ApplyMethod, "pending-reboot")})
	}
	name, err := s.rds.ModifyClusterParameterGroup(docdbFormString(r, "DBClusterParameterGroupName"), params)
	if err != nil {
		respondDocDBError(w, "ModifyDBClusterParameterGroup", err)
		return
	}
	respondDocDBParamGroupName(w, "ModifyDBClusterParameterGroup", name)
}

func (s *Server) handleDocDBResetDBClusterParameterGroup(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBClusterParameterGroupName") {
		return
	}
	all, ok := docdbBoolParam(w, r, "ResetAllParameters")
	if !ok {
		return
	}
	name, err := s.rds.ResetClusterParameterGroup(docdbFormString(r, "DBClusterParameterGroupName"), all != nil && *all,
		parseRDSParameterNames(r.Form, "Parameters.Parameter"))
	if err != nil {
		respondDocDBError(w, "ResetDBClusterParameterGroup", err)
		return
	}
	respondDocDBParamGroupName(w, "ResetDBClusterParameterGroup", name)
}

func (s *Server) handleDocDBDescribeEngineDefaultClusterParameters(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBParameterGroupFamily") {
		return
	}
	family := docdbFormString(r, "DBParameterGroupFamily")
	known := false
	for _, v := range rdssvc.DocDBEngineVersions {
		known = known || v.Family == family
	}
	if !known {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "DBParameterGroupFamily "+family+" is not a supported family")
		return
	}
	respondRDSXML(w, "DescribeEngineDefaultClusterParameters", struct {
		XMLName  xml.Name `xml:"DescribeEngineDefaultClusterParametersResult"`
		Defaults struct {
			Family     string              `xml:"DBParameterGroupFamily"`
			Parameters []docdbParameterXML `xml:"Parameters>Parameter"`
		} `xml:"EngineDefaults"`
	}{Defaults: struct {
		Family     string              `xml:"DBParameterGroupFamily"`
		Parameters []docdbParameterXML `xml:"Parameters>Parameter"`
	}{Family: family, Parameters: docdbParametersToXML(rdssvc.DocDBDefaultClusterParameters())}})
}

// ---- cluster snapshots ----

func (s *Server) respondDocDBSnapshot(w http.ResponseWriter, action string, snap rdssvc.DBClusterSnapshot) {
	respondRDSXML(w, action, struct {
		XMLName  xml.Name         `xml:""`
		Snapshot docdbSnapshotXML `xml:"DBClusterSnapshot"`
	}{XMLName: xml.Name{Local: action + "Result"}, Snapshot: docdbSnapshotToXML(snap)})
}

func (s *Server) handleDocDBCreateDBClusterSnapshot(w http.ResponseWriter, r *http.Request) {
	tags, ok := docdbCreateTags(w, r, "CreateDBClusterSnapshot")
	if !ok {
		return
	}
	if !docdbRequire(w, r, "DBClusterSnapshotIdentifier", "DBClusterIdentifier") {
		return
	}
	snap, err := s.rds.CreateClusterSnapshot(docdbFormString(r, "DBClusterSnapshotIdentifier"), docdbFormString(r, "DBClusterIdentifier"))
	if err != nil {
		respondDocDBError(w, "CreateDBClusterSnapshot", err)
		return
	}
	if !s.docdbApplyTags(w, "CreateDBClusterSnapshot", snap.ARN, tags) {
		return
	}
	s.respondDocDBSnapshot(w, "CreateDBClusterSnapshot", snap)
}

func (s *Server) handleDocDBCopyDBClusterSnapshot(w http.ResponseWriter, r *http.Request) {
	tags, ok := docdbCreateTags(w, r, "CopyDBClusterSnapshot")
	if !ok {
		return
	}
	if !docdbRequire(w, r, "SourceDBClusterSnapshotIdentifier", "TargetDBClusterSnapshotIdentifier") {
		return
	}
	copyTags, ok := docdbBoolParam(w, r, "CopyTags")
	if !ok {
		return
	}
	snap, err := s.rds.CopyClusterSnapshot(docdbFormString(r, "SourceDBClusterSnapshotIdentifier"), docdbFormString(r, "TargetDBClusterSnapshotIdentifier"), docdbFormString(r, "KmsKeyId"))
	if err != nil {
		respondDocDBError(w, "CopyDBClusterSnapshot", err)
		return
	}
	if copyTags != nil && *copyTags {
		if tags, err := s.rds.DocDBTags(snap.SourceARN); err == nil && len(tags) > 0 {
			if err := s.rds.DocDBTagResource(snap.ARN, tags); err != nil {
				respondDocDBError(w, "CopyDBClusterSnapshot", err)
				return
			}
		}
	}
	if !s.docdbApplyTags(w, "CopyDBClusterSnapshot", snap.ARN, tags) {
		return
	}
	s.respondDocDBSnapshot(w, "CopyDBClusterSnapshot", snap)
}

func (s *Server) handleDocDBDeleteDBClusterSnapshot(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBClusterSnapshotIdentifier") {
		return
	}
	snap, err := s.rds.DeleteClusterSnapshot(docdbFormString(r, "DBClusterSnapshotIdentifier"))
	if err != nil {
		respondDocDBError(w, "DeleteDBClusterSnapshot", err)
		return
	}
	s.respondDocDBSnapshot(w, "DeleteDBClusterSnapshot", snap)
}

func (s *Server) handleDocDBDescribeDBClusterSnapshots(w http.ResponseWriter, r *http.Request) {
	marker, max, ok := docdbPagingParams(w, r)
	if !ok {
		return
	}
	shared, ok := docdbBoolParam(w, r, "IncludeShared")
	if !ok {
		return
	}
	public, ok := docdbBoolParam(w, r, "IncludePublic")
	if !ok {
		return
	}
	snaps, next, err := s.rds.DescribeClusterSnapshots(rdssvc.ClusterSnapshotFilter{
		Identifier: docdbFormString(r, "DBClusterSnapshotIdentifier"), ClusterID: docdbFormString(r, "DBClusterIdentifier"),
		SnapshotType: docdbFormString(r, "SnapshotType"), IncludeShared: shared != nil && *shared, IncludePublic: public != nil && *public,
	}, marker, max)
	if err != nil {
		respondDocDBError(w, "DescribeDBClusterSnapshots", err)
		return
	}
	out := make([]docdbSnapshotXML, 0, len(snaps))
	for _, snap := range snaps {
		out = append(out, docdbSnapshotToXML(snap))
	}
	respondRDSXML(w, "DescribeDBClusterSnapshots", struct {
		XMLName   xml.Name           `xml:"DescribeDBClusterSnapshotsResult"`
		Marker    string             `xml:"Marker,omitempty"`
		Snapshots []docdbSnapshotXML `xml:"DBClusterSnapshots>DBClusterSnapshot"`
	}{Marker: next, Snapshots: out})
}

type docdbSnapshotAttributesXML struct {
	DBClusterSnapshotIdentifier string `xml:"DBClusterSnapshotIdentifier"`
	Attributes                  []struct {
		AttributeName   string   `xml:"AttributeName"`
		AttributeValues []string `xml:"AttributeValues>AttributeValue"`
	} `xml:"DBClusterSnapshotAttributes>DBClusterSnapshotAttribute"`
}

func respondDocDBSnapshotAttributes(w http.ResponseWriter, action string, snap rdssvc.DBClusterSnapshot) {
	attrs := docdbSnapshotAttributesXML{DBClusterSnapshotIdentifier: snap.Identifier}
	attrs.Attributes = append(attrs.Attributes, struct {
		AttributeName   string   `xml:"AttributeName"`
		AttributeValues []string `xml:"AttributeValues>AttributeValue"`
	}{AttributeName: "restore", AttributeValues: snap.RestoreAccounts})
	respondRDSXML(w, action, struct {
		XMLName xml.Name                   `xml:""`
		Result  docdbSnapshotAttributesXML `xml:"DBClusterSnapshotAttributesResult"`
	}{XMLName: xml.Name{Local: action + "Result"}, Result: attrs})
}

func (s *Server) handleDocDBDescribeDBClusterSnapshotAttributes(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBClusterSnapshotIdentifier") {
		return
	}
	snap, err := s.rds.ClusterSnapshotAttributes(docdbFormString(r, "DBClusterSnapshotIdentifier"))
	if err != nil {
		respondDocDBError(w, "DescribeDBClusterSnapshotAttributes", err)
		return
	}
	respondDocDBSnapshotAttributes(w, "DescribeDBClusterSnapshotAttributes", snap)
}

func (s *Server) handleDocDBModifyDBClusterSnapshotAttribute(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBClusterSnapshotIdentifier", "AttributeName") {
		return
	}
	snap, err := s.rds.ModifyClusterSnapshotAttribute(docdbFormString(r, "DBClusterSnapshotIdentifier"), docdbFormString(r, "AttributeName"),
		docdbFormList(r, "ValuesToAdd.AttributeValue"), docdbFormList(r, "ValuesToRemove.AttributeValue"))
	if err != nil {
		respondDocDBError(w, "ModifyDBClusterSnapshotAttribute", err)
		return
	}
	respondDocDBSnapshotAttributes(w, "ModifyDBClusterSnapshotAttribute", snap)
}

func (s *Server) handleDocDBRestoreDBClusterFromSnapshot(w http.ResponseWriter, r *http.Request) {
	tags, ok := docdbCreateTags(w, r, "RestoreDBClusterFromSnapshot")
	if !ok {
		return
	}
	if !docdbRequire(w, r, "DBClusterIdentifier", "SnapshotIdentifier", "Engine") {
		return
	}
	in, ok := docdbClusterInput(w, r)
	if !ok {
		return
	}
	cluster, err := s.rds.RestoreClusterFromSnapshot(docdbFormString(r, "SnapshotIdentifier"), in)
	if err != nil {
		respondDocDBError(w, "RestoreDBClusterFromSnapshot", err)
		return
	}
	if !s.docdbApplyTags(w, "RestoreDBClusterFromSnapshot", cluster.ARN, tags) {
		return
	}
	s.respondDocDBCluster(w, "RestoreDBClusterFromSnapshot", cluster)
}

func (s *Server) handleDocDBRestoreDBClusterToPointInTime(w http.ResponseWriter, r *http.Request) {
	tags, ok := docdbCreateTags(w, r, "RestoreDBClusterToPointInTime")
	if !ok {
		return
	}
	if !docdbRequire(w, r, "DBClusterIdentifier", "SourceDBClusterIdentifier") {
		return
	}
	in, ok := docdbClusterInput(w, r)
	if !ok {
		return
	}
	latest, ok := docdbBoolParam(w, r, "UseLatestRestorableTime")
	if !ok {
		return
	}
	var restoreTime time.Time
	if raw := docdbFormString(r, "RestoreToTime"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "RestoreToTime must be an ISO 8601 timestamp")
			return
		}
		restoreTime = t
	}
	cluster, err := s.rds.RestoreClusterToPointInTime(docdbFormString(r, "SourceDBClusterIdentifier"), restoreTime,
		latest != nil && *latest, docdbFormString(r, "RestoreType"), in)
	if err != nil {
		respondDocDBError(w, "RestoreDBClusterToPointInTime", err)
		return
	}
	if !s.docdbApplyTags(w, "RestoreDBClusterToPointInTime", cluster.ARN, tags) {
		return
	}
	s.respondDocDBCluster(w, "RestoreDBClusterToPointInTime", cluster)
}

// ---- global clusters ----

func respondDocDBGlobal(w http.ResponseWriter, action string, g rdssvc.GlobalCluster) {
	respondRDSXML(w, action, struct {
		XMLName xml.Name       `xml:""`
		Global  docdbGlobalXML `xml:"GlobalCluster"`
	}{XMLName: xml.Name{Local: action + "Result"}, Global: docdbGlobalToXML(g)})
}

func (s *Server) handleDocDBCreateGlobalCluster(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "GlobalClusterIdentifier") {
		return
	}
	encrypted, ok := docdbBoolParam(w, r, "StorageEncrypted")
	if !ok {
		return
	}
	protect, ok := docdbBoolParam(w, r, "DeletionProtection")
	if !ok {
		return
	}
	if engine := docdbFormString(r, "Engine"); engine != "" && engine != rdssvc.DocDBEngine {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Engine must be docdb")
		return
	}
	g, err := s.rds.CreateDocDBGlobalCluster(rdssvc.DocDBGlobalInput{
		Identifier: docdbFormString(r, "GlobalClusterIdentifier"), SourceClusterID: docdbFormString(r, "SourceDBClusterIdentifier"),
		EngineVersion: docdbFormString(r, "EngineVersion"), DatabaseName: docdbFormString(r, "DatabaseName"),
		StorageEncrypted: encrypted, DeletionProtection: protect != nil && *protect,
	})
	if err != nil {
		respondDocDBError(w, "CreateGlobalCluster", err)
		return
	}
	respondDocDBGlobal(w, "CreateGlobalCluster", g)
}

func (s *Server) handleDocDBDescribeGlobalClusters(w http.ResponseWriter, r *http.Request) {
	marker, max, ok := docdbPagingParams(w, r)
	if !ok {
		return
	}
	globals, next, err := s.rds.DescribeDocDBGlobalClusters(docdbFormString(r, "GlobalClusterIdentifier"), marker, max)
	if err != nil {
		respondDocDBError(w, "DescribeGlobalClusters", err)
		return
	}
	out := make([]docdbGlobalXML, 0, len(globals))
	for _, g := range globals {
		out = append(out, docdbGlobalToXML(g))
	}
	// The model's list member is named GlobalClusterMember, not GlobalCluster.
	respondRDSXML(w, "DescribeGlobalClusters", struct {
		XMLName xml.Name         `xml:"DescribeGlobalClustersResult"`
		Marker  string           `xml:"Marker,omitempty"`
		Globals []docdbGlobalXML `xml:"GlobalClusters>GlobalClusterMember"`
	}{Marker: next, Globals: out})
}

func (s *Server) handleDocDBModifyGlobalCluster(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "GlobalClusterIdentifier") {
		return
	}
	protect, ok := docdbBoolParam(w, r, "DeletionProtection")
	if !ok {
		return
	}
	g, err := s.rds.ModifyDocDBGlobalCluster(docdbFormString(r, "GlobalClusterIdentifier"), docdbFormString(r, "NewGlobalClusterIdentifier"), protect)
	if err != nil {
		respondDocDBError(w, "ModifyGlobalCluster", err)
		return
	}
	respondDocDBGlobal(w, "ModifyGlobalCluster", g)
}

func (s *Server) handleDocDBDeleteGlobalCluster(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "GlobalClusterIdentifier") {
		return
	}
	g, err := s.rds.DeleteDocDBGlobalCluster(docdbFormString(r, "GlobalClusterIdentifier"))
	if err != nil {
		respondDocDBError(w, "DeleteGlobalCluster", err)
		return
	}
	respondDocDBGlobal(w, "DeleteGlobalCluster", g)
}

func (s *Server) handleDocDBRemoveFromGlobalCluster(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "GlobalClusterIdentifier", "DbClusterIdentifier") {
		return
	}
	g, err := s.rds.RemoveFromDocDBGlobalCluster(docdbFormString(r, "GlobalClusterIdentifier"), docdbFormString(r, "DbClusterIdentifier"))
	if err != nil {
		respondDocDBError(w, "RemoveFromGlobalCluster", err)
		return
	}
	respondDocDBGlobal(w, "RemoveFromGlobalCluster", g)
}

func (s *Server) docdbGlobalFailover(w http.ResponseWriter, r *http.Request, action string, switchover bool) {
	if !docdbRequire(w, r, "GlobalClusterIdentifier", "TargetDbClusterIdentifier") {
		return
	}
	g, err := s.rds.FailoverDocDBGlobalCluster(docdbFormString(r, "GlobalClusterIdentifier"), docdbFormString(r, "TargetDbClusterIdentifier"), switchover)
	if err != nil {
		respondDocDBError(w, action, err)
		return
	}
	respondDocDBGlobal(w, action, g)
}

func (s *Server) handleDocDBFailoverGlobalCluster(w http.ResponseWriter, r *http.Request) {
	s.docdbGlobalFailover(w, r, "FailoverGlobalCluster", false)
}

func (s *Server) handleDocDBSwitchoverGlobalCluster(w http.ResponseWriter, r *http.Request) {
	s.docdbGlobalFailover(w, r, "SwitchoverGlobalCluster", true)
}

// ---- event subscriptions, tags, subnet groups ----

func (s *Server) docdbSubscriptionSource(w http.ResponseWriter, r *http.Request, action string, add bool) {
	if !docdbRequire(w, r, "SubscriptionName", "SourceIdentifier") {
		return
	}
	sub, err := s.rds.UpdateSubscriptionSource(docdbFormString(r, "SubscriptionName"), docdbFormString(r, "SourceIdentifier"), add)
	if err != nil {
		respondDocDBError(w, action, err)
		return
	}
	respondRDSXML(w, action, struct {
		XMLName      xml.Name                `xml:""`
		Subscription rdsEventSubscriptionXML `xml:"EventSubscription"`
	}{XMLName: xml.Name{Local: action + "Result"}, Subscription: rdsEventSubscriptionToXML(sub)})
}

func (s *Server) handleDocDBAddSourceIdentifierToSubscription(w http.ResponseWriter, r *http.Request) {
	s.docdbSubscriptionSource(w, r, "AddSourceIdentifierToSubscription", true)
}

func (s *Server) handleDocDBRemoveSourceIdentifierFromSubscription(w http.ResponseWriter, r *http.Request) {
	s.docdbSubscriptionSource(w, r, "RemoveSourceIdentifierFromSubscription", false)
}

func (s *Server) handleDocDBAddTagsToResource(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "ResourceName") {
		return
	}
	tags := parseRDSTagMembers(r.Form, "Tags.Tag")
	if len(tags) == 0 {
		respondRDSErrorXML(w, http.StatusBadRequest, "MissingParameter", "The request must contain the parameter Tags")
		return
	}
	if err := s.rds.DocDBTagResource(docdbFormString(r, "ResourceName"), tags); err != nil {
		respondDocDBError(w, "AddTagsToResource", err)
		return
	}
	respondRDSXML(w, "AddTagsToResource", struct {
		XMLName xml.Name `xml:"AddTagsToResourceResult"`
	}{})
}

func (s *Server) handleDocDBListTagsForResource(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "ResourceName") {
		return
	}
	tags, err := s.rds.DocDBTags(docdbFormString(r, "ResourceName"))
	if err != nil {
		respondDocDBError(w, "ListTagsForResource", err)
		return
	}
	respondRDSXML(w, "ListTagsForResource", struct {
		XMLName xml.Name    `xml:"ListTagsForResourceResult"`
		TagList []rdsTagXML `xml:"TagList>Tag"`
	}{TagList: docdbTagsToXML(tags)})
}

func (s *Server) handleDocDBRemoveTagsFromResource(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "ResourceName") {
		return
	}
	keys := docdbFormList(r, "TagKeys.member")
	if len(keys) == 0 {
		respondRDSErrorXML(w, http.StatusBadRequest, "MissingParameter", "The request must contain the parameter TagKeys")
		return
	}
	if err := s.rds.DocDBUntagResource(docdbFormString(r, "ResourceName"), keys); err != nil {
		respondDocDBError(w, "RemoveTagsFromResource", err)
		return
	}
	respondRDSXML(w, "RemoveTagsFromResource", struct {
		XMLName xml.Name `xml:"RemoveTagsFromResourceResult"`
	}{})
}

// handleDocDBDeleteDBSubnetGroup refuses to delete a subnet group a cluster
// still uses, then defers to the shared RDS handler.
func (s *Server) handleDocDBDeleteDBSubnetGroup(w http.ResponseWriter, r *http.Request) {
	if !docdbRequire(w, r, "DBSubnetGroupName") {
		return
	}
	name := docdbFormString(r, "DBSubnetGroupName")
	clusters, _, err := s.rds.DescribeDocDBClusters(rdssvc.DocDBClusterFilter{}, "", 0)
	if err == nil {
		for _, c := range clusters {
			if c.DBSubnetGroupName == name {
				respondRDSErrorXML(w, http.StatusBadRequest, "InvalidDBSubnetGroupStateFault", "DB subnet group "+name+" is in use by cluster "+c.Identifier)
				return
			}
		}
	}
	s.handleRDSDeleteDBSubnetGroup(w, r)
}
