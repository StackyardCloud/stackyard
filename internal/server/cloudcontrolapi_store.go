package server

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cloudControlAPIResourceRecord struct {
	TypeName   string
	Identifier string
	Properties string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type cloudControlAPIProgressEvent struct {
	Operation       string
	OperationStatus string
	TypeName        string
	Identifier      string
	RequestToken    string
	ResourceModel   string
	StatusMessage   string
	ErrorCode       string
	EventTime       time.Time
}

type cloudControlAPIRequestSummary struct {
	Operation       string
	OperationStatus string
	TypeName        string
	Identifier      string
	RequestToken    string
	EventTime       time.Time
}

type cloudControlAPIResourceRequestStatusFilter struct {
	TypeName          string
	Operations        []string
	OperationStatuses []string
}

type cloudControlAPIStore struct {
	mu             sync.Mutex
	resources      map[string]map[string]cloudControlAPIResourceRecord
	requests       map[string]cloudControlAPIProgressEvent
	requestOrder   []string
	clientTokens   map[string]string
	resourceSerial int64
}

func newCloudControlAPIStore() *cloudControlAPIStore {
	return &cloudControlAPIStore{
		resources:    map[string]map[string]cloudControlAPIResourceRecord{},
		requests:     map[string]cloudControlAPIProgressEvent{},
		requestOrder: []string{},
		clientTokens: map[string]string{},
	}
}

func (s *cloudControlAPIStore) CreateResource(typeName, desiredState, clientToken string) (cloudControlAPIProgressEvent, error) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return cloudControlAPIProgressEvent{}, validationCloudControlAPI("TypeName is required")
	}
	desiredState = strings.TrimSpace(desiredState)
	if desiredState == "" {
		return cloudControlAPIProgressEvent{}, validationCloudControlAPI("DesiredState is required")
	}
	if !cloudControlAPIIsJSONObjectString(desiredState) {
		return cloudControlAPIProgressEvent{}, validationCloudControlAPI("DesiredState must be a valid JSON object")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if requestToken := s.lookupClientTokenLocked("CreateResource", typeName, "", clientToken); requestToken != "" {
		if existing, ok := s.requests[requestToken]; ok {
			return existing, nil
		}
	}

	identifier := cloudControlAPIIdentifierFromModel(desiredState)
	if identifier == "" {
		s.resourceSerial++
		identifier = "resource-" + strconv.FormatInt(s.resourceSerial, 10)
	}

	if s.resources[typeName] == nil {
		s.resources[typeName] = map[string]cloudControlAPIResourceRecord{}
	}
	if _, exists := s.resources[typeName][identifier]; exists {
		s.resourceSerial++
		identifier = identifier + "-" + strconv.FormatInt(s.resourceSerial, 10)
	}

	now := time.Now().UTC()
	record := cloudControlAPIResourceRecord{
		TypeName:   typeName,
		Identifier: identifier,
		Properties: desiredState,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.resources[typeName][identifier] = record

	event := cloudControlAPIProgressEvent{
		Operation:       "CREATE",
		OperationStatus: "SUCCESS",
		TypeName:        typeName,
		Identifier:      identifier,
		RequestToken:    cloudControlAPIRequestToken(),
		ResourceModel:   desiredState,
		StatusMessage:   "CreateResource completed",
		EventTime:       now,
	}
	s.recordRequestLocked(event)
	s.storeClientTokenLocked("CreateResource", typeName, "", clientToken, event.RequestToken)
	return event, nil
}

func (s *cloudControlAPIStore) GetResource(typeName, identifier string) (cloudControlAPIResourceRecord, error) {
	typeName = strings.TrimSpace(typeName)
	identifier = strings.TrimSpace(identifier)
	if typeName == "" {
		return cloudControlAPIResourceRecord{}, validationCloudControlAPI("TypeName is required")
	}
	if identifier == "" {
		return cloudControlAPIResourceRecord{}, validationCloudControlAPI("Identifier is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getResourceLocked(typeName, identifier)
}

func (s *cloudControlAPIStore) UpdateResource(typeName, identifier, patchDocument, clientToken string) (cloudControlAPIProgressEvent, error) {
	typeName = strings.TrimSpace(typeName)
	identifier = strings.TrimSpace(identifier)
	patchDocument = strings.TrimSpace(patchDocument)
	if typeName == "" {
		return cloudControlAPIProgressEvent{}, validationCloudControlAPI("TypeName is required")
	}
	if identifier == "" {
		return cloudControlAPIProgressEvent{}, validationCloudControlAPI("Identifier is required")
	}
	if patchDocument == "" {
		return cloudControlAPIProgressEvent{}, validationCloudControlAPI("PatchDocument is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if requestToken := s.lookupClientTokenLocked("UpdateResource", typeName, identifier, clientToken); requestToken != "" {
		if existing, ok := s.requests[requestToken]; ok {
			return existing, nil
		}
	}

	record, err := s.getResourceLocked(typeName, identifier)
	if err != nil {
		return cloudControlAPIProgressEvent{}, err
	}

	updatedModel, err := cloudControlAPIApplyPatch(record.Properties, patchDocument)
	if err != nil {
		return cloudControlAPIProgressEvent{}, err
	}
	record.Properties = updatedModel
	record.UpdatedAt = time.Now().UTC()
	s.resources[typeName][identifier] = record

	event := cloudControlAPIProgressEvent{
		Operation:       "UPDATE",
		OperationStatus: "SUCCESS",
		TypeName:        typeName,
		Identifier:      identifier,
		RequestToken:    cloudControlAPIRequestToken(),
		ResourceModel:   record.Properties,
		StatusMessage:   "UpdateResource completed",
		EventTime:       record.UpdatedAt,
	}
	s.recordRequestLocked(event)
	s.storeClientTokenLocked("UpdateResource", typeName, identifier, clientToken, event.RequestToken)
	return event, nil
}

func (s *cloudControlAPIStore) DeleteResource(typeName, identifier, clientToken string) (cloudControlAPIProgressEvent, error) {
	typeName = strings.TrimSpace(typeName)
	identifier = strings.TrimSpace(identifier)
	if typeName == "" {
		return cloudControlAPIProgressEvent{}, validationCloudControlAPI("TypeName is required")
	}
	if identifier == "" {
		return cloudControlAPIProgressEvent{}, validationCloudControlAPI("Identifier is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if requestToken := s.lookupClientTokenLocked("DeleteResource", typeName, identifier, clientToken); requestToken != "" {
		if existing, ok := s.requests[requestToken]; ok {
			return existing, nil
		}
	}

	record, err := s.getResourceLocked(typeName, identifier)
	if err != nil {
		return cloudControlAPIProgressEvent{}, err
	}

	delete(s.resources[typeName], identifier)
	if len(s.resources[typeName]) == 0 {
		delete(s.resources, typeName)
	}

	now := time.Now().UTC()
	event := cloudControlAPIProgressEvent{
		Operation:       "DELETE",
		OperationStatus: "SUCCESS",
		TypeName:        typeName,
		Identifier:      identifier,
		RequestToken:    cloudControlAPIRequestToken(),
		ResourceModel:   record.Properties,
		StatusMessage:   "DeleteResource completed",
		EventTime:       now,
	}
	s.recordRequestLocked(event)
	s.storeClientTokenLocked("DeleteResource", typeName, identifier, clientToken, event.RequestToken)
	return event, nil
}

func (s *cloudControlAPIStore) ListResources(typeName string, maxResults int, nextToken string) ([]cloudControlAPIResourceRecord, string, error) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return nil, "", validationCloudControlAPI("TypeName is required")
	}
	if maxResults <= 0 {
		maxResults = 100
	}
	if maxResults > 100 {
		maxResults = 100
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	itemsByID := s.resources[typeName]
	ids := make([]string, 0, len(itemsByID))
	for id := range itemsByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	start, err := parseCloudControlAPINextToken(nextToken, len(ids))
	if err != nil {
		return nil, "", err
	}

	end := start + maxResults
	if end > len(ids) {
		end = len(ids)
	}

	items := make([]cloudControlAPIResourceRecord, 0, end-start)
	for _, id := range ids[start:end] {
		items = append(items, itemsByID[id])
	}

	outNextToken := ""
	if end < len(ids) {
		outNextToken = strconv.Itoa(end)
	}
	return items, outNextToken, nil
}

func (s *cloudControlAPIStore) GetResourceRequestStatus(requestToken string) (cloudControlAPIProgressEvent, error) {
	requestToken = strings.TrimSpace(requestToken)
	if requestToken == "" {
		return cloudControlAPIProgressEvent{}, validationCloudControlAPI("RequestToken is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.requests[requestToken]
	if !ok {
		return cloudControlAPIProgressEvent{}, requestTokenNotFoundCloudControlAPI("Request token not found")
	}
	return event, nil
}

func (s *cloudControlAPIStore) CancelResourceRequest(requestToken string) (cloudControlAPIProgressEvent, error) {
	requestToken = strings.TrimSpace(requestToken)
	if requestToken == "" {
		return cloudControlAPIProgressEvent{}, validationCloudControlAPI("RequestToken is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.requests[requestToken]
	if !ok {
		return cloudControlAPIProgressEvent{}, requestTokenNotFoundCloudControlAPI("Request token not found")
	}
	event.OperationStatus = "CANCEL_COMPLETE"
	event.StatusMessage = "CancelResourceRequest completed"
	event.EventTime = time.Now().UTC()
	s.requests[requestToken] = event
	return event, nil
}

func (s *cloudControlAPIStore) ListResourceRequests(maxResults int, nextToken string, filter cloudControlAPIResourceRequestStatusFilter) ([]cloudControlAPIRequestSummary, string, error) {
	if maxResults <= 0 {
		maxResults = 100
	}
	if maxResults > 100 {
		maxResults = 100
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	operationSet := map[string]struct{}{}
	for _, op := range filter.Operations {
		op = strings.ToUpper(strings.TrimSpace(op))
		if op != "" {
			operationSet[op] = struct{}{}
		}
	}
	statusSet := map[string]struct{}{}
	for _, status := range filter.OperationStatuses {
		status = strings.ToUpper(strings.TrimSpace(status))
		if status != "" {
			statusSet[status] = struct{}{}
		}
	}
	typeName := strings.TrimSpace(filter.TypeName)

	summaries := make([]cloudControlAPIRequestSummary, 0, len(s.requestOrder))
	for _, token := range s.requestOrder {
		event, ok := s.requests[token]
		if !ok {
			continue
		}
		if typeName != "" && event.TypeName != typeName {
			continue
		}
		if len(operationSet) > 0 {
			if _, ok := operationSet[strings.ToUpper(event.Operation)]; !ok {
				continue
			}
		}
		if len(statusSet) > 0 {
			if _, ok := statusSet[strings.ToUpper(event.OperationStatus)]; !ok {
				continue
			}
		}
		summaries = append(summaries, cloudControlAPIRequestSummary{
			Operation:       event.Operation,
			OperationStatus: event.OperationStatus,
			TypeName:        event.TypeName,
			Identifier:      event.Identifier,
			RequestToken:    event.RequestToken,
			EventTime:       event.EventTime,
		})
	}

	start, err := parseCloudControlAPINextToken(nextToken, len(summaries))
	if err != nil {
		return nil, "", err
	}
	end := start + maxResults
	if end > len(summaries) {
		end = len(summaries)
	}
	items := summaries[start:end]
	outNextToken := ""
	if end < len(summaries) {
		outNextToken = strconv.Itoa(end)
	}
	return items, outNextToken, nil
}

func (s *cloudControlAPIStore) getResourceLocked(typeName, identifier string) (cloudControlAPIResourceRecord, error) {
	resourcesByType := s.resources[typeName]
	record, ok := resourcesByType[identifier]
	if !ok {
		return cloudControlAPIResourceRecord{}, notFoundCloudControlAPI("Resource not found")
	}
	return record, nil
}

func (s *cloudControlAPIStore) recordRequestLocked(event cloudControlAPIProgressEvent) {
	s.requests[event.RequestToken] = event
	s.requestOrder = append(s.requestOrder, event.RequestToken)
}

func (s *cloudControlAPIStore) lookupClientTokenLocked(operation, typeName, identifier, clientToken string) string {
	clientToken = strings.TrimSpace(clientToken)
	if clientToken == "" {
		return ""
	}
	key := strings.ToUpper(operation) + "|" + typeName + "|" + identifier + "|" + clientToken
	return strings.TrimSpace(s.clientTokens[key])
}

func (s *cloudControlAPIStore) storeClientTokenLocked(operation, typeName, identifier, clientToken, requestToken string) {
	clientToken = strings.TrimSpace(clientToken)
	if clientToken == "" {
		return
	}
	key := strings.ToUpper(operation) + "|" + typeName + "|" + identifier + "|" + clientToken
	s.clientTokens[key] = strings.TrimSpace(requestToken)
}

func cloudControlAPIIsJSONObjectString(value string) bool {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return false
	}
	return parsed != nil
}

func cloudControlAPIIdentifierFromModel(model string) string {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(model), &parsed); err != nil {
		return ""
	}
	candidates := []string{"Identifier", "Id", "Name", "BucketName", "QueueName", "TableName", "FunctionName", "TopicName", "DomainName"}
	for _, key := range candidates {
		raw, ok := parsed[key]
		if !ok {
			continue
		}
		asString, ok := raw.(string)
		if !ok {
			continue
		}
		asString = strings.TrimSpace(asString)
		if asString != "" {
			return asString
		}
	}
	return ""
}

func cloudControlAPIApplyPatch(currentModel, patchDocument string) (string, error) {
	currentModel = strings.TrimSpace(currentModel)
	if currentModel == "" {
		currentModel = "{}"
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(currentModel), &obj); err != nil {
		return "", validationCloudControlAPI("ResourceModel is invalid JSON")
	}
	if obj == nil {
		obj = map[string]any{}
	}

	var patches []map[string]any
	if err := json.Unmarshal([]byte(patchDocument), &patches); err != nil {
		return "", validationCloudControlAPI("PatchDocument must be a valid JSON patch array")
	}

	for _, patch := range patches {
		op, _ := patch["op"].(string)
		path, _ := patch["path"].(string)
		op = strings.ToLower(strings.TrimSpace(op))
		path = strings.TrimSpace(path)
		if path == "" || !strings.HasPrefix(path, "/") {
			return "", validationCloudControlAPI("PatchDocument contains an invalid path")
		}
		segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if len(segments) == 0 || strings.TrimSpace(segments[0]) == "" {
			return "", validationCloudControlAPI("PatchDocument contains an invalid path")
		}
		key := segments[0]
		switch op {
		case "add", "replace":
			obj[key] = patch["value"]
		case "remove":
			delete(obj, key)
		default:
			return "", validationCloudControlAPI("PatchDocument contains an unsupported operation")
		}
	}

	encoded, err := json.Marshal(obj)
	if err != nil {
		return "", validationCloudControlAPI("Unable to encode patched resource model")
	}
	return string(encoded), nil
}

func parseCloudControlAPINextToken(nextToken string, max int) (int, error) {
	nextToken = strings.TrimSpace(nextToken)
	if nextToken == "" {
		return 0, nil
	}
	start, err := strconv.Atoi(nextToken)
	if err != nil || start < 0 || start > max {
		return 0, validationCloudControlAPI("NextToken is invalid")
	}
	return start, nil
}

func cloudControlAPIRequestToken() string {
	return "req-" + randomHex(12)
}

type cloudControlAPIErrorInfo struct {
	Status  int
	Code    string
	Message string
}

func (e *cloudControlAPIErrorInfo) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func validationCloudControlAPI(message string) error {
	return &cloudControlAPIErrorInfo{Status: httpStatusBadRequest, Code: "ValidationException", Message: message}
}

func notFoundCloudControlAPI(message string) error {
	return &cloudControlAPIErrorInfo{Status: httpStatusBadRequest, Code: "ResourceNotFoundException", Message: message}
}

func requestTokenNotFoundCloudControlAPI(message string) error {
	return &cloudControlAPIErrorInfo{Status: httpStatusBadRequest, Code: "RequestTokenNotFoundException", Message: message}
}

func asCloudControlAPIErrorInfo(err error) *cloudControlAPIErrorInfo {
	if err == nil {
		return nil
	}
	var apiErr *cloudControlAPIErrorInfo
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}
