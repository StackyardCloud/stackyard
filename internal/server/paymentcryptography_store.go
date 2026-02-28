package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type paymentCryptographyStore struct {
	mu sync.Mutex

	nextID int

	keys    map[string]*paymentKey
	aliases map[string]string
	tags    map[string]map[string]string
}

type paymentKey struct {
	ARN         string
	Usage       string
	Class       string
	Algorithm   string
	Enabled     bool
	Exportable  bool
	State       string
	Origin      string
	CheckValue  string
	CheckAlgo   string
	CreatedTime string
}

func newPaymentCryptographyStore() *paymentCryptographyStore {
	s := &paymentCryptographyStore{
		nextID:  1,
		keys:    map[string]*paymentKey{},
		aliases: map[string]string{},
		tags:    map[string]map[string]string{},
	}

	seed := &paymentKey{
		ARN:         paymentCryptographyKeyARN("stackyard-key"),
		Usage:       "TR31_B0_BASE_DERIVATION_KEY",
		Class:       "SYMMETRIC_KEY",
		Algorithm:   "TDES_2KEY",
		Enabled:     true,
		Exportable:  true,
		State:       "CREATE_COMPLETE",
		Origin:      "AWS_PAYMENT_CRYPTOGRAPHY",
		CheckValue:  "000000",
		CheckAlgo:   "ANSI_X9_24",
		CreatedTime: time.Now().UTC().Format(time.RFC3339),
	}
	s.keys[seed.ARN] = seed
	s.aliases["alias/stackyard"] = seed.ARN
	s.tags[seed.ARN] = map[string]string{"env": "dev"}

	return s
}

func (s *paymentCryptographyStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, known := paymentCryptographyOperationByName[action]; !known {
		return map[string]any{}
	}

	switch action {
	case "CreateKey":
		key := s.createKey(payload)
		return map[string]any{"Key": s.keyOutput(key)}
	case "ImportKey":
		key := s.createKey(payload)
		key.Origin = "EXTERNAL"
		return map[string]any{"Key": s.keyOutput(key)}
	case "GetKey", "DeleteKey", "RestoreKey", "StartKeyUsage", "StopKeyUsage":
		key := s.getOrCreateKeyForPayload(payload)
		s.mutateKeyForAction(key, action)
		return map[string]any{"Key": s.keyOutput(key)}
	case "ListKeys":
		items := make([]any, 0, len(s.keys))
		for _, key := range s.sortedKeys() {
			items = append(items, map[string]any{
				"KeyArn":        key.ARN,
				"KeyState":      key.State,
				"KeyAttributes": s.keyAttributesOutput(key),
				"KeyCheckValue": key.CheckValue,
				"Exportable":    key.Exportable,
				"Enabled":       key.Enabled,
			})
		}
		return map[string]any{"Keys": items}

	case "CreateAlias":
		alias := paymentCryptographyPayloadString(payload, "AliasName", "alias/stackyard")
		keyARN := paymentCryptographyPayloadString(payload, "KeyArn", paymentCryptographyKeyARN("stackyard-key"))
		s.aliases[alias] = keyARN
		return map[string]any{"Alias": map[string]any{"AliasName": alias}}
	case "GetAlias", "UpdateAlias":
		alias := paymentCryptographyPayloadString(payload, "AliasName", "alias/stackyard")
		if action == "UpdateAlias" {
			keyARN := paymentCryptographyPayloadString(payload, "KeyArn", paymentCryptographyKeyARN("stackyard-key"))
			s.aliases[alias] = keyARN
		}
		if _, ok := s.aliases[alias]; !ok {
			s.aliases[alias] = paymentCryptographyKeyARN("stackyard-key")
		}
		return map[string]any{"Alias": map[string]any{"AliasName": alias}}
	case "DeleteAlias":
		alias := paymentCryptographyPayloadString(payload, "AliasName", "alias/stackyard")
		delete(s.aliases, alias)
		return map[string]any{}
	case "ListAliases":
		items := make([]any, 0, len(s.aliases))
		for _, alias := range s.sortedAliases() {
			items = append(items, map[string]any{"AliasName": alias})
		}
		return map[string]any{"Aliases": items}

	case "GetParametersForImport":
		return map[string]any{
			"WrappingKeyCertificate":        paymentCryptographyCert(),
			"WrappingKeyCertificateChain":   paymentCryptographyCert(),
			"WrappingKeyAlgorithm":          "RSA_3072",
			"ImportToken":                   "import-token-stackyard",
			"ParametersValidUntilTimestamp": time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
		}
	case "GetParametersForExport":
		return map[string]any{
			"SigningKeyCertificate":         paymentCryptographyCert(),
			"SigningKeyCertificateChain":    paymentCryptographyCert(),
			"SigningKeyAlgorithm":           "RSA_3072",
			"ExportToken":                   "export-token-stackyard",
			"ParametersValidUntilTimestamp": time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
		}
	case "GetPublicKeyCertificate":
		return map[string]any{
			"KeyCertificate":      paymentCryptographyCert(),
			"KeyCertificateChain": paymentCryptographyCert(),
		}

	case "TagResource":
		arn := paymentCryptographyPayloadString(payload, "ResourceArn", paymentCryptographyKeyARN("stackyard-key"))
		tags := s.ensureTagsLocked(arn)
		for _, kv := range paymentCryptographyPayloadTags(payload) {
			tags[kv["Key"]] = kv["Value"]
		}
		return map[string]any{}
	case "ListTagsForResource":
		arn := paymentCryptographyPayloadString(payload, "ResourceArn", paymentCryptographyKeyARN("stackyard-key"))
		tags := s.ensureTagsLocked(arn)
		keys := make([]string, 0, len(tags))
		for k := range tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]any, 0, len(keys))
		for _, k := range keys {
			out = append(out, map[string]any{"Key": k, "Value": tags[k]})
		}
		return map[string]any{"Tags": out}
	case "UntagResource":
		arn := paymentCryptographyPayloadString(payload, "ResourceArn", paymentCryptographyKeyARN("stackyard-key"))
		tags := s.ensureTagsLocked(arn)
		for _, key := range paymentCryptographyPayloadTagKeys(payload) {
			delete(tags, key)
		}
		return map[string]any{}

	default:
		return map[string]any{}
	}
}

func (s *paymentCryptographyStore) createKey(payload map[string]any) *paymentKey {
	s.nextID++
	name := fmt.Sprintf("stackyard-key-%06d", s.nextID)
	key := &paymentKey{
		ARN:         paymentCryptographyKeyARN(name),
		Usage:       paymentCryptographyNestedString(payload, []string{"KeyAttributes", "KeyUsage"}, "TR31_B0_BASE_DERIVATION_KEY"),
		Class:       paymentCryptographyNestedString(payload, []string{"KeyAttributes", "KeyClass"}, "SYMMETRIC_KEY"),
		Algorithm:   paymentCryptographyNestedString(payload, []string{"KeyAttributes", "KeyAlgorithm"}, "TDES_2KEY"),
		Enabled:     true,
		Exportable:  true,
		State:       "CREATE_COMPLETE",
		Origin:      "AWS_PAYMENT_CRYPTOGRAPHY",
		CheckValue:  "000000",
		CheckAlgo:   "ANSI_X9_24",
		CreatedTime: time.Now().UTC().Format(time.RFC3339),
	}
	if e, ok := paymentCryptographyPayloadBool(payload, "Enabled"); ok {
		key.Enabled = e
	}
	if ex, ok := paymentCryptographyPayloadBool(payload, "Exportable"); ok {
		key.Exportable = ex
	}
	s.keys[key.ARN] = key
	if _, ok := s.tags[key.ARN]; !ok {
		s.tags[key.ARN] = map[string]string{}
	}
	return key
}

func (s *paymentCryptographyStore) getOrCreateKeyForPayload(payload map[string]any) *paymentKey {
	for _, keyName := range []string{"KeyIdentifier", "KeyArn", "ResourceArn"} {
		arn := paymentCryptographyPayloadString(payload, keyName, "")
		if arn != "" {
			if key, ok := s.keys[arn]; ok {
				return key
			}
			name := paymentCryptographyNameFromARN(arn, fmt.Sprintf("imported-%06d", s.nextID))
			key := &paymentKey{
				ARN:         paymentCryptographyKeyARN(name),
				Usage:       "TR31_B0_BASE_DERIVATION_KEY",
				Class:       "SYMMETRIC_KEY",
				Algorithm:   "TDES_2KEY",
				Enabled:     true,
				Exportable:  true,
				State:       "CREATE_COMPLETE",
				Origin:      "AWS_PAYMENT_CRYPTOGRAPHY",
				CheckValue:  "000000",
				CheckAlgo:   "ANSI_X9_24",
				CreatedTime: time.Now().UTC().Format(time.RFC3339),
			}
			s.keys[key.ARN] = key
			return key
		}
	}
	if key, ok := s.keys[paymentCryptographyKeyARN("stackyard-key")]; ok {
		return key
	}
	return s.createKey(map[string]any{})
}

func (s *paymentCryptographyStore) mutateKeyForAction(key *paymentKey, action string) {
	switch action {
	case "DeleteKey":
		key.State = "DELETE_PENDING"
	case "RestoreKey", "StartKeyUsage":
		key.State = "CREATE_COMPLETE"
		key.Enabled = true
	case "StopKeyUsage":
		key.Enabled = false
		key.State = "CREATE_COMPLETE"
	}
}

func (s *paymentCryptographyStore) keyOutput(key *paymentKey) map[string]any {
	return map[string]any{
		"KeyArn":                 key.ARN,
		"KeyAttributes":          s.keyAttributesOutput(key),
		"KeyCheckValue":          key.CheckValue,
		"KeyCheckValueAlgorithm": key.CheckAlgo,
		"Enabled":                key.Enabled,
		"Exportable":             key.Exportable,
		"KeyState":               key.State,
		"KeyOrigin":              key.Origin,
		"CreateTimestamp":        key.CreatedTime,
	}
}

func (s *paymentCryptographyStore) keyAttributesOutput(key *paymentKey) map[string]any {
	return map[string]any{
		"KeyUsage":      key.Usage,
		"KeyClass":      key.Class,
		"KeyAlgorithm":  key.Algorithm,
		"KeyModesOfUse": map[string]any{},
	}
}

func (s *paymentCryptographyStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = paymentCryptographyKeyARN("stackyard-key")
	}
	if existing, ok := s.tags[resourceARN]; ok {
		return existing
	}
	created := map[string]string{}
	s.tags[resourceARN] = created
	return created
}

func (s *paymentCryptographyStore) sortedKeys() []*paymentKey {
	keys := make([]string, 0, len(s.keys))
	for arn := range s.keys {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]*paymentKey, 0, len(keys))
	for _, arn := range keys {
		out = append(out, s.keys[arn])
	}
	return out
}

func (s *paymentCryptographyStore) sortedAliases() []string {
	aliases := make([]string, 0, len(s.aliases))
	for alias := range s.aliases {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

func paymentCryptographyPayloadString(payload map[string]any, key, fallback string) string {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		if s, ok := v.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return fallback
}

func paymentCryptographyPayloadBool(payload map[string]any, key string) (bool, bool) {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		b, ok := v.(bool)
		if ok {
			return b, true
		}
	}
	return false, false
}

func paymentCryptographyNestedString(payload map[string]any, path []string, fallback string) string {
	if len(path) == 0 {
		return fallback
	}
	var cur any = payload
	for _, p := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return fallback
		}
		next, ok := m[p]
		if !ok {
			for k, v := range m {
				if strings.EqualFold(k, p) {
					next = v
					ok = true
					break
				}
			}
		}
		if !ok {
			return fallback
		}
		cur = next
	}
	s, ok := cur.(string)
	if !ok {
		return fallback
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

func paymentCryptographyPayloadTags(payload map[string]any) []map[string]string {
	raw, ok := payload["Tags"]
	if !ok {
		raw = payload["tags"]
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]string, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		k := strings.TrimSpace(fmt.Sprintf("%v", m["Key"]))
		v := strings.TrimSpace(fmt.Sprintf("%v", m["Value"]))
		if k == "" {
			continue
		}
		out = append(out, map[string]string{"Key": k, "Value": v})
	}
	return out
}

func paymentCryptographyPayloadTagKeys(payload map[string]any) []string {
	raw, ok := payload["TagKeys"]
	if !ok {
		raw = payload["tagKeys"]
	}
	list, ok := raw.([]any)
	if !ok {
		if strList, ok2 := raw.([]string); ok2 {
			out := make([]string, 0, len(strList))
			for _, item := range strList {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
			return out
		}
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		key := strings.TrimSpace(fmt.Sprintf("%v", item))
		if key != "" {
			out = append(out, key)
		}
	}
	return out
}

func paymentCryptographyKeyARN(name string) string {
	return fmt.Sprintf("arn:aws:payment-cryptography:us-east-1:123456789012:key/%s", strings.TrimSpace(name))
}

func paymentCryptographyNameFromARN(arn, fallback string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return fallback
	}
	parts := strings.Split(arn, "/")
	if len(parts) == 0 {
		return fallback
	}
	name := strings.TrimSpace(parts[len(parts)-1])
	if name == "" {
		return fallback
	}
	return name
}

func paymentCryptographyCert() string {
	return "-----BEGIN CERTIFICATE-----\\nU1RBQ0tZQVJE\\n-----END CERTIFICATE-----"
}
