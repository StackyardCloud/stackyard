package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type resourceExplorer2Store struct {
	mu sync.Mutex

	setupEnabled bool

	defaultIndexArn string
	defaultViewArn  string

	indices      map[string]map[string]any
	views        map[string]map[string]any
	managedViews map[string]map[string]any
	tags         map[string]map[string]string

	resources              []map[string]any
	supportedResourceTypes []map[string]any
	streamingAccess        []map[string]any

	accountConfig map[string]any
}

func newResourceExplorer2Store() *resourceExplorer2Store {
	now := time.Now().UTC().Format(time.RFC3339)
	indexArn := "arn:aws:resource-explorer-2:us-east-1:123456789012:index/00000000-0000-0000-0000-000000000000"
	viewArn := "arn:aws:resource-explorer-2:us-east-1:123456789012:view/stackyard-default/00000000-0000-0000-0000-000000000000"
	managedViewArn := "arn:aws:resource-explorer-2:us-east-1:123456789012:view/AWS-QuickSetup-Default-View/11111111-1111-1111-1111-111111111111"

	indices := map[string]map[string]any{
		indexArn: {
			"Arn":       indexArn,
			"Region":    "us-east-1",
			"Type":      "LOCAL",
			"State":     "ACTIVE",
			"CreatedAt": now,
		},
	}

	views := map[string]map[string]any{
		viewArn: {
			"ViewArn":   viewArn,
			"Owner":     "123456789012",
			"Scope":     "arn:aws:iam::123456789012:root",
			"CreatedAt": now,
			"UpdatedAt": now,
			"Filters": map[string]any{
				"FilterString": "region:us-east-1",
			},
			"IncludedProperties": []any{
				map[string]any{"Name": "tags"},
			},
		},
	}

	managedViews := map[string]map[string]any{
		managedViewArn: {
			"ViewArn":     managedViewArn,
			"ManagedView": "AWS-QuickSetup-Default-View",
			"CreatedAt":   now,
			"UpdatedAt":   now,
		},
	}

	resources := []map[string]any{
		{
			"Arn":             "arn:aws:ec2:us-east-1:123456789012:instance/i-00000000000000001",
			"OwningAccountId": "123456789012",
			"Region":          "us-east-1",
			"ResourceType":    "AWS::EC2::Instance",
			"Service":         "ec2",
			"Properties": []any{
				map[string]any{"Name": "Name", "Data": "stackyard-instance", "LastReportedAt": now},
			},
			"LastReportedAt": now,
		},
	}

	return &resourceExplorer2Store{
		setupEnabled:           true,
		defaultIndexArn:        indexArn,
		defaultViewArn:         viewArn,
		indices:                indices,
		views:                  views,
		managedViews:           managedViews,
		tags:                   map[string]map[string]string{viewArn: {"stackyard": "true"}, indexArn: {"stackyard": "true"}},
		resources:              resources,
		supportedResourceTypes: []map[string]any{{"ResourceType": "AWS::EC2::Instance", "Service": "ec2"}},
		streamingAccess:        []map[string]any{{"Service": "ec2", "CanStream": true}, {"Service": "s3", "CanStream": true}},
		accountConfig: map[string]any{
			"RoleArn": "arn:aws:iam::123456789012:role/AWSServiceRoleForResourceExplorer",
			"OrgConfiguration": map[string]any{
				"EnableAWSServiceAccess": true,
			},
		},
	}
}

func (s *resourceExplorer2Store) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.syncPayloadWithQuery(payload, query)

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "AssociateDefaultView":
		viewArn := re2ViewArn(payload, s.defaultViewArn)
		s.defaultViewArn = viewArn
		return map[string]any{"ViewArn": viewArn}
	case "BatchGetView":
		arns := re2StringSliceAny(payload["ViewArns"])
		if len(arns) == 0 {
			arns = []string{s.defaultViewArn}
		}
		views := make([]any, 0, len(arns))
		errs := make([]any, 0)
		for _, arn := range arns {
			if view, ok := s.views[arn]; ok {
				views = append(views, re2CloneMap(view))
				continue
			}
			if view, ok := s.managedViews[arn]; ok {
				views = append(views, re2CloneMap(view))
				continue
			}
			errs = append(errs, map[string]any{"ViewArn": arn, "ErrorMessage": "View not found"})
		}
		return map[string]any{"Views": views, "Errors": errs}
	case "CreateIndex":
		indexArn := re2StringAny(payload, "Arn", "")
		if strings.TrimSpace(indexArn) == "" {
			indexArn = fmt.Sprintf("arn:aws:resource-explorer-2:us-east-1:123456789012:index/%d", len(s.indices)+1)
		}
		indexType := re2StringAny(payload, "Type", "LOCAL")
		idx := map[string]any{
			"Arn":       indexArn,
			"Region":    "us-east-1",
			"Type":      indexType,
			"State":     "ACTIVE",
			"CreatedAt": now,
		}
		s.indices[indexArn] = idx
		s.defaultIndexArn = indexArn
		if s.tags[indexArn] == nil {
			s.tags[indexArn] = map[string]string{"stackyard": "true"}
		}
		return re2CloneMap(idx)
	case "CreateResourceExplorerSetup":
		s.setupEnabled = true
		return map[string]any{"Status": "ACTIVE"}
	case "CreateView":
		viewArn := re2StringAny(payload, "ViewArn", "")
		if strings.TrimSpace(viewArn) == "" {
			viewName := re2StringAny(payload, "ViewName", "stackyard-view")
			viewArn = fmt.Sprintf("arn:aws:resource-explorer-2:us-east-1:123456789012:view/%s/%d", viewName, len(s.views)+1)
		}
		view := map[string]any{
			"ViewArn":   viewArn,
			"Owner":     "123456789012",
			"Scope":     "arn:aws:iam::123456789012:root",
			"CreatedAt": now,
			"UpdatedAt": now,
			"Filters": map[string]any{
				"FilterString": re2StringAny(payload, "Filters.FilterString", "region:us-east-1"),
			},
			"IncludedProperties": []any{
				map[string]any{"Name": "tags"},
			},
		}
		s.views[viewArn] = view
		if strings.TrimSpace(s.defaultViewArn) == "" {
			s.defaultViewArn = viewArn
		}
		if s.tags[viewArn] == nil {
			s.tags[viewArn] = map[string]string{"stackyard": "true"}
		}
		return map[string]any{"View": re2CloneMap(view)}
	case "DeleteIndex":
		arn := re2StringAny(payload, "Arn", s.defaultIndexArn)
		delete(s.indices, arn)
		if s.defaultIndexArn == arn {
			s.defaultIndexArn = re2FirstKey(s.indices)
		}
		return map[string]any{}
	case "DeleteResourceExplorerSetup":
		s.setupEnabled = false
		return map[string]any{}
	case "DeleteView":
		arn := re2ViewArn(payload, s.defaultViewArn)
		delete(s.views, arn)
		if s.defaultViewArn == arn {
			s.defaultViewArn = re2FirstKey(s.views)
		}
		return map[string]any{}
	case "DisassociateDefaultView":
		s.defaultViewArn = ""
		return map[string]any{}
	case "GetAccountLevelServiceConfiguration":
		return re2CloneMap(s.accountConfig)
	case "GetDefaultView":
		return map[string]any{"ViewArn": s.defaultViewArn}
	case "GetIndex":
		arn := re2StringAny(payload, "Arn", s.defaultIndexArn)
		if idx, ok := s.indices[arn]; ok {
			return re2CloneMap(idx)
		}
		if len(s.indices) > 0 {
			return re2CloneMap(s.indices[re2FirstKey(s.indices)])
		}
		return map[string]any{"Arn": arn, "State": "DELETED"}
	case "GetManagedView":
		arn := re2ViewArn(payload, re2FirstKey(s.managedViews))
		if v, ok := s.managedViews[arn]; ok {
			return map[string]any{"ManagedView": re2CloneMap(v)}
		}
		return map[string]any{"ManagedView": map[string]any{"ViewArn": arn}}
	case "GetResourceExplorerSetup":
		status := "DELETED"
		if s.setupEnabled {
			status = "ACTIVE"
		}
		return map[string]any{"Status": status}
	case "GetServiceIndex":
		if idx, ok := s.indices[s.defaultIndexArn]; ok {
			return re2CloneMap(idx)
		}
		return map[string]any{"State": "DELETED"}
	case "GetServiceView":
		arn := re2ViewArn(payload, s.defaultViewArn)
		if v, ok := s.views[arn]; ok {
			return map[string]any{"ServiceView": re2CloneMap(v)}
		}
		return map[string]any{"ServiceView": map[string]any{"ViewArn": arn}}
	case "GetView":
		arn := re2ViewArn(payload, s.defaultViewArn)
		if v, ok := s.views[arn]; ok {
			return map[string]any{"View": re2CloneMap(v)}
		}
		if v, ok := s.managedViews[arn]; ok {
			return map[string]any{"View": re2CloneMap(v)}
		}
		return map[string]any{"View": map[string]any{"ViewArn": arn}}
	case "ListIndexes":
		return map[string]any{"Indexes": re2ListFromMap(s.indices), "NextToken": ""}
	case "ListIndexesForMembers":
		return map[string]any{"MemberIndexes": re2ListFromMap(s.indices), "NextToken": ""}
	case "ListManagedViews":
		return map[string]any{"ManagedViews": re2ListFromMap(s.managedViews), "NextToken": ""}
	case "ListResources":
		return map[string]any{"Resources": re2CloneList(s.resources), "NextToken": ""}
	case "ListServiceIndexes":
		return map[string]any{"ServiceIndexes": re2ListFromMap(s.indices), "NextToken": ""}
	case "ListServiceViews":
		serviceViews := make([]any, 0, len(s.views))
		for _, item := range re2ListFromMap(s.views) {
			serviceViews = append(serviceViews, item)
		}
		return map[string]any{"ServiceViews": serviceViews, "NextToken": ""}
	case "ListStreamingAccessForServices":
		return map[string]any{"StreamingAccessDetails": re2CloneList(s.streamingAccess), "NextToken": ""}
	case "ListSupportedResourceTypes":
		return map[string]any{"ResourceTypes": re2CloneList(s.supportedResourceTypes), "NextToken": ""}
	case "ListTagsForResource":
		arn := re2ResourceARN(pathParams, payload, s.defaultViewArn)
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{}
		}
		return map[string]any{"Tags": re2CloneTags(s.tags[arn])}
	case "ListViews":
		return map[string]any{"Views": re2ListFromMap(s.views), "NextToken": ""}
	case "Search":
		return map[string]any{
			"Resources": re2CloneList(s.resources),
			"Count": map[string]any{
				"Complete":       true,
				"TotalResources": len(s.resources),
			},
			"NextToken": "",
		}
	case "TagResource":
		arn := re2ResourceARN(pathParams, payload, s.defaultViewArn)
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{}
		}
		re2MergeTags(s.tags[arn], payload["Tags"])
		re2MergeTags(s.tags[arn], payload["tags"])
		return map[string]any{}
	case "UntagResource":
		arn := re2ResourceARN(pathParams, payload, s.defaultViewArn)
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{}
		}
		for _, k := range re2TagKeys(payload) {
			delete(s.tags[arn], k)
		}
		return map[string]any{}
	case "UpdateIndexType":
		arn := re2StringAny(payload, "Arn", s.defaultIndexArn)
		newType := re2StringAny(payload, "Type", "LOCAL")
		if idx, ok := s.indices[arn]; ok {
			idx["Type"] = newType
			idx["UpdatedAt"] = now
			return re2CloneMap(idx)
		}
		idx := map[string]any{"Arn": arn, "Type": newType, "State": "ACTIVE", "UpdatedAt": now}
		s.indices[arn] = idx
		return re2CloneMap(idx)
	case "UpdateView":
		arn := re2ViewArn(payload, s.defaultViewArn)
		view, ok := s.views[arn]
		if !ok {
			view = map[string]any{"ViewArn": arn, "CreatedAt": now}
			s.views[arn] = view
		}
		view["UpdatedAt"] = now
		if f := re2StringAny(payload, "Filters.FilterString", ""); strings.TrimSpace(f) != "" {
			view["Filters"] = map[string]any{"FilterString": f}
		}
		return map[string]any{"View": re2CloneMap(view)}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"Items": []any{}, "NextToken": ""}
	}
	if strings.HasPrefix(action, "Get") {
		return map[string]any{"Status": "ACTIVE"}
	}
	if strings.HasPrefix(action, "Create") {
		return map[string]any{"Status": "CREATED"}
	}
	if strings.HasPrefix(action, "Update") {
		return map[string]any{"Status": "UPDATED"}
	}
	if strings.HasPrefix(action, "Delete") || strings.HasPrefix(action, "Disassociate") || strings.HasPrefix(action, "Untag") {
		return map[string]any{}
	}
	return map[string]any{}
}

func (s *resourceExplorer2Store) syncPayloadWithQuery(payload map[string]any, query url.Values) {
	if payload == nil {
		return
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if strings.EqualFold(key, "tagKeys") {
			items := make([]any, 0, len(values))
			for _, value := range values {
				value = strings.TrimSpace(value)
				if value != "" {
					items = append(items, value)
				}
			}
			if len(items) > 0 {
				payload["tagKeys"] = items
			}
			continue
		}
		if _, exists := payload[key]; exists {
			continue
		}
		payload[key] = values[len(values)-1]
	}
}

func re2ViewArn(payload map[string]any, fallback string) string {
	if v := strings.TrimSpace(re2StringAny(payload, "ViewArn", "")); v != "" {
		return v
	}
	if v := strings.TrimSpace(re2StringAny(payload, "viewArn", "")); v != "" {
		return v
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "arn:aws:resource-explorer-2:us-east-1:123456789012:view/stackyard-default/00000000-0000-0000-0000-000000000000"
}

func re2ResourceARN(pathParams map[string]string, payload map[string]any, fallback string) string {
	if v := strings.TrimSpace(pathParams["resourceArn"]); v != "" {
		return v
	}
	if v := strings.TrimSpace(re2StringAny(payload, "resourceArn", "")); v != "" {
		return v
	}
	if v := strings.TrimSpace(re2StringAny(payload, "ResourceArn", "")); v != "" {
		return v
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "arn:aws:resource-explorer-2:us-east-1:123456789012:view/stackyard-default/00000000-0000-0000-0000-000000000000"
}

func re2StringAny(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	if strings.Contains(key, ".") {
		parts := strings.Split(key, ".")
		var cur any = payload
		for _, part := range parts {
			m, ok := cur.(map[string]any)
			if !ok {
				return fallback
			}
			v, ok := m[part]
			if !ok {
				return fallback
			}
			cur = v
		}
		if cur == nil {
			return fallback
		}
		return strings.TrimSpace(fmt.Sprintf("%v", cur))
	}
	if v, ok := payload[key]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	return fallback
}

func re2StringSliceAny(raw any) []string {
	out := []string{}
	seen := map[string]struct{}{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	switch vv := raw.(type) {
	case []any:
		for _, item := range vv {
			add(fmt.Sprintf("%v", item))
		}
	case []string:
		for _, item := range vv {
			add(item)
		}
	case string:
		add(vv)
	}
	sort.Strings(out)
	return out
}

func re2MergeTags(dst map[string]string, raw any) {
	if dst == nil {
		return
	}
	switch vv := raw.(type) {
	case map[string]any:
		for key, value := range vv {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			dst[key] = strings.TrimSpace(fmt.Sprintf("%v", value))
		}
	case map[string]string:
		for key, value := range vv {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			dst[key] = strings.TrimSpace(value)
		}
	}
}

func re2TagKeys(payload map[string]any) []string {
	out := []string{}
	seen := map[string]struct{}{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}

	for _, key := range []string{"tagKeys", "TagKeys"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch vv := raw.(type) {
		case []any:
			for _, item := range vv {
				add(fmt.Sprintf("%v", item))
			}
		case []string:
			for _, item := range vv {
				add(item)
			}
		case string:
			add(vv)
		}
	}
	sort.Strings(out)
	return out
}

func re2CloneTags(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func re2CloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = re2CloneAny(v)
	}
	return out
}

func re2CloneAny(v any) any {
	switch vv := v.(type) {
	case map[string]any:
		return re2CloneMap(vv)
	case []any:
		out := make([]any, 0, len(vv))
		for _, item := range vv {
			out = append(out, re2CloneAny(item))
		}
		return out
	default:
		return vv
	}
}

func re2CloneList(items []map[string]any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, re2CloneMap(item))
	}
	return out
}

func re2ListFromMap(items map[string]map[string]any) []any {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, re2CloneMap(items[key]))
	}
	return out
}

func re2FirstKey(items map[string]map[string]any) string {
	if len(items) == 0 {
		return ""
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}
