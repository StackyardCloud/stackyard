package server

import (
	"time"
)

// Workload estimates price usage lines directly (amount x catalog rate).
// Any usage change leaves the estimate UPDATING until the next request.

type bcmpcWorkload struct {
	id, name, rateType, status string
	created, expires, rateTime time.Time
	usage                      []map[string]any
	tags                       map[string]string
	token, tokenSum            string
	size                       int
}

const bcmpcWorkloadKind = "WorkloadEstimate"

func (w *bcmpcWorkload) out() map[string]any {
	total := 0.0
	for _, u := range w.usage {
		cost, _ := u["cost"].(float64)
		total += cost
	}
	return map[string]any{
		"id": w.id, "name": w.name, "createdAt": budEpoch(w.created), "expiresAt": budEpoch(w.expires),
		"rateType": w.rateType, "rateTimestamp": budEpoch(w.rateTime), "status": w.status,
		"totalCost": bcmpcRound(total), "costCurrency": "USD",
	}
}

func (st *bcmpcStore) workload(id string) (*bcmpcWorkload, *budErr) {
	if w := st.workloads[id]; w != nil {
		return w, nil
	}
	return nil, bcmpcNotFound(id, bcmpcWorkloadKind)
}

// touch stores a changed workload as UPDATING until the next request.
// Caller holds st.mu and has charged next.size.
func (st *bcmpcStore) touch(next *bcmpcWorkload) {
	next.status = "UPDATING"
	st.workloads[next.id] = next
	st.pendingWorkloads = append(st.pendingWorkloads, next.id)
}

func (s *Server) bcmpcCreateWorkload(in map[string]any) (map[string]any, *budErr) {
	tags := bcmpcTagsIn(in)
	if err := bcmpcCheckTags(tags); err != nil {
		return nil, err
	}
	rate := budStr(in, "rateType")
	if rate == "" {
		rate = "BEFORE_DISCOUNTS"
	}
	token, sum := budStr(in, "clientToken"), bcmdSum(in)
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	if token != "" {
		for _, w := range st.workloads {
			if w.token == token {
				if w.tokenSum != sum {
					return nil, bcmpcResourceErr("ConflictException", w.id, bcmpcWorkloadKind, "clientToken %s was used with a different request", token)
				}
				return w.out(), nil
			}
		}
	}
	if err := st.allowRate(rate); err != nil {
		return nil, err
	}
	if len(st.workloads) >= bcmpcMaxWorkloads {
		return nil, bcmpcResourceErr("ServiceQuotaExceededException", "", bcmpcWorkloadKind, "an account can have at most %d workload estimates", bcmpcMaxWorkloads)
	}
	now := time.Now().UTC()
	w := &bcmpcWorkload{
		id: budNewID(), name: budStr(in, "name"), rateType: rate, status: "VALID",
		created: now, expires: now.Add(bcmpcLifetime), rateTime: now, tags: tags, token: token, tokenSum: sum,
	}
	w.size = bcmpcSize(w.name, w.tags, w.token)
	if err := st.charge(0, w.size, w.id, bcmpcWorkloadKind); err != nil {
		return nil, err
	}
	st.workloads[w.id] = w
	return w.out(), nil
}

func (s *Server) bcmpcGetWorkload(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	w, err := st.workload(budStr(in, "identifier"))
	if err != nil {
		return nil, err
	}
	return w.out(), nil
}

func (s *Server) bcmpcUpdateWorkload(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	w, err := st.workload(budStr(in, "identifier"))
	if err != nil {
		return nil, err
	}
	next := *w
	if v, ok := in["name"]; ok {
		next.name, _ = v.(string)
	}
	if t, ok, err := bcmpcExpiry(in, time.Now().UTC()); err != nil {
		return nil, err
	} else if ok {
		next.expires = t
	}
	next.size = w.size - bcmpcSize(w.name) + bcmpcSize(next.name)
	if err := st.charge(w.size, next.size, w.id, bcmpcWorkloadKind); err != nil {
		return nil, err
	}
	st.workloads[w.id] = &next
	return next.out(), nil
}

func (s *Server) bcmpcDeleteWorkload(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	w, err := st.workload(budStr(in, "identifier"))
	if err != nil {
		return nil, err
	}
	st.bytes -= w.size
	delete(st.workloads, w.id)
	return map[string]any{}, nil
}

func (s *Server) bcmpcListWorkloads(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	created, expires := budMap(in, "createdAtFilter"), budMap(in, "expiresAtFilter")
	items := make([]map[string]any, 0, len(st.workloads))
	for _, w := range st.workloads {
		if bcmpcInWindow(w.created, created) && bcmpcInWindow(w.expires, expires) {
			items = append(items, w.out())
		}
	}
	items = bcmpcFiltered(items, budList(in, "filters"), func(item map[string]any, name string) string {
		return budStr(item, map[string]string{"STATUS": "status", "NAME": "name"}[name])
	})
	bcmpcSortByCreated(items)
	return bcmpcPage(in, items, 25)
}

// bcmpcPriceUsage prices one workload usage line.
func bcmpcPriceUsage(item map[string]any, p bcmpcPrice, amount float64) {
	item["quantity"] = map[string]any{"unit": p.unit, "amount": amount}
	item["cost"] = bcmpcRound(amount * p.rate)
	item["currency"], item["status"] = "USD", "VALID"
}

func (s *Server) bcmpcCreateWorkloadUsage(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	w, err := st.workload(budStr(in, "workloadEstimateId"))
	if err != nil {
		return nil, err
	}
	entries := budList(in, "usage")
	if len(w.usage)+len(entries) > bcmpcMaxUsage {
		return nil, bcmpcResourceErr("ServiceQuotaExceededException", w.id, bcmpcWorkloadKind, "a workload estimate can have at most %d usage lines", bcmpcMaxUsage)
	}
	items, errs, added := []any{}, []any{}, []map[string]any{}
	keys, grow := map[string]bool{}, 0
	for _, e := range entries {
		m, _ := e.(map[string]any)
		key := budStr(m, "key")
		p, found := bcmpcLookup(budStr(m, "serviceCode"), budStr(m, "usageType"), budStr(m, "operation"))
		amount, _ := m["amount"].(float64)
		problem := bcmpcCheckHistorical(budMap(m, "historicalUsage"))
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
		case amount < 0:
			problem = "amount must not be negative"
		}
		if problem != "" {
			errs = append(errs, bcmpcBatchErr("key", key, "BAD_REQUEST", problem))
			continue
		}
		keys[key] = true
		item := map[string]any{
			"id": budNewID(), "serviceCode": m["serviceCode"], "usageType": m["usageType"], "operation": m["operation"],
			"location": p.location, "usageAccountId": m["usageAccountId"],
		}
		for _, k := range []string{"group", "historicalUsage"} {
			if v, ok := m[k]; ok {
				item[k] = budClone(v)
			}
		}
		bcmpcPriceUsage(item, p, amount)
		grow += bcmpcSize(item)
		added = append(added, item)
		out := budCloneMap(item)
		out["key"] = key
		items = append(items, out)
	}
	if err := st.charge(w.size, w.size+grow, w.id, bcmpcWorkloadKind); err != nil {
		return nil, err
	}
	next := *w
	next.usage = append(append([]map[string]any{}, w.usage...), added...)
	next.size = w.size + grow
	if len(added) > 0 {
		st.touch(&next)
	} else {
		st.workloads[w.id] = &next
	}
	return map[string]any{"items": items, "errors": errs}, nil
}

func (s *Server) bcmpcUpdateWorkloadUsage(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	w, err := st.workload(budStr(in, "workloadEstimateId"))
	if err != nil {
		return nil, err
	}
	usage := append([]map[string]any{}, w.usage...)
	items, errs, size := []any{}, []any{}, w.size
	for _, e := range budList(in, "usage") {
		m, _ := e.(map[string]any)
		id := budStr(m, "id")
		i := bcmpcIndex(usage, id)
		if i < 0 {
			errs = append(errs, bcmpcBatchErr("id", id, "NOT_FOUND", "usage line does not exist"))
			continue
		}
		item := budCloneMap(usage[i])
		if v, ok := m["group"]; ok {
			item["group"] = v
		}
		if amount, ok := m["amount"].(float64); ok {
			if amount < 0 {
				errs = append(errs, bcmpcBatchErr("id", id, "BAD_REQUEST", "amount must not be negative"))
				continue
			}
			p, _ := bcmpcLookup(budStr(item, "serviceCode"), budStr(item, "usageType"), budStr(item, "operation"))
			bcmpcPriceUsage(item, p, amount)
		}
		size += bcmpcSize(item) - bcmpcSize(usage[i])
		usage[i] = item
		items = append(items, budCloneMap(item))
	}
	if err := st.charge(w.size, size, w.id, bcmpcWorkloadKind); err != nil {
		return nil, err
	}
	next := *w
	next.usage, next.size = usage, size
	if len(items) > 0 {
		st.touch(&next)
	} else {
		st.workloads[w.id] = &next
	}
	return map[string]any{"items": items, "errors": errs}, nil
}

func (s *Server) bcmpcDeleteWorkloadUsage(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	w, err := st.workload(budStr(in, "workloadEstimateId"))
	if err != nil {
		return nil, err
	}
	ids := budStrings(in, "ids")
	kept, errs, freed := bcmpcDeleteItems(w.usage, ids)
	for _, e := range errs {
		e.(map[string]any)["errorCode"] = "NOT_FOUND"
	}
	next := *w
	next.usage, next.size = kept, w.size-freed
	st.bytes -= freed
	if len(errs) < len(ids) {
		st.touch(&next)
	} else {
		st.workloads[w.id] = &next
	}
	return map[string]any{"errors": errs}, nil
}

func (s *Server) bcmpcListWorkloadUsage(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	w, err := st.workload(budStr(in, "workloadEstimateId"))
	if err != nil {
		return nil, err
	}
	return bcmpcPage(in, bcmpcFiltered(w.usage, budList(in, "filters"), bcmpcUsageField), 300)
}

// bcmpcUsageLinesCost prices usage lines without storing anything.
func (s *Server) bcmpcUsageLinesCost(in map[string]any) (map[string]any, *budErr) {
	rate := budStr(in, "rateType")
	st := s.bcmPricingCalculator
	st.lock()
	err := st.allowRate(rate)
	st.mu.Unlock()
	if err != nil {
		return nil, err
	}
	lines, errs, total := []any{}, []any{}, 0.0
	keys := map[string]bool{}
	for _, u := range budList(in, "usage") {
		m, _ := u.(map[string]any)
		key, svc, ut, op := budStr(m, "key"), budStr(m, "serviceCode"), budStr(m, "usageType"), budStr(m, "operation")
		fail := func(code, msg string) {
			errs = append(errs, map[string]any{"key": key, "serviceCode": svc, "usageType": ut, "operation": op, "errorCode": code, "errorMessage": msg})
		}
		p, found := bcmpcLookup(svc, ut, op)
		switch {
		case keys[key]:
			fail("CONFLICT", "key is used twice in this request")
			continue
		case !found:
			fail("NOT_FOUND", "no price for this service code, usage type and operation")
			continue
		}
		keys[key] = true
		amount, _ := m["amount"].(float64)
		cost := bcmpcRound(amount * p.rate)
		total += cost
		lines = append(lines, map[string]any{
			"key": key, "serviceCode": svc, "usageType": ut, "operation": op,
			"estimatedCost": map[string]any{"usageAmount": amount, "usageUnit": p.unit, "cost": cost, "currency": "USD"},
		})
	}
	return map[string]any{"rateType": rate, "totalCost": bcmpcRound(total), "currency": "USD", "lineItems": lines, "errors": errs}, nil
}
