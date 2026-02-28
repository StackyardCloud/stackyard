package server

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type licenseManagerStore struct {
	mu          sync.Mutex
	nextID      int64
	resourceTag map[string]map[string]string
}

func newLicenseManagerStore() *licenseManagerStore {
	seedArn := "arn:aws:license-manager:us-east-1:123456789012:license-configuration:lic-000001"
	return &licenseManagerStore{
		nextID: 2,
		resourceTag: map[string]map[string]string{
			seedArn: {"stackyard": "true"},
		},
	}
}

func (s *licenseManagerStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "GetServiceSettings":
		return map[string]any{
			"S3BucketArn":                    "arn:aws:s3:::stackyard-license-manager",
			"SnsTopicArn":                    "arn:aws:sns:us-east-1:123456789012:stackyard-license-manager",
			"OrganizationConfiguration":      map[string]any{"EnableIntegration": true},
			"EnableCrossAccountsDiscovery":   true,
			"LicenseManagerResourceShareArn": "arn:aws:ram:us-east-1:123456789012:resource-share/stackyard-license-manager",
		}
	case "ListLicenses":
		return map[string]any{
			"Licenses": []any{map[string]any{
				"LicenseArn":  "arn:aws:license-manager:us-east-1:123456789012:license:l-000001",
				"LicenseName": "stackyard-license",
				"Status":      "AVAILABLE",
			}},
			"NextToken": "",
		}
	case "CreateLicenseConfiguration":
		id := s.nextIDLocked()
		arn := fmt.Sprintf("arn:aws:license-manager:us-east-1:123456789012:license-configuration:lic-%06d", id)
		s.resourceTag[arn] = map[string]string{"stackyard": "true"}
		return map[string]any{"LicenseConfigurationArn": arn}
	case "GetLicenseConfiguration":
		arn := licenseManagerPayloadString(payload, "LicenseConfigurationArn", "arn:aws:license-manager:us-east-1:123456789012:license-configuration:lic-000001")
		return map[string]any{
			"LicenseConfiguration": map[string]any{
				"LicenseConfigurationArn": arn,
				"Name":                    "stackyard-license-config",
				"Description":             "seeded by Stackyard",
				"LicenseCountingType":     "vCPU",
				"Status":                  "AVAILABLE",
				"ConsumedLicenses":        0,
				"CreatedTime":             now,
			},
		}
	case "CreateLicense":
		id := s.nextIDLocked()
		return map[string]any{
			"LicenseArn": fmt.Sprintf("arn:aws:license-manager:us-east-1:123456789012:license:l-%06d", id),
			"Status":     "AVAILABLE",
			"Version":    "1",
		}
	case "CreateToken":
		id := s.nextIDLocked()
		return map[string]any{
			"TokenId":   fmt.Sprintf("token-%06d", id),
			"TokenType": licenseManagerPayloadString(payload, "TokenType", "REFRESH_TOKEN"),
			"Token":     fmt.Sprintf("stackyard-token-%06d", id),
		}
	case "GetAccessToken":
		return map[string]any{
			"AccessToken": "stackyard-access-token",
			"ExpiresAt":   now,
		}
	case "CheckoutLicense", "CheckoutBorrowLicense":
		return map[string]any{
			"CheckoutType":            "PROVISIONAL",
			"LicenseConsumptionToken": fmt.Sprintf("consumption-%06d", s.nextIDLocked()),
			"EntitlementsAllowed":     []any{},
			"IssuedAt":                now,
			"Expiration":              now,
		}
	case "ExtendLicenseConsumption":
		return map[string]any{
			"LicenseConsumptionToken": fmt.Sprintf("consumption-%06d", s.nextIDLocked()),
			"Expiration":              now,
		}
	case "ListTagsForResource":
		arn := licenseManagerPayloadString(payload, "ResourceArn", "arn:aws:license-manager:us-east-1:123456789012:license-configuration:lic-000001")
		tags := []any{}
		for k, v := range s.resourceTag[arn] {
			tags = append(tags, map[string]any{"Key": k, "Value": v})
		}
		return map[string]any{"Tags": tags}
	case "TagResource":
		arn := licenseManagerPayloadString(payload, "ResourceArn", "arn:aws:license-manager:us-east-1:123456789012:license-configuration:lic-000001")
		if s.resourceTag[arn] == nil {
			s.resourceTag[arn] = map[string]string{}
		}
		if rawTags, ok := payload["Tags"]; ok {
			switch tags := rawTags.(type) {
			case map[string]any:
				for k, v := range tags {
					s.resourceTag[arn][k] = fmt.Sprintf("%v", v)
				}
			case []any:
				for _, item := range tags {
					m, ok := item.(map[string]any)
					if !ok {
						continue
					}
					k := licenseManagerPayloadString(m, "Key", "")
					if strings.TrimSpace(k) == "" {
						continue
					}
					s.resourceTag[arn][k] = licenseManagerPayloadString(m, "Value", "")
				}
			}
		}
		return map[string]any{}
	case "UntagResource":
		arn := licenseManagerPayloadString(payload, "ResourceArn", "arn:aws:license-manager:us-east-1:123456789012:license-configuration:lic-000001")
		if keysRaw, ok := payload["TagKeys"]; ok {
			if keys, ok := keysRaw.([]any); ok {
				for _, key := range keys {
					delete(s.resourceTag[arn], strings.TrimSpace(fmt.Sprintf("%v", key)))
				}
			}
		}
		return map[string]any{}
	}

	if strings.HasPrefix(action, "List") {
		key := licenseManagerListKey(action)
		if key == "" {
			key = "Items"
		}
		return map[string]any{key: []any{}, "NextToken": ""}
	}

	if strings.HasPrefix(action, "Get") {
		key := strings.TrimPrefix(action, "Get")
		if key == "" {
			key = "Result"
		}
		return map[string]any{key: map[string]any{}}
	}

	if strings.HasPrefix(action, "Create") {
		resource := strings.TrimPrefix(action, "Create")
		if resource == "" {
			resource = "Resource"
		}
		id := s.nextIDLocked()
		arn := fmt.Sprintf("arn:aws:license-manager:us-east-1:123456789012:%s/%06d", strings.ToLower(resource), id)
		return map[string]any{resource + "Arn": arn}
	}

	if strings.HasPrefix(action, "Update") ||
		strings.HasPrefix(action, "Delete") ||
		strings.HasPrefix(action, "Accept") ||
		strings.HasPrefix(action, "Reject") ||
		strings.HasPrefix(action, "Check") {
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *licenseManagerStore) nextIDLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func licenseManagerListKey(action string) string {
	key := strings.TrimPrefix(action, "List")
	key = strings.TrimSuffix(key, "ForOrganization")
	if key == "" {
		return "Items"
	}
	return key
}

func licenseManagerPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" && s != "%!v(<nil>)" {
				return s
			}
		}
	}
	return fallback
}
