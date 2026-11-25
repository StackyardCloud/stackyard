package server

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Dataset groups, schemas, datasets, event trackers, filters and metric
// attributions. Solutions, campaigns and recommenders are in
// personalize_models.go; jobs in personalize_jobs.go.

func pszShortID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// dataset returns the dataset of typ in a dataset group, or nil.
func (st *personalizeStore) dataset(groupARN, typ string) *pszRes {
	g := st.res[groupARN]
	if g == nil {
		return nil
	}
	return st.res[pszARNBase+"dataset/"+g.fields["name"].(string)+"/"+strings.ToUpper(typ)]
}

func (st *personalizeStore) createDatasetGroup(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	name := pszStr(in, "name")
	arn := pszARNBase + "dataset-group/" + name
	if err := st.exists(arn); err != nil {
		return nil, err
	}
	if pszStr(in, "kmsKeyArn") != "" && pszStr(in, "roleArn") == "" {
		return nil, pszInvalid("A roleArn is required when kmsKeyArn is set.")
	}
	fields := map[string]any{"name": name}
	for _, k := range []string{"roleArn", "kmsKeyArn", "domain"} {
		if v := pszStr(in, k); v != "" {
			fields[k] = v
		}
	}
	if _, err := st.add("datasetGroup", arn, fields, nil, pszTags(in), time.Now().UTC()); err != nil {
		return nil, err
	}
	out := map[string]any{"datasetGroupArn": arn}
	if d := pszStr(in, "domain"); d != "" {
		out["domain"] = d
	}
	return out, nil
}

func (st *personalizeStore) createSchema(in map[string]any) (map[string]any, *pszErr) {
	fields, perr := pszParseSchema(pszStr(in, "schema"))
	if perr != nil {
		return nil, perr
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	name := pszStr(in, "name")
	arn := pszARNBase + "schema/" + name
	if err := st.exists(arn); err != nil {
		return nil, err
	}
	f := map[string]any{"name": name, "schema": pszStr(in, "schema")}
	if d := pszStr(in, "domain"); d != "" {
		f["domain"] = d
	}
	r, err := st.add("schema", arn, f, nil, nil, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	r.status, r.next, r.schemaFields = pszStatusActive, nil, fields
	return map[string]any{"schemaArn": arn}, nil
}

// checkSchema returns the schema at arn if it suits a dataset of typ in group g.
func (st *personalizeStore) checkSchema(arn, typ string, g *pszRes) (*pszRes, *pszErr) {
	sc, err := st.get("schema", arn)
	if err != nil {
		return nil, err
	}
	if missing := pszLacks(sc.schemaFields, pszDatasetTypes[strings.ToLower(typ)].required); missing != "" {
		return nil, pszInvalid("A %s dataset schema needs a %s field.", typ, missing)
	}
	if sc.fields["domain"] != g.fields["domain"] {
		return nil, pszInvalid("The schema domain must match the dataset group domain.")
	}
	return sc, nil
}

func (st *personalizeStore) createDataset(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	t, ok := pszDatasetTypes[strings.ToLower(pszStr(in, "datasetType"))]
	if !ok {
		return nil, pszInvalid("datasetType must be Interactions, Items, Users, Actions or Action_Interactions.")
	}
	g, err := st.active("datasetGroup", pszStr(in, "datasetGroupArn"))
	if err != nil {
		return nil, err
	}
	sc, err := st.checkSchema(pszStr(in, "schemaArn"), t.name, g)
	if err != nil {
		return nil, err
	}
	arn := pszARNBase + "dataset/" + g.fields["name"].(string) + "/" + strings.ToUpper(t.name)
	if err := st.exists(arn); err != nil {
		return nil, err
	}
	datasets := 0
	for _, r := range st.res {
		if r.kind.id == "dataset" {
			datasets++
		}
	}
	if datasets >= pszMaxDatasets {
		return nil, pszError(pszErrLimit, "Stackyard keeps at most %d Personalize datasets.", pszMaxDatasets)
	}
	r, err := st.add("dataset", arn, map[string]any{
		"name": pszStr(in, "name"), "datasetGroupArn": g.arn, "datasetType": t.name, "schemaArn": sc.arn,
	}, map[string]string{"datasetGroupArn": g.arn}, pszTags(in), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	r.data = newPSZData(t.name, sc.schemaFields)
	return map[string]any{"datasetArn": arn}, nil
}

func (st *personalizeStore) updateDataset(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.active("dataset", pszStr(in, "datasetArn"))
	if err != nil {
		return nil, err
	}
	if r.updating() {
		return nil, pszError(pszErrInUse, "%s has an update in progress.", r.arn)
	}
	sc, err := st.checkSchema(pszStr(in, "schemaArn"), r.data.typ, st.res[r.rel["datasetGroupArn"]])
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	r.fields = pszWith(r.fields, "schemaArn", sc.arn)
	r.data.fields = sc.schemaFields
	r.startUpdate("latestDatasetUpdate", map[string]any{"schemaArn": sc.arn}, now)
	return map[string]any{"datasetArn": r.arn}, nil
}

// pszWith returns a copy of m with k set to v (stored maps aren't mutated).
func pszWith(m map[string]any, k string, v any) map[string]any {
	out := make(map[string]any, len(m)+1)
	for key, val := range m {
		out[key] = val
	}
	if v == nil {
		delete(out, k)
	} else {
		out[k] = v
	}
	return out
}

func (st *personalizeStore) createEventTracker(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.active("datasetGroup", pszStr(in, "datasetGroupArn"))
	if err != nil {
		return nil, err
	}
	name := pszStr(in, "name")
	for _, r := range st.res {
		if r.kind.id == "eventTracker" && r.fields["name"] == name {
			return nil, pszError(pszErrExists, "Event tracker %s already exists.", name)
		}
	}
	if st.trackerFor(g.arn) != nil {
		return nil, pszError(pszErrLimit, "A dataset group can have one event tracker.")
	}
	arn := pszARNBase + "event-tracker/" + pszShortID()
	trackingID := uuid.NewString()
	if _, err := st.add("eventTracker", arn, map[string]any{
		"name": name, "accountId": pszAccount, "trackingId": trackingID, "datasetGroupArn": g.arn,
	}, map[string]string{"datasetGroupArn": g.arn}, pszTags(in), time.Now().UTC()); err != nil {
		return nil, err
	}
	st.trackers[trackingID] = arn
	return map[string]any{"eventTrackerArn": arn, "trackingId": trackingID}, nil
}

func (st *personalizeStore) createFilter(in map[string]any) (map[string]any, *pszErr) {
	f, perr := pszParseFilter(pszStr(in, "filterExpression"))
	if perr != nil {
		return nil, perr
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.get("datasetGroup", pszStr(in, "datasetGroupArn"))
	if err != nil {
		return nil, err
	}
	if err := st.checkFilterDatasets(f, g.arn); err != nil {
		return nil, err
	}
	name := pszStr(in, "name")
	arn := pszARNBase + "filter/" + name
	if err := st.exists(arn); err != nil {
		return nil, err
	}
	count := 0
	for _, r := range st.res {
		if r.kind.id == "filter" && r.rel["datasetGroupArn"] == g.arn {
			count++
		}
	}
	if count >= pszMaxFilters {
		return nil, pszError(pszErrLimit, "A dataset group can have at most %d filters.", pszMaxFilters)
	}
	r, err := st.add("filter", arn, map[string]any{
		"name": name, "datasetGroupArn": g.arn, "filterExpression": pszStr(in, "filterExpression"),
	}, map[string]string{"datasetGroupArn": g.arn}, pszTags(in), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	r.filter = f
	return map[string]any{"filterArn": arn}, nil
}

// checkFilterDatasets requires each dataset and field a filter names.
func (st *personalizeStore) checkFilterDatasets(f *pszFilter, groupARN string) *pszErr {
	for _, c := range f.conds {
		d := st.dataset(groupARN, c.dataset)
		if d == nil {
			return pszInvalid("The filter uses the %s dataset, which the dataset group doesn't have.", c.dataset)
		}
		if pszField(d.data.fields, c.field) == "" {
			return pszInvalid("%s isn't a field of the %s dataset.", c.field, c.dataset)
		}
	}
	return nil
}

// pszFilterFor returns the ACTIVE filter at arn, which must belong to groupARN.
func (st *personalizeStore) filterFor(arn, groupARN string) (*pszFilter, *pszErr) {
	if arn == "" {
		return nil, nil
	}
	r, err := st.active("filter", arn)
	if err != nil {
		return nil, err
	}
	if r.rel["datasetGroupArn"] != groupARN {
		return nil, pszInvalid("Filter %s belongs to another dataset group.", arn)
	}
	return r.filter, nil
}

var pszMetricExpr = regexp.MustCompile(`^(SUM\(Items\.([A-Za-z_][A-Za-z0-9_]*)\)|SAMPLECOUNT\(\))$`)

func (st *personalizeStore) checkMetrics(groupARN string, metrics []any, have map[string]bool) *pszErr {
	for _, item := range metrics {
		m, _ := item.(map[string]any)
		name, expr := pszStr(m, "metricName"), pszStr(m, "expression")
		if have[name] {
			return pszError(pszErrExists, "Metric %s already exists.", name)
		}
		have[name] = true
		match := pszMetricExpr.FindStringSubmatch(expr)
		if match == nil {
			return pszInvalid("Metric expression %q must be SUM(Items.<column>) or SAMPLECOUNT().", expr)
		}
		if match[2] != "" {
			items := st.dataset(groupARN, pszItems)
			if items == nil || pszField(items.data.fields, match[2]) == "" {
				return pszInvalid("%s isn't a field of the dataset group's Items dataset.", match[2])
			}
		}
	}
	if len(have) > 10 {
		return pszInvalid("A metric attribution can have at most 10 metrics.")
	}
	return nil
}

func (st *personalizeStore) createMetricAttribution(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.active("datasetGroup", pszStr(in, "datasetGroupArn"))
	if err != nil {
		return nil, err
	}
	for _, r := range st.res {
		if r.kind.id == "metricAttribution" && r.rel["datasetGroupArn"] == g.arn {
			return nil, pszError(pszErrExists, "Dataset group %s already has metric attribution %s.", g.arn, r.arn)
		}
	}
	arn := pszARNBase + "metric-attribution/" + pszStr(in, "name")
	if err := st.exists(arn); err != nil {
		return nil, err
	}
	metrics := pszList(in, "metrics")
	if err := st.checkMetrics(g.arn, metrics, map[string]bool{}); err != nil {
		return nil, err
	}
	r, err := st.add("metricAttribution", arn, map[string]any{
		"name": pszStr(in, "name"), "datasetGroupArn": g.arn, "metricsOutputConfig": pszMap(in, "metricsOutputConfig"),
	}, map[string]string{"datasetGroupArn": g.arn}, nil, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	r.metrics = metrics
	return map[string]any{"metricAttributionArn": arn}, nil
}

func (st *personalizeStore) updateMetricAttribution(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	arn := pszStr(in, "metricAttributionArn")
	if arn == "" {
		return nil, pszInvalid("metricAttributionArn is required.")
	}
	r, err := st.active("metricAttribution", arn)
	if err != nil {
		return nil, err
	}
	remove := map[string]bool{}
	for _, name := range pszStrings(in["removeMetrics"]) {
		remove[name] = true
	}
	kept, have := []any{}, map[string]bool{}
	for _, item := range r.metrics {
		name := pszStr(item.(map[string]any), "metricName")
		if remove[name] {
			delete(remove, name)
			continue
		}
		kept, have[name] = append(kept, item), true
	}
	for name := range remove {
		return nil, pszInvalid("Metric %s isn't part of %s.", name, arn)
	}
	add := pszList(in, "addMetrics")
	if err := st.checkMetrics(r.rel["datasetGroupArn"], add, have); err != nil {
		return nil, err
	}
	if cfg := pszMap(in, "metricsOutputConfig"); cfg != nil {
		r.fields = pszWith(r.fields, "metricsOutputConfig", cfg)
	}
	r.metrics = append(kept, add...)
	r.status, r.next, r.updated = pszStatusUpdate, []string{pszStatusActive}, time.Now().UTC()
	return map[string]any{"metricAttributionArn": arn}, nil
}

func (st *personalizeStore) listMetricAttributionMetrics(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	arn := pszStr(in, "metricAttributionArn")
	if arn == "" {
		return nil, pszInvalid("metricAttributionArn is required.")
	}
	r, err := st.get("metricAttribution", arn)
	if err != nil {
		return nil, err
	}
	page, next, err := pszPage(len(r.metrics), pszStr(in, "nextToken"), pszMax(in))
	if err != nil {
		return nil, err
	}
	out := map[string]any{"metrics": st.model.Conform("MetricAttributes", append([]any(nil), r.metrics[page[0]:page[1]]...))}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}
