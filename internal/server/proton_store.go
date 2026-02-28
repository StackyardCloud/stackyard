package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type protonStore struct {
	mu sync.Mutex

	nextServiceID     int64
	nextEnvironmentID int64
	nextTemplateID    int64
	nextRepositoryID  int64

	services             map[string]map[string]any
	environments         map[string]map[string]any
	serviceTemplates     map[string]map[string]any
	environmentTemplates map[string]map[string]any
	repositories         map[string]map[string]any
	tags                 map[string]map[string]string
}

func newProtonStore() *protonStore {
	s := &protonStore{
		nextServiceID:        2,
		nextEnvironmentID:    2,
		nextTemplateID:       2,
		nextRepositoryID:     2,
		services:             map[string]map[string]any{},
		environments:         map[string]map[string]any{},
		serviceTemplates:     map[string]map[string]any{},
		environmentTemplates: map[string]map[string]any{},
		repositories:         map[string]map[string]any{},
		tags:                 map[string]map[string]string{},
	}

	service := s.ensureServiceLocked("stackyard-service")
	environment := s.ensureEnvironmentLocked("stackyard-environment")
	serviceTemplate := s.ensureServiceTemplateLocked("stackyard-service-template")
	environmentTemplate := s.ensureEnvironmentTemplateLocked("stackyard-environment-template")
	repository := s.ensureRepositoryLocked("stackyard-repository")

	s.tags[protonDefaultStringAny(service, "arn", "")] = map[string]string{"stackyard": "true"}
	s.tags[protonDefaultStringAny(environment, "arn", "")] = map[string]string{"stackyard": "true"}
	s.tags[protonDefaultStringAny(serviceTemplate, "arn", "")] = map[string]string{"stackyard": "true"}
	s.tags[protonDefaultStringAny(environmentTemplate, "arn", "")] = map[string]string{"stackyard": "true"}
	s.tags[protonDefaultStringAny(repository, "arn", "")] = map[string]string{"stackyard": "true"}

	return s
}

func (s *protonStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateService":
		name := protonDefaultStringAny(payload, "name", fmt.Sprintf("stackyard-service-%06d", s.nextServiceID))
		s.nextServiceID++
		service := s.ensureServiceLocked(name)
		for k, v := range payload {
			service[k] = v
		}
		return map[string]any{"service": protonCloneAnyMap(service)}

	case "CreateEnvironment":
		name := protonDefaultStringAny(payload, "name", fmt.Sprintf("stackyard-environment-%06d", s.nextEnvironmentID))
		s.nextEnvironmentID++
		environment := s.ensureEnvironmentLocked(name)
		for k, v := range payload {
			environment[k] = v
		}
		return map[string]any{"environment": protonCloneAnyMap(environment)}

	case "CreateServiceTemplate":
		name := protonDefaultStringAny(payload, "name", fmt.Sprintf("stackyard-service-template-%06d", s.nextTemplateID))
		s.nextTemplateID++
		template := s.ensureServiceTemplateLocked(name)
		for k, v := range payload {
			template[k] = v
		}
		return map[string]any{"serviceTemplate": protonCloneAnyMap(template)}

	case "CreateEnvironmentTemplate":
		name := protonDefaultStringAny(payload, "name", fmt.Sprintf("stackyard-environment-template-%06d", s.nextTemplateID))
		s.nextTemplateID++
		template := s.ensureEnvironmentTemplateLocked(name)
		for k, v := range payload {
			template[k] = v
		}
		return map[string]any{"environmentTemplate": protonCloneAnyMap(template)}

	case "CreateRepository":
		name := protonDefaultStringAny(payload, "name", fmt.Sprintf("stackyard-repository-%06d", s.nextRepositoryID))
		s.nextRepositoryID++
		repository := s.ensureRepositoryLocked(name)
		for k, v := range payload {
			repository[k] = v
		}
		return map[string]any{"repository": protonCloneAnyMap(repository)}

	case "ListServices":
		keys := protonSortedKeys(s.services)
		items := make([]any, 0, len(keys))
		for _, key := range keys {
			items = append(items, protonCloneAnyMap(s.services[key]))
		}
		return map[string]any{"services": items, "nextToken": ""}

	case "ListEnvironments":
		keys := protonSortedKeys(s.environments)
		items := make([]any, 0, len(keys))
		for _, key := range keys {
			items = append(items, protonCloneAnyMap(s.environments[key]))
		}
		return map[string]any{"environments": items, "nextToken": ""}

	case "ListServiceTemplates":
		keys := protonSortedKeys(s.serviceTemplates)
		items := make([]any, 0, len(keys))
		for _, key := range keys {
			items = append(items, protonCloneAnyMap(s.serviceTemplates[key]))
		}
		return map[string]any{"templates": items, "nextToken": ""}

	case "ListEnvironmentTemplates":
		keys := protonSortedKeys(s.environmentTemplates)
		items := make([]any, 0, len(keys))
		for _, key := range keys {
			items = append(items, protonCloneAnyMap(s.environmentTemplates[key]))
		}
		return map[string]any{"templates": items, "nextToken": ""}

	case "ListRepositories":
		keys := protonSortedKeys(s.repositories)
		items := make([]any, 0, len(keys))
		for _, key := range keys {
			items = append(items, protonCloneAnyMap(s.repositories[key]))
		}
		return map[string]any{"repositories": items, "nextToken": ""}

	case "GetService":
		return map[string]any{"service": protonCloneAnyMap(s.ensureServiceLocked("stackyard-service"))}
	case "GetEnvironment":
		return map[string]any{"environment": protonCloneAnyMap(s.ensureEnvironmentLocked("stackyard-environment"))}
	case "GetServiceTemplate":
		return map[string]any{"serviceTemplate": protonCloneAnyMap(s.ensureServiceTemplateLocked("stackyard-service-template"))}
	case "GetEnvironmentTemplate":
		return map[string]any{"environmentTemplate": protonCloneAnyMap(s.ensureEnvironmentTemplateLocked("stackyard-environment-template"))}
	case "GetRepository":
		return map[string]any{"repository": protonCloneAnyMap(s.ensureRepositoryLocked("stackyard-repository"))}
	case "GetAccountSettings":
		return map[string]any{"accountSettings": map[string]any{"pipelineCodebuildRoleArn": "arn:aws:iam::123456789012:role/stackyard-proton"}}

	case "TagResource":
		resourceArn := protonDefaultStringAny(payload, "resourceArn", "arn:aws:proton:us-east-1:123456789012:service/stackyard-service")
		if s.tags[resourceArn] == nil {
			s.tags[resourceArn] = map[string]string{}
		}
		if raw, ok := payload["tags"].([]any); ok {
			for _, item := range raw {
				tag, ok := item.(map[string]any)
				if !ok {
					continue
				}
				key := protonDefaultStringAny(tag, "key", "")
				value := protonDefaultStringAny(tag, "value", "")
				if key != "" {
					s.tags[resourceArn][key] = value
				}
			}
		}
		return map[string]any{}

	case "UntagResource":
		resourceArn := protonDefaultStringAny(payload, "resourceArn", "arn:aws:proton:us-east-1:123456789012:service/stackyard-service")
		if raw, ok := payload["tagKeys"].([]any); ok {
			for _, item := range raw {
				key := strings.TrimSpace(fmt.Sprintf("%v", item))
				if key != "" && s.tags[resourceArn] != nil {
					delete(s.tags[resourceArn], key)
				}
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceArn := protonDefaultStringAny(payload, "resourceArn", "arn:aws:proton:us-east-1:123456789012:service/stackyard-service")
		out := make([]any, 0, len(s.tags[resourceArn]))
		for _, key := range protonSortedKeys(s.tags[resourceArn]) {
			out = append(out, map[string]any{"key": key, "value": s.tags[resourceArn][key]})
		}
		return map[string]any{"tags": out}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"items": []any{}, "nextToken": ""}
	}
	if strings.HasPrefix(action, "Get") || strings.HasPrefix(action, "Describe") {
		return map[string]any{"status": "ACTIVE"}
	}
	if strings.HasPrefix(action, "Create") {
		return map[string]any{"status": "CREATED"}
	}
	if strings.HasPrefix(action, "Update") {
		return map[string]any{"status": "UPDATED"}
	}
	if strings.HasPrefix(action, "Delete") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "Cancel") || strings.HasPrefix(action, "Accept") || strings.HasPrefix(action, "Reject") || strings.HasPrefix(action, "Notify") {
		return map[string]any{"status": "SUCCESS"}
	}

	return map[string]any{}
}

func (s *protonStore) ensureServiceLocked(name string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-service"
	}
	if existing := s.services[name]; existing != nil {
		return existing
	}
	item := map[string]any{
		"name":      name,
		"arn":       fmt.Sprintf("arn:aws:proton:us-east-1:123456789012:service/%s", name),
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"status":    "ACTIVE",
	}
	s.services[name] = item
	return item
}

func (s *protonStore) ensureEnvironmentLocked(name string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-environment"
	}
	if existing := s.environments[name]; existing != nil {
		return existing
	}
	item := map[string]any{
		"name":      name,
		"arn":       fmt.Sprintf("arn:aws:proton:us-east-1:123456789012:environment/%s", name),
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"status":    "ACTIVE",
	}
	s.environments[name] = item
	return item
}

func (s *protonStore) ensureServiceTemplateLocked(name string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-service-template"
	}
	if existing := s.serviceTemplates[name]; existing != nil {
		return existing
	}
	item := map[string]any{
		"name":      name,
		"arn":       fmt.Sprintf("arn:aws:proton:us-east-1:123456789012:service-template/%s", name),
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"status":    "ACTIVE",
	}
	s.serviceTemplates[name] = item
	return item
}

func (s *protonStore) ensureEnvironmentTemplateLocked(name string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-environment-template"
	}
	if existing := s.environmentTemplates[name]; existing != nil {
		return existing
	}
	item := map[string]any{
		"name":      name,
		"arn":       fmt.Sprintf("arn:aws:proton:us-east-1:123456789012:environment-template/%s", name),
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"status":    "ACTIVE",
	}
	s.environmentTemplates[name] = item
	return item
}

func (s *protonStore) ensureRepositoryLocked(name string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-repository"
	}
	if existing := s.repositories[name]; existing != nil {
		return existing
	}
	item := map[string]any{
		"name":      name,
		"arn":       fmt.Sprintf("arn:aws:proton:us-east-1:123456789012:repository/%s", name),
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"status":    "ACTIVE",
	}
	s.repositories[name] = item
	return item
}

func protonCloneAnyMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func protonDefaultStringAny(values map[string]any, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if trimmed := strings.TrimSpace(fmt.Sprintf("%v", v)); trimmed != "" && trimmed != "<nil>" {
				return trimmed
			}
		}
	}
	return fallback
}

func protonSortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
