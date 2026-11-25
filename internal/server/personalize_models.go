package server

import (
	"regexp"
	"strconv"
	"time"
)

// Solutions, solution versions, campaigns and recommenders.

var pszRate = regexp.MustCompile(`^rate\((\d+) days?\)$`)

func pszCheckAutoTraining(cfg map[string]any) *pszErr {
	expr := pszStr(pszMap(cfg, "autoTrainingConfig"), "schedulingExpression")
	if expr == "" {
		return nil
	}
	if m := pszRate.FindStringSubmatch(expr); m != nil {
		if n, err := strconv.ParseInt(m[1], 10, 32); err == nil && n >= 1 && n <= 30 {
			return nil
		}
	}
	return pszInvalid("schedulingExpression must be rate(<1-30> days).")
}

// requiredDatasets lists the datasets a recipe trains on.
func pszRequiredDatasets(p *pszRecipe) []string {
	if p.recipeType == pszPersonalizedActions {
		return []string{pszActions, pszActionInteractions}
	}
	return []string{pszInteractions}
}

func (st *personalizeStore) createSolution(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.active("datasetGroup", pszStr(in, "datasetGroupArn"))
	if err != nil {
		return nil, err
	}
	name := pszStr(in, "name")
	arn := pszARNBase + "solution/" + name
	if err := st.exists(arn); err != nil {
		return nil, err
	}
	cfg := pszMap(in, "solutionConfig")
	if err := pszCheckAutoTraining(cfg); err != nil {
		return nil, err
	}
	autoML := pszBool(in, "performAutoML", false)
	recipeARN := pszStr(in, "recipeArn")
	fields := map[string]any{
		"name": name, "datasetGroupArn": g.arn, "performHPO": pszBool(in, "performHPO", false), "performAutoML": autoML,
		"performAutoTraining": pszBool(in, "performAutoTraining", true), "performIncrementalUpdate": pszBool(in, "performIncrementalUpdate", false),
	}
	var recipe *pszRecipe
	switch {
	case autoML && recipeARN != "":
		return nil, pszInvalid("Specify recipeArn or performAutoML, not both.")
	case autoML:
		best := pszCatalogBase + "recipe/aws-user-personalization"
		if list := pszStrings(pszMap(cfg, "autoMLConfig")["recipeList"]); len(list) > 0 {
			best = list[0]
		}
		if recipe = pszFindRecipe(best); recipe == nil || recipe.recipeType != pszUserPersonalization || recipe.domain != "" {
			return nil, pszInvalid("AutoML candidates must be USER_PERSONALIZATION recipes.")
		}
		fields["autoMLResult"] = map[string]any{"bestRecipeArn": best}
	default:
		if recipe = pszFindRecipe(recipeARN); recipe == nil || recipe.domain != "" {
			return nil, pszInvalid("recipeArn must name a custom (non-domain) Amazon Personalize recipe.")
		}
		fields["recipeArn"] = recipeARN
	}
	if v := pszStr(in, "eventType"); v != "" {
		fields["eventType"] = v
	}
	if cfg != nil {
		fields["solutionConfig"] = cfg
	}
	r, err := st.add("solution", arn, fields, map[string]string{"datasetGroupArn": g.arn}, pszTags(in), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	r.recipe = recipe
	return map[string]any{"solutionArn": arn}, nil
}

func (st *personalizeStore) updateSolution(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.active("solution", pszStr(in, "solutionArn"))
	if err != nil {
		return nil, err
	}
	if r.updating() {
		return nil, pszError(pszErrInUse, "%s has an update in progress.", r.arn)
	}
	upd := pszMap(in, "solutionUpdateConfig")
	if err := pszCheckAutoTraining(upd); err != nil {
		return nil, err
	}
	summary := map[string]any{}
	fields := r.fields
	for _, k := range []string{"performAutoTraining", "performIncrementalUpdate"} {
		if v, ok := in[k].(bool); ok {
			fields, summary[k] = pszWith(fields, k, v), v
		}
	}
	if upd != nil {
		summary["solutionUpdateConfig"] = upd
		cfg := map[string]any{}
		for k, v := range pszMap(fields, "solutionConfig") {
			cfg[k] = v
		}
		for _, k := range []string{"autoTrainingConfig", "eventsConfig"} {
			if v, ok := upd[k]; ok {
				cfg[k] = v
			}
		}
		fields = pszWith(fields, "solutionConfig", cfg)
	}
	r.fields = fields
	r.startUpdate("latestSolutionUpdate", summary, time.Now().UTC())
	return map[string]any{"solutionArn": r.arn}, nil
}

func (st *personalizeStore) createSolutionVersion(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	sol, err := st.active("solution", pszStr(in, "solutionArn"))
	if err != nil {
		return nil, err
	}
	mode := pszStr(in, "trainingMode")
	if mode == "" {
		mode = "FULL"
	}
	if mode == "AUTOTRAIN" {
		return nil, pszInvalid("AUTOTRAIN versions are created by automatic training, not by CreateSolutionVersion.")
	}
	name := pszStr(in, "name")
	hasActive := false
	for _, v := range st.res {
		if v.kind.id != "solutionVersion" || v.rel["solutionArn"] != sol.arn {
			continue
		}
		if v.status == pszStatusCreate || v.status == "CREATE PENDING" {
			return nil, pszError(pszErrInUse, "Solution version %s is still being created.", v.arn)
		}
		if name != "" && v.fields["name"] == name {
			return nil, pszError(pszErrExists, "Solution version %s already exists.", name)
		}
		hasActive = hasActive || v.status == pszStatusActive
	}
	if mode == "UPDATE" && (!hasActive || sol.recipe.name != "aws-user-personalization") {
		return nil, pszInvalid("trainingMode UPDATE needs an ACTIVE aws-user-personalization solution version to update.")
	}
	group := sol.rel["datasetGroupArn"]
	for _, typ := range pszRequiredDatasets(sol.recipe) {
		if d := st.dataset(group, typ); d == nil || d.status != pszStatusActive {
			return nil, pszError(pszErrInUse, "Training %s needs an ACTIVE %s dataset in %s.", sol.recipe.name, typ, group)
		}
	}
	fields := map[string]any{
		"solutionArn": sol.arn, "datasetGroupArn": group, "recipeArn": sol.recipe.arn(), "trainingMode": mode,
		"trainingType": "MANUAL", "trainingHours": pszRound(0.2 + pszUnit(sol.arn, name)),
	}
	if name != "" {
		fields["name"] = name
	}
	for _, k := range []string{"performHPO", "performAutoML", "performIncrementalUpdate", "eventType", "solutionConfig"} {
		if v, ok := sol.fields[k]; ok {
			fields[k] = v
		}
	}
	if sol.fields["performHPO"] == true {
		fields["tunedHPOParams"] = map[string]any{"algorithmHyperParameters": map[string]any{"hidden_dimension": "112", "bptt": "32"}}
	}
	arn := sol.arn + "/" + pszShortID()
	r, err := st.add("solutionVersion", arn, fields, map[string]string{"solutionArn": sol.arn, "datasetGroupArn": group}, pszTags(in), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	r.recipe = sol.recipe
	return map[string]any{"solutionVersionArn": arn}, nil
}

func (st *personalizeStore) stopSolutionVersion(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.get("solutionVersion", pszStr(in, "solutionVersionArn"))
	if err != nil {
		return nil, err
	}
	if r.status != pszStatusCreate && r.status != "CREATE PENDING" {
		return nil, pszError(pszErrInUse, "Solution version %s is %s; only versions being created can be stopped.", r.arn, r.status)
	}
	r.status, r.next, r.updated = "CREATE STOPPING", []string{"CREATE STOPPED"}, time.Now().UTC()
	return nil, nil
}

func (st *personalizeStore) solutionMetrics(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.active("solutionVersion", pszStr(in, "solutionVersionArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"solutionVersionArn": r.arn, "metrics": pszModelMetrics(r.arn)}, nil
}

// servingVersion returns an ACTIVE solution version a campaign can serve.
func (st *personalizeStore) servingVersion(arn string) (*pszRes, *pszErr) {
	v, err := st.active("solutionVersion", arn)
	if err != nil {
		return nil, err
	}
	if v.recipe.recipeType == pszUserSegmentation {
		return nil, pszInvalid("%s recipes produce batch segments; they can't back a campaign.", v.recipe.name)
	}
	return v, nil
}

func (st *personalizeStore) createCampaign(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v, err := st.servingVersion(pszStr(in, "solutionVersionArn"))
	if err != nil {
		return nil, err
	}
	name := pszStr(in, "name")
	arn := pszARNBase + "campaign/" + name
	if err := st.exists(arn); err != nil {
		return nil, err
	}
	fields := map[string]any{"name": name, "solutionVersionArn": v.arn, "minProvisionedTPS": pszInt(in, "minProvisionedTPS", 1)}
	if cfg := pszMap(in, "campaignConfig"); cfg != nil {
		fields["campaignConfig"] = cfg
	}
	r, err := st.add("campaign", arn, fields, map[string]string{"solutionArn": v.rel["solutionArn"], "datasetGroupArn": v.rel["datasetGroupArn"]}, pszTags(in), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	r.recipe = v.recipe
	return map[string]any{"campaignArn": arn}, nil
}

func (st *personalizeStore) updateCampaign(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.active("campaign", pszStr(in, "campaignArn"))
	if err != nil {
		return nil, err
	}
	if r.updating() {
		return nil, pszError(pszErrInUse, "%s has an update in progress.", r.arn)
	}
	fields, summary := r.fields, map[string]any{}
	if arn := pszStr(in, "solutionVersionArn"); arn != "" {
		v, err := st.servingVersion(arn)
		if err != nil {
			return nil, err
		}
		if v.recipe.recipeType != r.recipe.recipeType {
			return nil, pszInvalid("The new solution version must use a %s recipe.", r.recipe.recipeType)
		}
		fields, summary["solutionVersionArn"] = pszWith(fields, "solutionVersionArn", arn), arn
		r.rel = map[string]string{"solutionArn": v.rel["solutionArn"], "datasetGroupArn": v.rel["datasetGroupArn"]}
		r.recipe = v.recipe
	}
	if _, ok := in["minProvisionedTPS"]; ok {
		n := pszInt(in, "minProvisionedTPS", 1)
		fields, summary["minProvisionedTPS"] = pszWith(fields, "minProvisionedTPS", n), n
	}
	if cfg := pszMap(in, "campaignConfig"); cfg != nil {
		fields, summary["campaignConfig"] = pszWith(fields, "campaignConfig", cfg), cfg
	}
	r.fields = fields
	r.startUpdate("latestCampaignUpdate", summary, time.Now().UTC())
	return map[string]any{"campaignArn": r.arn}, nil
}

func (st *personalizeStore) createRecommender(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.active("datasetGroup", pszStr(in, "datasetGroupArn"))
	if err != nil {
		return nil, err
	}
	domain, _ := g.fields["domain"].(string)
	if domain == "" {
		return nil, pszInvalid("Recommenders need a Domain dataset group; %s has no domain.", g.arn)
	}
	recipe := pszFindRecipe(pszStr(in, "recipeArn"))
	if recipe == nil || recipe.domain != domain {
		return nil, pszInvalid("recipeArn must be a %s use case recipe.", domain)
	}
	if d := st.dataset(g.arn, pszInteractions); d == nil || d.status != pszStatusActive {
		return nil, pszInvalid("Recommenders need an ACTIVE Interactions dataset in %s.", g.arn)
	}
	name := pszStr(in, "name")
	arn := pszARNBase + "recommender/" + name
	if err := st.exists(arn); err != nil {
		return nil, err
	}
	cfg := map[string]any{"minRecommendationRequestsPerSecond": 1}
	for k, v := range pszMap(in, "recommenderConfig") {
		cfg[k] = v
	}
	r, err := st.add("recommender", arn, map[string]any{
		"name": name, "datasetGroupArn": g.arn, "recipeArn": recipe.arn(), "recommenderConfig": cfg,
	}, map[string]string{"datasetGroupArn": g.arn}, pszTags(in), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	r.recipe = recipe
	return map[string]any{"recommenderArn": arn}, nil
}

func (st *personalizeStore) updateRecommender(in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.active("recommender", pszStr(in, "recommenderArn"))
	if err != nil {
		return nil, err
	}
	if r.updating() {
		return nil, pszError(pszErrInUse, "%s has an update in progress.", r.arn)
	}
	cfg := map[string]any{}
	for k, v := range pszMap(r.fields, "recommenderConfig") {
		cfg[k] = v
	}
	for k, v := range pszMap(in, "recommenderConfig") {
		cfg[k] = v
	}
	r.fields = pszWith(r.fields, "recommenderConfig", cfg)
	r.startUpdate("latestRecommenderUpdate", map[string]any{"recommenderConfig": cfg}, time.Now().UTC())
	return map[string]any{"recommenderArn": r.arn}, nil
}

func (st *personalizeStore) startStopRecommender(arn string, start bool) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.get("recommender", arn)
	if err != nil {
		return nil, err
	}
	switch {
	case start && r.status == "INACTIVE":
		r.status, r.next = "START IN_PROGRESS", []string{pszStatusActive}
	case !start && r.status == pszStatusActive:
		r.status, r.next = "STOP IN_PROGRESS", []string{"INACTIVE"}
	default:
		want := pszStatusActive
		if start {
			want = "INACTIVE"
		}
		return nil, pszError(pszErrInUse, "Recommender %s is %s; it must be %s.", arn, r.status, want)
	}
	r.updated = time.Now().UTC()
	return map[string]any{"recommenderArn": arn}, nil
}
