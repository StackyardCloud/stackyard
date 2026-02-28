package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type iotWirelessStore struct {
	mu   sync.Mutex
	next int64
	tags map[string]map[string]string
}

func newIoTWirelessStore() *iotWirelessStore {
	return &iotWirelessStore{
		next: 1,
		tags: map[string]map[string]string{
			"arn:aws:iotwireless:us-east-1:123456789012:WirelessDevice/stackyard-device": {
				"seed": "true",
			},
		},
	}
}

func (s *iotWirelessStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "TagResource":
		arn := iotWirelessResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = "arn:aws:iotwireless:us-east-1:123456789012:WirelessDevice/stackyard-device"
		}
		incoming := iotWirelessExtractTagMap(iotWirelessValue(payload, "tags"))
		current := s.tags[arn]
		if current == nil {
			current = map[string]string{}
		}
		for k, v := range incoming {
			current[k] = v
		}
		s.tags[arn] = current
		return map[string]any{}

	case "UntagResource":
		arn := iotWirelessResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = "arn:aws:iotwireless:us-east-1:123456789012:WirelessDevice/stackyard-device"
		}
		current := s.tags[arn]
		if current == nil {
			return map[string]any{}
		}
		for _, key := range iotWirelessStringSlice(iotWirelessValue(payload, "tagKeys")) {
			delete(current, key)
		}
		for _, key := range query["tagKeys"] {
			key = strings.TrimSpace(key)
			if key != "" {
				delete(current, key)
			}
		}
		s.tags[arn] = current
		return map[string]any{}

	case "ListTagsForResource":
		arn := iotWirelessResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = "arn:aws:iotwireless:us-east-1:123456789012:WirelessDevice/stackyard-device"
		}
		return map[string]any{"Tags": iotWirelessCloneStringMap(s.tags[arn])}

	case "GetServiceEndpoint":
		return map[string]any{
			"ServiceEndpoint": "https://iotwireless.us-east-1.amazonaws.com",
			"ServerTrust":     "CERTIFICATE",
		}
	case "GetMetricConfiguration":
		return map[string]any{
			"SummaryMetric": map[string]any{
				"Name": "JoinReq",
			},
		}
	case "GetMetrics":
		return map[string]any{
			"MetricQueryResults": []any{
				map[string]any{
					"Label":      "stackyard",
					"Timestamps": []any{now},
					"Values":     []any{1.0},
				},
			},
		}
	case "GetLogLevelsByResourceTypes":
		return map[string]any{"DefaultLogLevel": "ERROR"}
	case "GetEventConfigurationByResourceTypes":
		return map[string]any{"DeviceRegistrationState": map[string]any{"LoRaWAN": map[string]any{"LogLevel": "INFO"}}}
	}

	if strings.HasPrefix(action, "List") {
		key := iotWirelessListKey(action)
		item := map[string]any{
			"Id":               "stackyard",
			"Name":             "stackyard",
			"Arn":              "arn:aws:iotwireless:us-east-1:123456789012:resource/stackyard",
			"DestinationName":  "stackyard-destination",
			"WirelessDeviceId": "stackyard-device",
		}
		return map[string]any{
			key:         []any{item},
			"NextToken": "",
		}
	}

	if strings.HasPrefix(action, "Get") || strings.HasPrefix(action, "Describe") {
		id := iotWirelessResolveID(payload, pathParams)
		return map[string]any{
			"Id":             id,
			"Name":           id,
			"Arn":            iotWirelessARNFor(action, id),
			"Description":    "stackyard " + action,
			"CreationDate":   now,
			"LastModifiedAt": now,
		}
	}

	if strings.HasPrefix(action, "Create") {
		id := iotWirelessResolveID(payload, pathParams)
		if id == "" {
			id = s.nextID("resource")
		}
		return map[string]any{
			"Id":          id,
			"Arn":         iotWirelessARNFor(action, id),
			"Name":        id,
			"MessageId":   s.nextID("msg"),
			"Destination": "stackyard-destination",
		}
	}

	if strings.HasPrefix(action, "Update") ||
		strings.HasPrefix(action, "Associate") ||
		strings.HasPrefix(action, "Disassociate") ||
		strings.HasPrefix(action, "Delete") ||
		strings.HasPrefix(action, "Start") ||
		strings.HasPrefix(action, "Put") ||
		strings.HasPrefix(action, "Reset") ||
		strings.HasPrefix(action, "Send") {
		return map[string]any{
			"Id":        iotWirelessResolveID(payload, pathParams),
			"MessageId": s.nextID("msg"),
			"Status":    "SUCCESS",
			"Timestamp": now,
		}
	}

	return map[string]any{"Operation": action, "Status": "SUCCESS", "Timestamp": now}
}

func (s *iotWirelessStore) nextID(prefix string) string {
	s.next++
	return fmt.Sprintf("%s-%06d", prefix, s.next)
}

func iotWirelessListKey(action string) string {
	keys := map[string]string{
		"ListDestinations":                              "DestinationList",
		"ListDeviceProfiles":                            "DeviceProfileList",
		"ListEventConfigurations":                       "EventConfigurationsList",
		"ListFuotaTasks":                                "FuotaTaskList",
		"ListMulticastGroups":                           "MulticastGroupList",
		"ListMulticastGroupsByFuotaTask":                "MulticastGroupList",
		"ListNetworkAnalyzerConfigurations":             "NetworkAnalyzerConfigurationList",
		"ListPartnerAccounts":                           "Sidewalk",
		"ListPositionConfigurations":                    "PositionConfigurationList",
		"ListQueuedMessages":                            "DownlinkQueueMessagesList",
		"ListServiceProfiles":                           "ServiceProfileList",
		"ListTagsForResource":                           "Tags",
		"ListWirelessDevices":                           "WirelessDeviceList",
		"ListWirelessGateways":                          "WirelessGatewayList",
		"ListWirelessGatewayTaskDefinitions":            "TaskDefinitions",
		"ListWirelessGatewaysByCertificate":             "WirelessGatewayList",
		"ListWirelessGatewayTaskDefinitionVersions":     "TaskDefinitionVersions",
		"ListWirelessDevicesForThing":                   "WirelessDeviceList",
		"ListWirelessGatewaysForThing":                  "WirelessGatewayList",
		"ListDevicesForWirelessDeviceImportTask":        "DestinationNameList",
		"ListWirelessDeviceImportTasks":                 "WirelessDeviceImportTaskList",
		"ListWirelessDevicesForMulticastGroup":          "WirelessDeviceList",
		"ListMulticastGroupsByWirelessDevice":           "MulticastGroupList",
		"ListManagedThingAssociationsForWirelessDevice": "ManagedThingAssociationList",
	}
	if key := keys[action]; key != "" {
		return key
	}
	return "Items"
}

func iotWirelessResolveID(payload map[string]any, pathParams map[string]string) string {
	keys := []string{
		"Id",
		"id",
		"Identifier",
		"ResourceIdentifier",
		"WirelessDeviceId",
		"WirelessGatewayId",
		"PartnerAccountId",
		"Name",
		"ConfigurationName",
		"MulticastGroupId",
	}
	for _, key := range keys {
		if v := iotWirelessPathParam(pathParams, key, ""); v != "" {
			return v
		}
	}
	for _, key := range keys {
		if v := iotWirelessDefaultString(payload, key, ""); v != "" {
			return v
		}
	}
	return "stackyard"
}

func iotWirelessARNFor(action, id string) string {
	typeByAction := map[string]string{
		"WirelessDevice":               "WirelessDevice",
		"WirelessGateway":              "WirelessGateway",
		"Destination":                  "Destination",
		"DeviceProfile":                "DeviceProfile",
		"ServiceProfile":               "ServiceProfile",
		"FuotaTask":                    "FuotaTask",
		"MulticastGroup":               "MulticastGroup",
		"NetworkAnalyzerConfiguration": "NetworkAnalyzerConfiguration",
		"PositionConfiguration":        "PositionConfiguration",
	}
	for marker, resourceType := range typeByAction {
		if strings.Contains(action, marker) {
			return fmt.Sprintf("arn:aws:iotwireless:us-east-1:123456789012:%s/%s", resourceType, id)
		}
	}
	return fmt.Sprintf("arn:aws:iotwireless:us-east-1:123456789012:Resource/%s", id)
}

func iotWirelessResolveResourceARN(payload map[string]any, pathParams map[string]string, query url.Values) string {
	if value := strings.TrimSpace(iotWirelessDefaultString(payload, "ResourceArn", "")); value != "" {
		return value
	}
	if value := strings.TrimSpace(iotWirelessPathParam(pathParams, "ResourceArn", "")); value != "" {
		return value
	}
	if value := strings.TrimSpace(iotWirelessPathParam(pathParams, "ResourceIdentifier", "")); value != "" {
		if strings.HasPrefix(value, "arn:") {
			return value
		}
		return iotWirelessARNFor("Resource", value)
	}
	if value := strings.TrimSpace(query.Get("resourceArn")); value != "" {
		return value
	}
	return ""
}

func iotWirelessValue(payload map[string]any, key string) any {
	if payload == nil {
		return nil
	}
	if value, ok := payload[key]; ok {
		return value
	}
	for k, value := range payload {
		if strings.EqualFold(k, key) {
			return value
		}
	}
	return nil
}

func iotWirelessDefaultString(payload map[string]any, key, fallback string) string {
	value := iotWirelessValue(payload, key)
	text := strings.TrimSpace(iotWirelessToString(value))
	if text == "" {
		return fallback
	}
	return text
}

func iotWirelessPathParam(pathParams map[string]string, key, fallback string) string {
	if pathParams == nil {
		return fallback
	}
	if value, ok := pathParams[key]; ok {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	for k, value := range pathParams {
		if strings.EqualFold(k, key) {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
	}
	return fallback
}

func iotWirelessToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func iotWirelessExtractTagMap(value any) map[string]string {
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]any:
		for key, val := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(iotWirelessToString(val))
		}
	case map[string]string:
		for key, val := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(val)
		}
	}
	return out
}

func iotWirelessCloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = input[key]
	}
	return out
}

func iotWirelessStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			text := strings.TrimSpace(iotWirelessToString(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{strings.TrimSpace(v)}
	default:
		return nil
	}
}
