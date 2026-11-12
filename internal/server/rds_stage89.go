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
	switch action {
	case "EnableHttpEndpoint":
		respondRDSXML(w, action, rdsHTTPEndpointResultXML{
			XMLName:             xml.Name{Local: action + "Result"},
			ResourceArn:         strings.TrimSpace(r.Form.Get("ResourceArn")),
			HttpEndpointEnabled: true,
		})
	case "DisableHttpEndpoint":
		respondRDSXML(w, action, rdsHTTPEndpointResultXML{
			XMLName:             xml.Name{Local: action + "Result"},
			ResourceArn:         strings.TrimSpace(r.Form.Get("ResourceArn")),
			HttpEndpointEnabled: false,
		})
	case "ModifyCertificates":
		identifier := strings.TrimSpace(r.Form.Get("CertificateIdentifier"))
		if identifier == "" {
			identifier = "rds-ca-rsa4096-g1"
		}
		respondRDSXML(w, action, struct {
			XMLName     xml.Name          `xml:"ModifyCertificatesResult"`
			Certificate rdsCertificateXML `xml:"Certificate"`
		}{
			Certificate: rdsCertificateXML{CertificateIdentifier: identifier},
		})
	case "ModifyActivityStream":
		respondRDSXML(w, action, rdsModifyActivityStreamResultXML{
			XMLName:                         xml.Name{Local: action + "Result"},
			KmsKeyId:                        strings.TrimSpace(r.Form.Get("KmsKeyId")),
			KinesisStreamName:               "stackyard-rds-activity-stream",
			Status:                          "started",
			Mode:                            firstNonEmpty(strings.TrimSpace(r.Form.Get("Mode")), "async"),
			EngineNativeAuditFieldsIncluded: true,
			PolicyStatus:                    "locked",
		})
	case "AddSourceIdentifierToSubscription", "RemoveSourceIdentifierFromSubscription":
		sourceID := strings.TrimSpace(firstNonEmpty(r.Form.Get("SourceIdentifier"), r.Form.Get("SourceId")))
		respondRDSXML(w, action, struct {
			XMLName           xml.Name                `xml:""`
			EventSubscription rdsEventSubscriptionXML `xml:"EventSubscription"`
		}{
			XMLName: xml.Name{Local: action + "Result"},
			EventSubscription: rdsEventSubscriptionXML{
				CustSubscriptionId:   strings.TrimSpace(r.Form.Get("SubscriptionName")),
				SnsTopicArn:          strings.TrimSpace(r.Form.Get("SnsTopicArn")),
				SourceType:           strings.TrimSpace(r.Form.Get("SourceType")),
				SourceIdsList:        []string{sourceID},
				EventCategoriesList:  []string{"availability"},
				Enabled:              true,
				Status:               "active",
				EventSubscriptionArn: "arn:aws:rds:us-east-1:123456789012:es:stackyard",
			},
		})
	case "CopyOptionGroup":
		name := strings.TrimSpace(firstNonEmpty(
			r.Form.Get("TargetOptionGroupIdentifier"),
			r.Form.Get("OptionGroupName"),
			r.Form.Get("TargetOptionGroupName"),
		))
		if name == "" {
			name = "stackyard-option-group-copy"
		}
		respondRDSXML(w, action, struct {
			XMLName     xml.Name          `xml:"CopyOptionGroupResult"`
			OptionGroup rdsOptionGroupXML `xml:"OptionGroup"`
		}{
			OptionGroup: rdsOptionGroupXML{
				OptionGroupName:        name,
				OptionGroupDescription: strings.TrimSpace(firstNonEmpty(r.Form.Get("TargetOptionGroupDescription"), r.Form.Get("OptionGroupDescription"))),
				EngineName:             strings.TrimSpace(firstNonEmpty(r.Form.Get("EngineName"), r.Form.Get("Engine"))),
				MajorEngineVersion:     strings.TrimSpace(r.Form.Get("MajorEngineVersion")),
			},
		})
	case "RemoveFromGlobalCluster":
		globalClusterIdentifier := strings.TrimSpace(r.Form.Get("GlobalClusterIdentifier"))
		dbClusterIdentifier := strings.TrimSpace(firstNonEmpty(r.Form.Get("DbClusterIdentifier"), r.Form.Get("DBClusterIdentifier")))
		respondRDSXML(w, action, struct {
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
	case "DescribeEngineDefaultClusterParameters", "DescribeEngineDefaultParameters":
		family := strings.TrimSpace(firstNonEmpty(r.Form.Get("DBParameterGroupFamily"), r.Form.Get("DBClusterParameterGroupFamily")))
		if family == "" {
			family = "mysql8.0"
		}
		respondRDSXML(w, action, struct {
			XMLName        xml.Name             `xml:""`
			EngineDefaults rdsEngineDefaultsXML `xml:"EngineDefaults"`
		}{
			XMLName: xml.Name{Local: action + "Result"},
			EngineDefaults: rdsEngineDefaultsXML{
				DBParameterGroupFamily: family,
				Parameters: []rdsParameterXML{
					{
						ParameterName:  "autocommit",
						ParameterValue: "1",
						ApplyType:      "dynamic",
						ApplyMethod:    "immediate",
						Source:         "engine-default",
						IsModifiable:   true,
					},
				},
			},
		})
	case "DescribeEventCategories":
		sourceType := strings.TrimSpace(r.Form.Get("SourceType"))
		if sourceType == "" {
			sourceType = "db-instance"
		}
		respondRDSXML(w, action, struct {
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
	case "DescribeOptionGroupOptions":
		engine := strings.TrimSpace(firstNonEmpty(r.Form.Get("EngineName"), r.Form.Get("Engine")))
		if engine == "" {
			engine = "mysql"
		}
		majorVersion := strings.TrimSpace(r.Form.Get("MajorEngineVersion"))
		if majorVersion == "" {
			majorVersion = "8.0"
		}
		respondRDSXML(w, action, struct {
			XMLName            xml.Name                  `xml:"DescribeOptionGroupOptionsResult"`
			OptionGroupOptions []rdsOptionGroupOptionXML `xml:"OptionGroupOptions>OptionGroupOption"`
		}{
			OptionGroupOptions: []rdsOptionGroupOptionXML{
				{
					Name:                 "MEMCACHED",
					Description:          "Stackyard compatibility option",
					EngineName:           engine,
					MajorEngineVersion:   majorVersion,
					PortRequired:         false,
					DefaultPort:          0,
					Persistent:           false,
					Permanent:            false,
					VpcOnly:              false,
					CopyableCrossAccount: true,
				},
			},
		})
	default:
		respondRDSXML(w, action, rdsCompatResult{XMLName: xml.Name{Local: action + "Result"}})
	}
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

type rdsHTTPEndpointResultXML struct {
	XMLName             xml.Name `xml:""`
	ResourceArn         string   `xml:"ResourceArn,omitempty"`
	HttpEndpointEnabled bool     `xml:"HttpEndpointEnabled"`
}

type rdsModifyActivityStreamResultXML struct {
	XMLName                         xml.Name `xml:""`
	KmsKeyId                        string   `xml:"KmsKeyId,omitempty"`
	KinesisStreamName               string   `xml:"KinesisStreamName,omitempty"`
	Status                          string   `xml:"Status,omitempty"`
	Mode                            string   `xml:"Mode,omitempty"`
	EngineNativeAuditFieldsIncluded bool     `xml:"EngineNativeAuditFieldsIncluded"`
	PolicyStatus                    string   `xml:"PolicyStatus,omitempty"`
}

type rdsEngineDefaultsXML struct {
	DBParameterGroupFamily string            `xml:"DBParameterGroupFamily,omitempty"`
	Marker                 string            `xml:"Marker,omitempty"`
	Parameters             []rdsParameterXML `xml:"Parameters>Parameter,omitempty"`
}

type rdsEventCategoriesMapXML struct {
	SourceType      string   `xml:"SourceType,omitempty"`
	EventCategories []string `xml:"EventCategories>EventCategory,omitempty"`
}

type rdsOptionGroupOptionXML struct {
	Name                 string `xml:"Name,omitempty"`
	Description          string `xml:"Description,omitempty"`
	EngineName           string `xml:"EngineName,omitempty"`
	MajorEngineVersion   string `xml:"MajorEngineVersion,omitempty"`
	PortRequired         bool   `xml:"PortRequired"`
	DefaultPort          int    `xml:"DefaultPort,omitempty"`
	Persistent           bool   `xml:"Persistent"`
	Permanent            bool   `xml:"Permanent"`
	VpcOnly              bool   `xml:"VpcOnly"`
	CopyableCrossAccount bool   `xml:"CopyableCrossAccount"`
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
