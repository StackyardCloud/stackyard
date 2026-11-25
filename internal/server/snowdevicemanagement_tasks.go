package server

import (
	"strings"
	"time"
)

// Snow Device Management tasks, executions and tags.

func (t *sdmTask) out() map[string]any {
	out := map[string]any{
		"taskId": t.id, "taskArn": t.arn, "state": t.state, "targets": opstAnyList(t.targets), "tags": sdmTagsOut(t.tags),
		"createdAt": budEpoch(t.created), "lastUpdatedAt": budEpoch(t.updated),
	}
	if t.description != "" {
		out["description"] = t.description
	}
	if !t.completed.IsZero() {
		out["completedAt"] = budEpoch(t.completed)
	}
	return out
}

func sdmTerminal(state string) bool {
	return state != "QUEUED" && state != "IN_PROGRESS"
}

// step advances one execution: QUEUED -> IN_PROGRESS -> SUCCEEDED. A device
// that left management (its job was cancelled) fails the execution; an
// unlock moves the device through UNLOCKING to UNLOCKED.
func (st *sdmStore) step(t *sdmTask, e *sdmExecution, devices []sdmDevice, now time.Time) {
	if sdmTerminal(e.state) {
		return
	}
	fail := func() {
		if d := st.devices[e.device]; d != nil && e.state == "IN_PROGRESS" && t.command == "unlock" && d.state == "UNLOCKING" {
			d.state, d.updated = "LOCKED", now
		}
		e.state, e.updated = "FAILED", now
		st.settle(t, now)
	}
	if sdmFind(devices, e.device) == nil {
		fail()
		return
	}
	d, err := st.state(e.device, now)
	if err != nil {
		fail()
		return
	}
	switch e.state {
	case "QUEUED":
		e.state, e.started, e.updated = "IN_PROGRESS", now, now
		if t.command == "unlock" && d.state == "LOCKED" {
			d.state, d.updated = "UNLOCKING", now
		}
	case "IN_PROGRESS":
		e.state, e.updated = "SUCCEEDED", now
		d.reachedOut, d.updated = now, now
		if t.command == "unlock" {
			d.state = "UNLOCKED"
		}
	}
	st.settle(t, now)
}

// settle completes a task once every execution has finished.
func (st *sdmStore) settle(t *sdmTask, now time.Time) {
	if t.state != "IN_PROGRESS" {
		return
	}
	for _, e := range t.executions {
		if !sdmTerminal(e.state) {
			return
		}
	}
	t.state, t.updated, t.completed = "COMPLETED", now, now
}

func (st *sdmStore) task(id string) (*sdmTask, *budErr) {
	if t := st.tasks[id]; t != nil {
		return t, nil
	}
	return nil, sdmNotFound("Task", id)
}

func (st *sdmStore) createTask(devices []sdmDevice, in map[string]any, now time.Time) (map[string]any, *budErr) {
	token := budStr(in, "clientToken")
	if token != "" {
		for _, t := range st.tasks {
			if t.clientToken == token {
				return map[string]any{"taskId": t.id, "taskArn": t.arn}, nil
			}
		}
	}
	tags, err := sdmTagsIn(in)
	if err != nil {
		return nil, err
	}
	cmd := budMap(in, "command")
	var command string
	switch {
	case len(cmd) != 1:
		return nil, sdmErr("ValidationException", "command must set exactly one of reboot or unlock")
	case cmd["reboot"] != nil:
		command = "reboot"
	default:
		command = "unlock"
	}
	raw := budStrings(in, "targets")
	if len(raw) == 0 || len(raw) > sdmMaxTargets {
		return nil, sdmErr("ValidationException", "targets must list 1 to %d managed devices", sdmMaxTargets)
	}
	if len(st.tasks) >= sdmMaxTasks {
		return nil, sdmErr("ServiceQuotaExceededException", "at most %d tasks are kept", sdmMaxTasks)
	}
	id := "st-" + opstHex(17)
	t := &sdmTask{
		id: id, arn: sdmARNBase + "task/" + id, command: command, description: budStr(in, "description"), state: "IN_PROGRESS",
		clientToken: token, executions: map[string]*sdmExecution{}, tags: tags, created: now, updated: now,
	}
	for _, target := range raw {
		d := sdmFind(devices, target)
		if d == nil {
			return nil, sdmNotFound("Managed device", target)
		}
		if t.executions[d.id] != nil {
			continue
		}
		t.targets = append(t.targets, d.id)
		t.executions[d.id] = &sdmExecution{id: "1", device: d.id, state: "QUEUED", updated: now}
	}
	st.tasks[id] = t
	st.taskOrder = append(st.taskOrder, id)
	return map[string]any{"taskId": t.id, "taskArn": t.arn}, nil
}

func (st *sdmStore) describeTask(devices []sdmDevice, in map[string]any, now time.Time) (map[string]any, *budErr) {
	t, err := st.task(budStr(in, "taskId"))
	if err != nil {
		return nil, err
	}
	for _, d := range t.targets {
		st.step(t, t.executions[d], devices, now)
	}
	return t.out(), nil
}

func (st *sdmStore) listTasks(_ []sdmDevice, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	state := budStr(in, "state")
	var match []*sdmTask
	for _, id := range st.taskOrder {
		if t := st.tasks[id]; state == "" || t.state == state {
			match = append(match, t)
		}
	}
	start, end, next, err := sdmPage(in, len(match))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, t := range match[start:end] {
		list = append(list, map[string]any{"taskId": t.id, "taskArn": t.arn, "state": t.state, "tags": sdmTagsOut(t.tags)})
	}
	return map[string]any{"tasks": list, "nextToken": next}, nil
}

func (st *sdmStore) cancelTask(_ []sdmDevice, in map[string]any, now time.Time) (map[string]any, *budErr) {
	t, err := st.task(budStr(in, "taskId"))
	if err != nil {
		return nil, err
	}
	if t.state != "IN_PROGRESS" {
		return nil, sdmErr("ValidationException", "task %s is %s and can't be cancelled", t.id, t.state)
	}
	for _, e := range t.executions {
		if !sdmTerminal(e.state) {
			// Only a running unlock has moved its device to UNLOCKING.
			if d := st.devices[e.device]; d != nil && e.state == "IN_PROGRESS" && d.state == "UNLOCKING" && t.command == "unlock" {
				d.state, d.updated = "LOCKED", now
			}
			e.state, e.updated = "CANCELED", now
		}
	}
	t.state, t.updated, t.completed = "CANCELED", now, now
	return map[string]any{"taskId": t.id}, nil
}

func (e *sdmExecution) out(taskID string) map[string]any {
	out := map[string]any{"executionId": e.id, "taskId": taskID, "managedDeviceId": e.device, "state": e.state, "lastUpdatedAt": budEpoch(e.updated)}
	if !e.started.IsZero() {
		out["startedAt"] = budEpoch(e.started)
	}
	return out
}

func (st *sdmStore) describeExecution(devices []sdmDevice, in map[string]any, now time.Time) (map[string]any, *budErr) {
	t, err := st.task(budStr(in, "taskId"))
	if err != nil {
		return nil, err
	}
	dev := budStr(in, "managedDeviceId")
	e := t.executions[dev]
	if e == nil {
		return nil, sdmNotFound("Execution of task "+t.id+" on device", dev)
	}
	st.step(t, e, devices, now)
	return e.out(t.id), nil
}

func (st *sdmStore) listExecutions(_ []sdmDevice, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	t, err := st.task(budStr(in, "taskId"))
	if err != nil {
		return nil, err
	}
	state := budStr(in, "state")
	var match []any
	for _, d := range t.targets {
		e := t.executions[d]
		if state == "" || e.state == state {
			full := e.out(t.id)
			match = append(match, map[string]any{"executionId": e.id, "taskId": t.id, "managedDeviceId": e.device, "state": full["state"]})
		}
	}
	start, end, next, err := sdmPage(in, len(match))
	if err != nil {
		return nil, err
	}
	return map[string]any{"executions": append([]any{}, match[start:end]...), "nextToken": next}, nil
}

// ---- tags ----

// tagsOf resolves a task or managed device ARN to its tag map.
func (st *sdmStore) tagsOf(devices []sdmDevice, arn string, now time.Time) (*map[string]string, *budErr) {
	rest, ok := strings.CutPrefix(arn, sdmARNBase)
	if ok {
		kind, id, _ := strings.Cut(rest, "/")
		switch kind {
		case "task":
			if t := st.tasks[id]; t != nil {
				return &t.tags, nil
			}
		case "managed-device":
			if sdmFind(devices, id) != nil {
				d, err := st.state(id, now)
				if err != nil {
					return nil, err
				}
				return &d.tags, nil
			}
		}
	}
	return nil, sdmNotFound("Resource", arn)
}

func (st *sdmStore) tagResource(devices []sdmDevice, in map[string]any, now time.Time) (map[string]any, *budErr) {
	add, err := sdmTagsIn(in)
	if err != nil {
		return nil, err
	}
	tags, err := st.tagsOf(devices, budStr(in, "resourceArn"), now)
	if err != nil {
		return nil, err
	}
	merged, err := budMergeTags(*tags, add, sdmMaxTags, "ValidationException")
	if err != nil {
		return nil, err
	}
	*tags = merged
	return map[string]any{}, nil
}

func (st *sdmStore) untagResource(devices []sdmDevice, in map[string]any, now time.Time) (map[string]any, *budErr) {
	tags, err := st.tagsOf(devices, budStr(in, "resourceArn"), now)
	if err != nil {
		return nil, err
	}
	*tags = budDropTags(*tags, budStrings(in, "tagKeys"))
	return map[string]any{}, nil
}

func (st *sdmStore) listTags(devices []sdmDevice, in map[string]any, now time.Time) (map[string]any, *budErr) {
	arn := budStr(in, "resourceArn")
	if id, ok := strings.CutPrefix(arn, sdmARNBase+"managed-device/"); ok && sdmFind(devices, id) != nil {
		return map[string]any{"tags": sdmTagsOut(st.peek(id, now).tags)}, nil // reading doesn't track the device
	}
	tags, err := st.tagsOf(devices, arn, now)
	if err != nil {
		return nil, err
	}
	return map[string]any{"tags": sdmTagsOut(*tags)}, nil
}
