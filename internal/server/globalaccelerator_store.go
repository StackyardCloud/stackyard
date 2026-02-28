package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type globalAcceleratorStore struct {
	mu                            sync.Mutex
	accelerator                   map[string]any
	customRoutingAccelerator      map[string]any
	listener                      map[string]any
	customRoutingListener         map[string]any
	endpointGroup                 map[string]any
	customRoutingEndpointGroup    map[string]any
	crossAccountAttachment        map[string]any
	byoipCidr                     map[string]any
	acceleratorAttributes         map[string]any
	customRoutingAcceleratorAttrs map[string]any
	tags                          map[string]map[string]string
}

func newGlobalAcceleratorStore() *globalAcceleratorStore {
	now := time.Now().UTC()

	acceleratorARN := "arn:aws:globalaccelerator::123456789012:accelerator/stackyard-accelerator"
	customAcceleratorARN := "arn:aws:globalaccelerator::123456789012:accelerator/stackyard-custom-accelerator"
	listenerARN := acceleratorARN + "/listener/stackyard-listener"
	customListenerARN := customAcceleratorARN + "/listener/stackyard-custom-listener"
	endpointGroupARN := listenerARN + "/endpoint-group/stackyard-endpoint-group"
	customEndpointGroupARN := customListenerARN + "/endpoint-group/stackyard-custom-endpoint-group"
	attachmentARN := "arn:aws:globalaccelerator::123456789012:attachment/stackyard-cross-account-attachment"

	accelerator := map[string]any{
		"AcceleratorArn":   acceleratorARN,
		"Name":             "stackyard-accelerator",
		"IpAddressType":    "IPV4",
		"Enabled":          true,
		"Status":           "DEPLOYED",
		"CreatedTime":      now,
		"LastModifiedTime": now,
		"DnsName":          "stackyard.awsglobalaccelerator.com",
		"IpSets": []any{
			map[string]any{
				"IpFamily":    "IPv4",
				"IpAddresses": []any{"198.51.100.10", "198.51.100.11"},
			},
		},
	}
	customAccelerator := map[string]any{
		"AcceleratorArn":   customAcceleratorARN,
		"Name":             "stackyard-custom-accelerator",
		"IpAddressType":    "IPV4",
		"Enabled":          true,
		"Status":           "DEPLOYED",
		"CreatedTime":      now,
		"LastModifiedTime": now,
		"DnsName":          "stackyard-custom.awsglobalaccelerator.com",
		"IpSets": []any{
			map[string]any{
				"IpFamily":    "IPv4",
				"IpAddresses": []any{"198.51.100.20"},
			},
		},
	}
	listener := map[string]any{
		"ListenerArn":    listenerARN,
		"AcceleratorArn": acceleratorARN,
		"Protocol":       "TCP",
		"ClientAffinity": "NONE",
		"PortRanges": []any{
			map[string]any{"FromPort": int64(80), "ToPort": int64(80)},
		},
	}
	customListener := map[string]any{
		"ListenerArn":    customListenerARN,
		"AcceleratorArn": customAcceleratorARN,
		"PortRanges": []any{
			map[string]any{"FromPort": int64(8080), "ToPort": int64(8080)},
		},
	}
	endpointGroup := map[string]any{
		"EndpointGroupArn":           endpointGroupARN,
		"ListenerArn":                listenerARN,
		"EndpointGroupRegion":        "us-east-1",
		"TrafficDialPercentage":      float64(100),
		"HealthCheckPort":            int64(80),
		"HealthCheckProtocol":        "TCP",
		"HealthCheckIntervalSeconds": int64(30),
		"ThresholdCount":             int64(3),
		"EndpointDescriptions": []any{
			map[string]any{
				"EndpointId":                  "i-0123456789abcdef0",
				"HealthState":                 "HEALTHY",
				"Weight":                      int64(128),
				"ClientIPPreservationEnabled": false,
			},
		},
	}
	customEndpointGroup := map[string]any{
		"EndpointGroupArn":    customEndpointGroupARN,
		"ListenerArn":         customListenerARN,
		"EndpointGroupRegion": "us-east-1",
		"DestinationDescriptions": []any{
			map[string]any{
				"EndpointId": "i-0fedcba9876543210",
				"FromPort":   int64(8080),
				"ToPort":     int64(8080),
				"Protocols":  []any{"TCP"},
			},
		},
	}
	attachment := map[string]any{
		"CrossAccountAttachmentArn": attachmentARN,
		"Name":                      "stackyard-cross-account-attachment",
		"AttachmentStatus":          "DEPLOYED",
		"Principals":                []any{"123456789012"},
		"Resources":                 []any{acceleratorARN},
	}
	byoip := map[string]any{
		"Cidr":   "192.0.2.0/24",
		"State":  "ADVERTISED",
		"Events": []any{},
	}

	return &globalAcceleratorStore{
		accelerator:                accelerator,
		customRoutingAccelerator:   customAccelerator,
		listener:                   listener,
		customRoutingListener:      customListener,
		endpointGroup:              endpointGroup,
		customRoutingEndpointGroup: customEndpointGroup,
		crossAccountAttachment:     attachment,
		byoipCidr:                  byoip,
		acceleratorAttributes: map[string]any{
			"FlowLogsEnabled":  false,
			"FlowLogsS3Bucket": "",
			"FlowLogsS3Prefix": "",
		},
		customRoutingAcceleratorAttrs: map[string]any{
			"FlowLogsEnabled": false,
		},
		tags: map[string]map[string]string{
			acceleratorARN:       {"seed": "true"},
			customAcceleratorARN: {"seed": "true"},
		},
	}
}

func (s *globalAcceleratorStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateAccelerator", "UpdateAccelerator", "DescribeAccelerator":
		s.patchAcceleratorLocked(payload)
		return map[string]any{"Accelerator": gaCloneMapAny(s.accelerator)}
	case "DeleteAccelerator":
		return map[string]any{}
	case "DescribeAcceleratorAttributes", "UpdateAcceleratorAttributes":
		s.patchAttributesLocked(payload, s.acceleratorAttributes)
		return map[string]any{"AcceleratorAttributes": gaCloneMapAny(s.acceleratorAttributes)}
	case "ListAccelerators":
		return map[string]any{"Accelerators": []any{gaCloneMapAny(s.accelerator)}, "NextToken": ""}

	case "CreateCustomRoutingAccelerator", "UpdateCustomRoutingAccelerator", "DescribeCustomRoutingAccelerator":
		s.patchCustomRoutingAcceleratorLocked(payload)
		return map[string]any{"Accelerator": gaCloneMapAny(s.customRoutingAccelerator)}
	case "DeleteCustomRoutingAccelerator":
		return map[string]any{}
	case "DescribeCustomRoutingAcceleratorAttributes", "UpdateCustomRoutingAcceleratorAttributes":
		s.patchAttributesLocked(payload, s.customRoutingAcceleratorAttrs)
		return map[string]any{"AcceleratorAttributes": gaCloneMapAny(s.customRoutingAcceleratorAttrs)}
	case "ListCustomRoutingAccelerators":
		return map[string]any{"Accelerators": []any{gaCloneMapAny(s.customRoutingAccelerator)}, "NextToken": ""}

	case "CreateListener", "UpdateListener", "DescribeListener":
		s.patchListenerLocked(payload, s.listener)
		return map[string]any{"Listener": gaCloneMapAny(s.listener)}
	case "DeleteListener":
		return map[string]any{}
	case "ListListeners":
		return map[string]any{"Listeners": []any{gaCloneMapAny(s.listener)}, "NextToken": ""}

	case "CreateCustomRoutingListener", "UpdateCustomRoutingListener", "DescribeCustomRoutingListener":
		s.patchListenerLocked(payload, s.customRoutingListener)
		return map[string]any{"Listener": gaCloneMapAny(s.customRoutingListener)}
	case "DeleteCustomRoutingListener":
		return map[string]any{}
	case "ListCustomRoutingListeners":
		return map[string]any{"Listeners": []any{gaCloneMapAny(s.customRoutingListener)}, "NextToken": ""}

	case "CreateEndpointGroup", "UpdateEndpointGroup", "DescribeEndpointGroup":
		s.patchEndpointGroupLocked(payload, s.endpointGroup)
		return map[string]any{"EndpointGroup": gaCloneMapAny(s.endpointGroup)}
	case "DeleteEndpointGroup":
		return map[string]any{}
	case "ListEndpointGroups":
		return map[string]any{"EndpointGroups": []any{gaCloneMapAny(s.endpointGroup)}, "NextToken": ""}
	case "AddEndpoints", "RemoveEndpoints":
		return map[string]any{"EndpointDescriptions": gaCloneAny(s.endpointGroup["EndpointDescriptions"])}

	case "CreateCustomRoutingEndpointGroup", "DescribeCustomRoutingEndpointGroup":
		s.patchEndpointGroupLocked(payload, s.customRoutingEndpointGroup)
		return map[string]any{"EndpointGroup": gaCloneMapAny(s.customRoutingEndpointGroup)}
	case "DeleteCustomRoutingEndpointGroup":
		return map[string]any{}
	case "ListCustomRoutingEndpointGroups":
		return map[string]any{"EndpointGroups": []any{gaCloneMapAny(s.customRoutingEndpointGroup)}, "NextToken": ""}
	case "AddCustomRoutingEndpoints", "RemoveCustomRoutingEndpoints":
		return map[string]any{}
	case "AllowCustomRoutingTraffic", "DenyCustomRoutingTraffic":
		return map[string]any{}

	case "CreateCrossAccountAttachment", "UpdateCrossAccountAttachment", "DescribeCrossAccountAttachment":
		s.patchCrossAccountAttachmentLocked(payload)
		return map[string]any{"CrossAccountAttachment": gaCloneMapAny(s.crossAccountAttachment)}
	case "DeleteCrossAccountAttachment":
		return map[string]any{}
	case "ListCrossAccountAttachments":
		return map[string]any{"CrossAccountAttachments": []any{gaCloneMapAny(s.crossAccountAttachment)}, "NextToken": ""}
	case "ListCrossAccountResourceAccounts":
		return map[string]any{"CrossAccountResourceAccounts": []any{"123456789012"}, "NextToken": ""}
	case "ListCrossAccountResources":
		return map[string]any{
			"CrossAccountResources": []any{
				map[string]any{
					"EndpointId":    "i-0123456789abcdef0",
					"Cidr":          "198.51.100.0/24",
					"AttachmentArn": s.crossAccountAttachment["CrossAccountAttachmentArn"],
				},
			},
			"NextToken": "",
		}

	case "AdvertiseByoipCidr", "ProvisionByoipCidr", "WithdrawByoipCidr", "DeprovisionByoipCidr":
		s.patchByoipLocked(action, payload)
		return map[string]any{}
	case "ListByoipCidrs":
		return map[string]any{"ByoipCidrs": []any{gaCloneMapAny(s.byoipCidr)}, "NextToken": ""}

	case "ListCustomRoutingPortMappings":
		return map[string]any{
			"PortMappings": []any{
				map[string]any{
					"AcceleratorPort":  int64(8080),
					"EndpointGroupArn": s.customRoutingEndpointGroup["EndpointGroupArn"],
					"EndpointId":       "i-0fedcba9876543210",
					"DestinationSocketAddress": map[string]any{
						"IpAddress": "198.51.100.20",
						"Port":      int64(8080),
					},
				},
			},
			"NextToken": "",
		}
	case "ListCustomRoutingPortMappingsByDestination":
		return map[string]any{
			"DestinationPortMappings": []any{
				map[string]any{
					"AcceleratorArn":   s.customRoutingAccelerator["AcceleratorArn"],
					"EndpointGroupArn": s.customRoutingEndpointGroup["EndpointGroupArn"],
					"EndpointId":       "i-0fedcba9876543210",
					"DestinationSocketAddress": map[string]any{
						"IpAddress": "198.51.100.20",
						"Port":      int64(8080),
					},
					"DestinationTrafficState": "ALLOW",
				},
			},
			"NextToken": "",
		}

	case "TagResource":
		resourceARN := gaDefaultString(payload, []string{"ResourceArn", "resourceArn"}, "")
		if resourceARN == "" {
			resourceARN = gaDefaultString(s.accelerator, []string{"AcceleratorArn"}, "")
		}
		if resourceARN == "" {
			return map[string]any{}
		}
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		for k, v := range gaReadTags(payload) {
			s.tags[resourceARN][k] = v
		}
		return map[string]any{}
	case "UntagResource":
		resourceARN := gaDefaultString(payload, []string{"ResourceArn", "resourceArn"}, "")
		if resourceARN == "" {
			resourceARN = gaDefaultString(s.accelerator, []string{"AcceleratorArn"}, "")
		}
		keys := gaReadTagKeys(payload)
		for _, key := range keys {
			delete(s.tags[resourceARN], key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		resourceARN := gaDefaultString(payload, []string{"ResourceArn", "resourceArn"}, gaDefaultString(s.accelerator, []string{"AcceleratorArn"}, ""))
		return map[string]any{"Tags": gaWriteTags(s.tags[resourceARN])}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"NextToken": ""}
	}

	return map[string]any{}
}

func (s *globalAcceleratorStore) patchAcceleratorLocked(payload map[string]any) {
	now := time.Now().UTC()
	if name := gaDefaultString(payload, []string{"Name", "name"}, ""); name != "" {
		s.accelerator["Name"] = name
	}
	if enabled, ok := gaReadBool(payload, []string{"Enabled", "enabled"}); ok {
		s.accelerator["Enabled"] = enabled
	}
	s.accelerator["LastModifiedTime"] = now
}

func (s *globalAcceleratorStore) patchCustomRoutingAcceleratorLocked(payload map[string]any) {
	now := time.Now().UTC()
	if name := gaDefaultString(payload, []string{"Name", "name"}, ""); name != "" {
		s.customRoutingAccelerator["Name"] = name
	}
	if enabled, ok := gaReadBool(payload, []string{"Enabled", "enabled"}); ok {
		s.customRoutingAccelerator["Enabled"] = enabled
	}
	s.customRoutingAccelerator["LastModifiedTime"] = now
}

func (s *globalAcceleratorStore) patchListenerLocked(payload map[string]any, listener map[string]any) {
	if protocol := gaDefaultString(payload, []string{"Protocol", "protocol"}, ""); protocol != "" {
		listener["Protocol"] = strings.ToUpper(protocol)
	}
	if clientAffinity := gaDefaultString(payload, []string{"ClientAffinity", "clientAffinity"}, ""); clientAffinity != "" {
		listener["ClientAffinity"] = clientAffinity
	}
	if ranges, ok := gaReadAny(payload, []string{"PortRanges", "portRanges"}); ok {
		listener["PortRanges"] = gaCloneAny(ranges)
	}
}

func (s *globalAcceleratorStore) patchEndpointGroupLocked(payload map[string]any, endpointGroup map[string]any) {
	if region := gaDefaultString(payload, []string{"EndpointGroupRegion", "endpointGroupRegion"}, ""); region != "" {
		endpointGroup["EndpointGroupRegion"] = region
	}
	if traffic, ok := gaReadAny(payload, []string{"TrafficDialPercentage", "trafficDialPercentage"}); ok {
		endpointGroup["TrafficDialPercentage"] = traffic
	}
	if endpoints, ok := gaReadAny(payload, []string{"EndpointConfigurations", "endpointConfigurations"}); ok {
		endpointGroup["EndpointDescriptions"] = gaEndpointDescriptionsFromConfigs(endpoints)
	}
}

func (s *globalAcceleratorStore) patchCrossAccountAttachmentLocked(payload map[string]any) {
	if name := gaDefaultString(payload, []string{"Name", "name"}, ""); name != "" {
		s.crossAccountAttachment["Name"] = name
	}
	if principals, ok := gaReadAny(payload, []string{"Principals", "principals"}); ok {
		s.crossAccountAttachment["Principals"] = gaCloneAny(principals)
	}
	if resources, ok := gaReadAny(payload, []string{"Resources", "resources"}); ok {
		s.crossAccountAttachment["Resources"] = gaCloneAny(resources)
	}
}

func (s *globalAcceleratorStore) patchByoipLocked(action string, payload map[string]any) {
	if cidr := gaDefaultString(payload, []string{"Cidr", "cidr"}, ""); cidr != "" {
		s.byoipCidr["Cidr"] = cidr
	}
	switch action {
	case "ProvisionByoipCidr":
		s.byoipCidr["State"] = "PENDING_PROVISIONING"
	case "AdvertiseByoipCidr":
		s.byoipCidr["State"] = "ADVERTISED"
	case "WithdrawByoipCidr":
		s.byoipCidr["State"] = "PENDING_WITHDRAWING"
	case "DeprovisionByoipCidr":
		s.byoipCidr["State"] = "PENDING_DEPROVISIONING"
	}
}

func (s *globalAcceleratorStore) patchAttributesLocked(payload map[string]any, attrs map[string]any) {
	if enabled, ok := gaReadBool(payload, []string{"FlowLogsEnabled", "flowLogsEnabled"}); ok {
		attrs["FlowLogsEnabled"] = enabled
	}
	if bucket := gaDefaultString(payload, []string{"FlowLogsS3Bucket", "flowLogsS3Bucket"}, ""); bucket != "" {
		attrs["FlowLogsS3Bucket"] = bucket
	}
	if prefix := gaDefaultString(payload, []string{"FlowLogsS3Prefix", "flowLogsS3Prefix"}, ""); prefix != "" {
		attrs["FlowLogsS3Prefix"] = prefix
	}
}

func gaReadTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	raw, ok := gaReadAny(payload, []string{"Tags", "tags"})
	if !ok || raw == nil {
		return out
	}

	switch v := raw.(type) {
	case map[string]any:
		for key, value := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(gaAnyString(value))
		}
	case map[string]string:
		for key, value := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(value)
		}
	case []any:
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := strings.TrimSpace(gaDefaultString(m, []string{"Key", "key"}, ""))
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(gaDefaultString(m, []string{"Value", "value"}, ""))
		}
	}
	return out
}

func gaReadTagKeys(payload map[string]any) []string {
	raw, ok := gaReadAny(payload, []string{"TagKeys", "tagKeys"})
	if !ok || raw == nil {
		return nil
	}

	out := []string{}
	switch v := raw.(type) {
	case []any:
		for _, item := range v {
			if s := strings.TrimSpace(gaAnyString(item)); s != "" {
				out = append(out, s)
			}
		}
	case []string:
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
	}
	return out
}

func gaWriteTags(tags map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}

	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}

func gaReadBool(payload map[string]any, keys []string) (bool, bool) {
	value, ok := gaReadAny(payload, keys)
	if !ok {
		return false, false
	}
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "y":
			return true, true
		case "false", "0", "no", "n":
			return false, true
		}
	}
	return false, false
}

func gaDefaultString(payload map[string]any, keys []string, fallback string) string {
	if payload == nil {
		return fallback
	}
	if value, ok := gaReadAny(payload, keys); ok {
		if s := strings.TrimSpace(gaAnyString(value)); s != "" {
			return s
		}
	}
	return fallback
}

func gaReadAny(payload map[string]any, keys []string) (any, bool) {
	for _, key := range keys {
		for existingKey, value := range payload {
			if strings.EqualFold(strings.TrimSpace(existingKey), strings.TrimSpace(key)) {
				return value, true
			}
		}
	}
	return nil, false
}

func gaAnyString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func gaEndpointDescriptionsFromConfigs(configs any) []any {
	items, ok := configs.([]any)
	if !ok || len(items) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		cfg, ok := item.(map[string]any)
		if !ok {
			continue
		}
		endpointID := gaDefaultString(cfg, []string{"EndpointId", "endpointId"}, "i-0123456789abcdef0")
		out = append(out, map[string]any{
			"EndpointId":                  endpointID,
			"HealthState":                 "HEALTHY",
			"Weight":                      int64(128),
			"ClientIPPreservationEnabled": false,
		})
	}
	return out
}

func gaCloneMapAny(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = gaCloneAny(value)
	}
	return out
}

func gaCloneAny(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return gaCloneMapAny(v)
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = gaCloneAny(v[i])
		}
		return out
	default:
		return v
	}
}
