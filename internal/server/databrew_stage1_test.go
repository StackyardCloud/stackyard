package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const dbwTestRole = "arn:aws:iam::123456789012:role/databrew"

func dbwSeed(t *testing.T, srv *Server) {
	t.Helper()
	for _, b := range []string{"brew-in", "brew-out"} {
		if _, err := srv.s3.CreateBucket(b); err != nil {
			t.Fatal(err)
		}
	}
	csv := "name,city,score\nalice,paris,3\nbob,,5\ncarol,rome,4\n"
	if _, err := srv.s3.PutObject("brew-in", "people.csv", "text/csv", []byte(csv), nil, "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestDataBrewStage1DatasetsAndRecipes(t *testing.T) {
	srv, ts := newDataBrewTestServer(t)
	dbwSeed(t, srv)
	if code := dbwErrType(t, ts, http.MethodPost, "/datasets", `{"Name":"bad","Input":{"S3InputDefinition":{"Bucket":"brew-in","Key":"nope.csv"}}}`); code != "ValidationException" {
		t.Fatalf("missing S3 object: %s", code)
	}
	if code := dbwErrType(t, ts, http.MethodPost, "/datasets", `{"Name":"cat","Input":{"DataCatalogInputDefinition":{"DatabaseName":"db","TableName":"t"}}}`); code != "ValidationException" {
		t.Fatalf("missing Glue table: %s", code)
	}
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/datasets", `{"Name":"people","Format":"CSV","Input":{"S3InputDefinition":{"Bucket":"brew-in","Key":"people.csv"}},"Tags":{"team":"data"}}`)
	if code := dbwErrType(t, ts, http.MethodPost, "/datasets", `{"Name":"people","Input":{"S3InputDefinition":{"Bucket":"brew-in"}}}`); code != "ConflictException" {
		t.Fatalf("duplicate dataset: %s", code)
	}
	ds := dbwCall(t, ts, http.StatusOK, http.MethodGet, "/datasets/people", "")
	if ds["Source"] != "S3" || ds["Tags"].(map[string]any)["team"] != "data" {
		t.Fatalf("dataset: %v", ds)
	}

	steps := `[{"Action":{"Operation":"UPPER_CASE","Parameters":{"sourceColumn":"name"}}}]`
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/recipes", `{"Name":"clean","Steps":`+steps+`}`)
	if v := dbwCall(t, ts, http.StatusOK, http.MethodGet, "/recipes/clean", "")["RecipeVersion"]; v != "0.1" {
		t.Fatalf("working version: %v", v)
	}
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/recipes/clean/publishRecipe", `{"Description":"v1"}`)
	if v := dbwCall(t, ts, http.StatusOK, http.MethodGet, "/recipes/clean", "")["RecipeVersion"]; v != "1.0" {
		t.Fatalf("default describe returns the latest published version: %v", v)
	}
	if v := dbwCall(t, ts, http.StatusOK, http.MethodGet, "/recipes/clean?recipeVersion=LATEST_WORKING", "")["RecipeVersion"]; v != "1.1" {
		t.Fatalf("working version after publish: %v", v)
	}
	versions := dbwCall(t, ts, http.StatusOK, http.MethodGet, "/recipeVersions?name=clean", "")["Recipes"].([]any)
	if len(versions) != 1 {
		t.Fatalf("published versions: %v", versions)
	}
	if code := dbwErrType(t, ts, http.MethodDelete, "/recipes/clean/recipeVersion/LATEST_WORKING", ""); code != "ConflictException" {
		t.Fatalf("deleting LATEST_WORKING with published versions: %s", code)
	}
	batch := dbwCall(t, ts, http.StatusOK, http.MethodPost, "/recipes/clean/batchDeleteRecipeVersion", `{"RecipeVersions":["9.0"]}`)
	if errs := batch["Errors"].([]any); len(errs) != 1 {
		t.Fatalf("batch delete errors: %v", batch)
	}
}

func TestDataBrewStage1ProjectSessionAndJobs(t *testing.T) {
	srv, ts := newDataBrewTestServer(t)
	dbwSeed(t, srv)
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/datasets", `{"Name":"people","Input":{"S3InputDefinition":{"Bucket":"brew-in","Key":"people.csv"}}}`)
	steps := `[{"Action":{"Operation":"UPPER_CASE","Parameters":{"sourceColumn":"name"}}},{"Action":{"Operation":"FILL_WITH_CUSTOM","Parameters":{"sourceColumn":"city","value":"unknown"}}}]`
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/recipes", `{"Name":"clean","Steps":`+steps+`}`)

	// Projects and sessions.
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/projects", `{"Name":"proj","DatasetName":"people","RecipeName":"clean","RoleArn":"`+dbwTestRole+`"}`)
	if code := dbwErrType(t, ts, http.MethodPost, "/projects", `{"Name":"proj2","DatasetName":"people","RecipeName":"clean","RoleArn":"`+dbwTestRole+`"}`); code != "ConflictException" {
		t.Fatalf("recipe already in a project: %s", code)
	}
	if code := dbwErrType(t, ts, http.MethodPut, "/projects/proj/sendProjectSessionAction", `{"Preview":true}`); code != "ConflictException" {
		t.Fatalf("action without a session: %s", code)
	}
	sess := dbwCall(t, ts, http.StatusOK, http.MethodPut, "/projects/proj/startProjectSession", `{}`)
	if code := dbwErrType(t, ts, http.MethodPut, "/projects/proj/startProjectSession", `{}`); code != "ConflictException" {
		t.Fatalf("second session without AssumeControl: %s", code)
	}
	preview := dbwCall(t, ts, http.StatusOK, http.MethodPut, "/projects/proj/sendProjectSessionAction",
		`{"Preview":true,"ClientSessionId":"`+sess["ClientSessionId"].(string)+`","RecipeStep":{"Action":{"Operation":"RENAME","Parameters":{"sourceColumn":"score","targetColumn":"points"}}}}`)
	var view struct {
		Columns []string   `json:"columns"`
		Rows    [][]string `json:"rows"`
	}
	if err := json.Unmarshal([]byte(preview["Result"].(string)), &view); err != nil {
		t.Fatal(err)
	}
	if strings.Join(view.Columns, ",") != "name,city,points" || view.Rows[0][0] != "ALICE" || view.Rows[1][1] != "unknown" {
		t.Fatalf("preview: %+v", view)
	}
	if code := dbwErrType(t, ts, http.MethodPut, "/projects/proj/sendProjectSessionAction", `{"Preview":true,"RecipeStep":{"Action":{"Operation":"PIVOT"}}}`); code != "ValidationException" {
		t.Fatalf("unsupported operation: %s", code)
	}

	// A recipe job runs the published recipe and writes CSV to S3.
	if code := dbwErrType(t, ts, http.MethodPost, "/recipeJobs", `{"Name":"run","DatasetName":"people","RecipeReference":{"Name":"clean"},"RoleArn":"`+dbwTestRole+`","Outputs":[{"Location":{"Bucket":"brew-out","Key":"out/"}}]}`); code != "ValidationException" {
		t.Fatalf("job on an unpublished recipe: %s", code)
	}
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/recipes/clean/publishRecipe", `{}`)
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/recipeJobs", `{"Name":"run","DatasetName":"people","RecipeReference":{"Name":"clean"},"RoleArn":"`+dbwTestRole+`","Outputs":[{"Location":{"Bucket":"brew-out","Key":"out/"}}]}`)
	runID := dbwCall(t, ts, http.StatusOK, http.MethodPost, "/jobs/run/startJobRun", "")["RunId"].(string)
	states := []string{}
	for i := 0; i < 3; i++ {
		states = append(states, dbwCall(t, ts, http.StatusOK, http.MethodGet, "/jobs/run/jobRun/"+runID, "")["State"].(string))
	}
	if strings.Join(states, ",") != "RUNNING,SUCCEEDED,SUCCEEDED" {
		t.Fatalf("run states: %v", states)
	}
	objs, _ := srv.s3.ListObjects("brew-out", "out/")
	if len(objs) != 1 {
		t.Fatalf("recipe job output objects: %d", len(objs))
	}
	out, _ := srv.s3.GetObject("brew-out", objs[0].Key)
	if want := "name,city,score\nALICE,paris,3\nBOB,unknown,5\nCAROL,rome,4\n"; string(out.Body) != want {
		t.Fatalf("recipe job output:\n%s", out.Body)
	}
	if code := dbwErrType(t, ts, http.MethodDelete, "/recipes/clean/recipeVersion/1.0", ""); code != "ConflictException" {
		t.Fatalf("deleting a version a job runs: %s", code)
	}

	// A profile job writes per-column statistics.
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/profileJobs", `{"Name":"prof","DatasetName":"people","RoleArn":"`+dbwTestRole+`","OutputLocation":{"Bucket":"brew-out","Key":"profile"}}`)
	pid := dbwCall(t, ts, http.StatusOK, http.MethodPost, "/jobs/prof/startJobRun", "")["RunId"].(string)
	pobjs, _ := srv.s3.ListObjects("brew-out", "profile/")
	if len(pobjs) != 1 || !strings.Contains(pobjs[0].Key, pid) {
		t.Fatalf("profile output: %v", pobjs)
	}
	pbody, _ := srv.s3.GetObject("brew-out", pobjs[0].Key)
	var prof struct {
		Columns []map[string]any `json:"columns"`
	}
	_ = json.Unmarshal(pbody.Body, &prof)
	if len(prof.Columns) != 3 || prof.Columns[1]["missingValuesCount"] != float64(1) || prof.Columns[2]["max"] != float64(5) {
		t.Fatalf("profile: %+v", prof.Columns)
	}

	// Stopping, run listing, schedules and deletes.
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/jobs/prof/jobRun/"+pid+"/stopJobRun", "")
	if code := dbwErrType(t, ts, http.MethodPost, "/jobs/prof/jobRun/"+pid+"/stopJobRun", ""); code != "ValidationException" {
		t.Fatalf("stopping a stopped run: %s", code)
	}
	if st := dbwCall(t, ts, http.StatusOK, http.MethodGet, "/jobs/prof/jobRun/"+pid, "")["State"]; st != "STOPPED" {
		t.Fatalf("stopped run state: %v", st)
	}
	runs := dbwCall(t, ts, http.StatusOK, http.MethodGet, "/jobs/run/jobRuns", "")["JobRuns"].([]any)
	if len(runs) != 1 {
		t.Fatalf("job runs: %v", runs)
	}
	if code := dbwErrType(t, ts, http.MethodPost, "/schedules", `{"Name":"nightly","CronExpression":"cron(0 1 * * ? *)","JobNames":["ghost"]}`); code != "ValidationException" {
		t.Fatalf("schedule for a missing job: %s", code)
	}
	if code := dbwErrType(t, ts, http.MethodPost, "/schedules", `{"Name":"nightly","CronExpression":"rate(1 day)"}`); code != "ValidationException" {
		t.Fatalf("non-cron schedule: %s", code)
	}
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/schedules", `{"Name":"nightly","CronExpression":"cron(0 1 * * ? *)","JobNames":["run"]}`)
	if code := dbwErrType(t, ts, http.MethodDelete, "/jobs/run", ""); code != "ConflictException" {
		t.Fatalf("deleting a scheduled job: %s", code)
	}
	if code := dbwErrType(t, ts, http.MethodDelete, "/datasets/people", ""); code != "ConflictException" {
		t.Fatalf("deleting a dataset in use: %s", code)
	}
	if n := len(dbwCall(t, ts, http.StatusOK, http.MethodGet, "/schedules?jobName=run", "")["Schedules"].([]any)); n != 1 {
		t.Fatalf("schedules for job: %d", n)
	}
}

func TestDataBrewStage1RulesetsAndTags(t *testing.T) {
	srv, ts := newDataBrewTestServer(t)
	dbwSeed(t, srv)
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/datasets", `{"Name":"people","Input":{"S3InputDefinition":{"Bucket":"brew-in","Key":"people.csv"}}}`)
	target := dbwARN("dataset", "people")
	if code := dbwErrType(t, ts, http.MethodPost, "/rulesets", `{"Name":"rs","TargetArn":"`+target+`","Rules":[{"Name":"r","CheckExpression":":col1 > :val1","SubstitutionMap":{":col1":"`+"`score`"+`"}}]}`); code != "ValidationException" {
		t.Fatalf("undefined placeholder: %s", code)
	}
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/rulesets", `{"Name":"rs","TargetArn":"`+target+`","Rules":[{"Name":"r","CheckExpression":":col1 > :val1","SubstitutionMap":{":col1":"`+"`score`"+`",":val1":"0"}}]}`)
	items := dbwCall(t, ts, http.StatusOK, http.MethodGet, "/rulesets?targetArn="+url.QueryEscape(target), "")["Rulesets"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["RuleCount"] != float64(1) {
		t.Fatalf("rulesets: %v", items)
	}
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/profileJobs", `{"Name":"prof","DatasetName":"people","RoleArn":"`+dbwTestRole+`","OutputLocation":{"Bucket":"brew-out"},"ValidationConfigurations":[{"RulesetArn":"`+dbwARN("ruleset", "rs")+`"}]}`)
	if code := dbwErrType(t, ts, http.MethodDelete, "/rulesets/rs", ""); code != "ConflictException" {
		t.Fatalf("deleting a ruleset in use: %s", code)
	}

	arn := url.PathEscape(target)
	dbwCall(t, ts, http.StatusOK, http.MethodPost, "/tags/"+arn, `{"Tags":{"env":"dev"}}`)
	if code := dbwErrType(t, ts, http.MethodPost, "/tags/"+arn, `{"Tags":{"aws:x":"y"}}`); code != "ValidationException" {
		t.Fatalf("aws: tag key: %s", code)
	}
	if tags := dbwCall(t, ts, http.StatusOK, http.MethodGet, "/tags/"+arn, "")["Tags"].(map[string]any); tags["env"] != "dev" {
		t.Fatalf("tags: %v", tags)
	}
	dbwCall(t, ts, http.StatusOK, http.MethodDelete, "/tags/"+arn+"?tagKeys=env", "")
	if code := dbwErrType(t, ts, http.MethodGet, "/tags/"+url.PathEscape(dbwARN("dataset", "ghost")), ""); code != "ResourceNotFoundException" {
		t.Fatalf("tags of a missing resource: %s", code)
	}
}

// Recipe steps: each supported operation, and unsupported ones failing clearly.
func TestDataBrewRecipeOperations(t *testing.T) {
	newTable := func() *dbwTable {
		return &dbwTable{cols: []string{"a", "b"}, rows: [][]string{{"hello world", ""}, {"x", "y"}}}
	}
	step := func(op string, params map[string]any) []any {
		return []any{map[string]any{"Action": map[string]any{"Operation": op, "Parameters": params}}}
	}
	cases := []struct {
		op     string
		params map[string]any
		cols   string
		first  string
	}{
		{"UPPER_CASE", map[string]any{"sourceColumn": "a"}, "a,b", "HELLO WORLD|"},
		{"CAPITAL_CASE", map[string]any{"sourceColumn": "a"}, "a,b", "Hello World|"},
		{"SENTENCE_CASE", map[string]any{"sourceColumn": "a"}, "a,b", "Hello world|"},
		{"ADD_PREFIX", map[string]any{"sourceColumn": "a", "pattern": ">"}, "a,b", ">hello world|"},
		{"FILL_WITH_CUSTOM", map[string]any{"sourceColumn": "b", "value": "?"}, "a,b", "hello world|?"},
		{"RENAME", map[string]any{"sourceColumn": "b", "targetColumn": "c"}, "a,c", "hello world|"},
		{"DUPLICATE", map[string]any{"sourceColumn": "a", "targetColumn": "a2"}, "a,b,a2", "hello world||hello world"},
		{"DELETE", map[string]any{"sourceColumns": `["a"]`}, "b", ""},
	}
	for _, c := range cases {
		tb := newTable()
		if err := tb.apply(step(c.op, c.params)); err != nil {
			t.Fatalf("%s: %v", c.op, err)
		}
		if got := strings.Join(tb.cols, ","); got != c.cols {
			t.Fatalf("%s columns: %s", c.op, got)
		}
		if got := strings.Join(tb.rows[0], "|"); got != c.first {
			t.Fatalf("%s first row: %q", c.op, got)
		}
	}
	tb := newTable()
	if err := tb.apply(step("REMOVE_MISSING", map[string]any{"sourceColumn": "b"})); err != nil || len(tb.rows) != 1 {
		t.Fatalf("REMOVE_MISSING: %v %v", err, tb.rows)
	}
	if err := newTable().apply(step("PIVOT", nil)); err == nil || !strings.Contains(err.Error(), "isn't supported") {
		t.Fatalf("unsupported operation: %v", err)
	}
	if err := newTable().apply(step("UPPER_CASE", map[string]any{"sourceColumn": "zz"})); err == nil {
		t.Fatal("missing column accepted")
	}
}
