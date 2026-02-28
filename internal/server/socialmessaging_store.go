package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type socialMessagingStore struct {
	mu sync.Mutex

	nextID int64

	wabas             map[string]map[string]any
	phoneNumbers      map[string]map[string]any
	media             map[string]map[string]any
	templates         map[string]map[string]any
	templateMedia     map[string]map[string]any
	templateLibrary   map[string]map[string]any
	messages          map[string]map[string]any
	eventDestinations map[string][]any
	tags              map[string]map[string]string
}

func newSocialMessagingStore() *socialMessagingStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &socialMessagingStore{
		nextID: 2,
		wabas: map[string]map[string]any{
			"waba-000001": {
				"id":                 "waba-000001",
				"arn":                socialMessagingWABAArn("waba-000001"),
				"status":             "ACTIVE",
				"name":               "stackyard-whatsapp-business-account",
				"registrationStatus": "COMPLETED",
				"createdTime":        now,
				"lastUpdatedTime":    now,
				"phoneNumberId":      "phone-000001",
				"phoneNumberDisplay": "+12065550100",
			},
		},
		phoneNumbers: map[string]map[string]any{
			"phone-000001": {
				"id":                 "phone-000001",
				"wabaId":             "waba-000001",
				"phoneNumber":        "+12065550100",
				"displayPhoneNumber": "+1 206-555-0100",
				"qualityRating":      "GREEN",
				"verifiedName":       "Stackyard",
				"createdTime":        now,
			},
		},
		media: map[string]map[string]any{},
		templates: map[string]map[string]any{
			"template-000001": {
				"id":              "template-000001",
				"name":            "stackyard-template",
				"language":        "en_US",
				"status":          "APPROVED",
				"category":        "UTILITY",
				"wabaId":          "waba-000001",
				"createdTime":     now,
				"lastUpdatedTime": now,
			},
		},
		templateMedia: map[string]map[string]any{},
		templateLibrary: map[string]map[string]any{
			"library-template-000001": {
				"id":          "library-template-000001",
				"name":        "order_update",
				"language":    "en_US",
				"category":    "UTILITY",
				"description": "Order update template",
			},
		},
		messages:          map[string]map[string]any{},
		eventDestinations: map[string][]any{"waba-000001": {}},
		tags: map[string]map[string]string{
			socialMessagingWABAArn("waba-000001"): {
				"stackyard": "true",
			},
		},
	}
	return s
}

func (s *socialMessagingStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := socialMessagingMergePayload(payload, pathParams, query)
	wabaID := socialMessagingGetString(ctx, "id", "waba-000001")
	resourceArn := socialMessagingGetString(ctx, "resourceArn", socialMessagingWABAArn(wabaID))
	templateID := socialMessagingGetString(ctx, "templateId", socialMessagingGetString(ctx, "id", "template-000001"))
	mediaID := socialMessagingGetString(ctx, "mediaId", socialMessagingGetString(ctx, "id", "media-000001"))
	now := time.Now().UTC().Format(time.RFC3339)

	s.ensureWABALocked(wabaID, now)
	s.ensureTagsLocked(resourceArn)

	switch action {
	case "AssociateWhatsAppBusinessAccount":
		waba := s.ensureWABALocked(wabaID, now)
		waba["status"] = "ACTIVE"
		waba["registrationStatus"] = "COMPLETED"
		waba["lastUpdatedTime"] = now
		return map[string]any{"id": wabaID, "arn": socialMessagingWABAArn(wabaID), "registrationStatus": "COMPLETED"}

	case "DisassociateWhatsAppBusinessAccount":
		if existing := s.wabas[wabaID]; existing != nil {
			existing["status"] = "DISASSOCIATED"
			existing["lastUpdatedTime"] = now
		}
		return map[string]any{}

	case "GetLinkedWhatsAppBusinessAccount":
		waba := s.ensureWABALocked(wabaID, now)
		return socialMessagingCloneMap(waba)

	case "GetLinkedWhatsAppBusinessAccountPhoneNumber":
		phoneID := socialMessagingGetString(ctx, "phoneNumberId", "phone-000001")
		phone := s.ensurePhoneNumberLocked(phoneID, wabaID, now)
		return socialMessagingCloneMap(phone)

	case "ListLinkedWhatsAppBusinessAccounts":
		return map[string]any{"linkedAccounts": socialMessagingListMaps(s.wabas), "nextToken": ""}

	case "PutWhatsAppBusinessAccountEventDestinations":
		destinations := socialMessagingExtractAnySlice(ctx, "eventDestinations")
		if len(destinations) == 0 {
			destinations = []any{map[string]any{"name": "stackyard-default", "enabled": true}}
		}
		s.eventDestinations[wabaID] = destinations
		return map[string]any{"id": wabaID, "eventDestinations": destinations}

	case "PostWhatsAppMessageMedia":
		id := socialMessagingGetString(ctx, "mediaId", fmt.Sprintf("media-%06d", s.nextIDLocked()))
		item := map[string]any{
			"mediaId":     id,
			"mimeType":    socialMessagingGetString(ctx, "mimeType", "image/png"),
			"fileName":    socialMessagingGetString(ctx, "fileName", "stackyard-media.bin"),
			"createdTime": now,
			"resourceArn": resourceArn,
		}
		s.media[id] = item
		return map[string]any{"mediaId": id}

	case "GetWhatsAppMessageMedia":
		item := s.ensureMediaLocked(mediaID, resourceArn, now)
		return map[string]any{
			"mediaId": mediaID,
			"presignedUrl": map[string]any{
				"url":        "https://stackyard.local/social-messaging/media/" + mediaID,
				"httpMethod": "GET",
				"expiresAt":  now,
			},
			"metadata": socialMessagingCloneMap(item),
		}

	case "DeleteWhatsAppMessageMedia":
		delete(s.media, mediaID)
		return map[string]any{}

	case "SendWhatsAppMessage":
		messageID := fmt.Sprintf("message-%06d", s.nextIDLocked())
		s.messages[messageID] = map[string]any{
			"messageId":   messageID,
			"status":      "QUEUED",
			"to":          socialMessagingGetString(ctx, "to", "+12065550101"),
			"from":        socialMessagingGetString(ctx, "from", "+12065550100"),
			"wabaId":      wabaID,
			"createdTime": now,
		}
		return map[string]any{"messageId": messageID, "messageStatus": "QUEUED"}

	case "CreateWhatsAppMessageTemplate":
		id := fmt.Sprintf("template-%06d", s.nextIDLocked())
		name := socialMessagingGetString(ctx, "name", "stackyard-template")
		template := map[string]any{
			"id":              id,
			"name":            name,
			"language":        socialMessagingGetString(ctx, "language", "en_US"),
			"status":          "PENDING",
			"category":        socialMessagingGetString(ctx, "category", "UTILITY"),
			"wabaId":          wabaID,
			"createdTime":     now,
			"lastUpdatedTime": now,
		}
		s.templates[id] = template
		return map[string]any{"id": id, "name": name, "status": "PENDING"}

	case "CreateWhatsAppMessageTemplateFromLibrary":
		id := fmt.Sprintf("template-%06d", s.nextIDLocked())
		name := socialMessagingGetString(ctx, "name", "stackyard-template-from-library")
		template := map[string]any{
			"id":              id,
			"name":            name,
			"language":        socialMessagingGetString(ctx, "language", "en_US"),
			"status":          "PENDING",
			"category":        socialMessagingGetString(ctx, "category", "UTILITY"),
			"wabaId":          wabaID,
			"source":          "LIBRARY",
			"createdTime":     now,
			"lastUpdatedTime": now,
		}
		s.templates[id] = template
		return map[string]any{"id": id, "name": name, "status": "PENDING"}

	case "CreateWhatsAppMessageTemplateMedia":
		id := fmt.Sprintf("template-media-%06d", s.nextIDLocked())
		s.templateMedia[id] = map[string]any{
			"id":          id,
			"templateId":  templateID,
			"mimeType":    socialMessagingGetString(ctx, "mimeType", "image/png"),
			"createdTime": now,
		}
		return map[string]any{"id": id, "templateId": templateID}

	case "GetWhatsAppMessageTemplate":
		tpl := s.ensureTemplateLocked(templateID, wabaID, now)
		return socialMessagingCloneMap(tpl)

	case "ListWhatsAppMessageTemplates":
		return map[string]any{"templates": socialMessagingListMaps(s.templates), "nextToken": ""}

	case "ListWhatsAppTemplateLibrary":
		return map[string]any{"templates": socialMessagingListMaps(s.templateLibrary), "nextToken": ""}

	case "UpdateWhatsAppMessageTemplate":
		tpl := s.ensureTemplateLocked(templateID, wabaID, now)
		if name := socialMessagingGetString(ctx, "name", ""); name != "" {
			tpl["name"] = name
		}
		tpl["status"] = "APPROVED"
		tpl["lastUpdatedTime"] = now
		return map[string]any{"id": tpl["id"], "name": tpl["name"], "status": tpl["status"]}

	case "DeleteWhatsAppMessageTemplate":
		deleteAll := strings.EqualFold(socialMessagingGetString(ctx, "deleteAllTemplates", "false"), "true")
		if deleteAll {
			s.templates = map[string]map[string]any{}
		} else {
			delete(s.templates, templateID)
		}
		return map[string]any{}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceArn)
		for key, value := range socialMessagingExtractTags(ctx) {
			tags[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		tags := s.ensureTagsLocked(resourceArn)
		for _, key := range socialMessagingExtractTagKeys(ctx) {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": socialMessagingTagsToList(s.ensureTagsLocked(resourceArn))}
	}

	return map[string]any{}
}

func (s *socialMessagingStore) nextIDLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *socialMessagingStore) ensureWABALocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "waba-000001"
	}
	if existing := s.wabas[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"id":                 id,
		"arn":                socialMessagingWABAArn(id),
		"status":             "ACTIVE",
		"name":               "stackyard-whatsapp-business-account",
		"registrationStatus": "COMPLETED",
		"createdTime":        now,
		"lastUpdatedTime":    now,
		"phoneNumberId":      "phone-000001",
		"phoneNumberDisplay": "+12065550100",
	}
	s.wabas[id] = item
	return item
}

func (s *socialMessagingStore) ensurePhoneNumberLocked(id, wabaID, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "phone-000001"
	}
	if existing := s.phoneNumbers[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"id":                 id,
		"wabaId":             wabaID,
		"phoneNumber":        "+12065550100",
		"displayPhoneNumber": "+1 206-555-0100",
		"qualityRating":      "GREEN",
		"verifiedName":       "Stackyard",
		"createdTime":        now,
	}
	s.phoneNumbers[id] = item
	return item
}

func (s *socialMessagingStore) ensureTemplateLocked(id, wabaID, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "template-000001"
	}
	if existing := s.templates[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"id":              id,
		"name":            "stackyard-template",
		"language":        "en_US",
		"status":          "APPROVED",
		"category":        "UTILITY",
		"wabaId":          wabaID,
		"createdTime":     now,
		"lastUpdatedTime": now,
	}
	s.templates[id] = item
	return item
}

func (s *socialMessagingStore) ensureMediaLocked(id, resourceArn, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "media-000001"
	}
	if existing := s.media[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"mediaId":     id,
		"mimeType":    "image/png",
		"fileName":    "stackyard-media.bin",
		"createdTime": now,
		"resourceArn": resourceArn,
	}
	s.media[id] = item
	return item
}

func (s *socialMessagingStore) ensureTagsLocked(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = socialMessagingWABAArn("waba-000001")
	}
	if existing := s.tags[resourceArn]; existing != nil {
		return existing
	}
	out := map[string]string{"stackyard": "true"}
	s.tags[resourceArn] = out
	return out
}

func socialMessagingMergePayload(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := socialMessagingCloneMap(payload)
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		out[key] = values[len(values)-1]
	}
	return out
}

func socialMessagingGetString(payload map[string]any, key, fallback string) string {
	for current, value := range payload {
		if strings.EqualFold(current, key) {
			out := strings.TrimSpace(fmt.Sprint(value))
			if out != "" && out != "<nil>" {
				return out
			}
		}
	}
	return fallback
}

func socialMessagingExtractAnySlice(payload map[string]any, key string) []any {
	for current, value := range payload {
		if !strings.EqualFold(current, key) {
			continue
		}
		switch typed := value.(type) {
		case []any:
			return typed
		case []string:
			out := make([]any, 0, len(typed))
			for _, item := range typed {
				out = append(out, item)
			}
			return out
		}
	}
	return nil
}

func socialMessagingExtractTags(payload map[string]any) map[string]string {
	for current, value := range payload {
		if !strings.EqualFold(current, "tags") {
			continue
		}
		out := map[string]string{}
		switch typed := value.(type) {
		case map[string]string:
			for key, val := range typed {
				out[key] = val
			}
		case map[string]any:
			for key, val := range typed {
				out[key] = strings.TrimSpace(fmt.Sprint(val))
			}
		case []any:
			for _, item := range typed {
				entry, ok := item.(map[string]any)
				if !ok {
					continue
				}
				key := socialMessagingGetString(entry, "key", socialMessagingGetString(entry, "Key", ""))
				if key == "" {
					continue
				}
				out[key] = socialMessagingGetString(entry, "value", socialMessagingGetString(entry, "Value", ""))
			}
		}
		return out
	}
	return map[string]string{}
}

func socialMessagingExtractTagKeys(payload map[string]any) []string {
	for current, value := range payload {
		if !strings.EqualFold(current, "tagKeys") {
			continue
		}
		out := []string{}
		switch typed := value.(type) {
		case []string:
			for _, item := range typed {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
		case []any:
			for _, item := range typed {
				asString := strings.TrimSpace(fmt.Sprint(item))
				if asString != "" && asString != "<nil>" {
					out = append(out, asString)
				}
			}
		}
		return out
	}
	return nil
}

func socialMessagingCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func socialMessagingListMaps(values map[string]map[string]any) []any {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, socialMessagingCloneMap(values[key]))
	}
	return out
}

func socialMessagingTagsToList(tags map[string]string) []any {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"key": key, "value": tags[key]})
	}
	return out
}

func socialMessagingWABAArn(id string) string {
	return "arn:aws:social-messaging:us-east-1:123456789012:whatsapp-business-account/" + strings.TrimSpace(id)
}
