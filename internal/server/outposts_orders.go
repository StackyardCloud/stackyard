package server

import (
	"math"
	"strings"
	"time"
)

// Outposts quotes, orders and billing. An order advances one step per
// GetOrder (RECEIVED -> PENDING -> PROCESSING -> INSTALLING -> FULFILLED);
// on fulfilment each line item unit installs one compute asset and the
// order's subscription becomes ACTIVE.

const opstQuoteLifetime = 30 * 24 * time.Hour

var (
	opstOrderSteps = []string{"RECEIVED", "PENDING", "PROCESSING", "INSTALLING", "FULFILLED"}
	opstLineSteps  = []string{"PREPARING", "BUILDING", "SHIPPED", "INSTALLING", "INSTALLED"}
)

type opstLine struct {
	id, catalog string
	quantity    int
}

type opstQuoteOption struct {
	id    string
	lines []opstLine
}

type opstQuote struct {
	id, outpostID, country, description, status, orderID string
	capacities, constraints                              []any
	payments, terms                                      []string
	option                                               *opstQuoteOption
	created, expires                                     time.Time
}

type opstOrder struct {
	id, outpostID, quoteID, quoteOption, payment, term, subscriptionID, tracking string
	step                                                                         int // index into opstOrderSteps
	cancelled                                                                    bool
	lines                                                                        []opstLine
	submitted, fulfilled                                                         time.Time
}

type opstSubscription struct {
	id, typ, payment, term, clientToken string
	orderIDs                            []string
	monthlyList                         float64
	begin                               time.Time
}

func (o *opstOrder) status() string {
	if o.cancelled {
		return "CANCELLED"
	}
	return opstOrderSteps[o.step]
}

func (o *opstOrder) monthlyList() float64 {
	total := 0.0
	for _, l := range o.lines {
		total += opstCatalogItemByID(l.catalog).monthly * float64(l.quantity)
	}
	return total
}

func opstStripARN(id string) string {
	if i := strings.LastIndexByte(id, '/'); i >= 0 {
		return id[i+1:]
	}
	return id
}

// ---- quotes ----

func (st *opstStore) quote(id string, now time.Time) (*opstQuote, *budErr) {
	q := st.quotes[opstStripARN(id)]
	if q == nil {
		return nil, opstNotFound("Quote", id)
	}
	if q.status == "CREATED" && !now.Before(q.expires) {
		q.status = "EXPIRED"
	}
	return q, nil
}

func opstListMax(in map[string]any, keys ...string) *budErr {
	for _, k := range keys {
		if len(budList(in, k)) > opstMaxListItems {
			return opstInvalid("%s can have at most %d entries", k, opstMaxListItems)
		}
	}
	return nil
}

// options sizes the hardware for a quote: EC2 capacities in vCPUs per
// family pick catalog items; with none, one unit of the outpost's form factor.
func (st *opstStore) quoteOption(q *opstQuote) (*opstQuoteOption, *budErr) {
	vcpus := map[string]float64{}
	var families []string
	for _, c := range q.capacities {
		m := c.(map[string]any)
		if budStr(m, "QuoteCapacityType") != "EC2" {
			continue
		}
		f := opstFamilyOf(budStr(m, "Unit"))
		item := opstCatalogByFamily(f)
		if item == nil {
			return nil, opstInvalid("no catalog item offers EC2 family %q", budStr(m, "Unit"))
		}
		if o := st.outposts[q.outpostID]; o != nil && item.class != o.hardware {
			return nil, opstInvalid("EC2 family %s comes as %s hardware; outpost %s takes %s", f, item.class, o.id, o.hardware)
		}
		qty, _ := m["Quantity"].(float64)
		if qty < 0 || math.IsNaN(qty) || math.IsInf(qty, 0) {
			return nil, opstInvalid("capacity quantities must be non-negative")
		}
		if _, seen := vcpus[f]; !seen {
			families = append(families, f)
		}
		vcpus[f] += qty
	}
	opt := &opstQuoteOption{id: "oqo-" + opstHex(17)}
	for _, f := range families {
		c := opstCatalogByFamily(f)
		n := int(math.Max(1, math.Ceil(vcpus[f]/float64(c.vcpus))))
		if vcpus[f] > float64(opstMaxQuantity*c.vcpus) {
			return nil, opstInvalid("a quote can size at most %d units of %s", opstMaxQuantity, c.id)
		}
		opt.lines = append(opt.lines, opstLine{catalog: c.id, quantity: n})
	}
	if len(opt.lines) == 0 {
		item := opstCatalog[0]
		if o := st.outposts[q.outpostID]; o != nil && o.hardware == "SERVER" {
			item = opstCatalog[3]
		}
		opt.lines = []opstLine{{catalog: item.id, quantity: 1}}
	}
	return opt, nil
}

func (st *opstStore) existingVCPUs(outpostID string) map[string]int {
	out := map[string]int{}
	for _, a := range st.assets {
		if a.outpostID == outpostID && a.state == "ACTIVE" {
			out[a.family] += a.maxVcpus
		}
	}
	return out
}

func opstCapList(byFamily map[string]int) []any {
	out := []any{}
	for _, c := range opstCatalog {
		if n := byFamily[c.family]; n > 0 {
			out = append(out, map[string]any{"QuoteCapacityType": "EC2", "Unit": c.family, "Quantity": float64(n)})
		}
	}
	return out
}

func (st *opstStore) requirements(q *opstQuote) []any {
	req := func(typ string, pass bool, msg string) map[string]any {
		r := map[string]any{"OrderingRequirementType": typ, "Status": "PASS"}
		if !pass {
			r["Status"], r["StatusMessage"] = "FAIL", msg
		}
		return r
	}
	if q.outpostID == "" {
		return []any{req("OUTPOST_ID_MISSING_ON_QUOTE_ERROR", false, "Add an outpost to the quote before ordering.")}
	}
	o := st.outposts[q.outpostID]
	if o == nil {
		return []any{req("OUTPOST_NOT_FOUND_ERROR", false, "The quote's outpost no longer exists.")}
	}
	s := st.sites[o.siteID]
	op, ship := s.addresses["OPERATING_ADDRESS"], s.addresses["SHIPPING_ADDRESS"]
	return []any{
		req("OUTPOST_ID_MISSING_ON_QUOTE_ERROR", true, ""),
		req("OUTPOST_ACTIVE_CHECK_ERROR", o.lifecycle == "ACTIVE", "The outpost is not ACTIVE."),
		req("OPERATING_ADDRESS_EXISTENCE_CHECK_ERROR", op != nil, "The site has no operating address."),
		req("SHIPPING_ADDRESS_EXISTENCE_CHECK_ERROR", ship != nil, "The site has no shipping address."),
		req("COUNTRY_CODE_MISMATCH_CHECK_ERROR", op == nil || budStr(op, "CountryCode") == q.country,
			"The quote's country does not match the site's operating address."),
	}
}

func opstOr(list []string, def string) []string {
	if len(list) == 0 {
		return []string{def}
	}
	return list
}

func (st *opstStore) quoteOut(q *opstQuote) map[string]any {
	monthly := 0.0
	change := map[string]int{}
	specs := []any{}
	for _, l := range q.option.lines {
		c := opstCatalogItemByID(l.catalog)
		monthly += c.monthly * float64(l.quantity)
		change[c.family] += c.vcpus * l.quantity
		for i := 0; i < l.quantity; i++ {
			if c.class == "SERVER" {
				specs = append(specs, map[string]any{"QuoteSpecificationType": "SERVER", "ServerSpecificationDetails": map[string]any{
					"EC2Capacities": c.ec2Capacities(), "ServerPowerDrawKva": c.powerKva, "ServerWeightLbs": float64(c.weightLbs), "RackUnitHeight": "HEIGHT_2U"}})
				continue
			}
			specs = append(specs, map[string]any{"QuoteSpecificationType": "NEW_RACK", "FinalRackSpecificationDetails": map[string]any{
				"EC2Capacities": c.ec2Capacities(), "RackPowerDrawKva": c.powerKva, "RackWeightLbs": float64(c.weightLbs), "RackUnitHeight": "HEIGHT_42U"}})
		}
	}
	existing := st.existingVCPUs(q.outpostID)
	final := map[string]int{}
	for f, n := range existing {
		final[f] += n
	}
	for f, n := range change {
		final[f] += n
	}
	pricing := []any{}
	for _, term := range opstOr(q.terms, "THREE_YEARS") {
		for _, pay := range opstOr(q.payments, "ALL_UPFRONT") {
			pricing = append(pricing, map[string]any{"PricingType": "SUBSCRIPTION", "SubscriptionPricingDetails": opstPricing(monthly, pay, term)})
		}
	}
	out := map[string]any{
		"QuoteId": q.id, "AccountId": opstAccount, "QuoteStatus": q.status, "CountryCode": q.country,
		"RequestedCapacities": budClone(q.capacities), "RequestedConstraints": budClone(q.constraints),
		"RequestedPaymentOptions": opstAnyList(q.payments), "RequestedPaymentTerms": opstAnyList(q.terms),
		"QuoteOptions": []any{map[string]any{
			"QuoteOptionIdentifier": q.option.id, "Capacities": budClone(q.capacities), "Specifications": specs, "PricingOptions": pricing,
			"CapacitySummary": map[string]any{"ExistingCapacities": opstCapList(existing), "CapacityChange": opstCapList(change), "FinalCapacities": opstCapList(final)},
		}},
		"OrderingRequirements": st.requirements(q), "CreatedDate": budEpoch(q.created), "ExpirationDate": budEpoch(q.expires),
	}
	if o := st.outposts[q.outpostID]; o != nil {
		out["OutpostArn"] = o.arn
	}
	if q.description != "" {
		out["Description"] = q.description
	}
	if q.orderID != "" {
		out["SubmittedOrderId"] = q.orderID
	}
	return out
}

func opstAnyList(list []string) []any {
	out := make([]any, len(list))
	for i, v := range list {
		out[i] = v
	}
	return out
}

// applyQuoteInput copies the caller's request members onto q and resizes it.
func (st *opstStore) applyQuoteInput(q *opstQuote, in map[string]any) *budErr {
	if err := opstListMax(in, "RequestedCapacities", "RequestedConstraints", "RequestedPaymentOptions", "RequestedPaymentTerms"); err != nil {
		return err
	}
	next := *q
	if v, ok := in["OutpostIdentifier"].(string); ok {
		next.outpostID = ""
		if v != "" {
			o, err := st.outpost(v)
			if err != nil {
				return err
			}
			next.outpostID = o.id
		}
	}
	if v := budStr(in, "CountryCode"); v != "" {
		next.country = v
	}
	if v, ok := in["Description"].(string); ok {
		next.description = v
	}
	if v, ok := in["RequestedCapacities"].([]any); ok {
		next.capacities = budClone(v).([]any)
	}
	if v, ok := in["RequestedConstraints"].([]any); ok {
		next.constraints = budClone(v).([]any)
	}
	if _, ok := in["RequestedPaymentOptions"]; ok {
		next.payments = budStrings(in, "RequestedPaymentOptions")
	}
	if _, ok := in["RequestedPaymentTerms"]; ok {
		next.terms = budStrings(in, "RequestedPaymentTerms")
	}
	opt, err := st.quoteOption(&next)
	if err != nil {
		return err
	}
	next.option = opt
	*q = next
	return nil
}

func (st *opstStore) createQuote(in map[string]any, now time.Time) (map[string]any, *budErr) {
	if len(st.quotes) >= opstMaxQuotes {
		return nil, opstErrorf("ValidationException", "you can have at most %d quotes; delete some first", opstMaxQuotes)
	}
	id := "oq-" + opstHex(17)
	q := &opstQuote{id: id, status: "CREATED", created: now, expires: now.Add(opstQuoteLifetime)}
	if err := st.applyQuoteInput(q, in); err != nil {
		return nil, err
	}
	st.quotes[id] = q
	st.quoteOrder = append(st.quoteOrder, id)
	return map[string]any{"Quote": st.quoteOut(q)}, nil
}

func (st *opstStore) getQuote(in map[string]any, now time.Time) (map[string]any, *budErr) {
	q, err := st.quote(budStr(in, "QuoteIdentifier"), now)
	if err != nil {
		return nil, err
	}
	return map[string]any{"Quote": st.quoteOut(q)}, nil
}

func (st *opstStore) listQuotes(in map[string]any, now time.Time) (map[string]any, *budErr) {
	start, end, next, err := opstPageOf(in, len(st.quoteOrder))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, id := range st.quoteOrder[start:end] {
		q, _ := st.quote(id, now)
		out := st.quoteOut(q)
		delete(out, "OrderingRequirements")
		list = append(list, out)
	}
	return map[string]any{"Quotes": list, "NextToken": next}, nil
}

func (st *opstStore) updateQuote(in map[string]any, now time.Time) (map[string]any, *budErr) {
	q, err := st.quote(budStr(in, "QuoteIdentifier"), now)
	if err != nil {
		return nil, err
	}
	if q.status != "CREATED" {
		return nil, opstInvalid("quote %s is %s and can no longer change", q.id, q.status)
	}
	if err := st.applyQuoteInput(q, in); err != nil {
		return nil, err
	}
	return map[string]any{"Quote": st.quoteOut(q)}, nil
}

func (st *opstStore) deleteQuote(in map[string]any, now time.Time) (map[string]any, *budErr) {
	q, err := st.quote(budStr(in, "QuoteIdentifier"), now)
	if err != nil {
		return nil, err
	}
	if q.status == "ORDER_SUBMITTED" {
		return nil, opstInvalid("quote %s has a submitted order and can't be deleted", q.id)
	}
	delete(st.quotes, q.id)
	st.quoteOrder = opstRemove(st.quoteOrder, q.id)
	return map[string]any{}, nil
}

// ---- orders ----

func (st *opstStore) order(id string) (*opstOrder, *budErr) {
	if o := st.orders[id]; o != nil {
		return o, nil
	}
	return nil, opstNotFound("Order", id)
}

func (st *opstStore) orderedUnits() int {
	n := len(st.assets)
	for _, o := range st.orders {
		if !o.cancelled && o.status() != "FULFILLED" {
			for _, l := range o.lines {
				n += l.quantity
			}
		}
	}
	return n
}

func (st *opstStore) createOrder(in map[string]any, now time.Time) (map[string]any, *budErr) {
	out, err := st.outpost(budStr(in, "OutpostIdentifier"))
	if err != nil {
		return nil, err
	}
	if out.lifecycle != "ACTIVE" {
		return nil, opstConflict("outpost %s is %s", out.id, out.lifecycle)
	}
	if len(st.orders) >= opstMaxOrders && !st.evictCancelledOrder() {
		return nil, opstQuota("orders", opstMaxOrders)
	}
	ord := &opstOrder{id: "oo-" + opstHex(17), outpostID: out.id, payment: budStr(in, "PaymentOption"), term: budStr(in, "PaymentTerm"), submitted: now}
	if ord.term == "" {
		ord.term = "THREE_YEARS"
	}
	var quote *opstQuote
	if qid := budStr(in, "QuoteIdentifier"); qid != "" {
		if quote, err = st.quote(qid, now); err != nil {
			return nil, err
		}
		if quote.status != "CREATED" {
			return nil, opstConflict("quote %s is %s", quote.id, quote.status)
		}
		if quote.outpostID != out.id {
			return nil, opstInvalid("quote %s is not for outpost %s", quote.id, out.id)
		}
		if budStr(in, "QuoteOptionIdentifier") != quote.option.id {
			return nil, opstInvalid("QuoteOptionIdentifier must name an option of quote %s", quote.id)
		}
		for _, r := range st.requirements(quote) {
			if m := r.(map[string]any); m["Status"] == "FAIL" {
				return nil, opstInvalid("ordering requirement %s failed: %s", m["OrderingRequirementType"], m["StatusMessage"])
			}
		}
		if !opstMatches(opstOr(quote.payments, "ALL_UPFRONT"), ord.payment) || !opstMatches(opstOr(quote.terms, "THREE_YEARS"), ord.term) {
			return nil, opstInvalid("the payment option and term must be among those the quote priced")
		}
		ord.quoteID, ord.quoteOption = quote.id, quote.option.id
		for _, l := range quote.option.lines {
			if c := opstCatalogItemByID(l.catalog); c.class != out.hardware {
				return nil, opstInvalid("quote %s sizes %s hardware; outpost %s takes %s", quote.id, c.class, out.id, out.hardware)
			}
			ord.lines = append(ord.lines, opstLine{catalog: l.catalog, quantity: l.quantity})
		}
	} else {
		items := budList(in, "LineItems")
		if len(items) == 0 || len(items) > opstMaxListItems {
			return nil, opstInvalid("an order without a quote needs 1 to %d LineItems", opstMaxListItems)
		}
		for _, it := range items {
			m := it.(map[string]any)
			c := opstCatalogItemByID(budStr(m, "CatalogItemId"))
			if c == nil {
				return nil, opstNotFound("Catalog item", budStr(m, "CatalogItemId"))
			}
			if c.class != out.hardware {
				return nil, opstInvalid("catalog item %s is a %s; outpost %s takes %s hardware", c.id, c.class, out.id, out.hardware)
			}
			n := budInt(m, "Quantity", 1)
			if n > opstMaxQuantity {
				return nil, opstInvalid("a line item can order at most %d units", opstMaxQuantity)
			}
			ord.lines = append(ord.lines, opstLine{catalog: c.id, quantity: n})
		}
	}
	units := 0
	for i := range ord.lines {
		ord.lines[i].id = "ooi-" + opstHex(17)
		units += ord.lines[i].quantity
	}
	if st.orderedUnits()+units > opstMaxAssets {
		return nil, opstQuota("installed or ordered hardware units", opstMaxAssets)
	}
	ord.subscriptionID = "sub-" + opstHex(17)
	ord.tracking = strings.ToUpper(opstHex(12))
	if quote != nil {
		quote.status, quote.orderID = "ORDER_SUBMITTED", ord.id
	}
	st.orders[ord.id] = ord
	st.orderOrder = append(st.orderOrder, ord.id)
	return map[string]any{"Order": st.orderOut(ord)}, nil
}

// evictCancelledOrder drops the oldest cancelled order to make room.
func (st *opstStore) evictCancelledOrder() bool {
	for _, id := range st.orderOrder {
		if st.orders[id].cancelled {
			delete(st.orders, id)
			st.orderOrder = opstRemove(st.orderOrder, id)
			return true
		}
	}
	return false
}

func (st *opstStore) orderOut(o *opstOrder) map[string]any {
	lines := make([]any, 0, len(o.lines))
	for _, l := range o.lines {
		status := "CANCELLED"
		if !o.cancelled {
			status = opstLineSteps[o.step]
		}
		li := map[string]any{"CatalogItemId": l.catalog, "LineItemId": l.id, "Quantity": l.quantity, "Status": status}
		if !o.cancelled && o.step >= 2 {
			li["ShipmentInformation"] = map[string]any{"ShipmentTrackingNumber": o.tracking, "ShipmentCarrier": "UPS"}
		}
		if o.status() == "FULFILLED" {
			assets := []any{}
			for _, aid := range st.assetOrder {
				if a := st.assets[aid]; a.lineID == l.id {
					assets = append(assets, map[string]any{"AssetId": a.id, "MacAddressList": opstAnyList(a.macs)})
				}
			}
			li["AssetInformationList"] = assets
		}
		lines = append(lines, li)
	}
	out := map[string]any{
		"OrderId": o.id, "OutpostId": o.outpostID, "Status": o.status(), "LineItems": lines, "PaymentOption": o.payment,
		"PaymentTerm": o.term, "OrderSubmissionDate": budEpoch(o.submitted), "OrderType": "OUTPOST",
	}
	if o.quoteID != "" {
		out["QuoteIdentifier"], out["QuoteOptionIdentifier"] = o.quoteID, o.quoteOption
	}
	if !o.fulfilled.IsZero() {
		out["OrderFulfilledDate"] = budEpoch(o.fulfilled)
	}
	return out
}

// advance moves an order one step; fulfilment installs its assets.
func (st *opstStore) advance(o *opstOrder, now time.Time) {
	if o.cancelled || o.step == len(opstOrderSteps)-1 {
		return
	}
	o.step++
	if o.status() == "FULFILLED" {
		o.fulfilled = now
		st.installAssets(o)
	}
}

func (st *opstStore) getOrder(in map[string]any, now time.Time) (map[string]any, *budErr) {
	o, err := st.order(budStr(in, "OrderId"))
	if err != nil {
		return nil, err
	}
	st.advance(o, now)
	return map[string]any{"Order": st.orderOut(o)}, nil
}

func (st *opstStore) listOrders(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	filter := ""
	if f := budStr(in, "OutpostIdentifierFilter"); f != "" {
		filter = opstStripARN(f)
	}
	var match []*opstOrder
	for _, id := range st.orderOrder {
		if o := st.orders[id]; filter == "" || o.outpostID == filter {
			match = append(match, o)
		}
	}
	start, end, next, err := opstPageOf(in, len(match))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, o := range match[start:end] {
		lineStatus := "CANCELLED"
		if !o.cancelled {
			lineStatus = opstLineSteps[o.step]
		}
		sum := map[string]any{"OrderId": o.id, "OutpostId": o.outpostID, "OrderType": "OUTPOST", "Status": o.status(),
			"LineItemCountsByStatus": map[string]any{lineStatus: len(o.lines)}, "OrderSubmissionDate": budEpoch(o.submitted)}
		if !o.fulfilled.IsZero() {
			sum["OrderFulfilledDate"] = budEpoch(o.fulfilled)
		}
		list = append(list, sum)
	}
	return map[string]any{"Orders": list, "NextToken": next}, nil
}

func (st *opstStore) cancelOrder(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	o, err := st.order(budStr(in, "OrderId"))
	if err != nil {
		return nil, err
	}
	if s := o.status(); s != "RECEIVED" && s != "PENDING" {
		return nil, opstConflict("order %s is %s and can no longer be cancelled", o.id, s)
	}
	o.cancelled = true
	if q := st.quotes[o.quoteID]; q != nil && q.orderID == o.id {
		q.status, q.orderID = "CREATED", ""
	}
	return map[string]any{}, nil
}

// ---- billing ----

func (st *opstStore) subscriptions(outpostID string) []opstSubscription {
	var subs []opstSubscription
	for _, id := range st.orderOrder {
		o := st.orders[id]
		if o.outpostID != outpostID || o.cancelled {
			continue
		}
		sub := opstSubscription{id: o.subscriptionID, typ: "ORIGINAL", payment: o.payment, term: o.term, orderIDs: []string{o.id}, monthlyList: o.monthlyList(), begin: o.fulfilled}
		subs = append(subs, sub)
	}
	return append(subs, st.renewals[outpostID]...)
}

func (s opstSubscription) end() time.Time { return s.begin.AddDate(0, opstTermMonths[s.term], 0) }

// contractEnd is when the latest active or renewed subscription ends.
func (st *opstStore) contractEnd(outpostID string) (time.Time, bool) {
	var end time.Time
	for _, s := range st.subscriptions(outpostID) {
		if !s.begin.IsZero() && s.end().After(end) {
			end = s.end()
		}
	}
	return end, !end.IsZero()
}

func (st *opstStore) billingInformation(in map[string]any, now time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostIdentifier"))
	if err != nil {
		return nil, err
	}
	subs := st.subscriptions(o.id)
	start, end, next, err := opstPageOf(in, len(subs))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, s := range subs[start:end] {
		status := "PENDING"
		if !s.begin.IsZero() && !now.Before(s.begin) {
			status = "ACTIVE"
			if !now.Before(s.end()) {
				status = "INACTIVE"
			}
		}
		up, mo := opstPrice(s.monthlyList, s.payment, s.term)
		item := map[string]any{"SubscriptionId": s.id, "SubscriptionType": s.typ, "SubscriptionStatus": status,
			"OrderIds": opstAnyList(s.orderIDs), "Currency": "USD", "UpfrontPrice": up, "MonthlyRecurringPrice": mo}
		if !s.begin.IsZero() {
			item["BeginDate"], item["EndDate"] = budEpoch(s.begin), budEpoch(s.end())
		}
		list = append(list, item)
	}
	out := map[string]any{"Subscriptions": list, "NextToken": next}
	if e, ok := st.contractEnd(o.id); ok {
		out["ContractEndDate"] = e.Format("2006-01-02")
	}
	if len(subs) > 0 {
		last := subs[len(subs)-1]
		out["PaymentTerm"], out["PaymentOption"] = last.term, last.payment
	}
	return out, nil
}

// renewable returns the monthly list price of the installed hardware, which
// is what a renewal prices; false when nothing is installed yet.
func (st *opstStore) renewable(outpostID string) (float64, bool) {
	total := 0.0
	for _, o := range st.orders {
		if o.outpostID == outpostID && o.status() == "FULFILLED" {
			total += o.monthlyList()
		}
	}
	return total, total > 0
}

func (st *opstStore) renewalPricing(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostIdentifier"))
	if err != nil {
		return nil, err
	}
	monthly, ok := st.renewable(o.id)
	if !ok {
		return map[string]any{"PricingResult": "UNABLE_TO_PRICE", "PricingOptions": []any{}}, nil
	}
	options := []any{}
	for _, term := range []string{"ONE_YEAR", "THREE_YEARS", "FIVE_YEARS"} {
		for _, pay := range []string{"ALL_UPFRONT", "PARTIAL_UPFRONT", "NO_UPFRONT"} {
			options = append(options, map[string]any{"PricingType": "SUBSCRIPTION", "SubscriptionPricingDetails": opstPricing(monthly, pay, term)})
		}
	}
	return map[string]any{"PricingResult": "PRICED", "PricingOptions": options}, nil
}

func (st *opstStore) createRenewal(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostIdentifier"))
	if err != nil {
		return nil, err
	}
	pay, term, token := budStr(in, "PaymentOption"), budStr(in, "PaymentTerm"), budStr(in, "ClientToken")
	result := func(s opstSubscription) map[string]any {
		up, mo := opstPrice(s.monthlyList, s.payment, s.term)
		return map[string]any{"OutpostId": o.id, "PaymentOption": s.payment, "PaymentTerm": s.term, "UpfrontPrice": up, "MonthlyRecurringPrice": mo, "Currency": "USD"}
	}
	for _, s := range st.renewals[o.id] {
		if token != "" && s.clientToken == token {
			return result(s), nil
		}
	}
	monthly, ok := st.renewable(o.id)
	if !ok {
		return nil, opstInvalid("outpost %s has no installed hardware to renew", o.id)
	}
	if len(st.renewals[o.id]) >= opstMaxRenewals {
		return nil, opstInvalid("outpost %s already has %d renewals", o.id, opstMaxRenewals)
	}
	begin, _ := st.contractEnd(o.id)
	var orderIDs []string
	for _, id := range st.orderOrder {
		if ord := st.orders[id]; ord.outpostID == o.id && ord.status() == "FULFILLED" {
			orderIDs = append(orderIDs, ord.id)
		}
	}
	s := opstSubscription{id: "sub-" + opstHex(17), typ: "RENEWAL", payment: pay, term: term, clientToken: token, orderIDs: orderIDs, monthlyList: monthly, begin: begin}
	st.renewals[o.id] = append(st.renewals[o.id], s)
	return result(s), nil
}
