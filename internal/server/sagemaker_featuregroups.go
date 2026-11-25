package server

import (
	"strings"
	"time"
)

// SageMaker feature groups: the control plane half of Feature Store. Records
// live on the group (records, bytes), so deleting a group drops its online
// store; sagemakerfeaturestoreruntime.go reads and writes them.

const (
	smfsMaxGroups      = 100
	smfsMaxDefinitions = 2500
)

// smfsReservedNames can't be feature names; Feature Store adds them to the
// offline store.
var smfsReservedNames = []string{"is_deleted", "write_time", "api_invocation_time"}

type smfsVectorConfig struct {
	Dimension int `json:"Dimension"`
}

type smfsCollectionConfig struct {
	VectorConfig *smfsVectorConfig `json:"VectorConfig,omitempty"`
}

type smfsFeatureDef struct {
	FeatureName      string                `json:"FeatureName"`
	FeatureType      string                `json:"FeatureType"`
	CollectionType   string                `json:"CollectionType,omitempty"`
	CollectionConfig *smfsCollectionConfig `json:"CollectionConfig,omitempty"`
}

type smfsTTL struct {
	Unit  string `json:"Unit,omitempty"`
	Value int64  `json:"Value,omitempty"`
}

var smfsTTLUnits = map[string]time.Duration{
	"Seconds": time.Second, "Minutes": time.Minute, "Hours": time.Hour,
	"Days": 24 * time.Hour, "Weeks": 7 * 24 * time.Hour,
}

// smfsMaxTTL caps a TTL so Value*Unit can't overflow; nothing outlives it.
const smfsMaxTTL = 100 * 365 * 24 * time.Hour

func (t *smfsTTL) duration() time.Duration {
	if t == nil || t.Value <= 0 {
		return 0
	}
	unit := smfsTTLUnits[t.Unit]
	if unit == 0 {
		return 0
	}
	if t.Value > int64(smfsMaxTTL/unit) {
		return smfsMaxTTL
	}
	return time.Duration(t.Value) * unit
}

type smfsKMS struct {
	KmsKeyId string `json:"KmsKeyId,omitempty"`
}

type smfsOnlineConfig struct {
	SecurityConfig    *smfsKMS `json:"SecurityConfig,omitempty"`
	EnableOnlineStore bool     `json:"EnableOnlineStore"`
	TtlDuration       *smfsTTL `json:"TtlDuration,omitempty"`
	StorageType       string   `json:"StorageType,omitempty"`
}

type smfsS3Config struct {
	S3Uri               string `json:"S3Uri"`
	KmsKeyId            string `json:"KmsKeyId,omitempty"`
	ResolvedOutputS3Uri string `json:"ResolvedOutputS3Uri,omitempty"`
}

type smfsCatalogConfig struct {
	TableName string `json:"TableName"`
	Catalog   string `json:"Catalog"`
	Database  string `json:"Database"`
}

type smfsOfflineConfig struct {
	S3StorageConfig          smfsS3Config       `json:"S3StorageConfig"`
	DisableGlueTableCreation bool               `json:"DisableGlueTableCreation,omitempty"`
	DataCatalogConfig        *smfsCatalogConfig `json:"DataCatalogConfig,omitempty"`
	TableFormat              string             `json:"TableFormat,omitempty"`
}

type smfsThroughput struct {
	ThroughputMode                string `json:"ThroughputMode,omitempty"`
	ProvisionedReadCapacityUnits  *int   `json:"ProvisionedReadCapacityUnits,omitempty"`
	ProvisionedWriteCapacityUnits *int   `json:"ProvisionedWriteCapacityUnits,omitempty"`
}

type smfsGroup struct {
	Name, ARN            string
	RecordID, EventTime  string
	Defs                 []smfsFeatureDef
	defIndex             map[string]int
	Online               *smfsOnlineConfig
	Offline              *smfsOfflineConfig
	Throughput           smfsThroughput
	RoleArn, Description string
	Created, Modified    time.Time
	updated              bool
	records              map[string]*smfsRecord
	bytes                int
}

func (g *smfsGroup) online() bool { return g.Online != nil && g.Online.EnableOnlineStore }

func (g *smfsGroup) def(name string) *smfsFeatureDef {
	if i, ok := g.defIndex[name]; ok {
		return &g.Defs[i]
	}
	return nil
}

// smfsCheckDefs validates feature definitions to be added to existing ones.
func smfsCheckDefs(existing, adds []smfsFeatureDef) *sagemakerResourceError {
	if len(existing)+len(adds) > smfsMaxDefinitions {
		return sagemakerResErr(sagemakerErrLimit, "A feature group can have at most %d feature definitions.", smfsMaxDefinitions)
	}
	seen := map[string]bool{}
	for _, d := range existing {
		seen[strings.ToLower(d.FeatureName)] = true
	}
	for _, d := range adds {
		lower := strings.ToLower(d.FeatureName)
		if seen[lower] {
			return sagemakerResErr(sagemakerErrValidation, "Feature name %s is defined more than once (feature names are case-insensitive).", d.FeatureName)
		}
		seen[lower] = true
		for _, reserved := range smfsReservedNames {
			if lower == reserved {
				return sagemakerResErr(sagemakerErrValidation, "Feature name %s is reserved.", d.FeatureName)
			}
		}
		vector := d.CollectionConfig != nil && d.CollectionConfig.VectorConfig != nil
		switch {
		case d.CollectionType == "Vector" && !vector:
			return sagemakerResErr(sagemakerErrValidation, "Feature %s is a Vector and needs CollectionConfig.VectorConfig.Dimension.", d.FeatureName)
		case d.CollectionType != "Vector" && d.CollectionConfig != nil:
			return sagemakerResErr(sagemakerErrValidation, "CollectionConfig is only valid for Vector features; %s is not one.", d.FeatureName)
		case d.CollectionType == "Vector" && d.FeatureType == "String":
			return sagemakerResErr(sagemakerErrValidation, "Vector feature %s must be Integral or Fractional.", d.FeatureName)
		}
	}
	return nil
}

func smfsIndex(defs []smfsFeatureDef) map[string]int {
	out := make(map[string]int, len(defs))
	for i, d := range defs {
		out[d.FeatureName] = i
	}
	return out
}

type smfsCreateGroupReq struct {
	FeatureGroupName            string
	RecordIdentifierFeatureName string
	EventTimeFeatureName        string
	FeatureDefinitions          []smfsFeatureDef
	OnlineStoreConfig           *smfsOnlineConfig
	OfflineStoreConfig          *smfsOfflineConfig
	ThroughputConfig            *smfsThroughput
	RoleArn                     string
	Description                 string
	Tags                        []sagemakerTag
}

func smfsCreateFeatureGroup(st *sagemakerStore, req smfsCreateGroupReq, now time.Time) (any, *sagemakerResourceError) {
	name := req.FeatureGroupName
	if st.featureGroups[name] != nil {
		return nil, sagemakerResErr(sagemakerErrInUse, "Resource Already Exists: FeatureGroup with name %s already exists. Choose a different name.", name)
	}
	if len(st.featureGroups) >= smfsMaxGroups {
		return nil, sagemakerResErr(sagemakerErrLimit, "Stackyard keeps at most %d feature groups.", smfsMaxGroups)
	}
	if err := smfsCheckDefs(nil, req.FeatureDefinitions); err != nil {
		return nil, err
	}
	g := &smfsGroup{
		Name: name, ARN: sagemakerResourceARN("feature-group", name),
		RecordID: req.RecordIdentifierFeatureName, EventTime: req.EventTimeFeatureName,
		Defs: req.FeatureDefinitions, defIndex: smfsIndex(req.FeatureDefinitions),
		Online: req.OnlineStoreConfig, Offline: req.OfflineStoreConfig,
		Throughput: smfsThroughput{ThroughputMode: "OnDemand"},
		RoleArn:    req.RoleArn, Description: req.Description,
		Created: now, Modified: now, records: map[string]*smfsRecord{},
	}
	if req.ThroughputConfig != nil {
		g.Throughput = *req.ThroughputConfig
	}
	id, et := g.def(g.RecordID), g.def(g.EventTime)
	switch {
	case id == nil:
		return nil, sagemakerResErr(sagemakerErrValidation, "RecordIdentifierFeatureName %s is not in FeatureDefinitions.", g.RecordID)
	case et == nil:
		return nil, sagemakerResErr(sagemakerErrValidation, "EventTimeFeatureName %s is not in FeatureDefinitions.", g.EventTime)
	case id.FeatureType == "Fractional" || id.CollectionType != "":
		return nil, sagemakerResErr(sagemakerErrValidation, "The record identifier feature must be a scalar Integral or String feature.")
	case et.FeatureType == "Integral" || et.CollectionType != "":
		return nil, sagemakerResErr(sagemakerErrValidation, "The event time feature must be a scalar Fractional or String feature.")
	case !g.online() && g.Offline == nil:
		return nil, sagemakerResErr(sagemakerErrValidation, "Enable an online store or configure an offline store.")
	case g.Offline != nil && g.RoleArn == "":
		return nil, sagemakerResErr(sagemakerErrValidation, "RoleArn is required when OfflineStoreConfig is specified.")
	case g.Online != nil && !g.Online.EnableOnlineStore && g.Online.TtlDuration != nil:
		return nil, sagemakerResErr(sagemakerErrValidation, "TtlDuration needs EnableOnlineStore.")
	}
	if g.Offline != nil {
		off := *g.Offline
		stamp := now.Format("20060102T150405Z")
		off.S3StorageConfig.ResolvedOutputS3Uri = strings.TrimRight(off.S3StorageConfig.S3Uri, "/") +
			"/" + sagemakerAccountID + "/sagemaker/us-east-1/offline-store/" + name + "-" + stamp + "/data"
		if off.DataCatalogConfig == nil && !off.DisableGlueTableCreation {
			off.DataCatalogConfig = &smfsCatalogConfig{TableName: strings.ToLower(name) + "_" + stamp, Catalog: "AwsDataCatalog", Database: "sagemaker_featurestore"}
		}
		g.Offline = &off
	}
	st.featureGroups[name] = g
	st.setTagsLocked(g.ARN, req.Tags)
	return map[string]any{"FeatureGroupArn": g.ARN}, nil
}

// smfsGroupLocked resolves a feature group name or ARN.
func (s *sagemakerStore) smfsGroupLocked(ref string) (*smfsGroup, *sagemakerResourceError) {
	name := sagemakerNameFromARN(ref, "feature-group")
	if g := s.featureGroups[name]; g != nil {
		return g, nil
	}
	return nil, sagemakerResErr(sagemakerErrNotFound, "Resource Not Found: Amazon SageMaker can't find a FeatureGroup with name %s", name)
}

type smfsNameReq struct {
	FeatureGroupName string
}

func smfsDescribeFeatureGroup(st *sagemakerStore, req smfsNameReq, _ time.Time) (any, *sagemakerResourceError) {
	g, err := st.smfsGroupLocked(req.FeatureGroupName)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"FeatureGroupArn": g.ARN, "FeatureGroupName": g.Name,
		"RecordIdentifierFeatureName": g.RecordID, "EventTimeFeatureName": g.EventTime,
		"FeatureDefinitions": g.Defs, "CreationTime": sagemakerEpoch(g.Created),
		"LastModifiedTime": sagemakerEpoch(g.Modified), "FeatureGroupStatus": "Created",
		"ThroughputConfig": g.Throughput, "NextToken": "",
	}
	if g.Online != nil {
		out["OnlineStoreConfig"] = g.Online
		out["OnlineStoreTotalSizeBytes"] = g.bytes
	}
	if g.Offline != nil {
		out["OfflineStoreConfig"] = g.Offline
		out["OfflineStoreStatus"] = map[string]any{"Status": "Active"}
	}
	if g.RoleArn != "" {
		out["RoleArn"] = g.RoleArn
	}
	if g.Description != "" {
		out["Description"] = g.Description
	}
	if g.updated {
		out["LastUpdateStatus"] = map[string]any{"Status": "Successful"}
	}
	return out, nil
}

type smfsUpdateGroupReq struct {
	FeatureGroupName  string
	FeatureAdditions  []smfsFeatureDef
	OnlineStoreConfig *struct {
		TtlDuration *smfsTTL
		StorageType string
	}
	ThroughputConfig *smfsThroughput
}

func smfsUpdateFeatureGroup(st *sagemakerStore, req smfsUpdateGroupReq, now time.Time) (any, *sagemakerResourceError) {
	g, err := st.smfsGroupLocked(req.FeatureGroupName)
	if err != nil {
		return nil, err
	}
	if err := smfsCheckDefs(g.Defs, req.FeatureAdditions); err != nil {
		return nil, err
	}
	if up := req.OnlineStoreConfig; up != nil {
		if !g.online() {
			return nil, sagemakerResErr(sagemakerErrValidation, "Feature group %s has no online store to update.", g.Name)
		}
		online := *g.Online
		if up.TtlDuration != nil {
			online.TtlDuration = up.TtlDuration
		}
		if up.StorageType != "" {
			online.StorageType = up.StorageType
		}
		g.Online = &online
	}
	if tp := req.ThroughputConfig; tp != nil {
		if tp.ThroughputMode != "" {
			g.Throughput.ThroughputMode = tp.ThroughputMode
		}
		if tp.ProvisionedReadCapacityUnits != nil {
			g.Throughput.ProvisionedReadCapacityUnits = tp.ProvisionedReadCapacityUnits
		}
		if tp.ProvisionedWriteCapacityUnits != nil {
			g.Throughput.ProvisionedWriteCapacityUnits = tp.ProvisionedWriteCapacityUnits
		}
	}
	if len(req.FeatureAdditions) > 0 {
		// Copy on write: handed-out responses may still reference the old slice.
		defs := make([]smfsFeatureDef, 0, len(g.Defs)+len(req.FeatureAdditions))
		g.Defs = append(append(defs, g.Defs...), req.FeatureAdditions...)
		g.defIndex = smfsIndex(g.Defs)
	}
	g.Modified, g.updated = now, true
	return map[string]any{"FeatureGroupArn": g.ARN}, nil
}

func smfsDeleteFeatureGroup(st *sagemakerStore, req smfsNameReq, _ time.Time) (any, *sagemakerResourceError) {
	g, err := st.smfsGroupLocked(req.FeatureGroupName)
	if err != nil {
		return nil, err
	}
	st.smfsBytes -= g.bytes
	delete(st.featureGroups, g.Name)
	delete(st.tags, g.ARN)
	return nil, nil
}

type smfsListGroupsReq struct {
	NameContains             string
	FeatureGroupStatusEquals string
	OfflineStoreStatusEquals string
	CreationTimeAfter        *sagemakerTimestamp
	CreationTimeBefore       *sagemakerTimestamp
	SortOrder                string
	SortBy                   string
	MaxResults               int
	NextToken                string
}

func smfsListFeatureGroups(st *sagemakerStore, req smfsListGroupsReq, _ time.Time) (any, *sagemakerResourceError) {
	byTime := req.SortBy == "" || req.SortBy == "CreationTime"
	names := sagemakerSortNames(st.featureGroups, byTime, req.SortOrder == "Descending", func(g *smfsGroup) time.Time { return g.Created })
	var keep []*smfsGroup
	for _, name := range names {
		g := st.featureGroups[name]
		offline := "Disabled"
		if g.Offline != nil {
			offline = "Active"
		}
		switch {
		case req.NameContains != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(req.NameContains)),
			req.FeatureGroupStatusEquals != "" && req.FeatureGroupStatusEquals != "Created",
			req.OfflineStoreStatusEquals != "" && req.OfflineStoreStatusEquals != offline,
			req.CreationTimeAfter != nil && !g.Created.After(req.CreationTimeAfter.Time),
			req.CreationTimeBefore != nil && !g.Created.Before(req.CreationTimeBefore.Time):
			continue
		}
		keep = append(keep, g)
	}
	start, end, next, err := sagemakerPage(len(keep), req.NextToken, req.MaxResults)
	if err != nil {
		return nil, err
	}
	summaries := []any{}
	for _, g := range keep[start:end] {
		item := map[string]any{"FeatureGroupName": g.Name, "FeatureGroupArn": g.ARN,
			"CreationTime": sagemakerEpoch(g.Created), "FeatureGroupStatus": "Created"}
		if g.Offline != nil {
			item["OfflineStoreStatus"] = map[string]any{"Status": "Active"}
		}
		summaries = append(summaries, item)
	}
	out := map[string]any{"FeatureGroupSummaries": summaries}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}
