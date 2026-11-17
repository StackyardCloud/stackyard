package server

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
)

// Inline children: ARM types such as virtual networks embed child resources
// (subnets) in a property as well as serving them at their own URLs. The
// parent record stores the embedded list as the children's current
// fixtures, refreshed whenever the parent or a child changes.

// validateAzureARMInline checks the inline items of a parent PUT payload as
// if each were PUT on its own, returning a non-empty message to reject it.
func validateAzureARMInline(typ azureARMResourceType, payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, prop := range slices.Sorted(maps.Keys(typ.InlineChildren)) {
		childType := typ.InlineChildren[prop]
		raw := props[prop]
		if raw == nil {
			continue
		}
		items, ok := raw.([]any)
		if !ok {
			return "properties." + prop + " must be an array."
		}
		child, _ := lookupAzureARMResourceType(typ.Namespace, childType)
		seen := map[string]bool{}
		for i, rawItem := range items {
			item := azureARMObject(rawItem)
			at := fmt.Sprintf("properties.%s[%d]", prop, i)
			name := strings.TrimSpace(azureARMString(item["name"]))
			switch {
			case name == "":
				return at + ".name is required."
			case strings.ContainsAny(name, "/?#"):
				return at + ".name must not contain /, ? or #."
			case seen[strings.ToLower(name)]:
				return at + ".name " + name + " is used twice."
			}
			seen[strings.ToLower(name)] = true
			itemPayload := map[string]any{"properties": azureARMObject(item["properties"])}
			msg := ""
			if child.ValidateName != nil {
				msg = child.ValidateName(name)
			}
			if msg == "" && child.Validate != nil {
				msg = child.Validate(itemPayload)
			}
			if msg == "" && child.ValidateInParent != nil {
				msg = child.ValidateInParent(props, name, itemPayload)
			}
			if msg != "" {
				return at + ": " + msg
			}
		}
	}
	return ""
}

// validateAzureARMKeptInlineLocked checks the children a parent PUT keeps,
// because it omits their property, against the new parent payload, such as
// subnets against a changed address space. Caller holds providerStorageMu.
func (s *Server) validateAzureARMKeptInlineLocked(typ azureARMResourceType, id string, payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, prop := range slices.Sorted(maps.Keys(typ.InlineChildren)) {
		child, _ := lookupAzureARMResourceType(typ.Namespace, typ.InlineChildren[prop])
		if props[prop] != nil || child.ValidateInParent == nil {
			continue
		}
		kept := s.azureARMInlineChildrenLocked(id, child)
		list := make([]any, 0, len(kept))
		for _, rec := range kept {
			list = append(list, map[string]any{"name": rec.Name, "properties": rec.Properties})
		}
		parent := azureARMMerge(props, map[string]any{prop: list})
		for _, rec := range kept {
			if msg := child.ValidateInParent(parent, rec.Name, map[string]any{"properties": rec.Properties}); msg != "" {
				return "properties." + prop + " keeps " + rec.Name + ", which would be invalid: " + msg
			}
		}
	}
	return ""
}

// syncAzureARMInlineLocked replaces the children of record named by each
// inline property the payload sets, and returns record's properties with
// every inline property listing the current children. Caller holds
// providerStorageMu; record is not yet stored and not modified.
func (s *Server) syncAzureARMInlineLocked(typ azureARMResourceType, record *azureARMResource, payloadProps map[string]any) map[string]any {
	for prop, childType := range typ.InlineChildren {
		items, present := payloadProps[prop].([]any)
		if !present {
			// Absent or null: keep the children.
			continue
		}
		child, _ := lookupAzureARMResourceType(typ.Namespace, childType)
		keep := map[string]bool{}
		for _, rawItem := range items {
			item := azureARMObject(rawItem)
			name := strings.TrimSpace(azureARMString(item["name"]))
			rec := azureARMChild(record, child, name, azureARMObject(item["properties"]))
			key := azureARMKey(rec.ID)
			if existing := s.azureARMResources[key]; existing != nil {
				rec.ID, rec.Name, rec.CreatedAt = existing.ID, existing.Name, existing.CreatedAt
				rec.Properties = azureARMWithState(child, azureARMProperties(child, rec), existing)
			}
			s.azureARMResources[key] = rec
			keep[key] = true
		}
		for _, stale := range s.azureARMInlineChildrenLocked(record.ID, child) {
			if key := azureARMKey(stale.ID); !keep[key] {
				s.deleteAzureARMTreeLocked(stale.ID)
			}
		}
	}
	return s.azureARMInlinePropertiesLocked(typ, record)
}

// azureARMInlinePropertiesLocked returns record's properties with each
// inline property listing its current children. Caller holds
// providerStorageMu.
func (s *Server) azureARMInlinePropertiesLocked(typ azureARMResourceType, record *azureARMResource) map[string]any {
	if len(typ.InlineChildren) == 0 {
		return record.Properties
	}
	overlay := map[string]any{}
	for prop, childType := range typ.InlineChildren {
		child, _ := lookupAzureARMResourceType(typ.Namespace, childType)
		list := []any{}
		for _, rec := range s.azureARMInlineChildrenLocked(record.ID, child) {
			list = append(list, map[string]any{"id": rec.ID, "name": rec.Name, "type": rec.Type, "properties": rec.Properties})
		}
		overlay[prop] = list
	}
	return azureARMMerge(record.Properties, overlay)
}

// refreshAzureARMParentLocked re-lists the inline children of the record
// at parentID after one of its children changed. Caller holds
// providerStorageMu.
func (s *Server) refreshAzureARMParentLocked(parentID string) {
	parent := s.azureARMResources[azureARMKey(parentID)]
	if parent == nil {
		return
	}
	namespace, typePath, _ := strings.Cut(parent.Type, "/")
	typ, ok := lookupAzureARMResourceType(namespace, typePath)
	if !ok || len(typ.InlineChildren) == 0 {
		return
	}
	updated := *parent
	updated.Properties = s.azureARMInlinePropertiesLocked(typ, parent)
	s.azureARMResources[azureARMKey(parentID)] = &updated
}

// azureARMInlineChildrenLocked returns the direct children of parentID of
// the given type, sorted by ID. Caller holds providerStorageMu.
func (s *Server) azureARMInlineChildrenLocked(parentID string, child azureARMResourceType) []*azureARMResource {
	prefix := azureARMKey(parentID + "/")
	return s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, child.fullType()) && strings.HasPrefix(azureARMKey(rec.ID), prefix)
	})
}

// deleteAzureARMTreeLocked deletes the record at id with its child and
// extension resources, reporting whether it existed. Caller holds
// providerStorageMu.
func (s *Server) deleteAzureARMTreeLocked(id string) bool {
	record, existed := s.azureARMResources[azureARMKey(id)]
	delete(s.azureARMResources, azureARMKey(id))
	prefix := azureARMKey(id + "/")
	for k := range s.azureARMResources {
		if strings.HasPrefix(k, prefix) {
			delete(s.azureARMResources, k)
		}
	}
	// Resources created alongside, such as a private endpoint's interface.
	// ponytail: only the deleted record's own; no child type links resources yet.
	if existed {
		for _, linkedID := range record.LinkedIDs {
			s.deleteAzureARMTreeLocked(linkedID)
		}
	}
	return existed
}

// storeAzureARMLinkedLocked runs typ.Linked for a record about to be
// stored: it merges the returned properties into it, stores the linked
// resources, and deletes those the previous version linked but this one
// does not. On rejection it writes the error and returns false. Caller
// holds providerStorageMu.
func (s *Server) storeAzureARMLinkedLocked(w http.ResponseWriter, typ azureARMResourceType, record, existing *azureARMResource, sent map[string]any) bool {
	// The hook sees what the client sent, write-only properties included,
	// under the record's own properties.
	view := *record
	view.Properties = azureARMMerge(sent, record.Properties)
	props, linked, msg := typ.Linked(s, &view)
	if msg != "" {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
		return false
	}
	record.Properties = azureARMMerge(record.Properties, props)
	record.LinkedIDs = make([]string, 0, len(linked))
	for _, rec := range linked {
		s.azureARMResources[azureARMKey(rec.ID)] = rec
		record.LinkedIDs = append(record.LinkedIDs, rec.ID)
	}
	if existing == nil {
		return true
	}
	for _, old := range existing.LinkedIDs {
		if !slices.ContainsFunc(record.LinkedIDs, func(id string) bool { return strings.EqualFold(id, old) }) {
			s.deleteAzureARMTreeLocked(old)
		}
	}
	return true
}
