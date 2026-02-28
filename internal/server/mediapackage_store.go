package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type mediaPackageStore struct {
	mu sync.Mutex

	nextGroupID     int64
	nextChannelID   int64
	nextEndpointID  int64
	nextHarvestID   int64
	channelGroups   map[string]map[string]any
	channels        map[string]map[string]any
	originEndpoints map[string]map[string]any
	harvestJobs     map[string]map[string]any
	channelPolicies map[string]string
	endpointPolicy  map[string]string
	tags            map[string]map[string]string
}

func newMediaPackageStore() *mediaPackageStore {
	s := &mediaPackageStore{
		nextGroupID:     2,
		nextChannelID:   2,
		nextEndpointID:  2,
		nextHarvestID:   2,
		channelGroups:   map[string]map[string]any{},
		channels:        map[string]map[string]any{},
		originEndpoints: map[string]map[string]any{},
		harvestJobs:     map[string]map[string]any{},
		channelPolicies: map[string]string{},
		endpointPolicy:  map[string]string{},
		tags:            map[string]map[string]string{},
	}

	group := s.ensureChannelGroupLocked("channel-group-00000001")
	channel := s.ensureChannelLocked("channel-group-00000001", "channel-00000001")
	endpoint := s.ensureOriginEndpointLocked("channel-group-00000001", "channel-00000001", "origin-endpoint-00000001")
	job := s.ensureHarvestJobLocked("channel-group-00000001", "channel-00000001", "origin-endpoint-00000001", "harvest-job-00000001")
	s.channelPolicies[mpkgChannelKey("channel-group-00000001", "channel-00000001")] = `{"Version":"2012-10-17","Statement":[]}`
	s.endpointPolicy[mpkgEndpointKey("channel-group-00000001", "channel-00000001", "origin-endpoint-00000001")] = `{"Version":"2012-10-17","Statement":[]}`
	s.tags[mpkgStringAny(group, "Arn")] = map[string]string{"seed": "true"}
	s.tags[mpkgStringAny(channel, "Arn")] = map[string]string{"seed": "true"}
	s.tags[mpkgStringAny(endpoint, "Arn")] = map[string]string{"seed": "true"}
	s.tags[mpkgStringAny(job, "Arn")] = map[string]string{"seed": "true"}

	return s
}

func (s *mediaPackageStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	channelGroupName := mpkgFirstNonEmpty(
		mpkgPathParam(pathParams, "ChannelGroupName"),
		mpkgStringAny(payload, "ChannelGroupName", "channelGroupName"),
		"channel-group-00000001",
	)
	channelName := mpkgFirstNonEmpty(
		mpkgPathParam(pathParams, "ChannelName"),
		mpkgStringAny(payload, "ChannelName", "channelName"),
		"channel-00000001",
	)
	originEndpointName := mpkgFirstNonEmpty(
		mpkgPathParam(pathParams, "OriginEndpointName"),
		mpkgStringAny(payload, "OriginEndpointName", "originEndpointName"),
		"origin-endpoint-00000001",
	)
	harvestJobName := mpkgFirstNonEmpty(
		mpkgPathParam(pathParams, "HarvestJobName"),
		mpkgStringAny(payload, "HarvestJobName", "harvestJobName"),
		"harvest-job-00000001",
	)
	resourceARN := mpkgFirstNonEmpty(
		mpkgPathParam(pathParams, "ResourceArn"),
		mpkgStringAny(payload, "ResourceArn", "resourceArn"),
		mpkgChannelGroupARN(channelGroupName),
	)

	switch action {
	case "CreateChannelGroup":
		channelGroupName = mpkgFirstNonEmpty(
			mpkgStringAny(payload, "ChannelGroupName", "channelGroupName"),
			fmt.Sprintf("channel-group-%08d", s.nextGroupIDLocked()),
		)
		group := s.ensureChannelGroupLocked(channelGroupName)
		for k, v := range payload {
			group[k] = v
		}
		group["ChannelGroupName"] = channelGroupName
		group["Arn"] = mpkgChannelGroupARN(channelGroupName)
		group["CreatedAt"] = mpkgFirstNonEmpty(mpkgStringAny(group, "CreatedAt"), now)
		group["ModifiedAt"] = now
		return mpkgCloneMap(group)

	case "GetChannelGroup":
		return mpkgCloneMap(s.ensureChannelGroupLocked(channelGroupName))

	case "ListChannelGroups":
		return map[string]any{
			"Items":     s.listChannelGroupsLocked(),
			"NextToken": "",
		}

	case "UpdateChannelGroup":
		group := s.ensureChannelGroupLocked(channelGroupName)
		for k, v := range payload {
			group[k] = v
		}
		group["ChannelGroupName"] = channelGroupName
		group["Arn"] = mpkgChannelGroupARN(channelGroupName)
		group["ModifiedAt"] = now
		return mpkgCloneMap(group)

	case "DeleteChannelGroup":
		delete(s.channelGroups, channelGroupName)
		for key := range s.channels {
			if strings.HasPrefix(key, channelGroupName+"/") {
				delete(s.channels, key)
			}
		}
		for key := range s.originEndpoints {
			if strings.HasPrefix(key, channelGroupName+"/") {
				delete(s.originEndpoints, key)
			}
		}
		for key := range s.harvestJobs {
			if strings.HasPrefix(key, channelGroupName+"/") {
				delete(s.harvestJobs, key)
			}
		}
		return map[string]any{}

	case "CreateChannel":
		channelName = mpkgFirstNonEmpty(
			mpkgStringAny(payload, "ChannelName", "channelName"),
			fmt.Sprintf("channel-%08d", s.nextChannelIDLocked()),
		)
		ch := s.ensureChannelLocked(channelGroupName, channelName)
		for k, v := range payload {
			ch[k] = v
		}
		ch["ChannelName"] = channelName
		ch["ChannelGroupName"] = channelGroupName
		ch["Arn"] = mpkgChannelARN(channelGroupName, channelName)
		ch["CreatedAt"] = mpkgFirstNonEmpty(mpkgStringAny(ch, "CreatedAt"), now)
		ch["ModifiedAt"] = now
		return mpkgCloneMap(ch)

	case "GetChannel":
		return mpkgCloneMap(s.ensureChannelLocked(channelGroupName, channelName))

	case "ListChannels":
		return map[string]any{
			"Items":     s.listChannelsLocked(channelGroupName),
			"NextToken": "",
		}

	case "UpdateChannel":
		ch := s.ensureChannelLocked(channelGroupName, channelName)
		for k, v := range payload {
			ch[k] = v
		}
		ch["ChannelName"] = channelName
		ch["ChannelGroupName"] = channelGroupName
		ch["ModifiedAt"] = now
		return mpkgCloneMap(ch)

	case "DeleteChannel":
		delete(s.channels, mpkgChannelKey(channelGroupName, channelName))
		for key := range s.originEndpoints {
			if strings.HasPrefix(key, mpkgChannelKey(channelGroupName, channelName)+"/") {
				delete(s.originEndpoints, key)
			}
		}
		for key := range s.harvestJobs {
			if strings.HasPrefix(key, mpkgChannelKey(channelGroupName, channelName)+"/") {
				delete(s.harvestJobs, key)
			}
		}
		delete(s.channelPolicies, mpkgChannelKey(channelGroupName, channelName))
		return map[string]any{}

	case "CreateOriginEndpoint":
		originEndpointName = mpkgFirstNonEmpty(
			mpkgStringAny(payload, "OriginEndpointName", "originEndpointName"),
			fmt.Sprintf("origin-endpoint-%08d", s.nextEndpointIDLocked()),
		)
		ep := s.ensureOriginEndpointLocked(channelGroupName, channelName, originEndpointName)
		for k, v := range payload {
			ep[k] = v
		}
		ep["OriginEndpointName"] = originEndpointName
		ep["ChannelName"] = channelName
		ep["ChannelGroupName"] = channelGroupName
		ep["Arn"] = mpkgOriginEndpointARN(channelGroupName, channelName, originEndpointName)
		ep["CreatedAt"] = mpkgFirstNonEmpty(mpkgStringAny(ep, "CreatedAt"), now)
		ep["ModifiedAt"] = now
		return mpkgCloneMap(ep)

	case "GetOriginEndpoint":
		return mpkgCloneMap(s.ensureOriginEndpointLocked(channelGroupName, channelName, originEndpointName))

	case "ListOriginEndpoints":
		return map[string]any{
			"Items":     s.listOriginEndpointsLocked(channelGroupName, channelName),
			"NextToken": "",
		}

	case "UpdateOriginEndpoint":
		ep := s.ensureOriginEndpointLocked(channelGroupName, channelName, originEndpointName)
		for k, v := range payload {
			ep[k] = v
		}
		ep["ModifiedAt"] = now
		return mpkgCloneMap(ep)

	case "DeleteOriginEndpoint":
		delete(s.originEndpoints, mpkgEndpointKey(channelGroupName, channelName, originEndpointName))
		for key := range s.harvestJobs {
			if strings.HasPrefix(key, mpkgEndpointKey(channelGroupName, channelName, originEndpointName)+"/") {
				delete(s.harvestJobs, key)
			}
		}
		delete(s.endpointPolicy, mpkgEndpointKey(channelGroupName, channelName, originEndpointName))
		return map[string]any{}

	case "CreateHarvestJob":
		harvestJobName = mpkgFirstNonEmpty(
			mpkgStringAny(payload, "HarvestJobName", "harvestJobName"),
			fmt.Sprintf("harvest-job-%08d", s.nextHarvestIDLocked()),
		)
		job := s.ensureHarvestJobLocked(channelGroupName, channelName, originEndpointName, harvestJobName)
		for k, v := range payload {
			job[k] = v
		}
		job["HarvestJobName"] = harvestJobName
		job["OriginEndpointName"] = originEndpointName
		job["ChannelName"] = channelName
		job["ChannelGroupName"] = channelGroupName
		job["Arn"] = mpkgHarvestJobARN(channelGroupName, channelName, originEndpointName, harvestJobName)
		job["Status"] = mpkgFirstNonEmpty(mpkgStringAny(job, "Status"), "IN_PROGRESS")
		job["CreatedAt"] = mpkgFirstNonEmpty(mpkgStringAny(job, "CreatedAt"), now)
		job["ModifiedAt"] = now
		return mpkgCloneMap(job)

	case "GetHarvestJob":
		return mpkgCloneMap(s.ensureHarvestJobLocked(channelGroupName, channelName, originEndpointName, harvestJobName))

	case "ListHarvestJobs":
		return map[string]any{
			"Items":     s.listHarvestJobsLocked(channelGroupName),
			"NextToken": "",
		}

	case "CancelHarvestJob":
		job := s.ensureHarvestJobLocked(channelGroupName, channelName, originEndpointName, harvestJobName)
		job["Status"] = "CANCELLED"
		job["ModifiedAt"] = now
		return mpkgCloneMap(job)

	case "PutChannelPolicy":
		key := mpkgChannelKey(channelGroupName, channelName)
		s.channelPolicies[key] = mpkgFirstNonEmpty(mpkgStringAny(payload, "Policy", "policy"), `{"Version":"2012-10-17","Statement":[]}`)
		return map[string]any{}

	case "GetChannelPolicy":
		key := mpkgChannelKey(channelGroupName, channelName)
		if _, ok := s.channelPolicies[key]; !ok {
			s.channelPolicies[key] = `{"Version":"2012-10-17","Statement":[]}`
		}
		return map[string]any{
			"Policy": s.channelPolicies[key],
		}

	case "DeleteChannelPolicy":
		delete(s.channelPolicies, mpkgChannelKey(channelGroupName, channelName))
		return map[string]any{}

	case "PutOriginEndpointPolicy":
		key := mpkgEndpointKey(channelGroupName, channelName, originEndpointName)
		s.endpointPolicy[key] = mpkgFirstNonEmpty(mpkgStringAny(payload, "Policy", "policy"), `{"Version":"2012-10-17","Statement":[]}`)
		return map[string]any{}

	case "GetOriginEndpointPolicy":
		key := mpkgEndpointKey(channelGroupName, channelName, originEndpointName)
		if _, ok := s.endpointPolicy[key]; !ok {
			s.endpointPolicy[key] = `{"Version":"2012-10-17","Statement":[]}`
		}
		return map[string]any{
			"Policy": s.endpointPolicy[key],
		}

	case "DeleteOriginEndpointPolicy":
		delete(s.endpointPolicy, mpkgEndpointKey(channelGroupName, channelName, originEndpointName))
		return map[string]any{}

	case "ResetChannelState":
		ch := s.ensureChannelLocked(channelGroupName, channelName)
		ch["State"] = "RESET"
		ch["ResetAt"] = now
		ch["ModifiedAt"] = now
		return mpkgCloneMap(ch)

	case "ResetOriginEndpointState":
		ep := s.ensureOriginEndpointLocked(channelGroupName, channelName, originEndpointName)
		ep["State"] = "RESET"
		ep["ResetAt"] = now
		ep["ModifiedAt"] = now
		return mpkgCloneMap(ep)

	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for k, v := range mpkgStringMapAny(payload, "Tags", "tags") {
			tags[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range mpkgStringSliceAny(payload, "TagKeys", "tagKeys") {
			delete(tags, key)
		}
		for _, key := range query["tagKeys"] {
			key = strings.TrimSpace(key)
			if key != "" {
				delete(tags, key)
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{
			"Tags": mpkgCloneStringMap(s.ensureTagsLocked(resourceARN)),
		}
	}

	return map[string]any{}
}

func (s *mediaPackageStore) nextGroupIDLocked() int64 {
	id := s.nextGroupID
	s.nextGroupID++
	return id
}

func (s *mediaPackageStore) nextChannelIDLocked() int64 {
	id := s.nextChannelID
	s.nextChannelID++
	return id
}

func (s *mediaPackageStore) nextEndpointIDLocked() int64 {
	id := s.nextEndpointID
	s.nextEndpointID++
	return id
}

func (s *mediaPackageStore) nextHarvestIDLocked() int64 {
	id := s.nextHarvestID
	s.nextHarvestID++
	return id
}

func (s *mediaPackageStore) ensureChannelGroupLocked(groupName string) map[string]any {
	groupName = mpkgFirstNonEmpty(strings.TrimSpace(groupName), "channel-group-00000001")
	if v, ok := s.channelGroups[groupName]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := map[string]any{
		"ChannelGroupName": groupName,
		"Arn":              mpkgChannelGroupARN(groupName),
		"DomainName":       fmt.Sprintf("%s.mediapackagev2.us-east-1.stackyard.local", groupName),
		"CreatedAt":        now,
		"ModifiedAt":       now,
	}
	s.channelGroups[groupName] = item
	return item
}

func (s *mediaPackageStore) ensureChannelLocked(groupName, channelName string) map[string]any {
	_ = s.ensureChannelGroupLocked(groupName)
	key := mpkgChannelKey(groupName, channelName)
	if v, ok := s.channels[key]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := map[string]any{
		"ChannelGroupName": groupName,
		"ChannelName":      channelName,
		"Arn":              mpkgChannelARN(groupName, channelName),
		"InputType":        "HLS",
		"CreatedAt":        now,
		"ModifiedAt":       now,
	}
	s.channels[key] = item
	return item
}

func (s *mediaPackageStore) ensureOriginEndpointLocked(groupName, channelName, endpointName string) map[string]any {
	_ = s.ensureChannelLocked(groupName, channelName)
	key := mpkgEndpointKey(groupName, channelName, endpointName)
	if v, ok := s.originEndpoints[key]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := map[string]any{
		"ChannelGroupName":   groupName,
		"ChannelName":        channelName,
		"OriginEndpointName": endpointName,
		"Arn":                mpkgOriginEndpointARN(groupName, channelName, endpointName),
		"ContainerType":      "TS",
		"CreatedAt":          now,
		"ModifiedAt":         now,
	}
	s.originEndpoints[key] = item
	return item
}

func (s *mediaPackageStore) ensureHarvestJobLocked(groupName, channelName, endpointName, harvestName string) map[string]any {
	_ = s.ensureOriginEndpointLocked(groupName, channelName, endpointName)
	key := mpkgHarvestKey(groupName, channelName, endpointName, harvestName)
	if v, ok := s.harvestJobs[key]; ok {
		return v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := map[string]any{
		"ChannelGroupName":   groupName,
		"ChannelName":        channelName,
		"OriginEndpointName": endpointName,
		"HarvestJobName":     harvestName,
		"Arn":                mpkgHarvestJobARN(groupName, channelName, endpointName, harvestName),
		"Status":             "IN_PROGRESS",
		"CreatedAt":          now,
		"ModifiedAt":         now,
	}
	s.harvestJobs[key] = item
	return item
}

func (s *mediaPackageStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = mpkgFirstNonEmpty(strings.TrimSpace(resourceARN), mpkgChannelGroupARN("channel-group-00000001"))
	if v, ok := s.tags[resourceARN]; ok {
		return v
	}
	m := map[string]string{}
	s.tags[resourceARN] = m
	return m
}

func (s *mediaPackageStore) listChannelGroupsLocked() []any {
	keys := make([]string, 0, len(s.channelGroups))
	for k := range s.channelGroups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, mpkgCloneMap(s.channelGroups[k]))
	}
	return out
}

func (s *mediaPackageStore) listChannelsLocked(groupName string) []any {
	prefix := strings.TrimSpace(groupName) + "/"
	keys := make([]string, 0, len(s.channels))
	for k := range s.channels {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, mpkgCloneMap(s.channels[k]))
	}
	return out
}

func (s *mediaPackageStore) listOriginEndpointsLocked(groupName, channelName string) []any {
	prefix := mpkgChannelKey(groupName, channelName) + "/"
	keys := make([]string, 0, len(s.originEndpoints))
	for k := range s.originEndpoints {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, mpkgCloneMap(s.originEndpoints[k]))
	}
	return out
}

func (s *mediaPackageStore) listHarvestJobsLocked(groupName string) []any {
	prefix := strings.TrimSpace(groupName) + "/"
	keys := make([]string, 0, len(s.harvestJobs))
	for k := range s.harvestJobs {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, mpkgCloneMap(s.harvestJobs[k]))
	}
	return out
}

func mpkgChannelKey(groupName, channelName string) string {
	return fmt.Sprintf("%s/%s", strings.TrimSpace(groupName), strings.TrimSpace(channelName))
}

func mpkgEndpointKey(groupName, channelName, endpointName string) string {
	return fmt.Sprintf("%s/%s", mpkgChannelKey(groupName, channelName), strings.TrimSpace(endpointName))
}

func mpkgHarvestKey(groupName, channelName, endpointName, harvestName string) string {
	return fmt.Sprintf("%s/%s", mpkgEndpointKey(groupName, channelName, endpointName), strings.TrimSpace(harvestName))
}

func mpkgChannelGroupARN(groupName string) string {
	return fmt.Sprintf("arn:aws:mediapackagev2:us-east-1:123456789012:channelGroup/%s", strings.TrimSpace(groupName))
}

func mpkgChannelARN(groupName, channelName string) string {
	return fmt.Sprintf("arn:aws:mediapackagev2:us-east-1:123456789012:channelGroup/%s/channel/%s", strings.TrimSpace(groupName), strings.TrimSpace(channelName))
}

func mpkgOriginEndpointARN(groupName, channelName, endpointName string) string {
	return fmt.Sprintf("arn:aws:mediapackagev2:us-east-1:123456789012:channelGroup/%s/channel/%s/originEndpoint/%s", strings.TrimSpace(groupName), strings.TrimSpace(channelName), strings.TrimSpace(endpointName))
}

func mpkgHarvestJobARN(groupName, channelName, endpointName, harvestName string) string {
	return fmt.Sprintf("arn:aws:mediapackagev2:us-east-1:123456789012:channelGroup/%s/channel/%s/originEndpoint/%s/harvestJob/%s", strings.TrimSpace(groupName), strings.TrimSpace(channelName), strings.TrimSpace(endpointName), strings.TrimSpace(harvestName))
}

func mpkgPathParam(params map[string]string, key string) string {
	if len(params) == 0 {
		return ""
	}
	return strings.TrimSpace(params[key])
}

func mpkgStringAny(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if m == nil {
			continue
		}
		if v, ok := m[key]; ok {
			switch t := v.(type) {
			case string:
				s := strings.TrimSpace(t)
				if s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func mpkgStringMapAny(m map[string]any, keys ...string) map[string]string {
	out := map[string]string{}
	if m == nil {
		return out
	}
	for _, key := range keys {
		raw, ok := m[key]
		if !ok || raw == nil {
			continue
		}
		if data, ok := raw.(map[string]any); ok {
			for k, v := range data {
				if ks := strings.TrimSpace(k); ks != "" {
					if vs, ok := v.(string); ok {
						out[ks] = strings.TrimSpace(vs)
					}
				}
			}
		}
	}
	return out
}

func mpkgStringSliceAny(m map[string]any, keys ...string) []string {
	out := []string{}
	if m == nil {
		return out
	}
	for _, key := range keys {
		raw, ok := m[key]
		if !ok || raw == nil {
			continue
		}
		if arr, ok := raw.([]any); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						out = append(out, s)
					}
				}
			}
		}
	}
	return out
}

func mpkgFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func mpkgCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mpkgCloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
