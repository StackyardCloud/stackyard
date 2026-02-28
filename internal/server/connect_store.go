package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type connectStore struct {
	mu sync.Mutex

	nextInstanceID    int64
	nextUserID        int64
	nextQueueID       int64
	nextContactFlowID int64

	instances    map[string]map[string]any
	users        map[string]map[string]any
	queues       map[string]map[string]any
	contactFlows map[string]map[string]any
	tags         map[string]map[string]string
}

func newConnectStore() *connectStore {
	s := &connectStore{
		nextInstanceID:    2,
		nextUserID:        2,
		nextQueueID:       2,
		nextContactFlowID: 2,
		instances:         map[string]map[string]any{},
		users:             map[string]map[string]any{},
		queues:            map[string]map[string]any{},
		contactFlows:      map[string]map[string]any{},
		tags:              map[string]map[string]string{},
	}
	s.seedLocked(time.Now().UTC())
	return s
}

func (s *connectStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.seedLocked(now)

	ctx := connectMergeMaps(payload, pathParams, query)
	instanceID := connectStringAny(ctx, []string{"InstanceId", "instanceId"}, "instance-000001")
	userID := connectStringAny(ctx, []string{"UserId", "userId"}, "user-000001")
	queueID := connectStringAny(ctx, []string{"QueueId", "queueId"}, "queue-000001")
	contactFlowID := connectStringAny(ctx, []string{"ContactFlowId", "contactFlowId"}, "contact-flow-000001")

	instance := s.ensureInstanceLocked(instanceID, now)
	s.ensureUserLocked(instanceID, userID, now)
	s.ensureQueueLocked(instanceID, queueID, now)
	flow := s.ensureContactFlowLocked(instanceID, contactFlowID, now)
	resourceArn := connectStringAny(ctx, []string{"resourceArn", "ResourceArn"}, connectString(flow, "Arn", connectInstanceARN(instanceID)))

	switch action {
	case "CreateInstance":
		id := fmt.Sprintf("instance-%06d", s.nextInstanceID)
		s.nextInstanceID++
		created := s.ensureInstanceLocked(id, now)
		if alias := connectStringAny(ctx, []string{"InstanceAlias", "instanceAlias"}, ""); alias != "" {
			created["InstanceAlias"] = alias
		}
		return map[string]any{
			"Id":  connectString(created, "Id", id),
			"Arn": connectString(created, "Arn", connectInstanceARN(id)),
		}

	case "DescribeInstance":
		return map[string]any{"Instance": connectCloneMap(instance)}

	case "ListInstances":
		items := make([]any, 0, len(s.instances))
		for _, key := range connectSortedKeys(s.instances) {
			inst := s.instances[key]
			items = append(items, map[string]any{
				"Id":            connectString(inst, "Id", key),
				"Arn":           connectString(inst, "Arn", connectInstanceARN(key)),
				"InstanceAlias": connectString(inst, "InstanceAlias", key),
				"InstanceStatus": connectString(
					inst,
					"InstanceStatus",
					"ACTIVE",
				),
				"CreatedTime": connectString(inst, "CreatedTime", now.Format(time.RFC3339)),
			})
		}
		return map[string]any{"InstanceSummaryList": items, "NextToken": ""}

	case "DeleteInstance":
		delete(s.instances, instanceID)
		prefix := instanceID + ":"
		for key := range s.users {
			if strings.HasPrefix(key, prefix) {
				delete(s.users, key)
			}
		}
		for key := range s.queues {
			if strings.HasPrefix(key, prefix) {
				delete(s.queues, key)
			}
		}
		for key := range s.contactFlows {
			if strings.HasPrefix(key, prefix) {
				delete(s.contactFlows, key)
			}
		}
		return map[string]any{}

	case "CreateUser":
		id := fmt.Sprintf("user-%06d", s.nextUserID)
		s.nextUserID++
		user := s.ensureUserLocked(instanceID, id, now)
		if username := connectStringAny(ctx, []string{"Username", "username"}, ""); username != "" {
			user["Username"] = username
		}
		return map[string]any{
			"UserId":  connectString(user, "Id", id),
			"UserArn": connectString(user, "Arn", connectUserARN(instanceID, id)),
		}

	case "DescribeUser":
		return map[string]any{"User": connectCloneMap(s.ensureUserLocked(instanceID, userID, now))}

	case "ListUsers":
		items := []any{}
		prefix := instanceID + ":"
		for _, key := range connectSortedKeys(s.users) {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			user := s.users[key]
			items = append(items, map[string]any{
				"Id":       connectString(user, "Id", ""),
				"Arn":      connectString(user, "Arn", ""),
				"Username": connectString(user, "Username", ""),
			})
		}
		return map[string]any{"UserSummaryList": items, "NextToken": ""}

	case "DeleteUser":
		delete(s.users, connectResourceKey(instanceID, userID))
		return map[string]any{}

	case "CreateQueue":
		id := fmt.Sprintf("queue-%06d", s.nextQueueID)
		s.nextQueueID++
		queue := s.ensureQueueLocked(instanceID, id, now)
		if name := connectStringAny(ctx, []string{"Name", "name"}, ""); name != "" {
			queue["Name"] = name
		}
		return map[string]any{
			"QueueId":  connectString(queue, "Id", id),
			"QueueArn": connectString(queue, "Arn", connectQueueARN(instanceID, id)),
		}

	case "DescribeQueue":
		return map[string]any{"Queue": connectCloneMap(s.ensureQueueLocked(instanceID, queueID, now))}

	case "ListQueues":
		items := []any{}
		prefix := instanceID + ":"
		for _, key := range connectSortedKeys(s.queues) {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			queue := s.queues[key]
			items = append(items, map[string]any{
				"Id":        connectString(queue, "Id", ""),
				"Arn":       connectString(queue, "Arn", ""),
				"Name":      connectString(queue, "Name", ""),
				"QueueType": connectString(queue, "QueueType", "STANDARD"),
				"LastModifiedTime": connectString(
					queue,
					"LastModifiedTime",
					now.Format(time.RFC3339),
				),
			})
		}
		return map[string]any{"QueueSummaryList": items, "NextToken": ""}

	case "DeleteQueue":
		delete(s.queues, connectResourceKey(instanceID, queueID))
		return map[string]any{}

	case "CreateContactFlow":
		id := fmt.Sprintf("contact-flow-%06d", s.nextContactFlowID)
		s.nextContactFlowID++
		contactFlow := s.ensureContactFlowLocked(instanceID, id, now)
		if name := connectStringAny(ctx, []string{"Name", "name"}, ""); name != "" {
			contactFlow["Name"] = name
		}
		return map[string]any{
			"ContactFlowId":  connectString(contactFlow, "Id", id),
			"ContactFlowArn": connectString(contactFlow, "Arn", connectContactFlowARN(instanceID, id)),
		}

	case "DescribeContactFlow":
		return map[string]any{"ContactFlow": connectCloneMap(s.ensureContactFlowLocked(instanceID, contactFlowID, now))}

	case "ListContactFlows":
		items := []any{}
		prefix := instanceID + ":"
		for _, key := range connectSortedKeys(s.contactFlows) {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			contactFlow := s.contactFlows[key]
			items = append(items, map[string]any{
				"Id":              connectString(contactFlow, "Id", ""),
				"Arn":             connectString(contactFlow, "Arn", ""),
				"Name":            connectString(contactFlow, "Name", ""),
				"ContactFlowType": connectString(contactFlow, "ContactFlowType", "CONTACT_FLOW"),
				"ContactFlowState": connectString(
					contactFlow,
					"ContactFlowState",
					"ACTIVE",
				),
			})
		}
		return map[string]any{"ContactFlowSummaryList": items, "NextToken": ""}

	case "DeleteContactFlow":
		delete(s.contactFlows, connectResourceKey(instanceID, contactFlowID))
		return map[string]any{}

	case "TagResource":
		existing := s.ensureTagsLocked(resourceArn)
		for key, value := range connectMapString(ctx["tags"]) {
			existing[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		existing := s.ensureTagsLocked(resourceArn)
		for _, key := range connectTagKeys(ctx, query) {
			delete(existing, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": connectCloneMapString(s.ensureTagsLocked(resourceArn))}
	}

	return map[string]any{}
}

func (s *connectStore) seedLocked(now time.Time) {
	instanceID := "instance-000001"
	userID := "user-000001"
	queueID := "queue-000001"
	contactFlowID := "contact-flow-000001"

	s.ensureInstanceLocked(instanceID, now)
	s.ensureUserLocked(instanceID, userID, now)
	s.ensureQueueLocked(instanceID, queueID, now)
	flow := s.ensureContactFlowLocked(instanceID, contactFlowID, now)
	s.ensureTagsLocked(connectString(flow, "Arn", connectContactFlowARN(instanceID, contactFlowID)))
}

func (s *connectStore) ensureInstanceLocked(instanceID string, now time.Time) map[string]any {
	if instance := s.instances[instanceID]; instance != nil {
		return instance
	}
	instance := map[string]any{
		"Id":             instanceID,
		"Arn":            connectInstanceARN(instanceID),
		"InstanceAlias":  "stackyard-" + instanceID,
		"InstanceStatus": "ACTIVE",
		"CreatedTime":    now.Add(-15 * time.Minute).Format(time.RFC3339),
	}
	s.instances[instanceID] = instance
	return instance
}

func (s *connectStore) ensureUserLocked(instanceID, userID string, now time.Time) map[string]any {
	key := connectResourceKey(instanceID, userID)
	if user := s.users[key]; user != nil {
		return user
	}
	user := map[string]any{
		"Id":       userID,
		"Arn":      connectUserARN(instanceID, userID),
		"Username": "stackyard-" + userID,
		"DirectoryUserId": map[string]any{
			"Id": userID,
		},
		"LastModifiedTime": now.Format(time.RFC3339),
	}
	s.users[key] = user
	return user
}

func (s *connectStore) ensureQueueLocked(instanceID, queueID string, now time.Time) map[string]any {
	key := connectResourceKey(instanceID, queueID)
	if queue := s.queues[key]; queue != nil {
		return queue
	}
	queue := map[string]any{
		"Id":               queueID,
		"Arn":              connectQueueARN(instanceID, queueID),
		"Name":             "stackyard-" + queueID,
		"QueueType":        "STANDARD",
		"LastModifiedTime": now.Format(time.RFC3339),
	}
	s.queues[key] = queue
	return queue
}

func (s *connectStore) ensureContactFlowLocked(instanceID, contactFlowID string, now time.Time) map[string]any {
	key := connectResourceKey(instanceID, contactFlowID)
	if flow := s.contactFlows[key]; flow != nil {
		return flow
	}
	flow := map[string]any{
		"Id":               contactFlowID,
		"Arn":              connectContactFlowARN(instanceID, contactFlowID),
		"Name":             "stackyard-" + contactFlowID,
		"Description":      "Stackyard contact flow",
		"ContactFlowType":  "CONTACT_FLOW",
		"ContactFlowState": "ACTIVE",
		"Content":          "{}",
		"LastModifiedTime": now.Format(time.RFC3339),
	}
	s.contactFlows[key] = flow
	return flow
}

func (s *connectStore) ensureTagsLocked(resourceARN string) map[string]string {
	if tags := s.tags[resourceARN]; tags != nil {
		return tags
	}
	tags := map[string]string{
		"env":     "local",
		"service": "connect",
	}
	s.tags[resourceARN] = tags
	return tags
}

func connectMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) > 0 {
			out[key] = values[len(values)-1]
		}
	}
	return out
}

func connectStringAny(values map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		if value := connectString(values, key, ""); value != "" {
			return value
		}
	}
	return fallback
}

func connectString(values map[string]any, key, fallback string) string {
	if values == nil {
		return fallback
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return fallback
	}
	switch typed := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed != "" {
			return trimmed
		}
	case fmt.Stringer:
		trimmed := strings.TrimSpace(typed.String())
		if trimmed != "" {
			return trimmed
		}
	}
	return fallback
}

func connectMapString(value any) map[string]string {
	out := map[string]string{}
	rawMap, ok := value.(map[string]any)
	if !ok {
		return out
	}
	for key, val := range rawMap {
		str := strings.TrimSpace(fmt.Sprint(val))
		if strings.TrimSpace(key) == "" || str == "" {
			continue
		}
		out[key] = str
	}
	return out
}

func connectTagKeys(ctx map[string]any, query url.Values) []string {
	out := []string{}
	seen := map[string]struct{}{}

	add := func(key string) {
		key = strings.TrimSpace(key)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}

	for _, raw := range strings.Split(connectString(ctx, "tagKeys", ""), ",") {
		add(raw)
	}
	for _, raw := range query["tagKeys"] {
		for _, token := range strings.Split(raw, ",") {
			add(token)
		}
	}
	if payloadValue, ok := ctx["tagKeys"].([]any); ok {
		for _, item := range payloadValue {
			add(fmt.Sprint(item))
		}
	}
	return out
}

func connectCloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func connectCloneMapString(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func connectSortedKeys[T any](input map[string]T) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func connectResourceKey(instanceID, resourceID string) string {
	return instanceID + ":" + resourceID
}

func connectInstanceARN(instanceID string) string {
	return fmt.Sprintf("arn:aws:connect:us-east-1:123456789012:instance/%s", instanceID)
}

func connectUserARN(instanceID, userID string) string {
	return fmt.Sprintf("%s/agent/%s", connectInstanceARN(instanceID), userID)
}

func connectQueueARN(instanceID, queueID string) string {
	return fmt.Sprintf("%s/queue/%s", connectInstanceARN(instanceID), queueID)
}

func connectContactFlowARN(instanceID, contactFlowID string) string {
	return fmt.Sprintf("%s/contact-flow/%s", connectInstanceARN(instanceID), contactFlowID)
}
