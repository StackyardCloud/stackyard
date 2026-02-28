package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type migrationHubRefactorSpacesStore struct {
	mu sync.Mutex

	nextEnvironment int64
	nextApplication int64
	nextService     int64
	nextRoute       int64

	environments    map[string]map[string]any
	applications    map[string]map[string]map[string]any
	services        map[string]map[string]map[string]map[string]any
	routes          map[string]map[string]map[string]map[string]any
	environmentVpcs map[string][]map[string]any
	resourcePolicy  map[string]map[string]any
	tags            map[string]map[string]string
}

func newMigrationHubRefactorSpacesStore() *migrationHubRefactorSpacesStore {
	s := &migrationHubRefactorSpacesStore{
		nextEnvironment: 2,
		nextApplication: 2,
		nextService:     2,
		nextRoute:       2,
		environments:    map[string]map[string]any{},
		applications:    map[string]map[string]map[string]any{},
		services:        map[string]map[string]map[string]map[string]any{},
		routes:          map[string]map[string]map[string]map[string]any{},
		environmentVpcs: map[string][]map[string]any{},
		resourcePolicy:  map[string]map[string]any{},
		tags:            map[string]map[string]string{},
	}

	env := s.ensureEnvironmentLocked("env-00000001")
	app := s.ensureApplicationLocked("env-00000001", "app-00000001")
	svc := s.ensureServiceLocked("env-00000001", "app-00000001", "service-00000001")
	route := s.ensureRouteLocked("env-00000001", "app-00000001", "route-00000001")
	route["serviceIdentifier"] = mhrsFirstNonEmpty(mhrsStringAny(svc, "serviceIdentifier"), "service-00000001")

	s.resourcePolicy["env-00000001"] = map[string]any{
		"identifier":      "env-00000001",
		"resourceArn":     mhrsEnvironmentARN("env-00000001"),
		"policy":          `{"Version":"2012-10-17","Statement":[]}`,
		"lastUpdatedTime": time.Now().UTC().Format(time.RFC3339),
	}
	s.tags[mhrsFirstNonEmpty(mhrsStringAny(env, "arn"), mhrsEnvironmentARN("env-00000001"))] = map[string]string{"seed": "true"}
	s.tags[mhrsFirstNonEmpty(mhrsStringAny(app, "arn"), mhrsApplicationARN("app-00000001"))] = map[string]string{"seed": "true"}
	s.tags[mhrsFirstNonEmpty(mhrsStringAny(svc, "arn"), mhrsServiceARN("service-00000001"))] = map[string]string{"seed": "true"}

	return s
}

func (s *migrationHubRefactorSpacesStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	environmentID := mhrsFirstNonEmpty(
		mhrsPathParam(pathParams, "environmentIdentifier", "EnvironmentIdentifier"),
		mhrsStringAny(payload, "environmentIdentifier", "EnvironmentIdentifier"),
		"env-00000001",
	)
	applicationID := mhrsFirstNonEmpty(
		mhrsPathParam(pathParams, "applicationIdentifier", "ApplicationIdentifier"),
		mhrsStringAny(payload, "applicationIdentifier", "ApplicationIdentifier"),
		"app-00000001",
	)
	serviceID := mhrsFirstNonEmpty(
		mhrsPathParam(pathParams, "serviceIdentifier", "ServiceIdentifier"),
		mhrsStringAny(payload, "serviceIdentifier", "ServiceIdentifier"),
		"service-00000001",
	)
	routeID := mhrsFirstNonEmpty(
		mhrsPathParam(pathParams, "routeIdentifier", "RouteIdentifier"),
		mhrsStringAny(payload, "routeIdentifier", "RouteIdentifier"),
		"route-00000001",
	)
	identifier := mhrsFirstNonEmpty(
		mhrsPathParam(pathParams, "identifier", "Identifier"),
		mhrsStringAny(payload, "identifier", "Identifier"),
		environmentID,
	)
	resourceARN := mhrsFirstNonEmpty(
		mhrsPathParam(pathParams, "resourceArn", "ResourceArn"),
		mhrsStringAny(payload, "resourceArn", "ResourceArn"),
		mhrsEnvironmentARN(environmentID),
	)

	switch action {
	case "CreateEnvironment":
		environmentID = fmt.Sprintf("env-%08d", s.nextEnvironmentIDLocked())
		env := s.ensureEnvironmentLocked(environmentID)
		env["name"] = mhrsFirstNonEmpty(mhrsStringAny(payload, "name", "Name"), fmt.Sprintf("stackyard-env-%s", environmentID))
		env["description"] = mhrsFirstNonEmpty(mhrsStringAny(payload, "description", "Description"), "Refactor Spaces environment")
		env["updatedTime"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{
			"environmentIdentifier": environmentID,
			"arn":                   mhrsFirstNonEmpty(mhrsStringAny(env, "arn"), mhrsEnvironmentARN(environmentID)),
			"environment":           mhrsCloneMap(env),
		}

	case "DeleteEnvironment":
		s.ensureEnvironmentLocked(environmentID)
		delete(s.environments, environmentID)
		delete(s.applications, environmentID)
		delete(s.services, environmentID)
		delete(s.routes, environmentID)
		delete(s.environmentVpcs, environmentID)
		delete(s.resourcePolicy, environmentID)
		return map[string]any{"environmentIdentifier": environmentID}

	case "GetEnvironment":
		env := s.ensureEnvironmentLocked(environmentID)
		return map[string]any{"environment": mhrsCloneMap(env)}

	case "ListEnvironments":
		items := make([]any, 0, len(s.environments))
		keys := make([]string, 0, len(s.environments))
		for id := range s.environments {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		for _, id := range keys {
			env := s.environments[id]
			items = append(items, map[string]any{
				"environmentIdentifier": id,
				"name":                  mhrsFirstNonEmpty(mhrsStringAny(env, "name"), id),
				"arn":                   mhrsFirstNonEmpty(mhrsStringAny(env, "arn"), mhrsEnvironmentARN(id)),
			})
		}
		return map[string]any{"environmentSummaryList": items, "nextToken": ""}

	case "ListEnvironmentVpcs":
		vpcs := s.ensureEnvironmentVpcsLocked(environmentID)
		items := make([]any, 0, len(vpcs))
		for _, vpc := range vpcs {
			items = append(items, mhrsCloneMap(vpc))
		}
		return map[string]any{"environmentVpcList": items, "nextToken": ""}

	case "CreateApplication":
		s.ensureEnvironmentLocked(environmentID)
		applicationID = fmt.Sprintf("app-%08d", s.nextApplicationIDLocked())
		app := s.ensureApplicationLocked(environmentID, applicationID)
		app["name"] = mhrsFirstNonEmpty(mhrsStringAny(payload, "name", "Name"), fmt.Sprintf("stackyard-app-%s", applicationID))
		app["proxyType"] = mhrsFirstNonEmpty(mhrsStringAny(payload, "proxyType", "ProxyType"), "API_GATEWAY")
		app["updatedTime"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{
			"applicationIdentifier": applicationID,
			"arn":                   mhrsFirstNonEmpty(mhrsStringAny(app, "arn"), mhrsApplicationARN(applicationID)),
			"application":           mhrsCloneMap(app),
		}

	case "DeleteApplication":
		s.ensureApplicationLocked(environmentID, applicationID)
		delete(s.applications[environmentID], applicationID)
		if byEnv, ok := s.services[environmentID]; ok {
			delete(byEnv, applicationID)
		}
		if byEnv, ok := s.routes[environmentID]; ok {
			delete(byEnv, applicationID)
		}
		return map[string]any{"applicationIdentifier": applicationID}

	case "GetApplication":
		app := s.ensureApplicationLocked(environmentID, applicationID)
		return map[string]any{"application": mhrsCloneMap(app)}

	case "ListApplications":
		items := []any{}
		appsByEnv := s.applications[environmentID]
		keys := make([]string, 0, len(appsByEnv))
		for id := range appsByEnv {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		for _, id := range keys {
			app := appsByEnv[id]
			items = append(items, map[string]any{
				"applicationIdentifier": id,
				"name":                  mhrsFirstNonEmpty(mhrsStringAny(app, "name"), id),
				"arn":                   mhrsFirstNonEmpty(mhrsStringAny(app, "arn"), mhrsApplicationARN(id)),
				"proxyType":             mhrsFirstNonEmpty(mhrsStringAny(app, "proxyType"), "API_GATEWAY"),
			})
		}
		return map[string]any{"applicationSummaryList": items, "nextToken": ""}

	case "CreateService":
		s.ensureApplicationLocked(environmentID, applicationID)
		serviceID = fmt.Sprintf("service-%08d", s.nextServiceIDLocked())
		svc := s.ensureServiceLocked(environmentID, applicationID, serviceID)
		svc["name"] = mhrsFirstNonEmpty(mhrsStringAny(payload, "name", "Name"), fmt.Sprintf("stackyard-service-%s", serviceID))
		svc["endpointType"] = mhrsFirstNonEmpty(mhrsStringAny(payload, "endpointType", "EndpointType"), "URL")
		svc["updatedTime"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{
			"serviceIdentifier": serviceID,
			"arn":               mhrsFirstNonEmpty(mhrsStringAny(svc, "arn"), mhrsServiceARN(serviceID)),
			"service":           mhrsCloneMap(svc),
		}

	case "DeleteService":
		s.ensureServiceLocked(environmentID, applicationID, serviceID)
		if byEnv, ok := s.services[environmentID]; ok {
			if byApp, ok := byEnv[applicationID]; ok {
				delete(byApp, serviceID)
			}
		}
		return map[string]any{"serviceIdentifier": serviceID}

	case "GetService":
		svc := s.ensureServiceLocked(environmentID, applicationID, serviceID)
		return map[string]any{"service": mhrsCloneMap(svc)}

	case "ListServices":
		items := []any{}
		servicesByApp := s.services[environmentID][applicationID]
		keys := make([]string, 0, len(servicesByApp))
		for id := range servicesByApp {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		for _, id := range keys {
			svc := servicesByApp[id]
			items = append(items, map[string]any{
				"serviceIdentifier": id,
				"name":              mhrsFirstNonEmpty(mhrsStringAny(svc, "name"), id),
				"arn":               mhrsFirstNonEmpty(mhrsStringAny(svc, "arn"), mhrsServiceARN(id)),
			})
		}
		return map[string]any{"serviceSummaryList": items, "nextToken": ""}

	case "CreateRoute":
		s.ensureApplicationLocked(environmentID, applicationID)
		routeID = fmt.Sprintf("route-%08d", s.nextRouteIDLocked())
		route := s.ensureRouteLocked(environmentID, applicationID, routeID)
		if v := mhrsFirstNonEmpty(mhrsStringAny(payload, "routeType", "RouteType"), "URI_PATH"); v != "" {
			route["routeType"] = v
		}
		route["updatedTime"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{
			"routeIdentifier": routeID,
			"arn":             mhrsFirstNonEmpty(mhrsStringAny(route, "arn"), mhrsRouteARN(routeID)),
			"route":           mhrsCloneMap(route),
		}

	case "DeleteRoute":
		s.ensureRouteLocked(environmentID, applicationID, routeID)
		if byEnv, ok := s.routes[environmentID]; ok {
			if byApp, ok := byEnv[applicationID]; ok {
				delete(byApp, routeID)
			}
		}
		return map[string]any{"routeIdentifier": routeID}

	case "GetRoute":
		route := s.ensureRouteLocked(environmentID, applicationID, routeID)
		return map[string]any{"route": mhrsCloneMap(route)}

	case "ListRoutes":
		items := []any{}
		routesByApp := s.routes[environmentID][applicationID]
		keys := make([]string, 0, len(routesByApp))
		for id := range routesByApp {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		for _, id := range keys {
			route := routesByApp[id]
			items = append(items, map[string]any{
				"routeIdentifier": id,
				"arn":             mhrsFirstNonEmpty(mhrsStringAny(route, "arn"), mhrsRouteARN(id)),
				"routeType":       mhrsFirstNonEmpty(mhrsStringAny(route, "routeType"), "URI_PATH"),
			})
		}
		return map[string]any{"routeSummaryList": items, "nextToken": ""}

	case "UpdateRoute":
		route := s.ensureRouteLocked(environmentID, applicationID, routeID)
		for k, v := range payload {
			route[k] = v
		}
		route["updatedTime"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{
			"routeIdentifier": routeID,
			"route":           mhrsCloneMap(route),
		}

	case "PutResourcePolicy":
		policy := mhrsFirstNonEmpty(mhrsStringAny(payload, "policy", "Policy"), `{"Version":"2012-10-17","Statement":[]}`)
		if policy == "" {
			policy = `{"Version":"2012-10-17","Statement":[]}`
		}
		resourceARN = mhrsFirstNonEmpty(
			mhrsStringAny(payload, "resourceArn", "ResourceArn"),
			resourceARN,
		)
		s.resourcePolicy[identifier] = map[string]any{
			"identifier":      identifier,
			"resourceArn":     resourceARN,
			"policy":          policy,
			"lastUpdatedTime": time.Now().UTC().Format(time.RFC3339),
		}
		return map[string]any{"resourcePolicy": mhrsCloneMap(s.resourcePolicy[identifier])}

	case "GetResourcePolicy":
		entry, ok := s.resourcePolicy[identifier]
		if !ok {
			entry = map[string]any{
				"identifier":      identifier,
				"resourceArn":     resourceARN,
				"policy":          `{"Version":"2012-10-17","Statement":[]}`,
				"lastUpdatedTime": time.Now().UTC().Format(time.RFC3339),
			}
			s.resourcePolicy[identifier] = entry
		}
		return map[string]any{"resourcePolicy": mhrsCloneMap(entry)}

	case "DeleteResourcePolicy":
		delete(s.resourcePolicy, identifier)
		return map[string]any{"identifier": identifier}

	case "TagResource":
		tagMap := s.ensureTagMapLocked(resourceARN)
		for k, v := range mhrsExtractTags(payload) {
			tagMap[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		tagMap := s.ensureTagMapLocked(resourceARN)
		for _, key := range mhrsExtractTagKeys(payload, query) {
			delete(tagMap, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": mhrsCloneStringMap(s.ensureTagMapLocked(resourceARN))}
	}

	return map[string]any{}
}

func (s *migrationHubRefactorSpacesStore) ensureEnvironmentLocked(environmentID string) map[string]any {
	environmentID = strings.TrimSpace(environmentID)
	if environmentID == "" {
		environmentID = "env-00000001"
	}
	if env, ok := s.environments[environmentID]; ok {
		return env
	}
	now := time.Now().UTC().Format(time.RFC3339)
	env := map[string]any{
		"environmentIdentifier": environmentID,
		"name":                  "stackyard-environment",
		"description":           "Refactor Spaces environment",
		"arn":                   mhrsEnvironmentARN(environmentID),
		"createdTime":           now,
	}
	s.environments[environmentID] = env
	if _, ok := s.applications[environmentID]; !ok {
		s.applications[environmentID] = map[string]map[string]any{}
	}
	if _, ok := s.services[environmentID]; !ok {
		s.services[environmentID] = map[string]map[string]map[string]any{}
	}
	if _, ok := s.routes[environmentID]; !ok {
		s.routes[environmentID] = map[string]map[string]map[string]any{}
	}
	if _, ok := s.environmentVpcs[environmentID]; !ok {
		s.environmentVpcs[environmentID] = []map[string]any{
			{
				"vpcId":      "vpc-00000000000000001",
				"vpcName":    "stackyard-vpc",
				"isDefault":  true,
				"cidrBlocks": []any{"10.0.0.0/16"},
			},
		}
	}
	return env
}

func (s *migrationHubRefactorSpacesStore) ensureApplicationLocked(environmentID, applicationID string) map[string]any {
	s.ensureEnvironmentLocked(environmentID)
	applicationID = strings.TrimSpace(applicationID)
	if applicationID == "" {
		applicationID = "app-00000001"
	}
	if app, ok := s.applications[environmentID][applicationID]; ok {
		return app
	}
	now := time.Now().UTC().Format(time.RFC3339)
	app := map[string]any{
		"applicationIdentifier": applicationID,
		"name":                  "stackyard-application",
		"proxyType":             "API_GATEWAY",
		"arn":                   mhrsApplicationARN(applicationID),
		"createdTime":           now,
	}
	s.applications[environmentID][applicationID] = app
	if _, ok := s.services[environmentID][applicationID]; !ok {
		s.services[environmentID][applicationID] = map[string]map[string]any{}
	}
	if _, ok := s.routes[environmentID][applicationID]; !ok {
		s.routes[environmentID][applicationID] = map[string]map[string]any{}
	}
	return app
}

func (s *migrationHubRefactorSpacesStore) ensureServiceLocked(environmentID, applicationID, serviceID string) map[string]any {
	s.ensureApplicationLocked(environmentID, applicationID)
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		serviceID = "service-00000001"
	}
	if svc, ok := s.services[environmentID][applicationID][serviceID]; ok {
		return svc
	}
	now := time.Now().UTC().Format(time.RFC3339)
	svc := map[string]any{
		"serviceIdentifier": serviceID,
		"name":              "stackyard-service",
		"endpointType":      "URL",
		"arn":               mhrsServiceARN(serviceID),
		"createdTime":       now,
	}
	s.services[environmentID][applicationID][serviceID] = svc
	return svc
}

func (s *migrationHubRefactorSpacesStore) ensureRouteLocked(environmentID, applicationID, routeID string) map[string]any {
	s.ensureApplicationLocked(environmentID, applicationID)
	routeID = strings.TrimSpace(routeID)
	if routeID == "" {
		routeID = "route-00000001"
	}
	if route, ok := s.routes[environmentID][applicationID][routeID]; ok {
		return route
	}
	now := time.Now().UTC().Format(time.RFC3339)
	route := map[string]any{
		"routeIdentifier": routeID,
		"routeType":       "URI_PATH",
		"arn":             mhrsRouteARN(routeID),
		"createdTime":     now,
	}
	s.routes[environmentID][applicationID][routeID] = route
	return route
}

func (s *migrationHubRefactorSpacesStore) ensureEnvironmentVpcsLocked(environmentID string) []map[string]any {
	s.ensureEnvironmentLocked(environmentID)
	return s.environmentVpcs[environmentID]
}

func (s *migrationHubRefactorSpacesStore) ensureTagMapLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = mhrsEnvironmentARN("env-00000001")
	}
	tagMap, ok := s.tags[resourceARN]
	if !ok {
		tagMap = map[string]string{}
		s.tags[resourceARN] = tagMap
	}
	return tagMap
}

func (s *migrationHubRefactorSpacesStore) nextEnvironmentIDLocked() int64 {
	id := s.nextEnvironment
	s.nextEnvironment++
	return id
}

func (s *migrationHubRefactorSpacesStore) nextApplicationIDLocked() int64 {
	id := s.nextApplication
	s.nextApplication++
	return id
}

func (s *migrationHubRefactorSpacesStore) nextServiceIDLocked() int64 {
	id := s.nextService
	s.nextService++
	return id
}

func (s *migrationHubRefactorSpacesStore) nextRouteIDLocked() int64 {
	id := s.nextRoute
	s.nextRoute++
	return id
}

func mhrsPathParam(pathParams map[string]string, keys ...string) string {
	for _, key := range keys {
		for k, v := range pathParams {
			if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

func mhrsStringAny(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			if s, ok := value.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s
				}
			}
		}
		for k, v := range payload {
			if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
				if s, ok := v.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						return s
					}
				}
			}
		}
	}
	return ""
}

func mhrsFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func mhrsExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	for _, key := range []string{"tags", "Tags"} {
		value, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case map[string]string:
			for k, v := range typed {
				k = strings.TrimSpace(k)
				if k == "" {
					continue
				}
				out[k] = strings.TrimSpace(v)
			}
		case map[string]any:
			for k, v := range typed {
				k = strings.TrimSpace(k)
				if k == "" {
					continue
				}
				out[k] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
	}
	return out
}

func mhrsExtractTagKeys(payload map[string]any, query url.Values) []string {
	keys := []string{}
	appendKey := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			keys = append(keys, value)
		}
	}

	if raw, ok := payload["tagKeys"]; ok {
		switch typed := raw.(type) {
		case []string:
			for _, item := range typed {
				appendKey(item)
			}
		case []any:
			for _, item := range typed {
				appendKey(fmt.Sprintf("%v", item))
			}
		case string:
			for _, item := range strings.Split(typed, ",") {
				appendKey(item)
			}
		}
	}

	for _, item := range query["tagKeys"] {
		for _, key := range strings.Split(item, ",") {
			appendKey(key)
		}
	}

	return keys
}

func mhrsCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mhrsCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mhrsEnvironmentARN(id string) string {
	return fmt.Sprintf("arn:aws:refactor-spaces:us-east-1:123456789012:environment/%s", id)
}

func mhrsApplicationARN(id string) string {
	return fmt.Sprintf("arn:aws:refactor-spaces:us-east-1:123456789012:application/%s", id)
}

func mhrsServiceARN(id string) string {
	return fmt.Sprintf("arn:aws:refactor-spaces:us-east-1:123456789012:service/%s", id)
}

func mhrsRouteARN(id string) string {
	return fmt.Sprintf("arn:aws:refactor-spaces:us-east-1:123456789012:route/%s", id)
}
