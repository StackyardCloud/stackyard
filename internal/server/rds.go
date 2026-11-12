package server

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	rdssvc "github.com/stackyard/stackyard/internal/services/rds"
)

const rdsNamespace = "http://rds.amazonaws.com/doc/2014-10-31/"

var rdsQueryHandlers = func() map[string]func(*Server, http.ResponseWriter, *http.Request) {
	handlers := make(map[string]func(*Server, http.ResponseWriter, *http.Request), len(rdsOperations))
	for _, op := range rdsOperations {
		handlers[op.Name] = (*Server).handleRDSNotImplemented
	}

	// Stage 1
	handlers["CreateDBInstance"] = (*Server).handleRDSCreateDBInstance
	handlers["DescribeDBInstances"] = (*Server).handleRDSDescribeDBInstances
	handlers["ModifyDBInstance"] = (*Server).handleRDSModifyDBInstance
	handlers["DeleteDBInstance"] = (*Server).handleRDSDeleteDBInstance
	handlers["StartDBInstance"] = (*Server).handleRDSStartDBInstance
	handlers["StopDBInstance"] = (*Server).handleRDSStopDBInstance
	handlers["RebootDBInstance"] = (*Server).handleRDSRebootDBInstance

	// Stage 2
	handlers["CreateDBSnapshot"] = (*Server).handleRDSCreateDBSnapshot
	handlers["DescribeDBSnapshots"] = (*Server).handleRDSDescribeDBSnapshots
	handlers["DeleteDBSnapshot"] = (*Server).handleRDSDeleteDBSnapshot
	handlers["CopyDBSnapshot"] = (*Server).handleRDSCopyDBSnapshot
	handlers["RestoreDBInstanceFromDBSnapshot"] = (*Server).handleRDSRestoreDBInstanceFromDBSnapshot
	handlers["RestoreDBInstanceToPointInTime"] = (*Server).handleRDSRestoreDBInstanceToPointInTime
	handlers["StartExportTask"] = (*Server).handleRDSStartExportTask
	handlers["DescribeExportTasks"] = (*Server).handleRDSDescribeExportTasks
	handlers["CancelExportTask"] = (*Server).handleRDSCancelExportTask
	handlers["StartDBInstanceAutomatedBackupsReplication"] = (*Server).handleRDSStartDBInstanceAutomatedBackupsReplication
	handlers["StopDBInstanceAutomatedBackupsReplication"] = (*Server).handleRDSStopDBInstanceAutomatedBackupsReplication
	handlers["DescribeDBInstanceAutomatedBackups"] = (*Server).handleRDSDescribeDBInstanceAutomatedBackups
	handlers["DeleteDBInstanceAutomatedBackup"] = (*Server).handleRDSDeleteDBInstanceAutomatedBackup

	// Stage 3
	handlers["CreateDBParameterGroup"] = (*Server).handleRDSCreateDBParameterGroup
	handlers["DescribeDBParameterGroups"] = (*Server).handleRDSDescribeDBParameterGroups
	handlers["DescribeDBParameters"] = (*Server).handleRDSDescribeDBParameters
	handlers["ModifyDBParameterGroup"] = (*Server).handleRDSModifyDBParameterGroup
	handlers["ResetDBParameterGroup"] = (*Server).handleRDSResetDBParameterGroup
	handlers["DeleteDBParameterGroup"] = (*Server).handleRDSDeleteDBParameterGroup
	handlers["CreateOptionGroup"] = (*Server).handleRDSCreateOptionGroup
	handlers["DescribeOptionGroups"] = (*Server).handleRDSDescribeOptionGroups
	handlers["ModifyOptionGroup"] = (*Server).handleRDSModifyOptionGroup
	handlers["DeleteOptionGroup"] = (*Server).handleRDSDeleteOptionGroup
	handlers["CreateDBSubnetGroup"] = (*Server).handleRDSCreateDBSubnetGroup
	handlers["DescribeDBSubnetGroups"] = (*Server).handleRDSDescribeDBSubnetGroups
	handlers["ModifyDBSubnetGroup"] = (*Server).handleRDSModifyDBSubnetGroup
	handlers["DeleteDBSubnetGroup"] = (*Server).handleRDSDeleteDBSubnetGroup
	handlers["CreateDBSecurityGroup"] = (*Server).handleRDSCreateDBSecurityGroup
	handlers["DescribeDBSecurityGroups"] = (*Server).handleRDSDescribeDBSecurityGroups
	handlers["AuthorizeDBSecurityGroupIngress"] = (*Server).handleRDSAuthorizeDBSecurityGroupIngress
	handlers["RevokeDBSecurityGroupIngress"] = (*Server).handleRDSRevokeDBSecurityGroupIngress
	handlers["DeleteDBSecurityGroup"] = (*Server).handleRDSDeleteDBSecurityGroup
	handlers["DescribeCertificates"] = (*Server).handleRDSDescribeCertificates

	// Stage 4
	handlers["CreateDBCluster"] = (*Server).handleRDSCreateDBCluster
	handlers["DescribeDBClusters"] = (*Server).handleRDSDescribeDBClusters
	handlers["ModifyDBCluster"] = (*Server).handleRDSModifyDBCluster
	handlers["DeleteDBCluster"] = (*Server).handleRDSDeleteDBCluster
	handlers["StartDBCluster"] = (*Server).handleRDSStartDBCluster
	handlers["StopDBCluster"] = (*Server).handleRDSStopDBCluster
	handlers["RebootDBCluster"] = (*Server).handleRDSRebootDBCluster
	handlers["FailoverDBCluster"] = (*Server).handleRDSFailoverDBCluster
	handlers["CreateDBClusterEndpoint"] = (*Server).handleRDSCreateDBClusterEndpoint
	handlers["DescribeDBClusterEndpoints"] = (*Server).handleRDSDescribeDBClusterEndpoints
	handlers["ModifyDBClusterEndpoint"] = (*Server).handleRDSModifyDBClusterEndpoint
	handlers["DeleteDBClusterEndpoint"] = (*Server).handleRDSDeleteDBClusterEndpoint
	handlers["CreateGlobalCluster"] = (*Server).handleRDSCreateGlobalCluster
	handlers["DescribeGlobalClusters"] = (*Server).handleRDSDescribeGlobalClusters
	handlers["ModifyGlobalCluster"] = (*Server).handleRDSModifyGlobalCluster
	handlers["DeleteGlobalCluster"] = (*Server).handleRDSDeleteGlobalCluster
	handlers["FailoverGlobalCluster"] = (*Server).handleRDSFailoverGlobalCluster
	handlers["SwitchoverGlobalCluster"] = (*Server).handleRDSSwitchoverGlobalCluster

	// Stage 5
	handlers["CreateDBInstanceReadReplica"] = (*Server).handleRDSCreateDBInstanceReadReplica
	handlers["PromoteReadReplica"] = (*Server).handleRDSPromoteReadReplica
	handlers["SwitchoverReadReplica"] = (*Server).handleRDSSwitchoverReadReplica
	handlers["CreateBlueGreenDeployment"] = (*Server).handleRDSCreateBlueGreenDeployment
	handlers["DescribeBlueGreenDeployments"] = (*Server).handleRDSDescribeBlueGreenDeployments
	handlers["SwitchoverBlueGreenDeployment"] = (*Server).handleRDSSwitchoverBlueGreenDeployment
	handlers["DeleteBlueGreenDeployment"] = (*Server).handleRDSDeleteBlueGreenDeployment
	handlers["CreateTenantDatabase"] = (*Server).handleRDSCreateTenantDatabase
	handlers["DescribeTenantDatabases"] = (*Server).handleRDSDescribeTenantDatabases
	handlers["ModifyTenantDatabase"] = (*Server).handleRDSModifyTenantDatabase
	handlers["DeleteTenantDatabase"] = (*Server).handleRDSDeleteTenantDatabase

	// Stage 6
	handlers["AddTagsToResource"] = (*Server).handleRDSAddTagsToResource
	handlers["ListTagsForResource"] = (*Server).handleRDSListTagsForResource
	handlers["RemoveTagsFromResource"] = (*Server).handleRDSRemoveTagsFromResource
	handlers["CreateEventSubscription"] = (*Server).handleRDSCreateEventSubscription
	handlers["DescribeEventSubscriptions"] = (*Server).handleRDSDescribeEventSubscriptions
	handlers["ModifyEventSubscription"] = (*Server).handleRDSModifyEventSubscription
	handlers["DeleteEventSubscription"] = (*Server).handleRDSDeleteEventSubscription
	handlers["DescribePendingMaintenanceActions"] = (*Server).handleRDSDescribePendingMaintenanceActions
	handlers["ApplyPendingMaintenanceAction"] = (*Server).handleRDSApplyPendingMaintenanceAction
	handlers["DescribeEvents"] = (*Server).handleRDSDescribeEvents
	handlers["DescribeAccountAttributes"] = (*Server).handleRDSDescribeAccountAttributes
	handlers["DescribeDBEngineVersions"] = (*Server).handleRDSDescribeDBEngineVersions
	handlers["DescribeOrderableDBInstanceOptions"] = (*Server).handleRDSDescribeOrderableDBInstanceOptions
	handlers["DescribeSourceRegions"] = (*Server).handleRDSDescribeSourceRegions
	handlers["DescribeValidDBInstanceModifications"] = (*Server).handleRDSDescribeValidDBInstanceModifications

	// Stage 7
	handlers["AddRoleToDBInstance"] = (*Server).handleRDSAddRoleToDBInstance
	handlers["RemoveRoleFromDBInstance"] = (*Server).handleRDSRemoveRoleFromDBInstance
	handlers["AddRoleToDBCluster"] = (*Server).handleRDSAddRoleToDBCluster
	handlers["RemoveRoleFromDBCluster"] = (*Server).handleRDSRemoveRoleFromDBCluster
	handlers["StartActivityStream"] = (*Server).handleRDSStartActivityStream
	handlers["StopActivityStream"] = (*Server).handleRDSStopActivityStream
	handlers["CreateDBProxy"] = (*Server).handleRDSCreateDBProxy
	handlers["DescribeDBProxies"] = (*Server).handleRDSDescribeDBProxies
	handlers["ModifyDBProxy"] = (*Server).handleRDSModifyDBProxy
	handlers["DeleteDBProxy"] = (*Server).handleRDSDeleteDBProxy
	handlers["CreateDBProxyEndpoint"] = (*Server).handleRDSCreateDBProxyEndpoint
	handlers["DescribeDBProxyEndpoints"] = (*Server).handleRDSDescribeDBProxyEndpoints
	handlers["ModifyDBProxyEndpoint"] = (*Server).handleRDSModifyDBProxyEndpoint
	handlers["DeleteDBProxyEndpoint"] = (*Server).handleRDSDeleteDBProxyEndpoint
	handlers["RegisterDBProxyTargets"] = (*Server).handleRDSRegisterDBProxyTargets
	handlers["DeregisterDBProxyTargets"] = (*Server).handleRDSDeregisterDBProxyTargets
	handlers["DescribeDBProxyTargets"] = (*Server).handleRDSDescribeDBProxyTargets
	handlers["CreateIntegration"] = (*Server).handleRDSCreateIntegration
	handlers["DescribeIntegrations"] = (*Server).handleRDSDescribeIntegrations
	handlers["ModifyIntegration"] = (*Server).handleRDSModifyIntegration
	handlers["DeleteIntegration"] = (*Server).handleRDSDeleteIntegration

	// Stage 8
	handlers["DescribeReservedDBInstances"] = (*Server).handleRDSDescribeReservedDBInstances
	handlers["DescribeReservedDBInstancesOfferings"] = (*Server).handleRDSDescribeReservedDBInstancesOfferings
	handlers["PurchaseReservedDBInstancesOffering"] = (*Server).handleRDSPurchaseReservedDBInstancesOffering

	// Stage 9
	for _, action := range rdsStage9CompatActions {
		handlers[action] = (*Server).handleRDSCompatNoop
	}

	return handlers
}()

func isRDSQueryCandidate(r *http.Request) bool {
	if strings.Contains(strings.ToLower(r.Host), "rds") {
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/rds") {
		return true
	}
	if service := strings.TrimSpace(sigV4ServiceHint(r)); service != "" && service != "rds" {
		return false
	}

	action := strings.TrimSpace(r.URL.Query().Get("Action"))
	if action != "" {
		if !isRDSAction(action) {
			return false
		}
		if version := strings.TrimSpace(r.URL.Query().Get("Version")); version != "" && version != "2014-10-31" {
			return false
		}
		return true
	}

	if r.Method != http.MethodPost {
		return false
	}
	if !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/x-www-form-urlencoded") {
		return false
	}
	bodyBytes, err := readBodyBytes(r)
	if err != nil {
		return false
	}
	values, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		return false
	}
	action = strings.TrimSpace(values.Get("Action"))
	if action == "" || !isRDSAction(action) {
		return false
	}
	if version := strings.TrimSpace(values.Get("Version")); version != "" && version != "2014-10-31" {
		return false
	}
	return true
}

func isRDSAction(action string) bool {
	_, ok := rdsOperationByName[strings.TrimSpace(action)]
	return ok
}

func (s *Server) handleRDSQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isRDSQueryCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "rds")
	if !ok {
		respondRDSErrorXML(w, status, code, msg)
		return true
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		respondRDSErrorXML(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}

	if err := r.ParseForm(); err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid form body")
		return true
	}
	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondRDSErrorXML(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return true
	}
	if !isRDSAction(action) {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidAction", "unknown operation")
		return true
	}
	if version := strings.TrimSpace(r.Form.Get("Version")); version != "" && version != "2014-10-31" {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "unsupported Version")
		return true
	}

	handler, ok := rdsQueryHandlers[action]
	if !ok {
		respondRDSErrorXML(w, http.StatusNotImplemented, "NotImplemented", "operation not implemented")
		return true
	}
	handler(s, w, r)
	return true
}

func (s *Server) handleRDSNotImplemented(w http.ResponseWriter, _ *http.Request) {
	respondRDSErrorXML(w, http.StatusNotImplemented, "NotImplemented", "operation not implemented")
}

func (s *Server) handleRDSCreateDBInstance(w http.ResponseWriter, r *http.Request) {
	allocatedStorage, err := parseRequiredRDSInt(r.Form.Get("AllocatedStorage"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "AllocatedStorage is required")
		return
	}
	backupRetention, err := parseOptionalRDSInt(r.Form.Get("BackupRetentionPeriod"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "BackupRetentionPeriod is invalid")
		return
	}
	publiclyAccessible, err := parseOptionalRDSBool(r.Form.Get("PubliclyAccessible"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "PubliclyAccessible is invalid")
		return
	}

	instance, err := s.rds.CreateDBInstance(rdssvc.CreateDBInstanceInput{
		Identifier:            strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")),
		Engine:                strings.TrimSpace(r.Form.Get("Engine")),
		DBInstanceClass:       strings.TrimSpace(r.Form.Get("DBInstanceClass")),
		AllocatedStorage:      allocatedStorage,
		MasterUsername:        strings.TrimSpace(r.Form.Get("MasterUsername")),
		MasterUserPassword:    strings.TrimSpace(r.Form.Get("MasterUserPassword")),
		DBName:                strings.TrimSpace(r.Form.Get("DBName")),
		BackupRetentionPeriod: backupRetention,
		PubliclyAccessible:    publiclyAccessible,
		DBSubnetGroupName:     strings.TrimSpace(r.Form.Get("DBSubnetGroupName")),
		DBParameterGroupName:  strings.TrimSpace(r.Form.Get("DBParameterGroupName")),
		OptionGroupName:       strings.TrimSpace(r.Form.Get("OptionGroupName")),
	})
	if err != nil {
		respondRDSServiceError(w, "CreateDBInstance", err)
		return
	}

	respondRDSXML(w, "CreateDBInstance", struct {
		XMLName    xml.Name         `xml:"CreateDBInstanceResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleRDSDescribeDBInstances(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}

	instances, marker, err := s.rds.DescribeDBInstances(rdssvc.DescribeDBInstancesInput{
		Identifier: strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBInstances", err)
		return
	}

	out := make([]rdsDBInstanceXML, 0, len(instances))
	for _, item := range instances {
		out = append(out, rdsDBInstanceToXML(item))
	}

	respondRDSXML(w, "DescribeDBInstances", struct {
		XMLName     xml.Name           `xml:"DescribeDBInstancesResult"`
		Marker      string             `xml:"Marker,omitempty"`
		DBInstances []rdsDBInstanceXML `xml:"DBInstances>DBInstance"`
	}{
		Marker:      marker,
		DBInstances: out,
	})
}

func (s *Server) handleRDSModifyDBInstance(w http.ResponseWriter, r *http.Request) {
	allocatedStorage, err := parseOptionalRDSInt(r.Form.Get("AllocatedStorage"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "AllocatedStorage is invalid")
		return
	}
	backupRetention, err := parseOptionalRDSInt(r.Form.Get("BackupRetentionPeriod"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "BackupRetentionPeriod is invalid")
		return
	}
	publiclyAccessible, err := parseOptionalRDSBoolPtr(r.Form.Get("PubliclyAccessible"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "PubliclyAccessible is invalid")
		return
	}
	applyImmediately, err := parseOptionalRDSBoolPtr(r.Form.Get("ApplyImmediately"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "ApplyImmediately is invalid")
		return
	}

	instance, err := s.rds.ModifyDBInstance(rdssvc.ModifyDBInstanceInput{
		Identifier:            strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")),
		DBInstanceClass:       strings.TrimSpace(r.Form.Get("DBInstanceClass")),
		AllocatedStorage:      allocatedStorage,
		BackupRetentionPeriod: backupRetention,
		PubliclyAccessible:    publiclyAccessible,
		ApplyImmediately:      boolValue(applyImmediately),
	})
	if err != nil {
		respondRDSServiceError(w, "ModifyDBInstance", err)
		return
	}

	respondRDSXML(w, "ModifyDBInstance", struct {
		XMLName    xml.Name         `xml:"ModifyDBInstanceResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleRDSDeleteDBInstance(w http.ResponseWriter, r *http.Request) {
	skipFinalSnapshot, err := parseOptionalRDSBool(r.Form.Get("SkipFinalSnapshot"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "SkipFinalSnapshot is invalid")
		return
	}

	instance, err := s.rds.DeleteDBInstance(rdssvc.DeleteDBInstanceInput{
		Identifier:                strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")),
		SkipFinalSnapshot:         skipFinalSnapshot,
		FinalDBSnapshotIdentifier: strings.TrimSpace(r.Form.Get("FinalDBSnapshotIdentifier")),
	})
	if err != nil {
		respondRDSServiceError(w, "DeleteDBInstance", err)
		return
	}

	respondRDSXML(w, "DeleteDBInstance", struct {
		XMLName    xml.Name         `xml:"DeleteDBInstanceResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleRDSStartDBInstance(w http.ResponseWriter, r *http.Request) {
	instance, err := s.rds.StartDBInstance(strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "StartDBInstance", err)
		return
	}
	respondRDSXML(w, "StartDBInstance", struct {
		XMLName    xml.Name         `xml:"StartDBInstanceResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleRDSStopDBInstance(w http.ResponseWriter, r *http.Request) {
	instance, err := s.rds.StopDBInstance(strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "StopDBInstance", err)
		return
	}
	respondRDSXML(w, "StopDBInstance", struct {
		XMLName    xml.Name         `xml:"StopDBInstanceResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleRDSRebootDBInstance(w http.ResponseWriter, r *http.Request) {
	instance, err := s.rds.RebootDBInstance(strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "RebootDBInstance", err)
		return
	}
	respondRDSXML(w, "RebootDBInstance", struct {
		XMLName    xml.Name         `xml:"RebootDBInstanceResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleRDSCreateDBSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.rds.CreateDBSnapshot(rdssvc.CreateDBSnapshotInput{
		Identifier:           strings.TrimSpace(r.Form.Get("DBSnapshotIdentifier")),
		DBInstanceIdentifier: strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")),
	})
	if err != nil {
		respondRDSServiceError(w, "CreateDBSnapshot", err)
		return
	}
	respondRDSXML(w, "CreateDBSnapshot", struct {
		XMLName    xml.Name         `xml:"CreateDBSnapshotResult"`
		DBSnapshot rdsDBSnapshotXML `xml:"DBSnapshot"`
	}{
		DBSnapshot: rdsDBSnapshotToXML(snapshot),
	})
}

func (s *Server) handleRDSDescribeDBSnapshots(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}

	snapshots, marker, err := s.rds.DescribeDBSnapshots(rdssvc.DescribeDBSnapshotsInput{
		Identifier:           strings.TrimSpace(r.Form.Get("DBSnapshotIdentifier")),
		DBInstanceIdentifier: strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")),
		MaxRecords:           maxRecords,
		Marker:               strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBSnapshots", err)
		return
	}

	out := make([]rdsDBSnapshotXML, 0, len(snapshots))
	for _, item := range snapshots {
		out = append(out, rdsDBSnapshotToXML(item))
	}

	respondRDSXML(w, "DescribeDBSnapshots", struct {
		XMLName     xml.Name           `xml:"DescribeDBSnapshotsResult"`
		Marker      string             `xml:"Marker,omitempty"`
		DBSnapshots []rdsDBSnapshotXML `xml:"DBSnapshots>DBSnapshot"`
	}{
		Marker:      marker,
		DBSnapshots: out,
	})
}

func (s *Server) handleRDSDeleteDBSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.rds.DeleteDBSnapshot(strings.TrimSpace(r.Form.Get("DBSnapshotIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "DeleteDBSnapshot", err)
		return
	}
	respondRDSXML(w, "DeleteDBSnapshot", struct {
		XMLName    xml.Name         `xml:"DeleteDBSnapshotResult"`
		DBSnapshot rdsDBSnapshotXML `xml:"DBSnapshot"`
	}{
		DBSnapshot: rdsDBSnapshotToXML(snapshot),
	})
}

func (s *Server) handleRDSCopyDBSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.rds.CopyDBSnapshot(rdssvc.CopyDBSnapshotInput{
		SourceIdentifier: strings.TrimSpace(r.Form.Get("SourceDBSnapshotIdentifier")),
		TargetIdentifier: strings.TrimSpace(r.Form.Get("TargetDBSnapshotIdentifier")),
	})
	if err != nil {
		respondRDSServiceError(w, "CopyDBSnapshot", err)
		return
	}
	respondRDSXML(w, "CopyDBSnapshot", struct {
		XMLName    xml.Name         `xml:"CopyDBSnapshotResult"`
		DBSnapshot rdsDBSnapshotXML `xml:"DBSnapshot"`
	}{
		DBSnapshot: rdsDBSnapshotToXML(snapshot),
	})
}

func (s *Server) handleRDSRestoreDBInstanceFromDBSnapshot(w http.ResponseWriter, r *http.Request) {
	publiclyAccessible, err := parseOptionalRDSBool(r.Form.Get("PubliclyAccessible"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "PubliclyAccessible is invalid")
		return
	}

	instance, err := s.rds.RestoreDBInstanceFromSnapshot(rdssvc.RestoreDBInstanceFromSnapshotInput{
		DBInstanceIdentifier: strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")),
		DBSnapshotIdentifier: strings.TrimSpace(r.Form.Get("DBSnapshotIdentifier")),
		DBInstanceClass:      strings.TrimSpace(r.Form.Get("DBInstanceClass")),
		PubliclyAccessible:   publiclyAccessible,
	})
	if err != nil {
		respondRDSServiceError(w, "RestoreDBInstanceFromDBSnapshot", err)
		return
	}

	respondRDSXML(w, "RestoreDBInstanceFromDBSnapshot", struct {
		XMLName    xml.Name         `xml:"RestoreDBInstanceFromDBSnapshotResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleRDSRestoreDBInstanceToPointInTime(w http.ResponseWriter, r *http.Request) {
	publiclyAccessible, err := parseOptionalRDSBool(r.Form.Get("PubliclyAccessible"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "PubliclyAccessible is invalid")
		return
	}

	instance, err := s.rds.RestoreDBInstanceToPointInTime(rdssvc.RestoreDBInstanceToPointInTimeInput{
		SourceDBInstanceIdentifier: strings.TrimSpace(r.Form.Get("SourceDBInstanceIdentifier")),
		TargetDBInstanceIdentifier: strings.TrimSpace(r.Form.Get("TargetDBInstanceIdentifier")),
		DBInstanceClass:            strings.TrimSpace(r.Form.Get("DBInstanceClass")),
		PubliclyAccessible:         publiclyAccessible,
	})
	if err != nil {
		respondRDSServiceError(w, "RestoreDBInstanceToPointInTime", err)
		return
	}

	respondRDSXML(w, "RestoreDBInstanceToPointInTime", struct {
		XMLName    xml.Name         `xml:"RestoreDBInstanceToPointInTimeResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleRDSStartExportTask(w http.ResponseWriter, r *http.Request) {
	task, err := s.rds.StartExportTask(rdssvc.StartExportTaskInput{
		Identifier: strings.TrimSpace(r.Form.Get("ExportTaskIdentifier")),
		SourceArn:  strings.TrimSpace(r.Form.Get("SourceArn")),
		S3Bucket:   strings.TrimSpace(r.Form.Get("S3BucketName")),
		S3Prefix:   strings.TrimSpace(r.Form.Get("S3Prefix")),
		KmsKeyID:   strings.TrimSpace(r.Form.Get("KmsKeyId")),
	})
	if err != nil {
		respondRDSServiceError(w, "StartExportTask", err)
		return
	}
	respondRDSXML(w, "StartExportTask", rdsExportTaskResultToXML("StartExportTaskResult", task))
}

func (s *Server) handleRDSDescribeExportTasks(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeExportTasks(rdssvc.DescribeExportTasksInput{
		Identifier: strings.TrimSpace(r.Form.Get("ExportTaskIdentifier")),
		SourceArn:  strings.TrimSpace(r.Form.Get("SourceArn")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeExportTasks", err)
		return
	}

	out := make([]rdsExportTaskXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsExportTaskToXML(item))
	}

	respondRDSXML(w, "DescribeExportTasks", struct {
		XMLName     xml.Name           `xml:"DescribeExportTasksResult"`
		Marker      string             `xml:"Marker,omitempty"`
		ExportTasks []rdsExportTaskXML `xml:"ExportTasks>ExportTask"`
	}{
		Marker:      marker,
		ExportTasks: out,
	})
}

func (s *Server) handleRDSCancelExportTask(w http.ResponseWriter, r *http.Request) {
	task, err := s.rds.CancelExportTask(strings.TrimSpace(r.Form.Get("ExportTaskIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "CancelExportTask", err)
		return
	}
	respondRDSXML(w, "CancelExportTask", rdsExportTaskResultToXML("CancelExportTaskResult", task))
}

func (s *Server) handleRDSStartDBInstanceAutomatedBackupsReplication(w http.ResponseWriter, r *http.Request) {
	backupRetention, err := parseOptionalRDSInt(r.Form.Get("BackupRetentionPeriod"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "BackupRetentionPeriod is invalid")
		return
	}
	backup, err := s.rds.StartDBInstanceAutomatedBackupsReplication(rdssvc.StartDBInstanceAutomatedBackupsReplicationInput{
		SourceDBInstanceArn:   strings.TrimSpace(r.Form.Get("SourceDBInstanceArn")),
		BackupRetentionPeriod: backupRetention,
		KmsKeyID:              strings.TrimSpace(r.Form.Get("KmsKeyId")),
		Region:                strings.TrimSpace(r.Form.Get("SourceRegion")),
	})
	if err != nil {
		respondRDSServiceError(w, "StartDBInstanceAutomatedBackupsReplication", err)
		return
	}
	respondRDSXML(w, "StartDBInstanceAutomatedBackupsReplication", struct {
		XMLName                    xml.Name                   `xml:"StartDBInstanceAutomatedBackupsReplicationResult"`
		DBInstanceAutomatedBackups rdsAutomatedBackupShortXML `xml:"DBInstanceAutomatedBackup"`
	}{
		DBInstanceAutomatedBackups: rdsAutomatedBackupToShortXML(backup),
	})
}

func (s *Server) handleRDSStopDBInstanceAutomatedBackupsReplication(w http.ResponseWriter, r *http.Request) {
	backup, err := s.rds.StopDBInstanceAutomatedBackupsReplication(rdssvc.StopDBInstanceAutomatedBackupsReplicationInput{
		SourceDBInstanceArn: strings.TrimSpace(r.Form.Get("SourceDBInstanceArn")),
	})
	if err != nil {
		respondRDSServiceError(w, "StopDBInstanceAutomatedBackupsReplication", err)
		return
	}
	respondRDSXML(w, "StopDBInstanceAutomatedBackupsReplication", struct {
		XMLName                    xml.Name                   `xml:"StopDBInstanceAutomatedBackupsReplicationResult"`
		DBInstanceAutomatedBackups rdsAutomatedBackupShortXML `xml:"DBInstanceAutomatedBackup"`
	}{
		DBInstanceAutomatedBackups: rdsAutomatedBackupToShortXML(backup),
	})
}

func (s *Server) handleRDSDescribeDBInstanceAutomatedBackups(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeDBInstanceAutomatedBackups(rdssvc.DescribeDBInstanceAutomatedBackupsInput{
		DBInstanceIdentifier:          strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")),
		DBInstanceAutomatedBackupsArn: strings.TrimSpace(r.Form.Get("DBInstanceAutomatedBackupsArn")),
		DbiResourceID:                 strings.TrimSpace(firstNonEmpty(r.Form.Get("DbiResourceId"), r.Form.Get("DbiResourceID"))),
		MaxRecords:                    maxRecords,
		Marker:                        strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBInstanceAutomatedBackups", err)
		return
	}

	out := make([]rdsAutomatedBackupXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsAutomatedBackupToXML(item))
	}

	respondRDSXML(w, "DescribeDBInstanceAutomatedBackups", struct {
		XMLName                    xml.Name                `xml:"DescribeDBInstanceAutomatedBackupsResult"`
		Marker                     string                  `xml:"Marker,omitempty"`
		DBInstanceAutomatedBackups []rdsAutomatedBackupXML `xml:"DBInstanceAutomatedBackups>DBInstanceAutomatedBackup"`
	}{
		Marker:                     marker,
		DBInstanceAutomatedBackups: out,
	})
}

func (s *Server) handleRDSDeleteDBInstanceAutomatedBackup(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.DeleteDBInstanceAutomatedBackup(rdssvc.DeleteDBInstanceAutomatedBackupInput{
		DbiResourceID:                 strings.TrimSpace(firstNonEmpty(r.Form.Get("DbiResourceId"), r.Form.Get("DbiResourceID"))),
		DBInstanceAutomatedBackupsArn: strings.TrimSpace(r.Form.Get("DBInstanceAutomatedBackupsArn")),
	})
	if err != nil {
		respondRDSServiceError(w, "DeleteDBInstanceAutomatedBackup", err)
		return
	}
	respondRDSXML(w, "DeleteDBInstanceAutomatedBackup", struct {
		XMLName                   xml.Name              `xml:"DeleteDBInstanceAutomatedBackupResult"`
		DBInstanceAutomatedBackup rdsAutomatedBackupXML `xml:"DBInstanceAutomatedBackup"`
	}{
		DBInstanceAutomatedBackup: rdsAutomatedBackupToXML(item),
	})
}

func respondRDSXML(w http.ResponseWriter, action string, result any) {
	env := rdsResponseEnvelope{
		XMLName: xml.Name{Local: action + "Response"},
		Xmlns:   rdsNamespace,
		Result:  result,
		Metadata: rdsResponseMetadata{
			RequestID: "stackyard-request",
		},
	}
	respondXML(w, http.StatusOK, env)
}

func respondRDSErrorXML(w http.ResponseWriter, status int, code, message string) {
	respondXML(w, status, rdsErrorResponse{
		Xmlns: rdsNamespace,
		Error: rdsErrorBody{
			Type:    "Sender",
			Code:    code,
			Message: message,
		},
		RequestID: "stackyard-request",
	})
}

func respondRDSServiceError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, rdssvc.ErrInvalidParameter):
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid request parameters")
	case errors.Is(err, rdssvc.ErrAlreadyExists):
		respondRDSErrorXML(w, http.StatusBadRequest, rdsAlreadyExistsCode(action), "resource already exists")
	case errors.Is(err, rdssvc.ErrNotFound):
		respondRDSErrorXML(w, http.StatusNotFound, rdsNotFoundCode(action), "resource not found")
	case errors.Is(err, rdssvc.ErrInvalidState):
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidDBInstanceState", "resource is not in a valid state")
	default:
		respondRDSErrorXML(w, http.StatusInternalServerError, "InternalFailure", "internal server error")
	}
}

func rdsAlreadyExistsCode(action string) string {
	switch action {
	case "CreateDBInstance", "RestoreDBInstanceFromDBSnapshot", "RestoreDBInstanceToPointInTime":
		return "DBInstanceAlreadyExists"
	case "CreateDBSnapshot", "CopyDBSnapshot":
		return "DBSnapshotAlreadyExists"
	case "StartExportTask":
		return "ExportTaskAlreadyExists"
	case "CreateDBParameterGroup":
		return "DBParameterGroupAlreadyExists"
	case "CreateOptionGroup":
		return "OptionGroupAlreadyExistsFault"
	case "CreateDBSubnetGroup":
		return "DBSubnetGroupAlreadyExistsFault"
	case "CreateDBSecurityGroup":
		return "DBSecurityGroupAlreadyExists"
	case "CreateDBCluster":
		return "DBClusterAlreadyExistsFault"
	case "CreateDBClusterEndpoint":
		return "DBClusterEndpointAlreadyExistsFault"
	case "CreateGlobalCluster":
		return "GlobalClusterAlreadyExistsFault"
	case "CreateBlueGreenDeployment":
		return "BlueGreenDeploymentAlreadyExistsFault"
	case "CreateTenantDatabase":
		return "TenantDatabaseAlreadyExistsFault"
	case "CreateEventSubscription":
		return "SubscriptionAlreadyExist"
	case "CreateDBProxy":
		return "DBProxyAlreadyExistsFault"
	case "CreateDBProxyEndpoint":
		return "DBProxyEndpointAlreadyExistsFault"
	case "CreateIntegration":
		return "IntegrationAlreadyExistsFault"
	case "PurchaseReservedDBInstancesOffering":
		return "ReservedDBInstanceAlreadyExists"
	default:
		return "ResourceAlreadyExists"
	}
}

func rdsNotFoundCode(action string) string {
	switch action {
	case "CreateDBInstance", "DescribeDBInstances", "ModifyDBInstance", "DeleteDBInstance", "StartDBInstance", "StopDBInstance", "RebootDBInstance", "RestoreDBInstanceToPointInTime":
		return "DBInstanceNotFound"
	case "CreateDBSnapshot", "DescribeDBSnapshots", "DeleteDBSnapshot", "CopyDBSnapshot", "RestoreDBInstanceFromDBSnapshot":
		return "DBSnapshotNotFound"
	case "DescribeExportTasks", "CancelExportTask":
		return "ExportTaskNotFound"
	case "DescribeDBInstanceAutomatedBackups", "DeleteDBInstanceAutomatedBackup", "StopDBInstanceAutomatedBackupsReplication":
		return "DBInstanceAutomatedBackupNotFound"
	case "DescribeDBParameterGroups", "DescribeDBParameters", "ModifyDBParameterGroup", "ResetDBParameterGroup", "DeleteDBParameterGroup":
		return "DBParameterGroupNotFound"
	case "DescribeOptionGroups", "ModifyOptionGroup", "DeleteOptionGroup":
		return "OptionGroupNotFoundFault"
	case "DescribeDBSubnetGroups", "ModifyDBSubnetGroup", "DeleteDBSubnetGroup":
		return "DBSubnetGroupNotFoundFault"
	case "DescribeDBSecurityGroups", "AuthorizeDBSecurityGroupIngress", "RevokeDBSecurityGroupIngress", "DeleteDBSecurityGroup":
		return "DBSecurityGroupNotFound"
	case "DescribeCertificates":
		return "CertificateNotFound"
	case "CreateDBCluster", "DescribeDBClusters", "ModifyDBCluster", "DeleteDBCluster", "StartDBCluster", "StopDBCluster", "RebootDBCluster", "FailoverDBCluster":
		return "DBClusterNotFoundFault"
	case "CreateDBClusterEndpoint", "DescribeDBClusterEndpoints", "ModifyDBClusterEndpoint", "DeleteDBClusterEndpoint":
		return "DBClusterEndpointNotFoundFault"
	case "CreateGlobalCluster", "DescribeGlobalClusters", "ModifyGlobalCluster", "DeleteGlobalCluster", "FailoverGlobalCluster", "SwitchoverGlobalCluster":
		return "GlobalClusterNotFoundFault"
	case "CreateBlueGreenDeployment", "DescribeBlueGreenDeployments", "SwitchoverBlueGreenDeployment", "DeleteBlueGreenDeployment":
		return "BlueGreenDeploymentNotFoundFault"
	case "CreateTenantDatabase", "DescribeTenantDatabases", "ModifyTenantDatabase", "DeleteTenantDatabase":
		return "TenantDatabaseNotFoundFault"
	case "AddTagsToResource", "ListTagsForResource", "RemoveTagsFromResource", "DescribePendingMaintenanceActions", "ApplyPendingMaintenanceAction", "DescribeValidDBInstanceModifications":
		return "ResourceNotFoundFault"
	case "DescribeEventSubscriptions", "ModifyEventSubscription", "DeleteEventSubscription":
		return "SubscriptionNotFound"
	case "AddRoleToDBInstance", "RemoveRoleFromDBInstance":
		return "DBInstanceNotFound"
	case "AddRoleToDBCluster", "RemoveRoleFromDBCluster":
		return "DBClusterNotFoundFault"
	case "StartActivityStream", "StopActivityStream":
		return "ResourceNotFoundFault"
	case "DescribeDBProxies", "ModifyDBProxy", "DeleteDBProxy", "RegisterDBProxyTargets", "DeregisterDBProxyTargets", "DescribeDBProxyTargets":
		return "DBProxyNotFoundFault"
	case "DescribeDBProxyEndpoints", "ModifyDBProxyEndpoint", "DeleteDBProxyEndpoint":
		return "DBProxyEndpointNotFoundFault"
	case "DescribeIntegrations", "ModifyIntegration", "DeleteIntegration":
		return "IntegrationNotFoundFault"
	case "DescribeReservedDBInstances":
		return "ReservedDBInstanceNotFound"
	case "DescribeReservedDBInstancesOfferings", "PurchaseReservedDBInstancesOffering":
		return "ReservedDBInstancesOfferingNotFound"
	default:
		return "ResourceNotFound"
	}
}

func parseRequiredRDSInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, strconv.ErrSyntax
	}
	out, err := strconv.Atoi(value)
	if err != nil || out <= 0 {
		return 0, strconv.ErrSyntax
	}
	return out, nil
}

func parseOptionalRDSInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	out, err := strconv.Atoi(value)
	if err != nil || out < 0 {
		return 0, strconv.ErrSyntax
	}
	return out, nil
}

func parseOptionalRDSBool(value string) (bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return false, nil
	}
	out, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}
	return out, nil
}

func parseOptionalRDSBoolPtr(value string) (*bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	out, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func rdsDBInstanceToXML(in rdssvc.DBInstance) rdsDBInstanceXML {
	instance := rdsDBInstanceXML{
		DBInstanceIdentifier:  in.Identifier,
		DBInstanceArn:         in.ARN,
		Engine:                in.Engine,
		DBInstanceClass:       in.DBInstanceClass,
		AllocatedStorage:      in.AllocatedStorage,
		MasterUsername:        in.MasterUsername,
		DBName:                in.DBName,
		DBInstanceStatus:      in.Status,
		BackupRetentionPeriod: in.BackupRetentionPeriod,
		PubliclyAccessible:    in.PubliclyAccessible,
		InstanceCreateTime:    formatRDSTime(in.CreatedAt),
	}
	if in.EndpointAddress != "" {
		instance.Endpoint = &rdsEndpointXML{
			Address: in.EndpointAddress,
			Port:    in.Port,
		}
	}
	if in.DBSubnetGroupName != "" {
		instance.DBSubnetGroup = &rdsDBSubnetGroupXML{DBSubnetGroupName: in.DBSubnetGroupName}
	}
	if in.DBParameterGroupName != "" {
		instance.DBParameterGroups = []rdsDBParameterGroupStatusXML{{
			DBParameterGroupName: in.DBParameterGroupName,
			ParameterApplyStatus: "in-sync",
		}}
	}
	if in.OptionGroupName != "" {
		instance.OptionGroupMemberships = []rdsOptionGroupMembershipXML{{
			OptionGroupName: in.OptionGroupName,
			Status:          "in-sync",
		}}
	}
	return instance
}

func rdsDBSnapshotToXML(in rdssvc.DBSnapshot) rdsDBSnapshotXML {
	return rdsDBSnapshotXML{
		DBSnapshotIdentifier: in.Identifier,
		DBSnapshotArn:        in.ARN,
		DBInstanceIdentifier: in.DBInstanceIdentifier,
		Status:               in.Status,
		SnapshotType:         in.SnapshotType,
		Engine:               in.Engine,
		AllocatedStorage:     in.AllocatedStorage,
		SnapshotCreateTime:   formatRDSTime(in.CreatedAt),
	}
}

func rdsExportTaskToXML(in rdssvc.ExportTask) rdsExportTaskXML {
	return rdsExportTaskXML{
		ExportTaskIdentifier: in.Identifier,
		ExportTaskArn:        in.ARN,
		SourceArn:            in.SourceArn,
		S3Bucket:             in.S3Bucket,
		S3Prefix:             in.S3Prefix,
		KmsKeyId:             in.KmsKeyID,
		Status:               in.Status,
	}
}

func rdsAutomatedBackupToShortXML(in rdssvc.DBInstanceAutomatedBackup) rdsAutomatedBackupShortXML {
	return rdsAutomatedBackupShortXML{
		DbiResourceId: in.DbiResourceID,
		DBInstanceArn: in.DBInstanceARN,
		Status:        in.Status,
		Region:        in.Region,
		KmsKeyId:      in.KmsKeyID,
	}
}

func rdsAutomatedBackupToXML(in rdssvc.DBInstanceAutomatedBackup) rdsAutomatedBackupXML {
	return rdsAutomatedBackupXML{
		DbiResourceId:                 in.DbiResourceID,
		DBInstanceArn:                 in.DBInstanceARN,
		DBInstanceAutomatedBackupsArn: rdsAutomatedBackupARN(in.DbiResourceID),
		Status:                        in.Status,
		Region:                        in.Region,
		KmsKeyId:                      in.KmsKeyID,
		InstanceCreateTime:            formatRDSTime(in.CreatedAt),
	}
}

func formatRDSTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func rdsAutomatedBackupARN(dbiResourceID string) string {
	if strings.TrimSpace(dbiResourceID) == "" {
		return ""
	}
	return "arn:aws:rds:us-east-1:123456789012:auto-backup:" + strings.TrimSpace(dbiResourceID)
}

type rdsResponseEnvelope struct {
	XMLName  xml.Name            `xml:""`
	Xmlns    string              `xml:"xmlns,attr,omitempty"`
	Result   any                 `xml:",any"`
	Metadata rdsResponseMetadata `xml:"ResponseMetadata"`
}

type rdsResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type rdsErrorResponse struct {
	XMLName   xml.Name     `xml:"ErrorResponse"`
	Xmlns     string       `xml:"xmlns,attr,omitempty"`
	Error     rdsErrorBody `xml:"Error"`
	RequestID string       `xml:"RequestId"`
}

type rdsErrorBody struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

type rdsEndpointXML struct {
	Address string `xml:"Address,omitempty"`
	Port    int    `xml:"Port,omitempty"`
}

type rdsDBSubnetGroupXML struct {
	DBSubnetGroupName string `xml:"DBSubnetGroupName,omitempty"`
}

type rdsDBParameterGroupStatusXML struct {
	DBParameterGroupName string `xml:"DBParameterGroupName,omitempty"`
	ParameterApplyStatus string `xml:"ParameterApplyStatus,omitempty"`
}

type rdsOptionGroupMembershipXML struct {
	OptionGroupName string `xml:"OptionGroupName,omitempty"`
	Status          string `xml:"Status,omitempty"`
}

type rdsDBInstanceXML struct {
	DBInstanceIdentifier   string                         `xml:"DBInstanceIdentifier,omitempty"`
	DBInstanceArn          string                         `xml:"DBInstanceArn,omitempty"`
	Engine                 string                         `xml:"Engine,omitempty"`
	DBInstanceClass        string                         `xml:"DBInstanceClass,omitempty"`
	AllocatedStorage       int                            `xml:"AllocatedStorage,omitempty"`
	MasterUsername         string                         `xml:"MasterUsername,omitempty"`
	DBName                 string                         `xml:"DBName,omitempty"`
	DBInstanceStatus       string                         `xml:"DBInstanceStatus,omitempty"`
	Endpoint               *rdsEndpointXML                `xml:"Endpoint,omitempty"`
	BackupRetentionPeriod  int                            `xml:"BackupRetentionPeriod,omitempty"`
	PubliclyAccessible     bool                           `xml:"PubliclyAccessible"`
	DBSubnetGroup          *rdsDBSubnetGroupXML           `xml:"DBSubnetGroup,omitempty"`
	DBParameterGroups      []rdsDBParameterGroupStatusXML `xml:"DBParameterGroups>DBParameterGroup,omitempty"`
	OptionGroupMemberships []rdsOptionGroupMembershipXML  `xml:"OptionGroupMemberships>OptionGroupMembership,omitempty"`
	InstanceCreateTime     string                         `xml:"InstanceCreateTime,omitempty"`
}

type rdsDBSnapshotXML struct {
	DBSnapshotIdentifier string `xml:"DBSnapshotIdentifier,omitempty"`
	DBSnapshotArn        string `xml:"DBSnapshotArn,omitempty"`
	DBInstanceIdentifier string `xml:"DBInstanceIdentifier,omitempty"`
	Status               string `xml:"Status,omitempty"`
	SnapshotType         string `xml:"SnapshotType,omitempty"`
	Engine               string `xml:"Engine,omitempty"`
	AllocatedStorage     int    `xml:"AllocatedStorage,omitempty"`
	SnapshotCreateTime   string `xml:"SnapshotCreateTime,omitempty"`
}

type rdsExportTaskXML struct {
	ExportTaskIdentifier string `xml:"ExportTaskIdentifier,omitempty"`
	ExportTaskArn        string `xml:"ExportTaskArn,omitempty"`
	SourceArn            string `xml:"SourceArn,omitempty"`
	S3Bucket             string `xml:"S3Bucket,omitempty"`
	S3Prefix             string `xml:"S3Prefix,omitempty"`
	KmsKeyId             string `xml:"KmsKeyId,omitempty"`
	Status               string `xml:"Status,omitempty"`
}

type rdsExportTaskResultXML struct {
	XMLName                xml.Name `xml:""`
	ExportTaskIdentifier   string   `xml:"ExportTaskIdentifier,omitempty"`
	SourceArn              string   `xml:"SourceArn,omitempty"`
	S3Bucket               string   `xml:"S3Bucket,omitempty"`
	S3Prefix               string   `xml:"S3Prefix,omitempty"`
	KmsKeyId               string   `xml:"KmsKeyId,omitempty"`
	Status                 string   `xml:"Status,omitempty"`
	TaskStartTime          string   `xml:"TaskStartTime,omitempty"`
	TaskEndTime            string   `xml:"TaskEndTime,omitempty"`
	PercentProgress        int      `xml:"PercentProgress,omitempty"`
	TotalExtractedDataInGB int      `xml:"TotalExtractedDataInGB,omitempty"`
	SourceType             string   `xml:"SourceType,omitempty"`
}

type rdsAutomatedBackupShortXML struct {
	DbiResourceId string `xml:"DbiResourceId,omitempty"`
	DBInstanceArn string `xml:"DBInstanceArn,omitempty"`
	Status        string `xml:"Status,omitempty"`
	Region        string `xml:"Region,omitempty"`
	KmsKeyId      string `xml:"KmsKeyId,omitempty"`
}

func rdsExportTaskResultToXML(root string, in rdssvc.ExportTask) rdsExportTaskResultXML {
	return rdsExportTaskResultXML{
		XMLName:                xml.Name{Local: root},
		ExportTaskIdentifier:   in.Identifier,
		SourceArn:              in.SourceArn,
		S3Bucket:               in.S3Bucket,
		S3Prefix:               in.S3Prefix,
		KmsKeyId:               in.KmsKeyID,
		Status:                 in.Status,
		TaskStartTime:          formatRDSTime(in.CreatedAt),
		TaskEndTime:            formatRDSTime(in.UpdatedAt),
		PercentProgress:        0,
		TotalExtractedDataInGB: 0,
		SourceType:             "SNAPSHOT",
	}
}

type rdsAutomatedBackupXML struct {
	DbiResourceId                 string `xml:"DbiResourceId,omitempty"`
	DBInstanceArn                 string `xml:"DBInstanceArn,omitempty"`
	DBInstanceAutomatedBackupsArn string `xml:"DBInstanceAutomatedBackupsArn,omitempty"`
	Status                        string `xml:"Status,omitempty"`
	Region                        string `xml:"Region,omitempty"`
	KmsKeyId                      string `xml:"KmsKeyId,omitempty"`
	InstanceCreateTime            string `xml:"InstanceCreateTime,omitempty"`
}
