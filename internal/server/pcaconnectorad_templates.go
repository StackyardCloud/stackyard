package server

import "strings"

// padTemplateVersions maps a TemplateDefinition member to its PolicySchema.
var padTemplateVersions = map[string]int{"TemplateV2": 2, "TemplateV3": 3, "TemplateV4": 4}

// padValidityHours converts a ValidityPeriod unit to hours (a month is 730
// hours and a year 8,766, the averages Windows uses).
var padValidityHours = map[string]float64{"HOURS": 1, "DAYS": 24, "WEEKS": 168, "MONTHS": 730, "YEARS": 8766}

// padCheckDefinition checks what the model validator can't: the
// TemplateDefinition, ApplicationPolicy and KeyUsageProperty unions (exactly
// one member), template names in SupersededTemplates, and a renewal period
// shorter than the validity period. It returns the PolicySchema (2, 3 or 4).
func padCheckDefinition(def map[string]any) (int, *padError) {
	if err := padOneOf("Definition", def, "TemplateV2", "TemplateV3", "TemplateV4"); err != nil {
		return 0, err
	}
	var key string
	for k := range def {
		key = k
	}
	tv, _ := def[key].(map[string]any)

	cv, _ := tv["CertificateValidity"].(map[string]any)
	validity, renewal := padHours(cv["ValidityPeriod"]), padHours(cv["RenewalPeriod"])
	if renewal >= validity {
		return 0, padValidation("FIELD_VALIDATION_FAILED", "%s.CertificateValidity.RenewalPeriod must be shorter than ValidityPeriod.", key)
	}

	ext, _ := tv["Extensions"].(map[string]any)
	if ap, ok := ext["ApplicationPolicies"].(map[string]any); ok {
		policies, _ := ap["Policies"].([]any)
		for _, p := range policies {
			pm, _ := p.(map[string]any)
			if err := padOneOf(key+".Extensions.ApplicationPolicies.Policies[]", pm, "PolicyObjectIdentifier", "PolicyType"); err != nil {
				return 0, err
			}
		}
	}
	pka, _ := tv["PrivateKeyAttributes"].(map[string]any)
	if kup, ok := pka["KeyUsageProperty"].(map[string]any); ok {
		if err := padOneOf(key+".PrivateKeyAttributes.KeyUsageProperty", kup, "PropertyFlags", "PropertyType"); err != nil {
			return 0, err
		}
	}
	if names, ok := tv["SupersededTemplates"].([]any); ok {
		for _, n := range names {
			s, _ := n.(string)
			if err := padCheckTemplateName(key+".SupersededTemplates[]", s); err != nil {
				return 0, err
			}
		}
	}
	return padTemplateVersions[key], nil
}

// padOneOf enforces a union: exactly one of members is set.
func padOneOf(path string, m map[string]any, members ...string) *padError {
	set := 0
	for _, k := range members {
		if v, ok := m[k]; ok && v != nil {
			set++
		}
	}
	if set != 1 {
		return padValidation("FIELD_VALIDATION_FAILED", "%s must set exactly one of %s.", path, strings.Join(members, ", "))
	}
	return nil
}

func padHours(v any) float64 {
	p, _ := v.(map[string]any)
	n, _ := p["Period"].(float64)
	unit, _ := p["PeriodType"].(string)
	return n * padValidityHours[unit]
}

// padCheckTemplateName applies TemplateName's pattern, whose lookaheads RE2
// can't compile: printable ASCII, not only whitespace, and none of \ ' + , ; < = > # ".
func padCheckTemplateName(field, name string) *padError {
	if name == "" || len(name) > 64 || strings.TrimSpace(name) == "" {
		return padValidation("FIELD_VALIDATION_FAILED", "%s must be 1-64 characters and not only whitespace.", field)
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c < 0x20 || c > 0x7e || strings.IndexByte(`\'+,;<=>#"`, c) >= 0 {
			return padValidation("FIELD_VALIDATION_FAILED", "%s contains a character that isn't allowed.", field)
		}
	}
	return nil
}
