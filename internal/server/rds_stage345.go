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

func (s *Server) handleRDSCreateDBParameterGroup(w http.ResponseWriter, r *http.Request) {
	group, err := s.rds.CreateDBParameterGroup(rdssvc.CreateDBParameterGroupInput{
		Name:        strings.TrimSpace(r.Form.Get("DBParameterGroupName")),
		Family:      strings.TrimSpace(r.Form.Get("DBParameterGroupFamily")),
		Description: strings.TrimSpace(r.Form.Get("Description")),
	})
	if err != nil {
		respondRDSServiceError(w, "CreateDBParameterGroup", err)
		return
	}
	respondRDSXML(w, "CreateDBParameterGroup", struct {
		XMLName          xml.Name               `xml:"CreateDBParameterGroupResult"`
		DBParameterGroup rdsDBParameterGroupXML `xml:"DBParameterGroup"`
	}{
		DBParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleRDSDescribeDBParameterGroups(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	groups, marker, err := s.rds.DescribeDBParameterGroups(rdssvc.DescribeDBParameterGroupsInput{
		Name:       strings.TrimSpace(r.Form.Get("DBParameterGroupName")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBParameterGroups", err)
		return
	}
	out := make([]rdsDBParameterGroupXML, 0, len(groups))
	for _, group := range groups {
		out = append(out, rdsDBParameterGroupToXML(group))
	}
	respondRDSXML(w, "DescribeDBParameterGroups", struct {
		XMLName           xml.Name                 `xml:"DescribeDBParameterGroupsResult"`
		Marker            string                   `xml:"Marker,omitempty"`
		DBParameterGroups []rdsDBParameterGroupXML `xml:"DBParameterGroups>DBParameterGroup"`
	}{
		Marker:            marker,
		DBParameterGroups: out,
	})
}

func (s *Server) handleRDSDescribeDBParameters(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	params, marker, err := s.rds.DescribeDBParameters(rdssvc.DescribeDBParametersInput{
		GroupName:  strings.TrimSpace(r.Form.Get("DBParameterGroupName")),
		Source:     strings.TrimSpace(r.Form.Get("Source")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBParameters", err)
		return
	}
	out := make([]rdsParameterXML, 0, len(params))
	for _, param := range params {
		out = append(out, rdsParameterToXML(param))
	}
	respondRDSXML(w, "DescribeDBParameters", struct {
		XMLName    xml.Name          `xml:"DescribeDBParametersResult"`
		Marker     string            `xml:"Marker,omitempty"`
		Parameters []rdsParameterXML `xml:"Parameters>Parameter"`
	}{
		Marker:     marker,
		Parameters: out,
	})
}

func (s *Server) handleRDSModifyDBParameterGroup(w http.ResponseWriter, r *http.Request) {
	params := parseRDSParameterMembers(r.Form, "Parameters.member")
	if len(params) == 0 {
		params = parseRDSParameterMembers(r.Form, "Parameters.Parameter")
	}
	group, err := s.rds.ModifyDBParameterGroup(rdssvc.ModifyDBParameterGroupInput{
		Name:       strings.TrimSpace(r.Form.Get("DBParameterGroupName")),
		Parameters: params,
	})
	if err != nil {
		respondRDSServiceError(w, "ModifyDBParameterGroup", err)
		return
	}
	respondRDSXML(w, "ModifyDBParameterGroup", struct {
		XMLName          xml.Name               `xml:"ModifyDBParameterGroupResult"`
		DBParameterGroup rdsDBParameterGroupXML `xml:"DBParameterGroup"`
	}{
		DBParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleRDSResetDBParameterGroup(w http.ResponseWriter, r *http.Request) {
	resetAll, err := parseOptionalRDSBool(r.Form.Get("ResetAllParameters"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "ResetAllParameters is invalid")
		return
	}
	paramNames := parseRDSParameterNames(r.Form, "Parameters.member")
	if len(paramNames) == 0 {
		paramNames = parseRDSParameterNames(r.Form, "Parameters.Parameter")
	}
	group, err := s.rds.ResetDBParameterGroup(rdssvc.ResetDBParameterGroupInput{
		Name:                  strings.TrimSpace(r.Form.Get("DBParameterGroupName")),
		ResetAllParameters:    resetAll,
		ParameterNamesToReset: paramNames,
	})
	if err != nil {
		respondRDSServiceError(w, "ResetDBParameterGroup", err)
		return
	}
	respondRDSXML(w, "ResetDBParameterGroup", struct {
		XMLName          xml.Name               `xml:"ResetDBParameterGroupResult"`
		DBParameterGroup rdsDBParameterGroupXML `xml:"DBParameterGroup"`
	}{
		DBParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleRDSDeleteDBParameterGroup(w http.ResponseWriter, r *http.Request) {
	group, err := s.rds.DeleteDBParameterGroup(strings.TrimSpace(r.Form.Get("DBParameterGroupName")))
	if err != nil {
		respondRDSServiceError(w, "DeleteDBParameterGroup", err)
		return
	}
	respondRDSXML(w, "DeleteDBParameterGroup", struct {
		XMLName          xml.Name               `xml:"DeleteDBParameterGroupResult"`
		DBParameterGroup rdsDBParameterGroupXML `xml:"DBParameterGroup"`
	}{
		DBParameterGroup: rdsDBParameterGroupToXML(group),
	})
}

func (s *Server) handleRDSCreateOptionGroup(w http.ResponseWriter, r *http.Request) {
	group, err := s.rds.CreateOptionGroup(rdssvc.CreateOptionGroupInput{
		Name:               strings.TrimSpace(r.Form.Get("OptionGroupName")),
		EngineName:         strings.TrimSpace(r.Form.Get("EngineName")),
		MajorEngineVersion: strings.TrimSpace(r.Form.Get("MajorEngineVersion")),
		Description:        strings.TrimSpace(r.Form.Get("OptionGroupDescription")),
	})
	if err != nil {
		respondRDSServiceError(w, "CreateOptionGroup", err)
		return
	}
	respondRDSXML(w, "CreateOptionGroup", struct {
		XMLName     xml.Name          `xml:"CreateOptionGroupResult"`
		OptionGroup rdsOptionGroupXML `xml:"OptionGroup"`
	}{
		OptionGroup: rdsOptionGroupToXML(group),
	})
}

func (s *Server) handleRDSDescribeOptionGroups(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	groups, marker, err := s.rds.DescribeOptionGroups(rdssvc.DescribeOptionGroupsInput{
		Name:       strings.TrimSpace(firstNonEmpty(r.Form.Get("OptionGroupName"), r.Form.Get("OptionGroupNameContains"))),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeOptionGroups", err)
		return
	}
	out := make([]rdsOptionGroupXML, 0, len(groups))
	for _, group := range groups {
		out = append(out, rdsOptionGroupToXML(group))
	}
	respondRDSXML(w, "DescribeOptionGroups", struct {
		XMLName      xml.Name            `xml:"DescribeOptionGroupsResult"`
		Marker       string              `xml:"Marker,omitempty"`
		OptionGroups []rdsOptionGroupXML `xml:"OptionGroupsList>OptionGroup"`
	}{
		Marker:       marker,
		OptionGroups: out,
	})
}

func (s *Server) handleRDSModifyOptionGroup(w http.ResponseWriter, r *http.Request) {
	group, err := s.rds.ModifyOptionGroup(rdssvc.ModifyOptionGroupInput{
		Name:             strings.TrimSpace(r.Form.Get("OptionGroupName")),
		OptionsToInclude: parseRDSOptionNames(r.Form, "OptionsToInclude.member"),
		OptionsToRemove:  parseRDSOptionNames(r.Form, "OptionsToRemove.member"),
	})
	if err != nil {
		respondRDSServiceError(w, "ModifyOptionGroup", err)
		return
	}
	respondRDSXML(w, "ModifyOptionGroup", struct {
		XMLName     xml.Name          `xml:"ModifyOptionGroupResult"`
		OptionGroup rdsOptionGroupXML `xml:"OptionGroup"`
	}{
		OptionGroup: rdsOptionGroupToXML(group),
	})
}

func (s *Server) handleRDSDeleteOptionGroup(w http.ResponseWriter, r *http.Request) {
	group, err := s.rds.DeleteOptionGroup(strings.TrimSpace(r.Form.Get("OptionGroupName")))
	if err != nil {
		respondRDSServiceError(w, "DeleteOptionGroup", err)
		return
	}
	respondRDSXML(w, "DeleteOptionGroup", struct {
		XMLName     xml.Name          `xml:"DeleteOptionGroupResult"`
		OptionGroup rdsOptionGroupXML `xml:"OptionGroup"`
	}{
		OptionGroup: rdsOptionGroupToXML(group),
	})
}

func (s *Server) handleRDSCreateDBSubnetGroup(w http.ResponseWriter, r *http.Request) {
	subnetIDs := parseRDSListMembers(r.Form, "SubnetIds.member")
	if len(subnetIDs) == 0 {
		subnetIDs = parseRDSListMembers(r.Form, "SubnetIds.SubnetIdentifier")
	}
	group, err := s.rds.CreateDBSubnetGroup(rdssvc.CreateDBSubnetGroupInput{
		Name:        strings.TrimSpace(r.Form.Get("DBSubnetGroupName")),
		Description: strings.TrimSpace(r.Form.Get("DBSubnetGroupDescription")),
		SubnetIDs:   subnetIDs,
	})
	if err != nil {
		respondRDSServiceError(w, "CreateDBSubnetGroup", err)
		return
	}
	respondRDSXML(w, "CreateDBSubnetGroup", struct {
		XMLName       xml.Name                `xml:"CreateDBSubnetGroupResult"`
		DBSubnetGroup rdsDBSubnetGroupItemXML `xml:"DBSubnetGroup"`
	}{
		DBSubnetGroup: rdsDBSubnetGroupToXML(group),
	})
}

func (s *Server) handleRDSDescribeDBSubnetGroups(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	groups, marker, err := s.rds.DescribeDBSubnetGroups(rdssvc.DescribeDBSubnetGroupsInput{
		Name:       strings.TrimSpace(r.Form.Get("DBSubnetGroupName")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBSubnetGroups", err)
		return
	}
	out := make([]rdsDBSubnetGroupItemXML, 0, len(groups))
	for _, group := range groups {
		out = append(out, rdsDBSubnetGroupToXML(group))
	}
	respondRDSXML(w, "DescribeDBSubnetGroups", struct {
		XMLName        xml.Name                  `xml:"DescribeDBSubnetGroupsResult"`
		Marker         string                    `xml:"Marker,omitempty"`
		DBSubnetGroups []rdsDBSubnetGroupItemXML `xml:"DBSubnetGroups>DBSubnetGroup"`
	}{
		Marker:         marker,
		DBSubnetGroups: out,
	})
}

func (s *Server) handleRDSModifyDBSubnetGroup(w http.ResponseWriter, r *http.Request) {
	subnetIDs := parseRDSListMembers(r.Form, "SubnetIds.member")
	if len(subnetIDs) == 0 {
		subnetIDs = parseRDSListMembers(r.Form, "SubnetIds.SubnetIdentifier")
	}
	group, err := s.rds.ModifyDBSubnetGroup(rdssvc.ModifyDBSubnetGroupInput{
		Name:        strings.TrimSpace(r.Form.Get("DBSubnetGroupName")),
		Description: strings.TrimSpace(r.Form.Get("DBSubnetGroupDescription")),
		SubnetIDs:   subnetIDs,
	})
	if err != nil {
		respondRDSServiceError(w, "ModifyDBSubnetGroup", err)
		return
	}
	respondRDSXML(w, "ModifyDBSubnetGroup", struct {
		XMLName       xml.Name                `xml:"ModifyDBSubnetGroupResult"`
		DBSubnetGroup rdsDBSubnetGroupItemXML `xml:"DBSubnetGroup"`
	}{
		DBSubnetGroup: rdsDBSubnetGroupToXML(group),
	})
}

func (s *Server) handleRDSDeleteDBSubnetGroup(w http.ResponseWriter, r *http.Request) {
	group, err := s.rds.DeleteDBSubnetGroup(strings.TrimSpace(r.Form.Get("DBSubnetGroupName")))
	if err != nil {
		respondRDSServiceError(w, "DeleteDBSubnetGroup", err)
		return
	}
	respondRDSXML(w, "DeleteDBSubnetGroup", struct {
		XMLName       xml.Name                `xml:"DeleteDBSubnetGroupResult"`
		DBSubnetGroup rdsDBSubnetGroupItemXML `xml:"DBSubnetGroup"`
	}{
		DBSubnetGroup: rdsDBSubnetGroupToXML(group),
	})
}

func (s *Server) handleRDSCreateDBSecurityGroup(w http.ResponseWriter, r *http.Request) {
	group, err := s.rds.CreateDBSecurityGroup(rdssvc.CreateDBSecurityGroupInput{
		Name:        strings.TrimSpace(r.Form.Get("DBSecurityGroupName")),
		Description: strings.TrimSpace(r.Form.Get("DBSecurityGroupDescription")),
	})
	if err != nil {
		respondRDSServiceError(w, "CreateDBSecurityGroup", err)
		return
	}
	respondRDSXML(w, "CreateDBSecurityGroup", struct {
		XMLName         xml.Name              `xml:"CreateDBSecurityGroupResult"`
		DBSecurityGroup rdsDBSecurityGroupXML `xml:"DBSecurityGroup"`
	}{
		DBSecurityGroup: rdsDBSecurityGroupToXML(group),
	})
}

func (s *Server) handleRDSDescribeDBSecurityGroups(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	groups, marker, err := s.rds.DescribeDBSecurityGroups(rdssvc.DescribeDBSecurityGroupsInput{
		Name:       strings.TrimSpace(r.Form.Get("DBSecurityGroupName")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBSecurityGroups", err)
		return
	}
	out := make([]rdsDBSecurityGroupXML, 0, len(groups))
	for _, group := range groups {
		out = append(out, rdsDBSecurityGroupToXML(group))
	}
	respondRDSXML(w, "DescribeDBSecurityGroups", struct {
		XMLName          xml.Name                `xml:"DescribeDBSecurityGroupsResult"`
		Marker           string                  `xml:"Marker,omitempty"`
		DBSecurityGroups []rdsDBSecurityGroupXML `xml:"DBSecurityGroups>DBSecurityGroup"`
	}{
		Marker:           marker,
		DBSecurityGroups: out,
	})
}

func (s *Server) handleRDSAuthorizeDBSecurityGroupIngress(w http.ResponseWriter, r *http.Request) {
	group, err := s.rds.AuthorizeDBSecurityGroupIngress(rdssvc.IngressRuleInput{
		DBSecurityGroupName:     strings.TrimSpace(r.Form.Get("DBSecurityGroupName")),
		CIDRIP:                  strings.TrimSpace(r.Form.Get("CIDRIP")),
		EC2SecurityGroupName:    strings.TrimSpace(r.Form.Get("EC2SecurityGroupName")),
		EC2SecurityGroupOwnerID: strings.TrimSpace(r.Form.Get("EC2SecurityGroupOwnerId")),
	})
	if err != nil {
		respondRDSServiceError(w, "AuthorizeDBSecurityGroupIngress", err)
		return
	}
	respondRDSXML(w, "AuthorizeDBSecurityGroupIngress", struct {
		XMLName         xml.Name              `xml:"AuthorizeDBSecurityGroupIngressResult"`
		DBSecurityGroup rdsDBSecurityGroupXML `xml:"DBSecurityGroup"`
	}{
		DBSecurityGroup: rdsDBSecurityGroupToXML(group),
	})
}

func (s *Server) handleRDSRevokeDBSecurityGroupIngress(w http.ResponseWriter, r *http.Request) {
	group, err := s.rds.RevokeDBSecurityGroupIngress(rdssvc.IngressRuleInput{
		DBSecurityGroupName:     strings.TrimSpace(r.Form.Get("DBSecurityGroupName")),
		CIDRIP:                  strings.TrimSpace(r.Form.Get("CIDRIP")),
		EC2SecurityGroupName:    strings.TrimSpace(r.Form.Get("EC2SecurityGroupName")),
		EC2SecurityGroupOwnerID: strings.TrimSpace(r.Form.Get("EC2SecurityGroupOwnerId")),
	})
	if err != nil {
		respondRDSServiceError(w, "RevokeDBSecurityGroupIngress", err)
		return
	}
	respondRDSXML(w, "RevokeDBSecurityGroupIngress", struct {
		XMLName         xml.Name              `xml:"RevokeDBSecurityGroupIngressResult"`
		DBSecurityGroup rdsDBSecurityGroupXML `xml:"DBSecurityGroup"`
	}{
		DBSecurityGroup: rdsDBSecurityGroupToXML(group),
	})
}

func (s *Server) handleRDSDeleteDBSecurityGroup(w http.ResponseWriter, r *http.Request) {
	group, err := s.rds.DeleteDBSecurityGroup(strings.TrimSpace(r.Form.Get("DBSecurityGroupName")))
	if err != nil {
		respondRDSServiceError(w, "DeleteDBSecurityGroup", err)
		return
	}
	respondRDSXML(w, "DeleteDBSecurityGroup", struct {
		XMLName         xml.Name              `xml:"DeleteDBSecurityGroupResult"`
		DBSecurityGroup rdsDBSecurityGroupXML `xml:"DBSecurityGroup"`
	}{
		DBSecurityGroup: rdsDBSecurityGroupToXML(group),
	})
}

func (s *Server) handleRDSDescribeCertificates(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	certs, marker, err := s.rds.DescribeCertificates(rdssvc.DescribeCertificatesInput{
		Identifier: strings.TrimSpace(r.Form.Get("CertificateIdentifier")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeCertificates", err)
		return
	}
	out := make([]rdsCertificateXML, 0, len(certs))
	for _, cert := range certs {
		out = append(out, rdsCertificateToXML(cert))
	}
	respondRDSXML(w, "DescribeCertificates", struct {
		XMLName      xml.Name            `xml:"DescribeCertificatesResult"`
		Marker       string              `xml:"Marker,omitempty"`
		Certificates []rdsCertificateXML `xml:"Certificates>Certificate"`
	}{
		Marker:       marker,
		Certificates: out,
	})
}

func (s *Server) handleRDSCreateDBCluster(w http.ResponseWriter, r *http.Request) {
	backupRetention, err := parseOptionalRDSInt(r.Form.Get("BackupRetentionPeriod"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "BackupRetentionPeriod is invalid")
		return
	}
	cluster, err := s.rds.CreateDBCluster(rdssvc.CreateDBClusterInput{
		Identifier:              strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		Engine:                  strings.TrimSpace(r.Form.Get("Engine")),
		MasterUsername:          strings.TrimSpace(r.Form.Get("MasterUsername")),
		MasterUserPassword:      strings.TrimSpace(r.Form.Get("MasterUserPassword")),
		DatabaseName:            strings.TrimSpace(r.Form.Get("DatabaseName")),
		DBSubnetGroupName:       strings.TrimSpace(r.Form.Get("DBSubnetGroupName")),
		DBClusterParameterGroup: strings.TrimSpace(r.Form.Get("DBClusterParameterGroupName")),
		VpcSecurityGroupIDs:     parseRDSListMembers(r.Form, "VpcSecurityGroupIds.member"),
		BackupRetentionPeriod:   backupRetention,
	})
	if err != nil {
		respondRDSServiceError(w, "CreateDBCluster", err)
		return
	}
	respondRDSXML(w, "CreateDBCluster", struct {
		XMLName   xml.Name        `xml:"CreateDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleRDSDescribeDBClusters(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	clusters, marker, err := s.rds.DescribeDBClusters(rdssvc.DescribeDBClustersInput{
		Identifier: strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBClusters", err)
		return
	}
	out := make([]rdsDBClusterXML, 0, len(clusters))
	for _, cluster := range clusters {
		out = append(out, rdsDBClusterToXML(cluster))
	}
	respondRDSXML(w, "DescribeDBClusters", struct {
		XMLName    xml.Name          `xml:"DescribeDBClustersResult"`
		Marker     string            `xml:"Marker,omitempty"`
		DBClusters []rdsDBClusterXML `xml:"DBClusters>DBCluster"`
	}{
		Marker:     marker,
		DBClusters: out,
	})
}

func (s *Server) handleRDSModifyDBCluster(w http.ResponseWriter, r *http.Request) {
	backupRetention, err := parseOptionalRDSInt(r.Form.Get("BackupRetentionPeriod"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "BackupRetentionPeriod is invalid")
		return
	}
	cluster, err := s.rds.ModifyDBCluster(rdssvc.ModifyDBClusterInput{
		Identifier:              strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		BackupRetentionPeriod:   backupRetention,
		DBClusterParameterGroup: strings.TrimSpace(r.Form.Get("DBClusterParameterGroupName")),
	})
	if err != nil {
		respondRDSServiceError(w, "ModifyDBCluster", err)
		return
	}
	respondRDSXML(w, "ModifyDBCluster", struct {
		XMLName   xml.Name        `xml:"ModifyDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleRDSDeleteDBCluster(w http.ResponseWriter, r *http.Request) {
	skipFinalSnapshot, err := parseOptionalRDSBool(r.Form.Get("SkipFinalSnapshot"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "SkipFinalSnapshot is invalid")
		return
	}
	cluster, err := s.rds.DeleteDBCluster(rdssvc.DeleteDBClusterInput{
		Identifier:                strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		SkipFinalSnapshot:         skipFinalSnapshot,
		FinalDBSnapshotIdentifier: strings.TrimSpace(r.Form.Get("FinalDBSnapshotIdentifier")),
	})
	if err != nil {
		respondRDSServiceError(w, "DeleteDBCluster", err)
		return
	}
	respondRDSXML(w, "DeleteDBCluster", struct {
		XMLName   xml.Name        `xml:"DeleteDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleRDSStartDBCluster(w http.ResponseWriter, r *http.Request) {
	cluster, err := s.rds.StartDBCluster(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "StartDBCluster", err)
		return
	}
	respondRDSXML(w, "StartDBCluster", struct {
		XMLName   xml.Name        `xml:"StartDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleRDSStopDBCluster(w http.ResponseWriter, r *http.Request) {
	cluster, err := s.rds.StopDBCluster(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "StopDBCluster", err)
		return
	}
	respondRDSXML(w, "StopDBCluster", struct {
		XMLName   xml.Name        `xml:"StopDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleRDSRebootDBCluster(w http.ResponseWriter, r *http.Request) {
	cluster, err := s.rds.RebootDBCluster(strings.TrimSpace(r.Form.Get("DBClusterIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "RebootDBCluster", err)
		return
	}
	respondRDSXML(w, "RebootDBCluster", struct {
		XMLName   xml.Name        `xml:"RebootDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleRDSFailoverDBCluster(w http.ResponseWriter, r *http.Request) {
	cluster, err := s.rds.FailoverDBCluster(rdssvc.FailoverDBClusterInput{
		Identifier:                 strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		TargetDBInstanceIdentifier: strings.TrimSpace(r.Form.Get("TargetDBInstanceIdentifier")),
	})
	if err != nil {
		respondRDSServiceError(w, "FailoverDBCluster", err)
		return
	}
	respondRDSXML(w, "FailoverDBCluster", struct {
		XMLName   xml.Name        `xml:"FailoverDBClusterResult"`
		DBCluster rdsDBClusterXML `xml:"DBCluster"`
	}{
		DBCluster: rdsDBClusterToXML(cluster),
	})
}

func (s *Server) handleRDSCreateDBClusterEndpoint(w http.ResponseWriter, r *http.Request) {
	endpoint, err := s.rds.CreateDBClusterEndpoint(rdssvc.CreateDBClusterEndpointInput{
		Identifier:        strings.TrimSpace(r.Form.Get("DBClusterEndpointIdentifier")),
		ClusterIdentifier: strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		EndpointType:      strings.TrimSpace(r.Form.Get("EndpointType")),
		StaticMembers:     parseRDSListMembers(r.Form, "StaticMembers.member"),
		ExcludedMembers:   parseRDSListMembers(r.Form, "ExcludedMembers.member"),
	})
	if err != nil {
		respondRDSServiceError(w, "CreateDBClusterEndpoint", err)
		return
	}
	respondRDSXML(w, "CreateDBClusterEndpoint", struct {
		XMLName           xml.Name                `xml:"CreateDBClusterEndpointResult"`
		DBClusterEndpoint rdsDBClusterEndpointXML `xml:"DBClusterEndpoint"`
	}{
		DBClusterEndpoint: rdsDBClusterEndpointToXML(endpoint),
	})
}

func (s *Server) handleRDSDescribeDBClusterEndpoints(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeDBClusterEndpoints(rdssvc.DescribeDBClusterEndpointsInput{
		Identifier:        strings.TrimSpace(r.Form.Get("DBClusterEndpointIdentifier")),
		ClusterIdentifier: strings.TrimSpace(r.Form.Get("DBClusterIdentifier")),
		MaxRecords:        maxRecords,
		Marker:            strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeDBClusterEndpoints", err)
		return
	}
	out := make([]rdsDBClusterEndpointXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsDBClusterEndpointToXML(item))
	}
	respondRDSXML(w, "DescribeDBClusterEndpoints", struct {
		XMLName            xml.Name                  `xml:"DescribeDBClusterEndpointsResult"`
		Marker             string                    `xml:"Marker,omitempty"`
		DBClusterEndpoints []rdsDBClusterEndpointXML `xml:"DBClusterEndpoints>DBClusterEndpoint"`
	}{
		Marker:             marker,
		DBClusterEndpoints: out,
	})
}

func (s *Server) handleRDSModifyDBClusterEndpoint(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.ModifyDBClusterEndpoint(rdssvc.ModifyDBClusterEndpointInput{
		Identifier:      strings.TrimSpace(r.Form.Get("DBClusterEndpointIdentifier")),
		EndpointType:    strings.TrimSpace(r.Form.Get("EndpointType")),
		StaticMembers:   parseRDSListMembers(r.Form, "StaticMembers.member"),
		ExcludedMembers: parseRDSListMembers(r.Form, "ExcludedMembers.member"),
	})
	if err != nil {
		respondRDSServiceError(w, "ModifyDBClusterEndpoint", err)
		return
	}
	respondRDSXML(w, "ModifyDBClusterEndpoint", struct {
		XMLName           xml.Name                `xml:"ModifyDBClusterEndpointResult"`
		DBClusterEndpoint rdsDBClusterEndpointXML `xml:"DBClusterEndpoint"`
	}{
		DBClusterEndpoint: rdsDBClusterEndpointToXML(item),
	})
}

func (s *Server) handleRDSDeleteDBClusterEndpoint(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.DeleteDBClusterEndpoint(strings.TrimSpace(r.Form.Get("DBClusterEndpointIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "DeleteDBClusterEndpoint", err)
		return
	}
	respondRDSXML(w, "DeleteDBClusterEndpoint", struct {
		XMLName           xml.Name                `xml:"DeleteDBClusterEndpointResult"`
		DBClusterEndpoint rdsDBClusterEndpointXML `xml:"DBClusterEndpoint"`
	}{
		DBClusterEndpoint: rdsDBClusterEndpointToXML(item),
	})
}

func (s *Server) handleRDSCreateGlobalCluster(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.CreateGlobalCluster(rdssvc.CreateGlobalClusterInput{
		Identifier:         strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")),
		SourceDBClusterArn: strings.TrimSpace(firstNonEmpty(r.Form.Get("SourceDBClusterArn"), r.Form.Get("SourceDBClusterIdentifier"))),
		EngineVersion:      strings.TrimSpace(r.Form.Get("EngineVersion")),
	})
	if err != nil {
		respondRDSServiceError(w, "CreateGlobalCluster", err)
		return
	}
	respondRDSXML(w, "CreateGlobalCluster", struct {
		XMLName       xml.Name            `xml:"CreateGlobalClusterResult"`
		GlobalCluster rdsGlobalClusterXML `xml:"GlobalCluster"`
	}{
		GlobalCluster: rdsGlobalClusterToXML(item),
	})
}

func (s *Server) handleRDSDescribeGlobalClusters(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeGlobalClusters(rdssvc.DescribeGlobalClustersInput{
		Identifier: strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeGlobalClusters", err)
		return
	}
	out := make([]rdsGlobalClusterXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsGlobalClusterToXML(item))
	}
	respondRDSXML(w, "DescribeGlobalClusters", struct {
		XMLName        xml.Name              `xml:"DescribeGlobalClustersResult"`
		Marker         string                `xml:"Marker,omitempty"`
		GlobalClusters []rdsGlobalClusterXML `xml:"GlobalClusters>GlobalCluster"`
	}{
		Marker:         marker,
		GlobalClusters: out,
	})
}

func (s *Server) handleRDSModifyGlobalCluster(w http.ResponseWriter, r *http.Request) {
	deletionProtection, err := parseOptionalRDSBoolPtr(r.Form.Get("DeletionProtection"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "DeletionProtection is invalid")
		return
	}
	item, err := s.rds.ModifyGlobalCluster(rdssvc.ModifyGlobalClusterInput{
		Identifier:         strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")),
		DeletionProtection: deletionProtection,
		EngineVersion:      strings.TrimSpace(r.Form.Get("EngineVersion")),
	})
	if err != nil {
		respondRDSServiceError(w, "ModifyGlobalCluster", err)
		return
	}
	respondRDSXML(w, "ModifyGlobalCluster", struct {
		XMLName       xml.Name            `xml:"ModifyGlobalClusterResult"`
		GlobalCluster rdsGlobalClusterXML `xml:"GlobalCluster"`
	}{
		GlobalCluster: rdsGlobalClusterToXML(item),
	})
}

func (s *Server) handleRDSDeleteGlobalCluster(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.DeleteGlobalCluster(strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "DeleteGlobalCluster", err)
		return
	}
	respondRDSXML(w, "DeleteGlobalCluster", struct {
		XMLName       xml.Name            `xml:"DeleteGlobalClusterResult"`
		GlobalCluster rdsGlobalClusterXML `xml:"GlobalCluster"`
	}{
		GlobalCluster: rdsGlobalClusterToXML(item),
	})
}

func (s *Server) handleRDSFailoverGlobalCluster(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.FailoverGlobalCluster(rdssvc.FailoverGlobalClusterInput{
		Identifier:         strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")),
		TargetDBClusterArn: strings.TrimSpace(firstNonEmpty(r.Form.Get("TargetDbClusterIdentifier"), r.Form.Get("TargetDBClusterIdentifier"), r.Form.Get("TargetDbClusterArn"), r.Form.Get("TargetDBClusterArn"))),
	})
	if err != nil {
		respondRDSServiceError(w, "FailoverGlobalCluster", err)
		return
	}
	respondRDSXML(w, "FailoverGlobalCluster", struct {
		XMLName       xml.Name            `xml:"FailoverGlobalClusterResult"`
		GlobalCluster rdsGlobalClusterXML `xml:"GlobalCluster"`
	}{
		GlobalCluster: rdsGlobalClusterToXML(item),
	})
}

func (s *Server) handleRDSSwitchoverGlobalCluster(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.SwitchoverGlobalCluster(rdssvc.FailoverGlobalClusterInput{
		Identifier:         strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier")),
		TargetDBClusterArn: strings.TrimSpace(firstNonEmpty(r.Form.Get("TargetDbClusterIdentifier"), r.Form.Get("TargetDBClusterIdentifier"), r.Form.Get("TargetDbClusterArn"), r.Form.Get("TargetDBClusterArn"))),
	})
	if err != nil {
		respondRDSServiceError(w, "SwitchoverGlobalCluster", err)
		return
	}
	respondRDSXML(w, "SwitchoverGlobalCluster", struct {
		XMLName       xml.Name            `xml:"SwitchoverGlobalClusterResult"`
		GlobalCluster rdsGlobalClusterXML `xml:"GlobalCluster"`
	}{
		GlobalCluster: rdsGlobalClusterToXML(item),
	})
}

func (s *Server) handleRDSCreateDBInstanceReadReplica(w http.ResponseWriter, r *http.Request) {
	publiclyAccessible, err := parseOptionalRDSBool(r.Form.Get("PubliclyAccessible"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "PubliclyAccessible is invalid")
		return
	}
	instance, err := s.rds.CreateDBInstanceReadReplica(rdssvc.CreateDBInstanceReadReplicaInput{
		Identifier:         strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")),
		SourceIdentifier:   strings.TrimSpace(r.Form.Get("SourceDBInstanceIdentifier")),
		DBInstanceClass:    strings.TrimSpace(r.Form.Get("DBInstanceClass")),
		PubliclyAccessible: publiclyAccessible,
	})
	if err != nil {
		respondRDSServiceError(w, "CreateDBInstanceReadReplica", err)
		return
	}
	respondRDSXML(w, "CreateDBInstanceReadReplica", struct {
		XMLName    xml.Name         `xml:"CreateDBInstanceReadReplicaResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleRDSPromoteReadReplica(w http.ResponseWriter, r *http.Request) {
	instance, err := s.rds.PromoteReadReplica(strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "PromoteReadReplica", err)
		return
	}
	respondRDSXML(w, "PromoteReadReplica", struct {
		XMLName    xml.Name         `xml:"PromoteReadReplicaResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleRDSSwitchoverReadReplica(w http.ResponseWriter, r *http.Request) {
	instance, err := s.rds.SwitchoverReadReplica(strings.TrimSpace(r.Form.Get("DBInstanceIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "SwitchoverReadReplica", err)
		return
	}
	respondRDSXML(w, "SwitchoverReadReplica", struct {
		XMLName    xml.Name         `xml:"SwitchoverReadReplicaResult"`
		DBInstance rdsDBInstanceXML `xml:"DBInstance"`
	}{
		DBInstance: rdsDBInstanceToXML(instance),
	})
}

func (s *Server) handleRDSCreateBlueGreenDeployment(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.CreateBlueGreenDeployment(rdssvc.CreateBlueGreenDeploymentInput{
		Name:                strings.TrimSpace(r.Form.Get("BlueGreenDeploymentName")),
		Source:              strings.TrimSpace(firstNonEmpty(r.Form.Get("Source"), r.Form.Get("SourceArn"))),
		TargetEngineVersion: strings.TrimSpace(r.Form.Get("TargetEngineVersion")),
	})
	if err != nil {
		respondRDSServiceError(w, "CreateBlueGreenDeployment", err)
		return
	}
	respondRDSXML(w, "CreateBlueGreenDeployment", struct {
		XMLName             xml.Name                  `xml:"CreateBlueGreenDeploymentResult"`
		BlueGreenDeployment rdsBlueGreenDeploymentXML `xml:"BlueGreenDeployment"`
	}{
		BlueGreenDeployment: rdsBlueGreenDeploymentToXML(item),
	})
}

func (s *Server) handleRDSDescribeBlueGreenDeployments(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeBlueGreenDeployments(rdssvc.DescribeBlueGreenDeploymentsInput{
		Identifier: strings.TrimSpace(r.Form.Get("BlueGreenDeploymentIdentifier")),
		Name:       strings.TrimSpace(r.Form.Get("BlueGreenDeploymentName")),
		MaxRecords: maxRecords,
		Marker:     strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeBlueGreenDeployments", err)
		return
	}
	out := make([]rdsBlueGreenDeploymentXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsBlueGreenDeploymentToXML(item))
	}
	respondRDSXML(w, "DescribeBlueGreenDeployments", struct {
		XMLName              xml.Name                    `xml:"DescribeBlueGreenDeploymentsResult"`
		Marker               string                      `xml:"Marker,omitempty"`
		BlueGreenDeployments []rdsBlueGreenDeploymentXML `xml:"BlueGreenDeployments>BlueGreenDeployment"`
	}{
		Marker:               marker,
		BlueGreenDeployments: out,
	})
}

func (s *Server) handleRDSSwitchoverBlueGreenDeployment(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.SwitchoverBlueGreenDeployment(strings.TrimSpace(r.Form.Get("BlueGreenDeploymentIdentifier")))
	if err != nil {
		respondRDSServiceError(w, "SwitchoverBlueGreenDeployment", err)
		return
	}
	respondRDSXML(w, "SwitchoverBlueGreenDeployment", struct {
		XMLName             xml.Name                  `xml:"SwitchoverBlueGreenDeploymentResult"`
		BlueGreenDeployment rdsBlueGreenDeploymentXML `xml:"BlueGreenDeployment"`
	}{
		BlueGreenDeployment: rdsBlueGreenDeploymentToXML(item),
	})
}

func (s *Server) handleRDSDeleteBlueGreenDeployment(w http.ResponseWriter, r *http.Request) {
	deleteTarget, err := parseOptionalRDSBool(r.Form.Get("DeleteTarget"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "DeleteTarget is invalid")
		return
	}
	item, err := s.rds.DeleteBlueGreenDeployment(rdssvc.DeleteBlueGreenDeploymentInput{
		Identifier:   strings.TrimSpace(r.Form.Get("BlueGreenDeploymentIdentifier")),
		DeleteTarget: deleteTarget,
	})
	if err != nil {
		respondRDSServiceError(w, "DeleteBlueGreenDeployment", err)
		return
	}
	respondRDSXML(w, "DeleteBlueGreenDeployment", struct {
		XMLName             xml.Name                  `xml:"DeleteBlueGreenDeploymentResult"`
		BlueGreenDeployment rdsBlueGreenDeploymentXML `xml:"BlueGreenDeployment"`
	}{
		BlueGreenDeployment: rdsBlueGreenDeploymentToXML(item),
	})
}

func (s *Server) handleRDSCreateTenantDatabase(w http.ResponseWriter, r *http.Request) {
	clusterID := strings.TrimSpace(firstNonEmpty(r.Form.Get("DBClusterIdentifier"), r.Form.Get("DBInstanceIdentifier")))
	item, err := s.rds.CreateTenantDatabase(rdssvc.CreateTenantDatabaseInput{
		ClusterIdentifier:  clusterID,
		TenantIdentifier:   strings.TrimSpace(r.Form.Get("TenantDBName")),
		MasterUsername:     strings.TrimSpace(r.Form.Get("MasterUsername")),
		MasterUserPassword: strings.TrimSpace(r.Form.Get("MasterUserPassword")),
	})
	if err != nil {
		respondRDSServiceError(w, "CreateTenantDatabase", err)
		return
	}
	respondRDSXML(w, "CreateTenantDatabase", struct {
		XMLName        xml.Name             `xml:"CreateTenantDatabaseResult"`
		TenantDatabase rdsTenantDatabaseXML `xml:"TenantDatabase"`
	}{
		TenantDatabase: rdsTenantDatabaseToXML(item),
	})
}

func (s *Server) handleRDSDescribeTenantDatabases(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeTenantDatabases(rdssvc.DescribeTenantDatabasesInput{
		ClusterIdentifier: strings.TrimSpace(firstNonEmpty(r.Form.Get("DBClusterIdentifier"), r.Form.Get("DBInstanceIdentifier"))),
		TenantIdentifier:  strings.TrimSpace(r.Form.Get("TenantDBName")),
		MaxRecords:        maxRecords,
		Marker:            strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeTenantDatabases", err)
		return
	}
	out := make([]rdsTenantDatabaseXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsTenantDatabaseToXML(item))
	}
	respondRDSXML(w, "DescribeTenantDatabases", struct {
		XMLName         xml.Name               `xml:"DescribeTenantDatabasesResult"`
		Marker          string                 `xml:"Marker,omitempty"`
		TenantDatabases []rdsTenantDatabaseXML `xml:"TenantDatabases>TenantDatabase"`
	}{
		Marker:          marker,
		TenantDatabases: out,
	})
}

func (s *Server) handleRDSModifyTenantDatabase(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.ModifyTenantDatabase(rdssvc.ModifyTenantDatabaseInput{
		ClusterIdentifier:   strings.TrimSpace(firstNonEmpty(r.Form.Get("DBClusterIdentifier"), r.Form.Get("DBInstanceIdentifier"))),
		TenantIdentifier:    strings.TrimSpace(r.Form.Get("TenantDBName")),
		NewTenantIdentifier: strings.TrimSpace(r.Form.Get("NewTenantDBName")),
	})
	if err != nil {
		respondRDSServiceError(w, "ModifyTenantDatabase", err)
		return
	}
	respondRDSXML(w, "ModifyTenantDatabase", struct {
		XMLName        xml.Name             `xml:"ModifyTenantDatabaseResult"`
		TenantDatabase rdsTenantDatabaseXML `xml:"TenantDatabase"`
	}{
		TenantDatabase: rdsTenantDatabaseToXML(item),
	})
}

func (s *Server) handleRDSDeleteTenantDatabase(w http.ResponseWriter, r *http.Request) {
	item, err := s.rds.DeleteTenantDatabase(rdssvc.DeleteTenantDatabaseInput{
		ClusterIdentifier: strings.TrimSpace(firstNonEmpty(r.Form.Get("DBClusterIdentifier"), r.Form.Get("DBInstanceIdentifier"))),
		TenantIdentifier:  strings.TrimSpace(r.Form.Get("TenantDBName")),
	})
	if err != nil {
		respondRDSServiceError(w, "DeleteTenantDatabase", err)
		return
	}
	respondRDSXML(w, "DeleteTenantDatabase", struct {
		XMLName        xml.Name             `xml:"DeleteTenantDatabaseResult"`
		TenantDatabase rdsTenantDatabaseXML `xml:"TenantDatabase"`
	}{
		TenantDatabase: rdsTenantDatabaseToXML(item),
	})
}

func parseRDSListMembers(values url.Values, prefix string) []string {
	type item struct {
		index int
		value string
	}
	items := make([]item, 0)
	base := prefix
	if idx := strings.LastIndex(base, "."); idx > 0 {
		base = base[:idx]
	}
	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}
		index := -1
		if strings.HasPrefix(key, prefix+".") {
			if parsed, err := strconv.Atoi(strings.TrimPrefix(key, prefix+".")); err == nil {
				index = parsed
			}
		}
		if index == -1 && strings.HasPrefix(key, base+".") {
			rest := strings.TrimPrefix(key, base+".")
			parts := strings.Split(rest, ".")
			if len(parts) >= 2 {
				if parsed, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
					index = parsed
				}
			}
		}
		if index == -1 {
			continue
		}
		value := strings.TrimSpace(vals[0])
		if value == "" {
			continue
		}
		items = append(items, item{index: index, value: value})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].index < items[j].index })
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.value)
	}
	return out
}

func parseRDSParameterMembers(values url.Values, prefix string) []rdssvc.Parameter {
	type partial struct {
		name        string
		value       string
		applyMethod string
	}
	params := map[int]*partial{}
	prefixes := []string{prefix}
	if strings.HasSuffix(prefix, ".member") {
		prefixes = append(prefixes, strings.TrimSuffix(prefix, ".member")+".Parameter")
	}
	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}
		rest := ""
		for _, currentPrefix := range prefixes {
			if strings.HasPrefix(key, currentPrefix+".") {
				rest = strings.TrimPrefix(key, currentPrefix+".")
				break
			}
		}
		if rest == "" {
			continue
		}
		parts := strings.Split(rest, ".")
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		item, ok := params[idx]
		if !ok {
			item = &partial{}
			params[idx] = item
		}
		switch strings.ToLower(parts[1]) {
		case "parametername":
			item.name = strings.TrimSpace(vals[0])
		case "parametervalue":
			item.value = strings.TrimSpace(vals[0])
		case "applymethod":
			item.applyMethod = strings.TrimSpace(vals[0])
		}
	}
	indices := make([]int, 0, len(params))
	for idx := range params {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	out := make([]rdssvc.Parameter, 0, len(indices))
	for _, idx := range indices {
		item := params[idx]
		if item == nil || item.name == "" {
			continue
		}
		out = append(out, rdssvc.Parameter{Name: item.name, Value: item.value, ApplyMethod: item.applyMethod})
	}
	return out
}

func parseRDSParameterNames(values url.Values, prefix string) []string {
	params := parseRDSParameterMembers(values, prefix)
	out := make([]string, 0, len(params))
	for _, param := range params {
		if strings.TrimSpace(param.Name) != "" {
			out = append(out, strings.TrimSpace(param.Name))
		}
	}
	return out
}

func parseRDSOptionNames(values url.Values, prefix string) []string {
	type optItem struct {
		index int
		name  string
	}
	items := make([]optItem, 0)
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
		if strings.ToLower(parts[1]) != "optionname" {
			continue
		}
		name := strings.TrimSpace(vals[0])
		if name != "" {
			items = append(items, optItem{index: idx, name: name})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].index < items[j].index })
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.name)
	}
	return out
}

func rdsDBParameterGroupToXML(in rdssvc.DBParameterGroup) rdsDBParameterGroupXML {
	return rdsDBParameterGroupXML{
		DBParameterGroupName:   in.Name,
		DBParameterGroupFamily: in.Family,
		Description:            in.Description,
	}
}

func rdsParameterToXML(in rdssvc.Parameter) rdsParameterXML {
	return rdsParameterXML{
		ParameterName:  in.Name,
		ParameterValue: in.Value,
		ApplyMethod:    firstNonEmpty(in.ApplyMethod, "immediate"),
		ApplyType:      "dynamic",
		Source:         "user",
		IsModifiable:   true,
	}
}

func rdsOptionGroupToXML(in rdssvc.OptionGroup) rdsOptionGroupXML {
	out := rdsOptionGroupXML{
		OptionGroupName:        in.Name,
		OptionGroupDescription: in.Description,
		EngineName:             in.EngineName,
		MajorEngineVersion:     in.MajorEngineVersion,
	}
	for _, option := range in.Options {
		out.Options = append(out.Options, rdsOptionConfigurationXML{OptionName: option})
	}
	return out
}

func rdsDBSubnetGroupToXML(in rdssvc.DBSubnetGroup) rdsDBSubnetGroupItemXML {
	out := rdsDBSubnetGroupItemXML{
		DBSubnetGroupName:        in.Name,
		DBSubnetGroupDescription: in.Description,
		SubnetGroupStatus:        in.Status,
	}
	for _, subnetID := range in.SubnetIDs {
		out.Subnets = append(out.Subnets, rdsSubnetXML{SubnetIdentifier: subnetID})
	}
	return out
}

func rdsDBSecurityGroupToXML(in rdssvc.DBSecurityGroup) rdsDBSecurityGroupXML {
	out := rdsDBSecurityGroupXML{
		DBSecurityGroupName:        in.Name,
		DBSecurityGroupDescription: in.Description,
	}
	for _, cidr := range in.CIDRIPs {
		out.IPRanges = append(out.IPRanges, rdsIPRangeXML{CIDRIP: cidr, Status: "authorized"})
	}
	for _, group := range in.EC2SecurityGroups {
		out.EC2SecurityGroups = append(out.EC2SecurityGroups, rdsEC2SecurityGroupXML{
			EC2SecurityGroupName:    group.Name,
			EC2SecurityGroupOwnerId: group.OwnerID,
			Status:                  "authorized",
		})
	}
	return out
}

func rdsCertificateToXML(in rdssvc.Certificate) rdsCertificateXML {
	return rdsCertificateXML{
		CertificateIdentifier: in.Identifier,
		Thumbprint:            in.Thumbprint,
		ValidFrom:             formatRDSTime(in.ValidFrom),
		ValidTill:             formatRDSTime(in.ValidTill),
	}
}

func rdsDBClusterToXML(in rdssvc.DBCluster) rdsDBClusterXML {
	out := rdsDBClusterXML{
		DBClusterIdentifier:     in.Identifier,
		DBClusterArn:            in.ARN,
		Engine:                  in.Engine,
		Status:                  in.Status,
		MasterUsername:          in.MasterUsername,
		DatabaseName:            in.DatabaseName,
		DBSubnetGroup:           in.DBSubnetGroupName,
		DBClusterParameterGroup: in.DBClusterParameterGroup,
		Endpoint:                in.Endpoint,
		ReaderEndpoint:          in.ReaderEndpoint,
		BackupRetentionPeriod:   in.BackupRetentionPeriod,
		ClusterCreateTime:       formatRDSTime(in.CreatedAt),
	}
	for _, secGroup := range in.VpcSecurityGroupIDs {
		out.VpcSecurityGroups = append(out.VpcSecurityGroups, rdsVpcSecurityGroupMembershipXML{VpcSecurityGroupId: secGroup, Status: "active"})
	}
	return out
}

func rdsDBClusterEndpointToXML(in rdssvc.DBClusterEndpoint) rdsDBClusterEndpointXML {
	out := rdsDBClusterEndpointXML{
		DBClusterEndpointIdentifier: in.Identifier,
		DBClusterEndpointArn:        in.ARN,
		DBClusterIdentifier:         in.ClusterIdentifier,
		EndpointType:                in.EndpointType,
		Endpoint:                    in.Endpoint,
		Status:                      in.Status,
	}
	out.StaticMembers = append(out.StaticMembers, in.StaticMembers...)
	out.ExcludedMembers = append(out.ExcludedMembers, in.ExcludedMembers...)
	return out
}

func rdsGlobalClusterToXML(in rdssvc.GlobalCluster) rdsGlobalClusterXML {
	out := rdsGlobalClusterXML{
		GlobalClusterIdentifier: in.Identifier,
		GlobalClusterArn:        in.ARN,
		Status:                  in.Status,
		DeletionProtection:      in.DeletionProtection,
		EngineVersion:           in.EngineVersion,
	}
	for _, member := range in.Members {
		out.GlobalClusterMembers = append(out.GlobalClusterMembers, rdsGlobalClusterMemberXML{DBClusterArn: member, IsWriter: member == in.SourceDBClusterArn})
	}
	return out
}

func rdsBlueGreenDeploymentToXML(in rdssvc.BlueGreenDeployment) rdsBlueGreenDeploymentXML {
	return rdsBlueGreenDeploymentXML{
		BlueGreenDeploymentIdentifier: in.Identifier,
		BlueGreenDeploymentName:       in.Name,
		Source:                        in.Source,
		Target:                        in.Target,
		Status:                        in.Status,
		CreateTime:                    formatRDSTime(in.CreatedAt),
	}
}

func rdsTenantDatabaseToXML(in rdssvc.TenantDatabase) rdsTenantDatabaseXML {
	return rdsTenantDatabaseXML{
		DBClusterIdentifier: in.ClusterIdentifier,
		TenantDatabaseName:  in.TenantIdentifier,
		Status:              in.Status,
		MasterUsername:      in.MasterUsername,
		CreateTime:          formatRDSTime(in.CreatedAt),
	}
}

type rdsDBParameterGroupXML struct {
	DBParameterGroupName   string `xml:"DBParameterGroupName,omitempty"`
	DBParameterGroupFamily string `xml:"DBParameterGroupFamily,omitempty"`
	Description            string `xml:"Description,omitempty"`
}

type rdsParameterXML struct {
	ParameterName  string `xml:"ParameterName,omitempty"`
	ParameterValue string `xml:"ParameterValue,omitempty"`
	ApplyType      string `xml:"ApplyType,omitempty"`
	ApplyMethod    string `xml:"ApplyMethod,omitempty"`
	Source         string `xml:"Source,omitempty"`
	IsModifiable   bool   `xml:"IsModifiable"`
}

type rdsOptionConfigurationXML struct {
	OptionName string `xml:"OptionName,omitempty"`
}

type rdsOptionGroupXML struct {
	OptionGroupName        string                      `xml:"OptionGroupName,omitempty"`
	OptionGroupDescription string                      `xml:"OptionGroupDescription,omitempty"`
	EngineName             string                      `xml:"EngineName,omitempty"`
	MajorEngineVersion     string                      `xml:"MajorEngineVersion,omitempty"`
	Options                []rdsOptionConfigurationXML `xml:"Options>OptionConfiguration,omitempty"`
}

type rdsSubnetXML struct {
	SubnetIdentifier string `xml:"SubnetIdentifier,omitempty"`
}

type rdsDBSubnetGroupItemXML struct {
	DBSubnetGroupName        string         `xml:"DBSubnetGroupName,omitempty"`
	DBSubnetGroupDescription string         `xml:"DBSubnetGroupDescription,omitempty"`
	SubnetGroupStatus        string         `xml:"SubnetGroupStatus,omitempty"`
	Subnets                  []rdsSubnetXML `xml:"Subnets>Subnet,omitempty"`
}

type rdsIPRangeXML struct {
	Status string `xml:"Status,omitempty"`
	CIDRIP string `xml:"CIDRIP,omitempty"`
}

type rdsEC2SecurityGroupXML struct {
	Status                  string `xml:"Status,omitempty"`
	EC2SecurityGroupName    string `xml:"EC2SecurityGroupName,omitempty"`
	EC2SecurityGroupOwnerId string `xml:"EC2SecurityGroupOwnerId,omitempty"`
}

type rdsDBSecurityGroupXML struct {
	DBSecurityGroupName        string                   `xml:"DBSecurityGroupName,omitempty"`
	DBSecurityGroupDescription string                   `xml:"DBSecurityGroupDescription,omitempty"`
	EC2SecurityGroups          []rdsEC2SecurityGroupXML `xml:"EC2SecurityGroups>EC2SecurityGroup,omitempty"`
	IPRanges                   []rdsIPRangeXML          `xml:"IPRanges>IPRange,omitempty"`
}

type rdsCertificateXML struct {
	CertificateIdentifier string `xml:"CertificateIdentifier,omitempty"`
	Thumbprint            string `xml:"Thumbprint,omitempty"`
	ValidFrom             string `xml:"ValidFrom,omitempty"`
	ValidTill             string `xml:"ValidTill,omitempty"`
}

type rdsVpcSecurityGroupMembershipXML struct {
	VpcSecurityGroupId string `xml:"VpcSecurityGroupId,omitempty"`
	Status             string `xml:"Status,omitempty"`
}

type rdsDBClusterXML struct {
	DBClusterIdentifier     string                             `xml:"DBClusterIdentifier,omitempty"`
	DBClusterArn            string                             `xml:"DBClusterArn,omitempty"`
	Engine                  string                             `xml:"Engine,omitempty"`
	Status                  string                             `xml:"Status,omitempty"`
	MasterUsername          string                             `xml:"MasterUsername,omitempty"`
	DatabaseName            string                             `xml:"DatabaseName,omitempty"`
	DBSubnetGroup           string                             `xml:"DBSubnetGroup,omitempty"`
	DBClusterParameterGroup string                             `xml:"DBClusterParameterGroup,omitempty"`
	Endpoint                string                             `xml:"Endpoint,omitempty"`
	ReaderEndpoint          string                             `xml:"ReaderEndpoint,omitempty"`
	BackupRetentionPeriod   int                                `xml:"BackupRetentionPeriod,omitempty"`
	VpcSecurityGroups       []rdsVpcSecurityGroupMembershipXML `xml:"VpcSecurityGroups>VpcSecurityGroupMembership,omitempty"`
	ClusterCreateTime       string                             `xml:"ClusterCreateTime,omitempty"`
}

type rdsDBClusterEndpointXML struct {
	DBClusterEndpointIdentifier string   `xml:"DBClusterEndpointIdentifier,omitempty"`
	DBClusterEndpointArn        string   `xml:"DBClusterEndpointArn,omitempty"`
	DBClusterIdentifier         string   `xml:"DBClusterIdentifier,omitempty"`
	Endpoint                    string   `xml:"Endpoint,omitempty"`
	EndpointType                string   `xml:"EndpointType,omitempty"`
	Status                      string   `xml:"Status,omitempty"`
	StaticMembers               []string `xml:"StaticMembers>member,omitempty"`
	ExcludedMembers             []string `xml:"ExcludedMembers>member,omitempty"`
}

type rdsGlobalClusterMemberXML struct {
	DBClusterArn string `xml:"DBClusterArn,omitempty"`
	IsWriter     bool   `xml:"IsWriter"`
}

type rdsGlobalClusterXML struct {
	GlobalClusterIdentifier string                      `xml:"GlobalClusterIdentifier,omitempty"`
	GlobalClusterArn        string                      `xml:"GlobalClusterArn,omitempty"`
	Status                  string                      `xml:"Status,omitempty"`
	DeletionProtection      bool                        `xml:"DeletionProtection"`
	EngineVersion           string                      `xml:"EngineVersion,omitempty"`
	GlobalClusterMembers    []rdsGlobalClusterMemberXML `xml:"GlobalClusterMembers>GlobalClusterMember,omitempty"`
}

type rdsBlueGreenDeploymentXML struct {
	BlueGreenDeploymentIdentifier string `xml:"BlueGreenDeploymentIdentifier,omitempty"`
	BlueGreenDeploymentName       string `xml:"BlueGreenDeploymentName,omitempty"`
	Status                        string `xml:"Status,omitempty"`
	Source                        string `xml:"Source,omitempty"`
	Target                        string `xml:"Target,omitempty"`
	CreateTime                    string `xml:"CreateTime,omitempty"`
}

type rdsTenantDatabaseXML struct {
	DBClusterIdentifier string `xml:"DBClusterIdentifier,omitempty"`
	TenantDatabaseName  string `xml:"TenantDatabaseName,omitempty"`
	Status              string `xml:"Status,omitempty"`
	MasterUsername      string `xml:"MasterUsername,omitempty"`
	CreateTime          string `xml:"CreateTime,omitempty"`
}
