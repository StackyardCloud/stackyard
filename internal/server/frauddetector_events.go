package server

import (
	"strconv"
	"strings"
	"time"
)

func frdParseTime(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, s)
	return t, err == nil
}

// checkEvent checks event variables, entities and the timestamp against the
// event type.
func (st *frdStore) checkEvent(in map[string]any) (map[string]any, *fcErr) {
	et := st.coll("event-type")[fcStr(in, "eventTypeName")]
	if et == nil {
		return nil, frdNotFound("event type %s not found", fcStr(in, "eventTypeName"))
	}
	for name := range fcObj(in, "eventVariables") {
		if !fcContains(fcStrs(et, "eventVariables"), name) {
			return nil, frdValidation("%s is not a variable of event type %s", name, fcStr(et, "name"))
		}
	}
	for _, e := range fcObjList(in["entities"]) {
		if t := fcStr(e, "entityType"); !fcContains(fcStrs(et, "entityTypes"), t) {
			return nil, frdValidation("%s is not an entity type of event type %s", t, fcStr(et, "name"))
		}
	}
	if _, ok := frdParseTime(fcStr(in, "eventTimestamp")); !ok {
		return nil, frdValidation("eventTimestamp must be an ISO 8601 UTC timestamp")
	}
	return et, nil
}

func (st *frdStore) storeEvent(in map[string]any) *fcErr {
	key := fcStr(in, "eventTypeName") + "/" + fcStr(in, "eventId")
	if st.events[key] == nil && len(st.events) >= frdMaxEvents {
		return frdValidation("at most %d stored events", frdMaxEvents)
	}
	ev := map[string]any{"eventId": in["eventId"], "eventTypeName": in["eventTypeName"], "eventTimestamp": in["eventTimestamp"],
		"eventVariables": in["eventVariables"], "entities": in["entities"]}
	if l := fcStr(in, "assignedLabel"); l != "" {
		ev["currentLabel"], ev["labelTimestamp"] = l, in["labelTimestamp"]
	}
	size := 0
	for k, v := range fcObj(in, "eventVariables") {
		s, _ := v.(string)
		size += len(k) + len(s)
	}
	ev["_size"] = float64(size)
	if err := st.charge("event/"+key, ev); err != nil {
		return err
	}
	st.events[key] = ev
	return nil
}

func (st *frdStore) checkLabel(et map[string]any, label, ts string) *fcErr {
	if label == "" {
		return nil
	}
	if !fcContains(fcStrs(et, "labels"), label) {
		return frdValidation("%s is not a label of event type %s", label, fcStr(et, "name"))
	}
	if _, ok := frdParseTime(ts); !ok {
		return frdValidation("a label needs an ISO 8601 labelTimestamp")
	}
	return nil
}

func (st *frdStore) sendEvent(in map[string]any) (map[string]any, *fcErr) {
	et, err := st.checkEvent(in)
	if err != nil {
		return nil, err
	}
	if fcStr(et, "eventIngestion") != "ENABLED" {
		return nil, frdValidation("event ingestion is disabled for event type %s", fcStr(et, "name"))
	}
	if err := st.checkLabel(et, fcStr(in, "assignedLabel"), fcStr(in, "labelTimestamp")); err != nil {
		return nil, err
	}
	if st.events[fcStr(in, "eventTypeName")+"/"+fcStr(in, "eventId")] != nil {
		return nil, frdConflict("event %s already exists", fcStr(in, "eventId"))
	}
	return nil, st.storeEvent(in)
}

func (st *frdStore) event(in map[string]any) (string, map[string]any, *fcErr) {
	key := fcStr(in, "eventTypeName") + "/" + fcStr(in, "eventId")
	ev := st.events[key]
	if ev == nil {
		return "", nil, frdNotFound("event %s not found", fcStr(in, "eventId"))
	}
	return key, ev, nil
}

func (st *frdStore) updateEventLabel(in map[string]any) (map[string]any, *fcErr) {
	_, ev, err := st.event(in)
	if err != nil {
		return nil, err
	}
	et := st.coll("event-type")[fcStr(in, "eventTypeName")]
	if err := st.checkLabel(et, fcStr(in, "assignedLabel"), fcStr(in, "labelTimestamp")); err != nil {
		return nil, err
	}
	ev["currentLabel"], ev["labelTimestamp"] = in["assignedLabel"], in["labelTimestamp"]
	return nil, nil
}

func (st *frdStore) deleteEvent(in map[string]any) (map[string]any, *fcErr) {
	key, _, err := st.event(in)
	if err != nil {
		return nil, nil // deleting a missing event succeeds
	}
	delete(st.events, key)
	st.refund("event/" + key)
	if fcBool(in, "deleteAuditHistory") {
		kept := st.preds[:0:0]
		for _, p := range st.preds {
			if fcStr(p, "eventId") != fcStr(in, "eventId") || fcStr(p, "eventTypeName") != fcStr(in, "eventTypeName") {
				kept = append(kept, p)
			} else {
				st.refund(fcStr(p, "_key"))
			}
		}
		st.preds = kept
	}
	return nil, nil
}

func (st *frdStore) deleteEvents(eventType string) {
	for key := range st.events {
		if strings.HasPrefix(key, eventType+"/") {
			delete(st.events, key)
			st.refund("event/" + key)
		}
	}
}

func (st *frdStore) eventStats(eventType string) map[string]any {
	n, size := 0.0, 0.0
	var oldest, newest string
	for key, ev := range st.events {
		if !strings.HasPrefix(key, eventType+"/") {
			continue
		}
		n++
		s, _ := fcNum(ev, "_size")
		size += s
		ts := fcStr(ev, "eventTimestamp")
		if oldest == "" || ts < oldest {
			oldest = ts
		}
		if ts > newest {
			newest = ts
		}
	}
	out := map[string]any{"numberOfEvents": n, "eventDataSizeInBytes": size, "lastUpdatedTime": frdNow()}
	if n > 0 {
		out["leastRecentEvent"], out["mostRecentEvent"] = oldest, newest
	}
	return out
}

// --- predictions ---

func (st *frdStore) getEventPrediction(in map[string]any) (map[string]any, *fcErr) {
	det, err := st.detector(fcStr(in, "detectorId"))
	if err != nil {
		return nil, err
	}
	var ver map[string]any
	if id := fcStr(in, "detectorVersionId"); id != "" {
		ver, err = st.version(fcStr(det, "detectorId"), id)
	} else {
		ver, err = st.activeVersion(fcStr(det, "detectorId"))
	}
	if err != nil {
		return nil, err
	}
	if fcStr(in, "eventTypeName") != fcStr(det, "eventTypeName") {
		return nil, frdValidation("detector %s scores event type %s", fcStr(det, "detectorId"), fcStr(det, "eventTypeName"))
	}
	et, err := st.checkEvent(in)
	if err != nil {
		return nil, err
	}
	vars := fcObj(in, "eventVariables")
	values := map[string]string{}
	for k, v := range vars {
		values[k], _ = v.(string)
	}
	// Model scores are deterministic per model version and event variables.
	keys := frdSortedKeys(values)
	var modelScores, evaluatedModels []any
	for _, mv := range fcObjList(ver["modelVersions"]) {
		parts := []string{fcStr(mv, "modelId"), fcStr(mv, "modelVersionNumber")}
		for _, k := range keys {
			parts = append(parts, k+"="+values[k])
		}
		score := float64(fcSeed(parts...) % 1000)
		name := fcStr(mv, "modelId") + "_insightscore"
		values[name] = strconv.FormatFloat(score, 'f', -1, 64)
		modelScores = append(modelScores, map[string]any{"modelVersion": mv, "scores": map[string]any{name: score}})
		evaluatedModels = append(evaluatedModels, map[string]any{"modelId": mv["modelId"], "modelType": mv["modelType"],
			"modelVersion": mv["modelVersionNumber"], "evaluations": []any{map[string]any{"outputVariableName": name, "evaluationScore": values[name]}}})
	}
	var externalOutputs, evaluatedExternal []any
	for _, ep := range fcStrs(ver, "externalModelEndpoints") {
		em := st.coll("external-model")[ep]
		outputs := map[string]any{}
		for _, v := range fcObj(fcObj(em, "outputConfiguration"), "jsonKeyToVariableMap") {
			name, _ := v.(string)
			outputs[name] = strconv.FormatUint(fcSeed(ep, name, strings.Join(keys, ","))%1000, 10)
			values[name] = outputs[name].(string)
		}
		externalOutputs = append(externalOutputs, map[string]any{"externalModel": map[string]any{"modelEndpoint": ep, "modelSource": em["modelSource"]}, "outputs": outputs})
		evaluatedExternal = append(evaluatedExternal, map[string]any{"modelEndpoint": ep, "outputVariables": outputs, "useEventVariables": true})
	}
	lookup := func(name string) string {
		if v, ok := values[name]; ok {
			return v
		}
		return fcStr(st.coll("variable")[name], "defaultValue")
	}
	ruleResults, evaluated, outcomes := []any{}, []any{}, []string{}
	matchedOne := false
	for _, ref := range fcObjList(ver["rules"]) {
		_, rule, rerr := st.ruleOf(ref)
		if rerr != nil {
			continue
		}
		if matchedOne && fcStr(ver, "ruleExecutionMode") != "ALL_MATCHED" {
			evaluated = append(evaluated, map[string]any{"ruleId": ref["ruleId"], "ruleVersion": ref["ruleVersion"], "evaluated": false})
			continue
		}
		matched, _, _ := frdRunExpr(fcStr(rule, "expression"), lookup)
		evaluated = append(evaluated, map[string]any{"ruleId": ref["ruleId"], "ruleVersion": ref["ruleVersion"], "expression": rule["expression"],
			"evaluated": true, "matched": matched, "outcomes": rule["outcomes"]})
		if matched {
			matchedOne = true
			ruleResults = append(ruleResults, map[string]any{"ruleId": ref["ruleId"], "outcomes": rule["outcomes"]})
			outcomes = append(outcomes, fcStrs(rule, "outcomes")...)
		}
	}
	if fcStr(et, "eventIngestion") == "ENABLED" {
		_ = st.storeEvent(in) // ponytail: a full event store skips storing, the prediction still succeeds
	}
	st.recordPrediction(in, det, ver, evaluated, evaluatedModels, evaluatedExternal, outcomes)
	return map[string]any{"modelScores": modelScores, "ruleResults": ruleResults, "externalModelOutputs": externalOutputs}, nil
}

func (st *frdStore) recordPrediction(in, det, ver map[string]any, rules, models, external []any, outcomes []string) {
	vars := []any{}
	for _, k := range frdSortedKeys(fcObj(in, "eventVariables")) {
		vars = append(vars, map[string]any{"name": k, "value": fcObj(in, "eventVariables")[k], "source": "EVENT"})
	}
	p := map[string]any{"eventId": in["eventId"], "eventTypeName": in["eventTypeName"], "eventTimestamp": in["eventTimestamp"],
		"detectorId": det["detectorId"], "detectorVersionId": ver["detectorVersionId"], "detectorVersionStatus": ver["status"],
		"ruleExecutionMode": ver["ruleExecutionMode"], "predictionTimestamp": frdNow(), "eventVariables": vars,
		"rules": rules, "evaluatedModelVersions": models, "evaluatedExternalModels": external, "outcomes": fcAnyList(outcomes)}
	if ents := fcObjList(in["entities"]); len(ents) > 0 {
		p["entityId"], p["entityType"] = ents[0]["entityId"], ents[0]["entityType"]
	}
	st.predSeq++
	key := "prediction/" + strconv.Itoa(st.predSeq)
	if st.charge(key, p) != nil {
		return // ponytail: past the byte bound the prediction is answered but not remembered
	}
	p["_key"] = key
	st.preds = append(st.preds, p)
	if len(st.preds) > frdMaxPreds {
		for _, old := range st.preds[:len(st.preds)-frdMaxPreds] {
			st.refund(fcStr(old, "_key"))
		}
		st.preds = append(st.preds[:0:0], st.preds[len(st.preds)-frdMaxPreds:]...)
	}
}

func (st *frdStore) getEventPredictionMetadata(in map[string]any) (map[string]any, *fcErr) {
	for _, p := range st.preds {
		match := true
		for _, key := range []string{"eventId", "eventTypeName", "detectorId", "detectorVersionId", "predictionTimestamp"} {
			if fcStr(p, key) != fcStr(in, key) {
				match = false
			}
		}
		if match {
			return p, nil
		}
	}
	return nil, frdNotFound("no prediction for event %s at %s", fcStr(in, "eventId"), fcStr(in, "predictionTimestamp"))
}

func (st *frdStore) listEventPredictions(in map[string]any) (map[string]any, *fcErr) {
	filters := map[string]string{"eventId": "eventId", "eventType": "eventTypeName", "detectorId": "detectorId", "detectorVersionId": "detectorVersionId"}
	rng := fcObj(in, "predictionTimeRange")
	var from, to time.Time
	if rng != nil {
		var ok1, ok2 bool
		from, ok1 = frdParseTime(fcStr(rng, "startTime"))
		to, ok2 = frdParseTime(fcStr(rng, "endTime"))
		if !ok1 || !ok2 || to.Before(from) {
			return nil, frdValidation("predictionTimeRange needs ISO 8601 startTime <= endTime")
		}
	}
	var items []any
	for i := len(st.preds) - 1; i >= 0; i-- {
		p := st.preds[i]
		keep := true
		for inKey, pKey := range filters {
			if f := fcObj(in, inKey); f != nil && fcStr(f, "value") != fcStr(p, pKey) {
				keep = false
			}
		}
		if t, _ := frdParseTime(fcStr(p, "predictionTimestamp")); rng != nil && (t.Before(from) || t.After(to)) {
			keep = false
		}
		if keep {
			items = append(items, p)
		}
	}
	page, next, err := frdPage(items, in, "nextToken", 100)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"eventPredictionSummaries": page}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

func frdEventHandlers() map[string]frdHandler {
	return map[string]frdHandler{
		"SendEvent":        (*frdStore).sendEvent,
		"UpdateEventLabel": (*frdStore).updateEventLabel,
		"DeleteEvent":      (*frdStore).deleteEvent,
		"GetEvent": func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			_, ev, err := st.event(in)
			if err != nil {
				return nil, err
			}
			return map[string]any{"event": ev}, nil
		},
		"DeleteEventsByEventType": func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			name := fcStr(in, "eventTypeName")
			if st.coll("event-type")[name] == nil {
				return nil, frdNotFound("event type %s not found", name)
			}
			st.deleteEvents(name)
			st.deletes[name] = "COMPLETE"
			return map[string]any{"eventTypeName": name, "eventsDeletionStatus": "IN_PROGRESS"}, nil
		},
		"GetDeleteEventsByEventTypeStatus": func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			name := fcStr(in, "eventTypeName")
			status, ok := st.deletes[name]
			if !ok {
				return nil, frdNotFound("no event deletion for event type %s", name)
			}
			return map[string]any{"eventTypeName": name, "eventsDeletionStatus": status}, nil
		},
		"GetEventPrediction":         (*frdStore).getEventPrediction,
		"GetEventPredictionMetadata": (*frdStore).getEventPredictionMetadata,
		"ListEventPredictions":       (*frdStore).listEventPredictions,
	}
}
