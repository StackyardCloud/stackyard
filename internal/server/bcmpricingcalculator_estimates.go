package server

import (
	"sort"
	"strings"
	"time"
)

// Bill estimates price a snapshot of a bill scenario. The snapshot and its
// priced line items never change after creation; only name, expiry and tags
// do.

type bcmpcEstimate struct {
	id, name, scenarioID, status string
	sharing, costCategoryArn     string
	created, expires, start, end time.Time
	lines, commitments           []map[string]any
	inputUsage, inputCommits     []map[string]any
	costSummary                  map[string]any
	tags                         map[string]string
	token, tokenSum              string
	size                         int
}

const (
	bcmpcEstimateKind = "BillEstimate"
	bcmpcHoursPerMo   = 730
)

// bcmpcPriced is one usage line while commitments are applied to it.
type bcmpcPriced struct {
	line      map[string]any
	price     bcmpcPrice
	quantity  float64
	remaining float64 // on-demand cost not yet covered by a commitment
}

func bcmpcLine(id, lineType, serviceCode, usageType, operation, account string, quantity float64, unit string, cost float64) map[string]any {
	return map[string]any{
		"id": id, "lineItemId": budNewID(), "lineItemType": lineType, "serviceCode": serviceCode, "usageType": usageType,
		"operation": operation, "location": bcmpcUSEast, "payerAccountId": bcmpcAccountID, "usageAccountId": account,
		"estimatedUsageQuantity":  map[string]any{"amount": bcmpcRound(quantity), "unit": unit},
		"estimatedCost":           bcmpcCost(cost),
		"historicalUsageQuantity": map[string]any{"amount": 0.0, "unit": unit},
		"historicalCost":          bcmpcCost(0),
	}
}

// bcmpcPriceScenario prices usage on demand, then applies Reserved Instances
// (covering matching instance hours) and Savings Plans (covering eligible
// spend at bcmpcSPDiscount) in modification order. Stackyard records no
// billing history, so historical cost is zero.
func bcmpcPriceScenario(sc *bcmpcScenario) (lines, commitments []map[string]any) {
	hours := sc.end.Sub(sc.start).Hours()
	priced := make([]*bcmpcPriced, 0, len(sc.usage))
	for _, u := range sc.usage {
		p, _ := bcmpcLookup(budStr(u, "serviceCode"), budStr(u, "usageType"), budStr(u, "operation"))
		q := bcmpcQuantity(budList(u, "quantities"), p, sc.start, sc.end)
		line := bcmpcLine(budStr(u, "id"), "Usage", budStr(u, "serviceCode"), budStr(u, "usageType"), budStr(u, "operation"), budStr(u, "usageAccountId"), q, p.unit, q*p.rate)
		if az, ok := u["availabilityZone"]; ok {
			line["availabilityZone"] = az
		}
		priced = append(priced, &bcmpcPriced{line: line, price: p, quantity: q, remaining: q * p.rate})
		lines = append(lines, line)
	}
	for _, c := range sc.commits {
		action := budMap(c, "commitmentAction")
		id, account := budStr(c, "id"), budStr(c, "usageAccountId")
		if ri := budMap(action, "addReservedInstanceAction"); ri != nil {
			offeringID := budStr(ri, "reservedInstancesOfferingId")
			off := bcmpcRIOfferings[offeringID]
			count := float64(budInt(ri, "instanceCount", 1))
			lines = append(lines, bcmpcLine(id, "RIFee", "AmazonEC2", "HeavyUsage:"+strings.TrimPrefix(off.usageType, "BoxUsage:"), "RunInstances", account, count*hours, "Hrs", count*(off.hourly*hours+off.upfront)))
			coverHours := count * hours
			for _, pl := range priced {
				if coverHours <= 0 || budStr(pl.line, "usageType") != off.usageType || pl.price.rate == 0 {
					continue
				}
				covered := min(coverHours, pl.remaining/pl.price.rate)
				if covered <= 0 {
					continue
				}
				coverHours -= covered
				pl.remaining -= covered * pl.price.rate
				lines = append(lines, bcmpcLine(budStr(pl.line, "id"), "DiscountedUsage", "AmazonEC2", off.usageType, "RunInstances", account, covered, "Hrs", -covered*pl.price.rate))
			}
			commitments = append(commitments, map[string]any{
				"id": id, "purchaseAgreementType": "RESERVED_INSTANCE", "offeringId": offeringID, "usageAccountId": account,
				"region": "us-east-1", "termLength": off.term, "paymentOption": off.payment,
				"upfrontPayment": bcmpcCost(count * off.upfront), "monthlyPayment": bcmpcCost(count * off.hourly * bcmpcHoursPerMo),
			})
		}
		if sp := budMap(action, "addSavingsPlanAction"); sp != nil {
			commit, _ := sp["commitment"].(float64)
			lines = append(lines, bcmpcLine(id, "SavingsPlanRecurringFee", "ComputeSavingsPlans", "ComputeSP:1yrNoUpfront", "SavingsPlan", account, hours, "Hrs", commit*hours))
			budget := commit * hours / (1 - bcmpcSPDiscount) // on-demand spend the commitment covers
			for _, pl := range priced {
				if budget <= 0 || !pl.price.spEligible || pl.remaining <= 0 {
					continue
				}
				covered := min(budget, pl.remaining)
				budget -= covered
				pl.remaining -= covered
				lines = append(lines, bcmpcLine(budStr(pl.line, "id"), "SavingsPlanNegation", budStr(pl.line, "serviceCode"), budStr(pl.line, "usageType"), budStr(pl.line, "operation"), account, 0, pl.price.unit, -covered))
			}
			commitments = append(commitments, map[string]any{
				"id": id, "purchaseAgreementType": "SAVINGS_PLANS", "offeringId": budStr(sp, "savingsPlanOfferingId"), "usageAccountId": account,
				"region": "us-east-1", "termLength": "1yr", "paymentOption": "No Upfront",
				"upfrontPayment": bcmpcCost(0), "monthlyPayment": bcmpcCost(commit * bcmpcHoursPerMo),
			})
		}
		// ponytail: negate actions remove commitments Stackyard never sold, so they price at zero.
	}
	return lines, commitments
}

func bcmpcSummary(lines []map[string]any) map[string]any {
	total := 0.0
	perService := map[string]float64{}
	for _, l := range lines {
		cost, _ := budMap(l, "estimatedCost")["amount"].(float64)
		total += cost
		perService[budStr(l, "serviceCode")] += cost
	}
	diff := func(v float64) map[string]any {
		return map[string]any{"historicalCost": bcmpcCost(0), "estimatedCost": bcmpcCost(v)}
	}
	services := map[string]any{}
	for k, v := range perService {
		services[k] = diff(v)
	}
	return map[string]any{"totalCostDifference": diff(total), "serviceCostDifferences": services}
}

func (e *bcmpcEstimate) out() map[string]any {
	out := map[string]any{
		"id": e.id, "name": e.name, "status": e.status, "billInterval": bcmpcIntervalOut(e.start, e.end),
		"createdAt": budEpoch(e.created), "expiresAt": budEpoch(e.expires), "groupSharingPreference": e.sharing,
	}
	if e.status == "COMPLETE" {
		out["costSummary"] = budClone(e.costSummary)
	}
	if e.costCategoryArn != "" {
		out["costCategoryGroupSharingPreferenceArn"] = e.costCategoryArn
		out["costCategoryGroupSharingPreferenceEffectiveDate"] = budEpoch(e.created)
	}
	return out
}

func (st *bcmpcStore) estimate(id string) (*bcmpcEstimate, *budErr) {
	if e := st.estimates[id]; e != nil {
		return e, nil
	}
	return nil, bcmpcNotFound(id, bcmpcEstimateKind)
}

func (s *Server) bcmpcCreateEstimate(in map[string]any) (map[string]any, *budErr) {
	tags := bcmpcTagsIn(in)
	if err := bcmpcCheckTags(tags); err != nil {
		return nil, err
	}
	token, sum := budStr(in, "clientToken"), bcmdSum(in)
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	if token != "" {
		for _, e := range st.estimates {
			if e.token == token {
				if e.tokenSum != sum {
					return nil, bcmpcResourceErr("ConflictException", e.id, bcmpcEstimateKind, "clientToken %s was used with a different request", token)
				}
				return e.out(), nil
			}
		}
	}
	now := time.Now().UTC()
	sc, err := st.scenario(budStr(in, "billScenarioId"), true, now)
	if err != nil {
		return nil, err
	}
	if len(st.estimates) >= bcmpcMaxEstimates {
		return nil, bcmpcResourceErr("ServiceQuotaExceededException", "", bcmpcEstimateKind, "an account can have at most %d bill estimates", bcmpcMaxEstimates)
	}
	lines, commitments := bcmpcPriceScenario(sc)
	e := &bcmpcEstimate{
		id: budNewID(), name: budStr(in, "name"), scenarioID: sc.id, status: "IN_PROGRESS",
		sharing: sc.sharing, costCategoryArn: sc.costCategoryArn, created: now, expires: now.Add(bcmpcLifetime),
		start: sc.start, end: sc.end, lines: lines, commitments: commitments,
		inputUsage: sc.usage, inputCommits: sc.commits, costSummary: bcmpcSummary(lines),
		tags: tags, token: token, tokenSum: sum,
	}
	e.size = bcmpcSize(e.name, e.lines, e.commitments, e.inputUsage, e.inputCommits, e.costSummary, e.tags, e.token)
	if err := st.charge(0, e.size, e.id, bcmpcEstimateKind); err != nil {
		return nil, err
	}
	st.estimates[e.id] = e
	st.pendingEstimates = append(st.pendingEstimates, e.id)
	return e.out(), nil
}

func (s *Server) bcmpcGetEstimate(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	e, err := st.estimate(budStr(in, "identifier"))
	if err != nil {
		return nil, err
	}
	return e.out(), nil
}

func (s *Server) bcmpcUpdateEstimate(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	e, err := st.estimate(budStr(in, "identifier"))
	if err != nil {
		return nil, err
	}
	next := *e
	if v, ok := in["name"]; ok {
		next.name, _ = v.(string)
	}
	if t, ok, err := bcmpcExpiry(in, time.Now().UTC()); err != nil {
		return nil, err
	} else if ok {
		next.expires = t
	}
	next.size = e.size - bcmpcSize(e.name) + bcmpcSize(next.name)
	if err := st.charge(e.size, next.size, e.id, bcmpcEstimateKind); err != nil {
		return nil, err
	}
	st.estimates[e.id] = &next
	return next.out(), nil
}

func (s *Server) bcmpcDeleteEstimate(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	e, err := st.estimate(budStr(in, "identifier"))
	if err != nil {
		return nil, err
	}
	if e.status == "IN_PROGRESS" {
		return nil, bcmpcResourceErr("ConflictException", e.id, bcmpcEstimateKind, "bill estimate %s is still in progress", e.id)
	}
	st.bytes -= e.size
	delete(st.estimates, e.id)
	return map[string]any{}, nil
}

func (s *Server) bcmpcListEstimates(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	created, expires := budMap(in, "createdAtFilter"), budMap(in, "expiresAtFilter")
	items := make([]map[string]any, 0, len(st.estimates))
	for _, e := range st.estimates {
		if bcmpcInWindow(e.created, created) && bcmpcInWindow(e.expires, expires) {
			items = append(items, map[string]any{
				"id": e.id, "name": e.name, "status": e.status, "billInterval": bcmpcIntervalOut(e.start, e.end),
				"createdAt": budEpoch(e.created), "expiresAt": budEpoch(e.expires),
			})
		}
	}
	items = bcmpcFiltered(items, budList(in, "filters"), func(item map[string]any, name string) string {
		return budStr(item, map[string]string{"STATUS": "status", "NAME": "name"}[name])
	})
	bcmpcSortByCreated(items)
	return bcmpcPage(in, items, 25)
}

// bcmpcEstimateList pages one of an estimate's stored lists.
func (s *Server) bcmpcEstimateList(in map[string]any, pick func(*bcmpcEstimate) []map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	e, err := st.estimate(budStr(in, "billEstimateId"))
	if err != nil {
		return nil, err
	}
	items := bcmpcFiltered(pick(e), budList(in, "filters"), bcmpcUsageField)
	return bcmpcPage(in, items, 25)
}

func (s *Server) bcmpcListLineItems(in map[string]any) (map[string]any, *budErr) {
	return s.bcmpcEstimateList(in, func(e *bcmpcEstimate) []map[string]any {
		lines := append([]map[string]any{}, e.lines...)
		sort.SliceStable(lines, func(i, j int) bool { return budStr(lines[i], "serviceCode") < budStr(lines[j], "serviceCode") })
		return lines
	})
}

func (s *Server) bcmpcListEstimateCommitments(in map[string]any) (map[string]any, *budErr) {
	return s.bcmpcEstimateList(in, func(e *bcmpcEstimate) []map[string]any { return e.commitments })
}

func (s *Server) bcmpcListEstimateInputUsage(in map[string]any) (map[string]any, *budErr) {
	return s.bcmpcEstimateList(in, func(e *bcmpcEstimate) []map[string]any { return e.inputUsage })
}

func (s *Server) bcmpcListEstimateInputCommits(in map[string]any) (map[string]any, *budErr) {
	return s.bcmpcEstimateList(in, func(e *bcmpcEstimate) []map[string]any { return e.inputCommits })
}
