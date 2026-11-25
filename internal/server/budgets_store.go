package server

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	budMaxBudgets       = 1000 // per account
	budMaxNotifications = 10   // per budget
	budMaxSubscribers   = 11   // per notification: 10 email + 1 SNS
	budMaxActions       = 10   // per budget
	budMaxTags          = 200
	budMaxPeriods       = 1000 // performance-history periods per request
)

// budDefaultEnd is the end date AWS gives budgets created without one.
var budDefaultEnd = time.Date(2087, 6, 15, 0, 0, 0, 0, time.UTC)

type budStore struct {
	mu      sync.Mutex
	budgets map[string]*budBudget // account + "/" + name
}

type budBudget struct {
	account, name string
	budget        map[string]any
	notes         []*budNote
	actions       []*budAction
	tags          map[string]string
}

type budNote struct {
	n    map[string]any // ThresholdType and NotificationState filled in
	subs []map[string]any
}

func newBudgetsStore() *budStore { return &budStore{budgets: map[string]*budBudget{}} }

func budKey(account, name string) string { return account + "/" + name }

func budARN(account, name string) string {
	return fmt.Sprintf("arn:aws:budgets::%s:budget/%s", account, name)
}

// get returns the budget or NotFoundException; callers hold st.mu.
func (st *budStore) get(account, name string) (*budBudget, *budErr) {
	b := st.budgets[budKey(account, name)]
	if b == nil {
		return nil, budNotFound("Unable to get budget: %s - the budget doesn't exist.", name)
	}
	return b, nil
}

// sorted returns the account's budgets by name; callers hold st.mu.
func (st *budStore) sorted(account string) []*budBudget {
	var out []*budBudget
	for _, b := range st.budgets {
		if b.account == account {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// ---- budget shape ----

// budPeriodStart is the start of the budget period holding t.
func budPeriodStart(t time.Time, unit string) time.Time {
	y, m, d := t.UTC().Date()
	switch unit {
	case "MONTHLY":
		return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	case "QUARTERLY":
		return time.Date(y, m-(m-1)%3, 1, 0, 0, 0, 0, time.UTC)
	case "ANNUALLY":
		return time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func budNextPeriod(t time.Time, unit string) time.Time {
	switch unit {
	case "MONTHLY":
		return t.AddDate(0, 1, 0)
	case "QUARTERLY":
		return t.AddDate(0, 3, 0)
	case "ANNUALLY":
		return t.AddDate(1, 0, 0)
	}
	return t.AddDate(0, 0, 1)
}

var budCostTypeDefaults = map[string]bool{
	"IncludeTax": true, "IncludeSubscription": true, "UseBlended": false, "IncludeRefund": false,
	"IncludeCredit": false, "IncludeUpfront": true, "IncludeRecurring": true, "IncludeOtherSubscription": true,
	"IncludeSupport": true, "IncludeDiscount": true, "UseAmortized": false,
}

// budNormalizeBudget checks a caller's Budget and fills the fields AWS owns.
// prevStart keeps an existing budget's start when an update omits it.
func budNormalizeBudget(raw map[string]any, prevStart any, now time.Time) (map[string]any, *budErr) {
	b := budCloneMap(raw)
	name, unit, typ := budStr(b, "BudgetName"), budStr(b, "TimeUnit"), budStr(b, "BudgetType")
	if strings.ContainsAny(name, `:\`) || strings.Contains(name, "/action/") {
		return nil, budInvalid("BudgetName can't contain ':' or '\\' or '/action/'.")
	}
	planned := budMap(b, "PlannedBudgetLimits")
	auto := budMap(b, "AutoAdjustData")
	if budMap(b, "BudgetLimit") == nil && len(planned) == 0 && auto == nil {
		return nil, budInvalid("BudgetLimit is required unless PlannedBudgetLimits or AutoAdjustData is set.")
	}
	if len(planned) > 0 && unit != "MONTHLY" && unit != "QUARTERLY" {
		return nil, budInvalid("PlannedBudgetLimits are supported only for MONTHLY and QUARTERLY budgets.")
	}
	if len(budMap(b, "CostFilters")) > 0 && budMap(b, "FilterExpression") != nil {
		return nil, budInvalid("CostFilters and FilterExpression can't both be set.")
	}
	if auto != nil {
		if typ != "COST" {
			return nil, budInvalid("AutoAdjustData is supported only for COST budgets.")
		}
		if budStr(auto, "AutoAdjustType") == "HISTORICAL" && budMap(auto, "HistoricalOptions") == nil {
			return nil, budInvalid("HistoricalOptions is required for HISTORICAL auto-adjusting budgets.")
		}
		auto["LastAutoAdjustTime"] = budEpoch(now)
		if budMap(b, "BudgetLimit") == nil {
			b["BudgetLimit"] = map[string]any{"Amount": "0", "Unit": "USD"}
		}
	}
	period := budMap(b, "TimePeriod")
	start, hasStart := budTime(period["Start"])
	end, hasEnd := budTime(period["End"])
	if !hasStart {
		if t, ok := budTime(prevStart); ok {
			start = t
		} else {
			start = budPeriodStart(now, unit)
		}
	}
	if !hasEnd {
		if unit == "CUSTOM" {
			return nil, budInvalid("CUSTOM budgets need a TimePeriod End.")
		}
		end = budDefaultEnd
	}
	if !start.Before(end) {
		return nil, budInvalid("TimePeriod Start must be before End.")
	}
	b["TimePeriod"] = map[string]any{"Start": budEpoch(start), "End": budEpoch(end)}
	if typ == "COST" || typ == "USAGE" {
		ct := budMap(b, "CostTypes")
		if ct == nil {
			ct = map[string]any{}
			b["CostTypes"] = ct
		}
		for k, v := range budCostTypeDefaults {
			if _, ok := ct[k]; !ok {
				ct[k] = v
			}
		}
	}
	unitName := "USD"
	if l := budMap(b, "BudgetLimit"); l != nil {
		unitName = budStr(l, "Unit")
	}
	// ponytail: Stackyard records no billing data, so spend is always zero.
	zero := map[string]any{"Amount": "0", "Unit": unitName}
	b["CalculatedSpend"] = map[string]any{"ActualSpend": zero, "ForecastedSpend": budCloneMap(zero)}
	b["LastUpdatedTime"] = budEpoch(now)
	delete(b, "HealthStatus")
	if budStr(b, "BillingViewArn") != "" {
		b["HealthStatus"] = map[string]any{"Status": "HEALTHY", "LastUpdatedTime": budEpoch(now)}
	}
	return b, nil
}

// budBudgetOut clones a stored budget; FilterExpression only when asked for.
func budBudgetOut(b *budBudget, showFilter bool) map[string]any {
	out := budCloneMap(b.budget)
	if !showFilter {
		delete(out, "FilterExpression")
	}
	return out
}

// ---- notifications and subscribers ----

func budNormNote(raw map[string]any) map[string]any {
	n := budCloneMap(raw)
	if budStr(n, "ThresholdType") == "" {
		n["ThresholdType"] = "PERCENTAGE"
	}
	if budStr(n, "NotificationState") == "" {
		n["NotificationState"] = "OK"
	}
	return n
}

// budNoteKey identifies a notification; NotificationState isn't part of it.
func budNoteKey(n map[string]any) string {
	t := budStr(n, "ThresholdType")
	if t == "" {
		t = "PERCENTAGE"
	}
	th, _ := n["Threshold"].(float64)
	return budStr(n, "NotificationType") + "|" + budStr(n, "ComparisonOperator") + "|" + strconv.FormatFloat(th, 'g', -1, 64) + "|" + t
}

func budSubKey(s map[string]any) string {
	return budStr(s, "SubscriptionType") + "|" + budStr(s, "Address")
}

func budCheckSubscriber(sub map[string]any) *budErr {
	addr := budStr(sub, "Address")
	switch budStr(sub, "SubscriptionType") {
	case "EMAIL":
		if at := strings.Index(addr, "@"); at <= 0 || at == len(addr)-1 || strings.ContainsAny(addr, " \t\r\n") {
			return budInvalid("Subscriber address is not a valid email address.")
		}
	case "SNS":
		if !strings.HasPrefix(addr, "arn:aws") || !strings.Contains(addr, ":sns:") {
			return budInvalid("Subscriber address is not a valid SNS topic ARN.")
		}
	}
	return nil
}

// budCheckSubscribers validates a subscriber list: no duplicates, at most 10
// email and 1 SNS subscriber.
func budCheckSubscribers(subs []map[string]any) *budErr {
	seen := map[string]bool{}
	emails, sns := 0, 0
	for _, sub := range subs {
		if err := budCheckSubscriber(sub); err != nil {
			return err
		}
		if seen[budSubKey(sub)] {
			return budErrorf("DuplicateRecordException", "Subscriber %s is listed twice.", budStr(sub, "Address"))
		}
		seen[budSubKey(sub)] = true
		if budStr(sub, "SubscriptionType") == "SNS" {
			sns++
		} else {
			emails++
		}
	}
	if emails > 10 || sns > 1 {
		return budErrorf("CreationLimitExceededException", "A notification can have at most 10 email subscribers and 1 SNS subscriber.")
	}
	return nil
}

func budSubsIn(list []any) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			out = append(out, budCloneMap(m))
		}
	}
	return out
}

func (b *budBudget) note(raw map[string]any) (int, *budErr) {
	key := budNoteKey(raw)
	for i, n := range b.notes {
		if budNoteKey(n.n) == key {
			return i, nil
		}
	}
	return -1, budNotFound("Unable to find the notification on budget %s.", b.name)
}

func budSubIndex(subs []map[string]any, sub map[string]any) int {
	for i, s := range subs {
		if budSubKey(s) == budSubKey(sub) {
			return i
		}
	}
	return -1
}

// ---- budget handlers ----

func (s *Server) budCreateBudget(in map[string]any) (map[string]any, *budErr) {
	account, now := budStr(in, "AccountId"), time.Now().UTC()
	budget, err := budNormalizeBudget(budMap(in, "Budget"), nil, now)
	if err != nil {
		return nil, err
	}
	tags, err := budTagsIn(budList(in, "ResourceTags"), "InvalidParameterException")
	if err != nil {
		return nil, err
	}
	var notes []*budNote
	seen := map[string]bool{}
	for _, item := range budList(in, "NotificationsWithSubscribers") {
		m, _ := item.(map[string]any)
		n := budNormNote(budMap(m, "Notification"))
		subs := budSubsIn(budList(m, "Subscribers"))
		if err := budCheckSubscribers(subs); err != nil {
			return nil, err
		}
		if seen[budNoteKey(n)] {
			return nil, budErrorf("DuplicateRecordException", "Notification is listed twice.")
		}
		seen[budNoteKey(n)] = true
		notes = append(notes, &budNote{n: n, subs: subs})
	}
	if len(notes) > budMaxNotifications {
		return nil, budErrorf("CreationLimitExceededException", "A budget can have at most %d notifications.", budMaxNotifications)
	}
	name := budStr(budget, "BudgetName")
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.budgets[budKey(account, name)] != nil {
		return nil, budErrorf("DuplicateRecordException", "Error creating budget: %s - the budget already exists.", name)
	}
	if len(st.sorted(account)) >= budMaxBudgets {
		return nil, budErrorf("CreationLimitExceededException", "The account already has %d budgets.", budMaxBudgets)
	}
	st.budgets[budKey(account, name)] = &budBudget{account: account, name: name, budget: budget, notes: notes, tags: tags}
	return map[string]any{}, nil
}

func (s *Server) budDescribeBudget(in map[string]any) (map[string]any, *budErr) {
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	b, err := st.get(budStr(in, "AccountId"), budStr(in, "BudgetName"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"Budget": budBudgetOut(b, budBool(in, "ShowFilterExpression"))}, nil
}

func (s *Server) budDescribeBudgets(in map[string]any) (map[string]any, *budErr) {
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	all := st.sorted(budStr(in, "AccountId"))
	start, end, next, err := budPage(budStr(in, "NextToken"), len(all), budInt(in, "MaxResults", 100), "InvalidNextTokenException")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, b := range all[start:end] {
		items = append(items, budBudgetOut(b, budBool(in, "ShowFilterExpression")))
	}
	return map[string]any{"Budgets": items, "NextToken": next}, nil
}

func (s *Server) budUpdateBudget(in map[string]any) (map[string]any, *budErr) {
	account, raw := budStr(in, "AccountId"), budMap(in, "NewBudget")
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	b, err := st.get(account, budStr(raw, "BudgetName"))
	if err != nil {
		return nil, err
	}
	if budStr(raw, "BudgetType") != budStr(b.budget, "BudgetType") {
		return nil, budInvalid("BudgetType can't be changed.")
	}
	budget, err := budNormalizeBudget(raw, budMap(b.budget, "TimePeriod")["Start"], time.Now().UTC())
	if err != nil {
		return nil, err
	}
	b.budget = budget
	return map[string]any{}, nil
}

func (s *Server) budDeleteBudget(in map[string]any) (map[string]any, *budErr) {
	account, name := budStr(in, "AccountId"), budStr(in, "BudgetName")
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.get(account, name); err != nil {
		return nil, err
	}
	delete(st.budgets, budKey(account, name))
	return map[string]any{}, nil
}

func (s *Server) budPerformanceHistory(in map[string]any) (map[string]any, *budErr) {
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	b, err := st.get(budStr(in, "AccountId"), budStr(in, "BudgetName"))
	if err != nil {
		return nil, err
	}
	unit := budStr(b.budget, "TimeUnit")
	stored := budMap(b.budget, "TimePeriod")
	from, _ := budTime(stored["Start"])
	to, _ := budTime(stored["End"])
	if req := budMap(in, "TimePeriod"); req != nil {
		if t, ok := budTime(req["Start"]); ok {
			from = t
		}
		if t, ok := budTime(req["End"]); ok {
			to = t
		}
	}
	if !from.Before(to) {
		return nil, budInvalid("TimePeriod Start must be before End.")
	}
	if now := time.Now().UTC(); now.Before(to) {
		to = now
	}
	limit := budMap(b.budget, "BudgetLimit")
	planned := budMap(b.budget, "PlannedBudgetLimits")
	currency := "USD"
	if limit != nil {
		currency = budStr(limit, "Unit")
	}
	var periods []any
	// ponytail: at most budMaxPeriods periods from the start; page through them.
	p := budPeriodStart(from, unit)
	if unit == "CUSTOM" {
		p = from
	}
	for i := 0; i < budMaxPeriods && (i == 0 || p.Before(to)); i++ {
		next := budNextPeriod(p, unit)
		if unit == "CUSTOM" {
			next, _ = budTime(stored["End"])
		}
		budgeted := limit
		if pl := budMap(planned, strconv.FormatInt(p.Unix(), 10)); pl != nil {
			budgeted = pl
		}
		if budgeted == nil {
			budgeted = map[string]any{"Amount": "0", "Unit": currency}
		}
		periods = append(periods, map[string]any{
			"BudgetedAmount": budCloneMap(budgeted),
			"ActualAmount":   map[string]any{"Amount": "0", "Unit": budStr(budgeted, "Unit")},
			"TimePeriod":     map[string]any{"Start": budEpoch(p), "End": budEpoch(next)},
		})
		if unit == "CUSTOM" {
			break
		}
		p = next
	}
	start, end, next, err := budPage(budStr(in, "NextToken"), len(periods), budInt(in, "MaxResults", 100), "InvalidNextTokenException")
	if err != nil {
		return nil, err
	}
	hist := map[string]any{
		"BudgetName": b.name, "BudgetType": b.budget["BudgetType"], "TimeUnit": unit,
		"BudgetedAndActualAmountsList": periods[start:end],
	}
	for _, k := range []string{"CostFilters", "CostTypes", "BillingViewArn", "FilterExpression", "Metrics"} {
		if v, ok := b.budget[k]; ok {
			hist[k] = budClone(v)
		}
	}
	return map[string]any{"BudgetPerformanceHistory": hist, "NextToken": next}, nil
}

// ---- notification handlers ----

func (s *Server) budCreateNotification(in map[string]any) (map[string]any, *budErr) {
	n := budNormNote(budMap(in, "Notification"))
	subs := budSubsIn(budList(in, "Subscribers"))
	if err := budCheckSubscribers(subs); err != nil {
		return nil, err
	}
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	b, err := st.get(budStr(in, "AccountId"), budStr(in, "BudgetName"))
	if err != nil {
		return nil, err
	}
	if _, err := b.note(n); err == nil {
		return nil, budErrorf("DuplicateRecordException", "The notification already exists on budget %s.", b.name)
	}
	if len(b.notes) >= budMaxNotifications {
		return nil, budErrorf("CreationLimitExceededException", "A budget can have at most %d notifications.", budMaxNotifications)
	}
	b.notes = append(b.notes, &budNote{n: n, subs: subs})
	return map[string]any{}, nil
}

func (s *Server) budUpdateNotification(in map[string]any) (map[string]any, *budErr) {
	nn := budNormNote(budMap(in, "NewNotification"))
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	b, err := st.get(budStr(in, "AccountId"), budStr(in, "BudgetName"))
	if err != nil {
		return nil, err
	}
	i, err := b.note(budMap(in, "OldNotification"))
	if err != nil {
		return nil, err
	}
	if j, _ := b.note(nn); j >= 0 && j != i {
		return nil, budErrorf("DuplicateRecordException", "The new notification already exists on budget %s.", b.name)
	}
	b.notes[i].n = nn
	return map[string]any{}, nil
}

func (s *Server) budDeleteNotification(in map[string]any) (map[string]any, *budErr) {
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	b, err := st.get(budStr(in, "AccountId"), budStr(in, "BudgetName"))
	if err != nil {
		return nil, err
	}
	i, err := b.note(budMap(in, "Notification"))
	if err != nil {
		return nil, err
	}
	b.notes = append(b.notes[:i:i], b.notes[i+1:]...)
	return map[string]any{}, nil
}

func (s *Server) budNotificationsForBudget(in map[string]any) (map[string]any, *budErr) {
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	b, err := st.get(budStr(in, "AccountId"), budStr(in, "BudgetName"))
	if err != nil {
		return nil, err
	}
	start, end, next, err := budPage(budStr(in, "NextToken"), len(b.notes), budInt(in, "MaxResults", 100), "InvalidNextTokenException")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, n := range b.notes[start:end] {
		items = append(items, budCloneMap(n.n))
	}
	return map[string]any{"Notifications": items, "NextToken": next}, nil
}

func (s *Server) budNotificationsForAccount(in map[string]any) (map[string]any, *budErr) {
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	var with []*budBudget
	for _, b := range st.sorted(budStr(in, "AccountId")) {
		if len(b.notes) > 0 {
			with = append(with, b)
		}
	}
	start, end, next, err := budPage(budStr(in, "NextToken"), len(with), budInt(in, "MaxResults", 100), "InvalidNextTokenException")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, b := range with[start:end] {
		notes := make([]any, 0, len(b.notes))
		for _, n := range b.notes {
			notes = append(notes, budCloneMap(n.n))
		}
		items = append(items, map[string]any{"BudgetName": b.name, "Notifications": notes})
	}
	return map[string]any{"BudgetNotificationsForAccount": items, "NextToken": next}, nil
}

// ---- subscriber handlers ----

// budLockedNote locks the store and finds the request's budget and
// notification; on success the caller must unlock st.mu.
func (s *Server) budLockedNote(in map[string]any) (*budNote, *budErr) {
	st := s.budgets
	st.mu.Lock()
	b, err := st.get(budStr(in, "AccountId"), budStr(in, "BudgetName"))
	if err != nil {
		st.mu.Unlock()
		return nil, err
	}
	i, err := b.note(budMap(in, "Notification"))
	if err != nil {
		st.mu.Unlock()
		return nil, err
	}
	return b.notes[i], nil
}

func (s *Server) budCreateSubscriber(in map[string]any) (map[string]any, *budErr) {
	sub := budCloneMap(budMap(in, "Subscriber"))
	if err := budCheckSubscriber(sub); err != nil {
		return nil, err
	}
	n, err := s.budLockedNote(in)
	if err != nil {
		return nil, err
	}
	defer s.budgets.mu.Unlock()
	if budSubIndex(n.subs, sub) >= 0 {
		return nil, budErrorf("DuplicateRecordException", "Subscriber %s already exists.", budStr(sub, "Address"))
	}
	subs := append(append([]map[string]any{}, n.subs...), sub)
	if err := budCheckSubscribers(subs); err != nil {
		return nil, err
	}
	n.subs = subs
	return map[string]any{}, nil
}

func (s *Server) budUpdateSubscriber(in map[string]any) (map[string]any, *budErr) {
	old, nu := budMap(in, "OldSubscriber"), budCloneMap(budMap(in, "NewSubscriber"))
	n, err := s.budLockedNote(in)
	if err != nil {
		return nil, err
	}
	defer s.budgets.mu.Unlock()
	i := budSubIndex(n.subs, old)
	if i < 0 {
		return nil, budNotFound("Subscriber %s doesn't exist.", budStr(old, "Address"))
	}
	if j := budSubIndex(n.subs, nu); j >= 0 && j != i {
		return nil, budErrorf("DuplicateRecordException", "Subscriber %s already exists.", budStr(nu, "Address"))
	}
	subs := append([]map[string]any{}, n.subs...)
	subs[i] = nu
	if err := budCheckSubscribers(subs); err != nil {
		return nil, err
	}
	n.subs = subs
	return map[string]any{}, nil
}

func (s *Server) budDeleteSubscriber(in map[string]any) (map[string]any, *budErr) {
	sub := budMap(in, "Subscriber")
	n, err := s.budLockedNote(in)
	if err != nil {
		return nil, err
	}
	defer s.budgets.mu.Unlock()
	i := budSubIndex(n.subs, sub)
	if i < 0 {
		return nil, budNotFound("Subscriber %s doesn't exist.", budStr(sub, "Address"))
	}
	if len(n.subs) == 1 {
		return nil, budInvalid("A notification must keep at least one subscriber; delete the notification instead.")
	}
	n.subs = append(n.subs[:i:i], n.subs[i+1:]...)
	return map[string]any{}, nil
}

func (s *Server) budSubscribersForNotification(in map[string]any) (map[string]any, *budErr) {
	n, err := s.budLockedNote(in)
	if err != nil {
		return nil, err
	}
	defer s.budgets.mu.Unlock()
	start, end, next, err := budPage(budStr(in, "NextToken"), len(n.subs), budInt(in, "MaxResults", 100), "InvalidNextTokenException")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, sub := range n.subs[start:end] {
		items = append(items, budCloneMap(sub))
	}
	return map[string]any{"Subscribers": items, "NextToken": next}, nil
}

// ---- tags ----

// budTagTarget resolves a budget or budget-action ARN to its tag map holder;
// callers hold st.mu.
func (st *budStore) tagTarget(arn string) (*map[string]string, *budErr) {
	rest, ok := strings.CutPrefix(arn, "arn:aws:budgets::")
	account, path, ok2 := strings.Cut(rest, ":budget/")
	if !ok || !ok2 {
		return nil, budInvalid("ResourceARN %s is not a budget or budget action ARN.", arn)
	}
	name, actionID, isAction := path, "", false
	if i := strings.LastIndex(path, "/action/"); i >= 0 {
		name, actionID, isAction = path[:i], path[i+len("/action/"):], true
	}
	b := st.budgets[budKey(account, name)]
	if b == nil {
		return nil, budNotFound("Resource %s doesn't exist.", arn)
	}
	if !isAction {
		return &b.tags, nil
	}
	if a := b.action(actionID); a != nil {
		return &a.tags, nil
	}
	return nil, budNotFound("Resource %s doesn't exist.", arn)
}

func (s *Server) budTagResource(in map[string]any) (map[string]any, *budErr) {
	add, err := budTagsIn(budList(in, "ResourceTags"), "InvalidParameterException")
	if err != nil {
		return nil, err
	}
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, err := st.tagTarget(budStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	merged, err := budMergeTags(*tags, add, budMaxTags, "ServiceQuotaExceededException")
	if err != nil {
		return nil, err
	}
	*tags = merged
	return map[string]any{}, nil
}

func (s *Server) budUntagResource(in map[string]any) (map[string]any, *budErr) {
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, err := st.tagTarget(budStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	*tags = budDropTags(*tags, budStrings(in, "ResourceTagKeys"))
	return map[string]any{}, nil
}

func (s *Server) budListTags(in map[string]any) (map[string]any, *budErr) {
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, err := st.tagTarget(budStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ResourceTags": budTagsOut(*tags)}, nil
}
