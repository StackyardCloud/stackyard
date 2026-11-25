package server

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// ponytail: models are not trained. A version reports TRAINING_IN_PROGRESS
// on its first read and TRAINING_COMPLETE after (or after fcSettle); batch
// jobs go IN_PROGRESS -> COMPLETE the same way without reading S3.

// frdSettle moves rec from status from to to once it has been read or has
// waited long enough, then marks it read.
func frdSettle(rec map[string]any, from, to string) {
	created, _ := rec["_created"].(time.Time)
	if fcStr(rec, "status") == from && (fcBool(rec, "_seen") || time.Since(created) > fcSettle) {
		rec["status"] = to
		rec["lastUpdatedTime"] = frdNow()
		if to == "COMPLETE" {
			rec["completionTime"] = frdNow()
		}
		if to == "TRAINING_COMPLETE" {
			auc := 0.8 + float64(fcSeed(fcStr(rec, "arn"))%190)/1000
			rec["trainingResult"] = map[string]any{"trainingMetrics": map[string]any{"auc": auc, "metricDataPoints": []any{
				map[string]any{"fpr": 0.01, "precision": 0.9, "tpr": auc - 0.2, "threshold": 900.0},
				map[string]any{"fpr": 0.05, "precision": 0.7, "tpr": auc - 0.1, "threshold": 500.0},
			}}}
		}
	}
	rec["_seen"] = true
}

func frdSettleJob(rec map[string]any) { frdSettle(rec, "IN_PROGRESS", "COMPLETE") }

func frdModelKey(in map[string]any) string {
	return fcStr(in, "modelType") + "/" + fcStr(in, "modelId")
}

func (st *frdStore) createModel(in map[string]any) (map[string]any, *fcErr) {
	key, id := frdModelKey(in), fcStr(in, "modelId")
	if st.coll("model")[key] != nil {
		return nil, frdValidation("model %s already exists", id)
	}
	if st.coll("event-type")[fcStr(in, "eventTypeName")] == nil {
		return nil, frdNotFound("event type %s not found", fcStr(in, "eventTypeName"))
	}
	rec := map[string]any{"modelId": id, "modelType": in["modelType"], "eventTypeName": in["eventTypeName"], "description": in["description"]}
	if err := st.add("model", key, "model/"+key, rec, in["tags"]); err != nil {
		return nil, err
	}
	score := id + "_insightscore"
	if st.coll("variable")[score] == nil {
		_ = st.add("variable", score, "variable/"+score, map[string]any{"name": score, "dataType": "FLOAT",
			"dataSource": "MODEL_SCORE", "defaultValue": "0.0"}, nil)
	}
	return nil, nil
}

func (st *frdStore) model(in map[string]any) (map[string]any, *fcErr) {
	m := st.coll("model")[frdModelKey(in)]
	if m == nil {
		return nil, frdNotFound("model %s of type %s not found", fcStr(in, "modelId"), fcStr(in, "modelType"))
	}
	return m, nil
}

func (st *frdStore) deleteModel(in map[string]any) (map[string]any, *fcErr) {
	key := frdModelKey(in)
	if st.coll("model")[key] == nil {
		return nil, nil
	}
	for k := range st.mvs {
		if strings.HasPrefix(k, key+"/") {
			return nil, frdConflict("model %s still has versions", fcStr(in, "modelId"))
		}
	}
	st.drop("model", key)
	st.drop("variable", fcStr(in, "modelId")+"_insightscore")
	return nil, nil
}

func (st *frdStore) getModels(in map[string]any) (map[string]any, *fcErr) {
	var items []any
	for _, key := range frdSortedKeys(st.coll("model")) {
		m := st.coll("model")[key]
		if (fcStr(in, "modelId") == "" || fcStr(m, "modelId") == fcStr(in, "modelId")) && (fcStr(in, "modelType") == "" || fcStr(m, "modelType") == fcStr(in, "modelType")) {
			items = append(items, m)
		}
	}
	page, next, err := frdPage(items, in, "nextToken", 10)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"models": page}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

// checkTraining checks a version's training source against its details and
// its variables against the model's event type.
func (st *frdStore) checkTraining(m map[string]any, in map[string]any) *fcErr {
	switch fcStr(in, "trainingDataSource") {
	case "EXTERNAL_EVENTS":
		if in["externalEventsDetail"] == nil {
			return frdValidation("EXTERNAL_EVENTS training needs externalEventsDetail")
		}
	case "INGESTED_EVENTS":
		if in["ingestedEventsDetail"] == nil {
			return frdValidation("INGESTED_EVENTS training needs ingestedEventsDetail")
		}
	}
	et := st.coll("event-type")[fcStr(m, "eventTypeName")]
	for _, v := range fcStrs(fcObj(in, "trainingDataSchema"), "modelVariables") {
		if !fcContains(fcStrs(et, "eventVariables"), v) {
			return frdValidation("model variable %s is not a variable of event type %s", v, fcStr(m, "eventTypeName"))
		}
	}
	return nil
}

func (st *frdStore) addModelVersion(m map[string]any, ver string, rec map[string]any, tags any) (map[string]any, *fcErr) {
	if len(st.mvs) >= frdMaxPerKind {
		return nil, frdValidation("at most %d model versions", frdMaxPerKind)
	}
	t, err := frdTagMap(nil, tags)
	if err != nil {
		return nil, err
	}
	key := fcStr(m, "modelType") + "/" + fcStr(m, "modelId") + "/" + ver
	now := frdNow()
	rec["modelId"], rec["modelType"], rec["modelVersionNumber"], rec["status"] = m["modelId"], m["modelType"], ver, "TRAINING_IN_PROGRESS"
	rec["arn"], rec["createdTime"], rec["lastUpdatedTime"], rec["_created"] = frdArn("model-version/"+key), now, now, time.Now()
	if err := st.charge("model-version/"+key, rec); err != nil {
		return nil, err
	}
	st.mvs[key] = rec
	st.tags[fcStr(rec, "arn")] = t
	return map[string]any{"modelId": m["modelId"], "modelType": m["modelType"], "modelVersionNumber": ver, "status": "TRAINING_IN_PROGRESS"}, nil
}

// versionsOf returns a model's version numbers, oldest first.
func (st *frdStore) versionsOf(m map[string]any) []string {
	prefix := fcStr(m, "modelType") + "/" + fcStr(m, "modelId") + "/"
	var out []string
	for k := range st.mvs {
		if v, ok := strings.CutPrefix(k, prefix); ok {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, _ := strconv.ParseFloat(out[i], 64)
		b, _ := strconv.ParseFloat(out[j], 64)
		return a < b
	})
	return out
}

func (st *frdStore) createModelVersion(in map[string]any) (map[string]any, *fcErr) {
	m, err := st.model(in)
	if err != nil {
		return nil, err
	}
	if err := st.checkTraining(m, in); err != nil {
		return nil, err
	}
	major := 0
	for _, v := range st.versionsOf(m) {
		n, _ := strconv.Atoi(strings.SplitN(v, ".", 2)[0])
		major = max(major, n)
	}
	rec := map[string]any{}
	for _, key := range []string{"trainingDataSource", "trainingDataSchema", "externalEventsDetail", "ingestedEventsDetail"} {
		rec[key] = in[key]
	}
	return st.addModelVersion(m, strconv.Itoa(major+1)+".0", rec, in["tags"])
}

func (st *frdStore) updateModelVersion(in map[string]any) (map[string]any, *fcErr) {
	m, err := st.model(in)
	if err != nil {
		return nil, err
	}
	major := fcStr(in, "majorVersionNumber")
	var base map[string]any
	minor := 0
	for _, v := range st.versionsOf(m) {
		if parts := strings.SplitN(v, ".", 2); parts[0] == major {
			base = st.mvs[fcStr(m, "modelType")+"/"+fcStr(m, "modelId")+"/"+v]
			n, _ := strconv.Atoi(parts[1])
			minor = max(minor, n)
		}
	}
	if base == nil {
		return nil, frdNotFound("model %s has no major version %s", fcStr(m, "modelId"), major)
	}
	rec := map[string]any{"trainingDataSource": base["trainingDataSource"], "trainingDataSchema": base["trainingDataSchema"],
		"externalEventsDetail": base["externalEventsDetail"], "ingestedEventsDetail": base["ingestedEventsDetail"]}
	for _, key := range []string{"externalEventsDetail", "ingestedEventsDetail"} {
		if v, ok := in[key]; ok {
			rec[key] = v
		}
	}
	if err := st.checkTraining(m, rec); err != nil {
		return nil, err
	}
	return st.addModelVersion(m, major+"."+strconv.Itoa(minor+1), rec, in["tags"])
}

func (st *frdStore) modelVersion(in map[string]any) (string, map[string]any, *fcErr) {
	key := frdModelKey(in) + "/" + fcStr(in, "modelVersionNumber")
	rec := st.mvs[key]
	if rec == nil {
		return "", nil, frdNotFound("model %s version %s not found", fcStr(in, "modelId"), fcStr(in, "modelVersionNumber"))
	}
	frdSettle(rec, "TRAINING_IN_PROGRESS", "TRAINING_COMPLETE")
	return key, rec, nil
}

func (st *frdStore) updateModelVersionStatus(in map[string]any) (map[string]any, *fcErr) {
	_, rec, err := st.modelVersion(in)
	if err != nil {
		return nil, err
	}
	from, to := fcStr(rec, "status"), fcStr(in, "status")
	ok := map[string][]string{"ACTIVE": {"TRAINING_COMPLETE", "INACTIVE"}, "INACTIVE": {"ACTIVE"}, "TRAINING_CANCELLED": {"TRAINING_IN_PROGRESS"}}
	if !fcContains(ok[to], from) {
		return nil, frdValidation("a %s model version can't become %s", from, to)
	}
	if to == "INACTIVE" {
		if det, ver := st.versionUsing(rec); det != "" {
			return nil, frdConflict("model version is used by detector %s version %s", det, ver)
		}
	}
	rec["status"], rec["lastUpdatedTime"] = to, frdNow()
	return nil, nil
}

// versionUsing returns a detector version that uses the model version.
func (st *frdStore) versionUsing(mv map[string]any) (string, string) {
	for det, vs := range st.versions {
		for ver, v := range vs {
			for _, used := range fcObjList(v["modelVersions"]) {
				if fcStr(used, "modelId") == fcStr(mv, "modelId") && fcStr(used, "modelType") == fcStr(mv, "modelType") &&
					fcStr(used, "modelVersionNumber") == fcStr(mv, "modelVersionNumber") {
					return det, ver
				}
			}
		}
	}
	return "", ""
}

func (st *frdStore) deleteModelVersion(in map[string]any) (map[string]any, *fcErr) {
	key, rec, err := st.modelVersion(in)
	if err != nil {
		return nil, nil
	}
	if fcStr(rec, "status") == "ACTIVE" {
		return nil, frdConflict("an ACTIVE model version can't be deleted")
	}
	if det, ver := st.versionUsing(rec); det != "" {
		return nil, frdConflict("model version is used by detector %s version %s", det, ver)
	}
	delete(st.tags, fcStr(rec, "arn"))
	st.refund("model-version/" + key)
	delete(st.mvs, key)
	return nil, nil
}

func (st *frdStore) describeModelVersions(in map[string]any) (map[string]any, *fcErr) {
	var items []any
	for _, key := range frdSortedKeys(st.mvs) {
		rec := st.mvs[key]
		if (fcStr(in, "modelId") == "" || fcStr(rec, "modelId") == fcStr(in, "modelId")) &&
			(fcStr(in, "modelType") == "" || fcStr(rec, "modelType") == fcStr(in, "modelType")) &&
			(fcStr(in, "modelVersionNumber") == "" || fcStr(rec, "modelVersionNumber") == fcStr(in, "modelVersionNumber")) {
			frdSettle(rec, "TRAINING_IN_PROGRESS", "TRAINING_COMPLETE")
			items = append(items, rec)
		}
	}
	page, next, err := frdPage(items, in, "nextToken", 10)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"modelVersionDetails": page}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

// --- batch jobs ---

func (st *frdStore) createBatchJob(kind string, in map[string]any) (map[string]any, *fcErr) {
	id := fcStr(in, "jobId")
	if st.coll(kind)[id] != nil {
		return nil, frdValidation("job %s already exists", id)
	}
	if st.coll("event-type")[fcStr(in, "eventTypeName")] == nil {
		return nil, frdNotFound("event type %s not found", fcStr(in, "eventTypeName"))
	}
	rec := map[string]any{"jobId": id, "eventTypeName": in["eventTypeName"], "inputPath": in["inputPath"], "outputPath": in["outputPath"],
		"iamRoleArn": in["iamRoleArn"], "status": "IN_PROGRESS", "startTime": frdNow(), "_created": time.Now(),
		"processedRecordsCount": 0.0, "totalRecordsCount": 0.0}
	if kind == "batch-import" {
		rec["failedRecordsCount"] = 0.0
	} else {
		d, err := st.detector(fcStr(in, "detectorName"))
		if err != nil {
			return nil, err
		}
		if fcStr(d, "eventTypeName") != fcStr(in, "eventTypeName") {
			return nil, frdValidation("detector %s scores event type %s", fcStr(in, "detectorName"), fcStr(d, "eventTypeName"))
		}
		ver := fcStr(in, "detectorVersion")
		if ver == "" {
			v, err := st.activeVersion(fcStr(in, "detectorName"))
			if err != nil {
				return nil, err
			}
			ver = fcStr(v, "detectorVersionId")
		} else if _, err := st.version(fcStr(in, "detectorName"), ver); err != nil {
			return nil, err
		}
		rec["detectorName"], rec["detectorVersion"], rec["lastHeartbeatTime"] = in["detectorName"], ver, frdNow()
	}
	return nil, st.add(kind, id, kind+"/"+id, rec, in["tags"])
}

func (st *frdStore) cancelBatchJob(kind string, in map[string]any) (map[string]any, *fcErr) {
	rec := st.coll(kind)[fcStr(in, "jobId")]
	if rec == nil {
		return nil, frdNotFound("job %s not found", fcStr(in, "jobId"))
	}
	if s := fcStr(rec, "status"); s != "IN_PROGRESS" && s != "IN_PROGRESS_INITIALIZING" {
		return nil, frdValidation("a %s job can't be cancelled", s)
	}
	rec["status"], rec["completionTime"] = "CANCELED", frdNow()
	return nil, nil
}

func frdModelHandlers() map[string]frdHandler {
	h := map[string]frdHandler{
		"CreateModel": (*frdStore).createModel,
		"DeleteModel": (*frdStore).deleteModel,
		"GetModels":   (*frdStore).getModels,
		"UpdateModel": func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			m, err := st.model(in)
			if err != nil {
				return nil, err
			}
			m["description"], m["lastUpdatedTime"] = in["description"], frdNow()
			return nil, nil
		},
		"CreateModelVersion":       (*frdStore).createModelVersion,
		"UpdateModelVersion":       (*frdStore).updateModelVersion,
		"UpdateModelVersionStatus": (*frdStore).updateModelVersionStatus,
		"DeleteModelVersion":       (*frdStore).deleteModelVersion,
		"DescribeModelVersions":    (*frdStore).describeModelVersions,
		"GetModelVersion": func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			_, rec, err := st.modelVersion(in)
			return rec, err
		},
	}
	for kind, op := range map[string]string{"batch-import": "BatchImportJob", "batch-prediction": "BatchPredictionJob"} {
		kind, listKey := kind, map[string]string{"batch-import": "batchImports", "batch-prediction": "batchPredictions"}[kind]
		h["Create"+op] = func(st *frdStore, in map[string]any) (map[string]any, *fcErr) { return st.createBatchJob(kind, in) }
		h["Cancel"+op] = func(st *frdStore, in map[string]any) (map[string]any, *fcErr) { return st.cancelBatchJob(kind, in) }
		h["Delete"+op] = func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			st.drop(kind, fcStr(in, "jobId"))
			return nil, nil
		}
		h["Get"+op+"s"] = func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			return st.getNamed(kind, fcStr(in, "jobId"), in, listKey, 50)
		}
	}
	return h
}
