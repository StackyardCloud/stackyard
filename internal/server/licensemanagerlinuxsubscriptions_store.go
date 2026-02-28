package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type licenseManagerLinuxSubscriptionsStore struct {
	mu sync.Mutex

	nextID int64

	providers     map[string]map[string]any
	subscriptions map[string]map[string]any
	instances     map[string]map[string]any
	settings      map[string]any
	tags          map[string]map[string]string
}

func newLicenseManagerLinuxSubscriptionsStore() *licenseManagerLinuxSubscriptionsStore {
	s := &licenseManagerLinuxSubscriptionsStore{
		nextID:        2,
		providers:     map[string]map[string]any{},
		subscriptions: map[string]map[string]any{},
		instances:     map[string]map[string]any{},
		settings: map[string]any{
			"LinuxSubscriptionsDiscovery": "ENABLED",
			"LinuxSubscriptionsDiscoverySettings": map[string]any{
				"OrganizationIntegration": "ENABLED",
				"SourceRegions":           []any{"us-east-1"},
			},
		},
		tags: map[string]map[string]string{},
	}

	provider := s.ensureProviderLocked("stackyard-provider")
	sub := s.ensureSubscriptionLocked("sub-000001")
	inst := s.ensureInstanceLocked("i-00000000000000001")
	s.tags[providerArn(provider)] = map[string]string{"stackyard": "true"}
	s.tags[subscriptionArn(sub)] = map[string]string{"stackyard": "true"}
	s.tags[instanceArn(inst)] = map[string]string{"stackyard": "true"}

	return s
}

func (s *licenseManagerLinuxSubscriptionsStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "RegisterSubscriptionProvider":
		name := lmlsDefaultStringAny(payload, "SubscriptionProviderArn", "")
		if name == "" {
			name = fmt.Sprintf("stackyard-provider-%06d", s.nextIDLocked())
		}
		provider := s.ensureProviderLocked(name)
		return map[string]any{"RegisteredSubscriptionProvider": lmlsCloneMap(provider)}

	case "GetRegisteredSubscriptionProvider":
		name := lmlsDefaultStringAny(payload, "SubscriptionProviderArn", "stackyard-provider")
		provider := s.ensureProviderLocked(name)
		return map[string]any{"RegisteredSubscriptionProvider": lmlsCloneMap(provider)}

	case "ListRegisteredSubscriptionProviders":
		keys := make([]string, 0, len(s.providers))
		for key := range s.providers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		providers := make([]any, 0, len(keys))
		for _, key := range keys {
			providers = append(providers, lmlsCloneMap(s.providers[key]))
		}
		return map[string]any{"RegisteredSubscriptionProviders": providers, "NextToken": ""}

	case "DeregisterSubscriptionProvider":
		name := lmlsDefaultStringAny(payload, "SubscriptionProviderArn", "stackyard-provider")
		delete(s.providers, name)
		return map[string]any{}

	case "GetServiceSettings":
		return map[string]any{"ServiceSettings": lmlsCloneMapAny(s.settings)}

	case "UpdateServiceSettings":
		for key, value := range payload {
			s.settings[key] = value
		}
		return map[string]any{"ServiceSettings": lmlsCloneMapAny(s.settings)}

	case "ListLinuxSubscriptions":
		keys := make([]string, 0, len(s.subscriptions))
		for key := range s.subscriptions {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		subs := make([]any, 0, len(keys))
		for _, key := range keys {
			subs = append(subs, lmlsCloneMap(s.subscriptions[key]))
		}
		return map[string]any{"Subscriptions": subs, "NextToken": ""}

	case "ListLinuxSubscriptionInstances":
		keys := make([]string, 0, len(s.instances))
		for key := range s.instances {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		instances := make([]any, 0, len(keys))
		for _, key := range keys {
			instances = append(instances, lmlsCloneMap(s.instances[key]))
		}
		return map[string]any{"Instances": instances, "NextToken": ""}

	case "ListTagsForResource":
		resourceArn := lmlsDefaultString(pathParams, "resourceArn", lmlsResourceArn("stackyard-resource"))
		return map[string]any{"Tags": lmlsCloneStringMap(s.tags[resourceArn])}

	case "TagResource":
		resourceArn := lmlsDefaultString(pathParams, "resourceArn", lmlsResourceArn("stackyard-resource"))
		if s.tags[resourceArn] == nil {
			s.tags[resourceArn] = map[string]string{}
		}
		switch tagsRaw := payload["Tags"].(type) {
		case map[string]any:
			for k, v := range tagsRaw {
				s.tags[resourceArn][k] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		case map[string]string:
			for k, v := range tagsRaw {
				s.tags[resourceArn][k] = v
			}
		}
		return map[string]any{}

	case "UntagResource":
		resourceArn := lmlsDefaultString(pathParams, "resourceArn", lmlsResourceArn("stackyard-resource"))
		if keysRaw, ok := payload["TagKeys"].([]any); ok {
			for _, key := range keysRaw {
				delete(s.tags[resourceArn], strings.TrimSpace(fmt.Sprintf("%v", key)))
			}
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *licenseManagerLinuxSubscriptionsStore) ensureProviderLocked(key string) map[string]any {
	k := strings.TrimSpace(key)
	if k == "" {
		k = fmt.Sprintf("stackyard-provider-%06d", s.nextIDLocked())
	}
	if provider := s.providers[k]; provider != nil {
		return provider
	}
	provider := map[string]any{
		"SubscriptionProviderArn":    k,
		"SubscriptionProviderSource": "AWS",
		"SecretArn":                  "arn:aws:secretsmanager:us-east-1:123456789012:secret:stackyard",
		"RegisteredAt":               time.Now().UTC(),
		"Health":                     "HEALTHY",
	}
	s.providers[k] = provider
	return provider
}

func (s *licenseManagerLinuxSubscriptionsStore) ensureSubscriptionLocked(key string) map[string]any {
	k := strings.TrimSpace(key)
	if k == "" {
		k = fmt.Sprintf("sub-%06d", s.nextIDLocked())
	}
	if sub := s.subscriptions[k]; sub != nil {
		return sub
	}
	sub := map[string]any{
		"SubscriptionArn":         subscriptionArnByKey(k),
		"InstanceType":            "m6i.large",
		"OperatingSystem":         "RHEL",
		"UsageOperation":          "RunInstances",
		"Status":                  "ACTIVE",
		"SubscriptionProviderArn": "stackyard-provider",
	}
	s.subscriptions[k] = sub
	return sub
}

func (s *licenseManagerLinuxSubscriptionsStore) ensureInstanceLocked(instanceID string) map[string]any {
	k := strings.TrimSpace(instanceID)
	if k == "" {
		k = "i-00000000000000001"
	}
	if inst := s.instances[k]; inst != nil {
		return inst
	}
	inst := map[string]any{
		"InstanceId":                         k,
		"InstanceType":                       "m6i.large",
		"Region":                             "us-east-1",
		"LastUpdatedTime":                    time.Now().UTC(),
		"SubscriptionProviderArn":            "stackyard-provider",
		"AmiId":                              "ami-0123456789abcdef0",
		"RegisteredWithSubscriptionProvider": true,
	}
	s.instances[k] = inst
	return inst
}

func (s *licenseManagerLinuxSubscriptionsStore) nextIDLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func providerArn(provider map[string]any) string {
	return lmlsDefaultStringAny(provider, "SubscriptionProviderArn", "stackyard-provider")
}

func subscriptionArn(sub map[string]any) string {
	return lmlsDefaultStringAny(sub, "SubscriptionArn", subscriptionArnByKey("sub-000001"))
}

func instanceArn(instance map[string]any) string {
	id := lmlsDefaultStringAny(instance, "InstanceId", "i-00000000000000001")
	return fmt.Sprintf("arn:aws:ec2:us-east-1:123456789012:instance/%s", id)
}

func subscriptionArnByKey(key string) string {
	k := strings.TrimSpace(key)
	if k == "" {
		k = "sub-000001"
	}
	if strings.HasPrefix(k, "arn:") {
		return k
	}
	return fmt.Sprintf("arn:aws:license-manager-linux-subscriptions:us-east-1:123456789012:subscription/%s", k)
}

func lmlsResourceArn(name string) string {
	k := strings.TrimSpace(name)
	if k == "" {
		k = "stackyard-resource"
	}
	return fmt.Sprintf("arn:aws:license-manager-linux-subscriptions:us-east-1:123456789012:resource/%s", k)
}

func lmlsDefaultString(values map[string]string, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(k, key) {
			trimmed := strings.TrimSpace(v)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}

func lmlsDefaultStringAny(values map[string]any, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(k, key) {
			trimmed := strings.TrimSpace(fmt.Sprintf("%v", v))
			if trimmed != "" && trimmed != "<nil>" {
				return trimmed
			}
		}
	}
	return fallback
}

func lmlsCloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func lmlsCloneMapAny(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func lmlsCloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
