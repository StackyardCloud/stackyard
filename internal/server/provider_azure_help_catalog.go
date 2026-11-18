package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
)

// Microsoft.Help draws on a catalog of diagnostics, solutions,
// troubleshooters and self-help articles that Microsoft authors. The
// emulator ships a small catalog of its own: every ID starts with
// "stackyard-", so nothing here passes for Microsoft content.
// ponytail: two problem classifications with a troubleshooter each; loading a catalog from a file is
// the upgrade if tests need their own.

type azureHelpSolution struct {
	id, kind, title, description, problem string
	requiredInputs                        []string
	// insights are what a diagnostic finds; content is a solution's or
	// article's HTML; steps are a troubleshooter's workflow, first step
	// first.
	insights []map[string]any
	content  string
	steps    []azureHelpStep
}

// azureHelpStep is one troubleshooter step. A decision step asks a
// question whose answer keys map to the next step's ID; a solution step
// ends the workflow.
type azureHelpStep struct {
	id, kind, title, description, question string
	next                                   map[string]string
}

type azureHelpProblem struct {
	id, title, description  string
	resourceTypes, keywords []string
}

var azureHelpProblems = []azureHelpProblem{
	{
		id: "stackyard-vm-connectivity", title: "Can't connect to a virtual machine",
		description:   "Remote Desktop or SSH connections to a virtual machine fail or time out.",
		resourceTypes: []string{"microsoft.compute/virtualmachines"},
		keywords:      []string{"virtual machine", "vm", "rdp", "ssh", "remote desktop", "connect"},
	},
	{
		id: "stackyard-keyvault-recovery", title: "Recover deleted Key Vault objects",
		description:   "Keys, secrets or certificates were deleted from a key vault.",
		resourceTypes: []string{"microsoft.keyvault/vaults"},
		keywords:      []string{"key vault", "keyvault", "deleted", "recover", "certificate", "secret"},
	},
}

var azureHelpCatalog = []azureHelpSolution{
	{
		id: "stackyard-vm-rdp-diagnostic", kind: "Diagnostics", problem: "stackyard-vm-connectivity",
		title: "Check virtual machine connectivity", description: "Checks the network rules in front of a virtual machine.",
		requiredInputs: []string{"SubscriptionId", "ResourceUri"},
		insights: []map[string]any{{
			"id": "stackyard-vm-nsg-blocks-rdp", "importanceLevel": "Warning", "title": "A network security group may block remote connections",
			"results": "Check that an inbound rule allows TCP 3389 (RDP) or 22 (SSH) from your address.",
		}},
	},
	{
		id: "stackyard-vm-connectivity-solution", kind: "Solutions", problem: "stackyard-vm-connectivity",
		title: "Fix virtual machine connectivity", description: "Steps to restore Remote Desktop and SSH access.",
		requiredInputs: []string{"SubscriptionId"},
		content:        "<p>Make sure the virtual machine is running and that inbound rules allow your connection.</p><!--stackyard-vm-rdp-diagnostic-->",
	},
	{
		id: "stackyard-vm-connectivity-troubleshooter", kind: "Troubleshooters", problem: "stackyard-vm-connectivity",
		title: "Troubleshoot virtual machine connectivity", description: "Walks through the common causes of failed connections.",
		requiredInputs: []string{"SubscriptionId"},
		steps: []azureHelpStep{
			{id: "running", kind: "Decision", title: "Is the virtual machine running?", description: "Stopped machines refuse connections.",
				question: "Does the portal show the virtual machine as Running?", next: map[string]string{"Yes": "rules", "No": "start"}},
			{id: "rules", kind: "Decision", title: "Do inbound rules allow the connection?", description: "Network security groups filter inbound traffic.",
				question: "Does an inbound rule allow TCP 3389 or 22 from your address?", next: map[string]string{"Yes": "resolved", "No": "open-port"}},
			{id: "start", kind: "Solution", title: "Start the virtual machine", description: "Start the machine, then connect again."},
			{id: "open-port", kind: "Solution", title: "Allow the connection", description: "Add an inbound rule for your address and port."},
			{id: "resolved", kind: "Solution", title: "Contact support", description: "The common causes are ruled out; open a support request."},
		},
	},
	{
		id: "stackyard-keyvault-recover-selfhelp", kind: "SelfHelp", problem: "stackyard-keyvault-recovery",
		title: "Recover deleted Key Vault objects", description: "How soft delete lets you recover keys, secrets and certificates.",
		content: "<p>While soft delete is on, list deleted objects and recover them within the retention period.</p>",
	},
	{
		id: "stackyard-keyvault-recovery-troubleshooter", kind: "Troubleshooters", problem: "stackyard-keyvault-recovery",
		title: "Troubleshoot Key Vault recovery", description: "Checks whether a deleted object can still be recovered.",
		requiredInputs: []string{"SubscriptionId"},
		steps: []azureHelpStep{
			{id: "soft-delete", kind: "Decision", title: "Is soft delete on?", description: "Only soft-deleted objects can be recovered.",
				question: "Does the vault have soft delete turned on?", next: map[string]string{"Yes": "recover", "No": "restore"}},
			{id: "recover", kind: "Solution", title: "Recover the object", description: "Recover it before its retention period ends."},
			{id: "restore", kind: "Solution", title: "Restore from a backup", description: "Without soft delete, restore the object from a backup."},
		},
	},
	{
		id: "stackyard-keyvault-recovery-solution", kind: "Solutions", problem: "stackyard-keyvault-recovery",
		title: "Recover a deleted key vault object", description: "Recover keys, secrets and certificates that soft delete keeps.",
		requiredInputs: []string{"SubscriptionId"},
		content:        "<p>Recover the object before its retention period ends, or purge it to reuse the name.</p>",
	},
}

func azureHelpFind(id string) *azureHelpSolution {
	for i := range azureHelpCatalog {
		if strings.EqualFold(azureHelpCatalog[i].id, id) {
			return &azureHelpCatalog[i]
		}
	}
	return nil
}

// azureHelpFindOf returns the catalog solution of a kind with the ID, or
// nil.
func azureHelpFindOf(kind, id string) *azureHelpSolution {
	if sol := azureHelpFind(id); sol != nil && sol.kind == kind {
		return sol
	}
	return nil
}

func azureHelpMetadata(sol azureHelpSolution) map[string]any {
	inputs := sol.requiredInputs
	if inputs == nil {
		inputs = []string{}
	}
	return map[string]any{"solutionId": sol.id, "solutionType": sol.kind, "description": sol.description, "requiredInputs": inputs}
}

// azureHelpDiscovery groups the matching catalog solutions by problem
// classification, at the collection path given.
func azureHelpDiscovery(collection string, problems []azureHelpProblem, match func(azureHelpSolution) bool) []map[string]any {
	items := []map[string]any{}
	for _, p := range problems {
		solutions := []any{}
		for _, sol := range azureHelpCatalog {
			if sol.problem == p.id && match(sol) {
				solutions = append(solutions, azureHelpMetadata(sol))
			}
		}
		if len(solutions) == 0 {
			continue
		}
		items = append(items, map[string]any{
			"id": collection + "/" + p.id, "name": p.id, "type": azureHelpNamespace + "/discoverySolutions",
			"properties": map[string]any{"problemClassificationId": p.id, "problemTitle": p.title, "problemDescription": p.description, "solutions": solutions},
		})
	}
	return items
}

var (
	azureHelpFilterTerm = regexp.MustCompile(`^\s*(\w+)\s+eq\s+'([^']*)'\s*$`)
	azureHelpFilterOr   = regexp.MustCompile(`(?i)\s+or\s+`)
	azureHelpFilterAnd  = regexp.MustCompile(`(?i)\s+and\s+`)
)

// azureHelpFilter compiles a discovery $filter: eq terms joined by `and`
// and `or` (and binds tighter), over ProblemClassificationId (required in
// every `or` clause), ResourceType and SolutionType.
// ponytail: values can't contain " and " or " or ".
func azureHelpFilter(filter string) (func(p azureHelpProblem, sol azureHelpSolution) bool, string) {
	var clauses [][][2]string
	for _, clause := range azureHelpFilterOr.Split(filter, -1) {
		var terms [][2]string
		sawProblem := false
		for _, term := range azureHelpFilterAnd.Split(clause, -1) {
			m := azureHelpFilterTerm.FindStringSubmatch(strings.Trim(strings.TrimSpace(term), "()"))
			if m == nil {
				return nil, "$filter supports only `Field eq 'value'` terms joined by and/or; got `" + strings.TrimSpace(term) + "`."
			}
			field := strings.ToLower(m[1])
			if !slices.Contains([]string{"problemclassificationid", "resourcetype", "solutiontype"}, field) {
				return nil, "$filter supports ProblemClassificationId, ResourceType and SolutionType; got " + m[1] + "."
			}
			sawProblem = sawProblem || field == "problemclassificationid"
			terms = append(terms, [2]string{field, m[2]})
		}
		if !sawProblem {
			return nil, "Each $filter clause must include ProblemClassificationId."
		}
		clauses = append(clauses, terms)
	}
	return func(p azureHelpProblem, sol azureHelpSolution) bool {
		for _, terms := range clauses {
			ok := true
			for _, t := range terms {
				switch t[0] {
				case "problemclassificationid":
					ok = ok && strings.EqualFold(p.id, t[1])
				case "resourcetype":
					ok = ok && slices.Contains(p.resourceTypes, strings.ToLower(t[1]))
				case "solutiontype":
					ok = ok && strings.EqualFold(sol.kind, t[1])
				}
			}
			if ok {
				return true
			}
		}
		return false
	}, ""
}

// handleAzureHelpRouter serves the tenant reads whose query the shared
// views can't see: discovery by $filter and self-help articles.
func (s *Server) handleAzureHelpRouter(w http.ResponseWriter, r *http.Request) bool {
	seg := splitPathSegments(strings.TrimPrefix(rawRequestPath(r), "/azure/"))
	if len(seg) < 3 || !strings.EqualFold(seg[0], "providers") || !strings.EqualFold(seg[1], azureHelpNamespace) {
		return false
	}
	switch {
	case len(seg) == 3 && strings.EqualFold(seg[2], "discoverySolutions"):
	case len(seg) == 4 && strings.EqualFold(seg[2], "selfHelp"):
	default:
		return false
	}
	if r.Method != http.MethodGet {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	if len(seg) == 4 {
		sol := azureHelpFindOf("SelfHelp", seg[3])
		if sol == nil {
			azureARMNotFound(w, "ResourceNotFound", "/providers/"+azureHelpNamespace+"/selfHelp/"+seg[3], "")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"id": "/providers/" + azureHelpNamespace + "/selfHelp/" + sol.id, "name": sol.id, "type": azureHelpNamespace + "/selfHelp",
			"properties": map[string]any{
				"solutionId": sol.id, "title": sol.title, "content": sol.content, "replacementMaps": map[string]any{},
				"sections": []any{map[string]any{"title": sol.title, "content": sol.content, "replacementMaps": map[string]any{}}},
			},
		})
		return true
	}
	match, msg := azureHelpFilter(r.URL.Query().Get("$filter"))
	if msg != "" {
		azureARMError(w, http.StatusBadRequest, "InvalidFilter", msg)
		return true
	}
	var problems []azureHelpProblem
	items := []map[string]any{}
	for _, p := range azureHelpProblems {
		problems = append(problems[:0], p)
		items = append(items, azureHelpDiscovery("/providers/"+azureHelpNamespace+"/discoverySolutions", problems, func(sol azureHelpSolution) bool { return match(p, sol) })...)
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
	return true
}

// azureHelpDiscoverNLP matches an issue summary against the problems'
// keywords.
// ponytail: keyword matching stands in for natural language search.
func azureHelpDiscoverNLP(collection string, payload map[string]any) (int, any) {
	summary, ok := payload["issueSummary"].(string)
	if !ok || strings.TrimSpace(summary) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "issueSummary is required.")
	}
	for _, key := range []string{"additionalContext", "resourceId", "serviceId"} {
		if raw, present := payload[key]; present {
			if _, ok := raw.(string); !ok {
				return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", key+" must be a string.")
			}
		}
	}
	text := strings.ToLower(summary + " " + azureARMString(payload["additionalContext"]))
	resource := strings.ToLower(azureARMString(payload["resourceId"]))
	var problems []azureHelpProblem
	for _, p := range azureHelpProblems {
		if slices.ContainsFunc(p.keywords, func(k string) bool { return strings.Contains(text, k) }) ||
			slices.ContainsFunc(p.resourceTypes, func(t string) bool { return strings.Contains(resource, "/providers/"+t+"/") }) {
			problems = append(problems, p)
		}
	}
	return http.StatusOK, map[string]any{"value": azureHelpDiscovery(collection, problems, func(azureHelpSolution) bool { return true })}
}
