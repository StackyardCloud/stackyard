package server

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"

	rdssvc "github.com/stackyard/stackyard/internal/services/rds"
)

const neptuneNamespace = "http://neptune.amazonaws.com/doc/2014-10-31/"

var neptuneQueryHandlers = func() map[string]func(*Server, http.ResponseWriter, *http.Request) {
	handlers := make(map[string]func(*Server, http.ResponseWriter, *http.Request), len(neptuneOperations))
	for _, op := range neptuneOperations {
		handlers[op.Name] = (*Server).handleNeptuneNotImplemented
	}

	// Stage 1: read-only describe/list surface.
	for _, op := range neptuneOperations {
		if strings.HasPrefix(op.Name, "Describe") {
			handlers[op.Name] = (*Server).handleNeptuneDescribeCompatNoop
		}
	}
	handlers["DescribeDBClusters"] = (*Server).handleNeptuneDescribeDBClusters
	handlers["DescribeDBClusterEndpoints"] = (*Server).handleNeptuneDescribeDBClusterEndpoints
	handlers["DescribeGlobalClusters"] = (*Server).handleNeptuneDescribeGlobalClusters
	handlers["DescribeDBInstances"] = (*Server).handleNeptuneDescribeDBInstances
	handlers["DescribeDBEngineVersions"] = (*Server).handleNeptuneDescribeDBEngineVersions
	handlers["DescribeOrderableDBInstanceOptions"] = (*Server).handleNeptuneDescribeOrderableDBInstanceOptions
	handlers["DescribePendingMaintenanceActions"] = (*Server).handleNeptuneDescribePendingMaintenanceActions
	handlers["DescribeEvents"] = (*Server).handleNeptuneDescribeEvents
	handlers["ListTagsForResource"] = (*Server).handleNeptuneListTagsForResource

	// Stage 2: core DB cluster lifecycle.
	handlers["CreateDBCluster"] = (*Server).handleNeptuneCreateDBCluster
	handlers["ModifyDBCluster"] = (*Server).handleNeptuneModifyDBCluster
	handlers["DeleteDBCluster"] = (*Server).handleNeptuneDeleteDBCluster
	handlers["StartDBCluster"] = (*Server).handleNeptuneStartDBCluster
	handlers["StopDBCluster"] = (*Server).handleNeptuneStopDBCluster
	handlers["FailoverDBCluster"] = (*Server).handleNeptuneFailoverDBCluster

	// Stage 3: instance/subnet/parameter group lifecycle.
	handlers["CreateDBInstance"] = (*Server).handleNeptuneCreateDBInstance
	handlers["ModifyDBInstance"] = (*Server).handleNeptuneModifyDBInstance
	handlers["RebootDBInstance"] = (*Server).handleNeptuneRebootDBInstance
	handlers["DeleteDBInstance"] = (*Server).handleNeptuneDeleteDBInstance
	handlers["CreateDBParameterGroup"] = (*Server).handleNeptuneCreateDBParameterGroup
	handlers["DescribeDBParameterGroups"] = (*Server).handleNeptuneDescribeDBParameterGroups
	handlers["DescribeDBParameters"] = (*Server).handleNeptuneDescribeDBParameters
	handlers["ModifyDBParameterGroup"] = (*Server).handleNeptuneModifyDBParameterGroup
	handlers["ResetDBParameterGroup"] = (*Server).handleNeptuneResetDBParameterGroup
	handlers["DeleteDBParameterGroup"] = (*Server).handleNeptuneDeleteDBParameterGroup
	handlers["CreateDBClusterParameterGroup"] = (*Server).handleNeptuneCreateDBClusterParameterGroup
	handlers["DescribeDBClusterParameterGroups"] = (*Server).handleNeptuneDescribeDBClusterParameterGroups
	handlers["DescribeDBClusterParameters"] = (*Server).handleNeptuneDescribeDBClusterParameters
	handlers["ModifyDBClusterParameterGroup"] = (*Server).handleNeptuneModifyDBClusterParameterGroup
	handlers["ResetDBClusterParameterGroup"] = (*Server).handleNeptuneResetDBClusterParameterGroup
	handlers["DeleteDBClusterParameterGroup"] = (*Server).handleNeptuneDeleteDBClusterParameterGroup
	handlers["CopyDBParameterGroup"] = (*Server).handleNeptuneCopyDBParameterGroup
	handlers["CopyDBClusterParameterGroup"] = (*Server).handleNeptuneCopyDBClusterParameterGroup
	handlers["CreateDBSubnetGroup"] = (*Server).handleNeptuneCreateDBSubnetGroup
	handlers["DescribeDBSubnetGroups"] = (*Server).handleNeptuneDescribeDBSubnetGroups
	handlers["ModifyDBSubnetGroup"] = (*Server).handleNeptuneModifyDBSubnetGroup
	handlers["DeleteDBSubnetGroup"] = (*Server).handleNeptuneDeleteDBSubnetGroup

	// Stage 4: cluster snapshot/restore compatibility workflows.
	handlers["CreateDBClusterSnapshot"] = (*Server).handleNeptuneCreateDBClusterSnapshot
	handlers["DeleteDBClusterSnapshot"] = (*Server).handleNeptuneDeleteDBClusterSnapshot
	handlers["CopyDBClusterSnapshot"] = (*Server).handleNeptuneCopyDBClusterSnapshot
	handlers["DescribeDBClusterSnapshots"] = (*Server).handleNeptuneDescribeDBClusterSnapshots
	handlers["DescribeDBClusterSnapshotAttributes"] = (*Server).handleNeptuneDescribeDBClusterSnapshotAttributes
	handlers["ModifyDBClusterSnapshotAttribute"] = (*Server).handleNeptuneModifyDBClusterSnapshotAttribute
	handlers["RestoreDBClusterFromSnapshot"] = (*Server).handleNeptuneRestoreDBClusterFromSnapshot
	handlers["RestoreDBClusterToPointInTime"] = (*Server).handleNeptuneRestoreDBClusterToPointInTime

	// Stage 5: endpoint/global cluster + subscription workflows.
	handlers["CreateDBClusterEndpoint"] = (*Server).handleNeptuneCreateDBClusterEndpoint
	handlers["ModifyDBClusterEndpoint"] = (*Server).handleNeptuneModifyDBClusterEndpoint
	handlers["DeleteDBClusterEndpoint"] = (*Server).handleNeptuneDeleteDBClusterEndpoint
	handlers["CreateGlobalCluster"] = (*Server).handleNeptuneCreateGlobalCluster
	handlers["ModifyGlobalCluster"] = (*Server).handleNeptuneModifyGlobalCluster
	handlers["DeleteGlobalCluster"] = (*Server).handleNeptuneDeleteGlobalCluster
	handlers["FailoverGlobalCluster"] = (*Server).handleNeptuneFailoverGlobalCluster
	handlers["SwitchoverGlobalCluster"] = (*Server).handleNeptuneSwitchoverGlobalCluster
	handlers["CreateEventSubscription"] = (*Server).handleNeptuneCreateEventSubscription
	handlers["ModifyEventSubscription"] = (*Server).handleNeptuneModifyEventSubscription
	handlers["DeleteEventSubscription"] = (*Server).handleNeptuneDeleteEventSubscription
	handlers["DescribeEventSubscriptions"] = (*Server).handleNeptuneDescribeEventSubscriptions
	handlers["AddSourceIdentifierToSubscription"] = (*Server).handleNeptuneUpdateEventSubscriptionSources
	handlers["RemoveSourceIdentifierFromSubscription"] = (*Server).handleNeptuneUpdateEventSubscriptionSources
	handlers["PromoteReadReplicaDBCluster"] = (*Server).handleNeptuneCompatNoop
	handlers["RemoveFromGlobalCluster"] = (*Server).handleNeptuneRemoveFromGlobalCluster

	// Stage 6: tags/roles/pending maintenance and compatibility hardening.
	handlers["AddTagsToResource"] = (*Server).handleNeptuneAddTagsToResource
	handlers["RemoveTagsFromResource"] = (*Server).handleNeptuneRemoveTagsFromResource
	handlers["ApplyPendingMaintenanceAction"] = (*Server).handleNeptuneApplyPendingMaintenanceAction
	handlers["AddRoleToDBCluster"] = (*Server).handleNeptuneAddRoleToDBCluster
	handlers["RemoveRoleFromDBCluster"] = (*Server).handleNeptuneRemoveRoleFromDBCluster
	handlers["DescribeEngineDefaultClusterParameters"] = (*Server).handleNeptuneDescribeEngineDefaults
	handlers["DescribeEngineDefaultParameters"] = (*Server).handleNeptuneDescribeEngineDefaults
	handlers["DescribeEventCategories"] = (*Server).handleNeptuneDescribeEventCategories

	return handlers
}()

func (s *Server) handleNeptuneQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isNeptuneQueryCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "rds")
	if !ok {
		respondNeptuneErrorXML(w, status, code, msg)
		return true
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		respondNeptuneErrorXML(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}

	if err := r.ParseForm(); err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid form body")
		return true
	}

	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return true
	}
	if !isNeptuneAction(action) {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidAction", "unknown operation")
		return true
	}
	if version := strings.TrimSpace(r.Form.Get("Version")); version != "" && version != "2014-10-31" {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "unsupported Version")
		return true
	}

	handler, ok := neptuneQueryHandlers[action]
	if !ok {
		respondNeptuneErrorXML(w, http.StatusNotImplemented, "NotImplemented", "operation not implemented")
		return true
	}
	handler(s, w, r)
	return true
}

func isNeptuneQueryCandidate(r *http.Request) bool {
	if !hasNeptuneRequestHint(r) {
		return false
	}
	if service := strings.TrimSpace(sigV4ServiceHint(r)); service != "" && service != "rds" {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		return false
	}

	if action := strings.TrimSpace(r.URL.Query().Get("Action")); action != "" {
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
	return strings.TrimSpace(values.Get("Action")) != ""
}

func hasNeptuneRequestHint(r *http.Request) bool {
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".neptune.") || strings.HasPrefix(host, "neptune.") {
		return true
	}
	path := strings.TrimSpace(r.URL.Path)
	if strings.HasPrefix(path, "/neptune") {
		return true
	}
	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#neptune") || strings.Contains(userAgent, " neptune/") {
		return true
	}
	amzUserAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Amz-User-Agent")))
	return strings.Contains(amzUserAgent, "command#neptune") || strings.Contains(amzUserAgent, " neptune/")
}

func isNeptuneAction(action string) bool {
	_, ok := neptuneOperationByName[strings.TrimSpace(action)]
	return ok
}

func (s *Server) handleNeptuneNotImplemented(w http.ResponseWriter, _ *http.Request) {
	respondNeptuneErrorXML(w, http.StatusNotImplemented, "NotImplemented", "operation not implemented")
}

func (s *Server) handleNeptuneDescribeCompatNoop(w http.ResponseWriter, r *http.Request) {
	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return
	}
	respondNeptuneXML(w, action, neptuneDynamicResult{XMLName: xml.Name{Local: action + "Result"}})
}

func (s *Server) handleNeptuneDescribeDBClusters(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	clusters, marker, err := s.rds.DescribeDBClusters(rdssvc.DescribeDBClustersInput{
		Identifier: strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		clusters = nil
		err = nil
		marker = ""
	}
	if err != nil {
		respondNeptuneServiceError(w, "DescribeDBClusters", err)
		return
	}
	out := make([]rdsDBClusterXML, 0, len(clusters))
	for _, cluster := range clusters {
		out = append(out, rdsDBClusterToXML(cluster))
	}
	respondNeptuneXML(w, "DescribeDBClusters", struct {
		XMLName    xml.Name          `xml:"DescribeDBClustersResult"`
		Marker     string            `xml:"Marker,omitempty"`
		DBClusters []rdsDBClusterXML `xml:"DBClusters>DBCluster"`
	}{
		Marker:     marker,
		DBClusters: out,
	})
}

func (s *Server) handleNeptuneDescribeDBClusterEndpoints(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	endpoints, marker, err := s.rds.DescribeDBClusterEndpoints(rdssvc.DescribeDBClusterEndpointsInput{
		Identifier:        strings.TrimSpace(r.Form.Get("DBClusterEndpointIdentifier")),
		ClusterIdentifier: strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		MaxRecords:        maxRecords,
		Marker:            strings.TrimSpace(r.Form.Get("Marker")),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		endpoints = nil
		err = nil
		marker = ""
	}
	if err != nil {
		respondNeptuneServiceError(w, "DescribeDBClusterEndpoints", err)
		return
	}
	out := make([]rdsDBClusterEndpointXML, 0, len(endpoints))
	for _, endpoint := range endpoints {
		out = append(out, rdsDBClusterEndpointToXML(endpoint))
	}
	respondNeptuneXML(w, "DescribeDBClusterEndpoints", struct {
		XMLName            xml.Name                  `xml:"DescribeDBClusterEndpointsResult"`
		Marker             string                    `xml:"Marker,omitempty"`
		DBClusterEndpoints []rdsDBClusterEndpointXML `xml:"DBClusterEndpoints>DBClusterEndpoint"`
	}{
		Marker:             marker,
		DBClusterEndpoints: out,
	})
}

func (s *Server) handleNeptuneDescribeGlobalClusters(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	clusters, marker, err := s.rds.DescribeGlobalClusters(rdssvc.DescribeGlobalClustersInput{
		Identifier: strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		clusters = nil
		err = nil
		marker = ""
	}
	if err != nil {
		respondNeptuneServiceError(w, "DescribeGlobalClusters", err)
		return
	}
	out := make([]rdsGlobalClusterXML, 0, len(clusters))
	for _, cluster := range clusters {
		out = append(out, rdsGlobalClusterToXML(cluster))
	}
	respondNeptuneXML(w, "DescribeGlobalClusters", struct {
		XMLName        xml.Name              `xml:"DescribeGlobalClustersResult"`
		Marker         string                `xml:"Marker,omitempty"`
		GlobalClusters []rdsGlobalClusterXML `xml:"GlobalClusters>GlobalCluster"`
	}{
		Marker:         marker,
		GlobalClusters: out,
	})
}

func (s *Server) handleNeptuneDescribeDBInstances(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	instances, marker, err := s.rds.DescribeDBInstances(rdssvc.DescribeDBInstancesInput{
		Identifier: strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		instances = nil
		err = nil
		marker = ""
	}
	if err != nil {
		respondNeptuneServiceError(w, "DescribeDBInstances", err)
		return
	}
	out := make([]rdsDBInstanceXML, 0, len(instances))
	for _, instance := range instances {
		out = append(out, rdsDBInstanceToXML(instance))
	}
	respondNeptuneXML(w, "DescribeDBInstances", struct {
		XMLName     xml.Name           `xml:"DescribeDBInstancesResult"`
		Marker      string             `xml:"Marker,omitempty"`
		DBInstances []rdsDBInstanceXML `xml:"DBInstances>DBInstance"`
	}{
		Marker:      marker,
		DBInstances: out,
	})
}

func (s *Server) handleNeptuneDescribeDBEngineVersions(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeDBEngineVersions(rdssvc.DescribeDBEngineVersionsInput{
		Engine:        strings.TrimSpace(r.Form.Get("Engine")),
		EngineVersion: strings.TrimSpace(r.Form.Get("EngineVersion")),
		MaxRecords:    maxRecords,
		Marker:        strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondNeptuneServiceError(w, "DescribeDBEngineVersions", err)
		return
	}
	out := make([]rdsDBEngineVersionXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsDBEngineVersionToXML(item))
	}
	respondNeptuneXML(w, "DescribeDBEngineVersions", struct {
		XMLName          xml.Name                `xml:"DescribeDBEngineVersionsResult"`
		Marker           string                  `xml:"Marker,omitempty"`
		DBEngineVersions []rdsDBEngineVersionXML `xml:"DBEngineVersions>DBEngineVersion"`
	}{
		Marker:           marker,
		DBEngineVersions: out,
	})
}

func (s *Server) handleNeptuneDescribeOrderableDBInstanceOptions(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	vpc, err := parseOptionalRDSBoolPtr(r.Form.Get("Vpc"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Vpc is invalid")
		return
	}
	items, marker, err := s.rds.DescribeOrderableDBInstanceOptions(rdssvc.DescribeOrderableDBInstanceOptionsInput{
		Engine:          strings.TrimSpace(r.Form.Get("Engine")),
		EngineVersion:   strings.TrimSpace(r.Form.Get("EngineVersion")),
		DBInstanceClass: strings.TrimSpace(r.Form.Get("DBInstanceClass")),
		LicenseModel:    strings.TrimSpace(r.Form.Get("LicenseModel")),
		Vpc:             vpc,
		MaxRecords:      maxRecords,
		Marker:          strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondNeptuneServiceError(w, "DescribeOrderableDBInstanceOptions", err)
		return
	}
	out := make([]rdsOrderableDBInstanceOptionXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsOrderableDBInstanceOptionToXML(item))
	}
	respondNeptuneXML(w, "DescribeOrderableDBInstanceOptions", struct {
		XMLName                    xml.Name                          `xml:"DescribeOrderableDBInstanceOptionsResult"`
		Marker                     string                            `xml:"Marker,omitempty"`
		OrderableDBInstanceOptions []rdsOrderableDBInstanceOptionXML `xml:"OrderableDBInstanceOptions>OrderableDBInstanceOption"`
	}{
		Marker:                     marker,
		OrderableDBInstanceOptions: out,
	})
}

func (s *Server) handleNeptuneDescribePendingMaintenanceActions(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribePendingMaintenanceActions(rdssvc.DescribePendingMaintenanceActionsInput{
		ResourceIdentifier: strings.TrimSpace(r.Form.Get("ResourceIdentifier")),
		MaxRecords:         maxRecords,
		Marker:             strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondNeptuneServiceError(w, "DescribePendingMaintenanceActions", err)
		return
	}
	out := make([]rdsPendingMaintenanceActionXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsPendingMaintenanceActionToXML(item))
	}
	respondNeptuneXML(w, "DescribePendingMaintenanceActions", struct {
		XMLName                   xml.Name                         `xml:"DescribePendingMaintenanceActionsResult"`
		Marker                    string                           `xml:"Marker,omitempty"`
		PendingMaintenanceActions []rdsPendingMaintenanceActionXML `xml:"PendingMaintenanceActions>ResourcePendingMaintenanceActions"`
	}{
		Marker:                    marker,
		PendingMaintenanceActions: out,
	})
}

func (s *Server) handleNeptuneDescribeEvents(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	duration, err := parseOptionalRDSInt(r.Form.Get("Duration"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Duration is invalid")
		return
	}
	items, marker, err := s.rds.DescribeEvents(rdssvc.DescribeEventsInput{
		SourceIdentifier: strings.TrimSpace(r.Form.Get("SourceIdentifier")),
		SourceType:       strings.TrimSpace(r.Form.Get("SourceType")),
		Duration:         duration,
		MaxRecords:       maxRecords,
		Marker:           strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondNeptuneServiceError(w, "DescribeEvents", err)
		return
	}
	out := make([]rdsEventRecordXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsEventRecordToXML(item))
	}
	respondNeptuneXML(w, "DescribeEvents", struct {
		XMLName xml.Name            `xml:"DescribeEventsResult"`
		Marker  string              `xml:"Marker,omitempty"`
		Events  []rdsEventRecordXML `xml:"Events>Event"`
	}{
		Marker: marker,
		Events: out,
	})
}

func (s *Server) handleNeptuneListTagsForResource(w http.ResponseWriter, r *http.Request) {
	resource := strings.TrimSpace(firstNonEmpty(r.Form.Get("ResourceName"), r.Form.Get("ResourceArn"), r.Form.Get("ResourceARN")))
	tags, err := s.rds.ListTagsForResource(resource)
	if errors.Is(err, rdssvc.ErrNotFound) {
		tags = map[string]string{}
		err = nil
	}
	if err != nil {
		respondNeptuneServiceError(w, "ListTagsForResource", err)
		return
	}
	items := make([]rdsTagXML, 0, len(tags))
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		items = append(items, rdsTagXML{Key: key, Value: tags[key]})
	}
	respondNeptuneXML(w, "ListTagsForResource", struct {
		XMLName xml.Name    `xml:"ListTagsForResourceResult"`
		TagList []rdsTagXML `xml:"TagList>Tag"`
	}{
		TagList: items,
	})
}

func (s *Server) handleNeptuneCreateDBCluster(w http.ResponseWriter, r *http.Request) {
	backupRetention, err := parseOptionalRDSInt(r.Form.Get("BackupRetentionPeriod"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "BackupRetentionPeriod is invalid")
		return
	}
	cluster, err := s.rds.CreateDBCluster(rdssvc.CreateDBClusterInput{
		Identifier:              strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		Engine:                  firstNonEmpty(strings.TrimSpace(r.Form.Get("Engine")), "neptune"),
		MasterUsername:          firstNonEmpty(strings.TrimSpace(r.Form.Get("MasterUsername")), "admin"),
		MasterUserPassword:      firstNonEmpty(strings.TrimSpace(r.Form.Get("MasterUserPassword")), "Secret1234"),
		DatabaseName:            strings.TrimSpace(r.Form.Get("DatabaseName")),
		DBSubnetGroupName:       strings.TrimSpace(r.Form.Get("DBSubnetGroupName")),
		DBClusterParameterGroup: strings.TrimSpace(r.Form.Get("DBClusterParameterGroupName")),
		VpcSecurityGroupIDs:     parseRDSListMembers(r.Form, "VpcSecurityGroupIds.member"),
		BackupRetentionPeriod:   backupRetention,
	})
	if err != nil {
		respondNeptuneServiceError(w, "CreateDBCluster", err)
		return
	}
	respondNeptuneXML(w, "CreateDBCluster", struct {
		XMLName   xml.Name        `xml:"CreateDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneModifyDBCluster(w http.ResponseWriter, r *http.Request) {
	backupRetention, err := parseOptionalRDSInt(r.Form.Get("BackupRetentionPeriod"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "BackupRetentionPeriod is invalid")
		return
	}
	cluster, err := s.rds.ModifyDBCluster(rdssvc.ModifyDBClusterInput{
		Identifier:              strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		BackupRetentionPeriod:   backupRetention,
		DBClusterParameterGroup: strings.TrimSpace(r.Form.Get("DBClusterParameterGroupName")),
	})
	if err != nil {
		respondNeptuneServiceError(w, "ModifyDBCluster", err)
		return
	}
	respondNeptuneXML(w, "ModifyDBCluster", struct {
		XMLName   xml.Name        `xml:"ModifyDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneDeleteDBCluster(w http.ResponseWriter, r *http.Request) {
	skipFinalSnapshot, err := parseOptionalRDSBool(r.Form.Get("SkipFinalSnapshot"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "SkipFinalSnapshot is invalid")
		return
	}
	if strings.TrimSpace(r.Form.Get("SkipFinalSnapshot")) == "" {
		skipFinalSnapshot = true
	}
	cluster, err := s.rds.DeleteDBCluster(rdssvc.DeleteDBClusterInput{
		Identifier:                strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		SkipFinalSnapshot:         skipFinalSnapshot,
		FinalDBSnapshotIdentifier: strings.TrimSpace(r.Form.Get("FinalDBSnapshotIdentifier")),
	})
	if err != nil {
		respondNeptuneServiceError(w, "DeleteDBCluster", err)
		return
	}
	respondNeptuneXML(w, "DeleteDBCluster", struct {
		XMLName   xml.Name        `xml:"DeleteDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneStartDBCluster(w http.ResponseWriter, r *http.Request) {
	cluster, err := s.rds.StartDBCluster(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")))
	if err != nil {
		respondNeptuneServiceError(w, "StartDBCluster", err)
		return
	}
	respondNeptuneXML(w, "StartDBCluster", struct {
		XMLName   xml.Name        `xml:"StartDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneStopDBCluster(w http.ResponseWriter, r *http.Request) {
	cluster, err := s.rds.StopDBCluster(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")))
	if err != nil {
		respondNeptuneServiceError(w, "StopDBCluster", err)
		return
	}
	respondNeptuneXML(w, "StopDBCluster", struct {
		XMLName   xml.Name        `xml:"StopDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneFailoverDBCluster(w http.ResponseWriter, r *http.Request) {
	cluster, err := s.rds.FailoverDBCluster(rdssvc.FailoverDBClusterInput{
		Identifier:                 strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		TargetDBInstanceIdentifier: strings.TrimSpace(r.Form.Get("TargetDBInstanceIdentifier")),
	})
	if err != nil {
		respondNeptuneServiceError(w, "FailoverDBCluster", err)
		return
	}
	respondNeptuneXML(w, "FailoverDBCluster", struct {
		XMLName   xml.Name        `xml:"FailoverDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneCompatNoop(w http.ResponseWriter, r *http.Request) {
	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return
	}
	respondNeptuneXML(w, action, neptuneDynamicResult{XMLName: xml.Name{Local: action + "Result"}})
}

func (s *Server) handleNeptuneDescribeEngineDefaults(w http.ResponseWriter, r *http.Request) {
	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return
	}
	family := strings.TrimSpace(firstNonEmpty(r.Form.Get("DBParameterGroupFamily"), r.Form.Get("DBClusterParameterGroupFamily")))
	if family == "" {
		family = "neptune1"
	}
	respondNeptuneXML(w, action, struct {
		XMLName        xml.Name             `xml:""`
		EngineDefaults rdsEngineDefaultsXML `xml:"EngineDefaults"`
	}{
		XMLName: xml.Name{Local: action + "Result"},
		EngineDefaults: rdsEngineDefaultsXML{
			DBParameterGroupFamily: family,
			Parameters: []rdsParameterXML{
				{
					ParameterName:  "neptune_query_timeout",
					ParameterValue: "120000",
					ApplyType:      "dynamic",
					ApplyMethod:    "immediate",
					Source:         "engine-default",
					IsModifiable:   true,
				},
			},
		},
	})
}

func (s *Server) handleNeptuneDescribeEventCategories(w http.ResponseWriter, r *http.Request) {
	sourceType := strings.TrimSpace(r.Form.Get("SourceType"))
	if sourceType == "" {
		sourceType = "db-instance"
	}
	respondNeptuneXML(w, "DescribeEventCategories", struct {
		XMLName                xml.Name                   `xml:"DescribeEventCategoriesResult"`
		EventCategoriesMapList []rdsEventCategoriesMapXML `xml:"EventCategoriesMapList>EventCategoriesMap"`
	}{
		EventCategoriesMapList: []rdsEventCategoriesMapXML{
			{
				SourceType:      sourceType,
				EventCategories: []string{"availability", "backup"},
			},
		},
	})
}

func (s *Server) handleNeptuneUpdateEventSubscriptionSources(w http.ResponseWriter, r *http.Request) {
	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return
	}
	sourceID := strings.TrimSpace(firstNonEmpty(r.Form.Get("SourceIdentifier"), r.Form.Get("SourceId")))
	respondNeptuneXML(w, action, struct {
		XMLName           xml.Name                `xml:""`
		EventSubscription rdsEventSubscriptionXML `xml:"EventSubscription"`
	}{
		XMLName: xml.Name{Local: action + "Result"},
		EventSubscription: rdsEventSubscriptionXML{
			CustSubscriptionId:       firstNonEmpty(strings.TrimSpace(r.Form.Get("SubscriptionName")), "stackyard-neptune-subscription"),
			SnsTopicArn:              firstNonEmpty(strings.TrimSpace(r.Form.Get("SnsTopicArn")), "arn:aws:sns:us-east-1:123456789012:stackyard-neptune-topic"),
			SourceType:               firstNonEmpty(strings.TrimSpace(r.Form.Get("SourceType")), "db-cluster"),
			SourceIdsList:            []string{sourceID},
			EventCategoriesList:      []string{"availability"},
			Enabled:                  true,
			Status:                   "active",
			EventSubscriptionArn:     "arn:aws:rds:us-east-1:123456789012:es:stackyard-neptune-subscription",
			SubscriptionCreationTime: "2026-01-01T00:00:00Z",
		},
	})
}

func (s *Server) handleNeptuneRemoveFromGlobalCluster(w http.ResponseWriter, r *http.Request) {
	globalClusterIdentifier := firstNonEmpty(strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")), "stackyard-neptune-global-cluster")
	dbClusterIdentifier := firstNonEmpty(strings.TrimSpace(r.Form.Get("DbClusterIdentifier")), strings.TrimSpace(r.Form.Get("DBClusterIdentifier")), "stackyard-neptune-cluster")
	respondNeptuneXML(w, "RemoveFromGlobalCluster", struct {
		XMLName       xml.Name            `xml:"RemoveFromGlobalClusterResult"`
		GlobalCluster rdsGlobalClusterXML `xml:"GlobalCluster"`
	}{
		GlobalCluster: rdsGlobalClusterXML{
			GlobalClusterIdentifier: globalClusterIdentifier,
			GlobalClusterArn:        "arn:aws:rds:us-east-1:123456789012:global-cluster:" + globalClusterIdentifier,
			Status:                  "available",
			GlobalClusterMembers: []rdsGlobalClusterMemberXML{
				{
					DBClusterArn: "arn:aws:rds:us-east-1:123456789012:cluster:" + dbClusterIdentifier,
					IsWriter:     false,
				},
			},
		},
	})
}

func (s *Server) handleNeptuneCreateDBInstance(w http.ResponseWriter, r *http.Request) {
	allocatedStorage, err := parseOptionalRDSInt(r.Form.Get("AllocatedStorage"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "AllocatedStorage is invalid")
		return
	}
	if allocatedStorage <= 0 {
		allocatedStorage = 20
	}
	backupRetention, err := parseOptionalRDSInt(r.Form.Get("BackupRetentionPeriod"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "BackupRetentionPeriod is invalid")
		return
	}
	publiclyAccessible, err := parseOptionalRDSBool(r.Form.Get("PubliclyAccessible"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "PubliclyAccessible is invalid")
		return
	}
	identifier := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")), "stackyard-neptune-instance")
	instance, err := s.rds.CreateDBInstance(rdssvc.CreateDBInstanceInput{
		Identifier:            identifier,
		Engine:                firstNonEmpty(strings.TrimSpace(r.Form.Get("Engine")), "neptune"),
		DBInstanceClass:       firstNonEmpty(strings.TrimSpace(r.Form.Get("DBInstanceClass")), "db.r5.large"),
		AllocatedStorage:      allocatedStorage,
		MasterUsername:        firstNonEmpty(strings.TrimSpace(r.Form.Get("MasterUsername")), "admin"),
		MasterUserPassword:    firstNonEmpty(strings.TrimSpace(r.Form.Get("MasterUserPassword")), "Secret1234"),
		DBName:                strings.TrimSpace(r.Form.Get("DBName")),
		BackupRetentionPeriod: backupRetention,
		PubliclyAccessible:    publiclyAccessible,
		DBSubnetGroupName:     strings.TrimSpace(r.Form.Get("DBSubnetGroupName")),
		DBParameterGroupName:  strings.TrimSpace(r.Form.Get("DBParameterGroupName")),
		OptionGroupName:       strings.TrimSpace(r.Form.Get("OptionGroupName")),
	})
	if errors.Is(err, rdssvc.ErrAlreadyExists) {
		items, _, describeErr := s.rds.DescribeDBInstances(rdssvc.DescribeDBInstancesInput{Identifier: identifier})
		if describeErr == nil && len(items) > 0 {
			instance = items[0]
			err = nil
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "CreateDBInstance", err)
		return
	}
	respondNeptuneXML(w, "CreateDBInstance", struct {
		XMLName    xml.Name         `xml:"CreateDBInstanceResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleNeptuneModifyDBInstance(w http.ResponseWriter, r *http.Request) {
	allocatedStorage, err := parseOptionalRDSInt(r.Form.Get("AllocatedStorage"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "AllocatedStorage is invalid")
		return
	}
	backupRetention, err := parseOptionalRDSInt(r.Form.Get("BackupRetentionPeriod"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "BackupRetentionPeriod is invalid")
		return
	}
	publiclyAccessible, err := parseOptionalRDSBoolPtr(r.Form.Get("PubliclyAccessible"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "PubliclyAccessible is invalid")
		return
	}
	identifier := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")), "stackyard-neptune-instance")
	createAllocatedStorage := allocatedStorage
	if createAllocatedStorage < 20 {
		createAllocatedStorage = 20
	}
	instance, err := s.rds.ModifyDBInstance(rdssvc.ModifyDBInstanceInput{
		Identifier:            identifier,
		DBInstanceClass:       strings.TrimSpace(r.Form.Get("DBInstanceClass")),
		AllocatedStorage:      allocatedStorage,
		BackupRetentionPeriod: backupRetention,
		PubliclyAccessible:    publiclyAccessible,
		ApplyImmediately:      true,
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateDBInstance(rdssvc.CreateDBInstanceInput{
			Identifier:           identifier,
			Engine:               "neptune",
			DBInstanceClass:      firstNonEmpty(strings.TrimSpace(r.Form.Get("DBInstanceClass")), "db.r5.large"),
			AllocatedStorage:     createAllocatedStorage,
			MasterUsername:       "admin",
			MasterUserPassword:   "Secret1234",
			PubliclyAccessible:   boolValue(publiclyAccessible),
			DBSubnetGroupName:    strings.TrimSpace(r.Form.Get("DBSubnetGroupName")),
			DBParameterGroupName: strings.TrimSpace(r.Form.Get("DBParameterGroupName")),
		}); createErr == nil {
			instance, err = s.rds.ModifyDBInstance(rdssvc.ModifyDBInstanceInput{
				Identifier:            identifier,
				DBInstanceClass:       strings.TrimSpace(r.Form.Get("DBInstanceClass")),
				AllocatedStorage:      allocatedStorage,
				BackupRetentionPeriod: backupRetention,
				PubliclyAccessible:    publiclyAccessible,
				ApplyImmediately:      true,
			})
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "ModifyDBInstance", err)
		return
	}
	respondNeptuneXML(w, "ModifyDBInstance", struct {
		XMLName    xml.Name         `xml:"ModifyDBInstanceResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleNeptuneRebootDBInstance(w http.ResponseWriter, r *http.Request) {
	identifier := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")), "stackyard-neptune-instance")
	instance, err := s.rds.RebootDBInstance(identifier)
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateDBInstance(rdssvc.CreateDBInstanceInput{
			Identifier:         identifier,
			Engine:             "neptune",
			DBInstanceClass:    "db.r5.large",
			AllocatedStorage:   20,
			MasterUsername:     "admin",
			MasterUserPassword: "Secret1234",
		}); createErr == nil {
			instance, err = s.rds.RebootDBInstance(identifier)
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "RebootDBInstance", err)
		return
	}
	respondNeptuneXML(w, "RebootDBInstance", struct {
		XMLName    xml.Name         `xml:"RebootDBInstanceResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleNeptuneDeleteDBInstance(w http.ResponseWriter, r *http.Request) {
	skipFinalSnapshot, err := parseOptionalRDSBool(r.Form.Get("SkipFinalSnapshot"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "SkipFinalSnapshot is invalid")
		return
	}
	if strings.TrimSpace(r.Form.Get("SkipFinalSnapshot")) == "" {
		skipFinalSnapshot = true
	}
	identifier := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")), "stackyard-neptune-instance")
	instance, err := s.rds.DeleteDBInstance(rdssvc.DeleteDBInstanceInput{
		Identifier:                identifier,
		SkipFinalSnapshot:         skipFinalSnapshot,
		FinalDBSnapshotIdentifier: strings.TrimSpace(r.Form.Get("FinalDBSnapshotIdentifier")),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		instance = rdssvc.DBInstance{
			Identifier: identifier,
			ARN:        "arn:aws:rds:us-east-1:123456789012:db:" + identifier,
			Engine:     "neptune",
			Status:     "deleting",
		}
		err = nil
	}
	if err != nil {
		respondNeptuneServiceError(w, "DeleteDBInstance", err)
		return
	}
	respondNeptuneXML(w, "DeleteDBInstance", struct {
		XMLName    xml.Name         `xml:"DeleteDBInstanceResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleNeptuneCreateDBParameterGroup(w http.ResponseWriter, r *http.Request) {
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBParameterGroupName")), "stackyard-neptune-param-group")
	group, err := s.rds.CreateDBParameterGroup(rdssvc.CreateDBParameterGroupInput{
		Name:        name,
		Family:      firstNonEmpty(strings.TrimSpace(r.Form.Get("DBParameterGroupFamily")), "neptune1"),
		Description: firstNonEmpty(strings.TrimSpace(r.Form.Get("Description")), "stackyard neptune parameter group"),
	})
	if errors.Is(err, rdssvc.ErrAlreadyExists) {
		items, _, describeErr := s.rds.DescribeDBParameterGroups(rdssvc.DescribeDBParameterGroupsInput{Name: name})
		if describeErr == nil && len(items) > 0 {
			group = items[0]
			err = nil
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "CreateDBParameterGroup", err)
		return
	}
	respondNeptuneXML(w, "CreateDBParameterGroup", struct {
		XMLName          xml.Name               `xml:"CreateDBParameterGroupResult"`
		DBParameterGroup rdsDBParameterGroupXML `xml:"DBParameterGroup"`
	}{
		DBParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleNeptuneDescribeDBParameterGroups(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeDBParameterGroups(rdssvc.DescribeDBParameterGroupsInput{
		Name:       strings.TrimSpace(r.Form.Get("DBParameterGroupName")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		items = nil
		err = nil
		marker = ""
	}
	if err != nil {
		respondNeptuneServiceError(w, "DescribeDBParameterGroups", err)
		return
	}
	out := make([]rdsDBParameterGroupXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsDBParameterGroupToXML(item))
	}
	respondNeptuneXML(w, "DescribeDBParameterGroups", struct {
		XMLName           xml.Name                 `xml:"DescribeDBParameterGroupsResult"`
		Marker            string                   `xml:"Marker,omitempty"`
		DBParameterGroups []rdsDBParameterGroupXML `xml:"DBParameterGroups>DBParameterGroup"`
	}{
		Marker:            marker,
		DBParameterGroups: out,
	})
}

func (s *Server) handleNeptuneDescribeDBParameters(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeDBParameters(rdssvc.DescribeDBParametersInput{
		GroupName:  strings.TrimSpace(r.Form.Get("DBParameterGroupName")),
		Source:     strings.TrimSpace(r.Form.Get("Source")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		items = nil
		err = nil
		marker = ""
	}
	if err != nil {
		respondNeptuneServiceError(w, "DescribeDBParameters", err)
		return
	}
	out := make([]rdsParameterXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsParameterToXML(item))
	}
	respondNeptuneXML(w, "DescribeDBParameters", struct {
		XMLName    xml.Name          `xml:"DescribeDBParametersResult"`
		Marker     string            `xml:"Marker,omitempty"`
		Parameters []rdsParameterXML `xml:"Parameters>Parameter"`
	}{
		Marker:     marker,
		Parameters: out,
	})
}

func (s *Server) handleNeptuneModifyDBParameterGroup(w http.ResponseWriter, r *http.Request) {
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBParameterGroupName")), "stackyard-neptune-param-group")
	params := parseRDSParameterMembers(r.Form, "Parameters.member")
	if len(params) == 0 {
		params = parseRDSParameterMembers(r.Form, "Parameters.Parameter")
	}
	if len(params) == 0 {
		params = []rdssvc.Parameter{{Name: "neptune_query_timeout", Value: "120000", ApplyMethod: "immediate"}}
	}
	group, err := s.rds.ModifyDBParameterGroup(rdssvc.ModifyDBParameterGroupInput{Name: name, Parameters: params})
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateDBParameterGroup(rdssvc.CreateDBParameterGroupInput{
			Name:        name,
			Family:      "neptune1",
			Description: "stackyard neptune parameter group",
		}); createErr == nil {
			group, err = s.rds.ModifyDBParameterGroup(rdssvc.ModifyDBParameterGroupInput{Name: name, Parameters: params})
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "ModifyDBParameterGroup", err)
		return
	}
	respondNeptuneXML(w, "ModifyDBParameterGroup", struct {
		XMLName          xml.Name               `xml:"ModifyDBParameterGroupResult"`
		DBParameterGroup rdsDBParameterGroupXML `xml:"DBParameterGroup"`
	}{
		DBParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleNeptuneResetDBParameterGroup(w http.ResponseWriter, r *http.Request) {
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBParameterGroupName")), "stackyard-neptune-param-group")
	resetAll, err := parseOptionalRDSBool(r.Form.Get("ResetAllParameters"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "ResetAllParameters is invalid")
		return
	}
	paramNames := parseRDSParameterNames(r.Form, "Parameters.member")
	if len(paramNames) == 0 {
		paramNames = parseRDSParameterNames(r.Form, "Parameters.Parameter")
	}
	group, err := s.rds.ResetDBParameterGroup(rdssvc.ResetDBParameterGroupInput{
		Name:                  name,
		ResetAllParameters:    resetAll,
		ParameterNamesToReset: paramNames,
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateDBParameterGroup(rdssvc.CreateDBParameterGroupInput{
			Name:        name,
			Family:      "neptune1",
			Description: "stackyard neptune parameter group",
		}); createErr == nil {
			group, err = s.rds.ResetDBParameterGroup(rdssvc.ResetDBParameterGroupInput{
				Name:                  name,
				ResetAllParameters:    resetAll,
				ParameterNamesToReset: paramNames,
			})
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "ResetDBParameterGroup", err)
		return
	}
	respondNeptuneXML(w, "ResetDBParameterGroup", struct {
		XMLName          xml.Name               `xml:"ResetDBParameterGroupResult"`
		DBParameterGroup rdsDBParameterGroupXML `xml:"DBParameterGroup"`
	}{
		DBParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleNeptuneDeleteDBParameterGroup(w http.ResponseWriter, r *http.Request) {
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBParameterGroupName")), "stackyard-neptune-param-group")
	group, err := s.rds.DeleteDBParameterGroup(name)
	if errors.Is(err, rdssvc.ErrNotFound) {
		group = rdssvc.DBParameterGroup{Name: name, Family: "neptune1", Description: "stackyard neptune parameter group"}
		err = nil
	}
	if err != nil {
		respondNeptuneServiceError(w, "DeleteDBParameterGroup", err)
		return
	}
	respondNeptuneXML(w, "DeleteDBParameterGroup", struct {
		XMLName          xml.Name               `xml:"DeleteDBParameterGroupResult"`
		DBParameterGroup rdsDBParameterGroupXML `xml:"DBParameterGroup"`
	}{
		DBParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleNeptuneCreateDBClusterParameterGroup(w http.ResponseWriter, r *http.Request) {
	groupName := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterParameterGroupName")), "stackyard-neptune-cluster-param-group")
	group, err := s.rds.CreateDBParameterGroup(rdssvc.CreateDBParameterGroupInput{
		Name:        groupName,
		Family:      firstNonEmpty(strings.TrimSpace(r.Form.Get("DBParameterGroupFamily")), strings.TrimSpace(r.Form.Get("DBClusterParameterGroupFamily")), "neptune1"),
		Description: firstNonEmpty(strings.TrimSpace(r.Form.Get("Description")), "stackyard neptune cluster parameter group"),
	})
	if errors.Is(err, rdssvc.ErrAlreadyExists) {
		items, _, describeErr := s.rds.DescribeDBParameterGroups(rdssvc.DescribeDBParameterGroupsInput{Name: groupName})
		if describeErr == nil && len(items) > 0 {
			group = items[0]
			err = nil
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "CreateDBClusterParameterGroup", err)
		return
	}
	respondNeptuneXML(w, "CreateDBClusterParameterGroup", struct {
		XMLName                 xml.Name               `xml:"CreateDBClusterParameterGroupResult"`
		DBClusterParameterGroup rdsDBParameterGroupXML `xml:"DBClusterParameterGroup"`
	}{
		DBClusterParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleNeptuneDescribeDBClusterParameterGroups(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeDBParameterGroups(rdssvc.DescribeDBParameterGroupsInput{
		Name:       strings.TrimSpace(r.Form.Get("DBClusterParameterGroupName")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		items = nil
		err = nil
		marker = ""
	}
	if err != nil {
		respondNeptuneServiceError(w, "DescribeDBClusterParameterGroups", err)
		return
	}
	out := make([]rdsDBParameterGroupXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsDBParameterGroupToXML(item))
	}
	respondNeptuneXML(w, "DescribeDBClusterParameterGroups", struct {
		XMLName                  xml.Name                 `xml:"DescribeDBClusterParameterGroupsResult"`
		Marker                   string                   `xml:"Marker,omitempty"`
		DBClusterParameterGroups []rdsDBParameterGroupXML `xml:"DBClusterParameterGroups>DBClusterParameterGroup"`
	}{
		Marker:                   marker,
		DBClusterParameterGroups: out,
	})
}

func (s *Server) handleNeptuneDescribeDBClusterParameters(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeDBParameters(rdssvc.DescribeDBParametersInput{
		GroupName:  strings.TrimSpace(r.Form.Get("DBClusterParameterGroupName")),
		Source:     strings.TrimSpace(r.Form.Get("Source")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		items = nil
		err = nil
		marker = ""
	}
	if err != nil {
		respondNeptuneServiceError(w, "DescribeDBClusterParameters", err)
		return
	}
	out := make([]rdsParameterXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsParameterToXML(item))
	}
	respondNeptuneXML(w, "DescribeDBClusterParameters", struct {
		XMLName    xml.Name          `xml:"DescribeDBClusterParametersResult"`
		Marker     string            `xml:"Marker,omitempty"`
		Parameters []rdsParameterXML `xml:"Parameters>Parameter"`
	}{
		Marker:     marker,
		Parameters: out,
	})
}

func (s *Server) handleNeptuneModifyDBClusterParameterGroup(w http.ResponseWriter, r *http.Request) {
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterParameterGroupName")), "stackyard-neptune-cluster-param-group")
	params := parseRDSParameterMembers(r.Form, "Parameters.member")
	if len(params) == 0 {
		params = parseRDSParameterMembers(r.Form, "Parameters.Parameter")
	}
	if len(params) == 0 {
		params = []rdssvc.Parameter{{Name: "neptune_query_timeout", Value: "120000", ApplyMethod: "immediate"}}
	}
	group, err := s.rds.ModifyDBParameterGroup(rdssvc.ModifyDBParameterGroupInput{Name: name, Parameters: params})
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateDBParameterGroup(rdssvc.CreateDBParameterGroupInput{
			Name:        name,
			Family:      "neptune1",
			Description: "stackyard neptune cluster parameter group",
		}); createErr == nil {
			group, err = s.rds.ModifyDBParameterGroup(rdssvc.ModifyDBParameterGroupInput{Name: name, Parameters: params})
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "ModifyDBClusterParameterGroup", err)
		return
	}
	respondNeptuneXML(w, "ModifyDBClusterParameterGroup", struct {
		XMLName                 xml.Name               `xml:"ModifyDBClusterParameterGroupResult"`
		DBClusterParameterGroup rdsDBParameterGroupXML `xml:"DBClusterParameterGroup"`
	}{
		DBClusterParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleNeptuneResetDBClusterParameterGroup(w http.ResponseWriter, r *http.Request) {
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterParameterGroupName")), "stackyard-neptune-cluster-param-group")
	resetAll, err := parseOptionalRDSBool(r.Form.Get("ResetAllParameters"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "ResetAllParameters is invalid")
		return
	}
	paramNames := parseRDSParameterNames(r.Form, "Parameters.member")
	if len(paramNames) == 0 {
		paramNames = parseRDSParameterNames(r.Form, "Parameters.Parameter")
	}
	group, err := s.rds.ResetDBParameterGroup(rdssvc.ResetDBParameterGroupInput{
		Name:                  name,
		ResetAllParameters:    resetAll,
		ParameterNamesToReset: paramNames,
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateDBParameterGroup(rdssvc.CreateDBParameterGroupInput{
			Name:        name,
			Family:      "neptune1",
			Description: "stackyard neptune cluster parameter group",
		}); createErr == nil {
			group, err = s.rds.ResetDBParameterGroup(rdssvc.ResetDBParameterGroupInput{
				Name:                  name,
				ResetAllParameters:    resetAll,
				ParameterNamesToReset: paramNames,
			})
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "ResetDBClusterParameterGroup", err)
		return
	}
	respondNeptuneXML(w, "ResetDBClusterParameterGroup", struct {
		XMLName                 xml.Name               `xml:"ResetDBClusterParameterGroupResult"`
		DBClusterParameterGroup rdsDBParameterGroupXML `xml:"DBClusterParameterGroup"`
	}{
		DBClusterParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleNeptuneDeleteDBClusterParameterGroup(w http.ResponseWriter, r *http.Request) {
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterParameterGroupName")), "stackyard-neptune-cluster-param-group")
	group, err := s.rds.DeleteDBParameterGroup(name)
	if errors.Is(err, rdssvc.ErrNotFound) {
		group = rdssvc.DBParameterGroup{Name: name, Family: "neptune1", Description: "stackyard neptune cluster parameter group"}
		err = nil
	}
	if err != nil {
		respondNeptuneServiceError(w, "DeleteDBClusterParameterGroup", err)
		return
	}
	respondNeptuneXML(w, "DeleteDBClusterParameterGroup", struct {
		XMLName                 xml.Name               `xml:"DeleteDBClusterParameterGroupResult"`
		DBClusterParameterGroup rdsDBParameterGroupXML `xml:"DBClusterParameterGroup"`
	}{
		DBClusterParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleNeptuneCopyDBParameterGroup(w http.ResponseWriter, r *http.Request) {
	source := strings.TrimSpace(firstNonEmpty(r.Form.Get("SourceDBParameterGroupIdentifier"), r.Form.Get("SourceDBParameterGroupName")))
	target := firstNonEmpty(strings.TrimSpace(r.Form.Get("TargetDBParameterGroupIdentifier")), strings.TrimSpace(r.Form.Get("TargetDBParameterGroupName")), "stackyard-neptune-param-group-copy")
	items, _, _ := s.rds.DescribeDBParameterGroups(rdssvc.DescribeDBParameterGroupsInput{Name: source})
	family := "neptune1"
	description := "copied neptune parameter group"
	if len(items) > 0 {
		family = firstNonEmpty(items[0].Family, family)
		description = firstNonEmpty(items[0].Description, description)
	}
	group, err := s.rds.CreateDBParameterGroup(rdssvc.CreateDBParameterGroupInput{
		Name:        target,
		Family:      family,
		Description: description,
	})
	if errors.Is(err, rdssvc.ErrAlreadyExists) {
		existing, _, describeErr := s.rds.DescribeDBParameterGroups(rdssvc.DescribeDBParameterGroupsInput{Name: target})
		if describeErr == nil && len(existing) > 0 {
			group = existing[0]
			err = nil
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "CopyDBParameterGroup", err)
		return
	}
	respondNeptuneXML(w, "CopyDBParameterGroup", struct {
		XMLName          xml.Name               `xml:"CopyDBParameterGroupResult"`
		DBParameterGroup rdsDBParameterGroupXML `xml:"DBParameterGroup"`
	}{
		DBParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleNeptuneCopyDBClusterParameterGroup(w http.ResponseWriter, r *http.Request) {
	source := strings.TrimSpace(firstNonEmpty(r.Form.Get("SourceDBClusterParameterGroupIdentifier"), r.Form.Get("SourceDBClusterParameterGroupName")))
	target := firstNonEmpty(strings.TrimSpace(r.Form.Get("TargetDBClusterParameterGroupIdentifier")), strings.TrimSpace(r.Form.Get("TargetDBClusterParameterGroupName")), "stackyard-neptune-cluster-param-group-copy")
	items, _, _ := s.rds.DescribeDBParameterGroups(rdssvc.DescribeDBParameterGroupsInput{Name: source})
	family := "neptune1"
	description := "copied neptune cluster parameter group"
	if len(items) > 0 {
		family = firstNonEmpty(items[0].Family, family)
		description = firstNonEmpty(items[0].Description, description)
	}
	group, err := s.rds.CreateDBParameterGroup(rdssvc.CreateDBParameterGroupInput{
		Name:        target,
		Family:      family,
		Description: description,
	})
	if errors.Is(err, rdssvc.ErrAlreadyExists) {
		existing, _, describeErr := s.rds.DescribeDBParameterGroups(rdssvc.DescribeDBParameterGroupsInput{Name: target})
		if describeErr == nil && len(existing) > 0 {
			group = existing[0]
			err = nil
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "CopyDBClusterParameterGroup", err)
		return
	}
	respondNeptuneXML(w, "CopyDBClusterParameterGroup", struct {
		XMLName                 xml.Name               `xml:"CopyDBClusterParameterGroupResult"`
		DBClusterParameterGroup rdsDBParameterGroupXML `xml:"DBClusterParameterGroup"`
	}{
		DBClusterParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleNeptuneCreateDBSubnetGroup(w http.ResponseWriter, r *http.Request) {
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBSubnetGroupName")), "stackyard-neptune-subnet-group")
	subnetIDs := parseRDSListMembers(r.Form, "SubnetIds.member")
	if len(subnetIDs) == 0 {
		subnetIDs = []string{"subnet-12345678"}
	}
	group, err := s.rds.CreateDBSubnetGroup(rdssvc.CreateDBSubnetGroupInput{
		Name:        name,
		Description: firstNonEmpty(strings.TrimSpace(r.Form.Get("DBSubnetGroupDescription")), "stackyard neptune subnet group"),
		SubnetIDs:   subnetIDs,
	})
	if errors.Is(err, rdssvc.ErrAlreadyExists) {
		items, _, describeErr := s.rds.DescribeDBSubnetGroups(rdssvc.DescribeDBSubnetGroupsInput{Name: name})
		if describeErr == nil && len(items) > 0 {
			group = items[0]
			err = nil
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "CreateDBSubnetGroup", err)
		return
	}
	respondNeptuneXML(w, "CreateDBSubnetGroup", struct {
		XMLName       xml.Name                `xml:"CreateDBSubnetGroupResult"`
		DBSubnetGroup rdsDBSubnetGroupItemXML `xml:"DBSubnetGroup"`
	}{
		DBSubnetGroup: rdsDBSubnetGroupToXML(group),
	})
}

func (s *Server) handleNeptuneDescribeDBSubnetGroups(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeDBSubnetGroups(rdssvc.DescribeDBSubnetGroupsInput{
		Name:       strings.TrimSpace(r.Form.Get("DBSubnetGroupName")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		items = nil
		err = nil
		marker = ""
	}
	if err != nil {
		respondNeptuneServiceError(w, "DescribeDBSubnetGroups", err)
		return
	}
	out := make([]rdsDBSubnetGroupItemXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsDBSubnetGroupToXML(item))
	}
	respondNeptuneXML(w, "DescribeDBSubnetGroups", struct {
		XMLName        xml.Name                  `xml:"DescribeDBSubnetGroupsResult"`
		Marker         string                    `xml:"Marker,omitempty"`
		DBSubnetGroups []rdsDBSubnetGroupItemXML `xml:"DBSubnetGroups>DBSubnetGroup"`
	}{
		Marker:         marker,
		DBSubnetGroups: out,
	})
}

func (s *Server) handleNeptuneModifyDBSubnetGroup(w http.ResponseWriter, r *http.Request) {
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBSubnetGroupName")), "stackyard-neptune-subnet-group")
	subnetIDs := parseRDSListMembers(r.Form, "SubnetIds.member")
	group, err := s.rds.ModifyDBSubnetGroup(rdssvc.ModifyDBSubnetGroupInput{
		Name:        name,
		Description: strings.TrimSpace(r.Form.Get("DBSubnetGroupDescription")),
		SubnetIDs:   subnetIDs,
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateDBSubnetGroup(rdssvc.CreateDBSubnetGroupInput{
			Name:        name,
			Description: "stackyard neptune subnet group",
			SubnetIDs:   []string{"subnet-12345678"},
		}); createErr == nil {
			group, err = s.rds.ModifyDBSubnetGroup(rdssvc.ModifyDBSubnetGroupInput{
				Name:        name,
				Description: strings.TrimSpace(r.Form.Get("DBSubnetGroupDescription")),
				SubnetIDs:   subnetIDs,
			})
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "ModifyDBSubnetGroup", err)
		return
	}
	respondNeptuneXML(w, "ModifyDBSubnetGroup", struct {
		XMLName       xml.Name                `xml:"ModifyDBSubnetGroupResult"`
		DBSubnetGroup rdsDBSubnetGroupItemXML `xml:"DBSubnetGroup"`
	}{
		DBSubnetGroup: rdsDBSubnetGroupToXML(group),
	})
}

func (s *Server) handleNeptuneDeleteDBSubnetGroup(w http.ResponseWriter, r *http.Request) {
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBSubnetGroupName")), "stackyard-neptune-subnet-group")
	group, err := s.rds.DeleteDBSubnetGroup(name)
	if errors.Is(err, rdssvc.ErrNotFound) {
		group = rdssvc.DBSubnetGroup{Name: name, Description: "stackyard neptune subnet group", Status: "Complete"}
		err = nil
	}
	if err != nil {
		respondNeptuneServiceError(w, "DeleteDBSubnetGroup", err)
		return
	}
	respondNeptuneXML(w, "DeleteDBSubnetGroup", struct {
		XMLName       xml.Name                `xml:"DeleteDBSubnetGroupResult"`
		DBSubnetGroup rdsDBSubnetGroupItemXML `xml:"DBSubnetGroup"`
	}{
		DBSubnetGroup: rdsDBSubnetGroupToXML(group),
	})
}

func (s *Server) handleNeptuneCreateDBClusterSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshotID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterSnapshotIdentifier")), "stackyard-neptune-cluster-snapshot")
	clusterID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")), "stackyard-neptune-cluster")
	respondNeptuneXML(w, "CreateDBClusterSnapshot", struct {
		XMLName           xml.Name                    `xml:"CreateDBClusterSnapshotResult"`
		DBClusterSnapshot neptuneDBClusterSnapshotXML `xml:"DBClusterSnapshot"`
	}{
		DBClusterSnapshot: neptuneDBClusterSnapshotXML{
			DBClusterSnapshotIdentifier: snapshotID,
			DBClusterIdentifier:         clusterID,
			Status:                      "available",
			Engine:                      "neptune",
		},
	})
}

func (s *Server) handleNeptuneDeleteDBClusterSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshotID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterSnapshotIdentifier")), "stackyard-neptune-cluster-snapshot")
	respondNeptuneXML(w, "DeleteDBClusterSnapshot", struct {
		XMLName           xml.Name                    `xml:"DeleteDBClusterSnapshotResult"`
		DBClusterSnapshot neptuneDBClusterSnapshotXML `xml:"DBClusterSnapshot"`
	}{
		DBClusterSnapshot: neptuneDBClusterSnapshotXML{
			DBClusterSnapshotIdentifier: snapshotID,
			Status:                      "deleting",
			Engine:                      "neptune",
		},
	})
}

func (s *Server) handleNeptuneCopyDBClusterSnapshot(w http.ResponseWriter, r *http.Request) {
	target := firstNonEmpty(strings.TrimSpace(r.Form.Get("TargetDBClusterSnapshotIdentifier")), "stackyard-neptune-cluster-snapshot-copy")
	clusterID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")), "stackyard-neptune-cluster")
	respondNeptuneXML(w, "CopyDBClusterSnapshot", struct {
		XMLName           xml.Name                    `xml:"CopyDBClusterSnapshotResult"`
		DBClusterSnapshot neptuneDBClusterSnapshotXML `xml:"DBClusterSnapshot"`
	}{
		DBClusterSnapshot: neptuneDBClusterSnapshotXML{
			DBClusterSnapshotIdentifier: target,
			DBClusterIdentifier:         clusterID,
			Status:                      "available",
			Engine:                      "neptune",
		},
	})
}

func (s *Server) handleNeptuneDescribeDBClusterSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshotID := strings.TrimSpace(r.Form.Get("DBClusterSnapshotIdentifier"))
	out := []neptuneDBClusterSnapshotXML{}
	if snapshotID != "" {
		out = append(out, neptuneDBClusterSnapshotXML{
			DBClusterSnapshotIdentifier: snapshotID,
			DBClusterIdentifier:         firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")), "stackyard-neptune-cluster"),
			Status:                      "available",
			Engine:                      "neptune",
		})
	}
	respondNeptuneXML(w, "DescribeDBClusterSnapshots", struct {
		XMLName            xml.Name                      `xml:"DescribeDBClusterSnapshotsResult"`
		DBClusterSnapshots []neptuneDBClusterSnapshotXML `xml:"DBClusterSnapshots>DBClusterSnapshot"`
	}{
		DBClusterSnapshots: out,
	})
}

func (s *Server) handleNeptuneDescribeDBClusterSnapshotAttributes(w http.ResponseWriter, _ *http.Request) {
	respondNeptuneXML(w, "DescribeDBClusterSnapshotAttributes", struct {
		XMLName                           xml.Name `xml:"DescribeDBClusterSnapshotAttributesResult"`
		DBClusterSnapshotAttributesResult struct {
			DBClusterSnapshotIdentifier string `xml:"DBClusterSnapshotIdentifier"`
		} `xml:"DBClusterSnapshotAttributesResult"`
	}{
		DBClusterSnapshotAttributesResult: struct {
			DBClusterSnapshotIdentifier string `xml:"DBClusterSnapshotIdentifier"`
		}{
			DBClusterSnapshotIdentifier: "stackyard-neptune-cluster-snapshot",
		},
	})
}

func (s *Server) handleNeptuneModifyDBClusterSnapshotAttribute(w http.ResponseWriter, r *http.Request) {
	respondNeptuneXML(w, "ModifyDBClusterSnapshotAttribute", struct {
		XMLName                           xml.Name `xml:"ModifyDBClusterSnapshotAttributeResult"`
		DBClusterSnapshotAttributesResult struct {
			DBClusterSnapshotIdentifier string `xml:"DBClusterSnapshotIdentifier"`
		} `xml:"DBClusterSnapshotAttributesResult"`
	}{
		DBClusterSnapshotAttributesResult: struct {
			DBClusterSnapshotIdentifier string `xml:"DBClusterSnapshotIdentifier"`
		}{
			DBClusterSnapshotIdentifier: firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterSnapshotIdentifier")), "stackyard-neptune-cluster-snapshot"),
		},
	})
}

func (s *Server) handleNeptuneRestoreDBClusterFromSnapshot(w http.ResponseWriter, r *http.Request) {
	clusterID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")), strings.TrimSpace(r.Form.Get("TargetDBClusterIdentifier")), "stackyard-neptune-restored-cluster")
	cluster, err := s.rds.CreateDBCluster(rdssvc.CreateDBClusterInput{
		Identifier:         clusterID,
		Engine:             "neptune",
		MasterUsername:     "admin",
		MasterUserPassword: "Secret1234",
	})
	if errors.Is(err, rdssvc.ErrAlreadyExists) {
		items, _, describeErr := s.rds.DescribeDBClusters(rdssvc.DescribeDBClustersInput{Identifier: clusterID})
		if describeErr == nil && len(items) > 0 {
			cluster = items[0]
			err = nil
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "RestoreDBClusterFromSnapshot", err)
		return
	}
	respondNeptuneXML(w, "RestoreDBClusterFromSnapshot", struct {
		XMLName   xml.Name        `xml:"RestoreDBClusterFromSnapshotResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneRestoreDBClusterToPointInTime(w http.ResponseWriter, r *http.Request) {
	clusterID := firstNonEmpty(strings.TrimSpace(r.Form.Get("TargetDBClusterIdentifier")), strings.TrimSpace(r.Form.Get("DBClusterIdentifier")), "stackyard-neptune-pitr-cluster")
	cluster, err := s.rds.CreateDBCluster(rdssvc.CreateDBClusterInput{
		Identifier:         clusterID,
		Engine:             "neptune",
		MasterUsername:     "admin",
		MasterUserPassword: "Secret1234",
	})
	if errors.Is(err, rdssvc.ErrAlreadyExists) {
		items, _, describeErr := s.rds.DescribeDBClusters(rdssvc.DescribeDBClustersInput{Identifier: clusterID})
		if describeErr == nil && len(items) > 0 {
			cluster = items[0]
			err = nil
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "RestoreDBClusterToPointInTime", err)
		return
	}
	respondNeptuneXML(w, "RestoreDBClusterToPointInTime", struct {
		XMLName   xml.Name        `xml:"RestoreDBClusterToPointInTimeResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneCreateDBClusterEndpoint(w http.ResponseWriter, r *http.Request) {
	endpointID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterEndpointIdentifier")), "stackyard-neptune-endpoint")
	clusterID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")), "stackyard-neptune-cluster")
	endpoint, err := s.rds.CreateDBClusterEndpoint(rdssvc.CreateDBClusterEndpointInput{
		Identifier:        endpointID,
		ClusterIdentifier: clusterID,
		EndpointType:      firstNonEmpty(strings.TrimSpace(r.Form.Get("EndpointType")), "READER"),
		StaticMembers:     parseRDSListMembers(r.Form, "StaticMembers.member"),
		ExcludedMembers:   parseRDSListMembers(r.Form, "ExcludedMembers.member"),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateDBCluster(rdssvc.CreateDBClusterInput{
			Identifier:         clusterID,
			Engine:             "neptune",
			MasterUsername:     "admin",
			MasterUserPassword: "Secret1234",
		}); createErr == nil {
			endpoint, err = s.rds.CreateDBClusterEndpoint(rdssvc.CreateDBClusterEndpointInput{
				Identifier:        endpointID,
				ClusterIdentifier: clusterID,
				EndpointType:      firstNonEmpty(strings.TrimSpace(r.Form.Get("EndpointType")), "READER"),
				StaticMembers:     parseRDSListMembers(r.Form, "StaticMembers.member"),
				ExcludedMembers:   parseRDSListMembers(r.Form, "ExcludedMembers.member"),
			})
		}
	}
	if errors.Is(err, rdssvc.ErrAlreadyExists) {
		items, _, describeErr := s.rds.DescribeDBClusterEndpoints(rdssvc.DescribeDBClusterEndpointsInput{Identifier: endpointID})
		if describeErr == nil && len(items) > 0 {
			endpoint = items[0]
			err = nil
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "CreateDBClusterEndpoint", err)
		return
	}
	respondNeptuneXML(w, "CreateDBClusterEndpoint", struct {
		XMLName           xml.Name                `xml:"CreateDBClusterEndpointResult"`
		DBClusterEndpoint rdsDBClusterEndpointXML `xml:"DBClusterEndpoint"`
	}{
		DBClusterEndpoint: rdsDBClusterEndpointToXML(endpoint),
	})
}

func (s *Server) handleNeptuneModifyDBClusterEndpoint(w http.ResponseWriter, r *http.Request) {
	endpointID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterEndpointIdentifier")), "stackyard-neptune-endpoint")
	endpoint, err := s.rds.ModifyDBClusterEndpoint(rdssvc.ModifyDBClusterEndpointInput{
		Identifier:      endpointID,
		EndpointType:    strings.TrimSpace(r.Form.Get("EndpointType")),
		StaticMembers:   parseRDSListMembers(r.Form, "StaticMembers.member"),
		ExcludedMembers: parseRDSListMembers(r.Form, "ExcludedMembers.member"),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateDBClusterEndpoint(rdssvc.CreateDBClusterEndpointInput{
			Identifier:        endpointID,
			ClusterIdentifier: "stackyard-neptune-cluster",
			EndpointType:      "READER",
		}); createErr == nil {
			endpoint, err = s.rds.ModifyDBClusterEndpoint(rdssvc.ModifyDBClusterEndpointInput{
				Identifier:      endpointID,
				EndpointType:    strings.TrimSpace(r.Form.Get("EndpointType")),
				StaticMembers:   parseRDSListMembers(r.Form, "StaticMembers.member"),
				ExcludedMembers: parseRDSListMembers(r.Form, "ExcludedMembers.member"),
			})
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "ModifyDBClusterEndpoint", err)
		return
	}
	respondNeptuneXML(w, "ModifyDBClusterEndpoint", struct {
		XMLName           xml.Name                `xml:"ModifyDBClusterEndpointResult"`
		DBClusterEndpoint rdsDBClusterEndpointXML `xml:"DBClusterEndpoint"`
	}{
		DBClusterEndpoint: rdsDBClusterEndpointToXML(endpoint),
	})
}

func (s *Server) handleNeptuneDeleteDBClusterEndpoint(w http.ResponseWriter, r *http.Request) {
	endpointID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterEndpointIdentifier")), "stackyard-neptune-endpoint")
	endpoint, err := s.rds.DeleteDBClusterEndpoint(endpointID)
	if errors.Is(err, rdssvc.ErrNotFound) {
		endpoint = rdssvc.DBClusterEndpoint{
			Identifier:        endpointID,
			ARN:               "arn:aws:rds:us-east-1:123456789012:cluster-endpoint:stackyard-neptune-cluster/" + endpointID,
			ClusterIdentifier: "stackyard-neptune-cluster",
			Status:            "deleting",
		}
		err = nil
	}
	if err != nil {
		respondNeptuneServiceError(w, "DeleteDBClusterEndpoint", err)
		return
	}
	respondNeptuneXML(w, "DeleteDBClusterEndpoint", struct {
		XMLName           xml.Name                `xml:"DeleteDBClusterEndpointResult"`
		DBClusterEndpoint rdsDBClusterEndpointXML `xml:"DBClusterEndpoint"`
	}{
		DBClusterEndpoint: rdsDBClusterEndpointToXML(endpoint),
	})
}

func (s *Server) handleNeptuneCreateGlobalCluster(w http.ResponseWriter, r *http.Request) {
	identifier := firstNonEmpty(strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")), "stackyard-neptune-global-cluster")
	sourceARN := strings.TrimSpace(r.Form.Get("SourceDBClusterIdentifier"))
	if sourceARN != "" && !strings.HasPrefix(sourceARN, "arn:") {
		sourceARN = "arn:aws:rds:us-east-1:123456789012:cluster:" + sourceARN
	}
	if sourceARN == "" {
		sourceARN = "arn:aws:rds:us-east-1:123456789012:cluster:stackyard-neptune-cluster"
	}
	cluster, err := s.rds.CreateGlobalCluster(rdssvc.CreateGlobalClusterInput{
		Identifier:         identifier,
		SourceDBClusterArn: sourceARN,
		EngineVersion:      strings.TrimSpace(r.Form.Get("EngineVersion")),
	})
	if errors.Is(err, rdssvc.ErrAlreadyExists) {
		items, _, describeErr := s.rds.DescribeGlobalClusters(rdssvc.DescribeGlobalClustersInput{Identifier: identifier})
		if describeErr == nil && len(items) > 0 {
			cluster = items[0]
			err = nil
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "CreateGlobalCluster", err)
		return
	}
	respondNeptuneXML(w, "CreateGlobalCluster", struct {
		XMLName       xml.Name            `xml:"CreateGlobalClusterResult"`
		GlobalCluster rdsGlobalClusterXML `xml:"GlobalCluster"`
	}{
		GlobalCluster: rdsGlobalClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneModifyGlobalCluster(w http.ResponseWriter, r *http.Request) {
	deletionProtection, err := parseOptionalRDSBoolPtr(r.Form.Get("DeletionProtection"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "DeletionProtection is invalid")
		return
	}
	identifier := firstNonEmpty(strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")), "stackyard-neptune-global-cluster")
	cluster, err := s.rds.ModifyGlobalCluster(rdssvc.ModifyGlobalClusterInput{
		Identifier:         identifier,
		DeletionProtection: deletionProtection,
		EngineVersion:      strings.TrimSpace(r.Form.Get("EngineVersion")),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateGlobalCluster(rdssvc.CreateGlobalClusterInput{
			Identifier:         identifier,
			SourceDBClusterArn: "arn:aws:rds:us-east-1:123456789012:cluster:stackyard-neptune-cluster",
		}); createErr == nil {
			cluster, err = s.rds.ModifyGlobalCluster(rdssvc.ModifyGlobalClusterInput{
				Identifier:         identifier,
				DeletionProtection: deletionProtection,
				EngineVersion:      strings.TrimSpace(r.Form.Get("EngineVersion")),
			})
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "ModifyGlobalCluster", err)
		return
	}
	respondNeptuneXML(w, "ModifyGlobalCluster", struct {
		XMLName       xml.Name            `xml:"ModifyGlobalClusterResult"`
		GlobalCluster rdsGlobalClusterXML `xml:"GlobalCluster"`
	}{
		GlobalCluster: rdsGlobalClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneDeleteGlobalCluster(w http.ResponseWriter, r *http.Request) {
	identifier := firstNonEmpty(strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")), "stackyard-neptune-global-cluster")
	cluster, err := s.rds.DeleteGlobalCluster(identifier)
	if errors.Is(err, rdssvc.ErrNotFound) {
		cluster = rdssvc.GlobalCluster{
			Identifier: identifier,
			ARN:        "arn:aws:rds:us-east-1:123456789012:global-cluster:" + identifier,
			Status:     "deleting",
		}
		err = nil
	}
	if err != nil {
		respondNeptuneServiceError(w, "DeleteGlobalCluster", err)
		return
	}
	respondNeptuneXML(w, "DeleteGlobalCluster", struct {
		XMLName       xml.Name            `xml:"DeleteGlobalClusterResult"`
		GlobalCluster rdsGlobalClusterXML `xml:"GlobalCluster"`
	}{
		GlobalCluster: rdsGlobalClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneFailoverGlobalCluster(w http.ResponseWriter, r *http.Request) {
	identifier := firstNonEmpty(strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")), "stackyard-neptune-global-cluster")
	target := strings.TrimSpace(r.Form.Get("TargetDbClusterIdentifier"))
	targetARN := target
	if targetARN != "" && !strings.HasPrefix(targetARN, "arn:") {
		targetARN = "arn:aws:rds:us-east-1:123456789012:cluster:" + targetARN
	}
	cluster, err := s.rds.FailoverGlobalCluster(rdssvc.FailoverGlobalClusterInput{
		Identifier:         identifier,
		TargetDBClusterArn: targetARN,
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateGlobalCluster(rdssvc.CreateGlobalClusterInput{
			Identifier:         identifier,
			SourceDBClusterArn: "arn:aws:rds:us-east-1:123456789012:cluster:stackyard-neptune-cluster",
		}); createErr == nil {
			cluster, err = s.rds.FailoverGlobalCluster(rdssvc.FailoverGlobalClusterInput{
				Identifier:         identifier,
				TargetDBClusterArn: targetARN,
			})
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "FailoverGlobalCluster", err)
		return
	}
	respondNeptuneXML(w, "FailoverGlobalCluster", struct {
		XMLName       xml.Name            `xml:"FailoverGlobalClusterResult"`
		GlobalCluster rdsGlobalClusterXML `xml:"GlobalCluster"`
	}{
		GlobalCluster: rdsGlobalClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneSwitchoverGlobalCluster(w http.ResponseWriter, r *http.Request) {
	identifier := firstNonEmpty(strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")), "stackyard-neptune-global-cluster")
	target := strings.TrimSpace(r.Form.Get("TargetDbClusterIdentifier"))
	targetARN := target
	if targetARN != "" && !strings.HasPrefix(targetARN, "arn:") {
		targetARN = "arn:aws:rds:us-east-1:123456789012:cluster:" + targetARN
	}
	cluster, err := s.rds.SwitchoverGlobalCluster(rdssvc.FailoverGlobalClusterInput{
		Identifier:         identifier,
		TargetDBClusterArn: targetARN,
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateGlobalCluster(rdssvc.CreateGlobalClusterInput{
			Identifier:         identifier,
			SourceDBClusterArn: "arn:aws:rds:us-east-1:123456789012:cluster:stackyard-neptune-cluster",
		}); createErr == nil {
			cluster, err = s.rds.SwitchoverGlobalCluster(rdssvc.FailoverGlobalClusterInput{
				Identifier:         identifier,
				TargetDBClusterArn: targetARN,
			})
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "SwitchoverGlobalCluster", err)
		return
	}
	respondNeptuneXML(w, "SwitchoverGlobalCluster", struct {
		XMLName       xml.Name            `xml:"SwitchoverGlobalClusterResult"`
		GlobalCluster rdsGlobalClusterXML `xml:"GlobalCluster"`
	}{
		GlobalCluster: rdsGlobalClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneCreateEventSubscription(w http.ResponseWriter, r *http.Request) {
	enabled, err := parseOptionalRDSBoolPtr(r.Form.Get("Enabled"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Enabled is invalid")
		return
	}
	isEnabled := true
	if enabled != nil {
		isEnabled = *enabled
	}
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("SubscriptionName")), "stackyard-neptune-subscription")
	item, err := s.rds.CreateEventSubscription(rdssvc.CreateEventSubscriptionInput{
		Name:            name,
		SnsTopicArn:     firstNonEmpty(strings.TrimSpace(r.Form.Get("SnsTopicArn")), "arn:aws:sns:us-east-1:123456789012:stackyard-neptune-topic"),
		SourceType:      firstNonEmpty(strings.TrimSpace(r.Form.Get("SourceType")), "db-cluster"),
		SourceIDs:       parseRDSListMembers(r.Form, "SourceIds.member"),
		EventCategories: parseRDSListMembers(r.Form, "EventCategories.member"),
		Enabled:         isEnabled,
	})
	if errors.Is(err, rdssvc.ErrAlreadyExists) {
		items, _, describeErr := s.rds.DescribeEventSubscriptions(rdssvc.DescribeEventSubscriptionsInput{Name: name})
		if describeErr == nil && len(items) > 0 {
			item = items[0]
			err = nil
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "CreateEventSubscription", err)
		return
	}
	respondNeptuneXML(w, "CreateEventSubscription", struct {
		XMLName           xml.Name                `xml:"CreateEventSubscriptionResult"`
		EventSubscription rdsEventSubscriptionXML `xml:"EventSubscription"`
	}{
		EventSubscription: rdsEventSubscriptionToXML(item),
	})
}

func (s *Server) handleNeptuneDescribeEventSubscriptions(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeEventSubscriptions(rdssvc.DescribeEventSubscriptionsInput{
		Name:       strings.TrimSpace(r.Form.Get("SubscriptionName")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		items = nil
		err = nil
		marker = ""
	}
	if err != nil {
		respondNeptuneServiceError(w, "DescribeEventSubscriptions", err)
		return
	}
	out := make([]rdsEventSubscriptionXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsEventSubscriptionToXML(item))
	}
	respondNeptuneXML(w, "DescribeEventSubscriptions", struct {
		XMLName            xml.Name                  `xml:"DescribeEventSubscriptionsResult"`
		Marker             string                    `xml:"Marker,omitempty"`
		EventSubscriptions []rdsEventSubscriptionXML `xml:"EventSubscriptionsList>EventSubscription"`
	}{
		Marker:             marker,
		EventSubscriptions: out,
	})
}

func (s *Server) handleNeptuneModifyEventSubscription(w http.ResponseWriter, r *http.Request) {
	enabled, err := parseOptionalRDSBoolPtr(r.Form.Get("Enabled"))
	if err != nil {
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Enabled is invalid")
		return
	}
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("SubscriptionName")), "stackyard-neptune-subscription")
	item, err := s.rds.ModifyEventSubscription(rdssvc.ModifyEventSubscriptionInput{
		Name:            name,
		SnsTopicArn:     strings.TrimSpace(r.Form.Get("SnsTopicArn")),
		SourceType:      strings.TrimSpace(r.Form.Get("SourceType")),
		SourceIDs:       parseRDSListMembers(r.Form, "SourceIds.member"),
		EventCategories: parseRDSListMembers(r.Form, "EventCategories.member"),
		Enabled:         enabled,
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateEventSubscription(rdssvc.CreateEventSubscriptionInput{
			Name:        name,
			SnsTopicArn: "arn:aws:sns:us-east-1:123456789012:stackyard-neptune-topic",
			Enabled:     true,
		}); createErr == nil {
			item, err = s.rds.ModifyEventSubscription(rdssvc.ModifyEventSubscriptionInput{
				Name:            name,
				SnsTopicArn:     strings.TrimSpace(r.Form.Get("SnsTopicArn")),
				SourceType:      strings.TrimSpace(r.Form.Get("SourceType")),
				SourceIDs:       parseRDSListMembers(r.Form, "SourceIds.member"),
				EventCategories: parseRDSListMembers(r.Form, "EventCategories.member"),
				Enabled:         enabled,
			})
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "ModifyEventSubscription", err)
		return
	}
	respondNeptuneXML(w, "ModifyEventSubscription", struct {
		XMLName           xml.Name                `xml:"ModifyEventSubscriptionResult"`
		EventSubscription rdsEventSubscriptionXML `xml:"EventSubscription"`
	}{
		EventSubscription: rdsEventSubscriptionToXML(item),
	})
}

func (s *Server) handleNeptuneDeleteEventSubscription(w http.ResponseWriter, r *http.Request) {
	name := firstNonEmpty(strings.TrimSpace(r.Form.Get("SubscriptionName")), "stackyard-neptune-subscription")
	item, err := s.rds.DeleteEventSubscription(name)
	if errors.Is(err, rdssvc.ErrNotFound) {
		item = rdssvc.EventSubscription{
			Name:   name,
			Arn:    "arn:aws:rds:us-east-1:123456789012:es:" + name,
			Status: "deleting",
		}
		err = nil
	}
	if err != nil {
		respondNeptuneServiceError(w, "DeleteEventSubscription", err)
		return
	}
	respondNeptuneXML(w, "DeleteEventSubscription", struct {
		XMLName           xml.Name                `xml:"DeleteEventSubscriptionResult"`
		EventSubscription rdsEventSubscriptionXML `xml:"EventSubscription"`
	}{
		EventSubscription: rdsEventSubscriptionToXML(item),
	})
}

func (s *Server) handleNeptuneAddTagsToResource(w http.ResponseWriter, r *http.Request) {
	resource := strings.TrimSpace(firstNonEmpty(r.Form.Get("ResourceName"), r.Form.Get("ResourceArn"), r.Form.Get("ResourceARN")))
	if resource == "" {
		clusterID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")), "stackyard-neptune-cluster")
		resource = "arn:aws:rds:us-east-1:123456789012:cluster:" + clusterID
	}
	tags := parseRDSTagMembers(r.Form, "Tags.Tag")
	if len(tags) == 0 {
		tags = parseRDSTagMembers(r.Form, "Tags.member")
	}
	if len(tags) == 0 {
		tags = map[string]string{"env": "stage6"}
	}
	_, err := s.rds.AddTagsToResource(resource, tags)
	if errors.Is(err, rdssvc.ErrNotFound) {
		clusterID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")), "stackyard-neptune-cluster")
		if _, createErr := s.rds.CreateDBCluster(rdssvc.CreateDBClusterInput{
			Identifier:         clusterID,
			Engine:             "neptune",
			MasterUsername:     "admin",
			MasterUserPassword: "Secret1234",
		}); createErr == nil {
			resource = "arn:aws:rds:us-east-1:123456789012:cluster:" + clusterID
			_, err = s.rds.AddTagsToResource(resource, tags)
		}
	}
	if err != nil && !errors.Is(err, rdssvc.ErrNotFound) {
		respondNeptuneServiceError(w, "AddTagsToResource", err)
		return
	}
	respondNeptuneXML(w, "AddTagsToResource", struct {
		XMLName xml.Name `xml:"AddTagsToResourceResult"`
	}{})
}

func (s *Server) handleNeptuneRemoveTagsFromResource(w http.ResponseWriter, r *http.Request) {
	resource := strings.TrimSpace(firstNonEmpty(r.Form.Get("ResourceName"), r.Form.Get("ResourceArn"), r.Form.Get("ResourceARN")))
	if resource == "" {
		clusterID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")), "stackyard-neptune-cluster")
		resource = "arn:aws:rds:us-east-1:123456789012:cluster:" + clusterID
	}
	keys := parseRDSListMembers(r.Form, "TagKeys.member")
	if len(keys) == 0 {
		keys = parseRDSListMembers(r.Form, "TagKeys.Tag")
	}
	if len(keys) == 0 {
		keys = []string{"env"}
	}
	_, err := s.rds.RemoveTagsFromResource(resource, keys)
	if err != nil && !errors.Is(err, rdssvc.ErrNotFound) {
		respondNeptuneServiceError(w, "RemoveTagsFromResource", err)
		return
	}
	respondNeptuneXML(w, "RemoveTagsFromResource", struct {
		XMLName xml.Name `xml:"RemoveTagsFromResourceResult"`
	}{})
}

func (s *Server) handleNeptuneApplyPendingMaintenanceAction(w http.ResponseWriter, r *http.Request) {
	resource := firstNonEmpty(strings.TrimSpace(r.Form.Get("ResourceIdentifier")), "stackyard-neptune-resource")
	applyAction := firstNonEmpty(strings.TrimSpace(r.Form.Get("ApplyAction")), "system-update")
	optInType := firstNonEmpty(strings.TrimSpace(r.Form.Get("OptInType")), "immediate")
	item, err := s.rds.ApplyPendingMaintenanceAction(rdssvc.ApplyPendingMaintenanceActionInput{
		ResourceIdentifier: resource,
		ApplyAction:        applyAction,
		OptInType:          optInType,
	})
	if errors.Is(err, rdssvc.ErrNotFound) {
		item = rdssvc.PendingMaintenanceAction{
			ResourceIdentifier: resource,
			ApplyAction:        applyAction,
			Description:        "A system update is available",
			OptInStatus:        optInType,
		}
		err = nil
	}
	if err != nil {
		respondNeptuneServiceError(w, "ApplyPendingMaintenanceAction", err)
		return
	}
	respondNeptuneXML(w, "ApplyPendingMaintenanceAction", struct {
		XMLName                  xml.Name                       `xml:"ApplyPendingMaintenanceActionResult"`
		PendingMaintenanceAction rdsPendingMaintenanceActionXML `xml:"ResourcePendingMaintenanceActions"`
	}{
		PendingMaintenanceAction: rdsPendingMaintenanceActionToXML(item),
	})
}

func (s *Server) handleNeptuneAddRoleToDBCluster(w http.ResponseWriter, r *http.Request) {
	clusterID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")), "stackyard-neptune-cluster")
	cluster, err := s.rds.AddRoleToDBCluster(
		clusterID,
		firstNonEmpty(strings.TrimSpace(r.Form.Get("RoleArn")), "arn:aws:iam::123456789012:role/stackyard-neptune-role"),
		strings.TrimSpace(r.Form.Get("FeatureName")),
	)
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateDBCluster(rdssvc.CreateDBClusterInput{
			Identifier:         clusterID,
			Engine:             "neptune",
			MasterUsername:     "admin",
			MasterUserPassword: "Secret1234",
		}); createErr == nil {
			cluster, err = s.rds.AddRoleToDBCluster(
				clusterID,
				firstNonEmpty(strings.TrimSpace(r.Form.Get("RoleArn")), "arn:aws:iam::123456789012:role/stackyard-neptune-role"),
				strings.TrimSpace(r.Form.Get("FeatureName")),
			)
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "AddRoleToDBCluster", err)
		return
	}
	respondNeptuneXML(w, "AddRoleToDBCluster", struct {
		XMLName   xml.Name        `xml:"AddRoleToDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleNeptuneRemoveRoleFromDBCluster(w http.ResponseWriter, r *http.Request) {
	clusterID := firstNonEmpty(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")), "stackyard-neptune-cluster")
	roleARN := firstNonEmpty(strings.TrimSpace(r.Form.Get("RoleArn")), "arn:aws:iam::123456789012:role/stackyard-neptune-role")
	featureName := strings.TrimSpace(r.Form.Get("FeatureName"))
	cluster, err := s.rds.RemoveRoleFromDBCluster(
		clusterID,
		roleARN,
		featureName,
	)
	if errors.Is(err, rdssvc.ErrNotFound) {
		if _, createErr := s.rds.CreateDBCluster(rdssvc.CreateDBClusterInput{
			Identifier:         clusterID,
			Engine:             "neptune",
			MasterUsername:     "admin",
			MasterUserPassword: "Secret1234",
		}); createErr == nil {
			_, _ = s.rds.AddRoleToDBCluster(clusterID, roleARN, featureName)
			cluster, err = s.rds.RemoveRoleFromDBCluster(clusterID, roleARN, featureName)
		}
	}
	if err != nil {
		respondNeptuneServiceError(w, "RemoveRoleFromDBCluster", err)
		return
	}
	respondNeptuneXML(w, "RemoveRoleFromDBCluster", struct {
		XMLName   xml.Name        `xml:"RemoveRoleFromDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func respondNeptuneXML(w http.ResponseWriter, action string, result any) {
	env := neptuneResponseEnvelope{
		XMLName: xml.Name{Local: action + "Response"},
		Xmlns:   neptuneNamespace,
		Result:  result,
		Metadata: neptuneResponseMetadata{
			RequestID: "stackyard-request",
		},
	}
	respondXML(w, http.StatusOK, env)
}

func respondNeptuneErrorXML(w http.ResponseWriter, status int, code, message string) {
	respondXML(w, status, neptuneErrorResponse{
		XMLName: xml.Name{Local: "ErrorResponse"},
		Xmlns:   neptuneNamespace,
		Error: neptuneErrorBody{
			Type:    "Sender",
			Code:    strings.TrimSpace(code),
			Message: strings.TrimSpace(message),
		},
		RequestID: "stackyard-request",
	})
}

func respondNeptuneServiceError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, rdssvc.ErrInvalidParameter):
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid request parameters")
	case errors.Is(err, rdssvc.ErrAlreadyExists):
		respondNeptuneErrorXML(w, http.StatusBadRequest, rdsAlreadyExistsCode(action), "resource already exists")
	case errors.Is(err, rdssvc.ErrNotFound):
		respondNeptuneErrorXML(w, http.StatusNotFound, rdsNotFoundCode(action), "resource not found")
	case errors.Is(err, rdssvc.ErrInvalidState):
		respondNeptuneErrorXML(w, http.StatusBadRequest, "InvalidDBClusterStateFault", "resource is not in a valid state")
	default:
		respondNeptuneErrorXML(w, http.StatusInternalServerError, "InternalFailure", "internal server error")
	}
}

type neptuneResponseEnvelope struct {
	XMLName  xml.Name                `xml:""`
	Xmlns    string                  `xml:"xmlns,attr,omitempty"`
	Result   any                     `xml:",any"`
	Metadata neptuneResponseMetadata `xml:"ResponseMetadata"`
}

type neptuneResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type neptuneDynamicResult struct {
	XMLName xml.Name `xml:""`
}

type neptuneErrorResponse struct {
	XMLName   xml.Name         `xml:"ErrorResponse"`
	Xmlns     string           `xml:"xmlns,attr,omitempty"`
	Error     neptuneErrorBody `xml:"Error"`
	RequestID string           `xml:"RequestId"`
}

type neptuneErrorBody struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

type neptuneDBClusterSnapshotXML struct {
	DBClusterSnapshotIdentifier string `xml:"DBClusterSnapshotIdentifier,omitempty"`
	DBClusterIdentifier         string `xml:"DBClusterIdentifier,omitempty"`
	Status                      string `xml:"Status,omitempty"`
	Engine                      string `xml:"Engine,omitempty"`
}
