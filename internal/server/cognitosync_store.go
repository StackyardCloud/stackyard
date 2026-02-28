package server

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cognitoSyncPushSync struct {
	ApplicationArns []string
	RoleArn         string
}

type cognitoSyncCognitoStreams struct {
	StreamName      string
	RoleArn         string
	StreamingStatus string
}

type cognitoSyncBulkPublishDetails struct {
	IdentityPoolID      string
	BulkPublishStart    time.Time
	BulkPublishComplete time.Time
	BulkPublishStatus   string
	FailureMessage      string
}

type cognitoSyncDevice struct {
	DeviceID         string
	Platform         string
	Token            string
	CreationDate     time.Time
	LastModifiedDate time.Time
}

type cognitoSyncRecord struct {
	Key                    string
	Value                  string
	SyncCount              int64
	LastModifiedDate       time.Time
	LastModifiedBy         string
	DeviceLastModifiedDate time.Time
}

type cognitoSyncDataset struct {
	Name             string
	IdentityID       string
	CreationDate     time.Time
	LastModifiedDate time.Time
	LastModifiedBy   string
	SyncCount        int64
	Records          map[string]cognitoSyncRecord
	Subscriptions    map[string]time.Time
}

type cognitoSyncIdentity struct {
	ID               string
	CreationDate     time.Time
	LastModifiedDate time.Time
	Datasets         map[string]cognitoSyncDataset
	DeletedDatasets  map[string]int64
	Devices          map[string]cognitoSyncDevice
	DeviceByToken    map[string]string
	NextDeviceID     int64
}

type cognitoSyncIdentityPool struct {
	ID               string
	CreationDate     time.Time
	LastModifiedDate time.Time
	Events           map[string]string
	PushSync         *cognitoSyncPushSync
	CognitoStreams   *cognitoSyncCognitoStreams
	Identities       map[string]cognitoSyncIdentity
	BulkPublish      *cognitoSyncBulkPublishDetails
}

type cognitoSyncIdentityPoolUsage struct {
	IdentityPoolID   string
	SyncSessions     int
	DataStorage      int64
	LastModifiedDate time.Time
}

type cognitoSyncIdentityUsage struct {
	IdentityID       string
	IdentityPoolID   string
	DatasetCount     int
	DataStorage      int64
	LastModifiedDate time.Time
}

type cognitoSyncDatasetSummary struct {
	Name             string
	IdentityID       string
	CreationDate     time.Time
	LastModifiedDate time.Time
	LastModifiedBy   string
	DataStorage      int64
	NumRecords       int
}

type cognitoSyncRecordSummary struct {
	Key                    string
	Value                  string
	SyncCount              int64
	LastModifiedDate       time.Time
	LastModifiedBy         string
	DeviceLastModifiedDate time.Time
}

type cognitoSyncRecordPatchInput struct {
	Op                     string
	Key                    string
	Value                  string
	SyncCount              int64
	DeviceLastModifiedDate *time.Time
}

type cognitoSyncListRecordsOutput struct {
	Records                      []cognitoSyncRecordSummary
	Count                        int
	DatasetExists                bool
	DatasetDeletedAfterSyncCount bool
	MergedDatasetNames           []string
	DatasetSyncCount             int64
	SyncSessionToken             string
	LastModifiedBy               string
	NextToken                    string
}

type cognitoSyncStore struct {
	mu    sync.Mutex
	pools map[string]cognitoSyncIdentityPool
	now   func() time.Time
}

type cognitoSyncAPIError struct {
	Status  int
	Code    string
	Message string
}

func (e *cognitoSyncAPIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func newCognitoSyncStore() *cognitoSyncStore {
	return &cognitoSyncStore{
		pools: map[string]cognitoSyncIdentityPool{},
		now:   time.Now,
	}
}

func (s *cognitoSyncStore) DescribeIdentityPoolUsage(identityPoolID string) (cognitoSyncIdentityPoolUsage, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return cognitoSyncIdentityPoolUsage{}, validationCognitoSync("IdentityPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	return s.poolUsageLocked(pool), nil
}

func (s *cognitoSyncStore) DescribeIdentityUsage(identityPoolID, identityID string) (cognitoSyncIdentityUsage, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	identityID = strings.TrimSpace(identityID)
	if identityPoolID == "" {
		return cognitoSyncIdentityUsage{}, validationCognitoSync("IdentityPoolId is required")
	}
	if identityID == "" {
		return cognitoSyncIdentityUsage{}, validationCognitoSync("IdentityId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	_, identity := s.ensureIdentityLocked(pool, identityID)
	return s.identityUsageLocked(pool.ID, identity), nil
}

func (s *cognitoSyncStore) ListIdentityPoolUsage(maxResults int, nextToken string) ([]cognitoSyncIdentityPoolUsage, string, error) {
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 60 {
		return nil, "", validationCognitoSync("MaxResults must be less than or equal to 60")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.pools))
	for id := range s.pools {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	start, err := parseCognitoSyncNextToken(nextToken, len(ids))
	if err != nil {
		return nil, "", err
	}
	if start >= len(ids) {
		return []cognitoSyncIdentityPoolUsage{}, "", nil
	}

	end := start + maxResults
	if end > len(ids) {
		end = len(ids)
	}

	out := make([]cognitoSyncIdentityPoolUsage, 0, end-start)
	for _, id := range ids[start:end] {
		out = append(out, s.poolUsageLocked(s.pools[id]))
	}

	next := ""
	if end < len(ids) {
		next = strconv.Itoa(end)
	}
	return out, next, nil
}

func (s *cognitoSyncStore) ListDatasets(identityPoolID, identityID string, maxResults int, nextToken string) ([]cognitoSyncDatasetSummary, string, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	identityID = strings.TrimSpace(identityID)
	if identityPoolID == "" {
		return nil, "", validationCognitoSync("IdentityPoolId is required")
	}
	if identityID == "" {
		return nil, "", validationCognitoSync("IdentityId is required")
	}
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 100 {
		return nil, "", validationCognitoSync("MaxResults must be less than or equal to 100")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	_, identity := s.ensureIdentityLocked(pool, identityID)

	names := make([]string, 0, len(identity.Datasets))
	for name := range identity.Datasets {
		names = append(names, name)
	}
	sort.Strings(names)

	start, err := parseCognitoSyncNextToken(nextToken, len(names))
	if err != nil {
		return nil, "", err
	}
	if start >= len(names) {
		return []cognitoSyncDatasetSummary{}, "", nil
	}

	end := start + maxResults
	if end > len(names) {
		end = len(names)
	}

	out := make([]cognitoSyncDatasetSummary, 0, end-start)
	for _, name := range names[start:end] {
		out = append(out, s.datasetSummaryLocked(identity.Datasets[name]))
	}

	next := ""
	if end < len(names) {
		next = strconv.Itoa(end)
	}
	return out, next, nil
}

func (s *cognitoSyncStore) DescribeDataset(identityPoolID, identityID, datasetName string) (cognitoSyncDatasetSummary, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	identityID = strings.TrimSpace(identityID)
	datasetName = strings.TrimSpace(datasetName)
	if identityPoolID == "" {
		return cognitoSyncDatasetSummary{}, validationCognitoSync("IdentityPoolId is required")
	}
	if identityID == "" {
		return cognitoSyncDatasetSummary{}, validationCognitoSync("IdentityId is required")
	}
	if datasetName == "" {
		return cognitoSyncDatasetSummary{}, validationCognitoSync("DatasetName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	pool, identity := s.ensureIdentityLocked(pool, identityID)
	if deletedSyncCount, deleted := identity.DeletedDatasets[datasetName]; deleted {
		_ = deletedSyncCount
		return cognitoSyncDatasetSummary{}, notFoundCognitoSync("Dataset not found")
	}
	dataset, ok := identity.Datasets[datasetName]
	if !ok {
		pool, identity, dataset = s.ensureDatasetWritableLocked(pool, identity, datasetName, "stackyard")
		_ = pool
	}
	_ = identity
	return s.datasetSummaryLocked(dataset), nil
}

func (s *cognitoSyncStore) ListRecords(identityPoolID, identityID, datasetName string, lastSyncCount int64, maxResults int, nextToken string) (cognitoSyncListRecordsOutput, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	identityID = strings.TrimSpace(identityID)
	datasetName = strings.TrimSpace(datasetName)
	if identityPoolID == "" {
		return cognitoSyncListRecordsOutput{}, validationCognitoSync("IdentityPoolId is required")
	}
	if identityID == "" {
		return cognitoSyncListRecordsOutput{}, validationCognitoSync("IdentityId is required")
	}
	if datasetName == "" {
		return cognitoSyncListRecordsOutput{}, validationCognitoSync("DatasetName is required")
	}
	if lastSyncCount < 0 {
		return cognitoSyncListRecordsOutput{}, validationCognitoSync("LastSyncCount must be greater than or equal to 0")
	}
	if maxResults <= 0 {
		maxResults = 1024
	}
	if maxResults > 1024 {
		return cognitoSyncListRecordsOutput{}, validationCognitoSync("MaxResults must be less than or equal to 1024")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	pool, identity := s.ensureIdentityLocked(pool, identityID)

	sessionToken := cognitoSyncSessionToken(pool.ID, identity.ID, datasetName)

	if deletedSyncCount, deleted := identity.DeletedDatasets[datasetName]; deleted {
		if _, err := parseCognitoSyncNextToken(nextToken, 0); err != nil {
			return cognitoSyncListRecordsOutput{}, err
		}
		return cognitoSyncListRecordsOutput{
			Records:                      []cognitoSyncRecordSummary{},
			Count:                        0,
			DatasetExists:                false,
			DatasetDeletedAfterSyncCount: lastSyncCount < deletedSyncCount,
			MergedDatasetNames:           []string{},
			DatasetSyncCount:             deletedSyncCount,
			SyncSessionToken:             sessionToken,
			LastModifiedBy:               "stackyard",
		}, nil
	}

	dataset, ok := identity.Datasets[datasetName]
	if !ok {
		pool, identity, dataset = s.ensureDatasetWritableLocked(pool, identity, datasetName, "stackyard")
		_ = pool
	}
	_ = identity

	recordKeys := make([]string, 0, len(dataset.Records))
	for key := range dataset.Records {
		recordKeys = append(recordKeys, key)
	}
	sort.Strings(recordKeys)

	start, err := parseCognitoSyncNextToken(nextToken, len(recordKeys))
	if err != nil {
		return cognitoSyncListRecordsOutput{}, err
	}
	if start >= len(recordKeys) {
		return cognitoSyncListRecordsOutput{
			Records:                      []cognitoSyncRecordSummary{},
			Count:                        0,
			DatasetExists:                true,
			DatasetDeletedAfterSyncCount: false,
			MergedDatasetNames:           []string{},
			DatasetSyncCount:             dataset.SyncCount,
			SyncSessionToken:             sessionToken,
			LastModifiedBy:               dataset.LastModifiedBy,
		}, nil
	}

	end := start + maxResults
	if end > len(recordKeys) {
		end = len(recordKeys)
	}

	records := make([]cognitoSyncRecordSummary, 0, end-start)
	for _, key := range recordKeys[start:end] {
		record := dataset.Records[key]
		records = append(records, cognitoSyncRecordSummary{
			Key:                    record.Key,
			Value:                  record.Value,
			SyncCount:              record.SyncCount,
			LastModifiedDate:       record.LastModifiedDate,
			LastModifiedBy:         record.LastModifiedBy,
			DeviceLastModifiedDate: record.DeviceLastModifiedDate,
		})
	}

	next := ""
	if end < len(recordKeys) {
		next = strconv.Itoa(end)
	}

	return cognitoSyncListRecordsOutput{
		Records:                      records,
		Count:                        len(records),
		DatasetExists:                true,
		DatasetDeletedAfterSyncCount: false,
		MergedDatasetNames:           []string{},
		DatasetSyncCount:             dataset.SyncCount,
		SyncSessionToken:             sessionToken,
		LastModifiedBy:               dataset.LastModifiedBy,
		NextToken:                    next,
	}, nil
}

func (s *cognitoSyncStore) GetCognitoEvents(identityPoolID string) (map[string]string, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return nil, validationCognitoSync("IdentityPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	return cloneCognitoSyncStringMap(pool.Events), nil
}

func (s *cognitoSyncStore) SetCognitoEvents(identityPoolID string, events map[string]string) (map[string]string, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return nil, validationCognitoSync("IdentityPoolId is required")
	}
	if len(events) > 50 {
		return nil, limitExceededCognitoSync("Events exceeds maximum size")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	if pool.Events == nil {
		pool.Events = map[string]string{}
	}
	for key, value := range events {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return nil, validationCognitoSync("event name is required")
		}
		if strings.TrimSpace(value) == "" {
			delete(pool.Events, trimmedKey)
			continue
		}
		pool.Events[trimmedKey] = strings.TrimSpace(value)
	}
	pool.LastModifiedDate = s.now().UTC()
	s.pools[identityPoolID] = pool
	return cloneCognitoSyncStringMap(pool.Events), nil
}

func (s *cognitoSyncStore) GetIdentityPoolConfiguration(identityPoolID string) (*cognitoSyncPushSync, *cognitoSyncCognitoStreams, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return nil, nil, validationCognitoSync("IdentityPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	return cloneCognitoSyncPushSync(pool.PushSync), cloneCognitoSyncCognitoStreams(pool.CognitoStreams), nil
}

func (s *cognitoSyncStore) SetIdentityPoolConfiguration(identityPoolID string, pushSync *cognitoSyncPushSync, streams *cognitoSyncCognitoStreams) (*cognitoSyncPushSync, *cognitoSyncCognitoStreams, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return nil, nil, validationCognitoSync("IdentityPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	if pushSync != nil {
		pool.PushSync = cloneCognitoSyncPushSync(pushSync)
	}
	if streams != nil {
		pool.CognitoStreams = cloneCognitoSyncCognitoStreams(streams)
	}
	pool.LastModifiedDate = s.now().UTC()
	s.pools[identityPoolID] = pool
	return cloneCognitoSyncPushSync(pool.PushSync), cloneCognitoSyncCognitoStreams(pool.CognitoStreams), nil
}

func (s *cognitoSyncStore) RegisterDevice(identityPoolID, identityID, platform, token string) (string, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	identityID = strings.TrimSpace(identityID)
	platform = strings.ToUpper(strings.TrimSpace(platform))
	token = strings.TrimSpace(token)
	if identityPoolID == "" {
		return "", validationCognitoSync("IdentityPoolId is required")
	}
	if identityID == "" {
		return "", validationCognitoSync("IdentityId is required")
	}
	if token == "" {
		return "", validationCognitoSync("Token is required")
	}
	if !isCognitoSyncDevicePlatform(platform) {
		return "", validationCognitoSync("Platform is invalid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	pool, identity := s.ensureIdentityLocked(pool, identityID)

	if existingID, ok := identity.DeviceByToken[token]; ok {
		device := identity.Devices[existingID]
		device.Platform = platform
		device.LastModifiedDate = s.now().UTC()
		identity.Devices[existingID] = device
		identity.LastModifiedDate = device.LastModifiedDate
		pool.Identities[identity.ID] = identity
		pool.LastModifiedDate = device.LastModifiedDate
		s.pools[pool.ID] = pool
		return existingID, nil
	}

	identity.NextDeviceID++
	deviceID := "device-" + strconv.FormatInt(identity.NextDeviceID, 10)
	now := s.now().UTC()
	identity.Devices[deviceID] = cognitoSyncDevice{
		DeviceID:         deviceID,
		Platform:         platform,
		Token:            token,
		CreationDate:     now,
		LastModifiedDate: now,
	}
	identity.DeviceByToken[token] = deviceID
	identity.LastModifiedDate = now
	pool.Identities[identity.ID] = identity
	pool.LastModifiedDate = now
	s.pools[pool.ID] = pool
	return deviceID, nil
}

func (s *cognitoSyncStore) SubscribeToDataset(identityPoolID, identityID, datasetName, deviceID string) error {
	identityPoolID = strings.TrimSpace(identityPoolID)
	identityID = strings.TrimSpace(identityID)
	datasetName = strings.TrimSpace(datasetName)
	deviceID = strings.TrimSpace(deviceID)
	if identityPoolID == "" {
		return validationCognitoSync("IdentityPoolId is required")
	}
	if identityID == "" {
		return validationCognitoSync("IdentityId is required")
	}
	if datasetName == "" {
		return validationCognitoSync("DatasetName is required")
	}
	if deviceID == "" {
		return validationCognitoSync("DeviceId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	pool, identity := s.ensureIdentityLocked(pool, identityID)
	if _, ok := identity.Devices[deviceID]; !ok {
		return notFoundCognitoSync("Device not found")
	}
	if _, exists := identity.Datasets[datasetName]; !exists && len(identity.Datasets) >= 20 {
		return limitExceededCognitoSync("Identity has reached maximum dataset count")
	}
	pool, identity, dataset := s.ensureDatasetWritableLocked(pool, identity, datasetName, deviceID)
	if dataset.Subscriptions == nil {
		dataset.Subscriptions = map[string]time.Time{}
	}
	dataset.Subscriptions[deviceID] = s.now().UTC()
	now := s.now().UTC()
	dataset.LastModifiedDate = now
	dataset.LastModifiedBy = deviceID
	identity.Datasets[datasetName] = dataset
	identity.LastModifiedDate = now
	pool.Identities[identity.ID] = identity
	pool.LastModifiedDate = now
	s.pools[pool.ID] = pool
	return nil
}

func (s *cognitoSyncStore) UnsubscribeFromDataset(identityPoolID, identityID, datasetName, deviceID string) error {
	identityPoolID = strings.TrimSpace(identityPoolID)
	identityID = strings.TrimSpace(identityID)
	datasetName = strings.TrimSpace(datasetName)
	deviceID = strings.TrimSpace(deviceID)
	if identityPoolID == "" {
		return validationCognitoSync("IdentityPoolId is required")
	}
	if identityID == "" {
		return validationCognitoSync("IdentityId is required")
	}
	if datasetName == "" {
		return validationCognitoSync("DatasetName is required")
	}
	if deviceID == "" {
		return validationCognitoSync("DeviceId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	pool, identity := s.ensureIdentityLocked(pool, identityID)
	dataset, ok := identity.Datasets[datasetName]
	if !ok {
		return notFoundCognitoSync("Dataset not found")
	}
	if _, ok := dataset.Subscriptions[deviceID]; !ok {
		return notFoundCognitoSync("Subscription not found")
	}
	delete(dataset.Subscriptions, deviceID)
	now := s.now().UTC()
	dataset.LastModifiedDate = now
	dataset.LastModifiedBy = deviceID
	identity.Datasets[datasetName] = dataset
	identity.LastModifiedDate = now
	pool.Identities[identity.ID] = identity
	pool.LastModifiedDate = now
	s.pools[pool.ID] = pool
	return nil
}

func (s *cognitoSyncStore) UpdateRecords(identityPoolID, identityID, datasetName, deviceID, syncSessionToken string, patches []cognitoSyncRecordPatchInput) ([]cognitoSyncRecordSummary, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	identityID = strings.TrimSpace(identityID)
	datasetName = strings.TrimSpace(datasetName)
	deviceID = strings.TrimSpace(deviceID)
	syncSessionToken = strings.TrimSpace(syncSessionToken)
	if identityPoolID == "" {
		return nil, validationCognitoSync("IdentityPoolId is required")
	}
	if identityID == "" {
		return nil, validationCognitoSync("IdentityId is required")
	}
	if datasetName == "" {
		return nil, validationCognitoSync("DatasetName is required")
	}
	if deviceID == "" {
		return nil, validationCognitoSync("DeviceId is required")
	}
	if len(patches) == 0 {
		return nil, validationCognitoSync("RecordPatches is required")
	}
	if len(patches) > 1024 {
		return nil, limitExceededCognitoSync("RecordPatches exceeds maximum size")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	pool, identity := s.ensureIdentityLocked(pool, identityID)
	if _, ok := identity.Devices[deviceID]; !ok {
		return nil, notFoundCognitoSync("Device not found")
	}
	if _, exists := identity.Datasets[datasetName]; !exists && len(identity.Datasets) >= 20 {
		return nil, limitExceededCognitoSync("Identity has reached maximum dataset count")
	}
	pool, identity, dataset := s.ensureDatasetWritableLocked(pool, identity, datasetName, deviceID)

	expectedToken := cognitoSyncSessionToken(pool.ID, identity.ID, datasetName)
	if syncSessionToken != "" && syncSessionToken != expectedToken {
		return nil, conflictCognitoSync("SyncSessionToken is invalid")
	}

	for _, patch := range patches {
		op := strings.ToLower(strings.TrimSpace(patch.Op))
		if op != "replace" && op != "remove" {
			return nil, validationCognitoSync("RecordPatch.Op is invalid")
		}
		key := strings.TrimSpace(patch.Key)
		if key == "" {
			return nil, validationCognitoSync("RecordPatch.Key is required")
		}
		currentSyncCount := int64(0)
		if current, exists := dataset.Records[key]; exists {
			currentSyncCount = current.SyncCount
		}
		if patch.SyncCount != currentSyncCount {
			return nil, conflictCognitoSync("Record patch sync count conflict")
		}
	}

	now := s.now().UTC()
	for _, patch := range patches {
		op := strings.ToLower(strings.TrimSpace(patch.Op))
		key := strings.TrimSpace(patch.Key)
		if op == "remove" {
			delete(dataset.Records, key)
			dataset.SyncCount++
			continue
		}

		deviceLastModified := now
		if patch.DeviceLastModifiedDate != nil {
			deviceLastModified = patch.DeviceLastModifiedDate.UTC()
		}
		currentSyncCount := int64(0)
		if current, exists := dataset.Records[key]; exists {
			currentSyncCount = current.SyncCount
		}
		record := cognitoSyncRecord{
			Key:                    key,
			Value:                  patch.Value,
			SyncCount:              currentSyncCount + 1,
			LastModifiedDate:       now,
			LastModifiedBy:         deviceID,
			DeviceLastModifiedDate: deviceLastModified,
		}
		dataset.Records[key] = record
		dataset.SyncCount++
	}

	if s.datasetDataStorageLocked(dataset) > 1024*1024 {
		return nil, limitExceededCognitoSync("Dataset exceeds maximum data size")
	}
	if len(dataset.Records) > 1024 {
		return nil, limitExceededCognitoSync("Dataset exceeds maximum record count")
	}

	dataset.LastModifiedDate = now
	dataset.LastModifiedBy = deviceID
	identity.Datasets[datasetName] = dataset
	delete(identity.DeletedDatasets, datasetName)
	identity.LastModifiedDate = now
	pool.Identities[identity.ID] = identity
	pool.LastModifiedDate = now
	s.pools[pool.ID] = pool

	records := make([]cognitoSyncRecordSummary, 0, len(dataset.Records))
	keys := make([]string, 0, len(dataset.Records))
	for key := range dataset.Records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		record := dataset.Records[key]
		records = append(records, cognitoSyncRecordSummary{
			Key:                    record.Key,
			Value:                  record.Value,
			SyncCount:              record.SyncCount,
			LastModifiedDate:       record.LastModifiedDate,
			LastModifiedBy:         record.LastModifiedBy,
			DeviceLastModifiedDate: record.DeviceLastModifiedDate,
		})
	}
	return records, nil
}

func (s *cognitoSyncStore) DeleteDataset(identityPoolID, identityID, datasetName string) (cognitoSyncDatasetSummary, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	identityID = strings.TrimSpace(identityID)
	datasetName = strings.TrimSpace(datasetName)
	if identityPoolID == "" {
		return cognitoSyncDatasetSummary{}, validationCognitoSync("IdentityPoolId is required")
	}
	if identityID == "" {
		return cognitoSyncDatasetSummary{}, validationCognitoSync("IdentityId is required")
	}
	if datasetName == "" {
		return cognitoSyncDatasetSummary{}, validationCognitoSync("DatasetName is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	pool, identity := s.ensureIdentityLocked(pool, identityID)
	dataset, ok := identity.Datasets[datasetName]
	if !ok {
		return cognitoSyncDatasetSummary{}, notFoundCognitoSync("Dataset not found")
	}

	now := s.now().UTC()
	dataset.LastModifiedDate = now
	dataset.LastModifiedBy = identityID
	summary := s.datasetSummaryLocked(dataset)

	delete(identity.Datasets, datasetName)
	identity.DeletedDatasets[datasetName] = dataset.SyncCount + 1
	identity.LastModifiedDate = now
	pool.Identities[identity.ID] = identity
	pool.LastModifiedDate = now
	s.pools[pool.ID] = pool
	return summary, nil
}

func (s *cognitoSyncStore) BulkPublish(identityPoolID string) (string, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return "", validationCognitoSync("IdentityPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	pool := s.ensurePoolLocked(identityPoolID)
	if pool.BulkPublish != nil && strings.EqualFold(pool.BulkPublish.BulkPublishStatus, "SUCCEEDED") {
		if now.Sub(pool.BulkPublish.BulkPublishComplete) < 24*time.Hour {
			return "", limitExceededCognitoSync("Bulk publish can only be invoked once every 24 hours")
		}
	}

	pool.BulkPublish = &cognitoSyncBulkPublishDetails{
		IdentityPoolID:      identityPoolID,
		BulkPublishStart:    now,
		BulkPublishComplete: now,
		BulkPublishStatus:   "SUCCEEDED",
		FailureMessage:      "",
	}
	pool.LastModifiedDate = now
	s.pools[identityPoolID] = pool
	return identityPoolID, nil
}

func (s *cognitoSyncStore) GetBulkPublishDetails(identityPoolID string) (cognitoSyncBulkPublishDetails, error) {
	identityPoolID = strings.TrimSpace(identityPoolID)
	if identityPoolID == "" {
		return cognitoSyncBulkPublishDetails{}, validationCognitoSync("IdentityPoolId is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ensurePoolLocked(identityPoolID)
	if pool.BulkPublish == nil {
		return cognitoSyncBulkPublishDetails{
			IdentityPoolID:    identityPoolID,
			BulkPublishStatus: "NOT_STARTED",
		}, nil
	}
	return *pool.BulkPublish, nil
}

func (s *cognitoSyncStore) ensurePoolLocked(identityPoolID string) cognitoSyncIdentityPool {
	if pool, ok := s.pools[identityPoolID]; ok {
		if pool.Identities == nil {
			pool.Identities = map[string]cognitoSyncIdentity{}
		}
		if pool.Events == nil {
			pool.Events = map[string]string{}
		}
		return pool
	}
	now := s.now().UTC()
	pool := cognitoSyncIdentityPool{
		ID:               identityPoolID,
		CreationDate:     now,
		LastModifiedDate: now,
		Events:           map[string]string{},
		Identities:       map[string]cognitoSyncIdentity{},
	}
	s.pools[identityPoolID] = pool
	return pool
}

func (s *cognitoSyncStore) ensureIdentityLocked(pool cognitoSyncIdentityPool, identityID string) (cognitoSyncIdentityPool, cognitoSyncIdentity) {
	if identity, ok := pool.Identities[identityID]; ok {
		if identity.Datasets == nil {
			identity.Datasets = map[string]cognitoSyncDataset{}
		}
		if identity.DeletedDatasets == nil {
			identity.DeletedDatasets = map[string]int64{}
		}
		if identity.Devices == nil {
			identity.Devices = map[string]cognitoSyncDevice{}
		}
		if identity.DeviceByToken == nil {
			identity.DeviceByToken = map[string]string{}
		}
		pool.Identities[identityID] = identity
		s.pools[pool.ID] = pool
		return pool, identity
	}
	now := s.now().UTC()
	identity := cognitoSyncIdentity{
		ID:               identityID,
		CreationDate:     now,
		LastModifiedDate: now,
		Datasets:         map[string]cognitoSyncDataset{},
		DeletedDatasets:  map[string]int64{},
		Devices:          map[string]cognitoSyncDevice{},
		DeviceByToken:    map[string]string{},
	}
	pool.Identities[identityID] = identity
	pool.LastModifiedDate = now
	s.pools[pool.ID] = pool
	return pool, identity
}

func (s *cognitoSyncStore) ensureDatasetWritableLocked(pool cognitoSyncIdentityPool, identity cognitoSyncIdentity, datasetName string, actor string) (cognitoSyncIdentityPool, cognitoSyncIdentity, cognitoSyncDataset) {
	if dataset, ok := identity.Datasets[datasetName]; ok {
		if dataset.Records == nil {
			dataset.Records = map[string]cognitoSyncRecord{}
		}
		if dataset.Subscriptions == nil {
			dataset.Subscriptions = map[string]time.Time{}
		}
		identity.Datasets[datasetName] = dataset
		pool.Identities[identity.ID] = identity
		s.pools[pool.ID] = pool
		return pool, identity, dataset
	}
	now := s.now().UTC()
	dataset := cognitoSyncDataset{
		Name:             datasetName,
		IdentityID:       identity.ID,
		CreationDate:     now,
		LastModifiedDate: now,
		LastModifiedBy:   strings.TrimSpace(actor),
		SyncCount:        0,
		Records:          map[string]cognitoSyncRecord{},
		Subscriptions:    map[string]time.Time{},
	}
	identity.Datasets[datasetName] = dataset
	delete(identity.DeletedDatasets, datasetName)
	identity.LastModifiedDate = now
	pool.Identities[identity.ID] = identity
	pool.LastModifiedDate = now
	s.pools[pool.ID] = pool
	return pool, identity, dataset
}

func (s *cognitoSyncStore) poolUsageLocked(pool cognitoSyncIdentityPool) cognitoSyncIdentityPoolUsage {
	dataStorage := int64(0)
	syncSessions := 0
	for _, identity := range pool.Identities {
		for _, dataset := range identity.Datasets {
			dataStorage += s.datasetDataStorageLocked(dataset)
			syncSessions += int(dataset.SyncCount)
		}
	}
	return cognitoSyncIdentityPoolUsage{
		IdentityPoolID:   pool.ID,
		SyncSessions:     syncSessions,
		DataStorage:      dataStorage,
		LastModifiedDate: pool.LastModifiedDate,
	}
}

func (s *cognitoSyncStore) identityUsageLocked(identityPoolID string, identity cognitoSyncIdentity) cognitoSyncIdentityUsage {
	dataStorage := int64(0)
	for _, dataset := range identity.Datasets {
		dataStorage += s.datasetDataStorageLocked(dataset)
	}
	return cognitoSyncIdentityUsage{
		IdentityID:       identity.ID,
		IdentityPoolID:   identityPoolID,
		DatasetCount:     len(identity.Datasets),
		DataStorage:      dataStorage,
		LastModifiedDate: identity.LastModifiedDate,
	}
}

func (s *cognitoSyncStore) datasetSummaryLocked(dataset cognitoSyncDataset) cognitoSyncDatasetSummary {
	return cognitoSyncDatasetSummary{
		Name:             dataset.Name,
		IdentityID:       dataset.IdentityID,
		CreationDate:     dataset.CreationDate,
		LastModifiedDate: dataset.LastModifiedDate,
		LastModifiedBy:   dataset.LastModifiedBy,
		DataStorage:      s.datasetDataStorageLocked(dataset),
		NumRecords:       len(dataset.Records),
	}
}

func (s *cognitoSyncStore) datasetDataStorageLocked(dataset cognitoSyncDataset) int64 {
	total := int64(0)
	for _, record := range dataset.Records {
		total += int64(len(record.Key) + len(record.Value))
	}
	return total
}

func parseCognitoSyncNextToken(nextToken string, max int) (int, error) {
	nextToken = strings.TrimSpace(nextToken)
	if nextToken == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(nextToken)
	if err != nil || value < 0 || value > max {
		return 0, validationCognitoSync("NextToken is invalid")
	}
	return value, nil
}

func cloneCognitoSyncStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneCognitoSyncPushSync(in *cognitoSyncPushSync) *cognitoSyncPushSync {
	if in == nil {
		return nil
	}
	out := &cognitoSyncPushSync{RoleArn: strings.TrimSpace(in.RoleArn)}
	if len(in.ApplicationArns) > 0 {
		out.ApplicationArns = append([]string(nil), in.ApplicationArns...)
	}
	return out
}

func cloneCognitoSyncCognitoStreams(in *cognitoSyncCognitoStreams) *cognitoSyncCognitoStreams {
	if in == nil {
		return nil
	}
	return &cognitoSyncCognitoStreams{
		StreamName:      strings.TrimSpace(in.StreamName),
		RoleArn:         strings.TrimSpace(in.RoleArn),
		StreamingStatus: strings.TrimSpace(in.StreamingStatus),
	}
}

func cognitoSyncSessionToken(identityPoolID, identityID, datasetName string) string {
	return strings.Join([]string{strings.TrimSpace(identityPoolID), strings.TrimSpace(identityID), strings.TrimSpace(datasetName)}, "|")
}

func isCognitoSyncDevicePlatform(platform string) bool {
	switch strings.ToUpper(strings.TrimSpace(platform)) {
	case "APNS", "APNS_SANDBOX", "GCM", "ADM":
		return true
	default:
		return false
	}
}

func validationCognitoSync(message string) error {
	return &cognitoSyncAPIError{Status: 400, Code: "InvalidParameterException", Message: message}
}

func notFoundCognitoSync(message string) error {
	return &cognitoSyncAPIError{Status: 404, Code: "ResourceNotFoundException", Message: message}
}

func conflictCognitoSync(message string) error {
	return &cognitoSyncAPIError{Status: 409, Code: "ResourceConflictException", Message: message}
}

func limitExceededCognitoSync(message string) error {
	return &cognitoSyncAPIError{Status: 400, Code: "LimitExceededException", Message: message}
}

func asCognitoSyncAPIError(err error) *cognitoSyncAPIError {
	if err == nil {
		return nil
	}
	var apiErr *cognitoSyncAPIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}
