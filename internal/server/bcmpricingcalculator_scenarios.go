package server

import (
	"time"
)

// Bill scenarios and their usage and commitment modifications. Records are
// replaced (copy on write), never changed in place; size tracks the bytes
// each scenario holds in the store budget.

type bcmpcScenario struct {
	id, name, sharing, costCategoryArn string
	created, expires, start, end       time.Time
	usage, commits                     []map[string]any // stored items, oldest first
	tags                               map[string]string
	token, tokenSum                    string
	size                               int
}

const bcmpcScenarioKind = "BillScenario"

// status derives the scenario status now. Caller holds st.mu.
func (st *bcmpcStore) scenarioStatus(sc *bcmpcScenario, now time.Time) string {
	for _, e := range st.estimates {
		if e.scenarioID == sc.id && e.status == "IN_PROGRESS" {
			return "LOCKED"
		}
	}
	if now.After(sc.expires) {
		return "STALE"
	}
	return "READY"
}

func (st *bcmpcStore) scenarioOut(sc *bcmpcScenario, now time.Time) map[string]any {
	out := map[string]any{
		"id": sc.id, "name": sc.name, "billInterval": bcmpcIntervalOut(sc.start, sc.end), "status": st.scenarioStatus(sc, now),
		"createdAt": budEpoch(sc.created), "expiresAt": budEpoch(sc.expires), "groupSharingPreference": sc.sharing,
	}
	if sc.costCategoryArn != "" {
		out["costCategoryGroupSharingPreferenceArn"] = sc.costCategoryArn
	}
	return out
}

// bcmpcSharing checks that PRIORITIZED and RESTRICTED sharing name a cost
// category and OPEN does not.
func bcmpcSharing(pref, arn string) *budErr {
	if (pref == "OPEN") == (arn != "") {
		return bcmpcInvalid("costCategoryGroupSharingPreferenceArn is required for PRIORITIZED and RESTRICTED group sharing, and not allowed for OPEN")
	}
	return nil
}

// scenario finds a scenario; editable also refuses a LOCKED one. Caller
// holds st.mu.
func (st *bcmpcStore) scenario(id string, editable bool, now time.Time) (*bcmpcScenario, *budErr) {
	sc := st.scenarios[id]
	if sc == nil {
		return nil, bcmpcNotFound(id, bcmpcScenarioKind)
	}
	if editable && st.scenarioStatus(sc, now) == "LOCKED" {
		return nil, bcmpcResourceErr("ConflictException", id, bcmpcScenarioKind, "bill scenario %s is locked while a bill estimate is in progress", id)
	}
	return sc, nil
}

func bcmpcCheckTags(tags map[string]string) *budErr {
	for k := range tags {
		if len(k) >= 4 && (k[:4] == "aws:" || k[:4] == "AWS:") {
			return bcmpcInvalid("tag keys starting with aws: are reserved")
		}
	}
	return nil
}

func (s *Server) bcmpcCreateScenario(in map[string]any) (map[string]any, *budErr) {
	sharing := budStr(in, "groupSharingPreference")
	if sharing == "" {
		sharing = "OPEN"
	}
	ccArn := budStr(in, "costCategoryGroupSharingPreferenceArn")
	if err := bcmpcSharing(sharing, ccArn); err != nil {
		return nil, err
	}
	tags := bcmpcTagsIn(in)
	if err := bcmpcCheckTags(tags); err != nil {
		return nil, err
	}
	token, sum := budStr(in, "clientToken"), bcmdSum(in)
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	if token != "" {
		for _, sc := range st.scenarios {
			if sc.token == token {
				if sc.tokenSum != sum {
					return nil, bcmpcResourceErr("ConflictException", sc.id, bcmpcScenarioKind, "clientToken %s was used with a different request", token)
				}
				return st.scenarioOut(sc, now), nil
			}
		}
	}
	if len(st.scenarios) >= bcmpcMaxScenarios {
		return nil, bcmpcResourceErr("ServiceQuotaExceededException", "", bcmpcScenarioKind, "an account can have at most %d bill scenarios", bcmpcMaxScenarios)
	}
	start, end := bcmpcInterval(now)
	sc := &bcmpcScenario{
		id: budNewID(), name: budStr(in, "name"), sharing: sharing, costCategoryArn: ccArn,
		created: now, expires: now.Add(bcmpcLifetime), start: start, end: end, tags: tags, token: token, tokenSum: sum,
	}
	sc.size = bcmpcSize(sc.name, sc.costCategoryArn, sc.tags, sc.token)
	if err := st.charge(0, sc.size, sc.id, bcmpcScenarioKind); err != nil {
		return nil, err
	}
	st.scenarios[sc.id] = sc
	return st.scenarioOut(sc, now), nil
}

func (s *Server) bcmpcGetScenario(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	sc, err := st.scenario(budStr(in, "identifier"), false, now)
	if err != nil {
		return nil, err
	}
	return st.scenarioOut(sc, now), nil
}

func (s *Server) bcmpcUpdateScenario(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	sc, err := st.scenario(budStr(in, "identifier"), true, now)
	if err != nil {
		return nil, err
	}
	next := *sc
	if v, ok := in["name"]; ok {
		next.name, _ = v.(string)
	}
	if t, ok, err := bcmpcExpiry(in, now); err != nil {
		return nil, err
	} else if ok {
		next.expires = t
	}
	if v, ok := in["groupSharingPreference"]; ok {
		next.sharing, _ = v.(string)
		next.costCategoryArn = budStr(in, "costCategoryGroupSharingPreferenceArn")
	} else if v, ok := in["costCategoryGroupSharingPreferenceArn"]; ok {
		next.costCategoryArn, _ = v.(string)
	}
	if err := bcmpcSharing(next.sharing, next.costCategoryArn); err != nil {
		return nil, err
	}
	next.size = sc.size - bcmpcSize(sc.name, sc.costCategoryArn) + bcmpcSize(next.name, next.costCategoryArn)
	if err := st.charge(sc.size, next.size, sc.id, bcmpcScenarioKind); err != nil {
		return nil, err
	}
	st.scenarios[sc.id] = &next
	return st.scenarioOut(&next, now), nil
}

func (s *Server) bcmpcDeleteScenario(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	sc, err := st.scenario(budStr(in, "identifier"), true, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	st.bytes -= sc.size
	delete(st.scenarios, sc.id)
	return map[string]any{}, nil
}

func (s *Server) bcmpcListScenarios(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	created, expires := budMap(in, "createdAtFilter"), budMap(in, "expiresAtFilter")
	items := make([]map[string]any, 0, len(st.scenarios))
	for _, sc := range st.scenarios {
		if bcmpcInWindow(sc.created, created) && bcmpcInWindow(sc.expires, expires) {
			items = append(items, st.scenarioOut(sc, now))
		}
	}
	items = bcmpcFiltered(items, budList(in, "filters"), func(item map[string]any, name string) string {
		return budStr(item, map[string]string{
			"STATUS": "status", "NAME": "name", "GROUP_SHARING_PREFERENCE": "groupSharingPreference",
			"COST_CATEGORY_ARN": "costCategoryGroupSharingPreferenceArn",
		}[name])
	})
	bcmpcSortByCreated(items)
	return bcmpcPage(in, items, 25)
}

// ---- usage modifications ----

// bcmpcBatchErr is one rejected batch entry.
func bcmpcBatchErr(idKey, id, code, msg string) map[string]any {
	return map[string]any{idKey: id, "errorCode": code, "errorMessage": msg}
}

// bcmpcCheckAmounts checks that amounts are non-negative and start inside
// [start, end); it returns them as quantities in unit.
func bcmpcCheckAmounts(amounts []any, unit string, start, end time.Time) ([]any, string) {
	out := make([]any, 0, len(amounts))
	for _, a := range amounts {
		m, _ := a.(map[string]any)
		at, ok := budTime(m["startHour"])
		amount, _ := m["amount"].(float64)
		if !ok || at.Before(start) || !at.Before(end) {
			return nil, "startHour must fall inside the bill interval"
		}
		if amount < 0 {
			return nil, "amount must not be negative"
		}
		out = append(out, map[string]any{"startHour": budEpoch(at), "unit": unit, "amount": amount})
	}
	return out, ""
}

func bcmpcCheckHistorical(h map[string]any) string {
	if h == nil {
		return ""
	}
	bi := budMap(h, "billInterval")
	start, ok1 := budTime(bi["start"])
	end, ok2 := budTime(bi["end"])
	if !ok1 || !ok2 || !end.After(start) {
		return "historicalUsage.billInterval must have a start before its end"
	}
	return ""
}

func (s *Server) bcmpcCreateUsageMods(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	sc, err := st.scenario(budStr(in, "billScenarioId"), true, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	entries := budList(in, "usageModifications")
	if len(sc.usage)+len(entries) > bcmpcMaxUsage {
		return nil, bcmpcResourceErr("ServiceQuotaExceededException", sc.id, bcmpcScenarioKind, "a bill scenario can have at most %d usage modifications", bcmpcMaxUsage)
	}
	items, errs, added := []any{}, []any{}, []map[string]any{}
	keys, grow := map[string]bool{}, 0
	for _, e := range entries {
		m, _ := e.(map[string]any)
		key := budStr(m, "key")
		p, found := bcmpcLookup(budStr(m, "serviceCode"), budStr(m, "usageType"), budStr(m, "operation"))
		quantities, problem := bcmpcCheckAmounts(budList(m, "amounts"), p.unit, sc.start, sc.end)
		if problem == "" {
			problem = bcmpcCheckHistorical(budMap(m, "historicalUsage"))
		}
		switch {
		case keys[key]:
			errs = append(errs, bcmpcBatchErr("key", key, "CONFLICT", "key is used twice in this request"))
			continue
		case !found:
			errs = append(errs, bcmpcBatchErr("key", key, "NOT_FOUND", "no price for this service code, usage type and operation"))
			continue
		case budStr(m, "usageAccountId") != bcmpcAccountID:
			errs = append(errs, bcmpcBatchErr("key", key, "BAD_REQUEST", "usageAccountId is not an account in this organization"))
			continue
		case problem != "":
			errs = append(errs, bcmpcBatchErr("key", key, "BAD_REQUEST", problem))
			continue
		}
		keys[key] = true
		item := map[string]any{
			"id": budNewID(), "serviceCode": m["serviceCode"], "usageType": m["usageType"], "operation": m["operation"],
			"location": p.location, "usageAccountId": m["usageAccountId"], "quantities": quantities,
		}
		for _, k := range []string{"availabilityZone", "group", "historicalUsage"} {
			if v, ok := m[k]; ok {
				item[k] = budClone(v)
			}
		}
		grow += bcmpcSize(item)
		added = append(added, item)
		out := budCloneMap(item)
		out["key"] = key
		items = append(items, out)
	}
	if err := st.charge(sc.size, sc.size+grow, sc.id, bcmpcScenarioKind); err != nil {
		return nil, err
	}
	next := *sc
	next.usage = append(append([]map[string]any{}, sc.usage...), added...)
	next.size = sc.size + grow
	st.scenarios[sc.id] = &next
	return map[string]any{"items": items, "errors": errs}, nil
}

func bcmpcIndex(list []map[string]any, id string) int {
	for i, item := range list {
		if budStr(item, "id") == id {
			return i
		}
	}
	return -1
}

func (s *Server) bcmpcUpdateUsageMods(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	sc, err := st.scenario(budStr(in, "billScenarioId"), true, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	usage := append([]map[string]any{}, sc.usage...)
	items, errs, size := []any{}, []any{}, sc.size
	for _, e := range budList(in, "usageModifications") {
		m, _ := e.(map[string]any)
		id := budStr(m, "id")
		i := bcmpcIndex(usage, id)
		if i < 0 {
			errs = append(errs, bcmpcBatchErr("id", id, "NOT_FOUND", "usage modification does not exist"))
			continue
		}
		item := budCloneMap(usage[i])
		if v, ok := m["group"]; ok {
			item["group"] = v
		}
		if amounts, ok := m["amounts"].([]any); ok {
			p, _ := bcmpcLookup(budStr(item, "serviceCode"), budStr(item, "usageType"), budStr(item, "operation"))
			quantities, problem := bcmpcCheckAmounts(amounts, p.unit, sc.start, sc.end)
			if problem != "" {
				errs = append(errs, bcmpcBatchErr("id", id, "BAD_REQUEST", problem))
				continue
			}
			item["quantities"] = quantities
		}
		size += bcmpcSize(item) - bcmpcSize(usage[i])
		usage[i] = item
		items = append(items, budCloneMap(item))
	}
	if err := st.charge(sc.size, size, sc.id, bcmpcScenarioKind); err != nil {
		return nil, err
	}
	next := *sc
	next.usage, next.size = usage, size
	st.scenarios[sc.id] = &next
	return map[string]any{"items": items, "errors": errs}, nil
}

// bcmpcDeleteItems removes the listed ids; unknown ids are BAD_REQUEST.
func bcmpcDeleteItems(list []map[string]any, ids []string) ([]map[string]any, []any, int) {
	kept := append([]map[string]any{}, list...)
	errs, freed := []any{}, 0
	for _, id := range ids {
		i := bcmpcIndex(kept, id)
		if i < 0 {
			errs = append(errs, bcmpcBatchErr("id", id, "BAD_REQUEST", "modification does not exist"))
			continue
		}
		freed += bcmpcSize(kept[i])
		kept = append(kept[:i:i], kept[i+1:]...)
	}
	return kept, errs, freed
}

func (s *Server) bcmpcDeleteUsageMods(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	sc, err := st.scenario(budStr(in, "billScenarioId"), true, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	kept, errs, freed := bcmpcDeleteItems(sc.usage, budStrings(in, "ids"))
	next := *sc
	next.usage, next.size = kept, sc.size-freed
	st.bytes -= freed
	st.scenarios[sc.id] = &next
	return map[string]any{"errors": errs}, nil
}

func (s *Server) bcmpcListUsageMods(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	sc, err := st.scenario(budStr(in, "billScenarioId"), false, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return bcmpcPage(in, bcmpcFiltered(sc.usage, budList(in, "filters"), bcmpcUsageField), 25)
}

// ---- commitment modifications ----

// bcmpcCheckAction checks a commitment action union: exactly one action,
// with the members it needs, and RI offerings from the catalog.
func bcmpcCheckAction(a map[string]any) *budErr {
	if len(a) != 1 {
		return bcmpcInvalid("commitmentAction must set exactly one action")
	}
	switch {
	case a["addReservedInstanceAction"] != nil:
		ri := budMap(a, "addReservedInstanceAction")
		if _, ok := bcmpcRIOfferings[budStr(ri, "reservedInstancesOfferingId")]; !ok {
			return bcmpcInvalid("reserved instances offering %q is not in the price list", budStr(ri, "reservedInstancesOfferingId"))
		}
		if budInt(ri, "instanceCount", 0) < 1 {
			return bcmpcInvalid("addReservedInstanceAction needs an instanceCount")
		}
	case a["addSavingsPlanAction"] != nil:
		sp := budMap(a, "addSavingsPlanAction")
		if budStr(sp, "savingsPlanOfferingId") == "" || sp["commitment"] == nil {
			return bcmpcInvalid("addSavingsPlanAction needs a savingsPlanOfferingId and a commitment")
		}
	case a["negateReservedInstanceAction"] != nil:
		if budStr(budMap(a, "negateReservedInstanceAction"), "reservedInstancesId") == "" {
			return bcmpcInvalid("negateReservedInstanceAction needs a reservedInstancesId")
		}
	default:
		if budStr(budMap(a, "negateSavingsPlanAction"), "savingsPlanId") == "" {
			return bcmpcInvalid("negateSavingsPlanAction needs a savingsPlanId")
		}
	}
	return nil
}

func (s *Server) bcmpcCreateCommitMods(in map[string]any) (map[string]any, *budErr) {
	entries := budList(in, "commitmentModifications")
	for _, e := range entries {
		if err := bcmpcCheckAction(budMap(bcmAsMap(e), "commitmentAction")); err != nil {
			return nil, err
		}
	}
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	sc, err := st.scenario(budStr(in, "billScenarioId"), true, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if len(sc.commits)+len(entries) > bcmpcMaxCommits {
		return nil, bcmpcResourceErr("ServiceQuotaExceededException", sc.id, bcmpcScenarioKind, "a bill scenario can have at most %d commitment modifications", bcmpcMaxCommits)
	}
	items, errs, added := []any{}, []any{}, []map[string]any{}
	keys, grow := map[string]bool{}, 0
	for _, e := range entries {
		m, _ := e.(map[string]any)
		key := budStr(m, "key")
		switch {
		case keys[key]:
			errs = append(errs, bcmpcBatchErr("key", key, "CONFLICT", "key is used twice in this request"))
			continue
		case budStr(m, "usageAccountId") != bcmpcAccountID:
			errs = append(errs, bcmpcBatchErr("key", key, "INVALID_ACCOUNT", "usageAccountId is not an account in this organization"))
			continue
		}
		keys[key] = true
		item := map[string]any{"id": budNewID(), "usageAccountId": m["usageAccountId"], "commitmentAction": budClone(m["commitmentAction"])}
		if v, ok := m["group"]; ok {
			item["group"] = v
		}
		grow += bcmpcSize(item)
		added = append(added, item)
		out := budCloneMap(item)
		out["key"] = key
		items = append(items, out)
	}
	if err := st.charge(sc.size, sc.size+grow, sc.id, bcmpcScenarioKind); err != nil {
		return nil, err
	}
	next := *sc
	next.commits = append(append([]map[string]any{}, sc.commits...), added...)
	next.size = sc.size + grow
	st.scenarios[sc.id] = &next
	return map[string]any{"items": items, "errors": errs}, nil
}

func (s *Server) bcmpcUpdateCommitMods(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	sc, err := st.scenario(budStr(in, "billScenarioId"), true, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	commits := append([]map[string]any{}, sc.commits...)
	items, errs, size := []any{}, []any{}, sc.size
	for _, e := range budList(in, "commitmentModifications") {
		m, _ := e.(map[string]any)
		id := budStr(m, "id")
		i := bcmpcIndex(commits, id)
		if i < 0 {
			errs = append(errs, bcmpcBatchErr("id", id, "NOT_FOUND", "commitment modification does not exist"))
			continue
		}
		item := budCloneMap(commits[i])
		if v, ok := m["group"]; ok {
			item["group"] = v
		}
		size += bcmpcSize(item) - bcmpcSize(commits[i])
		commits[i] = item
		items = append(items, budCloneMap(item))
	}
	if err := st.charge(sc.size, size, sc.id, bcmpcScenarioKind); err != nil {
		return nil, err
	}
	next := *sc
	next.commits, next.size = commits, size
	st.scenarios[sc.id] = &next
	return map[string]any{"items": items, "errors": errs}, nil
}

func (s *Server) bcmpcDeleteCommitMods(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	sc, err := st.scenario(budStr(in, "billScenarioId"), true, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	kept, errs, freed := bcmpcDeleteItems(sc.commits, budStrings(in, "ids"))
	next := *sc
	next.commits, next.size = kept, sc.size-freed
	st.bytes -= freed
	st.scenarios[sc.id] = &next
	return map[string]any{"errors": errs}, nil
}

func (s *Server) bcmpcListCommitMods(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	sc, err := st.scenario(budStr(in, "billScenarioId"), false, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return bcmpcPage(in, sc.commits, 25)
}
