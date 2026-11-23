package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	dbwAccountID     = "123456789012"
	dbwRegion        = "us-east-1"
	dbwMaxBody       = 1 << 20
	dbwMaxPerKind    = 1000 // datasets, recipes, projects, jobs, schedules, rulesets each
	dbwMaxVersions   = 100  // published versions per recipe
	dbwMaxRunsPerJob = 100  // kept per job; the oldest are dropped
	dbwMaxTags       = 200
	dbwDefaultSample = 500
)

var dbwCreatedBy = "arn:aws:iam::" + dbwAccountID + ":root"

// dbwKinds maps a resource kind to its ARN segment, describe/list output
// shapes and the list response key.
var dbwKinds = map[string]struct{ describe, item, listKey string }{
	"dataset":  {"DescribeDatasetResponse", "Dataset", "Datasets"},
	"project":  {"DescribeProjectResponse", "Project", "Projects"},
	"job":      {"DescribeJobResponse", "Job", "Jobs"},
	"schedule": {"DescribeScheduleResponse", "Schedule", "Schedules"},
	"ruleset":  {"DescribeRulesetResponse", "RulesetItem", "Rulesets"},
	"recipe":   {"DescribeRecipeResponse", "Recipe", "Recipes"},
}

type dbwRun struct {
	rec    map[string]any
	final  string // SUCCEEDED or FAILED once settled
	errMsg string
	reads  int
}

type dbwSession struct {
	id       string
	actionID int
}

type dataBrewStore struct {
	mu       sync.Mutex
	recs     map[string]map[string]map[string]any // kind -> name -> record (recipes: LATEST_WORKING)
	versions map[string][]map[string]any          // recipe -> published versions, oldest first
	runs     map[string][]*dbwRun                 // job -> runs, oldest first
	tags     map[string]map[string]string         // ARN -> tags
	sessions map[string]*dbwSession               // project -> open session
}

func newDataBrewStore() *dataBrewStore {
	st := &dataBrewStore{
		recs:     map[string]map[string]map[string]any{},
		versions: map[string][]map[string]any{},
		runs:     map[string][]*dbwRun{},
		tags:     map[string]map[string]string{},
		sessions: map[string]*dbwSession{},
	}
	for kind := range dbwKinds {
		st.recs[kind] = map[string]map[string]any{}
	}
	return st
}

func dbwARN(kind, name string) string {
	return fmt.Sprintf("arn:aws:databrew:%s:%s:%s/%s", dbwRegion, dbwAccountID, kind, name)
}

// dbwParseARN splits a DataBrew ARN into kind and name.
func dbwParseARN(arn string) (string, string, bool) {
	rest, ok := strings.CutPrefix(arn, "arn:aws:databrew:"+dbwRegion+":"+dbwAccountID+":")
	if !ok {
		return "", "", false
	}
	kind, name, ok := strings.Cut(rest, "/")
	if _, known := dbwKinds[kind]; !ok || !known || name == "" {
		return "", "", false
	}
	return kind, name, true
}

func dbwNow() float64 { return float64(time.Now().UnixMilli()) / 1000 }

func dbwRandomID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// shape conforms v to a DataBrew output shape (a fresh copy, so callers can
// encode it after the lock is released).
func dbwShape(shape string, v map[string]any) map[string]any {
	out, _ := awsmodels.DataBrewModel().Conform(shape, v).(map[string]any)
	if out == nil {
		out = map[string]any{}
	}
	return out
}

// newRecordLocked stamps the common metadata and stores rec under kind/name.
func (st *dataBrewStore) newRecordLocked(kind, name string, rec map[string]any, tags map[string]string) *dbwError {
	if len(st.recs[kind]) >= dbwMaxPerKind {
		return dbwQuota("An account can have at most %d DataBrew %ss.", dbwMaxPerKind, kind)
	}
	now := dbwNow()
	rec["Name"], rec["AccountId"], rec["CreatedBy"], rec["CreateDate"] = name, dbwAccountID, dbwCreatedBy, now
	rec["LastModifiedBy"], rec["LastModifiedDate"], rec["ResourceArn"] = dbwCreatedBy, now, dbwARN(kind, name)
	st.recs[kind][name] = rec
	if len(tags) > 0 {
		st.tags[dbwARN(kind, name)] = tags
	}
	return nil
}

func dbwTouch(rec map[string]any) {
	rec["LastModifiedBy"], rec["LastModifiedDate"] = dbwCreatedBy, dbwNow()
}

func (st *dataBrewStore) getLocked(kind, name string) (map[string]any, *dbwError) {
	rec := st.recs[kind][name]
	if rec == nil {
		return nil, dbwNotFound("The %s %s wasn't found.", kind, name)
	}
	return rec, nil
}

func (st *dataBrewStore) withTagsLocked(kind string, rec map[string]any) map[string]any {
	out := make(map[string]any, len(rec)+1)
	for k, v := range rec {
		out[k] = v
	}
	if tags := st.tags[dbwARN(kind, dbwStr(rec, "Name"))]; len(tags) > 0 {
		m := make(map[string]any, len(tags))
		for k, v := range tags {
			m[k] = v
		}
		out["Tags"] = m
	}
	return out
}

func (st *dataBrewStore) describe(kind, name string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	rec, err := st.getLocked(kind, name)
	if err != nil {
		return nil, err
	}
	out := st.withTagsLocked(kind, rec)
	if kind == "project" {
		if sess := st.sessions[name]; sess != nil {
			out["SessionStatus"] = "READY"
		}
	}
	return dbwShape(dbwKinds[kind].describe, out), nil
}

// list pages the records of kind (sorted by name) that keep returns true for.
func (st *dataBrewStore) list(kind string, q url.Values, keep func(map[string]any) bool) (any, *dbwError) {
	max, err := dbwMaxResults(q)
	if err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	names := make([]string, 0, len(st.recs[kind]))
	for name, rec := range st.recs[kind] {
		if keep == nil || keep(rec) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	start, end, next, err := dbwPage(len(names), q.Get("nextToken"), max)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, name := range names[start:end] {
		rec := st.withTagsLocked(kind, st.recs[kind][name])
		if kind == "ruleset" {
			rules, _ := rec["Rules"].([]any)
			rec["RuleCount"] = float64(len(rules))
		}
		items = append(items, dbwShape(dbwKinds[kind].item, rec))
	}
	out := map[string]any{dbwKinds[kind].listKey: items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func dbwPage(n int, token string, max int) (int, int, string, *dbwError) {
	start := 0
	if token != "" {
		v, err := strconv.ParseInt(token, 10, 32)
		if err != nil || v < 0 || int(v) > n {
			return 0, 0, "", dbwValidation("The pagination token is invalid.")
		}
		start = int(v)
	}
	end := min(n, start+max)
	next := ""
	if end < n {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

func (st *dataBrewStore) remove(kind, name string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.getLocked(kind, name); err != nil {
		return nil, err
	}
	st.dropLocked(kind, name)
	return map[string]any{"Name": name}, nil
}

func (st *dataBrewStore) dropLocked(kind, name string) {
	delete(st.recs[kind], name)
	delete(st.tags, dbwARN(kind, name))
}

// ---- tags ----

func (st *dataBrewStore) existsLocked(arn string) bool {
	kind, name, ok := dbwParseARN(arn)
	return ok && st.recs[kind][name] != nil
}

func (st *dataBrewStore) listTags(arn string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.existsLocked(arn) {
		return nil, dbwNotFound("The resource %s wasn't found.", arn)
	}
	tags := map[string]any{}
	for k, v := range st.tags[arn] {
		tags[k] = v
	}
	return map[string]any{"Tags": tags}, nil
}

func dbwCheckTags(tags map[string]string) *dbwError {
	for k := range tags {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return dbwValidation("Tag keys can't start with aws:.")
		}
	}
	if len(tags) > dbwMaxTags {
		return dbwValidation("A resource can have at most %d tags.", dbwMaxTags)
	}
	return nil
}

func (st *dataBrewStore) tag(arn string, tags map[string]string) *dbwError {
	if err := dbwCheckTags(tags); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.existsLocked(arn) {
		return dbwNotFound("The resource %s wasn't found.", arn)
	}
	merged := map[string]string{}
	for k, v := range st.tags[arn] {
		merged[k] = v
	}
	for k, v := range tags {
		merged[k] = v
	}
	if len(merged) > dbwMaxTags {
		return dbwValidation("A resource can have at most %d tags.", dbwMaxTags)
	}
	st.tags[arn] = merged
	return nil
}

func (st *dataBrewStore) untag(arn string, keys []string) *dbwError {
	if len(keys) == 0 {
		return dbwValidation("tagKeys is required.")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.existsLocked(arn) {
		return dbwNotFound("The resource %s wasn't found.", arn)
	}
	for _, k := range keys {
		delete(st.tags[arn], k)
	}
	return nil
}

// ---- datasets ----

func (st *dataBrewStore) putDataset(name string, in map[string]any, create bool) (any, *dbwError) {
	tags := dbwTagMap(in["Tags"])
	if err := dbwCheckTags(tags); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	source := "S3"
	input := dbwMap(in, "Input")
	switch {
	case input["DataCatalogInputDefinition"] != nil:
		source = "DATA-CATALOG"
	case input["DatabaseInputDefinition"] != nil:
		source = "DATABASE"
	}
	if create {
		if st.recs["dataset"][name] != nil {
			return nil, dbwConflict("A dataset named %s already exists.", name)
		}
		rec := map[string]any{}
		for _, k := range []string{"Format", "FormatOptions", "Input", "PathOptions"} {
			if v, ok := in[k]; ok {
				rec[k] = v
			}
		}
		rec["Source"] = source
		if err := st.newRecordLocked("dataset", name, rec, tags); err != nil {
			return nil, err
		}
		return map[string]any{"Name": name}, nil
	}
	rec, err := st.getLocked("dataset", name)
	if err != nil {
		return nil, err
	}
	next := map[string]any{}
	for k, v := range rec {
		next[k] = v
	}
	for _, k := range []string{"Format", "FormatOptions", "Input", "PathOptions"} {
		delete(next, k)
		if v, ok := in[k]; ok {
			next[k] = v
		}
	}
	next["Source"] = source
	dbwTouch(next)
	st.recs["dataset"][name] = next
	return map[string]any{"Name": name}, nil
}

func (st *dataBrewStore) deleteDataset(name string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.getLocked("dataset", name); err != nil {
		return nil, err
	}
	for pn, p := range st.recs["project"] {
		if dbwStr(p, "DatasetName") == name {
			return nil, dbwConflict("Dataset %s is used by project %s.", name, pn)
		}
	}
	for jn, j := range st.recs["job"] {
		if dbwStr(j, "DatasetName") == name {
			return nil, dbwConflict("Dataset %s is used by job %s.", name, jn)
		}
	}
	for rn, r := range st.recs["ruleset"] {
		if dbwStr(r, "TargetArn") == dbwARN("dataset", name) {
			return nil, dbwConflict("Dataset %s is the target of ruleset %s.", name, rn)
		}
	}
	st.dropLocked("dataset", name)
	return map[string]any{"Name": name}, nil
}

// ---- recipes ----

func dbwWorkingVersion(published int) string { return strconv.Itoa(published) + ".1" }

func (st *dataBrewStore) createRecipe(in map[string]any) (any, *dbwError) {
	name := dbwStr(in, "Name")
	tags := dbwTagMap(in["Tags"])
	if err := dbwCheckTags(tags); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.recs["recipe"][name] != nil {
		return nil, dbwConflict("A recipe named %s already exists.", name)
	}
	rec := map[string]any{"Steps": in["Steps"], "RecipeVersion": dbwWorkingVersion(0)}
	if d := dbwStr(in, "Description"); d != "" {
		rec["Description"] = d
	}
	if err := st.newRecordLocked("recipe", name, rec, tags); err != nil {
		return nil, err
	}
	return map[string]any{"Name": name}, nil
}

func (st *dataBrewStore) updateRecipe(name string, in map[string]any) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	rec, err := st.getLocked("recipe", name)
	if err != nil {
		return nil, err
	}
	next := map[string]any{}
	for k, v := range rec {
		next[k] = v
	}
	if v, ok := in["Steps"]; ok {
		next["Steps"] = v
	}
	if v, ok := in["Description"]; ok {
		next["Description"] = v
	}
	dbwTouch(next)
	st.recs["recipe"][name] = next
	return map[string]any{"Name": name}, nil
}

func (st *dataBrewStore) publishRecipe(name, description string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	rec, err := st.getLocked("recipe", name)
	if err != nil {
		return nil, err
	}
	versions := st.versions[name]
	if len(versions) >= dbwMaxVersions {
		return nil, dbwQuota("A recipe can have at most %d published versions.", dbwMaxVersions)
	}
	major := 1
	if len(versions) > 0 {
		last, _ := strconv.ParseFloat(dbwStr(versions[len(versions)-1], "RecipeVersion"), 64)
		major = int(last) + 1
	}
	pub := map[string]any{}
	for k, v := range rec {
		if k != "ProjectName" {
			pub[k] = v
		}
	}
	now := dbwNow()
	pub["RecipeVersion"] = strconv.Itoa(major) + ".0"
	pub["PublishedBy"], pub["PublishedDate"] = dbwCreatedBy, now
	if description != "" {
		pub["Description"] = description
	}
	st.versions[name] = append(versions, pub)
	next := map[string]any{}
	for k, v := range rec {
		next[k] = v
	}
	next["RecipeVersion"] = dbwWorkingVersion(major)
	st.recs["recipe"][name] = next
	return map[string]any{"Name": name}, nil
}

// recipeVersionLocked resolves a version selector: "" (latest published, else
// the working version), LATEST_WORKING, LATEST_PUBLISHED or an exact version.
func (st *dataBrewStore) recipeVersionLocked(name, version string) (map[string]any, *dbwError) {
	rec, err := st.getLocked("recipe", name)
	if err != nil {
		return nil, err
	}
	versions := st.versions[name]
	switch version {
	case "":
		// ponytail: AWS returns the latest published version by default; with
		// none published we return the working version instead of failing.
		if len(versions) == 0 {
			return rec, nil
		}
		return versions[len(versions)-1], nil
	case "LATEST_WORKING":
		return rec, nil
	case "LATEST_PUBLISHED":
		if len(versions) == 0 {
			return nil, dbwNotFound("Recipe %s has no published version.", name)
		}
		return versions[len(versions)-1], nil
	}
	if version == dbwStr(rec, "RecipeVersion") {
		return rec, nil
	}
	for _, v := range versions {
		if dbwStr(v, "RecipeVersion") == version {
			return v, nil
		}
	}
	return nil, dbwNotFound("Recipe %s has no version %s.", name, version)
}

func (st *dataBrewStore) describeRecipe(name, version string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.recipeVersionLocked(name, version)
	if err != nil {
		return nil, err
	}
	return dbwShape("DescribeRecipeResponse", st.withTagsLocked("recipe", v)), nil
}

func (st *dataBrewStore) listRecipes(q url.Values) (any, *dbwError) {
	max, err := dbwMaxResults(q)
	if err != nil {
		return nil, err
	}
	sel := q.Get("recipeVersion")
	if sel != "" && sel != "LATEST_WORKING" && sel != "LATEST_PUBLISHED" {
		return nil, dbwValidation("recipeVersion must be LATEST_WORKING or LATEST_PUBLISHED.")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	var items []map[string]any
	for name, rec := range st.recs["recipe"] {
		if sel == "LATEST_PUBLISHED" {
			if vs := st.versions[name]; len(vs) > 0 {
				items = append(items, vs[len(vs)-1])
			}
			continue
		}
		items = append(items, rec)
	}
	sort.Slice(items, func(i, j int) bool { return dbwStr(items[i], "Name") < dbwStr(items[j], "Name") })
	return st.recipePageLocked(items, q.Get("nextToken"), max)
}

func (st *dataBrewStore) listRecipeVersions(q url.Values) (any, *dbwError) {
	max, err := dbwMaxResults(q)
	if err != nil {
		return nil, err
	}
	name := q.Get("name")
	if name == "" {
		return nil, dbwValidation("name is required.")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.getLocked("recipe", name); err != nil {
		return nil, err
	}
	return st.recipePageLocked(st.versions[name], q.Get("nextToken"), max)
}

func (st *dataBrewStore) recipePageLocked(items []map[string]any, token string, max int) (any, *dbwError) {
	start, end, next, err := dbwPage(len(items), token, max)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		out = append(out, dbwShape("Recipe", st.withTagsLocked("recipe", item)))
	}
	resp := map[string]any{"Recipes": out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

// recipeVersionInUseLocked names the job that runs recipe name at version.
func (st *dataBrewStore) recipeVersionInUseLocked(name, version string) string {
	versions := st.versions[name]
	for jn, j := range st.recs["job"] {
		ref := dbwMap(j, "RecipeReference")
		if dbwStr(ref, "Name") != name {
			continue
		}
		v := dbwStr(ref, "RecipeVersion")
		if v == version || ((v == "" || v == "LATEST_PUBLISHED") && len(versions) > 0 && dbwStr(versions[len(versions)-1], "RecipeVersion") == version) {
			return jn
		}
	}
	return ""
}

// deleteVersionLocked deletes one version; LATEST_WORKING deletes the recipe
// and only when no published version is left.
func (st *dataBrewStore) deleteVersionLocked(name, version string) *dbwError {
	if version == "LATEST_WORKING" {
		if len(st.versions[name]) > 0 {
			return dbwConflict("LATEST_WORKING can be deleted only when recipe %s has no published versions.", name)
		}
		if p := dbwStr(st.recs["recipe"][name], "ProjectName"); p != "" {
			return dbwConflict("Recipe %s is used by project %s.", name, p)
		}
		st.dropLocked("recipe", name)
		delete(st.versions, name)
		return nil
	}
	versions := st.versions[name]
	for i, v := range versions {
		if dbwStr(v, "RecipeVersion") != version {
			continue
		}
		if job := st.recipeVersionInUseLocked(name, version); job != "" {
			return dbwConflict("Recipe %s version %s is used by job %s.", name, version, job)
		}
		st.versions[name] = append(versions[:i:i], versions[i+1:]...)
		return nil
	}
	return dbwNotFound("Recipe %s has no version %s.", name, version)
}

func (st *dataBrewStore) deleteRecipeVersion(name, version string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.getLocked("recipe", name); err != nil {
		return nil, err
	}
	if version == "LATEST_PUBLISHED" {
		return nil, dbwValidation("Specify a version number or LATEST_WORKING.")
	}
	if err := st.deleteVersionLocked(name, version); err != nil {
		return nil, err
	}
	return map[string]any{"Name": name, "RecipeVersion": version}, nil
}

func (st *dataBrewStore) batchDeleteRecipeVersions(name string, versions []string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.getLocked("recipe", name); err != nil {
		return nil, err
	}
	// Published versions first, so LATEST_WORKING can go once they're gone.
	sort.SliceStable(versions, func(i, j int) bool { return versions[j] == "LATEST_WORKING" && versions[i] != "LATEST_WORKING" })
	errs := []any{}
	for _, v := range versions {
		var err *dbwError
		if v == "LATEST_PUBLISHED" {
			err = dbwValidation("Specify a version number or LATEST_WORKING.")
		} else {
			err = st.deleteVersionLocked(name, v)
		}
		if err != nil {
			errs = append(errs, map[string]any{"ErrorCode": err.code, "ErrorMessage": err.msg, "RecipeVersion": v})
		}
	}
	return map[string]any{"Name": name, "Errors": errs}, nil
}

// ---- projects ----

func (st *dataBrewStore) createProject(in map[string]any) (any, *dbwError) {
	name := dbwStr(in, "Name")
	tags := dbwTagMap(in["Tags"])
	if err := dbwCheckTags(tags); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.recs["project"][name] != nil {
		return nil, dbwConflict("A project named %s already exists.", name)
	}
	ds, rn := dbwStr(in, "DatasetName"), dbwStr(in, "RecipeName")
	if st.recs["dataset"][ds] == nil {
		return nil, dbwValidation("Dataset %s doesn't exist.", ds)
	}
	recipe := st.recs["recipe"][rn]
	if recipe == nil {
		return nil, dbwValidation("Recipe %s doesn't exist.", rn)
	}
	if p := dbwStr(recipe, "ProjectName"); p != "" {
		return nil, dbwConflict("Recipe %s is already used by project %s.", rn, p)
	}
	sample := dbwMap(in, "Sample")
	if sample == nil {
		sample = map[string]any{"Type": "FIRST_N", "Size": float64(dbwDefaultSample)}
	}
	rec := map[string]any{"DatasetName": ds, "RecipeName": rn, "RoleArn": in["RoleArn"], "Sample": sample}
	if err := st.newRecordLocked("project", name, rec, tags); err != nil {
		return nil, err
	}
	next := map[string]any{}
	for k, v := range recipe {
		next[k] = v
	}
	next["ProjectName"] = name
	st.recs["recipe"][rn] = next
	return map[string]any{"Name": name}, nil
}

func (st *dataBrewStore) updateProject(name string, in map[string]any) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	rec, err := st.getLocked("project", name)
	if err != nil {
		return nil, err
	}
	next := map[string]any{}
	for k, v := range rec {
		next[k] = v
	}
	next["RoleArn"] = in["RoleArn"]
	if s := dbwMap(in, "Sample"); s != nil {
		next["Sample"] = s
	}
	dbwTouch(next)
	st.recs["project"][name] = next
	return map[string]any{"Name": name, "LastModifiedDate": next["LastModifiedDate"]}, nil
}

func (st *dataBrewStore) deleteProject(name string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	rec, err := st.getLocked("project", name)
	if err != nil {
		return nil, err
	}
	for jn, j := range st.recs["job"] {
		if dbwStr(j, "ProjectName") == name {
			return nil, dbwConflict("Project %s is used by job %s.", name, jn)
		}
	}
	if recipe := st.recs["recipe"][dbwStr(rec, "RecipeName")]; recipe != nil {
		next := map[string]any{}
		for k, v := range recipe {
			if k != "ProjectName" {
				next[k] = v
			}
		}
		st.recs["recipe"][dbwStr(rec, "RecipeName")] = next
	}
	delete(st.sessions, name)
	st.dropLocked("project", name)
	return map[string]any{"Name": name}, nil
}

func (st *dataBrewStore) startSession(name string, assume bool) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	rec, err := st.getLocked("project", name)
	if err != nil {
		return nil, err
	}
	if st.sessions[name] != nil && !assume {
		return nil, dbwConflict("A session is already open for project %s; set AssumeControl to take it over.", name)
	}
	sess := &dbwSession{id: "session-" + dbwRandomID(12)}
	st.sessions[name] = sess
	next := map[string]any{}
	for k, v := range rec {
		next[k] = v
	}
	next["OpenedBy"], next["OpenDate"] = dbwCreatedBy, dbwNow()
	st.recs["project"][name] = next
	return map[string]any{"Name": name, "ClientSessionId": sess.id}, nil
}

// ---- schedules ----

var dbwCronExpr = regexp.MustCompile(`^cron\(.+\)$`)

func (st *dataBrewStore) putSchedule(name string, in map[string]any) (any, *dbwError) {
	create := name == ""
	if create {
		name = dbwStr(in, "Name")
	}
	cron := dbwStr(in, "CronExpression")
	if !dbwCronExpr.MatchString(cron) {
		return nil, dbwValidation("CronExpression must be a cron() expression.")
	}
	if err := schedulerValidateExpression(cron, ""); err != nil {
		return nil, dbwValidation("%s", err.msg)
	}
	jobs := dbwStrings(in["JobNames"])
	tags := dbwTagMap(in["Tags"])
	if err := dbwCheckTags(tags); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, j := range jobs {
		if st.recs["job"][j] == nil {
			return nil, dbwValidation("Job %s doesn't exist.", j)
		}
	}
	jobList := make([]any, 0, len(jobs))
	for _, j := range jobs {
		jobList = append(jobList, j)
	}
	if create {
		if st.recs["schedule"][name] != nil {
			return nil, dbwConflict("A schedule named %s already exists.", name)
		}
		if err := st.newRecordLocked("schedule", name, map[string]any{"CronExpression": cron, "JobNames": jobList}, tags); err != nil {
			return nil, err
		}
		return map[string]any{"Name": name}, nil
	}
	rec, err := st.getLocked("schedule", name)
	if err != nil {
		return nil, err
	}
	next := map[string]any{}
	for k, v := range rec {
		next[k] = v
	}
	next["CronExpression"], next["JobNames"] = cron, jobList
	dbwTouch(next)
	st.recs["schedule"][name] = next
	return map[string]any{"Name": name}, nil
}

func (st *dataBrewStore) scheduleFilter(q url.Values) func(map[string]any) bool {
	job := q.Get("jobName")
	if job == "" {
		return nil
	}
	return func(rec map[string]any) bool {
		for _, j := range dbwStrings(rec["JobNames"]) {
			if j == job {
				return true
			}
		}
		return false
	}
}

// ---- rulesets ----

var dbwPlaceholder = regexp.MustCompile(`:[A-Za-z0-9_]+`)

// dbwCheckRules checks rule names are unique and every :placeholder in a
// CheckExpression is defined in its SubstitutionMap.
func dbwCheckRules(rules []any) *dbwError {
	seen := map[string]bool{}
	for _, r := range rules {
		rule, _ := r.(map[string]any)
		name := dbwStr(rule, "Name")
		if seen[name] {
			return dbwValidation("Rule names must be unique; %s is repeated.", name)
		}
		seen[name] = true
		subs := dbwMap(rule, "SubstitutionMap")
		for k := range subs {
			if !strings.HasPrefix(k, ":") {
				return dbwValidation("Rule %s: SubstitutionMap keys must start with ':'.", name)
			}
		}
		for _, ph := range dbwPlaceholder.FindAllString(dbwStr(rule, "CheckExpression"), 100) {
			if _, ok := subs[ph]; !ok {
				return dbwValidation("Rule %s: %s in CheckExpression isn't defined in SubstitutionMap.", name, ph)
			}
		}
	}
	return nil
}

func (st *dataBrewStore) putRuleset(name string, in map[string]any) (any, *dbwError) {
	create := name == ""
	if create {
		name = dbwStr(in, "Name")
	}
	rules, _ := in["Rules"].([]any)
	if err := dbwCheckRules(rules); err != nil {
		return nil, err
	}
	tags := dbwTagMap(in["Tags"])
	if err := dbwCheckTags(tags); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if create {
		if st.recs["ruleset"][name] != nil {
			return nil, dbwConflict("A ruleset named %s already exists.", name)
		}
		target := dbwStr(in, "TargetArn")
		if kind, ds, ok := dbwParseARN(target); !ok || kind != "dataset" || st.recs["dataset"][ds] == nil {
			return nil, dbwValidation("TargetArn %s isn't an existing DataBrew dataset.", target)
		}
		rec := map[string]any{"TargetArn": target, "Rules": rules}
		if d := dbwStr(in, "Description"); d != "" {
			rec["Description"] = d
		}
		if err := st.newRecordLocked("ruleset", name, rec, tags); err != nil {
			return nil, err
		}
		return map[string]any{"Name": name}, nil
	}
	rec, err := st.getLocked("ruleset", name)
	if err != nil {
		return nil, err
	}
	next := map[string]any{}
	for k, v := range rec {
		next[k] = v
	}
	next["Rules"] = rules
	if v, ok := in["Description"]; ok {
		next["Description"] = v
	}
	dbwTouch(next)
	st.recs["ruleset"][name] = next
	return map[string]any{"Name": name}, nil
}

func (st *dataBrewStore) rulesetFilter(q url.Values) func(map[string]any) bool {
	target := q.Get("targetArn")
	if target == "" {
		return nil
	}
	return func(rec map[string]any) bool { return dbwStr(rec, "TargetArn") == target }
}

func (st *dataBrewStore) deleteRuleset(name string) (any, *dbwError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.getLocked("ruleset", name); err != nil {
		return nil, err
	}
	arn := dbwARN("ruleset", name)
	for jn, j := range st.recs["job"] {
		for _, vc := range dbwList(j, "ValidationConfigurations") {
			if dbwStr(vc, "RulesetArn") == arn {
				return nil, dbwConflict("Ruleset %s is used by job %s.", name, jn)
			}
		}
	}
	st.dropLocked("ruleset", name)
	return map[string]any{"Name": name}, nil
}

func dbwList(m map[string]any, key string) []map[string]any {
	items, _ := m[key].([]any)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if obj, ok := item.(map[string]any); ok {
			out = append(out, obj)
		}
	}
	return out
}
