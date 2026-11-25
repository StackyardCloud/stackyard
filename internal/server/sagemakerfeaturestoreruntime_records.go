package server

import (
	"math"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Feature Store online-store records. A record keeps its feature values in
// feature-definition order; stored value slices are replaced, never modified.

const (
	smfsMaxRecordBytes     = 512 << 10
	smfsMaxRecordsPerGroup = 10000
	smfsMaxGroupBytes      = 16 << 20
	smfsMaxStoredBytes     = 64 << 20
	// smfsMaxBatchGetBytes caps the record bytes of one BatchGetRecord
	// response; the rest is returned as UnprocessedIdentifiers.
	smfsMaxBatchGetBytes = 8 << 20
	smfsMaxUpdateFeature = 100
	smfsDefaultListPage  = 100
)

type smfsValue struct {
	FeatureName       string    `json:"FeatureName"`
	ValueAsString     *string   `json:"ValueAsString,omitempty"`
	ValueAsStringList *[]string `json:"ValueAsStringList,omitempty"`
}

func (v smfsValue) null() bool { return v.ValueAsString == nil && v.ValueAsStringList == nil }

type smfsRecord struct {
	id        string
	values    []smfsValue
	eventTime time.Time
	ttl       time.Duration
	deleted   bool
	bytes     int
}

func (r *smfsRecord) expiresAt() time.Time {
	if r.ttl <= 0 {
		return time.Time{}
	}
	return r.eventTime.Add(r.ttl)
}

func (r *smfsRecord) live(now time.Time) bool {
	exp := r.expiresAt()
	return !r.deleted && (exp.IsZero() || now.Before(exp))
}

func smfsNotFound(format string, args ...any) *sagemakerDataError {
	e := smfsInvalid(format, args...)
	e.status, e.code = http.StatusNotFound, "ResourceNotFound"
	return e
}

func (s *sagemakerStore) smfsGroupDataLocked(ref string) (*smfsGroup, *sagemakerDataError) {
	g, err := s.smfsGroupLocked(ref)
	if err != nil {
		return nil, smfsNotFound("%s", err.msg)
	}
	return g, nil
}

func (s *sagemakerStore) smfsOnlineGroupLocked(ref string) (*smfsGroup, *sagemakerDataError) {
	g, err := s.smfsGroupDataLocked(ref)
	if err == nil && !g.online() {
		err = smfsInvalid("Feature group %s does not have an online store.", g.Name)
	}
	return g, err
}

// checkValue checks one feature value against its definition and returns its size.
func (g *smfsGroup) checkValue(v smfsValue) (int, *sagemakerDataError) {
	d := g.def(v.FeatureName)
	if d == nil {
		return 0, smfsInvalid("Feature %s is not defined in feature group %s.", v.FeatureName, g.Name)
	}
	if v.ValueAsString != nil && v.ValueAsStringList != nil {
		return 0, smfsInvalid("Feature %s has both ValueAsString and ValueAsStringList.", v.FeatureName)
	}
	size := len(v.FeatureName)
	check := func(s string) *sagemakerDataError {
		size += len(s)
		switch d.FeatureType {
		case "Integral":
			if _, err := strconv.ParseInt(s, 10, 64); err != nil {
				return smfsInvalid("Feature %s is Integral; %q is not a 64-bit integer.", d.FeatureName, s)
			}
		case "Fractional":
			if f, err := strconv.ParseFloat(s, 64); err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
				return smfsInvalid("Feature %s is Fractional; %q is not a finite number.", d.FeatureName, s)
			}
		}
		return nil
	}
	switch {
	case v.null():
	case d.CollectionType == "" && v.ValueAsStringList != nil:
		return 0, smfsInvalid("Feature %s is a scalar; use ValueAsString.", d.FeatureName)
	case d.CollectionType != "" && v.ValueAsString != nil:
		return 0, smfsInvalid("Feature %s is a %s; use ValueAsStringList.", d.FeatureName, d.CollectionType)
	case v.ValueAsString != nil:
		if err := check(*v.ValueAsString); err != nil {
			return 0, err
		}
	default:
		list := *v.ValueAsStringList
		if d.CollectionType == "Vector" && d.CollectionConfig != nil && d.CollectionConfig.VectorConfig != nil &&
			len(list) != d.CollectionConfig.VectorConfig.Dimension {
			return 0, smfsInvalid("Feature %s is a Vector of dimension %d; got %d values.", d.FeatureName, d.CollectionConfig.VectorConfig.Dimension, len(list))
		}
		seen := map[string]bool{}
		for _, item := range list {
			if err := check(item); err != nil {
				return 0, err
			}
			if d.CollectionType == "Set" && seen[item] {
				return 0, smfsInvalid("Feature %s is a Set; %q appears more than once.", d.FeatureName, item)
			}
			seen[item] = true
		}
	}
	if size > smfsMaxRecordBytes {
		return 0, smfsInvalid("A record can hold at most %d bytes.", smfsMaxRecordBytes)
	}
	return size, nil
}

// merge applies updates to base (both may be nil) and returns the record's
// values in definition order and their size. A null update clears a feature.
func (g *smfsGroup) merge(base, updates []smfsValue) ([]smfsValue, int, *sagemakerDataError) {
	byName := make(map[string]smfsValue, len(base)+len(updates))
	for _, v := range base {
		byName[v.FeatureName] = v
	}
	seen := map[string]bool{}
	for _, v := range updates {
		if seen[v.FeatureName] {
			return nil, 0, smfsInvalid("Feature %s appears more than once.", v.FeatureName)
		}
		seen[v.FeatureName] = true
		if _, err := g.checkValue(v); err != nil {
			return nil, 0, err
		}
		byName[v.FeatureName] = v
	}
	out := make([]smfsValue, 0, len(byName))
	size := 0
	for _, v := range byName {
		if v.null() {
			continue
		}
		n, _ := g.checkValue(v)
		size += n
		out = append(out, v)
	}
	if size > smfsMaxRecordBytes {
		return nil, 0, smfsInvalid("A record can hold at most %d bytes.", smfsMaxRecordBytes)
	}
	sort.Slice(out, func(i, j int) bool { return g.defIndex[out[i].FeatureName] < g.defIndex[out[j].FeatureName] })
	return out, size, nil
}

func smfsFind(values []smfsValue, name string) *string {
	for _, v := range values {
		if v.FeatureName == name {
			return v.ValueAsString
		}
	}
	return nil
}

// parseEventTime reads an event time: epoch seconds for a Fractional feature,
// ISO 8601 in UTC ("...Z") for a String one.
func (g *smfsGroup) parseEventTime(v string) (time.Time, *sagemakerDataError) {
	if g.def(g.EventTime).FeatureType == "Fractional" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f < 0 || f > 1e11 {
			return time.Time{}, smfsInvalid("Event time %q is not epoch seconds.", v)
		}
		sec := int64(f)
		return time.Unix(sec, int64((f-float64(sec))*1e9)).UTC(), nil
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil || !strings.HasSuffix(v, "Z") {
		return time.Time{}, smfsInvalid("Event time %q must be ISO 8601 in UTC, like 2024-01-02T03:04:05Z.", v)
	}
	return t, nil
}

// targetsOnline checks TargetStores against the group's stores and reports
// whether the online store is written.
func (g *smfsGroup) targetsOnline(targets []string) (bool, *sagemakerDataError) {
	if err := smfsCheckTargetList(targets); err != nil {
		return false, err
	}
	if len(targets) == 0 {
		return g.online(), nil
	}
	for _, t := range targets {
		if t == "OnlineStore" && !g.online() {
			return false, smfsInvalid("Feature group %s does not have an online store.", g.Name)
		}
		if t == "OfflineStore" && g.Offline == nil {
			return false, smfsInvalid("Feature group %s does not have an offline store.", g.Name)
		}
	}
	return slices.Contains(targets, "OnlineStore"), nil
}

// storeLocked replaces id's record (old may be nil) within the size limits.
func (s *sagemakerStore) smfsStoreLocked(g *smfsGroup, rec *smfsRecord) *sagemakerDataError {
	old := g.records[rec.id]
	oldBytes := 0
	if old != nil {
		oldBytes = old.bytes
	} else if len(g.records) >= smfsMaxRecordsPerGroup {
		return smfsInvalid("Stackyard keeps at most %d records per feature group.", smfsMaxRecordsPerGroup)
	}
	delta := rec.bytes - oldBytes
	if g.bytes+delta > smfsMaxGroupBytes || s.smfsBytes+delta > smfsMaxStoredBytes {
		return smfsInvalid("The online store is full (%d bytes per feature group, %d in all).", smfsMaxGroupBytes, smfsMaxStoredBytes)
	}
	g.records[rec.id] = rec
	g.bytes += delta
	s.smfsBytes += delta
	return nil
}

type smfsWriteReq struct {
	Record       []smfsValue
	TargetStores []string
	TtlDuration  *smfsTTL
}

func (s *sagemakerStore) smfsPutLocked(ref string, values []smfsValue, targets []string, ttl *smfsTTL, now time.Time) *sagemakerDataError {
	g, err := s.smfsGroupDataLocked(ref)
	if err != nil {
		return err
	}
	online, err := g.targetsOnline(targets)
	if err != nil {
		return err
	}
	ordered, size, err := g.merge(nil, values)
	if err != nil {
		return err
	}
	id, et := smfsFind(ordered, g.RecordID), smfsFind(ordered, g.EventTime)
	if id == nil || et == nil {
		return smfsInvalid("A record needs the record identifier %s and the event time %s.", g.RecordID, g.EventTime)
	}
	at, err := g.parseEventTime(*et)
	if err != nil {
		return err
	}
	// ponytail: the offline store is not emulated; offline-only writes are
	// accepted and not kept.
	if !online {
		return nil
	}
	// An older event is history: it would only reach the offline store.
	if old := g.records[*id]; old != nil && old.eventTime.After(at) {
		return nil
	}
	if ttl == nil && g.Online != nil {
		ttl = g.Online.TtlDuration
	}
	return s.smfsStoreLocked(g, &smfsRecord{id: *id, values: ordered, eventTime: at, ttl: ttl.duration(), bytes: size})
}

func (g *smfsGroup) project(rec *smfsRecord, names []string) []smfsValue {
	if len(names) == 0 {
		return rec.values
	}
	out := []smfsValue{}
	for _, v := range rec.values {
		if slices.Contains(names, v.FeatureName) {
			out = append(out, v)
		}
	}
	return out
}

func (g *smfsGroup) checkNames(names []string) *sagemakerDataError {
	for _, n := range names {
		if g.def(n) == nil {
			return smfsInvalid("Feature %s is not defined in feature group %s.", n, g.Name)
		}
	}
	return nil
}

func (s *sagemakerStore) smfsGetRecordLocked(ref, id string, names []string, expires bool, now time.Time) (any, *sagemakerDataError) {
	g, err := s.smfsOnlineGroupLocked(ref)
	if err != nil {
		return nil, err
	}
	if err := g.checkNames(names); err != nil {
		return nil, err
	}
	rec := g.records[id]
	if rec == nil || !rec.live(now) {
		return map[string]any{}, nil // a missing record is an empty response
	}
	out := map[string]any{"Record": g.project(rec, names)}
	if exp := rec.expiresAt(); expires && !exp.IsZero() {
		out["ExpiresAt"] = exp.Format(time.RFC3339)
	}
	return out, nil
}

func (s *sagemakerStore) smfsDeleteRecordLocked(ref, id, eventTime, mode string, targets []string) *sagemakerDataError {
	g, err := s.smfsGroupDataLocked(ref)
	if err != nil {
		return err
	}
	online, err := g.targetsOnline(targets)
	if err != nil {
		return err
	}
	at, err := g.parseEventTime(eventTime)
	if err != nil {
		return err
	}
	rec := g.records[id]
	if !online || rec == nil {
		return nil
	}
	if at.Before(rec.eventTime) {
		if mode == "HardDelete" {
			return smfsInvalid("EventTime %s is older than the record's event time; the record was not deleted.", eventTime)
		}
		return nil // the existing, newer record stays
	}
	if mode == "HardDelete" {
		delete(g.records, id)
		g.bytes -= rec.bytes
		s.smfsBytes -= rec.bytes
		return nil
	}
	// A soft delete keeps only the identifier, marked deleted.
	idValue := id
	marker := &smfsRecord{id: id, deleted: true, eventTime: at, bytes: len(g.RecordID) + len(id),
		values: []smfsValue{{FeatureName: g.RecordID, ValueAsString: &idValue}}}
	return s.smfsStoreLocked(g, marker)
}

type smfsUpdateReq struct {
	RecordIdentifierValueAsString string
	Features                      []smfsValue
	TargetStores                  []string
	TtlDuration                   *smfsTTL
}

func (s *sagemakerStore) smfsUpdateLocked(ref string, req smfsUpdateReq, now time.Time) *sagemakerDataError {
	g, err := s.smfsOnlineGroupLocked(ref)
	if err != nil {
		return err
	}
	if len(req.TargetStores) > 0 && !slices.Contains(req.TargetStores, "OnlineStore") {
		return smfsInvalid("UpdateRecord works on the online store; TargetStores must include OnlineStore.")
	}
	if _, err := g.targetsOnline(req.TargetStores); err != nil {
		return err
	}
	if len(req.Features) > smfsMaxUpdateFeature {
		return smfsInvalid("Update at most %d features per call.", smfsMaxUpdateFeature)
	}
	rec := g.records[req.RecordIdentifierValueAsString]
	if rec == nil || !rec.live(now) {
		return smfsNotFound("Record %s does not exist in feature group %s.", req.RecordIdentifierValueAsString, g.Name)
	}
	at := rec.eventTime
	newTime := false
	for _, v := range req.Features {
		switch v.FeatureName {
		case g.RecordID:
			return smfsInvalid("The record identifier %s can't be updated.", g.RecordID)
		case g.EventTime:
			if v.ValueAsString == nil {
				return smfsInvalid("The event time %s can't be cleared.", g.EventTime)
			}
			if at, err = g.parseEventTime(*v.ValueAsString); err != nil {
				return err
			}
			newTime = true
		}
	}
	if req.TtlDuration != nil && !newTime {
		return smfsInvalid("TtlDuration needs an EventTime (the %s feature) in Features.", g.EventTime)
	}
	if at.Before(rec.eventTime) {
		e := smfsInvalid("EventTime is older than the record's current event time.")
		e.status, e.code = http.StatusConflict, "ConflictException"
		return e
	}
	values, size, err := g.merge(rec.values, req.Features)
	if err != nil {
		return err
	}
	ttl := rec.ttl
	if req.TtlDuration != nil {
		ttl = req.TtlDuration.duration()
	}
	return s.smfsStoreLocked(g, &smfsRecord{id: rec.id, values: values, eventTime: at, ttl: ttl, bytes: size})
}

type smfsListReq struct {
	MaxResults                int
	NextToken                 string
	IncludeSoftDeletedRecords bool
}

// smfsListRecords copies the matching identifiers under the lock (linear in
// the group's bounded record count) and sorts and pages them after.
func (s *sagemakerStore) smfsListRecords(ref string, req smfsListReq, now time.Time) (any, *sagemakerDataError) {
	s.mu.Lock()
	s.initResourcesLocked()
	g, err := s.smfsOnlineGroupLocked(ref)
	var ids []string
	if err == nil {
		ids = make([]string, 0, len(g.records))
		for id, rec := range g.records {
			if rec.live(now) || (rec.deleted && req.IncludeSoftDeletedRecords) {
				ids = append(ids, id)
			}
		}
	}
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	sort.Strings(ids)
	max := req.MaxResults
	if max <= 0 {
		max = smfsDefaultListPage
	}
	start, end, next, perr := sagemakerPage(len(ids), req.NextToken, max)
	if perr != nil {
		return nil, smfsInvalid("%s", perr.msg)
	}
	out := map[string]any{"RecordIdentifiers": ids[start:end]}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

type smfsBatchIdentifier struct {
	FeatureGroupName               string   `json:"FeatureGroupName"`
	RecordIdentifiersValueAsString []string `json:"RecordIdentifiersValueAsString"`
	FeatureNames                   []string `json:"FeatureNames,omitempty"`
}

type smfsBatchGetReq struct {
	Identifiers            []smfsBatchIdentifier
	ExpirationTimeResponse string
}

func (s *sagemakerStore) smfsBatchGetLocked(req smfsBatchGetReq, now time.Time) any {
	records, errs, unprocessed := []any{}, []any{}, []any{}
	budget := smfsMaxBatchGetBytes
	fail := func(group, id, code, msg string) {
		errs = append(errs, map[string]any{"FeatureGroupName": group, "RecordIdentifierValueAsString": id, "ErrorCode": code, "ErrorMessage": msg})
	}
	for _, ident := range req.Identifiers {
		g, err := s.smfsOnlineGroupLocked(ident.FeatureGroupName)
		if err == nil {
			err = g.checkNames(ident.FeatureNames)
		}
		for i, id := range ident.RecordIdentifiersValueAsString {
			if err != nil {
				fail(ident.FeatureGroupName, id, err.code, err.msg)
				continue
			}
			if budget <= 0 {
				rest := ident
				rest.RecordIdentifiersValueAsString = ident.RecordIdentifiersValueAsString[i:]
				unprocessed = append(unprocessed, rest)
				break
			}
			rec := g.records[id]
			if rec == nil || !rec.live(now) {
				fail(ident.FeatureGroupName, id, "ResourceNotFound", "Record "+id+" does not exist.")
				continue
			}
			budget -= rec.bytes
			item := map[string]any{"FeatureGroupName": ident.FeatureGroupName, "RecordIdentifierValueAsString": id,
				"Record": g.project(rec, ident.FeatureNames)}
			if exp := rec.expiresAt(); req.ExpirationTimeResponse == "Enabled" && !exp.IsZero() {
				item["ExpiresAt"] = exp.Format(time.RFC3339)
			}
			records = append(records, item)
		}
	}
	return map[string]any{"Records": records, "Errors": errs, "UnprocessedIdentifiers": unprocessed}
}

type smfsBatchEntry struct {
	FeatureGroupName string      `json:"FeatureGroupName"`
	Record           []smfsValue `json:"Record"`
	TargetStores     []string    `json:"TargetStores,omitempty"`
	TtlDuration      *smfsTTL    `json:"TtlDuration,omitempty"`
}

type smfsBatchWriteReq struct {
	Entries     []smfsBatchEntry
	TtlDuration *smfsTTL
}

func (s *sagemakerStore) smfsBatchWriteLocked(req smfsBatchWriteReq, now time.Time) any {
	errs := []any{}
	for _, e := range req.Entries {
		ttl := e.TtlDuration
		if ttl == nil {
			ttl = req.TtlDuration
		}
		if err := s.smfsPutLocked(e.FeatureGroupName, e.Record, e.TargetStores, ttl, now); err != nil {
			errs = append(errs, map[string]any{"Entry": e, "ErrorCode": err.code, "ErrorMessage": err.msg})
		}
	}
	return map[string]any{"Errors": errs, "UnprocessedEntries": []any{}}
}
