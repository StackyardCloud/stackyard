package server

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"
)

// AWS Migration Hub state. Stackyard has one region, so it is always the
// Migration Hub home region and HomeRegionNotSetException never applies.
// ponytail: migrationhub-config isn't emulated; add it if a caller needs to
// read or change the home region.

const (
	mghMaxStreams        = 1000
	mghMaxTasks          = 10000 // across all streams
	mghMaxPerTaskList    = 100   // created artifacts, discovered and source resources each
	mghMaxUpdatesPerTask = 1000  // oldest updates are dropped first
	mghMaxApplications   = 10000
	mghDefaultPageSize   = 100
)

type mghErr struct{ code, msg string }

func mghError(code, format string, args ...any) *mghErr {
	return &mghErr{code: code, msg: fmt.Sprintf(format, args...)}
}

type mghNamed struct{ name, description, statusDetail string }

type mghUpdate struct {
	at                   time.Time
	status, statusDetail string
	percent              *int
}

type mghTask struct {
	stream, name         string
	status, statusDetail string
	percent              *int
	updated              time.Time
	attributes           []map[string]any
	artifacts            []mghNamed // CreatedArtifact
	discovered           []mghNamed // DiscoveredResource: name holds the ConfigurationId
	sources              []mghNamed // SourceResource
	updates              []mghUpdate
}

type mghAppState struct {
	status  string
	updated time.Time
}

type migrationHubStore struct {
	mu      sync.Mutex
	streams map[string]map[string]*mghTask // stream name -> task name -> task
	tasks   int
	apps    map[string]*mghAppState
}

func newMigrationHubStore() *migrationHubStore {
	return &migrationHubStore{streams: map[string]map[string]*mghTask{}, apps: map[string]*mghAppState{}}
}

// ---- progress update streams ----

func (st *migrationHubStore) createStream(name string, dryRun bool) *mghErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.streams[name]; !ok && len(st.streams) >= mghMaxStreams {
		return mghError("InvalidInputException", "An account can have at most %d progress update streams.", mghMaxStreams)
	}
	if dryRun {
		return mghDryRun()
	}
	if _, ok := st.streams[name]; !ok { // creating an existing stream is a no-op
		st.streams[name] = map[string]*mghTask{}
	}
	return nil
}

func (st *migrationHubStore) deleteStream(name string, dryRun bool) *mghErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	tasks, ok := st.streams[name]
	if !ok {
		return mghError("ResourceNotFoundException", "Progress update stream %s does not exist.", name)
	}
	if dryRun {
		return mghDryRun()
	}
	st.tasks -= len(tasks) // AWS deletes the stream's tasks with it
	delete(st.streams, name)
	return nil
}

func (st *migrationHubStore) listStreams(token string, max int) (map[string]any, *mghErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	names := make([]string, 0, len(st.streams))
	for n := range st.streams {
		names = append(names, n)
	}
	sort.Strings(names)
	page, next, err := mghPage(names, token, max)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, n := range page {
		out = append(out, map[string]any{"ProgressUpdateStreamName": n})
	}
	return mghWithToken(map[string]any{"ProgressUpdateStreamSummaryList": out}, next), nil
}

// ---- migration tasks ----

// taskLocked returns the task, or ResourceNotFoundException for a missing
// stream or task.
func (st *migrationHubStore) taskLocked(stream, name string) (*mghTask, *mghErr) {
	tasks, ok := st.streams[stream]
	if !ok {
		return nil, mghError("ResourceNotFoundException", "Progress update stream %s does not exist.", stream)
	}
	t, ok := tasks[name]
	if !ok {
		return nil, mghError("ResourceNotFoundException", "Migration task %s does not exist in progress update stream %s.", name, stream)
	}
	return t, nil
}

func (st *migrationHubStore) importTask(stream, name string, dryRun bool, now time.Time) *mghErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	tasks, ok := st.streams[stream]
	if !ok {
		return mghError("ResourceNotFoundException", "Progress update stream %s does not exist.", stream)
	}
	if _, exists := tasks[name]; !exists && st.tasks >= mghMaxTasks {
		return mghError("InvalidInputException", "An account can have at most %d migration tasks.", mghMaxTasks)
	}
	if dryRun {
		return mghDryRun()
	}
	if _, exists := tasks[name]; !exists { // importing an existing task is a no-op
		tasks[name] = &mghTask{stream: stream, name: name, status: "NOT_STARTED", updated: now}
		st.tasks++
	}
	return nil
}

// notifyTask records a state update. An update stamped earlier than the
// task's latest one is stale and rejected, so out-of-order notifications
// can't roll the task back.
func (st *migrationHubStore) notifyTask(stream, name string, u mghUpdate, dryRun bool) *mghErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	t, err := st.taskLocked(stream, name)
	if err != nil {
		return err
	}
	if len(t.updates) > 0 && u.at.Before(t.updated) {
		return mghError("InvalidInputException", "UpdateDateTime %s is older than the task's latest update at %s.", u.at.Format(time.RFC3339), t.updated.Format(time.RFC3339))
	}
	if dryRun {
		return mghDryRun()
	}
	t.status, t.statusDetail, t.percent, t.updated = u.status, u.statusDetail, u.percent, u.at
	t.updates = append(t.updates, u)
	if len(t.updates) > mghMaxUpdatesPerTask {
		t.updates = t.updates[len(t.updates)-mghMaxUpdatesPerTask:]
	}
	return nil
}

func (st *migrationHubStore) putAttributes(stream, name string, attrs []map[string]any, dryRun bool) *mghErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	t, err := st.taskLocked(stream, name)
	if err != nil {
		return err
	}
	if dryRun {
		return mghDryRun()
	}
	t.attributes = attrs
	return nil
}

func (t *mghTask) summary() map[string]any {
	out := map[string]any{
		"ProgressUpdateStream": t.stream, "MigrationTaskName": t.name,
		"Status": t.status, "UpdateDateTime": mghEpoch(t.updated),
	}
	if t.statusDetail != "" {
		out["StatusDetail"] = t.statusDetail
	}
	if t.percent != nil {
		out["ProgressPercent"] = *t.percent
	}
	return out
}

func mghTaskState(status, detail string, percent *int) map[string]any {
	out := map[string]any{"Status": status}
	if detail != "" {
		out["StatusDetail"] = detail
	}
	if percent != nil {
		out["ProgressPercent"] = *percent
	}
	return out
}

func (st *migrationHubStore) describeTask(stream, name string) (map[string]any, *mghErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	t, err := st.taskLocked(stream, name)
	if err != nil {
		return nil, err
	}
	attrs := make([]any, 0, len(t.attributes))
	for _, a := range t.attributes {
		attrs = append(attrs, map[string]any{"Type": a["Type"], "Value": a["Value"]})
	}
	return map[string]any{"MigrationTask": map[string]any{
		"ProgressUpdateStream": t.stream, "MigrationTaskName": t.name,
		"Task": mghTaskState(t.status, t.statusDetail, t.percent), "UpdateDateTime": mghEpoch(t.updated),
		"ResourceAttributeList": attrs,
	}}, nil
}

// listTasks lists every task, or with resourceName only tasks associated
// with that discovered resource or carrying it as a resource attribute value.
func (st *migrationHubStore) listTasks(resourceName, token string, max int) (map[string]any, *mghErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	var all []*mghTask
	for _, tasks := range st.streams {
		for _, t := range tasks {
			if resourceName == "" || t.matches(resourceName) {
				all = append(all, t)
			}
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].stream != all[j].stream {
			return all[i].stream < all[j].stream
		}
		return all[i].name < all[j].name
	})
	page, next, err := mghPage(all, token, max)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, t := range page {
		out = append(out, t.summary())
	}
	return mghWithToken(map[string]any{"MigrationTaskSummaryList": out}, next), nil
}

func (t *mghTask) matches(resource string) bool {
	for _, d := range t.discovered {
		if d.name == resource {
			return true
		}
	}
	for _, a := range t.attributes {
		if v, _ := a["Value"].(string); v == resource {
			return true
		}
	}
	return false
}

func (st *migrationHubStore) listUpdates(stream, name, token string, max int) (map[string]any, *mghErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	t, err := st.taskLocked(stream, name)
	if err != nil {
		return nil, err
	}
	page, next, perr := mghPage(t.updates, token, max)
	if perr != nil {
		return nil, perr
	}
	out := []any{}
	for _, u := range page {
		out = append(out, map[string]any{
			"UpdateDateTime": mghEpoch(u.at), "UpdateType": "MIGRATION_TASK_STATE_UPDATED",
			"MigrationTaskState": mghTaskState(u.status, u.statusDetail, u.percent),
		})
	}
	return mghWithToken(map[string]any{"MigrationTaskUpdateList": out}, next), nil
}

// ---- task associations (created artifacts, discovered and source resources) ----

// mghList picks one of a task's association lists.
type mghList func(t *mghTask) *[]mghNamed

var (
	mghArtifacts  mghList = func(t *mghTask) *[]mghNamed { return &t.artifacts }
	mghDiscovered mghList = func(t *mghTask) *[]mghNamed { return &t.discovered }
	mghSources    mghList = func(t *mghTask) *[]mghNamed { return &t.sources }
)

// associate adds or, for a name already associated, updates an entry.
func (st *migrationHubStore) associate(stream, name string, list mghList, item mghNamed, dryRun bool) *mghErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	t, err := st.taskLocked(stream, name)
	if err != nil {
		return err
	}
	items := list(t)
	idx := -1
	for i, existing := range *items {
		if existing.name == item.name {
			idx = i
		}
	}
	if idx < 0 && len(*items) >= mghMaxPerTaskList {
		return mghError("InvalidInputException", "A migration task can have at most %d associations of this kind.", mghMaxPerTaskList)
	}
	if dryRun {
		return mghDryRun()
	}
	if idx >= 0 {
		(*items)[idx] = item
	} else {
		*items = append(*items, item)
	}
	return nil
}

func (st *migrationHubStore) disassociate(stream, name string, list mghList, itemName, kind string, dryRun bool) *mghErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	t, err := st.taskLocked(stream, name)
	if err != nil {
		return err
	}
	items := list(t)
	for i, existing := range *items {
		if existing.name == itemName {
			if dryRun {
				return mghDryRun()
			}
			*items = append((*items)[:i], (*items)[i+1:]...)
			return nil
		}
	}
	return mghError("ResourceNotFoundException", "%s %s is not associated with migration task %s.", kind, itemName, name)
}

// listAssociations renders a task's list with render, which maps an entry to
// its output shape.
func (st *migrationHubStore) listAssociations(stream, name string, list mghList, render func(mghNamed) map[string]any, key, token string, max int) (map[string]any, *mghErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	t, err := st.taskLocked(stream, name)
	if err != nil {
		return nil, err
	}
	page, next, perr := mghPage(*list(t), token, max)
	if perr != nil {
		return nil, perr
	}
	out := []any{}
	for _, item := range page {
		out = append(out, render(item))
	}
	return mghWithToken(map[string]any{key: out}, next), nil
}

// ---- application states ----

func (st *migrationHubStore) notifyApplication(id, status string, at time.Time, dryRun bool) *mghErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.apps[id]; !ok && len(st.apps) >= mghMaxApplications {
		return mghError("InvalidInputException", "At most %d application states can be recorded.", mghMaxApplications)
	}
	if dryRun {
		return mghDryRun()
	}
	st.apps[id] = &mghAppState{status: status, updated: at}
	return nil
}

func (st *migrationHubStore) describeApplication(id string) (map[string]any, *mghErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, ok := st.apps[id]
	if !ok {
		return nil, mghError("ResourceNotFoundException", "No state has been recorded for application %s.", id)
	}
	return map[string]any{"ApplicationStatus": a.status, "LastUpdatedTime": mghEpoch(a.updated)}, nil
}

func (st *migrationHubStore) listApplications(ids []string, token string, max int) (map[string]any, *mghErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(ids) == 0 {
		for id := range st.apps {
			ids = append(ids, id)
		}
	}
	var keep []string
	for _, id := range ids {
		if _, ok := st.apps[id]; ok {
			keep = append(keep, id)
		}
	}
	sort.Strings(keep)
	page, next, err := mghPage(keep, token, max)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, id := range page {
		a := st.apps[id]
		out = append(out, map[string]any{"ApplicationId": id, "ApplicationStatus": a.status, "LastUpdatedTime": mghEpoch(a.updated)})
	}
	return mghWithToken(map[string]any{"ApplicationStateList": out}, next), nil
}

// ---- helpers ----

func mghDryRun() *mghErr {
	return mghError("DryRunOperation", "Request would have succeeded, but the DryRun flag is set.")
}

func mghEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

// mghPage slices items at an offset token; the token is the decimal offset.
func mghPage[T any](items []T, token string, max int) ([]T, string, *mghErr) {
	start := 0
	if token != "" {
		n, err := strconv.ParseInt(token, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", mghError("InvalidInputException", "The NextToken is not valid.")
		}
		start = int(n)
	}
	end := min(start+max, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[start:end], next, nil
}

func mghWithToken(out map[string]any, next string) map[string]any {
	if next != "" {
		out["NextToken"] = next
	}
	return out
}

// mghCheckAttribute validates a resource attribute value for its type.
func mghCheckAttribute(typ, value string) *mghErr {
	bad := false
	switch typ {
	case "IPV4_ADDRESS":
		ip := net.ParseIP(value)
		bad = ip == nil || ip.To4() == nil
	case "IPV6_ADDRESS":
		ip := net.ParseIP(value)
		bad = ip == nil || ip.To4() != nil
	case "MAC_ADDRESS":
		_, err := net.ParseMAC(value)
		bad = err != nil
	}
	if bad {
		return mghError("InvalidInputException", "%q is not a valid %s.", value, typ)
	}
	return nil
}
