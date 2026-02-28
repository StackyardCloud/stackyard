package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type licenseManagerUserSubscriptionsStore struct {
	mu sync.Mutex

	nextID int64

	identityProviders    map[string]map[string]any
	licenseServerEPs     map[string]map[string]any
	instances            map[string]map[string]any
	userAssociations     map[string]map[string]any
	productSubscriptions map[string]map[string]any
	tags                 map[string]map[string]string
	settings             map[string]any
}

func newLicenseManagerUserSubscriptionsStore() *licenseManagerUserSubscriptionsStore {
	s := &licenseManagerUserSubscriptionsStore{
		nextID:               2,
		identityProviders:    map[string]map[string]any{},
		licenseServerEPs:     map[string]map[string]any{},
		instances:            map[string]map[string]any{},
		userAssociations:     map[string]map[string]any{},
		productSubscriptions: map[string]map[string]any{},
		tags:                 map[string]map[string]string{},
		settings: map[string]any{
			"SecurityGroupId": "sg-0123456789abcdef0",
			"Subnets":         []any{"subnet-0123456789abcdef0"},
		},
	}

	idp := s.ensureIdentityProviderLocked(userSubIdentityProviderARN("stackyard-idp"))
	ep := s.ensureLicenseServerEndpointLocked("lse-000001")
	inst := s.ensureInstanceLocked("i-00000000000000001")
	a := s.ensureAssociationLocked("jdoe", inst["InstanceId"].(string), "VISUAL_STUDIO_ENTERPRISE")
	p := s.ensureProductSubscriptionLocked("jdoe", "VISUAL_STUDIO_ENTERPRISE", inst["InstanceId"].(string))

	s.tags[userSubIdentityProviderArn(idp)] = map[string]string{"stackyard": "true"}
	s.tags[userSubLicenseServerEndpointArn(ep)] = map[string]string{"stackyard": "true"}
	s.tags[userSubInstanceArn(inst)] = map[string]string{"stackyard": "true"}
	s.tags[userSubAssociationArn(a)] = map[string]string{"stackyard": "true"}
	s.tags[userSubProductArn(p)] = map[string]string{"stackyard": "true"}

	return s
}

func (s *licenseManagerUserSubscriptionsStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "RegisterIdentityProvider":
		arn := userSubDefaultStringAny(payload, "IdentityProviderArn", "")
		if arn == "" {
			arn = userSubIdentityProviderARN(fmt.Sprintf("stackyard-idp-%06d", s.nextIDLocked()))
		}
		idp := s.ensureIdentityProviderLocked(arn)
		for k, v := range payload {
			idp[k] = v
		}
		return map[string]any{"IdentityProvider": userSubCloneMap(idp)}

	case "DeregisterIdentityProvider":
		arn := userSubDefaultStringAny(payload, "IdentityProviderArn", userSubIdentityProviderARN("stackyard-idp"))
		delete(s.identityProviders, arn)
		return map[string]any{}

	case "ListIdentityProviders":
		keys := userSubSortedKeys(s.identityProviders)
		items := make([]any, 0, len(keys))
		for _, k := range keys {
			items = append(items, userSubCloneMap(s.identityProviders[k]))
		}
		return map[string]any{"IdentityProviderSummaries": items, "NextToken": ""}

	case "UpdateIdentityProviderSettings":
		arn := userSubDefaultStringAny(payload, "IdentityProviderArn", userSubIdentityProviderARN("stackyard-idp"))
		idp := s.ensureIdentityProviderLocked(arn)
		for k, v := range payload {
			idp[k] = v
		}
		if settings, ok := payload["UpdateSettings"]; ok {
			idp["Settings"] = settings
		}
		return map[string]any{"IdentityProvider": userSubCloneMap(idp)}

	case "CreateLicenseServerEndpoint":
		serverType := userSubDefaultStringAny(payload, "ServerType", "RDS_SAL")
		epID := fmt.Sprintf("lse-%06d", s.nextIDLocked())
		ep := s.ensureLicenseServerEndpointLocked(epID)
		ep["ServerType"] = serverType
		for k, v := range payload {
			ep[k] = v
		}
		return map[string]any{"LicenseServerEndpoint": userSubCloneMap(ep)}

	case "DeleteLicenseServerEndpoint":
		endpointArn := userSubDefaultStringAny(payload, "LicenseServerEndpointArn", userSubLicenseServerEndpointARN("lse-000001"))
		for key, v := range s.licenseServerEPs {
			if strings.EqualFold(userSubDefaultStringAny(v, "LicenseServerEndpointArn", ""), endpointArn) || strings.EqualFold(key, endpointArn) {
				delete(s.licenseServerEPs, key)
			}
		}
		return map[string]any{}

	case "ListLicenseServerEndpoints":
		keys := userSubSortedKeys(s.licenseServerEPs)
		items := make([]any, 0, len(keys))
		for _, k := range keys {
			items = append(items, userSubCloneMap(s.licenseServerEPs[k]))
		}
		return map[string]any{"LicenseServerEndpoints": items, "NextToken": ""}

	case "AssociateUser":
		username := userSubDefaultStringAny(payload, "Username", "jdoe")
		instanceID := userSubDefaultStringAny(payload, "InstanceId", "i-00000000000000001")
		product := userSubDefaultStringAny(payload, "Product", "VISUAL_STUDIO_ENTERPRISE")
		a := s.ensureAssociationLocked(username, instanceID, product)
		return map[string]any{"InstanceUserSummary": userSubCloneMap(a)}

	case "DisassociateUser":
		username := userSubDefaultStringAny(payload, "Username", "jdoe")
		instanceID := userSubDefaultStringAny(payload, "InstanceId", "i-00000000000000001")
		product := userSubDefaultStringAny(payload, "Product", "VISUAL_STUDIO_ENTERPRISE")
		key := userSubAssocKey(username, instanceID, product)
		delete(s.userAssociations, key)
		return map[string]any{}

	case "ListUserAssociations":
		keys := userSubSortedKeys(s.userAssociations)
		items := make([]any, 0, len(keys))
		for _, k := range keys {
			items = append(items, userSubCloneMap(s.userAssociations[k]))
		}
		return map[string]any{"InstanceUserSummaries": items, "NextToken": ""}

	case "StartProductSubscription":
		username := userSubDefaultStringAny(payload, "Username", "jdoe")
		product := userSubDefaultStringAny(payload, "Product", "VISUAL_STUDIO_ENTERPRISE")
		instanceID := userSubDefaultStringAny(payload, "InstanceId", "i-00000000000000001")
		p := s.ensureProductSubscriptionLocked(username, product, instanceID)
		p["Status"] = "ACTIVE"
		return map[string]any{"ProductUserSummary": userSubCloneMap(p)}

	case "StopProductSubscription":
		username := userSubDefaultStringAny(payload, "Username", "jdoe")
		product := userSubDefaultStringAny(payload, "Product", "VISUAL_STUDIO_ENTERPRISE")
		instanceID := userSubDefaultStringAny(payload, "InstanceId", "i-00000000000000001")
		p := s.ensureProductSubscriptionLocked(username, product, instanceID)
		p["Status"] = "STOPPED"
		return map[string]any{"ProductUserSummary": userSubCloneMap(p)}

	case "ListProductSubscriptions":
		keys := userSubSortedKeys(s.productSubscriptions)
		items := make([]any, 0, len(keys))
		for _, k := range keys {
			items = append(items, userSubCloneMap(s.productSubscriptions[k]))
		}
		return map[string]any{"ProductUserSummaries": items, "NextToken": ""}

	case "ListInstances":
		keys := userSubSortedKeys(s.instances)
		items := make([]any, 0, len(keys))
		for _, k := range keys {
			items = append(items, userSubCloneMap(s.instances[k]))
		}
		return map[string]any{"InstanceSummaries": items, "NextToken": ""}

	case "ListTagsForResource":
		resourceArn := userSubDefaultString(pathParams, "ResourceArn", userSubResourceArn("stackyard-resource"))
		return map[string]any{"Tags": userSubCloneStringMap(s.tags[resourceArn])}

	case "TagResource":
		resourceArn := userSubDefaultString(pathParams, "ResourceArn", userSubResourceArn("stackyard-resource"))
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
		resourceArn := userSubDefaultString(pathParams, "ResourceArn", userSubResourceArn("stackyard-resource"))
		if keysRaw, ok := payload["TagKeys"].([]any); ok {
			for _, key := range keysRaw {
				delete(s.tags[resourceArn], strings.TrimSpace(fmt.Sprintf("%v", key)))
			}
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *licenseManagerUserSubscriptionsStore) ensureIdentityProviderLocked(arn string) map[string]any {
	key := strings.TrimSpace(arn)
	if key == "" {
		key = userSubIdentityProviderARN(fmt.Sprintf("stackyard-idp-%06d", s.nextIDLocked()))
	}
	if v := s.identityProviders[key]; v != nil {
		return v
	}
	v := map[string]any{
		"IdentityProviderArn": key,
		"Product":             "VISUAL_STUDIO_ENTERPRISE",
		"Settings":            map[string]any{"DomainName": "stackyard.local"},
		"LastUpdatedDate":     time.Now().UTC(),
		"FailureMessage":      "",
		"Status":              "REGISTERED",
		"IdentityProvider":    "ACTIVE_DIRECTORY",
	}
	s.identityProviders[key] = v
	return v
}

func (s *licenseManagerUserSubscriptionsStore) ensureLicenseServerEndpointLocked(id string) map[string]any {
	key := strings.TrimSpace(id)
	if key == "" {
		key = fmt.Sprintf("lse-%06d", s.nextIDLocked())
	}
	if v := s.licenseServerEPs[key]; v != nil {
		return v
	}
	v := map[string]any{
		"IdentityProviderArn":      userSubIdentityProviderARN("stackyard-idp"),
		"LicenseServerEndpointArn": userSubLicenseServerEndpointARN(key),
		"ServerType":               "RDS_SAL",
		"Status":                   "PROVISIONED",
		"Endpoint":                 fmt.Sprintf("%s.stackyard.local", key),
		"CreationTime":             time.Now().UTC(),
	}
	s.licenseServerEPs[key] = v
	return v
}

func (s *licenseManagerUserSubscriptionsStore) ensureInstanceLocked(instanceID string) map[string]any {
	key := strings.TrimSpace(instanceID)
	if key == "" {
		key = "i-00000000000000001"
	}
	if v := s.instances[key]; v != nil {
		return v
	}
	v := map[string]any{
		"InstanceId":          key,
		"Status":              "ACTIVE",
		"LastStatusCheckDate": time.Now().UTC(),
		"Products":            []any{"VISUAL_STUDIO_ENTERPRISE"},
		"IdentityProvider":    userSubIdentityProviderARN("stackyard-idp"),
	}
	s.instances[key] = v
	return v
}

func (s *licenseManagerUserSubscriptionsStore) ensureAssociationLocked(username, instanceID, product string) map[string]any {
	key := userSubAssocKey(username, instanceID, product)
	if v := s.userAssociations[key]; v != nil {
		return v
	}
	v := map[string]any{
		"Username":           username,
		"InstanceId":         instanceID,
		"AssociationDate":    time.Now().UTC(),
		"DisassociationDate": nil,
		"IdentityProvider":   userSubIdentityProviderARN("stackyard-idp"),
		"Status":             "ASSOCIATED",
		"Domain":             "stackyard.local",
		"Product":            product,
	}
	s.userAssociations[key] = v
	return v
}

func (s *licenseManagerUserSubscriptionsStore) ensureProductSubscriptionLocked(username, product, instanceID string) map[string]any {
	key := userSubAssocKey(username, instanceID, product)
	if v := s.productSubscriptions[key]; v != nil {
		return v
	}
	v := map[string]any{
		"Username":              username,
		"Product":               product,
		"Status":                "ACTIVE",
		"SubscriptionStartDate": time.Now().UTC().Add(-1 * time.Hour),
		"SubscriptionEndDate":   nil,
		"IdentityProvider":      userSubIdentityProviderARN("stackyard-idp"),
		"InstanceId":            instanceID,
		"Domain":                "stackyard.local",
	}
	s.productSubscriptions[key] = v
	return v
}

func (s *licenseManagerUserSubscriptionsStore) nextIDLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func userSubIdentityProviderARN(name string) string {
	k := strings.TrimSpace(name)
	if k == "" {
		k = "stackyard-idp"
	}
	if strings.HasPrefix(k, "arn:") {
		return k
	}
	return fmt.Sprintf("arn:aws:license-manager-user-subscriptions:us-east-1:123456789012:identity-provider/%s", k)
}

func userSubLicenseServerEndpointARN(name string) string {
	k := strings.TrimSpace(name)
	if k == "" {
		k = "lse-000001"
	}
	if strings.HasPrefix(k, "arn:") {
		return k
	}
	return fmt.Sprintf("arn:aws:license-manager-user-subscriptions:us-east-1:123456789012:license-server-endpoint/%s", k)
}

func userSubInstanceArn(instance map[string]any) string {
	id := userSubDefaultStringAny(instance, "InstanceId", "i-00000000000000001")
	return fmt.Sprintf("arn:aws:ec2:us-east-1:123456789012:instance/%s", id)
}

func userSubAssociationArn(assoc map[string]any) string {
	username := userSubDefaultStringAny(assoc, "Username", "jdoe")
	instance := userSubDefaultStringAny(assoc, "InstanceId", "i-00000000000000001")
	product := strings.ToLower(userSubDefaultStringAny(assoc, "Product", "visual_studio_enterprise"))
	return fmt.Sprintf("arn:aws:license-manager-user-subscriptions:us-east-1:123456789012:user-association/%s/%s/%s", username, instance, product)
}

func userSubProductArn(product map[string]any) string {
	username := userSubDefaultStringAny(product, "Username", "jdoe")
	instance := userSubDefaultStringAny(product, "InstanceId", "i-00000000000000001")
	prod := strings.ToLower(userSubDefaultStringAny(product, "Product", "visual_studio_enterprise"))
	return fmt.Sprintf("arn:aws:license-manager-user-subscriptions:us-east-1:123456789012:product-subscription/%s/%s/%s", username, instance, prod)
}

func userSubResourceArn(name string) string {
	k := strings.TrimSpace(name)
	if k == "" {
		k = "stackyard-resource"
	}
	return fmt.Sprintf("arn:aws:license-manager-user-subscriptions:us-east-1:123456789012:resource/%s", k)
}

func userSubLicenseServerEndpointArn(endpoint map[string]any) string {
	return userSubDefaultStringAny(endpoint, "LicenseServerEndpointArn", userSubLicenseServerEndpointARN("lse-000001"))
}

func userSubIdentityProviderArn(identityProvider map[string]any) string {
	return userSubDefaultStringAny(identityProvider, "IdentityProviderArn", userSubIdentityProviderARN("stackyard-idp"))
}

func userSubAssocKey(username, instanceID, product string) string {
	return strings.ToLower(strings.TrimSpace(username) + "|" + strings.TrimSpace(instanceID) + "|" + strings.TrimSpace(product))
}

func userSubSortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func userSubDefaultString(values map[string]string, key, fallback string) string {
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

func userSubDefaultStringAny(values map[string]any, key, fallback string) string {
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

func userSubCloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func userSubCloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
