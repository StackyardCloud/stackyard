package server

import (
	"strings"
	"testing"
)

const budAcct = `"AccountId":"123456789012"`

func budBody(fields ...string) string {
	return "{" + strings.Join(append([]string{budAcct}, fields...), ",") + "}"
}

const (
	budNote80  = `{"NotificationType":"ACTUAL","ComparisonOperator":"GREATER_THAN","Threshold":80}`
	budEmail   = `{"SubscriptionType":"EMAIL","Address":"finance@example.com"}`
	budEmail2  = `{"SubscriptionType":"EMAIL","Address":"ops@example.com"}`
	budIAMBody = `"NotificationType":"ACTUAL","ActionType":"APPLY_IAM_POLICY",` +
		`"ActionThreshold":{"ActionThresholdValue":90,"ActionThresholdType":"PERCENTAGE"},` +
		`"Definition":{"IamActionDefinition":{"PolicyArn":"arn:aws:iam::aws:policy/AWSDenyAll","Roles":["dev"]}},` +
		`"ExecutionRoleArn":"arn:aws:iam::123456789012:role/budgets","ApprovalModel":"MANUAL","Subscribers":[` + budEmail + `]`
)

func budNewBudget(name string) string {
	return budBody(`"Budget":{"BudgetName":"`+name+`","BudgetLimit":{"Amount":"100","Unit":"USD"},"TimeUnit":"MONTHLY","BudgetType":"COST"}`,
		`"NotificationsWithSubscribers":[{"Notification":`+budNote80+`,"Subscribers":[`+budEmail+`]}]`,
		`"ResourceTags":[{"Key":"team","Value":"finance"}]`)
}

func TestBudgetsStage1BudgetLifecycle(t *testing.T) {
	_, ts := newBudTestServer(t)
	budOK(t, ts, "CreateBudget", budNewBudget("monthly"))
	if code := budErrCode(t, ts, "CreateBudget", budNewBudget("monthly")); code != "DuplicateRecordException" {
		t.Fatalf("duplicate: %s", code)
	}
	b := budOK(t, ts, "DescribeBudget", budBody(`"BudgetName":"monthly"`))["Budget"].(map[string]any)
	if b["CostTypes"].(map[string]any)["IncludeTax"] != true || b["CalculatedSpend"] == nil {
		t.Fatalf("defaults: %v", b)
	}
	if end := b["TimePeriod"].(map[string]any)["End"].(float64); end != float64(budDefaultEnd.Unix()) {
		t.Fatalf("default end: %v", end)
	}

	budOK(t, ts, "UpdateBudget", budBody(`"NewBudget":{"BudgetName":"monthly","BudgetLimit":{"Amount":"150","Unit":"USD"},"TimeUnit":"MONTHLY","BudgetType":"COST"}`))
	b = budOK(t, ts, "DescribeBudget", budBody(`"BudgetName":"monthly"`))["Budget"].(map[string]any)
	if b["BudgetLimit"].(map[string]any)["Amount"] != "150" {
		t.Fatalf("update: %v", b)
	}
	if code := budErrCode(t, ts, "UpdateBudget", budBody(`"NewBudget":{"BudgetName":"monthly","BudgetLimit":{"Amount":"1","Unit":"USD"},"TimeUnit":"MONTHLY","BudgetType":"USAGE"}`)); code != "InvalidParameterException" {
		t.Fatalf("type change: %s", code)
	}

	hist := budOK(t, ts, "DescribeBudgetPerformanceHistory", budBody(`"BudgetName":"monthly"`, `"TimePeriod":{"Start":1767225600,"End":1775001600}`))
	if periods := hist["BudgetPerformanceHistory"].(map[string]any)["BudgetedAndActualAmountsList"].([]any); len(periods) != 3 {
		t.Fatalf("Jan-Mar 2026 monthly periods: %d", len(periods))
	}

	for i := range 3 {
		budOK(t, ts, "CreateBudget", budNewBudget("page-"+string(rune('a'+i))))
	}
	page := budOK(t, ts, "DescribeBudgets", budBody(`"MaxResults":2`))
	if len(page["Budgets"].([]any)) != 2 || page["NextToken"] == "" {
		t.Fatalf("page 1: %v", page)
	}
	page = budOK(t, ts, "DescribeBudgets", budBody(`"MaxResults":2`, `"NextToken":"`+page["NextToken"].(string)+`"`))
	if len(page["Budgets"].([]any)) != 2 || page["NextToken"] != nil {
		t.Fatalf("page 2: %v", page)
	}
	if code := budErrCode(t, ts, "DescribeBudgets", budBody(`"NextToken":"bogus"`)); code != "InvalidNextTokenException" {
		t.Fatalf("bad token: %s", code)
	}

	budOK(t, ts, "DeleteBudget", budBody(`"BudgetName":"monthly"`))
	if code := budErrCode(t, ts, "DescribeBudget", budBody(`"BudgetName":"monthly"`)); code != "NotFoundException" {
		t.Fatalf("deleted: %s", code)
	}
}

func TestBudgetsStage1Validation(t *testing.T) {
	_, ts := newBudTestServer(t)
	cases := map[string]string{
		"no limit":         `"Budget":{"BudgetName":"x","TimeUnit":"MONTHLY","BudgetType":"COST"}`,
		"colon in name":    `"Budget":{"BudgetName":"a:b","BudgetLimit":{"Amount":"1","Unit":"USD"},"TimeUnit":"MONTHLY","BudgetType":"COST"}`,
		"planned on daily": `"Budget":{"BudgetName":"x","PlannedBudgetLimits":{"1767225600":{"Amount":"1","Unit":"USD"}},"TimeUnit":"DAILY","BudgetType":"COST"}`,
		"custom no end":    `"Budget":{"BudgetName":"x","BudgetLimit":{"Amount":"1","Unit":"USD"},"TimeUnit":"CUSTOM","BudgetType":"COST"}`,
		"start after end":  `"Budget":{"BudgetName":"x","BudgetLimit":{"Amount":"1","Unit":"USD"},"TimeUnit":"MONTHLY","BudgetType":"COST","TimePeriod":{"Start":1775001600,"End":1767225600}}`,
		"bad amount":       `"Budget":{"BudgetName":"x","BudgetLimit":{"Amount":"ten","Unit":"USD"},"TimeUnit":"MONTHLY","BudgetType":"COST"}`,
		"bad email": `"Budget":{"BudgetName":"x","BudgetLimit":{"Amount":"1","Unit":"USD"},"TimeUnit":"MONTHLY","BudgetType":"COST"},` +
			`"NotificationsWithSubscribers":[{"Notification":` + budNote80 + `,"Subscribers":[{"SubscriptionType":"EMAIL","Address":"nope"}]}]`,
		"reserved tag": `"Budget":{"BudgetName":"x","BudgetLimit":{"Amount":"1","Unit":"USD"},"TimeUnit":"MONTHLY","BudgetType":"COST"},"ResourceTags":[{"Key":"aws:x","Value":"y"}]`,
	}
	for name, body := range cases {
		if code := budErrCode(t, ts, "CreateBudget", budBody(body)); code != "InvalidParameterException" {
			t.Errorf("%s: %s", name, code)
		}
	}
	if code := budErrCode(t, ts, "DescribeBudget", `{"AccountId":"12","BudgetName":"x"}`); code != "InvalidParameterException" {
		t.Errorf("short account: %s", code)
	}
}

func TestBudgetsStage1NotificationsAndSubscribers(t *testing.T) {
	_, ts := newBudTestServer(t)
	budOK(t, ts, "CreateBudget", budNewBudget("b"))
	on := `"BudgetName":"b"`
	note95 := `{"NotificationType":"ACTUAL","ComparisonOperator":"GREATER_THAN","Threshold":95}`
	budOK(t, ts, "CreateNotification", budBody(on, `"Notification":`+note95, `"Subscribers":[`+budEmail+`]`))
	if code := budErrCode(t, ts, "CreateNotification", budBody(on, `"Notification":`+note95, `"Subscribers":[`+budEmail+`]`)); code != "DuplicateRecordException" {
		t.Fatalf("duplicate notification: %s", code)
	}
	notes := budOK(t, ts, "DescribeNotificationsForBudget", budBody(on))["Notifications"].([]any)
	if len(notes) != 2 || notes[0].(map[string]any)["ThresholdType"] != "PERCENTAGE" {
		t.Fatalf("notifications: %v", notes)
	}
	acct := budOK(t, ts, "DescribeBudgetNotificationsForAccount", budBody())["BudgetNotificationsForAccount"].([]any)
	if len(acct) != 1 {
		t.Fatalf("for account: %v", acct)
	}

	note99 := `{"NotificationType":"FORECASTED","ComparisonOperator":"GREATER_THAN","Threshold":99}`
	budOK(t, ts, "UpdateNotification", budBody(on, `"OldNotification":`+note95, `"NewNotification":`+note99))
	if code := budErrCode(t, ts, "UpdateNotification", budBody(on, `"OldNotification":`+note99, `"NewNotification":`+budNote80)); code != "DuplicateRecordException" {
		t.Fatalf("update onto existing: %s", code)
	}

	n := `"Notification":` + budNote80
	budOK(t, ts, "CreateSubscriber", budBody(on, n, `"Subscriber":`+budEmail2))
	if code := budErrCode(t, ts, "CreateSubscriber", budBody(on, n, `"Subscriber":`+budEmail2)); code != "DuplicateRecordException" {
		t.Fatalf("duplicate subscriber: %s", code)
	}
	sns := `{"SubscriptionType":"SNS","Address":"arn:aws:sns:us-east-1:123456789012:alerts"}`
	budOK(t, ts, "UpdateSubscriber", budBody(on, n, `"OldSubscriber":`+budEmail2, `"NewSubscriber":`+sns))
	subs := budOK(t, ts, "DescribeSubscribersForNotification", budBody(on, n))["Subscribers"].([]any)
	if len(subs) != 2 || subs[1].(map[string]any)["SubscriptionType"] != "SNS" {
		t.Fatalf("subscribers: %v", subs)
	}
	sns2 := `{"SubscriptionType":"SNS","Address":"arn:aws:sns:us-east-1:123456789012:other"}`
	if code := budErrCode(t, ts, "CreateSubscriber", budBody(on, n, `"Subscriber":`+sns2)); code != "CreationLimitExceededException" {
		t.Fatalf("second SNS subscriber: %s", code)
	}
	budOK(t, ts, "DeleteSubscriber", budBody(on, n, `"Subscriber":`+sns))
	if code := budErrCode(t, ts, "DeleteSubscriber", budBody(on, n, `"Subscriber":`+budEmail)); code != "InvalidParameterException" {
		t.Fatalf("deleting the last subscriber: %s", code)
	}
	budOK(t, ts, "DeleteNotification", budBody(on, n))
	if code := budErrCode(t, ts, "DescribeSubscribersForNotification", budBody(on, n)); code != "NotFoundException" {
		t.Fatalf("deleted notification: %s", code)
	}
}

func TestBudgetsStage1Actions(t *testing.T) {
	_, ts := newBudTestServer(t)
	budOK(t, ts, "CreateBudget", budNewBudget("b"))
	on := `"BudgetName":"b"`
	id := budOK(t, ts, "CreateBudgetAction", budBody(on, budIAMBody))["ActionId"].(string)
	if len(id) != 36 {
		t.Fatalf("action id %q", id)
	}
	bad := strings.Replace(budIAMBody, `"Roles":["dev"]`, `"Roles":[]`, 1)
	if code := budErrCode(t, ts, "CreateBudgetAction", budBody(on, bad)); code != "InvalidParameterException" {
		t.Fatalf("IAM action without targets: %s", code)
	}
	wrong := strings.Replace(budIAMBody, "APPLY_IAM_POLICY", "RUN_SSM_DOCUMENTS", 1)
	if code := budErrCode(t, ts, "CreateBudgetAction", budBody(on, wrong)); code != "InvalidParameterException" {
		t.Fatalf("definition for another type: %s", code)
	}

	act := `"ActionId":"` + id + `"`
	a := budOK(t, ts, "DescribeBudgetAction", budBody(on, act))["Action"].(map[string]any)
	if a["Status"] != "STANDBY" {
		t.Fatalf("new action: %v", a)
	}
	up := budOK(t, ts, "UpdateBudgetAction", budBody(on, act, `"ApprovalModel":"AUTOMATIC"`))
	if up["OldAction"].(map[string]any)["ApprovalModel"] != "MANUAL" || up["NewAction"].(map[string]any)["ApprovalModel"] != "AUTOMATIC" {
		t.Fatalf("update: %v", up)
	}
	if code := budErrCode(t, ts, "ExecuteBudgetAction", budBody(on, act, `"ExecutionType":"REVERSE_BUDGET_ACTION"`)); code != "ResourceLockedException" {
		t.Fatalf("reverse from standby: %s", code)
	}
	for _, step := range []struct{ exec, want string }{
		{"APPROVE_BUDGET_ACTION", "EXECUTION_SUCCESS"}, {"REVERSE_BUDGET_ACTION", "REVERSE_SUCCESS"}, {"RESET_BUDGET_ACTION", "STANDBY"},
	} {
		budOK(t, ts, "ExecuteBudgetAction", budBody(on, act, `"ExecutionType":"`+step.exec+`"`))
		if got := budOK(t, ts, "DescribeBudgetAction", budBody(on, act))["Action"].(map[string]any)["Status"]; got != step.want {
			t.Fatalf("%s: %v", step.exec, got)
		}
	}
	hist := budOK(t, ts, "DescribeBudgetActionHistories", budBody(on, act))["ActionHistories"].([]any)
	if len(hist) != 5 || hist[0].(map[string]any)["EventType"] != "CREATE_ACTION" {
		t.Fatalf("history: %v", hist)
	}
	if n := len(budOK(t, ts, "DescribeBudgetActionsForAccount", budBody())["Actions"].([]any)); n != 1 {
		t.Fatalf("actions for account: %d", n)
	}

	arn := `"ResourceARN":"arn:aws:budgets::123456789012:budget/b/action/` + id + `"`
	budOK(t, ts, "TagResource", `{`+arn+`,"ResourceTags":[{"Key":"env","Value":"prod"}]}`)
	if tags := budOK(t, ts, "ListTagsForResource", `{`+arn+`}`)["ResourceTags"].([]any); len(tags) != 1 {
		t.Fatalf("action tags: %v", tags)
	}
	budOK(t, ts, "DeleteBudgetAction", budBody(on, act))
	if code := budErrCode(t, ts, "DescribeBudgetAction", budBody(on, act)); code != "NotFoundException" {
		t.Fatalf("deleted action: %s", code)
	}
	if code := budErrCode(t, ts, "ListTagsForResource", `{`+arn+`}`); code != "NotFoundException" {
		t.Fatalf("deleted action tags: %s", code)
	}
}

func TestBudgetsStage1Tags(t *testing.T) {
	_, ts := newBudTestServer(t)
	budOK(t, ts, "CreateBudget", budNewBudget("b"))
	arn := `"ResourceARN":"arn:aws:budgets::123456789012:budget/b"`
	budOK(t, ts, "TagResource", `{`+arn+`,"ResourceTags":[{"Key":"env","Value":"prod"}]}`)
	tags := budOK(t, ts, "ListTagsForResource", `{`+arn+`}`)["ResourceTags"].([]any)
	if len(tags) != 2 || tags[0].(map[string]any)["Key"] != "env" {
		t.Fatalf("tags: %v", tags)
	}
	budOK(t, ts, "UntagResource", `{`+arn+`,"ResourceTagKeys":["env","team"]}`)
	if tags := budOK(t, ts, "ListTagsForResource", `{`+arn+`}`)["ResourceTags"].([]any); len(tags) != 0 {
		t.Fatalf("after untag: %v", tags)
	}
	if code := budErrCode(t, ts, "ListTagsForResource", `{"ResourceARN":"arn:aws:budgets::123456789012:budget/missing"}`); code != "NotFoundException" {
		t.Fatalf("missing budget: %s", code)
	}
	if code := budErrCode(t, ts, "ListTagsForResource", `{"ResourceARN":"arn:aws:s3:::bucket"}`); code != "InvalidParameterException" {
		t.Fatalf("foreign ARN: %s", code)
	}
}
