package server

import (
	"strings"
	"sync"
)

type paymentCryptographyDataStore struct {
	mu sync.Mutex

	defaultKeyARN string
}

func newPaymentCryptographyDataStore() *paymentCryptographyDataStore {
	return &paymentCryptographyDataStore{
		defaultKeyARN: "arn:aws:payment-cryptography:us-east-1:123456789012:key/stackyard-key",
	}
}

func (s *paymentCryptographyDataStore) Handle(action string, payload, pathParams map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, known := paymentCryptographyDataOperationByName[action]; !known {
		return map[string]any{}
	}

	keyARN := s.resolveKeyARN(payload, pathParams)

	switch action {
	case "DecryptData":
		return map[string]any{
			"KeyArn":        keyARN,
			"KeyCheckValue": "000000",
			"PlainText":     "c3RhY2t5YXJk",
		}
	case "EncryptData", "ReEncryptData":
		return map[string]any{
			"KeyArn":        keyARN,
			"KeyCheckValue": "000000",
			"CipherText":    "U1RBQ0tZQVJE",
		}
	case "GenerateAs2805KekValidation":
		return map[string]any{
			"KekValidationResponse": map[string]any{
				"KeyCheckValue": "000000",
				"Nonce":         "ABCDEF0123456789",
			},
		}
	case "GenerateCardValidationData":
		return map[string]any{
			"KeyArn":               keyARN,
			"KeyCheckValue":        "000000",
			"ValidationData":       "123",
			"CardValidationData":   "123",
			"DynamicData":          "123",
			"GeneratedDataVersion": "v1",
		}
	case "GenerateMac", "GenerateMacEmvPinChange":
		out := map[string]any{
			"KeyArn":        keyARN,
			"KeyCheckValue": "000000",
			"Mac":           "1122334455667788",
		}
		if action == "GenerateMacEmvPinChange" {
			out["PinBlock"] = "1122334455667788"
		}
		return out
	case "GeneratePinData":
		return map[string]any{
			"KeyArn":        keyARN,
			"KeyCheckValue": "000000",
			"PinData": map[string]any{
				"PinBlock":          "1122334455667788",
				"PinBlockFormat":    "ISO_FORMAT_0",
				"VerificationValue": "1234",
			},
		}
	case "TranslateKeyMaterial":
		return map[string]any{
			"KeyArn":             keyARN,
			"KeyCheckValue":      "000000",
			"TranslatedMaterial": "A1B2C3D4E5F60708",
		}
	case "TranslatePinData":
		return map[string]any{
			"KeyArn":        keyARN,
			"KeyCheckValue": "000000",
			"PinBlock":      "1122334455667788",
		}
	case "VerifyAuthRequestCryptogram":
		return map[string]any{
			"KeyArn":            keyARN,
			"KeyCheckValue":     "000000",
			"AuthResponseValue": "3030",
		}
	case "VerifyCardValidationData":
		return map[string]any{
			"KeyArn":                keyARN,
			"KeyCheckValue":         "000000",
			"ValidationDataMatched": true,
		}
	case "VerifyMac":
		return map[string]any{
			"KeyArn":        keyARN,
			"KeyCheckValue": "000000",
			"MacValid":      true,
		}
	case "VerifyPinData":
		return map[string]any{
			"KeyArn":        keyARN,
			"KeyCheckValue": "000000",
			"PinData": map[string]any{
				"VerificationValue": "1234",
			},
			"VerificationResult": "PIN_VALID",
		}
	default:
		return map[string]any{}
	}
}

func (s *paymentCryptographyDataStore) resolveKeyARN(payload, pathParams map[string]any) string {
	for _, key := range []string{
		"KeyIdentifier",
		"IncomingKeyIdentifier",
		"OutgoingKeyIdentifier",
	} {
		if v := paymentCryptographyDataMapString(pathParams, key); v != "" {
			return paymentCryptographyDataNormalizeKeyIdentifier(v, s.defaultKeyARN)
		}
		if v := paymentCryptographyDataMapString(payload, key); v != "" {
			return paymentCryptographyDataNormalizeKeyIdentifier(v, s.defaultKeyARN)
		}
	}
	return s.defaultKeyARN
}

func paymentCryptographyDataNormalizeKeyIdentifier(keyIdentifier, fallback string) string {
	keyIdentifier = strings.TrimSpace(keyIdentifier)
	if keyIdentifier == "" {
		return fallback
	}
	if strings.HasPrefix(keyIdentifier, "arn:aws:payment-cryptography:") {
		return keyIdentifier
	}
	if strings.HasPrefix(keyIdentifier, "alias/") {
		return "arn:aws:payment-cryptography:us-east-1:123456789012:" + keyIdentifier
	}
	return fallback
}

func paymentCryptographyDataMapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}
