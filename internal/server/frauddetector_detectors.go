package server

import (
	"sort"
	"strconv"
)

func (st *frdStore) putDetector(in map[string]any) (map[string]any, *fcErr) {
	id, et := fcStr(in, "detectorId"), fcStr(in, "eventTypeName")
	if st.coll("event-type")[et] == nil {
		return nil, frdNotFound("event type %s not found", et)
	}
	if old := st.coll("detector")[id]; old != nil {
		if fcStr(old, "eventTypeName") != et && len(st.versions[id]) > 0 {
			return nil, frdValidation("a detector with versions can't change its event type")
		}
		rec := frdCopy(old)
		rec["eventTypeName"], rec["lastUpdatedTime"] = et, frdNow()
		if d, ok := in["description"]; ok {
			rec["description"] = d
		}
		return nil, st.replace("detector", id, rec)
	}
	return nil, st.add("detector", id, "detector/"+id, map[string]any{"detectorId": id, "eventTypeName": et, "description": in["description"]}, in["tags"])
}

func (st *frdStore) detector(id string) (map[string]any, *fcErr) {
	d := st.coll("detector")[id]
	if d == nil {
		return nil, frdNotFound("detector %s not found", id)
	}
	return d, nil
}

func (st *frdStore) deleteDetector(in map[string]any) (map[string]any, *fcErr) {
	id := fcStr(in, "detectorId")
	if st.coll("detector")[id] == nil {
		return nil, nil
	}
	if len(st.versions[id]) > 0 {
		return nil, frdConflict("detector %s still has versions", id)
	}
	for _, r := range st.rules {
		if fcStr(r, "detectorId") == id {
			return nil, frdConflict("detector %s still has rules", id)
		}
	}
	st.drop("detector", id)
	delete(st.nextVer, id)
	return nil, nil
}

// sortedVersions returns a detector's version ids in numeric order.
func (st *frdStore) sortedVersions(id string) []string {
	ids := frdSortedKeys(st.versions[id])
	sort.Slice(ids, func(i, j int) bool { a, _ := strconv.Atoi(ids[i]); b, _ := strconv.Atoi(ids[j]); return a < b })
	return ids
}

func (st *frdStore) describeDetector(in map[string]any) (map[string]any, *fcErr) {
	d, err := st.detector(fcStr(in, "detectorId"))
	if err != nil {
		return nil, err
	}
	id := fcStr(d, "detectorId")
	var items []any
	for _, v := range st.sortedVersions(id) {
		items = append(items, st.versions[id][v])
	}
	page, next, err := frdPage(items, in, "nextToken", 2500)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"detectorId": id, "arn": d["arn"], "detectorVersionSummaries": page}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

// checkVersionParts checks the rules, model versions and external models a
// detector version uses.
func (st *frdStore) checkVersionParts(id string, in map[string]any) *fcErr {
	for _, r := range fcObjList(in["rules"]) {
		if fcStr(r, "detectorId") != id {
			return frdValidation("rule %s belongs to detector %s, not %s", fcStr(r, "ruleId"), fcStr(r, "detectorId"), id)
		}
		if st.rules[id+"/"+fcStr(r, "ruleId")+"/"+fcStr(r, "ruleVersion")] == nil {
			return frdNotFound("rule %s version %s not found", fcStr(r, "ruleId"), fcStr(r, "ruleVersion"))
		}
	}
	for _, mv := range fcObjList(in["modelVersions"]) {
		rec := st.mvs[fcStr(mv, "modelType")+"/"+fcStr(mv, "modelId")+"/"+fcStr(mv, "modelVersionNumber")]
		if rec == nil {
			return frdNotFound("model %s version %s not found", fcStr(mv, "modelId"), fcStr(mv, "modelVersionNumber"))
		}
		if fcStr(rec, "status") != "ACTIVE" {
			return frdValidation("model %s version %s is not ACTIVE", fcStr(mv, "modelId"), fcStr(mv, "modelVersionNumber"))
		}
	}
	for _, ep := range fcStrs(in, "externalModelEndpoints") {
		if st.coll("external-model")[ep] == nil {
			return frdNotFound("external model %s not found", ep)
		}
	}
	return nil
}

func (st *frdStore) createDetectorVersion(in map[string]any) (map[string]any, *fcErr) {
	id := fcStr(in, "detectorId")
	if _, err := st.detector(id); err != nil {
		return nil, err
	}
	if err := st.checkVersionParts(id, in); err != nil {
		return nil, err
	}
	if len(st.versions[id]) >= frdMaxPerKind {
		return nil, frdValidation("at most %d versions per detector", frdMaxPerKind)
	}
	st.nextVer[id]++
	ver := strconv.Itoa(st.nextVer[id])
	rec := map[string]any{"detectorId": id, "detectorVersionId": ver, "status": "DRAFT", "ruleExecutionMode": "FIRST_MATCHED",
		"rules": in["rules"], "modelVersions": in["modelVersions"], "externalModelEndpoints": in["externalModelEndpoints"], "description": in["description"]}
	if m := fcStr(in, "ruleExecutionMode"); m != "" {
		rec["ruleExecutionMode"] = m
	}
	t, err := frdTagMap(nil, in["tags"])
	if err != nil {
		return nil, err
	}
	if err := st.charge("detector-version/"+id+"/"+ver, rec); err != nil {
		return nil, err
	}
	now := frdNow()
	rec["arn"], rec["createdTime"], rec["lastUpdatedTime"] = frdArn("detector-version/"+id+"/"+ver), now, now
	if st.versions[id] == nil {
		st.versions[id] = map[string]map[string]any{}
	}
	st.versions[id][ver] = rec
	st.tags[fcStr(rec, "arn")] = t
	return map[string]any{"detectorId": id, "detectorVersionId": ver, "status": "DRAFT"}, nil
}

func (st *frdStore) version(id, ver string) (map[string]any, *fcErr) {
	v := st.versions[id][ver]
	if v == nil {
		return nil, frdNotFound("detector %s version %s not found", id, ver)
	}
	return v, nil
}

func (st *frdStore) updateDetectorVersion(in map[string]any) (map[string]any, *fcErr) {
	id, ver := fcStr(in, "detectorId"), fcStr(in, "detectorVersionId")
	v, err := st.version(id, ver)
	if err != nil {
		return nil, err
	}
	if fcStr(v, "status") != "DRAFT" {
		return nil, frdValidation("only DRAFT detector versions can be updated")
	}
	if err := st.checkVersionParts(id, in); err != nil {
		return nil, err
	}
	rec := frdCopy(v)
	for _, key := range []string{"rules", "modelVersions", "externalModelEndpoints", "description", "ruleExecutionMode"} {
		if val, ok := in[key]; ok {
			rec[key] = val
		}
	}
	rec["lastUpdatedTime"] = frdNow()
	if err := st.charge("detector-version/"+id+"/"+ver, rec); err != nil {
		return nil, err
	}
	st.versions[id][ver] = rec
	return nil, nil
}

func (st *frdStore) updateDetectorVersionMetadata(in map[string]any) (map[string]any, *fcErr) {
	id, ver := fcStr(in, "detectorId"), fcStr(in, "detectorVersionId")
	v, err := st.version(id, ver)
	if err != nil {
		return nil, err
	}
	rec := frdCopy(v)
	rec["description"], rec["lastUpdatedTime"] = in["description"], frdNow()
	if err := st.charge("detector-version/"+id+"/"+ver, rec); err != nil {
		return nil, err
	}
	st.versions[id][ver] = rec
	return nil, nil
}

func (st *frdStore) updateDetectorVersionStatus(in map[string]any) (map[string]any, *fcErr) {
	id, ver, to := fcStr(in, "detectorId"), fcStr(in, "detectorVersionId"), fcStr(in, "status")
	v, err := st.version(id, ver)
	if err != nil {
		return nil, err
	}
	from := fcStr(v, "status")
	allowed := map[string]string{"DRAFT": "ACTIVE", "ACTIVE": "INACTIVE", "INACTIVE": "ACTIVE"}
	if allowed[from] != to {
		return nil, frdValidation("a %s detector version can't become %s", from, to)
	}
	now := frdNow()
	if to == "ACTIVE" {
		for other, ov := range st.versions[id] {
			if fcStr(ov, "status") == "ACTIVE" {
				demoted := frdCopy(ov)
				demoted["status"], demoted["lastUpdatedTime"] = "INACTIVE", now
				st.versions[id][other] = demoted
			}
		}
	}
	rec := frdCopy(v)
	rec["status"], rec["lastUpdatedTime"] = to, now
	st.versions[id][ver] = rec
	return nil, nil
}

func (st *frdStore) deleteDetectorVersion(in map[string]any) (map[string]any, *fcErr) {
	id, ver := fcStr(in, "detectorId"), fcStr(in, "detectorVersionId")
	v, err := st.version(id, ver)
	if err != nil {
		return nil, err
	}
	if fcStr(v, "status") == "ACTIVE" {
		return nil, frdConflict("an ACTIVE detector version can't be deleted")
	}
	delete(st.tags, fcStr(v, "arn"))
	st.refund("detector-version/" + id + "/" + ver)
	delete(st.versions[id], ver)
	if len(st.versions[id]) == 0 {
		delete(st.versions, id)
	}
	return nil, nil
}

// activeVersion returns the detector's ACTIVE version.
func (st *frdStore) activeVersion(id string) (map[string]any, *fcErr) {
	for _, ver := range st.sortedVersions(id) {
		if v := st.versions[id][ver]; fcStr(v, "status") == "ACTIVE" {
			return v, nil
		}
	}
	return nil, frdNotFound("detector %s has no ACTIVE version", id)
}

// --- rules ---

// checkRule parses the expression and checks its variables and outcomes.
func (st *frdStore) checkRule(in map[string]any) *fcErr {
	_, vars, err := frdRunExpr(fcStr(in, "expression"), nil)
	if err != nil {
		return frdValidation("invalid rule expression: %v", err)
	}
	for _, v := range vars {
		if st.coll("variable")[v] == nil {
			return frdValidation("rule expression uses unknown variable $%s", v)
		}
	}
	for _, o := range fcStrs(in, "outcomes") {
		if st.coll("outcome")[o] == nil {
			return frdNotFound("outcome %s not found", o)
		}
	}
	return nil
}

func (st *frdStore) putRule(det, ruleID, ver string, in map[string]any, description any) (map[string]any, *fcErr) {
	if len(st.rules) >= frdMaxPerKind*10 {
		return nil, frdValidation("too many rule versions")
	}
	t, err := frdTagMap(nil, in["tags"])
	if err != nil {
		return nil, err
	}
	now := frdNow()
	key := det + "/" + ruleID + "/" + ver
	rec := map[string]any{"detectorId": det, "ruleId": ruleID, "ruleVersion": ver, "description": description,
		"expression": in["expression"], "language": in["language"], "outcomes": in["outcomes"],
		"arn": frdArn("rule/" + key), "createdTime": now, "lastUpdatedTime": now}
	if err := st.charge("rule/"+key, rec); err != nil {
		return nil, err
	}
	st.rules[key] = rec
	st.tags[fcStr(rec, "arn")] = t
	return map[string]any{"rule": map[string]any{"detectorId": det, "ruleId": ruleID, "ruleVersion": ver}}, nil
}

func (st *frdStore) createRule(in map[string]any) (map[string]any, *fcErr) {
	det, id := fcStr(in, "detectorId"), fcStr(in, "ruleId")
	if _, err := st.detector(det); err != nil {
		return nil, err
	}
	for _, r := range st.rules {
		if fcStr(r, "detectorId") == det && fcStr(r, "ruleId") == id {
			return nil, frdValidation("rule %s already exists in detector %s", id, det)
		}
	}
	if err := st.checkRule(in); err != nil {
		return nil, err
	}
	return st.putRule(det, id, "1", in, in["description"])
}

func (st *frdStore) ruleOf(ref map[string]any) (string, map[string]any, *fcErr) {
	key := fcStr(ref, "detectorId") + "/" + fcStr(ref, "ruleId") + "/" + fcStr(ref, "ruleVersion")
	r := st.rules[key]
	if r == nil {
		return "", nil, frdNotFound("rule %s version %s not found", fcStr(ref, "ruleId"), fcStr(ref, "ruleVersion"))
	}
	return key, r, nil
}

func (st *frdStore) updateRuleVersion(in map[string]any) (map[string]any, *fcErr) {
	ref := fcObj(in, "rule")
	if _, _, err := st.ruleOf(ref); err != nil {
		return nil, err
	}
	if err := st.checkRule(in); err != nil {
		return nil, err
	}
	det, id := fcStr(ref, "detectorId"), fcStr(ref, "ruleId")
	latest := 0
	for _, r := range st.rules {
		if fcStr(r, "detectorId") == det && fcStr(r, "ruleId") == id {
			n, _ := strconv.Atoi(fcStr(r, "ruleVersion"))
			latest = max(latest, n)
		}
	}
	return st.putRule(det, id, strconv.Itoa(latest+1), in, in["description"])
}

func (st *frdStore) updateRuleMetadata(in map[string]any) (map[string]any, *fcErr) {
	key, r, err := st.ruleOf(fcObj(in, "rule"))
	if err != nil {
		return nil, err
	}
	rec := frdCopy(r)
	rec["description"], rec["lastUpdatedTime"] = in["description"], frdNow()
	if err := st.charge("rule/"+key, rec); err != nil {
		return nil, err
	}
	st.rules[key] = rec
	return nil, nil
}

func (st *frdStore) deleteRule(in map[string]any) (map[string]any, *fcErr) {
	ref := fcObj(in, "rule")
	key, r, err := st.ruleOf(ref)
	if err != nil {
		return nil, nil // deleting a missing rule succeeds
	}
	for ver, v := range st.versions[fcStr(ref, "detectorId")] {
		for _, used := range fcObjList(v["rules"]) {
			if fcStr(used, "ruleId") == fcStr(ref, "ruleId") && fcStr(used, "ruleVersion") == fcStr(ref, "ruleVersion") {
				return nil, frdConflict("rule %s is used by detector version %s", fcStr(ref, "ruleId"), ver)
			}
		}
	}
	delete(st.tags, fcStr(r, "arn"))
	st.refund("rule/" + key)
	delete(st.rules, key)
	return nil, nil
}

func (st *frdStore) getRules(in map[string]any) (map[string]any, *fcErr) {
	det, id, ver := fcStr(in, "detectorId"), fcStr(in, "ruleId"), fcStr(in, "ruleVersion")
	if _, err := st.detector(det); err != nil {
		return nil, err
	}
	if ver != "" && id == "" {
		return nil, frdValidation("ruleVersion needs a ruleId")
	}
	var items []any
	for _, key := range frdSortedKeys(st.rules) {
		r := st.rules[key]
		if fcStr(r, "detectorId") == det && (id == "" || fcStr(r, "ruleId") == id) && (ver == "" || fcStr(r, "ruleVersion") == ver) {
			items = append(items, r)
		}
	}
	if id != "" && len(items) == 0 {
		return nil, frdNotFound("rule %s not found", id)
	}
	page, next, err := frdPage(items, in, "nextToken", 100)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"ruleDetails": page}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

func frdDetectorHandlers() map[string]frdHandler {
	return map[string]frdHandler{
		"PutDetector":      (*frdStore).putDetector,
		"DeleteDetector":   (*frdStore).deleteDetector,
		"DescribeDetector": (*frdStore).describeDetector,
		"GetDetectors": func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			return st.getNamed("detector", fcStr(in, "detectorId"), in, "detectors", 10)
		},
		"CreateDetectorVersion":         (*frdStore).createDetectorVersion,
		"UpdateDetectorVersion":         (*frdStore).updateDetectorVersion,
		"UpdateDetectorVersionMetadata": (*frdStore).updateDetectorVersionMetadata,
		"UpdateDetectorVersionStatus":   (*frdStore).updateDetectorVersionStatus,
		"DeleteDetectorVersion":         (*frdStore).deleteDetectorVersion,
		"GetDetectorVersion": func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			return st.version(fcStr(in, "detectorId"), fcStr(in, "detectorVersionId"))
		},
		"CreateRule":         (*frdStore).createRule,
		"UpdateRuleVersion":  (*frdStore).updateRuleVersion,
		"UpdateRuleMetadata": (*frdStore).updateRuleMetadata,
		"DeleteRule":         (*frdStore).deleteRule,
		"GetRules":           (*frdStore).getRules,
	}
}
