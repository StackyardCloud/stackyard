package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	rdssvc "github.com/stackyard/stackyard/internal/services/rds"
)

var rdsStage9CompatActions = []string{
	"AddSourceIdentifierToSubscription",
	"BacktrackDBCluster",
	"CopyDBClusterParameterGroup",
	"CopyDBClusterSnapshot",
	"CopyDBParameterGroup",
	"CopyOptionGroup",
	"CreateCustomDBEngineVersion",
	"CreateDBClusterParameterGroup",
	"CreateDBClusterSnapshot",
	"CreateDBShardGroup",
	"DeleteCustomDBEngineVersion",
	"DeleteDBClusterAutomatedBackup",
	"DeleteDBClusterParameterGroup",
	"DeleteDBClusterSnapshot",
	"DeleteDBShardGroup",
	"DescribeDBClusterAutomatedBackups",
	"DescribeDBClusterBacktracks",
	"DescribeDBClusterParameterGroups",
	"DescribeDBClusterParameters",
	"DescribeDBClusterSnapshotAttributes",
	"DescribeDBClusterSnapshots",
	"DescribeDBLogFiles",
	"DescribeDBMajorEngineVersions",
	"DescribeDBProxyTargetGroups",
	"DescribeDBRecommendations",
	"DescribeDBShardGroups",
	"DescribeDBSnapshotAttributes",
	"DescribeDBSnapshotTenantDatabases",
	"DescribeEngineDefaultClusterParameters",
	"DescribeEngineDefaultParameters",
	"DescribeEventCategories",
	"DescribeOptionGroupOptions",
	"DisableHttpEndpoint",
	"DownloadDBLogFilePortion",
	"EnableHttpEndpoint",
	"ModifyActivityStream",
	"ModifyCertificates",
	"ModifyCurrentDBClusterCapacity",
	"ModifyCustomDBEngineVersion",
	"ModifyDBClusterParameterGroup",
	"ModifyDBClusterSnapshotAttribute",
	"ModifyDBProxyTargetGroup",
	"ModifyDBRecommendation",
	"ModifyDBShardGroup",
	"ModifyDBSnapshot",
	"ModifyDBSnapshotAttribute",
	"PromoteReadReplicaDBCluster",
	"RebootDBShardGroup",
	"RemoveFromGlobalCluster",
	"RemoveSourceIdentifierFromSubscription",
	"ResetDBClusterParameterGroup",
	"RestoreDBClusterFromS3",
	"RestoreDBClusterFromSnapshot",
	"RestoreDBClusterToPointInTime",
	"RestoreDBInstanceFromS3",
}

func (s *Server) handleRDSDescribeReservedDBInstancesOfferings(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeReservedDBInstancesOfferings(rdssvc.DescribeReservedDBInstancesOfferingsInput{
		OfferingID:         strings.TrimSpace(firstNonEmpty(r.Form.Get("ReservedDBInstancesOfferingId"), r.Form.Get("ReservedDBInstancesOfferingID"))),
		DBInstanceClass:    strings.TrimSpace(r.Form.Get("DBInstanceClass")),
		ProductDescription: strings.TrimSpace(r.Form.Get("ProductDescription")),
		MaxRecords:         maxRecords,
		Marker:             strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeReservedDBInstancesOfferings", err)
		return
	}
	out := make([]rdsReservedDBInstancesOfferingXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsReservedDBInstancesOfferingToXML(item))
	}
	respondRDSXML(w, "DescribeReservedDBInstancesOfferings", struct {
		XMLName                      xml.Name                            `xml:"DescribeReservedDBInstancesOfferingsResult"`
		Marker                       string                              `xml:"Marker,omitempty"`
		ReservedDBInstancesOfferings []rdsReservedDBInstancesOfferingXML `xml:"ReservedDBInstancesOfferings>ReservedDBInstancesOffering"`
	}{
		Marker:                       marker,
		ReservedDBInstancesOfferings: out,
	})
}

func (s *Server) handleRDSPurchaseReservedDBInstancesOffering(w http.ResponseWriter, r *http.Request) {
	instanceCount, err := parseOptionalRDSInt(r.Form.Get("DBInstanceCount"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "DBInstanceCount is invalid")
		return
	}
	item, err := s.rds.PurchaseReservedDBInstancesOffering(rdssvc.PurchaseReservedDBInstancesOfferingInput{
		OfferingID:           strings.TrimSpace(firstNonEmpty(r.Form.Get("ReservedDBInstancesOfferingId"), r.Form.Get("ReservedDBInstancesOfferingID"))),
		ReservedDBInstanceID: strings.TrimSpace(firstNonEmpty(r.Form.Get("ReservedDBInstanceId"), r.Form.Get("ReservedDBInstanceID"))),
		DBInstanceCount:      instanceCount,
	})
	if err != nil {
		respondRDSServiceError(w, "PurchaseReservedDBInstancesOffering", err)
		return
	}
	respondRDSXML(w, "PurchaseReservedDBInstancesOffering", struct {
		XMLName            xml.Name                 `xml:"PurchaseReservedDBInstancesOfferingResult"`
		ReservedDBInstance rdsReservedDBInstanceXML `xml:"ReservedDBInstance"`
	}{
		ReservedDBInstance: rdsReservedDBInstanceToXML(item),
	})
}

func (s *Server) handleRDSDescribeReservedDBInstances(w http.ResponseWriter, r *http.Request) {
	maxRecords, err := parseOptionalRDSInt(r.Form.Get("MaxRecords"))
	if err != nil {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "MaxRecords is invalid")
		return
	}
	items, marker, err := s.rds.DescribeReservedDBInstances(rdssvc.DescribeReservedDBInstancesInput{
		ReservedDBInstanceID: strings.TrimSpace(firstNonEmpty(r.Form.Get("ReservedDBInstanceId"), r.Form.Get("ReservedDBInstanceID"))),
		DBInstanceClass:      strings.TrimSpace(r.Form.Get("DBInstanceClass")),
		MaxRecords:           maxRecords,
		Marker:               strings.TrimSpace(r.Form.Get("Marker")),
	})
	if err != nil {
		respondRDSServiceError(w, "DescribeReservedDBInstances", err)
		return
	}
	out := make([]rdsReservedDBInstanceXML, 0, len(items))
	for _, item := range items {
		out = append(out, rdsReservedDBInstanceToXML(item))
	}
	respondRDSXML(w, "DescribeReservedDBInstances", struct {
		XMLName             xml.Name                   `xml:"DescribeReservedDBInstancesResult"`
		Marker              string                     `xml:"Marker,omitempty"`
		ReservedDBInstances []rdsReservedDBInstanceXML `xml:"ReservedDBInstances>ReservedDBInstance"`
	}{
		Marker:              marker,
		ReservedDBInstances: out,
	})
}

func (s *Server) handleRDSCompatNoop(w http.ResponseWriter, r *http.Request) {
	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondRDSErrorXML(w, http.StatusBadRequest, "MissingParameter", "Action is required")
		return
	}
	respondRDSXML(w, action, rdsCompatResult{XMLName: xml.Name{Local: action + "Result"}})
}

func rdsReservedDBInstancesOfferingToXML(in rdssvc.ReservedDBInstancesOffering) rdsReservedDBInstancesOfferingXML {
	return rdsReservedDBInstancesOfferingXML{
		ReservedDBInstancesOfferingId: in.OfferingID,
		DBInstanceClass:               in.DBInstanceClass,
		Duration:                      in.Duration,
		FixedPrice:                    in.FixedPrice,
		UsagePrice:                    in.UsagePrice,
		ProductDescription:            in.ProductDescription,
		OfferingType:                  in.OfferingType,
		MultiAZ:                       in.MultiAZ,
		CurrencyCode:                  in.CurrencyCode,
	}
}

func rdsReservedDBInstanceToXML(in rdssvc.ReservedDBInstance) rdsReservedDBInstanceXML {
	return rdsReservedDBInstanceXML{
		ReservedDBInstanceId:          in.ReservedDBInstanceID,
		ReservedDBInstancesOfferingId: in.ReservedDBInstancesOfferingID,
		DBInstanceClass:               in.DBInstanceClass,
		Duration:                      in.Duration,
		FixedPrice:                    in.FixedPrice,
		UsagePrice:                    in.UsagePrice,
		ProductDescription:            in.ProductDescription,
		OfferingType:                  in.OfferingType,
		MultiAZ:                       in.MultiAZ,
		StartTime:                     formatRDSTime(in.StartTime),
		State:                         in.State,
		DBInstanceCount:               in.DBInstanceCount,
		CurrencyCode:                  in.CurrencyCode,
	}
}

type rdsCompatResult struct {
	XMLName xml.Name `xml:""`
}

type rdsReservedDBInstancesOfferingXML struct {
	ReservedDBInstancesOfferingId string  `xml:"ReservedDBInstancesOfferingId,omitempty"`
	DBInstanceClass               string  `xml:"DBInstanceClass,omitempty"`
	Duration                      int     `xml:"Duration,omitempty"`
	FixedPrice                    float64 `xml:"FixedPrice,omitempty"`
	UsagePrice                    float64 `xml:"UsagePrice,omitempty"`
	ProductDescription            string  `xml:"ProductDescription,omitempty"`
	OfferingType                  string  `xml:"OfferingType,omitempty"`
	MultiAZ                       bool    `xml:"MultiAZ"`
	CurrencyCode                  string  `xml:"CurrencyCode,omitempty"`
}

type rdsReservedDBInstanceXML struct {
	ReservedDBInstanceId          string  `xml:"ReservedDBInstanceId,omitempty"`
	ReservedDBInstancesOfferingId string  `xml:"ReservedDBInstancesOfferingId,omitempty"`
	DBInstanceClass               string  `xml:"DBInstanceClass,omitempty"`
	Duration                      int     `xml:"Duration,omitempty"`
	FixedPrice                    float64 `xml:"FixedPrice,omitempty"`
	UsagePrice                    float64 `xml:"UsagePrice,omitempty"`
	ProductDescription            string  `xml:"ProductDescription,omitempty"`
	OfferingType                  string  `xml:"OfferingType,omitempty"`
	MultiAZ                       bool    `xml:"MultiAZ"`
	StartTime                     string  `xml:"StartTime,omitempty"`
	State                         string  `xml:"State,omitempty"`
	DBInstanceCount               int     `xml:"DBInstanceCount,omitempty"`
	CurrencyCode                  string  `xml:"CurrencyCode,omitempty"`
}
