package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	locationDefaultRegion    = "us-east-1"
	locationDefaultAccountID = "123456789012"
)

type locationStore struct {
	mu sync.Mutex

	maps                map[string]map[string]any
	placeIndexes        map[string]map[string]any
	routeCalculators    map[string]map[string]any
	trackers            map[string]map[string]any
	geofenceCollections map[string]map[string]any
	geofences           map[string]map[string]map[string]any
	keys                map[string]map[string]any
	tags                map[string]map[string]string

	trackerDevicePositions map[string]map[string]map[string]any
	trackerConsumers       map[string]map[string]struct{}
}

func newLocationStore() *locationStore {
	now := time.Now().UTC()
	s := &locationStore{
		maps:                   map[string]map[string]any{},
		placeIndexes:           map[string]map[string]any{},
		routeCalculators:       map[string]map[string]any{},
		trackers:               map[string]map[string]any{},
		geofenceCollections:    map[string]map[string]any{},
		geofences:              map[string]map[string]map[string]any{},
		keys:                   map[string]map[string]any{},
		tags:                   map[string]map[string]string{},
		trackerDevicePositions: map[string]map[string]map[string]any{},
		trackerConsumers:       map[string]map[string]struct{}{},
	}

	s.ensureMapLocked("stackyard-map", now)
	s.ensurePlaceIndexLocked("stackyard-place-index", now)
	s.ensureRouteCalculatorLocked("stackyard-route-calculator", now)
	s.ensureTrackerLocked("stackyard-tracker", now)
	s.ensureGeofenceCollectionLocked("stackyard-geofence-collection", now)
	s.ensureKeyLocked("stackyard-api-key", now)
	s.ensureGeofenceLocked("stackyard-geofence-collection", "stackyard-geofence", now)
	s.upsertDevicePositionLocked("stackyard-tracker", "device-000001", []any{-122.341, 47.609}, now)

	return s
}

func (s *locationStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	mapName := s.nameFromPathOrPayload(pathParams, payload, []string{"MapName"}, "stackyard-map")
	indexName := s.nameFromPathOrPayload(pathParams, payload, []string{"IndexName"}, "stackyard-place-index")
	calculatorName := s.nameFromPathOrPayload(pathParams, payload, []string{"CalculatorName"}, "stackyard-route-calculator")
	trackerName := s.nameFromPathOrPayload(pathParams, payload, []string{"TrackerName"}, "stackyard-tracker")
	collectionName := s.nameFromPathOrPayload(pathParams, payload, []string{"CollectionName"}, "stackyard-geofence-collection")
	keyName := s.nameFromPathOrPayload(pathParams, payload, []string{"KeyName"}, "stackyard-api-key")
	geofenceID := s.nameFromPathOrPayload(pathParams, payload, []string{"GeofenceId"}, "stackyard-geofence")
	deviceID := s.nameFromPathOrPayload(pathParams, payload, []string{"DeviceId"}, "device-000001")
	placeID := s.nameFromPathOrPayload(pathParams, payload, []string{"PlaceId"}, "place-000001")
	resourceARN := s.nameFromPathOrPayload(pathParams, payload, []string{"ResourceArn"}, locationResourceARN("map", mapName))
	consumerArn := s.nameFromPathOrPayload(pathParams, payload, []string{"ConsumerArn"}, "arn:aws:kinesis:us-east-1:123456789012:stream/stackyard-tracker-consumer")

	m := s.ensureMapLocked(mapName, now)
	idx := s.ensurePlaceIndexLocked(indexName, now)
	calc := s.ensureRouteCalculatorLocked(calculatorName, now)
	tracker := s.ensureTrackerLocked(trackerName, now)
	collection := s.ensureGeofenceCollectionLocked(collectionName, now)
	key := s.ensureKeyLocked(keyName, now)
	geofence := s.ensureGeofenceLocked(collectionName, geofenceID, now)
	s.ensureTagsLocked(resourceARN)
	s.ensureTrackerPositionMapLocked(trackerName)
	s.ensureTrackerConsumerSetLocked(trackerName)

	s.applyUpdatesLocked(action, payload, m, idx, calc, tracker, collection, key, geofence, now)

	switch action {
	case "CreateMap":
		name := locationPayloadString(payload, []string{"MapName"}, mapName)
		created := s.ensureMapLocked(name, now)
		s.mergeTagsLocked(locationMapAny(payload, "Tags", "tags"), locationAnyString(created, "MapArn", ""))
		return map[string]any{"MapArn": created["MapArn"], "MapName": created["MapName"], "CreateTime": created["CreateTime"]}
	case "CreatePlaceIndex":
		name := locationPayloadString(payload, []string{"IndexName"}, indexName)
		created := s.ensurePlaceIndexLocked(name, now)
		s.mergeTagsLocked(locationMapAny(payload, "Tags", "tags"), locationAnyString(created, "IndexArn", ""))
		return map[string]any{"IndexArn": created["IndexArn"], "IndexName": created["IndexName"], "CreateTime": created["CreateTime"]}
	case "CreateRouteCalculator":
		name := locationPayloadString(payload, []string{"CalculatorName"}, calculatorName)
		created := s.ensureRouteCalculatorLocked(name, now)
		s.mergeTagsLocked(locationMapAny(payload, "Tags", "tags"), locationAnyString(created, "CalculatorArn", ""))
		return map[string]any{"CalculatorArn": created["CalculatorArn"], "CalculatorName": created["CalculatorName"], "CreateTime": created["CreateTime"]}
	case "CreateTracker":
		name := locationPayloadString(payload, []string{"TrackerName"}, trackerName)
		created := s.ensureTrackerLocked(name, now)
		s.mergeTagsLocked(locationMapAny(payload, "Tags", "tags"), locationAnyString(created, "TrackerArn", ""))
		return map[string]any{"TrackerArn": created["TrackerArn"], "TrackerName": created["TrackerName"], "CreateTime": created["CreateTime"]}
	case "CreateGeofenceCollection":
		name := locationPayloadString(payload, []string{"CollectionName"}, collectionName)
		created := s.ensureGeofenceCollectionLocked(name, now)
		s.mergeTagsLocked(locationMapAny(payload, "Tags", "tags"), locationAnyString(created, "CollectionArn", ""))
		return map[string]any{"CollectionArn": created["CollectionArn"], "CollectionName": created["CollectionName"], "CreateTime": created["CreateTime"]}
	case "CreateKey":
		name := locationPayloadString(payload, []string{"KeyName"}, keyName)
		created := s.ensureKeyLocked(name, now)
		s.mergeTagsLocked(locationMapAny(payload, "Tags", "tags"), locationAnyString(created, "KeyArn", ""))
		return map[string]any{"KeyArn": created["KeyArn"], "KeyName": created["KeyName"], "CreateTime": created["CreateTime"]}

	case "DeleteMap":
		s.deleteMapLocked(mapName)
		return map[string]any{}
	case "DeletePlaceIndex":
		s.deletePlaceIndexLocked(indexName)
		return map[string]any{}
	case "DeleteRouteCalculator":
		s.deleteRouteCalculatorLocked(calculatorName)
		return map[string]any{}
	case "DeleteTracker":
		s.deleteTrackerLocked(trackerName)
		return map[string]any{}
	case "DeleteGeofenceCollection":
		s.deleteGeofenceCollectionLocked(collectionName)
		return map[string]any{}
	case "DeleteKey":
		s.deleteKeyLocked(keyName)
		return map[string]any{}

	case "DescribeMap":
		return locationCloneMap(m)
	case "DescribePlaceIndex":
		return locationCloneMap(idx)
	case "DescribeRouteCalculator":
		return locationCloneMap(calc)
	case "DescribeTracker":
		return locationCloneMap(tracker)
	case "DescribeGeofenceCollection":
		return locationCloneMap(collection)
	case "DescribeKey":
		return locationCloneMap(key)

	case "ListMaps":
		return map[string]any{"Entries": s.listResourceSummariesLocked(s.maps), "NextToken": ""}
	case "ListPlaceIndexes":
		return map[string]any{"Entries": s.listResourceSummariesLocked(s.placeIndexes), "NextToken": ""}
	case "ListRouteCalculators":
		return map[string]any{"Entries": s.listResourceSummariesLocked(s.routeCalculators), "NextToken": ""}
	case "ListTrackers":
		return map[string]any{"Entries": s.listResourceSummariesLocked(s.trackers), "NextToken": ""}
	case "ListGeofenceCollections":
		return map[string]any{"Entries": s.listResourceSummariesLocked(s.geofenceCollections), "NextToken": ""}
	case "ListKeys":
		return map[string]any{"Entries": s.listResourceSummariesLocked(s.keys), "NextToken": ""}

	case "UpdateMap", "UpdatePlaceIndex", "UpdateRouteCalculator", "UpdateTracker", "UpdateGeofenceCollection", "UpdateKey":
		return map[string]any{}

	case "PutGeofence":
		item := s.ensureGeofenceLocked(collectionName, geofenceID, now)
		if geometry, ok := locationPayloadAny(payload, []string{"GeofenceGeometry", "Geometry"}); ok {
			item["Geometry"] = geometry
		}
		item["UpdateTime"] = locationTime(now)
		return map[string]any{"CreateTime": item["CreateTime"], "UpdateTime": item["UpdateTime"]}
	case "GetGeofence":
		return locationCloneMap(geofence)
	case "ListGeofences":
		items := []any{}
		for _, id := range s.sortedGeofenceIDsLocked(collectionName) {
			items = append(items, locationCloneMap(s.geofences[collectionName][id]))
		}
		return map[string]any{"Entries": items, "NextToken": ""}
	case "BatchPutGeofence":
		successes := []any{}
		for _, entry := range locationPayloadSlice(payload, "Entries") {
			em, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			id := locationPayloadString(em, []string{"GeofenceId"}, "")
			if id == "" {
				continue
			}
			item := s.ensureGeofenceLocked(collectionName, id, now)
			if geometry, ok := locationPayloadAny(em, []string{"GeofenceGeometry", "Geometry"}); ok {
				item["Geometry"] = geometry
			}
			item["UpdateTime"] = locationTime(now)
			successes = append(successes, map[string]any{"GeofenceId": id, "CreateTime": item["CreateTime"], "UpdateTime": item["UpdateTime"]})
		}
		return map[string]any{"Successes": successes, "Errors": []any{}}
	case "BatchDeleteGeofence":
		for _, id := range locationPayloadStringSlice(payload, "GeofenceIds") {
			if geofencesByID := s.geofences[collectionName]; geofencesByID != nil {
				delete(geofencesByID, id)
			}
		}
		return map[string]any{"Errors": []any{}}
	case "BatchEvaluateGeofences":
		results := []any{}
		for _, entry := range locationPayloadSlice(payload, "DevicePositionUpdates") {
			em, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			id := locationPayloadString(em, []string{"DeviceId"}, "device-000001")
			results = append(results, map[string]any{"DeviceId": id, "SampleTime": locationTime(now), "Position": locationPayloadAnyOr(em, "Position", []any{-122.341, 47.609}), "Events": []any{}})
		}
		return map[string]any{"Results": results, "Errors": []any{}}
	case "ForecastGeofenceEvents":
		return map[string]any{"ForecastedEvents": []any{}}

	case "BatchUpdateDevicePosition":
		for _, entry := range locationPayloadSlice(payload, "Updates") {
			em, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			id := locationPayloadString(em, []string{"DeviceId"}, "")
			if id == "" {
				continue
			}
			position := locationPayloadAnyOr(em, "Position", []any{-122.341, 47.609})
			posSlice, _ := position.([]any)
			s.upsertDevicePositionLocked(trackerName, id, posSlice, now)
		}
		return map[string]any{"Errors": []any{}}
	case "BatchGetDevicePosition":
		items := []any{}
		for _, id := range locationPayloadStringSlice(payload, "DeviceIds") {
			items = append(items, locationCloneMap(s.ensureDevicePositionLocked(trackerName, id, now)))
		}
		if len(items) == 0 {
			items = append(items, locationCloneMap(s.ensureDevicePositionLocked(trackerName, deviceID, now)))
		}
		return map[string]any{"DevicePositions": items, "Errors": []any{}}
	case "GetDevicePosition":
		return locationCloneMap(s.ensureDevicePositionLocked(trackerName, deviceID, now))
	case "GetDevicePositionHistory":
		fallthrough
	case "ListDevicePositions":
		items := []any{}
		for _, id := range s.sortedDeviceIDsLocked(trackerName) {
			items = append(items, locationCloneMap(s.trackerDevicePositions[trackerName][id]))
		}
		if len(items) == 0 {
			items = append(items, locationCloneMap(s.ensureDevicePositionLocked(trackerName, deviceID, now)))
		}
		return map[string]any{"DevicePositions": items, "NextToken": ""}
	case "BatchDeleteDevicePositionHistory":
		for _, id := range locationPayloadStringSlice(payload, "DeviceIds") {
			if positions := s.trackerDevicePositions[trackerName]; positions != nil {
				delete(positions, id)
			}
		}
		return map[string]any{"Errors": []any{}}
	case "VerifyDevicePosition":
		entry := s.ensureDevicePositionLocked(trackerName, deviceID, now)
		return map[string]any{
			"InferredState": map[string]any{
				"DeviceId":          entry["DeviceId"],
				"Position":          entry["Position"],
				"SampleTime":        entry["SampleTime"],
				"DeviationDistance": 0.0,
			},
		}

	case "AssociateTrackerConsumer":
		s.ensureTrackerConsumerSetLocked(trackerName)[consumerArn] = struct{}{}
		return map[string]any{}
	case "DisassociateTrackerConsumer":
		delete(s.ensureTrackerConsumerSetLocked(trackerName), consumerArn)
		return map[string]any{}
	case "ListTrackerConsumers":
		consumers := make([]string, 0, len(s.ensureTrackerConsumerSetLocked(trackerName)))
		for arn := range s.trackerConsumers[trackerName] {
			consumers = append(consumers, arn)
		}
		sort.Strings(consumers)
		return map[string]any{"ConsumerArns": consumers, "NextToken": ""}

	case "CalculateRoute":
		return map[string]any{
			"Summary": map[string]any{"RouteBBox": []any{-122.35, 47.60, -122.30, 47.65}, "DataSource": "Here", "Distance": 2400.0, "DurationSeconds": 420.0},
			"Legs": []any{
				map[string]any{"Distance": 2400.0, "DurationSeconds": 420.0, "Geometry": map[string]any{"LineString": []any{[]any{-122.35, 47.60}, []any{-122.30, 47.65}}}, "Steps": []any{}},
			},
		}
	case "CalculateRouteMatrix":
		return map[string]any{
			"Summary":     map[string]any{"DataSource": "Here", "RouteCount": 1, "ErrorCount": 0},
			"RouteMatrix": []any{[]any{map[string]any{"Distance": 1000.0, "DurationSeconds": 180.0}}},
		}

	case "GetMapStyleDescriptor":
		return map[string]any{"version": 8, "name": mapName, "sources": map[string]any{}, "layers": []any{}}
	case "GetMapTile", "GetMapGlyphs", "GetMapSprites":
		return map[string]any{"Data": "c3RhY2t5YXJk"}

	case "GetPlace":
		return map[string]any{"Place": map[string]any{"Label": "Stackyard Place", "Geometry": map[string]any{"Point": []any{-122.341, 47.609}}, "Country": "USA", "Region": "Washington", "Municipality": "Seattle", "PostalCode": "98101", "AddressNumber": "1", "Street": "Main"}, "PlaceId": placeID}
	case "SearchPlaceIndexForPosition":
		return map[string]any{"Summary": map[string]any{"DataSource": "Here", "Position": []any{-122.341, 47.609}}, "Results": []any{map[string]any{"Place": map[string]any{"Label": "Seattle, WA, USA"}, "Distance": 0.0}}}
	case "SearchPlaceIndexForText":
		return map[string]any{"Summary": map[string]any{"DataSource": "Here", "Text": locationPayloadString(payload, []string{"Text"}, "stackyard")}, "Results": []any{map[string]any{"Place": map[string]any{"Label": "Stackyard HQ, Seattle, WA"}, "PlaceId": placeID}}}
	case "SearchPlaceIndexForSuggestions":
		return map[string]any{"Summary": map[string]any{"DataSource": "Here", "Text": locationPayloadString(payload, []string{"Text"}, "stack")}, "Results": []any{map[string]any{"Text": "Stackyard HQ"}}}

	case "ListTagsForResource":
		return map[string]any{"Tags": locationCloneStringMap(s.ensureTagsLocked(resourceARN))}
	case "TagResource":
		s.mergeTagsLocked(locationMapAny(payload, "Tags", "tags"), resourceARN)
		return map[string]any{}
	case "UntagResource":
		for _, keyName := range locationTagKeys(payload, query) {
			delete(s.ensureTagsLocked(resourceARN), keyName)
		}
		return map[string]any{}

	default:
		return map[string]any{}
	}
}

func (s *locationStore) applyUpdatesLocked(
	action string,
	payload map[string]any,
	m map[string]any,
	idx map[string]any,
	calc map[string]any,
	tracker map[string]any,
	collection map[string]any,
	key map[string]any,
	geofence map[string]any,
	now time.Time,
) {
	_ = action
	if desc := locationPayloadString(payload, []string{"Description"}, ""); desc != "" {
		m["Description"] = desc
		idx["Description"] = desc
		calc["Description"] = desc
		tracker["Description"] = desc
		collection["Description"] = desc
		key["Description"] = desc
		geofence["Description"] = desc
	}
	if filtering := locationPayloadString(payload, []string{"PositionFiltering"}, ""); filtering != "" {
		tracker["PositionFiltering"] = filtering
	}
	if ds := locationPayloadString(payload, []string{"DataSource"}, ""); ds != "" {
		m["DataSource"] = ds
		idx["DataSource"] = ds
		calc["DataSource"] = ds
	}
	if cfg, ok := locationPayloadAny(payload, []string{"Configuration"}); ok {
		m["Configuration"] = cfg
		idx["Configuration"] = cfg
		calc["Configuration"] = cfg
	}
	if restrictions, ok := locationPayloadAny(payload, []string{"Restrictions"}); ok {
		key["Restrictions"] = restrictions
	}
	if expires := locationPayloadString(payload, []string{"ExpireTime"}, ""); expires != "" {
		key["ExpireTime"] = expires
	}
	m["UpdateTime"] = locationTime(now)
	idx["UpdateTime"] = locationTime(now)
	calc["UpdateTime"] = locationTime(now)
	tracker["UpdateTime"] = locationTime(now)
	collection["UpdateTime"] = locationTime(now)
	key["UpdateTime"] = locationTime(now)
	geofence["UpdateTime"] = locationTime(now)
}

func (s *locationStore) ensureMapLocked(name string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-map"
	}
	if item := s.maps[name]; item != nil {
		return item
	}
	item := map[string]any{
		"MapName":     name,
		"MapArn":      locationResourceARN("map", name),
		"DataSource":  "Esri",
		"Description": "",
		"PricingPlan": "RequestBasedUsage",
		"CreateTime":  locationTime(now),
		"UpdateTime":  locationTime(now),
	}
	s.maps[name] = item
	s.ensureTagsLocked(locationAnyString(item, "MapArn", ""))
	return item
}

func (s *locationStore) ensurePlaceIndexLocked(name string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-place-index"
	}
	if item := s.placeIndexes[name]; item != nil {
		return item
	}
	item := map[string]any{
		"IndexName":   name,
		"IndexArn":    locationResourceARN("place-index", name),
		"DataSource":  "Here",
		"Description": "",
		"PricingPlan": "RequestBasedUsage",
		"CreateTime":  locationTime(now),
		"UpdateTime":  locationTime(now),
	}
	s.placeIndexes[name] = item
	s.ensureTagsLocked(locationAnyString(item, "IndexArn", ""))
	return item
}

func (s *locationStore) ensureRouteCalculatorLocked(name string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-route-calculator"
	}
	if item := s.routeCalculators[name]; item != nil {
		return item
	}
	item := map[string]any{
		"CalculatorName": name,
		"CalculatorArn":  locationResourceARN("route-calculator", name),
		"DataSource":     "Here",
		"Description":    "",
		"PricingPlan":    "RequestBasedUsage",
		"CreateTime":     locationTime(now),
		"UpdateTime":     locationTime(now),
	}
	s.routeCalculators[name] = item
	s.ensureTagsLocked(locationAnyString(item, "CalculatorArn", ""))
	return item
}

func (s *locationStore) ensureTrackerLocked(name string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-tracker"
	}
	if item := s.trackers[name]; item != nil {
		return item
	}
	item := map[string]any{
		"TrackerName":       name,
		"TrackerArn":        locationResourceARN("tracker", name),
		"Description":       "",
		"PricingPlan":       "RequestBasedUsage",
		"PositionFiltering": "TimeBased",
		"CreateTime":        locationTime(now),
		"UpdateTime":        locationTime(now),
	}
	s.trackers[name] = item
	s.ensureTagsLocked(locationAnyString(item, "TrackerArn", ""))
	return item
}

func (s *locationStore) ensureGeofenceCollectionLocked(name string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-geofence-collection"
	}
	if item := s.geofenceCollections[name]; item != nil {
		return item
	}
	item := map[string]any{
		"CollectionName": name,
		"CollectionArn":  locationResourceARN("geofence-collection", name),
		"Description":    "",
		"PricingPlan":    "RequestBasedUsage",
		"CreateTime":     locationTime(now),
		"UpdateTime":     locationTime(now),
	}
	s.geofenceCollections[name] = item
	if s.geofences[name] == nil {
		s.geofences[name] = map[string]map[string]any{}
	}
	s.ensureTagsLocked(locationAnyString(item, "CollectionArn", ""))
	return item
}

func (s *locationStore) ensureKeyLocked(name string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-api-key"
	}
	if item := s.keys[name]; item != nil {
		return item
	}
	item := map[string]any{
		"KeyName":      name,
		"KeyArn":       locationResourceARN("key", name),
		"Description":  "",
		"PricingPlan":  "RequestBasedUsage",
		"Key":          "stackyard-key-token",
		"Restrictions": map[string]any{},
		"CreateTime":   locationTime(now),
		"UpdateTime":   locationTime(now),
	}
	s.keys[name] = item
	s.ensureTagsLocked(locationAnyString(item, "KeyArn", ""))
	return item
}

func (s *locationStore) ensureGeofenceLocked(collectionName, geofenceID string, now time.Time) map[string]any {
	collectionName = strings.TrimSpace(collectionName)
	if collectionName == "" {
		collectionName = "stackyard-geofence-collection"
	}
	geofenceID = strings.TrimSpace(geofenceID)
	if geofenceID == "" {
		geofenceID = "stackyard-geofence"
	}
	s.ensureGeofenceCollectionLocked(collectionName, now)
	if s.geofences[collectionName] == nil {
		s.geofences[collectionName] = map[string]map[string]any{}
	}
	if item := s.geofences[collectionName][geofenceID]; item != nil {
		return item
	}
	item := map[string]any{
		"GeofenceId":  geofenceID,
		"Geometry":    map[string]any{"Polygon": []any{[]any{-122.35, 47.60}, []any{-122.30, 47.60}, []any{-122.30, 47.65}, []any{-122.35, 47.65}}},
		"CreateTime":  locationTime(now),
		"UpdateTime":  locationTime(now),
		"Status":      "ACTIVE",
		"Description": "",
	}
	s.geofences[collectionName][geofenceID] = item
	return item
}

func (s *locationStore) ensureDevicePositionLocked(trackerName, deviceID string, now time.Time) map[string]any {
	trackerName = strings.TrimSpace(trackerName)
	if trackerName == "" {
		trackerName = "stackyard-tracker"
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		deviceID = "device-000001"
	}
	s.ensureTrackerPositionMapLocked(trackerName)
	if entry := s.trackerDevicePositions[trackerName][deviceID]; entry != nil {
		return entry
	}
	s.upsertDevicePositionLocked(trackerName, deviceID, []any{-122.341, 47.609}, now)
	return s.trackerDevicePositions[trackerName][deviceID]
}

func (s *locationStore) ensureTrackerPositionMapLocked(trackerName string) {
	if s.trackerDevicePositions[trackerName] == nil {
		s.trackerDevicePositions[trackerName] = map[string]map[string]any{}
	}
}

func (s *locationStore) ensureTrackerConsumerSetLocked(trackerName string) map[string]struct{} {
	if s.trackerConsumers[trackerName] == nil {
		s.trackerConsumers[trackerName] = map[string]struct{}{}
	}
	return s.trackerConsumers[trackerName]
}

func (s *locationStore) upsertDevicePositionLocked(trackerName, deviceID string, position []any, now time.Time) {
	s.ensureTrackerPositionMapLocked(trackerName)
	if len(position) == 0 {
		position = []any{-122.341, 47.609}
	}
	s.trackerDevicePositions[trackerName][deviceID] = map[string]any{
		"DeviceId":       deviceID,
		"Position":       position,
		"SampleTime":     locationTime(now),
		"ReceivedTime":   locationTime(now),
		"TrackerName":    trackerName,
		"PositionSource": "Sampled",
	}
}

func (s *locationStore) listResourceSummariesLocked(items map[string]map[string]any) []any {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, locationCloneMap(items[key]))
	}
	return out
}

func (s *locationStore) sortedGeofenceIDsLocked(collectionName string) []string {
	keys := make([]string, 0, len(s.geofences[collectionName]))
	for id := range s.geofences[collectionName] {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys
}

func (s *locationStore) sortedDeviceIDsLocked(trackerName string) []string {
	keys := make([]string, 0, len(s.trackerDevicePositions[trackerName]))
	for id := range s.trackerDevicePositions[trackerName] {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys
}

func (s *locationStore) deleteMapLocked(name string) {
	if item := s.maps[name]; item != nil {
		delete(s.tags, locationAnyString(item, "MapArn", ""))
	}
	delete(s.maps, name)
}

func (s *locationStore) deletePlaceIndexLocked(name string) {
	if item := s.placeIndexes[name]; item != nil {
		delete(s.tags, locationAnyString(item, "IndexArn", ""))
	}
	delete(s.placeIndexes, name)
}

func (s *locationStore) deleteRouteCalculatorLocked(name string) {
	if item := s.routeCalculators[name]; item != nil {
		delete(s.tags, locationAnyString(item, "CalculatorArn", ""))
	}
	delete(s.routeCalculators, name)
}

func (s *locationStore) deleteTrackerLocked(name string) {
	if item := s.trackers[name]; item != nil {
		delete(s.tags, locationAnyString(item, "TrackerArn", ""))
	}
	delete(s.trackers, name)
	delete(s.trackerDevicePositions, name)
	delete(s.trackerConsumers, name)
}

func (s *locationStore) deleteGeofenceCollectionLocked(name string) {
	if item := s.geofenceCollections[name]; item != nil {
		delete(s.tags, locationAnyString(item, "CollectionArn", ""))
	}
	delete(s.geofenceCollections, name)
	delete(s.geofences, name)
}

func (s *locationStore) deleteKeyLocked(name string) {
	if item := s.keys[name]; item != nil {
		delete(s.tags, locationAnyString(item, "KeyArn", ""))
	}
	delete(s.keys, name)
}

func (s *locationStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = locationResourceARN("map", "stackyard-map")
	}
	if s.tags[resourceARN] == nil {
		s.tags[resourceARN] = map[string]string{"stackyard": "true", "service": "location"}
	}
	return s.tags[resourceARN]
}

func (s *locationStore) mergeTagsLocked(tags map[string]any, resourceARN string) {
	if len(tags) == 0 {
		return
	}
	target := s.ensureTagsLocked(resourceARN)
	for key, value := range tags {
		cleanKey := strings.TrimSpace(key)
		if cleanKey == "" {
			continue
		}
		target[cleanKey] = locationAnyString(value, "", "")
	}
}

func (s *locationStore) nameFromPathOrPayload(pathParams map[string]string, payload map[string]any, keys []string, def string) string {
	for _, key := range keys {
		if raw, ok := pathParams[key]; ok {
			if trimmed := strings.TrimSpace(raw); trimmed != "" {
				return trimmed
			}
		}
	}
	return locationPayloadString(payload, keys, def)
}

func locationPayloadString(payload map[string]any, keys []string, def string) string {
	for _, key := range keys {
		if value, ok := locationPayloadAny(payload, []string{key}); ok {
			trimmed := strings.TrimSpace(locationAnyString(value, "", ""))
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return def
}

func locationPayloadAny(payload map[string]any, keys []string) (any, bool) {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return value, true
		}
		for k, value := range payload {
			if strings.EqualFold(strings.TrimSpace(k), key) {
				return value, true
			}
		}
	}
	return nil, false
}

func locationPayloadAnyOr(payload map[string]any, key string, def any) any {
	if value, ok := locationPayloadAny(payload, []string{key}); ok {
		return value
	}
	return def
}

func locationPayloadSlice(payload map[string]any, key string) []any {
	if raw, ok := locationPayloadAny(payload, []string{key}); ok {
		if out, ok := raw.([]any); ok {
			return out
		}
	}
	return []any{}
}

func locationPayloadStringSlice(payload map[string]any, key string) []string {
	if raw, ok := locationPayloadAny(payload, []string{key}); ok {
		if arr, ok := raw.([]any); ok {
			out := make([]string, 0, len(arr))
			for _, item := range arr {
				value := strings.TrimSpace(locationAnyString(item, "", ""))
				if value != "" {
					out = append(out, value)
				}
			}
			return out
		}
	}
	return nil
}

func locationTagKeys(payload map[string]any, query url.Values) []string {
	set := map[string]struct{}{}
	for _, key := range query["tagKeys"] {
		trimmed := strings.TrimSpace(key)
		if trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	for _, key := range query["TagKeys"] {
		trimmed := strings.TrimSpace(key)
		if trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	for _, payloadKey := range []string{"TagKeys", "tagKeys"} {
		if raw, ok := locationPayloadAny(payload, []string{payloadKey}); ok {
			if arr, ok := raw.([]any); ok {
				for _, item := range arr {
					trimmed := strings.TrimSpace(locationAnyString(item, "", ""))
					if trimmed != "" {
						set[trimmed] = struct{}{}
					}
				}
			}
		}
	}
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func locationMapAny(payload map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if raw, ok := locationPayloadAny(payload, []string{key}); ok {
			if out, ok := raw.(map[string]any); ok {
				return out
			}
		}
	}
	return map[string]any{}
}

func locationAnyString(value any, key, def string) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case map[string]any:
		if key != "" {
			if inner, ok := v[key]; ok {
				return locationAnyString(inner, "", def)
			}
		}
		return def
	default:
		if value == nil {
			return def
		}
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func locationCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func locationCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func locationTime(now time.Time) string {
	return now.UTC().Format(time.RFC3339)
}

func locationResourceARN(resourceType, name string) string {
	resourceType = strings.TrimSpace(resourceType)
	if resourceType == "" {
		resourceType = "resource"
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard"
	}
	return fmt.Sprintf("arn:aws:geo:%s:%s:%s/%s", locationDefaultRegion, locationDefaultAccountID, resourceType, name)
}
