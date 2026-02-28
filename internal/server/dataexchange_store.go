package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type dataExchangeStore struct {
	mu sync.Mutex

	nextDataSetID     int64
	nextRevisionID    int64
	nextAssetID       int64
	nextJobID         int64
	nextEventActionID int64
	nextDataGrantID   int64

	dataSets           map[string]map[string]any
	revisionsByDataSet map[string]map[string]map[string]any
	assetsByRevision   map[string]map[string]map[string]map[string]any
	jobs               map[string]map[string]any
	eventActions       map[string]map[string]any
	dataGrants         map[string]map[string]any
	receivedDataGrants map[string]map[string]any
	tags               map[string]map[string]string
}

func newDataExchangeStore() *dataExchangeStore {
	s := &dataExchangeStore{
		nextDataSetID:      2,
		nextRevisionID:     2,
		nextAssetID:        2,
		nextJobID:          2,
		nextEventActionID:  2,
		nextDataGrantID:    2,
		dataSets:           map[string]map[string]any{},
		revisionsByDataSet: map[string]map[string]map[string]any{},
		assetsByRevision:   map[string]map[string]map[string]map[string]any{},
		jobs:               map[string]map[string]any{},
		eventActions:       map[string]map[string]any{},
		dataGrants:         map[string]map[string]any{},
		receivedDataGrants: map[string]map[string]any{},
		tags:               map[string]map[string]string{},
	}
	s.seedLocked(time.Now().UTC())
	return s
}

func (s *dataExchangeStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ctx := dataExchangeMergeMaps(payload, pathParams, query)

	dataSetID := dataExchangeString(ctx, "DataSetId", "ds-000001")
	revisionID := dataExchangeString(ctx, "RevisionId", "rev-000001")
	assetID := dataExchangeString(ctx, "AssetId", "asset-000001")
	jobID := dataExchangeString(ctx, "JobId", "job-000001")
	eventActionID := dataExchangeString(ctx, "EventActionId", "ea-000001")
	dataGrantID := dataExchangeString(ctx, "DataGrantId", "dg-000001")
	dataGrantArn := dataExchangeString(ctx, "DataGrantArn", dataExchangeDataGrantARN(dataGrantID))
	resourceArn := dataExchangeString(ctx, "ResourceArn", dataExchangeDataSetARN(dataSetID))

	s.ensureDataSetLocked(dataSetID, now)
	s.ensureRevisionLocked(dataSetID, revisionID, now)
	s.ensureAssetLocked(dataSetID, revisionID, assetID, now)
	s.ensureJobLocked(jobID, dataSetID, revisionID, now)
	s.ensureEventActionLocked(eventActionID, now)
	s.ensureDataGrantLocked(dataGrantID, dataGrantArn, now)
	s.ensureReceivedDataGrantLocked(dataGrantArn, dataGrantID, now)
	s.ensureTagsLocked(resourceArn)

	switch action {
	case "CreateDataSet":
		id := dataExchangeString(ctx, "Id", "")
		if id == "" {
			id = fmt.Sprintf("ds-%06d", s.nextDataSetID)
			s.nextDataSetID++
		}
		item := s.ensureDataSetLocked(id, now)
		if name := dataExchangeString(ctx, "Name", ""); name != "" {
			item["Name"] = name
		}
		if desc := dataExchangeString(ctx, "Description", ""); desc != "" {
			item["Description"] = desc
		}
		item["UpdatedAt"] = now.Format(time.RFC3339)
		return dataExchangeCloneMap(item)

	case "GetDataSet":
		return dataExchangeCloneMap(s.ensureDataSetLocked(dataSetID, now))

	case "UpdateDataSet":
		item := s.ensureDataSetLocked(dataSetID, now)
		if name := dataExchangeString(ctx, "Name", ""); name != "" {
			item["Name"] = name
		}
		if desc := dataExchangeString(ctx, "Description", ""); desc != "" {
			item["Description"] = desc
		}
		item["UpdatedAt"] = now.Format(time.RFC3339)
		return dataExchangeCloneMap(item)

	case "DeleteDataSet":
		delete(s.dataSets, dataSetID)
		delete(s.revisionsByDataSet, dataSetID)
		delete(s.assetsByRevision, dataSetID)
		return map[string]any{}

	case "ListDataSets":
		items := make([]any, 0, len(s.dataSets))
		for _, id := range dataExchangeSortedKeysAny(s.dataSets) {
			items = append(items, dataExchangeCloneMap(s.dataSets[id]))
		}
		return map[string]any{"DataSets": items, "NextToken": ""}

	case "CreateRevision":
		id := dataExchangeString(ctx, "Id", "")
		if id == "" {
			id = fmt.Sprintf("rev-%06d", s.nextRevisionID)
			s.nextRevisionID++
		}
		item := s.ensureRevisionLocked(dataSetID, id, now)
		if comment := dataExchangeString(ctx, "Comment", ""); comment != "" {
			item["Comment"] = comment
		}
		item["UpdatedAt"] = now.Format(time.RFC3339)
		return dataExchangeCloneMap(item)

	case "GetRevision":
		return dataExchangeCloneMap(s.ensureRevisionLocked(dataSetID, revisionID, now))

	case "UpdateRevision":
		item := s.ensureRevisionLocked(dataSetID, revisionID, now)
		if comment := dataExchangeString(ctx, "Comment", ""); comment != "" {
			item["Comment"] = comment
		}
		item["UpdatedAt"] = now.Format(time.RFC3339)
		return dataExchangeCloneMap(item)

	case "DeleteRevision":
		if revisions := s.revisionsByDataSet[dataSetID]; revisions != nil {
			delete(revisions, revisionID)
		}
		if revisions := s.assetsByRevision[dataSetID]; revisions != nil {
			delete(revisions, revisionID)
		}
		return map[string]any{}

	case "RevokeRevision":
		item := s.ensureRevisionLocked(dataSetID, revisionID, now)
		item["Finalized"] = true
		item["Revoked"] = true
		item["State"] = "REVOKED"
		item["UpdatedAt"] = now.Format(time.RFC3339)
		return dataExchangeCloneMap(item)

	case "ListDataSetRevisions":
		items := []any{}
		if revisions := s.revisionsByDataSet[dataSetID]; revisions != nil {
			for _, id := range dataExchangeSortedKeysAny(revisions) {
				items = append(items, dataExchangeCloneMap(revisions[id]))
			}
		}
		return map[string]any{"Revisions": items, "NextToken": ""}

	case "GetAsset":
		return dataExchangeCloneMap(s.ensureAssetLocked(dataSetID, revisionID, assetID, now))

	case "UpdateAsset":
		item := s.ensureAssetLocked(dataSetID, revisionID, assetID, now)
		if name := dataExchangeString(ctx, "Name", ""); name != "" {
			item["Name"] = name
		}
		item["UpdatedAt"] = now.Format(time.RFC3339)
		return dataExchangeCloneMap(item)

	case "DeleteAsset":
		if revAssets := s.assetsByRevision[dataSetID]; revAssets != nil {
			if assets := revAssets[revisionID]; assets != nil {
				delete(assets, assetID)
			}
		}
		return map[string]any{}

	case "ListRevisionAssets":
		items := []any{}
		if revAssets := s.assetsByRevision[dataSetID]; revAssets != nil {
			if assets := revAssets[revisionID]; assets != nil {
				for _, id := range dataExchangeSortedKeysAny(assets) {
					items = append(items, dataExchangeCloneMap(assets[id]))
				}
			}
		}
		return map[string]any{"Assets": items, "NextToken": ""}

	case "CreateJob":
		id := dataExchangeString(ctx, "Id", "")
		if id == "" {
			id = fmt.Sprintf("job-%06d", s.nextJobID)
			s.nextJobID++
		}
		item := s.ensureJobLocked(id, dataSetID, revisionID, now)
		if kind := dataExchangeString(ctx, "Type", ""); kind != "" {
			item["Type"] = kind
		}
		item["State"] = "WAITING"
		item["UpdatedAt"] = now.Format(time.RFC3339)
		return dataExchangeCloneMap(item)

	case "GetJob":
		return dataExchangeCloneMap(s.ensureJobLocked(jobID, dataSetID, revisionID, now))

	case "StartJob":
		item := s.ensureJobLocked(jobID, dataSetID, revisionID, now)
		item["State"] = "IN_PROGRESS"
		item["StartedAt"] = now.Format(time.RFC3339)
		item["UpdatedAt"] = now.Format(time.RFC3339)
		return dataExchangeCloneMap(item)

	case "CancelJob":
		item := s.ensureJobLocked(jobID, dataSetID, revisionID, now)
		item["State"] = "CANCELLED"
		item["UpdatedAt"] = now.Format(time.RFC3339)
		return dataExchangeCloneMap(item)

	case "ListJobs":
		items := make([]any, 0, len(s.jobs))
		for _, id := range dataExchangeSortedKeysAny(s.jobs) {
			items = append(items, dataExchangeCloneMap(s.jobs[id]))
		}
		return map[string]any{"Jobs": items, "NextToken": ""}

	case "CreateEventAction":
		id := dataExchangeString(ctx, "Id", "")
		if id == "" {
			id = fmt.Sprintf("ea-%06d", s.nextEventActionID)
			s.nextEventActionID++
		}
		item := s.ensureEventActionLocked(id, now)
		if name := dataExchangeString(ctx, "Name", ""); name != "" {
			item["Name"] = name
		}
		item["UpdatedAt"] = now.Format(time.RFC3339)
		return dataExchangeCloneMap(item)

	case "GetEventAction":
		return dataExchangeCloneMap(s.ensureEventActionLocked(eventActionID, now))

	case "UpdateEventAction":
		item := s.ensureEventActionLocked(eventActionID, now)
		if name := dataExchangeString(ctx, "Name", ""); name != "" {
			item["Name"] = name
		}
		item["UpdatedAt"] = now.Format(time.RFC3339)
		return dataExchangeCloneMap(item)

	case "DeleteEventAction":
		delete(s.eventActions, eventActionID)
		return map[string]any{}

	case "ListEventActions":
		items := make([]any, 0, len(s.eventActions))
		for _, id := range dataExchangeSortedKeysAny(s.eventActions) {
			items = append(items, dataExchangeCloneMap(s.eventActions[id]))
		}
		return map[string]any{"EventActions": items, "NextToken": ""}

	case "CreateDataGrant":
		id := dataExchangeString(ctx, "DataGrantId", "")
		if id == "" {
			id = fmt.Sprintf("dg-%06d", s.nextDataGrantID)
			s.nextDataGrantID++
		}
		arn := dataExchangeString(ctx, "DataGrantArn", dataExchangeDataGrantARN(id))
		item := s.ensureDataGrantLocked(id, arn, now)
		item["UpdatedAt"] = now.Format(time.RFC3339)
		s.ensureReceivedDataGrantLocked(arn, id, now)
		return dataExchangeCloneMap(item)

	case "GetDataGrant":
		return dataExchangeCloneMap(s.ensureDataGrantLocked(dataGrantID, dataExchangeDataGrantARN(dataGrantID), now))

	case "DeleteDataGrant":
		delete(s.dataGrants, dataGrantID)
		return map[string]any{}

	case "ListDataGrants":
		items := make([]any, 0, len(s.dataGrants))
		for _, id := range dataExchangeSortedKeysAny(s.dataGrants) {
			items = append(items, dataExchangeCloneMap(s.dataGrants[id]))
		}
		return map[string]any{"DataGrantSummaries": items, "NextToken": ""}

	case "GetReceivedDataGrant":
		return dataExchangeCloneMap(s.ensureReceivedDataGrantLocked(dataGrantArn, dataGrantID, now))

	case "ListReceivedDataGrants":
		items := make([]any, 0, len(s.receivedDataGrants))
		filter := strings.ToUpper(strings.TrimSpace(dataExchangeString(ctx, "AcceptanceState", "")))
		for _, arn := range dataExchangeSortedKeysAny(s.receivedDataGrants) {
			item := s.receivedDataGrants[arn]
			state := strings.ToUpper(strings.TrimSpace(dataExchangeString(item, "AcceptanceState", "PENDING")))
			if filter != "" && filter != state {
				continue
			}
			items = append(items, dataExchangeCloneMap(item))
		}
		return map[string]any{"DataGrantSummaries": items, "NextToken": ""}

	case "AcceptDataGrant":
		item := s.ensureReceivedDataGrantLocked(dataGrantArn, dataGrantID, now)
		item["AcceptanceState"] = "ACCEPTED"
		item["AcceptedAt"] = now.Format(time.RFC3339)
		if id := dataExchangeString(item, "DataGrantId", ""); id != "" {
			if grant := s.dataGrants[id]; grant != nil {
				grant["AcceptanceState"] = "ACCEPTED"
				grant["AcceptedAt"] = now.Format(time.RFC3339)
			}
		}
		return dataExchangeCloneMap(item)

	case "SendDataSetNotification":
		return map[string]any{
			"DataSetId": dataSetID,
			"State":     "SENT",
			"Message":   "notification dispatched",
		}

	case "SendApiAsset":
		return map[string]any{
			"AssetId":    assetID,
			"DataSetId":  dataSetID,
			"RevisionId": revisionID,
			"State":      "SENT",
		}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceArn)
		for k, v := range dataExchangeExtractTags(payload) {
			tags[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		tags := s.ensureTagsLocked(resourceArn)
		for _, key := range dataExchangeExtractTagKeys(payload, query) {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"Tags": dataExchangeCloneStringMap(s.ensureTagsLocked(resourceArn))}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"Items": []any{}, "NextToken": ""}
	}
	if strings.HasPrefix(action, "Get") || strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Update") || strings.HasPrefix(action, "Start") || strings.HasPrefix(action, "Accept") || strings.HasPrefix(action, "Send") {
		return map[string]any{"Action": action, "Status": "SUCCESS"}
	}
	return map[string]any{}
}

func (s *dataExchangeStore) seedLocked(now time.Time) {
	ds := s.ensureDataSetLocked("ds-000001", now)
	rev := s.ensureRevisionLocked("ds-000001", "rev-000001", now)
	asset := s.ensureAssetLocked("ds-000001", "rev-000001", "asset-000001", now)
	s.ensureJobLocked("job-000001", "ds-000001", "rev-000001", now)
	s.ensureEventActionLocked("ea-000001", now)
	grantArn := dataExchangeDataGrantARN("dg-000001")
	s.ensureDataGrantLocked("dg-000001", grantArn, now)
	s.ensureReceivedDataGrantLocked(grantArn, "dg-000001", now)
	s.ensureTagsLocked(dataExchangeDataSetARN(dataExchangeString(ds, "Id", "ds-000001")))
	s.ensureTagsLocked(dataExchangeDataSetARN(dataExchangeString(rev, "DataSetId", "ds-000001")))
	s.ensureTagsLocked(dataExchangeDataSetARN(dataExchangeString(asset, "DataSetId", "ds-000001")))
}

func (s *dataExchangeStore) ensureDataSetLocked(dataSetID string, now time.Time) map[string]any {
	id := dataExchangeNormalizeID(dataSetID, "ds-000001")
	if existing := s.dataSets[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"Id":          id,
		"Arn":         dataExchangeDataSetARN(id),
		"Name":        "stackyard-data-set-" + id,
		"Description": "Stackyard Data Exchange data set",
		"Origin":      "OWNED",
		"CreatedAt":   now.Format(time.RFC3339),
		"UpdatedAt":   now.Format(time.RFC3339),
	}
	s.dataSets[id] = item
	return item
}

func (s *dataExchangeStore) ensureRevisionLocked(dataSetID, revisionID string, now time.Time) map[string]any {
	dsID := dataExchangeNormalizeID(dataSetID, "ds-000001")
	revID := dataExchangeNormalizeID(revisionID, "rev-000001")
	revisions := s.revisionsByDataSet[dsID]
	if revisions == nil {
		revisions = map[string]map[string]any{}
		s.revisionsByDataSet[dsID] = revisions
	}
	if existing := revisions[revID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"Id":        revID,
		"Arn":       dataExchangeRevisionARN(dsID, revID),
		"DataSetId": dsID,
		"Comment":   "Stackyard revision " + revID,
		"Finalized": false,
		"State":     "ACTIVE",
		"CreatedAt": now.Format(time.RFC3339),
		"UpdatedAt": now.Format(time.RFC3339),
	}
	revisions[revID] = item
	return item
}

func (s *dataExchangeStore) ensureAssetLocked(dataSetID, revisionID, assetID string, now time.Time) map[string]any {
	dsID := dataExchangeNormalizeID(dataSetID, "ds-000001")
	revID := dataExchangeNormalizeID(revisionID, "rev-000001")
	aID := dataExchangeNormalizeID(assetID, "asset-000001")
	byDataSet := s.assetsByRevision[dsID]
	if byDataSet == nil {
		byDataSet = map[string]map[string]map[string]any{}
		s.assetsByRevision[dsID] = byDataSet
	}
	byRevision := byDataSet[revID]
	if byRevision == nil {
		byRevision = map[string]map[string]any{}
		byDataSet[revID] = byRevision
	}
	if existing := byRevision[aID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"Id":         aID,
		"Arn":        dataExchangeAssetARN(dsID, revID, aID),
		"DataSetId":  dsID,
		"RevisionId": revID,
		"Name":       "stackyard-asset-" + aID,
		"AssetType":  "S3_SNAPSHOT",
		"CreatedAt":  now.Format(time.RFC3339),
		"UpdatedAt":  now.Format(time.RFC3339),
	}
	byRevision[aID] = item
	return item
}

func (s *dataExchangeStore) ensureJobLocked(jobID, dataSetID, revisionID string, now time.Time) map[string]any {
	id := dataExchangeNormalizeID(jobID, "job-000001")
	if existing := s.jobs[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"Id":         id,
		"Arn":        dataExchangeJobARN(id),
		"DataSetId":  dataExchangeNormalizeID(dataSetID, "ds-000001"),
		"RevisionId": dataExchangeNormalizeID(revisionID, "rev-000001"),
		"Type":       "IMPORT_ASSETS_FROM_S3",
		"State":      "COMPLETED",
		"CreatedAt":  now.Add(-2 * time.Minute).Format(time.RFC3339),
		"UpdatedAt":  now.Format(time.RFC3339),
	}
	s.jobs[id] = item
	return item
}

func (s *dataExchangeStore) ensureEventActionLocked(eventActionID string, now time.Time) map[string]any {
	id := dataExchangeNormalizeID(eventActionID, "ea-000001")
	if existing := s.eventActions[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"Id":        id,
		"Arn":       dataExchangeEventActionARN(id),
		"Name":      "stackyard-event-action-" + id,
		"State":     "ENABLED",
		"CreatedAt": now.Format(time.RFC3339),
		"UpdatedAt": now.Format(time.RFC3339),
	}
	s.eventActions[id] = item
	return item
}

func (s *dataExchangeStore) ensureDataGrantLocked(dataGrantID, dataGrantArn string, now time.Time) map[string]any {
	id := dataExchangeNormalizeID(dataGrantID, "dg-000001")
	arn := strings.TrimSpace(dataGrantArn)
	if arn == "" {
		arn = dataExchangeDataGrantARN(id)
	}
	if existing := s.dataGrants[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"DataGrantId":     id,
		"DataGrantArn":    arn,
		"Name":            "stackyard-data-grant-" + id,
		"AcceptanceState": "PENDING",
		"CreatedAt":       now.Format(time.RFC3339),
		"UpdatedAt":       now.Format(time.RFC3339),
	}
	s.dataGrants[id] = item
	return item
}

func (s *dataExchangeStore) ensureReceivedDataGrantLocked(dataGrantArn, dataGrantID string, now time.Time) map[string]any {
	arn := strings.TrimSpace(dataGrantArn)
	if arn == "" {
		arn = dataExchangeDataGrantARN(dataExchangeNormalizeID(dataGrantID, "dg-000001"))
	}
	if existing := s.receivedDataGrants[arn]; existing != nil {
		return existing
	}
	id := dataExchangeNormalizeID(dataGrantID, "dg-000001")
	item := map[string]any{
		"DataGrantId":     id,
		"DataGrantArn":    arn,
		"Name":            "stackyard-received-grant-" + id,
		"AcceptanceState": "PENDING",
		"CreatedAt":       now.Format(time.RFC3339),
		"UpdatedAt":       now.Format(time.RFC3339),
	}
	s.receivedDataGrants[arn] = item
	return item
}

func (s *dataExchangeStore) ensureTagsLocked(resourceArn string) map[string]string {
	arn := strings.TrimSpace(resourceArn)
	if arn == "" {
		arn = dataExchangeDataSetARN("ds-000001")
	}
	if existing := s.tags[arn]; existing != nil {
		return existing
	}
	out := map[string]string{"service": "dataexchange"}
	s.tags[arn] = out
	return out
}

func dataExchangeMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if len(values) == 1 {
			out[key] = values[0]
		} else {
			dup := make([]string, len(values))
			copy(dup, values)
			out[key] = dup
		}
	}
	return out
}

func dataExchangeString(values map[string]any, key, def string) string {
	if values == nil {
		return def
	}
	for existingKey, raw := range values {
		if !strings.EqualFold(strings.TrimSpace(existingKey), strings.TrimSpace(key)) {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(raw))
		if text != "" {
			return text
		}
	}
	return def
}

func dataExchangeExtractTags(payload map[string]any) map[string]string {
	for _, key := range []string{"Tags", "tags"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case map[string]string:
			out := map[string]string{}
			for k, v := range typed {
				k = strings.TrimSpace(k)
				if k == "" {
					continue
				}
				out[k] = strings.TrimSpace(v)
			}
			return out
		case map[string]any:
			out := map[string]string{}
			for k, v := range typed {
				k = strings.TrimSpace(k)
				if k == "" {
					continue
				}
				out[k] = strings.TrimSpace(fmt.Sprint(v))
			}
			return out
		}
	}
	return map[string]string{}
}

func dataExchangeExtractTagKeys(payload map[string]any, query url.Values) []string {
	for _, key := range []string{"tagKeys", "TagKeys"} {
		if raw, ok := payload[key]; ok {
			switch typed := raw.(type) {
			case string:
				return dataExchangeSplitCSV(typed)
			case []string:
				out := make([]string, 0, len(typed))
				for _, item := range typed {
					item = strings.TrimSpace(item)
					if item != "" {
						out = append(out, item)
					}
				}
				sort.Strings(out)
				return out
			case []any:
				out := make([]string, 0, len(typed))
				for _, item := range typed {
					token := strings.TrimSpace(fmt.Sprint(item))
					if token != "" {
						out = append(out, token)
					}
				}
				sort.Strings(out)
				return out
			}
		}
	}
	if values := query.Get("tagKeys"); strings.TrimSpace(values) != "" {
		return dataExchangeSplitCSV(values)
	}
	return nil
}

func dataExchangeSplitCSV(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token != "" {
			out = append(out, token)
		}
	}
	sort.Strings(out)
	return out
}

func dataExchangeSortedKeysAny[V any](in map[string]V) []string {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func dataExchangeCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func dataExchangeCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func dataExchangeNormalizeID(id, def string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return def
	}
	return id
}

func dataExchangeDataSetARN(dataSetID string) string {
	return fmt.Sprintf("arn:aws:dataexchange:us-east-1:123456789012:data-sets/%s", dataExchangeNormalizeID(dataSetID, "ds-000001"))
}

func dataExchangeRevisionARN(dataSetID, revisionID string) string {
	return fmt.Sprintf(
		"arn:aws:dataexchange:us-east-1:123456789012:data-sets/%s/revisions/%s",
		dataExchangeNormalizeID(dataSetID, "ds-000001"),
		dataExchangeNormalizeID(revisionID, "rev-000001"),
	)
}

func dataExchangeAssetARN(dataSetID, revisionID, assetID string) string {
	return fmt.Sprintf(
		"arn:aws:dataexchange:us-east-1:123456789012:data-sets/%s/revisions/%s/assets/%s",
		dataExchangeNormalizeID(dataSetID, "ds-000001"),
		dataExchangeNormalizeID(revisionID, "rev-000001"),
		dataExchangeNormalizeID(assetID, "asset-000001"),
	)
}

func dataExchangeJobARN(jobID string) string {
	return fmt.Sprintf("arn:aws:dataexchange:us-east-1:123456789012:jobs/%s", dataExchangeNormalizeID(jobID, "job-000001"))
}

func dataExchangeEventActionARN(eventActionID string) string {
	return fmt.Sprintf("arn:aws:dataexchange:us-east-1:123456789012:event-actions/%s", dataExchangeNormalizeID(eventActionID, "ea-000001"))
}

func dataExchangeDataGrantARN(dataGrantID string) string {
	return fmt.Sprintf("arn:aws:dataexchange:us-east-1:123456789012:data-grants/%s", dataExchangeNormalizeID(dataGrantID, "dg-000001"))
}
