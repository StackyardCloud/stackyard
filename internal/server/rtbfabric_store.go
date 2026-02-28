package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type rtbFabricStore struct {
	mu sync.Mutex

	nextID int64

	requesterGateways map[string]map[string]any
	responderGateways map[string]map[string]any
	links             map[string]map[string]any
	inboundLinks      map[string]map[string]any
	outboundLinks     map[string]map[string]any
	tags              map[string]map[string]string
}

func newRTBFabricStore() *rtbFabricStore {
	now := time.Now().UTC().Format(time.RFC3339)
	gatewayID := "stackyard-gateway"
	linkID := "stackyard-link"
	resourceARN := "arn:aws:rtb-fabric:us-east-1:123456789012:requester-gateway/stackyard-gateway"

	s := &rtbFabricStore{
		nextID: 2,
		requesterGateways: map[string]map[string]any{
			gatewayID: {
				"gatewayId":          gatewayID,
				"gatewayArn":         resourceARN,
				"name":               "stackyard-requester-gateway",
				"state":              "ACTIVE",
				"customerProvidedId": "stackyard-requester",
				"createdAt":          now,
				"updatedAt":          now,
			},
		},
		responderGateways: map[string]map[string]any{
			gatewayID: {
				"gatewayId":          gatewayID,
				"gatewayArn":         "arn:aws:rtb-fabric:us-east-1:123456789012:responder-gateway/stackyard-gateway",
				"name":               "stackyard-responder-gateway",
				"state":              "ACTIVE",
				"customerProvidedId": "stackyard-responder",
				"createdAt":          now,
				"updatedAt":          now,
			},
		},
		links: map[string]map[string]any{
			linkID: {
				"linkId":             linkID,
				"gatewayId":          gatewayID,
				"state":              "PENDING",
				"requesterGatewayId": gatewayID,
				"responderGatewayId": gatewayID,
				"name":               "stackyard-link",
				"createdAt":          now,
				"updatedAt":          now,
			},
		},
		inboundLinks:  map[string]map[string]any{},
		outboundLinks: map[string]map[string]any{},
		tags: map[string]map[string]string{
			resourceARN: {"seed": "true"},
		},
	}

	s.inboundLinks[linkID] = map[string]any{
		"linkId":    linkID,
		"gatewayId": gatewayID,
		"name":      "stackyard-inbound-link",
		"state":     "ACTIVE",
		"createdAt": now,
		"updatedAt": now,
	}
	s.outboundLinks[linkID] = map[string]any{
		"linkId":    linkID,
		"gatewayId": gatewayID,
		"name":      "stackyard-outbound-link",
		"state":     "ACTIVE",
		"createdAt": now,
		"updatedAt": now,
	}

	return s
}

func (s *rtbFabricStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.syncPayloadWithQuery(payload, query)

	switch action {
	case "CreateRequesterGateway":
		gw := s.upsertRequesterGateway(payload, pathParams)
		return map[string]any{"gateway": rtbFabricCloneAnyMap(gw)}
	case "GetRequesterGateway", "UpdateRequesterGateway":
		gw := s.upsertRequesterGateway(payload, pathParams)
		if action == "UpdateRequesterGateway" {
			rtbFabricPatchMap(gw, payload)
			gw["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		}
		return map[string]any{"gateway": rtbFabricCloneAnyMap(gw)}
	case "DeleteRequesterGateway":
		id := s.resolveGatewayID(payload, pathParams)
		delete(s.requesterGateways, id)
		return map[string]any{}
	case "ListRequesterGateways":
		return map[string]any{"gateways": s.listSortedMaps(s.requesterGateways), "nextToken": ""}

	case "CreateResponderGateway":
		gw := s.upsertResponderGateway(payload, pathParams)
		return map[string]any{"gateway": rtbFabricCloneAnyMap(gw)}
	case "GetResponderGateway", "UpdateResponderGateway":
		gw := s.upsertResponderGateway(payload, pathParams)
		if action == "UpdateResponderGateway" {
			rtbFabricPatchMap(gw, payload)
			gw["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		}
		return map[string]any{"gateway": rtbFabricCloneAnyMap(gw)}
	case "DeleteResponderGateway":
		id := s.resolveGatewayID(payload, pathParams)
		delete(s.responderGateways, id)
		return map[string]any{}
	case "ListResponderGateways":
		return map[string]any{"gateways": s.listSortedMaps(s.responderGateways), "nextToken": ""}

	case "CreateLink":
		link := s.upsertLink(payload, pathParams)
		link["state"] = "PENDING"
		return map[string]any{"link": rtbFabricCloneAnyMap(link)}
	case "GetLink", "UpdateLink":
		link := s.upsertLink(payload, pathParams)
		if action == "UpdateLink" {
			rtbFabricPatchMap(link, payload)
			link["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		}
		return map[string]any{"link": rtbFabricCloneAnyMap(link)}
	case "AcceptLink":
		link := s.upsertLink(payload, pathParams)
		link["state"] = "ACCEPTED"
		link["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"link": rtbFabricCloneAnyMap(link)}
	case "RejectLink":
		link := s.upsertLink(payload, pathParams)
		link["state"] = "REJECTED"
		link["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"link": rtbFabricCloneAnyMap(link)}
	case "UpdateLinkModuleFlow":
		link := s.upsertLink(payload, pathParams)
		if moduleFlow, ok := payload["moduleFlow"]; ok {
			link["moduleFlow"] = rtbFabricCloneAny(moduleFlow)
		}
		link["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"link": rtbFabricCloneAnyMap(link)}
	case "DeleteLink":
		id := s.resolveLinkID(payload, pathParams)
		delete(s.links, id)
		return map[string]any{}
	case "ListLinks":
		gatewayID := s.resolveGatewayID(payload, pathParams)
		items := make([]any, 0, len(s.links))
		for _, candidate := range s.listSortedMaps(s.links) {
			m, _ := candidate.(map[string]any)
			if gatewayID == "" || strings.EqualFold(rtbFabricString(m["gatewayId"]), gatewayID) {
				items = append(items, m)
			}
		}
		return map[string]any{"links": items, "nextToken": ""}

	case "CreateInboundExternalLink":
		link := s.upsertExternalLink(s.inboundLinks, payload, pathParams)
		return map[string]any{"link": rtbFabricCloneAnyMap(link)}
	case "GetInboundExternalLink":
		link := s.upsertExternalLink(s.inboundLinks, payload, pathParams)
		return map[string]any{"link": rtbFabricCloneAnyMap(link)}
	case "DeleteInboundExternalLink":
		delete(s.inboundLinks, s.resolveLinkID(payload, pathParams))
		return map[string]any{}

	case "CreateOutboundExternalLink":
		link := s.upsertExternalLink(s.outboundLinks, payload, pathParams)
		return map[string]any{"link": rtbFabricCloneAnyMap(link)}
	case "GetOutboundExternalLink":
		link := s.upsertExternalLink(s.outboundLinks, payload, pathParams)
		return map[string]any{"link": rtbFabricCloneAnyMap(link)}
	case "DeleteOutboundExternalLink":
		delete(s.outboundLinks, s.resolveLinkID(payload, pathParams))
		return map[string]any{}

	case "TagResource":
		resourceARN := s.resolveResourceARN(payload, pathParams)
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		for k, v := range rtbFabricReadTags(payload) {
			s.tags[resourceARN][k] = v
		}
		return map[string]any{}
	case "UntagResource":
		resourceARN := s.resolveResourceARN(payload, pathParams)
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		for _, k := range rtbFabricReadTagKeys(payload) {
			delete(s.tags[resourceARN], k)
		}
		return map[string]any{}
	case "ListTagsForResource":
		resourceARN := s.resolveResourceARN(payload, pathParams)
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		return map[string]any{"tags": rtbFabricCloneTags(s.tags[resourceARN])}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"nextToken": ""}
	}
	return map[string]any{}
}

func (s *rtbFabricStore) upsertRequesterGateway(payload map[string]any, pathParams map[string]string) map[string]any {
	id := s.resolveGatewayID(payload, pathParams)
	gw := s.requesterGateways[id]
	if gw == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		gw = map[string]any{
			"gatewayId":          id,
			"gatewayArn":         fmt.Sprintf("arn:aws:rtb-fabric:us-east-1:123456789012:requester-gateway/%s", id),
			"name":               id,
			"state":              "ACTIVE",
			"customerProvidedId": id,
			"createdAt":          now,
			"updatedAt":          now,
		}
		s.requesterGateways[id] = gw
	}
	return gw
}

func (s *rtbFabricStore) upsertResponderGateway(payload map[string]any, pathParams map[string]string) map[string]any {
	id := s.resolveGatewayID(payload, pathParams)
	gw := s.responderGateways[id]
	if gw == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		gw = map[string]any{
			"gatewayId":          id,
			"gatewayArn":         fmt.Sprintf("arn:aws:rtb-fabric:us-east-1:123456789012:responder-gateway/%s", id),
			"name":               id,
			"state":              "ACTIVE",
			"customerProvidedId": id,
			"createdAt":          now,
			"updatedAt":          now,
		}
		s.responderGateways[id] = gw
	}
	return gw
}

func (s *rtbFabricStore) upsertLink(payload map[string]any, pathParams map[string]string) map[string]any {
	linkID := s.resolveLinkID(payload, pathParams)
	link := s.links[linkID]
	if link == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		gatewayID := s.resolveGatewayID(payload, pathParams)
		link = map[string]any{
			"linkId":             linkID,
			"gatewayId":          gatewayID,
			"requesterGatewayId": gatewayID,
			"responderGatewayId": gatewayID,
			"name":               linkID,
			"state":              "PENDING",
			"createdAt":          now,
			"updatedAt":          now,
		}
		s.links[linkID] = link
	}
	return link
}

func (s *rtbFabricStore) upsertExternalLink(index map[string]map[string]any, payload map[string]any, pathParams map[string]string) map[string]any {
	linkID := s.resolveLinkID(payload, pathParams)
	link := index[linkID]
	if link == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		gatewayID := s.resolveGatewayID(payload, pathParams)
		link = map[string]any{
			"linkId":    linkID,
			"gatewayId": gatewayID,
			"name":      linkID,
			"state":     "ACTIVE",
			"createdAt": now,
			"updatedAt": now,
		}
		index[linkID] = link
	}
	return link
}

func (s *rtbFabricStore) resolveGatewayID(payload map[string]any, pathParams map[string]string) string {
	if id := strings.TrimSpace(pathParams["gatewayId"]); id != "" {
		return id
	}
	for _, key := range []string{"gatewayId", "GatewayId", "requesterGatewayId", "responderGatewayId"} {
		for existing, value := range payload {
			if strings.EqualFold(existing, key) {
				if id := strings.TrimSpace(rtbFabricString(value)); id != "" {
					return id
				}
			}
		}
	}
	return "stackyard-gateway"
}

func (s *rtbFabricStore) resolveLinkID(payload map[string]any, pathParams map[string]string) string {
	if id := strings.TrimSpace(pathParams["linkId"]); id != "" {
		return id
	}
	for _, key := range []string{"linkId", "LinkId"} {
		for existing, value := range payload {
			if strings.EqualFold(existing, key) {
				if id := strings.TrimSpace(rtbFabricString(value)); id != "" {
					return id
				}
			}
		}
	}
	return "stackyard-link"
}

func (s *rtbFabricStore) resolveResourceARN(payload map[string]any, pathParams map[string]string) string {
	if arn := strings.TrimSpace(pathParams["resourceArn"]); arn != "" {
		return arn
	}
	for _, key := range []string{"resourceArn", "ResourceArn", "arn", "Arn"} {
		for existing, value := range payload {
			if strings.EqualFold(existing, key) {
				if arn := strings.TrimSpace(rtbFabricString(value)); arn != "" {
					return arn
				}
			}
		}
	}
	return "arn:aws:rtb-fabric:us-east-1:123456789012:requester-gateway/stackyard-gateway"
}

func (s *rtbFabricStore) listSortedMaps(items map[string]map[string]any) []any {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, rtbFabricCloneAnyMap(items[key]))
	}
	return out
}

func (s *rtbFabricStore) syncPayloadWithQuery(payload map[string]any, query url.Values) {
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if _, exists := payload[key]; !exists {
			payload[key] = values[0]
		}
	}
}

func rtbFabricReadTags(payload map[string]any) map[string]string {
	result := map[string]string{}
	for _, key := range []string{"tags", "Tags"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch tags := raw.(type) {
		case map[string]any:
			for k, v := range tags {
				k = strings.TrimSpace(k)
				if k == "" {
					continue
				}
				result[k] = strings.TrimSpace(rtbFabricString(v))
			}
		case map[string]string:
			for k, v := range tags {
				k = strings.TrimSpace(k)
				if k == "" {
					continue
				}
				result[k] = strings.TrimSpace(v)
			}
		}
	}
	return result
}

func rtbFabricReadTagKeys(payload map[string]any) []string {
	for _, key := range []string{"tagKeys", "TagKeys", "keys", "Keys"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch keys := raw.(type) {
		case []any:
			out := make([]string, 0, len(keys))
			for _, item := range keys {
				if k := strings.TrimSpace(rtbFabricString(item)); k != "" {
					out = append(out, k)
				}
			}
			if len(out) > 0 {
				return out
			}
		case []string:
			out := make([]string, 0, len(keys))
			for _, item := range keys {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
			if len(out) > 0 {
				return out
			}
		case string:
			if strings.TrimSpace(keys) != "" {
				return []string{strings.TrimSpace(keys)}
			}
		}
	}
	return []string{}
}

func rtbFabricPatchMap(dst map[string]any, src map[string]any) {
	for key, value := range src {
		dst[key] = rtbFabricCloneAny(value)
	}
}

func rtbFabricString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func rtbFabricCloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = rtbFabricCloneAny(v)
	}
	return out
}

func rtbFabricCloneAny(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return rtbFabricCloneAnyMap(v)
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = rtbFabricCloneAny(v[i])
		}
		return out
	default:
		return v
	}
}

func rtbFabricCloneTags(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
