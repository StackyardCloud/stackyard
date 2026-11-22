package server

import (
	"encoding/xml"
	"net/http"
	"sort"

	rdssvc "github.com/stackyard/stackyard/internal/services/rds"
)

// Static DocumentDB catalogs: engine versions, orderable instance options,
// event categories and serverless platform versions.

func docdbTagsToXML(tags map[string]string) []rdsTagXML {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]rdsTagXML, 0, len(keys))
	for _, k := range keys {
		out = append(out, rdsTagXML{Key: k, Value: tags[k]})
	}
	return out
}

// docdbEngineFilter rejects any engine other than docdb, writing the error.
func docdbEngineFilter(w http.ResponseWriter, r *http.Request, required bool) bool {
	engine := docdbFormString(r, "Engine")
	if engine == "" && required {
		respondRDSErrorXML(w, http.StatusBadRequest, "MissingParameter", "The request must contain the parameter Engine")
		return false
	}
	if engine != "" && engine != rdssvc.DocDBEngine {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "Engine must be docdb")
		return false
	}
	return true
}

type docdbUpgradeTargetXML struct {
	Engine                string `xml:"Engine"`
	EngineVersion         string `xml:"EngineVersion"`
	Description           string `xml:"Description"`
	AutoUpgrade           bool   `xml:"AutoUpgrade"`
	IsMajorVersionUpgrade bool   `xml:"IsMajorVersionUpgrade"`
}

type docdbEngineVersionXML struct {
	Engine                             string                  `xml:"Engine"`
	EngineVersion                      string                  `xml:"EngineVersion"`
	DBParameterGroupFamily             string                  `xml:"DBParameterGroupFamily"`
	DBEngineDescription                string                  `xml:"DBEngineDescription"`
	DBEngineVersionDescription         string                  `xml:"DBEngineVersionDescription"`
	ValidUpgradeTarget                 []docdbUpgradeTargetXML `xml:"ValidUpgradeTarget>UpgradeTarget"`
	ExportableLogTypes                 []string                `xml:"ExportableLogTypes>member"`
	SupportsLogExportsToCloudwatchLogs bool                    `xml:"SupportsLogExportsToCloudwatchLogs"`
	SupportedCACertificateIdentifiers  []string                `xml:"SupportedCACertificateIdentifiers>member"`
	SupportsCertRotationWithoutRestart bool                    `xml:"SupportsCertificateRotationWithoutRestart"`
}

func (s *Server) handleDocDBDescribeDBEngineVersions(w http.ResponseWriter, r *http.Request) {
	if !docdbEngineFilter(w, r, false) {
		return
	}
	want, family := docdbFormString(r, "EngineVersion"), docdbFormString(r, "DBParameterGroupFamily")
	defaultOnly, ok := docdbBoolParam(w, r, "DefaultOnly")
	if !ok {
		return
	}
	var out []docdbEngineVersionXML
	versions := rdssvc.DocDBEngineVersions
	for i, v := range versions {
		if (want != "" && v.Version != want) || (family != "" && v.Family != family) ||
			(defaultOnly != nil && *defaultOnly && v.Version != "5.0.0") {
			continue
		}
		item := docdbEngineVersionXML{
			Engine: rdssvc.DocDBEngine, EngineVersion: v.Version, DBParameterGroupFamily: v.Family,
			DBEngineDescription: "Amazon DocumentDB (with MongoDB compatibility)", DBEngineVersionDescription: "docdb " + v.Version,
			ExportableLogTypes: []string{"audit", "profiler"}, SupportsLogExportsToCloudwatchLogs: true,
			SupportedCACertificateIdentifiers:  []string{"rds-ca-2019", "rds-ca-ecc384-g1", "rds-ca-rsa2048-g1", "rds-ca-rsa4096-g1"},
			SupportsCertRotationWithoutRestart: true,
		}
		for _, next := range versions[i+1:] {
			item.ValidUpgradeTarget = append(item.ValidUpgradeTarget, docdbUpgradeTargetXML{
				Engine: rdssvc.DocDBEngine, EngineVersion: next.Version, Description: "docdb " + next.Version, IsMajorVersionUpgrade: true,
			})
		}
		out = append(out, item)
	}
	respondRDSXML(w, "DescribeDBEngineVersions", struct {
		XMLName  xml.Name                `xml:"DescribeDBEngineVersionsResult"`
		Versions []docdbEngineVersionXML `xml:"DBEngineVersions>DBEngineVersion"`
	}{Versions: out})
}

type docdbOrderableXML struct {
	Engine            string `xml:"Engine"`
	EngineVersion     string `xml:"EngineVersion"`
	DBInstanceClass   string `xml:"DBInstanceClass"`
	LicenseModel      string `xml:"LicenseModel"`
	AvailabilityZones []struct {
		Name string `xml:"Name"`
	} `xml:"AvailabilityZones>AvailabilityZone"`
	Vpc         bool   `xml:"Vpc"`
	StorageType string `xml:"StorageType"`
}

func (s *Server) handleDocDBDescribeOrderableDBInstanceOptions(w http.ResponseWriter, r *http.Request) {
	if !docdbEngineFilter(w, r, true) {
		return
	}
	marker, max, ok := docdbPagingParams(w, r)
	if !ok {
		return
	}
	want, class := docdbFormString(r, "EngineVersion"), docdbFormString(r, "DBInstanceClass")
	var all []docdbOrderableXML
	for _, v := range rdssvc.DocDBEngineVersions {
		for _, c := range rdssvc.DocDBInstanceClasses {
			if (want != "" && v.Version != want) || (class != "" && c != class) {
				continue
			}
			for _, storage := range []string{"standard", "iopt1"} {
				item := docdbOrderableXML{Engine: rdssvc.DocDBEngine, EngineVersion: v.Version, DBInstanceClass: c, LicenseModel: "na", Vpc: true, StorageType: storage}
				for _, az := range []string{"us-east-1a", "us-east-1b", "us-east-1c"} {
					item.AvailabilityZones = append(item.AvailabilityZones, struct {
						Name string `xml:"Name"`
					}{Name: az})
				}
				all = append(all, item)
			}
		}
	}
	start, end, next, err := rdssvc.DocDBPage(len(all), marker, max)
	if err != nil {
		respondDocDBError(w, "DescribeOrderableDBInstanceOptions", err)
		return
	}
	respondRDSXML(w, "DescribeOrderableDBInstanceOptions", struct {
		XMLName xml.Name            `xml:"DescribeOrderableDBInstanceOptionsResult"`
		Options []docdbOrderableXML `xml:"OrderableDBInstanceOptions>OrderableDBInstanceOption"`
		Marker  string              `xml:"Marker,omitempty"`
	}{Options: all[start:end], Marker: next})
}

// docdbEventCategories lists the categories per source type.
var docdbEventCategories = []struct {
	SourceType string
	Categories []string
}{
	{"db-instance", []string{"availability", "backup", "configuration change", "creation", "deletion", "failover", "failure", "maintenance", "notification", "recovery"}},
	{"db-cluster", []string{"creation", "deletion", "failover", "failure", "maintenance", "notification"}},
	{"db-parameter-group", []string{"configuration change"}},
	{"db-security-group", []string{"configuration change", "failure"}},
	{"db-cluster-snapshot", []string{"backup"}},
}

func (s *Server) handleDocDBDescribeEventCategories(w http.ResponseWriter, r *http.Request) {
	source := docdbFormString(r, "SourceType")
	type entry struct {
		SourceType      string   `xml:"SourceType"`
		EventCategories []string `xml:"EventCategories>EventCategory"`
	}
	var out []entry
	for _, e := range docdbEventCategories {
		if source == "" || source == e.SourceType {
			out = append(out, entry{e.SourceType, e.Categories})
		}
	}
	if source != "" && len(out) == 0 {
		respondRDSErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "SourceType "+source+" is not valid")
		return
	}
	respondRDSXML(w, "DescribeEventCategories", struct {
		XMLName xml.Name `xml:"DescribeEventCategoriesResult"`
		Maps    []entry  `xml:"EventCategoriesMapList>EventCategoriesMap"`
	}{Maps: out})
}

func (s *Server) handleDocDBDescribeServerlessV2PlatformVersions(w http.ResponseWriter, r *http.Request) {
	if !docdbEngineFilter(w, r, false) {
		return
	}
	type capacity struct {
		Min float64 `xml:"MinCapacity"`
		Max float64 `xml:"MaxCapacity"`
	}
	type info struct {
		Version     string   `xml:"ServerlessV2PlatformVersion"`
		Description string   `xml:"ServerlessV2PlatformVersionDescription"`
		Engine      string   `xml:"Engine"`
		Support     capacity `xml:"ServerlessV2FeaturesSupport"`
		Status      string   `xml:"Status"`
		IsDefault   bool     `xml:"IsDefault"`
	}
	want := docdbFormString(r, "ServerlessV2PlatformVersion")
	var out []info
	for _, v := range []info{
		{Version: "1", Description: "DocumentDB serverless platform version 1", Engine: rdssvc.DocDBEngine, Support: capacity{0.5, 256}, Status: "enabled", IsDefault: true},
	} {
		if want == "" || want == v.Version {
			out = append(out, v)
		}
	}
	respondRDSXML(w, "DescribeServerlessV2PlatformVersions", struct {
		XMLName  xml.Name `xml:"DescribeServerlessV2PlatformVersionsResult"`
		Versions []info   `xml:"ServerlessV2PlatformVersions>ServerlessV2PlatformVersionInfo"`
	}{Versions: out})
}
