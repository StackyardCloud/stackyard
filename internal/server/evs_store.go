package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type evsStore struct {
	mu sync.Mutex

	nextEnvironment int64
	nextHost        int64
	nextAllocation  int64

	environments map[string]map[string]any
	hosts        map[string]map[string]map[string]any
	vlans        map[string]map[string]map[string]any
	associations map[string]map[string]any
	tags         map[string]map[string]string
}

func newEVSStore() *evsStore {
	s := &evsStore{
		nextEnvironment: 2,
		nextHost:        2,
		nextAllocation:  2,
		environments:    map[string]map[string]any{},
		hosts:           map[string]map[string]map[string]any{},
		vlans:           map[string]map[string]map[string]any{},
		associations:    map[string]map[string]any{},
		tags:            map[string]map[string]string{},
	}

	env := s.ensureEnvironmentLocked("evs-env-00000001")
	host := s.ensureHostLocked("evs-env-00000001", "h-00000001")
	_ = host
	s.ensureVLANLocked("evs-env-00000001", "vlan-0001")
	s.ensureVLANLocked("evs-env-00000001", "vlan-0002")
	s.tags[evsEnvironmentARN(evsDefaultStringAny(env, "environmentId", "evs-env-00000001"))] = map[string]string{"seed": "true"}
	return s
}

func (s *evsStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	defaultEnvID := "evs-env-00000001"
	environmentID := evsDefaultString(pathParams, "environmentId", "")
	if environmentID == "" {
		environmentID = evsDefaultStringAny(payload, "environmentId", defaultEnvID)
	}
	if strings.TrimSpace(environmentID) == "" {
		environmentID = defaultEnvID
	}

	switch action {
	case "CreateEnvironment":
		environmentID = fmt.Sprintf("evs-env-%08d", s.nextEnvironmentIDLocked())
		environment := s.ensureEnvironmentLocked(environmentID)
		environment["environmentName"] = evsDefaultStringAny(payload, "environmentName", fmt.Sprintf("stackyard-evs-%s", environmentID))
		environment["status"] = "CREATING"
		environment["siteId"] = evsDefaultStringAny(payload, "siteId", "site-12345678")
		environment["termsAccepted"] = evsDefaultBool(payload, "termsAccepted", true)
		environment["updatedAt"] = time.Now().UTC()
		s.ensureVLANLocked(environmentID, "vlan-0001")
		s.ensureVLANLocked(environmentID, "vlan-0002")
		return map[string]any{"environment": evsCloneMap(environment)}

	case "CreateEnvironmentHost":
		hostID := fmt.Sprintf("h-%08d", s.nextHostIDLocked())
		host := s.ensureHostLocked(environmentID, hostID)
		host["hostName"] = evsDefaultStringAny(payload, "hostName", fmt.Sprintf("esxi-%s", hostID))
		host["instanceType"] = evsDefaultStringAny(payload, "instanceType", "i4i.metal")
		host["status"] = "CREATING"
		host["updatedAt"] = time.Now().UTC()
		return map[string]any{"host": evsCloneMap(host)}

	case "DeleteEnvironment":
		environment := s.ensureEnvironmentLocked(environmentID)
		environment["status"] = "DELETING"
		environment["updatedAt"] = time.Now().UTC()
		delete(s.hosts, environmentID)
		delete(s.vlans, environmentID)
		return map[string]any{"environment": evsCloneMap(environment)}

	case "DeleteEnvironmentHost":
		hostID := evsDefaultString(pathParams, "hostId", evsDefaultStringAny(payload, "hostId", "h-00000001"))
		host := s.ensureHostLocked(environmentID, hostID)
		host["status"] = "DELETING"
		host["updatedAt"] = time.Now().UTC()
		if s.hosts[environmentID] != nil {
			delete(s.hosts[environmentID], hostID)
		}
		return map[string]any{"host": evsCloneMap(host)}

	case "AssociateEipToVlan":
		vlanID := evsDefaultString(pathParams, "vlanId", evsDefaultStringAny(payload, "vlanId", "vlan-0001"))
		allocationID := evsDefaultStringAny(payload, "allocationId", "")
		if allocationID == "" {
			allocationID = evsDefaultStringAny(payload, "eipAllocationId", "")
		}
		if allocationID == "" {
			allocationID = fmt.Sprintf("eipalloc-%08d", s.nextAllocationIDLocked())
		}
		association := map[string]any{
			"environmentId": environmentID,
			"vlanId":        vlanID,
			"allocationId":  allocationID,
			"status":        "ASSOCIATED",
			"associatedAt":  time.Now().UTC(),
		}
		s.associations[evsAssociationKey(environmentID, vlanID, allocationID)] = evsCloneMap(association)
		s.ensureVLANLocked(environmentID, vlanID)
		return map[string]any{"eipAssociation": evsCloneMap(association)}

	case "DisassociateEipFromVlan":
		vlanID := evsDefaultString(pathParams, "vlanId", evsDefaultStringAny(payload, "vlanId", "vlan-0001"))
		allocationID := evsDefaultString(pathParams, "allocationId", evsDefaultStringAny(payload, "allocationId", "eipalloc-00000001"))
		key := evsAssociationKey(environmentID, vlanID, allocationID)
		association := s.associations[key]
		if association == nil {
			association = map[string]any{
				"environmentId": environmentID,
				"vlanId":        vlanID,
				"allocationId":  allocationID,
			}
		}
		association["status"] = "DISASSOCIATED"
		association["updatedAt"] = time.Now().UTC()
		delete(s.associations, key)
		return map[string]any{"eipAssociation": evsCloneMap(association)}

	case "GetEnvironment":
		return map[string]any{"environment": evsCloneMap(s.ensureEnvironmentLocked(environmentID))}

	case "GetVersions":
		return map[string]any{
			"versions": []any{
				map[string]any{"version": "5.2.0", "isDefault": true},
				map[string]any{"version": "5.1.1", "isDefault": false},
			},
			"nextToken": "",
		}

	case "ListEnvironmentHosts":
		hosts := s.listHostsLocked(environmentID)
		return map[string]any{"hosts": hosts, "nextToken": ""}

	case "ListEnvironments":
		environments := make([]any, 0, len(s.environments))
		keys := make([]string, 0, len(s.environments))
		for k := range s.environments {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			env := s.environments[key]
			environments = append(environments, map[string]any{
				"environmentId":   evsDefaultStringAny(env, "environmentId", key),
				"environmentName": evsDefaultStringAny(env, "environmentName", key),
				"status":          evsDefaultStringAny(env, "status", "CREATED"),
			})
		}
		return map[string]any{"environments": environments, "nextToken": ""}

	case "ListEnvironmentVlans":
		vlans := s.listVLANsLocked(environmentID)
		return map[string]any{"vlans": vlans, "nextToken": ""}

	case "TagResource":
		resourceARN := evsDefaultString(pathParams, "resourceArn", "")
		if resourceARN == "" {
			resourceARN = evsDefaultStringAny(payload, "resourceArn", evsEnvironmentARN(environmentID))
		}
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		for key, value := range evsStringMap(payload, "tags") {
			s.tags[resourceARN][key] = value
		}
		if len(s.tags[resourceARN]) == 0 {
			for key, value := range evsStringMap(payload, "Tags") {
				s.tags[resourceARN][key] = value
			}
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := evsDefaultString(pathParams, "resourceArn", "")
		if resourceARN == "" {
			resourceARN = evsDefaultStringAny(payload, "resourceArn", evsEnvironmentARN(environmentID))
		}
		tagKeys := evsStringSlice(payload, "tagKeys")
		if len(tagKeys) == 0 {
			tagKeys = evsStringSlice(payload, "TagKeys")
		}
		if len(tagKeys) == 0 {
			tagKeys = query["tagKeys"]
		}
		for _, key := range tagKeys {
			if s.tags[resourceARN] != nil {
				delete(s.tags[resourceARN], key)
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := evsDefaultString(pathParams, "resourceArn", "")
		if resourceARN == "" {
			resourceARN = evsDefaultStringAny(payload, "resourceArn", evsEnvironmentARN(environmentID))
		}
		return map[string]any{"tags": evsCloneStringMap(s.tags[resourceARN])}
	}

	return map[string]any{}
}

func (s *evsStore) ensureEnvironmentLocked(environmentID string) map[string]any {
	id := strings.TrimSpace(environmentID)
	if id == "" {
		id = "evs-env-00000001"
	}
	if env := s.environments[id]; env != nil {
		return env
	}
	env := map[string]any{
		"environmentId":   id,
		"environmentName": fmt.Sprintf("stackyard-%s", id),
		"status":          "CREATED",
		"createdAt":       time.Now().UTC(),
		"siteId":          "site-12345678",
		"termsAccepted":   true,
		"environmentArn":  evsEnvironmentARN(id),
	}
	s.environments[id] = env
	return env
}

func (s *evsStore) ensureHostLocked(environmentID, hostID string) map[string]any {
	envID := strings.TrimSpace(environmentID)
	if envID == "" {
		envID = "evs-env-00000001"
	}
	_ = s.ensureEnvironmentLocked(envID)
	if s.hosts[envID] == nil {
		s.hosts[envID] = map[string]map[string]any{}
	}
	id := strings.TrimSpace(hostID)
	if id == "" {
		id = "h-00000001"
	}
	if host := s.hosts[envID][id]; host != nil {
		return host
	}
	host := map[string]any{
		"environmentId": envID,
		"hostId":        id,
		"hostName":      fmt.Sprintf("esxi-%s", id),
		"instanceType":  "i4i.metal",
		"status":        "CREATED",
		"createdAt":     time.Now().UTC(),
		"hostArn":       evsHostARN(envID, id),
	}
	s.hosts[envID][id] = host
	return host
}

func (s *evsStore) ensureVLANLocked(environmentID, vlanID string) map[string]any {
	envID := strings.TrimSpace(environmentID)
	if envID == "" {
		envID = "evs-env-00000001"
	}
	_ = s.ensureEnvironmentLocked(envID)
	if s.vlans[envID] == nil {
		s.vlans[envID] = map[string]map[string]any{}
	}
	id := strings.TrimSpace(vlanID)
	if id == "" {
		id = "vlan-0001"
	}
	if vlan := s.vlans[envID][id]; vlan != nil {
		return vlan
	}
	vlan := map[string]any{
		"environmentId": envID,
		"vlanId":        id,
		"cidr":          "10.0.0.0/24",
		"status":        "AVAILABLE",
	}
	s.vlans[envID][id] = vlan
	return vlan
}

func (s *evsStore) listHostsLocked(environmentID string) []any {
	envID := strings.TrimSpace(environmentID)
	if envID == "" {
		envID = "evs-env-00000001"
	}
	if s.hosts[envID] == nil {
		_ = s.ensureHostLocked(envID, "h-00000001")
	}
	keys := make([]string, 0, len(s.hosts[envID]))
	for k := range s.hosts[envID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, evsCloneMap(s.hosts[envID][key]))
	}
	return out
}

func (s *evsStore) listVLANsLocked(environmentID string) []any {
	envID := strings.TrimSpace(environmentID)
	if envID == "" {
		envID = "evs-env-00000001"
	}
	if s.vlans[envID] == nil {
		_ = s.ensureVLANLocked(envID, "vlan-0001")
		_ = s.ensureVLANLocked(envID, "vlan-0002")
	}
	keys := make([]string, 0, len(s.vlans[envID]))
	for k := range s.vlans[envID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, evsCloneMap(s.vlans[envID][key]))
	}
	return out
}

func (s *evsStore) nextEnvironmentIDLocked() int64 {
	id := s.nextEnvironment
	s.nextEnvironment++
	return id
}

func (s *evsStore) nextHostIDLocked() int64 {
	id := s.nextHost
	s.nextHost++
	return id
}

func (s *evsStore) nextAllocationIDLocked() int64 {
	id := s.nextAllocation
	s.nextAllocation++
	return id
}

func evsAssociationKey(environmentID, vlanID, allocationID string) string {
	return strings.TrimSpace(environmentID) + ":" + strings.TrimSpace(vlanID) + ":" + strings.TrimSpace(allocationID)
}

func evsEnvironmentARN(environmentID string) string {
	return fmt.Sprintf("arn:aws:evs:us-east-1:123456789012:environment/%s", strings.TrimSpace(environmentID))
}

func evsHostARN(environmentID, hostID string) string {
	return fmt.Sprintf("arn:aws:evs:us-east-1:123456789012:environment/%s/host/%s", strings.TrimSpace(environmentID), strings.TrimSpace(hostID))
}

func evsDefaultString(values map[string]string, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			value := strings.TrimSpace(v)
			if value != "" {
				return value
			}
			break
		}
	}
	return fallback
}

func evsDefaultStringAny(values map[string]any, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if str, ok := v.(string); ok {
				str = strings.TrimSpace(str)
				if str != "" {
					return str
				}
			}
			break
		}
	}
	return fallback
}

func evsDefaultBool(values map[string]any, key string, fallback bool) bool {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if b, ok := v.(bool); ok {
				return b
			}
			break
		}
	}
	return fallback
}

func evsStringMap(payload map[string]any, key string) map[string]string {
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		switch raw := v.(type) {
		case map[string]any:
			out := map[string]string{}
			for rk, rv := range raw {
				if rs, ok := rv.(string); ok {
					out[rk] = rs
				}
			}
			return out
		case map[string]string:
			return evsCloneStringMap(raw)
		}
	}
	return map[string]string{}
}

func evsStringSlice(payload map[string]any, key string) []string {
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		raw, ok := v.([]any)
		if !ok {
			return nil
		}
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if str, ok := item.(string); ok {
				str = strings.TrimSpace(str)
				if str != "" {
					out = append(out, str)
				}
			}
		}
		return out
	}
	return nil
}

func evsCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = evsCloneAny(v)
	}
	return out
}

func evsCloneAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return evsCloneMap(t)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = evsCloneAny(item)
		}
		return out
	case map[string]string:
		return evsCloneStringMap(t)
	default:
		return t
	}
}

func evsCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
