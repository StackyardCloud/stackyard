package server

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// S3-to-S3 executions really copy objects; these bound one execution.
	dsyMaxCopyObjects = 10000
	dsyMaxCopyBytes   = 256 << 20
	dsyMaxPatterns    = 100
	// ponytail: filters are capped at 4 KiB (AWS allows 100 KiB) so matching
	// every object stays cheap; raise with care, it runs under the store lock.
	dsyMaxFilterBytes = 4096
	dsyMaxExecs       = 10000
)

type dsyTask struct {
	seq     int
	arn     string
	src     string
	dst     string
	created time.Time
	attrs   map[string]any // Name, Options, Includes, Excludes, Schedule, ...
	current string         // execution in flight, if any
	queue   []string       // executions waiting behind current
	history []string       // finished executions, oldest first
	tags    map[string]string
}

type dsyExec struct {
	seq      int
	arn      string
	task     string
	status   string
	options  map[string]any
	includes []any
	excludes []any
	extra    map[string]any // ManifestConfig, TaskReportConfig, TaskMode
	start    time.Time
	launched time.Time
	ended    time.Time
	counts   map[string]float64
	result   map[string]any
	tags     map[string]string
}

var (
	dsyLogGroupPattern = regexp.MustCompile(`^arn:aws[-a-z]*:logs:[a-z0-9-]+:\d{12}:log-group:[^*]{1,512}?(:\*)?$`)
	dsyRatePattern     = regexp.MustCompile(`^rate\((\d{1,6}) (minute|minutes|hour|hours|day|days)\)$`)
	dsyCronPattern     = regexp.MustCompile(`^cron\(\S+( \S+){5}\)$`)
)

// dsyDefaultOptions are what DescribeTask reports for options never set.
var dsyDefaultOptions = map[string]any{
	"VerifyMode": "POINT_IN_TIME_CONSISTENT", "OverwriteMode": "ALWAYS", "Atime": "BEST_EFFORT", "Mtime": "PRESERVE",
	"Uid": "INT_VALUE", "Gid": "INT_VALUE", "PreserveDeletedFiles": "PRESERVE", "PreserveDevices": "NONE",
	"PosixPermissions": "PRESERVE", "BytesPerSecond": float64(-1), "TaskQueueing": "ENABLED", "LogLevel": "OFF",
	"TransferMode": "CHANGED", "SecurityDescriptorCopyFlags": "OWNER_DACL", "ObjectTags": "PRESERVE", "ObjectMetadata": "PRESERVE",
}

func init() {
	dsyHandlers["CreateTask"] = dsyCreateTask
	dsyHandlers["DescribeTask"] = dsyDescribeTask
	dsyHandlers["UpdateTask"] = dsyUpdateTask
	dsyHandlers["DeleteTask"] = dsyDeleteTask
	dsyHandlers["ListTasks"] = dsyListTasks
	dsyHandlers["StartTaskExecution"] = dsyStartTaskExecution
	dsyHandlers["DescribeTaskExecution"] = dsyDescribeTaskExecution
	dsyHandlers["UpdateTaskExecution"] = dsyUpdateTaskExecution
	dsyHandlers["CancelTaskExecution"] = dsyCancelTaskExecution
	dsyHandlers["ListTaskExecutions"] = dsyListTaskExecutions
}

// dsyMergeOptions overlays opts onto base (defaults when base is nil) and
// checks the combinations DataSync rejects.
func dsyMergeOptions(base, opts map[string]any) (map[string]any, *dsyErr) {
	out := dsyCopyMap(dsyDefaultOptions)
	for k, v := range base {
		out[k] = v
	}
	for k, v := range opts {
		out[k] = v
	}
	atime, mtime := out["Atime"], out["Mtime"]
	if (atime == "BEST_EFFORT") != (mtime == "PRESERVE") {
		return nil, dsyInvalid("Atime BEST_EFFORT needs Mtime PRESERVE, and Atime NONE needs Mtime NONE")
	}
	if bps, _ := out["BytesPerSecond"].(float64); bps == 0 {
		return nil, dsyInvalid("Options.BytesPerSecond must be -1 (no limit) or at least 1")
	}
	return out, nil
}

// dsyCheckFilters validates Includes/Excludes SIMPLE_PATTERN filters.
func dsyCheckFilters(label string, filters []any) *dsyErr {
	for _, raw := range filters {
		f, _ := raw.(map[string]any)
		if len(dsyStr(f, "Value")) > dsyMaxFilterBytes {
			return dsyInvalid("%s filters can be at most %d bytes", label, dsyMaxFilterBytes)
		}
		patterns := strings.Split(dsyStr(f, "Value"), "|")
		if len(patterns) > dsyMaxPatterns {
			return dsyInvalid("%s can hold at most %d patterns", label, dsyMaxPatterns)
		}
		for _, p := range patterns {
			if p == "" || (!strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "*")) {
				return dsyInvalid("%s pattern %q must start with / or *", label, p)
			}
		}
	}
	return nil
}

func dsyCheckSchedule(sched map[string]any) *dsyErr {
	if sched == nil {
		return nil
	}
	expr := dsyStr(sched, "ScheduleExpression")
	if m := dsyRatePattern.FindStringSubmatch(expr); m != nil {
		n, _ := strconv.ParseInt(m[1], 10, 32)
		if strings.HasPrefix(m[2], "minute") && n < 60 {
			return dsyInvalid("schedules can run at most once an hour")
		}
		if n < 1 {
			return dsyInvalid("schedule rate must be at least 1")
		}
		return nil
	}
	if !dsyCronPattern.MatchString(expr) {
		return dsyInvalid("ScheduleExpression %q must be rate(...) or cron(...)", expr)
	}
	return nil
}

// dsyCheckTaskAttrs validates the task members shared by Create and Update.
func (st *dsyStore) dsyCheckTaskAttrs(t *dsyTask) *dsyErr {
	if lg := dsyStr(t.attrs, "CloudWatchLogGroupArn"); lg != "" && !dsyLogGroupPattern.MatchString(lg) {
		return dsyInvalid("CloudWatchLogGroupArn %q is not a log group ARN", lg)
	}
	for _, label := range []string{"Includes", "Excludes"} {
		list, _ := t.attrs[label].([]any)
		if err := dsyCheckFilters(label, list); err != nil {
			return err
		}
	}
	if err := dsyCheckSchedule(dsyMap(t.attrs, "Schedule")); err != nil {
		return err
	}
	if mc := dsyMap(t.attrs, "ManifestConfig"); mc != nil && mc["Source"] == nil {
		return dsyInvalid("ManifestConfig needs a Source")
	}
	// ponytail: Enhanced mode is S3-to-S3 only here; AWS has since added a
	// few more location pairs.
	if dsyStr(t.attrs, "TaskMode") == "ENHANCED" && (st.locations[t.src].kind != "S3" || st.locations[t.dst].kind != "S3") {
		return dsyInvalid("Enhanced mode tasks need S3 source and destination locations")
	}
	return nil
}

func dsyCreateTask(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	st := s.datasync
	if len(st.tasks) >= dsyMaxTasks {
		return nil, dsyInvalid("an account can have at most %d tasks", dsyMaxTasks)
	}
	t := &dsyTask{src: dsyStr(in, "SourceLocationArn"), dst: dsyStr(in, "DestinationLocationArn"), tags: map[string]string{}}
	if _, err := st.location(t.src, ""); err != nil {
		return nil, err
	}
	if _, err := st.location(t.dst, ""); err != nil {
		return nil, err
	}
	if t.src == t.dst {
		return nil, dsyInvalid("SourceLocationArn and DestinationLocationArn must differ")
	}
	t.attrs = dsyCopyMap(in)
	for _, k := range []string{"SourceLocationArn", "DestinationLocationArn", "Tags"} {
		delete(t.attrs, k)
	}
	opts, err := dsyMergeOptions(nil, dsyMap(in, "Options"))
	if err != nil {
		return nil, err
	}
	t.attrs["Options"] = opts
	if _, ok := t.attrs["TaskMode"]; !ok {
		t.attrs["TaskMode"] = "BASIC"
	}
	if err := st.dsyCheckTaskAttrs(t); err != nil {
		return nil, err
	}
	if err := dsyTagsFrom(in, "Tags", t.tags); err != nil {
		return nil, err
	}
	t.seq, t.arn, t.created = st.next(), dsyARN("task/"+dsyID("task-")), time.Now().UTC()
	st.tasks[t.arn] = t
	return map[string]any{"TaskArn": t.arn}, nil
}

func (st *dsyStore) task(arn string) (*dsyTask, *dsyErr) {
	if t, ok := st.tasks[arn]; ok {
		return t, nil
	}
	return nil, dsyNotFound("Task", arn)
}

func (t *dsyTask) status() string {
	if t.current != "" {
		return "RUNNING"
	}
	return "AVAILABLE"
}

func dsyDescribeTask(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	t, err := s.datasync.task(dsyStr(in, "TaskArn"))
	if err != nil {
		return nil, err
	}
	out := dsyCopyMap(t.attrs)
	out["TaskArn"], out["Status"], out["CreationTime"] = t.arn, t.status(), t.created
	out["SourceLocationArn"], out["DestinationLocationArn"] = t.src, t.dst
	out["CurrentTaskExecutionArn"] = dsyOptional(t.current)
	out["SourceNetworkInterfaceArns"], out["DestinationNetworkInterfaceArns"] = []any{}, []any{}
	if sched := dsyMap(t.attrs, "Schedule"); sched != nil {
		if _, ok := sched["Status"]; !ok {
			sched["Status"] = "ENABLED"
		}
		out["ScheduleDetails"] = map[string]any{"StatusUpdateTime": t.created}
	}
	return out, nil
}

func dsyUpdateTask(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	st := s.datasync
	t, err := st.task(dsyStr(in, "TaskArn"))
	if err != nil {
		return nil, err
	}
	next := *t
	next.attrs = dsyCopyMap(t.attrs)
	for k, v := range dsyCopyMap(in) {
		if k != "TaskArn" && k != "Options" {
			next.attrs[k] = v
		}
	}
	base, _ := t.attrs["Options"].(map[string]any)
	opts, oerr := dsyMergeOptions(base, dsyMap(in, "Options"))
	if oerr != nil {
		return nil, oerr
	}
	next.attrs["Options"] = opts
	if err := st.dsyCheckTaskAttrs(&next); err != nil {
		return nil, err
	}
	t.attrs = next.attrs
	return map[string]any{}, nil
}

func dsyDeleteTask(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	st := s.datasync
	t, err := st.task(dsyStr(in, "TaskArn"))
	if err != nil {
		return nil, err
	}
	for _, arn := range append(append(append([]string{}, t.history...), t.queue...), t.current) {
		delete(st.execs, arn)
	}
	delete(st.tasks, t.arn)
	return map[string]any{}, nil
}

func dsyListTasks(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	tasks := make([]*dsyTask, 0, len(s.datasync.tasks))
	for _, t := range s.datasync.tasks {
		if dsyTaskMatches(t, in) {
			tasks = append(tasks, t)
		}
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].seq < tasks[j].seq })
	start, end, next, err := dsyPage(in, len(tasks))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, t := range tasks[start:end] {
		list = append(list, map[string]any{"TaskArn": t.arn, "Status": t.status(), "Name": t.attrs["Name"], "TaskMode": t.attrs["TaskMode"]})
	}
	return map[string]any{"Tasks": list, "NextToken": dsyOptional(next)}, nil
}

func dsyTaskMatches(t *dsyTask, in map[string]any) bool {
	filters, _ := in["Filters"].([]any)
	for _, raw := range filters {
		f, _ := raw.(map[string]any)
		op, values := dsyStr(f, "Operator"), dsyStrings(f, "Values")
		switch dsyStr(f, "Name") {
		case "LocationId":
			if !dsyFilterMatch(op, t.src, values) && !dsyFilterMatch(op, t.dst, values) {
				return false
			}
		case "CreationTime":
			if !dsyFilterMatch(op, fmt.Sprint(t.created.Unix()), values) {
				return false
			}
		}
	}
	return true
}

// --- executions ------------------------------------------------------------

func dsyStartTaskExecution(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	st := s.datasync
	t, err := st.task(dsyStr(in, "TaskArn"))
	if err != nil {
		return nil, err
	}
	base, _ := t.attrs["Options"].(map[string]any)
	opts, oerr := dsyMergeOptions(base, dsyMap(in, "OverrideOptions"))
	if oerr != nil {
		return nil, oerr
	}
	e := &dsyExec{task: t.arn, options: opts, counts: map[string]float64{}, tags: map[string]string{}, start: time.Now().UTC(), extra: map[string]any{}}
	e.includes, _ = t.attrs["Includes"].([]any)
	e.excludes, _ = t.attrs["Excludes"].([]any)
	if v, ok := in["Includes"].([]any); ok {
		e.includes = v
	}
	if v, ok := in["Excludes"].([]any); ok {
		e.excludes = v
	}
	if err := dsyCheckFilters("Includes", e.includes); err != nil {
		return nil, err
	}
	if err := dsyCheckFilters("Excludes", e.excludes); err != nil {
		return nil, err
	}
	for _, k := range []string{"ManifestConfig", "TaskReportConfig", "TaskMode"} {
		if v, ok := in[k]; ok {
			e.extra[k] = v
		} else if v, ok := t.attrs[k]; ok {
			e.extra[k] = v
		}
	}
	if err := dsyTagsFrom(in, "Tags", e.tags); err != nil {
		return nil, err
	}
	switch {
	case t.current == "":
		e.status, e.launched = "LAUNCHING", e.start
	case opts["TaskQueueing"] == "ENABLED":
		e.status = "QUEUED"
	default:
		return nil, dsyInvalid("task %s already has an execution running and task queueing is disabled", t.arn)
	}
	if len(st.execs) >= dsyMaxExecs {
		return nil, dsyInvalid("an account can keep at most %d task executions; delete tasks to free some", dsyMaxExecs)
	}
	if len(t.queue) >= dsyMaxExecHistory {
		return nil, dsyInvalid("task %s has too many queued executions", t.arn)
	}
	e.seq, e.arn = st.next(), t.arn+"/execution/"+dsyID("exec-")
	st.execs[e.arn] = e
	if e.status == "QUEUED" {
		t.queue = append(t.queue, e.arn)
	} else {
		t.current = e.arn
	}
	return map[string]any{"TaskExecutionArn": e.arn}, nil
}

func (st *dsyStore) exec(arn string) (*dsyExec, *dsyErr) {
	if e, ok := st.execs[arn]; ok {
		return e, nil
	}
	return nil, dsyNotFound("Task execution", arn)
}

func dsyTerminal(status string) bool { return status == "SUCCESS" || status == "ERROR" }

// finish records a finished execution and starts the next queued one.
func (st *dsyStore) finish(s *Server, e *dsyExec) {
	e.ended = time.Now().UTC()
	t := st.tasks[e.task]
	if t == nil {
		return
	}
	if t.current == e.arn {
		t.current = ""
	}
	t.history = append(t.history, e.arn)
	for len(t.history) > dsyMaxExecHistory {
		delete(st.execs, t.history[0])
		t.history = t.history[1:]
	}
	if t.current == "" && len(t.queue) > 0 {
		t.current, t.queue = t.queue[0], t.queue[1:]
		if q := st.execs[t.current]; q != nil {
			q.status, q.launched = "LAUNCHING", time.Now().UTC()
		}
	}
}

// advance moves an execution one phase on, as a read of it would observe.
func (st *dsyStore) advance(s *Server, e *dsyExec) {
	switch e.status {
	case "LAUNCHING":
		e.status = "PREPARING"
	case "PREPARING":
		e.status = "TRANSFERRING"
		e.result = map[string]any{"PrepareStatus": "SUCCESS", "TransferStatus": "PENDING", "VerifyStatus": "PENDING"}
	case "TRANSFERRING":
		if err := st.transfer(s, e); err != nil {
			e.status = "ERROR"
			e.result["TransferStatus"], e.result["ErrorCode"], e.result["ErrorDetail"] = "ERROR", err.code, err.msg
			st.finish(s, e)
			return
		}
		e.status = "VERIFYING"
		e.result["TransferStatus"] = "SUCCESS"
	case "VERIFYING":
		e.status = "SUCCESS"
		e.result["VerifyStatus"] = "SUCCESS"
		if e.options["VerifyMode"] != "NONE" {
			e.counts["FilesVerified"] = e.counts["FilesTransferred"]
		}
		st.finish(s, e)
	case "CANCELLING":
		e.status = "ERROR"
		if e.result == nil {
			e.result = map[string]any{}
		}
		e.result["ErrorCode"], e.result["ErrorDetail"] = "OpCancelled", "The task execution was cancelled."
		st.finish(s, e)
	}
}

func dsyDescribeTaskExecution(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	st := s.datasync
	e, err := st.exec(dsyStr(in, "TaskExecutionArn"))
	if err != nil {
		return nil, err
	}
	st.advance(s, e)
	out := map[string]any{
		"TaskExecutionArn": e.arn, "Status": e.status, "Options": e.options, "StartTime": e.start,
		"Includes": e.includes, "Excludes": e.excludes, "Result": e.result,
		"FilesListed": map[string]any{"AtSource": e.counts["FilesListed"], "AtDestinationForDelete": e.counts["DestListed"]},
	}
	for k, v := range e.extra {
		out[k] = v
	}
	for _, k := range []string{"EstimatedFilesToTransfer", "EstimatedBytesToTransfer", "FilesTransferred", "BytesWritten",
		"BytesTransferred", "FilesDeleted", "FilesSkipped", "FilesVerified", "FilesPrepared", "EstimatedFilesToDelete"} {
		out[k] = e.counts[k]
	}
	if !e.launched.IsZero() {
		out["LaunchTime"] = e.launched
	}
	if !e.ended.IsZero() {
		out["EndTime"] = e.ended
	}
	return out, nil
}

func dsyUpdateTaskExecution(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	e, err := s.datasync.exec(dsyStr(in, "TaskExecutionArn"))
	if err != nil {
		return nil, err
	}
	if dsyTerminal(e.status) {
		return nil, dsyInvalid("task execution %s has already finished", e.arn)
	}
	// Only the bandwidth limit can change while an execution runs.
	opts := dsyMap(in, "Options")
	for k := range opts {
		if k != "BytesPerSecond" {
			return nil, dsyInvalid("only Options.BytesPerSecond can be updated on a running execution")
		}
	}
	merged, oerr := dsyMergeOptions(e.options, opts)
	if oerr != nil {
		return nil, oerr
	}
	e.options = merged
	return map[string]any{}, nil
}

func dsyCancelTaskExecution(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	st := s.datasync
	e, err := st.exec(dsyStr(in, "TaskExecutionArn"))
	if err != nil {
		return nil, err
	}
	if dsyTerminal(e.status) || e.status == "CANCELLING" {
		return nil, dsyInvalid("task execution %s is not running", e.arn)
	}
	if e.status == "QUEUED" {
		if t := st.tasks[e.task]; t != nil {
			for i, arn := range t.queue {
				if arn == e.arn {
					t.queue = append(t.queue[:i], t.queue[i+1:]...)
					break
				}
			}
		}
	}
	e.status = "CANCELLING"
	return map[string]any{}, nil
}

func dsyListTaskExecutions(s *Server, in map[string]any) (map[string]any, *dsyErr) {
	st := s.datasync
	taskARN := dsyStr(in, "TaskArn")
	if taskARN != "" {
		if _, err := st.task(taskARN); err != nil {
			return nil, err
		}
	}
	execs := make([]*dsyExec, 0, len(st.execs))
	for _, e := range st.execs {
		if taskARN == "" || e.task == taskARN {
			execs = append(execs, e)
		}
	}
	sort.Slice(execs, func(i, j int) bool { return execs[i].seq < execs[j].seq })
	start, end, next, err := dsyPage(in, len(execs))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, e := range execs[start:end] {
		list = append(list, map[string]any{"TaskExecutionArn": e.arn, "Status": e.status, "TaskMode": e.extra["TaskMode"]})
	}
	return map[string]any{"TaskExecutions": list, "NextToken": dsyOptional(next)}, nil
}
