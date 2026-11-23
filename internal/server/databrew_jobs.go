package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// dbwPutDataset checks the dataset input against the S3 or Glue store (before
// the DataBrew lock is taken) and creates (name == "") or updates it.
func (s *Server) dbwPutDataset(name string, r *http.Request) (any, *dbwError) {
	op := "UpdateDataset"
	if name == "" {
		op = "CreateDataset"
	}
	in, err := dbwDecode(r, op)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = dbwStr(in, "Name")
	}
	input := dbwMap(in, "Input")
	defs := 0
	for _, k := range []string{"S3InputDefinition", "DataCatalogInputDefinition", "DatabaseInputDefinition"} {
		if input[k] != nil {
			defs++
		}
	}
	if defs != 1 {
		return nil, dbwValidation("Input must have exactly one of S3InputDefinition, DataCatalogInputDefinition or DatabaseInputDefinition.")
	}
	if loc := dbwMap(input, "S3InputDefinition"); loc != nil {
		if err := s.dbwCheckS3Input(dbwStr(loc, "Bucket"), dbwStr(loc, "Key")); err != nil {
			return nil, err
		}
	}
	if cat := dbwMap(input, "DataCatalogInputDefinition"); cat != nil {
		if _, gerr := s.glue.handle("GetTable", "", glueRecord{"DatabaseName": dbwStr(cat, "DatabaseName"), "Name": dbwStr(cat, "TableName")}); gerr != nil {
			return nil, dbwValidation("Glue table %s.%s doesn't exist.", dbwStr(cat, "DatabaseName"), dbwStr(cat, "TableName"))
		}
	}
	return s.dataBrew.putDataset(name, in, op == "CreateDataset")
}

// dbwCheckS3Input requires the bucket, and the key as an object or a prefix
// with objects under it.
func (s *Server) dbwCheckS3Input(bucket, key string) *dbwError {
	if _, err := s.s3.GetBucket(bucket); err != nil {
		return dbwValidation("S3 bucket %s doesn't exist.", bucket)
	}
	if key == "" {
		return nil
	}
	if _, err := s.s3.GetObject(bucket, key); err == nil {
		return nil
	}
	if objs, err := s.s3.ListObjects(bucket, key); err == nil && len(objs) > 0 {
		return nil
	}
	return dbwValidation("s3://%s/%s doesn't exist.", bucket, key)
}

func (s *Server) dbwCheckBucket(loc map[string]any) *dbwError {
	if loc == nil {
		return nil
	}
	if _, err := s.s3.GetBucket(dbwStr(loc, "Bucket")); err != nil {
		return dbwValidation("S3 bucket %s doesn't exist.", dbwStr(loc, "Bucket"))
	}
	return nil
}

// dbwPutJob validates a profile or recipe job and creates (name == "") or
// updates it. Output buckets and catalog tables are checked before the lock.
func (s *Server) dbwPutJob(kind, name string, in map[string]any) (any, *dbwError) {
	create := name == ""
	if create {
		name = dbwStr(in, "Name")
	}
	if err := s.dbwCheckBucket(dbwMap(in, "OutputLocation")); err != nil {
		return nil, err
	}
	for _, o := range dbwList(in, "Outputs") {
		if err := s.dbwCheckBucket(dbwMap(o, "Location")); err != nil {
			return nil, err
		}
	}
	for _, o := range dbwList(in, "DataCatalogOutputs") {
		if _, gerr := s.glue.handle("GetTable", "", glueRecord{"DatabaseName": dbwStr(o, "DatabaseName"), "Name": dbwStr(o, "TableName")}); gerr != nil {
			return nil, dbwValidation("Glue table %s.%s doesn't exist.", dbwStr(o, "DatabaseName"), dbwStr(o, "TableName"))
		}
		if err := s.dbwCheckBucket(dbwMap(dbwMap(o, "S3Options"), "Location")); err != nil {
			return nil, err
		}
	}
	tags := dbwTagMap(in["Tags"])
	if err := dbwCheckTags(tags); err != nil {
		return nil, err
	}
	return s.dataBrew.putJob(kind, name, in, create, tags)
}

func (st *dataBrewStore) putJob(kind, name string, in map[string]any, create bool, tags map[string]string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	var prev map[string]any
	if create {
		if st.recs["job"][name] != nil {
			return nil, dbwConflict("A job named %s already exists.", name)
		}
	} else {
		var err *dbwError
		if prev, err = st.getLocked("job", name); err != nil {
			return nil, err
		}
		if dbwStr(prev, "Type") != kind {
			return nil, dbwValidation("Job %s is a %s job.", name, strings.ToLower(dbwStr(prev, "Type")))
		}
	}
	rec := map[string]any{"Type": kind}
	for _, k := range []string{"EncryptionKeyArn", "EncryptionMode", "LogSubscription", "MaxCapacity", "MaxRetries", "RoleArn", "Timeout", "JobSample", "ValidationConfigurations"} {
		if v, ok := in[k]; ok {
			rec[k] = v
		}
	}
	if kind == "PROFILE" {
		ds := dbwStr(in, "DatasetName")
		if prev != nil {
			ds = dbwStr(prev, "DatasetName")
		}
		if st.recs["dataset"][ds] == nil {
			return nil, dbwValidation("Dataset %s doesn't exist.", ds)
		}
		rec["DatasetName"] = ds
		rec["Outputs"] = []any{map[string]any{"Location": in["OutputLocation"]}}
		if c, ok := in["Configuration"]; ok {
			rec["ProfileConfiguration"] = c
		}
		for _, vc := range dbwList(in, "ValidationConfigurations") {
			arn := dbwStr(vc, "RulesetArn")
			kind, rs, ok := dbwParseARN(arn)
			if !ok || kind != "ruleset" || st.recs["ruleset"][rs] == nil {
				return nil, dbwValidation("Ruleset %s doesn't exist.", arn)
			}
			if dbwStr(st.recs["ruleset"][rs], "TargetArn") != dbwARN("dataset", ds) {
				return nil, dbwValidation("Ruleset %s doesn't target dataset %s.", rs, ds)
			}
		}
	} else {
		if prev != nil {
			for _, k := range []string{"DatasetName", "ProjectName", "RecipeReference"} {
				if v, ok := prev[k]; ok {
					rec[k] = v
				}
			}
		} else if err := st.recipeJobSourceLocked(in, rec); err != nil {
			return nil, err
		}
		for _, k := range []string{"Outputs", "DataCatalogOutputs", "DatabaseOutputs"} {
			if v, ok := in[k]; ok {
				rec[k] = v
			}
		}
		if len(dbwList(rec, "Outputs"))+len(dbwList(rec, "DataCatalogOutputs"))+len(dbwList(rec, "DatabaseOutputs")) == 0 {
			return nil, dbwValidation("A recipe job needs Outputs, DataCatalogOutputs or DatabaseOutputs.")
		}
	}
	if prev == nil {
		if err := st.newRecordLocked("job", name, rec, tags); err != nil {
			return nil, err
		}
		return map[string]any{"Name": name}, nil
	}
	for _, k := range []string{"Name", "AccountId", "CreatedBy", "CreateDate", "ResourceArn"} {
		rec[k] = prev[k]
	}
	dbwTouch(rec)
	st.recs["job"][name] = rec
	return map[string]any{"Name": name}, nil
}

// recipeJobSourceLocked takes a recipe job's input from a project, or from a
// dataset plus a published recipe version.
func (st *dataBrewStore) recipeJobSourceLocked(in, rec map[string]any) *dbwError {
	if p := dbwStr(in, "ProjectName"); p != "" {
		if in["DatasetName"] != nil || in["RecipeReference"] != nil {
			return dbwValidation("Specify ProjectName or DatasetName with RecipeReference, not both.")
		}
		project := st.recs["project"][p]
		if project == nil {
			return dbwValidation("Project %s doesn't exist.", p)
		}
		rec["ProjectName"], rec["DatasetName"] = p, dbwStr(project, "DatasetName")
		return nil
	}
	ds := dbwStr(in, "DatasetName")
	ref := dbwMap(in, "RecipeReference")
	if ds == "" || ref == nil {
		return dbwValidation("A recipe job needs ProjectName, or DatasetName and RecipeReference.")
	}
	if st.recs["dataset"][ds] == nil {
		return dbwValidation("Dataset %s doesn't exist.", ds)
	}
	version := dbwStr(ref, "RecipeVersion")
	if version == "" {
		version = "LATEST_PUBLISHED"
	}
	if version == "LATEST_WORKING" {
		return dbwValidation("A recipe job runs a published recipe version.")
	}
	if _, err := st.recipeVersionLocked(dbwStr(ref, "Name"), version); err != nil {
		return dbwValidation("%s", err.msg)
	}
	rec["DatasetName"], rec["RecipeReference"] = ds, ref
	return nil
}

func (st *dataBrewStore) jobFilter(q url.Values) func(map[string]any) bool {
	ds, p := q.Get("datasetName"), q.Get("projectName")
	if ds == "" && p == "" {
		return nil
	}
	return func(rec map[string]any) bool {
		return (ds == "" || dbwStr(rec, "DatasetName") == ds) && (p == "" || dbwStr(rec, "ProjectName") == p)
	}
}

func (st *dataBrewStore) deleteJob(name string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.getLocked("job", name); err != nil {
		return nil, err
	}
	for sn, sc := range st.recs["schedule"] {
		for _, j := range dbwStrings(sc["JobNames"]) {
			if j == name {
				return nil, dbwConflict("Job %s is used by schedule %s.", name, sn)
			}
		}
	}
	delete(st.runs, name)
	st.dropLocked("job", name)
	return map[string]any{"Name": name}, nil
}

// dbwJobSpec is what a run needs, copied out of the store so the S3 work runs
// without the lock.
type dbwJobSpec struct {
	job     map[string]any
	dataset map[string]any
	steps   []any
}

func (s *Server) dbwStartJobRun(name string) (any, *dbwError) {
	spec, err := s.dataBrew.jobSpec(name)
	if err != nil {
		return nil, err
	}
	started := time.Now().UTC()
	runID := "db_" + dbwRandomID(16)
	runErr := s.dbwRunJob(spec, runID, started)
	return s.dataBrew.addRun(name, spec, runID, started, runErr)
}

func (st *dataBrewStore) jobSpec(name string) (*dbwJobSpec, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	job, err := st.getLocked("job", name)
	if err != nil {
		return nil, err
	}
	spec := &dbwJobSpec{job: job, dataset: st.recs["dataset"][dbwStr(job, "DatasetName")]}
	if spec.dataset == nil {
		return nil, dbwValidation("Dataset %s no longer exists.", dbwStr(job, "DatasetName"))
	}
	switch {
	case dbwStr(job, "ProjectName") != "":
		project := st.recs["project"][dbwStr(job, "ProjectName")]
		if project == nil {
			return nil, dbwValidation("Project %s no longer exists.", dbwStr(job, "ProjectName"))
		}
		recipe := st.recs["recipe"][dbwStr(project, "RecipeName")]
		spec.steps, _ = recipe["Steps"].([]any)
	case dbwStr(job, "Type") == "RECIPE":
		ref := dbwMap(job, "RecipeReference")
		v, err := st.recipeVersionLocked(dbwStr(ref, "Name"), dbwFirst(dbwStr(ref, "RecipeVersion"), "LATEST_PUBLISHED"))
		if err != nil {
			return nil, dbwValidation("%s", err.msg)
		}
		spec.steps, _ = v["Steps"].([]any)
	}
	return spec, nil
}

func dbwFirst(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func (st *dataBrewStore) addRun(name string, spec *dbwJobSpec, runID string, started time.Time, runErr error) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.recs["job"][name] == nil {
		return nil, dbwNotFound("The job %s wasn't found.", name)
	}
	job := spec.job
	rec := map[string]any{
		"RunId": runID, "JobName": name, "Attempt": float64(0), "State": "STARTING",
		"StartedOn": float64(started.UnixMilli()) / 1000, "StartedBy": dbwCreatedBy,
		"DatasetName": job["DatasetName"], "LogGroupName": "/aws-glue-databrew/jobs/" + name,
	}
	for _, k := range []string{"LogSubscription", "Outputs", "DataCatalogOutputs", "DatabaseOutputs", "RecipeReference", "JobSample", "ValidationConfigurations", "ProfileConfiguration"} {
		if v, ok := job[k]; ok {
			rec[k] = v
		}
	}
	run := &dbwRun{rec: rec, final: "SUCCEEDED"}
	if runErr != nil {
		run.final, run.errMsg = "FAILED", runErr.Error()
	}
	runs := append(st.runs[name], run)
	if len(runs) > dbwMaxRunsPerJob {
		runs = runs[len(runs)-dbwMaxRunsPerJob:]
	}
	st.runs[name] = runs
	return map[string]any{"RunId": runID}, nil
}

// settle moves a run one state on per read: STARTING, RUNNING, then its result.
// ponytail: the run's S3 work happens at StartJobRun; reads only report it.
func (run *dbwRun) settle() {
	switch run.rec["State"] {
	case "STARTING":
		run.rec["State"] = "RUNNING"
	case "RUNNING":
		run.rec["State"] = run.final
		now := time.Now()
		run.rec["CompletedOn"] = float64(now.UnixMilli()) / 1000
		started, _ := run.rec["StartedOn"].(float64)
		run.rec["ExecutionTime"] = float64(max(0, int64(float64(now.Unix())-started)))
		if run.errMsg != "" {
			run.rec["ErrorMessage"] = run.errMsg
		}
	}
}

func (st *dataBrewStore) findRunLocked(name, runID string) (*dbwRun, *dbwError) {
	if st.recs["job"][name] == nil {
		return nil, dbwNotFound("The job %s wasn't found.", name)
	}
	for _, run := range st.runs[name] {
		if dbwStr(run.rec, "RunId") == runID {
			return run, nil
		}
	}
	return nil, dbwNotFound("Job %s has no run %s.", name, runID)
}

func (st *dataBrewStore) describeRun(name, runID string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	run, err := st.findRunLocked(name, runID)
	if err != nil {
		return nil, err
	}
	run.settle()
	return dbwShape("DescribeJobRunResponse", run.rec), nil
}

func (st *dataBrewStore) listRuns(name string, q url.Values) (any, *dbwError) {
	max, err := dbwMaxResults(q)
	if err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.recs["job"][name] == nil {
		return nil, dbwNotFound("The job %s wasn't found.", name)
	}
	runs := st.runs[name]
	start, end, next, err := dbwPage(len(runs), q.Get("nextToken"), max)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for i := start; i < end; i++ {
		run := runs[len(runs)-1-i] // newest first
		run.settle()
		items = append(items, dbwShape("JobRun", run.rec))
	}
	out := map[string]any{"JobRuns": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func (st *dataBrewStore) stopRun(name, runID string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	run, err := st.findRunLocked(name, runID)
	if err != nil {
		return nil, err
	}
	if state := run.rec["State"]; state != "STARTING" && state != "RUNNING" {
		return nil, dbwValidation("Job run %s is %s and can't be stopped.", runID, state)
	}
	run.rec["State"], run.final, run.errMsg = "STOPPED", "STOPPED", ""
	run.rec["CompletedOn"] = dbwNow()
	return map[string]any{"RunId": runID}, nil
}

// dbwSessionAction previews (or, without Preview, appends) a recipe step on a
// sample of the project's dataset.
func (s *Server) dbwSessionAction(name string, in map[string]any) (any, *dbwError) {
	st := s.dataBrew
	st.mu.Lock()
	project, err := st.getLocked("project", name)
	if err != nil {
		st.mu.Unlock()
		return nil, err
	}
	sess := st.sessions[name]
	if sess == nil {
		st.mu.Unlock()
		return nil, dbwConflict("Project %s has no open session; call StartProjectSession first.", name)
	}
	if id := dbwStr(in, "ClientSessionId"); id != "" && id != sess.id {
		st.mu.Unlock()
		return nil, dbwValidation("ClientSessionId doesn't match the project's open session.")
	}
	dataset := st.recs["dataset"][dbwStr(project, "DatasetName")]
	recipeName := dbwStr(project, "RecipeName")
	steps, _ := st.recs["recipe"][recipeName]["Steps"].([]any)
	sample := dbwMap(project, "Sample")
	st.mu.Unlock()

	step := dbwMap(in, "RecipeStep")
	idx := len(steps)
	if v, ok := in["StepIndex"].(float64); ok {
		if int(v) > len(steps) {
			return nil, dbwValidation("StepIndex %d is past the end of the recipe (%d steps).", int(v), len(steps))
		}
		idx = int(v)
	}
	applied := append([]any{}, steps[:idx]...)
	if step != nil {
		applied = append(applied, step)
	}
	if dataset == nil {
		return nil, dbwValidation("The project's dataset no longer exists.")
	}
	size, _ := sample["Size"].(float64)
	table, lerr := s.dbwLoadTable(dataset, max(1, int(size)), dbwStr(sample, "Type") == "LAST_N")
	if lerr != nil {
		return nil, dbwValidation("%s", lerr.Error())
	}
	if aerr := table.apply(applied); aerr != nil {
		return nil, dbwValidation("%s", aerr.Error())
	}
	result, _ := json.Marshal(table.view(dbwMap(in, "ViewFrame")))

	st.mu.Lock()
	defer st.mu.Unlock()
	if st.sessions[name] != sess {
		return nil, dbwConflict("The project session was taken over.")
	}
	sess.actionID++
	if step != nil && !dbwBool(in, "Preview") {
		recipe := st.recs["recipe"][recipeName]
		if recipe == nil {
			return nil, dbwValidation("The project's recipe no longer exists.")
		}
		cur, _ := recipe["Steps"].([]any)
		if idx > len(cur) {
			idx = len(cur)
		}
		next := map[string]any{}
		for k, v := range recipe {
			next[k] = v
		}
		updated := append(append(append([]any{}, cur[:idx]...), step), cur[idx:]...)
		next["Steps"] = updated
		dbwTouch(next)
		st.recs["recipe"][recipeName] = next
	}
	return map[string]any{"Name": name, "ActionId": float64(sess.actionID), "Result": string(result)}, nil
}

// view applies a ViewFrame: a column window (at most 20 columns, minus hidden
// ones) and a row window.
func (t *dbwTable) view(frame map[string]any) map[string]any {
	colStart, _ := frame["StartColumnIndex"].(float64)
	colRange, ok := frame["ColumnRange"].(float64)
	if !ok {
		colRange = 20
	}
	rowStart, _ := frame["StartRowIndex"].(float64)
	rowRange, ok := frame["RowRange"].(float64)
	if !ok {
		rowRange = float64(len(t.rows))
	}
	hidden := map[string]bool{}
	for _, h := range dbwStrings(frame["HiddenColumns"]) {
		hidden[h] = true
	}
	var cols []int
	for i := int(colStart); i < len(t.cols) && len(cols) < int(colRange); i++ {
		if !hidden[t.cols[i]] {
			cols = append(cols, i)
		}
	}
	names := make([]string, 0, len(cols))
	for _, c := range cols {
		names = append(names, t.cols[c])
	}
	rows := [][]string{}
	for i := int(rowStart); i < len(t.rows) && i < int(rowStart+rowRange); i++ {
		row := make([]string, 0, len(cols))
		for _, c := range cols {
			row = append(row, t.rows[i][c])
		}
		rows = append(rows, row)
	}
	return map[string]any{"columns": names, "rows": rows, "rowCount": len(t.rows)}
}

// dbwSortedKeys returns m's keys sorted.
func dbwSortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
