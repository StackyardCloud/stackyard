package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Free Tier: awsJson1_0, X-Amz-Target AWSFreeTierService.*, signed
// freetier. The caller's account starts on the FREE plan with USD 100 of
// credits; sign-up activities add USD 20 each when completed, and the AWS
// Budgets one completes once the account has a budget in Stackyard's Budgets.

const (
	ftierSignupCredit = 100.0
	ftierMaxCredit    = 200.0
	ftierMaxDepth     = 10  // filter expression nesting
	ftierMaxValues    = 100 // Values per Dimensions node
)

var ftierMatchers = map[string]func(string, string) bool{
	"EQUALS": func(a, b string) bool { return a == b }, "STARTS_WITH": strings.HasPrefix,
	"ENDS_WITH": strings.HasSuffix, "CONTAINS": strings.Contains,
}

var ftierAPI = &budJSONAPI{
	prefix: "AWSFreeTierService.", signing: "freetier", contentType: "application/x-amz-json-1.0",
	invalidCode: "ValidationException", model: awsmodels.FreeTierModel, respondError: ftierRespondError,
}

// ftierRespondError writes an awsJson1_0 error in the Free Tier model's
// shape (__type and a lower-case message).
func ftierRespondError(w http.ResponseWriter, e *budErr) {
	raw, _ := json.Marshal(map[string]string{"__type": e.code, "message": e.msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}

var ftierOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range ftierOperations {
		out[op.Name] = true
	}
	return out
}()

func init() {
	ftierAPI.handlers = map[string]budHandler{
		"GetAccountPlanState":   (*Server).ftierGetAccountPlanState,
		"UpgradeAccountPlan":    (*Server).ftierUpgradeAccountPlan,
		"ListAccountActivities": (*Server).ftierListAccountActivities,
		"GetAccountActivity":    (*Server).ftierGetAccountActivity,
		"GetFreeTierUsage":      (*Server).ftierGetFreeTierUsage,
	}
}

func (s *Server) handleFreeTierJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.budServe(w, r, ftierAPI, ftierOperationByName)
}

type ftierActivity struct {
	id, title, description, url string
	minutes                     int
	budgets                     bool // completes when the account has a budget
}

func ftierActivityID(title string) string {
	sum := sha256.Sum256([]byte(title))
	return hex.EncodeToString(sum[:16])
}

var ftierActivities = func() []ftierActivity {
	out := []ftierActivity{
		{title: "Launch an instance using EC2", description: "Launch an Amazon EC2 instance and stop or terminate it when you're done.", url: "https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EC2_GetStarted.html", minutes: 10},
		{title: "Set up a cost budget using AWS Budgets", description: "Create a cost budget so you're alerted before you spend more than planned.", url: "https://docs.aws.amazon.com/cost-management/latest/userguide/budgets-create.html", minutes: 5, budgets: true},
		{title: "Use a foundation model in the Amazon Bedrock playground", description: "Send a prompt to a foundation model in the Amazon Bedrock playground.", url: "https://docs.aws.amazon.com/bedrock/latest/userguide/playgrounds.html", minutes: 10},
		{title: "Create a web app using AWS Lambda", description: "Create a Lambda function with a function URL that serves a web page.", url: "https://docs.aws.amazon.com/lambda/latest/dg/getting-started.html", minutes: 15},
		{title: "Create an Aurora or RDS database", description: "Create an Amazon Aurora or Amazon RDS database.", url: "https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_GettingStarted.html", minutes: 15},
	}
	for i := range out {
		out[i].id = ftierActivityID(out[i].title)
	}
	return out
}()

// ftierStore is the caller's plan: a fixed-size record.
type ftierStore struct {
	mu        sync.Mutex
	started   time.Time // first Free Tier call
	plan      string
	completed map[string]time.Time // activity id -> completion
}

func newFreeTierStore() *ftierStore {
	return &ftierStore{plan: "FREE", completed: map[string]time.Time{}}
}

// hasBudgets reports whether account has any budget. It takes the Budgets
// lock; Free Tier calls it before taking its own.
func (st *budStore) hasBudgets(account string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, b := range st.budgets {
		if b.account == account {
			return true
		}
	}
	return false
}

// ftierDo refreshes activity completion from Budgets, then runs fn under the
// Free Tier lock.
func (s *Server) ftierDo(fn func(st *ftierStore, now time.Time) (map[string]any, *budErr)) (map[string]any, *budErr) {
	budgeted := s.budgets.hasBudgets(acctCaller)
	st := s.freeTier
	st.mu.Lock()
	defer st.mu.Unlock()
	now := nowUTC()
	if st.started.IsZero() {
		st.started = now
	}
	for _, a := range ftierActivities {
		if a.budgets && budgeted {
			if _, done := st.completed[a.id]; !done {
				st.completed[a.id] = now
			}
		}
	}
	return fn(st, now)
}

func (st *ftierStore) credits() float64 {
	return min(ftierMaxCredit, ftierSignupCredit+20*float64(len(st.completed)))
}

func (st *ftierStore) expires() time.Time { return st.started.AddDate(0, 6, 0) }

func (st *ftierStore) planStatus(now time.Time) string {
	if st.plan == "FREE" && now.After(st.expires()) {
		return "EXPIRED"
	}
	return "ACTIVE"
}

func (s *Server) ftierGetAccountPlanState(map[string]any) (map[string]any, *budErr) {
	return s.ftierDo(func(st *ftierStore, now time.Time) (map[string]any, *budErr) {
		return map[string]any{
			"accountId": acctCaller, "accountPlanType": st.plan, "accountPlanStatus": st.planStatus(now),
			"accountPlanRemainingCredits": map[string]any{"amount": st.credits(), "unit": "USD"},
			"accountPlanExpirationDate":   budEpoch(st.expires()),
		}, nil
	})
}

// ftierUpgradeAccountPlan moves the account to PAID. There is no way back to
// FREE; upgrading a PAID account again is a no-op.
func (s *Server) ftierUpgradeAccountPlan(in map[string]any) (map[string]any, *budErr) {
	return s.ftierDo(func(st *ftierStore, now time.Time) (map[string]any, *budErr) {
		if budStr(in, "accountPlanType") != "PAID" {
			return nil, budErrorf("ValidationException", "An account can only be upgraded to the PAID plan.")
		}
		st.plan = "PAID"
		return map[string]any{"accountId": acctCaller, "accountPlanType": st.plan, "accountPlanStatus": st.planStatus(now)}, nil
	})
}

func (st *ftierStore) activity(a ftierActivity, full bool) map[string]any {
	out := map[string]any{
		"activityId": a.id, "title": a.title, "status": "NOT_STARTED",
		"reward": map[string]any{"credit": map[string]any{"amount": 20.0, "unit": "USD"}},
	}
	if done, ok := st.completed[a.id]; ok {
		out["status"] = "COMPLETED"
		if full {
			out["startedAt"], out["completedAt"] = budEpoch(done), budEpoch(done)
		}
	}
	if full {
		out["description"], out["instructionsUrl"], out["estimatedTimeToCompleteInMinutes"] = a.description, a.url, a.minutes
		out["expiresAt"] = budEpoch(st.expires())
	}
	return out
}

func (s *Server) ftierListAccountActivities(in map[string]any) (map[string]any, *budErr) {
	want := budStrings(in, "filterActivityStatuses")
	return s.ftierDo(func(st *ftierStore, _ time.Time) (map[string]any, *budErr) {
		var list []any
		for _, a := range ftierActivities {
			item := st.activity(a, false)
			if len(want) == 0 || budContains(want, item["status"].(string)) {
				list = append(list, item)
			}
		}
		start, end, next, err := budPage(budStr(in, "nextToken"), len(list), budInt(in, "maxResults", 100), "ValidationException")
		if err != nil {
			return nil, err
		}
		return map[string]any{"activities": append([]any{}, list[start:end]...), "nextToken": next}, nil
	})
}

func (s *Server) ftierGetAccountActivity(in map[string]any) (map[string]any, *budErr) {
	id := budStr(in, "activityId")
	return s.ftierDo(func(st *ftierStore, _ time.Time) (map[string]any, *budErr) {
		for _, a := range ftierActivities {
			if a.id == id {
				return st.activity(a, true), nil
			}
		}
		return nil, budErrorf("ResourceNotFoundException", "Activity %s was not found.", id)
	})
}

// ---- free tier usage ----

type ftierUsage struct {
	service, operation, usageType, region, unit, description, freeTierType string
	actual, forecast, limit                                                float64
}

// ftierUsages is the fixed usage catalog: Stackyard meters no usage.
var ftierUsages = []ftierUsage{
	{"Amazon Elastic Compute Cloud", "RunInstances", "USE1-BoxUsage:t3.micro", "us-east-1", "Hrs", "750 hours of t2.micro, t3.micro or t4g.small instances", "12 Months Free", 120, 744, 750},
	{"Amazon Elastic Compute Cloud", "CreateVolume-Gp3", "USE1-EBS:VolumeUsage.gp3", "us-east-1", "GB-Mo", "30 GB-month of EBS storage", "12 Months Free", 8, 8, 30},
	{"Amazon Simple Storage Service", "PutObject", "USE1-TimedStorage-ByteHrs", "us-east-1", "GB-Mo", "5 GB-month of S3 Standard storage", "12 Months Free", 1.2, 1.5, 5},
	{"Amazon Simple Storage Service", "PutObject", "USE1-Requests-Tier1", "us-east-1", "Requests", "2,000 PUT, COPY, POST or LIST requests", "12 Months Free", 340, 1100, 2000},
	{"AWS Lambda", "Invoke", "Global-Request", "global", "Requests", "1,000,000 requests per month", "Always Free", 52000, 160000, 1000000},
	{"AWS Lambda", "Invoke", "Global-Lambda-GB-Second", "global", "Seconds", "400,000 GB-seconds of compute per month", "Always Free", 9100, 28000, 400000},
	{"Amazon DynamoDB", "PayPerRequestThroughput", "USE1-TimedStorage-ByteHrs", "us-east-1", "GB-Mo", "25 GB of DynamoDB storage", "Always Free", 0.4, 0.4, 25},
	{"Amazon Relational Database Service", "CreateDBInstance", "USE1-InstanceUsage:db.t3.micro", "us-east-1", "Hrs", "750 hours of db.t3.micro Single-AZ instances", "12 Months Free", 0, 0, 750},
	{"Amazon CloudWatch", "PutMetricData", "CW:MetricMonitorUsage", "global", "Metrics", "10 custom metrics", "Always Free", 3, 3, 10},
	{"Amazon Simple Notification Service", "Publish", "Requests-Tier1", "global", "Requests", "1,000,000 publishes", "Always Free", 120, 400, 1000000},
}

func (u ftierUsage) out() map[string]any {
	return map[string]any{
		"service": u.service, "operation": u.operation, "usageType": u.usageType, "region": u.region,
		"actualUsageAmount": u.actual, "forecastedUsageAmount": u.forecast, "limit": u.limit,
		"unit": u.unit, "description": u.description, "freeTierType": u.freeTierType,
	}
}

func (s *Server) ftierGetFreeTierUsage(in map[string]any) (map[string]any, *budErr) {
	filter := budMap(in, "filter")
	var list []any
	for _, u := range ftierUsages {
		ok := true
		if filter != nil {
			var err *budErr
			if ok, err = ftierMatch(filter, u, 0); err != nil {
				return nil, err
			}
		}
		if ok {
			list = append(list, u.out())
		}
	}
	start, end, next, err := budPage(budStr(in, "nextToken"), len(list), budInt(in, "maxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	return map[string]any{"freeTierUsages": append([]any{}, list[start:end]...), "nextToken": next}, nil
}

// ftierMatch evaluates a filter Expression, which must set exactly one of
// And, Or, Not and Dimensions.
func ftierMatch(expr map[string]any, u ftierUsage, depth int) (bool, *budErr) {
	if depth > ftierMaxDepth {
		return false, budErrorf("ValidationException", "filter expressions can nest at most %d levels", ftierMaxDepth)
	}
	set := 0
	for _, k := range []string{"And", "Or", "Not", "Dimensions"} {
		if expr[k] != nil {
			set++
		}
	}
	if set != 1 {
		return false, budErrorf("ValidationException", "each filter expression must set exactly one of And, Or, Not or Dimensions")
	}
	if d := budMap(expr, "Dimensions"); d != nil {
		return ftierDimension(d, u)
	}
	if n := budMap(expr, "Not"); n != nil {
		ok, err := ftierMatch(n, u, depth+1)
		return !ok, err
	}
	and := expr["And"] != nil
	list := budList(expr, "And")
	if !and {
		list = budList(expr, "Or")
	}
	for _, item := range list {
		sub, _ := item.(map[string]any)
		ok, err := ftierMatch(sub, u, depth+1)
		if err != nil {
			return false, err
		}
		if ok != and { // first false for And, first true for Or
			return ok, nil
		}
	}
	return and, nil
}

func ftierDimension(d map[string]any, u ftierUsage) (bool, *budErr) {
	key := budStr(d, "Key")
	values := budStrings(d, "Values")
	if len(values) > ftierMaxValues {
		return false, budErrorf("ValidationException", "a Dimensions filter can list at most %d values", ftierMaxValues)
	}
	// MatchOptions is an enum of five, so its distinct entries bound the work.
	opts := taxsUnique(budStrings(d, "MatchOptions"))
	if key == "USAGE_PERCENTAGE" {
		pct := 0.0
		if u.limit > 0 {
			pct = u.actual / u.limit * 100
		}
		for _, opt := range opts {
			if opt != "GREATER_THAN_OR_EQUAL" {
				return false, budErrorf("ValidationException", "USAGE_PERCENTAGE supports only GREATER_THAN_OR_EQUAL")
			}
		}
		for _, v := range values {
			n, err := strconv.ParseFloat(strings.TrimSuffix(v, "%"), 64)
			if err != nil {
				return false, budErrorf("ValidationException", "USAGE_PERCENTAGE values must be numbers")
			}
			if pct >= n {
				return true, nil
			}
		}
		return false, nil
	}
	field := map[string]string{
		"SERVICE": u.service, "OPERATION": u.operation, "USAGE_TYPE": u.usageType, "REGION": u.region,
		"FREE_TIER_TYPE": u.freeTierType, "DESCRIPTION": u.description,
	}[key]
	if len(opts) == 0 {
		opts = []string{"EQUALS"}
	}
	for _, opt := range opts {
		match := ftierMatchers[opt]
		if match == nil {
			return false, budErrorf("ValidationException", "%s supports EQUALS, STARTS_WITH, ENDS_WITH and CONTAINS", key)
		}
		for _, v := range values {
			if match(field, v) {
				return true, nil
			}
		}
	}
	return false, nil
}
