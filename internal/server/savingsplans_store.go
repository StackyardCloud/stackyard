package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	savingsPlansDefaultRegion    = "us-east-1"
	savingsPlansDefaultAccountID = "123456789012"
)

type savingsPlansStore struct {
	mu sync.Mutex

	nextPlanID int64

	savingsPlans       map[string]map[string]any
	savingsPlanARNToID map[string]string
	clientTokenToID    map[string]string
	offerings          map[string]map[string]any
	tags               map[string]map[string]string
}

func newSavingsPlansStore() *savingsPlansStore {
	s := &savingsPlansStore{
		nextPlanID:         2,
		savingsPlans:       map[string]map[string]any{},
		savingsPlanARNToID: map[string]string{},
		clientTokenToID:    map[string]string{},
		offerings:          map[string]map[string]any{},
		tags:               map[string]map[string]string{},
	}
	s.seedLocked(time.Now().UTC())
	return s
}

func (s *savingsPlansStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.seedLocked(now)
	ctx := savingsPlansMergeMaps(payload, pathParams, query)

	savingsPlanID := savingsPlansString(ctx, []string{"savingsPlanId", "savingsPlanID"}, "sp-000001")
	savingsPlanArn := savingsPlansString(ctx, []string{"savingsPlanArn", "resourceArn", "resourceARN"}, savingsPlansPlanARN(savingsPlanID))
	if resolved := s.savingsPlanARNToID[savingsPlanArn]; strings.TrimSpace(resolved) != "" {
		savingsPlanID = resolved
	}
	offeringID := savingsPlansString(ctx, []string{"savingsPlanOfferingId"}, "offering-000001")
	clientToken := savingsPlansString(ctx, []string{"clientToken"}, "")

	switch action {
	case "CreateSavingsPlan":
		if clientToken == "" {
			clientToken = fmt.Sprintf("stackyard-token-%06d", s.nextPlanID)
		}
		if existingID := strings.TrimSpace(s.clientTokenToID[clientToken]); existingID != "" {
			plan := s.ensureSavingsPlanLocked(existingID, now)
			return map[string]any{
				"savingsPlanId":  existingID,
				"savingsPlanArn": savingsPlansString(plan, []string{"savingsPlanArn"}, savingsPlansPlanARN(existingID)),
			}
		}
		newID := fmt.Sprintf("sp-%06d", s.nextPlanID)
		s.nextPlanID++
		plan := s.ensureSavingsPlanLocked(newID, now)
		plan["state"] = "payment-pending"
		plan["clientToken"] = clientToken
		plan["savingsPlanOfferingId"] = offeringID
		plan["upfrontPaymentAmount"] = savingsPlansString(ctx, []string{"upfrontPaymentAmount"}, "0.0")
		plan["commitment"] = savingsPlansString(ctx, []string{"commitment"}, "0.001")
		plan["purchaseTime"] = now.Format(time.RFC3339)
		planARN := savingsPlansString(plan, []string{"savingsPlanArn"}, savingsPlansPlanARN(newID))
		s.savingsPlanARNToID[planARN] = newID
		s.clientTokenToID[clientToken] = newID
		s.ensureTagsLocked(planARN)
		return map[string]any{
			"savingsPlanId":  newID,
			"savingsPlanArn": planARN,
		}

	case "DeleteQueuedSavingsPlan":
		plan := s.ensureSavingsPlanLocked(savingsPlanID, now)
		plan["state"] = "payment-pending-deleted"
		return map[string]any{}

	case "DescribeSavingsPlanRates":
		plan := s.ensureSavingsPlanLocked(savingsPlanID, now)
		rate := map[string]any{
			"rate":                       "0.010000",
			"currency":                   "USD",
			"unit":                       "Hrs",
			"serviceCode":                "AmazonEC2",
			"usageType":                  "BoxUsage:m5.large",
			"operation":                  "RunInstances",
			"properties":                 []any{map[string]any{"name": "region", "value": savingsPlansDefaultRegion}},
			"savingsPlanRateCardEntryId": "rate-card-000001",
		}
		return map[string]any{
			"searchResults": []any{
				map[string]any{
					"savingsPlanArn":  savingsPlansString(plan, []string{"savingsPlanArn"}, savingsPlansPlanARN(savingsPlanID)),
					"savingsPlanRate": rate,
				},
			},
			"nextToken": "",
		}

	case "DescribeSavingsPlans":
		plans := s.listSavingsPlansLocked()
		filteredIDs := savingsPlansStringSlice(ctx["savingsPlanIds"])
		filteredARNs := savingsPlansStringSlice(ctx["savingsPlanArns"])
		filtered := make([]any, 0, len(plans))
		for _, planAny := range plans {
			plan, _ := planAny.(map[string]any)
			id := savingsPlansString(plan, []string{"savingsPlanId"}, "")
			arn := savingsPlansString(plan, []string{"savingsPlanArn"}, "")
			if len(filteredIDs) > 0 && !savingsPlansContains(filteredIDs, id) {
				continue
			}
			if len(filteredARNs) > 0 && !savingsPlansContains(filteredARNs, arn) {
				continue
			}
			filtered = append(filtered, planAny)
		}
		return map[string]any{
			"savingsPlans": filtered,
			"nextToken":    "",
		}

	case "DescribeSavingsPlansOfferingRates":
		offering := s.ensureOfferingLocked(offeringID, now)
		return map[string]any{
			"searchResults": []any{
				map[string]any{
					"savingsPlanOffering": map[string]any{
						"savingsPlanOfferingId": savingsPlansString(offering, []string{"savingsPlanOfferingId"}, offeringID),
						"durationSeconds":       offering["durationSeconds"],
						"planType":              offering["planType"],
						"paymentOption":         offering["paymentOption"],
						"currency":              offering["currency"],
					},
					"rate":                       "0.010000",
					"currency":                   "USD",
					"unit":                       "Hrs",
					"serviceCode":                "AmazonEC2",
					"usageType":                  "BoxUsage:m5.large",
					"operation":                  "RunInstances",
					"properties":                 []any{map[string]any{"name": "region", "value": savingsPlansDefaultRegion}},
					"savingsPlanRateCardEntryId": "offering-rate-card-000001",
				},
			},
			"nextToken": "",
		}

	case "DescribeSavingsPlansOfferings":
		offerings := make([]any, 0, len(s.offerings))
		keys := make([]string, 0, len(s.offerings))
		for key := range s.offerings {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			offerings = append(offerings, savingsPlansCloneMap(s.offerings[key]))
		}
		return map[string]any{
			"searchResults": offerings,
			"nextToken":     "",
		}

	case "ListTagsForResource":
		if strings.TrimSpace(savingsPlanArn) == "" {
			savingsPlanArn = savingsPlansPlanARN(savingsPlanID)
		}
		return map[string]any{
			"tags": savingsPlansCloneMapString(s.ensureTagsLocked(savingsPlanArn)),
		}

	case "ReturnSavingsPlan":
		plan := s.ensureSavingsPlanLocked(savingsPlanID, now)
		plan["state"] = "returned"
		plan["returnTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "TagResource":
		if strings.TrimSpace(savingsPlanArn) == "" {
			savingsPlanArn = savingsPlansPlanARN(savingsPlanID)
		}
		existing := s.ensureTagsLocked(savingsPlanArn)
		for key, value := range savingsPlansMapString(ctx["tags"]) {
			existing[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		if strings.TrimSpace(savingsPlanArn) == "" {
			savingsPlanArn = savingsPlansPlanARN(savingsPlanID)
		}
		existing := s.ensureTagsLocked(savingsPlanArn)
		for _, key := range savingsPlansStringSlice(ctx["tagKeys"]) {
			delete(existing, key)
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *savingsPlansStore) seedLocked(now time.Time) {
	plan := s.ensureSavingsPlanLocked("sp-000001", now)
	arn := savingsPlansString(plan, []string{"savingsPlanArn"}, savingsPlansPlanARN("sp-000001"))
	s.savingsPlanARNToID[arn] = "sp-000001"
	s.ensureOfferingLocked("offering-000001", now)
	tags := s.ensureTagsLocked(arn)
	if len(tags) == 0 {
		tags["env"] = "local"
		tags["service"] = "savingsplans"
	}
}

func (s *savingsPlansStore) ensureSavingsPlanLocked(id string, now time.Time) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "sp-000001"
	}
	if plan := s.savingsPlans[id]; plan != nil {
		return plan
	}
	plan := map[string]any{
		"savingsPlanId":         id,
		"savingsPlanArn":        savingsPlansPlanARN(id),
		"state":                 "active",
		"region":                savingsPlansDefaultRegion,
		"ec2InstanceFamily":     "m5",
		"commitment":            "0.001",
		"upfrontPaymentAmount":  "0.0",
		"termDurationInSeconds": 31536000,
		"paymentOption":         "No Upfront",
		"savingsPlanType":       "Compute",
		"currency":              "USD",
		"start":                 now.Add(-24 * time.Hour).Format(time.RFC3339),
		"end":                   now.Add(365 * 24 * time.Hour).Format(time.RFC3339),
		"description":           "stackyard savings plan",
	}
	s.savingsPlans[id] = plan
	s.savingsPlanARNToID[savingsPlansString(plan, []string{"savingsPlanArn"}, savingsPlansPlanARN(id))] = id
	return plan
}

func (s *savingsPlansStore) ensureOfferingLocked(id string, now time.Time) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "offering-000001"
	}
	if offering := s.offerings[id]; offering != nil {
		return offering
	}
	offering := map[string]any{
		"savingsPlanOfferingId": id,
		"description":           "stackyard compute savings plan offering",
		"currency":              "USD",
		"planType":              "Compute",
		"paymentOption":         "No Upfront",
		"durationSeconds":       31536000,
		"serviceCode":           "AmazonEC2",
		"usageType":             "BoxUsage:m5.large",
		"operation":             "RunInstances",
		"properties":            []any{map[string]any{"name": "region", "value": savingsPlansDefaultRegion}},
		"offeringId":            id,
		"creationTime":          now.Format(time.RFC3339),
	}
	s.offerings[id] = offering
	return offering
}

func (s *savingsPlansStore) ensureTagsLocked(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = savingsPlansPlanARN("sp-000001")
	}
	if tags, ok := s.tags[resourceArn]; ok {
		return tags
	}
	tags := map[string]string{}
	s.tags[resourceArn] = tags
	return tags
}

func (s *savingsPlansStore) listSavingsPlansLocked() []any {
	keys := make([]string, 0, len(s.savingsPlans))
	for key := range s.savingsPlans {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, savingsPlansCloneMap(s.savingsPlans[key]))
	}
	return out
}

func savingsPlansPlanARN(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "sp-000001"
	}
	return fmt.Sprintf("arn:aws:savingsplans:%s:%s:savingsplan/%s", savingsPlansDefaultRegion, savingsPlansDefaultAccountID, id)
}

func savingsPlansMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for k, v := range payload {
		out[k] = v
	}
	for k, v := range pathParams {
		out[k] = v
	}
	for k, values := range query {
		if len(values) > 0 {
			out[k] = values[len(values)-1]
		}
	}
	return out
}

func savingsPlansString(payload map[string]any, keys []string, def string) string {
	if payload == nil {
		return def
	}
	for _, key := range keys {
		for actual, raw := range payload {
			if !strings.EqualFold(strings.TrimSpace(actual), strings.TrimSpace(key)) {
				continue
			}
			value := strings.TrimSpace(fmt.Sprint(raw))
			if value != "" && value != "<nil>" {
				return value
			}
		}
	}
	return def
}

func savingsPlansStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, raw := range typed {
			item := strings.TrimSpace(fmt.Sprint(raw))
			if item != "" && item != "<nil>" {
				out = append(out, item)
			}
		}
		return out
	case string:
		parts := strings.Split(typed, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
		return out
	default:
		return nil
	}
}

func savingsPlansMapString(value any) map[string]string {
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]string:
		for key, raw := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(raw)
		}
	case map[string]any:
		for key, raw := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprint(raw))
		}
	}
	return out
}

func savingsPlansCloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = savingsPlansCloneMap(typed)
		case map[string]string:
			out[key] = savingsPlansCloneMapString(typed)
		case []any:
			clone := make([]any, 0, len(typed))
			for _, item := range typed {
				if itemMap, ok := item.(map[string]any); ok {
					clone = append(clone, savingsPlansCloneMap(itemMap))
					continue
				}
				clone = append(clone, item)
			}
			out[key] = clone
		default:
			out[key] = value
		}
	}
	return out
}

func savingsPlansCloneMapString(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func savingsPlansContains(items []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), target) {
			return true
		}
	}
	return false
}
