package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	rdssvc "github.com/stackyard/stackyard/internal/services/rds"
)

func (s *Server) handleRDSAddTagsToResource(w http.ResponseWriter, r *http.Request) {
	resource := strings.TrimSpace(firstNonEmpty(r.Form.Get("ResourceName"), r.Form.Get("ResourceArn"), r.Form.Get("ResourceARN")))
	tags := parseRDSTagMembers(r.Form, "Tags.Tag")
	if len(tags) == 0 {
		tags = parseRDSTagMembers(r.Form, "Tags.member")
	}
	_, err := s.rds.AddTagsToResource(resource, tags)
	if err != nil {
		respondRDSServiceError(w, "AddTagsToResource", err)
		return
	}
	respondRDSXML(w, "AddTagsToResource", struct {
		XMLName xml.Name `xml:"AddTagsToResourceResult"`
	}{})
}

func (s *Server) handleRDSListTagsForResource(w http.ResponseWriter, r *http.Request) {
	resource := strings.TrimSpace(firstNonEmpty(r.Form.Get("ResourceName"), r.Form.Get("ResourceArn"), r.Form.Get("ResourceARN")))
	tags, err := s.rds.ListTagsForResource(resource)
	if err != nil {
		respondRDSServiceError(w, "ListTagsForResource", err)
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
	respondRDSXML(w, "ListTagsForResource", struct {
		XMLName xml.Name    `xml:"ListTagsForResourceResult"`
		TagList []rdsTagXML `xml:"TagList>Tag"`
	}{
		TagList: items,
	})
}

func (s *Server) handleRDSRemoveTagsFromResource(w http.ResponseWriter, r *http.Request) {
	resource := strings.TrimSpace(firstNonEmpty(r.Form.Get("ResourceName"), r.Form.Get("ResourceArn"), r.Form.Get("ResourceARN")))
	keys := parseRDSListMembers(r.Form, "TagKeys.member")
	if len(keys) == 0 {
		keys = parseRDSListMembers(r.Form, "TagKeys.Tag")
	}
	_, err := s.rds.RemoveTagsFromResource(resource, keys)
	if err != nil {
		respondRDSServiceError(w, "RemoveTagsFromResource", err)
		return
	}
	respondRDSXML(w, "RemoveTagsFromResource", struct {
		XMLName xml.Name `xml:"RemoveTagsFromResourceResult"`
	}{})
}

func (s *Server) handleRDSCreateEventSubscription(w http.ResponseWriter, r *http.Request) {
	enabled, err := parseOptionalRDSBoolPtr(r.Form.Get("Enabled"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Enabled is invalid")
		return
	}
	isEnabled := true
	if enabled != nil {
		isEnabled = *enabled
	}
	item, err := s.rds.CreateEventSubscription(rdssvc.CreateEventSubscriptionInput{
		Name:            strings.TrimSpace(r.Form.Get("SubscriptionName")),
		SnsTopicArn:     strings.TrimSpace(r.Form.Get("SnsTopicArn")),
		SourceType:      strings.TrimSpace(r.Form.Get("SourceType")),
		SourceIDs:       parseRDSListMembers(r.Form, "SourceIds.member"),
		EventCategories: parseRDSListMembers(r.Form, "EventCategories.member"),
		Enabled:         isEnabled,
	})
	if err != nil {
		respondRDSServiceError(w, "CreateEventSubscription", err)
		return
	}
	respondRDSXML(w, "CreateEventSubscription", struct {
		XMLName           xml.Name                `xml:"CreateEventSubscriptionResult"`
		EventSubscription rdsEventSubscriptionXML `xml:"EventSubscription"`
	}{
		EventSubscription: rdsEventSubscriptionToXML(item),
	})
}

func (s *Server) handleRDSDescribeEventSubscriptions(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeEventSubscriptions(rdssvc.DescribeEventSubscriptionsInput{
		Name:       strings.TrimSpace(r.Form.Get("SubscriptionName")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeEventSubscriptions", err)
		return
	}
	out := make([]rdsEventSubscriptionXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsEventSubscriptionToXML(item))
	}
	respondRDSXML(w, "DescribeEventSubscriptions", struct {
		XMLName            xml.Name                  `xml:"DescribeEventSubscriptionsResult"`
		Marker             string                    `xml:"Marker,omitempty"`
		EventSubscriptions []rdsEventSubscriptionXML `xml:"EventSubscriptionsList>EventSubscription"`
	}{
		Marker:             marker,
		EventSubscriptions: out,
	})
}

func (s *Server) handleRDSModifyEventSubscription(w http.ResponseWriter, r *http.Request) {
	enabled, err := parseOptionalRDSBoolPtr(r.Form.Get("Enabled"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Enabled is invalid")
		return
	}
	item, err := s.rds.ModifyEventSubscription(rdssvc.ModifyEventSubscriptionInput{
		Name:            strings.TrimSpace(r.Form.Get("SubscriptionName")),
		SnsTopicArn:     strings.TrimSpace(r.Form.Get("SnsTopicArn")),
		SourceType:      strings.TrimSpace(r.Form.Get("SourceType")),
		SourceIDs:       parseRDSListMembers(r.Form, "SourceIds.member"),
		EventCategories: parseRDSListMembers(r.Form, "EventCategories.member"),
		Enabled:         enabled,
	})
	if err != nil {
		respondRDSServiceError(w, "ModifyEventSubscription", err)
		return
	}
	respondRDSXML(w, "ModifyEventSubscription", struct {
		XMLName           xml.Name                `xml:"ModifyEventSubscriptionResult"`
		EventSubscription rdsEventSubscriptionXML `xml:"EventSubscription"`
	}{
		EventSubscription: rdsEventSubscriptionToXML(item),
	})
}

func (s *Server) handleRDSDeleteEventSubscription(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.DeleteEventSubscription(strings.TrimSpace(r.Form.Get("SubscriptionName")))
	if err != nil {
		respondRDSServiceError(w, "DeleteEventSubscription", err)
		return
	}
	respondRDSXML(w, "DeleteEventSubscription", struct {
		XMLName           xml.Name                `xml:"DeleteEventSubscriptionResult"`
		EventSubscription rdsEventSubscriptionXML `xml:"EventSubscription"`
	}{
		EventSubscription: rdsEventSubscriptionToXML(item),
	})
}

func (s *Server) handleRDSDescribePendingMaintenanceActions(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribePendingMaintenanceActions(rdssvc.DescribePendingMaintenanceActionsInput{
		ResourceIdentifier: strings.TrimSpace(r.Form.Get("ResourceIdentifier")),
		MaxRecords:         maxRecords,
		Marker:             strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribePendingMaintenanceActions", err)
		return
	}
	out := make([]rdsPendingMaintenanceActionXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsPendingMaintenanceActionToXML(item))
	}
	respondRDSXML(w, "DescribePendingMaintenanceActions", struct {
		XMLName                   xml.Name                         `xml:"DescribePendingMaintenanceActionsResult"`
		Marker                    string                           `xml:"Marker,omitempty"`
		PendingMaintenanceActions []rdsPendingMaintenanceActionXML `xml:"PendingMaintenanceActions>ResourcePendingMaintenanceActions"`
	}{
		Marker:                    marker,
		PendingMaintenanceActions: out,
	})
}

func (s *Server) handleRDSApplyPendingMaintenanceAction(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.ApplyPendingMaintenanceAction(rdssvc.ApplyPendingMaintenanceActionInput{
		ResourceIdentifier: strings.TrimSpace(r.Form.Get("ResourceIdentifier")),
		ApplyAction:        strings.TrimSpace(r.Form.Get("ApplyAction")),
		OptInType:          strings.TrimSpace(r.Form.Get("OptInType")),
	})
	if err != nil {
		respondRDSServiceError(w, "ApplyPendingMaintenanceAction", err)
		return
	}
	respondRDSXML(w, "ApplyPendingMaintenanceAction", struct {
		XMLName                  xml.Name                       `xml:"ApplyPendingMaintenanceActionResult"`
		PendingMaintenanceAction rdsPendingMaintenanceActionXML `xml:"ResourcePendingMaintenanceActions"`
	}{
		PendingMaintenanceAction: rdsPendingMaintenanceActionToXML(item),
	})
}

func (s *Server) handleRDSDescribeEvents(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	duration, err := parseOptionalRDSInt(r.Form.Get("Duration"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Duration is invalid")
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
		respondRDSServiceError(w, "DescribeEvents", err)
		return
	}
	out := make([]rdsEventRecordXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsEventRecordToXML(item))
	}
	respondRDSXML(w, "DescribeEvents", struct {
		XMLName xml.Name            `xml:"DescribeEventsResult"`
		Marker  string              `xml:"Marker,omitempty"`
		Events  []rdsEventRecordXML `xml:"Events>Event"`
	}{
		Marker: marker,
		Events: out,
	})
}

func (s *Server) handleRDSDescribeAccountAttributes(w http.ResponseWriter, _ *http.Request) {
	attrs := s.rds.DescribeAccountAttributes()
	out := make([]rdsAccountAttributeXML, 0, len(attrs))
	for _, item := range attrs {
		out = append(out, rdsAccountAttributeToXML(item))
	}
	respondRDSXML(w, "DescribeAccountAttributes", struct {
		XMLName           xml.Name                 `xml:"DescribeAccountAttributesResult"`
		AccountAttributes []rdsAccountAttributeXML `xml:"AccountAttributes>AccountQuota"`
	}{
		AccountAttributes: out,
	})
}

func (s *Server) handleRDSDescribeDBEngineVersions(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeDBEngineVersions(rdssvc.DescribeDBEngineVersionsInput{
		Engine:        strings.TrimSpace(r.Form.Get("Engine")),
		EngineVersion: strings.TrimSpace(r.Form.Get("EngineVersion")),
		MaxRecords:    maxRecords,
		Marker:        strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBEngineVersions", err)
		return
	}
	out := make([]rdsDBEngineVersionXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsDBEngineVersionToXML(item))
	}
	respondRDSXML(w, "DescribeDBEngineVersions", struct {
		XMLName          xml.Name                `xml:"DescribeDBEngineVersionsResult"`
		Marker           string                  `xml:"Marker,omitempty"`
		DBEngineVersions []rdsDBEngineVersionXML `xml:"DBEngineVersions>DBEngineVersion"`
	}{
		Marker:           marker,
		DBEngineVersions: out,
	})
}

func (s *Server) handleRDSDescribeOrderableDBInstanceOptions(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	vpc, err := parseOptionalRDSBoolPtr(r.Form.Get("Vpc"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Vpc is invalid")
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
		respondRDSServiceError(w, "DescribeOrderableDBInstanceOptions", err)
		return
	}
	out := make([]rdsOrderableDBInstanceOptionXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsOrderableDBInstanceOptionToXML(item))
	}
	respondRDSXML(w, "DescribeOrderableDBInstanceOptions", struct {
		XMLName                    xml.Name                          `xml:"DescribeOrderableDBInstanceOptionsResult"`
		Marker                     string                            `xml:"Marker,omitempty"`
		OrderableDBInstanceOptions []rdsOrderableDBInstanceOptionXML `xml:"OrderableDBInstanceOptions>OrderableDBInstanceOption"`
	}{
		Marker:                     marker,
		OrderableDBInstanceOptions: out,
	})
}

func (s *Server) handleRDSDescribeSourceRegions(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeSourceRegions(rdssvc.DescribeSourceRegionsInput{
		RegionName: strings.TrimSpace(r.Form.Get("RegionName")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeSourceRegions", err)
		return
	}
	out := make([]rdsSourceRegionXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsSourceRegionToXML(item))
	}
	respondRDSXML(w, "DescribeSourceRegions", struct {
		XMLName       xml.Name             `xml:"DescribeSourceRegionsResult"`
		Marker        string               `xml:"Marker,omitempty"`
		SourceRegions []rdsSourceRegionXML `xml:"SourceRegions>SourceRegion"`
	}{
		Marker:        marker,
		SourceRegions: out,
	})
}

func (s *Server) handleRDSDescribeValidDBInstanceModifications(w http.ResponseWriter, r *http.Request) {
	items, err := s.rds.DescribeValidDBInstanceModifications(strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "DescribeValidDBInstanceModifications", err)
		return
	}
	out := make([]rdsValidDBInstanceModificationXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsValidDBInstanceModificationToXML(item))
	}
	respondRDSXML(w, "DescribeValidDBInstanceModifications", struct {
		XMLName                             xml.Name                              `xml:"DescribeValidDBInstanceModificationsResult"`
		ValidDBInstanceModificationsMessage rdsValidDBInstanceModificationsMsgXML `xml:"ValidDBInstanceModificationsMessage"`
	}{
		ValidDBInstanceModificationsMessage: rdsValidDBInstanceModificationsMsgXML{
			Items: out,
		},
	})
}

func (s *Server) handleRDSAddRoleToDBInstance(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.AddRoleToDBInstance(
		strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")),
		strings.TrimSpace(r.Form.Get("RoleArn")),
		strings.TrimSpace(r.Form.Get("FeatureName")),
	)
	if err != nil {
		respondRDSServiceError(w, "AddRoleToDBInstance", err)
		return
	}
	respondRDSXML(w, "AddRoleToDBInstance", struct {
		XMLName    xml.Name         `xml:"AddRoleToDBInstanceResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(item),
	})
}

func (s *Server) handleRDSRemoveRoleFromDBInstance(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.RemoveRoleFromDBInstance(
		strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")),
		strings.TrimSpace(r.Form.Get("RoleArn")),
		strings.TrimSpace(r.Form.Get("FeatureName")),
	)
	if err != nil {
		respondRDSServiceError(w, "RemoveRoleFromDBInstance", err)
		return
	}
	respondRDSXML(w, "RemoveRoleFromDBInstance", struct {
		XMLName    xml.Name         `xml:"RemoveRoleFromDBInstanceResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(item),
	})
}

func (s *Server) handleRDSAddRoleToDBCluster(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.AddRoleToDBCluster(
		strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		strings.TrimSpace(r.Form.Get("RoleArn")),
		strings.TrimSpace(r.Form.Get("FeatureName")),
	)
	if err != nil {
		respondRDSServiceError(w, "AddRoleToDBCluster", err)
		return
	}
	respondRDSXML(w, "AddRoleToDBCluster", struct {
		XMLName   xml.Name        `xml:"AddRoleToDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(item),
	})
}

func (s *Server) handleRDSRemoveRoleFromDBCluster(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.RemoveRoleFromDBCluster(
		strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		strings.TrimSpace(r.Form.Get("RoleArn")),
		strings.TrimSpace(r.Form.Get("FeatureName")),
	)
	if err != nil {
		respondRDSServiceError(w, "RemoveRoleFromDBCluster", err)
		return
	}
	respondRDSXML(w, "RemoveRoleFromDBCluster", struct {
		XMLName   xml.Name        `xml:"RemoveRoleFromDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(item),
	})
}

func (s *Server) handleRDSStartActivityStream(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.StartActivityStream(rdssvc.StartActivityStreamInput{
		ResourceArn: strings.TrimSpace(r.Form.Get("ResourceArn")),
		Mode:        strings.TrimSpace(r.Form.Get("Mode")),
		KmsKeyID:    strings.TrimSpace(r.Form.Get("KmsKeyId")),
	})
	if err != nil {
		respondRDSServiceError(w, "StartActivityStream", err)
		return
	}
	respondRDSXML(w, "StartActivityStream", struct {
		XMLName        xml.Name             `xml:"StartActivityStreamResult"`
		ActivityStream rdsActivityStreamXML `xml:"ActivityStream"`
	}{
		ActivityStream: rdsActivityStreamToXML(item),
	})
}

func (s *Server) handleRDSStopActivityStream(w http.ResponseWriter, r *http.Request) {
	applyImmediately, err := parseOptionalRDSBool(r.Form.Get("ApplyImmediately"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "ApplyImmediately is invalid")
		return
	}
	item, err := s.rds.StopActivityStream(rdssvc.StopActivityStreamInput{
		ResourceArn:      strings.TrimSpace(r.Form.Get("ResourceArn")),
		ApplyImmediately: applyImmediately,
	})
	if err != nil {
		respondRDSServiceError(w, "StopActivityStream", err)
		return
	}
	respondRDSXML(w, "StopActivityStream", struct {
		XMLName        xml.Name             `xml:"StopActivityStreamResult"`
		ActivityStream rdsActivityStreamXML `xml:"ActivityStream"`
	}{
		ActivityStream: rdsActivityStreamToXML(item),
	})
}

func (s *Server) handleRDSCreateDBProxy(w http.ResponseWriter, r *http.Request) {
	requireTLS, err := parseOptionalRDSBool(r.Form.Get("RequireTLS"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "RequireTLS is invalid")
		return
	}
	idleTimeout, err := parseOptionalRDSInt(r.Form.Get("IdleClientTimeout"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "IdleClientTimeout is invalid")
		return
	}
	debugLogging, err := parseOptionalRDSBool(r.Form.Get("DebugLogging"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "DebugLogging is invalid")
		return
	}
	item, err := s.rds.CreateDBProxy(rdssvc.CreateDBProxyInput{
		Name:                strings.TrimSpace(firstNonEmpty(r.Form.Get("DBProxyName"), r.Form.Get("Name"))),
		EngineFamily:        strings.TrimSpace(r.Form.Get("EngineFamily")),
		RoleArn:             strings.TrimSpace(r.Form.Get("RoleArn")),
		VpcSubnetIDs:        parseRDSListMembers(r.Form, "VpcSubnetIds.member"),
		VpcSecurityGroupIDs: parseRDSListMembers(r.Form, "VpcSecurityGroupIds.member"),
		RequireTLS:          requireTLS,
		IdleClientTimeout:   idleTimeout,
		DebugLogging:        debugLogging,
		Auth:                parseRDSProxyAuthMembers(r.Form, "Auth.member"),
	})
	if err != nil {
		respondRDSServiceError(w, "CreateDBProxy", err)
		return
	}
	respondRDSXML(w, "CreateDBProxy", struct {
		XMLName xml.Name      `xml:"CreateDBProxyResult"`
		DBProxy rdsDBProxyXML `xml:"DBProxy"`
	}{
		DBProxy: rdsDBProxyToXML(item),
	})
}

func (s *Server) handleRDSDescribeDBProxies(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeDBProxies(rdssvc.DescribeDBProxiesInput{
		Name:       strings.TrimSpace(r.Form.Get("DBProxyName")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBProxies", err)
		return
	}
	out := make([]rdsDBProxyXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsDBProxyToXML(item))
	}
	respondRDSXML(w, "DescribeDBProxies", struct {
		XMLName   xml.Name        `xml:"DescribeDBProxiesResult"`
		Marker    string          `xml:"Marker,omitempty"`
		DBProxies []rdsDBProxyXML `xml:"DBProxies>DBProxy"`
	}{
		Marker:    marker,
		DBProxies: out,
	})
}

func (s *Server) handleRDSModifyDBProxy(w http.ResponseWriter, r *http.Request) {
	requireTLS, err := parseOptionalRDSBoolPtr(r.Form.Get("RequireTLS"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "RequireTLS is invalid")
		return
	}
	debugLogging, err := parseOptionalRDSBoolPtr(r.Form.Get("DebugLogging"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "DebugLogging is invalid")
		return
	}
	idleTimeout, err := parseOptionalRDSInt(r.Form.Get("IdleClientTimeout"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "IdleClientTimeout is invalid")
		return
	}
	item, err := s.rds.ModifyDBProxy(rdssvc.ModifyDBProxyInput{
		Name:              strings.TrimSpace(r.Form.Get("DBProxyName")),
		RoleArn:           strings.TrimSpace(r.Form.Get("RoleArn")),
		RequireTLS:        requireTLS,
		IdleClientTimeout: idleTimeout,
		DebugLogging:      debugLogging,
	})
	if err != nil {
		respondRDSServiceError(w, "ModifyDBProxy", err)
		return
	}
	respondRDSXML(w, "ModifyDBProxy", struct {
		XMLName xml.Name      `xml:"ModifyDBProxyResult"`
		DBProxy rdsDBProxyXML `xml:"DBProxy"`
	}{
		DBProxy: rdsDBProxyToXML(item),
	})
}

func (s *Server) handleRDSDeleteDBProxy(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.DeleteDBProxy(strings.TrimSpace(r.Form.Get("DBProxyName")))
	if err != nil {
		respondRDSServiceError(w, "DeleteDBProxy", err)
		return
	}
	respondRDSXML(w, "DeleteDBProxy", struct {
		XMLName xml.Name      `xml:"DeleteDBProxyResult"`
		DBProxy rdsDBProxyXML `xml:"DBProxy"`
	}{
		DBProxy: rdsDBProxyToXML(item),
	})
}

func (s *Server) handleRDSCreateDBProxyEndpoint(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.CreateDBProxyEndpoint(rdssvc.CreateDBProxyEndpointInput{
		DBProxyName:         strings.TrimSpace(r.Form.Get("DBProxyName")),
		Name:                strings.TrimSpace(r.Form.Get("DBProxyEndpointName")),
		VpcSubnetIDs:        parseRDSListMembers(r.Form, "VpcSubnetIds.member"),
		VpcSecurityGroupIDs: parseRDSListMembers(r.Form, "VpcSecurityGroupIds.member"),
		TargetRole:          strings.TrimSpace(r.Form.Get("TargetRole")),
	})
	if err != nil {
		respondRDSServiceError(w, "CreateDBProxyEndpoint", err)
		return
	}
	respondRDSXML(w, "CreateDBProxyEndpoint", struct {
		XMLName         xml.Name              `xml:"CreateDBProxyEndpointResult"`
		DBProxyEndpoint rdsDBProxyEndpointXML `xml:"DBProxyEndpoint"`
	}{
		DBProxyEndpoint: rdsDBProxyEndpointToXML(item),
	})
}

func (s *Server) handleRDSDescribeDBProxyEndpoints(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeDBProxyEndpoints(rdssvc.DescribeDBProxyEndpointsInput{
		Name:        strings.TrimSpace(r.Form.Get("DBProxyEndpointName")),
		DBProxyName: strings.TrimSpace(r.Form.Get("DBProxyName")),
		MaxRecords:  maxRecords,
		Marker:      strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBProxyEndpoints", err)
		return
	}
	out := make([]rdsDBProxyEndpointXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsDBProxyEndpointToXML(item))
	}
	respondRDSXML(w, "DescribeDBProxyEndpoints", struct {
		XMLName          xml.Name                `xml:"DescribeDBProxyEndpointsResult"`
		Marker           string                  `xml:"Marker,omitempty"`
		DBProxyEndpoints []rdsDBProxyEndpointXML `xml:"DBProxyEndpoints>DBProxyEndpoint"`
	}{
		Marker:           marker,
		DBProxyEndpoints: out,
	})
}

func (s *Server) handleRDSModifyDBProxyEndpoint(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.ModifyDBProxyEndpoint(rdssvc.ModifyDBProxyEndpointInput{
		Name:                strings.TrimSpace(r.Form.Get("DBProxyEndpointName")),
		VpcSecurityGroupIDs: parseRDSListMembers(r.Form, "VpcSecurityGroupIds.member"),
		TargetRole:          strings.TrimSpace(r.Form.Get("TargetRole")),
	})
	if err != nil {
		respondRDSServiceError(w, "ModifyDBProxyEndpoint", err)
		return
	}
	respondRDSXML(w, "ModifyDBProxyEndpoint", struct {
		XMLName         xml.Name              `xml:"ModifyDBProxyEndpointResult"`
		DBProxyEndpoint rdsDBProxyEndpointXML `xml:"DBProxyEndpoint"`
	}{
		DBProxyEndpoint: rdsDBProxyEndpointToXML(item),
	})
}

func (s *Server) handleRDSDeleteDBProxyEndpoint(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.DeleteDBProxyEndpoint(strings.TrimSpace(r.Form.Get("DBProxyEndpointName")))
	if err != nil {
		respondRDSServiceError(w, "DeleteDBProxyEndpoint", err)
		return
	}
	respondRDSXML(w, "DeleteDBProxyEndpoint", struct {
		XMLName         xml.Name              `xml:"DeleteDBProxyEndpointResult"`
		DBProxyEndpoint rdsDBProxyEndpointXML `xml:"DBProxyEndpoint"`
	}{
		DBProxyEndpoint: rdsDBProxyEndpointToXML(item),
	})
}

func (s *Server) handleRDSRegisterDBProxyTargets(w http.ResponseWriter, r *http.Request) {
	items, err := s.rds.RegisterDBProxyTargets(rdssvc.RegisterDBProxyTargetsInput{
		DBProxyName:           strings.TrimSpace(r.Form.Get("DBProxyName")),
		TargetGroupName:       strings.TrimSpace(r.Form.Get("TargetGroupName")),
		DBInstanceIdentifiers: parseRDSListMembers(r.Form, "DBInstanceIdentifiers.member"),
		DBClusterIdentifiers:  parseRDSListMembers(r.Form, "DBClusterIdentifiers.member"),
	})
	if err != nil {
		respondRDSServiceError(w, "RegisterDBProxyTargets", err)
		return
	}
	out := make([]rdsDBProxyTargetXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsDBProxyTargetToXML(item))
	}
	respondRDSXML(w, "RegisterDBProxyTargets", struct {
		XMLName        xml.Name              `xml:"RegisterDBProxyTargetsResult"`
		DBProxyTargets []rdsDBProxyTargetXML `xml:"DBProxyTargets>DBProxyTarget"`
	}{
		DBProxyTargets: out,
	})
}

func (s *Server) handleRDSDeregisterDBProxyTargets(w http.ResponseWriter, r *http.Request) {
	items, err := s.rds.DeregisterDBProxyTargets(rdssvc.DeregisterDBProxyTargetsInput{
		DBProxyName:           strings.TrimSpace(r.Form.Get("DBProxyName")),
		TargetGroupName:       strings.TrimSpace(r.Form.Get("TargetGroupName")),
		DBInstanceIdentifiers: parseRDSListMembers(r.Form, "DBInstanceIdentifiers.member"),
		DBClusterIdentifiers:  parseRDSListMembers(r.Form, "DBClusterIdentifiers.member"),
	})
	if err != nil {
		respondRDSServiceError(w, "DeregisterDBProxyTargets", err)
		return
	}
	out := make([]rdsDBProxyTargetXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsDBProxyTargetToXML(item))
	}
	respondRDSXML(w, "DeregisterDBProxyTargets", struct {
		XMLName        xml.Name              `xml:"DeregisterDBProxyTargetsResult"`
		DBProxyTargets []rdsDBProxyTargetXML `xml:"DBProxyTargets>DBProxyTarget"`
	}{
		DBProxyTargets: out,
	})
}

func (s *Server) handleRDSDescribeDBProxyTargets(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeDBProxyTargets(rdssvc.DescribeDBProxyTargetsInput{
		DBProxyName:     strings.TrimSpace(r.Form.Get("DBProxyName")),
		TargetGroupName: strings.TrimSpace(r.Form.Get("TargetGroupName")),
		MaxRecords:      maxRecords,
		Marker:          strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBProxyTargets", err)
		return
	}
	out := make([]rdsDBProxyTargetXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsDBProxyTargetToXML(item))
	}
	respondRDSXML(w, "DescribeDBProxyTargets", struct {
		XMLName        xml.Name              `xml:"DescribeDBProxyTargetsResult"`
		Marker         string                `xml:"Marker,omitempty"`
		DBProxyTargets []rdsDBProxyTargetXML `xml:"DBProxyTargets>DBProxyTarget"`
	}{
		Marker:         marker,
		DBProxyTargets: out,
	})
}

func (s *Server) handleRDSCreateIntegration(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.CreateIntegration(rdssvc.CreateIntegrationInput{
		Identifier: strings.TrimSpace(firstNonEmpty(r.Form.Get("IntegrationIdentifier"), r.Form.Get("Identifier"))),
		Name:       strings.TrimSpace(firstNonEmpty(r.Form.Get("IntegrationName"), r.Form.Get("Name"))),
		SourceArn:  strings.TrimSpace(r.Form.Get("SourceArn")),
		TargetArn:  strings.TrimSpace(r.Form.Get("TargetArn")),
	})
	if err != nil {
		respondRDSServiceError(w, "CreateIntegration", err)
		return
	}
	respondRDSXML(w, "CreateIntegration", struct {
		XMLName     xml.Name          `xml:"CreateIntegrationResult"`
		Integration rdsIntegrationXML `xml:"Integration"`
	}{
		Integration: rdsIntegrationToXML(item),
	})
}

func (s *Server) handleRDSDescribeIntegrations(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeIntegrations(rdssvc.DescribeIntegrationsInput{
		Identifier: strings.TrimSpace(firstNonEmpty(r.Form.Get("IntegrationIdentifier"), r.Form.Get("Identifier"))),
		SourceArn:  strings.TrimSpace(r.Form.Get("SourceArn")),
		TargetArn:  strings.TrimSpace(r.Form.Get("TargetArn")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeIntegrations", err)
		return
	}
	out := make([]rdsIntegrationXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsIntegrationToXML(item))
	}
	respondRDSXML(w, "DescribeIntegrations", struct {
		XMLName      xml.Name            `xml:"DescribeIntegrationsResult"`
		Marker       string              `xml:"Marker,omitempty"`
		Integrations []rdsIntegrationXML `xml:"Integrations>Integration"`
	}{
		Marker:       marker,
		Integrations: out,
	})
}

func (s *Server) handleRDSModifyIntegration(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.ModifyIntegration(rdssvc.ModifyIntegrationInput{
		Identifier: strings.TrimSpace(firstNonEmpty(r.Form.Get("IntegrationIdentifier"), r.Form.Get("Identifier"))),
		Name:       strings.TrimSpace(firstNonEmpty(r.Form.Get("IntegrationName"), r.Form.Get("Name"))),
	})
	if err != nil {
		respondRDSServiceError(w, "ModifyIntegration", err)
		return
	}
	respondRDSXML(w, "ModifyIntegration", struct {
		XMLName     xml.Name          `xml:"ModifyIntegrationResult"`
		Integration rdsIntegrationXML `xml:"Integration"`
	}{
		Integration: rdsIntegrationToXML(item),
	})
}

func (s *Server) handleRDSDeleteIntegration(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.DeleteIntegration(strings.TrimSpace(firstNonEmpty(r.Form.Get("IntegrationIdentifier"), r.Form.Get("Identifier"))))
	if err != nil {
		respondRDSServiceError(w, "DeleteIntegration", err)
		return
	}
	respondRDSXML(w, "DeleteIntegration", struct {
		XMLName     xml.Name          `xml:"DeleteIntegrationResult"`
		Integration rdsIntegrationXML `xml:"Integration"`
	}{
		Integration: rdsIntegrationToXML(item),
	})
}

func parseRDSTagMembers(values url.Values, prefix string) map[string]string {
	type partial struct {
		key   string
		value string
	}
	items := map[int]*partial{}
	for key, vals := range values {
		if len(vals) == 0 || !strings.HasPrefix(key, prefix+".") {
			continue
		}
		rest := strings.TrimPrefix(key, prefix+".")
		parts := strings.Split(rest, ".")
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		item, ok := items[idx]
		if !ok {
			item = &partial{}
			items[idx] = item
		}
		switch strings.ToLower(parts[1]) {
		case "key":
			item.key = strings.TrimSpace(vals[0])
		case "value":
			item.value = strings.TrimSpace(vals[0])
		}
	}
	indices := make([]int, 0, len(items))
	for idx := range items {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	out := map[string]string{}
	for _, idx := range indices {
		item := items[idx]
		if item == nil || item.key == "" {
			continue
		}
		out[item.key] = item.value
	}
	return out
}

func parseRDSProxyAuthMembers(values url.Values, prefix string) []rdssvc.DBProxyAuth {
	type partial struct {
		authScheme string
		secretArn  string
		iamAuth    string
	}
	authByIndex := map[int]*partial{}
	for key, vals := range values {
		if len(vals) == 0 || !strings.HasPrefix(key, prefix+".") {
			continue
		}
		rest := strings.TrimPrefix(key, prefix+".")
		parts := strings.Split(rest, ".")
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		item, ok := authByIndex[idx]
		if !ok {
			item = &partial{}
			authByIndex[idx] = item
		}
		switch strings.ToLower(parts[1]) {
		case "authscheme":
			item.authScheme = strings.TrimSpace(vals[0])
		case "secretarn":
			item.secretArn = strings.TrimSpace(vals[0])
		case "iamauth":
			item.iamAuth = strings.TrimSpace(vals[0])
		}
	}
	indices := make([]int, 0, len(authByIndex))
	for idx := range authByIndex {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	out := make([]rdssvc.DBProxyAuth, 0, len(indices))
	for _, idx := range indices {
		item := authByIndex[idx]
		if item == nil {
			continue
		}
		out = append(out, rdssvc.DBProxyAuth{
			AuthScheme: item.authScheme,
			SecretArn:  item.secretArn,
			IAMAuth:    item.iamAuth,
		})
	}
	return out
}

func rdsEventSubscriptionToXML(in rdssvc.EventSubscription) rdsEventSubscriptionXML {
	return rdsEventSubscriptionXML{
		CustSubscriptionId:       in.Name,
		EventSubscriptionArn:     in.Arn,
		SnsTopicArn:              in.SnsTopicArn,
		SourceType:               in.SourceType,
		SourceIdsList:            append([]string{}, in.SourceIDs...),
		EventCategoriesList:      append([]string{}, in.EventCategories...),
		Enabled:                  in.Enabled,
		Status:                   in.Status,
		SubscriptionCreationTime: formatRDSTime(in.CreatedAt),
	}
}

func rdsPendingMaintenanceActionToXML(in rdssvc.PendingMaintenanceAction) rdsPendingMaintenanceActionXML {
	return rdsPendingMaintenanceActionXML{
		ResourceIdentifier: in.ResourceIdentifier,
		ApplyAction:        in.ApplyAction,
		Description:        in.Description,
		OptInStatus:        in.OptInStatus,
		CurrentApplyDate:   formatRDSTime(in.CurrentApplyDate),
	}
}

func rdsEventRecordToXML(in rdssvc.EventRecord) rdsEventRecordXML {
	return rdsEventRecordXML{
		SourceIdentifier: in.SourceIdentifier,
		SourceType:       in.SourceType,
		Date:             formatRDSTime(in.Date),
		Message:          in.Message,
		EventCategories:  append([]string{}, in.EventCategories...),
	}
}

func rdsAccountAttributeToXML(in rdssvc.AccountAttribute) rdsAccountAttributeXML {
	max := ""
	if len(in.Values) > 0 {
		max = in.Values[0]
	}
	return rdsAccountAttributeXML{
		AccountQuotaName: in.Name,
		Max:              max,
	}
}

func rdsDBEngineVersionToXML(in rdssvc.DBEngineVersion) rdsDBEngineVersionXML {
	return rdsDBEngineVersionXML{
		Engine:                 in.Engine,
		EngineVersion:          in.EngineVersion,
		DBParameterGroupFamily: in.DBParameterGroupFamily,
		Status:                 in.Status,
	}
}

func rdsOrderableDBInstanceOptionToXML(in rdssvc.OrderableDBInstanceOption) rdsOrderableDBInstanceOptionXML {
	return rdsOrderableDBInstanceOptionXML{
		Engine:          in.Engine,
		EngineVersion:   in.EngineVersion,
		DBInstanceClass: in.DBInstanceClass,
		LicenseModel:    in.LicenseModel,
		Vpc:             in.Vpc,
	}
}

func rdsSourceRegionToXML(in rdssvc.SourceRegion) rdsSourceRegionXML {
	return rdsSourceRegionXML{
		RegionName:         in.RegionName,
		Endpoint:           in.Endpoint,
		Status:             in.Status,
		SupportsDBInstance: in.SupportsDBInstance,
	}
}

func rdsValidDBInstanceModificationToXML(in rdssvc.ValidDBInstanceModification) rdsValidDBInstanceModificationXML {
	return rdsValidDBInstanceModificationXML{
		DBInstanceClass: in.DBInstanceClass,
		Storage:         in.Storage,
		StorageType:     in.StorageType,
	}
}

func rdsActivityStreamToXML(in rdssvc.ActivityStream) rdsActivityStreamXML {
	return rdsActivityStreamXML{
		ResourceArn: in.ResourceArn,
		Mode:        in.Mode,
		KmsKeyId:    in.KmsKeyID,
		Status:      in.Status,
		CreateTime:  formatRDSTime(in.CreatedAt),
	}
}

func rdsDBProxyToXML(in rdssvc.DBProxy) rdsDBProxyXML {
	auth := make([]rdsDBProxyAuthXML, 0, len(in.Auth))
	for _, item := range in.Auth {
		auth = append(auth, rdsDBProxyAuthXML{
			AuthScheme: item.AuthScheme,
			SecretArn:  item.SecretArn,
			IAMAuth:    item.IAMAuth,
		})
	}
	return rdsDBProxyXML{
		DBProxyName:         in.Name,
		DBProxyArn:          in.Arn,
		EngineFamily:        in.EngineFamily,
		RoleArn:             in.RoleArn,
		VpcSubnetIds:        append([]string{}, in.VpcSubnetIDs...),
		VpcSecurityGroupIds: append([]string{}, in.VpcSecurityGroupIDs...),
		RequireTLS:          in.RequireTLS,
		IdleClientTimeout:   in.IdleClientTimeout,
		DebugLogging:        in.DebugLogging,
		Status:              in.Status,
		Auth:                auth,
		CreatedDate:         formatRDSTime(in.CreatedAt),
	}
}

func rdsDBProxyEndpointToXML(in rdssvc.DBProxyEndpoint) rdsDBProxyEndpointXML {
	return rdsDBProxyEndpointXML{
		DBProxyEndpointName: in.Name,
		DBProxyEndpointArn:  in.Arn,
		DBProxyName:         in.DBProxyName,
		VpcSubnetIds:        append([]string{}, in.VpcSubnetIDs...),
		VpcSecurityGroupIds: append([]string{}, in.VpcSecurityGroupIDs...),
		TargetRole:          in.TargetRole,
		IsDefault:           in.IsDefault,
		Status:              in.Status,
		Endpoint:            in.Endpoint,
		CreatedDate:         formatRDSTime(in.CreatedAt),
	}
}

func rdsDBProxyTargetToXML(in rdssvc.DBProxyTarget) rdsDBProxyTargetXML {
	return rdsDBProxyTargetXML{
		Type:          in.Type,
		RdsResourceId: in.RdsResourceID,
		Port:          in.Port,
		TargetHealth:  in.TargetHealth,
	}
}

func rdsIntegrationToXML(in rdssvc.Integration) rdsIntegrationXML {
	return rdsIntegrationXML{
		IntegrationIdentifier: in.Identifier,
		IntegrationArn:        in.Arn,
		IntegrationName:       in.Name,
		SourceArn:             in.SourceArn,
		TargetArn:             in.TargetArn,
		Status:                in.Status,
		CreateTime:            formatRDSTime(in.CreatedAt),
	}
}

type rdsTagXML struct {
	Key   string `xml:"Key,omitempty"`
	Value string `xml:"Value,omitempty"`
}

type rdsEventSubscriptionXML struct {
	CustSubscriptionId       string   `xml:"CustSubscriptionId,omitempty"`
	EventSubscriptionArn     string   `xml:"EventSubscriptionArn,omitempty"`
	SnsTopicArn              string   `xml:"SnsTopicArn,omitempty"`
	SourceType               string   `xml:"SourceType,omitempty"`
	SourceIdsList            []string `xml:"SourceIdsList>SourceId,omitempty"`
	EventCategoriesList      []string `xml:"EventCategoriesList>EventCategory,omitempty"`
	Enabled                  bool     `xml:"Enabled"`
	Status                   string   `xml:"Status,omitempty"`
	SubscriptionCreationTime string   `xml:"SubscriptionCreationTime,omitempty"`
}

type rdsPendingMaintenanceActionXML struct {
	ResourceIdentifier string `xml:"ResourceIdentifier,omitempty"`
	ApplyAction        string `xml:"ApplyAction,omitempty"`
	Description        string `xml:"Description,omitempty"`
	OptInStatus        string `xml:"OptInStatus,omitempty"`
	CurrentApplyDate   string `xml:"CurrentApplyDate,omitempty"`
}

type rdsEventRecordXML struct {
	SourceIdentifier string   `xml:"SourceIdentifier,omitempty"`
	SourceType       string   `xml:"SourceType,omitempty"`
	Date             string   `xml:"Date,omitempty"`
	Message          string   `xml:"Message,omitempty"`
	EventCategories  []string `xml:"EventCategories>EventCategory,omitempty"`
}

type rdsAccountAttributeXML struct {
	AccountQuotaName string `xml:"AccountQuotaName,omitempty"`
	Max              string `xml:"Max,omitempty"`
}

type rdsDBEngineVersionXML struct {
	Engine                 string `xml:"Engine,omitempty"`
	EngineVersion          string `xml:"EngineVersion,omitempty"`
	DBParameterGroupFamily string `xml:"DBParameterGroupFamily,omitempty"`
	Status                 string `xml:"Status,omitempty"`
}

type rdsOrderableDBInstanceOptionXML struct {
	Engine          string `xml:"Engine,omitempty"`
	EngineVersion   string `xml:"EngineVersion,omitempty"`
	DBInstanceClass string `xml:"DBInstanceClass,omitempty"`
	LicenseModel    string `xml:"LicenseModel,omitempty"`
	Vpc             bool   `xml:"Vpc"`
}

type rdsSourceRegionXML struct {
	RegionName         string `xml:"RegionName,omitempty"`
	Endpoint           string `xml:"Endpoint,omitempty"`
	Status             string `xml:"Status,omitempty"`
	SupportsDBInstance bool   `xml:"SupportsDBInstance"`
}

type rdsValidDBInstanceModificationsMsgXML struct {
	Items []rdsValidDBInstanceModificationXML `xml:"ValidDBInstanceModifications>ValidDBInstanceModification,omitempty"`
}

type rdsValidDBInstanceModificationXML struct {
	DBInstanceClass string `xml:"DBInstanceClass,omitempty"`
	Storage         int    `xml:"Storage,omitempty"`
	StorageType     string `xml:"StorageType,omitempty"`
}

type rdsActivityStreamXML struct {
	ResourceArn string `xml:"ResourceArn,omitempty"`
	Mode        string `xml:"Mode,omitempty"`
	KmsKeyId    string `xml:"KmsKeyId,omitempty"`
	Status      string `xml:"Status,omitempty"`
	CreateTime  string `xml:"CreateTime,omitempty"`
}

type rdsDBProxyAuthXML struct {
	AuthScheme string `xml:"AuthScheme,omitempty"`
	SecretArn  string `xml:"SecretArn,omitempty"`
	IAMAuth    string `xml:"IAMAuth,omitempty"`
}

type rdsDBProxyXML struct {
	DBProxyName         string              `xml:"DBProxyName,omitempty"`
	DBProxyArn          string              `xml:"DBProxyArn,omitempty"`
	EngineFamily        string              `xml:"EngineFamily,omitempty"`
	RoleArn             string              `xml:"RoleArn,omitempty"`
	VpcSubnetIds        []string            `xml:"VpcSubnetIds>member,omitempty"`
	VpcSecurityGroupIds []string            `xml:"VpcSecurityGroupIds>member,omitempty"`
	RequireTLS          bool                `xml:"RequireTLS"`
	IdleClientTimeout   int                 `xml:"IdleClientTimeout,omitempty"`
	DebugLogging        bool                `xml:"DebugLogging"`
	Status              string              `xml:"Status,omitempty"`
	Auth                []rdsDBProxyAuthXML `xml:"Auth>member,omitempty"`
	CreatedDate         string              `xml:"CreatedDate,omitempty"`
}

type rdsDBProxyEndpointXML struct {
	DBProxyEndpointName string   `xml:"DBProxyEndpointName,omitempty"`
	DBProxyEndpointArn  string   `xml:"DBProxyEndpointArn,omitempty"`
	DBProxyName         string   `xml:"DBProxyName,omitempty"`
	VpcSubnetIds        []string `xml:"VpcSubnetIds>member,omitempty"`
	VpcSecurityGroupIds []string `xml:"VpcSecurityGroupIds>member,omitempty"`
	TargetRole          string   `xml:"TargetRole,omitempty"`
	IsDefault           bool     `xml:"IsDefault"`
	Status              string   `xml:"Status,omitempty"`
	Endpoint            string   `xml:"Endpoint,omitempty"`
	CreatedDate         string   `xml:"CreatedDate,omitempty"`
}

type rdsDBProxyTargetXML struct {
	Type          string `xml:"Type,omitempty"`
	RdsResourceId string `xml:"RdsResourceId,omitempty"`
	Port          int    `xml:"Port,omitempty"`
	TargetHealth  string `xml:"TargetHealth,omitempty"`
}

type rdsIntegrationXML struct {
	IntegrationIdentifier string `xml:"IntegrationIdentifier,omitempty"`
	IntegrationArn        string `xml:"IntegrationArn,omitempty"`
	IntegrationName       string `xml:"IntegrationName,omitempty"`
	SourceArn             string `xml:"SourceArn,omitempty"`
	TargetArn             string `xml:"TargetArn,omitempty"`
	Status                string `xml:"Status,omitempty"`
	CreateTime            string `xml:"CreateTime,omitempty"`
}
