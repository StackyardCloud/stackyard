package server

import (
	"cmp"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const azureEducation20211201PreviewAPIVersion = "2021-12-01-preview"

// Azure Education (Microsoft.Education 2021-12-01-preview): labs under a
// billing invoice section, their invite codes, join requests and
// students, the grant every billing profile holds, and the student labs
// view. Labs, students and join requests live in the shared ARM store at
// their billing-scoped IDs; this router serves them, since the ARM handler
// only routes subscription and tenant scopes. No subscriptions are made
// for students.

const (
	eduNamespace   = "Microsoft.Education"
	eduMaxStudents = 1000 // students (and pending join requests) per lab
	eduGrantCap    = 10000
)

var (
	eduBillingRE = regexp.MustCompile(`(?i)^/providers/Microsoft\.Billing/billingAccounts/[^/]+/billingProfiles/[^/]+$`)
	eduRoles     = []string{"Student", "Admin"}
	eduAliasRE   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._@-]{0,63}$`)

	eduLabType = azureARMResourceType{
		// Tenant keeps the billing-scoped ID the client sent; the type is
		// never registered, so no tenant route reaches it.
		Namespace: eduNamespace, Type: "labs", Tenant: true, Proxy: true,
		ValidateName: func(name string) string {
			if name != "default" {
				return "The lab name must be 'default'."
			}
			return ""
		},
		Validate: eduValidateLab,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{
				"status": "Active", "effectiveDate": rec.CreatedAt.Format(time.RFC3339),
				"currency": azureARMString(azureARMObject(rec.Properties["budgetPerStudent"])["currency"]),
			}
		},
		StateProperties: map[string]any{
			"invitationCode": nil, "invitationCodeExpirationDate": nil, "maxStudentCount": 0,
			"totalBudget": nil, "totalAllocatedBudget": nil,
		},
	}
	eduStudentType = azureARMResourceType{
		Namespace: eduNamespace, Type: "labs/students", Proxy: true,
		ValidateName: func(name string) string {
			if !eduAliasRE.MatchString(name) {
				return "The student alias '" + name + "' is invalid."
			}
			return ""
		},
		Validate: eduValidateStudent,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{
				"status": "Active", "effectiveDate": rec.CreatedAt.Format(time.RFC3339),
				"subscriptionId": azureHZGUID(rec.ID),
			}
		},
		ValidatePut: func(s *Server, existing, rec *azureARMResource) (string, string) {
			return esanSiblingLimit(s, existing, rec, "students", eduMaxStudents)
		},
		Reconcile: func(s *Server, _, rec *azureARMResource) (map[string]any, string) {
			lab := s.azureARMResources[azureARMKey(esanParentID(rec.ID))]
			if lab != nil && !strings.EqualFold(azureARMString(azureARMObject(rec.Properties["budget"])["currency"]), azureARMString(lab.Properties["currency"])) {
				return nil, "properties.budget.currency is invalid: the lab budgets in " + azureARMString(lab.Properties["currency"]) + "."
			}
			return nil, ""
		},
	}
	eduJoinType = azureARMResourceType{Namespace: eduNamespace, Type: "labs/joinRequests", Proxy: true, NoPut: true, NoPatch: true, NoDelete: true}
)

func init() {
	ns := eduNamespace
	registerAzureARMView(ns, "operations", elasOperations("Microsoft Education", ns, "labs", "labs/students", "labs/joinRequests", "grants", "studentLabs"))
	registerAzureARMQueryView(ns, "grants", func(s *Server, scope string, query url.Values) []map[string]any {
		return s.eduGrants(scope, strings.EqualFold(query.Get("includeAllocatedBudget"), "true"))
	})
	registerAzureARMView(ns, "studentLabs", func(s *Server, scope string) []map[string]any {
		if scope != "" {
			return nil
		}
		return s.eduStudentLabs()
	})
	registerAzureARMProviderAction(ns, "redeemInvitationCode", func(s *Server, payload map[string]any) (int, any) { return s.eduRedeem(payload) })
}

// handleAzureEducation20211201PreviewRouter serves labs, their students and
// join requests under /providers/Microsoft.Billing/.../invoiceSections/
// {name}/providers/Microsoft.Education, and the labs list under a billing
// profile.
func (s *Server) handleAzureEducation20211201PreviewRouter(w http.ResponseWriter, r *http.Request) bool {
	seg := splitPathSegments(strings.TrimPrefix(rawRequestPath(r), "/azure/"))
	// providers/Microsoft.Billing/billingAccounts/{a}/billingProfiles/{b}[/invoiceSections/{c}]/providers/Microsoft.Education/...
	if len(seg) < 9 || !strings.EqualFold(seg[0], "providers") || !strings.EqualFold(seg[1], "Microsoft.Billing") ||
		!strings.EqualFold(seg[2], "billingAccounts") || !strings.EqualFold(seg[4], "billingProfiles") {
		return false
	}
	scope := "/providers/Microsoft.Billing/billingAccounts/" + seg[3] + "/billingProfiles/" + seg[5]
	rest := seg[6:]
	section := len(rest) >= 2 && strings.EqualFold(rest[0], "invoiceSections")
	if section {
		scope += "/invoiceSections/" + rest[1]
		rest = rest[2:]
	}
	if len(rest) < 3 || !strings.EqualFold(rest[0], "providers") || !strings.EqualFold(rest[1], eduNamespace) || !strings.EqualFold(rest[2], "labs") {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	rest = rest[3:]
	if len(rest) == 0 { // {scope}/providers/Microsoft.Education/labs
		if r.Method != http.MethodGet {
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return true
		}
		s.listAzureARMResources(w, eduLabType, "", "", "", scope)
		return true
	}
	if !section {
		azureARMNotFound(w, "ResourceNotFound", rawRequestPath(r), "")
		return true
	}
	labID := scope + "/providers/" + eduNamespace + "/labs/" + rest[0]
	switch {
	case len(rest) == 1:
		s.serveAzureARMItem(w, r, eduLabType, labID, "", "", "", rest[0])
		if r.Method == http.MethodPut {
			s.eduRefreshLab(labID)
		}
	case len(rest) == 2 && strings.EqualFold(rest[1], "generateInviteCode") && r.Method == http.MethodPost:
		s.invokeAzureARMAction(w, r, s.eduGenerateInviteCode, false, false, labID, "", "", "")
	case len(rest) == 2 && strings.EqualFold(rest[1], "students"), len(rest) == 2 && strings.EqualFold(rest[1], "joinRequests"):
		typ := map[bool]azureARMResourceType{true: eduStudentType, false: eduJoinType}[strings.EqualFold(rest[1], "students")]
		if r.Method != http.MethodGet {
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return true
		}
		s.listAzureARMResources(w, typ, "", "", labID, "")
	case len(rest) == 3 && strings.EqualFold(rest[1], "students"):
		s.serveAzureARMItem(w, r, eduStudentType, labID+"/students/"+rest[2], "", "", labID, rest[2])
		if r.Method == http.MethodPut || r.Method == http.MethodDelete {
			s.eduRefreshLab(labID)
		}
	case len(rest) == 3 && strings.EqualFold(rest[1], "joinRequests"):
		s.serveAzureARMItem(w, r, eduJoinType, labID+"/joinRequests/"+rest[2], "", "", labID, rest[2])
	case len(rest) == 4 && strings.EqualFold(rest[1], "joinRequests") && r.Method == http.MethodPost &&
		(strings.EqualFold(rest[3], "approve") || strings.EqualFold(rest[3], "deny")):
		approve := strings.EqualFold(rest[3], "approve")
		s.invokeAzureARMAction(w, r, func(rec *azureARMResource, _ map[string]any) (int, any) { return s.eduDecide(rec, labID, approve) },
			false, false, labID+"/joinRequests/"+rest[2], "", "", labID)
	default:
		azureARMNotFound(w, "ResourceNotFound", rawRequestPath(r), "")
	}
	return true
}

// eduMoney checks a {currency, value} amount.
func eduMoney(m map[string]any, path string) string {
	if msg := azureWorkloadsRequired(m, path, "currency"); msg != "" {
		return msg
	}
	if v, ok := m["value"].(float64); !ok || v < 0 {
		return path + "value is invalid: it must be a non-negative number."
	}
	return ""
}

// eduFuture checks that props[key] is an RFC 3339 time in the future.
func eduFuture(props map[string]any, path, key string) string {
	t, err := time.Parse(time.RFC3339, azureARMString(props[key]))
	switch {
	case azureARMString(props[key]) == "":
		return path + key + " is required."
	case err != nil:
		return path + key + " is invalid: it must be an RFC 3339 time."
	case !t.After(time.Now()):
		return path + key + " is invalid: it must be in the future."
	}
	return ""
}

func eduValidateLab(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	budget, ok := props["budgetPerStudent"].(map[string]any)
	if !ok {
		return "properties.budgetPerStudent is required."
	}
	return cmp.Or(
		azureWorkloadsRequired(props, "properties.", "displayName", "description"),
		azureLabLength(props, "properties.", "displayName", 1, 256),
		eduMoney(budget, "properties.budgetPerStudent."),
		eduFuture(props, "properties.", "expirationDate"),
	)
}

func eduValidateStudent(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	budget, ok := props["budget"].(map[string]any)
	if !ok {
		return "properties.budget is required."
	}
	msg := cmp.Or(
		azureWorkloadsEnums(props, "properties.", []string{"firstName", "lastName", "email", "role"}, map[string][]string{"role": eduRoles}),
		eduMoney(budget, "properties.budget."),
		eduFuture(props, "properties.", "expirationDate"),
	)
	if msg == "" && !strings.Contains(azureARMString(props["email"]), "@") {
		msg = "properties.email is invalid."
	}
	return msg
}

// eduRefreshLab recomputes a lab's budget totals from its students.
func (s *Server) eduRefreshLab(labID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	lab := s.azureARMResources[azureARMKey(labID)]
	if lab == nil {
		return
	}
	total := 0.0
	for _, st := range s.azureARMInlineChildrenLocked(labID, eduStudentType) {
		v, _ := azureARMObject(st.Properties["budget"])["value"].(float64)
		total += v
	}
	money := map[string]any{"currency": azureARMString(lab.Properties["currency"]), "value": total}
	updated := *lab
	updated.Properties = azureARMMerge(lab.Properties, map[string]any{"totalBudget": money, "totalAllocatedBudget": money})
	s.azureARMResources[azureARMKey(labID)] = &updated
}

// eduGenerateInviteCode sets a lab's invite code and student cap.
func (s *Server) eduGenerateInviteCode(rec *azureARMResource, payload map[string]any) (int, any) {
	n, ok := esanInt(payload["maxStudentCount"])
	if !ok || n < 1 || n > eduMaxStudents {
		return azureOracleBad("maxStudentCount is invalid: it must be a whole number from 1 to " + strconv.Itoa(eduMaxStudents) + ".")
	}
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azureARMResources[azureARMKey(rec.ID)]
	if current == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	code := strings.NewReplacer("+", "", "/", "", "=", "").Replace(azureARMDerivedKey(rec.ID, now.Format(time.RFC3339Nano)))[:12]
	updated := *current
	updated.UpdatedAt = now
	updated.Properties = azureARMMerge(current.Properties, map[string]any{
		"invitationCode": code, "maxStudentCount": n,
		"invitationCodeExpirationDate": azureARMString(current.Properties["expirationDate"]),
	})
	s.azureARMResources[azureARMKey(rec.ID)] = &updated
	return http.StatusOK, updated.fixture()
}

// eduRedeem turns an invite code into a pending join request on its lab.
func (s *Server) eduRedeem(payload map[string]any) (int, any) {
	if msg := azureWorkloadsRequired(payload, "", "redeemCode", "firstName", "lastName"); msg != "" {
		return azureOracleBad(msg)
	}
	code := azureARMString(payload["redeemCode"])
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	labs := s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool {
		return strings.EqualFold(r.Type, eduLabType.fullType()) && azureARMString(r.Properties["invitationCode"]) == code
	})
	if len(labs) == 0 {
		return http.StatusNotFound, azureARMErrorBody("InvalidRedeemCode", "The invitation code isn't valid.")
	}
	lab := labs[0]
	if t, err := time.Parse(time.RFC3339, azureARMString(lab.Properties["invitationCodeExpirationDate"])); err == nil && !t.After(time.Now()) {
		return http.StatusConflict, azureARMErrorBody("InvitationCodeExpired", "The invitation code has expired.")
	}
	first, last := azureARMString(payload["firstName"]), azureARMString(payload["lastName"])
	name := azureHZGUID(lab.ID + "#" + first + "#" + last)
	id := lab.ID + "/joinRequests/" + name
	if s.azureARMResources[azureARMKey(id)] != nil {
		return http.StatusConflict, azureARMErrorBody("Conflict", first+" "+last+" already asked to join this lab.")
	}
	limit, _ := esanInt(lab.Properties["maxStudentCount"])
	pending := len(s.azureARMInlineChildrenLocked(lab.ID, eduJoinType)) + len(s.azureARMInlineChildrenLocked(lab.ID, eduStudentType))
	if int64(pending) >= limit {
		return http.StatusConflict, azureARMErrorBody("MaxStudentCountReached", "The lab is full.")
	}
	now := time.Now().UTC()
	s.azureARMResources[azureARMKey(id)] = &azureARMResource{
		ID: id, Name: name, Type: eduJoinType.fullType(), CreatedAt: now, UpdatedAt: now,
		Properties: map[string]any{
			"firstName": first, "lastName": last, "status": "Pending",
			"email": strings.ToLower(first+"."+last) + "@students.stackyard.local",
		},
	}
	return http.StatusOK, nil
}

// eduDecide approves (adding the student) or denies a pending join request.
func (s *Server) eduDecide(rec *azureARMResource, labID string, approve bool) (int, any) {
	s.providerStorageMu.Lock()
	current := s.azureARMResources[azureARMKey(rec.ID)]
	lab := s.azureARMResources[azureARMKey(labID)]
	if current == nil || lab == nil {
		s.providerStorageMu.Unlock()
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	if status := azureARMString(current.Properties["status"]); status != "Pending" {
		s.providerStorageMu.Unlock()
		return http.StatusConflict, azureARMErrorBody("Conflict", "The join request is already "+status+".")
	}
	now := time.Now().UTC()
	updated := *current
	updated.UpdatedAt = now
	updated.Properties = azureARMMerge(current.Properties, map[string]any{"status": map[bool]string{true: "Approved", false: "Denied"}[approve]})
	s.azureARMResources[azureARMKey(rec.ID)] = &updated
	if approve {
		student := &azureARMResource{
			ID: labID + "/students/" + current.Name, Name: current.Name, Type: eduStudentType.fullType(), CreatedAt: now, UpdatedAt: now,
			Properties: map[string]any{
				"firstName": current.Properties["firstName"], "lastName": current.Properties["lastName"], "email": current.Properties["email"],
				"role": "Student", "budget": lab.Properties["budgetPerStudent"], "expirationDate": lab.Properties["expirationDate"],
			},
		}
		student.Properties = azureARMProperties(eduStudentType, student)
		s.azureARMResources[azureARMKey(student.ID)] = student
	}
	s.providerStorageMu.Unlock()
	if approve {
		s.eduRefreshLab(labID)
	}
	return http.StatusOK, nil
}

// eduGrants lists the default grant of the billing profile scope names, or
// of every billing profile with a lab at tenant scope.
func (s *Server) eduGrants(scope string, allocated bool) []map[string]any {
	labs := s.azureARMResourcesWhere(func(r *azureARMResource) bool { return strings.EqualFold(r.Type, eduLabType.fullType()) })
	var profiles []string
	switch {
	case scope == "":
		seen := map[string]bool{}
		for _, lab := range labs {
			profile := lab.ID[:strings.Index(strings.ToLower(lab.ID), "/invoicesections/")]
			if !seen[azureARMKey(profile)] {
				seen[azureARMKey(profile)] = true
				profiles = append(profiles, profile)
			}
		}
	case eduBillingRE.MatchString(scope):
		profiles = []string{scope}
	}
	var out []map[string]any
	for _, profile := range profiles {
		props := map[string]any{
			"offerCap": map[string]any{"currency": "USD", "value": eduGrantCap}, "offerType": "Academic", "status": "Active",
			"effectiveDate": "2025-01-01T00:00:00Z", "expirationDate": "2030-01-01T00:00:00Z",
		}
		if allocated {
			total := 0.0
			for _, lab := range labs {
				if strings.HasPrefix(azureARMKey(lab.ID), azureARMKey(profile+"/")) {
					v, _ := azureARMObject(lab.Properties["totalAllocatedBudget"])["value"].(float64)
					total += v
				}
			}
			props["allocatedBudget"] = map[string]any{"currency": "USD", "value": total}
		}
		out = append(out, map[string]any{
			"id": profile + "/providers/" + eduNamespace + "/grants/default", "name": "default",
			"type": eduNamespace + "/grants", "properties": props,
		})
	}
	return out
}

// eduStudentLabs lists every student's view of their lab.
func (s *Server) eduStudentLabs() []map[string]any {
	students := s.azureARMResourcesWhere(func(r *azureARMResource) bool { return strings.EqualFold(r.Type, eduStudentType.fullType()) })
	var out []map[string]any
	for _, st := range students {
		lab, _ := s.getAzureARMResource(esanParentID(st.ID), "", "", "")
		if lab == nil {
			continue
		}
		name := azureHZGUID(st.ID)
		out = append(out, map[string]any{
			"id": "/providers/" + eduNamespace + "/studentLabs/" + name, "name": name, "type": eduNamespace + "/studentLabs",
			"properties": map[string]any{
				"displayName": lab.Properties["displayName"], "description": lab.Properties["description"],
				"expirationDate": st.Properties["expirationDate"], "role": st.Properties["role"], "budget": st.Properties["budget"],
				"subscriptionId": st.Properties["subscriptionId"], "status": "Active", "effectiveDate": st.Properties["effectiveDate"],
				"labScope": lab.ID,
			},
		})
	}
	return out
}
