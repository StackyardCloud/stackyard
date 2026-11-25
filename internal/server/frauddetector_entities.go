package server

import (
	"regexp"
	"strconv"
	"strings"
)

var frdVarName = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

var frdDataTypes = []string{"STRING", "INTEGER", "FLOAT", "BOOLEAN", "DATETIME"}

// frdCheckValue reports whether v is a valid value of dataType.
func frdCheckValue(dataType, v string) bool {
	switch dataType {
	case "INTEGER":
		_, err := strconv.ParseInt(v, 10, 64)
		return err == nil
	case "FLOAT":
		_, err := strconv.ParseFloat(v, 64)
		return err == nil
	case "BOOLEAN":
		return v == "true" || v == "false"
	}
	return true
}

// newVariable checks and builds a variable record. dataSource and dataType
// are plain strings in BatchCreateVariable, so they are checked here.
func (st *frdStore) newVariable(m map[string]any) (map[string]any, *fcErr) {
	name, dt, ds := fcStr(m, "name"), fcStr(m, "dataType"), fcStr(m, "dataSource")
	switch {
	case !frdVarName.MatchString(name):
		return nil, frdValidation("variable names are 1-64 lowercase letters, digits and underscores")
	case st.coll("variable")[name] != nil:
		return nil, frdValidation("variable %s already exists", name)
	case !fcContains(frdDataTypes, dt):
		return nil, frdValidation("dataType must be one of %s", strings.Join(frdDataTypes, ", "))
	case ds != "EVENT" && ds != "EXTERNAL_MODEL_SCORE":
		return nil, frdValidation("dataSource must be EVENT or EXTERNAL_MODEL_SCORE")
	case !frdCheckValue(dt, fcStr(m, "defaultValue")):
		return nil, frdValidation("defaultValue %q is not a valid %s", fcStr(m, "defaultValue"), dt)
	}
	rec := map[string]any{"name": name, "dataType": dt, "dataSource": ds, "defaultValue": fcStr(m, "defaultValue")}
	for _, key := range []string{"description", "variableType"} {
		if v := fcStr(m, key); v != "" {
			rec[key] = v
		}
	}
	return rec, nil
}

func (st *frdStore) createVariable(in map[string]any) (map[string]any, *fcErr) {
	rec, err := st.newVariable(in)
	if err != nil {
		return nil, err
	}
	return nil, st.add("variable", rec["name"].(string), "variable/"+rec["name"].(string), rec, in["tags"])
}

func (st *frdStore) batchCreateVariable(in map[string]any) (map[string]any, *fcErr) {
	errs := []any{}
	for _, e := range fcObjList(in["variableEntries"]) {
		rec, err := st.newVariable(e)
		if err == nil {
			err = st.add("variable", rec["name"].(string), "variable/"+rec["name"].(string), rec, in["tags"])
		}
		if err != nil {
			errs = append(errs, map[string]any{"name": fcStr(e, "name"), "code": 400.0, "message": err.msg})
		}
	}
	return map[string]any{"errors": errs}, nil
}

func (st *frdStore) batchGetVariable(in map[string]any) (map[string]any, *fcErr) {
	vars, errs := []any{}, []any{}
	for _, name := range fcStrs(in, "names") {
		if rec := st.coll("variable")[name]; rec != nil {
			vars = append(vars, rec)
		} else {
			errs = append(errs, map[string]any{"name": name, "code": 404.0, "message": "variable not found"})
		}
	}
	return map[string]any{"variables": vars, "errors": errs}, nil
}

func (st *frdStore) updateVariable(in map[string]any) (map[string]any, *fcErr) {
	name := fcStr(in, "name")
	old := st.coll("variable")[name]
	if old == nil {
		return nil, frdNotFound("variable %s not found", name)
	}
	rec := frdCopy(old)
	if v, ok := in["defaultValue"].(string); ok {
		if !frdCheckValue(fcStr(rec, "dataType"), v) {
			return nil, frdValidation("defaultValue %q is not a valid %s", v, fcStr(rec, "dataType"))
		}
		rec["defaultValue"] = v
	}
	for _, key := range []string{"description", "variableType"} {
		if v, ok := in[key]; ok {
			rec[key] = v
		}
	}
	rec["lastUpdatedTime"] = frdNow()
	st.coll("variable")[name] = rec
	return nil, nil
}

func (st *frdStore) deleteVariable(in map[string]any) (map[string]any, *fcErr) {
	name := fcStr(in, "name")
	rec := st.coll("variable")[name]
	if rec == nil {
		return nil, nil // deleting a missing variable succeeds, as in AWS
	}
	if fcStr(rec, "dataSource") == "MODEL_SCORE" {
		return nil, frdValidation("model score variables are deleted with their model")
	}
	for _, et := range st.coll("event-type") {
		if fcContains(fcStrs(et, "eventVariables"), name) {
			return nil, frdConflict("variable %s is used by event type %s", name, fcStr(et, "name"))
		}
	}
	st.drop("variable", name)
	return nil, nil
}

// deleteSimple deletes an entity type, label or outcome unless it is used.
func (st *frdStore) deleteSimple(kind, name string) *fcErr {
	if st.coll(kind)[name] == nil {
		return nil
	}
	member := map[string]string{"entity-type": "entityTypes", "label": "labels"}[kind]
	for _, et := range st.coll("event-type") {
		if member != "" && fcContains(fcStrs(et, member), name) {
			return frdConflict("%s %s is used by event type %s", kind, name, fcStr(et, "name"))
		}
	}
	if kind == "outcome" {
		for _, r := range st.rules {
			if fcContains(fcStrs(r, "outcomes"), name) {
				return frdConflict("outcome %s is used by rule %s", name, fcStr(r, "ruleId"))
			}
		}
	}
	st.drop(kind, name)
	return nil
}

func (st *frdStore) putEventType(in map[string]any) (map[string]any, *fcErr) {
	name := fcStr(in, "name")
	for _, v := range fcStrs(in, "eventVariables") {
		rec := st.coll("variable")[v]
		if rec == nil {
			return nil, frdValidation("variable %s does not exist", v)
		}
		if fcStr(rec, "dataSource") != "EVENT" {
			return nil, frdValidation("event variable %s must have dataSource EVENT", v)
		}
	}
	for kind, key := range map[string]string{"label": "labels", "entity-type": "entityTypes"} {
		for _, v := range fcStrs(in, key) {
			if st.coll(kind)[v] == nil {
				return nil, frdValidation("%s %s does not exist", kind, v)
			}
		}
	}
	rec := map[string]any{}
	for _, key := range []string{"name", "description", "eventVariables", "labels", "entityTypes", "eventIngestion", "eventOrchestration"} {
		if v, ok := in[key]; ok {
			rec[key] = v
		}
	}
	fcDefault(rec, "eventIngestion", "ENABLED")
	fcDefault(rec, "labels", []any{})
	if old := st.coll("event-type")[name]; old != nil {
		rec["arn"], rec["createdTime"], rec["lastUpdatedTime"] = old["arn"], old["createdTime"], frdNow()
		st.coll("event-type")[name] = rec
		return nil, nil
	}
	return nil, st.add("event-type", name, "event-type/"+name, rec, in["tags"])
}

func (st *frdStore) deleteEventType(in map[string]any) (map[string]any, *fcErr) {
	name := fcStr(in, "name")
	if st.coll("event-type")[name] == nil {
		return nil, nil
	}
	for _, d := range st.coll("detector") {
		if fcStr(d, "eventTypeName") == name {
			return nil, frdConflict("event type %s is used by detector %s", name, fcStr(d, "detectorId"))
		}
	}
	for _, m := range st.coll("model") {
		if fcStr(m, "eventTypeName") == name {
			return nil, frdConflict("event type %s is used by model %s", name, fcStr(m, "modelId"))
		}
	}
	st.deleteEvents(name)
	delete(st.deletes, name)
	st.drop("event-type", name)
	return nil, nil
}

func (st *frdStore) putExternalModel(in map[string]any) (map[string]any, *fcErr) {
	name := fcStr(in, "modelEndpoint")
	if et := fcStr(fcObj(in, "inputConfiguration"), "eventTypeName"); et != "" && st.coll("event-type")[et] == nil {
		return nil, frdNotFound("event type %s not found", et)
	}
	out := fcObj(in, "outputConfiguration")
	for _, key := range []string{"jsonKeyToVariableMap", "csvIndexToVariableMap"} {
		for _, v := range fcObj(out, key) {
			if s, _ := v.(string); st.coll("variable")[s] == nil {
				return nil, frdValidation("output variable %s does not exist", s)
			}
		}
	}
	rec := map[string]any{}
	for _, key := range []string{"modelEndpoint", "modelSource", "invokeModelEndpointRoleArn", "inputConfiguration", "outputConfiguration", "modelEndpointStatus"} {
		rec[key] = in[key]
	}
	if old := st.coll("external-model")[name]; old != nil {
		rec["arn"], rec["createdTime"], rec["lastUpdatedTime"] = old["arn"], old["createdTime"], frdNow()
		st.coll("external-model")[name] = rec
		return nil, nil
	}
	return nil, st.add("external-model", name, "external-model/"+name, rec, in["tags"])
}

func (st *frdStore) deleteExternalModel(in map[string]any) (map[string]any, *fcErr) {
	name := fcStr(in, "modelEndpoint")
	if st.coll("external-model")[name] == nil {
		return nil, frdNotFound("external model %s not found", name)
	}
	for det, vs := range st.versions {
		for id, v := range vs {
			if fcContains(fcStrs(v, "externalModelEndpoints"), name) {
				return nil, frdConflict("external model %s is used by detector %s version %s", name, det, id)
			}
		}
	}
	st.drop("external-model", name)
	return nil, nil
}

// --- lists ---

func (st *frdStore) allElements() int {
	n := 0
	for _, e := range st.elements {
		n += len(e)
	}
	return n
}

// setElements replaces a list's elements, deduplicated, within the bounds.
func (st *frdStore) setElements(name string, elems []string) *fcErr {
	seen := make(map[string]bool, len(elems))
	out := make([]string, 0, len(elems))
	for _, e := range elems {
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	if len(out) > frdMaxElements {
		return frdValidation("a list holds at most %d elements", frdMaxElements)
	}
	if st.allElements()-len(st.elements[name])+len(out) > frdMaxAllElems {
		return frdValidation("lists hold at most %d elements in total", frdMaxAllElems)
	}
	st.elements[name] = out
	return nil
}

func (st *frdStore) createList(in map[string]any) (map[string]any, *fcErr) {
	name := fcStr(in, "name")
	if st.coll("list")[name] != nil {
		return nil, frdValidation("list %s already exists", name)
	}
	if err := st.setElements(name, fcStrs(in, "elements")); err != nil {
		return nil, err
	}
	rec := map[string]any{"name": name, "description": in["description"], "variableType": in["variableType"]}
	if err := st.add("list", name, "list/"+name, rec, in["tags"]); err != nil {
		delete(st.elements, name)
		return nil, err
	}
	rec["updatedTime"] = rec["lastUpdatedTime"]
	return nil, nil
}

func (st *frdStore) updateList(in map[string]any) (map[string]any, *fcErr) {
	name := fcStr(in, "name")
	old := st.coll("list")[name]
	if old == nil {
		return nil, frdNotFound("list %s not found", name)
	}
	if _, ok := in["elements"]; ok {
		elems := fcStrs(in, "elements")
		switch fcStr(in, "updateMode") {
		case "APPEND":
			elems = append(append([]string{}, st.elements[name]...), elems...)
		case "REMOVE":
			drop := map[string]bool{}
			for _, e := range elems {
				drop[e] = true
			}
			elems = nil
			for _, e := range st.elements[name] {
				if !drop[e] {
					elems = append(elems, e)
				}
			}
		}
		if err := st.setElements(name, elems); err != nil {
			return nil, err
		}
	}
	rec := frdCopy(old)
	for _, key := range []string{"description", "variableType"} {
		if v, ok := in[key]; ok {
			rec[key] = v
		}
	}
	rec["updatedTime"] = frdNow()
	st.coll("list")[name] = rec
	return nil, nil
}

func (st *frdStore) getListElements(in map[string]any) (map[string]any, *fcErr) {
	name := fcStr(in, "name")
	if st.coll("list")[name] == nil {
		return nil, frdNotFound("list %s not found", name)
	}
	page, next, err := frdPage(fcAnyList(st.elements[name]), in, "nextToken", 5000)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"elements": page}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

func frdEntityHandlers() map[string]frdHandler {
	h := map[string]frdHandler{}
	simple := func(op, kind, listKey string, def int) {
		h["Put"+op] = func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			return nil, st.putSimple(kind, in)
		}
		h["Delete"+op] = func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			return nil, st.deleteSimple(kind, fcStr(in, "name"))
		}
		h["Get"+op+"s"] = func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			return st.getNamed(kind, fcStr(in, "name"), in, listKey, def)
		}
	}
	simple("EntityType", "entity-type", "entityTypes", 10)
	simple("Label", "label", "labels", 50)
	simple("Outcome", "outcome", "outcomes", 100)
	h["CreateVariable"] = (*frdStore).createVariable
	h["BatchCreateVariable"] = (*frdStore).batchCreateVariable
	h["BatchGetVariable"] = (*frdStore).batchGetVariable
	h["UpdateVariable"] = (*frdStore).updateVariable
	h["DeleteVariable"] = (*frdStore).deleteVariable
	h["GetVariables"] = func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
		return st.getNamed("variable", fcStr(in, "name"), in, "variables", 100)
	}
	h["PutEventType"] = (*frdStore).putEventType
	h["DeleteEventType"] = (*frdStore).deleteEventType
	h["GetEventTypes"] = func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
		return st.getNamed("event-type", fcStr(in, "name"), in, "eventTypes", 10)
	}
	h["PutExternalModel"] = (*frdStore).putExternalModel
	h["DeleteExternalModel"] = (*frdStore).deleteExternalModel
	h["GetExternalModels"] = func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
		return st.getNamed("external-model", fcStr(in, "modelEndpoint"), in, "externalModels", 10)
	}
	h["CreateList"] = (*frdStore).createList
	h["UpdateList"] = (*frdStore).updateList
	h["GetListElements"] = (*frdStore).getListElements
	h["GetListsMetadata"] = func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
		return st.getNamed("list", fcStr(in, "name"), in, "lists", 50)
	}
	h["DeleteList"] = func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
		name := fcStr(in, "name")
		if st.coll("list")[name] == nil {
			return nil, frdNotFound("list %s not found", name)
		}
		st.drop("list", name)
		delete(st.elements, name)
		return nil, nil
	}
	return h
}
