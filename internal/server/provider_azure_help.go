package server

import (
	"maps"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"
)

// Microsoft.Help: diagnostics, solutions, simplified solutions and
// troubleshooters attached to any scope, plus discovery of what applies to
// a problem. Everything runs against the emulator's catalog (see
// provider_azure_help_catalog.go); diagnostics report the catalog's
// insights without inspecting the scope.
const azureHelpNamespace = "Microsoft.Help"

func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureHelpNamespace, Type: "diagnostics", Extension: true, NoPatch: true, NoDelete: true,
		Validate:           azureHelpDiagnosticsValidate,
		ReadOnlyProperties: azureHelpDiagnosticsRun,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureHelpNamespace, Type: "solutions", Extension: true, NoDelete: true,
		Validate:           azureHelpSolutionValidate,
		PatchBody:          azureHelpSolutionPatch,
		ReadOnlyProperties: azureHelpSolutionRender,
		Actions: map[string]azureARMAction{
			"warmup": func(_ *azureARMResource, payload map[string]any) (int, any) {
				if msg := azureHelpParameters(payload, "", "parameters"); msg != "" {
					return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
				}
				return http.StatusNoContent, nil
			},
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureHelpNamespace, Type: "simplifiedSolutions", Extension: true, NoPatch: true, NoDelete: true,
		Validate: func(payload map[string]any) string { return azureHelpSolutionIDValidate(payload, "Solutions") },
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			sol := azureHelpFindOf("Solutions", azureARMString(rec.Properties["solutionId"]))
			if sol == nil {
				return nil
			}
			content := sol.content
			if i := strings.Index(content, "<!--"); i >= 0 {
				content = content[:i]
			}
			return map[string]any{"title": sol.title, "content": content, "provisioningState": "Succeeded"}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureHelpNamespace, Type: "troubleshooters", Extension: true, NoPatch: true, NoDelete: true,
		Validate:        func(payload map[string]any) string { return azureHelpSolutionIDValidate(payload, "Troubleshooters") },
		StateProperties: map[string]any{"steps": nil, "provisioningState": nil},
		Reconcile: func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
			if existing != nil {
				if !strings.EqualFold(azureARMString(existing.Properties["solutionId"]), azureARMString(rec.Properties["solutionId"])) {
					return nil, "properties.solutionId can't be updated; restart the troubleshooter or create another."
				}
				return nil, ""
			}
			return azureHelpTroubleshooterStart(rec), ""
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"continue": azureHelpTroubleshooterContinue,
			"end": func(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
				return s.azureHelpTroubleshooterUpdate(rec, func(cur *azureARMResource) (map[string]any, int, any) {
					if cur.Properties["provisioningState"] != "Running" {
						return nil, http.StatusNoContent, nil
					}
					return map[string]any{"provisioningState": "Canceled"}, http.StatusNoContent, nil
				})
			},
			"restart": func(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
				return s.azureHelpTroubleshooterUpdate(rec, func(cur *azureARMResource) (map[string]any, int, any) {
					return azureHelpTroubleshooterStart(cur), http.StatusOK, map[string]any{"troubleshooterResourceName": cur.Name}
				})
			},
		},
	})
	registerAzureARMScopeAction(azureHelpNamespace, "checkNameAvailability", azureHelpCheckName)
	registerAzureARMProviderAction(azureHelpNamespace, "discoverSolutions", func(_ *Server, payload map[string]any) (int, any) {
		return azureHelpDiscoverNLP("/providers/"+azureHelpNamespace+"/discoverSolutions", payload)
	})
	registerAzureARMScopeAction(azureHelpNamespace, "discoverSolutions", func(_ *Server, scope string, payload map[string]any) (int, any) {
		if _, ok := azureARMSubscriptionScope(scope); !ok {
			return http.StatusBadRequest, azureARMErrorBody("InvalidScope", "Solutions are discovered for a tenant or a subscription.")
		}
		return azureHelpDiscoverNLP(scope+"/providers/"+azureHelpNamespace+"/discoverSolutions", payload)
	})
}

// azureHelpParameters checks that m[key], when present, is an object of
// strings.
func azureHelpParameters(m map[string]any, path, key string) string {
	raw, present := m[key]
	if !present {
		return ""
	}
	params, ok := raw.(map[string]any)
	if !ok || slices.ContainsFunc(azureMapValues(params), func(v any) bool { _, ok := v.(string); return !ok }) {
		return path + key + " must be an object of strings."
	}
	return ""
}

func azureMapValues(m map[string]any) []any {
	out := make([]any, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// azureHelpScope returns the scope a Microsoft.Help resource attaches to.
func azureHelpScope(id string) string {
	return id[:strings.LastIndex(strings.ToLower(id), "/providers/"+strings.ToLower(azureHelpNamespace)+"/")]
}

func azureHelpDiagnosticsValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureHelpParameters(props, "properties.", "globalParameters"); msg != "" {
		return msg
	}
	insights, ok := props["insights"].([]any)
	if !ok || len(insights) == 0 {
		return "properties.insights must list the diagnostics to run."
	}
	for _, raw := range insights {
		item, ok := raw.(map[string]any)
		if !ok || azureARMString(item["solutionId"]) == "" {
			return "properties.insights items need a solutionId."
		}
		if msg := azureHelpParameters(item, "properties.insights[].", "additionalParameters"); msg != "" {
			return msg
		}
	}
	return ""
}

// azureHelpRunDiagnostic runs a catalog diagnostic at a scope: it needs a
// resource scope (or a ResourceUri parameter) when it asks for ResourceUri.
func azureHelpRunDiagnostic(id, scope string, params map[string]any) map[string]any {
	sol := azureHelpFindOf("Diagnostics", id)
	if sol == nil {
		return map[string]any{"solutionId": id, "status": "Failed", "insights": []any{},
			"error": map[string]any{"code": "SolutionNotFound", "message": "No diagnostic has solution ID '" + id + "'."}}
	}
	missing := []any{}
	for _, input := range sol.requiredInputs {
		_, given := params[input]
		// A resource scope is /subscriptions/{id}/resourceGroups/{rg}/providers/{ns}/{type}/{name}.
		if input == "ResourceUri" && !given && strings.Count(scope, "/") < 8 {
			missing = append(missing, input)
		}
	}
	if len(missing) > 0 {
		return map[string]any{"solutionId": sol.id, "status": "MissingInputs", "insights": []any{}, "requiredParameters": missing}
	}
	insights := make([]any, 0, len(sol.insights))
	for _, insight := range sol.insights {
		insights = append(insights, maps.Clone(insight))
	}
	return map[string]any{"solutionId": sol.id, "status": "Succeeded", "insights": insights}
}

func azureHelpDiagnosticsRun(rec *azureARMResource) map[string]any {
	global := azureARMObject(rec.Properties["globalParameters"])
	results, succeeded := []any{}, 0
	for _, raw := range azureARMSlice(rec.Properties["insights"]) {
		item := azureARMObject(raw)
		result := azureHelpRunDiagnostic(azureARMString(item["solutionId"]), azureHelpScope(rec.ID), azureARMMerge(global, azureARMObject(item["additionalParameters"])))
		if result["status"] == "Succeeded" {
			succeeded++
		}
		results = append(results, result)
	}
	state := "PartialComplete"
	switch succeeded {
	case len(results):
		state = "Succeeded"
	case 0:
		state = "Failed"
	}
	return map[string]any{"acceptedAt": rec.CreatedAt.Format(time.RFC3339Nano), "provisioningState": state, "diagnostics": results}
}

func azureHelpSolutionIDValidate(payload map[string]any, kind string) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureHelpParameters(props, "properties.", "parameters"); msg != "" {
		return msg
	}
	if azureHelpFindOf(kind, azureARMString(props["solutionId"])) == nil {
		return "properties.solutionId must name a " + strings.ToLower(strings.TrimSuffix(kind, "s")) + " from discovery."
	}
	return ""
}

// azureHelpTriggers checks triggerCriteria against the names allowed and
// returns the criteria by name.
func azureHelpTriggers(props map[string]any, allowed ...string) (map[string]string, string) {
	criteria, ok := props["triggerCriteria"].([]any)
	if !ok {
		return nil, "properties.triggerCriteria must be a list."
	}
	out := map[string]string{}
	for _, raw := range criteria {
		item := azureARMObject(raw)
		name := azureARMString(item["name"])
		if !slices.Contains(allowed, name) {
			return nil, "properties.triggerCriteria names must be " + strings.Join(allowed, " or ") + "."
		}
		out[name] = azureARMString(item["value"])
	}
	return out, ""
}

// azureHelpSolutionFor resolves solution triggers to a catalog solution:
// by its ID or the first one for a problem classification.
func azureHelpSolutionFor(triggers map[string]string) *azureHelpSolution {
	if id, ok := triggers["SolutionId"]; ok {
		return azureHelpFindOf("Solutions", id)
	}
	for _, sol := range azureHelpCatalog {
		if sol.kind == "Solutions" && strings.EqualFold(sol.problem, triggers["ProblemClassificationId"]) {
			return &sol
		}
	}
	return nil
}

func azureHelpSolutionValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureHelpParameters(props, "properties.", "parameters"); msg != "" {
		return msg
	}
	triggers, msg := azureHelpTriggers(props, "SolutionId", "ProblemClassificationId")
	if msg != "" {
		return msg
	}
	if len(triggers) != 1 {
		return "properties.triggerCriteria must hold one SolutionId or ProblemClassificationId criterion."
	}
	if azureHelpSolutionFor(triggers) == nil {
		return "properties.triggerCriteria matches no solution from discovery."
	}
	return ""
}

// azureHelpSolutionPatch takes new parameters and ReplacementKey criteria,
// which rerun the solution's diagnostics.
func azureHelpSolutionPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return nil, nil, msg
	}
	props := azureARMObject(payload["properties"])
	for key := range props {
		if key != "parameters" && key != "triggerCriteria" {
			return nil, nil, "properties." + key + " can't be updated."
		}
	}
	if msg := azureHelpParameters(props, "properties.", "parameters"); msg != "" {
		return nil, nil, msg
	}
	if _, present := props["triggerCriteria"]; present {
		if _, msg := azureHelpTriggers(props, "ReplacementKey"); msg != "" {
			return nil, nil, msg
		}
	}
	return payload, nil, ""
}

// azureHelpSolutionRender returns a solution's content, running the
// diagnostics its content embeds and listing its problem's
// troubleshooters. After a PATCH the triggers hold replacement keys, so
// the stored solution ID stands.
func azureHelpSolutionRender(rec *azureARMResource) map[string]any {
	triggers, _ := azureHelpTriggers(rec.Properties, "SolutionId", "ProblemClassificationId")
	sol := azureHelpSolutionFor(triggers)
	if sol == nil {
		sol = azureHelpFindOf("Solutions", azureARMString(rec.Properties["solutionId"]))
	}
	if sol == nil {
		return nil
	}
	params := azureARMObject(rec.Properties["parameters"])
	diagnostics, troubleshooters := []any{}, []any{}
	for _, other := range azureHelpCatalog {
		key := "<!--" + other.id + "-->"
		switch {
		case other.kind == "Diagnostics" && strings.Contains(sol.content, key):
			result := azureHelpRunDiagnostic(other.id, azureHelpScope(rec.ID), params)
			result["replacementKey"] = key
			result["statusDetails"] = ""
			diagnostics = append(diagnostics, result)
		case other.kind == "Troubleshooters" && other.problem == sol.problem:
			troubleshooters = append(troubleshooters, map[string]any{"solutionId": other.id, "title": other.title, "summary": other.description})
		}
	}
	replacements := func() map[string]any {
		return map[string]any{"diagnostics": slices.Clone(diagnostics), "troubleshooters": slices.Clone(troubleshooters)}
	}
	return map[string]any{
		"solutionId": sol.id, "title": sol.title, "content": sol.content, "provisioningState": "Succeeded", "replacementMaps": replacements(),
		"sections": []any{map[string]any{"title": sol.title, "content": sol.content, "replacementMaps": replacements()}},
	}
}

// azureHelpStepView renders a troubleshooter step, with the answer chosen
// at a decision once there is one.
func azureHelpStepView(step azureHelpStep, selected string) map[string]any {
	view := map[string]any{
		"id": step.id, "type": step.kind, "title": step.title, "description": step.description, "guidance": step.description,
		"executionStatus": "Success", "isLastStep": step.kind == "Solution", "inputs": []any{},
	}
	if step.kind == "Decision" {
		keys := make([]string, 0, len(step.next))
		for key := range step.next {
			keys = append(keys, key)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(keys)))
		options := []any{}
		for _, key := range keys {
			options = append(options, map[string]any{"key": key, "value": key})
		}
		input := map[string]any{
			"questionId": step.id, "questionTitle": step.title, "questionContent": step.question, "questionContentType": "Text",
			"questionType": "RadioButton", "responseOptions": options, "responseValidationProperties": map[string]any{"isRequired": true},
		}
		if selected != "" {
			input["selectedOptionValue"] = selected
		}
		view["inputs"] = []any{input}
	}
	return view
}

func azureHelpTroubleshooterStart(rec *azureARMResource) map[string]any {
	sol := azureHelpFindOf("Troubleshooters", azureARMString(rec.Properties["solutionId"]))
	return map[string]any{"steps": []any{azureHelpStepView(sol.steps[0], "")}, "provisioningState": "Running"}
}

// azureHelpTroubleshooterUpdate applies change to the stored troubleshooter
// under the lock and stores the properties it returns.
func (s *Server) azureHelpTroubleshooterUpdate(rec *azureARMResource, change func(cur *azureARMResource) (map[string]any, int, any)) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	cur := s.azureARMResources[azureARMKey(rec.ID)]
	if cur == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The resource '"+rec.ID+"' was not found.")
	}
	props, status, body := change(cur)
	if props != nil {
		updated := *cur
		updated.Properties = azureARMMerge(cur.Properties, props)
		updated.UpdatedAt = time.Now().UTC()
		s.azureARMResources[azureARMKey(rec.ID)] = &updated
	}
	return status, body
}

// azureHelpTroubleshooterContinue answers the current decision step and
// moves to the step the answer leads to.
func azureHelpTroubleshooterContinue(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	return s.azureHelpTroubleshooterUpdate(rec, func(cur *azureARMResource) (map[string]any, int, any) {
		if cur.Properties["provisioningState"] != "Running" {
			return nil, http.StatusConflict, azureARMErrorBody("Conflict", "The troubleshooter has ended; restart it to continue.")
		}
		steps := azureARMSlice(cur.Properties["steps"])
		currentID := azureARMString(azureARMObject(steps[len(steps)-1])["id"])
		if stepID := azureARMString(payload["stepId"]); stepID != currentID {
			return nil, http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "stepId must be the current step, '"+currentID+"'.")
		}
		sol := azureHelpFindOf("Troubleshooters", azureARMString(cur.Properties["solutionId"]))
		var step azureHelpStep
		for _, candidate := range sol.steps {
			if candidate.id == currentID {
				step = candidate
			}
		}
		answer := ""
		for _, raw := range azureARMSlice(payload["responses"]) {
			if response := azureARMObject(raw); azureARMString(response["questionId"]) == step.id {
				answer = azureARMString(response["response"])
			}
		}
		nextID, ok := step.next[answer]
		if !ok {
			return nil, http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "responses must answer question '"+step.id+"' with one of its response options.")
		}
		var next azureHelpStep
		for _, candidate := range sol.steps {
			if candidate.id == nextID {
				next = candidate
			}
		}
		updated := append(slices.Clone(steps[:len(steps)-1]), azureHelpStepView(step, answer), azureHelpStepView(next, ""))
		state := "Running"
		if next.kind == "Solution" {
			state = "Succeeded"
		}
		return map[string]any{"steps": updated, "provisioningState": state}, http.StatusNoContent, nil
	})
}

func azureHelpCheckName(s *Server, scope string, payload map[string]any) (int, any) {
	name, typ := azureARMString(payload["name"]), azureARMString(payload["type"])
	collection := strings.TrimPrefix(strings.ToLower(typ), strings.ToLower(azureHelpNamespace)+"/")
	if !slices.Contains([]string{"diagnostics", "solutions", "simplifiedsolutions", "troubleshooters"}, collection) || !strings.Contains(typ, "/") {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "type must be Microsoft.Help/diagnostics, solutions, simplifiedSolutions or troubleshooters.")
	}
	if strings.TrimSpace(name) == "" || strings.Contains(name, "/") {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required and can't contain '/'.")
	}
	if stored, _ := s.getAzureARMResource(scope+"/providers/"+azureHelpNamespace+"/"+collection+"/"+name, "", "", ""); stored != nil {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "The name '" + name + "' is already in use at this scope."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true}
}
