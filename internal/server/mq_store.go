package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	mqDefaultRegion    = "us-east-1"
	mqDefaultAccountID = "123456789012"
)

type mqStore struct {
	mu sync.Mutex

	nextBrokerID int64
	nextConfigID int64

	brokers map[string]*mqBroker
	configs map[string]*mqConfiguration
	users   map[string]map[string]*mqUser
	tags    map[string]map[string]string
}

type mqBroker struct {
	ID                 string
	Arn                string
	BrokerName         string
	BrokerState        string
	Created            string
	Updated            string
	EngineType         string
	EngineVersion      string
	HostInstanceType   string
	DeploymentMode     string
	PubliclyAccessible bool
}

type mqConfiguration struct {
	ID             string
	Arn            string
	Name           string
	EngineType     string
	EngineVersion  string
	Created        string
	Updated        string
	LatestRevision int
	Revisions      map[int]*mqConfigurationRevision
}

type mqConfigurationRevision struct {
	Revision    int
	Created     string
	Description string
	Data        string
}

type mqUser struct {
	Username      string
	ConsoleAccess bool
	Groups        []string
	Created       string
	Updated       string
}

func newMQStore() *mqStore {
	now := time.Now().UTC().Format(time.RFC3339)

	broker := &mqBroker{
		ID:                 "b-000001",
		Arn:                mqBrokerARN("stackyard-broker", "b-000001"),
		BrokerName:         "stackyard-broker",
		BrokerState:        "RUNNING",
		Created:            now,
		Updated:            now,
		EngineType:         "ACTIVEMQ",
		EngineVersion:      "5.17.6",
		HostInstanceType:   "mq.t3.micro",
		DeploymentMode:     "SINGLE_INSTANCE",
		PubliclyAccessible: false,
	}

	rev := &mqConfigurationRevision{Revision: 1, Created: now, Description: "seed revision", Data: "<broker/>"}
	config := &mqConfiguration{
		ID:             "c-000001",
		Arn:            mqConfigurationARN("stackyard-configuration", "c-000001"),
		Name:           "stackyard-configuration",
		EngineType:     "ACTIVEMQ",
		EngineVersion:  "5.17.6",
		Created:        now,
		Updated:        now,
		LatestRevision: 1,
		Revisions: map[int]*mqConfigurationRevision{
			1: rev,
		},
	}

	users := map[string]map[string]*mqUser{
		broker.ID: {
			"admin": {
				Username:      "admin",
				ConsoleAccess: true,
				Groups:        []string{"admins"},
				Created:       now,
				Updated:       now,
			},
		},
	}

	return &mqStore{
		nextBrokerID: 2,
		nextConfigID: 2,
		brokers: map[string]*mqBroker{
			broker.ID: broker,
		},
		configs: map[string]*mqConfiguration{
			config.ID: config,
		},
		users: users,
		tags: map[string]map[string]string{
			broker.Arn: {"stackyard": "true", "service": "mq", "env": "coverage"},
			config.Arn: {"stackyard": "true", "service": "mq", "env": "coverage"},
		},
	}
}

func (s *mqStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := mqMergeMaps(payload, pathParams, query)

	brokerID := mqStringAny(ctx, []string{"broker-id", "brokerId", "BrokerId"}, "b-000001")
	username := mqStringAny(ctx, []string{"username", "Username"}, "admin")
	configID := mqStringAny(ctx, []string{"configuration-id", "configurationId", "ConfigurationId"}, "c-000001")
	resourceARN := mqStringAny(ctx, []string{"resource-arn", "resourceArn", "ResourceArn"}, "")

	broker := s.ensureBrokerLocked(brokerID)
	config := s.ensureConfigurationLocked(configID)
	_ = config

	if resourceARN == "" {
		resourceARN = broker.Arn
	}
	s.ensureTagSetLocked(resourceARN)
	s.ensureTagSetLocked(broker.Arn)

	s.applyMutationsLocked(action, payload, broker, config, username)

	switch action {
	case "CreateBroker":
		return map[string]any{"BrokerArn": broker.Arn, "BrokerId": broker.ID}
	case "DescribeBroker":
		return s.describeBrokerOutput(broker)
	case "ListBrokers":
		items := make([]any, 0, len(s.brokers))
		for _, id := range s.sortedBrokerIDsLocked() {
			b := s.brokers[id]
			items = append(items, map[string]any{
				"BrokerArn":          b.Arn,
				"BrokerId":           b.ID,
				"BrokerName":         b.BrokerName,
				"BrokerState":        b.BrokerState,
				"Created":            b.Created,
				"DeploymentMode":     b.DeploymentMode,
				"EngineType":         b.EngineType,
				"HostInstanceType":   b.HostInstanceType,
				"PubliclyAccessible": b.PubliclyAccessible,
			})
		}
		return map[string]any{"BrokerSummaries": items, "NextToken": ""}
	case "UpdateBroker":
		return map[string]any{"BrokerId": broker.ID}
	case "DeleteBroker":
		return map[string]any{"BrokerId": broker.ID}
	case "RebootBroker":
		return map[string]any{"BrokerId": broker.ID}
	case "Promote":
		return map[string]any{"BrokerId": broker.ID, "PromotionMode": "ACTIVE_STANDBY_MULTI_AZ"}
	case "DescribeBrokerEngineTypes":
		return map[string]any{"BrokerEngineTypes": []any{map[string]any{"EngineType": "ACTIVEMQ", "EngineVersions": []any{"5.17.6", "5.18.0"}}, map[string]any{"EngineType": "RABBITMQ", "EngineVersions": []any{"3.13.0"}}}}
	case "DescribeBrokerInstanceOptions":
		return map[string]any{"BrokerInstanceOptions": []any{map[string]any{"EngineType": "ACTIVEMQ", "HostInstanceType": "mq.t3.micro", "StorageType": "EBS"}}, "MaxResults": 20}
	case "CreateConfiguration":
		return map[string]any{"Arn": config.Arn, "Id": config.ID, "LatestRevision": s.configRevisionSummary(config.Revisions[config.LatestRevision])}
	case "DescribeConfiguration":
		return map[string]any{
			"Arn":            config.Arn,
			"Created":        config.Created,
			"Description":    "stackyard configuration",
			"EngineType":     config.EngineType,
			"EngineVersion":  config.EngineVersion,
			"Id":             config.ID,
			"LatestRevision": s.configRevisionSummary(config.Revisions[config.LatestRevision]),
			"Name":           config.Name,
		}
	case "UpdateConfiguration":
		return map[string]any{"Arn": config.Arn, "Id": config.ID, "LatestRevision": s.configRevisionSummary(config.Revisions[config.LatestRevision])}
	case "DeleteConfiguration":
		return map[string]any{"ConfigurationId": config.ID}
	case "ListConfigurations":
		items := make([]any, 0, len(s.configs))
		for _, id := range s.sortedConfigurationIDsLocked() {
			c := s.configs[id]
			items = append(items, map[string]any{
				"Arn":            c.Arn,
				"Created":        c.Created,
				"EngineType":     c.EngineType,
				"EngineVersion":  c.EngineVersion,
				"Id":             c.ID,
				"LatestRevision": s.configRevisionSummary(c.Revisions[c.LatestRevision]),
				"Name":           c.Name,
			})
		}
		return map[string]any{"Configurations": items, "NextToken": ""}
	case "ListConfigurationRevisions":
		revisions := make([]any, 0, len(config.Revisions))
		for _, revNo := range s.sortedRevisionNumbersLocked(config) {
			revisions = append(revisions, s.configRevisionSummary(config.Revisions[revNo]))
		}
		return map[string]any{"Revisions": revisions, "NextToken": ""}
	case "DescribeConfigurationRevision":
		revNo := mqIntAny(ctx, []string{"configuration-revision", "configurationRevision", "ConfigurationRevision"}, config.LatestRevision)
		rev := config.Revisions[revNo]
		if rev == nil {
			rev = config.Revisions[config.LatestRevision]
		}
		return map[string]any{"ConfigurationId": config.ID, "Created": rev.Created, "Data": rev.Data, "Description": rev.Description, "Revision": rev.Revision}
	case "CreateUser":
		user := s.ensureUserLocked(broker.ID, username)
		return map[string]any{"Username": user.Username}
	case "DescribeUser":
		user := s.ensureUserLocked(broker.ID, username)
		return map[string]any{"BrokerId": broker.ID, "Username": user.Username, "ConsoleAccess": user.ConsoleAccess, "Groups": append([]string(nil), user.Groups...), "Pending": map[string]any{}}
	case "UpdateUser":
		user := s.ensureUserLocked(broker.ID, username)
		return map[string]any{"Username": user.Username}
	case "DeleteUser":
		if s.users[broker.ID] != nil {
			delete(s.users[broker.ID], username)
		}
		return map[string]any{}
	case "ListUsers":
		users := s.ensureUserMapLocked(broker.ID)
		items := make([]any, 0, len(users))
		for _, name := range mqSortedUsernames(users) {
			u := users[name]
			items = append(items, map[string]any{"Username": u.Username, "PendingChange": "", "ConsoleAccess": u.ConsoleAccess, "Groups": append([]string(nil), u.Groups...)})
		}
		return map[string]any{"BrokerId": broker.ID, "Users": items}
	case "CreateTags":
		tagSet := s.ensureTagSetLocked(resourceARN)
		for k, v := range mqTagsFromAny(payload["Tags"]) {
			tagSet[k] = v
		}
		for k, v := range mqTagsFromAny(payload["tags"]) {
			tagSet[k] = v
		}
		return map[string]any{}
	case "DeleteTags":
		tagSet := s.ensureTagSetLocked(resourceARN)
		for _, key := range mqTagKeys(ctx, query) {
			delete(tagSet, key)
		}
		return map[string]any{}
	case "ListTags":
		return map[string]any{"Tags": mqCloneStringMap(s.ensureTagSetLocked(resourceARN))}
	default:
		return map[string]any{}
	}
}

func (s *mqStore) applyMutationsLocked(action string, payload map[string]any, broker *mqBroker, config *mqConfiguration, username string) {
	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateBroker":
		name := mqStringAny(payload, []string{"BrokerName", "brokerName", "broker-name"}, "")
		if name == "" {
			name = fmt.Sprintf("stackyard-broker-%06d", s.nextBrokerID)
		}
		id := fmt.Sprintf("b-%06d", s.nextBrokerID)
		s.nextBrokerID++
		created := &mqBroker{
			ID:                 id,
			Arn:                mqBrokerARN(name, id),
			BrokerName:         name,
			BrokerState:        "RUNNING",
			Created:            now,
			Updated:            now,
			EngineType:         strings.ToUpper(mqStringAny(payload, []string{"EngineType", "engineType"}, "ACTIVEMQ")),
			EngineVersion:      mqStringAny(payload, []string{"EngineVersion", "engineVersion"}, "5.17.6"),
			HostInstanceType:   mqStringAny(payload, []string{"HostInstanceType", "hostInstanceType"}, "mq.t3.micro"),
			DeploymentMode:     mqStringAny(payload, []string{"DeploymentMode", "deploymentMode"}, "SINGLE_INSTANCE"),
			PubliclyAccessible: mqBoolAny(payload, []string{"PubliclyAccessible", "publiclyAccessible"}, false),
		}
		s.brokers[id] = created
		s.ensureTagSetLocked(created.Arn)
		broker.ID = created.ID
		broker.Arn = created.Arn
		broker.BrokerName = created.BrokerName
		broker.BrokerState = created.BrokerState
		broker.Created = created.Created
		broker.Updated = created.Updated
		broker.EngineType = created.EngineType
		broker.EngineVersion = created.EngineVersion
		broker.HostInstanceType = created.HostInstanceType
		broker.DeploymentMode = created.DeploymentMode
		broker.PubliclyAccessible = created.PubliclyAccessible

	case "UpdateBroker":
		if engineVersion := mqStringAny(payload, []string{"EngineVersion", "engineVersion"}, ""); engineVersion != "" {
			broker.EngineVersion = engineVersion
		}
		if host := mqStringAny(payload, []string{"HostInstanceType", "hostInstanceType"}, ""); host != "" {
			broker.HostInstanceType = host
		}
		broker.Updated = now
	case "DeleteBroker":
		broker.BrokerState = "DELETED"
		broker.Updated = now
	case "RebootBroker":
		broker.BrokerState = "RUNNING"
		broker.Updated = now
	case "Promote":
		broker.BrokerState = "RUNNING"
		broker.Updated = now
	case "CreateConfiguration":
		name := mqStringAny(payload, []string{"Name", "name"}, "")
		if name == "" {
			name = fmt.Sprintf("stackyard-configuration-%06d", s.nextConfigID)
		}
		id := fmt.Sprintf("c-%06d", s.nextConfigID)
		s.nextConfigID++
		rev := &mqConfigurationRevision{Revision: 1, Created: now, Description: mqStringAny(payload, []string{"Description", "description"}, "created"), Data: mqStringAny(payload, []string{"Data", "data"}, "<broker/>")}
		created := &mqConfiguration{
			ID:             id,
			Arn:            mqConfigurationARN(name, id),
			Name:           name,
			EngineType:     strings.ToUpper(mqStringAny(payload, []string{"EngineType", "engineType"}, "ACTIVEMQ")),
			EngineVersion:  mqStringAny(payload, []string{"EngineVersion", "engineVersion"}, "5.17.6"),
			Created:        now,
			Updated:        now,
			LatestRevision: 1,
			Revisions: map[int]*mqConfigurationRevision{
				1: rev,
			},
		}
		s.configs[id] = created
		s.ensureTagSetLocked(created.Arn)
		*config = *created
	case "UpdateConfiguration":
		config.LatestRevision++
		config.Updated = now
		config.Revisions[config.LatestRevision] = &mqConfigurationRevision{
			Revision:    config.LatestRevision,
			Created:     now,
			Description: mqStringAny(payload, []string{"Description", "description"}, "updated"),
			Data:        mqStringAny(payload, []string{"Data", "data"}, "<broker/>"),
		}
	case "DeleteConfiguration":
		delete(s.configs, config.ID)
	case "CreateUser":
		user := s.ensureUserLocked(broker.ID, username)
		user.ConsoleAccess = mqBoolAny(payload, []string{"ConsoleAccess", "consoleAccess"}, user.ConsoleAccess)
		groups := mqStringSliceAny(payload, []string{"Groups", "groups"})
		if len(groups) > 0 {
			user.Groups = groups
		}
		user.Updated = now
	case "UpdateUser":
		user := s.ensureUserLocked(broker.ID, username)
		user.ConsoleAccess = mqBoolAny(payload, []string{"ConsoleAccess", "consoleAccess"}, user.ConsoleAccess)
		groups := mqStringSliceAny(payload, []string{"Groups", "groups"})
		if len(groups) > 0 {
			user.Groups = groups
		}
		user.Updated = now
	}
}

func (s *mqStore) ensureBrokerLocked(id string) *mqBroker {
	brokerID := strings.TrimSpace(id)
	if brokerID == "" {
		brokerID = "b-000001"
	}
	if existing, ok := s.brokers[brokerID]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := &mqBroker{
		ID:                 brokerID,
		Arn:                mqBrokerARN("stackyard-broker", brokerID),
		BrokerName:         "stackyard-broker",
		BrokerState:        "RUNNING",
		Created:            now,
		Updated:            now,
		EngineType:         "ACTIVEMQ",
		EngineVersion:      "5.17.6",
		HostInstanceType:   "mq.t3.micro",
		DeploymentMode:     "SINGLE_INSTANCE",
		PubliclyAccessible: false,
	}
	s.brokers[brokerID] = created
	s.ensureTagSetLocked(created.Arn)
	return created
}

func (s *mqStore) ensureConfigurationLocked(id string) *mqConfiguration {
	configID := strings.TrimSpace(id)
	if configID == "" {
		configID = "c-000001"
	}
	if existing, ok := s.configs[configID]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	rev := &mqConfigurationRevision{Revision: 1, Created: now, Description: "seed revision", Data: "<broker/>"}
	created := &mqConfiguration{
		ID:             configID,
		Arn:            mqConfigurationARN("stackyard-configuration", configID),
		Name:           "stackyard-configuration",
		EngineType:     "ACTIVEMQ",
		EngineVersion:  "5.17.6",
		Created:        now,
		Updated:        now,
		LatestRevision: 1,
		Revisions: map[int]*mqConfigurationRevision{
			1: rev,
		},
	}
	s.configs[configID] = created
	s.ensureTagSetLocked(created.Arn)
	return created
}

func (s *mqStore) ensureUserMapLocked(brokerID string) map[string]*mqUser {
	id := strings.TrimSpace(brokerID)
	if id == "" {
		id = "b-000001"
	}
	if s.users[id] == nil {
		s.users[id] = map[string]*mqUser{}
	}
	return s.users[id]
}

func (s *mqStore) ensureUserLocked(brokerID, username string) *mqUser {
	users := s.ensureUserMapLocked(brokerID)
	name := strings.TrimSpace(username)
	if name == "" {
		name = "admin"
	}
	if existing, ok := users[name]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := &mqUser{
		Username:      name,
		ConsoleAccess: true,
		Groups:        []string{"admins"},
		Created:       now,
		Updated:       now,
	}
	users[name] = created
	return created
}

func (s *mqStore) ensureTagSetLocked(resourceARN string) map[string]string {
	arn := strings.TrimSpace(resourceARN)
	if arn == "" {
		arn = mqBrokerARN("stackyard-broker", "b-000001")
	}
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{"stackyard": "true", "service": "mq"}
	}
	return s.tags[arn]
}

func (s *mqStore) describeBrokerOutput(b *mqBroker) map[string]any {
	if b == nil {
		b = s.ensureBrokerLocked("b-000001")
	}
	return map[string]any{
		"AuthenticationStrategy":  "simple",
		"AutoMinorVersionUpgrade": true,
		"BrokerArn":               b.Arn,
		"BrokerId":                b.ID,
		"BrokerInstances": []any{map[string]any{
			"ConsoleURL":         "https://" + b.BrokerName + ".mq.console.aws/",
			"Endpoints":          []any{"ssl://" + b.BrokerName + ".mq.local:61617"},
			"IpAddress":          "10.0.0.10",
			"PubliclyAccessible": b.PubliclyAccessible,
		}},
		"BrokerName":         b.BrokerName,
		"BrokerState":        b.BrokerState,
		"Created":            b.Created,
		"DeploymentMode":     b.DeploymentMode,
		"EngineType":         b.EngineType,
		"EngineVersion":      b.EngineVersion,
		"HostInstanceType":   b.HostInstanceType,
		"PubliclyAccessible": b.PubliclyAccessible,
		"Users":              []any{map[string]any{"Username": "admin"}},
	}
}

func (s *mqStore) configRevisionSummary(rev *mqConfigurationRevision) map[string]any {
	if rev == nil {
		return map[string]any{"Revision": 1, "Created": time.Now().UTC().Format(time.RFC3339), "Description": "seed revision"}
	}
	return map[string]any{"Revision": rev.Revision, "Created": rev.Created, "Description": rev.Description}
}

func (s *mqStore) sortedBrokerIDsLocked() []string {
	keys := make([]string, 0, len(s.brokers))
	for key := range s.brokers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *mqStore) sortedConfigurationIDsLocked() []string {
	keys := make([]string, 0, len(s.configs))
	for key := range s.configs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *mqStore) sortedRevisionNumbersLocked(config *mqConfiguration) []int {
	if config == nil {
		return nil
	}
	keys := make([]int, 0, len(config.Revisions))
	for key := range config.Revisions {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}

func mqSortedUsernames(users map[string]*mqUser) []string {
	keys := make([]string, 0, len(users))
	for key := range users {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mqBrokerARN(name, id string) string {
	return fmt.Sprintf("arn:aws:mq:%s:%s:broker:%s:%s", mqDefaultRegion, mqDefaultAccountID, strings.TrimSpace(name), strings.TrimSpace(id))
}

func mqConfigurationARN(name, id string) string {
	return fmt.Sprintf("arn:aws:mq:%s:%s:configuration:%s:%s", mqDefaultRegion, mqDefaultAccountID, strings.TrimSpace(name), strings.TrimSpace(id))
}

func mqMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 1 {
			out[key] = values[0]
		} else if len(values) > 1 {
			arr := make([]any, 0, len(values))
			for _, v := range values {
				arr = append(arr, v)
			}
			out[key] = arr
		}
	}
	return out
}

func mqStringAny(in map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		raw, ok := in[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case string:
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		case json.Number:
			trimmed := strings.TrimSpace(value.String())
			if trimmed != "" {
				return trimmed
			}
		default:
			trimmed := strings.TrimSpace(fmt.Sprintf("%v", raw))
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}

func mqIntAny(in map[string]any, keys []string, fallback int) int {
	for _, key := range keys {
		raw, ok := in[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case int:
			return value
		case int64:
			return int(value)
		case float64:
			return int(value)
		case json.Number:
			if parsed, err := value.Int64(); err == nil {
				return int(parsed)
			}
			if parsed, err := value.Float64(); err == nil {
				return int(parsed)
			}
		case string:
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			if parsed, err := strconv.Atoi(trimmed); err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func mqBoolAny(in map[string]any, keys []string, fallback bool) bool {
	for _, key := range keys {
		raw, ok := in[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case bool:
			return value
		case string:
			trimmed := strings.TrimSpace(strings.ToLower(value))
			if trimmed == "true" {
				return true
			}
			if trimmed == "false" {
				return false
			}
		}
	}
	return fallback
}

func mqStringSliceAny(in map[string]any, keys []string) []string {
	for _, key := range keys {
		raw, ok := in[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case []string:
			out := make([]string, 0, len(value))
			for _, item := range value {
				trimmed := strings.TrimSpace(item)
				if trimmed != "" {
					out = append(out, trimmed)
				}
			}
			if len(out) > 0 {
				return out
			}
		case []any:
			out := make([]string, 0, len(value))
			for _, item := range value {
				trimmed := strings.TrimSpace(fmt.Sprintf("%v", item))
				if trimmed != "" {
					out = append(out, trimmed)
				}
			}
			if len(out) > 0 {
				return out
			}
		case string:
			parts := strings.Split(value, ",")
			out := make([]string, 0, len(parts))
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					out = append(out, trimmed)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return nil
}

func mqTagsFromAny(raw any) map[string]string {
	out := map[string]string{}
	if raw == nil {
		return out
	}
	if typed, ok := raw.(map[string]string); ok {
		for key, value := range typed {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(value)
		}
		return out
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return out
	}
	for key, value := range obj {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(fmt.Sprintf("%v", value))
	}
	return out
}

func mqTagKeys(ctx map[string]any, query url.Values) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	appendKey := func(value string) {
		key := strings.TrimSpace(value)
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}

	for _, keyName := range []string{"TagKeys", "tagKeys"} {
		raw, ok := ctx[keyName]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case []any:
			for _, item := range value {
				appendKey(fmt.Sprintf("%v", item))
			}
		case []string:
			for _, item := range value {
				appendKey(item)
			}
		case string:
			for _, part := range strings.Split(value, ",") {
				appendKey(part)
			}
		default:
			appendKey(fmt.Sprintf("%v", value))
		}
	}

	for _, value := range query["tagKeys"] {
		for _, part := range strings.Split(value, ",") {
			appendKey(part)
		}
	}

	return out
}

func mqCloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
