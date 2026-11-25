package server

import (
	"fmt"
	"time"
)

const budMaxHistory = 100 // per action; oldest entries drop first

type budAction struct {
	a       map[string]any // the Action shape
	created time.Time
	history []map[string]any
	tags    map[string]string
}

// budDefinitionFor names the Definition member each ActionType needs.
var budDefinitionFor = map[string]string{
	"APPLY_IAM_POLICY":  "IamActionDefinition",
	"APPLY_SCP_POLICY":  "ScpActionDefinition",
	"RUN_SSM_DOCUMENTS": "SsmActionDefinition",
}

// budTransitions maps an ExecutionType to the statuses it may start from and
// the status it ends in. Stackyard runs actions synchronously, so the
// *_IN_PROGRESS statuses are never observed.
// ponytail: actions record their effect but don't touch IAM, Organizations or SSM.
var budTransitions = map[string]struct {
	from []string
	to   string
}{
	"APPROVE_BUDGET_ACTION": {from: []string{"STANDBY", "PENDING"}, to: "EXECUTION_SUCCESS"},
	"RETRY_BUDGET_ACTION":   {from: []string{"EXECUTION_FAILURE", "REVERSE_FAILURE", "RESET_FAILURE"}, to: "EXECUTION_SUCCESS"},
	"REVERSE_BUDGET_ACTION": {from: []string{"EXECUTION_SUCCESS"}, to: "REVERSE_SUCCESS"},
	"RESET_BUDGET_ACTION":   {from: []string{"EXECUTION_SUCCESS", "REVERSE_SUCCESS", "EXECUTION_FAILURE", "REVERSE_FAILURE"}, to: "STANDBY"},
}

func budActionARN(account, budget, id string) string {
	return fmt.Sprintf("%s/action/%s", budARN(account, budget), id)
}

func (b *budBudget) action(id string) *budAction {
	for _, a := range b.actions {
		if budStr(a.a, "ActionId") == id {
			return a
		}
	}
	return nil
}

func (b *budBudget) actionOrErr(id string) (*budAction, *budErr) {
	if a := b.action(id); a != nil {
		return a, nil
	}
	return nil, budNotFound("Budget action %s doesn't exist on budget %s.", id, b.name)
}

func budCheckDefinition(actionType string, def map[string]any) *budErr {
	want := budDefinitionFor[actionType]
	for member := range def {
		if member != want {
			return budInvalid("%s actions take only %s.", actionType, want)
		}
	}
	d := budMap(def, want)
	if d == nil {
		return budInvalid("%s actions need Definition.%s.", actionType, want)
	}
	if want == "IamActionDefinition" && len(budList(d, "Roles"))+len(budList(d, "Groups"))+len(budList(d, "Users")) == 0 {
		return budInvalid("IamActionDefinition needs at least one of Roles, Groups or Users.")
	}
	return nil
}

func budCheckThreshold(th map[string]any) *budErr {
	v, _ := th["ActionThresholdValue"].(float64)
	if v < 0 || (budStr(th, "ActionThresholdType") == "PERCENTAGE" && v > 1000) {
		return budInvalid("ActionThresholdValue is out of range.")
	}
	return nil
}

func (a *budAction) record(event, message string, now time.Time) {
	a.history = append(a.history, map[string]any{
		"Timestamp": budEpoch(now), "Status": a.a["Status"], "EventType": event,
		"ActionHistoryDetails": map[string]any{"Message": message, "Action": budCloneMap(a.a)},
	})
	if len(a.history) > budMaxHistory {
		a.history = a.history[len(a.history)-budMaxHistory:]
	}
}

func (s *Server) budCreateAction(in map[string]any) (map[string]any, *budErr) {
	account, name, now := budStr(in, "AccountId"), budStr(in, "BudgetName"), time.Now().UTC()
	actionType := budStr(in, "ActionType")
	if err := budCheckDefinition(actionType, budMap(in, "Definition")); err != nil {
		return nil, err
	}
	if err := budCheckThreshold(budMap(in, "ActionThreshold")); err != nil {
		return nil, err
	}
	subs := budSubsIn(budList(in, "Subscribers"))
	if err := budCheckSubscribers(subs); err != nil {
		return nil, err
	}
	tags, err := budTagsIn(budList(in, "ResourceTags"), "InvalidParameterException")
	if err != nil {
		return nil, err
	}
	subList := make([]any, 0, len(subs))
	for _, sub := range subs {
		subList = append(subList, sub)
	}
	id := budNewID()
	record := map[string]any{
		"ActionId": id, "BudgetName": name, "NotificationType": in["NotificationType"], "ActionType": actionType,
		"ActionThreshold": budClone(in["ActionThreshold"]), "Definition": budClone(in["Definition"]),
		"ExecutionRoleArn": in["ExecutionRoleArn"], "ApprovalModel": in["ApprovalModel"], "Status": "STANDBY",
		"Subscribers": subList,
	}
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	b, err := st.get(account, name)
	if err != nil {
		return nil, err
	}
	if len(b.actions) >= budMaxActions {
		return nil, budErrorf("CreationLimitExceededException", "A budget can have at most %d actions.", budMaxActions)
	}
	a := &budAction{a: record, created: now, tags: tags}
	a.record("CREATE_ACTION", "Action created.", now)
	b.actions = append(b.actions, a)
	return map[string]any{"AccountId": account, "BudgetName": name, "ActionId": id}, nil
}

// budLockedAction locks the store and finds the request's budget and action;
// on success the caller must unlock st.mu.
func (s *Server) budLockedAction(in map[string]any) (*budBudget, *budAction, *budErr) {
	st := s.budgets
	st.mu.Lock()
	b, err := st.get(budStr(in, "AccountId"), budStr(in, "BudgetName"))
	if err != nil {
		st.mu.Unlock()
		return nil, nil, err
	}
	a, err := b.actionOrErr(budStr(in, "ActionId"))
	if err != nil {
		st.mu.Unlock()
		return nil, nil, err
	}
	return b, a, nil
}

func (s *Server) budDescribeAction(in map[string]any) (map[string]any, *budErr) {
	b, a, err := s.budLockedAction(in)
	if err != nil {
		return nil, err
	}
	defer s.budgets.mu.Unlock()
	return map[string]any{"AccountId": b.account, "BudgetName": b.name, "Action": budCloneMap(a.a)}, nil
}

func (s *Server) budUpdateAction(in map[string]any) (map[string]any, *budErr) {
	b, a, err := s.budLockedAction(in)
	if err != nil {
		return nil, err
	}
	defer s.budgets.mu.Unlock()
	next := budCloneMap(a.a)
	if def := budMap(in, "Definition"); def != nil {
		if err := budCheckDefinition(budStr(a.a, "ActionType"), def); err != nil {
			return nil, err
		}
		next["Definition"] = budClone(def)
	}
	if th := budMap(in, "ActionThreshold"); th != nil {
		if err := budCheckThreshold(th); err != nil {
			return nil, err
		}
		next["ActionThreshold"] = budClone(th)
	}
	if list := budList(in, "Subscribers"); list != nil {
		subs := budSubsIn(list)
		if err := budCheckSubscribers(subs); err != nil {
			return nil, err
		}
		next["Subscribers"] = budClone(list)
	}
	for _, k := range []string{"NotificationType", "ExecutionRoleArn", "ApprovalModel"} {
		if v := budStr(in, k); v != "" {
			next[k] = v
		}
	}
	old := a.a
	a.a = next
	a.record("UPDATE_ACTION", "Action updated.", time.Now().UTC())
	return map[string]any{"AccountId": b.account, "BudgetName": b.name, "OldAction": budCloneMap(old), "NewAction": budCloneMap(next)}, nil
}

func (s *Server) budDeleteAction(in map[string]any) (map[string]any, *budErr) {
	b, a, err := s.budLockedAction(in)
	if err != nil {
		return nil, err
	}
	defer s.budgets.mu.Unlock()
	kept := make([]*budAction, 0, len(b.actions))
	for _, x := range b.actions {
		if x != a {
			kept = append(kept, x)
		}
	}
	b.actions = kept
	return map[string]any{"AccountId": b.account, "BudgetName": b.name, "Action": budCloneMap(a.a)}, nil
}

func (s *Server) budExecuteAction(in map[string]any) (map[string]any, *budErr) {
	b, a, err := s.budLockedAction(in)
	if err != nil {
		return nil, err
	}
	defer s.budgets.mu.Unlock()
	execType := budStr(in, "ExecutionType")
	tr := budTransitions[execType]
	status := budStr(a.a, "Status")
	allowed := false
	for _, f := range tr.from {
		allowed = allowed || f == status
	}
	if !allowed {
		return nil, budErrorf("ResourceLockedException", "%s isn't allowed while the action is %s.", execType, status)
	}
	next := budCloneMap(a.a)
	next["Status"] = tr.to
	a.a = next
	a.record("EXECUTE_ACTION", execType+" completed.", time.Now().UTC())
	return map[string]any{"AccountId": b.account, "BudgetName": b.name, "ActionId": budStr(a.a, "ActionId"), "ExecutionType": execType}, nil
}

func budActionsPage(in map[string]any, actions []*budAction) (map[string]any, *budErr) {
	start, end, next, err := budPage(budStr(in, "NextToken"), len(actions), budInt(in, "MaxResults", 100), "InvalidNextTokenException")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, a := range actions[start:end] {
		items = append(items, budCloneMap(a.a))
	}
	return map[string]any{"Actions": items, "NextToken": next}, nil
}

func (s *Server) budActionsForBudget(in map[string]any) (map[string]any, *budErr) {
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	b, err := st.get(budStr(in, "AccountId"), budStr(in, "BudgetName"))
	if err != nil {
		return nil, err
	}
	return budActionsPage(in, b.actions)
}

func (s *Server) budActionsForAccount(in map[string]any) (map[string]any, *budErr) {
	st := s.budgets
	st.mu.Lock()
	defer st.mu.Unlock()
	var all []*budAction
	for _, b := range st.sorted(budStr(in, "AccountId")) {
		all = append(all, b.actions...)
	}
	return budActionsPage(in, all)
}

func (s *Server) budActionHistories(in map[string]any) (map[string]any, *budErr) {
	_, a, err := s.budLockedAction(in)
	if err != nil {
		return nil, err
	}
	defer s.budgets.mu.Unlock()
	period := budMap(in, "TimePeriod")
	from, hasFrom := budTime(period["Start"])
	to, hasTo := budTime(period["End"])
	var hist []map[string]any
	for _, h := range a.history {
		t, _ := budTime(h["Timestamp"])
		if (hasFrom && t.Before(from)) || (hasTo && t.After(to)) {
			continue
		}
		hist = append(hist, h)
	}
	start, end, next, err := budPage(budStr(in, "NextToken"), len(hist), budInt(in, "MaxResults", 100), "InvalidNextTokenException")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, h := range hist[start:end] {
		items = append(items, budCloneMap(h))
	}
	return map[string]any{"ActionHistories": items, "NextToken": next}, nil
}
