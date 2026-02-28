package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type oamStore struct {
	mu sync.Mutex

	nextSink int64
	nextLink int64

	sinks        map[string]map[string]any
	links        map[string]map[string]any
	sinkPolicies map[string]string
	tags         map[string]map[string]string
}

func newOAMStore() *oamStore {
	s := &oamStore{
		nextSink:     2,
		nextLink:     2,
		sinks:        map[string]map[string]any{},
		links:        map[string]map[string]any{},
		sinkPolicies: map[string]string{},
		tags:         map[string]map[string]string{},
	}

	sink := s.ensureSinkLocked("stackyard-sink")
	link := s.ensureLinkLocked("stackyard-link", oamSinkARN(sink))
	s.tags[oamSinkARN(sink)] = map[string]string{"stackyard": "true"}
	s.tags[oamLinkARN(link)] = map[string]string{"stackyard": "true"}
	s.sinkPolicies[oamSinkKeyByIdentifier(oamSinkARN(sink))] = `{"Version":"2012-10-17","Statement":[]}`
	return s
}

func (s *oamStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateSink":
		id := oamDefaultStringAny(payload, "Name", "")
		if id == "" {
			id = fmt.Sprintf("stackyard-sink-%06d", s.nextSinkIDLocked())
		}
		sink := s.ensureSinkLocked(id)
		s.applySinkPayloadLocked(sink, payload)
		arn := oamSinkARN(sink)
		s.ensureTagsLocked(arn)
		for k, v := range oamMapString(payload, "Tags") {
			s.tags[arn][k] = v
		}
		return map[string]any{"Sink": oamCloneMap(sink)}

	case "GetSink":
		identifier := oamDefaultStringAny(payload, "Identifier", "")
		if identifier == "" {
			identifier = oamDefaultStringAny(payload, "SinkIdentifier", "")
		}
		sink := s.ensureSinkByIdentifierLocked(identifier)
		return map[string]any{"Sink": oamCloneMap(sink)}

	case "ListSinks":
		items := make([]any, 0, len(s.sinks))
		keys := make([]string, 0, len(s.sinks))
		for key := range s.sinks {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			items = append(items, oamCloneMap(s.sinks[key]))
		}
		return map[string]any{"Items": items, "NextToken": ""}

	case "DeleteSink":
		identifier := oamDefaultStringAny(payload, "Identifier", "")
		if identifier == "" {
			identifier = oamDefaultStringAny(payload, "SinkIdentifier", "")
		}
		key := oamSinkKeyByIdentifier(identifier)
		if key == "" {
			key = "stackyard-sink"
		}
		if sink := s.sinks[key]; sink != nil {
			delete(s.tags, oamSinkARN(sink))
		}
		delete(s.sinks, key)
		delete(s.sinkPolicies, key)
		for linkKey, link := range s.links {
			if oamDefaultStringAny(link, "SinkArn", "") == oamSinkARNByID(key) {
				delete(s.tags, oamLinkARN(link))
				delete(s.links, linkKey)
			}
		}
		return map[string]any{}

	case "CreateLink":
		sinkIdentifier := oamDefaultStringAny(payload, "SinkIdentifier", "")
		if sinkIdentifier == "" {
			sinkIdentifier = oamSinkARNByID("stackyard-sink")
		}
		sink := s.ensureSinkByIdentifierLocked(sinkIdentifier)
		linkID := oamDefaultStringAny(payload, "LabelTemplate", "")
		if linkID == "" {
			linkID = fmt.Sprintf("stackyard-link-%06d", s.nextLinkIDLocked())
		}
		link := s.ensureLinkLocked(linkID, oamSinkARN(sink))
		s.applyLinkPayloadLocked(link, payload)
		arn := oamLinkARN(link)
		s.ensureTagsLocked(arn)
		for k, v := range oamMapString(payload, "Tags") {
			s.tags[arn][k] = v
		}
		return map[string]any{"Link": oamCloneMap(link)}

	case "GetLink":
		identifier := oamDefaultStringAny(payload, "Identifier", "")
		link := s.ensureLinkByIdentifierLocked(identifier)
		return map[string]any{"Link": oamCloneMap(link)}

	case "ListLinks":
		items := make([]any, 0, len(s.links))
		keys := make([]string, 0, len(s.links))
		for key := range s.links {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			items = append(items, oamCloneMap(s.links[key]))
		}
		return map[string]any{"Items": items, "NextToken": ""}

	case "ListAttachedLinks":
		sinkIdentifier := oamDefaultStringAny(payload, "SinkIdentifier", "")
		if sinkIdentifier == "" {
			sinkIdentifier = oamSinkARNByID("stackyard-sink")
		}
		sink := s.ensureSinkByIdentifierLocked(sinkIdentifier)
		sinkArn := oamSinkARN(sink)
		items := make([]any, 0, len(s.links))
		keys := make([]string, 0, len(s.links))
		for key := range s.links {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			link := s.links[key]
			if oamDefaultStringAny(link, "SinkArn", "") != sinkArn {
				continue
			}
			items = append(items, oamCloneMap(link))
		}
		return map[string]any{"Items": items, "NextToken": ""}

	case "UpdateLink":
		identifier := oamDefaultStringAny(payload, "Identifier", "")
		link := s.ensureLinkByIdentifierLocked(identifier)
		s.applyLinkPayloadLocked(link, payload)
		return map[string]any{"Link": oamCloneMap(link)}

	case "DeleteLink":
		identifier := oamDefaultStringAny(payload, "Identifier", "")
		key := oamLinkKeyByIdentifier(identifier)
		if key == "" {
			key = "stackyard-link"
		}
		if link := s.links[key]; link != nil {
			delete(s.tags, oamLinkARN(link))
		}
		delete(s.links, key)
		return map[string]any{}

	case "GetSinkPolicy":
		identifier := oamDefaultStringAny(payload, "SinkIdentifier", "")
		if identifier == "" {
			identifier = oamSinkARNByID("stackyard-sink")
		}
		sink := s.ensureSinkByIdentifierLocked(identifier)
		key := oamSinkKeyByIdentifier(oamSinkARN(sink))
		policy := strings.TrimSpace(s.sinkPolicies[key])
		if policy == "" {
			policy = `{"Version":"2012-10-17","Statement":[]}`
			s.sinkPolicies[key] = policy
		}
		return map[string]any{
			"SinkIdentifier": oamSinkARN(sink),
			"SinkPolicy":     policy,
		}

	case "PutSinkPolicy":
		identifier := oamDefaultStringAny(payload, "SinkIdentifier", "")
		if identifier == "" {
			identifier = oamSinkARNByID("stackyard-sink")
		}
		sink := s.ensureSinkByIdentifierLocked(identifier)
		key := oamSinkKeyByIdentifier(oamSinkARN(sink))
		policy := oamDefaultStringAny(payload, "Policy", `{"Version":"2012-10-17","Statement":[]}`)
		s.sinkPolicies[key] = policy
		return map[string]any{
			"SinkIdentifier": oamSinkARN(sink),
			"SinkPolicy":     policy,
		}

	case "ListTagsForResource":
		resourceARN := oamDefaultString(pathParams, "ResourceArn", oamSinkARNByID("stackyard-sink"))
		return map[string]any{"Tags": oamCloneStringMap(s.tags[resourceARN])}

	case "TagResource":
		resourceARN := oamDefaultString(pathParams, "ResourceArn", oamSinkARNByID("stackyard-sink"))
		s.ensureTagsLocked(resourceARN)
		for k, v := range oamMapString(payload, "Tags") {
			s.tags[resourceARN][k] = v
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := oamDefaultString(pathParams, "ResourceArn", oamSinkARNByID("stackyard-sink"))
		for _, key := range oamStringSlice(payload, "TagKeys") {
			if s.tags[resourceARN] != nil {
				delete(s.tags[resourceARN], key)
			}
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *oamStore) ensureSinkLocked(identifier string) map[string]any {
	key := oamSinkKeyByIdentifier(identifier)
	if key == "" {
		key = "stackyard-sink"
	}
	if sink := s.sinks[key]; sink != nil {
		return sink
	}

	sink := map[string]any{
		"Id":        key,
		"Name":      key,
		"Arn":       oamSinkARNByID(key),
		"CreatedAt": time.Now().UTC(),
	}
	s.sinks[key] = sink
	return sink
}

func (s *oamStore) ensureSinkByIdentifierLocked(identifier string) map[string]any {
	return s.ensureSinkLocked(identifier)
}

func (s *oamStore) ensureLinkLocked(identifier, sinkArn string) map[string]any {
	key := oamLinkKeyByIdentifier(identifier)
	if key == "" {
		key = "stackyard-link"
	}
	if link := s.links[key]; link != nil {
		return link
	}
	if strings.TrimSpace(sinkArn) == "" {
		sinkArn = oamSinkARNByID("stackyard-sink")
	}

	link := map[string]any{
		"Id":            key,
		"Arn":           oamLinkARNByID(key),
		"SinkArn":       sinkArn,
		"LabelTemplate": key,
		"ResourceTypes": []any{"AWS::CloudWatch::Metric"},
		"CreatedAt":     time.Now().UTC(),
	}
	s.links[key] = link
	return link
}

func (s *oamStore) ensureLinkByIdentifierLocked(identifier string) map[string]any {
	key := oamLinkKeyByIdentifier(identifier)
	if key == "" {
		key = "stackyard-link"
	}
	if link := s.links[key]; link != nil {
		return link
	}
	sink := s.ensureSinkLocked("stackyard-sink")
	return s.ensureLinkLocked(key, oamSinkARN(sink))
}

func (s *oamStore) applySinkPayloadLocked(sink map[string]any, payload map[string]any) {
	if name := oamDefaultStringAny(payload, "Name", ""); name != "" {
		key := oamSinkKeyByIdentifier(name)
		sink["Name"] = name
		sink["Id"] = key
		sink["Arn"] = oamSinkARNByID(key)
	}
	for key, value := range payload {
		if strings.EqualFold(strings.TrimSpace(key), "Tags") {
			continue
		}
		sink[key] = value
	}
}

func (s *oamStore) applyLinkPayloadLocked(link map[string]any, payload map[string]any) {
	if label := oamDefaultStringAny(payload, "LabelTemplate", ""); label != "" {
		link["LabelTemplate"] = label
	}
	if sinkID := oamDefaultStringAny(payload, "SinkIdentifier", ""); sinkID != "" {
		sink := s.ensureSinkByIdentifierLocked(sinkID)
		link["SinkArn"] = oamSinkARN(sink)
	}
	for key, value := range payload {
		if strings.EqualFold(strings.TrimSpace(key), "Identifier") ||
			strings.EqualFold(strings.TrimSpace(key), "Tags") {
			continue
		}
		link[key] = value
	}
}

func (s *oamStore) ensureTagsLocked(resourceARN string) {
	if s.tags[resourceARN] == nil {
		s.tags[resourceARN] = map[string]string{}
	}
}

func (s *oamStore) nextSinkIDLocked() int64 {
	id := s.nextSink
	s.nextSink++
	return id
}

func (s *oamStore) nextLinkIDLocked() int64 {
	id := s.nextLink
	s.nextLink++
	return id
}

func oamSinkARN(sink map[string]any) string {
	return oamSinkARNByID(oamDefaultStringAny(sink, "Id", "stackyard-sink"))
}

func oamSinkARNByID(id string) string {
	key := oamSinkKeyByIdentifier(id)
	if strings.HasPrefix(strings.TrimSpace(id), "arn:") {
		return strings.TrimSpace(id)
	}
	return fmt.Sprintf("arn:aws:oam:us-east-1:123456789012:sink/%s", key)
}

func oamLinkARN(link map[string]any) string {
	return oamLinkARNByID(oamDefaultStringAny(link, "Id", "stackyard-link"))
}

func oamLinkARNByID(id string) string {
	key := oamLinkKeyByIdentifier(id)
	if strings.HasPrefix(strings.TrimSpace(id), "arn:") {
		return strings.TrimSpace(id)
	}
	return fmt.Sprintf("arn:aws:oam:us-east-1:123456789012:link/%s", key)
}

func oamSinkKeyByIdentifier(identifier string) string {
	value := strings.TrimSpace(identifier)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "arn:") {
		if idx := strings.LastIndex(value, "/"); idx >= 0 && idx+1 < len(value) {
			return strings.TrimSpace(value[idx+1:])
		}
		return value
	}
	return value
}

func oamLinkKeyByIdentifier(identifier string) string {
	value := strings.TrimSpace(identifier)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "arn:") {
		if idx := strings.LastIndex(value, "/"); idx >= 0 && idx+1 < len(value) {
			return strings.TrimSpace(value[idx+1:])
		}
		return value
	}
	return value
}

func oamDefaultString(values map[string]string, key, fallback string) string {
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

func oamDefaultStringAny(values map[string]any, key, fallback string) string {
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

func oamMapString(payload map[string]any, key string) map[string]string {
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		if raw, ok := v.(map[string]any); ok {
			out := map[string]string{}
			for rk, rv := range raw {
				if str, ok := rv.(string); ok {
					out[rk] = str
				}
			}
			return out
		}
		if raw, ok := v.(map[string]string); ok {
			return oamCloneStringMap(raw)
		}
	}
	return map[string]string{}
}

func oamStringSlice(payload map[string]any, key string) []string {
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

func oamCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = oamCloneAny(v)
	}
	return out
}

func oamCloneAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return oamCloneMap(t)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = oamCloneAny(item)
		}
		return out
	case map[string]string:
		return oamCloneStringMap(t)
	default:
		return t
	}
}

func oamCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
