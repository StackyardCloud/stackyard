package server

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	appFabricDefaultRegion    = "us-east-1"
	appFabricDefaultAccountID = "123456789012"
)

type appFabricStore struct {
	mu sync.Mutex

	nextBundleID          int64
	nextAuthID            int64
	nextIngestionID       int64
	nextDestinationID     int64
	nextUserAccessTaskID  int64
	appBundles            map[string]map[string]any
	appAuthorizations     map[string]map[string]map[string]any
	ingestions            map[string]map[string]map[string]any
	ingestionDestinations map[string]map[string]map[string]map[string]any
	userAccessTasks       map[string]map[string]any
	tags                  map[string]map[string]string
}

func newAppFabricStore() *appFabricStore {
	s := &appFabricStore{
		nextBundleID:          2,
		nextAuthID:            2,
		nextIngestionID:       2,
		nextDestinationID:     2,
		nextUserAccessTaskID:  2,
		appBundles:            map[string]map[string]any{},
		appAuthorizations:     map[string]map[string]map[string]any{},
		ingestions:            map[string]map[string]map[string]any{},
		ingestionDestinations: map[string]map[string]map[string]map[string]any{},
		userAccessTasks:       map[string]map[string]any{},
		tags:                  map[string]map[string]string{},
	}
	s.ensureSeedLocked()
	return s
}

func (s *appFabricStore) Handle(
	action string,
	payload map[string]any,
	pathParams map[string]string,
	query url.Values,
) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSeedLocked()
	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateAppBundle":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		if bundleID == "" {
			bundleID = fmt.Sprintf("ab-%06d", s.nextBundleID)
			s.nextBundleID++
		}
		bundle := s.ensureAppBundleLocked(bundleID, now)
		bundle["updatedAt"] = now
		return map[string]any{"appBundle": appFabricCloneMap(bundle)}

	case "GetAppBundle":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		bundle := s.ensureAppBundleLocked(bundleID, now)
		return map[string]any{"appBundle": appFabricCloneMap(bundle)}

	case "ListAppBundles":
		items := make([]any, 0, len(s.appBundles))
		for _, bundle := range appFabricSortedMapValues(s.appBundles, "appBundleIdentifier") {
			items = append(items, map[string]any{
				"arn":                 bundle["arn"],
				"appBundleIdentifier": bundle["appBundleIdentifier"],
				"status":              bundle["status"],
			})
		}
		return map[string]any{"appBundles": items, "nextToken": ""}

	case "DeleteAppBundle":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		if bundleID == "" {
			bundleID = s.defaultAppBundleIdentifierLocked()
		}
		bundle := s.ensureAppBundleLocked(bundleID, now)
		delete(s.appBundles, bundleID)
		delete(s.appAuthorizations, bundleID)
		delete(s.ingestions, bundleID)
		delete(s.ingestionDestinations, bundleID)
		delete(s.tags, appFabricString(bundle, "arn", ""))
		return map[string]any{}

	case "CreateAppAuthorization":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		bundle := s.ensureAppBundleLocked(bundleID, now)
		authID := appFabricLookupString(pathParams, payload, query, "appAuthorizationIdentifier")
		if authID == "" {
			authID = fmt.Sprintf("auth-%06d", s.nextAuthID)
			s.nextAuthID++
		}
		auth := s.ensureAppAuthorizationLocked(appFabricString(bundle, "appBundleIdentifier", ""), authID, now)
		auth["updatedAt"] = now
		return map[string]any{"appAuthorization": appFabricCloneMap(auth)}

	case "ConnectAppAuthorization":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		authID := appFabricLookupString(pathParams, payload, query, "appAuthorizationIdentifier")
		auth := s.ensureAppAuthorizationLocked(bundleID, authID, now)
		auth["status"] = "CONNECTED"
		auth["updatedAt"] = now
		return map[string]any{"appAuthorization": appFabricCloneMap(auth)}

	case "UpdateAppAuthorization":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		authID := appFabricLookupString(pathParams, payload, query, "appAuthorizationIdentifier")
		auth := s.ensureAppAuthorizationLocked(bundleID, authID, now)
		if status := appFabricLookupString(pathParams, payload, query, "status"); status != "" {
			auth["status"] = status
		}
		auth["updatedAt"] = now
		return map[string]any{"appAuthorization": appFabricCloneMap(auth)}

	case "GetAppAuthorization":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		authID := appFabricLookupString(pathParams, payload, query, "appAuthorizationIdentifier")
		auth := s.ensureAppAuthorizationLocked(bundleID, authID, now)
		return map[string]any{"appAuthorization": appFabricCloneMap(auth)}

	case "ListAppAuthorizations":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		auths := s.ensureAuthBucketLocked(bundleID, now)
		items := make([]any, 0, len(auths))
		for _, auth := range appFabricSortedMapValues(auths, "appAuthorizationIdentifier") {
			items = append(items, map[string]any{
				"appAuthorizationIdentifier": auth["appAuthorizationIdentifier"],
				"appBundleIdentifier":        auth["appBundleIdentifier"],
				"app":                        auth["app"],
				"status":                     auth["status"],
			})
		}
		return map[string]any{"appAuthorizations": items, "nextToken": ""}

	case "DeleteAppAuthorization":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		authID := appFabricLookupString(pathParams, payload, query, "appAuthorizationIdentifier")
		auths := s.ensureAuthBucketLocked(bundleID, now)
		delete(auths, appFabricDefaultIfEmpty(authID, "auth-000001"))
		return map[string]any{}

	case "CreateIngestion":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		_ = s.ensureAppBundleLocked(bundleID, now)
		ingestionID := appFabricLookupString(pathParams, payload, query, "ingestionIdentifier")
		if ingestionID == "" {
			ingestionID = fmt.Sprintf("ing-%06d", s.nextIngestionID)
			s.nextIngestionID++
		}
		ingestion := s.ensureIngestionLocked(bundleID, ingestionID, now)
		ingestion["updatedAt"] = now
		return map[string]any{"ingestion": appFabricCloneMap(ingestion)}

	case "GetIngestion":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		ingestionID := appFabricLookupString(pathParams, payload, query, "ingestionIdentifier")
		ingestion := s.ensureIngestionLocked(bundleID, ingestionID, now)
		return map[string]any{"ingestion": appFabricCloneMap(ingestion)}

	case "ListIngestions":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		ings := s.ensureIngestionBucketLocked(bundleID, now)
		items := make([]any, 0, len(ings))
		for _, ingestion := range appFabricSortedMapValues(ings, "ingestionIdentifier") {
			items = append(items, map[string]any{
				"arn":                 ingestion["arn"],
				"appBundleIdentifier": ingestion["appBundleIdentifier"],
				"ingestionIdentifier": ingestion["ingestionIdentifier"],
				"state":               ingestion["state"],
			})
		}
		return map[string]any{"ingestions": items, "nextToken": ""}

	case "DeleteIngestion":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		ingestionID := appFabricLookupString(pathParams, payload, query, "ingestionIdentifier")
		ings := s.ensureIngestionBucketLocked(bundleID, now)
		delete(ings, appFabricDefaultIfEmpty(ingestionID, "ing-000001"))
		if s.ingestionDestinations[bundleID] != nil {
			delete(s.ingestionDestinations[bundleID], appFabricDefaultIfEmpty(ingestionID, "ing-000001"))
		}
		return map[string]any{}

	case "StartIngestion":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		ingestionID := appFabricLookupString(pathParams, payload, query, "ingestionIdentifier")
		ingestion := s.ensureIngestionLocked(bundleID, ingestionID, now)
		ingestion["state"] = "ACTIVE"
		ingestion["updatedAt"] = now
		return map[string]any{"ingestion": appFabricCloneMap(ingestion)}

	case "StopIngestion":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		ingestionID := appFabricLookupString(pathParams, payload, query, "ingestionIdentifier")
		ingestion := s.ensureIngestionLocked(bundleID, ingestionID, now)
		ingestion["state"] = "STOPPED"
		ingestion["updatedAt"] = now
		return map[string]any{"ingestion": appFabricCloneMap(ingestion)}

	case "CreateIngestionDestination":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		ingestionID := appFabricLookupString(pathParams, payload, query, "ingestionIdentifier")
		_ = s.ensureIngestionLocked(bundleID, ingestionID, now)
		destinationID := appFabricLookupString(pathParams, payload, query, "ingestionDestinationIdentifier")
		if destinationID == "" {
			destinationID = fmt.Sprintf("dest-%06d", s.nextDestinationID)
			s.nextDestinationID++
		}
		dest := s.ensureIngestionDestinationLocked(bundleID, ingestionID, destinationID, now)
		dest["updatedAt"] = now
		return map[string]any{"ingestionDestination": appFabricCloneMap(dest)}

	case "GetIngestionDestination":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		ingestionID := appFabricLookupString(pathParams, payload, query, "ingestionIdentifier")
		destinationID := appFabricLookupString(pathParams, payload, query, "ingestionDestinationIdentifier")
		dest := s.ensureIngestionDestinationLocked(bundleID, ingestionID, destinationID, now)
		return map[string]any{"ingestionDestination": appFabricCloneMap(dest)}

	case "ListIngestionDestinations":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		ingestionID := appFabricLookupString(pathParams, payload, query, "ingestionIdentifier")
		dests := s.ensureIngestionDestinationBucketLocked(bundleID, ingestionID, now)
		items := make([]any, 0, len(dests))
		for _, dest := range appFabricSortedMapValues(dests, "ingestionDestinationIdentifier") {
			items = append(items, map[string]any{
				"arn":                            dest["arn"],
				"appBundleIdentifier":            dest["appBundleIdentifier"],
				"ingestionIdentifier":            dest["ingestionIdentifier"],
				"ingestionDestinationIdentifier": dest["ingestionDestinationIdentifier"],
				"state":                          dest["state"],
			})
		}
		return map[string]any{"ingestionDestinations": items, "nextToken": ""}

	case "UpdateIngestionDestination":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		ingestionID := appFabricLookupString(pathParams, payload, query, "ingestionIdentifier")
		destinationID := appFabricLookupString(pathParams, payload, query, "ingestionDestinationIdentifier")
		dest := s.ensureIngestionDestinationLocked(bundleID, ingestionID, destinationID, now)
		if state := appFabricLookupString(pathParams, payload, query, "state"); state != "" {
			dest["state"] = state
		}
		dest["updatedAt"] = now
		return map[string]any{"ingestionDestination": appFabricCloneMap(dest)}

	case "DeleteIngestionDestination":
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		ingestionID := appFabricLookupString(pathParams, payload, query, "ingestionIdentifier")
		destinationID := appFabricLookupString(pathParams, payload, query, "ingestionDestinationIdentifier")
		dests := s.ensureIngestionDestinationBucketLocked(bundleID, ingestionID, now)
		delete(dests, appFabricDefaultIfEmpty(destinationID, "dest-000001"))
		return map[string]any{}

	case "StartUserAccessTasks":
		taskID := appFabricLookupString(pathParams, payload, query, "userAccessTaskIdentifier", "taskId", "userAccessTaskId")
		if taskID == "" {
			taskID = fmt.Sprintf("uat-%06d", s.nextUserAccessTaskID)
			s.nextUserAccessTaskID++
		}
		bundleID := appFabricLookupString(pathParams, payload, query, "appBundleIdentifier")
		if bundleID == "" {
			bundleID = s.defaultAppBundleIdentifierLocked()
		}
		task := map[string]any{
			"taskId":              taskID,
			"appBundleIdentifier": bundleID,
			"status":              "COMPLETED",
			"startedAt":           now,
			"updatedAt":           now,
		}
		s.userAccessTasks[taskID] = task
		return map[string]any{"userAccessTaskId": taskID}

	case "BatchGetUserAccessTasks":
		requestedIDs := appFabricStringSlice(payload, "userAccessTaskIds", "taskIds")
		if len(requestedIDs) == 0 {
			for id := range s.userAccessTasks {
				requestedIDs = append(requestedIDs, id)
			}
			sort.Strings(requestedIDs)
		}
		results := make([]any, 0, len(requestedIDs))
		for _, taskID := range requestedIDs {
			task := s.ensureUserAccessTaskLocked(taskID, now)
			results = append(results, map[string]any{
				"taskId":              task["taskId"],
				"appBundleIdentifier": task["appBundleIdentifier"],
				"status":              task["status"],
			})
		}
		return map[string]any{"userAccessResultsList": results}

	case "TagResource":
		resourceARN := appFabricLookupString(pathParams, payload, query, "resourceArn")
		if resourceARN == "" {
			resourceARN = appFabricAppBundleARN(s.defaultAppBundleIdentifierLocked())
		}
		s.upsertTagsLocked(resourceARN, appFabricExtractTags(payload))
		return map[string]any{}

	case "UntagResource":
		resourceARN := appFabricLookupString(pathParams, payload, query, "resourceArn")
		if resourceARN == "" {
			resourceARN = appFabricAppBundleARN(s.defaultAppBundleIdentifierLocked())
		}
		tagKeys := appFabricTagKeys(payload, query)
		s.removeTagsLocked(resourceARN, tagKeys)
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := appFabricLookupString(pathParams, payload, query, "resourceArn")
		if resourceARN == "" {
			resourceARN = appFabricAppBundleARN(s.defaultAppBundleIdentifierLocked())
		}
		return map[string]any{"tags": appFabricCloneStringMap(s.tags[resourceARN])}
	}

	return map[string]any{}
}

func (s *appFabricStore) ensureSeedLocked() {
	if len(s.appBundles) > 0 {
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	bundleID := "ab-000001"
	authID := "auth-000001"
	ingestionID := "ing-000001"
	destinationID := "dest-000001"
	taskID := "uat-000001"

	bundle := s.ensureAppBundleLocked(bundleID, now)
	auth := s.ensureAppAuthorizationLocked(bundleID, authID, now)
	ingestion := s.ensureIngestionLocked(bundleID, ingestionID, now)
	dest := s.ensureIngestionDestinationLocked(bundleID, ingestionID, destinationID, now)
	task := s.ensureUserAccessTaskLocked(taskID, now)

	auth["status"] = "CONNECTED"
	ingestion["state"] = "STOPPED"
	dest["state"] = "ACTIVE"
	task["status"] = "COMPLETED"

	s.tags[appFabricString(bundle, "arn", "")] = map[string]string{"stackyard": "true"}
}

func (s *appFabricStore) ensureAppBundleLocked(bundleID, now string) map[string]any {
	if bundleID == "" {
		bundleID = s.defaultAppBundleIdentifierLocked()
	}
	if bundleID == "" {
		bundleID = "ab-000001"
	}
	if existing, ok := s.appBundles[bundleID]; ok {
		return existing
	}
	item := map[string]any{
		"arn":                 appFabricAppBundleARN(bundleID),
		"appBundleIdentifier": bundleID,
		"status":              "ACTIVE",
		"createdAt":           now,
		"updatedAt":           now,
	}
	s.appBundles[bundleID] = item
	return item
}

func (s *appFabricStore) ensureAuthBucketLocked(bundleID, now string) map[string]map[string]any {
	if bundleID == "" {
		bundleID = s.defaultAppBundleIdentifierLocked()
	}
	bundleID = appFabricDefaultIfEmpty(bundleID, "ab-000001")
	s.ensureAppBundleLocked(bundleID, now)
	bucket, ok := s.appAuthorizations[bundleID]
	if !ok {
		bucket = map[string]map[string]any{}
		s.appAuthorizations[bundleID] = bucket
	}
	return bucket
}

func (s *appFabricStore) ensureAppAuthorizationLocked(bundleID, authID, now string) map[string]any {
	bundleID = appFabricDefaultIfEmpty(bundleID, s.defaultAppBundleIdentifierLocked())
	bundleID = appFabricDefaultIfEmpty(bundleID, "ab-000001")
	authID = appFabricDefaultIfEmpty(authID, "auth-000001")
	bucket := s.ensureAuthBucketLocked(bundleID, now)
	if existing, ok := bucket[authID]; ok {
		return existing
	}
	item := map[string]any{
		"arn":                        appFabricAppAuthorizationARN(bundleID, authID),
		"appBundleIdentifier":        bundleID,
		"appAuthorizationIdentifier": authID,
		"app":                        "okta",
		"status":                     "PENDING_CONNECT",
		"createdAt":                  now,
		"updatedAt":                  now,
	}
	bucket[authID] = item
	return item
}

func (s *appFabricStore) ensureIngestionBucketLocked(bundleID, now string) map[string]map[string]any {
	bundleID = appFabricDefaultIfEmpty(bundleID, s.defaultAppBundleIdentifierLocked())
	bundleID = appFabricDefaultIfEmpty(bundleID, "ab-000001")
	s.ensureAppBundleLocked(bundleID, now)
	bucket, ok := s.ingestions[bundleID]
	if !ok {
		bucket = map[string]map[string]any{}
		s.ingestions[bundleID] = bucket
	}
	return bucket
}

func (s *appFabricStore) ensureIngestionLocked(bundleID, ingestionID, now string) map[string]any {
	bundleID = appFabricDefaultIfEmpty(bundleID, s.defaultAppBundleIdentifierLocked())
	bundleID = appFabricDefaultIfEmpty(bundleID, "ab-000001")
	ingestionID = appFabricDefaultIfEmpty(ingestionID, "ing-000001")
	bucket := s.ensureIngestionBucketLocked(bundleID, now)
	if existing, ok := bucket[ingestionID]; ok {
		return existing
	}
	item := map[string]any{
		"arn":                 appFabricIngestionARN(bundleID, ingestionID),
		"appBundleIdentifier": bundleID,
		"ingestionIdentifier": ingestionID,
		"state":               "CREATED",
		"createdAt":           now,
		"updatedAt":           now,
	}
	bucket[ingestionID] = item
	return item
}

func (s *appFabricStore) ensureIngestionDestinationBucketLocked(bundleID, ingestionID, now string) map[string]map[string]any {
	bundleID = appFabricDefaultIfEmpty(bundleID, s.defaultAppBundleIdentifierLocked())
	bundleID = appFabricDefaultIfEmpty(bundleID, "ab-000001")
	ingestionID = appFabricDefaultIfEmpty(ingestionID, "ing-000001")
	_ = s.ensureIngestionLocked(bundleID, ingestionID, now)
	bundleBucket, ok := s.ingestionDestinations[bundleID]
	if !ok {
		bundleBucket = map[string]map[string]map[string]any{}
		s.ingestionDestinations[bundleID] = bundleBucket
	}
	destBucket, ok := bundleBucket[ingestionID]
	if !ok {
		destBucket = map[string]map[string]any{}
		bundleBucket[ingestionID] = destBucket
	}
	return destBucket
}

func (s *appFabricStore) ensureIngestionDestinationLocked(bundleID, ingestionID, destinationID, now string) map[string]any {
	bundleID = appFabricDefaultIfEmpty(bundleID, s.defaultAppBundleIdentifierLocked())
	bundleID = appFabricDefaultIfEmpty(bundleID, "ab-000001")
	ingestionID = appFabricDefaultIfEmpty(ingestionID, "ing-000001")
	destinationID = appFabricDefaultIfEmpty(destinationID, "dest-000001")
	destBucket := s.ensureIngestionDestinationBucketLocked(bundleID, ingestionID, now)
	if existing, ok := destBucket[destinationID]; ok {
		return existing
	}
	item := map[string]any{
		"arn":                            appFabricIngestionDestinationARN(bundleID, ingestionID, destinationID),
		"appBundleIdentifier":            bundleID,
		"ingestionIdentifier":            ingestionID,
		"ingestionDestinationIdentifier": destinationID,
		"state":                          "ACTIVE",
		"createdAt":                      now,
		"updatedAt":                      now,
	}
	destBucket[destinationID] = item
	return item
}

func (s *appFabricStore) ensureUserAccessTaskLocked(taskID, now string) map[string]any {
	taskID = appFabricDefaultIfEmpty(taskID, "uat-000001")
	if existing, ok := s.userAccessTasks[taskID]; ok {
		return existing
	}
	item := map[string]any{
		"taskId":              taskID,
		"appBundleIdentifier": s.defaultAppBundleIdentifierLocked(),
		"status":              "COMPLETED",
		"startedAt":           now,
		"updatedAt":           now,
	}
	s.userAccessTasks[taskID] = item
	return item
}

func (s *appFabricStore) upsertTagsLocked(resourceARN string, tags map[string]string) {
	if resourceARN == "" || len(tags) == 0 {
		return
	}
	existing := s.tags[resourceARN]
	if existing == nil {
		existing = map[string]string{}
		s.tags[resourceARN] = existing
	}
	for key, value := range tags {
		if strings.TrimSpace(key) == "" {
			continue
		}
		existing[key] = value
	}
}

func (s *appFabricStore) removeTagsLocked(resourceARN string, keys []string) {
	if resourceARN == "" {
		return
	}
	existing := s.tags[resourceARN]
	if len(existing) == 0 {
		return
	}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		delete(existing, key)
	}
}

func (s *appFabricStore) defaultAppBundleIdentifierLocked() string {
	if len(s.appBundles) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.appBundles))
	for key := range s.appBundles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func appFabricAppBundleARN(bundleID string) string {
	return fmt.Sprintf(
		"arn:aws:appfabric:%s:%s:appbundle/%s",
		appFabricDefaultRegion,
		appFabricDefaultAccountID,
		bundleID,
	)
}

func appFabricAppAuthorizationARN(bundleID, authID string) string {
	return fmt.Sprintf(
		"arn:aws:appfabric:%s:%s:appbundle/%s/appauthorization/%s",
		appFabricDefaultRegion,
		appFabricDefaultAccountID,
		bundleID,
		authID,
	)
}

func appFabricIngestionARN(bundleID, ingestionID string) string {
	return fmt.Sprintf(
		"arn:aws:appfabric:%s:%s:appbundle/%s/ingestion/%s",
		appFabricDefaultRegion,
		appFabricDefaultAccountID,
		bundleID,
		ingestionID,
	)
}

func appFabricIngestionDestinationARN(bundleID, ingestionID, destinationID string) string {
	return fmt.Sprintf(
		"arn:aws:appfabric:%s:%s:appbundle/%s/ingestion/%s/destination/%s",
		appFabricDefaultRegion,
		appFabricDefaultAccountID,
		bundleID,
		ingestionID,
		destinationID,
	)
}

func appFabricLookupString(pathParams map[string]string, payload map[string]any, query url.Values, key string, aliases ...string) string {
	keys := make([]string, 0, 1+len(aliases))
	keys = append(keys, key)
	keys = append(keys, aliases...)
	for _, name := range keys {
		for k, v := range pathParams {
			if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(name)) {
				return strings.TrimSpace(v)
			}
		}
		if raw, ok := appFabricMapLookupInsensitive(payload, name); ok {
			if val := appFabricAnyToString(raw); val != "" {
				return val
			}
		}
		for qk, values := range query {
			if !strings.EqualFold(strings.TrimSpace(qk), strings.TrimSpace(name)) {
				continue
			}
			for i := len(values) - 1; i >= 0; i-- {
				if strings.TrimSpace(values[i]) != "" {
					return strings.TrimSpace(values[i])
				}
			}
		}
	}
	return ""
}

func appFabricExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	raw, ok := appFabricMapLookupInsensitive(payload, "tags")
	if !ok {
		return out
	}
	switch tv := raw.(type) {
	case map[string]any:
		for key, value := range tv {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = appFabricAnyToString(value)
		}
	case map[string]string:
		for key, value := range tv {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(value)
		}
	case []any:
		for _, item := range tv {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := appFabricAnyToString(entry["key"])
			if key == "" {
				key = appFabricAnyToString(entry["Key"])
			}
			if key == "" {
				continue
			}
			value := appFabricAnyToString(entry["value"])
			if value == "" {
				value = appFabricAnyToString(entry["Value"])
			}
			out[key] = value
		}
	}
	return out
}

func appFabricTagKeys(payload map[string]any, query url.Values) []string {
	keys := []string{}
	for qk, values := range query {
		if !strings.EqualFold(strings.TrimSpace(qk), "tagKeys") {
			continue
		}
		for _, value := range values {
			for _, token := range strings.Split(value, ",") {
				token = strings.TrimSpace(token)
				if token != "" {
					keys = append(keys, token)
				}
			}
		}
	}
	raw, ok := appFabricMapLookupInsensitive(payload, "tagKeys")
	if !ok {
		return appFabricUniqueStrings(keys)
	}
	switch tv := raw.(type) {
	case []any:
		for _, item := range tv {
			if value := appFabricAnyToString(item); value != "" {
				keys = append(keys, value)
			}
		}
	case []string:
		for _, value := range tv {
			value = strings.TrimSpace(value)
			if value != "" {
				keys = append(keys, value)
			}
		}
	default:
		for _, token := range strings.Split(appFabricAnyToString(tv), ",") {
			token = strings.TrimSpace(token)
			if token != "" {
				keys = append(keys, token)
			}
		}
	}
	return appFabricUniqueStrings(keys)
}

func appFabricStringSlice(payload map[string]any, keys ...string) []string {
	for _, key := range keys {
		raw, ok := appFabricMapLookupInsensitive(payload, key)
		if !ok {
			continue
		}
		switch tv := raw.(type) {
		case []any:
			out := make([]string, 0, len(tv))
			for _, item := range tv {
				if v := appFabricAnyToString(item); v != "" {
					out = append(out, v)
				}
			}
			if len(out) > 0 {
				return out
			}
		case []string:
			out := make([]string, 0, len(tv))
			for _, item := range tv {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
			if len(out) > 0 {
				return out
			}
		default:
			if v := appFabricAnyToString(tv); v != "" {
				return []string{v}
			}
		}
	}
	return nil
}

func appFabricMapLookupInsensitive(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	if raw, ok := payload[key]; ok {
		return raw, true
	}
	for k, value := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return value, true
		}
	}
	return nil, false
}

func appFabricAnyToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case float64:
		return strings.TrimSpace(strconv.FormatFloat(v, 'f', -1, 64))
	case float32:
		return strings.TrimSpace(strconv.FormatFloat(float64(v), 'f', -1, 32))
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func appFabricString(m map[string]any, key, def string) string {
	if m == nil {
		return def
	}
	if raw, ok := m[key]; ok {
		if value := appFabricAnyToString(raw); value != "" {
			return value
		}
	}
	return def
}

func appFabricCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func appFabricCloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func appFabricSortedMapValues[T map[string]any](in map[string]T, sortKey string) []map[string]any {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		item := appFabricCloneMap(in[key])
		if _, exists := item[sortKey]; !exists {
			item[sortKey] = key
		}
		out = append(out, item)
	}
	return out
}

func appFabricDefaultIfEmpty(value, def string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
}

func appFabricUniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
